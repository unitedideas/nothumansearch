# QLimit score-fix cleanup retry

WorkItem: `work_machine_ce8388b63752a5b3`
UTC: `2026-05-27T20:11:26Z`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- `harness/work_machine_0288ea9945bc8692.md` was requested by the work item but is not present in this worktree.

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-27T20:10:32Z`:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

Verification:

- `python3 -m json.tool harness/generated-work-items.json >/dev/null`: passed.
- `python3 tools/test-redact-geo-jobs.py`: passed.
- `GOCACHE=/private/tmp/nothumansearch-go-build go test ./...`: failed in pre-existing `cmd/monitor-check` compile drift: undefined `firstCheckFailedQuarantineReason`, `firstCheckZeroScoreQuarantineReason`, and `firstCheckQuarantineReason`.

Commit status:

- Commit is blocked in this executor because git cannot create `.git/index.lock`: `Operation not permitted`.
- `harness/generated-work-items.json` and `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md` are still marked assume-unchanged in this checkout because clearing those bits also requires `.git/index.lock`.
- A git-writable executor should clear assume-unchanged for those two paths, stage them with this file, and commit the score-fix closeout.
