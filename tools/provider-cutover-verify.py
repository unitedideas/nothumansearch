#!/usr/bin/env python3
"""Fail-closed local verification for NHS provider-exchange cutover evidence.

This tool has two deliberately separate surfaces:

* ``restore-receipt`` validates an offline, privacy-bounded restore-drill JSON
  receipt against exact operator-supplied bindings. It performs no network or
  database operation.
* ``database`` performs one explicit read-only PostgreSQL query when
  ``DATABASE_URL`` has been injected into this process. The credential is never
  accepted on argv, forwarded on argv, or included in output.

Neither surface deploys, changes Fly state, runs a migration, or writes to the
database. A successful receipt is evidence for one bounded gate only; it is not
a deployment authorization.
"""

from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
import pathlib
import re
import resource
import shutil
import stat
import subprocess
import sys
import urllib.parse
from collections.abc import Callable, Mapping, Sequence
from typing import Any


RESTORE_CONTRACT = "nhs-provider-restore-drill-v1"
RESTORE_VERIFICATION_CONTRACT = "nhs-provider-restore-drill-verification-v1"
DATABASE_VERIFICATION_CONTRACT = "nhs-provider-migration-verification-v1"
DATABASE_URL_ENV = "DATABASE_URL"
PSQL_BINARY_ENV = "NHS_PSQL_BINARY"
MAX_RECEIPT_BYTES = 64 * 1024
MAX_MIGRATION_BYTES = 1024 * 1024
MAX_DATABASE_OUTPUT_BYTES = 16 * 1024
DATABASE_TIMEOUT_SECONDS = 20
DEFAULT_MIGRATIONS_DIR = pathlib.Path(__file__).absolute().parent.parent / "migrations"

PROTECTED_MIGRATIONS = (
    "019_provider_exchange.sql",
    "020_action_interest_receipts.sql",
    "021_provider_capacity_reservations.sql",
    "022_provider_commercial_proof.sql",
    "023_provider_controlled_intent_disclosure.sql",
    "024_provider_pilot_boundary.sql",
    "025_stage1_fact_integrity.sql",
    "026_provider_pilot_proof_integrity.sql",
    "027_provider_pilot_review_evidence.sql",
    "028_provider_commercial_proof_manifest.sql",
    "029_provider_settlement_receipts.sql",
    "030_provider_processor_net_receipts.sql",
    "031_action_interest_attempt_funnel.sql",
)

_REVISION_PATTERN = re.compile(r"^[0-9a-f]{40}$")
_SHA256_PATTERN = re.compile(r"^[0-9a-f]{64}$")
_APP_PATTERN = re.compile(r"^[a-z0-9][a-z0-9-]{0,62}$")
_VOLUME_PATTERN = re.compile(r"^vol_[A-Za-z0-9]{8,64}$")
_SNAPSHOT_PATTERN = re.compile(r"^vs_[A-Za-z0-9]{8,64}$")
_MACHINE_PATTERN = re.compile(r"^[0-9a-f]{14}$")
_MIGRATION_NAME_PATTERN = re.compile(r"^[0-9]{3}_[a-z0-9_]+\.sql$")
_RFC3339_UTC_PATTERN = re.compile(
    r"^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}"
    r"(?:\.[0-9]{1,6})?Z$"
)

_RESTORE_TOP_LEVEL_KEYS = frozenset(
    {
        "contract",
        "candidate_revision",
        "source",
        "snapshot",
        "restore",
        "migration_inventory",
        "drill_started_at",
        "drill_completed_at",
        "cleanup",
    }
)
_SOURCE_KEYS = frozenset(
    {"app", "volume_id", "machine_id", "database_identity_sha256"}
)
_SNAPSHOT_KEYS = frozenset({"id", "sha256", "created_at", "size_bytes"})
_RESTORE_KEYS = frozenset(
    {
        "app",
        "volume_id",
        "machine_id",
        "postgres_major",
        "database_identity_sha256",
        "verified_at",
    }
)
_INVENTORY_KEYS = frozenset({"ledger_present", "receipts"})
_MIGRATION_KEYS = frozenset({"name", "sha256", "applied_by_commit"})
_CLEANUP_KEYS = frozenset(
    {
        "outcome",
        "restored_machine_destroyed",
        "restored_volume_destroyed",
        "verified_at",
    }
)

