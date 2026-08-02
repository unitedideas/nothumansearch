package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

const (
	ActionInterestConfirmationV1 = "nhs-action-interest-v1"
	ActionInterestRetentionDays  = 30
	Stage1ObservationWindowDays  = 14
)

var (
	ErrActionInterestStoreUnavailable   = errors.New("action-interest store unavailable")
	ErrActionInterestEntropyUnavailable = errors.New("action-interest receipt entropy unavailable")
	ErrInvalidActionInterest            = errors.New("invalid action-interest request")
	ErrActionInterestUnavailable        = errors.New("action-interest source unavailable")
	ErrActionInterestConflict           = errors.New("action interest already recorded with a different action")

	readActionInterestEntropy = rand.Read
	actionInterestSearchID    = regexp.MustCompile(`^nhs_sr_[A-Za-z0-9_-]{16}$`)
	actionInterestDomainLabel = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
)

var actionInterestTypes = []string{
	"quote", "trial", "demo", "booking", "application", "signup", "purchase",
}

type ActionInterestInput struct {
	SearchID                       string
	Domain                         string
	ActionType                     string
	Surface                        string
	CallerAttestsPrincipalInterest bool
	ConfirmationVersion            string
}

// ActionInterestReceipt is a non-contacting, non-commercial demand receipt.
// Replayed is response metadata and is not persisted.
type ActionInterestReceipt struct {
	PublicID                       string    `json:"id"`
	SearchID                       string    `json:"search_id"`
	Domain                         string    `json:"domain"`
	ActionType                     string    `json:"action_type"`
	Surface                        string    `json:"surface"`
	CallerAttestsPrincipalInterest bool      `json:"caller_attests_principal_interest"`
	ConfirmationVersion            string    `json:"confirmation_version"`
	CreatedAt                      time.Time `json:"created_at"`
	ExpiresAt                      time.Time `json:"expires_at"`
	Replayed                       bool      `json:"idempotent_replay"`
}

type Stage1DemandBucket struct {
	Value        string `json:"value"`
	ReceiptCount int    `json:"receipt_count"`
}

type Stage1DemandProof struct {
	Days                             int                  `json:"days"`
	RetentionDays                    int                  `json:"retention_days"`
	SyntheticExcluded                bool                 `json:"synthetic_excluded"`
	CountsAreReceiptsNotUniqueAgents bool                 `json:"counts_are_receipts_not_unique_agents"`
	CommercialProof                  bool                 `json:"commercial_proof"`
	MeaningfulSearchReceipts         int                  `json:"meaningful_search_receipts"`
	ResultSelections                 int                  `json:"result_selections"`
	SearchReceiptsWithSelection      int                  `json:"search_receipts_with_selection"`
	ActionInterestReceipts           int                  `json:"action_interest_receipts"`
	SearchReceiptsWithActionInterest int                  `json:"search_receipts_with_action_interest"`
	DistinctInterestDomains          int                  `json:"distinct_interest_domains"`
	BucketReceiptThreshold           int                  `json:"bucket_receipt_threshold"`
	TopicBucketsMayOverlap           bool                 `json:"topic_buckets_may_overlap"`
	DemandTopics                     []Stage1DemandBucket `json:"demand_topics"`
	ActionTypes                      []Stage1DemandBucket `json:"action_types"`
	ObservationWindowDays            int                  `json:"observation_window_days"`
	ObservationSpanDays              int                  `json:"observation_span_days"`
	ObservationWindowMet             bool                 `json:"observation_window_met"`
	Stage1Ready                      bool                 `json:"stage1_ready"`
	Targets                          map[string]int       `json:"targets"`
	TargetsMet                       map[string]bool      `json:"targets_met"`
}

