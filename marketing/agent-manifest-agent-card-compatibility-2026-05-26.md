# Agent Manifest to Agent Card Compatibility

Run: 2026-05-26
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, checkout completion, or QLimit/global-queue write was
performed. This is a sanitized scout artifact for a later gated operator.

No raw users, emails, API keys, checkout URLs, payment identifiers, private
monitor rows, private score-fix rows, private query logs, raw user-agent
strings, buyer data, or customer identifiers are included here.

## Evidence

- Public categories remain current at `developer=1230`, `ai-tools=904`,
  `other=774`, `data=402`, `finance=192`, `productivity=171`,
  `ecommerce=149`, `communication=118`, `security=113`, `health=59`,
  `jobs=26`, `education=21`, `news=12`, and `spam=1`.
- Live public probes returned 200 for `/.well-known/agent.json` and `/mcp`.
- Live public probe returned 404 for `/.well-known/agent-card.json`.
- Aggregate MCP analytics, 7 days: `tools/list=176373`,
  `initialize=26069`, and `tools/call=373`.
- Aggregate MCP tool calls, 7 days: `search_agents=146`, `check_url=86`,
  `get_site_details=55`, `get_stats=27`, `submit_site=20`,
  `verify_mcp=13`, `list_categories=7`, `find_mcp_servers=7`,
  `recent_additions=5`, `register_monitor=4`, and `get_top_sites=3`.
- Aggregate traffic, 168 hours: `/=3408`, `/badge/xquik.com.svg=2647`,
  `/.well-known/commerce.json=1350`, `/site/xquik.com=1106`,
  `/.well-known/ai-plugin.json=609`, `/llms.txt=442`,
  `/openapi.yaml=393`, `/api/v1/catalog=324`, `/api/v1/checkout=257`,
  `/api/v1/quote=257`, `/api/v1/search=223`, `/api/v1/submit=146`,
  `/digest=128`, `/about=89`, `/.well-known/agent.json=78`, and
  `/api/v1=77`.
- Aggregate referrers, 168 hours: `https://nothumansearch.ai/=1968`,
  `https://google.com=569`, `https://nothumansearch.com/=146`,
  `https://www.nothumansearch.ai/=133`, and
  `https://www.nothumansearch.com/=131`.
- Latest monitor worker proof remains 2026-05-25: five due monitors processed,
  two first-check zero-score monitors quarantined, two first-check partial or
  low-score checks recorded, and one stable high-score monitor confirmed.

## Segment

This is a compatibility packet, not a submission packet. Agent and commerce
manifests are receiving material automated traffic, but Agent Card remains
absent. Some A2A and agent-directory surfaces accept or look for
`/.well-known/agent.json`; others require `/.well-known/agent-card.json`.

The useful next step is a gated product handoff or directory-readiness test:
keep existing `agent.json`, MCP, OpenAPI, catalog, commerce, and API-root
claims intact, but do not submit to Agent Card or A2A directories until the
live Agent Card path exists and is synchronized with the current discovery
surface.

## Draft Operator Note

Not Human Search is already machine-readable through MCP, OpenAPI, `llms.txt`,
`agent.json`, commerce metadata, catalog, quote, checkout, and the API root.

The gap is compatibility, not positioning: A2A-style directories are splitting
between `agent.json` and Agent Card paths. Until
`/.well-known/agent-card.json` is live, the safe directory stance is
`agent.json`/MCP/OpenAPI-ready, not A2A-ready.

## Next Gated Action

Prepare exactly one gated product-handoff or directory-readiness test that:

- Confirms whether the target directory accepts `/.well-known/agent.json`,
  requires `/.well-known/agent-card.json`, or supports both.
- Refreshes `/api/v1/stats`, `/api/v1/categories`, `/mcp` JSON-RPC
  `tools/list`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/agent-card.json`, `/.well-known/commerce.json`,
  `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`,
  `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`,
  `/score`, `/monitor`, `/report`, aggregate admin MCP/traffic data, and
  the latest monitor worker proof.
- Blocks A2A and Agent Card claims until the live Agent Card path exists and
  matches the repo's current public discovery surfaces.

## Claims To Avoid

Do not imply A2A support, Agent Card support, directory endorsement, customer
demand, private demand, completed payments, revenue, uptime proof, crawler
compliance, legal permission, SEO lift, x402/ACP/SPT/MPP support for NHS,
paid placement, preferred inclusion, or score-methodology bypass.

Do not publish raw user-agent strings, raw MCP queries, private monitor rows,
raw checkout URLs, payment identifiers, buyer emails, private score-fix rows,
or private customer identifiers.
