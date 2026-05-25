# Collaboration Document Bot Source Readiness

Run: 2026-05-25
Automation: `business-marketer-not-human-search`
Status: no-submit scout artifact; public use still requires account identity verification, duplicate checks, and a sync-state public-action lock.

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, checkout completion, or QLimit/global-queue write was
performed. This is a sanitized scout artifact for a later gated operator.

No raw users, emails, API keys, checkout URLs, payment identifiers, private
monitor rows, private score-fix rows, private query logs, raw user-agent
strings, buyer data, or customer identifiers are included here.

## Evidence

- Public stats: 4,172 indexed sites, average score 35, top category
  `developer`.
- Public categories: `developer=1,230`, `ai-tools=904`, `other=774`,
  `data=402`, `finance=192`, `productivity=171`, `ecommerce=149`,
  `communication=118`, `security=113`, `health=59`, `jobs=26`,
  `education=21`, `news=12`, and `spam=1`.
- Live public surfaces returned 200: `/score`, `/monitor`, `/report`,
  `/newest`, `/top`, `/feed.xml`, `/.well-known/mcp.json`,
  `/.well-known/agent.json`, `/.well-known/commerce.json`,
  `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`,
  `/llms.txt`, and `/openapi.yaml`.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card claims
  remain blocked.
- MCP `tools/list` returned 11 tools: `search_agents`, `get_site_details`,
  `get_stats`, `submit_site`, `check_url`, `verify_mcp`,
  `register_monitor`, `list_categories`, `get_top_sites`,
  `recent_additions`, and `find_mcp_servers`.
- Aggregate MCP analytics, 7 days: `tools/list=175,325`,
  `initialize=25,063`, and `tools/call=378`.
- Aggregate MCP tool calls, 7 days: `search_agents=147`, `check_url=86`,
  `get_site_details=58`, `get_stats=26`, `submit_site=20`,
  `verify_mcp=13`, `list_categories=7`, `find_mcp_servers=7`,
  `recent_additions=6`, `register_monitor=4`, and `get_top_sites=4`.
- Sanitized aggregate query themes included collaboration bot APIs, document
  reads, Lark/Feishu-style workspaces, local events, model gateways, local
  runtimes, ecommerce/product review, finance/ETF data, music/acoustics,
  health-claims research, and agent marketplace payment terms.
- Aggregate traffic, 168 hours: `/=3,377`, `/badge/xquik.com.svg=2,665`,
  `/.well-known/commerce.json=1,342`, `/site/xquik.com=1,108`,
  `/.well-known/ai-plugin.json=612`, `/llms.txt=444`,
  `/openapi.yaml=380`, `/api/v1/catalog=320`, `/robots.txt=296`,
  `/api/v1/checkout=256`, `/api/v1/quote=256`, `/api/v1/search=219`,
  `/favicon.ico=189`, `/api/v1/submit=146`, `/digest=127`,
  `/about=91`, `/api/v1=82`, `/.well-known/mcp.json=81`,
  `/.well-known/agent.json=75`, `/score=74`, `/top=73`, and
  `/guide=70`.
- Public productivity top-list examples: `tabai.dev=100`, `blooio.com=100`,
  `barevalue.com=100`, `simplepdf.com=100`, `angshumangupta.com=80`,
  `attio.com=80`, `monday.com=70`, `berger.team=70`,
  `planharmony.com=70`, `pinmeto.com=70`, `itsconvo.com=70`, and
  `catchintent.com=70`.
- Public communication top-list examples: `postalform.com=100`,
  `mail.misar.io=90`, `resend.com=75`, `secondsim.co.uk=70`,
  `slack.com=60`, `api.slack.com=60`, `pantrypersona.com=60`,
  `kweenkl.com=55`, `plain.com=50`, `loops.so=50`, `adsagent.org=50`,
  and `muddyterrain.com=50`.
