package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
)

type providerCommercialAcceptanceRequest struct {
	EventType                   string `json:"event_type"`
	OfferID                     string `json:"offer_id,omitempty"`
	RelatedAcceptanceEventID    string `json:"related_acceptance_event_id,omitempty"`
	OfferVersion                int    `json:"offer_version,omitempty"`
	ExactTermsSHA256            string `json:"exact_terms_sha256,omitempty"`
	ProviderAcceptanceReference string `json:"provider_acceptance_reference"`
}

// ProviderCommercialAcceptances records the provider-authenticated half of a
// commercial commitment. It never treats provider self-attestation as owner-
// verified funding, exact terms, or completed pilot proof.
func (h *ProviderExchangeHandler) ProviderCommercialAcceptances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	key := h.requireProviderKey(w, r)
	if key == nil {
		return
	}
	if !h.allowCommercialAcceptance(w, key.ProviderClaimID) {
		return
	}
	var request providerCommercialAcceptanceRequest
	if err := decodeProviderJSON(w, r, &request); err != nil {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid provider commercial acceptance request"})
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	request.EventType = strings.ToLower(strings.TrimSpace(request.EventType))
	request.OfferID = strings.ToLower(strings.TrimSpace(request.OfferID))
	request.RelatedAcceptanceEventID = strings.ToLower(strings.TrimSpace(request.RelatedAcceptanceEventID))
	request.ExactTermsSHA256 = strings.ToLower(strings.TrimSpace(request.ExactTermsSHA256))
	request.ProviderAcceptanceReference = strings.TrimSpace(request.ProviderAcceptanceReference)
	if len(idempotencyKey) < 8 || len(idempotencyKey) > 128 ||
		!validProviderEvidenceReference(request.ProviderAcceptanceReference) {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "bounded Idempotency-Key and non-secret provider_acceptance_reference required",
		})
		return
	}
	record := h.recordCommercialAcceptance
	if record == nil {
		record = models.RecordProviderCommercialAcceptance
	}
	event, created, err := record(h.DB, key, models.ProviderCommercialAcceptanceInput{
		EventType:                request.EventType,
		ProviderOfferID:          request.OfferID,
		RelatedAcceptanceEventID: request.RelatedAcceptanceEventID,
		ExpectedOfferVersion:     request.OfferVersion,
		ExpectedExactTermsSHA256: request.ExactTermsSHA256,
		IdempotencyKey:           idempotencyKey,
		ProviderReference:        request.ProviderAcceptanceReference,
	})
	if err != nil {
		status, message := providerExchangeStatus(err)
		providerWriteJSON(w, status, map[string]any{
			"error":                       message,
			"commercial_proof_created":    false,
			"owner_verification_required": true,
		})
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	providerWriteJSON(w, status, map[string]any{
		"acceptance":                  event,
		"created":                     created,
		"idempotent_replay":           !created,
		"provider_authenticated":      true,
		"owner_verification_required": true,
		"commercial_proof_created":    false,
		"evidence_scope":              "Provider-authenticated acceptance only; it cannot count as funding, exact CPA terms, renewal, or pilot proof until separately owner-verified.",
	})
}

func (h *ProviderExchangeHandler) allowCommercialAcceptance(w http.ResponseWriter, claimID string) bool {
	if h.commercialLimit == nil {
		h.commercialLimit = newMCPDiscoveryRateLimiter(200, time.Hour)
	}
	_, retry, ok := h.commercialLimit.allow("provider-commercial:"+claimID, time.Now())
	if ok {
		return true
	}
	w.Header().Set("Retry-After", fmt.Sprintf("%d", max(1, int(retry.Seconds()))))
	providerWriteJSON(w, http.StatusTooManyRequests, map[string]string{"error": "provider commercial acceptance safety limit exceeded"})
	return false
}

