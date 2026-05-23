# Outdoor gear and product-review source readiness - 2026-05-23

Run context: `business-marketer-not-human-search` recurring scout. No outreach, posting, browser, Computer Use, deploy, product-code edit, full recrawl, account creation, or QLimit write was performed.

## Fresh aggregate signal

- Public stats: `/api/v1/stats` returned 200 with 4,171 indexed sites and average score 35.
- Public category counts: `ecommerce=148 avg_score=41`, `data=402 avg_score=32`, `finance=194 avg_score=40`, `security=114 avg_score=38`.
- Aggregate MCP analytics over 7 days: `tools/list=169490`, `initialize=27742`, `tools/call=218`.
- Tool-call mix: `search_agents=104`, `get_site_details=37`, `check_url=31`, `get_stats=19`, `get_top_sites=6`, `list_categories=5`, `find_mcp_servers=5`, `recent_additions=5`, `submit_site=4`, `verify_mcp=2`.
- Visible aggregate query themes include outdoor gear review, trail-running shoes, hiking/backpacking reviews, electronics retail, scanner hardware, Singapore news/housing, genetics/wellness, model APIs, secrets management, and Hermes/agent skills.
- Aggregate traffic over 168 hours: `/=3364`, `/badge/xquik.com.svg=2477`, `/.well-known/commerce.json=1464`, `/site/xquik.com=887`, `/.well-known/ai-plugin.json=675`, `/llms.txt=446`, `/openapi.yaml=409`, `/api/v1/catalog=322`, `/robots.txt=302`, `/api/v1/quote=286`, `/api/v1/checkout=286`, `/api/v1/search=177`, `/api/v1/submit=142`, `/top=96`, `/.well-known/mcp.json=88`.
- Live discovery checks: `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1/catalog`, `/openapi.yaml`, `/score`, `/monitor`, `/mcp-servers` returned 200; `/.well-known/agent-card.json` returned 404.

## Public examples

Current `/api/v1/top?category=ecommerce&limit=10` includes public ecommerce and product/service examples with mixed readiness:

- `budgetfitter.uk` - score 100; llms.txt, OpenAPI, structured API, and MCP present.
- `rettfrabonden.com` - score 100; llms.txt, OpenAPI, structured API, and MCP present.
- `skillboss.co` - score 100; llms.txt, OpenAPI, structured API, and MCP present.
- `ai.immoswipe.ch` - score 95; llms.txt, OpenAPI, structured API, and MCP present.
- `packrift.com` - score 80; llms.txt and MCP present; structured API missing in the public top response.
- `can-tap-verified.com` - score 80; llms.txt, structured API, and MCP present; OpenAPI missing in the public top response.
- `businesshotels.com` - score 75; llms.txt and OpenAPI present; structured API and MCP missing in the public top response.
- `store.farcomindustrial.com` - score 75; llms.txt and OpenAPI present; structured API and MCP missing in the public top response.
- `la-palma24.net` - score 75; llms.txt, structured API, and MCP present; OpenAPI missing in the public top response.
- `maplebridge.io` - score 70; llms.txt, OpenAPI, and structured API present; MCP missing in the public top response.

These are examples of public readiness states, not customers, endorsements, private demand, paid leads, or market-share proof.

## Useful angle

Outdoor-gear, product-review, electronics-retail, and affiliate-commerce sites increasingly get queried by agents before a user buys. The owner-side gap is not whether the review is true. The gap is whether agents can verify stable source contracts before relying on it:

1. Machine-readable product, review, test-methodology, author, disclosure, and update-date metadata.
2. Stable `llms.txt`, OpenAPI/API, catalog, feed, sitemap, and robots policy.
3. Monitorable drift for score drops, missing manifests, broken API/catalog paths, and stale product/review source files.
4. Score-band-aware routing: high-score owners to free monitor/report/badge proof; partial-score owners to `/score` before remediation.

## Draft channel note

Subject: Product review pages agents can verify

NHS is seeing agent queries for hiking shoes, backpacking gear, scanner hardware, electronics retail, and product-review sources.

The useful owner-side test is whether an agent can verify the source contract before it trusts the page: `llms.txt`, OpenAPI/API, product or review metadata, update dates, robots policy, and a stable public profile it can monitor for drift.

NHS can score the public surface and show the missing machine-readable pieces:

https://nothumansearch.ai/score

For high-score review/catalog sites, the next useful step is the free monitor so owners know when an agent-facing surface regresses. For partial-score sites, the first step is a fresh score check and missing-surface list before any paid remediation.

## Use boundaries

Use this as a gated owner-channel or product-handoff test for product-review, outdoor-gear, electronics-retail, affiliate-commerce, and buyer-guide owners. It is not a claim that NHS verifies review truth, product quality, inventory, price freshness, affiliate revenue, or recommendation accuracy.

Before external use, refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=ecommerce&limit=12`, `/score`, `/monitor`, `/report`, representative high-score and partial-score `/site/{host}` pages, high-score and partial-score `/fix/{host}` routes, `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, aggregate MCP analytics, and aggregate traffic.

Do not imply product-review, outdoor-gear, electronics, ecommerce, affiliate, or profiled domains are customers, partners, endorsements, paid leads, monitor registrations, badge-install consent, private demand, completed payments, revenue, review truth, product quality, inventory accuracy, price freshness, affiliate revenue, safety claims, crawler compliance, SEO lift, uptime proof, A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/MPP support for NHS, paid ranking placement, preferred inclusion, or score-methodology bypass.
