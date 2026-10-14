package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
)

const (
	providerMaximumBountyCents         int64 = models.ProviderBountyMaximumCents
	providerMaximumPrincipalPriceCents int64 = models.ProviderPrincipalPriceMaximumCents
	providerMaximumTermsCreditCents    int64 = models.ProviderTermsCreditMaximumCents
)

type providerOfferRequest struct {
	ClaimID               string `json:"claim_id,omitempty"`
	Name                  string `json:"name"`
	Summary               string `json:"summary"`
	ActionType            string `json:"action_type"`
	ActionURL             string `json:"action_url"`
	ChargeEvent           string `json:"charge_event"`
	BountyCents           int64  `json:"bounty_cents"`
	Currency              string `json:"currency"`
	PrincipalPriceMode    string `json:"principal_price_mode"`
	PrincipalPriceCents   *int64 `json:"principal_price_cents,omitempty"`
	PrincipalCurrency     string `json:"principal_currency"`
	BillingMode           string `json:"billing_mode"`
	TermsCreditLimitCents *int64 `json:"terms_credit_limit_cents,omitempty"`
	TermsPeriodDays       *int   `json:"terms_period_days,omitempty"`
}

func (h *ProviderExchangeHandler) Offers(w http.ResponseWriter, r *http.Request) {
	account := h.requireAccount(w, r, r.Method != http.MethodGet)
	if account == nil {
		return
	}
	switch r.Method {
	case http.MethodGet:
		claimID := strings.TrimSpace(r.URL.Query().Get("claim_id"))
		offers, err := models.ListProviderOffers(h.DB, account.ID, claimID)
		if err != nil {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		providerWriteJSON(w, http.StatusOK, map[string]any{"offers": offers})
	case http.MethodPost:
		if !h.allowOfferMutation(w, r, account.ID) {
			return
		}
		var request providerOfferRequest
		if err := decodeProviderJSON(w, r, &request); err != nil {
			providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid offer request"})
			return
		}
		claim, err := models.GetProviderClaim(h.DB, account.ID, request.ClaimID)
		if err != nil {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		input, err := h.providerOfferInput(request, claim.Domain)
		if err != nil {
			providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		offer, err := models.CreateProviderOffer(h.DB, account.ID, request.ClaimID, input)
		if err != nil {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		providerWriteJSON(w, http.StatusCreated, map[string]any{
			"offer":             offer,
			"commercial_status": "draft",
			"note":              "A draft cannot appear beside organic results until the provider accepts its exact capped CPA terms, the owner verifies that acceptance, and an administrator activates it.",
		})
	default:
		w.Header().Set("Allow", "GET, POST")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET or POST required"})
	}
}

func (h *ProviderExchangeHandler) OfferAction(w http.ResponseWriter, r *http.Request) {
	account := h.requireAccount(w, r, r.Method != http.MethodGet)
	if account == nil {
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/provider/offers/"), "/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		providerWriteJSON(w, http.StatusNotFound, map[string]string{"error": "offer not found"})
		return
	}
	offerID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = strings.Join(parts[1:], "/")
	}
	if r.Method == http.MethodGet && action == "" {
		offer, err := models.GetProviderOffer(h.DB, account.ID, offerID)
		if err != nil {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		providerWriteJSON(w, http.StatusOK, map[string]any{"offer": offer})
		return
	}
	if r.Method == http.MethodPut && action == "" {
		if !h.allowOfferMutation(w, r, account.ID) {
			return
		}
		current, err := models.GetProviderOffer(h.DB, account.ID, offerID)
		if err != nil {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		var request providerOfferRequest
		if err := decodeProviderJSON(w, r, &request); err != nil {
			providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid offer request"})
			return
		}
		input, err := h.providerOfferInput(request, current.Domain)
		if err != nil {
			providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		updated, err := models.UpdateProviderOffer(h.DB, account.ID, offerID, input)
		if err != nil {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		providerWriteJSON(w, http.StatusOK, map[string]any{"offer": updated})
		return
	}
	if r.Method == http.MethodPost && action == "pause" {
		if !h.allowOfferMutation(w, r, account.ID) {
			return
		}
		paused, err := models.PauseProviderOffer(h.DB, account.ID, offerID)
		if err != nil {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		providerWriteJSON(w, http.StatusOK, map[string]any{"offer": paused})
		return
	}
	w.Header().Set("Allow", "GET, PUT, POST")
	providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "unsupported offer action"})
}

func (h *ProviderExchangeHandler) providerOfferInput(request providerOfferRequest, claimedDomain string) (models.ProviderOfferInput, error) {
	actionURL, err := normalizeProviderActionURL(request.ActionURL, claimedDomain)
	if err != nil {
		return models.ProviderOfferInput{}, err
	}
	if request.BountyCents < 1 || request.BountyCents > providerMaximumBountyCents {
		return models.ProviderOfferInput{}, models.ErrInvalidProviderExchange
	}
	if request.PrincipalPriceCents != nil && (*request.PrincipalPriceCents < 0 || *request.PrincipalPriceCents > providerMaximumPrincipalPriceCents) {
		return models.ProviderOfferInput{}, models.ErrInvalidProviderExchange
	}
	request.Currency = strings.ToLower(strings.TrimSpace(request.Currency))
	request.PrincipalCurrency = strings.ToLower(strings.TrimSpace(request.PrincipalCurrency))
	request.BillingMode = strings.ToLower(strings.TrimSpace(request.BillingMode))
	if request.Currency != "usd" || request.PrincipalCurrency != "usd" {
		return models.ProviderOfferInput{}, models.ErrInvalidProviderExchange
	}
	if request.BillingMode == models.ProviderPilotBillingMode {
		if request.TermsCreditLimitCents == nil || request.TermsPeriodDays == nil ||
			*request.TermsCreditLimitCents < request.BountyCents || *request.TermsCreditLimitCents > providerMaximumTermsCreditCents ||
			*request.TermsPeriodDays < 1 || *request.TermsPeriodDays > 90 {
			return models.ProviderOfferInput{}, models.ErrInvalidProviderExchange
		}
	} else {
		return models.ProviderOfferInput{}, models.ErrInvalidProviderExchange
	}
	return models.ProviderOfferInput{
		OfferName:              request.Name,
		OfferSummary:           request.Summary,
		ActionType:             request.ActionType,
		ActionURL:              actionURL,
		ChargeEvent:            request.ChargeEvent,
		BountyCents:            request.BountyCents,
		Currency:               request.Currency,
		PrincipalPriceMode:     request.PrincipalPriceMode,
		PrincipalPriceCents:    request.PrincipalPriceCents,
		PrincipalCurrency:      request.PrincipalCurrency,
		BillingMode:            request.BillingMode,
		TermsCreditLimitCents:  request.TermsCreditLimitCents,
		TermsPeriodDays:        request.TermsPeriodDays,
		TermsEvidenceReference: "",
	}, nil
}

func (h *ProviderExchangeHandler) allowOfferMutation(w http.ResponseWriter, r *http.Request, accountID int64) bool {
	if h.offerLimit == nil {
		h.offerLimit = newMCPDiscoveryRateLimiter(60, time.Hour)
	}
	now := time.Now()
	_, accountRetry, accountOK := h.offerLimit.allow(fmt.Sprintf("offer-account:%d", accountID), now)
	_, networkRetry, networkOK := h.offerLimit.allow("offer-network:"+submitHashIP(r), now)
	if accountOK && networkOK {
		return true
	}
	retry := accountRetry
	if networkRetry > retry {
		retry = networkRetry
	}
	w.Header().Set("Retry-After", fmt.Sprintf("%d", max(1, int(retry.Seconds()))))
	providerWriteJSON(w, http.StatusTooManyRequests, map[string]string{"error": "provider offer mutation safety limit exceeded"})
	return false
}
