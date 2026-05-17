# Finance Market-Data Owner Segment

Date: 2026-05-17
Agent: `business-marketer-not-human-search`
Scope: no-submit owner-channel scout artifact for Not Human Search.

## Live Evidence

- Public stats: `total_sites=4175`, `avg_score=35`, `top_category=developer`.
- Public finance category: `count=200`, `avg_score=40`.
- Public top-list source: `GET https://nothumansearch.ai/api/v1/top?category=finance&limit=8`.
- MCP analytics, aggregate 7-day read: `tools/list=137686`, `initialize=18576`, `tools/call=292`.
- Top MCP tool calls: `search_agents=188`, `get_site_details=38`, `verify_mcp=14`, `check_url=12`, `get_stats=12`, `recent_additions=9`, `get_top_sites=8`, `find_mcp_servers=8`.
- Current discovery surfaces checked: `/.well-known/mcp.json=200`, `/.well-known/agent.json=200`, `/.well-known/commerce.json=200`, `/api/v1/catalog=200`, `/llms.txt=200`, `/openapi.yaml=200`.
- Compatibility blocker still present: `/.well-known/agent-card.json=404`.
- Monitor surface checked: `/monitor=200`.
- Score-fix split checked against finance examples: high-score `/fix/terminalfeed.io=200` routes as already-strong/monitor-style copy; partial-score `/fix/debridge.com=200` routes to remediation intake.

## Segment

Finance and market-data owners with public machine-readable surfaces but uneven agent-readiness completeness. The public top list is strong enough for an owner-channel test because it includes a clean score-band split:

| Domain | Score | Missing public readiness surface | Owner-channel route |
|---|---:|---|---|
| `terminalfeed.io` | 100 | none in NHS scoring | Monitor/report/badge proof, not score-fix. |
| `chartlibrary.io` | 100 | none in NHS scoring | Monitor/report/badge proof, not score-fix. |
| `prereason.com` | 100 | none in NHS scoring | Monitor/report/badge proof, not score-fix. |
| `devdrops.run` | 95 | Schema.org | Score result first; remediation only if owner wants complete public metadata. |
| `ticksurfers.com` | 90 | MCP | Score result first; remediation can focus on MCP if owner wants agent-tool discoverability. |
| `razorpay.com` | 85 | MCP, Schema.org | Score result first; remediation can focus on public agent and structured metadata surfaces. |
| `lendtrain.com` | 85 | structured API | Score result first; remediation can focus on public API contract clarity. |
| `debridge.com` | 80 | OpenAPI | Score result first; remediation can focus on public OpenAPI visibility. |

These are public readiness examples only. They are not customers, endorsements, paid leads, private demand, completed payments, revenue proof, certification, investment advice, price-data proof, compliance proof, or proof of data freshness.

## Draft Angle

Market-data and fintech sites are already being evaluated by agents as sources, tools, and API surfaces. The practical owner question is not whether the site has finance content; it is whether an agent can discover the public contract, inspect the terms, call a structured endpoint, and monitor whether those surfaces regress.

NHS can support a narrow owner-channel test:

1. High-score finance owners get a monitor/report/badge proof angle: keep the score visible and catch regressions.
2. Partial-score finance owners get a `/score` first touch: show exactly which public readiness surface is missing before any remediation offer.
3. API-heavy or paid-data owners get the catalog/API-plan angle only when the live buying surface supports the claim.

## Gated Use

Before any external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=finance&limit=8`, `/score`, `/monitor`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, and aggregate `/api/v1/admin/mcp?days=7`.
- Verify the active Foundry/Owl-owned account identity for the selected channel.
- Check `marketing/social-post-ledger.json` if present, `outreach/distribution_log.csv`, and sync-state public-action locks.
- Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.
- Do not claim investment advice, trading performance, price accuracy, data freshness, compliance certification, customer endorsement, private demand, completed payments, revenue, paid ranking placement, preferred inclusion, x402/ACP/MPP support for NHS, A2A support, or score-methodology bypass.
