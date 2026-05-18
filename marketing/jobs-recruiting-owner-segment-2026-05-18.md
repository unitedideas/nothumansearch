# Jobs and Recruiting Owner Segment

Date: 2026-05-18
Automation: `business-marketer-not-human-search`

## Evidence Snapshot

- Public stats: `total_sites=4173`, `avg_score=35`, `top_category=developer`.
- Public category count: `jobs=27`, `avg_score=41`.
- Public jobs top list source: `https://nothumansearch.ai/api/v1/top?category=jobs&limit=10`.
- MCP aggregate, 7 days: `tools/list=142584`, `tools/call=321`, `search_agents=207`, `get_site_details=42`, `check_url=17`, `recent_additions=11`, `verify_mcp=10`, `submit_site=4`.
- Traffic aggregate, 168 hours: `/.well-known/commerce.json=1604`, `/api/v1/catalog=359`, `/api/v1/quote=333`, `/api/v1/checkout=333`, `/top=120`, `/newest=90`, `/score=70`, `/.well-known/mcp.json=85`.
- Compatibility gap still present: `/.well-known/agent-card.json` returns `404`; do not claim A2A Agent Card support.

## Segment

Recruiting, job-board, talent-intelligence, and hiring-platform owners whose public sites already expose some agent-readable signals but are incomplete for agents that need stable job/search/catalog access.

This is a narrow owner segment, not broad market proof. The public jobs category has 27 sites, and the top result is Foundry-owned dogfood (`aidevboard.com`, score 100). Treat ADB as the internal reference implementation, not as third-party validation.

## Public Examples

| Domain | Score | Current Signals | Owner Route |
|---|---:|---|---|
| `aidevboard.com` | 100 | llms.txt, ai-plugin, OpenAPI, robots AI, structured API, MCP, schema | Label as Foundry-owned dogfood only. |
| `jseek.co` | 75 | llms.txt, ai-plugin, OpenAPI, robots AI, schema | High-score owner: free monitor/report/badge proof; structured API and MCP are visible next-step gaps. |
| `himalayas.app` | 65 | llms.txt, ai-plugin, structured API, schema | Mid-score owner: route to `/score`, then monitor; remediation angle is OpenAPI/robots/MCP completeness. |
| `ctojobshq.com` | 50 | llms.txt, robots AI, structured API, schema | Partial-score owner: route to `/score` before score-fix; ai-plugin/OpenAPI/MCP are visible gaps. |
| `reed.co.uk` | 50 | llms.txt, robots AI, structured API, schema | Partial-score owner: route to `/score`; keep claims limited to public readiness signals. |
| `smartrecruiters.com` | 45 | llms.txt, structured API, schema | API-heavy owner: readiness angle is machine-readable plan/auth/API docs, not ranking or placement. |

## Channel Brief

Recruiting platforms increasingly get queried by agents that need direct job discovery, company watchlists, or hiring-market data. The public jobs slice in NHS is small enough to be concrete: 27 indexed jobs/recruiting sites, average readiness score 41, with most top examples publishing llms.txt or a structured API but stopping short of full OpenAPI/MCP coverage.

Good public framing:

> Recruiting platforms are halfway to agent-readable job discovery. In the current Not Human Search jobs slice, the stronger examples already publish llms.txt or structured APIs, but most still miss one or more of OpenAPI, MCP, or complete machine-readable handoffs. The useful owner path is: run `/score`, register a free monitor, then close the missing public signals.

Do not use:

- Do not imply any listed domain is a customer, endorsement, paid lead, private demand, or proof of market share.
- Do not claim job freshness, hiring volume, salary accuracy, apply-flow reliability, legal compliance, candidate quality, private customer demand, completed payments, revenue, paid ranking placement, preferred inclusion, A2A support, or score-methodology bypass.
- Do not market ADB as third-party validation; label it as Foundry-owned dogfood if mentioned.

## Gated Follow-Up

Prepare one owner-channel touch or post for job-board and recruiting-platform owners after refreshing:

- `/api/v1/stats`
- `/api/v1/categories`
- `/api/v1/top?category=jobs&limit=10`
- representative `/site/{host}` profiles
- `/score`
- `/monitor`
- `/.well-known/mcp.json`
- `/.well-known/agent.json`
- `/.well-known/agent-card.json`
- `/.well-known/commerce.json`
- `/api/v1/catalog`
- `/api/v1/quote`
- `/api/v1/checkout`
- `/llms.txt`
- `/openapi.yaml`
- aggregate `/api/v1/admin/mcp?days=7`

Before any external use, verify active Foundry/Owl-owned channel identity, check duplicate fingerprints and public-action locks, and avoid `modelcontextprotocol/*` plus `punkpeye/*` surfaces from `unitedideas`.
