# NHS score-fix private cleanup closeout - work_machine_88ebe484f12b3331

Timestamp: 2026-06-03T18:11:16Z

Required pre-read completed before any score-fix state change:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

Helper execution:

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

Aggregate-only proof from the WorkItem originating ref:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep `harness/generated-work-items.json` on the existing credential-gated private cleanup follow-up rather than adding a duplicate row.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- A future credential-capable executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

Verification:

- `python3 -m json.tool harness/generated-work-items.json`
- `python3 tools/test-redact-geo-jobs.py`
- `GOCACHE=/private/tmp/nothumansearch-go-cache go test ./...`

Commit blocker:

- `git update-index --no-assume-unchanged harness/generated-work-items.json ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md ops/ledgers/score-fix-pending-followup-2026-05-12.md` failed with `.git/index.lock: Operation not permitted`.
