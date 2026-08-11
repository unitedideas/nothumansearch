#!/usr/bin/env python3
"""Read NHS Stage 1, pilot proof, and proof-manifest state safely.

The tool is deliberately production-host-fixed, read-only, bounded, and strict
about the proof relationships it reports. It never emits response fields that
are not part of the reviewed aggregate status contract.
"""

from __future__ import annotations

import argparse
import datetime
import hashlib
import json
import os
import re
import resource
import ssl
import sys
import urllib.error
import urllib.parse
import urllib.request
from typing import Mapping, Sequence


BASE_URL = "https://nothumansearch.ai"
ADMIN_KEY_ENV = "NHS_PROVIDER_OPERATOR_ADMIN_KEY"
MAX_RESPONSE_BYTES = 64 * 1024
REQUEST_TIMEOUT_SECONDS = 10
MAX_COUNT = 2**31 - 1
MAX_MONEY_CENTS = 2**63 - 1

STAGE1_TARGETS = {
    "meaningful_search_receipts": 100,
    "search_receipts_with_selection": 20,
    "search_receipts_with_action_interest": 10,
    "pilot_candidate_topic_receipts": 20,
    "observation_window_days": 14,
}
PROOF_TARGETS = {
    "verified_provider_companies": 3,
    "verified_provider_accepted_handoffs": 5,
    "verified_provider_confirmed_activations": 2,
    "verified_provider_renewals": 1,
}
PROOF_MANIFEST_CONTRACT_VERSION = "nhs-provider-proof-manifest-v1"
PROOF_MANIFEST_REVIEW_EVIDENCE_VERSION = "nhs-provider-proof-review-root-v1"
PROOF_MANIFEST_MARKET_POLICY_VERSION = "nhs-free-organic-provider-funded-v1"
PROOF_MANIFEST_VERIFICATION_SCOPE = "nhs-private-keyring"
PROOF_MANIFEST_SCOPE = (
    "NHS-recorded exact closed-pilot aggregate; HMAC-signed and verifiable only by NHS; not "
    "independent proof of provider truth, cash collection, or agent identity."
)
PROOF_MANIFEST_EVIDENCE_SCOPE = (
    "One owner-issued, privacy-redacted HMAC-signed aggregate for an exact closed Not Human Search "
    "pilot. Issuance requires authenticated outcome integrity, the 3/5/2/1 threshold, and complete "
    "chronological owner review of every qualifying provider, offer, ticket, handoff, and callback. "
    "Its signature is verifiable only by NHS; it proves what NHS recorded, not independent provider "
    "truth or cash collection, and it does not publish automatically."
)
ALLOWED_TOPICS = frozenset(
    {
        "payments", "commerce", "jobs", "data", "search", "weather", "maps",
        "email", "messaging", "image", "video", "audio", "documents", "security",
        "finance", "health", "education", "news", "analytics", "automation",
        "productivity", "identity", "storage", "ai-tools", "developer-tools", "other",
    }
)
ALLOWED_ACTION_TYPES = frozenset(
    {"quote", "trial", "demo", "booking", "application", "signup", "purchase"}
)
ALLOWED_ATTEMPT_SURFACES = frozenset({"rest", "mcp"})
ALLOWED_ATTEMPT_OUTCOMES = frozenset(
    {
        "created", "replayed", "invalid_request", "unavailable", "conflict",
        "rate_limited", "cross_origin", "store_unavailable", "internal_error",
    }
)
CHARGE_EVENTS = ("accepted", "activated", "converted")
_CURRENCY_PATTERN = re.compile(r"^[a-z]{3}$")
_HASH_PATTERN = re.compile(r"^[0-9a-f]{64}$")
_KEY_ID_PATTERN = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{2,63}$")
_SIGNATURE_PATTERN = re.compile(r"^[A-Za-z0-9_-]{43}$")
_UUID_PATTERN = re.compile(
    r"^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"
)


class StatusError(Exception):
    """A stable public error code with an optional safe HTTP status."""

    def __init__(self, code: str, *, http_status: int | None = None):
        super().__init__(code)
        self.code = code
        self.http_status = http_status


class SafeArgumentParser(argparse.ArgumentParser):
    def error(self, _message: str) -> None:
        raise StatusError("invalid_arguments")


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):  # noqa: ANN001
        return None


def build_parser() -> argparse.ArgumentParser:
    parser = SafeArgumentParser(description="Read aggregate NHS pilot status safely.")
    parser.add_argument(
        "--scope", choices=("all", "stage1", "proof", "proof-manifest"), default="all"
    )
    parser.add_argument("--stage1-days", type=int, default=30)
    parser.add_argument("--pilot-id", default="")
    return parser


def disable_core_dumps() -> None:
    try:
        resource.setrlimit(resource.RLIMIT_CORE, (0, 0))
    except (OSError, ValueError) as error:
        raise StatusError("core_dump_hardening_unavailable") from error


def admin_key(environ: Mapping[str, str]) -> str:
    value = environ.get(ADMIN_KEY_ENV, "")
    if not isinstance(value, str):
        raise StatusError("admin_key_unavailable")
    try:
        encoded = value.encode("ascii", "strict")
    except UnicodeEncodeError as error:
        raise StatusError("admin_key_unavailable") from error
    if not 8 <= len(encoded) <= 4096 or any(byte <= 0x20 or byte == 0x7F for byte in encoded):
        raise StatusError("admin_key_unavailable")
    return value


def build_opener():
    return urllib.request.build_opener(
        urllib.request.ProxyHandler({}),
        NoRedirect(),
        urllib.request.HTTPSHandler(context=ssl.create_default_context()),
    )


def build_request(path: str, key: str) -> urllib.request.Request:
    if not path.startswith("/api/v1/admin/") or "//" in path:
        raise StatusError("invalid_status_path")
    return urllib.request.Request(
        BASE_URL + path,
        method="GET",
        headers={
            "Accept": "application/json",
            "Authorization": "Bearer " + key,
            "User-Agent": "NHS-Provider-Pilot-Status/1.0",
        },
    )


