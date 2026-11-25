#!/usr/bin/env python3

from __future__ import annotations

import argparse
import importlib.util
import json
import subprocess
import tempfile
import unittest
from pathlib import Path


PATH = Path(__file__).with_name("nhs-demand-trigger-monitor.py")
SPEC = importlib.util.spec_from_file_location("nhs_demand_trigger_monitor", PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
SPEC.loader.exec_module(MODULE)

REVISION = "a" * 40
DIGEST = "sha256:" + "b" * 64
MACHINE = "e2869126f75686"
SINCE = "2026-08-11T09:41:28Z"


def funnel(selected: int = 0, interested: int = 0) -> dict:
    report = {
        "since": SINCE,
        "commercial_state_events_total": 0,
        "developer_tools_search_receipts": 15,
        "developer_tools_result_selections": selected,
        "developer_tools_search_receipts_with_selection": selected,
        "developer_tools_action_interest_receipts": interested,
        "developer_tools_search_receipts_with_action_interest": interested,
        "developer_tools_post_selection_action_interest_receipts": interested,
        "developer_tools_post_selection_search_receipts": interested,
    }
    return {
        "contract": "nhs-post-selection-action-interest-experiment-read-v3",
        "report_sha256": MODULE.report_digest(report),
        "candidate_revision": REVISION,
        "binary_revision": REVISION,
        "report": report,
        "contains_identifiers": False,
        "contains_queries_or_prompts": False,
        "contains_contact_data": False,
        "operator_contacted_provider": False,
        "operator_changed_commercial_state": False,
        "operator_affected_organic_rank": False,
    }


def stage1(ready: bool = False) -> dict:
    report = {
        "as_of": "2026-08-12T03:00:00Z",
        "stage1_ready": ready,
        "search_receipts_with_selection": 7,
        "search_receipts_with_action_interest": 0,
    }
    attempts = {"total_attempts": 37}
    return {
        "contract": "nhs-stage1-demand-read-v1",
        "stage1_report_sha256": MODULE.report_digest(report),
        "attempt_funnel_sha256": MODULE.report_digest(attempts),
        "candidate_revision": REVISION,
        "binary_revision": REVISION,
        "stage1_demand": report,
        "action_interest_attempt_funnel": attempts,
        "contains_identifiers": False,
        "contains_queries_or_prompts": False,
        "contains_contact_data": False,
        "operator_contacted_provider": False,
        "operator_changed_commercial_state": False,
    }


def inventory(mode: str = "disabled") -> str:
    return json.dumps([{
        "id": MACHINE,
        "state": "started",
        "image_ref": {
            "digest": DIGEST,
            "labels": {"org.opencontainers.image.revision": REVISION},
        },
        "config": {"env": {"NHS_PROVIDER_EXCHANGE_MODE": mode}},
    }])


class FakeRunner:
    def __init__(self, outputs: list[str]) -> None:
        self.outputs = outputs
        self.commands: list[list[str]] = []

    def __call__(self, command: list[str], **_: object) -> subprocess.CompletedProcess[str]:
        self.commands.append(command)
        output = self.outputs.pop(0) if self.outputs else ""
        return subprocess.CompletedProcess(command, 0, output, "")


class MonitorTest(unittest.TestCase):
    def setup_files(self, temporary: str) -> argparse.Namespace:
        root = Path(temporary)
        baseline = root / "baseline.json"
        baseline.write_text(json.dumps({"reader_receipt": funnel()}), encoding="utf-8")
        control = root / "control.json"
        control.write_text(json.dumps({"enabled": True}), encoding="utf-8")
        return argparse.Namespace(
            baseline=str(baseline), control=str(control), state=str(root / "state.json"),
            trigger=str(root / "trigger.json"), app="nothumansearch",
            flyctl="flyctl", codex_secret="codex-secret",
        )

    def test_quiet_recheck_records_no_trigger(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            args = self.setup_files(temporary)
            runner = FakeRunner([inventory(), json.dumps(funnel()), json.dumps(stage1())])
            result = MODULE.monitor(args, runner)
            self.assertFalse(result["triggered"])
            self.assertEqual(result["exit_code"], 0)
            self.assertFalse(result["notification_emitted"])
            self.assertFalse(Path(args.trigger).exists())
            self.assertEqual(len(runner.commands), 3)

    def test_new_interest_writes_trigger_and_notifies_once(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            args = self.setup_files(temporary)
            outputs = [inventory(), json.dumps(funnel(1, 1)), json.dumps(stage1())]
            runner = FakeRunner(outputs)
            first = MODULE.monitor(args, runner)
            self.assertTrue(first["triggered"])
            self.assertTrue(first["notification_emitted"])
            self.assertTrue(Path(args.trigger).exists())
            runner = FakeRunner([inventory(), json.dumps(funnel(1, 1)), json.dumps(stage1())])
            second = MODULE.monitor(args, runner)
            self.assertFalse(second["notification_emitted"])

    def test_refuses_non_disabled_mode_before_ssh(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            args = self.setup_files(temporary)
            runner = FakeRunner([inventory("shadow")])
            with self.assertRaisesRegex(MODULE.MonitorError, "exactly one started disabled-mode"):
                MODULE.monitor(args, runner)
            self.assertEqual(len(runner.commands), 1)

    def test_commercial_state_fails_closed(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            args = self.setup_files(temporary)
            changed = funnel()
            changed["report"]["commercial_state_events_total"] = 1
            changed["report_sha256"] = MODULE.report_digest(changed["report"])
            runner = FakeRunner([inventory(), json.dumps(changed)])
            with self.assertRaisesRegex(MODULE.MonitorError, "commercial state changed"):
                MODULE.monitor(args, runner)


if __name__ == "__main__":
    unittest.main()
