package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/unitedideas/nothumansearch/internal/crawler"
	"github.com/unitedideas/nothumansearch/internal/models"
)

type APIHandler struct {
	DB *sql.DB
}

// submitCrawlSem caps concurrent inline crawl goroutines spawned by /api/v1/submit.
// Without this, a bulk submitter can OOM a small Postgres instance by spawning
// hundreds of simultaneous crawl+upsert goroutines. Requests above the cap still
// queue in submissions table and get picked up by the scheduled recrawl.
var submitCrawlSem = make(chan struct{}, 4)

// submitRateLimiter is a per-IP rolling hourly limiter for /api/v1/submit.
// 20/hour/IP lets legitimate bulk submitters (e.g. directory importers) run
// through a few hundred URLs over an hour without blocking, but prevents
// single-IP floods of the submissions table. Legitimate agent use is far
// below this: most agents submit 1-3 sites total.
var (
	submitRLMu      sync.Mutex
	submitRLCounts  = map[string]int{}
	submitRLResetAt = time.Now().Add(time.Hour)
)

const submitRateLimit = 20

func submitRLAllow(ipHash string) (allowed bool, remaining int, resetUnix int64) {
	submitRLMu.Lock()
	defer submitRLMu.Unlock()
	now := time.Now()
	if now.After(submitRLResetAt) {
		submitRLCounts = map[string]int{}
		submitRLResetAt = now.Add(time.Hour)
	}
	if submitRLCounts[ipHash] >= submitRateLimit {
		return false, 0, submitRLResetAt.Unix()
	}
	submitRLCounts[ipHash]++
	remaining = submitRateLimit - submitRLCounts[ipHash]
	return true, remaining, submitRLResetAt.Unix()
}

func submitHashIP(r *http.Request) string {
	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.Split(fwd, ",")[0]
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(ip)))
	return hex.EncodeToString(sum[:8])
}

func NewAPIHandler(db *sql.DB) *APIHandler {
	return &APIHandler{DB: db}
}

func (h *APIHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// GET /api/v1 — API index. Returned so that crawlers (including our own
// agent-first filter) can discover the structured API from the apex. The
// crawler's isAPIResponse check requires a JSON body at /api/v1; without
// this, NHS's own site loses the structured_api signal (15 points).
func (h *APIHandler) Index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1" && r.URL.Path != "/api/v1/" {
		http.NotFound(w, r)
		return
	}
	h.writeJSON(w, 200, map[string]any{
		"$schema":            "https://schema.org/WebAPI",
		"name":               "Not Human Search API v1",
		"description":        "Search engine for agent-ready sites ranked by agentic readiness score (0-100).",
		"version":            "1.0.0",
		"base_url":           "https://nothumansearch.ai/api/v1",
		"openapi_spec":       "https://nothumansearch.ai/openapi.yaml",
		"ai_plugin_manifest": "https://nothumansearch.ai/.well-known/ai-plugin.json",
		"mcp_endpoint":       "https://nothumansearch.ai/mcp",
		"endpoints": map[string]string{
			"search":            "GET /api/v1/search?q=&category=&tag=&min_score=&has_api=&has_mcp=&has_openapi=&has_llms_txt=&page=",
			"site":              "GET /api/v1/site/{domain}",
			"submit":            "POST /api/v1/submit",
			"stats":             "GET /api/v1/stats",
			"top":               "GET /api/v1/top?category=&has_mcp=&has_openapi=&has_llms_txt=&limit=",
			"categories":        "GET /api/v1/categories",
			"check":             "GET /api/v1/check?url=",
			"verify_mcp":        "GET /api/v1/verify-mcp?url=",
			"commerce_catalog":  "GET /api/v1/catalog",
			"commerce_quote":    "POST /api/v1/quote",
			"commerce_checkout": "POST /api/v1/checkout",
			"monitor_register":  "POST /api/v1/monitor/register",
		},
		"auth":       "none",
		"rate_limit": "60 req/min per IP",
	})
}

