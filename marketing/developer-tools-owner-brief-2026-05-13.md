# NHS Developer Tools Owner Brief

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-13T16:08Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, or QLimit/global-queue write was performed. This artifact is a channel brief for a later gated operator. External use still requires active account verification, duplicate-fingerprint checks, and a sync-state public-action lock.

## Live Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4181`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. The developer bucket has `1233` sites with average score `34`.
- `https://nothumansearch.ai/api/v1/top?category=developer&limit=8`: the top developer examples score 100/100 and expose the full public agent-readiness surface: `llms.txt`, AI plugin manifest, OpenAPI, structured API, MCP, AI-friendly robots policy, and Schema.org.
- `https://nothumansearch.ai/llms.txt`: developer is listed as a public category; `other` and `spam` are audit-only.
- `https://nothumansearch.ai/.well-known/mcp.json`: category parameter copy matches the public developer category and excludes audit-only buckets from promoted discovery inventory.

No raw user identifiers, private customer rows, API keys, checkout URLs, or private query logs were written.

## Brief Copy

Subject/heading:

`1,233 developer-tool sites are agent-readable. The highest-scoring ones expose the whole tool surface.`

Short post:

Not Human Search currently tracks 1,233 developer-tool sites in the public developer bucket.

The top results show the pattern agent builders can actually use: public `llms.txt`, an AI plugin manifest, OpenAPI, a structured API, MCP where real tools exist, an AI-friendly robots policy, and Schema.org all visible from the public web.

That matters for developer tools because agents do not browse product pages the way humans do. They need deterministic docs, machine-readable endpoints, and a way to verify whether a tool is safe to call before wiring it into a workflow.

For developer-tool owners, the practical checklist is:

1. Publish `llms.txt` with docs, API links, pricing/contact boundaries, and tool scope.
2. Keep OpenAPI current for read-only product, docs, status, package, or account endpoints.
3. Add MCP only for real operational tools, not a badge claim.
4. Make `/.well-known/ai-plugin.json` point at the same maintained public API surface.
5. Register monitoring after the score is fixed so deploys do not silently remove the evidence path.

Search the public developer bucket:

`https://nothumansearch.ai/api/v1/top?category=developer&limit=25`

Check a developer-tool site:

`https://nothumansearch.ai/score`

## Owner/Buyer Angle

This is for developer infrastructure, package/API, MCP tooling, agent-framework, docs, and automation-product owners. The sell is not ranking placement. The sell is making the operational surface legible to agents and keeping it from drifting after deploys.

## Publication Guard

Before external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=developer&limit=8`, `/llms.txt`, and `/.well-known/mcp.json`.
- Check the shared social ledger for duplicate fingerprints.
- Verify the active account identity for the selected channel.
- Take a sync-state public-action lock.
- Do not claim private demand, revenue, conversion, paid ranking placement, preferred inclusion, or score-methodology bypass.
