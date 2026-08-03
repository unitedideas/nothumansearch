#!/bin/bash
# Import one exact, already-verified NHS candidate into a durable namespaced ref.
# This command never deploys and requires an explicit owner authorization flag.
set -euo pipefail

if [ "$#" -ne 6 ] || [ "$6" != "--confirm-owner-authorized" ]; then
    echo "usage: $0 CANDIDATE_REPOSITORY EXPECTED_COMMIT EXPECTED_TREE EXPECTED_PARENT EXACT_RELEASE_MANIFEST --confirm-owner-authorized" >&2
    exit 2
fi

CANDIDATE_REPOSITORY="$1"
EXPECTED_COMMIT="$2"
EXPECTED_TREE="$3"
EXPECTED_PARENT="$4"
EXACT_RELEASE_MANIFEST="$5"
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
CANONICAL_REPOSITORY=$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel)

for value in "$EXPECTED_COMMIT" "$EXPECTED_TREE" "$EXPECTED_PARENT"; do
    if ! [[ "$value" =~ ^[0-9a-f]{40}$ ]]; then
        echo "candidate identities must be full lowercase Git object IDs" >&2
        exit 2
    fi
done
if [ ! -f "$EXACT_RELEASE_MANIFEST" ]; then
    echo "exact release verification manifest is unavailable" >&2
    exit 1
fi

manifest_value() {
    local key="$1"
    /usr/bin/awk -F= -v wanted="$key" '
        $1 == wanted { count++; value=substr($0, length($1)+2) }
        END { if (count != 1) exit 1; print value }
    ' "$EXACT_RELEASE_MANIFEST"
}

# Every source is staged in the destination directory. os.link invokes link(2)
# with the exact destination; it cannot overwrite or reinterpret that path.
atomic_publish_file_if_absent() {
    /usr/bin/python3 -c \
        'import os, sys; os.link(sys.argv[1], sys.argv[2], follow_symlinks=False)' \
        "$1" "$2" 2>/dev/null
}
VERIFICATION_CONTRACT=$(manifest_value contract) || {
    echo "exact release verification manifest is malformed" >&2
    exit 1
}
VERIFICATION_COMMIT=$(manifest_value release_commit) || exit 1
VERIFICATION_TREE=$(manifest_value release_tree) || exit 1
VERIFICATION_PARENT=$(manifest_value release_base_commit) || exit 1
VERIFICATION_CHANGED_COUNT=$(manifest_value changed_path_count) || exit 1
VERIFICATION_BUILD_ARG=$(manifest_value build_arg) || exit 1
VERIFICATION_ARCHIVE_SHA256=$(manifest_value source_archive_sha256) || exit 1
VERIFICATION_MIGRATION_019=$(manifest_value migration_019_sha256) || exit 1
VERIFICATION_MIGRATION_020=$(manifest_value migration_020_sha256) || exit 1
VERIFICATION_MIGRATION_021=$(manifest_value migration_021_sha256) || exit 1
VERIFICATION_MIGRATION_022=$(manifest_value migration_022_sha256) || exit 1
VERIFICATION_MIGRATION_023=$(manifest_value migration_023_sha256) || exit 1
VERIFICATION_MIGRATION_024=$(manifest_value migration_024_sha256) || exit 1
VERIFICATION_MIGRATION_025=$(manifest_value migration_025_sha256) || exit 1
VERIFICATION_MIGRATION_026=$(manifest_value migration_026_sha256) || exit 1
VERIFICATION_MIGRATION_027=$(manifest_value migration_027_sha256) || exit 1
VERIFICATION_MIGRATION_028=$(manifest_value migration_028_sha256) || exit 1
VERIFICATION_MIGRATION_029=$(manifest_value migration_029_sha256) || exit 1
VERIFICATION_ARCHIVE_TESTS=$(manifest_value exact_archive_tests_passed) || exit 1
VERIFICATION_POSTGRES_TESTS=$(manifest_value postgres_release_tests_passed) || exit 1
VERIFICATION_RECOVERY_SMOKE=$(manifest_value disabled_recovery_smoke_passed) || exit 1
VERIFICATION_PREFLIGHT_BOUND=$(manifest_value preflight_binary_revision_bound) || exit 1
VERIFICATION_SECRET_FINDINGS=$(manifest_value secret_scan_findings) || exit 1
VERIFICATION_OCI_IMAGE=$(manifest_value oci_image_digest_verified) || exit 1
VERIFICATION_TARGET_PREFLIGHT=$(manifest_value target_cutover_preflight_verified) || exit 1
VERIFICATION_RESTORE_DRILL=$(manifest_value restore_drill_verified) || exit 1
VERIFICATION_DEPLOYMENT_READY=$(manifest_value deployment_ready) || exit 1
VERIFICATION_DEPLOYMENT_EMITTED=$(manifest_value deployment_command_emitted) || exit 1
if [ "$VERIFICATION_CONTRACT" != "nhs-exact-release-verification-v2" ] ||
   [ "$VERIFICATION_COMMIT" != "$EXPECTED_COMMIT" ] ||
   [ "$VERIFICATION_TREE" != "$EXPECTED_TREE" ] ||
   [ "$VERIFICATION_PARENT" != "$EXPECTED_PARENT" ] ||
   [ "$VERIFICATION_BUILD_ARG" != "RELEASE_REVISION=$EXPECTED_COMMIT" ] ||
   [ "$VERIFICATION_ARCHIVE_TESTS" != "true" ] ||
   [ "$VERIFICATION_POSTGRES_TESTS" != "true" ] ||
   [ "$VERIFICATION_RECOVERY_SMOKE" != "true" ] ||
   [ "$VERIFICATION_PREFLIGHT_BOUND" != "true" ] ||
   [ "$VERIFICATION_SECRET_FINDINGS" != "0" ] ||
   [ "$VERIFICATION_OCI_IMAGE" != "false" ] ||
   [ "$VERIFICATION_TARGET_PREFLIGHT" != "false" ] ||
   [ "$VERIFICATION_RESTORE_DRILL" != "false" ] ||
   [ "$VERIFICATION_DEPLOYMENT_READY" != "false" ] ||
   [ "$VERIFICATION_DEPLOYMENT_EMITTED" != "false" ] ||
   ! [[ "$VERIFICATION_CHANGED_COUNT" =~ ^[1-9][0-9]*$ ]] ||
   ! [[ "$VERIFICATION_ARCHIVE_SHA256" =~ ^[0-9a-f]{64}$ ]]; then
    echo "exact release verification manifest does not authorize this candidate identity" >&2
    exit 1
