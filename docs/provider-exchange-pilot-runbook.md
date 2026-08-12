# Not Human Search provider exchange pilot

This runbook is only for Not Human Search. AI Dev Board is not part of the
product, implementation, economics, telemetry, or proof gate described here.

## Commercial contract

- Organic search, result membership, result order, readiness score, canonical
  provider links, and direct provider access stay free.
- Providers may fund a separately disclosed action only for a site that the
  neutral search already returned.
- Discovery, ticket preparation, handoff, and controlled-intent resolution do
  not issue a payment challenge to the searching agent or principal. Provider
  budget/capacity unavailability is a state conflict, not an HTTP 402 paywall.
- NHS charges only the provider-defined `accepted`, `activated`, or `converted`
  event recorded by an authenticated, idempotent provider callback.
- The bounded launch pilot is exact capped CPA terms only. Provider offer
  creation rejects `prepaid`, and both legacy budget writes and verified-funding
  admin actions fail closed. Migration 029 adds only an owner-created,
  post-action Stripe Checkout for the exact charged terms outcome; it does not
  enable prepaid accounting or charge an agent. Checkout creation is forbidden
  unless both the Stripe secret and the signature-verification webhook secret
  are present, and only a signed paid webhook can append a settlement receipt.
- NHS never sells raw queries, per-agent histories, alleged agent identities,
  principal identities, or contact data.
- A ticket contains controlled intent fields and the caller's exact
  `nhs-principal-consent-v1` authorization attestation. NHS does not claim to
  verify the caller's identity, agency, or legal authority.
- Ticket creation returns the exact bearer plus the NHS handoff endpoint, not
  the attributed provider URL. Continuing through NHS requires a separate
  `principal_handoff_consent=true` attestation under
  `nhs-provider-handoff-consent-v1`; do not expand or reinterpret the ticket
  consent contract to cover this later action.
- The handoff transaction must append one `nhs-action-handoff-v1` receipt bound
  to the exact ticket, offer version, commercial-terms hash, one-way presented-
  bearer hash, and handoff-consent version before returning the attributed
  provider URL. It stores no query, identity, contact, network, referrer, or
  user-agent data and charges neither party.
- Controlled-intent disclosure is a third, optional authorization under
  `nhs-provider-controlled-intent-disclosure-consent-v1`; it is not implied by
  ticket authorization or handoff consent. Declining it must not block the
  observed handoff or free direct provider access. When consented, the same
  DNS-verified provider claim may resolve only the ticket's controlled demand
  topic, optional region code, USD budget band, urgency, allowlisted
  requirement flags, opaque ticket/offer binding, action type, observation
  time, and availability boundary. NHS must not return the search query or
  hash, prompt, free-form text, identity, contact, network, user-agent,
  referrer, credentials, payment, action-URL, price, outcome, or accounting
  fields.
- The `nhs-provider-controlled-intent-resolver-v1` surface is provider-key-
  authenticated REST only, free, read-only, same-claim, and bounded by the
  shorter of ticket authorization and 30-day controlled-intent availability. It
  creates no durable analytics, quota, receipt, outcome, budget, charge, or
  proof record. Its one-hour counting windows use process-local buckets keyed
  before authentication by a 64-bit truncated SHA-256 network-address hash and
  after authentication by provider-key row ID plus that hash; expired entries
  are evicted on later limiter use or process restart and are never logged or
  persisted.
- Pilot terms must require the provider to acknowledge that it remains Merchant
  of Record. This is a provider contractual acknowledgement, not an NHS-verified
  fact. NHS records the agreed debit or capped receivable; the pilot endpoint
  does not move principal money.
- The exact terms hash binds the offer/version, action and destination, charge
  event and CPA, principal price, cap and period, full invalid/duplicate credit
  rule, callback-before-attribution-expiry expectation, the provider's
  Merchant-of-Record acknowledgement, and the rule that a terms period begins
  at first activation.

## Owner-controlled release gates

Do not deploy or activate provider inventory until Shane explicitly authorizes
the release and each applicable external action. In particular, owner authority
is required before:

1. creating and storing the dedicated production signing secret and key ID, or
   the separate owner-only company-deduplication HMAC key;
2. deploying migrations 019 through 032 and the provider exchange revision;
3. sending provider outreach or invitations;
4. accepting a contract, affiliate term, CPA term, funding representation, or
   provider Merchant-of-Record acknowledgement;
5. recording external funding or activating an offer;
6. issuing the immutable signed commercial-proof manifest for an exact pilot;
   and
7. publishing that manifest, making any other public account action, or
   incurring spend.

The signing key must be at least 32 random bytes, dedicated to this exchange,
and injected through the approved secret bridge. A rotation must retain every
prior key ID still referenced by an action ticket, outcome receipt, or issued
proof manifest; production startup reconstructs or verifies retained proof
samples and fails closed if referenced material is absent or replaced under a
reused ID.

This runbook describes the local release candidate and the evidence required
for a future authorized pilot. It is not evidence that migrations 019 through
032 are deployed, that the provider exchange is active in production, or that
any live commercial threshold has been met.

## Technical release checklist

- [ ] Resolve the candidate to a full 40-character Git commit and build only
      from `git archive <commit>` extracted into a new temporary directory.
      Never deploy from the working checkout. Pass the same commit as Docker
      build argument `RELEASE_REVISION`; a missing or malformed argument must
      fail the image build. With two different disposable PostgreSQL DSNs set,
      use `tools/prepare-exact-release.sh <commit> <base-commit>`. For
      local-only exact-archive verification, use
      `go run ./cmd/provider-release-local-postgres --candidate <commit> --base <base-commit>`;
      it starts exactly two temporary PostgreSQL 17 instances and invokes that
      same verifier without printing their DSNs. Both paths verify the archive
      and deliberately emit no deploy command. The
      archive-expanded `release-source-revision` must equal the build argument,
      so a normal working-tree context fails closed. Its
      `nhs-exact-release-verification-v2` receipt binds the exact source archive,
      candidate commit/tree/parent, changed-path count, and the raw SHA-256 of
      every protected migration from 019 through 032.
- [ ] Full Go tests, race tests, vet, build, formatting, OpenAPI parse, and
      changed-path secret scan pass on the exact candidate revision. The
      preparer must observe explicit pass events—not a package-level `PASS`
      after a skip—for both real-PostgreSQL release tests.
- [ ] For local diagnostic coverage without a provisioned DSN, an operator may
      explicitly run `NHS_EMBEDDED_POSTGRES=1 go test -count=1 -run
      'TestProtectedMigrationLedgerPostgres|TestProviderExchangePostgresReleaseRegressions'
      ./internal/database ./internal/models`. This starts temporary
      PostgreSQL 17 fixtures and leaves ordinary test runs unchanged. It is not
      an exact-archive release receipt; use the local release runner above or
      separately provisioned DSNs for that full verification. Neither local
      path replaces the snapshot drill or protected cutover requirements above.
- [ ] After exact verification and separate owner authorization, adopt only the
      verified candidate into its durable namespaced canonical ref with
      `tools/adopt-provider-exchange-candidate.sh CANDIDATE_REPOSITORY COMMIT TREE PARENT EXACT_RELEASE_MANIFEST --confirm-owner-authorized`.
      The adoption command must independently reconstruct the exact source
      archive, recompute every migration-019-through-032 digest, and preserve
      the exact-release manifest plus its SHA-256 beside the Git bundle. It
      emits no build or deployment command; an unadopted temporary-clone commit
      is not a release identity.
