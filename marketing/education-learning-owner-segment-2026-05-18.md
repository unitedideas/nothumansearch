# NHS Education and Learning Owner Segment

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-18T01:24Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, checkout completion, or QLimit/global-queue write was
performed. This is a sanitized owner-channel scout segment for a later gated
operator.

No raw users, emails, API keys, checkout URLs, payment identifiers, private
monitor rows, private score-fix rows, private query logs, learner data, or raw
buyer data are included here.

## Live Evidence

Public surfaces checked during this scout:

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4174`,
  `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: education has `21` sites with
  average score `49`.
- `https://nothumansearch.ai/api/v1/top?category=education&limit=12`: public
  education top list returned HTTP 200 with a `results` array.
- `https://nothumansearch.ai/llms.txt`: advertises 4,174+ indexed sites and 11
  MCP tools.
- `https://nothumansearch.ai/.well-known/mcp.json`: advertises the current MCP
  endpoint and public category parameter copy.
- `https://nothumansearch.ai/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/api/v1/catalog`, `/score`, `/monitor`,
  `/api/v1`, and `/openapi.yaml`: HTTP 200.
- `https://nothumansearch.ai/.well-known/agent-card.json`: HTTP 404, so strict
  Agent Card and A2A-style directory claims remain gated.
- High-score `/fix/nothumansearch.ai` and `/fix/aidevboard.com` route to an
  "already meets the NHS score target" page, so high-score score-fix gating is
  live.

Aggregate admin evidence, sanitized:

- MCP analytics, 7 days: `tools/list=143185`, `initialize=19240`,
  `tools/call=319`.
- Top MCP tool calls: `search_agents=205`, `get_site_details=42`,
  `check_url=18`, `get_stats=12`, `recent_additions=11`, `verify_mcp=10`,
  `get_top_sites=8`, `find_mcp_servers=6`, `submit_site=4`,
  `list_categories=3`.
- Traffic, 168 hours: `/score` appears as a referrer, and explicit
  machine-readable commerce surfaces remain active: `/.well-known/commerce.json`
  1604 hits, `/api/v1/catalog` 359, `/api/v1/checkout` 333, `/api/v1/quote`
  333. These are route counts only, not payment or buyer claims.

## Public Owner Targets

These are public top-list examples only. Treat them as owner-channel targets or
readiness-pattern examples, not customers, endorsements, paid leads, private
demand, completed purchases, or proof of market share.

| Domain | Score | Current agent-readiness shape | Safe owner route |
|---|---:|---|---|
| `sourcelibrary.org` | 100 | Full public signal set: `llms.txt`, AI plugin, OpenAPI, structured API, MCP, AI-friendly robots, and Schema.org. | Free monitor/report/badge proof, not score-fix. |
| `coursera.org` | 90 | Strong course/catalog surface; current crawl does not show MCP. | Monitor/report proof; score result first if discussing MCP. |
| `admit-coach.com` | 70 | College application platform with `llms.txt`, OpenAPI, structured API, AI-friendly robots, and Schema.org; missing AI plugin and MCP. | `/score` first; remediation can focus on agent discovery metadata and bounded agent tools. |
| `quizapi.io` | 65 | Developer-first quiz API with `llms.txt`, OpenAPI, and structured API; missing AI plugin, AI-friendly robots, MCP, and Schema.org. | `/score` first; remediation can focus on missing public metadata and monitoring after fixes. |
| `slidemaster.tw` | 65 | AI course-design tool with `llms.txt`, AI plugin, structured API, and Schema.org; missing OpenAPI, AI-friendly robots, and MCP. | `/score` first; remediation can focus on OpenAPI and agent access policy. |
| `sansfiction.com` | 55 | Digital library with `llms.txt`, AI plugin, AI-friendly robots, and Schema.org; missing OpenAPI, structured API, and MCP. | `/score` first; remediation only if owner has a real API/tool surface. |
| `samiolearning.com` | 50 | Kids learning app with `llms.txt`, structured API, AI-friendly robots, and Schema.org; missing AI plugin, OpenAPI, and MCP. | `/score` first; avoid learner-data or education-outcome claims. |

## Angle

Education and learning products are a clean owner-channel segment because agents
already need course catalogs, library metadata, quiz APIs, admissions tools, and
learning-resource boundaries. The useful claim is not that NHS certifies
education quality. It is that a product can expose enough public machine-readable
surface for agents to inspect it without scraping or guessing.

Safe short copy:

`The public education bucket in Not Human Search is small but concrete: 21 sites, average score 49, with examples ranging from complete 100/100 machine-readable libraries to course and quiz products missing one or two public agent-readiness surfaces. For education owners, the fix is not paid ranking. It is making course catalogs, learning metadata, quiz APIs, admissions flows, and support boundaries legible to agents, then monitoring the signals so deploys do not erase them.`

Owner-channel routes:

- High-score owners: free monitor registration, public report page, and badge
  proof.
- Partial-score owners: `/score` first, then score-fix only when missing public
  surfaces justify remediation.
- API-heavy learning products: API/catalog readiness and paid API plans only
  when the buyer asks for higher-volume NHS access.

## Publication Guard

Before external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`,
  `/api/v1/top?category=education&limit=12`, representative `/site/{host}`
  profiles, `/score`, `/monitor`, `/.well-known/mcp.json`,
  `/.well-known/agent.json`, `/.well-known/agent-card.json`,
  `/.well-known/commerce.json`, `/api/v1/catalog`, `/api/v1/quote`,
  `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, and aggregate
  `/api/v1/admin/mcp?days=7`.
- Verify the active Foundry/Owl-owned account identity for the selected channel.
- Check `marketing/social-post-ledger.json` if present, sync-state
  public-action locks, and `outreach/distribution_log.csv`.
- Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.
- Do not imply listed domains are customers, endorsements, paid leads, private
  demand, completed payments, revenue, enrollment growth, education quality,
  learning outcome certification, learner-data access, privacy compliance,
  child-safety certification, data freshness, seller certification,
  x402/ACP/MPP support for NHS, paid ranking placement, preferred inclusion,
  A2A support, or score-methodology bypass.
