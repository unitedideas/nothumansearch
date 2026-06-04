# Work machine 8cf828cc0518686b

Date: 2026-06-03T22:10:15Z
Business: `nothumansearch`

## Scope

Private score-fix cleanup retry for pending GEO fix rows. Committed proof is aggregate-only; no row ids, hosts, emails, notes, Stripe ids, checkout URLs, payment identifiers, customer identifiers, or raw admin rows are recorded here.

## Required Pre-Read

- Re-read `tools/geo-jobs-redacted-read.sh`.
- Re-read `ops/ledgers/score-fix-pending-followup-2026-05-12.md`.
- Re-read `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`.
- Attempted to re-read `harness/work_machine_0288ea9945bc8692.md`; it is not present in this worktree.

## Helper Result

Command:

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

## Aggregate Proof

Latest aggregate proof remains the planner-provided aggregate from `2026-06-03T22:10:15Z`:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

## Decision

The score-fix cleanup lane remains `credential_required`. External `real_candidate` pending rows stay untouched. The prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.

## Verification

Command:

```sh
GOCACHE=/private/tmp/nothumansearch-gocache go test ./...
```

Result: passed.

## Commit Blocker

Commit was attempted but this worker runtime cannot create Git index locks:

```sh
git add harness/work_machine_8cf828cc0518686b.md harness/generated-work-items.json ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md
```

Result: `fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted`.

The worktree changes are left in place for a git-writable executor to commit.