- [ ] Migrations 001 through 032 and immediate replay pass on disposable real
      PostgreSQL from an empty database.
- [ ] The target database has neither an unreceipted
      migration-019-through-032 footprint nor a newer protected receipt.
      Migration 019 must atomically create its
      schema and `nhs_schema_migrations` receipt with the raw-file SHA-256 and
      embedded release commit; migration 020 must do the same for the
      provider-independent action-interest schema and cumulative protected
      fingerprint; migration 021 must do the same for the capacity-reservation
      schema and rolling-writer constraint; migration 022 must do the same for
      provider-authenticated company/terms acceptances and owner-verified
      commercial commitments plus the observed-handoff receipt. Migration 023
      must do the same for the optional controlled-intent consent
      pair, one-way authorization/redaction/status invariants, and the resolver's
      bounded disclosure contract. Migration 024 must do the same for the exact
      Stage 2 epoch, frozen 3-to-20-provider cohort, selected-topic binding,
      active-window offer/return/ticket relationships, and hard per-provider
      plus total ticket caps. Migration 025 must do the same for the exact
      returned-result/selection relationship, immutable database-clock-owned
      Stage 1 facts, trusted-generation markers, and the protected Stage 1
      epoch anchor. Migration 026 must do the same for complete canonical pilot
      lifecycle events and exact pilot-bound outcome ticket, handoff, clock,
      and canonical-row integrity. Migration 027 must do the same for immutable
      provider/offer/ticket/handoff/callback owner-review receipts, exact
      database-derived subject bindings, the canonical snapshot-hash function,
      and append-only rules. Migration 028 must do the same for the one-per-pilot
      immutable signed commercial-proof manifest, exact closed-pilot snapshot
      binding, database-owned issue time, privacy-redacted canonical payload,
      and append-only rules. A checksum mismatch, missing required object,
      unreceipted footprint, or database-ahead state is a hard stop. Migration
      029 must add the immutable post-action settlement order, exact Stripe
      Checkout binding, and Stripe-paid receipt tables. It must bind amount,
      currency, provider claim, offer, ticket, charged outcome, and exact terms
      snapshot without storing provider billing contact or agent data. Never adopt
      an unrecorded schema because it looks current.
- [ ] The server is the sole schema-migration owner. Crawler and recrawl jobs
      connect to the already-current schema and contain no migration or direct
      schema-repair path.
- [ ] Migration 019 constraints, append-only rules, redaction rule, composite
      tenant keys, one-charge/one-credit invariants, and cap failures pass.
- [ ] Migration 020 exact-organic-result and non-synthetic composite foreign
      keys, action allowlist, immutable confirmation, idempotency, expiry, and
      30-day cascade pass. The table has no query, contact, identity, free-form,
      provider-offer, ticket, budget, or outcome field.
- [ ] Migration 024 uses database-owned timestamps, serializes enrollment and
      ticket-cap decisions on the exact epoch, refuses enrollment outside draft,
      requires each enrolled claim's exact current non-spam site/domain to have
      appeared in a generation-1 organic result for the selected topic before
      the epoch cutoff, rechecks all enrollment bindings before activation, and
      makes offer, returned sidecar, and ticket epoch bindings immutable. The
      enrollment stores a database-owned site UUID snapshot, domain digest, and
      opaque eligibility digest; the eligibility portion of operator responses
      exposes only status and the opaque digest. Historical rows with a NULL
      epoch binding are ineligible for Stage 2 sidecars, tickets, or proof.
- [ ] Migration 025 leaves every pre-025 search, returned-result, selection,
      and action-interest row with a NULL `stage1_integrity_generation`. Those
      legacy caller-clock facts are quarantined and cannot enter Stage 1
      readiness. Every later insert receives generation 1 and its observation
      time from the database; those facts and the protected migration receipt
      are immutable. Pilot creation locks the exact 025 receipt and rejects an
      eligible cohort containing any fact without generation 1.
- [ ] Migration 026 first locks the pilot lifecycle relations and refuses to
      grandfather any epoch or enrollment missing its canonical `created`,
      `provider_enrolled`, `activated`, or `closed` event. Deferred constraints
      then require the matching event for every future lifecycle transition.
      Each pilot-bound outcome must match its exact ticket and, for positive
      states, its in-window handoff and still-current enrollment eligibility;
      its whole-second database-current clock and canonical JSON must match the
      immutable row. Negative and reversal outcomes remain available for
      economic cleanup after eligibility changes. Historical tickets with a
      NULL pilot binding remain excluded from pilot proof.
- [ ] Migration 027 records no query, search receipt, bearer/token hash,
      company-deduplication hash, principal or agent identity/contact/network
      metadata, signed-receipt body, signature, or free-form intent. Its trigger
      derives every claim/offer/ticket/handoff/callback binding from the exact
      subject and recomputes the canonical SHA-256 before insert. Reviews are
      immutable, exact replay is idempotent, provider candidates require fresh
      DNS ownership plus a current opaque enrollment-eligibility binding, and
      offer candidates require that binding plus a current owner-verified
      exact-terms commitment.
- [ ] Migration 028 permits at most one manifest for an exact closed pilot. Its
      candidate and signed payload contain aggregate counters, fixed-shape
      review coverage, a SHA-256 commitment to the exact sorted immutable review
      set, and controlled contract fields only. Every monetary array is present
      but empty and `monetary_amounts_withheld_for_privacy=true`; the private
      provider-count threshold cannot leak a one-provider currency bucket. The
      database refuses issuance if any enrollment binding is no longer current.
      The
      payload contains no provider, offer, ticket, handoff, callback, query,
      search-receipt, principal, or agent identifiers and no owner/evidence
      references. The database rejects extra keys, wrong JSON types, invalid
      aggregate relationships, or a review-root mismatch before the unique row
      can be consumed. It binds the canonical signed JSON, payload SHA-256,
      signature text, key ID, exact proof/review SHA-256 values, and database-
      owned issue time. The record is append-only, and startup retains
      verification material for every manifest key ID. The v1 HMAC is explicitly
      NHS-private and must never be described as independently verifiable.
- [ ] Dedicated `NHS_PROVIDER_EXCHANGE_SIGNING_KEY` and
      `NHS_PROVIDER_EXCHANGE_SIGNING_KEY_ID` references resolve without
      displaying their values. The optional
      `NHS_PROVIDER_EXCHANGE_PREVIOUS_SIGNING_KEYS_JSON` may be absent only for
      the first signing key.
- [ ] `NHS_PROVIDER_EXCHANGE_MODE` is explicitly set to exactly `pilot` for the
      commercial release. A missing or different value fails startup; there is
      no implicit commercial default. The exact same
      new-schema-compatible binary also passes a loopback recovery smoke with
      mode `disabled`, no signing material, free search and action-interest
      observation live, paid-offer sidecars suppressed, DNS reverification
      stopped, and every provider/ticket/handoff/commercial mutation returning
      private `503`.
- [ ] Inside migration 022's own transaction, after taking the same
      `ACCESS EXCLUSIVE` ticket-writer lock and before any 022 DDL, startup
      reconstructs or verifies a persisted proof sample for every existing key
      ID and signing domain. An empty store needs no signer; persisted proof
      requires compatible retained material regardless of the requested
      runtime mode. The pilot constructor checks again after migration.
      Material removed or replaced under a reused ID rolls the transaction
      back with zero 022 footprint. After 022 is already receipted, emergency
      `disabled` containment may boot without loading signing material because
      every receipt-verification and provider mutation route is closed; the
      retained key references remain mandatory and must pass the read-only
      retention preflight before `pilot` can be re-enabled.
