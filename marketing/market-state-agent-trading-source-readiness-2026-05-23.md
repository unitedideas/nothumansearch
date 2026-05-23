# Market-state and agent-trading source readiness - 2026-05-23

Run context: `business-marketer-not-human-search` recurring scout. No outreach, posting, browser, Computer Use, deploy, product-code edit, full recrawl, account creation, checkout completion, or QLimit/global-queue write was performed.

## Fresh aggregate signal

- Public stats: `/api/v1/stats` returned 200 with `total_sites=4171`, `avg_score=35`, and `top_category=developer`.
- Public category counts: `finance=194 avg_score=40`, `data=402 avg_score=32`, `developer=1228 avg_score=34`, `ecommerce=148 avg_score=41`.
- Aggregate MCP analytics over 7 days: `tools/list=169955`, `initialize=27823`, `tools/call=225`.
- Top MCP tool calls: `search_agents=110`, `get_site_details=35`, `check_url=32`, `get_stats=19`, `list_categories=6`, `get_top_sites=6`, `recent_additions=6`, `find_mcp_servers=5`, `submit_site=4`, `verify_mcp=2`.
- Aggregate traffic over 168 hours: `/=3361`, `/badge/xquik.com.svg=2513`, `/.well-known/commerce.json=1444`, `/site/xquik.com=925`, `/.well-known/ai-plugin.json=667`, `/llms.txt=441`, `/openapi.yaml=405`, `/api/v1/catalog=318`, `/robots.txt=304`, `/api/v1/checkout=282`, `/api/v1/quote=282`, `/api/v1/search=193`, `/api/v1/submit=143`, `/about=99`, `/top=95`.
- Live discovery checks returned 200 for `/score`, `/monitor`, `/newest`, `/top`, `/mcp-servers`, `/openapi-apis`, `/llms-txt-sites`, `/api/v1/catalog`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/llms.txt`, `/openapi.yaml`, and `/feed.xml`.
- `/.well-known/agent-card.json` returned 404, so strict Agent Card and A2A-style claims remain gated.

## Public examples

Current `/api/v1/top?category=finance&limit=12` shows a narrower owner-channel segment than the older broad finance brief: market-state APIs, on-chain data, trading signals, payment processors, and x402/pay-per-query data products.

| Domain | Score | Public readiness pattern | Safe owner route |
|---|---:|---|---|
| `terminalfeed.io` | 100 | Live crypto, stocks, news, prediction-market, earthquake, and launch feeds with all readiness signals present. | Monitor/report/badge proof. |
| `chartlibrary.io` | 100 | Stock chart pattern search built for AI agents with all readiness signals present. | Monitor/report/badge proof. |
| `prereason.com` | 100 | Market-context API for financial agents with all readiness signals present. | Monitor/report/badge proof. |
| `devdrops.run` | 95 | x402 pay-per-query data APIs with llms.txt, plugin, OpenAPI, API, MCP, and AI-friendly robots. | Monitor/report proof plus missing schema checklist; do not imply NHS supports x402 checkout. |
| `razorpay.com` | 90 | Payment gateway with llms.txt, plugin, OpenAPI, API, robots, and schema; no MCP in the public top response. | `/score` first, then missing-surface checklist. |
| `ticksurfers.com` | 90 | Trading indicators and algorithmic systems with strong API/readiness signals; no MCP in the public top response. | `/score` first, then missing-surface checklist. |
| `lendtrain.com` | 85 | Mortgage refinance/rate surface with llms.txt, plugin, OpenAPI, MCP, robots, and schema; structured API missing in the public top response. | Monitor plus missing-surface checklist. |
| `debridge.com` | 80 | Cross-chain swap/transfer surface with llms.txt, plugin, API, MCP, robots, and schema; OpenAPI missing in the public top response. | `/score` first, then OpenAPI/source-contract checklist. |
| `emc2ai.io` | 80 | Autonomous crypto intelligence with OpenAPI, API, MCP, robots, schema, and llms.txt; plugin missing in the public top response. | `/score` first; avoid trading-quality claims. |
| `bullrundata.com` | 70 | Market intelligence API with llms.txt, OpenAPI, API, robots, and schema; MCP and plugin missing in the public top response. | `/score` first, then remediation only for concrete missing public contracts. |

These are public readiness examples or owner-channel targets only. They are not customers, endorsements, paid leads, monitor registrations, badge-install consent, completed payments, revenue, private demand, or market-share proof.

## Useful angle

Market-data, payment, trading, and agent-commerce owners have a source-readiness problem that is narrower than generic finance marketing. Agents need to know whether a market-state or transaction surface is inspectable before relying on it:

1. Stable `llms.txt`, OpenAPI/API, MCP where real, catalog/quote/checkout metadata where relevant, and explicit unsupported-rail boundaries.
2. Clear source-contract metadata for market state, update windows, auth, pricing, payment rails, and refund/contact surfaces.
3. Monitorable drift for broken manifests, removed API paths, missing MCP endpoints, stale catalog pages, and score drops.
4. Score-band-aware routing: high-score owners to free monitor/report/badge proof; partial-score owners to `/score` before remediation.

Safe short copy:

`Agents do not need another finance landing page. They need to know whether the market-data, trading, payment, or pay-per-query API surface is inspectable before a workflow touches it: llms.txt, OpenAPI/API, real MCP if present, catalog/quote metadata, robots policy, and a public profile that can be monitored for drift. NHS scores that public surface and routes high-score owners to monitor/report proof, with remediation reserved for concrete missing contracts.`

## Gated use

Use this for exactly one gated owner-channel touch, product-handoff test, or API-plan conversion test for market-data, trading-signal, payment, x402/pay-per-query, and agent-commerce API owners. Refresh the live evidence before external use.

Required refresh:

- `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=finance&limit=12`, `/api/v1/top?category=data&limit=12`
- `/score`, `/monitor`, `/report`, representative `/site/{host}` pages
- High-score, near-high-score, and partial-score `/fix/{host}` routes
- `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`
- `/api/v1`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/api/v1/api-keys/subscribe`, `/llms.txt`, `/openapi.yaml`
- Aggregate `/api/v1/admin/mcp?days=7` and `/api/v1/admin/traffic?hours=168`

Before public use, verify the active Foundry/Owl-owned account identity, check `marketing/social-post-ledger.json` if present, `outreach/distribution_log.csv`, and sync-state public-action locks, and avoid `modelcontextprotocol/*` plus `punkpeye/*` surfaces from `unitedideas`.

Do not imply market-data, trading, payment, mortgage, crypto, x402, agent-commerce, or profiled domains are customers, partners, endorsements, paid leads, monitor registrations, badge-install consent, private demand, completed payments, revenue, trading performance, market-data accuracy, price freshness, payment reliability, mortgage-rate accuracy, security/compliance certification, x402/ACP/MPP support for NHS, crawler compliance, SEO lift, uptime proof, A2A support while `/.well-known/agent-card.json` is 404, paid ranking placement, preferred inclusion, or score-methodology bypass. Do not publish raw user-agent strings, private query logs, raw checkout URLs, payment identifiers, buyer emails, private monitor rows, private score-fix rows, or private customer identifiers.