def perform_get(opener, request: urllib.request.Request) -> dict[str, object]:
    status = None
    try:
        with opener.open(request, timeout=REQUEST_TIMEOUT_SECONDS) as response:
            status = int(response.getcode())
            if 300 <= status <= 399:
                raise StatusError("redirect_refused", http_status=status)
            if status != 200:
                raise StatusError("http_error", http_status=status)
            body = response.read(MAX_RESPONSE_BYTES + 1)
    except urllib.error.HTTPError as error:
        safe_status = error.code if isinstance(error.code, int) and 100 <= error.code <= 599 else None
        code = "redirect_refused" if safe_status is not None and 300 <= safe_status <= 399 else "http_error"
        raise StatusError(code, http_status=safe_status) from error
    except (urllib.error.URLError, TimeoutError, OSError, ssl.SSLError) as error:
        raise StatusError("network_error") from error

    if len(body) > MAX_RESPONSE_BYTES:
        raise StatusError("response_too_large", http_status=status)
    try:
        document = json.loads(body.decode("utf-8", "strict"))
    except (UnicodeDecodeError, json.JSONDecodeError) as error:
        raise StatusError("invalid_response", http_status=status) from error
    if not isinstance(document, dict):
        raise StatusError("invalid_response", http_status=status)
    return document


def _integer(document: Mapping[str, object], key: str, low: int = 0, high: int = MAX_COUNT) -> int:
    value = document.get(key)
    if isinstance(value, bool) or not isinstance(value, int) or not low <= value <= high:
        raise StatusError("invalid_response")
    return value


def _boolean(document: Mapping[str, object], key: str) -> bool:
    value = document.get(key)
    if not isinstance(value, bool):
        raise StatusError("invalid_response")
    return value


def _timestamp(document: Mapping[str, object], key: str) -> str:
    value = document.get(key)
    if not isinstance(value, str) or len(value) > 64:
        raise StatusError("invalid_response")
    try:
        parsed = datetime.datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError as error:
        raise StatusError("invalid_response") from error
    if parsed.tzinfo is None or parsed.utcoffset() is None:
        raise StatusError("invalid_response")
    return value


def _parsed_timestamp(value: str) -> datetime.datetime:
    return datetime.datetime.fromisoformat(value.replace("Z", "+00:00"))


def _integer_map(document: object, expected_keys: set[str]) -> dict[str, int]:
    if not isinstance(document, dict) or set(document) != expected_keys:
        raise StatusError("invalid_response")
    return {key: _integer(document, key) for key in sorted(expected_keys)}


def _boolean_map(document: object, expected_keys: set[str]) -> dict[str, bool]:
    if not isinstance(document, dict) or set(document) != expected_keys:
        raise StatusError("invalid_response")
    return {key: _boolean(document, key) for key in sorted(expected_keys)}


def _buckets(value: object, allowed: frozenset[str], minimum_count: int) -> list[dict[str, object]]:
    if not isinstance(value, list) or len(value) > len(allowed):
        raise StatusError("invalid_response")
    projected: list[dict[str, object]] = []
    seen: set[str] = set()
    for item in value:
        if not isinstance(item, dict) or set(item) != {"value", "receipt_count"}:
            raise StatusError("invalid_response")
        name = item.get("value")
        count = item.get("receipt_count")
        if (
            not isinstance(name, str)
            or name not in allowed
            or name in seen
            or isinstance(count, bool)
            or not isinstance(count, int)
            or not minimum_count <= count <= MAX_COUNT
        ):
            raise StatusError("invalid_response")
        seen.add(name)
        projected.append({"value": name, "receipt_count": count})
    return projected


