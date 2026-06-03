# WorkItem work_machine_805f1de42d933237

Date: 2026-06-03T09:10:27Z
Business: nothumansearch

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

Aggregate-only proof from planner input:

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
- `harness/generated-work-items.json` keeps the credential-gated private score-fix cleanup follow-up rather than adding a duplicate lane.

Verification:

- `python3 -m json.tool harness/generated-work-items.json >/dev/null`: passed.
- `python3 tools/test-redact-geo-jobs.py`: passed.
- `GOCACHE=/private/tmp/nothumansearch-go-cache go test ./...`: passed.

Commit status:

- `git update-index --no-assume-unchanged harness/generated-work-items.json ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md` failed because this executor cannot create `.git/index.lock`: `Operation not permitted`.
- `git add harness/work_machine_805f1de42d933237.md ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md harness/generated-work-items.json && git commit -m "Record score-fix credential-required proof"` failed for the same `.git/index.lock` permission blocker.
