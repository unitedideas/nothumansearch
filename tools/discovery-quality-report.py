#!/usr/bin/env python3
"""Aggregate-only quality report for NHS discovery crawler logs.

The report intentionally emits counts only. Do not print candidate domains
from discovery logs in planner artifacts; those logs can include private or
low-confidence discovery rows that should remain audit-only.
"""
from __future__ import annotations

import argparse
import json
import re
from collections import Counter
from dataclasses import dataclass
from pathlib import Path


SUMMARY_RE = re.compile(
    r"^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}\s+"
    r"(?P<domain>\S+) score=(?P<score>\d+) cat=(?P<category>\S+) "
    r"signals=\[(?P<signals>[^\]]*)\]$"
)

HARD_SIGNALS = {"API", "OpenAPI", "MCP", "ai-plugin"}


@dataclass(frozen=True)
class DiscoveryRow:
    score: int
    category: str
    signals: frozenset[str]

    @property
    def has_hard_signal(self) -> bool:
        return bool(self.signals & HARD_SIGNALS)

    @property
    def primary_bucket(self) -> str:
        if self.has_hard_signal:
            return "hard_agent_signal"
        if self.score == 0:
            return "zero_score"
        if self.signals == frozenset({"llms.txt"}):
            return "llms_only"
        if self.signals == frozenset({"schema.org"}):
            return "schema_only"
        return "passive_or_soft_signal"


def parse_rows(path: Path) -> list[DiscoveryRow]:
    rows: list[DiscoveryRow] = []
    for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
        match = SUMMARY_RE.match(line.strip())
        if not match:
            continue
        signals = frozenset(
            part.strip()
            for part in match.group("signals").split(",")
            if part.strip()
        )
        rows.append(
            DiscoveryRow(
                score=int(match.group("score")),
                category=match.group("category"),
                signals=signals,
            )
        )
    return rows


def _aggregate(
    rows: list[DiscoveryRow],
) -> tuple[Counter[str], Counter[str], Counter[str], Counter[str], Counter[str]]:
    primary = Counter(row.primary_bucket for row in rows)
    category = Counter(row.category for row in rows)
    hard_by_category = Counter(row.category for row in rows if row.has_hard_signal)
    soft_by_category = Counter(row.category for row in rows if not row.has_hard_signal)
    signal_sets = Counter(
        ",".join(sorted(row.signals)) if row.signals else "none"
        for row in rows
    )
    return primary, category, hard_by_category, soft_by_category, signal_sets


def quality_gate(
    total: int,
    hard: int,
    low: int,
    other_hard_signal: int,
    other_low_signal: int,
) -> dict[str, str]:
    if total == 0:
        return {"status": "pass", "trigger": "empty_sample"}
    if low > hard:
        return {
            "status": "review",
            "trigger": "low_signal_rows_exceed_hard_signal_rows",
        }
    if other_low_signal > other_hard_signal:
        return {
            "status": "review",
            "trigger": "category_other_low_signal_exceeds_hard_signal",
        }
    return {"status": "pass", "trigger": "none"}


def build_report(rows: list[DiscoveryRow]) -> dict[str, object]:
    primary, category, hard_by_category, soft_by_category, _ = _aggregate(rows)
    total = len(rows)
    hard = primary["hard_agent_signal"]
    low = total - hard
    other_low_signal = soft_by_category["other"]
    other_hard_signal = hard_by_category["other"]
    return {
        "counts": {
            "sample_rows": total,
            "hard_signal_rows": hard,
            "low_signal_rows": low,
            "hard_signal_rate": round(hard / total, 4) if total else 0,
            "category_other_low_signal": other_low_signal,
            "category_other_hard_agent_signal": other_hard_signal,
        },
        "sample_breakdown": {
            "hard_agent_signal": hard,
            "llms_only": primary["llms_only"],
            "schema_only": primary["schema_only"],
            "zero_score": primary["zero_score"],
            "passive_or_soft_signal": primary["passive_or_soft_signal"],
            "category_other": category["other"],
            "category_other_low_signal": other_low_signal,
            "category_other_hard_agent_signal": other_hard_signal,
        },
        "quality_gate": quality_gate(
            total,
            hard,
            low,
            other_hard_signal,
            other_low_signal,
        ),
    }


def build_legacy_report(rows: list[DiscoveryRow]) -> dict[str, object]:
    primary, category, hard_by_category, soft_by_category, signal_sets = _aggregate(rows)

    total = len(rows)
    hard = primary["hard_agent_signal"]
    low = total - hard
    other = category["other"]
    return {
        "sample_source": "tools/discover.err",
        "sample_rows": total,
        "primary_buckets": dict(sorted(primary.items())),
        "category_other": {
            "total": other,
            "hard_agent_signal": hard_by_category["other"],
            "low_signal": soft_by_category["other"],
        },
        "hard_signal_rows": hard,
        "low_signal_rows": low,
        "hard_signal_rate": round(hard / total, 4) if total else 0,
        "top_categories": dict(category.most_common(8)),
        "top_signal_sets": dict(signal_sets.most_common(8)),
        "quarantine_rule": (
            "Rows without API, OpenAPI, MCP, or ai-plugin are audit-only; "
            "AgentFirstFilter remains the search/public-index contract."
        ),
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--input",
        default="tools/discover.err",
        help="discovery crawler stderr log to summarize",
    )
    parser.add_argument(
        "--output",
        help="optional JSON output path; parent directory must already exist",
    )
    parser.add_argument(
        "--format",
        choices=("planner", "legacy"),
        default="planner",
        help="planner emits the sanitized quarantine-compatible schema",
    )
    args = parser.parse_args()

    rows = parse_rows(Path(args.input))
    report = (
        build_report(rows)
        if args.format == "planner"
        else build_legacy_report(rows)
    )
    rendered = json.dumps(report, indent=2, sort_keys=True)
    if args.output:
        Path(args.output).write_text(rendered + "\n", encoding="utf-8")
    print(rendered)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
