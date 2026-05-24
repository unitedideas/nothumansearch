# Cyber Threat and Risk Intelligence Source Readiness

Run: 2026-05-24T13:00:00Z
Automation: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, checkout completion, or QLimit/global-queue write was
performed. This is a sanitized scout artifact for a later gated operator.

No raw users, emails, API keys, checkout URLs, payment identifiers, private
monitor rows, private score-fix rows, private query logs, buyer data, or raw
customer data are included here.

## Evidence

- Public stats: 4,177 indexed sites, average score 35, top category
  `developer`.
- Public categories: `developer=1230`, `ai-tools=904`, `other=781`,
  `data=399`, `finance=195`, `productivity=171`, `ecommerce=146`,
  `communication=119`, `security=113`, `health=59`, `jobs=26`,
  `education=21`, `news=12`, and `spam=1`.
- Public discovery and owner surfaces returned 200: `/score`, `/monitor`,
  `/report`, `/newest`, `/top`, `/.well-known/mcp.json`,
  `/.well-known/agent.json`, `/api/v1/catalog`, `/llms.txt`,
  `/openapi.yaml`, and `/feed.xml`.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card
  directory claims remain blocked.
- JSON-RPC `/mcp` `tools/list` returned 11 tools:
  `search_agents`, `get_site_details`, `get_stats`, `submit_site`,
  `check_url`, `verify_mcp`, `register_monitor`, `list_categories`,
  `get_top_sites`, `recent_additions`, and `find_mcp_servers`.
- Public security top-list examples: `feedoracle.io=100`, `ansvar.eu=100`,
  `agent-module.dev=95`, `tickerr.ai=85`, `rnwy.com=80`,
  `easysend.co=80`, `qnsp.cuilabs.io=70`, `hefestoai.narapallc.com=70`,
  `usefathom.com=65`, `certman.app=65`, `twitterapi.io=65`, and
  `txrisk.xyz=60`.
- Public data top-list examples: `api.contrastcyber.com=100`,
  `api.headlessoracle.com=95`, `dchub.cloud=95`, `daedalmap.com=90`,
  `api.theartofservice.com=90`, `api.agentry.com=90`,
  `api.socialintel.dev=90`, `blocklens.co=90`, `api.meacheal.ai=85`,
  `meetlark.ai=85`, `app.daedalmap.com=80`, and `sofiastage.com=80`.
- Aggregate MCP analytics, 7 days: `tools/list=172992`,
  `initialize=26752`, and `tools/call=404`.
- Aggregate MCP tool calls, 7 days: `search_agents=171`, `check_url=85`,
  `get_site_details=60`, `submit_site=21`, `get_stats=20`,
  `verify_mcp=14`, `list_categories=8`, `find_mcp_servers=8`,
  `recent_additions=7`, `get_top_sites=6`, and `register_monitor=4`.
- Aggregate client families included registry, MCP catalog, scoring, Cherry
  Studio, Claude Code, Python, and Node clients.
- Aggregate traffic, 168 hours: `/=3369`, `/badge/xquik.com.svg=2532`,
  `/.well-known/commerce.json=1384`, `/site/xquik.com=974`,
  `/.well-known/ai-plugin.json=630`, `/llms.txt=447`,
  `/openapi.yaml=383`, `/api/v1/catalog=326`, `/robots.txt=298`,
  `/badge/aidevboard.com.svg=277`, `/api/v1/checkout=265`,
  `/api/v1/quote=265`, `/badge/8bitconcepts.com.svg=256`, and
  `/api/v1/search=214`.
- Latest local monitor-check proof remains clean for the most recent run:
  2026-05-18 processed one due monitor and completed without quarantine.

## Segment

Cyber, threat-intelligence, risk-scoring, compliance automation, social
intelligence, and fraud-monitoring owners need public source contracts agents
can inspect before using or recommending the data:

- source provenance, update cadence, and stale-data boundaries;
- API/OpenAPI/MCP contract docs for lookups and enrichment;
- pricing, plan, auth, and rate-limit metadata for automated callers;
- machine-readable confidence, caveat, contact, refund, and support metadata;
- monitorable readiness so security and risk surfaces do not drift silently.

This segment is not proof that NHS certifies security vendors or validates
threat intelligence. It is a readiness packet for owners whose products already
depend on agent-readable contracts and careful caveats.

## Owner Routing

- High-score cyber, risk, data, and compliance owners: route to free monitor
  registration, public report sharing, and badge/report proof.
- Partial-score security or risk-data owners: route to `/score` first, then a
  missing-surface checklist before any score-fix remediation.
- API-heavy callers: route to API-key/catalog surfaces only when NHS docs
  remain useful.
- Directory or public-channel use stays gated behind account identity,
  duplicate checks, a public-action lock, and fresh surface probes.

## Claims To Avoid

Do not claim threat-intelligence accuracy, security certification, fraud
detection quality, compliance certification, risk-score correctness, privacy
compliance, uptime, data freshness, source completeness, customer demand,
private demand, paid leads, completed payments, revenue, endorsement, paid
placement, preferred inclusion, A2A support while
`/.well-known/agent-card.json` is 404, x402/ACP/MPP support for NHS, or
score-methodology bypass.
