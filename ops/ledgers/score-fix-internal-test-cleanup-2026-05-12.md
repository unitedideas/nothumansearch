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

## Credential-blocked executor retry - 2026-06-05T18:10:47Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_b1bb4630a1ddc761`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- `ops/ledgers/work_machine_afd0f95ec4033bed.md`

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

Latest aggregate proof remains the WorkItem-provided aggregate from `2026-05-20T15:08Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due.

## Credential-blocked executor retry - 2026-06-07T09:11:19Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_e605c8763275e8b2`

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

Latest aggregate proof remains the WorkItem-provided aggregate from `2026-06-07T09:10:24Z`:

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

## Credential-blocked executor retry - 2026-06-07T07:10:40Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_e5bfcf3b1328c89e`

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

Latest aggregate proof remains the WorkItem-provided aggregate from `2026-05-18T15:08Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `7_29d`.
- `test_like paid`: 2 `dot_com`; age bucket `7_29d`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due.
- Future private cleanup must classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked QLimit retry - 2026-06-07T03:11:28Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_da1ef890ac56571f`

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

Latest aggregate proof remains the WorkItem-provided planner aggregate from `2026-06-07T03:10:32Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`.
- `test_like pending`: 4 `dot_com`.
- `test_like lead`: 1 `dot_com`.
- `test_like paid`: 2 `dot_com`.
- `test_like internal_test`: 2 `foundry_owned`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the private score-fix cleanup lane open as `credential_required` for an executor where `tools/geo-jobs-redacted-read.sh` can read one of the NHS admin Keychain aliases.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due.
- The next credential-capable executor may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked aggregate closeout - 2026-06-06T23:11:01Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_cf9c40b3758caf74`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

Fresh helper execution:

```sh
NHS_GEO_JOBS_LIMIT=500 ./tools/geo-jobs-redacted-read.sh
```

Result:

- The helper failed closed before fetching admin rows: `missing Keychain service: nhs-admin-api-key nothumansearch-admin-key`.
- No raw admin rows were fetched.
- No private score-fix mutation was attempted.
- No customer-visible score-fix email was sent.
- No public-action lock was created or reused.
- No external customer row was mutated.

Latest aggregate proof remains the WorkItem-provided planner aggregate from 2026-06-06T23:11:01Z:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`.
- `test_like pending`: 4 `dot_com`.
- `test_like lead`: 1 `dot_com`.
- `test_like paid`: 2 `dot_com`.
- `test_like internal_test`: 2 `foundry_owned`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due.
- Only `test_like pending` rows are eligible for a future private admin cleanup pass.

## Credential-blocked executor retry - 2026-06-06T19:10:45Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_c8b082b3e6c8c649`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

Requested pre-read that is absent from this checkout:

- `harness/work_machine_0288ea9945bc8692.md`

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

Latest aggregate proof remains the WorkItem-provided aggregate from `2026-05-23T22:08:03Z`:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 2 in `7_29d`, 1 in `lt_1d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due.
- A future credential-capable executor should classify or clean up only `test_like pending` rows through the private admin workflow, with committed proof limited to class, status, age bucket, and host class.

## Credential-blocked executor retry - 2026-06-06T04:15:23Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_b94917061bade9e6`

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

Latest aggregate proof remains the WorkItem-provided aggregate from `2026-05-22T08:08Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`, age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`, age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`, age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`, age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`, age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.

## Credential-blocked executor retry - 2026-06-06T06:10:38Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_ba85c2f42bb3a051`

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

Latest aggregate proof remains the WorkItem-provided aggregate from `2026-05-21T11:07Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`, age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`, age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`, age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`, age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`, age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due.
- A credential-capable executor may classify or clean up only `test_like pending` rows through the private admin workflow after the redacted helper succeeds.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due.

## Credential-blocked executor retry - 2026-06-06T04:10:59Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_b7e89e60d41756d9`

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

Latest aggregate proof remains the WorkItem-provided aggregate from `2026-06-06T04:10:59Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`.
- `test_like pending`: 4 `dot_com`.
- `test_like lead`: 1 `dot_com`.
- `test_like paid`: 2 `dot_com`.
- `test_like internal_test`: 2 `foundry_owned`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due.
- A credential-capable executor may classify or clean up only `test_like pending` rows through the private admin workflow and must keep committed proof aggregate-only.

