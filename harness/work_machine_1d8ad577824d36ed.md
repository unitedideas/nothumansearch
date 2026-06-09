# NHS discovery-quality fixed point - work_machine_1d8ad577824d36ed

Scope: aggregate-only discovery-quality and discovery-quarantine refresh from the completed boundary. No full recrawl, broad crawl, public posting, browser automation, production mutation, or raw row output was used.

Refresh command:

```bash
NHS_DISCOVERY_INPUT=tools/seed-refresh.log PYTHONDONTWRITEBYTECODE=1 ./tools/refresh-discovery-quality.sh
```

Refresh result:

```text
discovery_quality_refresh hard_signal_rows=8526 low_signal_rows=3273 category_other_low_signal=225 quarantine_active=true planner_priority=quarantine_first history=harness/discovery-quarantine-history.jsonl
```

Aggregate state:

- `sample_rows=11799`
- `hard_signal_rows=8526`
- `low_signal_rows=3273`
- `category_other=299`
- `category_other_low_signal=225`
- `category_other_hard_agent_signal=74`
- `llms_only=578`
- `schema_only=596`
- `zero_score=1292`
- `hard_signal_other_review.top_signal_sets`: `API=50`, `API,schema.org=24`

Decision: `no_op_fixed_point`.

Decision matrix:

- Taxonomy-rule change: `false`
- Threshold adjustment: `false`
- No-op fixed point: `true`

Reason: the current `category=other` low-signal aggregate still does not prove a narrow reusable taxonomy rule. Lowering thresholds would promote passive or zero-score rows without hard agent-signal evidence, so the existing hard-signal boundary stays unchanged.

Guard state:

- `llms_only`: audit-only, `public_search=false`, `score_fix_targeting=false`
- `schema_only`: audit-only, `public_search=false`, `score_fix_targeting=false`
- `zero_score`: audit-only, `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: aggregate-review-only, `public_search=false`, `score_fix_targeting=false`

Generated work items: unchanged. This discovery-quality lane is at a true aggregate fixed point, so no new follow-up is warranted from this evidence.

Verification:

```bash
python3 -m json.tool harness/discovery-quality-latest.json
python3 -m json.tool harness/discovery-quarantine-latest.json
PYTHONDONTWRITEBYTECODE=1 python3 tools/quality-gate-discovery.py --quarantine harness/discovery-quarantine-latest.json --repo-root .
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py
```

Verification output:

```text
discovery_quality_gate quarantine_active=true hard_signal_rows=8526 low_signal_rows=3273 category_other_low_signal=225 planner_priority=quarantine_first public_filters=agent_first
Ran 18 tests in 0.025s
OK
```

Commit blocker:

```text
git update-index --no-assume-unchanged harness/discovery-quality-latest.json harness/discovery-quarantine-latest.json harness/discovery-quarantine-history.jsonl harness/generated-work-items.json
fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted
```

Required commit from a git-writable executor:

```bash
git update-index --no-assume-unchanged harness/discovery-quality-latest.json harness/discovery-quarantine-latest.json harness/discovery-quarantine-history.jsonl harness/generated-work-items.json
git add harness/discovery-quality-latest.json harness/discovery-quarantine-latest.json harness/discovery-quarantine-history.jsonl harness/work_machine_1d8ad577824d36ed.md
git commit -m "Refresh discovery quality fixed point"
```
