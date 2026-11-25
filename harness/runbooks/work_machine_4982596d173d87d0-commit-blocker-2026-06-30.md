# work_machine_4982596d173d87d0 commit blocker

## Scope

QLimit WorkItem `work_machine_4982596d173d87d0` asked for local-only harness contract guards:

- `harness/README.md` must document `tools/verify-harness-local.sh`.
- `.github/workflows/agentic-readiness.yml` must invoke `tools/verify-harness-local.sh` for the harness verification job.
- Verification must stay local-only and must not call production endpoints, require secrets, or inspect raw crawl rows.

## Current local state

`tools/test-harness-verifier-contract.py` is present and checks:

- `tools/verify-harness-local.sh` runs the contract test.
- `harness/README.md` documents `tools/verify-harness-local.sh` under `Local harness verification` with the shell command block.
- `.github/workflows/agentic-readiness.yml` contains `discovery-quality-gate` and its `Run local harness verification` step invokes `tools/verify-harness-local.sh`.
- The workflow path filters include the README, workflow, verifier script, and contract test.

`tools/verify-harness-local.sh` runs the contract test along with the existing aggregate-only local harness tests.

## Verification

Local-only verification passed on 2026-06-30:

```text
tools/verify-harness-local.sh
Ran 31 tests in 0.345s
OK
```

The verification used only repo-local fixtures and scripts. It did not call production endpoints, require secrets, or inspect raw crawl rows.

## Blocker

The required commit cannot be created in this executor because git cannot create the index lock:

```text
git update-index --no-assume-unchanged -- .github/workflows/agentic-readiness.yml harness/README.md harness/generated-work-items.json tools/test-harness-verifier-contract.py tools/verify-harness-local.sh
fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted
```

`git ls-files -v` still reports these paths as assume-unchanged:

```text
h .github/workflows/agentic-readiness.yml
h harness/README.md
h harness/generated-work-items.json
h tools/test-harness-verifier-contract.py
h tools/verify-harness-local.sh
```

## Next action

Run in an executor where `.git/index.lock` can be created:

```bash
git update-index --no-assume-unchanged -- \
  .github/workflows/agentic-readiness.yml \
  harness/README.md \
  harness/generated-work-items.json \
  tools/test-harness-verifier-contract.py \
  tools/verify-harness-local.sh
git status --short --ignored=no
tools/verify-harness-local.sh
git add \
  .github/workflows/agentic-readiness.yml \
  harness/README.md \
  harness/generated-work-items.json \
  harness/runbooks/work_machine_4982596d173d87d0-commit-blocker-2026-06-30.md \
  tools/test-harness-verifier-contract.py \
  tools/verify-harness-local.sh
git commit -m "Guard local harness verifier contract"
```

Do not touch `ops/sweeper/planner-inbox.jsonl`.
