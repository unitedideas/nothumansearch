#!/usr/bin/env python3
import importlib.util
import json
import pathlib
import sys
import tempfile
import unittest


MODULE_PATH = pathlib.Path(__file__).with_name("discovery-quality-report.py")
SPEC = importlib.util.spec_from_file_location("discovery_quality_report", MODULE_PATH)
reporter = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = reporter
SPEC.loader.exec_module(reporter)

QUARANTINE_MODULE_PATH = pathlib.Path(__file__).with_name("discovery-quarantine-report.py")
QUARANTINE_SPEC = importlib.util.spec_from_file_location(
    "discovery_quarantine_report",
    QUARANTINE_MODULE_PATH,
)
quarantine_reporter = importlib.util.module_from_spec(QUARANTINE_SPEC)
assert QUARANTINE_SPEC.loader is not None
sys.modules[QUARANTINE_SPEC.name] = quarantine_reporter
QUARANTINE_SPEC.loader.exec_module(quarantine_reporter)


class DiscoveryQualityReportTest(unittest.TestCase):
    def test_builds_quarantine_compatible_artifact_without_domain_output(self):
        sample = """2026/05/04 12:09:19   api.example score=65 cat=ai-tools signals=[llms.txt, OpenAPI, API, AI-bots]
2026/05/04 12:09:20   llms.example score=25 cat=other signals=[llms.txt]
2026/05/04 12:09:21   schema.example score=5 cat=developer signals=[schema.org]
2026/05/04 12:09:22   zero.example score=0 cat=other signals=[]
2026/05/04 12:09:23   robots.example score=10 cat=other signals=[AI-bots, schema.org]
"""
        with tempfile.NamedTemporaryFile("w+", encoding="utf-8") as handle:
            handle.write(sample)
            handle.flush()
            rows = reporter.parse_rows(pathlib.Path(handle.name))

        result = reporter.build_report(rows)

        self.assertEqual(
            set(result),
            {"counts", "sample_breakdown", "quality_gate"},
        )
        self.assertEqual(result["counts"]["sample_rows"], 5)
        self.assertEqual(result["counts"]["hard_signal_rows"], 1)
        self.assertEqual(result["counts"]["low_signal_rows"], 4)
        self.assertEqual(result["counts"]["category_other_low_signal"], 3)
        self.assertEqual(result["counts"]["category_other_hard_agent_signal"], 0)
        self.assertEqual(
            result["sample_breakdown"],
            {
                "category_other": 3,
                "category_other_hard_agent_signal": 0,
                "category_other_low_signal": 3,
                "hard_agent_signal": 1,
                "llms_only": 1,
                "passive_or_soft_signal": 1,
                "schema_only": 1,
                "zero_score": 1,
            },
        )
        self.assertEqual(
            result["quality_gate"],
            {
                "status": "review",
                "trigger": "low_signal_rows_exceed_hard_signal_rows",
            },
        )
        self.assertNotIn("example", json.dumps(result))

        with tempfile.NamedTemporaryFile("w+", encoding="utf-8") as handle:
            json.dump(result, handle)
            handle.flush()
            loaded = quarantine_reporter.load_planner_artifact(pathlib.Path(handle.name))

        quarantine = quarantine_reporter.build_quarantine_report(loaded)
        self.assertTrue(quarantine["quarantine"]["active"])
        self.assertNotIn("example", json.dumps(quarantine))

    def test_legacy_report_keeps_old_aggregate_shape(self):
        rows = [
            reporter.DiscoveryRow(
                score=65,
                category="ai-tools",
                signals=frozenset({"OpenAPI", "API"}),
            )
        ]
        result = reporter.build_legacy_report(rows)

        self.assertEqual(result["sample_rows"], 1)
        self.assertEqual(result["hard_signal_rows"], 1)
        self.assertEqual(
            result["category_other"],
            {"total": 0, "hard_agent_signal": 0, "low_signal": 0},
        )


if __name__ == "__main__":
    unittest.main()