## Credential-blocked executor retry - 2026-06-06T00:10:56Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_b4d131351e27451b`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- attempted `harness/work_machine_0288ea9945bc8692.md`; it is not present in this worktree, and prior score-fix ledgers already record the same missing-file condition.

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

Latest aggregate proof remains the WorkItem-provided aggregate from `2026-05-24T13:08:26Z`:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due.
- A future credential-capable executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-06-03T05:09:52Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_7f077f45c2bb6830`

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-20T00:08Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `7_29d`.
- `test_like paid`: 2 `dot_com`; age bucket `7_29d`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-06-05T06:10:39Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_a263c6219f9fd446`

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

Latest aggregate proof remains the planner-provided aggregate for this WorkItem:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`.
- `test_like pending`: 4 `dot_com`.
- `test_like lead`: 1 `dot_com`.
- `test_like paid`: 2 `dot_com`.
- `test_like internal_test`: 2 `foundry_owned`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- This executor cannot classify the `test_like pending` `dot_com` cohort because the private admin credential path is unavailable here. A future credential-capable executor may classify or clean up only `test_like pending` rows through the private admin workflow and must keep committed proof aggregate-only.

## Credential-blocked executor retry - 2026-06-05T00:12:02Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_9d43cf53693420d1`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

The WorkItem also required `harness/work_machine_0288ea9945bc8692.md`, but that file is not present in this worktree. The available `harness/work_machine_*.md` files were checked and no matching path exists.

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

Latest aggregate proof remains the planner-provided aggregate from `2026-06-05T00:10:57Z`:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 rows; age buckets: 2 in `7_29d`, 1 in `lt_1d`.
- `test_like pending`: 4 rows.
- `test_like lead`: 1 row.
- `test_like paid`: 2 rows.
- `test_like internal_test`: 2 rows.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-06-04T22:09:43Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_9a70f11587fd758f`

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
- No customer-visible score-fix email was sent.
- No public-action lock was created or reused.
- No external customer row was mutated.

Latest aggregate proof remains the planner-provided aggregate from the WorkItem:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- `test_like pending`: 4 `dot_com`, age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`, age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`, age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`, age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-06-04T12:09:45Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_96e45bf4c9901f66`

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

Aggregate-only proof from the WorkItem planner evidence:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep external `real_candidate` pending rows untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- Keep the score-fix cleanup lane open as `credential_required`.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-06-04T10:10:22Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_910f84697d4a31b7`

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

Latest aggregate-only proof remains the planner-provided aggregate from `2026-06-04T10:10:22Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; visible age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; visible age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; visible age bucket `7_29d`.
- `test_like paid`: 2 `dot_com`; visible age bucket `7_29d`.
- `test_like internal_test`: 2 `foundry_owned`; visible age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep external `real_candidate` pending rows untouched; prior follow-up proof still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove a new touch is due.
- Keep the private cleanup lane open as `credential_required`.
- A future credential-capable executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-06-04T08:11:31Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_8fcba3cb50f588f2`

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-21T15:07Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-06-03T22:10:15Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_8cf828cc0518686b`

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
- No customer-visible score-fix email was sent.
- No public-action lock was created or reused.
- No external customer row was mutated.

Latest aggregate proof remains the planner-provided aggregate from `2026-06-03T22:10:15Z`:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-06-03T11:10:45Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_8412a19b598aefbb`

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
- No customer-visible score-fix email was sent.
- No public-action lock was created or reused.
- No external customer row was mutated.

Latest aggregate proof remains the planner-provided aggregate from `2026-05-23T09:08:09Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- A future credential-capable executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-06-03T16:11:36Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_88228a1573358f8f`

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

Aggregate-only proof available to this executor from the WorkItem:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; visible age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; visible age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; visible age bucket `7_29d`.
- `test_like paid`: 2 `dot_com`; visible age bucket `7_29d`.
- `test_like internal_test`: 2 `foundry_owned`; visible age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- A future credential-capable executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-06-03T18:11:16Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_88ebe484f12b3331`

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

Aggregate-only proof from the WorkItem originating ref:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- A future credential-capable executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-06-03T09:10:27Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_805f1de42d933237`

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-22T14:10Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Planner aggregate refresh - 2026-06-03T07:16Z

