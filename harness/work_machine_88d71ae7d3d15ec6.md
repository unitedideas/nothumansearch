# WorkItem work_machine_88d71ae7d3d15ec6

Date: 2026-05-27T05:10:14Z
Business: nothumansearch

## Required pre-read

- Re-read `tools/geo-jobs-redacted-read.sh`.
- Re-read `ops/ledgers/score-fix-pending-followup-2026-05-12.md`.
- Re-read `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`.
- Attempted to re-read `harness/work_machine_0288ea9945bc8692.md`; it is not present in this worktree.

## Helper execution

Command:

```sh
./tools/geo-jobs-redacted-read.sh
```

Result:

- Failed closed before any admin rows were fetched: `missing Keychain service: nhs-admin-api-key nothumansearch-admin-key`.
- No private score-fix mutation was attempted.
- No customer-visible score-fix email was sent.
- No public-action lock was created or reused.
- No external customer row was mutated.

## Aggregate-only state

Latest planner-provided aggregate proof from this WorkItem:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

## Decision

The score-fix cleanup lane remains `credential_required`. A future executor must first run `tools/geo-jobs-redacted-read.sh` successfully with `nhs-admin-api-key` or `nothumansearch-admin-key` available, then may classify or clean up only `test_like pending` rows through the private admin workflow.

External `real_candidate` pending rows stay untouched. The existing follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.

## Commit blocker

Attempted to stage the local proof/state update with:

```sh
git add -f harness/work_machine_88d71ae7d3d15ec6.md harness/generated-work-items.json ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md
```

Result:

- `fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted`

The repo-local files were updated, but this worker could not create a commit because `.git` metadata writes are blocked in the executor runtime.
