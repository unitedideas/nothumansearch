#!/bin/bash
# Prepare and verify an immutable Not Human Search release context.
# This command never deploys. Deployment remains owner-authorized.
set -euo pipefail

REF="${1:-HEAD}"
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPOSITORY=$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel)
COMMIT=$(git -C "$REPOSITORY" rev-parse --verify "${REF}^{commit}")
TREE=$(git -C "$REPOSITORY" rev-parse "${COMMIT}^{tree}")

if ! [[ "$COMMIT" =~ ^[0-9a-f]{40}$ ]]; then
    echo "release ref did not resolve to a full Git commit" >&2
    exit 1
fi

RELEASE_DIR=$(mktemp -d "${TMPDIR:-/tmp}/nhs-release-${COMMIT:0:12}.XXXXXX")
ARCHIVE="$RELEASE_DIR/source.tar"
CONTEXT="$RELEASE_DIR/context"
MANIFEST="$RELEASE_DIR/release-manifest.txt"
mkdir -p "$CONTEXT"

git -C "$REPOSITORY" archive --format=tar --output="$ARCHIVE" "$COMMIT"
tar -xf "$ARCHIVE" -C "$CONTEXT"

if ! grep -Fxq "$COMMIT" "$CONTEXT/release-source-revision"; then
    echo "archived source revision does not match requested commit" >&2
    exit 1
fi

ARCHIVE_SHA=$(/usr/bin/shasum -a 256 "$ARCHIVE" | /usr/bin/awk '{print $1}')
MIGRATION_019_SHA=$(/usr/bin/shasum -a 256 "$CONTEXT/migrations/019_provider_exchange.sql" | /usr/bin/awk '{print $1}')
MIGRATION_020_SHA=$(/usr/bin/shasum -a 256 "$CONTEXT/migrations/020_action_interest_receipts.sql" | /usr/bin/awk '{print $1}')

GO_BINARY="${NHS_GO_BINARY:-/Users/shane/.local/bin/go}"
if [ ! -x "$GO_BINARY" ]; then
    echo "Go toolchain not found at $GO_BINARY" >&2
    exit 1
fi

(
    cd "$CONTEXT"
    "$GO_BINARY" test ./...
    "$GO_BINARY" test -race ./internal/database ./internal/handlers ./internal/models ./internal/providerexchange ./cmd/server ./cmd/crawler
    "$GO_BINARY" vet ./...
    "$GO_BINARY" build ./...
    /Users/shane/.local/bin/codex-secret scan .
)

{
    echo "release_commit=$COMMIT"
    echo "release_tree=$TREE"
    echo "migration_019_sha256=$MIGRATION_019_SHA"
    echo "migration_020_sha256=$MIGRATION_020_SHA"
    echo "source_archive_sha256=$ARCHIVE_SHA"
    echo "source_context=$CONTEXT"
} > "$MANIFEST"

echo "Exact NHS release context verified. No deployment was performed."
echo "Manifest: $MANIFEST"
echo "Candidate commit: $COMMIT"
echo "Owner-authorized deploy command:"
printf '  /Users/shane/.local/bin/codex-secret run --env FLY_API_TOKEN=FLY_API_TOKEN -- /Users/shane/.fly/bin/flyctl deploy %q --config %q --dockerfile %q --build-arg %q --remote-only\n' \
    "$CONTEXT" "$CONTEXT/fly.toml" "$CONTEXT/Dockerfile" "RELEASE_REVISION=$COMMIT"
