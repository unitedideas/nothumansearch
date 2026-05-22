# Commerce Manifest and API Plan Conversion Refresh

Run: 2026-05-22 business-marketer-not-human-search

## Signal

The commerce and catalog surfaces are still material discovery paths in the 168-hour aggregate:

- `/.well-known/commerce.json`: 1,509 requests
- `/api/v1/catalog`: 325 requests
- `/api/v1/quote`: 295 requests
- `/api/v1/checkout`: 295 requests
- `/api/v1/search`: 176 requests
- `/api/v1/submit`: 148 requests

Live surfaces checked in this run:

- `/api/v1/stats`: 4,171 total sites, average score 35, top category developer
- `/.well-known/mcp.json`: 200
- `/.well-known/agent.json`: 200
- `/.well-known/agent-card.json`: 404
- `/.well-known/commerce.json`: 200
- `/api/v1/catalog`: 200
- `/score`: 200
- `/monitor`: 200
- `/report`: 200

Aggregate MCP analytics also show programmatic use:

- `tools/list`: 169,385 calls
- `tools/call`: 181 calls
- `search_agents`: 84 calls
- `get_site_details`: 30 calls
- `check_url`: 20 calls
- `get_stats`: 19 calls

## Product-Safe Angle

NHS has two buyer paths exposed to agents:

1. Site owners can buy the done-for-you GEO uplift only after a concrete readiness gap is visible.
2. Programmatic users can buy API plans when anonymous REST/MCP usage is not enough.

The conversion test should keep those paths separate:

- Commerce/catalog readers with owner intent go to `/score`, then `/monitor` or score-fix by score band.
- API-heavy readers go to API-key plans, with the docs and free surfaces kept useful first.
- High-score site profiles go to free monitor/report/badge proof, not paid remediation.
- Partial-score profiles go to `/score` before any score-fix ask.

## Draft Test

Add or queue a product-safe handoff from commerce/catalog discovery paths:

1. Catalog product cards: make the API plans and score-fix service easy to compare without implying a completed checkout.
2. Commerce manifest adjacent docs: clarify supported modes, unsupported ACP/x402/MPP modes, and the difference between score-fix service and API subscriptions.
3. Score-band routing: high-score domains see monitor/report/badge proof; partial-score domains see missing-surface repair next steps.
4. API quota path: users who hit quota should land on the same plan metadata as `/api/v1/catalog`.

## Boundaries

Do not claim completed payments, revenue, customer demand, private buyer intent, endorsement, badge-install consent, A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/MPP support, paid ranking placement, preferred inclusion, or score-methodology bypass.

Do not publish raw checkout URLs, payment identifiers, buyer emails, private query logs, raw user agents, or private monitor/score-fix rows.

## Next Gated Action

Prepare one product-handoff or owner-channel test from this packet after refreshing the live catalog, commerce manifest, score/monitor/report routes, high-score and partial-score `/fix/{host}` routes, aggregate admin traffic, and aggregate MCP analytics.
