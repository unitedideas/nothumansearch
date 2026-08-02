package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
)

const actionInterestHourlyLimit = 120

type ActionInterestHandler struct {
	DB      *sql.DB
	BaseURL string
	limiter *mcpDiscoveryRateLimiter
}

type actionInterestRequest struct {
	SearchID                       string `json:"search_id"`
	Domain                         string `json:"domain"`
	ActionType                     string `json:"action_type"`
	CallerAttestsPrincipalInterest bool   `json:"caller_attests_principal_interest"`
	ConfirmationVersion            string `json:"confirmation_version"`
}

func NewActionInterestHandler(db *sql.DB, baseURL string) *ActionInterestHandler {
	return &ActionInterestHandler{
		DB:      db,
		BaseURL: strings.TrimRight(baseURL, "/"),
		limiter: newMCPDiscoveryRateLimiter(actionInterestHourlyLimit, time.Hour),
	}
}

func (h *ActionInterestHandler) consume(r *http.Request) (remaining int, retryAfter time.Duration, ok bool) {
	if h.limiter == nil {
		h.limiter = newMCPDiscoveryRateLimiter(actionInterestHourlyLimit, time.Hour)
	}
	return h.limiter.allow("action-interest:"+submitHashIP(r), time.Now())
}

func (h *ActionInterestHandler) gate(r *http.Request) (int, time.Duration, error) {
	if !requestOriginAllowed(r, h.BaseURL) {
		return actionInterestHourlyLimit, 0, errActionInterestCrossOrigin
	}
	remaining, retryAfter, ok := h.consume(r)
	if !ok {
		return remaining, retryAfter, errActionInterestRateLimited
	}
	return remaining, retryAfter, nil
}

func (h *ActionInterestHandler) recordAfterGate(r *http.Request, request actionInterestRequest, surface string) (*models.ActionInterestReceipt, error) {
	if demandRequestIsSynthetic(r) {
		return nil, models.ErrActionInterestUnavailable
	}
	receipt, err := models.RecordActionInterest(h.DB, models.ActionInterestInput{
		SearchID:                       request.SearchID,
		Domain:                         request.Domain,
		ActionType:                     request.ActionType,
		Surface:                        surface,
		CallerAttestsPrincipalInterest: request.CallerAttestsPrincipalInterest,
		ConfirmationVersion:            request.ConfirmationVersion,
	})
	return receipt, err
}

var (
	errActionInterestRateLimited = errors.New("action-interest safety limit exceeded")
	errActionInterestCrossOrigin = errors.New("cross-origin action-interest mutation rejected")
)

func actionInterestResponse(receipt *models.ActionInterestReceipt) map[string]any {
	return map[string]any{
		"receipt":                        receipt,
		"created":                        receipt != nil && !receipt.Replayed,
		"idempotent_replay":              receipt != nil && receipt.Replayed,
		"provider_contacted":             false,
		"row_level_shared_with_provider": false,
		"action_ticket_created":          false,
		"charge_created":                 false,
		"provider_or_principal_charged":  false,
		"commercial_proof":               false,
		"organic_rank_affected":          false,
		"rank_or_score_input":            false,
		"retention_days":                 models.ActionInterestRetentionDays,
		"evidence_scope":                 "Caller-attested principal interest recorded by NHS for aggregate Stage 1 demand only; no provider was contacted and no outcome was verified.",
	}
}

func actionInterestStatus(err error) (int, string) {
	switch {
	case err == nil:
		return http.StatusOK, ""
	case errors.Is(err, errActionInterestRateLimited):
		return http.StatusTooManyRequests, "action-interest safety limit exceeded; retry after the indicated interval"
	case errors.Is(err, errActionInterestCrossOrigin):
		return http.StatusForbidden, "cross-origin action-interest mutation rejected"
	case errors.Is(err, models.ErrInvalidActionInterest):
		return http.StatusBadRequest, "invalid action-interest request"
	case errors.Is(err, models.ErrActionInterestUnavailable):
		return http.StatusNotFound, "eligible organic search result unavailable"
	case errors.Is(err, models.ErrActionInterestConflict):
		return http.StatusConflict, "this search result already has a different recorded action interest"
	case errors.Is(err, models.ErrActionInterestStoreUnavailable):
		return http.StatusServiceUnavailable, "action-interest store unavailable"
	default:
		return http.StatusInternalServerError, "action-interest operation failed"
	}
}

// Record accepts controlled, caller-attested principal demand only. A search receipt
// is a bearer capability, so every response that consumes it is private and
// non-cacheable even when the request is invalid or rate limited.
func (h *ActionInterestHandler) Record(w http.ResponseWriter, r *http.Request) {
	protectReceiptBearingResponse(w)
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	remaining, retryAfter, err := h.gate(r)
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(actionInterestHourlyLimit))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(retryAfter).Unix(), 10))
	if err != nil {
		status, message := actionInterestStatus(err)
		if status == http.StatusForbidden {
			w.Header().Del("Access-Control-Allow-Origin")
		}
		if status == http.StatusTooManyRequests {
			w.Header().Set("Retry-After", strconv.Itoa(max(1, int(retryAfter.Seconds()))))
		}
		providerWriteJSON(w, status, map[string]string{"error": message})
		return
	}
	var request actionInterestRequest
	if err := decodeProviderJSON(w, r, &request); err != nil {
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid action-interest request"})
		return
	}

	receipt, err := h.recordAfterGate(r, request, "rest")
	if err != nil {
		status, message := actionInterestStatus(err)
		providerWriteJSON(w, status, map[string]string{"error": message})
		return
	}
	status := http.StatusCreated
	if receipt.Replayed {
		status = http.StatusOK
	}
	providerWriteJSON(w, status, actionInterestResponse(receipt))
}

func (h *ActionInterestHandler) Stage1DemandProof(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}
	if !requireAdminAPIKey(w, r) {
		return
	}
	days := models.ActionInterestRetentionDays
	if raw := strings.TrimSpace(r.URL.Query().Get("days")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > models.ActionInterestRetentionDays {
			providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("days must be 1..%d", models.ActionInterestRetentionDays)})
			return
		}
		days = parsed
	}
	proof, err := models.GetStage1DemandProof(h.DB, days)
	if err != nil {
		providerWriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "stage 1 demand query failed"})
		return
	}
	providerWriteJSON(w, http.StatusOK, map[string]any{
		"stage1_demand":  proof,
		"evidence_scope": "Demand receipts are not unique agents, principals, provider-accepted handoffs, activations, revenue, or commercial proof.",
	})
}
