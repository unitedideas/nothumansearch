# WorkItem work_machine_13c360d48537a68d

Date: 2026-05-26T11:16:00Z
Business: nothumansearch

## Required pre-read

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

## Execution

Command:

```sh
./tools/geo-jobs-redacted-read.sh
```

Result:

- Failed closed before any admin rows were fetched.
- Missing Keychain services in this worker runtime: `nhs-admin-api-key`, `nothumansearch-admin-key`.
- No score-fix private mutation was attempted.
- No customer-visible score-fix email was sent.
- No public-action lock was created or reused.
- No external customer row was mutated.

## Aggregate proof boundary

Latest aggregate-only proof is from the WorkItem intake:

- Total score-fix rows: 12.
- Real-candidate pending rows: 3, host class `dot_com`.
- Real-candidate pending age buckets: 1 in `1_6d`, 2 in `7_29d`.
- Test-like pending rows: 4, host class `dot_com`, age bucket `7_29d`.
- Test-like lead rows: 1, host class `dot_com`, age bucket `30d_plus`.
- Test-like paid rows: 2, host class `dot_com`, age bucket `30d_plus`.
- Test-like internal-test rows: 2, host class `foundry_owned`, age bucket `7_29d`.

## Decision

- Customer follow-up due now: 0.
- The already-contacted external pending cohort remains blocked from another customer-visible email unless a future duplicate-ledger review and fresh public-action lock prove a new touch is due.
- Remaining cleanup is credential-gated private admin work limited to `test_like` pending rows.
