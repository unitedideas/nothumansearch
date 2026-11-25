# WorkItem work_machine_8e09d039c0e9ff4e

Date: 2026-06-03
Business: nothumansearch

Required pre-read completed before any score-fix state change:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- Attempted to re-read `harness/work_machine_0288ea9945bc8692.md`; it is not present in this worktree. Prior score-fix ledgers already record that missing-file condition.

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
| `real_candidate` | `pending` | `dot_com` | `lt_1d` | 1 |
| `test_like` | `pending` | `dot_com` | `7_29d` | 4 |
| `test_like` | `lead` | `dot_com` | `30d_plus` | 1 |
| `test_like` | `paid` | `dot_com` | `30d_plus` | 2 |
| `test_like` | `internal_test` | `foundry_owned` | `7_29d` | 2 |

Decision:

- Customer-visible score-fix follow-up due now: 0.
- External `real_candidate` pending rows stay untouched.
- The already-contacted external pending cohort remains under prior follow-up proof and cannot receive another customer-visible score-fix email without a future duplicate check plus fresh public-action lock.
- Cleanup remains `credential_required`; a credential-capable executor may classify or clean up only `test_like pending` rows through the private admin workflow after `tools/geo-jobs-redacted-read.sh` succeeds.
