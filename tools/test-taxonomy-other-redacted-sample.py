#!/usr/bin/env python3
import importlib.util
import json
import pathlib
import sys
import unittest
from unittest import mock


MODULE_PATH = pathlib.Path(__file__).with_name("taxonomy-other-redacted-sample.py")
SPEC = importlib.util.spec_from_file_location("taxonomy_other_redacted_sample", MODULE_PATH)
sampler = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = sampler
SPEC.loader.exec_module(sampler)


class TaxonomyOtherRedactedSampleTest(unittest.TestCase):
    def test_builds_aggregate_without_raw_fields(self):
        rows = [
            {
                "domain": "developer.example",
                "url": "https://developer.example",
                "name": "Developer API",
                "description": "OpenAPI docs and webhook SDK for deploy automation",
                "has_openapi": True,
            },
            {
                "domain": "passive.example",
                "name": "Passive Docs",
                "description": "llms.txt only documentation",
                "has_llms_txt": True,
            },
            {
                "domain": "finance.example",
                "name": "Invoice Wallet",
                "description": "Payment invoice wallet API",
                "has_structured_api": True,
            },
        ]

        report = sampler.build_aggregate(rows)
        rendered = json.dumps(report, sort_keys=True)

        self.assertEqual(report["sample"]["rows_seen"], 3)
        self.assertEqual(report["sample"]["hard_signal_rows"], 2)
        self.assertEqual(report["pattern_counts"]["developer_api_infra"], 2)
        self.assertEqual(report["pattern_counts"]["finance_or_crypto"], 1)
        self.assertFalse(report["artifact_policy"]["domain_output"])
        self.assertFalse(report["artifact_policy"]["name_description_output"])
        self.assertNotIn("developer.example", rendered)
        self.assertNotIn("OpenAPI docs", rendered)

    def test_keychain_secret_uses_foundry_account_without_printing_secret(self):
        proc = mock.Mock(returncode=0, stdout="secret-value\n", stderr="")
        with mock.patch.object(sampler.subprocess, "run", return_value=proc) as run:
            secret = sampler._keychain_secret("nhs-admin-api-key")

        self.assertEqual(secret, "secret-value")
        args = run.call_args.args[0]
        self.assertEqual(args[:4], ["/usr/bin/security", "find-generic-password", "-a", "foundry"])
        self.assertIn("nhs-admin-api-key", args)
        self.assertNotIn("secret-value", args)

    def test_authenticated_loader_sets_bearer_header(self):
        captured = {}

        class FakeResponse:
            def __enter__(self):
                return self

            def __exit__(self, exc_type, exc, tb):
                return False

        def fake_urlopen(req, timeout):
            captured["authorization"] = req.headers.get("Authorization")
            captured["user_agent"] = req.headers.get("User-agent")
            return FakeResponse()

        with mock.patch.object(sampler.urllib.request, "urlopen", side_effect=fake_urlopen), mock.patch.object(
            sampler.json, "load", return_value={"results": []}
        ):
            payload = sampler._load_json_from_url("https://example.test/api", 5, "secret")

        self.assertEqual(payload, {"results": []})
        self.assertEqual(captured["authorization"], "Bearer secret")
        self.assertEqual(captured["user_agent"], "curl/8.7.1")


if __name__ == "__main__":
    unittest.main()
