# Foundry Dogfood Badge Conversion Scout

Run: 2026-05-19T09:20Z
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
- `/score`, `/monitor`, `/report`, `/api/v1/catalog`,
  `/.well-known/commerce.json`, `/.well-known/agent.json`,
  `/.well-known/mcp.json`, `/llms.txt`, and `/openapi.yaml`: HTTP 200.
- `/.well-known/agent-card.json`: HTTP 404, so strict A2A/Agent Card claims
  remain gated.
- Public profile pages for `aidevboard.com`, `8bitconcepts.com`,
  `bringyour.ai`, and `nothumansearch.ai` each render score 100/100.
- Anonymous `/api/v1/site/{host}` detail reads returned HTTP 402 in this run,
  so use public profile pages or an approved keyed/internal read path before
  writing per-domain API-detail claims.

Aggregate admin evidence, sanitized:

- Last 168 hours traffic: `/=3537`, `/badge/xquik.com.svg=1717`,
  `/.well-known/commerce.json=1687`, `/.well-known/ai-plugin.json=799`,
  `/llms.txt=510`, `/site/xquik.com=483`, `/openapi.yaml=469`,
  `/api/v1/catalog=375`, `/api/v1/quote=348`, `/api/v1/checkout=348`,
  `/badge/aidevboard.com.svg=313`, `/badge/8bitconcepts.com.svg=302`,
  `/api/v1/submit=147`, `/top=114`, `/api/v1=96`,
  `/.well-known/mcp.json=95`, `/about=95`, `/guide=91`, `/newest=87`,
  `/score=76`, `/api/v1/check=60`.
- Referrer aggregates included `google.com=542`, `/score=117`,
  `/top=46`, `/mcp=35`, and `/site/chainray.online=34`.
- Last 7 days MCP analytics: `tools/list=153988`, `initialize=20103`,
  `tools/call=293`.
- Top called MCP tools: `search_agents=185`, `get_site_details=40`,
  `check_url=15`, `get_stats=12`, `verify_mcp=10`,
  `recent_additions=10`, `get_top_sites=8`, `find_mcp_servers=7`,
  `submit_site=4`, `list_categories=2`.

## Read

The badge loop is now showing two distinct signals:

1. Third-party badge/profile interest remains material through
   `/badge/xquik.com.svg` and `/site/xquik.com`.
2. Foundry-owned dogfood badges for `aidevboard.com` and `8bitconcepts.com`
   still receive meaningful aggregate traffic while both public profiles score
   100/100.

The useful next test is not another generic badge pitch. It is a dogfood proof
loop:

- High-score Foundry profiles should demonstrate the intended next step for
  strong owners: monitor the domain, share the badge/report, and keep discovery
  surfaces from drifting.
- Lower-score third-party owners should still be routed to `/score` before
  remediation.
- API-heavy visitors from catalog, quote, checkout, OpenAPI, and MCP surfaces
  should see the API-key plan path only where repeated machine access is the
  real need.

## Proposed Gated Test

Create one product-handoff or owner-channel test around Foundry-owned badge
dogfood:

- Use `aidevboard.com` and `8bitconcepts.com` as labeled Foundry-owned examples
  of high-score public profiles, not as customer proof.
- Add or draft a "what to do with a 100/100 profile" path: free monitoring,
  shareable report, badge proof, and periodic drift checks.
- Keep the third-party badge route separate: no claims about `xquik.com` being a
  customer, endorsement, paid lead, monitor registrant, or badge-install owner.

## Acceptance Guard

Before implementation or external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/score`, `/monitor`,
  `/report`, representative high-score and partial-score `/site/{host}` pages,
  high-score and partial-score `/fix/{host}` routes, badge SVG routes for the
  selected examples, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/agent-card.json`, `/.well-known/commerce.json`,
  `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`,
  `/openapi.yaml`, aggregate `/api/v1/admin/mcp?days=7`, and aggregate
  `/api/v1/admin/traffic?hours=168`.
- Verify active Foundry/Owl-owned account identity before public use, check
  `marketing/social-post-ledger.json` if present plus
  `outreach/distribution_log.csv` and sync-state public-action locks, and avoid
  `modelcontextprotocol/*` plus `punkpeye/*` surfaces from `unitedideas`.
- Use only aggregate route/referrer counts and public URLs in committed
  artifacts.
- Do not imply profiled domains are customers, endorsements, paid leads,
  private demand, completed payments, revenue, monitor registrations,
  badge-install consent, security/compliance certification, A2A support,
  x402/ACP support, paid ranking placement, preferred inclusion, or
  score-methodology bypass.
