# Badge/Profile Monitor Conversion Refresh

Run: 2026-05-18T23:45Z
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, checkout completion, or QLimit/global-queue write was
performed. This is a sanitized scout artifact for a later gated operator.

No raw users, emails, API keys, checkout URLs, payment identifiers, private
monitor rows, private score-fix rows, private query logs, buyer data, or raw
customer data are included here.

## Fresh Evidence

Public surfaces checked:

- `/api/v1/stats`: `total_sites=4174`, `avg_score=35`,
  `top_category=developer`.
- `/api/v1/categories`: developer 1,237; ai-tools 900; other 765; data 399;
  finance 199; productivity 173; ecommerce 149; communication 119; security
  115; health 57; jobs 27; education 21; news 12.
- `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/api/v1/catalog`, `/api/v1`, `/score`,
  `/monitor`, and `/openapi.yaml`: HTTP 200.
- `/.well-known/agent-card.json`: HTTP 404, so A2A/Agent Card directory claims
  remain gated.
- `/api/v1/top?category=finance&limit=10`,
  `/api/v1/top?category=data&limit=10`,
  `/api/v1/top?category=ecommerce&limit=10`, and
  `/api/v1/top?category=developer&limit=10`: HTTP 200 with `results` arrays.

Aggregate admin evidence, sanitized:

- Last 7 days MCP analytics: `tools/list=151491`, `initialize=19746`,
  `tools/call=296`.
- Top called tools: `search_agents=189`, `get_site_details=38`,
  `check_url=17`, `get_stats=13`, `verify_mcp=10`, `recent_additions=8`,
  `get_top_sites=7`, `find_mcp_servers=7`, `submit_site=4`,
  `list_categories=3`.
- Query themes again included Hermes/agent skills, memory MCP servers,
  local-news/housing, model/API options, document indexing/RAG, document
  scanner hardware, autonomous-agent source-code loops, and embedded hardware.
- Last 168 hours traffic: `/badge/xquik.com.svg=1681`,
  `/.well-known/commerce.json=1661`, `/.well-known/ai-plugin.json=788`,
  `/llms.txt=508`, `/site/xquik.com=481`, `/openapi.yaml=465`,
  `/api/v1/catalog=372`, `/api/v1/quote=344`, `/api/v1/checkout=344`,
  `/badge/aidevboard.com.svg=318`.
- Google referrer aggregate: `google.com=542`.
- Payment aggregate returned no completed payment signal in this scout output.

Monitor worker proof:

- Latest scheduled run completed on 2026-05-18 07:30 PT.
- It checked one due active monitor and held score 100 -> 100.
- Prior quarantine still matters for broad monitor-growth copy; use the free
  monitor offer as score-preservation proof, not as a growth metric.

Score-fix routing proof:

- `/fix/xquik.com`: HTTP 200 high-score handoff.
- `/fix/aidevboard.com`: HTTP 200 high-score handoff.
- `/fix/amalgix.io`: HTTP 200 high-score handoff.
- `/fix/crabbitmq.com`: HTTP 200 remediation intake.

## Why This Segment Matters

The 2026-05-14 badge/profile scout showed `/badge/xquik.com.svg=843` and
`/site/xquik.com=159` over 336 hours. The current 168-hour aggregate shows
`/badge/xquik.com.svg=1681` and `/site/xquik.com=481`. The badge/profile loop
is still the clearest owner-conversion surface because it creates repeat visits
to public proof pages without outreach.

The score-fix high-score gate is now safe enough to use in this flow: high-score
domains route to monitor/report proof, while a partial-score example still sees
paid remediation. That makes the next test cleaner than the older generic badge
CTA handoff.

## Proposed Gated Test

Create one owner-channel or product-handoff test around badge/profile visitors:

1. High-score profile route:
   - Primary CTA: free monitor registration for regression alerts.
   - Secondary CTA: copy badge/report proof.
   - Example checks: `xquik.com`, `aidevboard.com`, `amalgix.io`.

2. Partial-score profile route:
   - Primary CTA: run `/score` and inspect missing public signals.
   - Secondary CTA: score-fix remediation only when the public gaps are
     concrete.
   - Example check: `crabbitmq.com`.

3. Agent-commerce route:
   - Keep catalog/quote/checkout traffic on API-plan and score-fix buyer
     surfaces.
   - Do not claim payment conversion until a completed-payment ledger proves it.

## Acceptance Guard

Before external use or implementation:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/score`, `/monitor`,
  representative `/site/{host}` pages, high-score and partial-score
  `/fix/{host}` routes, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/agent-card.json`, `/.well-known/commerce.json`,
  `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`,
  `/openapi.yaml`, aggregate `/api/v1/admin/mcp?days=7`, and aggregate
  `/api/v1/admin/traffic?hours=168`.
- Verify active Foundry/Owl-owned account identity before any public post or
  directory action.
- Check `marketing/social-post-ledger.json` if present,
  `outreach/distribution_log.csv`, and sync-state public-action locks.
- Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.
- Do not imply the listed domains are customers, endorsements, paid leads,
  private demand, completed payments, revenue, badge-install consent, monitor
  registration proof, security/compliance certification, A2A support, x402/ACP
  support, paid ranking placement, preferred inclusion, or score-methodology
  bypass.