- [ ] Stage 1 and the controlled pilot run on one application machine. Before
      horizontal scaling, replace the process-local provider, action-interest, and
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
      `converted` callbacks. Invoke it with the exact authorized revision:
      `tools/smoke-test.sh https://nothumansearch.ai <40-character-commit> pilot`.
- [ ] The full paid-flow smoke runs only against disposable PostgreSQL and a
      loopback server. It proves separate offers, consent rejection, exact
      ticket idempotency, absence of a provider URL in the ticket response,
      separate handoff-consent rejection, exact durable handoff idempotency,
      positive-outcome rejection before handoff, nonfinancial terminal
      rejection, signed outcome-receipt replay and tamper detection, and zero
      commercial-proof delta before destroying the fixture.

### No-write agent tool-choice evaluation

Before interpreting zero action-interest receipts as absent demand, validate
that a representative agent can distinguish explicit current principal intent
from research, ranking, selection, provider availability, future possibility,
and an explicit refusal. This is an instrumentation check, not demand evidence
and not permission to synthesize receipts.

`tools/action-interest-agent-eval.py` reads the exact public MCP
`record_action_interest` schema and runs six synthetic tool-choice scenarios
through the OpenAI Responses API. It never executes the returned function
call, never uses a live search receipt, retains neither response IDs nor raw
model text, and requires no provider or commercial mode. The grader requires
exact strict-schema arguments for two authorized scenarios, zero calls for four
non-authorized scenarios, and zero contact or identity leakage.

Validate the live MCP schema without an API charge or production write:

```sh
/usr/bin/python3 tools/action-interest-agent-eval.py --dry-run
```

Run the bounded behavioral evaluation only through the named secret bridge:

```sh
/Users/shane/.local/bin/codex-secret run \
  --env OPENAI_API_KEY=OPENAI_API_KEY \
  -- /usr/bin/python3 tools/action-interest-agent-eval.py \
  --model gpt-5.6-luna
```

The pass/fail receipt proves only that this model understood the consent rail
under controlled synthetic prompts. It is not a user, agent, demand, provider,
handoff, activation, revenue, or mechanism-selection receipt. Do not store the
API key in the repository, command arguments, report, or shell history, and do
not substitute an unregistered or stale credential when the named reference is
unavailable.

### Provider-funded price-floor hypothesis

Do not assign one arbitrary bounty to accepted, activated, and converted. Their
charge frequencies differ, so equal prices create different abilities to cover
the same fixed NHS infrastructure. Refresh the NHS-only Cost Explorer baseline,
preserve the source-period and non-invoice boundary, then run:

```sh
/usr/bin/python3 tools/provider-mechanism-price-floor.py
```

The standalone view answers what each event must gross if that mechanism, at
the declared event count, had to cover the whole observed NHS fixed cost. The
equal-arm view allocates the experimental fixed cost across all three arms.
Both use a conservative published percentage-plus-fixed processing allowance.
Neither is an offer, payment, revenue receipt, final invoice, full-cost model,
profit claim, or mechanism-selection result. Owner labor, support, fraud, tax,
and margin remain outside the baseline. Actual available processor-net
settlement receipts control the eventual selection.

Before any processor sandbox exercise, use an isolated Stripe test-mode secret;
never reuse the live production key. The bounded drill requires an explicit
`--confirm-test-mode-write`, reads `/v1/balance`, and refuses its first POST
unless Stripe returns `livemode=false`. It creates and refunds one 50-to-100
cent test PaymentIntent, reconciles the expanded charge balance transaction,
and emits no Stripe object IDs or customer/provider data:

```sh
/Users/shane/.local/bin/codex-secret run \
  --env STRIPE_SECRET_KEY=NHS_STRIPE_TEST_SECRET_KEY -- \
  /usr/bin/python3 tools/provider-stripe-test-mode-drill.py \
  --drill-id OWNER_APPROVED_UNIQUE_ID --amount-cents 50 \
  --confirm-test-mode-write
```

The test credential and test webhook secret must remain separate from
production. A passing receipt proves only the processor sandbox boundary; it is
not a provider payment, production settlement, commercial proof, or mechanism
selection.

### Mandatory one-way cutover order

The order below controls over the surrounding evidence checklist. Do not start
a later step from a green source receipt alone, and do not let server startup
reach migrations until steps 1 through 6 are complete:

1. With exact owner authorization, build and push the verified v2 source archive
   as the forward-fix image. Preserve its immutable OCI registry digest; inspect
   `org.opencontainers.image.revision` and require the authorized 40-character
   commit. `tools/build-exact-provider-image.sh` reconstructs and re-hashes the
   Git archive immediately before streaming that tar as the entire build context;
   its local receipt deliberately says `registry_digest_verified=false` and
   cannot authorize a deploy. Every later command uses the separately verified
   registry digest, never a mutable tag.
2. Before quiescence, prove the snapshot mechanism with a disposable restore
   drill and confirm cleanup. This validates the recovery procedure, but it is
   not the final zero-loss recovery point.
3. Re-run the live-pre-handoff read-only query. Stop on any nonzero row; never
   revoke or fabricate evidence merely to force the cutover.
4. Cordon the old application machine, disable autostart, stop every writer, and
   drain every client database session. Do not release or restart the old image.
5. After the drain, force the database's supported durable checkpoint and create
   the final pre-migration snapshot. Restore that exact snapshot to the
   disposable target and verify the bound database/schema receipt before any
   new binary can start. If the platform cannot prove this final restore, stop
   and record an explicit nonzero RPO/WAL recovery contract for owner review;
   never imply zero-loss rollback from the earlier snapshot.
6. Run the exact digest's revision-bound `provider-cutover-preflight` against the
   target database. It must report the same binary/candidate revision, the exact
   protected set through migration 032, zero other sessions, zero live tickets,
   valid signer retention for pilot mode, and `ready_for_quiesced_cutover=true`.
7. Start one machine from the exact digest with traffic held. This is the first
   step allowed to run the automatic protected migrations.
8. Before traffic release, verify `/health.release_revision`, the OCI label, and
   every migration-019-through-032 `applied_by_commit` value against the same
   authorized commit. Run the safe smoke in the chosen `disabled` or `pilot`
   mode. On failure, use the same digest in disabled containment or execute the
   tested database recovery plan; never roll the upgraded database back to the
   old writer.
9. Release traffic only after step 8 is green. Provider invitations, funding,
   offer activation, proof-manifest issuance, and publication remain separate
   owner-authorized actions.

The local v2 source receipt deliberately records
`oci_image_digest_verified=false`, `target_cutover_preflight_verified=false`,
`restore_drill_verified=false`, and `deployment_ready=false`. No current local
artifact may be substituted for those external receipts.

- [ ] Migration 021 is present before any pilot ticket is minted. Each live,
      uncharged ticket has one append-only capacity reservation; the configured
      charge consumes it, an uncharged terminal outcome releases it, and
      expired or emergency-revoked tickets stop consuming capacity logically.
- [ ] Migration 022 has not inferred or backfilled commercial proof from legacy
      accounts, active offers, evidence-reference strings, or budget rows. Every
      qualifying company has a provider-key-authenticated pilot acceptance and
      a separate owner-verified keyed company digest; every qualifying prepaid
      fund or exact-terms commitment has its own append-only evidence event, and
      accepted hash-bearing offer terms are database-immutable.
