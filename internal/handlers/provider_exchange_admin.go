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

const providerMaximumAdminLedgerCents int64 = models.ProviderMoneyMaximumCents

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
		if request.AmountCents < 1 || request.AmountCents > providerMaximumAdminLedgerCents || strings.ToLower(strings.TrimSpace(request.Currency)) != "usd" {
			providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "fund amount must be 1..100000000 cents in usd"})
			return
		}
		entry, err := models.FundProviderOffer(h.DB, request.OfferID, request.AmountCents, request.Currency, request.EvidenceReference)
		if err != nil {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		status := http.StatusCreated
		if entry.Replayed {
			status = http.StatusOK
		}
		providerWriteJSON(w, status, map[string]any{
			"budget_entry":      entry,
			"created":           !entry.Replayed,
			"idempotent_replay": entry.Replayed,
			"evidence_scope":    "operator-recorded external funding evidence; this endpoint does not move money",
		})
	case "adjust":
		if request.AmountCents == 0 || request.AmountCents < -providerMaximumAdminLedgerCents || request.AmountCents > providerMaximumAdminLedgerCents || strings.ToLower(strings.TrimSpace(request.Currency)) != "usd" {
			providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "adjustment must be non-zero, within +/-100000000 cents, and in usd"})
			return
		}
		entry, err := models.AdjustProviderOfferBudget(h.DB, request.OfferID, request.AmountCents, request.Currency, request.EvidenceReference)
		if err != nil {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		status := http.StatusCreated
		if entry.Replayed {
			status = http.StatusOK
		}
		providerWriteJSON(w, status, map[string]any{
			"budget_entry":      entry,
			"created":           !entry.Replayed,
			"idempotent_replay": entry.Replayed,
			"evidence_scope":    "operator-recorded adjustment; this endpoint does not move money",
		})
	case "activate":
		offer, err := models.ActivateProviderOffer(h.DB, request.OfferID, request.OperatorReference, request.EvidenceReference)
		if err != nil {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		providerWriteJSON(w, http.StatusOK, map[string]any{
			"offer":          offer,
			"evidence_scope": "operator-recorded prepaid funding or exact CPA terms; not independently audited by NHS",
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
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "action must be fund, adjust, activate, or pause"})
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
	proof, err := models.GetProviderExchangeProof(h.DB)
	if err != nil {
		status, message := providerExchangeStatus(err)
		providerWriteJSON(w, status, map[string]string{"error": message})
		return
	}
	providerWriteJSON(w, http.StatusOK, map[string]any{
		"proof": proof,
		"targets": map[string]int{
			"operator_recorded_provider_budgets":  3,
			"provider_reported_accepted_handoffs": 5,
			"provider_reported_activations":       2,
			"post_charge_provider_replenishments": 1,
		},
		"evidence_scope":        "Budgets and CPA terms are operator-recorded external evidence. Outcomes are provider-authenticated callbacks. Signed receipts prove NHS recorded those claims and budget effects; they do not independently audit the provider's business event.",
		"organic_rank_sold":     false,
		"raw_queries_sold":      false,
		"agent_identities_sold": false,
	})
}
