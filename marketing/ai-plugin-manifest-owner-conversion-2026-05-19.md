# AI Plugin Manifest Owner Conversion Scout

Run: 2026-05-19T08:16Z
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

Public surfaces checked:

- `/api/v1/stats`: `total_sites=4174`, `avg_score=35`,
  `top_category=developer`.
- `/api/v1/categories`: developer 1,237; ai-tools 900; other 765; data 399;
  finance 199; productivity 173; ecommerce 149; communication 119; security
  115; health 57; jobs 27; education 21; news 12; spam 1.
- `/.well-known/ai-plugin.json`: HTTP 200; points agents to
  `https://nothumansearch.ai/openapi.yaml`, advertises no auth, the public
  score/search/check/submit/monitor surfaces, and the `/mcp` server as the
  richer tool path.
- `/.well-known/mcp.json`: HTTP 200 with 11 tools.
- `/.well-known/agent.json`, `/.well-known/commerce.json`, `/api/v1/catalog`,
  `/api/v1`, `/openapi.yaml`, `/monitor`, `/score`, `/guide`, and `/about`:
  HTTP 200.
- `/.well-known/agent-card.json`: HTTP 404, so A2A/Agent Card claims remain
  gated.
- Public top examples for developer and ai-tools categories still include
  score-100 examples; Foundry-owned examples must be labeled before external
  use.

Aggregate admin evidence, sanitized:

- Last 168 hours traffic: `/=3562`, `/badge/xquik.com.svg=1708`,
  `/.well-known/commerce.json=1672`, `/.well-known/ai-plugin.json=796`,
  `/llms.txt=510`, `/site/xquik.com=485`, `/openapi.yaml=469`,
  `/api/v1/catalog=372`, `/api/v1/checkout=345`, `/api/v1/quote=345`,
  `/api/v1/submit=147`, `/top=116`, `/api/v1=97`,
  `/.well-known/mcp.json=96`, `/about=95`, `/guide=92`, `/newest=89`,
  `/score=78`, `/api/v1/check=60`.
- Referrer aggregates included `google.com=542`, `/score=117`, and normal
  self-referrer/domain variants. No private referrer rows are included here.
- Last 7 days MCP aggregate returned a low-volume/zeroed method view in this
  run, with coarse query themes around API, developer, RAG, search, research,
  and database use. Treat it as directional only; refresh before publication.

## Read

`/.well-known/ai-plugin.json` is now a material discovery surface in its own
right. It is ahead of `/llms.txt`, `/openapi.yaml`, `/api/v1/catalog`, and
normal guide/about traffic in the current 168-hour aggregate. That suggests
agents and crawlers still look for plugin manifests even though modern agent
integrations prefer MCP, OpenAPI, `llms.txt`, and commerce/agent manifests.

The useful owner-channel angle is not "ChatGPT plugins are back." It is:

1. Legacy and current agent crawlers still inspect plugin manifests.
2. A useful manifest should point at the same maintained OpenAPI, support,
   legal, contact, monitor, and commerce surfaces as the rest of the machine
   packet.
3. NHS can test and monitor whether those public files drift apart.

## Proposed Gated Test

Create one owner-channel or product-handoff test for teams that already have
OpenAPI or `llms.txt` but stale or missing plugin manifests:

- Route technical owners to `/score` first, then `/monitor` so manifest drift
  is caught after launch.
- Route repeated programmatic users from the plugin/OpenAPI path to API-key
  plans only after the docs remain useful and self-serve.
- Route low-score site owners to remediation only after a fresh public score
  proves a concrete gap.
- Keep high-score profiles on monitor/report/badge proof rather than paid
  remediation.

## Acceptance Guard

Before implementation or external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`,
  `/.well-known/ai-plugin.json`, `/openapi.yaml`, `/llms.txt`,
  `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/agent-card.json`, `/.well-known/commerce.json`,
  `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/api/v1`,
  `/score`, `/monitor`, `/guide`, `/about`, representative high-score and
  partial-score `/site/{host}` pages, high-score and partial-score
  `/fix/{host}` routes, aggregate `/api/v1/admin/traffic?hours=168`, and
  aggregate `/api/v1/admin/mcp?days=7`.
- Verify active Foundry/Owl-owned account identity before public use, check
  duplicate ledgers and sync-state public-action locks, and avoid
  modelcontextprotocol/* plus punkpeye/* surfaces from `unitedideas`.
- Use only aggregate route/referrer counts and public URLs in committed
  artifacts.
- Do not imply plugin visitors, listed domains, searched domains, or profiled
  domains are customers, endorsements, paid leads, private demand, monitor
  registrations, completed payments, revenue, A2A support, current ChatGPT
  plugin ecosystem adoption, x402/ACP support, paid ranking placement,
  preferred inclusion, or score-methodology bypass.
