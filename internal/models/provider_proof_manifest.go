package models

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/unitedideas/nothumansearch/internal/providerexchange"
)

var (
	ErrProviderProofManifestNotIssuable     = errors.New("provider proof manifest is not issuable")
	ErrProviderProofManifestSnapshotChanged = errors.New("provider proof manifest snapshot changed")
	ErrProviderProofManifestRequestConflict = errors.New("provider proof manifest request conflicts with the issued manifest")
	ErrProviderProofManifestIntegrity       = errors.New("persisted provider proof manifest failed integrity verification")
)

// ProviderCommercialProofManifestCandidate is the complete owner preview of a
// potential signed statement. It contains exact-pilot aggregate facts and
// aggregate review coverage only; it has no row-level provider, offer, ticket,
// handoff, callback, query, search-receipt, principal, or agent identifiers.
type ProviderCommercialProofManifestCandidate struct {
	ManifestContractVersion           string                                           `json:"manifest_contract_version"`
	SignatureVerificationScope        string                                           `json:"signature_verification_scope"`
	ProviderPilotEpochID              string                                           `json:"provider_pilot_epoch_id"`
	ProviderPilotContractVersion      string                                           `json:"provider_pilot_contract_version"`
	ReviewContractVersion             string                                           `json:"review_contract_version"`
	ReviewEvidenceContractVersion     string                                           `json:"review_evidence_contract_version"`
	MarketPolicyContractVersion       string                                           `json:"market_policy_contract_version"`
	ProofSnapshotSHA256               string                                           `json:"proof_snapshot_sha256"`
	ReviewEvidenceSHA256              string                                           `json:"review_evidence_sha256"`
	PilotDemandTopic                  string                                           `json:"pilot_demand_topic"`
	PilotStatus                       string                                           `json:"pilot_status"`
	OutcomeReceiptIntegrityValid      bool                                             `json:"outcome_receipt_integrity_valid"`
	ReviewIntegrityValid              bool                                             `json:"review_integrity_valid"`
	VerifiedOutcomeReceipts           int                                              `json:"verified_outcome_receipts"`
	RejectedOutcomeReceipts           int                                              `json:"rejected_outcome_receipts"`
	VerifiedOutcomeLedgerEntries      int                                              `json:"verified_outcome_ledger_entries"`
	RejectedOutcomeLedgerEntries      int                                              `json:"rejected_outcome_ledger_entries"`
	VerifiedProviderCompanies         int                                              `json:"verified_provider_companies"`
	VerifiedProviderAcceptedHandoffs  int                                              `json:"verified_provider_accepted_handoffs"`
	VerifiedProviderActivations       int                                              `json:"verified_provider_confirmed_activations"`
	VerifiedProviderRenewals          int                                              `json:"verified_provider_renewals"`
	VerifiedProviderConversions       int                                              `json:"verified_provider_confirmed_conversions"`
	ReviewCoverage                    providerexchange.CommercialProofReviewCoverage   `json:"review_coverage"`
	MonetaryAmountsWithheldForPrivacy bool                                             `json:"monetary_amounts_withheld_for_privacy"`
	VerifiedPrepaidSettled            []providerexchange.CommercialProofCurrencyAmount `json:"verified_prepaid_settled"`
	VerifiedPrepaidNetDebited         []providerexchange.CommercialProofCurrencyAmount `json:"verified_prepaid_net_debited"`
	VerifiedTermsNetReceivable        []providerexchange.CommercialProofCurrencyAmount `json:"verified_terms_net_receivable"`
	PilotThresholdsMet                bool                                             `json:"pilot_thresholds_met"`
	OrganicRankSold                   bool                                             `json:"organic_rank_sold"`
	RawQueriesSold                    bool                                             `json:"raw_queries_sold"`
	AgentIdentitiesSold               bool                                             `json:"agent_identities_sold"`
	Issuable                          bool                                             `json:"issuable"`
	IssuanceBlockers                  []string                                         `json:"issuance_blockers"`
	EvidenceScope                     string                                           `json:"evidence_scope"`
}

