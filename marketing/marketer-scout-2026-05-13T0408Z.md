# NHS marketing scout segment - 2026-05-13T04:08Z

Automation: `business-marketer-not-human-search`

## Boundary

No outreach was sent. No posts, browser/Computer Use, account creation, deploys, full recrawls, product-code edits, public submissions, or QLimit/global-queue writes were performed.

## Live surface checks

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4238`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. Largest buckets: `developer=1300`, `ai-tools=892`, `other=775`, `data=403`, `finance=201`.
- `/.well-known/mcp.json`: advertises 11 tools and now describes public categories plus audit-only `other`/`spam` correctly.
- `/llms.txt`: current site count and category language are aligned with live categories.
- `/monitor`: live form posts to `/api/v1/monitor/register` and tells quarantined shared-host registrations they are queued for review.
- `/fix/stripe.com`: low-score remediation page remains a paid score-fix intake, currently `45 -> target 95+`.
- `/fix/nothumansearch.ai`: high-score remediation page still shows a paid score-fix intake at `100 -> target 95+`; this is already queued from the prior scout run and was not duplicated.
- `/report`: OpenGraph/meta data still claims `10205` sites, avg `23.2`, and `219` sites scoring 70+ while `/api/v1/stats` says `4238` and avg `35`; this is already queued from the prior scout run and was not duplicated.

## Sanitized aggregate checks

- MCP analytics, last 7 days: `tools/list=98366`, `initialize=12368`, `tools/call=431`, unknown tool names: 0.
- MCP tool calls: `search_agents=234`, `get_site_details=56`, `check_url=30`, `find_mcp_servers=29`, `get_stats=24`, `verify_mcp=21`, `get_top_sites=16`, `recent_additions=10`, `submit_site=7`, `list_categories=4`.
- MCP top-query themes include news, exact-company/operator lookup, finance/trading research, video/filmmaking workflows, and API/browser automation lookup. No raw user identifiers were written.
- Traffic, last 14 days: `/=3686`, `/.well-known/commerce.json=1043`, `/.well-known/ai-plugin.json=550`, `/badge/xquik.com.svg=548`, `/llms.txt=376`, `/openapi.yaml=371`, `/api/v1/catalog=233`, `/api/v1/checkout=224`, `/api/v1/quote=224`, `/top=139`, `/newest=127`, `/.well-known/mcp.json=95`, `/api/v1=93`.
- Score-fix intake aggregate via `tools/geo-jobs-redacted-read.sh`: 11 total rows; `real_candidate pending=2`, both `dot_com` and `7_29d`; `test_like internal_test=1`, `lead=1`, `paid=2`, `pending=5`. Raw rows were not written.
- Monitor action aggregate via `tools/monitor-actions-redacted-read.sh`: `{"counts":[],"days":30}`.
- Scheduled monitor proof remains current from 2026-05-11 07:30 PT: two due monitors processed, one zero-score monitor quarantined, one 100-score monitor stable.

## Artifact produced

Created `marketing/finance-trading-owner-brief-2026-05-13.md`.

This converts the existing queued finance/trading scout row into a concrete channel-ready brief using current public category evidence and aggregate MCP query themes.

## Duplicate and channel checks

- `ops/sweeper/marketer-inbox.jsonl` already contains rows for stale `/report`, high-score `/fix` gating, finance/trading brief preparation, data/API brief preparation, vertical owner-channel briefs, monitor quarantine, score-fix abandonment, private API read path, and category-copy repair.
- `outreach/distribution_log.csv` is saturated with broad MCP/API/GEO directory PRs, gists, newsletter pitches, Glama/PulseMCP/APIs.guru/mcpservers.org, RSS/content submissions, and score-check action distribution.
- Shared social ledger includes broad NHS copy and company-profile NHS mentions, but no finance/trading vertical brief proof. Any external publication still needs active account verification, duplicate fingerprinting, and a sync-state public-action lock.

## Appended intake rows

Rows appended to `ops/sweeper/marketer-inbox.jsonl`:

- `Publish the prepared finance/trading agent-readiness brief through a gated channel operator.`
