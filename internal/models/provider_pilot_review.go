package models

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/lib/pq"
)

const ProviderPilotReviewContractV1 = "nhs-provider-pilot-review-v1"

var (
	ErrProviderPilotReviewSnapshotChanged = errors.New("provider pilot review snapshot changed")
	ErrProviderPilotReviewRequired        = errors.New("current provider pilot review required before this transition")
	providerPilotReviewTypes              = map[string]bool{
		"provider": true,
		"offer":    true,
		"ticket":   true,
		"handoff":  true,
		"callback": true,
	}
)

// ProviderPilotReviewCandidate is an owner-only, bounded projection of one
// exact review subject. It deliberately has no search receipt, query, bearer,
// token hash, company-deduplication hash, principal or agent identity/contact/
// network metadata, raw signed-receipt body/signature, or free-form intent.
type ProviderPilotReviewCandidate struct {
	ReviewContractVersion        string `json:"review_contract_version"`
	ProviderPilotEpochID         string `json:"provider_pilot_epoch_id"`
	ProviderPilotContractVersion string `json:"provider_pilot_contract_version"`
	PilotDemandTopic             string `json:"pilot_demand_topic"`
	ReviewType                   string `json:"review_type"`
	SubjectID                    string `json:"subject_id"`
	SubjectSnapshotSHA256        string `json:"subject_snapshot_sha256"`
	ProviderClaimID              string `json:"provider_claim_id"`
	Domain                       string `json:"domain"`

	ProviderPilotCompanyID      string     `json:"provider_pilot_company_id,omitempty"`
	ProviderPilotEnrollmentID   string     `json:"provider_pilot_enrollment_id,omitempty"`
	Stage1EligibilitySHA256     string     `json:"stage1_eligibility_sha256,omitempty"`
	ProviderAcceptanceEventID   string     `json:"provider_acceptance_event_id,omitempty"`
	ProviderAcceptanceReference string     `json:"provider_acceptance_reference,omitempty"`
	ProviderAcceptedAt          *time.Time `json:"provider_accepted_at,omitempty"`
	CompanyOwnerVerifiedAt      *time.Time `json:"company_owner_verified_at,omitempty"`
	EnrolledAt                  *time.Time `json:"enrolled_at,omitempty"`

	ProviderOfferID                string     `json:"provider_offer_id,omitempty"`
	OfferVersion                   *int       `json:"offer_version,omitempty"`
	OfferName                      string     `json:"offer_name,omitempty"`
	OfferSummary                   string     `json:"offer_summary,omitempty"`
	ActionType                     string     `json:"action_type,omitempty"`
	ActionURL                      string     `json:"action_url,omitempty"`
	DisclosureLabel                string     `json:"disclosure_label,omitempty"`
	ChargeEvent                    string     `json:"charge_event,omitempty"`
	BountyCents                    *int64     `json:"bounty_cents,omitempty"`
	Currency                       string     `json:"currency,omitempty"`
	PrincipalPriceMode             string     `json:"principal_price_mode,omitempty"`
	PrincipalPriceCents            *int64     `json:"principal_price_cents,omitempty"`
	PrincipalCurrency              string     `json:"principal_currency,omitempty"`
	BillingMode                    string     `json:"billing_mode,omitempty"`
	TermsCreditLimitCents          *int64     `json:"terms_credit_limit_cents,omitempty"`
	TermsPeriodDays                *int       `json:"terms_period_days,omitempty"`
	CommercialTermsContractVersion string     `json:"commercial_terms_contract_version,omitempty"`
	CommercialTermsSHA256          string     `json:"commercial_terms_sha256,omitempty"`
	CommitmentEventID              string     `json:"commitment_event_id,omitempty"`
	CommitmentEventType            string     `json:"commitment_event_type,omitempty"`
	CommitmentProviderAcceptedAt   *time.Time `json:"commitment_provider_accepted_at,omitempty"`
	CommitmentValidUntil           *time.Time `json:"commitment_valid_until,omitempty"`
	CommitmentOwnerVerifiedAt      *time.Time `json:"commitment_owner_verified_at,omitempty"`

	ActionTicketID   string     `json:"action_ticket_id,omitempty"`
	DemandTopic      string     `json:"demand_topic,omitempty"`
	RegionCode       string     `json:"region_code,omitempty"`
	BudgetBand       string     `json:"budget_band,omitempty"`
	Urgency          string     `json:"urgency,omitempty"`
	RequirementFlags []string   `json:"requirement_flags,omitempty"`
	PrincipalConsent bool       `json:"principal_consent,omitempty"`
	ConsentVersion   string     `json:"consent_version,omitempty"`
	TicketCreatedAt  *time.Time `json:"ticket_created_at,omitempty"`
	TicketExpiresAt  *time.Time `json:"ticket_expires_at,omitempty"`

	HandoffReceiptID                         string     `json:"handoff_receipt_id,omitempty"`
	PrincipalHandoffConsent                  bool       `json:"principal_handoff_consent,omitempty"`
	HandoffConsentVersion                    string     `json:"handoff_consent_version,omitempty"`
	ControlledIntentDisclosureConsent        bool       `json:"controlled_intent_disclosure_consent,omitempty"`
	ControlledIntentDisclosureConsentVersion string     `json:"controlled_intent_disclosure_consent_version,omitempty"`
	HandoffEventContractVersion              string     `json:"handoff_event_contract_version,omitempty"`
	HandoffObservedAt                        *time.Time `json:"handoff_observed_at,omitempty"`

	OutcomeReceiptID           string     `json:"outcome_receipt_id,omitempty"`
	OutcomeNHSEventID          string     `json:"outcome_nhs_event_id,omitempty"`
	ProviderAPIKeyID           *int64     `json:"provider_api_key_id,omitempty"`
	Outcome                    string     `json:"outcome,omitempty"`
	ChargeStatus               string     `json:"charge_status,omitempty"`
	BilledCents                *int64     `json:"billed_cents,omitempty"`
	ProviderReportedAt         *time.Time `json:"provider_reported_at,omitempty"`
	OutcomeRecordedAt          *time.Time `json:"outcome_recorded_at,omitempty"`
	OutcomeSignedReceiptSHA256 string     `json:"outcome_signed_receipt_sha256,omitempty"`
	OutcomeSignatureSHA256     string     `json:"outcome_signature_sha256,omitempty"`
	ExistingReviewID           string     `json:"existing_review_id,omitempty"`
	ExistingReviewedAt         *time.Time `json:"existing_reviewed_at,omitempty"`
}

