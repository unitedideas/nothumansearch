# Recent 100-score monitor owner brief

Date: 2026-07-29
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, crawl, checkout, or global-queue write was performed. This is a
business-local, no-send owner-conversion brief for a later gated operator.

## Fresh evidence

- Public NHS stats report 4,367 indexed sites with an average score of 38.
- MCP `recent_additions` returns five sites added in the last seven days.
  Two are currently scored 100/100: `ai-akari.ai` and `bbwbelles.com`.
- Both public NHS profiles return 200 and expose a current 100/100 badge.
  Neither profile contains a direct `/monitor` or `/fix/{host}` link.
- Both `/fix/{host}` routes correctly suppress paid remediation and route to
  the prefilled free monitor:
  - `https://nothumansearch.ai/monitor?domain=ai-akari.ai`
  - `https://nothumansearch.ai/monitor?domain=bbwbelles.com`
- `/newest` has a generic free-monitor CTA, but the two site cards do not carry
  a domain-prefilled monitor action.
- `ai-akari.ai` publishes a public owner contact in its AI-plugin manifest and
  public X/GitHub owner channels. Its declared non-root OpenAPI document is
  parseable, and its live MCP server lists three tools.
- `bbwbelles.com` publishes a public partner contact in its AI-plugin
  manifest. Its root OpenAPI document is parseable, and its live MCP server
  lists seven tools.
- NHS discovery surfaces remain coherent at 11 MCP tools. Aggregate seven-day
  MCP activity records 48,603 `tools/list` calls, 29 `recent_additions` calls,
  and 15 `register_monitor` calls. These are funnel observations, not customer
  or demand proof.
- The latest monitor worker completed on 2026-07-27. Aggregate state is five
  active monitors and three quarantined monitors; quarantine details stay
  private/admin-only.
- `/.well-known/agent-card.json` remains 404, and the separate API-plan
  discovery-pricing drift remains open. Neither claim belongs in this owner
  recognition test.

## Segment

Recently indexed 100/100 owners whose public machine-readable contacts make a
single recognition-and-monitor note possible without selling remediation.
High-score owners should receive proof, regression monitoring, and badge/report
links—not a score-fix pitch.

## No-send draft: ai-akari.ai

Subject: `ai-akari.ai is 100/100 on Not Human Search`

`ai-akari.ai` is now indexed at 100/100:
https://nothumansearch.ai/site/ai-akari.ai

The paid remediation path is disabled at that score. The useful follow-on is
the free weekly regression monitor:
https://nothumansearch.ai/monitor?domain=ai-akari.ai

The report page also has a badge embed if public score proof is useful.

## No-send draft: bbwbelles.com

Subject: `bbwbelles.com is 100/100 on Not Human Search`

`bbwbelles.com` is now indexed at 100/100:
https://nothumansearch.ai/site/bbwbelles.com

The paid remediation path is disabled at that score. The useful follow-on is
the free weekly regression monitor:
https://nothumansearch.ai/monitor?domain=bbwbelles.com

The report page also has a badge embed if public score proof is useful.

## Execution gate

1. Pick exactly one owner; do not send both in the same execution window.
2. Refresh the public profile, badge, prefilled monitor URL, high-score
   `/fix/{host}` handoff, owner contact, and latest monitor-worker proof.
3. Verify the active Foundry/Owl-owned sending account, check the portfolio
   social-post ledger if the selected channel is social, check
   `outreach/distribution_log.csv`, and confirm no matching owner fingerprint.
4. Take and verify a sync-state public-action lock before the external action.
5. Send one proof-first note with no paid offer. Record the message id or public
   URL and the exact channel result in the appropriate ledger.

## Guardrails

- Do not imply either domain, owner, profile, badge, or recent-additions entry
  is a customer, partner, endorsement, paid lead, monitor registration,
  badge-install consent, private demand, completed payment, or revenue.
- Do not claim medical or wellness efficacy, formalwear fit, inventory,
  delivery, MCP certification, security/privacy compliance, API uptime,
  crawler compliance, SEO lift, paid ranking, preferred inclusion, A2A
  support, or a score-methodology bypass.
- Do not mention NHS API plan names, prices, or quotas while the separate
  discovery-pricing drift remains open.
- Respect `ai-akari.ai`'s published boundary: do not turn its free support
  surfaces or difficult-state content into purchase pressure.