- [ ] Before quiescence, prove the snapshot/restore mechanism against a
      disposable target. After all writers and sessions are drained, create and
      restore-test a second, final pre-migration snapshot before running
      preflight or starting the candidate. The final receipt must bind source
      volume, snapshot ID/digest and creation time, restored volume/machine
      identity, PostgreSQL major version, database identity hash, schema-receipt
      inventory, drill start/end time, and cleanup outcome. Merely observing
      that automatic snapshots exist is not a tested recovery plan, and the
      earlier drill cannot prove zero-loss recovery for writes committed before
      drain. Snapshot creation, restore resources, and cleanup are separate
      owner-authorized external actions.

      Validate the completed external receipt locally before treating either
      restore gate as satisfied. This command is offline: it does not create,
      restore, or delete any Fly resource, and a passing result still reports
      `deployment_ready=false`:

      ```sh
      /usr/bin/python3 tools/provider-cutover-verify.py restore-receipt \
        --receipt EXACT_RESTORE_RECEIPT.json --revision COMMIT \
        --expected-source-app nothumansearch-db \
        --expected-source-volume-id SOURCE_VOLUME_ID \
        --expected-source-database-identity-sha256 SOURCE_DATABASE_SHA256 \
        --expected-snapshot-id SNAPSHOT_ID \
        --expected-snapshot-sha256 SNAPSHOT_SHA256 \
        --expected-snapshot-created-at SNAPSHOT_CREATED_AT \
        --expected-postgres-major 17
      ```

      The verifier requires the restored database identity to equal the bound
      source identity, requires a distinct disposable app/volume/machine, and
      requires cleanup of both restored resources. Automatic snapshot metadata
      alone cannot produce this receipt.
- [ ] Before migration 022, run the read-only cutover query below against the
      exact target database. The result must be zero. A nonzero result is a hard
      stop: complete, explicitly revoke, or allow those legacy authorizations to
      expire; do not backfill an observed handoff and do not discard a live
      principal authorization merely to force deployment.

      ```sql
      SELECT COUNT(*) AS live_pre_handoff_tickets
      FROM action_tickets
      WHERE status IN ('created','redirected','accepted','activated')
        AND expires_at > clock_timestamp()
        AND authorization_revoked_at IS NULL;
      ```

      Migration 022 enforces the same condition as
      `provider_action_handoff_cutover_inflight`, so a stale or skipped manual
      check still fails closed.
- [ ] Quiesce the old application revision, stop all ticket writers, and drain
      their database connections before applying migration 022. The migration
      takes an `ACCESS EXCLUSIVE` lock before repeating the zero-live-ticket
      check, then drops the two new ticket-snapshot defaults so an old binary
      cannot resume afterward and silently mint an empty-contract ticket. Do not
      release traffic until the exact new revision is healthy.
- [ ] After the drain and before target preflight, take the final durable
      checkpoint/snapshot and complete its disposable restore verification. Keep
      the old machine cordoned during this gate. If the final restore receipt is
      absent, migration startup is forbidden.
- [ ] Run the exact candidate's `provider-cutover-preflight --revision COMMIT
      --mode disabled` (or `pilot` for the separately authorized commercial
      release) against the target database after quiescing every other client
      backend. Preserve its bounded JSON receipt. A false
      `ready_for_quiesced_cutover` exits nonzero; every other client backend,
      including another candidate-tagged session or a nonempty foreign
      `application_name`, is blocking. The receipt is point-in-time evidence,
      not a lock: all old machines must remain cordoned with autostart disabled
      until the candidate owns the migration lock and becomes healthy.
- [ ] After the held candidate has applied migrations 019 through 032, verify
      their immutable ledger rows against the same exact 40-character revision.
      Inject the target URL from its canonical secret reference; never place it
      in argv. The verifier sets PostgreSQL read-only mode and emits bounded JSON
      with `deployment_ready=false`:

      ```sh
      /Users/shane/.local/bin/codex-secret run \
        --env DATABASE_URL=EXACT_TARGET_DATABASE_URL_REFERENCE \
        -- /usr/bin/python3 tools/provider-cutover-verify.py database \
        --revision COMMIT --migrations-dir "$PWD/migrations" \
        --confirm-read-only-database-check
      ```

      Require `protected_migration_count=14` and the exact migration names and
      require every ledger SHA-256 to match the exact local migration bytes.
      This is post-migration evidence only; it does not replace the
      quiesced preflight, private smoke, restore receipt, or traffic-release gate.
      If `psql` is not installed, the tool fails closed with `psql_unavailable`;
      `NHS_PSQL_BINARY` may identify a separately verified executable without
      placing the database URL in argv.
- [ ] Do not use the preparer's old default rolling-deploy pattern. Migration
      022 is a one-way writer cutover: the older binary rejects a database with
      the newer protected receipt and cannot mint the new snapshot fields. Have
      an exact forward-fix image ready, plus a tested database recovery plan,
      before quiescing the old revision. A normal rollback to the old image is
      not a valid recovery procedure after migration 022 commits.
      The verified forward containment is the same authorized image in
      `NHS_PROVIDER_EXCHANGE_MODE=disabled`; verify it with
      `tools/smoke-test.sh https://nothumansearch.ai <40-character-commit> disabled`.
      This containment mode is not a database rollback and does not erase any
      receipt; it stops new commercial exchange writes while preserving free
      discovery and Stage 1 evidence collection.
- [ ] Treat descriptive API/MCP release `1.1.0` and exact provider contract
      `nhs-action-ticket-preparation-v2` as an explicit controlled-pilot client
      cutover. Reverify every known ticket-preparation consumer against the new
      response (bearer plus handoff endpoint, no provider URL), separately test
      `nhs-provider-handoff-consent-v1`, and obtain owner release authorization.
      Zero in-flight rows proves no live authorization is stranded; it does not
      by itself prove that an old client understands the changed response.
      The `1.1.0` release field is not a semantic-compatibility promise for these
      pre-GA provider endpoints; their machine contract versions are authoritative.
- [ ] Ticket creation does not return the provider action URL. The dedicated
      REST and MCP handoff surfaces require the exact bearer plus
      `nhs-provider-handoff-consent-v1`; missing or false consent, an unknown
      version, or an expired/revoked ticket fails before any receipt or URL
      disclosure. The raw bearer is never stored as a ticket/handoff-row field,
      placed in an NHS URL, or returned in the public handoff receipt. The ticket
      nonce/key metadata and retained signing material can reconstruct it for an
      exact idempotent replay.
- [ ] Every qualifying handoff has one append-only `nhs-action-handoff-v1`
      receipt with the exact offer version, commercial-terms contract/hash,
      handoff consent/version, one-way presented-token hash, and observation
      time. It has no raw query, identity, contact, network, referrer, user-agent,
      or free-form intent column. Ticket creation alone and a direct canonical
      provider visit do not create this receipt.
- [ ] Every `fund` or `adjustment` row on a commercially eligible offer is linked
      to its exact verified funding or reversal event. A legacy/operator-only
      fund or adjustment contaminates and disqualifies the offer; it can never
      borrow credibility from one small verified payment. Positive provider
      callbacks recheck this evidence before recording an outcome or charge.
- [ ] Migration 022 tests prove that fund reversals remain recordable after
      ownership-proof staleness or revocation and remove reversed value, a
      delayed external funding event cannot masquerade as replenishment, and a
      terms renewal extends the same provider-authenticated exact-terms chain.
      They also prove that a positive provider outcome cannot precede the
      durable observed-handoff receipt, and that a receipt replay cannot create
      a second handoff.
