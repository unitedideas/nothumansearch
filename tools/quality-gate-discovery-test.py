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
        "business_local_handoff": {
            "required": True,
            "kind": "bounded_aggregate_review",
            "reason": "low_signal_rows_exceed_hard_signal_rows",
            "scope": "review aggregate seed-refresh cohorts only; do not trigger broad crawl",
            "public": False,
            "domain_output": False,
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
        "cleanup_review_gate": {
            "status": "active",
            "cohorts": {
                "category_other_low_signal": {
                    "decision": "aggregate_review_only",
                    "public_search": False,
                    "rows": 7,
                    "score_fix_targeting": False,
                },
                "llms_only": {
                    "decision": "audit_only",
                    "public_search": False,
                    "rows": 2,
                    "score_fix_targeting": False,
                },
                "schema_only": {
                    "decision": "audit_only",
                    "public_search": False,
                    "rows": 1,
                    "score_fix_targeting": False,
                },
                "zero_score": {
                    "decision": "audit_only",
                    "public_search": False,
                    "rows": 4,
                    "score_fix_targeting": False,
                },
            },
            "public_search_effect": "none; AgentFirstFilter remains required",
            "score_fix_effect": "none; HasHardAgentSignal remains required",
        },
        "hard_signal_other_review": {
            "review_policy": "aggregate-only; executor samples must not enter planner artifacts",
            "rows": 2,
            "score_buckets": {"0_24": 0, "25_39": 1, "40_59": 1, "60_plus": 0},
            "top_signal_sets": {"API": 2},
        },
        "seed_refresh_report": {
            "bounded_action": "write business-local aggregate handoff row; do not trigger broad crawl",
            "cohorts": {
                "category_other_low_signal": 7,
                "llms_only": 2,
                "schema_only": 1,
                "zero_score": 4,
            },
            "hard_signal_rows": 3,
            "passive_only_rows": 8,
            "passive_only_share": 0.7273,
            "sample_rows": 11,
            "source": "tools/seed-refresh.log",
            "threshold": {
                "handoff_required": True,
                "name": "low_signal_rows_exceed_hard_signal_rows",
                "status": "review",
            },
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

    def test_gate_rejects_cleanup_cohort_that_targets_score_fix(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            write_repo(root)
            artifact = quarantine_artifact()
            artifact["cleanup_review_gate"]["cohorts"]["llms_only"][
                "score_fix_targeting"
            ] = True
            quarantine = root / "quarantine.json"
            quarantine.write_text(json.dumps(artifact), encoding="utf-8")

            with self.assertRaisesRegex(gate.DiscoveryQualityGateError, "score_fix_targeting"):
                gate.run_gate(quarantine, root)

    def test_gate_rejects_review_threshold_without_bounded_handoff(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            write_repo(root)
            artifact = quarantine_artifact()
            artifact["seed_refresh_report"]["bounded_action"] = "run broad crawl"
            quarantine = root / "quarantine.json"
            quarantine.write_text(json.dumps(artifact), encoding="utf-8")

            with self.assertRaisesRegex(gate.DiscoveryQualityGateError, "bounded handoff"):
                gate.run_gate(quarantine, root)

    def test_gate_rejects_review_threshold_without_local_handoff(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            write_repo(root)
            artifact = quarantine_artifact()
            artifact["business_local_handoff"] = {
                "required": False,
                "kind": "none",
                "public": False,
                "domain_output": False,
            }
            quarantine = root / "quarantine.json"
            quarantine.write_text(json.dumps(artifact), encoding="utf-8")

            with self.assertRaisesRegex(gate.DiscoveryQualityGateError, "business-local handoff"):
                gate.run_gate(quarantine, root)


if __name__ == "__main__":
    unittest.main()