func GenerateActionInterestID() (string, error) {
	var raw [12]byte
	if _, err := readActionInterestEntropy(raw[:]); err != nil {
		return "", fmt.Errorf("%w: %v", ErrActionInterestEntropyUnavailable, err)
	}
	return "nhs_air_" + base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func ValidActionInterestType(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, candidate := range actionInterestTypes {
		if value == candidate {
			return true
		}
	}
	return false
}

func ActionInterestTypes() []string {
	return append([]string(nil), actionInterestTypes...)
}

// NormalizeActionInterestDomain accepts only a bare public-style DNS name.
// It deliberately rejects URLs and paths so a caller cannot smuggle secret
// path/query data into a domain snapshot retained only until source expiry,
// no later than 30 days after the search.
func NormalizeActionInterestDomain(value string) string {
	domain := strings.ToLower(strings.TrimSpace(value))
	domain = strings.TrimSuffix(domain, ".")
	if domain == "" || len(domain) > 253 || net.ParseIP(domain) != nil || !strings.Contains(domain, ".") {
		return ""
	}
	if strings.ContainsAny(domain, "/?#@:\\") {
		return ""
	}
	for _, label := range strings.Split(domain, ".") {
		if len(label) > 63 || !actionInterestDomainLabel.MatchString(label) {
			return ""
		}
	}
	return domain
}

func validateActionInterestInput(input ActionInterestInput) (ActionInterestInput, error) {
	input.SearchID = strings.TrimSpace(input.SearchID)
	input.Domain = NormalizeActionInterestDomain(input.Domain)
	input.ActionType = strings.ToLower(strings.TrimSpace(input.ActionType))
	input.Surface = normalizeDemandSurface(input.Surface)
	input.ConfirmationVersion = strings.TrimSpace(input.ConfirmationVersion)
	if !actionInterestSearchID.MatchString(input.SearchID) || input.Domain == "" ||
		!ValidActionInterestType(input.ActionType) || !input.CallerAttestsPrincipalInterest ||
		input.ConfirmationVersion != ActionInterestConfirmationV1 {
		return ActionInterestInput{}, ErrInvalidActionInterest
	}
	return input, nil
}

// RecordActionInterest serializes on the exact returned-result row. That makes
// the natural key idempotent even under concurrent REST/MCP retries without
// relying on a same-statement ON CONFLICT snapshot that may not see the winner.
func RecordActionInterest(db *sql.DB, input ActionInterestInput) (*ActionInterestReceipt, error) {
	normalized, err := validateActionInterestInput(input)
	if err != nil {
		return nil, err
	}
	if db == nil {
		return nil, ErrActionInterestStoreUnavailable
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var receiptID string
	err = tx.QueryRow(`
		SELECT receipt.id::text
		FROM search_receipts receipt
		JOIN organic_results_returned returned
		  ON returned.search_receipt_id=receipt.id
		WHERE receipt.public_id=$1
		  AND returned.site_domain_snapshot=$2
		  AND NOT receipt.is_synthetic
		  AND receipt.created_at + INTERVAL '30 days' > NOW()
		FOR SHARE OF receipt
		FOR UPDATE OF returned`, normalized.SearchID, normalized.Domain).
		Scan(&receiptID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrActionInterestUnavailable
	}
	if err != nil {
		return nil, err
	}

	existing := &ActionInterestReceipt{SearchID: normalized.SearchID}
	err = tx.QueryRow(`
		SELECT interest.public_id, interest.site_domain_snapshot,
		       interest.action_type, interest.surface,
		       interest.caller_attests_principal_interest,
		       interest.confirmation_version,
		       interest.created_at, interest.expires_at
		FROM action_interest_receipts interest
		JOIN search_receipts receipt ON receipt.id=interest.search_receipt_id
		WHERE interest.search_receipt_id=$1::uuid
		  AND interest.site_domain_snapshot=$2
		  AND NOT receipt.is_synthetic
		  AND receipt.created_at + INTERVAL '30 days' > clock_timestamp()`, receiptID, normalized.Domain).
		Scan(
			&existing.PublicID, &existing.Domain, &existing.ActionType,
			&existing.Surface, &existing.CallerAttestsPrincipalInterest,
			&existing.ConfirmationVersion, &existing.CreatedAt, &existing.ExpiresAt,
		)
	switch {
	case err == nil:
		if existing.ActionType != normalized.ActionType ||
			existing.ConfirmationVersion != normalized.ConfirmationVersion ||
			!existing.CallerAttestsPrincipalInterest {
			return nil, ErrActionInterestConflict
		}
		existing.Replayed = true
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return existing, nil
	case !errors.Is(err, sql.ErrNoRows):
		return nil, err
	}
	publicID, err := GenerateActionInterestID()
	if err != nil {
		return nil, err
	}

	receipt := &ActionInterestReceipt{SearchID: normalized.SearchID}
	err = tx.QueryRow(`
		WITH current_clock AS (
			SELECT clock_timestamp() AS at
		)
		INSERT INTO action_interest_receipts (
			public_id, search_receipt_id, source_is_synthetic,
			site_domain_snapshot, action_type, surface,
			caller_attests_principal_interest, confirmation_version,
			created_at, expires_at
		)
		SELECT $1, source.id, false,
		       returned.site_domain_snapshot, $4, $5, true, $6,
		       current_clock.at, source.created_at + INTERVAL '30 days'
		FROM search_receipts source
		JOIN organic_results_returned returned
		  ON returned.search_receipt_id=source.id
		CROSS JOIN current_clock
		WHERE source.id=$2::uuid
		  AND returned.site_domain_snapshot=$3
		  AND NOT source.is_synthetic
		  AND source.created_at + INTERVAL '30 days' > current_clock.at
		RETURNING public_id, site_domain_snapshot, action_type, surface,
		          caller_attests_principal_interest, confirmation_version,
		          created_at, expires_at`, publicID, receiptID, normalized.Domain,
		normalized.ActionType, normalized.Surface, normalized.ConfirmationVersion).
		Scan(
			&receipt.PublicID, &receipt.Domain, &receipt.ActionType, &receipt.Surface,
			&receipt.CallerAttestsPrincipalInterest, &receipt.ConfirmationVersion,
			&receipt.CreatedAt, &receipt.ExpiresAt,
		)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrActionInterestUnavailable
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return receipt, nil
}

// GetStage1DemandProof returns owner-only, privacy-safe aggregate demand. It
// cannot prove unique agents or principals because those identities are not
// retained, and it is deliberately separate from commercial exchange proof.
func GetStage1DemandProof(db *sql.DB, days int) (*Stage1DemandProof, error) {
	if db == nil {
		return nil, ErrActionInterestStoreUnavailable
	}
	if days < 1 || days > ActionInterestRetentionDays {
		days = ActionInterestRetentionDays
	}
	proof := &Stage1DemandProof{
		Days:                             days,
		RetentionDays:                    ActionInterestRetentionDays,
		SyntheticExcluded:                true,
		CountsAreReceiptsNotUniqueAgents: true,
		CommercialProof:                  false,
		BucketReceiptThreshold:           ProviderDemandPrivacyThreshold,
		TopicBucketsMayOverlap:           true,
		DemandTopics:                     []Stage1DemandBucket{},
		ActionTypes:                      []Stage1DemandBucket{},
		ObservationWindowDays:            Stage1ObservationWindowDays,
		Targets: map[string]int{
			"meaningful_search_receipts":           100,
			"search_receipts_with_selection":       20,
			"search_receipts_with_action_interest": 10,
			"observation_window_days":              Stage1ObservationWindowDays,
		},
	}

	err := db.QueryRow(`
		SELECT
		  (SELECT COUNT(*)::int
		     FROM search_receipts receipt
		    WHERE NOT receipt.is_synthetic
		      AND receipt.result_count > 0
		      AND receipt.created_at >= NOW() - $1::int * INTERVAL '1 day'),
		  (SELECT COUNT(*)::int
		     FROM result_selections selection
		     JOIN search_receipts receipt ON receipt.id=selection.search_receipt_id
		    WHERE NOT receipt.is_synthetic
		      AND selection.selected_at >= NOW() - $1::int * INTERVAL '1 day'),
		  (SELECT COUNT(DISTINCT selection.search_receipt_id)::int
		     FROM result_selections selection
		     JOIN search_receipts receipt ON receipt.id=selection.search_receipt_id
		    WHERE NOT receipt.is_synthetic
		      AND selection.selected_at >= NOW() - $1::int * INTERVAL '1 day'),
		  (SELECT COUNT(*)::int
		     FROM action_interest_receipts interest
		     JOIN search_receipts receipt ON receipt.id=interest.search_receipt_id
		    WHERE NOT receipt.is_synthetic
		      AND interest.created_at >= NOW() - $1::int * INTERVAL '1 day'
		      AND interest.expires_at > NOW()),
		  (SELECT COUNT(DISTINCT interest.search_receipt_id)::int
		     FROM action_interest_receipts interest
		     JOIN search_receipts receipt ON receipt.id=interest.search_receipt_id
		    WHERE NOT receipt.is_synthetic
		      AND interest.created_at >= NOW() - $1::int * INTERVAL '1 day'
		      AND interest.expires_at > NOW()),
		  (SELECT COUNT(DISTINCT interest.site_domain_snapshot)::int
		     FROM action_interest_receipts interest
		     JOIN search_receipts receipt ON receipt.id=interest.search_receipt_id
		    WHERE NOT receipt.is_synthetic
		      AND interest.created_at >= NOW() - $1::int * INTERVAL '1 day'
		      AND interest.expires_at > NOW()),
		  (SELECT COALESCE(FLOOR(EXTRACT(EPOCH FROM (MAX(receipt.created_at) - MIN(receipt.created_at))) / 86400), 0)::int
		     FROM search_receipts receipt
		    WHERE NOT receipt.is_synthetic
		      AND receipt.result_count > 0
		      AND receipt.created_at >= NOW() - $1::int * INTERVAL '1 day')`, days).
		Scan(
			&proof.MeaningfulSearchReceipts,
			&proof.ResultSelections,
			&proof.SearchReceiptsWithSelection,
			&proof.ActionInterestReceipts,
			&proof.SearchReceiptsWithActionInterest,
			&proof.DistinctInterestDomains,
			&proof.ObservationSpanDays,
		)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`
		SELECT topic, COUNT(DISTINCT receipt.id)::int
		FROM search_receipts receipt
		CROSS JOIN LATERAL unnest(receipt.demand_topics) AS topic
		WHERE NOT receipt.is_synthetic
		  AND receipt.result_count > 0
		  AND receipt.created_at >= NOW() - $1::int * INTERVAL '1 day'
		GROUP BY topic
		HAVING COUNT(DISTINCT receipt.id) >= $2
		ORDER BY COUNT(DISTINCT receipt.id) DESC, topic`, days, ProviderDemandPrivacyThreshold)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var bucket Stage1DemandBucket
		if err := rows.Scan(&bucket.Value, &bucket.ReceiptCount); err != nil {
			_ = rows.Close()
			return nil, err
		}
		proof.DemandTopics = append(proof.DemandTopics, bucket)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	rows, err = db.Query(`
		SELECT interest.action_type,
		       COUNT(DISTINCT interest.search_receipt_id)::int
		FROM action_interest_receipts interest
		JOIN search_receipts receipt ON receipt.id=interest.search_receipt_id
		WHERE NOT receipt.is_synthetic
		  AND interest.created_at >= NOW() - $1::int * INTERVAL '1 day'
		  AND interest.expires_at > NOW()
		GROUP BY interest.action_type
		HAVING COUNT(DISTINCT interest.search_receipt_id) >= $2
		ORDER BY COUNT(DISTINCT interest.search_receipt_id) DESC,
		         interest.action_type`, days, ProviderDemandPrivacyThreshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var bucket Stage1DemandBucket
		if err := rows.Scan(&bucket.Value, &bucket.ReceiptCount); err != nil {
			return nil, err
		}
		proof.ActionTypes = append(proof.ActionTypes, bucket)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	proof.ObservationWindowMet = proof.ObservationSpanDays >= proof.ObservationWindowDays
	proof.TargetsMet = map[string]bool{
		"meaningful_search_receipts":           proof.MeaningfulSearchReceipts >= proof.Targets["meaningful_search_receipts"],
		"search_receipts_with_selection":       proof.SearchReceiptsWithSelection >= proof.Targets["search_receipts_with_selection"],
		"search_receipts_with_action_interest": proof.SearchReceiptsWithActionInterest >= proof.Targets["search_receipts_with_action_interest"],
		"observation_window_days":              proof.ObservationWindowMet,
	}
	proof.Stage1Ready = true
	for _, met := range proof.TargetsMet {
		proof.Stage1Ready = proof.Stage1Ready && met
	}
	return proof, nil
}
