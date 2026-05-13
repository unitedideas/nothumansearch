#!/usr/bin/env python3
"""Executable aggregate-only gate for NHS discovery quarantine.

This gate intentionally reads only sanitized aggregate artifacts plus local
source files. It never reads crawler logs and never emits candidate domains.
"""
from __future__ import annotations

import argparse
import json
import re
from pathlib import Path
from typing import Any


EXPECTED_TOP_LEVEL = {
    "artifact_policy",
    "business_local_handoff",
    "cleanup_review_gate",
    "hard_signal_other_review",
    "public_guards",
    "quarantine",
    "recommended_actions",
    "sample_breakdown",
    "seed_refresh_report",
    "source",
}
EXPECTED_PUBLIC_GUARDS = {
    "public_search": "protected_by_models.AgentFirstFilter",
    "score_fix_targeting": "requires_has_hard_agent_signal",
}
EXPECTED_COHORTS = {
    "category_other_low_signal": "aggregate_review_only",
    "llms_only": "audit_only",
    "schema_only": "audit_only",
    "zero_score": "audit_only",
}
DISALLOWED_AGGREGATE_FIELDS = {
    "candidate_domains",
    "candidate_urls",
    "domains",
    "raw_rows",
    "row_ids",
    "urls",
}
REQUIRED_HARD_COLUMNS = {
    "has_structured_api",
    "has_openapi",
    "has_ai_plugin",
    "has_mcp_server",
}
PASSIVE_COLUMNS = {
    "has_llms_txt",
    "has_schema_org",
    "has_robots_ai",
}
PASSIVE_OR_RE = re.compile(
    r"has_(?:structured_api|openapi|ai_plugin|mcp_server)\s*=\s*true\s+OR\s+"
    r"has_(?:llms_txt|schema_org|robots_ai)\s*=\s*true|"
    r"has_(?:llms_txt|schema_org|robots_ai)\s*=\s*true\s+OR\s+"
    r"has_(?:structured_api|openapi|ai_plugin|mcp_server)\s*=\s*true",
    re.DOTALL,
)


class DiscoveryQualityGateError(ValueError):
    pass


def _walk_keys(value: Any) -> list[str]:
    keys: list[str] = []
    if isinstance(value, dict):
        for key, child in value.items():
            keys.append(str(key))
            keys.extend(_walk_keys(child))
    elif isinstance(value, list):
        for child in value:
            keys.extend(_walk_keys(child))
    return keys


def load_quarantine(path: Path) -> dict[str, Any]:
    data = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        raise DiscoveryQualityGateError("quarantine artifact must be a JSON object")
    extra = set(data) - EXPECTED_TOP_LEVEL
    if extra:
        raise DiscoveryQualityGateError(
            "quarantine artifact contains non-aggregate fields: "
            + ", ".join(sorted(extra))
        )
    disallowed = DISALLOWED_AGGREGATE_FIELDS & set(_walk_keys(data))
    if disallowed:
        raise DiscoveryQualityGateError(
            "quarantine artifact includes candidate fields: "
            + ", ".join(sorted(disallowed))
        )
    policy = data.get("artifact_policy")
    if not isinstance(policy, dict):
        raise DiscoveryQualityGateError("artifact_policy must be an object")
    if policy.get("domain_output") is not False:
        raise DiscoveryQualityGateError("artifact_policy.domain_output must be false")
    if policy.get("crawler_log_access") is not False:
        raise DiscoveryQualityGateError("artifact_policy.crawler_log_access must be false")

    public_guards = data.get("public_guards")
    if not isinstance(public_guards, dict):
        raise DiscoveryQualityGateError("public_guards must be an object")
    for key, expected in EXPECTED_PUBLIC_GUARDS.items():
        if public_guards.get(key) != expected:
            raise DiscoveryQualityGateError(
                f"public_guards.{key} must be {expected!r}"
            )
    cleanup_gate = data.get("cleanup_review_gate")
    if not isinstance(cleanup_gate, dict):
        raise DiscoveryQualityGateError("cleanup_review_gate must be an object")
    cohorts = cleanup_gate.get("cohorts")
    if not isinstance(cohorts, dict):
        raise DiscoveryQualityGateError("cleanup_review_gate.cohorts must be an object")
    for cohort, expected_decision in EXPECTED_COHORTS.items():
        row = cohorts.get(cohort)
        if not isinstance(row, dict):
            raise DiscoveryQualityGateError(f"cleanup_review_gate missing {cohort}")
        if row.get("decision") != expected_decision:
            raise DiscoveryQualityGateError(
                f"cleanup_review_gate.{cohort}.decision must be {expected_decision!r}"
            )
        if row.get("public_search") is not False:
            raise DiscoveryQualityGateError(
                f"cleanup_review_gate.{cohort}.public_search must be false"
            )
        if row.get("score_fix_targeting") is not False:
            raise DiscoveryQualityGateError(
                f"cleanup_review_gate.{cohort}.score_fix_targeting must be false"
            )
    seed_report = data.get("seed_refresh_report")
    if not isinstance(seed_report, dict):
        raise DiscoveryQualityGateError("seed_refresh_report must be an object")
    if seed_report.get("threshold", {}).get("status") == "review":
        if seed_report.get("bounded_action") != (
            "write business-local aggregate handoff row; do not trigger broad crawl"
        ):
            raise DiscoveryQualityGateError(
                "seed_refresh_report review threshold must require bounded handoff"
            )
        handoff = data.get("business_local_handoff")
        if not isinstance(handoff, dict):
            raise DiscoveryQualityGateError("business_local_handoff must be an object")
        if handoff.get("required") is not True:
            raise DiscoveryQualityGateError(
                "review threshold must require business-local handoff"
            )
        if handoff.get("kind") != "bounded_aggregate_review":
            raise DiscoveryQualityGateError(
                "business-local handoff must stay bounded to aggregate review"
            )
        if handoff.get("public") is not False or handoff.get("domain_output") is not False:
            raise DiscoveryQualityGateError(
                "business-local handoff must not be public or domain-level"
            )
    return data


