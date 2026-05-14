# NHS Productivity Owner Brief

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-13T22:25Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, or QLimit/global-queue write was performed. This artifact is a channel brief for a later gated operator. External use still requires active account verification, duplicate-fingerprint checks, and a sync-state public-action lock.

## Live Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4169`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. The productivity bucket has `172` sites with average score `38`.
- `https://nothumansearch.ai/api/v1/top?category=productivity&limit=8`: top productivity examples range from 100/100 complete agent-readiness to 70/100 partial surfaces.
- `https://nothumansearch.ai/llms.txt`: advertises productivity as a public category and keeps `other`/`spam` as audit-only buckets.
- `https://nothumansearch.ai/.well-known/mcp.json`: category parameter copy matches the public productivity category and excludes audit-only buckets from promoted discovery inventory.

No raw user identifiers, private customer rows, API keys, checkout URLs, or private query logs were written.

## Brief Copy

Subject/heading:

`172 productivity sites are agent-readable. CRM and document workflows are the near-term owner channel.`

Short post:

Not Human Search currently tracks 172 productivity sites with enough agent-facing structure to appear in the public productivity bucket.

The top of the bucket shows the useful pattern for agent workflows: public `llms.txt`, an AI plugin manifest, OpenAPI, a structured API, MCP where there is a real tool surface, AI-friendly robots policy, and Schema.org all visible from the public web.

This is a practical owner channel because agents already need CRM records, documents, messages, trip plans, and local-presence tools. Several products are close to fully agent-readable but still miss one or more signals that make the surface easier to inspect or call without browser scraping.

For productivity-tool owners, the practical checklist is:

1. Publish `llms.txt` with product scope, docs, support, and API links.
2. Expose OpenAPI for read-only product, account, document, CRM, or workflow endpoints.
3. Keep `/.well-known/ai-plugin.json` aligned with the same public API surface.
4. Add MCP only where agents can call real workflow tools.
5. Register monitoring after the score is fixed so deploys do not silently remove the agent path.

Search the public productivity bucket:

`https://nothumansearch.ai/api/v1/top?category=productivity&limit=25`

Check a productivity site:

`https://nothumansearch.ai/score`

## Owner/Buyer Angle

This is for CRM, document, collaboration, planning, and workflow-tool operators. The sell is not ranking placement. The sell is making useful public product and API surfaces legible to agents, then monitoring them so deploy drift does not break the signals.

## Publication Guard

Before external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=productivity&limit=8`, `/llms.txt`, and `/.well-known/mcp.json`.
- Check the shared social ledger for duplicate fingerprints.
- Verify the active account identity for the selected channel.
- Take a sync-state public-action lock.
- Do not claim private demand, revenue, conversion, paid ranking placement, preferred inclusion, or score-methodology bypass.
