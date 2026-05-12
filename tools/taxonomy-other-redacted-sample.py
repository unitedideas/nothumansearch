#!/usr/bin/env python3
"""Inspect category=other hard-signal rows and emit aggregate taxonomy hints only.

The script may read row-level names/descriptions from the NHS API in memory, but
it never prints domains, URLs, ids, names, or descriptions. This keeps planner
artifacts aggregate-only while still letting an executor decide whether a
category rule is defensible.
"""
from __future__ import annotations

import argparse
import collections
import json
import re
import subprocess
import sys
import urllib.parse
import urllib.request
from typing import Any


BUCKETS: tuple[tuple[str, str], ...] = (
    (
        "developer_api_infra",
        r"\b(api|developer|code|deploy|monitor|mcp|server|sdk|webhook|database|docs|tool|openapi)\b",
    ),
    (
        "ai_agent_tooling",
        r"\b(ai|agent|llm|prompt|model|gpt|automation|inference|embedding)\b",
    ),
    (
        "data_or_research",
        r"\b(data|dataset|search|analytics|crawler|scrap|index|knowledge|research)\b",
    ),
    (
        "communication_or_social",
        r"\b(email|message|sms|chat|social|notification|inbox|voice|call)\b",
    ),
    (
        "commerce_or_booking",
        r"\b(shop|store|catalog|booking|checkout|order|hotel|travel|rental|property|product)\b",
    ),
    (
        "finance_or_crypto",
        r"\b(finance|financial|payment|invoice|tax|stock|crypto|bitcoin|defi|trading|wallet)\b",
    ),
    (
        "health_or_wellness",
        r"\b(health|medical|doctor|patient|clinic|fitness|wellness|pharmacy)\b",
    ),
    (
        "education_or_content",
        r"\b(course|learn|education|student|school|lesson|library|book|article|journal)\b",
    ),
    (
        "news_or_media",
        r"\b(news|media|press|publication|magazine)\b",
    ),
)


def _load_json_from_url(
    url: str,
    timeout: int,
    bearer_token: str | None = None,
) -> dict[str, Any]:
    headers = {"User-Agent": "curl/8.7.1"}
    if bearer_token:
        headers["Authorization"] = f"Bearer {bearer_token}"
    req = urllib.request.Request(url, headers=headers)
    with urllib.request.urlopen(req, timeout=timeout) as response:
        payload = json.load(response)
    if not isinstance(payload, dict):
        raise ValueError("API response must be a JSON object")
    return payload


def _keychain_secret(service: str) -> str:
    proc = subprocess.run(
        [
            "/usr/bin/security",
            "find-generic-password",
            "-a",
            "foundry",
            "-s",
            service,
            "-w",
        ],
        check=False,
        capture_output=True,
        text=True,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"keychain service unavailable: {service}")
    secret = proc.stdout.strip()
    if not secret:
        raise RuntimeError(f"keychain service empty: {service}")
    return secret


def _rows(payload: dict[str, Any]) -> list[dict[str, Any]]:
    raw = payload.get("results") or payload.get("sites") or []
    if not isinstance(raw, list):
        return []
    return [row for row in raw if isinstance(row, dict)]


def _has_hard_signal(row: dict[str, Any]) -> bool:
    return any(
        bool(row.get(key))
        for key in ("has_structured_api", "has_openapi", "has_mcp", "has_ai_plugin")
    )


def build_aggregate(rows: list[dict[str, Any]]) -> dict[str, Any]:
    patterns: collections.Counter[str] = collections.Counter()
    hard_signal = 0
    usable_name_desc = 0

    for row in rows:
        if not _has_hard_signal(row):
            continue
        hard_signal += 1
        name = str(row.get("name") or "").strip().lower()
        desc = str(row.get("description") or "").strip().lower()
        if not name or not desc:
            continue
        usable_name_desc += 1
        text = f"{name} {desc}"
        hit = False
        for bucket, pattern in BUCKETS:
            if re.search(pattern, text):
                patterns[bucket] += 1
                hit = True
        if not hit:
            patterns["no_generalized_pattern"] += 1

    return {
        "artifact_policy": {
            "raw_fields_output": False,
            "domain_output": False,
            "url_output": False,
            "row_id_output": False,
            "name_description_output": False,
        },
        "sample": {
            "rows_seen": len(rows),
            "hard_signal_rows": hard_signal,
            "usable_name_description_rows": usable_name_desc,
        },
        "pattern_counts": dict(sorted(patterns.items())),
        "decision_hint": (
            "Add taxonomy rules only when the aggregate pattern is backed by "
            "a private executor review and a crawler unit test."
        ),
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base-url", default="https://nothumansearch.ai")
    parser.add_argument("--limit", type=int, default=50)
    parser.add_argument("--timeout", type=int, default=20)
    parser.add_argument(
        "--keychain-service",
        default="",
        help="optional foundry Keychain service used as Bearer auth token",
    )
    args = parser.parse_args()

    if args.limit < 1 or args.limit > 200:
        raise SystemExit("--limit must be between 1 and 200")

    query = urllib.parse.urlencode({"category": "other", "limit": args.limit})
    url = f"{args.base_url.rstrip('/')}/api/v1/search?{query}"
    try:
        bearer_token = (
            _keychain_secret(args.keychain_service) if args.keychain_service else None
        )
        payload = _load_json_from_url(url, args.timeout, bearer_token)
    except Exception as exc:
        print(
            json.dumps(
                {
                    "sample_status": "failed",
                    "reason": type(exc).__name__,
                    "artifact_policy": {
                        "raw_fields_output": False,
                        "domain_output": False,
                        "url_output": False,
                        "row_id_output": False,
                        "name_description_output": False,
                    },
                },
                sort_keys=True,
            )
        )
        return 1

    report = build_aggregate(_rows(payload))
    report["sample_status"] = "ok"
    print(json.dumps(report, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    sys.exit(main())
