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
Tools (14): search_agents, get_site_details, get_stats, list_categories, get_top_sites, submit_site, register_monitor, verify_mcp, find_mcp_servers, recent_additions, check_url, record_action_interest, prepare_provider_action, handoff_provider_action

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

Each public offer sidecar exposes offer_version and the exact
commercial_terms_contract_version/commercial_terms_sha256, fixed credit rule,
provider response expectation, first-activation terms anchor rule, and provider
Merchant-of-Record acknowledgement used for that version.

POST /api/v1/action-tickets
Creates a ticket after the caller supplies search_id, offer_id, one controlled
demand topic already on the receipt, and the exact versioned principal-
authorization attestation. It returns the raw attribution_token once (or
reconstructs it for an exact replay) plus the POST handoff_endpoint; it does not
return the provider action URL. Creating a ticket charges neither party.

POST /api/v1/action-tickets/handoff
Body: {"ticket_id":"...","attribution_token":"...",
"principal_handoff_consent":true,
"handoff_consent_version":"nhs-provider-handoff-consent-v1"}
Presents the ticket bearer and separate handoff-time principal attestation only
in a JSON body; every response is private, no-store. NHS first commits one
append-only nhs-action-handoff-v1 receipt bound to the exact ticket, offer
version, commercial-terms hash, and handoff-consent version, then returns the
attributed provider action URL. The receipt contains a one-way hash of the
presented bearer but no query, agent/principal identity, contact data, network
address, referrer, or user agent. The handoff charges neither party; only the
disclosed authenticated provider-reported downstream event can create a
provider charge.

POST /api/v1/provider/commercial-acceptances
Headers: X-NHS-Provider-Key and Idempotency-Key
Records a provider-authenticated pilot-company, exact-terms acceptance, or
exact-terms renewal. A terms event must carry the exact offer_version and
exact_terms_sha256 the provider reviewed; a changed draft is rejected rather
than silently accepted. This provider event is append-only but is not funding,
owner verification, or commercial proof by itself; the keyed company and exact
commercial evidence are verified separately by the NHS owner.

