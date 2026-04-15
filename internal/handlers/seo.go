package handlers

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
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
# RSS feed: %s/feed.xml (new agent-ready additions)
`, h.BaseURL, h.BaseURL)
}

func (h *SEOHandler) LLMsTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	var totalSites int
	if err := h.DB.QueryRowContext(r.Context(), "SELECT count(*) FROM sites WHERE "+models.AgentFirstFilter).Scan(&totalSites); err != nil {
		log.Printf("llms.txt count query: %v", err)
	}

	fmt.Fprintf(w, `# Not Human Search
> The Google for AI agents. Find any service, API, or tool — ranked by agentic readiness.

Not Human Search is an agentic-first search engine. When your AI agent needs to discover a payment API, a job board, an ecommerce platform, or any web service, NHS returns results ranked by how well each site serves non-human users.

We index %d+ sites and score them 0-100 based on 7 agentic signals: llms.txt, ai-plugin.json, OpenAPI specs, structured APIs, MCP servers, robots.txt AI rules, and Schema.org markup.

## MCP Server (preferred for agents)
Not Human Search is itself an MCP server. Wire it into your agent once and get live agentic-web search as a first-class tool.

Endpoint: %s/mcp
Transport: streamable-http
Tools: search_agents, get_site_details, get_stats, submit_site, register_monitor

Claude Code setup:
  claude mcp add --transport http nothumansearch %s/mcp

## Quick Start — Search the Agentic Web
GET %s/api/v1/search?q=payment+API
GET %s/api/v1/search?q=AI+jobs
GET %s/api/v1/search?q=ecommerce+api
GET %s/api/v1/search?q=weather+data
GET %s/api/v1/search?q=authentication

## API Reference
Base URL: %s/api/v1

### Search
GET /search?q={query}&category={cat}&tag={tag}&min_score={0-100}&has_api=true&has_mcp=true&has_openapi=true&has_llms_txt=true&page={n}
Returns: {results: [{domain, name, description, agentic_score, category, tags, signals...}], total, page, has_next}

### Site Details
GET /site/{domain}
Returns: full site profile with llms.txt content, OpenAPI summary, all signals

### Submit a Site
POST /submit  Body: {"url": "https://example.com"}
We crawl immediately and add it to the index.

### On-Demand Check (CI / pre-deploy)
POST /check  Body: {"url": "https://example.com"}
Returns live agentic readiness score without waiting for the crawl queue.
Free tier: 10 checks/hour per IP. Great for CI pipelines that fail the build
when a site's agent signals regress.

### Stats
GET /stats

### Categories
GET /categories
Returns: {categories: [{name, count}]} — all 12 buckets with live counts.

### Monitor a Site
POST /monitor/register  Body: {"email": "you@x.com", "domain": "site.com"}
Email alert when a site's agentic readiness drops. Returns an unsubscribe URL.
Free tier: multiple monitors per email allowed, one per domain.

## Categories
ai-tools, developer, data, finance, ecommerce, jobs, security, health, education, communication, productivity.

## Scoring (0-100)
- llms.txt: 25 pts
- ai-plugin.json: 20 pts
- OpenAPI spec: 20 pts
- Structured API: 15 pts
- MCP server: 10 pts
- robots.txt AI rules: 5 pts
- Schema.org: 5 pts

## Make Your Site Agent-Ready
Step-by-step recipes for each of the 7 signals — copy-paste examples for llms.txt, ai-plugin.json, OpenAPI, MCP, and more.

Guide: %s/guide
Live scorer: %s/score

