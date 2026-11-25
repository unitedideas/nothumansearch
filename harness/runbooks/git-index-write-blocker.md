# Git Index Write Blocker

Created: 2026-06-29

QLimit work item `work_machine_4e2bd33fd8263aea` requires clearing
`assume-unchanged` on these harness guard files before further harness
contract edits:

- `harness/README.md`
- `.github/workflows/agentic-readiness.yml`
- `tools/verify-harness-local.sh`
- `tools/test-harness-verifier-contract.py`
- `harness/generated-work-items.json`

The executor attempted:

```bash
git update-index --no-assume-unchanged harness/README.md .github/workflows/agentic-readiness.yml tools/verify-harness-local.sh tools/test-harness-verifier-contract.py harness/generated-work-items.json
```

Git failed before changing the index:

```text
fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted
```

Direct `.git` write probe also failed:

```text
touch: .git/codex-write-test: Operation not permitted
```

Current `git ls-files -v` still reports the target files with `h`, including
`ops/sweeper/planner-inbox.jsonl`. The planner inbox was not touched.

Next action for a runner with `.git/index` write access:

1. Clear `assume-unchanged` for the five harness files above, not
   `ops/sweeper/planner-inbox.jsonl`.
2. Run `git status --short --ignored=no` and confirm harness guard edits are
   visible normally.
3. Only then make further harness contract changes and update
   `harness/generated-work-items.json`.
