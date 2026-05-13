#!/usr/bin/env python3
"""Rerun one quarantined monitor and apply an aggregate-safe admin outcome."""

import json
import os
import subprocess
import sys
import urllib.error
import urllib.request
from collections import Counter


ZERO_SCORE_REASON = "first monitor check returned zero agentic score"


def keychain_value(service):
    return subprocess.check_output(
        ["/usr/bin/security", "find-generic-password", "-a", "foundry", "-s", service, "-w"],
        text=True,
        stderr=subprocess.DEVNULL,
    ).strip()


def fetch_json(url, token=None, data=None):
    headers = {"User-Agent": "curl/8.7.1"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    if data is not None:
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, headers=headers, method="POST" if data else "GET")
    with urllib.request.urlopen(req, timeout=35) as response:
        return response.status, json.loads(response.read().decode("utf-8"))


def status_counts(monitors):
    counts = Counter()
    for monitor in monitors:
        status = monitor.get("status") or "unknown"
        reason = monitor.get("quarantine_reason") or ""
        counts[(status, reason)] += 1
    return [
        {"status": key[0], "quarantine_reason": key[1], "count": value}
        for key, value in sorted(counts.items())
    ]


def score_bucket(score):
    if score is None:
        return "crawl_failed"
    if score == 0:
        return "zero"
    if score < 70:
        return "1_69"
    return "70_plus"


def find_target(monitors):
    targets = [
        monitor
        for monitor in monitors
        if monitor.get("status") == "quarantined"
        and (monitor.get("quarantine_reason") or "") == ZERO_SCORE_REASON
    ]
    if len(targets) != 1:
        return None, len(targets)
    return targets[0], 1


def main():
    base_url = os.environ.get("NHS_BASE_URL", "https://nothumansearch.ai").rstrip("/")
    service = os.environ.get("NHS_ADMIN_KEYCHAIN_SERVICE", "nhs-admin-api-key")
    try:
        token = keychain_value(service)
        _, before_payload = fetch_json(f"{base_url}/api/v1/admin/monitors?limit=100", token=token)
        monitors = before_payload.get("monitors", [])
        target, target_count = find_target(monitors)
        if target is None:
            print(
                json.dumps(
                    {
                        "ok": False,
                        "reason": "expected exactly one quarantined zero-score monitor",
                        "eligible_quarantined_count": target_count,
                        "before_status_counts": status_counts(monitors),
                    },
                    indent=2,
                    sort_keys=True,
                )
            )
            return 1

        rerun_score = None
        rerun_error = None
        try:
            _, check_payload = fetch_json(
                f"{base_url}/api/v1/check",
                data=json.dumps({"url": target.get("domain")}).encode("utf-8"),
            )
            rerun_score = int(check_payload.get("agentic_score", 0))
        except (urllib.error.URLError, ValueError, TypeError, json.JSONDecodeError) as err:
            rerun_error = err.__class__.__name__

        action = "keep_quarantined" if not rerun_score else "approve_monitoring"
        notes = "bounded rerun still zero score" if action == "keep_quarantined" else "bounded rerun returned agentic score"
        if rerun_error:
            notes = f"bounded rerun failed: {rerun_error}"

        action_body = json.dumps(
            {
                "id": target.get("id"),
                "action": action,
                "operator": "business-agent-not-human-search",
                "source": "launchd_business_agent",
                "notes": notes,
            }
        ).encode("utf-8")
        action_status, _ = fetch_json(f"{base_url}/api/v1/admin/monitors/action", token=token, data=action_body)
        _, after_payload = fetch_json(f"{base_url}/api/v1/admin/monitors?limit=100", token=token)
        _, action_counts = fetch_json(f"{base_url}/api/v1/admin/monitors/actions?days=30", token=token)

        print(
            json.dumps(
                {
                    "ok": action_status == 200,
                    "action": action,
                    "rerun_score_bucket": score_bucket(rerun_score),
                    "rerun_error_class": rerun_error,
                    "before_status_counts": status_counts(monitors),
                    "after_status_counts": status_counts(after_payload.get("monitors", [])),
                    "action_counts": action_counts.get("counts", []),
                },
                indent=2,
                sort_keys=True,
            )
        )
        return 0 if action_status == 200 else 1
    except (subprocess.CalledProcessError, urllib.error.URLError, json.JSONDecodeError) as err:
        print(json.dumps({"ok": False, "error": err.__class__.__name__}, indent=2, sort_keys=True))
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
