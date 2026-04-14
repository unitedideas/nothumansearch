package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/unitedideas/nothumansearch/internal/crawler"
	"github.com/unitedideas/nothumansearch/internal/models"
)

type APIHandler struct {
	DB *sql.DB
}

func NewAPIHandler(db *sql.DB) *APIHandler {
	return &APIHandler{DB: db}
}

func (h *APIHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
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
		Query:    q.Get("q"),
		Category: q.Get("category"),
		MinScore: minScore,
		HasAPI:   q.Get("has_api") == "true",
		Limit:    20,
		Page:     page,
	}

	sites, total, err := models.SearchSites(h.DB, params)
	if err != nil {
		h.writeJSON(w, 500, map[string]string{"error": "search failed"})
		return
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

	// Crawl immediately in background
	go func() {
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
