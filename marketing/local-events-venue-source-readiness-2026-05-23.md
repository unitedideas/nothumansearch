# Local Events and Venue Source Readiness - 2026-05-23

Run context: `business-marketer-not-human-search` recurring scout. No outreach, posting, browser, Computer Use, deploy, product-code edit, full recrawl, account creation, checkout completion, raw customer readout, or QLimit/global-queue write was performed.

## Fresh Aggregate Signal

- Public stats: `/api/v1/stats` returned `total_sites=4177`, `avg_score=35`, and `top_category=developer`.
- Public category counts: `ecommerce=146 avg_score=41`, `productivity=171 avg_score=39`, `news=12 avg_score=50`, `data=399 avg_score=32`, and `other=780 avg_score=27`.
- Aggregate MCP analytics over 7 days: `tools/list=171172`, `initialize=27686`, and `tools/call=308`.
- Top MCP tool calls: `search_agents=141`, `check_url=58`, `get_site_details=47`, `get_stats=19`, `submit_site=11`, `list_categories=7`, `find_mcp_servers=7`, `recent_additions=6`, `get_top_sites=6`, `verify_mcp=5`, and `register_monitor=1`.
- Aggregate MCP query themes included `entertainment events Mississauga Ontario` alongside publisher, hardware, skill-marketplace, health-claims, and agent-commerce themes. No raw user identifiers were written.
- Aggregate traffic over 168 hours: `/=3332`, `/badge/xquik.com.svg=2538`, `/.well-known/commerce.json=1412`, `/site/xquik.com=950`, `/.well-known/ai-plugin.json=651`, `/llms.txt=446`, `/openapi.yaml=399`, `/api/v1/catalog=322`, `/robots.txt=304`, `/api/v1/quote=273`, `/api/v1/checkout=273`, `/api/v1/search=196`, and `/favicon.ico=179`.
- Live discovery checks returned HTTP 200 for `/score`, `/monitor`, `/report`, `/newest`, `/top`, `/api/v1`, `/api/v1/catalog`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/llms.txt`, `/openapi.yaml`, `/feed.xml`, `/mcp-servers`, `/openapi-apis`, and `/llms-txt-sites`.
- `/.well-known/agent-card.json` returned HTTP 404, so strict Agent Card and A2A-style claims remain gated.

## Public Example Pool

These are public readiness examples or owner-channel targets only. They are not customers, endorsements, paid leads, private demand, monitor registrations, badge installs, completed payments, revenue, or proof of local-events market share.

| Domain | Score | Public readiness pattern | Safe owner route |
|---|---:|---|---|
| `businesshotels.com` | 75 | Business hotel and travel-booking surface in the ecommerce top list. | `/score` first, then missing-surface checklist before remediation; do not claim booking availability or rate accuracy. |
| `simcorner.com` | 55 | Travel connectivity and prepaid SIM/eSIM surface. | `/score` first; route to checklist before any score-fix copy. |
| `rowhint.com` | 50 | Seat/story and venue-adjacent information surface. | `/score` first; no event or seating accuracy claims. |
| `planharmony.com` | 70 | Collaborative trip-planning surface in the productivity list. | `/score` first; monitor/report only after high-score proof. |
| `1trip.app` | 65 | Multi-city travel planning surface. | `/score` first; missing-surface checklist before remediation. |
| `fullmoonparty-thailand.com` | 50 | Event and travel-information surface in the `other` list. | `/score` first; no event-date, ticketing, safety, or travel-fulfillment claims. |

## Useful Angle

City event, venue, travel, and local-discovery owners have a source-readiness problem that is narrower than generic lifestyle or publisher copy. Agents may need to answer whether an event exists, where it happens, whether tickets are available, what the venue policy is, or which source is canonical. A human landing page does not give agents enough stable contract surface to trust the result.

The owner-side contract is:

1. Publish `llms.txt` with source scope, geography, update cadence, and non-coverage boundaries.
2. Expose structured feeds, OpenAPI, or stable API metadata for events, venues, calendars, ticket links, and local guides when those surfaces are meant for agent access.
3. Make pricing, ticketing, affiliate, refund, accessibility, safety, and contact boundaries explicit.
4. Add Schema.org for events, places, offers, articles, and organizations where appropriate.
5. Use free monitor/report/badge proof once the public surface is high-scoring.
6. Use paid remediation only for concrete missing public contracts after `/score`.

Safe short copy:

`Agents answering local event and travel questions need a source contract, not just a page to scrape. Not Human Search checks whether event, venue, ticketing, and local-guide owners expose inspectable surfaces: llms.txt, OpenAPI/API or feeds, plugin metadata, robots policy, Schema.org, MCP where present, and monitorable public profiles. The score is not an event recommendation or ticketing guarantee; it is a checklist for what an agent can verify before trusting the source.`

## Gated Use

Use this for exactly one gated owner-channel touch, post, or product-handoff test for local event guides, venue/ticketing surfaces, travel-planning apps, local publishers, calendar products, or tourism-information owners.

Required refresh before external use:

- `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=ecommerce&limit=12`, `/api/v1/top?category=productivity&limit=12`, `/api/v1/top?category=news&limit=12`, and `/api/v1/top?category=other&limit=12`
- `/score`, `/monitor`, `/report`, representative `/site/{host}` pages
- High-score and partial-score `/fix/{host}` routes
- `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`
- `/api/v1`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, `/feed.xml`
- Aggregate `/api/v1/admin/mcp?days=7` and `/api/v1/admin/traffic?hours=168`

Before public use, verify the active Foundry/Owl-owned account identity, check `marketing/social-post-ledger.json` if present, `outreach/distribution_log.csv`, and sync-state public-action locks, and avoid `modelcontextprotocol/*` plus `punkpeye/*` surfaces from `unitedideas`.

Do not imply local-event, venue, travel, ticketing, tourism, publisher, or profiled domains are customers, partners, endorsements, paid leads, monitor registrations, badge-install consent, private demand, completed payments, revenue, event-date accuracy, ticket availability, price freshness, venue safety, accessibility compliance, travel fulfillment, local-news freshness, affiliate revenue, crawler compliance, SEO lift, uptime proof, A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/MPP support for NHS, paid ranking placement, preferred inclusion, or score-methodology bypass. Do not publish raw user-agent strings, private query logs, raw checkout URLs, payment identifiers, buyer emails, private monitor rows, private score-fix rows, or private customer identifiers.
