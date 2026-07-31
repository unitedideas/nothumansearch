# MCP stats-to-category tool-chaining test

Date: 2026-07-31
Automation: `business-marketer-not-human-search`
Status: private product handoff only; no outreach, post, account action, browser action, deploy, crawl, checkout, or global-queue write performed

## Segment

Turn the informational `get_stats` and `list_categories` MCP tools into a
machine-actionable discovery chain. These tools currently return useful
aggregate data but no explicit next tool, so an agent must infer how to move
from the index overview to a category-specific result set.

This is a product-handoff conversion test. It does not authorize public copy or
owner outreach.

## Fresh evidence

- Public stats report 4,351 indexed sites, average score 38, and `developer` as
  the top category. The public categories response contains 14 categories.
- Seven-day aggregate MCP analytics recorded 49,164 `tools/list`, 14,373
  `initialize`, and 245 `tools/call` requests.
- `get_stats=49` and `list_categories=22`, or 71 of 245 tool calls, reached
  informational overview tools. Treat this as product-flow evidence, not
  customer demand or owner intent.
- Live JSON-RPC `tools/list` returned the expected 11 tools.
- A bounded live `get_stats` call returned text plus
  `structuredContent` fields for total sites, average score, MCP sites,
  perfect-score sites, weekly additions, and top category. It contained no
  `next_tool`, `suggested_tools`, `list_categories`, `get_top_sites`, or
  `search_agents` hint.
- A bounded live `list_categories` call returned text plus
  `structuredContent.categories`. It contained no `next_tool`,
  `suggested_tools`, `get_top_sites`, or `search_agents` hint.
- The previous result-list handoff already covers report, monitor, and
  score-band-aware remediation once a domain has been selected. This test stops
  before that stage and does not duplicate the `get_top_sites` or
  `recent_additions` result contract.
- The monitor aggregate contains five active and three quarantined
  registrations. The redacted score-fix aggregate contains ten pending
  real-candidate rows and no real-candidate lead or paid rows. Do not treat
  either aggregate as proof that overview-tool callers are owners or buyers.
- `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/llms.txt`,
  `/openapi.yaml`, `/api/v1`, `/api/v1/catalog`, `/score`, `/monitor`,
  `/report`, `/top`, and `/newest` returned HTTP 200.
  `/.well-known/agent-card.json` returned 404, so A2A claims remain blocked.

No raw MCP queries, user-agent strings, emails, private monitor rows, private
score-fix rows, checkout URLs, payment identifiers, API keys, or customer
identifiers were written to this artifact.

## Conversion hypothesis

An agent that asks for aggregate stats should receive a small, stable set of
machine-readable next-tool suggestions:

1. `list_categories` to inspect the available taxonomy.
2. `get_top_sites` to retrieve ranked results, optionally using the current top
   category.
3. `search_agents` when the caller already has a specific capability need.

An agent that lists categories should receive invocation-ready examples for
`get_top_sites(category=...)` and `search_agents`, without embedding volatile
site counts, plan copy, or paid remediation. Report, monitor, and score-fix
routing belongs only after a domain result exists.

## Acceptance test

1. `get_stats` keeps every existing text and structured field while adding a
   backwards-compatible `suggested_tools` array for `list_categories`,
   `get_top_sites`, and `search_agents`.
2. `list_categories` keeps every existing category field while adding
   invocation-ready `suggested_tools` entries for category-filtered
   `get_top_sites` and capability-oriented `search_agents`.
3. Suggestions use the canonical names and input shapes from the same live
   `tools/list` response; tests fail if the tool schema drifts.
4. No overview response includes a paid CTA, owner claim, monitor-registration
   claim, or score-fix URL before a domain is selected.
5. MCP unit tests cover text and `structuredContent` compatibility for both
   tools.
6. After a later product-worker deploy, live `tools/list`, `get_stats`,
   `list_categories`, one suggested `get_top_sites` call, and one suggested
   `search_agents` call pass end to end.
7. Analytics distinguish overview delivery, suggested-tool selection, result
   delivery, report visits, monitor registrations, and paid conversions.
   Earlier stages may not be described as demand, customers, consent, or
   revenue.

## Duplicate and claim boundary

The portfolio social ledger, NHS distribution history, marketer inbox, and
existing MCP conversion artifacts had no exact stats-to-category-to-result
contract. Existing search-result and top/recent work begins after result
selection; this test covers only the preceding tool-chaining gap.

Do not claim MCP traffic proves customers, owners, endorsements, private demand,
monitor consent, completed payments, revenue, data quality, freshness, uptime,
security certification, paid ranking, preferred inclusion, A2A support while
the Agent Card route is 404, x402/ACP/SPT/MPP support for NHS, or a
score-methodology bypass.
