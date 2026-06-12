# Work Machine AC34279093E6626F

## Scope

Refreshed the sanitized aggregate discovery-quality boundary with:

```bash
./tools/refresh-discovery-quality.sh
```

The helper used bounded aggregate logs only and regenerated the aggregate latest artifacts. It did not start a full recrawl or broad crawl, and no raw domains, URLs, row IDs, descriptions, emails, tokens, or private notes were copied into committed artifacts.

## Result

The regenerated `harness/discovery-quality-latest.json` and `harness/discovery-quarantine-latest.json` were byte-identical to the current artifacts.

Aggregate state:

- `hard_signal_rows`: 9208
- `low_signal_rows`: 3537
- `category_other_low_signal`: 243
- `category_other_hard_agent_signal`: 80
- `quarantine.active`: true
- `planner_priority`: `quarantine_first`

## Decision

Post-recrawl `category=other` is a **no-op fixed point**.

It is not a taxonomy-rule change because aggregate counts alone do not prove a narrow category rule, and the low-signal `category=other` cohort lacks hard agent signals.

It is not a threshold adjustment because `AgentFirstFilter` and `HasHardAgentSignal` remain the correct public-search and score-fix boundaries.

The hard-signal `category=other` cohort remains aggregate-review-only until a targeted, sanitized sampler proves a specific taxonomy rule.

## Guardrails

These cohorts remain audit-only with `public_search=false` and `score_fix_targeting=false` unless a hard agent signal is proven:

- `llms_only`
- `schema_only`
- `zero_score`
- `category_other_low_signal`

No public ranking, score-fix targeting, or lead-generation behavior changes are authorized by this refresh.

## Follow-Up

No new follow-up was discovered beyond the existing entries in `harness/generated-work-items.json`. The next useful action remains credential-capable private cleanup/review work already listed there; broad recrawls are not indicated by this aggregate refresh.
