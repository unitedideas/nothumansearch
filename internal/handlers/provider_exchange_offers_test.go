package handlers

import (
	"testing"
)

func TestProviderOfferInputCannotSelfAttestCommercialEvidence(t *testing.T) {
	t.Parallel()
	h := &ProviderExchangeHandler{}
	zero := int64(0)
	input, err := h.providerOfferInput(providerOfferRequest{
		Name:                "Start a supported trial",
		Summary:             "A provider-supported trial action.",
		ActionType:          "trial",
		ActionURL:           "https://example.com/trial",
		ChargeEvent:         "activated",
		BountyCents:         2500,
		Currency:            "usd",
		PrincipalPriceMode:  "free",
		PrincipalPriceCents: &zero,
		PrincipalCurrency:   "usd",
		BillingMode:         "prepaid",
	}, "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if input.TermsEvidenceReference != "" {
		t.Fatalf("provider supplied evidence reference = %q, want empty admin boundary", input.TermsEvidenceReference)
	}
}
