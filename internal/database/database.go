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
		tx, err := DB.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", f, err)
		}
		statements := migrationStatements(string(data))
		for i, stmt := range statements {
			if _, err := tx.Exec(stmt); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("migration %s statement %d: %w", f, i+1, err)
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", f, err)
		}
		log.Printf("migration applied atomically: %s (%d statements)", f, len(statements))
	}
	return nil
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
