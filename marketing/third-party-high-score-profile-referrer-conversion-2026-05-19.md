# Third-Party High-Score Profile Referrer Conversion Packet

Date: 2026-05-19
Automation: `business-marketer-not-human-search`

## Scout Signal

Aggregate traffic for the last 168 hours shows public profile and query loops around high-score third-party site profiles:

- `/site/chainray.online`: 34 referrer hits.
- `/?q=chainray.online`: 28 referrer hits.
- `/site/xquik.com`: 495 page hits and 32 referrer hits.
- `/badge/xquik.com.svg`: 1,742 page hits.
- `/score`: 76 page hits.
- `/api/v1/check`: 60 page hits.
- `/monitor`: live at HTTP 200.

This is owner-conversion evidence only. It does not prove customer demand, endorsement, paid leads, monitor registrations, completed payments, revenue, badge-install consent, private intent, paid placement, preferred inclusion, or score-methodology bypass.

## Live Surface Checks

- `/api/v1/stats`: `total_sites=4174`, `avg_score=35`, `top_category=developer`.
- `/api/v1/categories`: developer `1237`, ai-tools `900`, other `765`, data `399`, finance `199`, productivity `173`, ecommerce `149`, communication `119`, security `115`, health `57`, jobs `27`, education `21`, news `12`.
- `/.well-known/mcp.json`: HTTP 200.
- `/.well-known/agent.json`: HTTP 200.
- `/.well-known/commerce.json`: HTTP 200.
- `/.well-known/agent-card.json`: HTTP 404, so strict A2A/Agent Card claims remain blocked.
- `/openapi.yaml`: HTTP 200.
- `/llms.txt`: HTTP 200.
- `/site/chainray.online`: HTTP 200, public profile title says score `100/100`.
- `/fix/chainray.online`: HTTP 200 high-score handoff saying the domain already meets the target.
- `/badge/chainray.online.svg`: HTTP 200 badge SVG with score `100 / 100`.

## Segment

Use high-score third-party profile and badge visitors as the next owner-channel proof loop:

1. Public high-score profiles should lead with free monitor/report/badge proof, not paid remediation.
2. Public profile pages with partial scores should lead to `/score` first, then score-fix only after a current public score proves the gap.
3. Badge/profile loops should make it easy for site owners to add or monitor the badge without implying they are already customers.
4. API-heavy callers arriving through `/api/v1`, `/openapi.yaml`, `/mcp`, catalog, quote, or checkout should be routed toward API-key plans without hiding the free docs.

## Candidate Copy Boundary

Usable phrasing:

> If your profile already scores highly, the next useful step is monitoring. NHS can watch the same public readiness signals and alert when an agent-facing surface regresses.

Avoid:

- Any claim that `chainray.online`, `xquik.com`, or another profiled domain is a customer, endorsement, paid lead, or private signal.
- Any claim of A2A support while `/.well-known/agent-card.json` is 404.
- Any claim of completed payments, revenue, monitor registrations, badge-install consent, paid ranking placement, preferred inclusion, or score-methodology bypass.

## Execution Gate

Before public use or implementation:

1. Refresh `/api/v1/stats`, `/api/v1/categories`, `/score`, `/monitor`, `/report`, representative high-score and partial-score `/site/{host}` pages, high-score and partial-score `/fix/{host}` routes, selected badge SVG routes, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, aggregate `/api/v1/admin/mcp?days=7`, and aggregate `/api/v1/admin/traffic?hours=168`.
2. Verify active Foundry/Owl-owned account identity for any public channel.
3. Check `marketing/social-post-ledger.json` if it exists, `outreach/distribution_log.csv`, and sync-state public-action locks.
4. Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.