def project_stage1(document: Mapping[str, object], requested_days: int) -> dict[str, object]:
    raw = document.get("stage1_demand")
    if not isinstance(raw, dict):
        raise StatusError("invalid_response")
    days = _integer(raw, "days", 1, 30)
    if days != requested_days or _integer(raw, "retention_days", 1, 30) != 30:
        raise StatusError("invalid_response")
    as_of = _timestamp(raw, "as_of")
    stage1_started_at = _timestamp(raw, "stage1_started_at")
    if _parsed_timestamp(stage1_started_at) > _parsed_timestamp(as_of):
        raise StatusError("invalid_response")
    if (
        _boolean(raw, "stage1_epoch_enforced") is not True
        or
        _boolean(raw, "synthetic_excluded") is not True
        or _boolean(raw, "counts_are_receipts_not_unique_agents") is not True
        or _boolean(raw, "commercial_proof") is not False
        or _boolean(raw, "topic_buckets_may_overlap") is not True
    ):
        raise StatusError("invalid_response")
    eligible_surfaces = raw.get("eligible_surfaces")
    if eligible_surfaces != ["mcp", "rest"]:
        raise StatusError("invalid_response")
    bucket_threshold = _integer(raw, "bucket_receipt_threshold", 1, MAX_COUNT)
    if bucket_threshold != 20:
        raise StatusError("invalid_response")

    meaningful = _integer(raw, "meaningful_search_receipts")
    selections = _integer(raw, "result_selections")
    selected_receipts = _integer(raw, "search_receipts_with_selection")
    interests = _integer(raw, "action_interest_receipts")
    interest_receipts = _integer(raw, "search_receipts_with_action_interest")
    distinct_domains = _integer(raw, "distinct_interest_domains")
    if (
        selections < selected_receipts
        or selected_receipts > meaningful
        or interests < interest_receipts
        or interest_receipts > meaningful
        or interests < distinct_domains
    ):
        raise StatusError("invalid_response")

    observation_days = _integer(raw, "observation_window_days", 1, 30)
    observation_span_seconds = _integer(raw, "observation_span_seconds", 0, days * 24 * 60 * 60)
    observation_span = _integer(raw, "observation_span_days", 0, days)
    if observation_span != observation_span_seconds // (24 * 60 * 60):
        raise StatusError("invalid_response")
    if observation_days != 14:
        raise StatusError("invalid_response")
    observation_met = _boolean(raw, "observation_window_met")
    if observation_met != (observation_span_seconds >= observation_days * 24 * 60 * 60):
        raise StatusError("invalid_response")

    demand_topics = _buckets(raw.get("demand_topics"), ALLOWED_TOPICS, bucket_threshold)
    candidate_topics = _buckets(
        raw.get("pilot_candidate_topics"), ALLOWED_TOPICS - {"other"},
        STAGE1_TARGETS["pilot_candidate_topic_receipts"],
    )
    demand_by_name = {item["value"]: item["receipt_count"] for item in demand_topics}
    if any(demand_by_name.get(item["value"]) != item["receipt_count"] for item in candidate_topics):
        raise StatusError("invalid_response")
    if any(item["receipt_count"] > meaningful for item in demand_topics):
        raise StatusError("invalid_response")
    candidate_available = _boolean(raw, "pilot_candidate_topic_available")
    if candidate_available != bool(candidate_topics):
        raise StatusError("invalid_response")
    action_types = _buckets(raw.get("action_types"), ALLOWED_ACTION_TYPES, bucket_threshold)
    if any(item["receipt_count"] > interest_receipts for item in action_types):
        raise StatusError("invalid_response")

    targets = _integer_map(raw.get("targets"), set(STAGE1_TARGETS))
    if targets != STAGE1_TARGETS:
        raise StatusError("invalid_response")
    expected_met = {
        "meaningful_search_receipts": meaningful >= STAGE1_TARGETS["meaningful_search_receipts"],
        "search_receipts_with_selection": selected_receipts >= STAGE1_TARGETS["search_receipts_with_selection"],
        "search_receipts_with_action_interest": interest_receipts >= STAGE1_TARGETS["search_receipts_with_action_interest"],
        "pilot_candidate_topic_receipts": candidate_available,
        "observation_window_days": observation_met,
    }
    targets_met = _boolean_map(raw.get("targets_met"), set(STAGE1_TARGETS))
    ready = _boolean(raw, "stage1_ready")
    if targets_met != expected_met or ready != all(expected_met.values()):
        raise StatusError("invalid_response")

    attempt_funnel = project_action_interest_attempt_funnel(document, requested_days)

    return {
        "days": days,
        "as_of": as_of,
        "stage1_started_at": stage1_started_at,
        "stage1_epoch_enforced": True,
        "eligible_surfaces": ["mcp", "rest"],
        "meaningful_search_receipts": meaningful,
        "result_selections": selections,
        "search_receipts_with_selection": selected_receipts,
        "action_interest_receipts": interests,
        "search_receipts_with_action_interest": interest_receipts,
        "distinct_interest_domains": distinct_domains,
        "demand_topics": demand_topics,
        "pilot_candidate_topics": candidate_topics,
        "pilot_candidate_topic_available": candidate_available,
        "action_types": action_types,
        "observation_window_days": observation_days,
        "observation_span_seconds": observation_span_seconds,
        "observation_span_days": observation_span,
        "observation_window_met": observation_met,
        "targets": targets,
        "targets_met": targets_met,
        "stage1_ready": ready,
        "synthetic_excluded": True,
        "counts_are_receipts_not_unique_agents": True,
        "commercial_proof": False,
        "readiness_does_not_authorize_stage2": True,
        "action_interest_attempt_funnel": attempt_funnel,
    }


def project_action_interest_attempt_funnel(
    document: Mapping[str, object], requested_days: int
) -> dict[str, object]:
    raw = document.get("action_interest_attempt_funnel")
    if not isinstance(raw, dict):
        raise StatusError("invalid_response")
    if _integer(raw, "days", 1, 30) != requested_days:
        raise StatusError("invalid_response")
    as_of = _timestamp(raw, "as_of")
    if (
        _boolean(raw, "counts_are_attempts_not_unique_agents") is not True
        or _boolean(raw, "contains_request_coordinates") is not False
        or _boolean(raw, "commercial_proof") is not False
    ):
        raise StatusError("invalid_response")
    total = _integer(raw, "total_attempts", 0, MAX_COUNT)
    outcomes = raw.get("outcomes")
    if not isinstance(outcomes, list) or len(outcomes) > (
        len(ALLOWED_ATTEMPT_SURFACES) * len(ALLOWED_ATTEMPT_OUTCOMES)
    ):
        raise StatusError("invalid_response")
    projected: list[dict[str, object]] = []
    seen: set[tuple[str, str]] = set()
    computed_total = 0
    for item in outcomes:
        if not isinstance(item, dict) or set(item) != {"surface", "outcome", "attempt_count"}:
            raise StatusError("invalid_response")
        surface = item.get("surface")
        outcome = item.get("outcome")
        count = item.get("attempt_count")
        coordinate = (surface, outcome)
        if (
            not isinstance(surface, str)
            or surface not in ALLOWED_ATTEMPT_SURFACES
            or not isinstance(outcome, str)
            or outcome not in ALLOWED_ATTEMPT_OUTCOMES
            or coordinate in seen
            or isinstance(count, bool)
            or not isinstance(count, int)
            or not 1 <= count <= MAX_COUNT
        ):
            raise StatusError("invalid_response")
        seen.add(coordinate)
        computed_total += count
        projected.append({"surface": surface, "outcome": outcome, "attempt_count": count})
    if computed_total != total:
        raise StatusError("invalid_response")
    return {
        "days": requested_days,
        "as_of": as_of,
        "counts_are_attempts_not_unique_agents": True,
        "contains_request_coordinates": False,
        "commercial_proof": False,
        "total_attempts": total,
        "outcomes": projected,
        "operational_diagnostic_only": True,
    }


def _money_map(value: object) -> dict[str, int]:
    if not isinstance(value, dict) or len(value) > 16:
        raise StatusError("invalid_response")
    projected: dict[str, int] = {}
    for currency, amount in value.items():
        if (
            not isinstance(currency, str)
            or not _CURRENCY_PATTERN.fullmatch(currency)
            or isinstance(amount, bool)
            or not isinstance(amount, int)
            or not 0 <= amount <= MAX_MONEY_CENTS
        ):
            raise StatusError("invalid_response")
        projected[currency] = amount
    return dict(sorted(projected.items()))


