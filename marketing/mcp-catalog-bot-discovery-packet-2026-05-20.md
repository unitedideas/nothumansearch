# MCP Catalog Bot Discovery Packet - 2026-05-20

Scope: one Not Human Search marketing scout segment. Artifact only; no outreach, directory submission, account creation, browser action, product-code edit, deploy, or crawl was performed.

## Evidence

- `/api/v1/stats`: `total_sites=4175`, `avg_score=35`, `top_category=developer`.
- `/api/v1/categories`: `developer=1231`, `ai-tools=904`, `data=400`, `finance=196`, `productivity=173`, `ecommerce=148`, `security=115`.
- `/.well-known/mcp.json`: HTTP 200; advertises 11 tools and streamable HTTP endpoint `https://nothumansearch.ai/mcp`.
- `/mcp` GET: HTTP 200; advertises the same endpoint, transport, setup command, and 11-tool list.
- `/.well-known/agent.json`: HTTP 200; advertises MCP, OpenAPI, catalog, quote, checkout, and API-key subscribe surfaces.
- `/.well-known/agent-card.json`: HTTP 404; strict A2A Agent Card claims remain blocked.
- `/api/v1/catalog`: HTTP 200; lists score-fix plus Starter, Pro, and Scale API subscriptions.
- `/api/v1`: HTTP 200; lists search, site, submit, check, monitor, API-key, catalog, quote, checkout, top, stats, and verify-MCP routes.
- `/openapi.yaml`: HTTP 200; includes catalog and quote paths.
- `/llms.txt`: HTTP 200; advertises 4,175+ sites, scoring methodology, 11 MCP tools, API docs, paid API-key plans, and monitor registration.

Aggregate MCP analytics for the last 7 days:

- `tools/list=158791`, `initialize=20706`, `tools/call=276`.
- Top tool calls: `search_agents=175`, `get_site_details=38`, `check_url=16`, `get_stats=12`, `verify_mcp=8`, `get_top_sites=8`, `recent_additions=8`, `find_mcp_servers=5`, `submit_site=4`, `list_categories=2`.
- Bot/catalog user agents in aggregate: `MCP-Catalog-Bot/1.0=102`, `MCPScoringEngine/1.0=96`, `AgentFinderBot/0.3 (+https://agentfinder.dev/bot)=68`.
- Claude Code clients are present in aggregate (`claude-code/2.1.140` through `2.1.144` variants).

Aggregate traffic for the last 168 hours:

- `/.well-known/commerce.json=1624`
- `/.well-known/ai-plugin.json=740`
- `/llms.txt=482`
- `/openapi.yaml=443`
- `/api/v1/catalog=350`
- `/api/v1/quote=319`
- `/api/v1/checkout=319`
- `/.well-known/mcp.json=92`
- `/api/v1=92`
- `/api/v1/search=91`
- `/guide=91`
- `/score=79`
- `/api/v1/check=60`

## Segment

MCP catalog and scoring bots are already inspecting NHS, mostly through discovery surfaces and `tools/list`. The useful marketing action is a bot-friendly directory packet, not another broad social post.

Best-fit channel families:

1. MCP catalog and MCP scoring directories that accept a service URL, MCP endpoint, manifest, OpenAPI spec, and contact email.
2. Agent finder and service discovery directories that rank services by machine-readable surfaces.
3. Claude/Codex/MCP tooling lists that prefer installable MCP servers and API discovery utilities.

## Packet

Short listing:

> Not Human Search is a search engine and MCP server for agent-ready websites, APIs, and tools. It scores 4,175+ sites by seven machine-readable readiness signals: `llms.txt`, AI plugin manifest, OpenAPI, structured API, MCP, AI-friendly robots policy, and Schema.org.

Links:

- Site: `https://nothumansearch.ai`
- MCP endpoint: `https://nothumansearch.ai/mcp`
- MCP manifest: `https://nothumansearch.ai/.well-known/mcp.json`
- Agent manifest: `https://nothumansearch.ai/.well-known/agent.json`
- Commerce/catalog: `https://nothumansearch.ai/api/v1/catalog`
- OpenAPI: `https://nothumansearch.ai/openapi.yaml`
- LLM instructions: `https://nothumansearch.ai/llms.txt`
- Contact: `hello@nothumansearch.ai`

Claude Code install:

```bash
claude mcp add --transport http nothumansearch https://nothumansearch.ai/mcp
```

Tool list:

`search_agents`, `get_site_details`, `get_stats`, `submit_site`, `check_url`, `verify_mcp`, `register_monitor`, `list_categories`, `get_top_sites`, `recent_additions`, `find_mcp_servers`.

## Guardrails

- Do not claim A2A support while `/.well-known/agent-card.json` returns HTTP 404.
- Do not submit to `modelcontextprotocol/*` or `punkpeye/*` from `unitedideas`.
- Do not claim private demand, customer endorsement, paid leads, completed payments, revenue, x402/ACP/MPP support, score-methodology bypass, certification, paid ranking placement, preferred inclusion, or uptime proof.
- Do not publish raw query logs, raw user agents beyond aggregate counts, private monitor rows, private score-fix rows, emails, payment identifiers, checkout URLs, or API keys.
- Before any public action, refresh the live surfaces above, verify active Foundry/Owl-owned account identity, check `outreach/distribution_log.csv`, `marketing/social-post-ledger.json` when applicable, and take a sync-state public-action lock.

