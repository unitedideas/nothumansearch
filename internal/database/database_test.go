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
