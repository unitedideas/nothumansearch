# MCP site-details owner-action handoff

Date: 2026-07-31
Automation: `business-marketer-not-human-search`
Status: private product handoff only; no outreach, public post, account action,
browser action, product-code edit, deploy, crawl, monitor registration, checkout,
or global-queue write performed

## Segment

Turn `get_site_details`, the domain-specific MCP report tool, into a safe
proof-to-action response. The current text includes the public NHS profile, but
its `structuredContent` is the raw site record and does not provide explicit
NHS report, monitor, or score-fix routing fields.

This is narrower than the existing search-result and top/recent result
handoffs. It covers the highest-intent detail response after an agent has
already selected a domain.

## Fresh evidence

- Public stats report 4,349 indexed sites, average score 38, and `developer` as
  the top category.
- Seven-day aggregate MCP analytics recorded 49,520 `tools/list`, 14,269
  `initialize`, and 235 `tools/call` requests. The current top tool-call table
  is dominated by overview and list tools; this handoff does not infer demand
  or owner intent from those counts.
- Live JSON-RPC `tools/list` exposes 11 tools. `get_site_details` promises a
  full seven-signal report for a selected domain.
- Current handler inspection shows the text response ends with
  `Full report: /site/{domain}`, while `structuredContent` is the site model
  itself. It does not add explicit `profile_url`, `report_url`, `monitor_url`,
  `fix_url`, eligibility, or recommended-action fields.
- An anonymous bounded live `get_site_details` call hit the current subscription
  boundary, so this run did not claim a deployed result shape from that call.
  The response-contract evidence above comes from the current server source and
  the live `tools/list` schema.
- Aggregate monitor status is five active and three quarantined registrations.
  The latest monitor worker proof completed five due checks on 2026-07-27.
- The redacted score-fix aggregate contains ten pending real-candidate rows and
  no real-candidate paid or lead rows. Do not frame a detail response or link as
  a sale or owner lead.
- `/health`, `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/llms.txt`,
  `/openapi.yaml`, `/api/v1`, `/api/v1/catalog`, `/score`, `/monitor`, and
  `/report` returned HTTP 200. `/.well-known/agent-card.json` returned 404, so
  A2A positioning remains blocked.
- A full recrawl that started at 06:00 PT is still active. This scout did not
  deploy, recrawl, or compete with it.

No raw MCP queries, emails, private monitor rows, private score-fix rows,
checkout URLs, payment identifiers, API keys, IP data, or customer identifiers
were written to this artifact.

## Conversion hypothesis

After an agent selects a domain, `get_site_details` should return one explicit,
machine-actionable next step derived from the same server-side decision used by
the public profile and score-fix route:

1. Always expose the public NHS profile/report URL.
2. For a high-score, stable, available domain, expose the domain-prefilled free
   monitor URL and describe monitoring or report/badge proof as the next step.
3. For a partial-score domain, keep the report and a fresh score check ahead of
   paid remediation; expose a fix URL only when the current score-fix
   eligibility decision confirms a legitimate gap.
4. Suppress remediation and monitor acquisition for spam, quarantined,
   unavailable-origin, stale-profile, temporary-tunnel, canonical-host, and
   intentionally unavailable capability states.
5. Do not suggest `register_monitor` until the separate consent and email
   telemetry handoff is complete. A prefilled `/monitor?domain=` link is the
   safe interim action because it does not register anyone automatically.

## Acceptance test

1. Preserve every existing text and structured site field while adding stable
   `profile_url`, `report_url`, and `recommended_action` fields.
2. Add a domain-prefilled `monitor_url` only for eligible stable profiles. Do
   not auto-register a monitor or require an email in this response.
3. Add `fix_url` only when the existing server-side score-fix eligibility
   decision confirms remediation is appropriate; do not duplicate a numeric
   threshold in MCP code.
4. Keep high-score sites on monitor/report/badge proof and partial-score sites
   on report plus fresh score verification before remediation.
5. Preserve the existing unavailable-origin, stale-profile, temporary-tunnel,
   canonical-host, declared-non-root-path, spam, and quarantine guards.
6. Add MCP unit coverage for high-score, legitimate partial-score, unavailable,
   stale, canonical-alias, declared-non-root-path, spam, and quarantine result
   contracts, including text and `structuredContent` compatibility.
7. After the active recrawl is complete and a later product worker deploys the
   change, verify `tools/list`, authenticated `get_site_details` for one high-
   and one partial-score fixture, the report, prefilled monitor, high-score and
   partial-score fix routes, the consent-safe monitor boundary, and latest
   monitor-worker proof.
8. Track detail delivery, report visits, monitor-form visits, confirmed monitor
   activations, and paid conversions separately. None of the earlier funnel
   stages is a customer, consent signal, demand signal, or revenue.

## Duplicate and claim boundary

The May search-result artifact covers search-result intent routing, and the
July top/recent artifact covers list-result contracts. Neither is an exact
`get_site_details` text-plus-structured response contract for a domain already
selected by an agent. No matching row exists in the current marketer inbox,
distribution log, or portfolio social ledger.

Do not claim detail calls, profile links, monitor-form links, listed domains,
or score-fix links prove customers, owners, consent, endorsements, paid leads,
private demand, completed payments, revenue, crawler compliance, uptime,
security certification, paid ranking, preferred inclusion, A2A support while
the Agent Card route is 404, x402/ACP/SPT/MPP support for NHS, or a
score-methodology bypass.
