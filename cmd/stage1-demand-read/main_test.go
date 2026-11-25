package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
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

func validAttemptFixture(asOf time.Time) *models.ActionInterestAttemptFunnel {
	return &models.ActionInterestAttemptFunnel{
		Days:                             30,
		AsOf:                             asOf,
		CountsAreAttemptsNotUniqueAgents: true,
		ContainsRequestCoordinates:       false,
		CommercialProof:                  false,
		TotalAttempts:                    31,
		Outcomes: []models.ActionInterestAttemptBucket{
			{Surface: "mcp", Outcome: "invalid_request", AttemptCount: 1},
			{Surface: "rest", Outcome: "unavailable", AttemptCount: 30},
		},
	}
}

func checkpointReceipt(t *testing.T, proof *models.Stage1DemandProof, funnel *models.ActionInterestAttemptFunnel) *stage1CheckpointReceipt {
	t.Helper()
	proofJSON, err := json.Marshal(proof)
	if err != nil {
		t.Fatal(err)
	}
	funnelJSON, err := json.Marshal(funnel)
	if err != nil {
		t.Fatal(err)
	}
	proofDigest := sha256.Sum256(proofJSON)
	funnelDigest := sha256.Sum256(funnelJSON)
	return &stage1CheckpointReceipt{
		Contract:                    stage1ReadContract,
		Stage1ReportSHA256:          hex.EncodeToString(proofDigest[:]),
		AttemptFunnelSHA256:         hex.EncodeToString(funnelDigest[:]),
		CandidateRevision:           "0123456789abcdef0123456789abcdef01234567",
		BinaryRevision:              "0123456789abcdef0123456789abcdef01234567",
		Stage1Demand:                proof,
		ActionInterestAttemptFunnel: funnel,
	}
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
	funnel := validAttemptFixture(time.Date(2026, 8, 12, 1, 30, 1, 0, time.UTC))
	if err := validateAttemptFunnel(funnel, 30); err != nil {
		t.Fatalf("valid attempt funnel rejected: %v", err)
	}
	invalid := *funnel
	invalid.TotalAttempts = 3
	if err := validateAttemptFunnel(&invalid, 30); err == nil {
		t.Fatal("attempt funnel accepted inconsistent total")
	}
	invalid = *funnel
	invalid.Outcomes = []models.ActionInterestAttemptBucket{{Surface: "mcp", Outcome: "lead", AttemptCount: 31}}
	if err := validateAttemptFunnel(&invalid, 30); err == nil {
		t.Fatal("attempt funnel accepted forbidden outcome")
	}
}

