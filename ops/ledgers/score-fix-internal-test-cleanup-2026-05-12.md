# Score-Fix Internal Test Cleanup - 2026-05-12

Automation: `business-agent-not-human-search`

Action: used the private admin action `POST /api/v1/admin/geo-jobs/action` with `action=mark_internal_test` to reclassify one Foundry-owned pending score-fix row as internal test data.

Boundary: aggregate-only. No row ids, hosts, emails, notes, Stripe ids, or raw admin rows are committed here.

Before:

```json
{
  "summary": [
    {"class": "real_candidate", "count": 3, "status": "pending"},
    {"class": "test_like", "count": 1, "status": "lead"},
    {"class": "test_like", "count": 2, "status": "paid"},
    {"class": "test_like", "count": 5, "status": "pending"}
  ],
  "by_status_host_class": [
    {"class": "real_candidate", "count": 2, "host_class": "dot_com", "status": "pending"},
    {"class": "real_candidate", "count": 1, "host_class": "foundry_owned", "status": "pending"},
    {"class": "test_like", "count": 1, "host_class": "dot_com", "status": "lead"},
    {"class": "test_like", "count": 2, "host_class": "dot_com", "status": "paid"},
    {"class": "test_like", "count": 4, "host_class": "dot_com", "status": "pending"},
    {"class": "test_like", "count": 1, "host_class": "foundry_owned", "status": "pending"}
  ]
}
```

After:

```json
{
  "summary": [
    {"class": "real_candidate", "count": 2, "status": "pending"},
    {"class": "test_like", "count": 1, "status": "internal_test"},
    {"class": "test_like", "count": 1, "status": "lead"},
    {"class": "test_like", "count": 2, "status": "paid"},
    {"class": "test_like", "count": 5, "status": "pending"}
  ],
  "by_status_host_class": [
    {"class": "real_candidate", "count": 2, "host_class": "dot_com", "status": "pending"},
    {"class": "test_like", "count": 1, "host_class": "foundry_owned", "status": "internal_test"},
    {"class": "test_like", "count": 1, "host_class": "dot_com", "status": "lead"},
    {"class": "test_like", "count": 2, "host_class": "dot_com", "status": "paid"},
    {"class": "test_like", "count": 4, "host_class": "dot_com", "status": "pending"},
    {"class": "test_like", "count": 1, "host_class": "foundry_owned", "status": "pending"}
  ]
}
```

Proof commands:

```sh
NHS_GEO_JOBS_LIMIT=500 ./tools/geo-jobs-redacted-read.sh
python3 tools/geo-jobs-mark-internal-test.py
```

Remaining actionable score-fix queue: two real-candidate external `dot_com` pending rows. Customer follow-up, if sent later, must use the email-outreach public-action lock path and commit only aggregate counts plus message ids.

## Aggregate closeout retry - 2026-05-13

Automation: `business-agent-not-human-search`

Required pre-read completed before this update:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

Credential path in this worker runtime:

- `nhs-admin-api-key`: `SET`
- `nothumansearch-admin-key`: `SET`

Command:

```sh
./tools/geo-jobs-redacted-read.sh
```

Fresh aggregate-only result:

- Total score-fix rows: 11
- Real-candidate pending rows: 2, host class `dot_com`
- Test-like internal-test rows: 2, host class `foundry_owned`
- Test-like pending rows: 4, host class `dot_com`
- Test-like lead rows: 1, host class `dot_com`
- Test-like paid rows: 2, host class `dot_com`

Decision:

- The Foundry-owned pending cleanup lane is closed; no Foundry-owned pending row remains.
- The two external pending rows already have follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md`.
- Customer-visible score-fix follow-up due now: 0.
- No customer-visible email was sent, no public-action lock was created or reused, and no external customer row was mutated.

## Credential-blocked retry - 2026-05-26

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_21a187c343189f15`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- `harness/work_machine_0288ea9945bc8692.md` was requested by the work item but is not present in this worktree.

Fresh helper execution:

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

Latest aggregate proof remains the planner-provided aggregate from 2026-05-26T13:08:24Z:

- Total score-fix rows: 12.
- Real-candidate pending rows: 3, host class `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- Test-like pending rows: 4, host class `dot_com`, age bucket `7_29d`.
- Test-like lead rows: 1, host class `dot_com`, age bucket `30d_plus`.
- Test-like paid rows: 2, host class `dot_com`, age bucket `30d_plus`.
- Test-like internal-test rows: 2, host class `foundry_owned`, age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep external real-candidate pending rows untouched; the prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- Keep the cleanup lane as `credential_required` until an executor with `nhs-admin-api-key` or `nothumansearch-admin-key` available can run the redacted helper successfully and classify only test-like pending rows through the private admin workflow.

## Credential-blocked executor retry - 2026-05-26

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_2756000bf088589b`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

Fresh helper execution:

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

Latest aggregate proof remains the planner-provided aggregate from 2026-05-26T03:08:34Z:

- Total score-fix rows: 12.
- Real-candidate pending rows: 3, host class `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- Test-like pending rows: 4, host class `dot_com`, age bucket `7_29d`.
- Test-like lead rows: 1, host class `dot_com`, age bucket `30d_plus`.
- Test-like paid rows: 2, host class `dot_com`, age bucket `30d_plus`.
- Test-like internal-test rows: 2, host class `foundry_owned`, age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External real-candidate pending rows stay untouched; the prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run the redacted helper successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-05-26T23:10:44Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_82ca41d17e53a004`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

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

Latest aggregate proof remains the planner-provided aggregate from 2026-05-26T23:09:59Z:

- Total score-fix rows: 12.
- Real-candidate pending rows: 3, host class `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- Test-like pending rows: 4, host class `dot_com`, age bucket `7_29d`.
- Test-like lead rows: 1, host class `dot_com`, age bucket `30d_plus`.
- Test-like paid rows: 2, host class `dot_com`, age bucket `30d_plus`.
- Test-like internal-test rows: 2, host class `foundry_owned`, age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External real-candidate pending rows stay untouched; the prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run the redacted helper successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-05-26T20:09:50Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_7ca215cefa27ac90`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- `harness/work_machine_0288ea9945bc8692.md` was requested by the WorkItem but is not present in this worktree.

Fresh helper execution:

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

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External real-candidate pending rows stay untouched; the prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run the redacted helper successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-05-26T18:11:28Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_5cda397b1c0dc275`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

Fresh helper execution:

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

Latest aggregate proof remains the planner-provided aggregate from 2026-05-26T18:10:30Z:

- Total score-fix rows: 12.
- Real-candidate pending rows: 3, host class `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- Test-like pending rows: 4, host class `dot_com`, age bucket `7_29d`.
- Test-like lead rows: 1, host class `dot_com`, age bucket `30d_plus`.
- Test-like paid rows: 2, host class `dot_com`, age bucket `30d_plus`.
- Test-like internal-test rows: 2, host class `foundry_owned`, age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External real-candidate pending rows stay untouched; the prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run the redacted helper successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.
