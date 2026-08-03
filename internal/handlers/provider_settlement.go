package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	gostripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/unitedideas/nothumansearch/internal/models"
)

const providerSettlementStripeProduct = "nhs_provider_outcome_settlement"

type providerSettlementCheckoutCreator func(*gostripe.CheckoutSessionParams) (*gostripe.CheckoutSession, error)

// ProviderSettlementHandler is an owner-only collection boundary. It does not
// participate in ranking, search, MCP discovery, agent checkout, or provider
// callbacks. A Stripe session is created only from a charged exact outcome
// already verified by the provider exchange.
type ProviderSettlementHandler struct {
	DB                  modelsProviderSettlementDB
	BaseURL             string
	prepareSettlement   func(modelsProviderSettlementDB, string) (*models.ProviderSettlementOrder, bool, error)
	recordCheckout      func(modelsProviderSettlementDB, string, string) (bool, error)
	recordPayment       func(modelsProviderSettlementDB, models.ProviderSettlementPaymentInput) (*models.ProviderSettlementPaymentReceipt, bool, error)
	getSettlementStatus func(modelsProviderSettlementDB, string) (*models.ProviderSettlementStatus, error)
	createCheckout      providerSettlementCheckoutCreator
}

// modelsProviderSettlementDB intentionally stays a concrete database pointer
// alias. The handler function fields keep request behavior unit-testable while
// the model remains the only place that writes receipt rows.
type modelsProviderSettlementDB = *sql.DB

func NewProviderSettlementHandler(db *sql.DB, baseURL string) *ProviderSettlementHandler {
	return &ProviderSettlementHandler{
		DB:                  db,
		BaseURL:             strings.TrimSuffix(strings.TrimSpace(baseURL), "/"),
		prepareSettlement:   models.PrepareProviderSettlement,
		recordCheckout:      models.RecordProviderSettlementCheckoutSession,
		recordPayment:       models.RecordProviderSettlementPayment,
		getSettlementStatus: models.GetProviderSettlementStatus,
		createCheckout:      session.New,
	}
}

type providerSettlementCheckoutRequest struct {
	OutcomeReceiptID string `json:"outcome_receipt_id"`
}

func (h *ProviderSettlementHandler) AdminCreateCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	if !requireAdminAPIKey(w, r) {
		return
	}
	if strings.TrimSpace(gostripe.Key) == "" {
		providerWriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "stripe_not_configured"})
		return
	}
	var request providerSettlementCheckoutRequest
	if err := decodeProviderJSON(w, r, &request); err != nil {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid settlement checkout request"})
		return
	}
	request.OutcomeReceiptID = strings.ToLower(strings.TrimSpace(request.OutcomeReceiptID))
	prepare := h.prepareSettlement
	if prepare == nil {
		prepare = models.PrepareProviderSettlement
	}
	order, created, err := prepare(h.DB, request.OutcomeReceiptID)
	if err != nil {
		h.writeSettlementError(w, err)
		return
	}
	if strings.TrimSpace(order.ProviderBillingEmail) == "" {
		providerWriteJSON(w, http.StatusConflict, map[string]string{"error": "provider billing account is unavailable"})
		return
	}
	params := &gostripe.CheckoutSessionParams{
		LineItems: []*gostripe.CheckoutSessionLineItemParams{{
			PriceData: &gostripe.CheckoutSessionLineItemPriceDataParams{
				Currency: gostripe.String(order.Currency),
				ProductData: &gostripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name:        gostripe.String("Not Human Search provider action settlement"),
					Description: gostripe.String("Settlement for one verified provider action under the accepted exact CPA terms."),
				},
				UnitAmount: gostripe.Int64(order.AmountCents),
			},
			Quantity: gostripe.Int64(1),
		}},
		Mode:              gostripe.String(string(gostripe.CheckoutSessionModePayment)),
		SuccessURL:        gostripe.String(h.providerSettlementReturnURL("completed")),
		CancelURL:         gostripe.String(h.providerSettlementReturnURL("cancelled")),
		CustomerEmail:     gostripe.String(order.ProviderBillingEmail),
		ClientReferenceID: gostripe.String(order.ID),
		PaymentIntentData: &gostripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: map[string]string{
				"product":                      providerSettlementStripeProduct,
				"provider_settlement_order_id": order.ID,
			},
		},
	}
	params.AddMetadata("product", providerSettlementStripeProduct)
	params.AddMetadata("provider_settlement_order_id", order.ID)
	params.SetIdempotencyKey("nhs-provider-settlement:" + order.ID)
	createCheckout := h.createCheckout
	if createCheckout == nil {
		createCheckout = session.New
	}
	stripeSession, err := createCheckout(params)
	if err != nil || stripeSession == nil || strings.TrimSpace(stripeSession.ID) == "" || strings.TrimSpace(stripeSession.URL) == "" {
		providerWriteJSON(w, http.StatusBadGateway, map[string]string{"error": "provider settlement checkout could not be created"})
		return
	}
	recordCheckout := h.recordCheckout
	if recordCheckout == nil {
		recordCheckout = models.RecordProviderSettlementCheckoutSession
	}
	checkoutCreated, err := recordCheckout(h.DB, order.ID, stripeSession.ID)
	if err != nil {
		h.writeSettlementError(w, err)
		return
	}
	status := http.StatusOK
	if created || checkoutCreated {
		status = http.StatusCreated
	}
	providerWriteJSON(w, status, map[string]any{
		"settlement_order_id": order.ID,
		"outcome_receipt_id":  order.OutcomeReceiptID,
		"amount_cents":        order.AmountCents,
		"currency":            order.Currency,
		"checkout_url":        stripeSession.URL,
		"payment_recorded":    false,
		"evidence_scope":      "A Stripe Checkout was created for the exact verified outcome. It is not a completed payment until the signed Stripe webhook appends a settlement receipt.",
	})
}

