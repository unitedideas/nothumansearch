# Aurelianflo High-Score Referrer Conversion

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

- Public stats: 4,172 indexed sites, average score 35, top category
  `developer`.
- Public categories: `developer=1230`, `ai-tools=904`, `other=774`,
  `data=402`, `finance=192`, `productivity=171`, `ecommerce=149`,
  `communication=118`, `security=113`, `health=59`, `jobs=26`,
  `education=21`, `news=12`, and `spam=1`.
- Live public surfaces returned 200: `/score`, `/monitor`, `/report`,
  `/newest`, `/top`, `/mcp-servers`, `/openapi-apis`, `/llms-txt-sites`,
  `/api/v1`, `/api/v1/catalog`, `/.well-known/mcp.json`,
  `/.well-known/agent.json`, `/.well-known/commerce.json`,
  `/.well-known/ai-plugin.json`, `/llms.txt`, `/openapi.yaml`, and
  `/feed.xml`.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card claims
  remain blocked.
- MCP `tools/list` returned 11 tools: `search_agents`, `get_site_details`,
  `get_stats`, `submit_site`, `check_url`, `verify_mcp`, `register_monitor`,
  `list_categories`, `get_top_sites`, `recent_additions`, and
  `find_mcp_servers`.
- Aggregate MCP analytics, 7 days: `tools/list=176629`,
  `initialize=26316`, and `tools/call=372`.
- Aggregate MCP tool calls, 7 days: `search_agents=146`, `check_url=86`,
  `get_site_details=55`, `get_stats=27`, `submit_site=20`,
  `verify_mcp=12`, `list_categories=7`, `find_mcp_servers=7`,
  `recent_additions=5`, `register_monitor=4`, and `get_top_sites=3`.
- Aggregate traffic, 168 hours: `/=3381`, `/badge/xquik.com.svg=2650`,
  `/.well-known/commerce.json=1345`, `/site/xquik.com=1105`,
  `/.well-known/ai-plugin.json=607`, `/llms.txt=441`,
  `/openapi.yaml=391`, `/api/v1/catalog=323`, `/api/v1/quote=256`,
  `/api/v1/checkout=256`, `/api/v1/search=223`, `/api/v1/submit=146`,
  `/digest=127`, `/about=88`, `/.well-known/agent.json=78`, and
  `/api/v1=77`.
- Aggregate referrers, 168 hours: `google.com=572`, `/score=79`, and a
  public third-party high-score referrer at `aurelianflo.com=57`.
- Public `/site/aurelianflo.com` returned 200 and showed a 100/100 profile
  with API, OpenAPI, and MCP signals visible in the public page text.
- Public `/fix/aurelianflo.com` returned the already-meets-target handoff,
  not a paid remediation intake.
- Public badge SVG routes returned 200 for `xquik.com`, `aidevboard.com`,
  `8bitconcepts.com`, and `aurelianflo.com`.
- Latest local monitor worker proof, 2026-05-25: completed normally with
  five due monitors; aggregate outcome was two first-check zero-score
  quarantines, two first-check partial or low-score checks, and one stable
  high-score check.

## Segment

This is a high-score profile-followthrough segment, not a demand or customer
claim. The public traffic/referrer shape suggests a narrow conversion test:
when a high-score owner, referrer, or badge/profile reader lands on NHS, the
next step should be free monitor/report/badge proof rather than paid
remediation.

The safe owner-side message:

- A 100/100 profile is a proof asset, not a reason to buy score-fix.
- High-score owners should register free monitoring so deploys do not erase
  their public machine-readable surfaces.
- Badge/report links should route owners and readers to public proof, not to
  paid ranking or preferred placement.
- Partial-score owners still start with `/score` and a missing-surface
  checklist before remediation.

## Draft Channel Angle

Some sites already expose the public source contract agents need: `llms.txt`,
OpenAPI, structured API metadata, MCP, agent manifests, and clear automated
access boundaries.

For those owners, the useful NHS path is not score repair. It is proof and
drift monitoring: a public report, badge, and free monitor so deploys do not
silently remove the signals agents rely on.

Partial-score owners should still run `/score` first and fix the missing
machine-readable surfaces before considering remediation.

## Next Gated Action

Prepare exactly one gated owner-channel touch, product-handoff test, or
high-score profile followthrough experiment for owners whose public profiles,
badges, or referrers already show high agent-readiness.

Before external use, refresh `/api/v1/stats`, `/api/v1/categories`, `/score`,
`/monitor`, `/report`, representative high-score and partial-score
`/site/{host}` pages, high-score and partial-score `/fix/{host}` routes, badge
SVG behavior, `/mcp` JSON-RPC `tools/list`, `/.well-known/mcp.json`,
`/.well-known/agent.json`, `/.well-known/agent-card.json`,
`/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`,
`/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`,
`/openapi.yaml`, `/feed.xml`, aggregate `/api/v1/admin/mcp?days=7`,
aggregate `/api/v1/admin/traffic?hours=168`, and latest monitor worker proof.

Verify the active Foundry/Owl-owned account identity before public use, check
`marketing/social-post-ledger.json` if present plus
`outreach/distribution_log.csv` and sync-state public-action locks, and avoid
`modelcontextprotocol/*` plus `punkpeye/*` surfaces from `unitedideas`.

## Claims To Avoid

Do not imply `aurelianflo.com`, `xquik.com`, ADB, 8bitconcepts, badge/profile
routes, referrers, MCP clients, or profiled domains are customers, partners,
endorsements, paid leads, monitor registrations, badge-install consent, private
demand, completed payments, revenue, uptime proof, crawler compliance, SEO
lift, A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/SPT/MPP
support for NHS, paid placement, preferred inclusion, paid ranking, or
score-methodology bypass.
