# Score-fix pending cohort recovery

Date: 2026-07-30
Status: no outreach or checkout action performed

## Segment

Use the private score-fix intake cohort as a sales-recovery segment only after
row-level reconciliation in the credentialed fulfillment workflow. Do not use
the aggregate as public demand, revenue, or conversion proof.

## Aggregate evidence

- The redacted score-fix reader reports 20 total rows.
- Ten rows are real-candidate pending intakes: seven are at least 30 days old
  and three are 7-29 days old.
- The real-candidate pending host classes are six `dot_com` and four
  `other_tld`.
- No real-candidate paid or lead rows appear in the aggregate.
- The May score-fix ledger records two external follow-up sends. Those prior
  recipients and locks must be reconciled before any new contact.
- The free monitor has five active rows and three quarantined rows. The
  2026-07-27 worker completed five due checks.

No emails, hostnames, row ids, notes, Stripe ids, checkout URLs, payment ids,
or private customer data were read into or written to this artifact.

## Public conversion boundary

- `https://nothumansearch.ai/score`, `/monitor`, `/report`, `/newest`, `/top`,
  `/mcp`, `/llms.txt`, `/openapi.yaml`, `/api/v1`, `/api/v1/catalog`,
  `/api/v1/api-keys/subscribe`, `/.well-known/mcp.json`,
  `/.well-known/agent.json`, `/.well-known/commerce.json`, and
  `/.well-known/ai-plugin.json` returned HTTP 200.
- `https://nothumansearch.ai/.well-known/agent-card.json` returned HTTP 404,
  so A2A and Agent Card positioning remains blocked.
- High-score `/fix/{host}` checks for `nothumansearch.ai` and
  `claudereviews.com` returned the already-meets-target handoff.
- The partial-score `stripe.com` route still returned the $199 managed-fix
  offer.
- Current public stats are 4,367 indexed sites with average score 38.
- Live MCP `tools/list` returned 11 tools.

The API-plan discovery drift remains open, so recovery copy must omit API plan
names, prices, and quotas.

## Recovery brief

The next credentialed sales worker should:

1. Read the ten pending rows privately and reconcile them against
   `ops/ledgers/score-fix-pending-followup-2026-05-12.md`, delivery records,
   duplicate ledgers, and prior public-action locks.
2. Classify each row as already contacted, internal/test, invalid or stale
   owner route, no longer eligible after a fresh score check, or eligible for
   one owner follow-up.
3. Re-probe the exact public profile, canonical host, declared OpenAPI/MCP/API
   paths, current robots policy, and `/fix/{host}` checklist before drafting.
   Recent NHS reviews found cached signals and conventional-path assumptions
   that could make an old checklist inaccurate.
4. For an eligible row, verify the active Foundry/Owl-owned sending identity,
   confirm no duplicate touch, claim a sync-state public-action lock, and send
   at most one recovery touch in that execution window.
5. Record the message id and delivery result in the private workflow. Commit
   only aggregate counts by classification and outcome.

## Copy boundary

Frame any later message as help completing implementation against the same
public readiness checks applied to every indexed site. Do not sell placement,
preferred inclusion, ranking, certification, A2A support, score-methodology
bypass, or capabilities the owner already exposes at declared non-root paths.
Do not imply the cohort proves customers, demand, payments, or revenue.
