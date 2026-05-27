# Local Service Marketplace Source Readiness - 2026-05-27

Automation: `business-marketer-not-human-search`
Status: no-submit scout artifact. Public use still requires active account identity verification, duplicate checks, and a sync-state public-action lock.

## Boundary

No outreach, public post, browser action, Computer Use, account creation, product-code edit, deploy, full recrawl, checkout completion, raw customer readout, or QLimit/global-queue write was performed.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, private query logs, raw user-agent strings, buyer data, or customer identifiers are included here.

## Evidence

- Public stats: `total_sites=4174`, `avg_score=35`, and `top_category=developer`.
- Public categories: `developer=1231`, `ai-tools=905`, `other=774`, `data=402`, `finance=192`, `productivity=171`, `ecommerce=149`, `communication=118`, `security=113`, `health=59`, `jobs=26`, `education=21`, `news=12`, and `spam=1`.
- Public `other` top-list examples included `astranl.com=100`, `lobehub.com=95`, `agentgrade.com=80`, `surprise-buddy.com=65`, `infinity-folder.org=65`, `twzrd.xyz=55`, `proshares.com=50`, and `crabbitmq.com=50`. Treat these as public readiness examples only.
- Representative public profile checks: `astranl.com=100`, `lobehub.com=95`, `agentgrade.com=80`, and `surprise-buddy.com=65`.
- Representative score-fix route checks: high-score `astranl.com` returned the already-meets-target handoff; partial-score examples did not expose high-score routing and need `/score` plus a missing-surface checklist before any remediation copy.
- Public ecommerce top-list examples included `budgetfitter.uk=100`, `rettfrabonden.com=100`, `skillboss.co=100`, `ai.immoswipe.ch=95`, `can-tap-verified.com=80`, `businesshotels.com=75`, `store.farcomindustrial.com=75`, and `la-palma24.net=75`. Treat these as adjacent public readiness examples, not customers or endorsements.
- Live public surfaces returned 200 for `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`, `/api/v1/api-keys/subscribe`, `/score`, `/monitor`, `/report`, `/openapi.yaml`, `/feed.xml`, and `/mcp` JSON-RPC `tools/list`.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card claims remain blocked.
- Live MCP `tools/list` returned 11 tools: `search_agents`, `get_site_details`, `get_stats`, `submit_site`, `check_url`, `verify_mcp`, `register_monitor`, `list_categories`, `get_top_sites`, `recent_additions`, and `find_mcp_servers`.
- Aggregate MCP analytics, 7 days: `tools/list=181985`, `initialize=28417`, and `tools/call=402`.
- Aggregate MCP tool calls, 7 days: `search_agents=149`, `check_url=89`, `get_site_details=67`, `get_stats=30`, `submit_site=20`, `verify_mcp=13`, `find_mcp_servers=10`, `list_categories=8`, `recent_additions=6`, `get_top_sites=6`, and `register_monitor=4`.
- Sanitized aggregate query themes included local events, agent marketplace payments, local AI runtimes, AI agent jobs, WeChat/RSS monitoring, genetic wellness, model gateways, product-review/source discovery, ecommerce product search, ETF market data, function-calling pricing, and web search.
- Aggregate traffic, 168 hours: `/=3383`, `/badge/xquik.com.svg=2645`, `/.well-known/commerce.json=1331`, `/site/xquik.com=1106`, `/.well-known/ai-plugin.json=591`, `/llms.txt=436`, `/openapi.yaml=379`, `/api/v1/catalog=321`, `/badge/aidevboard.com.svg=287`, `/robots.txt=281`, `/badge/8bitconcepts.com.svg=264`, `/api/v1/checkout=253`, `/api/v1/quote=253`, `/api/v1/search=230`, `/api/v1/submit=146`, `/about=91`, and `/digest=89`.
- Aggregate referrers, 168 hours: Google contributed 640 visits and `/score` contributed 79 visits. Treat these as aggregate discovery and score-flow signals only.
- Latest local monitor worker proof, 2026-05-25: completed normally with five due monitors. Aggregate outcome was two first-check zero-score quarantines, two first-check partial or low-score checks, and one stable high-score check.

