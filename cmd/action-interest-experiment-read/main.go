// Command action-interest-experiment-read emits one privacy-bounded,
// read-only owner receipt for the post-selection action-interest experiment.
// It has no public route and never mutates NHS or provider state.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"os"
	"strings"
	"time"

	"github.com/unitedideas/nothumansearch/internal/database"
	"github.com/unitedideas/nothumansearch/internal/models"
)

const (
	readContract = "nhs-post-selection-action-interest-experiment-read-v2"
	readTimeout  = 30 * time.Second
)

var releaseRevision = "development"

type readReceipt struct {
	Contract                       string                                        `json:"contract"`
	ReportSHA256                   string                                        `json:"report_sha256"`
	CandidateRevision              string                                        `json:"candidate_revision"`
	BinaryRevision                 string                                        `json:"binary_revision"`
	Report                         *models.PostSelectionActionInterestExperiment `json:"report"`
	ContainsIdentifiers            bool                                          `json:"contains_identifiers"`
	ContainsQueriesOrPrompts       bool                                          `json:"contains_queries_or_prompts"`
	ContainsContactData            bool                                          `json:"contains_contact_data"`
	OperatorContactedProvider      bool                                          `json:"operator_contacted_provider"`
	OperatorChangedCommercialState bool                                          `json:"operator_changed_commercial_state"`
	OperatorAffectedOrganicRank    bool                                          `json:"operator_affected_organic_rank"`
}

func main() {
	revision := flag.String("revision", "", "exact 40-character deployed commit")
	sinceRaw := flag.String("since", "", "UTC RFC3339 experiment boundary, no older than 30 days")
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
	receipt := readReceipt{
		Contract:                       readContract,
		ReportSHA256:                   hex.EncodeToString(digest[:]),
		CandidateRevision:              candidate,
		BinaryRevision:                 compiled,
		Report:                         report,
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
