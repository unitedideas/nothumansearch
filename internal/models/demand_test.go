package models

import (
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestClassifyDemandTopicsReturnsOnlyControlledTopics(t *testing.T) {
	query := "Payment API for buyer@example.com https://private.example/path confidential-credential-material"
	got := ClassifyDemandTopics(query, "developer")
	for _, want := range []string{"developer-tools", "payments"} {
		if !slices.Contains(got, want) {
			t.Fatalf("ClassifyDemandTopics = %v, missing %q", got, want)
		}
	}
	joined := strings.Join(got, " ")
	for _, forbidden := range []string{"buyer", "example", "private", "confidential"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("controlled topics %q retained caller text %q", joined, forbidden)
		}
	}
}

func TestClassifyDemandTopicsDoesNotEchoUnknownText(t *testing.T) {
	got := ClassifyDemandTopics("Acme confidential zephyr launch", "not-a-real-category")
	if !slices.Equal(got, []string{"other"}) {
		t.Fatalf("ClassifyDemandTopics = %v, want [other]", got)
	}
}

func TestGenerateDemandSearchIDUsesPublicPrefixAndVaries(t *testing.T) {
	first, err := GenerateDemandSearchID()
	if err != nil {
		t.Fatalf("first GenerateDemandSearchID: %v", err)
	}
	second, err := GenerateDemandSearchID()
	if err != nil {
		t.Fatalf("second GenerateDemandSearchID: %v", err)
	}
	if !strings.HasPrefix(first, "nhs_sr_") {
		t.Fatalf("GenerateDemandSearchID = %q, missing prefix", first)
	}
	if first == second {
		t.Fatal("GenerateDemandSearchID returned a duplicate")
	}
}

func TestGenerateDemandSearchIDFailsClosedWithoutEntropy(t *testing.T) {
	original := readDemandSearchEntropy
	readDemandSearchEntropy = func([]byte) (int, error) {
		return 0, errors.New("entropy unavailable")
	}
	t.Cleanup(func() { readDemandSearchEntropy = original })

	searchID, err := GenerateDemandSearchID()
	if !errors.Is(err, ErrDemandEntropyUnavailable) {
		t.Fatalf("GenerateDemandSearchID error = %v, want ErrDemandEntropyUnavailable", err)
	}
	if searchID != "" {
		t.Fatalf("GenerateDemandSearchID returned capability %q after entropy failure", searchID)
	}
}

func TestNormalizeProviderDomain(t *testing.T) {
	got := NormalizeProviderDomain(" HTTPS://WWW.Example.COM/path?token=secret ")
	if got != "example.com" {
		t.Fatalf("NormalizeProviderDomain = %q, want example.com", got)
	}
}

func TestDemandMigrationCannotPersistRawOrNetworkIdentityFields(t *testing.T) {
	source, err := os.ReadFile("../../migrations/018_agent_demand.sql")
	if err != nil {
		t.Fatalf("read demand migration: %v", err)
	}
	text := string(source)
	for _, forbidden := range []string{
		"query_fingerprint TEXT",
		"query_label TEXT",
		"anonymous_hash TEXT",
		"ip_hash TEXT",
		"user_agent TEXT",
		"result_impressions",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("demand migration contains forbidden field/table %q", forbidden)
		}
	}
	for _, required := range []string{
		"organic_results_returned",
		"result_selections",
		"demand_topics TEXT[]",
		"is_synthetic BOOLEAN NOT NULL DEFAULT false",
		"ADD COLUMN IF NOT EXISTS is_synthetic",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("demand migration missing %q", required)
		}
	}
}

