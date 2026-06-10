# Discovery-quality fixed point closeout

WorkItem: `work_machine_2e8bad9273375480`
Observed: `2026-06-10`

## Inputs

- Completed full-recrawl boundary: `2026-05-21 10:25:48`, `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- Full-recrawl aggregate: `success=9847`, `failed=389`, `total=10236`.
- Seed-refresh aggregate boundary: `2026-05-22 05:39:47`, `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- Seed-refresh aggregate: `success=476`, `failed=7`, `total=483`.
- Lock check: no `tools/full-recrawl.lock`, `tools/recrawl.lock`, or matching recrawl lock file was present.

## Sanitized Aggregate Refresh

The bounded May 22 seed-refresh section was regenerated through the aggregate-only discovery-quality and discovery-quarantine helpers. No broad crawl, full recrawl, row-level sampler, raw domain, URL, row id, description, email, token, private note, or crawler row output was committed.

May 22 aggregate output:

- `sample_rows=473`
- `hard_signal_rows=343`
- `low_signal_rows=130`
- `hard_signal_rate=0.7252`
- `category_other=12`
- `category_other_hard_agent_signal=3`
- `category_other_low_signal=9`
- `llms_only=23`
- `schema_only=23`
- `zero_score=52`
- `passive_or_soft_signal=32`

Current tracked aggregate snapshot remains `harness/discovery-quality-latest.json` and `harness/discovery-quarantine-latest.json`. Those files already hold the newer aggregate snapshot and guard state, so this closeout does not downgrade them to the older May-only sample.

Current tracked aggregate guard snapshot:

- `sample_rows=11799`
- `hard_signal_rows=8526`
- `low_signal_rows=3273`
- `category_other_low_signal=225`
- `llms_only=578`
- `schema_only=596`
- `zero_score=1292`

## Decision

`category=other` remains a no-op fixed point.

- Taxonomy-rule change: no.
- Threshold adjustment: no.
- No-op fixed point: yes.

Reason: the low-signal `category=other` cohort has no aggregate hard-agent-signal proof that supports a narrow taxonomy rule. Threshold adjustment would weaken the public/search and score-fix boundary for passive rows, so the correct state is fixed-point quarantine.

## Guard State

- `llms-only`: audit-only, `public_search=false`, `score_fix_targeting=false`.
- `schema-only`: audit-only, `public_search=false`, `score_fix_targeting=false`.
- `zero-score`: audit-only, `public_search=false`, `score_fix_targeting=false`.
- Low-signal `category=other`: aggregate-review-only, `public_search=false`, `score_fix_targeting=false`.

`harness/generated-work-items.json` was left unchanged because this lane is at a true fixed point. Existing queued follow-ups are separate credential/deploy-gated lanes.

## Verification

- `python3 tools/discovery-quality-report.py --input /private/tmp/nhs-seed-refresh-2026-05-22-work_machine_2e8.log --output /private/tmp/nhs-discovery-quality-2026-05-22-work_machine_2e8.json`
- `python3 tools/discovery-quarantine-report.py --input /private/tmp/nhs-discovery-quality-2026-05-22-work_machine_2e8.json --output /private/tmp/nhs-discovery-quarantine-2026-05-22-work_machine_2e8.json --observed-at 2026-05-22T14:10:23Z`
- `python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/test-taxonomy-other-redacted-sample.py`
- `python3 -m json.tool harness/discovery-quality-latest.json`
- `python3 -m json.tool harness/discovery-quarantine-latest.json`
- `python3 -m json.tool harness/generated-work-items.json`
- `python3 tools/quality-gate-discovery.py --quarantine harness/discovery-quarantine-latest.json --repo-root .`
