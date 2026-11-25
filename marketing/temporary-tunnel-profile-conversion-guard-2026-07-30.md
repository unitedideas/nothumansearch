# Temporary-tunnel profile conversion guard

Date: 2026-07-30
Automation: `business-marketer-not-human-search`
Status: no outreach, public post, account action, browser action, deploy, crawl, checkout, or QLimit/global-queue write performed

## Segment

Keep expired temporary-tunnel profiles out of NHS owner-conversion and monitor
acquisition. Two random `*.trycloudflare.com` detail routes are material in the
seven-day aggregate, but both origins are currently unresolvable and neither is
a valid owner lead.

This is a private product-handoff conversion guard. It is not an outreach or
public-content brief.

## Fresh evidence

- Seven-day aggregate traffic included
  `/api/v1/site/foot-sept-oil-til.trycloudflare.com=169` and
  `/api/v1/site/glow-compliance-stanford-enormous.trycloudflare.com=96`.
  Treat these route counts as automated or unknown detail reads, not human
  demand, owner intent, or monitor interest.
- Both origin hostnames failed DNS resolution during bounded root and discovery
  route probes.
- Both NHS public profile routes returned HTTP 200 with `0/100` and `Not
  found`.
- Both public badge routes still returned HTTP 200 even though the origins no
  longer resolved.
- Both `/fix/{host}` routes returned HTTP 404, so no paid remediation offer was
  exposed.
- Both `/monitor?domain={host}` routes returned HTTP 200 with the expired host
  prefilled. The public API detail routes returned HTTP 402 without an API key.
- Public NHS stats reported 4,351 indexed sites with average score 38.
- The live MCP server negotiated protocol `2025-06-18` and returned 11 tools.
  Seven-day aggregate MCP analytics recorded 48,961 `tools/list`, 14,421
  `initialize`, and 238 `tools/call` requests. This is discovery-funnel
  evidence, not customer demand.
- The free-monitor aggregate contains five active and three quarantined
  registrations. The redacted score-fix aggregate contains ten real-candidate
  pending rows and no real-candidate paid or lead rows.
- `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/llms.txt`,
  `/openapi.yaml`, `/api/v1`, `/api/v1/catalog`,
  `/api/v1/api-keys/subscribe`, `/score`, `/monitor`, `/report`, `/newest`,
  `/top`, and `/robots.txt` returned HTTP 200.
  `/.well-known/agent-card.json` returned HTTP 404, so A2A positioning remains
  blocked.

No raw MCP queries, raw user-agent strings, emails, private monitor rows,
private score-fix rows, checkout URLs, payment identifiers, API keys, or
customer identifiers were written to this artifact.

## Conversion guard

Before a profile becomes an owner-conversion or monitor-acquisition candidate:

1. Resolve the current origin and follow its canonical redirect.
2. Treat a non-resolving random `*.trycloudflare.com` host as unavailable until
   a stable canonical domain is supplied and verified.
3. Show an unavailable or expired-profile state instead of presenting the
   `0/100` page as a current readiness assessment.
4. Keep paid score-fix unavailable and require a stable canonical domain before
   accepting a monitor registration for the expired hostname.
5. Exclude these route counts from marketing-demand, owner-lead, monitor-growth,
   and revenue evidence.

Do not apply a blanket rule to every Cloudflare-hosted site. The bounded guard
is for currently non-resolving `*.trycloudflare.com` hostnames with no verified
canonical owner domain.

## Acceptance test

1. The two sampled profile routes render a neutral unavailable/expired state,
   not an actionable owner score.
2. Their badge routes no longer imply a current readiness result without an
   availability qualifier.
3. Their monitor-prefill routes reject the expired host or require a verified
   stable canonical domain before registration.
4. Their paid score-fix routes remain unavailable.
5. Marketing target generation and conversion analytics classify their detail
   reads as temporary-tunnel or unavailable-origin traffic rather than owner
   demand.
6. A stable high-score custom domain still routes to free
   monitor/report/badge proof, and a stable partial-score custom domain still
   routes through `/score` before legitimate remediation.
7. Product tests cover non-resolving temporary hosts, stable custom domains,
   canonical redirects, and recovery if a previously unavailable origin later
   resolves.
8. After a later product-worker deploy, bounded live smokes pass for the two
   sampled profiles, badges, monitor prefills, fix routes, a stable high-score
   profile, a stable partial-score profile, MCP discovery, and the latest
   monitor-worker evidence.

## Duplicate and claim boundary

The portfolio social ledger, NHS distribution history, marketing artifacts,
and marketer inbox had no exact temporary-tunnel owner-conversion guard.
Existing scanner and unavailable-origin briefs cover generic scanner traffic
and one named custom-domain outage; this packet is limited to random,
non-resolving `*.trycloudflare.com` profile traffic.

Do not claim the sampled hosts are customers, owners, partners, endorsements,
paid leads, monitor registrants, human visitors, malicious scanners, abandoned
businesses, permanent outages, completed payments, revenue, security failures,
paid ranking, preferred inclusion, A2A support while the Agent Card route is
404, or a score-methodology bypass.
