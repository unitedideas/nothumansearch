# PDF, Form, and Document-Workflow Source Readiness - 2026-05-23

Run context: `business-marketer-not-human-search` recurring scout. No outreach, posting, browser, Computer Use, deploy, product-code edit, full recrawl, account creation, checkout completion, or QLimit/global-queue write was performed.

## Fresh Aggregate Signal

- Public stats: `/api/v1/stats` returned `total_sites=4174`, `avg_score=35`, and `top_category=developer`.
- Public category counts: `productivity=171 avg_score=39`, `communication=119 avg_score=38`, `developer=1230 avg_score=34`, and `data=399 avg_score=32`.
- Aggregate MCP analytics over 7 days: `tools/list=170940`, `initialize=27792`, `tools/call=244`.
- Top MCP tool calls: `search_agents=112`, `get_site_details=41`, `check_url=40`, `get_stats=18`, `submit_site=7`, `list_categories=6`, `find_mcp_servers=6`, `get_top_sites=6`, `recent_additions=6`, `verify_mcp=2`.
- Aggregate traffic over 168 hours: `/=3352`, `/badge/xquik.com.svg=2542`, `/.well-known/commerce.json=1418`, `/site/xquik.com=950`, `/.well-known/ai-plugin.json=656`, `/llms.txt=445`, `/openapi.yaml=403`, `/api/v1/catalog=320`, `/api/v1/quote=275`, `/api/v1/checkout=275`, `/api/v1/search=196`, `/api/v1/submit=143`, `/about=98`, `/top=96`, `/.well-known/mcp.json=91`, `/api/v1=91`, `/digest=85`, `/guide=75`, `/score=72`, `/newest=63`.
- Live discovery checks returned HTTP 200 for `/score`, `/monitor`, `/report`, `/newest`, `/top`, `/mcp-servers`, `/openapi-apis`, `/llms-txt-sites`, `/api/v1`, `/api/v1/catalog`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/llms.txt`, `/openapi.yaml`, and `/feed.xml`.
- `/.well-known/agent-card.json` returned HTTP 404, so strict Agent Card and A2A-style claims remain gated.

## Public Examples

Current `/api/v1/top?category=productivity&limit=15` exposes a concrete document, form, CRM, and workflow owner segment:

| Domain | Score | Public readiness pattern | Safe owner route |
|---|---:|---|---|
| `simplepdf.com` | 100 | PDF/document workflow surface with complete readiness signals; `/fix/simplepdf.com` routes to monitor/report proof instead of paid remediation. | Free monitor/report/badge proof; do not claim document security or PDF accuracy. |
| `tally.so` | 65 | Form and submission workflow surface with partial readiness; `/fix/tally.so` routes to remediation intake. | `/score` first, then missing-surface checklist before paid remediation. |
| `epublys.com` | 70 | Publishing/document workflow surface with partial readiness; `/fix/epublys.com` routes to remediation intake. | `/score` first, then missing-surface checklist. |
| `tabai.dev` | 100 | Productivity/developer workflow surface with complete readiness signals; `/fix/tabai.dev` routes to monitor/report proof. | Free monitor/report/badge proof. |
| `barevalue.com` | 100 | Productivity/workflow surface with complete readiness signals. | Free monitor/report/badge proof. |
| `blooio.com` | 100 | Productivity/workflow surface with complete readiness signals. | Free monitor/report/badge proof. |

These are public readiness examples or owner-channel targets only. They are not customers, endorsements, paid leads, monitor registrations, badge-install consent, completed payments, revenue, private demand, or proof of category market share.

## Useful Angle

PDF, form, and document-workflow owners have a source-readiness problem that is narrower than generic productivity copy. Agents may need to fill forms, inspect document workflows, route uploaded files, or cite generated documents, but a human landing page does not tell an agent enough.

The owner-side contract is:

1. Publish clear `llms.txt` scope and workflow boundaries.
2. Expose OpenAPI or structured API metadata when forms, documents, uploads, exports, or templates are meant to be agent-accessible.
3. Make auth, plan, support, privacy, and data-retention boundaries explicit.
4. Use free monitor/report/badge proof when the public surface is already high-scoring.
5. Use paid remediation only for concrete missing public contracts after `/score`.

Safe short copy:

`Agents that touch PDFs, forms, and document workflows need a public source contract before they submit data or cite outputs. Not Human Search checks whether that contract is inspectable: llms.txt, OpenAPI/API, real MCP if present, plugin metadata, robots policy, schema, and monitorable public profiles. High-score owners can prove and monitor readiness; partial-score owners get a concrete missing-surface checklist before remediation.`

## Gated Use

Use this for exactly one gated owner-channel touch, post, or product-handoff test for PDF tools, form builders, document workflow apps, publishing tools, upload/export systems, or productivity workflow owners.

Required refresh before external use:

- `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=productivity&limit=15`, and `/api/v1/top?category=communication&limit=10`
- `/score`, `/monitor`, `/report`, representative `/site/{host}` pages
- High-score and partial-score `/fix/{host}` routes
- `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`
- `/api/v1`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, `/feed.xml`
- Aggregate `/api/v1/admin/mcp?days=7` and `/api/v1/admin/traffic?hours=168`

Before public use, verify the active Foundry/Owl-owned account identity, check `marketing/social-post-ledger.json` if present, `outreach/distribution_log.csv`, and sync-state public-action locks, and avoid `modelcontextprotocol/*` plus `punkpeye/*` surfaces from `unitedideas`.

Do not imply PDF, form, publishing, document-workflow, productivity, or profiled domains are customers, partners, endorsements, paid leads, monitor registrations, badge-install consent, private demand, completed payments, revenue, document correctness, form-delivery reliability, PDF rendering quality, upload safety, privacy compliance, legal compliance, data-retention compliance, OCR accuracy, workflow quality, crawler compliance, SEO lift, uptime proof, A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/MPP support for NHS, paid ranking placement, preferred inclusion, or score-methodology bypass. Do not publish raw user-agent strings, private query logs, raw checkout URLs, payment identifiers, buyer emails, private monitor rows, private score-fix rows, or private customer identifiers.