Fresh planner helper execution:

```sh
./tools/geo-jobs-redacted-read.sh
```

Result:

- The planner runtime could read aggregate redacted score-fix proof.
- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets `30d_plus=2`, `7_29d=1`.
- `test_like pending`: 4 `dot_com`; age buckets `30d_plus=3`, `7_29d=1`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus=1`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus=2`.
- `test_like internal_test`: 2 `foundry_owned`; age buckets `30d_plus=1`, `7_29d=1`.
- Customer-visible score-fix follow-up due now remains 0 unless a future duplicate check and fresh public-action lock prove otherwise.

Decision:

- Keep the executor lane `credential_required` because the executor runtime, not this planner runtime, failed closed on missing NHS admin Keychain aliases.
- A future credential-capable executor may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-06-03T07:11:02Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_800eb0b312e595ab`

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-19T16:08Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; visible age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; visible age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; visible age bucket `7_29d`.
- `test_like paid`: 2 `dot_com`; visible age bucket `7_29d`.
- `test_like internal_test`: 2 `foundry_owned`; visible age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- A future credential-capable executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.
 
## Credential-blocked executor retry - 2026-05-27T01:10:38Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_844c2305c8bb8b6e`

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-27T01:09:49Z`:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-06-03T03:11:18Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_7998b109d60f4551`

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-19T12:08:00Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `7_29d`.
- `test_like paid`: 2 `dot_com`; age bucket `7_29d`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-06-03T01:10:01Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_8103c7137af8d820`

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

Latest aggregate proof remains the planner-provided aggregate from `2026-06-03T01:08:50Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`.
- `test_like pending`: 4 `dot_com`.
- `test_like lead`: 1 `dot_com`.
- `test_like paid`: 2 `dot_com`.
- `test_like internal_test`: 2 `foundry_owned`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-06-02T23:10:32Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_7549b6af7f16072a`

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

Latest aggregate proof remains the planner-provided aggregate from `2026-06-02T23:09:35Z`:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 2 in `7_29d`, 1 in `lt_1d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.
- This executor could not clear assume-unchanged flags or commit because git cannot create `.git/index.lock`: `Operation not permitted`.

## Credential-blocked executor retry - 2026-06-02T18:10:31Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_73d1be0b5c01420f`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

Required pre-read not available in this worktree:

- `harness/work_machine_0288ea9945bc8692.md`

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