// ProviderCommercialProofManifestRecord is an immutable signed aggregate. The
// owner/evidence references used to authorize issuance stay internal and are
// never serialized with the issued record.
type ProviderCommercialProofManifestRecord struct {
	ID                      string    `json:"id"`
	ProviderPilotEpochID    string    `json:"provider_pilot_epoch_id"`
	ManifestContractVersion string    `json:"manifest_contract_version"`
	ProofSnapshotSHA256     string    `json:"proof_snapshot_sha256"`
	ReviewEvidenceSHA256    string    `json:"review_evidence_sha256"`
	KeyID                   string    `json:"key_id"`
	SignedManifest          string    `json:"signed_manifest"`
	Signature               string    `json:"signature"`
	PayloadSHA256           string    `json:"payload_sha256"`
	IssuedAt                time.Time `json:"issued_at"`
	Replayed                bool      `json:"-"`
	OwnerReference          string    `json:"-"`
	EvidenceReference       string    `json:"-"`
}

type ProviderCommercialProofManifestInput struct {
	ProviderPilotEpochID   string
	ExpectedSnapshotSHA256 string
	OwnerReference         string
	EvidenceReference      string
}

const providerCommercialProofManifestColumns = `
	id::text, provider_pilot_epoch_id::text, manifest_contract_version,
	proof_snapshot_sha256, review_evidence_sha256, key_id,
	signed_manifest, signature, payload_sha256,
	owner_reference, evidence_reference, issued_at`

func scanProviderCommercialProofManifest(
	row rowScanner,
	signer *providerexchange.Signer,
) (*ProviderCommercialProofManifestRecord, error) {
	var record ProviderCommercialProofManifestRecord
	if err := row.Scan(
		&record.ID, &record.ProviderPilotEpochID,
		&record.ManifestContractVersion, &record.ProofSnapshotSHA256,
		&record.ReviewEvidenceSHA256, &record.KeyID,
		&record.SignedManifest, &record.Signature,
		&record.PayloadSHA256, &record.OwnerReference,
		&record.EvidenceReference, &record.IssuedAt,
	); err != nil {
		return nil, err
	}
	record.ID = strings.ToLower(record.ID)
	record.ProviderPilotEpochID = strings.ToLower(record.ProviderPilotEpochID)
	if !validProviderUUID(record.ID) || !validProviderUUID(record.ProviderPilotEpochID) ||
		record.ManifestContractVersion != providerexchange.CommercialProofManifestContractV1 ||
		!providerHashPattern.MatchString(record.ProofSnapshotSHA256) ||
		!providerHashPattern.MatchString(record.ReviewEvidenceSHA256) ||
		!providerHashPattern.MatchString(record.PayloadSHA256) ||
		!providerSigningKeyIDPattern.MatchString(record.KeyID) ||
		!providerSignaturePattern.MatchString(record.Signature) {
		return nil, ErrProviderProofManifestIntegrity
	}
	payloadHash := sha256.Sum256([]byte(record.SignedManifest))
	if hex.EncodeToString(payloadHash[:]) != record.PayloadSHA256 {
		return nil, ErrProviderProofManifestIntegrity
	}
	manifest, err := signer.VerifyCommercialProofManifestSignature(
		record.SignedManifest, record.Signature,
	)
	if err != nil || manifest.ManifestID != record.ID ||
		manifest.ProviderPilotEpochID != record.ProviderPilotEpochID ||
		manifest.ManifestContractVersion != record.ManifestContractVersion ||
		manifest.ProofSnapshotSHA256 != record.ProofSnapshotSHA256 ||
		manifest.ReviewEvidenceSHA256 != record.ReviewEvidenceSHA256 ||
		manifest.KeyID != record.KeyID ||
		manifest.IssuedAt != record.IssuedAt.UTC().Unix() {
		return nil, ErrProviderProofManifestIntegrity
	}
	return &record, nil
}

