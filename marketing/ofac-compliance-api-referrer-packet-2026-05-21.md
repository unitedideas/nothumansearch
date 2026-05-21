# OFAC Compliance API Referrer Packet - 2026-05-21

Automation: `business-marketer-not-human-search`
Scope: business-local marketing scout artifact only. No outreach, public post, directory submission, account creation, browser action, product-code edit, deploy, checkout completion, or crawl was performed.

## Evidence

- Public stats: `total_sites=4182`, `avg_score=35`, `top_category=developer`.
- Public categories: `developer=1235`, `ai-tools=904`, `other=770`, `data=400`, `finance=196`, `productivity=173`, `ecommerce=149`, `communication=120`, `security=115`, `health=59`, `jobs=27`, `education=21`, `news=12`, `spam=1`.
- Aggregate traffic, 168 hours: `/=3387`, `/badge/xquik.com.svg=2123`, `/.well-known/commerce.json=1563`, `/.well-known/ai-plugin.json=719`, `/site/xquik.com=703`, `/llms.txt=466`, `/openapi.yaml=433`, `/api/v1/catalog=337`, `/robots.txt=313`, `/api/v1/quote=307`, `/api/v1/checkout=307`, `/api/v1/search=169`, `/api/v1/submit=154`, `/api/v1=95`, `/.well-known/mcp.json=93`, `/guide=87`, `/score=82`, `/api/v1/check=61`.
- Aggregate referrers, 168 hours: `https://nothumansearch.ai/=2016`, `https://google.com=542`, `https://nothumansearch.com/=149`, `https://nothumansearch.ai/score=122`, `http://nothumansearch.fly.dev=61`, `https://nothumansearch.ai/top=44`, `https://nothumansearch.ai/site/chainray.online=37`, `https://nothumansearch.ai/mcp-servers=35`, `https://nothumansearch.ai/site/xquik.com=32`, `https://nothumansearch.ai/mcp=31`, `https://nothumansearch.ai/submit=29`, `https://aurelianflo.com/=25`.
- Public check on the external referrer: `https://aurelianflo.com/` returned HTTP 200 with title `AurelianFlo - OFAC Wallet Screening and Compliance APIs`.
- Public finance examples from `/api/v1/top?category=finance&limit=8`: `terminalfeed.io=100`, `chartlibrary.io=100`, `prereason.com=100`, `devdrops.run=95`, `razorpay.com=90`, `ticksurfers.com=90`, `lendtrain.com=85`, `debridge.com=80`.
- Discovery surfaces checked: `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1/catalog`, `/api/v1`, `/openapi.yaml`, `/llms.txt`, `/score`, `/monitor`, `/report`, and `/mcp` returned HTTP 200. `/.well-known/agent-card.json` returned HTTP 404, so strict A2A Agent Card claims remain blocked.
- Aggregate MCP analytics, 7 days: `tools/list=163911`, `initialize=23421`, `tools/call=279`; top called tools include `search_agents=164`, `get_site_details=38`, `check_url=20`, `get_stats=15`, `find_mcp_servers=9`, `verify_mcp=8`, `get_top_sites=8`, and `recent_additions=8`.

## Segment

The fresh scout signal is a public external referrer from an OFAC wallet-screening and compliance API site. Treat this as a referral-partner and owner-conversion signal, not as endorsement or private demand.

The useful segment is compliance and fintech API owners whose products are sensitive enough that agents need source contracts before they touch wallet screening, sanctions checks, KYB/KYC workflows, payment risk, identity, or financial data.

Safe owner routes:

1. High-score finance/compliance API owners: free monitor registration, public report sharing, and badge/report proof.
2. Partial-score finance/compliance API owners: `/score` first, then remediation only if a current public score shows concrete missing readiness surfaces.
3. API-heavy callers: `/api/v1/catalog`, `/api/v1/quote`, and `/api/v1/checkout` for API-key plans.
4. MCP users: `/mcp`, `/.well-known/mcp.json`, and `search_agents` / `get_site_details` / `check_url` flows.

## Draft Channel Angle

Agents that inspect compliance APIs need a stable source contract before touching sanctions, wallet screening, payment risk, or identity workflows.

Not Human Search checks the public machine-readable surfaces around those APIs: `llms.txt`, OpenAPI, structured API responses, MCP, plugin manifests, robots policy, and Schema.org. High-score owners can monitor drift and show proof. Lower-score owners can run a public score check before deciding whether remediation is worth doing.

## Gated Test

Prepare exactly one gated referral-partner, compliance API owner, fintech API owner, crypto compliance, sanctions-screening, or API-risk channel touch using this packet.

Before external use, refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=finance&limit=8`, `/api/v1/top?category=security&limit=8`, representative `/site/{host}` profiles, `/score`, `/monitor`, high-score and partial-score `/fix/{host}` routes, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, aggregate `/api/v1/admin/mcp?days=7`, and aggregate `/api/v1/admin/traffic?hours=168`.

Verify the active Foundry/Owl-owned account identity for the selected channel, check `marketing/social-post-ledger.json` if present plus `outreach/distribution_log.csv` and sync-state public-action locks, and avoid `modelcontextprotocol/*` plus `punkpeye/*` surfaces from `unitedideas`.

## Guardrails

Do not imply AurelianFlo, Xquik, ChainRay, Razorpay, Debridge, or any listed domain is a customer, endorsement, partner, paid lead, monitor registration, badge-install consent, private demand, completed payment, revenue, wallet-screening accuracy proof, sanctions-compliance certification, KYC/KYB compliance, security certification, uptime proof, pricing accuracy, seller certification, x402/ACP/MPP support for NHS, A2A support, paid ranking placement, preferred inclusion, or score-methodology bypass.
