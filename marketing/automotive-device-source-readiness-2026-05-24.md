# Automotive Device Source Readiness

Run: 2026-05-24T19:08Z
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, checkout completion, or QLimit/global-queue write was
performed. This is a sanitized scout artifact for a later gated operator.

No raw users, emails, API keys, checkout URLs, payment identifiers, private
monitor rows, private score-fix rows, buyer data, private query logs, raw user
agents, or customer identifiers are included here.

## Evidence

- Public stats: 4,178 indexed sites, average score 35.
- Public discovery and owner surfaces checked live: `/score`, `/monitor`,
  `/report`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`,
  `/api/v1/catalog`, `/llms.txt`, and `/openapi.yaml` returned 200.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card claims
  remain blocked.
- Aggregate MCP analytics, 7 days: `tools/list=172007`,
  `initialize=25921`, and `tools/call=378`.
- Aggregate MCP tool calls, 7 days: `search_agents=155`,
  `check_url=85`, `get_site_details=54`, `submit_site=21`,
  `get_stats=20`, `verify_mcp=14`, and `register_monitor=4`.
- Aggregate query themes included self-driving device pricing, Linux gaming
  device/runtime UX, local quantized models, scanner hardware, electronics
  retail, and product-review lookups.
- Public developer top-list examples: `agentprobe.fly.dev=100`,
  `xquik.com=100`, `mcp.depscope.dev=100`, `deadends.dev=100`,
  `agentdomainsearch.com=100`, `blackveilsecurity.com=100`,
  `agentndx.ai=100`, `entia.systems=100`, `rendoc.dev=100`,
  `gptr.dev=100`, `wearewarp.com=100`, and `mycloudclaw.com=100`.
- Public data top-list examples: `api.contrastcyber.com=100`,
  `api.headlessoracle.com=95`, `dchub.cloud=95`, `daedalmap.com=90`,
  `api.theartofservice.com=90`, `api.agentry.com=90`,
  `api.socialintel.dev=90`, `blocklens.co=90`, `api.meacheal.ai=85`,
  `meetlark.ai=85`, `app.daedalmap.com=80`, and `sofiastage.com=80`.
- Aggregate traffic, 168 hours: `/=3362`, `/badge/xquik.com.svg=2567`,
  `/.well-known/commerce.json=1354`, `/site/xquik.com=1008`,
  `/.well-known/ai-plugin.json=615`, `/llms.txt=436`,
  `/openapi.yaml=375`, `/api/v1/catalog=320`, `/api/v1/checkout=259`,
  `/api/v1/quote=259`, and `/api/v1/search=216`.
- Latest local monitor worker proof remains the 2026-05-18 clean run:
  one due monitor processed and completed without quarantine.

## Segment

Automotive, self-driving-device, local runtime, electronics, scanner, and
hardware owners need public source contracts agents can inspect before they
quote, compare, install, recommend, or monitor anything:

- device/model/SKU identifiers and compatibility boundaries;
- firmware, driver, package, update, and install metadata;
- pricing, plan, availability, warranty, support, and refund metadata;
- API/OpenAPI/MCP/catalog surfaces for automated checks;
- caveats for unsupported platforms, safety limits, and stale product data;
- monitorable readiness so agent-facing docs do not drift silently.

This segment is not proof that NHS validates vehicle safety, hardware
compatibility, product quality, or pricing. It is an owner-channel readiness
packet for hardware and device-adjacent sites whose public facts agents may try
to reuse.

## Owner Routing

- High-score hardware, runtime, or device-data owners: route to free monitor
  registration, public report sharing, and badge/report proof.
- Partial-score owners: route to `/score` first, then a missing-surface
  checklist before score-fix remediation.
- API-heavy callers: route to API-key/catalog surfaces only when NHS docs
  remain useful.
- Public-channel use stays gated behind active account identity, duplicate
  checks, a public-action lock, and fresh surface probes.

## Claims To Avoid

Do not claim vehicle safety, self-driving performance, hardware compatibility,
firmware quality, driver reliability, install safety, product quality, inventory
accuracy, price freshness, warranty quality, support SLA, security/privacy
compliance, regulatory compliance, customer demand, private demand, paid leads,
completed payments, revenue, endorsement, paid placement, preferred inclusion,
A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/MPP support
for NHS, or score-methodology bypass.
