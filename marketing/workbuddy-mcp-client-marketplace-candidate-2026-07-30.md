# WorkBuddy MCP client and marketplace candidate

Date: 2026-07-30
Status: no submission, account action, outreach, or public post performed

## Why this segment

- NHS aggregate MCP analytics recorded 204 requests from the named WorkBuddy
  client family in the last seven days.
- WorkBuddy's official MCP guide says the client supports MCP through a
  visual configuration flow, user-level `~/.workbuddy/mcp.json`, and
  project-level `.workbuddy/mcp.json`.
- The same official guide points users to the Tencent Cloud MCP Marketplace.
  The public marketplace currently exposes 1,053 listings across search,
  developer tools, databases/files, payments, documents, browser automation,
  and other categories.
- Exact-match public searches of the marketplace and WorkBuddy documentation
  did not surface a Not Human Search listing. Treat that as a candidate signal,
  not definitive proof of absence; the later execution worker must repeat the
  marketplace's own logged-in search before submission.

Official surfaces:

- WorkBuddy MCP guide:
  `https://www.codebuddy.cn/docs/workbuddy/From-Beginner-to-Expert-Guide/Function-Description/MCP-Guide`
- WorkBuddy:
  `https://www.workbuddy.ai/`
- Tencent Cloud MCP Marketplace:
  `https://cloud.tencent.com/developer/mcp`

## Verified NHS packet

- Name: Not Human Search
- MCP endpoint: `https://nothumansearch.ai/mcp`
- Transport: streamable HTTP
- Registry id: `ai.nothumansearch/search`
- Registry manifest version: `1.7.1`
- Website: `https://nothumansearch.ai`
- Operator contact: `hello@8bitconcepts.com`
- Public stats: 4,367 indexed sites; average score 38
- Live tool count: 11
- Tools: `search_agents`, `get_site_details`, `get_stats`, `submit_site`,
  `check_url`, `verify_mcp`, `register_monitor`, `list_categories`,
  `get_top_sites`, `recent_additions`, `find_mcp_servers`

A bounded compatibility smoke using `User-Agent: WorkBuddy/5.3.5` completed
both MCP `initialize` and `tools/list` against the live NHS endpoint.
`initialize` negotiated protocol version `2025-06-18`; `tools/list` returned
all 11 tools. This proves the public endpoint accepted the tested request
shape. It does not prove WorkBuddy endorsement, marketplace acceptance, user
adoption, or successful real-user tool invocation.

Live NHS discovery surfaces returned HTTP 200 for `/mcp`,
`/.well-known/mcp.json`, `/.well-known/agent.json`, `/llms.txt`,
`/openapi.yaml`, `/api/v1`, `/monitor`, `/score`, and `/report`.
`/.well-known/agent-card.json` returned HTTP 404, so A2A and Agent Card
positioning remains blocked.

## Gated execution paths

Use exactly one path after fresh verification:

1. Prefer a Tencent Cloud MCP Marketplace listing if an existing
   Foundry/Owl-owned publisher identity can be verified and the intake accepts
   a remote streamable-HTTP server.
2. If the marketplace does not accept this remote shape but an existing
   Foundry/Owl-owned WorkBuddy community identity is available, use one
   technical operator-channel note with the tested endpoint and install
   metadata.
3. If neither identity exists, record the exact account-access blocker and
   stop. Do not create an account from this recurring worker.

Before any external action, search the logged-in marketplace for the registry
id, endpoint, brand, and domain; check the portfolio social ledger,
`outreach/distribution_log.csv`, and sync-state public-action locks; verify the
active account identity; claim a public-action lock; then perform at most one
submission or note and record the receipt or public URL.

## Positioning boundary

Safe positioning:

> Not Human Search gives MCP clients a machine-readable way to search
> agent-ready sites, inspect public readiness signals, verify live MCP
> endpoints, and register free score monitoring.

Do not claim WorkBuddy or Tencent endorsement, a partnership, marketplace
acceptance, customer demand, real-user tool success, completed payments,
revenue, certification, crawler compliance, security approval, uptime proof,
paid placement, preferred inclusion, A2A support while the Agent Card route is
404, or a score-methodology bypass. Do not publish raw user-agent logs, private
monitor rows, private score-fix rows, emails, payment identifiers, checkout
URLs, API keys, or customer identifiers.
