# Score-Fix Private Cleanup Retry - 2026-05-31

WorkItem: `work_machine_05b0a8e8b65d0c13`
Automation: `business-agent-not-human-search`
Timestamp: `2026-05-31T03:11:37Z`

## Required pre-read

Completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- `harness/work_machine_3cb7432451fd76f7.md`
- `harness/work_machine_454002e6e1ed2d10.md`

## Helper execution

Command:

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

## Latest aggregate planning proof

The freshest available aggregate remains the planner-provided source for this work item:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age buckets: 1 in `7_29d`, 3 in `30d_plus`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

## Decision

- Keep the private score-fix cleanup lane open as `credential_required` for executor runtime.
- External `real_candidate` pending rows stay untouched.
- The prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` continues to block another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Verification

```sh
python3 -m json.tool harness/generated-work-items.json >/dev/null
GOCACHE=/private/tmp/nothumansearch-gocache go test ./...
```

Result:

- `harness/generated-work-items.json` is valid JSON.
- `go test ./...` passed with the writable Go cache path.

## Commit status

The repo-local proof was written and `harness/generated-work-items.json` was refreshed in the worktree, but committing is blocked in this executor because Git cannot create `.git/index.lock`:

```text
fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted
```

Required git-writable follow-up:

- Commit `harness/work_machine_05b0a8e8b65d0c13.md`.
- Commit the refreshed `harness/generated-work-items.json` after clearing the assume-unchanged index flag if needed.
