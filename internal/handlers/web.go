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

// MCPServersPage renders a dedicated landing page listing every MCP server
// in the index. Canonical URL /mcp-servers — targets the "mcp server
// directory" query class without the noise from /?q=mcp.
func (h *WebHandler) MCPServersPage(w http.ResponseWriter, r *http.Request) {
	h.renderCategoryLanding(w, r, categoryLanding{
		Mode:       "mcp-servers",
		Path:       "/mcp-servers",
		Title:      "MCP Server Directory — Browse All Model Context Protocol Servers | Not Human Search",
		Desc:       "Complete directory of MCP servers ranked by agentic readiness. Find Model Context Protocol endpoints for every AI agent use case — search, data, automation, commerce, and more.",
		Heading:    "MCP Server Directory",
		Subheading: "Every Model Context Protocol server in our index, ranked by agentic readiness score.",
		Params:     models.SearchParams{HasMCP: true, Limit: 30},
	})
}

// AIToolsPage: /ai-tools landing — targets "AI tools directory" / "ai agent tools" queries.
func (h *WebHandler) AIToolsPage(w http.ResponseWriter, r *http.Request) {
	h.renderCategoryLanding(w, r, categoryLanding{
		Mode:       "ai-tools",
		Path:       "/ai-tools",
		Title:      "AI Tools Directory — Browse Agent-Ready AI Tools & APIs | Not Human Search",
		Desc:       "Curated directory of AI tools that expose llms.txt, OpenAPI, or MCP endpoints — ranked by how well they serve AI agents. The agent-native alternative to generic AI tool lists.",
		Heading:    "AI Tools Directory",
		Subheading: "Every AI tool in our index, ranked by agentic readiness. These are the tools AI agents can actually use programmatically.",
		Params:     models.SearchParams{Category: "ai-tools", Limit: 30},
	})
}

// DeveloperPage: /developer-apis — targets "developer API directory" / "agent-ready APIs" queries.
func (h *WebHandler) DeveloperPage(w http.ResponseWriter, r *http.Request) {
	h.renderCategoryLanding(w, r, categoryLanding{
		Mode:       "developer",
		Path:       "/developer-apis",
		Title:      "Developer API Directory — Agent-Ready APIs for AI Engineers | Not Human Search",
		Desc:       "Every developer API in our index that AI agents can discover and call — ranked by agentic readiness. Find APIs with OpenAPI specs, llms.txt, ai-plugin.json, or MCP endpoints.",
		Heading:    "Developer API Directory",
		Subheading: "Every developer API in our index, ranked by agentic readiness. All entries expose at least one programmatic signal agents can discover at build time.",
		Params:     models.SearchParams{Category: "developer", Limit: 30},
	})
}

type categoryLanding struct {
	Mode       string
	Path       string
	Title      string
	Desc       string
	Heading    string
	Subheading string
	Params     models.SearchParams
}

func (h *WebHandler) renderCategoryLanding(w http.ResponseWriter, r *http.Request, c categoryLanding) {
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if pn, err := strconv.Atoi(p); err == nil && pn > 0 {
			page = pn
		}
	}
	c.Params.Page = page

	sites, total, err := models.SearchSites(h.DB, c.Params)
	if err != nil {
		log.Printf("%s search: %v", c.Path, err)
	}

	totalSites, avgScore, _ := models.GetStats(h.DB)

	data := map[string]interface{}{
		"Mode":       c.Mode,
		"PageTitle":  c.Title,
		"PageDesc":   c.Desc,
		"Heading":    c.Heading,
		"Subheading": c.Subheading,
		"Sites":      sites,
		"Total":      total,
		"Page":       page,
		"HasNext":    page*c.Params.Limit < total,
		"TotalSites": totalSites,
		"AvgScore":   avgScore,
		"Canonical":  "https://nothumansearch.ai" + c.Path,
		"BasePath":   c.Path,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "home.html", data); err != nil {
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
