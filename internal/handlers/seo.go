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

var publicSearchCategories = []string{
	"ai-tools",
	"developer",
	"data",
	"finance",
	"ecommerce",
	"jobs",
	"security",
	"health",
	"education",
	"communication",
	"productivity",
	"news",
}

var auditSearchCategories = []string{"other", "spam"}

func publicSearchCategoryCSV() string {
	return strings.Join(publicSearchCategories, ", ")
}

func searchCategoryDescription() string {
	return "Filter by public category (" + publicSearchCategoryCSV() + "). Audit-only buckets may appear in /api/v1/categories as other or spam, but are not promoted as discovery inventory."
}

func searchCategoryOpenAPIEnum() string {
	cats := append([]string{}, publicSearchCategories...)
	cats = append(cats, auditSearchCategories...)
	return strings.Join(cats, ", ")
}

func NewSEOHandler(db *sql.DB, baseURL string) *SEOHandler {
	return &SEOHandler{DB: db, BaseURL: baseURL}
}

func (h *SEOHandler) Robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	fmt.Fprintf(w, `# Not Human Search — robots.txt
# We index sites for AI agent discovery. We welcome all crawlers.

User-agent: *
Allow: /

# AI agent crawlers — explicitly welcome (matches 8bc coverage, 2026-04-15)
User-agent: GPTBot
Allow: /

User-agent: ChatGPT-User
Allow: /

User-agent: OAI-SearchBot
Allow: /

User-agent: ClaudeBot
Allow: /

User-agent: Claude-Web
Allow: /

User-agent: anthropic-ai
Allow: /

User-agent: PerplexityBot
Allow: /

User-agent: Google-Extended
Allow: /

User-agent: Applebot-Extended
Allow: /

User-agent: Meta-ExternalAgent
Allow: /

User-agent: FacebookBot
Allow: /

User-agent: CCBot
Allow: /

User-agent: Bytespider
Allow: /

User-agent: Amazonbot
Allow: /

User-agent: cohere-ai
Allow: /

User-agent: Diffbot
Allow: /

User-agent: YouBot
Allow: /

User-agent: DuckAssistBot
Allow: /

User-agent: PetalBot
Allow: /

User-agent: FirecrawlAgent
Allow: /

Sitemap: %s/sitemap.xml
# RSS feed: %s/feed.xml (new agent-ready additions)

# AI agent discovery
# llms.txt: %s/llms.txt
# llms-full.txt: %s/llms-full.txt
# OpenAPI spec: %s/openapi.yaml
# AI plugin manifest: %s/.well-known/ai-plugin.json
# MCP manifest: %s/.well-known/mcp.json
# MCP endpoint (streamable-http JSON-RPC): %s/mcp
# Registry auth proof: %s/.well-known/mcp-registry-auth
# Security contact: %s/.well-known/security.txt
`, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL)
}

