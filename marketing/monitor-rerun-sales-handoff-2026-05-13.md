# NHS Monitor Rerun Sales Handoff

Status: prepared, private handoff only.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-13T10:18Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, or QLimit/global-queue write was performed. This artifact is a private sales/owner-conversion handoff for a later admin or channel operator.

No raw monitor rows, emails, submitted domains tied to private rows, tokens, private notes, API keys, checkout URLs, or payment identifiers were written.

## Live Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4239`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories; largest public buckets are `developer=1300`, `ai-tools=892`, and `data=403`.
- `https://nothumansearch.ai/score`: HTTP 200; the score-result flow contains owner handoffs to `/monitor?domain=...` and `/fix/...`.
- `https://nothumansearch.ai/monitor`: HTTP 200; the page registers monitor requests through `/api/v1/monitor/register` and explains quarantine review when monitoring is not immediately active.
- `https://nothumansearch.ai/.well-known/mcp.json`: advertises `register_monitor` as an MCP-visible owner tool.
- `https://nothumansearch.ai/api/v1/catalog`: exposes score-fix and API subscription products; monitor remains a free owner retention path, not a paid ranking product.

Aggregate private/admin evidence checked through repo-local redacted helpers:

- Monitor status aggregate: `active=1`, `quarantined=1`, quarantine reason `first monitor check returned zero agentic score`.
- Monitor-admin action aggregate, last 30 days: `2026-05-13 request_score_rerun count=1`.
- Latest observed monitor worker completion in `tools/monitor-check.log`: `2026-05-11`; it processed 2 due monitors and quarantined one first-check zero-score row.

Aggregate MCP analytics, last 7 days:

- `tools/list=101664`, `initialize=13180`, `tools/call=416`.
- Tool calls: `search_agents=223`, `get_site_details=55`, `check_url=30`, `find_mcp_servers=29`, `get_stats=22`, `verify_mcp=21`, `get_top_sites=16`, `recent_additions=10`, `submit_site=6`, `list_categories=4`.
- Unknown tool names: none observed in this aggregate read.

## Sales Handoff

The free monitor surface is now part of the owner funnel, but one quarantined monitor still needs a private outcome loop. The recorded `request_score_rerun` action is a useful sales signal only after the private admin workflow confirms whether the owner got a useful next step.

Use this sequence:

1. Reconcile the quarantined monitor through the private admin workflow without copying raw row data into committed artifacts.
2. Record only aggregate action/status counts after the review.
3. If the rerun produces a valid score, activate monitoring or send the owner to the free monitor confirmation path.
4. If the rerun still shows missing hard agent signals, offer score-fix remediation as implementation help for public agent-readiness files.
5. If the row is junk or shared-host noise, keep it quarantined and do not turn it into a marketing proof point.

## Draft Copy For Later Private Follow-Up

Use only after the admin workflow confirms this is a real owner row and a follow-up is allowed:

`Your Not Human Search monitor request was queued for review because the first readiness check returned a zero-score result. The useful next step is to rerun the public score check and either activate monitoring if the site has a stable baseline, or use the score-fix path if the public agent-readiness files are missing. This does not buy ranking placement; it is implementation help for the same checks NHS applies to every indexed site.`

## Publication Guard

Do not publish this as a public social/directory post. Before any owner follow-up:

- Use only the private bearer-auth admin workflow.
- Keep raw emails, submitted domains tied to private rows, tokens, private notes, and payment identifiers out of committed artifacts.
- Rerun `tools/monitor-status-redacted-read.sh` and `tools/monitor-actions-redacted-read.sh`.
- Confirm the monitor action outcome has been reconciled, not just requested.
- Do not claim monitor demand, revenue, conversion, paid ranking placement, or score-methodology bypass.
