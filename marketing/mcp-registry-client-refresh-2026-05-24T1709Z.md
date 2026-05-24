# MCP Registry and Client Refresh

Run: 2026-05-24T17:09:02Z
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, checkout completion, or QLimit/global-queue write was
performed. This is a sanitized no-submit scout artifact for a later gated
operator.

No raw users, emails, API keys, checkout URLs, payment identifiers, private
monitor rows, private score-fix rows, private query logs, buyer data, or raw
customer data are included here.

## Fresh Evidence

- Public stats: 4,177 indexed sites, average score 35, top category
  `developer`.
- Public categories: `developer=1230`, `ai-tools=904`, `other=781`,
  `data=399`, `finance=195`, `productivity=171`, `ecommerce=146`,
  `communication=119`, `security=113`, `health=59`, `jobs=26`,
  `education=21`, `news=12`, and `spam=1`.
- Live discovery surfaces returned 200: `/llms.txt`,
  `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/api/v1/catalog`, `/api/v1`,
  `/mcp`, `/monitor`, and `/score`.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card
  directory claims remain blocked.
- Aggregate MCP analytics, 7 days: `tools/list=172267`,
  `initialize=26230`, and `tools/call=397`.
- Aggregate MCP tool calls, 7 days: `search_agents=167`, `check_url=85`,
  `get_site_details=58`, `submit_site=21`, `get_stats=20`,
  `verify_mcp=14`, `list_categories=8`, `find_mcp_servers=8`,
  `recent_additions=6`, `get_top_sites=6`, and `register_monitor=4`.
- Aggregate client families included Python HTTP clients, Node clients,
  Cherry Studio, Claude Code, `mcpregistry-bot`, `MCP-Catalog-Bot`,
  `MCPScoringEngine`, `mcp-verify`, and Go HTTP clients.
- Aggregate top pages, 168 hours: `/=3372`,
  `/badge/xquik.com.svg=2554`, `/.well-known/commerce.json=1364`,
  `/site/xquik.com=997`, `/.well-known/ai-plugin.json=620`,
  `/llms.txt=441`, `/openapi.yaml=377`, `/api/v1/catalog=322`,
  `/robots.txt=295`, `/api/v1/checkout=261`, `/api/v1/quote=261`,
  `/api/v1/search=215`, `/api/v1/submit=145`, `/digest=124`,
  `/.well-known/mcp.json=80`, and `/api/v1=80`.
- Latest local monitor-check proof remains 2026-05-18: one due monitor
  processed cleanly, score stayed 100 to 100, and monitor-check completed.

## Segment

MCP registries, MCP scoring clients, catalog bots, and desktop clients are
still using NHS as a machine-readable discovery and readiness source. The
useful next segment is not another broad directory submission. It is a
registry/client maintenance and conversion test:

- keep live `/mcp` `tools/list`, `tools/mcp-registry/server.json`,
  `/.well-known/mcp.json`, `/llms.txt`, `/openapi.yaml`, `/api/v1`, and
  commerce/catalog metadata synchronized before any registry bump;
- route MCP client users to install, search, `check_url`, and `verify_mcp`;
- route site owners who arrive from profile, badge, score, or check paths to
  free monitor registration or score-band remediation;
- keep strict A2A/Agent Card language blocked until the Agent Card path exists.

## Draft Angle

Safe short copy for a gated channel:

`NHS is being crawled by MCP registries, scoring clients, and desktop MCP
tools. The maintenance burden is keeping the machine-readable surfaces in
sync: live tools/list, mcp.json, llms.txt, OpenAPI, API root, catalog, and
commerce metadata. For site owners, a one-time check should route into
monitoring if the score is already high, or a missing-surface checklist if it
is not.`

## Owner Routing

- MCP registry or client users: route to install/search examples, then
  `check_url` and `verify_mcp`.
- High-score site owners: route to free monitor registration, report sharing,
  and badge/report proof.
- Partial-score owners: route to `/score` and a missing-surface checklist
  before score-fix remediation.
- API-heavy callers: route to API-key/catalog surfaces only when the docs
  remain useful.

## Acceptance Before Public Use

1. Refresh `/api/v1/stats`, `/api/v1/categories`, `/mcp` JSON-RPC
   `tools/list`, `tools/mcp-registry/server.json`, `/.well-known/mcp.json`,
   `/.well-known/agent.json`, `/.well-known/agent-card.json`,
   `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`,
   `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`,
   `/openapi.yaml`, `/score`, `/monitor`, `/report`, representative
   `/site/{host}` pages, and high-score plus partial-score `/fix/{host}`
   routes.
2. Refresh aggregate `/api/v1/admin/mcp?days=7` and
   `/api/v1/admin/traffic?hours=168` without writing raw users, raw
   user-agent strings, private queries, buyer data, or identifiers.
3. Verify the active Foundry/Owl-owned account identity before public use.
4. Check `marketing/social-post-ledger.json` if present,
   `outreach/distribution_log.csv`, and sync-state public-action locks.
5. Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from
   `unitedideas`.

## Claims To Avoid

Do not imply registry endorsement, MCP client endorsement, customer demand,
private demand, paid leads, monitor registrations, badge-install consent,
completed payments, revenue, uptime proof, crawler compliance, legal
permission, SEO lift, A2A support while `/.well-known/agent-card.json` is
404, x402/ACP/MPP support for NHS, paid placement, preferred inclusion, or
score-methodology bypass.
