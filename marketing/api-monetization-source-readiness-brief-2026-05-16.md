# API monetization source-readiness brief

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-16T10:18Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a sanitized scout artifact for a later gated channel or product operator.

## Fresh Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: 4,176 indexed sites, average score 35, top category `developer`.
- `https://nothumansearch.ai/api/v1/categories`: largest public buckets include `developer=1229`, `ai-tools=901`, `data=403`, `finance=200`, `productivity=172`, `ecommerce=152`, `communication=117`, and `security=115`.
- `https://nothumansearch.ai/.well-known/mcp.json`: live and advertises 11 MCP tools.
- `https://nothumansearch.ai/.well-known/agent.json`: live and advertises REST API, OpenAPI, MCP, commerce catalog, quote, checkout, and API-key subscription surfaces.
- `https://nothumansearch.ai/.well-known/commerce.json`: live and lists score-fix remediation plus Starter/Pro/Scale API products.
- `https://nothumansearch.ai/api/v1/catalog`: live and lists `nhs_geo_fix_my_score`, `nhs_api_starter`, `nhs_api_pro`, and `nhs_api_scale`.
- `https://nothumansearch.ai/score`, `/monitor`, `/top`, `/newest`, `/fix/nothumansearch.ai`, and `/fix/cohere.com`: all returned HTTP 200.
- `https://nothumansearch.ai/.well-known/agent-card.json`: HTTP 404, so strict Agent Card directory submissions remain gated.

Aggregate admin evidence, last 7 days and 336 hours:

- MCP methods: `tools/list=132301`, `initialize=18215`, `tools/call=318`.
- Top called MCP tools: `search_agents=196`, `get_site_details=39`, `find_mcp_servers=20`, `get_stats=15`, `verify_mcp=15`, `check_url=13`, `get_top_sites=9`.
- Top-query theme counts from the aggregate 30-row set: `dev_api=18`, `commerce_payments=10`, `model_llm=10`, `marketplace_data=9`, `health_data=3`, `agent_card_a2a=1`, `security_infra=1`.
- Traffic: `/.well-known/commerce.json=1475`, `/api/v1/catalog=333`, `/api/v1/quote=307`, `/api/v1/checkout=307`, `/llms.txt=472`, `/openapi.yaml=445`, `/.well-known/ai-plugin.json=706`, `/.well-known/mcp.json` remains a material discovery route.
- Organic/search loop: Google referrers totaled 343 combined requests from `google.com` and `www.google.com`; `/top` had 136 route requests.
- Monitor aggregate: `active=1`, `quarantined=1`, quarantine reason `bounded rerun still zero score`.
- Score-fix aggregate: 11 total rows; `real_candidate pending=2`; test-like rows remain separate.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, or private query logs were written.

## Read

The current query mix is not just "find MCP servers." Agents are clustering around API contracts, pricing or payment language, model/API providers, and marketplace data. That makes API monetization and marketplace-source readiness the best current marketing segment.

The safe claim is narrow:

- Sellers and API owners need machine-readable product, pricing, plan, auth, and checkout boundaries.
- Agents need source surfaces they can inspect before recommending, quoting, or integrating an API.
- NHS can show whether those public surfaces are present and monitor whether they regress.

This is not proof of buyer demand, revenue, completed payments, price accuracy, model quality, marketplace coverage, x402 support, or seller certification.

## Channel Brief

Short:

Agents are querying NHS for APIs, commerce/payment surfaces, model providers, and marketplace data. The owner-side takeaway is that monetized APIs need public machine-readable contracts: OpenAPI, MCP where relevant, catalog/plan metadata, auth boundaries, checkout or quote handoffs, refund/contact metadata, and monitorable readiness.

Long:

NHS is a useful readiness layer for API sellers because it separates discoverability from claims of accuracy or endorsement. It can show whether a public site exposes the surfaces agents need before using it: llms.txt, OpenAPI, structured API docs, MCP, commerce metadata, catalog/quote/checkout routes, robots rules, and schema.

For API marketplaces, model gateways, pricing-data services, and agent-commerce sellers, the channel angle should be source-readiness. Agents should be able to find the API contract, understand unsupported payment rails, see plans or quote endpoints, and monitor public-readiness regressions without scraping a dashboard.

## Suggested Follow-Up

Prepare a gated channel operator packet for one of these audience families:

- API monetization and developer-tool directories.
- Agent-commerce and agent-payment communities.
- Model/API gateway owners.
- Marketplace-data and pricing-data API owners.
- API-first founder/operator communities where source-readiness is a useful owner-side problem.

Use public links only:

- `https://nothumansearch.ai/score`
- `https://nothumansearch.ai/monitor`
- `https://nothumansearch.ai/api/v1/catalog`
- `https://nothumansearch.ai/.well-known/commerce.json`
- `https://nothumansearch.ai/.well-known/agent.json`
- `https://nothumansearch.ai/.well-known/mcp.json`
- `https://nothumansearch.ai/openapi.yaml`
- `https://nothumansearch.ai/llms.txt`

## Publication Guard

Before any external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, and `/api/v1/admin/mcp?days=7`.
- Verify active channel account identity.
- Check `marketing/social-post-ledger.json` if a social/channel post is involved.
- Check sync-state public-action locks and `outreach/distribution_log.csv`.
- Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.
- Do not claim private demand, completed payments, revenue, customer endorsement, pricing accuracy, benchmark accuracy, model-quality certification, marketplace coverage, seller certification, x402/ACP/MPP support for NHS, paid ranking placement, preferred inclusion, or score-methodology bypass.