## Segment

This is narrower than the older `other` bucket, local-events, restaurant, and ecommerce briefs. The useful segment is local-service, home-service, recommendation, gift-finder, marketplace, and booking-style owners whose pages may be used by agents to select a provider, recommend a product, or route a human to a local service.

The safe owner-side contract is:

- `llms.txt` with source scope, geography, update cadence, non-coverage boundaries, contact, and escalation path.
- Schema.org for LocalBusiness, Service, Offer, Product, Review only where owner-controlled, Organization, and ContactPoint.
- API, feed, catalog, OpenAPI, or MCP metadata only where the owner intends agents or partners to read structured provider, product, booking, quote, or availability data.
- Robots policy that clearly states whether major AI crawlers are allowed.
- Free monitor/report/badge proof for high-score owners.
- `/score` plus a missing-surface checklist before any paid remediation for partial-score owners.

## Draft Brief

Agents recommending a contractor, gift, venue, product, or local service need a source contract, not just a page to scrape.

For local-service marketplaces and recommendation products, the machine-readable surface should tell an agent which provider, product, quote, booking, geography, contact, review, and update sources are canonical. Not Human Search does not certify the provider or recommendation. It checks whether an agent can inspect the public source before trusting it.

High-score owners should attach the public report, badge, and free monitor path. Partial-score owners should run `/score`, fix missing public source contracts, and only then consider remediation.

## Owner Routing

- High-score local-service, marketplace, recommendation, and booking surfaces: route to free monitor registration, public report sharing, and badge/report proof.
- Partial-score owners: route to `/score` first, then a missing-surface checklist before score-fix remediation.
- Directory and API-heavy callers: route to API-key/catalog surfaces only when NHS docs remain useful and API-plan price metadata is not overstated.
- A2A or Agent Card claims stay blocked until `/.well-known/agent-card.json` exists.

## Gated Use

Use this for exactly one gated owner-channel touch, channel post, directory candidate, or product-handoff test for local-service marketplaces, home-service products, gift/recommendation engines, booking products, local commerce, or service-provider directories.

Before external use, refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=other&limit=8`, `/api/v1/top?category=ecommerce&limit=8`, `/score`, `/monitor`, `/report`, representative `/site/{host}` pages, high-score and partial-score `/fix/{host}` routes, `/mcp` JSON-RPC `tools/list`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`, `/api/v1/api-keys/subscribe`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, `/feed.xml`, aggregate `/api/v1/admin/mcp?days=7`, aggregate `/api/v1/admin/traffic?hours=168`, and latest monitor worker proof.

Before public use, verify the active Foundry/Owl-owned account identity, check `marketing/social-post-ledger.json` if present plus `outreach/distribution_log.csv` and sync-state public-action locks, and avoid `modelcontextprotocol/*` plus `punkpeye/*` surfaces from `unitedideas`.

## Claims To Avoid

Do not imply local-service, marketplace, recommendation, gift, booking, ecommerce, top-list, referrer, badge, or profiled domains are customers, partners, endorsements, paid leads, monitor registrations, badge-install consent, private demand, completed payments, revenue, provider quality, contractor reliability, gift quality, product quality, booking availability, quote accuracy, price freshness, review truth, inventory accuracy, delivery reliability, service-area coverage, legal/regulatory compliance, safety, accessibility compliance, crawler compliance, SEO lift, uptime proof, A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/SPT/MPP support for NHS, paid ranking placement, preferred inclusion, or score-methodology bypass.

Do not publish raw user-agent strings, raw MCP queries, private monitor rows, raw checkout URLs, payment identifiers, buyer emails, private score-fix rows, or private customer identifiers.
