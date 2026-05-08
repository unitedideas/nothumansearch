#!/usr/bin/env python3
import importlib.util
import pathlib
import tempfile
import unittest


MODULE_PATH = pathlib.Path(__file__).with_name("discovery-quality-report.py")
SPEC = importlib.util.spec_from_file_location("discovery_quality_report", MODULE_PATH)
reporter = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
SPEC.loader.exec_module(reporter)


class DiscoveryQualityReportTest(unittest.TestCase):
    def test_buckets_hard_and_low_signal_rows_without_domain_output(self):
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

        self.assertEqual(result["sample_rows"], 5)
        self.assertEqual(result["hard_signal_rows"], 1)
        self.assertEqual(result["low_signal_rows"], 4)
        self.assertEqual(
            result["primary_buckets"],
            {
                "hard_agent_signal": 1,
                "llms_only": 1,
                "passive_or_soft_signal": 1,
                "schema_only": 1,
                "zero_score": 1,
            },
        )
        self.assertEqual(
            result["category_other"],
            {"total": 3, "hard_agent_signal": 0, "low_signal": 3},
        )
        self.assertNotIn("example", str(result))


if __name__ == "__main__":
    unittest.main()
