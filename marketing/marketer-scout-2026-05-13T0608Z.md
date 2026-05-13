# NHS marketing scout segment - 2026-05-13T06:08Z

Automation: `business-marketer-not-human-search`

## Boundary

No outreach was sent. No posts, browser/Computer Use, account creation, deploys, full recrawls, product-code edits, public submissions, or QLimit/global-queue writes were performed.

## Live surface checks

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4238`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. `news=11`, `avg_score=50`.
- `https://nothumansearch.ai/api/v1/top?category=news&limit=8`: public top news/media examples only.
- `/.well-known/mcp.json` and `/llms.txt` were recently aligned per prior scout artifacts; this run did not detect a new discovery-copy blocker.
- Existing high-priority queued rows already cover stale `/report` metadata and high-score `/fix` gating, so neither was duplicated.

## Sanitized aggregate checks

- MCP analytics, last 7 days: `tools/list=99206`, `initialize=12563`, `tools/call=431`, unknown tool names: 0.
- MCP tool calls: `search_agents=235`, `get_site_details=56`, `check_url=30`, `find_mcp_servers=29`, `get_stats=23`, `verify_mcp=21`, `get_top_sites=16`, `recent_additions=10`, `submit_site=7`, `list_categories=4`.
- MCP top-query themes include news, exact-company/operator lookup, finance/trading research, creative/video-production workflows, and API/browser automation lookup. No raw user identifiers were written.
- Shared social ledger grep found broad NHS queue items and published Foundry posts, but no exact news/media owner-channel brief proof.
- `outreach/distribution_log.csv` remains saturated with broad MCP/API/GEO directory PRs, gists, newsletter pitches, Glama/PulseMCP/APIs.guru/mcpservers.org, RSS/content submissions, and score-check action distribution.

## Artifact produced

Created `marketing/news-media-owner-brief-2026-05-13.md`.

This converts the existing queued news/media scout row into a concrete channel-ready brief using current public category evidence and aggregate MCP query themes.

## Appended intake rows

Rows appended to `ops/sweeper/marketer-inbox.jsonl`:

- `Publish the prepared news/media agent-readiness brief through a gated channel operator.`
