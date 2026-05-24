# MCP Check-URL To Monitor Conversion

Date: 2026-05-24
Automation: business-marketer-not-human-search
Status: no-submit packet; public use remains gated on account identity, duplicate checks, and public-action lock.

## Fresh Signals

- Public stats: 4,177 indexed sites, average score 35, top category developer.
- Live MCP tools/list: 11 tools: search_agents, get_site_details, get_stats, submit_site, check_url, verify_mcp, register_monitor, list_categories, get_top_sites, recent_additions, find_mcp_servers.
- Aggregate MCP analytics, 7 days: tools/list 172,947; initialize 27,160; tools/call 407.
- Owner-action tools, 7 days: search_agents 173; check_url 85; get_site_details 60; get_stats 21; submit_site 21; verify_mcp 14; register_monitor 4.
- Live monitor surface: /monitor returns 200.
- Latest scheduled monitor worker evidence in tools/monitor-check.log: 2026-05-18 completed one due monitor, score stayed 100 to 100, and monitor-check finished cleanly.
- Score-band routing smoke:
  - /site/nothumansearch.ai returns score 100/100; /fix/nothumansearch.ai returns the already-meets-target handoff.
  - /site/manifest.ly returns score 65/100; /fix/manifest.ly returns the paid remediation intake.
- Discovery and commerce surfaces checked live: /score, /monitor, /report, /newest, /top, /mcp-servers, /openapi-apis, /llms-txt-sites, /api/v1, /api/v1/catalog, /.well-known/mcp.json, /.well-known/agent.json, /.well-known/commerce.json, /.well-known/ai-plugin.json, /llms.txt, /openapi.yaml, and /feed.xml return 200. /.well-known/agent-card.json returns 404.

## Segment

MCP clients are already using NHS as a readiness checker. The gap is not awareness of the check_url tool; it is the next step after a check succeeds or exposes drift. This packet is for one owner-channel or MCP-client onboarding test that turns a readiness check into a monitor registration, not a broader directory submission.

Recommended angle:

> If an agent checks a site once, it should also know when that readiness changes. NHS exposes both check_url and register_monitor over MCP, plus the same flow on /score and /monitor for site owners.

Use this only where the audience already understands MCP tools or agent-readable websites: MCP client users, developer-tooling communities, API owner channels, or site-owner docs/newsletters.

## Draft Copy

Short post:

> Agents can check whether a site is ready for machine use, but a one-time check goes stale quickly.
>
> Not Human Search exposes both pieces over MCP:
>
> 1. check_url for a live readiness check.
> 2. register_monitor for weekly drift alerts when signals disappear or scores drop.
>
> The same owner flow is available at /score and /monitor.

Owner-channel note:

> Your site already has a public agent-readiness profile. The useful next step is monitoring drift, not buying placement.
>
> A high score can be watched for regressions. A partial score can be checked against the missing machine-readable surfaces first, then remediated if the owner wants help.

## Routing Rules

- MCP client users: install/search examples first, then check_url, then register_monitor.
- High-score owners: route to free monitor, report, and badge/profile proof.
- Partial-score owners: route to /score and a missing-surface checklist before paid score-fix remediation.
- API-heavy callers: route to /api/v1/catalog and API-key plans only when the docs remain useful.
- A2A or Agent Card claims stay blocked while /.well-known/agent-card.json returns 404.

## Guardrails

- Do not imply monitor registrations prove customers, paid leads, private demand, revenue, or completed payments.
- Do not imply profiled domains are customers, endorsements, badge-install consent, or monitor users.
- Do not claim paid placement, preferred inclusion, score-methodology bypass, A2A support, x402/ACP/MPP support, uptime, security certification, legal compliance, or crawler compliance.
- Do not publish raw user-agent strings, private query logs, emails, payment identifiers, raw checkout URLs, private monitor rows, private score-fix rows, or private customer identifiers.

## Acceptance Before Public Use

1. Refresh /api/v1/stats, /mcp tools/list, /score, /monitor, /report, /llms.txt, /.well-known/mcp.json, /.well-known/agent.json, /.well-known/agent-card.json, /.well-known/commerce.json, /.well-known/ai-plugin.json, /api/v1, /api/v1/catalog, /openapi.yaml, and /feed.xml.
2. Refresh aggregate /api/v1/admin/mcp?days=7 and /api/v1/admin/traffic?hours=168.
3. Recheck a high-score /fix/{host} route and a partial-score /fix/{host} route.
4. Verify active Foundry/Owl-owned channel identity.
5. Check marketing/social-post-ledger.json if present, outreach/distribution_log.csv, and sync-state public-action locks.
6. Avoid modelcontextprotocol/* and punkpeye/* surfaces from unitedideas.