- [ ] Migration 023 preserves every pre-existing handoff as an explicit
      controlled-intent disclosure decline. The consent boolean/version pair is
      database-enforced; replay cannot upgrade a decline. Offer version,
      commercial contract/hash, action type, signed authorization times, and
      controlled intent are immutable except for the exact one-way redaction;
      authorization revocation and negative outcome states cannot be cleared or
      reopened. Real-PostgreSQL tests prove resolver reads occur only after all
      authoritative rows are locked, including a row-lock wait that crosses the
      expiry boundary and returns no data.
- [ ] The resolver accepts only the exact signed attribution bearer and an
      active key for the same fresh DNS claim. It exposes no MCP tool, performs
      no write, returns only the allowlisted response schema, and returns no row
      after consent decline, token mismatch, cross-claim access, expiry,
      retention expiry, redaction, revocation, negative terminal state, stale
      ownership, or key rotation. Invalid random keys hit the pre-auth network
      limiter before a database key lookup.
- [ ] Ticket/resolver reads enforce the 30-day controlled-intent deadline
      independently of cleanup. Physical redaction runs immediately on each
      successful server start and hourly thereafter; downtime or cleanup
      failure may delay physical redaction but must never extend availability.
      Public retention and consent copy discloses that distinction.

## Owner-verified company deduplication

Company proof uses a stable keyed digest of an owner-verified authoritative
identifier. It never uses a company name, domain, email, provider claim, offer,
acceptance ID, or evidence reference as the identity input. The raw identifier
must never enter argv, environment variables, files, stdin pipes, clipboard,
logs, chat, evidence references, or the NHS database.

The v1 canonical contract is:

```text
company = legal or billing counterparty established by owner-held evidence
authority = lowercase ASCII source, for example lei or registry:us-wa:ubi
identifier = hidden authority-issued identifier
normalized identifier = trim ASCII whitespace, collapse internal ASCII
                        whitespace, lowercase ASCII, retain punctuation
message = "nhs-provider-company-dedup-v1" || NUL || authority || NUL || identifier
company_key_hash = lowercase hex(HMAC-SHA-256(key, message))
```

Reject an empty identifier, non-ASCII/control input, more than 200 input bytes,
an invalid authority, or a key other than exactly 32 random bytes encoded
as 64 lowercase/uppercase hexadecimal characters. The dedicated
`NHS_PROVIDER_COMPANY_DEDUP_KEY` belongs only in canonical 1Password plus its
Keychain bridge and is injected into the one operator subprocess. It must not be
added to Fly or the application environment. Do not rotate or delete it while
v1 company digests exist: the current schema has no key-version field, so
rotation first requires an owner-authorized versioned dual-key migration.

Required sequence:

1. The provider records its claim-scoped `pilot_company` acceptance.
2. The owner independently establishes the legal/billing counterparty and
   authoritative evidence.
3. The owner authorizes that exact company verification and its non-secret
   operator/evidence references.
4. The operator enters the authority-issued identifier only at the tool's
   echo-disabled `/dev/tty` prompt and runs:

   ```sh
   /Users/shane/.local/bin/codex-secret run \
     --env NHS_PROVIDER_COMPANY_DEDUP_KEY=NHS_PROVIDER_COMPANY_DEDUP_KEY \
     --env NHS_PROVIDER_OPERATOR_ADMIN_KEY=NHS_ADMIN_API_KEY \
     -- /usr/bin/python3 tools/provider-company-verify.py \
     --provider-acceptance-event-id ACCEPTANCE_UUID \
     --identity-authority lei \
     --operator-reference owner-case-2026-001 \
     --identity-evidence-reference identity-case-2026-001 \
     --confirm-owner-authorized
   ```

5. The tool performs one direct verified-HTTPS `verify_company` request with no
   proxy inheritance, redirects, automatic retry, raw response, or secret
   output. It prints only the allowlisted receipt fields.
6. Company verification remains identity evidence only. It is not funding,
   terms acceptance, an accepted handoff, activation, renewal, collection, or
   pilot proof by itself.

After an uncertain network result, an exact rerun is allowed only with the same
acceptance, authority, hidden identifier, and evidence references; the server
treats the exact event as an idempotent replay. A different authoritative
identifier or evidence record requires a new owner decision, not a guessed
substitute.

## Provider continuity and owner work queue

Do not use ad hoc `curl`, browser storage, `.env` files, chat, clipboard, or
manual SQL for the pilot control path. The returned-once provider key belongs in
the provider's approved secret manager; the raw value is never an argument and
NHS stores only its hash. The local reference name in the examples below is a
structured placeholder that must resolve through the approved secret bridge,
never a pasted value.

The provider can recover its own acceptance chain, activation readiness,
recent NHS-observed handoffs and callbacks, and privacy-thresholded aggregate
demand without choosing another claim or domain:

```sh
/Users/shane/.local/bin/codex-secret run \
  --env NHS_PROVIDER_PILOT_KEY=NHS_PROVIDER_PILOT_KEY \
  -- /usr/bin/python3 tools/provider-pilot-client.py status --limit 25

/Users/shane/.local/bin/codex-secret run \
  --env NHS_PROVIDER_PILOT_KEY=NHS_PROVIDER_PILOT_KEY \
  -- /usr/bin/python3 tools/provider-pilot-client.py demand --days 30
```

The demand command derives the domain from the authenticated claim. It has no
domain argument. Its counts are privacy-thresholded receipts, not unique agents
or principals, and it returns no raw query, identity, contact, network field,
credential, attribution material, action URL, or individual receipt.

Provider commercial mutations are real provider-party actions. Run each only
after the provider authorized that exact reference, offer version, terms hash,
and idempotency key:

```sh
/Users/shane/.local/bin/codex-secret run \
  --env NHS_PROVIDER_PILOT_KEY=NHS_PROVIDER_PILOT_KEY \
  -- /usr/bin/python3 tools/provider-pilot-client.py accept-company \
  --provider-acceptance-reference provider-company-acceptance-001 \
  --idempotency-key provider-company-event-001 \
  --confirm-provider-authorized

/Users/shane/.local/bin/codex-secret run \
  --env NHS_PROVIDER_PILOT_KEY=NHS_PROVIDER_PILOT_KEY \
  -- /usr/bin/python3 tools/provider-pilot-client.py accept-terms \
  --offer-id OFFER_UUID \
  --offer-version OFFER_VERSION \
  --exact-terms-sha256 EXACT_64_CHARACTER_SHA256 \
  --provider-acceptance-reference provider-terms-acceptance-001 \
  --idempotency-key provider-terms-event-001 \
  --confirm-provider-authorized

/Users/shane/.local/bin/codex-secret run \
  --env NHS_PROVIDER_PILOT_KEY=NHS_PROVIDER_PILOT_KEY \
  -- /usr/bin/python3 tools/provider-pilot-client.py renew-terms \
  --offer-id OFFER_UUID \
  --offer-version OFFER_VERSION \
  --exact-terms-sha256 EXACT_64_CHARACTER_SHA256 \
  --related-acceptance-event-id PRECEDING_ACCEPTANCE_UUID \
  --provider-acceptance-reference provider-terms-renewal-001 \
  --idempotency-key provider-renewal-event-001 \
  --confirm-provider-authorized
```

The owner discovers bounded pending work without SQL. The queue returns only
opaque workflow IDs, public domain, exact offer contract, states, and event
times; `returned_counts` counts only the bounded rows in that response, not the
unbounded total. Reading the queue does not verify evidence or authorize a
mutation:

