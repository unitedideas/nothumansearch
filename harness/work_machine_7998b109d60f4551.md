# WorkItem `work_machine_7998b109d60f4551`

Date: 2026-06-03T03:11:18Z

Scope: NHS score-fix private cleanup retry.

Required pre-read completed before any score-fix state change:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

Fresh helper execution:

```sh
./tools/geo-jobs-redacted-read.sh
```

Result:

- Helper failed closed before fetching admin rows: `missing Keychain service: nhs-admin-api-key nothumansearch-admin-key`.
- No raw admin rows were fetched.
- No private score-fix mutation was attempted.
- No customer-visible score-fix email was sent.
- No public-action lock was created or reused.
- No external customer row was mutated.

Aggregate-only proof carried from the planner-provided read at 2026-05-19T12:08Z:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `7_29d`.
- `test_like paid`: 2 `dot_com`; age bucket `7_29d`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the lane `credential_required`.
- External pending rows remain untouched under the prior follow-up proof.
- Only `test_like pending` rows may be classified or cleaned up, and only from a credential-capable executor through the private admin workflow.
