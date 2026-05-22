# WeChat/RSS Publisher Source-Readiness Packet - 2026-05-22

Automation: `business-marketer-not-human-search`
Scope: business-local marketing scout artifact only. No outreach, public post, directory submission, account creation, browser action, product-code edit, deploy, checkout completion, or crawl was performed.

## Evidence

- Public stats: `total_sites=4171`, `avg_score=35`, `top_category=developer`.
- Public categories: `developer=1228`, `ai-tools=897`, `other=777`, `data=402`, `finance=194`, `productivity=174`, `ecommerce=148`, `communication=118`, `security=114`, `health=58`, `jobs=27`, `education=21`, `news=12`, `spam=1`.
- Aggregate traffic, 168 hours: `/=3350`, `/badge/xquik.com.svg=2401`, `/.well-known/commerce.json=1506`, `/site/xquik.com=819`, `/.well-known/ai-plugin.json=693`, `/llms.txt=449`, `/openapi.yaml=419`, `/api/v1/catalog=328`, `/robots.txt=301`, `/api/v1/checkout=295`, `/api/v1/quote=295`, `/api/v1/search=180`, `/api/v1/submit=142`, `/about=100`, `/top=94`, `/api/v1=90`, `/.well-known/mcp.json=88`, `/score=75`, `/guide=73`, `/digest=66`, `/newest=64`, `/site/manifest.ly=63`.
- Aggregate referrers, 168 hours: `google.com=542`, canonical and alias referrers remain material, `/score=92`, `nothumansearch.fly.dev=64`, `/top=42`, `/site/chainray.online=38`, `aurelianflo.com=34`, `/mcp=33`, `/mcp-servers=33`, `/submit=28`.
- Aggregate MCP analytics, 7 days: `tools/list=169315`, `initialize=27515`, `tools/call=184`; top called tools include `search_agents=87`, `get_site_details=30`, `check_url=20`, `get_stats=19`, `get_top_sites=6`, `find_mcp_servers=6`, `recent_additions=6`, `list_categories=5`, `submit_site=3`, `verify_mcp=2`.
- Aggregate MCP query themes include Singapore news/housing, WeChat article monitor/RSS feed, local publishers, scanner/document workflows, IoT hardware, model APIs, nutrition/health, and secrets management.
- Public news examples from `/api/v1/top?category=news&limit=12`: `informedclearly.com=70`, `hallucinationherald.com=65`, `biztoc.com=65`, `zadar.tv=55`, `aibtc.news=50`, plus several `45` score publishers.
- Public communication examples from `/api/v1/top?category=communication&limit=8`: `resend.com=75`, `secondsim.co.uk=70`, `mail.misar.io=70`, `postalform.com=65`, `slack.com=60`, `api.slack.com=60`.
- Discovery surfaces checked: `/score`, `/monitor`, `/report`, `/api/v1`, `/api/v1/catalog`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/llms.txt`, `/openapi.yaml`, `/mcp-servers`, `/openapi-apis`, and `/llms-txt-sites` returned HTTP 200. `/.well-known/agent-card.json` returned HTTP 404, so strict A2A Agent Card claims remain blocked.

## Segment

The fresh scout angle is not generic publisher SEO. It is WeChat/RSS/article-monitor source readiness for agent workflows that need to discover, cite, monitor, or transform publisher and newsletter content through stable machine-readable routes.

The useful owners are:

1. Publisher and newsletter tools with RSS, archive, or API surfaces.
2. WeChat/article-monitor and social-content archiving tools that need clear public source contracts.
3. Local-news and topic-news publishers whose pages should expose `llms.txt`, feeds, OpenAPI/API metadata where applicable, and monitorable profile proof.
4. Communication/API platforms that help agents subscribe, notify, or send updates from monitored content.

## Draft Channel Angle

Agents can read a page, but recurring article monitoring works better when publishers expose stable machine-readable surfaces.

Not Human Search checks whether a site exposes the pieces agents look for before relying on it: `llms.txt`, OpenAPI, structured API responses, MCP, plugin manifests, robots policy, Schema.org, feeds, and public score/report pages. High-score owners can monitor drift and show proof. Partial-score owners can run a public score check before deciding whether remediation is worth doing.

## Gated Test

Prepare exactly one gated owner-channel touch, post, or product-handoff test for publisher tooling, WeChat article monitoring, RSS/feed readers, newsletter infrastructure, local-news publishers, or content-monitoring APIs.

Before external use, refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=news&limit=12`, `/api/v1/top?category=communication&limit=8`, `/feed.xml`, `/digest`, `/newest`, `/score`, `/monitor`, `/report`, representative `/site/{host}` pages, high-score and partial-score `/fix/{host}` routes, `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, aggregate `/api/v1/admin/mcp?days=7`, and aggregate `/api/v1/admin/traffic?hours=168`.

Verify the active Foundry/Owl-owned account identity for the selected channel, check `marketing/social-post-ledger.json` if present plus `outreach/distribution_log.csv` and sync-state public-action locks, and avoid `modelcontextprotocol/*` plus `punkpeye/*` surfaces from `unitedideas`.

## Guardrails

Do not imply listed publishers, newsletter tools, WeChat/article-monitor tools, communication platforms, searched domains, feed readers, or referrers are customers, endorsements, partners, paid leads, monitor registrations, badge-install consent, private demand, completed payments, revenue, crawler compliance, SEO lift, editorial accuracy, article freshness, translation quality, RSS completeness, WeChat access reliability, deliverability, privacy compliance, legal compliance, uptime proof, A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/MPP support for NHS, paid ranking placement, preferred inclusion, or score-methodology bypass.