```sh
/Users/shane/.local/bin/codex-secret run \
  --env NHS_PROVIDER_OPERATOR_ADMIN_KEY=NHS_ADMIN_API_KEY \
  -- /usr/bin/python3 tools/provider-pilot-operate.py queue \
  --state all --limit 25
```

After the owner independently reviews and authorizes the exact queue item, the
owner can join provider-authenticated terms to owner-held evidence and then
activate the exact eligible draft. Both commands require the explicit
authorization flag and emit only a bounded non-secret receipt:

```sh
/Users/shane/.local/bin/codex-secret run \
  --env NHS_PROVIDER_OPERATOR_ADMIN_KEY=NHS_ADMIN_API_KEY \
  -- /usr/bin/python3 tools/provider-pilot-operate.py verify-terms \
  --offer-id OFFER_UUID \
  --provider-acceptance-event-id ACCEPTANCE_UUID \
  --source-system owner-ledger \
  --source-event-id owner-terms-event-001 \
  --source-effective-at 2026-08-02T10:00:00Z \
  --operator-reference owner-terms-case-001 \
  --owner-evidence-reference owner-evidence-case-001 \
  --confirm-owner-authorized

# For an accepted terms_renewal only, add both of these exact queue values:
# --provider-acceptance-event-id RENEWAL_ACCEPTANCE_UUID
# --related-commitment-event-id PRECEDING_COMMITMENT_UUID

/Users/shane/.local/bin/codex-secret run \
  --env NHS_PROVIDER_OPERATOR_ADMIN_KEY=NHS_ADMIN_API_KEY \
  -- /usr/bin/python3 tools/provider-pilot-operate.py activate \
  --offer-id OFFER_UUID \
  --operator-reference owner-activation-case-001 \
  --evidence-reference owner-activation-evidence-001 \
  --confirm-owner-authorized
```

For a `pending_terms` item whose `acceptance_event_type` is
`terms_renewal`, the owner must pass the queue's exact
`related_commitment_event_id` to `verify-terms`. Do not substitute
`related_acceptance_event_id`: the provider uses the acceptance ID to extend
its authenticated chain, while the owner joins that renewal to the preceding
owner-verified commitment. The bounded owner receipt returns the newly recorded
`commitment_event_id` so the chain remains recoverable without manual SQL.

Use the same owner client with `pause` and exact references for an authorized
emergency stop. Activation is operational state, not commercial proof. An
unexplained legacy fund or adjustment makes both provider status and the owner
activation queue fail closed rather than label the draft ready.

For a separately consented handoff, `resolve` reads the attribution bearer from
an echo-disabled TTY or bounded stdin, never argv. `outcome` uses the same
bearer rule and requires an exact idempotency key plus
`--confirm-provider-authorized`. The signed bearer is
authoritative; `--ticket-id` is only an optional matching compatibility
assertion. Preserve the returned receipt ID in the provider's normal records;
if the callback response is lost, `status` recovers the bounded recent event and
`receipt --receipt-id RECEIPT_UUID` retrieves the provider-owned signed receipt.
Never reconstruct a bearer, acceptance ID, or receipt ID from logs or memory.

## Stage 1: observe free discovery

Run free search for at least 14 days before choosing the pilot category. Select
one category from persisted, non-synthetic, controlled-topic demand—not index
size, `initialize`, `tools/list`, denied calls, raw prompts, pre-cutover
receipts, total matches on an empty result page, or the catch-all `other`
topic. The protected migration-025 receipt is the trusted Stage 1 epoch. Only
generation-1 search, returned-result, selection, and action-interest facts
inserted under migration 025's database-owned clocks may enter readiness;
pre-025 rows remain quarantined even if their caller-supplied timestamps fall
inside the measured window. Both a search receipt and every returned result
used to make that search meaningful, validate its selection, or establish
candidate-topic breadth must fall inside the same Stage 1 window and evidence
cutoff; a result attached to an older receipt after the cutoff never counts.

An eligible result-producing MCP discovery receipt comes only from
`search_agents`, `find_mcp_servers`, or a category-bound `get_top_sites` or
`recent_additions` call whose caller supplied a recognized public category.
Unfiltered `get_top_sites` and `recent_additions` calls are generic catalog
browsing and must not receive an eligible Stage 1 receipt. Handshake and
metadata calls, including `initialize`, `tools/list`, `get_stats`, and
`list_categories`, are also excluded. Never infer a demand topic from the
categories of providers returned by a call: only the caller's recognized
category or the existing controlled, in-memory query classification may
produce a controlled aggregate topic. Provider-funded sidecars remain a
separate field attached after organic results are recorded; they cannot alter
organic inclusion or order.

Decision evidence:

- at least 100 non-synthetic meaningful discovery receipts;
- at least 20 persisted result selections on distinct non-synthetic search
  receipts; and
- at least 10 explicit `nhs-action-interest-v1` receipts on distinct
  non-synthetic search receipts for a quote, trial, demo, booking, application,
  signup, or purchase; and
- at least one non-`other` controlled topic represented by 20 distinct eligible
  search receipts whose generation-1 organic results collectively include at
  least 10 distinct exact, currently indexed non-spam domains, so the owner has
  a feasible and privacy-safe candidate category to select.

These are receipt counts, not unique agents, principals, provider contacts,
accepted handoffs, revenue, or commercial proof. An action-interest receipt
does not contact the provider; it only proves that NHS recorded the caller's
versioned attestation of principal interest against an exact organic result.
The 10-domain feasibility test is internal. The owner report continues to show
only the candidate topic and its receipt count; it never returns the qualifying
domain count, list, individual receipt, rank, or domain. The owner-only Stage 1
report must show the protected epoch, an exact measured
activity span of at least 14 x 86,400 seconds, all three distinct-receipt
targets, and at least one candidate topic before `stage1_ready` can be true.
Readiness does not select a category,
authorize provider invitations, or start Stage 2. Read it without placing the
admin key in argv or logs:

```sh
/Users/shane/.local/bin/codex-secret run \
  --env NHS_PROVIDER_OPERATOR_ADMIN_KEY=NHS_ADMIN_API_KEY \
  -- /usr/bin/python3 tools/provider-pilot-status.py \
  --scope stage1 --stage1-days 30
```

## Stage 2: bounded provider pilot

Invite 3 to 20 providers in the one selected category only after owner release.
The three-provider minimum is deliberately the smallest cohort that can prove
the stated first-commercial milestone; it is an existence test, not evidence
of broad market repeatability. A later pilot may use a larger cohort. This
commercial cohort minimum does not replace or reduce the separate Stage 1
requirement for at least 10 distinct non-spam organic domains.
The owner must first authorize one exact draft epoch from a `stage1_ready=true`
snapshot. The selected topic, cohort limit, per-provider ticket cap, total ticket
cap, Stage 1 timestamps/hash, and evidence references become immutable. Only one
draft or active epoch may exist. Enrollment is accepted only while draft and
only when the exact fresh verified claim site/domain is currently non-spam and
was actually returned by a non-synthetic generation-1 search for the selected
topic inside that epoch's Stage 1 window and evidence cutoff. PostgreSQL binds
that fact to the epoch/company/claim/site with a database-owned domain digest
and opaque eligibility digest. The enrollment response returns `eligible` and
that digest, not the site UUID, domain digest, raw domain, receipt, rank, domain
count, or domain list. Later Stage 1 retention cleanup does not invalidate the
stored historical bind and later gates do not reread expired raw receipts;
current claim/site/domain drift or a spam reclassification does invalidate
activation, sidecars, tickets, handoffs, positive outcomes, reviews, and proof.
Negative or reversal outcomes remain usable for economic cleanup.
Activation requires 3 through the configured limit with every claim still
fresh, and closure immediately stops new paid sidecars, tickets, and handoffs.
Idempotent ticket replay does not consume another cap slot. Migration 026 makes
each epoch and enrollment state inseparable from its canonical append-only
lifecycle event and rejects pilot-bound outcome proof that is not tied to the
exact in-window ticket and required handoff. Migration 027 makes each explicit
owner review an append-only receipt bound to the canonical digest of one exact
provider, offer, ticket, handoff, or callback snapshot. A review receipt is a
safety/audit fact only; it cannot create or replace provider acceptance,
commercial terms, a handoff, a callback, revenue, or 3/5/2/1 proof.
Migration 028 can issue one immutable signed aggregate only after the exact pilot
is closed and every required chronological review remains valid against its
current subject snapshot.