type providerAdminCommercialActionRequest struct {
	Action                    string     `json:"action"`
	OfferID                   string     `json:"offer_id,omitempty"`
	ProviderAcceptanceEventID string     `json:"provider_acceptance_event_id,omitempty"`
	RelatedCommitmentEventID  string     `json:"related_commitment_event_id,omitempty"`
	CompanyKeyHash            string     `json:"company_key_hash,omitempty"`
	AmountCents               int64      `json:"amount_cents,omitempty"`
	Currency                  string     `json:"currency,omitempty"`
	SourceSystem              string     `json:"source_system,omitempty"`
	SourceEventID             string     `json:"source_event_id,omitempty"`
	SourceEffectiveAt         *time.Time `json:"source_effective_at,omitempty"`
	QualifyingActionTicketID  string     `json:"qualifying_action_ticket_id,omitempty"`
	OperatorReference         string     `json:"operator_reference"`
	IdentityEvidenceReference string     `json:"identity_evidence_reference,omitempty"`
	OwnerEvidenceReference    string     `json:"owner_evidence_reference,omitempty"`
}

func (request *providerAdminCommercialActionRequest) normalize() {
	request.Action = strings.ToLower(strings.TrimSpace(request.Action))
	request.OfferID = strings.ToLower(strings.TrimSpace(request.OfferID))
	request.ProviderAcceptanceEventID = strings.ToLower(strings.TrimSpace(request.ProviderAcceptanceEventID))
	request.RelatedCommitmentEventID = strings.ToLower(strings.TrimSpace(request.RelatedCommitmentEventID))
	request.CompanyKeyHash = strings.ToLower(strings.TrimSpace(request.CompanyKeyHash))
	request.Currency = strings.ToLower(strings.TrimSpace(request.Currency))
	request.SourceSystem = strings.ToLower(strings.TrimSpace(request.SourceSystem))
	request.SourceEventID = strings.TrimSpace(request.SourceEventID)
	request.QualifyingActionTicketID = strings.ToLower(strings.TrimSpace(request.QualifyingActionTicketID))
	request.OperatorReference = strings.TrimSpace(request.OperatorReference)
	request.IdentityEvidenceReference = strings.TrimSpace(request.IdentityEvidenceReference)
	request.OwnerEvidenceReference = strings.TrimSpace(request.OwnerEvidenceReference)
}

func (request providerAdminCommercialActionRequest) sourceEffectiveTime() time.Time {
	if request.SourceEffectiveAt == nil {
		return time.Time{}
	}
	return request.SourceEffectiveAt.UTC()
}

func (request providerAdminCommercialActionRequest) validShape() bool {
	switch request.Action {
	case "verify_company":
		return request.ProviderAcceptanceEventID != "" && request.CompanyKeyHash != "" &&
			request.IdentityEvidenceReference != "" && request.OfferID == "" &&
			request.RelatedCommitmentEventID == "" && request.AmountCents == 0 &&
			request.Currency == "" && request.SourceSystem == "" && request.SourceEventID == "" &&
			request.SourceEffectiveAt == nil && request.QualifyingActionTicketID == "" &&
			request.OwnerEvidenceReference == ""
	case "verify_funding":
		return request.OfferID != "" && request.AmountCents > 0 && request.Currency == "usd" &&
			request.SourceSystem != "" && request.SourceEventID != "" && request.SourceEffectiveAt != nil &&
			request.OwnerEvidenceReference != "" && request.ProviderAcceptanceEventID == "" &&
			request.RelatedCommitmentEventID == "" && request.CompanyKeyHash == "" &&
			request.IdentityEvidenceReference == ""
	case "verify_terms":
		return request.OfferID != "" && request.ProviderAcceptanceEventID != "" &&
			request.SourceSystem != "" && request.SourceEventID != "" && request.SourceEffectiveAt != nil &&
			request.OwnerEvidenceReference != "" && request.CompanyKeyHash == "" &&
			request.AmountCents == 0 && request.Currency == "" &&
			request.QualifyingActionTicketID == "" && request.IdentityEvidenceReference == ""
	case "reverse_funding":
		return request.RelatedCommitmentEventID != "" && request.AmountCents > 0 &&
			request.SourceSystem != "" && request.SourceEventID != "" && request.SourceEffectiveAt != nil &&
			request.OwnerEvidenceReference != "" && request.OfferID == "" &&
			request.ProviderAcceptanceEventID == "" && request.CompanyKeyHash == "" &&
			request.Currency == "" && request.QualifyingActionTicketID == "" &&
			request.IdentityEvidenceReference == ""
	default:
		return false
	}
}

