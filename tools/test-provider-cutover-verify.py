#!/usr/bin/env python3
"""Offline and fake-client tests for provider-cutover-verify.py."""

from __future__ import annotations

import contextlib
import hashlib
import importlib.util
import io
import json
import os
import pathlib
import subprocess
import sys
import tempfile
import unittest
from unittest import mock


MODULE_PATH = pathlib.Path(__file__).with_name("provider-cutover-verify.py")
SPEC = importlib.util.spec_from_file_location("provider_cutover_verify", MODULE_PATH)
cutover = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = cutover
SPEC.loader.exec_module(cutover)

REVISION = "c4378e86fe2dbd4cb0483324ca0ea7358d7c1392"
OTHER_REVISION = "267435481c6885bf3d0e035092dfa2ed423ca509"
SOURCE_APP = "nothumansearch-db"
SOURCE_VOLUME = "vol_vlyp5p7y1n7od8p4"
SOURCE_MACHINE = "1781e030a59389"
SOURCE_DATABASE_IDENTITY = "a" * 64
SNAPSHOT_ID = "vs_8AL9b2qGe3RlH87kkNJV"
SNAPSHOT_SHA256 = "b" * 64
SNAPSHOT_CREATED_AT = "2026-08-02T01:16:00Z"
RESTORED_VOLUME = "vol_restore12345678"
RESTORED_MACHINE = "abcdef12345678"
RESTORED_DATABASE_IDENTITY = SOURCE_DATABASE_IDENTITY


def write_migration_files(directory: pathlib.Path) -> dict[str, str]:
    directory.mkdir()
    hashes: dict[str, str] = {}
    for index, name in enumerate(cutover.PROTECTED_MIGRATIONS, start=1):
        payload = f"-- fixture {index}: {name}\n".encode("utf-8")
        (directory / name).write_bytes(payload)
        hashes[name] = hashlib.sha256(payload).hexdigest()
    return hashes


def migration_rows(
    revision: str = REVISION,
    hashes: dict[str, str] | None = None,
) -> list[dict[str, str]]:
    return [
        {
            "name": name,
            "sha256": hashes[name] if hashes is not None else f"{index:064x}",
            "applied_by_commit": revision,
        }
        for index, name in enumerate(cutover.PROTECTED_MIGRATIONS, start=1)
    ]


def valid_restore_receipt() -> dict[str, object]:
    return {
        "contract": cutover.RESTORE_CONTRACT,
        "candidate_revision": REVISION,
        "source": {
            "app": SOURCE_APP,
            "volume_id": SOURCE_VOLUME,
            "machine_id": SOURCE_MACHINE,
            "database_identity_sha256": SOURCE_DATABASE_IDENTITY,
        },
        "snapshot": {
            "id": SNAPSHOT_ID,
            "sha256": SNAPSHOT_SHA256,
            "created_at": SNAPSHOT_CREATED_AT,
            "size_bytes": 290520105,
        },
        "restore": {
            "app": "nothumansearch-db-restore",
            "volume_id": RESTORED_VOLUME,
            "machine_id": RESTORED_MACHINE,
            "postgres_major": 17,
            "database_identity_sha256": RESTORED_DATABASE_IDENTITY,
            "verified_at": "2026-08-02T02:05:00Z",
        },
        "migration_inventory": {
            "ledger_present": True,
            "receipts": migration_rows(),
        },
        "drill_started_at": "2026-08-02T02:00:00Z",
        "drill_completed_at": "2026-08-02T02:10:00Z",
        "cleanup": {
            "outcome": "complete",
            "restored_machine_destroyed": True,
            "restored_volume_destroyed": True,
            "verified_at": "2026-08-02T02:09:00Z",
        },
    }


