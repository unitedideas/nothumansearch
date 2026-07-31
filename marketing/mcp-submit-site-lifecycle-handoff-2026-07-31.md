# MCP submit-site lifecycle handoff

Date: 2026-07-31
Automation: `business-marketer-not-human-search`
Status: private product handoff only; no site submission, outreach, public post, account action, browser action, deploy, crawl, checkout, or global-queue write performed

## Segment

Make the MCP `submit_site` lifecycle truthful and machine-actionable. The tool
can index a site inline, fail inline, or leave it queued, but the current
response does not give agents a stable status contract. Its queued copy also
promises pickup "within the hour" even though the installed full-recrawl job is
scheduled at 06:00 and 12:30 local time.

This is a product-handoff conversion test. It does not authorize a site
submission or owner contact.

## Fresh evidence

- Public stats report 4,351 indexed sites, average score 38, and `developer` as
  the top category.
- Seven-day aggregate MCP analytics recorded 49,294 `tools/list`, 14,372
  `initialize`, and 235 `tools/call` requests.
- `submit_site=17` and `register_monitor=15`. Treat these as product-flow
  evidence, not customer demand, owner intent, or monitor consent.
- Live JSON-RPC `tools/list` returned the expected 11 tools. The `submit_site`
  schema requires only a URL and says a successful inline crawl can become
  searchable within seconds.
- Source inspection shows three `submit_site` outcomes:
  1. inline crawl and index success;
  2. inline crawl or index-write failure with a later retry claim;
  3. semaphore-busy queueing with the text "scheduled recrawl will pick it up
     within the hour."
- The installed `com.foundry.nothumansearch.recrawl` calendar has two daily
  windows: 06:00 and 12:30 local time. It is not an hourly worker.
- The structured response contains only `url`, `crawled`, and `queued`. It does
  not expose a stable lifecycle status, submission id, status URL, profile URL,
  report URL, score, category, next scheduled window, or suggested next tool.
- The public API root documents `POST /api/v1/submit`, but exposes no
  submission-status endpoint. No mutating submit smoke was run in this scout.
- The monitor aggregate contains five active and three quarantined
  registrations. The redacted score-fix aggregate contains ten pending
  real-candidate rows and no real-candidate lead or paid rows. Neither proves a
  submitter became an owner lead or buyer.
- `/health`, `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/llms.txt`,
  `/openapi.yaml`, `/api/v1`, `/api/v1/catalog`, `/score`, `/monitor`, and
  `/report` returned HTTP 200. `/.well-known/agent-card.json` returned 404, so
  A2A claims remain blocked.

No raw MCP arguments, submitted URLs, user-agent strings, emails, private
monitor rows, private score-fix rows, checkout URLs, payment identifiers, API
keys, or customer identifiers were written to this artifact.

## Conversion hypothesis

Agents should be able to distinguish a completed index write from deferred
work without guessing from prose:

1. Inline success returns `status=crawled`, domain, score, category,
   `profile_url`, and `report_url`.
2. Deferred work returns `status=pending` and a truthful timing field grounded
   in the installed worker cadence, or explicitly makes no latency promise.
3. Inline failure returns `status=retry_scheduled` or `status=failed` with a
   bounded error class, not raw upstream error text.
4. A safe opaque status handle or documented bounded `check_url` follow-up lets
   the caller recheck without exposing the submission queue.
5. `register_monitor` becomes a suggested next tool only after a domain has a
   valid indexed profile; it is not consented or called automatically.

## Acceptance test

1. Preserve the existing text response while adding a backwards-compatible
   lifecycle object with an explicit status enum.
2. Remove the ungrounded "within the hour" promise. If an ETA is returned, it
   must come from the installed recurring schedule or durable queue state.
3. Inline success includes domain, score, category, profile URL, and report URL
   in `structuredContent`.
4. Deferred and failed results include a safe recheck path without exposing raw
   submission rows, emails, IP data, error bodies, or predictable queue ids.
5. Add tests for inline success, crawl failure, index-write failure,
   semaphore-busy queueing, duplicate submission, schedule-derived timing, and
   backwards-compatible fields.
6. After a later product-worker deploy, verify `tools/list` plus the outcome
   branches with fixtures or bounded internal test URLs; do not run a broad
   recrawl for acceptance.
7. Measure submission accepted, inline indexed, deferred, failed, profile
   viewed, report viewed, and monitor registered as separate events. None of
   the earlier stages may be described as demand, customers, consent, or
   revenue.

## Duplicate and claim boundary

An older May submit-site onboarding packet proposes score and monitor next
steps. This test is narrower: it fixes the asynchronous lifecycle, schedule
promise, and structured response contract before that conversion path. The
portfolio social ledger and current marketer inbox contain no exact
submit-status and schedule-truthfulness handoff.

Do not claim submit calls prove customers, owners, endorsements, private
demand, indexing success, crawl freshness, monitor consent, completed payments,
revenue, uptime, paid ranking, preferred inclusion, A2A support while the Agent
Card route is 404, or a score-methodology bypass.
