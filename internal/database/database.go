package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

const (
	protectedMigrationStart = "019_"
	migrationLockClass      = 0x4e4853 // "NHS"
	migrationLockID         = 1
)

const migrationReceiptTableSQL = `
CREATE TABLE IF NOT EXISTS public.nhs_schema_migrations (
    name              TEXT PRIMARY KEY,
    sha256            TEXT NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    schema_sha256     TEXT NOT NULL CHECK (schema_sha256 ~ '^[0-9a-f]{64}$'),
    applied_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    applied_by_commit TEXT NOT NULL CHECK (applied_by_commit ~ '^[0-9a-f]{40}$')
)`

type migrationFile struct {
	name   string
	data   []byte
	sha256 string
}

type migrationRelation struct {
	name    string
	relkind string
	parent  string
}

type migrationRule struct {
	relation string
	name     string
}

type protectedMigrationSpec struct {
	relations              []migrationRelation
	rules                  []migrationRule
	allObjectsAreFootprint bool
	footprintRelations     map[string]bool
	footprintRules         map[string]bool
}

// Every protected migration must declare the PostgreSQL objects that prove its
// complete application. A new protected migration without a contract fails
// before any older migration is replayed. This prevents an unrecorded partial
// schema from being adopted merely because its DDL uses IF NOT EXISTS.
var protectedMigrationSpecs = map[string]protectedMigrationSpec{
	"019_provider_exchange.sql": {
		allObjectsAreFootprint: true,
		relations: []migrationRelation{
			{name: "provider_claims", relkind: "r"},
			{name: "idx_provider_claims_one_verified_site", relkind: "i", parent: "provider_claims"},
			{name: "idx_provider_claims_one_pending_account_site", relkind: "i", parent: "provider_claims"},
			{name: "idx_provider_claims_account_status", relkind: "i", parent: "provider_claims"},
			{name: "idx_provider_claims_pending_expiry", relkind: "i", parent: "provider_claims"},
			{name: "idx_provider_claims_verification_due", relkind: "i", parent: "provider_claims"},
			{name: "idx_provider_claims_verification_freshness", relkind: "i", parent: "provider_claims"},
			{name: "provider_api_keys", relkind: "r"},
			{name: "idx_provider_api_keys_claim_status", relkind: "i", parent: "provider_api_keys"},
			{name: "idx_provider_api_keys_one_active_claim", relkind: "i", parent: "provider_api_keys"},
			{name: "provider_offers", relkind: "r"},
			{name: "idx_provider_offers_claim_status", relkind: "i", parent: "provider_offers"},
			{name: "idx_provider_offers_public_active", relkind: "i", parent: "provider_offers"},
			{name: "provider_offers_returned", relkind: "r"},
			{name: "idx_provider_offers_returned_offer", relkind: "i", parent: "provider_offers_returned"},
			{name: "action_tickets", relkind: "r"},
			{name: "idx_action_tickets_offer_status", relkind: "i", parent: "action_tickets"},
			{name: "idx_action_tickets_receipt", relkind: "i", parent: "action_tickets"},
			{name: "idx_action_tickets_expiry", relkind: "i", parent: "action_tickets"},
			{name: "provider_budget_ledger", relkind: "r"},
			{name: "idx_provider_budget_offer_created", relkind: "i", parent: "provider_budget_ledger"},
			{name: "idx_provider_budget_claim_created", relkind: "i", parent: "provider_budget_ledger"},
			{name: "idx_provider_budget_one_charge_per_ticket", relkind: "i", parent: "provider_budget_ledger"},
			{name: "idx_provider_budget_one_credit_per_ticket", relkind: "i", parent: "provider_budget_ledger"},
			{name: "idx_provider_budget_unique_funding_reference", relkind: "i", parent: "provider_budget_ledger"},
			{name: "outcome_receipts", relkind: "r"},
			{name: "idx_outcome_receipts_ticket_created", relkind: "i", parent: "outcome_receipts"},
			{name: "idx_outcome_receipts_claim_created", relkind: "i", parent: "outcome_receipts"},
			{name: "provider_admin_audit_events", relkind: "r"},
			{name: "idx_provider_admin_audit_offer_created", relkind: "i", parent: "provider_admin_audit_events"},
		},
		rules: []migrationRule{
			{relation: "provider_offers_returned", name: "provider_offers_returned_no_update"},
			{relation: "provider_budget_ledger", name: "provider_budget_ledger_no_update"},
			{relation: "provider_budget_ledger", name: "provider_budget_ledger_no_delete"},
			{relation: "outcome_receipts", name: "outcome_receipts_no_update"},
			{relation: "outcome_receipts", name: "outcome_receipts_no_delete"},
			{relation: "provider_admin_audit_events", name: "provider_admin_audit_no_update"},
			{relation: "provider_admin_audit_events", name: "provider_admin_audit_no_delete"},
			{relation: "search_receipts", name: "redact_action_ticket_intent_on_receipt_delete"},
		},
	},
}

