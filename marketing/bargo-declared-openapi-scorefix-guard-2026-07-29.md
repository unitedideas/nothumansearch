# Bargo declared-OpenAPI score-fix guard

Date: 2026-07-29
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, crawl, checkout, or global-queue write was performed. This is a
sanitized product/sales handoff for a later NHS operator.

## Fresh evidence

- The public NHS `recent_additions` MCP tool lists `bargo.ai` as a developer
  site added on 2026-07-27 with a cached score of 55/100. It is not present in
  `outreach/distribution_log.csv`, the NHS marketer inbox, prior NHS marketing
  artifacts, or the checked portfolio social-post ledger.
- The public profile returns HTTP 200 and detects `llms.txt`, a structured API,
  Schema.org, and MCP. It marks AI plugin, OpenAPI, and explicit AI-crawler
  rules as missing.
- The public score-fix page returns HTTP 200, projects 55 to 100, and offers a
  $29 report or $199 implementation. Its proposed AI plugin points to
  `https://bargo.ai/openapi.yaml`, and its implementation checklist proposes
  creating a root `openapi.yaml`.
- The live origin redirects to `www.bargo.ai`. Its `llms.txt` declares a real
  OpenAPI 3.0.3 contract at
  `/free-apis/congress/openapi.json`; that document returns HTTP 200 and has
  six paths. Conventional `/openapi.yaml` and `/openapi.json` routes return
  HTTP 404.
- The origin also exposes `/.well-known/mcp.json`, a separate MCP server card,
  the full `/mcp` service, and a focused Congress-trades MCP endpoint.
  Read-only `tools/list` against the full `/mcp` service succeeds with the
  required streamable-HTTP Accept header and returns 51 tools. The MCP
  manifest says initialization and `tools/list` are open while tool calls
  require an API key.
- The root `/api/v1` returns HTTP 401 rather than a public JSON index. NHS
  nevertheless records the structured-API signal, so the immediate conversion
  conflict is the missed declared OpenAPI path and the generated plugin's
  nonexistent root-spec URL, not MCP or API absence.
- The public contact page exposes `support@bargo.ai`. No owner contact was
  attempted.
- Public NHS stats report 4,388 sites with average score 38. Live NHS MCP
  discovery lists 11 tools. Aggregate seven-day use is
  `tools/list=48,855`, `initialize=14,977`, and `tools/call=228`;
  `get_top_sites=97` and `recent_additions=24`. These are discovery-funnel
  counts, not customer or revenue proof.
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

Do not contact the Bargo owner or use this fresh profile in score-fix proof
until NHS follows the OpenAPI URL declared in the site's `llms.txt`. The
current paid checklist can sell creation of a contract that already exists,
then generate an AI-plugin manifest pointing to a root spec that returns 404.

The useful owner outcome may be a conventional-path compatibility alias plus
an AI-plugin manifest that references the existing Congress-trades contract.
That is materially different from selling a new OpenAPI implementation.

## Product/sales handoff

1. Run one bounded path-resolution review for the OpenAPI URL declared in
   Bargo's live `llms.txt`, plus the MCP URLs declared in its MCP manifests.
2. Decide whether valid declared non-root OpenAPI documents qualify for NHS
   scoring or require a conventional root alias.
3. If the existing document qualifies, refresh the profile and remove
   OpenAPI creation from the paid checklist. If a root alias is required,
   describe the work as discoverability compatibility rather than API design.
4. Ensure any generated `ai-plugin.json` references a live OpenAPI URL. Do not
   generate the currently proposed `https://bargo.ai/openapi.yaml` reference
   while that route returns 404.
5. Only after the profile and paid checklist agree, prepare one deduplicated
   owner touch through the canonical public channel. Verify active account
   identity and take the sync-state public-action lock first.

## Claims to avoid

Do not imply Bargo, its owners, its datasets, or its indexed profile are NHS
customers, partners, endorsements, paid leads, monitor registrations,
badge-install consent, private demand, completed payments, or revenue proof.
Do not claim market-data accuracy, trading performance, investment advice,
SEC-data completeness, API uptime, MCP certification, security/privacy proof,
human traffic, paid ranking, preferred inclusion, score-methodology bypass,
or A2A support while `/.well-known/agent-card.json` is 404.
