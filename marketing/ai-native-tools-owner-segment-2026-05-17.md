# AI-Native Tools Owner Segment

Date: 2026-05-17
Automation: `business-marketer-not-human-search`
Status: prepared, not published

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a sanitized scout segment for a later gated owner-channel operator.

## Fresh Evidence

Public surfaces checked:

- `https://nothumansearch.ai/api/v1/stats`: 4,174 indexed sites, average score 35, top category `developer`.
- `https://nothumansearch.ai/api/v1/categories`: `ai-tools=900`, average score 40; adjacent public buckets include `developer=1235`, `other=767`, `data=399`, `finance=199`, `productivity=173`, `ecommerce=149`, `communication=119`, and `security=115`.
- `https://nothumansearch.ai/api/v1/top?category=ai-tools&limit=8`: public top-list source for this segment.
- `https://nothumansearch.ai/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, `/openapi.yaml`, `/llms.txt`, `/score`, and `/monitor`: HTTP 200.
- `https://nothumansearch.ai/.well-known/agent-card.json`: HTTP 404, so strict Agent Card and A2A-style directory submissions remain gated.

Aggregate admin signals, sanitized:

- MCP analytics, last 7 days: `tools/list=140097`, `initialize=18964`, `tools/call=279`.
- Top called MCP tools: `search_agents=181`, `get_site_details=35`, `check_url=12`, `get_stats=12`, `verify_mcp=12`, `recent_additions=9`, `get_top_sites=8`, `find_mcp_servers=7`, `list_categories=2`, `submit_site=1`.
- Aggregate traffic, last 7 days: `/.well-known/commerce.json=1588`, `/llms.txt=483`, `/openapi.yaml=437`, `/api/v1/catalog=355`, `/api/v1/checkout=330`, `/api/v1/quote=330`, `/.well-known/mcp.json=85`, `/api/v1=82`.
- Aggregate referrers include Google traffic, but no private user identifiers are included here.
- Errors last hour: 0.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, private query logs, or raw buyer data are included here.

## Public AI-Tools Examples

These are public top-list examples only. Treat them as owner-channel targets or readiness-pattern examples, not customers, endorsements, paid leads, private demand, completed purchases, or proof of market share.

| Domain | Score | Owner-channel angle | Boundary |
| --- | ---: | --- | --- |
| `8bitconcepts.com` | 100 | Foundry-owned dogfood reference for full readiness and service-offer metadata. | Label as Foundry-owned; do not use as third-party proof. |
| `chainray.online` | 100 | Third-party AI-agent data service with full readiness signals and agent-commerce-like positioning. | Do not claim pricing accuracy, chain coverage, x402 functionality, or endorsement. |
| `bringyour.ai` | 100 | Foundry-owned dogfood reference for full readiness and harness/agent-tool distribution. | Label as Foundry-owned; do not use as third-party proof. |
| `nothumansearch.ai` | 100 | Foundry-owned dogfood reference for the search/MCP/API surface itself. | Label as Foundry-owned; do not use as third-party proof. |
| `memestack.ai` | 100 | Third-party AI-native media/search surface with full readiness signals. | Do not claim content quality, copyright posture, or private demand. |
| `sincetmw.ai` | 100 | Third-party trend/culture search surface with full readiness signals. | Do not claim data freshness, platform coverage, or market authority. |
| `teenanxiety.ai` | 100 | High-score health-content surface currently categorized as AI tools. | Label as health-adjacent/misbucket-risk; do not claim clinical endorsement. |
| `teenadhd.ai` | 100 | High-score health-content surface currently categorized as AI tools. | Label as health-adjacent/misbucket-risk; do not claim clinical endorsement. |

## Segment Read

The AI-tools bucket is large enough for owner-channel work, but the public top list is not clean enough for broad market copy. It includes Foundry-owned dogfood surfaces and health-content sites that may be better handled as health/education or taxonomy-cleanup examples.

Safe use:

1. AI-native product owners with strong readiness signals: monitor/report/badge proof.
2. Third-party AI data, media, search, and agent-service owners: score page first, then remediation only when a public readiness signal is missing.
3. Foundry-owned examples: label as dogfood and use only to show the readiness pattern, not as market proof.
4. Health-adjacent AI examples: keep clinical and compliance boundaries explicit or route to a health-data segment instead.

Unsafe use:

- Generic "top AI tools" ranking copy.
- Claims of private demand, customer endorsement, clinical endorsement, model quality, copyright safety, data freshness, pricing accuracy, completed purchases, revenue, paid placement, or preferred inclusion.
- A2A support claims while `/.well-known/agent-card.json` returns 404.

## Draft Operator Copy

`Not Human Search checks whether AI-native products expose public surfaces agents can inspect: llms.txt, OpenAPI, structured APIs, MCP, AI-friendly robots rules, plugin metadata, and schema.`

`For high-scoring AI tools, the next step is usually monitoring and a shareable report. For tools with missing surfaces, the score page shows the gap before any remediation offer.`

Proof links:

- `https://nothumansearch.ai/top?category=ai-tools`
- `https://nothumansearch.ai/score`
- `https://nothumansearch.ai/monitor`
- `https://nothumansearch.ai/.well-known/mcp.json`
- `https://nothumansearch.ai/api/v1/catalog`
- `https://nothumansearch.ai/llms.txt`

## Publication Guard

Before any external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=ai-tools&limit=8`, representative `/site/{host}` profiles, `/score`, `/monitor`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, and aggregate `/api/v1/admin/mcp?days=7`.
- Verify the active Foundry/Owl-owned account identity for the selected channel.
- Check `marketing/social-post-ledger.json` if present, `outreach/distribution_log.csv`, and sync-state public-action locks.
- Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.

## Do Not Claim

- Private demand, customers, endorsements, paid leads, completed payments, revenue, product quality, clinical endorsement, compliance certification, model quality, copyright safety, data freshness, pricing accuracy, seller certification, x402/ACP/MPP support for NHS, paid ranking placement, preferred inclusion, A2A support, or score-methodology bypass.
