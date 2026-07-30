# MCP Find package-gated directory candidate

Date: 2026-07-30
Status: no package publication, submission, account action, outreach, or public post performed

## Target

- Directory: [MCP Find](https://mcpfind.org/)
- Submission form: [mcpfind.org/submit](https://mcpfind.org/submit)
- Contribution contract:
  [MCPFind/mcp-find CONTRIBUTING.md](https://github.com/MCPFind/mcp-find/blob/main/CONTRIBUTING.md)
- Intended category: `search`

MCP Find currently describes an open directory of 18,182 MCP servers across
19 categories. Its submission path creates a GitHub pull request rather than a
separate directory account.

## Qualification check

MCP Find requires all of the following:

1. A public GitHub repository.
2. A recognized open-source license.
3. A README with install or configuration instructions.
4. At least one callable MCP tool.
5. A package published to npm, PyPI, or a Docker registry.

NHS currently satisfies the first four:

- Public repo: `https://github.com/unitedideas/nothumansearch`
- License: MIT, detected by GitHub's license endpoint.
- README: documents the hosted streamable-HTTP MCP endpoint and tool surface.
- Live endpoint: `https://nothumansearch.ai/mcp`
- Live tool count: 11.
- Official registry id: `ai.nothumansearch/search`, manifest version `1.7.1`.
- The repo was updated on 2026-07-30.

The package requirement is the blocker. Exact npm and PyPI searches returned no
NHS package. The repo contains a Dockerfile, but a Dockerfile is not a
published image and no qualifying public NHS Docker package was verified in
this run. The submission form requires a package name and accepts only
`npm`, `pypi`, or `docker`.

Do not submit a fake package name, use the registry id as though it were a
package, or open a PR before a public installable artifact exists.

## Duplicate check

- No `mcpfind`, `mcp find`, or `mcp-find` fingerprint exists in the local
  marketer inbox, NHS distribution history, or portfolio social ledgers.
- Public search by the brand name returned no exact NHS result.
- Searches containing the registry id or endpoint echoed the query text inside
  the page's application state. A plain HTML grep therefore creates a false
  positive. A later execution worker must inspect actual result cards or use
  the directory's own search data before treating a match as a duplicate.

## Fresh NHS evidence

- Public stats: 4,364 indexed sites; average score 38.
- JSON-RPC `tools/list`: 11 tools.
- HTTP 200: `/mcp`, `/.well-known/mcp.json`,
  `/.well-known/agent.json`, `/.well-known/commerce.json`,
  `/.well-known/ai-plugin.json`, `/llms.txt`, `/openapi.yaml`, `/api/v1`,
  `/api/v1/catalog`, `/monitor`, `/score`, and `/report`.
- HTTP 404: `/.well-known/agent-card.json`; A2A positioning remains blocked.
- Seven-day MCP aggregate: 48,667 `tools/list`, 14,482 `initialize`, and
  235 `tools/call` requests. `register_monitor` recorded 15 calls.
- Free-monitor aggregate: five active and three quarantined registrations.
  The latest scheduled worker completed five due checks on 2026-07-27.
- Score-band routing remains intact: a 100-score host receives the
  monitor/report handoff, while a current 90-score host receives the legitimate
  self-serve report and managed-remediation options.

No raw user-agent strings, MCP queries, monitor rows, score-fix rows, emails,
payment identifiers, checkout URLs, API keys, or customer identifiers were
written to this artifact.

## Execution contract

1. Choose one minimal public package that accurately launches or connects to
   the NHS MCP server. A public Docker image is the native fit for the current
   Go/Docker repo; do not publish an npm or PyPI wrapper that cannot be
   installed and smoked end to end.
2. Verify the public package can be pulled and run, the README describes the
   exact install/configuration path, and the resulting server exposes the
   intended 11-tool surface without embedding credentials or private state.
3. Refresh MCP Find's contribution requirements and repeat exact searches for
   the brand, repo, package, registry id, domain, and endpoint. Stop if an
   existing listing is found.
4. Verify the active Foundry GitHub identity is `unitedideas`, check the
   portfolio social ledger and `outreach/distribution_log.csv`, then claim and
   verify a sync-state public-action lock.
5. Open exactly one contribution PR using the real package name and `search`
   category. Record the PR URL and later review outcome in
   `outreach/distribution_log.csv`.

The listing must describe the free hosted MCP search and verification tools.
Do not claim MCP Find acceptance before a receipt or merged listing, directory
endorsement, customer demand, successful user tool calls, completed payments,
revenue, security certification, uptime proof, paid placement, preferred
inclusion, x402/ACP/SPT/MPP support, A2A support while the Agent Card route is
404, or score-methodology bypass.
