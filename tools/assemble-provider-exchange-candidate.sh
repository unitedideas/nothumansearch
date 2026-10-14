#!/bin/bash
# Assemble the exact NHS provider-exchange worktree delta in an isolated clone.
# This never changes the source repository index, branch, refs, or working tree.
set -euo pipefail

EXPECTED_BASE="${1:-267435481c6885bf3d0e035092dfa2ed423ca509}"
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPOSITORY=$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel)
PATH_MANIFEST="$SCRIPT_DIR/provider-exchange-release-paths.txt"
BASE_COMMIT=$(git -C "$REPOSITORY" rev-parse --verify "${EXPECTED_BASE}^{commit}")
CURRENT_HEAD=$(git -C "$REPOSITORY" rev-parse HEAD)

if [ "$CURRENT_HEAD" != "$BASE_COMMIT" ]; then
    echo "source HEAD changed; re-audit the release base before assembling" >&2
    exit 1
fi
if [ ! -s "$PATH_MANIFEST" ]; then
    echo "provider-exchange release path manifest is missing or empty" >&2
    exit 1
fi

ASSEMBLY_DIR=$(mktemp -d "${TMPDIR:-/tmp}/nhs-provider-candidate.XXXXXX")
EXPECTED_PATHS="$ASSEMBLY_DIR/expected-paths.txt"
ACTUAL_PATHS="$ASSEMBLY_DIR/actual-paths.txt"
CANDIDATE="$ASSEMBLY_DIR/repository"

LC_ALL=C /usr/bin/sort -u "$PATH_MANIFEST" >"$EXPECTED_PATHS"
git -C "$REPOSITORY" status --porcelain=v1 --untracked-files=all |
    /usr/bin/awk '{print substr($0,4)}' |
    /usr/bin/awk '$0 !~ /^\.gocache\// {print}' |
    LC_ALL=C /usr/bin/sort -u >"$ACTUAL_PATHS"

if ! /usr/bin/diff -u "$EXPECTED_PATHS" "$ACTUAL_PATHS"; then
    echo "meaningful worktree paths differ from the reviewed NHS release manifest" >&2
    exit 1
fi

while IFS= read -r path; do
    if [ -z "$path" ] || [ ! -f "$REPOSITORY/$path" ]; then
        echo "reviewed release path is missing or not a file: $path" >&2
        exit 1
    fi
done <"$EXPECTED_PATHS"

git clone --quiet --no-hardlinks "$REPOSITORY" "$CANDIDATE"
/usr/bin/rsync -aR --files-from="$EXPECTED_PATHS" "$REPOSITORY/" "$CANDIDATE/"
git -C "$CANDIDATE" config user.name "Codex NHS Release Assembly"
git -C "$CANDIDATE" config user.email "codex-nhs-release@localhost"
git -C "$CANDIDATE" config commit.gpgsign false
git -C "$CANDIDATE" add --all

CANDIDATE_COMMIT_DATE=$(/bin/date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_AUTHOR_DATE="$CANDIDATE_COMMIT_DATE" \
GIT_COMMITTER_DATE="$CANDIDATE_COMMIT_DATE" \
    git -C "$CANDIDATE" commit --quiet \
        -m "Prepare NHS provider-funded action exchange"

CANDIDATE_COMMIT=$(git -C "$CANDIDATE" rev-parse HEAD)
CANDIDATE_PARENT=$(git -C "$CANDIDATE" rev-parse HEAD^)
CANDIDATE_TREE=$(git -C "$CANDIDATE" rev-parse HEAD^{tree})
CHANGED_COUNT=$(git -C "$CANDIDATE" diff-tree --no-commit-id --name-only -r HEAD | /usr/bin/wc -l | /usr/bin/tr -d ' ')

if [ "$CANDIDATE_PARENT" != "$BASE_COMMIT" ]; then
    echo "candidate parent does not match the reviewed release base" >&2
    exit 1
fi
if [ "$CHANGED_COUNT" -ne "$(/usr/bin/wc -l <"$EXPECTED_PATHS" | /usr/bin/tr -d ' ')" ]; then
    echo "candidate changed-path count does not match the release manifest" >&2
    exit 1
fi

echo "NHS provider-exchange candidate assembled. No source-repository state changed."
echo "Candidate repository: $CANDIDATE"
echo "Candidate commit: $CANDIDATE_COMMIT"
echo "Candidate tree: $CANDIDATE_TREE"
echo "Candidate parent: $CANDIDATE_PARENT"
echo "Changed paths: $CHANGED_COUNT"
echo "Next: run tools/prepare-exact-release.sh HEAD HEAD^ inside the candidate repository with two disposable PostgreSQL DSNs."
