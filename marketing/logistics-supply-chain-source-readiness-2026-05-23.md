# Logistics and Supply-Chain Source Readiness - 2026-05-23

Run context: `business-marketer-not-human-search` recurring scout. No outreach, posting, browser, Computer Use, deploy, product-code edit, full recrawl, account creation, checkout completion, or QLimit/global-queue write was performed.

## Fresh aggregate signal

- Public stats: `/api/v1/stats` returned 200 with `total_sites=4174`, `avg_score=35`, and `top_category=developer`.
- Public category counts: `developer=1229 avg_score=34`, `data=402 avg_score=32`, `ecommerce=148 avg_score=41`, and `ai-tools=897 avg_score=41`.
- Aggregate MCP analytics over 7 days: `tools/list=170120`, `initialize=27847`, `tools/call=225`.
- Top MCP tool calls: `search_agents=110`, `get_site_details=35`, `check_url=32`, `get_stats=19`, `list_categories=6`, `get_top_sites=6`, `recent_additions=6`, `find_mcp_servers=5`, `submit_site=4`, `verify_mcp=2`.
- Aggregate traffic over 168 hours: `/=3364`, `/badge/xquik.com.svg=2527`, `/.well-known/commerce.json=1436`, `/site/xquik.com=937`, `/.well-known/ai-plugin.json=663`, `/llms.txt=441`, `/openapi.yaml=403`, `/api/v1/catalog=318`, `/robots.txt=304`, `/api/v1/checkout=280`, `/api/v1/quote=280`, `/api/v1/search=193`, `/api/v1/submit=143`, `/about=98`, `/top=96`.
- Live discovery checks returned 200 for `/score`, `/monitor`, `/report`, `/newest`, `/top`, `/mcp-servers`, `/openapi-apis`, `/llms-txt-sites`, `/api/v1`, `/api/v1/catalog`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/llms.txt`, `/openapi.yaml`, and `/feed.xml`.
- `/.well-known/agent-card.json` returned 404, so strict Agent Card and A2A-style claims remain gated.

## Public examples

Current public top lists expose a narrower logistics and supply-chain owner segment across developer, data, ecommerce, and AI-tool categories:

| Domain | Score | Public readiness pattern | Safe owner route |
|---|---:|---|---|
| `wearewarp.com` | 100 | Freight network with lane, carrier, cross-dock, and pricing claims plus full public readiness signals. | Monitor/report/badge proof; do not claim carrier reliability or freight performance. |
| `api.meacheal.ai` | 85 | Apparel supply-chain data API with llms.txt, plugin, OpenAPI, API, and schema; no MCP in the public top response. | `/score` first, then missing-surface checklist. |
| `packrift.com` | 80 | Ecommerce, shipping, packaging, and fulfillment supplies with llms.txt and MCP present; structured API missing in the public top response. | `/score` first, then missing catalog/API surface checklist. |
| `api.boostedchat.com` | 95 | Travel booking API/MCP surface with strong machine-readable contracts; schema missing in the public top response. | Monitor/report proof; do not claim travel price accuracy or fulfillment quality. |
| `businesshotels.com` | 75 | Booking/catalog-style surface with llms.txt and OpenAPI; structured API and MCP missing in the public top response. | `/score` first, then source-contract checklist. |
| `store.farcomindustrial.com` | 75 | Industrial supplies/catalog surface with llms.txt and OpenAPI; structured API and MCP missing in the public top response. | `/score` first, then catalog/API checklist. |

These are public readiness examples or owner-channel targets only. They are not customers, endorsements, paid leads, monitor registrations, badge-install consent, completed payments, revenue, private demand, or market-share proof.

## Useful angle

Logistics, freight, travel booking, supply-chain data, and fulfillment/catalog owners are a good score-band segment because agents need source contracts before they act on operational data:

1. Stable public contracts for lanes, capacity, SKUs, booking paths, fulfillment boundaries, rate/pricing metadata, support contacts, and update windows.
2. `llms.txt`, OpenAPI/API, MCP where real, catalog/feed metadata where relevant, and robots policy.
3. Monitorable drift when deploys remove manifests, API paths, MCP endpoints, schema, or catalog files.
4. Score-band-aware routing: high-score owners to free monitor/report/badge proof; partial-score owners to `/score` before remediation.

Safe short copy:

`Agents that touch freight, booking, fulfillment, or supply-chain data need more than a human landing page. They need to inspect the public source contract: llms.txt, OpenAPI/API, real MCP if present, catalog or feed metadata, robots policy, and a public profile that can be monitored for drift. NHS scores that surface and routes high-score owners to monitor/report proof, with remediation reserved for concrete missing contracts.`

## Gated use

Use this for exactly one gated owner-channel touch, post, or product-handoff test for logistics, freight, travel-booking, supply-chain-data, industrial-catalog, packaging, or fulfillment owners. Refresh the live evidence before external use.

Required refresh:

- `/api/v1/stats`, `/api/v1/categories`
- `/api/v1/top?category=developer&limit=12`, `/api/v1/top?category=data&limit=12`, `/api/v1/top?category=ecommerce&limit=12`, `/api/v1/top?category=ai-tools&limit=12`
- `/score`, `/monitor`, `/report`, representative `/site/{host}` pages
- High-score and partial-score `/fix/{host}` routes
- `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`
- `/api/v1`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, `/feed.xml`
- Aggregate `/api/v1/admin/mcp?days=7` and `/api/v1/admin/traffic?hours=168`

Before public use, verify the active Foundry/Owl-owned account identity, check `marketing/social-post-ledger.json` if present, `outreach/distribution_log.csv`, and sync-state public-action locks, and avoid `modelcontextprotocol/*` plus `punkpeye/*` surfaces from `unitedideas`.

Do not imply logistics, freight, travel, supply-chain, packaging, industrial-catalog, ecommerce, or profiled domains are customers, partners, endorsements, paid leads, monitor registrations, badge-install consent, private demand, completed payments, revenue, carrier reliability, freight performance, lane coverage, booking availability, travel fulfillment, travel price accuracy, supply-chain accuracy, inventory accuracy, delivery reliability, operational safety, customs/legal compliance, privacy compliance, crawler compliance, SEO lift, uptime proof, A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/MPP support for NHS, paid ranking placement, preferred inclusion, or score-methodology bypass. Do not publish raw user-agent strings, private query logs, raw checkout URLs, payment identifiers, buyer emails, private monitor rows, private score-fix rows, or private customer identifiers.
