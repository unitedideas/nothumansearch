# Score-fix cleanup credential-required closeout

Date: 2026-06-08T05:10:53Z
WorkItem: `work_machine_f67d14c4db36b42e`

Required pre-read completed before any score-fix state change:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

Fresh helper execution failed closed before fetching admin rows:

```sh
./tools/geo-jobs-redacted-read.sh
```

Result:

- `missing Keychain service: nhs-admin-api-key nothumansearch-admin-key`
- No raw admin rows fetched.
- No private score-fix mutation attempted.
- No customer-visible score-fix email sent.
- No public-action lock created or reused.
- No external customer row mutated.

Aggregate-only proof from the WorkItem-provided helper result at `2026-05-20T09:08Z`:

- Total score-fix rows: 11.
- By class and host class: `real_candidate pending` 2 `dot_com`; `test_like pending` 4 `dot_com`; `test_like lead` 1 `dot_com`; `test_like paid` 2 `dot_com`; `test_like internal_test` 2 `foundry_owned`.
- By class and age bucket: `real_candidate pending` `7_29d`; `test_like pending` `7_29d`; `test_like lead` `30d_plus`; `test_like paid` `30d_plus`; `test_like internal_test` `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- External `real_candidate pending` rows remain untouched.
- The already-contacted external pending cohort remains blocked from another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove a new touch is due.
- The private cleanup lane remains `credential_required`; a credential-capable executor must classify or clean up only `test_like pending` rows through a repo-supported private admin workflow.

Commit status:

- `git update-index --no-assume-unchanged harness/generated-work-items.json ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md` failed with `.git/index.lock: Operation not permitted`.
- Commit creation is blocked in this runner until Git metadata writes are available.
