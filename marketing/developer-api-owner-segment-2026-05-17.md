# Developer/API Owner Segment - 2026-05-17

Purpose: no-submit owner-channel segment for developer/API services that already expose strong agent-readiness signals. This is a preparation artifact only; no outreach, account creation, browser work, public posting, or directory submission was performed by this recurring scout.

## Evidence Snapshot

- Public stats: 4,175 indexed sites, average score 35, top category developer.
- Public categories: developer 1,228 sites at average score 34; ai-tools 902 at 40; data 403 at 32.
- Public developer top list checked with `/api/v1/top?category=developer&limit=8`.
- Aggregate MCP analytics, last 7 days: `tools/list` 137,330, `initialize` 18,581, `tools/call` 292.
- Aggregate MCP tool calls, last 7 days: `search_agents` 188, `get_site_details` 38, `verify_mcp` 14, `check_url` 12, `get_stats` 12, `recent_additions` 9, `get_top_sites` 8, `find_mcp_servers` 8, `list_categories` 2, `submit_site` 1.
- Aggregate traffic, last 7 days: `/.well-known/commerce.json` 1,563, `/api/v1/catalog` 350, `/api/v1/quote` 325, `/api/v1/checkout` 325, `/.well-known/mcp.json` 86, `/api/v1` 83.
- Discovery compatibility check: `/.well-known/agent.json`, `/.well-known/commerce.json`, and `/api/v1/catalog` returned 200; `/.well-known/agent-card.json` returned 404.
- Duplicate/social blocker: repo-local `marketing/social-post-ledger.json` is still absent, so any external post must use the shared sync-state public-action locks and the channel operator duplicate path before publication.

## Public Candidate Pattern

Use the public developer top list as an owner-channel segment, not as a market-share claim. The candidates are public examples or possible owner targets only; they are not customers, endorsements, paid leads, or private demand proof.

Current public examples from the top developer list:

| Domain | Score | Owner-channel angle | Boundary |
| --- | ---: | --- | --- |
| `agentprobe.fly.dev` | 100 | Foundry dogfood reference for full readiness and agent-commerce surfaces. | Label as Foundry-owned dogfood; do not use as third-party proof. |
| `xquik.com` | 100 | High-score owner path: monitor/report/badge proof, plus agent-commerce catalog readiness. | Do not imply endorsement, customer relationship, or revenue. |
| `mcp.depscope.dev` | 100 | Package intelligence/API owner path: preserve MCP/OpenAPI/API visibility and monitor drift. | Do not claim package-health accuracy or certification. |
| `deadends.dev` | 100 | Developer knowledge-base/API path: monitor public agent contracts and use badge proof. | Do not claim content quality certification. |
| `agentdomainsearch.com` | 100 | Agent-commerce/developer-service path: catalog, quote, checkout metadata and unsupported-rail boundaries. | Do not claim x402 support for NHS; only the profiled site text mentions x402. |
| `blackveilsecurity.com` | 100 | Security/developer scanner path: monitor readiness, OpenAPI/MCP availability, and machine-readable offer metadata. | Do not claim security certification, privacy compliance, or uptime. |
| `agentndx.ai` | 100 | Directory/index owner path: keep discovery manifests, MCP, OpenAPI, and catalog metadata synchronized. | Do not imply partnership or listing reciprocity. |
| `entia.systems` | 100 | Business identity/developer API path: monitor agent-readable identity metadata. | Do not claim identity verification authority or legal compliance. |

## Draft Channel Angle

Developer tools are already using machine-readable surfaces, but the owner-side conversion path should stay practical:

1. High-score developer/API owners: point to `/site/{domain}`, badge proof, and free `/monitor` so they can catch future drift.
2. Partial-score developer/API owners: point to `/score` first, then `/fix/{host}` only when missing public agent-readiness signals justify remediation.
3. Agent-commerce or API-plan owners: point to catalog/quote/checkout readiness and explicit unsupported-rail metadata; do not claim completed payments or demand.

## Next Gated Action

Prepare one developer/API owner-channel touch or post using this segment after refreshing:

- `/api/v1/stats`
- `/api/v1/categories`
- `/api/v1/top?category=developer&limit=8`
- representative `/site/{host}` profiles
- `/score`
- `/monitor`
- `/.well-known/mcp.json`
- `/.well-known/agent.json`
- `/.well-known/agent-card.json`
- `/.well-known/commerce.json`
- `/api/v1/catalog`
- `/api/v1/quote`
- `/api/v1/checkout`
- aggregate `/api/v1/admin/mcp?days=7`

Before external use, verify the active Foundry/Owl-owned account identity, check duplicate locks and ledgers, avoid `modelcontextprotocol/*` and `punkpeye/*` from `unitedideas`, and record the resulting public URL or exact blocker.

## Do Not Claim

- Private demand, customer endorsement, paid leads, completed payments, or revenue.
- Seller certification, package quality, security certification, legal identity certification, uptime, pricing accuracy, data freshness, x402/ACP/MPP support for NHS, paid ranking placement, preferred inclusion, or score-methodology bypass.
- A2A support until `/.well-known/agent-card.json` or target-directory documentation proves compatibility.
