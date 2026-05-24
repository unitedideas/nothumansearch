# Local AI Runtime Distribution Source Readiness

Run: 2026-05-24T05:08:29Z  
Automation: business-marketer-not-human-search

## Evidence

- Public stats: 4,177 indexed sites, average score 35, top category `developer`.
- Public discovery surfaces returned 200: `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/api/v1/catalog`, `/openapi.yaml`, `/monitor`, `/score`.
- `/.well-known/agent-card.json` returned 404, so A2A/Agent Card directory claims stay blocked.
- Aggregate MCP analytics, 7 days: `tools/list` 172,314, `initialize` 27,431, `tools/call` 401.
- Aggregate tool calls, 7 days: `search_agents` 174, `check_url` 84, `get_site_details` 60, `submit_site` 21, `get_stats` 21.
- Aggregate MCP query themes included Linux gaming distribution, Steam Deck UI/HDR/VRR, local quantized model runtimes, embedded hardware, and agent marketplace publishing.
- Aggregate traffic, 168 hours: `/` 3,319, `/badge/xquik.com.svg` 2,525, `/.well-known/commerce.json` 1,381, `/site/xquik.com` 949, `/.well-known/ai-plugin.json` 639, `/llms.txt` 452, `/openapi.yaml` 391, `/api/v1/catalog` 323, `/api/v1/search` 202.

## Segment

Local AI runtime distributors, Linux/Steam Deck AI tooling, GGUF/model package publishers, embedded AI hardware projects, and developer-runtime owners need source contracts an agent can inspect before recommending or installing anything:

- supported hardware and OS boundaries;
- model/package source and update metadata;
- API or CLI contract docs;
- `llms.txt`, OpenAPI, MCP, or structured API discovery;
- monitorable drift on install, compatibility, pricing, and support pages.

## Owner Routing

- High-score owners: route to free monitor registration, public report sharing, and badge/report proof.
- Partial-score owners: route to `/score` first, then a missing-surface checklist before score-fix remediation.
- Runtime/package/API-heavy callers: route to API-key/catalog surfaces only when the docs remain useful.
- Directory/public-channel use: gated on active account identity, duplicate checks, public-action lock, and fresh surface probes.

## Claims To Avoid

Do not claim model quality, benchmark truth, hardware compatibility, gaming performance, driver reliability, install safety, package integrity, security/privacy compliance, support quality, pricing accuracy, update freshness, customer demand, completed payments, revenue, endorsement, paid placement, preferred inclusion, A2A support while `/.well-known/agent-card.json` is 404, or score-methodology bypass.