_LIBPQ_QUERY_ENV = {
    "sslmode": "PGSSLMODE",
    "sslrootcert": "PGSSLROOTCERT",
    "sslcert": "PGSSLCERT",
    "sslkey": "PGSSLKEY",
    "sslcrl": "PGSSLCRL",
    "sslpassword": "PGSSLPASSWORD",
    "channel_binding": "PGCHANNELBINDING",
    "gssencmode": "PGGSSENCMODE",
    "target_session_attrs": "PGTARGETSESSIONATTRS",
}

_MIGRATION_QUERY = """
SELECT COALESCE(
  json_agg(
    json_build_object(
      'name', protected.name,
      'sha256', protected.sha256,
      'applied_by_commit', protected.applied_by_commit
    ) ORDER BY protected.name
  ),
  '[]'::json
)::text
FROM (
  SELECT name, sha256, applied_by_commit
  FROM public.nhs_schema_migrations
  WHERE name >= '019_'
  ORDER BY name
  LIMIT 32
) AS protected
""".strip()


class VerificationError(Exception):
    """Stable, non-sensitive verifier failure."""

    def __init__(self, code: str):
        super().__init__(code)
        self.code = code


class SafeArgumentParser(argparse.ArgumentParser):
    """Keep parser failures bounded and independent of supplied values."""

    def error(self, _message: str) -> None:
        raise VerificationError("invalid_arguments")


def build_parser() -> argparse.ArgumentParser:
    parser = SafeArgumentParser(
        description=(
            "Validate offline NHS restore-drill evidence or explicitly perform "
            "a read-only protected-migration check. No deployment is performed."
        )
    )
    subparsers = parser.add_subparsers(dest="command", required=True)

    restore = subparsers.add_parser(
        "restore-receipt",
        help="Validate one offline restore-drill receipt against exact bindings.",
    )
    restore.add_argument("--receipt", required=True)
    restore.add_argument("--revision", required=True)
    restore.add_argument("--expected-source-app", required=True)
    restore.add_argument("--expected-source-volume-id", required=True)
    restore.add_argument("--expected-source-database-identity-sha256", required=True)
    restore.add_argument("--expected-snapshot-id", required=True)
    restore.add_argument("--expected-snapshot-sha256", required=True)
    restore.add_argument("--expected-snapshot-created-at", required=True)
    restore.add_argument("--expected-postgres-major", required=True, type=int)

    database = subparsers.add_parser(
        "database",
        help=(
            "Read DATABASE_URL only from the injected environment and verify "
            "protected migration receipts without writing."
        ),
    )
    database.add_argument("--revision", required=True)
    database.add_argument(
        "--migrations-dir",
        default=str(DEFAULT_MIGRATIONS_DIR),
        help=(
            "Directory containing the exact migration files whose bytes must "
            "match the protected database receipts."
        ),
    )
    database.add_argument(
        "--confirm-read-only-database-check",
        action="store_true",
        help="Confirm this exact read-only database inspection is intended.",
    )
    return parser


def _exact_keys(value: Any, expected: frozenset[str], code: str) -> dict[str, Any]:
    if not isinstance(value, dict) or set(value) != expected:
        raise VerificationError(code)
    return value


def _validated_text(
    value: Any,
    pattern: re.Pattern[str],
    code: str,
    *,
    maximum: int = 256,
) -> str:
    if not isinstance(value, str) or len(value) > maximum or not pattern.fullmatch(value):
        raise VerificationError(code)
    return value


def _revision(value: Any) -> str:
    return _validated_text(value, _REVISION_PATTERN, "invalid_revision", maximum=40)


def _sha256(value: Any, code: str) -> str:
    return _validated_text(value, _SHA256_PATTERN, code, maximum=64)


def _timestamp(value: Any, code: str) -> tuple[str, dt.datetime]:
    if not isinstance(value, str) or not _RFC3339_UTC_PATTERN.fullmatch(value):
        raise VerificationError(code)
    try:
        parsed = dt.datetime.fromisoformat(value[:-1] + "+00:00")
    except ValueError as error:
        raise VerificationError(code) from error
    if parsed.tzinfo != dt.timezone.utc:
        raise VerificationError(code)
    return value, parsed