_MANIFEST_AGGREGATE_FIELDS = frozenset(
    {
        "manifest_contract_version",
        "signature_verification_scope",
        "provider_pilot_epoch_id",
        "provider_pilot_contract_version",
        "review_contract_version",
        "review_evidence_contract_version",
        "market_policy_contract_version",
        "proof_snapshot_sha256",
        "review_evidence_sha256",
        "pilot_demand_topic",
        "pilot_status",
        "outcome_receipt_integrity_valid",
        "review_integrity_valid",
        "verified_outcome_receipts",
        "rejected_outcome_receipts",
        "verified_outcome_ledger_entries",
        "rejected_outcome_ledger_entries",
        "verified_provider_companies",
        "verified_provider_accepted_handoffs",
        "verified_provider_confirmed_activations",
        "verified_provider_renewals",
        "verified_provider_confirmed_conversions",
        "review_coverage",
        "monetary_amounts_withheld_for_privacy",
        "verified_prepaid_settled",
        "verified_prepaid_net_debited",
        "verified_terms_net_receivable",
        "pilot_thresholds_met",
        "organic_rank_sold",
        "raw_queries_sold",
        "agent_identities_sold",
        "evidence_scope",
    }
)
_MANIFEST_CANDIDATE_FIELDS = _MANIFEST_AGGREGATE_FIELDS | {
    "issuable",
    "issuance_blockers",
}
_SIGNED_MANIFEST_FIELDS = _MANIFEST_AGGREGATE_FIELDS | {"v", "kid", "manifest_id", "issued_at"}
_MANIFEST_RECORD_FIELDS = frozenset(
    {
        "id",
        "provider_pilot_epoch_id",
        "manifest_contract_version",
        "proof_snapshot_sha256",
        "review_evidence_sha256",
        "key_id",
        "signed_manifest",
        "signature",
        "payload_sha256",
        "issued_at",
    }
)
_REVIEW_COVERAGE_KEYS = frozenset({"providers", "offers", "tickets", "handoffs", "callbacks"})
_ISSUANCE_BLOCKERS = frozenset(
    {
        "pilot_not_closed",
        "outcome_integrity_failed",
        "commercial_thresholds_not_met",
        "chronological_review_incomplete",
    }
)


def _manifest_currency_amounts(value: object) -> list[dict[str, object]]:
    if not isinstance(value, list) or len(value) > 16:
        raise StatusError("invalid_response")
    result: list[dict[str, object]] = []
    previous = ""
    for item in value:
        if not isinstance(item, dict) or set(item) != {"currency", "amount_minor"}:
            raise StatusError("invalid_response")
        currency = item.get("currency")
        amount = item.get("amount_minor")
        if (
            not isinstance(currency, str)
            or not _CURRENCY_PATTERN.fullmatch(currency)
            or currency <= previous
            or isinstance(amount, bool)
            or not isinstance(amount, int)
            or not -(2**63) <= amount <= MAX_MONEY_CENTS
            or amount == 0
        ):
            raise StatusError("invalid_response")
        previous = currency
        result.append({"currency": currency, "amount_minor": amount})
    return result


def _manifest_review_coverage(value: object) -> tuple[dict[str, dict[str, int]], bool]:
    if not isinstance(value, dict) or set(value) != _REVIEW_COVERAGE_KEYS:
        raise StatusError("invalid_response")
    result: dict[str, dict[str, int]] = {}
    for category in sorted(_REVIEW_COVERAGE_KEYS):
        count = value.get(category)
        if not isinstance(count, dict) or set(count) != {"required", "valid"}:
            raise StatusError("invalid_response")
        required = _integer(count, "required")
        valid = _integer(count, "valid")
        if valid > required:
            raise StatusError("invalid_response")
        result[category] = {"required": required, "valid": valid}
    complete = all(item["required"] > 0 and item["valid"] == item["required"] for item in result.values())
    return result, complete


