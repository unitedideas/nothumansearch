# Data Intelligence API Owner Segment - 2026-05-17

Purpose: no-submit owner-channel segment for data intelligence and API providers that need agent-readable source contracts. This is a preparation artifact only; no outreach, account creation, browser work, public posting, deploy, recrawl, or directory submission was performed by this recurring scout.

## Evidence Snapshot

- Public stats: 4,172 indexed sites, average score 35, top category `developer`.
- Public categories: `data=399` sites at average score 32; adjacent public buckets include `developer=1236`, `ai-tools=900`, `finance=199`, `ecommerce=148`, `security=115`, and `health=57`.
- Public data top list checked with `/api/v1/top?category=data&limit=10`.
- Public conversion surfaces checked: `/score` 200, `/monitor` 200, `/fix/nothumansearch.ai` 200, `/fix/dchub.cloud` 200.
- Public discovery and commerce surfaces checked: `/.well-known/mcp.json` 200, `/.well-known/agent.json` 200, `/.well-known/commerce.json` 200, `/api/v1/catalog` 200, `/api/v1/quote` 200, `/api/v1/checkout` GET 400 as expected for a POST endpoint, `/llms.txt` 200, and `/openapi.yaml` 200.
- Compatibility blocker: `/.well-known/agent-card.json` still returns 404, so strict Agent Card and A2A-style directory submissions remain gated.

Aggregate admin signals, sanitized:

- MCP analytics, last 7 days: `tools/list=140909`, `initialize=19110`, `tools/call=308`.
- Top called MCP tools: `search_agents=199`, `get_site_details=41`, `check_url=13`, `verify_mcp=12`, `get_stats=12`, `recent_additions=11`, `get_top_sites=9`, `find_mcp_servers=7`, `list_categories=3`, `submit_site=1`.
- Aggregate traffic, last 7 days: `/.well-known/commerce.json=1589`, `/llms.txt=483`, `/openapi.yaml=436`, `/api/v1/catalog=356`, `/api/v1/checkout=330`, `/api/v1/quote=330`, `/top=122`.
- Aggregate referrers include Google traffic, but no private user identifiers are included here.
- Errors last hour: 0.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, private query logs, or raw buyer data are included here.

## Public Data/API Examples

These are public top-list examples only. Treat them as owner-channel targets or readiness-pattern examples, not customers, endorsements, paid leads, private demand, completed purchases, or proof of market share.

| Domain | Score | Owner-channel angle | Boundary |
| --- | ---: | --- | --- |
| `dchub.cloud` | 100 | Data-center intelligence platform with all readiness signals; high-score owner path is monitor/report/badge proof. | Do not claim data-center coverage accuracy, freshness, customer relationship, or endorsement. |
| `api.contrastcyber.com` | 100 | Security-intelligence API with MCP and OpenAPI; useful for source-contract copy aimed at security data providers. | Do not claim security certification, threat-intel accuracy, or operational reliability. |
| `api.boostedchat.com` | 95 | Travel booking/data API with strong machine-readable surfaces. | Do not claim price accuracy, fare availability, savings, or travel fulfillment quality. |
| `api.theartofservice.com` | 90 | Compliance framework API; readiness gap is primarily AI-friendly robots and schema. | Do not claim compliance certification, legal advice, or framework completeness. |
| `api.headlessoracle.com` | 90 | Market-state API; owner path can focus on monitorable API/MCP readiness. | Do not claim market-data accuracy, trading advice, or uptime. |
| `api.agentry.com` | 90 | Agent-commerce infrastructure with API, MCP, and identity/payment metadata. | Do not claim A2A support for NHS, Lightning/Cashu reliability, seller certification, or partnership. |
| `api.socialintel.dev` | 90 | Pay-per-request influencer-data API; useful for catalog/API readiness angle. | Do not claim lead accuracy, email deliverability, pricing freshness, or private demand. |
| `blocklens.co` | 90 | Crypto/on-chain analytics surface with OpenAPI and structured API signals. | Do not claim investment advice, data freshness, price accuracy, or trading performance. |
| `api.meacheal.ai` | 85 | Apparel supply-chain data API with OpenAPI/API surfaces. | Do not claim supply-chain coverage accuracy or commercial endorsement. |
| `meetlark.ai` | 85 | Scheduling/data API for agent workflows. | Do not claim calendar integration reliability, availability, or productivity outcomes. |

## Segment Read

The data category is a better owner-channel segment than a generic "data APIs" post. It spans data-center intelligence, security intelligence, travel booking, compliance frameworks, market-state verification, agent commerce, influencer data, crypto analytics, apparel supply-chain data, and scheduling APIs.

Safe use:

1. High-score data/API owners: route to `/site/{domain}`, badge proof, and free `/monitor` so they can catch future readiness drift.
2. Partial-score data/API owners: route to `/score` first; use `/fix/{host}` only when missing public agent-readiness signals justify remediation.
3. Security, compliance, finance, travel, and health-adjacent data owners: keep domain-specific claim boundaries explicit.
4. Agent-commerce data owners: mention catalog/quote/checkout readiness only as public metadata, not completed payment proof.

Unsafe use:

- Broad claims that NHS validates data quality, freshness, pricing, travel fulfillment, compliance, security, market accuracy, investment outcomes, or API uptime.
- Claims of private demand, customer endorsement, revenue, completed purchases, paid placement, preferred inclusion, or score-methodology bypass.
- A2A support claims while `/.well-known/agent-card.json` returns 404.

## Draft Operator Copy

`Data APIs are where agent-readiness stops being cosmetic. If an autonomous agent needs a source, it needs a public contract it can inspect before scraping: llms.txt, OpenAPI, structured API responses, MCP where relevant, robots policy, plugin metadata, and schema.`

`Not Human Search ranks these surfaces by readiness. High-score owners can use the profile and monitor as proof; partial-score owners can use the score page to see what is missing before any remediation offer.`

Proof links:

- `https://nothumansearch.ai/top?category=data`
- `https://nothumansearch.ai/score`
- `https://nothumansearch.ai/monitor`
- `https://nothumansearch.ai/.well-known/mcp.json`
- `https://nothumansearch.ai/api/v1/catalog`
- `https://nothumansearch.ai/llms.txt`

## Publication Guard

Before any external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=data&limit=10`, representative `/site/{host}` profiles, `/score`, `/monitor`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, and aggregate `/api/v1/admin/mcp?days=7`.
- Verify the active Foundry/Owl-owned account identity for the selected channel.
- Check `marketing/social-post-ledger.json` if present, `outreach/distribution_log.csv`, and sync-state public-action locks.
- Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.

## Do Not Claim

- Private demand, customers, endorsements, paid leads, completed payments, revenue, data quality, data freshness, price accuracy, travel fulfillment, compliance certification, legal advice, security certification, threat-intel accuracy, investment advice, trading performance, API uptime, seller certification, x402/ACP/MPP support for NHS, paid ranking placement, preferred inclusion, A2A support, or score-methodology bypass.
