package database

import (
	"os"
	"path/filepath"
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