type ProviderPilotReviewInput struct {
	ProviderPilotEpochID   string
	ReviewType             string
	SubjectID              string
	ExpectedSnapshotSHA256 string
	OwnerReference         string
	EvidenceReference      string
}

type ProviderPilotReviewEvent struct {
	ID                    string    `json:"id"`
	ProviderPilotEpochID  string    `json:"provider_pilot_epoch_id"`
	ReviewContractVersion string    `json:"review_contract_version"`
	ReviewType            string    `json:"review_type"`
	SubjectID             string    `json:"subject_id"`
	ProviderClaimID       string    `json:"provider_claim_id"`
	ProviderOfferID       string    `json:"provider_offer_id,omitempty"`
	ActionTicketID        string    `json:"action_ticket_id,omitempty"`
	HandoffReceiptID      string    `json:"handoff_receipt_id,omitempty"`
	OutcomeReceiptID      string    `json:"outcome_receipt_id,omitempty"`
	SubjectSnapshotSHA256 string    `json:"subject_snapshot_sha256"`
	OwnerReference        string    `json:"owner_reference"`
	EvidenceReference     string    `json:"evidence_reference"`
	ReviewedAt            time.Time `json:"reviewed_at"`
	Replayed              bool      `json:"-"`
}

type providerPilotReviewQueryer interface {
	QueryRow(query string, args ...any) *sql.Row
}

