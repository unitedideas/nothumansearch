# Automated Integration Crawler Conversion Packet - 2026-05-22

Automation: `business-marketer-not-human-search`

Scope: business-local marketing scout artifact only. No outreach, public post, directory submission, account creation, browser action, Computer Use, product-code edit, deploy, checkout completion, or crawl was performed.

## Evidence

- Public stats: `total_sites=4171`, `avg_score=35`, `top_category=developer`.
- Public categories: `developer=1228`, `ai-tools=897`, `other=777`, `data=402`, `finance=194`, `productivity=174`, `ecommerce=148`, `communication=118`, `security=114`, `health=58`, `jobs=27`, `education=21`, `news=12`, `spam=1`.
- Live discovery surfaces checked: `/score`, `/monitor`, `/report`, `/top`, `/digest`, `/newest`, `/mcp-servers`, `/openapi-apis`, `/llms-txt-sites`, `/api/v1`, `/api/v1/catalog`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/llms.txt`, `/openapi.yaml`, and `/feed.xml` returned HTTP 200.
- `/.well-known/agent-card.json` returned HTTP 404, so strict A2A Agent Card claims remain blocked.
- Aggregate MCP analytics, 7 days: `tools/list=169469`, `initialize=27618`, `tools/call=196`.
- Aggregate tool calls: `search_agents=99`, `get_site_details=30`, `check_url=20`, `get_stats=19`, `get_top_sites=6`, `find_mcp_servers=6`, `recent_additions=6`, `list_categories=5`, `submit_site=3`, `verify_mcp=2`.
- Aggregate client families include automated HTTP clients, desktop MCP clients, Claude Code variants, MCP catalog/scoring bots, and MCP verification clients. This is aggregate family evidence only, not endorsement.
- Aggregate traffic, 168 hours: `/=3326`, `/badge/xquik.com.svg=2426`, `/.well-known/commerce.json=1493`, `/site/xquik.com=843`, `/.well-known/ai-plugin.json=687`, `/llms.txt=448`, `/openapi.yaml=416`, `/api/v1/catalog=327`, `/robots.txt=301`, `/api/v1/quote=292`, `/api/v1/checkout=292`, `/api/v1/search=181`, `/favicon.ico=176`, `/api/v1/submit=142`, `/about=99`, `/top=96`.
- Aggregate referrers include the canonical NHS domain, Google, the `.com` alias, score/search flows, and public profile paths. No raw user identifiers are included here.
- Representative MCP query themes included local news and RSS monitoring, agent skills, hardware/device search, model API providers, scanner/electronics lookup, and source-code/tooling names.
- Public data top-list examples: `api.contrastcyber.com=100`, `api.boostedchat.com=95`, `api.headlessoracle.com=95`, `dchub.cloud=95`, `daedalmap.com=90`, `api.theartofservice.com=90`, `api.agentry.com=90`, `api.socialintel.dev=90`, `blocklens.co=90`, `api.meacheal.ai=85`.

## Segment

This segment is not another MCP catalog or desktop-client packet. The fresh angle is automated integration traffic: agents and crawlers are repeatedly reading discovery, commerce, OpenAPI, catalog, quote, checkout, search, submit, and badge/profile surfaces before a human owner ever enters the flow.

Useful followthrough:

1. Automated integrators get stable machine-readable entry points: `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/api/v1`, `/openapi.yaml`, `/llms.txt`, `/api/v1/catalog`.
2. Site owners arriving through submit/search/profile paths get routed to `/score` first.
3. High-score profile or badge visitors get routed to free monitor, report, and badge proof.
4. Partial-score owners get a missing-surface checklist and `/score` before paid remediation.
5. API-heavy callers get API-key plan surfaces only when docs stay useful and the handoff does not hide the free score/monitor path.

## Draft Channel Angle

NHS is already being read like infrastructure, not only as a website. The useful owner message is that agent-readiness is a public contract: manifests, OpenAPI, catalog, scoring, and monitorable profiles need to agree.

Safe short copy:

> Agents and catalogs do not wait for a sales page. They read manifests, OpenAPI, API catalogs, score pages, and public profiles. Not Human Search gives site owners a way to check those surfaces before agents rely on them, then monitor the score so a deploy does not erase the contract.

Owner routing:

- High-score owners: free monitor, public report, and badge proof.
- Partial-score owners: `/score` first, then remediation only for concrete missing public surfaces.
- API-heavy users: API-key plan handoff after the free docs, score, and monitor paths remain visible.

## Gated Test

Prepare exactly one gated automated-integration, machine-readable-docs, owner-channel, or API-plan conversion test from this packet.

Before external use, refresh `/api/v1/stats`, `/api/v1/categories`, `/mcp`, JSON-RPC `tools/list`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/api/v1/search`, `/api/v1/submit`, `/score`, `/monitor`, `/report`, `/llms.txt`, `/openapi.yaml`, `/feed.xml`, representative high-score and partial-score `/site/{host}` pages, high-score and partial-score `/fix/{host}` routes, aggregate `/api/v1/admin/mcp?days=7`, and aggregate `/api/v1/admin/traffic?hours=168`.

Verify the active Foundry/Owl-owned account identity for the selected channel, check `marketing/social-post-ledger.json` if present plus `outreach/distribution_log.csv` and sync-state public-action locks, and avoid `modelcontextprotocol/*` plus `punkpeye/*` surfaces from `unitedideas`.

## Guardrails

Do not imply automated HTTP clients, desktop MCP clients, Claude Code, MCP catalog/scoring bots, MCP verification clients, Google, Xquik, listed data APIs, or any profiled domain is a customer, partner, endorsement, paid lead, monitor registration, badge-install consent, private demand, completed payment, revenue, certification proof, vendor approval, crawler compliance, SEO lift, uptime proof, A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/MPP support for NHS, paid ranking placement, preferred inclusion, or score-methodology bypass.

Do not publish raw user-agent strings, private query logs, raw checkout URLs, payment identifiers, buyer emails, private monitor rows, private score-fix rows, or private customer identifiers.
