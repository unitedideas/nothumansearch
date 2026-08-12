package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
)

func TestValidRevision(t *testing.T) {
	valid := "0123456789abcdef0123456789abcdef01234567"
	for name, test := range map[string]struct {
		value string
		want  bool
	}{
		"valid":      {valid, true},
		"short":      {valid[:39], false},
		"uppercase":  {"0123456789ABCDEF0123456789ABCDEF01234567", false},
		"non hex":    {"g123456789abcdef0123456789abcdef01234567", false},
		"whitespace": {" " + valid, false},
	} {
		t.Run(name, func(t *testing.T) {
			if got := validRevision(test.value); got != test.want {
				t.Fatalf("validRevision(%q) = %t, want %t", test.value, got, test.want)
			}
		})
	}
}

func TestAttemptCheckpointComparison(t *testing.T) {
	since := time.Date(2026, 8, 11, 9, 41, 28, 0, time.UTC)
	checkpointReport := &models.PostSelectionActionInterestExperiment{
		Contract:                        models.PostSelectionActionInterestExperimentContract,
		Since:                           since,
		CheckedAt:                       time.Date(2026, 8, 11, 10, 11, 43, 525805000, time.UTC),
		EligibleSurfaces:                []string{"mcp", "rest"},
		BoundarySpanningAttemptBuckets:  1,
		BoundarySpanningAttemptCount:    18,
		SpanningUnavailableAttemptCount: 18,
	}
	rawReport, err := json.Marshal(checkpointReport)
	if err != nil {
		t.Fatalf("marshal checkpoint report: %v", err)
	}
	digest := sha256.Sum256(rawReport)
	checkpoint := &attemptCheckpointReceipt{
		Contract:          readContract,
		ReportSHA256:      hex.EncodeToString(digest[:]),
		CandidateRevision: "7ae09dff9274506d58f4ccdc318d85a89813d948",
		BinaryRevision:    "7ae09dff9274506d58f4ccdc318d85a89813d948",
		Report:            checkpointReport,
	}
	current := *checkpointReport
	current.CheckedAt = checkpointReport.CheckedAt.Add(time.Hour)
	current.BoundarySpanningAttemptCount = 21
	current.SpanningUnavailableAttemptCount = 20
	current.ExactPostBoundaryAttempts = 1
	current.ExactInvalidAttempts = 1
	comparison, err := compareAttemptCheckpoint(checkpoint, &current)
	if err != nil {
		t.Fatalf("compare checkpoint: %v", err)
	}
	if comparison.AttemptDelta != 4 || comparison.UnavailableAttemptDelta != 2 ||
		comparison.InvalidRequestAttemptDelta != 1 || comparison.DemandEvidence ||
		comparison.CommercialProof || !comparison.CountsAreAttemptsNotUniqueAgents {
		t.Fatalf("checkpoint comparison = %#v", comparison)
	}

	current.BoundarySpanningAttemptCount = 17
	current.ExactPostBoundaryAttempts = 0
	if _, err := compareAttemptCheckpoint(checkpoint, &current); err == nil {
		t.Fatal("checkpoint comparison accepted a regressed counter")
	}
}

