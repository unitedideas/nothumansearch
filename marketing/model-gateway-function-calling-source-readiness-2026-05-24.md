# Model Gateway And Function-Calling Source Readiness

Run: `business-marketer-not-human-search`, 2026-05-24.

## Aggregate Signal

- Live public stats: `4,179` indexed sites, average score `35`, top category `developer`.
- Live categories: `developer=1,230`, `ai-tools=905`, `data=399`, `finance=195`, `productivity=171`, `ecommerce=146`, `security=113`.
- MCP analytics, last 7 days: `tools/list=172,218`, `initialize=25,455`, `tools/call=397`.
- MCP tool calls, last 7 days: `search_agents=161`, `check_url=91`, `get_site_details=59`, `submit_site=23`, `get_stats=20`, `verify_mcp=14`, `find_mcp_servers=8`, `list_categories=7`, `recent_additions=5`, `get_top_sites=5`, `register_monitor=4`.
- Relevant aggregate query themes include `openrouter groq fireworks mistral LLM API agentic`, `LLM API function calling cheap pricing`, `DeepSeek R1 QwQ MoE GGUF quantized 128GB CPU`, and agent marketplace/payment queries.
- Traffic, last 168 hours: `/api/v1/search=218`, `/api/v1/submit=145`, `/api/v1/catalog=320`, `/api/v1/quote=257`, `/api/v1/checkout=257`, `/.well-known/commerce.json=1,346`, `/.well-known/ai-plugin.json=609`, `/llms.txt=435`, `/openapi.yaml=374`, `/.well-known/mcp.json=78`.

## Live Surface Checks

- `/score`, `/monitor`, and `/report` returned HTTP `200`.
- `/.well-known/agent-card.json` returned HTTP `404`; A2A/Agent Card claims stay blocked.
- High-score score-fix route check: `/fix/nothumansearch.ai` returned the already-meets-target handoff.
- Partial-score score-fix route check: `/fix/manifest.ly` returned paid remediation intake.
- Public `ai-tools` top list is not clean model-gateway proof: it includes Foundry-owned dogfood examples and health/child-anxiety examples, so use query themes plus owner-source requirements instead of claiming category-level model-provider coverage.

## Segment

This segment is narrower than the older model/API provider brief. It is for model gateways, inference API providers, function-calling/tool-use API owners, pricing-plan surfaces, and local/quantized model distribution owners.

Useful owner-side angle:

Agents are looking for model APIs and function-calling options, but a human pricing page is not enough. The owner-side checklist should be machine-readable and probeable:

- `llms.txt` with supported model/API scope and current docs links.
- OpenAPI or equivalent API contract for model, tool-use, auth, pricing, and limits where public.
- MCP or structured API metadata for discovery and status checks.
- Machine-readable plan/auth boundary through catalog, quote, or documented pricing metadata.
- Free monitor registration for high-score providers so readiness drift is visible.
- `/score` first for partial-score providers, then remediation only after concrete missing surfaces are identified.

## Draft Brief

Agents are already querying for model gateways, function calling, cheap LLM APIs, OpenRouter/Groq/Fireworks/Mistral-style routing, and local quantized runtimes.

The gap is not model quality. It is source readiness: can another agent inspect the provider's API contract, pricing/auth boundary, tool-use support, and update surface without scraping a marketing page?

Not Human Search can score that surface and monitor it for drift. High-score providers should use the public report/badge/monitor path. Partial-score providers should start with `/score` and fix the missing machine-readable surfaces before pushing paid remediation.

## Boundaries

Do not claim pricing accuracy, model quality, benchmark truth, function-calling reliability, local-runtime compatibility, package integrity, uptime, privacy/security compliance, customer demand, private demand, paid leads, completed payments, revenue, x402/ACP/MPP support for NHS, A2A support while `/.well-known/agent-card.json` is 404, paid placement, preferred inclusion, or score-methodology bypass.

Do not publish raw user-agent strings, private query logs, raw checkout URLs, payment identifiers, buyer emails, private monitor rows, private score-fix rows, or private customer identifiers.