- Public developer top-list examples included high-score developer surfaces:
  `agentprobe.fly.dev=100`, `xquik.com=100`, `deadends.dev=100`,
  `agentdomainsearch.com=100`, `blackveilsecurity.com=100`,
  `agentndx.ai=100`, `flowzap.xyz=100`, and `entia.systems=100`.
  Treat Foundry-owned dogfood examples as dogfood, not third-party proof.
- Latest local monitor worker proof, 2026-05-25: completed normally with
  five due monitors; aggregate outcome was two first-check zero-score
  quarantines, two first-check partial/low-score finance or market-data style
  monitors, and one stable high-score monitor.
- Score-fix route check: high-score `/fix/nothumansearch.ai` returned the
  already-meets-target handoff. Partial-score `/fix/manifest.ly` returned
  HTTP 200 but did not expose a stable grep-able headline in this run, so
  refresh the rendered route before using score-fix copy externally.

## Segment

This segment is narrower than the older productivity, communication,
document-workflow, and package/dependency briefs. It is for collaboration
platforms, document workspaces, bot APIs, forms/workflow tools, email/chat
products, and internal-knowledge products whose public surfaces agents may
inspect before reading, summarizing, routing, or automating documents.

Useful owner-side angle:

- `llms.txt` should identify canonical bot/API docs, document-read boundaries,
  workspace permissions, rate limits, contact/support paths, and update policy.
- OpenAPI, API, or MCP surfaces should expose document, bot, webhook, search,
  export, and auth contracts only where public automated access is intended.
- Agent and commerce manifests should separate public docs, account-gated
  actions, unsupported payment rails, checkout handoffs, and support/refund
  metadata.
- Schema.org and feeds are useful where docs, changelogs, forms, or knowledge
  pages change over time.
- High-score owners should route to free monitor registration, public report
  sharing, and badge/report proof.
- Partial-score owners should start with `/score` and a missing-surface
  checklist before any paid remediation.

## Draft Brief

Agents reading workspace documents need an access contract, not a scraped help
center.

For collaboration platforms, document tools, bot APIs, form products, and
chat/email workspaces, the machine-readable surface should say which document
read, search, webhook, export, auth, rate-limit, support, and update-policy
docs are canonical. Not Human Search does not certify workflow quality,
privacy compliance, or integration reliability. It checks whether an agent can
inspect the public source before using it.

High-score workspace owners should use the public report, badge, and free
monitor path. Partial-score owners should run `/score`, fix missing
machine-readable surfaces, and only then consider remediation.

## Owner Routing

- High-score collaboration, document, bot, form, chat, and email workspace
  owners: route to free monitor registration, public report sharing, and
  badge/report proof.
- Partial-score owners: route to `/score` first, then a missing-surface
  checklist before score-fix remediation.
- API-heavy callers: route to API-key/catalog surfaces only when NHS docs
  remain useful.
- A2A or Agent Card claims stay blocked until `/.well-known/agent-card.json`
  exists.
- Monitor quarantine cases stay private/admin-only and must not be used as
  public monitor-growth proof.

## Claims To Avoid

Do not claim workspace quality, document correctness, workflow reliability,
bot safety, email/chat deliverability, privacy compliance, security
certification, data-retention compliance, permission correctness, OCR/export
quality, integration uptime, customer demand, private demand, paid leads,
completed payments, revenue, endorsement, paid placement, preferred inclusion,
A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/SPT/MPP
support for NHS, or score-methodology bypass.

Do not publish raw user-agent strings, private query logs, raw checkout URLs,
payment identifiers, buyer emails, private monitor rows, private score-fix
rows, or private customer identifiers.

## Next Gated Action

Prepare one owner-channel touch, channel post, directory candidate, or
product-handoff test for collaboration, document-workspace, bot API, form,
chat, or email-workflow owners. Before external use, refresh all live route
probes, aggregate admin analytics, monitor worker proof, representative
`/site/{host}` pages, and high-score plus partial-score `/fix/{host}` routes.
