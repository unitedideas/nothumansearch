# API Subscribe Price Metadata Handoff - 2026-05-27

Automation: `business-marketer-not-human-search`

Scope: no public action, outreach, browser/Computer Use, account creation,
product-code edit, deploy, broad crawl, checkout completion, or QLimit/global
queue write. This is a sanitized scout artifact for a later gated operator.

## Evidence

- Public stats: `total_sites=4174`, `avg_score=35`, and
  `top_category=developer`.
- Public categories: `developer=1231`, `ai-tools=905`, `data=402`,
  `finance=192`, `productivity=171`, `ecommerce=149`,
  `communication=118`, `security=113`, `health=59`, `jobs=26`,
  `education=21`, and `news=12`.
- `/.well-known/commerce.json` returned 200 and listed API Starter, Pro, and
  Scale plans with `$19/mo`, `$49/mo`, and `$199/mo` displays.
- `/.well-known/agent.json` returned 200 and pointed agents to
  `/api/v1/api-keys/subscribe`, `/api/v1/catalog`, `/api/v1/quote`, and
  `/api/v1/checkout`.
- `GET /api/v1/api-keys/subscribe` returned starter/pro/scale plan ids and
  monthly limits, but `amount_cents` and `price_display` were null on all
  three plans.
- Live public surfaces returned 200 for `/llms.txt`,
  `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/api/v1`, `/api/v1/catalog`, `/score`,
  `/monitor`, and `/mcp` JSON-RPC `tools/list`.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card claims
  remain blocked.
- Live MCP `tools/list` returned 11 tools: `search_agents`,
  `get_site_details`, `get_stats`, `submit_site`, `check_url`,
  `verify_mcp`, `register_monitor`, `list_categories`, `get_top_sites`,
  `recent_additions`, and `find_mcp_servers`.
- Aggregate MCP analytics, 7 days: `tools/list=181084`,
  `initialize=28579`, and `tools/call=401`.
- Aggregate MCP tool calls, 7 days: `search_agents=149`, `check_url=89`,
  `get_site_details=67`, `get_stats=30`, `submit_site=20`,
  `verify_mcp=13`, `find_mcp_servers=9`, `list_categories=8`,
  `recent_additions=6`, `get_top_sites=6`, and `register_monitor=4`.
- Aggregate traffic, 168 hours: `/=3360`,
  `/badge/xquik.com.svg=2641`, `/.well-known/commerce.json=1341`,
  `/site/xquik.com=1105`, `/.well-known/ai-plugin.json=595`,
  `/llms.txt=438`, `/openapi.yaml=380`, `/api/v1/catalog=323`,
  `/api/v1/checkout=255`, `/api/v1/quote=255`, `/api/v1/search=229`,
  and `/api/v1/submit=146`.
- Aggregate referrers, 168 hours: Google contributed 637 visits and `/score`
  contributed 78 visits. Treat these as aggregate discovery signals only.
- Latest local monitor worker proof, 2026-05-25: completed normally with
  five due monitors; aggregate outcome was two first-check zero-score
  quarantines, two first-check partial or low-score checks, and one stable
  high-score check.

## Read

The agent-commerce packet is stronger than the older subscribe GET handoff:
the commerce manifest tells agents the monthly prices, but the direct
subscribe GET omits machine-readable price values. That creates a conversion
gap for agents that enter through the subscription endpoint instead of the
catalog or commerce manifest.

This should be treated as a product handoff before public API-plan promotion,
not as a reason to publish pricing copy from the recurring worker.

## Candidate Test

Run one gated API-subscription conversion test:

1. Compare `/llms.txt`, `/.well-known/commerce.json`, `/api/v1/catalog`, and
   `GET /api/v1/api-keys/subscribe`.
2. Decide whether subscribe GET should expose the same price metadata as the
   commerce manifest or intentionally stay price-light.
3. If public, align subscribe GET with the manifest before API-plan directory
   or owner-channel copy.
4. If intentionally hidden, keep future copy on catalog/commerce-manifest
   language and avoid promising a fully price-readable subscribe handoff.
5. Do not complete checkout from the recurring worker.

## Boundaries

Do not imply catalog, quote, checkout, subscribe, MCP, profile, badge,
referrer, or manifest traffic proves customers, endorsements, partners, paid
leads, private demand, completed payments, revenue, payment reliability,
pricing accuracy, x402/ACP/SPT/MPP support for NHS, paid ranking placement,
preferred inclusion, A2A support while `/.well-known/agent-card.json` is 404,
or score-methodology bypass.

Do not publish raw checkout URLs, payment identifiers, buyer emails, private
monitor rows, private score-fix rows, private query logs, raw user-agent
strings, or private customer identifiers.
