# Score-Fix Owner Conversion Packet

Date: 2026-05-16
Automation: `business-marketer-not-human-search`
Source agent: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a sanitized no-submit packet for a later product or channel operator.

## Fresh Evidence

Public surfaces checked:

- `https://nothumansearch.ai/api/v1/stats`: 4,176 indexed sites, average score 35, top category `developer`.
- `https://nothumansearch.ai/api/v1/categories`: largest public buckets are `developer=1229`, `ai-tools=901`, `other=770`, `data=403`, `finance=200`, `productivity=172`, `ecommerce=152`, `communication=117`, `security=115`.
- `https://nothumansearch.ai/llms.txt`: advertises 4,176+ sites and 11 MCP tools.
- `https://nothumansearch.ai/.well-known/mcp.json`: live and advertises the MCP endpoint and 11 tool definitions.
- `https://nothumansearch.ai/.well-known/agent.json`: live and advertises REST API, OpenAPI, MCP, commerce catalog, quote, checkout, and API-key subscription surfaces.
- `https://nothumansearch.ai/.well-known/commerce.json`: live and lists score-fix remediation plus Starter/Pro/Scale API plans.
- `https://nothumansearch.ai/api/v1/catalog`: live and lists `nhs_geo_fix_my_score`, `nhs_api_starter`, `nhs_api_pro`, and `nhs_api_scale`.
- `https://nothumansearch.ai/score`: HTTP 200 public score checker.
- `https://nothumansearch.ai/monitor`: HTTP 200 free monitor signup page.
- `https://nothumansearch.ai/fix/nothumansearch.ai`: HTTP 200 high-score handoff page, title says the domain already meets the NHS score target.
- `https://nothumansearch.ai/fix/cohere.com`: HTTP 200 remediation intake page for a lower-score target.
- `https://nothumansearch.ai/.well-known/agent-card.json`: HTTP 404, so strict Agent Card directory submissions remain gated.

Aggregate admin signals checked:

- MCP analytics, last 7 days: `tools/list=130481`, `initialize=17720`, `tools/call=337`.
- Top called MCP tools: `search_agents=209`, `get_site_details=42`, `find_mcp_servers=23`, `get_stats=15`, `verify_mcp=15`, `check_url=13`.
- Traffic, last 336 hours: `/.well-known/commerce.json=1454`, `/badge/xquik.com.svg=1081`, `/api/v1/catalog=328`, `/api/v1/quote=303`, `/api/v1/checkout=303`, `/site/xquik.com=234`, `/top=137`, `/newest=109`, `/score=68`.
- Monitor aggregate: `active=1`, `quarantined=1`, quarantine reason `bounded rerun still zero score`; monitor actions in the last 30 days include `request_score_rerun=1` and `keep_quarantined=1`.
- Score-fix aggregate: 11 total rows; `real_candidate pending=2`; test-like rows remain separate. No raw rows were written.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, or private query logs were written.

## Read

The prior blocker for pushing score-fix owner conversion was the high-score `/fix/{host}` trust gap. That gap is now repaired: high-score domains see a monitor/report handoff instead of a paid remediation form, while lower-score domains still see the score-fix intake.

This makes a gated owner-channel packet usable again, with a clear split:

- High-score owners: monitor score changes and share the badge/report.
- Low-score owners: paid remediation for missing public agent-readiness signals.
- API-heavy agents: paid API plans after quota or repeated machine-readable discovery.

Do not treat the two real pending score-fix rows as public demand proof. They are private sales workflow evidence only, already covered by the admin follow-up boundary.

## Draft Channel Copy

Subject or title:

`Agent-readiness score checks for site owners`

Short post or directory note:

`Not Human Search scores public websites on whether agents can use them without guessing: llms.txt, OpenAPI, structured APIs, MCP, AI-friendly robots rules, plugin metadata, and schema.`

`Site owners can check a domain for free, monitor regressions, or request a remediation pull request when public agent-readiness signals are missing. High-score domains are routed to monitoring/reporting rather than a paid fix.`

Proof links:

- `https://nothumansearch.ai/score`
- `https://nothumansearch.ai/monitor`
- `https://nothumansearch.ai/api/v1/catalog`
- `https://nothumansearch.ai/llms.txt`
- `https://nothumansearch.ai/.well-known/mcp.json`

## Guardrails

- Do not claim private demand, completed payments, revenue, customer endorsement, paid ranking placement, preferred inclusion, ACP/x402 support for NHS, or score-methodology bypass.
- Do not imply profiled domains, badge-heavy domains, or score-fix candidates are customers, endorsements, paid leads, or private demand signals.
- Do not expose raw emails, private monitor rows, private score-fix rows, raw checkout URLs, payment identifiers, or private query logs.
- Do not submit to strict A2A/Agent Card directories until `/.well-known/agent-card.json` exists or a target explicitly accepts the current `/.well-known/agent.json` service manifest.

## Next Gated Action

Use this packet through a channel operator for one site-owner or developer-tool channel. The operator must:

1. Refresh `/api/v1/stats`, `/api/v1/categories`, `/score`, `/monitor`, `/fix/{high-score-host}`, `/fix/{low-score-host}`, `/.well-known/commerce.json`, `/.well-known/agent.json`, `/api/v1/catalog`, `/llms.txt`, and `/.well-known/mcp.json`.
2. Verify active account identity for the selected Foundry/Owl-owned channel.
3. Check duplicate fingerprints in `marketing/social-post-ledger.json` if applicable plus sync-state public-action locks.
4. Record the public URL or exact blocker after the action.