func (h *SEOHandler) LLMsTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Cache-Control", "public, max-age=300")

	var totalSites int
	if h.DB != nil {
		if err := h.DB.QueryRowContext(r.Context(), "SELECT count(*) FROM sites WHERE "+models.AgentFirstFilter).Scan(&totalSites); err != nil {
			log.Printf("llms.txt count query: %v", err)
		}
	}

	fmt.Fprintf(w, `# Not Human Search
> The Google for AI agents. Find any service, API, or tool — ranked by agentic readiness.

Not Human Search is an agentic-first search engine. When your AI agent needs to discover a payment API, a job board, an ecommerce platform, or any web service, NHS returns results ranked by how well each site serves non-human users.

Site owners: score your own site's agent-readiness (0-100, 7 signals) and monitor it over time, so AI agents can actually find and use you.

We index %d+ sites and score them 0-100 based on 7 agentic signals: llms.txt, ai-plugin.json, OpenAPI specs, structured APIs, MCP servers, robots.txt AI rules, and Schema.org markup.

## MCP Server (preferred for agents)
Not Human Search is itself an MCP server. Wire it into your agent once and get live agentic-web search as a first-class tool.

Endpoint: %s/mcp
Transport: streamable-http
Tools (13): search_agents, get_site_details, get_stats, list_categories, get_top_sites, submit_site, register_monitor, verify_mcp, find_mcp_servers, recent_additions, check_url, record_action_interest, prepare_provider_action

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
Returns: {results: [...unchanged neutral organic results...], paid_offers: [...separate disclosed provider-funded actions...], total, page, has_next, search_id}

### Optional Provider-Funded Actions
Search and canonical provider links remain free. An active provider offer can
appear only in the separate paid_offers collection beside a site already returned
organically. Money never changes organic membership, score, order, or total.

POST /api/v1/action-tickets
Creates a signed action URL after the caller supplies search_id, offer_id, one
controlled demand topic already on the receipt, and the exact versioned principal-
authorization attestation. Creating a ticket charges neither party.

Consent wording: /privacy#consent-v1
Provider setup: /providers

### Caller-Attested Action Interest (No Provider Contact)
POST /api/v1/action-interests
After an organic search, a caller can record that its human or company principal
currently wants one controlled next step with an exact returned domain: quote,
trial, demo, booking, application, signup, or purchase. This private receipt
expires with the source search, no later than 30 days after that search, and is
used only for aggregate Stage 1 demand. It does not contact the
provider, create a provider action ticket or charge, affect rank or score, or
count as commercial proof. Exact wording: /privacy#action-interest-v1

### Site Details
GET /site/{domain}
Returns: full site profile with llms.txt content, OpenAPI summary, all signals

### Submit a Site
POST /submit  Body: {"url": "https://example.com"}
We crawl immediately and add it to the index.

### On-Demand Check (CI / pre-deploy)
POST /api/v1/check  Body: {"url": "https://example.com"}
Returns live agentic readiness score without waiting for the crawl queue.
Free tier: 10 checks/hour per IP. Great for CI pipelines that fail the build
when a site's agent signals regress.

### Optional Priority API Key
Search, site details, and organic results are free without a key. An active key
only raises hourly safety ceilings; it never changes results, rank, or score.

GET /api/v1/api-keys/subscribe
Returns the optional priority-throughput plan and the machine contract for creating a Stripe Checkout session.

POST /api/v1/api-keys/subscribe  Body: {"email": "you@example.com", "plan": "unlimited"}
Plan: $9.99/month for 50,000 priority-throughput calls. After that allocation,
requests continue at free safety limits instead of returning a payment error.
Returns: {checkout_url, plan, monthly_limit, amount_cents, activation_url}

### Stats
GET /stats

### Top Sites
GET /top?has_mcp=true&category=ai-tools&limit=25
Returns the highest-scored sites in the index, sorted by agentic_score DESC.
Designed for embedding / mirroring: stable URL, cached 5min, no auth. Max 100 results.

### Categories
GET /categories
Returns: {categories: [{name, count}]} — live public categories plus audit-only buckets when present.

### Monitor a Site
POST /monitor/register  Body: {"email": "you@x.com", "domain": "site.com"}
Email alert when a site's agentic readiness drops. Returns an unsubscribe URL.
Free tier: multiple monitors per email allowed, one per domain.

## Categories
Public categories: %s.
Audit-only buckets: other, spam. These are exposed for transparency and filtering, not promoted as agent-ready inventory.

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
Report: %s/report

## Links
- Search: %s/api/v1/search?q=
- MCP Server Directory: %s/mcp-servers
- Full Index: %s/llms-full.txt
- OpenAPI: %s/openapi.yaml
- Plugin: %s/.well-known/ai-plugin.json

## Developer Resources
- One-line install (Claude Code / Cursor / Cline / Continue):
    curl -fsSL https://nothumansearch.ai/install | sh
- Verify any MCP server in 3 curls: https://gist.github.com/unitedideas/ce709323717b95eb56f7be7392a0a557
- Q2 2026 agent-ready sites curation (120 categorized): https://gist.github.com/unitedideas/c60bb35943ef609f99123bdfae146e55
- NHS Score Check GitHub Action (fail CI on score drop): https://github.com/unitedideas/nhs-score-check-action
- Q2 2026 MCP Ecosystem Health data + methodology: https://8bitconcepts.com/research/q2-2026-mcp-ecosystem-health.html
`, totalSites, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, publicSearchCategoryCSV(), h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL, h.BaseURL)
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
	w.Header().Set("Cache-Control", "public, max-age=86400")
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
					"category":     map[string]string{"type": "string", "description": searchCategoryDescription()},
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
			{
				"name":        "verify_mcp",
				"description": "Live JSON-RPC probe of any URL to confirm it's a spec-compliant MCP server. Returns {verified, endpoint, note}.",
			},
			{
				"name":        "check_url",
				"description": "On-demand agentic-readiness score for any URL. Runs the 7-signal crawler live and returns score + per-signal breakdown. Like submit_site but without the submissions-table side-effect — for verify-before-use workflows.",
			},
			{
				"name":        "list_categories",
				"description": "List every index category with site counts and average agentic scores. Use before searching to discover where density lives.",
			},
			{
				"name":        "get_top_sites",
				"description": "Highest-scored agent-ready sites overall or in a specific category.",
			},
			{
				"name":        "find_mcp_servers",
				"description": "Convenience wrapper over search — returns only sites that expose an MCP server, ranked by agentic score. Pairs with verify_mcp for probe-before-use.",
			},
			{
				"name":        "recent_additions",
				"description": "Agent-first sites added to the index within the last N days (default 7, max 90). For tracking ecosystem momentum and weekly digests.",
			},
			{
				"name":        "record_action_interest",
				"description": "Record caller-attested principal interest in one controlled next step for an exact returned organic domain. This private demand receipt expires with the source search, no later than 30 days after that search; it does not contact the provider, create a ticket or charge, affect rank, or count as commercial proof.",
				"endpoint":    h.BaseURL + "/api/v1/action-interests",
				"method":      "POST",
			},
			{
				"name":        "prepare_provider_action",
				"description": "Create a signed, authorization-attested action ticket for a separately disclosed provider-funded offer. Requires an organic search receipt; accepts controlled fields only. Exact wording is published at /privacy#consent-v1.",
				"endpoint":    h.BaseURL + "/api/v1/action-tickets",
				"method":      "POST",
			},
		},
	})
}

func (h *SEOHandler) GlamaManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"$schema": "https://glama.ai/mcp/schemas/connector.json",
		"maintainers": []map[string]string{
			{"email": "hello@8bitconcepts.com"},
		},
	})
}

