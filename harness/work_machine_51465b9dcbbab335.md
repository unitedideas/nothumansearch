# WorkItem `work_machine_51465b9dcbbab335`

Date: 2026-05-31T17:09:59Z
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

Latest aggregate proof remains the planner-provided aggregate from 2026-05-31T17:09:59Z:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`.
- `test_like pending`: 4 `dot_com`.
- `test_like lead`: 1 `dot_com`.
- `test_like paid`: 2 `dot_com`.
- `test_like internal_test`: 2 `foundry_owned`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the external `real_candidate pending` rows untouched. The two external pending rows already have follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md`.
- Keep the private cleanup lane open as `credential_required` until an executor can run `tools/geo-jobs-redacted-read.sh` successfully with `nhs-admin-api-key` or `nothumansearch-admin-key`.
- A future executor may classify or clean up only `test_like pending` rows through the private admin workflow and must keep committed proof aggregate-only.
