# MCP top and recent result conversion test

Date: 2026-07-30
Automation: `business-marketer-not-human-search`
Status: no outreach, public post, account action, browser action, deploy, crawl, checkout, or QLimit/global-queue write performed

## Segment

Turn the two most-used result-list MCP tools into proof-to-monitor and
score-fix handoffs. `get_top_sites` and `recent_additions` currently return the
origin URL and score, but not the NHS report, free monitor, or score-band-aware
remediation path.

This is a product-handoff conversion test. It is not an owner outreach or
public-post brief.

## Fresh evidence

- Public stats report 4,351 indexed sites, average score 38, and `developer` as
  the top category.
- Seven-day aggregate MCP analytics recorded 48,850 `tools/list`, 14,432
  `initialize`, and 236 `tools/call` requests.
- The leading real tool calls were `get_top_sites=94`, `get_stats=49`,
  `recent_additions=30`, `list_categories=21`, `submit_site=17`, and
  `register_monitor=15`. Top and recent discovery accounted for 124 of 236
  tool-call requests. Treat this as conversion-design evidence, not demand or
  owner intent.
- Live JSON-RPC `tools/list` returned the expected 11 tools.
- A bounded live `get_top_sites` call returned `content` plus
  `structuredContent.results`. Each result had its origin URL, score, category,
  signals, and timestamps, but no `report_url`, `monitor_url`, or `fix_url`.
  The text response also contained no report or monitor next step.
- A bounded live `recent_additions` call had the same gap: origin URL and score,
  but no NHS report, monitor, or remediation link in text or structured output.
- The public score-band handoff already exists elsewhere. The 100/100 NHS
  profile and `/fix/nothumansearch.ai` route point to the domain-prefilled free
  monitor. The 65/100 Manifestly profile retains a legitimate score-fix route,
  and `/monitor?domain=manifest.ly` is also available.
- The monitor has five active and three quarantined registrations. The latest
  worker evidence completed five due checks on 2026-07-27.
- The redacted score-fix aggregate contains ten pending real-candidate rows and
  no real-candidate paid or lead rows. Do not frame the result-list traffic as
  sales or revenue proof.
- `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/llms.txt`,
  `/openapi.yaml`, `/api/v1`, `/api/v1/catalog`,
  `/api/v1/api-keys/subscribe`, `/score`, `/monitor`, `/report`, `/newest`, and
  `/top` returned HTTP 200. `/.well-known/agent-card.json` returned 404, so A2A
  positioning remains blocked.

No raw MCP queries, emails, private monitor rows, private score-fix rows,
checkout URLs, payment identifiers, API keys, or customer identifiers were
written to this artifact.

## Conversion hypothesis

Agents using `get_top_sites` or `recent_additions` should receive one safe NHS
next step with every result:

1. Always include the public NHS `report_url`.
2. Include the domain-prefilled `monitor_url` for free drift monitoring.
3. Include a score-fix URL only when the same current eligibility rule used by
   `/fix/{host}` says remediation is appropriate.
4. For sites already meeting the remediation target, describe the monitor as
   the next action and do not present paid remediation.
5. For partial-score sites, keep `/score` and the public report ahead of paid
   remediation so a stale or non-root capability is not sold as missing.

Use the existing server-side score-fix eligibility decision rather than
duplicating a numeric threshold in MCP response code.

## Acceptance test

1. `get_top_sites` and `recent_additions` keep their existing text and
   structured fields while adding `report_url` and `monitor_url` for every
   result.
2. Each text result includes a concise NHS report line and one score-band-aware
   next action.
3. A high-score result routes to the prefilled free monitor and does not expose
   the paid remediation offer.
4. A representative partial-score result exposes its report and legitimate
   score-fix path only after current public evidence confirms the gap.
5. Spam, quarantined, unavailable-origin, stale-profile, canonical-host, and
   declared-non-root-path guards remain intact.
6. MCP unit tests cover high-score and partial-score result contracts for both
   tools. After a later product-worker deploy, live `tools/list`, both tool
   calls, the report, monitor, high- and partial-score fix routes, and the
   latest monitor-worker evidence pass.
7. Analytics distinguish tool result delivery, report clicks, monitor-form
   visits, monitor registrations, and paid conversions. None of the earlier
   funnel stages may be described as a customer, consent, demand, or revenue.

## Duplicate and claim boundary

The portfolio social ledger, NHS distribution history, and marketer inbox had
no exact MCP `get_top_sites`/`recent_additions` result-to-monitor contract. An
older organic CTA row covers the human `/top` and `/newest` pages; the current
test is limited to the MCP response contract and does not duplicate a public
post or owner touch.

Do not claim listed domains are customers, partners, endorsements, paid leads,
monitor registrations, badge-install consent, private demand, completed
payments, or revenue. Do not claim API or data quality, freshness, uptime,
security certification, paid ranking, preferred inclusion, A2A support while
the Agent Card route is 404, x402/ACP/SPT/MPP support for NHS, or a
score-methodology bypass.