func Connect() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL not set")
	}

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}

	if err := DB.Ping(); err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	DB.SetMaxOpenConns(10)
	DB.SetMaxIdleConns(5)
	return nil
}

func RunMigrations(dir, releaseRevision string) error {
	if DB == nil {
		return fmt.Errorf("database is not connected")
	}
	migrations, err := loadMigrations(dir)
	if err != nil {
		return err
	}

	ctx := context.Background()
	conn, err := DB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("reserve migration connection: %w", err)
	}
	defer conn.Close()
	lockCtx, cancelLock := context.WithTimeout(ctx, 30*time.Second)
	defer cancelLock()
	if _, err := conn.ExecContext(lockCtx, `SELECT pg_advisory_lock($1, $2)`, migrationLockClass, migrationLockID); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer func() {
		unlockCtx, cancelUnlock := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelUnlock()
		if _, err := conn.ExecContext(unlockCtx, `SELECT pg_advisory_unlock($1, $2)`, migrationLockClass, migrationLockID); err != nil {
			log.Printf("release migration lock: %v", err)
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
		}
	}()

	// Check every protected receipt and schema footprint before replaying even
	// the legacy migrations. Ambiguity must stop all schema mutation.
	latestReceipt, err := preflightProtectedMigrations(ctx, conn, migrations, releaseRevision)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if isProtectedMigration(migration.name) {
			spec := protectedMigrationSpecs[migration.name]
			applied, err := applyProtectedMigration(ctx, conn, migration, spec, releaseRevision, migration.name == latestReceipt)
			if err != nil {
				return err
			}
			if applied {
				log.Printf("protected migration applied with exact receipt: %s", migration.name)
			} else {
				log.Printf("protected migration receipt verified: %s", migration.name)
			}
			continue
		}
		if err := applyLegacyMigration(ctx, conn, migration); err != nil {
			return err
		}
	}
	return nil
}

func loadMigrations(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	migrations := make([]migrationFile, 0, len(files))
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f, err)
		}
		digest := sha256.Sum256(data)
		migrations = append(migrations, migrationFile{
			name: f, data: data, sha256: hex.EncodeToString(digest[:]),
		})
	}
	return migrations, nil
}

func isProtectedMigration(name string) bool {
	return name >= protectedMigrationStart
}

