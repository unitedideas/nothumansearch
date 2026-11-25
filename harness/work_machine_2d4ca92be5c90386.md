# 2026-05-21 Full Recrawl Aggregate Closeout

WorkItem: `work_machine_2d4ca92be5c90386`
Observed: `2026-06-10`

Scope: aggregate-only closeout for the completed 2026-05-21 full recrawl and bounded discovery-quality/quarantine refresh. This artifact intentionally excludes raw domains, URLs, row IDs, descriptions, emails, tokens, private notes, crawler row output, public posting, browser automation, desktop automation, credential reads, production-data deletion, broad crawl, and replacement full recrawl.

## Boundary Proof

- `tools/recrawl-health.log` records 2026-05-21 `seed_refresh` completion at `05:42:37` with `api_status=200`, `api_ok=1`, `workers=10`, and `dry_run=0`.
- `tools/recrawl-health.log` records 2026-05-21 `full_recrawl` start at `06:00:12` with preflight `api_status=200`, `api_ok=1`, `health_outcome=full_pressure`, and `workers=10`.
- `tools/recrawl-health.log` records the remote command as `/app/crawler_-recrawl_-workers_10`.
- `tools/recrawl-health.log` records 2026-05-21 `full_recrawl` completion at `10:25:48` with `api_status=200`, `api_ok=1`, `workers=10`, and `dry_run=0`.
- `tools/recrawl.log` records the matching aggregate completion: `success=9847`, `failed=389`, `total=10236`.
- `tools/seed-refresh.log` records the matching seed-refresh aggregate: `success=469`, `failed=14`, `total=483`.
- Repo-local lock check found no active recrawl, full-recrawl, seed-refresh, or crawler lock file under `tools/`.

## Bounded Refresh

Command used:

```bash
NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh
```

This is the existing aggregate helper path. It reads sanitized crawler summary lines and writes only aggregate planner artifacts; it does not start a crawl.

Current aggregate output:

- `sample_rows=11799`
- `hard_signal_rows=8526`
- `low_signal_rows=3273`
- `hard_signal_rate=0.7226`
- `category_other=299`
- `category_other_hard_agent_signal=74`
- `category_other_low_signal=225`
- `llms_only=578`
- `schema_only=596`
- `zero_score=1292`

Hard-signal `category=other` remains aggregate-only:

- `rows=74`
- `top_signal_sets`: `API=50`, `API,schema.org=24`
- `score_buckets`: `0_24=74`, `25_39=0`, `40_59=0`, `60_plus=0`

## Decision

Decision: `no_op_fixed_point`.

- `taxonomy_rule_change=false`
- `threshold_adjustment=false`
- `no_op_fixed_point=true`

Reason: the current low-signal `category=other` aggregate lacks proven hard agent signals. Aggregate counts do not identify a narrow reusable taxonomy rule, and a threshold adjustment would weaken the API/OpenAPI/MCP/ai-plugin boundary.

## Guard State

These cohorts remain audit-only unless a hard agent signal is proven through an executor-only path:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `models.AgentFirstFilter`. Score-fix targeting remains gated on `HasHardAgentSignal`.

## Queue Decision

No new discovery-quality WorkItem is warranted. The lane is at a true fixed point for this boundary: no taxonomy-rule change, no threshold adjustment, no public-search eligibility change, and no score-fix targeting change. `harness/generated-work-items.json` was left unchanged because its current rows are unrelated deploy-gated or credential-gated lanes.

## Verification

- `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh`
- `find tools -maxdepth 1 -name '*recrawl*.lock' -o -name '*crawler*.lock' -o -name '*seed-refresh*.lock'`
- `python3 -m json.tool harness/discovery-quality-latest.json`
- `python3 -m json.tool harness/discovery-quarantine-latest.json`
- `python3 -m json.tool harness/generated-work-items.json`
- `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py`
