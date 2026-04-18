package handlers

import (
	"database/sql"
	"fmt"
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
		"tof": func(a int) float64 { return float64(a) },
		"mulf": func(a, b float64) float64 { return a * b },
		"divf": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
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
		h.NotFoundPage(w, r)
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
	popularTags, _ := models.TopTags(h.DB, 12)

	data := map[string]interface{}{
		"Query":       q,
		"Category":    category,
		"Sites":       sites,
		"Total":       total,
		"Page":        page,
		"HasNext":     page*20 < total,
		"TotalSites":  totalSites,
		"AvgScore":    avgScore,
		"PopularTags": popularTags,
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

// GuidePage renders /guide — evergreen long-form content covering the 7 signals.
// Primary SEO surface for "how to add llms.txt" / "make site agent-ready" queries
// NotFoundPage renders a branded 404 that surfaces the core navigation instead
// of leaving visitors at a bare "404 page not found" response. Sent on any
// path that didn't match a registered handler.
func (h *WebHandler) NotFoundPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, notFoundHTML)
}

const notFoundHTML = `<!DOCTYPE html>
<html lang="en"><head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Not found — Not Human Search</title>
<meta name="robots" content="noindex">
<link rel="icon" type="image/svg+xml" href="/static/img/logo.svg">
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{background:#0d0d0e;color:#e8e8e9;font-family:'Inter',system-ui,sans-serif;line-height:1.7;padding:40px 20px;min-height:100vh;display:flex;align-items:center;justify-content:center}
.wrap{max-width:640px;margin:0 auto;text-align:center}
.code{font-family:'IBM Plex Mono',ui-monospace,monospace;font-size:96px;font-weight:700;color:#d97757;line-height:1;margin-bottom:16px;letter-spacing:-0.04em}
h1{font-size:28px;color:#fff;margin-bottom:14px}
p{color:#c5c5c9;margin-bottom:28px}
form.search{display:flex;gap:8px;margin-bottom:32px}
form.search input{flex:1;padding:14px 18px;background:#111214;border:1px solid rgba(255,255,255,0.1);border-radius:8px;color:#e8e8e9;font-size:16px;font-family:inherit;outline:none}
form.search input:focus{border-color:#d97757}
form.search button{padding:14px 22px;background:#d97757;color:#fff;border:none;border-radius:8px;font-weight:600;font-size:15px;cursor:pointer;font-family:inherit}
form.search button:hover{background:#c76645}
.pills{display:flex;flex-wrap:wrap;gap:8px;margin-bottom:36px;justify-content:center}
.pills a{padding:6px 12px;background:rgba(217,119,87,0.08);border:1px solid rgba(217,119,87,0.25);border-radius:999px;color:#d97757;text-decoration:none;font-size:13px}
.pills a:hover{background:rgba(217,119,87,0.18)}
.links{display:grid;grid-template-columns:1fr;gap:10px;text-align:left}
.links a{display:block;padding:14px 18px;background:#111214;border:1px solid rgba(255,255,255,0.07);border-radius:8px;color:#e8e8e9;text-decoration:none;font-size:14px}
.links a:hover{border-color:#d97757}
.links a strong{color:#fff;display:block;margin-bottom:2px;font-size:15px}
.links a span{color:#8b8d91}
.home{display:inline-block;margin-top:28px;padding:10px 22px;background:transparent;color:#d97757;border:1px solid rgba(217,119,87,0.3);border-radius:6px;text-decoration:none;font-weight:500;font-size:14px}
</style>
</head><body><div class="wrap">
<div class="code">404</div>
<h1>That page doesn't exist</h1>
<p>Search the agentic web instead:</p>
<form class="search" action="/" method="get" role="search">
  <input type="search" name="q" placeholder="e.g. payment API, weather data, code review MCP" aria-label="Search agent-ready sites" autofocus>
  <button type="submit">Search</button>
</form>
<div class="pills">
  <a href="/?q=payment+API">payment API</a>
  <a href="/?q=weather">weather</a>
  <a href="/?q=AI+jobs">AI jobs</a>
  <a href="/?q=authentication">auth</a>
  <a href="/?q=code+hosting">code hosting</a>
  <a href="/mcp-servers">MCP servers</a>
</div>
<div class="links">
  <a href="/score"><strong>Score any URL →</strong><span>Run the 7-signal agentic-readiness check live.</span></a>
  <a href="/mcp-servers"><strong>MCP Server Directory →</strong><span>Every Model Context Protocol server, verified.</span></a>
  <a href="/mcp"><strong>Wire NHS into your agent →</strong><span><code>claude mcp add --transport http nothumansearch https://nothumansearch.ai/mcp</code></span></a>
</div>
<a class="home" href="/">← Not Human Search home</a>
</div></body></html>`

