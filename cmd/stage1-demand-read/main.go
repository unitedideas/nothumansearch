// Command stage1-demand-read emits one privacy-bounded, read-only owner receipt
// for NHS Stage 1 readiness. It has no public route and never mutates NHS or
// provider state.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/unitedideas/nothumansearch/internal/database"
	"github.com/unitedideas/nothumansearch/internal/models"
)

const (
	stage1ReadContract = "nhs-stage1-demand-read-v1"
	stage1ReadTimeout  = 30 * time.Second
)

var releaseRevision = "development"

var stage1Targets = map[string]int{
	"meaningful_search_receipts":           100,
	"search_receipts_with_selection":       20,
	"search_receipts_with_action_interest": 10,
	"pilot_candidate_topic_receipts":       20,
	"observation_window_days":              14,
}

var allowedAttemptOutcomes = map[string]struct{}{
	"created": {}, "replayed": {}, "invalid_request": {}, "unavailable": {},
	"conflict": {}, "rate_limited": {}, "cross_origin": {},
	"store_unavailable": {}, "internal_error": {},
}

var allowedDemandTopics = map[string]struct{}{
	"payments": {}, "commerce": {}, "jobs": {}, "data": {}, "search": {},
	"weather": {}, "maps": {}, "email": {}, "messaging": {}, "image": {},
	"video": {}, "audio": {}, "documents": {}, "security": {}, "finance": {},
	"health": {}, "education": {}, "news": {}, "analytics": {}, "automation": {},
	"productivity": {}, "identity": {}, "storage": {}, "ai-tools": {},
	"developer-tools": {}, "other": {},
}

var allowedActionTypes = map[string]struct{}{
	"quote": {}, "trial": {}, "demo": {}, "booking": {},
	"application": {}, "signup": {}, "purchase": {},
}

type stage1ReadReceipt struct {
	Contract                       string                              `json:"contract"`
	Stage1ReportSHA256             string                              `json:"stage1_report_sha256"`
	AttemptFunnelSHA256            string                              `json:"attempt_funnel_sha256"`
	CandidateRevision              string                              `json:"candidate_revision"`
	BinaryRevision                 string                              `json:"binary_revision"`
	Stage1Demand                   *models.Stage1DemandProof           `json:"stage1_demand"`
	ActionInterestAttemptFunnel    *models.ActionInterestAttemptFunnel `json:"action_interest_attempt_funnel"`
	IndependentReadOnlySnapshots   bool                                `json:"independent_read_only_snapshots"`
	SearchesAreNotLeads            bool                                `json:"searches_are_not_leads"`
	ReadinessDoesNotAuthorizePilot bool                                `json:"readiness_does_not_authorize_stage2"`
	StrongestMechanismSelected     bool                                `json:"strongest_mechanism_selected"`
	ContainsIdentifiers            bool                                `json:"contains_identifiers"`
	ContainsQueriesOrPrompts       bool                                `json:"contains_queries_or_prompts"`
	ContainsContactData            bool                                `json:"contains_contact_data"`
	OperatorContactedProvider      bool                                `json:"operator_contacted_provider"`
	OperatorChangedCommercialState bool                                `json:"operator_changed_commercial_state"`
	OperatorAffectedOrganicRank    bool                                `json:"operator_affected_organic_rank"`
}