def _read_receipt(path_value: str) -> dict[str, Any]:
    receipt_path = pathlib.Path(path_value)
    flags = os.O_RDONLY | getattr(os, "O_CLOEXEC", 0) | getattr(os, "O_NOFOLLOW", 0)
    try:
        descriptor = os.open(receipt_path, flags)
    except OSError as error:
        raise VerificationError("restore_receipt_unavailable") from error
    try:
        metadata = os.fstat(descriptor)
        if not stat.S_ISREG(metadata.st_mode) or not 1 <= metadata.st_size <= MAX_RECEIPT_BYTES:
            raise VerificationError("invalid_restore_receipt_file")
        with os.fdopen(descriptor, "rb", closefd=False) as receipt_file:
            payload = receipt_file.read(MAX_RECEIPT_BYTES + 1)
        if len(payload) != metadata.st_size or len(payload) > MAX_RECEIPT_BYTES:
            raise VerificationError("invalid_restore_receipt_file")
    finally:
        os.close(descriptor)
    try:
        document = json.loads(payload.decode("utf-8", "strict"))
    except (UnicodeDecodeError, json.JSONDecodeError) as error:
        raise VerificationError("invalid_restore_receipt_json") from error
    if not isinstance(document, dict):
        raise VerificationError("invalid_restore_receipt_schema")
    return document


def _migration_inventory(value: Any) -> tuple[list[dict[str, str]], str]:
    inventory = _exact_keys(value, _INVENTORY_KEYS, "invalid_migration_inventory")
    ledger_present = inventory["ledger_present"]
    receipts = inventory["receipts"]
    if not isinstance(ledger_present, bool) or not isinstance(receipts, list) or len(receipts) > 128:
        raise VerificationError("invalid_migration_inventory")
    if (not ledger_present and receipts) or (ledger_present and not receipts):
        raise VerificationError("invalid_migration_inventory")

    normalized: list[dict[str, str]] = []
    for raw in receipts:
        item = _exact_keys(raw, _MIGRATION_KEYS, "invalid_migration_inventory")
        name = _validated_text(
            item["name"], _MIGRATION_NAME_PATTERN, "invalid_migration_inventory"
        )
        normalized.append(
            {
                "name": name,
                "sha256": _sha256(item["sha256"], "invalid_migration_inventory"),
                "applied_by_commit": _revision(item["applied_by_commit"]),
            }
        )
    names = [item["name"] for item in normalized]
    if names != sorted(names) or len(names) != len(set(names)):
        raise VerificationError("invalid_migration_inventory")
    canonical = json.dumps(normalized, sort_keys=True, separators=(",", ":")).encode("utf-8")
    return normalized, hashlib.sha256(canonical).hexdigest()