func (h *SEOHandler) AIPluginManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"schema_version":        "v1",
		"name_for_human":        "Not Human Search",
		"name_for_model":        "nothumansearch",
		"description_for_human": "Search engine that finds websites AI agents can actually use, ranked by agentic readiness score.",
		"description_for_model": "Search for websites and APIs that are agent-ready. Returns sites scored 0-100 on agentic readiness based on 7 signals (llms.txt, OpenAPI, ai-plugin.json, structured APIs, MCP server, robots.txt AI rules, Schema.org). Key REST endpoints: GET /api/v1/search (with filters has_mcp, has_openapi, has_llms_txt), GET /api/v1/top (top-scored sites, filterable by signal), GET /api/v1/site/{domain}, GET /api/v1/verify-mcp?url=, and POST /api/v1/check. Organic search is free and neutral. Caller-attested principal interest can be recorded without provider contact, payment, or rank effects. Separately disclosed provider-funded actions may appear only beside an already-returned organic result. For richer capabilities connect via MCP at /mcp — 13 tools including record_action_interest and prepare_provider_action.",
		"auth":                  map[string]string{"type": "none"},
		"api": map[string]string{
			"type": "openapi",
			"url":  h.BaseURL + "/openapi.yaml",
		},
		"logo_url":          h.BaseURL + "/static/img/logo.svg",
		"contact_email":     "hello@nothumansearch.ai",
		"legal_info_url":    h.BaseURL + "/about",
		"refund_policy_url": h.BaseURL + "/about#refund",
	})
}

// SecurityTxt serves RFC 9116 /.well-known/security.txt — the standard file
// security researchers check before reporting vulnerabilities. Required by
// some compliance frameworks + scanned by automated security bots.
// Expires field MUST be ≤ 1 year in the future per RFC 9116 §2.5.5.
func (h *SEOHandler) SecurityTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	// Expires one year from now, updated on each deploy.
	expires := time.Now().UTC().AddDate(1, 0, 0).Format(time.RFC3339)
	fmt.Fprintf(w, `Contact: mailto:security@nothumansearch.ai
Contact: https://github.com/unitedideas/nothumansearch/issues
Expires: %s
Preferred-Languages: en
Canonical: https://nothumansearch.ai/.well-known/security.txt
Policy: https://github.com/unitedideas/nothumansearch/blob/main/SECURITY.md

# Not Human Search — security contact
# Report vulnerabilities privately via the Contact addresses above.
# Please do not test authenticated flows against production — our admin
# endpoints rate-limit + log per-IP; accidental DoS triggers paging.
# Responsible-disclosure window: 30 days before public write-up.
`, expires)
}