## Links
- Search: %s/api/v1/search?q=
- MCP Server Directory: %s/mcp-servers
- Full Index: %s/llms-full.txt
- OpenAPI: %s/openapi.yaml
- Plugin: %s/.well-known/ai-plugin.json
`, totalSites, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL)
}

func (h *SEOHandler) LLMsFullTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT domain, name, description, agentic_score, category,
			has_llms_txt, has_ai_plugin, has_openapi, has_structured_api, has_mcp_server
		FROM sites WHERE `+models.AgentFirstFilter+`
		ORDER BY agentic_score DESC`)
	if err != nil {
		http.Error(w, "internal error", 500)
		return
	}
	defer rows.Close()

	fmt.Fprintf(w, "# Not Human Search — Full Index\n")
	fmt.Fprintf(w, "> %s/llms-full.txt\n", h.BaseURL)
	fmt.Fprintf(w, "> Complete directory of agent-ready tools, ranked by agentic readiness.\n\n")

	for rows.Next() {
		var domain, name, desc, category string
		var score int
		var llms, plugin, openapi, api, mcp bool
		if err := rows.Scan(&domain, &name, &desc, &score, &category, &llms, &plugin, &openapi, &api, &mcp); err != nil {
			continue
		}
		var signals []string
		if llms {
			signals = append(signals, "llms.txt")
		}
		if plugin {
			signals = append(signals, "ai-plugin")
		}
		if openapi {
			signals = append(signals, "openapi")
		}
		if api {
			signals = append(signals, "api")
		}
		if mcp {
			signals = append(signals, "mcp")
		}
		fmt.Fprintf(w, "## %s [%d/100] (%s)\n", domain, score, category)
		fmt.Fprintf(w, "%s\n", name)
		if desc != "" {
			fmt.Fprintf(w, "%s\n", desc)
		}
		fmt.Fprintf(w, "Signals: %s\n", strings.Join(signals, ", "))
		fmt.Fprintf(w, "Details: %s/api/v1/site/%s\n\n", h.BaseURL, domain)
	}
}

func (h *SEOHandler) MCPManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":        "nothumansearch",
		"version":     "1.0.0",
		"description": "Search engine for AI agents. Find websites and APIs ranked by agentic readiness score (0-100). Query by keyword, category, or minimum score.",
		"mcp_server": map[string]interface{}{
			"transport": "streamable-http",
			"endpoint":  h.BaseURL + "/mcp",
			"setup":     "claude mcp add --transport http nothumansearch " + h.BaseURL + "/mcp",
		},
		"tools": []map[string]interface{}{
			{
				"name":        "search_agents",
				"description": "Search for agent-ready websites and APIs by keyword, category, or minimum agentic score.",
				"endpoint":    h.BaseURL + "/api/v1/search",
				"method":      "GET",
				"parameters": map[string]interface{}{
					"q":            map[string]string{"type": "string", "description": "Search query"},
					"category":     map[string]string{"type": "string", "description": "Filter by category (ai-tools, developer, data, finance, ecommerce, jobs, security, health, education, communication, productivity)"},
					"tag":          map[string]string{"type": "string", "description": "Filter by exact tag (e.g. mcp, openapi, llms-txt, payment, search)"},
					"min_score":    map[string]string{"type": "integer", "description": "Minimum agentic readiness score (0-100)"},
					"has_api":      map[string]string{"type": "boolean", "description": "Filter to sites with structured APIs"},
					"has_mcp":      map[string]string{"type": "boolean", "description": "Filter to sites with an MCP server"},
					"has_openapi":  map[string]string{"type": "boolean", "description": "Filter to sites with an OpenAPI spec"},
					"has_llms_txt": map[string]string{"type": "boolean", "description": "Filter to sites publishing llms.txt"},
					"page":         map[string]string{"type": "integer", "description": "Page number (default 1)"},
				},
			},
			{
				"name":        "get_site_details",
				"description": "Get detailed agentic readiness report for a specific domain.",
				"endpoint":    h.BaseURL + "/api/v1/site/{domain}",
				"method":      "GET",
			},
			{
				"name":        "submit_site",
				"description": "Submit a new site for crawling and indexing.",
				"endpoint":    h.BaseURL + "/api/v1/submit",
				"method":      "POST",
			},
			{
				"name":        "get_stats",
				"description": "Get index statistics: total sites, average score, top category.",
				"endpoint":    h.BaseURL + "/api/v1/stats",
				"method":      "GET",
			},
			{
				"name":        "register_monitor",
				"description": "Subscribe an email to get alerted when the indicated domain's agentic readiness score drops.",
				"endpoint":    h.BaseURL + "/api/v1/monitor/register",
				"method":      "POST",
			},
		},
	})
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
		"contact_email":   "hello@nothumansearch.ai",
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
    email: hello@nothumansearch.ai
servers:
  - url: %s/api/v1