def _project_manifest_aggregate(
    value: Mapping[str, object], expected_pilot_id: str
) -> dict[str, object]:
    if (
        value.get("manifest_contract_version") != PROOF_MANIFEST_CONTRACT_VERSION
        or value.get("signature_verification_scope") != PROOF_MANIFEST_VERIFICATION_SCOPE
        or value.get("provider_pilot_epoch_id") != expected_pilot_id
        or value.get("provider_pilot_contract_version") != "nhs-provider-pilot-v1"
        or value.get("review_contract_version") != "nhs-provider-pilot-review-v1"
        or value.get("review_evidence_contract_version")
        != PROOF_MANIFEST_REVIEW_EVIDENCE_VERSION
        or value.get("market_policy_contract_version")
        != PROOF_MANIFEST_MARKET_POLICY_VERSION
        or not isinstance(value.get("proof_snapshot_sha256"), str)
        or not _HASH_PATTERN.fullmatch(value["proof_snapshot_sha256"])
        or not isinstance(value.get("review_evidence_sha256"), str)
        or not _HASH_PATTERN.fullmatch(value["review_evidence_sha256"])
        or value.get("pilot_demand_topic") not in ALLOWED_TOPICS - {"other"}
        or value.get("pilot_status") not in {"draft", "active", "closed"}
        or value.get("evidence_scope") != PROOF_MANIFEST_SCOPE
    ):
        raise StatusError("invalid_response")

    outcome_integrity = _boolean(value, "outcome_receipt_integrity_valid")
    review_integrity = _boolean(value, "review_integrity_valid")
    verified_outcomes = _integer(value, "verified_outcome_receipts")
    rejected_outcomes = _integer(value, "rejected_outcome_receipts")
    verified_ledger = _integer(value, "verified_outcome_ledger_entries")
    rejected_ledger = _integer(value, "rejected_outcome_ledger_entries")
    companies = _integer(value, "verified_provider_companies")
    handoffs = _integer(value, "verified_provider_accepted_handoffs")
    activations = _integer(value, "verified_provider_confirmed_activations")
    renewals = _integer(value, "verified_provider_renewals")
    conversions = _integer(value, "verified_provider_confirmed_conversions")
    if (
        outcome_integrity != (rejected_outcomes == 0 and rejected_ledger == 0)
        or renewals > companies
        or handoffs > verified_outcomes
        or activations > handoffs
        or conversions > activations
        or verified_ledger > verified_outcomes
    ):
        raise StatusError("invalid_response")

    threshold_met = (
        outcome_integrity
        and companies >= PROOF_TARGETS["verified_provider_companies"]
        and handoffs >= PROOF_TARGETS["verified_provider_accepted_handoffs"]
        and activations >= PROOF_TARGETS["verified_provider_confirmed_activations"]
        and renewals >= PROOF_TARGETS["verified_provider_renewals"]
    )
    if _boolean(value, "pilot_thresholds_met") != threshold_met:
        raise StatusError("invalid_response")

    coverage, coverage_complete = _manifest_review_coverage(value.get("review_coverage"))
    coverage_consistent = (
        coverage["providers"]["required"] == companies
        and coverage["offers"]["required"] >= companies
        and coverage["tickets"]["required"] >= handoffs
        and coverage["handoffs"]["required"] == coverage["tickets"]["required"]
        and coverage["callbacks"]["required"] == verified_outcomes
    )
    if review_integrity != (coverage_complete and coverage_consistent):
        raise StatusError("invalid_response")
    for field in ("organic_rank_sold", "raw_queries_sold", "agent_identities_sold"):
        if _boolean(value, field) is not False:
            raise StatusError("invalid_response")
    if _boolean(value, "monetary_amounts_withheld_for_privacy") is not True or any(
        value.get(field) != []
        for field in (
            "verified_prepaid_settled",
            "verified_prepaid_net_debited",
            "verified_terms_net_receivable",
        )
    ):
        raise StatusError("invalid_response")

    return {
        "manifest_contract_version": PROOF_MANIFEST_CONTRACT_VERSION,
        "signature_verification_scope": PROOF_MANIFEST_VERIFICATION_SCOPE,
        "provider_pilot_epoch_id": expected_pilot_id,
        "provider_pilot_contract_version": "nhs-provider-pilot-v1",
        "review_contract_version": "nhs-provider-pilot-review-v1",
        "review_evidence_contract_version": PROOF_MANIFEST_REVIEW_EVIDENCE_VERSION,
        "market_policy_contract_version": PROOF_MANIFEST_MARKET_POLICY_VERSION,
        "proof_snapshot_sha256": value["proof_snapshot_sha256"],
        "review_evidence_sha256": value["review_evidence_sha256"],
        "pilot_demand_topic": value["pilot_demand_topic"],
        "pilot_status": value["pilot_status"],
        "outcome_receipt_integrity_valid": outcome_integrity,
        "review_integrity_valid": review_integrity,
        "verified_outcome_receipts": verified_outcomes,
        "rejected_outcome_receipts": rejected_outcomes,
        "verified_outcome_ledger_entries": verified_ledger,
        "rejected_outcome_ledger_entries": rejected_ledger,
        "verified_provider_companies": companies,
        "verified_provider_accepted_handoffs": handoffs,
        "verified_provider_confirmed_activations": activations,
        "verified_provider_renewals": renewals,
        "verified_provider_confirmed_conversions": conversions,
        "review_coverage": coverage,
        "monetary_amounts_withheld_for_privacy": True,
        "verified_prepaid_settled": [],
        "verified_prepaid_net_debited": [],
        "verified_terms_net_receivable": [],
        "pilot_thresholds_met": threshold_met,
        "organic_rank_sold": False,
        "raw_queries_sold": False,
        "agent_identities_sold": False,
        "evidence_scope": PROOF_MANIFEST_SCOPE,
    }


def _project_manifest_candidate(value: object, expected_pilot_id: str) -> dict[str, object]:
    if not isinstance(value, dict) or set(value) != _MANIFEST_CANDIDATE_FIELDS:
        raise StatusError("invalid_response")
    result = _project_manifest_aggregate(value, expected_pilot_id)
    blockers = value.get("issuance_blockers")
    if (
        not isinstance(blockers, list)
        or len(blockers) > len(_ISSUANCE_BLOCKERS)
        or any(not isinstance(item, str) or item not in _ISSUANCE_BLOCKERS for item in blockers)
        or len(set(blockers)) != len(blockers)
    ):
        raise StatusError("invalid_response")
    expected_blockers = []
    if result["pilot_status"] != "closed":
        expected_blockers.append("pilot_not_closed")
    if not result["outcome_receipt_integrity_valid"]:
        expected_blockers.append("outcome_integrity_failed")
    if not result["pilot_thresholds_met"]:
        expected_blockers.append("commercial_thresholds_not_met")
    if not result["review_integrity_valid"]:
        expected_blockers.append("chronological_review_incomplete")
    issuable = _boolean(value, "issuable")
    if blockers != expected_blockers or issuable != (not expected_blockers):
        raise StatusError("invalid_response")
    result["issuable"] = issuable
    result["issuance_blockers"] = list(blockers)
    return result


def _project_manifest_record(value: object, expected_pilot_id: str) -> dict[str, object]:
    if not isinstance(value, dict) or set(value) != _MANIFEST_RECORD_FIELDS:
        raise StatusError("invalid_response")
    manifest_id = value.get("id")
    key_id = value.get("key_id")
    signed_manifest = value.get("signed_manifest")
    signature = value.get("signature")
    payload_sha256 = value.get("payload_sha256")
    review_evidence_sha256 = value.get("review_evidence_sha256")
    if (
        not isinstance(manifest_id, str)
        or not _UUID_PATTERN.fullmatch(manifest_id)
        or value.get("provider_pilot_epoch_id") != expected_pilot_id
        or value.get("manifest_contract_version") != PROOF_MANIFEST_CONTRACT_VERSION
        or not isinstance(value.get("proof_snapshot_sha256"), str)
        or not _HASH_PATTERN.fullmatch(value["proof_snapshot_sha256"])
        or not isinstance(review_evidence_sha256, str)
        or not _HASH_PATTERN.fullmatch(review_evidence_sha256)
        or not isinstance(key_id, str)
        or not _KEY_ID_PATTERN.fullmatch(key_id)
        or not isinstance(signed_manifest, str)
        or not 1 <= len(signed_manifest) <= 16 * 1024
        or not isinstance(signature, str)
        or not _SIGNATURE_PATTERN.fullmatch(signature)
        or not isinstance(payload_sha256, str)
        or not _HASH_PATTERN.fullmatch(payload_sha256)
        or hashlib.sha256(signed_manifest.encode("utf-8")).hexdigest() != payload_sha256
    ):
        raise StatusError("invalid_response")
    try:
        signed = json.loads(signed_manifest)
    except json.JSONDecodeError as error:
        raise StatusError("invalid_response") from error
    if (
        not isinstance(signed, dict)
        or set(signed) != _SIGNED_MANIFEST_FIELDS
        or json.dumps(signed, separators=(",", ":"), ensure_ascii=True) != signed_manifest
    ):
        raise StatusError("invalid_response")
    aggregate = _project_manifest_aggregate(signed, expected_pilot_id)
    issued_at = _timestamp(value, "issued_at")
    signed_issued_at = _integer(signed, "issued_at", 1, 2**63 - 1)
    if (
        _integer(signed, "v", 1, 1) != 1
        or signed.get("kid") != key_id
        or signed.get("manifest_id") != manifest_id
        or signed.get("proof_snapshot_sha256") != value["proof_snapshot_sha256"]
        or signed.get("review_evidence_sha256") != review_evidence_sha256
        or aggregate["pilot_status"] != "closed"
        or aggregate["outcome_receipt_integrity_valid"] is not True
        or aggregate["review_integrity_valid"] is not True
        or aggregate["pilot_thresholds_met"] is not True
        or int(_parsed_timestamp(issued_at).timestamp()) != signed_issued_at
    ):
        raise StatusError("invalid_response")
    return {
        "id": manifest_id,
        "provider_pilot_epoch_id": expected_pilot_id,
        "manifest_contract_version": PROOF_MANIFEST_CONTRACT_VERSION,
        "proof_snapshot_sha256": value["proof_snapshot_sha256"],
        "review_evidence_sha256": review_evidence_sha256,
        "key_id": key_id,
        "signed_manifest": signed_manifest,
        "signature": signature,
        "payload_sha256": payload_sha256,
        "issued_at": issued_at,
    }


