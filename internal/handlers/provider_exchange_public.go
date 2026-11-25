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
	ID                                   string                     `json:"id"`
	OfferVersion                         int                        `json:"offer_version"`
	ProviderDomain                       string                     `json:"provider_domain"`
	OrganicPosition                      int                        `json:"organic_position"`
	Name                                 string                     `json:"name"`
	Summary                              string                     `json:"summary"`
	ActionType                           string                     `json:"action_type"`
	Disclosure                           string                     `json:"disclosure"`
	OrganicRankPaid                      bool                       `json:"organic_rank_paid"`
	PrincipalPrice                       publicPrincipalPrice       `json:"principal_price"`
	NHSCompensation                      publicProviderCompensation `json:"nhs_compensation"`
	CommercialTermsContractVersion       string                     `json:"commercial_terms_contract_version"`
	CommercialTermsSHA256                string                     `json:"commercial_terms_sha256"`
	CreditRule                           string                     `json:"credit_rule"`
	ResponseExpectation                  string                     `json:"response_expectation"`
	TermsPeriodAnchorRule                string                     `json:"terms_period_anchor_rule"`
	ProviderAcknowledgesMerchantOfRecord bool                       `json:"provider_acknowledges_merchant_of_record"`
	PrepareAction                        string                     `json:"prepare_action_endpoint"`
}

// publicProviderOfferView deliberately omits the provider's action URL,
// internal evidence reference, billing mode, and balance. Agents do not receive
// the attributed action URL until the consented ticket crosses the NHS-observed
// handoff. The canonical provider URL remains free in the organic result.
func publicProviderOfferView(offer models.ProviderOffer, baseURL string) publicProviderOffer {
	return publicProviderOffer{
		ID:              offer.ID,
		OfferVersion:    offer.Version,
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
		CommercialTermsContractVersion:       offer.CommercialTermsContractVersion,
		CommercialTermsSHA256:                offer.CommercialTermsSHA256,
		CreditRule:                           offer.CreditRule,
		ResponseExpectation:                  offer.ResponseExpectation,
		TermsPeriodAnchorRule:                offer.TermsPeriodAnchorRule,
		ProviderAcknowledgesMerchantOfRecord: offer.ProviderAcknowledgesMerchantOfRecord,
		PrepareAction:                        baseURL + "/api/v1/action-tickets",
	}
}

func publicProviderOfferModelView(offer models.PublicProviderOffer, baseURL string) publicProviderOffer {
	return publicProviderOffer{
		ID:              offer.OfferID,
		OfferVersion:    offer.OfferVersion,
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
		CommercialTermsContractVersion:       offer.CommercialTermsContractVersion,
		CommercialTermsSHA256:                offer.CommercialTermsSHA256,
		CreditRule:                           offer.CreditRule,
		ResponseExpectation:                  offer.ResponseExpectation,
		TermsPeriodAnchorRule:                offer.TermsPeriodAnchorRule,
		ProviderAcknowledgesMerchantOfRecord: offer.ProviderAcknowledgesMerchantOfRecord,
		PrepareAction:                        baseURL + "/api/v1/action-tickets",
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