func providerCommercialProofReviewCoverage(
	queryer providerProofQuerier,
	pilotID string,
) (providerexchange.CommercialProofReviewCoverage, string, error) {
	var coverage providerexchange.CommercialProofReviewCoverage
	rows, err := queryer.Query(providerVerifiedCommercialCTEs+`, required_reviews AS (
		SELECT DISTINCT 'provider'::text AS review_type,
		       provider_claim_id AS subject_id,
		       NULL::timestamptz AS must_be_on_or_after,
		       pilot_activated_at AS must_be_on_or_before
		FROM qualified_offers
		UNION ALL
		SELECT DISTINCT 'offer'::text, provider_offer_id,
		       NULL::timestamptz, activated_at
		FROM qualified_offers
		UNION ALL
		SELECT DISTINCT 'ticket'::text, id,
		       NULL::timestamptz, handoff_observed_at
		FROM pilot_tickets
		UNION ALL
		SELECT DISTINCT 'handoff'::text, handoff_receipt_id,
		       handoff_observed_at, NULL::timestamptz
		FROM pilot_tickets
		UNION ALL
		SELECT DISTINCT 'callback'::text, receipt.id,
		       receipt.created_at, NULL::timestamptz
		FROM outcome_receipts receipt
		JOIN pilot_tickets ticket ON ticket.id=receipt.action_ticket_id
	), review_assessment AS (
		SELECT required.review_type, required.subject_id,
		       review.id AS review_id,
		       review.review_contract_version,
		       review.subject_snapshot_sha256,
		       review.owner_reference, review.evidence_reference,
		       review.reviewed_at,
		       CASE WHEN review.id IS NULL THEN false ELSE
		           review.review_contract_version=$5
		           AND review.subject_snapshot_sha256=
		               provider_pilot_review_snapshot_sha256(
		                   $4::uuid, required.review_type, required.subject_id
		               )
		           AND (required.must_be_on_or_after IS NULL OR
		                review.reviewed_at >= required.must_be_on_or_after)
		           AND (required.must_be_on_or_before IS NULL OR
		                review.reviewed_at <= required.must_be_on_or_before)
		       END AS valid
		FROM required_reviews required
		LEFT JOIN provider_pilot_review_events review
		  ON review.provider_pilot_epoch_id=$4::uuid
		 AND review.review_type=required.review_type
		 AND review.subject_id=required.subject_id
	)
	SELECT review_type, subject_id::text, review_id::text,
	       review_contract_version, subject_snapshot_sha256,
	       owner_reference, evidence_reference, reviewed_at, valid
	FROM review_assessment
	ORDER BY review_type, subject_id`,
		int64(ProviderClaimVerificationFreshness/time.Second),
		ProviderCommercialTermsContractV1,
		ProviderActionHandoffContractV1,
		pilotID,
		ProviderPilotReviewContractV1,
	)
	if err != nil {
		return coverage, "", err
	}
	defer rows.Close()

	root := sha256.New()
	writeField := func(value string) {
		_, _ = root.Write([]byte(strconv.Itoa(len(value))))
		_, _ = root.Write([]byte{':'})
		_, _ = root.Write([]byte(value))
	}
	writeField(providerexchange.CommercialProofReviewEvidenceV1)
	writeField(pilotID)
	for rows.Next() {
		var reviewType, subjectID string
		var reviewID, reviewContract, snapshot, ownerReference, evidenceReference sql.NullString
		var reviewedAt sql.NullTime
		var valid bool
		if err := rows.Scan(
			&reviewType, &subjectID, &reviewID, &reviewContract, &snapshot,
			&ownerReference, &evidenceReference, &reviewedAt, &valid,
		); err != nil {
			return coverage, "", err
		}
		var count *providerexchange.CommercialProofReviewCount
		switch reviewType {
		case "provider":
			count = &coverage.Providers
		case "offer":
			count = &coverage.Offers
		case "ticket":
			count = &coverage.Tickets
		case "handoff":
			count = &coverage.Handoffs
		case "callback":
			count = &coverage.Callbacks
		default:
			return coverage, "", ErrProviderProofManifestIntegrity
		}
		count.Required++
		if valid {
			count.Valid++
		}
		reviewedAtMicros := ""
		if reviewedAt.Valid {
			reviewedAtMicros = strconv.FormatInt(reviewedAt.Time.UTC().UnixMicro(), 10)
		}
		writeField(reviewType)
		writeField(subjectID)
		writeField(reviewID.String)
		writeField(reviewContract.String)
		writeField(snapshot.String)
		writeField(ownerReference.String)
		writeField(evidenceReference.String)
		writeField(reviewedAtMicros)
		writeField(strconv.FormatBool(valid))
	}
	if err := rows.Err(); err != nil {
		return coverage, "", err
	}
	return coverage, hex.EncodeToString(root.Sum(nil)), nil
}

