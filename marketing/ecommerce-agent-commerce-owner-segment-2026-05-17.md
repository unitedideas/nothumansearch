# Ecommerce Agent-Commerce Owner Segment

Date: 2026-05-17
Automation: `business-marketer-not-human-search`
Status: prepared, not published

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a sanitized scout segment for a later gated owner-channel or directory operator.

## Fresh Evidence

Public surfaces checked:

- `https://nothumansearch.ai/api/v1/stats`: 4,175 indexed sites, average score 35, top category `developer`.
- `https://nothumansearch.ai/api/v1/categories`: `ecommerce=151`, average score 41; larger adjacent buckets include `developer=1228`, `ai-tools=902`, `other=770`, `data=403`, and `finance=200`.
- `https://nothumansearch.ai/.well-known/mcp.json`: HTTP 200.
- `https://nothumansearch.ai/.well-known/agent.json`: HTTP 200.
- `https://nothumansearch.ai/.well-known/commerce.json`: HTTP 200.
- `https://nothumansearch.ai/api/v1/catalog`: HTTP 200 and lists score-fix plus Starter/Pro/Scale API products.
- `https://nothumansearch.ai/score`: HTTP 200.
- `https://nothumansearch.ai/monitor`: HTTP 200.
- `https://nothumansearch.ai/.well-known/agent-card.json`: HTTP 404, so strict Agent Card directory submissions remain gated.

Aggregate admin signals, sanitized:

- MCP analytics, last 7 days: `tools/list=136462`, `initialize=18582`, `tools/call=291`.
- Top called MCP tools: `search_agents=188`, `get_site_details=37`, `verify_mcp=14`, `check_url=13`, `get_stats=13`, `recent_additions=8`, `find_mcp_servers=8`, `get_top_sites=7`.
- Traffic, last 336 hours: `/.well-known/commerce.json=1548`, `/api/v1/catalog=347`, `/api/v1/quote=322`, `/api/v1/checkout=322`, `/llms.txt=475`, `/openapi.yaml=453`.
- Errors last hour: 0.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, private query logs, or raw buyer data are included here.

## Public Ecommerce Examples

These are public top-list examples only. Treat them as owner-channel targets or readiness-pattern examples, not customers, endorsements, paid leads, private demand, completed purchases, or proof of market share.

- `budgetfitter.uk` - score 100.
- `rettfrabonden.com` - score 100.
- `skillboss.co` - score 100.
- `ai.immoswipe.ch` - score 95.
- `can-tap-verified.com` - score 80.
- `businesshotels.com` - score 75.
- `store.farcomindustrial.com` - score 75.

## Segment Read

The commerce surfaces are now receiving material machine-readable traffic, and ecommerce remains a compact enough public category for one focused owner-channel test. The strongest safe angle is not "NHS drives buyers"; it is "agents need public seller surfaces they can inspect without guessing."

Good fit:

- Agent-commerce sellers with catalog, quote, checkout, refund/contact, and unsupported-rail metadata.
- Marketplace or ecommerce owners that already expose high-score signals and should monitor regressions.
- Mid-score sellers missing OpenAPI, structured API, MCP, or AI-friendly discovery surfaces.

Bad fit:

- Broad revenue copy.
- Claims that NHS validates product quality, pricing accuracy, fulfillment quality, seller certification, x402/ACP/MPP support, or buyer demand.
- Paid ranking, preferred inclusion, or score-methodology bypass language.

## Draft Operator Copy

`Not Human Search checks whether ecommerce and marketplace sites expose enough public structure for agents to evaluate them: llms.txt, OpenAPI, structured APIs, MCP, AI-friendly robots rules, plugin metadata, and schema.`

`For high-scoring sellers, the useful next step is usually monitoring and a shareable report. For sellers with missing machine-readable surfaces, the score page shows the gap before any remediation offer.`

Proof links:

- `https://nothumansearch.ai/top?category=ecommerce`
- `https://nothumansearch.ai/score`
- `https://nothumansearch.ai/monitor`
- `https://nothumansearch.ai/api/v1/catalog`
- `https://nothumansearch.ai/.well-known/commerce.json`
- `https://nothumansearch.ai/llms.txt`

## Publication Guard

Before any external use:

1. Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=ecommerce&limit=8`, `/score`, `/monitor`, representative `/site/{host}` profiles, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, and aggregate `/api/v1/admin/mcp?days=7`.
2. Verify active account identity for the selected Foundry/Owl-owned channel.
3. Check `marketing/social-post-ledger.json` if present, sync-state public-action locks, and `outreach/distribution_log.csv`.
4. Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.
5. Do not use browser or Computer Use from the recurring worker.
6. Do not claim listed domains are customers, endorsements, paid leads, private demand, completed payments, revenue, seller certification, fulfillment quality, price accuracy, data freshness, x402/ACP/MPP support for NHS, paid ranking placement, preferred inclusion, or score-methodology bypass.

## Blockers

- `/.well-known/agent-card.json` returns 404; strict Agent Card directory submissions remain gated.
- No repo-local `marketing/social-post-ledger.json` was found; a channel operator must still check the applicable social ledger or sync-state duplicate locks before posting.
- `tools/full-recrawl.lock/` is present as a pre-existing untracked lock directory; no deploy, broad crawl, or runtime mutation should be attempted from this scout run.
