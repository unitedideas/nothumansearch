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
