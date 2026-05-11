#!/usr/bin/env python3
import importlib.util
import json
import pathlib
import sys
import tempfile
import unittest


MODULE_PATH = pathlib.Path(__file__).with_name("quality-gate-discovery.py")
SPEC = importlib.util.spec_from_file_location("discovery_quality_gate", MODULE_PATH)
gate = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = gate
SPEC.loader.exec_module(gate)


def quarantine_artifact(**overrides):
    data = {
        "source": "harness/discovery-quality-latest.json",
        "artifact_policy": {
            "input": "sanitized aggregate planner artifact only",
            "domain_output": False,
            "crawler_log_access": False,
        },
        "quarantine": {
            "active": True,
            "reason": "low_signal_rows_exceed_hard_signal_rows",
            "low_signal_rows": 8,
            "hard_signal_rows": 3,
            "category_other_low_signal": 7,
            "category_other_hard_agent_signal": 2,
        },
        "public_guards": {
            "public_search": "protected_by_models.AgentFirstFilter",
            "score_fix_targeting": "requires_has_hard_agent_signal",
            "planner_priority": "quarantine_first",
        },
        "sample_breakdown": {
            "hard_agent_signal": 3,
            "llms_only": 2,
            "schema_only": 1,
            "zero_score": 4,
            "passive_or_soft_signal": 1,
            "category_other": 9,
            "category_other_low_signal": 7,
            "category_other_hard_agent_signal": 2,
        },
        "recommended_actions": [
            "Keep rows without API, OpenAPI, MCP, or ai-plugin audit-only.",
        ],
    }
    data.update(overrides)
    return data


def write_repo(
    root,
    agent_filter=None,
    seo_sql="",
    digest_sql="",
    fix_go=None,
    site_template=None,
):
    agent_filter = agent_filter or (
        "crawl_status='success' AND (has_structured_api = true OR "
        "has_openapi = true OR has_ai_plugin = true OR has_mcp_server = true)"
    )
    models_dir = root / "internal" / "models"
    handlers_dir = root / "internal" / "handlers"
    models_dir.mkdir(parents=True)
    handlers_dir.mkdir(parents=True)
    (models_dir / "queries.go").write_text(
        f'package models\n\nconst AgentFirstFilter = "{agent_filter}"\n',
        encoding="utf-8",
    )
    (handlers_dir / "seo.go").write_text(seo_sql, encoding="utf-8")
    (handlers_dir / "digest.go").write_text(digest_sql, encoding="utf-8")
    (handlers_dir / "fix.go").write_text(
        fix_go
        or "package handlers\nfunc scoreFixEligible(site Site) bool { return site.HasHardAgentSignal() }\nfunc checkout() { scoreFixEligible(site) }\n",
        encoding="utf-8",
    )
    templates_dir = root / "templates"
    templates_dir.mkdir(parents=True)
    (templates_dir / "site.html").write_text(
        site_template or "{{if and (lt .AgenticScore 70) (hasHardAgentSignal .)}}",
        encoding="utf-8",
    )


class DiscoveryQualityGateTest(unittest.TestCase):
    def test_gate_accepts_aggregate_artifact_and_hard_signal_filters(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            write_repo(
                root,
                seo_sql="WHERE "+gate.extract_agent_first_filter(pathlib.Path.cwd()),
                digest_sql="WHERE "+gate.extract_agent_first_filter(pathlib.Path.cwd()),
            )
            quarantine = root / "quarantine.json"
            quarantine.write_text(json.dumps(quarantine_artifact()), encoding="utf-8")

            report = gate.run_gate(quarantine, root)

        self.assertTrue(report["quarantine"]["active"])

    def test_gate_rejects_candidate_fields_in_planner_artifact(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            write_repo(root)
            quarantine = root / "quarantine.json"
            quarantine.write_text(
                json.dumps(quarantine_artifact(candidate_domains=["example.com"])),
                encoding="utf-8",
            )

            with self.assertRaisesRegex(gate.DiscoveryQualityGateError, "candidate"):
                gate.run_gate(quarantine, root)

    def test_gate_rejects_passive_signal_in_agent_first_filter(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            write_repo(
                root,
                agent_filter=(
                    "crawl_status='success' AND (has_structured_api = true OR "
                    "has_llms_txt = true OR has_openapi = true OR has_ai_plugin = true "
                    "OR has_mcp_server = true)"
                ),
            )
            quarantine = root / "quarantine.json"
            quarantine.write_text(json.dumps(quarantine_artifact()), encoding="utf-8")

            with self.assertRaisesRegex(gate.DiscoveryQualityGateError, "passive-only"):
                gate.run_gate(quarantine, root)

    def test_gate_rejects_public_sql_that_mixes_llms_into_agent_first_filter(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            write_repo(
                root,
                digest_sql=(
                    "WHERE crawl_status = 'success' AND "
                    "(has_structured_api = true OR has_llms_txt = true OR "
                    "has_openapi = true OR has_ai_plugin = true OR has_mcp_server = true)"
                ),
            )
            quarantine = root / "quarantine.json"
            quarantine.write_text(json.dumps(quarantine_artifact()), encoding="utf-8")

            with self.assertRaisesRegex(gate.DiscoveryQualityGateError, "public discovery SQL"):
                gate.run_gate(quarantine, root)

    def test_gate_rejects_score_fix_checkout_without_hard_signal_guard(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            write_repo(
                root,
                fix_go=(
                    "package handlers\n"
                    "func scoreFixEligible(site Site) bool { return site.AgenticScore < 70 }\n"
                    "func checkout() { scoreFixEligible(site) }\n"
                ),
            )
            quarantine = root / "quarantine.json"
            quarantine.write_text(json.dumps(quarantine_artifact()), encoding="utf-8")

            with self.assertRaisesRegex(gate.DiscoveryQualityGateError, "score-fix checkout"):
                gate.run_gate(quarantine, root)

    def test_gate_rejects_score_fix_cta_without_hard_signal_guard(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            write_repo(root, site_template="{{if lt .AgenticScore 70}}")
            quarantine = root / "quarantine.json"
            quarantine.write_text(json.dumps(quarantine_artifact()), encoding="utf-8")

            with self.assertRaisesRegex(gate.DiscoveryQualityGateError, "score-fix CTA"):
                gate.run_gate(quarantine, root)


if __name__ == "__main__":
    unittest.main()
