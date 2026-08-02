package database

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestProtectedMigrationLedgerPostgres is opt-in because it owns and mutates
// the isolated database named by NHS_MIGRATION_TEST_POSTGRES_DSN.
func TestProtectedMigrationLedgerPostgres(t *testing.T) {
	dsn := os.Getenv("NHS_MIGRATION_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set NHS_MIGRATION_TEST_POSTGRES_DSN to an isolated disposable PostgreSQL database")
	}
	t.Setenv("DATABASE_URL", dsn)
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
			migrationDone <- RunMigrations(repositoryMigrations, revision)
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

	t.Run("database ahead", func(t *testing.T) {
		if _, err := DB.Exec(`
			INSERT INTO nhs_schema_migrations (name, sha256, schema_sha256, applied_by_commit)
			VALUES ('022_future.sql', $1, $2, $3)`, strings.Repeat("f", 64), strings.Repeat("e", 64), revision); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "database_ahead_of_binary") {
			t.Fatalf("database-ahead error = %v", err)
		}
		if _, err := DB.Exec(`DELETE FROM nhs_schema_migrations WHERE name = '022_future.sql'`); err != nil {
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
	var siteID string
	domain := "capacity-migration-" + seed + ".example.test"
	if err := DB.QueryRow(`
		INSERT INTO sites (domain, url, name)
		VALUES ($1, $2, 'Capacity migration fixture')
		RETURNING id`, domain, "https://"+domain).Scan(&siteID); err != nil {
		t.Fatalf("seed capacity site: %v", err)
	}
	fixture := providerCapacityMigrationFixture{seed: seed}
	if err := DB.QueryRow(`
		INSERT INTO provider_claims (
			account_id, site_id, domain_snapshot, verification_record_name,
			verification_token_hash, challenge_expires_at
		) VALUES ($1, $2, $3, $4, $5, NOW() + INTERVAL '1 day')
		RETURNING id`,
		accountID,
		siteID,
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

func postgresMigrationTestHash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
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
