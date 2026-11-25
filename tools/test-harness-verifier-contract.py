#!/usr/bin/env python3
import pathlib
import re
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]
VERIFIER = "tools/verify-harness-local.sh"


class HarnessVerifierContractTest(unittest.TestCase):
    def test_harness_readme_documents_local_verifier(self):
        readme = (ROOT / "harness" / "README.md").read_text(encoding="utf-8")

        self.assertIn(VERIFIER, readme)
        self.assertIn("Local harness verification", readme)

    def test_agentic_readiness_workflow_runs_local_verifier(self):
        workflow = (
            ROOT / ".github" / "workflows" / "agentic-readiness.yml"
        ).read_text(encoding="utf-8")

        match = re.search(
            r"discovery-quality-gate:\n(?P<body>.*?)(?:\n  [a-zA-Z0-9_-]+:|\Z)",
            workflow,
            flags=re.DOTALL,
        )
        self.assertIsNotNone(match, "discovery-quality-gate job is missing")
        self.assertIn(VERIFIER, match.group("body"))


if __name__ == "__main__":
    unittest.main()
