# Reliability-oracle interoperability brief

Date: 2026-07-30
Status: no outreach, submission, account action, or public post performed

## Segment

Treat named MCP reliability and risk crawlers as a technical ecosystem segment,
not as proof of owner demand or a directory-placement opportunity. The best
bounded channel test is one interoperability conversation with Glimind: NHS
measures public agent-readiness and exposes live URL/MCP checks, while Glimind
publishes live reliability and call-preparation data.

## Fresh evidence

- NHS aggregate MCP analytics for seven days recorded 48,642 `tools/list`,
  14,559 `initialize`, and 233 `tools/call` requests.
- The named Glimind reliability-oracle crawler family accounted for 3,754
  requests. Its public FAQ says its breadth data comes from safe MCP
  `initialize` plus `tools/list` liveness probes that never invoke a tool.
- Glimind's public docs expose MCP, REST, SDK, OpenTelemetry, status, and
  reliability-feed integration paths. This is a complementary technical
  surface, not a public directory intake.
- The named AgentSure scanner family accounted for 84 requests. AgentSure's
  public site offers AI-risk scans and an AI Risk Map, but no public MCP
  directory or listing intake was found; `/llms.txt` and
  `/.well-known/mcp.json` returned 404.
- Two smaller named scanner/registry families could not be qualified as
  public acquisition channels: one advertised domain had a certificate-host
  mismatch and another advertised domain did not resolve. They remain
  no-submit candidates.
- `agent-tools.cloud` remains the only qualified crawler-derived directory
  candidate in this cohort. It already has a separate packet and marketer
  inbox row, so this run did not duplicate it.
- NHS live JSON-RPC `tools/list` returned 11 tools and matched the 11 tool
  names in `/.well-known/mcp.json`. The in-repo registry manifest remains
  version `1.7.1` with the same streamable-HTTP endpoint.
- NHS `/mcp`, `/llms.txt`, `/openapi.yaml`, `/api/v1`,
  `/.well-known/mcp.json`, `/.well-known/agent.json`, `/monitor`,
  `/score`, and `/report` returned 200.
  `/.well-known/agent-card.json` returned 404, so A2A positioning remains
  blocked.
- Current public stats are 4,367 indexed sites with average score 38.
- The free monitor has five active rows and three quarantined rows. Its latest
  worker evidence completed five due checks on 2026-07-27.

No raw MCP queries, private monitor rows, private score-fix rows, emails,
payment identifiers, checkout URLs, API keys, or customer identifiers were
written to this artifact.

## Interpretation guard

The 209:1 `tools/list`-to-`tools/call` ratio is substantially explained by
liveness and catalog polling. Do not cite those discovery requests as customer
demand, owner intent, adoption, revenue, endorsement, or successful tool use.
Use `tools/call`, monitor registrations, verified directory receipts, and
delivery results for later funnel claims.

## Gated owner-channel test

Prepare one technical note to Glimind's public operator channel after identity,
duplicate, and public-action-lock checks:

> Not Human Search scores public agent-readiness and exposes live `check_url`
> and `verify_mcp` tools. Glimind publishes live reliability and call-prep
> data. A small interoperability check could document where readiness ends and
> runtime reliability begins without merging either score or implying an
> endorsement.

The test should ask only whether a documented cross-reference or example is
useful. It should not ask for paid placement, preferred inclusion, ranking
treatment, certification, or access to private telemetry.

## Execution gate

Before any external touch:

1. Refresh Glimind's official docs, FAQ, status, MCP, and contact surfaces.
2. Refresh NHS stats, JSON-RPC `tools/list`, MCP/agent manifests, `llms.txt`,
   OpenAPI, `/api/v1`, monitor worker proof, and Agent Card status.
3. Confirm no Glimind fingerprint exists in the portfolio social ledger,
   `outreach/distribution_log.csv`, or a current public-action lock.
4. Verify the active Foundry/Owl-owned sending identity and claim the required
   sync-state public-action lock.
5. Send at most one technical note and record the delivery result or public
   URL in the appropriate ledger.

Do not imply Glimind, AgentSure, any crawler, registry, or scanner is a customer,
partner, endorsement, paid lead, monitor registration, certification, private
demand, completed payment, revenue source, or proof of NHS uptime or security.
Do not claim A2A support while `/.well-known/agent-card.json` is 404, or claim
x402/ACP/SPT/MPP support, paid ranking, preferred inclusion, or a score-method
bypass.

