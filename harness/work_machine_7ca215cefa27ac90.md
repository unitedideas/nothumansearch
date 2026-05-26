# WorkItem work_machine_7ca215cefa27ac90

Date: 2026-05-26T20:09:50Z
Automation: `business-agent-not-human-search`

## Required pre-read

Completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- `harness/work_machine_0288ea9945bc8692.md` was requested by the WorkItem but is not present in this worktree.

## Fresh helper execution

Command:

```sh
./tools/geo-jobs-redacted-read.sh
```

Result:

- The helper failed closed before fetching admin rows: `missing Keychain service: nhs-admin-api-key nothumansearch-admin-key`.
- No raw admin rows were fetched.
- No private score-fix mutation was attempted.
- No customer-visible email was sent.
- No public-action lock was created or reused.
- No external customer row was mutated.

Latest aggregate proof remains the planner-provided aggregate from 2026-05-26T20:09:50Z:

- Total score-fix rows: 12.
- Real-candidate pending rows: 3, host class `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- Test-like pending rows: 4, host class `dot_com`, age bucket `7_29d`.
- Test-like lead rows: 1, host class `dot_com`, age bucket `30d_plus`.
- Test-like paid rows: 2, host class `dot_com`, age bucket `30d_plus`.
- Test-like internal-test rows: 2, host class `foundry_owned`, age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

## Decision

- Keep the score-fix cleanup lane open as `credential_required`.
- External real-candidate pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run the redacted helper successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.
