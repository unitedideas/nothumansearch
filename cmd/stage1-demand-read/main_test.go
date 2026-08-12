package main

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
)

func validStage1Fixture() *models.Stage1DemandProof {
	asOf := time.Date(2026, 8, 12, 1, 30, 0, 0, time.UTC)
	return &models.Stage1DemandProof{
		Days:                             30,
		RetentionDays:                    30,
		AsOf:                             asOf,
		Stage1StartedAt:                  asOf.Add(-15 * 24 * time.Hour),
		Stage1EpochEnforced:              true,
		SyntheticExcluded:                true,
		EligibleSurfaces:                 []string{"mcp", "rest"},
		CountsAreReceiptsNotUniqueAgents: true,
		CommercialProof:                  false,
		MeaningfulSearchReceipts:         120,
		ResultSelections:                 22,
		SearchReceiptsWithSelection:      20,
		ActionInterestReceipts:           10,
		SearchReceiptsWithActionInterest: 10,
		DistinctInterestDomains:          3,
		BucketReceiptThreshold:           20,
		TopicBucketsMayOverlap:           true,
		DemandTopics:                     []models.Stage1DemandBucket{{Value: "developer-tools", ReceiptCount: 25}},
		PilotCandidateTopics:             []models.Stage1DemandBucket{{Value: "developer-tools", ReceiptCount: 25}},
		PilotCandidateTopicAvailable:     true,
		ObservationWindowDays:            14,
		ObservationSpanSeconds:           15 * 24 * 60 * 60,
		ObservationSpanDays:              15,
		ObservationWindowMet:             true,
		Stage1Ready:                      true,
		Targets:                          cloneIntMap(stage1Targets),
		TargetsMet: map[string]bool{
			"meaningful_search_receipts":           true,
			"search_receipts_with_selection":       true,
			"search_receipts_with_action_interest": true,
			"pilot_candidate_topic_receipts":       true,
			"observation_window_days":              true,
		},
	}
}

