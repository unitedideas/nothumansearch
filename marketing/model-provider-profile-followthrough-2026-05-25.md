# Model Provider Profile Followthrough

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
  `/llms.txt`, `/openapi.yaml`, and `/mcp`.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card claims
  remain blocked.
- MCP `tools/list` returned 11 tools: `search_agents`,
  `get_site_details`, `get_stats`, `submit_site`, `check_url`,
  `verify_mcp`, `register_monitor`, `list_categories`, `get_top_sites`,
  `recent_additions`, and `find_mcp_servers`.
- Aggregate MCP analytics, 7 days: `tools/list=175,081`,
  `initialize=25,122`, and `tools/call=378`.
- Aggregate MCP tool calls, 7 days: `search_agents=148`, `check_url=86`,
  `get_site_details=58`, `get_stats=25`, `submit_site=20`,
  `verify_mcp=13`, `list_categories=7`, `find_mcp_servers=7`,
  `recent_additions=6`, `register_monitor=4`, and `get_top_sites=4`.
- Sanitized aggregate query themes included model gateways, function-calling
  API pricing, OpenRouter/Groq/Fireworks/Mistral-style API discovery, local
  quantized models, agent marketplace terms, and finance/data API lookups.
- Aggregate traffic, 168 hours: `/=3,421`, `/badge/xquik.com.svg=2,666`,
  `/.well-known/commerce.json=1,342`, `/site/xquik.com=1,106`,
  `/.well-known/ai-plugin.json=611`, `/llms.txt=443`,
  `/openapi.yaml=379`, `/api/v1/catalog=320`, `/robots.txt=295`,
  `/api/v1/quote=256`, `/api/v1/checkout=256`, `/api/v1/search=219`,
  `/favicon.ico=189`, `/api/v1/submit=146`, `/digest=127`,
  `/about=90`, `/api/v1=81`, `/.well-known/mcp.json=80`,
  `/.well-known/agent.json=75`, `/score=74`, `/top=73`,
  `/site/openai.com=73`, and `/guide=70`.
- Public AI-tools top-list examples: `8bitconcepts.com=100`,
  `bringyour.ai=100`, `nothumansearch.ai=100`, `amalgix.io=100`,
  `claudereviews.com=100`, `chainray.online=100`, `memestack.ai=100`,
  `sincetmw.ai=100`, `teenanxiety.ai=100`, `teenadhd.ai=100`,
  `childanxiety.ai=100`, and `childadhd.ai=100`. Treat Foundry-owned
  examples as dogfood, and treat health-adjacent examples with clinical
  boundaries.
- Public developer top-list examples: `agentprobe.fly.dev=100`,
  `xquik.com=100`, `deadends.dev=100`, `agentdomainsearch.com=100`,
  `blackveilsecurity.com=100`, `agentndx.ai=100`, `flowzap.xyz=100`,
  `entia.systems=100`, `rendoc.dev=100`, `gptr.dev=100`,
  `wearewarp.com=100`, and `mycloudclaw.com=100`.
- Latest local monitor worker proof, 2026-05-25: completed normally with
  five due monitors; aggregate outcome was two first-check zero-score
  quarantines, two first-check partial or low-score finance/market-data
  style monitors, and one stable high-score monitor.
- Score-fix route checks: `/fix/openai.com` returned 200; high-score
  `/fix/aurelianflo.com` returned the already-meets-target monitor handoff.
  Refresh representative rendered routes before using score-fix copy
  externally.

## Segment

This segment is narrower than the older broad model-gateway and AI-native
tools briefs. It is for owners and channel operators around model providers,
model gateways, inference APIs, function-calling APIs, and AI-tool directories
where users inspect a specific model-provider profile before deciding whether
an agent can trust the provider's public source contract.

Useful owner-side angle:

- `llms.txt` should identify canonical docs, model/catalog pages, API
  references, auth/plan boundaries, pricing policy, deprecation policy,
  support/contact paths, and update policy.
- OpenAPI, API, MCP, or catalog surfaces should expose model, tool-use,
  function-calling, auth, quota, and status contracts only where public
  automated access is intended.
- Agent and commerce manifests should separate public docs, account-gated
  actions, unsupported payment rails, checkout handoffs, and support/refund
  metadata.
- High-score owners should route to free monitor registration, public report
  sharing, and badge/report proof.
- Partial-score owners should start with `/score` and a missing-surface
  checklist before any paid remediation.

## Draft Brief

Agents checking a model provider need a source contract, not a scraped pricing
claim.

For model providers, model gateways, inference APIs, and function-calling
platforms, the machine-readable surface should say which docs, model catalogs,
API references, auth boundaries, quota policies, pricing pages, deprecation
notices, and support paths are canonical. Not Human Search does not certify
model quality, pricing accuracy, benchmark truth, uptime, or provider
preference. It checks whether an agent can inspect the public source before
using it.

High-score providers should use the public report, badge, and free monitor
path. Partial-score providers should run `/score`, fix missing
machine-readable surfaces, and only then consider remediation.

## Owner Routing

- High-score model providers, model gateways, inference APIs, and AI-tool
  directories: route to free monitor registration, public report sharing, and
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

Do not claim model quality, benchmark truth, function-calling reliability,
pricing accuracy, quota freshness, uptime, provider preference, local-runtime
compatibility, package integrity, clinical validity for health-adjacent AI
tools, customer demand, private demand, paid leads, completed payments,
revenue, endorsement, paid placement, preferred inclusion, A2A support while
`/.well-known/agent-card.json` is 404, x402/ACP/SPT/MPP support for NHS, or
score-methodology bypass.

Do not publish raw user-agent strings, private query logs, raw checkout URLs,
payment identifiers, buyer emails, private monitor rows, private score-fix
rows, or private customer identifiers.

## Next Gated Action

Prepare one owner-channel touch, channel post, directory candidate, or
product-handoff test for model-provider, model-gateway, inference API,
function-calling API, or AI-tool directory owners. Before external use,
refresh all live route probes, aggregate admin analytics, monitor worker proof,
representative `/site/{host}` pages, and high-score plus partial-score
`/fix/{host}` routes.
