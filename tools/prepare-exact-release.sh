#!/bin/bash
# Prepare and verify an immutable Not Human Search release context.
# This command never deploys. Deployment remains owner-authorized.
set -euo pipefail

REF="${1:-HEAD}"
BASE_REF="${2:-${REF}^}"
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPOSITORY=$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel)
COMMIT=$(git -C "$REPOSITORY" rev-parse --verify "${REF}^{commit}")
BASE_COMMIT=$(git -C "$REPOSITORY" rev-parse --verify "${BASE_REF}^{commit}")
TREE=$(git -C "$REPOSITORY" rev-parse "${COMMIT}^{tree}")

if ! [[ "$COMMIT" =~ ^[0-9a-f]{40}$ ]]; then
    echo "release ref did not resolve to a full Git commit" >&2
    exit 1
fi
if ! git -C "$REPOSITORY" merge-base --is-ancestor "$BASE_COMMIT" "$COMMIT"; then
    echo "release base must be an ancestor of the candidate commit" >&2
    exit 1
fi

: "${NHS_TEST_POSTGRES_DSN:?set NHS_TEST_POSTGRES_DSN to an isolated disposable PostgreSQL database}"
: "${NHS_MIGRATION_TEST_POSTGRES_DSN:?set NHS_MIGRATION_TEST_POSTGRES_DSN to a second isolated disposable PostgreSQL database}"
if [ "$NHS_TEST_POSTGRES_DSN" = "$NHS_MIGRATION_TEST_POSTGRES_DSN" ]; then
    echo "provider-model and migration-ledger tests require two different disposable PostgreSQL databases" >&2
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

CHANGED_PATHS=()
CHANGED_GO_PATHS=()
while IFS= read -r -d '' path; do
    if [ ! -f "$CONTEXT/$path" ]; then
        echo "changed release path is missing from archive: $path" >&2
        exit 1
    fi
    CHANGED_PATHS+=("$CONTEXT/$path")
    case "$path" in
        *.go) CHANGED_GO_PATHS+=("$CONTEXT/$path") ;;
    esac
done < <(git -C "$REPOSITORY" diff --name-only --diff-filter=ACMR -z "$BASE_COMMIT" "$COMMIT")

if [ "${#CHANGED_PATHS[@]}" -eq 0 ]; then
    echo "candidate has no added, copied, modified, or renamed paths relative to its release base" >&2
    exit 1
fi

if ! git -C "$REPOSITORY" diff --check "$BASE_COMMIT" "$COMMIT"; then
    echo "candidate diff failed whitespace validation" >&2
    exit 1
fi

ARCHIVE_SHA=$(/usr/bin/shasum -a 256 "$ARCHIVE" | /usr/bin/awk '{print $1}')
MIGRATION_019_SHA=$(/usr/bin/shasum -a 256 "$CONTEXT/migrations/019_provider_exchange.sql" | /usr/bin/awk '{print $1}')
MIGRATION_020_SHA=$(/usr/bin/shasum -a 256 "$CONTEXT/migrations/020_action_interest_receipts.sql" | /usr/bin/awk '{print $1}')
MIGRATION_021_SHA=$(/usr/bin/shasum -a 256 "$CONTEXT/migrations/021_provider_capacity_reservations.sql" | /usr/bin/awk '{print $1}')
MIGRATION_022_SHA=$(/usr/bin/shasum -a 256 "$CONTEXT/migrations/022_provider_commercial_proof.sql" | /usr/bin/awk '{print $1}')
MIGRATION_023_SHA=$(/usr/bin/shasum -a 256 "$CONTEXT/migrations/023_provider_controlled_intent_disclosure.sql" | /usr/bin/awk '{print $1}')
MIGRATION_024_SHA=$(/usr/bin/shasum -a 256 "$CONTEXT/migrations/024_provider_pilot_boundary.sql" | /usr/bin/awk '{print $1}')
MIGRATION_025_SHA=$(/usr/bin/shasum -a 256 "$CONTEXT/migrations/025_stage1_fact_integrity.sql" | /usr/bin/awk '{print $1}')
MIGRATION_026_SHA=$(/usr/bin/shasum -a 256 "$CONTEXT/migrations/026_provider_pilot_proof_integrity.sql" | /usr/bin/awk '{print $1}')
MIGRATION_027_SHA=$(/usr/bin/shasum -a 256 "$CONTEXT/migrations/027_provider_pilot_review_evidence.sql" | /usr/bin/awk '{print $1}')
MIGRATION_028_SHA=$(/usr/bin/shasum -a 256 "$CONTEXT/migrations/028_provider_commercial_proof_manifest.sql" | /usr/bin/awk '{print $1}')
MIGRATION_029_SHA=$(/usr/bin/shasum -a 256 "$CONTEXT/migrations/029_provider_settlement_receipts.sql" | /usr/bin/awk '{print $1}')
MIGRATION_030_SHA=$(/usr/bin/shasum -a 256 "$CONTEXT/migrations/030_provider_processor_net_receipts.sql" | /usr/bin/awk '{print $1}')
MIGRATION_031_SHA=$(/usr/bin/shasum -a 256 "$CONTEXT/migrations/031_action_interest_attempt_funnel.sql" | /usr/bin/awk '{print $1}')
MIGRATION_032_SHA=$(/usr/bin/shasum -a 256 "$CONTEXT/migrations/032_stage1_agent_surface_cohort.sql" | /usr/bin/awk '{print $1}')

