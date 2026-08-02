package database

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestMigrationStatementsIgnoresSemicolonsInLineComments(t *testing.T) {
	data := `-- Created at /fix/{host} intake; paid_at set by Stripe webhook;
CREATE TABLE IF NOT EXISTS geo_fix_jobs (
    id BIGSERIAL PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'pending',
    paid_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_geo_fix_jobs_status ON geo_fix_jobs(status);
`

	statements := migrationStatements(data)
	if len(statements) != 2 {
		t.Fatalf("expected 2 executable statements, got %d: %#v", len(statements), statements)
	}
	for _, stmt := range statements {
		if strings.HasPrefix(strings.TrimSpace(stmt), "paid_at set by Stripe webhook") {
			t.Fatalf("comment fragment became executable SQL: %q", stmt)
		}
	}
}

func TestGeoFixMigrationHasNoCommentFragments(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "012_geo_fix_jobs.sql"))
	if err != nil {
		t.Fatal(err)
	}
	statements := migrationStatements(string(data))
	if len(statements) == 0 {
		t.Fatal("expected geo-fix migration statements")
	}
	for _, stmt := range statements {
		if strings.HasPrefix(strings.TrimSpace(stmt), "paid_at") {
			t.Fatalf("unexpected paid_at-leading statement: %q", stmt)
		}
	}
}

func TestMigrationStatementsKeepDollarQuotedFunctionAtomic(t *testing.T) {
	data := `
		CREATE OR REPLACE FUNCTION redact_ticket()
		RETURNS TRIGGER LANGUAGE plpgsql AS $$
		BEGIN
			UPDATE action_tickets SET demand_topic='redacted' WHERE id=OLD.id;
			RETURN OLD;
		END;
		$$;
		CREATE TRIGGER redact BEFORE DELETE ON search_receipts
		FOR EACH ROW EXECUTE FUNCTION redact_ticket();
	`
	statements := migrationStatements(data)
	if len(statements) != 2 {
		t.Fatalf("expected function and trigger statements, got %d: %#v", len(statements), statements)
	}
	if !strings.Contains(statements[0], "UPDATE action_tickets") || !strings.Contains(statements[0], "RETURN OLD;") {
		t.Fatalf("dollar-quoted function body was split: %q", statements[0])
	}
}

func TestProviderExchangeMigrationKeepsRedactionRuleAtomic(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "019_provider_exchange.sql"))
	if err != nil {
		t.Fatal(err)
	}
	statements := migrationStatements(string(data))
	redactionRules := 0
	for _, stmt := range statements {
		if strings.Contains(stmt, "CREATE OR REPLACE RULE redact_action_ticket_intent_on_receipt_delete") {
			redactionRules++
			if !strings.Contains(stmt, "UPDATE action_tickets") || !strings.Contains(stmt, "WHERE search_receipt_id=OLD.id") {
				t.Fatalf("provider exchange redaction rule was not kept intact: %q", stmt)
			}
		}
	}
	if redactionRules != 1 {
		t.Fatalf("expected exactly one intact provider exchange redaction rule, got %d", redactionRules)
	}
}

func TestProviderExchangeProtectedContractCoversDeclaredObjects(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "019_provider_exchange.sql"))
	if err != nil {
		t.Fatal(err)
	}
	spec, ok := protectedMigrationSpecs["019_provider_exchange.sql"]
	if !ok {
		t.Fatal("provider exchange migration has no protected schema contract")
	}

	declaredRelations := map[string]string{}
	relationPattern := regexp.MustCompile(`(?m)^CREATE (?:UNIQUE )?(TABLE|INDEX) IF NOT EXISTS ([a-z0-9_]+)`)
	for _, match := range relationPattern.FindAllStringSubmatch(string(data), -1) {
		kind := "i"
		if match[1] == "TABLE" {
			kind = "r"
		}
		declaredRelations[match[2]] = kind
	}
	declaredIndexParents := map[string]string{}
	indexPattern := regexp.MustCompile(`(?m)^CREATE (?:UNIQUE )?INDEX IF NOT EXISTS ([a-z0-9_]+)\s+ON ([a-z0-9_]+)`)
	for _, match := range indexPattern.FindAllStringSubmatch(string(data), -1) {
		declaredIndexParents[match[1]] = match[2]
	}
	if len(declaredRelations) != len(spec.relations) {
		t.Fatalf("protected relation contract has %d entries, migration declares %d", len(spec.relations), len(declaredRelations))
	}
	for _, relation := range spec.relations {
		if want, ok := declaredRelations[relation.name]; !ok || relation.relkind != want {
			t.Fatalf("protected relation contract mismatch for %s: kind=%q declared=%q present=%t", relation.name, relation.relkind, want, ok)
		}
		if relation.relkind == "i" && relation.parent != declaredIndexParents[relation.name] {
			t.Fatalf("protected index %s parent=%q, migration declares %q", relation.name, relation.parent, declaredIndexParents[relation.name])
		}
	}

	declaredRules := map[string]string{}
	rulePattern := regexp.MustCompile(`(?m)^CREATE OR REPLACE RULE ([a-z0-9_]+) AS\s+ON (?:UPDATE|DELETE) TO ([a-z0-9_]+)`)
	for _, match := range rulePattern.FindAllStringSubmatch(string(data), -1) {
		declaredRules[match[1]] = match[2]
	}
	if len(declaredRules) != len(spec.rules) {
		t.Fatalf("protected rule contract has %d entries, migration declares %d", len(spec.rules), len(declaredRules))
	}
	for _, rule := range spec.rules {
		if want, ok := declaredRules[rule.name]; !ok || rule.relation != want {
			t.Fatalf("protected rule contract mismatch for %s: relation=%q declared=%q present=%t", rule.name, rule.relation, want, ok)
		}
	}
}

