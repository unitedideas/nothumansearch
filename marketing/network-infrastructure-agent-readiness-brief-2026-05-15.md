# Network infrastructure agent-readiness brief

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-15T16:08Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a sanitized scout artifact for a later gated channel or product operator.

## Fresh Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: 4,175 indexed sites, average score 35, top category `developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories; `developer=1229`, `ai-tools=901`, `data=403`, `finance=200`, `security=115`.
- `https://nothumansearch.ai/.well-known/mcp.json`: advertises 11 tool definitions.
- `https://nothumansearch.ai/.well-known/commerce.json`: advertises the score-fix product plus starter, pro, and scale API products.
- `https://nothumansearch.ai/api/v1/catalog`: lists `nhs_geo_fix_my_score`, `nhs_api_starter`, `nhs_api_pro`, and `nhs_api_scale`.
- `https://nothumansearch.ai/api/v1/top?category=security&limit=6`: returns 6 public security results, scores 80-100.
- `https://nothumansearch.ai/api/v1/top?category=developer&limit=6`: returns 6 public developer results, all score 100.

Aggregate admin evidence, last 7 days:

- MCP `tools/list`: 125,039 calls.
- MCP `initialize`: 16,711 calls.
- MCP `tools/call`: 394 calls.
- Top called tools: `search_agents=237`, `get_site_details=51`, `find_mcp_servers=27`, `get_stats=19`, `get_top_sites=17`, `check_url=16`, `verify_mcp=15`, `recent_additions=8`, `list_categories=4`.
- Query themes included VPN panel/API lookup, web scraping/browser automation, security/compliance tooling, model/API pricing, marketplace data, health-data APIs, and agent-commerce monetization.

Aggregate admin traffic, last 336 hours:

- `/.well-known/commerce.json`: 1,409 requests.
- `/.well-known/ai-plugin.json`: 692 requests.
- `/llms.txt`: 462 requests.
- `/openapi.yaml`: 443 requests.
- `/api/v1/catalog`: 315 requests.
- `/api/v1/checkout`: 295 requests.
- `/api/v1/quote`: 295 requests.
- `/.well-known/mcp.json`: 95 requests.
- `/api/v1`: 93 requests.
- `/mcp-servers`: 71 requests.

Private workflow aggregates checked:

- Monitor status: `active=1`, `quarantined=1`, quarantine reason `bounded rerun still zero score`.
- Monitor actions in the last 30 days: `request_score_rerun=1`, `keep_quarantined=1`.
- Score-fix aggregate: 11 rows; `real_candidate pending=2`; no raw rows were exposed.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, or private query logs were written.

## Read

Network and infrastructure operators are a useful owner-channel segment for NHS, but the safe claim is readiness, not operational correctness.

The recent VPN panel/API query theme suggests agents are looking for programmable infrastructure surfaces. NHS can support that motion by showing which infrastructure, security, automation, proxy, VPN, DNS, and API-control-plane sites expose agent-readable contracts.

The channel angle:

- Infrastructure products need stable public API documentation, OpenAPI, MCP where relevant, and machine-readable pricing or plan metadata.
- Agents need to know whether a control-plane API is probeable before they can integrate or recommend it.
- Site owners need monitorable readiness because docs, MCP manifests, and API routes drift after deploys.
- NHS can route high-score owners to free monitoring and badge/share proof, and low-score owners to score-fix remediation.

## Channel Brief

Short:

Agents are using NHS for infrastructure-source discovery: VPN panels, API control planes, browser automation, security tooling, and developer infrastructure. The owner-side takeaway is simple: if the API is meant to be used by agents, the public site needs machine-readable docs, OpenAPI or MCP, stable catalog metadata, and monitorable readiness.

Long:

Network and infrastructure tools increasingly advertise automation surfaces, but many still rely on human-readable docs and dashboard flows. Agents can only evaluate or integrate them safely when the public surface has a probeable contract: OpenAPI, MCP, structured API metadata, clear auth/plan boundaries, and explicit unsupported modes.

NHS is useful here as a readiness layer. It does not certify uptime, security, privacy, or operational safety. It checks whether the public site gives agents enough structured information to find, score, and monitor the integration surface.

## Suggested Follow-Up

Prepare a gated channel operator packet for infrastructure, VPN/proxy, DNS, security, browser-automation, and developer-control-plane audiences:

- Use aggregate MCP query themes as evidence that agents are searching for programmable infrastructure surfaces.
- Use public security/developer top lists as readiness-pattern examples, not customer-demand proof.
- Point owners toward free monitor registration if their score is already high.
- Point low-score owners toward score-fix remediation without implying ranking placement or methodology bypass.

## Publication Guard

Before any external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=security`, `/api/v1/top?category=developer`, `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/commerce.json`, `/.well-known/agent.json`, `/api/v1/catalog`, and `/api/v1/admin/mcp?days=7`.
- Verify active channel account identity.
- Check `marketing/social-post-ledger.json` if a social/channel post is involved.
- Check sync-state public-action locks and `outreach/distribution_log.csv`.
- Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.
- Do not claim private demand, completed payments, revenue, customer endorsement, uptime, security certification, privacy compliance, VPN safety, operational reliability, pricing accuracy, data freshness, paid ranking placement, preferred inclusion, ACP/x402 support for NHS, or score-methodology bypass.
