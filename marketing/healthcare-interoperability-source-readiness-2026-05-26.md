# Healthcare Interoperability Source Readiness

Run: 2026-05-26
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
- Public health category: 59 sites, average score 42.
- Live public health top-list examples: `emorahealth.com=100`,
  `zgts.in=100`, `opdstar.com=80`, `plith.ai=80`,
  `monarchinitiative.org=65`, `fhirfly.io=65`, `lakmesalon.in=60`,
  and `tau.edu.gy=50`.
- Adjacent public security/compliance examples: `hipaaagent.ai=80`,
  `feedoracle.io=100`, `ansvar.eu=100`, `agent-module.dev=95`,
  `tickerr.ai=85`, `rnwy.com=80`, `easysend.co=80`, and `file.kiwi=80`.
- Live public surfaces returned 200: `/llms.txt`,
  `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/commerce.json`, `/api/v1/catalog`, `/api/v1`, and `/mcp`.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card
  claims remain blocked.
- Aggregate MCP analytics, 7 days: `tools/list=175,874`,
  `initialize=25,558`, and `tools/call=377`.
- Aggregate MCP tool calls, 7 days: `search_agents=148`, `check_url=86`,
  `get_site_details=57`, `get_stats=27`, `submit_site=20`,
  `verify_mcp=13`, `list_categories=7`, `find_mcp_servers=7`,
  `recent_additions=5`, `register_monitor=4`, and `get_top_sites=3`.
- Sanitized aggregate query themes included health/genetics, model/API,
  ecommerce/product search, local events, finance/market data, publisher
  feeds, and agent marketplace/payment terms.
- Aggregate traffic, 168 hours: `/=3,410`, `/badge/xquik.com.svg=2,646`,
  `/.well-known/commerce.json=1,345`, `/site/xquik.com=1,106`,
  `/.well-known/ai-plugin.json=607`, `/llms.txt=441`,
  `/openapi.yaml=377`, `/api/v1/catalog=323`, `/api/v1/checkout=256`,
  `/api/v1/quote=256`, `/api/v1/search=219`, `/api/v1/submit=146`,
  `/top=75`, `/score=74`, and `/site/openai.com=73`.
- Latest local monitor worker proof, 2026-05-25: completed normally with
  five due monitors; aggregate outcome was two first-check zero-score
  quarantines, two first-check partial or low-score checks, and one stable
  high-score check.

## Segment

This segment is narrower than the older generic healthcare, consumer-health,
and genetic-wellness briefs. It is for FHIR, clinical workflow, health-data,
patient intake, provider directory, health AI, and HIPAA-adjacent tool owners
whose public surfaces agents may inspect before reading docs, routing data, or
attempting an integration.

Useful owner-side angle:

- `llms.txt` should identify canonical API docs, FHIR or health-data
  boundaries, auth requirements, rate limits, support/contact paths, update
  policy, and intended automated-access limits.
- OpenAPI, API, or MCP surfaces should expose only public integration
  contracts the owner intends agents to call.
- Agent and commerce manifests should separate public docs, account-gated
  actions, unsupported payment rails, checkout handoffs, support/refund
  metadata, and privacy/compliance disclaimers.
- High-score healthcare and health-data owners should route to free monitor
  registration, public report sharing, and badge/report proof.
- Partial-score owners should start with `/score` and a missing-surface
  checklist before any paid remediation.
- Quarantined monitor cases remain private/admin-only until review.

## Draft Brief

Healthcare APIs need a source contract before agents touch them.

For FHIR tools, clinical workflow products, health-data APIs, patient-intake
systems, provider directories, and HIPAA-adjacent products, the useful
machine-readable surface is not marketing copy. It is the public contract an
agent can inspect: API shape, auth boundary, data scope, update policy,
support path, and what automated access is not allowed to do.

Not Human Search does not certify medical accuracy, HIPAA compliance, privacy
controls, clinical safety, or integration reliability. It checks whether an
agent can verify the public source surface before relying on it.

High-score owners should use the public report, badge, and free monitor path.
Partial-score owners should run `/score`, fix the missing machine-readable
surfaces, and only then consider remediation.

## Boundaries

Do not claim medical accuracy, clinical endorsement, HIPAA compliance,
privacy compliance, regulatory compliance, security certification, patient-data
safety, FHIR correctness, integration reliability, uptime, provider-directory
accuracy, diagnosis quality, care quality, live data freshness, customer
demand, private demand, paid leads, completed payments, revenue, endorsement,
paid placement, preferred inclusion, A2A support while
`/.well-known/agent-card.json` is 404, x402/ACP/SPT/MPP support for NHS, or
score-methodology bypass.

Do not publish raw user-agent strings, private query logs, private monitor
rows, raw checkout URLs, payment identifiers, buyer emails, private score-fix
rows, or private customer identifiers.

## Next Gated Action

Prepare exactly one owner-channel touch, channel post, directory candidate, or
product-handoff test for FHIR, clinical workflow, health-data, patient intake,
provider directory, health AI, or HIPAA-adjacent product owners. Before
external use, refresh public stats, categories, representative health/security
top lists, `/score`, `/monitor`, `/report`, representative `/site/{host}`
pages, high-score and partial-score `/fix/{host}` routes, MCP `tools/list`,
machine-readable manifests, catalog/quote/checkout surfaces, aggregate admin
MCP/traffic data, and latest monitor worker proof.
