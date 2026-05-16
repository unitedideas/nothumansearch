# Other-Category Owner Segmentation Brief

Date: 2026-05-16
Automation: `business-marketer-not-human-search`
Status: prepared, not published

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a sanitized scout artifact for a later gated operator or product owner-conversion test.

## Fresh Evidence

Public surfaces checked:

- `https://nothumansearch.ai/api/v1/stats`: 4,175 indexed sites, average score 35, top category `developer`.
- `https://nothumansearch.ai/api/v1/categories`: `other=770` with average score 27; larger buckets include `developer=1228`, `ai-tools=902`, `data=403`, `finance=200`, `productivity=172`, `ecommerce=151`, `communication=117`, `security=115`, and `health=57`.
- `https://nothumansearch.ai/.well-known/mcp.json`: HTTP 200 and advertises 11 tools: `search_agents`, `get_site_details`, `submit_site`, `get_stats`, `register_monitor`, `verify_mcp`, `check_url`, `list_categories`, `get_top_sites`, `find_mcp_servers`, and `recent_additions`.
- `https://nothumansearch.ai/.well-known/agent.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, `/monitor`, `/score`, `/top`, and `/newest`: HTTP 200.
- `https://nothumansearch.ai/.well-known/agent-card.json`: HTTP 404, so strict Agent Card directory submissions remain gated.

Aggregate admin signals, sanitized:

- MCP analytics, last 7 days: `tools/list=135762`, `initialize=18617`, `tools/call=292`.
- Top called MCP tools: `search_agents=189`, `get_site_details=37`, `verify_mcp=14`, `check_url=13`, `get_stats=13`, `recent_additions=8`, `find_mcp_servers=8`, `get_top_sites=7`.
- Traffic, last 168 hours: `/.well-known/commerce.json=1516`, `/api/v1/catalog=343`, `/api/v1/checkout=315`, `/api/v1/quote=315`, `/site/xquik.com=260`, `/top=130`, `/newest=94`, `/.well-known/mcp.json=86`, `/api/v1=83`.
- Score-fix aggregate read: 2 real-candidate pending rows in the 7-29 day bucket; test-like/internal rows are excluded from marketing proof.
- Monitor admin actions, last 30 days: `request_score_rerun=1`, `keep_quarantined=1`.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, or private query logs are included here.

## Public Other-Category Examples

The public `other` top list is useful, but too mixed for a broad marketing claim. Treat these as segmentation candidates or owner-conversion examples, not customers, endorsements, paid leads, private demand, or proof of market share.

- `astranl.com` - score 100; full public signal set including llms.txt, ai-plugin, OpenAPI, structured API, MCP, AI-friendly robots, and Schema.org.
- `lobehub.com` - score 95; agent/productivity platform that appears under `other` despite strong MCP/API signals.
- `surprise-buddy.com` - score 65; consumer/gifting surface with llms.txt, OpenAPI, MCP, AI-friendly robots, and Schema.org.
- `infinity-folder.org` - score 65; public OpenAPI/structured API/llms surface without MCP.
- `crabbitmq.com` - score 50; agent queueing surface with llms.txt, structured API, and MCP.

## Segment Read

The `other` bucket now contains both legitimate high-score agent-first services and mixed consumer/business sites. That makes it a poor public positioning bucket, but a useful product and owner-channel scout signal:

- High-score `other` sites can be routed toward free monitor registration, badge/report proof, and category correction requests.
- Mid-score `other` sites can be routed to `/score` first and then to score-fix only when missing public agent-readiness signals justify remediation.
- Agent-builder or developer-tool examples should be separated from consumer/local-service examples before publication.
- Taxonomy cleanup should happen before using `other` as market proof; otherwise the copy will look like a generic website directory rather than an agent-readiness engine.

## Draft Operator Copy

`Not Human Search has a visible "other" bucket now, but the useful signal is not the bucket label. It is the score split: some sites already expose complete machine-readable surfaces, while others have partial llms.txt/OpenAPI/API/MCP coverage that agents can inspect but owners probably are not monitoring.`

`For high-score owners, the useful ask is monitoring and badge/report proof. For partial-score owners, the score page shows the missing public agent-readiness signals before any remediation offer.`

Proof links:

- `https://nothumansearch.ai/api/v1/top?category=other&limit=8`
- `https://nothumansearch.ai/score`
- `https://nothumansearch.ai/monitor`
- `https://nothumansearch.ai/top`
- `https://nothumansearch.ai/newest`

## Operator Use

Good next actions:

1. Pick exactly one sub-segment from the `other` list before external use: agent-builder tools, API/developer services, consumer/local-service owners, or commerce-like owners.
2. Refresh `/api/v1/top?category=other&limit=8` and the representative public profile pages before publication.
3. Use high-score owners for monitor/report/badge proof; use partial-score owners for `/score` first, then score-fix only if missing public signals justify remediation.
4. Queue taxonomy cleanup separately if the selected examples are obviously misbucketed.

## Publication Guard

Before any external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=other&limit=8`, `/score`, `/monitor`, representative `/site/{host}` profiles, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, and aggregate `/api/v1/admin/mcp?days=7`.
- Verify active account identity for the selected Foundry/Owl-owned channel.
- Check `marketing/social-post-ledger.json` if it exists, sync-state public-action locks, and `outreach/distribution_log.csv`.
- Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.
- Do not claim listed domains are customers, endorsements, paid leads, private demand, completed payments, revenue, category endorsement, data freshness, seller certification, A2A support, x402/ACP/MPP support for NHS, paid ranking placement, preferred inclusion, or score-methodology bypass.

## Blockers

- `tools/full-recrawl.lock` exists in the worktree; no runtime mutation, deploy, or broad crawl should be attempted from this scout run.
- `/.well-known/agent-card.json` returns 404; strict Agent Card directory submissions remain gated.
- The `other` category is too mixed for broad market copy without sub-segmentation.
