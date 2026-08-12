#!/usr/bin/env python3
"""Read the privacy-bounded NHS Stage 1 receipt from an existing Fly machine."""

from __future__ import annotations

import argparse
import hashlib
import json
import re
import subprocess
import sys
from typing import Any, Callable


MAX_COMMAND_OUTPUT_BYTES = 256 * 1024
REVISION_RE = re.compile(r"^[0-9a-f]{40}$")
DIGEST_RE = re.compile(r"^sha256:[0-9a-f]{64}$")
ARCHIVE_RE = re.compile(r"^[0-9a-f]{64}$")
APP_RE = re.compile(r"^[a-z0-9][a-z0-9-]{0,62}$")
MACHINE_RE = re.compile(r"^[0-9a-f]{14}$")
FORBIDDEN_RECEIPT_KEYS = {
    "search_id",
    "query",
    "queries",
    "prompt",
    "prompts",
    "domain",
    "domains",
    "contact",
    "contact_data",
    "email",
    "phone",
    "ip",
    "ip_address",
    "user_agent",
    "referrer",
    "agent_id",
    "principal_id",
    "provider_id",
    "ticket_id",
    "outcome_id",
    "settlement_id",
    "request_id",
    "attribution_token",
}


class ReadError(RuntimeError):
    pass


Runner = Callable[..., subprocess.CompletedProcess[str]]


def _objects(raw: str) -> list[Any]:
    if len(raw.encode("utf-8")) > MAX_COMMAND_OUTPUT_BYTES:
        raise ReadError("command output exceeds the bounded receipt size")
    decoder = json.JSONDecoder()
    found: list[Any] = []
    for index, character in enumerate(raw):
        if character not in "[{":
            continue
        try:
            value, _ = decoder.raw_decode(raw[index:])
        except json.JSONDecodeError:
            continue
        found.append(value)
    return found


def _run(runner: Runner, command: list[str]) -> str:
    result = runner(command, capture_output=True, text=True, timeout=45, check=False)
    if result.returncode != 0:
        message = (result.stderr or result.stdout or "command failed").strip()
        raise ReadError(message[-1000:])
    return result.stdout


def _secret_command(codex_secret: str, flyctl: str, arguments: list[str]) -> list[str]:
    return [
        codex_secret,
        "run",
        "--env",
        "FLY_API_TOKEN=FLY_API_TOKEN",
        "--",
        flyctl,
        *arguments,
    ]


def _machine(raw: str, machine_id: str, revision: str) -> dict[str, Any]:
    inventories = [value for value in _objects(raw) if isinstance(value, list)]
    if not inventories:
        raise ReadError("Fly machine inventory was not valid JSON")
    matches = [
        item
        for inventory in inventories
        for item in inventory
        if isinstance(item, dict) and item.get("id") == machine_id
    ]
    if len(matches) != 1:
        raise ReadError("the exact Fly machine was not found once")
    machine = matches[0]
    image_ref = machine.get("image_ref")
    config = machine.get("config")
    if not isinstance(image_ref, dict) or not isinstance(config, dict):
        raise ReadError("Fly machine image metadata is unavailable")
    labels = image_ref.get("labels")
    env = config.get("env")
    digest = image_ref.get("digest")
    if machine.get("state") != "started":
        raise ReadError("the selected Fly machine is not started")
    if not isinstance(labels, dict) or labels.get("org.opencontainers.image.revision") != revision:
        raise ReadError("Fly image revision does not match the requested receipt revision")
    if not isinstance(env, dict) or env.get("NHS_PROVIDER_EXCHANGE_MODE") != "disabled":
        raise ReadError("provider exchange is not explicitly disabled")
    if not isinstance(digest, str) or DIGEST_RE.fullmatch(digest) is None:
        raise ReadError("Fly image digest is not exact")
    archive = labels.get("org.opencontainers.image.source_archive_sha256")
    if not isinstance(archive, str) or ARCHIVE_RE.fullmatch(archive) is None:
        raise ReadError("Fly source archive label is missing")
    return machine


def _reject_sensitive_keys(value: Any) -> None:
    if isinstance(value, dict):
        for key, nested in value.items():
            if key in FORBIDDEN_RECEIPT_KEYS:
                raise ReadError(f"receipt contains forbidden field {key}")
            _reject_sensitive_keys(nested)
    elif isinstance(value, list):
        for nested in value:
            _reject_sensitive_keys(nested)


def _sha256_json(value: Any) -> str:
    encoded = json.dumps(value, ensure_ascii=True, separators=(",", ":")).encode("utf-8")
    return hashlib.sha256(encoded).hexdigest()


