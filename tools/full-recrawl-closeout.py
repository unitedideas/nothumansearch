#!/usr/bin/env python3
"""Write aggregate-only NHS full-recrawl closeout notes.

This helper deliberately avoids crawl row/domain inspection. Crawler totals
must be supplied by the caller from an already-summarized wrapper/planner line.
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import sys
import urllib.error
import urllib.request
from pathlib import Path


APP_DIR = Path(__file__).resolve().parents[1]
DEFAULT_STATS_URL = "https://nothumansearch.ai/api/v1/stats"
DEFAULT_CATEGORIES_URL = "https://nothumansearch.ai/api/v1/categories"


class ProbeError(Exception):
    pass


def load_json_fixture(path: str | None) -> object | None:
    if not path:
        return None
    with open(path, "r", encoding="utf-8") as handle:
        return json.load(handle)


def fetch_json(url: str, timeout: int = 20) -> object:
    request = urllib.request.Request(
        url,
        headers={
            "Accept": "application/json",
            "User-Agent": "nothumansearch-closeout/1.0",
        },
    )
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            status = getattr(response, "status", 0)
            body = response.read()
    except urllib.error.URLError as exc:
        raise ProbeError(str(exc.reason)) from exc
    except TimeoutError as exc:
        raise ProbeError("timeout") from exc

    if status != 200:
        raise ProbeError(f"http_status={status}")
    try:
        return json.loads(body.decode("utf-8"))
    except json.JSONDecodeError as exc:
        raise ProbeError("invalid_json") from exc


def normalize_stats(data: object) -> dict[str, object]:
    if not isinstance(data, dict):
        raise ValueError("stats JSON must be an object")
    required = ("total_sites", "avg_score", "top_category")
    missing = [key for key in required if key not in data]
    if missing:
        raise ValueError(f"stats JSON missing keys: {', '.join(missing)}")
    return {
        "total_sites": data["total_sites"],
        "avg_score": data["avg_score"],
        "top_category": data["top_category"],
    }


def normalize_categories(data: object) -> list[tuple[str, object]]:
    if isinstance(data, dict) and isinstance(data.get("categories"), list):
        rows = data["categories"]
    elif isinstance(data, list):
        rows = data
    else:
        raise ValueError("categories JSON must be a list or object with categories list")

    categories: list[tuple[str, object]] = []
    for row in rows:
        if not isinstance(row, dict):
            raise ValueError("category row must be an object")
        name = row.get("category") or row.get("name")
        count = row.get("count")
        if name is None or count is None:
            raise ValueError("category row missing category/name or count")
        categories.append((str(name), count))
    return categories


def latest_health_events(health_log: Path, date_prefix: str) -> list[str]:
    if not health_log.exists():
        return []
    matches: list[str] = []
    with open(health_log, "r", encoding="utf-8", errors="replace") as handle:
        for line in handle:
            line = line.strip()
            if (
                line.startswith(date_prefix)
                and "wrapper=full-recrawl" in line
                and ("event=start" in line or "event=completion" in line or "phase=post_full_recrawl" in line)
            ):
                matches.append(line)
    return matches[-6:]


def render_closeout(args: argparse.Namespace, stats: dict[str, object] | None, categories: list[tuple[str, object]] | None, probe_error: str | None) -> str:
    now = dt.datetime.now(dt.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")
    date_title = args.date
    lines = [
        f"# NHS Full Recrawl Closeout - {date_title}",
        "",
        f"Generated: `{now}`",
        "",
        "Scope: aggregate-only full-recrawl closeout. This helper does not read raw crawl rows, raw domains, private identifiers, process environments, or secrets.",
        "",
        "Wrapper evidence:",
    ]

    health_events = latest_health_events(Path(args.health_log), args.date)
    if health_events:
        lines.extend(f"- `{event}`" for event in health_events)
    else:
        lines.append("- No same-date wrapper health events found in the configured health log.")

    lines.extend(["", "Crawler aggregate:"])
    if args.crawler_success is not None and args.crawler_failed is not None and args.crawler_total is not None:
        lines.extend([
            f"- `success={args.crawler_success}`",
            f"- `failed={args.crawler_failed}`",
            f"- `total={args.crawler_total}`",
        ])
    else:
        lines.append("- Not supplied. Caller should attach only an already-summarized `Success/Failed/Total` line if needed.")

    lines.extend(["", "Public aggregate probe:"])
    if stats is not None and categories is not None:
        lines.extend([
            f"- `total_sites={stats['total_sites']}`",
            f"- `avg_score={stats['avg_score']}`",
            f"- `top_category={stats['top_category']}`",
            "- Categories: " + ", ".join(f"{name}={count}" for name, count in categories),
        ])
    else:
        lines.append(f"- DNS/public probe failed: `{probe_error or 'unknown'}`")
        if args.same_day_public_proof:
            lines.append(f"- Same-day aggregate public proof attached: {args.same_day_public_proof}")
            lines.append("- Decision: DNS failure does not block wrapper-completion closeout when same-day aggregate public proof is attached.")
        else:
            lines.append("- Decision: leave the public aggregate section pending until a same-day aggregate public proof is attached.")

    lines.extend([
        "",
        "Decision:",
        "- Close only on wrapper completion evidence plus either live public aggregates or attached same-day aggregate public proof.",
        "- Do not reopen this boundary to fetch raw crawl rows/domains. Use public aggregate probes or targeted future work.",
        "",
    ])
    return "\n".join(lines)


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--date", default=dt.date.today().isoformat())
    parser.add_argument("--output", default="")
    parser.add_argument("--health-log", default=str(APP_DIR / "tools" / "recrawl-health.log"))
    parser.add_argument("--stats-url", default=DEFAULT_STATS_URL)
    parser.add_argument("--categories-url", default=DEFAULT_CATEGORIES_URL)
    parser.add_argument("--stats-fixture", default="")
    parser.add_argument("--categories-fixture", default="")
    parser.add_argument("--simulate-dns-failure", default="")
    parser.add_argument("--same-day-public-proof", default="")
    parser.add_argument("--crawler-success", type=int)
    parser.add_argument("--crawler-failed", type=int)
    parser.add_argument("--crawler-total", type=int)
    return parser.parse_args(argv)


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    stats = None
    categories = None
    probe_error = None

    try:
        if args.simulate_dns_failure:
            raise ProbeError(args.simulate_dns_failure)
        stats_raw = load_json_fixture(args.stats_fixture) if args.stats_fixture else fetch_json(args.stats_url)
        categories_raw = load_json_fixture(args.categories_fixture) if args.categories_fixture else fetch_json(args.categories_url)
        stats = normalize_stats(stats_raw)
        categories = normalize_categories(categories_raw)
    except (ProbeError, ValueError) as exc:
        probe_error = str(exc)

    body = render_closeout(args, stats, categories, probe_error)
    output = Path(args.output) if args.output else APP_DIR / "harness" / f"full-recrawl-closeout-{args.date}.md"
    output.parent.mkdir(parents=True, exist_ok=True)
    tmp = output.with_suffix(output.suffix + ".tmp")
    tmp.write_text(body, encoding="utf-8")
    os.replace(tmp, output)
    print(output)
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