func TestFunnelCheckpointComparisonCoversDiscoveryIntentAndSettlement(t *testing.T) {
	since := time.Date(2026, 8, 11, 9, 41, 28, 0, time.UTC)
	checkpointReport := &models.PostSelectionActionInterestExperiment{
		Contract:                             models.PostSelectionActionInterestExperimentContract,
		Since:                                since,
		CheckedAt:                            time.Date(2026, 8, 12, 0, 41, 26, 0, time.UTC),
		EligibleSurfaces:                     []string{"mcp", "rest"},
		MeaningfulSearchReceipts:             10,
		DeveloperToolsSearchReceipts:         3,
		DeveloperToolsResultSelections:       1,
		DeveloperToolsSearchesSelected:       1,
		DeveloperToolsInterestReceipts:       1,
		DeveloperToolsSearchesInterested:     1,
		DeveloperToolsPostSelectionInterests: 1,
		DeveloperToolsPostSelectionSearches:  1,
		ResultSelections:                     3,
		SearchReceiptsWithSelection:          2,
		ActiveActionInterestReceipts:         1,
		SearchReceiptsWithActionInterest:     1,
		PostSelectionInterestReceipts:        1,
		PostSelectionSearchReceipts:          1,
		MCPSearchReceipts:                    4,
		MCPResultSelections:                  2,
		MCPPostSelectionInterests:            1,
		RESTSearchReceipts:                   6,
		RESTResultSelections:                 1,
		ProviderOffersReturned:               3,
		ProviderTicketsCreated:               2,
		ProviderHandoffsObserved:             1,
		ProviderOutcomesReported:             1,
		ProviderPaidSettlements:              1,
		CommercialStateEventsTotal:           8,
	}
	rawReport, err := json.Marshal(checkpointReport)
	if err != nil {
		t.Fatalf("marshal checkpoint report: %v", err)
	}
	digest := sha256.Sum256(rawReport)
	checkpoint := &attemptCheckpointReceipt{
		Contract:          readContract,
		ReportSHA256:      hex.EncodeToString(digest[:]),
		CandidateRevision: "7ad7cbfde04b239b6e12a563e05f765de9701df8",
		BinaryRevision:    "7ad7cbfde04b239b6e12a563e05f765de9701df8",
		Report:            checkpointReport,
	}
	current := *checkpointReport
	current.CheckedAt = checkpointReport.CheckedAt.Add(time.Hour)
	current.MeaningfulSearchReceipts = 15
	current.DeveloperToolsSearchReceipts = 5
	current.DeveloperToolsResultSelections = 2
	current.DeveloperToolsSearchesSelected = 2
	current.DeveloperToolsInterestReceipts = 2
	current.DeveloperToolsSearchesInterested = 2
	current.DeveloperToolsPostSelectionInterests = 2
	current.DeveloperToolsPostSelectionSearches = 2
	current.MCPSearchReceipts = 6
	current.RESTSearchReceipts = 9
	current.ResultSelections = 5
	current.SearchReceiptsWithSelection = 3
	current.MCPResultSelections = 3
	current.RESTResultSelections = 2
	current.ActiveActionInterestReceipts = 2
	current.SearchReceiptsWithActionInterest = 2
	current.PostSelectionInterestReceipts = 2
	current.PostSelectionSearchReceipts = 2
	current.MCPPostSelectionInterests = 1
	current.RESTPostSelectionInterests = 1
	current.ProviderOffersReturned = 4
	current.ProviderTicketsCreated = 3
	current.ProviderHandoffsObserved = 2
	current.ProviderOutcomesReported = 2
	current.ProviderPaidSettlements = 2
	current.ProviderAvailableSettlements = 1
	current.CommercialStateEventsTotal = 14

	attempts, err := compareAttemptCheckpoint(checkpoint, &current)
	if err != nil {
		t.Fatalf("compare attempt checkpoint: %v", err)
	}
	comparison, err := compareFunnelCheckpoint(checkpoint, &current, attempts)
	if err != nil {
		t.Fatalf("compare funnel checkpoint: %v", err)
	}
	if comparison.MeaningfulSearchReceiptsDelta != 5 ||
		comparison.DeveloperToolsSearchReceiptsDelta != 2 ||
		comparison.DeveloperToolsResultSelectionsDelta != 1 ||
		comparison.DeveloperToolsSearchesSelectedDelta != 1 ||
		comparison.DeveloperToolsInterestReceiptsNetChange != 1 ||
		comparison.DeveloperToolsSearchesInterestedNetChange != 1 ||
		comparison.DeveloperToolsPostSelectionInterestsNetChange != 1 ||
		comparison.DeveloperToolsPostSelectionSearchesNetChange != 1 ||
		comparison.MCPSearchReceiptsDelta != 2 || comparison.RESTSearchReceiptsDelta != 3 ||
		comparison.ResultSelectionsDelta != 2 || comparison.SearchReceiptsWithSelectionDelta != 1 ||
		comparison.PostSelectionInterestReceiptsNetChange != 1 ||
		comparison.ProviderHandoffsObservedDelta != 1 ||
		comparison.ProviderPaidSettlementsDelta != 1 ||
		comparison.ProviderAvailableSettlementsDelta != 1 ||
		!comparison.DiscoveryUsageObserved || !comparison.ResultSelectionObserved ||
		!comparison.ExplicitPostSelectionInterestNetIncrease || !comparison.ProviderHandoffObserved ||
		!comparison.PaidSettlementObserved || !comparison.AvailableSettlementObserved ||
		!comparison.CountsAreEventsNotUniqueAgents || !comparison.ActiveInterestStateMayExpire ||
		!comparison.ActiveInterestNetChangeIsNotCreatedEvents ||
		!comparison.SearchesAreNotLeads || comparison.StrongestMechanismSelected ||
		comparison.ContainsIdentifiers || comparison.ContainsQueriesOrPrompts || comparison.ContainsContactData {
		t.Fatalf("funnel checkpoint comparison = %#v", comparison)
	}
	rawComparison, err := json.Marshal(comparison)
	if err != nil {
		t.Fatalf("marshal funnel comparison: %v", err)
	}
	var decodedComparison any
	if err := json.Unmarshal(rawComparison, &decodedComparison); err != nil {
		t.Fatalf("decode funnel comparison: %v", err)
	}
	assertNoForbiddenFunnelKeys(t, decodedComparison)

	expired := current
	expired.ActiveActionInterestReceipts = 0
	expired.SearchReceiptsWithActionInterest = 0
	expired.PostSelectionInterestReceipts = 0
	expired.PostSelectionSearchReceipts = 0
	expired.MCPPostSelectionInterests = 0
	expired.RESTPostSelectionInterests = 0
	expired.DeveloperToolsInterestReceipts = 0
	expired.DeveloperToolsSearchesInterested = 0
	expired.DeveloperToolsPostSelectionInterests = 0
	expired.DeveloperToolsPostSelectionSearches = 0
	expiredComparison, err := compareFunnelCheckpoint(checkpoint, &expired, attempts)
	if err != nil {
		t.Fatalf("active-state expiration was rejected: %v", err)
	}
	if expiredComparison.ActiveActionInterestReceiptsNetChange != -1 ||
		expiredComparison.PostSelectionInterestReceiptsNetChange != -1 ||
		expiredComparison.DeveloperToolsInterestReceiptsNetChange != -1 ||
		expiredComparison.DeveloperToolsPostSelectionInterestsNetChange != -1 ||
		expiredComparison.ExplicitPostSelectionInterestNetIncrease {
		t.Fatalf("expired active-state comparison = %#v", expiredComparison)
	}

	regressed := current
	regressed.MeaningfulSearchReceipts = 9
	regressed.MCPSearchReceipts = 4
	regressed.RESTSearchReceipts = 5
	if _, err := compareFunnelCheckpoint(checkpoint, &regressed, attempts); err == nil {
		t.Fatal("funnel comparison accepted a regressed durable counter")
	}

	invalid := current
	invalid.DeveloperToolsSearchReceipts = invalid.MeaningfulSearchReceipts + 1
	if _, err := compareFunnelCheckpoint(checkpoint, &invalid, attempts); err == nil {
		t.Fatal("funnel comparison accepted inconsistent counters")
	}
}