func TestProviderDemandReportsExcludeSyntheticReceiptsAndUseReceiptTerms(t *testing.T) {
	source, err := os.ReadFile("demand.go")
	if err != nil {
		t.Fatalf("read demand model: %v", err)
	}
	text := string(source)
	start := strings.Index(text, "func GetProviderDemandAnalytics")
	if start < 0 {
		t.Fatal("GetProviderDemandAnalytics not found")
	}
	reportSource := text[start:]
	if got := strings.Count(reportSource, "AND NOT sr.is_synthetic"); got != 4 {
		t.Fatalf("provider report synthetic exclusions = %d, want 4 (summary, surfaces, topics, action types)", got)
	}
	for _, forbidden := range []string{`"search_sessions"`, `"detail_requests"`, `"detail_request_rate"`} {
		if strings.Contains(reportSource, forbidden) {
			t.Fatalf("provider report retained misleading metric %s", forbidden)
		}
	}
	for _, required := range []string{`"search_receipts"`, `"result_selections"`, `"result_selection_rate"`, `"action_interest_receipts"`, `"action_interest_rate"`, `"action_types"`} {
		if !strings.Contains(reportSource, required) {
			t.Fatalf("provider report missing receipt-accurate metric %s", required)
		}
	}
	if got := strings.Count(reportSource, "interest.expires_at > $3::timestamptz"); got != 4 {
		t.Fatalf("provider report live action-interest filters = %d, want 4 (summary, surfaces, topics, action types)", got)
	}
	if got := strings.Count(reportSource, "interest.created_at <= $3::timestamptz"); got != 4 {
		t.Fatalf("provider report action-interest upper cohort bounds = %d, want 4", got)
	}
	if got := strings.Count(reportSource, "returned.returned_at >= $3::timestamptz - $2::int * INTERVAL '1 day'"); got != 4 {
		t.Fatalf("provider report source-cohort filters = %d, want 4 (summary, surfaces, topics, action types)", got)
	}
	if got := strings.Count(reportSource, "returned.returned_at <= $3::timestamptz"); got != 4 {
		t.Fatalf("provider report source-cohort upper bounds = %d, want 4", got)
	}
	if !strings.Contains(reportSource, "SELECT clock_timestamp()") {
		t.Fatal("provider report does not capture one database cohort timestamp")
	}
	for _, required := range []string{"sql.LevelRepeatableRead", "ReadOnly:  true", "tx.QueryRow", "tx.Query", "tx.Commit"} {
		if !strings.Contains(reportSource, required) {
			t.Fatalf("provider report does not hold one repeatable-read cohort: missing %q", required)
		}
	}
	if !strings.Contains(reportSource, `"action_interest_cohort":`) || !strings.Contains(reportSource, `"organic_result_returned_at"`) {
		t.Fatal("provider report does not declare its action-interest cohort")
	}
	if got := strings.Count(reportSource, `"action_interest_suppressed"`); got < 3 {
		t.Fatalf("provider report action-interest suppression branches = %d, want summary, surface, and topic protection", got)
	}
	for _, required := range []string{
		`if actionInterests >= ProviderDemandPrivacyThreshold`,
		`if interests >= ProviderDemandPrivacyThreshold`,
		`"action_interest_receipt_threshold"`,
	} {
		if !strings.Contains(reportSource, required) {
			t.Fatalf("provider report missing privacy suppression contract %s", required)
		}
	}
	for _, required := range []string{`days > 30`, `"retention_days":`} {
		if !strings.Contains(reportSource, required) {
			t.Fatalf("provider report missing truthful 30-day retention contract %s", required)
		}
	}
}

func TestDemandReceiptWritesAreBatchedAndNilStoreFailsClosed(t *testing.T) {
	if err := RecordDemandSearch(nil, DemandSearchReceipt{}, nil); !errors.Is(err, ErrDemandStoreUnavailable) {
		t.Fatalf("nil demand store error = %v, want ErrDemandStoreUnavailable", err)
	}
	source, err := os.ReadFile("demand.go")
	if err != nil {
		t.Fatalf("read demand model: %v", err)
	}
	if !strings.Contains(string(source), "FROM unnest($2::text[], $3::text[], $4::bigint[], $5::bigint[])") {
		t.Fatal("demand results are not written with the bounded set-based insert")
	}
}

func TestDemandSelectionReportsOnlyNewlyInsertedSelections(t *testing.T) {
	source, err := os.ReadFile("demand.go")
	if err != nil {
		t.Fatalf("read demand model: %v", err)
	}
	text := string(source)
	start := strings.Index(text, "func RecordDemandSelection")
	end := strings.Index(text, "func NormalizeProviderDomain")
	if start < 0 || end <= start {
		t.Fatal("could not isolate RecordDemandSelection source")
	}
	selectionSource := text[start:end]
	if !strings.Contains(selectionSource, "SELECT EXISTS(SELECT 1 FROM inserted)") {
		t.Fatal("selection truth is not derived from the inserted row")
	}
	if strings.Contains(selectionSource, "SELECT EXISTS(SELECT 1 FROM candidate)") {
		t.Fatal("selection truth incorrectly reports a pre-existing candidate as newly recorded")
	}
}

func TestRetentionAndSmokeTelemetryGuards(t *testing.T) {
	serverSource, err := os.ReadFile("../../cmd/server/main.go")
	if err != nil {
		t.Fatalf("read server: %v", err)
	}
	for _, required := range []string{
		"DELETE FROM search_queries WHERE created_at < now() - interval '30 days'",
		"DELETE FROM usage_events WHERE created_at < now() - interval '35 days'",
		"DELETE FROM search_receipts WHERE created_at < now() - interval '30 days'",
		"models.RedactExpiredActionTicketIntent(database.DB)",
		"DELETE FROM magic_links WHERE used_at IS NOT NULL OR expires_at <= NOW()",
		"DELETE FROM sessions WHERE expires_at <= NOW()",
	} {
		if !strings.Contains(string(serverSource), required) {
			t.Fatalf("server retention missing %q", required)
		}
	}

	smokeSource, err := os.ReadFile("../../tools/smoke-test.sh")
	if err != nil {
		t.Fatalf("read smoke test: %v", err)
	}
	for _, line := range strings.Split(string(smokeSource), "\n") {
		if strings.Contains(line, "/usr/bin/curl") && !strings.Contains(line, `-H "$SYNTHETIC_HEADER"`) {
			t.Fatalf("smoke curl omitted synthetic marker: %s", line)
		}
	}
}

func TestPublicSiteJSONRetainsDeprecatedFeaturedShapeWithoutAffectingRank(t *testing.T) {
	payload, err := json.Marshal(Site{Domain: "example.com", IsFeatured: true})
	if err != nil {
		t.Fatalf("marshal site: %v", err)
	}
	if !strings.Contains(string(payload), `"is_featured":true`) {
		t.Fatalf("public v1 site JSON broke legacy is_featured shape: %s", payload)
	}
}
