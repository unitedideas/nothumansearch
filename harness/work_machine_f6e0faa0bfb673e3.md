# work_machine_f6e0faa0bfb673e3

## Scope

Refresh sanitized discovery-quality and discovery-quarantine artifacts from the completed recrawl boundary plus the bounded seed-refresh aggregate. Decide whether current `category=other` state needs a taxonomy-rule change, threshold adjustment, or no-op fixed point.

## Inputs Used

- `tools/refresh-discovery-quality.sh`
- `tools/discovery-quality-report.py`
- `tools/discovery-quarantine-report.py`
- `tools/taxonomy-other-redacted-sample.py`
- `tools/recrawl-health.log`
- `tools/recrawl.log`
- `tools/seed-refresh.log`

No full recrawl, broad crawl, browser automation, desktop automation, public posting, or production mutation was started.

## Refreshed Aggregate State

- Latest aggregate refresh completed through the repo helper.
- `sample_rows`: 13218
- `hard_signal_rows`: 9548
- `low_signal_rows`: 3670
- `hard_signal_rate`: 0.7223
- `category_other_low_signal`: 252
- `category_other_hard_agent_signal`: 83
- `llms_only`: 649
- `schema_only`: 673
- `zero_score`: 1439
- Quarantine remains active because `category_other_low_signal` exceeds `category_other_hard_agent_signal`.

## Decision

`category=other` is a no-op fixed point for this WorkItem.

- `taxonomy_rule_change`: false
- `threshold_adjustment`: false
- `no_op_fixed_point`: true

Reason: the remaining low-signal `category=other` cohort has no proven hard agent signal in the aggregate artifact. Aggregate counts are not enough evidence for a narrow taxonomy rule. The hard-signal `category=other` subset is small and still aggregate-only; adding taxonomy rules would require a private bounded sampler plus crawler unit test, not planner artifacts.

## Guard State

The following cohorts remain audit-only:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `models.AgentFirstFilter`. Score-fix targeting remains gated on hard agent signal.

## Bounded Sampler

The optional `taxonomy-other-redacted-sample.py --limit 50` sampler failed closed with `sample_status=failed` and `reason=URLError`. It emitted no raw domains, URLs, row IDs, names, descriptions, emails, tokens, or crawler row output.

## Artifacts Updated

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`
- `harness/generated-work-items.json`