GO_BINARY="${NHS_GO_BINARY:-/Users/shane/.local/bin/go}"
if [ ! -x "$GO_BINARY" ]; then
    echo "Go toolchain not found at $GO_BINARY" >&2
    exit 1
fi
# Test the archived module exactly. Inherited flags can silently narrow tests,
# inject build tags or a modfile, or select a parent workspace while leaving
# superficially green output.
export GOFLAGS=''
export GOWORK=off
if [ -n "$($GO_BINARY env GOFLAGS)" ] || [ "$($GO_BINARY env GOWORK)" != "off" ]; then
    echo "failed to isolate Go release verification from external flags or workspaces" >&2
    exit 1
fi
GOFMT_BINARY="$($GO_BINARY env GOROOT)/bin/gofmt"
if [ ! -x "$GOFMT_BINARY" ]; then
    echo "gofmt not found for the selected Go toolchain" >&2
    exit 1
fi

if [ "${#CHANGED_GO_PATHS[@]}" -gt 0 ]; then
    UNFORMATTED=$("$GOFMT_BINARY" -l "${CHANGED_GO_PATHS[@]}")
    if [ -n "$UNFORMATTED" ]; then
        echo "changed Go files are not gofmt-clean:" >&2
        echo "$UNFORMATTED" >&2
        exit 1
    fi
fi

