# NHS API Quota Conversion Brief

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-13T07:08Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, or QLimit/global-queue write was performed. This artifact is a sales/owner-channel brief for a later gated operator. External use still requires active account verification, duplicate-fingerprint checks, and a sync-state public-action lock.

## Live Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4239`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories; largest public buckets are `developer=1300`, `ai-tools=892`, `data=403`, `finance=201`, and `productivity=171`.
- `https://nothumansearch.ai/api/v1/search?q=mcp&limit=1`: HTTP 402 `quota_exceeded`, `used=100`, `limit=100`, with `subscribe_url`, `subscribe_method=POST`, and `subscribe_fields=["email","plan"]`.
- `https://nothumansearch.ai/api/v1/api-keys/subscribe`: HTTP 200 plan metadata for Starter, Pro, and Scale API subscriptions.
- `https://nothumansearch.ai/api/v1/catalog`: exposes the same API subscription plans as agent-readable products.
- `POST /api/v1/api-keys/subscribe` with a Foundry-owned smoke email and `plan=starter`: HTTP 200 with `plan=starter`, `monthly_limit=1000`, `amount_cents=1900`, and redacted `checkout_url` / `activation_url`.

Aggregate admin MCP analytics, last 7 days:

- `tools/list=99917`, `initialize=12763`, `tools/call=430`.
- Tool calls: `search_agents=233`, `get_site_details=56`, `check_url=31`, `find_mcp_servers=29`, `get_stats=23`, `verify_mcp=21`, `get_top_sites=16`, `recent_additions=10`, `submit_site=7`, `list_categories=4`.
- Unknown tool names: none observed in this aggregate read.

No raw checkout URL, API key, customer row, user identifier, or private query log was written.

## Brief Copy

Subject/heading:

`NHS API plans are now agent-readable after the anonymous quota`

Short post:

Not Human Search now exposes a clean paid API handoff after the anonymous quota.

The public REST search path returns a structured `quota_exceeded` response once the free 100-call monthly quota is used. That response names the subscription endpoint, required fields, and allowed plan flow instead of dead-ending an agent.

The buying surface is also machine-readable:

1. `GET /api/v1/api-keys/subscribe` lists the Starter, Pro, and Scale API plans.
2. `POST /api/v1/api-keys/subscribe` returns a Stripe Checkout handoff and activation URL.
3. `/api/v1/catalog` exposes the same plans as agent-readable products.

Current public plans:

- Starter: 1,000 REST/MCP calls per month, `$19/mo`.
- Pro: 10,000 REST/MCP calls per month, `$49/mo`.
- Scale: 100,000 REST/MCP calls per month, `$199/mo`.

Agents can discover tools through `/mcp`, inspect products through `/api/v1/catalog`, and start API-key checkout through `/api/v1/api-keys/subscribe`.

## Owner/Buyer Angle

This is useful for agent builders who need repeatable discovery reads without relying on the anonymous public quota. The sell is not "pay for ranking." The sell is predictable REST/MCP usage after the free quota and an agent-legible purchase path.

## Publication Guard

Before external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/catalog`, and `/api/v1/api-keys/subscribe`.
- Smoke the POST contract with a Foundry-owned test email, redact checkout and activation URLs from any notes, and do not complete checkout from the recurring worker.
- Check the shared social ledger for duplicate fingerprints.
- Verify the active account identity for the selected channel.
- Take a sync-state public-action lock.
- Do not claim private demand, revenue, conversion, paid ranking placement, or score-methodology bypass.
