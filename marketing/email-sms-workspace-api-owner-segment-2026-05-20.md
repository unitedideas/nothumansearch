# Email, SMS, and Workspace API Owner Segment

Date: 2026-05-20
Agent: business-marketer-not-human-search
Purpose: gated owner-channel packet for communication API owners whose public agent-readiness profiles can route to monitor, report, badge proof, or score-fix only after a fresh score check.

## Live Evidence

- Public stats: 4,180 total sites, average score 35, top category developer.
- Public communication top list:
  - mail.misar.io: score 100, communication.
  - resend.com: score 75, communication.
  - secondsim.co.uk: score 70, communication.
  - postalform.com: score 65, communication.
  - slack.com: score 60, communication.
  - api.slack.com: score 60, communication.
  - pantrypersona.com: score 60, communication.
  - kweenkl.com: score 55, communication.
- Aggregate admin traffic, 168h:
  - `/`: 3,483
  - `/badge/xquik.com.svg`: 2,093
  - `/.well-known/commerce.json`: 1,578
  - `/.well-known/ai-plugin.json`: 719
  - `/site/xquik.com`: 703
  - `/llms.txt`: 467
  - `/openapi.yaml`: 434
  - `/api/v1/catalog`: 340
  - `/api/v1/checkout`: 310
  - `/api/v1/quote`: 310
  - `/robots.txt`: 307
  - `/api/v1/search`: 171
- Aggregate MCP analytics, 7d:
  - `tools/list`: 160,528 calls.
  - `tools/call`: 281 calls.
  - Tool calls: `search_agents` 175, `get_site_details` 36, `check_url` 17, `get_stats` 14, `verify_mcp` 8, `get_top_sites` 8, `recent_additions` 8, `find_mcp_servers` 7, `list_categories` 4, `submit_site` 4.
  - Client/directory signals include Cherry Studio, Claude Code, MCP-Catalog-Bot, MCPScoringEngine, mcp-verify, and AgentFinderBot.
- Live public discovery surfaces checked:
  - `/`, `/score`, `/monitor`, `/fix/nothumansearch.ai`, `/api/v1`, `/openapi.yaml`, `/llms.txt`, and `/.well-known/agent.json` returned HTTP 200.
  - `/.well-known/agent-card.json` returned HTTP 404.
  - `/.well-known/mcp.json` advertises 11 tools.
  - `/api/v1/catalog` lists score-fix plus Starter, Pro, and Scale API plans.

## Segment

This segment is for email, SMS, forms, and workspace/API owners where agent-readiness is operationally relevant:

- Transactional email providers and API-first deliverability tools.
- SMS/phone verification services.
- Form-to-email or inbox workflow tools.
- Workspace communication platforms and developer-facing app APIs.

The useful NHS framing is not "best communication tools." It is: agents need probeable contracts before relying on communication surfaces for sends, notifications, and workspace actions. Owners with strong scores can use monitor/report/badge proof. Owners below 90 can start with `/score` and route to remediation only when the public score shows missing agent-readiness files.

## Candidate Owner Routes

- High-score owners: monitor/report/badge proof.
- 70-89 owners: public score check first, then owner-side gap review.
- Below 70 owners: score check, then score-fix only if the missing surfaces are public and fixable.
- API-heavy users: API plan handoff through `/api/v1/catalog`, `/api/v1/quote`, and `/api/v1/checkout`.
- MCP client users: streamable HTTP MCP onboarding with `search_agents`, `check_url`, `verify_mcp`, and `get_top_sites`.

## Draft Angle

Agents that send email, SMS, form submissions, or workspace messages need a source contract they can inspect before acting.

Not Human Search profiles communication APIs by machine-readable readiness: llms.txt, OpenAPI, structured API responses, MCP, plugin manifests, robots policy, and Schema.org. High-score owners can monitor drift and show proof. Lower-score owners can see what is missing before paying for remediation.

Current public communication examples include mail.misar.io, resend.com, secondsim.co.uk, postalform.com, slack.com, and api.slack.com.

## Guardrails

- Do not imply any listed domain is a customer, endorsement, paid lead, private demand, monitor registration, badge-install consent, completed payment, or revenue source.
- Do not claim deliverability quality, SMS reliability, workspace integration reliability, privacy compliance, security certification, uptime, data freshness, preferred inclusion, paid placement, or score-methodology bypass.
- Do not claim A2A support while `/.well-known/agent-card.json` is 404.
- Do not publish raw user-agent logs, private query logs, checkout URLs, payment identifiers, emails, or private monitor/score-fix rows.
- Avoid modelcontextprotocol/* and punkpeye/* surfaces from `unitedideas`.

## Acceptance For Public Use

Before any external post, directory edit, or owner touch:

1. Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=communication&limit=8`, representative `/site/{host}` pages, `/score`, `/monitor`, high-score and partial-score `/fix/{host}` routes, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, aggregate MCP analytics, and aggregate traffic.
2. Verify the active Foundry/Owl-owned account identity.
3. Check `marketing/social-post-ledger.json` if present, `outreach/distribution_log.csv`, and sync-state public-action locks.
4. Keep the score-band routing intact: high-score owners to monitor/report/badge proof; partial-score owners to `/score` before remediation; API-heavy callers to API-key/catalog surfaces.
