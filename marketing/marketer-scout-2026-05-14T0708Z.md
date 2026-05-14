# NHS marketing scout segment - 2026-05-14T07:08Z

Automation: `business-marketer-not-human-search`

## Boundary

No outreach was sent. No posts, browser/Computer Use, account creation, deploys, full recrawls, product-code edits, public submissions, or QLimit/global-queue writes were performed.

## Live surface checks

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4170`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories; `jobs=26`, average score `41`.
- `https://nothumansearch.ai/api/v1/top?category=jobs&limit=8`: public jobs examples range from a Foundry-owned 100/100 dogfood reference to third-party job and recruiting surfaces scoring 75, 65, 50, and 45.
- `https://nothumansearch.ai/api/v1/catalog`: score-fix and API subscription products present; no checkout was started.
- `https://nothumansearch.ai/llms.txt`: jobs is listed as a public category; `other` and `spam` are audit-only.
- `https://nothumansearch.ai/.well-known/mcp.json`: category parameter copy matches the public/audit-only split.
- `https://nothumansearch.ai/monitor`: live monitor form returned HTTP 200 and still frames shared-host quarantine review.

## Sanitized aggregate checks

- Score-fix aggregate via `tools/geo-jobs-redacted-read.sh`: 11 total rows; real-candidate pending remains `2`, both `dot_com` and `7_29d`; no real paid or real lead row was exposed.
- Monitor aggregate via `tools/monitor-status-redacted-read.sh`: `active=1`, `quarantined=1`, quarantine reason `bounded rerun still zero score`.
- Monitor actions via `tools/monitor-actions-redacted-read.sh`: `request_score_rerun=1` and `keep_quarantined=1` on 2026-05-13.
- Shared social ledger grep found broad Not Human Search rows and AI Dev Board jobs posts, but no prepared jobs-platform NHS category brief publication proof.
- `outreach/distribution_log.csv` remains saturated with broad MCP/API/GEO directory PRs, gists, newsletter pitches, Glama/PulseMCP/APIs.guru/mcpservers.org, RSS/content submissions, and score-check action distribution.

No raw user identifiers, private customer rows, API keys, checkout URLs, or private query logs were written.

## Artifact produced

Created `marketing/jobs-platform-owner-brief-2026-05-14.md`.

This converts the queued jobs-platform owner-channel prep item into a durable, channel-ready brief for a later gated operator. It explicitly labels AI Dev Board as Foundry-owned dogfood rather than third-party market proof.

## Appended intake rows

Rows appended to `ops/sweeper/marketer-inbox.jsonl`:

- `Publish the prepared jobs-platform agent-readiness brief through a gated channel operator.`
