# Not Human Search provider exchange pilot

This runbook is only for Not Human Search. AI Dev Board is not part of the
product, implementation, economics, telemetry, or proof gate described here.

## Commercial contract

- Organic search, result membership, result order, readiness score, canonical
  provider links, and direct provider access stay free.
- Providers may fund a separately disclosed action only for a site that the
  neutral search already returned.
- NHS charges only the provider-defined `accepted`, `activated`, or `converted`
  event recorded by an authenticated, idempotent provider callback.
- NHS never sells raw queries, per-agent histories, alleged agent identities,
  principal identities, or contact data.
- A ticket contains controlled intent fields and the caller's exact
  `nhs-principal-consent-v1` authorization attestation. NHS does not claim to
  verify the caller's identity, agency, or legal authority.
- Pilot terms must make the provider Merchant of Record. NHS records the agreed
  debit or capped receivable; the pilot endpoint does not move principal money.

## Owner-controlled release gates

Do not deploy or activate provider inventory until Shane explicitly authorizes
the release and each applicable external action. In particular, owner authority
is required before:

1. creating and storing the dedicated production signing secret and key ID;
2. deploying migration 019 and the provider exchange revision;
3. sending provider outreach or invitations;
4. accepting a contract, affiliate term, CPA term, funding representation, or
   Merchant-of-Record representation;
5. recording external funding or activating an offer; and
6. making any public account action or incurring spend.

The signing key must be at least 32 random bytes, dedicated to this exchange,
and injected through the approved secret bridge. A rotation must retain every
prior key ID still referenced by an action ticket or outcome receipt; production
startup reconstructs or verifies retained proof samples and fails closed if
referenced material is absent or replaced under a reused ID.

## Technical release checklist

- [ ] Full Go tests, race tests, vet, build, formatting, OpenAPI parse, and
      secret scan pass on the exact candidate revision.
- [ ] Migrations 001 through 019 and immediate replay pass on disposable real
      PostgreSQL from an empty database.
- [ ] The target database is confirmed not to have recorded an earlier form of
      migration 019. If it has, ship freshness changes in a new ALTER/backfill
      migration instead of editing the already-applied file.
- [ ] Migration 019 constraints, append-only rules, redaction rule, composite
      tenant keys, one-charge/one-credit invariants, and cap failures pass.
- [ ] Dedicated signing references resolve without displaying their values.
- [ ] Startup reconstructs or verifies a persisted proof sample for every key ID and signing domain; the process fails closed if material was removed or replaced under a reused ID.
- [ ] The controlled pilot runs on one application machine. Before horizontal
      scaling, replace the process-local provider and magic-link rate limits
      with a shared limiter; PostgreSQL DNS leases are already multi-instance
      safe, but the request limiters are not distributed.
- [ ] DNS ownership is current. The persistent TXT proof is automatically
      rechecked; stale ownership cannot publish offers, create tickets, rotate
      keys, or report a positive outcome.
- [ ] Desktop and mobile provider/privacy renders match their reviewed source
      hashes and retain the free-organic, consent, privacy, and evidence limits.
- [ ] Deployment smoke proves free REST/MCP search, neutral organic order,
      separate paid offers, ticket idempotency, signed receipt verification,
      current receipt state, and non-secret health responses.

## Stage 1: observe free discovery

Run free search for at least 14 days before choosing the pilot category. Select
one category from persisted, non-synthetic, controlled-topic demand—not index
size, `initialize`, `tools/list`, denied calls, or raw prompts.

Decision evidence:

- at least 100 non-synthetic meaningful search receipts;
- at least 20 detail, verification, or result-selection actions; and
- at least 10 voluntary requests for a quote, trial, demo, booking, application,
  or equivalent provider action.

These are demand signals, not revenue or commercial proof.

## Stage 2: bounded provider pilot

Invite 10 to 20 providers in the one selected category only after owner release.
Every provider must:

1. control an indexed domain and keep the returned DNS TXT proof published;
2. use a same-domain HTTPS action URL;
3. state the action, principal price, NHS bounty, charge event, refund/credit
   rule, response expectation, and Merchant-of-Record responsibility exactly;
4. choose prepaid funding or exact capped CPA terms with non-secret evidence;
5. save its returned-once callback key; and
6. accept that organic rank and readiness score cannot be bought.

Manually review each provider's first offer and each provider's first ticket.
Start with the smallest credible event, normally a provider-accepted handoff or
real activation—not an impression, API call, click, redirect, or model score.

## Truthful proof gate

The pilot is commercially proven only when the production evidence tables and
owner-held external receipts jointly establish all of the following:

- 3 distinct provider accounts placed real prepaid budgets or accepted exact
  capped CPA terms;
- 5 non-synthetic, consented handoffs were accepted by providers;
- 2 non-synthetic activations were reported by providers;
- 1 provider meaningfully replenished a budget after a real charge or renewed
  its exact terms; and
- any claimed collection is backed by an external payment/ledger receipt, not
  merely an internal debit or receivable.

Provider callbacks are authenticated assertions. NHS-signed receipts prove what
NHS recorded; neither fact independently audits the underlying business event.

## Stop conditions

Pause the pilot and do not broaden inventory if any of these occurs:

- providers consume reports but none will prepay or accept exact CPA terms;
- agents bypass tickets because the structured action adds no useful value;
- paid offers change or appear to change organic membership, order, or score;
- invalid/duplicate disputes cannot be resolved from signed, idempotent state;
- consented handoffs expose raw queries, identities, contact data, or free-form
  sensitive intent;
- provider ownership, action URL, signing verification, or accounting state is
  stale or unverifiable; or
- provider-reported acceptances fail to produce real activations or renewal.

Do not substitute a successful deploy, healthy scheduler, signed receipt,
internal ledger entry, or green dashboard for the commercial proof gate.