func preflightProtectedMigrations(ctx context.Context, conn *sql.Conn, migrations []migrationFile, releaseRevision string) (string, error) {
	known := make(map[string]bool, len(migrations))
	for _, migration := range migrations {
		if isProtectedMigration(migration.name) {
			known[migration.name] = true
		}
	}
	if err := validateProtectedMigrationSpecChain(migrations); err != nil {
		return "", err
	}

	var ledgerExists bool
	if err := conn.QueryRowContext(ctx, `SELECT to_regclass('public.nhs_schema_migrations') IS NOT NULL`).Scan(&ledgerExists); err != nil {
		return "", fmt.Errorf("inspect migration receipt table: %w", err)
	}
	latestReceipt := ""
	if ledgerExists {
		rows, err := conn.QueryContext(ctx, `
			SELECT name
			FROM public.nhs_schema_migrations
			WHERE name >= $1
			ORDER BY name`, protectedMigrationStart)
		if err != nil {
			return "", fmt.Errorf("read protected migration receipts: %w", err)
		}
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				_ = rows.Close()
				return "", fmt.Errorf("scan protected migration receipt: %w", err)
			}
			if !known[name] {
				_ = rows.Close()
				return "", fmt.Errorf("database_ahead_of_binary: protected migration receipt %s is absent from this release", name)
			}
			latestReceipt = name
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return "", fmt.Errorf("read protected migration receipts: %w", err)
		}
		if err := rows.Close(); err != nil {
			return "", fmt.Errorf("close protected migration receipts: %w", err)
		}
	}

	for _, migration := range migrations {
		if !isProtectedMigration(migration.name) {
			continue
		}
		spec := protectedMigrationSpecs[migration.name]
		state, err := inspectProtectedMigration(ctx, conn, migration.name, spec)
		if err != nil {
			return "", fmt.Errorf("preflight protected migration %s: %w", migration.name, err)
		}
		if err := validateProtectedMigrationState(migration, state, migration.name == latestReceipt); err != nil {
			return "", err
		}
		if !state.receiptExists {
			if _, err := protectedMigrationRevision(releaseRevision); err != nil {
				return "", fmt.Errorf("protected migration %s: %w", migration.name, err)
			}
		}
	}
	return latestReceipt, nil
}

// Each later protected specification is cumulative so the latest receipt can
// fingerprint the complete protected schema after reviewed ALTER migrations.
// Earlier receipts retain immutable file checksums but no longer own the final
// catalog shape after a successor migration has been applied.
func validateProtectedMigrationSpecChain(migrations []migrationFile) error {
	priorRelations := map[string]migrationRelation{}
	priorRules := map[string]migrationRule{}
	first := true
	for _, migration := range migrations {
		if !isProtectedMigration(migration.name) {
			continue
		}
		spec, ok := protectedMigrationSpecs[migration.name]
		if !ok {
			return fmt.Errorf("protected migration %s has no schema-footprint contract", migration.name)
		}
		if !first && spec.allObjectsAreFootprint {
			return fmt.Errorf("protected migration %s cannot treat cumulative prior objects as a new footprint", migration.name)
		}
		currentRelations := map[string]migrationRelation{}
		for _, relation := range spec.relations {
			if _, duplicate := currentRelations[relation.name]; duplicate {
				return fmt.Errorf("protected migration %s repeats relation %s", migration.name, relation.name)
			}
			currentRelations[relation.name] = relation
		}
		currentRules := map[string]migrationRule{}
		for _, rule := range spec.rules {
			if _, duplicate := currentRules[rule.name]; duplicate {
				return fmt.Errorf("protected migration %s repeats rule %s", migration.name, rule.name)
			}
			currentRules[rule.name] = rule
		}
		for name, prior := range priorRelations {
			current, ok := currentRelations[name]
			if !ok || current.relkind != prior.relkind || current.parent != prior.parent {
				return fmt.Errorf("protected migration %s does not carry forward relation contract %s", migration.name, name)
			}
		}
		for name, prior := range priorRules {
			current, ok := currentRules[name]
			if !ok || current.relation != prior.relation {
				return fmt.Errorf("protected migration %s does not carry forward rule contract %s", migration.name, name)
			}
		}
		for name := range spec.footprintRelations {
			if _, ok := currentRelations[name]; !ok {
				return fmt.Errorf("protected migration %s names unknown footprint relation %s", migration.name, name)
			}
		}
		for name := range spec.footprintRules {
			if _, ok := currentRules[name]; !ok {
				return fmt.Errorf("protected migration %s names unknown footprint rule %s", migration.name, name)
			}
		}
		priorRelations = currentRelations
		priorRules = currentRules
		first = false
	}
	return nil
}

type migrationQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type protectedMigrationState struct {
	ledgerExists        bool
	receiptExists       bool
	receiptSHA256       string
	receiptSchemaSHA256 string
	receiptRevision     string
	currentSchemaSHA256 string
	anyFootprint        bool
	complete            bool
}

func inspectProtectedMigration(ctx context.Context, db migrationQueryer, name string, spec protectedMigrationSpec) (protectedMigrationState, error) {
	state := protectedMigrationState{complete: true}
	var ledgerExists bool
	if err := db.QueryRowContext(ctx, `SELECT to_regclass('public.nhs_schema_migrations') IS NOT NULL`).Scan(&ledgerExists); err != nil {
		return state, fmt.Errorf("inspect migration receipt table: %w", err)
	}
	state.ledgerExists = ledgerExists
	if ledgerExists {
		err := db.QueryRowContext(ctx, `
			SELECT sha256, schema_sha256, applied_by_commit
			FROM public.nhs_schema_migrations
			WHERE name = $1`, name).Scan(&state.receiptSHA256, &state.receiptSchemaSHA256, &state.receiptRevision)
		switch {
		case err == nil:
			state.receiptExists = true
		case err == sql.ErrNoRows:
		case err != nil:
			return state, fmt.Errorf("read migration receipt: %w", err)
		}
	}

	for _, relation := range spec.relations {
		var actualKind, actualParent string
		if err := db.QueryRowContext(ctx, `
			SELECT COALESCE((
				SELECT c.relkind::text || ':' || COALESCE(parent.relname, '')
				FROM pg_catalog.pg_class c
				JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
				LEFT JOIN pg_catalog.pg_index ix ON ix.indexrelid = c.oid
				LEFT JOIN pg_catalog.pg_class parent ON parent.oid = ix.indrelid
				WHERE n.nspname = 'public' AND c.relname = $1
				LIMIT 1
			), '')`, relation.name).Scan(&actualKind); err != nil {
			return state, fmt.Errorf("inspect relation %s: %w", relation.name, err)
		}
		if actualKind != "" {
			if spec.allObjectsAreFootprint || spec.footprintRelations[relation.name] {
				state.anyFootprint = true
			}
			parts := strings.SplitN(actualKind, ":", 2)
			actualKind = parts[0]
			if len(parts) == 2 {
				actualParent = parts[1]
			}
		}
		if actualKind != relation.relkind || actualParent != relation.parent {
			state.complete = false
		}
	}
	for _, rule := range spec.rules {
		var actualRelation string
		if err := db.QueryRowContext(ctx, `
			SELECT COALESCE((
				SELECT c.relname
				FROM pg_catalog.pg_rewrite r
				JOIN pg_catalog.pg_class c ON c.oid = r.ev_class
				JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
				WHERE n.nspname = 'public' AND r.rulename = $1
				LIMIT 1
			), '')`, rule.name).Scan(&actualRelation); err != nil {
			return state, fmt.Errorf("inspect rule %s: %w", rule.name, err)
		}
		if actualRelation != "" {
			if spec.allObjectsAreFootprint || spec.footprintRules[rule.name] {
				state.anyFootprint = true
			}
		}
		if actualRelation != rule.relation {
			state.complete = false
		}
	}
	if state.complete {
		fingerprint, err := protectedMigrationSchemaFingerprint(ctx, db, spec)
		if err != nil {
			return state, err
		}
		state.currentSchemaSHA256 = fingerprint
	}
	return state, nil
}

