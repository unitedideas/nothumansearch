# Recent additions owner onboarding - Packrift cluster and adjacent sources

Run context: `business-marketer-not-human-search` recurring scout on 2026-05-23. No outreach, posting, browser, Computer Use, deploy, product-code edit, full recrawl, account creation, checkout completion, or QLimit/global-queue write was performed.

## Fresh aggregate signal

- Public stats: `/api/v1/stats` returned 200 with `total_sites=4171`, `avg_score=35`, and `top_category=developer`.
- Public category counts: `developer=1228`, `ai-tools=897`, `other=777`, `data=402`, `finance=194`, `productivity=174`, `ecommerce=148`, `communication=118`, `security=114`, `health=58`, `jobs=27`, `education=21`, `news=12`.
- Aggregate MCP analytics over 7 days: `tools/list=169705`, `initialize=27787`, `tools/call=225`.
- Top MCP tool calls: `search_agents=108`, `get_site_details=37`, `check_url=32`, `get_stats=19`, `list_categories=6`, `get_top_sites=6`, `recent_additions=6`, `find_mcp_servers=5`, `submit_site=4`, `verify_mcp=2`.
- Visible aggregate query themes included Singapore news/housing, USDC agent marketplaces, AI skill publishing, scanner/electronics retail, health/wellness data, model/API pricing, secrets management, Hermes/agent skills, and outdoor/product reviews.
- Aggregate traffic over 168 hours: `/=3368`, `/badge/xquik.com.svg=2489`, `/.well-known/commerce.json=1454`, `/site/xquik.com=903`, `/.well-known/ai-plugin.json=671`, `/llms.txt=443`, `/openapi.yaml=407`, `/api/v1/catalog=320`, `/robots.txt=305`, `/api/v1/quote=284`, `/api/v1/checkout=284`, `/api/v1/search=177`, `/favicon.ico=175`, `/api/v1/submit=142`, `/about=99`, `/top=95`, `/api/v1=88`, `/.well-known/mcp.json=88`.
- Live discovery checks returned 200 for `/score`, `/monitor`, `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/api/v1/catalog`, `/openapi.yaml`, `/feed.xml`, `/newest`, `/top`, `/mcp-servers`, and `/openapi-apis`.
- `/.well-known/agent-card.json` returned 404, so strict Agent Card and A2A-style claims remain gated.

## Recent public examples

`/newest` is the useful public source for this segment. Anonymous `/api/v1/newest` and `/api/v1/recent` returned 404, so later operators should use the public page or an approved internal/API-key path before claiming API support for newest lists.

Recent visible profiles showed a compact owner-onboarding pattern:

| Domain | Public score | Segment | Safe owner route |
|---|---:|---|---|
| `packrift.com` | 80 | Ecommerce, shipping, packaging, fulfillment supplies. | `/score` first, then missing-surface remediation if the owner wants complete agent-readiness. |
| `mcp.packrift.com` | 90 | MCP-specific Packrift surface. | Monitor/report proof first; remediation only for the remaining missing signal. |
| `packrift-agent-discovery-hub.vercel.app` | 45 | Discovery hub tied to the same packaging/commerce cluster. | `/score` and missing-surface checklist before paid remediation. |
| `packrift-flex-packaging-fit-lab.vercel.app` | 85 | Packaging fit tool. | Monitor plus targeted missing-surface checklist. |
| `packrift-benchmark-navigator.vercel.app` | 85 | Packaging benchmark tool. | Monitor plus targeted missing-surface checklist. |
| `twitterapi.io` | 65 | Social-data API. | `/score` first; API/OpenAPI/MCP readiness checklist before remediation. |
| `api.coingecko.com` | 15 | Market-data API. | `/score` first; do not claim coverage quality or data freshness. |
| `cruisecritic.com` | 20 | Travel/review content. | `/score` first; content-source contract checklist before remediation. |
| `evomap.ai` | 70 | AI self-evolution infrastructure. | Monitor/report proof plus missing-surface checklist. |
| `macrosfirst.com` | 15 | Nutrition/wellness app. | `/score` first; keep health-claim boundaries explicit. |
| `guavahealth.com` | 15 | Health-data app. | `/score` first; keep privacy/medical boundaries explicit. |

These are public readiness examples and possible owner-channel targets only. They are not customers, endorsements, paid leads, monitor registrations, badge-install consent, completed payments, revenue, private demand, or proof of market share.

## Useful angle

Recent additions give NHS a clean owner-onboarding lane: a domain appears in the index, the public profile shows a concrete score band, and the next step can be score-band aware.

For a multi-surface owner like Packrift, the message is not "buy placement." The useful path is:

1. Main domain and satellite app surfaces should each run `/score` because readiness can differ by host.
2. High or near-high score surfaces should register the free monitor and use the public report/badge as proof.
3. Partial-score surfaces should fix missing machine-readable files, API/OpenAPI/MCP/catalog signals, and robots/metadata drift before any paid remediation.
4. API-heavy or commerce-heavy readers should be routed to stable catalog/API-key surfaces only when the docs remain useful.

Safe short copy:

`NHS recently indexed several related product/API surfaces with scores from 45 to 90. That spread is the owner-side problem: agents see each host separately, so a main domain, MCP endpoint, discovery hub, and product tool can all have different readiness states. The safe path is score each host, monitor the ones already close, and remediate only the missing public contracts.`

## Gated use

Use this for exactly one gated owner-channel touch, product-handoff test, or recent-additions onboarding experiment. Refresh the live evidence before external use.

Required refresh:

- `/api/v1/stats`, `/api/v1/categories`, `/newest`, `/score`, `/monitor`, `/report`
- Representative `/site/{host}` pages for the target domains
- High-score, near-high-score, and partial-score `/fix/{host}` routes
- `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`
- `/api/v1`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`
- Aggregate `/api/v1/admin/mcp?days=7` and `/api/v1/admin/traffic?hours=168`

Before any public use, verify the active Foundry/Owl-owned account identity, check `marketing/social-post-ledger.json` if present, `outreach/distribution_log.csv`, and sync-state public-action locks, and avoid `modelcontextprotocol/*` plus `punkpeye/*` surfaces from `unitedideas`.

Do not imply any listed domain is a customer, partner, endorsement, paid lead, monitor registration, badge-install consent, private demand, completed payment, revenue, product quality proof, inventory accuracy, price freshness, health accuracy, medical endorsement, privacy compliance, travel review truth, market-data freshness, API reliability, crawler compliance, SEO lift, uptime proof, A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/MPP support for NHS, paid ranking placement, preferred inclusion, or score-methodology bypass. Do not publish raw user-agent strings, private query logs, raw checkout URLs, payment identifiers, buyer emails, private monitor rows, private score-fix rows, or private customer identifiers.
