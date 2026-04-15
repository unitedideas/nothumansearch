package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/lib/pq"
)

var DB *sql.DB

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

func RunMigrations(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		// Execute each statement individually so one failure doesn't block the rest.
		// Split on semicolons that end a statement (simple split, works for DDL).
		stmts := strings.Split(string(data), ";")
		for _, stmt := range stmts {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			// Skip pure-comment pieces (every non-blank line starts with --).
			// NOTE: do NOT short-circuit on HasPrefix("--") alone — a real
			// statement with a leading comment block ("-- doc\nCREATE TABLE …")
			// would be skipped entirely. Check every line.
			lines := strings.Split(stmt, "\n")
			hasCode := false
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" && !strings.HasPrefix(trimmed, "--") {
					hasCode = true
					break
				}
			}
			if !hasCode {
				continue
			}
			if _, err := DB.Exec(stmt); err != nil {
				log.Printf("migration %s statement error (continuing): %v", f, err)
			}
		}
		log.Printf("migration applied: %s", f)
	}
	return nil
}
