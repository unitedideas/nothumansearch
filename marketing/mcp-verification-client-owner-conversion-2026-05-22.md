# MCP Verification Client Owner Conversion - 2026-05-22

Automation: `business-marketer-not-human-search`

Scope: no public action, no outreach, no browser, no deploy. This packet is a gated owner-conversion artifact for a later execution worker.

## Fresh Evidence

- Public stats: `4,171` indexed sites, average score `35`, top category `developer`.
- Public categories: `developer=1,228`, `ai-tools=897`, `other=777`, `data=402`, `finance=194`, `productivity=174`, `ecommerce=148`, `communication=118`, `security=114`, `health=58`, `jobs=27`, `education=21`, `news=12`.
- Live discovery surfaces:
  - `/llms.txt`: HTTP 200.
  - `/.well-known/mcp.json`: HTTP 200.
  - `/.well-known/agent.json`: HTTP 200.
  - `/.well-known/agent-card.json`: HTTP 404.
  - `/.well-known/commerce.json`: HTTP 200.
  - `/api/v1/catalog`: HTTP 200.
  - `/openapi.yaml`: HTTP 200.
  - `/monitor`, `/report`, `/score`: HTTP 200.
- Aggregate MCP analytics, last 7 days:
  - `tools/list=166,692`, `initialize=25,635`, `tools/call=233`.
  - tool calls include `search_agents=128`, `get_site_details=28`, `get_stats=19`, `check_url=18`, `get_top_sites=10`, `verify_mcp=7`, `find_mcp_servers=7`, `recent_additions=7`, `list_categories=5`, `submit_site=4`.
  - visible aggregate client families include Cherry Studio, Claude Code, `MCP-Catalog-Bot`, `MCPScoringEngine`, and `mcp-verify`.
- Aggregate traffic, last 168 hours:
  - root `/` = `3,419`.
  - `/badge/xquik.com.svg` = `2,290`; `/site/xquik.com` = `782`.
  - `/.well-known/commerce.json` = `1,512`.
  - `/.well-known/ai-plugin.json` = `699`; `/llms.txt` = `452`; `/openapi.yaml` = `422`.
  - `/api/v1/catalog` = `326`; `/api/v1/checkout` = `297`; `/api/v1/quote` = `297`.
  - `/api/v1/search` = `175`; `/api/v1/submit` = `148`; `/top` = `96`.
- Score-band smoke:
  - `xquik.com`: public profile `100/100`; `/fix/xquik.com` routes to already-meets-target monitor handoff.
  - `chainray.online`: public profile `100/100`; `/fix/chainray.online` routes to already-meets-target monitor handoff.
  - `aidevboard.com`: Foundry-owned dogfood profile `100/100`; `/fix/aidevboard.com` routes to already-meets-target monitor handoff.
  - `manifest.ly`: public profile `65/100`; `/fix/manifest.ly` shows paid remediation intake.

## Segment

This run should target MCP verification-client and scoring-bot users, not another broad directory scan. The useful pattern is:

1. verification/scoring clients inspect NHS discovery surfaces and call `check_url` or `verify_mcp`;
2. high-score profile visitors can be routed to free monitor/report/badge proof;
3. partial-score profile visitors can be routed to `/score` first, then score-fix only when the missing public readiness surfaces are visible.

## Draft Positioning

NHS is a verification surface for agent-readiness checks, not a claim engine.

For MCP client and directory maintainers:

> Not Human Search exposes a streamable HTTP MCP server plus REST/OpenAPI surfaces for checking whether a site has agent-readable discovery metadata. It can help a client or catalog verify public readiness signals before showing a tool or API to an agent.

For site owners:

> If the score is already high, the next step is monitor/report/badge proof. If the score is partial, start with the public score page and remediate missing machine-readable files before paying for help.

## Candidate Execution Tests

1. Add a no-send checklist for MCP verification-client onboarding copy: install string, `tools/list`, one `verify_mcp` example, one `check_url` example, and `/monitor` handoff.
2. Prepare a gated owner-channel post for MCP directory maintainers and client builders, with the A2A blocker named explicitly because `/.well-known/agent-card.json` still returns 404.
3. Prepare a product handoff to route successful `verify_mcp`/`check_url` users toward `/monitor` before they leave the MCP flow.

## Guardrails

- Do not claim Cherry Studio, Claude Code, MCP Catalog, MCPScoringEngine, `mcp-verify`, Xquik, ChainRay, Manifestly, or any listed domain is a customer, partner, endorsement, paid lead, monitor registration, badge-install consent, private demand, completed payment, revenue source, certification proof, or vendor approval.
- Do not claim A2A support while `/.well-known/agent-card.json` is 404.
- Do not claim x402, ACP, MPP, or completed-payment support from traffic to catalog/quote/checkout routes.
- Do not sell ranking placement, preferred inclusion, or score-methodology bypass.
- Before external use, refresh public stats, discovery surfaces, aggregate MCP analytics, aggregate traffic, representative high-score and partial-score `/site/{host}` pages, and high-score plus partial-score `/fix/{host}` behavior.
