# Scanner and Document-Capture Source-Readiness Refresh

Run: 2026-05-24T12:08:00Z
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, checkout completion, or QLimit/global-queue write was
performed. This is a sanitized scout artifact for a later gated operator.

No raw users, emails, API keys, checkout URLs, payment identifiers, private
monitor rows, private score-fix rows, private query logs, buyer data, or raw
customer data are included here.

## Evidence

- Public stats: 4,177 indexed sites, average score 35, top category
  `developer`.
- Public categories: `developer=1230`, `ai-tools=904`, `data=399`,
  `finance=195`, `productivity=171`, `ecommerce=146`,
  `communication=119`, `security=113`, `health=59`, `jobs=26`,
  `education=21`, and `news=12`.
- Public discovery and owner surfaces returned 200: `/score`, `/monitor`,
  `/report`, `/newest`, `/top`, `/mcp-servers`, `/.well-known/mcp.json`,
  `/.well-known/agent.json`, `/.well-known/commerce.json`,
  `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`,
  `/llms.txt`, `/openapi.yaml`, and `/feed.xml`.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card
  directory claims remain blocked.
- Public productivity top-list examples: `tabai.dev=100`, `blooio.com=100`,
  `barevalue.com=100`, `simplepdf.com=100`, `angshumangupta.com=80`,
  `attio.com=80`, `monday.com=70`, and `berger.team=70`.
- Aggregate MCP analytics, 7 days: `tools/list=173191`,
  `initialize=27011`, and `tools/call=405`.
- Aggregate MCP tool calls, 7 days: `search_agents=172`,
  `check_url=85`, `get_site_details=60`, `submit_site=21`,
  `get_stats=20`, `verify_mcp=14`, `list_categories=8`,
  `find_mcp_servers=8`, `recent_additions=7`, and `get_top_sites=6`.
- Aggregate query themes included portable document scanners, ADF/batch
  scanning, electronics retail, hardware makers, RAG/document indexing,
  and product-review lookup.
- Aggregate traffic, 168 hours: `/=3367`, `/badge/xquik.com.svg=2534`,
  `/.well-known/commerce.json=1379`, `/site/xquik.com=966`,
  `/.well-known/ai-plugin.json=636`, `/llms.txt=454`,
  `/openapi.yaml=390`, `/api/v1/catalog=325`, `/api/v1/search=214`,
  `/api/v1/submit=143`, `/about=88`, and `/digest=86`.
- Latest local monitor-check proof remains clean for the most recent run:
  2026-05-18 processed one due monitor and completed without quarantine.

## Segment

Scanner, document-capture, electronics-retail, and RAG/document-indexing
owners need public source contracts that agents can inspect before
recommending a device, retailer, capture workflow, or indexing tool:

- supported model, SKU, OS, and driver boundaries;
- export, OCR, batch, ADF, or upload limitations;
- machine-readable support, warranty, pricing, and update metadata;
- API, OpenAPI, MCP, catalog, or feed contracts where a workflow is
  programmable;
- a monitorable readiness surface so deploys do not erase the metadata.

This refresh should not be used as proof that NHS covers the scanner market.
It is a recurring source-readiness signal from aggregate MCP themes and public
route traffic.

## Owner Routing

- High-score document, PDF, workflow, API, or catalog owners: route to free
  monitor registration, public report sharing, and badge/report proof.
- Partial-score scanner, electronics, hardware, or document-capture owners:
  route to `/score` first, then a missing-surface checklist before any
  score-fix remediation.
- API-heavy document capture, RAG, indexing, or ecommerce owners: route to
  API-key/catalog surfaces only when NHS docs remain useful.
- Directory or public-channel use stays gated behind account identity,
  duplicate checks, a public-action lock, and fresh surface probes.

## Claims To Avoid

Do not claim scanner hardware reliability, electrical safety, device
compatibility certification, driver correctness, OCR accuracy, RAG answer
quality, inventory accuracy, price freshness, warranty quality, support SLA,
privacy compliance, security certification, customer demand, private demand,
paid leads, completed payments, revenue, endorsement, paid placement,
preferred inclusion, A2A support while `/.well-known/agent-card.json` is 404,
x402/ACP/MPP support for NHS, or score-methodology bypass.
