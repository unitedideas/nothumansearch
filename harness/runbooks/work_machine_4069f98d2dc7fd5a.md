# WorkItem work_machine_4069f98d2dc7fd5a

Status: implemented locally, verified, commit blocked by local git index write permissions.

Changes made:

- Tightened `tools/test-harness-verifier-contract.py` with a local-only assertion that `.github/workflows/agentic-readiness.yml` triggers on harness contract file changes, including `tools/verify-harness-local.sh` and `tools/test-harness-verifier-contract.py`.
- Updated `harness/generated-work-items.json` by removing the now-closed workflow-path guard follow-up and adding a narrower follow-up for separating production-facing CI documentation from local-only harness verification.

Verification:

```text
tools/verify-harness-local.sh
Ran 31 tests in 0.361s
OK
```

Local-only scope held: the verifier used fixtures and repository files only. It did not call production endpoints, require secrets, or inspect raw crawl rows.

Commit blocker:

```text
git update-index --no-assume-unchanged tools/test-harness-verifier-contract.py harness/generated-work-items.json
fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted
```

`git ls-files -v` still reports both edited files with `h`, so `git status` hides the changes. This is the same blocker documented in `harness/runbooks/git-index-write-blocker.md`.

Next action for a runner with `.git/index` write access:

```bash
git update-index --no-assume-unchanged tools/test-harness-verifier-contract.py harness/generated-work-items.json
git add tools/test-harness-verifier-contract.py harness/generated-work-items.json harness/runbooks/work_machine_4069f98d2dc7fd5a.md
git commit -m "Guard local harness verifier contract"
```
