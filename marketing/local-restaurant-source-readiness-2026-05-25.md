# Local Restaurant Source Readiness

Run: 2026-05-25
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, checkout completion, or QLimit/global-queue write was
performed. This is a sanitized scout artifact for a later gated operator.

No raw users, emails, API keys, checkout URLs, payment identifiers, private
monitor rows, private score-fix rows, private query logs, raw user-agent
strings, buyer data, or customer identifiers are included here.

## Evidence

- Public stats: 4,179 indexed sites, average score 35, top category
  `developer`.
- Public categories: `developer=1,230`, `ai-tools=905`, `other=782`,
  `data=399`, `finance=195`, `productivity=171`, `ecommerce=146`,
  `communication=119`, `security=113`, `health=59`, `jobs=26`,
  `education=21`, `news=12`, and `spam=1`.
- Live public surfaces returned 200: `/score`, `/monitor`, `/report`,
  `/newest`, `/top`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`,
  `/api/v1/catalog`, `/llms.txt`, `/openapi.yaml`, and `/feed.xml`.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card claims
  remain blocked.
- MCP `tools/list` returned 11 tools.
- Aggregate MCP analytics, 7 days: `tools/list=173,210`,
  `initialize=25,328`, and `tools/call=377`.
- Aggregate MCP tool calls, 7 days: `search_agents=148`, `check_url=86`,
  `get_site_details=59`, `get_stats=20`, `submit_site=20`,
  `verify_mcp=14`, `find_mcp_servers=8`, `list_categories=7`,
  `recent_additions=6`, `get_top_sites=5`, and `register_monitor=4`.
- Sanitized aggregate query themes included local restaurants, halal food,
  local events, city entertainment, product reviews, commerce, pricing,
  model gateways, scanners, hardware, health claims, and publisher feeds.
- Aggregate traffic, 168 hours: `/=3,385`,
  `/badge/xquik.com.svg=2,637`, `/.well-known/commerce.json=1,352`,
  `/site/xquik.com=1,078`, `/.well-known/ai-plugin.json=613`,
  `/llms.txt=440`, `/openapi.yaml=378`, `/api/v1/catalog=322`,
  `/robots.txt=294`, `/api/v1/checkout=258`, `/api/v1/quote=258`,
  `/api/v1/search=222`, `/favicon.ico=177`, `/api/v1/submit=145`,
  `/digest=127`, `/about=91`, `/api/v1=79`, and
  `/.well-known/mcp.json=78`.
- Public `other` top-list examples: `astranl.com=100`, `lobehub.com=95`,
  `agentgrade.com=80`, `surprise-buddy.com=65`, `infinity-folder.org=65`,
  `twzrd.xyz=55`, `proshares.com=50`, `crabbitmq.com=50`,
  `afairway.com=50`, `optixthreatintelligence.co.uk=50`,
  `lightpollutionmap.app=50`, and `fullmoonparty-thailand.com=50`.
- Public ecommerce top-list examples: `budgetfitter.uk=100`,
  `rettfrabonden.com=100`, `skillboss.co=100`, `ai.immoswipe.ch=95`,
  `packrift.com=80`, `can-tap-verified.com=80`,
  `businesshotels.com=75`, `store.farcomindustrial.com=75`,
  `la-palma24.net=75`, `maplebridge.io=70`, `photo-fotograf.com=70`,
  and `freetv-app.com=60`.
- Latest local monitor worker proof remains the 2026-05-18 clean run:
  one due monitor processed and completed without quarantine.
- High-score score-fix route check: `/fix/nothumansearch.ai` returned 200
  with the already-meets-target handoff. Partial-score check:
  `/fix/manifest.ly` returned 200 with the paid remediation intake.

## Segment

This segment is narrower than the older local-events and lifestyle briefs. It
is for local restaurant, food discovery, halal directory, venue, tourism, and
local-business owners whose information agents may inspect before making a
recommendation, itinerary, booking, or local-search answer.

Useful owner-side angle:

- `llms.txt` should describe canonical menu, cuisine, location, hours,
  reservation, delivery, accessibility, and contact sources.
- Schema.org should cover restaurants, local businesses, menus, offers,
  reviews only where owner-controlled, and organization/contact metadata.
- API, feed, catalog, or OpenAPI surfaces should exist only where the owner
  intends agents or partners to read structured local-business data.
- Robots policy should clearly state whether major AI crawlers are allowed.
- Free monitor registration is the right next step for high-score owners.
- Partial-score owners should start with `/score` and a missing-surface
  checklist before any paid remediation.

## Draft Brief

Agents answering local food questions need a source contract, not just a page
to scrape.

For a restaurant, venue, halal directory, or local guide, the machine-readable
surface should say which source is canonical for hours, menu, reservation,
delivery, location, contact, accessibility, and update policy. Not Human Search
does not certify the restaurant or the recommendation. It checks whether an
agent can inspect the public source before trusting it.

High-score local-business owners should use the public report, badge, and free
monitor path. Partial-score owners should run `/score`, fix missing
machine-readable surfaces, and only then consider remediation.

## Owner Routing

- High-score restaurants, venues, local guides, and tourism/local-business
  surfaces: route to free monitor registration, public report sharing, and
  badge/report proof.
- Partial-score owners: route to `/score` first, then a missing-surface
  checklist before score-fix remediation.
- Directory and API-heavy callers: route to API-key/catalog surfaces only
  when NHS docs remain useful.
- A2A or Agent Card claims stay blocked until `/.well-known/agent-card.json`
  exists.

## Claims To Avoid

Do not claim restaurant quality, food safety, halal certification,
reservation availability, delivery reliability, accessibility compliance,
menu accuracy, hours freshness, review truth, local-guide completeness,
tourism fulfillment, customer demand, private demand, paid leads, completed
payments, revenue, endorsement, paid placement, preferred inclusion, A2A
support while `/.well-known/agent-card.json` is 404, x402/ACP/SPT/MPP support
for NHS, or score-methodology bypass.
