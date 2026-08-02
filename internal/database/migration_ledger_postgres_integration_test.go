package database

import (
	"crypto/sha256"
	"encoding/hex"
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

	t.Run("concurrent exact apply and replay", func(t *testing.T) {
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

	t.Run("database ahead", func(t *testing.T) {
		if _, err := DB.Exec(`
			INSERT INTO nhs_schema_migrations (name, sha256, schema_sha256, applied_by_commit)
			VALUES ('020_future.sql', $1, $2, $3)`, strings.Repeat("f", 64), strings.Repeat("e", 64), revision); err != nil {
			t.Fatal(err)
		}
		if err := RunMigrations(repositoryMigrations, revision); err == nil || !strings.Contains(err.Error(), "database_ahead_of_binary") {
			t.Fatalf("database-ahead error = %v", err)
		}
		if _, err := DB.Exec(`DELETE FROM nhs_schema_migrations WHERE name = '020_future.sql'`); err != nil {
			t.Fatal(err)
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
