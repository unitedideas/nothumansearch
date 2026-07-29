# API unlimited-plan discovery drift

Date: 2026-07-29
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, recrawl, checkout, or global-queue write was performed. This is a
sanitized product/sales handoff for a later NHS operator.

## Fresh evidence

- Public stats report 4,407 indexed sites, average score 38, with `developer`
  as the largest category.
- Live `/mcp` `tools/list` returns the same 11 tools advertised by
  `/.well-known/mcp.json` and `/llms.txt`.
- `/.well-known/commerce.json`, `/api/v1/catalog`, and
  `GET /api/v1/api-keys/subscribe` now expose one API product:
  `nhs_api_unlimited` at `$9.99/mo`.
- `POST /api/v1/quote` returns the same `$9.99/mo` unlimited product for the
  current id and the legacy starter, pro, and scale ids.
- `/llms.txt` still advertises three plans: Starter `$19/mo`, Pro `$49/mo`,
  and Scale `$199/mo`.
- `/openapi.yaml` still limits API product and plan enums to the legacy
  starter, pro, and scale ids and describes the three-plan subscription
  model.
- Aggregate traffic over 168 hours includes 1,955 reads of
  `/.well-known/commerce.json`, 392 reads of `/api/v1/catalog`, and 391 reads
  each of `/api/v1/quote` and `/api/v1/checkout`. The contradiction is on
  active machine-readable conversion paths, not an unused archive.
- Score-band routing is intact: a 100/100 example reaches the already-meets-
  target handoff, while a 45/100 example reaches the score-fix intake.
- The latest monitor worker completed on 2026-07-27. Aggregate monitor state
  is 5 active and 3 quarantined; quarantined rows remain private/admin-only.
- Aggregate score-fix state is 10 real-candidate pending rows and no exposed
  real paid/lead row. This is private sales-workflow evidence, not a public
  demand or revenue claim.
- `/.well-known/agent-card.json` remains 404, so A2A/Agent Card claims and
  strict Agent Card directory submissions remain blocked.

## Decision

Do not promote NHS API pricing from owner-channel or agent-directory copy
until every public discovery surface names one canonical plan contract. The
live catalog/subscribe/quote contract currently points to the `$9.99/mo`
unlimited plan, while the most readable agent instructions and OpenAPI schema
still point to the retired three-tier contract.

## Product handoff

1. Confirm whether `nhs_api_unlimited` at `$9.99/mo` is canonical.
2. If yes, align `/llms.txt`, `/openapi.yaml`, `/api/v1`,
   `/.well-known/agent.json`, `/.well-known/commerce.json`,
   `/api/v1/catalog`, `GET/POST /api/v1/api-keys/subscribe`, quote, checkout,
   and MCP-facing sales copy to the unlimited-plan id and price.
3. Keep legacy starter/pro/scale ids only as explicit compatibility aliases;
   do not present them as orderable plans.
4. If the three-tier contract is still intended, restore it across catalog,
   subscribe, quote, and checkout instead. Do not leave mixed contracts.
5. Verify the repaired surfaces with read-only GETs plus quotes for current and
   legacy ids. Do not complete checkout from a recurring worker and do not
   record raw checkout or activation URLs.

## Marketing guardrail

Until the handoff closes, API-related briefs may describe search and readiness
capabilities but must omit plan names, prices, quotas, and claims that the
purchase contract is fully synchronized. Do not claim customers, private
demand, completed payments, revenue, price accuracy, x402/ACP/MPP support,
paid ranking, preferred inclusion, A2A support, or score-methodology bypass.
