Scope: sanitized aggregate discovery-quality and discovery-quarantine refresh after the completed 2026-05-19 full recrawl.

WorkItem: `work_machine_713555fe9226e4fb`

Bounded command used:

```bash
NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh
```

Helper result:

```text
discovery_quality_refresh hard_signal_rows=8866 low_signal_rows=3405 category_other_low_signal=234 quarantine_active=true planner_priority=quarantine_first history=harness/discovery-quarantine-history.jsonl
```

Refreshed sanitized aggregate artifacts:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

Current aggregate state:

- `sample_rows`: 12271
- `hard_signal_rows`: 8866
- `low_signal_rows`: 3405
- `hard_signal_rate`: 0.7225
- `category_other_low_signal`: 234
- `category_other_hard_agent_signal`: 77
- `quality_gate`: `review`
- `quality_gate.trigger`: `category_other_low_signal_exceeds_hard_signal`
- `hard_signal_other_review.top_signal_sets`: `API=52`, `API,schema.org=25`

Post-recrawl `category=other` decision:

- Taxonomy-rule change: no.
- Threshold adjustment: no.
- No-op fixed point: yes.

Reason: the low-signal `category=other` cohort lacks hard agent signals. Aggregate counts alone do not prove a narrow taxonomy rule, and a threshold adjustment would weaken the hard-agent-signal boundary. The hard-signal `category=other` cohort remains aggregate-review-only from sanitized signal-set counts.

Guard state:

- `llms_only`: audit-only, `public_search=false`, `score_fix_targeting=false`
- `schema_only`: audit-only, `public_search=false`, `score_fix_targeting=false`
- `zero_score`: audit-only, `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: aggregate-review-only, `public_search=false`, `score_fix_targeting=false`

Boundary:

- No full recrawl or broad crawl was started.
- No browser or desktop automation was used.
- No raw domains, URLs, row IDs, descriptions, emails, tokens, private notes, private query logs, or crawler row output were committed.
- Only the repo-local bounded aggregate helper and sanitized planner artifacts were used.

Follow-up queue:

`harness/generated-work-items.json` was left unchanged. This discovery-quality lane is at a true fixed point; the existing queue already contains unrelated deploy-required and credential-required follow-ups.
