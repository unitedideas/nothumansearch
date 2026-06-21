package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
		Email string `json:"email"`
		Plan  string `json:"plan"`
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

	metadata := map[string]string{
		"product":       "nhs_api_subscription",
		"plan":          plan.Name,
		"monthly_limit": fmt.Sprintf("%d", plan.MonthlyLimit),
		"email":         req.Email,
	}
	params := &gostripe.CheckoutSessionParams{
		LineItems: []*gostripe.CheckoutSessionLineItemParams{{
			PriceData: &gostripe.CheckoutSessionLineItemPriceDataParams{
				Currency: gostripe.String("usd"),
				ProductData: &gostripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name:        gostripe.String("Not Human Search API " + strings.Title(plan.Name)),
					Description: gostripe.String(fmt.Sprintf("%d billable API/MCP calls per month", plan.MonthlyLimit)),
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
		CancelURL:     gostripe.String(h.BaseURL + "/"),
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
			"name":          "Not Human Search API " + strings.Title(plan.Name),
			"monthly_limit": plan.MonthlyLimit,
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
		"description":    "Paid API keys for Not Human Search REST and MCP usage after the anonymous quota.",
		"plans":          plans,
		"subscribe": map[string]any{
			"method":          "POST",
			"endpoint":        h.BaseURL + "/api/v1/api-keys/subscribe",
			"content_type":    "application/json",
			"required_fields": []string{"email", "plan"},
			"allowed_plans":   []string{"starter", "pro", "scale"},
			"example_body": map[string]string{
				"email": "buyer@example.com",
				"plan":  "starter",
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
