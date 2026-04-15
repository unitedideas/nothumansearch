package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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
// surface — free tier is rate-limited; paid tiers (future) unlock higher
// limits and CI-grade webhooks.
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
	checkWindow       = time.Hour
	checkFreeLimit    = 10 // requests per hour per IP
	checkMaxBodyBytes = 2048
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
			"endpoint":    "/api/v1/check",
			"method":      "POST",
			"description": "On-demand agentic readiness check. Returns the same 7-signal score the NHS crawler computes.",
			"body":        map[string]string{"url": "https://example.com"},
			"free_tier":   "10 checks/hour per IP. No key required.",
			"example":     "curl -X POST https://nothumansearch.ai/api/v1/check -H 'Content-Type: application/json' -d '{\"url\":\"https://stripe.com\"}'",
		})
		return
	}
	if r.Method != http.MethodPost {
		h.writeJSON(w, 405, map[string]string{"error": "POST or GET only"})
		return
	}

	ipHash := hashIP(r)
	if !h.allow(ipHash) {
		remaining, resetUnix := h.rateLimitState(ipHash)
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Until(time.Unix(resetUnix, 0)).Seconds())+1))
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", checkFreeLimit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetUnix))
		h.writeJSON(w, 429, map[string]any{
			"error":      "rate limit exceeded",
			"limit":      checkFreeLimit,
			"window_sec": int(checkWindow.Seconds()),
			"upgrade":    "Higher limits and CI-grade webhooks coming soon. Email hello@nothumansearch.ai to join the paid-tier waitlist.",
		})
		return
	}
	// Emit rate-limit headers on successful responses too so callers can pace themselves.
	{
		remaining, resetUnix := h.rateLimitState(ipHash)
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", checkFreeLimit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetUnix))
	}

	var req struct {
		URL string `json:"url"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, checkMaxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeJSON(w, 400, map[string]string{"error": "invalid JSON body"})
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" {
		h.writeJSON(w, 400, map[string]string{"error": "url required"})
		return
	}
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		req.URL = "https://" + req.URL
	}

	// Run the crawler inline. CrawlSite hits the network; cap total time so a
	// slow target can't pin the request open.
	done := make(chan struct{})
	var site *models.Site
	var crawlErr error
	go func() {
		site, crawlErr = crawler.CrawlSite(req.URL)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(25 * time.Second):
		h.writeJSON(w, 504, map[string]string{"error": "target site took too long to respond"})
		return
	}
	if crawlErr != nil {
		h.writeJSON(w, 502, map[string]string{"error": "crawl failed: " + crawlErr.Error()})
		return
	}

	// Persist the result so this check also improves the index. Fire-and-forget;
	// failures here don't affect the caller's response.
	go func() {
		if err := models.UpsertSite(h.DB, site); err != nil {
			// Log via package-level logger to avoid logging in the hot path.
			// (import "log" kept out of this file intentionally; UpsertSite errors
			// are non-critical for the check response.)
			_ = err
		}
	}()

	h.writeJSON(w, 200, map[string]any{
		"domain":         site.Domain,
		"url":            site.URL,
		"agentic_score":  site.AgenticScore,
		"category":       site.Category,
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
func (h *CheckHandler) allow(ipHash string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	if now.After(h.resetAt) {
		h.counts = map[string]int{}
		h.resetAt = now.Add(checkWindow)
	}
	if h.counts[ipHash] >= checkFreeLimit {
		return false
	}
	h.counts[ipHash]++
	return true
}

// rateLimitState returns the current (remaining, resetUnix) for a given IP
// without incrementing. Used to emit X-RateLimit-* headers on every response
// so callers can back off gracefully instead of surprise-429ing.
func (h *CheckHandler) rateLimitState(ipHash string) (remaining int, resetUnix int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	remaining = checkFreeLimit - h.counts[ipHash]
	if remaining < 0 {
		remaining = 0
	}
	resetUnix = h.resetAt.Unix()
	return
}

func hashIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}
	if idx := strings.Index(ip, ","); idx > 0 {
		ip = ip[:idx]
	}
	if idx := strings.LastIndex(ip, ":"); idx > 0 {
		// strip port
		ip = ip[:idx]
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(ip)))
	return hex.EncodeToString(sum[:8])
}
