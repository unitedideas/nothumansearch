# NHS marketing scout segment - 2026-05-13T02:08Z

Automation: `business-marketer-not-human-search`

## Boundary

No outreach was sent. No posts, browser/Computer Use, account creation, deploys, full recrawls, product-code edits, public submissions, or QLimit/global-queue writes were performed.

## Live Surface Checks

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4238`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. Public category counts observed: developer 1300, ai-tools 892, data 403, finance 201, productivity 171, ecommerce 149, communication 118, security 116, health 54, jobs 26, education 21, news 11. Audit-only buckets remain `other` and `spam`.
- `https://nothumansearch.ai/api/v1`: advertises search, site, stats, categories, top, monitor, score check, commerce, and API-key subscription endpoints.
- `https://nothumansearch.ai/.well-known/mcp.json`: advertises 11 tools and public-category language matching the current taxonomy split.
- `/monitor`, `/score`, `/install`, `/api/v1/api-keys/subscribe`, and `/fix/nothumansearch.ai` returned HTTP 200.
- `/api/v1/api-keys/subscribe` now returns readable plan metadata for Starter, Pro, and Scale via GET.

## Fresh Finding

`/fix/nothumansearch.ai` returns a paid score-fix page even though the target domain already scores 100/100:

- Page title: `Fix the agent-readiness score for nothumansearch.ai`.
- Visible state: `Currently: 100` and `target: 95+`.
- Form accepts email/repo URL and presents the 72-hour score-fix offer.

The page is `noindex`, so this is not an SEO issue. It is a conversion/trust issue for owner-channel flows: if a score-100 owner follows or guesses `/fix/{host}`, NHS still presents a paid remediation offer that the site does not need. This should be gated or reframed for high-score sites before score-fix links are pushed harder.

Suggested behavior for a later product worker:

- For scores at or above the target threshold, make `/fix/{host}` an audit/monitor handoff instead of a checkout/intake page.
- Primary CTA: monitor this score for regressions.
- Secondary CTA: request an audit only if the owner wants a maintained readiness bundle or an implementation review.
- Keep score-fix as remediation for missing public agent-readiness signals, not paid placement or rank bypass.

## Duplicate And Channel Checks

- Existing marketer inbox rows already cover stale public drafts, score-results handoffs, monitor conversion, API-key handoff, MCP registry copy drift, `/report` metadata drift, and vertical owner-channel briefs.
- No existing dedicated row covered high-score `/fix/{host}` gating.
- `outreach/distribution_log.csv` remains saturated with broad MCP/API/GEO directory submissions; no new directory packet was queued.
- No public action was taken, so no public-action lock was claimed.

## Appended Intake Rows

Rows appended to `ops/sweeper/marketer-inbox.jsonl`:

- `Gate score-fix intake for domains already at or above the target score.`
