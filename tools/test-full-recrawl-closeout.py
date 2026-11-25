"""Verify closeout reporting with local fixtures and no crawler execution."""

import json
import pathlib
import subprocess
import sys
import tempfile
import unittest


SCRIPT = pathlib.Path(__file__).with_name("full-recrawl-closeout.py")


class FullRecrawlCloseoutTest(unittest.TestCase):
    def run_closeout(self, root, *extra):
        (root / "stats.json").write_text(json.dumps({
            "total_sites": 11, "avg_score": 42, "top_category": "developer",
            "domain": "not-exported.invalid",
        }))
        (root / "categories.json").write_text(json.dumps([
            {"category": "developer", "count": 11, "domain": "not-exported.invalid"}
        ]))
        args = [
            sys.executable, str(SCRIPT), "--date", "2099-01-02",
            "--output", str(root / "report.md"),
            "--health-log", str(root / "health.log"),
            "--stats-fixture", str(root / "stats.json"),
            "--categories-fixture", str(root / "categories.json"),
            "--stats-url", "http://127.0.0.1:9/unused",
            "--categories-url", "http://127.0.0.1:9/unused",
            *extra,
        ]
        result = subprocess.run(args, capture_output=True, text=True, timeout=5)
        self.assertEqual(result.returncode, 0, result.stderr)
        return (root / "report.md").read_text()

    def test_fixture_closeout_preserves_supplied_totals_without_identifiers(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            (root / "health.log").write_text(
                "2099-01-02T12:00:00Z wrapper=full-recrawl event=completion exit=0\n"
            )
            body = self.run_closeout(
                root, "--crawler-success", "9", "--crawler-failed", "2",
                "--crawler-total", "11",
            )
            for value in ["success=9", "failed=2", "total=11", "total_sites=11"]:
                self.assertIn(value, body)
            self.assertNotIn("not-exported.invalid", body)

    def test_missing_crawler_totals_are_not_fabricated(self):
        with tempfile.TemporaryDirectory() as tmp:
            body = self.run_closeout(pathlib.Path(tmp))
            self.assertIn("Not supplied.", body)
            self.assertNotIn("`success=0`", body)
            self.assertIn("No same-date wrapper health events", body)

    def test_failed_probe_without_proof_stays_pending(self):
        with tempfile.TemporaryDirectory() as tmp:
            body = self.run_closeout(
                pathlib.Path(tmp), "--simulate-dns-failure", "fixture-only failure"
            )
            self.assertIn("leave the public aggregate section pending", body)
            self.assertNotIn("`total_sites=", body)


if __name__ == "__main__":
    unittest.main()