// and the canonical link target for badge + outreach CTAs.
func (h *WebHandler) GuidePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "guide.html", nil); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "internal error", 500)
	}
}

// ScorePage renders the public /score UI — a form that POSTs to /api/v1/check
// and displays the 7-signal breakdown inline. Free marketing surface.
func (h *WebHandler) ScorePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "score.html", nil); err != nil {
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
		OGImage:    "og-mcp-servers.png",
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
		OGImage:    "og-ai-tools.png",
		Params:     models.SearchParams{Category: "ai-tools", Limit: 30},
	})
}

// OpenAPIPage: /openapi-apis — surfaces every site exposing an OpenAPI spec.
func (h *WebHandler) OpenAPIPage(w http.ResponseWriter, r *http.Request) {
	h.renderCategoryLanding(w, r, categoryLanding{
		Mode:       "openapi",
		Path:       "/openapi-apis",
		Title:      "OpenAPI Directory — Browse APIs With OpenAPI Specs for AI Agents | Not Human Search",
		Desc:       "Every site in our index publishing a valid OpenAPI spec — the machine-readable contract AI agents use to call APIs at build time. Ranked by agentic readiness.",
		Heading:    "OpenAPI Spec Directory",
		Subheading: "Every API in our index that publishes an OpenAPI/Swagger spec — ranked by agentic readiness.",
		Params:     models.SearchParams{HasOpenAPI: true, Limit: 30},
	})
}

// NewestPage: /newest — sites most recently added to the index. Fresh content,
// shareable feed that updates daily as the discovery pipeline runs.
func (h *WebHandler) NewestPage(w http.ResponseWriter, r *http.Request) {
	h.renderCategoryLanding(w, r, categoryLanding{
		Mode:       "newest",
		Path:       "/newest",
		Title:      "Newest Agent-Ready Sites — Recently Added to Not Human Search",
		Desc:       "The most recently added agent-ready sites in our index. Fresh picks from the discovery pipeline — all verified as LLM/agent-usable (llms.txt, OpenAPI, ai-plugin, MCP, or structured API).",
		Heading:    "Newest Agent-Ready Sites",
		Subheading: "Recently indexed, verified agent-first. Check back weekly — discovery runs every Monday.",
		Params:     models.SearchParams{OrderNewest: true, Limit: 50},
	})
}

// TopPage: /top — evergreen leaderboard of the 100 highest-scoring agent-ready sites.
// Designed as a shareable, linkable reference — "the 100 sites that actually work with AI agents".
func (h *WebHandler) TopPage(w http.ResponseWriter, r *http.Request) {
	h.renderCategoryLanding(w, r, categoryLanding{
		Mode:       "top",
		Path:       "/top",
		Title:      "Top 100 Agent-Ready Sites — Ranked by Agentic Readiness | Not Human Search",
		Desc:       "The 100 highest-scoring agent-ready sites on the internet. Ranked by llms.txt, OpenAPI, ai-plugin, MCP, and other signals AI agents use at runtime.",
		Heading:    "Top 100 Agent-Ready Sites",
		Subheading: "The sites AI agents can actually use. Ranked by agentic readiness — publish llms.txt, OpenAPI, ai-plugin, MCP, or a structured API to appear here.",
		OGImage:    "og-top.png",
		Params:     models.SearchParams{Limit: 100},
	})
}

// LLMsTxtPage: /llms-txt-sites — surfaces every site publishing an llms.txt manifest.
func (h *WebHandler) LLMsTxtPage(w http.ResponseWriter, r *http.Request) {
	h.renderCategoryLanding(w, r, categoryLanding{
		Mode:       "llms-txt",
		Path:       "/llms-txt-sites",
		Title:      "llms.txt Directory — Sites Publishing llms.txt Manifests | Not Human Search",
		Desc:       "Every site in our index that publishes an llms.txt manifest — the /llms.txt convention AI agents read at crawl time. Ranked by agentic readiness.",
		Heading:    "llms.txt Directory",
		Subheading: "Every site in our index that ships an llms.txt manifest — ranked by agentic readiness.",
		Params:     models.SearchParams{HasLLMsTxt: true, Limit: 30},
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
		OGImage:    "og-developer-apis.png",
		Params:     models.SearchParams{Category: "developer", Limit: 30},
	})
}

