# Docs And API-Plan Conversion Scout

Run: 2026-05-19T07:08Z
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, checkout completion, or QLimit/global-queue write was
performed. This is a sanitized scout artifact for a later product or channel
operator.

No raw users, emails, API keys, checkout URLs, payment identifiers, private
monitor rows, private score-fix rows, private query logs, buyer data, or raw
customer data are included here.

## Fresh Evidence

Public surfaces checked:

- `/api/v1/stats`: `total_sites=4174`, `avg_score=35`,
  `top_category=developer`.
- `/api/v1/categories`: developer 1,237; ai-tools 900; other 765; data 399;
  finance 199; productivity 173; ecommerce 149; communication 119; security
  115; health 57; jobs 27; education 21; news 12; spam 1.
- `/llms.txt`: advertises 4,174+ sites, 11 MCP tools, `/submit`, `/check`,
  `/monitor/register`, paid API-key plans, `/score`, `/guide`, and `/report`.
- `/.well-known/mcp.json`: HTTP 200 with 11 tools.
- `/.well-known/agent.json`, `/.well-known/commerce.json`,
  `/api/v1/catalog`, `/api/v1`, `/openapi.yaml`, `/guide`, `/about`, `/mcp`,
  `/score`, and `/monitor`: HTTP 200.
- `/.well-known/agent-card.json`: HTTP 404, so strict A2A/Agent Card claims
  remain gated.
- `/site/chainray.online`: HTTP 200 and public score 100.
- `/fix/chainray.online`: HTTP 200 high-score handoff toward monitoring/report
  proof, not paid remediation.

Aggregate admin evidence, sanitized:

- Last 7 days MCP analytics: `tools/list=153185`, `initialize=20110`,
  `tools/call=292`.
- Top called MCP tools: `search_agents=185`, `get_site_details=38`,
  `check_url=15`, `get_stats=13`, `verify_mcp=10`, `recent_additions=10`,
  `get_top_sites=8`, `find_mcp_servers=7`, `submit_site=4`,
  `list_categories=2`.
- Last 168 hours traffic: `/=3524`, `/badge/xquik.com.svg=1699`,
  `/.well-known/commerce.json=1662`, `/.well-known/ai-plugin.json=790`,
  `/llms.txt=506`, `/site/xquik.com=482`, `/openapi.yaml=465`,
  `/api/v1/catalog=370`, `/api/v1/checkout=343`, `/api/v1/quote=343`,
  `/api/v1/submit=147`, `/top=112`, `/api/v1=97`, `/.well-known/mcp.json=96`,
  `/about=95`, `/guide=90`.
- Referrer aggregates included `google.com=542`, `/score=115`, `/top=45`,
  `/mcp=34`, `/site/chainray.online=34`, `/site/xquik.com=32`, and a
  site-search URL for `chainray.online=28`.
- Payment aggregate returned no completed payment signal in this scout output.

## Read

The docs/API discovery surfaces are now meaningful conversion surfaces, not
just passive documentation. `/about`, `/guide`, `/api/v1`, `/openapi.yaml`,
`/.well-known/mcp.json`, `/api/v1/catalog`, `/api/v1/quote`, and
`/api/v1/checkout` are all receiving aggregate traffic in the same weekly
window.

The safest next product test is a docs-to-API-plan conversion path:

1. Treat `/guide`, `/about`, `/api/v1`, `/openapi.yaml`, and `/mcp` as
   developer intent surfaces.
2. Add score-band-aware next steps without turning docs into a sales page:
   free `/score`, free `/monitor`, API-key plans for repeated programmatic
   access, and score-fix only when a concrete missing signal exists.
3. Keep high-score profile visitors such as `chainray.online` on monitor/report
   proof rather than remediation.

## Proposed Gated Test

Design one docs/API conversion test:

- `/guide`: add a compact next-step block after the implementation checklist:
  check a URL, monitor the domain, or use the API if the caller needs repeated
  checks.
- `/api/v1` and `/openapi.yaml`: keep the machine contract primary, but expose
  the catalog/API-key handoff where repeated usage or quota pressure is likely.
- `/mcp`: keep the install string and tool list primary, but point API-heavy
  users to the paid API-key plan and site owners to `/monitor`.
- Public site profiles: high-score domains route to monitor/report/badge proof;
  partial-score domains route to `/score` before score-fix remediation.

## Acceptance Guard

Before implementation or external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/guide`, `/about`, `/mcp`,
  `/api/v1`, `/openapi.yaml`, JSON-RPC `tools/list`,
  `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/agent-card.json`, `/.well-known/commerce.json`,
  `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/score`,
  `/monitor`, representative high-score and partial-score `/site/{host}` pages,
  high-score and partial-score `/fix/{host}` routes, aggregate
  `/api/v1/admin/mcp?days=7`, and aggregate
  `/api/v1/admin/traffic?hours=168`.
- Use only aggregate route/referrer counts and public URLs in committed
  artifacts.
- Do not run broad crawls, send outreach, create accounts, use browser or
  Computer Use, complete checkout, or write QLimit/global-queue state from a
  recurring worker.
- Do not imply docs visitors, searched domains, or profiled domains are
  customers, endorsements, paid leads, private demand, monitor registrations,
  completed payments, revenue, A2A support, x402/ACP support, paid ranking
  placement, preferred inclusion, or score-methodology bypass.
