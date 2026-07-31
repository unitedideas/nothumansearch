# AgentNDX free curated MCP directory candidate

Date: 2026-07-30
Automation: `business-marketer-not-human-search`
Status: no submission, outreach, account creation, public post, purchase, browser action, deploy, or crawl performed

## Candidate

[AgentNDX](https://agentndx.ai/) is a viable free directory candidate for the
existing Not Human Search remote MCP server.

The public directory currently exposes 911 curated entries through
`/api/servers.json`. Its submission page says:

- submissions are reviewed rather than self-published;
- review normally occurs within 48 hours;
- eligible entries need open source or a documented API, a stable endpoint,
  and genuine utility for agents;
- required fields are server name, public GitHub URL, protocols, description,
  and contact email, with an optional homepage URL.

The free index contains no exact match for `Not Human Search`,
`nothumansearch.ai`, or `ai.nothumansearch/search`. No AgentNDX submission or
listing fingerprint appears in the NHS distribution log, marketer inbox, or
portfolio social ledger. A later executor must repeat all exact checks before
submitting because the directory changes daily.

AgentNDX also sells a $149/month featured slot. Do not buy or request it. NHS
does not sell ranking placement and should not use paid placement as directory
proof.

## Submission packet

- Server name: `Not Human Search`
- GitHub URL: `https://github.com/unitedideas/nothumansearch`
- Homepage: `https://nothumansearch.ai`
- Endpoint: `https://nothumansearch.ai/mcp`
- Protocols: `MCP` only
- Transport: Streamable HTTP
- Authentication: none
- Category: `web`
- Contact: `hello@8bitconcepts.com`

One-line description:

> Search and verify agent-ready APIs and services with live readiness scores,
> MCP checks, and score monitoring.

Do not select A2A or x402. NHS still returns 404 at
`/.well-known/agent-card.json`, and its commerce surfaces explicitly describe
x402 as unavailable.

## Fresh NHS evidence

- Public stats: 4,350 indexed sites; average score 38.
- Live MCP `tools/list`: 11 tools.
- HTTP 200: `/mcp`, `/.well-known/mcp.json`,
  `/.well-known/agent.json`, `/.well-known/commerce.json`,
  `/.well-known/ai-plugin.json`, `/llms.txt`, `/openapi.yaml`, `/api/v1`,
  `/api/v1/catalog`, `/api/v1/api-keys/subscribe`, `/score`, `/monitor`,
  `/report`, `/newest`, and `/top`.
- HTTP 404: `/.well-known/agent-card.json`.
- Seven-day MCP aggregate: 48,771 `tools/list`, 14,465 `initialize`, and 236
  `tools/call` requests; `register_monitor` received 15 calls.
- Free-monitor aggregate: five active and three quarantined registrations.
  The latest scheduled worker completed five due checks on 2026-07-27.
- Score-fix aggregate: ten real-candidate pending rows, with no real paid or
  lead rows.
- The active GitHub identity and repository remote both resolve to
  `unitedideas`.

The live `/llms.txt` still advertises retired starter/pro/scale API plans while
the current catalog and subscribe contract expose one unlimited plan. That
existing product handoff remains open. The AgentNDX entry must describe the
free MCP server only and omit API plan names, prices, and quotas.

No raw MCP queries, raw user-agent strings, emails from private rows, monitor
rows, score-fix rows, checkout URLs, payment identifiers, API keys, or
customer identifiers were written to this artifact.

## Execution contract

1. Refresh AgentNDX's `/submit`, `/llms.txt`, criteria, protocols, categories,
   and free `/api/servers.json` index.
2. Repeat exact duplicate checks for the brand, domain, endpoint, official
   registry id, and GitHub repository. Stop and record the existing URL if any
   match appears.
3. Refresh NHS stats, `initialize`, `tools/list`, discovery manifests,
   OpenAPI, API root, score, monitor, report, and latest monitor-worker proof.
4. Verify the active Foundry GitHub identity is `unitedideas`, recheck the
   portfolio social ledger and NHS distribution log, and claim a verified
   sync-state public-action lock.
5. Submit exactly one free MCP-only entry using the packet above. Do not
   create reviews, buy featured placement, connect a wallet, or select A2A or
   x402.
6. Record the receipt and later live listing URL in
   `outreach/distribution_log.csv`. Verify the listing's endpoint, transport,
   authentication, tool inventory, and description after publication.

Do not claim AgentNDX acceptance before a receipt or live URL, endorsement,
verification before the directory completes its review, customer demand,
real-user tool success, completed payments, revenue, security certification,
uptime guarantees, paid placement, preferred inclusion, x402/ACP/SPT/MPP
support, A2A support while the Agent Card route is 404, or score-methodology
bypass.
