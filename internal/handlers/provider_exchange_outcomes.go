package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
	"github.com/unitedideas/nothumansearch/internal/providerexchange"
)

type providerOutcomeRequest struct {
	// TicketID is accepted only for pre-pilot client compatibility. The signed
	// attribution token is authoritative, so normal providers never need to
	// decode the bearer or obtain optional controlled-intent disclosure just to
	// report an outcome.
	TicketID         string `json:"ticket_id,omitempty"`
	AttributionToken string `json:"attribution_token"`
	Outcome          string `json:"outcome"`
}

func (h *ProviderExchangeHandler) ProviderOutcomes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	if !h.allowOutcomeRequest(w, r) {
		return
	}
	if h.Signer == nil {
		providerWriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "signed provider outcomes are not configured"})
		return
	}
	var request providerOutcomeRequest
	if err := decodeProviderJSON(w, r, &request); err != nil {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid provider outcome request"})
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	request.AttributionToken = strings.TrimSpace(request.AttributionToken)
	request.Outcome = strings.ToLower(strings.TrimSpace(request.Outcome))
	claims, expiredChargeResolution, err := verifyProviderOutcomeAttribution(
		h.Signer,
		request.AttributionToken,
		request.Outcome,
		time.Now().UTC(),
	)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, providerexchange.ErrExpired) {
			status = http.StatusGone
		}
		providerWriteJSON(w, status, map[string]string{"error": "invalid or expired NHS attribution token"})
		return
	}
	request.TicketID, err = providerOutcomeTicketID(request.TicketID, claims.TicketID)
	if err != nil {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "attribution token does not match ticket"})
		return
	}
	var ticket *models.ActionTicket
	if expiredChargeResolution {
		ticket, err = models.ResolveActionTicketForChargeResolution(h.DB, claims.TicketID, request.AttributionToken)
	} else {
		ticket, err = models.ResolveActionTicket(h.DB, claims.TicketID, request.AttributionToken)
		// Emergency revocation intentionally removes live authorization. A
		// provider-authenticated invalid/duplicate callback may still resolve an
		// existing charge, and only that narrow model query can cross revocation.
		if err != nil && providerChargeResolutionOutcome(request.Outcome) {
			ticket, err = models.ResolveActionTicketForChargeResolution(h.DB, claims.TicketID, request.AttributionToken)
		}
	}
	if err != nil || ticket == nil || ticket.ProviderOfferID != claims.OfferID {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "attribution token does not match an eligible action ticket"})
		return
	}
	key := h.resolveProviderOutcomeKey(w, r, request.Outcome, claims.TicketID)
	if key == nil {
		return
	}
	receipt, created, err := models.RecordProviderOutcome(h.DB, key, models.ProviderOutcomeInput{
		ActionTicketID: ticket.ID,
		IdempotencyKey: idempotencyKey,
		PayloadHash:    providerOutcomePayloadHash(ticket.ID, request.Outcome, request.AttributionToken),
		Outcome:        request.Outcome,
	}, h.Signer)
	if err != nil {
		status, message := providerExchangeStatus(err)
		providerWriteJSON(w, status, map[string]any{
			"error":             message,
			"principal_charged": false,
		})
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	providerWriteJSON(w, status, map[string]any{
		"receipt":                        receipt,
		"created":                        created,
		"idempotent_replay":              !created,
		"principal_charged":              false,
		"provider_mor_contract_required": true,
		"principal_charged_by_nhs":       false,
	})
}

func providerOutcomeTicketID(asserted, signed string) (string, error) {
	asserted = strings.ToLower(strings.TrimSpace(asserted))
	signed = strings.ToLower(strings.TrimSpace(signed))
	if signed == "" || (asserted != "" && !constantTimeStringEqual(asserted, signed)) {
		return "", errors.New("signed provider ticket mismatch")
	}
	return signed, nil
}

func (h *ProviderExchangeHandler) resolveProviderOutcomeKey(w http.ResponseWriter, r *http.Request, outcome, ticketID string) *models.ProviderAPIKey {
	raw := extractProviderKey(r)
	if raw == "" || h.DB == nil {
		providerWriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "valid provider callback key required"})
		return nil
	}
	key, err := models.ResolveProviderAPIKey(h.DB, raw)
	if err != nil && providerChargeResolutionOutcome(outcome) {
		// Claim revocation invalidates ordinary key use, but the exact key revoked
		// with that claim may still resolve an existing charged ticket. The model
		// binds it to this ticket and later permits only invalid/duplicate credit.
		key, err = models.ResolveProviderAPIKeyForChargeResolution(h.DB, raw, ticketID)
	}
	if err != nil || key == nil {
		providerWriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "valid provider callback key required"})
		return nil
	}
	return key
}

