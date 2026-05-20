# ChainRay high-score profile referrer conversion - 2026-05-20

Automation: `business-marketer-not-human-search`
Scope: business-local marketing scout artifact only. No outreach, public post, account creation, browser action, deploy, product-code edit, or crawl was performed.

## Evidence

- Public stats: `total_sites=4180`, `avg_score=35`, `top_category=developer`.
- Aggregate traffic, 168 hours:
  - `/=3502`
  - `/badge/xquik.com.svg=2097`
  - `/.well-known/commerce.json=1578`
  - `/.well-known/ai-plugin.json=722`
  - `/site/xquik.com=702`
  - `/llms.txt=468`
  - `/openapi.yaml=435`
  - `/api/v1/catalog=340`
  - `/api/v1/checkout=310`
  - `/api/v1/quote=310`
  - `/robots.txt=309`
  - `/badge/aidevboard.com.svg=287`
  - `/badge/8bitconcepts.com.svg=278`
  - `/api/v1/search=171`
  - `/api/v1/submit=154`
  - `/top=98`
  - `/about=98`
  - `/api/v1=92`
  - `/.well-known/mcp.json=90`
- Aggregate referrers, 168 hours:
  - `https://nothumansearch.ai/=2010`
  - `https://google.com=542`
  - `https://nothumansearch.com/=150`
  - `https://www.nothumansearch.ai/=137`
  - `https://www.nothumansearch.com/=129`
  - `https://nothumansearch.ai/score=122`
  - `https://nothumansearch.ai/top=44`
  - `https://nothumansearch.ai/site/chainray.online=37`
  - `https://nothumansearch.ai/mcp-servers=35`
- MCP analytics, 7 days aggregate:
  - `tools/list=160939`
  - `initialize=22019`
  - `tools/call=274`
  - top tool calls: `search_agents=170`, `get_site_details=35`, `check_url=16`, `get_stats=14`, `verify_mcp=8`, `get_top_sites=8`, `recent_additions=8`, `find_mcp_servers=7`, `list_categories=4`, `submit_site=4`
  - visible aggregate client families include Cherry Studio, Claude Code, `MCP-Catalog-Bot`, and `MCPScoringEngine`.
- Public profile checks:
  - `/site/chainray.online` returns HTTP 200 and shows `Agentic Readiness 100/100`.
  - `/fix/chainray.online` returns the high-score handoff: `already meets the NHS score target`.
  - `/site/xquik.com` returns HTTP 200 and shows `Agentic Readiness 100/100`.
- Discovery surface checks:
  - `/.well-known/mcp.json`, `/.well-known/agent.json`, `/api/v1/catalog`, `/api/v1`, `/openapi.yaml`, `/llms.txt`, and `/mcp` return HTTP 200.
  - `/.well-known/agent-card.json` returns HTTP 404, so strict A2A Agent Card claims remain blocked.

## Segment

The fresh signal is a third-party high-score profile path appearing in aggregate referrers: `/site/chainray.online`. This is similar to the earlier Xquik badge/profile loop but without treating the domain as a customer, endorsement, paid lead, monitor registration, or badge-install consent.

The usable owner-conversion angle is:

1. High-score public profiles should route toward free monitor registration, report sharing, and badge/report proof.
2. Badge/profile visitors should be shown what will drift if machine-readable surfaces disappear.
3. Score-fix remains gated to lower-score or partial-score domains after a fresh public score check.
4. API-heavy visitors should stay on catalog/API-key surfaces where relevant.

## Draft Channel Angle

> If a site already scores 100/100, the next useful action is not paid remediation. It is keeping that score from drifting: monitor the machine-readable files, keep the public report link handy, and use the badge/report as proof for agents and buyers.

Use this as a gated owner-channel test for high-score profiles and badge/report proof. It is not evidence that ChainRay, Xquik, or any profiled site installed a badge, became a customer, registered a monitor, endorsed NHS, or paid for anything.

## Guardrails

- Do not claim A2A support while `/.well-known/agent-card.json` returns HTTP 404.
- Do not imply `chainray.online`, `xquik.com`, `aidevboard.com`, `8bitconcepts.com`, or any profiled domain is a customer, endorsement, paid lead, monitor registration, badge-install consent, private demand, completed payment, revenue, paid placement, preferred inclusion, or score-methodology bypass.
- Do not publish raw private query logs, raw emails, payment identifiers, checkout URLs, private monitor rows, private score-fix rows, or API keys.
- Before any public action, refresh `/api/v1/stats`, `/api/v1/categories`, representative `/site/{host}` pages, high-score and partial-score `/fix/{host}` routes, `/score`, `/monitor`, `/report`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/api/v1`, `/openapi.yaml`, `/llms.txt`, aggregate MCP analytics, and aggregate traffic.
- Before public use, verify the active Foundry/Owl-owned account identity, check `marketing/social-post-ledger.json` if present plus `outreach/distribution_log.csv`, and take a sync-state public-action lock.
