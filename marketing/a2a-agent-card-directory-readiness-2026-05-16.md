# A2A Agent Card Directory Readiness Packet

Date: 2026-05-16
Automation: `business-marketer-not-human-search`
Source agent: `business-marketer-not-human-search`

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a sanitized no-submit scout packet for a later product or channel operator.

## Fresh NHS Evidence

- `https://nothumansearch.ai/api/v1/stats`: 4,176 indexed sites, average score 35, top category `developer`.
- `https://nothumansearch.ai/api/v1/categories`: largest public buckets are `developer=1229`, `ai-tools=901`, `data=403`, `finance=200`, `productivity=172`, `ecommerce=152`, `communication=117`, `security=115`.
- `https://nothumansearch.ai/.well-known/mcp.json`: live and advertises 11 tools.
- JSON-RPC `tools/list`: 11 tools: `search_agents`, `get_site_details`, `get_stats`, `submit_site`, `check_url`, `verify_mcp`, `register_monitor`, `list_categories`, `get_top_sites`, `recent_additions`, `find_mcp_servers`.
- `https://nothumansearch.ai/.well-known/agent.json`: live and advertises REST API, OpenAPI, MCP, commerce catalog, quote, checkout, and API-key subscription surfaces.
- `https://nothumansearch.ai/.well-known/commerce.json`: live and lists score-fix remediation plus Starter/Pro/Scale API plans.
- `https://nothumansearch.ai/api/v1/catalog`: live and lists score-fix plus API subscription products.
- `https://nothumansearch.ai/.well-known/agent-card.json`: HTTP 404.

Aggregate admin signals, last 7 days and last 336 hours:

- MCP analytics: 30 top-query rows; repeated theme words include agent, card, autonomous, tools, GitHub, OpenRouter, price, and cost.
- Traffic: `/.well-known/commerce.json=1442`, `/api/v1/catalog=324`, `/api/v1/checkout=301`, `/api/v1/quote=301`, `/.well-known/ai-plugin.json=694`, `/llms.txt=464`, `/openapi.yaml=443`.
- Badge/profile loop remains material: `/badge/xquik.com.svg=1071`, `/badge/aidevboard.com.svg=355`, `/badge/8bitconcepts.com.svg=344`.
- Monitor aggregate remains `active=1`, `quarantined=1`; score-fix aggregate remains `real_candidate pending=2`, with no raw rows exposed.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, or private query logs were written.

## External Directory Scan

Current A2A/Agent Card directory surfaces worth gating behind an Agent Card compatibility repair:

- AgentRolodex: public A2A directory that lists both agents and services and says listings publish standard `/.well-known/agent.json` cards.
- A2A Registry: registration/API docs describe Agent Cards and JSON-RPC/REST registration.
- a2a.directory: public A2A agent directory; current listings include agents with REST, MCP, and A2A discovery endpoints.
- a2alist.ai: public A2A/x402 directory; fit is only after the NHS card truthfully declares unsupported x402/ACP boundaries.
- A2ABase registry: submission language requires a current AgentCard.
- TangleTwo: public registry for modular agents and A2A-style discovery; fit is unproven until card compatibility exists.
- `prassanna-ravishankar/a2a-registry`: GitHub PR-based registry; repository notes `/.well-known/agent-card.json` is preferred.

Already-covered or risky surfaces still apply:

- Do not resubmit to existing A2A/Hermes/awesome-list placements in `outreach/distribution_log.csv`.
- Avoid `modelcontextprotocol/*` and `punkpeye/*` from `unitedideas`.
- Do not submit to strict A2A directories until NHS either serves a valid Agent Card or the directory explicitly accepts the current service manifest.

## Safe Positioning

Short description:

`Not Human Search is an agent-readiness search service and MCP server. It helps agents discover websites, APIs, and tools that expose public machine-readable surfaces such as llms.txt, OpenAPI, structured APIs, MCP, commerce metadata, robots rules, and schema.`

Directory-specific framing:

- Use `service` or `tooling` when the directory supports non-conversational services.
- Say NHS is useful for probe-before-use discovery, not for A2A task execution.
- Mention MCP and REST API support only where allowed.
- Keep commerce claims to the live Stripe Checkout/Link/SPT handoff and explicit ACP/x402 unsupported/private-preview boundaries.

## Do Not Claim

- Do not claim A2A support until a valid Agent Card path exists or a target explicitly accepts the current `agent.json` service manifest.
- Do not claim x402, ACP, MPP, autonomous task execution, conversational-agent behavior, certification, endorsement, customer demand, private demand, completed payments, revenue, paid ranking placement, preferred inclusion, or score-methodology bypass.
- Do not imply any profiled domain or badge-heavy domain is a customer, endorsement, paid lead, or private demand signal.

## Next Gated Action

Prepare a channel-operator packet after the product worker resolves Agent Card compatibility. The operator must:

1. Verify `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/mcp.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, and JSON-RPC `tools/list`.
2. Validate the card shape against the selected directory's schema or submission docs.
3. Check `outreach/distribution_log.csv`, `marketing/social-post-ledger.json` if applicable, and sync-state public-action locks.
4. Verify active account identity before any public edit, form submission, or post.
5. Record the submitted URL or exact blocker.