def project_proof_manifest(
    document: Mapping[str, object], expected_pilot_id: str
) -> dict[str, object]:
    issued = document.get("issued")
    if (
        not isinstance(issued, bool)
        or document.get("publicly_released") is not False
        or document.get("independently_verifiable") is not False
        or document.get("evidence_scope") != PROOF_MANIFEST_EVIDENCE_SCOPE
    ):
        raise StatusError("invalid_response")
    if issued:
        if (
            set(document)
            != {
                "manifest", "issued", "commercial_proof_created", "publicly_released",
                "independently_verifiable", "evidence_scope",
            }
            or document.get("commercial_proof_created") is not True
        ):
            raise StatusError("invalid_response")
        return {
            "issued": True,
            "commercial_proof_created": True,
            "publicly_released": False,
            "independently_verifiable": False,
            "manifest": _project_manifest_record(document.get("manifest"), expected_pilot_id),
            "evidence_scope": PROOF_MANIFEST_EVIDENCE_SCOPE,
        }
    if (
        set(document)
        != {
            "manifest_candidate",
            "issued",
            "commercial_proof_created",
            "publicly_released",
            "independently_verifiable",
            "evidence_scope",
        }
        or document.get("commercial_proof_created") is not False
    ):
        raise StatusError("invalid_response")
    return {
        "issued": False,
        "commercial_proof_created": False,
        "publicly_released": False,
        "independently_verifiable": False,
        "manifest_candidate": _project_manifest_candidate(
            document.get("manifest_candidate"), expected_pilot_id
        ),
        "evidence_scope": PROOF_MANIFEST_EVIDENCE_SCOPE,
    }


