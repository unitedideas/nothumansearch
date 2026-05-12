# NHS marketing scout segment - 2026-05-12T05:08Z

Automation: `business-marketer-not-human-search`

## Boundary

No outreach was sent. No posts, browser/Computer Use, account creation, deploys, full recrawls, product-code edits, or public submissions were performed.

## Live surface checks

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4238`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. Largest buckets: `developer=1300`, `ai-tools=892`, `other=775`, `data=403`, `finance=201`.
- `/.well-known/mcp.json`: advertises 11 tools.
- `GET /api/v1`: lists search, site, submit, stats, categories, score-check, monitor registration, commerce, and paid API-key routes.
- `/api/v1/catalog`: includes score-fix plus `starter`, `pro`, and `scale` API subscription products.
- `GET /api/v1/api-keys/subscribe`: returns a readable API-key plan contract with required fields and activation path.
- `llms.txt`: current site count is live, but category copy still says `/categories` returns "all 12 buckets" while the API returns 14.

## Sanitized admin aggregates

- MCP analytics, last 7 days: `tools/list=83080`, `initialize=11119`, `tools/call=435`, unknown tool names: 0.
- MCP tool calls: `search_agents=234`, `get_site_details=57`, `check_url=31`, `find_mcp_servers=28`, `get_stats=26`, `verify_mcp=23`, `get_top_sites=17`, `recent_additions=8`, `submit_site=7`, `list_categories=4`.
- MCP top-query themes still show finance/trading and creative/media intent: financial research, stock-market strategy, SEC/earnings sentiment, video generation, filmmaking, screenplay writing, and film-production management.
- Traffic, last 14 days: `/=3795`, `/.well-known/commerce.json=903`, `/.well-known/ai-plugin.json=482`, `/badge/xquik.com.svg=450`, `/badge/aidevboard.com.svg=382`, `/badge/8bitconcepts.com.svg=360`, `/llms.txt=337`, `/openapi.yaml=333`, `/api/v1/catalog=205`, `/api/v1/checkout=196`, `/api/v1/quote=196`, `/mcp-servers=83`, `/.well-known/mcp.json=82`.
- Score-fix intake: 11 total rows; aggregate-safe read shows `real_candidate pending=3`, `test_like pending=5`, `test_like lead=1`, `test_like paid=2`. Raw rows were not written.
- Scheduled monitor proof remains current: `tools/monitor-check.log` shows the 2026-05-11 07:30 PT run completed, processed 2 due monitors, quarantined one zero-score monitor, and kept one 100-score monitor stable.

## Duplicate and channel checks

- `ops/sweeper/marketer-inbox.jsonl` already covers score-results owner handoff, monitor quarantine handoff, generic directory submissions, API-key commerce handoff, check_url-to-monitor conversion, badge-traffic conversion, monitor proof repair, score-fix abandonment, query-intent briefing, and category-count copy drift.
- `outreach/distribution_log.csv` is saturated with MCP/API/GEO directory PRs, gists, newsletter pitches, Glama/PulseMCP/APIs.guru/mcpservers.org, RSS/content submissions, and existing NHS score-check action distribution.
- This run avoided duplicate generic directory/social rows.

## New findings

1. Anonymous REST search reads are blocked for this worker by the monthly quota. The live response includes API-key purchase metadata and the API-key catalog is now machine-readable, but the recurring marketer still lacks a private read path for bounded scout queries. That prevents producing fresh owner-channel target lists from `/api/v1/search` without using public quota.
2. MCP search intent has split into two concrete vertical clusters that are better than another broad agent-discovery post: finance/trading research and creative/video-production tooling. These should become narrow owner-channel briefs or landing-page briefs, not generic "NHS has search" copy.
3. Agent-commerce discovery traffic is now material: `/.well-known/commerce.json`, `/api/v1/catalog`, `/api/v1/quote`, and `/api/v1/checkout` collectively receive enough bot/agent traffic to justify a dedicated "agent-readable API key buying" proof brief once the private scout read path exists.

## Appended intake rows

Rows appended to `ops/sweeper/marketer-inbox.jsonl`:

- `Provision a private NHS API read path for recurring marketer scouts.`
- `Draft vertical query-intent briefs for finance/trading and creative AI tooling.`
