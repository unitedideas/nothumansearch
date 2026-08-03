package models

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestActionInterestVocabularyAndDomainBoundary(t *testing.T) {
	want := []string{"quote", "trial", "demo", "booking", "application", "signup", "purchase"}
	if got := ActionInterestTypes(); !slices.Equal(got, want) {
		t.Fatalf("ActionInterestTypes = %v, want %v", got, want)
	}
	for _, value := range want {
		if !ValidActionInterestType(value) {
			t.Fatalf("allowlisted action %q was rejected", value)
		}
	}
	for _, value := range []string{"lead", "contact", "other", "quote now", ""} {
		if ValidActionInterestType(value) {
			t.Fatalf("non-action %q was accepted", value)
		}
	}

	if got := NormalizeActionInterestDomain(" Example.COM. "); got != "example.com" {
		t.Fatalf("NormalizeActionInterestDomain = %q, want example.com", got)
	}
	for _, value := range []string{
		"https://example.com", "example.com/private", "example.com?token=secret",
		"user@example.com", "127.0.0.1", "localhost", "-bad.example",
	} {
		if got := NormalizeActionInterestDomain(value); got != "" {
			t.Fatalf("unsafe domain %q normalized to %q", value, got)
		}
	}
}

func TestGenerateActionInterestIDIsOpaqueAndFailsClosed(t *testing.T) {
	first, err := GenerateActionInterestID()
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateActionInterestID()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(first, "nhs_air_") || len(first) != len("nhs_air_")+16 || first == second {
		t.Fatalf("generated action-interest IDs %q and %q", first, second)
	}

	original := readActionInterestEntropy
	readActionInterestEntropy = func([]byte) (int, error) { return 0, errors.New("entropy unavailable") }
	t.Cleanup(func() { readActionInterestEntropy = original })
	if id, err := GenerateActionInterestID(); id != "" || !errors.Is(err, ErrActionInterestEntropyUnavailable) {
		t.Fatalf("entropy failure returned id=%q err=%v", id, err)
	}
}

func TestInvalidActionInterestFailsBeforeStoreAccess(t *testing.T) {
	valid := ActionInterestInput{
		SearchID: "nhs_sr_AAAAAAAAAAAAAAAA", Domain: "example.com", ActionType: "quote",
		Surface: "rest", CallerAttestsPrincipalInterest: true,
		ConfirmationVersion: ActionInterestConfirmationV1,
	}
	if _, err := RecordActionInterest(nil, valid); !errors.Is(err, ErrActionInterestStoreUnavailable) {
		t.Fatalf("valid request with nil store error = %v", err)
	}
	invalid := valid
	invalid.CallerAttestsPrincipalInterest = false
	if _, err := RecordActionInterest(nil, invalid); !errors.Is(err, ErrInvalidActionInterest) {
		t.Fatalf("false confirmation error = %v, want invalid request", err)
	}
	invalid = valid
	invalid.Domain = "https://example.com/private?secret=value"
	if _, err := RecordActionInterest(nil, invalid); !errors.Is(err, ErrInvalidActionInterest) {
		t.Fatalf("URL domain error = %v, want invalid request", err)
	}
}

