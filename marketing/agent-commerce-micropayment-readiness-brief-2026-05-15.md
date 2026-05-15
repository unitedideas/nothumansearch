# NHS agent-commerce micropayment readiness brief

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-15T12:40Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a sanitized scout artifact for a later gated channel or product operator.

## Fresh Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: 4,175 indexed sites, average score 35, top category `developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories; `developer=1229`, `ai-tools=901`, `data=403`, `finance=200`, `ecommerce=152`.
- `https://nothumansearch.ai/.well-known/mcp.json`: advertises 11 tool definitions.
- `https://nothumansearch.ai/.well-known/commerce.json`: advertises 4 products for Not Human Search.
- `https://nothumansearch.ai/api/v1/catalog`: lists `nhs_geo_fix_my_score`, `nhs_api_starter`, `nhs_api_pro`, and `nhs_api_scale`.
- `https://nothumansearch.ai/api/v1/top?category=developer&limit=5`: returns 5 public results, all score 100.
- `https://nothumansearch.ai/api/v1/top?category=data&limit=5`: returns 5 public results, scores 90-100.
- `https://nothumansearch.ai/api/v1/top?category=ai-tools&limit=5`: returns 5 public results, but includes Foundry-owned dogfood domains, so use it as readiness-pattern evidence only.

Aggregate admin evidence, last 7 days:

- MCP `tools/list`: 124,138 calls.
- MCP `initialize`: 16,618 calls.
- MCP `tools/call`: 400 calls.
- Top called tools: `search_agents=240`, `get_site_details=51`, `find_mcp_servers=27`, `get_stats=20`, `check_url=17`, `get_top_sites=17`, `verify_mcp=16`, `recent_additions=8`, `list_categories=4`.
- Query themes included `x402 micropayment agent commerce monetize API pay per call`, `agent earn money freelance no upfront cost`, model/API pricing, marketplace price data, and agent-builder source discovery.

Aggregate admin traffic, last 336 hours:

- `/.well-known/commerce.json`: 1,404 requests.
- `/api/v1/catalog`: 314 requests.
- `/api/v1/quote`: 294 requests.
- `/api/v1/checkout`: 294 requests.
- `/.well-known/mcp.json`: 94 requests.
- `/api/v1`: 93 requests.
- `/top`: 142 requests.
- `/newest`: 113 requests.

Private workflow aggregates checked:

- Monitor status: `active=1`, `quarantined=1`, quarantine reason `bounded rerun still zero score`.
- Monitor actions in the last 30 days: `request_score_rerun=1`, `keep_quarantined=1`.
- Score-fix aggregate: 11 rows; `real_candidate pending=2`; no raw rows were exposed.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, or private query logs were written.

## Read

Agents are using NHS to look around the edges of agent commerce: x402-style API monetization, pay-per-call APIs, agent work/gig markets, model/API pricing, and marketplace data sources.

The safe owner-side angle is not that NHS supports x402 or certifies monetization. NHS currently exposes Stripe Checkout and agent-readable catalog/quote/checkout surfaces, while ACP/SPT/x402-style rails must remain explicit as unsupported unless live.

The useful message is source readiness for sellers:

- Agents need machine-readable product catalogs.
- Agents need quote endpoints that state price, rail, limits, and unsupported payment modes.
- API sellers need stable OpenAPI, MCP, or structured API metadata before agents can safely buy or integrate.
- Pay-per-call and micropayment experiments still need fallback rails, refund/contact metadata, and monitorable public docs.
- NHS can score and monitor the public surfaces, then route owners toward score-fix remediation or API-key products without selling ranking placement.

## Channel Brief

Short:

NHS agent traffic now includes x402/API monetization and pay-per-call discovery themes. The owner-side takeaway is not "use x402 now." It is that API sellers need machine-readable catalog, quote, checkout, refund, and unsupported-rail metadata before autonomous agents can evaluate or buy safely.

Long:

Agents are starting to search for API monetization paths, micropayment rails, marketplace data, and ways for agents to earn or buy services without manual browsing. Those queries are early and rail-specific, so NHS should not position itself as a payment-rail authority.

The stronger angle is readiness. If a seller wants agents to buy an API plan, score-fix package, data product, or pay-per-call service, the public site needs probeable commercial surfaces: `commerce.json`, `agent.json`, catalog, quote, checkout handoff, OpenAPI, MCP where relevant, and explicit unsupported-rail language.

## Suggested Follow-Up

Prepare a gated channel operator packet for agent-commerce, API-monetization, x402-adjacent, and developer-tool audiences:

- Use the commerce route traffic as evidence that agents inspect seller packets.
- Use the MCP query themes as evidence that agent-commerce discovery is happening, not as proof of completed purchases.
- Keep the copy centered on public source readiness and checkout legibility.
- If product work follows, add an example docs section: "Agent-readable commerce readiness for API sellers."

## Publication Guard

Before any external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=developer`, `/api/v1/top?category=data`, `/api/v1/top?category=ecommerce`, `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/commerce.json`, `/.well-known/agent.json`, `/api/v1/catalog`, and `/api/v1/admin/mcp?days=7`.
- Verify active channel account identity.
- Check `marketing/social-post-ledger.json` if a social/channel post is involved.
- Check sync-state public-action locks and `outreach/distribution_log.csv`.
- Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.
- Do not claim private demand, completed payments, revenue, customer endorsement, x402/ACP/SPT support for NHS, pricing accuracy, data freshness, seller certification, paid ranking placement, preferred inclusion, or score-methodology bypass.
