package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
)

func (h *ProviderExchangeHandler) ProviderPilotStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}
	key := h.requireProviderKey(w, r)
	if key == nil || !h.allowProviderRead(w, key, "pilot-status") {
		return
	}
	limit, ok := providerReadLimitParameter(r, 25)
	if !ok {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be an integer from 1 to 100"})
		return
	}
	status, err := models.GetProviderPilotStatus(h.DB, key, limit)
	if err != nil {
		code, message := providerExchangeStatus(err)
		providerWriteJSON(w, code, map[string]string{"error": message})
		return
	}
	providerWriteJSON(w, http.StatusOK, map[string]any{
		"pilot_status":   status,
		"evidence_scope": "Claim-key-scoped provider continuity only. Recent events appear only after an NHS-observed handoff. This response contains no credentials, attribution material, search receipts, controlled intent, queries, identities, contacts, network data, company hashes, or action URLs.",
	})
}

func (h *ProviderExchangeHandler) ProviderDemand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}
	key := h.requireProviderKey(w, r)
	if key == nil || !h.allowProviderRead(w, key, "demand") {
		return
	}
	days := 30
	if raw := strings.TrimSpace(r.URL.Query().Get("days")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 30 {
			providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "days must be an integer from 1 to 30"})
			return
		}
		days = parsed
	}
	demand, err := models.GetProviderDemandAnalyticsForClaim(h.DB, key.ProviderClaimID, days)
	if err != nil {
		code, message := providerExchangeStatus(err)
		providerWriteJSON(w, code, map[string]string{"error": message})
		return
	}
	providerWriteJSON(w, http.StatusOK, map[string]any{
		"demand":         demand,
		"evidence_scope": "Privacy-thresholded, claim-scoped aggregate demand only. Counts are receipts, not unique agents or principals; no raw query, identity, contact, network, or individual receipt is returned.",
	})
}

func (h *ProviderExchangeHandler) AdminProviderPilotQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}
	if !h.requireAdmin(w, r) {
		return
	}
	state := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("state")))
	if err := models.ValidateProviderPilotQueueState(state); err != nil {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "state must be all, review_required, pre_event_review_required, provider_review_required, offer_review_required, ticket_review_required, handoff_review_required, callback_review_required, pending_company, pending_terms, activation_review, expiring_terms, handoff_awaiting_callback, or recent_callback",
		})
		return
	}
	limit, ok := providerReadLimitParameter(r, 25)
	if !ok {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be an integer from 1 to 100"})
		return
	}
	queue, err := models.GetProviderPilotQueue(h.DB, state, limit)
	if err != nil {
		code, message := providerExchangeStatus(err)
		providerWriteJSON(w, code, map[string]string{"error": message})
		return
	}
	providerWriteJSON(w, http.StatusOK, map[string]any{
		"queue":          queue,
		"evidence_scope": "Read-only owner work queue. Queue rows identify evidence to review but do not verify a company, accept terms, activate an offer, report an outcome, or establish commercial proof.",
	})
}

func providerReadLimitParameter(r *http.Request, fallback int) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return fallback, true
	}
	parsed, err := strconv.Atoi(raw)
	return parsed, err == nil && parsed >= 1 && parsed <= models.ProviderPilotStatusMaximumItems
}

func (h *ProviderExchangeHandler) allowProviderRead(w http.ResponseWriter, key *models.ProviderAPIKey, surface string) bool {
	if h.providerReadLimit == nil {
		h.providerReadLimit = newMCPDiscoveryRateLimiter(240, time.Hour)
	}
	_, retry, ok := h.providerReadLimit.allow(
		fmt.Sprintf("provider-read:%d:%s", key.ID, surface), time.Now(),
	)
	if ok {
		return true
	}
	w.Header().Set("Retry-After", fmt.Sprintf("%d", max(1, int(retry.Seconds()))))
	providerWriteJSON(w, http.StatusTooManyRequests, map[string]string{"error": "provider read safety limit exceeded"})
	return false
}