Migration 027 also makes chronology an execution gate. Pilot activation fails
until every enrolled provider has a current review; offer activation fails
until that exact offer has a current review; and handoff observation fails until
the exact ticket has a current review. A premature REST handoff returns the
machine-readable `review_pending` state with no handoff receipt, provider URL,
or charge. This is a bounded pilot safety hold, not a search paywall or a paid
event.

Use only the fixed-host operator client. These commands mutate production and
therefore require the separate, exact owner authorization represented by the
confirmation flag; this runbook is not that authorization:

```sh
/Users/shane/.local/bin/codex-secret run \
  --env NHS_PROVIDER_OPERATOR_ADMIN_KEY=NHS_ADMIN_API_KEY \
  -- /usr/bin/python3 tools/provider-pilot-operate.py authorize-pilot \
  --topic SELECTED_TOPIC --cohort-limit 3 \
  --provider-ticket-cap 5 --total-ticket-cap 5 \
  --owner-reference OWNER_REFERENCE --evidence-reference EVIDENCE_REFERENCE \
  --confirm-owner-authorized

/Users/shane/.local/bin/codex-secret run \
  --env NHS_PROVIDER_OPERATOR_ADMIN_KEY=NHS_ADMIN_API_KEY \
  -- /usr/bin/python3 tools/provider-pilot-operate.py enroll-pilot \
  --pilot-id PILOT_UUID --claim-id CLAIM_UUID \
  --owner-reference OWNER_REFERENCE --evidence-reference EVIDENCE_REFERENCE \
  --confirm-owner-authorized

/Users/shane/.local/bin/codex-secret run \
  --env NHS_PROVIDER_OPERATOR_ADMIN_KEY=NHS_ADMIN_API_KEY \
  -- /usr/bin/python3 tools/provider-pilot-operate.py status-pilot \
  --pilot-id PILOT_UUID
```

For every required review, first fetch the bounded candidate and inspect it.
The read returns the current `subject_snapshot_sha256`; it does not record that
a review happened. Only after the owner authorizes that exact snapshot may the
separate mutation echo the digest and non-secret references:

```sh
/Users/shane/.local/bin/codex-secret run \
  --env NHS_PROVIDER_OPERATOR_ADMIN_KEY=NHS_ADMIN_API_KEY \
  -- /usr/bin/python3 tools/provider-pilot-operate.py queue \
  --state provider_review_required

/Users/shane/.local/bin/codex-secret run \
  --env NHS_PROVIDER_OPERATOR_ADMIN_KEY=NHS_ADMIN_API_KEY \
  -- /usr/bin/python3 tools/provider-pilot-operate.py review-candidate \
  --pilot-id PILOT_UUID --review-type REVIEW_TYPE --subject-id SUBJECT_UUID

/Users/shane/.local/bin/codex-secret run \
  --env NHS_PROVIDER_OPERATOR_ADMIN_KEY=NHS_ADMIN_API_KEY \
  -- /usr/bin/python3 tools/provider-pilot-operate.py record-review \
  --pilot-id PILOT_UUID --review-type REVIEW_TYPE --subject-id SUBJECT_UUID \
  --expected-snapshot-sha256 EXACT_CANDIDATE_SHA256 \
  --owner-reference OWNER_REVIEW_REFERENCE \
  --evidence-reference OWNER_REVIEW_EVIDENCE_REFERENCE \
  --confirm-owner-authorized
```

Only after `provider_review_required` is empty may the separately authorized
pilot activation run:

```sh
/Users/shane/.local/bin/codex-secret run \
  --env NHS_PROVIDER_OPERATOR_ADMIN_KEY=NHS_ADMIN_API_KEY \
  -- /usr/bin/python3 tools/provider-pilot-operate.py activate-pilot \
  --pilot-id PILOT_UUID --owner-reference OWNER_REFERENCE \
  --evidence-reference EVIDENCE_REFERENCE --confirm-owner-authorized
```

Use `queue --state offer_review_required` before each offer activation and
`queue --state ticket_review_required` before retrying a `review_pending`
handoff. `review-preflight` is a read-only combined view of all three pre-event
queues; it does not record approval. After events, use
`handoff_review_required` and `callback_review_required` to complete the exact
post-event reviews required by the proof manifest.

`REVIEW_TYPE` and `SUBJECT_UUID` must be one exact pair: `provider` plus claim
ID, `offer` plus offer ID, `ticket` plus action-ticket ID, `handoff` plus
handoff-receipt ID, or `callback` plus outcome-receipt ID. Never join IDs from
different queue items. A changed subject returns a conflict and requires a new
read and owner decision; it must not be auto-retried as approval of new facts.
The candidate includes the provider offer/action details and bounded consented
intent needed for review, but no raw search query/receipt, bearer/token hash,
company-deduplication hash, principal or agent identity/contact/network
metadata, raw signed receipt/signature, or free-form intent.

Every provider must:

1. control an indexed domain and keep the returned DNS TXT proof published;
2. record the claim-scoped `pilot_company` acceptance, after which the owner
   independently verifies the legal/billing counterparty with the HMAC workflow;
3. use a same-domain HTTPS action URL;
4. state the action, principal price, NHS bounty, charge event, refund/credit
   rule, response expectation, first-activation period-anchor rule, and
   provider Merchant-of-Record acknowledgement exactly;
5. accept exact capped CPA terms with non-secret owner-held evidence; prepaid
   funding is not launched in this pilot;
6. save its returned-once callback key;
7. understand that optional controlled-intent resolution requires the exact
   third principal consent and never grants outreach authority; and
8. accept that organic rank and readiness score cannot be bought.

Record each qualifying provider review no later than pilot activation, each
qualifying offer review no later than that offer's activation, and each
qualifying ticket review no later than its handoff observation. Review every
qualifying handoff no earlier than its observation time and every qualifying
callback no earlier than its outcome-receipt creation time. Preserve these
database-owned review times; a retrospective review
cannot be represented as a pre-event safety check. Start with the smallest
credible paid event, normally an NHS-observed handoff followed by a
provider-authenticated acceptance, or a real activation—not an impression, API
call, direct canonical visit, bare redirect, or model score.

## Truthful proof gate

The pilot is commercially proven only when the production evidence tables and
owner-held external receipts jointly establish all of the following:

- 3 externally deduplicated provider companies accepted provider-key-
  authenticated exact capped CPA terms, with separate owner-held identity and
  commercial evidence;
- 5 non-synthetic handoffs with both versioned attestations were first observed
  by NHS and then accepted by providers;
- 2 non-synthetic activations were reported by providers;
- 1 provider extended its exact authenticated terms-renewal chain. Prepaid
  replenishment is not launched and cannot satisfy this terms-only pilot; and
