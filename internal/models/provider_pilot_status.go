package models

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

const ProviderPilotStatusMaximumItems = 100

type ProviderPilotOfferStatus struct {
	OfferID                              string     `json:"offer_id"`
	Status                               string     `json:"status"`
	Version                              int        `json:"version"`
	Name                                 string     `json:"name"`
	ActionType                           string     `json:"action_type"`
	ChargeEvent                          string     `json:"charge_event"`
	BountyCents                          int64      `json:"bounty_cents"`
	Currency                             string     `json:"currency"`
	BillingMode                          string     `json:"billing_mode"`
	TermsCreditLimitCents                *int64     `json:"terms_credit_limit_cents,omitempty"`
	TermsPeriodDays                      *int       `json:"terms_period_days,omitempty"`
	CommercialTermsContractVersion       string     `json:"commercial_terms_contract_version"`
	CommercialTermsSHA256                string     `json:"commercial_terms_sha256"`
	ProviderMORAcknowledgementRequired   bool       `json:"provider_mor_acknowledgement_required"`
	ProviderAcknowledgesMerchantOfRecord bool       `json:"provider_acknowledges_merchant_of_record"`
	LatestAcceptanceID                   string     `json:"latest_acceptance_id,omitempty"`
	LatestAcceptanceType                 string     `json:"latest_acceptance_type,omitempty"`
	LatestAcceptanceAt                   *time.Time `json:"latest_acceptance_at,omitempty"`
	LatestAcceptanceValidUntil           *time.Time `json:"latest_acceptance_valid_until,omitempty"`
	LatestAcceptanceOwnerVerified        bool       `json:"latest_acceptance_owner_verified"`
	LatestAcceptanceOwnerVerifiedAt      *time.Time `json:"latest_acceptance_owner_verified_at,omitempty"`
	CurrentTermsOwnerVerified            bool       `json:"current_terms_owner_verified"`
	CurrentTermsValidUntil               *time.Time `json:"current_terms_valid_until,omitempty"`
	RenewalEligible                      bool       `json:"renewal_eligible"`
	ActivationReady                      bool       `json:"activation_ready"`
}

type ProviderPilotRecentEvent struct {
	TicketID          string     `json:"ticket_id"`
	OfferID           string     `json:"offer_id"`
	OfferVersion      int        `json:"offer_version"`
	TicketStatus      string     `json:"ticket_status"`
	HandoffReceiptID  string     `json:"handoff_receipt_id"`
	HandoffObservedAt time.Time  `json:"handoff_observed_at"`
	OutcomeReceiptID  string     `json:"outcome_receipt_id,omitempty"`
	Outcome           string     `json:"outcome,omitempty"`
	ChargeStatus      string     `json:"charge_status,omitempty"`
	BilledCents       *int64     `json:"billed_cents,omitempty"`
	OutcomeRecordedAt *time.Time `json:"outcome_recorded_at,omitempty"`
}

type ProviderPilotStatus struct {
	AsOf                            time.Time                  `json:"as_of"`
	ProviderClaimID                 string                     `json:"provider_claim_id"`
	Domain                          string                     `json:"domain"`
	ClaimStatus                     string                     `json:"claim_status"`
	VerificationLastSucceededAt     time.Time                  `json:"verification_last_succeeded_at"`
	VerificationNextCheckAt         *time.Time                 `json:"verification_next_check_at,omitempty"`
	VerificationConsecutiveFailures int                        `json:"verification_consecutive_failures"`
	CompanyAcceptanceID             string                     `json:"company_acceptance_id,omitempty"`
	CompanyAcceptedAt               *time.Time                 `json:"company_accepted_at,omitempty"`
	CompanyOwnerVerified            bool                       `json:"company_owner_verified"`
	CompanyOwnerVerifiedAt          *time.Time                 `json:"company_owner_verified_at,omitempty"`
	Offers                          []ProviderPilotOfferStatus `json:"offers"`
	RecentObservedHandoffs          []ProviderPilotRecentEvent `json:"recent_observed_handoffs"`
}