func TestParseStage1CheckpointEnvelopeAndDigest(t *testing.T) {
	proof := validStage1Fixture()
	funnel := validAttemptFixture(proof.AsOf.Add(time.Second))
	receipt := checkpointReceipt(t, proof, funnel)
	envelope := map[string]any{
		"contract":       "nhs-stage1-demand-evidence-v1",
		"reader_receipt": receipt,
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parseStage1Checkpoint(raw)
	if err != nil {
		t.Fatalf("valid evidence envelope rejected: %v", err)
	}
	if parsed.Stage1ReportSHA256 != receipt.Stage1ReportSHA256 ||
		parsed.AttemptFunnelSHA256 != receipt.AttemptFunnelSHA256 {
		t.Fatalf("parsed checkpoint drifted: %#v", parsed)
	}

	encoded := base64.StdEncoding.EncodeToString(raw)
	t.Setenv("NHS_STAGE1_CHECKPOINT_B64", encoded)
	fromEnvironment, err := loadStage1CheckpointEnvironment("NHS_STAGE1_CHECKPOINT_B64")
	if err != nil || fromEnvironment.Stage1ReportSHA256 != receipt.Stage1ReportSHA256 {
		t.Fatalf("environment checkpoint = %#v, %v", fromEnvironment, err)
	}
	if _, err := loadStage1CheckpointEnvironment("OTHER_CHECKPOINT"); err == nil {
		t.Fatal("arbitrary checkpoint environment accepted")
	}

	proof.MeaningfulSearchReceipts++
	tampered, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseStage1Checkpoint(tampered); err == nil {
		t.Fatal("tampered checkpoint digest accepted")
	}
	os.Unsetenv("NHS_STAGE1_CHECKPOINT_B64")
}

func TestCompareStage1CheckpointUsesRollingNetChanges(t *testing.T) {
	checkpointProof := validStage1Fixture()
	checkpointProof.ResultSelections = 19
	checkpointProof.SearchReceiptsWithSelection = 19
	checkpointProof.ActionInterestReceipts = 9
	checkpointProof.SearchReceiptsWithActionInterest = 9
	checkpointProof.DistinctInterestDomains = 2
	checkpointProof.Stage1Ready = false
	checkpointProof.TargetsMet["search_receipts_with_selection"] = false
	checkpointProof.TargetsMet["search_receipts_with_action_interest"] = false
	checkpointFunnel := validAttemptFixture(checkpointProof.AsOf.Add(time.Second))
	checkpoint := checkpointReceipt(t, checkpointProof, checkpointFunnel)

	currentProof := validStage1Fixture()
	currentProof.AsOf = checkpointProof.AsOf.Add(time.Hour)
	currentProof.MeaningfulSearchReceipts = 125
	currentProof.ObservationSpanSeconds += 3600
	currentFunnel := validAttemptFixture(checkpointFunnel.AsOf.Add(time.Hour))
	currentFunnel.Outcomes[0].AttemptCount = 3
	currentFunnel.Outcomes[1].AttemptCount = 29
	currentFunnel.TotalAttempts = 32

	comparison, err := compareStage1Checkpoint(checkpoint, currentProof, currentFunnel)
	if err != nil {
		t.Fatalf("compare checkpoint: %v", err)
	}
	if comparison.MeaningfulSearchReceiptsNetChange != 5 ||
		comparison.SearchReceiptsWithSelectionNetChange != 1 ||
		comparison.SearchesWithActionInterestNetChange != 1 ||
		!comparison.DiscoveryReceiptNetIncrease || !comparison.SelectionReceiptNetIncrease ||
		!comparison.ExplicitInterestReceiptNetIncrease || !comparison.MCPInvalidAttemptNetIncrease ||
		!comparison.Stage1BecameReady || !comparison.CountsAreRollingWindowNetChanges ||
		!comparison.NetChangesAreNotCreatedEventCounts || !comparison.SearchesAreNotLeads ||
		comparison.StrongestMechanismSelected || comparison.CommercialProof ||
		comparison.ContainsIdentifiers || comparison.ContainsQueriesOrPrompts ||
		comparison.ContainsContactData || comparison.ContainsRequestCoordinates {
		t.Fatalf("checkpoint comparison = %#v", comparison)
	}
	if !reflect.DeepEqual(comparison.AttemptBucketNetChanges, []attemptBucketNetChange{
		{Surface: "mcp", Outcome: "invalid_request", NetChange: 2},
		{Surface: "rest", Outcome: "unavailable", NetChange: -1},
	}) {
		t.Fatalf("attempt net changes = %#v", comparison.AttemptBucketNetChanges)
	}

	expiredProof := *currentProof
	expiredProof.AsOf = currentProof.AsOf.Add(time.Hour)
	expiredProof.ResultSelections = 19
	expiredProof.SearchReceiptsWithSelection = 19
	expiredProof.ActionInterestReceipts = 9
	expiredProof.SearchReceiptsWithActionInterest = 9
	expiredProof.DistinctInterestDomains = 2
	expiredProof.Stage1Ready = false
	expiredProof.TargetsMet = map[string]bool{
		"meaningful_search_receipts":           true,
		"search_receipts_with_selection":       false,
		"search_receipts_with_action_interest": false,
		"pilot_candidate_topic_receipts":       true,
		"observation_window_days":              true,
	}
	expiredFunnel := *currentFunnel
	expiredFunnel.AsOf = currentFunnel.AsOf.Add(time.Hour)
	currentCheckpoint := checkpointReceipt(t, currentProof, currentFunnel)
	expiredComparison, err := compareStage1Checkpoint(currentCheckpoint, &expiredProof, &expiredFunnel)
	if err != nil {
		t.Fatalf("rolling-window expiration rejected: %v", err)
	}
	if expiredComparison.SearchReceiptsWithSelectionNetChange != -1 ||
		expiredComparison.SearchesWithActionInterestNetChange != -1 ||
		expiredComparison.SelectionReceiptNetIncrease ||
		expiredComparison.ExplicitInterestReceiptNetIncrease {
		t.Fatalf("expired comparison = %#v", expiredComparison)
	}
}

func TestCompareStage1CheckpointRejectsIncompatibleWindows(t *testing.T) {
	checkpointProof := validStage1Fixture()
	checkpointFunnel := validAttemptFixture(checkpointProof.AsOf.Add(time.Second))
	checkpoint := checkpointReceipt(t, checkpointProof, checkpointFunnel)

	tests := map[string]func(*models.Stage1DemandProof, *models.ActionInterestAttemptFunnel){
		"different Stage 1 epoch": func(proof *models.Stage1DemandProof, _ *models.ActionInterestAttemptFunnel) {
			proof.Stage1StartedAt = proof.Stage1StartedAt.Add(time.Second)
		},
		"non-increasing proof timestamp": func(proof *models.Stage1DemandProof, _ *models.ActionInterestAttemptFunnel) {
			proof.AsOf = checkpointProof.AsOf
		},
		"non-increasing funnel timestamp": func(_ *models.Stage1DemandProof, funnel *models.ActionInterestAttemptFunnel) {
			funnel.AsOf = checkpointFunnel.AsOf
		},
		"different rolling window": func(proof *models.Stage1DemandProof, funnel *models.ActionInterestAttemptFunnel) {
			proof.Days = 29
			funnel.Days = 29
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			proof := validStage1Fixture()
			proof.AsOf = checkpointProof.AsOf.Add(time.Hour)
			funnel := validAttemptFixture(checkpointFunnel.AsOf.Add(time.Hour))
			mutate(proof, funnel)
			if _, err := compareStage1Checkpoint(checkpoint, proof, funnel); err == nil {
				t.Fatal("incompatible checkpoint window accepted")
			}
		})
	}
}

func TestStage1ReceiptContainsNoPrivateCoordinates(t *testing.T) {
	checkpointProof := validStage1Fixture()
	checkpointProof.AsOf = checkpointProof.AsOf.Add(-time.Hour)
	checkpointFunnel := validAttemptFixture(checkpointProof.AsOf.Add(time.Second))
	comparison, err := compareStage1Checkpoint(
		checkpointReceipt(t, checkpointProof, checkpointFunnel),
		validStage1Fixture(),
		validAttemptFixture(checkpointFunnel.AsOf.Add(time.Hour)),
	)
	if err != nil {
		t.Fatalf("build privacy fixture comparison: %v", err)
	}
	receipt := stage1ReadReceipt{
		Contract:                       stage1ReadContract,
		Stage1ReportSHA256:             "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		AttemptFunnelSHA256:            "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		CandidateRevision:              "0123456789abcdef0123456789abcdef01234567",
		BinaryRevision:                 "0123456789abcdef0123456789abcdef01234567",
		Stage1Demand:                   validStage1Fixture(),
		CheckpointComparison:           comparison,
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
