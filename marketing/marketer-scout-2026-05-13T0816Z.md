# NHS marketing scout segment - 2026-05-13T08:16Z

Automation: `business-marketer-not-human-search`

## Boundary

No outreach was sent. No posts, browser/Computer Use, account creation, deploys, full recrawls, product-code edits, public submissions, or QLimit/global-queue writes were performed.

## Live surface checks

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4239`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories; `ecommerce=149`, average score `41`.
- `https://nothumansearch.ai/api/v1/top?category=ecommerce&limit=8`: public ecommerce examples range from 100/100 complete agent-readiness to 75/100 partial surfaces.
- `https://nothumansearch.ai/llms.txt`: ecommerce is listed as a public category; `other` and `spam` are audit-only.
- `https://nothumansearch.ai/.well-known/mcp.json`: category parameter copy matches the public/audit-only split.

No raw user identifiers, private customer rows, API keys, checkout URLs, or private query logs were written.

## Artifact produced

Created `marketing/ecommerce-owner-brief-2026-05-13.md`.

This converts the queued ecommerce owner-channel prep item into a durable, channel-ready brief for a later gated operator.

## Appended intake rows

Rows appended to `ops/sweeper/marketer-inbox.jsonl`:

- `Publish the prepared ecommerce agent-readiness brief through a gated channel operator.`
