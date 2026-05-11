# NHS marketing scout segment - 2026-05-11T23:08Z

Automation: `business-marketer-not-human-search`

## Boundary

No outreach was sent. No posts, browser/Computer Use, account creation, deploys, full recrawls, product-code edits, or public submissions were performed.

## Live surface checks

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4238`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. Largest buckets: `developer=1300`, `ai-tools=892`, `other=775`, `data=403`, `finance=201`.
- `/.well-known/mcp.json`: 11 tools advertised.
- `GET /api/v1`: includes API-key plans, monitor registration, commerce catalog/quote/checkout, search, site, score-check, submit, top, and verify-MCP routes.
- `tools/mcp-registry/server.json`: published copy says `4,100+ sites` and `11 tools`; still consistent enough with live 4,238-site count for directory copy.
- `/api/v1/catalog`: includes score-fix plus `starter`, `pro`, and `scale` API subscription products.
- `/monitor`: live landing page returned 200, includes queued-for-admin-review copy, but does not link to `/fix/{host}` or `/score`.
- `/score`: live checker returned 200 and posts to `/api/v1/check`, but does not expose monitor signup or score-fix next steps in the page shell.
- `/fix/xquik.com`: live score-fix page returned 200, shows `$199`, and did not include paid-ranking or bypass language.
- `/badge/xquik.com.svg`: returned 200 `image/svg+xml`. The related `/site/xquik.com` page is live and score 100, with no monitor or fix CTA needed for that specific 100/100 page.

## Sanitized admin aggregates

- MCP analytics, last 7 days: `tools/list=79200`, `initialize=11022`, `tools/call=426`, unknown tool names: 0.
- MCP tool calls: `search_agents=230`, `get_site_details=53`, `check_url=29`, `find_mcp_servers=28`, `get_stats=26`, `verify_mcp=25`, `get_top_sites=17`, `recent_additions=7`, `submit_site=7`, `list_categories=4`.
- Traffic, last 14 days: `/=3797`, `/.well-known/commerce.json=888`, `/.well-known/ai-plugin.json=471`, `/badge/xquik.com.svg=410`, `/badge/aidevboard.com.svg=385`, `/robots.txt=368`, `/badge/8bitconcepts.com.svg=364`, `/llms.txt=329`, `/openapi.yaml=326`, `/api/v1/catalog=202`, `/api/v1/quote=193`, `/api/v1/checkout=193`.
- Score-fix intake: 11 total rows; statuses `pending=8`, `lead=1`, `paid=2`. Raw rows were not written.
- Monitor signups: 2 total; `active=1`, `quarantined=1`. Score buckets: one `0`, one `70_100`. Quarantine reason aggregate: `first monitor check returned zero agentic score`.
- Scheduled monitor proof is current: `tools/monitor-check.log` shows the 2026-05-11 07:30 PT run completed, processed 2 due monitors, quarantined one zero-score monitor, and kept one 100-score monitor stable.

## Duplicate and channel checks

- `outreach/distribution_log.csv` is already saturated with MCP, A2A, awesome-list, gist, email, IndexNow, Glama, PulseMCP, APIs.guru, and directory activity.
- Existing marketer inbox rows already cover generic directory submissions, API-key commerce handoff, check_url-to-monitor conversion, badge-traffic conversion, monitor proof repair, and score-fix abandonment.
- This run did not add another generic directory or social-post row.

## New findings

The current best marketing work is owner-conversion repair, not another channel blast.

1. The public `/score` page is where site owners self-identify, but its page shell has no visible next step to monitor the checked domain or buy score-fix remediation. A future product worker should turn a score result into the next owner action without changing scoring methodology.
2. The `/monitor` landing page can queue shared-host or zero-score domains for admin review, but the page shell does not offer a score-fix route or score-check route for owners who need implementation help instead of passive alerts.
3. The May 11 monitor worker proof removes the prior blocker against pushing monitor more broadly, but the quarantine state creates a new conversion handoff: when a domain cannot be safely auto-monitored, owners need an explainable path to review/remediation.

## Appended intake rows

Rows appended to `ops/sweeper/marketer-inbox.jsonl`:

- `Add a score-results owner handoff to monitor or score-fix remediation.`
- `Create a monitor quarantine review/remediation handoff.`
