# API Commerce Score-Band Conversion

Run: 2026-05-26
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, checkout completion, or QLimit/global-queue write was
performed. This is a sanitized scout artifact for a later gated operator.

No raw users, emails, API keys, checkout URLs, payment identifiers, private
monitor rows, private score-fix rows, private query logs, raw user-agent
strings, buyer data, or customer identifiers are included here.

## Evidence

- Public stats: `total_sites=4172`, `avg_score=35`, `top_category=developer`.
- Public categories: `developer=1230`, `ai-tools=904`, `other=774`,
  `data=402`, `finance=192`, `productivity=171`, `ecommerce=149`,
  `communication=118`, `security=113`, `health=59`, `jobs=26`,
  `education=21`, `news=12`, and `spam=1`.
- Live public surfaces returned 200: `/score`, `/monitor`, `/report`,
  `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`,
  `/api/v1/catalog`, `/openapi.yaml`, `/feed.xml`, and `/mcp` JSON-RPC
  `tools/list`.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card claims
  remain blocked.
- MCP `tools/list` returned 11 tools.
- Aggregate MCP analytics, 7 days: `tools/list=176176`,
  `initialize=25831`, and `tools/call=377`.
- Sanitized aggregate MCP query families in the sampled top-query set:
  `developer_api=7`, `commerce_payment=2`, and `other=21`.
- Aggregate traffic, 168 hours: `/=3416`,
  `/badge/xquik.com.svg=2648`, `/.well-known/commerce.json=1345`,
  `/site/xquik.com=1106`, `/.well-known/ai-plugin.json=607`,
  `/llms.txt=441`, `/openapi.yaml=377`, `/api/v1/catalog=323`,
  `/api/v1/checkout=256`, `/api/v1/quote=256`, `/api/v1/search=219`,
  `/api/v1/submit=146`.
- Public security top-list examples: `feedoracle.io=100`, `ansvar.eu=100`,
  `agent-module.dev=95`, `tickerr.ai=85`, `rnwy.com=80`,
  `hipaaagent.ai=80`, `easysend.co=80`, and `file.kiwi=80`.
- Public data top-list examples: `api.headlessoracle.com=100`,
  `api.contrastcyber.com=100`, `dchub.cloud=95`, `daedalmap.com=90`,
  `api.theartofservice.com=90`, `api.agentry.com=90`,
  `api.socialintel.dev=90`, and `blocklens.co=90`.
- Score-fix route checks: high-score `/fix/nothumansearch.ai` and
  `/fix/api.contrastcyber.com` returned the already-meets-target handoff;
  partial-score `/fix/manifest.ly` and `/fix/tickerr.ai` returned live
  remediation pages.
- Latest monitor worker proof: 2026-05-25 processed five due monitors,
  kept two zero-score first checks in quarantine, recorded two partial or
  low first checks, and confirmed one stable high-score monitor.

## Segment

The current useful segment is not another broad directory submission. It is a
score-band conversion path for API, commerce, security-data, and developer-data
owners whose agents or users are already touching machine-readable commerce and
API surfaces.

The safe owner-side message:

- Machine clients are reading `commerce.json`, OpenAPI, catalog, quote,
  checkout, API root, and MCP surfaces.
- Owners should run `/score` before any paid remediation.
- High-score owners should use free monitor, public report, and badge proof.
- Partial-score owners should use the missing-surface checklist before
  score-fix remediation.
- API-heavy users should stay on API-key/catalog paths when docs remain useful.

## Draft Channel Angle

Agents do not only read API docs. They also read commerce manifests, catalogs,
quote endpoints, checkout handoffs, and plugin metadata before deciding what is
safe to call or recommend.

Not Human Search can make that readiness visible without selling placement:
score the public surfaces, route high-score owners to free monitoring and report
proof, and route partial-score owners to missing-surface repair before any paid
remediation.

## Next Gated Action

Prepare one owner-channel or product-handoff test from this packet for API
owners, developer-data products, security-data products, payment/commerce API
owners, or agent-commerce sellers.

Before external use, refresh `/api/v1/stats`, `/api/v1/categories`,
`/api/v1/top?category=data&limit=8`, `/api/v1/top?category=security&limit=8`,
`/score`, `/monitor`, `/report`, representative `/site/{host}` pages,
high-score and partial-score `/fix/{host}` routes, `/mcp` JSON-RPC
`tools/list`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
`/.well-known/agent-card.json`, `/.well-known/commerce.json`,
`/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`,
`/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`,
`/feed.xml`, aggregate `/api/v1/admin/mcp?days=7`, aggregate
`/api/v1/admin/traffic?hours=168`, and the latest monitor-check worker proof.

Verify the active Foundry/Owl-owned account identity before public use, check
`marketing/social-post-ledger.json` if present plus `outreach/distribution_log.csv`
and sync-state public-action locks, and avoid `modelcontextprotocol/*` plus
`punkpeye/*` surfaces from `unitedideas`.

## Claims To Avoid

Do not imply API quality, security certification, threat-intelligence accuracy,
data freshness, pricing accuracy, checkout reliability, completed payments,
revenue, customer demand, private demand, paid leads, endorsements, badge-install
consent, seller certification, x402/ACP/SPT/MPP support for NHS, paid ranking
placement, preferred inclusion, A2A support while
`/.well-known/agent-card.json` is 404, or score-methodology bypass.
