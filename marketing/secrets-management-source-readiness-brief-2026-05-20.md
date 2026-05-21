# Secrets Management Source-Readiness Brief - 2026-05-20

Automation: `business-marketer-not-human-search`
Scope: business-local marketing scout artifact only. No outreach, public post, directory submission, account creation, browser action, product-code edit, deploy, or crawl was performed.

## Evidence

- Public stats: `total_sites=4180`, `avg_score=35`, `top_category=developer`.
- Public categories: `developer=1234`, `ai-tools=904`, `other=769`, `data=400`, `finance=196`, `productivity=173`, `ecommerce=149`, `communication=120`, `security=115`.
- Aggregate traffic, 168 hours:
  - `/=3387`
  - `/badge/xquik.com.svg=2100`
  - `/.well-known/commerce.json=1573`
  - `/.well-known/ai-plugin.json=723`
  - `/site/xquik.com=702`
  - `/llms.txt=470`
  - `/openapi.yaml=436`
  - `/api/v1/catalog=339`
  - `/robots.txt=312`
  - `/api/v1/quote=309`
  - `/api/v1/checkout=309`
  - `/api/v1/search=171`
  - `/api/v1/submit=154`
  - `/api/v1=95`
  - `/.well-known/mcp.json=93`
  - `/guide=88`
  - `/score=81`
  - `/newest=71`
  - `/api/v1/check=61`
- Aggregate referrers, 168 hours:
  - `https://nothumansearch.ai/=1995`
  - `https://google.com=542`
  - canonical-domain and alternate-origin referrers remain visible, including `.com`, `www`, `http`, and `nothumansearch.fly.dev` paths.
- Aggregate MCP analytics, 7 days:
  - `tools/list=161657`
  - `initialize=22376`
  - `tools/call=263`
  - top tool calls: `search_agents=165`, `get_site_details=35`, `check_url=16`, `get_stats=12`, `find_mcp_servers=10`, `verify_mcp=8`, `get_top_sites=8`, `recent_additions=8`, `submit_site=4`, `list_categories=4`
  - visible aggregate client families include Cherry Studio, Claude Code, `MCP-Catalog-Bot`, `MCPScoringEngine`, `mcp-verify`, and `AgentFinderBot`.
- Aggregate MCP query themes included `secrets management`, developer security, source-code/agent-skill lookups, and infrastructure/toolchain queries.
- Public top-list checks:
  - Security examples: `feedoracle.io=100`, `ansvar.eu=100`, `agent-module.dev=95`, `tickerr.ai=85`, `rnwy.com=80`, `easysend.co=80`, `qnsp.cuilabs.io=70`, `hefestoai.narapallc.com=70`.
  - Developer examples: `agentprobe.fly.dev=100` (Foundry-owned dogfood), `xquik.com=100`, `mcp.depscope.dev=100`, `deadends.dev=100`, `agentdomainsearch.com=100`, `blackveilsecurity.com=100`, `agentndx.ai=100`, `entia.systems=100`.
- Public profile checks for well-known secrets/toolchain owners:
  - `1password.com` returned a public profile with `Agentic Readiness 15/100`.
  - `bitwarden.com` returned a public profile with `Agentic Readiness 45/100`.
  - `infisical.com` returned a public profile with `Agentic Readiness 0/100`.
  - `doppler.com` returned HTTP 404 for the public NHS profile at the time of this run.
  - `blackveilsecurity.com` returned a public profile with `Agentic Readiness 100/100`.
- Score-fix handoff checks:
  - High-score `blackveilsecurity.com` returned the high-score handoff: `already meets the NHS score target`.
  - Lower-score `1password.com` and `bitwarden.com` returned HTTP 200 on `/fix/{host}` but should still route through a fresh `/score` check before any remediation message.
  - `infisical.com` returned HTTP 404 on `/fix/{host}` in this run; use `/score` first if this owner segment is activated.
- Discovery surface checks:
  - `/.well-known/mcp.json`, `/.well-known/agent.json`, `/api/v1/catalog`, `/api/v1`, `/openapi.yaml`, `/llms.txt`, `/monitor`, and `/score` return HTTP 200.
  - `/.well-known/agent-card.json` returns HTTP 404, so strict A2A Agent Card claims remain blocked.

## Segment

This segment is for secrets-management, password-manager, developer-security, and secure-toolchain owners.

The useful NHS framing is not "these tools are secure" or "these are the best secrets managers." It is: agents and developer workflows increasingly need machine-readable source contracts before touching authentication, secrets, key rotation, audit logs, CLI setup, integrations, or support docs.

Safe owner routes:

1. High-score security/toolchain owners: free monitor registration, public report sharing, and badge/report proof.
2. Partial-score secrets-management owners: `/score` first, then remediation only if the public score shows concrete missing readiness surfaces.
3. API-heavy callers: `/api/v1/catalog`, `/api/v1/quote`, and `/api/v1/checkout` for API-key plans.
4. MCP client users: streamable HTTP MCP install/search using `search_agents`, `get_site_details`, `check_url`, and `verify_mcp`.

## Draft Channel Angle

Agents that inspect secrets-management docs or developer-security tooling need a source contract before they touch auth, keys, audit logs, CLI setup, or integration instructions.

Not Human Search checks the public machine-readable surfaces around those tools: llms.txt, OpenAPI, structured API responses, MCP, plugin manifests, robots policy, and Schema.org. High-score owners can monitor drift and show proof. Lower-score owners can run a public score check before deciding whether remediation is worth doing.

Example public profiles in this segment include 1password.com, bitwarden.com, infisical.com, blackveilsecurity.com, and adjacent security/developer API surfaces from the public top lists.

## Guardrails

- Do not imply any listed domain is a customer, endorsement, paid lead, private demand, monitor registration, badge-install consent, completed payment, or revenue source.
- Do not claim password-manager quality, secret-storage safety, cryptographic correctness, security certification, compliance certification, privacy compliance, uptime, integration reliability, audit-readiness, support quality, data freshness, preferred inclusion, paid placement, or score-methodology bypass.
- Do not claim A2A support while `/.well-known/agent-card.json` is 404.
- Do not publish raw user-agent logs, private query logs, checkout URLs, payment identifiers, emails, secrets, private monitor rows, or private score-fix rows.
- Label Foundry-owned dogfood examples, such as `agentprobe.fly.dev`, if using developer top-list examples.
- Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.

## Acceptance For Public Use

Before any external post, directory edit, owner touch, or product-copy change:

1. Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=security&limit=8`, `/api/v1/top?category=developer&limit=8`, representative `/site/{host}` pages, `/score`, `/monitor`, high-score and partial-score `/fix/{host}` routes, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, aggregate MCP analytics, and aggregate traffic.
2. Verify the active Foundry/Owl-owned account identity.
3. Check `marketing/social-post-ledger.json` if present, `outreach/distribution_log.csv`, and sync-state public-action locks.
4. Keep the score-band routing intact: high-score owners to monitor/report/badge proof; partial-score owners to `/score` before remediation; API-heavy callers to API-key/catalog surfaces.
