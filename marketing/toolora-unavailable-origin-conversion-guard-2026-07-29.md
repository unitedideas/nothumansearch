# Toolora unavailable-origin conversion guard

Date: 2026-07-29
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, crawl, checkout, or global-queue write was performed. This is a
sanitized product/sales handoff for a later NHS operator.

## Fresh evidence

- Aggregate traffic over 168 hours puts `/site/toolora.dev` at 1,595 requests,
  behind only `/` and `/.well-known/commerce.json` among the returned top
  pages. Treat this as route activity only, not owner intent or buyer demand.
- The public NHS profile returns HTTP 200 with a cached score of 45/100 and a
  live `$199` score-fix CTA. It reports `llms.txt`, MCP, AI-friendly robots,
  and Schema.org as present, while plugin metadata, OpenAPI, and a structured
  API are missing.
- The public score-fix route and badge still return HTTP 200.
- Fresh read-only origin probes returned HTTP 404 for `https://toolora.dev/`,
  `/mcp`, `/api/mcp`, `/llms.txt`, `/.well-known/mcp.json`,
  `/.well-known/ai-plugin.json`, `/openapi.yaml`, `/api/v1`, `/robots.txt`,
  `/health`, `/docs`, and `/sitemap.xml`, including browser-like and NHS bot
  user agents.
- No matching entry exists in `outreach/distribution_log.csv`, and
  `marketing/social-post-ledger.json` is absent. The canonical origin does not
  currently expose a usable owner-contact route.
- NHS discovery surfaces remain available: `/score`, `/monitor`, `/report`,
  `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`,
  `/api/v1/catalog`, `/api/v1/api-keys/subscribe`, `/llms.txt`, and
  `/openapi.yaml` returned HTTP 200. `/.well-known/agent-card.json` remains
  HTTP 404.
- Live MCP discovery lists 11 tools. Aggregate seven-day usage is
  `tools/list=48,805`, `initialize=15,143`, and `tools/call=225`; the main
  successful tool calls are `get_top_sites=95`, `get_stats=41`,
  `recent_additions=23`, `submit_site=21`, `list_categories=21`, and
  `register_monitor=15`.
- The latest monitor worker completed on 2026-07-27. Aggregate monitor state
  is five active and three quarantined. Aggregate score-fix state is ten real
  candidates still pending and no real paid or lead row. These are private
  workflow guards, not public demand or revenue claims.

## Decision

Do not use the Toolora profile for owner outreach, public score-fix copy, or a
case study until a later product/operator run resolves the conflict between
the cached 45/100 profile and the currently unavailable origin.

This is a conversion-boundary problem: an active paid remediation CTA on a
profile whose origin currently returns 404 can turn high route traffic into a
misleading offer. The route count is large enough to justify one product
handoff, but it does not prove a human owner saw the page.

## Product/sales handoff

1. Confirm whether the origin outage is transient, an intentional shutdown,
   or a stale indexed record using a bounded read-only availability check.
2. If the origin remains unavailable, mark the profile stale/unreachable and
   replace the paid score-fix CTA with a rerun or availability handoff.
3. If the origin returns, refresh the public score before using any signal,
   score, MCP, or missing-surface claim.
4. Only after a current profile exists, identify a canonical public owner
   channel and prepare one gated score-band-aware touch. Verify active account
   identity, duplicate ledgers, and the sync-state public-action lock first.
5. Keep high-score owners on free monitor/report/badge proof. Offer paid
   remediation only for current, concrete, owner-fixable public gaps.

## Claims to avoid

Do not claim Toolora is a customer, partner, endorsement, paid lead, monitor
registration, badge-install consent, private demand, completed payment,
revenue, active service, current MCP provider, security/privacy proof, uptime
proof, ranking customer, or score-fix prospect. Do not claim the traffic is
human, the origin is permanently dead, or the cached score is current. NHS
does not sell ranking placement or score-methodology bypasses.
