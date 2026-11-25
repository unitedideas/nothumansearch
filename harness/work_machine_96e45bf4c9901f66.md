# WorkItem `work_machine_96e45bf4c9901f66`

Objective: reconcile NHS score-fix private cleanup without another customer-visible touch.

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

Aggregate-only proof from the WorkItem planner evidence:

| class | status | host_class | age_bucket | count |
|---|---|---|---|---:|
| `real_candidate` | `pending` | `dot_com` | `7_29d` | 2 |
| `test_like` | `pending` | `dot_com` | `7_29d` | 4 |
| `test_like` | `lead` | `dot_com` | `30d_plus` | 1 |
| `test_like` | `paid` | `dot_com` | `30d_plus` | 2 |
| `test_like` | `internal_test` | `foundry_owned` | `7_29d` | 2 |

Decision:

- Customer-visible score-fix follow-up due now: 0.
- External `real_candidate` pending rows stay untouched; the external pending cohort already has follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md`.
- No second customer-visible score-fix email is due unless a future duplicate check and fresh public-action lock prove it.
- The cleanup lane remains `credential_required` until an executor can run `tools/geo-jobs-redacted-read.sh` successfully.
- A credential-capable executor may classify or clean up only `test_like pending` rows through the private admin workflow.
