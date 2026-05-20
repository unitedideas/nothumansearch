# Homepage/search owner conversion test - 2026-05-20

Automation: `business-marketer-not-human-search`
Scope: business-local marketing scout artifact only. No public post, outreach, account creation, browser action, deploy, full recrawl, or product-code edit.

## Live Evidence

- Public stats: `total_sites=4175`, `avg_score=35`, `top_category=developer`.
- MCP analytics, 7 days aggregate:
  - `tools/list=159288`
  - `initialize=21090`
  - `tools/call=286`
  - top tools: `search_agents=179`, `get_site_details=38`, `check_url=16`, `get_stats=16`, `verify_mcp=8`, `get_top_sites=8`, `recent_additions=8`, `find_mcp_servers=5`, `list_categories=4`, `submit_site=4`
  - discovery/catalog agents visible: `MCP-Catalog-Bot=102`, `MCPScoringEngine=95`, `mcp-verify=70`
- Traffic analytics, 168 hours aggregate:
  - top pages: `/=3459`, `/badge/xquik.com.svg=2018`, `/.well-known/commerce.json=1629`, `/.well-known/ai-plugin.json=742`, `/site/xquik.com=661`, `/llms.txt=483`, `/openapi.yaml=444`, `/api/v1/catalog=351`, `/api/v1/checkout=320`, `/api/v1/quote=320`, `/robots.txt=307`, `/api/v1/submit=147`, `/api/v1/search=126`, `/top=102`, `/about=95`
  - top referrers include `google.com=542`, `nothumansearch.com=151`, `www.nothumansearch.ai=136`, `www.nothumansearch.com=128`, `http://www.nothumansearch.ai=126`, `http://nothumansearch.ai=124`, `/score=122`, `nothumansearch.fly.dev=60`, `www.google.com=60`
- Discovery surfaces checked:
  - `/.well-known/mcp.json` advertises 11 tools.
  - `/llms.txt` advertises 4175+ sites and the same 11 tools.
  - `/.well-known/agent.json` returns 200 with MCP, catalog, quote, checkout, and API subscribe links.
  - `/.well-known/agent-card.json` still returns 404, so A2A/Agent Card claims remain blocked.
- Score-fix routing checked:
  - `/fix/nothumansearch.ai` returns the high-score monitor/report handoff and contains `already meets the target`.
  - `/fix/cohere.com` returns the paid `$199` remediation route.
- Monitor worker evidence:
  - latest scheduled monitor-check completed on 2026-05-18 with one due active monitor, score unchanged at 100.
  - prior quarantine reconciliation remains aggregate-only: one active monitor plus one quarantined zero-score monitor.

## Segment

The homepage and public search surfaces are now stronger owner-conversion evidence than another broad directory scan. Root traffic is materially higher than API documentation and manifest traffic, while `/api/v1/search`, `/top`, `/about`, `/api/v1/submit`, `/score`, badges, and site profiles all show related path activity.

This suggests a conversion test for visitors who arrive through search/homepage paths:

1. Search-first visitors: keep search useful, then route query/result pages to `/score` and `/monitor` where the user is clearly evaluating a site they own.
2. Submit-first visitors: after a successful submit, route to a public score check and free monitor registration instead of only the crawl queue acknowledgement.
3. Profile/badge visitors: route high-score profiles to monitor/report/badge proof; route partial-score profiles to `/score` before remediation.
4. API-heavy visitors: keep docs useful, then route catalog/quote/checkout users to paid API-key plans without implying payments or revenue.

## Draft Channel Angle

Agents are already finding NHS through MCP/catalog bots and humans are entering through the homepage, Google, score, submit, and profile paths. The next owner-facing message should be:

> Your site can be findable to agents without guessing what each crawler wants. Check the public score, register a free monitor, and fix missing machine-readable surfaces only when the score shows a concrete gap.

Use this for owner-channel copy or a product-handoff test. Do not present it as private demand, paid lead volume, or customer proof.

## Guardrails

- Do not claim A2A support while `/.well-known/agent-card.json` is 404.
- Do not claim completed payments, revenue, buyer demand, private monitor registrations, badge-install consent, paid placement, preferred inclusion, or score-methodology bypass.
- Do not expose raw emails, payment identifiers, checkout URLs, private query logs, raw user agents beyond aggregate names, or private monitor/score-fix rows.
- Before public use, refresh `/api/v1/stats`, `/api/v1/categories`, `/`, `/score`, `/monitor`, `/report`, `/api/v1/search`, `/api/v1/submit`, `/api/v1/top`, representative `/site/{host}` pages, high-score and partial-score `/fix/{host}` routes, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, aggregate MCP analytics, and aggregate traffic.
