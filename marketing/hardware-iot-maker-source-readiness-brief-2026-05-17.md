# Hardware and IoT Maker Source-Readiness Brief

Date: 2026-05-17
Automation: business-marketer-not-human-search

## Segment

Recent MCP query themes included hardware and maker lookups:

- ESP32 Arduino embedded hardware IoT DIY maker
- electronics hardware circuit board maker

This is a useful owner-channel segment for Not Human Search because hardware, IoT, electronics, and maker tooling often need agents to discover stable docs, board metadata, firmware/API surfaces, inventory/catalog APIs, examples, and support boundaries before taking action.

## Live NHS Evidence

Public stats refreshed during this run:

- Total indexed sites: 4,172
- Average score: 35
- Top category: developer
- Developer category: 1,236 sites, average score 34
- Data category: 399 sites, average score 32
- Other category: 765 sites, average score 27

Live discovery surfaces checked:

- `/llms.txt`: 200
- `/.well-known/mcp.json`: 200
- `/.well-known/agent.json`: 200
- `/.well-known/commerce.json`: 200
- `/api/v1/catalog`: 200
- `/api/v1`: 200
- `/monitor`: 200
- `/score`: 200
- `/.well-known/agent-card.json`: 404

Aggregate MCP usage, last 7 days:

- `tools/list`: 141,870
- `initialize`: 19,214
- `tools/call`: 297
- Top called tools: `search_agents` 192, `get_site_details` 39, `check_url` 13, `get_stats` 12, `recent_additions` 11, `verify_mcp` 10

Aggregate traffic, last 168 hours:

- `/.well-known/commerce.json`: 1,604
- `/api/v1/catalog`: 359
- `/api/v1/checkout`: 333
- `/api/v1/quote`: 333
- `/top`: 122
- `/newest`: 90
- `/score`: 69

## Public Example Boundaries

Current public top lists do not isolate a clean hardware/IoT category. The closest public inventory is mixed across developer, data, and other:

- Developer top list is dominated by agent/developer infrastructure examples such as `xquik.com`, `mcp.depscope.dev`, `deadends.dev`, `agentdomainsearch.com`, `agentndx.ai`, and `rendoc.dev`.
- Data top list is API-heavy and includes security, compliance, market, supply-chain, scheduling, and travel-data examples such as `api.contrastcyber.com`, `api.theartofservice.com`, `api.meacheal.ai`, and `app.daedalmap.com`.
- Other top list is mixed and includes high-score agent services plus local/consumer examples such as `astranl.com`, `lobehub.com`, `surprise-buddy.com`, `crabbitmq.com`, and `holz.biz`.

Use these as readiness examples only. Do not imply they are hardware customers, endorsements, private demand, paid leads, completed payments, or proof of category market share.

## Owner-Channel Angle

Hardware and IoT owners can use NHS as a readiness check before agents depend on their docs or APIs:

- Can an agent discover current docs through `llms.txt`, OpenAPI, MCP, or a structured API?
- Are firmware, board, SDK, catalog, inventory, and support boundaries machine-readable?
- Are pricing, licensing, auth, contact, and refund/support surfaces explicit?
- Can owners monitor score regressions after docs or API changes?

Score-band routing:

- High-score owners: free monitor, report page, and badge proof.
- Partial-score owners: `/score` first, then score-fix only when missing public agent-readiness signals justify remediation.
- API/catalog-heavy owners: API plans and commerce/catalog readiness, without claiming x402/ACP/MPP support.

## Do Not Claim

- Do not claim hardware reliability, electrical safety, firmware correctness, security certification, compliance certification, uptime, pricing accuracy, inventory freshness, fulfillment quality, private demand, seller certification, paid ranking placement, preferred inclusion, or score-methodology bypass.
- Do not claim A2A support while `/.well-known/agent-card.json` returns 404.
- Do not treat scanner paths or generic badge/profile traffic as buyer demand.

## Gated Next Step

Prepare one owner-channel touch or post for hardware, IoT, electronics, robotics, or maker-tool owners after refreshing live stats and selected public examples. The channel operator must verify account identity, check duplicate ledgers and public-action locks, and avoid `modelcontextprotocol/*` plus `punkpeye/*` surfaces from `unitedideas`.
