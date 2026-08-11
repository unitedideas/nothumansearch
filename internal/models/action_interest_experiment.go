package models

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

const PostSelectionActionInterestExperimentContract = "nhs-post-selection-action-interest-experiment-v1"

var ErrInvalidPostSelectionExperimentWindow = errors.New("invalid post-selection experiment window")

// PostSelectionActionInterestExperiment is an owner-only aggregate. It has no
// receipt, domain, query, prompt, contact, network, agent, or principal
// identifier. The post-selection numerator is restricted to an active
// action-interest receipt for the exact receipt/domain pair after its exact
// selection, so search-time interest and selection alone cannot be relabeled as
// evidence for this intervention.
type PostSelectionActionInterestExperiment struct {
	Contract                         string    `json:"contract"`
	Since                            time.Time `json:"since"`
	CheckedAt                        time.Time `json:"checked_at"`
	EligibleSurfaces                 []string  `json:"eligible_surfaces"`
	MeaningfulSearchReceipts         int       `json:"meaningful_search_receipts"`
	DeveloperToolsSearchReceipts     int       `json:"developer_tools_search_receipts"`
	ResultSelections                 int       `json:"result_selections"`
	SearchReceiptsWithSelection      int       `json:"search_receipts_with_selection"`
	ActiveActionInterestReceipts     int       `json:"active_action_interest_receipts"`
	SearchReceiptsWithActionInterest int       `json:"search_receipts_with_action_interest"`
	PostSelectionInterestReceipts    int       `json:"post_selection_action_interest_receipts"`
	PostSelectionSearchReceipts      int       `json:"post_selection_search_receipts"`
	PostSelectionConversionRate      *float64  `json:"post_selection_conversion_rate"`
	MCPSearchReceipts                int       `json:"mcp_search_receipts"`
	MCPResultSelections              int       `json:"mcp_result_selections"`
	MCPPostSelectionInterests        int       `json:"mcp_post_selection_action_interests"`
	RESTSearchReceipts               int       `json:"rest_search_receipts"`
	RESTResultSelections             int       `json:"rest_result_selections"`
	RESTPostSelectionInterests       int       `json:"rest_post_selection_action_interests"`
}

