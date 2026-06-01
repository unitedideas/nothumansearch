#!/usr/bin/env python3
"""Rerun quarantined monitors and apply aggregate-safe admin outcomes."""

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


def admin_token():
    service = os.environ.get("NHS_ADMIN_KEYCHAIN_SERVICE")
    candidates = [service] if service else ["nhs-admin-api-key", "nothumansearch-admin-key"]
    for candidate in candidates:
        if not candidate:
            continue
        try:
            return keychain_value(candidate)
        except subprocess.CalledProcessError:
            continue
    raise RuntimeError("missing Keychain service: nhs-admin-api-key or nothumansearch-admin-key")


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


def find_targets(monitors):
    return [
        monitor
        for monitor in monitors
        if monitor.get("status") == "quarantined"
        and (monitor.get("quarantine_reason") or "") == ZERO_SCORE_REASON
    ]


def rerun_and_apply(base_url, token, target):
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
    return {
        "ok": action_status == 200,
        "action": action,
        "rerun_score_bucket": score_bucket(rerun_score),
        "rerun_error_class": rerun_error,
    }


def main():
    base_url = os.environ.get("NHS_BASE_URL", "https://nothumansearch.ai").rstrip("/")
    try:
        token = admin_token()
        _, before_payload = fetch_json(f"{base_url}/api/v1/admin/monitors?limit=100", token=token)
        monitors = before_payload.get("monitors", [])
        targets = find_targets(monitors)
        if not targets:
            print(
                json.dumps(
                    {
                        "ok": True,
                        "reason": "no first-check zero-score quarantines require review",
                        "eligible_quarantined_count": 0,
                        "before_status_counts": status_counts(monitors),
                    },
                    indent=2,
                    sort_keys=True,
                )
            )
            return 0

        outcomes = [rerun_and_apply(base_url, token, target) for target in targets]
        _, after_payload = fetch_json(f"{base_url}/api/v1/admin/monitors?limit=100", token=token)
        _, action_counts = fetch_json(f"{base_url}/api/v1/admin/monitors/actions?days=30", token=token)
        action_summary = Counter(outcome["action"] for outcome in outcomes)
        score_bucket_summary = Counter(outcome["rerun_score_bucket"] for outcome in outcomes)

        print(
            json.dumps(
                {
                    "ok": all(outcome["ok"] for outcome in outcomes),
                    "reviewed_quarantined_count": len(outcomes),
                    "action_summary": dict(sorted(action_summary.items())),
                    "rerun_score_bucket_summary": dict(sorted(score_bucket_summary.items())),
                    "rerun_error_classes": sorted(
                        {
                            outcome["rerun_error_class"]
                            for outcome in outcomes
                            if outcome["rerun_error_class"]
                        }
                    ),
                    "before_status_counts": status_counts(monitors),
                    "after_status_counts": status_counts(after_payload.get("monitors", [])),
                    "action_counts": action_counts.get("counts", []),
                },
                indent=2,
                sort_keys=True,
            )
        )
        return 0 if all(outcome["ok"] for outcome in outcomes) else 1
    except (RuntimeError, urllib.error.URLError, json.JSONDecodeError) as err:
        payload = {"ok": False, "error": err.__class__.__name__}
        if isinstance(err, RuntimeError):
            payload["message"] = str(err)
        print(json.dumps(payload, indent=2, sort_keys=True))
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
