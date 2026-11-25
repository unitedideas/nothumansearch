# WorkItem closeout: discovery-quality quarantine refresh

WorkItem: `work_machine_9c95e8ac13e79c7d`

No full recrawl, broad crawl, deploy, browser automation, desktop automation, public action, production data deletion, private row inspection, process-environment inspection, or secret read was performed.

## Aggregate Refresh

Regenerated the sanitized aggregate artifacts with:

`./tools/refresh-discovery-quality.sh`

The wrapper used bounded aggregate discovery output only and regenerated:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

No raw domains, URLs, row ids, descriptions, emails, tokens, private notes, or crawler row output were copied into committed artifacts.

## Refreshed Aggregate Counts

- `sample_rows=12745`
- `hard_signal_rows=9208`
- `low_signal_rows=3537`
- `hard_signal_rate=0.7225`
- `category_other=323`
- `category_other_hard_agent_signal=80`
- `category_other_low_signal=243`
- `llms_only=625`
- `schema_only=647`
- `zero_score=1390`
- `passive_or_soft_signal=875`

`hard_signal_other_review` remains aggregate-only: `rows=80`; score buckets `0_24=80`, `25_39=0`, `40_59=0`, `60_plus=0`; top signal sets are `API=54` and `API,schema.org=26`.

## Decision

Post-recrawl `category=other` state is a no-op fixed point for this lane.

- `taxonomy_rule_change=false`
- `threshold_adjustment=false`
- `no_op_fixed_point=true`
- `category_other_low_signal=no_op_fixed_point`
- `taxonomy_rule=not_from_low_signal_aggregate_cohort`
- `threshold_adjustment=none`

Reason: low-signal `category=other` rows lack hard agent signals by definition. The aggregate-only refresh does not prove a narrow taxonomy rule, and a threshold change would promote passive or soft-signal rows instead of proving agent usability.

## Guard State

These cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `models.AgentFirstFilter`. Score-fix targeting remains gated on `HasHardAgentSignal`.

## Follow-up State

`harness/generated-work-items.json` is intentionally unchanged because the post-recrawl `category=other` quarantine decision is a true fixed point for this aggregate-only lane. No replacement full recrawl, broad crawl, taxonomy-rule change, threshold adjustment, public-search promotion, score-fix targeting, browser task, deploy task, public action, or private-row follow-up is warranted from the refreshed low-signal cohorts.

## Verification

- `python3 tools/test-discovery-quality-report.py`
- `python3 tools/test-discovery-quarantine-report.py`
- `python3 tools/test-refresh-discovery-quality.py`
- `GOCACHE=/tmp/nhs-go-cache go test ./...`

## Commit Blocker

Normal git index writes are blocked in this runner:

`fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted`

The tracked aggregate artifact files are also marked assume-unchanged in the local index, and clearing that flag requires the same blocked index write. Commit command to run in a writable git runtime:

`git update-index --no-assume-unchanged harness/discovery-quality-latest.json harness/discovery-quarantine-latest.json harness/discovery-quarantine-history.jsonl harness/generated-work-items.json && git add harness/discovery-quality-latest.json harness/discovery-quarantine-latest.json harness/discovery-quarantine-history.jsonl harness/work_machine_9c95e8ac13e79c7d.md && git commit -m "Refresh discovery quarantine aggregate"`
