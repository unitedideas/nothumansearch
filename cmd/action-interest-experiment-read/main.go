// Command action-interest-experiment-read emits one privacy-bounded,
// read-only owner receipt for the post-selection action-interest experiment.
// Given a sealed prior receipt, it also emits a hash-verified delta for the
// complete privacy-safe discovery-to-settlement funnel. It has no public route
// and never mutates NHS or provider state.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/unitedideas/nothumansearch/internal/database"
	"github.com/unitedideas/nothumansearch/internal/models"
)

const (
	readContract           = "nhs-post-selection-action-interest-experiment-read-v2"
	readTimeout            = 30 * time.Second
	checkpointMaximumBytes = 128 * 1024
)

var releaseRevision = "development"

type readReceipt struct {
	Contract                       string                                        `json:"contract"`
	ReportSHA256                   string                                        `json:"report_sha256"`
	CandidateRevision              string                                        `json:"candidate_revision"`
	BinaryRevision                 string                                        `json:"binary_revision"`
	Report                         *models.PostSelectionActionInterestExperiment `json:"report"`
	AttemptCheckpoint              *attemptCheckpointComparison                  `json:"attempt_checkpoint,omitempty"`
	FunnelCheckpoint               *funnelCheckpointComparison                   `json:"funnel_checkpoint,omitempty"`
	ContainsIdentifiers            bool                                          `json:"contains_identifiers"`
	ContainsQueriesOrPrompts       bool                                          `json:"contains_queries_or_prompts"`
	ContainsContactData            bool                                          `json:"contains_contact_data"`
	OperatorContactedProvider      bool                                          `json:"operator_contacted_provider"`
	OperatorChangedCommercialState bool                                          `json:"operator_changed_commercial_state"`
	OperatorAffectedOrganicRank    bool                                          `json:"operator_affected_organic_rank"`
}

type attemptCheckpointReceipt struct {
	Contract          string                                        `json:"contract"`
	ReportSHA256      string                                        `json:"report_sha256"`
	CandidateRevision string                                        `json:"candidate_revision"`
	BinaryRevision    string                                        `json:"binary_revision"`
	Report            *models.PostSelectionActionInterestExperiment `json:"report"`
}

type attemptCheckpointComparison struct {
	Contract                         string    `json:"contract"`
	CheckpointReportSHA256           string    `json:"checkpoint_report_sha256"`
	CheckpointRevision               string    `json:"checkpoint_revision"`
	CheckpointCheckedAt              time.Time `json:"checkpoint_checked_at"`
	CurrentCheckedAt                 time.Time `json:"current_checked_at"`
	AttemptDelta                     int64     `json:"attempt_delta"`
	UnavailableAttemptDelta          int64     `json:"unavailable_attempt_delta"`
	InvalidRequestAttemptDelta       int64     `json:"invalid_request_attempt_delta"`
	CountsAreAttemptsNotUniqueAgents bool      `json:"counts_are_attempts_not_unique_agents"`
	DemandEvidence                   bool      `json:"demand_evidence"`
	CommercialProof                  bool      `json:"commercial_proof"`
}

