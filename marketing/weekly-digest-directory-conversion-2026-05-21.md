# Weekly Digest and Directory Conversion Scout - 2026-05-21

Automation: `business-marketer-not-human-search`
Scope: business-local marketing scout artifact only. No outreach, public post, directory submission, account creation, browser action, product-code edit, deploy, checkout completion, or crawl was performed.

## Evidence

- Public stats: `total_sites=4170`, `avg_score=35`, `top_category=developer`.
- Public categories: developer `1227`, ai-tools `897`, other `777`, data `402`, finance `194`, productivity `174`, ecommerce `148`, communication `118`, security `114`, health `58`, jobs `27`, education `21`, news `12`, spam `1`.
- Aggregate traffic, last 168 hours: `/=3399`, `/badge/xquik.com.svg=2199`, `/.well-known/commerce.json=1528`, `/site/xquik.com=723`, `/.well-known/ai-plugin.json=703`, `/llms.txt=458`, `/openapi.yaml=426`, `/api/v1/catalog=330`, `/robots.txt=306`, `/api/v1/quote=300`, `/api/v1/checkout=300`, `/api/v1/search=170`, `/api/v1/submit=151`, `/top=94`, `/score=79`, `/guide=77`, `/digest=66`, `/newest=66`, `/site/manifest.ly=63`.
- Aggregate referrers, last 168 hours: Google remains material, canonical-domain aliases remain material, `/score`, `/top`, `/site/chainray.online`, `/site/xquik.com`, `/mcp`, `/mcp-servers`, and `/submit` referrers remain visible.
- Public route smokes: `/digest`, `/newest`, `/top`, `/mcp-servers`, `/openapi-apis`, and `/llms-txt-sites` returned HTTP `200`.
- Route titles checked: `/digest` is `Weekly MCP Ecosystem Digest`, `/newest` is `Newest Agent-Ready Sites`, `/top` is `Top 100 Agent-Ready Sites`, `/mcp-servers` is `MCP Server Directory`, `/openapi-apis` is `OpenAPI Directory`, and `/llms-txt-sites` is `llms.txt Directory`.
- Discovery surfaces checked: `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`, `/openapi.yaml`, `/score`, `/monitor`, `/report`, and `/mcp` returned HTTP `200`. `/.well-known/agent-card.json` returned HTTP `404`.
- Aggregate MCP analytics, last 7 days: `tools/list=166368`, `initialize=24670`, `tools/call=270`; top tool calls include `search_agents=151`, `get_site_details=37`, `get_stats=20`, `check_url=19`, `get_top_sites=10`, `find_mcp_servers=9`, `recent_additions=8`, `verify_mcp=7`, `list_categories=5`, `submit_site=4`.
- Aggregate MCP client and directory-bot evidence includes desktop clients, catalog/scoring bots, and verifier-style callers, but this artifact does not publish raw user-agent or private query logs.

## Segment

NHS has a recurring-content surface that is already getting traffic: weekly digest, newest additions, top sites, and protocol-specific directory pages.

The useful segment is not another broad directory submission. It is a digest-to-owner conversion path:

1. Readers of `/digest`, `/newest`, `/top`, `/mcp-servers`, `/openapi-apis`, and `/llms-txt-sites` should have a clear path to check their own site.
2. High-score owners should be routed to free monitor registration, report sharing, and badge proof.
3. Partial-score owners should run `/score` before any remediation offer.
4. API-heavy readers should keep useful docs first, then see API-key plan handoffs when their usage pattern is clearly programmatic.
5. Agent-directory readers should see MCP/OpenAPI/llms.txt discovery surfaces without A2A claims while `/.well-known/agent-card.json` is still 404.

## Draft Channel Angle

Weekly agent-readiness digests are useful because the machine-readable web is moving in small increments, not because raw index size proves quality.

Not Human Search already exposes the public surfaces agents inspect: weekly digest, newest additions, top ranked sites, MCP servers, OpenAPI APIs, llms.txt sites, OpenAPI, MCP JSON, API root, catalog, and score reports.

The owner-side path should stay simple:

1. Check the current public score.
2. Register a free monitor when the score is strong.
3. Use the report or badge as proof.
4. Fix missing machine-readable surfaces only when the current score shows a real gap.

## Gated Test

Prepare exactly one gated channel touch, owner-channel brief, or product-handoff test around weekly digest and protocol-directory readers.

Before external use, refresh `/api/v1/stats`, `/api/v1/categories`, `/digest`, `/newest`, `/top`, `/mcp-servers`, `/openapi-apis`, `/llms-txt-sites`, `/score`, `/monitor`, `/report`, representative `/site/{host}` profiles, high-score and partial-score `/fix/{host}` routes, `/llms.txt`, `/openapi.yaml`, `/api/v1`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, aggregate MCP analytics, and aggregate traffic.

Verify active Foundry/Owl-owned account identity before public use, check `marketing/social-post-ledger.json` if present plus `outreach/distribution_log.csv` and sync-state public-action locks, and avoid `modelcontextprotocol/*` plus `punkpeye/*` surfaces from `unitedideas`.

## Guardrails

Do not imply digest, newest, top-list, protocol-directory, manifest, API, catalog, profile, badge, Google, alias, or route traffic proves customers, endorsements, partners, paid leads, monitor registrations, badge-install consent, private demand, completed payments, revenue, crawler compliance, legal permission, SEO lift, uptime proof, A2A support, x402/ACP support, paid ranking placement, preferred inclusion, or score-methodology bypass.
