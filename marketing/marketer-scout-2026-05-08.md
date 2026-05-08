# NHS marketing scout segment — 2026-05-08

Automation: `business-marketer-not-human-search`
Run time: `2026-05-08T08:10:58Z`

## Live surface checks

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4232`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories; `other=767`, `other_avg_score=26`.
- JSON-RPC `tools/list`: 11 tools: `search_agents`, `get_site_details`, `get_stats`, `submit_site`, `check_url`, `verify_mcp`, `register_monitor`, `list_categories`, `get_top_sites`, `recent_additions`, `find_mcp_servers`.
- `GET /mcp`: still advertises only 8 tools, so the existing marketer row for discovery-surface drift remains valid.
- Public discovery routes returned 200: `/.well-known/mcp.json`, `/llms.txt`, `/api/v1`, `/openapi.yaml`, `/score`, `/monitor`, `/fix/nothumansearch.ai`.
- `tools/mcp-registry/server.json`: v1.7.1 still says `4,100+ sites, 11 tools`.

## Sanitized admin aggregates

- Score-fix intake: 11 rows total.
- Real candidate score-fix rows: 3 pending, all `1_6d`; no real paid or lead rows in the redacted aggregate.
- Test-like score-fix rows: 8 total, including 2 paid, 1 lead, 5 pending.
- Monitor signups: 2 total, both checked; 1 low-score, 1 high-score.
- MCP analytics, last 7 days: 29,771 `tools/list`, 9,639 `initialize`, 621 `tools/call`, 0 unknown tool names.
- MCP tool calls: `search_agents=389`, `check_url=78`, `get_site_details=73`, `get_stats=21`, `submit_site=18`, `verify_mcp=16`, `find_mcp_servers=12`, `recent_additions=5`, `get_top_sites=5`, `list_categories=4`.
- Traffic, last 14 days: top page `/` had 3,735 hits; machine-readable commerce/discovery routes dominate visible top pages, including `/.well-known/commerce.json=468`, `/api/v1/catalog=118`, `/api/v1/checkout=109`, `/api/v1/quote=109`.
- The 8bitconcepts shared social post ledger has 3 entries and no NHS-related entry.

## Scout conclusions

The next marketing movement should not be another generic launch post. Existing drafts are stale and a refresh row already exists. The live signal is site-owner and agent-developer demand: agents are calling `check_url`, `submit_site`, and commerce/quote routes, while the score-fix table has three fresh real pending candidates and no real paid/lead fulfillment yet.

New rows were appended to `ops/sweeper/marketer-inbox.jsonl` for:

- a score-fix abandonment brief using only redacted aggregate evidence,
- an agent-commerce discovery target list from the machine-readable route traffic,
- a check_url to monitor conversion test,
- a current MCP usage proof brief that can later feed public copy after social duplicate locks and channel identity checks.