// GetProviderPilotStatus is a claim-key-scoped continuity surface. It exposes
// only the provider's own public offer contract, acceptance/verification state,
// and events that have crossed the explicit NHS-observed handoff boundary. It
// never returns action URLs, attribution material, search receipts, controlled
// intent, queries, identities, contacts, network data, or owner company hashes.
func GetProviderPilotStatus(db *sql.DB, key *ProviderAPIKey, limit int) (*ProviderPilotStatus, error) {
	if db == nil || key == nil || key.ID < 1 || !validProviderUUID(key.ProviderClaimID) {
		return nil, ErrInvalidProviderExchange
	}
	if limit < 1 || limit > ProviderPilotStatusMaximumItems {
		limit = 25
	}
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	status := &ProviderPilotStatus{
		ProviderClaimID:        key.ProviderClaimID,
		Offers:                 []ProviderPilotOfferStatus{},
		RecentObservedHandoffs: []ProviderPilotRecentEvent{},
	}
	if err := tx.QueryRow(`SELECT clock_timestamp()`).Scan(&status.AsOf); err != nil {
		return nil, err
	}
	var nextCheck, companyAcceptedAt, companyVerifiedAt sql.NullTime
	var companyAcceptanceID sql.NullString
	if err := tx.QueryRow(`
		SELECT claim.domain_snapshot, claim.status,
		       claim.verification_last_succeeded_at,
		       claim.verification_next_check_at,
		       claim.verification_consecutive_failures,
		       accepted.id::text, accepted.provider_accepted_at,
		       company.owner_verified_at
		FROM provider_claims claim
		JOIN provider_api_keys api_key
		  ON api_key.id=$1 AND api_key.provider_claim_id=claim.id
		 AND api_key.status='active'
		LEFT JOIN provider_pilot_companies company
		  ON company.provider_claim_id=claim.id
		LEFT JOIN LATERAL (
			SELECT event.id, event.provider_accepted_at
			FROM provider_commercial_acceptance_events event
			WHERE event.provider_claim_id=claim.id
			  AND event.event_type='pilot_company'
			  AND (
				company.id IS NULL OR
				event.id=company.provider_acceptance_event_id
			  )
			ORDER BY event.provider_accepted_at DESC, event.id DESC
			LIMIT 1
		) accepted ON true
		WHERE claim.id=$2::uuid
		  AND claim.status='verified'
		  AND claim.verification_last_succeeded_at >
		      $3::timestamptz - $4::bigint * INTERVAL '1 second'`,
		key.ID, key.ProviderClaimID, status.AsOf,
		int64(ProviderClaimVerificationFreshness/time.Second)).Scan(
		&status.Domain, &status.ClaimStatus,
		&status.VerificationLastSucceededAt, &nextCheck,
		&status.VerificationConsecutiveFailures, &companyAcceptanceID,
		&companyAcceptedAt, &companyVerifiedAt,
	); err != nil {
		return nil, err
	}
	if nextCheck.Valid {
		value := nextCheck.Time
		status.VerificationNextCheckAt = &value
	}
	if companyAcceptanceID.Valid {
		status.CompanyAcceptanceID = companyAcceptanceID.String
	}
	if companyAcceptedAt.Valid {
		value := companyAcceptedAt.Time
		status.CompanyAcceptedAt = &value
	}
	if companyVerifiedAt.Valid {
		value := companyVerifiedAt.Time
		status.CompanyOwnerVerified = true
		status.CompanyOwnerVerifiedAt = &value
	}

	rows, err := tx.Query(`
		SELECT offer.id::text, offer.status, offer.version, offer.offer_name,
		       offer.action_type, offer.charge_event, offer.bounty_cents,
		       offer.currency, offer.billing_mode, offer.terms_credit_limit_cents,
		       offer.terms_period_days, offer.commercial_terms_contract_version,
		       offer.commercial_terms_sha256,
		       accepted.id::text, accepted.event_type,
		       accepted.provider_accepted_at, accepted.valid_until,
		       accepted_commitment.owner_verified_at,
		       current_terms.valid_until,
		       accepted.id IS NOT NULL AND NOT EXISTS (
			   SELECT 1 FROM provider_commercial_acceptance_events child
			   WHERE child.related_acceptance_event_id=accepted.id
		       ) AS renewal_eligible,
		       offer.status='draft' AND company.id IS NOT NULL AND
		       current_terms.id IS NOT NULL AND NOT EXISTS (
			SELECT 1 FROM provider_budget_ledger unverified
			WHERE unverified.provider_offer_id=offer.id
			  AND unverified.entry_type IN ('fund','adjustment')
			  AND NOT EXISTS (
				SELECT 1 FROM provider_commercial_commitment_events linked
				WHERE linked.budget_ledger_entry_id=unverified.id
				  AND (
					(linked.event_type='prepaid_fund' AND unverified.entry_type='fund') OR
					(linked.event_type='fund_reversal' AND unverified.entry_type='adjustment')
				  )
			  )
		       ) AS activation_ready
		FROM provider_offers offer
		JOIN provider_claims claim ON claim.id=offer.provider_claim_id
		LEFT JOIN provider_pilot_companies company
		  ON company.provider_claim_id=offer.provider_claim_id
		LEFT JOIN LATERAL (
			SELECT event.id, event.event_type, event.provider_accepted_at,
			       event.valid_until
			FROM provider_commercial_acceptance_events event
			WHERE event.provider_offer_id=offer.id
			  AND event.offer_version_snapshot=offer.version
			  AND event.terms_contract_version=offer.commercial_terms_contract_version
			  AND event.exact_terms_sha256=offer.commercial_terms_sha256
			ORDER BY event.provider_accepted_at DESC, event.id DESC
			LIMIT 1
		) accepted ON true
		LEFT JOIN provider_commercial_commitment_events accepted_commitment
		  ON accepted_commitment.provider_acceptance_event_id=accepted.id
		LEFT JOIN LATERAL (
			SELECT commitment.id, commitment.valid_until
			FROM provider_commercial_commitment_events commitment
			WHERE commitment.provider_offer_id=offer.id
			  AND commitment.event_type IN ('terms_acceptance','terms_renewal')
			  AND commitment.offer_version_snapshot=offer.version
			  AND commitment.terms_contract_version=offer.commercial_terms_contract_version
			  AND commitment.exact_terms_sha256=offer.commercial_terms_sha256
			  AND commitment.valid_until > $2::timestamptz
			ORDER BY commitment.valid_until DESC, commitment.id DESC
			LIMIT 1
		) current_terms ON true
		WHERE offer.provider_claim_id=$1::uuid
		ORDER BY offer.created_at DESC, offer.id DESC
		LIMIT $3`, key.ProviderClaimID, status.AsOf, limit)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var offer ProviderPilotOfferStatus
		var termsLimit, termsDays sql.NullInt64
		var acceptanceID, acceptanceType sql.NullString
		var acceptedAt, acceptedUntil, acceptedVerifiedAt, currentUntil sql.NullTime
		if err := rows.Scan(
			&offer.OfferID, &offer.Status, &offer.Version, &offer.Name,
			&offer.ActionType, &offer.ChargeEvent, &offer.BountyCents,
			&offer.Currency, &offer.BillingMode, &termsLimit, &termsDays,
			&offer.CommercialTermsContractVersion, &offer.CommercialTermsSHA256,
			&acceptanceID, &acceptanceType, &acceptedAt, &acceptedUntil,
			&acceptedVerifiedAt, &currentUntil, &offer.RenewalEligible,
			&offer.ActivationReady,
		); err != nil {
			_ = rows.Close()
			return nil, err
		}
		offer.ProviderMORAcknowledgementRequired = true
		offer.ProviderAcknowledgesMerchantOfRecord = offer.Status == "active"
		if termsLimit.Valid {
			value := termsLimit.Int64
			offer.TermsCreditLimitCents = &value
		}
		if termsDays.Valid {
			value := int(termsDays.Int64)
			offer.TermsPeriodDays = &value
		}
		if acceptanceID.Valid {
			offer.LatestAcceptanceID = acceptanceID.String
		}
		if acceptanceType.Valid {
			offer.LatestAcceptanceType = acceptanceType.String
		}
		if acceptedAt.Valid {
			value := acceptedAt.Time
			offer.LatestAcceptanceAt = &value
		}
		if acceptedUntil.Valid {
			value := acceptedUntil.Time
			offer.LatestAcceptanceValidUntil = &value
		}
		if acceptedVerifiedAt.Valid {
			value := acceptedVerifiedAt.Time
			offer.LatestAcceptanceOwnerVerified = true
			offer.LatestAcceptanceOwnerVerifiedAt = &value
		}
		if currentUntil.Valid {
			value := currentUntil.Time
			offer.CurrentTermsOwnerVerified = true
			offer.CurrentTermsValidUntil = &value
		}
		status.Offers = append(status.Offers, offer)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	rows, err = tx.Query(`
		SELECT ticket.id::text, ticket.provider_offer_id::text,
		       ticket.offer_version_snapshot, ticket.status,
		       handoff.id::text, handoff.observed_at,
		       outcome.id::text, outcome.outcome, outcome.charge_status,
		       outcome.billed_cents, outcome.created_at
		FROM action_tickets ticket
		JOIN provider_action_handoff_receipts handoff
		  ON handoff.action_ticket_id=ticket.id
		LEFT JOIN LATERAL (
			SELECT receipt.id, receipt.outcome, receipt.charge_status,
			       receipt.billed_cents, receipt.created_at
			FROM outcome_receipts receipt
			WHERE receipt.action_ticket_id=ticket.id
			ORDER BY receipt.created_at DESC, receipt.id DESC
			LIMIT 1
		) outcome ON true
		WHERE ticket.provider_claim_id=$1::uuid
		ORDER BY COALESCE(outcome.created_at,handoff.observed_at) DESC,
		         handoff.id DESC
		LIMIT $2`, key.ProviderClaimID, limit)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var event ProviderPilotRecentEvent
		var outcomeID, outcome, chargeStatus sql.NullString
		var billed sql.NullInt64
		var outcomeAt sql.NullTime
		if err := rows.Scan(
			&event.TicketID, &event.OfferID, &event.OfferVersion,
			&event.TicketStatus, &event.HandoffReceiptID,
			&event.HandoffObservedAt, &outcomeID, &outcome,
			&chargeStatus, &billed, &outcomeAt,
		); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if outcomeID.Valid {
			event.OutcomeReceiptID = outcomeID.String
		}
		if outcome.Valid {
			event.Outcome = outcome.String
		}
		if chargeStatus.Valid {
			event.ChargeStatus = chargeStatus.String
		}
		if billed.Valid {
			value := billed.Int64
			event.BilledCents = &value
		}
		if outcomeAt.Valid {
			value := outcomeAt.Time
			event.OutcomeRecordedAt = &value
		}
		status.RecentObservedHandoffs = append(status.RecentObservedHandoffs, event)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return status, nil
}