func providerChargeResolutionOutcome(outcome string) bool {
	outcome = strings.ToLower(strings.TrimSpace(outcome))
	return outcome == string(providerexchange.OutcomeInvalid) || outcome == string(providerexchange.OutcomeDuplicate)
}

func verifyProviderOutcomeAttribution(signer *providerexchange.Signer, token, outcome string, now time.Time) (providerexchange.AttributionClaims, bool, error) {
	if signer == nil {
		return providerexchange.AttributionClaims{}, false, providerexchange.ErrSecretRequired
	}
	claims, err := signer.VerifyAttribution(token, now)
	if err == nil {
		return claims, false, nil
	}
	if !errors.Is(err, providerexchange.ErrExpired) || !providerChargeResolutionOutcome(outcome) {
		return providerexchange.AttributionClaims{}, false, err
	}
	claims, err = signer.VerifyAttributionSignature(token)
	if err != nil {
		return providerexchange.AttributionClaims{}, false, err
	}
	return claims, true, nil
}

func (h *ProviderExchangeHandler) allowOutcomeRequest(w http.ResponseWriter, r *http.Request) bool {
	if h.outcomeLimit == nil {
		h.outcomeLimit = newMCPDiscoveryRateLimiter(1000, time.Hour)
	}
	_, retry, ok := h.outcomeLimit.allow("provider-outcome:"+submitHashIP(r), time.Now())
	if ok {
		return true
	}
	w.Header().Set("Retry-After", fmt.Sprintf("%d", max(1, int(retry.Seconds()))))
	providerWriteJSON(w, http.StatusTooManyRequests, map[string]string{"error": "provider outcome safety limit exceeded"})
	return false
}

func (h *ProviderExchangeHandler) ProviderReceipt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}
	key := h.requireProviderKey(w, r)
	if key == nil {
		return
	}
	receiptID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/provider/receipts/"), "/")
	receipt, err := models.GetOutcomeReceipt(h.DB, key, receiptID)
	if err != nil {
		status, message := providerExchangeStatus(err)
		providerWriteJSON(w, status, map[string]string{"error": message})
		return
	}
	providerWriteJSON(w, http.StatusOK, map[string]any{"receipt": receipt})
}

func (h *ProviderExchangeHandler) VerifyOutcomeReceipt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	if h.Signer == nil {
		providerWriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "receipt verification is not configured"})
		return
	}
	var request struct {
		SignedReceipt string `json:"signed_receipt"`
		Signature     string `json:"signature"`
	}
	if err := decodeProviderJSON(w, r, &request); err != nil {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid receipt verification request"})
		return
	}
	receipt, err := h.Signer.VerifyOutcomeReceiptSignature(request.SignedReceipt, request.Signature)
	if err != nil {
		providerWriteJSON(w, http.StatusOK, map[string]any{
			"signature_valid":         false,
			"within_validity_window":  false,
			"current_state_available": false,
		})
		return
	}

	withinValidityWindow := true
	timeStatus := "current"
	if _, freshnessErr := h.Signer.VerifyOutcomeReceipt(
		request.SignedReceipt,
		request.Signature,
		time.Now().UTC(),
	); freshnessErr != nil {
		withinValidityWindow = false
		switch {
		case errors.Is(freshnessErr, providerexchange.ErrExpired):
			timeStatus = "expired"
		case errors.Is(freshnessErr, providerexchange.ErrNotYetValid):
			timeStatus = "not_yet_valid"
		default:
			timeStatus = "invalid_time"
		}
	}

	lookup := h.receiptState
	if lookup == nil {
		lookup = models.GetPublicOutcomeReceiptState
	}
	currentState, stateErr := lookup(h.DB, receipt.ReceiptID, receipt.TicketID)
	stateStatus := "current"
	if stateErr != nil || currentState == nil {
		currentState = nil
		stateStatus = "unavailable"
		if errors.Is(stateErr, sql.ErrNoRows) {
			stateStatus = "not_found"
		}
	}
	providerWriteJSON(w, http.StatusOK, map[string]any{
		"signature_valid":         true,
		"within_validity_window":  withinValidityWindow,
		"time_status":             timeStatus,
		"receipt":                 receipt,
		"current_state_available": currentState != nil,
		"current_state_status":    stateStatus,
		"current_state":           currentState,
		"verification_scope":      "The NHS signature authenticates the immutable receipt fields. Current state reports later revocation, supersession, credit, and net commercial effect when online storage is available. The provider-reported business outcome is not independently audited.",
	})
}