def _receipt(raw: str, revision: str) -> dict[str, Any]:
    matches = [
        value
        for value in _objects(raw)
        if isinstance(value, dict) and value.get("contract") == "nhs-stage1-demand-read-v1"
    ]
    if len(matches) != 1:
        raise ReadError("exactly one Stage 1 receipt was not returned")
    receipt = matches[0]
    stage1 = receipt.get("stage1_demand")
    attempts = receipt.get("action_interest_attempt_funnel")
    if receipt.get("candidate_revision") != revision or receipt.get("binary_revision") != revision:
        raise ReadError("Stage 1 receipt revision is not exact")
    if not isinstance(stage1, dict) or not isinstance(attempts, dict):
        raise ReadError("Stage 1 receipt projections are missing")
    if receipt.get("stage1_report_sha256") != _sha256_json(stage1):
        raise ReadError("Stage 1 projection digest does not match")
    if receipt.get("attempt_funnel_sha256") != _sha256_json(attempts):
        raise ReadError("action-interest attempt projection digest does not match")
    required_true = {
        "independent_read_only_snapshots": receipt.get("independent_read_only_snapshots"),
        "searches_are_not_leads": receipt.get("searches_are_not_leads"),
        "readiness_does_not_authorize_stage2": receipt.get("readiness_does_not_authorize_stage2"),
        "synthetic_excluded": stage1.get("synthetic_excluded"),
        "counts_are_receipts_not_unique_agents": stage1.get("counts_are_receipts_not_unique_agents"),
        "counts_are_attempts_not_unique_agents": attempts.get("counts_are_attempts_not_unique_agents"),
    }
    required_false = {
        "contains_identifiers": receipt.get("contains_identifiers"),
        "contains_queries_or_prompts": receipt.get("contains_queries_or_prompts"),
        "contains_contact_data": receipt.get("contains_contact_data"),
        "operator_contacted_provider": receipt.get("operator_contacted_provider"),
        "operator_changed_commercial_state": receipt.get("operator_changed_commercial_state"),
        "operator_affected_organic_rank": receipt.get("operator_affected_organic_rank"),
        "stage1_commercial_proof": stage1.get("commercial_proof"),
        "attempt_commercial_proof": attempts.get("commercial_proof"),
        "attempt_contains_request_coordinates": attempts.get("contains_request_coordinates"),
    }
    if any(value is not True for value in required_true.values()):
        raise ReadError("Stage 1 receipt is missing a required safety assertion")
    if any(value is not False for value in required_false.values()):
        raise ReadError("Stage 1 receipt violates a required safety assertion")
    _reject_sensitive_keys(receipt)
    return receipt


def collect(args: argparse.Namespace, runner: Runner = subprocess.run) -> dict[str, Any]:
    if REVISION_RE.fullmatch(args.revision) is None:
        raise ReadError("revision must be exactly 40 lowercase hexadecimal characters")
    if APP_RE.fullmatch(args.app) is None or MACHINE_RE.fullmatch(args.machine) is None:
        raise ReadError("Fly app or machine identifier is invalid")
    if args.days < 1 or args.days > 30:
        raise ReadError("days must be between 1 and 30")
    inventory_raw = _run(
        runner,
        _secret_command(args.codex_secret, args.flyctl, ["machines", "list", "--app", args.app, "--json"]),
    )
    machine = _machine(inventory_raw, args.machine, args.revision)
    reader_command = f"./stage1-demand-read --revision {args.revision} --days {args.days}"
    receipt_raw = _run(
        runner,
        _secret_command(
            args.codex_secret,
            args.flyctl,
            [
                "ssh",
                "console",
                "--app",
                args.app,
                "--machine",
                args.machine,
                "--quiet",
                "--command",
                reader_command,
            ],
        ),
    )
    receipt = _receipt(receipt_raw, args.revision)
    image_ref = machine["image_ref"]
    return {
        "contract": "nhs-stage1-fly-read-evidence-v1",
        "operator_run": {
            "app": args.app,
            "machine_id": args.machine,
            "machine_state": machine["state"],
            "image": machine["config"].get("image"),
            "image_digest": image_ref["digest"],
            "source_archive_sha256": image_ref["labels"]["org.opencontainers.image.source_archive_sha256"],
            "provider_exchange_mode": "disabled",
            "used_existing_machine": True,
            "created_or_destroyed_machine": False,
        },
        "reader_receipt": receipt,
    }


def parser() -> argparse.ArgumentParser:
    value = argparse.ArgumentParser(description=__doc__)
    value.add_argument("--revision", required=True)
    value.add_argument("--machine", required=True)
    value.add_argument("--days", type=int, default=30)
    value.add_argument("--app", default="nothumansearch")
    value.add_argument("--flyctl", default="/Users/shane/.fly/bin/flyctl")
    value.add_argument("--codex-secret", default="/Users/shane/.local/bin/codex-secret")
    return value


def main() -> int:
    try:
        evidence = collect(parser().parse_args())
    except (ReadError, subprocess.SubprocessError) as error:
        print(f"stage1_fly_read_error: {error}", file=sys.stderr)
        return 2
    json.dump(evidence, sys.stdout, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