Latest aggregate proof remains the planner-provided aggregate from `2026-06-02T18:09:32Z`:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must have `nhs-admin-api-key` or `nothumansearch-admin-key` available, re-read the current score-fix ledgers plus the referenced harness note if present, run the redacted helper successfully, and classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-06-01T03:10:52Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_6912459e1a86cb9d`

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
- No customer-visible score-fix email was sent.
- No public-action lock was created or reused.
- No external customer row was mutated.

Latest aggregate proof remains the planner-provided aggregate from `2026-06-01T03:10:52Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-06-01T01:10:27Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_66720fe7b4379bd7`

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

Planner aggregate-only proof for this work item:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; visible age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; visible age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; visible age bucket `7_29d`.
- `test_like paid`: 2 `dot_com`; visible age bucket `7_29d`.
- `test_like internal_test`: 2 `foundry_owned`; visible age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-05-31T23:10:45Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_5f2d8aead0a9da5f`

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-31T23:09:55Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; visible age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; visible age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; visible age bucket `7_29d`.
- `test_like paid`: 2 `dot_com`; visible age bucket `7_29d`.
- `test_like internal_test`: 2 `foundry_owned`; visible age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External real-candidate pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-05-31T19:10:48Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_53ae003b3be3d67a`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- `harness/work_machine_0288ea9945bc8692.md` was requested by the WorkItem but is not present in this checkout.
- `harness/work_machine_51465b9dcbbab335.md` was read as the recent tracked closeout for this lane.

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-31T19:09:41Z`:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 2 in `7_29d`, 1 in `lt_1d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next credential-capable executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-05-31T17:09:59Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_51465b9dcbbab335`

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-31T17:09:59Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`.
- `test_like pending`: 4 `dot_com`.
- `test_like lead`: 1 `dot_com`.
- `test_like paid`: 2 `dot_com`.
- `test_like internal_test`: 2 `foundry_owned`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate pending` rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-05-31T15:10:21Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_50f99e33530b8abf`

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-18T21:08Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; visible age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; visible age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; visible age bucket `7_29d`.
- `test_like paid`: 2 `dot_com`; visible age bucket `7_29d`.
- `test_like internal_test`: 2 `foundry_owned`; visible age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- A future executor may classify or clean up only `test_like pending` rows through the private admin workflow after a successful redacted helper read.

## Credential-blocked executor retry - 2026-05-31T13:09:17Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_50e939bf54d3db5e`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- `harness/work_machine_3cb7432451fd76f7.md`
- `harness/work_machine_454002e6e1ed2d10.md`
- `harness/work_machine_05b0a8e8b65d0c13.md`

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-20T19:08Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the private score-fix cleanup lane open as `credential_required` for executor runtime.
- External `real_candidate` pending rows stay untouched.
- The prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` continues to block another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- A future executor may classify or clean up only `test_like pending` rows through the private admin workflow after `tools/geo-jobs-redacted-read.sh` succeeds in that executor runtime.

## Credential-blocked executor retry - 2026-05-31T11:11:05Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_4f4c1b1d7fd41bd3`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- `harness/work_machine_3cb7432451fd76f7.md`
- `harness/work_machine_454002e6e1ed2d10.md`
- `harness/work_machine_05b0a8e8b65d0c13.md`
- `harness/work_machine_4c99ba3904d81586.md`

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-19T20:08Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `7_29d`.
- `test_like paid`: 2 `dot_com`; age bucket `7_29d`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the private score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-05-31T09:09:52Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_4c99ba3904d81586`

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-31T09:09:52Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`.
- `test_like pending`: 4 `dot_com`.
- `test_like lead`: 1 `dot_com`.
- `test_like paid`: 2 `dot_com`.
- `test_like internal_test`: 2 `foundry_owned`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-05-30T21:12:35Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_3cb7432451fd76f7`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

The requested `harness/work_machine_0288ea9945bc8692.md` file is not present in this worktree.

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-30T21:11:11Z`:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 2 in `7_29d`, 1 in `lt_1d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-05-30T21:00:36Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_3bb350e11511370c`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- `harness/work_machine_3889b067eadd52ee.md`

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

Latest aggregate proof available to this worker is the planner-provided aggregate from `2026-05-30T21:00:36Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-05-30T20:56:46Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_3889b067eadd52ee`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- `harness/work_machine_0288ea9945bc8692.md` was requested by the work item but is not present in this worktree. The prior ledger already records the same missing-file condition.

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-30T20:55:44Z`:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked QLimit retry - 2026-05-28T04:59:03Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_35368b1590bdff6e`

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-28T04:58:16Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `7_29d`.
- `test_like paid`: 2 `dot_com`; age bucket `7_29d`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep external `real_candidate` pending rows untouched; the prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- Keep the score-fix cleanup lane open as `credential_required`.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-05-27T20:11:26Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_ce8388b63752a5b3`

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
- No customer-visible score-fix email was sent.
- No public-action lock was created or reused.
- No external customer row was mutated.

Latest aggregate proof remains the planner-provided aggregate from `2026-05-27T20:10:32Z`:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-05-27T18:11:56Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_ce00d4b302bb607c`

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
- No customer-visible score-fix email was sent.
- No public-action lock was created or reused.
- No external customer row was mutated.

Latest aggregate proof remains the planner-provided aggregate from `2026-05-27T18:10:58Z`:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-05-27T17:47:25Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_cb4575465993faad`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- attempted `harness/work_machine_0288ea9945bc8692.md`; it is not present in this worktree

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-27T17:46:38Z`:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-05-27T05:10:14Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_88d71ae7d3d15ec6`

Required pre-read completed before any private score-fix mutation:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- attempted `harness/work_machine_0288ea9945bc8692.md`; it is not present in this worktree

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-27T05:09:34Z`:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

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

