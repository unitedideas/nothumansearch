package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/unitedideas/nothumansearch/internal/models"

	gostripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
)

type APIKeyHandler struct {
	DB      *sql.DB
	BaseURL string
	Auth    *AuthService
}

func NewAPIKeyHandler(db *sql.DB, baseURL string) *APIKeyHandler {
	return &APIKeyHandler{DB: db, BaseURL: baseURL}
}

func apiSubscribeMetadataValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 500 {
		return value[:500]
	}
	return value
}

func apiSubscribeAttributionFromRequest(r *http.Request) map[string]string {
	values := map[string]string{}
	for _, key := range []string{"qc", "utm_source", "utm_medium", "utm_campaign"} {
		if value := apiSubscribeMetadataValue(r.FormValue(key)); value != "" {
			values[key] = value
		}
	}
	return values
}

func apiSubscribeMetadata(email string, plan models.APIPlan, attribution map[string]string) map[string]string {
	metadata := map[string]string{
		"tenant":        "nothumansearch",
		"product":       "nhs_api_subscription",
		"product_id":    "nhs_api_" + plan.Name,
		"source":        "api_key_subscribe",
		"plan":          plan.Name,
		"monthly_limit": fmt.Sprintf("%d", plan.MonthlyLimit),
		"email":         email,
	}
	for key, value := range attribution {
		if clean := apiSubscribeMetadataValue(value); clean != "" {
			metadata[key] = clean
		}
	}
	return metadata
}

func (h *APIKeyHandler) subscribeCancelURL(email string, attribution map[string]string) string {
	values := url.Values{}
	if email = strings.TrimSpace(email); email != "" {
		values.Set("email", email)
	}
	for _, key := range []string{"qc", "utm_source", "utm_medium", "utm_campaign"} {
		if value := attribution[key]; value != "" {
			values.Set(key, value)
		}
	}
	if encoded := values.Encode(); encoded != "" {
		return h.BaseURL + "/subscribe?" + encoded
	}
	return h.BaseURL + "/subscribe"
}

func (h *APIKeyHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.subscribeDocument(w)
		return
	case http.MethodPost:
		// Continue below.
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Email       string `json:"email"`
		Plan        string `json:"plan"`
		QC          string `json:"qc"`
		UTMSource   string `json:"utm_source"`
		UTMMedium   string `json:"utm_medium"`
		UTMCampaign string `json:"utm_campaign"`
	}
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		_ = json.NewDecoder(r.Body).Decode(&req)
	} else {
		_ = r.ParseForm()
		req.Email = r.Form.Get("email")
		req.Plan = r.Form.Get("plan")
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	plan := models.APIPlanFor(req.Plan)
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		writeFixJSON(w, http.StatusBadRequest, map[string]any{"error": "valid email required"})
		return
	}
	if gostripe.Key == "" {
		writeFixJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "stripe_not_configured"})
		return
	}

	attribution := apiSubscribeAttributionFromRequest(r)
	for key, value := range map[string]string{
		"qc":           req.QC,
		"utm_source":   req.UTMSource,
		"utm_medium":   req.UTMMedium,
		"utm_campaign": req.UTMCampaign,
	} {
		if clean := apiSubscribeMetadataValue(value); clean != "" {
			attribution[key] = clean
		}
	}
	metadata := apiSubscribeMetadata(req.Email, plan, attribution)
	params := &gostripe.CheckoutSessionParams{
		LineItems: []*gostripe.CheckoutSessionLineItemParams{{
			PriceData: &gostripe.CheckoutSessionLineItemPriceDataParams{
				Currency: gostripe.String("usd"),
				ProductData: &gostripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name:        gostripe.String("Not Human Search Priority API"),
					Description: gostripe.String(fmt.Sprintf("%d priority-throughput REST/MCP calls per month; baseline discovery remains free afterward", plan.MonthlyLimit)),
				},
				Recurring: &gostripe.CheckoutSessionLineItemPriceDataRecurringParams{
					Interval: gostripe.String("month"),
				},
				UnitAmount: gostripe.Int64(plan.PriceCents),
			},
			Quantity: gostripe.Int64(1),
		}},
		Mode:          gostripe.String(string(gostripe.CheckoutSessionModeSubscription)),
		SuccessURL:    gostripe.String(h.BaseURL + "/api/v1/api-keys/activate?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:     gostripe.String(h.subscribeCancelURL(req.Email, attribution)),
		CustomerEmail: gostripe.String(req.Email),
		Metadata:      metadata,
		SubscriptionData: &gostripe.CheckoutSessionSubscriptionDataParams{
			Metadata: metadata,
		},
	}
	s, err := session.New(params)
	if err != nil {
		writeFixJSON(w, http.StatusBadGateway, map[string]any{"error": "checkout_unavailable"})
		return
	}
	writeFixJSON(w, http.StatusOK, map[string]any{
		"checkout_url":   s.URL,
		"plan":           plan.Name,
		"monthly_limit":  plan.MonthlyLimit,
		"amount_cents":   plan.PriceCents,
		"activation_url": h.BaseURL + "/api/v1/api-keys/activate?session_id={CHECKOUT_SESSION_ID}",
	})
}

