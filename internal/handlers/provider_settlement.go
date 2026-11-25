package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	gostripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/balancetransaction"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/unitedideas/nothumansearch/internal/models"
)

const providerSettlementStripeProduct = "nhs_provider_outcome_settlement"

type providerSettlementCheckoutCreator func(*gostripe.CheckoutSessionParams) (*gostripe.CheckoutSession, error)
type providerSettlementPaymentIntentRetriever func(string, *gostripe.PaymentIntentParams) (*gostripe.PaymentIntent, error)
type providerSettlementBalanceTransactionRetriever func(string, *gostripe.BalanceTransactionParams) (*gostripe.BalanceTransaction, error)

// ProviderSettlementHandler is an owner-only collection boundary. It does not
// participate in ranking, search, MCP discovery, agent checkout, or provider
// callbacks. A Stripe session is created only from a charged exact outcome
// already verified by the provider exchange.
type ProviderSettlementHandler struct {
	DB                         modelsProviderSettlementDB
	BaseURL                    string
	WebhookSecret              string
	prepareSettlement          func(modelsProviderSettlementDB, string) (*models.ProviderSettlementOrder, bool, error)
	recordCheckout             func(modelsProviderSettlementDB, string, string) (bool, error)
	recordPayment              func(modelsProviderSettlementDB, models.ProviderSettlementPaymentInput) (*models.ProviderSettlementPaymentReceipt, bool, error)
	recordAvailability         func(modelsProviderSettlementDB, models.ProviderSettlementAvailabilityInput) (string, bool, error)
	getSettlementStatus        func(modelsProviderSettlementDB, string) (*models.ProviderSettlementStatus, error)
	getProcessorReference      func(modelsProviderSettlementDB, string) (*models.ProviderSettlementProcessorReference, error)
	createCheckout             providerSettlementCheckoutCreator
	retrievePaymentIntent      providerSettlementPaymentIntentRetriever
	retrieveBalanceTransaction providerSettlementBalanceTransactionRetriever
	now                        func() time.Time
}

// modelsProviderSettlementDB intentionally stays a concrete database pointer
// alias. The handler function fields keep request behavior unit-testable while
// the model remains the only place that writes receipt rows.
type modelsProviderSettlementDB = *sql.DB

func NewProviderSettlementHandler(db *sql.DB, baseURL, webhookSecret string) *ProviderSettlementHandler {
	return &ProviderSettlementHandler{
		DB:                         db,
		BaseURL:                    strings.TrimSuffix(strings.TrimSpace(baseURL), "/"),
		WebhookSecret:              strings.TrimSpace(webhookSecret),
		prepareSettlement:          models.PrepareProviderSettlement,
		recordCheckout:             models.RecordProviderSettlementCheckoutSession,
		recordPayment:              models.RecordProviderSettlementPayment,
		recordAvailability:         models.RecordProviderSettlementAvailability,
		getSettlementStatus:        models.GetProviderSettlementStatus,
		getProcessorReference:      models.GetProviderSettlementProcessorReference,
		createCheckout:             session.New,
		retrievePaymentIntent:      paymentintent.Get,
		retrieveBalanceTransaction: balancetransaction.Get,
		now:                        time.Now,
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
	if h.WebhookSecret == "" {
		// Never create a payment opportunity that this binary is unable to
		// authenticate and turn into the required immutable payment receipt.
		providerWriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "stripe_webhook_not_configured"})
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
		"settlement":              status,
		"completed_paid_receipt":  status.Paid,
		"processor_net_available": status.ProcessorNetAvailable,
		"evidence_scope":          "A paid receipt proves gross collection. Retained processor net counts only after an exact Stripe balance transaction is separately observed as available.",
	})
}

type providerSettlementAvailabilityRequest struct {
	OrderID string `json:"order_id"`
}

