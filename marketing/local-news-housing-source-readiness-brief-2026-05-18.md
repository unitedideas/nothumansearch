# Local News and Housing Source-Readiness Brief

Created: 2026-05-18
Agent: `business-marketer-not-human-search`

## Evidence Checked

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4174`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: `news=12`, `avg_score=50`; `data=399`, `avg_score=32`; `other=765`, `avg_score=27`.
- `https://nothumansearch.ai/api/v1/top?category=news&limit=12`: public news examples score from 70 down to 20.
- Aggregate MCP analytics for the last 7 days: `tools/list=145761`, `tools/call=323`, with query themes including local Singapore news and housing, document/RAG source lookup, hardware retail lookup, and current-events/source discovery. No raw user identifiers were written.
- `/.well-known/agent-card.json` still returns 404, so A2A/Agent Card claims remain blocked.
- Score-fix route check: high-score `nothumansearch.ai` routes to monitor/report instead of paid remediation; sub-95 public news examples such as `informedclearly.com`, `biztoc.com`, and `groundhog-day.com` still expose the paid remediation path.

## Segment

Local news, civic information, housing, property, and public-data publishers that expect agents to find and verify changing information.

This is narrower than the earlier generic news brief. The live MCP query themes show agents asking for local/contextual source discovery, not only broad technology or business news. The owner-side gap is machine-readable source readiness: agents need stable source metadata, APIs, feeds, OpenAPI specs, `llms.txt`, robots policy, and monitorable score regression, especially when the underlying facts are volatile.

## Draft Angle

`Local news and housing sources need agent-readable contracts`

Agents are already using Not Human Search for source discovery around local news and housing-style queries. NHS should not claim to answer current facts or certify editorial accuracy. The useful message is simpler: if a publisher, civic-data site, housing portal, or local-information provider wants agents to use it directly, the site needs a probeable public contract rather than a browser-only page.

Useful owner-side checklist:

1. Publish `llms.txt` with scope, update cadence, and source boundaries.
2. Expose a structured API or feed for high-change datasets.
3. Publish OpenAPI if the API is meant for agents or developers.
4. Make robots policy explicit for major AI crawlers.
5. Add Schema.org for articles, datasets, organization identity, and update metadata.
6. Register free score monitoring so deploys do not silently remove agent-readable surfaces.

## Public Examples

These are public NHS index examples only, not customers or endorsements.

| Domain | Score | Notes |
|---|---:|---|
| `informedclearly.com` | 70 | Public news/category example that remains below the score-fix target. |
| `hallucinationherald.com` | 65 | AI/news-style publisher with partial machine-readable readiness. |
| `biztoc.com` | 65 | Business/news aggregation example with room for stronger agent-facing metadata. |
| `zadar.tv` | 55 | Regional media example; useful for the local-publisher angle. |
| `aibtc.news` | 50 | Niche news example with partial readiness. |
| `groundhog-day.com` | 20 | Low-score news example where remediation should start with `/score`. |

## Channel Use

Prepare one gated owner-channel test for publisher, civic-data, housing, property-data, newsletter, or local-information audiences.

Before external use, refresh:

- `/api/v1/stats`
- `/api/v1/categories`
- `/api/v1/top?category=news&limit=12`
- `/api/v1/top?category=data&limit=10`
- Representative `/site/{host}` pages
- `/score`
- `/monitor`
- `/.well-known/mcp.json`
- `/.well-known/agent.json`
- `/.well-known/agent-card.json`
- `/.well-known/commerce.json`
- `/api/v1/catalog`
- `/llms.txt`
- `/openapi.yaml`
- Aggregate `/api/v1/admin/mcp?days=7`

## Claim Boundaries

Do not claim factual freshness, editorial coverage, housing-market accuracy, property availability, legal/compliance correctness, civic-data completeness, publisher endorsement, private demand, paid leads, completed payments, revenue, paid ranking placement, preferred inclusion, A2A support, or score-methodology bypass.
