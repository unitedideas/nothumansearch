# Fly App Host Canonical Origin Conversion Scout

Run: 2026-05-20T04:08Z `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, checkout completion, or QLimit/global-queue write was
performed. This is a sanitized scout artifact for a later product or channel
operator.

No raw users, emails, API keys, checkout URLs, payment identifiers, private
monitor rows, private score-fix rows, private query logs, buyer data, or raw
customer data are included here.

## Fresh Evidence

- Public stats: `total_sites=4175`, `avg_score=35`, `top_category=developer`.
- Public categories: developer 1,231; ai-tools 904; other 768; data 400;
  finance 196; productivity 173; ecommerce 148; communication 120; security
  115; health 59; jobs 27; education 21; news 12; spam 1.
- `/score`, `/monitor`, `/api/v1`, `/.well-known/mcp.json`,
  `/.well-known/agent.json`, `/.well-known/commerce.json`, `/llms.txt`,
  `/openapi.yaml`, `/api/v1/catalog`, `/newest`, and `/top`: HTTP 200.
- `/.well-known/agent-card.json`: HTTP 404, so strict A2A/Agent Card claims
  remain gated.
- `/api/v1/checkout`: HTTP 400 without a POST body, the expected no-input
  response for this smoke.
- `https://nothumansearch.ai`, `https://nothumansearch.com`, and
  `http://nothumansearch.com` all end at `https://nothumansearch.ai/`.
- `https://nothumansearch.fly.dev/` returns HTTP 200 and remains a reachable
  alternate origin.
- High-score `/fix/xquik.com` and `/fix/nothumansearch.ai` show the
  already-meets-target branch. Lower-score `/fix/cohere.com` shows score-fix
  remediation intake.

Aggregate admin evidence, sanitized:

- Last 168 hours traffic included `http://nothumansearch.fly.dev` as a top
  referrer with 60 visits.
- Last 168 hours route traffic still concentrates on machine-readable and
  owner-readiness surfaces: `/badge/xquik.com.svg=1929`,
  `/.well-known/commerce.json=1683`, `/.well-known/ai-plugin.json=764`,
  `/site/xquik.com=606`, `/llms.txt=493`, `/openapi.yaml=454`,
  `/api/v1/catalog=361`, `/api/v1/quote=331`, `/api/v1/checkout=331`,
  `/robots.txt=311`, `/api/v1/submit=147`, `/top=105`, `/api/v1=92`,
  `/.well-known/mcp.json=92`, `/guide=90`, `/newest=79`, `/score=78`, and
  `/api/v1/check=60`.
- Last 7 days MCP query themes still include model/API pricing, Singapore news
  and housing, OpenRouter/free-model API searches, agent skills, RAG/document
  indexing, scanners/electronics, embedded hardware/IoT, collectibles price
  data, nutrition API, Home Assistant, and agent labor/work searches. Treat
  these as source-readiness themes only, not private demand or coverage proof.

## Read

The Fly app host is not a customer signal. It is an alternate-origin and
canonical-routing signal. Because it returns a full 200 instead of converging
to the canonical `https://nothumansearch.ai/` path, agents, crawlers, and
directory tools can see two public origins for the same service.

This is useful as an owner-facing conversion pattern:

1. Public runtime aliases should converge on the canonical machine-readable
   source of truth.
2. Score, monitor, report, badge, OpenAPI, MCP, commerce, and plugin surfaces
   should not split across canonical and platform-host origins.
3. High-score owners should be routed to free monitor/report/badge proof.
4. Partial-score owners should be routed to `/score` before remediation.
5. API-heavy callers should stay on useful docs and API-key plan surfaces
   without checkout-pressure copy.

## Draft Owner-Channel Angle

Usable phrasing:

> Agent-readable sites often have more than one public origin: the branded
> domain, `www`, old domains, platform hosts, badge URLs, OpenAPI files, and
> plugin manifests. NHS checks whether those paths converge on the same
> machine-readable source of truth.

Avoid claiming Fly-host traffic, profiled domains, badge routes, or alias
referrers are customers, endorsements, paid leads, private demand, monitor
registrations, badge-install consent, revenue, completed payments, SEO lift,
uptime proof, platform reliability proof, A2A support, x402/ACP support, paid
placement, preferred inclusion, or score-methodology bypass.

## Execution Gate

Before implementation or external use, refresh canonical redirect behavior for
`nothumansearch.ai`, `www.nothumansearch.ai`, `nothumansearch.com`,
`www.nothumansearch.com`, and `nothumansearch.fly.dev` over HTTP and HTTPS;
refresh the main stats, category, score, monitor, report, fix, profile,
manifest, OpenAPI, MCP, commerce, and aggregate admin surfaces; verify active
Foundry/Owl-owned account identity; check duplicate ledgers and public-action
locks; and avoid `modelcontextprotocol/*` plus `punkpeye/*` surfaces from
`unitedideas`.