func completeProviderCommercialProofReviews(
	proof *ProviderExchangeProof,
	coverage providerexchange.CommercialProofReviewCoverage,
) bool {
	if proof == nil {
		return false
	}
	counts := []providerexchange.CommercialProofReviewCount{
		coverage.Providers, coverage.Offers, coverage.Tickets,
		coverage.Handoffs, coverage.Callbacks,
	}
	for _, count := range counts {
		if count.Required <= 0 || count.Valid != count.Required {
			return false
		}
	}
	return coverage.Providers.Required == proof.VerifiedProviderCompanies &&
		coverage.Offers.Required >= proof.VerifiedProviderCompanies &&
		coverage.Tickets.Required >= proof.VerifiedProviderAcceptedHandoffs &&
		coverage.Handoffs.Required == coverage.Tickets.Required &&
		coverage.Callbacks.Required == proof.VerifiedOutcomeReceipts
}

func buildProviderCommercialProofManifestCandidate(
	queryer providerProofQuerier,
	pilotID string,
	signer *providerexchange.Signer,
) (*ProviderCommercialProofManifestCandidate, error) {
	proof, err := getProviderExchangeProof(queryer, pilotID, signer)
	if err != nil {
		return nil, err
	}
	var pilotContractVersion string
	if err := queryer.QueryRow(`
		SELECT contract_version
		FROM provider_pilot_epochs
		WHERE id=$1::uuid`, pilotID).Scan(&pilotContractVersion); err != nil {
		return nil, err
	}
	coverage, reviewEvidenceSHA256, err := providerCommercialProofReviewCoverage(queryer, pilotID)
	if err != nil {
		return nil, err
	}
	candidate := &ProviderCommercialProofManifestCandidate{
		ManifestContractVersion:           providerexchange.CommercialProofManifestContractV1,
		SignatureVerificationScope:        providerexchange.CommercialProofVerificationScopeV1,
		ProviderPilotEpochID:              proof.ProviderPilotEpochID,
		ProviderPilotContractVersion:      pilotContractVersion,
		ReviewContractVersion:             ProviderPilotReviewContractV1,
		ReviewEvidenceContractVersion:     providerexchange.CommercialProofReviewEvidenceV1,
		MarketPolicyContractVersion:       providerexchange.CommercialProofMarketPolicyV1,
		ReviewEvidenceSHA256:              reviewEvidenceSHA256,
		PilotDemandTopic:                  proof.ProviderPilotDemandTopic,
		PilotStatus:                       proof.ProviderPilotStatus,
		OutcomeReceiptIntegrityValid:      proof.OutcomeReceiptIntegrityValid,
		VerifiedOutcomeReceipts:           proof.VerifiedOutcomeReceipts,
		RejectedOutcomeReceipts:           proof.RejectedOutcomeReceipts,
		VerifiedOutcomeLedgerEntries:      proof.VerifiedOutcomeLedgerEntries,
		RejectedOutcomeLedgerEntries:      proof.RejectedOutcomeLedgerEntries,
		VerifiedProviderCompanies:         proof.VerifiedProviderCompanies,
		VerifiedProviderAcceptedHandoffs:  proof.VerifiedProviderAcceptedHandoffs,
		VerifiedProviderActivations:       proof.VerifiedProviderConfirmedActivations,
		VerifiedProviderRenewals:          proof.VerifiedProviderRenewals,
		VerifiedProviderConversions:       proof.VerifiedProviderConfirmedConversions,
		ReviewCoverage:                    coverage,
		MonetaryAmountsWithheldForPrivacy: true,
		VerifiedPrepaidSettled:            []providerexchange.CommercialProofCurrencyAmount{},
		VerifiedPrepaidNetDebited:         []providerexchange.CommercialProofCurrencyAmount{},
		VerifiedTermsNetReceivable:        []providerexchange.CommercialProofCurrencyAmount{},
		PilotThresholdsMet:                proof.PilotThresholdsMet,
		OrganicRankSold:                   false,
		RawQueriesSold:                    false,
		AgentIdentitiesSold:               false,
		IssuanceBlockers:                  []string{},
		EvidenceScope:                     providerexchange.CommercialProofManifestScopeV1,
	}
	candidate.ReviewIntegrityValid = completeProviderCommercialProofReviews(proof, coverage)
	if candidate.PilotStatus != "closed" {
		candidate.IssuanceBlockers = append(candidate.IssuanceBlockers, "pilot_not_closed")
	}
	if !candidate.OutcomeReceiptIntegrityValid ||
		candidate.RejectedOutcomeReceipts != 0 ||
		candidate.RejectedOutcomeLedgerEntries != 0 {
		candidate.IssuanceBlockers = append(candidate.IssuanceBlockers, "outcome_integrity_failed")
	}
	if !candidate.PilotThresholdsMet {
		candidate.IssuanceBlockers = append(candidate.IssuanceBlockers, "commercial_thresholds_not_met")
	}
	if !candidate.ReviewIntegrityValid {
		candidate.IssuanceBlockers = append(candidate.IssuanceBlockers, "chronological_review_incomplete")
	}
	candidate.Issuable = len(candidate.IssuanceBlockers) == 0
	hashInput := *candidate
	hashInput.ProofSnapshotSHA256 = ""
	encoded, err := json.Marshal(hashInput)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(encoded)
	candidate.ProofSnapshotSHA256 = hex.EncodeToString(digest[:])
	return candidate, nil
}

