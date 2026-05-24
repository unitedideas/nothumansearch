# MCP Registry Bot Manifest Maintenance

Run: 2026-05-24T05:24:00Z
Automation: business-marketer-not-human-search

## Evidence

- Public stats: 4,177 indexed sites, average score 35, top category `developer`.
- Public discovery surfaces returned 200: `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, `/openapi.yaml`, `/monitor`, and `/score`.
- `/.well-known/agent-card.json` returned 404, so A2A or Agent Card directory claims stay blocked.
- JSON-RPC `/mcp` `tools/list` returned 11 tools: `search_agents`, `get_site_details`, `get_stats`, `submit_site`, `check_url`, `verify_mcp`, `register_monitor`, `list_categories`, `get_top_sites`, `recent_additions`, and `find_mcp_servers`.
- In-repo MCP registry manifest is `tools/mcp-registry/server.json`, version `1.7.1`, description `Search the agentic web. 4,100+ sites, 11 tools incl. check_url + verify_mcp for probe-before-use.`
- Aggregate MCP analytics, 7 days: `tools/list` 172,579, `initialize` 27,256, `tools/call` 400.
- Aggregate MCP client families, 7 days: generic Python HTTP client 195,530, `node` 2,306, Cherry Studio 577, `python-requests` 306, Claude Code CLI 276 across visible variants, `MCP-Catalog-Bot` 174, `mcpregistry-bot` 128, `MCPScoringEngine` 82, and `mcp-verify` 70.
- Aggregate tool calls, 7 days: `search_agents` 173, `check_url` 84, `get_site_details` 60, `submit_site` 21, `get_stats` 21, `verify_mcp` 8, `find_mcp_servers` 8, `list_categories` 8, `recent_additions` 7, `get_top_sites` 6, and `register_monitor` 4.
- Aggregate traffic, 168 hours: `/` 3,345, `/badge/xquik.com.svg` 2,522, `/.well-known/commerce.json` 1,382, `/site/xquik.com` 949, `/.well-known/ai-plugin.json` 638, `/llms.txt` 453, `/openapi.yaml` 390, `/api/v1/catalog` 324, `/robots.txt` 306, `/api/v1/search` 208, `/api/v1/submit` 143, `/api/v1` 90, and `/.well-known/mcp.json` 90.

## Segment

MCP registry crawlers and client catalogs are using NHS as a machine-readable discovery source, not just a human directory. The useful marketing motion is a maintenance packet for MCP registry and catalog channels:

- keep `tools/mcp-registry/server.json`, live `/mcp` `tools/list`, `/.well-known/mcp.json`, `/llms.txt`, `/openapi.yaml`, and `/api/v1` synchronized before any registry bump;
- frame NHS as a probe-before-use discovery server with `check_url`, `verify_mcp`, and readiness scoring;
- route MCP client users to install/search examples first;
- route site owners from `check_url`, profile, score, and badge paths to free monitor/report/badge proof or `/score` before remediation;
- keep A2A and Agent Card claims blocked until `/.well-known/agent-card.json` exists.

## Candidate Channels

- Official MCP registry version/description refresh, only after a real tool-surface or count change.
- MCP catalog and MCP scoring surfaces that already crawl NHS manifests.
- MCP client onboarding docs or directory packets for streamable HTTP clients.
- Registry-maintenance social or technical note, gated behind account identity, duplicate checks, and a public-action lock.

## Claims To Avoid

Do not claim registry endorsement, customer demand, private demand, completed payments, revenue, uptime proof, client-vendor approval, A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/MPP support for NHS, paid placement, preferred inclusion, or score-methodology bypass. Do not publish raw user-agent strings, private query logs, checkout URLs, payment identifiers, buyer emails, private monitor rows, private score-fix rows, or private customer identifiers.