func (h *SEOHandler) OpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/yaml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
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
  /catalog:
    get:
      summary: Agent-readable commerce catalog
      operationId: getCommerceCatalog
      responses:
        "200":
          description: Catalog for score-fix service and API subscription plans
  /quote:
    post:
      summary: Create a deterministic quote
      operationId: createCommerceQuote
      requestBody:
        required: false
        content:
          application/json:
            schema:
              type: object
              properties:
                product_id: { type: string, enum: [nhs_geo_fix_my_score, nhs_api_unlimited] }
                plan:       { type: string, enum: [unlimited] }
      responses:
        "200":
          description: Quote with amount, total, currency, and required checkout metadata
  /checkout:
    post:
      summary: Create a Stripe Checkout URL for a GEO uplift order
      operationId: createCommerceCheckout
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [host, email]
              properties:
                product_id:   { type: string, default: nhs_geo_fix_my_score }
                payment_mode: { type: string, enum: [stripe_checkout, stripe] }
                host:         { type: string }
                email:        { type: string, format: email }
                repo_url:     { type: string }
                notes:        { type: string }
      responses:
        "201":
          description: Stripe Checkout URL
        "501":
          description: Requested payment mode is not supported
  /api-keys/subscribe:
    get:
      summary: List the optional priority-throughput API key plan and checkout contract
      operationId: getAPIKeySubscriptionPlans
      responses:
        "200":
          description: Priority throughput only; baseline discovery and organic results remain free
    post:
      summary: Create a Stripe Checkout session for a paid API key subscription
      operationId: createAPIKeySubscriptionCheckout
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [email, plan]
              properties:
                email: { type: string, format: email }
                plan:  { type: string, enum: [unlimited], default: unlimited }
      responses:
        "200":
          description: Stripe Checkout URL and activation URL
        "503":
          description: Stripe is not configured
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
          schema: { type: string, enum: [%s] }
          description: Public categories plus audit-only buckets other and spam. Do not treat audit-only buckets as promoted discovery inventory.
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
                  access: { type: string, enum: [free] }
                  receipt_recorded: { type: boolean, description: True only when the query-free receipt transaction committed }
                  search_id: { type: string, description: Query-free receipt for an optional detail-selection request }
                  results: { type: array, items: { $ref: "#/components/schemas/Site" } }
                  paid_offers_available: { type: boolean, description: True only when a committed search receipt has separately disclosed provider-funded actions for exact returned organic sites }
                  paid_offers:
                    type: array
                    description: Separate optional actions; never included in results, score, total, or organic ordering
                    items: { $ref: "#/components/schemas/PublicProviderOffer" }
                  action_interest:
                    type: object
                    description: Provider-independent way to record caller-attested principal interest against an exact returned organic result
                    properties:
                      available: { type: boolean }
                      endpoint: { type: string, format: uri }
                      confirmation_version: { type: string, enum: [nhs-action-interest-v1] }
                      provider_contacted: { type: boolean, enum: [false] }
                      commercial_proof: { type: boolean, enum: [false] }
                      organic_rank_affected: { type: boolean, enum: [false] }
                  total: { type: integer }
                  page: { type: integer }
                  per_page: { type: integer }
                  has_next: { type: boolean }
        "429":
          description: Temporary search safety limit exceeded; free access resumes after the reset
  /provider/claims:
    get:
      summary: List provider claims and DNS ownership-check status for the signed-in account
      operationId: listProviderClaims
      description: Each claim exposes the last successful ownership check, next scheduled check, and consecutive failure count without exposing the stored token hash or raw DNS answers.
      security: [{ SessionCookie: [] }]
      responses:
        "200":
          description: Provider claims with safe ownership-freshness status
          content:
            application/json:
              schema:
                type: object
                required: [claims]
                properties:
                  claims: { type: array, items: { $ref: "#/components/schemas/ProviderClaim" } }
        "401": { description: Human account session required }
    post:
      summary: Begin an indexed-domain provider claim
      operationId: createProviderClaim
      description: The returned TXT value must remain published after verification. NHS persists only its SHA-256 token hash, then automatically rechecks domain control; raw DNS answers are not retained.
      security: [{ SessionCookie: [] }]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              additionalProperties: false
              required: [domain]
              properties:
                domain: { type: string, description: Domain already present in the NHS index }
      responses:
        "201":
          description: Claim, one-time DNS TXT challenge, and ownership-freshness contract
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ProviderClaimChallengeResponse" }
        "401": { description: Human account session required }
        "409": { description: Domain is already claimed or the account already has a claim }
  /provider/claims/{claim_id}/verify:
    post:
      summary: Verify the claim's DNS TXT challenge and return the callback key once
      operationId: verifyProviderClaim
      description: Keep the verified TXT value published. NHS stores only the token hash and automatically rechecks it. Paid-action eligibility stops after %d consecutive failures or when the last successful check reaches %d days old.
      security: [{ SessionCookie: [] }]
      parameters:
        - { name: claim_id, in: path, required: true, schema: { type: string, format: uuid } }
      responses:
        "200":
          description: Verified claim, current ownership freshness, and a newly issued provider key returned once when applicable
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ProviderClaimVerifyResponse" }
        "409": { description: DNS challenge missing, mismatched, or expired; a real failed repeat check on a verified claim advances its freshness failure count }
  /provider/claims/{claim_id}/challenge:
    post:
      summary: Rotate a pending claim's DNS challenge
      operationId: rotateProviderClaimChallenge
      description: Replaces the pending token hash and returns the new TXT value once. After verification, keep that value published for automatic ownership rechecks.
      security: [{ SessionCookie: [] }]
      parameters:
        - { name: claim_id, in: path, required: true, schema: { type: string, format: uuid } }
      responses:
        "200":
          description: Rotated one-time DNS TXT challenge and ownership-freshness contract
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ProviderClaimChallengeResponse" }
  /provider/claims/{claim_id}/revoke:
    post:
      summary: Revoke a provider claim, keys, offers, and outstanding action authorization
      operationId: revokeProviderClaim
      security: [{ SessionCookie: [] }]
      parameters:
        - { name: claim_id, in: path, required: true, schema: { type: string, format: uuid } }
      responses:
        "200": { description: Claim revoked }
  /provider/claims/{claim_id}/keys/rotate:
    post:
      summary: Rotate the claim-scoped provider callback key
      operationId: rotateProviderCallbackKey
      security: [{ SessionCookie: [] }]
      parameters:
        - { name: claim_id, in: path, required: true, schema: { type: string, format: uuid } }
      responses:
        "200": { description: New provider key returned once; prior keys revoked }
  /provider/offers:
    get:
      summary: List offers owned by the signed-in provider account
      operationId: listProviderOffers
      security: [{ SessionCookie: [] }]
      parameters:
        - { name: claim_id, in: query, required: false, schema: { type: string, format: uuid } }
      responses:
        "200": { description: Provider offers and commercial states }
    post:
      summary: Create a draft provider-funded action offer
      operationId: createProviderOffer
      description: Drafts cannot appear beside organic results until NHS records real prepaid funding or exact capped CPA terms and activates the offer.
      security: [{ SessionCookie: [] }]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              allOf:
                - { $ref: "#/components/schemas/ProviderOfferRequest" }
                - { type: object, required: [claim_id] }
      responses:
        "201": { description: Draft offer created }
        "409": { description: Offer inventory limit reached }
  /provider/offers/{offer_id}:
    get:
      summary: Get one offer owned by the signed-in provider account
      operationId: getProviderOffer
      security: [{ SessionCookie: [] }]
      parameters:
        - { name: offer_id, in: path, required: true, schema: { type: string, format: uuid } }
      responses:
        "200": { description: Provider offer }
    put:
      summary: Update an owned draft offer
      operationId: updateProviderOffer
      security: [{ SessionCookie: [] }]
      parameters:
        - { name: offer_id, in: path, required: true, schema: { type: string, format: uuid } }
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: "#/components/schemas/ProviderOfferRequest" }
      responses:
        "200": { description: Updated offer }
  /provider/offers/{offer_id}/pause:
    post:
      summary: Pause an owned offer
      operationId: pauseProviderOffer
      security: [{ SessionCookie: [] }]
      parameters:
        - { name: offer_id, in: path, required: true, schema: { type: string, format: uuid } }
      responses:
        "200": { description: Offer paused }
  /action-interests:
    post:
      summary: Record caller-attested principal interest in one controlled next step
      operationId: recordActionInterest
      description: Creates a private, query-free Stage 1 demand receipt bound to an exact returned organic domain. It expires with the source search, no later than 30 days after that search. It does not contact the provider, create an action ticket or charge, affect organic rank or readiness score, or count as commercial proof. The caller attests current principal interest under the exact wording at https://nothumansearch.ai/privacy#action-interest-v1; NHS does not verify identity, agency, or legal authority.
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: "#/components/schemas/ActionInterestRequest" }
      responses:
        "201":
          description: New provider-independent action-interest receipt
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ActionInterestResponse" }
        "200":
          description: Exact idempotent replay of the existing receipt
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ActionInterestResponse" }
        "403": { description: Cross-origin browser mutation rejected; native agents without browser-origin headers remain supported }
        "404": { description: Missing, stale, synthetic, or non-returned organic source; intentionally indistinguishable }
        "409": { description: This search result already recorded a different controlled action }
        "429": { description: Temporary free abuse limit exceeded }
  /action-tickets:
    post:
      summary: Prepare an authorization-attested action for a disclosed paid offer
      operationId: createActionTicket
      description: Requires a committed organic search receipt and exact principal-consent v1 attestation. Accepts controlled constraints only; no name, email, contact detail, raw prompt, agent identity, or principal identity. See https://nothumansearch.ai/privacy#consent-v1.
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: "#/components/schemas/ActionTicketRequest" }
      responses:
        "201": { description: New signed action ticket and attributed provider URL }
        "200": { description: Exact idempotent replay reconstructed from the persisted ticket snapshot }
        "402": { description: Provider budget unavailable; the principal is not charged }
        "409": { description: Offer unavailable, revoked, or request conflicts with a prior ticket }
  /provider/outcomes:
    post:
      summary: Record an idempotent provider-reported action outcome
      operationId: recordProviderOutcome
      description: Provider-authenticated assertion, not an independent NHS audit. A charged ticket may later receive an invalid or duplicate credit after expiry or revocation; no positive outcome or new charge may cross those boundaries.
      security: [{ ProviderKey: [] }]
      parameters:
        - name: Idempotency-Key
          in: header
          required: true
          schema: { type: string, minLength: 8, maxLength: 200 }
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              additionalProperties: false
              required: [ticket_id, attribution_token, outcome]
              properties:
                ticket_id: { type: string, format: uuid }
                attribution_token: { type: string }
                outcome: { type: string, enum: [accepted, activated, converted, rejected, duplicate, invalid] }
      responses:
        "201": { description: New signed provider-outcome receipt }
        "200": { description: Exact idempotent replay }
        "401": { description: Valid claim-scoped provider key required }
        "409": { description: Invalid transition, revoked authorization, or conflicting idempotency payload }
  /provider/receipts/{receipt_id}:
    get:
      summary: Retrieve one receipt owned by the authenticated provider
      operationId: getProviderReceipt
      security: [{ ProviderKey: [] }]
      parameters:
        - { name: receipt_id, in: path, required: true, schema: { type: string, format: uuid } }
      responses:
        "200": { description: Signed provider receipt }
        "401": { description: Valid claim-scoped provider key required }
        "404": { description: Receipt not found for this provider }
  /action-receipts/verify:
    post:
      summary: Verify immutable NHS signature separately from freshness and current accounting state
      operationId: verifyActionReceipt
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              additionalProperties: false
              required: [signed_receipt, signature]
              properties:
                signed_receipt: { type: string, description: Exact canonical JSON returned by NHS }
                signature: { type: string, description: Unpadded base64url HMAC }
      responses:
        "200":
          description: Signature validity, time-window status, and current online commercial state when available
          content:
            application/json:
              schema:
                type: object
                properties:
                  signature_valid: { type: boolean }
                  within_validity_window: { type: boolean }
                  time_status: { type: string, enum: [current, expired, not_yet_valid, invalid_time] }
                  receipt: { $ref: "#/components/schemas/SignedOutcomeReceipt" }
                  current_state_available: { type: boolean }
                  current_state_status: { type: string, enum: [current, not_found, unavailable] }
                  current_state: { $ref: "#/components/schemas/PublicOutcomeReceiptState" }
  /site/{domain}:
    get:
      summary: Get detailed agentic readiness report for a site
      operationId: getSite
      parameters:
        - name: domain
          in: path
          required: true
          schema: { type: string }
        - name: search_id
          in: query
          required: false
          description: Optional receipt returned by search; records a detail selection only when this domain was returned
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
        An optional priority key raises this to 100/hour while its monthly
        allocation remains; exhaustion falls back to the free tier.
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
  /verify-mcp:
    get:
      summary: Live-probe a URL for MCP compliance
      operationId: verifyMCP
      description: |
        Sends a JSON-RPC tools/list request to the target URL and reports
        whether it responded with a spec-compliant reply. REST peer of
        the MCP verify_mcp tool. No caching — caller is asking "is it
        live right now?"
      parameters:
        - name: url
          in: query
          required: true
          description: Target URL (with or without scheme)
          schema: { type: string }
      responses:
        "200":
          description: Probe result
          content:
            application/json:
              schema:
                type: object
                properties:
                  verified: { type: boolean }
                  endpoint: { type: string }
                  note:     { type: string }
        "400":
          description: url query param missing
        "429":
          description: Temporary live-probe safety limit exceeded; free access resumes after the reset
  /stats:
    get:
      summary: Get index statistics
      operationId: getStats
      responses:
        "200":
          description: Index stats
  /top:
    get:
      summary: Top-scored agent-ready sites
      description: >
        Returns the highest-scored sites in the index (sorted by agentic_score DESC).
        Designed as a stable JSON other sites can mirror / embed. Cached 5 min.
      operationId: getTop
      parameters:
        - in: query
          name: category
          schema: { type: string }
          description: Public category filter. Audit-only buckets other and spam may be queried directly but are not promoted as discovery inventory.
        - in: query
          name: has_mcp
          schema: { type: boolean }
          description: Return only sites with a verified MCP server
        - in: query
          name: has_openapi
          schema: { type: boolean }
        - in: query
          name: has_llms_txt
          schema: { type: boolean }
        - in: query
          name: has_api
          schema: { type: boolean }
        - in: query
          name: limit
          schema: { type: integer, default: 50, maximum: 100 }
      responses:
        "200":
          description: Top sites
          content:
            application/json:
              schema:
                type: object
                properties:
                  results: { type: array, items: { $ref: "#/components/schemas/Site" } }
                  total:   { type: integer }
                  limit:   { type: integer }
