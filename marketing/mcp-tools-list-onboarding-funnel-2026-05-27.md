# MCP tools-list onboarding funnel - 2026-05-27

Automation: `business-marketer-not-human-search`

Scope: no public action, outreach, browser/Computer Use, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write. This is a sanitized scout artifact for a later gated operator.

## Evidence

- Public stats: `total_sites=4174`, `avg_score=35`, and `top_category=developer`.
- Public categories: `developer=1231`, `ai-tools=905`, `other=774`, `data=402`, `finance=192`, `productivity=171`, `ecommerce=149`, `communication=118`, `security=113`, `health=59`, `jobs=26`, `education=21`, `news=12`, and `spam=1`.
- Live public surfaces returned 200: `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/api/v1`, `/api/v1/catalog`, `/api/v1/api-keys/subscribe`, `/monitor`, `/score`, and `/mcp`.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card claims remain blocked.
- Live MCP `tools/list` returned 11 tools: `search_agents`, `get_site_details`, `get_stats`, `submit_site`, `check_url`, `verify_mcp`, `register_monitor`, `list_categories`, `get_top_sites`, `recent_additions`, and `find_mcp_servers`.
- Aggregate MCP analytics, 7 days: `tools/list=180554`, `initialize=28507`, and `tools/call=402`.
- Aggregate MCP tool calls, 7 days: `search_agents=150`, `check_url=89`, `get_site_details=67`, `get_stats=30`, `submit_site=20`, `verify_mcp=13`, `find_mcp_servers=9`, `list_categories=8`, `recent_additions=6`, and `get_top_sites=6`. `register_monitor` remains present but small relative to `check_url`.
- Aggregate MCP query themes included local events, agent marketplace payments, local AI runtimes, AI agent jobs, WeChat/RSS monitoring, genetic wellness, model gateways, product-review/source discovery, ETF market data, function-calling pricing, and web search.
- Aggregate traffic, 168 hours: `/=3370`, `/badge/xquik.com.svg=2641`, `/.well-known/commerce.json=1326`, `/site/xquik.com=1105`, `/.well-known/ai-plugin.json=589`, `/llms.txt=433`, `/openapi.yaml=377`, `/api/v1/catalog=320`, `/badge/aidevboard.com.svg=286`, `/robots.txt=280`, `/badge/8bitconcepts.com.svg=264`, `/api/v1/checkout=252`, `/api/v1/quote=252`, `/api/v1/search=228`, and `/favicon.ico=222`.
- Aggregate referrers, 168 hours: `nothumansearch.ai root=1958`, `google.com=612`, `.com/www/http aliases remain material`, and `/score=78`.
- API-plan subscribe GET returned starter/pro/scale plan ids and limits, but `amount_cents` stayed `null` for each plan.
- Score-band route checks: high-score `/fix/nothumansearch.ai` returned the already-meets-target handoff; partial-score `/fix/manifest.ly` returned the paid remediation page; `/site/xquik.com` emphasized monitor proof; `/site/openai.com` showed low-score remediation copy.
- Latest local monitor worker proof, 2026-05-25: completed normally with five due monitors. Aggregate outcome was two first-check zero-score quarantines, two first-check partial or low-score checks, and one stable high-score check.

## Read

The broadest current marketing signal is not another individual query theme. It is the onboarding funnel itself:

1. A large number of agents and crawlers enumerate NHS tools.
2. A much smaller set calls tools.
3. `check_url` remains one of the highest-intent tools.
4. `register_monitor` is visible but still thin.
5. Commerce/catalog/quote/checkout routes are also being read, but API-plan price metadata is not fully machine-readable.

The next useful scout output should be a gated MCP-client onboarding test, not a public demand claim. The test should show a concrete next step after `tools/list` or `check_url`: search, check a URL, then either register a free monitor/report/badge proof path for high-score sites or send partial-score owners through `/score` and a missing-surface checklist before remediation.

## Candidate Test

Prepare one gated MCP-client onboarding or product-handoff test:

1. Discovery path: `tools/list` or `initialize` traffic -> install/use note for `search_agents`, `check_url`, and `get_site_details`.
2. Owner path: `check_url` or `/score` result -> high-score monitor/report/badge proof.
3. Remediation path: partial-score page -> `/score` first, then score-fix only after a fresh public score confirms missing public agent-readiness surfaces.
4. Sales path: API-key and catalog surfaces can be mentioned only after the API plan price-metadata handoff is resolved or copy avoids fully price-readable API-plan claims.
5. Admin path: zero-score and quarantined monitor outcomes stay private until bounded review records a safe outcome.

## Boundaries

Do not imply `tools/list`, `initialize`, `tools/call`, `check_url`, monitor registrations, badge routes, profile views, commerce/catalog traffic, referrers, MCP clients, or listed domains prove customers, endorsements, partners, paid leads, private demand, completed payments, revenue, crawler compliance, legal permission, SEO lift, uptime proof, A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/SPT/MPP support for NHS, paid placement, preferred inclusion, or score-methodology bypass.

Do not publish raw user-agent strings, raw MCP queries, private monitor rows, raw checkout URLs, payment identifiers, buyer emails, private score-fix rows, or private customer identifiers.

## Next Gated Action

Use this packet for exactly one gated MCP-client onboarding, owner-channel, or product-handoff test after refreshing public stats, discovery surfaces, aggregate MCP analytics, aggregate traffic, latest monitor worker proof, high-score plus partial-score `/fix/{host}` behavior, and the API-plan subscribe metadata.
