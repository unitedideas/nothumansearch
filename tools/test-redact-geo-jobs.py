#!/usr/bin/env python3
"""Regression tests for aggregate-only geo-fix redaction."""

import json
import subprocess
import sys


def run_redactor(payload, *args):
    proc = subprocess.run(
        [sys.executable, "tools/redact-geo-jobs.py", *args],
        input=payload,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=True,
    )
    return json.loads(proc.stdout)


def assert_no_row_refs(output):
    forbidden = {"real_paid_or_lead_refs", "test_like_refs"}
    present = forbidden.intersection(output)
    if present:
        raise AssertionError(f"aggregate output exposed row-ref keys: {sorted(present)}")


def main():
    fixture = {
        "count": 3,
        "jobs": [
            {
                "id": 101,
                "host": "buyerdomain.com",
                "email": "owner@buyerdomain.com",
                "notes": "real checkout",
                "stripe_session_id": "cs_live_123",
                "status": "pending",
                "created_at": "2026-05-08T00:00:00Z",
            },
            {
                "id": 102,
                "host": "example.com",
                "email": "test@example.com",
                "notes": "test",
                "stripe_session_id": "cs_test_123",
                "status": "paid",
                "created_at": "2026-04-28T00:00:00Z",
            },
            {
                "id": 103,
                "host": "nothumansearch.ai",
                "email": "ops@nothumansearch.ai",
                "status": "lead",
                "created_at": "2026-05-07T00:00:00Z",
            },
        ],
    }

    full = run_redactor(json.dumps(fixture))
    if len(full.get("real_paid_or_lead_refs", [])) != 1:
        raise AssertionError("expected one real lead ref in full redacted output")
    if len(full.get("test_like_refs", [])) != 1:
        raise AssertionError("expected one test-like ref in full redacted output")

    aggregate = run_redactor(json.dumps(fixture), "--aggregate-only")
    assert_no_row_refs(aggregate)
    if aggregate["summary"] != [
        {"class": "real_candidate", "count": 1, "status": "lead"},
        {"class": "real_candidate", "count": 1, "status": "pending"},
        {"class": "test_like", "count": 1, "status": "paid"},
    ]:
        raise AssertionError(f"unexpected aggregate summary: {aggregate['summary']}")

    assert_no_row_refs(run_redactor("", "--aggregate-only"))
    assert_no_row_refs(run_redactor("{", "--aggregate-only"))
    print("ok")


if __name__ == "__main__":
    main()
