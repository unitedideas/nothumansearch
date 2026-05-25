# Search And Scraping Source-Readiness Brief

Run: 2026-05-25.
Agent: `business-marketer-not-human-search`.
Scope: no-submit marketing scout artifact for Not Human Search.

No public action was taken. No outreach was sent. This is a business-local packet for a later gated owner-channel test.

## Live Evidence

- Public stats: `total_sites=4172`, `avg_score=35`, `top_category=developer`.
- Public category counts: `developer=1230 avg_score=34`, `data=402 avg_score=32`, `ecommerce=149 avg_score=41`, `security=113 avg_score=39`.
- Public data top-list examples: `api.headlessoracle.com=100`, `api.contrastcyber.com=100`, `dchub.cloud=95`, `daedalmap.com=90`, `api.theartofservice.com=90`, `api.agentry.com=90`, `api.socialintel.dev=90`, `blocklens.co=90`.
- Public developer top-list examples: `agentprobe.fly.dev=100`, `xquik.com=100`, `deadends.dev=100`, `agentdomainsearch.com=100`, `blackveilsecurity.com=100`, `agentndx.ai=100`, `flowzap.xyz=100`, `entia.systems=100`.
- Live discovery surfaces checked: `/score=200`, `/monitor=200`, `/report=200`, `/feed.xml=200`, `/api/v1=200`, `/api/v1/catalog=200`, `/llms.txt=200`, `/openapi.yaml=200`, `/.well-known/mcp.json=200`, `/.well-known/agent.json=200`, `/.well-known/commerce.json=200`, `/.well-known/ai-plugin.json=200`.
- Compatibility blocker still present: `/.well-known/agent-card.json=404`, so A2A/Agent Card claims remain blocked.
- Live MCP `tools/list`: 11 tools: `search_agents`, `get_site_details`, `get_stats`, `submit_site`, `check_url`, `verify_mcp`, `register_monitor`, `list_categories`, `get_top_sites`, `recent_additions`, `find_mcp_servers`.
- Aggregate MCP, 7d: `tools/list=175253`, `initialize=25009`, `tools/call=377`; top called tools include `search_agents=147`, `check_url=86`, `get_site_details=58`, `get_stats=25`, `submit_site=20`, `verify_mcp=13`.
- Aggregate query themes included web search, ecommerce product search and scraping, outdoor product review discovery, model/API lookups, finance data APIs, and local events.
- Aggregate traffic, 168h: `/=3406`, `/badge/xquik.com.svg=2661`, `/.well-known/commerce.json=1352`, `/site/xquik.com=1107`, `/.well-known/ai-plugin.json=616`, `/llms.txt=446`, `/openapi.yaml=382`, `/api/v1/catalog=322`, `/api/v1/quote=258`, `/api/v1/checkout=258`.
- `tools/monitor-check.log`: latest scheduled proof on 2026-05-25 completed 5 due monitors. Two zero-score first checks were quarantined, two low/partial first checks were recorded, and one high-score monitor remained stable.
- Score-fix routing check: `/fix/nothumansearch.ai` routes a high-score domain to monitor/report language; `/fix/manifest.ly` shows partial-score paid intake.

## Segment

Search, scraping, SERP, product-discovery, and crawler API owners need agent-readable contracts because their buyers are already trying to delegate discovery and extraction tasks to tools.

NHS can support a narrow owner-channel test around:

- public `llms.txt`, OpenAPI, API, MCP, catalog, and contact/support metadata;
- score-band routing from `/score` into free monitor/report/badge proof for high-score profiles;
- missing-surface checklists before score-fix remediation for partial-score owners;
- API-key/catalog handoff only where the live docs remain useful;
- monitor registration for sites whose search or scraping surfaces drift often.

This should be framed as source-readiness for agent workflows, not as proof of result quality, scraping reliability, crawler compliance, current index freshness, or private demand.

## Public Examples

Use these as public readiness examples only. They are not customers, endorsements, paid leads, or demand proof.

| Domain | Score | Scout use |
|---|---:|---|
| `api.headlessoracle.com` | 100 | Data/API example with complete public agent-readiness signals. |
| `api.socialintel.dev` | 90 | Social/web intelligence API example with strong readiness signals. |
| `blocklens.co` | 90 | Data-product example with strong readiness signals. |
| `agentdomainsearch.com` | 100 | Developer/search-tool example with complete readiness signals. |
| `agentndx.ai` | 100 | Agent directory/search example with complete readiness signals. |
| `xquik.com` | 100 | High-score public profile with material badge/profile traffic; use only as badge-loop routing evidence. |

## Gated Channel Test

Draft one short owner-channel post or direct-owner packet:

> Agents are starting to call search, scraping, and product-discovery tools directly.
>
> The useful owner-side question is not whether the page is readable by a human. It is whether an agent can find the public source contract, inspect the API shape, verify the MCP/tool surface, and monitor whether those paths drift after a deploy.
>
> Free check: `https://nothumansearch.ai/score`
> Free monitor: `https://nothumansearch.ai/monitor`

Before external use, refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=data&limit=12`, `/api/v1/top?category=developer&limit=12`, `/api/v1/top?category=ecommerce&limit=12`, `/score`, `/monitor`, `/report`, representative `/site/{host}` pages, high-score and partial-score `/fix/{host}` routes, `/mcp` `tools/list`, all machine-readable manifests, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, aggregate admin MCP/traffic data, and latest monitor worker proof.

## Boundaries

Do not claim search-result quality, scraping success, crawler compliance, anti-bot bypass, product-review truth, product quality, inventory accuracy, price freshness, data freshness, index completeness, customer demand, private demand, paid leads, completed payments, revenue, endorsement, paid placement, preferred inclusion, A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/SPT/MPP support for NHS, or score-methodology bypass.

Do not publish raw user-agent strings, private query logs, raw checkout URLs, payment identifiers, buyer emails, private monitor rows, private score-fix rows, or private customer identifiers.