// GetProviderCommercialProofManifestCandidate returns a repeatable-read owner
// preview. Previewing never signs, stores, publishes, or creates proof.
func GetProviderCommercialProofManifestCandidate(
	db *sql.DB,
	pilotID string,
	signer *providerexchange.Signer,
) (*ProviderCommercialProofManifestCandidate, error) {
	pilotID = strings.ToLower(strings.TrimSpace(pilotID))
	if db == nil || signer == nil || !validProviderUUID(pilotID) {
		return nil, ErrInvalidProviderExchange
	}
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	candidate, err := buildProviderCommercialProofManifestCandidate(tx, pilotID, signer)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return candidate, nil
}

// GetProviderCommercialProofManifest retrieves and cryptographically verifies
// the immutable signed aggregate for one exact pilot.
func GetProviderCommercialProofManifest(
	db *sql.DB,
	pilotID string,
	signer *providerexchange.Signer,
) (*ProviderCommercialProofManifestRecord, error) {
	pilotID = strings.ToLower(strings.TrimSpace(pilotID))
	if db == nil || signer == nil || !validProviderUUID(pilotID) {
		return nil, ErrInvalidProviderExchange
	}
	return scanProviderCommercialProofManifest(db.QueryRow(`
		SELECT `+providerCommercialProofManifestColumns+`
		FROM provider_commercial_proof_manifests
		WHERE provider_pilot_epoch_id=$1::uuid`, pilotID), signer)
}

