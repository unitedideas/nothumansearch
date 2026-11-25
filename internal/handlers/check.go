package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/unitedideas/nothumansearch/internal/crawler"
	"github.com/unitedideas/nothumansearch/internal/models"
)

// CheckHandler exposes on-demand agentic-readiness checks at /api/v1/check.
// Developers can POST a URL and get back the same 7-signal score the crawler
// produces, without waiting for indexing. This is the primary monetization
// surface. Every caller retains a free safety-limited path; active legacy API
// keys receive a higher priority-throughput ceiling while their monthly
// allocation remains.
type CheckHandler struct {
	DB *sql.DB

	// naive in-memory rate limiter, per IP hash. Resets every window.
	mu      sync.Mutex
	counts  map[string]int
	resetAt time.Time
}

func NewCheckHandler(db *sql.DB) *CheckHandler {
	return &CheckHandler{
		DB:      db,
		counts:  map[string]int{},
		resetAt: time.Now().Add(time.Hour),
	}
}

const (
	checkWindow        = time.Hour
	checkFreeLimit     = freeCheckHourlyLimit
	checkPriorityLimit = priorityCheckHourlyLimit
	checkMaxBodyBytes  = 2048
)

func (h *CheckHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// ServeHTTP handles GET (docs) and POST (check) at /api/v1/check.
func (h *CheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.writeJSON(w, 200, map[string]any{
			"endpoint":      "/api/v1/check",
			"method":        "POST",
			"description":   "On-demand agentic readiness check. Returns the same 7-signal score the NHS crawler computes.",
			"body":          map[string]string{"url": "https://example.com"},
			"free_tier":     "10 checks/hour per IP. No key required.",
			"priority_tier": "100 checks/hour per active API key while its 50,000-call monthly priority allocation remains; free access continues afterward.",
			"example":       "curl -X POST https://nothumansearch.ai/api/v1/check -H 'Content-Type: application/json' -d '{\"url\":\"https://stripe.com\"}'",
		})
		return
	}
	if r.Method != http.MethodPost {
		h.writeJSON(w, 405, map[string]string{"error": "POST or GET only"})
		return
	}

	access := resolveRequestRateAccess(h.DB, r)
	limit := checkFreeLimit
	if access.tier == priorityRateLimitTier {
		limit = checkPriorityLimit
	}
	allowed := h.allow(access.bucket, limit)
	if allowed && access.key != nil {
		var reserved bool
		access, reserved = reservePriorityUnit(h.DB, access, "rest", r.Method, "/api/v1/check", "")
		if !reserved {
			access = freeRequestRateAccess(r)
			limit = checkFreeLimit
			allowed = h.allow(access.bucket, limit)
		}
	}
	access.setHeaders(w)
	if !allowed {
		remaining, resetUnix := h.rateLimitState(access.bucket, limit)
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Until(time.Unix(resetUnix, 0)).Seconds())+1))
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetUnix))
		h.writeJSON(w, 429, map[string]any{
			"error":            "rate limit exceeded",
			"limit":            limit,
			"window_sec":       int(checkWindow.Seconds()),
			"reset_at_unix":    resetUnix,
			"plans_url":        "https://nothumansearch.ai/api/v1/api-keys/subscribe",
			"subscribe_url":    "https://nothumansearch.ai/api/v1/api-keys/subscribe",
			"subscribe_method": "POST",
			"subscribe_fields": []string{"email", "plan"},
			"upgrade":          "An active API key raises the safety ceiling; free checking remains available without one.",
		})
		go models.LogIntentFromRequest(h.DB, r, "check_rate_limit_hit", "api", "/api/v1/check", map[string]any{
			"limit": limit,
		})
		return
	}
	// Emit rate-limit headers on successful responses too so callers can pace themselves.
	{
		remaining, resetUnix := h.rateLimitState(access.bucket, limit)
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetUnix))
	}

	var req struct {
		URL string `json:"url"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, checkMaxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		access = releasePriorityUnit(h.DB, access)
		access.setHeaders(w)
		h.writeJSON(w, 400, map[string]string{"error": "invalid JSON body"})
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" {
		access = releasePriorityUnit(h.DB, access)
		access.setHeaders(w)
		h.writeJSON(w, 400, map[string]string{"error": "url required"})
		return
	}
	// Case-insensitive prefix check — without ToLower, an input like
	// "HTTPS://stripe.com" would fail both HasPrefix checks and get
	// re-prefixed to "https://HTTPS://stripe.com" (broken URL).
	lowerURL := strings.ToLower(req.URL)
	if !strings.HasPrefix(lowerURL, "http://") && !strings.HasPrefix(lowerURL, "https://") {
		req.URL = "https://" + req.URL
	}
	if err := crawler.ValidatePublicURL(req.URL); err != nil {
		access = releasePriorityUnit(h.DB, access)
		access.setHeaders(w)
		h.writeJSON(w, 400, map[string]string{"error": "url must resolve to a public HTTP(S) address"})
		return
	}
	select {
	case submitCrawlSem <- struct{}{}:
		// Released by the crawl goroutine when it actually exits. A handler
		// timeout cannot release early and accumulate runaway background crawls.
	default:
		access = releasePriorityUnit(h.DB, access)
		access.setHeaders(w)
		w.Header().Set("Retry-After", "2")
		h.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "live check capacity is busy; retry shortly"})
		return
	}

	// Run the crawler inline. CrawlSite hits the network; cap total time so a
	// slow target can't pin the request open.
	done := make(chan struct{})
	var site *models.Site
	var crawlErr error
	go func() {
		defer func() { <-submitCrawlSem }()
		site, crawlErr = crawler.CrawlSite(req.URL)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(25 * time.Second):
		access = releasePriorityUnit(h.DB, access)
		access.setHeaders(w)
		h.writeJSON(w, 504, map[string]string{"error": "target site took too long to respond"})
		return
	}
	if crawlErr != nil {
		access = releasePriorityUnit(h.DB, access)
		access.setHeaders(w)
		h.writeJSON(w, 502, map[string]string{"error": "crawl failed: " + crawlErr.Error()})
		return
	}
	go models.LogIntentFromRequest(h.DB, r, "score_checked", "site", site.Domain, map[string]any{
		"score":          site.AgenticScore,
		"category":       site.Category,
		"has_mcp":        site.HasMCPServer,
		"has_openapi":    site.HasOpenAPI,
		"structured_api": site.HasStructuredAPI,
	})

	// Persist the result so this check also improves the index. Fire-and-forget;
	// failures here don't affect the caller's response. Skip when DB is unset
	// (test mode / lead-capture-only) — a nil handle would panic the process.
	go func() {
		if h.DB == nil {
			return
		}
		if err := models.UpsertSite(h.DB, site); err != nil {
			// Log via package-level logger to avoid logging in the hot path.
			// (import "log" kept out of this file intentionally; UpsertSite errors
			// are non-critical for the check response.)
			_ = err
		}
	}()

	h.writeJSON(w, 200, map[string]any{
		"domain":        site.Domain,
		"url":           site.URL,
		"agentic_score": site.AgenticScore,
		"category":      site.Category,
		"signals": map[string]bool{
			"llms_txt":       site.HasLLMsTxt,
			"ai_plugin":      site.HasAIPlugin,
			"openapi":        site.HasOpenAPI,
			"structured_api": site.HasStructuredAPI,
			"mcp_server":     site.HasMCPServer,
			"robots_ai":      site.HasRobotsAI,
			"schema_org":     site.HasSchemaOrg,
		},
		"report_url": "https://nothumansearch.ai/site/" + site.Domain,
	})
}

// allow increments the per-IP counter and returns true if the request is
// under the free-tier limit.
func (h *CheckHandler) allow(ipHash string, limit int) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	if now.After(h.resetAt) {
		h.counts = map[string]int{}
		h.resetAt = now.Add(checkWindow)
	}
	if h.counts[ipHash] >= limit {
		return false
	}
	h.counts[ipHash]++
	return true
}

// rateLimitState returns the current (remaining, resetUnix) for a given IP
// without incrementing. Used to emit X-RateLimit-* headers on every response
// so callers can back off gracefully instead of surprise-429ing.
func (h *CheckHandler) rateLimitState(ipHash string, limit int) (remaining int, resetUnix int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	remaining = limit - h.counts[ipHash]
	if remaining < 0 {
		remaining = 0
	}
	resetUnix = h.resetAt.Unix()
	return
}
