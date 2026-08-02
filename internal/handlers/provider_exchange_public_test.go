package handlers

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/unitedideas/nothumansearch/internal/models"
)

func TestPublicProviderOfferSeparatesPaidActionFromOrganicRank(t *testing.T) {
	t.Parallel()
	price := int64(9900)
	view := publicProviderOfferView(models.ProviderOffer{
		ID:                     "offer-id",
		Domain:                 "example.com",
		OrganicPosition:        3,
		OfferName:              "Start the API trial",
		OfferSummary:           "A provider-supported trial handoff.",
		ActionType:             "trial",
		ActionURL:              "https://example.com/private/action?internal=value",
		DisclosureLabel:        models.ProviderDisclosureLabel,
		ChargeEvent:            "activated",
		BountyCents:            2500,
		Currency:               "usd",
		PrincipalPriceMode:     "fixed",
		PrincipalPriceCents:    &price,
		PrincipalCurrency:      "usd",
		BillingMode:            "prepaid",
		TermsEvidenceReference: "contract:provider-1",
	}, "https://nothumansearch.ai")

	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	body := string(encoded)
	for _, want := range []string{
		`"organic_position":3`,
		`"organic_rank_paid":false`,
		`"disclosure":"Provider-funded action"`,
		`"event":"activated"`,
		`"amount_minor":2500`,
		`"prepare_action_endpoint":"https://nothumansearch.ai/api/v1/action-tickets"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("public offer %s missing %s", body, want)
		}
	}
	for _, forbidden := range []string{"private/action", "internal=value", "contract:provider-1", `"billing_mode"`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("public offer leaked internal field %q: %s", forbidden, body)
		}
	}
}
