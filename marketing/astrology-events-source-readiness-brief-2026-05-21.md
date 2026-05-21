# Astrology, Events, and Lifestyle Source-Readiness Brief

Created: 2026-05-21
Source agent: `business-marketer-not-human-search`

## Segment

Recent MCP query themes include astrology, moon phase, planetary transits, local event/travel pages, gift recommendation, and affiliate/referral API lookups. This is a narrow lifestyle-source segment, not a broad market claim.

## Current Evidence

- Live stats: `total_sites=4171`, `avg_score=35`, `top_category=developer`.
- Aggregate MCP 7d: `tools/list=166321`, `tools/call=259`, `search_agents=144`, `get_site_details=35`, `check_url=18`.
- Aggregate traffic 168h: `/=3412`, `/.well-known/commerce.json=1522`, `/.well-known/ai-plugin.json=700`, `/llms.txt=454`, `/openapi.yaml=424`, `/api/v1/catalog=328`, `/api/v1/search=174`, `/api/v1/submit=152`, `/score=79`.
- Current public discovery surfaces are live: `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, `/api/v1`, `/llms.txt`, `/openapi.yaml`, `/score`, `/monitor`, and `/report`.
- `/.well-known/agent-card.json` is still `404`, so A2A/Agent Card claims stay blocked.

## Public Examples

These are public readiness examples only. They are not customers, endorsements, leads, or demand proof.

- `astranl.com`: public profile score `100/100`; `/fix/astranl.com` routes as already meeting the target.
- `fullmoonparty-thailand.com`: public profile score `50/100`; useful as a local event/travel readiness example.
- `surprise-buddy.com`: public profile score `65/100`; useful as a gift recommendation / affiliate-adjacent readiness example.
- `twitterapi.io`: public profile score `65/100`; useful as an API/source-data boundary example.

## Gated Channel Angle

Lifestyle and event sites increasingly get queried by agents as sources, not destinations. The owner-facing point is concrete:

1. Publish machine-readable source contracts: `llms.txt`, OpenAPI, structured API/catalog metadata, robots policy, and stable profile pages.
2. Use free monitor registration to catch drift in those files.
3. Route high-score owners to monitor/report/badge proof.
4. Route partial-score owners through `/score` before any remediation offer.

## Boundaries

Do not claim astrology accuracy, event-date accuracy, travel fulfillment, gift recommendation quality, affiliate revenue, inventory accuracy, private demand, paid leads, completed payments, endorsement, SEO lift, paid placement, preferred inclusion, A2A support, or score-methodology bypass.

Before external use, refresh `/api/v1/stats`, `/api/v1/categories`, representative `/site/{host}` pages, high-score and partial-score `/fix/{host}` routes, `/score`, `/monitor`, `/report`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/api/v1`, `/llms.txt`, `/openapi.yaml`, aggregate `/api/v1/admin/mcp?days=7`, and aggregate `/api/v1/admin/traffic?hours=168`.
