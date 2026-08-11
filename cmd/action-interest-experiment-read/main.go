// Command action-interest-experiment-read emits one privacy-bounded,
// read-only owner receipt for the post-selection action-interest experiment.
// It has no public route and never mutates NHS or provider state.
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

func main() {
	revision := flag.String("revision", "", "exact 40-character deployed commit")
	sinceRaw := flag.String("since", "", "UTC RFC3339 experiment boundary, no older than 30 days")
	checkpointPath := flag.String("attempt-checkpoint", "", "optional prior v2 receipt or evidence envelope")
	checkpointEnv := flag.String("attempt-checkpoint-base64-env", "", "optional environment name containing a base64 prior v2 receipt")
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
	if checkpoint != nil {
		comparison, err = compareAttemptCheckpoint(checkpoint, report)
		if err != nil {
			fail("attempt_checkpoint_comparison_failed")
		}
	}
	receipt := readReceipt{
		Contract:                       readContract,
		ReportSHA256:                   hex.EncodeToString(digest[:]),
		CandidateRevision:              candidate,
		BinaryRevision:                 compiled,
		Report:                         report,
		AttemptCheckpoint:              comparison,
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
