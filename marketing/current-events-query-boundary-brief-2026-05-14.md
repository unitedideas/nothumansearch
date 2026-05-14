# NHS current-events query boundary brief

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-14T15:40Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a sanitized scout artifact for a later gated channel or product operator.

## Fresh Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: 4,172 indexed sites, average score 35, top category `developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories; `news=12` with average score 50, `data=403` with average score 32, `communication=117` with average score 38.
- `https://nothumansearch.ai/llms.txt`: advertises 4,172+ sites and 11 MCP tools.
- `https://nothumansearch.ai/.well-known/mcp.json`: advertises the current MCP endpoint and public/audit-only category split.
- `https://nothumansearch.ai/api/v1/top?category=news&limit=8`: public news category is small and mixed; top examples include news, autonomous-news, business-news, and local-news surfaces scoring 70, 65, and 65 at the top.
- `https://nothumansearch.ai/api/v1/top?category=data&limit=8`: data category includes 100/95/90-score API and MCP examples across data-center intelligence, security intelligence, travel booking, compliance, markets, agent commerce, influencer data, and on-chain analytics.
- `https://nothumansearch.ai/api/v1/top?category=communication&limit=8`: communication category includes email, messaging, phone, and outreach-capable agent-readable services.

Aggregate admin evidence, last 7 days:

- MCP `tools/list`: 115,705 calls.
- MCP `initialize`: 15,167 calls.
- MCP `tools/call`: 337 calls.
- Top called tools: `search_agents=199`, `get_site_details=39`, `find_mcp_servers=26`, `get_stats=18`, `check_url=16`, `verify_mcp=15`, `get_top_sites=13`.
- Top query themes included current-events or entity-style lookups such as `news`, `Gibson Energy CEO`, `Canvas leak data released`, and market/news/sentiment queries.

Aggregate admin traffic, last 336 hours:

- `/`: 3,860 requests.
- `/.well-known/commerce.json`: 1,324 requests.
- `/llms.txt`: 439 requests.
- `/openapi.yaml`: 428 requests.
- `/api/v1/catalog`: 298 requests.
- `/api/v1/checkout`: 278 requests.
- `/api/v1/quote`: 278 requests.
- `/.well-known/mcp.json`: 94 requests.
- `/api/v1`: 93 requests.
- Google referrers: 201 combined requests from `google.com` and `www.google.com`.

Private workflow aggregates checked:

- Monitor status: `active=1`, `quarantined=1`, quarantine reason `bounded rerun still zero score`.
- Monitor actions in the last 30 days: `request_score_rerun=1`, `keep_quarantined=1`.
- Score-fix aggregate: 11 rows; `real_candidate pending=2`; no real paid or real lead row was exposed.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, or private query logs were written.

## Read

NHS is getting MCP usage that looks like live news, executive/entity lookup, and market-event discovery. That is useful demand signal for agent-facing news/data/API owners, but it is not proof that NHS should market itself as a current-events answer engine.

The safe positioning is:

- NHS finds agent-readable sources and tells agents which sites are machine-usable.
- It should not claim factual freshness, editorial coverage, or answer accuracy for volatile news and people/company queries.
- For current-events style channels, the better angle is "agents need probeable, machine-readable news/data sources" rather than "NHS answers the news."

## Channel Brief

Short:

NHS MCP usage now includes current-events and entity-style lookups: news, executive/company queries, market/news sentiment, and incident-style searches. The right angle is not that NHS is a news engine. It is that agents need machine-readable news and data sources they can verify before use.

Long:

Not Human Search is seeing agents use MCP discovery for volatile information needs: news, company/executive lookups, market sentiment, and incident-style queries. That points to an owner-side gap for news and data providers. If agents are going to ask for current facts, the source site needs public machine-readable affordances: `llms.txt`, OpenAPI, structured API responses, clear robots policy, and ideally MCP.

NHS should frame this as source-readiness and probe-before-use. It should not claim to be the source of truth for breaking news or live executive data.

## Suggested Follow-Up

Prepare a gated channel operator packet for news/data/API owners:

- Use the brief to target owner-side communities, API directories, or agent-builder channels.
- Keep the copy boundary explicit: NHS helps agents find machine-readable sources; it does not verify volatile facts or replace primary sources.
- If product work follows, add a small copy guard near news/current-event examples in docs or examples: "Use NHS to find agent-readable sources; verify volatile facts with the source."

## Publication Guard

Before any external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=news`, `/api/v1/top?category=data`, `/llms.txt`, `/.well-known/mcp.json`, and `/api/v1/admin/mcp?days=7`.
- Verify active channel account identity.
- Check `marketing/social-post-ledger.json` if a social/channel post is involved.
- Check sync-state public-action locks and `outreach/distribution_log.csv`.
- Do not claim private demand, completed payments, revenue, customer endorsement, factual freshness, editorial coverage, answer accuracy, paid ranking placement, preferred inclusion, ACP/x402 support for NHS, or score-methodology bypass.
