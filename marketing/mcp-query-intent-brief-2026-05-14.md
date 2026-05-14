# NHS MCP Query Intent Brief

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-14T14:08Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a sanitized scout artifact for a later gated channel operator.

## Live Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4172`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories; largest public categories were developer 1229, ai-tools 899, data 403, finance 200, productivity 172, ecommerce 152, communication 117, security 115.
- `https://nothumansearch.ai/llms.txt`: current count line is `4172+ sites`; MCP tool count is 11.
- `https://nothumansearch.ai/.well-known/mcp.json`: advertises the current public category split and 11 tool definitions.
- `https://nothumansearch.ai/api/v1/top?category=data&limit=8`: data top list includes 100/95/90-score API and MCP examples across data-center intelligence, security intelligence, travel booking, compliance, market state, agent commerce, influencer data, and on-chain analytics.
- `https://nothumansearch.ai/api/v1/top?category=finance&limit=8`: finance top list includes 100-score market/crypto/trading examples and 95/90-score payment or data APIs.

Aggregate admin evidence, last 7 days:

- MCP `tools/list`: 110,591 calls.
- MCP `initialize`: 14,355 calls.
- MCP `tools/call`: 336 calls.
- Tool calls: `search_agents=196`, `get_site_details=37`, `find_mcp_servers=27`, `get_stats=18`, `verify_mcp=17`, `check_url=15`, `get_top_sites=14`, `recent_additions=8`, `list_categories=4`.
- No unknown MCP tool names appeared in the aggregate tool list.

Aggregate traffic evidence, last 336 hours:

- `/.well-known/commerce.json`: 1,261 requests.
- `/api/v1/catalog`: 283 requests.
- `/api/v1/checkout`: 266 requests.
- `/api/v1/quote`: 266 requests.
- `/site/xquik.com`: 147 requests.
- `/badge/xquik.com.svg`: 817 requests.
- `/.well-known/mcp.json`: 97 requests.
- `/api/v1`: 96 requests.
- Google referrers: 203 combined requests from `google.com` / `www.google.com`.

No raw users, API keys, emails, checkout URLs, payment identifiers, private monitor rows, or private query logs were written.

## Intent Themes

The 7-day MCP query sample points to four useful channel angles:

1. Finance and trading agent readiness

Evidence: finance category has 200 sites at average score 40, and the MCP query sample included intraday/VWAP/ORB/backtest, NSE India trading, market sentiment/news, and stock-card-price-style lookup patterns.

Use: finance API owner or agent-builder brief. Keep it about machine-readable market data, safe tool discovery, and probe-before-use. Do not imply trading performance, investment advice, or private customer demand.

2. Agent-commerce and pay-per-call APIs

Evidence: commerce manifest traffic remains the strongest agent-readable surface, and the MCP query sample included x402 micropayment / pay-per-call API discovery.

Use: agent-commerce directory packet or seller-readiness post. Keep it tied to `commerce.json`, catalog, quote, checkout, explicit unsupported rails, and NHS as dogfood. Do not claim ACP/x402 support for NHS; current surfaces explicitly mark those unsupported.

3. Free/low-cost LLM API discovery

Evidence: multiple MCP queries asked for free LLM APIs, OpenRouter free models, NVIDIA NIM free-tier APIs, and function-calling/tool-use options.

Use: future "agent API discovery" brief or target shortlist after the private API read path exists. Public search is quota-blocked for anonymous scout reads, so do not build an exact target list from `/api/v1/search` until the private read path is available.

4. Developer/productivity tools

Evidence: aggregate MCP queries included Notion API, Penpot MCP, GitHub repository/code-search tooling, browser automation/web scraping, agent skills, and Hermes agent MCP.

Use: developer-tool channel brief that shows NHS is being used as live agent infrastructure discovery rather than a static directory. Refresh `developer`, `communication`, and `productivity` category top lists before publication.

## Badge/Owner Loop

The most visible new owner-channel signal in traffic is `xquik.com`:

- `/badge/xquik.com.svg`: 817 requests.
- `/site/xquik.com`: 147 requests.
- Public report page: `https://nothumansearch.ai/site/xquik.com`.
- Public page score: 100/100 with all seven agentic signals found.

This is a useful dogfood-style proof that third-party badge embeds can send owner/profile traffic back into NHS. It should become a conversion test, not a public customer claim:

- High-score badge owners should be routed toward free monitoring and "keep this score from regressing".
- Low-score badge/profile pages should keep the score-fix remediation path.
- Public copy must avoid saying xquik is a customer, paid lead, or endorsement.

## Draft Channel Copy

Short:

NHS saw 336 MCP tool calls in the last 7 days. Most were search and site-detail calls, but the query themes are concrete: finance/trading APIs, x402/pay-per-call API discovery, free LLM APIs, Notion/Penpot/GitHub-style developer tools, and MCP server lookup.

Long:

Not Human Search is being used less like a static directory and more like agent-side discovery infrastructure. In the last 7 days, the MCP endpoint handled 110,591 `tools/list` calls, 14,355 initializes, and 336 `tools/call` requests. The top called tool was `search_agents`, followed by site details, MCP-server discovery, stats, MCP verification, live URL checks, and top-site lookups.

The useful part is the shape of demand: agents are looking for finance and trading data, pay-per-call API surfaces, free LLM API options, developer tool APIs, and working MCP servers. NHS should keep pushing probe-before-use, machine-readable surfaces, and owner remediation rather than raw index size.

## Publication Guard

Before any external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=data`, `/api/v1/top?category=finance`, `/llms.txt`, `/.well-known/mcp.json`, and `/api/v1/admin/mcp?days=7`.
- Verify active channel account identity.
- Check `marketing/social-post-ledger.json` if a social/channel post is involved.
- Check sync-state public-action locks and `outreach/distribution_log.csv`.
- Do not claim private demand, completed payments, revenue, customer endorsement, investment advice, paid ranking placement, preferred inclusion, ACP/x402 support for NHS, or score-methodology bypass.