fi
for migration_digest in \
    "$VERIFICATION_MIGRATION_019" "$VERIFICATION_MIGRATION_020" \
    "$VERIFICATION_MIGRATION_021" "$VERIFICATION_MIGRATION_022" \
    "$VERIFICATION_MIGRATION_023" "$VERIFICATION_MIGRATION_024" \
    "$VERIFICATION_MIGRATION_025" "$VERIFICATION_MIGRATION_026" \
    "$VERIFICATION_MIGRATION_027" "$VERIFICATION_MIGRATION_028" \
    "$VERIFICATION_MIGRATION_029"; do
    if ! [[ "$migration_digest" =~ ^[0-9a-f]{64}$ ]]; then
        echo "exact release verification manifest has an invalid migration digest" >&2
        exit 1
    fi
done
VERIFICATION_MANIFEST_SHA256=$(/usr/bin/shasum -a 256 "$EXACT_RELEASE_MANIFEST" | /usr/bin/awk '{print $1}')
if [ ! -d "$CANDIDATE_REPOSITORY" ] ||
   [ "$(git -C "$CANDIDATE_REPOSITORY" rev-parse --is-inside-work-tree 2>/dev/null || true)" != "true" ]; then
    echo "candidate repository is unavailable" >&2
    exit 1
fi
if [ -n "$(git -C "$CANDIDATE_REPOSITORY" status --porcelain)" ]; then
    echo "candidate repository must be clean" >&2
    exit 1
fi

ACTUAL_COMMIT=$(git -C "$CANDIDATE_REPOSITORY" rev-parse --verify "${EXPECTED_COMMIT}^{commit}")
ACTUAL_TREE=$(git -C "$CANDIDATE_REPOSITORY" rev-parse "${EXPECTED_COMMIT}^{tree}")
ACTUAL_PARENT=$(git -C "$CANDIDATE_REPOSITORY" rev-parse "${EXPECTED_COMMIT}^")
CANONICAL_PARENT=$(git -C "$CANONICAL_REPOSITORY" rev-parse HEAD)
if [ "$ACTUAL_COMMIT" != "$EXPECTED_COMMIT" ] ||
   [ "$ACTUAL_TREE" != "$EXPECTED_TREE" ] ||
   [ "$ACTUAL_PARENT" != "$EXPECTED_PARENT" ] ||
   [ "$CANONICAL_PARENT" != "$EXPECTED_PARENT" ]; then
    echo "candidate commit, tree, parent, or canonical base does not match" >&2
    exit 1
