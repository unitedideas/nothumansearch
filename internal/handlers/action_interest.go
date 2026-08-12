package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
)

const actionInterestInvocationCondition = "Use only when the caller can truthfully attest that its principal currently wants one listed action with one eligible domain; do not infer interest from the search, result order, selection, or provider availability."

var actionInterestArgumentNames = []string{
	"search_id", "domain", "action_type",
	"caller_attests_principal_interest", "confirmation_version",
}

// actionInterestCallContract separates the fixed receipt/consent protocol from
// the two choices that must come from explicit current-principal intent. It is
// deliberately not an executable tool action: an agent still has to select one
// returned domain and one action type under the invocation condition.
func actionInterestCallContract(searchID string, domains []string, available bool) map[string]any {
	fixedArguments := map[string]any{}
	eligibleDomains := []string{}
	actionTypes := []string{}
	if available && strings.TrimSpace(searchID) != "" && len(domains) > 0 {
		fixedArguments = map[string]any{
			"search_id":                         searchID,
			"caller_attests_principal_interest": true,
			"confirmation_version":              models.ActionInterestConfirmationV1,
		}
		eligibleDomains = append(eligibleDomains, domains...)
		actionTypes = models.ActionInterestTypes()
	} else {
		available = false
	}
	return map[string]any{
		"available": available,
		"tool":      "record_action_interest",
		"fixed_arguments_if_invocation_condition_met":       fixedArguments,
		"domain_must_be_one_of":                             eligibleDomains,
		"action_type_must_be_one_of":                        actionTypes,
		"arguments_must_contain_only":                       append([]string(nil), actionInterestArgumentNames...),
		"invoke_only_if":                                    actionInterestInvocationCondition,
		"executable_without_explicit_principal_intent":      false,
		"query_prompt_contact_identity_fields_are_accepted": false,
	}
}

// publicActionInterestOpportunity keeps the optional next step machine-readable
// without converting discovery or a result selection into intent. Every domain
// returned here was already present in the public organic result set.
func publicActionInterestOpportunity(baseURL, searchID string, sites []models.Site, available bool) map[string]any {
	domains := []string{}
	if available {
		seen := make(map[string]struct{}, len(sites))
		for _, site := range sites {
			domain := strings.ToLower(strings.TrimSpace(site.Domain))
			if domain == "" {
				continue
			}
			if _, exists := seen[domain]; exists {
				continue
			}
			seen[domain] = struct{}{}
			domains = append(domains, domain)
		}
	}
	if len(domains) == 0 {
		available = false
		searchID = ""
	}
	return map[string]any{
		"available":                   available,
		"search_id":                   searchID,
		"eligible_domains":            domains,
		"action_types":                models.ActionInterestTypes(),
		"caller_attestation_required": true,
		"confirmation_version":        models.ActionInterestConfirmationV1,
		"confirmation_url":            strings.TrimRight(baseURL, "/") + "/privacy#action-interest-v1",
		"invocation_condition":        actionInterestInvocationCondition,
		"call_contract":               actionInterestCallContract(searchID, domains, available),
		"provider_contacted":          false,
		"commercial_proof":            false,
		"organic_rank_affected":       false,
	}
}

// selectedActionInterestOpportunity is returned only after NHS has just
// recorded an exact receipt-to-organic-result selection. It moves the truthful
// optional next step to the point where an agent has inspected one result,
// without treating that inspection as intent. The caller still has to make the
// separate current-principal attestation in record_action_interest.
func selectedActionInterestOpportunity(baseURL, searchID string, site *models.Site) map[string]any {
	if site == nil {
		return publicActionInterestOpportunity(baseURL, "", nil, false)
	}
	return publicActionInterestOpportunity(baseURL, searchID, []models.Site{*site}, true)
}

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

func (h *ActionInterestHandler) recordAttempt(surface, outcome string) {
	if h == nil || h.DB == nil {
		return
	}
	if err := models.RecordActionInterestAttempt(h.DB, surface, outcome); err != nil {
		log.Printf("action-interest aggregate surface=%s outcome=%s: %v", surface, outcome, err)
	}
}

func actionInterestAttemptOutcome(err error) string {
	switch {
	case err == nil:
		return "created"
	case errors.Is(err, errActionInterestRateLimited):
		return "rate_limited"
	case errors.Is(err, errActionInterestCrossOrigin):
		return "cross_origin"
	case errors.Is(err, models.ErrInvalidActionInterest):
		return "invalid_request"
	case errors.Is(err, models.ErrActionInterestUnavailable):
		return "unavailable"
	case errors.Is(err, models.ErrActionInterestConflict):
		return "conflict"
	case errors.Is(err, models.ErrActionInterestStoreUnavailable):
		return "store_unavailable"
	default:
		return "internal_error"
	}
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
		h.recordAttempt("rest", actionInterestAttemptOutcome(err))
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
		h.recordAttempt("rest", "invalid_request")
		providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid action-interest request"})
		return
	}

	receipt, err := h.recordAfterGate(r, request, "rest")
	if err != nil {
		h.recordAttempt("rest", actionInterestAttemptOutcome(err))
		status, message := actionInterestStatus(err)
		providerWriteJSON(w, status, map[string]string{"error": message})
		return
	}
	status := http.StatusCreated
	if receipt.Replayed {
		status = http.StatusOK
		h.recordAttempt("rest", "replayed")
	} else {
		h.recordAttempt("rest", "created")
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
		if err != nil || parsed < models.Stage1ReportMinimumDays || parsed > models.ActionInterestRetentionDays {
			providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("days must be %d..%d", models.Stage1ReportMinimumDays, models.ActionInterestRetentionDays)})
			return
		}
		days = parsed
	}
	proof, err := models.GetStage1DemandProof(h.DB, days)
	if err != nil {
		providerWriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "stage 1 demand query failed"})
		return
	}
	funnel, err := models.GetActionInterestAttemptFunnel(h.DB, days)
	if err != nil {
		providerWriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "action-interest attempt funnel query failed"})
		return
	}
	providerWriteJSON(w, http.StatusOK, map[string]any{
		"stage1_demand":                  proof,
		"action_interest_attempt_funnel": funnel,
		"evidence_scope":                 "Demand receipts are not unique agents, principals, provider-accepted handoffs, activations, revenue, or commercial proof. Attempt aggregates are operational diagnostics only and contain no request or entity coordinates.",
	})
}
