# NHS marketing scout segment - 2026-05-13T16:08Z

Automation: `business-marketer-not-human-search`

## Boundary

No outreach was sent. No posts, browser/Computer Use, account creation, deploys, full recrawls, product-code edits, public submissions, or QLimit/global-queue writes were performed.

## Live surface checks

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4181`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories; `developer=1233`, average score `34`.
- `https://nothumansearch.ai/api/v1/top?category=developer&limit=8`: public developer examples at the top score 100/100 and expose all seven signals.
- `https://nothumansearch.ai/llms.txt`: developer is listed as a public category; `other` and `spam` are audit-only.
- `https://nothumansearch.ai/.well-known/mcp.json`: category parameter copy matches the public/audit-only split.

No raw user identifiers, private customer rows, API keys, checkout URLs, or private query logs were written.

## Artifact produced

Created `marketing/developer-tools-owner-brief-2026-05-13.md`.

This converts the queued developer-tools owner-channel prep item into a durable, channel-ready brief for a later gated operator.

## Appended intake rows

Rows appended to `ops/sweeper/marketer-inbox.jsonl`:

- `Publish the prepared developer-tools agent-readiness brief through a gated channel operator.`