fi
ACTUAL_CHANGED_COUNT=$(git -C "$CANDIDATE_REPOSITORY" diff-tree \
    --no-commit-id --name-only -r "$EXPECTED_COMMIT" | /usr/bin/wc -l | /usr/bin/tr -d ' ')
if [ "$ACTUAL_CHANGED_COUNT" != "$VERIFICATION_CHANGED_COUNT" ]; then
    echo "candidate changed-path count does not match exact release verification" >&2
    exit 1
fi

# Recompute the archive and every protected-migration digest from the exact
# candidate object. A copied or hand-written green manifest cannot authorize a
# different source archive, omit a later protected migration, or relabel one.
VERIFY_DIR=$(mktemp -d "${TMPDIR:-/tmp}/nhs-candidate-adoption.XXXXXX")
VERIFY_ARCHIVE="$VERIFY_DIR/source.tar"
BUNDLE_STAGING_DIR=''
BUNDLE_CANDIDATE=''
VERIFICATION_CANDIDATE=''
MANIFEST_CANDIDATE=''
cleanup_verification() {
    /bin/rm -f "$VERIFY_ARCHIVE"
    if [ -n "${BUNDLE_CANDIDATE:-}" ]; then
        /bin/rm -f "$BUNDLE_CANDIDATE"
    fi
    if [ -n "${BUNDLE_STAGING_DIR:-}" ]; then
        /bin/rmdir "$BUNDLE_STAGING_DIR" 2>/dev/null || true
    fi
    if [ -n "${VERIFICATION_CANDIDATE:-}" ]; then
        /bin/rm -f "$VERIFICATION_CANDIDATE"
    fi
    if [ -n "${MANIFEST_CANDIDATE:-}" ]; then
        /bin/rm -f "$MANIFEST_CANDIDATE"
    fi
    /bin/rmdir "$VERIFY_DIR" 2>/dev/null || true
}
trap cleanup_verification EXIT
trap 'exit 1' INT TERM
git -C "$CANDIDATE_REPOSITORY" archive --format=tar \
    --output="$VERIFY_ARCHIVE" "$EXPECTED_COMMIT"
ACTUAL_ARCHIVE_SHA256=$(/usr/bin/shasum -a 256 "$VERIFY_ARCHIVE" | /usr/bin/awk '{print $1}')
if [ "$ACTUAL_ARCHIVE_SHA256" != "$VERIFICATION_ARCHIVE_SHA256" ]; then
    echo "candidate source archive does not match exact release verification" >&2
    exit 1
fi
verify_migration_digest() {
    local path="$1"
    local expected="$2"
    local actual
    actual=$(git -C "$CANDIDATE_REPOSITORY" show \
        "$EXPECTED_COMMIT:$path" | /usr/bin/shasum -a 256 | /usr/bin/awk '{print $1}')
    if [ "$actual" != "$expected" ]; then
        echo "candidate migration digest does not match: $path" >&2
        exit 1
    fi
}
verify_migration_digest migrations/019_provider_exchange.sql "$VERIFICATION_MIGRATION_019"
verify_migration_digest migrations/020_action_interest_receipts.sql "$VERIFICATION_MIGRATION_020"
verify_migration_digest migrations/021_provider_capacity_reservations.sql "$VERIFICATION_MIGRATION_021"
verify_migration_digest migrations/022_provider_commercial_proof.sql "$VERIFICATION_MIGRATION_022"
verify_migration_digest migrations/023_provider_controlled_intent_disclosure.sql "$VERIFICATION_MIGRATION_023"
verify_migration_digest migrations/024_provider_pilot_boundary.sql "$VERIFICATION_MIGRATION_024"
verify_migration_digest migrations/025_stage1_fact_integrity.sql "$VERIFICATION_MIGRATION_025"
verify_migration_digest migrations/026_provider_pilot_proof_integrity.sql "$VERIFICATION_MIGRATION_026"
verify_migration_digest migrations/027_provider_pilot_review_evidence.sql "$VERIFICATION_MIGRATION_027"
verify_migration_digest migrations/028_provider_commercial_proof_manifest.sql "$VERIFICATION_MIGRATION_028"
verify_migration_digest migrations/029_provider_settlement_receipts.sql "$VERIFICATION_MIGRATION_029"