(
    cd "$CONTEXT"
	/usr/bin/python3 tools/test-provider-company-verify.py
	/usr/bin/python3 tools/test-adopt-provider-exchange-candidate.py
	/usr/bin/python3 tools/test-build-exact-provider-image.py
	/usr/bin/python3 tools/test-provider-cutover-verify.py
    /usr/bin/python3 tools/test-provider-pilot-client.py
    /usr/bin/python3 tools/test-provider-pilot-operate.py
    /usr/bin/python3 tools/test-provider-pilot-status.py
    GO_TEST_LOG="$RELEASE_DIR/go-test.jsonl"
    if ! "$GO_BINARY" test -json -count=1 ./... >"$GO_TEST_LOG"; then
        echo "exact-archive Go tests failed; inspect $GO_TEST_LOG" >&2
        exit 1
    fi
    if ! /usr/bin/jq -e 'select(.Action == "pass" and .Test == "TestProviderExchangePostgresReleaseRegressions")' "$GO_TEST_LOG" >/dev/null; then
        echo "provider exchange PostgreSQL release regression test did not run and pass" >&2
        exit 1
    fi
    if ! /usr/bin/jq -e 'select(.Action == "pass" and .Test == "TestProtectedMigrationLedgerPostgres")' "$GO_TEST_LOG" >/dev/null; then
        echo "protected migration-ledger PostgreSQL test did not run and pass" >&2
        exit 1
    fi
    if /usr/bin/jq -e 'select(.Action == "skip" and (.Test == "TestProviderExchangePostgresReleaseRegressions" or .Test == "TestProtectedMigrationLedgerPostgres"))' "$GO_TEST_LOG" >/dev/null; then
        echo "a required PostgreSQL release regression test was skipped" >&2
        exit 1
    fi
    NHS_TEST_POSTGRES_DSN='' NHS_MIGRATION_TEST_POSTGRES_DSN='' \
        "$GO_BINARY" test -race ./internal/database ./internal/handlers ./internal/models ./internal/providerexchange ./cmd/server ./cmd/crawler ./cmd/provider-cutover-preflight
	"$GO_BINARY" vet ./...
	"$GO_BINARY" build ./...

	# Build the exact cutover preflight binary with the same immutable revision
	# used by the server image. Prove that a caller cannot relabel it with
	# --revision before any target database or credential is involved.
	CUTOVER_PREFLIGHT="$RELEASE_DIR/provider-cutover-preflight"
	CUTOVER_MISMATCH_RECEIPT="$RELEASE_DIR/provider-cutover-preflight-mismatch.json"
	CUTOVER_MATCH_RECEIPT="$RELEASE_DIR/provider-cutover-preflight-match.json"
	CGO_ENABLED=0 "$GO_BINARY" build -trimpath \
		-ldflags "-X main.releaseRevision=$COMMIT" \
		-o "$CUTOVER_PREFLIGHT" ./cmd/provider-cutover-preflight
	if /usr/bin/env -i APP_ROOT="$CONTEXT" "$CUTOVER_PREFLIGHT" \
		--revision 0000000000000000000000000000000000000000 \
		--mode disabled --migrations-dir "$CONTEXT/migrations" \
		>"$CUTOVER_MISMATCH_RECEIPT"; then
		echo "cutover preflight accepted a caller-selected wrong revision" >&2
		exit 1
	fi
	if ! /usr/bin/jq -e \
		'.contract == "nhs-provider-cutover-preflight-v2" and .ok == false and .error == "candidate_revision_mismatch"' \
		"$CUTOVER_MISMATCH_RECEIPT" >/dev/null; then
		echo "cutover preflight revision-mismatch receipt drifted" >&2
		exit 1
	fi
	if /usr/bin/env -i APP_ROOT="$CONTEXT" "$CUTOVER_PREFLIGHT" \
		--revision "$COMMIT" --mode disabled \
		--migrations-dir "$CONTEXT/migrations" >"$CUTOVER_MATCH_RECEIPT"; then
		echo "cutover preflight unexpectedly connected without a target database" >&2
		exit 1
	fi
	if ! /usr/bin/jq -e \
		'.contract == "nhs-provider-cutover-preflight-v2" and .ok == false and .error == "database_connection_failed"' \
		"$CUTOVER_MATCH_RECEIPT" >/dev/null; then
		echo "cutover preflight compiled-revision receipt drifted" >&2
		exit 1
	fi

	# Prove the exact archived binary can serve the only valid forward-recovery
	# mode after migration 022: free REST/MCP discovery and Stage 1 interest stay
	# online, paid sidecars disappear, and every provider mutation fails closed.
	RECOVERY_SERVER="$RELEASE_DIR/server-disabled"
	RECOVERY_LOG="$RELEASE_DIR/server-disabled.log"
	RECOVERY_PID=''
	cleanup_recovery() {
		if [ -n "${RECOVERY_PID:-}" ]; then
			kill "$RECOVERY_PID" 2>/dev/null || true
			wait "$RECOVERY_PID" 2>/dev/null || true
			RECOVERY_PID=''
		fi
	}
	trap cleanup_recovery EXIT
	trap 'exit 1' INT TERM
	RECOVERY_PORT=$(/usr/bin/python3 - <<'PY'
import socket
with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as listener:
    listener.bind(("127.0.0.1", 0))
    print(listener.getsockname()[1])
PY
)
	"$GO_BINARY" build -trimpath -ldflags "-X main.releaseRevision=$COMMIT" -o "$RECOVERY_SERVER" ./cmd/server
	/usr/bin/env -i \
	DATABASE_URL="$NHS_TEST_POSTGRES_DSN" \
	NHS_PROVIDER_EXCHANGE_MODE=disabled \
	NHS_PROVIDER_EXCHANGE_SIGNING_KEY_ID='' \
	NHS_PROVIDER_EXCHANGE_SIGNING_KEY='' \
	NHS_PROVIDER_EXCHANGE_PREVIOUS_SIGNING_KEYS_JSON='' \
	NHS_LISTEN_HOST=127.0.0.1 \
	BASE_URL="http://127.0.0.1:$RECOVERY_PORT" \
	APP_ROOT="$CONTEXT" PORT="$RECOVERY_PORT" \
		"$RECOVERY_SERVER" >"$RECOVERY_LOG" 2>&1 &
	RECOVERY_PID=$!
	RECOVERY_READY=false
	for _ in {1..75}; do
		if /usr/bin/curl --silent --show-error --noproxy '*' \
		   --connect-timeout 1 --max-time 2 \
		   "http://127.0.0.1:$RECOVERY_PORT/health" >/dev/null 2>&1; then
			RECOVERY_READY=true
			break
		fi
		/bin/sleep 0.2
	done
	if [ "$RECOVERY_READY" != "true" ]; then
		echo "exact disabled-recovery server did not become healthy; inspect $RECOVERY_LOG" >&2
		exit 1
	fi
	if ! /usr/bin/python3 tools/disabled-recovery-smoke.py \
		--base-url "http://127.0.0.1:$RECOVERY_PORT" \
		--expected-revision "$COMMIT"; then
		echo "exact disabled-recovery contract failed; inspect $RECOVERY_LOG" >&2
		exit 1
	fi
	cleanup_recovery
	trap - EXIT INT TERM

	"$GO_BINARY" run ./cmd/openapi-dump >"$RELEASE_DIR/openapi.yaml"
    /usr/bin/ruby -ryaml -e '
      doc = YAML.safe_load(File.read(ARGV.fetch(0)), permitted_classes: [], aliases: false)
      abort "OpenAPI document is not a mapping" unless doc.is_a?(Hash)
      abort "OpenAPI version missing" unless doc["openapi"]
      ticket = doc.dig("paths", "/action-tickets", "post", "responses")
      handoff = doc.dig("paths", "/action-tickets/handoff", "post", "responses")
      resolver = doc.dig("paths", "/provider/action-tickets/resolve", "post")
      abort "provider action paths missing" unless ticket.is_a?(Hash) && handoff.is_a?(Hash) && resolver.is_a?(Hash)
      required = %w[200 201 400 404 409 410 429 503]
      missing = required - ticket.keys.map(&:to_s)
      abort "action-ticket statuses missing: #{missing.join(",")}" unless missing.empty?
      abort "action-ticket path presents provider capacity as a caller payment challenge" if ticket.key?("402")

      security = resolver["security"]
      abort "controlled-intent resolver is not ProviderKey scoped" unless security == [{"ProviderKey" => []}]
      resolver_responses = resolver["responses"]
      required_resolver = %w[200 400 401 404 410 429 503]
      missing_resolver = required_resolver - resolver_responses.keys.map(&:to_s)
      abort "controlled-intent resolver statuses missing: #{missing_resolver.join(",")}" unless missing_resolver.empty?
      request_ref = resolver.dig("requestBody", "content", "application/json", "schema", "$ref")
      response_ref = resolver.dig("responses", "200", "content", "application/json", "schema", "$ref")
      abort "controlled-intent resolver request reference drifted" unless request_ref == "#/components/schemas/ProviderControlledIntentResolveRequest"
      abort "controlled-intent resolver response reference drifted" unless response_ref == "#/components/schemas/ProviderControlledIntentResolution"

      status_read = doc.dig("paths", "/provider/pilot-status", "get")
      demand_read = doc.dig("paths", "/provider/demand", "get")
      abort "provider status/demand reads missing" unless status_read.is_a?(Hash) && demand_read.is_a?(Hash)
      {"status" => [status_read, ["limit"], "ProviderPilotStatusResponse"],
       "demand" => [demand_read, ["days"], "ProviderDemandResponse"]}.each do |name, contract|
        operation, expected_parameters, schema_name = contract
        abort "provider #{name} read is not ProviderKey scoped" unless operation["security"] == [{"ProviderKey" => []}]
        parameters = operation.fetch("parameters").map { |parameter| parameter["name"] }
        abort "provider #{name} parameters drifted" unless parameters == expected_parameters
        reference = operation.dig("responses", "200", "content", "application/json", "schema", "$ref")
        abort "provider #{name} response reference drifted" unless reference == "#/components/schemas/#{schema_name}"
        required_statuses = %w[200 400 401 404 429 500 503]
        missing_statuses = required_statuses - operation.fetch("responses").keys.map(&:to_s)
        abort "provider #{name} statuses missing: #{missing_statuses.join(",")}" unless missing_statuses.empty?
      end
      abort "provider demand accepts a caller-selected domain" if demand_read.fetch("parameters").any? { |parameter| parameter["name"] == "domain" }

      schemas = doc.dig("components", "schemas")
      request_schema = schemas["ProviderControlledIntentResolveRequest"]
      abort "controlled-intent request is not strict" unless request_schema["additionalProperties"] == false && request_schema["required"] == ["attribution_token"] && request_schema.fetch("properties").keys == ["attribution_token"]
      response_schema = schemas["ProviderControlledIntentResolution"]
      response_allowed = %w[resolver_contract_version ticket_id offer_id offer_version action_type controlled_intent observed_at intent_available_until consent_version]
      abort "controlled-intent response fields drifted" unless response_schema["additionalProperties"] == false && response_schema.fetch("properties").keys.sort == response_allowed.sort && response_schema.fetch("required").sort == response_allowed.sort
      intent_schema = schemas["ProviderControlledIntent"]
      intent_allowed = %w[demand_topic region_code budget_band urgency requirement_flags]
      abort "controlled-intent bundle fields drifted" unless intent_schema["additionalProperties"] == false && intent_schema.fetch("properties").keys.sort == intent_allowed.sort
      forbidden = %w[query search_receipt_id provider_claim_id contact email name agent_id principal_id action_url bounty_cents currency charge_status]
      abort "controlled-intent schema exposes forbidden fields" unless (forbidden & (response_schema.fetch("properties").keys + intent_schema.fetch("properties").keys)).empty?

      provider_status_schema = schemas["ProviderPilotStatus"]
      provider_demand_schema = schemas["ProviderDemandAnalytics"]
      abort "provider status/demand schemas are not strict" unless provider_status_schema["additionalProperties"] == false && provider_demand_schema["additionalProperties"] == false
      provider_forbidden = %w[attribution_token token_hash action_url search_receipt_id query contact email agent_id principal_id company_hash provider_api_key_id]
      provider_fields = provider_status_schema.fetch("properties").keys + provider_demand_schema.fetch("properties").keys
      abort "provider status/demand exposes forbidden fields" unless (provider_forbidden & provider_fields).empty?

      handoff_schema = schemas["ActionTicketHandoffRequest"]
      disclosure_field = handoff_schema.dig("properties", "principal_controlled_intent_disclosure_consent")
      disclosure_version = handoff_schema.dig("properties", "controlled_intent_disclosure_consent_version")
      abort "optional disclosure consent fields missing" unless disclosure_field.is_a?(Hash) && disclosure_field["default"] == false && disclosure_version.is_a?(Hash)
      abort "optional disclosure consent was made mandatory" unless (handoff_schema["required"] & %w[principal_controlled_intent_disclosure_consent controlled_intent_disclosure_consent_version]).empty?

      mcp_source = File.read(File.join(ARGV.fetch(1), "internal/handlers/mcp.go"))
      abort "provider controlled-intent resolver must not be an MCP tool" if mcp_source.include?(%q{"resolve_provider_controlled_intent"})
    ' "$RELEASE_DIR/openapi.yaml" "$CONTEXT"

    NHS_RELEASE_CONTEXT="$CONTEXT" /usr/bin/python3 - <<'PY'
import hashlib
import json
import os
from pathlib import Path

root = Path(os.environ["NHS_RELEASE_CONTEXT"])
contract = json.loads((root / "design/reviews/2026-08-01-provider-exchange.json").read_text())
audit = json.loads((root / "design/reviews/2026-08-01-provider-exchange-audit.json").read_text())
if audit.get("pass") is not True or audit.get("issue_count") != 0:
    raise SystemExit("provider-exchange design audit is not passing")
if audit.get("review_scope_sha256") != contract.get("review_scope_sha256"):
    raise SystemExit("provider-exchange design audit is stale")
surfaces = contract.get("surfaces", [])
if len(surfaces) != 2:
    raise SystemExit("provider-exchange design contract must cover exactly two surfaces")
for surface in surfaces:
    source = root / surface["source"]
    digest = hashlib.sha256(source.read_bytes()).hexdigest()
    if digest != surface["reviewed_source_sha256"]:
        raise SystemExit(f"stale design approval for {surface['source']}")
    if surface.get("verdict") != "approved" or surface.get("blocking_issues"):
        raise SystemExit(f"unapproved design surface {surface['id']}")
    for render in surface["renders"].values():
        if not (root / render).is_file():
            raise SystemExit(f"missing required render {render}")
for evidence in contract.get("additional_render_evidence", []):
    artifact = root / evidence["file"]
    if hashlib.sha256(artifact.read_bytes()).hexdigest() != evidence["sha256"]:
        raise SystemExit(f"render evidence hash mismatch for {evidence['file']}")
PY

    /Users/shane/.local/bin/codex-secret scan "${CHANGED_PATHS[@]}"
)

