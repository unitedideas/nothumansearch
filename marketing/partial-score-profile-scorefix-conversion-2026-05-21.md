# Partial-Score Profile Score-Fix Conversion Scout - 2026-05-21

Automation: `business-marketer-not-human-search`
Scope: business-local marketing scout artifact only. No outreach, public post, directory submission, account creation, browser action, product-code edit, deploy, checkout completion, or crawl was performed.

## Evidence

- Public stats: `total_sites=4170`, `avg_score=35`, `top_category=developer`.
- Public categories: `developer=1227`, `ai-tools=897`, `other=777`, `data=402`, `finance=194`, `productivity=174`, `ecommerce=148`, `communication=118`, `security=114`, `health=58`, `jobs=27`, `education=21`, `news=12`, `spam=1`.
- Discovery surfaces checked: `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`, `/openapi.yaml`, `/score`, `/monitor`, `/report`, and `/mcp` returned HTTP `200`.
- A2A Agent Card remains blocked: `/.well-known/agent-card.json` returned HTTP `404`.
- Aggregate admin traffic, last 168 hours: `/=3391`, `/badge/xquik.com.svg=2176`, `/.well-known/commerce.json=1533`, `/site/xquik.com=712`, `/.well-known/ai-plugin.json=706`, `/llms.txt=459`, `/openapi.yaml=426`, `/api/v1/catalog=331`, `/robots.txt=308`, `/api/v1/quote=301`, `/api/v1/checkout=301`, `/api/v1/search=170`, `/api/v1/submit=151`, `/top=94`, `/score=79`, `/newest=66`, `/site/manifest.ly=63`.
- Aggregate referrers, last 168 hours: `google.com=542`, canonical/alias referrers remain material, `/score=106`, platform host `nothumansearch.fly.dev=63`, `/top=41`, `/site/chainray.online=38`, `/site/xquik.com=32`, `/mcp=30`, `/submit=28`.
- Aggregate MCP analytics, last 7 days: `tools/list=166040`, `initialize=24468`, `tools/call=273`; top called tools include `search_agents=152`, `get_site_details=38`, `get_stats=20`, `check_url=19`, `get_top_sites=10`, `find_mcp_servers=9`, `verify_mcp=8`, `recent_additions=8`, `list_categories=5`, and `submit_site=4`.
- Public profile check: `/site/manifest.ly` returned HTTP `200` with score `65/100`; the page exposes badge/profile proof and a score-fix path.
- Score-fix gate check: `/fix/manifest.ly` returned HTTP `200` with paid remediation form; `/fix/xquik.com` returned the high-score handoff with `already meets the target` and a monitor link instead of a paid form.
- Adjacent public owner examples from current top lists include productivity owners `tabai.dev=100`, `blooio.com=100`, `barevalue.com=100`, `simplepdf.com=100`, `angshumangupta.com=80`, `attio.com=80`, `berger.team=70`, `planharmony.com=70`; communication owners `resend.com=75`, `secondsim.co.uk=70`, `mail.misar.io=70`, `postalform.com=65`, `api.slack.com=60`; and `other` owners `agentgrade.com=80`, `surprise-buddy.com=65`, `twitterapi.io=65`, `infinity-folder.org=65`.

## Segment

The fresh segment is partial-score public profile traffic. `/site/manifest.ly` is now visible in aggregate top pages, and the public score is 65/100. That is different from the high-score Xquik and ChainRay profile loops: this one can route to `/score` and score-fix remediation after the public profile proves concrete missing readiness surfaces.

Use this as an owner-conversion test for workflow, checklist, SOP, productivity, communication, and agent-workflow SaaS teams.

Safe owner routes:

1. Partial-score profile visitors: confirm the public score, then offer the score-fix path only for missing readiness surfaces.
2. High-score profile or badge visitors: free monitor registration, report sharing, and badge/report proof.
3. API-heavy callers: keep `/api/v1/catalog`, `/api/v1/quote`, and `/api/v1/checkout` as API-key plan handoffs, with docs useful first.
4. MCP users: keep `/mcp`, `/.well-known/mcp.json`, `search_agents`, `get_site_details`, and `check_url` as the install/search path.

## Draft Channel Angle

Agent-readable workflow tools need more than a homepage.

The useful owner path is score-band aware:

1. Check the current public score.
2. If the site is already strong, register a free monitor and use the report or badge as proof.
3. If the score is partial, fix the missing public machine-readable surfaces before asking agents to rely on it.

Not Human Search does not sell ranking placement. It checks the public surfaces agents can actually inspect: `llms.txt`, OpenAPI, structured API responses, MCP, plugin metadata, robots policy, Schema.org, and reportable score history.

## Gated Test

Prepare exactly one gated owner-channel touch, channel post, or product-handoff test for workflow, checklist, SOP, productivity, communication, or agent-workflow SaaS owners using this packet.

Before external use, refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=productivity&limit=8`, `/api/v1/top?category=communication&limit=8`, representative `/site/{host}` profiles including `manifest.ly`, high-score and partial-score `/fix/{host}` routes, `/score`, `/monitor`, `/report`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, aggregate `/api/v1/admin/mcp?days=7`, and aggregate `/api/v1/admin/traffic?hours=168`.

Verify the active Foundry/Owl-owned account identity for the selected channel, check `marketing/social-post-ledger.json` if present plus `outreach/distribution_log.csv` and sync-state public-action locks, and avoid `modelcontextprotocol/*` plus `punkpeye/*` surfaces from `unitedideas`.

## Guardrails

Do not imply Manifestly, Xquik, ChainRay, Resend, Slack, or any listed domain is a customer, endorsement, partner, paid lead, monitor registration, badge-install consent, private demand, completed payment, revenue, workflow quality proof, productivity improvement proof, deliverability proof, integration reliability, security/privacy compliance, uptime proof, seller certification, x402/ACP/MPP support for NHS, A2A support, paid ranking placement, preferred inclusion, or score-methodology bypass.