def verify_restore_receipt(
    receipt_path: str,
    *,
    revision: str,
    expected_source_app: str,
    expected_source_volume_id: str,
    expected_source_database_identity_sha256: str,
    expected_snapshot_id: str,
    expected_snapshot_sha256: str,
    expected_snapshot_created_at: str,
    expected_postgres_major: int,
) -> dict[str, Any]:
    expected_revision = _revision(revision)
    expected_source_app = _validated_text(
        expected_source_app, _APP_PATTERN, "invalid_expected_source_app", maximum=63
    )
    expected_source_volume_id = _validated_text(
        expected_source_volume_id,
        _VOLUME_PATTERN,
        "invalid_expected_source_volume_id",
    )
    expected_source_database_identity_sha256 = _sha256(
        expected_source_database_identity_sha256,
        "invalid_expected_source_database_identity_sha256",
    )
    expected_snapshot_id = _validated_text(
        expected_snapshot_id, _SNAPSHOT_PATTERN, "invalid_expected_snapshot_id"
    )
    expected_snapshot_sha256 = _sha256(
        expected_snapshot_sha256, "invalid_expected_snapshot_sha256"
    )
    expected_snapshot_created_at, _ = _timestamp(
        expected_snapshot_created_at, "invalid_expected_snapshot_created_at"
    )
    if not isinstance(expected_postgres_major, int) or isinstance(expected_postgres_major, bool):
        raise VerificationError("invalid_expected_postgres_major")
    if not 12 <= expected_postgres_major <= 30:
        raise VerificationError("invalid_expected_postgres_major")

    document = _exact_keys(
        _read_receipt(receipt_path), _RESTORE_TOP_LEVEL_KEYS, "invalid_restore_receipt_schema"
    )
    if document["contract"] != RESTORE_CONTRACT:
        raise VerificationError("restore_contract_mismatch")
    if _revision(document["candidate_revision"]) != expected_revision:
        raise VerificationError("restore_revision_mismatch")

    source = _exact_keys(document["source"], _SOURCE_KEYS, "invalid_restore_source")
    source_app = _validated_text(source["app"], _APP_PATTERN, "invalid_restore_source", maximum=63)
    source_volume = _validated_text(
        source["volume_id"], _VOLUME_PATTERN, "invalid_restore_source"
    )
    source_machine = _validated_text(
        source["machine_id"], _MACHINE_PATTERN, "invalid_restore_source", maximum=14
    )
    source_database_identity = _sha256(
        source["database_identity_sha256"], "invalid_restore_source"
    )
    if (
        source_app != expected_source_app
        or source_volume != expected_source_volume_id
        or source_database_identity != expected_source_database_identity_sha256
    ):
        raise VerificationError("restore_source_binding_mismatch")

    snapshot = _exact_keys(document["snapshot"], _SNAPSHOT_KEYS, "invalid_restore_snapshot")
    snapshot_id = _validated_text(
        snapshot["id"], _SNAPSHOT_PATTERN, "invalid_restore_snapshot"
    )
    snapshot_sha = _sha256(snapshot["sha256"], "invalid_restore_snapshot")
    snapshot_created, snapshot_created_time = _timestamp(
        snapshot["created_at"], "invalid_restore_snapshot"
    )
    snapshot_size = snapshot["size_bytes"]
    if not isinstance(snapshot_size, int) or isinstance(snapshot_size, bool) or snapshot_size <= 0:
        raise VerificationError("invalid_restore_snapshot")
    if (
        snapshot_id != expected_snapshot_id
        or snapshot_sha != expected_snapshot_sha256
        or snapshot_created != expected_snapshot_created_at
    ):
        raise VerificationError("restore_snapshot_binding_mismatch")

    restore = _exact_keys(document["restore"], _RESTORE_KEYS, "invalid_restore_target")
    restored_app = _validated_text(
        restore["app"], _APP_PATTERN, "invalid_restore_target", maximum=63
    )
    restored_volume = _validated_text(
        restore["volume_id"], _VOLUME_PATTERN, "invalid_restore_target"
    )
    restored_machine = _validated_text(
        restore["machine_id"], _MACHINE_PATTERN, "invalid_restore_target", maximum=14
    )
    postgres_major = restore["postgres_major"]
    restored_database_identity = _sha256(
        restore["database_identity_sha256"], "invalid_restore_target"
    )
    restored_verified, restored_verified_time = _timestamp(
        restore["verified_at"], "invalid_restore_target"
    )
    if (
        not isinstance(postgres_major, int)
        or isinstance(postgres_major, bool)
        or postgres_major != expected_postgres_major
        or restored_app == source_app
        or restored_volume == source_volume
        or restored_machine == source_machine
        or restored_database_identity != source_database_identity
    ):
        raise VerificationError("restore_target_binding_mismatch")

    migrations, migration_inventory_sha256 = _migration_inventory(
        document["migration_inventory"]
    )
    drill_started, drill_started_time = _timestamp(
        document["drill_started_at"], "invalid_restore_drill_time"
    )
    drill_completed, drill_completed_time = _timestamp(
        document["drill_completed_at"], "invalid_restore_drill_time"
    )
    cleanup = _exact_keys(document["cleanup"], _CLEANUP_KEYS, "invalid_restore_cleanup")
    cleanup_verified, cleanup_verified_time = _timestamp(
        cleanup["verified_at"], "invalid_restore_cleanup"
    )
    if (
        cleanup["outcome"] != "complete"
        or cleanup["restored_machine_destroyed"] is not True
        or cleanup["restored_volume_destroyed"] is not True
    ):
        raise VerificationError("restore_cleanup_incomplete")
    if not (
        snapshot_created_time <= restored_verified_time
        and drill_started_time <= restored_verified_time
        and restored_verified_time <= cleanup_verified_time
        and cleanup_verified_time <= drill_completed_time
    ):
        raise VerificationError("restore_timeline_invalid")

    return {
        "ok": True,
        "contract": RESTORE_VERIFICATION_CONTRACT,
        "restore_contract": RESTORE_CONTRACT,
        "candidate_revision": expected_revision,
        "source_app": source_app,
        "source_volume_id": source_volume,
        "source_database_identity_sha256": source_database_identity,
        "snapshot_id": snapshot_id,
        "snapshot_sha256": snapshot_sha,
        "snapshot_created_at": snapshot_created,
        "snapshot_size_bytes": snapshot_size,
        "restored_app": restored_app,
        "restored_volume_id": restored_volume,
        "restored_machine_id": restored_machine,
        "postgres_major": postgres_major,
        "restored_database_identity_sha256": restored_database_identity,
        "restored_verified_at": restored_verified,
        "migration_ledger_present": document["migration_inventory"]["ledger_present"],
        "migration_receipt_count": len(migrations),
        "migration_inventory_sha256": migration_inventory_sha256,
        "drill_started_at": drill_started,
        "drill_completed_at": drill_completed,
        "cleanup_outcome": "complete",
        "cleanup_verified_at": cleanup_verified,
        "deployment_ready": False,
    }


