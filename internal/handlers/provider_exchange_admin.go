package handlers

import (
	"net/http"
	"strings"

	"github.com/unitedideas/nothumansearch/internal/models"
)

type providerAdminActionRequest struct {
	Action            string `json:"action"`
	OfferID           string `json:"offer_id"`
	AmountCents       int64  `json:"amount_cents,omitempty"`
	Currency          string `json:"currency,omitempty"`
	OperatorReference string `json:"operator_reference"`
	EvidenceReference string `json:"evidence_reference"`
}

func (h *ProviderExchangeHandler) AdminOfferAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	if !h.requireAdmin(w, r) {
		return
	}
	var request providerAdminActionRequest
	if err := decodeProviderJSON(w, r, &request); err != nil {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid admin action request"})
		return
	}
	request.Action = strings.ToLower(strings.TrimSpace(request.Action))
	request.OperatorReference = strings.TrimSpace(request.OperatorReference)
	request.EvidenceReference = strings.TrimSpace(request.EvidenceReference)
	if !validProviderEvidenceReference(request.OperatorReference) || !validProviderEvidenceReference(request.EvidenceReference) {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "bounded non-secret operator_reference and evidence_reference required"})
		return
	}
	switch request.Action {
	case "fund":
		providerWriteJSON(w, http.StatusConflict, map[string]string{
			"error": "prepaid and legacy funding writes are disabled for the terms-only pilot",
		})
	case "adjust":
		providerWriteJSON(w, http.StatusConflict, map[string]string{
			"error": "prepaid and legacy funding writes are disabled for the terms-only pilot",
		})
	case "activate":
		offer, err := models.ActivateProviderOffer(h.DB, request.OfferID, request.OperatorReference, request.EvidenceReference)
		if err != nil {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		providerWriteJSON(w, http.StatusOK, map[string]any{
			"offer":                    offer,
			"commercial_proof_created": false,
			"evidence_scope":           "Operational activation only; separate provider-authenticated and owner-verified commercial evidence is required and operator references alone cannot count as proof.",
		})
	case "pause":
		offer, err := models.AdminPauseProviderOffer(h.DB, request.OfferID, request.OperatorReference, request.EvidenceReference)
		if err != nil {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		providerWriteJSON(w, http.StatusOK, map[string]any{
			"offer":              offer,
			"paused":             true,
			"evidence_reference": request.EvidenceReference,
		})
	default:
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "action must be activate or pause; prepaid and legacy funding writes are disabled"})
	}
}

func (h *ProviderExchangeHandler) AdminProof(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}
	if !h.requireAdmin(w, r) {
		return
	}
	pilotID := strings.TrimSpace(r.URL.Query().Get("pilot_id"))
	if pilotID == "" {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "exact pilot_id required"})
		return
	}
	proof, err := models.GetProviderExchangeProof(h.DB, pilotID, h.Signer)
	if err != nil {
		status, message := providerExchangeStatus(err)
		providerWriteJSON(w, status, map[string]string{"error": message})
		return
	}
	providerWriteJSON(w, http.StatusOK, map[string]any{
		"proof": proof,
		"targets": map[string]int{
			"verified_provider_companies":             3,
			"verified_provider_accepted_handoffs":     5,
			"verified_provider_confirmed_activations": 2,
			"verified_provider_renewals":              1,
		},
		"evidence_scope":             "pilot_thresholds_met is scoped to the exact requested pilot_id and uses only its enrolled verified companies, epoch-bound offers and tickets, observed handoffs, provider-authenticated outcomes, owner-verified funding or exact terms, exact paid settlement aggregates, and nonreversed renewal evidence.",
		"operational_progress_scope": "operator_recorded and provider_reported legacy fields remain diagnostic observations and cannot satisfy pilot_thresholds_met.",
		"organic_rank_sold":          false,
		"raw_queries_sold":           false,
		"raw_prompts_sold":           false,
		"agent_identities_sold":      false,
		"principal_identities_sold":  false,
	})
}