// AdminRecordAvailability refreshes one private Stripe balance transaction and
// appends an immutable receipt only after Stripe reports its net as available.
func (h *ProviderSettlementHandler) AdminRecordAvailability(w http.ResponseWriter, r *http.Request) {
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
	var request providerSettlementAvailabilityRequest
	if err := decodeProviderJSON(w, r, &request); err != nil {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid settlement availability request"})
		return
	}
	getReference := h.getProcessorReference
	if getReference == nil {
		getReference = models.GetProviderSettlementProcessorReference
	}
	reference, err := getReference(h.DB, request.OrderID)
	if err != nil {
		h.writeSettlementError(w, err)
		return
	}
	retrieve := h.retrieveBalanceTransaction
	if retrieve == nil {
		retrieve = balancetransaction.Get
	}
	balance, err := retrieve(reference.StripeBalanceTransactionID, nil)
	if err != nil || balance == nil || string(balance.Status) != "available" {
		providerWriteJSON(w, http.StatusConflict, map[string]string{"error": "processor net is not yet verified as available"})
		return
	}
	availableOn := time.Unix(balance.AvailableOn, 0).UTC()
	if balance.ID != reference.StripeBalanceTransactionID || balance.Amount != reference.GrossAmountCents ||
		balance.Fee != reference.FeeCents || balance.Net != reference.NetCents ||
		strings.ToLower(string(balance.Currency)) != reference.Currency || !availableOn.Equal(reference.AvailableOn) {
		providerWriteJSON(w, http.StatusConflict, map[string]string{"error": "processor balance transaction conflicts with immutable receipt"})
		return
	}
	now := time.Now
	if h.now != nil {
		now = h.now
	}
	record := h.recordAvailability
	if record == nil {
		record = models.RecordProviderSettlementAvailability
	}
	_, created, err := record(h.DB, models.ProviderSettlementAvailabilityInput{
		OrderID: request.OrderID, StripeBalanceTransactionID: balance.ID,
		GrossAmountCents: balance.Amount, FeeCents: balance.Fee, NetCents: balance.Net,
		Currency: string(balance.Currency), AvailableOn: availableOn,
		ProcessorVerifiedAt: now().UTC(),
	})
	if err != nil {
		h.writeSettlementError(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	providerWriteJSON(w, status, map[string]any{
		"order_id": request.OrderID, "processor_net_available": true,
		"created":        created,
		"evidence_scope": "Stripe reports the exact immutable processor net as available; private Stripe identifiers are omitted.",
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
	retrieveIntent := h.retrievePaymentIntent
	if retrieveIntent == nil {
		retrieveIntent = paymentintent.Get
	}
	params := &gostripe.PaymentIntentParams{}
	params.AddExpand("latest_charge.balance_transaction")
	intent, err := retrieveIntent(stripeSession.PaymentIntent.ID, params)
	if err != nil || intent == nil || intent.ID != stripeSession.PaymentIntent.ID ||
		intent.Status != gostripe.PaymentIntentStatusSucceeded ||
		intent.AmountReceived != stripeSession.AmountTotal ||
		strings.ToLower(string(intent.Currency)) != "usd" || intent.LatestCharge == nil {
		return nil, false, models.ErrInvalidProviderExchange
	}
	charge := intent.LatestCharge
	if strings.TrimSpace(charge.ID) == "" || !charge.Paid || !charge.Captured || charge.Disputed ||
		charge.Refunded || charge.AmountRefunded != 0 || charge.AmountCaptured != stripeSession.AmountTotal ||
		charge.BalanceTransaction == nil || strings.TrimSpace(charge.BalanceTransaction.ID) == "" {
		return nil, false, models.ErrInvalidProviderExchange
	}
	retrieveBalance := h.retrieveBalanceTransaction
	if retrieveBalance == nil {
		retrieveBalance = balancetransaction.Get
	}
	balance, err := retrieveBalance(charge.BalanceTransaction.ID, nil)
	if err != nil || balance == nil || balance.ID != charge.BalanceTransaction.ID ||
		balance.Amount != stripeSession.AmountTotal || balance.Fee < 0 || balance.Net < 1 ||
		balance.Amount-balance.Fee != balance.Net || strings.ToLower(string(balance.Currency)) != "usd" ||
		(string(balance.Status) != "pending" && string(balance.Status) != "available") || balance.AvailableOn < 1 {
		return nil, false, models.ErrInvalidProviderExchange
	}
	now := time.Now
	if h.now != nil {
		now = h.now
	}
	recordPayment := h.recordPayment
	if recordPayment == nil {
		recordPayment = models.RecordProviderSettlementPayment
	}
	return recordPayment(h.DB, models.ProviderSettlementPaymentInput{
		OrderID:                    stripeSession.ClientReferenceID,
		StripeCheckoutSessionID:    stripeSession.ID,
		StripePaymentIntentID:      stripeSession.PaymentIntent.ID,
		StripeEventID:              stripeEventID,
		StripeChargeID:             charge.ID,
		StripeBalanceTransactionID: balance.ID,
		AmountCents:                stripeSession.AmountTotal,
		ProcessorFeeCents:          balance.Fee,
		ProcessorNetCents:          balance.Net,
		Currency:                   string(stripeSession.Currency),
		ProcessorStatus:            string(balance.Status),
		ProcessorAvailableOn:       time.Unix(balance.AvailableOn, 0).UTC(),
		ProcessorObservedAt:        now().UTC(),
		PaidAt:                     eventCreatedAt.UTC(),
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
