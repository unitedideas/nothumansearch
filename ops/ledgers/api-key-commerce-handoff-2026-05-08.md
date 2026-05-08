# NHS API Key Commerce Handoff

Date: 2026-05-08
Automation: business-agent-not-human-search

## Action

Completed the API-key checkout handoff repair.

- `GET /api/v1/api-keys/subscribe` now returns starter, pro, and scale plan metadata plus the POST checkout contract.
- Quota-exceeded REST and MCP responses now include the useful plan URL, POST method, and required fields.
- `/api/v1/catalog` and `/.well-known/commerce.json` now expose the API subscription products alongside the score-fix service.
- `/api/v1/quote` now returns deterministic quotes for `nhs_api_starter`, `nhs_api_pro`, and `nhs_api_scale`.
- `/api/v1`, `/llms.txt`, and `/openapi.yaml` now describe the paid API-key flow and no longer imply GET `/api/v1/check` performs checks.

No public post, directory submission, customer message, or browser action was taken.

## Proof

- `GOCACHE=/private/tmp/nhs-go-cache go test ./...`
- `fly deploy --remote-only`
- `curl -fsS https://nothumansearch.ai/health`
- `curl -fsS https://nothumansearch.ai/api/v1/api-keys/subscribe`
- `curl -fsS https://nothumansearch.ai/api/v1/catalog`
- `curl -fsS -X POST https://nothumansearch.ai/api/v1/quote -H 'Content-Type: application/json' -d '{"product_id":"nhs_api_pro"}'`
- Live quota-exceeded probes for `/api/v1/search?q=api` and `/api/v1/site/nothumansearch.ai` returned HTTP 402 with `subscribe_url=https://nothumansearch.ai/api/v1/api-keys/subscribe` and `subscribe_method=POST`.
- Live signed Stripe webhook smoke returned HTTP 200.

## Result

Agents that hit the anonymous quota now receive a reachable plan document instead of a POST-only dead end, and agent-commerce clients can discover API subscriptions from the catalog and quote surfaces.
