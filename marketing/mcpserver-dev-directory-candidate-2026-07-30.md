# MCP Server Hub directory candidate

Date: 2026-07-30
Automation: `business-marketer-not-human-search`
Status: candidate only; no submission, outreach, account action, browser action,
public post, deploy, crawl, checkout, or QLimit/global-queue write performed

## Candidate

- Directory: `https://mcpserver.dev/`
- Submission surface: the public home page exposes a `Submit server` form while
  signed out.
- Stated fit: a public MCP server repository or page submitted for review.
- Required fields: URL, title, and description.
- Optional fields: logo, category, and keywords.
- Recommended category: `Search & Knowledge`.
- Current directory state observed in page application data: 150 servers, 398
  skills, and 14 server categories.

This is a directory candidate, not evidence of acceptance, endorsement,
traffic, customer demand, or a relationship with the directory operator.

## Fresh NHS proof

- `https://nothumansearch.ai/mcp` negotiated MCP protocol `2025-06-18` and
  returned 11 tools without an API key.
- `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/.well-known/ai-plugin.json`, `/openapi.yaml`, `/api/v1`, `/monitor`, and
  `/score` returned HTTP 200.
- Public stats reported 4,351 indexed sites with average score 38 and
  `developer` as the top category.
- Seven-day aggregate MCP analytics recorded 49,048 `tools/list`, 14,384
  `initialize`, and 238 `tools/call` requests. This is discovery-funnel
  evidence, not customer or adoption evidence.
- The free-monitor aggregate contains five active and three quarantined
  registrations. The redacted score-fix aggregate contains ten real-candidate
  pending rows and no real-candidate paid or lead rows.
- `/.well-known/agent-card.json` remains HTTP 404, so the entry must be
  MCP-only and must not claim A2A support.
- API-plan discovery surfaces still have a separate open coherence handoff.
  Keep plan names, prices, quotas, and unlimited-access claims out of this
  directory packet.

No raw MCP queries, raw user-agent strings, emails, private monitor rows,
private score-fix rows, checkout URLs, payment identifiers, API keys, or
customer identifiers were written to this artifact.

## Duplicate check

No exact match for `Not Human Search`, `nothumansearch.ai`,
`https://nothumansearch.ai/mcp`, `ai.nothumansearch/search`, or
`https://github.com/unitedideas/nothumansearch` appeared in:

- the directory's embedded public listing data;
- result cards returned by exact brand, domain, registry-id, and repository
  searches;
- `outreach/distribution_log.csv`;
- `ops/sweeper/marketer-inbox.jsonl`;
- existing NHS marketing artifacts; or
- the portfolio social-post ledger.

Search pages echo query text into application state, so query-string text alone
is not a positive match. A later publisher must repeat this check against
actual result cards and listing data immediately before submission.

## Draft submission packet

**URL**

`https://github.com/unitedideas/nothumansearch`

**Title**

`Not Human Search`

**Description**

`Search and score websites for agent readiness, inspect discovery surfaces, verify MCP endpoints, and register score monitors through a free remote MCP server.`

**Category**

`Search & Knowledge`

**Keywords**

`MCP server, agent search, agent readiness, llms.txt, OpenAPI, website scoring, MCP verification`

**Remote endpoint**

`https://nothumansearch.ai/mcp`

Use the public repository as the submitted URL because the form explicitly
accepts a public repository or page. Preserve the remote endpoint in the
description or any endpoint field the refreshed form exposes.

## Publisher acceptance test

1. Refresh the directory home page, submission rules, field schema, category
   list, and current listing data without creating an account.
2. Repeat exact duplicate checks for the brand, domain, remote endpoint,
   registry id, and repository. Stop and record the existing listing URL if any
   actual match appears.
3. Refresh NHS stats, MCP `initialize` and `tools/list`, registry metadata,
   discovery surfaces, aggregate monitor state, aggregate score-fix state, and
   latest monitor-worker proof.
4. Verify the active Foundry identity, recheck the portfolio social ledger and
   NHS distribution history, then take a verified sync-state public-action lock
   before submitting exactly once.
5. Do not create an account from the recurring worker. If the refreshed
   submission requires login or another unavailable credential, record that
   exact blocker and stop.
6. Record the submission receipt and later live listing URL in
   `outreach/distribution_log.csv`; verify the published remote endpoint,
   transport, no-auth boundary, tool inventory, category, and description.

Do not claim directory acceptance before proof, endorsement, customer demand,
real-user tool success, completed verification before review, completed
payments, revenue, security certification, uptime, paid placement, preferred
inclusion, A2A support while the Agent Card route is 404, API-plan terms while
discovery surfaces disagree, or a score-methodology bypass.