// CategoryDirectoryPages exposes a single canonical URL per remaining
// category. Content and copy are data-driven from CategoryLandingConfig.
var CategoryLandingConfig = map[string]categoryLanding{
	"data": {
		Mode: "data", Path: "/data-apis",
		Title:      "Data API Directory — Agent-Ready Datasets & Data APIs | Not Human Search",
		Desc:       "Browse data APIs and datasets ranked by agentic readiness. Find market data, geospatial, weather, financial, and analytics APIs that AI agents can discover at build time.",
		Heading:    "Data API Directory",
		Subheading: "Every data API and dataset in our index, ranked by agentic readiness.",
		Params:     models.SearchParams{Category: "data", Limit: 30},
	},
	"finance": {
		Mode: "finance", Path: "/finance-apis",
		Title:      "Finance API Directory — Agent-Ready Payment & Banking APIs | Not Human Search",
		Desc:       "Finance APIs AI agents can actually call — payments, banking, market data, crypto. Ranked by agentic readiness score (OpenAPI, llms.txt, MCP).",
		Heading:    "Finance API Directory",
		Subheading: "Every finance API in our index — payments, banking, market data, crypto — ranked by agentic readiness.",
		Params:     models.SearchParams{Category: "finance", Limit: 30},
	},
	"ecommerce": {
		Mode: "ecommerce", Path: "/ecommerce-apis",
		Title:      "E-Commerce API Directory — Agent-Ready Shopping & Commerce APIs | Not Human Search",
		Desc:       "E-commerce APIs that AI agents can shop, search, and check out against. Every entry exposes structured signals AI agents can discover.",
		Heading:    "E-Commerce API Directory",
		Subheading: "Commerce APIs agents can shop against — search, catalog, cart, checkout. Ranked by agentic readiness.",
		Params:     models.SearchParams{Category: "ecommerce", Limit: 30},
	},
	"productivity": {
		Mode: "productivity", Path: "/productivity-apis",
		Title:      "Productivity API Directory — Agent-Ready Task & Workflow APIs | Not Human Search",
		Desc:       "Productivity APIs AI agents can use to manage tasks, calendars, docs, and workflows. All entries expose OpenAPI, llms.txt, or MCP endpoints.",
		Heading:    "Productivity API Directory",
		Subheading: "Productivity tools with agent-ready surfaces — tasks, calendars, docs, workflows.",
		Params:     models.SearchParams{Category: "productivity", Limit: 30},
	},
	"security": {
		Mode: "security", Path: "/security-apis",
		Title:      "Security API Directory — Agent-Ready Authentication & Security APIs | Not Human Search",
		Desc:       "Security APIs AI agents can call — authentication, secrets, WAF, threat intel. Every entry is discoverable at build time via OpenAPI or MCP.",
		Heading:    "Security API Directory",
		Subheading: "Security APIs with agent-ready surfaces — auth, secrets, threat intel, WAF.",
		Params:     models.SearchParams{Category: "security", Limit: 30},
	},
	"communication": {
		Mode: "communication", Path: "/communication-apis",
		Title:      "Communication API Directory — Agent-Ready Messaging & Email APIs | Not Human Search",
		Desc:       "Communication APIs AI agents can use — email, SMS, chat, voice. Ranked by agentic readiness.",
		Heading:    "Communication API Directory",
		Subheading: "Communication APIs with agent-ready surfaces — email, SMS, chat, voice.",
		Params:     models.SearchParams{Category: "communication", Limit: 30},
	},
	"jobs": {
		Mode: "jobs", Path: "/jobs-apis",
		Title:      "Job Board API Directory — Agent-Ready Jobs APIs | Not Human Search",
		Desc:       "Job boards with agent-ready APIs — AI agents can discover, filter, and apply to listings programmatically. Ranked by agentic readiness.",
		Heading:    "Job Board API Directory",
		Subheading: "Job boards with agent-ready APIs — listings, search, and apply endpoints agents can call.",
		Params:     models.SearchParams{Category: "jobs", Limit: 30},
	},
}

// CategoryLandingPage dispatches based on the URL path to CategoryLandingConfig.
// Each path is a dedicated HandleFunc so the Go mux matches exact paths (not a
// prefix) — prevents /productivity-apis-foo from accidentally matching.
func (h *WebHandler) categoryHandler(slug string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg, ok := CategoryLandingConfig[slug]
		if !ok {
			http.NotFound(w, r)
			return
		}
		h.renderCategoryLanding(w, r, cfg)
	}
}

// RegisterCategoryLandings wires every CategoryLandingConfig entry to its path.
// Call from main.go after constructing the WebHandler.
func (h *WebHandler) RegisterCategoryLandings(mux *http.ServeMux) {
	for slug, cfg := range CategoryLandingConfig {
		mux.HandleFunc(cfg.Path, h.categoryHandler(slug))
	}
}

