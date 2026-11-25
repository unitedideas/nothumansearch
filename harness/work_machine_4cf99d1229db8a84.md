# Full Recrawl Discovery-Quality Closeout

WorkItem: `work_machine_4cf99d1229db8a84`

Scope: aggregate closeout for the completed 2026-05-21 full recrawl and refreshed sanitized discovery-quality quarantine state. This run stayed inside the business repo and used only repo-local aggregate logs and helper output. It did not start a full recrawl, broad crawl, browser automation, public action, private row fetch, or score-fix targeting.

## Boundary Evidence

- `tools/recrawl-health.log` records the 2026-05-21 `seed_refresh` completion at 05:42:37 with `api_status=200`, `api_ok=1`, `workers=10`, and `dry_run=0`.
- `tools/recrawl-health.log` records the 2026-05-21 `full_recrawl` start at 06:00:12 with preflight `api_status=200`, `api_ok=1`, `health_outcome=full_pressure`, and `workers=10`.
- `tools/recrawl.log` records the 2026-05-21 full recrawl completion at 10:25:48 with success 9847, failed 389, total 10236.
- Repo-local lock check found no active `tools/full-recrawl.lock` or `tools/recrawl.lock` during this closeout.

No raw domains, URLs, row IDs, descriptions, emails, tokens, private notes, private query logs, or crawler row output are included here.

## Refreshed Aggregate Artifacts

Command:

```bash
NHS_DISCOVERY_INPUT=tools/seed-refresh.log PYTHONDONTWRITEBYTECODE=1 ./tools/refresh-discovery-quality.sh
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

Reason: the refreshed aggregate state still shows low-signal `category=other` rows without hard agent signals. Aggregate counts alone do not identify a narrow taxonomy rule, and the quarantine contract requires hard agent evidence before public-search or score-fix eligibility changes.

## Guard State

These cohorts remain audit-only:

- `llms_only`: rows 601, `public_search=false`, `score_fix_targeting=false`
- `schema_only`: rows 621, `public_search=false`, `score_fix_targeting=false`
- `zero_score`: rows 1342, `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: rows 234, `public_search=false`, `score_fix_targeting=false`

Hard-signal `category=other` remains aggregate-review-only. No candidate-row work item was promoted from discovery logs.

## Queue State

This lane is at a true fixed point, so `harness/generated-work-items.json` was left without a new discovery-quality follow-up. The existing generated items are unrelated deploy- or credential-gated work.

## Verification

- `python3 -m json.tool harness/discovery-quality-latest.json`
- `python3 -m json.tool harness/discovery-quarantine-latest.json`
- `python3 -m json.tool harness/generated-work-items.json`
- `python3 tools/quality-gate-discovery.py --quarantine harness/discovery-quarantine-latest.json --repo-root .`
- `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py`