func (h *APIKeyHandler) subscribeDocument(w http.ResponseWriter) {
	plans := make([]map[string]any, 0, len(models.APIPlans()))
	for _, plan := range models.APIPlans() {
		plans = append(plans, map[string]any{
			"id":            "nhs_api_" + plan.Name,
			"plan":          plan.Name,
			"name":          "Not Human Search Priority API",
			"monthly_limit": plan.MonthlyLimit,
			"benefit":       "Higher hourly safety ceilings for REST, MCP tools, and live checks. Organic results and baseline discovery remain free and identical.",
			"price": map[string]any{
				"amount":   plan.PriceCents,
				"currency": "USD",
				"display":  fmt.Sprintf("$%.2f/mo", float64(plan.PriceCents)/100),
			},
		})
	}
	writeFixJSON(w, http.StatusOK, map[string]any{
		"seller":         "nothumansearch",
		"product_family": "api_keys",
		"description":    "Optional priority-throughput API keys. Search, site details, organic rank, and baseline REST/MCP discovery remain free without a key; exhausted allocations fall back to free safety limits.",
		"plans":          plans,
		"subscribe": map[string]any{
			"method":          "POST",
			"endpoint":        h.BaseURL + "/api/v1/api-keys/subscribe",
			"content_type":    "application/json",
			"required_fields": []string{"email", "plan"},
			"allowed_plans":   []string{"unlimited"},
			"example_body": map[string]string{
				"email": "buyer@example.com",
				"plan":  "unlimited",
			},
			"response_fields": []string{"checkout_url", "plan", "monthly_limit", "amount_cents", "activation_url"},
		},
		"activate_after_checkout": h.BaseURL + "/api/v1/api-keys/activate?session_id={CHECKOUT_SESSION_ID}",
	})
}

func (h *APIKeyHandler) Activate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	if sessionID == "" {
		writeFixJSON(w, http.StatusBadRequest, map[string]any{"error": "session_id required"})
		return
	}
	if gostripe.Key == "" {
		writeFixJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "stripe_not_configured"})
		return
	}
	s, err := session.Get(sessionID, nil)
	if err != nil {
		writeFixJSON(w, http.StatusBadGateway, map[string]any{"error": "checkout_lookup_failed"})
		return
	}
	if s.Metadata["product"] != "nhs_api_subscription" || s.PaymentStatus != gostripe.CheckoutSessionPaymentStatusPaid {
		writeFixJSON(w, http.StatusPaymentRequired, map[string]any{"error": "checkout_not_paid"})
		return
	}
	email := s.CustomerEmail
	if email == "" {
		email = s.Metadata["email"]
	}
	customerID := ""
	if s.Customer != nil {
		customerID = s.Customer.ID
	}
	subscriptionID := ""
	if s.Subscription != nil {
		subscriptionID = s.Subscription.ID
	}
	raw, key, err := models.ActivateAPIKeyForCheckout(h.DB, sessionID, email, s.Metadata["plan"], customerID, subscriptionID)
	if err != nil {
		writeFixJSON(w, http.StatusInternalServerError, map[string]any{"error": "api_key_create_failed"})
		return
	}

	// Same subscription powers the human side: activate the account and sign the
	// buyer in immediately (session cookie) so they can search the website now.
	if _, aerr := models.SetAccountSubscription(h.DB, email, customerID, subscriptionID, s.Metadata["plan"], "active"); aerr != nil {
		log.Printf("activate: account subscription update failed for %s: %v", email, aerr)
	}
	if h.Auth != nil && email != "" {
		if serr := h.Auth.StartSession(w, r, email); serr != nil {
			log.Printf("activate: instant sign-in failed for %s: %v", email, serr)
		}
	}
	resp := map[string]any{
		"plan":          key.Plan,
		"monthly_limit": key.MonthlyLimit,
		"key_prefix":    key.KeyPrefix,
		"message":       "API key already activated for this checkout session; raw keys are only shown once.",
	}
	if raw != "" {
		resp["api_key"] = raw
		resp["message"] = "Store this API key now. It is shown once."
	}
	writeFixJSON(w, http.StatusOK, resp)
}
