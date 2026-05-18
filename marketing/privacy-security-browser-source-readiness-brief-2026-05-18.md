# Privacy and Security Browser Source-Readiness Brief

Date: 2026-05-18
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, checkout completion, or QLimit/global-queue write was
performed. This is a sanitized scout artifact for a later gated operator.

No raw users, emails, API keys, checkout URLs, payment identifiers, private
monitor rows, private score-fix rows, private query logs, buyer data, or raw
customer data are included here.

## Evidence Snapshot

Public surfaces checked:

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4174`,
  `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: `security=115`,
  `avg_score=38`; `developer=1237`, `avg_score=34`;
  `communication=119`, `avg_score=38`; `productivity=173`,
  `avg_score=39`.
- `https://nothumansearch.ai/api/v1/top?category=security&limit=8`,
  `/api/v1/top?category=developer&limit=8`,
  `/api/v1/top?category=communication&limit=8`, and
  `/api/v1/top?category=productivity&limit=8`: public top lists returned
  HTTP 200 with `results` arrays.
- `https://nothumansearch.ai/llms.txt`, `/.well-known/mcp.json`,
  `/.well-known/agent.json`, `/.well-known/commerce.json`,
  `/api/v1/catalog`, `/score`, `/monitor`, and `/openapi.yaml`: HTTP 200.
- `https://nothumansearch.ai/.well-known/agent-card.json`: HTTP 404, so
  strict Agent Card and A2A-style directory claims remain gated.
- Score-fix route check: high-score examples such as `nothumansearch.ai` and
  `feedoracle.io` route toward monitoring/reporting; partial-score examples
  such as `secondsim.co.uk`, `plain.com`, and `can-tap-verified.com` still
  expose the paid remediation path.

Aggregate admin evidence, sanitized:

- MCP analytics, 7 days: `tools/list=147268`, `initialize=19334`,
  `tools/call=323`.
- Top MCP tool calls: `search_agents=208`, `get_site_details=44`,
  `check_url=16`, `get_stats=12`, `verify_mcp=11`, `recent_additions=10`,
  `get_top_sites=8`, `find_mcp_servers=7`, `submit_site=4`,
  `list_categories=3`.
- Aggregate query themes included browser privacy, VPN/control panels,
  security infrastructure, developer tools, and probe-before-use source
  discovery. No raw user identifiers were written.
- Traffic, 168 hours: `/.well-known/commerce.json=1630`,
  `/api/v1/catalog=365`, `/api/v1/quote=338`, `/api/v1/checkout=338`,
  `/top=118`, `/newest=87`, `/score=73`, and `/.well-known/mcp.json=89`.

## Segment

Privacy browser, VPN/control-plane, secure developer-tool, and security
infrastructure owners whose products need agents to understand public
capabilities, auth boundaries, support channels, API contracts, and monitoring
state without scraping a human-only marketing page.

This is adjacent to the broader security and network-infrastructure briefs,
but narrower: the fresh query themes point at privacy and browser/network
control surfaces where agents need explicit machine-readable boundaries before
calling, recommending, or integrating a service.

## Public Examples

These are public NHS index examples only. Treat them as readiness-pattern
examples or owner-channel targets, not customers, endorsements, paid leads,
private demand, completed purchases, or proof of market coverage.

| Domain | Score | Current agent-readiness shape | Safe owner route |
|---|---:|---|---|
| `feedoracle.io` | 100 | Security-category service with complete public readiness signals. | Free monitor/report/badge proof; do not claim security certification. |
| `ansvar.eu` | 100 | Security-category service with complete public readiness signals. | Free monitor/report/badge proof; useful as an owner-side readiness pattern. |
| `agent-module.dev` | 95 | Security/developer tool with strong agent-readable signals. | Monitor/report proof; watch for regression below 95. |
| `tickerr.ai` | 85 | Security-category service with partial readiness. | `/score` first; remediation can focus on missing public contracts. |
| `secondsim.co.uk` | 70 | Communication/identity-adjacent service with partial readiness. | `/score` first, then score-fix only for missing agent-facing surfaces. |
| `plain.com` | 50 | Communication/support platform with low partial readiness in the current index. | `/score` first; owner copy should stay remediation-oriented, not accusatory. |

## Angle

Privacy and security tools need agent-readable boundaries before agents can
recommend or integrate them. A browser privacy page, VPN panel, support tool, or
security API should expose what agents can inspect: public API or OpenAPI
contracts, MCP availability when real, robots/LLM policy, support/contact
metadata, plan/auth boundaries, and a monitorable readiness score.

Safe short copy:

`Privacy and security products are getting pulled into agent workflows, but many
still present a human-only web surface. Not Human Search is useful as a source
readiness check: run /score, publish the missing public contracts, and register
a free monitor so deploys do not silently remove the signals agents rely on.`

Owner-channel routes:

- High-score security owners: free monitor registration, public report page, and
  badge proof.
- Partial-score privacy/browser/support owners: `/score` first, then score-fix
  only when missing public surfaces justify remediation.
- API-heavy security or VPN-control-plane products: API/catalog readiness and
  paid NHS API plans only when the buyer asks for higher-volume NHS access.

## Publication Guard

Before external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`,
  `/api/v1/top?category=security&limit=8`,
  `/api/v1/top?category=developer&limit=8`,
  `/api/v1/top?category=communication&limit=8`, representative
  `/site/{host}` profiles, `/score`, `/monitor`, `/.well-known/mcp.json`,
  `/.well-known/agent.json`, `/.well-known/agent-card.json`,
  `/.well-known/commerce.json`, `/api/v1/catalog`, `/api/v1/quote`,
  `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, and aggregate
  `/api/v1/admin/mcp?days=7`.
- Verify the active Foundry/Owl-owned account identity for the selected channel.
- Check `marketing/social-post-ledger.json` if present, sync-state
  public-action locks, and `outreach/distribution_log.csv`.
- Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.
- Do not imply listed domains are customers, endorsements, paid leads, private
  demand, completed payments, revenue, browser privacy certification, VPN
  safety, security certification, compliance certification, uptime proof,
  integration reliability, data freshness, pricing accuracy, seller
  certification, x402/ACP/MPP support for NHS, paid ranking placement,
  preferred inclusion, A2A support, or score-methodology bypass.