## Credential-blocked executor retry - 2026-05-31T21:09:53Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_562cecd13bf30eee`

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-19T08:08Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; visible age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; visible age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; visible age bucket `7_29d`.
- `test_like paid`: 2 `dot_com`; visible age bucket `7_29d`.
- `test_like internal_test`: 2 `foundry_owned`; visible age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External real-candidate pending rows stay untouched; the prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run the redacted helper successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Credential-blocked executor retry - 2026-05-27T11:10:36Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_b55f9deceff08ced`

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

Latest aggregate proof remains the planner-provided aggregate from `2026-05-27T11:09:42Z`:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 1 in `1_6d`, 2 in `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

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

## Credential-blocked executor retry - 2026-06-01T07:12:25Z

Automation: `business-agent-not-human-search`
WorkItem: `work_machine_6dff63a480dc65e0`

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

Latest aggregate proof remains the planner-provided aggregate from `2026-06-01T07:11:20Z`:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision:

- Keep the score-fix cleanup lane open as `credential_required`.
- External `real_candidate` pending rows stay untouched; prior follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md` still blocks another customer-visible score-fix email unless a future duplicate check and fresh public-action lock prove it is due.
- The next executor must run `tools/geo-jobs-redacted-read.sh` successfully before any private score-fix mutation and may classify or clean up only `test_like pending` rows through the private admin workflow.

## Current executor closeout - 2026-06-01T09:10:57Z

WorkItem: `work_machine_6f44993e08edee31`

This executor re-read the redacted helper and both score-fix ledgers, then ran:

```sh
./tools/geo-jobs-redacted-read.sh
```

The helper failed closed before fetching admin rows: `missing Keychain service: nhs-admin-api-key nothumansearch-admin-key`.

No raw admin rows were fetched. No private score-fix mutation was attempted. No customer-visible score-fix email was sent. No public-action lock was created or reused. No external customer row was mutated.

Aggregate-only proof available to this executor:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision remains `credential_required`: run the helper from a credential-capable executor before any private cleanup, classify only `test_like pending` rows, and keep external customer rows untouched unless a future duplicate check plus fresh public-action lock proves a new customer-visible touch is due.

## QLimit credential-blocked closeout - 2026-06-01T11:10:55Z

WorkItem: `work_machine_70c933b0ad431023`

Required pre-read completed before any score-fix state change:

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

Aggregate-only proof available to this executor from the WorkItem:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision remains `credential_required`: external `real_candidate` pending rows stay untouched; the already-contacted external pending cohort must not receive another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due. The next executor may classify or clean up only `test_like pending` rows through the private admin workflow after `tools/geo-jobs-redacted-read.sh` succeeds.

## QLimit credential-blocked closeout - 2026-06-07T12:10:50Z

WorkItem: `work_machine_e72202cb6f6c5b72`

Required pre-read completed before any score-fix state change:

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

Aggregate-only proof available to this executor from the WorkItem:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`.
- `test_like pending`: 4 `dot_com`.
- `test_like lead`: 1 `dot_com`.
- `test_like paid`: 2 `dot_com`.
- `test_like internal_test`: 2 `foundry_owned`.
- Customer-visible score-fix follow-up due now: 0.

Decision remains `credential_required`: external `real_candidate` pending rows stay untouched; the already-contacted external pending cohort must not receive another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due. The next executor may classify or clean up only `test_like pending` rows through the private admin workflow after `tools/geo-jobs-redacted-read.sh` succeeds.

## QLimit credential-blocked closeout - 2026-06-07T05:10:14Z

WorkItem: `work_machine_e3ab0d5db0dab5b4`

Required pre-read completed before any score-fix state change:

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

