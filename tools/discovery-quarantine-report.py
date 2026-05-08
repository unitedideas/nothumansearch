#!/usr/bin/env python3
"""Build an aggregate-only quarantine decision from the planner artifact.

This tool intentionally reads the sanitized planner artifact, not crawler logs.
It emits no candidate domains, URLs, or row identifiers.
"""
from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any


ALLOWED_TOP_LEVEL = {"counts", "sample_breakdown", "quality_gate"}
ALLOWED_QUALITY_GATE = {"status", "trigger"}
REQUIRED_COUNTS = {
    "sample_rows",
    "hard_signal_rows",
    "low_signal_rows",
    "hard_signal_rate",
    "category_other_low_signal",
    "category_other_hard_agent_signal",
}


class SanitizedArtifactError(ValueError):
    pass


def load_planner_artifact(path: Path) -> dict[str, Any]:
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise SanitizedArtifactError(f"invalid JSON: {exc}") from exc
    if not isinstance(data, dict):
        raise SanitizedArtifactError("planner artifact must be a JSON object")

    extra_keys = set(data) - ALLOWED_TOP_LEVEL
    if extra_keys:
        raise SanitizedArtifactError(
            "planner artifact is not sanitized aggregate-only: "
            + ", ".join(sorted(extra_keys))
        )

    counts = data.get("counts")
    quality_gate = data.get("quality_gate")
    sample_breakdown = data.get("sample_breakdown")
    if not isinstance(counts, dict):
        raise SanitizedArtifactError("counts must be an object")
    if not isinstance(quality_gate, dict):
        raise SanitizedArtifactError("quality_gate must be an object")
    if not isinstance(sample_breakdown, dict):
        raise SanitizedArtifactError("sample_breakdown must be an object")

    missing_counts = REQUIRED_COUNTS - set(counts)
    if missing_counts:
        raise SanitizedArtifactError(
            "counts missing required aggregate fields: "
            + ", ".join(sorted(missing_counts))
        )
    extra_quality_keys = set(quality_gate) - ALLOWED_QUALITY_GATE
    if extra_quality_keys:
        raise SanitizedArtifactError(
            "quality_gate contains non-planner fields: "
            + ", ".join(sorted(extra_quality_keys))
        )

    for key in REQUIRED_COUNTS:
        if not isinstance(counts[key], (int, float)):
            raise SanitizedArtifactError(f"counts.{key} must be numeric")
    for key in ALLOWED_QUALITY_GATE:
        if key in quality_gate and not isinstance(quality_gate[key], str):
            raise SanitizedArtifactError(f"quality_gate.{key} must be a string")

    return data


def build_quarantine_report(artifact: dict[str, Any]) -> dict[str, Any]:
    counts = artifact["counts"]
    sample_breakdown = artifact["sample_breakdown"]
    quality_gate = artifact["quality_gate"]

    low_signal_rows = int(counts["low_signal_rows"])
    hard_signal_rows = int(counts["hard_signal_rows"])
    other_low_signal = int(counts["category_other_low_signal"])
    other_hard_signal = int(counts["category_other_hard_agent_signal"])
    active = (
        low_signal_rows > 0
        or other_low_signal > 0
        or quality_gate.get("status") == "review"
    )

    planner_priority = "aggregate_review"
    if not active:
        planner_priority = "normal"
    elif low_signal_rows > hard_signal_rows or other_low_signal > other_hard_signal:
        planner_priority = "quarantine_first"

    return {
        "source": "harness/discovery-quality-latest.json",
        "artifact_policy": {
            "input": "sanitized aggregate planner artifact only",
            "domain_output": False,
            "crawler_log_access": False,
        },
        "quarantine": {
            "active": active,
            "reason": quality_gate.get("trigger", "none"),
            "low_signal_rows": low_signal_rows,
            "hard_signal_rows": hard_signal_rows,
            "category_other_low_signal": other_low_signal,
            "category_other_hard_agent_signal": other_hard_signal,
        },
        "public_guards": {
            "public_search": "protected_by_models.AgentFirstFilter",
            "score_fix_targeting": "requires_has_hard_agent_signal",
            "planner_priority": planner_priority,
        },
        "sample_breakdown": sample_breakdown,
        "recommended_actions": [
            "Keep rows without API, OpenAPI, MCP, or ai-plugin audit-only.",
            "Do not promote candidate-row work items from discovery logs.",
            "Refresh only aggregate quarantine counts after weekly discovery.",
        ],
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--input",
        default="harness/discovery-quality-latest.json",
        help="sanitized aggregate planner artifact",
    )
    parser.add_argument(
        "--output",
        default="harness/discovery-quarantine-latest.json",
        help="aggregate-only quarantine report path",
    )
    args = parser.parse_args()

    artifact = load_planner_artifact(Path(args.input))
    report = build_quarantine_report(artifact)
    rendered = json.dumps(report, indent=2, sort_keys=True)
    if args.output:
        Path(args.output).write_text(rendered + "\n", encoding="utf-8")
    print(rendered)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
