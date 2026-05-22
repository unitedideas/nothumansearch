# Guide Page Owner Conversion - 2026-05-22

Automation: `business-marketer-not-human-search`

Scope: no public action, no outreach, no browser, no deploy. This packet is a gated owner-conversion artifact for a later execution worker.

## Fresh Evidence

- Public stats: `4,171` indexed sites, average score `35`, top category `developer`.
- Public categories: `developer=1,228`, `ai-tools=897`, `other=777`, `data=402`, `finance=194`, `productivity=174`, `ecommerce=148`, `communication=118`, `security=114`, `health=58`, `jobs=27`, `education=21`, `news=12`, `spam=1`.
- Live discovery surfaces:
  - `/mcp`: HTTP 200.
  - `/.well-known/mcp.json`: HTTP 200.
  - `/.well-known/agent.json`: HTTP 200.
  - `/.well-known/agent-card.json`: HTTP 404.
  - `/api/v1/catalog`: HTTP 200.
  - `/score`: HTTP 200.
  - `/monitor`: HTTP 200.
- Aggregate traffic, last 168 hours:
  - root `/` = `3,376`.
  - `/badge/xquik.com.svg` = `2,339`; `/site/xquik.com` = `795`.
  - `/.well-known/commerce.json` = `1,517`.
  - `/.well-known/ai-plugin.json` = `701`; `/llms.txt` = `452`; `/openapi.yaml` = `423`.
  - `/api/v1/catalog` = `327`; `/robots.txt` = `303`; `/api/v1/quote` = `298`; `/api/v1/checkout` = `298`.
  - `/api/v1/search` = `176`; `/favicon.ico` = `169`; `/api/v1/submit` = `148`.
  - `/about` = `99`; `/top` = `96`; `/api/v1` = `92`; `/.well-known/mcp.json` = `90`; `/score` = `79`; `/guide` = `75`; `/digest` = `65`; `/newest` = `64`; `/site/manifest.ly` = `64`.
- Aggregate referrers, last 168 hours:
  - Google remains material at `542` plus `www.google.com=46`.
  - canonical and alternate domain refs remain material across `nothumansearch.ai`, `nothumansearch.com`, `www`, `http`, and the Fly app host.
  - `/score` referrer = `100`; `/top` referrer = `42`; `/site/chainray.online` referrer = `38`; `aurelianflo.com` referrer = `34`.
- Aggregate MCP analytics, last 7 days:
  - `tools/list=168,101`, `initialize=26,551`, `tools/call=174`.
  - tool calls include `search_agents=83`, `get_site_details=22`, `get_stats=19`, `check_url=17`, `verify_mcp=7`, `recent_additions=6`, `get_top_sites=6`, `find_mcp_servers=6`, `list_categories=4`, `submit_site=4`.
  - visible aggregate query themes include Singapore news/housing, Hermes/agent skills, document indexing/RAG, scanner hardware, ESP32/IoT, electronics hardware, secrets management, Home Assistant, astrology/moon phases, and model/agent tooling.
- Score-band smoke:
  - `aurelianflo.com`: public profile `100/100`; `/fix/aurelianflo.com` routes to monitor/report proof.
  - `chainray.online`: public profile `100/100`; `/fix/chainray.online` routes to monitor/report proof.
  - `manifest.ly`: public profile `65/100`.

## Segment

This run should treat `/guide` as an owner-education conversion surface. It sits in the same traffic band as `/score`, `/about`, `/top`, `/api/v1`, and manifest pages, so the useful test is not more generic guide copy. The useful test is whether the guide routes readers into the right next action:

1. site owners reading the scoring method should start with `/score`;
2. high-score owners should move to free monitor registration, report sharing, and badge proof;
3. partial-score owners should get a missing-surface checklist before score-fix remediation;
4. API-heavy readers should get the API catalog and plan surfaces only after the guide remains useful;
5. A2A-style directory readers should see the `/.well-known/agent-card.json` blocker named explicitly until it is fixed.

## Draft Positioning

For site owners:

> The guide explains the public files and protocols behind the score. Start with `/score`; if the score is strong, monitor it and use the report or badge as proof. If it is partial, fix the missing public surfaces before considering remediation.

For developer-tool readers:

> The guide should connect the scoring method to live machine-readable surfaces: MCP, OpenAPI, `llms.txt`, API catalog, commerce metadata, and the current Agent Card boundary.

## Candidate Execution Tests

1. Prepare a guide-page conversion test: method explanation -> `/score` -> high-score monitor/report/badge proof or partial-score missing-surface checklist.
2. Prepare a no-send owner-channel draft for teams that already read the guide or protocol pages before using `/score`.
3. Queue a product handoff only if a later worker confirms `/guide` lacks direct score, monitor, report, API-catalog, and Agent Card boundary routes.

## Guardrails

- Do not imply `/guide`, `/score`, `/top`, profile, badge, API, catalog, manifest, Google, alias, or external-referrer traffic proves customers, endorsements, partners, paid leads, monitor registrations, badge-install consent, private demand, completed payments, revenue, crawler compliance, SEO lift, or uptime proof.
- Do not claim A2A support while `/.well-known/agent-card.json` is 404.
- Do not claim x402, ACP, MPP, or completed-payment support from catalog/quote/checkout traffic.
- Do not sell ranking placement, preferred inclusion, or score-methodology bypass.
- Label Foundry-owned dogfood examples before using them in any proof packet.
- Before external use, refresh public stats, discovery surfaces, aggregate MCP analytics, aggregate traffic, representative high-score and partial-score `/site/{host}` pages, high-score plus partial-score `/fix/{host}` behavior, and `/guide` route behavior.
