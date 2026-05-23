# Skill Marketplace Source-Readiness Brief

Run: 2026-05-23 business-marketer-not-human-search

## Boundary

No outreach, public post, account creation, browser action, Computer Use, product-code edit, deploy, full recrawl, checkout completion, raw customer readout, or QLimit/global-queue write was performed.

This is a sanitized scout artifact for a later gated operator.

## Signal

Aggregate MCP analytics for the last 7 days showed skill-marketplace and agent-commerce-adjacent searches:

- `AI Skill Store publish skill`
- `agent marketplace usdc`
- `awesome-hermes-agent GitHub`
- `Nous Research Hermes Agent AI assistant`
- `hermes`

The useful segment is not another generic agent-builder brief. It is narrower: owners of agent skill stores, skill registries, MCP skill servers, and agent marketplaces need public machine-readable publication contracts before agents can discover, compare, or verify what is actually listed.

## Evidence Snapshot

Public NHS surfaces refreshed during this run:

- `/api/v1/stats`: `total_sites=4174`, `avg_score=35`, `top_category=developer`.
- `/api/v1/categories`: `developer=1230`, `ai-tools=903`, `ecommerce=146`, `security=113`, `other=779`.
- `/score`, `/monitor`, `/report`, `/newest`, `/top`, `/api/v1`, `/api/v1/catalog`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/llms.txt`, `/openapi.yaml`, and `/feed.xml`: HTTP 200.
- `/.well-known/agent-card.json`: HTTP 404, so strict A2A/Agent Card claims remain gated.

Aggregate admin evidence, sanitized:

- MCP methods: `tools/list=170728`, `initialize=27906`, `tools/call=244`.
- MCP tool calls: `search_agents=112`, `get_site_details=41`, `check_url=40`, `get_stats=18`, `submit_site=7`, `list_categories=6`, `find_mcp_servers=6`, `get_top_sites=6`, `recent_additions=6`, `verify_mcp=2`.
- Traffic, 168 hours: `/=3337`, `/badge/xquik.com.svg=2541`, `/.well-known/commerce.json=1414`, `/site/xquik.com=950`, `/.well-known/ai-plugin.json=656`, `/llms.txt=441`, `/openapi.yaml=403`, `/api/v1/catalog=316`, `/api/v1/checkout=275`, `/api/v1/quote=275`, `/api/v1/search=195`, `/api/v1/submit=143`, `/about=98`, `/top=96`.

## Public Example Pool

Use these only as public readiness examples or owner-channel targets. Do not frame them as customers, endorsements, paid leads, private demand, completed purchases, revenue, or proof of market share.

Developer examples:

- `agentprobe.fly.dev` score 100, Foundry-owned dogfood; label before use.
- `xquik.com` score 100, high-score owner route: monitor/report/badge proof.
- `mcp.depscope.dev` score 100, high-score owner route: monitor/report/badge proof.
- `deadends.dev` score 100, high-score owner route: monitor/report/badge proof.
- `agentdomainsearch.com` score 100, high-score owner route: monitor/report/badge proof.
- `agentndx.ai` score 100, high-score owner route: monitor/report/badge proof.

AI-tools examples:

- `8bitconcepts.com`, `bringyour.ai`, and `nothumansearch.ai` are Foundry-owned dogfood; label before use.
- `amalgix.io`, `claudereviews.com`, `chainray.online`, `memestack.ai`, and `sincetmw.ai` are high-score public examples; route to monitor/report/badge proof, not remediation.

Ecommerce/marketplace examples:

- `skillboss.co` score 100, agent-wallet/tool marketplace adjacent example.
- `packrift.com` score 80, multi-host product/API owner example; route to score-band-specific monitor or missing-surface checklist.
- `businesshotels.com` score 75 and `store.farcomindustrial.com` score 75, catalog/booking examples; do not claim fulfillment, inventory, or price accuracy.

## Owner-Channel Angle

Agent skill and marketplace owners can use NHS as a readiness check before agents depend on their listings:

- Can an agent find the publication contract without scraping a human dashboard?
- Are skill capabilities, install commands, auth boundaries, prices, payouts, contact/refund policies, and unsupported payment rails explicit?
- Is there an OpenAPI, MCP, structured API, `llms.txt`, plugin manifest, robots policy, or schema surface that can be monitored?
- Can high-score owners register a free monitor so a deploy does not erase the signals agents rely on?

Safe short copy:

`Agents are starting to search for skill stores and agent marketplaces, but the owner-side surface is still uneven. Not Human Search checks whether the public contract is probeable: MCP, OpenAPI, structured APIs, llms.txt, plugin metadata, robots policy, schema, and commerce/catalog metadata. The score is not paid placement or payment-rail certification. It is a checklist for what an agent can inspect before trusting the listing.`

## Gated Next Step

Prepare exactly one owner-channel touch, post, or product-handoff test for agent skill stores, MCP skill registries, agent marketplaces, agent wallets, or agent-commerce directory owners.

Before external use, refresh:

- `/api/v1/stats`
- `/api/v1/categories`
- `/api/v1/top?category=developer&limit=12`
- `/api/v1/top?category=ai-tools&limit=12`
- `/api/v1/top?category=ecommerce&limit=12`
- `/score`
- `/monitor`
- `/report`
- representative `/site/{host}` pages
- representative high-score and partial-score `/fix/{host}` routes
- `/mcp`
- `/.well-known/mcp.json`
- `/.well-known/agent.json`
- `/.well-known/agent-card.json`
- `/.well-known/commerce.json`
- `/.well-known/ai-plugin.json`
- `/api/v1`
- `/api/v1/catalog`
- `/api/v1/quote`
- `/api/v1/checkout`
- `/llms.txt`
- `/openapi.yaml`
- aggregate `/api/v1/admin/mcp?days=7`
- aggregate `/api/v1/admin/traffic?hours=168`

Guardrails:

- Verify active Foundry/Owl-owned account identity before public use.
- Check `marketing/social-post-ledger.json` if present, `outreach/distribution_log.csv`, and sync-state public-action locks.
- Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.
- Do not claim listed domains are customers, partners, endorsements, paid leads, monitor registrations, badge-install consent, private demand, completed payments, revenue, payment reliability, payout availability, USDC/x402/ACP/MPP support for NHS, seller certification, skill quality, agent earnings potential, task availability, marketplace safety, security/privacy compliance, crawler compliance, SEO lift, uptime proof, A2A support while `/.well-known/agent-card.json` is 404, paid ranking placement, preferred inclusion, or score-methodology bypass.
- Do not publish raw user-agent strings, private query logs, raw checkout URLs, payment identifiers, buyer emails, private monitor rows, private score-fix rows, or private customer identifiers.
