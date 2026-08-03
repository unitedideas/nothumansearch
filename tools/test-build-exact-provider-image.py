#!/usr/bin/env python3
"""Contract tests for the exact-archive local NHS image builder."""

from __future__ import annotations

import hashlib
import os
import pathlib
import subprocess
import tempfile
import unittest


SCRIPT = pathlib.Path(__file__).with_name("build-exact-provider-image.sh")


def run(
    *args: str,
    cwd: pathlib.Path,
    check: bool = True,
    env: dict[str, str] | None = None,
) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        list(args), cwd=cwd, check=check, text=True, env=env,
        stdout=subprocess.PIPE, stderr=subprocess.PIPE,
    )


class ExactProviderImageBuildTest(unittest.TestCase):
    def test_reconstructs_verified_archive_and_never_claims_registry_readiness(self) -> None:
        with tempfile.TemporaryDirectory(prefix="nhs-image-build-test-") as raw:
            root = pathlib.Path(raw)
            candidate = root / "candidate"
            candidate.mkdir()
            run("git", "init", "-q", cwd=candidate)
            run("git", "config", "user.email", "fixture@example.test", cwd=candidate)
            run("git", "config", "user.name", "Fixture", cwd=candidate)
            (candidate / "Dockerfile").write_text("FROM scratch\n", encoding="utf-8")
            (candidate / "source.txt").write_text("base\n", encoding="utf-8")
            run("git", "add", ".", cwd=candidate)
            run("git", "commit", "-q", "-m", "base", cwd=candidate)
            parent = run("git", "rev-parse", "HEAD", cwd=candidate).stdout.strip()
            (candidate / "source.txt").write_text("candidate\n", encoding="utf-8")
            run("git", "add", "source.txt", cwd=candidate)
            run("git", "commit", "-q", "-m", "candidate", cwd=candidate)
            commit = run("git", "rev-parse", "HEAD", cwd=candidate).stdout.strip()
            tree = run("git", "rev-parse", "HEAD^{tree}", cwd=candidate).stdout.strip()

            archive = root / "verified-source.tar"
            run("git", "archive", "--format=tar", f"--output={archive}", commit, cwd=candidate)
            archive_sha256 = hashlib.sha256(archive.read_bytes()).hexdigest()
            manifest = root / "release-manifest.txt"
            manifest.write_text(
                "\n".join((
                    "contract=nhs-exact-release-verification-v2",
                    f"release_commit={commit}",
                    f"release_base_commit={parent}",
                    f"release_tree={tree}",
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

            captured_archive = root / "builder-context.tar"
            fake_builder = root / "fake-builder.sh"
            fake_builder.write_text(
                """#!/bin/bash
set -euo pipefail
if [ "$1" = "build" ]; then
    shift
    iid_file=""
    while [ "$#" -gt 0 ]; do
        case "$1" in
            --iidfile) iid_file="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    /bin/cat >"$FAKE_ARCHIVE_CAPTURE"
    /usr/bin/printf 'sha256:%064d\n' 0 >"$iid_file"
    exit 0
fi
if [ "$1" = "image" ] && [ "$2" = "inspect" ]; then
    case "$4" in
        *source_archive_sha256*) /usr/bin/printf '%s\n' "$EXPECTED_ARCHIVE_SHA" ;;
        *) /usr/bin/printf '%s\n' "$EXPECTED_COMMIT" ;;
    esac
    exit 0
fi
exit 1
""",
                encoding="utf-8",
            )
            fake_builder.chmod(0o755)
            receipt = root / "local-image.receipt"
            command = (
                str(SCRIPT), str(candidate), str(manifest), "nhs-provider:test",
                str(receipt), "--confirm-owner-authorized",
            )
            denied = run(*command[:-1], cwd=root, check=False)
            self.assertEqual(denied.returncode, 2)
            self.assertFalse(receipt.exists())

            env = os.environ.copy()
            env.update({
                "NHS_OCI_BUILD_BINARY": str(fake_builder),
                "FAKE_ARCHIVE_CAPTURE": str(captured_archive),
                "EXPECTED_COMMIT": commit,
                "EXPECTED_ARCHIVE_SHA": archive_sha256,
            })
            built = run(*command, cwd=root, env=env)
            self.assertEqual(hashlib.sha256(captured_archive.read_bytes()).hexdigest(), archive_sha256)
            self.assertEqual(receipt.read_text(encoding="utf-8"), built.stdout)
            fields = dict(
                line.split("=", 1)
                for line in receipt.read_text(encoding="utf-8").splitlines()
            )
            self.assertEqual(fields["contract"], "nhs-provider-local-image-v1")
            self.assertEqual(fields["release_commit"], commit)
            self.assertEqual(fields["source_archive_sha256"], archive_sha256)
            self.assertEqual(fields["registry_digest_verified"], "false")
            self.assertEqual(fields["deployment_ready"], "false")
            self.assertEqual(fields["push_command_emitted"], "false")
            self.assertEqual(fields["deployment_command_emitted"], "false")

            receipt_bytes = receipt.read_bytes()
            no_clobber = run(*command, cwd=root, check=False, env=env)
            self.assertNotEqual(no_clobber.returncode, 0)
            self.assertIn("output receipt must be a new file", no_clobber.stderr)
            self.assertEqual(receipt.read_bytes(), receipt_bytes)

            symlink_target = root / "must-not-be-created.receipt"
            symlink_receipt = root / "broken-symlink.receipt"
            symlink_receipt.symlink_to(symlink_target)
            symlink_rejected = run(
                str(SCRIPT), str(candidate), str(manifest), "nhs-provider:symlink",
                str(symlink_receipt), "--confirm-owner-authorized",
                cwd=root, check=False, env=env,
            )
            self.assertNotEqual(symlink_rejected.returncode, 0)
            self.assertIn("output receipt must be a new file", symlink_rejected.stderr)
            self.assertTrue(symlink_receipt.is_symlink())
            self.assertFalse(symlink_target.exists())

            receipt_directory = root / "receipt-directory"
            receipt_directory.mkdir()
            directory_symlink = root / "directory-symlink.receipt"
            directory_symlink.symlink_to(receipt_directory, target_is_directory=True)
            directory_symlink_rejected = run(
                str(SCRIPT), str(candidate), str(manifest), "nhs-provider:directory-link",
                str(directory_symlink), "--confirm-owner-authorized",
                cwd=root, check=False, env=env,
            )
            self.assertNotEqual(directory_symlink_rejected.returncode, 0)
            self.assertIn(
                "output receipt must be a new file", directory_symlink_rejected.stderr,
            )
            self.assertTrue(directory_symlink.is_symlink())
            self.assertEqual(list(receipt_directory.iterdir()), [])

            (candidate / "source.txt").write_text("post-verification mutation\n", encoding="utf-8")
            rejected = run(
                str(SCRIPT), str(candidate), str(manifest), "nhs-provider:test2",
                str(root / "mutated.receipt"), "--confirm-owner-authorized",
                cwd=root, check=False, env=env,
            )
            self.assertNotEqual(rejected.returncode, 0)


if __name__ == "__main__":
    unittest.main()
