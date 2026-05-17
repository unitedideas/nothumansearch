# Workflow and Communication Owner Segment

Date: 2026-05-17
Automation: `business-marketer-not-human-search`
Status: prepared, not published

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a sanitized scout segment for a later gated owner-channel operator.

## Fresh Evidence

Public surfaces checked:

- `https://nothumansearch.ai/api/v1/stats`: 4,174 indexed sites, average score 35, top category `developer`.
- `https://nothumansearch.ai/api/v1/categories`: `productivity=172` average score 38; `communication=117` average score 38.
- `https://nothumansearch.ai/api/v1/top?category=productivity&limit=8`: public top-list source for workflow, CRM, document, and productivity examples.
- `https://nothumansearch.ai/api/v1/top?category=communication&limit=8`: public top-list source for email, messaging, notification, and collaboration examples.
- `https://nothumansearch.ai/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, `/score`, and `/monitor`: HTTP 200.
- `https://nothumansearch.ai/.well-known/agent-card.json`: HTTP 404, so strict Agent Card directory submissions remain gated.
- `/fix/simplepdf.com` and `/fix/attio.com`: HTTP 200. Keep the conversion split: high-score workflow owners should route to monitor/report/badge proof; partial-score owners should run `/score` before any remediation pitch.

Aggregate admin signals, sanitized:

- MCP analytics, last 7 days: `tools/list=139450`, `initialize=18839`, `tools/call=280`.
- Top called MCP tools: `search_agents=182`, `get_site_details=35`, `check_url=12`, `get_stats=12`, `verify_mcp=12`, `recent_additions=9`, `get_top_sites=8`, `find_mcp_servers=7`.
- Aggregate MCP query themes include workflow/design/API terms such as Roboflow computer-vision tooling, Penpot MCP, secrets management, free LLM API/tool-use, and agent jobs/tasks.
- Traffic, last 168 hours: `/.well-known/commerce.json=1578`, `/api/v1/catalog=353`, `/api/v1/quote=328`, `/api/v1/checkout=328`, `/llms.txt=479`, `/openapi.yaml=456`, `/top=124`, `/newest=88`.
- Errors last hour: 0.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, private query logs, or raw buyer data are included here.

## Public Productivity Examples

These are public readiness examples or possible owner-channel targets only. They are not customers, endorsements, paid leads, private demand, completed payments, revenue proof, or market-share proof.

| Domain | Score | Owner-channel route | Boundary |
|---|---:|---|---|
| `barevalue.com` | 100 | Monitor/report/badge proof for complete public readiness. | Do not claim workflow quality or customer demand. |
| `simplepdf.com` | 100 | Monitor/report/badge proof for a complete document-workflow surface. | Do not claim HIPAA/GDPR correctness from NHS. |
| `blooio.com` | 100 | Monitor/report/badge proof for CRM messaging readiness. | Do not claim deliverability or CRM integration quality. |
| `attio.com` | 80 | `/score` first; remediation can focus on public OpenAPI visibility if owner wants full readiness. | Do not imply partnership or private demand. |
| `catchintent.com` | 70 | `/score` first; remediation can focus on agent discovery metadata and MCP/API completeness. | Do not claim lead quality or outbound performance. |
| `epublys.com` | 70 | `/score` first; remediation can focus on ai-plugin/MCP completeness if relevant. | Do not claim document-processing quality. |

## Public Communication Examples

| Domain | Score | Owner-channel route | Boundary |
|---|---:|---|---|
| `mail.misar.io` | 100 | Monitor/report/badge proof for complete communication-tool readiness. | Do not claim deliverability, campaign performance, or endorsement. |
| `resend.com` | 75 | `/score` first; remediation can focus on ai-plugin, AI-friendly robots, and Schema.org if owner wants full public readiness. | Do not claim email deliverability or customer relationship. |
| `secondsim.co.uk` | 70 | `/score` first; remediation can focus on OpenAPI/MCP clarity if operational APIs exist. | Do not claim verification reliability. |
| `postalform.com` | 65 | `/score` first; remediation can focus on ai-plugin, robots, and MCP only if appropriate. | Do not claim postal fulfillment quality. |
| `kweenkl.com` | 55 | `/score` first; remediation can focus on OpenAPI, ai-plugin, and AI-friendly policy surfaces. | Do not claim notification reliability. |

## Segment Read

Workflow and communication tools are a practical owner-channel segment because the score gaps are concrete. Agents can only use CRM, document, email, messaging, and notification products safely when they can find public contracts, supported APIs, MCP or equivalent tool surfaces, and monitorable metadata.

Good fit:

- CRM, document, email, messaging, notification, and workflow owners that already expose some agent-readable structure.
- High-score owners that should monitor drift and show badge/report proof.
- Partial-score owners missing one or two public readiness surfaces.

Bad fit:

- Claims about workflow quality, deliverability, integration reliability, privacy compliance, or data freshness.
- Paid ranking, preferred inclusion, or score-methodology bypass language.
- Broad "NHS proves demand" copy.

## Draft Operator Copy

`Not Human Search checks whether workflow and communication products expose enough public structure for agents to inspect them: llms.txt, OpenAPI, structured APIs, MCP, AI-friendly robots rules, plugin metadata, and schema.`

`For high-scoring owners, the useful next step is monitoring and a shareable report. For partial-score owners, the score page shows the missing public readiness surfaces before any remediation offer.`

Proof links:

- `https://nothumansearch.ai/top?category=productivity`
- `https://nothumansearch.ai/top?category=communication`
- `https://nothumansearch.ai/score`
- `https://nothumansearch.ai/monitor`
- `https://nothumansearch.ai/api/v1/catalog`
- `https://nothumansearch.ai/llms.txt`

## Publication Guard

Before any external use:

1. Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=productivity&limit=8`, `/api/v1/top?category=communication&limit=8`, `/score`, `/monitor`, representative `/site/{host}` profiles, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/agent-card.json`, `/.well-known/commerce.json`, `/api/v1/catalog`, `/api/v1/quote`, `/api/v1/checkout`, `/llms.txt`, `/openapi.yaml`, and aggregate `/api/v1/admin/mcp?days=7`.
2. Verify active account identity for the selected Foundry/Owl-owned channel.
3. Check `marketing/social-post-ledger.json` if present, sync-state public-action locks, and `outreach/distribution_log.csv`.
4. Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.
5. Do not use browser or Computer Use from the recurring worker.
6. Do not claim listed domains are customers, endorsements, paid leads, private demand, completed payments, revenue, deliverability, workflow quality, integration reliability, privacy compliance, compliance certification, data freshness, seller certification, x402/ACP/MPP support for NHS, paid ranking placement, preferred inclusion, A2A support, or score-methodology bypass.

## Blockers

- `/.well-known/agent-card.json` returns 404; strict Agent Card directory submissions remain gated.
- No repo-local `marketing/social-post-ledger.json` was found; a channel operator must still check the applicable social ledger or sync-state duplicate lock before posting.
- `tools/full-recrawl.lock/` exists as a pre-existing untracked lock directory; no deploy, broad crawl, or runtime mutation should be attempted from this scout run.
