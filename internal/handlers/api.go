package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

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
		"$schema":             "https://schema.org/WebAPI",
		"name":                "Not Human Search API v1",
		"description":         "Search engine for agent-ready sites ranked by agentic readiness score (0-100).",
		"version":             "1.0.0",
		"base_url":            "https://nothumansearch.ai/api/v1",
		"openapi_spec":        "https://nothumansearch.ai/openapi.yaml",
		"ai_plugin_manifest":  "https://nothumansearch.ai/.well-known/ai-plugin.json",
		"mcp_endpoint":        "https://nothumansearch.ai/mcp",
		"endpoints": map[string]string{
			"search":           "GET /api/v1/search?q=&category=&min_score=&page=",
			"site":             "GET /api/v1/site/{domain}",
			"submit":           "POST /api/v1/submit",
			"stats":            "GET /api/v1/stats",
			"categories":       "GET /api/v1/categories",
			"check":            "GET /api/v1/check?url=",
			"monitor_register": "POST /api/v1/monitor/register",
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

	params := models.SearchParams{
		Query:      q.Get("q"),
		Category:   q.Get("category"),
		Tag:        q.Get("tag"),
		MinScore:   minScore,
		HasAPI:     q.Get("has_api") == "true",
		HasMCP:     q.Get("has_mcp") == "true",
		HasOpenAPI: q.Get("has_openapi") == "true",
		HasLLMsTxt: q.Get("has_llms_txt") == "true",
		Limit:      20,
		Page:       page,
	}

	sites, total, err := models.SearchSites(h.DB, params)
	if err != nil {
		h.writeJSON(w, 500, map[string]string{"error": "search failed"})
		return
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
	}

	h.writeJSON(w, 200, map[string]interface{}{
		"results":  sites,
		"total":    total,
		"page":     page,
		"has_next": page*20 < total,
	})
}

// GET /api/v1/site/:domain
func (h *APIHandler) GetSite(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Path[len("/api/v1/site/"):]
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
	h.writeJSON(w, 200, map[string]interface{}{
		"total_sites":  totalSites,
		"avg_score":    avgScore,
		"top_category": topCategory,
	})
}

// GET /api/v1/categories
func (h *APIHandler) Categories(w http.ResponseWriter, r *http.Request) {
	cats, err := models.GetCategories(h.DB)
	if err != nil {
		h.writeJSON(w, 500, map[string]string{"error": "failed to get categories"})
		return
	}
	h.writeJSON(w, 200, map[string]interface{}{
		"categories": cats,
	})
}