def restore_arguments(receipt_path: pathlib.Path) -> list[str]:
    return [
        "restore-receipt",
        "--receipt",
        str(receipt_path),
        "--revision",
        REVISION,
        "--expected-source-app",
        SOURCE_APP,
        "--expected-source-volume-id",
        SOURCE_VOLUME,
        "--expected-source-database-identity-sha256",
        SOURCE_DATABASE_IDENTITY,
        "--expected-snapshot-id",
        SNAPSHOT_ID,
        "--expected-snapshot-sha256",
        SNAPSHOT_SHA256,
        "--expected-snapshot-created-at",
        SNAPSHOT_CREATED_AT,
        "--expected-postgres-major",
        "17",
    ]


class RestoreReceiptTest(unittest.TestCase):
    def write_receipt(self, directory: pathlib.Path, document: object) -> pathlib.Path:
        receipt_path = directory / "restore-receipt.json"
        receipt_path.write_text(json.dumps(document), encoding="utf-8")
        return receipt_path

    def verify(self, receipt_path: pathlib.Path) -> dict[str, object]:
        return cutover.verify_restore_receipt(
            str(receipt_path),
            revision=REVISION,
            expected_source_app=SOURCE_APP,
            expected_source_volume_id=SOURCE_VOLUME,
            expected_source_database_identity_sha256=SOURCE_DATABASE_IDENTITY,
            expected_snapshot_id=SNAPSHOT_ID,
            expected_snapshot_sha256=SNAPSHOT_SHA256,
            expected_snapshot_created_at=SNAPSHOT_CREATED_AT,
            expected_postgres_major=17,
        )

    def test_valid_receipt_is_strictly_bound_and_bounded(self):
        with tempfile.TemporaryDirectory(prefix="nhs-restore-receipt-") as raw:
            receipt_path = self.write_receipt(pathlib.Path(raw), valid_restore_receipt())
            result = self.verify(receipt_path)

        self.assertTrue(result["ok"])
        self.assertEqual(result["candidate_revision"], REVISION)
        self.assertEqual(result["source_volume_id"], SOURCE_VOLUME)
        self.assertEqual(result["snapshot_id"], SNAPSHOT_ID)
        self.assertEqual(result["restored_volume_id"], RESTORED_VOLUME)
        self.assertEqual(result["restored_machine_id"], RESTORED_MACHINE)
        self.assertEqual(result["postgres_major"], 17)
        self.assertEqual(result["migration_receipt_count"], 12)
        self.assertEqual(result["cleanup_outcome"], "complete")
        self.assertFalse(result["deployment_ready"])
        self.assertLess(len(json.dumps(result)), 4096)

    def test_empty_preprotected_inventory_is_explicitly_valid(self):
        receipt = valid_restore_receipt()
        receipt["migration_inventory"] = {"ledger_present": False, "receipts": []}
        with tempfile.TemporaryDirectory(prefix="nhs-restore-receipt-") as raw:
            receipt_path = self.write_receipt(pathlib.Path(raw), receipt)
            result = self.verify(receipt_path)
        self.assertFalse(result["migration_ledger_present"])
        self.assertEqual(result["migration_receipt_count"], 0)

    def test_unknown_fields_binding_drift_and_incomplete_cleanup_fail(self):
        cases: list[tuple[str, dict[str, object], str]] = []
        unknown = valid_restore_receipt()
        unknown["query"] = "must not be accepted"
        cases.append(("unknown", unknown, "invalid_restore_receipt_schema"))

        source_drift = valid_restore_receipt()
        source_drift["source"]["volume_id"] = "vol_different123456"  # type: ignore[index]
        cases.append(("source", source_drift, "restore_source_binding_mismatch"))

        snapshot_drift = valid_restore_receipt()
        snapshot_drift["snapshot"]["sha256"] = "d" * 64  # type: ignore[index]
        cases.append(("snapshot", snapshot_drift, "restore_snapshot_binding_mismatch"))

        cleanup = valid_restore_receipt()
        cleanup["cleanup"]["restored_volume_destroyed"] = False  # type: ignore[index]
        cases.append(("cleanup", cleanup, "restore_cleanup_incomplete"))

        for label, document, error_code in cases:
            with self.subTest(label=label), tempfile.TemporaryDirectory(
                prefix="nhs-restore-receipt-"
            ) as raw:
                receipt_path = self.write_receipt(pathlib.Path(raw), document)
                with self.assertRaisesRegex(cutover.VerificationError, error_code):
                    self.verify(receipt_path)

    def test_restore_target_must_be_isolated_and_match_the_source_database(self):
        wrong_database = valid_restore_receipt()
        wrong_database["restore"]["database_identity_sha256"] = "c" * 64  # type: ignore[index]

        same_app = valid_restore_receipt()
        same_app["restore"]["app"] = SOURCE_APP  # type: ignore[index]

        for label, document in (("wrong_database", wrong_database), ("same_app", same_app)):
            with self.subTest(label=label), tempfile.TemporaryDirectory(
                prefix="nhs-restore-receipt-"
            ) as raw:
                receipt_path = self.write_receipt(pathlib.Path(raw), document)
                with self.assertRaisesRegex(
                    cutover.VerificationError, "restore_target_binding_mismatch"
                ):
                    self.verify(receipt_path)

    def test_inventory_and_timeline_invariants_fail_closed(self):
        unsorted = valid_restore_receipt()
        unsorted_rows = unsorted["migration_inventory"]["receipts"]  # type: ignore[index]
        unsorted_rows[0], unsorted_rows[1] = unsorted_rows[1], unsorted_rows[0]

        bad_timeline = valid_restore_receipt()
        bad_timeline["cleanup"]["verified_at"] = "2026-08-02T02:11:00Z"  # type: ignore[index]

        for label, document, error_code in (
            ("inventory", unsorted, "invalid_migration_inventory"),
            ("timeline", bad_timeline, "restore_timeline_invalid"),
        ):
            with self.subTest(label=label), tempfile.TemporaryDirectory(
                prefix="nhs-restore-receipt-"
            ) as raw:
                receipt_path = self.write_receipt(pathlib.Path(raw), document)
                with self.assertRaisesRegex(cutover.VerificationError, error_code):
                    self.verify(receipt_path)

    def test_symlink_receipt_is_rejected(self):
        with tempfile.TemporaryDirectory(prefix="nhs-restore-receipt-") as raw:
            directory = pathlib.Path(raw)
            target = self.write_receipt(directory, valid_restore_receipt())
            link = directory / "linked-receipt.json"
            link.symlink_to(target)
            with self.assertRaisesRegex(
                cutover.VerificationError, "restore_receipt_unavailable"
            ):
                self.verify(link)


