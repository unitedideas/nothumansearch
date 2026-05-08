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

	var failures int
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		for _, stmt := range migrationStatements(string(data)) {
			if _, err := DB.Exec(stmt); err != nil {
				log.Printf("migration %s statement error (continuing): %v", f, err)
				failures++
			}
		}
		log.Printf("migration applied: %s", f)
	}
	if failures > 0 {
		return fmt.Errorf("%d migration statements failed", failures)
	}
	return nil
}

func migrationStatements(data string) []string {
	var uncommented []string
	for _, line := range strings.Split(data, "\n") {
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		uncommented = append(uncommented, line)
	}

	var statements []string
	for _, stmt := range strings.Split(strings.Join(uncommented, "\n"), ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			statements = append(statements, stmt)
		}
	}
	return statements
}
