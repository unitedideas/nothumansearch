#!/usr/bin/env python3
"""Mark one Foundry-owned pending geo-fix row internal_test without printing row data."""

import datetime as dt
import json
import os
import subprocess
import sys
import urllib.error
import urllib.request
from collections import Counter


FOUNDRY_DOMAINS = {
    "nothumansearch.ai",
    "nothumansearch.com",
    "aidevboard.com",
    "8bitconcepts.com",
    "bringyour.ai",
}


def keychain_value(service):
    return subprocess.check_output(
        ["/usr/bin/security", "find-generic-password", "-a", "foundry", "-s", service, "-w"],
        text=True,
        stderr=subprocess.DEVNULL,
    ).strip()


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


def age_bucket(created_at, now):
    if not created_at:
        return "unknown"
    created = dt.datetime.fromisoformat(created_at.replace("Z", "+00:00"))
    days = (now - created).days
    if days < 1:
        return "lt_1d"
    if days < 7:
        return "1_6d"
    if days < 30:
        return "7_29d"
    return "30d_plus"


def fetch_jobs(base_url, token, limit):
    req = urllib.request.Request(
        f"{base_url}/api/v1/admin/geo-jobs?limit={limit}",
        headers={
            "Authorization": f"Bearer {token}",
            "User-Agent": "curl/8.7.1",
        },
    )
    with urllib.request.urlopen(req, timeout=30) as response:
        return json.loads(response.read().decode("utf-8"))


def post_action(base_url, token, job_id):
    body = json.dumps(
        {
            "id": job_id,
            "action": "mark_internal_test",
            "operator": "business-agent-not-human-search",
            "source": "business-agent-not-human-search",
            "notes": "aggregate-only internal test classification",
        }
    ).encode("utf-8")
    req = urllib.request.Request(
        f"{base_url}/api/v1/admin/geo-jobs/action",
        data=body,
        method="POST",
        headers={
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
            "User-Agent": "curl/8.7.1",
        },
    )
    with urllib.request.urlopen(req, timeout=30) as response:
        return response.status


def aggregate(payload):
    now = dt.datetime.now(dt.timezone.utc)
    summary = Counter()
    by_host_class = Counter()
    by_age = Counter()
    for job in payload.get("jobs", []):
        status = job.get("status") or "unknown"
        klass = "test_like" if is_test_like(job) else "real_candidate"
        hclass = host_class(job.get("host"))
        bucket = age_bucket(job.get("created_at"), now)
        summary[(klass, status)] += 1
        by_host_class[(klass, status, hclass)] += 1
        by_age[(klass, status, bucket)] += 1
    return {
        "count": payload.get("count", len(payload.get("jobs", []))),
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


def find_target(payload):
    targets = [
        job
        for job in payload.get("jobs", [])
        if (job.get("status") or "") == "pending"
        and host_class(job.get("host")) == "foundry_owned"
        and not is_test_like(job)
    ]
    if len(targets) != 1:
        return None, len(targets)
    return targets[0], 1


def main():
    base_url = os.environ.get("NHS_BASE_URL", "https://nothumansearch.ai").rstrip("/")
    limit = int(os.environ.get("NHS_GEO_JOBS_LIMIT", "500"))
    service = os.environ.get("NHS_ADMIN_KEYCHAIN_SERVICE", "nhs-admin-api-key")
    try:
        token = keychain_value(service)
        before_payload = fetch_jobs(base_url, token, limit)
        target, target_count = find_target(before_payload)
        if target is None:
            print(
                json.dumps(
                    {
                        "ok": False,
                        "reason": "expected exactly one eligible foundry_owned pending row",
                        "eligible_foundry_owned_pending_count": target_count,
                        "before": aggregate(before_payload),
                    },
                    indent=2,
                    sort_keys=True,
                )
            )
            return 1
        status = post_action(base_url, token, target["id"])
        after_payload = fetch_jobs(base_url, token, limit)
        print(
            json.dumps(
                {
                    "ok": status == 200,
                    "action": "mark_internal_test",
                    "mutated_class": "foundry_owned_pending",
                    "before": aggregate(before_payload),
                    "after": aggregate(after_payload),
                },
                indent=2,
                sort_keys=True,
            )
        )
        return 0 if status == 200 else 1
    except (subprocess.CalledProcessError, urllib.error.URLError, TimeoutError, ValueError) as err:
        print(json.dumps({"ok": False, "error": err.__class__.__name__}, indent=2, sort_keys=True))
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