REF="refs/nhs-provider-candidates/$EXPECTED_COMMIT"
EXISTING_REF=$(git -C "$CANONICAL_REPOSITORY" rev-parse --verify "$REF" 2>/dev/null || true)
if [ -n "$EXISTING_REF" ] && [ "$EXISTING_REF" != "$EXPECTED_COMMIT" ]; then
    echo "candidate ref already points to a different object" >&2
    exit 1
fi
if [ -z "$EXISTING_REF" ]; then
    git -C "$CANONICAL_REPOSITORY" fetch --quiet --no-tags --no-write-fetch-head \
        "$CANDIDATE_REPOSITORY" "$EXPECTED_COMMIT"
    if ! git -C "$CANONICAL_REPOSITORY" update-ref \
        "$REF" "$EXPECTED_COMMIT" "" 2>/dev/null; then
        EXISTING_REF=$(git -C "$CANONICAL_REPOSITORY" rev-parse \
            --verify "$REF" 2>/dev/null || true)
        if [ "$EXISTING_REF" != "$EXPECTED_COMMIT" ]; then
            echo "candidate ref appeared concurrently with a different object" >&2
            exit 1
        fi
    fi
fi

GIT_DIR=$(git -C "$CANONICAL_REPOSITORY" rev-parse --absolute-git-dir)
ARTIFACT_DIR="$GIT_DIR/nhs-provider-candidates"
BUNDLE="$ARTIFACT_DIR/$EXPECTED_COMMIT.bundle"
MANIFEST="$ARTIFACT_DIR/$EXPECTED_COMMIT.manifest"
VERIFICATION_COPY="$ARTIFACT_DIR/$EXPECTED_COMMIT.release-manifest"
if [ -L "$ARTIFACT_DIR" ]; then
    echo "candidate artifact directory must not be a symbolic link" >&2
    exit 1
fi
umask 077
mkdir -p "$ARTIFACT_DIR"
if [ -L "$ARTIFACT_DIR" ] || [ ! -d "$ARTIFACT_DIR" ]; then
    echo "candidate artifact directory must be a real directory" >&2
    exit 1
fi

if [ -L "$BUNDLE" ]; then
    echo "durable candidate bundle must not be a symbolic link" >&2
    exit 1
fi
if [ ! -e "$BUNDLE" ]; then
    BUNDLE_STAGING_DIR=$(mktemp -d \
        "$ARTIFACT_DIR/.bundle-$EXPECTED_COMMIT.XXXXXX")
    BUNDLE_CANDIDATE="$BUNDLE_STAGING_DIR/candidate.bundle"
    git -C "$CANONICAL_REPOSITORY" bundle create "$BUNDLE_CANDIDATE" "$REF"
    if ! atomic_publish_file_if_absent "$BUNDLE_CANDIDATE" "$BUNDLE"; then
        if [ -L "$BUNDLE" ]; then
            echo "durable candidate bundle appeared as a symbolic link" >&2
            exit 1
        fi
        if [ ! -f "$BUNDLE" ]; then
            echo "candidate bundle appeared concurrently in an invalid form" >&2
            exit 1
        fi
    fi
    /bin/rm -f "$BUNDLE_CANDIDATE"
    /bin/rmdir "$BUNDLE_STAGING_DIR"
    BUNDLE_CANDIDATE=''
    BUNDLE_STAGING_DIR=''
fi
if [ -L "$BUNDLE" ] || [ ! -f "$BUNDLE" ]; then
    echo "durable candidate bundle must be a regular non-symlink file" >&2
    exit 1
fi
if ! git -C "$CANONICAL_REPOSITORY" bundle verify "$BUNDLE" >/dev/null 2>&1; then
    echo "durable candidate bundle failed verification" >&2
    exit 1
fi
if ! git -C "$CANONICAL_REPOSITORY" bundle list-heads "$BUNDLE" | /usr/bin/grep -Fqx "$EXPECTED_COMMIT $REF"; then
    echo "durable candidate bundle does not contain the exact namespaced ref" >&2
    exit 1
fi

if [ -L "$VERIFICATION_COPY" ]; then
    echo "exact release manifest copy must not be a symbolic link" >&2
    exit 1
