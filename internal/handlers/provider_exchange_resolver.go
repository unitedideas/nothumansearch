package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
	"github.com/unitedideas/nothumansearch/internal/providerexchange"
)

type providerControlledIntentResolveRequest struct {
	AttributionToken string `json:"attribution_token"`
}

// ResolveProviderControlledIntent is a provider-key-authenticated read after a
// separately consented NHS-observed handoff. It returns only the controlled
// ticket bundle and creates no analytics, outcome, charge, receipt, or proof.
func (h *ProviderExchangeHandler) ResolveProviderControlledIntent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	if h.Signer == nil {
		providerWriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "controlled intent resolution is not configured"})
		return
	}
	if !h.allowControlledIntentAuthentication(w, r) {
		return
	}
	key := h.requireProviderKey(w, r)
	if key == nil {
		return
	}
	if !h.allowControlledIntentResolution(w, r, key) {
		return
	}
	var request providerControlledIntentResolveRequest
	if err := decodeProviderJSON(w, r, &request); err != nil {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid controlled intent resolution request; only attribution_token is accepted",
		})
		return
	}
	request.AttributionToken = strings.TrimSpace(request.AttributionToken)
	if request.AttributionToken == "" {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "valid NHS attribution token required"})
		return
	}
	claims, err := h.Signer.VerifyAttribution(request.AttributionToken, time.Now().UTC())
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, providerexchange.ErrExpired) {
			status = http.StatusGone
		}
		providerWriteJSON(w, status, map[string]string{"error": "invalid or expired NHS attribution token"})
		return
	}
	resolve := h.resolveControlledIntent
	if resolve == nil {
		resolve = models.ResolveProviderControlledIntent
	}
	resolution, err := resolve(
		h.DB, key, claims.TicketID, claims.OfferID, request.AttributionToken,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			providerWriteJSON(w, http.StatusNotFound, map[string]string{"error": "consented controlled intent unavailable"})
		case errors.Is(err, models.ErrInvalidProviderExchange):
			providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid controlled intent resolution request"})
		default:
			providerWriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "controlled intent resolution unavailable"})
		}
		return
	}
	providerWriteJSON(w, http.StatusOK, resolution)
}

func (h *ProviderExchangeHandler) allowControlledIntentAuthentication(w http.ResponseWriter, r *http.Request) bool {
	if h == nil {
		providerWriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "controlled intent authentication safety limit unavailable"})
		return false
	}
	if h.resolverAuthLimit == nil {
		h.resolverAuthLimit = newMCPDiscoveryRateLimiter(120, time.Hour)
	}
	bucket := "provider-controlled-intent-auth:" + submitHashIP(r)
	_, retry, ok := h.resolverAuthLimit.allow(bucket, time.Now())
	if ok {
		return true
	}
	w.Header().Set("Retry-After", fmt.Sprintf("%d", max(1, int(retry.Seconds()))))
	providerWriteJSON(w, http.StatusTooManyRequests, map[string]string{"error": "controlled intent authentication safety limit exceeded"})
	return false
}

func (h *ProviderExchangeHandler) allowControlledIntentResolution(
	w http.ResponseWriter, r *http.Request, key *models.ProviderAPIKey,
) bool {
	if h == nil || key == nil {
		providerWriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "controlled intent resolution safety limit unavailable"})
		return false
	}
	if h.resolverLimit == nil {
		h.resolverLimit = newMCPDiscoveryRateLimiter(1000, time.Hour)
	}
	bucket := fmt.Sprintf("provider-controlled-intent:%d:%s", key.ID, submitHashIP(r))
	_, retry, ok := h.resolverLimit.allow(bucket, time.Now())
	if ok {
		return true
	}
	w.Header().Set("Retry-After", fmt.Sprintf("%d", max(1, int(retry.Seconds()))))
	providerWriteJSON(w, http.StatusTooManyRequests, map[string]string{"error": "controlled intent resolution safety limit exceeded"})
	return false
}
