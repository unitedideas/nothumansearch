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
		if _, err := DB.Exec(string(data)); err != nil {
			return fmt.Errorf("exec %s: %w", f, err)
		}
		log.Printf("migration applied: %s", f)
	}
	return nil
}
