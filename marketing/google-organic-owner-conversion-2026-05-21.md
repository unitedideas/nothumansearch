# Google Organic Owner Conversion Scout - 2026-05-21

Automation: `business-marketer-not-human-search`
Scope: marketing scout artifact only. No outreach, posting, browser use, account creation, deploy, code edit, checkout, or public action.

## Fresh Evidence

Public surfaces:

- `/api/v1/stats`: `total_sites=4183`, `avg_score=35`, `top_category=developer`.
- `/api/v1/categories`: developer `1234`, ai-tools `905`, other `770`, data `401`, finance `196`, productivity `173`, ecommerce `149`, communication `120`, security `115`, health `59`, jobs `27`, education `21`, news `12`, spam `1`.
- `/.well-known/agent-card.json`: HTTP `404`.
- `/.well-known/mcp.json`: live and advertises `https://nothumansearch.ai/mcp` plus agent-readiness search tools.
- `/api/v1/catalog`: live and lists `nhs_geo_fix_my_score`, API Starter, and higher API-key plans.

Aggregate admin traffic, last 168 hours:

- Root `/`: `3409`.
- Google referrers: `google.com=542`, `www.google.com=45`.
- Canonical and alias referrers: `nothumansearch.com`, `www.nothumansearch.ai`, `www.nothumansearch.com`, and HTTP variants together remain material.
- `/score` referrer: `122`.
- High-volume owner/discovery routes: `/badge/xquik.com.svg=2150`, `/site/xquik.com=704`, `/.well-known/commerce.json=1548`, `/.well-known/ai-plugin.json=711`, `/llms.txt=461`, `/openapi.yaml=428`, `/api/v1/catalog=334`, `/api/v1/quote=304`, `/api/v1/checkout=304`, `/api/v1/search=168`, `/api/v1/submit=151`, `/top=97`, `/about=95`, `/api/v1=93`, `/.well-known/mcp.json=91`.

Aggregate MCP analytics, last 7 days:

- `tools/list=165314`, `initialize=24145`, `tools/call=263`.
- Tool calls: `search_agents=147`, `get_site_details=38`, `check_url=19`, `get_stats=17`, `find_mcp_servers=9`, `verify_mcp=8`, `get_top_sites=8`, `recent_additions=8`, `list_categories=5`, `submit_site=4`.
- Directory/client agents remain visible in aggregate: Cherry Studio, Claude Code, `MCP-Catalog-Bot`, `MCPScoringEngine`, and `mcp-verify`.

Owner-route smokes:

- `/score`, `/monitor`, and `/report`: HTTP `200`.
- `/fix/chainray.online`: HTTP `200`, high-score handoff includes `already meets the target` and `Monitor this score`.
- `/fix/bernstein.run`: HTTP `200`, remediation page shows score `90`, `target: 95+`, and refund language.

Public developer top-list sample:

- `agentprobe.fly.dev` score `100` - Foundry-owned dogfood, label before use.
- `xquik.com` score `100`.
- `mcp.depscope.dev` score `100`.
- `deadends.dev` score `100`.
- `agentdomainsearch.com` score `100`.

## Segment

Use Google organic and canonical-domain referrer traffic as an owner-conversion segment, not as proof of customer demand.

The traffic pattern says visitors are arriving through search and aliases, then touching score, submit, profile, badge, machine-readable manifest, API catalog, quote, and checkout surfaces. The next useful test is not another broad directory push. It is a routed owner path:

1. Search or root arrival -> `/score`.
2. Successful score or submit -> free `/monitor` registration.
3. High-score profile or badge traffic -> report sharing, badge proof, and monitor drift alerts.
4. Partial-score profile or fix traffic -> `/score` first, then score-fix remediation only after the current score proves a gap.
5. API/catalog-heavy traffic -> API-key plan handoff, with docs staying useful before the sales CTA.

## Draft Brief

Search traffic is already finding Not Human Search before it finds the score workflow.

The useful owner path is simple:

1. Check the public score.
2. Register a free monitor if the site is already agent-readable.
3. Use the report or badge as proof when the score is strong.
4. Fix missing machine-readable surfaces only when the current score shows a real gap.

Current corpus: 4,183 sites, average score 35. Google and alias traffic are already landing on the root, score, site-profile, badge, manifest, catalog, quote, and checkout paths.

No paid placement. No ranking bypass. Just public readiness evidence and a monitor when the evidence is already good.

## Gating Before Public Use

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/score`, `/monitor`, `/report`, `/api/v1/search`, `/api/v1/submit`, `/api/v1/top`, representative `/site/{host}` pages, high-score and partial-score `/fix/{host}` routes, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, aggregate MCP analytics, and aggregate traffic.
- Verify active Foundry/Owl-owned account identity for the selected channel.
- Check `marketing/social-post-ledger.json` if present, `outreach/distribution_log.csv`, and sync-state public-action locks.
- Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.

## Boundaries

Do not imply Google traffic, alias traffic, searched domains, profiled domains, badge routes, API/catalog traffic, or listed top-list domains are customers, endorsements, partners, paid leads, monitor registrations, badge-install consent, private demand, completed payments, revenue, crawler compliance, legal permission, SEO lift, uptime proof, A2A support, x402/ACP support, paid ranking placement, preferred inclusion, or score-methodology bypass.
