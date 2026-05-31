# WorkItem work_machine_50f99e33530b8abf

Date: 2026-05-31T15:10:21Z
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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-18T21:08Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; visible age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; visible age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; visible age bucket `7_29d`.
- `test_like paid`: 2 `dot_com`; visible age bucket `7_29d`.
- `test_like internal_test`: 2 `foundry_owned`; visible age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- A future executor may classify or clean up only `test_like pending` rows through the private admin workflow after a successful redacted helper read.

Commit status:

- Commit is blocked in this runtime because Git cannot create `.git/index.lock`.
- The existing tracked files `harness/generated-work-items.json` and `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md` are marked assume-unchanged in the index, and clearing that flag also requires writing `.git/index.lock`.