def _database_subprocess_environment(database_url: str, revision: str) -> dict[str, str]:
    if (
        not isinstance(database_url, str)
        or not database_url
        or len(database_url) > 8192
        or any(ord(character) < 0x20 or ord(character) == 0x7F for character in database_url)
    ):
        raise VerificationError("database_url_unavailable")
    try:
        parsed = urllib.parse.urlsplit(database_url)
    except ValueError as error:
        raise VerificationError("database_url_invalid") from error
    if parsed.scheme not in {"postgres", "postgresql"} or parsed.fragment:
        raise VerificationError("database_url_invalid")
    try:
        hostname = parsed.hostname
        port = parsed.port or 5432
        username = urllib.parse.unquote(parsed.username or "")
        database_authentication_value = urllib.parse.unquote(parsed.password or "")
        database_name = urllib.parse.unquote(parsed.path[1:]) if parsed.path.startswith("/") else ""
    except (ValueError, UnicodeDecodeError) as error:
        raise VerificationError("database_url_invalid") from error
    if (
        not hostname
        or not username
        or not database_name
        or "/" in database_name
        or not 1 <= port <= 65535
        or any(not value or len(value) > 1024 for value in (hostname, username, database_name))
        or any(
            ord(character) < 0x20 or ord(character) == 0x7F
            for value in (hostname, username, database_name, database_authentication_value)
            for character in value
        )
    ):
        raise VerificationError("database_url_invalid")

    child_environment = {
        "LC_ALL": "C",
        "PGHOST": hostname,
        "PGPORT": str(port),
        "PGUSER": username,
        "PGDATABASE": database_name,
        "PGAPPNAME": "nhs-cutover-verify:" + revision,
        "PGCONNECT_TIMEOUT": "10",
        "PGOPTIONS": (
            "-c default_transaction_read_only=on "
            "-c statement_timeout=10000 -c lock_timeout=2000"
        ),
    }
    if database_authentication_value:
        child_environment["PGPASSWORD"] = database_authentication_value
    try:
        query_values = urllib.parse.parse_qsl(
            parsed.query, keep_blank_values=True, strict_parsing=True
        )
    except ValueError as error:
        raise VerificationError("database_url_invalid") from error
    seen_query_keys: set[str] = set()
    for key, value in query_values:
        environment_key = _LIBPQ_QUERY_ENV.get(key)
        if (
            environment_key is None
            or key in seen_query_keys
            or not value
            or len(value) > 2048
            or any(ord(character) < 0x20 or ord(character) == 0x7F for character in value)
        ):
            raise VerificationError("database_url_invalid")
        seen_query_keys.add(key)
        child_environment[environment_key] = value
    return child_environment


