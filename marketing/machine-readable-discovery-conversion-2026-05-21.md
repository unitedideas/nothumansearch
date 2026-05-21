# Machine-Readable Discovery Conversion Scout - 2026-05-21

Automation: `business-marketer-not-human-search`
Scope: business-local marketing scout artifact only. No outreach, public post, directory submission, account creation, browser action, product-code edit, deploy, checkout completion, or crawl was performed.

## Evidence

- Public stats: `total_sites=4174`, `avg_score=35`, `top_category=developer`.
- Public categories: developer `1228`, ai-tools `901`, other `771`, data `403`, finance `194`, productivity `174`, ecommerce `149`, communication `120`, security `114`, health `59`, jobs `27`, education `21`, news `12`, spam `1`.
- Discovery surface smokes: `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`, `/openapi.yaml`, `/monitor`, `/score`, and `/report` returned HTTP `200`.
- A2A Agent Card remains blocked: `/.well-known/agent-card.json` returned HTTP `404`.
- Aggregate admin traffic, last 168 hours: `/=3398`, `/badge/xquik.com.svg=2158`, `/.well-known/commerce.json=1538`, `/.well-known/ai-plugin.json=707`, `/site/xquik.com=704`, `/llms.txt=459`, `/openapi.yaml=426`, `/api/v1/catalog=332`, `/robots.txt=307`, `/api/v1/checkout=302`, `/api/v1/quote=302`, `/api/v1/search=170`.
- Aggregate referrers, last 168 hours: `google.com=542`, `nothumansearch.com` and `www`/HTTP aliases remain material, and `/score` referrer traffic remains visible.
- Distribution history already includes MCP/awesome-list submissions, public gists, newsletter/editorial pitches, and the open InftyAI Awesome-LLMOps project request. This packet is not a resubmission prompt.

## Segment

The useful segment is machine-readable discovery traffic that is already landing on manifests, API docs, catalog, quote, checkout, robots, and profile/badge routes.

This should be treated as conversion routing evidence, not customer-demand proof. The next public-safe test is a channel packet for developer-tool and API-owner audiences:

1. Machine clients discover NHS through `llms.txt`, OpenAPI, MCP JSON, plugin manifest, commerce JSON, catalog, and API root.
2. Site owners who arrive through discovery pages should be routed to `/score` and free `/monitor`.
3. High-score profile or badge visitors should be routed to report sharing, badge proof, and drift monitoring.
4. Partial-score owners should run `/score` before any score-fix remediation.
5. API-heavy callers should see API-key plans only after docs remain useful.

## Draft Channel Angle

Agents do not need another directory entry. They need a source contract they can inspect before calling anything.

Not Human Search exposes its own contract the same way it scores other sites: `llms.txt`, OpenAPI, MCP, API root, plugin metadata, commerce metadata, catalog, quote, checkout, robots policy, and public score/report routes.

The owner-side path is the point:

1. Check the public score.
2. Register a free monitor when the site is already readable.
3. Use the report or badge as proof when the score is strong.
4. Fix missing machine-readable surfaces only when the current score shows a real gap.

## Gated Test

Prepare exactly one gated developer-tool, API-owner, agent-directory, or machine-readable-web channel touch from this packet.

Before external use, refresh `/api/v1/stats`, `/api/v1/categories`, `/llms.txt`, `/openapi.yaml`, `/api/v1`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/score`, `/monitor`, `/report`, representative `/site/{host}` profiles, high-score and partial-score `/fix/{host}` routes, aggregate MCP analytics, and aggregate traffic.

Verify the active Foundry/Owl-owned account identity for the selected channel, check `marketing/social-post-ledger.json` if present plus `outreach/distribution_log.csv` and sync-state public-action locks, and avoid `modelcontextprotocol/*` plus `punkpeye/*` surfaces from `unitedideas`.

## Guardrails

Do not imply manifest, API, catalog, profile, badge, Google, alias, or route traffic proves customers, endorsements, partners, paid leads, monitor registrations, badge-install consent, private demand, completed payments, revenue, crawler compliance, legal permission, SEO lift, uptime proof, A2A support, x402/ACP support, paid ranking placement, preferred inclusion, or score-methodology bypass.
