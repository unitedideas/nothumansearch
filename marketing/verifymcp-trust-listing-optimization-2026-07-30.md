# VerifyMCP trust-listing optimization brief

Date: 2026-07-30
Status: no submission, outreach, account action, purchase, or public post performed

## Segment

Treat VerifyMCP as an existing independent directory placement and product-trust
feedback loop, not as a new submission target. VerifyMCP already imports Not
Human Search from the official MCP registry, probes the remote endpoint daily,
and publishes a component page with reproducible diagnostics.

Public listing:
`https://verifymcp.io/servers/ai-nothumansearch-search`

## Fresh evidence

- VerifyMCP's exact-match directory search returned one NHS result, registry id
  `ai.nothumansearch/search`, remote endpoint
  `https://nothumansearch.ai/mcp`, and trust score `66/100`.
- The listing was modified on `2026-07-30T13:29:44.611Z` and showed a current
  same-day scan. VerifyMCP's public directory reported 18,633 indexed MCP
  servers.
- VerifyMCP's official About and scoring documentation says it continuously
  syncs the official MCP registry, re-probes indexed components daily, accepts
  no payment to improve a score or rank, and keeps advertising separate from
  the scoring rubric.
- NHS aggregate MCP analytics recorded 45 requests from the named VerifyMCP
  probe family in the last seven days. This is scanner activity, not customer
  demand or successful tool use.
- A bounded compatibility smoke using the VerifyMCP probe user-agent completed
  MCP `initialize` and `tools/list`. NHS negotiated protocol version
  `2025-06-18` and returned all 11 tools.
- The official registry and in-repo manifest both expose version `1.7.1`, the
  same remote streamable-HTTP endpoint, and a description that still says
  `4,100+ sites`. Live NHS stats report 4,367 indexed sites and average score
  38.
- NHS `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/llms.txt`, `/openapi.yaml`, `/api/v1`, `/api/v1/catalog`, `/monitor`,
  `/score`, and `/report` returned HTTP 200. The Agent Card compatibility route
  `/.well-known/agent-card.json` returned 404, so A2A positioning remains
  blocked.
- The free monitor currently has five active registrations and three
  quarantined registrations. The redacted score-fix aggregate has ten pending
  real-candidate rows and no real-candidate paid or lead rows. Keep both
  aggregates out of public demand and revenue claims.

No raw MCP queries, emails, private monitor rows, private score-fix rows,
checkout URLs, payment identifiers, API keys, or customer identifiers were
written to this artifact.

## Independent score gaps

VerifyMCP's live diagnostics separate the current `66/100` score into these
actionable and time-based gaps:

1. Endpoint Security: `63`. HTTPS, TLS, HSTS, and plaintext-to-HTTPS redirect
   checks passed. Authorization remained unverified because none of the 11
   tools declared `destructiveHint`; the MCP default treats an absent hint as
   destructive. DNSSEC was also absent.
2. Schema Quality and AI Usability: `72`. Instruction clarity passed and every
   tool plus parameter had a substantive description. The 11-tool schema used
   about 1,474 tokens, above VerifyMCP's context budget, and no tool declared a
   usage example.
3. Tool Coverage: `100`.
4. Stability and Change Management: `13`. VerifyMCP had observed only four of
   the required 30 stable days. This portion should accrue without marketing
   action if the surface remains stable.
5. Capabilities: `100`. The live streamable-HTTP handshake and supported MCP
   protocol version passed.

These findings are third-party diagnostics, not a security certification or
proof that the endpoint is safe for every use.

## Recommended handoff

The next product worker should use the public VerifyMCP page as an acceptance
surface:

1. Add accurate MCP tool annotations. Mark read-only tools as non-destructive
   and distinguish `submit_site` and `register_monitor` from read-only calls;
   do not blanket-mark every tool safe.
2. Add concise usage examples in the MCP schema format VerifyMCP recognizes.
3. Reduce schema context cost without removing the descriptions that currently
   earn full tool and parameter coverage.
4. Treat DNSSEC as a separate DNS-owner decision, not a marketing copy fix.
5. Keep the official registry manifest, live `tools/list`, MCP discovery
   manifest, `llms.txt`, OpenAPI, and `/api/v1` coherent. Refresh the rounded
   registry count only in the same registry maintenance cycle; do not bump a
   version solely for cosmetic count churn.
6. Wait for VerifyMCP's next daily probe and capture the public score and
   diagnostics. Only then consider one technical cross-reference or proof
   brief through an existing verified Foundry/Owl-owned channel.

There is no directory submission to perform. Do not buy advertising to imply a
better score or ranking, and do not ask the operator to override reproducible
diagnostics.

## Duplicate and claim boundary

No VerifyMCP fingerprint existed in the portfolio social-post ledger,
`outreach/distribution_log.csv`, or the NHS marketer inbox before this row was
queued. Any later public note still requires a fresh identity check, duplicate
check, and sync-state public-action lock.

Do not claim VerifyMCP endorsement, partnership, certification, customer
demand, real-user tool success, completed payments, revenue, security approval,
uptime proof, paid placement, preferred inclusion, A2A support while the Agent
Card route is 404, or a score-methodology bypass.