func providerCommercialProofManifestFromCandidate(
	candidate *ProviderCommercialProofManifestCandidate,
	manifestID, keyID string,
	issuedAt time.Time,
) providerexchange.CommercialProofManifest {
	return providerexchange.CommercialProofManifest{
		Version:                           providerexchange.CommercialProofManifestVersion,
		KeyID:                             keyID,
		SignatureVerificationScope:        candidate.SignatureVerificationScope,
		ManifestContractVersion:           candidate.ManifestContractVersion,
		ManifestID:                        manifestID,
		ProviderPilotEpochID:              candidate.ProviderPilotEpochID,
		ProviderPilotContractVersion:      candidate.ProviderPilotContractVersion,
		ReviewContractVersion:             candidate.ReviewContractVersion,
		ReviewEvidenceContractVersion:     candidate.ReviewEvidenceContractVersion,
		MarketPolicyContractVersion:       candidate.MarketPolicyContractVersion,
		ProofSnapshotSHA256:               candidate.ProofSnapshotSHA256,
		ReviewEvidenceSHA256:              candidate.ReviewEvidenceSHA256,
		PilotDemandTopic:                  candidate.PilotDemandTopic,
		PilotStatus:                       candidate.PilotStatus,
		IssuedAt:                          issuedAt.UTC().Unix(),
		OutcomeReceiptIntegrityValid:      candidate.OutcomeReceiptIntegrityValid,
		ReviewIntegrityValid:              candidate.ReviewIntegrityValid,
		VerifiedOutcomeReceipts:           candidate.VerifiedOutcomeReceipts,
		RejectedOutcomeReceipts:           candidate.RejectedOutcomeReceipts,
		VerifiedOutcomeLedgerEntries:      candidate.VerifiedOutcomeLedgerEntries,
		RejectedOutcomeLedgerEntries:      candidate.RejectedOutcomeLedgerEntries,
		VerifiedProviderCompanies:         candidate.VerifiedProviderCompanies,
		VerifiedProviderAcceptedHandoffs:  candidate.VerifiedProviderAcceptedHandoffs,
		VerifiedProviderActivations:       candidate.VerifiedProviderActivations,
		VerifiedProviderRenewals:          candidate.VerifiedProviderRenewals,
		VerifiedProviderConversions:       candidate.VerifiedProviderConversions,
		ReviewCoverage:                    candidate.ReviewCoverage,
		MonetaryAmountsWithheldForPrivacy: candidate.MonetaryAmountsWithheldForPrivacy,
		VerifiedPrepaidSettled:            candidate.VerifiedPrepaidSettled,
		VerifiedPrepaidNetDebited:         candidate.VerifiedPrepaidNetDebited,
		VerifiedTermsNetReceivable:        candidate.VerifiedTermsNetReceivable,
		PilotThresholdsMet:                candidate.PilotThresholdsMet,
		OrganicRankSold:                   false,
		RawQueriesSold:                    false,
		AgentIdentitiesSold:               false,
		EvidenceScope:                     candidate.EvidenceScope,
	}
}

// IssueProviderCommercialProofManifest is the sole signing path. It locks the
// exact pilot, re-evaluates proof and chronological reviews in one serializable
// transaction, compares the owner-reviewed digest, and inserts one append-only
// manifest. Exact retries return the same stored signature.
func IssueProviderCommercialProofManifest(
	db *sql.DB,
	input ProviderCommercialProofManifestInput,
	signer *providerexchange.Signer,
) (*ProviderCommercialProofManifestRecord, bool, error) {
	pilotID := strings.ToLower(strings.TrimSpace(input.ProviderPilotEpochID))
	input.ExpectedSnapshotSHA256 = strings.ToLower(strings.TrimSpace(input.ExpectedSnapshotSHA256))
	input.OwnerReference = strings.TrimSpace(input.OwnerReference)
	input.EvidenceReference = strings.TrimSpace(input.EvidenceReference)
	if db == nil || signer == nil || !validProviderUUID(pilotID) ||
		!providerHashPattern.MatchString(input.ExpectedSnapshotSHA256) ||
		!providerReferencePattern.MatchString(input.OwnerReference) ||
		!providerReferencePattern.MatchString(input.EvidenceReference) {
		return nil, false, ErrInvalidProviderExchange
	}
	// A serializable transaction takes its snapshot before a contended
	// SELECT FOR UPDATE finishes waiting. Two first-time, identical issuers can
	// therefore produce one committed manifest and one PostgreSQL 40001 even
	// though the second request is an exact replay. Retry only that transient
	// serialization result (or the exact per-pilot unique race) so the fresh
	// transaction can read and verify the immutable winner. All business and
	// integrity failures remain single-attempt and fail closed.
	for attempt := 0; attempt < 3; attempt++ {
		record, created, err := issueProviderCommercialProofManifestOnce(
			db, pilotID, input, signer,
		)
		if err == nil || !retryableProviderProofManifestIssue(err) || attempt == 2 {
			return record, created, err
		}
	}
	return nil, false, ErrProviderProofManifestIntegrity
}

