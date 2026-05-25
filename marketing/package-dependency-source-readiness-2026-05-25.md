# Package And Dependency Source Readiness

Generated: 2026-05-25
Automation: `business-marketer-not-human-search`
Status: no-submit scout artifact; public use still requires account identity verification, duplicate checks, and a sync-state public-action lock.

## Segment

Package registries, dependency-intelligence tools, software supply-chain scanners, SBOM/vulnerability feeds, SDK catalogs, and developer API owners that need agents to inspect source contracts without guessing from prose.

## Evidence Checked

- Public stats: `total_sites=4180`, `avg_score=35`, `top_category=developer`.
- Live surfaces returned HTTP 200: `/api/v1`, `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/commerce.json`, `/.well-known/agent.json`, `/.well-known/ai-plugin.json`, `/openapi.yaml`, `/mcp`, `/score`, `/monitor`, `/api/v1/catalog`.
- `/.well-known/agent-card.json` is still absent, so A2A/Agent Card claims stay blocked.
- MCP aggregate for the last 7 days: `tools/list=174394`, `initialize=25244`, `tools/call=365`; top tool calls were `search_agents=141`, `check_url=85`, `get_site_details=57`, `get_stats=21`, `submit_site=20`, `verify_mcp=13`.
- Traffic aggregate for the last 168 hours included machine-readable discovery and commerce routes: `/.well-known/commerce.json=1342`, `/.well-known/ai-plugin.json=609`, `/llms.txt=440`, `/openapi.yaml=376`, `/api/v1/catalog=320`, `/api/v1/search=222`, `/api/v1/submit=146`.
- Developer top-list examples included high-score developer surfaces such as `mcp.depscope.dev`, `deadends.dev`, `agentdomainsearch.com`, `agentndx.ai`, and `gptr.dev`. Treat these only as public readiness examples, not customers or endorsements.
- Score-fix routing check: high-score `/fix/nothumansearch.ai` routes to already-meets-target/monitor messaging; partial-score `/fix/manifest.ly` still shows paid remediation intake.
- Latest local monitor worker proof remains `2026-05-18` completed clean for one due monitor; refresh before publishing monitor-heavy copy.

## Owner-Channel Angle

Agents increasingly ask for machine-readable package, dependency, SDK, vulnerability, and developer-tool facts. NHS can position itself as the probe-before-use layer for whether those sources expose stable contracts:

- `llms.txt` and discovery metadata for agent instructions.
- OpenAPI/API/MCP surfaces for package, dependency, SDK, and vulnerability records.
- Monitorable drift when a source changes or loses a machine-readable surface.
- Score-band routing: high-score owners get monitor/report/badge proof; partial-score owners get `/score` plus a missing-surface checklist before any paid remediation.

## Draft Brief

Package and dependency tools are becoming agent inputs, not just developer websites.

NHS currently indexes 4,180 agent-readable sites and scores them on whether an agent can verify the machine-readable surface before using it. For package registries, dependency scanners, SBOM feeds, and SDK catalogs, that means `llms.txt`, OpenAPI/API, MCP, plugin metadata, robots policy, and monitorable drift.

This is useful for two groups:

1. Tool owners can check whether agents see a usable contract or only a marketing page.
2. Agent builders can route source discovery through a readiness check before trusting package or dependency metadata.

Run a public score first, then route by score band:

- High-score profile: monitor it and use the public report/badge as proof.
- Partial-score profile: fix missing machine-readable surfaces before paid remediation.
- API-heavy use: keep API-key/catalog paths separate from owner remediation.

## Boundaries

Do not claim package quality, dependency safety, vulnerability accuracy, SBOM completeness, supply-chain security, package integrity, registry coverage, package freshness, install safety, model/tool endorsement, private demand, completed payments, revenue, paid placement, preferred inclusion, A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/SPT/MPP support for NHS, or score-methodology bypass.

Do not publish raw user-agent strings, private query logs, raw checkout URLs, payment identifiers, buyer emails, private monitor rows, private score-fix rows, or private customer identifiers.

## Next Gated Action

Prepare one owner-channel or directory-candidate test for package/dependency/source-readiness audiences. Before external use, refresh the live route probes, admin aggregates, monitor worker proof, representative `/site/{host}` pages, and both high-score and partial-score `/fix/{host}` routes.
