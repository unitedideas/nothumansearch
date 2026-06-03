# WorkItem work_machine_800eb0b312e595ab

Date: 2026-06-03T07:11:02Z
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
| `test_like` | `lead` | `dot_com` | `7_29d` | 1 |
| `test_like` | `paid` | `dot_com` | `7_29d` | 2 |
| `test_like` | `internal_test` | `foundry_owned` | `7_29d` | 2 |

Decision:

- Customer-visible score-fix follow-up due now: 0.
- The already-contacted external pending cohort remains under prior follow-up proof and cannot receive another customer-visible score-fix email without a future duplicate check plus fresh public-action lock.
- Cleanup remains `credential_required`; a credential-capable executor may classify or clean up only `test_like pending` rows through the private admin workflow.
- `harness/generated-work-items.json` already carries the credential-gated private score-fix cleanup follow-up, so no duplicate generated WorkItem was added.
