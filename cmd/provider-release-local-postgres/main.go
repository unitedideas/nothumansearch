// provider-release-local-postgres runs the exact release verifier against two
// temporary local PostgreSQL 17 instances. It never deploys or contacts a
// provider, and never prints database connection strings.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/unitedideas/nothumansearch/internal/testpostgres"
)

func main() {
	repository := flag.String("repository", ".", "repository containing tools/prepare-exact-release.sh")
	candidate := flag.String("candidate", "HEAD", "candidate Git revision")
	base := flag.String("base", "", "ancestor Git revision; defaults to candidate parent")
	flag.Parse()

	root, err := filepath.Abs(*repository)
	if err != nil {
		fatalf("resolve repository: %v", err)
	}
	script := filepath.Join(root, "tools", "prepare-exact-release.sh")
	if info, err := os.Stat(script); err != nil || info.IsDir() {
		fatalf("exact release script unavailable at %s", script)
	}

	temporaryRoot, err := os.MkdirTemp("", "nhs-provider-release-postgres-")
	if err != nil {
		fatalf("create disposable PostgreSQL root: %v", err)
	}
	defer os.RemoveAll(temporaryRoot)

	providerDatabase, err := testpostgres.Start(filepath.Join(temporaryRoot, "provider-model"))
	if err != nil {
		fatalf("start provider-model PostgreSQL: %v", err)
	}
	defer closeDatabase("provider-model", providerDatabase)

	migrationDatabase, err := testpostgres.Start(filepath.Join(temporaryRoot, "migration-ledger"))
	if err != nil {
		fatalf("start migration-ledger PostgreSQL: %v", err)
	}
	defer closeDatabase("migration-ledger", migrationDatabase)

	arguments := []string{*candidate}
	if strings.TrimSpace(*base) != "" {
		arguments = append(arguments, *base)
	}
	command := exec.Command(script, arguments...)
	command.Dir = root
	command.Env = withEnvironment(os.Environ(), map[string]string{
		"NHS_TEST_POSTGRES_DSN":           providerDatabase.DSN(),
		"NHS_MIGRATION_TEST_POSTGRES_DSN": migrationDatabase.DSN(),
	})
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		fatalf("exact release verification: %v", err)
	}
	fmt.Println("exact release verification passed with disposable local PostgreSQL")
}

func closeDatabase(name string, database *testpostgres.Instance) {
	if err := database.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "stop %s PostgreSQL: %v\n", name, err)
	}
}

func withEnvironment(existing []string, replacements map[string]string) []string {
	result := make([]string, 0, len(existing)+len(replacements))
	for _, entry := range existing {
		name, _, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		if _, replaced := replacements[name]; !replaced {
			result = append(result, entry)
		}
	}
	for name, value := range replacements {
		result = append(result, name+"="+value)
	}
	return result
}

func fatalf(format string, values ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", values...)
	os.Exit(1)
}
