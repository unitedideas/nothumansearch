"""Exercise aggregate refresh in temporary files, without discovery or ingestion."""

import json
import os
import pathlib
import shutil
import subprocess
import tempfile
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]


class RefreshDiscoveryQualityTest(unittest.TestCase):
    def run_refresh(self, root, checkout=ROOT):
        env = dict(os.environ)
        for name, file in [
            ("INPUT", "discover.err"),
            ("QUALITY_OUTPUT", "quality.json"),
            ("QUARANTINE_OUTPUT", "quarantine.json"),
            ("HISTORY_OUTPUT", "history.jsonl"),
        ]:
            env["NHS_DISCOVERY_" + name] = str(root / file)
        return subprocess.run(
            ["bash", str(checkout / "tools/refresh-discovery-quality.sh")],
            cwd=root, env=env, capture_output=True, text=True, timeout=15,
        )

    def test_refresh_keeps_domains_out_of_all_artifacts(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            (root / "discover.err").write_text(
                "2099/01/02 12:00:00 api.private.example score=65 cat=developer signals=[OpenAPI, API]\n"
                "2099/01/02 12:00:01 llms.private.example score=25 cat=other signals=[llms.txt]\n"
                "2099/01/02 12:00:02 zero.private.example score=0 cat=other signals=[]\n"
            )
            result = self.run_refresh(root)
            self.assertEqual(result.returncode, 0, result.stderr)
            quality = json.loads((root / "quality.json").read_text())
            quarantine = json.loads((root / "quarantine.json").read_text())
            self.assertEqual(quality["counts"]["sample_rows"], 3)
            self.assertEqual(quality["counts"]["hard_signal_rows"], 1)
            self.assertEqual(quality["counts"]["low_signal_rows"], 2)
            self.assertTrue(quarantine["quarantine"]["active"])
            history = (root / "history.jsonl").read_text().splitlines()
            self.assertEqual(len(history), 1)
            json.loads(history[0])
            for file in ["quality.json", "quarantine.json", "history.jsonl"]:
                self.assertNotIn("private.example", (root / file).read_text())
            self.assertNotIn("private.example", result.stdout)

    def test_missing_input_does_not_create_completion_history(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            result = self.run_refresh(root)
            self.assertNotEqual(result.returncode, 0)
            self.assertFalse((root / "history.jsonl").exists())

    def test_failed_gate_does_not_append_refresh_history(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            checkout = root / "incomplete-checkout"
            (checkout / "tools").mkdir(parents=True)
            for name in [
                "refresh-discovery-quality.sh", "discovery-quality-report.py",
                "discovery-quarantine-report.py", "quality-gate-discovery.py",
            ]:
                shutil.copyfile(ROOT / "tools" / name, checkout / "tools" / name)
            (root / "discover.err").write_text(
                "2099/01/02 12:00:00 api.private.example score=65 cat=developer signals=[OpenAPI, API]\n"
            )
            result = self.run_refresh(root, checkout)
            self.assertNotEqual(result.returncode, 0)
            self.assertTrue((root / "quarantine.json").exists())
            self.assertFalse((root / "history.jsonl").exists())


if __name__ == "__main__":
    unittest.main()