paths:
  /:
    get:
      summary: API index — list of endpoints and base URLs
      operationId: getIndex
      responses:
        "200":
          description: API index document
          content:
            application/json:
              schema:
                type: object
                properties:
                  base_url:           { type: string }
                  openapi_spec:       { type: string }
                  ai_plugin_manifest: { type: string }
                  mcp_endpoint:       { type: string }
                  endpoints:          { type: object, additionalProperties: { type: string } }
  /categories:
    get:
      summary: Get all category buckets and their counts
      operationId: listCategories
      responses:
        "200":
          description: Category counts across the index
          content:
            application/json:
              schema:
                type: object
                properties:
                  categories:
                    type: array
                    items:
                      type: object
                      properties:
                        name:  { type: string }
                        count: { type: integer }
  /monitor/register:
    post:
      summary: Register an email to monitor a site's agentic readiness score
      operationId: registerMonitor
      description: Sends an alert via email when the indicated domain's score drops. Returns an unsubscribe URL.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [email, domain]
              properties:
                email:  { type: string, format: email }
                domain: { type: string, description: "Domain to monitor (no scheme)" }
      responses:
        "201":
          description: Monitor registered
          content:
            application/json:
              schema:
                type: object
                properties:
                  ok:              { type: boolean }
                  domain:          { type: string }
                  unsubscribe_url: { type: string, format: uri }
        "400":
          description: Invalid email or domain
        "429":
          description: Too many monitors for this email
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
          schema: { type: string, enum: [ai-tools, developer, data, jobs, finance, ecommerce, health, education, security, communication, productivity, other] }
        - name: tag
          in: query
          schema: { type: string }
          description: Exact tag match (e.g. mcp, openapi, payment, search). See /sitemap.xml for indexed tags.
        - name: min_score
          in: query
          schema: { type: integer, minimum: 0, maximum: 100 }
          description: Minimum agentic readiness score
        - name: has_api
          in: query
          schema: { type: boolean }
          description: Filter to sites with structured APIs
        - name: has_mcp
          in: query
          schema: { type: boolean }
          description: Filter to sites with a Model Context Protocol server
        - name: has_openapi
          in: query
          schema: { type: boolean }
          description: Filter to sites that publish an OpenAPI spec
        - name: has_llms_txt
          in: query
          schema: { type: boolean }
          description: Filter to sites that publish llms.txt
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
  /check:
    post:
      summary: On-demand agentic readiness check (live crawl)
      operationId: checkSite
      description: |
        Crawls the target URL on demand and returns its 7-signal agentic
        readiness score. Ideal for CI pipelines that should fail when an
        agent-facing site regresses. Free tier: 10 checks/hour per IP.
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [url]
              properties:
                url: { type: string, format: uri }
      responses:
        "200":
          description: Score + 7 signals
        "429":
          description: Rate limit exceeded
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
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/mcp-servers", ChangeFreq: "daily", Priority: "0.9"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/ai-tools", ChangeFreq: "daily", Priority: "0.9"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/developer-apis", ChangeFreq: "daily", Priority: "0.9"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/openapi-apis", ChangeFreq: "daily", Priority: "0.9"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/llms-txt-sites", ChangeFreq: "daily", Priority: "0.9"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/score", ChangeFreq: "weekly", Priority: "0.9"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/data-apis", ChangeFreq: "daily", Priority: "0.8"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/finance-apis", ChangeFreq: "daily", Priority: "0.8"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/ecommerce-apis", ChangeFreq: "daily", Priority: "0.8"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/productivity-apis", ChangeFreq: "daily", Priority: "0.8"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/security-apis", ChangeFreq: "daily", Priority: "0.8"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/communication-apis", ChangeFreq: "daily", Priority: "0.8"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/jobs-apis", ChangeFreq: "daily", Priority: "0.8"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/about", ChangeFreq: "weekly", Priority: "0.5"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/guide", ChangeFreq: "weekly", Priority: "0.9"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/score", ChangeFreq: "weekly", Priority: "0.7"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/feed.xml", ChangeFreq: "hourly", Priority: "0.7"})
	for _, cat := range []string{"ai-tools", "developer", "finance", "data", "ecommerce", "productivity", "security", "communication", "jobs"} {
		sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/feed/" + cat + ".xml", ChangeFreq: "daily", Priority: "0.5"})
	}

	// Tag landing pages — one URL per distinct tag with at least 2 agent-first
	// sites carrying it. Long-tail SEO surface; keeps low-signal one-off tags
	// out of the index.
	tagRows, terr := h.DB.QueryContext(r.Context(),
		`SELECT tag, COUNT(*) AS n
		   FROM (SELECT unnest(tags) AS tag FROM sites
		         WHERE crawl_status='success'
		           AND (has_structured_api = true OR has_openapi = true OR has_ai_plugin = true OR has_mcp_server = true)) t
		  WHERE tag ~ '^[a-z0-9-]+$'
		  GROUP BY tag HAVING COUNT(*) >= 2
		  ORDER BY n DESC LIMIT 200`)
	if terr == nil {
		defer tagRows.Close()
		for tagRows.Next() {
			var tag string
			var n int
			if err := tagRows.Scan(&tag, &n); err != nil {
				continue
			}
			sm.URLs = append(sm.URLs, sitemapURL{
				Loc:        h.BaseURL + "/tag/" + tag,
				ChangeFreq: "weekly",
				Priority:   "0.6",
			})
		}
	} else {
		log.Printf("sitemap tags: %v", terr)
	}

	// Site pages
	rows, err := h.DB.QueryContext(r.Context(), "SELECT domain, updated_at FROM sites WHERE crawl_status='success' AND (has_structured_api = true OR has_llms_txt = true OR has_openapi = true OR has_ai_plugin = true OR has_mcp_server = true) ORDER BY agentic_score DESC LIMIT 49999")
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