type funnelCheckpointComparison struct {
	Contract                                  string                       `json:"contract"`
	CheckpointReportSHA256                    string                       `json:"checkpoint_report_sha256"`
	CheckpointRevision                        string                       `json:"checkpoint_revision"`
	CheckpointCheckedAt                       time.Time                    `json:"checkpoint_checked_at"`
	CurrentCheckedAt                          time.Time                    `json:"current_checked_at"`
	MeaningfulSearchReceiptsDelta             int                          `json:"meaningful_search_receipts_delta"`
	DeveloperToolsSearchReceiptsDelta         int                          `json:"developer_tools_search_receipts_delta"`
	MCPSearchReceiptsDelta                    int                          `json:"mcp_search_receipts_delta"`
	RESTSearchReceiptsDelta                   int                          `json:"rest_search_receipts_delta"`
	ResultSelectionsDelta                     int                          `json:"result_selections_delta"`
	SearchReceiptsWithSelectionDelta          int                          `json:"search_receipts_with_selection_delta"`
	MCPResultSelectionsDelta                  int                          `json:"mcp_result_selections_delta"`
	RESTResultSelectionsDelta                 int                          `json:"rest_result_selections_delta"`
	ActiveActionInterestReceiptsNetChange     int                          `json:"active_action_interest_receipts_net_change"`
	SearchesWithActionInterestNetChange       int                          `json:"search_receipts_with_action_interest_net_change"`
	PostSelectionInterestReceiptsNetChange    int                          `json:"post_selection_action_interest_receipts_net_change"`
	PostSelectionSearchReceiptsNetChange      int                          `json:"post_selection_search_receipts_net_change"`
	MCPPostSelectionInterestsNetChange        int                          `json:"mcp_post_selection_action_interests_net_change"`
	RESTPostSelectionInterestsNetChange       int                          `json:"rest_post_selection_action_interests_net_change"`
	SyntheticActionInterestReceiptsDelta      int                          `json:"synthetic_action_interest_receipts_delta"`
	ProviderPilotActivationsDelta             int                          `json:"provider_pilot_activations_delta"`
	ProviderOfferActivationsDelta             int                          `json:"provider_offer_activations_delta"`
	ProviderCommercialAcceptancesDelta        int                          `json:"provider_commercial_acceptances_delta"`
	ProviderCommercialCommitmentsDelta        int                          `json:"provider_commercial_commitments_delta"`
	ProviderOffersReturnedDelta               int                          `json:"provider_offers_returned_delta"`
	ProviderTicketsCreatedDelta               int                          `json:"provider_tickets_created_delta"`
	ProviderHandoffsObservedDelta             int                          `json:"provider_handoffs_observed_delta"`
	ProviderOutcomesReportedDelta             int                          `json:"provider_outcomes_reported_delta"`
	ProviderPaidSettlementsDelta              int                          `json:"provider_paid_settlements_delta"`
	ProviderAvailableSettlementsDelta         int                          `json:"provider_available_settlements_delta"`
	CommercialStateEventsDelta                int                          `json:"commercial_state_events_delta"`
	Attempts                                  *attemptCheckpointComparison `json:"attempts"`
	CountsAreEventsNotUniqueAgents            bool                         `json:"counts_are_events_not_unique_agents"`
	ActiveInterestStateMayExpire              bool                         `json:"active_interest_state_may_expire"`
	ActiveInterestNetChangeIsNotCreatedEvents bool                         `json:"active_interest_net_change_is_not_created_event_count"`
	SearchesAreNotLeads                       bool                         `json:"searches_are_not_leads"`
	DiscoveryUsageObserved                    bool                         `json:"discovery_usage_observed"`
	ResultSelectionObserved                   bool                         `json:"result_selection_observed"`
	ExplicitPostSelectionInterestNetIncrease  bool                         `json:"explicit_post_selection_interest_net_increase"`
	ProviderHandoffObserved                   bool                         `json:"provider_handoff_observed"`
	PaidSettlementObserved                    bool                         `json:"paid_settlement_observed"`
	AvailableSettlementObserved               bool                         `json:"available_settlement_observed"`
	StrongestMechanismSelected                bool                         `json:"strongest_mechanism_selected"`
	ContainsIdentifiers                       bool                         `json:"contains_identifiers"`
	ContainsQueriesOrPrompts                  bool                         `json:"contains_queries_or_prompts"`
	ContainsContactData                       bool                         `json:"contains_contact_data"`
}

