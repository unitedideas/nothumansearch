# Submit-Site Owner Onboarding Scout

Run: 2026-05-19T04:20Z
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
  115; health 57; jobs 27; education 21; news 12.
- `/llms.txt`: advertises 4,174+ sites, 11 MCP tools, `/submit`, `/check`,
  `/monitor/register`, paid API-key plans, `/score`, `/guide`, and `/report`.
- `/.well-known/mcp.json`: HTTP 200 with 11 tools.
- `/.well-known/agent.json`, `/.well-known/commerce.json`,
  `/api/v1/catalog`, `/api/v1`, `/score`, `/monitor`, `/top`, and `/newest`:
  HTTP 200.
- `/.well-known/agent-card.json`: HTTP 404, so strict A2A/Agent Card claims
  remain gated.

Aggregate admin evidence, sanitized:

- Last 7 days MCP analytics: `tools/list=152807`, `initialize=20082`,
  `tools/call=294`.
- Top called MCP tools: `search_agents=187`, `get_site_details=37`,
  `check_url=15`, `get_stats=13`, `verify_mcp=10`, `recent_additions=10`,
  `get_top_sites=8`, `find_mcp_servers=7`, `submit_site=4`,
  `list_categories=3`.
- Last 168 hours traffic: `/api/v1/submit=147`, `/api/v1/check=59`,
  `/api/v1=96`, `/score=74`, `/top=113`, `/newest=88`,
  `/site/xquik.com=481`, `/badge/xquik.com.svg=1693`,
  `/.well-known/commerce.json=1663`, `/api/v1/catalog=371`,
  `/api/v1/quote=343`, `/api/v1/checkout=343`.
- Google referrer aggregate: `google.com=542`; `/score` referrer aggregate:
  115.
- Payment aggregate returned no completed payment signal in this scout output.

## Read

The strongest fresh owner-onboarding signal is not another vertical topic. It is
the public REST submit path. `/api/v1/submit` received 147 requests in the same
168-hour window where `/score` received 74 and `/api/v1/check` received 59.
That looks like agents or owners are trying to add sites, but the next-step
handoff is not as explicit as the paid commerce/catalog surfaces.

This should be treated as an onboarding/conversion design test:

1. After a successful submit, route the owner or agent to the free score check
   and monitor registration path.
2. For the submitted domain's later public profile, route high-score owners to
   monitor/report/badge proof and lower-score owners to `/score` before any
   score-fix remediation.
3. Keep API-heavy callers on the paid API plan path only when they hit usage or
   need deterministic machine access.

## Proposed Gated Test

Design a submit-to-monitor owner onboarding test across REST and MCP submit
flows:

- REST submit success response: include next-step links to `/score`,
  `/monitor`, and the public `/site/{host}` profile when available.
- MCP `submit_site` result: include a concise next-step note for monitor
  registration and score-checking without exposing private rows.
- Public `/newest` and site profile pages: keep owner CTAs score-band-aware.

## Acceptance Guard

Before implementation or external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1`, `/api/v1/submit`
  contract docs, JSON-RPC `tools/list`, MCP `submit_site` response shape,
  `/score`, `/monitor`, `/newest`, representative `/site/{host}` pages,
  `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/agent-card.json`, `/.well-known/commerce.json`,
  `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`,
  `/openapi.yaml`, aggregate `/api/v1/admin/mcp?days=7`, and aggregate
  `/api/v1/admin/traffic?hours=168`.
- Use only aggregate route counts and public URLs in committed artifacts.
- Do not run broad crawls, send outreach, create accounts, use browser or
  Computer Use, complete checkout, or write QLimit/global-queue state from a
  recurring worker.
- Do not imply submitters are customers, endorsements, paid leads, private
  demand, monitor registrations, completed payments, revenue, A2A support,
  x402/ACP support, paid ranking placement, preferred inclusion, or
  score-methodology bypass.
