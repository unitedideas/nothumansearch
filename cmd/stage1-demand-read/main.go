// Command stage1-demand-read emits one privacy-bounded, read-only owner receipt
// for NHS Stage 1 readiness. It has no public route and never mutates NHS or
// provider state.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/unitedideas/nothumansearch/internal/database"
	"github.com/unitedideas/nothumansearch/internal/models"
)

const (
	stage1ReadContract          = "nhs-stage1-demand-read-v1"
	stage1CheckpointContract    = "nhs-stage1-demand-checkpoint-comparison-v1"
	stage1ReadTimeout           = 30 * time.Second
	stage1CheckpointMaximumSize = 128 * 1024
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
	CheckpointComparison           *stage1CheckpointComparison         `json:"checkpoint_comparison,omitempty"`
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

type stage1CheckpointReceipt struct {
	Contract                    string                              `json:"contract"`
	Stage1ReportSHA256          string                              `json:"stage1_report_sha256"`
	AttemptFunnelSHA256         string                              `json:"attempt_funnel_sha256"`
	CandidateRevision           string                              `json:"candidate_revision"`
	BinaryRevision              string                              `json:"binary_revision"`
	Stage1Demand                *models.Stage1DemandProof           `json:"stage1_demand"`
	ActionInterestAttemptFunnel *models.ActionInterestAttemptFunnel `json:"action_interest_attempt_funnel"`
}

type attemptBucketNetChange struct {
	Surface   string `json:"surface"`
	Outcome   string `json:"outcome"`
	NetChange int64  `json:"net_change"`
}

type stage1CheckpointComparison struct {
	Contract                             string                   `json:"contract"`
	CheckpointStage1ReportSHA256         string                   `json:"checkpoint_stage1_report_sha256"`
	CheckpointAttemptFunnelSHA256        string                   `json:"checkpoint_attempt_funnel_sha256"`
	CheckpointRevision                   string                   `json:"checkpoint_revision"`
	CheckpointStage1AsOf                 time.Time                `json:"checkpoint_stage1_as_of"`
	CurrentStage1AsOf                    time.Time                `json:"current_stage1_as_of"`
	CheckpointAttemptFunnelAsOf          time.Time                `json:"checkpoint_attempt_funnel_as_of"`
	CurrentAttemptFunnelAsOf             time.Time                `json:"current_attempt_funnel_as_of"`
	MeaningfulSearchReceiptsNetChange    int                      `json:"meaningful_search_receipts_net_change"`
	ResultSelectionsNetChange            int                      `json:"result_selections_net_change"`
	SearchReceiptsWithSelectionNetChange int                      `json:"search_receipts_with_selection_net_change"`
	ActionInterestReceiptsNetChange      int                      `json:"action_interest_receipts_net_change"`
	SearchesWithActionInterestNetChange  int                      `json:"search_receipts_with_action_interest_net_change"`
	DistinctInterestDomainsNetChange     int                      `json:"distinct_interest_domains_net_change"`
	ObservationSpanSecondsNetChange      int64                    `json:"observation_span_seconds_net_change"`
	AttemptBucketNetChanges              []attemptBucketNetChange `json:"attempt_bucket_net_changes"`
	CountsAreRollingWindowNetChanges     bool                     `json:"counts_are_rolling_window_net_changes"`
	NetChangesAreNotCreatedEventCounts   bool                     `json:"net_changes_are_not_created_event_counts"`
	SearchesAreNotLeads                  bool                     `json:"searches_are_not_leads"`
	DiscoveryReceiptNetIncrease          bool                     `json:"discovery_receipt_net_increase"`
	SelectionReceiptNetIncrease          bool                     `json:"selection_receipt_net_increase"`
	ExplicitInterestReceiptNetIncrease   bool                     `json:"explicit_interest_receipt_net_increase"`
	MCPInvalidAttemptNetIncrease         bool                     `json:"mcp_invalid_attempt_net_increase"`
	Stage1BecameReady                    bool                     `json:"stage1_became_ready"`
	ReadinessDoesNotAuthorizePilot       bool                     `json:"readiness_does_not_authorize_stage2"`
	StrongestMechanismSelected           bool                     `json:"strongest_mechanism_selected"`
	CommercialProof                      bool                     `json:"commercial_proof"`
	ContainsIdentifiers                  bool                     `json:"contains_identifiers"`
	ContainsQueriesOrPrompts             bool                     `json:"contains_queries_or_prompts"`
	ContainsContactData                  bool                     `json:"contains_contact_data"`
	ContainsRequestCoordinates           bool                     `json:"contains_request_coordinates"`
}

func main() {
	revision := flag.String("revision", "", "exact 40-character deployed commit")
	days := flag.Int("days", models.ActionInterestRetentionDays, "Stage 1 window in days")
	checkpointPath := flag.String("checkpoint", "", "optional prior Stage 1 receipt or evidence envelope")
	checkpointEnv := flag.String("checkpoint-base64-env", "", "optional environment name containing a base64 prior Stage 1 receipt")
	flag.Parse()

	candidate := strings.ToLower(strings.TrimSpace(*revision))
	compiled := strings.ToLower(strings.TrimSpace(releaseRevision))
	if !validRevision(candidate) || candidate != compiled {
		fail("candidate_revision_mismatch")
	}
	if *days < models.Stage1ReportMinimumDays || *days > models.ActionInterestRetentionDays {
		fail("invalid_stage1_window")
	}
	if strings.TrimSpace(*checkpointPath) != "" && strings.TrimSpace(*checkpointEnv) != "" {
		fail("multiple_checkpoints")
	}
	var checkpoint *stage1CheckpointReceipt
	var err error
	if strings.TrimSpace(*checkpointPath) != "" {
		checkpoint, err = loadStage1Checkpoint(strings.TrimSpace(*checkpointPath))
	} else if strings.TrimSpace(*checkpointEnv) != "" {
		checkpoint, err = loadStage1CheckpointEnvironment(strings.TrimSpace(*checkpointEnv))
	}
	if err != nil {
		fail("invalid_checkpoint")
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
	var comparison *stage1CheckpointComparison
	if checkpoint != nil {
		comparison, err = compareStage1Checkpoint(checkpoint, proof, funnel)
		if err != nil {
			fail("checkpoint_comparison_failed")
		}
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
		CheckpointComparison:           comparison,
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

func loadStage1Checkpoint(path string) (*stage1CheckpointReceipt, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, stage1CheckpointMaximumSize+1))
	if err != nil || len(raw) == 0 || len(raw) > stage1CheckpointMaximumSize {
		return nil, fmt.Errorf("checkpoint size invalid")
	}
	return parseStage1Checkpoint(raw)
}

func loadStage1CheckpointEnvironment(name string) (*stage1CheckpointReceipt, error) {
	if name != "NHS_STAGE1_CHECKPOINT_B64" {
		return nil, fmt.Errorf("checkpoint environment name invalid")
	}
	encoded := strings.TrimSpace(os.Getenv(name))
	if encoded == "" || len(encoded) > base64.StdEncoding.EncodedLen(stage1CheckpointMaximumSize) {
		return nil, fmt.Errorf("checkpoint environment size invalid")
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(raw) == 0 || len(raw) > stage1CheckpointMaximumSize {
		return nil, fmt.Errorf("checkpoint environment encoding invalid")
	}
	return parseStage1Checkpoint(raw)
}

func parseStage1Checkpoint(raw []byte) (*stage1CheckpointReceipt, error) {
	var envelope struct {
		ReaderReceipt json.RawMessage `json:"reader_receipt"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	receiptRaw := raw
	if len(envelope.ReaderReceipt) > 0 && string(envelope.ReaderReceipt) != "null" {
		receiptRaw = envelope.ReaderReceipt
	}
	var receipt stage1CheckpointReceipt
	if err := json.Unmarshal(receiptRaw, &receipt); err != nil {
		return nil, err
	}
	if receipt.Contract != stage1ReadContract || receipt.Stage1Demand == nil ||
		receipt.ActionInterestAttemptFunnel == nil || !validRevision(receipt.CandidateRevision) ||
		receipt.BinaryRevision != receipt.CandidateRevision ||
		len(receipt.Stage1ReportSHA256) != sha256.Size*2 ||
		len(receipt.AttemptFunnelSHA256) != sha256.Size*2 ||
		receipt.Stage1Demand.Days != receipt.ActionInterestAttemptFunnel.Days {
		return nil, fmt.Errorf("checkpoint contract invalid")
	}
	if err := validateStage1Proof(receipt.Stage1Demand, receipt.Stage1Demand.Days); err != nil {
		return nil, err
	}
	if err := validateAttemptFunnel(receipt.ActionInterestAttemptFunnel, receipt.Stage1Demand.Days); err != nil {
		return nil, err
	}
	proofJSON, err := json.Marshal(receipt.Stage1Demand)
	if err != nil {
		return nil, err
	}
	funnelJSON, err := json.Marshal(receipt.ActionInterestAttemptFunnel)
	if err != nil {
		return nil, err
	}
	proofDigest := sha256.Sum256(proofJSON)
	funnelDigest := sha256.Sum256(funnelJSON)
	if receipt.Stage1ReportSHA256 != hex.EncodeToString(proofDigest[:]) ||
		receipt.AttemptFunnelSHA256 != hex.EncodeToString(funnelDigest[:]) {
		return nil, fmt.Errorf("checkpoint digest invalid")
	}
	return &receipt, nil
}

func compareStage1Checkpoint(
	checkpoint *stage1CheckpointReceipt,
	currentProof *models.Stage1DemandProof,
	currentFunnel *models.ActionInterestAttemptFunnel,
) (*stage1CheckpointComparison, error) {
	if checkpoint == nil || checkpoint.Stage1Demand == nil ||
		checkpoint.ActionInterestAttemptFunnel == nil || currentProof == nil || currentFunnel == nil ||
		checkpoint.Stage1Demand.Days != currentProof.Days ||
		checkpoint.ActionInterestAttemptFunnel.Days != currentFunnel.Days ||
		!checkpoint.Stage1Demand.Stage1StartedAt.Equal(currentProof.Stage1StartedAt) ||
		!checkpoint.Stage1Demand.AsOf.Before(currentProof.AsOf) ||
		!checkpoint.ActionInterestAttemptFunnel.AsOf.Before(currentFunnel.AsOf) {
		return nil, fmt.Errorf("checkpoint window invalid")
	}
	if err := validateStage1Proof(currentProof, currentProof.Days); err != nil {
		return nil, err
	}
	if err := validateAttemptFunnel(currentFunnel, currentFunnel.Days); err != nil {
		return nil, err
	}

	checkpointAttempts := make(map[string]int64, len(checkpoint.ActionInterestAttemptFunnel.Outcomes))
	currentAttempts := make(map[string]int64, len(currentFunnel.Outcomes))
	keys := make(map[string]struct{})
	for _, bucket := range checkpoint.ActionInterestAttemptFunnel.Outcomes {
		key := bucket.Surface + ":" + bucket.Outcome
		checkpointAttempts[key] = bucket.AttemptCount
		keys[key] = struct{}{}
	}
	for _, bucket := range currentFunnel.Outcomes {
		key := bucket.Surface + ":" + bucket.Outcome
		currentAttempts[key] = bucket.AttemptCount
		keys[key] = struct{}{}
	}
	orderedKeys := make([]string, 0, len(keys))
	for key := range keys {
		orderedKeys = append(orderedKeys, key)
	}
	sort.Strings(orderedKeys)
	attemptChanges := make([]attemptBucketNetChange, 0, len(orderedKeys))
	var mcpInvalidNet int64
	for _, key := range orderedKeys {
		surface, outcome, ok := strings.Cut(key, ":")
		if !ok {
			return nil, fmt.Errorf("attempt key invalid")
		}
		netChange := currentAttempts[key] - checkpointAttempts[key]
		attemptChanges = append(attemptChanges, attemptBucketNetChange{
			Surface: surface, Outcome: outcome, NetChange: netChange,
		})
		if surface == "mcp" && outcome == "invalid_request" {
			mcpInvalidNet = netChange
		}
	}

	comparison := &stage1CheckpointComparison{
		Contract:                             stage1CheckpointContract,
		CheckpointStage1ReportSHA256:         checkpoint.Stage1ReportSHA256,
		CheckpointAttemptFunnelSHA256:        checkpoint.AttemptFunnelSHA256,
		CheckpointRevision:                   checkpoint.CandidateRevision,
		CheckpointStage1AsOf:                 checkpoint.Stage1Demand.AsOf,
		CurrentStage1AsOf:                    currentProof.AsOf,
		CheckpointAttemptFunnelAsOf:          checkpoint.ActionInterestAttemptFunnel.AsOf,
		CurrentAttemptFunnelAsOf:             currentFunnel.AsOf,
		MeaningfulSearchReceiptsNetChange:    currentProof.MeaningfulSearchReceipts - checkpoint.Stage1Demand.MeaningfulSearchReceipts,
		ResultSelectionsNetChange:            currentProof.ResultSelections - checkpoint.Stage1Demand.ResultSelections,
		SearchReceiptsWithSelectionNetChange: currentProof.SearchReceiptsWithSelection - checkpoint.Stage1Demand.SearchReceiptsWithSelection,
		ActionInterestReceiptsNetChange:      currentProof.ActionInterestReceipts - checkpoint.Stage1Demand.ActionInterestReceipts,
		SearchesWithActionInterestNetChange:  currentProof.SearchReceiptsWithActionInterest - checkpoint.Stage1Demand.SearchReceiptsWithActionInterest,
		DistinctInterestDomainsNetChange:     currentProof.DistinctInterestDomains - checkpoint.Stage1Demand.DistinctInterestDomains,
		ObservationSpanSecondsNetChange:      currentProof.ObservationSpanSeconds - checkpoint.Stage1Demand.ObservationSpanSeconds,
		AttemptBucketNetChanges:              attemptChanges,
		CountsAreRollingWindowNetChanges:     true,
		NetChangesAreNotCreatedEventCounts:   true,
		SearchesAreNotLeads:                  true,
		Stage1BecameReady:                    !checkpoint.Stage1Demand.Stage1Ready && currentProof.Stage1Ready,
		ReadinessDoesNotAuthorizePilot:       true,
		StrongestMechanismSelected:           false,
		CommercialProof:                      false,
		ContainsIdentifiers:                  false,
		ContainsQueriesOrPrompts:             false,
		ContainsContactData:                  false,
		ContainsRequestCoordinates:           false,
	}
	comparison.DiscoveryReceiptNetIncrease = comparison.MeaningfulSearchReceiptsNetChange > 0
	comparison.SelectionReceiptNetIncrease = comparison.SearchReceiptsWithSelectionNetChange > 0
	comparison.ExplicitInterestReceiptNetIncrease = comparison.SearchesWithActionInterestNetChange > 0
	comparison.MCPInvalidAttemptNetIncrease = mcpInvalidNet > 0
	return comparison, nil
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