class DatabaseVerificationTest(unittest.TestCase):
    def make_fake_psql(self, directory: pathlib.Path) -> pathlib.Path:
        fake = directory / "psql-fixture"
        fake.write_text("#!/bin/sh\nexit 99\n", encoding="utf-8")
        fake.chmod(0o700)
        return fake

    def environment(self, fake_psql: pathlib.Path) -> dict[str, str]:
        return {
            cutover.DATABASE_URL_ENV: (
                "postgresql://nhs_user:private%20fixture@db.internal:5432/nhs"
                "?sslmode=disable"
            ),
            cutover.PSQL_BINARY_ENV: str(fake_psql),
        }

    def test_explicit_database_check_uses_env_only_and_is_read_only(self):
        with tempfile.TemporaryDirectory(prefix="nhs-cutover-db-") as raw:
            directory = pathlib.Path(raw)
            fake_psql = self.make_fake_psql(directory)
            migrations_dir = directory / "migrations"
            migration_hashes = write_migration_files(migrations_dir)
            environment = self.environment(fake_psql)
            calls: list[tuple[object, object, object]] = []

            def fake_runner(command, *, env, timeout):
                calls.append((command, env, timeout))
                return subprocess.CompletedProcess(
                    command,
                    0,
                    stdout=json.dumps(migration_rows(hashes=migration_hashes)) + "\n",
                    stderr="",
                )

            result = cutover.verify_database_migrations(
                REVISION,
                environ=environment,
                migrations_dir=migrations_dir,
                runner=fake_runner,
            )

        self.assertTrue(result["ok"])
        self.assertTrue(result["read_only"])
        self.assertTrue(result["migrations_dir_verified"])
        self.assertEqual(result["protected_migration_count"], 12)
        self.assertFalse(result["deployment_ready"])
        self.assertEqual(len(calls), 1)
        command, child_environment, timeout = calls[0]
        self.assertNotIn(environment[cutover.DATABASE_URL_ENV], command)
        self.assertNotIn(cutover.DATABASE_URL_ENV, child_environment)
        self.assertEqual(child_environment["PGHOST"], "db.internal")
        self.assertEqual(child_environment["PGPASSWORD"], "private fixture")
        self.assertIn("default_transaction_read_only=on", child_environment["PGOPTIONS"])
        self.assertIn("SELECT", command[-1])
        for forbidden in ("INSERT", "UPDATE", "DELETE", "ALTER", "CREATE", "DROP"):
            self.assertNotIn(forbidden, command[-1].upper())
        self.assertEqual(timeout, cutover.DATABASE_TIMEOUT_SECONDS)

    def test_missing_extra_or_wrong_revision_receipts_fail(self):
        for label in ("missing", "extra", "wrong_revision", "wrong_sha"):
            with self.subTest(label=label), tempfile.TemporaryDirectory(prefix="nhs-cutover-db-") as raw:
                directory = pathlib.Path(raw)
                fake_psql = self.make_fake_psql(directory)
                migrations_dir = directory / "migrations"
                migration_hashes = write_migration_files(migrations_dir)
                rows = migration_rows(hashes=migration_hashes)
                if label == "missing":
                    rows = rows[:-1]
                elif label == "extra":
                    rows.append(
                        {
                            "name": "029_future.sql",
                            "sha256": "e" * 64,
                            "applied_by_commit": REVISION,
                        }
                    )
                elif label == "wrong_revision":
                    rows[-1] = {**rows[-1], "applied_by_commit": OTHER_REVISION}
                else:
                    rows[-1] = {**rows[-1], "sha256": "f" * 64}

                def fake_runner(command, *, env, timeout):
                    return subprocess.CompletedProcess(
                        command, 0, stdout=json.dumps(rows), stderr=""
                    )

                expected = (
                    "protected_migration_revision_mismatch"
                    if label == "wrong_revision"
                    else "protected_migration_sha256_mismatch"
                    if label == "wrong_sha"
                    else "protected_migration_set_mismatch"
                )
                with self.assertRaisesRegex(cutover.VerificationError, expected):
                    cutover.verify_database_migrations(
                        REVISION,
                        environ=self.environment(fake_psql),
                        migrations_dir=migrations_dir,
                        runner=fake_runner,
                    )

    def test_migration_files_must_be_regular_bounded_and_not_symlinks(self):
        cases = ("missing", "symlink", "directory", "oversized")
        for label in cases:
            with self.subTest(label=label), tempfile.TemporaryDirectory(
                prefix="nhs-cutover-migrations-"
            ) as raw:
                root = pathlib.Path(raw)
                migrations_dir = root / "migrations"
                write_migration_files(migrations_dir)
                target = migrations_dir / cutover.PROTECTED_MIGRATIONS[-1]
                if label == "missing":
                    target.unlink()
                    expected = "migration_file_unavailable"
                elif label == "symlink":
                    original = root / "real-migration.sql"
                    target.replace(original)
                    target.symlink_to(original)
                    expected = "migration_file_unavailable"
                elif label == "directory":
                    target.unlink()
                    target.mkdir()
                    expected = "migration_file_invalid"
                else:
                    target.write_bytes(b"x" * (cutover.MAX_MIGRATION_BYTES + 1))
                    expected = "migration_file_invalid"
                with self.assertRaisesRegex(cutover.VerificationError, expected):
                    cutover._migration_file_hashes(migrations_dir)

    def test_confirmation_and_database_url_are_mandatory(self):
        with mock.patch.object(cutover, "verify_database_migrations") as verify:
            with self.assertRaisesRegex(
                cutover.VerificationError, "database_check_confirmation_required"
            ):
                cutover.run(["database", "--revision", REVISION], environ={})
            verify.assert_not_called()

        with self.assertRaisesRegex(cutover.VerificationError, "database_url_unavailable"):
            cutover.verify_database_migrations(
                REVISION,
                environ={cutover.PSQL_BINARY_ENV: "/must/not/run"},
            )

    def test_database_errors_are_sanitized(self):
        with tempfile.TemporaryDirectory(prefix="nhs-cutover-db-") as raw:
            directory = pathlib.Path(raw)
            fake_psql = self.make_fake_psql(directory)
            migrations_dir = directory / "migrations"
            write_migration_files(migrations_dir)

            def failing_runner(command, *, env, timeout):
                return subprocess.CompletedProcess(
                    command,
                    1,
                    stdout="",
                    stderr="postgresql://secret-user:secret-password@private/db",
                )

            with self.assertRaisesRegex(cutover.VerificationError, "database_check_failed") as failure:
                cutover.verify_database_migrations(
                    REVISION,
                    environ=self.environment(fake_psql),
                    migrations_dir=migrations_dir,
                    runner=failing_runner,
                )
        self.assertNotIn("secret", str(failure.exception))


