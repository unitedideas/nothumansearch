package handlers

import (
	"net/http"
	"strings"

	"github.com/unitedideas/nothumansearch/internal/models"
)

type providerPilotReviewRequest struct {
	ProviderPilotEpochID   string `json:"provider_pilot_epoch_id"`
	ReviewType             string `json:"review_type"`
	SubjectID              string `json:"subject_id"`
	ExpectedSnapshotSHA256 string `json:"expected_snapshot_sha256"`
	OwnerReference         string `json:"owner_reference"`
	EvidenceReference      string `json:"evidence_reference"`
}

const providerPilotReviewEvidenceScope = "Owner-only review of one exact privacy-bounded pilot snapshot. The candidate and receipt contain no search receipt, query, bearer, token hash, company-deduplication hash, principal or agent identity/contact/network metadata, raw signed-receipt body/signature, or free-form intent. Recording a review does not create a provider acceptance, handoff, outcome, renewal, revenue, or 3/5/2/1 proof."

// AdminProviderPilotReview previews one exact review candidate with GET and
// records its immutable hash-bound owner review with POST. The caller must echo
// the candidate digest so a changed subject fails closed between inspection and
// authorization.
func (h *ProviderExchangeHandler) AdminProviderPilotReview(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		pilotID := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("pilot_id")))
		reviewType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("review_type")))
		subjectID := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("subject_id")))
		candidate, err := models.GetProviderPilotReviewCandidate(
			h.DB, pilotID, reviewType, subjectID,
		)
		if err != nil {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		providerWriteJSON(w, http.StatusOK, map[string]any{
			"review_candidate": candidate,
			"evidence_scope":   providerPilotReviewEvidenceScope,
		})

	case http.MethodPost:
		var request providerPilotReviewRequest
		if err := decodeProviderJSON(w, r, &request); err != nil {
			providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid provider pilot review request"})
			return
		}
		event, created, err := models.RecordProviderPilotReview(
			h.DB,
			models.ProviderPilotReviewInput{
				ProviderPilotEpochID:   request.ProviderPilotEpochID,
				ReviewType:             request.ReviewType,
				SubjectID:              request.SubjectID,
				ExpectedSnapshotSHA256: request.ExpectedSnapshotSHA256,
				OwnerReference:         request.OwnerReference,
				EvidenceReference:      request.EvidenceReference,
			},
		)
		if err != nil {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		providerWriteJSON(w, status, map[string]any{
			"review":                   event,
			"created":                  created,
			"idempotent_replay":        !created,
			"commercial_proof_created": false,
			"evidence_scope":           providerPilotReviewEvidenceScope,
		})

	default:
		w.Header().Set("Allow", "GET, POST")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET or POST required"})
	}
}