func main() {
	revision := flag.String("revision", "", "exact 40-character deployed commit")
	days := flag.Int("days", models.ActionInterestRetentionDays, "Stage 1 window in days")
	flag.Parse()

	candidate := strings.ToLower(strings.TrimSpace(*revision))
	compiled := strings.ToLower(strings.TrimSpace(releaseRevision))
	if !validRevision(candidate) || candidate != compiled {
		fail("candidate_revision_mismatch")
	}
	if *days < models.Stage1ReportMinimumDays || *days > models.ActionInterestRetentionDays {
		fail("invalid_stage1_window")
	}

	ctx, cancel := context.WithTimeout(context.Background(), stage1ReadTimeout)
	defer cancel()
	if err := database.ConnectWithReleaseRevisionContext(ctx, candidate); err != nil {
		fail("database_connection_failed")
	}
	defer func() {
		_ = database.DB.Close()
		database.DB = nil
	}()
	database.DB.SetMaxOpenConns(1)
	database.DB.SetMaxIdleConns(0)

	proof, err := models.GetStage1DemandProof(database.DB, *days)
	if err != nil {
		fail("stage1_read_failed")
	}
	if err := validateStage1Proof(proof, *days); err != nil {
		fail("stage1_proof_invalid")
	}
	funnel, err := models.GetActionInterestAttemptFunnel(database.DB, *days)
	if err != nil {
		fail("attempt_funnel_read_failed")
	}
	if err := validateAttemptFunnel(funnel, *days); err != nil {
		fail("attempt_funnel_invalid")
	}

	proofJSON, err := json.Marshal(proof)
	if err != nil {
		fail("stage1_encoding_failed")
	}
	funnelJSON, err := json.Marshal(funnel)
	if err != nil {
		fail("attempt_funnel_encoding_failed")
	}
	proofDigest := sha256.Sum256(proofJSON)
	funnelDigest := sha256.Sum256(funnelJSON)
	receipt := stage1ReadReceipt{
		Contract:                       stage1ReadContract,
		Stage1ReportSHA256:             hex.EncodeToString(proofDigest[:]),
		AttemptFunnelSHA256:            hex.EncodeToString(funnelDigest[:]),
		CandidateRevision:              candidate,
		BinaryRevision:                 compiled,
		Stage1Demand:                   proof,
		ActionInterestAttemptFunnel:    funnel,
		IndependentReadOnlySnapshots:   true,
		SearchesAreNotLeads:            true,
		ReadinessDoesNotAuthorizePilot: true,
		StrongestMechanismSelected:     false,
		ContainsIdentifiers:            false,
		ContainsQueriesOrPrompts:       false,
		ContainsContactData:            false,
		OperatorContactedProvider:      false,
		OperatorChangedCommercialState: false,
		OperatorAffectedOrganicRank:    false,
	}
	if err := json.NewEncoder(os.Stdout).Encode(receipt); err != nil {
		os.Exit(1)
	}
}

