Scope: sanitized aggregate discovery-quality and discovery-quarantine refresh after the completed 2026-05-19 full recrawl.

WorkItem: `work_machine_2247d15593bb4097`

## Boundary

Used only repo-local aggregate helper output:

```bash
NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh
```

No full recrawl, broad crawl, browser automation, public submission, public posting, score-fix targeting, or raw candidate-row artifact was created for this work item.

## Refreshed Aggregate State

The helper refreshed the sanitized aggregate artifacts:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

Current aggregate helper output:

```text
discovery_quality_refresh hard_signal_rows=8526 low_signal_rows=3273 category_other_low_signal=225 quarantine_active=true planner_priority=quarantine_first history=harness/discovery-quarantine-history.jsonl
```

Current aggregate cohort state:

- `sample_rows`: 11799
- `hard_signal_rows`: 8526
- `low_signal_rows`: 3273
- `category_other`: 299
- `category_other_hard_agent_signal`: 74
- `category_other_low_signal`: 225
- `llms_only`: 578
- `schema_only`: 596
- `zero_score`: 1292

The committed artifacts remain aggregate-only. They do not contain raw domains, URLs, row IDs, descriptions, emails, tokens, private notes, or private query logs.

## Decision

Post-recrawl `category=other` state is a no-op fixed point.

Decision matrix:

- `taxonomy_rule_change`: false
- `threshold_adjustment`: false
- `no_op_fixed_point`: true

Reason: the low-signal `category=other` cohort lacks hard agent signals. Aggregate counts alone are not evidence for a narrow taxonomy rule, and changing thresholds would promote audit-only rows without proving API, OpenAPI, MCP, or ai-plugin support.

## Guard State

These cohorts remain audit-only unless a hard agent signal is proven by a future bounded helper or private executor path:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `models.AgentFirstFilter`; score-fix targeting remains gated on `HasHardAgentSignal`.

## Generated Work Items

No new discovery-quality row was added to `harness/generated-work-items.json`. This discovery-quality lane is at a true fixed point. The remaining generated work items cover unrelated deploy-required API-key commerce/admin traffic work and credential-required private monitor/score-fix lanes.

## Verification

- `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh` passed.
- `python3 -m json.tool harness/discovery-quality-latest.json` passed.
- `python3 -m json.tool harness/discovery-quarantine-latest.json` passed.
- `python3 -m json.tool harness/generated-work-items.json` passed.
- `python3 tools/quality-gate-discovery.py --quarantine harness/discovery-quarantine-latest.json --repo-root .` passed.
- `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py` passed.
