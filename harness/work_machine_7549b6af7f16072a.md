# QLimit score-fix cleanup retry

WorkItem: `work_machine_7549b6af7f16072a`
UTC: `2026-06-02T23:10:32Z`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- `harness/work_machine_0288ea9945bc8692.md` was requested by the work item but is not present in this checkout.

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

Latest aggregate proof remains the planner-provided aggregate from `2026-06-02T23:09:35Z`:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 2 in `7_29d`, 1 in `lt_1d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.
- This executor could not clear assume-unchanged flags or commit because git cannot create `.git/index.lock`: `Operation not permitted`.
