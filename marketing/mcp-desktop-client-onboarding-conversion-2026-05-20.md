# MCP Desktop Client Onboarding Conversion - 2026-05-20

Scope: one Not Human Search marketing scout segment. Artifact only; no outreach, public post, directory submission, account creation, browser action, product-code edit, deploy, or crawl was performed.

## Evidence

Live public surfaces checked during the run:

- `/api/v1/stats`: `total_sites=4175`, `avg_score=35`, `top_category=developer`.
- `/api/v1/categories`: `developer=1231`, `ai-tools=904`, `other=768`, `data=400`, `finance=196`, `productivity=173`, `ecommerce=148`, `communication=120`, `security=115`.
- `/mcp`: HTTP 200.
- `/.well-known/mcp.json`: HTTP 200; advertises 11 MCP tools.
- `/.well-known/agent.json`: HTTP 200.
- `/.well-known/agent-card.json`: HTTP 404; strict A2A Agent Card claims remain blocked.
- `/.well-known/commerce.json`: HTTP 200.
- `/.well-known/ai-plugin.json`: HTTP 200.
- `/api/v1`: HTTP 200.
- `/api/v1/catalog`: HTTP 200; lists score-fix plus Starter, Pro, and Scale API subscriptions.
- `/llms.txt`: HTTP 200.
- `/openapi.yaml`: HTTP 200.
- `/score`, `/monitor`, and `/report`: HTTP 200.

Aggregate MCP analytics for the last 7 days:

- `tools/list=159472`, `initialize=21170`, `tools/call=282`.
- Top tool calls: `search_agents=175`, `get_site_details=38`, `check_url=16`, `get_stats=16`, `verify_mcp=8`, `get_top_sites=8`, `recent_additions=8`, `find_mcp_servers=5`, `list_categories=4`, `submit_site=4`.
- Desktop/client signals in aggregate user agents: `CherryStudio/1.9.6` with 523 calls, Claude Code CLI/SDK variants with multiple versioned callers, plus `node`, `python-httpx`, and `python-requests` MCP clients.
- Directory-bot signals remain present: `MCP-Catalog-Bot/1.0=102`, `MCPScoringEngine/1.0=95`, `AgentFinderBot/0.3=68`, and `mcp-verify/0.1.0=65`.

Aggregate traffic for the last 168 hours:

- `/=3463`
- `/badge/xquik.com.svg=2042`
- `/.well-known/commerce.json=1609`
- `/.well-known/ai-plugin.json=733`
- `/site/xquik.com=675`
- `/llms.txt=477`
- `/openapi.yaml=439`
- `/api/v1/catalog=347`
- `/api/v1/quote=316`
- `/api/v1/checkout=316`
- `/robots.txt=306`
- `/api/v1/submit=147`
- `/api/v1/search=136`
- `/top=103`
- `/about=96`
- `/.well-known/mcp.json=91`
- `/api/v1=91`

## Segment

NHS is being used by desktop MCP clients, not just bots and server-side scripts. The scout action should make the install/use path easier for client users and route them toward owner actions without overstating demand.

Best-fit conversion test:

1. MCP client onboarding packet for Cherry Studio, Claude Code, and generic streamable-HTTP MCP clients.
2. Keep the first action as install/search, then route:
   - site owners to `/score` and `/monitor`;
   - high-score profile visitors to report/badge proof;
   - partial-score owners to `/score` before score-fix remediation;
   - API-heavy callers to API-key plans from `/api/v1/catalog`.

## Draft Packet

Short copy:

> Not Human Search is an MCP server for finding agent-ready websites, APIs, and tools. It scores public sites by machine-readable readiness signals and exposes search, score checks, MCP verification, top sites, recent additions, and monitor registration over streamable HTTP.

Install:

```bash
claude mcp add --transport http nothumansearch https://nothumansearch.ai/mcp
```

Endpoint:

```text
https://nothumansearch.ai/mcp
```

Useful first calls:

- `search_agents`: find agent-ready sites by query and category.
- `get_site_details`: inspect one public domain's readiness profile.
- `check_url`: run a fresh public readiness check.
- `verify_mcp`: verify whether a site exposes a working MCP endpoint.
- `register_monitor`: register a site for free readiness regression monitoring.
- `get_top_sites` and `recent_additions`: browse public discovery lists.

Owner handoff:

- If the site already scores high, register the free monitor and use the report/badge proof.
- If the site is partial-score, run `/score` first and use score-fix only when the public result shows concrete missing readiness surfaces.
- If the caller needs bulk search or API access, use `/api/v1/catalog` for API-key plans.

## Guardrails

- Do not claim Cherry Studio, Claude Code, or any MCP client endorses NHS.
- Do not publish raw user-agent logs or private query logs; aggregate counts only.
- Do not claim A2A support while `/.well-known/agent-card.json` returns HTTP 404.
- Do not claim private demand, customer endorsement, paid leads, monitor registrations, completed payments, revenue, x402/ACP/MPP support, paid ranking placement, preferred inclusion, uptime proof, crawler compliance, or score-methodology bypass.
- Before public use, refresh the live surfaces above, verify active Foundry/Owl-owned account identity, check `marketing/social-post-ledger.json` if present, check `outreach/distribution_log.csv`, and take a sync-state public-action lock.
