package handlers

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"time"
)

type SEOHandler struct {
	DB      *sql.DB
	BaseURL string
}

func NewSEOHandler(db *sql.DB, baseURL string) *SEOHandler {
	return &SEOHandler{DB: db, BaseURL: baseURL}
}

func (h *SEOHandler) Robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, `# Not Human Search — robots.txt
# We index sites for AI agent discovery. We welcome all crawlers.

User-agent: *
Allow: /

# AI agent crawlers — explicitly welcome
User-agent: GPTBot
Allow: /

User-agent: ChatGPT-User
Allow: /

User-agent: ClaudeBot
Allow: /

User-agent: PerplexityBot
Allow: /

User-agent: Applebot-Extended
Allow: /

User-agent: cohere-ai
Allow: /

Sitemap: %s/sitemap.xml
`, h.BaseURL)
}

func (h *SEOHandler) LLMsTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	var totalSites int
	if err := h.DB.QueryRowContext(r.Context(), "SELECT count(*) FROM sites WHERE crawl_status='success'").Scan(&totalSites); err != nil {
		log.Printf("llms.txt count query: %v", err)
	}

	fmt.Fprintf(w, `# Not Human Search
> The search engine for AI agents. Find websites ranked by agentic readiness.

Not Human Search indexes websites and scores them on how well they serve AI agents.
We check for: llms.txt, ai-plugin.json, OpenAPI specs, structured APIs, robots.txt AI bot rules, Schema.org markup, and MCP server endpoints.

## Stats
- %d sites indexed
- Scores range 0-100 (higher = more agent-ready)

## API
Base URL: %s/api/v1

### Search for agent-ready sites
GET /api/v1/search?q={query}&category={category}&min_score={0-100}&has_api=true&page={n}

### Get site details
GET /api/v1/site/{domain}

### Submit a site for crawling
POST /api/v1/submit
Body: {"url": "https://example.com"}

### Get index stats
GET /api/v1/stats

## Agentic Signals We Check
- llms.txt (/.well-known/llms.txt) — 25 points
- ai-plugin.json (/.well-known/ai-plugin.json) — 20 points
- OpenAPI spec (/openapi.yaml or /openapi.json) — 20 points
- Structured API (/api, /docs) — 15 points
- MCP server — 10 points
- robots.txt AI bot rules — 5 points
- Schema.org markup — 5 points

## Links
- Website: %s
- API: %s/api/v1/search
- Submit: %s/api/v1/submit
`, totalSites, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL)
}

func (h *SEOHandler) AIPluginManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"schema_version":       "v1",
		"name_for_human":       "Not Human Search",
		"name_for_model":       "nothumansearch",
		"description_for_human": "Search engine that finds websites AI agents can actually use, ranked by agentic readiness score.",
		"description_for_model": "Search for websites and APIs that are agent-ready. Returns sites scored 0-100 on agentic readiness based on llms.txt, OpenAPI specs, ai-plugin.json, structured APIs, and MCP server support. Use to find services an AI agent can interact with programmatically.",
		"auth":                 map[string]string{"type": "none"},
		"api": map[string]string{
			"type": "openapi",
			"url":  h.BaseURL + "/openapi.yaml",
		},
		"logo_url":        h.BaseURL + "/static/img/logo.svg",
		"contact_email":   "hello@nothumansearch.com",
		"legal_info_url":  h.BaseURL + "/about",
	})
}

func (h *SEOHandler) OpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/yaml")
	fmt.Fprintf(w, `openapi: "3.0.3"
info:
  title: Not Human Search API
  description: Search engine for AI agents. Find websites ranked by agentic readiness.
  version: "1.0.0"
  contact:
    email: hello@nothumansearch.com
servers:
  - url: %s/api/v1
paths:
  /search:
    get:
      summary: Search for agent-ready sites
      operationId: searchSites
      parameters:
        - name: q
          in: query
          schema: { type: string }
          description: Search query (matches name, description, domain)
        - name: category
          in: query
          schema: { type: string, enum: [ai-tools, developer, data, jobs, finance, ecommerce, health, education, security, other] }
        - name: min_score
          in: query
          schema: { type: integer, minimum: 0, maximum: 100 }
          description: Minimum agentic readiness score
        - name: has_api
          in: query
          schema: { type: boolean }
          description: Filter to sites with structured APIs
        - name: page
          in: query
          schema: { type: integer, default: 1 }
      responses:
        "200":
          description: Search results
          content:
            application/json:
              schema:
                type: object
                properties:
                  results: { type: array, items: { $ref: "#/components/schemas/Site" } }
                  total: { type: integer }
                  page: { type: integer }
                  has_next: { type: boolean }
  /site/{domain}:
    get:
      summary: Get detailed agentic readiness report for a site
      operationId: getSite
      parameters:
        - name: domain
          in: path
          required: true
          schema: { type: string }
      responses:
        "200":
          description: Site details
          content:
            application/json:
              schema: { $ref: "#/components/schemas/Site" }
  /submit:
    post:
      summary: Submit a site for crawling
      operationId: submitSite
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [url]
              properties:
                url: { type: string, format: uri }
      responses:
        "201":
          description: Submitted for crawling
  /stats:
    get:
      summary: Get index statistics
      operationId: getStats
      responses:
        "200":
          description: Index stats
components:
  schemas:
    Site:
      type: object
      properties:
        id: { type: string, format: uuid }
        domain: { type: string }
        url: { type: string, format: uri }
        name: { type: string }
        description: { type: string }
        has_llms_txt: { type: boolean }
        has_ai_plugin: { type: boolean }
        has_openapi: { type: boolean }
        has_robots_ai: { type: boolean }
        has_structured_api: { type: boolean }
        has_mcp_server: { type: boolean }
        has_schema_org: { type: boolean }
        agentic_score: { type: integer, minimum: 0, maximum: 100 }
        category: { type: string }
        tags: { type: array, items: { type: string } }
        is_verified: { type: boolean }
        is_featured: { type: boolean }
`, h.BaseURL)
}

type sitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

type sitemap struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

func (h *SEOHandler) Sitemap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")

	sm := sitemap{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9"}

	// Static pages
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/", ChangeFreq: "daily", Priority: "1.0"})

	// Site pages
	rows, err := h.DB.QueryContext(r.Context(), "SELECT domain, updated_at FROM sites WHERE crawl_status='success' ORDER BY agentic_score DESC LIMIT 49999")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var domain string
			var updated time.Time
			if err := rows.Scan(&domain, &updated); err != nil {
				log.Printf("sitemap scan: %v", err)
				continue
			}
			sm.URLs = append(sm.URLs, sitemapURL{
				Loc:        h.BaseURL + "/site/" + domain,
				LastMod:    updated.Format("2006-01-02"),
				ChangeFreq: "weekly",
				Priority:   "0.8",
			})
		}
		if err := rows.Err(); err != nil {
			log.Printf("sitemap rows: %v", err)
		}
	} else {
		log.Printf("sitemap query: %v", err)
	}

	w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(sm); err != nil {
		log.Printf("sitemap encode: %v", err)
	}
}
