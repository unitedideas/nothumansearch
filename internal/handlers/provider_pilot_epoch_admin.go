package handlers

import (
	"net/http"
	"strings"

	"github.com/unitedideas/nothumansearch/internal/models"
)

type providerPilotEpochAdminRequest struct {
	Action               string `json:"action"`
	ProviderPilotEpochID string `json:"provider_pilot_epoch_id,omitempty"`
	ProviderClaimID      string `json:"provider_claim_id,omitempty"`
	DemandTopic          string `json:"demand_topic,omitempty"`
	CohortLimit          int    `json:"cohort_limit,omitempty"`
	ProviderTicketCap    int    `json:"provider_ticket_cap,omitempty"`
	TotalTicketCap       int    `json:"total_ticket_cap,omitempty"`
	OwnerReference       string `json:"owner_reference"`
	EvidenceReference    string `json:"evidence_reference"`
}

func (request providerPilotEpochAdminRequest) boundedReferences() bool {
	return validProviderEvidenceReference(strings.TrimSpace(request.OwnerReference)) &&
		validProviderEvidenceReference(strings.TrimSpace(request.EvidenceReference))
}

func providerPilotEpochMutationResponse(action string, value any) map[string]any {
	return map[string]any{
		"action":                   action,
		"provider_pilot":           value,
		"commercial_proof_created": false,
		"evidence_scope":           "Owner-authorized pilot configuration only. This response is not provider funding, a handoff, an activation outcome, a renewal, or 3/5/2/1 proof.",
	}
}

// AdminProviderPilotEpochAction is the sole owner mutation surface for the
// bounded Stage 2 epoch. Each action has an exact, mutually exclusive shape;
// no endpoint accepts a query, contact, principal, agent, or free-form intent.
func (h *ProviderExchangeHandler) AdminProviderPilotEpochAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	if !h.requireAdmin(w, r) {
		return
	}
	var request providerPilotEpochAdminRequest
	if err := decodeProviderJSON(w, r, &request); err != nil {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid provider pilot action request"})
		return
	}
	request.Action = strings.ToLower(strings.TrimSpace(request.Action))
	request.ProviderPilotEpochID = strings.ToLower(strings.TrimSpace(request.ProviderPilotEpochID))
	request.ProviderClaimID = strings.ToLower(strings.TrimSpace(request.ProviderClaimID))
	request.DemandTopic = strings.ToLower(strings.TrimSpace(request.DemandTopic))
	request.OwnerReference = strings.TrimSpace(request.OwnerReference)
	request.EvidenceReference = strings.TrimSpace(request.EvidenceReference)
	if !request.boundedReferences() {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "bounded non-secret owner_reference and evidence_reference required"})
		return
	}

	writeError := func(err error) {
		status, message := providerExchangeStatus(err)
		providerWriteJSON(w, status, map[string]string{"error": message})
	}
	switch request.Action {
	case "authorize":
		if request.ProviderPilotEpochID != "" || request.ProviderClaimID != "" {
			providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "authorize accepts demand_topic and caps, not pilot or claim IDs"})
			return
		}
		epoch, err := models.CreateProviderPilotEpoch(h.DB, models.ProviderPilotEpochInput{
			DemandTopic:       request.DemandTopic,
			CohortLimit:       request.CohortLimit,
			ProviderTicketCap: request.ProviderTicketCap,
			TotalTicketCap:    request.TotalTicketCap,
			OwnerReference:    request.OwnerReference,
			EvidenceReference: request.EvidenceReference,
		})
		if err != nil {
			writeError(err)
			return
		}
		providerWriteJSON(w, http.StatusCreated, providerPilotEpochMutationResponse(request.Action, epoch))
	case "enroll":
		if request.DemandTopic != "" || request.CohortLimit != 0 ||
			request.ProviderTicketCap != 0 || request.TotalTicketCap != 0 {
			providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "enroll accepts only exact pilot and claim IDs plus evidence references"})
			return
		}
		enrollment, err := models.EnrollProviderPilotCompany(h.DB, models.ProviderPilotEnrollmentInput{
			ProviderPilotEpochID: request.ProviderPilotEpochID,
			ProviderClaimID:      request.ProviderClaimID,
			OwnerReference:       request.OwnerReference,
			EvidenceReference:    request.EvidenceReference,
		})
		if err != nil {
			writeError(err)
			return
		}
		providerWriteJSON(w, http.StatusOK, providerPilotEpochMutationResponse(request.Action, enrollment))
	case "activate", "close":
		if request.ProviderClaimID != "" || request.DemandTopic != "" ||
			request.CohortLimit != 0 || request.ProviderTicketCap != 0 ||
			request.TotalTicketCap != 0 {
			providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": request.Action + " accepts only the exact pilot ID and evidence references"})
			return
		}
		input := models.ProviderPilotEpochActionInput{
			ProviderPilotEpochID: request.ProviderPilotEpochID,
			OwnerReference:       request.OwnerReference,
			EvidenceReference:    request.EvidenceReference,
		}
		var epoch *models.ProviderPilotEpoch
		var err error
		if request.Action == "activate" {
			epoch, err = models.ActivateProviderPilotEpoch(h.DB, input)
		} else {
			epoch, err = models.CloseProviderPilotEpoch(h.DB, input)
		}
		if err != nil {
			writeError(err)
			return
		}
		providerWriteJSON(w, http.StatusOK, providerPilotEpochMutationResponse(request.Action, epoch))
	default:
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "action must be authorize, enroll, activate, or close"})
	}
}

func (h *ProviderExchangeHandler) AdminProviderPilotEpochStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}
	if !h.requireAdmin(w, r) {
		return
	}
	pilotID := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("pilot_id")))
	if pilotID == "" {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "exact pilot_id required"})
		return
	}
	status, err := models.GetProviderPilotEpochStatus(h.DB, pilotID)
	if err != nil {
		code, message := providerExchangeStatus(err)
		providerWriteJSON(w, code, map[string]string{"error": message})
		return
	}
	providerWriteJSON(w, http.StatusOK, map[string]any{
		"provider_pilot": status,
		"evidence_scope": "Exact epoch status and bounded counts only; no query, contact, principal, agent, funding, outcome, or revenue claim.",
	})
}
