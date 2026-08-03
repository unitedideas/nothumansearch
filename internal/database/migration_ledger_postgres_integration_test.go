package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/unitedideas/nothumansearch/internal/models"
)

// TestProtectedMigrationLedgerPostgres is opt-in because it owns and mutates
// the isolated database named by NHS_MIGRATION_TEST_POSTGRES_DSN.
func TestProtectedMigrationLedgerPostgres(t *testing.T) {
	dsn := os.Getenv("NHS_MIGRATION_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set NHS_MIGRATION_TEST_POSTGRES_DSN to an isolated disposable PostgreSQL database")
	}
	t.Setenv("DATABASE_URL", dsn)
	allowPartialProtectedMigrationsForTests = true
	t.Cleanup(func() { allowPartialProtectedMigrationsForTests = false })
	if err := Connect(); err != nil {
		t.Fatalf("connect isolated PostgreSQL: %v", err)
	}
	t.Cleanup(func() {
		_ = DB.Close()
		DB = nil
	})

	const revision = "2222222222222222222222222222222222222222"
	repositoryMigrations := filepath.Join("..", "..", "migrations")
	legacyMigrations := copyMigrationFixture(t, repositoryMigrations, func(name string, _ []byte) ([]byte, bool) {
		return nil, name < protectedMigrationStart
	})
	if err := RunMigrations(legacyMigrations, revision); err != nil {
		t.Fatalf("establish legacy baseline: %v", err)
	}

	t.Run("invalid release identity fails before legacy replay", func(t *testing.T) {
		if _, err := DB.Exec(`CREATE TABLE migration_preflight_marker (value INTEGER NOT NULL)`); err != nil {
			t.Fatal(err)
		}
		candidate := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name == "018_agent_demand.sql" {
				data = append(data, []byte("\nINSERT INTO migration_preflight_marker (value) VALUES (1);\n")...)
			}
			return data, true
		})
		if err := RunMigrations(candidate, "development"); err == nil || !strings.Contains(err.Error(), "full 40-character Git commit") {
			t.Fatalf("invalid release identity error = %v", err)
		}
		var writes int
		if err := DB.QueryRow(`SELECT COUNT(*) FROM migration_preflight_marker`).Scan(&writes); err != nil {
			t.Fatal(err)
		}
		if writes != 0 {
			t.Fatalf("legacy migration wrote %d rows before release identity rejection", writes)
		}
	})

	t.Run("atomic rollback", func(t *testing.T) {
		broken := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name == "019_provider_exchange.sql" {
				data = append(data, []byte("\nSELECT * FROM nhs_intentional_missing_relation;\n")...)
			}
			return data, true
		})
		if err := RunMigrations(broken, revision); err == nil || !strings.Contains(err.Error(), "nhs_intentional_missing_relation") {
			t.Fatalf("broken protected migration error = %v", err)
		}
		assertPostgresRelationAbsent(t, "nhs_schema_migrations")
		assertPostgresRelationAbsent(t, "provider_claims")
	})

	t.Run("ambiguous prior footprint", func(t *testing.T) {
		if _, err := DB.Exec(`CREATE TABLE provider_claims (id INTEGER PRIMARY KEY)`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "ambiguous_prior_019") {
			t.Fatalf("ambiguous footprint error = %v", err)
		}
		assertPostgresRelationAbsent(t, "nhs_schema_migrations")
		if _, err := DB.Exec(`DROP TABLE provider_claims`); err != nil {
			t.Fatal(err)
		}
	})

	through019 := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
		return data, name <= "019_provider_exchange.sql"
	})
	if err := RunMigrations(through019, revision); err != nil {
		t.Fatalf("establish exact 019 baseline: %v", err)
	}

	t.Run("020 atomic rollback preserves 019", func(t *testing.T) {
		broken := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name == "020_action_interest_receipts.sql" {
				data = append(data, []byte("\nSELECT * FROM nhs_intentional_missing_action_interest_relation;\n")...)
			}
			return data, true
		})
		if err := RunMigrations(broken, revision); err == nil || !strings.Contains(err.Error(), "nhs_intentional_missing_action_interest_relation") {
			t.Fatalf("broken 020 migration error = %v", err)
		}
		for _, relation := range []string{
			"action_interest_receipts",
			"idx_search_receipts_id_synthetic",
			"idx_action_interest_receipts_domain_created",
			"idx_action_interest_receipts_action_created",
			"idx_action_interest_receipts_expires",
		} {
			assertPostgresRelationAbsent(t, relation)
		}
		var rule020Exists bool
		if err := DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_rewrite WHERE rulename='action_interest_receipts_no_update')`).Scan(&rule020Exists); err != nil {
			t.Fatal(err)
		}
		if rule020Exists {
			t.Fatal("020 no-update rule survived failed protected migration")
		}
		var receipt020Exists bool
		if err := DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM nhs_schema_migrations WHERE name='020_action_interest_receipts.sql')`).Scan(&receipt020Exists); err != nil {
			t.Fatal(err)
		}
		if receipt020Exists {
			t.Fatal("020 receipt survived failed protected migration")
		}
		var receipt019Exists bool
		if err := DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM nhs_schema_migrations WHERE name='019_provider_exchange.sql')`).Scan(&receipt019Exists); err != nil {
			t.Fatal(err)
		}
		if !receipt019Exists {
			t.Fatal("valid 019 receipt was lost during failed 020 migration")
		}
	})

	t.Run("ambiguous prior 020 footprint", func(t *testing.T) {
		if _, err := DB.Exec(`CREATE TABLE action_interest_receipts (id INTEGER PRIMARY KEY)`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "ambiguous_prior_020") {
			t.Fatalf("ambiguous 020 footprint error = %v", err)
		}
		if _, err := DB.Exec(`DROP TABLE action_interest_receipts`); err != nil {
			t.Fatal(err)
		}
	})

	through020 := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
		return data, name <= "020_action_interest_receipts.sql"
	})
	if err := RunMigrations(through020, revision); err != nil {
		t.Fatalf("establish exact 020 baseline: %v", err)
	}

	t.Run("021 atomic rollback preserves 020", func(t *testing.T) {
		broken := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name == "021_provider_capacity_reservations.sql" {
				data = append(data, []byte("\nSELECT * FROM nhs_intentional_missing_capacity_relation;\n")...)
			}
			return data, true
		})
		if err := RunMigrations(broken, revision); err == nil || !strings.Contains(err.Error(), "nhs_intentional_missing_capacity_relation") {
			t.Fatalf("broken 021 migration error = %v", err)
		}
		for _, relation := range []string{
			"provider_capacity_events",
			"idx_provider_capacity_events_offer_created",
			"idx_provider_capacity_events_ticket_created",
			"idx_provider_capacity_one_terminal_per_ticket",
		} {
			assertPostgresRelationAbsent(t, relation)
		}
		var receipt021Exists, receipt020Exists bool
		if err := DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM nhs_schema_migrations WHERE name='021_provider_capacity_reservations.sql')`).Scan(&receipt021Exists); err != nil {
			t.Fatal(err)
		}
		if receipt021Exists {
			t.Fatal("021 receipt survived failed protected migration")
		}
		if err := DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM nhs_schema_migrations WHERE name='020_action_interest_receipts.sql')`).Scan(&receipt020Exists); err != nil {
			t.Fatal(err)
		}
		if !receipt020Exists {
			t.Fatal("valid 020 receipt was lost during failed 021 migration")
		}
	})

	t.Run("ambiguous prior 021 footprint", func(t *testing.T) {
		if _, err := DB.Exec(`CREATE TABLE provider_capacity_events (id INTEGER PRIMARY KEY)`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "ambiguous_prior_021") {
			t.Fatalf("ambiguous 021 footprint error = %v", err)
		}
		if _, err := DB.Exec(`DROP TABLE provider_capacity_events`); err != nil {
			t.Fatal(err)
		}
	})

	through021 := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
		return data, name <= "021_provider_capacity_reservations.sql"
	})
	capacityFixture := seedProviderCapacityMigrationFixture(t)
	legacyWriter, err := DB.Begin()
	if err != nil {
		t.Fatalf("begin legacy ticket writer: %v", err)
	}
	defer func() { _ = legacyWriter.Rollback() }()
	var legacyTicketID string
	if err := legacyWriter.QueryRow(`
		INSERT INTO action_tickets (
			provider_claim_id, provider_offer_id, source_is_synthetic,
			token_hash, token_nonce, creation_request_hash,
			offer_version_snapshot, offer_name_snapshot, offer_summary_snapshot,
			action_type_snapshot, action_url_snapshot, disclosure_snapshot,
			charge_event_snapshot, bounty_cents_snapshot, currency_snapshot,
			billing_mode_snapshot, terms_evidence_reference_snapshot,
			attribution_key_id_snapshot, principal_price_mode_snapshot,
			principal_price_cents_snapshot, principal_currency_snapshot,
			demand_topic, principal_consent, consent_version, expires_at
		) VALUES (
			$1, $2, true, $3, $4, $5,
			1, 'Legacy capacity offer', 'Ticket committed while migration waits',
			'signup', 'https://capacity-migration.example.test/start',
			'Provider-funded action', 'accepted', 125, 'usd',
			'prepaid', 'legacy-capacity-fixture', 'capacity-key-v1',
			'free', 0, 'usd', 'developer-tools', true,
			'nhs-principal-consent-v1', NOW() + INTERVAL '1 hour'
		)
		RETURNING id`,
		capacityFixture.claimID,
		capacityFixture.offerID,
		postgresMigrationTestHash(capacityFixture.seed+"-ticket-token"),
		postgresMigrationTestHash(capacityFixture.seed + "-ticket-nonce")[:32],
		postgresMigrationTestHash(capacityFixture.seed+"-ticket-request"),
	).Scan(&legacyTicketID); err != nil {
		t.Fatalf("insert uncommitted legacy ticket: %v", err)
	}

	t.Run("021 locks out legacy writer before backfill snapshot", func(t *testing.T) {
		migrationDone := make(chan error, 1)
		go func() {
			migrationDone <- RunMigrations(through021, revision)
		}()
		waitForPendingPostgresRelationLock(t, "action_tickets", "ShareRowExclusiveLock")
		if err := legacyWriter.Commit(); err != nil {
			t.Fatalf("commit pre-migration legacy ticket: %v", err)
		}
		select {
		case err := <-migrationDone:
			if err != nil {
				t.Fatalf("apply 021 after legacy writer commit: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("021 did not finish after legacy writer released action_tickets")
		}
		var reservations int
		if err := DB.QueryRow(`
			SELECT COUNT(*)
			FROM provider_capacity_events
			WHERE action_ticket_id=$1
			  AND event_type='reserve'
			  AND event_reason='ticket_created'
			  AND amount_cents=125
			  AND currency='usd'`, legacyTicketID).Scan(&reservations); err != nil {
			t.Fatal(err)
		}
		if reservations != 1 {
			t.Fatalf("legacy ticket backfill reservations=%d, want 1", reservations)
		}
	})

	t.Run("021 rejects post-migration ticket-only legacy writer at commit", func(t *testing.T) {
		tx, err := DB.Begin()
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback() }()
		var rejectedTicketID string
		if err := tx.QueryRow(`
			INSERT INTO action_tickets (
				provider_claim_id, provider_offer_id, source_is_synthetic,
				token_hash, token_nonce, creation_request_hash,
				offer_version_snapshot, offer_name_snapshot, offer_summary_snapshot,
				action_type_snapshot, action_url_snapshot, disclosure_snapshot,
				charge_event_snapshot, bounty_cents_snapshot, currency_snapshot,
				billing_mode_snapshot, terms_evidence_reference_snapshot,
				terms_credit_limit_cents_snapshot, terms_period_days_snapshot,
				terms_period_anchor_at_snapshot, attribution_key_id_snapshot,
				principal_price_mode_snapshot, principal_price_cents_snapshot,
				principal_currency_snapshot, demand_topic, principal_consent,
				consent_version, expires_at
			)
			SELECT provider_claim_id, provider_offer_id, true,
			       $2, $3, $4,
			       offer_version_snapshot, offer_name_snapshot, offer_summary_snapshot,
			       action_type_snapshot, action_url_snapshot, disclosure_snapshot,
			       charge_event_snapshot, bounty_cents_snapshot, currency_snapshot,
			       billing_mode_snapshot, terms_evidence_reference_snapshot,
			       terms_credit_limit_cents_snapshot, terms_period_days_snapshot,
			       terms_period_anchor_at_snapshot, attribution_key_id_snapshot,
			       principal_price_mode_snapshot, principal_price_cents_snapshot,
			       principal_currency_snapshot, demand_topic, true,
			       consent_version, NOW() + INTERVAL '1 hour'
			FROM action_tickets
			WHERE id=$1
			RETURNING id`,
			legacyTicketID,
			postgresMigrationTestHash(capacityFixture.seed+"-rejected-token"),
			postgresMigrationTestHash(capacityFixture.seed + "-rejected-nonce")[:32],
			postgresMigrationTestHash(capacityFixture.seed+"-rejected-request"),
		).Scan(&rejectedTicketID); err != nil {
			t.Fatalf("ticket-only insert should reach deferred constraint: %v", err)
		}
		if err := tx.Commit(); err == nil || !strings.Contains(err.Error(), "action ticket capacity reservation required") {
			t.Fatalf("ticket-only commit error = %v", err)
		}
		var persisted int
		if err := DB.QueryRow(`SELECT COUNT(*) FROM action_tickets WHERE id=$1`, rejectedTicketID).Scan(&persisted); err != nil {
			t.Fatal(err)
		}
		if persisted != 0 {
			t.Fatalf("ticket-only legacy row persisted after deferred constraint failure: %s", rejectedTicketID)
		}
	})

	t.Run("022 rejects live pre-handoff ticket cutover", func(t *testing.T) {
		err := RunMigrations(repositoryMigrations, revision)
		if err == nil || !strings.Contains(err.Error(), "zero live pre-handoff action tickets") {
			t.Fatalf("live pre-handoff cutover error = %v", err)
		}
		var receipt022Exists bool
		if err := DB.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM nhs_schema_migrations
				WHERE name='022_provider_commercial_proof.sql'
			)`).Scan(&receipt022Exists); err != nil {
			t.Fatal(err)
		}
		if receipt022Exists {
			t.Fatal("022 receipt survived live pre-handoff cutover rejection")
		}
	})

	// Resolve the exact legacy authorization deliberately; migration 022 must
	// never synthesize a handoff receipt or silently strand it. A revoked ticket
	// is logically outside active capacity while its append-only reservation
	// remains available for audit.
	if _, err := DB.Exec(`
		UPDATE action_tickets
		SET authorization_revoked_at=clock_timestamp(), updated_at=clock_timestamp()
		WHERE id=$1::uuid`, legacyTicketID); err != nil {
		t.Fatalf("revoke live migration-cutover fixture ticket: %v", err)
	}

	t.Run("022 atomic rollback preserves 021", func(t *testing.T) {
		broken := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name == "022_provider_commercial_proof.sql" {
				data = append(data, []byte("\nSELECT * FROM nhs_intentional_missing_commercial_proof_relation;\n")...)
			}
			return data, true
		})
		if err := RunMigrations(broken, revision); err == nil || !strings.Contains(err.Error(), "nhs_intentional_missing_commercial_proof_relation") {
			t.Fatalf("broken 022 migration error = %v", err)
		}
		for _, relation := range []string{
			"provider_commercial_acceptance_events",
			"idx_provider_commercial_acceptance_claim_created",
			"idx_provider_commercial_acceptance_offer_created",
			"idx_provider_commercial_acceptance_one_renewal",
			"provider_pilot_companies",
			"idx_provider_pilot_companies_claim",
			"provider_commercial_commitment_events",
			"idx_provider_commercial_commitment_company_created",
			"idx_provider_commercial_commitment_offer_created",
			"idx_provider_commercial_commitment_related",
			"idx_provider_commercial_one_terms_renewal",
			"provider_action_handoff_receipts",
			"idx_provider_action_handoff_offer_observed",
		} {
			assertPostgresRelationAbsent(t, relation)
		}
		for _, column := range []struct {
			table string
			name  string
		}{
			{table: "provider_offers", name: "commercial_terms_contract_version"},
			{table: "provider_offers", name: "commercial_terms_sha256"},
			{table: "provider_offers_returned", name: "commercial_terms_contract_version_snapshot"},
			{table: "provider_offers_returned", name: "commercial_terms_sha256_snapshot"},
			{table: "action_tickets", name: "commercial_terms_contract_version_snapshot"},
			{table: "action_tickets", name: "commercial_terms_sha256_snapshot"},
		} {
			assertPostgresColumnAbsent(t, column.table, column.name)
		}
		for _, function := range []string{
			"enforce_provider_commercial_acceptance_event",
			"enforce_provider_pilot_company",
			"enforce_provider_commercial_commitment_event",
			"enforce_provider_offer_commercial_immutability",
			"enforce_provider_action_handoff_receipt",
			"enforce_action_ticket_observed_handoff_status",
		} {
			assertPostgresFunctionAbsent(t, function)
		}
		var receipt022Exists, receipt021Exists bool
		if err := DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM nhs_schema_migrations WHERE name='022_provider_commercial_proof.sql')`).Scan(&receipt022Exists); err != nil {
			t.Fatal(err)
		}
		if receipt022Exists {
			t.Fatal("022 receipt survived failed protected migration")
		}
		if err := DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM nhs_schema_migrations WHERE name='021_provider_capacity_reservations.sql')`).Scan(&receipt021Exists); err != nil {
			t.Fatal(err)
		}
		if !receipt021Exists {
			t.Fatal("valid 021 receipt was lost during failed 022 migration")
		}
	})

	t.Run("ambiguous prior 022 footprint", func(t *testing.T) {
		if _, err := DB.Exec(`CREATE TABLE provider_commercial_acceptance_events (id INTEGER PRIMARY KEY)`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "ambiguous_prior_022") {
			t.Fatalf("ambiguous 022 footprint error = %v", err)
		}
		if _, err := DB.Exec(`DROP TABLE provider_commercial_acceptance_events`); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("022 transactional preflight failure leaves zero footprint", func(t *testing.T) {
		called := 0
		err := RunMigrationsWithPreflight(
			repositoryMigrations,
			revision,
			func(_ context.Context, _ *sql.Tx, name string) error {
				if name != "022_provider_commercial_proof.sql" {
					return nil
				}
				called++
				return fmt.Errorf("intentional signing retention rejection")
			},
		)
		if err == nil || !strings.Contains(err.Error(), "intentional signing retention rejection") || called != 1 {
			t.Fatalf("022 preflight error=%v calls=%d", err, called)
		}
		assertPostgresRelationAbsent(t, "provider_action_handoff_receipts")
		assertPostgresColumnAbsent(t, "action_tickets", "commercial_terms_sha256_snapshot")
		var receipt022Exists bool
		if err := DB.QueryRow(`SELECT EXISTS(
			SELECT 1 FROM nhs_schema_migrations
			WHERE name='022_provider_commercial_proof.sql'
		)`).Scan(&receipt022Exists); err != nil {
			t.Fatal(err)
		}
		if receipt022Exists {
			t.Fatal("022 receipt survived transactional preflight rejection")
		}
	})

	t.Run("022 conflicting writer fails within lock timeout and cleanly retries", func(t *testing.T) {
		blocker, err := DB.Begin()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := blocker.Exec(`LOCK TABLE action_tickets IN ACCESS SHARE MODE`); err != nil {
			_ = blocker.Rollback()
			t.Fatalf("hold conflicting action-ticket lock: %v", err)
		}
		previousTimeout := migrationTransactionLockTimeout
		migrationTransactionLockTimeout = 250 * time.Millisecond
		startedAt := time.Now()
		err = RunMigrationsWithPreflight(
			repositoryMigrations,
			revision,
			func(ctx context.Context, tx *sql.Tx, name string) error {
				if name != "022_provider_commercial_proof.sql" {
					return nil
				}
				_, err := tx.ExecContext(ctx, `LOCK TABLE action_tickets IN ACCESS EXCLUSIVE MODE`)
				return err
			},
		)
		migrationTransactionLockTimeout = previousTimeout
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "lock timeout") {
			_ = blocker.Rollback()
			t.Fatalf("022 conflicting-lock error=%v", err)
		}
		if elapsed := time.Since(startedAt); elapsed > 5*time.Second {
			_ = blocker.Rollback()
			t.Fatalf("022 lock timeout was not bounded: %s", elapsed)
		}
		assertPostgresRelationAbsent(t, "provider_action_handoff_receipts")
		assertPostgresColumnAbsent(t, "action_tickets", "commercial_terms_sha256_snapshot")
		var receipt022Exists bool
		if err := DB.QueryRow(`SELECT EXISTS(
			SELECT 1 FROM nhs_schema_migrations
			WHERE name='022_provider_commercial_proof.sql'
		)`).Scan(&receipt022Exists); err != nil {
			_ = blocker.Rollback()
			t.Fatal(err)
		}
		if receipt022Exists {
			_ = blocker.Rollback()
			t.Fatal("022 receipt survived the bounded conflicting-lock failure")
		}
		if err := blocker.Rollback(); err != nil {
			t.Fatalf("release conflicting action-ticket lock: %v", err)
		}
	})

	through022 := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
		return data, name <= "022_provider_commercial_proof.sql"
	})
	if err := RunMigrations(through022, revision); err != nil {
		t.Fatalf("apply exact 022 migration: %v", err)
	}
	t.Run("receipted 022 does not rerun transactional preflight", func(t *testing.T) {
		called := false
		err := RunMigrationsWithPreflight(
			through022,
			revision,
			func(_ context.Context, _ *sql.Tx, _ string) error {
				called = true
				return fmt.Errorf("receipted migration unexpectedly invoked preflight")
			},
		)
		if err != nil || called {
			t.Fatalf("receipted 022 preflight error=%v called=%t", err, called)
		}
	})
	t.Run("022 rejects resumed old ticket writer", func(t *testing.T) {
		for _, column := range []string{
			"commercial_terms_contract_version_snapshot",
			"commercial_terms_sha256_snapshot",
		} {
			var hasDefault bool
			if err := DB.QueryRow(`
				SELECT EXISTS (
					SELECT 1
					FROM pg_catalog.pg_attrdef def
					JOIN pg_catalog.pg_attribute attr
					  ON attr.attrelid=def.adrelid AND attr.attnum=def.adnum
					WHERE def.adrelid='public.action_tickets'::regclass
					  AND attr.attname=$1
				)`, column).Scan(&hasDefault); err != nil {
				t.Fatalf("inspect 022 ticket default %s: %v", column, err)
			}
			if hasDefault {
				t.Fatalf("022 retained old-writer-compatible default for %s", column)
			}
		}

		_, err := DB.Exec(`
			INSERT INTO action_tickets (
				provider_claim_id, provider_offer_id, source_is_synthetic,
				token_hash, token_nonce, creation_request_hash,
				offer_version_snapshot, offer_name_snapshot, offer_summary_snapshot,
				action_type_snapshot, action_url_snapshot, disclosure_snapshot,
				charge_event_snapshot, bounty_cents_snapshot, currency_snapshot,
				billing_mode_snapshot, terms_evidence_reference_snapshot,
				terms_credit_limit_cents_snapshot, terms_period_days_snapshot,
				terms_period_anchor_at_snapshot, attribution_key_id_snapshot,
				principal_price_mode_snapshot, principal_price_cents_snapshot,
				principal_currency_snapshot, demand_topic, principal_consent,
				consent_version, expires_at
			)
			SELECT provider_claim_id, provider_offer_id, true,
			       $2, $3, $4,
			       offer_version_snapshot, offer_name_snapshot, offer_summary_snapshot,
			       action_type_snapshot, action_url_snapshot, disclosure_snapshot,
			       charge_event_snapshot, bounty_cents_snapshot, currency_snapshot,
			       billing_mode_snapshot, terms_evidence_reference_snapshot,
			       terms_credit_limit_cents_snapshot, terms_period_days_snapshot,
			       terms_period_anchor_at_snapshot, attribution_key_id_snapshot,
			       principal_price_mode_snapshot, principal_price_cents_snapshot,
			       principal_currency_snapshot, demand_topic, true,
			       consent_version, NOW() + INTERVAL '1 hour'
			FROM action_tickets WHERE id=$1::uuid`,
			legacyTicketID,
			postgresMigrationTestHash(capacityFixture.seed+"-post-022-old-token"),
			postgresMigrationTestHash(capacityFixture.seed + "-post-022-old-nonce")[:32],
			postgresMigrationTestHash(capacityFixture.seed+"-post-022-old-request"),
		)
		if err == nil || !strings.Contains(err.Error(), "commercial_terms_contract_version_snapshot") {
			t.Fatalf("resumed old ticket writer error = %v, want missing contract snapshot", err)
		}
	})
	var legacyAcceptanceRows, legacyCompanyRows, legacyCommitmentRows int
	if err := DB.QueryRow(`
		SELECT
			(SELECT COUNT(*)::int FROM provider_commercial_acceptance_events),
			(SELECT COUNT(*)::int FROM provider_pilot_companies),
			(SELECT COUNT(*)::int FROM provider_commercial_commitment_events)`).Scan(
		&legacyAcceptanceRows, &legacyCompanyRows, &legacyCommitmentRows,
	); err != nil {
		t.Fatalf("read 022 legacy commercial-proof counts: %v", err)
	}
	if legacyAcceptanceRows != 0 || legacyCompanyRows != 0 || legacyCommitmentRows != 0 {
		t.Fatalf(
			"022 inferred legacy proof acceptances=%d companies=%d commitments=%d, want 0/0/0",
			legacyAcceptanceRows, legacyCompanyRows, legacyCommitmentRows,
		)
	}

	// Seed one valid-schema 022 handoff row before 023 exists so the next
	// migration must preserve it as an explicit disclosure decline. This fixture
	// bypasses only the 022 eligibility trigger because its legacy ticket was
	// deliberately revoked for the earlier cutover test; all foreign keys and
	// immutable snapshot constraints remain enforced.
	legacyHandoffTermsSHA256 := postgresMigrationTestHash(capacityFixture.seed + "-legacy-handoff-terms")
	if _, err := DB.Exec(`
		UPDATE provider_offers
		SET commercial_terms_contract_version='nhs-provider-commercial-terms-v1',
		    commercial_terms_sha256=$2
		WHERE id=$1::uuid`, capacityFixture.offerID, legacyHandoffTermsSHA256); err != nil {
		t.Fatalf("bind 022 legacy offer to exact handoff terms: %v", err)
	}
	if _, err := DB.Exec(`
		UPDATE action_tickets
		SET commercial_terms_contract_version_snapshot='nhs-provider-commercial-terms-v1',
		    commercial_terms_sha256_snapshot=$2
		WHERE id=$1::uuid`, legacyTicketID, legacyHandoffTermsSHA256); err != nil {
		t.Fatalf("bind 022 legacy ticket to exact handoff terms: %v", err)
	}
	if _, err := DB.Exec(`
		ALTER TABLE provider_action_handoff_receipts
		DISABLE TRIGGER provider_action_handoff_receipt_enforced`); err != nil {
		t.Fatalf("disable 022 handoff fixture trigger: %v", err)
	}
	var legacyHandoffID string
	insertLegacyHandoffErr := DB.QueryRow(`
		INSERT INTO provider_action_handoff_receipts (
			action_ticket_id, provider_claim_id, provider_offer_id,
			offer_version_snapshot,
			commercial_terms_contract_version_snapshot,
			commercial_terms_sha256_snapshot, presented_token_hash,
			principal_handoff_consent, handoff_consent_version,
			event_contract_version
		)
		SELECT id, provider_claim_id, provider_offer_id,
		       offer_version_snapshot,
		       commercial_terms_contract_version_snapshot,
		       commercial_terms_sha256_snapshot, token_hash,
		       true, 'nhs-provider-handoff-consent-v1',
		       'nhs-action-handoff-v1'
		FROM action_tickets WHERE id=$1::uuid
		RETURNING id::text`, legacyTicketID).Scan(&legacyHandoffID)
	_, reenableLegacyHandoffErr := DB.Exec(`
		ALTER TABLE provider_action_handoff_receipts
		ENABLE TRIGGER provider_action_handoff_receipt_enforced`)
	if insertLegacyHandoffErr != nil {
		t.Fatalf("seed 022 handoff fixture: %v", insertLegacyHandoffErr)
	}
	if reenableLegacyHandoffErr != nil {
		t.Fatalf("re-enable 022 handoff fixture trigger: %v", reenableLegacyHandoffErr)
	}

	t.Run("023 atomic rollback preserves 022", func(t *testing.T) {
		broken := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name == "023_provider_controlled_intent_disclosure.sql" {
				data = append(data, []byte("\nSELECT * FROM nhs_intentional_missing_controlled_intent_relation;\n")...)
			}
			return data, true
		})
		if err := RunMigrations(broken, revision); err == nil || !strings.Contains(err.Error(), "nhs_intentional_missing_controlled_intent_relation") {
			t.Fatalf("broken 023 migration error = %v", err)
		}
		assertPostgresColumnAbsent(t, "provider_action_handoff_receipts", "principal_controlled_intent_disclosure_consent")
		assertPostgresColumnAbsent(t, "provider_action_handoff_receipts", "controlled_intent_disclosure_consent_version")
		assertPostgresFunctionAbsent(t, "enforce_action_ticket_controlled_intent_immutability")
		var trigger023Exists, receipt023Exists, receipt022Exists bool
		if err := DB.QueryRow(`SELECT EXISTS(
			SELECT 1 FROM pg_trigger
			WHERE tgname='action_ticket_controlled_intent_immutability_enforced'
		)`).Scan(&trigger023Exists); err != nil {
			t.Fatal(err)
		}
		if err := DB.QueryRow(`SELECT EXISTS(
			SELECT 1 FROM nhs_schema_migrations
			WHERE name='023_provider_controlled_intent_disclosure.sql'
		)`).Scan(&receipt023Exists); err != nil {
			t.Fatal(err)
		}
		if err := DB.QueryRow(`SELECT EXISTS(
			SELECT 1 FROM nhs_schema_migrations
			WHERE name='022_provider_commercial_proof.sql'
		)`).Scan(&receipt022Exists); err != nil {
			t.Fatal(err)
		}
		if trigger023Exists || receipt023Exists || !receipt022Exists {
			t.Fatalf("failed 023 footprint trigger=%t receipt023=%t receipt022=%t", trigger023Exists, receipt023Exists, receipt022Exists)
		}
	})

	t.Run("ambiguous prior 023 footprint", func(t *testing.T) {
		if _, err := DB.Exec(`
			CREATE FUNCTION public.enforce_action_ticket_controlled_intent_immutability()
			RETURNS TRIGGER
			LANGUAGE plpgsql
			AS $$
			BEGIN
				RETURN NEW;
			END;
			$$`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "ambiguous_prior_023") {
			t.Fatalf("ambiguous 023 footprint error = %v", err)
		}
		if _, err := DB.Exec(`
			DROP FUNCTION public.enforce_action_ticket_controlled_intent_immutability()`); err != nil {
			t.Fatal(err)
		}
	})

	through023 := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
		return data, name <= "023_provider_controlled_intent_disclosure.sql"
	})
	if err := RunMigrations(through023, revision); err != nil {
		t.Fatalf("apply exact 023 migration: %v", err)
	}
	var legacyDisclosureConsent bool
	var legacyDisclosureVersion string
	if err := DB.QueryRow(`
		SELECT principal_controlled_intent_disclosure_consent,
		       controlled_intent_disclosure_consent_version
		FROM provider_action_handoff_receipts
		WHERE id=$1::uuid`, legacyHandoffID).Scan(
		&legacyDisclosureConsent, &legacyDisclosureVersion,
	); err != nil {
		t.Fatalf("read 022 handoff after 023: %v", err)
	}
	if legacyDisclosureConsent || legacyDisclosureVersion != "" {
		t.Fatalf("023 upgraded legacy handoff consent=%t version=%q", legacyDisclosureConsent, legacyDisclosureVersion)
	}

	t.Run("024 atomic rollback preserves 023", func(t *testing.T) {
		broken := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name == "024_provider_pilot_boundary.sql" {
				data = append(data, []byte("\nSELECT * FROM nhs_intentional_missing_provider_pilot_boundary;\n")...)
			}
			return data, true
		})
		if err := RunMigrations(broken, revision); err == nil ||
			!strings.Contains(err.Error(), "nhs_intentional_missing_provider_pilot_boundary") {
			t.Fatalf("broken 024 migration error = %v", err)
		}
		for _, relation := range []string{
			"provider_pilot_epochs",
			"provider_pilot_enrollments",
			"provider_pilot_epoch_events",
			"idx_action_tickets_pilot_epoch_claim",
		} {
			assertPostgresRelationAbsent(t, relation)
		}
		for _, column := range []struct {
			table string
			name  string
		}{
			{table: "provider_offers", name: "provider_pilot_epoch_id"},
			{table: "provider_offers_returned", name: "provider_pilot_epoch_id_snapshot"},
			{table: "action_tickets", name: "provider_pilot_epoch_id"},
		} {
			assertPostgresColumnAbsent(t, column.table, column.name)
		}
		for _, function := range []string{
			"enforce_provider_pilot_epoch_insert",
			"enforce_provider_pilot_enrollment",
			"enforce_provider_pilot_epoch_event",
			"enforce_provider_pilot_offer_binding",
			"enforce_provider_pilot_returned_offer",
			"enforce_action_ticket_pilot_insert",
			"enforce_action_ticket_pilot_binding",
			"enforce_provider_pilot_handoff_insert",
			"enforce_provider_pilot_epoch_transition",
		} {
			assertPostgresFunctionAbsent(t, function)
		}
		var receipt024Exists, receipt023Exists bool
		if err := DB.QueryRow(`SELECT EXISTS(
			SELECT 1 FROM nhs_schema_migrations
			WHERE name='024_provider_pilot_boundary.sql'
		)`).Scan(&receipt024Exists); err != nil {
			t.Fatal(err)
		}
		if err := DB.QueryRow(`SELECT EXISTS(
			SELECT 1 FROM nhs_schema_migrations
			WHERE name='023_provider_controlled_intent_disclosure.sql'
		)`).Scan(&receipt023Exists); err != nil {
			t.Fatal(err)
		}
		if receipt024Exists || !receipt023Exists {
			t.Fatalf("failed 024 receipt024=%t receipt023=%t", receipt024Exists, receipt023Exists)
		}
	})

	t.Run("ambiguous prior 024 footprint", func(t *testing.T) {
		if _, err := DB.Exec(`
			CREATE FUNCTION public.enforce_provider_pilot_epoch_insert()
			RETURNS TRIGGER
			LANGUAGE plpgsql
			AS $$
			BEGIN
				RETURN NEW;
			END;
			$$`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil ||
			!strings.Contains(err.Error(), "ambiguous_prior_024") {
			t.Fatalf("ambiguous 024 footprint error = %v", err)
		}
		if _, err := DB.Exec(`DROP FUNCTION public.enforce_provider_pilot_epoch_insert()`); err != nil {
			t.Fatal(err)
		}
	})

	through024 := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
		return data, name <= "024_provider_pilot_boundary.sql"
	})
	if err := RunMigrations(through024, revision); err != nil {
		t.Fatalf("apply exact 024 migration: %v", err)
	}
	stage1StartedAt := seedProviderPilotStage1MigrationEvidence(t, capacityFixture)
	var stage1AsOf time.Time
	if err := DB.QueryRow(`SELECT clock_timestamp()`).Scan(&stage1AsOf); err != nil {
		t.Fatal(err)
	}
	// This intentionally exercises 024 before 025 adds the integrity-generation
	// columns. Build the exact aggregate fixture in test code; the production
	// model is only valid against the complete protected migration chain.
	stage1Proof := &models.Stage1DemandProof{
		Days: 30, RetentionDays: 30, AsOf: stage1AsOf,
		Stage1StartedAt: stage1StartedAt, Stage1EpochEnforced: true,
		SyntheticExcluded: true, CountsAreReceiptsNotUniqueAgents: true,
		CommercialProof: false, MeaningfulSearchReceipts: 100,
		ResultSelections: 20, SearchReceiptsWithSelection: 20,
		ActionInterestReceipts: 10, SearchReceiptsWithActionInterest: 10,
		DistinctInterestDomains: 10, BucketReceiptThreshold: 20,
		TopicBucketsMayOverlap: true,
		DemandTopics: []models.Stage1DemandBucket{
			{Value: "developer-tools", ReceiptCount: 100},
		},
		PilotCandidateTopics: []models.Stage1DemandBucket{
			{Value: "developer-tools", ReceiptCount: 100},
		},
		ActionTypes:                  []models.Stage1DemandBucket{},
		PilotCandidateTopicAvailable: true,
		ObservationWindowDays:        14,
		ObservationSpanSeconds:       int64(15 * 24 * time.Hour / time.Second),
		ObservationSpanDays:          15,
		ObservationWindowMet:         true,
		Targets: map[string]int{
			"meaningful_search_receipts":           100,
			"search_receipts_with_selection":       20,
			"search_receipts_with_action_interest": 10,
			"pilot_candidate_topic_receipts":       20,
			"observation_window_days":              14,
		},
		TargetsMet: map[string]bool{
			"meaningful_search_receipts":           true,
			"search_receipts_with_selection":       true,
			"search_receipts_with_action_interest": true,
			"pilot_candidate_topic_receipts":       true,
			"observation_window_days":              true,
		},
		Stage1Ready: true,
	}
	stage1EvidenceSHA256, err := models.ProviderPilotStage1SnapshotSHA256(stage1Proof)
	if err != nil {
		t.Fatalf("hash seeded Stage 1 proof: %v", err)
	}
	if !stage1Proof.Stage1StartedAt.Equal(stage1StartedAt) || !stage1Proof.Stage1Ready {
		t.Fatalf("seeded Stage 1 proof start=%s ready=%t", stage1Proof.Stage1StartedAt, stage1Proof.Stage1Ready)
	}

	t.Run("024 database-owned timestamps and draft boundary", func(t *testing.T) {
		if _, err := DB.Exec(`
			INSERT INTO provider_pilot_epochs (
				contract_version, demand_topic, stage1_started_at,
				stage1_evidence_as_of, stage1_evidence_sha256,
				cohort_limit, provider_ticket_cap, total_ticket_cap,
				owner_reference, evidence_reference
			) VALUES (
				'nhs-provider-pilot-v1', 'developer-tools',
				$1::timestamptz - INTERVAL '1 second', $2::timestamptz,
				$3, 10, 1, 10, 'owner:forged-024', 'evidence:forged-024'
			)`, stage1StartedAt, stage1Proof.AsOf, stage1EvidenceSHA256); err == nil {
			t.Fatal("024 accepted a caller-forged Stage 1 start")
		} else if !strings.Contains(err.Error(), "exact completed Stage 1 observation window") {
			t.Fatalf("024 forged Stage 1 start error=%v", err)
		}
		if _, err := DB.Exec(`
			INSERT INTO provider_pilot_epochs (
				contract_version, demand_topic, stage1_started_at,
				stage1_evidence_as_of, stage1_evidence_sha256,
				cohort_limit, provider_ticket_cap, total_ticket_cap,
				owner_reference, evidence_reference
			) VALUES (
				'nhs-provider-pilot-v1', 'developer-tools', $1::timestamptz,
				$2::timestamptz, $3, 10, 1, 10,
				'owner:false-hash-024', 'evidence:false-hash-024'
			)`, stage1StartedAt, stage1Proof.AsOf,
			postgresMigrationTestHash("024-false-stage1-hash")); err == nil {
			t.Fatal("024 accepted a false Stage 1 snapshot hash")
		} else if !strings.Contains(err.Error(), "snapshot hash does not match database facts") {
			t.Fatalf("024 false Stage 1 snapshot hash error=%v", err)
		}
		if _, err := DB.Exec(`
			INSERT INTO provider_pilot_epochs (
				contract_version, demand_topic, stage1_started_at,
				stage1_evidence_as_of, stage1_evidence_sha256,
				cohort_limit, provider_ticket_cap, total_ticket_cap,
				owner_reference, evidence_reference
			) VALUES (
				'nhs-provider-pilot-v1', 'developer-tools', $1::timestamptz,
				$2::timestamptz, $3, 2, 1, 5,
				'owner:undersized-024', 'evidence:undersized-024'
			)`, stage1StartedAt, stage1Proof.AsOf, stage1EvidenceSHA256); err == nil {
			t.Fatal("024 accepted a provider cohort below the three-company minimum")
		} else {
			assertMigrationPostgresConstraintCode(
				t, err, "23514", "provider_pilot_epochs_cohort_limit_check",
			)
		}

		var pilotID string
		var createdAt, updatedAt time.Time
		if err := DB.QueryRow(`
			INSERT INTO provider_pilot_epochs (
				contract_version, demand_topic, stage1_started_at,
				stage1_evidence_as_of, stage1_evidence_sha256,
				cohort_limit, provider_ticket_cap, total_ticket_cap,
				owner_reference, evidence_reference, created_at, updated_at
			) VALUES (
				'nhs-provider-pilot-v1', 'developer-tools',
				$2::timestamptz, $3::timestamptz,
				$1, 10, 1, 10, 'owner:test-024', 'evidence:test-024',
				TIMESTAMPTZ '2000-01-01 00:00:00Z',
				TIMESTAMPTZ '2000-01-01 00:00:00Z'
			)
			RETURNING id::text, created_at, updated_at`,
			stage1EvidenceSHA256, stage1StartedAt, stage1Proof.AsOf,
		).Scan(&pilotID, &createdAt, &updatedAt); err != nil {
			t.Fatalf("insert 024 draft epoch: %v", err)
		}
		if createdAt.Year() == 2000 || !createdAt.Equal(updatedAt) || time.Since(createdAt) > time.Minute {
			t.Fatalf("024 epoch caller-forged timestamps survived created=%s updated=%s", createdAt, updatedAt)
		}

		var eventID string
		var eventCreatedAt time.Time
		if _, err := DB.Exec(`
			INSERT INTO provider_pilot_epoch_events (
				provider_pilot_epoch_id, event_type, event_snapshot_sha256,
				owner_reference, evidence_reference
			) VALUES ($1::uuid, 'created', $2, 'owner:false-event-024',
			          'evidence:false-event-024')`,
			pilotID, postgresMigrationTestHash("024-false-event-hash")); err == nil {
			t.Fatal("024 accepted a false pilot event snapshot hash")
		} else if !strings.Contains(err.Error(), "event snapshot hash does not match database facts") {
			t.Fatalf("024 false event snapshot hash error=%v", err)
		}
		if err := DB.QueryRow(`
			INSERT INTO provider_pilot_epoch_events (
				provider_pilot_epoch_id, event_type, event_snapshot_sha256,
				owner_reference, evidence_reference, created_at
			) VALUES (
				$1::uuid, 'created', $2, 'owner:test-024',
				'evidence:test-024', TIMESTAMPTZ '2000-01-01 00:00:00Z'
			)
			RETURNING id::text, created_at`,
			pilotID, postgresPilotEventSnapshotHash("created", pilotID, ""),
		).Scan(&eventID, &eventCreatedAt); err != nil {
			t.Fatalf("insert 024 created event: %v", err)
		}
		if eventCreatedAt.Year() == 2000 || time.Since(eventCreatedAt) > time.Minute {
			t.Fatalf("024 event caller-forged timestamp survived: %s", eventCreatedAt)
		}
		if _, err := DB.Exec(`
			UPDATE provider_pilot_epoch_events
			SET created_at=TIMESTAMPTZ '2000-01-01 00:00:00Z'
			WHERE id=$1::uuid`, eventID); err != nil {
			t.Fatalf("024 append-only event update rule: %v", err)
		}
		var persistedEventCreatedAt time.Time
		if err := DB.QueryRow(`
			SELECT created_at FROM provider_pilot_epoch_events WHERE id=$1::uuid`, eventID).
			Scan(&persistedEventCreatedAt); err != nil {
			t.Fatal(err)
		}
		if !persistedEventCreatedAt.Equal(eventCreatedAt) {
			t.Fatalf("024 append-only event timestamp changed from %s to %s", eventCreatedAt, persistedEventCreatedAt)
		}

		if _, err := DB.Exec(`
			UPDATE provider_pilot_epochs
			SET status='active', activated_at=TIMESTAMPTZ '2000-01-01 00:00:00Z'
			WHERE id=$1::uuid`, pilotID); err == nil ||
			!strings.Contains(err.Error(), "provider pilot cohort must contain 3") {
			t.Fatalf("024 underfilled activation error=%v", err)
		}
		if _, err := DB.Exec(`
			INSERT INTO provider_pilot_enrollments (
				provider_pilot_epoch_id, provider_pilot_company_id,
				provider_claim_id, owner_reference, evidence_reference
			) VALUES (
				$1::uuid, uuid_generate_v4(), uuid_generate_v4(),
				'owner:test-024', 'evidence:test-024'
			)`, pilotID); err == nil ||
			!strings.Contains(err.Error(), "exact fresh verified company claim") {
			t.Fatalf("024 unbound enrollment error=%v", err)
		}
		if _, err := DB.Exec(`
			UPDATE provider_offers
			SET status='active', provider_pilot_epoch_id=$2::uuid,
				activated_at=statement_timestamp(),
				terms_evidence_reference='evidence:test-024'
			WHERE id=$1::uuid`, capacityFixture.offerID, pilotID); err == nil ||
			!strings.Contains(err.Error(), "active pilot epoch") {
			t.Fatalf("024 draft-epoch offer activation error=%v", err)
		}
	})

	t.Run("025 atomic rollback preserves 024", func(t *testing.T) {
		broken := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name > "025_stage1_fact_integrity.sql" {
				return nil, false
			}
			if name == "025_stage1_fact_integrity.sql" {
				data = append(data, []byte("\nSELECT * FROM nhs_intentional_missing_stage1_fact_integrity;\n")...)
			}
			return data, true
		})
		if err := RunMigrations(broken, revision); err == nil ||
			!strings.Contains(err.Error(), "nhs_intentional_missing_stage1_fact_integrity") {
			t.Fatalf("broken 025 migration error = %v", err)
		}
		for _, function := range []string{
			"enforce_search_receipt_stage1_immutability",
			"enforce_organic_result_stage1_immutability",
			"enforce_result_selection_stage1_immutability",
			"own_search_receipt_created_at",
			"own_organic_result_returned_at",
			"own_result_selection_selected_at",
			"own_action_interest_created_at",
			"lock_provider_pilot_stage1_epoch_anchor",
			"enforce_provider_pilot_stage1_generation",
		} {
			assertPostgresFunctionAbsent(t, function)
		}
		for _, table := range []string{
			"search_receipts", "organic_results_returned",
			"result_selections", "action_interest_receipts",
		} {
			assertPostgresColumnAbsent(t, table, "stage1_integrity_generation")
		}
		var exactSelectionConstraintExists bool
		if err := DB.QueryRow(`SELECT EXISTS(
			SELECT 1 FROM pg_catalog.pg_constraint
			WHERE conname='result_selections_returned_result_fk'
		)`).Scan(&exactSelectionConstraintExists); err != nil {
			t.Fatal(err)
		}
		if exactSelectionConstraintExists {
			t.Fatal("failed 025 migration left its exact-selection constraint")
		}
		var receipt025Exists, receipt024Exists bool
		if err := DB.QueryRow(`SELECT EXISTS(
			SELECT 1 FROM nhs_schema_migrations
			WHERE name='025_stage1_fact_integrity.sql'
		)`).Scan(&receipt025Exists); err != nil {
			t.Fatal(err)
		}
		if err := DB.QueryRow(`SELECT EXISTS(
			SELECT 1 FROM nhs_schema_migrations
			WHERE name='024_provider_pilot_boundary.sql'
		)`).Scan(&receipt024Exists); err != nil {
			t.Fatal(err)
		}
		if receipt025Exists || !receipt024Exists {
			t.Fatalf("failed 025 receipt025=%t receipt024=%t", receipt025Exists, receipt024Exists)
		}
	})

	t.Run("ambiguous prior 025 footprint", func(t *testing.T) {
		if _, err := DB.Exec(`
			CREATE FUNCTION public.own_search_receipt_created_at()
			RETURNS TRIGGER
			LANGUAGE plpgsql
			AS $$
			BEGIN
				RETURN NEW;
			END;
			$$`); err != nil {
			t.Fatal(err)
		}
		through025 := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			return data, name <= "025_stage1_fact_integrity.sql"
		})
		if err := RunMigrations(through025, revision); err == nil ||
			!strings.Contains(err.Error(), "ambiguous_prior_025") {
			t.Fatalf("ambiguous 025 footprint error = %v", err)
		}
		if _, err := DB.Exec(`DROP FUNCTION public.own_search_receipt_created_at()`); err != nil {
			t.Fatal(err)
		}
	})

	through025 := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
		return data, name <= "025_stage1_fact_integrity.sql"
	})
	if err := RunMigrations(through025, revision); err != nil {
		t.Fatalf("apply exact 025 migration: %v", err)
	}

	t.Run("025 quarantines every pre-integrity fact generation", func(t *testing.T) {
		var legacySearches, taggedLegacySearches int
		if err := DB.QueryRow(`
			SELECT COUNT(*)::int,
			       COUNT(*) FILTER (WHERE stage1_integrity_generation IS NOT NULL)::int
			FROM search_receipts`).Scan(&legacySearches, &taggedLegacySearches); err != nil {
			t.Fatal(err)
		}
		if legacySearches < 100 || taggedLegacySearches != 0 {
			t.Fatalf("025 legacy search quarantine rows=%d tagged=%d", legacySearches, taggedLegacySearches)
		}

		var originalAnchor time.Time
		if err := DB.QueryRow(`
			SELECT applied_at FROM nhs_schema_migrations
			WHERE name='025_stage1_fact_integrity.sql'`).Scan(&originalAnchor); err != nil {
			t.Fatal(err)
		}
		forgedAnchor := time.Now().UTC().Add(-20 * 24 * time.Hour)
		setPostgresProtectedMigrationAppliedAtFixture(
			t, "025_stage1_fact_integrity.sql", forgedAnchor,
		)
		defer setPostgresProtectedMigrationAppliedAtFixture(
			t, "025_stage1_fact_integrity.sql", originalAnchor,
		)
		if _, err := DB.Exec(`
			INSERT INTO provider_pilot_epochs (
				contract_version, demand_topic, stage1_started_at,
				stage1_evidence_as_of, stage1_evidence_sha256,
				cohort_limit, provider_ticket_cap, total_ticket_cap,
				owner_reference, evidence_reference
			) VALUES (
				'nhs-provider-pilot-v1', 'developer-tools', $1,
				clock_timestamp(), $2, 10, 1, 10,
				'owner:legacy-generation-025',
				'evidence:legacy-generation-025'
			)`, forgedAnchor, postgresMigrationTestHash("025-legacy-generation")); err == nil {
			t.Fatal("025 accepted pre-integrity facts after their caller clocks became timestamp-eligible")
		} else {
			assertMigrationPostgresConstraintCode(
				t, err, "23514", "provider_pilot_stage1_integrity_generation",
			)
		}
	})

	t.Run("025 owns and freezes Stage 1 facts", func(t *testing.T) {
		var stage1Anchor time.Time
		if err := DB.QueryRow(`
			SELECT applied_at FROM nhs_schema_migrations
			WHERE name='025_stage1_fact_integrity.sql'`).Scan(&stage1Anchor); err != nil {
			t.Fatal(err)
		}
		if result, err := DB.Exec(`
			UPDATE nhs_schema_migrations
			SET applied_at=TIMESTAMPTZ '2000-01-01 00:00:00Z'
			WHERE name='025_stage1_fact_integrity.sql'`); err != nil {
			t.Fatalf("exercise 025 ledger update rule: %v", err)
		} else if affected, _ := result.RowsAffected(); affected != 0 {
			t.Fatalf("025 ledger update affected %d rows", affected)
		}
		if result, err := DB.Exec(`
			DELETE FROM nhs_schema_migrations
			WHERE name='025_stage1_fact_integrity.sql'`); err != nil {
			t.Fatalf("exercise 025 ledger delete rule: %v", err)
		} else if affected, _ := result.RowsAffected(); affected != 0 {
			t.Fatalf("025 ledger delete affected %d rows", affected)
		}
		var persistedAnchor time.Time
		if err := DB.QueryRow(`
			SELECT applied_at FROM nhs_schema_migrations
			WHERE name='025_stage1_fact_integrity.sql'`).Scan(&persistedAnchor); err != nil {
			t.Fatal(err)
		}
		if !persistedAnchor.Equal(stage1Anchor) {
			t.Fatalf("025 changed protected Stage 1 anchor from %s to %s", stage1Anchor, persistedAnchor)
		}

		var receiptID string
		var searchCreatedAt time.Time
		if err := DB.QueryRow(`
			INSERT INTO search_receipts (
				public_id, surface, explicit_category, demand_topics,
				result_count, page_number, page_size, is_synthetic, created_at
			) VALUES (
				'nhs_stage1_integrity_025', 'rest', 'developer',
				ARRAY['developer-tools']::text[], 1, 1, 10, false,
				TIMESTAMPTZ '2000-01-01 00:00:00Z'
			)
			RETURNING id::text, created_at`).Scan(&receiptID, &searchCreatedAt); err != nil {
			t.Fatalf("insert 025 search fact: %v", err)
		}
		var returnedID, selectionID int64
		var returnedAt, selectedAt time.Time
		if err := DB.QueryRow(`
			INSERT INTO organic_results_returned (
				search_receipt_id, site_id, site_domain_snapshot,
				organic_position, score_snapshot, returned_at
			) VALUES (
				$1::uuid, $2::uuid, $3, 1, 90,
				TIMESTAMPTZ '2000-01-01 00:00:00Z'
			)
			RETURNING id, returned_at`, receiptID, capacityFixture.siteID, capacityFixture.domain).
			Scan(&returnedID, &returnedAt); err != nil {
			t.Fatalf("insert 025 organic fact: %v", err)
		}
		if err := DB.QueryRow(`
			INSERT INTO result_selections (
				search_receipt_id, site_id, site_domain_snapshot, surface, selected_at
			) VALUES (
				$1::uuid, $2::uuid, $3, 'rest',
				TIMESTAMPTZ '2000-01-01 00:00:00Z'
			)
			RETURNING id, selected_at`, receiptID, capacityFixture.siteID, capacityFixture.domain).
			Scan(&selectionID, &selectedAt); err != nil {
			t.Fatalf("insert 025 selection fact: %v", err)
		}
		var interestID string
		var interestCreatedAt, interestExpiresAt time.Time
		if err := DB.QueryRow(`
			INSERT INTO action_interest_receipts (
				public_id, search_receipt_id, source_is_synthetic,
				site_domain_snapshot, action_type, surface,
				caller_attests_principal_interest, confirmation_version,
				created_at, expires_at
			) VALUES (
				'nhs_air_GGGGGGGGGGGGGGGG', $1::uuid, false, $2,
				'trial', 'rest', true, 'nhs-action-interest-v1',
				TIMESTAMPTZ '2000-01-01 00:00:00Z',
				clock_timestamp()+INTERVAL '1 day'
			)
			RETURNING id::text, created_at, expires_at`, receiptID, capacityFixture.domain).
			Scan(&interestID, &interestCreatedAt, &interestExpiresAt); err != nil {
			t.Fatalf("insert 025 action-interest fact: %v", err)
		}
		for label, observed := range map[string]time.Time{
			"search": searchCreatedAt, "organic": returnedAt,
			"selection": selectedAt, "action_interest": interestCreatedAt,
		} {
			if observed.Year() == 2000 || time.Since(observed) < 0 || time.Since(observed) > time.Minute {
				t.Fatalf("025 %s timestamp was not database-owned: %s", label, observed)
			}
		}
		if want := searchCreatedAt.Add(30 * 24 * time.Hour); !interestExpiresAt.Equal(want) {
			t.Fatalf("025 action-interest expiry=%s, want database-derived source boundary %s", interestExpiresAt, want)
		}
		var searchGeneration, returnedGeneration, selectionGeneration, interestGeneration int
		if err := DB.QueryRow(`
			SELECT
			  (SELECT stage1_integrity_generation FROM search_receipts WHERE id=$1::uuid),
			  (SELECT stage1_integrity_generation FROM organic_results_returned WHERE id=$2),
			  (SELECT stage1_integrity_generation FROM result_selections WHERE id=$3),
			  (SELECT stage1_integrity_generation FROM action_interest_receipts WHERE id=$4::uuid)`,
			receiptID, returnedID, selectionID, interestID).Scan(
			&searchGeneration, &returnedGeneration, &selectionGeneration, &interestGeneration,
		); err != nil {
			t.Fatal(err)
		}
		if searchGeneration != 1 || returnedGeneration != 1 ||
			selectionGeneration != 1 || interestGeneration != 1 {
			t.Fatalf("025 owned generations search=%d returned=%d selection=%d interest=%d",
				searchGeneration, returnedGeneration, selectionGeneration, interestGeneration)
		}

		for _, mutation := range []struct {
			query, constraint string
			args              []any
		}{
			{`UPDATE search_receipts SET demand_topics=ARRAY['jobs']::text[] WHERE id=$1::uuid`, "search_receipt_stage1_immutable", []any{receiptID}},
			{`UPDATE organic_results_returned SET score_snapshot=0 WHERE id=$1`, "organic_result_stage1_immutable", []any{returnedID}},
			{`UPDATE result_selections SET surface='mcp' WHERE id=$1`, "result_selection_stage1_immutable", []any{selectionID}},
			{`UPDATE result_selections SET stage1_integrity_generation=NULL WHERE id=$1`, "result_selection_stage1_immutable", []any{selectionID}},
		} {
			if _, err := DB.Exec(mutation.query, mutation.args...); err == nil {
				t.Fatalf("025 accepted mutation for %s", mutation.constraint)
			} else {
				assertMigrationPostgresConstraintCode(t, err, "23514", mutation.constraint)
			}
		}
		if _, err := DB.Exec(`
			UPDATE action_interest_receipts
			SET expires_at=clock_timestamp()+INTERVAL '2 days'
			WHERE id=$1::uuid`, interestID); err != nil {
			t.Fatalf("exercise 025 action-interest append-only rule: %v", err)
		}
		var persistedInterestExpiry time.Time
		if err := DB.QueryRow(`SELECT expires_at FROM action_interest_receipts WHERE id=$1::uuid`, interestID).
			Scan(&persistedInterestExpiry); err != nil {
			t.Fatal(err)
		}
		if !persistedInterestExpiry.Equal(interestExpiresAt) {
			t.Fatalf("025 changed action-interest expiry from %s to %s", interestExpiresAt, persistedInterestExpiry)
		}

		lockTx, err := DB.Begin()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := lockTx.Exec(`
			SELECT 1 FROM nhs_schema_migrations
			WHERE name='025_stage1_fact_integrity.sql'
			FOR NO KEY UPDATE`); err != nil {
			_ = lockTx.Rollback()
			t.Fatal(err)
		}
		insertDone := make(chan error, 1)
		go func() {
			_, insertErr := DB.Exec(`INSERT INTO provider_pilot_epochs DEFAULT VALUES`)
			insertDone <- insertErr
		}()
		select {
		case insertErr := <-insertDone:
			_ = lockTx.Rollback()
			t.Fatalf("025 pilot insert bypassed UPDATE-strength Stage 1 anchor lock: %v", insertErr)
		case <-time.After(200 * time.Millisecond):
		}
		if err := lockTx.Rollback(); err != nil {
			t.Fatal(err)
		}
		select {
		case insertErr := <-insertDone:
			if insertErr == nil {
				t.Fatal("025 invalid pilot insert unexpectedly succeeded after anchor lock release")
			}
		case <-time.After(5 * time.Second):
			t.Fatal("025 pilot insert did not resume after Stage 1 anchor lock release")
		}

		if _, err := DB.Exec(`DELETE FROM search_receipts WHERE id=$1::uuid`, receiptID); err != nil {
			t.Fatalf("clean 025 Stage 1 fact fixture: %v", err)
		}
	})

	t.Run("026 rejects incomplete inherited lifecycle", func(t *testing.T) {
		var eventID, pilotID, eventType, snapshotSHA, ownerReference, evidenceReference string
		var providerClaimID sql.NullString
		if err := DB.QueryRow(`
			SELECT event.id::text, event.provider_pilot_epoch_id::text,
			       event.event_type, event.provider_claim_id::text,
			       event.event_snapshot_sha256, event.owner_reference,
			       event.evidence_reference
			FROM provider_pilot_epoch_events event
			WHERE event.event_type='created'
			ORDER BY event.created_at, event.id
			LIMIT 1`).Scan(
			&eventID, &pilotID, &eventType, &providerClaimID,
			&snapshotSHA, &ownerReference, &evidenceReference,
		); err != nil {
			t.Fatal(err)
		}
		deletePostgresPilotEventFixture(t, eventID)
		migrationErr := RunMigrations(repositoryMigrations, revision)
		if migrationErr == nil {
			t.Fatal("incomplete inherited lifecycle unexpectedly accepted migration 026")
		}
		assertMigrationPostgresConstraintCode(
			t, migrationErr, "23514", "provider_pilot_lifecycle_legacy_incomplete",
		)
		var receipt026Exists bool
		if err := DB.QueryRow(`SELECT EXISTS(
			SELECT 1 FROM nhs_schema_migrations
			WHERE name='026_provider_pilot_proof_integrity.sql'
		)`).Scan(&receipt026Exists); err != nil {
			t.Fatal(err)
		}
		if receipt026Exists {
			t.Fatal("failed lifecycle preflight wrote a 026 receipt")
		}
		if _, err := DB.Exec(`
			INSERT INTO provider_pilot_epoch_events (
				id, provider_pilot_epoch_id, event_type, provider_claim_id,
				event_snapshot_sha256, owner_reference, evidence_reference
			) VALUES ($1::uuid,$2::uuid,$3,NULLIF($4,'')::uuid,$5,$6,$7)`,
			eventID, pilotID, eventType, providerClaimID.String,
			snapshotSHA, ownerReference, evidenceReference); err != nil {
			t.Fatalf("restore inherited lifecycle event fixture: %v", err)
		}
	})

	t.Run("026 atomic rollback preserves 025", func(t *testing.T) {
		broken := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name > "026_provider_pilot_proof_integrity.sql" {
				return nil, false
			}
			if name == "026_provider_pilot_proof_integrity.sql" {
				data = append(data, []byte("\nSELECT * FROM nhs_intentional_missing_provider_pilot_proof_integrity;\n")...)
			}
			return data, true
		})
		if err := RunMigrations(broken, revision); err == nil ||
			!strings.Contains(err.Error(), "nhs_intentional_missing_provider_pilot_proof_integrity") {
			t.Fatalf("broken 026 migration error = %v", err)
		}
		for _, function := range []string{
			"enforce_provider_pilot_outcome_receipt",
			"require_provider_pilot_lifecycle_event",
		} {
			assertPostgresFunctionAbsent(t, function)
		}
		var receipt026Exists, receipt025Exists bool
		if err := DB.QueryRow(`SELECT EXISTS(
			SELECT 1 FROM nhs_schema_migrations
			WHERE name='026_provider_pilot_proof_integrity.sql'
		)`).Scan(&receipt026Exists); err != nil {
			t.Fatal(err)
		}
		if err := DB.QueryRow(`SELECT EXISTS(
			SELECT 1 FROM nhs_schema_migrations
			WHERE name='025_stage1_fact_integrity.sql'
		)`).Scan(&receipt025Exists); err != nil {
			t.Fatal(err)
		}
		if receipt026Exists || !receipt025Exists {
			t.Fatalf("failed 026 receipt026=%t receipt025=%t", receipt026Exists, receipt025Exists)
		}
	})

	t.Run("ambiguous prior 026 footprint", func(t *testing.T) {
		if _, err := DB.Exec(`
			CREATE FUNCTION public.enforce_provider_pilot_outcome_receipt()
			RETURNS TRIGGER
			LANGUAGE plpgsql
			AS $$
			BEGIN
				RETURN NEW;
			END;
			$$`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil ||
			!strings.Contains(err.Error(), "ambiguous_prior_026") {
			t.Fatalf("ambiguous 026 footprint error = %v", err)
		}
		if _, err := DB.Exec(`DROP FUNCTION public.enforce_provider_pilot_outcome_receipt()`); err != nil {
			t.Fatal(err)
		}
	})

	through026 := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
		return data, name <= "026_provider_pilot_proof_integrity.sql"
	})
	if err := RunMigrations(through026, revision); err != nil {
		t.Fatalf("apply exact 026 migration: %v", err)
	}

	t.Run("027 atomic rollback preserves 026", func(t *testing.T) {
		broken := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name > "027_provider_pilot_review_evidence.sql" {
				return nil, false
			}
			if name == "027_provider_pilot_review_evidence.sql" {
				data = append(data, []byte("\nSELECT * FROM nhs_intentional_missing_provider_pilot_review_evidence;\n")...)
			}
			return data, true
		})
		if err := RunMigrations(broken, revision); err == nil ||
			!strings.Contains(err.Error(), "nhs_intentional_missing_provider_pilot_review_evidence") {
			t.Fatalf("broken 027 migration error = %v", err)
		}
		for _, relation := range []string{
			"provider_pilot_review_events",
			"idx_provider_pilot_reviews_pilot_type",
			"idx_provider_pilot_reviews_claim_type",
		} {
			assertPostgresRelationAbsent(t, relation)
		}
		for _, function := range []string{
			"provider_pilot_review_snapshot_sha256",
			"enforce_provider_pilot_review_event",
			"enforce_provider_pilot_epoch_provider_reviews",
			"enforce_provider_offer_pre_activation_review",
			"enforce_provider_handoff_ticket_review",
		} {
			assertPostgresFunctionAbsent(t, function)
		}
		var receipt027Exists, receipt026Exists bool
		if err := DB.QueryRow(`SELECT EXISTS(
			SELECT 1 FROM nhs_schema_migrations
			WHERE name='027_provider_pilot_review_evidence.sql'
		)`).Scan(&receipt027Exists); err != nil {
			t.Fatal(err)
		}
		if err := DB.QueryRow(`SELECT EXISTS(
			SELECT 1 FROM nhs_schema_migrations
			WHERE name='026_provider_pilot_proof_integrity.sql'
		)`).Scan(&receipt026Exists); err != nil {
			t.Fatal(err)
		}
		if receipt027Exists || !receipt026Exists {
			t.Fatalf("failed 027 receipt027=%t receipt026=%t", receipt027Exists, receipt026Exists)
		}
	})

	t.Run("ambiguous prior 027 standalone function footprint", func(t *testing.T) {
		if _, err := DB.Exec(`
			CREATE FUNCTION public.provider_pilot_review_snapshot_sha256(
				requested_pilot_id UUID,
				requested_review_type TEXT,
				requested_subject_id UUID
			)
			RETURNS TEXT
			LANGUAGE sql
			STABLE
			STRICT
			AS $$ SELECT repeat('0', 64) $$`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil ||
			!strings.Contains(err.Error(), "ambiguous_prior_027") {
			t.Fatalf("ambiguous 027 footprint error = %v", err)
		}
		if _, err := DB.Exec(`
			DROP FUNCTION public.provider_pilot_review_snapshot_sha256(UUID, TEXT, UUID)`); err != nil {
			t.Fatal(err)
		}
	})

	through027 := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
		return data, name <= "027_provider_pilot_review_evidence.sql"
	})
	if err := RunMigrations(through027, revision); err != nil {
		t.Fatalf("apply exact 027 migration: %v", err)
	}

	t.Run("028 atomic rollback preserves 027", func(t *testing.T) {
		broken := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name > "028_provider_commercial_proof_manifest.sql" {
				return nil, false
			}
			if name == "028_provider_commercial_proof_manifest.sql" {
				data = append(data, []byte("\nSELECT * FROM nhs_intentional_missing_provider_commercial_proof_manifest;\n")...)
			}
			return data, true
		})
		if err := RunMigrations(broken, revision); err == nil ||
			!strings.Contains(err.Error(), "nhs_intentional_missing_provider_commercial_proof_manifest") {
			t.Fatalf("broken 028 migration error = %v", err)
		}
		for _, relation := range []string{
			"provider_commercial_proof_manifests",
			"idx_provider_proof_manifests_issued",
			"idx_provider_proof_manifests_key",
		} {
			assertPostgresRelationAbsent(t, relation)
		}
		assertPostgresFunctionAbsent(t, "enforce_provider_commercial_proof_manifest")
		var receipt028Exists, receipt027Exists bool
		if err := DB.QueryRow(`SELECT EXISTS(
			SELECT 1 FROM nhs_schema_migrations
			WHERE name='028_provider_commercial_proof_manifest.sql'
		)`).Scan(&receipt028Exists); err != nil {
			t.Fatal(err)
		}
		if err := DB.QueryRow(`SELECT EXISTS(
			SELECT 1 FROM nhs_schema_migrations
			WHERE name='027_provider_pilot_review_evidence.sql'
		)`).Scan(&receipt027Exists); err != nil {
			t.Fatal(err)
		}
		if receipt028Exists || !receipt027Exists {
			t.Fatalf("failed 028 receipt028=%t receipt027=%t", receipt028Exists, receipt027Exists)
		}
	})

	t.Run("ambiguous prior 028 trigger-function footprint", func(t *testing.T) {
		if _, err := DB.Exec(`
			CREATE FUNCTION public.enforce_provider_commercial_proof_manifest()
			RETURNS TRIGGER
			LANGUAGE plpgsql
			AS $$
			BEGIN
				RETURN NEW;
			END;
			$$`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil ||
			!strings.Contains(err.Error(), "ambiguous_prior_028") {
			t.Fatalf("ambiguous 028 footprint error = %v", err)
		}
		if _, err := DB.Exec(`DROP FUNCTION public.enforce_provider_commercial_proof_manifest()`); err != nil {
			t.Fatal(err)
		}
	})

	if err := RunMigrations(repositoryMigrations, revision); err != nil {
		t.Fatalf("apply exact 029 migration: %v", err)
	}

	t.Run("concurrent exact replay", func(t *testing.T) {
		var wait sync.WaitGroup
		errorsByRunner := make([]error, 2)
		for i := range errorsByRunner {
			wait.Add(1)
			go func(index int) {
				defer wait.Done()
				errorsByRunner[index] = RunMigrations(repositoryMigrations, revision)
			}(i)
		}
		wait.Wait()
		for i, err := range errorsByRunner {
			if err != nil {
				t.Fatalf("migration runner %d: %v", i, err)
			}
		}
	})

	migrationData, err := os.ReadFile(filepath.Join(repositoryMigrations, "019_provider_exchange.sql"))
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(migrationData)
	wantSHA := hex.EncodeToString(digest[:])
	var gotSHA, gotSchemaSHA, gotRevision string
	var appliedAt time.Time
	if err := DB.QueryRow(`
		SELECT sha256, schema_sha256, applied_by_commit, applied_at
		FROM nhs_schema_migrations
		WHERE name = '019_provider_exchange.sql'`).Scan(&gotSHA, &gotSchemaSHA, &gotRevision, &appliedAt); err != nil {
		t.Fatal(err)
	}
	if gotSHA != wantSHA || len(gotSchemaSHA) != 64 || gotRevision != revision {
		t.Fatalf("protected receipt sha=%q schema_sha_length=%d revision=%q", gotSHA, len(gotSchemaSHA), gotRevision)
	}
	migration020Data, err := os.ReadFile(filepath.Join(repositoryMigrations, "020_action_interest_receipts.sql"))
	if err != nil {
		t.Fatal(err)
	}
	digest020 := sha256.Sum256(migration020Data)
	want020SHA := hex.EncodeToString(digest020[:])
	var got020SHA, got020SchemaSHA, got020Revision string
	var applied020At time.Time
	if err := DB.QueryRow(`
		SELECT sha256, schema_sha256, applied_by_commit, applied_at
		FROM nhs_schema_migrations
		WHERE name = '020_action_interest_receipts.sql'`).Scan(&got020SHA, &got020SchemaSHA, &got020Revision, &applied020At); err != nil {
		t.Fatal(err)
	}
	if got020SHA != want020SHA || len(got020SchemaSHA) != 64 || got020Revision != revision {
		t.Fatalf("020 receipt sha=%q schema_sha_length=%d revision=%q", got020SHA, len(got020SchemaSHA), got020Revision)
	}
	migration021Data, err := os.ReadFile(filepath.Join(repositoryMigrations, "021_provider_capacity_reservations.sql"))
	if err != nil {
		t.Fatal(err)
	}
	capacityReservationFunctionSQL := ""
	for _, statement := range migrationStatements(string(migration021Data)) {
		if strings.Contains(statement, "CREATE OR REPLACE FUNCTION public.enforce_action_ticket_capacity_reservation()") {
			capacityReservationFunctionSQL = statement
			break
		}
	}
	if capacityReservationFunctionSQL == "" {
		t.Fatal("021 capacity reservation trigger function statement not found")
	}
	digest021 := sha256.Sum256(migration021Data)
	want021SHA := hex.EncodeToString(digest021[:])
	var got021SHA, got021SchemaSHA, got021Revision string
	var applied021At time.Time
	if err := DB.QueryRow(`
		SELECT sha256, schema_sha256, applied_by_commit, applied_at
		FROM nhs_schema_migrations
		WHERE name = '021_provider_capacity_reservations.sql'`).Scan(&got021SHA, &got021SchemaSHA, &got021Revision, &applied021At); err != nil {
		t.Fatal(err)
	}
	if got021SHA != want021SHA || len(got021SchemaSHA) != 64 || got021Revision != revision {
		t.Fatalf("021 receipt sha=%q schema_sha_length=%d revision=%q", got021SHA, len(got021SchemaSHA), got021Revision)
	}
	migration022Data, err := os.ReadFile(filepath.Join(repositoryMigrations, "022_provider_commercial_proof.sql"))
	if err != nil {
		t.Fatal(err)
	}
	commercialCommitmentFunctionSQL := ""
	offerCommercialImmutabilityFunctionSQL := ""
	for _, statement := range migrationStatements(string(migration022Data)) {
		switch {
		case strings.Contains(statement, "CREATE OR REPLACE FUNCTION public.enforce_provider_commercial_commitment_event()"):
			commercialCommitmentFunctionSQL = statement
		case strings.Contains(statement, "CREATE OR REPLACE FUNCTION public.enforce_provider_offer_commercial_immutability()"):
			offerCommercialImmutabilityFunctionSQL = statement
		}
	}
	if commercialCommitmentFunctionSQL == "" {
		t.Fatal("022 commercial commitment trigger function statement not found")
	}
	if offerCommercialImmutabilityFunctionSQL == "" {
		t.Fatal("022 provider-offer commercial immutability trigger function statement not found")
	}
	digest022 := sha256.Sum256(migration022Data)
	want022SHA := hex.EncodeToString(digest022[:])
	var got022SHA, got022SchemaSHA, got022Revision string
	var applied022At time.Time
	if err := DB.QueryRow(`
		SELECT sha256, schema_sha256, applied_by_commit, applied_at
		FROM nhs_schema_migrations
		WHERE name = '022_provider_commercial_proof.sql'`).Scan(&got022SHA, &got022SchemaSHA, &got022Revision, &applied022At); err != nil {
		t.Fatal(err)
	}
	if got022SHA != want022SHA || len(got022SchemaSHA) != 64 || got022Revision != revision {
		t.Fatalf("022 receipt sha=%q schema_sha_length=%d revision=%q", got022SHA, len(got022SchemaSHA), got022Revision)
	}
	migration023Data, err := os.ReadFile(filepath.Join(repositoryMigrations, "023_provider_controlled_intent_disclosure.sql"))
	if err != nil {
		t.Fatal(err)
	}
	controlledIntentImmutabilityFunctionSQL := ""
	for _, statement := range migrationStatements(string(migration023Data)) {
		if strings.Contains(statement, "CREATE OR REPLACE FUNCTION public.enforce_action_ticket_controlled_intent_immutability()") {
			controlledIntentImmutabilityFunctionSQL = statement
			break
		}
	}
	if controlledIntentImmutabilityFunctionSQL == "" {
		t.Fatal("023 controlled-intent immutability trigger function statement not found")
	}
	digest023 := sha256.Sum256(migration023Data)
	want023SHA := hex.EncodeToString(digest023[:])
	var got023SHA, got023SchemaSHA, got023Revision string
	var applied023At time.Time
	if err := DB.QueryRow(`
		SELECT sha256, schema_sha256, applied_by_commit, applied_at
		FROM nhs_schema_migrations
		WHERE name = '023_provider_controlled_intent_disclosure.sql'`).Scan(
		&got023SHA, &got023SchemaSHA, &got023Revision, &applied023At,
	); err != nil {
		t.Fatal(err)
	}
	if got023SHA != want023SHA || len(got023SchemaSHA) != 64 || got023Revision != revision {
		t.Fatalf("023 receipt sha=%q schema_sha_length=%d revision=%q", got023SHA, len(got023SchemaSHA), got023Revision)
	}
	migration024Data, err := os.ReadFile(filepath.Join(repositoryMigrations, "024_provider_pilot_boundary.sql"))
	if err != nil {
		t.Fatal(err)
	}
	digest024 := sha256.Sum256(migration024Data)
	want024SHA := hex.EncodeToString(digest024[:])
	var got024SHA, got024SchemaSHA, got024Revision string
	var applied024At time.Time
	if err := DB.QueryRow(`
		SELECT sha256, schema_sha256, applied_by_commit, applied_at
		FROM nhs_schema_migrations
		WHERE name = '024_provider_pilot_boundary.sql'`).Scan(
		&got024SHA, &got024SchemaSHA, &got024Revision, &applied024At,
	); err != nil {
		t.Fatal(err)
	}
	if got024SHA != want024SHA || len(got024SchemaSHA) != 64 || got024Revision != revision {
		t.Fatalf("024 receipt sha=%q schema_sha_length=%d revision=%q", got024SHA, len(got024SchemaSHA), got024Revision)
	}
	migration025Data, err := os.ReadFile(filepath.Join(repositoryMigrations, "025_stage1_fact_integrity.sql"))
	if err != nil {
		t.Fatal(err)
	}
	digest025 := sha256.Sum256(migration025Data)
	want025SHA := hex.EncodeToString(digest025[:])
	var got025SHA, got025SchemaSHA, got025Revision string
	var applied025At time.Time
	if err := DB.QueryRow(`
		SELECT sha256, schema_sha256, applied_by_commit, applied_at
		FROM nhs_schema_migrations
		WHERE name = '025_stage1_fact_integrity.sql'`).Scan(
		&got025SHA, &got025SchemaSHA, &got025Revision, &applied025At,
	); err != nil {
		t.Fatal(err)
	}
	if got025SHA != want025SHA || len(got025SchemaSHA) != 64 || got025Revision != revision {
		t.Fatalf("025 receipt sha=%q schema_sha_length=%d revision=%q", got025SHA, len(got025SchemaSHA), got025Revision)
	}
	migration026Data, err := os.ReadFile(filepath.Join(repositoryMigrations, "026_provider_pilot_proof_integrity.sql"))
	if err != nil {
		t.Fatal(err)
	}
	digest026 := sha256.Sum256(migration026Data)
	want026SHA := hex.EncodeToString(digest026[:])
	var got026SHA, got026SchemaSHA, got026Revision string
	var applied026At time.Time
	if err := DB.QueryRow(`
		SELECT sha256, schema_sha256, applied_by_commit, applied_at
		FROM nhs_schema_migrations
		WHERE name = '026_provider_pilot_proof_integrity.sql'`).Scan(
		&got026SHA, &got026SchemaSHA, &got026Revision, &applied026At,
	); err != nil {
		t.Fatal(err)
	}
	if got026SHA != want026SHA || len(got026SchemaSHA) != 64 || got026Revision != revision {
		t.Fatalf("026 receipt sha=%q schema_sha_length=%d revision=%q", got026SHA, len(got026SchemaSHA), got026Revision)
	}
	migration027Data, err := os.ReadFile(filepath.Join(repositoryMigrations, "027_provider_pilot_review_evidence.sql"))
	if err != nil {
		t.Fatal(err)
	}
	providerPilotReviewSnapshotFunctionSQL := ""
	providerPilotReviewEnforcementFunctionSQL := ""
	for _, statement := range migrationStatements(string(migration027Data)) {
		switch {
		case strings.Contains(statement, "CREATE OR REPLACE FUNCTION public.provider_pilot_review_snapshot_sha256("):
			providerPilotReviewSnapshotFunctionSQL = statement
		case strings.Contains(statement, "CREATE OR REPLACE FUNCTION public.enforce_provider_pilot_review_event()"):
			providerPilotReviewEnforcementFunctionSQL = statement
		}
	}
	if providerPilotReviewSnapshotFunctionSQL == "" {
		t.Fatal("027 provider-pilot review snapshot function statement not found")
	}
	if providerPilotReviewEnforcementFunctionSQL == "" {
		t.Fatal("027 provider-pilot review enforcement function statement not found")
	}
	digest027 := sha256.Sum256(migration027Data)
	want027SHA := hex.EncodeToString(digest027[:])
	var got027SHA, got027SchemaSHA, got027Revision string
	var applied027At time.Time
	if err := DB.QueryRow(`
		SELECT sha256, schema_sha256, applied_by_commit, applied_at
		FROM nhs_schema_migrations
		WHERE name = '027_provider_pilot_review_evidence.sql'`).Scan(
		&got027SHA, &got027SchemaSHA, &got027Revision, &applied027At,
	); err != nil {
		t.Fatal(err)
	}
	if got027SHA != want027SHA || len(got027SchemaSHA) != 64 || got027Revision != revision {
		t.Fatalf("027 receipt sha=%q schema_sha_length=%d revision=%q", got027SHA, len(got027SchemaSHA), got027Revision)
	}
	migration028Data, err := os.ReadFile(filepath.Join(repositoryMigrations, "028_provider_commercial_proof_manifest.sql"))
	if err != nil {
		t.Fatal(err)
	}
	providerCommercialProofManifestFunctionSQL := ""
	for _, statement := range migrationStatements(string(migration028Data)) {
		if strings.Contains(statement, "CREATE OR REPLACE FUNCTION public.enforce_provider_commercial_proof_manifest()") {
			providerCommercialProofManifestFunctionSQL = statement
			break
		}
	}
	if providerCommercialProofManifestFunctionSQL == "" {
		t.Fatal("028 provider commercial-proof manifest trigger function statement not found")
	}
	digest028 := sha256.Sum256(migration028Data)
	want028SHA := hex.EncodeToString(digest028[:])
	var got028SHA, got028SchemaSHA, got028Revision string
	var applied028At time.Time
	if err := DB.QueryRow(`
		SELECT sha256, schema_sha256, applied_by_commit, applied_at
		FROM nhs_schema_migrations
		WHERE name = '028_provider_commercial_proof_manifest.sql'`).Scan(
		&got028SHA, &got028SchemaSHA, &got028Revision, &applied028At,
	); err != nil {
		t.Fatal(err)
	}
	if got028SHA != want028SHA || len(got028SchemaSHA) != 64 || got028Revision != revision {
		t.Fatalf("028 receipt sha=%q schema_sha_length=%d revision=%q", got028SHA, len(got028SchemaSHA), got028Revision)
	}
	migration029Data, err := os.ReadFile(filepath.Join(repositoryMigrations, "029_provider_settlement_receipts.sql"))
	if err != nil {
		t.Fatal(err)
	}
	digest029 := sha256.Sum256(migration029Data)
	want029SHA := hex.EncodeToString(digest029[:])
	var got029SHA, got029SchemaSHA, got029Revision string
	var applied029At time.Time
	if err := DB.QueryRow(`
		SELECT sha256, schema_sha256, applied_by_commit, applied_at
		FROM nhs_schema_migrations
		WHERE name = '029_provider_settlement_receipts.sql'`).Scan(
		&got029SHA, &got029SchemaSHA, &got029Revision, &applied029At,
	); err != nil {
		t.Fatal(err)
	}
	if got029SHA != want029SHA || len(got029SchemaSHA) != 64 || got029Revision != revision {
		t.Fatalf("029 receipt sha=%q schema_sha_length=%d revision=%q", got029SHA, len(got029SchemaSHA), got029Revision)
	}
	if err := RunMigrations(repositoryMigrations, "development"); err != nil {
		t.Fatalf("exact receipt replay should not require a new release identity: %v", err)
	}
	var replayedAt time.Time
	if err := DB.QueryRow(`SELECT applied_at FROM nhs_schema_migrations WHERE name = '019_provider_exchange.sql'`).Scan(&replayedAt); err != nil {
		t.Fatal(err)
	}
	if !replayedAt.Equal(appliedAt) {
		t.Fatalf("exact replay changed applied_at from %s to %s", appliedAt, replayedAt)
	}
	var replayed020At time.Time
	if err := DB.QueryRow(`SELECT applied_at FROM nhs_schema_migrations WHERE name = '020_action_interest_receipts.sql'`).Scan(&replayed020At); err != nil {
		t.Fatal(err)
	}
	if !replayed020At.Equal(applied020At) {
		t.Fatalf("exact replay changed 020 applied_at from %s to %s", applied020At, replayed020At)
	}
	var replayed021At time.Time
	if err := DB.QueryRow(`SELECT applied_at FROM nhs_schema_migrations WHERE name = '021_provider_capacity_reservations.sql'`).Scan(&replayed021At); err != nil {
		t.Fatal(err)
	}
	if !replayed021At.Equal(applied021At) {
		t.Fatalf("exact replay changed 021 applied_at from %s to %s", applied021At, replayed021At)
	}
	var replayed022At time.Time
	if err := DB.QueryRow(`SELECT applied_at FROM nhs_schema_migrations WHERE name = '022_provider_commercial_proof.sql'`).Scan(&replayed022At); err != nil {
		t.Fatal(err)
	}
	if !replayed022At.Equal(applied022At) {
		t.Fatalf("exact replay changed 022 applied_at from %s to %s", applied022At, replayed022At)
	}
	var replayed023At time.Time
	if err := DB.QueryRow(`SELECT applied_at FROM nhs_schema_migrations WHERE name = '023_provider_controlled_intent_disclosure.sql'`).Scan(&replayed023At); err != nil {
		t.Fatal(err)
	}
	if !replayed023At.Equal(applied023At) {
		t.Fatalf("exact replay changed 023 applied_at from %s to %s", applied023At, replayed023At)
	}
	var replayed024At time.Time
	if err := DB.QueryRow(`SELECT applied_at FROM nhs_schema_migrations WHERE name = '024_provider_pilot_boundary.sql'`).Scan(&replayed024At); err != nil {
		t.Fatal(err)
	}
	if !replayed024At.Equal(applied024At) {
		t.Fatalf("exact replay changed 024 applied_at from %s to %s", applied024At, replayed024At)
	}
	var replayed025At time.Time
	if err := DB.QueryRow(`SELECT applied_at FROM nhs_schema_migrations WHERE name = '025_stage1_fact_integrity.sql'`).Scan(&replayed025At); err != nil {
		t.Fatal(err)
	}
	if !replayed025At.Equal(applied025At) {
		t.Fatalf("exact replay changed 025 applied_at from %s to %s", applied025At, replayed025At)
	}
	var replayed026At time.Time
	if err := DB.QueryRow(`SELECT applied_at FROM nhs_schema_migrations WHERE name = '026_provider_pilot_proof_integrity.sql'`).Scan(&replayed026At); err != nil {
		t.Fatal(err)
	}
	if !replayed026At.Equal(applied026At) {
		t.Fatalf("exact replay changed 026 applied_at from %s to %s", applied026At, replayed026At)
	}
	var replayed027At time.Time
	if err := DB.QueryRow(`SELECT applied_at FROM nhs_schema_migrations WHERE name = '027_provider_pilot_review_evidence.sql'`).Scan(&replayed027At); err != nil {
		t.Fatal(err)
	}
	if !replayed027At.Equal(applied027At) {
		t.Fatalf("exact replay changed 027 applied_at from %s to %s", applied027At, replayed027At)
	}
	var replayed028At time.Time
	if err := DB.QueryRow(`SELECT applied_at FROM nhs_schema_migrations WHERE name = '028_provider_commercial_proof_manifest.sql'`).Scan(&replayed028At); err != nil {
		t.Fatal(err)
	}
	if !replayed028At.Equal(applied028At) {
		t.Fatalf("exact replay changed 028 applied_at from %s to %s", applied028At, replayed028At)
	}
	var replayed029At time.Time
	if err := DB.QueryRow(`SELECT applied_at FROM nhs_schema_migrations WHERE name = '029_provider_settlement_receipts.sql'`).Scan(&replayed029At); err != nil {
		t.Fatal(err)
	}
	if !replayed029At.Equal(applied029At) {
		t.Fatalf("exact replay changed 029 applied_at from %s to %s", applied029At, replayed029At)
	}

	t.Run("checksum mismatch", func(t *testing.T) {
		changed := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name == "019_provider_exchange.sql" {
				data = append(data, []byte("\n-- checksum drift fixture\n")...)
			}
			return data, true
		})
		if err := RunMigrations(changed, revision); err == nil || !strings.Contains(err.Error(), "checksum drift") {
			t.Fatalf("checksum mismatch error = %v", err)
		}
	})

	t.Run("021 checksum mismatch", func(t *testing.T) {
		changed := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name == "021_provider_capacity_reservations.sql" {
				data = append(data, []byte("\n-- capacity checksum drift fixture\n")...)
			}
			return data, true
		})
		if err := RunMigrations(changed, revision); err == nil || !strings.Contains(err.Error(), "checksum drift") {
			t.Fatalf("021 checksum mismatch error = %v", err)
		}
	})

	t.Run("022 checksum mismatch", func(t *testing.T) {
		changed := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name == "022_provider_commercial_proof.sql" {
				data = append(data, []byte("\n-- commercial proof checksum drift fixture\n")...)
			}
			return data, true
		})
		if err := RunMigrations(changed, revision); err == nil || !strings.Contains(err.Error(), "checksum drift") {
			t.Fatalf("022 checksum mismatch error = %v", err)
		}
	})

	t.Run("023 checksum mismatch", func(t *testing.T) {
		changed := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name == "023_provider_controlled_intent_disclosure.sql" {
				data = append(data, []byte("\n-- controlled intent checksum drift fixture\n")...)
			}
			return data, true
		})
		if err := RunMigrations(changed, revision); err == nil || !strings.Contains(err.Error(), "checksum drift") {
			t.Fatalf("023 checksum mismatch error = %v", err)
		}
	})

	t.Run("024 checksum mismatch", func(t *testing.T) {
		changed := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name == "024_provider_pilot_boundary.sql" {
				data = append(data, []byte("\n-- provider pilot boundary checksum drift fixture\n")...)
			}
			return data, true
		})
		if err := RunMigrations(changed, revision); err == nil || !strings.Contains(err.Error(), "checksum drift") {
			t.Fatalf("024 checksum mismatch error = %v", err)
		}
	})

	t.Run("025 checksum mismatch", func(t *testing.T) {
		changed := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name == "025_stage1_fact_integrity.sql" {
				data = append(data, []byte("\n-- Stage 1 fact-integrity checksum drift fixture\n")...)
			}
			return data, true
		})
		if err := RunMigrations(changed, revision); err == nil || !strings.Contains(err.Error(), "checksum drift") {
			t.Fatalf("025 checksum mismatch error = %v", err)
		}
	})

	t.Run("026 checksum mismatch", func(t *testing.T) {
		changed := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name == "026_provider_pilot_proof_integrity.sql" {
				data = append(data, []byte("\n-- provider pilot proof-integrity checksum drift fixture\n")...)
			}
			return data, true
		})
		if err := RunMigrations(changed, revision); err == nil || !strings.Contains(err.Error(), "checksum drift") {
			t.Fatalf("026 checksum mismatch error = %v", err)
		}
	})

	t.Run("027 checksum mismatch", func(t *testing.T) {
		changed := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name == "027_provider_pilot_review_evidence.sql" {
				data = append(data, []byte("\n-- provider pilot review-evidence checksum drift fixture\n")...)
			}
			return data, true
		})
		if err := RunMigrations(changed, revision); err == nil || !strings.Contains(err.Error(), "checksum drift") {
			t.Fatalf("027 checksum mismatch error = %v", err)
		}
	})

	t.Run("028 checksum mismatch", func(t *testing.T) {
		changed := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name == "028_provider_commercial_proof_manifest.sql" {
				data = append(data, []byte("\n-- provider commercial-proof manifest checksum drift fixture\n")...)
			}
			return data, true
		})
		if err := RunMigrations(changed, revision); err == nil || !strings.Contains(err.Error(), "checksum drift") {
			t.Fatalf("028 checksum mismatch error = %v", err)
		}
	})

	t.Run("029 checksum mismatch", func(t *testing.T) {
		changed := copyMigrationFixture(t, repositoryMigrations, func(name string, data []byte) ([]byte, bool) {
			if name == "029_provider_settlement_receipts.sql" {
				data = append(data, []byte("\n-- provider settlement receipt checksum drift fixture\n")...)
			}
			return data, true
		})
		if err := RunMigrations(changed, revision); err == nil || !strings.Contains(err.Error(), "checksum drift") {
			t.Fatalf("029 checksum mismatch error = %v", err)
		}
	})

	t.Run("database ahead", func(t *testing.T) {
		if _, err := DB.Exec(`
			INSERT INTO nhs_schema_migrations (name, sha256, schema_sha256, applied_by_commit)
			VALUES ('030_future.sql', $1, $2, $3)`, strings.Repeat("f", 64), strings.Repeat("e", 64), revision); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "database_ahead_of_binary") {
			t.Fatalf("database-ahead error = %v", err)
		}
		tx, err := DB.Begin()
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if _, err := tx.Exec(`
			ALTER TABLE nhs_schema_migrations
			DISABLE RULE nhs_schema_migrations_no_delete`); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(`DELETE FROM nhs_schema_migrations WHERE name = '029_future.sql'`); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(`
			ALTER TABLE nhs_schema_migrations
			ENABLE RULE nhs_schema_migrations_no_delete`); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("same-name 020 index definition drift", func(t *testing.T) {
		if _, err := DB.Exec(`DROP INDEX idx_action_interest_receipts_domain_created`); err != nil {
			t.Fatal(err)
		}
		if _, err := DB.Exec(`CREATE INDEX idx_action_interest_receipts_domain_created ON action_interest_receipts(expires_at)`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("020 index definition drift error = %v", err)
		}
		if _, err := DB.Exec(`DROP INDEX idx_action_interest_receipts_domain_created`); err != nil {
			t.Fatal(err)
		}
		if _, err := DB.Exec(`CREATE INDEX idx_action_interest_receipts_domain_created ON action_interest_receipts(site_domain_snapshot, created_at DESC)`); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("same-name 021 index definition drift", func(t *testing.T) {
		if _, err := DB.Exec(`DROP INDEX idx_provider_capacity_events_offer_created`); err != nil {
			t.Fatal(err)
		}
		if _, err := DB.Exec(`CREATE INDEX idx_provider_capacity_events_offer_created ON provider_capacity_events(action_ticket_id)`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("021 index definition drift error = %v", err)
		}
		if _, err := DB.Exec(`DROP INDEX idx_provider_capacity_events_offer_created`); err != nil {
			t.Fatal(err)
		}
		if _, err := DB.Exec(`CREATE INDEX idx_provider_capacity_events_offer_created ON provider_capacity_events(provider_offer_id, created_at, id)`); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("same-name 021 rule definition drift", func(t *testing.T) {
		if _, err := DB.Exec(`
			CREATE OR REPLACE RULE provider_capacity_events_no_delete AS
			ON DELETE TO provider_capacity_events DO ALSO NOTHING`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("021 rule definition drift error = %v", err)
		}
		if _, err := DB.Exec(`
			CREATE OR REPLACE RULE provider_capacity_events_no_delete AS
			ON DELETE TO provider_capacity_events DO INSTEAD NOTHING`); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("same-name 021 trigger function body drift", func(t *testing.T) {
		if _, err := DB.Exec(`
			CREATE OR REPLACE FUNCTION public.enforce_action_ticket_capacity_reservation()
			RETURNS TRIGGER
			LANGUAGE plpgsql
			AS $$
			BEGIN
				RETURN NEW;
			END;
			$$`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("021 trigger function body drift error = %v", err)
		}
		if _, err := DB.Exec(capacityReservationFunctionSQL); err != nil {
			t.Fatalf("restore 021 trigger function: %v", err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err != nil {
			t.Fatalf("restored 021 trigger function fingerprint: %v", err)
		}
	})

	t.Run("same-name 022 index definition drift", func(t *testing.T) {
		if _, err := DB.Exec(`DROP INDEX idx_provider_commercial_commitment_company_created`); err != nil {
			t.Fatal(err)
		}
		if _, err := DB.Exec(`
			CREATE INDEX idx_provider_commercial_commitment_company_created
			ON provider_commercial_commitment_events(provider_claim_id)`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("022 index definition drift error = %v", err)
		}
		if _, err := DB.Exec(`DROP INDEX idx_provider_commercial_commitment_company_created`); err != nil {
			t.Fatal(err)
		}
		if _, err := DB.Exec(`
			CREATE INDEX idx_provider_commercial_commitment_company_created
			ON provider_commercial_commitment_events(provider_pilot_company_id, created_at DESC, id DESC)`); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("same-name 022 rule definition drift", func(t *testing.T) {
		if _, err := DB.Exec(`
			CREATE OR REPLACE RULE provider_commercial_commitment_no_delete AS
			ON DELETE TO provider_commercial_commitment_events DO ALSO NOTHING`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("022 rule definition drift error = %v", err)
		}
		if _, err := DB.Exec(`
			CREATE OR REPLACE RULE provider_commercial_commitment_no_delete AS
			ON DELETE TO provider_commercial_commitment_events DO INSTEAD NOTHING`); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("same-name 022 trigger function body drift", func(t *testing.T) {
		if _, err := DB.Exec(`
			CREATE OR REPLACE FUNCTION public.enforce_provider_commercial_commitment_event()
			RETURNS TRIGGER
			LANGUAGE plpgsql
			AS $$
			BEGIN
				RETURN NEW;
			END;
			$$`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("022 trigger function body drift error = %v", err)
		}
		if _, err := DB.Exec(commercialCommitmentFunctionSQL); err != nil {
			t.Fatalf("restore 022 trigger function: %v", err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err != nil {
			t.Fatalf("restored 022 trigger function fingerprint: %v", err)
		}
	})

	t.Run("same-name 022 inherited-table trigger function body drift", func(t *testing.T) {
		if _, err := DB.Exec(`
			CREATE OR REPLACE FUNCTION public.enforce_provider_offer_commercial_immutability()
			RETURNS TRIGGER
			LANGUAGE plpgsql
			AS $$
			BEGIN
				RETURN NEW;
			END;
			$$`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("022 inherited-table trigger function body drift error = %v", err)
		}
		if _, err := DB.Exec(offerCommercialImmutabilityFunctionSQL); err != nil {
			t.Fatalf("restore 022 provider-offer commercial immutability trigger function: %v", err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err != nil {
			t.Fatalf("restored 022 inherited-table trigger function fingerprint: %v", err)
		}
	})

	t.Run("same-name 022 altered-column definition drift", func(t *testing.T) {
		if _, err := DB.Exec(`
			ALTER TABLE provider_offers
			ALTER COLUMN commercial_terms_contract_version DROP DEFAULT`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("022 altered-column definition drift error = %v", err)
		}
		if _, err := DB.Exec(`
			ALTER TABLE provider_offers
			ALTER COLUMN commercial_terms_contract_version SET DEFAULT ''`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err != nil {
			t.Fatalf("restored 022 altered-column fingerprint: %v", err)
		}
	})

	t.Run("same-name 023 inherited-table trigger function body drift", func(t *testing.T) {
		if _, err := DB.Exec(`
			CREATE OR REPLACE FUNCTION public.enforce_action_ticket_controlled_intent_immutability()
			RETURNS TRIGGER
			LANGUAGE plpgsql
			AS $$
			BEGIN
				RETURN NEW;
			END;
			$$`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("023 inherited-table trigger function body drift error = %v", err)
		}
		if _, err := DB.Exec(controlledIntentImmutabilityFunctionSQL); err != nil {
			t.Fatalf("restore 023 controlled-intent immutability trigger function: %v", err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err != nil {
			t.Fatalf("restored 023 inherited-table trigger function fingerprint: %v", err)
		}
	})

	t.Run("same-name 023 altered-column definition drift", func(t *testing.T) {
		if _, err := DB.Exec(`
			ALTER TABLE provider_action_handoff_receipts
			ALTER COLUMN principal_controlled_intent_disclosure_consent DROP DEFAULT`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("023 altered-column definition drift error = %v", err)
		}
		if _, err := DB.Exec(`
			ALTER TABLE provider_action_handoff_receipts
			ALTER COLUMN principal_controlled_intent_disclosure_consent SET DEFAULT false`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err != nil {
			t.Fatalf("restored 023 altered-column fingerprint: %v", err)
		}
	})

	t.Run("same-name 027 altered-column definition drift", func(t *testing.T) {
		if _, err := DB.Exec(`
			ALTER TABLE provider_pilot_review_events
			ALTER COLUMN reviewed_at DROP DEFAULT`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("027 altered-column definition drift error = %v", err)
		}
		if _, err := DB.Exec(`
			ALTER TABLE provider_pilot_review_events
			ALTER COLUMN reviewed_at SET DEFAULT statement_timestamp()`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err != nil {
			t.Fatalf("restored 027 altered-column fingerprint: %v", err)
		}
	})

	t.Run("same-name 027 index definition drift", func(t *testing.T) {
		if _, err := DB.Exec(`DROP INDEX idx_provider_pilot_reviews_pilot_type`); err != nil {
			t.Fatal(err)
		}
		if _, err := DB.Exec(`
			CREATE INDEX idx_provider_pilot_reviews_pilot_type
			ON provider_pilot_review_events(subject_id)`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("027 index definition drift error = %v", err)
		}
		if _, err := DB.Exec(`DROP INDEX idx_provider_pilot_reviews_pilot_type`); err != nil {
			t.Fatal(err)
		}
		if _, err := DB.Exec(`
			CREATE INDEX idx_provider_pilot_reviews_pilot_type
			ON provider_pilot_review_events(
				provider_pilot_epoch_id, review_type, reviewed_at, id
			)`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err != nil {
			t.Fatalf("restored 027 index fingerprint: %v", err)
		}
	})

	t.Run("same-name 027 rule definition drift", func(t *testing.T) {
		if _, err := DB.Exec(`
			CREATE OR REPLACE RULE provider_pilot_review_events_no_delete AS
			ON DELETE TO provider_pilot_review_events DO ALSO NOTHING`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("027 rule definition drift error = %v", err)
		}
		if _, err := DB.Exec(`
			CREATE OR REPLACE RULE provider_pilot_review_events_no_delete AS
			ON DELETE TO provider_pilot_review_events DO INSTEAD NOTHING`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err != nil {
			t.Fatalf("restored 027 rule fingerprint: %v", err)
		}
	})

	t.Run("same-name 027 trigger definition drift", func(t *testing.T) {
		if _, err := DB.Exec(`
			DROP TRIGGER provider_pilot_review_event_enforced
			ON provider_pilot_review_events`); err != nil {
			t.Fatal(err)
		}
		if _, err := DB.Exec(`
			CREATE TRIGGER provider_pilot_review_event_enforced
			AFTER INSERT ON provider_pilot_review_events
			FOR EACH ROW
			EXECUTE FUNCTION public.enforce_provider_pilot_review_event()`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("027 trigger definition drift error = %v", err)
		}
		if _, err := DB.Exec(`
			DROP TRIGGER provider_pilot_review_event_enforced
			ON provider_pilot_review_events`); err != nil {
			t.Fatal(err)
		}
		if _, err := DB.Exec(`
			CREATE TRIGGER provider_pilot_review_event_enforced
			BEFORE INSERT ON provider_pilot_review_events
			FOR EACH ROW
			EXECUTE FUNCTION public.enforce_provider_pilot_review_event()`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err != nil {
			t.Fatalf("restored 027 trigger fingerprint: %v", err)
		}
	})

	t.Run("same-name 027 trigger function body drift", func(t *testing.T) {
		if _, err := DB.Exec(`
			CREATE OR REPLACE FUNCTION public.enforce_provider_pilot_review_event()
			RETURNS TRIGGER
			LANGUAGE plpgsql
			AS $$
			BEGIN
				RETURN NEW;
			END;
			$$`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("027 trigger function body drift error = %v", err)
		}
		if _, err := DB.Exec(providerPilotReviewEnforcementFunctionSQL); err != nil {
			t.Fatalf("restore 027 review enforcement trigger function: %v", err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err != nil {
			t.Fatalf("restored 027 trigger function fingerprint: %v", err)
		}
	})

	t.Run("same-name 027 standalone snapshot function body drift", func(t *testing.T) {
		if _, err := DB.Exec(`
			CREATE OR REPLACE FUNCTION public.provider_pilot_review_snapshot_sha256(
				requested_pilot_id UUID,
				requested_review_type TEXT,
				requested_subject_id UUID
			)
			RETURNS TEXT
			LANGUAGE sql
			STABLE
			STRICT
			AS $$ SELECT repeat('0', 64) $$`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("027 standalone snapshot function body drift error = %v", err)
		}
		if _, err := DB.Exec(providerPilotReviewSnapshotFunctionSQL); err != nil {
			t.Fatalf("restore 027 standalone snapshot function: %v", err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err != nil {
			t.Fatalf("restored 027 standalone function fingerprint: %v", err)
		}
	})

	t.Run("same-name 028 altered-column definition drift", func(t *testing.T) {
		if _, err := DB.Exec(`
			ALTER TABLE provider_commercial_proof_manifests
			ALTER COLUMN owner_reference DROP NOT NULL`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("028 altered-column definition drift error = %v", err)
		}
		if _, err := DB.Exec(`
			ALTER TABLE provider_commercial_proof_manifests
			ALTER COLUMN owner_reference SET NOT NULL`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err != nil {
			t.Fatalf("restored 028 altered-column fingerprint: %v", err)
		}
	})

	t.Run("same-name 028 index definition drift", func(t *testing.T) {
		if _, err := DB.Exec(`DROP INDEX idx_provider_proof_manifests_issued`); err != nil {
			t.Fatal(err)
		}
		if _, err := DB.Exec(`
			CREATE INDEX idx_provider_proof_manifests_issued
			ON provider_commercial_proof_manifests(provider_pilot_epoch_id)`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("028 index definition drift error = %v", err)
		}
		if _, err := DB.Exec(`DROP INDEX idx_provider_proof_manifests_issued`); err != nil {
			t.Fatal(err)
		}
		if _, err := DB.Exec(`
			CREATE INDEX idx_provider_proof_manifests_issued
			ON provider_commercial_proof_manifests(issued_at, id)`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err != nil {
			t.Fatalf("restored 028 index fingerprint: %v", err)
		}
	})

	t.Run("same-name 028 rule definition drift", func(t *testing.T) {
		if _, err := DB.Exec(`
			CREATE OR REPLACE RULE provider_commercial_proof_manifests_no_delete AS
			ON DELETE TO provider_commercial_proof_manifests DO ALSO NOTHING`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("028 rule definition drift error = %v", err)
		}
		if _, err := DB.Exec(`
			CREATE OR REPLACE RULE provider_commercial_proof_manifests_no_delete AS
			ON DELETE TO provider_commercial_proof_manifests DO INSTEAD NOTHING`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err != nil {
			t.Fatalf("restored 028 rule fingerprint: %v", err)
		}
	})

	t.Run("same-name 028 trigger definition drift", func(t *testing.T) {
		if _, err := DB.Exec(`
			DROP TRIGGER provider_commercial_proof_manifest_enforced
			ON provider_commercial_proof_manifests`); err != nil {
			t.Fatal(err)
		}
		if _, err := DB.Exec(`
			CREATE TRIGGER provider_commercial_proof_manifest_enforced
			AFTER INSERT ON provider_commercial_proof_manifests
			FOR EACH ROW
			EXECUTE FUNCTION public.enforce_provider_commercial_proof_manifest()`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("028 trigger definition drift error = %v", err)
		}
		if _, err := DB.Exec(`
			DROP TRIGGER provider_commercial_proof_manifest_enforced
			ON provider_commercial_proof_manifests`); err != nil {
			t.Fatal(err)
		}
		if _, err := DB.Exec(`
			CREATE TRIGGER provider_commercial_proof_manifest_enforced
			BEFORE INSERT ON provider_commercial_proof_manifests
			FOR EACH ROW
			EXECUTE FUNCTION public.enforce_provider_commercial_proof_manifest()`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err != nil {
			t.Fatalf("restored 028 trigger fingerprint: %v", err)
		}
	})

	t.Run("same-name 028 trigger function body drift", func(t *testing.T) {
		if _, err := DB.Exec(`
			CREATE OR REPLACE FUNCTION public.enforce_provider_commercial_proof_manifest()
			RETURNS TRIGGER
			LANGUAGE plpgsql
			AS $$
			BEGIN
				RETURN NEW;
			END;
			$$`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("028 trigger function body drift error = %v", err)
		}
		if _, err := DB.Exec(providerCommercialProofManifestFunctionSQL); err != nil {
			t.Fatalf("restore 028 commercial-proof manifest trigger function: %v", err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err != nil {
			t.Fatalf("restored 028 trigger function fingerprint: %v", err)
		}
	})

	t.Run("same-name rule definition drift", func(t *testing.T) {
		if _, err := DB.Exec(`
			CREATE OR REPLACE RULE provider_budget_ledger_no_delete AS
			ON DELETE TO provider_budget_ledger DO ALSO NOTHING`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("rule definition drift error = %v", err)
		}
		if _, err := DB.Exec(`
			CREATE OR REPLACE RULE provider_budget_ledger_no_delete AS
			ON DELETE TO provider_budget_ledger DO INSTEAD NOTHING`); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("same-name index definition drift", func(t *testing.T) {
		if _, err := DB.Exec(`DROP INDEX idx_provider_offers_public_active`); err != nil {
			t.Fatal(err)
		}
		if _, err := DB.Exec(`CREATE INDEX idx_provider_offers_public_active ON provider_offers(created_at)`); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
			t.Fatalf("index definition drift error = %v", err)
		}
	})
}

type providerCapacityMigrationFixture struct {
	claimID string
	offerID string
	siteID  string
	domain  string
	seed    string
}

func seedProviderCapacityMigrationFixture(t *testing.T) providerCapacityMigrationFixture {
	t.Helper()
	seed := fmt.Sprintf("%d", time.Now().UnixNano())
	var accountID int64
	if err := DB.QueryRow(`
		INSERT INTO accounts (email)
		VALUES ($1)
		RETURNING id`, "capacity-migration-"+seed+"@example.test").Scan(&accountID); err != nil {
		t.Fatalf("seed capacity account: %v", err)
	}
	fixture := providerCapacityMigrationFixture{seed: seed}
	domain := "capacity-migration-" + seed + ".example.test"
	if err := DB.QueryRow(`
		INSERT INTO sites (domain, url, name)
		VALUES ($1, $2, 'Capacity migration fixture')
		RETURNING id`, domain, "https://"+domain).Scan(&fixture.siteID); err != nil {
		t.Fatalf("seed capacity site: %v", err)
	}
	fixture.domain = domain
	if err := DB.QueryRow(`
		INSERT INTO provider_claims (
			account_id, site_id, domain_snapshot, verification_record_name,
			verification_token_hash, challenge_expires_at
		) VALUES ($1, $2, $3, $4, $5, NOW() + INTERVAL '1 day')
		RETURNING id`,
		accountID,
		fixture.siteID,
		domain,
		"_nhs-verify."+domain,
		postgresMigrationTestHash(seed+"-claim-token"),
	).Scan(&fixture.claimID); err != nil {
		t.Fatalf("seed capacity provider claim: %v", err)
	}
	if err := DB.QueryRow(`
		INSERT INTO provider_offers (
			provider_claim_id, offer_name, offer_summary, action_type,
			action_url, charge_event, bounty_cents, currency,
			principal_price_mode, principal_price_cents, principal_currency,
			billing_mode
		) VALUES (
			$1, 'Legacy capacity offer', 'Migration compatibility fixture',
			'signup', 'https://capacity-migration.example.test/start',
			'accepted', 125, 'usd', 'free', 0, 'usd', 'prepaid'
		)
		RETURNING id`, fixture.claimID).Scan(&fixture.offerID); err != nil {
		t.Fatalf("seed capacity provider offer: %v", err)
	}
	return fixture
}

func seedProviderPilotStage1MigrationEvidence(
	t *testing.T,
	fixture providerCapacityMigrationFixture,
) time.Time {
	t.Helper()
	var stage1StartedAt time.Time
	if err := DB.QueryRow(`
		UPDATE nhs_schema_migrations
		SET applied_at=statement_timestamp()-INTERVAL '15 days'
		WHERE name='020_action_interest_receipts.sql'
		RETURNING applied_at`).Scan(&stage1StartedAt); err != nil {
		t.Fatalf("backdate disposable Stage 1 migration receipt: %v", err)
	}
	prefix := "nhs_stage1_" + fixture.seed + "_"
	stage1SitePrefix := "stage1-migration-" + fixture.seed + "-"
	if _, err := DB.Exec(`
		INSERT INTO search_receipts (
			public_id, surface, explicit_category, demand_topics,
			result_count, page_number, page_size, is_synthetic, created_at
		)
		SELECT $1 || series.n::text, 'rest', 'developer',
		       ARRAY['developer-tools']::text[], 1, 1, 10, false,
		       $2::timestamptz +
		       (series.n-1) * INTERVAL '15 days' / 99
		FROM generate_series(1,100) AS series(n)`, prefix, stage1StartedAt); err != nil {
		t.Fatalf("seed Stage 1 search receipts: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO sites (domain, url, name)
		SELECT $1 || series.n::text || '.example.test',
		       'https://' || $1 || series.n::text || '.example.test',
		       'Stage 1 migration breadth fixture'
		FROM generate_series(1,9) AS series(n)`, stage1SitePrefix); err != nil {
		t.Fatalf("seed Stage 1 breadth sites: %v", err)
	}
	if _, err := DB.Exec(`
		WITH numbered_receipts AS (
			SELECT receipt.id, receipt.created_at,
			       ROW_NUMBER() OVER (ORDER BY receipt.created_at, receipt.id) AS ordinal
			FROM search_receipts receipt
			WHERE LEFT(receipt.public_id, LENGTH($1))=$1
		), eligible_sites AS (
			SELECT site.id, site.domain,
			       ROW_NUMBER() OVER (
			           ORDER BY CASE WHEN site.id=$2::uuid THEN 0 ELSE 1 END,
			                    site.domain
			       ) AS ordinal
			FROM sites site
			WHERE site.id=$2::uuid OR LEFT(site.domain, LENGTH($3))=$3
		)
		INSERT INTO organic_results_returned (
			search_receipt_id, site_id, site_domain_snapshot,
			organic_position, score_snapshot, returned_at
		)
		SELECT receipt.id, site.id, site.domain, 1, 90, receipt.created_at
		FROM numbered_receipts receipt
		JOIN eligible_sites site
		  ON site.ordinal=((receipt.ordinal-1) % 10)+1`,
		prefix, fixture.siteID, stage1SitePrefix); err != nil {
		t.Fatalf("seed Stage 1 organic returns: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO result_selections (
			search_receipt_id, site_id, site_domain_snapshot, surface, selected_at
		)
		SELECT receipt.id, returned.site_id, returned.site_domain_snapshot,
		       'rest', receipt.created_at
		FROM search_receipts receipt
		JOIN organic_results_returned returned
		  ON returned.search_receipt_id=receipt.id
		WHERE LEFT(receipt.public_id, LENGTH($1))=$1
		ORDER BY receipt.created_at, receipt.id
		LIMIT 20`, prefix); err != nil {
		t.Fatalf("seed Stage 1 result selections: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO action_interest_receipts (
			public_id, search_receipt_id, source_is_synthetic,
			site_domain_snapshot, action_type, surface,
			caller_attests_principal_interest, confirmation_version,
			created_at, expires_at
		)
		SELECT 'nhs_air_' || SUBSTRING(MD5(receipt.public_id),1,16),
		       receipt.id, false, returned.site_domain_snapshot,
		       'trial', 'rest', true,
		       'nhs-action-interest-v1', receipt.created_at,
		       receipt.created_at + INTERVAL '30 days'
		FROM search_receipts receipt
		JOIN organic_results_returned returned
		  ON returned.search_receipt_id=receipt.id
		WHERE LEFT(receipt.public_id, LENGTH($1))=$1
		ORDER BY receipt.created_at, receipt.id
		LIMIT 10`, prefix); err != nil {
		t.Fatalf("seed Stage 1 action-interest receipts: %v", err)
	}
	return stage1StartedAt
}

func postgresMigrationTestHash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func postgresPilotEventSnapshotHash(eventType, pilotID, claimID string) string {
	return postgresMigrationTestHash(strings.Join([]string{
		"nhs-provider-pilot-event-snapshot-v1", eventType,
		strings.ToLower(pilotID), strings.ToLower(claimID),
	}, "\n"))
}

func assertMigrationPostgresConstraintCode(t *testing.T, err error, code, constraint string) {
	t.Helper()
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || string(pqErr.Code) != code || pqErr.Constraint != constraint {
		t.Fatalf("PostgreSQL error = %#v, want code=%s constraint=%s", err, code, constraint)
	}
}

func waitForPendingPostgresRelationLock(t *testing.T, relation, mode string) {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var waiting bool
		if err := DB.QueryRow(`
			SELECT EXISTS (
				SELECT 1
				FROM pg_catalog.pg_locks locks
				JOIN pg_catalog.pg_class relation ON relation.oid=locks.relation
				JOIN pg_catalog.pg_namespace namespace ON namespace.oid=relation.relnamespace
				WHERE namespace.nspname='public'
				  AND relation.relname=$1
				  AND locks.mode=$2
				  AND NOT locks.granted
			)`, relation, mode).Scan(&waiting); err != nil {
			t.Fatalf("inspect pending PostgreSQL lock: %v", err)
		}
		if waiting {
			return
		}
		select {
		case <-deadline.C:
			t.Fatalf("no pending %s on public.%s", mode, relation)
		case <-ticker.C:
		}
	}
}

func copyMigrationFixture(t *testing.T, source string, transform func(string, []byte) ([]byte, bool)) string {
	t.Helper()
	destination := t.TempDir()
	entries, err := os.ReadDir(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(source, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		changed, include := transform(entry.Name(), data)
		if !include {
			continue
		}
		if changed == nil {
			changed = data
		}
		if err := os.WriteFile(filepath.Join(destination, entry.Name()), changed, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return destination
}

// These two helpers exist only in the disposable PostgreSQL test binary. They
// temporarily disable one append-only rule inside the same transaction as the
// fixture mutation, restore the rule before commit, and expose no production
// path for rewriting protected evidence.
func setPostgresProtectedMigrationAppliedAtFixture(
	t *testing.T,
	name string,
	appliedAt time.Time,
) {
	t.Helper()
	tx, err := DB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`
		ALTER TABLE nhs_schema_migrations
		DISABLE RULE nhs_schema_migrations_no_update`); err != nil {
		t.Fatal(err)
	}
	result, err := tx.Exec(`
		UPDATE nhs_schema_migrations SET applied_at=$1 WHERE name=$2`,
		appliedAt, name)
	if err != nil {
		t.Fatal(err)
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		t.Fatalf("protected migration fixture affected=%d err=%v", affected, err)
	}
	if _, err := tx.Exec(`
		ALTER TABLE nhs_schema_migrations
		ENABLE RULE nhs_schema_migrations_no_update`); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func deletePostgresPilotEventFixture(t *testing.T, eventID string) {
	t.Helper()
	tx, err := DB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`
		ALTER TABLE provider_pilot_epoch_events
		DISABLE RULE provider_pilot_epoch_events_no_delete`); err != nil {
		t.Fatal(err)
	}
	result, err := tx.Exec(`DELETE FROM provider_pilot_epoch_events WHERE id=$1::uuid`, eventID)
	if err != nil {
		t.Fatal(err)
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		t.Fatalf("pilot event fixture delete affected=%d err=%v", affected, err)
	}
	if _, err := tx.Exec(`
		ALTER TABLE provider_pilot_epoch_events
		ENABLE RULE provider_pilot_epoch_events_no_delete`); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func assertPostgresRelationAbsent(t *testing.T, name string) {
	t.Helper()
	var exists bool
	if err := DB.QueryRow(`SELECT to_regclass('public.' || $1) IS NOT NULL`, name).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatalf("relation %s survived protected migration rollback", name)
	}
}

func assertPostgresColumnAbsent(t *testing.T, table, column string) {
	t.Helper()
	var exists bool
	if err := DB.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema='public' AND table_name=$1 AND column_name=$2
		)`, table, column).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatalf("column %s.%s survived protected migration rollback", table, column)
	}
}

func assertPostgresFunctionAbsent(t *testing.T, name string) {
	t.Helper()
	var exists bool
	if err := DB.QueryRow(`
		SELECT to_regprocedure('public.' || $1 || '()') IS NOT NULL`, name).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatalf("function %s survived protected migration rollback", name)
	}
}
