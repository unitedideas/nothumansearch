# NHS model/API owner-channel brief

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-15T03:40Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a sanitized scout artifact for a later gated channel or product operator.

## Fresh Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: 4,172 indexed sites, average score 35, top category `developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories; `developer=1229`, `ai-tools=899`, `data=403`, `finance=200`.
- `https://nothumansearch.ai/.well-known/mcp.json`: advertises 11 tool definitions and the public/audit-only category split.
- `https://nothumansearch.ai/.well-known/agent-card.json`: HTTP 404, so A2A-style directory work remains blocked until compatibility is added or intentionally documented.
- `https://nothumansearch.ai/api/v1/top?category=developer&limit=10`: public top examples include agent-commerce readiness, X automation, package intelligence, dependency/security scanning, agent domain search, and MCP directory surfaces.
- `https://nothumansearch.ai/api/v1/top?category=ai-tools&limit=10`: current top list still includes Foundry-owned dogfood examples, so use it as readiness-pattern evidence only, not third-party market proof.

Aggregate admin evidence, last 7 days:

- MCP `tools/list`: 118,953 calls.
- MCP `initialize`: 15,680 calls.
- MCP `tools/call`: 356 calls.
- Top called tools: `search_agents=209`, `get_site_details=45`, `find_mcp_servers=27`, `get_stats=19`, `check_url=16`, `verify_mcp=15`, `get_top_sites=13`, `recent_additions=9`, `list_categories=3`.
- Model/API-provider query themes included repeated free LLM API searches, OpenRouter/free-model phrasing, Hugging Face Pro subscription features/pricing, MiniMax benchmark/coding evaluation, function-calling/tool-use options, and coding-agent developer-tool searches.

Aggregate admin traffic, last 336 hours:

- `/.well-known/commerce.json`: 1,344 requests.
- `/llms.txt`: 443 requests.
- `/openapi.yaml`: 431 requests.
- `/api/v1/catalog`: 302 requests.
- `/api/v1/quote`: 282 requests.
- `/api/v1/checkout`: 282 requests.
- `/.well-known/mcp.json`: 94 requests.
- `/api/v1`: 93 requests.
- Google referrers: 202 combined requests from `google.com` and `www.google.com`.

Private workflow aggregates checked:

- Monitor status: `active=1`, `quarantined=1`, quarantine reason `bounded rerun still zero score`.
- Monitor actions in the last 30 days: `request_score_rerun=1`, `keep_quarantined=1`.
- Score-fix aggregate: 11 rows; `real_candidate pending=2`; no real paid or real lead row was exposed.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, or private query logs were written.

## Read

NHS MCP usage is showing agents looking for model providers, free or low-cost LLM API options, tool-use/function-calling support, model subscription/pricing details, and coding-agent benchmarks.

The safe owner-side angle is not "NHS ranks model providers" or "NHS recommends a best model." It is that model/API providers need agent-readable commercial and technical surfaces:

- `llms.txt` for agent instructions.
- OpenAPI or structured API docs for quote and usage flows.
- MCP or probeable tool endpoints where available.
- Machine-readable pricing/catalog metadata.
- Clear unsupported-rail language for agent-commerce surfaces.
- Score monitoring so public agent affordances do not silently regress.

## Channel Brief

Short:

NHS MCP usage now includes repeated model/API-provider discovery: free LLM APIs, tool-use support, Hugging Face subscription details, MiniMax coding benchmarks, and OpenRouter/free-model searches. The owner-side takeaway is simple: model providers need public, probeable surfaces agents can inspect before recommending or using them.

Long:

Agents are using Not Human Search to look for model/API options, pricing details, and coding-agent capabilities. Those queries are volatile and vendor-specific, so NHS should not present itself as a pricing authority or benchmark source of truth.

The useful marketing angle is source readiness. Model providers, gateways, and benchmark tools should make their product catalog, pricing, API docs, tool-use support, and terms machine-readable. NHS can show which public surfaces are probeable and route owners toward score-fix or free monitoring when those surfaces disappear.

## Suggested Follow-Up

Prepare a gated channel operator packet for model/API providers and AI developer-tool audiences:

- Use this brief for developer-tool communities, model-provider owner channels, API directories, or agent-commerce directories.
- Keep the copy boundary explicit: NHS helps agents find and verify machine-readable source surfaces; it does not certify pricing, benchmark truth, or model quality.
- If product work follows, add a docs/example guard for model-provider queries: "Use NHS to find agent-readable model/API sources; verify pricing, benchmarks, and availability with the source."

## Publication Guard

Before any external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=developer`, `/api/v1/top?category=ai-tools`, `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, and `/api/v1/admin/mcp?days=7`.
- Verify active channel account identity.
- Check `marketing/social-post-ledger.json` if a social/channel post is involved.
- Check sync-state public-action locks and `outreach/distribution_log.csv`.
- Do not claim private demand, completed payments, revenue, customer endorsement, pricing accuracy, benchmark accuracy, model-quality certification, paid ranking placement, preferred inclusion, ACP/x402 support for NHS, or score-methodology bypass.