// protectedMigrationSchemaFingerprint captures the behavior-bearing catalog
// definitions for every table touched by the protected migration. Object names
// alone are insufficient: a same-named non-unique index, changed partial-index
// predicate, or altered append-only rule can violate accounting invariants.
func protectedMigrationSchemaFingerprint(ctx context.Context, db migrationQueryer, spec protectedMigrationSpec) (string, error) {
	tableSet := map[string]bool{}
	for _, relation := range spec.relations {
		if relation.relkind == "r" {
			tableSet[relation.name] = true
		}
	}
	for _, rule := range spec.rules {
		tableSet[rule.relation] = true
	}
	tables := make([]string, 0, len(tableSet))
	for table := range tableSet {
		tables = append(tables, table)
	}
	sort.Strings(tables)

	digest := sha256.New()
	for _, table := range tables {
		var definition string
		if err := db.QueryRowContext(ctx, `
			SELECT jsonb_build_object(
				'relation', c.relname,
				'kind', c.relkind::text,
				'persistence', c.relpersistence::text,
				'replica_identity', c.relreplident::text,
				'row_security', c.relrowsecurity,
				'force_row_security', c.relforcerowsecurity,
				'columns', COALESCE((
					SELECT jsonb_agg(jsonb_build_array(
						a.attnum,
						a.attname,
						pg_catalog.format_type(a.atttypid, a.atttypmod),
						a.attnotnull,
						COALESCE(pg_catalog.pg_get_expr(d.adbin, d.adrelid, true), ''),
						a.attidentity::text,
						a.attgenerated::text,
						COALESCE(coll.collname, '')
					) ORDER BY a.attnum)
					FROM pg_catalog.pg_attribute a
					LEFT JOIN pg_catalog.pg_attrdef d
						ON d.adrelid = a.attrelid AND d.adnum = a.attnum
					LEFT JOIN pg_catalog.pg_collation coll ON coll.oid = a.attcollation
					WHERE a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped
				), '[]'::jsonb),
				'constraints', COALESCE((
					SELECT jsonb_agg(jsonb_build_array(
						con.conname,
						con.contype::text,
						con.condeferrable,
						con.condeferred,
						con.convalidated,
						pg_catalog.pg_get_constraintdef(con.oid, true)
					) ORDER BY con.conname)
					FROM pg_catalog.pg_constraint con
					WHERE con.conrelid = c.oid
				), '[]'::jsonb),
				'indexes', COALESCE((
					SELECT jsonb_agg(jsonb_build_array(
						ic.relname,
						i.indisunique,
						i.indisprimary,
						i.indisexclusion,
						i.indimmediate,
						i.indisvalid,
						i.indisready,
						pg_catalog.pg_get_indexdef(i.indexrelid),
						COALESCE(pg_catalog.pg_get_expr(i.indpred, i.indrelid, true), ''),
						COALESCE(pg_catalog.pg_get_expr(i.indexprs, i.indrelid, true), '')
					) ORDER BY ic.relname)
					FROM pg_catalog.pg_index i
					JOIN pg_catalog.pg_class ic ON ic.oid = i.indexrelid
					WHERE i.indrelid = c.oid
				), '[]'::jsonb),
				'rules', COALESCE((
					SELECT jsonb_agg(jsonb_build_array(
						r.rulename,
						r.ev_type::text,
						r.is_instead,
						r.ev_enabled::text,
						r.ev_qual::text,
						r.ev_action::text
					) ORDER BY r.rulename)
					FROM pg_catalog.pg_rewrite r
					WHERE r.ev_class = c.oid AND r.rulename <> '_RETURN'
				), '[]'::jsonb),
				'triggers', COALESCE((
					SELECT jsonb_agg(jsonb_build_array(
						tg.tgname,
						tg.tgenabled::text,
						pg_catalog.pg_get_triggerdef(tg.oid, true)
					) ORDER BY tg.tgname)
					FROM pg_catalog.pg_trigger tg
					WHERE tg.tgrelid = c.oid AND NOT tg.tgisinternal
				), '[]'::jsonb),
				'policies', COALESCE((
					SELECT jsonb_agg(jsonb_build_array(
						pol.polname,
						pol.polcmd::text,
						pol.polpermissive,
						pol.polroles::text,
						COALESCE(pg_catalog.pg_get_expr(pol.polqual, pol.polrelid, true), ''),
						COALESCE(pg_catalog.pg_get_expr(pol.polwithcheck, pol.polrelid, true), '')
					) ORDER BY pol.polname)
					FROM pg_catalog.pg_policy pol
					WHERE pol.polrelid = c.oid
				), '[]'::jsonb)
			)::text
			FROM pg_catalog.pg_class c
			JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = 'public' AND c.relname = $1`, table).Scan(&definition); err != nil {
			return "", fmt.Errorf("fingerprint relation %s: %w", table, err)
		}
		_, _ = digest.Write([]byte(table))
		_, _ = digest.Write([]byte{0})
		_, _ = digest.Write([]byte(definition))
		_, _ = digest.Write([]byte{0})
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func validateProtectedMigrationState(migration migrationFile, state protectedMigrationState, validateLatestFingerprint bool) error {
	if state.receiptExists {
		if state.receiptSHA256 != migration.sha256 {
			return fmt.Errorf("protected migration %s: checksum drift; recorded receipt does not match repository file", migration.name)
		}
		if !state.complete {
			return fmt.Errorf("protected migration %s: schema drift; recorded receipt is missing required objects", migration.name)
		}
		if validateLatestFingerprint && state.receiptSchemaSHA256 != state.currentSchemaSHA256 {
			return fmt.Errorf("protected migration %s: schema fingerprint drift; invariant-bearing definitions changed", migration.name)
		}
		return nil
	}
	if state.anyFootprint {
		return fmt.Errorf("protected migration %s: ambiguous_prior_019; schema footprint exists without an exact receipt", migration.name)
	}
	// The ledger and 019 receipt are born in the same transaction. A ledger
	// without the first protected receipt is therefore anomalous. For 020+
	// migrations, an existing ledger containing earlier receipts is expected.
	if state.ledgerExists && migration.name == "019_provider_exchange.sql" {
		return fmt.Errorf("protected migration %s: migration receipt table exists without its required receipt", migration.name)
	}
	return nil
}

func applyLegacyMigration(ctx context.Context, conn *sql.Conn, migration migrationFile) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", migration.name, err)
	}
	statements := migrationStatements(string(migration.data))
	for i, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %s statement %d: %w", migration.name, i+1, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", migration.name, err)
	}
	log.Printf("legacy migration replayed atomically: %s (%d statements)", migration.name, len(statements))
	return nil
}

