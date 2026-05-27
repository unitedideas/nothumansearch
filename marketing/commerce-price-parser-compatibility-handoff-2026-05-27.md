# Commerce Price Parser Compatibility Handoff - 2026-05-27

Automation: `business-marketer-not-human-search`

Scope: no public action, outreach, browser/Computer Use, account creation,
product-code edit, deploy, broad crawl, checkout completion, or QLimit/global
queue write. This is a sanitized scout artifact for a later gated operator.

## Evidence

- Public stats: `total_sites=4179`, `avg_score=35`, and
  `top_category=developer`.
- Public category counts: `developer=1232`, `ai-tools=904`,
  `other=780`, `data=401`, `finance=192`, `productivity=172`,
  `ecommerce=149`, `communication=117`, `security=112`,
  `health=60`, `jobs=26`, and `education=21`.
- `/.well-known/mcp.json` returned 200 and exposed 11 tools.
- `/llms.txt` returned 200 and advertised the same 11 MCP tools plus
  starter/pro/scale API plans at `$19/mo`, `$49/mo`, and `$199/mo`.
- `/.well-known/commerce.json` returned 200 and exposed nested
  `price.amount`, `price.currency`, and `price.display` for score-fix and
  API plans.
- `/api/v1/catalog` returned 200 and exposed the same nested `price` objects.
- `GET /api/v1/api-keys/subscribe` returned 200 and exposed nested `price`
  objects plus monthly limits, but `amount_cents` stayed null for each plan.
- A flat-field parser looking for `price_cents`, `amount_cents`, top-level
  `pricing`, or top-level `payment_modes` sees nulls on the same surfaces that
  nested-price-aware agents can read.
- Aggregate MCP analytics, 7 days: `tools/list=183009`,
  `initialize=28316`, and `tools/call=391`.
- Aggregate MCP tool calls, 7 days: `search_agents=144`, `check_url=88`,
  `get_site_details=67`, `get_stats=27`, `submit_site=20`,
  `verify_mcp=13`, `find_mcp_servers=10`, and `register_monitor=4`.
- Aggregate traffic, 168 hours: `/=3405`, `/badge/xquik.com.svg=2653`,
  `/.well-known/commerce.json=1295`, `/site/xquik.com=1107`,
  `/.well-known/ai-plugin.json=576`, `/llms.txt=427`,
  `/openapi.yaml=373`, `/api/v1/catalog=313`,
  `/api/v1/checkout=246`, `/api/v1/quote=246`, and
  `/api/v1/search=232`.
- High-score `/fix/xquik.com` and `/fix/aidevboard.com` returned 200 with
  already-meets-target language, so high-score owner routes should go to
  monitor/report/badge proof rather than remediation copy.

## Read

The commerce packet is usable for nested-price-aware agents, but weaker for
simple crawlers and directory importers that normalize product prices from flat
fields. That matters because `/.well-known/commerce.json`, `/api/v1/catalog`,
`/api/v1/quote`, and `/api/v1/checkout` are all receiving material aggregate
traffic.

This is a product/sales handoff before agent-commerce directory promotion. It
should not block existing catalog use, but public copy should avoid claiming a
fully flat-price-readable commerce contract until the intended schema boundary
is decided.

## Candidate Test

Run one gated commerce-parser compatibility test:

1. Compare `/.well-known/commerce.json`, `/api/v1/catalog`,
   `GET /api/v1/api-keys/subscribe`, `/api/v1/quote`, and `/api/v1/checkout`.
2. Decide whether flat compatibility fields should be added alongside nested
   `price`, or whether nested `price` is the only supported public contract.
3. If flat compatibility is supported, align score-fix and API-plan products
   across commerce, catalog, subscribe GET, OpenAPI, and llms.txt.
4. If nested-only is intentional, keep future directory packets on the nested
   schema and avoid targets that require flat `amount_cents` or `price_cents`.
5. Do not complete checkout from the recurring worker.

## Boundaries

Do not imply commerce, catalog, quote, checkout, subscribe, profile, badge,
referrer, or MCP traffic proves customers, endorsements, partners, paid leads,
private demand, completed payments, revenue, payment reliability, pricing
accuracy, x402/ACP/SPT/MPP support for NHS, paid ranking placement, preferred
inclusion, A2A support while `/.well-known/agent-card.json` is 404, or
score-methodology bypass.

Do not publish raw checkout URLs, activation URLs, payment identifiers, buyer
emails, private monitor rows, private score-fix rows, private query logs, raw
user-agent strings, or private customer identifiers.
