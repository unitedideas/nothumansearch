package main

import (
	"encoding/json"
	"reflect"
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