func main() {
	revision := flag.String("revision", "", "exact 40-character deployed commit")
	sinceRaw := flag.String("since", "", "UTC RFC3339 experiment boundary, no older than 30 days")
	checkpointPath := flag.String("attempt-checkpoint", "", "optional prior v2 receipt or evidence envelope for attempt and full-funnel comparison")
	checkpointEnv := flag.String("attempt-checkpoint-base64-env", "", "optional environment name containing a base64 prior v2 receipt for attempt and full-funnel comparison")
	flag.Parse()

	candidate := strings.ToLower(strings.TrimSpace(*revision))
	compiled := strings.ToLower(strings.TrimSpace(releaseRevision))
	if !validRevision(candidate) || compiled != candidate {
		fail("candidate_revision_mismatch")
	}
	since, err := time.Parse(time.RFC3339, strings.TrimSpace(*sinceRaw))
	if err != nil {
		fail("invalid_since")
	}
	_, offset := since.Zone()
	if offset != 0 || since.Format(time.RFC3339) != strings.TrimSpace(*sinceRaw) {
		fail("since_must_be_canonical_utc_rfc3339")
	}
	if strings.TrimSpace(*checkpointPath) != "" && strings.TrimSpace(*checkpointEnv) != "" {
		fail("multiple_attempt_checkpoints")
	}
	var checkpoint *attemptCheckpointReceipt
	if strings.TrimSpace(*checkpointPath) != "" {
		checkpoint, err = loadAttemptCheckpoint(strings.TrimSpace(*checkpointPath))
		if err != nil {
			fail("invalid_attempt_checkpoint")
		}
	} else if strings.TrimSpace(*checkpointEnv) != "" {
		checkpoint, err = loadAttemptCheckpointEnvironment(strings.TrimSpace(*checkpointEnv))
		if err != nil {
			fail("invalid_attempt_checkpoint")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
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

	report, err := models.ReadPostSelectionActionInterestExperiment(ctx, database.DB, since)
	if err != nil {
		if err == models.ErrInvalidPostSelectionExperimentWindow {
			fail("invalid_experiment_window")
		}
		fail("experiment_read_failed")
	}
	reportJSON, err := json.Marshal(report)
	if err != nil {
		fail("report_encoding_failed")
	}
	digest := sha256.Sum256(reportJSON)
	var comparison *attemptCheckpointComparison
	var funnelComparison *funnelCheckpointComparison
	if checkpoint != nil {
		comparison, err = compareAttemptCheckpoint(checkpoint, report)
		if err != nil {
			fail("attempt_checkpoint_comparison_failed")
		}
		funnelComparison, err = compareFunnelCheckpoint(checkpoint, report, comparison)
		if err != nil {
			fail("funnel_checkpoint_comparison_failed")
		}
	}
	receipt := readReceipt{
		Contract:                       readContract,
		ReportSHA256:                   hex.EncodeToString(digest[:]),
		CandidateRevision:              candidate,
		BinaryRevision:                 compiled,
		Report:                         report,
		AttemptCheckpoint:              comparison,
		FunnelCheckpoint:               funnelComparison,
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

func compareFunnelCheckpoint(
	checkpoint *attemptCheckpointReceipt,
	current *models.PostSelectionActionInterestExperiment,
	attempts *attemptCheckpointComparison,
) (*funnelCheckpointComparison, error) {
	if checkpoint == nil || checkpoint.Report == nil || current == nil || attempts == nil {
		return nil, fmt.Errorf("funnel checkpoint unavailable")
	}
	if err := validateExperimentCounters(checkpoint.Report); err != nil {
		return nil, fmt.Errorf("checkpoint report counters invalid: %w", err)
	}
	if err := validateExperimentCounters(current); err != nil {
		return nil, fmt.Errorf("current report counters invalid: %w", err)
	}

	monotonic := []struct {
		name                string
		checkpoint, current int
	}{
		{"meaningful_search_receipts", checkpoint.Report.MeaningfulSearchReceipts, current.MeaningfulSearchReceipts},
		{"developer_tools_search_receipts", checkpoint.Report.DeveloperToolsSearchReceipts, current.DeveloperToolsSearchReceipts},
		{"mcp_search_receipts", checkpoint.Report.MCPSearchReceipts, current.MCPSearchReceipts},
		{"rest_search_receipts", checkpoint.Report.RESTSearchReceipts, current.RESTSearchReceipts},
		{"result_selections", checkpoint.Report.ResultSelections, current.ResultSelections},
		{"search_receipts_with_selection", checkpoint.Report.SearchReceiptsWithSelection, current.SearchReceiptsWithSelection},
		{"mcp_result_selections", checkpoint.Report.MCPResultSelections, current.MCPResultSelections},
		{"rest_result_selections", checkpoint.Report.RESTResultSelections, current.RESTResultSelections},
		{"synthetic_action_interest_receipts", checkpoint.Report.SyntheticActionInterestReceipts, current.SyntheticActionInterestReceipts},
		{"provider_pilot_activations", checkpoint.Report.ProviderPilotActivations, current.ProviderPilotActivations},
		{"provider_offer_activations", checkpoint.Report.ProviderOfferActivations, current.ProviderOfferActivations},
		{"provider_commercial_acceptances", checkpoint.Report.ProviderCommercialAcceptances, current.ProviderCommercialAcceptances},
		{"provider_commercial_commitments", checkpoint.Report.ProviderCommercialCommitments, current.ProviderCommercialCommitments},
		{"provider_offers_returned", checkpoint.Report.ProviderOffersReturned, current.ProviderOffersReturned},
		{"provider_tickets_created", checkpoint.Report.ProviderTicketsCreated, current.ProviderTicketsCreated},
		{"provider_handoffs_observed", checkpoint.Report.ProviderHandoffsObserved, current.ProviderHandoffsObserved},
		{"provider_outcomes_reported", checkpoint.Report.ProviderOutcomesReported, current.ProviderOutcomesReported},
		{"provider_paid_settlements", checkpoint.Report.ProviderPaidSettlements, current.ProviderPaidSettlements},
		{"provider_available_settlements", checkpoint.Report.ProviderAvailableSettlements, current.ProviderAvailableSettlements},
		{"commercial_state_events_total", checkpoint.Report.CommercialStateEventsTotal, current.CommercialStateEventsTotal},
	}
	for _, counter := range monotonic {
		if counter.current < counter.checkpoint {
			return nil, fmt.Errorf("%s regressed", counter.name)
		}
	}

	result := &funnelCheckpointComparison{
		Contract:                                  "nhs-agent-monetization-funnel-checkpoint-comparison-v1",
		CheckpointReportSHA256:                    checkpoint.ReportSHA256,
		CheckpointRevision:                        checkpoint.CandidateRevision,
		CheckpointCheckedAt:                       checkpoint.Report.CheckedAt,
		CurrentCheckedAt:                          current.CheckedAt,
		MeaningfulSearchReceiptsDelta:             current.MeaningfulSearchReceipts - checkpoint.Report.MeaningfulSearchReceipts,
		DeveloperToolsSearchReceiptsDelta:         current.DeveloperToolsSearchReceipts - checkpoint.Report.DeveloperToolsSearchReceipts,
		MCPSearchReceiptsDelta:                    current.MCPSearchReceipts - checkpoint.Report.MCPSearchReceipts,
		RESTSearchReceiptsDelta:                   current.RESTSearchReceipts - checkpoint.Report.RESTSearchReceipts,
		ResultSelectionsDelta:                     current.ResultSelections - checkpoint.Report.ResultSelections,
		SearchReceiptsWithSelectionDelta:          current.SearchReceiptsWithSelection - checkpoint.Report.SearchReceiptsWithSelection,
		MCPResultSelectionsDelta:                  current.MCPResultSelections - checkpoint.Report.MCPResultSelections,
		RESTResultSelectionsDelta:                 current.RESTResultSelections - checkpoint.Report.RESTResultSelections,
		ActiveActionInterestReceiptsNetChange:     current.ActiveActionInterestReceipts - checkpoint.Report.ActiveActionInterestReceipts,
		SearchesWithActionInterestNetChange:       current.SearchReceiptsWithActionInterest - checkpoint.Report.SearchReceiptsWithActionInterest,
		PostSelectionInterestReceiptsNetChange:    current.PostSelectionInterestReceipts - checkpoint.Report.PostSelectionInterestReceipts,
		PostSelectionSearchReceiptsNetChange:      current.PostSelectionSearchReceipts - checkpoint.Report.PostSelectionSearchReceipts,
		MCPPostSelectionInterestsNetChange:        current.MCPPostSelectionInterests - checkpoint.Report.MCPPostSelectionInterests,
		RESTPostSelectionInterestsNetChange:       current.RESTPostSelectionInterests - checkpoint.Report.RESTPostSelectionInterests,
		SyntheticActionInterestReceiptsDelta:      current.SyntheticActionInterestReceipts - checkpoint.Report.SyntheticActionInterestReceipts,
		ProviderPilotActivationsDelta:             current.ProviderPilotActivations - checkpoint.Report.ProviderPilotActivations,
		ProviderOfferActivationsDelta:             current.ProviderOfferActivations - checkpoint.Report.ProviderOfferActivations,
		ProviderCommercialAcceptancesDelta:        current.ProviderCommercialAcceptances - checkpoint.Report.ProviderCommercialAcceptances,
		ProviderCommercialCommitmentsDelta:        current.ProviderCommercialCommitments - checkpoint.Report.ProviderCommercialCommitments,
		ProviderOffersReturnedDelta:               current.ProviderOffersReturned - checkpoint.Report.ProviderOffersReturned,
		ProviderTicketsCreatedDelta:               current.ProviderTicketsCreated - checkpoint.Report.ProviderTicketsCreated,
		ProviderHandoffsObservedDelta:             current.ProviderHandoffsObserved - checkpoint.Report.ProviderHandoffsObserved,
		ProviderOutcomesReportedDelta:             current.ProviderOutcomesReported - checkpoint.Report.ProviderOutcomesReported,
		ProviderPaidSettlementsDelta:              current.ProviderPaidSettlements - checkpoint.Report.ProviderPaidSettlements,
		ProviderAvailableSettlementsDelta:         current.ProviderAvailableSettlements - checkpoint.Report.ProviderAvailableSettlements,
		CommercialStateEventsDelta:                current.CommercialStateEventsTotal - checkpoint.Report.CommercialStateEventsTotal,
		Attempts:                                  attempts,
		CountsAreEventsNotUniqueAgents:            true,
		ActiveInterestStateMayExpire:              true,
		ActiveInterestNetChangeIsNotCreatedEvents: true,
		SearchesAreNotLeads:                       true,
		StrongestMechanismSelected:                false,
		ContainsIdentifiers:                       false,
		ContainsQueriesOrPrompts:                  false,
		ContainsContactData:                       false,
	}
	result.DiscoveryUsageObserved = result.MeaningfulSearchReceiptsDelta > 0
	result.ResultSelectionObserved = result.ResultSelectionsDelta > 0
	result.ExplicitPostSelectionInterestNetIncrease = result.PostSelectionInterestReceiptsNetChange > 0
	result.ProviderHandoffObserved = result.ProviderHandoffsObservedDelta > 0
	result.PaidSettlementObserved = result.ProviderPaidSettlementsDelta > 0
	result.AvailableSettlementObserved = result.ProviderAvailableSettlementsDelta > 0
	return result, nil
}

func validateExperimentCounters(report *models.PostSelectionActionInterestExperiment) error {
	if report == nil {
		return fmt.Errorf("report unavailable")
	}
	values := []int{
		report.MeaningfulSearchReceipts, report.DeveloperToolsSearchReceipts,
		report.ResultSelections, report.SearchReceiptsWithSelection,
		report.ActiveActionInterestReceipts, report.SearchReceiptsWithActionInterest,
		report.PostSelectionInterestReceipts, report.PostSelectionSearchReceipts,
		report.MCPSearchReceipts, report.MCPResultSelections, report.MCPPostSelectionInterests,
		report.RESTSearchReceipts, report.RESTResultSelections, report.RESTPostSelectionInterests,
		report.SyntheticActionInterestReceipts, report.ProviderPilotActivations,
		report.ProviderOfferActivations, report.ProviderCommercialAcceptances,
		report.ProviderCommercialCommitments, report.ProviderOffersReturned,
		report.ProviderTicketsCreated, report.ProviderHandoffsObserved,
		report.ProviderOutcomesReported, report.ProviderPaidSettlements,
		report.ProviderAvailableSettlements, report.CommercialStateEventsTotal,
	}
	for _, value := range values {
		if value < 0 {
			return fmt.Errorf("negative counter")
		}
	}
	if report.MeaningfulSearchReceipts != report.MCPSearchReceipts+report.RESTSearchReceipts ||
		report.ResultSelections != report.MCPResultSelections+report.RESTResultSelections ||
		report.PostSelectionInterestReceipts != report.MCPPostSelectionInterests+report.RESTPostSelectionInterests ||
		report.DeveloperToolsSearchReceipts > report.MeaningfulSearchReceipts ||
		report.SearchReceiptsWithSelection > report.MeaningfulSearchReceipts ||
		report.SearchReceiptsWithSelection > report.ResultSelections ||
		report.SearchReceiptsWithActionInterest > report.MeaningfulSearchReceipts ||
		report.PostSelectionSearchReceipts > report.SearchReceiptsWithSelection ||
		report.PostSelectionSearchReceipts > report.SearchReceiptsWithActionInterest ||
		report.ProviderAvailableSettlements > report.ProviderPaidSettlements {
		return fmt.Errorf("counter relationship invalid")
	}
	return nil
}

func loadAttemptCheckpoint(path string) (*attemptCheckpointReceipt, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, checkpointMaximumBytes+1))
	if err != nil || len(raw) == 0 || len(raw) > checkpointMaximumBytes {
		return nil, fmt.Errorf("checkpoint size invalid")
	}
	return parseAttemptCheckpoint(raw)
}

func loadAttemptCheckpointEnvironment(name string) (*attemptCheckpointReceipt, error) {
	if name != "NHS_ATTEMPT_CHECKPOINT_B64" {
		return nil, fmt.Errorf("checkpoint environment name invalid")
	}
	encoded := strings.TrimSpace(os.Getenv(name))
	if encoded == "" || len(encoded) > base64.StdEncoding.EncodedLen(checkpointMaximumBytes) {
		return nil, fmt.Errorf("checkpoint environment size invalid")
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(raw) == 0 || len(raw) > checkpointMaximumBytes {
		return nil, fmt.Errorf("checkpoint environment encoding invalid")
	}
	return parseAttemptCheckpoint(raw)
}

func parseAttemptCheckpoint(raw []byte) (*attemptCheckpointReceipt, error) {
	var envelope struct {
		ReaderReceipt json.RawMessage `json:"reader_receipt"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	receiptRaw := raw
	if len(envelope.ReaderReceipt) > 0 && !bytes.Equal(envelope.ReaderReceipt, []byte("null")) {
		receiptRaw = envelope.ReaderReceipt
	}
	var receipt attemptCheckpointReceipt
	if err := json.Unmarshal(receiptRaw, &receipt); err != nil {
		return nil, err
	}
	if receipt.Contract != readContract || receipt.Report == nil ||
		receipt.Report.Contract != models.PostSelectionActionInterestExperimentContract ||
		!validRevision(receipt.CandidateRevision) || receipt.BinaryRevision != receipt.CandidateRevision ||
		len(receipt.ReportSHA256) != sha256.Size*2 {
		return nil, fmt.Errorf("checkpoint contract invalid")
	}
	reportJSON, err := json.Marshal(receipt.Report)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(reportJSON)
	if receipt.ReportSHA256 != hex.EncodeToString(digest[:]) {
		return nil, fmt.Errorf("checkpoint digest invalid")
	}
	return &receipt, nil
}

func compareAttemptCheckpoint(
	checkpoint *attemptCheckpointReceipt,
	current *models.PostSelectionActionInterestExperiment,
) (*attemptCheckpointComparison, error) {
	if checkpoint == nil || checkpoint.Report == nil || current == nil ||
		!checkpoint.Report.Since.Equal(current.Since) ||
		!checkpoint.Report.CheckedAt.Before(current.CheckedAt) {
		return nil, fmt.Errorf("checkpoint window invalid")
	}
	checkpointAttempts := checkpoint.Report.ExactPostBoundaryAttempts + checkpoint.Report.BoundarySpanningAttemptCount
	currentAttempts := current.ExactPostBoundaryAttempts + current.BoundarySpanningAttemptCount
	checkpointUnavailable := checkpoint.Report.ExactUnavailableAttempts + checkpoint.Report.SpanningUnavailableAttemptCount
	currentUnavailable := current.ExactUnavailableAttempts + current.SpanningUnavailableAttemptCount
	checkpointInvalid := checkpoint.Report.ExactInvalidAttempts + checkpoint.Report.SpanningInvalidAttemptCount
	currentInvalid := current.ExactInvalidAttempts + current.SpanningInvalidAttemptCount
	if checkpoint.Report.ExactPostBoundaryAttempts < 0 || checkpoint.Report.BoundarySpanningAttemptCount < 0 ||
		checkpoint.Report.ExactUnavailableAttempts < 0 || checkpoint.Report.SpanningUnavailableAttemptCount < 0 ||
		checkpoint.Report.ExactInvalidAttempts < 0 || checkpoint.Report.SpanningInvalidAttemptCount < 0 ||
		current.ExactPostBoundaryAttempts < 0 || current.BoundarySpanningAttemptCount < 0 ||
		current.ExactUnavailableAttempts < 0 || current.SpanningUnavailableAttemptCount < 0 ||
		current.ExactInvalidAttempts < 0 || current.SpanningInvalidAttemptCount < 0 ||
		checkpointUnavailable+checkpointInvalid > checkpointAttempts ||
		currentUnavailable+currentInvalid > currentAttempts {
		return nil, fmt.Errorf("checkpoint counters invalid")
	}
	if currentAttempts < checkpointAttempts || currentUnavailable < checkpointUnavailable || currentInvalid < checkpointInvalid {
		return nil, fmt.Errorf("checkpoint counters regressed")
	}
	return &attemptCheckpointComparison{
		Contract:                         "nhs-action-interest-attempt-checkpoint-comparison-v1",
		CheckpointReportSHA256:           checkpoint.ReportSHA256,
		CheckpointRevision:               checkpoint.CandidateRevision,
		CheckpointCheckedAt:              checkpoint.Report.CheckedAt,
		CurrentCheckedAt:                 current.CheckedAt,
		AttemptDelta:                     currentAttempts - checkpointAttempts,
		UnavailableAttemptDelta:          currentUnavailable - checkpointUnavailable,
		InvalidRequestAttemptDelta:       currentInvalid - checkpointInvalid,
		CountsAreAttemptsNotUniqueAgents: true,
		DemandEvidence:                   false,
		CommercialProof:                  false,
	}, nil
}

func validRevision(value string) bool {
	if len(value) != 40 || value != strings.ToLower(value) {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 20
}

func fail(code string) {
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
		"contract": readContract,
		"ok":       false,
		"error":    code,
	})
	os.Exit(1)
}
