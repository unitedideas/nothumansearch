# WorkItem work_machine_8412a19b598aefbb

Date: 2026-06-03T11:10:45Z
Business: nothumansearch

Required pre-read completed before any score-fix state change:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- `harness/work_machine_0288ea9945bc8692.md` was requested by the WorkItem but is absent in this worktree.

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

Latest aggregate proof remains the planner-provided aggregate from 2026-05-23T09:08:09Z:

| class | status | host class | age bucket | count |
|---|---|---|---|---:|
| `real_candidate` | `pending` | `dot_com` | `7_29d` | 2 |
| `test_like` | `pending` | `dot_com` | `7_29d` | 4 |
| `test_like` | `lead` | `dot_com` | `30d_plus` | 1 |
| `test_like` | `paid` | `dot_com` | `30d_plus` | 2 |
| `test_like` | `internal_test` | `foundry_owned` | `7_29d` | 2 |

Decision:

- Customer-visible score-fix follow-up due now: 0.
- The already-contacted external pending cohort remains under prior follow-up proof and cannot receive another customer-visible score-fix email without a future duplicate check plus fresh public-action lock.
- Cleanup remains `credential_required`; a credential-capable executor may classify or clean up only `test_like pending` rows through the private admin workflow.
- `harness/generated-work-items.json` keeps a single credential-gated private score-fix cleanup follow-up rather than adding duplicate lanes.

Verification:

- `python3 -m json.tool harness/generated-work-items.json >/dev/null`: passed.
- `python3 tools/test-redact-geo-jobs.py`: passed.
- `GOCACHE=/private/tmp/nothumansearch-go-cache go test ./...`: passed.

Commit status:

- `git update-index --no-assume-unchanged ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md harness/generated-work-items.json` failed because this executor cannot create `.git/index.lock`: `Operation not permitted`.
- `git add harness/work_machine_8412a19b598aefbb.md && git commit -m "Record score-fix credential blocker"` failed for the same `.git/index.lock` permission blocker.
