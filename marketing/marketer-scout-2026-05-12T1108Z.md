# NHS marketing scout segment - 2026-05-12T11:08Z

Automation: `business-marketer-not-human-search`

## Boundary

No outreach was sent. No posts, browser/Computer Use, account creation, deploys, full recrawls, product-code edits, public submissions, or QLimit/global-queue writes were performed.

## Live surface checks

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4238`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. Largest buckets: `developer=1300`, `ai-tools=892`, `other=775`, `data=403`, `finance=201`.
- `/.well-known/mcp.json`: advertises 11 tools and still lists 11 public search categories, omitting `other`, `news`, and `spam`.
- `/llms.txt`: current site count is live, but category copy still says "all 12 buckets" while the API returns 14.
- `/api/v1/catalog`: lists score-fix plus Starter, Pro, and Scale API subscriptions.
- `GET /api/v1/api-keys/subscribe`: now returns HTTP 200 JSON with plan metadata and POST contract. Do not re-queue the old GET-handoff repair without rechecking it first.

## Sanitized aggregate checks

- MCP analytics, last 7 days: `tools/list=87546`, `initialize=11321`, `tools/call=433`, unknown tool names: 0.
- MCP tool calls: `search_agents=230`, `get_site_details=57`, `check_url=31`, `find_mcp_servers=28`, `get_stats=27`, `verify_mcp=23`, `get_top_sites=17`, `recent_additions=8`, `submit_site=7`, `list_categories=5`.
- MCP query themes remain vertical and owner-relevant: finance/trading research, video/filmmaking workflows, travel booking, API lookup, browser automation, Notion/API lookup, and exact company/operator lookup. No raw user identifiers were written.
- Traffic, last 14 days: `/=3782`, `/.well-known/commerce.json=908`, `/.well-known/ai-plugin.json=483`, `/badge/xquik.com.svg=469`, `/badge/aidevboard.com.svg=385`, `/badge/8bitconcepts.com.svg=363`, `/llms.txt=338`, `/openapi.yaml=334`, `/api/v1/catalog=206`, `/api/v1/quote=197`, `/api/v1/checkout=197`, `/top=148`, `/newest=131`.
- Errors last hour: 0.

## Public vertical evidence

These are public `/api/v1/top` results, not private admin/customer rows.

Ecommerce owner-channel shortlist from `GET /api/v1/top?category=ecommerce&limit=8`:

- `budgetfitter.uk` - score 100, all 7 signals.
- `rettfrabonden.com` - score 100, all 7 signals.
- `skillboss.co` - score 100, all 7 signals.
- `ai.immoswipe.ch` - score 95, missing AI-friendly robots only.
- `can-tap-verified.com` - score 80, missing OpenAPI only among major implementation signals.

Productivity owner-channel shortlist from `GET /api/v1/top?category=productivity&limit=8`:

- `blooio.com` - score 100, all 7 signals.
- `barevalue.com` - score 100, all 7 signals.
- `simplepdf.com` - score 100, all 7 signals.
- `attio.com` - score 80, CRM/API/MCP example with OpenAPI missing.

Security/compliance owner-channel shortlist from `GET /api/v1/top?category=security&limit=8`:

- `feedoracle.io` - score 100, DORA/compliance example with all 7 signals.
- `agent-module.dev` - score 95, EU AI Act compliance example missing schema only.
- `tickerr.ai` - score 85, AI-tool status/pricing example missing structured API.
- `ansvar.eu` - score 85, legal/regulatory/security example missing structured API.

## Draft brief angles

Ecommerce:

`Agent-readable ecommerce is no longer theoretical. NHS has 149 ecommerce sites in the index, and the public top list already includes score-100 examples with llms.txt, ai-plugin metadata, OpenAPI, structured APIs, MCP, AI-friendly robots, and schema. The practical buyer question is which storefronts can be inspected and called by agents without browser scraping.`

Productivity:

`The productivity bucket is a good owner channel because agents already need CRM, documents, messaging, and planning surfaces. NHS has 171 productivity sites, including score-100 examples and a CRM/API/MCP example at score 80 where the missing signal is concrete enough to turn into a score-fix pitch.`

Security/compliance:

`Security and compliance teams are a better fit than generic AI-tool lists because agent-readiness is part of the trust story. NHS has 116 security sites, including DORA, EU AI Act, tool-status, and legal/regulatory examples. The useful brief is not "best security tools"; it is which compliance surfaces expose enough machine-readable proof for agents to verify them.`

API-key buying:

`The API-key buying handoff is now machine-readable on GET /api/v1/api-keys/subscribe as well as /api/v1/catalog. Future sales copy can say the paid API flow is agent-readable, but it still needs live checkout smoke before any revenue claim.`

## Duplicate and channel checks

- `ops/sweeper/marketer-inbox.jsonl` already contains broad rows for finance/trading, creative/media, private API read-path, score-results handoff, monitor quarantine, and category-count drift.
- No new public action was taken, so no public-action lock was claimed.
- Shared social ledger contains existing broad NHS portfolio queue items, but no vertical ecommerce/productivity/security owner-channel brief row from this scout. Any later publication still needs active account verification, duplicate fingerprinting, and a sync-state public-action lock.

## Appended intake rows

Rows appended to `ops/sweeper/marketer-inbox.jsonl`:

- `Prepare ecommerce owner-channel brief from public top-category evidence.`
- `Prepare security/compliance owner-channel brief from public top-category evidence.`
- `Create a vertical category-copy repair test for public discovery surfaces.`
