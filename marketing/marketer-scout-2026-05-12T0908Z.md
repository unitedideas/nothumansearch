# NHS marketing scout segment - 2026-05-12T09:08Z

Automation: `business-marketer-not-human-search`

## Boundary

No outreach was sent. No posts, browser/Computer Use, account creation, deploys, full recrawls, product-code edits, or public submissions were performed.

## Live surface checks

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4238`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. Largest buckets: `developer=1300`, `ai-tools=892`, `other=775`, `data=403`, `finance=201`.
- `/.well-known/mcp.json`: advertises 11 tools.
- `GET /api/v1`: lists search, site, submit, stats, categories, score-check, monitor registration, commerce, and paid API-key routes.
- `/api/v1/catalog`: includes score-fix plus `starter`, `pro`, and `scale` API subscription products.
- `/llms.txt`: current site count is live, but category copy still says `/categories` returns "all 12 buckets" while the API returns 14.
- Anonymous `/api/v1/search?q=finance` still returns `quota_exceeded` for this worker: `used=100`, `limit=100`, with API-key purchase metadata.

## Sanitized aggregate checks

- MCP analytics, last 7 days: `tools/list=86049`, `initialize=11278`, `tools/call=434`, unknown tool names: 0.
- MCP tool calls: `search_agents=232`, `get_site_details=57`, `check_url=31`, `find_mcp_servers=28`, `get_stats=26`, `verify_mcp=23`, `get_top_sites=17`, `recent_additions=8`, `submit_site=7`, `list_categories=5`.
- MCP query themes: exact-brand/operator lookup, finance/trading research, video/filmmaking workflows, VPN panel/API lookup, AI cookbook/resource lookup, and MCP server lookup.
- Traffic, last 14 days: `/=3786`, `/.well-known/commerce.json=908`, `/.well-known/ai-plugin.json=483`, `/robots.txt=373`, `/llms.txt=338`, `/openapi.yaml=333`, `/api/v1/catalog=206`, `/api/v1/quote=197`, `/api/v1/checkout=197`, `/mcp-servers=83`, `/.well-known/mcp.json=82`.
- Score-fix intake remains 11 total rows: `real_candidate pending=3`, all now in the `7_29d` bucket; `test_like lead=1`, `test_like paid=2`, `test_like pending=5`. Raw rows were not written.
- Scheduled monitor proof remains current from 2026-05-11 07:30 PT: two due monitors processed, one zero-score monitor quarantined, one 100-score monitor stable. This artifact intentionally omits raw monitor domains and emails.

## Public shortlist evidence

Anonymous search is quota-blocked, but public `/api/v1/top` remains usable for non-row-level marketing scout evidence.

Finance/trading shortlist from `GET /api/v1/top?category=finance&limit=5`:

- `terminalfeed.io` - score 100, finance, full 7-signal readiness.
- `chartlibrary.io` - score 100, finance, full 7-signal readiness.
- `prereason.com` - score 100, finance, full 7-signal readiness.
- `devdrops.run` - score 95, finance, missing schema only in the returned signals.

Creative/media candidate source:

- Current MCP query themes include video generation, filmmaking, screenplay writing, and film-production management.
- Public category shortlist is weaker because those sites are mixed into `ai-tools`, not a dedicated creative/media category. The later brief should use the private API read path before naming exact creative/media targets.

## Draft brief angles

Finance/trading:

`Agents looking for market data and trading research need more than a homepage. NHS already has a finance category with 201 indexed sites and multiple score-100 examples exposing llms.txt, ai-plugin metadata, OpenAPI, structured APIs, MCP, AI-friendly robots, and schema. The useful comparison is not "best finance APIs" in general; it is which finance sites are ready for autonomous agents to inspect, call, and verify.`

Creative/media:

`NHS MCP query themes now include video generation, filmmaking, screenplay writing, and film-production management. That is a narrow owner-channel opportunity, but the public category model does not isolate creative/media yet. The safe next step is a private-key-backed scout read path that can produce a sanitized target shortlist without burning anonymous quota or exposing raw user queries.`

Agent-commerce/API-key buying:

`Agent traffic is already hitting NHS buying surfaces: commerce.json, catalog, quote, and checkout are all in the 14-day top-page set. The sales proof angle is that NHS is not only an agent-search index; it is also an agent-readable seller with API-key plans exposed through machine-readable catalog metadata.`

## Duplicate and channel checks

- `ops/sweeper/marketer-inbox.jsonl` already contains generic rows for query-intent briefing, private API read-path provisioning, category-count drift, score-results handoff, monitor quarantine, and API-key catalog exposure.
- `outreach/distribution_log.csv` is saturated with broad MCP/API/GEO directory PRs, gists, email pitches, and existing NHS score-check action distribution.
- Shared social ledger check found no current NHS vertical-brief post, but any later external publication still needs active account verification, duplicate fingerprinting, and a sync-state public-action lock.

## Appended intake rows

Rows appended to `ops/sweeper/marketer-inbox.jsonl`:

- `Prepare finance/trading agent-readiness brief from public top-category evidence.`
- `Prepare creative/media target shortlist after private NHS API read path exists.`