// ReadPostSelectionActionInterestExperiment performs one repeatable-read,
// read-only PostgreSQL transaction. Retention is at most 30 days, so accepting
// an older boundary would imply completeness the underlying rows cannot prove.
func ReadPostSelectionActionInterestExperiment(
	ctx context.Context,
	db *sql.DB,
	since time.Time,
) (*PostSelectionActionInterestExperiment, error) {
	if ctx == nil || db == nil || since.IsZero() {
		return nil, ErrInvalidPostSelectionExperimentWindow
	}
	since = since.UTC()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var checkedAt time.Time
	if err := tx.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&checkedAt); err != nil {
		return nil, err
	}
	checkedAt = checkedAt.UTC()
	if since.After(checkedAt) || checkedAt.Sub(since) > 30*24*time.Hour {
		return nil, ErrInvalidPostSelectionExperimentWindow
	}

	report := &PostSelectionActionInterestExperiment{
		Contract:         PostSelectionActionInterestExperimentContract,
		Since:            since,
		CheckedAt:        checkedAt,
		EligibleSurfaces: []string{"mcp", "rest"},
	}
	err = tx.QueryRowContext(ctx, `
		WITH eligible_searches AS (
			SELECT receipt.id, receipt.surface, receipt.demand_topics
			FROM search_receipts receipt
			WHERE NOT receipt.is_synthetic
			  AND receipt.surface IN ('mcp','rest')
			  AND receipt.stage1_integrity_generation=1
			  AND receipt.created_at >= $1
			  AND receipt.created_at <= $2
			  AND EXISTS (
				SELECT 1 FROM organic_results_returned returned
				WHERE returned.search_receipt_id=receipt.id
				  AND returned.stage1_integrity_generation=1
				  AND returned.returned_at >= $1
				  AND returned.returned_at <= $2
			  )
		), eligible_selections AS (
			SELECT selection.search_receipt_id, selection.site_domain_snapshot,
			       selection.selected_at, receipt.surface
			FROM result_selections selection
			JOIN eligible_searches receipt ON receipt.id=selection.search_receipt_id
			JOIN organic_results_returned returned
			  ON returned.search_receipt_id=selection.search_receipt_id
			 AND returned.site_domain_snapshot=selection.site_domain_snapshot
			WHERE selection.stage1_integrity_generation=1
			  AND returned.stage1_integrity_generation=1
			  AND selection.selected_at >= $1
			  AND selection.selected_at <= $2
			  AND returned.returned_at >= $1
			  AND returned.returned_at <= $2
		), eligible_interests AS (
			SELECT interest.search_receipt_id, interest.site_domain_snapshot,
			       interest.created_at, receipt.surface
			FROM action_interest_receipts interest
			JOIN eligible_searches receipt ON receipt.id=interest.search_receipt_id
			WHERE interest.stage1_integrity_generation=1
			  AND interest.created_at >= $1
			  AND interest.created_at <= $2
			  AND interest.expires_at > $2
		), post_selection_interests AS (
			SELECT DISTINCT interest.search_receipt_id,
			       interest.site_domain_snapshot, interest.surface
			FROM eligible_interests interest
			JOIN eligible_selections selection
			  ON selection.search_receipt_id=interest.search_receipt_id
			 AND selection.site_domain_snapshot=interest.site_domain_snapshot
			WHERE interest.created_at >= selection.selected_at
		)
		SELECT
		  (SELECT COUNT(*)::int FROM eligible_searches),
		  (SELECT COUNT(*)::int FROM eligible_searches
		    WHERE demand_topics @> ARRAY['developer-tools']::text[]),
		  (SELECT COUNT(*)::int FROM eligible_selections),
		  (SELECT COUNT(DISTINCT search_receipt_id)::int FROM eligible_selections),
		  (SELECT COUNT(*)::int FROM eligible_interests),
		  (SELECT COUNT(DISTINCT search_receipt_id)::int FROM eligible_interests),
		  (SELECT COUNT(*)::int FROM post_selection_interests),
		  (SELECT COUNT(DISTINCT search_receipt_id)::int FROM post_selection_interests),
		  (SELECT COUNT(*)::int FROM eligible_searches WHERE surface='mcp'),
		  (SELECT COUNT(*)::int FROM eligible_selections WHERE surface='mcp'),
		  (SELECT COUNT(*)::int FROM post_selection_interests WHERE surface='mcp'),
		  (SELECT COUNT(*)::int FROM eligible_searches WHERE surface='rest'),
		  (SELECT COUNT(*)::int FROM eligible_selections WHERE surface='rest'),
		  (SELECT COUNT(*)::int FROM post_selection_interests WHERE surface='rest')`,
		since, checkedAt,
	).Scan(
		&report.MeaningfulSearchReceipts,
		&report.DeveloperToolsSearchReceipts,
		&report.ResultSelections,
		&report.SearchReceiptsWithSelection,
		&report.ActiveActionInterestReceipts,
		&report.SearchReceiptsWithActionInterest,
		&report.PostSelectionInterestReceipts,
		&report.PostSelectionSearchReceipts,
		&report.MCPSearchReceipts,
		&report.MCPResultSelections,
		&report.MCPPostSelectionInterests,
		&report.RESTSearchReceipts,
		&report.RESTResultSelections,
		&report.RESTPostSelectionInterests,
	)
	if err != nil {
		return nil, err
	}
	if report.SearchReceiptsWithSelection > 0 {
		rate := float64(report.PostSelectionSearchReceipts) /
			float64(report.SearchReceiptsWithSelection)
		report.PostSelectionConversionRate = &rate
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return report, nil
}
