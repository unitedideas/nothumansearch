# NHS Hermes and agent-skills referrer brief

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-15T06:20Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a sanitized scout artifact for a later gated channel or product operator.

## Fresh Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: 4,174 indexed sites, average score 35, top category `developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories; `developer=1229`, `ai-tools=900`, `data=403`, `finance=200`, `productivity=172`.
- `https://nothumansearch.ai/.well-known/mcp.json`: advertises 11 tools and the public/audit-only category split.
- `https://nothumansearch.ai/.well-known/commerce.json`: advertises the score-fix product plus starter/pro/scale API plans.
- `https://nothumansearch.ai/api/v1/catalog`: exposes the same score-fix and API subscription products.
- `https://nothumansearch.ai/.well-known/agent-card.json`: HTTP 404, so strict A2A/Agent Card directory work remains blocked until compatibility is added or intentionally documented.
- `https://nothumansearch.ai/api/v1/top?category=developer&limit=20`: top developer examples include agent probing, monitoring, package intelligence, dependency/security scanning, agent domain search, PDF rendering, and MCP/API surfaces.

Aggregate admin evidence, last 7 days:

- MCP `tools/list`: 120,124 calls.
- MCP `initialize`: 15,909 calls.
- MCP `tools/call`: 405 calls.
- Top called tools: `search_agents=248`, `get_site_details=50`, `find_mcp_servers=28`, `get_stats=19`, `check_url=17`, `verify_mcp=15`, `get_top_sites=15`, `recent_additions=10`, `list_categories=3`.
- Query themes included agent skills, Hermes-style agent tooling, A2A agent protocol discovery, browser automation/web scraping, Notion API, free LLM APIs, function calling/tool use, finance/trading data, and model-pricing searches.

Aggregate admin traffic, last 336 hours:

- `https://github.com/0xNyk/awesome-hermes-agent`: 31 referring requests.
- `/.well-known/commerce.json`: 1,349 requests.
- `/badge/xquik.com.svg`: 919 requests.
- `/llms.txt`: 445 requests.
- `/openapi.yaml`: 433 requests.
- `/api/v1/catalog`: 303 requests.
- `/api/v1/quote`: 283 requests.
- `/api/v1/checkout`: 283 requests.
- `/site/xquik.com`: 187 requests.

Distribution history:

- `outreach/distribution_log.csv` records `0xNyk/awesome-hermes-agent` PR `https://github.com/0xNyk/awesome-hermes-agent/pull/35` submitted on 2026-04-17 for NHS under Integrations & Bridges.

Private workflow aggregates checked:

- Monitor status: `active=1`, `quarantined=1`, quarantine reason `bounded rerun still zero score`.
- Score-fix aggregate: 11 rows; `real_candidate pending=2`; no real paid or real lead row was exposed.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, or private query logs were written.

## Read

The earlier Hermes/agent-awesome-list placement is producing measurable referral traffic, and current MCP query themes still include agent-skills and A2A/Hermes-adjacent discovery. That makes agent-builder directory follow-up more concrete than a fresh broad directory scan.

The safe angle is not that NHS is a Hermes product, an A2A implementation, or a ranked placement marketplace. It is that self-improving agents and agent-skill systems need a probeable source of agent-ready APIs, MCP servers, commerce surfaces, and monitorable score reports.

## Channel Brief

Short:

The Hermes awesome-list placement is now sending traffic to NHS, and agents are still querying for agent skills, A2A discovery, browser automation, MCP servers, and API/tool sources. The next channel packet should target agent-builder directories and skill/tooling communities with a narrow claim: NHS helps agents find probeable, agent-readable sources.

Long:

NHS has a working fit with agent-builder audiences because it is itself an MCP server and exposes public REST, llms.txt, OpenAPI, commerce, and catalog surfaces. The useful follow-up is a gated packet for agent-skill directories, Hermes/A2A-adjacent communities, and agent-builder lists that already accept discovery tools.

The packet should not mention private demand, customer intent, or completed payments. It can cite public surfaces and aggregate traffic only: the existing Hermes list referral, current MCP tool-call volume, public category counts, and the 11-tool MCP surface.

## Suggested Follow-Up

Prepare a gated channel operator packet for agent-builder directories:

- Start with agent-skills, autonomous-agent, A2A, and MCP-adjacent directories that are not `modelcontextprotocol/*` or `punkpeye/*`.
- Use the existing Hermes placement as evidence of channel fit, not as a reason to repost to the same repo.
- Pair the listing copy with public install and probe commands: `claude mcp add --transport http nothumansearch https://nothumansearch.ai/mcp`, `/api/v1/top`, `/api/v1/catalog`, and `/.well-known/mcp.json`.
- Keep the conversion route machine-readable: free search/check/monitor first, paid score-fix/API only where relevant.

## Publication Guard

Before any external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=developer`, `/api/v1/top?category=ai-tools`, `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, and `/api/v1/admin/mcp?days=7`.
- Check `outreach/distribution_log.csv` and sync-state public-action locks for duplicates.
- Verify active channel account identity.
- Check `marketing/social-post-ledger.json` if a social/channel post is involved.
- Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.
- Do not claim A2A support, Hermes endorsement, private demand, completed payments, revenue, customer endorsement, pricing accuracy, benchmark accuracy, certification, paid ranking placement, preferred inclusion, ACP/x402 support for NHS, or score-methodology bypass.
