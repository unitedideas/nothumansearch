# Score Page Monitor Followthrough - 2026-05-22

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
  - `/api/v1`, `/api/v1/catalog`, `/api/v1/quote`, `/openapi.yaml`, `/score`, `/monitor`, `/report`, `/digest`, `/newest`, `/top`, and `/mcp`: HTTP 200.
  - `/api/v1/checkout`: HTTP 400 without a POST body, expected for a checkout-start contract.
- Aggregate traffic, last 168 hours:
  - root `/` = `3,418`.
  - `/badge/xquik.com.svg` = `2,305`; `/site/xquik.com` = `783`.
  - `/.well-known/commerce.json` = `1,517`.
  - `/.well-known/ai-plugin.json` = `701`; `/llms.txt` = `453`; `/openapi.yaml` = `423`.
  - `/api/v1/catalog` = `327`; `/api/v1/checkout` = `298`; `/api/v1/quote` = `298`.
  - `/api/v1/search` = `175`; `/api/v1/submit` = `148`; `/about` = `102`; `/top` = `96`; `/api/v1` = `92`; `/.well-known/mcp.json` = `90`.
  - score-page referrer = `https://nothumansearch.ai/score` with `104` aggregate referrals.
- Aggregate MCP analytics, last 7 days:
  - `tools/list=167,149`, `initialize=26,068`, `tools/call=178`.
  - tool calls include `search_agents=85`, `get_site_details=23`, `get_stats=19`, `check_url=17`, `verify_mcp=7`, `get_top_sites=7`, `recent_additions=6`, `find_mcp_servers=6`, `list_categories=4`, `submit_site=4`.
  - visible aggregate query themes include Singapore news/housing, Hermes/agent skills, document indexing/RAG, scanner hardware, ESP32/IoT, electronics hardware, codepraxis, MacrosFirst, and Nous Research Hermes Agent.
- Score-band smoke:
  - `xquik.com`: public profile `100/100`; `/fix/xquik.com` routes to an already-meets-target monitor handoff.
  - `chainray.online`: public profile `100/100`; `/fix/chainray.online` routes to an already-meets-target monitor handoff.
  - `nothumansearch.ai`: Foundry-owned dogfood profile `100/100`; `/fix/nothumansearch.ai` routes to an already-meets-target monitor handoff.
  - `manifest.ly`: public profile `65/100`; `/fix/manifest.ly` shows paid remediation intake.
- Public page text smoke:
  - `/score` is score-centered.
  - `/monitor` exposes free email monitoring language.

## Segment

This run should target people who have already reached the public score checker, not another broad directory or explanatory-page audience. The useful followthrough path is:

1. score checker users get a score result;
2. high-score owners should be routed to free monitor/report/badge proof;
3. partial-score owners should be routed to the public score result first, then score-fix only when missing readiness surfaces are visible;
4. API-heavy callers should stay on API-key/catalog surfaces when they are using machine-readable routes.

## Draft Positioning

For site owners after a score check:

> If the score is already strong, monitor it and use the report or badge as proof. If the score is partial, fix the missing machine-readable files first: `llms.txt`, OpenAPI, MCP, API metadata, robots policy, or schema.

For product handoff copy:

> The score checker should not dead-end. High scores should lead to monitor/report/badge proof. Partial scores should lead to a concrete missing-surface checklist before paid remediation.

## Candidate Execution Tests

1. Prepare a score-result followthrough test: high-score result -> free monitor/report/badge proof; partial-score result -> missing-surface checklist -> score-fix only after the public evidence is clear.
2. Prepare a no-send owner-channel draft for teams that already use `/score`, keeping monitor proof and paid remediation separate.
3. Queue a product handoff only if a later worker confirms score-result pages or score submission responses do not offer monitor/report/badge proof for high-score domains.

## Guardrails

- Do not imply `/score` traffic, score referrers, searched domains, submitted domains, profile pages, badge routes, API/catalog routes, or manifest traffic proves customers, endorsements, partners, paid leads, monitor registrations, badge-install consent, private demand, completed payments, revenue, crawler compliance, SEO lift, or uptime proof.
- Do not claim A2A support while `/.well-known/agent-card.json` is 404.
- Do not claim x402, ACP, MPP, or completed-payment support from catalog/quote/checkout traffic.
- Do not sell ranking placement, preferred inclusion, or score-methodology bypass.
- Label Foundry-owned dogfood examples before using them in any proof packet.
- Before external use, refresh public stats, discovery surfaces, aggregate MCP analytics, aggregate traffic, representative high-score and partial-score `/site/{host}` pages, and high-score plus partial-score `/fix/{host}` behavior.
