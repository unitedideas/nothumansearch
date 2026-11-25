"""Exercise the scheduled readiness probe without production ingestion."""

import json
import re
import subprocess
import textwrap
import threading
import unittest
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path


WORKFLOW = Path(__file__).resolve().parents[1] / ".github/workflows/agentic-readiness.yml"
ENDPOINT = "https://nothumansearch.ai/api/v1/site/nothumansearch.ai"


class ReadinessWorkflowTests(unittest.TestCase):
    def test_cached_probe_fails_closed_without_writes(self):
        workflow = WORKFLOW.read_text()
        match = re.search(
            r"      - name: Check cached readiness without ingestion\n"
            r"        shell: bash\n        run: \|\n((?:          .*\n)+)",
            workflow,
        )
        self.assertIsNotNone(match)
        script = textwrap.dedent(match.group(1))
        self.assertEqual(script.count(ENDPOINT), 1)
        cases = [
            (200, {"agentic_score": 100}, True),
            (200, {"agentic_score": 95}, True),
            (200, {"agentic_score": 94}, False),
            (200, {}, False),
            (200, {"agentic_score": None}, False),
            (200, {"agentic_score": "100"}, False),
            (200, {"agentic_score": -1}, False),
            (200, {"agentic_score": 101}, False),
            (200, b"invalid JSON", False),
            (500, {"agentic_score": 100}, False),
        ]
        for status, body, succeeds in cases:
            with self.subTest(status=status, body=body):
                requests = []

                class Handler(BaseHTTPRequestHandler):
                    def do_GET(self):
                        requests.append(("GET", self.path))
                        self.send_response(status)
                        self.end_headers()
                        self.wfile.write(body if isinstance(body, bytes) else json.dumps(body).encode())

                    def do_POST(self):
                        requests.append(("POST", self.path))
                        self.send_response(405)
                        self.end_headers()

                    def log_message(self, *args):
                        pass

                with HTTPServer(("127.0.0.1", 0), Handler) as server:
                    thread = threading.Thread(target=server.serve_forever, daemon=True)
                    thread.start()
                    url = f"http://127.0.0.1:{server.server_port}/api/v1/site/nothumansearch.ai"
                    try:
                        result = subprocess.run(
                            ["bash", "-c", script.replace(ENDPOINT, url)],
                            capture_output=True, text=True, timeout=5,
                        )
                    finally:
                        server.shutdown()
                        thread.join(timeout=2)
                    self.assertEqual(result.returncode == 0, succeeds, result.stderr)
                    self.assertEqual(requests, [("GET", "/api/v1/site/nothumansearch.ai")])

    def test_external_actions_are_immutable(self):
        refs = re.findall(r"^\s+- uses: (\S+)", WORKFLOW.read_text(), re.MULTILINE)
        self.assertTrue(refs)
        for ref in refs:
            self.assertRegex(ref, r"^[\w.-]+/[\w./-]+@[0-9a-f]{40}$")


if __name__ == "__main__":
    unittest.main()
