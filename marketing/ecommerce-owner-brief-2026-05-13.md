# NHS Ecommerce Owner Brief

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-13T08:16Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, or QLimit/global-queue write was performed. This artifact is a channel brief for a later gated operator. External use still requires active account verification, duplicate-fingerprint checks, and a sync-state public-action lock.

## Live Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4239`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. The ecommerce bucket has `149` sites with average score `41`.
- `https://nothumansearch.ai/api/v1/top?category=ecommerce&limit=8`: top ecommerce examples show the full spread from 100/100 agent-ready commerce surfaces to 75/100 sites missing either structured API, MCP, robots AI rules, or OpenAPI.
- `https://nothumansearch.ai/llms.txt`: advertises ecommerce as a public category and keeps `other`/`spam` as audit-only buckets.
- `https://nothumansearch.ai/.well-known/mcp.json`: category parameter copy matches the public ecommerce category and excludes audit-only buckets from promoted discovery inventory.

No raw user identifiers, private customer rows, API keys, checkout URLs, or private query logs were written.

## Brief Copy

Subject/heading:

`149 ecommerce sites are already agent-readable. Most still have missing buying signals.`

Short post:

Not Human Search currently tracks 149 ecommerce sites with enough agent-facing structure to appear in the ecommerce bucket.

The top of the category shows what an agent-readable buying surface can look like: `llms.txt`, an AI plugin manifest, OpenAPI, a structured API, MCP, AI-friendly robots policy, and Schema.org all visible from the public web.

The useful gap is lower in the same bucket. Several ecommerce sites expose some machine-readable surfaces but still miss one or more of the signals that help agents inspect, compare, and buy without guessing.

For ecommerce owners, the practical checklist is:

1. Publish `llms.txt` with products, policies, and API links.
2. Expose a small OpenAPI or catalog endpoint for product and checkout metadata.
3. Keep `/.well-known/ai-plugin.json` and Schema.org aligned with the same buying surface.
4. Add MCP only when there is a real tool surface, not just marketing copy.
5. Register monitoring after the score is fixed so regressions do not silently remove the buying path.

Search the public ecommerce bucket:

`https://nothumansearch.ai/api/v1/top?category=ecommerce&limit=25`

Check a store:

`https://nothumansearch.ai/score`

## Owner/Buyer Angle

This is for ecommerce operators and agent-commerce builders. The sell is not ranking placement. The sell is making the public buying surface legible to agents, then monitoring it so deploys do not break the signals.

## Publication Guard

Before external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=ecommerce&limit=8`, `/llms.txt`, and `/.well-known/mcp.json`.
- Check the shared social ledger for duplicate fingerprints.
- Verify the active account identity for the selected channel.
- Take a sync-state public-action lock.
- Do not claim private demand, revenue, conversion, paid ranking placement, or score-methodology bypass.
