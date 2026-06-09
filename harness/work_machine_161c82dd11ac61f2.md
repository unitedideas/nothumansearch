# 2026-05-21 Full Recrawl Discovery Closeout

Work item: `work_machine_161c82dd11ac61f2`

Scope: aggregate-only closeout for the completed 2026-05-21 full recrawl and the sanitized discovery-quality refresh already produced from the completed boundary. This note intentionally excludes raw domains, URLs, row IDs, descriptions, emails, tokens, and private review notes.

## Boundary Proof

- `tools/recrawl-health.log` records 2026-05-21 `seed_refresh` completion at `05:42:37` with `api_status=200`, `api_ok=1`, `workers=10`, and `dry_run=0`.
- `tools/recrawl-health.log` records 2026-05-21 `full_recrawl` start at `06:00:12` after preflight `api_status=200` and `api_ok=1`.
- `tools/recrawl-health.log` records 2026-05-21 `full_recrawl` completion at `10:25:48` with `api_status=200`, `api_ok=1`, `workers=10`, and `dry_run=0`.
- `tools/recrawl.log` records the matching full-recrawl aggregate: `Success=9847`, `Failed=389`, `Total=10236`.
- Repo-local lock check found no `tools/full-recrawl.lock` or `tools/recrawl.lock`.

No full recrawl, broad crawl, browser automation, public action, private row fetch, or credential read was performed for this WorkItem.

## Sanitized Refresh State

The active aggregate artifacts are:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

They were refreshed from a bounded seed-refresh aggregate slice for the completed 2026-05-21 boundary, not from a broad crawl. Current committed aggregate values:

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

Hard-signal `category=other` remains aggregate-only:

- `rows=3`
- `top_signal_sets`: `API=2`, `API,schema.org=1`
- `score_buckets`: `0_24=3`, `25_39=0`, `40_59=0`, `60_plus=0`

## Decision

Decision: `no_op_fixed_point`.

- `taxonomy_rule_change=false`
- `threshold_adjustment=false`
- `no_op_fixed_point=true`

Reason: the current `category=other` low-signal aggregate does not prove a reusable taxonomy rule. The low-signal cohorts lack API, OpenAPI, MCP, or ai-plugin evidence. A threshold change would weaken the hard-agent-signal boundary.

## Guard State

These cohorts remain audit-only unless a hard agent signal is proven through an executor-only path:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `models.AgentFirstFilter`. Score-fix targeting remains gated on `HasHardAgentSignal`.

## Queue Decision

No new discovery-quality WorkItem is warranted. This lane is at a true fixed point: no taxonomy-rule change, no threshold adjustment, no public-search eligibility change, and no score-fix targeting change.

`harness/generated-work-items.json` was left unchanged because remaining generated rows are unrelated credential-gated or deploy-gated lanes.

## Verification

- `python3 -m json.tool harness/discovery-quality-latest.json`
- `python3 -m json.tool harness/discovery-quarantine-latest.json`
- `python3 -m json.tool harness/generated-work-items.json`
- `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py`
