# MCPub inherited-listing metadata maintenance

Date: 2026-07-30
Automation: `business-marketer-not-human-search`
Status: no submission, outreach, account action, public post, registry publish, deploy, or crawl performed

## Segment decision

[MCPub](https://mcpub.dev/) is an open remote-MCP directory with a
programmatic submission and search surface. NHS does not need a new
submission: MCPub already carries and live-verifies the exact endpoint
`https://nothumansearch.ai/mcp`.

Read-only duplicate checks showed:

- `get("https://nothumansearch.ai/mcp")` returns the existing archive record.
- `search_live("verify_mcp")` returns the NHS endpoint as live, with protocol
  `2025-06-18`, 11 tools, and a successful scanner latency observation.
- `get("https://nothumansearch.ai")` returns `not_found`.
- Description-only searches for the brand, domain, and official registry id
  return zero results even though the exact endpoint is present.

The root miss and description-search misses are not valid absence evidence.
Any later executor must use the exact `/mcp` endpoint lookup before considering
a submission.

## Metadata drift

MCPub currently shows:

> Search: Search the agentic web. 4,100+ sites, 11 tools incl. check_url +
> verify_mcp for probe-before-use.

That description matches official MCP Registry record
`ai.nothumansearch/search` version `1.7.1` and the in-repo
`tools/mcp-registry/server.json`. Live NHS stats now report 4,354 indexed
sites with an average score of 38.

This is inherited registry-copy drift, not a reason to resubmit. The
count-bearing sentence will keep aging. At the next substantive registry
maintenance cycle, use count-independent copy no longer than the registry's
100-character limit:

> Search agent-ready sites. 11 tools for discovery, live scoring, MCP
> verification, and monitoring.

Do not bump the official registry version solely for count churn. Pair this
copy change with the next real tool, schema, annotation, or interoperability
maintenance cycle, then verify the official registry record and MCPub's next
sync or live rescan.

## Current conversion boundaries

- Public stats: 4,354 indexed sites; average score 38.
- Live JSON-RPC `tools/list`: 11 tools.
- HTTP 200: `/mcp`, `/.well-known/mcp.json`,
  `/.well-known/agent.json`, `/.well-known/commerce.json`,
  `/.well-known/ai-plugin.json`, `/llms.txt`, `/openapi.yaml`, `/api/v1`,
  `/api/v1/catalog`, `/api/v1/api-keys/subscribe`, `/score`, `/monitor`,
  `/report`, `/newest`, and `/top`.
- HTTP 404: `/.well-known/agent-card.json`; A2A claims remain blocked.
- Seven-day MCP aggregate: 48,694 `tools/list`, 14,480 `initialize`, and
  236 `tools/call` requests.
- Free-monitor aggregate: five active and three quarantined registrations.
  The latest scheduled worker completed five due checks on 2026-07-27.
- Score-fix aggregate: ten real-candidate pending rows, with no real paid or
  lead rows. The existing private cohort-recovery handoff remains the correct
  sales path; this directory segment does not replace it.

No raw user-agent strings, MCP queries, emails, monitor rows, score-fix rows,
checkout URLs, payment identifiers, API keys, or customer identifiers were
written to this artifact.

## Execution contract

1. Do not submit either the root domain or `/mcp` endpoint to MCPub while the
   exact endpoint record exists.
2. During the next substantive MCP Registry maintenance cycle, refresh the
   official registry, `tools/mcp-registry/server.json`, live `tools/list`,
   `/.well-known/mcp.json`, `/llms.txt`, `/openapi.yaml`, and `/api/v1`
   together. Use the count-independent description only if it remains
   accurate and within 100 characters.
3. Verify the active Foundry GitHub identity is `unitedideas`, recheck the
   portfolio social ledger and `outreach/distribution_log.csv`, and take a
   verified sync-state public-action lock before any registry publish.
4. After publication, wait for MCPub's normal sync or scanner pass. Verify the
   exact `/mcp` record, live status, protocol, and 11-tool inventory. Record
   the updated public URL or proof in `outreach/distribution_log.csv`.

Do not claim MCPub endorsement, customer demand, real-user tool success,
completed payments, revenue, security certification, uptime guarantees, paid
placement, preferred inclusion, x402/ACP/SPT/MPP support, A2A support while
the Agent Card route is 404, or score-methodology bypass.
