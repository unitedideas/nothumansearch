# High-score profile self-serve monitor conversion test

Date: 2026-07-30
Automation: `business-marketer-not-human-search`
Status: no outreach, public post, account action, browser action, deploy, crawl, or checkout performed

## Segment

Use high-score public profiles as a self-serve proof-to-monitor surface. Do not
turn this segment into owner outreach when the owner explicitly asks agents not
to create unrelated outreach messages.

## Fresh evidence

- Current NHS stats report 4,367 indexed sites with average score 38. Live MCP
  `tools/list` returns 11 tools.
- `https://nothumansearch.ai/site/emadibrahim.com` returns 200 and reports
  100/100. Its badge returns 200 with `100 / 100`.
- Aggregate 168-hour traffic records 100 requests to the public badge and 73
  requests carrying `https://emadibrahim.com/` as the referrer. These counts
  are a conversion-design signal only; they do not prove human demand, owner
  intent, badge installation, consent, customers, or revenue.
- The public profile includes badge embed snippets but no direct `/monitor` or
  `/fix/{host}` link.
- `https://nothumansearch.ai/fix/emadibrahim.com` correctly suppresses paid
  remediation for the 100/100 domain and links to the domain-prefilled free
  monitor. `https://nothumansearch.ai/monitor?domain=emadibrahim.com` returns
  200 and prefills the domain in the form.
- The origin currently exposes `llms.txt`, a parseable non-root
  `openapi.json`, an AI-plugin manifest pointing to that live spec, a JSON
  `/api/v1` index, AI-specific robots rules, an MCP manifest, and a live MCP
  server listing two read-only offer tools.
- The origin's public agent instructions say not to create test leads, fake
  reservations, speculative applications, or outreach messages to unrelated
  contacts. No owner touch should be queued from this evidence.
- NHS monitor state is five active registrations and three quarantined rows.
  The latest verified worker evidence remains the 2026-07-27 completed run.
- NHS `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/llms.txt`,
  `/openapi.yaml`, `/api/v1`, `/api/v1/catalog`, `/score`, `/monitor`,
  `/report`, `/newest`, and `/top` return 200. The strict Agent Card route
  `/.well-known/agent-card.json` remains 404.

No raw MCP queries, emails, private monitor rows, private score-fix rows,
checkout URLs, payment identifiers, API keys, or customer identifiers were
written to this artifact.

## Conversion hypothesis

For profiles already at or above the remediation target, place a domain-prefilled
`Monitor this score` action beside the badge/report proof. The paid score-fix
route already performs this handoff, but profile visitors should not need to
discover and traverse `/fix/{host}` to reach the correct free next step.

This is a product handoff, not a public owner-channel action. It should reuse
the existing `/monitor?domain={host}` contract and preserve the paid
remediation path for genuinely partial-score profiles.

## Acceptance test

1. A current high-score profile such as `/site/emadibrahim.com` displays a
   visible, accessible `Monitor this score` link to the correctly prefilled
   monitor form.
2. The same profile keeps the badge embed and report proof without presenting
   the $199 remediation offer.
3. A representative partial-score profile retains its current score-fix path;
   the new high-score CTA must not hide legitimate remediation.
4. Spam, quarantined, unavailable-origin, and stale-profile cases are not
   presented as owner-conversion proof without their existing guards.
5. Live profile, monitor, high-score fix, badge, MCP/API discovery, and monitor
   worker smokes pass after a later product-worker deploy.
6. Analytics distinguish profile-to-monitor clicks from registrations; neither
   metric may be described as demand, consent, a customer, or revenue without
   separate evidence.

## Duplicate and claim boundary

The current NHS repo has no `marketing/social-post-ledger.json`; the canonical
portfolio social ledger and `outreach/distribution_log.csv` contained no
Emad Ibrahim owner-touch fingerprint. The marketer inbox mentions this domain
only as a high-score comparison in an unrelated canonical-host guard. This
artifact intentionally queues no public touch, so no public-action lock was
taken.

Do not imply Emad Ibrahim, his business, profile, badge, referrer string, or
MCP/API surface is an NHS customer, partner, endorsement, paid lead, monitor
registration, badge-install consent, private demand, completed payment, or
revenue proof. Do not claim service quality, capacity, model quality, pricing
accuracy, API uptime, MCP certification, security/privacy compliance, paid
ranking, preferred inclusion, A2A support while the Agent Card route is 404,
or a score-methodology bypass.