Consent wording: /privacy#consent-v1
Handoff consent wording: /privacy#handoff-consent-v1
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
		"version":     "1.1.0",
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
				"description": "Create an authorization-attested action ticket for a separately disclosed provider-funded offer. Requires an organic search receipt; accepts controlled fields only; returns a bearer token and POST handoff endpoint instead of the provider URL. The handoff separately requires nhs-provider-handoff-consent-v1, creates a privacy-safe receipt, and charges neither party. Exact wording is published at /privacy#consent-v1 and /privacy#handoff-consent-v1.",
				"endpoint":    h.BaseURL + "/api/v1/action-tickets",
				"method":      "POST",
			},
			{
				"name":        "handoff_provider_action",
				"description": "Present the exact ticket bearer plus principal_handoff_consent=true and handoff_consent_version=nhs-provider-handoff-consent-v1. NHS records a privacy-safe observed-handoff receipt before returning the provider action URL; the handoff charges neither party.",
				"endpoint":    h.BaseURL + "/api/v1/action-tickets/handoff",
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
		"description_for_model": "Search for websites and APIs that are agent-ready. Returns sites scored 0-100 on agentic readiness based on 7 signals (llms.txt, OpenAPI, ai-plugin.json, structured APIs, MCP server, robots.txt AI rules, Schema.org). Key REST endpoints: GET /api/v1/search (with filters has_mcp, has_openapi, has_llms_txt), GET /api/v1/top (top-scored sites, filterable by signal), GET /api/v1/site/{domain}, GET /api/v1/verify-mcp?url=, and POST /api/v1/check. Organic search and canonical provider links are free and neutral. Caller-attested principal interest can be recorded without provider contact, payment, or rank effects. Separately disclosed provider-funded actions may appear only beside an already-returned organic result. Ticket creation returns a bearer token and POST handoff endpoint rather than the provider URL; the privacy-safe observed handoff separately requires nhs-provider-handoff-consent-v1 and charges neither party. For richer capabilities connect via MCP at /mcp — 14 tools including record_action_interest, prepare_provider_action, and handoff_provider_action.",
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
  description: Free neutral search for AI agents. Optional provider-funded actions stay separate from organic rank; only an authenticated downstream provider outcome can create the disclosed provider charge.
  version: "1.1.0"
  x-version-policy: Descriptive release version; controlled-pilot provider endpoints carry explicit contract versions and can require an owner-gated breaking cutover.
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
      description: Terms-only launch pilot. Drafts cannot appear beside organic results until the provider authenticates the exact capped CPA terms, the owner verifies that acceptance, and NHS activates the offer.
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
  /provider/commercial-acceptances:
    post:
      summary: Record one provider-authenticated commercial acceptance
      operationId: recordProviderCommercialAcceptance
      description: Provider-key-authenticated append-only acceptance only. It cannot establish a deduplicated company, funding, exact terms, renewal, or pilot proof until NHS separately records the applicable owner-verified company and commercial evidence. Accepted event shapes are exact and unknown fields are rejected.
      security: [{ ProviderKey: [] }]
      parameters:
        - name: Idempotency-Key
          in: header
          required: true
          description: Claim-scoped opaque replay key. Reuse with a different exact payload is rejected.
          schema:
            type: string
            minLength: 8
            maxLength: 128
            pattern: "^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$"
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: "#/components/schemas/ProviderCommercialAcceptanceRequest" }
      responses:
        "201":
          description: New provider-authenticated acceptance; owner verification is still required and commercial proof is false
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ProviderCommercialAcceptanceResponse" }
        "200":
          description: Exact idempotent replay of the existing provider-authenticated acceptance
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ProviderCommercialAcceptanceResponse" }
        "400": { description: Invalid header, event shape, reference, or unknown field }
        "401": { description: Valid active claim-scoped provider key required }
        "409": { description: Stale claim, conflicting replay, mismatched offer, or invalid renewal chain }
  /provider/pilot-status:
    get:
      summary: Read claim-scoped pilot continuity status
      operationId: getProviderPilotStatus
      description: Provider-key-authenticated read-only status for the key's own DNS-verified claim. It includes provider setup, exact owned offer and terms state, and only handoff or outcome events that have crossed the NHS-observed handoff boundary. It returns no credentials, attribution material, search receipts, controlled intent, queries, identities, contacts, network data, company hashes, or action URLs.
      security: [{ ProviderKey: [] }]
      parameters:
        - name: limit
          in: query
          required: false
          description: Maximum owned offers and recent observed handoff records returned in each collection.
          schema: { type: integer, minimum: 1, maximum: 100, default: 25 }
      responses:
        "200":
          description: Current status for the authenticated provider claim
          headers:
            Cache-Control:
              schema: { type: string, enum: ["private, no-store"] }
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ProviderPilotStatusResponse" }
        "400": { description: Invalid limit }
        "401": { description: Valid active claim-scoped provider key required }
        "404": { description: Authenticated claim is no longer verified and fresh }
        "429": { description: Temporary provider read safety limit exceeded }
        "500": { description: Provider pilot status query failed }
        "503": { description: Provider key authentication unavailable }
  /provider/demand:
    get:
      summary: Read privacy-thresholded demand for the authenticated claim domain
      operationId: getProviderDemand
      description: Provider-key-authenticated read-only aggregate demand for the key's own current DNS-verified claim domain. The domain is derived from the authenticated claim and cannot be selected by the caller. Counts represent retained receipts, not unique agents or principals. Result-selection and action-interest counts and rates are suppressed below their exact receipt thresholds; topic and action-type rows are omitted below their thresholds. Organic-return counts and controlled surface labels remain reportable. No raw query, identity, contact, network data, alleged agent identity, or individual receipt is returned.
      security: [{ ProviderKey: [] }]
      parameters:
        - name: days
          in: query
          required: false
          description: Inclusive lookback window for retained aggregate receipts.
          schema: { type: integer, minimum: 1, maximum: 30, default: 30 }
      responses:
        "200":
          description: Privacy-thresholded aggregate demand for the authenticated claim domain
          headers:
            Cache-Control:
              schema: { type: string, enum: ["private, no-store"] }
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ProviderDemandResponse" }
        "400": { description: Invalid days }
        "401": { description: Valid active claim-scoped provider key required }
        "404": { description: Authenticated claim is no longer verified and fresh }
        "429": { description: Temporary provider read safety limit exceeded }
        "500": { description: Provider demand query failed }
        "503": { description: Provider key authentication unavailable }
  /provider/action-tickets/resolve:
    post:
      summary: Resolve separately consented controlled intent after an observed handoff
      operationId: resolveProviderControlledIntent
      description: Read-only claim-scoped provider resolution. The body accepts only the exact signed attribution bearer. Resolution is available only after an NHS-observed handoff that included the separate nhs-provider-controlled-intent-disclosure-consent-v1 attestation. It returns only the controlled topic, optional region code, USD budget band, urgency, allowlisted requirement flags, and opaque binding metadata. It returns no query, search receipt, identity, contact, network, action URL, price, accounting data, charge, outcome, or commercial proof. Resolver access is free and does not change organic rank or readiness.
      security: [{ ProviderKey: [] }]
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: "#/components/schemas/ProviderControlledIntentResolveRequest" }
      responses:
        "200":
          description: Exact separately consented controlled-intent bundle; no charge or proof created
          headers:
            Cache-Control:
              schema: { type: string, enum: ["private, no-store"] }
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ProviderControlledIntentResolution" }
        "400": { description: Malformed, unknown-field, empty, or invalid-signature attribution bearer }
        "401": { description: Valid active claim-scoped provider key required }
        "404": { description: Consented controlled intent unavailable; wrong claim, absent consent, and ineligible state are intentionally indistinguishable }
        "410": { description: Correctly signed attribution bearer expired }
        "429": { description: Temporary provider resolver safety limit exceeded }
        "503": { description: Provider exchange, signer, or resolver dependency unavailable }
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
      description: Requires a committed organic search receipt and exact principal-consent v1 attestation. Accepts controlled constraints only; no name, email, contact detail, raw prompt, agent identity, or principal identity. Returns the raw ticket bearer and POST handoff endpoint, not the provider action URL. Creating a ticket charges neither party. See https://nothumansearch.ai/privacy#consent-v1.
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: "#/components/schemas/ActionTicketRequest" }
      responses:
        "201":
          description: New ticket, raw attribution bearer, and POST handoff endpoint; no provider action URL and no charge
          headers:
            Cache-Control:
              schema: { type: string, enum: ["private, no-store"] }
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ActionTicketPreparationResponse" }
        "200":
          description: Exact replay with the attribution bearer reconstructed from the persisted ticket snapshot
          headers:
            Cache-Control:
              schema: { type: string, enum: ["private, no-store"] }
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ActionTicketPreparationResponse" }
        "400": { description: Invalid JSON, unknown field, ticket input, or consent attestation }
        "404": { description: Exact public offer or returned-offer evidence unavailable; intentionally indistinguishable }
        "409": { description: Provider claim, authorization, commercial evidence, or provider-funded capacity unavailable; or request conflicts with a prior ticket. The principal is not charged. }
        "410": { description: Exact replay refers to an expired ticket authorization }
        "429": { description: Temporary action-ticket safety limit exceeded }
        "503": { description: Signed provider actions are not configured }
  /action-tickets/handoff:
    post:
      summary: Record an NHS-observed handoff and reveal the provider action URL
      operationId: handoffActionTicket
      description: Presents the raw ticket bearer and the separate exact nhs-provider-handoff-consent-v1 principal attestation only in a bounded JSON body, never in the NHS URL or query string; every response is private, no-store. NHS atomically records one append-only privacy-safe nhs-action-handoff-v1 receipt bound to the exact ticket, offer version, commercial-terms hash, and handoff-consent version before returning the attributed provider action URL. The principal may separately and optionally authorize the exact DNS-verified provider to resolve only the bounded controlled-intent bundle under nhs-provider-controlled-intent-disclosure-consent-v1; declining that disclosure does not block this handoff or free direct provider access. The receipt contains no query, agent or principal identity, contact data, network address, referrer, or user agent. This handoff and the optional resolver charge neither party; only the configured authenticated provider-reported downstream outcome can create the disclosed provider charge. Exact wording is at https://nothumansearch.ai/privacy#handoff-consent-v1 and https://nothumansearch.ai/privacy#controlled-intent-disclosure-consent-v1.
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: "#/components/schemas/ActionTicketHandoffRequest" }
      responses:
        "201":
          description: New durable observed-handoff receipt and attributed provider URL; neither party charged
          headers:
            Cache-Control:
              schema: { type: string, enum: ["private, no-store"] }
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ActionTicketHandoffResponse" }
        "200":
          description: Exact replay of the existing durable handoff receipt and provider URL; neither party charged
          headers:
            Cache-Control:
              schema: { type: string, enum: ["private, no-store"] }
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ActionTicketHandoffResponse" }
        "400": { description: Invalid JSON, unknown field, ticket ID, or empty bearer }
        "404": { description: Ticket or exact bearer not found }
        "409": { description: Verified commercial evidence unavailable, authorization revoked, ticket already terminal, or otherwise ineligible for handoff; neither party charged }
        "410": { description: Ticket attribution expired; neither party charged }
        "429": { description: Temporary handoff safety limit exceeded }
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
              required: [attribution_token, outcome]
              properties:
                ticket_id: { type: string, format: uuid, deprecated: true, description: "Optional compatibility assertion; NHS derives the authoritative ticket from the verified attribution token" }
                attribution_token: { type: string, description: "The exact signed bearer received in the attributed provider action URL; NHS derives its ticket and offer binding server-side" }
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
      description: Claim-scoped provider key returned once after DNS verification or explicit rotation; used for provider reads, acceptances, controlled-intent resolution, and outcome callbacks
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
        billing_mode: { type: string, enum: [terms], description: "The bounded launch pilot supports provider-authenticated exact capped CPA terms only; prepaid collection is not launched" }
        terms_credit_limit_cents: { type: integer, minimum: 1, maximum: 10000000 }
        terms_period_days: { type: integer, minimum: 1, maximum: 90 }
    ProviderCommercialAcceptanceRequest:
      description: Exact provider-authenticated acceptance shapes. Unknown fields and shape-inappropriate fields are rejected.
      oneOf:
        - type: object
          additionalProperties: false
          required: [event_type, provider_acceptance_reference]
          properties:
            event_type: { type: string, enum: [pilot_company] }
            provider_acceptance_reference: { type: string, minLength: 8, maxLength: 200, pattern: "^[A-Za-z0-9][A-Za-z0-9._:/-]{7,199}$", description: Non-secret provider evidence reference }
        - type: object
          additionalProperties: false
          required: [event_type, offer_id, offer_version, exact_terms_sha256, provider_acceptance_reference]
          properties:
            event_type: { type: string, enum: [terms_acceptance] }
            offer_id: { type: string, format: uuid }
            offer_version: { type: integer, minimum: 1, description: Exact version reviewed by the provider; rejected if the current draft differs }
            exact_terms_sha256: { type: string, pattern: "^[0-9a-f]{64}$", description: Exact commercial terms hash reviewed by the provider; rejected if the current draft differs }
            provider_acceptance_reference: { type: string, minLength: 8, maxLength: 200, pattern: "^[A-Za-z0-9][A-Za-z0-9._:/-]{7,199}$", description: Non-secret provider evidence reference }
        - type: object
          additionalProperties: false
          required: [event_type, offer_id, related_acceptance_event_id, offer_version, exact_terms_sha256, provider_acceptance_reference]
          properties:
            event_type: { type: string, enum: [terms_renewal] }
            offer_id: { type: string, format: uuid }
            related_acceptance_event_id: { type: string, format: uuid, description: Prior terms_acceptance or terms_renewal in the same exact-terms chain }
            offer_version: { type: integer, minimum: 1, description: Exact version reviewed by the provider; rejected if the current draft differs }
            exact_terms_sha256: { type: string, pattern: "^[0-9a-f]{64}$", description: Exact commercial terms hash reviewed by the provider; rejected if the current draft differs }
            provider_acceptance_reference: { type: string, minLength: 8, maxLength: 200, pattern: "^[A-Za-z0-9][A-Za-z0-9._:/-]{7,199}$", description: Non-secret provider evidence reference }
    ProviderCommercialAcceptanceEvent:
      type: object
      description: Append-only provider-key-authenticated event; not owner verification or commercial proof by itself
      required: [id, provider_claim_id, provider_api_key_id, event_type, provider_acceptance_reference, provider_accepted_at, created_at]
      properties:
        id: { type: string, format: uuid }
        provider_claim_id: { type: string, format: uuid }
        provider_offer_id: { type: string, format: uuid }
        provider_api_key_id: { type: integer, format: int64 }
        event_type: { type: string, enum: [pilot_company, terms_acceptance, terms_renewal] }
        related_acceptance_event_id: { type: string, format: uuid }
        offer_version: { type: integer, minimum: 1 }
        terms_contract_version: { type: string, enum: [nhs-provider-commercial-terms-v1] }
        exact_terms_sha256: { type: string, pattern: "^[0-9a-f]{64}$" }
        provider_acceptance_reference: { type: string, minLength: 8, maxLength: 200 }
        provider_accepted_at: { type: string, format: date-time }
        valid_until: { type: string, format: date-time }
        created_at: { type: string, format: date-time }
    ProviderCommercialAcceptanceResponse:
      type: object
      additionalProperties: false
      required: [acceptance, created, idempotent_replay, provider_authenticated, owner_verification_required, commercial_proof_created, evidence_scope]
      properties:
        acceptance: { $ref: "#/components/schemas/ProviderCommercialAcceptanceEvent" }
        created: { type: boolean }
        idempotent_replay: { type: boolean }
        provider_authenticated: { type: boolean, enum: [true] }
        owner_verification_required: { type: boolean, enum: [true] }
        commercial_proof_created: { type: boolean, enum: [false] }
        evidence_scope: { type: string }
    ProviderPilotOfferStatus:
      type: object
      additionalProperties: false
      description: Exact owned offer contract and provider/owner acceptance state. Draft status does not imply activation or Merchant-of-Record acknowledgement.
      required: [offer_id, status, version, name, action_type, charge_event, bounty_cents, currency, billing_mode, commercial_terms_contract_version, commercial_terms_sha256, provider_mor_acknowledgement_required, provider_acknowledges_merchant_of_record, latest_acceptance_owner_verified, current_terms_owner_verified, renewal_eligible, activation_ready]
      properties:
        offer_id: { type: string, format: uuid }
        status: { type: string, enum: [draft, active, paused] }
        version: { type: integer, minimum: 1 }
        name: { type: string }
        action_type: { type: string, enum: [lead, demo, trial, signup, purchase, quote, application, booking] }
        charge_event: { type: string, enum: [accepted, activated, converted] }
        bounty_cents: { type: integer, format: int64, minimum: 1 }
        currency: { type: string, enum: [usd] }
        billing_mode: { type: string, enum: [prepaid, terms], description: The bounded launch pilot activates terms offers only; prepaid is retained solely for historical status compatibility. }
        terms_credit_limit_cents: { type: integer, format: int64, minimum: 1 }
        terms_period_days: { type: integer, minimum: 1, maximum: 90 }
        commercial_terms_contract_version: { type: string, enum: [nhs-provider-commercial-terms-v1] }
        commercial_terms_sha256: { type: string, pattern: "^[0-9a-f]{64}$" }
        provider_mor_acknowledgement_required: { type: boolean, enum: [true] }
        provider_acknowledges_merchant_of_record: { type: boolean, description: True only for the active provider-accepted contract; this is not an independent NHS verification. }
        latest_acceptance_id: { type: string, format: uuid }
        latest_acceptance_type: { type: string, enum: [terms_acceptance, terms_renewal] }
        latest_acceptance_at: { type: string, format: date-time }
        latest_acceptance_valid_until: { type: string, format: date-time }
        latest_acceptance_owner_verified: { type: boolean }
        latest_acceptance_owner_verified_at: { type: string, format: date-time }
        current_terms_owner_verified: { type: boolean }
        current_terms_valid_until: { type: string, format: date-time }
        renewal_eligible: { type: boolean }
        activation_ready: { type: boolean, description: Read-only readiness evidence; it does not activate the offer. }
    ProviderPilotRecentEvent:
      type: object
      additionalProperties: false
      description: Provider-owned ticket state exposed only after an NHS-observed handoff. Attribution material and controlled intent are excluded.
      required: [ticket_id, offer_id, offer_version, ticket_status, handoff_receipt_id, handoff_observed_at]
      properties:
        ticket_id: { type: string, format: uuid }
        offer_id: { type: string, format: uuid }
        offer_version: { type: integer, minimum: 1 }
        ticket_status: { type: string, enum: [created, redirected, accepted, activated, converted, rejected, duplicate, invalid, expired, revoked] }
        handoff_receipt_id: { type: string, format: uuid }
        handoff_observed_at: { type: string, format: date-time }
        outcome_receipt_id: { type: string, format: uuid }
        outcome: { type: string, enum: [accepted, activated, converted, rejected, duplicate, invalid] }
        charge_status: { type: string, enum: [charged, credited, none] }
        billed_cents: { type: integer, format: int64, minimum: 0 }
        outcome_recorded_at: { type: string, format: date-time }
    ProviderPilotStatus:
      type: object
      additionalProperties: false
      required: [as_of, provider_claim_id, domain, claim_status, verification_last_succeeded_at, verification_consecutive_failures, company_owner_verified, offers, recent_observed_handoffs]
      properties:
        as_of: { type: string, format: date-time, description: Database wall-clock boundary for this repeatable-read report. }
        provider_claim_id: { type: string, format: uuid }
        domain: { type: string, description: Domain derived from the authenticated provider claim. }
        claim_status: { type: string, enum: [verified] }
        verification_last_succeeded_at: { type: string, format: date-time }
        verification_next_check_at: { type: string, format: date-time }
        verification_consecutive_failures: { type: integer, minimum: 0 }
        company_acceptance_id: { type: string, format: uuid }
        company_accepted_at: { type: string, format: date-time }
        company_owner_verified: { type: boolean }
        company_owner_verified_at: { type: string, format: date-time }
        offers:
          type: array
          maxItems: 100
          items: { $ref: "#/components/schemas/ProviderPilotOfferStatus" }
        recent_observed_handoffs:
          type: array
          maxItems: 100
          items: { $ref: "#/components/schemas/ProviderPilotRecentEvent" }
    ProviderPilotStatusResponse:
      type: object
      additionalProperties: false
      required: [pilot_status, evidence_scope]
      properties:
        pilot_status: { $ref: "#/components/schemas/ProviderPilotStatus" }
        evidence_scope: { type: string, description: Claim-key-scoped continuity and explicit redaction boundary. }
    ProviderDemandSummary:
      type: object
      additionalProperties: false
      required: [organic_results_returned, search_receipts, average_organic_position, result_selections, result_selection_rate, result_selection_suppressed, action_interest_receipts, action_interest_rate, action_interest_suppressed]
      properties:
        organic_results_returned: { type: integer, minimum: 0 }
        search_receipts: { type: integer, minimum: 0, description: Retained receipts, not unique agents or principals. }
        average_organic_position: { type: number, format: double, minimum: 0 }
        result_selections: { type: integer, minimum: 0, nullable: true }
        result_selection_rate: { type: number, format: double, minimum: 0, maximum: 1, nullable: true }
        result_selection_suppressed: { type: boolean }
        action_interest_receipts: { type: integer, minimum: 0, nullable: true }
        action_interest_rate: { type: number, format: double, minimum: 0, maximum: 1, nullable: true }
        action_interest_suppressed: { type: boolean }
    ProviderDemandSurface:
      type: object
      additionalProperties: false
      required: [surface, organic_results_returned, result_selections, result_selection_suppressed, action_interest_receipts, action_interest_suppressed]
      properties:
        surface: { type: string, enum: [web, rest, mcp, unknown] }
        organic_results_returned: { type: integer, minimum: 0 }
        result_selections: { type: integer, minimum: 0, nullable: true }
        result_selection_suppressed: { type: boolean }
        action_interest_receipts: { type: integer, minimum: 0, nullable: true }
        action_interest_suppressed: { type: boolean }
    ProviderDemandTopic:
      type: object
      additionalProperties: false
      required: [topic, search_receipts, average_organic_position, result_selections, result_selection_suppressed, action_interest_receipts, action_interest_suppressed]
      properties:
        topic: { type: string, enum: [payments, commerce, jobs, data, search, weather, maps, email, messaging, image, video, audio, documents, security, finance, health, education, news, analytics, automation, productivity, identity, storage, ai-tools, developer-tools, other] }
        search_receipts: { type: integer, minimum: %d, description: Topic rows are omitted below the published privacy threshold. }
        average_organic_position: { type: number, format: double, minimum: 0 }
        result_selections: { type: integer, minimum: 0, nullable: true }
        result_selection_suppressed: { type: boolean }
        action_interest_receipts: { type: integer, minimum: 0, nullable: true }
        action_interest_suppressed: { type: boolean }
    ProviderDemandActionType:
      type: object
      additionalProperties: false
      required: [action_type, receipt_count]
      properties:
        action_type: { type: string, enum: [quote, trial, demo, booking, application, signup, purchase] }
        receipt_count: { type: integer, minimum: %d, description: Action-type rows are omitted below the published privacy threshold. }
    ProviderDemandAnalytics:
      type: object
      additionalProperties: false
      required: [domain, days, retention_days, action_interest_cohort, topic_receipt_threshold, result_selection_receipt_threshold, action_interest_receipt_threshold, synthetic_excluded, summary, surfaces, demand_topics, action_types]
      properties:
        domain: { type: string, description: Domain derived from the authenticated provider claim; never caller-selected. }
        days: { type: integer, minimum: 1, maximum: 30 }
        retention_days: { type: integer, enum: [30] }
        action_interest_cohort: { type: string, enum: [organic_result_returned_at] }
        topic_receipt_threshold: { type: integer, enum: [%d] }
        result_selection_receipt_threshold: { type: integer, enum: [%d] }
        action_interest_receipt_threshold: { type: integer, enum: [%d] }
        synthetic_excluded: { type: boolean, enum: [true] }
        summary: { $ref: "#/components/schemas/ProviderDemandSummary" }
        surfaces:
          type: array
          maxItems: 4
          items: { $ref: "#/components/schemas/ProviderDemandSurface" }
        demand_topics:
          type: array
          maxItems: 20
          items: { $ref: "#/components/schemas/ProviderDemandTopic" }
        action_types:
          type: array
          maxItems: 7
          items: { $ref: "#/components/schemas/ProviderDemandActionType" }
    ProviderDemandResponse:
      type: object
      additionalProperties: false
      required: [demand, evidence_scope]
      properties:
        demand: { $ref: "#/components/schemas/ProviderDemandAnalytics" }
        evidence_scope: { type: string, description: Claim-scoped privacy-thresholded aggregates and explicit redaction boundary. }
    ProviderControlledIntentResolveRequest:
      type: object
      additionalProperties: false
      description: Exact attribution bearer only. Ticket IDs, queries, contact fields, notes, and arbitrary context are not accepted.
      required: [attribution_token]
      properties:
        attribution_token: { type: string, minLength: 1, description: Exact signed bearer returned during ticket preparation and presented during the observed handoff }
    ProviderControlledIntent:
      type: object
      additionalProperties: false
      required: [demand_topic, budget_band, urgency, requirement_flags]
      properties:
        demand_topic: { type: string, enum: [payments, commerce, jobs, data, search, weather, maps, email, messaging, image, video, audio, documents, security, finance, health, education, news, analytics, automation, productivity, identity, storage, ai-tools, developer-tools, other] }
        region_code: { type: string, pattern: "^[A-Z]{2}(-[A-Z0-9]{1,3})?$" }
        budget_band: { type: string, enum: [unspecified, under_100, 100_499, 500_1999, 2000_plus] }
        urgency: { type: string, enum: [unspecified, now, 7_days, 30_days, researching] }
        requirement_flags:
          type: array
          uniqueItems: true
          maxItems: 8
          items: { type: string, enum: [api_access, mcp, sandbox, self_serve, enterprise, compliance, multilingual, human_support] }
    ProviderControlledIntentResolution:
      type: object
      additionalProperties: false
      description: Free read-only provider view authorized separately after an observed handoff. No query, search receipt, identity, contact, network, action URL, pricing, accounting, outcome, or proof fields are returned.
      required: [resolver_contract_version, ticket_id, offer_id, offer_version, action_type, controlled_intent, observed_at, intent_available_until, consent_version]
      properties:
        resolver_contract_version: { type: string, enum: [nhs-provider-controlled-intent-resolver-v1] }
        ticket_id: { type: string, format: uuid }
        offer_id: { type: string, format: uuid }
        offer_version: { type: integer, minimum: 1 }
        action_type: { type: string, enum: [lead, demo, trial, signup, purchase, quote, application, booking] }
        controlled_intent: { $ref: "#/components/schemas/ProviderControlledIntent" }
        observed_at: { type: string, format: date-time }
        intent_available_until: { type: string, format: date-time }
        consent_version: { type: string, enum: [nhs-provider-controlled-intent-disclosure-consent-v1] }
    PublicProviderOffer:
      type: object
      description: Separate disclosed action attached to an exact returned organic site; the provider action URL is withheld until the exact ticket bearer creates an NHS-observed handoff receipt
      required: [id, offer_version, provider_domain, organic_position, name, summary, action_type, disclosure, organic_rank_paid, principal_price, nhs_compensation, commercial_terms_contract_version, commercial_terms_sha256, credit_rule, response_expectation, terms_period_anchor_rule, provider_acknowledges_merchant_of_record, prepare_action_endpoint]
      properties:
        id: { type: string, format: uuid }
        offer_version: { type: integer, minimum: 1 }
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
          required: [event, amount_minor, currency]
          properties:
            event: { type: string, enum: [accepted, activated, converted] }
            amount_minor: { type: integer, minimum: 1 }
            currency: { type: string, enum: [usd] }
        commercial_terms_contract_version: { type: string, enum: [nhs-provider-commercial-terms-v1] }
        commercial_terms_sha256: { type: string, pattern: "^[0-9a-f]{64}$" }
        credit_rule: { type: string, enum: [full_credit_on_provider_reported_invalid_or_duplicate] }
        response_expectation: { type: string, enum: [provider_callback_before_attribution_expiry] }
        terms_period_anchor_rule: { type: string, enum: [billing_period_begins_at_first_activation] }
        provider_acknowledges_merchant_of_record: { type: boolean, enum: [true], description: Provider contractual acknowledgement; NHS does not independently verify Merchant-of-Record status }
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
    ActionTicketPreparationResponse:
      type: object
      additionalProperties: false
      description: Ticket preparation returns a bearer plus the NHS handoff surface, never the provider action URL
      required: [ticket, offer, preparation_contract_version, attribution_token, handoff_endpoint, handoff_method, handoff_event_contract_version, handoff_consent_contract_url, controlled_intent_disclosure_optional, controlled_intent_disclosure_consent_version, controlled_intent_disclosure_consent_url, created, idempotent_replay, attribution_token_stored_by_nhs, token_reconstructed_for_exact_replay, principal_consent_attested, consent_contract_url, principal_charged, provider_mor_contract_required, principal_charged_by_nhs, organic_rank_affected, direct_provider_access_remains_free, disclosure]
      properties:
        ticket: { $ref: "#/components/schemas/PublicActionTicket" }
        offer: { $ref: "#/components/schemas/PublicProviderOffer" }
        preparation_contract_version: { type: string, enum: [nhs-action-ticket-preparation-v2], description: Explicit breaking contract revision that withholds the provider URL until separately consented handoff }
        attribution_token: { type: string, minLength: 1, description: Raw bearer returned in the no-store response. The raw string is not stored in ticket or handoff rows; NHS stores its SHA-256 hash and retains nonce/key metadata plus signing material that can reconstruct an exact replay. }
        handoff_endpoint: { type: string, format: uri }
        handoff_method: { type: string, enum: [POST] }
        handoff_event_contract_version: { type: string, enum: [nhs-action-handoff-v1] }
        handoff_consent_contract_url: { type: string, format: uri }
        controlled_intent_disclosure_optional: { type: boolean, enum: [true], description: The optional separate disclosure may be declined without blocking handoff or free direct provider access }
        controlled_intent_disclosure_consent_version: { type: string, enum: [nhs-provider-controlled-intent-disclosure-consent-v1] }
        controlled_intent_disclosure_consent_url: { type: string, format: uri }
        created: { type: boolean }
        idempotent_replay: { type: boolean }
        attribution_token_stored_by_nhs: { type: boolean, enum: [false] }
        token_reconstructed_for_exact_replay: { type: boolean }
        principal_consent_attested: { type: boolean, enum: [true] }
        consent_contract_url: { type: string, format: uri }
        principal_charged: { type: boolean, enum: [false] }
        provider_mor_contract_required: { type: boolean, enum: [true] }
        principal_charged_by_nhs: { type: boolean, enum: [false] }
        organic_rank_affected: { type: boolean, enum: [false] }
        direct_provider_access_remains_free: { type: boolean, enum: [true] }
        disclosure: { type: string }
    PublicActionTicket:
      type: object
      additionalProperties: false
      description: Controlled consent-attested ticket snapshot. Provider action URL, token hash, token nonce, signing-key metadata, and internal evidence references are excluded.
      required: [id, provider_claim_id, provider_offer_id, offer_version, offer_name, offer_summary, action_type, disclosure, charge_event, bounty_cents, currency, billing_mode, commercial_terms_contract_version, commercial_terms_sha256, principal_price_mode, principal_currency, demand_topic, budget_band, urgency, requirement_flags, principal_consent, consent_version, status, expires_at, created_at, updated_at]
      properties:
        id: { type: string, format: uuid }
        provider_claim_id: { type: string, format: uuid, description: Opaque provider claim identifier; not a provider identity or contact field }
        provider_offer_id: { type: string, format: uuid }
        search_receipt_id: { type: string, format: uuid, description: Removed when controlled intent is redacted }
        offer_version: { type: integer, minimum: 1 }
        offer_name: { type: string }
        offer_summary: { type: string }
        action_type: { type: string, enum: [lead, demo, trial, signup, purchase, quote, application, booking] }
        disclosure: { type: string, enum: [Provider-funded action] }
        charge_event: { type: string, enum: [accepted, activated, converted] }
        bounty_cents: { type: integer, format: int64, minimum: 1 }
        currency: { type: string, enum: [usd] }
        billing_mode: { type: string, enum: [terms] }
        commercial_terms_contract_version: { type: string, enum: [nhs-provider-commercial-terms-v1] }
        commercial_terms_sha256: { type: string, pattern: "^[0-9a-f]{64}$" }
        principal_price_mode: { type: string, enum: [free, fixed, quote, provider_pricing] }
        principal_price_cents: { type: integer, format: int64, minimum: 0 }
        principal_currency: { type: string, enum: [usd] }
        demand_topic: { type: string, enum: [payments, commerce, jobs, data, search, weather, maps, email, messaging, image, video, audio, documents, security, finance, health, education, news, analytics, automation, productivity, identity, storage, ai-tools, developer-tools, other, redacted] }
        region_code: { type: string }
        budget_band: { type: string, enum: [unspecified, under_100, 100_499, 500_1999, 2000_plus] }
        urgency: { type: string, enum: [unspecified, now, 7_days, 30_days, researching] }
        requirement_flags:
          type: array
          uniqueItems: true
          items: { type: string, enum: [api_access, mcp, sandbox, self_serve, enterprise, compliance, multilingual, human_support] }
        principal_consent: { type: boolean, enum: [true] }
        consent_version: { type: string, enum: [nhs-principal-consent-v1] }
        status: { type: string, enum: [created, redirected, accepted, activated, converted, rejected, duplicate, invalid] }
        expires_at: { type: string, format: date-time }
        intent_redacted_at: { type: string, format: date-time }
        authorization_revoked_at: { type: string, format: date-time }
        created_at: { type: string, format: date-time }
        updated_at: { type: string, format: date-time }
    ActionTicketHandoffRequest:
      type: object
      additionalProperties: false
      description: Exact ticket bearer and separate handoff-time principal attestation are accepted only in JSON, not in the NHS URL, query string, referrer, or cookie. The controlled-intent disclosure pair is optional; false or omission requires no version, while true requires the exact v1 version. Declining it does not block handoff.
      required: [ticket_id, attribution_token, principal_handoff_consent, handoff_consent_version]
      properties:
        ticket_id: { type: string, format: uuid }
        attribution_token: { type: string, minLength: 1 }
        principal_handoff_consent: { type: boolean, enum: [true], description: Caller attests the exact published handoff-time principal authorization }
        handoff_consent_version: { type: string, enum: [nhs-provider-handoff-consent-v1] }
        principal_controlled_intent_disclosure_consent: { type: boolean, default: false, description: Optional separate authorization for the exact DNS-verified provider to resolve the bounded controlled-intent bundle after this observed handoff }
        controlled_intent_disclosure_consent_version: { type: string, enum: [nhs-provider-controlled-intent-disclosure-consent-v1], description: Required only when principal_controlled_intent_disclosure_consent is true; otherwise omit }
    ProviderActionHandoffReceipt:
      type: object
      additionalProperties: false
      description: Durable append-only privacy-safe NHS observation. Internal claim ID and presented-token hash are not returned.
      required: [id, action_ticket_id, provider_offer_id, offer_version, commercial_terms_contract_version, commercial_terms_sha256, principal_handoff_consent, handoff_consent_version, principal_controlled_intent_disclosure_consent, event_contract_version, observed_at, created_at]
      properties:
        id: { type: string, format: uuid }
        action_ticket_id: { type: string, format: uuid }
        provider_offer_id: { type: string, format: uuid }
        offer_version: { type: integer, minimum: 1 }
        commercial_terms_contract_version: { type: string, enum: [nhs-provider-commercial-terms-v1] }
        commercial_terms_sha256: { type: string, pattern: "^[0-9a-f]{64}$" }
        principal_handoff_consent: { type: boolean, enum: [true] }
        handoff_consent_version: { type: string, enum: [nhs-provider-handoff-consent-v1] }
        principal_controlled_intent_disclosure_consent: { type: boolean, description: False means no provider resolution authorization; the handoff remains valid }
        controlled_intent_disclosure_consent_version: { type: string, enum: [nhs-provider-controlled-intent-disclosure-consent-v1] }
        event_contract_version: { type: string, enum: [nhs-action-handoff-v1] }
        observed_at: { type: string, format: date-time }
        created_at: { type: string, format: date-time }
    ActionTicketHandoffResponse:
      type: object
      additionalProperties: false
      required: [ticket, handoff_receipt, action_url, observed_handoff, idempotent_replay, principal_charged, provider_charged, organic_rank_affected, direct_provider_access_is_free]
      properties:
        ticket: { $ref: "#/components/schemas/PublicActionTicket" }
        handoff_receipt: { $ref: "#/components/schemas/ProviderActionHandoffReceipt" }
        action_url: { type: string, format: uri, description: Attributed HTTPS provider URL returned only after the durable receipt commits }
        observed_handoff: { type: boolean, enum: [true] }
        idempotent_replay: { type: boolean }
        principal_charged: { type: boolean, enum: [false] }
        provider_charged: { type: boolean, enum: [false] }
        organic_rank_affected: { type: boolean, enum: [false] }
        direct_provider_access_is_free: { type: boolean, enum: [true] }
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
		int64(models.ProviderClaimVerificationFreshness/time.Second),
		models.ProviderDemandPrivacyThreshold,
		models.ProviderDemandPrivacyThreshold,
		models.ProviderDemandPrivacyThreshold,
		models.ProviderDemandPrivacyThreshold,
		models.ProviderDemandPrivacyThreshold)
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
