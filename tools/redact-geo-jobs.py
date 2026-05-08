#!/usr/bin/env python3
"""Emit redacted aggregates for /api/v1/admin/geo-jobs JSON."""

import datetime as dt
import argparse
import json
import sys
from collections import Counter


FOUNDRY_DOMAINS = {
    "nothumansearch.ai",
    "nothumansearch.com",
    "aidevboard.com",
    "8bitconcepts.com",
    "bringyour.ai",
}


def parse_time(value):
    if not value:
        return None
    return dt.datetime.fromisoformat(value.replace("Z", "+00:00"))


def age_bucket(created_at, now):
    created = parse_time(created_at)
    if created is None:
        return "unknown"
    days = (now - created).days
    if days < 1:
        return "lt_1d"
    if days < 7:
        return "1_6d"
    if days < 30:
        return "7_29d"
    return "30d_plus"


def host_class(host):
    host = (host or "").lower().strip(".")
    if host in FOUNDRY_DOMAINS or any(host.endswith("." + d) for d in FOUNDRY_DOMAINS):
        return "foundry_owned"
    if host.endswith(".ai"):
        return "ai_domain"
    if host.endswith((".dev", ".io")):
        return "developer_domain"
    if host.endswith(".com"):
        return "dot_com"
    return "other_tld"


def is_test_like(job):
    email = (job.get("email") or "").lower()
    host = (job.get("host") or "").lower()
    notes = (job.get("notes") or "").lower()
    stripe_session_id = (job.get("stripe_session_id") or "").lower()
    return (
        any(marker in email for marker in ("test@", "example.com", "owl-test", "test+"))
        or any(marker in host for marker in ("example.", "localhost", "test."))
        or "test" in notes
        or stripe_session_id.startswith("cs_test")
    )


def empty_output(error, aggregate_only=False):
    output = {
        "error": error,
        "count": 0,
        "summary": [],
        "by_status_host_class": [],
        "age_buckets": [],
    }
    if not aggregate_only:
        output["real_paid_or_lead_refs"] = []
        output["test_like_refs"] = []
    json.dump(output, sys.stdout, indent=2, sort_keys=True)
    sys.stdout.write("\n")


def main():
    parser = argparse.ArgumentParser(
        description="Emit redacted aggregates for /api/v1/admin/geo-jobs JSON."
    )
    parser.add_argument(
        "--aggregate-only",
        action="store_true",
        help="omit row refs and emit only aggregate-safe buckets",
    )
    args = parser.parse_args()

    raw = sys.stdin.read()
    if not raw.strip():
        empty_output("empty_input", args.aggregate_only)
        return

    try:
        payload = json.loads(raw)
    except json.JSONDecodeError:
        empty_output("invalid_json_input", args.aggregate_only)
        return

    jobs = payload.get("jobs", [])
    now = dt.datetime.now(dt.timezone.utc)

    summary = Counter()
    by_host_class = Counter()
    by_age = Counter()
    real_paid_or_lead_refs = []
    test_like_refs = []

    for job in jobs:
        status = job.get("status") or "unknown"
        klass = "test_like" if is_test_like(job) else "real_candidate"
        hclass = host_class(job.get("host"))
        abucket = age_bucket(job.get("created_at"), now)
        row_ref = {
            "id": job.get("id"),
            "status": status,
            "host_class": hclass,
            "age_bucket": abucket,
        }

        summary[(klass, status)] += 1
        by_host_class[(klass, status, hclass)] += 1
        by_age[(klass, status, abucket)] += 1

        if klass == "test_like":
            test_like_refs.append(row_ref)
        elif status in ("paid", "lead"):
            real_paid_or_lead_refs.append(row_ref)

    output = {
        "count": payload.get("count", len(jobs)),
        "summary": [
            {"class": key[0], "status": key[1], "count": count}
            for key, count in sorted(summary.items())
        ],
        "by_status_host_class": [
            {"class": key[0], "status": key[1], "host_class": key[2], "count": count}
            for key, count in sorted(by_host_class.items())
        ],
        "age_buckets": [
            {"class": key[0], "status": key[1], "age_bucket": key[2], "count": count}
            for key, count in sorted(by_age.items())
        ],
    }
    if not args.aggregate_only:
        output["real_paid_or_lead_refs"] = real_paid_or_lead_refs
        output["test_like_refs"] = test_like_refs
    json.dump(output, sys.stdout, indent=2, sort_keys=True)
    sys.stdout.write("\n")


if __name__ == "__main__":
    main()
