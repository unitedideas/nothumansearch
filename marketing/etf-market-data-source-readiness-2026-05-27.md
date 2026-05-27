# ETF and Market-Data Source Readiness

Run: 2026-05-27
Automation: `business-marketer-not-human-search`
Status: no-submit scout artifact; public use still requires account identity verification, duplicate checks, and a sync-state public-action lock.

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, checkout completion, or QLimit/global-queue write was
performed. This is a sanitized business-local artifact for a later gated
operator.

No raw users, emails, API keys, checkout URLs, payment identifiers, private
monitor rows, private score-fix rows, private query logs, raw user-agent
strings, buyer data, or customer identifiers are included here.

## Evidence

- Public stats: `total_sites=4174`, `avg_score=35`, `top_category=developer`.
- Public category counts: `developer=1231`, `data=402`, `finance=192`, and
  `security=113`.
- Public finance top-list examples included `terminalfeed.io=100`,
  `chartlibrary.io=100`, `prereason.com=100`, `devdrops.run=95`,
  `ticksurfers.com=90`, `razorpay.com=85`, `lendtrain.com=85`, and
  `debridge.com=80`. Treat these as public readiness examples only.
- Public data top-list examples included `api.headlessoracle.com=100`,
  `api.contrastcyber.com=100`, `dchub.cloud=95`, `daedalmap.com=90`,
  `api.theartofservice.com=90`, `api.agentry.com=90`,
  `api.socialintel.dev=90`, and `blocklens.co=90`. Treat these as public
  readiness examples only.
- Live public surfaces returned 200 for `/score`, `/monitor`, `/report`,
  `/api/v1/catalog`, `/llms.txt`, `/openapi.yaml`, `/feed.xml`, and `/mcp`
  JSON-RPC `tools/list`.
- Live `/mcp` JSON-RPC `tools/list` returned 11 tools.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card claims
  remain blocked.
- Aggregate MCP analytics, 7 days: `tools/list=179792`,
  `initialize=28341`, and `tools/call=401`.
- Aggregate MCP tool calls, 7 days: `search_agents=149`, `check_url=89`,
  `get_site_details=67`, `get_stats=30`, `submit_site=20`,
  `verify_mcp=13`, `find_mcp_servers=9`, `list_categories=8`,
  `recent_additions=6`, `get_top_sites=6`, and `register_monitor=4`.
- Sanitized aggregate query themes included leveraged ETF research, stock and
  ETF price-history APIs, model/API pricing, marketplace data, ecommerce
  product search, and local-event discovery.
- Aggregate traffic, 168 hours: `/=3375`, `/badge/xquik.com.svg=2652`,
  `/.well-known/commerce.json=1306`, `/site/xquik.com=1105`,
  `/.well-known/ai-plugin.json=580`, `/llms.txt=424`,
  `/openapi.yaml=372`, `/api/v1/catalog=316`, `/robots.txt=280`,
  `/api/v1/checkout=248`, `/api/v1/quote=248`, `/api/v1/search=228`,
  `/api/v1/submit=146`, `/.well-known/agent.json=78`, and `/top=75`.
- Aggregate referrers, 168 hours: Google contributed 617 visits and `/score`
  contributed 78 visits. Treat these as aggregate discovery and score-flow
  signals only.
- Public score/fix route checks: `polygon.io` had a public profile and fix
  route with a 60/100 profile; `sec-api.io` had a public profile and fix route
  with a 15/100 profile. Two other sampled finance/API brands were not present
  in the public profile index during this run.
- Latest local monitor worker proof, 2026-05-25: completed normally with five
  due monitors; aggregate outcome was two first-check zero-score quarantines,
  two first-check partial or low-score checks in finance/market-data-style
  surfaces, and one stable high-score check.

## Segment

This is narrower than the existing generic finance and market-state briefs.
The useful owner segment is ETF, market-data, and financial research API
publishers that need machine-readable source contracts for agents:

- canonical quote/history endpoints or feed contracts,
- plan/auth/rate-limit metadata,
- OpenAPI or structured API docs,
- `llms.txt` and robots policy that point agents to canonical sources,
- MCP or API manifests where intended,
- monitorable drift on profiles and discovery files.

The owner path should start with `/score`, not a paid offer. High-score owners
go to free monitor/report/badge proof. Partial-score owners get a
missing-surface checklist before any score-fix remediation. Quarantined monitor
cases stay private/admin-only.

## Draft Brief

Agents looking up ETF or market-data sources need stable contracts, not another
marketing page.

For financial data products, the public surface should tell an agent which
quote/history endpoint is canonical, what the auth and rate limits are, where
the OpenAPI or API docs live, and how to detect when those surfaces drift.
Not Human Search can check those source contracts and monitor them after
deploys.

The safe owner flow is: run `/score`, fix missing public surfaces, then attach
free monitoring to the report. Paid remediation only belongs after a fresh
score confirms concrete missing machine-readable surfaces.

## Next Gated Action

Prepare exactly one gated owner-channel touch, channel post, directory
candidate, or product-handoff test for ETF, market-data, financial research,
trading-source, payment-data, or finance API owners.

Before external use, refresh `/api/v1/stats`, `/api/v1/categories`,
`/api/v1/top?category=finance&limit=8`, `/api/v1/top?category=data&limit=8`,
`/api/v1/top?category=security&limit=8`, `/score`, `/monitor`, `/report`,
representative `/site/{host}` pages, high-score and partial-score
`/fix/{host}` routes, `/mcp` JSON-RPC `tools/list`,
`/.well-known/mcp.json`, `/.well-known/agent.json`,
`/.well-known/agent-card.json`, `/.well-known/commerce.json`,
`/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`,
`/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`,
`/feed.xml`, aggregate `/api/v1/admin/mcp?days=7`, aggregate
`/api/v1/admin/traffic?hours=168`, and latest monitor worker proof.

## Claims To Avoid

Do not imply ETF, market-data, finance, payment, research, API, top-list,
referrer, or profiled domains are customers, partners, endorsements, paid
leads, monitor registrations, badge-install consent, private demand, completed
payments, revenue, trading performance, market-data accuracy, price freshness,
financial advice, investment suitability, payment reliability, API uptime,
data completeness, crawler compliance, SEO lift, A2A support while
`/.well-known/agent-card.json` is 404, x402/ACP/SPT/MPP support for NHS, paid
ranking placement, preferred inclusion, or score-methodology bypass.

Do not publish raw user-agent strings, raw MCP queries, private monitor rows,
raw checkout URLs, payment identifiers, buyer emails, private score-fix rows,
or private customer identifiers.
