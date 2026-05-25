# Payment Checkout Source Readiness

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
- Public categories: `developer=1,230`, `ai-tools=905`, `data=399`,
  `finance=195`, `productivity=171`, `ecommerce=146`, `security=113`,
  `health=59`, `jobs=26`, `education=21`, `news=12`, and `spam=1`.
- Live public surfaces returned 200: `/score`, `/monitor`, `/report`,
  `/newest`, `/top`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`,
  `/api/v1/catalog`, `/llms.txt`, `/openapi.yaml`, and `/feed.xml`.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card claims
  remain blocked.
- MCP `tools/list` returned 11 tools.
- Aggregate MCP analytics, 7 days: `tools/list=172,953`, `initialize=25,492`,
  and `tools/call=371`.
- Aggregate MCP tool calls, 7 days: `search_agents=146`, `check_url=86`,
  `get_site_details=56`, `get_stats=20`, `submit_site=20`,
  `verify_mcp=14`, `find_mcp_servers=8`, `list_categories=7`,
  `recent_additions=5`, and `get_top_sites=5`.
- Sanitized aggregate query themes included payment, checkout, commerce,
  pricing, subscription, quote, x402, model gateways, function calling,
  hardware/runtime, and publisher/feed monitoring.
- Aggregate traffic, 168 hours: `/=3,383`, `/badge/xquik.com.svg=2,629`,
  `/.well-known/commerce.json=1,342`, `/site/xquik.com=1,065`,
  `/.well-known/ai-plugin.json=609`, `/llms.txt=436`,
  `/openapi.yaml=374`, `/api/v1/catalog=320`, `/robots.txt=292`,
  `/api/v1/checkout=256`, `/api/v1/quote=256`, `/api/v1/search=219`,
  `/favicon.ico=178`, and `/api/v1/submit=145`.
- Public ecommerce top-list examples: `budgetfitter.uk=100`,
  `rettfrabonden.com=100`, `skillboss.co=100`, `ai.immoswipe.ch=95`,
  `packrift.com=80`, `can-tap-verified.com=80`, `businesshotels.com=75`,
  `store.farcomindustrial.com=75`, `la-palma24.net=75`,
  `maplebridge.io=70`, `photo-fotograf.com=70`, and `freetv-app.com=60`.
- Public finance top-list examples: `terminalfeed.io=100`,
  `chartlibrary.io=100`, `prereason.com=100`, `devdrops.run=95`,
  `razorpay.com=90`, `ticksurfers.com=90`, `lendtrain.com=85`,
  `debridge.com=80`, `emc2ai.io=80`, `fiasignals.com=75`,
  `mcp.frihet.io=75`, and `bullrundata.com=70`.
- Latest local monitor worker proof remains the 2026-05-18 clean run:
  one due monitor processed and completed without quarantine.
- High-score score-fix route check: `/fix/nothumansearch.ai` returned the
  already-meets-target handoff. Partial-score check: `/fix/manifest.ly`
  returned the paid remediation intake.

## Segment

This segment is narrower than the older agent-commerce and API monetization
briefs. It is for payment processors, checkout tools, subscription platforms,
commerce API owners, pricing/quote API owners, marketplace-data products, and
agent-commerce sellers whose public surfaces agents may inspect before
quoting, buying, comparing, or integrating.

Useful owner-side angle:

- `llms.txt` should describe payment, checkout, quote, subscription, API, and
  unsupported-rail boundaries.
- OpenAPI or equivalent API docs should expose checkout/session/quote/catalog
  contracts where public.
- Commerce and agent manifests should distinguish supported rails from
  unsupported rails, including x402, ACP, SPT, MPP, Link, and Stripe-style
  checkout handoffs.
- Catalog, quote, checkout, refund/contact, fulfillment, and pricing metadata
  should be machine-readable enough for an agent to decide what is safe to do.
- Free monitor registration is the right next step for high-score owners.
- Partial-score owners should start with `/score` and a missing-surface
  checklist before any paid remediation.

## Draft Brief

Agents are not only searching for APIs. They are reading commerce manifests,
catalogs, quote endpoints, checkout handoffs, pricing pages, and subscription
metadata.

The useful claim is narrow: Not Human Search can show whether a payment or
commerce surface is inspectable by another agent before that agent tries to
quote, compare, or start checkout.

High-score payment and commerce owners should use the public report, badge, and
free monitor path. Partial-score owners should run `/score`, fix the missing
machine-readable surfaces, and only then consider remediation.

## Owner Routing

- High-score payment, checkout, marketplace, or API-commerce owners: route to
  free monitor registration, public report sharing, and badge/report proof.
- Partial-score owners: route to `/score` first, then a missing-surface
  checklist before score-fix remediation.
- API-heavy callers: route to API-key/catalog surfaces only when NHS docs
  remain useful.
- A2A or Agent Card claims stay blocked until `/.well-known/agent-card.json`
  exists.

## Claims To Avoid

Do not claim payment reliability, checkout conversion, pricing accuracy,
subscription correctness, refund quality, fulfillment reliability, product
quality, inventory accuracy, x402/ACP/SPT/MPP support for NHS, completed
payments, revenue, customer demand, private demand, paid leads, endorsement,
seller certification, paid placement, preferred inclusion, A2A support while
`/.well-known/agent-card.json` is 404, or score-methodology bypass.
