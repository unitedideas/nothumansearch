package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
	"github.com/unitedideas/nothumansearch/internal/providerexchange"
)

func TestVerifyOutcomeReceiptLabelsProviderEvidenceHonestly(t *testing.T) {
	t.Parallel()
	signer, err := providerexchange.NewSigner("test-admin-key-0123456789abcdef012345")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	canonical, signature, err := signer.SignOutcomeReceipt(providerexchange.OutcomeReceipt{
		Version:            providerexchange.OutcomeReceiptVersion,
		ReceiptID:          "7f19ca8e-d61d-47e2-91dd-fecd9f711234",
		TicketID:           "3a59ca8e-d61d-47e2-91dd-fecd9f711234",
		OfferID:            "4b69ca8e-d61d-47e2-91dd-fecd9f711234",
		NHSEventID:         "5c79ca8e-d61d-47e2-91dd-fecd9f711234",
		Outcome:            providerexchange.OutcomeActivated,
		ProviderReportedAt: now.Unix(),
		RecordedAt:         now.Unix(),
		ExpiresAt:          now.Add(time.Hour).Unix(),
		ChargedMinor:       2500,
		Currency:           "usd",
		ChargeStatus:       providerexchange.ChargeStatusCharged,
	})
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"signed_receipt":` + quoteJSON(canonical) + `,"signature":` + quoteJSON(signature) + `}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/action-receipts/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler := &ProviderExchangeHandler{
		Signer: signer,
		receiptState: func(_ *sql.DB, receiptID, ticketID string) (*models.PublicOutcomeReceiptState, error) {
			return &models.PublicOutcomeReceiptState{
				ReceiptID:                   receiptID,
				ActionTicketID:              ticketID,
				ReceiptOutcome:              "activated",
				CurrentTicketStatus:         "invalid",
				OriginalChargeCredited:      true,
				SupersededByLaterState:      true,
				NetCommercialEffectCents:    0,
				NetCommercialEffectCurrency: "usd",
			}, nil
		},
	}
	handler.VerifyOutcomeReceipt(rr, req)
	if rr.Code != http.StatusOK || !bytes.Contains(rr.Body.Bytes(), []byte(`"signature_valid":true`)) ||
		!bytes.Contains(rr.Body.Bytes(), []byte(`"within_validity_window":true`)) ||
		!bytes.Contains(rr.Body.Bytes(), []byte(`"original_charge_credited":true`)) ||
		!bytes.Contains(rr.Body.Bytes(), []byte(`"superseded_by_later_state":true`)) ||
		!bytes.Contains(rr.Body.Bytes(), []byte("not independently audited")) {
		t.Fatalf("verify response status=%d body=%s", rr.Code, rr.Body.String())
	}
	for name, want := range map[string]string{
		"Cache-Control":   "private, no-store",
		"Pragma":          "no-cache",
		"Referrer-Policy": "no-referrer",
	} {
		if got := rr.Header().Get(name); got != want {
			t.Fatalf("receipt verification %s = %q, want %q", name, got, want)
		}
	}
}

func TestVerifyOutcomeReceiptKeepsHistoricalSignatureSeparateFromFreshness(t *testing.T) {
	t.Parallel()
	signer, err := providerexchange.NewSigner("test-signing-key-0123456789abcdef0123")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	canonical, signature, err := signer.SignOutcomeReceipt(providerexchange.OutcomeReceipt{
		Version:            providerexchange.OutcomeReceiptVersion,
		ReceiptID:          "7f19ca8e-d61d-47e2-91dd-fecd9f711234",
		TicketID:           "3a59ca8e-d61d-47e2-91dd-fecd9f711234",
		OfferID:            "4b69ca8e-d61d-47e2-91dd-fecd9f711234",
		NHSEventID:         "5c79ca8e-d61d-47e2-91dd-fecd9f711234",
		Outcome:            providerexchange.OutcomeAccepted,
		ProviderReportedAt: now.Add(-2 * time.Hour).Unix(),
		RecordedAt:         now.Add(-2 * time.Hour).Unix(),
		ExpiresAt:          now.Add(-time.Hour).Unix(),
		ChargedMinor:       0,
		Currency:           "usd",
		ChargeStatus:       providerexchange.ChargeStatusNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"signed_receipt":` + quoteJSON(canonical) + `,"signature":` + quoteJSON(signature) + `}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/action-receipts/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	(&ProviderExchangeHandler{Signer: signer}).VerifyOutcomeReceipt(rr, req)
	if rr.Code != http.StatusOK || !bytes.Contains(rr.Body.Bytes(), []byte(`"signature_valid":true`)) ||
		!bytes.Contains(rr.Body.Bytes(), []byte(`"within_validity_window":false`)) ||
		!bytes.Contains(rr.Body.Bytes(), []byte(`"time_status":"expired"`)) {
		t.Fatalf("verify response status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestExpiredAttributionOnlyAuthenticatesChargeResolution(t *testing.T) {
	t.Parallel()
	signer, err := providerexchange.NewSigner("test-signing-key-0123456789abcdef0123")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	claims, err := providerexchange.NewAttributionClaims(
		"3a59ca8e-d61d-47e2-91dd-fecd9f711234",
		"4b69ca8e-d61d-47e2-91dd-fecd9f711234",
		now.Add(-time.Hour),
		now.Add(-time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	token, err := signer.SignAttribution(claims)
	if err != nil {
		t.Fatal(err)
	}
	verified, late, err := verifyProviderOutcomeAttribution(signer, token, "invalid", now)
	if err != nil || !late || verified.TicketID != claims.TicketID {
		t.Fatalf("late invalid verification = %#v, late=%t, err=%v", verified, late, err)
	}
	for _, outcome := range []string{"accepted", "activated", "converted", "rejected"} {
		if _, late, err := verifyProviderOutcomeAttribution(signer, token, outcome, now); !errors.Is(err, providerexchange.ErrExpired) || late {
			t.Fatalf("expired %s outcome err=%v late=%t", outcome, err, late)
		}
	}
}

func quoteJSON(value string) string {
	var buffer bytes.Buffer
	_ = json.NewEncoder(&buffer).Encode(value)
	return strings.TrimSpace(buffer.String())
}