Aggregate-only proof available to this executor from the WorkItem planner evidence:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`.
- `test_like pending`: 4 `dot_com`.
- `test_like lead`: 1 `dot_com`.
- `test_like paid`: 2 `dot_com`.
- `test_like internal_test`: 2 `foundry_owned`.
- Customer-visible score-fix follow-up due now: 0.

Decision remains `credential_required`: external `real_candidate` pending rows stay untouched; the already-contacted external pending cohort must not receive another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due. The next executor may classify or clean up only `test_like pending` rows through the private admin workflow after `tools/geo-jobs-redacted-read.sh` succeeds.

## QLimit credential-blocked closeout - 2026-06-07T01:11:38Z

WorkItem: `work_machine_d8cc49d7154f8d02`

Required pre-read completed before any score-fix state change:

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

Aggregate-only proof available to this executor from the WorkItem:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision remains `credential_required`: external `real_candidate` pending rows stay untouched; the already-contacted external pending cohort must not receive another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due. The next executor may classify or clean up only `test_like pending` rows through the private admin workflow after `tools/geo-jobs-redacted-read.sh` succeeds.

## QLimit credential-blocked closeout - 2026-06-06T17:22:16Z

WorkItem: `work_machine_c64faded7d73b913`

Required pre-read completed before any score-fix state change:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

Required pre-read blocked by missing tracked artifact:

- `harness/work_machine_0288ea9945bc8692.md` is not present in this worktree.

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

Aggregate-only proof available to this executor from the WorkItem:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision remains `credential_required`: external `real_candidate` pending rows stay untouched; the already-contacted external pending cohort must not receive another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due. The next executor may classify or clean up only `test_like pending` rows through the private admin workflow after `tools/geo-jobs-redacted-read.sh` succeeds and the required score-fix context artifacts are present or superseded by a tracked closeout note.

## QLimit credential-blocked closeout - 2026-06-06T02:10:05Z

WorkItem: `work_machine_b60d65bc618dc7e8`

Required pre-read completed before any score-fix state change:

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

Aggregate-only proof available to this executor from the WorkItem:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision remains `credential_required`: external `real_candidate` pending rows stay untouched; the already-contacted external pending cohort must not receive another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due. The next executor may classify or clean up only `test_like pending` rows through the private admin workflow after `tools/geo-jobs-redacted-read.sh` succeeds.

## QLimit credential-blocked closeout - 2026-06-05T20:11:13Z

WorkItem: `work_machine_b494fbdc6e638f99`

Required pre-read completed before any score-fix state change:

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

Aggregate-only proof available to this executor from the WorkItem:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `7_29d`.
- `test_like paid`: 2 `dot_com`; age bucket `7_29d`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision remains `credential_required`: external `real_candidate` pending rows stay untouched; the already-contacted external pending cohort must not receive another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due. The next executor may classify or clean up only `test_like pending` rows through the private admin workflow after `tools/geo-jobs-redacted-read.sh` succeeds.

## QLimit credential-blocked closeout - 2026-06-05T16:12:00Z

WorkItem: `work_machine_afd0f95ec4033bed`

Required pre-read completed before any score-fix state change:

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

Aggregate-only proof available to this executor from the WorkItem source:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision remains `credential_required`: external `real_candidate` pending rows stay untouched; the already-contacted external pending cohort must not receive another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due. A credential-capable executor may classify or clean up only `test_like pending` rows through the private admin workflow after `tools/geo-jobs-redacted-read.sh` succeeds.

## QLimit credential-blocked closeout - 2026-06-05T15:34:00Z

WorkItem: `work_machine_afc7439301acbef4`

Required pre-read completed before any score-fix state change:

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

Aggregate-only proof available to this executor from the WorkItem:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision remains `credential_required`: external `real_candidate` pending rows stay untouched; the already-contacted external pending cohort must not receive another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due. A future credential-capable executor may classify or clean up only `test_like pending` rows through the private admin workflow after `tools/geo-jobs-redacted-read.sh` succeeds.

## QLimit credential-blocked closeout - 2026-06-05T14:12:14Z

WorkItem: `work_machine_ac861e59e6df7b41`

Required pre-read completed before any score-fix state change:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- Latest score-fix closeout note: `harness/work_machine_a84e5923ae8fca48-commit-blocker-2026-06-05.md`

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

Aggregate-only proof available to this executor from the WorkItem:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision remains `credential_required`: external `real_candidate` pending rows stay untouched; the already-contacted external pending cohort must not receive another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due. A future credential-capable executor may classify or clean up only `test_like pending` rows through the private admin workflow after `tools/geo-jobs-redacted-read.sh` succeeds.

## QLimit credential-blocked closeout - 2026-06-05T10:08:58Z

WorkItem: `work_machine_a84e5923ae8fca48`

Required pre-read completed before any score-fix state change:

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

Aggregate-only proof available to this executor from the WorkItem:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`.
- `test_like pending`: 4 `dot_com`.
- `test_like lead`: 1 `dot_com`.
- `test_like paid`: 2 `dot_com`.
- `test_like internal_test`: 2 `foundry_owned`.
- Customer-visible score-fix follow-up due now: 0.

