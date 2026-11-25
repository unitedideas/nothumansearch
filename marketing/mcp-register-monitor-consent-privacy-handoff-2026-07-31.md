# MCP register-monitor consent and privacy handoff

Date: 2026-07-31
Automation: `business-marketer-not-human-search`
Status: private product handoff only; no monitor registration, email, outreach,
public post, account action, browser action, deploy, crawl, checkout, or
global-queue write performed

## Segment

Make the MCP `register_monitor` path consent-aware and safe for email data
before using it as the next conversion step from `check_url`, result lists, or
public profiles.

The current MCP tool creates or refreshes a monitor immediately from an email
and domain, returns a bearer-style unsubscribe token in the tool result, and
copies the full tool arguments into two analytics tables. There is no
confirmation state or documented consent flag. This is a product-handoff
conversion test, not authorization to register a monitor for anyone.

## Fresh evidence

- Public stats report 4,351 indexed sites, average score 38, and `developer` as
  the top category.
- Seven-day aggregate MCP analytics recorded 49,375 `tools/list`, 14,290
  `initialize`, and 235 `tools/call` requests. `register_monitor=15` is flow
  evidence only; it does not prove 15 people, owners, confirmed inboxes, or
  consented registrations.
- Live JSON-RPC `tools/list` exposes 11 tools. `register_monitor` requires
  `email` and `domain`, says it registers alerts, and returns an unsubscribe
  URL. The tool definition has no side-effect annotation or explicit consent
  field.
- Source inspection shows `toolRegisterMonitor` calls `models.RegisterMonitor`
  immediately. The model inserts an active row by default, or quarantines a
  shared-host apex; there is no pending-confirmation state or confirmation
  token.
- The same MCP request path serializes the full arguments map after the call
  and stores it in `mcp_requests.arguments`. It also stores the arguments map
  again under `intent_events.metadata.arguments`. Because email is required,
  a successful or attempted registration can place the raw email in both
  analytics stores.
- The MCP error fallback returns `registration failed: ` plus the internal
  error string, while the REST monitor handler logs the internal error and
  returns a bounded public message.
- The MCP response includes the full unsubscribe URL in both text and
  `structuredContent`. That token deletes the monitor when visited and should
  not be copied into general-purpose analytics or downstream traces.
- The public `/monitor` page mentions alerts and unsubscribe links but exposes
  no confirmation, privacy, or terms marker. `/privacy`, `/privacy-policy`,
  `/terms`, and `/terms-of-service` returned HTTP 404.
- Aggregate monitor status is five active and three quarantined registrations.
  The redacted score-fix aggregate has ten pending real-candidate rows and no
  real-candidate paid or lead rows.
- `/health`, `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/llms.txt`,
  `/openapi.yaml`, `/api/v1`, `/api/v1/catalog`, `/score`, `/monitor`, and
  `/report` returned HTTP 200. The non-canonical `/catalog` path and
  `/.well-known/agent-card.json` returned 404; A2A claims remain blocked.

No raw MCP arguments, emails, unsubscribe tokens, private monitor rows, private
score-fix rows, checkout URLs, payment identifiers, API keys, IP data, or
customer identifiers were written to this artifact.

## Conversion hypothesis

An agent can safely offer monitoring only if the recipient controls the inbox
and explicitly confirms the subscription:

1. `register_monitor` records a pending confirmation, sends a bounded
   confirmation message, and activates monitoring only after the recipient
   follows the one-time confirmation path.
2. The tool requires an explicit consent assertion and advertises its external
   side effect in MCP annotations and description.
3. MCP and intent analytics retain only an allowlisted event shape such as
   tool, normalized domain, status, and email-domain or irreversible email
   hash. They never retain the raw email, confirmation token, or unsubscribe
   token.
4. The response returns `status=pending_confirmation` and confirmation delivery
   state. It does not expose an unsubscribe token before confirmation.
5. Duplicate requests are idempotent and do not generate repeated messages or
   refresh an active subscription without a bounded policy.

## Acceptance test

1. Add an explicit pending-confirmation state and one-time confirmation token;
   only confirmed rows are eligible for scheduled checks and alerts.
2. Require `consent=true` or an equivalent explicit ownership assertion, add
   MCP side-effect annotations, and keep the tool description clear that email
   is sent.
3. Redact at the shared MCP logging boundary so raw email and management tokens
   cannot reach either `mcp_requests.arguments` or
   `intent_events.metadata.arguments`; cover all current and future sensitive
   tool arguments with an allowlist or field-classification test.
4. Return a backward-compatible result with `ok`, `domain`, `status`, and
   `confirmation_delivery`. Preserve `unsubscribe_url` only for already-active
   legacy clients if necessary, and never log it.
5. Replace raw internal error propagation with stable public error classes and
   keep detailed errors in server logs only.
6. Add tests for missing consent, valid pending registration, confirmation,
   expired and replayed tokens, duplicate registration, delivery failure,
   active legacy rows, quarantine, rate limits, redacted analytics, and safe
   structured responses.
7. After a later product-worker deploy, verify the MCP schema, pending flow,
   confirmation delivery, activation, unsubscribe, analytics redaction, REST
   parity, and monitor worker eligibility with a Foundry-owned test inbox. Do
   not register third-party addresses or run a broad crawl.
8. Measure tool offered, consent asserted, confirmation sent, confirmation
   completed, monitor activated, alert sent, and unsubscribe as separate
   events. Earlier stages are not customers, owner intent, confirmed consent,
   demand, or revenue.

## Duplicate and claim boundary

Existing monitor conversion artifacts cover where to place monitor links and
how to route high-score owners. This handoff is narrower: it fixes the consent,
PII logging, management-token, and side-effect contract underneath every one
of those conversion paths. No exact confirmation-plus-MCP-analytics-redaction
handoff exists in the current marketer inbox, distribution log, or portfolio
social ledger.

Do not claim MCP calls prove customers, unique people, owners, verified inboxes,
consent, monitor activation, private demand, completed payments, revenue,
privacy compliance, security certification, uptime, paid ranking, preferred
inclusion, A2A support while the Agent Card route is 404, or a
score-methodology bypass.