def project_proof(document: Mapping[str, object], expected_pilot_id: str) -> dict[str, object]:
    raw = document.get("proof")
    if not isinstance(raw, dict):
        raise StatusError("invalid_response")
    targets = _integer_map(document.get("targets"), set(PROOF_TARGETS))
    if targets != PROOF_TARGETS:
        raise StatusError("invalid_response")
    for key in (
        "organic_rank_sold", "raw_queries_sold", "raw_prompts_sold",
        "agent_identities_sold", "principal_identities_sold",
    ):
        if _boolean(document, key) is not False:
            raise StatusError("invalid_response")

    pilot_id = raw.get("provider_pilot_epoch_id")
    pilot_topic = raw.get("provider_pilot_demand_topic")
    pilot_status = raw.get("provider_pilot_status")
    if (
        pilot_id != expected_pilot_id
        or not isinstance(pilot_topic, str)
        or pilot_topic not in ALLOWED_TOPICS - {"other"}
        or pilot_status not in {"draft", "active", "closed"}
    ):
        raise StatusError("invalid_response")

    verified_companies = _integer(raw, "verified_provider_companies")
    verified_offer_returns = _integer(raw, "verified_provider_offer_returns")
    verified_observed_handoffs = _integer(raw, "verified_observed_handoffs")
    verified_handoffs = _integer(raw, "verified_provider_accepted_handoffs")
    verified_activations = _integer(raw, "verified_provider_confirmed_activations")
    verified_renewals = _integer(raw, "verified_provider_renewals")
    verified_conversions = _integer(raw, "verified_provider_confirmed_conversions")
    accepted_latency_samples = _integer(raw, "verified_accepted_latency_samples")
    activated_latency_samples = _integer(raw, "verified_activated_latency_samples")
    converted_latency_samples = _integer(raw, "verified_converted_latency_samples")
    accepted_median_seconds = _integer(
        raw, "verified_accepted_median_handoff_to_outcome_seconds", 0, MAX_MONEY_CENTS
    )
    activated_median_seconds = _integer(
        raw, "verified_activated_median_handoff_to_outcome_seconds", 0, MAX_MONEY_CENTS
    )
    converted_median_seconds = _integer(
        raw, "verified_converted_median_handoff_to_outcome_seconds", 0, MAX_MONEY_CENTS
    )
    settlement_integrity = _boolean(raw, "settlement_receipt_integrity_valid")
    processor_net_integrity = _boolean(raw, "processor_net_receipt_integrity_valid")
    verified_paid_settlements = _integer(raw, "verified_provider_paid_settlements")
    verified_available_settlements = _integer(raw, "verified_provider_available_settlements")
    rejected_settlements = _integer(raw, "rejected_provider_settlement_receipts")
    rejected_processor_net = _integer(raw, "rejected_provider_processor_net_receipts")
    paid_latency_samples = _integer(raw, "verified_paid_latency_samples")
    paid_median_seconds = _integer(
        raw, "verified_paid_median_handoff_to_settlement_seconds", 0, MAX_MONEY_CENTS
    )
    verified_terms_paid = _money_map(raw.get("verified_terms_paid_by_currency"))
    verified_processor_fees = _money_map(raw.get("verified_processor_fees_by_currency"))
    verified_processor_net = _money_map(raw.get("verified_processor_net_by_currency"))

    mechanism_raw = raw.get("verified_mechanisms")
    if not isinstance(mechanism_raw, dict) or set(mechanism_raw) != set(CHARGE_EVENTS):
        raise StatusError("invalid_response")
    verified_mechanisms: dict[str, dict[str, int]] = {}
    for charge_event in CHARGE_EVENTS:
        item = mechanism_raw.get(charge_event)
        if not isinstance(item, dict) or set(item) != {
            "charged_provider_companies", "offer_returns", "observed_handoffs", "accepted", "activated", "converted", "reversed",
            "paid_settlements", "paid_cents",
            "available_settlements", "processor_fee_cents", "processor_net_cents",
            "paid_median_handoff_to_settlement_seconds",
        }:
            raise StatusError("invalid_response")
        projected = {
            "charged_provider_companies": _integer(item, "charged_provider_companies"),
            "offer_returns": _integer(item, "offer_returns"),
            "observed_handoffs": _integer(item, "observed_handoffs"),
            "accepted": _integer(item, "accepted"),
            "activated": _integer(item, "activated"),
            "converted": _integer(item, "converted"),
            "reversed": _integer(item, "reversed"),
            "paid_settlements": _integer(item, "paid_settlements"),
            "paid_cents": _integer(item, "paid_cents", 0, MAX_MONEY_CENTS),
            "available_settlements": _integer(item, "available_settlements"),
            "processor_fee_cents": _integer(item, "processor_fee_cents", 0, MAX_MONEY_CENTS),
            "processor_net_cents": _integer(item, "processor_net_cents", 0, MAX_MONEY_CENTS),
            "paid_median_handoff_to_settlement_seconds": _integer(
                item, "paid_median_handoff_to_settlement_seconds", 0, MAX_MONEY_CENTS
            ),
        }
        if (
            projected["observed_handoffs"] > projected["offer_returns"]
            or projected["charged_provider_companies"] > projected["observed_handoffs"]
            or projected["charged_provider_companies"] > verified_companies
            or projected["observed_handoffs"] < projected["accepted"]
            or projected["accepted"] < projected["activated"]
            or projected["activated"] < projected["converted"]
            or projected["reversed"] > projected["observed_handoffs"]
            or (projected["paid_settlements"] == 0) != (projected["paid_cents"] == 0)
            or (projected["paid_settlements"] == 0)
            != (projected["paid_median_handoff_to_settlement_seconds"] == 0)
            or projected["available_settlements"] > projected["paid_settlements"]
            or projected["processor_fee_cents"] + projected["processor_net_cents"]
            != (projected["paid_cents"] if projected["available_settlements"] == projected["paid_settlements"] else 0)
        ):
            raise StatusError("invalid_response")
        verified_mechanisms[charge_event] = projected
    outcome_integrity = _boolean(raw, "outcome_receipt_integrity_valid")
    verified_outcomes = _integer(raw, "verified_outcome_receipts")
    rejected_outcomes = _integer(raw, "rejected_outcome_receipts")
    verified_outcome_ledger = _integer(raw, "verified_outcome_ledger_entries")
    rejected_outcome_ledger = _integer(raw, "rejected_outcome_ledger_entries")
    if outcome_integrity != (rejected_outcomes == 0 and rejected_outcome_ledger == 0):
        raise StatusError("invalid_response")
    if settlement_integrity != (rejected_settlements == 0):
        raise StatusError("invalid_response")
    if processor_net_integrity != (rejected_processor_net == 0):
        raise StatusError("invalid_response")
    if (
        verified_handoffs > verified_observed_handoffs
        or verified_activations > verified_handoffs
        or verified_conversions > verified_activations
        or accepted_latency_samples != verified_handoffs
        or activated_latency_samples != verified_activations
        or converted_latency_samples != verified_conversions
        or (verified_handoffs == 0 and accepted_median_seconds != 0)
        or (verified_activations == 0 and activated_median_seconds != 0)
        or (verified_conversions == 0 and converted_median_seconds != 0)
        or paid_latency_samples != verified_paid_settlements
        or verified_available_settlements > verified_paid_settlements
        or (verified_paid_settlements == 0) != (len(verified_terms_paid) == 0)
        or (verified_paid_settlements == 0 and paid_median_seconds != 0)
        or sum(item["observed_handoffs"] for item in verified_mechanisms.values())
        != verified_observed_handoffs
        or sum(item["offer_returns"] for item in verified_mechanisms.values())
        != verified_offer_returns
        or sum(item["accepted"] for item in verified_mechanisms.values()) != verified_handoffs
        or sum(item["activated"] for item in verified_mechanisms.values()) != verified_activations
        or sum(item["converted"] for item in verified_mechanisms.values()) != verified_conversions
        or sum(item["paid_settlements"] for item in verified_mechanisms.values())
        != verified_paid_settlements
        or sum(item["paid_cents"] for item in verified_mechanisms.values())
        != verified_terms_paid.get("usd", 0)
        or sum(item["available_settlements"] for item in verified_mechanisms.values())
        != verified_available_settlements
        or sum(item["processor_fee_cents"] for item in verified_mechanisms.values())
        != verified_processor_fees.get("usd", 0)
        or sum(item["processor_net_cents"] for item in verified_mechanisms.values())
        != verified_processor_net.get("usd", 0)
    ):
        raise StatusError("invalid_response")
    expected_met = (
        outcome_integrity
        and settlement_integrity
        and processor_net_integrity
        and verified_available_settlements == verified_paid_settlements
        and verified_companies >= PROOF_TARGETS["verified_provider_companies"]
        and verified_handoffs >= PROOF_TARGETS["verified_provider_accepted_handoffs"]
        and verified_activations >= PROOF_TARGETS["verified_provider_confirmed_activations"]
        and verified_renewals >= PROOF_TARGETS["verified_provider_renewals"]
    )
    if _boolean(raw, "pilot_thresholds_met") != expected_met:
        raise StatusError("invalid_response")

    diagnostics = {
        "operator_recorded_provider_budgets": _integer(raw, "operator_recorded_provider_budgets"),
        "provider_reported_accepted_handoffs": _integer(raw, "provider_reported_accepted_handoffs"),
        "provider_reported_activations": _integer(raw, "provider_reported_activations"),
        "renewed_provider_budgets": _integer(raw, "renewed_provider_budgets"),
        "provider_reported_conversions": _integer(raw, "provider_reported_conversions"),
        "prepaid_net_debited_by_currency": _money_map(raw.get("prepaid_net_debited_by_currency")),
        "terms_net_receivable_by_currency": _money_map(raw.get("terms_net_receivable_by_currency")),
        "operator_recorded_collected_by_currency": _money_map(raw.get("operator_recorded_collected_by_currency")),
    }
    return {
        "provider_pilot_epoch_id": pilot_id,
        "provider_pilot_demand_topic": pilot_topic,
        "provider_pilot_status": pilot_status,
        "outcome_receipt_integrity_valid": outcome_integrity,
        "verified_outcome_receipts": verified_outcomes,
        "rejected_outcome_receipts": rejected_outcomes,
        "verified_outcome_ledger_entries": verified_outcome_ledger,
        "rejected_outcome_ledger_entries": rejected_outcome_ledger,
        "verified_provider_companies": verified_companies,
        "verified_provider_offer_returns": verified_offer_returns,
        "verified_observed_handoffs": verified_observed_handoffs,
        "verified_provider_accepted_handoffs": verified_handoffs,
        "verified_provider_confirmed_activations": verified_activations,
        "verified_provider_renewals": verified_renewals,
        "verified_provider_confirmed_conversions": verified_conversions,
        "verified_accepted_latency_samples": accepted_latency_samples,
        "verified_activated_latency_samples": activated_latency_samples,
        "verified_converted_latency_samples": converted_latency_samples,
        "verified_accepted_median_handoff_to_outcome_seconds": accepted_median_seconds,
        "verified_activated_median_handoff_to_outcome_seconds": activated_median_seconds,
        "verified_converted_median_handoff_to_outcome_seconds": converted_median_seconds,
        "settlement_receipt_integrity_valid": settlement_integrity,
        "processor_net_receipt_integrity_valid": processor_net_integrity,
        "verified_provider_paid_settlements": verified_paid_settlements,
        "verified_provider_available_settlements": verified_available_settlements,
        "rejected_provider_settlement_receipts": rejected_settlements,
        "rejected_provider_processor_net_receipts": rejected_processor_net,
        "verified_paid_latency_samples": paid_latency_samples,
        "verified_paid_median_handoff_to_settlement_seconds": paid_median_seconds,
        "verified_terms_paid_by_currency": verified_terms_paid,
        "verified_processor_fees_by_currency": verified_processor_fees,
        "verified_processor_net_by_currency": verified_processor_net,
        "verified_mechanisms": verified_mechanisms,
        "verified_prepaid_settled_by_currency": _money_map(raw.get("verified_prepaid_settled_by_currency")),
        "verified_prepaid_net_debited_by_currency": _money_map(raw.get("verified_prepaid_net_debited_by_currency")),
        "verified_terms_net_receivable_by_currency": _money_map(raw.get("verified_terms_net_receivable_by_currency")),
        "pilot_thresholds_met": expected_met,
        "targets": targets,
        "diagnostic_observations_not_proof": diagnostics,
        "organic_rank_sold": False,
        "raw_queries_sold": False,
        "raw_prompts_sold": False,
        "agent_identities_sold": False,
        "principal_identities_sold": False,
    }


