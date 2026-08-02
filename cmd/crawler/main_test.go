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
