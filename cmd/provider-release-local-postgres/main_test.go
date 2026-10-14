package main

import "testing"

func TestWithEnvironmentReplacesDatabaseVariables(t *testing.T) {
	environment := withEnvironment(
		[]string{
			"KEEP=this",
			"NHS_TEST_POSTGRES_DSN=old-provider",
			"NHS_MIGRATION_TEST_POSTGRES_DSN=old-ledger",
		},
		map[string]string{
			"NHS_TEST_POSTGRES_DSN":           "new-provider",
			"NHS_MIGRATION_TEST_POSTGRES_DSN": "new-ledger",
		},
	)
	seen := map[string]string{}
	for _, entry := range environment {
		for _, name := range []string{
			"KEEP",
			"NHS_TEST_POSTGRES_DSN",
			"NHS_MIGRATION_TEST_POSTGRES_DSN",
		} {
			prefix := name + "="
			if len(entry) >= len(prefix) && entry[:len(prefix)] == prefix {
				seen[name] = entry[len(prefix):]
			}
		}
	}
	if seen["KEEP"] != "this" {
		t.Fatalf("unrelated environment value = %q", seen["KEEP"])
	}
	if seen["NHS_TEST_POSTGRES_DSN"] != "new-provider" {
		t.Fatalf("provider DSN replacement = %q", seen["NHS_TEST_POSTGRES_DSN"])
	}
	if seen["NHS_MIGRATION_TEST_POSTGRES_DSN"] != "new-ledger" {
		t.Fatalf("migration DSN replacement = %q", seen["NHS_MIGRATION_TEST_POSTGRES_DSN"])
	}
}
