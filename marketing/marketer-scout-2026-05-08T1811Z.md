# NHS marketing scout segment - 2026-05-08T18:11Z

Automation: `business-marketer-not-human-search`

## Boundary

No outreach was sent. No posts, browser/Computer Use, account creation, deploys, full recrawls, or product-code edits were performed.

## Live surface checks

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4232`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. Top visible buckets: `developer=1301`, `ai-tools=888`, `other=767`, `data=402`, `finance=204`.
- `GET /mcp`: 11 tools, matching JSON-RPC `tools/list`.
- Tool names: `search_agents`, `get_site_details`, `get_stats`, `submit_site`, `check_url`, `verify_mcp`, `register_monitor`, `list_categories`, `get_top_sites`, `recent_additions`, `find_mcp_servers`.
- Public discovery and conversion routes returned 200: `/.well-known/mcp.json`, `/llms.txt`, `/api/v1`, `/openapi.yaml`, `/score`, `/monitor`, `/fix/nothumansearch.ai`, `/.well-known/commerce.json`, `/api/v1/catalog`, `/api/v1/quote`.
- `GET /api/v1/checkout` returned 400 because checkout is a POST endpoint with required metadata; do not count that as a surface outage.
- Shared 8bit social ledger: 0 entries, 0 NHS mentions. Still take the public-action lock and duplicate check before any future post.

## Sanitized admin aggregates

- Score-fix intake: 11 total rows.
- Real candidate score-fix rows: 3 pending, all `1_6d`; 2 `dot_com`, 1 `foundry_owned`.
- Real paid or lead rows: 0.
- Test-like score-fix rows: 8 total; 2 paid, 1 lead, 5 pending.
- Monitor signups: 2 total, both checked, 0 notified; one low-score bucket, one high-score bucket.
- MCP analytics, last 7 days: `tools/list=33513`, `initialize=9181`, `tools/call=550`, unknown tool names: 0.
- MCP tool calls: `search_agents=344`, `check_url=67`, `get_site_details=56`, `get_stats=21`, `verify_mcp=18`, `submit_site=17`, `find_mcp_servers=13`, `recent_additions=6`, `get_top_sites=5`, `list_categories=3`.
- Traffic, last 14 days: `/=3789`, `/.well-known/commerce.json=518`, `/badge/aidevboard.com.svg=381`, `/badge/8bitconcepts.com.svg=362`, `/robots.txt=336`, `/.well-known/ai-plugin.json=309`, `/llms.txt=237`, `/openapi.yaml=237`, `/api/v1/catalog=128`.
- Monitor worker log: last real monitor processing observed on 2026-05-04. May 8 attempts were dry-run or skipped because Fly was unavailable or token was missing in the local wrapper context.

## New scout conclusions

The prior `GET /mcp` drift finding is stale. The live GET response now lists all 11 tools and matches `tools/list`, so new rows should not keep importing that drift item unless a fresh check regresses.

The strongest new marketing signal is owner conversion, not another broad launch draft:

- Badge SVG views are high enough to justify a badge-to-owner conversion test. Those views come from pages already embedding NHS assets, which is a warmer owner channel than generic social posting.
- `check_url` has 67 MCP tool calls in 7 days, but `register_monitor` is not present in the tool-call aggregate. The existing check-to-monitor test row remains valid.
- Free monitor marketing should wait for a fresh worker-liveness proof. The landing page is live, but the scheduled monitor-check path needs current proof before pushing it harder as an owner promise.
- Agent-commerce routes are being crawled directly. `commerce.json`, catalog, quote, and checkout are a directory-candidate packet, but submissions must stay behind duplicate checks, account identity checks, and public-action locks.

## Directory and owner-channel candidates

No-submit target packet:

1. Agent-commerce and seller-readiness directories
   - Proof links: `/.well-known/commerce.json`, `/.well-known/agent.json`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`.
   - Current duplicate check: `outreach/distribution_log.csv` contains no agent-commerce-specific submission rows.
   - Gate: directory account/form access plus public-action lock.

2. MCP/API marketplace updates that can carry commerce metadata
   - Candidate families from current TODO: Glama, PulseMCP, Smithery, MCP Market, Cursor Marketplace, LobeHub, publicmcpregistry, cursor.directory.
   - Existing caution: Glama has connector metadata and must be claimed or updated, not duplicated.
   - Gate: browser/login or approved non-browser API channel; not from this recurring scout runtime.

3. Badge-owner conversion
   - Proof links: `/badge/aidevboard.com.svg`, `/badge/8bitconcepts.com.svg`, `/site/aidevboard.com`, `/site/8bitconcepts.com`.
   - Test idea: add a site-owner path from badge docs or site pages that says "monitor this score" and "fix missing signals" without implying paid placement.
   - Gate: product-worker implementation and live route verification.

## Appended intake rows

Rows appended to `ops/sweeper/marketer-inbox.jsonl`:

- `Repair monitor-check proof before pushing the free monitor CTA harder.`
- `Create a badge-traffic owner conversion test from existing NHS badge embeds.`
- `Prepare gated submissions from the agent-commerce directory target packet.`