// GET /api/v1/search?q=...&category=...&min_score=...&page=...
func (h *APIHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := 1
	if p := q.Get("page"); p != "" {
		if pn, err := strconv.Atoi(p); err == nil {
			page = pn
		}
	}
	minScore := 0
	if ms := q.Get("min_score"); ms != "" {
		if s, err := strconv.Atoi(ms); err == nil {
			minScore = s
		}
	}
	perPage := 20
	if pp := q.Get("per_page"); pp != "" {
		if n, err := strconv.Atoi(pp); err == nil && n > 0 {
			perPage = n
		}
	}
	if perPage > 50 {
		perPage = 50
	}

	params := models.SearchParams{
		Query:      q.Get("q"),
		Category:   q.Get("category"),
		Tag:        q.Get("tag"),
		MinScore:   minScore,
		HasAPI:     q.Get("has_api") == "true",
		HasMCP:     q.Get("has_mcp") == "true",
		HasOpenAPI: q.Get("has_openapi") == "true",
		HasLLMsTxt: q.Get("has_llms_txt") == "true",
		Limit:      perPage,
		Page:       page,
	}

	sites, total, err := models.SearchSites(h.DB, params)
	if err != nil {
		h.writeJSON(w, 500, map[string]string{"error": "search failed"})
		return
	}
	// Never return JSON null for results — consumers iterate without nil-check.
	if sites == nil {
		sites = []models.Site{}
	}

	// Log search query for analytics (non-blocking)
	if params.Query != "" {
		go func() {
			ip := r.Header.Get("X-Forwarded-For")
			if ip == "" {
				ip = strings.Split(r.RemoteAddr, ":")[0]
			}
			hash := sha256.Sum256([]byte(ip))
			ipHash := hex.EncodeToString(hash[:8])
			models.LogSearch(h.DB, params.Query, total, r.UserAgent(), ipHash)
		}()
		go models.LogIntentFromRequest(h.DB, r, "search_query", "query", params.Query, map[string]any{
			"results":  total,
			"category": params.Category,
			"tag":      params.Tag,
			"has_mcp":  params.HasMCP,
		})
	}

	h.writeJSON(w, 200, map[string]interface{}{
		"results":  sites,
		"total":    total,
		"page":     page,
		"per_page": perPage,
		"has_next": page*perPage < total,
	})
}

// GET /api/v1/site/:domain
func (h *APIHandler) GetSite(w http.ResponseWriter, r *http.Request) {
	domain := strings.TrimPrefix(r.URL.Path, "/api/v1/site/")
	domain = strings.TrimPrefix(domain, "sites/")
	domain = strings.TrimPrefix(domain, "/api/v1/sites/")
	if domain == "" {
		h.writeJSON(w, 400, map[string]string{"error": "domain required"})
		return
	}

	site, err := models.GetSiteByDomain(h.DB, domain)
	if err != nil {
		h.writeJSON(w, 404, map[string]string{"error": "site not found"})
		return
	}

	h.writeJSON(w, 200, site)
}

// POST /api/v1/submit  {"url": "https://example.com"}
func (h *APIHandler) SubmitSite(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		h.writeJSON(w, 405, map[string]string{"error": "POST required"})
		return
	}

	ipHash := submitHashIP(r)
	allowed, remaining, resetUnix := submitRLAllow(ipHash)
	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", submitRateLimit))
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
	w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetUnix))
	if !allowed {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Until(time.Unix(resetUnix, 0)).Seconds())+1))
		h.writeJSON(w, 429, map[string]any{
			"error":     "rate limit exceeded: 20 submissions per hour per IP",
			"retry_sec": int(time.Until(time.Unix(resetUnix, 0)).Seconds()) + 1,
		})
		return
	}

	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		h.writeJSON(w, 400, map[string]string{"error": "url required"})
		return
	}

	_, err := h.DB.Exec(`
		INSERT INTO submissions (url, status) VALUES ($1, 'pending')
		ON CONFLICT DO NOTHING`, req.URL)
	if err != nil {
		h.writeJSON(w, 500, map[string]string{"error": "submission failed"})
		return
	}

	// Try to crawl immediately, but only if we're below concurrency cap.
	// Otherwise the submission stays in 'pending' and the scheduled recrawl
	// picks it up — avoiding OOM storms during bulk submissions.
	select {
	case submitCrawlSem <- struct{}{}:
		go func() {
			defer func() { <-submitCrawlSem }()
			site, err := crawler.CrawlSite(req.URL)
			if err != nil {
				log.Printf("submit crawl failed for %s: %v", req.URL, err)
				h.DB.Exec("UPDATE submissions SET status='failed' WHERE url=$1", req.URL)
				return
			}
			if err := models.UpsertSite(h.DB, site); err != nil {
				log.Printf("submit upsert failed for %s: %v", req.URL, err)
			}
			h.DB.Exec("UPDATE submissions SET status='crawled' WHERE url=$1", req.URL)
			log.Printf("submit crawl success: %s score=%d", site.Domain, site.AgenticScore)
		}()
	default:
		// semaphore full — leave as 'pending', recrawl will handle it
	}

	h.writeJSON(w, 201, map[string]string{"message": "submitted for crawling"})
}

// GET /api/v1/stats
func (h *APIHandler) Stats(w http.ResponseWriter, r *http.Request) {
	totalSites, avgScore, topCategory := models.GetStats(h.DB)
	w.Header().Set("Cache-Control", "public, max-age=300")
	h.writeJSON(w, 200, map[string]interface{}{
		"total_sites":  totalSites,
		"avg_score":    avgScore,
		"top_category": topCategory,
	})
}

