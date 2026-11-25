package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
)

type actionTicketRequest struct {
	OfferID          string   `json:"offer_id"`
	SearchID         string   `json:"search_id"`
	DemandTopic      string   `json:"demand_topic"`
	RegionCode       string   `json:"region_code,omitempty"`
	BudgetBand       string   `json:"budget_band,omitempty"`
	Urgency          string   `json:"urgency,omitempty"`
	RequirementFlags []string `json:"requirement_flags,omitempty"`
	PrincipalConsent bool     `json:"principal_consent"`
	ConsentVersion   string   `json:"consent_version"`
}

type actionTicketHandoffRequest struct {
	TicketID                                   string `json:"ticket_id"`
	AttributionToken                           string `json:"attribution_token"`
	PrincipalHandoffConsent                    bool   `json:"principal_handoff_consent"`
	HandoffConsentVersion                      string `json:"handoff_consent_version"`
	PrincipalControlledIntentDisclosureConsent bool   `json:"principal_controlled_intent_disclosure_consent,omitempty"`
	ControlledIntentDisclosureConsentVersion   string `json:"controlled_intent_disclosure_consent_version,omitempty"`
}

func (h *ProviderExchangeHandler) ActionTickets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	if h.Signer == nil {
		providerWriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "signed provider actions are not configured"})
		return
	}
	if !h.allowActionTicket(w, r) {
		return
	}
	var request actionTicketRequest
	if err := decodeProviderJSON(w, r, &request); err != nil {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid action ticket request; contact details and free-form prompts are not accepted"})
		return
	}
	prepared, err := h.prepareActionTicket(request)
	if err != nil {
		status, message := actionTicketPreparationStatus(err)
		providerWriteJSON(w, status, map[string]any{
			"error":             message,
			"principal_charged": false,
		})
		return
	}
	status := http.StatusCreated
	if replayed, _ := prepared["idempotent_replay"].(bool); replayed {
		status = http.StatusOK
	}
	providerWriteJSON(w, status, prepared)
}

// Ticket preparation never asks the principal to fund provider capacity. Keep
// provider-side budget and capacity failures on the offer-state/conflict path
// rather than presenting a machine-payment challenge to a free-search caller.
func actionTicketPreparationStatus(err error) (int, string) {
	status, message := providerExchangeStatus(err)
	if status == http.StatusPaymentRequired || status == http.StatusUnprocessableEntity {
		return http.StatusConflict, "provider-funded commercial capacity is unavailable; the principal is not charged"
	}
	return status, message
}

func (h *ProviderExchangeHandler) prepareActionTicket(request actionTicketRequest) (map[string]any, error) {
	ticket, offer, rawToken, err := models.CreateActionTicket(h.DB, models.ActionTicketInput{
		ProviderOfferID:       request.OfferID,
		SearchReceiptPublicID: request.SearchID,
		DemandTopic:           request.DemandTopic,
		RegionCode:            request.RegionCode,
		BudgetBand:            request.BudgetBand,
		Urgency:               request.Urgency,
		RequirementFlags:      request.RequirementFlags,
		PrincipalConsent:      request.PrincipalConsent,
		ConsentVersion:        request.ConsentVersion,
	}, h.Signer)
	if err != nil {
		return nil, err
	}
	publicOffer := publicProviderOfferView(*offer, h.BaseURL)
	return map[string]any{
		"ticket":                                       ticket,
		"offer":                                        publicOffer,
		"preparation_contract_version":                 models.ProviderActionTicketPreparationV2,
		"attribution_token":                            rawToken,
		"handoff_endpoint":                             h.BaseURL + "/api/v1/action-tickets/handoff",
		"handoff_method":                               http.MethodPost,
		"handoff_event_contract_version":               models.ProviderActionHandoffContractV1,
		"handoff_consent_contract_url":                 h.BaseURL + "/privacy#handoff-consent-v1",
		"controlled_intent_disclosure_optional":        true,
		"controlled_intent_disclosure_consent_version": models.ProviderControlledIntentDisclosureConsentV1,
		"controlled_intent_disclosure_consent_url":     h.BaseURL + "/privacy#controlled-intent-disclosure-consent-v1",
		"created":                                      !ticket.Replayed,
		"idempotent_replay":                            ticket.Replayed,
		"attribution_token_stored_by_nhs":              false,
		"token_reconstructed_for_exact_replay":         ticket.Replayed,
		"principal_consent_attested":                   true,
		"consent_contract_url":                         h.BaseURL + "/privacy#consent-v1",
		"principal_charged":                            false,
		"provider_mor_contract_required":               true,
		"principal_charged_by_nhs":                     false,
		"organic_rank_affected":                        false,
		"direct_provider_access_remains_free":          true,
		"disclosure":                                   "NHS may receive the disclosed provider-funded amount only if the provider reports the configured downstream outcome. Creating or handing off this ticket charges neither the provider nor the principal.",
	}, nil
}

