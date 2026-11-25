# Work machine 05729d29d598ca21

Scope: sanitized aggregate discovery-quality and discovery-quarantine refresh after the completed 2026-05-19 full recrawl.

## Source Boundary

Used only repo-local wrapper and aggregate parser evidence:

- `tools/recrawl-health.log`: 2026-05-19 full_recrawl started at 06:00:11, preflight `api_status=200 api_ok=1`, `health_outcome=full_pressure workers=10`, `remote_start`, and completed at 10:26:32 with post-recrawl `api_status=200 api_ok=1 workers=10 dry_run=0`.
- `tools/recrawl.log`: 2026-05-19 full_recrawl completed with `Success=9830 Failed=396 Total=10226`.
- Bounded extraction source: 2026-05-19 recrawl summary lines only, written to `/private/tmp/nhs-2026-05-19-recrawl-summary.log` for local parsing.

No full recrawl, broad crawl, row-level admin fetch, browser automation, public action, or credential read was started.

## Refreshed Artifacts

`NHS_DISCOVERY_INPUT=/private/tmp/nhs-2026-05-19-recrawl-summary.log ./tools/refresh-discovery-quality.sh`

Output:

```text
discovery_quality_refresh hard_signal_rows=4114 low_signal_rows=5713 category_other_low_signal=2466 quarantine_active=true planner_priority=quarantine_first history=harness/discovery-quarantine-history.jsonl
```

The refresh updated the ignored aggregate working artifacts:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

The committed proof is this note. The artifacts remain sanitized aggregate-only and contain no domains, URLs, row IDs, descriptions, emails, tokens, or private notes.

## Aggregate Decision

Explicit post-recrawl decision:

- Taxonomy-rule change: `none`.
- Threshold adjustment: `none`.
- Fixed point: `category_other_low_signal=no_op_fixed_point`.
- Reason: the low-signal `category=other` cohort has no proven hard agent signal in this aggregate path. Aggregate counts alone do not identify a narrow safe taxonomy rule.

`hard_signal_other_review` remains aggregate-only:

- rows: 752
- score buckets: `0_24=431`, `25_39=29`, `40_59=288`, `60_plus=4`
- top signal sets are aggregate signal-set counts only.

## Guard State

The audit-only cohorts remain excluded:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`, rows=1004
- `schema_only`: `public_search=false`, `score_fix_targeting=false`, rows=538
- `zero_score`: `public_search=false`, `score_fix_targeting=false`, rows=2187
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`, rows=2466

Public guards remain:

- `public_search=protected_by_models.AgentFirstFilter`
- `score_fix_targeting=requires_has_hard_agent_signal`
- `planner_priority=quarantine_first`

## Generated Work Items

`harness/generated-work-items.json` is intentionally unchanged for this discovery-quality lane. This lane is at a true fixed point: no replacement full-recrawl, broad crawl, taxonomy-rule-change, threshold-adjustment, public-search, or score-fix-targeting follow-up is warranted from this aggregate evidence.

## Verification

- `python3 tools/quality-gate-discovery.py --quarantine harness/discovery-quarantine-latest.json --repo-root .` passed.
- `python3 -m json.tool harness/discovery-quality-latest.json` passed.
- `python3 -m json.tool harness/discovery-quarantine-latest.json` passed.
- `python3 -m json.tool harness/generated-work-items.json` passed.
- `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py` passed: 18 tests.

## Commit Blocker

This worker repaired and verified the local artifacts, but committing is blocked because Git cannot create `.git/index.lock` in this executor:

```text
fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted
```

Commit command for a git-writable executor:

```bash
git update-index --no-assume-unchanged tools/discovery-quality-report.py tools/discovery-quarantine-report.py tools/quality-gate-discovery.py tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py harness/discovery-quarantine-history.jsonl harness/generated-work-items.json
git add tools/discovery-quality-report.py tools/discovery-quarantine-report.py tools/quality-gate-discovery.py tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py harness/work_machine_05729d29d598ca21.md harness/discovery-quarantine-history.jsonl
git commit -m "Record discovery quarantine fixed point"
```