class CommandSurfaceTest(unittest.TestCase):
    def test_default_path_does_not_touch_database(self):
        stderr = io.StringIO()
        with (
            mock.patch.object(cutover.resource, "setrlimit"),
            mock.patch.object(cutover, "verify_database_migrations") as verify,
            contextlib.redirect_stderr(stderr),
        ):
            status = cutover.main([])
        self.assertEqual(status, 1)
        verify.assert_not_called()
        self.assertEqual(json.loads(stderr.getvalue()), {"error": "invalid_arguments", "ok": False})

    def test_help_and_offline_fixture_never_invoke_psql(self):
        with tempfile.TemporaryDirectory(prefix="nhs-cutover-command-") as raw:
            directory = pathlib.Path(raw)
            marker = directory / "psql-was-run"
            fake_psql = directory / "psql-fixture"
            fake_psql.write_text(
                "#!/bin/sh\n/usr/bin/touch \"$NHS_PSQL_MARKER\"\nexit 88\n",
                encoding="utf-8",
            )
            fake_psql.chmod(0o700)
            receipt_path = directory / "restore.json"
            receipt_path.write_text(json.dumps(valid_restore_receipt()), encoding="utf-8")
            environment = os.environ.copy()
            environment.update(
                {
                    cutover.DATABASE_URL_ENV: "postgresql://private:secret@db.internal/nhs",
                    cutover.PSQL_BINARY_ENV: str(fake_psql),
                    "NHS_PSQL_MARKER": str(marker),
                }
            )
            help_process = subprocess.run(
                [sys.executable, str(MODULE_PATH), "--help"],
                check=False,
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                env=environment,
            )
            fixture_process = subprocess.run(
                [sys.executable, str(MODULE_PATH), *restore_arguments(receipt_path)],
                check=False,
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                env=environment,
            )
        self.assertEqual(help_process.returncode, 0, help_process.stderr)
        self.assertIn("restore-receipt", help_process.stdout)
        self.assertEqual(fixture_process.returncode, 0, fixture_process.stderr)
        self.assertTrue(json.loads(fixture_process.stdout)["ok"])
        self.assertFalse(marker.exists())


if __name__ == "__main__":
    unittest.main()