func assertNoForbiddenFunnelKeys(t *testing.T, value any) {
	t.Helper()
	forbidden := map[string]struct{}{
		"agent_id": {}, "principal_id": {}, "domain": {}, "query": {}, "prompt": {},
		"contact": {}, "search_id": {}, "provider_id": {}, "offer_id": {},
		"ticket_id": {}, "outcome_id": {}, "settlement_id": {},
	}
	var walk func(any)
	walk = func(current any) {
		switch typed := current.(type) {
		case map[string]any:
			for key, child := range typed {
				if _, blocked := forbidden[key]; blocked {
					t.Fatalf("funnel comparison exposed forbidden key %q", key)
				}
				walk(child)
			}
		case []any:
			for _, child := range typed {
				walk(child)
			}
		}
	}
	walk(value)
}

func TestLoadAttemptCheckpointFromEvidenceEnvelope(t *testing.T) {
	report := &models.PostSelectionActionInterestExperiment{
		Contract:         models.PostSelectionActionInterestExperimentContract,
		Since:            time.Date(2026, 8, 11, 9, 41, 28, 0, time.UTC),
		CheckedAt:        time.Date(2026, 8, 11, 10, 11, 43, 525805000, time.UTC),
		EligibleSurfaces: []string{"mcp", "rest"},
	}
	reportJSON, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	digest := sha256.Sum256(reportJSON)
	envelope := map[string]any{
		"contract": "external-evidence-v1",
		"reader_receipt": attemptCheckpointReceipt{
			Contract:          readContract,
			ReportSHA256:      hex.EncodeToString(digest[:]),
			CandidateRevision: "7ae09dff9274506d58f4ccdc318d85a89813d948",
			BinaryRevision:    "7ae09dff9274506d58f4ccdc318d85a89813d948",
			Report:            report,
		},
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	path := t.TempDir() + "/checkpoint.json"
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write checkpoint: %v", err)
	}
	loaded, err := loadAttemptCheckpoint(path)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if loaded.ReportSHA256 != hex.EncodeToString(digest[:]) || !loaded.Report.Since.Equal(report.Since) {
		t.Fatalf("loaded checkpoint = %#v", loaded)
	}
	t.Setenv("NHS_ATTEMPT_CHECKPOINT_B64", base64.StdEncoding.EncodeToString(raw))
	environmentLoaded, err := loadAttemptCheckpointEnvironment("NHS_ATTEMPT_CHECKPOINT_B64")
	if err != nil {
		t.Fatalf("load checkpoint environment: %v", err)
	}
	if environmentLoaded.ReportSHA256 != loaded.ReportSHA256 {
		t.Fatalf("environment checkpoint = %#v, want %#v", environmentLoaded, loaded)
	}
	if _, err := loadAttemptCheckpointEnvironment("DATABASE_URL"); err == nil {
		t.Fatal("checkpoint loader accepted an unrestricted environment name")
	}

	var tampered map[string]any
	if err := json.Unmarshal(raw, &tampered); err != nil {
		t.Fatalf("decode envelope for tamper: %v", err)
	}
	tamperedReceipt := tampered["reader_receipt"].(map[string]any)
	tamperedReceipt["report_sha256"] = strings.Repeat("0", 64)
	tamperedRaw, err := json.Marshal(tampered)
	if err != nil {
		t.Fatalf("marshal tampered checkpoint: %v", err)
	}
	if err := os.WriteFile(path, tamperedRaw, 0o600); err != nil {
		t.Fatalf("write tampered checkpoint: %v", err)
	}
	if _, err := loadAttemptCheckpoint(path); err == nil {
		t.Fatal("checkpoint loader accepted tampered evidence")
	}
}

