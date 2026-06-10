# Discovery Quality Aggregate Refresh

Work item: `work_machine_2bd700a2cb981b16`

Observed at: `2026-06-10T06:10:02Z`

## Inputs

- Source artifact: `harness/discovery-quality-latest.json`
- Reporter: `python3 tools/discovery-quarantine-report.py`
- Scope: sanitized aggregate helper output only
- Broad crawl/full recrawl: not run
- Row-level samples: not read

## Regeneration

Command:

```bash
python3 tools/discovery-quarantine-report.py \
  --input harness/discovery-quality-latest.json \
  --output harness/discovery-quarantine-latest.json \
  --history-output harness/discovery-quarantine-history.jsonl \
  --observed-at 2026-06-10T06:10:02Z
```

Result: `harness/discovery-quarantine-latest.json` and the weekly history row are already at the same sanitized aggregate state, so regeneration produced no JSON diff.

## Aggregate Decision

Decision: no-op fixed point.

- Taxonomy-rule change: false
- Threshold adjustment: false
- No-op fixed point: true

Reason: `category=other` low-signal rows still lack proven hard agent signals in the aggregate artifact. Aggregate counts alone are not enough evidence for a narrow taxonomy rule or scoring threshold change.

## Guard State

The following cohorts remain audit-only:

- `llms_only`
- `schema_only`
- `zero_score`
- `category_other_low_signal`

Committed planner artifacts keep these cohorts out of public search and score-fix targeting:

- `public_search=false`
- `score_fix_targeting=false`

Promotion remains blocked unless a hard agent signal is proven through the private executor path.

## Follow-up Queue

No new discovery-quality follow-up was added to `harness/generated-work-items.json` because this lane is at a true fixed point. Existing queue entries are unrelated credential/deploy work and were left untouched.

## Privacy Boundary

No raw domains, URLs, row IDs, descriptions, emails, tokens, or private review notes were copied into committed artifacts.
