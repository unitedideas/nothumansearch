# agent-tools.cloud MCP directory candidate

Date: 2026-07-29
Status: no submission made

## Why this candidate

- NHS MCP analytics recorded 170 requests from the public `agent-tools.cloud` crawler family in the last seven days.
- `agent-tools.cloud` is a public discovery directory for MCP servers, x402 services, and A2A agents.
- Its public MCP search and unified resource search both returned zero matches for `nothumansearch`.
- Its submission page exposes a no-browser programmatic MCP intake at `POST https://agent-tools.cloud/api/v1/mcp/submit`.
- The directory advertises health checks and MCP safety scanning, which fits NHS's probe-before-use positioning.

## Verified NHS packet

- Name: `Not Human Search`
- MCP URL: `https://nothumansearch.ai/mcp`
- Transport: `streamable-http`
- Website: `https://nothumansearch.ai`
- Contact: `hello@8bitconcepts.com`
- Description: `Search 4,367 agent-ready sites by readiness score with 11 MCP tools for discovery, live URL checks, MCP verification, and free readiness monitoring.`
- Live tool count: 11
- Current tools: `search_agents`, `get_site_details`, `get_stats`, `submit_site`, `check_url`, `verify_mcp`, `register_monitor`, `list_categories`, `get_top_sites`, `recent_additions`, `find_mcp_servers`

## Evidence

- NHS stats: `https://nothumansearch.ai/api/v1/stats` returned 4,367 sites and average score 38.
- NHS live `tools/list` returned the 11 tools above.
- NHS discovery surfaces returned 200: `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/llms.txt`, `/openapi.yaml`, `/api/v1`, and `/api/v1/catalog`.
- `https://agent-tools.cloud/api/v1/mcp/search?q=nothumansearch` returned zero matches.
- `https://agent-tools.cloud/api/v1/resources/search?q=nothumansearch` returned zero matches.
- `https://agent-tools.cloud/submit` documents MCP fields for URL, name, description, transport, and contact.
- No matching entry exists in `outreach/distribution_log.csv`, `ops/sweeper/marketer-inbox.jsonl`, prior marketing artifacts, or the portfolio social ledger.

## Execution guard

Submit only the MCP entry. Do not submit NHS as x402 or A2A: NHS does not declare x402 support, and `https://nothumansearch.ai/.well-known/agent-card.json` still returns 404.

Before submission, refresh the target searches, NHS stats, live MCP `tools/list`, discovery surfaces, and target intake contract. Verify the active Foundry identity, take the sync-state public-action lock, and append the resulting directory URL or API receipt to `outreach/distribution_log.csv`.

Do not claim directory endorsement, safety certification, A2A support, x402/ACP/SPT/MPP support, paid placement, preferred inclusion, private demand, customers, completed payments, or score-methodology bypass.