Decision remains `credential_required`: external `real_candidate` pending rows stay untouched; the already-contacted external pending cohort must not receive another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due. The next executor may classify or clean up only `test_like pending` rows through the private admin workflow after `tools/geo-jobs-redacted-read.sh` succeeds.

## QLimit credential-blocked closeout - 2026-06-05T04:11:07Z

WorkItem: `work_machine_a071ade8b148576f`

Required pre-read completed before any score-fix state change:

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

Aggregate-only proof available to this executor from the WorkItem:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `7_29d`.
- `test_like paid`: 2 `dot_com`; age bucket `7_29d`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision remains `credential_required`: external `real_candidate` pending rows stay untouched; the already-contacted external pending cohort must not receive another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due. The next executor may classify or clean up only `test_like pending` rows through the private admin workflow after `tools/geo-jobs-redacted-read.sh` succeeds.

## QLimit credential-blocked closeout - 2026-06-03T17:20:00-07:00

WorkItem: `work_machine_8e09d039c0e9ff4e`

Required pre-read completed before any score-fix state change:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- `harness/work_machine_0288ea9945bc8692.md` was requested by the work item but is not present in this worktree. Prior score-fix ledgers already record that missing-file condition.

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

Aggregate-only proof available to this executor from the WorkItem:

- Total score-fix rows: 12.
- `real_candidate pending`: 3 `dot_com`; age buckets: 2 in `7_29d`, 1 in `lt_1d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision remains `credential_required`: external `real_candidate` pending rows stay untouched; the already-contacted external pending cohort must not receive another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due. The next executor may classify or clean up only `test_like pending` rows through the private admin workflow after `tools/geo-jobs-redacted-read.sh` succeeds.

## QLimit credential-blocked closeout - 2026-06-03T14:09:39Z

WorkItem: `work_machine_86d8753e8624a4f8`

Required pre-read completed before any score-fix state change:

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

Aggregate-only proof available to this executor from the WorkItem:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `30d_plus`.
- `test_like paid`: 2 `dot_com`; age bucket `30d_plus`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision remains `credential_required`: external `real_candidate` pending rows stay untouched; the already-contacted external pending cohort must not receive another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due. The next executor may classify or clean up only `test_like pending` rows through the private admin workflow after `tools/geo-jobs-redacted-read.sh` succeeds.

## QLimit credential-blocked closeout - 2026-06-02T21:14:42Z

WorkItem: `work_machine_75dffccae86cabc6`

Required pre-read completed before any score-fix state change:

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

Aggregate-only proof available to this executor from the WorkItem:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `7_29d`.
- `test_like paid`: 2 `dot_com`; age bucket `7_29d`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision remains `credential_required`: external `real_candidate` pending rows stay untouched; the already-contacted external pending cohort must not receive another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due. The next executor may classify or clean up only `test_like pending` rows through the private admin workflow after `tools/geo-jobs-redacted-read.sh` succeeds.

## QLimit credential-blocked closeout - 2026-06-02T17:33:57Z

WorkItem: `work_machine_7292e191790b1fca`

Required pre-read completed before any score-fix state change:

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

Aggregate-only proof available to this executor from the WorkItem:

- Total score-fix rows: 11.
- `real_candidate pending`: 2 `dot_com`; age bucket `7_29d`.
- `test_like pending`: 4 `dot_com`; age bucket `7_29d`.
- `test_like lead`: 1 `dot_com`; age bucket `7_29d`.
- `test_like paid`: 2 `dot_com`; age bucket `7_29d`.
- `test_like internal_test`: 2 `foundry_owned`; age bucket `7_29d`.
- Customer-visible score-fix follow-up due now: 0.

Decision remains `credential_required`: external `real_candidate` pending rows stay untouched; the already-contacted external pending cohort must not receive another customer-visible score-fix email unless a future duplicate check plus fresh public-action lock prove it is due. The next executor may classify or clean up only `test_like pending` rows through the private admin workflow after `tools/geo-jobs-redacted-read.sh` succeeds.