type categoryLanding struct {
	Mode       string
	Path       string
	Title      string
	Desc       string
	Heading    string
	Subheading string
	OGImage    string // filename under /static/img/, no leading slash. Empty = og-default.png.
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

	ogImage := c.OGImage
	if ogImage == "" {
		ogImage = "og-default.png"
	}

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
		"OGImage":    ogImage,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "home.html", data); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "internal error", 500)
	}
}

// TagPage renders /tag/{name} — a programmatic-SEO landing page showing every
// indexed site tagged with {name}. Provides long-tail ranking surface for
// tag-class queries like "agent-ready payment APIs" or "mcp server search".
func (h *WebHandler) TagPage(w http.ResponseWriter, r *http.Request) {
	tag := r.URL.Path[len("/tag/"):]
	// Strip optional trailing slash.
	tag = strings.TrimSuffix(tag, "/")
	// Only accept simple, lowercase slug-style tags.
	if tag == "" {
		http.NotFound(w, r)
		return
	}
	for _, c := range tag {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			http.NotFound(w, r)
			return
		}
	}

	// Human-readable form: "llms-txt" → "llms.txt", "mcp" → "MCP", etc.
	display := tag
	switch tag {
	case "llms-txt":
		display = "llms.txt"
	case "ai-plugin":
		display = "ai-plugin.json"
	case "openapi":
		display = "OpenAPI"
	case "api":
		display = "API"
	case "mcp":
		display = "MCP"
	case "ai":
		display = "AI"
	case "ai-friendly":
		display = "AI-Friendly"
	}

	h.renderCategoryLanding(w, r, categoryLanding{
		Mode:       "tag-" + tag,
		Path:       "/tag/" + tag,
		Title:      fmt.Sprintf("%s — Agent-First Sites Tagged %s | Not Human Search", display, display),
		Desc:       fmt.Sprintf("Every site in our index tagged %s, ranked by agentic readiness. Discover agent-first tools and APIs matching the %s tag.", display, display),
		Heading:    fmt.Sprintf("Sites tagged \"%s\"", display),
		Subheading: fmt.Sprintf("Every site in the index carrying the %s tag, ranked by agentic readiness score.", display),
		Params:     models.SearchParams{Tag: tag, Limit: 30},
	})
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

type ReportData struct {
	Total       int
	HighScore   int
	MediumScore int
	AvgScore    float64
	LlmsTxt    int
	OpenAPI     int
	AIPlugin    int
	API         int
	MCP         int
	SchemaOrg   int
	RobotsAI    int
	LlmsMCP     int
	Categories  []CategoryStat
	TopSites    []TopSite
}

type CategoryStat struct {
	Name     string
	Count    int
	AvgScore float64
}

type TopSite struct {
	Domain string
	Score  int
	Category string
}

func (h *WebHandler) ReportPage(w http.ResponseWriter, r *http.Request) {
	data := ReportData{}
	h.DB.QueryRow(`SELECT count(*), count(*) FILTER (WHERE agentic_score >= 70),
		count(*) FILTER (WHERE agentic_score >= 40 AND agentic_score < 70),
		round(avg(agentic_score)::numeric, 1),
		count(*) FILTER (WHERE has_llms_txt), count(*) FILTER (WHERE has_openapi),
		count(*) FILTER (WHERE has_ai_plugin), count(*) FILTER (WHERE has_structured_api),
		count(*) FILTER (WHERE has_mcp_server), count(*) FILTER (WHERE has_schema_org),
		count(*) FILTER (WHERE has_robots_ai),
		count(*) FILTER (WHERE has_llms_txt AND has_mcp_server)
		FROM sites`).Scan(&data.Total, &data.HighScore, &data.MediumScore, &data.AvgScore,
		&data.LlmsTxt, &data.OpenAPI, &data.AIPlugin, &data.API, &data.MCP, &data.SchemaOrg, &data.RobotsAI,
		&data.LlmsMCP)

	rows, err := h.DB.Query(`SELECT category, count(*), round(avg(agentic_score)::numeric,1)
		FROM sites GROUP BY category ORDER BY count(*) DESC LIMIT 12`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var c CategoryStat
			rows.Scan(&c.Name, &c.Count, &c.AvgScore)
			data.Categories = append(data.Categories, c)
		}
	}

	rows2, err := h.DB.Query(`SELECT domain, agentic_score, category FROM sites ORDER BY agentic_score DESC LIMIT 20`)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var t TopSite
			rows2.Scan(&t.Domain, &t.Score, &t.Category)
			data.TopSites = append(data.TopSites, t)
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	if err := h.tmpl.ExecuteTemplate(w, "report.html", data); err != nil {
		log.Printf("report template error: %v", err)
		http.Error(w, "internal error", 500)
	}
}
