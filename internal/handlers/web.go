package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/unitedideas/nothumansearch/internal/models"
)

type WebHandler struct {
	DB   *sql.DB
	tmpl *template.Template
}

func NewWebHandler(db *sql.DB, templatesDir string) (*WebHandler, error) {
	funcMap := template.FuncMap{
		"scoreClass": func(score int) string {
			if score >= 70 {
				return "high"
			}
			if score >= 40 {
				return "medium"
			}
			return "low"
		},
		"scoreLabel": func(score int) string {
			if score >= 70 {
				return "Agent Ready"
			}
			if score >= 40 {
				return "Partial"
			}
			return "Basic"
		},
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"initial": func(domain string) string {
			// First alphabetic character of the domain, uppercased — for favicon fallback.
			for _, r := range domain {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
					return strings.ToUpper(string(r))
				}
			}
			return "?"
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseGlob(filepath.Join(templatesDir, "*.html"))
	if err != nil {
		return nil, err
	}
	return &WebHandler{DB: db, tmpl: tmpl}, nil
}

func (h *WebHandler) HomePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	q := r.URL.Query().Get("q")
	category := r.URL.Query().Get("category")
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if pn, err := strconv.Atoi(p); err == nil {
			page = pn
		}
	}

	params := models.SearchParams{
		Query:    q,
		Category: category,
		Limit:    20,
		Page:     page,
	}

	sites, total, err := models.SearchSites(h.DB, params)
	if err != nil {
		log.Printf("search error: %v", err)
	}

	totalSites, avgScore, _ := models.GetStats(h.DB)

	data := map[string]interface{}{
		"Query":      q,
		"Category":   category,
		"Sites":      sites,
		"Total":      total,
		"Page":       page,
		"HasNext":    page*20 < total,
		"TotalSites": totalSites,
		"AvgScore":   avgScore,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "home.html", data); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "internal error", 500)
	}
}

func (h *WebHandler) AboutPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "about.html", nil); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "internal error", 500)
	}
}

func (h *WebHandler) SitePage(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Path[len("/site/"):]
	if domain == "" {
		http.NotFound(w, r)
		return
	}

	site, err := models.GetSiteByDomain(h.DB, domain)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "site.html", site); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "internal error", 500)
	}
}