func TestActionInterestMigrationPrivacyAndBindingContract(t *testing.T) {
	source, err := os.ReadFile("../../migrations/020_action_interest_receipts.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, required := range []string{
		"action_interest_receipts", "idx_search_receipts_id_synthetic",
		"CONSTRAINT action_interest_receipts_non_synthetic_fk",
		"FOREIGN KEY (search_receipt_id, source_is_synthetic)",
		"CHECK (NOT source_is_synthetic)",
		"CONSTRAINT action_interest_receipts_returned_result_fk",
		"FOREIGN KEY (search_receipt_id, site_domain_snapshot)",
		"REFERENCES organic_results_returned(search_receipt_id, site_domain_snapshot)",
		"ON DELETE CASCADE", "caller_attests_principal_interest", ActionInterestConfirmationV1,
		"action_interest_receipts_no_update", "expires_at",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("action-interest migration missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"raw_query text", "query_text text", "query_fingerprint text", "ip_hash text", "user_agent text",
		"email text", "phone text", "contact_data text", "free_form text", "notes text", "agent_identity text",
		"principal_identity text", "provider_offer_id uuid", "provider_claim_id uuid",
		"action_ticket_id uuid", "budget_band text", "bounty_cents", "outcome text", "region_code text",
		"site_id uuid",
	} {
		if strings.Contains(strings.ToLower(text), forbidden) {
			t.Fatalf("action-interest migration retained forbidden concept %q", forbidden)
		}
	}
	stage1IntegritySource, err := os.ReadFile("../../migrations/025_stage1_fact_integrity.sql")
	if err != nil {
		t.Fatal(err)
	}
	stage1Integrity := string(stage1IntegritySource)
	for _, required := range []string{
		"CONSTRAINT result_selections_returned_result_fk",
		"FOREIGN KEY (search_receipt_id, site_domain_snapshot)",
		"REFERENCES public.organic_results_returned(search_receipt_id, site_domain_snapshot)",
		"WHERE name = '025_stage1_fact_integrity.sql'",
	} {
		if !strings.Contains(stage1Integrity, required) {
			t.Fatalf("Stage 1 integrity migration missing %q", required)
		}
	}
}

func TestActionInterestReplayLookupPrecedesFreshEntropy(t *testing.T) {
	source, err := os.ReadFile("action_interest.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	start := strings.Index(text, "func RecordActionInterest")
	end := strings.Index(text[start:], "func GetStage1DemandProof")
	if start < 0 || end < 0 {
		t.Fatal("could not isolate RecordActionInterest")
	}
	recorder := text[start : start+end]
	existing := strings.Index(recorder, "existing := &ActionInterestReceipt")
	entropy := strings.Index(recorder, "GenerateActionInterestID()")
	if existing < 0 || entropy < 0 || entropy < existing {
		t.Fatalf("fresh entropy occurs before the locked replay lookup: existing=%d entropy=%d", existing, entropy)
	}
}

func TestStage1DemandProofSourceIsAggregateAndSyntheticSafe(t *testing.T) {
	source, err := os.ReadFile("action_interest.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	start := strings.Index(text, "func GetStage1DemandProof")
	if start < 0 {
		t.Fatal("GetStage1DemandProof missing")
	}
	report := text[start:]
	for _, required := range []string{
		"Isolation: sql.LevelRepeatableRead",
		"ReadOnly:  true",
		"SELECT clock_timestamp()",
		"AsOf:                             cohortAsOf",
		"COUNT(DISTINCT interest.search_receipt_id)",
		"COUNT(DISTINCT selection.search_receipt_id)",
		"name='025_stage1_fact_integrity.sql'",
		"ELSE '020_action_interest_receipts.sql'",
		"receipt.stage1_integrity_generation=1",
		"returned.stage1_integrity_generation=1",
		"selection.stage1_integrity_generation=1",
		"interest.stage1_integrity_generation=1",
		"eligible_searches AS",
		"EXISTS (\n\t\t\t\tSELECT 1 FROM organic_results_returned returned",
		"returned.returned_at >= cohort_window.started_at",
		"returned.returned_at <= cohort_window.now_at",
		"returned.returned_at >= GREATEST(",
		"returned.returned_at <= clock.now_at",
		"returned.returned_at >= receipt.returned_started_at",
		"returned.returned_at <= receipt.returned_as_of",
		"JOIN eligible_searches receipt ON receipt.id=selection.search_receipt_id",
		"returned.site_domain_snapshot=selection.site_domain_snapshot",
		"JOIN eligible_searches receipt ON receipt.id=interest.search_receipt_id",
		"HAVING COUNT(DISTINCT receipt.id) >= $2",
		"COUNT(DISTINCT returned.site_domain_snapshot) >= $5",
		"site.category<>'spam'",
		"HAVING COUNT(DISTINCT interest.search_receipt_id) >= $2",
		"CountsAreReceiptsNotUniqueAgents: true",
		"CommercialProof:                  false",
		"Stage1ObservationWindowDays",
		"Stage1CandidateTopicReceipts",
		"Stage1CandidateTopicDomains",
		"bucket.Value != \"other\"",
		"topic = ANY($4::text[])",
		"stage1ControlledDemandTopics()",
		"proof.PilotCandidateTopicAvailable",
		"proof.SearchReceiptsWithSelection >= proof.Targets[\"search_receipts_with_selection\"]",
	} {
		if !strings.Contains(report, required) {
			t.Fatalf("Stage 1 demand proof missing %q", required)
		}
	}
	if got := strings.Count(report, "NOT receipt.is_synthetic"); got < 3 {
		t.Fatalf("Stage 1 eligible-cohort synthetic exclusions = %d, want at least 3", got)
	}
	if got := strings.Count(report, "returned.returned_at >="); got < 5 {
		t.Fatalf("Stage 1 participating-result lower bounds = %d, want at least 5", got)
	}
	if got := strings.Count(report, "returned.returned_at <="); got < 5 {
		t.Fatalf("Stage 1 participating-result cutoffs = %d, want at least 5", got)
	}
	if strings.Contains(report, "receipt.result_count > 0") {
		t.Fatal("Stage 1 meaningful-search proof trusts total matches instead of returned rows")
	}
	if got := strings.Count(report, "interest.expires_at >"); got != 2 {
		t.Fatalf("Stage 1 live action-interest filters = %d, want 2 cohort queries", got)
	}
	for _, forbidden := range []string{"SELECT receipt.public_id", "SELECT interest.public_id", "agent_identity", "raw_query"} {
		if strings.Contains(report, forbidden) {
			t.Fatalf("Stage 1 aggregate exposes forbidden source %q", forbidden)
		}
	}
}