func (h *ProviderSettlementHandler) AdminStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}
	if !requireAdminAPIKey(w, r) {
		return
	}
	statusFn := h.getSettlementStatus
	if statusFn == nil {
		statusFn = models.GetProviderSettlementStatus
	}
	status, err := statusFn(h.DB, r.URL.Query().Get("order_id"))
	if err != nil {
		h.writeSettlementError(w, err)
		return
	}
	providerWriteJSON(w, http.StatusOK, map[string]any{
		"settlement":             status,
		"completed_paid_receipt": status.Paid,
		"evidence_scope":         "Only payment_receipt_id with paid_at proves collection; a checkout-created order is not revenue.",
	})
}

// RecordPaidCheckout is called only after FixHandler has signature-verified a
// Stripe checkout.session.completed event. It refuses redirect-only, unpaid,
// mismatched, or hand-assembled events before any database write.
func (h *ProviderSettlementHandler) RecordPaidCheckout(stripeSession *gostripe.CheckoutSession, stripeEventID string, eventCreatedAt time.Time) (*models.ProviderSettlementPaymentReceipt, bool, error) {
	if stripeSession == nil || stripeSession.Metadata["product"] != providerSettlementStripeProduct ||
		stripeSession.PaymentStatus != gostripe.CheckoutSessionPaymentStatusPaid ||
		strings.TrimSpace(stripeSession.ClientReferenceID) == "" ||
		strings.TrimSpace(stripeSession.Metadata["provider_settlement_order_id"]) == "" ||
		stripeSession.ClientReferenceID != stripeSession.Metadata["provider_settlement_order_id"] ||
		stripeSession.PaymentIntent == nil || strings.TrimSpace(stripeSession.PaymentIntent.ID) == "" ||
		strings.TrimSpace(stripeSession.ID) == "" || strings.TrimSpace(stripeEventID) == "" ||
		stripeSession.AmountTotal < 1 || strings.ToLower(string(stripeSession.Currency)) != "usd" ||
		eventCreatedAt.IsZero() {
		return nil, false, models.ErrInvalidProviderExchange
	}
	recordPayment := h.recordPayment
	if recordPayment == nil {
		recordPayment = models.RecordProviderSettlementPayment
	}
	return recordPayment(h.DB, models.ProviderSettlementPaymentInput{
		OrderID:                 stripeSession.ClientReferenceID,
		StripeCheckoutSessionID: stripeSession.ID,
		StripePaymentIntentID:   stripeSession.PaymentIntent.ID,
		StripeEventID:           stripeEventID,
		AmountCents:             stripeSession.AmountTotal,
		Currency:                string(stripeSession.Currency),
		PaidAt:                  eventCreatedAt.UTC(),
	})
}

func (h *ProviderSettlementHandler) providerSettlementReturnURL(status string) string {
	base := strings.TrimSuffix(h.BaseURL, "/")
	if base == "" {
		base = "https://nothumansearch.ai"
	}
	return base + "/providers?settlement=" + status
}

func (h *ProviderSettlementHandler) writeSettlementError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrInvalidProviderExchange):
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid provider settlement request"})
	case errors.Is(err, models.ErrProviderSettlementNotEligible):
		providerWriteJSON(w, http.StatusConflict, map[string]string{"error": "outcome is not eligible for a provider settlement"})
	case errors.Is(err, models.ErrProviderSettlementConflict), errors.Is(err, models.ErrProviderIdempotency):
		providerWriteJSON(w, http.StatusConflict, map[string]string{"error": "provider settlement conflicts with an immutable record"})
	case errors.Is(err, sql.ErrNoRows):
		providerWriteJSON(w, http.StatusNotFound, map[string]string{"error": "provider settlement was not found"})
	default:
		providerWriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "provider settlement could not be processed"})
	}
}
