# Public Top-List Owner Target Pack

Date: 2026-05-16
Automation: `business-marketer-not-human-search`
Status: prepared, not published

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a sanitized scout target pack for a later gated channel or owner-conversion operator.

## Fresh Evidence

Public surfaces checked:

- `https://nothumansearch.ai/api/v1/stats`: 4,177 indexed sites, average score 35, top category `developer`.
- `https://nothumansearch.ai/api/v1/categories`: largest public buckets include `developer=1229`, `ai-tools=902`, `other=770`, `data=403`, `finance=200`, `productivity=172`, `ecommerce=152`, `communication=117`, `security=115`, and `health=57`.
- `https://nothumansearch.ai/.well-known/mcp.json`: live and advertises 11 tools.
- `https://nothumansearch.ai/.well-known/agent.json`: live and advertises REST API, OpenAPI, MCP, commerce catalog, quote, checkout, and API-key subscription surfaces.
- `https://nothumansearch.ai/.well-known/commerce.json`: live and lists score-fix remediation plus Starter/Pro/Scale API products.
- `https://nothumansearch.ai/api/v1/catalog`: live and lists `nhs_geo_fix_my_score`, `nhs_api_starter`, `nhs_api_pro`, and `nhs_api_scale`.
- `https://nothumansearch.ai/.well-known/agent-card.json`: HTTP 404, so strict Agent Card directory submissions remain gated.
- `https://nothumansearch.ai/score`, `/monitor`, `/top`, `/newest`, `/fix/nothumansearch.ai`, and `/fix/cohere.com`: HTTP 200.

Aggregate admin signals, sanitized:

- MCP analytics, last 7 days: `tools/list=133689`, `initialize=18579`, `tools/call=303`.
- Top called MCP tools: `search_agents=194`, `get_site_details=38`, `get_stats=15`, `verify_mcp=15`, `check_url=12`, `find_mcp_servers=9`, `get_top_sites=9`.
- Traffic, last 336 hours: `/.well-known/commerce.json=1486`, `/api/v1/catalog=337`, `/api/v1/quote=309`, `/api/v1/checkout=309`, `/site/xquik.com=238`, `/top=135`, `/llms.txt=476`, `/openapi.yaml=446`.
- Monitor admin actions, last 30 days: `request_score_rerun=1`, `keep_quarantined=1`.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, or private query logs are included here.

## Public Target Segments

These are public top-list examples only. Treat them as owner-channel targets or readiness-pattern examples, not customers, endorsements, private leads, paid demand, or proof of market share.

### Data/API Owners

- `dchub.cloud` - score 100.
- `api.contrastcyber.com` - score 100.
- `api.boostedchat.com` - score 95.
- `api.theartofservice.com` - score 90.
- `api.headlessoracle.com` - score 90.

Angle: high-score API owners can be routed toward free monitoring, badge/report sharing, and API-plan visibility checks rather than paid remediation.

### Finance/Market Data Owners

- `terminalfeed.io` - score 100.
- `chartlibrary.io` - score 100.
- `prereason.com` - score 100.
- `devdrops.run` - score 95.
- `ticksurfers.com` - score 90.

Angle: finance and market-data providers need stable source contracts agents can inspect. Do not claim investment advice, price accuracy, data freshness, compliance certification, or trading reliability.

### Ecommerce/Marketplace Owners

- `budgetfitter.uk` - score 100.
- `rettfrabonden.com` - score 100.
- `skillboss.co` - score 100.
- `ai.immoswipe.ch` - score 95.
- `can-tap-verified.com` - score 80.

Angle: agent-commerce and marketplace owners can use NHS to verify public machine-readable product, catalog, quote, and checkout handoff surfaces. Do not claim revenue, completed purchases, x402/ACP support, seller certification, or private demand.

### Health-Data Owners

- `emorahealth.com` - score 100.
- `zgts.in` - score 100.
- `opdstar.com` - score 80.
- `hipaaagent.ai` - score 80.
- `monarchinitiative.org` - score 65.

Angle: healthcare and health-data owners need probeable API contracts and monitorable readiness. Do not claim clinical endorsement, HIPAA compliance, medical accuracy, or regulatory certification.

### Developer-Tool Owners

- `agentprobe.fly.dev` - score 100.
- `xquik.com` - score 100.
- `mcp.depscope.dev` - score 100.
- `deadends.dev` - score 100.
- `agentdomainsearch.com` - score 100.

Angle: developer-tool owners are the cleanest channel for monitor/badge sharing, MCP/API discovery, and source-readiness education. Do not imply badge-heavy traffic means the domain is a customer or endorsement.

## Operator Use

Good next actions:

1. Pick one owner segment and refresh the relevant `/api/v1/top?category=...` list before external use.
2. For high-score targets, use monitor/report/badge proof as the conversion path.
3. For mid-score or low-score targets, route to `/score` first and only then to `/fix/{host}` if missing agent-readiness signals justify remediation.
4. Keep API-plan copy separate from site-owner remediation.

## Publication Guard

Before any external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, the selected `/api/v1/top?category=...` list, `/score`, `/monitor`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, and aggregate `/api/v1/admin/mcp?days=7`.
- Verify active account identity for the selected Foundry/Owl-owned channel.
- Check `marketing/social-post-ledger.json` if it exists, sync-state public-action locks, and `outreach/distribution_log.csv`.
- Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.
- Do not claim private demand, completed payments, revenue, customer endorsement, pricing accuracy, data freshness, compliance certification, medical accuracy, seller certification, x402/ACP/MPP support for NHS, paid ranking placement, preferred inclusion, or score-methodology bypass.

## Blockers

- `tools/full-recrawl.lock` is active during this run; no deploy, broad crawl, or runtime mutation should be attempted from this marketing scout.
- `/.well-known/agent-card.json` returns 404; strict Agent Card directory submissions remain gated.
- No repo-local `marketing/social-post-ledger.json` was found; a channel operator must still check the applicable social ledger or sync-state duplicate lock before posting.