func validateStage1Proof(proof *models.Stage1DemandProof, days int) error {
	const secondsPerDay int64 = 24 * 60 * 60
	if proof == nil || proof.Days != days || proof.RetentionDays != models.ActionInterestRetentionDays ||
		proof.AsOf.IsZero() || proof.Stage1StartedAt.IsZero() || proof.Stage1StartedAt.After(proof.AsOf) ||
		!proof.Stage1EpochEnforced || !proof.SyntheticExcluded ||
		!proof.CountsAreReceiptsNotUniqueAgents || proof.CommercialProof ||
		!reflect.DeepEqual(proof.EligibleSurfaces, []string{"mcp", "rest"}) ||
		!reflect.DeepEqual(proof.Targets, stage1Targets) {
		return fmt.Errorf("Stage 1 contract invalid")
	}
	if proof.MeaningfulSearchReceipts < 0 || proof.ResultSelections < 0 ||
		proof.SearchReceiptsWithSelection < 0 || proof.ActionInterestReceipts < 0 ||
		proof.SearchReceiptsWithActionInterest < 0 || proof.DistinctInterestDomains < 0 ||
		proof.SearchReceiptsWithSelection > proof.MeaningfulSearchReceipts ||
		proof.ResultSelections < proof.SearchReceiptsWithSelection ||
		proof.SearchReceiptsWithActionInterest > proof.MeaningfulSearchReceipts ||
		proof.ActionInterestReceipts < proof.SearchReceiptsWithActionInterest ||
		proof.ActionInterestReceipts < proof.DistinctInterestDomains ||
		proof.BucketReceiptThreshold != 20 || !proof.TopicBucketsMayOverlap ||
		proof.ObservationWindowDays != 14 || proof.ObservationSpanSeconds < 0 ||
		proof.ObservationSpanDays != int(proof.ObservationSpanSeconds/secondsPerDay) ||
		proof.ObservationWindowMet != (proof.ObservationSpanSeconds >= 14*secondsPerDay) {
		return fmt.Errorf("Stage 1 counters invalid")
	}
	expectedMet := map[string]bool{
		"meaningful_search_receipts":           proof.MeaningfulSearchReceipts >= stage1Targets["meaningful_search_receipts"],
		"search_receipts_with_selection":       proof.SearchReceiptsWithSelection >= stage1Targets["search_receipts_with_selection"],
		"search_receipts_with_action_interest": proof.SearchReceiptsWithActionInterest >= stage1Targets["search_receipts_with_action_interest"],
		"pilot_candidate_topic_receipts":       proof.PilotCandidateTopicAvailable,
		"observation_window_days":              proof.ObservationWindowMet,
	}
	ready := true
	for _, met := range expectedMet {
		ready = ready && met
	}
	if !reflect.DeepEqual(proof.TargetsMet, expectedMet) || proof.Stage1Ready != ready {
		return fmt.Errorf("Stage 1 readiness invalid")
	}
	demandCounts := make(map[string]int, len(proof.DemandTopics))
	for _, bucket := range proof.DemandTopics {
		if _, ok := allowedDemandTopics[bucket.Value]; !ok || bucket.ReceiptCount < proof.BucketReceiptThreshold ||
			bucket.ReceiptCount > proof.MeaningfulSearchReceipts {
			return fmt.Errorf("demand bucket invalid")
		}
		if _, duplicate := demandCounts[bucket.Value]; duplicate {
			return fmt.Errorf("demand bucket duplicated")
		}
		demandCounts[bucket.Value] = bucket.ReceiptCount
	}
	if proof.PilotCandidateTopicAvailable != (len(proof.PilotCandidateTopics) > 0) {
		return fmt.Errorf("candidate topic availability invalid")
	}
	candidateSeen := make(map[string]struct{}, len(proof.PilotCandidateTopics))
	for _, bucket := range proof.PilotCandidateTopics {
		if _, ok := allowedDemandTopics[bucket.Value]; !ok || bucket.Value == "other" ||
			bucket.ReceiptCount < stage1Targets["pilot_candidate_topic_receipts"] ||
			demandCounts[bucket.Value] != bucket.ReceiptCount {
			return fmt.Errorf("candidate topic invalid")
		}
		if _, duplicate := candidateSeen[bucket.Value]; duplicate {
			return fmt.Errorf("candidate topic duplicated")
		}
		candidateSeen[bucket.Value] = struct{}{}
	}
	actionSeen := make(map[string]struct{}, len(proof.ActionTypes))
	for _, bucket := range proof.ActionTypes {
		if _, ok := allowedActionTypes[bucket.Value]; !ok || bucket.ReceiptCount < proof.BucketReceiptThreshold ||
			bucket.ReceiptCount > proof.SearchReceiptsWithActionInterest {
			return fmt.Errorf("action type bucket invalid")
		}
		if _, duplicate := actionSeen[bucket.Value]; duplicate {
			return fmt.Errorf("action type bucket duplicated")
		}
		actionSeen[bucket.Value] = struct{}{}
	}
	return nil
}

func validateAttemptFunnel(funnel *models.ActionInterestAttemptFunnel, days int) error {
	if funnel == nil || funnel.Days != days || funnel.AsOf.IsZero() ||
		!funnel.CountsAreAttemptsNotUniqueAgents || funnel.ContainsRequestCoordinates ||
		funnel.CommercialProof || funnel.TotalAttempts < 0 {
		return fmt.Errorf("attempt funnel contract invalid")
	}
	var total int64
	seen := make(map[string]struct{}, len(funnel.Outcomes))
	for _, bucket := range funnel.Outcomes {
		if (bucket.Surface != "mcp" && bucket.Surface != "rest") || bucket.AttemptCount < 0 {
			return fmt.Errorf("attempt bucket invalid")
		}
		if _, ok := allowedAttemptOutcomes[bucket.Outcome]; !ok {
			return fmt.Errorf("attempt outcome invalid")
		}
		key := bucket.Surface + ":" + bucket.Outcome
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("attempt bucket duplicated")
		}
		seen[key] = struct{}{}
		total += bucket.AttemptCount
	}
	if total != funnel.TotalAttempts {
		return fmt.Errorf("attempt total invalid")
	}
	return nil
}

func validRevision(value string) bool {
	if len(value) != 40 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && value == strings.ToLower(value)
}

func fail(code string) {
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
		"contract": stage1ReadContract,
		"ok":       false,
		"error":    code,
	})
	os.Exit(1)
}