def _psql_binary(environ: Mapping[str, str]) -> str:
    configured = environ.get(PSQL_BINARY_ENV, "psql")
    if not isinstance(configured, str) or not configured or len(configured) > 4096:
        raise VerificationError("psql_unavailable")
    if os.path.sep in configured:
        candidate = configured
    else:
        candidate = shutil.which(configured) or ""
    if not candidate or not os.path.isfile(candidate) or not os.access(candidate, os.X_OK):
        raise VerificationError("psql_unavailable")
    return os.path.abspath(candidate)


def _migration_file_hashes(migrations_dir: str | os.PathLike[str]) -> dict[str, str]:
    try:
        directory_path = os.fspath(migrations_dir)
    except TypeError as error:
        raise VerificationError("migrations_directory_invalid") from error
    if not isinstance(directory_path, str) or not directory_path or len(directory_path) > 4096:
        raise VerificationError("migrations_directory_invalid")

    directory_flags = (
        os.O_RDONLY
        | getattr(os, "O_CLOEXEC", 0)
        | getattr(os, "O_DIRECTORY", 0)
        | getattr(os, "O_NOFOLLOW", 0)
    )
    try:
        directory_descriptor = os.open(directory_path, directory_flags)
    except OSError as error:
        raise VerificationError("migrations_directory_invalid") from error
    try:
        if not stat.S_ISDIR(os.fstat(directory_descriptor).st_mode):
            raise VerificationError("migrations_directory_invalid")
        hashes: dict[str, str] = {}
        file_flags = (
            os.O_RDONLY
            | getattr(os, "O_CLOEXEC", 0)
            | getattr(os, "O_NOFOLLOW", 0)
            | getattr(os, "O_NONBLOCK", 0)
        )
        for name in PROTECTED_MIGRATIONS:
            try:
                file_descriptor = os.open(name, file_flags, dir_fd=directory_descriptor)
            except OSError as error:
                raise VerificationError("migration_file_unavailable") from error
            try:
                metadata = os.fstat(file_descriptor)
                if (
                    not stat.S_ISREG(metadata.st_mode)
                    or not 1 <= metadata.st_size <= MAX_MIGRATION_BYTES
                ):
                    raise VerificationError("migration_file_invalid")
                remaining = metadata.st_size
                digest = hashlib.sha256()
                while remaining:
                    chunk = os.read(file_descriptor, min(remaining, 64 * 1024))
                    if not chunk:
                        raise VerificationError("migration_file_invalid")
                    digest.update(chunk)
                    remaining -= len(chunk)
                if os.read(file_descriptor, 1):
                    raise VerificationError("migration_file_invalid")
                hashes[name] = digest.hexdigest()
            except OSError as error:
                raise VerificationError("migration_file_invalid") from error
            finally:
                os.close(file_descriptor)
        return hashes
    finally:
        os.close(directory_descriptor)


def _run_psql(
    command: Sequence[str],
    *,
    env: Mapping[str, str],
    timeout: int,
) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        list(command),
        check=False,
        text=True,
        stdin=subprocess.DEVNULL,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        env=dict(env),
        timeout=timeout,
    )


