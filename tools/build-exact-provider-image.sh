#!/bin/bash
# Build a local NHS provider-exchange image from the exact verified Git archive.
# This command never pushes, deploys, starts a machine, or changes production.
set -euo pipefail

if [ "$#" -ne 5 ] || [ "$5" != "--confirm-owner-authorized" ]; then
    echo "usage: $0 CANDIDATE_REPOSITORY EXACT_RELEASE_MANIFEST IMAGE_TAG OUTPUT_RECEIPT --confirm-owner-authorized" >&2
    exit 2
fi

CANDIDATE_REPOSITORY="$1"
EXACT_RELEASE_MANIFEST="$2"
IMAGE_TAG="$3"
OUTPUT_RECEIPT="$4"

if [ ! -d "$CANDIDATE_REPOSITORY" ] ||
   [ "$(git -C "$CANDIDATE_REPOSITORY" rev-parse --is-inside-work-tree 2>/dev/null || true)" != "true" ]; then
    echo "candidate repository is unavailable" >&2
    exit 1
fi
if [ -n "$(git -C "$CANDIDATE_REPOSITORY" status --porcelain)" ]; then
    echo "candidate repository must be clean" >&2
    exit 1
fi
if [ ! -f "$EXACT_RELEASE_MANIFEST" ]; then
    echo "exact release verification manifest is unavailable" >&2
    exit 1
fi
if ! [[ "$IMAGE_TAG" =~ ^[A-Za-z0-9][A-Za-z0-9._/:+-]{0,199}$ ]] ||
   [[ "$IMAGE_TAG" == *@* ]]; then
    echo "local image tag is invalid" >&2
    exit 2
fi
OUTPUT_DIRECTORY=$(dirname "$OUTPUT_RECEIPT")
if [ -e "$OUTPUT_RECEIPT" ] || [ -L "$OUTPUT_RECEIPT" ] ||
   [ ! -d "$OUTPUT_DIRECTORY" ]; then
    echo "output receipt must be a new file in an existing directory" >&2
    exit 1
fi

manifest_value() {
    local key="$1"
    /usr/bin/awk -F= -v wanted="$key" '
        $1 == wanted { count++; value=substr($0, length($1)+2) }
        END { if (count != 1) exit 1; print value }
    ' "$EXACT_RELEASE_MANIFEST"
}

# os.link invokes link(2) with the exact destination path. Unlike /bin/ln, it
# cannot reinterpret a directory raced into that path as a target directory.
atomic_publish_file_if_absent() {
    /usr/bin/python3 -c \
        'import os, sys; os.link(sys.argv[1], sys.argv[2], follow_symlinks=False)' \
        "$1" "$2" 2>/dev/null
}

CONTRACT=$(manifest_value contract) || exit 1
COMMIT=$(manifest_value release_commit) || exit 1
TREE=$(manifest_value release_tree) || exit 1
PARENT=$(manifest_value release_base_commit) || exit 1
ARCHIVE_SHA256=$(manifest_value source_archive_sha256) || exit 1
BUILD_ARG=$(manifest_value build_arg) || exit 1
ARCHIVE_TESTS=$(manifest_value exact_archive_tests_passed) || exit 1
POSTGRES_TESTS=$(manifest_value postgres_release_tests_passed) || exit 1
RECOVERY_SMOKE=$(manifest_value disabled_recovery_smoke_passed) || exit 1
PREFLIGHT_BOUND=$(manifest_value preflight_binary_revision_bound) || exit 1
SECRET_FINDINGS=$(manifest_value secret_scan_findings) || exit 1
SOURCE_OCI_VERIFIED=$(manifest_value oci_image_digest_verified) || exit 1
SOURCE_TARGET_PREFLIGHT=$(manifest_value target_cutover_preflight_verified) || exit 1
SOURCE_RESTORE_DRILL=$(manifest_value restore_drill_verified) || exit 1
SOURCE_DEPLOYMENT_READY=$(manifest_value deployment_ready) || exit 1
DEPLOYMENT_EMITTED=$(manifest_value deployment_command_emitted) || exit 1

if [ "$CONTRACT" != "nhs-exact-release-verification-v2" ] ||
   ! [[ "$COMMIT" =~ ^[0-9a-f]{40}$ ]] ||
   ! [[ "$TREE" =~ ^[0-9a-f]{40}$ ]] ||
   ! [[ "$PARENT" =~ ^[0-9a-f]{40}$ ]] ||
   ! [[ "$ARCHIVE_SHA256" =~ ^[0-9a-f]{64}$ ]] ||
   [ "$BUILD_ARG" != "RELEASE_REVISION=$COMMIT" ] ||
   [ "$ARCHIVE_TESTS" != "true" ] ||
   [ "$POSTGRES_TESTS" != "true" ] ||
   [ "$RECOVERY_SMOKE" != "true" ] ||
   [ "$PREFLIGHT_BOUND" != "true" ] ||
   [ "$SECRET_FINDINGS" != "0" ] ||
   [ "$SOURCE_OCI_VERIFIED" != "false" ] ||
   [ "$SOURCE_TARGET_PREFLIGHT" != "false" ] ||
   [ "$SOURCE_RESTORE_DRILL" != "false" ] ||
   [ "$SOURCE_DEPLOYMENT_READY" != "false" ] ||
   [ "$DEPLOYMENT_EMITTED" != "false" ]; then
    echo "exact release verification manifest is not an image-build input" >&2
    exit 1
fi

