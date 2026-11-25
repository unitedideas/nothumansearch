package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/unitedideas/nothumansearch/internal/models"
)

const providerProofManifestEvidenceScope = "One owner-issued, privacy-redacted HMAC-signed aggregate for an exact closed Not Human Search pilot. Issuance requires authenticated outcome integrity, the 3/5/2/1 threshold, and complete chronological owner review of every qualifying provider, offer, ticket, handoff, and callback. Its signature is verifiable only by NHS; it proves what NHS recorded, not independent provider truth or cash collection, and it does not publish automatically."

type providerProofManifestRequest struct {
	ProviderPilotEpochID   string `json:"provider_pilot_epoch_id"`
	ExpectedSnapshotSHA256 string `json:"expected_snapshot_sha256"`
	OwnerReference         string `json:"owner_reference"`
	EvidenceReference      string `json:"evidence_reference"`
}

// AdminProviderProofManifest previews the exact aggregate with GET and issues
// its append-only signed form with an owner-authorized POST. GET never signs or
// writes. An issued manifest remains private until a separately authorized
// publication action occurs.
func (h *ProviderExchangeHandler) AdminProviderProofManifest(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		pilotID := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("pilot_id")))
		if pilotID == "" {
			providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "exact pilot_id required"})
			return
		}
		record, err := models.GetProviderCommercialProofManifest(h.DB, pilotID, h.Signer)
		if err == nil {
			providerWriteJSON(w, http.StatusOK, map[string]any{
				"manifest":                 record,
				"issued":                   true,
				"commercial_proof_created": true,
				"publicly_released":        false,
				"independently_verifiable": false,
				"evidence_scope":           providerProofManifestEvidenceScope,
			})
			return
		}
		if !errors.Is(err, sql.ErrNoRows) {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		candidate, err := models.GetProviderCommercialProofManifestCandidate(
			h.DB, pilotID, h.Signer,
		)
		if err != nil {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		providerWriteJSON(w, http.StatusOK, map[string]any{
			"manifest_candidate":       candidate,
			"issued":                   false,
			"commercial_proof_created": false,
			"publicly_released":        false,
			"independently_verifiable": false,
			"evidence_scope":           providerProofManifestEvidenceScope,
		})

	case http.MethodPost:
		var request providerProofManifestRequest
		if err := decodeProviderJSON(w, r, &request); err != nil {
			providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid provider proof manifest request"})
			return
		}
		record, created, err := models.IssueProviderCommercialProofManifest(
			h.DB,
			models.ProviderCommercialProofManifestInput{
				ProviderPilotEpochID:   request.ProviderPilotEpochID,
				ExpectedSnapshotSHA256: request.ExpectedSnapshotSHA256,
				OwnerReference:         request.OwnerReference,
				EvidenceReference:      request.EvidenceReference,
			},
			h.Signer,
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
			"manifest":                 record,
			"created":                  created,
			"idempotent_replay":        !created,
			"commercial_proof_created": true,
			"publicly_released":        false,
			"independently_verifiable": false,
			"evidence_scope":           providerProofManifestEvidenceScope,
		})

	default:
		w.Header().Set("Allow", "GET, POST")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET or POST required"})
	}
}
