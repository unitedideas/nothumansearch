# Consumer Health Data Source Readiness Brief

Run: 2026-05-15 business-marketer-not-human-search

## Source Signal

- NHS public stats: 4,174 indexed sites, average score 35.
- Public category aggregate: `health` has 57 sites, average score 42.
- MCP analytics, 7 days: 410 `tools/call`, 250 `search_agents`, 50 `get_site_details`, 17 `check_url`, 0 unknown tool names observed in the top tool list.
- MCP top-query themes included consumer health and nutrition-source lookups:
  - `Nutritionix API nutrition database`
  - `Apple Health MCP`

## Public Evidence

Public health top-list checks should use:

- `https://nothumansearch.ai/api/v1/top?category=health&limit=8`
- `https://nothumansearch.ai/api/v1/categories`
- `https://nothumansearch.ai/llms.txt`
- `https://nothumansearch.ai/.well-known/mcp.json`

Current top health examples include a mix of 100-score, 80-score, and lower-score public profiles. The category is useful as a source-readiness angle, but the list is mixed enough that public copy should avoid broad claims about medical coverage, clinical quality, regulatory compliance, or live data freshness.

## Draft Angle

Agents are already trying to discover health and nutrition data sources through NHS-style MCP workflows. The owner-side gap is not whether a health site has a marketing page; it is whether an agent can find a stable API contract, schema, pricing or access metadata, and a monitorable public readiness score.

Use this for a gated owner/channel packet aimed at consumer health API providers, nutrition databases, quantified-self tools, and health-data developer audiences.

## Guardrails

- Do not claim clinical endorsement, medical accuracy, privacy compliance, HIPAA readiness, live data freshness, or comprehensive health coverage.
- Do not expose raw MCP request metadata, user identifiers, emails, payment ids, or private monitor rows.
- Do not imply private demand, revenue, completed payments, paid ranking placement, preferred inclusion, or score-methodology bypass.
- Before external use, refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=health&limit=8`, `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, and `/api/v1/admin/mcp?days=7`.

## Candidate Next Step

Prepare a gated channel packet or owner-conversion test for consumer health data providers. Keep the action behind channel identity verification, duplicate checks, and a sync-state public-action lock.