- any claimed collection is backed by an external payment/ledger receipt, not
  merely an internal debit or receivable.

An issuable private proof manifest must also bind the exact migration-027 review
receipts for every qualifying provider, offer, ticket, handoff, and callback. It
must reject provider reviews recorded after epoch activation, offer reviews
recorded after offer activation, and ticket reviews recorded after handoff
observation. Handoff reviews must be recorded at or after handoff observation;
callback reviews must be recorded at or after callback creation. Review coverage
does not increment any commercial counter; missing or late safety review keeps
the manifest unavailable even if the underlying aggregate counters reach
3/5/2/1. Until migration 029 is deployed and this exact candidate passes, the
aggregate proof response is diagnostic evidence, not a public commercial-
proof artifact.

Provider callbacks are authenticated assertions. NHS-signed receipts prove what
NHS recorded; neither fact independently audits the underlying business event.
An active offer, nonempty evidence-reference string, internal ledger fund, or
internal receipt timestamp is not independently a qualifying commitment.
Any unlinked legacy fund or adjustment makes the affected offer ineligible
rather than increasing its verified capacity.

The owner-only proof response must use the verified maps and counters by their
exact names. The 3/5/2/1 gate is
`verified_provider_companies`, `verified_provider_accepted_handoffs`,
`verified_provider_confirmed_activations`, and `verified_provider_renewals`.
`verified_provider_accepted_handoffs` requires the exact durable handoff receipt
and a later provider-authenticated positive state under the matching current
commercial snapshot; ticket creation or provider callback alone is insufficient.
Every qualifying outcome is reverified against the retained NHS signing keyring
and every signed ID, event, state, amount, currency, and timestamp must equal its
immutable database row. Its accounting must also match exactly: one matching
charge or credit ledger entry for a charged or credited receipt, and no outcome
ledger entry for a receipt whose charge status is `none`. Verification is
bidirectional: every charge or credit attached to an otherwise qualifying pilot
ticket must also map back to one authenticated outcome receipt. The response
must report `outcome_receipt_integrity_valid=true`,
`rejected_outcome_receipts=0`, `rejected_outcome_ledger_entries=0`, and the
bounded `verified_outcome_receipts` and
`verified_outcome_ledger_entries` counts. Any rejected receipt or unmatched
ledger entry forces `pilot_thresholds_met=false`; an operator must investigate
it rather than silently excluding the contaminated row. A credit reverses the
ticket's positive commercial state even when the credit itself is unsigned and
therefore also rejected as proof.
The live pilot money view is `verified_terms_net_receivable_by_currency`.
`verified_prepaid_settled_by_currency` and
`verified_prepaid_net_debited_by_currency` must remain zero while prepaid is
disabled. The legacy
`operator_recorded_*` and `provider_reported_*` fields remain diagnostics and
cannot satisfy `pilot_thresholds_met`.

Read the redacted aggregate gate without exposing the admin key:

```sh
/Users/shane/.local/bin/codex-secret run \
  --env NHS_PROVIDER_OPERATOR_ADMIN_KEY=NHS_ADMIN_API_KEY \
  -- /usr/bin/python3 tools/provider-pilot-status.py --scope proof \
  --pilot-id PILOT_UUID
```

The migration-028 manifest is a separate, chronological owner gate. Complete
the event-relative reviews above, finish the bounded observation window, and
close the exact pilot under separate owner authorization before previewing a
manifest candidate:

```sh
/Users/shane/.local/bin/codex-secret run \
  --env NHS_PROVIDER_OPERATOR_ADMIN_KEY=NHS_ADMIN_API_KEY \
  -- /usr/bin/python3 tools/provider-pilot-operate.py close-pilot \
  --pilot-id PILOT_UUID --owner-reference OWNER_CLOSE_REFERENCE \
  --evidence-reference OWNER_CLOSE_EVIDENCE_REFERENCE \
  --confirm-owner-authorized

/Users/shane/.local/bin/codex-secret run \
  --env NHS_PROVIDER_OPERATOR_ADMIN_KEY=NHS_ADMIN_API_KEY \
  -- /usr/bin/python3 tools/provider-pilot-status.py --scope proof-manifest \
  --pilot-id PILOT_UUID
```

The preview is read-only: it does not sign, store, publish, or create commercial
proof. Require `issued=false`, `commercial_proof_created=false`,
`publicly_released=false`, `independently_verifiable=false`, the exact pilot ID,
`pilot_status=closed`, outcome integrity, 3/5/2/1, and `valid == required` for
providers, offers, tickets, handoffs, and callbacks. The privacy-safe
`review_evidence_sha256` commits the sorted immutable review IDs, subject
snapshot digests, authorization references, database review times, and
chronology decisions without disclosing the underlying row IDs or references.
`review_integrity_valid` and `issuable` must both be true,
`issuance_blockers` must be empty, and the owner must review and authorize
the candidate's exact `proof_snapshot_sha256`. Never infer approval from the
earlier aggregate proof response.

Only after that exact candidate receives a separate issuance authorization may
the owner run the mutation below. `OWNER_MANIFEST_REFERENCE` and
`OWNER_MANIFEST_EVIDENCE_REFERENCE` are bounded, non-secret references, not
evidence bodies or credentials:

```sh
/Users/shane/.local/bin/codex-secret run \
  --env NHS_PROVIDER_OPERATOR_ADMIN_KEY=NHS_ADMIN_API_KEY \
  -- /usr/bin/python3 tools/provider-pilot-operate.py issue-proof-manifest \
  --pilot-id PILOT_UUID \
  --expected-snapshot-sha256 EXACT_PREVIEW_PROOF_SNAPSHOT_SHA256 \
  --owner-reference OWNER_MANIFEST_REFERENCE \
  --evidence-reference OWNER_MANIFEST_EVIDENCE_REFERENCE \
  --confirm-owner-authorized
```

Issuance re-evaluates the aggregate and all chronological reviews in one
serializable transaction. Snapshot drift or a failed gate returns a conflict;
fetch and review a new candidate instead of substituting its new digest or
automatically retrying. An exact retry must use the same pilot ID, digest, owner
reference, and evidence reference and returns the original immutable manifest;
different issuance evidence for the same pilot is a conflict.

Read the stored result back with the same `--scope proof-manifest` preview
command. Require `issued=true`, `commercial_proof_created=true`,
`publicly_released=false`, `independently_verifiable=false`, the exact pilot,
snapshot digest, and review-evidence digest. Preserve the manifest ID, payload
digest, signed manifest, signature, key ID, and issue time as the private owner
receipt. Exact monetary buckets are deliberately empty and
`monetary_amounts_withheld_for_privacy=true`; with only three companies, a
currency/funding-mode total could expose one provider. The versioned
`nhs-free-organic-provider-funded-v1` field is an NHS policy attestation for the
three no-sale booleans, not an independent behavioral audit.

Issuance is not publication and grants no public-account action: publication
requires another exact owner authorization and a separate release path. This v1
artifact uses an NHS-secret HMAC and is verifiable only against the retained NHS
private keyring. It must not be described as independently or publicly
verifiable. Public release remains blocked until a separately reviewed
asymmetric signing/public-key history (preferred) or an explicitly weaker
NHS-hosted verification design exists. The private signature authenticates the
privacy-redacted aggregate NHS recorded; it is not independent provider truth,
proof of cash collection, proof of the underlying business event, or proof of
any principal or agent identity.

## Stop conditions

Pause the pilot and do not broaden inventory if any of these occurs:

- providers consume reports but none will accept exact capped CPA terms;
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
