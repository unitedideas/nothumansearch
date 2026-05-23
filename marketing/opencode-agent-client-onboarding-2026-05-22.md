# OpenCode and Agent-Client Onboarding Packet - 2026-05-22

Automation: `business-marketer-not-human-search`
Scope: business-local marketing scout artifact only. No outreach, public post, directory submission, account creation, browser action, product-code edit, deploy, checkout completion, or crawl was performed.

## Evidence

- Public stats: `total_sites=4171`, `avg_score=35`, `top_category=developer`.
- Public categories: `developer=1228`, `ai-tools=897`, `other=777`, `data=402`, `finance=194`, `productivity=174`, `ecommerce=148`, `communication=118`, `security=114`, `health=58`, `jobs=27`, `education=21`, `news=12`, `spam=1`.
- Discovery surfaces checked: `/score`, `/monitor`, `/report`, `/top`, `/digest`, `/newest`, `/mcp-servers`, `/openapi-apis`, `/llms-txt-sites`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, and `/api/v1/catalog` returned HTTP 200.
- `/.well-known/agent-card.json` returned HTTP 404, so strict A2A Agent Card claims remain blocked.
- Aggregate MCP analytics, 7 days: `tools/list=169401`, `initialize=27557`, `tools/call=196`.
- Aggregate tool calls: `search_agents=99`, `get_site_details=30`, `check_url=20`, `get_stats=19`, `get_top_sites=6`, `find_mcp_servers=6`, `recent_additions=6`, `list_categories=5`, `submit_site=3`, `verify_mcp=2`.
- Aggregate client families include desktop MCP clients, Claude Code variants, OpenCode, MCP catalog/scoring bots, MCP verification clients, and agent-directory bots. This is aggregate family evidence only, not endorsement.
- Aggregate traffic, 168 hours: `/=3328`, `/badge/xquik.com.svg=2411`, `/.well-known/commerce.json=1498`, `/site/xquik.com=831`, `/.well-known/ai-plugin.json=689`, `/llms.txt=449`, `/openapi.yaml=417`, `/api/v1/catalog=328`, `/robots.txt=300`, `/api/v1/quote=293`, `/api/v1/checkout=293`, `/api/v1/search=181`, `/api/v1/submit=142`, `/about=100`, `/top=94`, `/api/v1=89`, `/.well-known/mcp.json=88`, `/score=75`, `/guide=73`, `/digest=66`, `/newest=64`, `/site/manifest.ly=63`.
- Public top developer examples: `agentprobe.fly.dev`, `xquik.com`, `mcp.depscope.dev`, `deadends.dev`, `agentdomainsearch.com`, `blackveilsecurity.com`, `agentndx.ai`, `entia.systems`, `rendoc.dev`, `gptr.dev`, `wearewarp.com`, `mycloudclaw.com` all show as score `100` in the public top list. `agentprobe.fly.dev` is Foundry-owned dogfood and must be labeled before use.
- Score-band smoke: `/fix/xquik.com` routes to an already-meets-target monitor/report handoff; `/fix/manifest.ly` returns paid remediation intake.

## Segment

This segment is not another generic MCP client packet. The fresh angle is multi-client onboarding for agents and desktop MCP clients that are discovering NHS through `tools/list`, then using `search_agents`, `get_site_details`, `check_url`, and verification tools.

Useful followthrough:

1. Agent-client users get a concise install/search path for streamable HTTP MCP.
2. Directory bots get stable discovery links: `/mcp`, `/.well-known/mcp.json`, `/api/v1`, `/openapi.yaml`, `/llms.txt`, `/api/v1/catalog`.
3. Site owners coming from client search results get routed to `/score` first.
4. High-score profiles route to free monitor/report/badge proof.
5. Partial-score profiles route to `/score` and a missing-surface checklist before paid remediation.

## Draft Channel Angle

NHS is useful inside MCP clients because the client can discover agent-ready services without trusting a static list.

The install/search path should stay concrete:

> Add Not Human Search as a streamable HTTP MCP server, search for an agent-ready service, then open the public score/report before relying on it.

For site owners:

> If agents are already finding the profile, the next step depends on the score. High-score sites should monitor and share the report. Partial-score sites should fix the missing machine-readable surfaces before asking agents to rely on them.

## Gated Test

Prepare exactly one gated MCP-client, OpenCode-compatible, agent-directory, or owner-channel test from this packet.

Before external use, refresh `/api/v1/stats`, `/api/v1/categories`, `/mcp`, JSON-RPC `tools/list`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/score`, `/monitor`, `/report`, `/llms.txt`, `/openapi.yaml`, representative high-score and partial-score `/site/{host}` pages, high-score and partial-score `/fix/{host}` routes, aggregate `/api/v1/admin/mcp?days=7`, and aggregate `/api/v1/admin/traffic?hours=168`.

Verify the active Foundry/Owl-owned account identity for the selected channel, check `marketing/social-post-ledger.json` if present plus `outreach/distribution_log.csv` and sync-state public-action locks, and avoid `modelcontextprotocol/*` plus `punkpeye/*` surfaces from `unitedideas`.

## Guardrails

Do not imply OpenCode, Cherry Studio, Claude Code, MCP Catalog, MCPScoringEngine, mcp-verify, agent-directory bots, Xquik, Manifestly, or any listed domain is a customer, partner, endorsement, paid lead, monitor registration, badge-install consent, private demand, completed payment, revenue, certification proof, vendor approval, A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/MPP support for NHS, paid ranking placement, preferred inclusion, crawler compliance, uptime proof, or score-methodology bypass.

Do not publish raw user-agent strings, private query logs, raw checkout URLs, payment identifiers, buyer emails, private monitor rows, or private score-fix rows.