// RSS feed of most recently added agent-first sites. Syndication surface —
// aggregators/readers can subscribe and repost, generating backlinks.
type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Atom    string     `xml:"xmlns:atom,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title         string     `xml:"title"`
	Link          string     `xml:"link"`
	AtomLink      atomLink   `xml:"atom:link"`
	Description   string     `xml:"description"`
	Language      string     `xml:"language"`
	LastBuildDate string     `xml:"lastBuildDate"`
	Items         []rssItem  `xml:"item"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	Category    string `xml:"category"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description"`
}

func (h *SEOHandler) Feed(w http.ResponseWriter, r *http.Request) {
	// Per-category feeds at /feed/{slug}.xml (route registered separately).
	// Empty slug = master feed across all categories.
	slug := strings.TrimPrefix(r.URL.Path, "/feed/")
	slug = strings.TrimSuffix(slug, ".xml")
	if r.URL.Path == "/feed.xml" || r.URL.Path == "/rss.xml" {
		slug = ""
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")

	title := "Not Human Search — New Agent-Ready Sites"
	desc := "Newly indexed agent-first sites (score ≥25) — ranked by agentic readiness. Updated continuously."
	selfHref := h.BaseURL + "/feed.xml"
	if slug != "" {
		title = fmt.Sprintf("Not Human Search — New %s Sites", strings.Title(slug))
		desc = fmt.Sprintf("Newly indexed agent-first sites in category: %s. Score ≥25.", slug)
		selfHref = h.BaseURL + "/feed/" + slug + ".xml"
	}

	feed := rssFeed{
		Version: "2.0",
		Atom:    "http://www.w3.org/2005/Atom",
		Channel: rssChannel{
			Title:         title,
			Link:          h.BaseURL + "/",
			AtomLink:      atomLink{Href: selfHref, Rel: "self", Type: "application/rss+xml"},
			Description:   desc,
			Language:      "en-us",
			LastBuildDate: time.Now().UTC().Format(time.RFC1123Z),
		},
	}

	// MinScore=25 filters out schema-only/robots-only noise; syndication
	// quality matters more than feed length.
	args := []interface{}{}
	query := `SELECT domain, name, description, category, agentic_score, created_at
		FROM sites WHERE ` + models.AgentFirstFilter + ` AND agentic_score >= 25`
	if slug != "" {
		query += ` AND category = $1`
		args = append(args, slug)
	}
	query += ` ORDER BY created_at DESC LIMIT 50`

	rows, err := h.DB.QueryContext(r.Context(), query, args...)
	if err != nil {
		log.Printf("feed query: %v", err)
		http.Error(w, "feed error", 500)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var domain, name, itemDesc, category string
		var score int
		var created time.Time
		if err := rows.Scan(&domain, &name, &itemDesc, &category, &score, &created); err != nil {
			continue
		}
		if name == "" {
			name = domain
		}
		itemTitle := fmt.Sprintf("%s — score %d/100 (%s)", name, score, category)
		body := itemDesc
		if body == "" {
			body = fmt.Sprintf("Agentic readiness report for %s. Category: %s. Score: %d/100.", domain, category, score)
		}
		feed.Channel.Items = append(feed.Channel.Items, rssItem{
			Title:       itemTitle,
			Link:        h.BaseURL + "/site/" + domain,
			GUID:        h.BaseURL + "/site/" + domain,
			Category:    category,
			PubDate:     created.UTC().Format(time.RFC1123Z),
			Description: body,
		})
	}

	w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(feed); err != nil {
		log.Printf("feed encode: %v", err)
	}
}
