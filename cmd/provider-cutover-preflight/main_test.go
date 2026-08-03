package main

import (
	"os"
	"strings"
	"testing"
)

func TestCutoverPreflightOutputContractIsBounded(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, required := range []string{
		preflightContract,
		"InspectMigrationReadiness",
		"ValidateProviderExchangeSigningRetentionReadOnlyContext",
		"ConnectWithReleaseRevisionContext",
		"candidate_revision_mismatch",
		"binary_revision",
		"live_pre_handoff_tickets",
		"old_writer_sessions_zero",
		"report_sha256",
		"if !report.ReadyForQuiescedCutover",
		"application_name NOT LIKE 'nhs-server:%'",
		"report.CandidateSessions == 0",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("preflight is missing %q", required)
		}
	}
	for _, forbidden := range []string{
		`os.Getenv("DATABASE_URL")`,
		"signed_receipt",
		"signature",
		"token_hash",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("preflight source contains forbidden output path %q", forbidden)
		}
	}
}

func TestValidRevision(t *testing.T) {
	if !validRevision(strings.Repeat("a", 40)) {
		t.Fatal("full lowercase Git revision was rejected")
	}
	for _, invalid := range []string{
		strings.Repeat("a", 39), strings.Repeat("A", 40), strings.Repeat("z", 40),
	} {
		if validRevision(invalid) {
			t.Fatalf("invalid revision was accepted: %q", invalid)
		}
	}
}
