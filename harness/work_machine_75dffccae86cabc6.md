# QLimit score-fix private cleanup closeout

WorkItem: `work_machine_75dffccae86cabc6`
Date: 2026-06-02T21:14:42Z

Required pre-read completed before any score-fix state change:

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

Aggregate-only proof available to this executor from the WorkItem:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `7_29d`.
- `test_like paid`: 2 `dot_com`; age bucket `7_29d`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision remains `credential_required`: external `real_candidate` pending rows stay untouched; the already-contacted external pending cohort must not receive another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due. The next executor may classify or clean up only `test_like pending` rows through the private admin workflow after `tools/geo-jobs-redacted-read.sh` succeeds.

Commit blocker:

- `git update-index` and commit preparation failed because Git could not create `.git/index.lock`: `Operation not permitted`.
- The ledger and generated-work-item files were updated in the worktree, but both are currently hidden from normal `git status` by index flags in this runtime.