func cloneIntMap(input map[string]int) map[string]int {
	output := make(map[string]int, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func TestValidateStage1Proof(t *testing.T) {
	if err := validateStage1Proof(validStage1Fixture(), 30); err != nil {
		t.Fatalf("valid Stage 1 proof rejected: %v", err)
	}
	notReady := validStage1Fixture()
	notReady.ResultSelections = 0
	notReady.SearchReceiptsWithSelection = 0
	notReady.ActionInterestReceipts = 0
	notReady.SearchReceiptsWithActionInterest = 0
	notReady.DistinctInterestDomains = 0
	notReady.PilotCandidateTopics = nil
	notReady.PilotCandidateTopicAvailable = false
	notReady.ObservationSpanSeconds = 6 * 24 * 60 * 60
	notReady.ObservationSpanDays = 6
	notReady.ObservationWindowMet = false
	notReady.Stage1Ready = false
	notReady.TargetsMet["search_receipts_with_selection"] = false
	notReady.TargetsMet["search_receipts_with_action_interest"] = false
	notReady.TargetsMet["pilot_candidate_topic_receipts"] = false
	notReady.TargetsMet["observation_window_days"] = false
	if err := validateStage1Proof(notReady, 30); err != nil {
		t.Fatalf("valid immature Stage 1 proof rejected: %v", err)
	}

	tests := map[string]func(*models.Stage1DemandProof){
		"synthetic not excluded":   func(proof *models.Stage1DemandProof) { proof.SyntheticExcluded = false },
		"sold as commercial proof": func(proof *models.Stage1DemandProof) { proof.CommercialProof = true },
		"impossible selection": func(proof *models.Stage1DemandProof) {
			proof.SearchReceiptsWithSelection = proof.MeaningfulSearchReceipts + 1
		},
		"wrong target":       func(proof *models.Stage1DemandProof) { proof.Targets["meaningful_search_receipts"] = 99 },
		"readiness mismatch": func(proof *models.Stage1DemandProof) { proof.Stage1Ready = false },
		"other candidate":    func(proof *models.Stage1DemandProof) { proof.PilotCandidateTopics[0].Value = "other" },
		"unknown demand topic": func(proof *models.Stage1DemandProof) {
			proof.DemandTopics[0].Value = "private-topic"
		},
		"candidate count mismatch": func(proof *models.Stage1DemandProof) {
			proof.PilotCandidateTopics[0].ReceiptCount = 24
		},
		"short observation": func(proof *models.Stage1DemandProof) {
			proof.ObservationSpanSeconds = 13 * 24 * 60 * 60
			proof.ObservationSpanDays = 13
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			proof := validStage1Fixture()
			mutate(proof)
			if err := validateStage1Proof(proof, 30); err == nil {
				t.Fatal("invalid Stage 1 proof accepted")
			}
		})
	}
}

func TestValidateAttemptFunnel(t *testing.T) {
	funnel := &models.ActionInterestAttemptFunnel{
		Days:                             30,
		AsOf:                             time.Date(2026, 8, 12, 1, 30, 1, 0, time.UTC),
		CountsAreAttemptsNotUniqueAgents: true,
		ContainsRequestCoordinates:       false,
		CommercialProof:                  false,
		TotalAttempts:                    2,
		Outcomes: []models.ActionInterestAttemptBucket{
			{Surface: "mcp", Outcome: "unavailable", AttemptCount: 2},
		},
	}
	if err := validateAttemptFunnel(funnel, 30); err != nil {
		t.Fatalf("valid attempt funnel rejected: %v", err)
	}
	invalid := *funnel
	invalid.TotalAttempts = 3
	if err := validateAttemptFunnel(&invalid, 30); err == nil {
		t.Fatal("attempt funnel accepted inconsistent total")
	}
	invalid = *funnel
	invalid.Outcomes = []models.ActionInterestAttemptBucket{{Surface: "mcp", Outcome: "lead", AttemptCount: 2}}
	if err := validateAttemptFunnel(&invalid, 30); err == nil {
		t.Fatal("attempt funnel accepted forbidden outcome")
	}
}

func TestStage1ReceiptContainsNoPrivateCoordinates(t *testing.T) {
	receipt := stage1ReadReceipt{
		Contract:                       stage1ReadContract,
		Stage1ReportSHA256:             "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		AttemptFunnelSHA256:            "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		CandidateRevision:              "0123456789abcdef0123456789abcdef01234567",
		BinaryRevision:                 "0123456789abcdef0123456789abcdef01234567",
		Stage1Demand:                   validStage1Fixture(),
		IndependentReadOnlySnapshots:   true,
		SearchesAreNotLeads:            true,
		ReadinessDoesNotAuthorizePilot: true,
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("marshal receipt: %v", err)
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode receipt: %v", err)
	}
	forbidden := map[string]struct{}{
		"agent_id": {}, "principal_id": {}, "provider_id": {}, "search_id": {},
		"domain": {}, "query": {}, "prompt": {}, "contact": {}, "email": {},
		"offer_id": {}, "ticket_id": {}, "outcome_id": {}, "settlement_id": {},
	}
	var walk func(any)
	walk = func(value any) {
		switch typed := value.(type) {
		case map[string]any:
			for key, child := range typed {
				if _, blocked := forbidden[key]; blocked {
					t.Fatalf("receipt exposed forbidden key %q", key)
				}
				walk(child)
			}
		case []any:
			for _, child := range typed {
				walk(child)
			}
		}
	}
	walk(decoded)

	if !reflect.DeepEqual(receipt.Stage1Demand.Targets, stage1Targets) {
		t.Fatal("fixture targets drifted")
	}
}

func TestValidRevision(t *testing.T) {
	valid := "0123456789abcdef0123456789abcdef01234567"
	if !validRevision(valid) || validRevision(valid[:39]) ||
		validRevision("0123456789ABCDEF0123456789ABCDEF01234567") {
		t.Fatal("revision validation drifted")
	}
}
