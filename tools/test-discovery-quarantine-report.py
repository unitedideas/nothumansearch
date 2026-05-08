#!/usr/bin/env python3
import importlib.util
import json
import pathlib
import sys
import tempfile
import unittest


MODULE_PATH = pathlib.Path(__file__).with_name("discovery-quarantine-report.py")
SPEC = importlib.util.spec_from_file_location("discovery_quarantine_report", MODULE_PATH)
reporter = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = reporter
SPEC.loader.exec_module(reporter)


def artifact(**overrides):
    data = {
        "counts": {
            "category_other_hard_agent_signal": 2,
            "category_other_low_signal": 7,
            "hard_signal_rate": 0.2,
            "hard_signal_rows": 3,
            "low_signal_rows": 8,
            "sample_rows": 11,
        },
        "quality_gate": {
            "status": "review",
            "trigger": "low_signal_rows_exceed_hard_signal_rows",
        },
        "sample_breakdown": {
            "category_other": 9,
            "category_other_hard_agent_signal": 2,
            "category_other_low_signal": 7,
            "hard_agent_signal": 3,
            "llms_only": 2,
            "schema_only": 1,
            "zero_score": 4,
        },
    }
    data.update(overrides)
    return data


class DiscoveryQuarantineReportTest(unittest.TestCase):
    def test_builds_quarantine_from_sanitized_aggregate_artifact(self):
        report = reporter.build_quarantine_report(artifact())

        self.assertTrue(report["quarantine"]["active"])
        self.assertEqual(report["public_guards"]["public_search"], "protected_by_models.AgentFirstFilter")
        self.assertEqual(report["public_guards"]["score_fix_targeting"], "requires_has_hard_agent_signal")
        self.assertEqual(report["public_guards"]["planner_priority"], "quarantine_first")
        self.assertFalse(report["artifact_policy"]["domain_output"])
        self.assertNotIn("example", json.dumps(report))

    def test_rejects_unsanitized_candidate_fields(self):
        with tempfile.NamedTemporaryFile("w+", encoding="utf-8") as handle:
            json.dump(artifact(candidate_domains=["example.com"]), handle)
            handle.flush()

            with self.assertRaisesRegex(reporter.SanitizedArtifactError, "not sanitized"):
                reporter.load_planner_artifact(pathlib.Path(handle.name))

    def test_writes_aggregate_only_output(self):
        with tempfile.TemporaryDirectory() as tmp:
            input_path = pathlib.Path(tmp) / "planner.json"
            output_path = pathlib.Path(tmp) / "quarantine.json"
            input_path.write_text(json.dumps(artifact()) + "\n", encoding="utf-8")

            loaded = reporter.load_planner_artifact(input_path)
            output_path.write_text(
                json.dumps(reporter.build_quarantine_report(loaded), sort_keys=True) + "\n",
                encoding="utf-8",
            )
            rendered = output_path.read_text(encoding="utf-8")

        self.assertIn("low_signal_rows", rendered)
        self.assertNotIn("candidate_domains", rendered)
        self.assertNotIn("example.com", rendered)

    def test_builds_weekly_history_entry_without_candidate_fields(self):
        report = reporter.build_quarantine_report(artifact())
        entry = reporter.build_history_entry(report, "2026-05-08T12:34:56Z")

        self.assertEqual(entry["history_key"], "discovery-quarantine:2026-05-04")
        self.assertEqual(entry["week_start"], "2026-05-04")
        self.assertEqual(entry["sample_rows"], 11)
        self.assertEqual(entry["hard_signal_rows"], 3)
        self.assertEqual(entry["low_signal_rows"], 8)
        self.assertEqual(entry["category_other_low_signal"], 7)
        self.assertEqual(entry["quarantine"], {"active": True})
        self.assertEqual(entry["planner_priority"], "quarantine_first")
        self.assertIn("business-local only", entry["planner_scope"])
        self.assertNotIn("example", json.dumps(entry))

    def test_appends_history_entry_as_jsonl(self):
        with tempfile.TemporaryDirectory() as tmp:
            history_path = pathlib.Path(tmp) / "history.jsonl"
            report = reporter.build_quarantine_report(artifact())
            entry = reporter.build_history_entry(report, "2026-05-08T12:34:56Z")

            reporter.append_history_entry(history_path, entry)

            lines = history_path.read_text(encoding="utf-8").splitlines()

        self.assertEqual(len(lines), 1)
        loaded = json.loads(lines[0])
        self.assertEqual(loaded["history_key"], "discovery-quarantine:2026-05-04")
        self.assertNotIn("candidate_domains", lines[0])
        self.assertNotIn("example.com", lines[0])

    def test_history_entry_replaces_same_week_key(self):
        with tempfile.TemporaryDirectory() as tmp:
            history_path = pathlib.Path(tmp) / "history.jsonl"
            report = reporter.build_quarantine_report(artifact())

            reporter.append_history_entry(
                history_path,
                reporter.build_history_entry(report, "2026-05-08T12:34:56Z"),
            )
            reporter.append_history_entry(
                history_path,
                reporter.build_history_entry(report, "2026-05-09T12:34:56Z"),
            )

            lines = history_path.read_text(encoding="utf-8").splitlines()

        self.assertEqual(len(lines), 1)
        loaded = json.loads(lines[0])
        self.assertEqual(loaded["history_key"], "discovery-quarantine:2026-05-04")
        self.assertEqual(loaded["observed_at"], "2026-05-09T12:34:56Z")


if __name__ == "__main__":
    unittest.main()
