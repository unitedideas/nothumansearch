# Score-fix funnel measurement handoff

Date: 2026-07-31
Automation: `business-marketer-not-human-search`
Status: private owner-conversion test only; no outreach, public post, account
action, browser action, product-code edit, deploy, crawl, checkout, monitor
registration, or global-queue write performed

## Segment

Make the score-fix funnel measurable without turning page views, bot traffic,
or private identifiers into marketing claims. NHS already logs report, fix-page,
score-page, score-check, and payment events, but the current admin surfaces do
not expose a safe stage-by-stage conversion contract for recurring business
agents.

This is narrower than the May organic CTA and score-fix abandonment artifacts.
Those define routing and pending-row follow-up. This handoff defines the
aggregate event taxonomy and privacy boundary required before sending more
owner traffic into that path.

## Fresh aggregate evidence

- Public stats report 4,343 indexed sites, average score 38, and `developer` as
  the top category.
- Seven-day aggregate intent totals are 20,268 events: 7,358 classified human
  and 12,910 classified bot.
- Event aggregates are:
  - `site_report_view`: 12,005 total; 4,080 classified human.
  - `fix_page_view`: 7,951 total; 3,241 classified human.
  - `mcp_tool_call`: 237 total; all classified bot.
  - `score_page_view`: 71 total; 33 classified human.
  - `score_checked`: 4 total; all classified human.
- These are request/event classifications, not people, owners, leads, or
  conversions. The large gap between report/fix views and score checks is not a
  conversion rate because the stages do not share a published aggregate-safe
  cohort or attribution contract.
- The redacted score-fix store contains ten real-candidate pending rows: seven
  older than 30 days and three aged 7–29 days. It contains no real-candidate
  paid or lead row. Test-like paid, lead, pending, and internal-test rows remain
  excluded from customer evidence.
- The admin traffic response has `daily`, `errors`, `top_pages`, and
  `top_referrers`, but no `route_buckets` or `scanner_pages`. Its current top
  owned paths include `/` at 3,573, `/llms.txt` at 460, and
  `/api/v1/catalog` at 355.
- Seven-day MCP aggregates record 49,967 `tools/list`, 14,429 `initialize`, and
  237 `tools/call`. The leading actual tools are `get_top_sites` 92,
  `get_stats` 51, `recent_additions` 30, `list_categories` 22, `submit_site`
  17, and `register_monitor` 15. Enumeration and tool calls are not owner or
  buyer intent.
- `/score`, `/monitor`, and `/report` returned HTTP 200. `/privacy`, `/terms`,
  and `/.well-known/agent-card.json` returned 404.
- The July 31 full recrawl closed green and no crawl lock remained during this
  scout.

No raw recent-event rows, entity ids, domains from private rows, emails, IP
hashes, user agents, queries, referrers, checkout URLs, payment identifiers,
management tokens, or customer identifiers were written to this artifact.

## Measurement hypothesis

A reliable owner-conversion test needs one aggregate-safe contract from public
proof to paid remediation:

1. Report/profile viewed.
2. Eligibility evaluated with a bounded reason class.
3. Score page viewed and score check completed.
4. Remediation form started and valid intake submitted.
5. Checkout started, payment completed, and fulfillment started.

Every stage should separate bot/scanner traffic from human-classified traffic,
deduplicate repeated requests within a bounded visit window, and report only
coarse score bands, eligibility classes, source families, and counts. The
conversion view must never require a raw host, email, IP hash, payment id,
checkout URL, query, or event-metadata dump.

## Acceptance test

1. Define stable events for `report_view`, `eligibility_evaluated`,
   `score_view`, `score_completed`, `fix_form_started`, `fix_intake_submitted`,
   `checkout_started`, `fix_paid`, and `fulfillment_started`; map current event
   names backward-compatibly.
2. Add bounded dimensions only: day, source family, coarse score band,
   eligibility status/reason, bot/human class, and payment mode. Do not expose
   raw host, email, visit id, IP hash, user agent, query, referrer, Stripe id,
   checkout URL, management token, or arbitrary metadata.
3. Add aggregate `route_buckets` and `scanner_pages` to traffic analytics so
   discovery, profile/report, score, fix, monitor, commerce, and scanner traffic
   can be separated without per-path domain disclosure.
4. Make the recurring-worker admin view aggregate-only by default. Remove or
   separately gate the current `recent` metadata and entity-level drilldown;
   marketer and planner workers must not need raw rows to measure the funnel.
5. Use allowlisted attribution values and a bounded dedupe window. Keep views,
   classified-human requests, form starts, valid intakes, checkout starts,
   payments, and fulfillment as distinct counts.
6. Add fixtures for scanners, bots, repeated views, missing visit ids,
   high-score ineligible routes, partial-score eligible routes, abandoned
   checkout, completed payment, test-like rows, and aggregate redaction.
7. After a later product-worker deploy, collect a bounded baseline for at least
   one full week before changing owner-channel volume. Reconcile aggregate
   funnel counts with the redacted score-fix status helper; never use raw rows
   in the marketing scout.
8. Do not suggest monitor registration from this funnel until the separate MCP
   monitor consent, confirmation, token, and telemetry repair is complete.

## Claim boundary

Do not describe report views, fix-page views, human classification, score
checks, MCP calls, pending rows, form starts, checkout starts, or profile traffic
as people, owners, customers, demand, leads, payments, revenue, endorsements,
uptime proof, security certification, paid ranking, preferred inclusion, A2A
support while the Agent Card route is 404, or a score-methodology bypass.
