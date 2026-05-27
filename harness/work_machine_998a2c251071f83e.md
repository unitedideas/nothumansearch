# Work machine 998a2c251071f83e

Date: 2026-05-27T09:10:48Z
Business: nothumansearch
Lane: score-fix private cleanup

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- `harness/work_machine_21a187c343189f15.md`

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

Planner-provided aggregate proof from 2026-05-26T16:08:24Z remains the latest aggregate evidence available to this executor:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Verification:

- `python3 tools/test-redact-geo-jobs.py`: passed.
- `GOCACHE=/private/tmp/nothumansearch-go-cache go test ./internal/... ./cmd/server ./cmd/crawler`: passed.
- `GOCACHE=/private/tmp/nothumansearch-go-cache go test ./...`: failed on existing `cmd/monitor-check` test drift (`firstCheckFailedQuarantineReason`, `firstCheckZeroScoreQuarantineReason`, and `firstCheckQuarantineReason` undefined).

Commit status:

- `git add harness/work_machine_998a2c251071f83e.md harness/generated-work-items.json` failed because this runtime cannot create `.git/index.lock`: `Operation not permitted`.
- `harness/generated-work-items.json` was updated in the worktree to point the score-fix follow-up to this proof file, but Git also has that file marked assume-unchanged and clearing the flag failed on the same `.git/index.lock` permission blocker.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.
