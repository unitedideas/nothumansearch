# Regional Marketplace and Travel Commerce Source Readiness - 2026-05-27

Automation: `business-marketer-not-human-search`
Status: no-submit scout artifact. Public use still requires active account identity verification, duplicate checks, and a sync-state public-action lock.

## Boundary

No outreach, public post, browser action, Computer Use, account creation, product-code edit, deploy, full recrawl, checkout completion, raw customer readout, or QLimit/global-queue write was performed.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, private query logs, raw user-agent strings, buyer data, or customer identifiers are included here.

## Evidence

- Public stats: `total_sites=4174`, `avg_score=35`, and `top_category=developer`.
- Public categories: `developer=1231`, `ai-tools=905`, `other=773`, `data=402`, `finance=192`, `productivity=171`, `ecommerce=149`, `communication=118`, `security=113`, `health=60`, `jobs=26`, `education=21`, `news=12`, and `spam=1`.
- Public ecommerce top-list examples included `budgetfitter.uk=100`, `rettfrabonden.com=100`, `skillboss.co=100`, `ai.immoswipe.ch=95`, `can-tap-verified.com=80`, `la-palma24.net=75`, `businesshotels.com=75`, `store.farcomindustrial.com=75`, `maplebridge.io=70`, `photo-fotograf.com=70`, `kismet.travel=65`, and `freetv-app.com=60`. Treat these as public readiness examples only.
- Public AI-tools and developer top lists still contain Foundry-owned dogfood examples, so owner-channel copy should label or omit owned examples before publication.
- Aggregate MCP query themes included `Daraz Pakistan`, local events, restaurant search, ecommerce product search, outdoor gear reviews, refrigerator reviews, ETF market-data, model-gateway pricing, local AI runtimes, and WeChat/RSS monitoring. Treat these as aggregate theme signals only, not private demand.
- Representative public profile and route checks: `/site/daraz.pk` returned 200 with score `25/100`; `/fix/daraz.pk` returned 404, so direct paid remediation is not proven for that host. High-score ecommerce examples such as `budgetfitter.uk` and `ai.immoswipe.ch` routed through monitor-style proof on `/fix/{host}`. Other partial-score commerce and travel examples need `/score` plus a missing-surface checklist before remediation copy.
- Live public surfaces returned 200 for `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`, `/api/v1/api-keys/subscribe`, `/score`, `/monitor`, `/report`, `/openapi.yaml`, `/feed.xml`, and `/mcp` JSON-RPC `tools/list`.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card claims remain blocked.
- Live MCP `tools/list` returned 11 tools: `search_agents`, `get_site_details`, `get_stats`, `submit_site`, `check_url`, `verify_mcp`, `register_monitor`, `list_categories`, `get_top_sites`, `recent_additions`, and `find_mcp_servers`.
- Aggregate MCP analytics, 7 days: `tools/list=182702`, `initialize=28434`, and `tools/call=392`.
- Aggregate MCP tool calls, 7 days: `search_agents=144`, `check_url=89`, `get_site_details=67`, `get_stats=27`, `submit_site=20`, `verify_mcp=13`, `find_mcp_servers=10`, `recent_additions=6`, `list_categories=6`, `get_top_sites=6`, and `register_monitor=4`.
- Aggregate traffic, 168 hours: `/=3403`, `/badge/xquik.com.svg=2645`, `/.well-known/commerce.json=1315`, `/site/xquik.com=1107`, `/.well-known/ai-plugin.json=584`, `/llms.txt=431`, `/openapi.yaml=377`, `/api/v1/catalog=317`, `/badge/aidevboard.com.svg=290`, `/robots.txt=281`, `/badge/8bitconcepts.com.svg=266`, `/api/v1/quote=250`, `/api/v1/checkout=250`, `/api/v1/search=232`, `/api/v1/submit=146`, `/about=94`, `/digest=90`, `/score=77`, and `/.well-known/agent.json=77`.
- Aggregate referrers, 168 hours: Google contributed 640 visits and `/score` contributed 80 visits. Treat these as aggregate discovery and score-flow signals only.
- Latest local monitor worker proof, 2026-05-25: completed normally with five due monitors. Aggregate outcome was two first-check zero-score quarantines, two first-check partial or low-score checks, and one stable high-score check.