{
	echo "contract=nhs-exact-release-verification-v2"
    echo "release_commit=$COMMIT"
    echo "release_base_commit=$BASE_COMMIT"
    echo "release_tree=$TREE"
    echo "changed_path_count=${#CHANGED_PATHS[@]}"
    echo "migration_019_sha256=$MIGRATION_019_SHA"
    echo "migration_020_sha256=$MIGRATION_020_SHA"
    echo "migration_021_sha256=$MIGRATION_021_SHA"
    echo "migration_022_sha256=$MIGRATION_022_SHA"
    echo "migration_023_sha256=$MIGRATION_023_SHA"
    echo "migration_024_sha256=$MIGRATION_024_SHA"
    echo "migration_025_sha256=$MIGRATION_025_SHA"
    echo "migration_026_sha256=$MIGRATION_026_SHA"
    echo "migration_027_sha256=$MIGRATION_027_SHA"
    echo "migration_028_sha256=$MIGRATION_028_SHA"
    echo "migration_029_sha256=$MIGRATION_029_SHA"
    echo "migration_030_sha256=$MIGRATION_030_SHA"
    echo "migration_031_sha256=$MIGRATION_031_SHA"
    echo "migration_032_sha256=$MIGRATION_032_SHA"
    echo "source_archive_sha256=$ARCHIVE_SHA"
    echo "build_arg=RELEASE_REVISION=$COMMIT"
	echo "exact_archive_tests_passed=true"
	echo "postgres_release_tests_passed=true"
	echo "disabled_recovery_smoke_passed=true"
	echo "preflight_binary_revision_bound=true"
	echo "secret_scan_findings=0"
	echo "oci_image_digest_verified=false"
	echo "target_cutover_preflight_verified=false"
	echo "restore_drill_verified=false"
	echo "deployment_ready=false"
    echo "deployment_command_emitted=false"
} > "$MANIFEST"

echo "Exact NHS release context verified. No deployment was performed."
echo "Manifest: $MANIFEST"
echo "Candidate commit: $COMMIT"
echo "No deploy command was emitted. Migrations 019-031 require an owner-authorized"
echo "single-machine cutover with old writers quiesced, a target-database"
echo "preflight, staged signer references, and a verified forward rollback or"
echo "database recovery plan. Use the pilot runbook for that separate gate."