fi
if [ ! -e "$VERIFICATION_COPY" ]; then
    VERIFICATION_CANDIDATE=$(mktemp \
        "$ARTIFACT_DIR/.release-manifest-$EXPECTED_COMMIT.XXXXXX")
    /bin/cp "$EXACT_RELEASE_MANIFEST" "$VERIFICATION_CANDIDATE"
    if [ "$(/usr/bin/shasum -a 256 "$VERIFICATION_CANDIDATE" | /usr/bin/awk '{print $1}')" != "$VERIFICATION_MANIFEST_SHA256" ]; then
        echo "exact release manifest changed while its durable copy was prepared" >&2
        exit 1
    fi
    if ! atomic_publish_file_if_absent \
        "$VERIFICATION_CANDIDATE" "$VERIFICATION_COPY"; then
        if [ -L "$VERIFICATION_COPY" ]; then
            echo "exact release manifest appeared as a symbolic link" >&2
            exit 1
        fi
        if [ ! -f "$VERIFICATION_COPY" ]; then
            echo "exact release manifest appeared concurrently in an invalid form" >&2
            exit 1
        fi
    fi
    /bin/rm -f "$VERIFICATION_CANDIDATE"
    VERIFICATION_CANDIDATE=''
fi
if [ -L "$VERIFICATION_COPY" ] || [ ! -f "$VERIFICATION_COPY" ]; then
    echo "exact release manifest copy must be a regular non-symlink file" >&2
    exit 1
fi
if [ "$(/usr/bin/shasum -a 256 "$VERIFICATION_COPY" | /usr/bin/awk '{print $1}')" != "$VERIFICATION_MANIFEST_SHA256" ]; then
    echo "existing exact release manifest does not match the verified receipt" >&2
    exit 1
fi

BUNDLE_SHA256=$(/usr/bin/shasum -a 256 "$BUNDLE" | /usr/bin/awk '{print $1}')
MANIFEST_BODY=$(printf '%s\n' \
    "contract=nhs-provider-candidate-adoption-v2" \
    "candidate_commit=$EXPECTED_COMMIT" \
    "candidate_tree=$EXPECTED_TREE" \
    "candidate_parent=$EXPECTED_PARENT" \
    "canonical_ref=$REF" \
    "bundle_sha256=$BUNDLE_SHA256" \
    "exact_release_contract=$VERIFICATION_CONTRACT" \
    "release_verification_manifest_sha256=$VERIFICATION_MANIFEST_SHA256" \
    "source_archive_sha256=$VERIFICATION_ARCHIVE_SHA256" \
    "oci_image_digest_verified=false" \
    "target_cutover_preflight_verified=false" \
    "restore_drill_verified=false" \
    "deployment_ready=false" \
    "deployment_command_emitted=false")
if [ -L "$MANIFEST" ]; then
    echo "candidate manifest must not be a symbolic link" >&2
    exit 1
fi
if [ ! -e "$MANIFEST" ]; then
    MANIFEST_CANDIDATE=$(mktemp \
        "$ARTIFACT_DIR/.manifest-$EXPECTED_COMMIT.XXXXXX")
    printf '%s\n' "$MANIFEST_BODY" > "$MANIFEST_CANDIDATE"
    if ! atomic_publish_file_if_absent "$MANIFEST_CANDIDATE" "$MANIFEST"; then
        if [ -L "$MANIFEST" ]; then
            echo "candidate manifest appeared as a symbolic link" >&2
            exit 1
        fi
        if [ ! -f "$MANIFEST" ]; then
            echo "candidate manifest appeared concurrently in an invalid form" >&2
            exit 1
        fi
    fi
    /bin/rm -f "$MANIFEST_CANDIDATE"
    MANIFEST_CANDIDATE=''
fi
if [ -L "$MANIFEST" ] || [ ! -f "$MANIFEST" ]; then
    echo "candidate manifest must be a regular non-symlink file" >&2
    exit 1
fi
if [ "$(/bin/cat "$MANIFEST")" != "$MANIFEST_BODY" ]; then
    echo "existing candidate manifest does not match the exact bundle" >&2
    exit 1
fi

printf 'candidate_adopted=true\n'
printf 'candidate_commit=%s\n' "$EXPECTED_COMMIT"
printf 'candidate_tree=%s\n' "$EXPECTED_TREE"
printf 'candidate_parent=%s\n' "$EXPECTED_PARENT"
printf 'canonical_ref=%s\n' "$REF"
printf 'bundle_sha256=%s\n' "$BUNDLE_SHA256"
printf 'release_verification_manifest_sha256=%s\n' "$VERIFICATION_MANIFEST_SHA256"
printf 'release_verification_manifest=%s\n' "$VERIFICATION_COPY"
printf 'manifest=%s\n' "$MANIFEST"
printf 'deployment_ready=false\n'
printf 'deployment_command_emitted=false\n'
