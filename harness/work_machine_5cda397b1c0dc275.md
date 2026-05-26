# Work machine 5cda397b1c0dc275

Date: 2026-05-26T18:11:28Z
Business: nothumansearch
Lane: score-fix private cleanup

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

Executor result:

- `./tools/geo-jobs-redacted-read.sh` failed closed with `missing Keychain service: nhs-admin-api-key nothumansearch-admin-key`.
- No admin rows were fetched.
- No score-fix mutation was attempted.
- No customer-visible score-fix email was sent.
- No public-action lock was created or reused.
- No external customer row was mutated.

Latest aggregate proof retained from the planner handoff:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets 1 `1_6d`, 2 `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- Run this lane from an executor where `nhs-admin-api-key` or `nothumansearch-admin-key` is available to `tools/geo-jobs-redacted-read.sh`.
- Classify or clean up only `test_like pending` rows through the private admin workflow.
- Do not send another score-fix customer email for the already-contacted external cohort without a fresh duplicate check and public-action lock.
