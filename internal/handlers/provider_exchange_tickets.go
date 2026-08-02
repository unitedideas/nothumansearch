package handlers

import (
	"fmt"
	"net/http"
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
		status, message := providerExchangeStatus(err)
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
	actionURL, err := actionURLWithAttribution(ticket.ActionURLSnapshot, rawToken)
	if err != nil {
		return nil, err
	}
	publicOffer := publicProviderOfferView(*offer, h.BaseURL)
	return map[string]any{
		"ticket":                               ticket,
		"offer":                                publicOffer,
		"action_url":                           actionURL,
		"attribution_token":                    rawToken,
		"created":                              !ticket.Replayed,
		"idempotent_replay":                    ticket.Replayed,
		"attribution_token_stored_by_nhs":      false,
		"token_reconstructed_for_exact_replay": ticket.Replayed,
		"principal_consent_attested":           true,
		"consent_contract_url":                 h.BaseURL + "/privacy#consent-v1",
		"principal_charged":                    false,
		"provider_mor_contract_required":       true,
		"principal_charged_by_nhs":             false,
		"organic_rank_affected":                false,
		"direct_provider_access_remains_free":  true,
		"disclosure":                           "NHS may receive the disclosed provider-funded amount only if the provider reports the configured outcome. Creating this ticket does not charge the provider or the principal.",
	}, nil
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
