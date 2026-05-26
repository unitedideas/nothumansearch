# OpenAI Low-Score Profile Conversion

Run: 2026-05-26
Automation: `business-marketer-not-human-search`
Status: no-submit scout artifact; public use still requires account identity verification, duplicate checks, and a sync-state public-action lock.

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, checkout completion, or QLimit/global-queue write was
performed. This is a sanitized scout artifact for a later gated operator.

No raw users, emails, API keys, checkout URLs, payment identifiers, private
monitor rows, private score-fix rows, private query logs, raw user-agent
strings, buyer data, or customer identifiers are included here.

## Evidence

- Public stats: 4,174 indexed sites, average score 35, top category
  `developer`.
- Public categories: `developer=1,231`, `ai-tools=905`, `other=774`,
  `data=402`, `finance=192`, `productivity=171`, `ecommerce=149`,
  `communication=118`, `security=113`, `health=59`, `jobs=26`,
  `education=21`, `news=12`, and `spam=1`.
- Live public surfaces returned 200: `/site/openai.com`, `/fix/openai.com`,
  `/score`, `/monitor`, `/report`, `/.well-known/mcp.json`,
  `/.well-known/agent.json`, `/.well-known/commerce.json`,
  `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`,
  `/llms.txt`, `/openapi.yaml`, `/feed.xml`, and `/mcp`.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card claims
  remain blocked.
- Public `/site/openai.com` rendered `15/100`.
- `/fix/openai.com` returned 200, so the remediation handoff is live for this
  low-score model-provider profile. Refresh the rendered copy before any
  external use.
- Aggregate traffic, 168 hours: `/=3,398`, `/badge/xquik.com.svg=2,656`,
  `/.well-known/commerce.json=1,280`, `/site/xquik.com=1,106`,
  `/.well-known/ai-plugin.json=573`, `/llms.txt=421`,
  `/openapi.yaml=372`, `/api/v1/catalog=310`, `/robots.txt=281`,
  `/api/v1/checkout=243`, `/api/v1/quote=243`, `/api/v1/search=226`,
  `/api/v1/submit=146`, `/digest=129`, `/score=74`,
  `/site/openai.com=73`, `/guide=72`, `/api/v1=69`, and
  `/.well-known/mcp.json=67`.
- Aggregate referrers, 168 hours: Google contributed 608 visits and
  `https://nothumansearch.ai/score` contributed 78 visits. Treat these as
  aggregate discovery and score-flow signals only.
- Aggregate MCP analytics, 7 days: `tools/list=178,120`,
  `initialize=27,491`, and `tools/call=389`.
- Aggregate MCP tool calls, 7 days: `search_agents=149`, `check_url=90`,
  `get_site_details=62`, `get_stats=27`, `submit_site=20`,
  `verify_mcp=13`, `find_mcp_servers=8`, `list_categories=7`,
  `recent_additions=5`, `register_monitor=4`, and `get_top_sites=4`.
- Sanitized aggregate query themes included model gateways, function-calling
  API pricing, local quantized models, agent marketplace terms, finance/data
  APIs, ecommerce product search, and local events.
- Latest local monitor worker proof, 2026-05-25: completed normally with five
  due monitors; aggregate outcome was two first-check zero-score quarantines,
  two first-check partial or low-score finance/market-data style monitors, and
  one stable high-score monitor.

## Segment

This is narrower than the existing model-provider profile brief. It is about a
specific high-visibility public profile route where a major AI provider's
profile is receiving traffic and currently scores low enough to make the
owner-side path concrete.

Useful owner-side angle:

- Even major AI providers can have public machine-readable gaps from an
  agent-readiness perspective.
- The owner flow should start with the public profile and `/score`, then a
  missing-surface checklist, then remediation only if the refreshed score still
  supports it.
- High-score profiles elsewhere should keep routing to free monitor/report/badge
  proof rather than paid score-fix.
- Monitor quarantine cases stay private/admin-only and must not be used as
  public monitor-growth proof.

## Draft Brief

Agents do not care whether a provider is famous. They need a source contract
they can inspect.

The current public profile for `openai.com` scores 15/100 in Not Human Search.
That does not say anything about model quality, API reliability, pricing
accuracy, or provider preference. It says the public site, as currently
observed, is missing several machine-readable surfaces agents use to decide
which docs, APIs, policies, and support paths are canonical.

For model providers and model gateways, the useful path is simple: run
`/score`, fix the missing public surfaces, then monitor the profile so future
drift is visible.

## Owner Routing

- Low-score model-provider profiles: route to `/score`, a missing-surface
  checklist, and score-fix only after a fresh public score confirms the gap.
- High-score model-provider profiles: route to free monitor registration,
  public report sharing, and badge/report proof.
- API-heavy callers: route to API-key/catalog surfaces only when NHS docs
  remain useful.
- A2A or Agent Card claims stay blocked until `/.well-known/agent-card.json`
  exists.

## Claims To Avoid

Do not claim OpenAI is a customer, endorsement, partner, paid lead, monitor
registration, badge-install consent, private demand, completed payment, revenue,
model-quality proof, benchmark truth, pricing accuracy, function-calling
reliability, API reliability, uptime, provider preference, customer demand, paid
placement, preferred inclusion, A2A support while
`/.well-known/agent-card.json` is 404, x402/ACP/SPT/MPP support for NHS, or
score-methodology bypass.

Do not publish raw user-agent strings, private query logs, raw checkout URLs,
payment identifiers, buyer emails, private monitor rows, private score-fix rows,
or private customer identifiers.

## Next Gated Action

Prepare one owner-channel touch, channel post, product-handoff test, or model
provider profile conversion test around low-score major AI-provider profiles.
Before external use, refresh all live route probes, aggregate admin analytics,
monitor worker proof, representative high-score and partial-score `/site/{host}`
pages, and high-score plus partial-score `/fix/{host}` routes.