func TestProtectedMigrationStateFailsClosed(t *testing.T) {
	migration := migrationFile{name: "019_provider_exchange.sql", sha256: strings.Repeat("a", 64)}
	for _, test := range []struct {
		name  string
		state protectedMigrationState
		want  string
	}{
		{name: "clean", state: protectedMigrationState{}, want: ""},
		{name: "ambiguous footprint", state: protectedMigrationState{anyFootprint: true}, want: "ambiguous_prior_019"},
		{name: "orphan ledger", state: protectedMigrationState{ledgerExists: true}, want: "without its required receipt"},
		{name: "checksum drift", state: protectedMigrationState{receiptExists: true, receiptSHA256: strings.Repeat("b", 64), receiptSchemaSHA256: strings.Repeat("c", 64), currentSchemaSHA256: strings.Repeat("c", 64), complete: true}, want: "checksum drift"},
		{name: "schema drift", state: protectedMigrationState{receiptExists: true, receiptSHA256: strings.Repeat("a", 64)}, want: "schema drift"},
		{name: "definition drift", state: protectedMigrationState{receiptExists: true, receiptSHA256: strings.Repeat("a", 64), receiptSchemaSHA256: strings.Repeat("b", 64), currentSchemaSHA256: strings.Repeat("c", 64), complete: true}, want: "schema fingerprint drift"},
		{name: "exact receipt", state: protectedMigrationState{receiptExists: true, receiptSHA256: strings.Repeat("a", 64), receiptSchemaSHA256: strings.Repeat("c", 64), currentSchemaSHA256: strings.Repeat("c", 64), complete: true}, want: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateProtectedMigrationState(migration, test.state, true)
			if test.want == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if test.want != "" && (err == nil || !strings.Contains(err.Error(), test.want)) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
	future := migrationFile{name: "020_future.sql", sha256: strings.Repeat("b", 64)}
	if err := validateProtectedMigrationState(future, protectedMigrationState{ledgerExists: true}, false); err != nil {
		t.Fatalf("future protected migration could not use existing ledger: %v", err)
	}
	superseded := protectedMigrationState{
		receiptExists: true, receiptSHA256: strings.Repeat("a", 64), complete: true,
		receiptSchemaSHA256: strings.Repeat("b", 64), currentSchemaSHA256: strings.Repeat("c", 64),
	}
	if err := validateProtectedMigrationState(migration, superseded, false); err != nil {
		t.Fatalf("superseded protected receipt still owned latest schema fingerprint: %v", err)
	}
}

func TestProtectedMigrationRequiresExactReleaseRevision(t *testing.T) {
	valid := "0123456789abcdef0123456789abcdef01234567"
	if got, err := protectedMigrationRevision(valid); err != nil || got != valid {
		t.Fatalf("valid release revision = %q, err=%v", got, err)
	}
	for _, invalid := range []string{"", "development", strings.Repeat("g", 40), valid[:39]} {
		if _, err := protectedMigrationRevision(invalid); err == nil {
			t.Fatalf("invalid release revision %q was accepted", invalid)
		}
	}
}

func TestProtectedMigrationSpecsMustBeCumulative(t *testing.T) {
	base := protectedMigrationSpecs["019_provider_exchange.sql"]
	future := base
	future.allObjectsAreFootprint = false
	future.footprintRelations = nil
	future.footprintRules = nil
	protectedMigrationSpecs["020_future.sql"] = future
	t.Cleanup(func() { delete(protectedMigrationSpecs, "020_future.sql") })
	migrations := []migrationFile{
		{name: "019_provider_exchange.sql"},
		{name: "020_future.sql"},
	}
	if err := validateProtectedMigrationSpecChain(migrations); err != nil {
		t.Fatalf("cumulative future migration contract rejected: %v", err)
	}

	future.relations = append([]migrationRelation(nil), base.relations[:len(base.relations)-1]...)
	protectedMigrationSpecs["020_future.sql"] = future
	if err := validateProtectedMigrationSpecChain(migrations); err == nil || !strings.Contains(err.Error(), "does not carry forward relation contract") {
		t.Fatalf("non-cumulative future migration contract error = %v", err)
	}
}
