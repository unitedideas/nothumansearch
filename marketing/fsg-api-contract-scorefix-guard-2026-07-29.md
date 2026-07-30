# FSG API-contract score-fix guard

Date: 2026-07-29
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, crawl, checkout, or global-queue write was performed. This is a
sanitized product/sales handoff for a later NHS operator.

## Fresh evidence

- The public NHS `recent_additions` MCP tool lists
  `freesoragenerator.com` as an AI-tools site added on 2026-07-24 with a
  cached score of 45/100. It is absent from prior NHS marketing artifacts,
  `outreach/distribution_log.csv`, the marketer inbox, and the checked
  portfolio social-post ledger.
- The NHS profile and badge return HTTP 200. The profile records `llms.txt`,
  structured API, and Schema.org signals; it marks AI plugin, OpenAPI,
  explicit AI-crawler rules, and MCP as missing.
- The public score-fix page returns HTTP 200, projects 45 to 100, and offers
  the $29 report and $199 managed implementation. It labels a literal
  `/api/v1` JSON index as already present.
- The live origin's `/api/v1` currently returns HTTP 404. The site does have
  real JSON API behavior: documented `/api/ai-studio/*` routes exist, and
  read-only requests to authentication-protected API routes return HTTP 401
  with JSON rather than an HTML catch-all. The public `/apidoc` page documents
  execution, task-status, history, authentication, and API-key flows.
- Conventional `/openapi.yaml` and `/openapi.json` routes return HTTP 500.
  `/.well-known/ai-plugin.json`, `/.well-known/mcp.json`, `/mcp`,
  `/.well-known/agent.json`, and `/.well-known/agent-card.json` return 404.
  The current paid checklist's OpenAPI, plugin, MCP, and AI-robots work is
  therefore directionally valid, but its claim that `/api/v1` already exists
  is not.
- The live `robots.txt` returns HTTP 200 with a generic `User-Agent: *` rule
  and API/dashboard exclusions, but no explicit AI-bot directives. The
  profile's missing AI-robots signal is consistent with the origin.
- The origin publishes `llms.txt`, `llms-full.txt`, API documentation,
  pricing, status, and a public support channel. Its machine-readable
  instructions identify the creator handle `@freesoragenerator`. No owner
  contact was attempted.
- Public NHS stats currently report 4,367 sites with average score 38. NHS MCP
  discovery lists 11 tools. Aggregate seven-day use is
  `tools/list=48,851`, `initialize=14,940`, and `tools/call=229`; these are
  discovery-funnel counts, not customer or revenue proof.
- NHS owner surfaces remain live: `/score`, `/monitor`, `/report`, `/mcp`,
  `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`,
  `/api/v1/catalog`, `/api/v1/api-keys/subscribe`, `/llms.txt`,
  `/openapi.yaml`, and `/feed.xml` return HTTP 200.
  `/.well-known/agent-card.json` remains HTTP 404.
- The latest monitor worker completed on 2026-07-27. Aggregate monitor state
  is five active and three quarantined. Aggregate score-fix state is ten real
  candidates still pending; paid and lead rows are test-like only. These are
  private workflow guards, not public demand or revenue claims.

## Decision

Do not contact the FSG owner with the current generated checklist. NHS has
correctly found a structured API surface, but the paid page translates that
signal into a false claim that a literal `/api/v1` index already exists. The
owner-facing offer should distinguish the site's documented, authenticated
API routes from an optional machine-readable API index.

The bounded conversion opportunity is still strong: the site already sells
API access and publishes detailed API docs, while conventional OpenAPI,
AI-plugin, MCP, and explicit AI-crawler surfaces are missing or broken.

## Product/sales handoff

1. Reconcile the structured-API evidence with the owner-facing checklist.
   Name the actual documented `/api/ai-studio/*` contract and stop saying
   `/api/v1` is already present while that route returns 404.
2. Treat `/api/v1` as optional discovery/index work, not an existing asset.
   Keep OpenAPI, AI-plugin, MCP, and explicit AI-robots work scoped to live
   routes and the site's declared access model.
3. Ensure any generated AI-plugin manifest references an OpenAPI document
   that returns 200 and parses successfully; do not point it at the current
   HTTP 500 root routes.
4. Refresh the NHS profile or checklist after the bounded review. Only then
   prepare one deduplicated owner touch through the site's public support
   channel or declared creator handle.
5. Before owner contact, verify the active Foundry/Owl-owned sender identity,
   recheck the portfolio social ledger and distribution history, and take a
   sync-state public-action lock.

## Claims to avoid

Do not imply FSG, its owners, models, API routes, or indexed profile are NHS
customers, partners, endorsements, paid leads, monitor registrations,
badge-install consent, private demand, completed payments, or revenue proof.
Do not claim model quality, video/image quality, provider affiliation, API
uptime, pricing accuracy, OpenAPI availability while the conventional routes
return 500, MCP support, crawler compliance, security/privacy proof, paid
ranking, preferred inclusion, score-methodology bypass, or A2A support while
NHS `/.well-known/agent-card.json` is 404.
