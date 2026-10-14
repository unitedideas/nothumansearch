#!/usr/bin/env python3

from __future__ import annotations

import argparse
import importlib.util
import json
import subprocess
import unittest
from pathlib import Path


MODULE_PATH = Path(__file__).with_name("stage1-fly-read.py")
SPEC = importlib.util.spec_from_file_location("stage1_fly_read", MODULE_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
SPEC.loader.exec_module(MODULE)

REVISION = "9" * 40
MACHINE = "e2869126f75686"
DIGEST = "sha256:" + "a" * 64
ARCHIVE = "b" * 64


def args() -> argparse.Namespace:
    return argparse.Namespace(
        revision=REVISION,
        machine=MACHINE,
        days=30,
        app="nothumansearch",
        flyctl="flyctl",
        codex_secret="codex-secret",
    )


def inventory(mode: str = "disabled") -> str:
    value = [{
        "id": MACHINE,
        "state": "started",
        "image_ref": {
            "digest": DIGEST,
            "labels": {
                "org.opencontainers.image.revision": REVISION,
                "org.opencontainers.image.source_archive_sha256": ARCHIVE,
            },
        },
        "config": {
            "image": "registry.fly.io/nothumansearch:test",
            "env": {"NHS_PROVIDER_EXCHANGE_MODE": mode},
        },
    }]
    return '{"event":"secret_injection_ready","bindings":[]}\n' + json.dumps(value)


def receipt(extra: dict | None = None) -> str:
    value = {
        "contract": "nhs-stage1-demand-read-v1",
        "candidate_revision": REVISION,
        "binary_revision": REVISION,
        "stage1_demand": {
            "synthetic_excluded": True,
            "counts_are_receipts_not_unique_agents": True,
            "commercial_proof": False,
            "stage1_ready": False,
        },
        "action_interest_attempt_funnel": {
            "counts_are_attempts_not_unique_agents": True,
            "contains_request_coordinates": False,
            "commercial_proof": False,
        },
        "independent_read_only_snapshots": True,
        "searches_are_not_leads": True,
        "readiness_does_not_authorize_stage2": True,
        "contains_identifiers": False,
        "contains_queries_or_prompts": False,
        "contains_contact_data": False,
        "operator_contacted_provider": False,
        "operator_changed_commercial_state": False,
        "operator_affected_organic_rank": False,
    }
    if extra:
        value.update(extra)
    value["stage1_report_sha256"] = MODULE._sha256_json(value["stage1_demand"])
    value["attempt_funnel_sha256"] = MODULE._sha256_json(value["action_interest_attempt_funnel"])
    return "Connecting...\n" + json.dumps(value)


class FakeRunner:
    def __init__(self, outputs: list[str]) -> None:
        self.outputs = outputs
        self.commands: list[list[str]] = []

    def __call__(self, command: list[str], **_: object) -> subprocess.CompletedProcess[str]:
        self.commands.append(command)
        return subprocess.CompletedProcess(command, 0, self.outputs.pop(0), "")


class Stage1FlyReadTest(unittest.TestCase):
    def test_collects_exact_disabled_existing_machine_receipt(self) -> None:
        runner = FakeRunner([inventory(), receipt()])
        evidence = MODULE.collect(args(), runner)
        self.assertEqual(evidence["contract"], "nhs-stage1-fly-read-evidence-v1")
        self.assertEqual(evidence["operator_run"]["image_digest"], DIGEST)
        self.assertEqual(evidence["operator_run"]["provider_exchange_mode"], "disabled")
        self.assertTrue(evidence["operator_run"]["used_existing_machine"])
        self.assertFalse(evidence["operator_run"]["created_or_destroyed_machine"])
        self.assertEqual(len(runner.commands), 2)
        self.assertIn("--quiet", runner.commands[1])
        self.assertIn("FLY_API_TOKEN=FLY_API_TOKEN", runner.commands[0])
        self.assertNotIn("--access-token", runner.commands[0])

    def test_refuses_non_disabled_machine_before_ssh(self) -> None:
        runner = FakeRunner([inventory("shadow")])
        with self.assertRaisesRegex(MODULE.ReadError, "not explicitly disabled"):
            MODULE.collect(args(), runner)
        self.assertEqual(len(runner.commands), 1)

    def test_refuses_revision_mismatch_before_ssh(self) -> None:
        raw = inventory().replace(REVISION, "8" * 40)
        runner = FakeRunner([raw])
        with self.assertRaisesRegex(MODULE.ReadError, "revision does not match"):
            MODULE.collect(args(), runner)
        self.assertEqual(len(runner.commands), 1)

    def test_refuses_sensitive_receipt_field(self) -> None:
        runner = FakeRunner([inventory(), receipt({"query": "private"})])
        with self.assertRaisesRegex(MODULE.ReadError, "forbidden field query"):
            MODULE.collect(args(), runner)

    def test_refuses_false_privacy_assertion(self) -> None:
        runner = FakeRunner([inventory(), receipt({"contains_contact_data": True})])
        with self.assertRaisesRegex(MODULE.ReadError, "violates a required safety assertion"):
            MODULE.collect(args(), runner)

    def test_refuses_tampered_projection_digest(self) -> None:
        raw = receipt().replace('"stage1_ready": false', '"stage1_ready": true')
        runner = FakeRunner([inventory(), raw])
        with self.assertRaisesRegex(MODULE.ReadError, "projection digest does not match"):
            MODULE.collect(args(), runner)


if __name__ == "__main__":
    unittest.main()
