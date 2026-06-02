# Work Machine 73d1be0b5c01420f

Date: 2026-06-02T18:10:31Z
Business: nothumansearch
Automation: business-agent-not-human-search

## Required Pre-Read

Completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

Not available in this worktree:

- `harness/work_machine_0288ea9945bc8692.md`

## Helper Result

Command:

```sh
./tools/geo-jobs-redacted-read.sh
```

Result:

- Failed closed before admin fetch: `missing Keychain service: nhs-admin-api-key nothumansearch-admin-key`.
- No raw admin rows were fetched.
- No private score-fix mutation was attempted.
- No customer-visible score-fix email was sent.
- No public-action lock was created or reused.
- No external customer row was mutated.

## Aggregate-Only State

Latest aggregate proof remains the planner-provided aggregate from 2026-06-02T18:09:32Z:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

## Decision

The cleanup lane remains `credential_required`.

External `real_candidate` pending rows stay untouched. The prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.

The next executor must have `nhs-admin-api-key` or `nothumansearch-admin-key` available, run `tools/geo-jobs-redacted-read.sh` successfully, and classify or clean up only `test_like pending` rows through the private admin workflow.
