# Home Automation Source-Readiness Scout

Run: 2026-05-19T19:20Z
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, checkout completion, or QLimit/global-queue write was
performed. This is a sanitized scout artifact for a later product or channel
operator.

No raw users, emails, API keys, checkout URLs, payment identifiers, private
monitor rows, private score-fix rows, private query logs, buyer data, or raw
customer data are included here.

## Fresh Evidence

Public surfaces checked:

- `/api/v1/stats`: `total_sites=4175`, `avg_score=35`,
  `top_category=developer`.
- `/api/v1/categories`: developer 1,231; ai-tools 904; other 768; data 400;
  finance 196; productivity 173; ecommerce 148; communication 120; security
  115; health 59; jobs 27; education 21; news 12; spam 1.
- `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`,
  `/api/v1/catalog`, `/api/v1`, `/openapi.yaml`, `/score`, `/monitor`,
  `/report`, `/top`, and `/newest`: HTTP 200.
- `/.well-known/agent-card.json`: HTTP 404, so strict A2A/Agent Card claims
  remain gated.

Aggregate admin evidence, sanitized:

- Last 7 days MCP analytics: `tools/list=155888`, `initialize=20182`,
  `tools/call=289`.
- Top called MCP tools: `search_agents=180`, `get_site_details=41`,
  `check_url=16`, `get_stats=12`, `verify_mcp=11`,
  `recent_additions=9`, `get_top_sites=8`, `find_mcp_servers=6`,
  `submit_site=4`, `list_categories=2`.
- MCP query themes included model/API pricing, local news and housing,
  agent skills, RAG/document indexing, scanner/electronics retail,
  hardware/IoT, and `Home Assistant open source home automation updates`.
- Last 168 hours traffic included `/api/v1/submit=147`, `/top=107`,
  `/api/v1=93`, `/.well-known/mcp.json=92`, `/guide=89`, `/newest=83`,
  `/score=78`, and `/api/v1/check=60`, alongside higher-volume manifest,
  badge, catalog, quote, and checkout routes.

Public index boundary:

- `home-assistant.io`, `nodered.org`, `esphome.io`, and `zigbee2mqtt.io`
  returned 404 on public `/site/{host}` pages in this run.
- This means the useful next action is not a public claim about coverage. It is
  a gated owner-channel/research packet or a later targeted discovery/import
  handoff for smart-home and automation projects with actual agent-readable
  surfaces.

## Read

Home automation and smart-home projects are a good fit for NHS only when the
marketing stays source-readiness focused. Agents need stable docs, install
metadata, release/update feeds, device/service capability boundaries, local API
contracts, security advisories, and monitorable machine-readable files before
they can safely recommend or operate against a home-automation stack.

The current public index does not yet expose the obvious Home Assistant,
Node-RED, ESPHome, or Zigbee2MQTT profiles. A later operator should either:

1. Build a small no-public-action target list from public project docs and
   score each candidate through approved discovery paths, or
2. Use this as a channel post about what smart-home projects should expose to
   agents, without naming unindexed projects as NHS-covered examples.

## Candidate Copy Boundary

Usable phrasing:

> Smart-home docs are becoming agent inputs. If an assistant can find release
> notes but not the API contract, install metadata, device boundary, or monitor
> surface, it still has to guess.

Avoid:

- Claims that Home Assistant, Node-RED, ESPHome, Zigbee2MQTT, or any listed
  project is indexed, endorsed, a customer, a paid lead, or private demand.
- Claims of device compatibility, hardware reliability, electrical safety,
  local-network safety, security certification, privacy compliance, uptime,
  release freshness, install correctness, A2A support, x402/ACP support, paid
  ranking placement, preferred inclusion, or score-methodology bypass.

## Execution Gate

Before implementation or external use:

1. Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top` for relevant
   categories, `/score`, `/monitor`, `/guide`, `/newest`, `/llms.txt`,
   `/.well-known/mcp.json`, `/.well-known/agent.json`,
   `/.well-known/agent-card.json`, `/.well-known/commerce.json`,
   `/.well-known/ai-plugin.json`, `/api/v1/catalog`, `/api/v1/quote`,
   `/api/v1/checkout`, `/api/v1`, `/openapi.yaml`, aggregate
   `/api/v1/admin/mcp?days=7`, and aggregate
   `/api/v1/admin/traffic?hours=168`.
2. Re-check public `/site/{host}` pages for any named project before using it
   as an NHS example.
3. Verify active Foundry/Owl-owned account identity before public use.
4. Check `marketing/social-post-ledger.json` if present,
   `outreach/distribution_log.csv`, and sync-state public-action locks.
5. Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from
   `unitedideas`.