def extract_agent_first_filter(repo_root: Path) -> str:
    text = (repo_root / "internal/models/queries.go").read_text(encoding="utf-8")
    match = re.search(r'const\s+AgentFirstFilter\s*=\s*"([^"]+)"', text)
    if not match:
        raise DiscoveryQualityGateError("models.AgentFirstFilter const not found")
    return match.group(1)


def validate_agent_first_filter(filter_sql: str) -> None:
    missing = [column for column in sorted(REQUIRED_HARD_COLUMNS) if column not in filter_sql]
    if missing:
        raise DiscoveryQualityGateError(
            "AgentFirstFilter missing hard-signal columns: " + ", ".join(missing)
        )
    passive = [column for column in sorted(PASSIVE_COLUMNS) if column in filter_sql]
    if passive:
        raise DiscoveryQualityGateError(
            "AgentFirstFilter includes passive-only columns: " + ", ".join(passive)
        )
    if "crawl_status='success'" not in filter_sql:
        raise DiscoveryQualityGateError("AgentFirstFilter must require successful crawl")


def validate_public_handler_filters(repo_root: Path) -> None:
    public_files = [
        repo_root / "internal/handlers/seo.go",
        repo_root / "internal/handlers/digest.go",
    ]
    offenders: list[str] = []
    for path in public_files:
        text = path.read_text(encoding="utf-8")
        if PASSIVE_OR_RE.search(text):
            offenders.append(str(path.relative_to(repo_root)))
    if offenders:
        raise DiscoveryQualityGateError(
            "public discovery SQL mixes passive signals into agent-first filters: "
            + ", ".join(offenders)
        )


def validate_score_fix_targeting(repo_root: Path) -> None:
    fix_text = (repo_root / "internal/handlers/fix.go").read_text(encoding="utf-8")
    web_text = (repo_root / "templates/site.html").read_text(encoding="utf-8")
    if "site.HasHardAgentSignal()" not in fix_text:
        raise DiscoveryQualityGateError(
            "score-fix checkout must require models.Site.HasHardAgentSignal"
        )
    if "scoreFixEligible(site)" not in fix_text:
        raise DiscoveryQualityGateError(
            "score-fix checkout paths must call scoreFixEligible"
        )
    if "(hasHardAgentSignal .)" not in web_text:
        raise DiscoveryQualityGateError(
            "score-fix CTA must require hasHardAgentSignal in site template"
        )


def run_gate(quarantine_path: Path, repo_root: Path) -> dict[str, Any]:
    report = load_quarantine(quarantine_path)
    validate_agent_first_filter(extract_agent_first_filter(repo_root))
    validate_public_handler_filters(repo_root)
    validate_score_fix_targeting(repo_root)
    return report


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--quarantine",
        default="harness/discovery-quarantine-latest.json",
        help="aggregate-only quarantine report",
    )
    parser.add_argument(
        "--repo-root",
        default=".",
        help="repository root for source guard checks",
    )
    args = parser.parse_args()

    report = run_gate(Path(args.quarantine), Path(args.repo_root))
    quarantine = report["quarantine"]
    guards = report["public_guards"]
    print(
        "discovery_quality_gate "
        f"quarantine_active={str(quarantine['active']).lower()} "
        f"hard_signal_rows={quarantine['hard_signal_rows']} "
        f"low_signal_rows={quarantine['low_signal_rows']} "
        f"category_other_low_signal={quarantine['category_other_low_signal']} "
        f"planner_priority={guards['planner_priority']} "
        "public_filters=agent_first"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