func applyProtectedMigration(ctx context.Context, conn *sql.Conn, migration migrationFile, spec protectedMigrationSpec, releaseRevision string, validateLatestFingerprint bool) (bool, error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin protected migration %s: %w", migration.name, err)
	}
	rollback := func(err error) (bool, error) {
		_ = tx.Rollback()
		return false, err
	}
	state, err := inspectProtectedMigration(ctx, tx, migration.name, spec)
	if err != nil {
		return rollback(fmt.Errorf("inspect protected migration %s: %w", migration.name, err))
	}
	if err := validateProtectedMigrationState(migration, state, validateLatestFingerprint); err != nil {
		return rollback(err)
	}
	if state.receiptExists {
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("commit protected migration receipt check %s: %w", migration.name, err)
		}
		return false, nil
	}

	revision, err := protectedMigrationRevision(releaseRevision)
	if err != nil {
		return rollback(fmt.Errorf("protected migration %s: %w", migration.name, err))
	}
	if _, err := tx.ExecContext(ctx, migrationReceiptTableSQL); err != nil {
		return rollback(fmt.Errorf("create migration receipt table for %s: %w", migration.name, err))
	}
	statements := migrationStatements(string(migration.data))
	for i, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return rollback(fmt.Errorf("protected migration %s statement %d: %w", migration.name, i+1, err))
		}
	}
	postState, err := inspectProtectedMigration(ctx, tx, migration.name, spec)
	if err != nil {
		return rollback(fmt.Errorf("verify protected migration %s: %w", migration.name, err))
	}
	if !postState.complete {
		return rollback(fmt.Errorf("protected migration %s did not create its complete schema footprint", migration.name))
	}
	if postState.currentSchemaSHA256 == "" {
		return rollback(fmt.Errorf("protected migration %s did not produce a schema fingerprint", migration.name))
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO public.nhs_schema_migrations (name, sha256, schema_sha256, applied_by_commit)
		VALUES ($1, $2, $3, $4)`, migration.name, migration.sha256, postState.currentSchemaSHA256, revision); err != nil {
		return rollback(fmt.Errorf("record protected migration %s: %w", migration.name, err))
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit protected migration %s: %w", migration.name, err)
	}
	return true, nil
}

func protectedMigrationRevision(candidate string) (string, error) {
	revision := strings.ToLower(strings.TrimSpace(candidate))
	if len(revision) != 40 {
		return "", fmt.Errorf("release revision must be the full 40-character Git commit before applying a protected migration")
	}
	if _, err := hex.DecodeString(revision); err != nil {
		return "", fmt.Errorf("release revision must be a hexadecimal Git commit")
	}
	return revision, nil
}

// migrationStatements splits PostgreSQL migration files without breaking
// quoted strings, quoted identifiers, comments, or dollar-quoted function
// bodies. Each file is still executed inside one transaction by RunMigrations,
// so any statement failure rolls the entire file back.
func migrationStatements(data string) []string {
	var statements []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	inLineComment := false
	blockCommentDepth := 0
	dollarTag := ""

	appendStatement := func() {
		stmt := strings.TrimSpace(current.String())
		if stmt != "" {
			statements = append(statements, stmt)
		}
		current.Reset()
	}

	for i := 0; i < len(data); {
		if inLineComment {
			if data[i] == '\n' {
				inLineComment = false
				current.WriteByte('\n')
			}
			i++
			continue
		}
		if blockCommentDepth > 0 {
			if i+1 < len(data) && data[i:i+2] == "/*" {
				blockCommentDepth++
				i += 2
				continue
			}
			if i+1 < len(data) && data[i:i+2] == "*/" {
				blockCommentDepth--
				i += 2
				if blockCommentDepth == 0 {
					current.WriteByte(' ')
				}
				continue
			}
			i++
			continue
		}
		if dollarTag != "" {
			if strings.HasPrefix(data[i:], dollarTag) {
				current.WriteString(dollarTag)
				i += len(dollarTag)
				dollarTag = ""
				continue
			}
			current.WriteByte(data[i])
			i++
			continue
		}
		if inSingle {
			current.WriteByte(data[i])
			if data[i] == '\\' && i+1 < len(data) {
				current.WriteByte(data[i+1])
				i += 2
				continue
			}
			if data[i] == '\'' {
				if i+1 < len(data) && data[i+1] == '\'' {
					current.WriteByte(data[i+1])
					i += 2
					continue
				}
				inSingle = false
			}
			i++
			continue
		}
		if inDouble {
			current.WriteByte(data[i])
			if data[i] == '"' {
				if i+1 < len(data) && data[i+1] == '"' {
					current.WriteByte(data[i+1])
					i += 2
					continue
				}
				inDouble = false
			}
			i++
			continue
		}

		if i+1 < len(data) && data[i:i+2] == "--" {
			inLineComment = true
			i += 2
			continue
		}
		if i+1 < len(data) && data[i:i+2] == "/*" {
			blockCommentDepth = 1
			i += 2
			continue
		}
		switch data[i] {
		case '\'':
			inSingle = true
			current.WriteByte(data[i])
			i++
		case '"':
			inDouble = true
			current.WriteByte(data[i])
			i++
		case '$':
			tag, width := postgresDollarQuoteTag(data[i:])
			if width == 0 {
				current.WriteByte(data[i])
				i++
				continue
			}
			dollarTag = tag
			current.WriteString(tag)
			i += width
		case ';':
			appendStatement()
			i++
		default:
			current.WriteByte(data[i])
			i++
		}
	}
	appendStatement()
	return statements
}

func postgresDollarQuoteTag(data string) (string, int) {
	if len(data) < 2 || data[0] != '$' {
		return "", 0
	}
	for i := 1; i < len(data); i++ {
		switch c := data[i]; {
		case c == '$':
			return data[:i+1], i + 1
		case c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9':
			continue
		default:
			return "", 0
		}
	}
	return "", 0
}
