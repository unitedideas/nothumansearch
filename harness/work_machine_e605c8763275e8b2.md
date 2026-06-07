# WorkItem work_machine_e605c8763275e8b2

Date: 2026-06-07T09:11:19Z
Business: nothumansearch
Automation: business-agent-not-human-search

Required pre-read completed before any score-fix state change:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

Fresh helper execution:

```sh
./tools/geo-jobs-redacted-read.sh
```

Result:

- Failed closed before fetching admin rows: `missing Keychain service: nhs-admin-api-key nothumansearch-admin-key`.
- No raw admin rows were fetched.
- No private score-fix mutation was attempted.
- No customer-visible score-fix email was sent.
- No public-action lock was created or reused.
- No external customer row was mutated.

Aggregate-only proof available to this worker from the WorkItem source:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `7_29d`.
- `test_like paid`: 2 `dot_com`; age bucket `7_29d`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep external `real_candidate pending` rows untouched.
- Keep customer-visible score-fix follow-up blocked unless a future duplicate check plus fresh public-action lock prove a new touch is due.
- Keep the private cleanup lane as `credential_required` for a credential-capable executor that can classify only `test_like pending` rows.