components:
  securitySchemes:
    SessionCookie:
      type: apiKey
      in: cookie
      name: nhs_session
      description: Human account session created by the fail-closed email sign-in flow
    ProviderKey:
      type: apiKey
      in: header
      name: X-NHS-Provider-Key
      description: Claim-scoped callback key returned once after DNS verification or explicit rotation
  schemas:
    ProviderClaim:
      type: object
      description: Provider ownership state. NHS never returns the persisted challenge-token hash or raw DNS answers.
      required: [id, site_id, domain, verification_method, verification_record_name, status, challenge_expires_at, verification_consecutive_failures, created_at, updated_at]
      properties:
        id: { type: string, format: uuid }
        site_id: { type: string, format: uuid }
        domain: { type: string }
        verification_method: { type: string, enum: [dns_txt] }
        verification_record_name: { type: string, description: TXT record name that must remain published while the claim is verified }
        status: { type: string, enum: [pending, verified, revoked] }
        challenge_expires_at: { type: string, format: date-time, description: Expiry for a pending one-time challenge; it does not authorize removal of a verified TXT record }
        verified_at: { type: string, format: date-time, nullable: true }
        verification_last_succeeded_at: { type: string, format: date-time, nullable: true, description: Last successful DNS ownership check; paid actions require this to remain within the freshness window }
        verification_last_attempted_at: { type: string, format: date-time, nullable: true }
        verification_consecutive_failures: { type: integer, minimum: 0, description: Consecutive automatic or owner-triggered DNS failures since the last success }
        verification_next_check_at: { type: string, format: date-time, nullable: true, description: Next scheduled automatic DNS ownership check }
        revoked_at: { type: string, format: date-time, nullable: true }
        created_at: { type: string, format: date-time }
        updated_at: { type: string, format: date-time }
    OwnershipFreshness:
      type: object
      description: Machine-readable persistent DNS ownership contract and claim-specific safe status; no token, token hash, or raw DNS answer is exposed.
      required: [proof_method, record_must_remain_published, stored_challenge_material, raw_dns_answers_retained, automatic_reverification, recheck_interval_seconds, paid_actions_stop_after_consecutive_failures, paid_actions_stop_when_last_success_age_reaches_seconds, last_succeeded_at, next_check_at, consecutive_failures]
      properties:
        proof_method: { type: string, enum: [dns_txt] }
        record_must_remain_published: { type: boolean, enum: [true] }
        stored_challenge_material: { type: string, enum: [sha256_hash_only], description: NHS persists only the SHA-256 hash of the challenge token }
        raw_dns_answers_retained: { type: boolean, enum: [false], description: TXT answers are compared in memory and are not persisted }
        automatic_reverification: { type: boolean, enum: [true] }
        recheck_interval_seconds: { type: integer, enum: [%d], description: Interval scheduled after a successful check; next_check_at reflects any earlier failure retry }
        paid_actions_stop_after_consecutive_failures: { type: integer, enum: [%d], description: This consecutive failed check revokes the claim and stops paid-action eligibility }
        paid_actions_stop_when_last_success_age_reaches_seconds: { type: integer, enum: [%d], description: Paid-action eligibility stops at this age even before a revocation update is recorded }
        last_succeeded_at: { type: string, format: date-time, nullable: true }
        next_check_at: { type: string, format: date-time, nullable: true }
        consecutive_failures: { type: integer, minimum: 0 }
    ProviderClaimChallengeResponse:
      type: object
      required: [claim, dns_challenge, ownership_freshness, verify_endpoint]
      properties:
        claim: { $ref: "#/components/schemas/ProviderClaim" }
        dns_challenge:
          type: object
          required: [record_type, record_name, record_value, expires_at, returned_once]
          properties:
            record_type: { type: string, enum: [TXT] }
            record_name: { type: string }
            record_value: { type: string, description: Returned once. Keep the full TXT value published after verification; NHS stores only the token hash. }
            expires_at: { type: string, format: date-time }
            returned_once: { type: boolean, enum: [true] }
        ownership_freshness: { $ref: "#/components/schemas/OwnershipFreshness" }
        verify_endpoint: { type: string }
    ProviderClaimVerifyResponse:
      type: object
      required: [claim, verified, provider_key_returned, ownership_freshness]
      properties:
        claim: { $ref: "#/components/schemas/ProviderClaim" }
        verified: { type: boolean, enum: [true] }
        provider_key: { type: string, description: Newly issued callback key returned once when this verification created it }
        provider_key_metadata: { type: object }
        provider_key_returned: { type: boolean }
        save_this_key_now: { type: boolean }
        key_endpoint: { type: string, description: Explicit rotation endpoint when a concurrent verification already issued the active key }
        ownership_freshness: { $ref: "#/components/schemas/OwnershipFreshness" }
    ProviderOfferRequest:
      type: object
      additionalProperties: false
      required: [name, summary, action_type, action_url, charge_event, bounty_cents, currency, principal_price_mode, principal_currency, billing_mode]
      properties:
        claim_id: { type: string, format: uuid }
        name: { type: string, minLength: 1, maxLength: 80 }
        summary: { type: string, minLength: 1, maxLength: 280 }
        action_type: { type: string, enum: [lead, demo, trial, signup, purchase, quote, application, booking] }
        action_url: { type: string, format: uri, description: HTTPS URL on the verified provider domain, with no query or fragment }
        charge_event: { type: string, enum: [accepted, activated, converted] }
        bounty_cents: { type: integer, minimum: 1, maximum: 1000000 }
        currency: { type: string, enum: [usd] }
        principal_price_mode: { type: string, enum: [free, fixed, quote, provider_pricing] }
        principal_price_cents: { type: integer, minimum: 0, maximum: 100000000 }
        principal_currency: { type: string, enum: [usd] }
        billing_mode: { type: string, enum: [prepaid, terms] }
        terms_credit_limit_cents: { type: integer, minimum: 1, maximum: 10000000 }
        terms_period_days: { type: integer, minimum: 1, maximum: 90 }
    PublicProviderOffer:
      type: object
      description: Separate disclosed action attached to an exact returned organic site; the provider action URL is withheld until a consented ticket is created
      properties:
        id: { type: string, format: uuid }
        provider_domain: { type: string }
        organic_position: { type: integer, minimum: 1 }
        name: { type: string }
        summary: { type: string }
        action_type: { type: string }
        disclosure: { type: string, enum: [Provider-funded action] }
        organic_rank_paid: { type: boolean, enum: [false] }
        principal_price:
          type: object
          properties:
            mode: { type: string, enum: [free, fixed, quote, provider_pricing] }
            amount_minor: { type: integer, minimum: 0 }
            currency: { type: string, enum: [usd] }
        nhs_compensation:
          type: object
          properties:
            event: { type: string, enum: [accepted, activated, converted] }
            amount_minor: { type: integer, minimum: 1 }
            currency: { type: string, enum: [usd] }
        prepare_action_endpoint: { type: string, format: uri }
    ActionInterestRequest:
      type: object
      additionalProperties: false
      required: [search_id, domain, action_type, caller_attests_principal_interest, confirmation_version]
      properties:
        search_id: { type: string, pattern: "^nhs_sr_[A-Za-z0-9_-]{16}$", description: Committed query-free organic search receipt }
        domain: { type: string, description: Bare domain present in the referenced organic results; schemes, paths, queries, and fragments are rejected }
        action_type: { type: string, enum: [quote, trial, demo, booking, application, signup, purchase] }
        caller_attests_principal_interest: { type: boolean, enum: [true], description: Caller attests current human/company principal interest; this is not authority to contact the provider }
        confirmation_version: { type: string, enum: [nhs-action-interest-v1] }
    ActionInterestReceipt:
      type: object
      description: Provider-independent Stage 1 demand receipt; not a provider request, action ticket, charge, outcome, or commercial proof
      properties:
        id: { type: string, pattern: "^nhs_air_[A-Za-z0-9_-]{16}$" }
        search_id: { type: string }
        domain: { type: string }
        action_type: { type: string, enum: [quote, trial, demo, booking, application, signup, purchase] }
        surface: { type: string, enum: [rest, mcp, web, unknown] }
        caller_attests_principal_interest: { type: boolean, enum: [true] }
        confirmation_version: { type: string, enum: [nhs-action-interest-v1] }
        created_at: { type: string, format: date-time }
        expires_at: { type: string, format: date-time }
        idempotent_replay: { type: boolean }
    ActionInterestResponse:
      type: object
      required: [receipt, created, idempotent_replay, provider_contacted, row_level_shared_with_provider, action_ticket_created, charge_created, provider_or_principal_charged, commercial_proof, organic_rank_affected, rank_or_score_input, retention_days, evidence_scope]
      properties:
        receipt: { $ref: "#/components/schemas/ActionInterestReceipt" }
        created: { type: boolean }
        idempotent_replay: { type: boolean }
        provider_contacted: { type: boolean, enum: [false] }
        row_level_shared_with_provider: { type: boolean, enum: [false] }
        action_ticket_created: { type: boolean, enum: [false] }
        charge_created: { type: boolean, enum: [false] }
        provider_or_principal_charged: { type: boolean, enum: [false] }
        commercial_proof: { type: boolean, enum: [false] }
        organic_rank_affected: { type: boolean, enum: [false] }
        rank_or_score_input: { type: boolean, enum: [false] }
        retention_days: { type: integer, enum: [30] }
        evidence_scope: { type: string }
    ActionTicketRequest:
      type: object
      additionalProperties: false
      required: [offer_id, search_id, demand_topic, principal_consent, consent_version]
      properties:
        offer_id: { type: string, format: uuid }
        search_id: { type: string, description: Committed query-free organic search receipt }
        demand_topic: { type: string, enum: [payments, commerce, jobs, data, search, weather, maps, email, messaging, image, video, audio, documents, security, finance, health, education, news, analytics, automation, productivity, identity, storage, ai-tools, developer-tools, other] }
        region_code: { type: string, pattern: "^[A-Z]{2}(-[A-Z0-9]{1,3})?$" }
        budget_band: { type: string, enum: [unspecified, under_100, 100_499, 500_1999, 2000_plus], default: unspecified }
        urgency: { type: string, enum: [unspecified, now, 7_days, 30_days, researching], default: unspecified }
        requirement_flags:
          type: array
          uniqueItems: true
          items: { type: string, enum: [api_access, mcp, sandbox, self_serve, enterprise, compliance, multilingual, human_support] }
        principal_consent: { type: boolean, enum: [true], description: Caller attests it is authorized by the principal under the exact published v1 wording }
        consent_version: { type: string, enum: [nhs-principal-consent-v1] }
    SignedOutcomeReceipt:
      type: object
      properties:
        v: { type: integer, enum: [1] }
        kid: { type: string }
        receipt_id: { type: string, format: uuid }
        ticket_id: { type: string, format: uuid }
        offer_id: { type: string, format: uuid }
        nhs_event_id: { type: string, format: uuid }
        outcome: { type: string, enum: [accepted, activated, converted, rejected, duplicate, invalid] }
        provider_reported_at: { type: integer, format: int64 }
        recorded_at: { type: integer, format: int64 }
        expires_at: { type: integer, format: int64 }
        charged_minor: { type: integer, minimum: 0 }
        currency: { type: string, enum: [usd] }
        charge_status: { type: string, enum: [charged, credited, none] }
    PublicOutcomeReceiptState:
      type: object
      description: Mutable online state, separate from immutable signature validity
      properties:
        receipt_id: { type: string, format: uuid }
        action_ticket_id: { type: string, format: uuid }
        receipt_outcome: { type: string }
        current_ticket_status: { type: string }
        original_charge_credited: { type: boolean }
        superseded_by_later_state: { type: boolean }
        authorization_revoked: { type: boolean }
        net_commercial_effect_cents: { type: integer, format: int64, minimum: 0 }
        net_commercial_effect_currency: { type: string, enum: [usd] }
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
        is_featured:
          type: boolean
          deprecated: true
          description: Legacy display metadata only; never affects organic score or ordering
