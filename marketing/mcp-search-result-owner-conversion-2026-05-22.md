# MCP Search Result Owner Conversion - 2026-05-22

Automation: `business-marketer-not-human-search`

Scope: business-local marketing scout artifact only. No outreach, public post, directory submission, account creation, browser action, product-code edit, deploy, checkout completion, or crawl was performed.

## Evidence

- Public stats: `total_sites=4171`, `avg_score=35`, `top_category=developer`.
- Public categories: `developer=1228`, `ai-tools=897`, `other=777`, `data=402`, `finance=194`, `productivity=174`, `ecommerce=148`, `communication=118`, `security=114`, `health=58`, `jobs=27`, `education=21`, `news=12`, `spam=1`.
- Public discovery surfaces checked: `/`, `/score`, `/monitor`, `/report`, `/digest`, `/newest`, `/top`, `/mcp`, `/api/v1`, `/api/v1/catalog`, `/api/v1/quote`, `/llms.txt`, `/openapi.yaml`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, and `/.well-known/ai-plugin.json` returned HTTP `200`.
- `/api/v1/checkout` returned HTTP `400` without a POST body, expected for a checkout-start contract.
- `/.well-known/agent-card.json` returned HTTP `404`, so strict A2A Agent Card claims remain blocked.
- Aggregate traffic, last 168 hours: `/=3366`, `/badge/xquik.com.svg=2354`, `/.well-known/commerce.json=1502`, `/site/xquik.com=783`, `/.well-known/ai-plugin.json=692`, `/llms.txt=445`, `/openapi.yaml=416`, `/api/v1/catalog=324`, `/robots.txt=302`, `/api/v1/checkout=295`, `/api/v1/quote=295`, `/api/v1/search=176`, `/api/v1/submit=148`, `/about=99`, `/top=95`, `/api/v1=89`, `/.well-known/mcp.json=87`, `/score=78`, `/guide=74`, `/newest=65`, `/digest=65`, `/site/manifest.ly=64`.
- Aggregate referrers, last 168 hours: `google.com=542`, `/score=100`, `nothumansearch.fly.dev=64`, `/top=42`, `/site/chainray.online=38`, `aurelianflo.com=34`, plus material canonical-domain and `www` alias referrers.
- Aggregate MCP analytics, last 7 days: `tools/list=168682`, `initialize=26914`, `tools/call=169`.
- Top MCP tool calls: `search_agents=80`, `get_site_details=22`, `get_stats=19`, `check_url=17`, `find_mcp_servers=6`, `get_top_sites=6`, `recent_additions=6`, `verify_mcp=5`, `submit_site=4`, `list_categories=4`.
- Aggregate MCP client families include Claude Code, `MCP-Catalog-Bot`, `MCPScoringEngine`, `mcp-verify`, Python, and Node clients.
- Aggregate query themes include Singapore news/housing, Hermes/agent skills, document scanners/RAG, ESP32/IoT, electronics hardware, secrets management, Home Assistant, astrology/moon phases, and model/agent tooling.
- Score-band smoke: `xquik.com`, `chainray.online`, and `aurelianflo.com` public profiles returned `100/100` and high-score `/fix/{host}` pages route to the monitor/report handoff. `manifest.ly` returned `65/100` and remains the partial-score comparison example.

## Segment

The useful segment is MCP and REST search-result followthrough. Search is still one of the highest real use paths: REST `/api/v1/search` has `176` aggregate route hits, and MCP `search_agents` is the top actual tool call at `80` calls.

The next owner-conversion test should not be another broad search or homepage packet. It should route search result users by intent:

1. MCP clients and directory bots get MCP/API examples first.
2. Search users with owner intent get `/score`.
3. High-score result/profile visitors get free monitor registration, report sharing, and badge proof.
4. Partial-score result/profile visitors get a missing-surface checklist and `/score` before score-fix remediation.
5. API-heavy callers stay on API-key/catalog surfaces only when docs remain useful.

## Draft Channel Angle

Search results are the first proof surface. The next click should explain what to do with the score:

1. If the site already scores well, monitor it and use the public report or badge as proof.
2. If the score is partial, fix the missing machine-readable surfaces first.
3. If the caller is a tool or catalog, use the MCP server, OpenAPI spec, and API catalog instead of scraping pages.

## Gated Test

Prepare exactly one gated MCP-search, REST-search, search-result, site-owner, or MCP-client conversion test using this packet.

Before external use, refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/search`, `/mcp`, JSON-RPC `tools/list`, `/score`, `/monitor`, `/report`, representative high-score and partial-score `/site/{host}` pages, high-score and partial-score `/fix/{host}` routes, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, aggregate `/api/v1/admin/mcp?days=7`, and aggregate `/api/v1/admin/traffic?hours=168`.

Verify active Foundry/Owl-owned account identity before public use, check `marketing/social-post-ledger.json` if present plus `outreach/distribution_log.csv` and sync-state public-action locks, and avoid `modelcontextprotocol/*` plus `punkpeye/*` surfaces from `unitedideas`.

## Guardrails

Do not imply search, MCP tool calls, client families, searched domains, profile pages, badge routes, API/catalog routes, Google, alias, or external-referrer traffic proves customers, endorsements, partners, paid leads, monitor registrations, badge-install consent, private demand, completed payments, revenue, crawler compliance, SEO lift, uptime proof, vendor approval, A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/MPP support for NHS, paid ranking placement, preferred inclusion, or score-methodology bypass.
