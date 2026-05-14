# NHS agent-commerce bot conversion test - 2026-05-14

## Scope

Marketing scout artifact only. No public submission, outreach, browser action, code edit, deploy, checkout, or account creation happened in this run.

## Fresh evidence

Live public/admin aggregate checks from 2026-05-14:

- `/api/v1/stats`: 4,172 indexed sites, average score 35, top category `developer`.
- `/llms.txt`: advertises 4,172+ sites and 11 MCP tools.
- `/.well-known/mcp.json`: advertises the live MCP endpoint and tool metadata.
- `/.well-known/commerce.json`: HTTP 200, 4,177 bytes.
- `/.well-known/agent.json`: HTTP 200, 1,009 bytes.
- `/api/v1/catalog`: HTTP 200, 3,092 bytes.
- `/api/v1/quote`: receiving material aggregate traffic.
- `/api/v1/checkout`: receiving material aggregate traffic.
- `/.well-known/agent-card.json`: HTTP 404, still not ready for strict A2A-style directory submission.

Aggregate admin traffic, last 336h:

- `/.well-known/commerce.json`: 1,298 requests.
- `/api/v1/catalog`: 292 requests.
- `/api/v1/quote`: 273 requests.
- `/api/v1/checkout`: 273 requests.
- `/.well-known/mcp.json`: 94 requests.
- `/api/v1`: 93 requests.
- `/site/xquik.com`: 169 requests after `/badge/xquik.com.svg` traffic.

MCP analytics, last 7d:

- `tools/list`: 112,608 calls.
- `initialize`: 14,814 calls.
- `tools/call`: 339 calls.
- Top called tools: `search_agents` 198, `get_site_details` 38, `find_mcp_servers` 26, `get_stats` 18, `verify_mcp` 18, `check_url` 15, `get_top_sites` 14.
- Query themes include finance/trading, A2A agent protocol discovery, x402 agent commerce, MCP server lookup, API/tool discovery, and brand-specific company lookups.

## Read

Bots and agent clients are already inspecting NHS as a seller, not only as a search engine. The commerce manifest is now one of the highest-traffic non-root public surfaces, and the catalog/quote/checkout trio has roughly matched aggregate traffic. That is enough to justify an owner-conversion test for commerce-reading agents.

Do not frame this as completed payment, revenue, private demand, customer intent, endorsement, or x402/ACP support. It is route traffic only.

## Test Idea

Add a product-safe conversion bridge from machine-readable commerce/catalog surfaces into the existing owner offers:

1. Score-fix remediation for low-score site owners.
2. Free monitor registration for high-score site owners and badge/profile visitors.
3. Paid API keys for agents hitting quota or repeatedly using search/top-site discovery.

The test should stay machine-readable first:

- In `/api/v1/catalog`, keep product objects explicit about live rails and unsupported rails.
- In quote responses, include the next allowed action and the human-safe boundary.
- In checkout responses, keep Stripe Checkout handoff explicit and do not imply ACP/x402 support until live.
- In public site copy, summarize the machine-readable products without overclaiming demand.

## Guardrails

- Do not complete checkout from recurring workers.
- Do not print or commit raw Stripe Checkout URLs.
- Do not claim private demand, completed payments, revenue, customer endorsement, paid ranking placement, preferred inclusion, ACP/x402 support, or score-methodology bypass.
- Do not submit to A2A directories until `/.well-known/agent-card.json` is repaired or deliberately documented as unsupported.
- Before publication or implementation, refresh `/api/v1/stats`, `/.well-known/commerce.json`, `/.well-known/agent.json`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, and `/.well-known/mcp.json`.

## Acceptance For Later Implementation

- One machine-readable path from catalog/quote/checkout to score-fix, monitor, and API-key offers is visible without browser-only state.
- High-score site owner traffic routes primarily to free monitor registration, not paid remediation.
- Low-score site owner traffic keeps score-fix remediation primary.
- API-heavy agent traffic can discover API plans without a quota dead end.
- Verification uses only aggregate traffic and public routes; no raw emails, monitor rows, payment ids, checkout URLs, or private customer data appear in committed artifacts.
