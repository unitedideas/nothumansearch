# Vatsal Shah canonical-host score-fix guard

Date: 2026-07-29
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, crawl, checkout, or global-queue write was performed. This is a
sanitized product/sales handoff for a later NHS operator.

## Fresh evidence

- Aggregate traffic over 168 hours includes 127 requests to
  `/site/vatsalshah.in`. Treat this as route activity only, not owner intent or
  buyer demand.
- The public NHS profile for `vatsalshah.in` returns HTTP 200 with a cached
  score of 55/100. Its score-fix page offers a $29 report and $199 managed
  remediation, including OpenAPI, `/api/v1`, and an MCP server.
- The live origin redirects every tested `vatsalshah.in` route to
  `vatsalshah.ca`. The canonical `.ca` homepage, `llms.txt`, AI plugin, and
  robots file return HTTP 200. The public NHS profile and score-fix routes for
  `vatsalshah.ca` return HTTP 404.
- The canonical `llms.txt` explicitly describes a professional-services
  business with no public API, SDK, MCP server, CLI, or self-serve API keys
  planned. It directs agents to public readiness, contact, and scheduling
  pages instead.
- The canonical AI plugin points its `api.url` at `llms-full.txt`, not an
  OpenAPI contract. Conventional `/openapi.yaml`, `/api/v1`, and `/mcp`
  probes return HTTP 404. The current 55 score is therefore not evidence of a
  missed non-root OpenAPI or MCP endpoint; the conversion conflict is the
  retired host plus an offer that ignores the canonical site's declared
  access model.
- A high-score comparison remains correctly gated: `emadibrahim.com` has a
  public 100/100 profile, a working two-tool MCP endpoint, a structured API,
  and a score-fix route that sends the owner to monitoring instead of paid
  remediation.
- Public NHS stats report 4,406 sites with average score 38. Live NHS MCP
  discovery lists 11 tools; aggregate seven-day use is
  `tools/list=48,824`, `initialize=14,990`, and `tools/call=227`. These are
  discovery-funnel counts, not customer or revenue proof.
- NHS owner surfaces remain live: `/score`, `/monitor`, `/report`, `/mcp`,
  `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`,
  `/api/v1/catalog`, `/llms.txt`, and `/openapi.yaml` return HTTP 200.
  `/.well-known/agent-card.json` remains HTTP 404.
- The latest monitor worker completed on 2026-07-27. Aggregate monitor state
  is five active and three quarantined. Aggregate score-fix state is ten real
  candidates still pending and no real paid or lead row. These are private
  workflow guards, not public demand or revenue claims.
- No Vatsal Shah row exists in `outreach/distribution_log.csv`, the NHS
  marketer inbox, existing NHS marketing artifacts, or the checked portfolio
  social-post ledger.

## Decision

Do not contact the owner or use this profile in score-fix copy until NHS
reconciles the `.in` alias with the canonical `.ca` site. The current paid
path can sell implementation against a retired hostname and projects a
100-point API/MCP uplift even though the canonical owner-facing instructions
say those integration surfaces are intentionally not offered.

The score can remain a truthful readiness measure without turning every
missing signal into a sales requirement. Remediation copy should distinguish
an implementation gap from an intentional access-model boundary.

## Product/sales handoff

1. Run one bounded canonical-host review for `vatsalshah.in` and its redirect
   target `vatsalshah.ca`.
2. Make the public profile and `/fix/{host}` flow resolve the canonical host,
   or mark the `.in` record as an alias/stale host rather than selling work
   against it.
3. Refresh the canonical `.ca` profile before any owner-channel action.
4. Rewrite the remediation checklist so intentionally absent API/MCP surfaces
   are described as optional capability choices, not capabilities the owner
   must buy to be agent-usable or to receive preferred placement.
5. Only after the profile, host, and checklist agree, prepare one
   deduplicated owner touch through a canonical public channel. Verify active
   account identity and take the sync-state public-action lock first.

## Claims to avoid

Do not imply Vatsal Shah, Agent Readiness Studio, Emad Ibrahim, any profile,
badge, route, redirect, or listed domain is a customer, partner, endorsement,
paid lead, monitor registration, badge-install consent, private demand,
completed payment, or revenue proof. Do not claim the `.in` host is a current
independent property, that the owner needs an API or MCP server, that the
canonical site is broken, that route traffic is human, or that remediation
buys ranking, preferred inclusion, score-methodology bypass, or A2A support.
