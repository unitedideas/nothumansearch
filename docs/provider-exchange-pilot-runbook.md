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
2. deploying migrations 019 through 021 and the provider exchange revision;
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

- [ ] Resolve the candidate to a full 40-character Git commit and build only
      from `git archive <commit>` extracted into a new temporary directory.
      Never deploy from the working checkout. Pass the same commit as Docker
      build argument `RELEASE_REVISION`; a missing or malformed argument must
      fail the image build. Use `tools/prepare-exact-release.sh <commit>`; it
      verifies the archive and prints but does not execute the owner-authorized
      deploy command. The archive-expanded `release-source-revision` must equal
      the build argument, so a normal working-tree context fails closed.
- [ ] Full Go tests, race tests, vet, build, formatting, OpenAPI parse, and
      secret scan pass on the exact candidate revision.
- [ ] Migrations 001 through 021 and immediate replay pass on disposable real
      PostgreSQL from an empty database.
- [ ] The target database has neither an unreceipted migration-019/020/021 footprint
      nor a newer protected receipt. Migration 019 must atomically create its
      schema and `nhs_schema_migrations` receipt with the raw-file SHA-256 and
      embedded release commit; migration 020 must do the same for the
      provider-independent action-interest schema and cumulative protected
      fingerprint; migration 021 must do the same for the capacity-reservation
      schema and rolling-writer constraint. A checksum mismatch, missing
      required object, unreceipted footprint, or database-ahead state is a hard
      stop. Never adopt an unrecorded schema because it looks current.
- [ ] The server is the sole schema-migration owner. Crawler and recrawl jobs
      connect to the already-current schema and contain no migration or direct
      schema-repair path.
- [ ] Migration 019 constraints, append-only rules, redaction rule, composite
      tenant keys, one-charge/one-credit invariants, and cap failures pass.
- [ ] Migration 020 exact-organic-result and non-synthetic composite foreign
      keys, action allowlist, immutable confirmation, idempotency, expiry, and
      30-day cascade pass. The table has no query, contact, identity, free-form,
      provider-offer, ticket, budget, or outcome field.
- [ ] Dedicated signing references resolve without displaying their values.
- [ ] Startup reconstructs or verifies a persisted proof sample for every key ID and signing domain; the process fails closed if material was removed or replaced under a reused ID.
- [ ] The controlled pilot runs on one application machine. Before horizontal
      scaling, replace the process-local provider, action-interest, and
      magic-link rate limits with a shared limiter; PostgreSQL DNS leases are
      already multi-instance safe, but the request limiters are not distributed.
- [ ] DNS ownership is current. The persistent TXT proof is automatically
      rechecked; stale ownership cannot publish offers, create tickets, rotate
      keys, or report a positive outcome.
- [ ] Desktop and mobile provider/privacy renders match their reviewed source
      hashes and retain the free-organic, consent, privacy, and evidence limits.
- [ ] The safe live smoke uses synthetic receipts only and proves health,
      release revision, free REST/MCP search, neutral organic order, an empty
      paid-offer sidecar for synthetic demand, selection, tool discovery, and
      fail-closed invalid-receipt verification. A synthetic search must also be
      rejected by `record_action_interest`. The live smoke proves that rejection;
      the disposable PostgreSQL regression proves that rejection inserts no
      action-interest row and creates zero commercial-proof delta. The live
      smoke must not mint a real ticket or submit `accepted`, `activated`, or
      `converted` callbacks.
- [ ] The full paid-flow smoke runs only against disposable PostgreSQL and a
      loopback server. It proves separate offers, consent rejection, exact
      ticket idempotency, nonfinancial terminal rejection, receipt replay and
      tamper detection, and zero commercial-proof delta before destroying the
      fixture.
- [ ] After an owner-authorized deploy, `/health.release_revision`, the image
      `org.opencontainers.image.revision` label, and migrations 019, 020, and
      021 `applied_by_commit` values all equal the authorized commit.
- [ ] Migration 021 is present before any pilot ticket is minted. Each live,
      uncharged ticket has one append-only capacity reservation; the configured
      charge consumes it, an uncharged terminal outcome releases it, and
      expired or emergency-revoked tickets stop consuming capacity logically.

## Stage 1: observe free discovery

Run free search for at least 14 days before choosing the pilot category. Select
one category from persisted, non-synthetic, controlled-topic demand—not index
size, `initialize`, `tools/list`, denied calls, or raw prompts.

Decision evidence:

- at least 100 non-synthetic meaningful search receipts;
- at least 20 persisted result selections on distinct non-synthetic search
  receipts; and
- at least 10 explicit `nhs-action-interest-v1` receipts on distinct
  non-synthetic search receipts for a quote, trial, demo, booking, application,
  signup, or purchase.

These are receipt counts, not unique agents, principals, provider contacts,
accepted handoffs, revenue, or commercial proof. An action-interest receipt
does not contact the provider; it only proves that NHS recorded the caller's
versioned attestation of principal interest against an exact organic result.
The owner-only Stage 1 report must show a 14-day observation span and all three
distinct-receipt targets before `stage1_ready` can be true.

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