// AdminCommercialAction verifies provider-originated evidence or records a
// funding reversal. The legacy fund/adjust/activate endpoint remains a separate
// operator-progress boundary and cannot manufacture these proof events.
func (h *ProviderExchangeHandler) AdminCommercialAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	if !h.requireAdmin(w, r) {
		return
	}
	var request providerAdminCommercialActionRequest
	if err := decodeProviderJSON(w, r, &request); err != nil {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid provider commercial verification request"})
		return
	}
	request.normalize()
	if !validProviderEvidenceReference(request.OperatorReference) || !request.validShape() {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "action-specific bounded commercial evidence fields are required; unrelated fields are rejected",
		})
		return
	}

	switch request.Action {
	case "verify_company":
		if !validProviderEvidenceReference(request.IdentityEvidenceReference) {
			providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "bounded non-secret identity_evidence_reference required"})
			return
		}
		verify := h.verifyPilotCompany
		if verify == nil {
			verify = models.VerifyProviderPilotCompany
		}
		company, created, err := verify(
			h.DB, request.ProviderAcceptanceEventID, request.CompanyKeyHash,
			request.OperatorReference, request.IdentityEvidenceReference,
		)
		if err != nil {
			h.writeCommercialError(w, err)
			return
		}
		h.writeCommercialSuccess(w, created, "company", company,
			"Provider-authenticated company participation joined to owner-verified deduplication; company verification alone is not funding, CPA acceptance, or pilot proof.")
	case "verify_funding":
		providerWriteJSON(w, http.StatusConflict, map[string]string{
			"error": "prepaid funding verification is disabled for the terms-only pilot",
		})
	case "verify_terms":
		if !validProviderEvidenceReference(request.OwnerEvidenceReference) {
			providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "bounded non-secret owner_evidence_reference required"})
			return
		}
		record := h.recordVerifiedTerms
		if record == nil {
			record = models.RecordVerifiedProviderTerms
		}
		commitment, created, err := record(h.DB, models.VerifiedProviderTermsInput{
			ProviderOfferID:           request.OfferID,
			ProviderAcceptanceEventID: request.ProviderAcceptanceEventID,
			RelatedCommitmentEventID:  request.RelatedCommitmentEventID,
			SourceSystem:              request.SourceSystem, SourceEventID: request.SourceEventID,
			SourceEffectiveAt:      request.sourceEffectiveTime(),
			OperatorReference:      request.OperatorReference,
			OwnerEvidenceReference: request.OwnerEvidenceReference,
		})
		if err != nil {
			h.writeCommercialError(w, err)
			return
		}
		h.writeCommercialSuccess(w, created, "commitment", commitment,
			"Provider-key-authenticated exact terms joined to owner-held evidence; they may qualify but do not alone satisfy the pilot proof threshold.")
	case "reverse_funding":
		providerWriteJSON(w, http.StatusConflict, map[string]string{
			"error": "prepaid funding reversals are disabled for the terms-only pilot",
		})
	}
}

func (h *ProviderExchangeHandler) writeCommercialError(w http.ResponseWriter, err error) {
	status, message := providerExchangeStatus(err)
	providerWriteJSON(w, status, map[string]any{
		"error":                    message,
		"commercial_proof_created": false,
	})
}

func (h *ProviderExchangeHandler) writeCommercialSuccess(w http.ResponseWriter, created bool, key string, value any, scope string) {
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	providerWriteJSON(w, status, map[string]any{
		key:                            value,
		"created":                      created,
		"idempotent_replay":            !created,
		"commercial_evidence_recorded": true,
		"pilot_threshold_evaluated":    false,
		"evidence_scope":               scope,
	})
}
