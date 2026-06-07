# Score-fix private cleanup closeout - 2026-06-07T17:11:01Z

WorkItem: `work_machine_ee744944470948cf`
Automation: `business-agent-not-human-search`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- `harness/work_machine_0288ea9945bc8692.md` was requested by the work item but is not present in this checkout.

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

Latest aggregate proof remains the WorkItem-provided aggregate from `2026-05-24T09:08:12Z`:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 2 `7_29d`, 1 `lt_1d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep score-fix cleanup open as `credential_required` for an executor runtime that can read `nhs-admin-api-key` or `nothumansearch-admin-key` through the repo helper.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due.
- The current private admin action model only supports `mark_internal_test` on Foundry-owned pending rows. Any cleanup of non-Foundry `test_like pending` rows needs a repo-supported private admin workflow that proves test-like eligibility without committing raw row data.
