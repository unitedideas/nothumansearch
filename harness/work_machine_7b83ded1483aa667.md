# NHS discovery-quality fixed point - 2026-05-25

WorkItem: `work_machine_7b83ded1483aa667`

Scope: sanitized discovery-quality and discovery-quarantine artifacts from the completed 2026-05-23 full-recrawl boundary plus the latest seed-refresh aggregate available in this repo.

## Boundary Evidence

- `tools/recrawl-health.log` records 2026-05-23 `full_recrawl` start and completion with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `tools/recrawl.log` records 2026-05-23 `Done. Success: 9867, Failed: 379, Total: 10246` and `NHS full_recrawl complete`.
- Newer local wrapper evidence is also present: 2026-05-25 `seed_refresh` completed with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`, followed by 2026-05-25 `full_recrawl` completion with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `tools/seed-refresh.log` records the latest seed refresh as `Done. Success: 477, Failed: 6, Total: 483` and `NHS seed_refresh complete`.

No full recrawl, broad crawl, public posting, browser automation, credential read, or row-level output was started for this WorkItem.

## Refresh

Command:

```bash
NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh
```

Output:

```text
discovery_quality_refresh hard_signal_rows=4440 low_signal_rows=1687 category_other_low_signal=117 quarantine_active=true planner_priority=quarantine_first history=harness/discovery-quarantine-history.jsonl
```

The command refreshed the existing sanitized aggregate artifacts:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

Those artifacts remain aggregate-only and must not contain private row-level crawler output.

## Aggregate Decision

Decision: `no_op_fixed_point`.

Current `category=other` state does not justify a taxonomy-rule change or threshold adjustment from aggregate evidence alone.

- Taxonomy-rule change: `none`
- Threshold adjustment: `none`
- Reason: the `category_other_low_signal` cohort lacks proven hard agent signals in this aggregate review path. Aggregate counts do not identify a narrow, safe taxonomy rule.
- `hard_signal_other_review` remains aggregate-only: rows=38, score bucket `0_24`=38, top signal sets `API`=26 and `API,schema.org`=12.

## Audit-Only Guards

The following cohorts remain private audit-only unless a future bounded executor proves a hard agent signal:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`, rows=303
- `schema_only`: `public_search=false`, `score_fix_targeting=false`, rows=308
- `zero_score`: `public_search=false`, `score_fix_targeting=false`, rows=672
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`, rows=117

Public search remains protected by `models.AgentFirstFilter`. Score-fix targeting still requires a hard agent signal.

## Generated Work Items

`harness/generated-work-items.json` is intentionally unchanged for this discovery-quality lane. The lane is at a true fixed point: no replacement full-recrawl, broad crawl, taxonomy-rule-change, threshold-adjustment, public-search, or score-fix-targeting follow-up is warranted from this aggregate evidence.

Existing generated items for API-key commerce/admin traffic buckets and private score-fix cleanup are unrelated and remain queued.

## Verification

- `python3 -m json.tool harness/discovery-quality-latest.json`
- `python3 -m json.tool harness/discovery-quarantine-latest.json`
- `python3 -m json.tool harness/generated-work-items.json`
- `python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py` passed: 17 tests.
- `GOCACHE=/private/tmp/nothumansearch-gocache go test ./...` did not pass because `cmd/monitor-check` still references missing helper symbols: `firstCheckFailedQuarantineReason`, `firstCheckZeroScoreQuarantineReason`, and `firstCheckQuarantineReason`. This is existing monitor-check drift outside the aggregate discovery-quality artifact lane.
