# Design and productivity source-readiness brief

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-15T17:08Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a sanitized scout artifact for a later gated channel or product operator.

## Fresh Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: 4,175 indexed sites, average score 35, top category `developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories; `developer=1229`, `ai-tools=901`, `productivity=172`, `communication=117`.
- `https://nothumansearch.ai/.well-known/mcp.json`: advertises 11 tool definitions.
- `https://nothumansearch.ai/llms.txt`: advertises 4,175+ sites and the same 11 MCP tools.
- `https://nothumansearch.ai/api/v1/catalog`: lists score-fix plus starter, pro, and scale API products.
- `https://nothumansearch.ai/api/v1/top?category=productivity&limit=6`: returns productivity examples including CRM messaging, PDF/document workflows, and CRM surfaces scoring 80-100.
- `https://nothumansearch.ai/api/v1/top?category=ai-tools&limit=6`: returns AI-tool examples scoring 100, but includes Foundry-owned dogfood entries that must be labeled if used externally.
- `https://nothumansearch.ai/api/v1/top?category=developer&limit=6`: returns developer/control-plane examples scoring 100.

Aggregate admin evidence, last 7 days:

- MCP `tools/list`: 125,994 calls.
- MCP `initialize`: 16,856 calls.
- MCP `tools/call`: 398 calls.
- Top called tools: `search_agents=236`, `get_site_details=52`, `find_mcp_servers=27`, `get_stats=19`, `verify_mcp=18`, `get_top_sites=17`, `check_url=16`, `recent_additions=8`, `list_categories=4`, `submit_site=1`.
- Query themes included Notion API, Penpot MCP, Roboflow/computer-vision Python tooling, email/calendar communication productivity, coding-agent developer tools, free LLM APIs, and browser automation/web scraping.

Aggregate admin traffic, last 336 hours:

- `/.well-known/commerce.json`: 1,414 requests.
- `/.well-known/ai-plugin.json`: 690 requests.
- `/llms.txt`: 462 requests.
- `/openapi.yaml`: 443 requests.
- `/api/v1/catalog`: 316 requests.
- `/api/v1/checkout`: 296 requests.
- `/api/v1/quote`: 296 requests.
- `/top`: 141 requests.
- `/newest`: 112 requests.
- `/.well-known/mcp.json`: 92 requests.
- `/api/v1`: 90 requests.
- `/score`: 66 requests.

Private workflow aggregates checked:

- Monitor status: `active=1`, `quarantined=1`, quarantine reason `bounded rerun still zero score`.
- Monitor actions in the last 30 days: `request_score_rerun=1`, `keep_quarantined=1`.
- Score-fix aggregate: 11 rows; `real_candidate pending=2`; no raw rows were exposed.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, or private query logs were written.

## Read

Design, productivity, and workflow tools are a useful owner-channel segment for NHS because agents are asking for specific work surfaces, not generic SaaS names.

The safe claim is source-readiness:

- Agents need probeable public contracts for design APIs, workspace APIs, document tools, CRM systems, communication/calendar surfaces, and computer-vision developer tools.
- A product can be excellent for humans and still weak for agents if the public site lacks OpenAPI, MCP, structured API metadata, or clear machine-readable plan/auth boundaries.
- NHS can show owners whether the public surface is ready for agent discovery and monitoring.
- NHS should not claim workflow quality, design-system correctness, private demand, usage endorsement, integration reliability, data freshness, privacy compliance, or paid ranking placement.

## Channel Brief

Short:

Agents are using NHS to look for design and workflow sources: Notion API, Penpot MCP, Roboflow/computer-vision tooling, email/calendar productivity, and coding-agent tools. The owner-side message is direct: if agents are expected to use the product, the public site needs a probeable contract, not only human docs.

Long:

Design and productivity products increasingly expose APIs, plugins, MCP servers, and automation hooks, but the discovery surface often remains human-first. Agents can only recommend or integrate these tools safely when they can find structured metadata, verify API contracts, and understand what is supported without scraping marketing pages.

NHS is a readiness layer for that gap. It checks public agent-readable signals and gives owners a way to monitor regressions after docs, manifests, or API routes drift. It is not a quality ranking for the underlying product, a privacy/compliance certification, or a paid placement surface.

## Suggested Follow-Up

Prepare a gated channel operator packet for design-tool, productivity, workflow-automation, computer-vision, CRM, and developer-tool audiences:

- Use aggregate MCP query themes as evidence that agents are searching for these surfaces.
- Use public productivity/developer top lists as readiness-pattern examples, not customer-demand proof.
- Label Foundry-owned examples as dogfood if they appear in any AI-tools top list.
- Point high-score owners toward free monitoring and badge/share proof.
- Point low-score owners toward score-fix remediation without implying ranking placement or methodology bypass.

## Publication Guard

Before any external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=productivity`, `/api/v1/top?category=developer`, `/api/v1/top?category=ai-tools`, `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/commerce.json`, `/.well-known/agent.json`, `/api/v1/catalog`, and `/api/v1/admin/mcp?days=7`.
- Verify active channel account identity.
- Check `marketing/social-post-ledger.json` if a social/channel post is involved.
- Check sync-state public-action locks and `outreach/distribution_log.csv`.
- Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.
- Do not claim private demand, completed payments, revenue, customer endorsement, workflow quality, design-system correctness, integration reliability, privacy compliance, data freshness, paid ranking placement, preferred inclusion, ACP/x402 support for NHS, or score-methodology bypass.
