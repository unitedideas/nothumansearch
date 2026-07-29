# LetsFG discovery-path score-fix guard

Date: 2026-07-29
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, crawl, checkout, or global-queue write was performed. This is a
sanitized product/sales handoff for a later NHS operator.

## Fresh evidence

- Aggregate traffic over 168 hours includes 162 requests to
  `/site/letsfg.co`. Treat this as route activity only, not owner intent or
  buyer demand.
- The public NHS profile returns HTTP 200 at a cached score of 65/100. It says
  OpenAPI and MCP are missing and links to paid score-fix offers: a $29 report
  and $199 managed remediation.
- The public score-fix page explicitly proposes shipping `openapi.yaml` for
  +20 and an MCP server for +10.
- The live `/.well-known/ai-plugin.json` is valid and declares
  `https://letsfg.co/developers/api/openapi.json` as the OpenAPI contract and
  `https://letsfg.co/developers/api/mcp` as the canonical MCP endpoint.
- The declared OpenAPI URL returns HTTP 200, parses as OpenAPI 3.1.0, and
  contains 22 paths.
- A read-only JSON-RPC `tools/list` call to the declared MCP endpoint returns
  HTTP 200 with eight tools. The root `/mcp` path returns 404 and GET on the
  declared MCP endpoint returns 405, which is consistent with a POST-only
  JSON-RPC service.
- The origin's `llms.txt` also documents the developer API and MCP access.
  Conventional root probes still return 404 at `/openapi.yaml`, `/api/v1`,
  `/.well-known/mcp.json`, and `/mcp`.
- No matching LetsFG row exists in `outreach/distribution_log.csv`,
  `ops/sweeper/marketer-inbox.jsonl`, or the existing marketing artifacts.
  The public plugin exposes an owner contact, but no outreach was sent.
- NHS discovery surfaces remain available: `/score`, `/monitor`, `/report`,
  `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`,
  `/api/v1/catalog`, `/api/v1/api-keys/subscribe`, `/llms.txt`, and
  `/openapi.yaml` returned HTTP 200. `/.well-known/agent-card.json` remains
  HTTP 404.
- Live NHS MCP discovery lists 11 tools. Aggregate seven-day usage is
  `tools/list=48,842`, `initialize=15,081`, and `tools/call=224`; these are
  discovery-funnel counts, not customer or revenue proof.
- The latest monitor worker completed on 2026-07-27. Aggregate monitor state
  is five active and three quarantined. Aggregate score-fix state is ten real
  candidates still pending and no real paid or lead row. These are private
  workflow guards, not public demand or revenue claims.

## Decision

Do not contact the LetsFG owner or use this profile in public score-fix copy
until NHS reconciles declared non-root discovery endpoints. The paid page
currently offers to create an OpenAPI contract and MCP server that the origin
already exposes at the URLs declared in its AI plugin and `llms.txt`.

This is not evidence that the score should be changed blindly. It is evidence
that the crawler or profile needs one bounded path-resolution review before
the profile can support an owner-conversion claim.

## Product/sales handoff

1. Re-run one bounded check that follows the OpenAPI URL declared in
   `ai-plugin.json` and the MCP URL declared in the plugin and `llms.txt`.
2. Decide whether NHS should score valid declared non-root OpenAPI and MCP
   endpoints, or explicitly explain why only conventional root paths count.
3. If declared paths qualify, refresh the profile and remove the already-built
   OpenAPI/MCP items from the paid remediation checklist.
4. If declared paths do not qualify, change the profile and score-fix language
   so the owner is offered discoverability aliases or compatibility work, not
   creation of capabilities that already exist.
5. Only after the profile is current, prepare one owner-channel touch using the
   canonical public contact. Verify active account identity, duplicate ledgers,
   and the sync-state public-action lock first.

## Claims to avoid

Do not claim LetsFG is a customer, partner, endorsement, paid lead, monitor
registration, badge-install consent, private demand, completed payment,
revenue, flight-price authority, booking provider endorsement, travel
fulfillment proof, MCP certification, security/privacy proof, crawler
compliance, uptime proof, A2A support, paid ranking customer, preferred
listing, or score-methodology bypass. Do not claim the profile traffic is
human or that the current score is definitively wrong before the product-side
path-resolution review.
