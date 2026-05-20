# Regulatory AI Compliance Source-Readiness Brief

Run: 2026-05-20.

Purpose: prepare one gated owner-channel test for legal, regulatory, AI-governance, DORA, EU AI Act, security, and compliance tooling owners whose public sites need to be machine-readable for agents.

No public action was taken. No outreach was sent. This is a business-local scout artifact for later channel execution behind account identity, duplicate-ledger, and public-action-lock checks.

## Live Evidence

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4175`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: `security=115`, `avg_score=38`.
- `https://nothumansearch.ai/api/v1/top?category=security&limit=5`: public examples include DORA compliance, legal/regulatory/security workflows, EU AI Act compliance, AI tool status/pricing, and trust-layer positioning.
- `/.well-known/mcp.json`: live, 11 tools.
- `/.well-known/agent.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, `/api/v1`, `/llms.txt`, and `/openapi.yaml`: live and internally consistent enough for an agent-readable proof packet.
- `/.well-known/agent-card.json`: still returns 404, so A2A/Agent Card claims remain blocked.
- `/monitor`: live free monitor surface.
- `/fix/nothumansearch.ai` and `/fix/xquik.com`: high-score domains route to monitor/report instead of paid remediation, so score-fix copy can stay score-band-aware.
- `tools/monitor-check.log`: latest scheduled proof on 2026-05-18 completed one due monitor and left the checked score unchanged at 100.
- Aggregate admin traffic, 168h: `/badge/xquik.com.svg=1964`, `/.well-known/commerce.json=1644`, `/.well-known/ai-plugin.json=748`, `/site/xquik.com=631`, `/llms.txt=486`, `/openapi.yaml=447`, `/api/v1/catalog=354`, `/api/v1/quote=323`, `/api/v1/checkout=323`, `/api/v1/submit=147`, `/score=79`, `/api/v1/check=60`.
- Aggregate MCP, 7d: `tools/list=158456`, `initialize=20550`, `tools/call=277`; top called tools include `search_agents=175`, `get_site_details=38`, `check_url=17`, `verify_mcp=8`, `get_top_sites=8`, and `submit_site=4`.

## Segment

Compliance and security-tool owners have a strong fit for NHS because agent users need source contracts they can inspect without guessing:

- stable `llms.txt` and robots AI policy;
- OpenAPI specs with non-empty paths;
- machine-readable product, plan, and support boundaries;
- MCP or JSON endpoints that can be probed before use;
- free monitoring so deploys do not silently remove agent-readable surfaces;
- score-band-aware remediation when public readiness signals are incomplete.

This should be framed as source-readiness for agent workflows, not as a certification of legal accuracy, security quality, DORA/EU AI Act compliance, uptime, or audit readiness.

## Public Examples

Use these as public readiness examples only. They are not customers, endorsements, paid leads, or demand proof.

| Domain | Score | Scout use |
|---|---:|---|
| `feedoracle.io` | 100 | DORA/compliance operating-system example with complete public agent-readiness signals. |
| `ansvar.eu` | 100 | Legal, regulatory, and security workflow example with complete public agent-readiness signals. |
| `agent-module.dev` | 95 | EU AI Act compliance knowledge example missing Schema.org in the current public profile. |
| `tickerr.ai` | 85 | AI tool status/pricing example missing structured API in the current public profile. |
| `rnwy.com` | 80 | Trust-layer example missing OpenAPI in the current public profile. |

## Gated Channel Test

Draft one short owner-channel post or direct-owner packet:

> Agents should not have to scrape compliance claims from marketing pages.
>
> The current NHS security top list shows the pattern: the strongest compliance and regulatory tooling sites expose `llms.txt`, OpenAPI, plugin metadata, MCP/API endpoints, robots policy, and schema. Lower-score profiles usually fail on one public contract, not on product substance.
>
> Free check: `https://nothumansearch.ai/score`
> Free monitor: `https://nothumansearch.ai/monitor`

Before any public use, refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=security&limit=5`, representative `/site/{host}` pages, `/score`, `/monitor`, high-score and partial-score `/fix/{host}` routes, all machine-readable manifests, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, and aggregate admin MCP/traffic data.

## Boundaries

Do not claim compliance certification, legal accuracy, security certification, audit readiness, DORA compliance, EU AI Act compliance, uptime, pricing accuracy, private demand, customer endorsement, paid leads, monitor registrations, completed payments, revenue, paid placement, preferred inclusion, A2A support while `/.well-known/agent-card.json` is 404, or score-methodology bypass.