// requireCurrentProviderPilotReview is the pre-event chronology gate used by
// irreversible pilot transitions. It accepts only the exact immutable review
// contract and the digest recomputed from current database facts. A missing or
// stale receipt is deliberately one state conflict, never an implicit review.
func requireCurrentProviderPilotReview(
	queryer providerPilotReviewQueryer,
	pilotID, reviewType, subjectID string,
) error {
	pilotID, reviewType, subjectID, err := normalizeProviderPilotReviewIdentity(
		pilotID, reviewType, subjectID,
	)
	if err != nil || queryer == nil {
		return ErrInvalidProviderExchange
	}
	var current bool
	err = queryer.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM provider_pilot_review_events review
			WHERE review.provider_pilot_epoch_id=$1::uuid
			  AND review.review_type=$2
			  AND review.subject_id=$3::uuid
			  AND review.review_contract_version=$4
			  AND review.subject_snapshot_sha256=
			      provider_pilot_review_snapshot_sha256($1::uuid,$2,$3::uuid)
		)`, pilotID, reviewType, subjectID, ProviderPilotReviewContractV1).Scan(&current)
	if err != nil {
		return err
	}
	if !current {
		return ErrProviderPilotReviewRequired
	}
	return nil
}

const providerPilotReviewEventColumns = `
	id::text, provider_pilot_epoch_id::text, review_contract_version,
	review_type, subject_id::text, provider_claim_id::text,
	COALESCE(provider_offer_id::text,''),
	COALESCE(action_ticket_id::text,''),
	COALESCE(handoff_receipt_id::text,''),
	COALESCE(outcome_receipt_id::text,''),
	subject_snapshot_sha256, owner_reference, evidence_reference, reviewed_at`

func normalizeProviderPilotReviewIdentity(pilotID, reviewType, subjectID string) (string, string, string, error) {
	pilotID = strings.ToLower(strings.TrimSpace(pilotID))
	reviewType = strings.ToLower(strings.TrimSpace(reviewType))
	subjectID = strings.ToLower(strings.TrimSpace(subjectID))
	if !validProviderUUID(pilotID) || !providerPilotReviewTypes[reviewType] || !validProviderUUID(subjectID) {
		return "", "", "", ErrInvalidProviderExchange
	}
	return pilotID, reviewType, subjectID, nil
}

func scanProviderPilotReviewEvent(row rowScanner) (*ProviderPilotReviewEvent, error) {
	var event ProviderPilotReviewEvent
	if err := row.Scan(
		&event.ID, &event.ProviderPilotEpochID, &event.ReviewContractVersion,
		&event.ReviewType, &event.SubjectID, &event.ProviderClaimID,
		&event.ProviderOfferID, &event.ActionTicketID, &event.HandoffReceiptID,
		&event.OutcomeReceiptID, &event.SubjectSnapshotSHA256,
		&event.OwnerReference, &event.EvidenceReference, &event.ReviewedAt,
	); err != nil {
		return nil, err
	}
	event.ID = strings.ToLower(event.ID)
	event.ProviderPilotEpochID = strings.ToLower(event.ProviderPilotEpochID)
	event.SubjectID = strings.ToLower(event.SubjectID)
	event.ProviderClaimID = strings.ToLower(event.ProviderClaimID)
	event.ProviderOfferID = strings.ToLower(event.ProviderOfferID)
	event.ActionTicketID = strings.ToLower(event.ActionTicketID)
	event.HandoffReceiptID = strings.ToLower(event.HandoffReceiptID)
	event.OutcomeReceiptID = strings.ToLower(event.OutcomeReceiptID)
	return &event, nil
}

func providerPilotReviewBase(pilotID, reviewType, subjectID string) *ProviderPilotReviewCandidate {
	return &ProviderPilotReviewCandidate{
		ReviewContractVersion: ProviderPilotReviewContractV1,
		ProviderPilotEpochID:  pilotID,
		ReviewType:            reviewType,
		SubjectID:             subjectID,
		RequirementFlags:      []string{},
	}
}

func getProviderPilotReviewCandidate(
	queryer providerPilotReviewQueryer,
	pilotID, reviewType, subjectID string,
) (*ProviderPilotReviewCandidate, error) {
	if queryer == nil {
		return nil, ErrInvalidProviderExchange
	}
	candidate := providerPilotReviewBase(pilotID, reviewType, subjectID)
	switch reviewType {
	case "provider":
		var acceptedAt, verifiedAt, enrolledAt time.Time
		err := queryer.QueryRow(`
			SELECT provider_pilot_review_snapshot_sha256($1::uuid,$2,$3::uuid),
			       epoch.contract_version, epoch.demand_topic,
			       claim.id::text, claim.domain_snapshot, company.id::text,
			       enrollment.id::text,
			       enrollment.stage1_eligibility_snapshot_sha256,
			       accepted.id::text,
			       accepted.provider_acceptance_reference,
			       accepted.provider_accepted_at, company.owner_verified_at,
			       enrollment.enrolled_at
			FROM provider_pilot_epochs epoch
			JOIN provider_pilot_enrollments enrollment
			  ON enrollment.provider_pilot_epoch_id=epoch.id
			 AND enrollment.provider_claim_id=$3::uuid
			JOIN provider_pilot_companies company
			  ON company.id=enrollment.provider_pilot_company_id
			 AND company.provider_claim_id=enrollment.provider_claim_id
			JOIN provider_claims claim ON claim.id=enrollment.provider_claim_id
			JOIN provider_commercial_acceptance_events accepted
			  ON accepted.id=company.provider_acceptance_event_id
			WHERE epoch.id=$1::uuid
			  AND claim.status='verified'
			  AND claim.verification_last_succeeded_at >
			      statement_timestamp()-($4 * INTERVAL '1 second')
			  AND provider_pilot_enrollment_eligibility_is_current(
			      epoch.id, claim.id
			  )`,
			pilotID, reviewType, subjectID,
			int64(ProviderClaimVerificationFreshness/time.Second)).Scan(
			&candidate.SubjectSnapshotSHA256,
			&candidate.ProviderPilotContractVersion, &candidate.PilotDemandTopic,
			&candidate.ProviderClaimID,
			&candidate.Domain, &candidate.ProviderPilotCompanyID,
			&candidate.ProviderPilotEnrollmentID,
			&candidate.Stage1EligibilitySHA256,
			&candidate.ProviderAcceptanceEventID,
			&candidate.ProviderAcceptanceReference, &acceptedAt, &verifiedAt,
			&enrolledAt,
		)
		if err != nil {
			return nil, err
		}
		candidate.ProviderAcceptedAt = &acceptedAt
		candidate.CompanyOwnerVerifiedAt = &verifiedAt
		candidate.EnrolledAt = &enrolledAt

	case "offer":
		var version int
		var bounty int64
		var principalPrice, termsLimit sql.NullInt64
		var termsDays sql.NullInt64
		var providerAcceptedAt, ownerVerifiedAt time.Time
		var validUntil sql.NullTime
		var providerAcceptanceEventID sql.NullString
		err := queryer.QueryRow(`
			SELECT provider_pilot_review_snapshot_sha256($1::uuid,$2,$3::uuid),
			       epoch.contract_version, epoch.demand_topic,
			       claim.id::text, claim.domain_snapshot, company.id::text,
			       offer.id::text, offer.version, offer.offer_name,
			       offer.offer_summary, offer.action_type, offer.action_url,
			       offer.disclosure_label,
			       offer.charge_event, offer.bounty_cents, offer.currency,
			       offer.principal_price_mode, offer.principal_price_cents,
			       offer.principal_currency,
			       offer.billing_mode, offer.terms_credit_limit_cents,
			       offer.terms_period_days,
			       offer.commercial_terms_contract_version,
			       offer.commercial_terms_sha256, commitment.id::text,
			       commitment.event_type,
			       commitment.provider_acceptance_event_id::text,
			       commitment.provider_accepted_at, commitment.valid_until,
			       commitment.owner_verified_at
			FROM provider_pilot_epochs epoch
			JOIN provider_pilot_enrollments enrollment
			  ON enrollment.provider_pilot_epoch_id=epoch.id
			JOIN provider_pilot_companies company
			  ON company.id=enrollment.provider_pilot_company_id
			 AND company.provider_claim_id=enrollment.provider_claim_id
			JOIN provider_claims claim
			  ON claim.id=enrollment.provider_claim_id
			JOIN provider_offers offer
			  ON offer.id=$3::uuid
			 AND offer.provider_claim_id=enrollment.provider_claim_id
			 AND (offer.provider_pilot_epoch_id IS NULL OR
			      offer.provider_pilot_epoch_id=epoch.id)
			JOIN LATERAL (
				SELECT commitment.*
				FROM provider_commercial_commitment_events commitment
				WHERE commitment.provider_pilot_company_id=company.id
				  AND commitment.provider_claim_id=offer.provider_claim_id
				  AND commitment.provider_offer_id=offer.id
				  AND commitment.offer_version_snapshot=offer.version
				  AND commitment.terms_contract_version=offer.commercial_terms_contract_version
				  AND commitment.exact_terms_sha256=offer.commercial_terms_sha256
				  AND (
					(offer.billing_mode='prepaid'
					 AND commitment.event_type='prepaid_fund'
					 AND commitment.amount_cents + COALESCE((
						SELECT SUM(reversal.amount_cents)
						FROM provider_commercial_commitment_events reversal
						WHERE reversal.related_event_id=commitment.id
						  AND reversal.event_type='fund_reversal'
					 ),0) > 0) OR
					(offer.billing_mode='terms'
					 AND commitment.event_type IN ('terms_acceptance','terms_renewal')
					 AND commitment.valid_until > statement_timestamp())
				  )
				ORDER BY commitment.owner_verified_at ASC,
				         commitment.provider_accepted_at ASC, commitment.id ASC
				LIMIT 1
			) commitment ON true
			WHERE epoch.id=$1::uuid
			  AND claim.status='verified'
			  AND claim.verification_last_succeeded_at >
			      statement_timestamp()-($4 * INTERVAL '1 second')
			  AND offer.commercial_terms_contract_version=$5
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
			  )`,
			pilotID, reviewType, subjectID,
			int64(ProviderClaimVerificationFreshness/time.Second),
			ProviderCommercialTermsContractV1).Scan(
			&candidate.SubjectSnapshotSHA256,
			&candidate.ProviderPilotContractVersion, &candidate.PilotDemandTopic,
			&candidate.ProviderClaimID,
			&candidate.Domain, &candidate.ProviderPilotCompanyID,
			&candidate.ProviderOfferID, &version, &candidate.OfferName,
			&candidate.OfferSummary, &candidate.ActionType, &candidate.ActionURL,
			&candidate.DisclosureLabel,
			&candidate.ChargeEvent, &bounty, &candidate.Currency,
			&candidate.PrincipalPriceMode, &principalPrice,
			&candidate.PrincipalCurrency,
			&candidate.BillingMode, &termsLimit, &termsDays,
			&candidate.CommercialTermsContractVersion,
			&candidate.CommercialTermsSHA256, &candidate.CommitmentEventID,
			&candidate.CommitmentEventType,
			&providerAcceptanceEventID, &providerAcceptedAt,
			&validUntil, &ownerVerifiedAt,
		)
		if err != nil {
			return nil, err
		}
		candidate.OfferVersion = &version
		candidate.BountyCents = &bounty
		if principalPrice.Valid {
			value := principalPrice.Int64
			candidate.PrincipalPriceCents = &value
		}
		if termsLimit.Valid {
			value := termsLimit.Int64
			candidate.TermsCreditLimitCents = &value
		}
		if termsDays.Valid {
			value := int(termsDays.Int64)
			candidate.TermsPeriodDays = &value
		}
		if providerAcceptanceEventID.Valid {
			candidate.ProviderAcceptanceEventID = providerAcceptanceEventID.String
		}
		if validUntil.Valid {
			value := validUntil.Time
			candidate.CommitmentValidUntil = &value
		}
		candidate.CommitmentOwnerVerifiedAt = &ownerVerifiedAt
		candidate.CommitmentProviderAcceptedAt = &providerAcceptedAt

	case "ticket":
		var version int
		var bounty int64
		var principalPrice, termsLimit sql.NullInt64
		var termsDays sql.NullInt64
		var createdAt, expiresAt time.Time
		var requirementFlags pq.StringArray
		err := queryer.QueryRow(`
			SELECT provider_pilot_review_snapshot_sha256($1::uuid,$2,$3::uuid),
			       epoch.contract_version, epoch.demand_topic,
			       ticket.provider_claim_id::text, claim.domain_snapshot,
			       ticket.provider_offer_id::text, ticket.id::text,
			       ticket.offer_version_snapshot, ticket.offer_name_snapshot,
			       ticket.offer_summary_snapshot, ticket.action_type_snapshot,
			       ticket.action_url_snapshot, ticket.disclosure_snapshot,
			       ticket.charge_event_snapshot,
			       ticket.bounty_cents_snapshot, ticket.currency_snapshot,
			       ticket.principal_price_mode_snapshot,
			       ticket.principal_price_cents_snapshot,
			       ticket.principal_currency_snapshot,
			       ticket.billing_mode_snapshot,
			       ticket.terms_credit_limit_cents_snapshot,
			       ticket.terms_period_days_snapshot,
			       ticket.commercial_terms_contract_version_snapshot,
			       ticket.commercial_terms_sha256_snapshot,
			       ticket.demand_topic, ticket.region_code, ticket.budget_band,
			       ticket.urgency, ticket.requirement_flags,
			       ticket.principal_consent, ticket.consent_version,
			       ticket.created_at, ticket.expires_at
			FROM provider_pilot_epochs epoch
			JOIN action_tickets ticket
			  ON ticket.provider_pilot_epoch_id=epoch.id
			JOIN provider_claims claim ON claim.id=ticket.provider_claim_id
			WHERE ticket.id=$3::uuid
			  AND epoch.id=$1::uuid
			  AND NOT ticket.source_is_synthetic
			  AND ticket.intent_redacted_at IS NULL`, pilotID, reviewType, subjectID).Scan(
			&candidate.SubjectSnapshotSHA256,
			&candidate.ProviderPilotContractVersion, &candidate.PilotDemandTopic,
			&candidate.ProviderClaimID,
			&candidate.Domain, &candidate.ProviderOfferID,
			&candidate.ActionTicketID, &version, &candidate.OfferName,
			&candidate.OfferSummary, &candidate.ActionType, &candidate.ActionURL,
			&candidate.DisclosureLabel,
			&candidate.ChargeEvent, &bounty, &candidate.Currency,
			&candidate.PrincipalPriceMode, &principalPrice,
			&candidate.PrincipalCurrency,
			&candidate.BillingMode, &termsLimit, &termsDays,
			&candidate.CommercialTermsContractVersion,
			&candidate.CommercialTermsSHA256, &candidate.DemandTopic,
			&candidate.RegionCode, &candidate.BudgetBand, &candidate.Urgency,
			&requirementFlags, &candidate.PrincipalConsent,
			&candidate.ConsentVersion, &createdAt, &expiresAt,
		)
		if err != nil {
			return nil, err
		}
		candidate.OfferVersion = &version
		candidate.BountyCents = &bounty
		if principalPrice.Valid {
			value := principalPrice.Int64
			candidate.PrincipalPriceCents = &value
		}
		if termsLimit.Valid {
			value := termsLimit.Int64
			candidate.TermsCreditLimitCents = &value
		}
		if termsDays.Valid {
			value := int(termsDays.Int64)
			candidate.TermsPeriodDays = &value
		}
		candidate.TicketCreatedAt = &createdAt
		candidate.TicketExpiresAt = &expiresAt
		candidate.RequirementFlags = []string(requirementFlags)

	case "handoff":
		var version int
		var observedAt time.Time
		err := queryer.QueryRow(`
			SELECT provider_pilot_review_snapshot_sha256($1::uuid,$2,$3::uuid),
			       epoch.contract_version, epoch.demand_topic,
			       handoff.provider_claim_id::text, claim.domain_snapshot,
			       handoff.provider_offer_id::text,
			       handoff.action_ticket_id::text, handoff.id::text,
			       handoff.offer_version_snapshot,
			       handoff.commercial_terms_contract_version_snapshot,
			       handoff.commercial_terms_sha256_snapshot,
			       ticket.action_type_snapshot,
			       handoff.principal_handoff_consent,
			       handoff.handoff_consent_version,
			       handoff.principal_controlled_intent_disclosure_consent,
			       handoff.controlled_intent_disclosure_consent_version,
			       handoff.event_contract_version, handoff.observed_at
			FROM provider_pilot_epochs epoch
			JOIN action_tickets ticket
			  ON ticket.provider_pilot_epoch_id=epoch.id
			JOIN provider_action_handoff_receipts handoff
			  ON handoff.action_ticket_id=ticket.id
			JOIN provider_claims claim ON claim.id=handoff.provider_claim_id
			WHERE handoff.id=$3::uuid
			  AND epoch.id=$1::uuid`, pilotID, reviewType, subjectID).Scan(
			&candidate.SubjectSnapshotSHA256,
			&candidate.ProviderPilotContractVersion, &candidate.PilotDemandTopic,
			&candidate.ProviderClaimID,
			&candidate.Domain, &candidate.ProviderOfferID,
			&candidate.ActionTicketID, &candidate.HandoffReceiptID, &version,
			&candidate.CommercialTermsContractVersion,
			&candidate.CommercialTermsSHA256, &candidate.ActionType,
			&candidate.PrincipalHandoffConsent,
			&candidate.HandoffConsentVersion,
			&candidate.ControlledIntentDisclosureConsent,
			&candidate.ControlledIntentDisclosureConsentVersion,
			&candidate.HandoffEventContractVersion, &observedAt,
		)
		if err != nil {
			return nil, err
		}
		candidate.OfferVersion = &version
		candidate.HandoffObservedAt = &observedAt

	case "callback":
		var billed, providerAPIKeyID int64
		var offerVersion int
		var providerReportedAt, recordedAt time.Time
		err := queryer.QueryRow(`
			SELECT provider_pilot_review_snapshot_sha256($1::uuid,$2,$3::uuid),
			       epoch.contract_version, epoch.demand_topic,
			       outcome.provider_claim_id::text, claim.domain_snapshot,
			       outcome.provider_offer_id::text,
			       outcome.action_ticket_id::text, handoff.id::text,
			       outcome.id::text, outcome.nhs_event_id::text,
			       outcome.provider_api_key_id, ticket.offer_version_snapshot,
			       ticket.action_type_snapshot, ticket.charge_event_snapshot,
			       ticket.commercial_terms_contract_version_snapshot,
			       ticket.commercial_terms_sha256_snapshot,
			       outcome.outcome, outcome.charge_status, outcome.billed_cents,
			       outcome.currency,
			       encode(sha256(convert_to(outcome.signed_receipt, 'UTF8')), 'hex'),
			       encode(sha256(convert_to(outcome.signature, 'UTF8')), 'hex'),
			       outcome.provider_reported_at,
			       outcome.created_at
			FROM provider_pilot_epochs epoch
			JOIN action_tickets ticket
			  ON ticket.provider_pilot_epoch_id=epoch.id
			JOIN provider_action_handoff_receipts handoff
			  ON handoff.action_ticket_id=ticket.id
			JOIN outcome_receipts outcome
			  ON outcome.action_ticket_id=ticket.id
			JOIN provider_claims claim ON claim.id=outcome.provider_claim_id
			WHERE outcome.id=$3::uuid
			  AND epoch.id=$1::uuid`, pilotID, reviewType, subjectID).Scan(
			&candidate.SubjectSnapshotSHA256,
			&candidate.ProviderPilotContractVersion, &candidate.PilotDemandTopic,
			&candidate.ProviderClaimID,
			&candidate.Domain, &candidate.ProviderOfferID,
			&candidate.ActionTicketID, &candidate.HandoffReceiptID,
			&candidate.OutcomeReceiptID, &candidate.OutcomeNHSEventID,
			&providerAPIKeyID, &offerVersion, &candidate.ActionType,
			&candidate.ChargeEvent, &candidate.CommercialTermsContractVersion,
			&candidate.CommercialTermsSHA256,
			&candidate.Outcome, &candidate.ChargeStatus, &billed,
			&candidate.Currency, &candidate.OutcomeSignedReceiptSHA256,
			&candidate.OutcomeSignatureSHA256, &providerReportedAt, &recordedAt,
		)
		if err != nil {
			return nil, err
		}
		candidate.BilledCents = &billed
		candidate.ProviderAPIKeyID = &providerAPIKeyID
		candidate.OfferVersion = &offerVersion
		candidate.ProviderReportedAt = &providerReportedAt
		candidate.OutcomeRecordedAt = &recordedAt
	}

	var existingID sql.NullString
	var existingAt sql.NullTime
	err := queryer.QueryRow(`
		SELECT id::text, reviewed_at
		FROM provider_pilot_review_events
		WHERE provider_pilot_epoch_id=$1::uuid
		  AND review_type=$2 AND subject_id=$3::uuid`, pilotID, reviewType, subjectID).
		Scan(&existingID, &existingAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if existingID.Valid {
		candidate.ExistingReviewID = existingID.String
	}
	if existingAt.Valid {
		value := existingAt.Time
		candidate.ExistingReviewedAt = &value
	}
	return candidate, nil
}

func GetProviderPilotReviewCandidate(
	db *sql.DB,
	pilotID, reviewType, subjectID string,
) (*ProviderPilotReviewCandidate, error) {
	pilotID, reviewType, subjectID, err := normalizeProviderPilotReviewIdentity(
		pilotID, reviewType, subjectID,
	)
	if err != nil || db == nil {
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
	candidate, err := getProviderPilotReviewCandidate(tx, pilotID, reviewType, subjectID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return candidate, nil
}

func RecordProviderPilotReview(
	db *sql.DB,
	input ProviderPilotReviewInput,
) (*ProviderPilotReviewEvent, bool, error) {
	pilotID, reviewType, subjectID, err := normalizeProviderPilotReviewIdentity(
		input.ProviderPilotEpochID, input.ReviewType, input.SubjectID,
	)
	input.ExpectedSnapshotSHA256 = strings.ToLower(strings.TrimSpace(input.ExpectedSnapshotSHA256))
	input.OwnerReference = strings.TrimSpace(input.OwnerReference)
	input.EvidenceReference = strings.TrimSpace(input.EvidenceReference)
	if err != nil || db == nil ||
		!providerHashPattern.MatchString(input.ExpectedSnapshotSHA256) ||
		!validProviderReference(input.OwnerReference) ||
		!validProviderReference(input.EvidenceReference) {
		return nil, false, ErrInvalidProviderExchange
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	var epochID string
	if err := tx.QueryRow(`
		SELECT id::text FROM provider_pilot_epochs
		WHERE id=$1::uuid FOR UPDATE`, pilotID).Scan(&epochID); err != nil {
		return nil, false, err
	}
	candidate, err := getProviderPilotReviewCandidate(tx, pilotID, reviewType, subjectID)
	if err != nil {
		return nil, false, err
	}
	if candidate.SubjectSnapshotSHA256 != input.ExpectedSnapshotSHA256 {
		return nil, false, ErrProviderPilotReviewSnapshotChanged
	}
	existing, err := scanProviderPilotReviewEvent(tx.QueryRow(`
		SELECT `+providerPilotReviewEventColumns+`
		FROM provider_pilot_review_events
		WHERE provider_pilot_epoch_id=$1::uuid
		  AND review_type=$2 AND subject_id=$3::uuid`,
		pilotID, reviewType, subjectID))
	if err == nil {
		if existing.SubjectSnapshotSHA256 != input.ExpectedSnapshotSHA256 ||
			existing.OwnerReference != input.OwnerReference ||
			existing.EvidenceReference != input.EvidenceReference {
			return nil, false, ErrProviderIdempotency
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
	event, err := scanProviderPilotReviewEvent(tx.QueryRow(`
		INSERT INTO provider_pilot_review_events (
			provider_pilot_epoch_id, review_contract_version,
			review_type, subject_id, provider_claim_id,
			provider_offer_id, action_ticket_id, handoff_receipt_id,
			outcome_receipt_id, subject_snapshot_sha256,
			owner_reference, evidence_reference
		) VALUES (
			$1::uuid,$2,$3,$4::uuid,$5::uuid,
			NULLIF($6,'')::uuid,NULLIF($7,'')::uuid,
			NULLIF($8,'')::uuid,NULLIF($9,'')::uuid,$10,$11,$12
		)
		RETURNING `+providerPilotReviewEventColumns,
		pilotID, ProviderPilotReviewContractV1, reviewType, subjectID,
		candidate.ProviderClaimID, candidate.ProviderOfferID,
		candidate.ActionTicketID, candidate.HandoffReceiptID,
		candidate.OutcomeReceiptID, candidate.SubjectSnapshotSHA256,
		input.OwnerReference, input.EvidenceReference))
	if err != nil {
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	return event, true, nil
}
