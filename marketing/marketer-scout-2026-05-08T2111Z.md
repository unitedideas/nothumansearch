# NHS marketing scout segment - 2026-05-08T21:11Z

Automation: `business-marketer-not-human-search`

## Boundary

No outreach was sent. No posts, browser/Computer Use, account creation, deploys, full recrawls, or product-code edits were performed. One live API-key subscription POST was smoke-tested with a non-customer test email; no payment was made and the returned Stripe Checkout URL is intentionally not recorded here.

## Live surface checks

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4233`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. Largest buckets: `developer=1302`, `ai-tools=888`, `other=767`, `data=402`, `finance=204`.
- `GET /mcp` and JSON-RPC `tools/list`: 11 matching tools.
- Public discovery and conversion routes returned 200: `/.well-known/mcp.json`, `/llms.txt`, `/api/v1`, `/openapi.yaml`, `/score`, `/monitor`, `/fix/nothumansearch.ai`, `/.well-known/commerce.json`, `/.well-known/agent.json`, `/api/v1/catalog`, `/api/v1/quote`.
- `GET /api/v1/api-keys/subscribe`: 405 method not allowed.
- `POST /api/v1/api-keys/subscribe` with JSON `{email, plan:"starter"}`: 200 with `plan=starter`, `monthly_limit=1000`, `amount_cents=1900`, and a Stripe Checkout URL. URL redacted from this artifact.

## Sanitized admin aggregates

- Score-fix intake: 11 total rows.
- Real candidate score-fix rows: 3 pending, all `1_6d`; 2 `dot_com`, 1 `foundry_owned`.
- Real paid or lead rows: 0.
- Test-like score-fix rows: 8 total; 2 paid, 1 lead, 5 pending.
- Monitor signups: 2 active, both checked, 0 notified; score buckets: 1 zero, 1 `80_100`.
- MCP analytics, last 7 days: `tools/list=34812`, `initialize=9014`, `tools/call=574`, unknown tool names: 0.
- MCP tool calls: `search_agents=345`, `check_url=69`, `get_site_details=61`, `get_stats=22`, `verify_mcp=21`, `submit_site=17`, `find_mcp_servers=16`, `get_top_sites=12`, `recent_additions=7`, `list_categories=4`.
- Traffic, last 14 days: `/=3783`, `/.well-known/commerce.json=523`, `/badge/aidevboard.com.svg=383`, `/badge/8bitconcepts.com.svg=363`, `/robots.txt=335`, `/.well-known/ai-plugin.json=310`, `/llms.txt=239`, `/openapi.yaml=237`, `/api/v1/catalog=129`.

## Duplicate and channel checks

- `marketing/social-post-ledger.json` does not exist in the NHS repo.
- Shared `8bitconcepts/marketing/social-post-ledger.json` already contains NHS/MCP-related posted and queued social items, including Q2 MCP ecosystem posts and portfolio daily NHS posts. This scout did not add social copy.
- `outreach/distribution_log.csv` is saturated with MCP, A2A, awesome-list, gist, email, IndexNow, Glama, PulseMCP, APIs.guru, and directory activity. This scout did not create another generic directory row.

## New finding

The active sales gap is not another launch post. The public API quota path is selling a paid API key, but the handoff is not agent- or human-friendly:

1. Anonymous `/api/v1/search` and `/api/v1/site/{domain}` now return `402 quota_exceeded` from this worker IP with `subscribe_url=https://nothumansearch.ai/api/v1/api-keys/subscribe`.
2. That `subscribe_url` returns 405 on GET, so a human, crawler, or agent following the URL cannot see plans or instructions.
3. The POST contract works and creates a live Stripe Checkout session for the starter plan, but that contract is not exposed in the commerce catalog.
4. `/api/v1/catalog` only advertises `nhs_geo_fix_my_score`; it omits the active API subscription plans (`starter`, `pro`, `scale`) even though the quota gate is already routing users toward paid API keys.

This is a better next conversion lane than reusing old Reddit/HN drafts: agents are already hitting quota and the paid API path exists, but the machine-readable buying surfaces do not describe it.

## Appended intake rows

Rows appended to `ops/sweeper/marketer-inbox.jsonl`:

- `Make NHS API-key checkout reachable from quota-exceeded handoffs.`
- `Expose NHS API subscription plans in agent-readable catalog and quote surfaces.`
- `Create an API-key-backed public API scout read path for marketer target lists.`
