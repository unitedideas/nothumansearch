# Agent-Builder Directory Submission Packet

Date: 2026-05-15
Automation: `business-marketer-not-human-search`
Source agent: `business-marketer-not-human-search`

## Purpose

Prepare a no-submit packet for agent-builder, agent-skills, autonomous-agent, and A2A-adjacent directories that accept discovery tools or service listings.

This packet is not a public submission. A later channel operator must verify account identity, duplicate locks, and any directory-specific rules before posting or submitting.

## Refreshed Evidence

- `https://nothumansearch.ai/api/v1/stats`: 4,175 indexed sites, average score 35, top category `developer`.
- `https://nothumansearch.ai/api/v1/categories`: developer 1,229; ai-tools 901; data 403; finance 200; productivity 172; ecommerce 152; security 115.
- `https://nothumansearch.ai/.well-known/mcp.json`: 11 tools advertised, including `search_agents`, `get_site_details`, `verify_mcp`, `check_url`, `get_top_sites`, `find_mcp_servers`, and `recent_additions`.
- `https://nothumansearch.ai/.well-known/agent.json`: advertises REST API, OpenAPI, MCP endpoint, commerce catalog, quote, checkout, and API-key subscription surfaces.
- `https://nothumansearch.ai/.well-known/commerce.json`: lists score-fix remediation and Starter/Pro/Scale API-key plans.
- `https://nothumansearch.ai/api/v1/catalog`: lists `nhs_geo_fix_my_score`, `nhs_api_starter`, `nhs_api_pro`, and `nhs_api_scale`.
- Admin MCP analytics, last 7 days, aggregate only: `tools/list=126,799`, `initialize=17,015`, `tools/call=368`.
- Top called MCP tools: `search_agents=228`, `get_site_details=47`, `find_mcp_servers=24`, `get_stats=18`, `verify_mcp=15`, `check_url=14`.
- Aggregate MCP query themes included A2A protocol discovery, coding-agent/developer tooling, MCP server lookup, browser automation, Notion/Penpot/design tooling, model/API lookup, finance/trading data, and marketplace/agent-commerce searches.
- Aggregate traffic, last 336 hours: `/.well-known/commerce.json=1,429`, `/api/v1/catalog=319`, `/api/v1/quote=299`, `/api/v1/checkout=299`, `/.well-known/mcp.json=92`, `/api/v1=90`.

No raw user identifiers, payment identifiers, checkout URLs, private query logs, monitor rows, score-fix rows, or customer data were written.

## Directory Fit

Use this packet for directories that accept one of these listing types:

- MCP servers for agents.
- Agent/tool discovery services.
- Agent-skills or autonomous-agent infrastructure.
- Developer tools that help agents find APIs, MCP servers, and machine-readable services.
- Agent-commerce or machine-readable buying surfaces, if the directory accepts Stripe Checkout/Link handoff rather than requiring x402 or ACP.

Do not use this packet for strict conversational-agent directories unless they explicitly accept service/tooling listings.

## Candidate Positioning

Short description:

> Not Human Search is a search engine and MCP server for agent-ready websites. It ranks services, APIs, and tools by public agent-readiness signals such as llms.txt, OpenAPI, structured APIs, MCP, AI-friendly robots rules, and Schema.org.

Longer description:

> Not Human Search helps agents and agent builders discover sources that are ready for non-human users. It exposes a public REST API, an MCP server, OpenAPI, llms.txt, machine-readable commerce metadata, and score reports for indexed sites. The useful boundary is probe-before-use: NHS helps identify whether a site publishes the public surfaces an agent can inspect before scraping or guessing.

Proof links:

- Site: `https://nothumansearch.ai`
- MCP endpoint: `https://nothumansearch.ai/mcp`
- MCP manifest: `https://nothumansearch.ai/.well-known/mcp.json`
- Agent manifest: `https://nothumansearch.ai/.well-known/agent.json`
- Commerce manifest: `https://nothumansearch.ai/.well-known/commerce.json`
- API root: `https://nothumansearch.ai/api/v1`
- OpenAPI: `https://nothumansearch.ai/openapi.yaml`
- llms.txt: `https://nothumansearch.ai/llms.txt`
- Public top developer examples: `https://nothumansearch.ai/api/v1/top?category=developer&limit=5`

## Duplicate And Boundary Checks

Already logged distribution includes broad MCP/A2A/GitHub/awesome-list work:

- `ai-boost/awesome-a2a` PR #87.
- `sing1ee/a2a-directory` PR #20.
- `pab1it0/awesome-a2a` PR #47.
- `isekOS/awesome-a2a-agents` PR #13.
- `forgewebO1/Awesome-A2A` PR #6.
- `0xNyk/awesome-hermes-agent` PR #35.
- Glama connector work and MCP/API/GEO directory work in `outreach/distribution_log.csv`.

Do not resubmit to the same directories from this packet. Use it only for a fresh target or for a maintainer-requested update.

Blocked or risky surfaces:

- `modelcontextprotocol/*` and `punkpeye/*` are excluded for `unitedideas`.
- Strict A2A/Agent Card directories remain blocked until `/.well-known/agent-card.json` exists or the directory accepts the current `/.well-known/agent.json` service manifest.
- Direct browser/form submissions require a later supervised or API-backed channel operator. This recurring worker must not use browser/Computer Use.

## Submission Draft

Title:

`Not Human Search - agent-readiness search and MCP discovery`

Body:

`Not Human Search is a search engine and MCP server for agent-ready websites. It indexes 4,175 services and ranks them by public machine-readable signals: llms.txt, OpenAPI, structured APIs, MCP, AI-friendly robots rules, plugin metadata, and Schema.org.`

`It is useful for agent builders because it exposes the discovery layer directly: REST API, MCP endpoint, OpenAPI, llms.txt, and machine-readable agent/commerce manifests. Agents can search for APIs, MCP servers, and site profiles before falling back to scraping.`

`Proof links:`

`- Site: https://nothumansearch.ai`
`- MCP: https://nothumansearch.ai/mcp`
`- MCP manifest: https://nothumansearch.ai/.well-known/mcp.json`
`- API root: https://nothumansearch.ai/api/v1`
`- OpenAPI: https://nothumansearch.ai/openapi.yaml`
`- llms.txt: https://nothumansearch.ai/llms.txt`

## Do Not Claim

- Do not claim A2A support until an Agent Card path exists or a directory explicitly accepts the current agent manifest.
- Do not claim Hermes endorsement, private demand, completed payments, revenue, customer endorsement, pricing accuracy, benchmark accuracy, certification, paid ranking placement, preferred inclusion, ACP/x402 support for NHS, or score-methodology bypass.
- Do not imply Foundry-owned examples in the `ai-tools` top list are third-party market proof.
- Do not use stale 8,600+ site or old MCP-claim counts from older drafts.

## Next Gated Action

Use this packet to prepare one fresh agent-builder directory submission or community post through a channel operator. The operator must:

1. Refresh `/api/v1/stats`, `/api/v1/categories`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, and `/api/v1/admin/mcp?days=7`.
2. Check `outreach/distribution_log.csv`, social duplicate ledgers if applicable, and sync-state public-action locks.
3. Verify active account identity.
4. Avoid `modelcontextprotocol/*` and `punkpeye/*` from `unitedideas`.
5. Record the public URL or exact blocker after the action.
