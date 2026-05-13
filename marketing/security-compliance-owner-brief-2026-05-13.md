# NHS Security and Compliance Owner Brief

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-13T14:08Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, or QLimit/global-queue write was performed. This artifact is a channel brief for a later gated operator. External use still requires active account verification, duplicate-fingerprint checks, and a sync-state public-action lock.

## Live Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4219`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. The security bucket has `116` sites with average score `38`.
- `https://nothumansearch.ai/api/v1/top?category=security&limit=8`: top security/compliance examples range from 100/100 complete agent-readiness to 70/100 partial surfaces. The partial examples usually miss MCP, AI plugin, OpenAPI, structured API, or Schema.org rather than basic crawlability.
- `https://nothumansearch.ai/llms.txt`: advertises security as a public category and keeps `other`/`spam` as audit-only buckets.
- `https://nothumansearch.ai/.well-known/mcp.json`: category parameter copy matches the public security category and excludes audit-only buckets from promoted discovery inventory.

No raw user identifiers, private customer rows, API keys, checkout URLs, or private query logs were written.

## Brief Copy

Subject/heading:

`116 security and compliance sites are agent-readable. The gap is auditability, not crawlability.`

Short post:

Not Human Search currently tracks 116 security and compliance sites with enough agent-facing structure to appear in the public security bucket.

The top of the bucket shows what works for security buyers and agent builders: public `llms.txt`, an AI plugin manifest, OpenAPI, a structured API, MCP where a real tool surface exists, AI-friendly robots policy, and Schema.org all visible from the public web.

The useful gap is below the top few results. Several security/compliance sites already expose part of the machine-readable surface, but still miss one or more signals that make the product inspectable by autonomous agents and procurement workflows.

For security and compliance owners, the practical checklist is:

1. Publish `llms.txt` with scope, docs, security contact, and API links.
2. Expose OpenAPI for read-only product, status, policy, or evidence endpoints.
3. Keep `/.well-known/ai-plugin.json` aligned with the same public API surface.
4. Add MCP only for real operational tools, not marketing claims.
5. Register monitoring after the score is fixed so deploys do not silently remove the evidence path.

Search the public security bucket:

`https://nothumansearch.ai/api/v1/top?category=security&limit=25`

Check a security or compliance site:

`https://nothumansearch.ai/score`

## Owner/Buyer Angle

This is for security tooling, compliance automation, trust-center, and AI-governance operators. The sell is not ranking placement. The sell is making public trust and compliance surfaces legible to agents, then monitoring them so deployment drift does not remove the evidence.

## Publication Guard

Before external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=security&limit=8`, `/llms.txt`, and `/.well-known/mcp.json`.
- Check the shared social ledger for duplicate fingerprints.
- Verify the active account identity for the selected channel.
- Take a sync-state public-action lock.
- Do not claim private demand, revenue, conversion, compliance certification, paid ranking placement, or score-methodology bypass.
