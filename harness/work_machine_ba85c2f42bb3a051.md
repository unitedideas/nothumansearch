# QLimit score-fix private cleanup closeout

WorkItem: `work_machine_ba85c2f42bb3a051`
Timestamp: `2026-06-06T06:10:38Z`

Private score-fix cleanup retry for pending GEO fix rows. Committed proof is aggregate-only; no row ids, hostnames, emails, notes, Stripe ids, checkout URLs, payment identifiers, customer identifiers, or raw admin rows are recorded here.

Required pre-read completed before any private score-fix mutation:

- Re-read `tools/geo-jobs-redacted-read.sh`.
- Re-read `ops/ledgers/score-fix-pending-followup-2026-05-12.md`.
- Re-read `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`.

Fresh helper execution:

```sh
./tools/geo-jobs-redacted-read.sh
```

Result:

- The helper failed closed before fetching admin rows: `missing Keychain service: nhs-admin-api-key nothumansearch-admin-key`.
- No raw admin rows were fetched.
- No private score-fix mutation was attempted.
- No customer-visible score-fix email was sent.
- No public-action lock was created or reused.
- No external customer row was mutated.

Latest aggregate proof remains the WorkItem-provided aggregate from `2026-05-21T11:07Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`, age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`, age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`, age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`, age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`, age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due.
- A credential-capable executor may classify or clean up only `test_like pending` rows through the private admin workflow after the redacted helper succeeds.

Verification:

```sh
python3 -m json.tool harness/generated-work-items.json >/dev/null
```

`harness/generated-work-items.json` already contains the credential-required follow-up lane, so no duplicate follow-up row was added.