ACTUAL_COMMIT=$(git -C "$CANDIDATE_REPOSITORY" rev-parse --verify "${COMMIT}^{commit}")
ACTUAL_TREE=$(git -C "$CANDIDATE_REPOSITORY" rev-parse "${COMMIT}^{tree}")
ACTUAL_PARENT=$(git -C "$CANDIDATE_REPOSITORY" rev-parse "${COMMIT}^")
if [ "$ACTUAL_COMMIT" != "$COMMIT" ] || [ "$ACTUAL_TREE" != "$TREE" ] ||
   [ "$ACTUAL_PARENT" != "$PARENT" ]; then
    echo "candidate identity does not match exact release verification" >&2
    exit 1
fi

BUILD_BINARY="${NHS_OCI_BUILD_BINARY:-docker}"
if [[ "$BUILD_BINARY" == */* ]]; then
    if [ ! -x "$BUILD_BINARY" ]; then
        echo "OCI build binary is unavailable" >&2
        exit 1
    fi
else
    BUILD_BINARY=$(command -v "$BUILD_BINARY" || true)
    if [ -z "$BUILD_BINARY" ]; then
        echo "OCI build binary is unavailable" >&2
        exit 1
    fi
fi

BUILD_DIR=$(mktemp -d "${TMPDIR:-/tmp}/nhs-provider-image.XXXXXX")
ARCHIVE="$BUILD_DIR/source.tar"
IID_FILE="$BUILD_DIR/image-id"
cleanup_build() {
    /bin/rm -f "$ARCHIVE" "$IID_FILE"
    if [ -n "${RECEIPT_CANDIDATE:-}" ]; then
        /bin/rm -f "$RECEIPT_CANDIDATE"
    fi
    /bin/rmdir "$BUILD_DIR" 2>/dev/null || true
}
trap cleanup_build EXIT
trap 'cleanup_build; exit 1' INT TERM

git -C "$CANDIDATE_REPOSITORY" archive --format=tar --output="$ARCHIVE" "$COMMIT"
ACTUAL_ARCHIVE_SHA256=$(/usr/bin/shasum -a 256 "$ARCHIVE" | /usr/bin/awk '{print $1}')
if [ "$ACTUAL_ARCHIVE_SHA256" != "$ARCHIVE_SHA256" ]; then
    echo "reconstructed source archive does not match exact release verification" >&2
    exit 1
fi

# Docker-compatible builders accept a tar stream as the entire context. This
# prevents a mutable extracted directory from contributing post-verification
# bytes to an image carrying the authorized revision label.
"$BUILD_BINARY" build --pull --no-cache \
    --build-arg "RELEASE_REVISION=$COMMIT" \
    --build-arg "SOURCE_ARCHIVE_SHA256=$ARCHIVE_SHA256" \
    --iidfile "$IID_FILE" --tag "$IMAGE_TAG" - <"$ARCHIVE"

if [ ! -f "$IID_FILE" ]; then
    echo "OCI builder did not produce an image identity" >&2
    exit 1
fi
IMAGE_ID=$(/usr/bin/tr -d '\r\n' <"$IID_FILE")
if ! [[ "$IMAGE_ID" =~ ^sha256:[0-9a-f]{64}$ ]]; then
    echo "OCI builder returned an invalid local image identity" >&2
    exit 1
fi
REVISION_LABEL=$("$BUILD_BINARY" image inspect --format \
    '{{ index .Config.Labels "org.opencontainers.image.revision" }}' "$IMAGE_ID")
SOURCE_LABEL=$("$BUILD_BINARY" image inspect --format \
    '{{ index .Config.Labels "org.opencontainers.image.source_archive_sha256" }}' "$IMAGE_ID")
REVISION_LABEL=$(/usr/bin/printf '%s' "$REVISION_LABEL" | /usr/bin/tr -d '\r\n')
SOURCE_LABEL=$(/usr/bin/printf '%s' "$SOURCE_LABEL" | /usr/bin/tr -d '\r\n')
if [ "$REVISION_LABEL" != "$COMMIT" ] || [ "$SOURCE_LABEL" != "$ARCHIVE_SHA256" ]; then
    echo "built image labels do not match the verified source archive" >&2
    exit 1
fi

RECEIPT_BODY=$(/usr/bin/printf '%s\n' \
    "contract=nhs-provider-local-image-v1" \
    "release_commit=$COMMIT" \
    "release_tree=$TREE" \
    "source_archive_sha256=$ARCHIVE_SHA256" \
    "local_image_id=$IMAGE_ID" \
    "image_revision_label=$REVISION_LABEL" \
    "image_source_archive_label=$SOURCE_LABEL" \
    "registry_digest_verified=false" \
    "target_cutover_preflight_verified=false" \
    "restore_drill_verified=false" \
    "deployment_ready=false" \
    "push_command_emitted=false" \
    "deployment_command_emitted=false")
umask 077
RECEIPT_CANDIDATE=$(mktemp "$OUTPUT_DIRECTORY/.nhs-provider-local-image-receipt.XXXXXX")
/usr/bin/printf '%s\n' "$RECEIPT_BODY" >"$RECEIPT_CANDIDATE"
if ! atomic_publish_file_if_absent "$RECEIPT_CANDIDATE" "$OUTPUT_RECEIPT"; then
    echo "output receipt appeared concurrently; refusing to overwrite" >&2
    exit 1
fi
/bin/rm -f "$RECEIPT_CANDIDATE"
RECEIPT_CANDIDATE=''

/usr/bin/printf '%s\n' "$RECEIPT_BODY"
