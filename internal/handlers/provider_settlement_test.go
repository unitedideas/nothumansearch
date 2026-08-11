package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gostripe "github.com/stripe/stripe-go/v82"
	"github.com/unitedideas/nothumansearch/internal/models"
)

func TestProviderSettlementCheckoutUsesOnlyFrozenOutcomeAmount(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "provider-settlement-test-admin")
	priorStripeKey := gostripe.Key
	gostripe.Key = "sk_test_provider_settlement"
	t.Cleanup(func() { gostripe.Key = priorStripeKey })

	order := &models.ProviderSettlementOrder{
		ID:                   "11111111-1111-4111-8111-111111111111",
		OutcomeReceiptID:     "22222222-2222-4222-8222-222222222222",
		AmountCents:          4750,
		Currency:             "usd",
		ProviderBillingEmail: "billing@example.com",
	}
	checkoutRecorded := false
	h := &ProviderSettlementHandler{
		BaseURL:       "https://nothumansearch.ai",
		WebhookSecret: "whsec_provider_settlement_test",
		prepareSettlement: func(_ modelsProviderSettlementDB, outcomeID string) (*models.ProviderSettlementOrder, bool, error) {
			if outcomeID != order.OutcomeReceiptID {
				t.Fatalf("outcome id=%q", outcomeID)
			}
			return order, true, nil
		},
		createCheckout: func(params *gostripe.CheckoutSessionParams) (*gostripe.CheckoutSession, error) {
			if got := params.Metadata["product"]; got != providerSettlementStripeProduct {
				t.Fatalf("checkout product=%q", got)
			}
			if got := params.Metadata["provider_settlement_order_id"]; got != order.ID {
				t.Fatalf("checkout order id=%q", got)
			}
			if params.IdempotencyKey == nil || *params.IdempotencyKey != "nhs-provider-settlement:"+order.ID {
				t.Fatalf("checkout idempotency key=%v", params.IdempotencyKey)
			}
			if params.CustomerEmail == nil || *params.CustomerEmail != order.ProviderBillingEmail {
				t.Fatalf("checkout email=%v", params.CustomerEmail)
			}
			if len(params.LineItems) != 1 || params.LineItems[0].PriceData == nil ||
				params.LineItems[0].PriceData.UnitAmount == nil || *params.LineItems[0].PriceData.UnitAmount != order.AmountCents {
				t.Fatalf("checkout did not use exact frozen amount")
			}
			return &gostripe.CheckoutSession{ID: "cs_12345678", URL: "https://checkout.stripe.test/session"}, nil
		},
		recordCheckout: func(_ modelsProviderSettlementDB, orderID, checkoutID string) (bool, error) {
			if orderID != order.ID || checkoutID != "cs_12345678" {
				t.Fatalf("checkout binding order=%q session=%q", orderID, checkoutID)
			}
			checkoutRecorded = true
			return true, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/provider-settlements/checkout", bytes.NewBufferString(`{"outcome_receipt_id":"22222222-2222-4222-8222-222222222222"}`))
	req.Header.Set("Authorization", "Bearer provider-settlement-test-admin")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.AdminCreateCheckout(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !checkoutRecorded {
		t.Fatal("checkout session was not bound")
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["amount_cents"] != float64(order.AmountCents) || body["payment_recorded"] != false {
		t.Fatalf("checkout body=%v", body)
	}
	if _, leaked := body["provider_billing_email"]; leaked {
		t.Fatalf("provider billing email leaked: %v", body)
	}
}

func TestProviderSettlementCheckoutRequiresWebhookVerification(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "provider-settlement-test-admin")
	priorStripeKey := gostripe.Key
	gostripe.Key = "sk_test_provider_settlement"
	t.Cleanup(func() { gostripe.Key = priorStripeKey })
	called := false
	h := &ProviderSettlementHandler{
		prepareSettlement: func(_ modelsProviderSettlementDB, _ string) (*models.ProviderSettlementOrder, bool, error) {
			called = true
			return nil, false, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/provider-settlements/checkout", bytes.NewBufferString(`{"outcome_receipt_id":"22222222-2222-4222-8222-222222222222"}`))
	req.Header.Set("Authorization", "Bearer provider-settlement-test-admin")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.AdminCreateCheckout(rr, req)
	if rr.Code != http.StatusServiceUnavailable || called {
		t.Fatalf("status=%d prepare_called=%t body=%s", rr.Code, called, rr.Body.String())
	}
}

func TestProviderSettlementCheckoutRejectsCallerSuppliedPrice(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "provider-settlement-test-admin")
	priorStripeKey := gostripe.Key
	gostripe.Key = "sk_test_provider_settlement"
	t.Cleanup(func() { gostripe.Key = priorStripeKey })
	called := false
	h := &ProviderSettlementHandler{
		WebhookSecret: "whsec_provider_settlement_test",
		prepareSettlement: func(_ modelsProviderSettlementDB, _ string) (*models.ProviderSettlementOrder, bool, error) {
			called = true
			return nil, false, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/provider-settlements/checkout", bytes.NewBufferString(`{"outcome_receipt_id":"22222222-2222-4222-8222-222222222222","amount_cents":1}`))
	req.Header.Set("Authorization", "Bearer provider-settlement-test-admin")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.AdminCreateCheckout(rr, req)
	if rr.Code != http.StatusBadRequest || called {
		t.Fatalf("status=%d prepare_called=%t body=%s", rr.Code, called, rr.Body.String())
	}
}

func TestProviderSettlementRejectsUnpaidCheckoutBeforeDatabase(t *testing.T) {
	h := &ProviderSettlementHandler{}
	_, _, err := h.RecordPaidCheckout(&gostripe.CheckoutSession{
		Metadata:          map[string]string{"product": providerSettlementStripeProduct, "provider_settlement_order_id": "11111111-1111-4111-8111-111111111111"},
		ClientReferenceID: "11111111-1111-4111-8111-111111111111",
		PaymentStatus:     gostripe.CheckoutSessionPaymentStatusUnpaid,
		ID:                "cs_12345678",
		AmountTotal:       4750,
		Currency:          gostripe.CurrencyUSD,
		PaymentIntent:     &gostripe.PaymentIntent{ID: "pi_12345678"},
	}, "evt_12345678", time.Now())
	if err != models.ErrInvalidProviderExchange {
		t.Fatalf("unpaid checkout error=%v", err)
	}
}

func TestProviderSettlementRecordsOnlyExactPaidCheckout(t *testing.T) {
	called := false
	availableOn := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	h := &ProviderSettlementHandler{
		retrievePaymentIntent: func(id string, _ *gostripe.PaymentIntentParams) (*gostripe.PaymentIntent, error) {
			return &gostripe.PaymentIntent{
				ID: id, Status: gostripe.PaymentIntentStatusSucceeded, AmountReceived: 4750,
				Currency: gostripe.CurrencyUSD,
				LatestCharge: &gostripe.Charge{
					ID: "ch_12345678", Paid: true, Captured: true,
					AmountCaptured:     4750,
					BalanceTransaction: &gostripe.BalanceTransaction{ID: "txn_12345678"},
				},
			}, nil
		},
		retrieveBalanceTransaction: func(id string, _ *gostripe.BalanceTransactionParams) (*gostripe.BalanceTransaction, error) {
			return &gostripe.BalanceTransaction{
				ID: id, Amount: 4750, Fee: 168, Net: 4582,
				Currency: gostripe.CurrencyUSD, Status: gostripe.BalanceTransactionStatusPending,
				AvailableOn: availableOn.Unix(),
			}, nil
		},
		now: func() time.Time { return time.Date(2026, 8, 3, 12, 1, 0, 0, time.UTC) },
		recordPayment: func(_ modelsProviderSettlementDB, input models.ProviderSettlementPaymentInput) (*models.ProviderSettlementPaymentReceipt, bool, error) {
			called = true
			if input.OrderID != "11111111-1111-4111-8111-111111111111" || input.AmountCents != 4750 || input.Currency != "usd" ||
				input.StripeCheckoutSessionID != "cs_12345678" || input.StripePaymentIntentID != "pi_12345678" || input.StripeEventID != "evt_12345678" ||
				input.StripeChargeID != "ch_12345678" || input.StripeBalanceTransactionID != "txn_12345678" ||
				input.ProcessorFeeCents != 168 || input.ProcessorNetCents != 4582 || input.ProcessorStatus != "pending" ||
				!input.ProcessorAvailableOn.Equal(availableOn) {
				t.Fatalf("payment input=%+v", input)
			}
			return &models.ProviderSettlementPaymentReceipt{ID: "33333333-3333-4333-8333-333333333333", OrderID: input.OrderID, AmountCents: input.AmountCents, Currency: input.Currency}, true, nil
		},
	}
	receipt, created, err := h.RecordPaidCheckout(&gostripe.CheckoutSession{
		Metadata:          map[string]string{"product": providerSettlementStripeProduct, "provider_settlement_order_id": "11111111-1111-4111-8111-111111111111"},
		ClientReferenceID: "11111111-1111-4111-8111-111111111111",
		PaymentStatus:     gostripe.CheckoutSessionPaymentStatusPaid,
		ID:                "cs_12345678",
		AmountTotal:       4750,
		Currency:          gostripe.CurrencyUSD,
		PaymentIntent:     &gostripe.PaymentIntent{ID: "pi_12345678"},
	}, "evt_12345678", time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC))
	if err != nil || !created || !called || receipt == nil {
		t.Fatalf("receipt=%+v created=%t called=%t err=%v", receipt, created, called, err)
	}
}

func TestProviderSettlementAvailabilityUsesExactPrivateBalanceReference(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "provider-settlement-test-admin")
	priorStripeKey := gostripe.Key
	gostripe.Key = "sk_test_provider_settlement"
	t.Cleanup(func() { gostripe.Key = priorStripeKey })
	availableOn := time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC)
	verifiedAt := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	recorded := false
	h := &ProviderSettlementHandler{
		getProcessorReference: func(_ modelsProviderSettlementDB, orderID string) (*models.ProviderSettlementProcessorReference, error) {
			if orderID != "11111111-1111-4111-8111-111111111111" {
				t.Fatalf("order id=%q", orderID)
			}
			return &models.ProviderSettlementProcessorReference{
				StripeBalanceTransactionID: "txn_12345678", GrossAmountCents: 4750,
				FeeCents: 168, NetCents: 4582, Currency: "usd", AvailableOn: availableOn,
			}, nil
		},
		retrieveBalanceTransaction: func(id string, _ *gostripe.BalanceTransactionParams) (*gostripe.BalanceTransaction, error) {
			if id != "txn_12345678" {
				t.Fatalf("balance transaction id=%q", id)
			}
			return &gostripe.BalanceTransaction{
				ID: id, Amount: 4750, Fee: 168, Net: 4582, Currency: gostripe.CurrencyUSD,
				Status: gostripe.BalanceTransactionStatusAvailable, AvailableOn: availableOn.Unix(),
			}, nil
		},
		recordAvailability: func(_ modelsProviderSettlementDB, input models.ProviderSettlementAvailabilityInput) (string, bool, error) {
			if input.StripeBalanceTransactionID != "txn_12345678" || input.GrossAmountCents != 4750 ||
				input.FeeCents != 168 || input.NetCents != 4582 || input.Currency != "usd" ||
				!input.AvailableOn.Equal(availableOn) || !input.ProcessorVerifiedAt.Equal(verifiedAt) {
				t.Fatalf("availability input=%+v", input)
			}
			recorded = true
			return "44444444-4444-4444-8444-444444444444", true, nil
		},
		now: func() time.Time { return verifiedAt },
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/provider-settlements/availability", bytes.NewBufferString(`{"order_id":"11111111-1111-4111-8111-111111111111"}`))
	req.Header.Set("Authorization", "Bearer provider-settlement-test-admin")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.AdminRecordAvailability(rr, req)
	if rr.Code != http.StatusCreated || !recorded {
		t.Fatalf("status=%d recorded=%t body=%s", rr.Code, recorded, rr.Body.String())
	}
	if bytes.Contains(rr.Body.Bytes(), []byte("txn_")) {
		t.Fatalf("private Stripe identifier leaked: %s", rr.Body.String())
	}
}
