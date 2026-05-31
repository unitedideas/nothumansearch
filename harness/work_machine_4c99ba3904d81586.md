# QLimit score-fix cleanup retry

Date: 2026-05-31T09:09:52Z
WorkItem: `work_machine_4c99ba3904d81586`
Business: `nothumansearch`
Lane: score-fix private cleanup

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

Fresh executor helper result:

```sh
./tools/geo-jobs-redacted-read.sh
```

- The helper failed closed before fetching admin rows: `missing Keychain service: nhs-admin-api-key nothumansearch-admin-key`.
- No raw admin rows were fetched.
- No private score-fix mutation was attempted.
- No customer-visible score-fix email was sent.
- No public-action lock was created or reused.
- No external customer row was mutated.

Latest aggregate proof retained from the planner handoff:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`.
- `test_like pending`: 4 `dot_com`.
- `test_like lead`: 1 `dot_com`.
- `test_like paid`: 2 `dot_com`.
- `test_like internal_test`: 2 `foundry_owned`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the private score-fix cleanup lane open as `credential_required` for this executor runtime.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.
- `harness/generated-work-items.json` already carries the credential-gated private score-fix cleanup follow-up, so no duplicate generated WorkItem was added.

Git closeout:

- `git update-index --no-assume-unchanged harness/generated-work-items.json ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md ops/ledgers/score-fix-pending-followup-2026-05-12.md` failed because `.git/index.lock` cannot be created in this sandbox.
- A git-writable executor should stage this file and any ledger/generated-work-item changes, then commit the score-fix closeout.
