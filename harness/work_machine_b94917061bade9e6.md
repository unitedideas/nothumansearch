# NHS Score-Fix Private Cleanup Closeout - 2026-06-06

WorkItem: `work_machine_b94917061bade9e6`
Automation: `business-agent-not-human-search`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

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

Latest aggregate proof remains the WorkItem-provided aggregate from `2026-05-22T08:08Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`, age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`, age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`, age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`, age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`, age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep external `real_candidate` pending rows untouched. The prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` blocks another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due.
- Keep the private cleanup lane as `credential_required` until an executor runtime can read `nhs-admin-api-key` or `nothumansearch-admin-key` through the repo helper.
- The next credential-capable executor may classify or clean up only `test_like pending` rows through the private admin workflow and must keep committed proof aggregate-only by class, age bucket, and host class.

Commit note:

- This worker could not stage or commit because `.git/index.lock` creation returned `Operation not permitted`.
