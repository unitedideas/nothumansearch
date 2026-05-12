# NHS marketing scout segment - 2026-05-12T23:08Z

Automation: `business-marketer-not-human-search`

## Boundary

No outreach was sent. No posts, browser/Computer Use, account creation, deploys, full recrawls, product-code edits, public submissions, or QLimit/global-queue writes were performed.

## Live Surface Checks

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4238`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. Public categories include `ai-tools`, `developer`, `data`, `finance`, `ecommerce`, `jobs`, `security`, `health`, `education`, `communication`, `productivity`, and `news`; audit-only buckets are `other` and `spam`.
- `/llms.txt`: advertises `4238+` indexed sites, 11 MCP tools, current API-key plan copy, and the public/audit-only category split.
- `/.well-known/mcp.json`: advertises 11 tools and current category language.
- JSON-RPC `tools/list` at `/mcp`: returns 11 tools matching the public discovery surface.
- `/api/v1/catalog`, `/.well-known/commerce.json`, and `/.well-known/agent.json`: list score-fix plus Starter, Pro, and Scale API subscriptions.
- `GET /api/v1/api-keys/subscribe`: returns HTTP 200 JSON with API plan metadata and the POST contract.
- `/install`: returns HTTP 200 shell installer and advertises the same 11 tools.
- `/guide`, `/score`, and `/monitor`: return HTTP 200.

## Fresh Drift Found

`/report` returns HTTP 200, but its page metadata is stale:

- Meta description says `10205 indexed domains`.
- OpenGraph description says `10205 sites`, average score `23.2/100`, and `219 sites score 70+`.
- Current public stats are `4238` sites and average score `35`.

This is a marketing-surface drift issue. The report is a shareable/public proof page, so stale counts can contaminate future social, directory, and owner-channel copy even though `/api/v1/stats` and `/llms.txt` are current.

Official MCP registry copy is also now behind the public site count:

- In-repo `tools/mcp-registry/server.json`: `4,100+ sites`, version `1.7.1`.
- Live registry response for `ai.nothumansearch/search`: same `4,100+ sites`, version `1.7.1`.
- Current public count remains `4,238+`; this is not urgent enough to publish by itself unless a registry update is already being made, but it should be refreshed before the next publish.

## Duplicate And Channel Checks

- `ops/sweeper/marketer-inbox.jsonl` already covers public posting draft refresh, category-copy repair, API-key handoff repair, vertical owner briefs, monitor conversion, score-fix triage, and private API read path.
- The stale `/report` metadata issue was not already present as a dedicated marketer inbox row.
- Shared social ledger grep found existing broad NHS posts and queue items; no public action was taken, so no public-action lock was claimed.
- `outreach/distribution_log.csv` remains saturated with broad MCP/API/GEO directory submissions. This run did not queue another directory packet.

## Appended Intake Rows

Rows appended to `ops/sweeper/marketer-inbox.jsonl`:

- `Repair stale /report metadata before using it as a shareable proof page.`
- `Refresh official MCP registry copy before the next NHS registry publish.`