## Segment

This is narrower than the older ecommerce, product-review, local-service, local-restaurant, and travel-adjacent briefs. The useful segment is regional marketplace and travel-commerce owners whose pages agents may use for product discovery, availability, booking, seller selection, or local inventory.

The safe owner-side contract is:

- `llms.txt` with marketplace scope, geography, seller/product/booking boundaries, update cadence, non-coverage boundaries, contact, and escalation path.
- Schema.org for Organization, Product, Offer, AggregateOffer, Service, LocalBusiness, Place, Event, Review only where owner-controlled, and ContactPoint.
- API, feed, catalog, OpenAPI, or MCP metadata only where the owner intends agents or partners to read structured product, seller, booking, availability, price, or itinerary data.
- Robots policy that clearly states whether major AI crawlers are allowed.
- Free monitor/report/badge proof for high-score owners.
- `/score` plus a missing-surface checklist before remediation for partial-score owners.

## Draft Brief

Agents shopping across a regional marketplace, booking a hotel, checking local availability, or comparing products need a source contract, not just a page to scrape.

For regional marketplace and travel-commerce owners, the machine-readable surface should tell an agent which product, seller, offer, inventory, availability, booking, price, review, geography, and support sources are canonical. Not Human Search does not certify the seller, product, inventory, price, or booking. It checks whether an agent can inspect the public source before trusting it.

High-score owners should attach the public report, badge, and free monitor path. Partial-score owners should run `/score`, fix missing public source contracts, and only then consider remediation.

## Owner Routing

- High-score regional marketplace and travel-commerce surfaces: route to free monitor registration, public report sharing, and badge/report proof.
- Partial-score owners: route to `/score` first, then a missing-surface checklist before score-fix remediation.
- Low-score hosts where `/fix/{host}` returns 404: do not route to paid remediation until a product/operator check proves the correct handoff.
- Directory and API-heavy callers: route to API-key/catalog surfaces only when NHS docs remain useful and API-plan price metadata is not overstated.
- A2A or Agent Card claims stay blocked until `/.well-known/agent-card.json` exists.

## Gated Use

Use this for exactly one gated owner-channel touch, channel post, directory candidate, or product-handoff test for regional marketplaces, travel-commerce sites, booking products, product-search engines, seller directories, local inventory products, or marketplace API owners.

Before external use, refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=ecommerce&limit=12`, `/api/v1/top?category=other&limit=8`, `/score`, `/monitor`, `/report`, representative `/site/{host}` pages, high-score and partial-score `/fix/{host}` routes, `/mcp` JSON-RPC `tools/list`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`, `/api/v1/api-keys/subscribe`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, `/feed.xml`, aggregate `/api/v1/admin/mcp?days=7`, aggregate `/api/v1/admin/traffic?hours=168`, and latest monitor worker proof.

Before public use, verify the active Foundry/Owl-owned account identity, check `marketing/social-post-ledger.json` if present plus `outreach/distribution_log.csv` and sync-state public-action locks, and avoid `modelcontextprotocol/*` plus `punkpeye/*` surfaces from `unitedideas`.

## Claims To Avoid

Do not imply regional marketplace, travel-commerce, booking, ecommerce, product-search, local inventory, seller-directory, top-list, query-theme, referrer, badge, or profiled domains are customers, partners, endorsements, paid leads, monitor registrations, badge-install consent, private demand, completed payments, revenue, seller quality, product quality, hotel quality, inventory accuracy, price freshness, booking availability, travel fulfillment, delivery reliability, review truth, service-area coverage, legal/regulatory compliance, safety, accessibility compliance, crawler compliance, SEO lift, uptime proof, A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/SPT/MPP support for NHS, paid ranking placement, preferred inclusion, or score-methodology bypass.

Do not publish raw user-agent strings, raw MCP queries, private monitor rows, raw checkout URLs, payment identifiers, buyer emails, private score-fix rows, or private customer identifiers.
