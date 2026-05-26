# MCP check-url to monitor conversion refresh - 2026-05-26

Automation: `business-marketer-not-human-search`

Scope: no public action, no outreach, no browser, no deploy, no full recrawl, no checkout completion, and no QLimit/global-queue write. This is a sanitized scout artifact for a later gated operator.

## Evidence

- Public stats: 4,172 indexed sites, average score 35, top category `developer`.
- Public categories: `developer=1230`, `ai-tools=904`, `other=774`, `data=402`, `finance=192`, `productivity=171`, `ecommerce=149`, `communication=118`, `security=113`, `health=59`, `jobs=26`, `education=21`, `news=12`, and `spam=1`.
- Live public surfaces returned 200: `/api/v1`, `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, `/mcp`, `/monitor`, `/score`, and `/fix/nothumansearch.ai`.
- `/.well-known/agent-card.json` is still not part of the live public proof set for this run, so do not claim A2A or Agent Card support.
- Aggregate MCP analytics, 7 days: `tools/list=175,398`, `initialize=25,349`, and `tools/call=379`.
- Aggregate MCP tool calls, 7 days: `search_agents=148`, `check_url=86`, `get_site_details=58`, `get_stats=26`, `submit_site=20`, `verify_mcp=13`, `list_categories=7`, `find_mcp_servers=7`, `recent_additions=6`, `register_monitor=4`, and `get_top_sites=4`.
- Aggregate traffic, 168 hours: `/=3,438`, `/badge/xquik.com.svg=2,645`, `/.well-known/commerce.json=1,345`, `/site/xquik.com=1,107`, `/.well-known/ai-plugin.json=607`, `/llms.txt=442`, `/openapi.yaml=376`, `/api/v1/catalog=323`, `/robots.txt=291`, `/badge/aidevboard.com.svg=268`, `/api/v1/checkout=256`, `/api/v1/quote=256`, `/badge/8bitconcepts.com.svg=249`, `/api/v1/search=220`, `/favicon.ico=194`, `/api/v1/submit=146`, `/digest=128`, and `/about=91`.
- Aggregate referrers, 168 hours: `nothumansearch.ai root=1974`, `google.com=568`, `.com/www/http aliases remain material`, and `/score=84`.
- Latest local monitor worker proof, 2026-05-25: completed normally with 5 due monitors. Aggregate outcome was two first-check zero-score quarantines, two first-check partial or low-score checks, and one stable high-score check.
- High-score `/fix/nothumansearch.ai` returned the already-meets-target handoff, not a paid remediation intake.

## Read

The MCP client funnel still has a visible gap:

- `check_url` remains the second-most-used MCP tool.
- `register_monitor` is present but much smaller.
- `/monitor` is live, but the latest monitor worker produced both stable and quarantined outcomes.

That means this should be framed as a gated owner-conversion test, not broad monitor-growth copy. The correct path is: MCP client checks a URL, high-score owners get monitor/report/badge proof, partial-score owners get `/score` plus a missing-surface checklist, and zero-score or quarantined monitor cases stay private/admin-only.

## Candidate Test

Prepare one gated product-handoff or channel test:

1. MCP client path: `search_agents` or `check_url` result -> `register_monitor` prompt for score drift.
2. Owner path: `/score` result -> high-score monitor/report/badge proof or partial-score missing-surface checklist.
3. Sales path: partial-score remediation only after a fresh public score shows missing public agent-readiness surfaces.
4. Admin path: quarantined monitor rows remain private until a bounded review records keep-quarantined, approve-monitoring, or remediation-offered.

## Boundaries

Do not imply `check_url` calls, monitor registrations, score checks, profiled domains, badge routes, API/catalog traffic, manifest traffic, or referrer traffic prove customers, endorsements, partners, paid leads, private demand, completed payments, revenue, crawler compliance, legal permission, SEO lift, uptime proof, A2A support, x402/ACP/SPT/MPP support for NHS, paid placement, preferred inclusion, or score-methodology bypass.

Do not publish raw user-agent strings, raw MCP queries, private monitor rows, raw checkout URLs, payment identifiers, buyer emails, private score-fix rows, or private customer identifiers.

## Next Gated Action

Use this packet for exactly one gated MCP-client onboarding, owner-channel, or product-handoff test after refreshing public stats, discovery surfaces, aggregate MCP analytics, aggregate traffic, latest monitor worker proof, and high-score plus partial-score `/fix/{host}` behavior.
