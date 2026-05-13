# NHS marketing scout segment - 2026-05-13T07:08Z

Automation: `business-marketer-not-human-search`

## Boundary

No outreach was sent. No posts, browser/Computer Use, account creation, deploys, full recrawls, product-code edits, public submissions, or QLimit/global-queue writes were performed.

## Live surface checks

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4239`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. Largest public buckets: `developer=1300`, `ai-tools=892`, `data=403`, `finance=201`, `productivity=171`.
- `https://nothumansearch.ai/api/v1/search?q=mcp&limit=1`: HTTP 402 `quota_exceeded`, `used=100`, `limit=100`, with a structured subscription handoff.
- `https://nothumansearch.ai/api/v1/api-keys/subscribe`: HTTP 200 plan metadata.
- `https://nothumansearch.ai/api/v1/catalog`: agent-readable API subscription products present.
- `POST /api/v1/api-keys/subscribe` with a Foundry-owned smoke email and `plan=starter`: HTTP 200 with `plan=starter`, `monthly_limit=1000`, and `amount_cents=1900`; checkout and activation URLs were redacted from notes.

## Sanitized aggregate checks

- MCP analytics, last 7 days: `tools/list=99917`, `initialize=12763`, `tools/call=430`, unknown tool names: 0.
- MCP tool calls: `search_agents=233`, `get_site_details=56`, `check_url=31`, `find_mcp_servers=29`, `get_stats=23`, `verify_mcp=21`, `get_top_sites=16`, `recent_additions=10`, `submit_site=7`, `list_categories=4`.
- No raw user identifiers, private customer rows, API keys, checkout session ids, or raw checkout URLs were written.

## Artifact produced

Created `marketing/api-quota-conversion-brief-2026-05-13.md`.

This converts the previously observed quota-exceeded/API-plan path into a concrete sales/channel brief using current public plan metadata and a redacted POST smoke.

## Appended intake rows

Rows appended to `ops/sweeper/marketer-inbox.jsonl`:

- `Publish the prepared NHS API quota-to-paid-plan brief through a gated channel operator.`
