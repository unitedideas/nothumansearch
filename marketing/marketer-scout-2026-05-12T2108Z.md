# NHS marketing scout segment - 2026-05-12T21:08Z

Automation: `business-marketer-not-human-search`

## Boundary

No outreach was sent. No posts, browser/Computer Use, account creation, deploys, full recrawls, product-code edits, public submissions, or QLimit/global-queue writes were performed.

## Live Surface Checks

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4238`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. Public categories include `ai-tools`, `developer`, `data`, `finance`, `ecommerce`, `jobs`, `security`, `health`, `education`, `communication`, `productivity`, and `news`; audit-only buckets are `other` and `spam`.
- `/.well-known/mcp.json`: advertises 11 tools and the public/audit-only category split.
- `/llms.txt`: advertises 4238+ indexed sites, 11 MCP tools, API-key plans, and the public/audit-only category split.
- `/api/v1/catalog`: lists score-fix plus Starter, Pro, and Scale API subscriptions.
- JSON-RPC `tools/list` at `/mcp`: returns 11 tools matching the public discovery surface.
- `tools/mcp-registry/server.json`: still describes `4,100+ sites` in registry copy; refresh before the next registry publish if current public count remains 4,238+.

## Distribution And Duplicate Checks

- `outreach/distribution_log.csv` is saturated with broad MCP/API/GEO directory PRs, gists, newsletter pitches, Glama/PulseMCP/APIs.guru/mcpservers.org, RSS/content submissions, and NHS score-check action distribution.
- Shared social ledger grep found existing broad NHS posts and queue items, but no exact `ai-tools` or `news/media owner-channel brief` rows for this scout packet.
- Prior same-day marketer rows already cover ecommerce, productivity, security, data, developer, communication, health, education, and jobs. This run used the remaining non-audit public categories.

## Public Category Evidence

These are public `/api/v1/top` results, not private admin/customer rows.

AI-tools owner-channel shortlist from `GET /api/v1/top?category=ai-tools&limit=8`:

- `8bitconcepts.com` - score 100, Foundry-owned consulting surface with all 7 signals.
- `nothumansearch.ai` - score 100, Foundry-owned agentic-readiness search with all 7 signals.
- `bringyour.ai` - score 100, Foundry-owned harness migration product with all 7 signals.
- `chainray.online` - score 100, on-chain intelligence for agents with all 7 signals.
- `sincetmw.ai` - score 100, culture-trend search surface with all 7 signals.
- `claudereviews.com` - score 100, AI reviews/data analysis/investigation surface with all 7 signals.
- `teenanxiety.ai` - score 100, health-content surface currently categorized as `ai-tools`.
- `teenadhd.ai` - score 100, health-content surface currently categorized as `ai-tools`.

News/media owner-channel shortlist from `GET /api/v1/top?category=news&limit=8`:

- `informedclearly.com` - score 70, news summaries/analysis missing AI-friendly robots, structured API, and MCP.
- `hallucinationherald.com` - score 65, autonomous AI newspaper missing OpenAPI and structured API.
- `biztoc.com` - score 65, business/finance news hub missing llms.txt and MCP.
- `zadar.tv` - score 55, regional news/media surface missing ai-plugin, structured API, and MCP.
- `aibtc.news` - score 50, Bitcoin/agent news surface missing ai-plugin, OpenAPI, and MCP.
- `thesansasyonel.com` - score 45, entertainment/news surface missing AI-friendly robots, ai-plugin, OpenAPI, and MCP.
- `sansasyonelgazete.com` - score 45, entertainment/news surface missing AI-friendly robots, ai-plugin, OpenAPI, and MCP.
- `yubigeek.com` - score 45, gaming/streaming/geek media surface missing AI-friendly robots, ai-plugin, OpenAPI, and MCP.

## Draft Brief Angles

AI-native tools:

`The AI-tools category is large enough to segment, but it should not be pushed as a generic top-list without cleanup. The public top eight include three Foundry-owned properties and two teen-health surfaces categorized as AI tools. The useful owner-channel brief should focus on third-party AI-native tools with concrete missing signals, and it should either exclude Foundry-owned examples or label them as dogfood references.`

News/media:

`News is small but useful because publisher surfaces increasingly want agent visibility without giving agents a brittle browser-only path. NHS has 11 news/media sites; top examples already expose some combination of llms.txt, ai-plugin, OpenAPI, structured API, MCP, robots AI rules, and schema. The owner-channel angle is not "AI news." It is whether agents can cite, inspect, subscribe to, or verify publisher content through machine-readable surfaces.`

## Appended Intake Rows

Rows appended to `ops/sweeper/marketer-inbox.jsonl`:

- `Prepare AI-native tools owner-channel brief with Foundry-owned examples excluded or labeled.`
- `Prepare news/media owner-channel brief from public top-category evidence.`