// ActionTicketHandoff is the NHS-observed boundary between free discovery and
// the provider's own action surface. The bearer token stays in a no-store JSON
// body rather than an NHS URL/query string, and the response reveals no user,
// agent, query, network, or contact identity.
func (h *ProviderExchangeHandler) ActionTicketHandoff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	if !h.allowActionHandoff(w, r) {
		return
	}
	var request actionTicketHandoffRequest
	if err := decodeProviderJSON(w, r, &request); err != nil {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid action handoff request; contact details and free-form prompts are not accepted",
		})
		return
	}
	record := h.recordActionHandoff
	if record == nil {
		record = models.RecordActionTicketHandoff
	}
	request.TicketID = strings.ToLower(strings.TrimSpace(request.TicketID))
	request.AttributionToken = strings.TrimSpace(request.AttributionToken)
	request.HandoffConsentVersion = strings.TrimSpace(request.HandoffConsentVersion)
	request.ControlledIntentDisclosureConsentVersion = strings.TrimSpace(request.ControlledIntentDisclosureConsentVersion)
	ticket, receipt, err := record(h.DB, models.ProviderActionHandoffInput{
		ActionTicketID:                             request.TicketID,
		AttributionToken:                           request.AttributionToken,
		PrincipalHandoffConsent:                    request.PrincipalHandoffConsent,
		HandoffConsentVersion:                      request.HandoffConsentVersion,
		PrincipalControlledIntentDisclosureConsent: request.PrincipalControlledIntentDisclosureConsent,
		ControlledIntentDisclosureConsentVersion:   request.ControlledIntentDisclosureConsentVersion,
	})
	if err != nil {
		if errors.Is(err, models.ErrProviderPilotReviewRequired) {
			providerWriteJSON(w, http.StatusConflict, map[string]any{
				"error":                          "current ticket review required before provider handoff",
				"status":                         "review_pending",
				"review_pending":                 true,
				"review_contract_version":        models.ProviderPilotReviewContractV1,
				"review_type":                    "ticket",
				"subject_id":                     request.TicketID,
				"retryable":                      true,
				"observed_handoff":               false,
				"handoff_receipt_created":        false,
				"action_url_available":           false,
				"principal_charged":              false,
				"provider_charged":               false,
				"organic_rank_affected":          false,
				"direct_provider_access_is_free": true,
			})
			return
		}
		status, message := providerExchangeStatus(err)
		providerWriteJSON(w, status, map[string]any{
			"error":             message,
			"principal_charged": false,
			"provider_charged":  false,
		})
		return
	}
	actionURL, err := actionURLWithAttribution(ticket.ActionURLSnapshot, request.AttributionToken)
	if err != nil {
		providerWriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "provider action handoff could not be constructed",
		})
		return
	}
	status := http.StatusCreated
	if receipt.Replayed {
		status = http.StatusOK
	}
	providerWriteJSON(w, status, map[string]any{
		"ticket":                         ticket,
		"handoff_receipt":                receipt,
		"action_url":                     actionURL,
		"observed_handoff":               true,
		"idempotent_replay":              receipt.Replayed,
		"principal_charged":              false,
		"provider_charged":               false,
		"organic_rank_affected":          false,
		"direct_provider_access_is_free": true,
	})
}

func (h *ProviderExchangeHandler) allowActionTicket(w http.ResponseWriter, r *http.Request) bool {
	retry, ok := h.consumeActionTicketLimit(r, time.Now())
	if ok {
		return true
	}
	w.Header().Set("Retry-After", fmt.Sprintf("%d", max(1, int(retry.Seconds()))))
	providerWriteJSON(w, http.StatusTooManyRequests, map[string]string{"error": "action ticket safety limit exceeded"})
	return false
}

// consumeActionTicketLimit is shared by the REST and MCP action surfaces so
// neither route can bypass the stricter ticket-creation safety budget. A nil
// limiter fails closed; production handlers always receive one at construction.
func (h *ProviderExchangeHandler) consumeActionTicketLimit(r *http.Request, now time.Time) (time.Duration, bool) {
	if h == nil || h.ticketLimit == nil {
		return time.Hour, false
	}
	_, retry, ok := h.ticketLimit.allow("action-ticket:"+submitHashIP(r), now)
	return retry, ok
}

func (h *ProviderExchangeHandler) allowActionHandoff(w http.ResponseWriter, r *http.Request) bool {
	retry, ok := h.consumeActionHandoffLimit(r, time.Now())
	if h == nil || h.handoffLimit == nil {
		providerWriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "action handoff safety limit unavailable"})
		return false
	}
	if ok {
		return true
	}
	w.Header().Set("Retry-After", fmt.Sprintf("%d", max(1, int(retry.Seconds()))))
	providerWriteJSON(w, http.StatusTooManyRequests, map[string]string{"error": "action handoff safety limit exceeded"})
	return false
}

func (h *ProviderExchangeHandler) consumeActionHandoffLimit(r *http.Request, now time.Time) (time.Duration, bool) {
	if h == nil || h.handoffLimit == nil {
		return time.Hour, false
	}
	_, retry, ok := h.handoffLimit.allow("action-handoff:"+submitHashIP(r), now)
	return retry, ok
}
