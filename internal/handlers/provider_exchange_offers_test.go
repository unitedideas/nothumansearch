package handlers

import (
	"testing"

	"github.com/unitedideas/nothumansearch/internal/models"
)

func TestProviderOfferInputCannotSelfAttestCommercialEvidence(t *testing.T) {
	t.Parallel()
	h := &ProviderExchangeHandler{}
	zero := int64(0)
	limit := int64(10_000)
	days := 30
	input, err := h.providerOfferInput(providerOfferRequest{
		Name:                  "Start a supported trial",
		Summary:               "A provider-supported trial action.",
		ActionType:            "trial",
		ActionURL:             "https://example.com/trial",
		ChargeEvent:           "activated",
		BountyCents:           2500,
		Currency:              "usd",
		PrincipalPriceMode:    "free",
		PrincipalPriceCents:   &zero,
		PrincipalCurrency:     "usd",
		BillingMode:           models.ProviderPilotBillingMode,
		TermsCreditLimitCents: &limit,
		TermsPeriodDays:       &days,
	}, "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if input.TermsEvidenceReference != "" {
		t.Fatalf("provider supplied evidence reference = %q, want empty admin boundary", input.TermsEvidenceReference)
	}
	if input.BillingMode != models.ProviderPilotBillingMode {
		t.Fatalf("pilot billing mode = %q", input.BillingMode)
	}
}

func TestProviderOfferInputRejectsUnlaunchedPrepaidMode(t *testing.T) {
	t.Parallel()
	zero := int64(0)
	_, err := (&ProviderExchangeHandler{}).providerOfferInput(providerOfferRequest{
		Name: "Prepaid is not launched", Summary: "This must stay out of the bounded pilot.",
		ActionType: "trial", ActionURL: "https://example.com/trial",
		ChargeEvent: "activated", BountyCents: 2500, Currency: "usd",
		PrincipalPriceMode: "free", PrincipalPriceCents: &zero,
		PrincipalCurrency: "usd", BillingMode: "prepaid",
	}, "example.com")
	if err == nil {
		t.Fatal("prepaid provider offer entered the terms-only pilot")
	}
}
