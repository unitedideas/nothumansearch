# NHS marketing scout segment - 2026-05-12T02:09Z

Automation: `business-marketer-not-human-search`

## Boundary

No outreach was sent. No posts, browser/Computer Use, account creation, deploys, full recrawls, product-code edits, or public submissions were performed.

## Live surface checks

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4238`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. Largest buckets: `developer=1300`, `ai-tools=892`, `other=775`, `data=403`, `finance=201`.
- `/.well-known/mcp.json`: 11 tools advertised and aligned with `llms.txt`.
- `GET /api/v1`: lists API-key plans, monitor registration, commerce catalog/quote/checkout, search, site, score-check, submit, top, and verify-MCP routes.
- `/api/v1/catalog`: includes score-fix plus `starter`, `pro`, and `scale` API subscription products.
- `GET /api/v1/api-keys/subscribe`: returns a readable API-key plan contract with `starter`, `pro`, and `scale`.
- `/.well-known/agent.json` and `/.well-known/commerce.json`: advertise paid API keys, score-fix, Stripe Checkout/Link, and unsupported ACP/x402 modes explicitly.
- `/top`, `/newest`, and `/mcp-servers`: live pages include score/monitor/fix vocabulary and links, so this run did not queue another generic owner-CTA row for those pages.

## Sanitized admin aggregates

- MCP analytics, last 7 days: `tools/list=81001`, `initialize=11062`, `tools/call=438`, unknown tool names: 0.
- MCP tool calls: `search_agents=236`, `get_site_details=57`, `check_url=29`, `find_mcp_servers=28`, `get_stats=26`, `verify_mcp=26`, `get_top_sites=17`, `recent_additions=8`, `submit_site=7`, `list_categories=4`.
- MCP top-query themes include exact-brand and vertical-commercial intent: `AFFiNE MCP server`, `Gibson Energy CEO`, `Amadeus API booking hotel`, `NSE India index trading algorithm strategy`, `video generation AI filmmaking`, `screenplay script writing AI tool`, `intraday trading strategy algorithm backtest ORB VWAP momentum mean reversion profit`.
- Traffic, last 14 days: `/=3803`, `/.well-known/commerce.json=898`, `/.well-known/ai-plugin.json=478`, `/badge/xquik.com.svg=435`, `/badge/aidevboard.com.svg=382`, `/robots.txt=373`, `/badge/8bitconcepts.com.svg=359`, `/llms.txt=334`, `/openapi.yaml=331`, `/api/v1/catalog=204`, `/api/v1/quote=195`, `/api/v1/checkout=195`, `/top=149`, `/newest=133`.
- Score-fix intake: 11 total rows; aggregate-safe read shows `real_candidate pending=3`, `test_like pending=5`, `test_like lead=1`, `test_like paid=2`. Raw rows were not written.
- Scheduled monitor proof remains current: `tools/monitor-check.log` shows the 2026-05-11 07:30 PT run completed, processed 2 due monitors, quarantined one zero-score monitor, and kept one 100-score monitor stable.

## Duplicate and channel checks

- `ops/sweeper/marketer-inbox.jsonl` already has rows for score-results owner handoff, monitor quarantine handoff, generic directory submissions, API-key commerce handoff, check_url-to-monitor conversion, badge-traffic conversion, monitor proof repair, and score-fix abandonment.
- `outreach/distribution_log.csv` is saturated with MCP/API/GEO directory PRs, gists, newsletter pitches, Glama/PulseMCP/APIs.guru/mcpservers.org, and RSS/content submissions.
- This run avoided duplicate generic directory/social rows.

## New findings

1. MCP `top_queries` now gives better marketing scout inputs than broad directory discovery. The useful next artifact is a query-intent brief for vertical pages or founder/operator outreach, using exact observed search themes but no raw user identifiers.
2. Public discovery-surface copy still has small category drift: `llms.txt` says `/categories` returns "all 12 buckets" and lists 11 named categories, while the live API returns 14 including `news`, `spam`, and `other`. This is low-risk but public-agent-facing copy drift.
3. API-key commerce handoff is now live enough for agent-commerce copy: the plan page and catalog agree on starter/pro/scale, and the prior API-key catalog gap is no longer current.

## Appended intake rows

Rows appended to `ops/sweeper/marketer-inbox.jsonl`:

- `Create a query-intent marketing brief from NHS MCP top searches.`
- `Repair public category-count copy drift in agent discovery surfaces.`