def run(
    argv: Sequence[str],
    *,
    environ: Mapping[str, str] | None = None,
    opener=None,
) -> dict[str, object]:
    args = build_parser().parse_args(list(argv))
    if not 15 <= args.stage1_days <= 30:
        raise StatusError("invalid_arguments")
    pilot_id = args.pilot_id.strip().lower()
    if args.pilot_id != pilot_id or (
        args.scope in ("all", "proof", "proof-manifest")
        and not _UUID_PATTERN.fullmatch(pilot_id)
    ):
        raise StatusError("invalid_arguments")
    environment = os.environ if environ is None else environ
    key = admin_key(environment)
    client = build_opener() if opener is None else opener
    result: dict[str, object] = {"ok": True, "scope": args.scope}
    if args.scope in ("all", "stage1"):
        query = urllib.parse.urlencode({"days": str(args.stage1_days)})
        document = perform_get(client, build_request("/api/v1/admin/demand-stage1?" + query, key))
        result["stage1"] = project_stage1(document, args.stage1_days)
    if args.scope in ("all", "proof"):
        query = urllib.parse.urlencode({"pilot_id": pilot_id})
        document = perform_get(
            client, build_request("/api/v1/admin/provider-proof?" + query, key)
        )
        result["commercial_proof"] = project_proof(document, pilot_id)
    if args.scope in ("all", "proof-manifest"):
        query = urllib.parse.urlencode({"pilot_id": pilot_id})
        document = perform_get(
            client, build_request("/api/v1/admin/provider-proof-manifest?" + query, key)
        )
        result["commercial_proof_manifest"] = project_proof_manifest(document, pilot_id)
    return result


def _emit(document: Mapping[str, object], stream) -> None:
    stream.write(json.dumps(dict(document), sort_keys=True, separators=(",", ":")) + "\n")


def main(argv: Sequence[str] | None = None) -> int:
    try:
        disable_core_dumps()
        receipt = run(sys.argv[1:] if argv is None else argv)
    except StatusError as error:
        document: dict[str, object] = {"ok": False, "error": error.code}
        if error.http_status is not None:
            document["http_status"] = error.http_status
        _emit(document, sys.stderr)
        return 1
    except KeyboardInterrupt:
        _emit({"ok": False, "error": "interrupted"}, sys.stderr)
        return 1
    except Exception:
        _emit({"ok": False, "error": "internal_error"}, sys.stderr)
        return 1
    _emit(receipt, sys.stdout)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
