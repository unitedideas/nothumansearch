# Static Asset and Badge Discovery Conversion - 2026-05-22

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
  - `/api/v1`, `/api/v1/catalog`, `/api/v1/quote`, `/openapi.yaml`, `/score`, `/monitor`, `/report`, `/digest`, `/newest`, `/top`, `/mcp`, and `/robots.txt`: HTTP 200.
  - `/api/v1/checkout`: HTTP 400 without a POST body, expected for a checkout-start contract.
- Aggregate traffic, last 168 hours:
  - root `/` = `3,415`.
  - `/badge/xquik.com.svg` = `2,319`; `/site/xquik.com` = `789`.
  - `/.well-known/commerce.json` = `1,517`.
  - `/.well-known/ai-plugin.json` = `701`; `/llms.txt` = `452`; `/openapi.yaml` = `423`.
  - `/api/v1/catalog` = `327`; `/robots.txt` = `304`; `/api/v1/quote` = `298`; `/api/v1/checkout` = `298`.
  - `/badge/aidevboard.com.svg` = `283`; `/badge/8bitconcepts.com.svg` = `276`.
  - `/api/v1/search` = `176`; `/favicon.ico` = `170`; `/api/v1/submit` = `148`; `/about` = `102`; `/top` = `96`; `/api/v1` = `92`; `/.well-known/mcp.json` = `90`.
- Aggregate referrers, last 168 hours:
  - `https://nothumansearch.ai/` = `2,021`.
  - `https://google.com` = `542`.
  - canonical and alternate domain refs are present across `nothumansearch.ai`, `nothumansearch.com`, `www`, `http`, and the Fly app host.
  - `https://nothumansearch.ai/score` = `104`.
- Aggregate MCP analytics, last 7 days:
  - `tools/list=167,667`, `initialize=26,315`, `tools/call=176`.
  - tool calls include `search_agents=83`, `get_site_details=23`, `get_stats=19`, `check_url=17`, `verify_mcp=7`, `get_top_sites=7`, `recent_additions=6`, `find_mcp_servers=6`, `list_categories=4`, `submit_site=4`.
  - visible aggregate query themes include Singapore news/housing, Hermes/agent skills, document indexing/RAG, scanner hardware, ESP32/IoT, electronics hardware, secrets management, and model/agent tooling.
- Score-band smoke:
  - `xquik.com`: public profile `100/100`; `/fix/xquik.com` routes to an already-meets-target monitor/report handoff.
  - `aidevboard.com`: Foundry-owned dogfood profile `100/100`; `/fix/aidevboard.com` routes to an already-meets-target handoff.
  - `manifest.ly`: public profile `65/100`; `/fix/manifest.ly` shows paid remediation intake.

## Segment

This run should treat static asset requests as owner-conversion routing evidence, not demand proof. The useful pattern is:

1. badge SVG traffic is materially higher than most application routes;
2. favicon traffic is also visible in the top page set;
3. high-score badge/profile visitors should be routed toward monitor/report/badge proof;
4. partial-score owners should be routed to `/score` before score-fix remediation;
5. static asset routes should never become the basis for customer, endorsement, or badge-install-consent claims.

## Draft Positioning

For owners with high scores:

> If the site already scores well, use the report, badge, and monitor as proof that the machine-readable surfaces stay healthy.

For owners with partial scores:

> The badge path should not skip the score. Start with the public profile, identify the missing machine-readable surfaces, and only then decide whether remediation is worth paying for.

For product handoff copy:

> Static assets like badges and favicons are high-volume entry points. Treat them as routes back to the report, monitor, and score checker rather than as isolated files.

## Candidate Execution Tests

1. Prepare a badge/static-asset conversion test: badge SVG or favicon-adjacent arrival -> matching public profile -> high-score monitor/report/badge proof or partial-score score checklist.
2. Prepare a no-send owner-channel draft for teams that embed or inspect readiness badges, with Foundry-owned dogfood examples labeled explicitly.
3. Queue a product handoff only if a later worker confirms badge/profile pages do not provide a clear path to monitor/report/badge proof for high-score domains and `/score` for partial-score domains.

## Guardrails

- Do not imply badge SVG hits, favicon requests, profile views, searched domains, submitted domains, manifest traffic, catalog traffic, or route traffic proves customers, endorsements, partners, paid leads, monitor registrations, badge-install consent, private demand, completed payments, revenue, crawler compliance, SEO lift, or uptime proof.
- Do not claim A2A support while `/.well-known/agent-card.json` is 404.
- Do not claim x402, ACP, MPP, or completed-payment support from catalog/quote/checkout traffic.
- Do not sell ranking placement, preferred inclusion, or score-methodology bypass.
- Label Foundry-owned dogfood examples before using them in any proof packet.
- Before external use, refresh public stats, discovery surfaces, aggregate MCP analytics, aggregate traffic, representative high-score and partial-score `/site/{host}` pages, high-score plus partial-score `/fix/{host}` behavior, and badge SVG route behavior.
