# Discovery Quality Fixed Point

WorkItem: `work_machine_49564c8e857fac1c`

Scope: sanitized aggregate discovery-quality and discovery-quarantine refresh for the completed full-recrawl boundary and seed-refresh aggregate. This run used only repo-local aggregate helper output. It did not start a full recrawl, broad crawl, browser automation, public submission, private row fetch, or score-fix targeting.

## Boundary Evidence

- `tools/recrawl-health.log` records completed `full_recrawl` boundaries with `api_status=200`, `api_ok=1`, `workers=10`, and `dry_run=0`. The WorkItem source boundary is 2026-05-21 10:25:48 with success 9847, failed 389, total 10236.
- `tools/seed-refresh.log` records the WorkItem source seed-refresh boundary on 2026-05-22 with success 476, failed 7, total 483.
- Current repo-local lock check found no `tools/full-recrawl.lock`, `tools/recrawl.lock`, or `tools/seed-refresh.lock`.

## Refreshed Aggregate Artifacts

Command:

```bash
NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh
```

Aggregate helper output:

```text
discovery_quality_refresh hard_signal_rows=8866 low_signal_rows=3405 category_other_low_signal=234 quarantine_active=true planner_priority=quarantine_first history=harness/discovery-quarantine-history.jsonl
```

Refreshed files:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

## Decision

Current `category=other` state is a no-op fixed point.

- Taxonomy-rule change: no.
- Threshold adjustment: no.
- No-op fixed point: yes.

Reason: the refreshed aggregate shows `category_other_low_signal=234` and `category_other_hard_agent_signal=77`. The hard-signal aggregate is only split across API and API plus schema signal sets, with all hard-signal other rows in the lowest score bucket. That is not enough aggregate evidence for a narrow taxonomy rule, and the low-signal cohort lacks hard agent signals by definition.

## Guard State

These cohorts remain audit-only:

- `llms_only`: rows 601, `public_search=false`, `score_fix_targeting=false`
- `schema_only`: rows 621, `public_search=false`, `score_fix_targeting=false`
- `zero_score`: rows 1342, `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: rows 234, `public_search=false`, `score_fix_targeting=false`

`hard_signal_other_review` remains aggregate-only. No row-level domains, URLs, row IDs, descriptions, emails, tokens, private notes, or crawler row output are included in this artifact.

## Queue State

The stale generated follow-up titled `Bounded aggregate review of hard-signal category=other taxonomy candidates` was removed from `harness/generated-work-items.json`. This discovery-quality lane is now at a true fixed point. The remaining generated WorkItems are unrelated credential/deploy-gated lanes.

## Verification

- `python3 -m json.tool harness/discovery-quality-latest.json`
- `python3 -m json.tool harness/discovery-quarantine-latest.json`
- `python3 -m json.tool harness/generated-work-items.json`
- `python3 tools/quality-gate-discovery.py --quarantine harness/discovery-quarantine-latest.json --repo-root .`
- `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py`
