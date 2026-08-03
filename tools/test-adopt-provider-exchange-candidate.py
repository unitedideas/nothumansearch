#!/usr/bin/env python3
"""Contract tests for the owner-gated exact candidate adoption command."""

from __future__ import annotations

import hashlib
import pathlib
import shutil
import subprocess
import tempfile
import unittest


SCRIPT = pathlib.Path(__file__).with_name("adopt-provider-exchange-candidate.sh")


def run(*args: str, cwd: pathlib.Path, check: bool = True) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        list(args), cwd=cwd, check=check, text=True,
        stdout=subprocess.PIPE, stderr=subprocess.PIPE,
    )


class CandidateAdoptionTest(unittest.TestCase):
    def test_requires_authority_and_adopts_exact_candidate_idempotently(self) -> None:
        with tempfile.TemporaryDirectory(prefix="nhs-adoption-test-") as raw:
            root = pathlib.Path(raw)
            canonical = root / "canonical"
            candidate = root / "candidate"
            canonical.mkdir()
            run("git", "init", "-q", cwd=canonical)
            run("git", "config", "user.email", "fixture@example.test", cwd=canonical)
            run("git", "config", "user.name", "Fixture", cwd=canonical)
            (canonical / "tools").mkdir()
            shutil.copy2(SCRIPT, canonical / "tools" / SCRIPT.name)
            migrations = canonical / "migrations"
            migrations.mkdir()
            migration_names = (
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
            )
            for migration_name in migration_names:
                (migrations / migration_name).write_text(
                    f"-- fixture {migration_name}\n", encoding="utf-8",
                )
            (canonical / "source.txt").write_text("base\n", encoding="utf-8")
            run("git", "add", ".", cwd=canonical)
            run("git", "commit", "-q", "-m", "base", cwd=canonical)
            parent = run("git", "rev-parse", "HEAD", cwd=canonical).stdout.strip()

            run("git", "clone", "-q", str(canonical), str(candidate), cwd=root)
            run("git", "config", "user.email", "fixture@example.test", cwd=candidate)
            run("git", "config", "user.name", "Fixture", cwd=candidate)
            (candidate / "source.txt").write_text("candidate\n", encoding="utf-8")
            run("git", "add", "source.txt", cwd=candidate)
            run("git", "commit", "-q", "-m", "candidate", cwd=candidate)
            commit = run("git", "rev-parse", "HEAD", cwd=candidate).stdout.strip()
            tree = run("git", "rev-parse", "HEAD^{tree}", cwd=candidate).stdout.strip()
            archive = root / "source.tar"
            run("git", "archive", "--format=tar", f"--output={archive}", commit, cwd=candidate)
            archive_sha256 = hashlib.sha256(archive.read_bytes()).hexdigest()
            migration_receipts = tuple(
                f"migration_{migration_name[:3]}_sha256="
                f"{hashlib.sha256((candidate / 'migrations' / migration_name).read_bytes()).hexdigest()}"
                for migration_name in migration_names
            )
            release_manifest = root / "exact-release.manifest"
            release_manifest.write_text(
                "\n".join((
                    "contract=nhs-exact-release-verification-v2",
                    f"release_commit={commit}",
                    f"release_base_commit={parent}",
                    f"release_tree={tree}",
                    "changed_path_count=1",
                    *migration_receipts,
                    f"source_archive_sha256={archive_sha256}",
                    f"build_arg=RELEASE_REVISION={commit}",
                    "exact_archive_tests_passed=true",
                    "postgres_release_tests_passed=true",
                    "disabled_recovery_smoke_passed=true",
                    "preflight_binary_revision_bound=true",
                    "secret_scan_findings=0",
                    "oci_image_digest_verified=false",
                    "target_cutover_preflight_verified=false",
                    "restore_drill_verified=false",
                    "deployment_ready=false",
                    "deployment_command_emitted=false",
                )) + "\n",
                encoding="utf-8",
            )

            denied = run(
                str(canonical / "tools" / SCRIPT.name), str(candidate), commit, tree, parent, str(release_manifest),
                cwd=canonical, check=False,
            )
            self.assertEqual(denied.returncode, 2)
            missing = run("git", "rev-parse", "--verify", f"refs/nhs-provider-candidates/{commit}", cwd=canonical, check=False)
            self.assertNotEqual(missing.returncode, 0)

            command = (
                str(canonical / "tools" / SCRIPT.name), str(candidate), commit, tree, parent,
                str(release_manifest),
                "--confirm-owner-authorized",
            )
            first_process = subprocess.Popen(
                command, cwd=canonical, text=True,
                stdout=subprocess.PIPE, stderr=subprocess.PIPE,
            )
            second_process = subprocess.Popen(
                command, cwd=canonical, text=True,
                stdout=subprocess.PIPE, stderr=subprocess.PIPE,
            )
            first_stdout, first_stderr = first_process.communicate()
            second_stdout, second_stderr = second_process.communicate()
            self.assertEqual(first_process.returncode, 0, first_stderr)
            self.assertEqual(second_process.returncode, 0, second_stderr)
            idempotent = run(*command, cwd=canonical)
            self.assertIn("candidate_adopted=true", first_stdout)
            self.assertIn("deployment_ready=false", first_stdout)
            self.assertIn("deployment_command_emitted=false", first_stdout)
            self.assertEqual(first_stdout, second_stdout)
            self.assertEqual(first_stdout, idempotent.stdout)
            adopted = run("git", "rev-parse", f"refs/nhs-provider-candidates/{commit}", cwd=canonical).stdout.strip()
            self.assertEqual(adopted, commit)
            artifact_dir = canonical / ".git" / "nhs-provider-candidates"
            bundle = artifact_dir / f"{commit}.bundle"
            manifest = artifact_dir / f"{commit}.manifest"
            verified_release = artifact_dir / f"{commit}.release-manifest"
            self.assertTrue(bundle.is_file())
            self.assertTrue(manifest.is_file())
            self.assertEqual(verified_release.read_bytes(), release_manifest.read_bytes())
            self.assertEqual(
                [path.name for path in artifact_dir.iterdir() if path.name.startswith(".")],
                [],
            )
            run("git", "bundle", "verify", str(bundle), cwd=canonical)

            manifest_bytes = manifest.read_bytes()
            symlink_target = root / "must-stay-empty"
            symlink_target.mkdir()
            manifest.unlink()
            manifest.symlink_to(symlink_target, target_is_directory=True)
            symlink_rejected = run(*command, cwd=canonical, check=False)
            self.assertNotEqual(symlink_rejected.returncode, 0)
            self.assertIn("candidate manifest must not be a symbolic link", symlink_rejected.stderr)
            self.assertTrue(manifest.is_symlink())
            self.assertEqual(list(symlink_target.iterdir()), [])
            manifest.unlink()
            manifest.write_bytes(manifest_bytes)

            tampered_manifest = root / "tampered-release.manifest"
            tampered_manifest.write_text(
                release_manifest.read_text(encoding="utf-8").replace(
                    migration_receipts[-1], f"migration_028_sha256={'0' * 64}",
                ),
                encoding="utf-8",
            )
            tampered = run(
                str(canonical / "tools" / SCRIPT.name), str(candidate), commit, tree, parent,
                str(tampered_manifest), "--confirm-owner-authorized",
                cwd=canonical, check=False,
            )
            self.assertNotEqual(tampered.returncode, 0)

            wrong_tree = "0" * 40
            rejected = run(
                str(canonical / "tools" / SCRIPT.name), str(candidate), commit, wrong_tree, parent,
                str(release_manifest),
                "--confirm-owner-authorized", cwd=canonical, check=False,
            )
            self.assertNotEqual(rejected.returncode, 0)


if __name__ == "__main__":
    unittest.main()