func TestReadReceiptPrivacyShape(t *testing.T) {
	rate := 0.5
	receipt := readReceipt{
		Contract:          readContract,
		ReportSHA256:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		CandidateRevision: "0123456789abcdef0123456789abcdef01234567",
		BinaryRevision:    "0123456789abcdef0123456789abcdef01234567",
		Report: &models.PostSelectionActionInterestExperiment{
			Contract:                    models.PostSelectionActionInterestExperimentContract,
			Since:                       time.Date(2026, 8, 11, 9, 41, 28, 0, time.UTC),
			CheckedAt:                   time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC),
			EligibleSurfaces:            []string{"mcp", "rest"},
			MeaningfulSearchReceipts:    2,
			ResultSelections:            2,
			PostSelectionSearchReceipts: 1,
			PostSelectionConversionRate: &rate,
		},
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("marshal privacy receipt: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode privacy receipt: %v", err)
	}
	wantKeys := []string{
		"binary_revision", "candidate_revision",
		"contains_contact_data", "contains_identifiers", "contains_queries_or_prompts",
		"contract", "operator_affected_organic_rank", "operator_changed_commercial_state",
		"operator_contacted_provider", "report", "report_sha256",
	}
	gotKeys := make([]string, 0, len(decoded))
	for key := range decoded {
		gotKeys = append(gotKeys, key)
	}
	slicesSort(gotKeys)
	if !reflect.DeepEqual(gotKeys, wantKeys) {
		t.Fatalf("receipt keys = %v, want privacy-bounded %v", gotKeys, wantKeys)
	}
	for _, key := range []string{
		"contains_identifiers", "contains_queries_or_prompts", "contains_contact_data",
		"operator_contacted_provider", "operator_changed_commercial_state",
		"operator_affected_organic_rank",
	} {
		if value, ok := decoded[key].(bool); !ok || value {
			t.Fatalf("receipt %s = %#v, want false", key, decoded[key])
		}
	}
}

func slicesSort(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
