# Harness verifier contract guard commit blocker

WorkItem: `work_machine_4faee922c90d5467`

## Completed locally

- `tools/test-harness-verifier-contract.py` now asserts that `tools/verify-harness-local.sh` runs the contract test.
- `tools/test-harness-verifier-contract.py` fails if `harness/README.md` stops documenting `tools/verify-harness-local.sh` under `Local harness verification` with the runnable bash block.
- `tools/test-harness-verifier-contract.py` fails if `.github/workflows/agentic-readiness.yml` stops invoking `tools/verify-harness-local.sh` in the `discovery-quality-gate` job.
- `.github/workflows/agentic-readiness.yml` includes harness contract paths in the push trigger.
- `harness/generated-work-items.json` was updated to remove the completed assume-unchanged follow-up, leaving the remaining live follow-ups.

## Verification

Command:

```bash
tools/verify-harness-local.sh
```

Result:

```text
Ran 31 tests in 0.361s
OK
discovery_quality_gate quarantine_active=true hard_signal_rows=3 low_signal_rows=8 category_other_low_signal=7 planner_priority=quarantine_first public_filters=agent_first
```

The guard remains local-only. It uses fixture/script tests and does not call production endpoints, require secrets, or inspect raw crawl rows.

## Commit blocker

Attempted command:

```bash
git update-index --no-assume-unchanged -- .github/workflows/agentic-readiness.yml harness/README.md harness/generated-work-items.json tools/test-harness-verifier-contract.py tools/verify-harness-local.sh
```

Observed failure:

```text
fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted
```

Current hidden index bits remain:

```text
h .github/workflows/agentic-readiness.yml
h harness/README.md
h harness/generated-work-items.json
h tools/test-harness-verifier-contract.py
h tools/verify-harness-local.sh
```

## Next action

Run in an executor that can create `.git/index.lock`, then:

```bash
git update-index --no-assume-unchanged -- .github/workflows/agentic-readiness.yml harness/README.md harness/generated-work-items.json tools/test-harness-verifier-contract.py tools/verify-harness-local.sh
git status --short --ignored=no
tools/verify-harness-local.sh
git add .github/workflows/agentic-readiness.yml harness/README.md harness/generated-work-items.json tools/test-harness-verifier-contract.py tools/verify-harness-local.sh harness/runbooks/work_machine_4faee922c90d5467-commit-blocker-2026-06-30.md
git commit -m "Guard local harness verifier contract"
```