func GetProviderDemandAnalyticsForClaim(db *sql.DB, claimID string, days int) (map[string]any, error) {
	if db == nil || !validProviderUUID(claimID) {
		return nil, ErrInvalidProviderExchange
	}
	var domain string
	err := db.QueryRow(`
		SELECT domain_snapshot
		FROM provider_claims
		WHERE id=$1::uuid AND status='verified'
		  AND verification_last_succeeded_at >
		      NOW() - $2::bigint * INTERVAL '1 second'`,
		claimID, int64(ProviderClaimVerificationFreshness/time.Second)).Scan(&domain)
	if err != nil {
		return nil, err
	}
	return GetProviderDemandAnalytics(db, domain, days)
}

var providerPilotQueueStates = map[string]bool{
	"review_required":           true,
	"pre_event_review_required": true,
	"provider_review_required":  true,
	"offer_review_required":     true,
	"ticket_review_required":    true,
	"handoff_review_required":   true,
	"callback_review_required":  true,
	"pending_company":           true,
	"pending_terms":             true,
	"activation_review":         true,
	"expiring_terms":            true,
	"handoff_awaiting_callback": true,
	"recent_callback":           true,
}

type ProviderPilotQueueItem struct {
	State                       string     `json:"state"`
	ProviderPilotEpochID        string     `json:"provider_pilot_epoch_id,omitempty"`
	ProviderClaimID             string     `json:"provider_claim_id"`
	Domain                      string     `json:"domain"`
	OfferID                     string     `json:"offer_id,omitempty"`
	OfferVersion                *int       `json:"offer_version,omitempty"`
	CommercialTermsSHA256       string     `json:"commercial_terms_sha256,omitempty"`
	CommitmentEventID           string     `json:"commitment_event_id,omitempty"`
	CommitmentEventType         string     `json:"commitment_event_type,omitempty"`
	AcceptanceEventID           string     `json:"acceptance_event_id,omitempty"`
	AcceptanceEventType         string     `json:"acceptance_event_type,omitempty"`
	RelatedAcceptanceEventID    string     `json:"related_acceptance_event_id,omitempty"`
	RelatedCommitmentEventID    string     `json:"related_commitment_event_id,omitempty"`
	ProviderAcceptanceReference string     `json:"provider_acceptance_reference,omitempty"`
	TicketID                    string     `json:"ticket_id,omitempty"`
	HandoffReceiptID            string     `json:"handoff_receipt_id,omitempty"`
	OutcomeReceiptID            string     `json:"outcome_receipt_id,omitempty"`
	Outcome                     string     `json:"outcome,omitempty"`
	ChargeStatus                string     `json:"charge_status,omitempty"`
	ReviewType                  string     `json:"review_type,omitempty"`
	SubjectID                   string     `json:"subject_id,omitempty"`
	SubjectSnapshotSHA256       string     `json:"subject_snapshot_sha256,omitempty"`
	OccurredAt                  time.Time  `json:"occurred_at"`
	ValidUntil                  *time.Time `json:"valid_until,omitempty"`
}