`, h.BaseURL, searchCategoryOpenAPIEnum(),
		models.ProviderClaimDNSFailureLimit,
		int(models.ProviderClaimVerificationFreshness/(24*time.Hour)),
		int64(models.ProviderClaimDNSRecheckInterval/time.Second),
		models.ProviderClaimDNSFailureLimit,
		int64(models.ProviderClaimVerificationFreshness/time.Second))
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
	w.Header().Set("Cache-Control", "public, max-age=3600")

	sm := sitemap{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9"}

	// Static pages
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/", ChangeFreq: "daily", Priority: "1.0"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/mcp-servers", ChangeFreq: "daily", Priority: "0.9"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/ai-tools", ChangeFreq: "daily", Priority: "0.9"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/developer-apis", ChangeFreq: "daily", Priority: "0.9"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/openapi-apis", ChangeFreq: "daily", Priority: "0.9"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/llms-txt-sites", ChangeFreq: "daily", Priority: "0.9"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/top", ChangeFreq: "daily", Priority: "0.9"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/newest", ChangeFreq: "daily", Priority: "0.9"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/score", ChangeFreq: "weekly", Priority: "0.9"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/data-apis", ChangeFreq: "daily", Priority: "0.8"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/finance-apis", ChangeFreq: "daily", Priority: "0.8"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/ecommerce-apis", ChangeFreq: "daily", Priority: "0.8"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/productivity-apis", ChangeFreq: "daily", Priority: "0.8"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/security-apis", ChangeFreq: "daily", Priority: "0.8"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/communication-apis", ChangeFreq: "daily", Priority: "0.8"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/jobs-apis", ChangeFreq: "daily", Priority: "0.8"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/about", ChangeFreq: "weekly", Priority: "0.5"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/providers", ChangeFreq: "weekly", Priority: "0.8"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/privacy", ChangeFreq: "monthly", Priority: "0.5"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/guide", ChangeFreq: "weekly", Priority: "0.9"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/report", ChangeFreq: "daily", Priority: "0.9"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/status", ChangeFreq: "hourly", Priority: "0.6"})
	// NOTE: /score already added above with priority 0.9 — don't duplicate.
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/feed.xml", ChangeFreq: "hourly", Priority: "0.7"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/digest", ChangeFreq: "daily", Priority: "0.8"})
	sm.URLs = append(sm.URLs, sitemapURL{Loc: h.BaseURL + "/digest.rss", ChangeFreq: "daily", Priority: "0.6"})
	for _, cat := range []string{"ai-tools", "developer", "finance", "data", "ecommerce", "productivity", "security", "communication", "jobs", "health", "education", "news"} {
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

	// Site pages. Keep this tied to AgentFirstFilter so passive llms.txt-only
	// rows never become public discovery targets through the sitemap.
	rows, err := h.DB.QueryContext(r.Context(), "SELECT domain, updated_at FROM sites WHERE "+models.AgentFirstFilter+" ORDER BY agentic_score DESC LIMIT 49999")
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
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	AtomLink      atomLink  `xml:"atom:link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language"`
	LastBuildDate string    `xml:"lastBuildDate"`
	Items         []rssItem `xml:"item"`
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
