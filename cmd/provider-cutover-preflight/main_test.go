package main

import (
	"os"
	"strings"
	"testing"
	"time"
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
		"database_internal_sessions",
		"old_writer_sessions_zero",
		"report_sha256",
		"if !report.ReadyForQuiescedCutover",
		"databaseInternalAdministrativeSession",
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

func TestDatabaseInternalAdministrativeSessionIsExact(t *testing.T) {
	server := "fdaa:54:1487:a7b::2"
	if !databaseInternalAdministrativeSession("flypgadmin", "", server, server, "idle") {
		t.Fatal("exact local idle flypgadmin session was not classified as database-internal")
	}
	for _, test := range []struct {
		name, user, app, client, server, state string
	}{
		{"active", "flypgadmin", "", server, server, "active"},
		{"remote", "flypgadmin", "", "fdaa:54:1487:a7b::3", server, "idle"},
		{"named", "flypgadmin", "maintenance", server, server, "idle"},
		{"other user", "postgres", "", server, server, "idle"},
		{"missing address", "flypgadmin", "", "", "", "idle"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if databaseInternalAdministrativeSession(test.user, test.app, test.client, test.server, test.state) {
				t.Fatal("non-exact session was classified as database-internal")
			}
		})
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

func TestPreflightTimeoutIsBoundedForColdSchemaInspection(t *testing.T) {
	if preflightTimeout != 90*time.Second {
		t.Fatalf("preflight timeout = %s, want 90s", preflightTimeout)
	}
}
