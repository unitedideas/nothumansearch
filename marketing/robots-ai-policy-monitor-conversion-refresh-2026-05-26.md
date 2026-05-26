# Robots AI Policy Monitor Conversion Refresh

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

- Public stats: `total_sites=4173`, `avg_score=35`, `top_category=developer`.
- Public category counts: `developer=1230`, `ai-tools=905`, `other=774`,
  `data=402`, `finance=192`, `productivity=171`, `ecommerce=149`,
  `communication=118`, `security=113`, `health=59`, `jobs=26`,
  `education=21`.
- Live public surfaces returned 200: `/robots.txt`, `/llms.txt`,
  `/openapi.yaml`, `/feed.xml`, `/api/v1`, `/api/v1/catalog`,
  `/.well-known/agent.json`, and `/.well-known/commerce.json`.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card claims
  remain blocked.
- Live `robots.txt` explicitly allows major AI crawlers and links sitemap,
  RSS, `llms.txt`, OpenAPI, AI plugin manifest, MCP manifest, MCP endpoint,
  registry auth proof, and security contact.
- Public developer top-list examples with complete or near-complete public
  readiness signals included `agentprobe.fly.dev=100`, `xquik.com=100`,
  `deadends.dev=100`, `agentdomainsearch.com=100`,
  `blackveilsecurity.com=100`, `agentndx.ai=100`, `flowzap.xyz=100`, and
  `entia.systems=100`. Treat these as public readiness examples only.
- Public AI-tools top-list examples included Foundry-owned dogfood domains
  plus third-party high-score examples: `amalgix.io=100`,
  `claudereviews.com=100`, `chainray.online=100`, `memestack.ai=100`, and
  `sincetmw.ai=100`. Label Foundry-owned examples before external use.
- Score-band route checks: high-score `/fix/nothumansearch.ai` and
  `/fix/xquik.com` returned the already-meets-target handoff; partial-score
  `/fix/manifest.ly` returned a live remediation page.
- Aggregate MCP analytics, 7 days: `tools/list=177292`,
  `initialize=26901`, and `tools/call=370`.
- Aggregate MCP tool calls, 7 days: `search_agents=146`, `check_url=85`,
  `get_site_details=54`, `get_stats=27`, `submit_site=20`,
  `verify_mcp=12`, `list_categories=7`, `find_mcp_servers=7`,
  `recent_additions=5`, `register_monitor=4`, and `get_top_sites=3`.
- Aggregate traffic, 168 hours: `/=3390`, `/badge/xquik.com.svg=2663`,
  `/.well-known/commerce.json=1340`, `/site/xquik.com=1105`,
  `/.well-known/ai-plugin.json=606`, `/llms.txt=441`,
  `/openapi.yaml=391`, `/api/v1/catalog=322`, `/robots.txt=289`,
  `/api/v1/checkout=255`, `/api/v1/quote=255`, `/api/v1/search=224`,
  `/api/v1/submit=146`, `/.well-known/agent.json=78`, `/api/v1=77`,
  `/top=77`, `/.well-known/mcp.json=76`, `/score=74`,
  `/site/openai.com=73`, and `/guide=71`.
- Aggregate referrers, 168 hours: `google.com=573`, `/score=77`, and a
  public high-score third-party referrer at `aurelianflo.com=57`.
- Latest local monitor worker proof, 2026-05-25: completed normally with
  five due monitors; aggregate outcome was two first-check zero-score
  quarantines, two first-check partial or low-score checks, and one stable
  high-score check.

## Segment

This segment is narrower than the older robots-policy and generic discovery
packets. The fresh signal is that `/robots.txt` is itself a top
machine-readable route, adjacent to `llms.txt`, OpenAPI, commerce metadata,
catalog, quote, checkout, and MCP manifest traffic.

The owner-side angle:

- `robots.txt` is not just a crawler allow/deny file. For agent-facing sites,
  it can be a discovery pointer to sitemap, RSS, `llms.txt`, OpenAPI, MCP,
  plugin manifests, security contact, and automated-access boundaries.
- High-score owners should monitor this surface because deploys and CMS
  changes can silently remove AI-crawler policy or discovery links.
- Partial-score owners should run `/score` first and fix missing public
  source surfaces before any paid remediation.
- Zero-score or quarantined monitor cases stay private/admin-only until
  reviewed.

## Draft Brief

Agents inspect more than API docs.

For agent-facing products, `robots.txt` can be the first public contract a
crawler sees: which bots are allowed, where the sitemap lives, where `llms.txt`
is, whether OpenAPI or MCP exists, and who to contact for security or abuse.

Not Human Search can check whether those source pointers exist and keep
monitoring them after deploys. High-score owners should treat the public report
and free monitor as proof. Partial-score owners should start with `/score` and
repair the missing surfaces before asking agents to trust the site.

## Next Gated Action

Prepare exactly one gated owner-channel touch, channel post, or product-handoff
test for developer, AI-tool, API, or ecommerce owners that use `robots.txt`,
`llms.txt`, OpenAPI, MCP, catalog, quote, checkout, RSS, or security-contact
metadata as agent-facing source contracts.

Before external use, refresh `/api/v1/stats`, `/api/v1/categories`,
`/api/v1/top?category=developer&limit=8`,
`/api/v1/top?category=ai-tools&limit=8`, `/score`, `/monitor`, `/report`,
representative `/site/{host}` pages, high-score and partial-score
`/fix/{host}` routes, `/robots.txt`, `/mcp` JSON-RPC `tools/list`,
`/.well-known/mcp.json`, `/.well-known/agent.json`,
`/.well-known/agent-card.json`, `/.well-known/commerce.json`,
`/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`,
`/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`,
`/feed.xml`, aggregate `/api/v1/admin/mcp?days=7`, aggregate
`/api/v1/admin/traffic?hours=168`, and latest monitor worker proof.

## Claims To Avoid

Do not imply crawler compliance, legal permission, SEO lift, AI-crawler
endorsement, uptime, security certification, source completeness, data
freshness, customer demand, private demand, paid leads, completed payments,
revenue, badge-install consent, partner endorsement, x402/ACP/SPT/MPP support
for NHS, A2A support while `/.well-known/agent-card.json` is 404, paid
placement, preferred inclusion, or score-methodology bypass.

Do not publish raw user-agent strings, private query logs, private monitor
rows, raw checkout URLs, payment identifiers, buyer emails, private score-fix
rows, or private customer identifiers.
