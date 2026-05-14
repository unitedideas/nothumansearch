# NHS Agent-Commerce Directory Packet

Status: prepared, not submitted.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-14T11:08Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a no-submit packet for a later gated operator.

## Live Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4170`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/.well-known/commerce.json`: advertises agentic payment readiness, catalog, quote, checkout, API subscription, activation, supported Stripe Checkout/Link/SPT modes, and explicit unsupported ACP/x402/MPP modes.
- `https://nothumansearch.ai/.well-known/agent.json`: advertises agentic-readiness search, MCP server discovery, GEO uplift service, paid API keys, commerce metadata, and MCP endpoint.
- `https://nothumansearch.ai/api/v1/catalog`: lists score-fix plus Starter, Pro, and Scale API subscriptions.
- `https://nothumansearch.ai/api/v1/quote`: returns an API Starter quote with `$19/mo`, 1,000 monthly billable REST/MCP calls, POST checkout contract, and activation endpoint.
- `https://nothumansearch.ai/api/v1`: lists commerce catalog, quote, checkout, API-key plans, subscribe, and activation routes.

Aggregate traffic proof from admin analytics, last 336 hours:

- `/.well-known/commerce.json`: 1,229 requests.
- `/api/v1/catalog`: 275 requests.
- `/api/v1/checkout`: 260 requests.
- `/api/v1/quote`: 260 requests.
- `/.well-known/ai-plugin.json`: 622 requests.
- `/llms.txt`: 417 requests.
- `/openapi.yaml`: 409 requests.

No raw users, API keys, checkout URLs, payment identifiers, or private query logs were written.

## Duplicate/History Check

`outreach/distribution_log.csv` already contains broad MCP/API/GEO distribution: official registry, Glama/PulseMCP/APIs.guru/mcpservers.org-adjacent work, GitHub PRs, gists, newsletter pitches, RSS/content submissions, and score-check-action distribution.

It does not show a completed agent-commerce-specific submission packet for NHS machine-readable buying surfaces. Older scout rows queued this idea before the catalog and commerce surfaces were fully repaired; this packet uses the current live state.

## Directory Families To Prepare

1. Agent-commerce and seller-readiness registries

Fit: directories that list sellers or APIs with machine-readable product catalogs, quotes, checkout handoffs, or agent-payment metadata.

Proof links:

- `https://nothumansearch.ai/.well-known/commerce.json`
- `https://nothumansearch.ai/.well-known/agent.json`
- `https://nothumansearch.ai/api/v1/catalog`
- `https://nothumansearch.ai/api/v1/quote`
- `https://nothumansearch.ai/api/v1/checkout`

Submission angle:

Not Human Search is both an agent-readiness search engine and a dogfooded agent-readable seller. Agents can discover the product catalog, quote API plans, start Stripe Checkout, and see unsupported payment rails explicitly instead of guessing.

2. API marketplace and developer-tool directories

Fit: API directories that accept paid API products, OpenAPI-backed tools, or agent-facing developer APIs.

Proof links:

- `https://nothumansearch.ai/api/v1`
- `https://nothumansearch.ai/openapi.yaml`
- `https://nothumansearch.ai/api/v1/catalog`
- `https://nothumansearch.ai/.well-known/agent.json`

Submission angle:

NHS offers a searchable agent-readiness index through REST and MCP, with paid API-key plans exposed in a machine-readable catalog. Do not frame this as generic website search or paid ranking.

3. Agent-discovery and MCP directories with commerce fields

Fit: MCP/API directories where the listing can include the MCP endpoint plus the seller catalog or agent manifest.

Proof links:

- `https://nothumansearch.ai/mcp`
- `https://nothumansearch.ai/.well-known/mcp.json`
- `https://nothumansearch.ai/.well-known/agent.json`
- `https://nothumansearch.ai/.well-known/commerce.json`

Submission angle:

NHS helps agents discover agent-ready sites and exposes its own MCP and commerce surfaces as dogfood. The listing should emphasize readiness scoring, MCP discovery, and machine-readable checkout metadata. Avoid the MCP-org and punkpeye surfaces from `unitedideas`.

## Draft Listing Copy

Short:

Not Human Search is an agent-readiness search engine for websites and APIs. It ranks sites by public machine-readable signals, exposes REST and MCP access, and dogfoods agent-commerce surfaces through `commerce.json`, `agent.json`, catalog, quote, and checkout endpoints.

Long:

Not Human Search helps AI agents and builders find sites that are actually usable without scraping. Each indexed site is scored on public agent-readiness signals such as `llms.txt`, OpenAPI, structured APIs, MCP, AI-friendly robots policy, and Schema.org.

The product is also agent-readable as a seller: agents can inspect the public catalog, quote score-fix and API subscription products, start Stripe Checkout, and see unsupported rails such as ACP/x402 called out explicitly rather than inferred.

Primary links:

- Site: `https://nothumansearch.ai`
- REST API: `https://nothumansearch.ai/api/v1`
- OpenAPI: `https://nothumansearch.ai/openapi.yaml`
- MCP: `https://nothumansearch.ai/mcp`
- Agent manifest: `https://nothumansearch.ai/.well-known/agent.json`
- Commerce manifest: `https://nothumansearch.ai/.well-known/commerce.json`

## Publication Guard

Before any external submission:

- Refresh `/api/v1/stats`, `/.well-known/commerce.json`, `/.well-known/agent.json`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1`, `/llms.txt`, and `/.well-known/mcp.json`.
- Check `outreach/distribution_log.csv` and the shared public-action lock store for duplicate submissions.
- Verify the active account identity for the selected directory/channel.
- Take the required sync-state public-action lock.
- Do not create accounts, use browser/Computer Use, or submit publicly from a recurring automation worker.
- Do not claim private demand, completed payments, revenue, certification, paid ranking placement, preferred inclusion, ACP/x402 support, or score-methodology bypass.