// Top returns the highest-scored sites in the index, sorted by agentic_score
// DESC. Filterable by category and signal (has_mcp, has_llms_txt, etc).
// Public, free, cached 5 min — designed as a stable JSON other sites can
// mirror/embed. GET /api/v1/top?category=&has_mcp=&limit=
func (h *APIHandler) Top(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		h.writeJSON(w, 405, map[string]string{"error": "GET only"})
		return
	}
	q := r.URL.Query()
	limit := 50
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 100 {
		limit = 100
	}

	sites, total, err := models.SearchSites(h.DB, models.SearchParams{
		Category:   q.Get("category"),
		Tag:        q.Get("tag"),
		HasAPI:     q.Get("has_api") == "true",
		HasMCP:     q.Get("has_mcp") == "true",
		HasOpenAPI: q.Get("has_openapi") == "true",
		HasLLMsTxt: q.Get("has_llms_txt") == "true",
		Limit:      limit,
		Page:       1,
	})
	if err != nil {
		h.writeJSON(w, 500, map[string]string{"error": "query failed"})
		return
	}
	if sites == nil {
		sites = []models.Site{}
	}

	w.Header().Set("Cache-Control", "public, max-age=300")
	h.writeJSON(w, 200, map[string]interface{}{
		"results":     sites,
		"total":       total,
		"limit":       limit,
		"source":      "https://nothumansearch.ai",
		"description": "Highest-scored agent-ready sites, sorted by agentic readiness. Free, public, no auth.",
	})
}

// VerifyMCP is the REST wrapper around crawler.ProbeMCPJSONRPC — same
// behavior as the MCP verify_mcp tool, reachable by agents that don't
// speak MCP themselves.
// GET /api/v1/verify-mcp?url=https://example.com/mcp
func (h *APIHandler) VerifyMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		h.writeJSON(w, 405, map[string]string{"error": "GET only"})
		return
	}
	raw := strings.TrimSpace(r.URL.Query().Get("url"))
	if raw == "" {
		h.writeJSON(w, 400, map[string]string{"error": "url query param required"})
		return
	}
	// Case-insensitive prefix check — without ToLower, an input like
	// "HTTPS://example.com/mcp" would fail both HasPrefix checks and get
	// re-prefixed to "https://HTTPS://example.com/mcp" (broken URL).
	lower := strings.ToLower(raw)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		raw = "https://" + raw
	}

	// Don't cache — the caller is asking "is it live RIGHT NOW".
	w.Header().Set("Cache-Control", "no-store")

	verified := crawler.ProbeMCPJSONRPC(raw)
	note := "Endpoint responded with valid JSON-RPC 2.0 — server is live and MCP-compliant."
	if !verified {
		note = "Endpoint did not respond with valid JSON-RPC 2.0. Could be down, not an MCP server, or requires an initialize() handshake this probe does not send."
	}
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	h.writeJSON(w, 200, map[string]interface{}{
		"verified": verified,
		"endpoint": raw,
		"note":     note,
	})
}

// GET /api/v1/categories
func (h *APIHandler) Categories(w http.ResponseWriter, r *http.Request) {
	cats, err := models.GetCategories(h.DB)
	if err != nil {
		h.writeJSON(w, 500, map[string]string{"error": "failed to get categories"})
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	h.writeJSON(w, 200, map[string]interface{}{
		"categories": cats,
	})
}

func (h *APIHandler) MCPAnalytics(w http.ResponseWriter, r *http.Request) {
	adminKey := os.Getenv("ADMIN_API_KEY")
	if adminKey == "" {
		h.writeJSON(w, 503, map[string]string{"error": "admin endpoint not configured"})
		return
	}
	auth := r.Header.Get("Authorization")
	if auth != "Bearer "+adminKey {
		h.writeJSON(w, 401, map[string]string{"error": "invalid admin key"})
		return
	}
	days := 14
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 90 {
			days = n
		}
	}
	data, err := models.GetMCPAnalytics(h.DB, days)
	if err != nil {
		h.writeJSON(w, 500, map[string]string{"error": "query failed"})
		return
	}
	data["days"] = days
	h.writeJSON(w, 200, data)
}

func (h *APIHandler) TrafficAnalytics(w http.ResponseWriter, r *http.Request) {
	adminKey := os.Getenv("ADMIN_API_KEY")
	if adminKey == "" {
		h.writeJSON(w, 503, map[string]string{"error": "admin endpoint not configured"})
		return
	}
	auth := r.Header.Get("Authorization")
	if auth != "Bearer "+adminKey {
		h.writeJSON(w, 401, map[string]string{"error": "invalid admin key"})
		return
	}
	days := 14
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 90 {
			days = n
		}
	}
	data, err := models.GetTrafficAnalytics(h.DB, days)
	if err != nil {
		h.writeJSON(w, 500, map[string]string{"error": "query failed"})
		return
	}
	data["days"] = days
	h.writeJSON(w, 200, data)
}

func (h *APIHandler) SignalAnalytics(w http.ResponseWriter, r *http.Request) {
	adminKey := os.Getenv("ADMIN_API_KEY")
	if adminKey == "" {
		h.writeJSON(w, 503, map[string]string{"error": "admin endpoint not configured"})
		return
	}
	auth := r.Header.Get("Authorization")
	if auth != "Bearer "+adminKey {
		h.writeJSON(w, 401, map[string]string{"error": "invalid admin key"})
		return
	}
	days := 14
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 90 {
			days = n
		}
	}
	data, err := models.GetIntentAnalytics(h.DB, days)
	if err != nil {
		h.writeJSON(w, 500, map[string]string{"error": "query failed"})
		return
	}
	data["days"] = days
	h.writeJSON(w, 200, data)
}
