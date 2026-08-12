#!/usr/bin/env python3
"""Bounded read-only recheck for genuine NHS developer-tools demand evidence."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import subprocess
import tempfile
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Callable


REVISION_RE = re.compile(r"^[0-9a-f]{40}$")
DIGEST_RE = re.compile(r"^sha256:[0-9a-f]{64}$")
MAX_OUTPUT_BYTES = 512 * 1024
Runner = Callable[..., subprocess.CompletedProcess[str]]


class MonitorError(RuntimeError):
    pass


def stamp() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


def atomic_json(path: Path, value: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile(
        mode="w", encoding="utf-8", dir=path.parent, delete=False
    ) as handle:
        json.dump(value, handle, indent=2, sort_keys=True)
        handle.write("\n")
        temporary = Path(handle.name)
    temporary.replace(path)


def read_object(path: Path) -> dict[str, Any]:
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        raise MonitorError(f"invalid JSON evidence at {path}") from error
    if not isinstance(value, dict):
        raise MonitorError(f"JSON evidence at {path} is not an object")
    return value


def json_values(raw: str) -> list[Any]:
    if len(raw.encode("utf-8")) > MAX_OUTPUT_BYTES:
        raise MonitorError("command output exceeds the bounded maximum")
    decoder = json.JSONDecoder()
    values: list[Any] = []
    for index, character in enumerate(raw):
        if character not in "[{":
            continue
        try:
            value, _ = decoder.raw_decode(raw[index:])
        except json.JSONDecodeError:
            continue
        values.append(value)
    return values


def run(runner: Runner, command: list[str]) -> str:
    result = runner(
        command,
        capture_output=True,
        text=True,
        timeout=60,
        check=False,
        stdin=subprocess.DEVNULL,
    )
    if result.returncode != 0:
        detail = (result.stderr or result.stdout or "command failed").strip()
        raise MonitorError(detail[-1200:])
    return result.stdout


def fly_command(codex_secret: str, flyctl: str, arguments: list[str]) -> list[str]:
    return [
        codex_secret,
        "run",
        "--env",
        "FLY_API_TOKEN=FLY_API_TOKEN",
        "--",
        flyctl,
        *arguments,
    ]


def select_machine(raw: str) -> dict[str, Any]:
    inventories = [value for value in json_values(raw) if isinstance(value, list)]
    candidates: list[dict[str, Any]] = []
    for inventory in inventories:
        for item in inventory:
            if not isinstance(item, dict) or item.get("state") != "started":
                continue
            config = item.get("config")
            image_ref = item.get("image_ref")
            if not isinstance(config, dict) or not isinstance(image_ref, dict):
                continue
            env = config.get("env")
            labels = image_ref.get("labels")
            revision = labels.get("org.opencontainers.image.revision") if isinstance(labels, dict) else None
            digest = image_ref.get("digest")
            if (
                isinstance(env, dict)
                and env.get("NHS_PROVIDER_EXCHANGE_MODE") == "disabled"
                and isinstance(revision, str)
                and REVISION_RE.fullmatch(revision)
                and isinstance(digest, str)
                and DIGEST_RE.fullmatch(digest)
            ):
                candidates.append(item)
    unique = {item.get("id"): item for item in candidates if isinstance(item.get("id"), str)}
    if len(unique) != 1:
        raise MonitorError("exactly one started disabled-mode NHS machine was not found")
    return next(iter(unique.values()))


def exact_receipt(raw: str, contract: str) -> dict[str, Any]:
    matches = [
        value
        for value in json_values(raw)
        if isinstance(value, dict) and value.get("contract") == contract
    ]
    if len(matches) != 1:
        raise MonitorError(f"exactly one {contract} receipt was not returned")
    return matches[0]


def report_digest(report: dict[str, Any]) -> str:
    encoded = json.dumps(report, separators=(",", ":"), ensure_ascii=True).encode("utf-8")
    return hashlib.sha256(encoded).hexdigest()


def validate_funnel(receipt: dict[str, Any], revision: str, since: str) -> dict[str, Any]:
    if receipt.get("candidate_revision") != revision or receipt.get("binary_revision") != revision:
        raise MonitorError("funnel receipt revision mismatch")
    report = receipt.get("report")
    if not isinstance(report, dict) or report.get("since") != since:
        raise MonitorError("funnel receipt window mismatch")
    if receipt.get("report_sha256") != report_digest(report):
        raise MonitorError("funnel receipt digest mismatch")
    false_assertions = (
        "contains_identifiers",
        "contains_queries_or_prompts",
        "contains_contact_data",
        "operator_contacted_provider",
        "operator_changed_commercial_state",
        "operator_affected_organic_rank",
    )
    if any(receipt.get(name) is not False for name in false_assertions):
        raise MonitorError("funnel privacy or non-action assertion failed")
    if report.get("commercial_state_events_total") != 0:
        raise MonitorError("commercial state changed while provider exchange should be disabled")
    return report


def validate_stage1(receipt: dict[str, Any], revision: str) -> dict[str, Any]:
    if receipt.get("candidate_revision") != revision or receipt.get("binary_revision") != revision:
        raise MonitorError("Stage 1 receipt revision mismatch")
    report = receipt.get("stage1_demand")
    attempts = receipt.get("action_interest_attempt_funnel")
    if not isinstance(report, dict) or not isinstance(attempts, dict):
        raise MonitorError("Stage 1 aggregate projections are missing")
    if receipt.get("stage1_report_sha256") != report_digest(report):
        raise MonitorError("Stage 1 projection digest mismatch")
    if receipt.get("attempt_funnel_sha256") != report_digest(attempts):
        raise MonitorError("attempt projection digest mismatch")
    if (
        receipt.get("operator_contacted_provider") is not False
        or receipt.get("operator_changed_commercial_state") is not False
        or receipt.get("contains_identifiers") is not False
        or receipt.get("contains_queries_or_prompts") is not False
        or receipt.get("contains_contact_data") is not False
    ):
        raise MonitorError("Stage 1 privacy or non-action assertion failed")
    return report


def count(report: dict[str, Any], name: str) -> int:
    value = report.get(name)
    if not isinstance(value, int) or isinstance(value, bool) or value < 0:
        raise MonitorError(f"invalid aggregate counter {name}")
    return value


def notify(title: str, body: str, runner: Runner) -> None:
    script = (
        "on run argv\n"
        "display notification (item 2 of argv) with title (item 1 of argv)\n"
        "end run"
    )
    runner(
        ["/usr/bin/osascript", "-e", script, title, body[:500]],
        capture_output=True,
        text=True,
        timeout=10,
        check=False,
        stdin=subprocess.DEVNULL,
    )


def monitor(args: argparse.Namespace, runner: Runner = subprocess.run) -> dict[str, Any]:
    control = read_object(Path(args.control))
    if control.get("enabled") is not True:
        return {
            "contract": "nhs-demand-trigger-monitor-v1",
            "status": "paused",
            "checked_at": stamp(),
            "exit_code": 0,
        }
    baseline = read_object(Path(args.baseline))
    baseline_receipt = baseline.get("reader_receipt")
    if not isinstance(baseline_receipt, dict):
        raise MonitorError("baseline reader receipt is missing")
    baseline_report = baseline_receipt.get("report")
    if not isinstance(baseline_report, dict):
        raise MonitorError("baseline funnel report is missing")
    since = baseline_report.get("since")
    if not isinstance(since, str) or not since.endswith("Z"):
        raise MonitorError("baseline experiment window is invalid")

    inventory_raw = run(
        runner,
        fly_command(args.codex_secret, args.flyctl, ["machines", "list", "--app", args.app, "--json"]),
    )
    machine = select_machine(inventory_raw)
    machine_id = machine["id"]
    image_ref = machine["image_ref"]
    revision = image_ref["labels"]["org.opencontainers.image.revision"]

    funnel_raw = run(
        runner,
        fly_command(
            args.codex_secret,
            args.flyctl,
            [
                "ssh", "console", "--app", args.app, "--machine", machine_id, "--quiet",
                "--command", f"./action-interest-experiment-read --revision {revision} --since {since}",
            ],
        ),
    )
    funnel_receipt = exact_receipt(funnel_raw, "nhs-post-selection-action-interest-experiment-read-v3")
    funnel = validate_funnel(funnel_receipt, revision, since)

    stage1_raw = run(
        runner,
        fly_command(
            args.codex_secret,
            args.flyctl,
            [
                "ssh", "console", "--app", args.app, "--machine", machine_id, "--quiet",
                "--command", f"./stage1-demand-read --revision {revision} --days 30",
            ],
        ),
    )
    stage1_receipt = exact_receipt(stage1_raw, "nhs-stage1-demand-read-v1")
    stage1 = validate_stage1(stage1_receipt, revision)

    fields = (
        "developer_tools_search_receipts",
        "developer_tools_result_selections",
        "developer_tools_search_receipts_with_selection",
        "developer_tools_action_interest_receipts",
        "developer_tools_search_receipts_with_action_interest",
        "developer_tools_post_selection_action_interest_receipts",
        "developer_tools_post_selection_search_receipts",
    )
    current = {name: count(funnel, name) for name in fields}
    initial = {name: count(baseline_report, name) for name in fields}
    deltas = {name: current[name] - initial[name] for name in fields}
    if any(value < 0 for value in deltas.values()):
        raise MonitorError("developer-tools aggregate regressed against the sealed baseline")
    trigger_reasons = []
    if deltas["developer_tools_search_receipts_with_selection"] > 0:
        trigger_reasons.append("new_developer_tools_selected_search_receipt")
    if deltas["developer_tools_search_receipts_with_action_interest"] > 0:
        trigger_reasons.append("new_developer_tools_explicit_interest_receipt")
    if deltas["developer_tools_post_selection_search_receipts"] > 0:
        trigger_reasons.append("new_developer_tools_post_selection_interest")
    if stage1.get("stage1_ready") is True:
        trigger_reasons.append("stage1_ready")

    trigger_payload = {
        "contract": "nhs-demand-evidence-trigger-v1",
        "status": "triggered" if trigger_reasons else "quiet",
        "exit_code": 0,
        "checked_at": stamp(),
        "triggered": bool(trigger_reasons),
        "trigger_reasons": trigger_reasons,
        "revision": revision,
        "image_digest": image_ref["digest"],
        "provider_exchange_mode": "disabled",
        "experiment_since": since,
        "developer_tools_current": current,
        "developer_tools_net_change_from_baseline": deltas,
        "stage1_as_of": stage1.get("as_of"),
        "stage1_ready": stage1.get("stage1_ready") is True,
        "stage1_selected_search_receipts": count(stage1, "search_receipts_with_selection"),
        "stage1_interest_search_receipts": count(stage1, "search_receipts_with_action_interest"),
        "commercial_proof": False,
        "authorizes_provider_contact_or_activation": False,
        "next_safe_action_if_triggered": (
            "Open the sealed aggregate receipt, re-evaluate Stage 1 and developer-tools intent, "
            "and request separate owner authorization before any provider contact or activation."
        ),
    }
    fingerprint = hashlib.sha256(
        json.dumps(
            {
                "reasons": trigger_reasons,
                "current": current,
                "stage1_ready": trigger_payload["stage1_ready"],
            },
            sort_keys=True,
            separators=(",", ":"),
        ).encode("utf-8")
    ).hexdigest()
    trigger_payload["trigger_fingerprint"] = fingerprint
    state_path = Path(args.state)
    previous = {}
    if state_path.exists():
        try:
            previous = read_object(state_path)
        except MonitorError:
            previous = {}
    trigger_payload["notification_emitted"] = False
    if trigger_reasons and previous.get("trigger_fingerprint") != fingerprint:
        notify(
            "NHS demand evidence changed",
            "Developer-tools or Stage 1 evidence crossed a trigger. Provider mode remains disabled; review is required.",
            runner,
        )
        trigger_payload["notification_emitted"] = True
    atomic_json(state_path, trigger_payload)
    if trigger_reasons:
        atomic_json(Path(args.trigger), trigger_payload)
    return trigger_payload


def parser() -> argparse.ArgumentParser:
    value = argparse.ArgumentParser(description=__doc__)
    value.add_argument("--baseline", required=True)
    value.add_argument("--control", required=True)
    value.add_argument("--state", required=True)
    value.add_argument("--trigger", required=True)
    value.add_argument("--app", default="nothumansearch")
    value.add_argument("--flyctl", default="/Users/shane/.fly/bin/flyctl")
    value.add_argument("--codex-secret", default="/Users/shane/.local/bin/codex-secret")
    return value


def main() -> int:
    args = parser().parse_args()
    try:
        result = monitor(args)
    except (MonitorError, OSError, subprocess.SubprocessError) as error:
        failure = {
            "contract": "nhs-demand-trigger-monitor-v1",
            "status": "error",
            "exit_code": 2,
            "checked_at": stamp(),
            "error": str(error)[-1200:],
        }
        atomic_json(Path(args.state), failure)
        print(json.dumps(failure, sort_keys=True))
        return 2
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
