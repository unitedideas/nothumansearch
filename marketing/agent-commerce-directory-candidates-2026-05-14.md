# NHS Agent-Commerce Directory Candidate Matrix

Status: prepared, not submitted.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-14T13:08Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a scout artifact for a later gated operator.

## NHS Surface Check

Live public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4170`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/llms.txt`: current count line is `4170+ sites`; MCP tool count is 11.
- `https://nothumansearch.ai/.well-known/mcp.json`: public category language matches the repaired public/audit-only taxonomy split.
- `https://nothumansearch.ai/.well-known/agent.json`: HTTP 200 JSON with API, MCP, commerce, score-fix, and paid API-key capabilities.
- `https://nothumansearch.ai/.well-known/agent-card.json`: HTTP 404. This blocks or weakens A2A directories that auto-import an Agent Card.
- `https://nothumansearch.ai/agent.json`: HTTP 404. The canonical manifest is only under `/.well-known/agent.json`.
- `https://nothumansearch.ai/api/v1/catalog` and `/api/v1/products`: list score-fix plus Starter, Pro, and Scale API subscription products.
- `https://nothumansearch.ai/api/v1/quote`: default score-fix quote returns `$199 one-time`.
- `https://nothumansearch.ai/api/v1/top?category=developer&limit=5`: includes current third-party and Foundry-owned high-score examples; do not use it as pure third-party proof without labeling dogfood.
- `https://nothumansearch.ai/api/v1/top?category=communication&limit=5`: confirms communication category has live high-score candidates for later vertical copy refresh.

No raw users, API keys, checkout URLs, payment identifiers, private query logs, or private admin rows were written.

## Candidate Families

### Agentic Commerce Registry

URL: `https://agenticcommerce.org/`

Fit: strong. The public positioning is a neutral registry for agentic commerce standards, merchant readiness, and playbooks. NHS fits as both a readiness-scoring system and a dogfooded merchant with agent-readable catalog, quote, checkout, and explicit unsupported-rail metadata.

Submission state: no direct submit endpoint was visible in the text surface; likely gated through contact/newsletter/community paths. Requires a later operator to use the contact path or a known account, with public-action lock.

Use this angle:

Not Human Search benchmarks public agent-readiness signals for websites and APIs, and exposes its own seller metadata through commerce, agent, catalog, quote, and checkout endpoints. It is useful as both an ecosystem readiness reference and a dogfood merchant example.

### AgentRolodex

URL: `https://agentrolodex.com/`

Fit: medium-high after an Agent Card alias exists. The directory imports `/.well-known/agent.json` cards for agents and services. NHS is a service agents can use, but the listing should be service/tooling, not a conversational agent.

Current blocker: the site says registration imports agent cards, while NHS only returns 200 at `/.well-known/agent.json`; if the registry specifically expects an A2A-style Agent Card shape or alternate `/.well-known/agent-card.json`, the current manifest may be rejected or under-scored.

Use this angle after compatibility repair:

Not Human Search is an agent-readable service for discovering websites and APIs that expose machine-readable signals. Agents can call it through REST or MCP, then use score/monitor/fix surfaces for site-owner workflows.

### A2A Registry

URL: `https://www.a2a-registry.org/`

Fit: medium after Agent Card compatibility. It is a broader A2A registry with direct submit links, DNS/GitHub identity claims, and agent-card discovery. NHS is not primarily an A2A agent, so the listing should be framed as an agentic discovery service only if the registry accepts services.

Current blocker: sign-up/submission/account gate plus missing `/.well-known/agent-card.json`.

Use this angle only if services are allowed:

NHS gives agents a public discovery endpoint for agent-ready websites, APIs, and MCP servers. It should be categorized as discovery/search infrastructure, not a personal agent.

### Agentry

URL: `https://agentry.com/`

Fit: medium. The site has a visible "List Your Agent" form and supports MCP/A2A fields, identity, reputation, payments, and observability. NHS already has MCP and commerce surfaces, but Agentry's copy is oriented toward deployable AI agents and agent-service vendors.

Current blocker: form/account path and active identity verification. Later operator must ensure the active submitting identity is Foundry/Owl-owned.

Use this angle:

Not Human Search is search/discovery infrastructure for agents, with MCP support and a machine-readable commerce surface. It is not a sales/support chatbot and should not be placed in generic customer-service categories.

### Agent Artifacts API Registry

URL: `https://www.agentartifacts.io/api-registry`

Fit: low-medium as a peer/research target, not necessarily a submission target. It documents a machine-readable product registry, catalog, checkout, entitlements, and delivery model. No public third-party submission path was visible.

Use as comparison copy only:

NHS should keep catalog/quote/checkout metadata concrete and avoid claiming unsupported entitlement/download flows.

### pay.sh and Targe

URLs: `https://pay.sh/`, `https://targe.io/`

Fit: low for immediate submission. These are API-payment marketplaces or rails, and NHS does not currently offer x402/USDC/pay-per-call provider registration. They are relevant only after a real provider listing path exists and payment rail support is truthful.

Current blocker: NHS explicitly marks ACP/x402/MPP unsupported. Do not submit or imply those rails work.

## Execution Guard

Before any external submission or listing edit:

- Refresh `/api/v1/stats`, `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, `/api/v1/quote`, and the chosen directory's submission requirements.
- Check `outreach/distribution_log.csv`, `marketing/social-post-ledger.json` if a social/channel post is involved, and sync-state public-action locks.
- Verify active account identity.
- Take the required sync-state public-action lock.
- Do not use browser/Computer Use or post publicly from a recurring automation worker.
- Do not claim private demand, completed payments, revenue, certification, paid ranking placement, preferred inclusion, ACP/x402 support, or score-methodology bypass.

## Recommended Next Rows

1. Product repair row: add an A2A-compatible Agent Card alias or explicitly document why NHS only supports `/.well-known/agent.json`.
2. Marketing execution row: prepare a gated Agentic Commerce Registry contact/submission packet from `marketing/agent-commerce-directory-packet-2026-05-14.md` plus this candidate matrix.
