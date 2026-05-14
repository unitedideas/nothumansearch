# NHS Jobs Platform Owner Brief

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-14T07:08Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, or QLimit/global-queue write was performed. This artifact is a channel brief for a later gated operator. External use still requires active account verification, duplicate-fingerprint checks, and a sync-state public-action lock.

## Live Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4170`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. The jobs bucket has `26` sites with average score `41`.
- `https://nothumansearch.ai/api/v1/top?category=jobs&limit=8`: public jobs examples range from a Foundry-owned 100/100 dogfood reference to third-party job and recruiting surfaces scoring 75, 65, 50, and 45.
- `https://nothumansearch.ai/llms.txt`: jobs is listed as a public category; `other` and `spam` are audit-only.
- `https://nothumansearch.ai/.well-known/mcp.json`: category parameter copy matches the public/audit-only split.

Public top-category examples observed during preparation:

- `aidevboard.com` - score 100, Foundry-owned dogfood reference with all seven signals present.
- `jseek.co` - score 75, job-tracking surface with `llms.txt`, AI plugin, OpenAPI, AI-friendly robots, and Schema.org; missing structured API and MCP.
- `himalayas.app` - score 65, remote jobs and AI job-search tools with `llms.txt`, AI plugin, structured API, and Schema.org; missing OpenAPI, MCP, and AI-friendly robots.
- `ctojobshq.com`, `reed.co.uk`, `upstaff.com`, and `ziprecruiter.com` - score 50 examples with partial public readiness surfaces.

Aggregate private surfaces checked without raw row output:

- Score-fix aggregate: 11 total rows; real-candidate pending remains `2`, both `dot_com` and `7_29d`; no real paid or real lead row was exposed in committed artifacts.
- Monitor aggregate: `active=1`, `quarantined=1`, quarantine reason `bounded rerun still zero score`.
- Monitor action aggregate, last 30 days: `request_score_rerun=1`, `keep_quarantined=1`.

No raw user identifiers, private customer rows, API keys, checkout URLs, or private query logs were written.

## Brief Copy

Subject/heading:

`26 jobs and recruiting sites are agent-readable. Most still stop short of a full agent workflow.`

Short post:

Not Human Search currently tracks 26 jobs and recruiting sites in the public jobs bucket.

The top result is AI Dev Board, which is Foundry-owned dogfood and should be labeled that way when used as proof: it exposes `llms.txt`, AI plugin metadata, OpenAPI, a structured REST API, MCP, AI-friendly robots policy, and Schema.org.

The more useful market signal is the third-party gap below it. Several job boards and recruiting surfaces already expose enough structure to be useful to agents, but most still miss one or more of the pieces that make a complete agent workflow possible:

1. A public `llms.txt` that tells agents what the site offers.
2. OpenAPI for search, company, job, or alert endpoints.
3. Structured API responses that agents can call without scraping HTML.
4. MCP only where real operational tools exist.
5. Monitoring so deploys do not silently remove the agent-facing surface.

Search the public jobs bucket:

`https://nothumansearch.ai/api/v1/top?category=jobs&limit=25`

Check a jobs or recruiting site:

`https://nothumansearch.ai/score`

## Owner/Buyer Angle

This is for job boards, ATS-adjacent platforms, recruiting data vendors, and hiring-market tooling. The sell is not ranking placement. The sell is making jobs, companies, feeds, and alert surfaces legible to agents, then monitoring them so crawler, applicant, and sourcing workflows keep working after deploys.

## Publication Guard

Before external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=jobs&limit=8`, `/llms.txt`, and `/.well-known/mcp.json`.
- Treat `aidevboard.com` as Foundry-owned dogfood, not third-party market proof.
- Check the shared social ledger for duplicate fingerprints.
- Verify the active account identity for the selected channel.
- Take a sync-state public-action lock.
- Do not claim private demand, revenue, conversion, hiring outcomes, paid ranking placement, preferred inclusion, or score-methodology bypass.
