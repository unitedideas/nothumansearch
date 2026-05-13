# NHS Healthcare Owner Brief

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-13T20:20Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, or QLimit/global-queue write was performed. This artifact
is a channel brief for a later gated operator. External use still requires active
account verification, duplicate-fingerprint checks, and a sync-state
public-action lock.

No raw user identifiers, private customer rows, API keys, checkout URLs, private
query logs, clinical claims, or payment identifiers were written.

## Live Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4169`, `avg_score=35`,
  `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. The health bucket
  has `56` sites with average score `41`.
- `https://nothumansearch.ai/api/v1/top?category=health&limit=8`: the top health
  example scores 100/100 and exposes all seven public agent-readiness signals.
- `https://nothumansearch.ai/llms.txt`: health is listed as a public category;
  `other` and `spam` are audit-only.
- `https://nothumansearch.ai/.well-known/mcp.json`: health appears in the public
  category parameter copy.

Public top-category examples observed during preparation:

- `zgts.in` - score 100, consumer health commerce surface with all seven signals.
- `opdstar.com` - score 80, clinical documentation workflow with `llms.txt`,
  AI plugin, structured API, MCP, AI-friendly robots, and Schema.org, but missing
  OpenAPI in the current crawl.
- `hipaaagent.ai` - score 80, healthcare compliance product with `llms.txt`,
  AI plugin, structured API, MCP, AI-friendly robots, and Schema.org, but missing
  OpenAPI in the current crawl.
- `monarchinitiative.org` - score 65, biomedical data surface with `llms.txt`,
  OpenAPI, structured API, and Schema.org, but missing AI plugin, AI-friendly
  robots, and MCP in the current crawl.

## Brief Copy

Subject/heading:

`56 healthcare sites are in Not Human Search. The readiness gap is mostly public infrastructure.`

Short post:

Not Human Search currently tracks 56 healthcare sites in the public health bucket.

The useful pattern is not a clinical claim. It is infrastructure. The top health
example exposes the full public agent-readiness surface: `llms.txt`, an AI
plugin manifest, OpenAPI, a structured API, MCP, an AI-friendly robots policy,
and Schema.org.

The next tier is where the owner-side work is visible. Healthcare compliance,
clinical documentation, biomedical data, and health-commerce sites often already
have some machine-readable surface, but one or two public pieces are missing:
OpenAPI, MCP where bounded operations exist, AI-friendly robots, or a maintained
AI plugin manifest.

For healthcare API, FHIR/data, HIPAA compliance, clinical workflow, and
health-commerce owners, the practical checklist is:

1. Publish `llms.txt` with scope, safety boundaries, API links, and support
   contact paths.
2. Keep OpenAPI current for non-clinical operational endpoints such as lookup,
   scheduling, documentation, eligibility, compliance, or data access.
3. Add MCP only for bounded, auditable operations where an agent should act.
4. Keep `/.well-known/ai-plugin.json` pointed at the maintained API surface.
5. Register monitoring after the public evidence path is fixed so deploys do not
   silently remove the signals.

Search the public health bucket:

`https://nothumansearch.ai/api/v1/top?category=health&limit=25`

Check a healthcare site:

`https://nothumansearch.ai/score`

## Owner/Buyer Angle

This is for healthcare API teams, clinical workflow tools, compliance products,
medical data providers, and health-commerce operators. The sell is not ranking
placement, certification, or clinical endorsement. The sell is making the public
operational surface legible to agents and catching drift before customers or
integrators do.

## Publication Guard

Before external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`,
  `/api/v1/top?category=health&limit=8`, `/llms.txt`, and
  `/.well-known/mcp.json`.
- Check the shared social ledger for duplicate fingerprints.
- Verify the active account identity for the selected channel.
- Take a sync-state public-action lock.
- Do not claim private demand, revenue, conversion, clinical endorsement,
  compliance certification, paid ranking placement, preferred inclusion, or
  score-methodology bypass.
