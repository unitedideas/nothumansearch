# Robots AI Policy Owner Conversion Scout

Date: 2026-05-19
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, checkout completion, or QLimit/global-queue write was
performed. This is a sanitized scout artifact for a later product or channel
operator.

No raw users, emails, API keys, checkout URLs, payment identifiers, private
monitor rows, private score-fix rows, private query logs, buyer data, or raw
customer data are included here.

## Fresh Evidence

Public surface checks:

- `/api/v1/stats`: `total_sites=4175`, `avg_score=35`,
  `top_category=developer`.
- `/api/v1/categories`: developer 1,231; ai-tools 904; other 768; data 400;
  finance 196; productivity 173; ecommerce 148; communication 120; security
  115; health 59; jobs 27; education 21; news 12; spam 1.
- `/robots.txt`: HTTP 200 and explicitly allows major AI/user-agent crawlers,
  with sitemap, `llms.txt`, OpenAPI, plugin, MCP, registry proof, and security
  contact pointers.
- `/.well-known/ai-plugin.json`: HTTP 200 and points to the maintained OpenAPI
  and MCP paths.
- `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/api/v1`, `/api/v1/catalog`, `/openapi.yaml`,
  `/score`, `/monitor`, and `/report`: HTTP 200.
- `/.well-known/agent-card.json`: HTTP 404, so strict A2A/Agent Card claims
  remain gated.

Aggregate admin evidence, sanitized:

- Last 168 hours traffic included `/robots.txt=308`, behind the homepage,
  badge/profile loops, commerce manifest, plugin manifest, `llms.txt`,
  OpenAPI, catalog, quote, and checkout routes, and ahead of `/score`,
  `/api/v1/check`, `/guide`, `/newest`, and `/digest`.
- Last 7 days MCP query themes still include model/API pricing, local news and
  housing, agent skills, RAG/document indexing, hardware/IoT, scanner/electronics
  retail, collectibles price data, and agent labor/work searches. Treat these
  as source-readiness themes only, not as private demand or factual-coverage
  proof.

Public top-list examples for later owner routing:

| Category | Example | Score | Use Boundary |
|---|---:|---:|---|
| communication | `resend.com` | 75 | `/score` first; robots policy and plugin/schema gaps can be concrete remediation topics. |
| communication | `slack.com` | 60 | `/score` first; avoid workflow, deliverability, privacy, or integration-quality claims. |
| developer | `xquik.com` | 100 | High-score third-party profile; route to monitor/report/badge proof, not remediation. |
| developer | `agentprobe.fly.dev` | 100 | Foundry-owned dogfood; label before any public use. |
| data | `api.theartofservice.com` | 90 | High-score API owner; route to monitor/report proof before remediation. |
| security | `rnwy.com` | 80 | `/score` first; avoid security certification or uptime claims. |

## Read

`/robots.txt` is a material owner-conversion surface, not just crawler plumbing.
It now sits above `/score`, `/api/v1/check`, `/guide`, and `/newest` in the
current 168-hour aggregate. The useful owner-channel angle is that AI crawler
policy should be tied to the rest of the machine-readable packet: `llms.txt`,
OpenAPI, plugin manifest, MCP, sitemap, security contact, and monitorable
change detection.

This should not become broad "allow every AI crawler" advice. The safer framing
is:

1. Make the policy explicit so agents and crawlers do not infer access from
   brittle page scraping.
2. Point the policy at maintained machine-readable surfaces.
3. Monitor the public packet for drift after launch.
4. Route high-score owners to free monitor/report/badge proof and partial-score
   owners to `/score` before score-fix remediation.

## Candidate Copy Boundary

Usable phrasing:

> Robots policy is part of agent readiness now. If crawlers can see the policy
> but not the API contract, `llms.txt`, plugin manifest, or monitorable MCP
> surface, agents still have to guess.

Avoid:

- Claims that listed domains are customers, endorsements, paid leads, private
  demand, completed payments, revenue, badge-install consent, or monitor
  registrations.
- Claims of A2A support while `/.well-known/agent-card.json` returns 404.
- Claims about crawler compliance, legal permission, privacy compliance,
  security certification, SEO lift, traffic lift, ranking placement, preferred
  inclusion, x402/ACP support, or score-methodology bypass.

## Execution Gate

Before implementation or external use:

1. Refresh `/api/v1/stats`, `/api/v1/categories`, `/robots.txt`, `/llms.txt`,
   `/.well-known/ai-plugin.json`, `/.well-known/mcp.json`,
   `/.well-known/agent.json`, `/.well-known/agent-card.json`,
   `/.well-known/commerce.json`, `/api/v1/catalog`, `/api/v1/quote`,
   `/api/v1/checkout`, `/api/v1`, `/openapi.yaml`, `/score`, `/monitor`,
   `/report`, representative high-score and partial-score `/site/{host}` pages,
   high-score and partial-score `/fix/{host}` routes, aggregate
   `/api/v1/admin/traffic?hours=168`, and aggregate
   `/api/v1/admin/mcp?days=7`.
2. Verify active Foundry/Owl-owned account identity before public use.
3. Check `marketing/social-post-ledger.json` if present,
   `outreach/distribution_log.csv`, and sync-state public-action locks.
4. Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.
