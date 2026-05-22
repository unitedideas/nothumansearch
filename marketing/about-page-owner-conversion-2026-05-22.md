# About Page Owner Conversion - 2026-05-22

Automation: `business-marketer-not-human-search`

Scope: no public action, no outreach, no browser, no deploy. This packet is a gated owner-conversion artifact for a later execution worker.

## Fresh Evidence

- Public stats: `4,171` indexed sites, average score `35`, top category `developer`.
- Public categories: `developer=1,228`, `ai-tools=897`, `other=777`, `data=402`, `finance=194`, `productivity=174`, `ecommerce=148`, `communication=118`, `security=114`, `health=58`, `jobs=27`, `education=21`, `news=12`.
- Live discovery surfaces:
  - `/llms.txt`: HTTP 200.
  - `/.well-known/mcp.json`: HTTP 200.
  - `/.well-known/agent.json`: HTTP 200.
  - `/.well-known/agent-card.json`: HTTP 404.
  - `/.well-known/commerce.json`: HTTP 200.
  - `/.well-known/ai-plugin.json`: HTTP 200.
  - `/api/v1`, `/api/v1/catalog`, `/openapi.yaml`, `/score`, `/monitor`, `/report`, `/newest`, `/top`, `/digest`: HTTP 200.
- Aggregate traffic, last 168 hours:
  - root `/` = `3,411`.
  - `/badge/xquik.com.svg` = `2,296`; `/site/xquik.com` = `783`.
  - `/.well-known/commerce.json` = `1,517`.
  - `/.well-known/ai-plugin.json` = `701`; `/llms.txt` = `453`; `/openapi.yaml` = `423`.
  - `/api/v1/catalog` = `327`; `/api/v1/checkout` = `298`; `/api/v1/quote` = `298`.
  - `/api/v1/search` = `175`; `/api/v1/submit` = `148`; `/about` = `102`; `/top` = `96`.
- Aggregate MCP analytics, last 7 days:
  - `tools/list=166,814`, `initialize=25,867`, `tools/call=183`.
  - tool calls include `search_agents=88`, `get_site_details=23`, `get_stats=19`, `check_url=17`, `get_top_sites=8`, `verify_mcp=7`, `recent_additions=6`, `find_mcp_servers=6`, `list_categories=5`, `submit_site=4`.
  - visible aggregate client families include Claude Code, `MCP-Catalog-Bot`, and Python/Node MCP clients.
- Score-band smoke:
  - `xquik.com`: public profile `100/100`; `/fix/xquik.com` routes to the high-score monitor handoff.
  - `aidevboard.com`: Foundry-owned dogfood profile `100/100`; `/fix/aidevboard.com` routes to the high-score monitor handoff.
  - `8bitconcepts.com`: Foundry-owned dogfood profile `100/100`; `/fix/8bitconcepts.com` routes to the high-score monitor handoff.
  - `manifest.ly`: public profile `65/100`.

## Segment

This run should target people who arrive through explanatory pages before they touch a score or API route. `/about` now appears in the top route set next to `/api/v1/search`, `/api/v1/submit`, `/top`, manifest routes, and catalog/quote/checkout surfaces.

Useful routing pattern:

1. `/about` readers need a short path to the public score checker, not a generic description loop.
2. successful score or submit users should be nudged to free monitor registration.
3. high-score profile and badge visitors should go to monitor/report/badge proof.
4. partial-score owners should go through `/score` before any score-fix remediation.
5. API/catalog-heavy callers should stay on paid API-key/catalog surfaces only while the docs remain useful.

## Draft Positioning

For site owners:

> Not Human Search checks whether a site exposes machine-readable discovery surfaces agents can use: `llms.txt`, OpenAPI, MCP, API metadata, robots policy, and schema. Start with the public score. If the score is already strong, monitor it. If it is partial, fix the missing public files first.

For developer-tool readers:

> The about page should point builders directly to the MCP server, OpenAPI spec, API catalog, and score checker, with the A2A boundary named because `/.well-known/agent-card.json` is not live yet.

## Candidate Execution Tests

1. Prepare an about-page conversion copy test: one compact score-first CTA, one monitor-after-score CTA, and one API-catalog CTA for developer-tool readers.
2. Prepare a no-send owner-channel draft for site owners who read explanatory pages before using `/score`, keeping high-score and partial-score routing separate.
3. Queue a product handoff only if the later worker confirms `/about` lacks direct score, monitor, report, and API-catalog routes in the rendered page.

## Guardrails

- Do not imply `/about`, manifest, API, catalog, profile, badge, Google, alias, or route traffic proves customers, endorsements, partners, paid leads, monitor registrations, badge-install consent, private demand, completed payments, revenue, crawler compliance, SEO lift, or uptime proof.
- Do not claim A2A support while `/.well-known/agent-card.json` is 404.
- Do not claim x402, ACP, MPP, or completed-payment support from catalog/quote/checkout traffic.
- Do not sell ranking placement, preferred inclusion, or score-methodology bypass.
- Label Foundry-owned dogfood examples before using them in any proof packet.
- Before external use, refresh public stats, discovery surfaces, aggregate MCP analytics, aggregate traffic, representative high-score and partial-score `/site/{host}` pages, and high-score plus partial-score `/fix/{host}` behavior.
