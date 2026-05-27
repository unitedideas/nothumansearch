# API Plan Price Metadata Conversion Gap

Run: 2026-05-26
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
- Public category counts: `developer=1231`, `ai-tools=905`, `other=774`,
  `data=402`, `finance=192`, `productivity=171`, `ecommerce=149`,
  `communication=118`, `security=113`, `health=59`, `jobs=26`,
  `education=21`, `news=12`, and `spam=1`.
- `/llms.txt` advertises paid API plans as starter `$19/mo`, pro `$49/mo`,
  and scale `$199/mo`, with limits of 1,000, 10,000, and 100,000 calls.
- `GET /api/v1/api-keys/subscribe` returned plan ids and limits for
  `nhs_api_starter`, `nhs_api_pro`, and `nhs_api_scale`, but each plan's
  `amount_cents` field was `null`.
- Live public surfaces returned 200 for `/api/v1/catalog`,
  `/.well-known/commerce.json`, `/.well-known/agent.json`, `/api/v1`,
  `/llms.txt`, `/openapi.yaml`, and `/mcp`.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card claims
  remain blocked.
- Aggregate MCP analytics, 7 days: `tools/list=179150`,
  `initialize=28155`, and `tools/call=390`.
- Aggregate MCP tool calls, 7 days: `search_agents=149`, `check_url=90`,
  `get_site_details=62`, `get_stats=28`, `submit_site=20`,
  `verify_mcp=13`, `find_mcp_servers=8`, `list_categories=7`,
  `recent_additions=5`, `register_monitor=4`, and `get_top_sites=4`.
- Aggregate traffic, 168 hours: `/=3376`, `/badge/xquik.com.svg=2649`,
  `/.well-known/commerce.json=1305`, `/site/xquik.com=1105`,
  `/.well-known/ai-plugin.json=582`, `/llms.txt=427`,
  `/openapi.yaml=376`, `/api/v1/catalog=315`, `/robots.txt=284`,
  `/api/v1/checkout=248`, `/api/v1/quote=248`, `/api/v1/search=227`,
  `/api/v1/submit=146`, `/digest=89`, and `/about=88`.
- Score-band route checks: high-score `/fix/api.headlessoracle.com`,
  `/fix/feedoracle.io`, `/fix/xquik.com`, and `/fix/nothumansearch.ai`
  returned the already-meets-target monitor handoff; partial-score
  `/fix/manifest.ly` returned a live remediation page.
- Latest local monitor worker proof, 2026-05-25: completed normally with
  five due monitors; aggregate outcome was two first-check zero-score
  quarantines, two first-check partial or low-score checks, and one stable
  high-score check.

## Segment

The agent-commerce and API-plan traffic is real enough to merit a conversion
test, but the public machine-readable subscribe packet has a price-metadata
gap. Agents can see the plan ids and monthly limits, yet cannot read a numeric
price from the same machine contract. Humans and `llms.txt` see the price; the
API plan handoff does not carry it in `amount_cents`.

This should be treated as a conversion-readiness handoff before more API-plan
promotion. Directory or channel copy can continue to say NHS has paid API
plans only after the live packet shows the same plan prices that the public
docs advertise.

## Draft Operator Note

NHS has the right agent-commerce surfaces live: catalog, quote, checkout,
commerce manifest, agent manifest, API root, OpenAPI, `llms.txt`, and MCP.

The weak point is the API subscription contract. `llms.txt` says starter,
pro, and scale cost $19, $49, and $199 per month, but
`GET /api/v1/api-keys/subscribe` returns `amount_cents: null` for each plan.
That makes the plan handoff less useful to buyer agents and any directory that
validates machine-readable pricing.

## Next Gated Action

Prepare exactly one gated product-handoff or conversion test that:

- Refreshes `/api/v1/api-keys/subscribe`, `/llms.txt`,
  `/.well-known/commerce.json`, `/.well-known/agent.json`, `/api/v1/catalog`,
  `/api/v1/quote`, `/api/v1/checkout`, `/api/v1`, `/openapi.yaml`, `/mcp`
  JSON-RPC `tools/list`, aggregate admin MCP/traffic data, and latest monitor
  worker proof.
- Confirms whether `amount_cents` is expected to be public or intentionally
  hidden for API-key plans.
- If public, aligns API plan price metadata with `llms.txt` before any API-plan
  directory, owner-channel, or agent-commerce promotion.
- If intentionally hidden, updates future copy to avoid promising a fully
  price-readable API-plan contract.

## Boundaries

Do not imply API-plan route traffic, catalog traffic, checkout/quote traffic,
profile views, badge views, MCP clients, or listed domains are customers,
endorsements, partners, paid leads, private demand, completed payments,
revenue, payment reliability, pricing accuracy, crawler compliance, legal
permission, SEO lift, uptime proof, A2A support while
`/.well-known/agent-card.json` is 404, x402/ACP/SPT/MPP support for NHS, paid
ranking placement, preferred inclusion, or score-methodology bypass.

Do not publish raw user-agent strings, raw MCP queries, private monitor rows,
raw checkout URLs, payment identifiers, buyer emails, private score-fix rows,
or private customer identifiers.