func issueProviderCommercialProofManifestOnce(
	db *sql.DB,
	pilotID string,
	input ProviderCommercialProofManifestInput,
	signer *providerexchange.Signer,
) (*ProviderCommercialProofManifestRecord, bool, error) {
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	var lockedPilotID string
	if err := tx.QueryRow(`
		SELECT id::text FROM provider_pilot_epochs
		WHERE id=$1::uuid
		FOR UPDATE`, pilotID).Scan(&lockedPilotID); err != nil {
		return nil, false, err
	}
	existing, err := scanProviderCommercialProofManifest(tx.QueryRow(`
		SELECT `+providerCommercialProofManifestColumns+`
		FROM provider_commercial_proof_manifests
		WHERE provider_pilot_epoch_id=$1::uuid`, pilotID), signer)
	if err == nil {
		if existing.ProofSnapshotSHA256 != input.ExpectedSnapshotSHA256 ||
			existing.OwnerReference != input.OwnerReference ||
			existing.EvidenceReference != input.EvidenceReference {
			return nil, false, ErrProviderProofManifestRequestConflict
		}
		existing.Replayed = true
		if err := tx.Commit(); err != nil {
			return nil, false, err
		}
		return existing, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}
	candidate, err := buildProviderCommercialProofManifestCandidate(tx, pilotID, signer)
	if err != nil {
		return nil, false, err
	}
	if candidate.ProofSnapshotSHA256 != input.ExpectedSnapshotSHA256 {
		return nil, false, ErrProviderProofManifestSnapshotChanged
	}
	if !candidate.Issuable {
		return nil, false, ErrProviderProofManifestNotIssuable
	}
	manifestID, err := newProviderUUID()
	if err != nil {
		return nil, false, err
	}
	var issuedAt time.Time
	if err := tx.QueryRow(`
		SELECT date_trunc('second', transaction_timestamp())`).Scan(&issuedAt); err != nil {
		return nil, false, err
	}
	manifest := providerCommercialProofManifestFromCandidate(
		candidate, manifestID, signer.ActiveKeyID(), issuedAt,
	)
	signedManifest, signature, err := signer.SignCommercialProofManifest(manifest)
	if err != nil {
		return nil, false, err
	}
	payloadDigest := sha256.Sum256([]byte(signedManifest))
	record, err := scanProviderCommercialProofManifest(tx.QueryRow(`
		INSERT INTO provider_commercial_proof_manifests (
			id, provider_pilot_epoch_id, manifest_contract_version,
			proof_snapshot_sha256, review_evidence_sha256,
			key_id, signed_manifest, signature,
			payload_sha256, owner_reference, evidence_reference,
			issued_at, created_at
		) VALUES (
			$1::uuid,$2::uuid,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$12
		)
		RETURNING `+providerCommercialProofManifestColumns,
		manifestID, pilotID, candidate.ManifestContractVersion,
		candidate.ProofSnapshotSHA256, candidate.ReviewEvidenceSHA256,
		signer.ActiveKeyID(), signedManifest, signature,
		hex.EncodeToString(payloadDigest[:]), input.OwnerReference,
		input.EvidenceReference, issuedAt,
	), signer)
	if err != nil {
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	return record, true, nil
}

func retryableProviderProofManifestIssue(err error) bool {
	var postgresError *pq.Error
	if !errors.As(err, &postgresError) {
		return false
	}
	if postgresError.Code == "40001" {
		return true
	}
	return postgresError.Code == "23505" &&
		postgresError.Constraint ==
			"provider_commercial_proof_manifests_provider_pilot_epoch_id_key"
}
