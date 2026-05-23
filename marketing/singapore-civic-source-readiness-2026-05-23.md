# Singapore Civic Source-Readiness Segment

Created: 2026-05-23
Agent: `business-marketer-not-human-search`

## Evidence Checked

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4173`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: `news=12`, `avg_score=50`; `data=399`, `avg_score=32`; `other=779`, `avg_score=27`.
- Aggregate MCP analytics for the last 7 days: `tools/list=170386`, `initialize=27915`, `tools/call=242`, with visible query themes including Singapore news, Singapore housing, HDB BTO launches, and local publisher/source discovery. No raw user identifiers were written.
- Aggregate traffic for the last 168 hours still shows owner-conversion surfaces receiving material traffic: `/score=72`, `/top=97`, `/newest=64`, `/api/v1/search=194`, `/api/v1/submit=143`, `/api/v1/catalog=316`.
- Live discovery surfaces checked: `/score`, `/monitor`, `/report`, `/newest`, `/top`, `/api/v1`, `/api/v1/catalog`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/llms.txt`, `/openapi.yaml`, `/feed.xml`, `/mcp-servers`, `/openapi-apis`, and `/llms-txt-sites` returned 200.
- `/.well-known/agent-card.json` returned 404, so A2A/Agent Card claims remain blocked.
- Public profile checks: `channelnewsasia.com` and `straitstimes.com` returned NHS profile pages with `5/100`; `hdb.gov.sg` returned a public profile with `0/100`; `mothership.sg` and `data.gov.sg` did not return public NHS profile pages during this run.
- `/fix/{host}` checks for `data.gov.sg`, `hdb.gov.sg`, and `channelnewsasia.com` returned 404 in this run, so this segment should route owners to `/score` first rather than direct paid remediation.

## Segment

Singapore publishers, public agencies, housing-information surfaces, civic-data portals, and local-information sites that expect agents to cite, monitor, or route users to changing public facts.

This is narrower than the existing local-news/housing brief. The reusable angle is not "NHS answers Singapore housing questions." It is that source owners with volatile public information need machine-readable contracts so agents can identify the right source, understand update boundaries, and monitor whether readiness regresses.

## Draft Angle

`Singapore civic sources need machine-readable boundaries`

Agents are using NHS-style source discovery for Singapore news and housing questions. The product-safe message is that publishers and civic-data owners should expose source contracts: `llms.txt`, structured APIs or feeds, OpenAPI where applicable, explicit robots policy, Schema.org metadata, and free monitoring so readiness does not silently disappear after site changes.

Owner-side checklist:

1. Publish `llms.txt` with source scope, update cadence, and non-coverage boundaries.
2. Expose a structured API or stable feed for high-change datasets.
3. Publish OpenAPI when the data surface is meant for agent or developer access.
4. Make AI crawler policy explicit in `robots.txt`.
5. Add Schema.org for articles, datasets, public-service organization identity, and update metadata.
6. Register free score monitoring after the score improves.

## Public Examples

These are public NHS profile checks only, not customers, endorsements, paid leads, or private demand.

| Domain | Public Check | Route |
|---|---:|---|
| `channelnewsasia.com` | `5/100` | Score first; do not direct-link paid remediation from this packet. |
| `straitstimes.com` | `5/100` | Score first; use missing-surface checklist language. |
| `hdb.gov.sg` | `0/100` | Score first; frame as civic-source readiness, not housing advice. |
| `mothership.sg` | No public profile in this run | Use only after a fresh score/profile check. |
| `data.gov.sg` | No public profile in this run | Use only after a fresh score/profile check. |

## Channel Use

Prepare exactly one gated owner-channel touch, post, or product-handoff test for Singapore civic-data, local-news, housing-information, publisher, or public-service source owners.

Before external use, refresh:

- `/api/v1/stats`
- `/api/v1/categories`
- `/api/v1/top?category=news&limit=12`
- `/api/v1/top?category=data&limit=12`
- `/score`
- `/monitor`
- `/report`
- Representative `/site/{host}` pages
- Representative `/fix/{host}` routes
- `/mcp`
- `/.well-known/mcp.json`
- `/.well-known/agent.json`
- `/.well-known/agent-card.json`
- `/.well-known/commerce.json`
- `/.well-known/ai-plugin.json`
- `/api/v1`
- `/api/v1/catalog`
- `/api/v1/quote`
- `/api/v1/checkout`
- `/llms.txt`
- `/openapi.yaml`
- `/feed.xml`
- Aggregate `/api/v1/admin/mcp?days=7`
- Aggregate `/api/v1/admin/traffic?hours=168`

## Claim Boundaries

Do not claim factual freshness, editorial accuracy, housing-market accuracy, BTO launch accuracy, public-service endorsement, civic-data completeness, legal/compliance correctness, publisher endorsement, private demand, paid leads, monitor registrations, badge-install consent, completed payments, revenue, crawler compliance, SEO lift, uptime proof, A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/MPP support for NHS, paid ranking placement, preferred inclusion, or score-methodology bypass.
