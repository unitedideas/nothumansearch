#!/usr/bin/env python3
"""Build an aggregate-only quarantine decision from the planner artifact.

This tool intentionally reads the sanitized planner artifact, not crawler logs.
It emits no candidate domains, URLs, or row identifiers.
"""
from __future__ import annotations

import argparse
from datetime import datetime, timezone
import json
from pathlib import Path
from typing import Any


ALLOWED_TOP_LEVEL = {
    "business_local_handoff",
    "cleanup_review_gate",
    "counts",
    "hard_signal_other_review",
    "sample_breakdown",
    "seed_refresh_report",
    "quality_gate",
}
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

    seed_refresh_report = artifact.get("seed_refresh_report", {})
    handoff_required = (
        seed_refresh_report.get("threshold", {}).get("handoff_required") is True
    )

    return {
        "source": "harness/discovery-quality-latest.json",
        "artifact_policy": {
            "input": "sanitized aggregate planner artifact only",
            "domain_output": False,
            "crawler_log_access": False,
        },
        "business_local_handoff": {
            "required": handoff_required,
            "kind": "bounded_aggregate_review" if handoff_required else "none",
            "reason": quality_gate.get("trigger", "none") if handoff_required else "none",
            "scope": (
                "review aggregate seed-refresh cohorts only; do not trigger broad crawl"
            ),
            "public": False,
            "domain_output": False,
        },
        "quarantine": {
            "active": active,
            "reason": quality_gate.get("trigger", "none"),
            "low_signal_rows": low_signal_rows,
            "hard_signal_rows": hard_signal_rows,
            "category_other_low_signal": other_low_signal,
            "category_other_hard_agent_signal": other_hard_signal,
        },
        "cleanup_review_gate": artifact.get("cleanup_review_gate", {}),
        "hard_signal_other_review": artifact.get("hard_signal_other_review", {}),
        "seed_refresh_report": seed_refresh_report,
        "public_guards": {
            "public_search": "protected_by_models.AgentFirstFilter",
            "score_fix_targeting": "requires_has_hard_agent_signal",
            "planner_priority": planner_priority,
        },
        "sample_breakdown": sample_breakdown,
        "recommended_actions": [
            "Keep rows without API, OpenAPI, MCP, or ai-plugin audit-only.",
            "Keep llms-only, schema-only, zero-score, and category=other low-signal cohorts out of score-fix targeting.",
            "Do not promote candidate-row work items from discovery logs.",
            "Refresh only aggregate quarantine counts after weekly discovery.",
        ],
    }


def _utc_now() -> datetime:
    return datetime.now(timezone.utc)


def _parse_observed_at(value: str | None) -> datetime:
    if not value:
        return _utc_now()
    normalized = value
    if normalized.endswith("Z"):
        normalized = normalized[:-1] + "+00:00"
    observed = datetime.fromisoformat(normalized)
    if observed.tzinfo is None:
        observed = observed.replace(tzinfo=timezone.utc)
    return observed.astimezone(timezone.utc)


def build_history_entry(
    report: dict[str, Any],
    observed_at: str | None = None,
) -> dict[str, Any]:
    observed = _parse_observed_at(observed_at)
    week_start = (observed.date()).toordinal() - observed.weekday()
    quarantine = report["quarantine"]
    public_guards = report["public_guards"]

    return {
        "history_key": (
            f"discovery-quarantine:{observed.date().fromordinal(week_start).isoformat()}"
        ),
        "observed_at": observed.isoformat().replace("+00:00", "Z"),
        "week_start": observed.date().fromordinal(week_start).isoformat(),
        "source": report["source"],
        "sample_rows": int(quarantine["hard_signal_rows"])
        + int(quarantine["low_signal_rows"]),
        "hard_signal_rows": int(quarantine["hard_signal_rows"]),
        "low_signal_rows": int(quarantine["low_signal_rows"]),
        "passive_only_share": float(
            report.get("seed_refresh_report", {}).get("passive_only_share", 0)
        ),
        "category_other_low_signal": int(quarantine["category_other_low_signal"]),
        "llms_only": int(report.get("sample_breakdown", {}).get("llms_only", 0)),
        "schema_only": int(report.get("sample_breakdown", {}).get("schema_only", 0)),
        "zero_score": int(report.get("sample_breakdown", {}).get("zero_score", 0)),
        "quarantine": {"active": bool(quarantine["active"])},
        "business_local_handoff": report["business_local_handoff"],
        "planner_priority": public_guards["planner_priority"],
        "planner_scope": "business-local only; never public ranking or score-fix targeting",
    }


def append_history_entry(
    history_path: Path,
    entry: dict[str, Any],
) -> None:
    history_path.parent.mkdir(parents=True, exist_ok=True)
    existing: list[dict[str, Any]] = []
    if history_path.exists():
        for raw_line in history_path.read_text(encoding="utf-8").splitlines():
            if raw_line.strip():
                existing.append(json.loads(raw_line))

    history_key = entry.get("history_key")
    retained = [row for row in existing if row.get("history_key") != history_key]
    retained.append(entry)
    rendered = "\n".join(json.dumps(row, sort_keys=True) for row in retained)
    history_path.write_text(rendered + "\n", encoding="utf-8")


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
    parser.add_argument(
        "--history-output",
        help="optional JSONL path for weekly aggregate quarantine trend history",
    )
    parser.add_argument(
        "--observed-at",
        help="optional ISO-8601 timestamp for deterministic history tests",
    )
    args = parser.parse_args()

    artifact = load_planner_artifact(Path(args.input))
    report = build_quarantine_report(artifact)
    rendered = json.dumps(report, indent=2, sort_keys=True)
    if args.output:
        Path(args.output).write_text(rendered + "\n", encoding="utf-8")
    if args.history_output:
        append_history_entry(
            Path(args.history_output),
            build_history_entry(report, args.observed_at),
        )
    print(rendered)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
