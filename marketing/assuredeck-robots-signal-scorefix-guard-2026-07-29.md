# AssureDeck robots-signal score-fix guard

Date: 2026-07-29
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, crawl, checkout, or global-queue write was performed. This is a
sanitized product/sales handoff for a later NHS operator.

## Fresh evidence

- The public NHS `recent_additions` MCP tool lists
  `assuredeck-evidence-2026.hanjunhee-o.chatgpt.site` as an AI-tools site
  added on 2026-07-24 with a cached score of 70/100. It is absent from prior
  NHS marketing artifacts, `outreach/distribution_log.csv`, the marketer
  inbox, and the checked portfolio social-post ledger.
- The NHS profile and badge return HTTP 200. The profile credits `llms.txt`,
  OpenAPI, a structured API, explicit AI-crawler rules, and Schema.org; it
  marks AI-plugin metadata and MCP as missing.
- The score-fix page returns HTTP 200, projects 70 to 100, and offers the $29
  report and $199 managed implementation. It labels `robots.txt` AI-crawler
  rules as already present, so the proposed uplift only adds AI-plugin and MCP
  points.
- The current origin `robots.txt` returns HTTP 200 but contains only a generic
  `User-Agent: *` policy, API allow/disallow rules, and a sitemap. It contains
  none of the explicit AI-agent tokens NHS currently recognizes:
  GPTBot, ChatGPT, ClaudeBot, Anthropic, Perplexity, Cohere, or Applebot.
  The cached +5 robots signal and the paid checklist are therefore stale or
  misattributed against the current public file.
- The other positive signals are current. `llms.txt` returns HTTP 200 and
  declares `/openapi.json` plus `/api/v1`; `/openapi.json` returns a parseable
  OpenAPI 3.1.0 document with two paths; `/api/v1` returns a public JSON API
  index; and the homepage contains Schema.org JSON-LD.
- The missing AI-plugin and MCP signals are current:
  `/.well-known/ai-plugin.json`, `/mcp`, and `/.well-known/mcp.json` return
  HTTP 404. The score-fix direction is valid, but the current 70-to-100
  promise omits a live robots-policy gap.
- The public owner channel is the site's own `#request` scope-review form.
  No form submission or owner contact was attempted.
- NHS score-band routing is intact: a current 100/100 example reaches the
  already-meets-target monitor handoff, while this partial-score profile
  reaches the paid remediation intake.
- Public NHS stats currently report 4,367 sites with average score 38. NHS
  MCP discovery exposes 11 tools. Aggregate seven-day use is
  `tools/list=48,783`, `initialize=14,840`, and `tools/call=234`; these are
  discovery-funnel counts, not customer or revenue proof.
- NHS owner and discovery surfaces remain live: `/score`, `/monitor`,
  `/report`, `/mcp`, `/.well-known/mcp.json`,
  `/.well-known/agent.json`, `/.well-known/commerce.json`,
  `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`,
  `/api/v1/api-keys/subscribe`, `/llms.txt`, `/openapi.yaml`, and `/feed.xml`
  return HTTP 200. `/.well-known/agent-card.json` remains HTTP 404.
- The API-plan discovery guard remains active: catalog and subscribe expose
  the $9.99/month unlimited plan, while `/llms.txt` and OpenAPI still
  advertise legacy starter/pro/scale contracts. Do not add API-plan pricing
  to owner copy from this packet.
- The latest monitor worker completed on 2026-07-27. Aggregate monitor state
  is five active and three quarantined. Aggregate score-fix state is ten real
  candidates still pending; paid and lead rows are test-like only. These are
  private workflow guards, not public demand or revenue claims.

## Decision

Do not contact the AssureDeck owner or use this profile as score-fix proof
until NHS reruns or reconciles the robots signal. The current paid page says
AI-crawler rules already exist and promises a 70-to-100 uplift, while the live
file does not contain an explicit AI-agent policy recognized by NHS.

The bounded owner opportunity remains valid after correction: the site
already publishes strong agent-readable API contracts and a direct scope
form, while AI-plugin and MCP discovery are absent. The remediation checklist
must also include the current robots-policy gap if 100/100 remains the stated
target.

## Product/sales handoff

1. Run one bounded read-only refresh for this exact host; do not run a broad
   recrawl.
2. Reconcile the cached robots signal with the current `robots.txt`. If the
   file remains generic, remove the +5 credit and mark explicit AI-agent rules
   as missing.
3. Regenerate the score-fix checklist from the refreshed signals. Keep the
   existing OpenAPI and `/api/v1` contracts credited; include AI-plugin, MCP,
   and explicit AI-agent robots policy as the current gaps.
4. Verify that the refreshed score, projected score, $29 report, and $199
   checklist agree before using the profile as proof or contacting the owner.
5. Only after the checklist is current, prepare one deduplicated owner touch
   through the site's public scope-review form. Verify active Foundry/Owl
   sender identity, recheck the portfolio social ledger and distribution
   history, and take a sync-state public-action lock first.

## Claims to avoid

Do not imply AssureDeck, its owner, its procurement guidance, its API, or its
indexed profile is an NHS customer, partner, endorsement, paid lead, monitor
registration, badge install, private demand, completed payment, or revenue
proof. Do not claim audit or certification authority, legal advice,
procurement outcomes, AI-CAIQ compliance, buyer approval, API uptime,
privacy/security proof, explicit AI-crawler permission while the current file
lacks recognized agent tokens, paid ranking, preferred inclusion,
score-methodology bypass, or A2A support while NHS
`/.well-known/agent-card.json` is 404.
