# Document Scanning and RAG Source-Readiness Brief

Date: 2026-05-18
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, checkout completion, or QLimit/global-queue write was
performed. This is a sanitized scout artifact for a later gated operator.

No raw users, emails, API keys, checkout URLs, payment identifiers, private
monitor rows, private score-fix rows, private query logs, buyer data, or raw
customer data are included here.

## Evidence Snapshot

Public surfaces checked:

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4174`,
  `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: `developer=1237`,
  `data=399`, `ecommerce=149`, `productivity=173`, `communication=119`,
  `security=115`, and `news=12`.
- `https://nothumansearch.ai/api/v1/top?category=developer&limit=12`,
  `/api/v1/top?category=data&limit=12`, and
  `/api/v1/top?category=ecommerce&limit=12`: public top lists returned HTTP
  200 with `results` arrays.
- `https://nothumansearch.ai/llms.txt`, `/.well-known/mcp.json`,
  `/.well-known/agent.json`, `/.well-known/commerce.json`, `/api/v1/catalog`,
  `/api/v1/quote`, `/score`, `/monitor`, and `/openapi.yaml`: HTTP 200.
- `https://nothumansearch.ai/api/v1/checkout`: HTTP 400 without a request body,
  which is expected for the POST contract and should not be treated as route
  downtime.
- `https://nothumansearch.ai/.well-known/agent-card.json`: HTTP 404, so strict
  Agent Card and A2A-style directory claims remain gated.

Aggregate admin evidence, sanitized:

- MCP analytics, 7 days: `tools/list=145197`, `initialize=19357`,
  `tools/call=324`.
- Top MCP tool calls: `search_agents=209`, `get_site_details=44`,
  `check_url=15`, `get_stats=13`, `verify_mcp=11`, `recent_additions=10`,
  `get_top_sites=8`, `find_mcp_servers=7`, `submit_site=4`,
  `list_categories=3`.
- Aggregate query themes included document indexing/RAG, knowledge graphs,
  document scanners, electronics retail, and local hardware/device lookup.
- Traffic, 168 hours: `/.well-known/commerce.json=1619`, `/api/v1/catalog=362`,
  `/api/v1/quote=336`, `/api/v1/checkout=336`, `/score=71`, `/top=119`,
  `/newest=89`, and `/.well-known/mcp.json=87`.

## Segment

Document-scanning, capture hardware, RAG indexing, and knowledge-base product
owners whose buyers increasingly expect agents to understand source catalogs,
device compatibility, indexing boundaries, API contracts, and support metadata.

This is a source-readiness segment, not proof that NHS has comprehensive
scanner, retail, or RAG coverage. The public categories do not isolate scanner
hardware or document capture cleanly, so later channel work should frame the
examples as adjacent readiness patterns rather than market-share evidence.

## Public Examples

These are public top-list examples only. Treat them as readiness-pattern
examples or owner-channel targets, not customers, endorsements, paid leads,
private demand, completed purchases, or proof of market coverage.

| Domain | Score | Current agent-readiness shape | Safe owner route |
|---|---:|---|---|
| `mcp.depscope.dev` | 100 | Package-intelligence MCP/API surface with complete public readiness signals. | Use as a developer-reference pattern for machine-readable indexing metadata, not document/RAG coverage proof. |
| `deadends.dev` | 100 | Developer knowledge base with full public readiness signals. | Free monitor/report/badge proof; useful adjacent example for searchable knowledge surfaces. |
| `dchub.cloud` | 100 | Data-intelligence platform with complete public readiness signals. | Monitor/report/badge proof; do not claim data freshness or coverage accuracy. |
| `api.contrastcyber.com` | 100 | Security-intelligence API/MCP with complete public readiness signals. | API-contract example only; do not claim security certification. |
| `budgetfitter.uk` | 100 | Ecommerce/catalog-style surface with complete public readiness signals. | Agent-commerce/catalog pattern only; do not claim retail conversion or endorsement. |
| `can-tap-verified.com` | 80 | Local-business hardware/reputation product with API, MCP, plugin, robots, and schema, missing OpenAPI. | `/score` first; remediation can focus on OpenAPI and support/catalog metadata. |

## Angle

Agents looking for scanners, document ingestion, or RAG sources need more than a
human product page. They need stable public contracts: model and SKU metadata,
device compatibility, API or export boundaries, indexing limits, support
channels, pricing or plan metadata, and a way to monitor whether those surfaces
disappear after a site deploy.

Safe short copy:

`Document capture and RAG workflows are starting to look like agent workflows, but the public web surface is still uneven. In Not Human Search, the adjacent examples that work best expose machine-readable catalogs, API contracts, MCP or OpenAPI surfaces, and monitorable metadata. The owner path is not paid ranking: run /score, publish the missing public contracts, and register a free monitor so deploys do not erase the signals agents rely on.`

Owner-channel routes:

- High-score knowledge/API/catalog owners: free monitor registration, public
  report page, and badge proof.
- Partial-score hardware or document-capture owners: `/score` first, then
  score-fix only when missing public surfaces justify remediation.
- API-heavy RAG and indexing products: API/catalog readiness and paid NHS API
  plans only when the buyer asks for higher-volume NHS access.

## Publication Guard

Before external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, representative public top
  lists or `/site/{host}` profiles, `/score`, `/monitor`,
  `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/agent-card.json`, `/.well-known/commerce.json`,
  `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`,
  `/openapi.yaml`, and aggregate `/api/v1/admin/mcp?days=7`.
- Verify the active Foundry/Owl-owned account identity for the selected channel.
- Check `marketing/social-post-ledger.json` if present, sync-state
  public-action locks, and `outreach/distribution_log.csv`.
- Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.
- Do not imply listed domains are customers, endorsements, paid leads, private
  demand, completed payments, revenue, scanner hardware reliability, electrical
  safety, device compatibility certification, OCR accuracy, RAG answer quality,
  data freshness, retailer inventory accuracy, support SLA, privacy compliance,
  seller certification, x402/ACP/MPP support for NHS, paid ranking placement,
  preferred inclusion, A2A support, or score-methodology bypass.
