# AI News Publisher Source Readiness

Run: 2026-05-25T07:08:49Z
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, checkout completion, or QLimit/global-queue write was
performed. This is a sanitized scout artifact for a later gated operator.

No raw users, emails, API keys, checkout URLs, payment identifiers, private
monitor rows, private score-fix rows, buyer data, raw user-agent strings,
private query logs, or customer identifiers are included here.

## Evidence

- Public stats: 4,180 indexed sites, average score 35, top category
  `developer`.
- Public categories: `developer=1,230`, `ai-tools=906`, `other=782`,
  `data=399`, `finance=195`, `productivity=171`, `ecommerce=146`,
  `communication=119`, `security=113`, `health=59`, `jobs=26`,
  `education=21`, `news=12`, and `spam=1`.
- Live public surfaces returned 200: `/score`, `/monitor`, `/report`,
  `/newest`, `/top`, `/feed.xml`, `/.well-known/mcp.json`,
  `/.well-known/agent.json`, `/.well-known/commerce.json`,
  `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`,
  `/llms.txt`, and `/openapi.yaml`.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card claims
  remain blocked.
- MCP `tools/list` returned 11 tools:
  `search_agents`, `get_site_details`, `get_stats`, `submit_site`,
  `check_url`, `verify_mcp`, `register_monitor`, `list_categories`,
  `get_top_sites`, `recent_additions`, and `find_mcp_servers`.
- Aggregate MCP analytics, 7 days: `tools/list=173,794`,
  `initialize=25,247`, and `tools/call=372`.
- Aggregate MCP tool calls, 7 days: `search_agents=147`, `check_url=86`,
  `get_site_details=58`, `get_stats=20`, `submit_site=20`,
  `verify_mcp=13`, `list_categories=7`, `find_mcp_servers=7`,
  `recent_additions=6`, `register_monitor=4`, and `get_top_sites=4`.
- Aggregate traffic, 168 hours: `/=3,379`, `/badge/xquik.com.svg=2,653`,
  `/.well-known/commerce.json=1,342`, `/site/xquik.com=1,088`,
  `/.well-known/ai-plugin.json=609`, `/llms.txt=438`,
  `/openapi.yaml=376`, `/api/v1/catalog=320`, `/robots.txt=293`,
  `/badge/aidevboard.com.svg=273`, `/api/v1/quote=256`,
  `/api/v1/checkout=256`, `/badge/8bitconcepts.com.svg=252`,
  `/api/v1/search=222`, `/favicon.ico=177`, and `/api/v1/submit=146`.
- Public news top-list examples: `informedclearly.com=70`,
  `hallucinationherald.com=65`, `biztoc.com=65`, `zadar.tv=55`,
  `aibtc.news=50`, `thesansasyonel.com=45`,
  `sansasyonelgazete.com=45`, `yubigeek.com=45`, `newsmesh.co=45`,
  `mansetetkisi.com=45`, `dly.to=45`, and `groundhog-day.com=20`.
- Latest local monitor worker proof remains the 2026-05-18 clean run:
  one due monitor processed and completed without quarantine.
- High-score score-fix route check: `/fix/nothumansearch.ai` returned 200
  with the already-meets-target handoff. Partial-score check:
  `/fix/manifest.ly` returned 200 with the paid remediation intake.

## Segment

This segment is narrower than the older news/media, publisher-feed, and local
news briefs. It is for AI-news publishers, market newsletters, current-events
publishers, topic-specific news sites, and feed-backed editorial products that
want agents to inspect their public source contract before quoting or
summarizing them.

Useful owner-side angle:

- `llms.txt` should say which archives, feeds, topic pages, bylines, update
  policies, and methodology pages are canonical for agents.
- RSS or feed metadata should be stable enough for agents to distinguish new
  items from evergreen pages.
- Schema.org should identify article, publisher, author, date, and update
  metadata where the owner controls it.
- OpenAPI, API, or MCP surfaces are useful only where the publisher intends
  automated access.
- Robots policy should state whether major AI crawlers are allowed.
- High-score publishers should route to free monitor registration, public
  report sharing, and badge/report proof.
- Partial-score publishers should start with `/score` and a missing-surface
  checklist before any paid remediation.

## Draft Brief

Agents reading news pages need a source contract, not another scraped headline.

For AI-news publishers, market newsletters, local editorial products, and
topic-specific media sites, the machine-readable surface should identify the
canonical feed, archive, update policy, article metadata, contact path, and AI
crawler policy. Not Human Search does not certify the editorial claims or
current-events accuracy. It checks whether an agent can inspect the public
source before relying on it.

High-score publishers should use the public report, badge, and free monitor
path. Partial-score publishers should run `/score`, fix the missing
machine-readable surfaces, and only then consider remediation.

## Owner Routing

- High-score publishers, newsletters, and feed-backed editorial products:
  route to free monitor registration, public report sharing, and badge/report
  proof.
- Partial-score publishers: route to `/score` first, then a missing-surface
  checklist before score-fix remediation.
- API-heavy media/data callers: route to API-key/catalog surfaces only when
  NHS docs remain useful.
- A2A or Agent Card claims stay blocked until `/.well-known/agent-card.json`
  exists.

## Claims To Avoid

Do not claim editorial accuracy, article freshness, fact-checking quality,
market-news truth, publication quality, feed completeness, byline accuracy,
local-news coverage, translation quality, copyright permission, crawler
compliance, customer demand, private demand, paid leads, completed payments,
revenue, endorsement, paid placement, preferred inclusion, A2A support while
`/.well-known/agent-card.json` is 404, x402/ACP/SPT/MPP support for NHS, or
score-methodology bypass.
