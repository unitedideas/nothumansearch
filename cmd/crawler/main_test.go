package main

import (
	"os"
	"strings"
	"testing"
)

func TestCrawlerCannotMutateSchema(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, forbidden := range []string{"RunMigrations(", "ALTER TABLE"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("crawler retained forbidden schema mutation %q", forbidden)
		}
	}
}

func TestDryRunRejectsDatabaseOnlyOperations(t *testing.T) {
	for _, tc := range []struct{ dry, recrawl, recategorize, reject bool }{
		{true, true, false, true}, {true, false, true, true}, {true, true, true, true},
		{true, false, false, false}, {false, true, false, false}, {false, false, true, false},
	} {
		if err := validateDryRun(tc.dry, tc.recrawl, tc.recategorize); (err != nil) != tc.reject {
			t.Fatalf("mode %+v returned %v", tc, err)
		}
	}
}
