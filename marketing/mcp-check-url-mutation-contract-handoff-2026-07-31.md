# MCP `check_url` mutation-contract handoff

Date: 2026-07-31
Automation: `business-marketer-not-human-search`
Status: private product/conversion handoff only; no MCP tool invocation, crawl,
outreach, public post, account action, browser action, product-code edit, deploy,
checkout, monitor registration, or global-queue write performed

## Segment

Make the on-demand readiness check trustworthy before routing MCP clients or
site owners back into it. The public tool reads like a verify-before-use check,
but the implementation also rewrites the indexed site record and duplicates the
full input arguments into two analytics stores.

This supersedes the public-use portion of the May `check_url`-to-monitor briefs.
Those briefs assumed the check was a safe activation surface. It should not be
promoted again until its mutation, telemetry, and side-effect contract are
explicit and tested.

## Fresh bounded evidence

- Public stats report 4,343 indexed sites, average score 38, and `developer` as
  the top category.
- Seven-day MCP aggregates report 50,212 `tools/list`, 14,538 `initialize`, and
  237 `tools/call`. The current nonzero tool aggregate has no `check_url` row;
  this is not evidence of zero lifetime use or lack of demand.
- Live MCP `tools/list` describes `check_url` as re-runnable without the
  submissions-table side effect of `submit_site`, but returns no MCP
  `annotations` declaring whether the tool is read-only or mutating.
- Live `GET /api/v1/check` documents an on-demand score and the free rate limit,
  but does not disclose that successful checks refresh the public index.
- Source inspection shows both MCP `check_url` and REST `POST /api/v1/check`
  call `models.UpsertSite` asynchronously after a successful crawl. The upsert
  rewrites score, signal flags, descriptive fields, category, tags, MCP endpoint,
  crawl state, favicon data, and `last_crawled_at` for an existing domain.
- The shared MCP boundary also writes the complete `url` argument into
  `mcp_requests.arguments` and `intent_events.metadata.arguments`. A caller can
  supply a full URL, so query strings, fragments, userinfo, or signed values must
  not be retained even though the crawler later normalizes the indexed site to
  scheme plus host.
- `/mcp`, `/api/v1/check`, `/llms.txt`, `/openapi.yaml`, `/api/v1`, and
  `/.well-known/mcp.json` returned HTTP 200. `/.well-known/agent-card.json`
  returned 404, so this is not A2A proof.
- The redacted monitor store remains five active and three quarantined. The
  redacted score-fix store remains ten real-candidate pending rows and no
  real-candidate paid or lead rows. Neither is evidence that `check_url` should
  be promoted more broadly.
- The repo-local `marketing/social-post-ledger.json` is absent. The distribution
  history and marketer inbox contain older `check_url` conversion briefs but no
  mutation-contract repair. No public fingerprint or submission was queued.

No raw URLs from tool calls, queries, user-agent strings, IP hashes, emails,
monitor rows, private hosts, payment ids, checkout URLs, management tokens, or
customer identifiers were written to this artifact.

## Contract decision

Choose one behavior and make every discovery surface agree:

1. **Read-only check:** return a live score without changing the index. Any
   refresh becomes a separately named, explicitly mutating action.
2. **Refresh-and-check:** keep the upsert, label it as a public-index refresh,
   declare the side effect in MCP annotations and REST/OpenAPI docs, and return
   whether persistence succeeded instead of hiding the asynchronous write.

The read-only contract is safer for a tool marketed as verify-before-use. If the
index-compounding behavior is retained, the response cannot claim or imply that
the operation is merely observational.

## Acceptance test

1. Decide and document whether MCP `check_url` and REST `/api/v1/check` are
   read-only or refresh-and-check; keep their behavior and response schema in
   parity.
2. If read-only, remove the `UpsertSite` side effect and prove a successful
   bounded check changes neither an existing row nor index membership.
3. If mutating, add accurate MCP annotations and descriptions, REST/OpenAPI and
   `/api/v1` disclosure, a persistence outcome field, and bounded public error
   classes. Do not report success before the write outcome is known.
4. Normalize input to scheme plus hostname before logging. Reject userinfo and
   strip query strings and fragments from both analytics stores; retain only
   allowlisted aggregate fields such as coarse result status and score band.
5. Add tests for existing and new domains, persistence failure, duplicate
   checks, timeouts, URL userinfo, signed query strings, fragments, private-host
   rejection, analytics redaction, REST/MCP parity, rate limits, and MCP
   annotation accuracy.
6. After a later product-worker deploy, verify live `tools/list`, MCP response,
   `GET /api/v1/check`, REST response, `/api/v1`, `/openapi.yaml`, `/llms.txt`,
   `/.well-known/mcp.json`, aggregate-only analytics, and the chosen persistence
   behavior with a Foundry-owned fixture.
7. Keep `register_monitor` out of the next-action response until its separate
   consent, confirmation, token, and telemetry repair is complete.
8. Do not revive the older public `check_url` conversion drafts until this
   acceptance test passes and any external use has identity verification,
   duplicate-ledger checks, a sync-state public-action lock, and delivery proof.

## Claim boundary

Do not describe checks, index refreshes, report views, monitor rows, MCP calls,
or human-classified traffic as people, owners, customers, demand, leads,
payments, revenue, endorsements, uptime proof, security certification, crawler
permission, paid placement, preferred inclusion, A2A support, or a
score-methodology bypass.