type ProviderPilotQueue struct {
	AsOf              time.Time                `json:"as_of"`
	State             string                   `json:"state"`
	LimitPerState     int                      `json:"limit_per_state"`
	ReturnedCounts    map[string]int           `json:"returned_counts"`
	Items             []ProviderPilotQueueItem `json:"items"`
	RedactionContract string                   `json:"redaction_contract"`
}

// GetProviderPilotQueue gives the owner enough opaque IDs to perform the
// existing evidence-gated mutations without SQL. It deliberately excludes all
// raw credentials, attribution material, company-key hashes, controlled intent,
// queries, identities, contacts, network data, and action URLs.
func GetProviderPilotQueue(db *sql.DB, state string, limit int) (*ProviderPilotQueue, error) {
	state = strings.ToLower(strings.TrimSpace(state))
	if state == "" {
		state = "all"
	}
	if db == nil || (state != "all" && !providerPilotQueueStates[state]) {
		return nil, ErrInvalidProviderExchange
	}
	if limit < 1 || limit > ProviderPilotStatusMaximumItems {
		limit = 25
	}
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	queue := &ProviderPilotQueue{
		State: state, LimitPerState: limit, ReturnedCounts: map[string]int{},
		Items:             []ProviderPilotQueueItem{},
		RedactionContract: "Opaque pilot workflow and review-subject IDs, current review snapshot digests, public provider domain, exact offer contract, state, and event times only; no credentials, attribution material, company hashes, controlled intent, queries, identities, contacts, network data, or action URLs.",
	}
	if err := tx.QueryRow(`SELECT clock_timestamp()`).Scan(&queue.AsOf); err != nil {
		return nil, err
	}
	wants := func(candidate string) bool {
		return state == "all" || state == candidate ||
			(state == "review_required" && strings.HasSuffix(candidate, "_review_required")) ||
			(state == "pre_event_review_required" &&
				(candidate == "provider_review_required" ||
					candidate == "offer_review_required" ||
					candidate == "ticket_review_required"))
	}
	appendRows := func(rows *sql.Rows, scan func(*sql.Rows) (ProviderPilotQueueItem, error)) error {
		defer rows.Close()
		for rows.Next() {
			item, err := scan(rows)
			if err != nil {
				return err
			}
			queue.ReturnedCounts[item.State]++
			queue.Items = append(queue.Items, item)
		}
		return rows.Err()
	}

	if wants("provider_review_required") {
		rows, err := tx.Query(`
			SELECT epoch.id::text, enrollment.provider_claim_id::text,
			       claim.domain_snapshot,
			       provider_pilot_review_snapshot_sha256(
			           epoch.id,'provider',enrollment.provider_claim_id
			       ), enrollment.enrolled_at
			FROM provider_pilot_epochs epoch
			JOIN provider_pilot_enrollments enrollment
			  ON enrollment.provider_pilot_epoch_id=epoch.id
			JOIN provider_claims claim ON claim.id=enrollment.provider_claim_id
			WHERE epoch.status='draft'
			  AND claim.status='verified'
			  AND claim.verification_last_succeeded_at >
			      statement_timestamp() - $1::bigint * INTERVAL '1 second'
			  AND NOT EXISTS (
				SELECT 1 FROM provider_pilot_review_events review
				WHERE review.provider_pilot_epoch_id=epoch.id
				  AND review.review_type='provider'
				  AND review.subject_id=enrollment.provider_claim_id
				  AND review.review_contract_version=$2
				  AND review.subject_snapshot_sha256=
				      provider_pilot_review_snapshot_sha256(
				          epoch.id,'provider',enrollment.provider_claim_id
				      )
			  )
			ORDER BY enrollment.enrolled_at ASC, enrollment.id ASC
			LIMIT $3`, int64(ProviderClaimVerificationFreshness/time.Second),
			ProviderPilotReviewContractV1, limit)
		if err != nil {
			return nil, err
		}
		if err := appendRows(rows, func(row *sql.Rows) (ProviderPilotQueueItem, error) {
			item := ProviderPilotQueueItem{
				State: "provider_review_required", ReviewType: "provider",
			}
			err := row.Scan(&item.ProviderPilotEpochID, &item.ProviderClaimID,
				&item.Domain, &item.SubjectSnapshotSHA256, &item.OccurredAt)
			item.SubjectID = item.ProviderClaimID
			return item, err
		}); err != nil {
			return nil, err
		}
	}

	if wants("offer_review_required") {
		rows, err := tx.Query(`
			SELECT epoch.id::text, offer.provider_claim_id::text,
			       claim.domain_snapshot, offer.id::text,
			       offer.version, offer.commercial_terms_sha256,
			       commitment.id::text, commitment.event_type,
			       provider_pilot_review_snapshot_sha256(
			           epoch.id,'offer',offer.id
			       ), commitment.owner_verified_at
			FROM provider_pilot_epochs epoch
			JOIN provider_pilot_enrollments enrollment
			  ON enrollment.provider_pilot_epoch_id=epoch.id
			JOIN provider_claims claim ON claim.id=enrollment.provider_claim_id
			JOIN provider_offers offer
			  ON offer.provider_claim_id=enrollment.provider_claim_id
			 AND (offer.provider_pilot_epoch_id IS NULL OR
			      offer.provider_pilot_epoch_id=epoch.id)
			JOIN LATERAL (
				SELECT event.*
				FROM provider_commercial_commitment_events event
				WHERE event.provider_pilot_company_id=enrollment.provider_pilot_company_id
				  AND event.provider_claim_id=offer.provider_claim_id
				  AND event.provider_offer_id=offer.id
				  AND event.offer_version_snapshot=offer.version
				  AND event.terms_contract_version=
				      offer.commercial_terms_contract_version
				  AND event.exact_terms_sha256=offer.commercial_terms_sha256
				  AND (
					(offer.billing_mode='prepaid'
					 AND event.event_type='prepaid_fund'
					 AND event.amount_cents + COALESCE((
						SELECT SUM(reversal.amount_cents)
						FROM provider_commercial_commitment_events reversal
						WHERE reversal.related_event_id=event.id
						  AND reversal.event_type='fund_reversal'
					 ),0) > 0) OR
					(offer.billing_mode='terms'
					 AND event.event_type IN ('terms_acceptance','terms_renewal')
					 AND event.valid_until > statement_timestamp())
				  )
				ORDER BY event.owner_verified_at ASC,
				         event.provider_accepted_at ASC, event.id ASC
				LIMIT 1
			) commitment ON true
			WHERE epoch.status IN ('draft','active')
			  AND offer.status IN ('draft','paused')
			  AND claim.status='verified'
			  AND claim.verification_last_succeeded_at >
			      statement_timestamp() - $1::bigint * INTERVAL '1 second'
			  AND offer.commercial_terms_contract_version=$3
			  AND offer.commercial_terms_sha256 ~ '^[0-9a-f]{64}$'
			  AND NOT EXISTS (
				SELECT 1 FROM provider_budget_ledger unverified
				WHERE unverified.provider_offer_id=offer.id
				  AND unverified.entry_type IN ('fund','adjustment')
				  AND NOT EXISTS (
					SELECT 1 FROM provider_commercial_commitment_events linked
					WHERE linked.budget_ledger_entry_id=unverified.id
					  AND (
						(linked.event_type='prepaid_fund' AND unverified.entry_type='fund') OR
						(linked.event_type='fund_reversal' AND unverified.entry_type='adjustment')
					  )
				  )
			  )
			  AND NOT EXISTS (
				SELECT 1 FROM provider_pilot_review_events review
				WHERE review.provider_pilot_epoch_id=epoch.id
				  AND review.review_type='offer'
				  AND review.subject_id=offer.id
				  AND review.review_contract_version=$2
				  AND review.subject_snapshot_sha256=
				      provider_pilot_review_snapshot_sha256(
				          epoch.id,'offer',offer.id
				      )
			  )
			ORDER BY commitment.owner_verified_at ASC, offer.id ASC
			LIMIT $4`, int64(ProviderClaimVerificationFreshness/time.Second),
			ProviderPilotReviewContractV1,
			ProviderCommercialTermsContractV1, limit)
		if err != nil {
			return nil, err
		}
		if err := appendRows(rows, func(row *sql.Rows) (ProviderPilotQueueItem, error) {
			item := ProviderPilotQueueItem{
				State: "offer_review_required", ReviewType: "offer",
			}
			var version int
			err := row.Scan(&item.ProviderPilotEpochID, &item.ProviderClaimID,
				&item.Domain, &item.OfferID, &version,
				&item.CommercialTermsSHA256, &item.CommitmentEventID,
				&item.CommitmentEventType, &item.SubjectSnapshotSHA256,
				&item.OccurredAt)
			item.OfferVersion = &version
			item.SubjectID = item.OfferID
			return item, err
		}); err != nil {
			return nil, err
		}
	}

	if wants("ticket_review_required") {
		rows, err := tx.Query(`
			SELECT ticket.provider_pilot_epoch_id::text,
			       ticket.provider_claim_id::text, claim.domain_snapshot,
			       ticket.provider_offer_id::text,
			       ticket.offer_version_snapshot,
			       ticket.commercial_terms_sha256_snapshot,
			       ticket.id::text,
			       provider_pilot_review_snapshot_sha256(
			           ticket.provider_pilot_epoch_id,'ticket',ticket.id
			       ), ticket.created_at, ticket.expires_at
			FROM action_tickets ticket
			JOIN provider_pilot_epochs epoch
			  ON epoch.id=ticket.provider_pilot_epoch_id AND epoch.status='active'
			JOIN provider_claims claim ON claim.id=ticket.provider_claim_id
			WHERE ticket.status='created'
			  AND NOT ticket.source_is_synthetic
			  AND ticket.intent_redacted_at IS NULL
			  AND ticket.authorization_revoked_at IS NULL
			  AND ticket.expires_at > $1::timestamptz
			  AND claim.status='verified'
			  AND claim.verification_last_succeeded_at >
			      $1::timestamptz - $2::bigint * INTERVAL '1 second'
			  AND NOT EXISTS (
				SELECT 1 FROM provider_pilot_review_events review
				WHERE review.provider_pilot_epoch_id=ticket.provider_pilot_epoch_id
				  AND review.review_type='ticket'
				  AND review.subject_id=ticket.id
				  AND review.review_contract_version=$3
				  AND review.subject_snapshot_sha256=
				      provider_pilot_review_snapshot_sha256(
				          ticket.provider_pilot_epoch_id,'ticket',ticket.id
				      )
			  )
			ORDER BY ticket.created_at ASC, ticket.id ASC
			LIMIT $4`, queue.AsOf,
			int64(ProviderClaimVerificationFreshness/time.Second),
			ProviderPilotReviewContractV1, limit)
		if err != nil {
			return nil, err
		}
		if err := appendRows(rows, func(row *sql.Rows) (ProviderPilotQueueItem, error) {
			item := ProviderPilotQueueItem{
				State: "ticket_review_required", ReviewType: "ticket",
			}
			var version int
			var validUntil time.Time
			err := row.Scan(&item.ProviderPilotEpochID, &item.ProviderClaimID,
				&item.Domain, &item.OfferID, &version,
				&item.CommercialTermsSHA256, &item.TicketID,
				&item.SubjectSnapshotSHA256, &item.OccurredAt, &validUntil)
			item.OfferVersion = &version
			item.SubjectID = item.TicketID
			item.ValidUntil = &validUntil
			return item, err
		}); err != nil {
			return nil, err
		}
	}

	if wants("handoff_review_required") {
		rows, err := tx.Query(`
			SELECT ticket.provider_pilot_epoch_id::text,
			       handoff.provider_claim_id::text, claim.domain_snapshot,
			       handoff.provider_offer_id::text,
			       handoff.offer_version_snapshot,
			       handoff.commercial_terms_sha256_snapshot,
			       ticket.id::text, handoff.id::text,
			       provider_pilot_review_snapshot_sha256(
			           ticket.provider_pilot_epoch_id,'handoff',handoff.id
			       ), handoff.observed_at
			FROM provider_action_handoff_receipts handoff
			JOIN action_tickets ticket ON ticket.id=handoff.action_ticket_id
			JOIN provider_pilot_epochs epoch
			  ON epoch.id=ticket.provider_pilot_epoch_id
			JOIN provider_claims claim ON claim.id=handoff.provider_claim_id
			WHERE NOT ticket.source_is_synthetic
			  AND NOT EXISTS (
				SELECT 1 FROM provider_pilot_review_events review
				WHERE review.provider_pilot_epoch_id=ticket.provider_pilot_epoch_id
				  AND review.review_type='handoff'
				  AND review.subject_id=handoff.id
				  AND review.review_contract_version=$1
				  AND review.subject_snapshot_sha256=
				      provider_pilot_review_snapshot_sha256(
				          ticket.provider_pilot_epoch_id,'handoff',handoff.id
				      )
			  )
			ORDER BY handoff.observed_at ASC, handoff.id ASC
			LIMIT $2`, ProviderPilotReviewContractV1, limit)
		if err != nil {
			return nil, err
		}
		if err := appendRows(rows, func(row *sql.Rows) (ProviderPilotQueueItem, error) {
			item := ProviderPilotQueueItem{
				State: "handoff_review_required", ReviewType: "handoff",
			}
			var version int
			err := row.Scan(&item.ProviderPilotEpochID, &item.ProviderClaimID,
				&item.Domain, &item.OfferID, &version,
				&item.CommercialTermsSHA256, &item.TicketID,
				&item.HandoffReceiptID, &item.SubjectSnapshotSHA256,
				&item.OccurredAt)
			item.OfferVersion = &version
			item.SubjectID = item.HandoffReceiptID
			return item, err
		}); err != nil {
			return nil, err
		}
	}

	if wants("callback_review_required") {
		rows, err := tx.Query(`
			SELECT ticket.provider_pilot_epoch_id::text,
			       outcome.provider_claim_id::text, claim.domain_snapshot,
			       outcome.provider_offer_id::text,
			       ticket.offer_version_snapshot,
			       ticket.commercial_terms_sha256_snapshot,
			       ticket.id::text, handoff.id::text, outcome.id::text,
			       outcome.outcome, outcome.charge_status,
			       provider_pilot_review_snapshot_sha256(
			           ticket.provider_pilot_epoch_id,'callback',outcome.id
			       ), outcome.created_at
			FROM outcome_receipts outcome
			JOIN action_tickets ticket ON ticket.id=outcome.action_ticket_id
			JOIN provider_action_handoff_receipts handoff
			  ON handoff.action_ticket_id=ticket.id
			JOIN provider_pilot_epochs epoch
			  ON epoch.id=ticket.provider_pilot_epoch_id
			JOIN provider_claims claim ON claim.id=outcome.provider_claim_id
			WHERE NOT ticket.source_is_synthetic
			  AND NOT EXISTS (
				SELECT 1 FROM provider_pilot_review_events review
				WHERE review.provider_pilot_epoch_id=ticket.provider_pilot_epoch_id
				  AND review.review_type='callback'
				  AND review.subject_id=outcome.id
				  AND review.review_contract_version=$1
				  AND review.subject_snapshot_sha256=
				      provider_pilot_review_snapshot_sha256(
				          ticket.provider_pilot_epoch_id,'callback',outcome.id
				      )
			  )
			ORDER BY outcome.created_at ASC, outcome.id ASC
			LIMIT $2`, ProviderPilotReviewContractV1, limit)
		if err != nil {
			return nil, err
		}
		if err := appendRows(rows, func(row *sql.Rows) (ProviderPilotQueueItem, error) {
			item := ProviderPilotQueueItem{
				State: "callback_review_required", ReviewType: "callback",
			}
			var version int
			err := row.Scan(&item.ProviderPilotEpochID, &item.ProviderClaimID,
				&item.Domain, &item.OfferID, &version,
				&item.CommercialTermsSHA256, &item.TicketID,
				&item.HandoffReceiptID, &item.OutcomeReceiptID,
				&item.Outcome, &item.ChargeStatus,
				&item.SubjectSnapshotSHA256, &item.OccurredAt)
			item.OfferVersion = &version
			item.SubjectID = item.OutcomeReceiptID
			return item, err
		}); err != nil {
			return nil, err
		}
	}

	if wants("pending_company") {
		rows, err := tx.Query(`
			SELECT accepted.provider_claim_id::text, claim.domain_snapshot,
			       accepted.id::text, accepted.event_type,
			       accepted.provider_acceptance_reference,
			       accepted.provider_accepted_at
			FROM provider_commercial_acceptance_events accepted
			JOIN provider_claims claim ON claim.id=accepted.provider_claim_id
			WHERE accepted.event_type='pilot_company'
			  AND claim.status='verified'
			  AND claim.verification_last_succeeded_at >
			      $1::timestamptz - $2::bigint * INTERVAL '1 second'
			  AND NOT EXISTS (
				SELECT 1 FROM provider_pilot_companies company
				WHERE company.provider_claim_id=accepted.provider_claim_id
			  )
			ORDER BY accepted.provider_accepted_at ASC, accepted.id ASC
			LIMIT $3`, queue.AsOf,
			int64(ProviderClaimVerificationFreshness/time.Second), limit)
		if err != nil {
			return nil, err
		}
		if err := appendRows(rows, func(row *sql.Rows) (ProviderPilotQueueItem, error) {
			item := ProviderPilotQueueItem{State: "pending_company"}
			err := row.Scan(&item.ProviderClaimID, &item.Domain,
				&item.AcceptanceEventID, &item.AcceptanceEventType,
				&item.ProviderAcceptanceReference, &item.OccurredAt)
			return item, err
		}); err != nil {
			return nil, err
		}
	}

	if wants("pending_terms") {
		rows, err := tx.Query(`
			SELECT accepted.provider_claim_id::text, claim.domain_snapshot,
			       accepted.provider_offer_id::text,
			       accepted.offer_version_snapshot,
			       accepted.exact_terms_sha256, accepted.id::text,
			       accepted.event_type,
			       accepted.related_acceptance_event_id::text,
			       related_commitment.id::text,
			       accepted.provider_acceptance_reference,
			       accepted.provider_accepted_at, accepted.valid_until
			FROM provider_commercial_acceptance_events accepted
			JOIN provider_claims claim ON claim.id=accepted.provider_claim_id
			JOIN provider_offers offer
			  ON offer.id=accepted.provider_offer_id
			 AND offer.provider_claim_id=accepted.provider_claim_id
			LEFT JOIN provider_commercial_commitment_events commitment
			  ON commitment.provider_acceptance_event_id=accepted.id
			LEFT JOIN provider_commercial_commitment_events related_commitment
			  ON related_commitment.provider_acceptance_event_id=
			     accepted.related_acceptance_event_id
			 AND related_commitment.provider_claim_id=accepted.provider_claim_id
			 AND related_commitment.provider_offer_id=accepted.provider_offer_id
			 AND related_commitment.event_type IN ('terms_acceptance','terms_renewal')
			 AND related_commitment.offer_version_snapshot=accepted.offer_version_snapshot
			 AND related_commitment.terms_contract_version=accepted.terms_contract_version
			 AND related_commitment.exact_terms_sha256=accepted.exact_terms_sha256
			WHERE accepted.event_type IN ('terms_acceptance','terms_renewal')
			  AND commitment.id IS NULL
			  AND (accepted.event_type='terms_acceptance' OR related_commitment.id IS NOT NULL)
			  AND claim.status='verified'
			  AND claim.verification_last_succeeded_at >
			      $1::timestamptz - $2::bigint * INTERVAL '1 second'
			  AND accepted.offer_version_snapshot=offer.version
			  AND accepted.terms_contract_version=offer.commercial_terms_contract_version
			  AND accepted.exact_terms_sha256=offer.commercial_terms_sha256
			  AND accepted.valid_until > $1::timestamptz
			ORDER BY accepted.provider_accepted_at ASC, accepted.id ASC
			LIMIT $3`, queue.AsOf,
			int64(ProviderClaimVerificationFreshness/time.Second), limit)
		if err != nil {
			return nil, err
		}
		if err := appendRows(rows, func(row *sql.Rows) (ProviderPilotQueueItem, error) {
			item := ProviderPilotQueueItem{State: "pending_terms"}
			var version sql.NullInt64
			var related, relatedCommitment sql.NullString
			var validUntil sql.NullTime
			err := row.Scan(&item.ProviderClaimID, &item.Domain, &item.OfferID,
				&version, &item.CommercialTermsSHA256, &item.AcceptanceEventID,
				&item.AcceptanceEventType, &related, &relatedCommitment,
				&item.ProviderAcceptanceReference, &item.OccurredAt, &validUntil)
			if version.Valid {
				value := int(version.Int64)
				item.OfferVersion = &value
			}
			if related.Valid {
				item.RelatedAcceptanceEventID = related.String
			}
			if relatedCommitment.Valid {
				item.RelatedCommitmentEventID = relatedCommitment.String
			}
			if validUntil.Valid {
				value := validUntil.Time
				item.ValidUntil = &value
			}
			return item, err
		}); err != nil {
			return nil, err
		}
	}

	if wants("activation_review") {
		rows, err := tx.Query(`
			SELECT offer.provider_claim_id::text, claim.domain_snapshot,
			       offer.id::text, offer.version, offer.commercial_terms_sha256,
			       commitment.provider_acceptance_event_id::text,
			       commitment.event_type, commitment.owner_verified_at,
			       commitment.valid_until
			FROM provider_offers offer
			JOIN provider_claims claim ON claim.id=offer.provider_claim_id
			JOIN provider_pilot_companies company
			  ON company.provider_claim_id=offer.provider_claim_id
			JOIN LATERAL (
				SELECT event.provider_acceptance_event_id, event.event_type,
				       event.owner_verified_at, event.valid_until
				FROM provider_commercial_commitment_events event
				WHERE event.provider_offer_id=offer.id
				  AND event.event_type IN ('terms_acceptance','terms_renewal')
				  AND event.offer_version_snapshot=offer.version
				  AND event.exact_terms_sha256=offer.commercial_terms_sha256
				  AND event.valid_until > $1::timestamptz
				ORDER BY event.valid_until DESC, event.id DESC LIMIT 1
			) commitment ON true
			WHERE offer.status='draft' AND claim.status='verified'
			  AND claim.verification_last_succeeded_at >
			      $1::timestamptz - $2::bigint * INTERVAL '1 second'
			  AND NOT EXISTS (
				SELECT 1 FROM provider_budget_ledger unverified
				WHERE unverified.provider_offer_id=offer.id
				  AND unverified.entry_type IN ('fund','adjustment')
				  AND NOT EXISTS (
					SELECT 1 FROM provider_commercial_commitment_events linked
					WHERE linked.budget_ledger_entry_id=unverified.id
					  AND (
						(linked.event_type='prepaid_fund' AND unverified.entry_type='fund') OR
						(linked.event_type='fund_reversal' AND unverified.entry_type='adjustment')
					  )
				  )
			  )
			ORDER BY commitment.owner_verified_at ASC, offer.id ASC
			LIMIT $3`, queue.AsOf,
			int64(ProviderClaimVerificationFreshness/time.Second), limit)
		if err != nil {
			return nil, err
		}
		if err := appendRows(rows, func(row *sql.Rows) (ProviderPilotQueueItem, error) {
			item := ProviderPilotQueueItem{State: "activation_review"}
			var version int
			var validUntil sql.NullTime
			err := row.Scan(&item.ProviderClaimID, &item.Domain, &item.OfferID,
				&version, &item.CommercialTermsSHA256, &item.AcceptanceEventID,
				&item.AcceptanceEventType, &item.OccurredAt, &validUntil)
			item.OfferVersion = &version
			if validUntil.Valid {
				value := validUntil.Time
				item.ValidUntil = &value
			}
			return item, err
		}); err != nil {
			return nil, err
		}
	}

	if wants("expiring_terms") {
		rows, err := tx.Query(`
			SELECT offer.provider_claim_id::text, claim.domain_snapshot,
			       offer.id::text, offer.version, offer.commercial_terms_sha256,
			       commitment.provider_acceptance_event_id::text,
			       commitment.event_type, commitment.owner_verified_at,
			       commitment.valid_until
			FROM provider_offers offer
			JOIN provider_claims claim ON claim.id=offer.provider_claim_id
			JOIN LATERAL (
				SELECT event.provider_acceptance_event_id, event.event_type,
				       event.owner_verified_at, event.valid_until
				FROM provider_commercial_commitment_events event
				WHERE event.provider_offer_id=offer.id
				  AND event.event_type IN ('terms_acceptance','terms_renewal')
				  AND event.offer_version_snapshot=offer.version
				  AND event.exact_terms_sha256=offer.commercial_terms_sha256
				  AND event.valid_until > $1::timestamptz
				ORDER BY event.valid_until DESC, event.id DESC LIMIT 1
			) commitment ON true
			WHERE offer.status='active'
			  AND commitment.valid_until <= $1::timestamptz + INTERVAL '7 days'
			ORDER BY commitment.valid_until ASC, offer.id ASC
			LIMIT $2`, queue.AsOf, limit)
		if err != nil {
			return nil, err
		}
		if err := appendRows(rows, func(row *sql.Rows) (ProviderPilotQueueItem, error) {
			item := ProviderPilotQueueItem{State: "expiring_terms"}
			var version int
			var validUntil sql.NullTime
			err := row.Scan(&item.ProviderClaimID, &item.Domain, &item.OfferID,
				&version, &item.CommercialTermsSHA256, &item.AcceptanceEventID,
				&item.AcceptanceEventType, &item.OccurredAt, &validUntil)
			item.OfferVersion = &version
			if validUntil.Valid {
				value := validUntil.Time
				item.ValidUntil = &value
			}
			return item, err
		}); err != nil {
			return nil, err
		}
	}

	if wants("handoff_awaiting_callback") {
		rows, err := tx.Query(`
			SELECT ticket.provider_claim_id::text, claim.domain_snapshot,
			       ticket.provider_offer_id::text, ticket.offer_version_snapshot,
			       ticket.commercial_terms_sha256_snapshot,
			       ticket.id::text, handoff.id::text, handoff.observed_at
			FROM provider_action_handoff_receipts handoff
			JOIN action_tickets ticket ON ticket.id=handoff.action_ticket_id
			JOIN provider_claims claim ON claim.id=ticket.provider_claim_id
			WHERE ticket.status='redirected'
			  AND ticket.expires_at > $1::timestamptz
			  AND ticket.authorization_revoked_at IS NULL
			  AND claim.status='verified'
			  AND claim.verification_last_succeeded_at >
			      $1::timestamptz - $2::bigint * INTERVAL '1 second'
			  AND NOT EXISTS (
				SELECT 1 FROM outcome_receipts outcome
				WHERE outcome.action_ticket_id=ticket.id
			)
			ORDER BY handoff.observed_at ASC, handoff.id ASC
			LIMIT $3`, queue.AsOf,
			int64(ProviderClaimVerificationFreshness/time.Second), limit)
		if err != nil {
			return nil, err
		}
		if err := appendRows(rows, func(row *sql.Rows) (ProviderPilotQueueItem, error) {
			item := ProviderPilotQueueItem{State: "handoff_awaiting_callback"}
			var version int
			err := row.Scan(&item.ProviderClaimID, &item.Domain, &item.OfferID,
				&version, &item.CommercialTermsSHA256, &item.TicketID,
				&item.HandoffReceiptID, &item.OccurredAt)
			item.OfferVersion = &version
			return item, err
		}); err != nil {
			return nil, err
		}
	}

	if wants("recent_callback") {
		rows, err := tx.Query(`
			SELECT outcome.provider_claim_id::text, claim.domain_snapshot,
			       outcome.provider_offer_id::text, ticket.offer_version_snapshot,
			       ticket.commercial_terms_sha256_snapshot,
			       ticket.id::text, handoff.id::text,
			       outcome.id::text, outcome.outcome, outcome.charge_status,
			       outcome.created_at
			FROM outcome_receipts outcome
			JOIN action_tickets ticket ON ticket.id=outcome.action_ticket_id
			JOIN provider_action_handoff_receipts handoff
			  ON handoff.action_ticket_id=ticket.id
			JOIN provider_claims claim ON claim.id=outcome.provider_claim_id
			ORDER BY outcome.created_at DESC, outcome.id DESC
			LIMIT $1`, limit)
		if err != nil {
			return nil, err
		}
		if err := appendRows(rows, func(row *sql.Rows) (ProviderPilotQueueItem, error) {
			item := ProviderPilotQueueItem{State: "recent_callback"}
			var version int
			err := row.Scan(&item.ProviderClaimID, &item.Domain, &item.OfferID,
				&version, &item.CommercialTermsSHA256, &item.TicketID,
				&item.HandoffReceiptID, &item.OutcomeReceiptID,
				&item.Outcome, &item.ChargeStatus, &item.OccurredAt)
			item.OfferVersion = &version
			return item, err
		}); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return queue, nil
}

func ValidateProviderPilotQueueState(state string) error {
	state = strings.ToLower(strings.TrimSpace(state))
	if state == "" || state == "all" || providerPilotQueueStates[state] {
		return nil
	}
	return errors.New("invalid provider pilot queue state")
}
