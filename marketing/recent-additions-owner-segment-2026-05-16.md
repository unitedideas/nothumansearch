# Recent Additions Owner Segment

Date: 2026-05-16
Automation: `business-marketer-not-human-search`
Status: prepared, not published

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a sanitized no-submit segment for a later gated channel or owner-conversion operator.

## Fresh Evidence

Public surfaces checked:

- `https://nothumansearch.ai/api/v1/stats`: 4,176 indexed sites, average score 35, top category `developer`.
- `https://nothumansearch.ai/api/v1/categories`: `developer=1228`, `ai-tools=902`, `other=770`, `data=403`, `finance=200`, `productivity=172`, `ecommerce=152`, `communication=117`, `security=115`, `health=57`.
- `https://nothumansearch.ai/newest`: HTTP 200 and exposes recent public site-profile links.
- `https://nothumansearch.ai/llms.txt`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, `/score`, `/monitor`, `/fix/cohere.com`, and `/fix/nothumansearch.ai`: HTTP 200.
- `https://nothumansearch.ai/.well-known/agent-card.json`: HTTP 404, so strict Agent Card directory submissions remain gated.

Aggregate admin signals, sanitized:

- MCP analytics, last 7 days: `tools/list=134346`, `initialize=18729`, `tools/call=297`.
- Top called MCP tools: `search_agents=193`, `get_site_details=38`, `verify_mcp=14`, `get_stats=13`, `check_url=13`.
- Traffic, last 336 hours: `/.well-known/commerce.json=1486`, `/badge/xquik.com.svg=1149`, `/api/v1/catalog=337`, `/api/v1/quote=309`, `/api/v1/checkout=309`, `/site/xquik.com=241`, `/top=135`, `/newest=95`, `/score=68`.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, or private query logs are included here.

## Public Recent-Addition Targets

These domains appeared on the public `/newest` page in this run. Treat them as public owner-channel targets or readiness-pattern examples, not customers, endorsements, paid leads, private demand, completed payments, or proof of market share.

- `amalgix.io`
- `surprise-buddy.com`
- `emorahealth.com`
- `felo.ai`
- `refunnel.com`
- `keenable.ai`
- `mpp.hyreagent.fun`
- `guavahealth.com`
- `memory.sylex.ai`
- `connect.composio.dev`
- `target.com`
- `kmengai.com`

## Segment Read

The `/newest` page is now a useful owner-channel source because it shows newly visible public agent-readiness profiles without needing a private read path. The segment is mixed, so the operator should not publish one broad claim across all targets. Split by owner fit after refreshing each public profile:

- High-score or complete-signal owners: route to free monitoring, badge/report sharing, and discovery-surface verification.
- Mid-score or low-score owners: route to `/score` first, then score-fix only if missing public agent-readiness signals justify remediation.
- Healthcare or health-data owners: avoid clinical, HIPAA, medical-accuracy, privacy-compliance, or regulatory claims.
- Marketplace, agent-commerce, or API owners: keep commerce/catalog/quote/checkout readiness separate from completed-payment or revenue claims.

## Draft Operator Copy

`Not Human Search now exposes a newest-sites feed for agent-readable web surfaces. It is a useful way for site owners to spot whether their llms.txt, OpenAPI, structured API, MCP, and monitorable metadata are visible to agents.`

`For owners already scoring high, the next step is usually monitoring and badge/report proof. For owners missing public agent-readiness signals, the score page shows the gap before any remediation offer.`

Proof links:

- `https://nothumansearch.ai/newest`
- `https://nothumansearch.ai/score`
- `https://nothumansearch.ai/monitor`
- `https://nothumansearch.ai/api/v1/catalog`
- `https://nothumansearch.ai/llms.txt`

## Publication Guard

Before any external use:

1. Refresh `/api/v1/stats`, `/api/v1/categories`, `/newest`, `/score`, `/monitor`, representative `/site/{host}` profiles, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, and aggregate `/api/v1/admin/mcp?days=7`.
2. Verify active account identity for the selected Foundry/Owl-owned channel.
3. Check `marketing/social-post-ledger.json` if it exists, sync-state public-action locks, and `outreach/distribution_log.csv`.
4. Do not use browser or Computer Use from the recurring worker.
5. Do not claim private demand, completed payments, revenue, customer endorsement, clinical endorsement, HIPAA/privacy compliance, price accuracy, data freshness, seller certification, x402/ACP/MPP support for NHS, paid ranking placement, preferred inclusion, or score-methodology bypass.

## Blockers

- Anonymous `/api/v1/site/{domain}` detail reads returned quota-gated HTTP 402 in this worker path; use public profile pages or an approved API-key/internal read path for per-domain detail enrichment.
- `/.well-known/agent-card.json` returns 404; strict Agent Card directory submissions remain gated.
- `tools/full-recrawl.lock` exists in the worktree; no runtime mutation, deploy, or broad crawl should be attempted from this scout run.
