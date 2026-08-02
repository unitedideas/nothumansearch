package handlers

import "github.com/unitedideas/nothumansearch/internal/models"

type publicProviderCompensation struct {
	Event       string `json:"event"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

type publicPrincipalPrice struct {
	Mode        string `json:"mode"`
	AmountMinor *int64 `json:"amount_minor,omitempty"`
	Currency    string `json:"currency,omitempty"`
}

type publicProviderOffer struct {
	ID              string                     `json:"id"`
	ProviderDomain  string                     `json:"provider_domain"`
	OrganicPosition int                        `json:"organic_position"`
	Name            string                     `json:"name"`
	Summary         string                     `json:"summary"`
	ActionType      string                     `json:"action_type"`
	Disclosure      string                     `json:"disclosure"`
	OrganicRankPaid bool                       `json:"organic_rank_paid"`
	PrincipalPrice  publicPrincipalPrice       `json:"principal_price"`
	NHSCompensation publicProviderCompensation `json:"nhs_compensation"`
	PrepareAction   string                     `json:"prepare_action_endpoint"`
}

// publicProviderOfferView deliberately omits the provider's action URL,
// internal evidence reference, billing mode, and balance. Agents receive the
// attributed action URL only after creating a consented ticket. The canonical
// provider URL remains available for free in the adjacent organic result.
func publicProviderOfferView(offer models.ProviderOffer, baseURL string) publicProviderOffer {
	return publicProviderOffer{
		ID:              offer.ID,
		ProviderDomain:  offer.Domain,
		OrganicPosition: offer.OrganicPosition,
		Name:            offer.OfferName,
		Summary:         offer.OfferSummary,
		ActionType:      offer.ActionType,
		Disclosure:      offer.DisclosureLabel,
		OrganicRankPaid: false,
		PrincipalPrice: publicPrincipalPrice{
			Mode:        offer.PrincipalPriceMode,
			AmountMinor: offer.PrincipalPriceCents,
			Currency:    offer.PrincipalCurrency,
		},
		NHSCompensation: publicProviderCompensation{
			Event:       offer.ChargeEvent,
			AmountMinor: offer.BountyCents,
			Currency:    offer.Currency,
		},
		PrepareAction: baseURL + "/api/v1/action-tickets",
	}
}

func publicProviderOfferModelView(offer models.PublicProviderOffer, baseURL string) publicProviderOffer {
	return publicProviderOffer{
		ID:              offer.OfferID,
		ProviderDomain:  offer.Domain,
		OrganicPosition: offer.OrganicPosition,
		Name:            offer.OfferName,
		Summary:         offer.OfferSummary,
		ActionType:      offer.ActionType,
		Disclosure:      offer.DisclosureLabel,
		OrganicRankPaid: false,
		PrincipalPrice: publicPrincipalPrice{
			Mode:        offer.PrincipalPriceMode,
			AmountMinor: offer.PrincipalPriceCents,
			Currency:    offer.PrincipalCurrency,
		},
		NHSCompensation: publicProviderCompensation{
			Event:       offer.ChargeEvent,
			AmountMinor: offer.ProviderFundedBountyCents,
			Currency:    offer.ProviderFundedCurrency,
		},
		PrepareAction: baseURL + "/api/v1/action-tickets",
	}
}

func publicProviderOfferViews(offers []models.ProviderOffer, baseURL string) []publicProviderOffer {
	views := make([]publicProviderOffer, 0, len(offers))
	for _, offer := range offers {
		views = append(views, publicProviderOfferView(offer, baseURL))
	}
	return views
}

func publicProviderOfferModelViews(offers []models.PublicProviderOffer, baseURL string) []publicProviderOffer {
	views := make([]publicProviderOffer, 0, len(offers))
	for _, offer := range offers {
		views = append(views, publicProviderOfferModelView(offer, baseURL))
	}
	return views
}
