# NHS Badge/Profile Owner Conversion Test

Status: prepared, not implemented.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-14T15:25Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a scout artifact for a later product or channel operator.

## Live Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4172`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories; `ai-tools=899`, average score `40`.
- `https://nothumansearch.ai/.well-known/mcp.json`: advertises 11 tool definitions and the public/audit-only category split.
- `https://nothumansearch.ai/.well-known/agent.json`: HTTP 200, with REST, MCP, commerce, quote, checkout, and API subscription metadata.
- `https://nothumansearch.ai/.well-known/agent-card.json`: HTTP 404, so A2A-style directory work remains blocked until compatibility is added or intentionally documented.
- `https://nothumansearch.ai/api/v1/catalog`: score-fix plus Starter, Pro, and Scale API subscriptions are listed.
- `https://nothumansearch.ai/site/xquik.com`: HTTP 200, public score `100/100`.
- `https://nothumansearch.ai/site/bernstein.run`: HTTP 200, public score `90/100`.

Aggregate admin evidence, last 336 hours:

- `/badge/xquik.com.svg`: 843 requests.
- `/site/xquik.com`: 159 requests.
- `/.well-known/commerce.json`: 1,282 requests.
- `/api/v1/catalog`: 288 requests.
- `/api/v1/checkout`: 270 requests.
- `/api/v1/quote`: 270 requests.
- Google referrers: 202 combined requests from `google.com` and `www.google.com`.

Aggregate MCP evidence, last 7 days:

- `tools/list`: 111,201 calls.
- `initialize`: 14,594 calls.
- `tools/call`: 335 calls.
- Top called tools: `search_agents=196`, `get_site_details=37`, `find_mcp_servers=26`, `get_stats=18`, `verify_mcp=17`, `check_url=15`, `get_top_sites=14`.
- No unknown MCP tool names appeared in the aggregate tool list.

Aggregate private workflow evidence:

- Monitor status: `active=1`, `quarantined=1`, quarantine reason `bounded rerun still zero score`.
- Monitor actions on 2026-05-13: `request_score_rerun=1`, `keep_quarantined=1`.
- Score-fix aggregate: 11 total rows; `real_candidate pending=2`, both `dot_com` and `7_29d`; no real paid or real lead row was exposed.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, or private query logs were written.

## Test Hypothesis

NHS badge embeds are already creating repeat visits to site-profile pages. High-score profile visitors do not need a paid score-fix pitch; they need a free regression monitor. Low-score profile visitors should still see score-fix remediation.

## Proposed UX

1. High-score profile path, score at or above 95:

   Show a compact owner CTA near the badge/embed block:

   `Keep this score from regressing. Get a free alert if llms.txt, OpenAPI, MCP, or another agent-readiness signal disappears.`

   Primary action: `/monitor?domain={host}`.

   Secondary action: copy badge/embed.

2. Mid-score profile path, score 70-94:

   Show both actions without implying a paid bypass:

   - Free monitor: alert on regressions.
   - Score-fix: implementation help for missing public signals.

3. Low-score profile path, score below 70:

   Keep score-fix remediation primary and monitor secondary:

   - Primary: `/fix/{host}`.
   - Secondary: `/monitor?domain={host}`.

4. Badge SVG click-through:

   Preserve current badge URL behavior. The conversion point should be the destination profile page, not the badge image itself.

## Measurement

Use existing aggregate analytics only:

- `/monitor?domain=` visits from site-profile pages.
- `/api/v1/monitor/register` registrations.
- `/fix/{host}` visits from site-profile pages.
- `/site/{host}` visits for badge-heavy domains.
- Score-fix checkout starts, without committing raw checkout URLs.

Success for this first test is small: at least one new monitor registration or a measurable increase in monitor-page visits from site-profile pages, without reducing score-fix visits for low-score pages.

## Copy Guardrails

- Do not claim xquik.com is a customer, endorsement, paid lead, or private demand signal.
- Do not sell rank placement.
- Do not imply a score-methodology bypass.
- Do not imply payment is useful when a site already scores 95+.
- Keep the high-score path about preserving public signals, not "optimizing ranking."

## Implementation Handoff

Queue a later product worker to add the profile-page CTA split and verify it on:

- High-score profile: `https://nothumansearch.ai/site/xquik.com`.
- Mid-score profile: `https://nothumansearch.ai/site/bernstein.run`.
- Low-score profile: choose a public low-score indexed site from `/api/v1/top` or private admin read path without exposing raw private rows.

Verification should include live page checks only; no broad recrawl, no outreach, no public posting.
