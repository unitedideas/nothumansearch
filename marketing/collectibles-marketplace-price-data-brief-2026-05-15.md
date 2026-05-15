# Collectibles Marketplace Price-Data Readiness Brief

Date: 2026-05-15
Source agent: business-marketer-not-human-search

## Scout Signal

Aggregate MCP usage for the last 7 days included multiple collectibles and marketplace price-data searches:

- `TCGPokemon card price`
- `charizard ex mega dream japanese price`
- `Pokemon card game news announcements 2026`
- `eBay Japan Yahoo Auction Mercari Pokemon card sell`

This is a source-readiness signal only. It is not customer demand, revenue, endorsement, price accuracy, or evidence that NHS has current collectibles pricing coverage.

## Live NHS Baseline

Public surfaces checked during this run:

- `/api/v1/stats`: 4,174 indexed sites, average score 35, top category `developer`
- `/api/v1/categories`: `ecommerce` 152 sites at average score 41, `data` 403 at 32, `finance` 200 at 40
- `/.well-known/mcp.json`: 11 tools, including `search_agents`, `get_site_details`, `check_url`, `get_top_sites`, and `recent_additions`
- `/api/v1/catalog`: score-fix and paid API plans are machine-readable
- `/llms.txt`: public category list and API key subscription handoff are current

Public top-list examples that are adjacent but not proof of collectibles coverage:

- `api.boostedchat.com` - score 95, data/travel marketplace API
- `api.socialintel.dev` - score 90, data/API lead source
- `budgetfitter.uk` - score 100, ecommerce discount-code surface
- `skillboss.co` - score 100, agent-wallet/tool marketplace
- `terminalfeed.io` - score 100, market-data feed
- `chartlibrary.io` - score 100, finance chart-pattern data

## Safe Channel Angle

Collectors and resale agents do not need another generic search result. They need sources that expose:

- Stable product identifiers and variants
- Machine-readable price or market data
- OpenAPI or structured API docs
- Clear freshness boundaries
- MCP or probeable tool surfaces for agent workflows
- Monitorable score profiles so site owners know when agent-readiness regresses

NHS can safely position itself as a way to discover and verify agent-readable price-data sources before an agent relies on them. Do not claim NHS validates card prices, predicts resale value, certifies marketplaces, or has comprehensive collectibles coverage.

## Draft Copy Seed

Agents searching for Pokemon card prices are really searching for source contracts: stable IDs, fresh market data, and a machine-readable path that will not collapse into HTML scraping.

Not Human Search is useful at the layer before the price lookup. It checks whether a marketplace, data API, or catalog source exposes agent-readable surfaces: OpenAPI, structured APIs, MCP, llms.txt, plugin manifests, robots rules, and schema.

Current index: 4,174 sites. Ecommerce has 152 sites at average score 41; data has 403 at average score 32; finance has 200 at average score 40.

Use it to find sources worth probing. Do not treat the score as price accuracy, data freshness, or marketplace endorsement.

## Publication Guardrails

Before any external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=ecommerce&limit=8`, `/api/v1/top?category=data&limit=8`, `/llms.txt`, and `/.well-known/mcp.json`
- Check `outreach/distribution_log.csv`, `marketing/social-post-ledger.json` if present, and sync-state public-action locks
- Verify active account identity for the selected channel
- Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`
- Do not claim private demand, completed payments, revenue, customer endorsement, pricing accuracy, data freshness, collectibles valuation, preferred inclusion, paid ranking placement, ACP/x402 support, or score-methodology bypass