def verify_database_migrations(
    revision: str,
    *,
    environ: Mapping[str, str],
    migrations_dir: str | os.PathLike[str] = DEFAULT_MIGRATIONS_DIR,
    runner: Callable[..., subprocess.CompletedProcess[str]] = _run_psql,
) -> dict[str, Any]:
    expected_revision = _revision(revision)
    expected_migration_hashes = _migration_file_hashes(migrations_dir)
    database_url = environ.get(DATABASE_URL_ENV, "")
    child_environment = _database_subprocess_environment(database_url, expected_revision)
    psql = _psql_binary(environ)
    command = (
        psql,
        "-X",
        "--quiet",
        "--no-align",
        "--tuples-only",
        "--set=ON_ERROR_STOP=1",
        "--command",
        _MIGRATION_QUERY,
    )
    try:
        completed = runner(
            command,
            env=child_environment,
            timeout=DATABASE_TIMEOUT_SECONDS,
        )
    except (OSError, subprocess.SubprocessError) as error:
        raise VerificationError("database_check_failed") from error
    if completed.returncode != 0:
        raise VerificationError("database_check_failed")
    encoded_output = completed.stdout.encode("utf-8", "strict")
    if not 1 <= len(encoded_output) <= MAX_DATABASE_OUTPUT_BYTES:
        raise VerificationError("database_output_invalid")
    try:
        rows = json.loads(completed.stdout)
    except json.JSONDecodeError as error:
        raise VerificationError("database_output_invalid") from error
    if not isinstance(rows, list) or len(rows) > 32:
        raise VerificationError("database_output_invalid")

    normalized: list[dict[str, str]] = []
    for row in rows:
        if not isinstance(row, dict) or set(row) != _MIGRATION_KEYS:
            raise VerificationError("database_output_invalid")
        name = _validated_text(
            row["name"], _MIGRATION_NAME_PATTERN, "database_output_invalid"
        )
        sha = _sha256(row["sha256"], "database_output_invalid")
        applied_by_commit = _revision(row["applied_by_commit"])
        normalized.append(
            {"name": name, "sha256": sha, "applied_by_commit": applied_by_commit}
        )
    if tuple(item["name"] for item in normalized) != PROTECTED_MIGRATIONS:
        raise VerificationError("protected_migration_set_mismatch")
    if any(item["applied_by_commit"] != expected_revision for item in normalized):
        raise VerificationError("protected_migration_revision_mismatch")
    if any(
        item["sha256"] != expected_migration_hashes[item["name"]]
        for item in normalized
    ):
        raise VerificationError("protected_migration_sha256_mismatch")
    canonical = json.dumps(normalized, sort_keys=True, separators=(",", ":")).encode("utf-8")
    file_inventory = [
        {"name": name, "sha256": expected_migration_hashes[name]}
        for name in PROTECTED_MIGRATIONS
    ]
    canonical_files = json.dumps(
        file_inventory, sort_keys=True, separators=(",", ":")
    ).encode("utf-8")
    return {
        "ok": True,
        "contract": DATABASE_VERIFICATION_CONTRACT,
        "candidate_revision": expected_revision,
        "protected_migration_count": len(normalized),
        "protected_migrations": [item["name"] for item in normalized],
        "migration_inventory_sha256": hashlib.sha256(canonical).hexdigest(),
        "migration_files_sha256": hashlib.sha256(canonical_files).hexdigest(),
        "migrations_dir_verified": True,
        "read_only": True,
        "deployment_ready": False,
    }


def run(argv: Sequence[str], *, environ: Mapping[str, str]) -> dict[str, Any]:
    args = build_parser().parse_args(list(argv))
    if args.command == "restore-receipt":
        return verify_restore_receipt(
            args.receipt,
            revision=args.revision,
            expected_source_app=args.expected_source_app,
            expected_source_volume_id=args.expected_source_volume_id,
            expected_source_database_identity_sha256=(
                args.expected_source_database_identity_sha256
            ),
            expected_snapshot_id=args.expected_snapshot_id,
            expected_snapshot_sha256=args.expected_snapshot_sha256,
            expected_snapshot_created_at=args.expected_snapshot_created_at,
            expected_postgres_major=args.expected_postgres_major,
        )
    if args.command == "database":
        if not args.confirm_read_only_database_check:
            raise VerificationError("database_check_confirmation_required")
        return verify_database_migrations(
            args.revision,
            environ=environ,
            migrations_dir=args.migrations_dir,
        )
    raise VerificationError("invalid_arguments")


def _emit(stream, document: Mapping[str, Any]) -> None:
    json.dump(document, stream, sort_keys=True, separators=(",", ":"))
    stream.write("\n")


def main(argv: Sequence[str] | None = None) -> int:
    try:
        resource.setrlimit(resource.RLIMIT_CORE, (0, 0))
    except (OSError, ValueError):
        _emit(sys.stderr, {"error": "core_dump_hardening_unavailable", "ok": False})
        return 1
    try:
        result = run(sys.argv[1:] if argv is None else argv, environ=os.environ)
    except VerificationError as error:
        _emit(sys.stderr, {"error": error.code, "ok": False})
        return 1
    _emit(sys.stdout, result)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
