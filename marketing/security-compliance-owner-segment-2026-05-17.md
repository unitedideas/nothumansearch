# Security Compliance Owner Segment

Date: 2026-05-17
Agent: `business-marketer-not-human-search`
Scope: no-submit owner-channel scout artifact for Not Human Search.

## Live Evidence

- Public stats: `total_sites=4175`, `avg_score=35`, `top_category=developer`.
- Public security category: `count=115`, `avg_score=38`.
- Public top-list source: `GET https://nothumansearch.ai/api/v1/top?category=security&limit=8`.
- MCP analytics, aggregate 7-day read: `tools/list=138113`, `initialize=18578`, `tools/call=288`.
- Top MCP tool calls: `search_agents=184`, `get_site_details=38`, `verify_mcp=14`, `check_url=12`, `get_stats=12`, `recent_additions=9`, `get_top_sites=8`, `find_mcp_servers=8`.
- Current discovery surfaces checked: `/.well-known/agent.json=200`, `/.well-known/commerce.json=200`, `/api/v1/catalog=200`, `/score=200`, `/monitor=200`, high-score `/fix/nothumansearch.ai=200`, partial-score `/fix/resend.com=200`.
- Compatibility blocker still present: `/.well-known/agent-card.json=404`.
- Aggregate traffic, 7-day read: `/.well-known/commerce.json=1568`, `/api/v1/catalog=351`, `/api/v1/quote=326`, `/api/v1/checkout=326`, `/top=123`. Badge/profile traffic remains material, led by `/badge/xquik.com.svg=1260` and `/site/xquik.com=285`.

## Segment

Security, compliance, and audit-tool owners with public machine-readable surfaces but uneven completeness. The public top list is useful for a narrow owner-channel test because the score gaps map cleanly to owner-side repairs:

| Domain | Score | Missing public readiness surface | Owner-channel route |
|---|---:|---|---|
| `feedoracle.io` | 100 | none in NHS scoring | Monitor/report/badge proof, not score-fix. |
| `agent-module.dev` | 95 | Schema.org | Score result first; remediation only if owner wants complete public metadata. |
| `tickerr.ai` | 85 | structured API | Score result first; remediation can focus on a stable public API contract. |
| `ansvar.eu` | 85 | structured API | Score result first; remediation can focus on a stable public API contract. |
| `rnwy.com` | 80 | OpenAPI | Score result first; remediation can focus on public OpenAPI visibility. |
| `easysend.co` | 80 | AI plugin | Score result first; remediation can focus on agent discovery metadata. |
| `qnsp.cuilabs.io` | 70 | OpenAPI, MCP | `/score` first; remediation can focus on OpenAPI plus agent-tool discoverability. |
| `hefestoai.narapallc.com` | 70 | AI plugin, MCP | `/score` first; remediation can focus on agent discovery metadata plus MCP. |

These are public readiness examples only. They are not customers, endorsements, paid leads, private demand, completed payments, revenue proof, certification, compliance proof, security proof, uptime proof, or proof of data freshness.

## Draft Angle

Security and compliance products already sell trust, evidence, and auditability. The agent-facing version is simpler: can an AI system discover the product contract, inspect the public API shape, find supported tool endpoints, and monitor whether those surfaces regress?

NHS can support a narrow owner-channel test:

1. High-score security owners get the monitor/report/badge proof angle: keep the agent-readiness score visible and catch regressions.
2. Partial-score owners get a `/score` first touch: show the exact missing public surface before any remediation offer.
3. API-heavy security products get the catalog/API-plan angle only when the live buying surface supports the claim.

## Gated Use

Before any external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=security&limit=8`, `/score`, `/monitor`, representative `/site/{host}` profiles, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, and aggregate `/api/v1/admin/mcp?days=7`.
- Verify the active Foundry/Owl-owned account identity for the selected channel.
- Check `marketing/social-post-ledger.json` if present, `outreach/distribution_log.csv`, and sync-state public-action locks.
- Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.
- Do not claim security certification, compliance certification, privacy compliance, uptime, customer endorsement, private demand, completed payments, revenue, paid ranking placement, preferred inclusion, x402/ACP/MPP support for NHS, A2A support, or score-methodology bypass.
