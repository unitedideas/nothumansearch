#!/usr/bin/env python3
"""Safely inspect and operate the bounded NHS provider pilot.

The production host and endpoint set are fixed. The admin key is accepted only
from ``NHS_PROVIDER_OPERATOR_ADMIN_KEY``. This client performs one TLS-verified,
no-proxy, no-redirect, no-retry request and emits only a reviewed projection of
the response. Mutations require an explicit owner-authorization flag.
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
QUEUE_PATH = "/api/v1/admin/provider-pilot-queue"
OFFER_ACTION_PATH = "/api/v1/admin/provider-offers/action"
COMMERCIAL_ACTION_PATH = "/api/v1/admin/provider-commercial/action"
PILOT_ACTION_PATH = "/api/v1/admin/provider-pilot/action"
PILOT_EPOCH_PATH = "/api/v1/admin/provider-pilot/epoch"
PILOT_REVIEW_PATH = "/api/v1/admin/provider-pilot-review"
PROOF_MANIFEST_PATH = "/api/v1/admin/provider-proof-manifest"
ADMIN_KEY_ENV = "NHS_PROVIDER_OPERATOR_ADMIN_KEY"
MAX_RESPONSE_BYTES = 64 * 1024
REQUEST_TIMEOUT_SECONDS = 10

PILOT_CONTRACT_VERSION = "nhs-provider-pilot-v1"
PILOT_MUTATION_EVIDENCE_SCOPE = (
    "Owner-authorized pilot configuration only. This response is not provider funding, a handoff, "
    "an activation outcome, a renewal, or 3/5/2/1 proof."
)
PILOT_STATUS_EVIDENCE_SCOPE = (
    "Exact epoch status and bounded counts only; no query, contact, principal, agent, funding, "
    "outcome, or revenue claim."
)
PILOT_REVIEW_CONTRACT_VERSION = "nhs-provider-pilot-review-v1"
PILOT_REVIEW_EVIDENCE_SCOPE = (
    "Owner-only review of one exact privacy-bounded pilot snapshot. The candidate and receipt "
    "contain no search receipt, query, bearer, token hash, company-deduplication hash, principal "
    "or agent identity/contact/network metadata, raw signed-receipt body/signature, or free-form "
    "intent. Recording a review does not create a provider acceptance, handoff, outcome, renewal, "
    "revenue, or 3/5/2/1 proof."
)
PILOT_REVIEW_TYPES = frozenset({"provider", "offer", "ticket", "handoff", "callback"})
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
PILOT_TOPICS = frozenset(
    {
        "payments", "commerce", "jobs", "data", "search", "weather", "maps", "email",
        "messaging", "image", "video", "audio", "documents", "security", "finance", "health",
        "education", "news", "analytics", "automation", "productivity", "identity", "storage",
        "ai-tools", "developer-tools",
    }
)
PILOT_STATUSES = frozenset({"draft", "active", "closed"})
PILOT_MINIMUM_COHORT = 3
PILOT_MAXIMUM_COHORT = 20
PILOT_MAXIMUM_PROVIDER_TICKETS = 100
PILOT_MINIMUM_TOTAL_TICKETS = 5
PILOT_MAXIMUM_TOTAL_TICKETS = 2000

QUEUE_STATES = frozenset(
    {
        "all",
        "review_required",
        "pre_event_review_required",
        "provider_review_required",
        "offer_review_required",
        "ticket_review_required",
        "handoff_review_required",
        "callback_review_required",
        "pending_company",
        "pending_terms",
        "activation_review",
        "expiring_terms",
        "handoff_awaiting_callback",
        "recent_callback",
    }
)
_UUID_PATTERN = re.compile(
    r"^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"
)
_REFERENCE_PATTERN = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._:/-]{7,199}$")
_SOURCE_SYSTEM_PATTERN = re.compile(r"^[a-z0-9][a-z0-9._:-]{1,99}$")
_HASH_PATTERN = re.compile(r"^[0-9a-f]{64}$")
_KEY_ID_PATTERN = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{2,63}$")
_SIGNATURE_PATTERN = re.compile(r"^[A-Za-z0-9_-]{43}$")
_DOMAIN_PATTERN = re.compile(
    r"^(?=.{1,253}$)(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$"
)


class OperatorError(Exception):
    """Stable public error code that never embeds request or response data."""

    def __init__(self, code: str, *, http_status: int | None = None):
        super().__init__(code)
        self.code = code
        self.http_status = http_status


class SafeArgumentParser(argparse.ArgumentParser):
    def error(self, _message: str) -> None:
        raise OperatorError("invalid_arguments")


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):  # noqa: ANN001
        return None


def build_parser() -> argparse.ArgumentParser:
    parser = SafeArgumentParser(description="Safely inspect or operate the NHS provider pilot.")
    subparsers = parser.add_subparsers(dest="command", required=True)

    queue = subparsers.add_parser("queue", add_help=True)
    queue.add_argument("--state", choices=sorted(QUEUE_STATES), default="all")
    queue.add_argument("--limit", type=int, default=25)

    review_preflight = subparsers.add_parser(
        "review-preflight",
        help="Read every bounded pre-event review blocker without mutating pilot state.",
    )
    review_preflight.add_argument("--limit", type=int, default=25)

    verify = subparsers.add_parser("verify-terms", add_help=True)
    verify.add_argument("--offer-id", required=True)
    verify.add_argument("--provider-acceptance-event-id", required=True)
    verify.add_argument("--related-commitment-event-id", default="")
    verify.add_argument("--source-system", required=True)
    verify.add_argument("--source-event-id", required=True)
    verify.add_argument("--source-effective-at", required=True)
    verify.add_argument("--operator-reference", required=True)
    verify.add_argument("--owner-evidence-reference", required=True)
    verify.add_argument("--confirm-owner-authorized", action="store_true")

    for command in ("activate", "pause"):
        action = subparsers.add_parser(command, add_help=True)
        action.add_argument("--offer-id", required=True)
        action.add_argument("--operator-reference", required=True)
        action.add_argument("--evidence-reference", required=True)
        action.add_argument("--confirm-owner-authorized", action="store_true")

    authorize_pilot = subparsers.add_parser("authorize-pilot", add_help=True)
    authorize_pilot.add_argument("--topic", required=True)
    authorize_pilot.add_argument("--cohort-limit", required=True, type=int)
    authorize_pilot.add_argument("--provider-ticket-cap", required=True, type=int)
    authorize_pilot.add_argument("--total-ticket-cap", required=True, type=int)
    authorize_pilot.add_argument("--owner-reference", required=True)
    authorize_pilot.add_argument("--evidence-reference", required=True)
    authorize_pilot.add_argument("--confirm-owner-authorized", action="store_true")

    enroll_pilot = subparsers.add_parser("enroll-pilot", add_help=True)
    enroll_pilot.add_argument("--pilot-id", required=True)
    enroll_pilot.add_argument("--claim-id", required=True)
    enroll_pilot.add_argument("--owner-reference", required=True)
    enroll_pilot.add_argument("--evidence-reference", required=True)
    enroll_pilot.add_argument("--confirm-owner-authorized", action="store_true")

    for command in ("activate-pilot", "close-pilot"):
        pilot_action = subparsers.add_parser(command, add_help=True)
        pilot_action.add_argument("--pilot-id", required=True)
        pilot_action.add_argument("--owner-reference", required=True)
        pilot_action.add_argument("--evidence-reference", required=True)
        pilot_action.add_argument("--confirm-owner-authorized", action="store_true")

    status_pilot = subparsers.add_parser("status-pilot", add_help=True)
    status_pilot.add_argument("--pilot-id", required=True)

    review_candidate = subparsers.add_parser("review-candidate", add_help=True)
    review_candidate.add_argument("--pilot-id", required=True)
    review_candidate.add_argument("--review-type", required=True, choices=sorted(PILOT_REVIEW_TYPES))
    review_candidate.add_argument("--subject-id", required=True)

    record_review = subparsers.add_parser("record-review", add_help=True)
    record_review.add_argument("--pilot-id", required=True)
    record_review.add_argument("--review-type", required=True, choices=sorted(PILOT_REVIEW_TYPES))
    record_review.add_argument("--subject-id", required=True)
    record_review.add_argument("--expected-snapshot-sha256", required=True)
    record_review.add_argument("--owner-reference", required=True)
    record_review.add_argument("--evidence-reference", required=True)
    record_review.add_argument("--confirm-owner-authorized", action="store_true")

    issue_manifest = subparsers.add_parser("issue-proof-manifest", add_help=True)
    issue_manifest.add_argument("--pilot-id", required=True)
    issue_manifest.add_argument("--expected-snapshot-sha256", required=True)
    issue_manifest.add_argument("--owner-reference", required=True)
    issue_manifest.add_argument("--evidence-reference", required=True)
    issue_manifest.add_argument("--confirm-owner-authorized", action="store_true")
    return parser


def disable_core_dumps() -> None:
    try:
        resource.setrlimit(resource.RLIMIT_CORE, (0, 0))
    except (OSError, ValueError) as error:
        raise OperatorError("core_dump_hardening_unavailable") from error


def admin_key(environ: Mapping[str, str]) -> str:
    value = environ.get(ADMIN_KEY_ENV, "")
    if not isinstance(value, str):
        raise OperatorError("admin_key_unavailable")
    try:
        encoded = value.encode("ascii", "strict")
    except UnicodeEncodeError as error:
        raise OperatorError("admin_key_unavailable") from error
    if not 8 <= len(encoded) <= 4096 or any(byte <= 0x20 or byte == 0x7F for byte in encoded):
        raise OperatorError("admin_key_unavailable")
    return value


def _uuid(value: str, code: str) -> str:
    normalized = value.strip().lower()
    if value != normalized or not _UUID_PATTERN.fullmatch(normalized):
        raise OperatorError(code)
    return normalized


def _optional_uuid(value: str, code: str) -> str:
    return "" if value == "" else _uuid(value, code)


def _reference(value: str, code: str) -> str:
    normalized = value.strip()
    if value != normalized or not _REFERENCE_PATTERN.fullmatch(normalized):
        raise OperatorError(code)
    return normalized


def _source_system(value: str) -> str:
    normalized = value.strip().lower()
    if value != normalized or not _SOURCE_SYSTEM_PATTERN.fullmatch(normalized):
        raise OperatorError("invalid_source_system")
    return normalized


def _pilot_topic(value: str) -> str:
    normalized = value.strip().lower()
    if value != normalized or normalized not in PILOT_TOPICS:
        raise OperatorError("invalid_pilot_topic")
    return normalized


def _review_type(value: str) -> str:
    normalized = value.strip().lower()
    if value != normalized or normalized not in PILOT_REVIEW_TYPES:
        raise OperatorError("invalid_review_type")
    return normalized


def _hash(value: str, code: str) -> str:
    normalized = value.strip().lower()
    if value != normalized or not _HASH_PATTERN.fullmatch(normalized):
        raise OperatorError(code)
    return normalized


def _timestamp(value: object, code: str = "invalid_response") -> str:
    if not isinstance(value, str) or len(value) > 64:
        raise OperatorError(code)
    try:
        parsed = datetime.datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError as error:
        raise OperatorError(code) from error
    if parsed.tzinfo is None or parsed.utcoffset() is None:
        raise OperatorError(code)
    return value


def build_opener():
    return urllib.request.build_opener(
        urllib.request.ProxyHandler({}),
        NoRedirect(),
        urllib.request.HTTPSHandler(context=ssl.create_default_context()),
    )


def build_request(path: str, key: str, *, payload: Mapping[str, object] | None = None) -> urllib.request.Request:
    fixed_paths = {
        QUEUE_PATH,
        OFFER_ACTION_PATH,
        COMMERCIAL_ACTION_PATH,
        PILOT_ACTION_PATH,
    }
    allowed = path in fixed_paths or path.startswith(QUEUE_PATH + "?")
    if path.startswith(PILOT_EPOCH_PATH + "?"):
        parsed = urllib.parse.urlsplit(path)
        try:
            query = urllib.parse.parse_qs(parsed.query, keep_blank_values=True, strict_parsing=True)
        except ValueError as error:
            raise OperatorError("invalid_operator_path") from error
        allowed = (
            parsed.path == PILOT_EPOCH_PATH
            and not parsed.fragment
            and set(query) == {"pilot_id"}
            and len(query["pilot_id"]) == 1
            and bool(_UUID_PATTERN.fullmatch(query["pilot_id"][0]))
        )
    if path == PILOT_REVIEW_PATH:
        allowed = payload is not None
    elif path.startswith(PILOT_REVIEW_PATH + "?"):
        parsed = urllib.parse.urlsplit(path)
        try:
            query = urllib.parse.parse_qs(parsed.query, keep_blank_values=True, strict_parsing=True)
        except ValueError as error:
            raise OperatorError("invalid_operator_path") from error
        allowed = (
            payload is None
            and parsed.path == PILOT_REVIEW_PATH
            and not parsed.fragment
            and set(query) == {"pilot_id", "review_type", "subject_id"}
            and all(len(values) == 1 for values in query.values())
            and bool(_UUID_PATTERN.fullmatch(query["pilot_id"][0]))
            and query["review_type"][0] in PILOT_REVIEW_TYPES
            and bool(_UUID_PATTERN.fullmatch(query["subject_id"][0]))
        )
    if path == PROOF_MANIFEST_PATH:
        allowed = payload is not None
    if not allowed:
        raise OperatorError("invalid_operator_path")
    data = None
    method = "GET"
    headers = {
        "Accept": "application/json",
        "Authorization": "Bearer " + key,
        "User-Agent": "NHS-Provider-Pilot-Operate/1.0",
    }
    if payload is not None:
        data = json.dumps(dict(payload), sort_keys=True, separators=(",", ":")).encode("ascii")
        method = "POST"
        headers["Content-Type"] = "application/json"
    return urllib.request.Request(BASE_URL + path, data=data, method=method, headers=headers)


def perform_request(opener, request: urllib.request.Request) -> tuple[int, dict[str, object]]:
    status = None
    try:
        with opener.open(request, timeout=REQUEST_TIMEOUT_SECONDS) as response:
            status = int(response.getcode())
            if 300 <= status <= 399:
                raise OperatorError("redirect_refused", http_status=status)
            if status not in (200, 201):
                raise OperatorError("http_error", http_status=status)
            body = response.read(MAX_RESPONSE_BYTES + 1)
    except urllib.error.HTTPError as error:
        safe_status = error.code if isinstance(error.code, int) and 100 <= error.code <= 599 else None
        code = "redirect_refused" if safe_status is not None and 300 <= safe_status <= 399 else "http_error"
        raise OperatorError(code, http_status=safe_status) from error
    except (urllib.error.URLError, TimeoutError, OSError, ssl.SSLError) as error:
        raise OperatorError("network_error") from error
    if len(body) > MAX_RESPONSE_BYTES:
        raise OperatorError("response_too_large", http_status=status)
    try:
        document = json.loads(body.decode("utf-8", "strict"))
    except (UnicodeDecodeError, json.JSONDecodeError) as error:
        raise OperatorError("invalid_response", http_status=status) from error
    if not isinstance(document, dict):
        raise OperatorError("invalid_response", http_status=status)
    return status, document


def _project_queue(
    document: Mapping[str, object],
    expected_state: str,
    expected_limit: int,
    *,
    action: str = "queue",
) -> dict[str, object]:
    queue = document.get("queue")
    if set(document) != {"queue", "evidence_scope"} or not isinstance(queue, dict):
        raise OperatorError("invalid_response")
    required = {"as_of", "state", "limit_per_state", "returned_counts", "items", "redaction_contract"}
    if set(queue) != required or queue.get("state") != expected_state or queue.get("limit_per_state") != expected_limit:
        raise OperatorError("invalid_response")
    as_of = _timestamp(queue.get("as_of"))
    counts = queue.get("returned_counts")
    items = queue.get("items")
    if not isinstance(counts, dict) or not isinstance(items, list) or len(items) > expected_limit * len(QUEUE_STATES):
        raise OperatorError("invalid_response")
    projected_counts: dict[str, int] = {}
    for state, count in counts.items():
        if state not in QUEUE_STATES - {"all"} or isinstance(count, bool) or not isinstance(count, int) or count < 0:
            raise OperatorError("invalid_response")
        projected_counts[state] = count
    projected_items = []
    allowed_item_fields = {
        "state", "provider_pilot_epoch_id", "provider_claim_id", "domain",
        "offer_id", "offer_version",
        "commercial_terms_sha256", "commitment_event_id", "commitment_event_type",
        "acceptance_event_id", "acceptance_event_type",
        "related_acceptance_event_id", "related_commitment_event_id",
        "provider_acceptance_reference", "ticket_id",
        "handoff_receipt_id", "outcome_receipt_id", "outcome", "charge_status",
        "review_type", "subject_id", "subject_snapshot_sha256",
        "occurred_at", "valid_until",
    }
    for item in items:
        if not isinstance(item, dict) or not {"state", "provider_claim_id", "domain", "occurred_at"} <= set(item):
            raise OperatorError("invalid_response")
        if not set(item) <= allowed_item_fields or item.get("state") not in QUEUE_STATES - {"all"}:
            raise OperatorError("invalid_response")
        if not isinstance(item.get("provider_claim_id"), str) or not _UUID_PATTERN.fullmatch(item["provider_claim_id"]):
            raise OperatorError("invalid_response")
        if not isinstance(item.get("domain"), str) or not _DOMAIN_PATTERN.fullmatch(item["domain"]):
            raise OperatorError("invalid_response")
        _timestamp(item.get("occurred_at"))
        if "valid_until" in item:
            _timestamp(item["valid_until"])
        for field in (
            "provider_pilot_epoch_id", "offer_id", "commitment_event_id",
            "acceptance_event_id",
            "related_acceptance_event_id",
            "related_commitment_event_id", "ticket_id", "handoff_receipt_id",
            "outcome_receipt_id", "subject_id",
        ):
            if field in item and (not isinstance(item[field], str) or not _UUID_PATTERN.fullmatch(item[field])):
                raise OperatorError("invalid_response")
        if "commercial_terms_sha256" in item and (
            not isinstance(item["commercial_terms_sha256"], str)
            or not _HASH_PATTERN.fullmatch(item["commercial_terms_sha256"])
        ):
            raise OperatorError("invalid_response")
        if "review_type" in item and item["review_type"] not in PILOT_REVIEW_TYPES:
            raise OperatorError("invalid_response")
        if "commitment_event_type" in item and item["commitment_event_type"] not in {
            "prepaid_fund", "terms_acceptance", "terms_renewal"
        }:
            raise OperatorError("invalid_response")
        if "subject_snapshot_sha256" in item and (
            not isinstance(item["subject_snapshot_sha256"], str)
            or not _HASH_PATTERN.fullmatch(item["subject_snapshot_sha256"])
        ):
            raise OperatorError("invalid_response")
        if item.get("state", "").endswith("_review_required") and (
            "provider_pilot_epoch_id" not in item
            or "review_type" not in item
            or "subject_id" not in item
            or "subject_snapshot_sha256" not in item
        ):
            raise OperatorError("invalid_response")
        if item.get("state") == "offer_review_required" and (
            item.get("review_type") != "offer"
            or "commitment_event_id" not in item
            or "commitment_event_type" not in item
        ):
            raise OperatorError("invalid_response")
        projected_items.append({field: item[field] for field in sorted(item)})
    if sum(projected_counts.values()) != len(projected_items):
        raise OperatorError("invalid_response")
    projected = {
        "ok": True,
        "action": action,
        "as_of": as_of,
        "state": expected_state,
        "limit_per_state": expected_limit,
        "returned_counts": dict(sorted(projected_counts.items())),
        "items": projected_items,
        "commercial_proof_created": False,
    }
    if action == "review_preflight":
        projected["pre_event_reviews_ready"] = len(projected_items) == 0
        projected["review_blocker_count"] = len(projected_items)
    return projected


def _project_verify_terms(status: int, document: Mapping[str, object]) -> dict[str, object]:
    created = document.get("created")
    replay = document.get("idempotent_replay")
    commitment = document.get("commitment")
    if (
        not isinstance(created, bool)
        or not isinstance(replay, bool)
        or created == replay
        or created != (status == 201)
        or document.get("commercial_evidence_recorded") is not True
        or document.get("pilot_threshold_evaluated") is not False
        or not isinstance(commitment, dict)
    ):
        raise OperatorError("invalid_response", http_status=status)
    commitment_id = commitment.get("id")
    event_type = commitment.get("event_type")
    related_event_id = commitment.get("related_event_id", "")
    if (
        not isinstance(commitment_id, str)
        or not _UUID_PATTERN.fullmatch(commitment_id)
        or event_type not in {"terms_acceptance", "terms_renewal"}
        or (
            related_event_id != ""
            and (not isinstance(related_event_id, str) or not _UUID_PATTERN.fullmatch(related_event_id))
        )
        or (event_type == "terms_acceptance" and related_event_id != "")
        or (event_type == "terms_renewal" and related_event_id == "")
    ):
        raise OperatorError("invalid_response", http_status=status)
    return {
        "ok": True,
        "action": "verify_terms",
        "http_status": status,
        "created": created,
        "idempotent_replay": replay,
        "commercial_evidence_recorded": True,
        "pilot_threshold_evaluated": False,
        "commitment_event_id": commitment_id,
        "commitment_event_type": event_type,
        "related_commitment_event_id": related_event_id,
    }


def _project_offer_action(status: int, document: Mapping[str, object], action: str, offer_id: str) -> dict[str, object]:
    offer = document.get("offer")
    if status != 200 or not isinstance(offer, dict) or offer.get("id") != offer_id or offer.get("status") != (
        "active" if action == "activate" else "paused"
    ):
        raise OperatorError("invalid_response", http_status=status)
    if action == "activate":
        if document.get("commercial_proof_created") is not False:
            raise OperatorError("invalid_response", http_status=status)
    elif document.get("paused") is not True:
        raise OperatorError("invalid_response", http_status=status)
    return {
        "ok": True,
        "action": action,
        "http_status": status,
        "offer_id": offer_id,
        "status": offer["status"],
        "commercial_proof_created": False,
    }


_PILOT_EPOCH_REQUIRED_FIELDS = frozenset(
    {
        "id",
        "contract_version",
        "demand_topic",
        "stage1_started_at",
        "stage1_evidence_as_of",
        "stage1_evidence_sha256",
        "cohort_limit",
        "provider_ticket_cap",
        "total_ticket_cap",
        "status",
        "owner_reference",
        "evidence_reference",
        "created_at",
        "updated_at",
    }
)
_PILOT_EPOCH_OPTIONAL_FIELDS = frozenset({"activated_at", "closed_at"})
_PILOT_STATUS_FIELDS = frozenset(
    {
        "as_of",
        "enrolled_provider_count",
        "fresh_enrolled_provider_count",
        "non_synthetic_ticket_count",
        "remaining_ticket_capacity",
        "event_count",
        "cohort_ready",
    }
)


def _response_int(value: object, *, minimum: int = 0, maximum: int | None = None) -> int:
    if isinstance(value, bool) or not isinstance(value, int) or value < minimum:
        raise OperatorError("invalid_response")
    if maximum is not None and value > maximum:
        raise OperatorError("invalid_response")
    return value


def _validate_pilot_epoch(
    value: object,
    *,
    expected_pilot_id: str = "",
    additional_fields: frozenset[str] = frozenset(),
) -> Mapping[str, object]:
    if not isinstance(value, dict):
        raise OperatorError("invalid_response")
    fields = set(value)
    required = _PILOT_EPOCH_REQUIRED_FIELDS | additional_fields
    if not required <= fields or not fields <= required | _PILOT_EPOCH_OPTIONAL_FIELDS:
        raise OperatorError("invalid_response")

    pilot_id = value.get("id")
    if not isinstance(pilot_id, str) or not _UUID_PATTERN.fullmatch(pilot_id):
        raise OperatorError("invalid_response")
    if expected_pilot_id and pilot_id != expected_pilot_id:
        raise OperatorError("invalid_response")
    if value.get("contract_version") != PILOT_CONTRACT_VERSION or value.get("demand_topic") not in PILOT_TOPICS:
        raise OperatorError("invalid_response")
    if (
        not isinstance(value.get("stage1_evidence_sha256"), str)
        or not _HASH_PATTERN.fullmatch(value["stage1_evidence_sha256"])
    ):
        raise OperatorError("invalid_response")
    for field in ("stage1_started_at", "stage1_evidence_as_of", "created_at", "updated_at"):
        _timestamp(value.get(field))
    if "activated_at" in value:
        _timestamp(value["activated_at"])
    if "closed_at" in value:
        _timestamp(value["closed_at"])
    for field in ("owner_reference", "evidence_reference"):
        reference = value.get(field)
        if not isinstance(reference, str) or not _REFERENCE_PATTERN.fullmatch(reference):
            raise OperatorError("invalid_response")

    cohort_limit = _response_int(
        value.get("cohort_limit"), minimum=PILOT_MINIMUM_COHORT, maximum=PILOT_MAXIMUM_COHORT
    )
    _response_int(
        value.get("provider_ticket_cap"), minimum=1, maximum=PILOT_MAXIMUM_PROVIDER_TICKETS
    )
    total_ticket_cap = _response_int(
        value.get("total_ticket_cap"),
        minimum=PILOT_MINIMUM_TOTAL_TICKETS,
        maximum=PILOT_MAXIMUM_TOTAL_TICKETS,
    )
    if total_ticket_cap < cohort_limit:
        raise OperatorError("invalid_response")

    pilot_status = value.get("status")
    if pilot_status not in PILOT_STATUSES:
        raise OperatorError("invalid_response")
    if pilot_status == "draft" and ("activated_at" in value or "closed_at" in value):
        raise OperatorError("invalid_response")
    if pilot_status == "active" and ("activated_at" not in value or "closed_at" in value):
        raise OperatorError("invalid_response")
    if pilot_status == "closed" and ("activated_at" not in value or "closed_at" not in value):
        raise OperatorError("invalid_response")
    return value


def _project_pilot_epoch_mutation(
    status: int,
    document: Mapping[str, object],
    action: str,
    *,
    expected_pilot_id: str = "",
) -> dict[str, object]:
    expected_http_status = 201 if action == "authorize" else 200
    expected_epoch_status = {"authorize": "draft", "activate": "active", "close": "closed"}[action]
    if (
        status != expected_http_status
        or set(document) != {"action", "provider_pilot", "commercial_proof_created", "evidence_scope"}
        or document.get("action") != action
        or document.get("commercial_proof_created") is not False
        or document.get("evidence_scope") != PILOT_MUTATION_EVIDENCE_SCOPE
    ):
        raise OperatorError("invalid_response", http_status=status)
    epoch = _validate_pilot_epoch(document.get("provider_pilot"), expected_pilot_id=expected_pilot_id)
    if epoch.get("status") != expected_epoch_status:
        raise OperatorError("invalid_response", http_status=status)
    return {
        "ok": True,
        "action": action + "_pilot",
        "http_status": status,
        "pilot_id": epoch["id"],
        "demand_topic": epoch["demand_topic"],
        "cohort_limit": epoch["cohort_limit"],
        "provider_ticket_cap": epoch["provider_ticket_cap"],
        "total_ticket_cap": epoch["total_ticket_cap"],
        "status": epoch["status"],
        "created_at": epoch["created_at"],
        "activated_at": epoch.get("activated_at", ""),
        "closed_at": epoch.get("closed_at", ""),
        "commercial_proof_created": False,
    }


def _project_pilot_enrollment(
    status: int,
    document: Mapping[str, object],
    pilot_id: str,
    claim_id: str,
) -> dict[str, object]:
    if (
        status != 200
        or set(document) != {"action", "provider_pilot", "commercial_proof_created", "evidence_scope"}
        or document.get("action") != "enroll"
        or document.get("commercial_proof_created") is not False
        or document.get("evidence_scope") != PILOT_MUTATION_EVIDENCE_SCOPE
    ):
        raise OperatorError("invalid_response", http_status=status)
    enrollment = document.get("provider_pilot")
    required = {
        "id", "provider_pilot_epoch_id", "provider_pilot_company_id", "provider_claim_id",
        "owner_reference", "evidence_reference", "enrolled_at",
    }
    if not isinstance(enrollment, dict) or set(enrollment) != required:
        raise OperatorError("invalid_response", http_status=status)
    for field in ("id", "provider_pilot_epoch_id", "provider_pilot_company_id", "provider_claim_id"):
        value = enrollment.get(field)
        if not isinstance(value, str) or not _UUID_PATTERN.fullmatch(value):
            raise OperatorError("invalid_response", http_status=status)
    if enrollment["provider_pilot_epoch_id"] != pilot_id or enrollment["provider_claim_id"] != claim_id:
        raise OperatorError("invalid_response", http_status=status)
    for field in ("owner_reference", "evidence_reference"):
        value = enrollment.get(field)
        if not isinstance(value, str) or not _REFERENCE_PATTERN.fullmatch(value):
            raise OperatorError("invalid_response", http_status=status)
    enrolled_at = _timestamp(enrollment.get("enrolled_at"))
    return {
        "ok": True,
        "action": "enroll_pilot",
        "http_status": status,
        "pilot_id": pilot_id,
        "provider_claim_id": claim_id,
        "enrollment_id": enrollment["id"],
        "provider_pilot_company_id": enrollment["provider_pilot_company_id"],
        "enrolled_at": enrolled_at,
        "commercial_proof_created": False,
    }


def _project_pilot_status(
    status: int,
    document: Mapping[str, object],
    pilot_id: str,
) -> dict[str, object]:
    if (
        status != 200
        or set(document) != {"provider_pilot", "evidence_scope"}
        or document.get("evidence_scope") != PILOT_STATUS_EVIDENCE_SCOPE
    ):
        raise OperatorError("invalid_response", http_status=status)
    epoch = _validate_pilot_epoch(
        document.get("provider_pilot"),
        expected_pilot_id=pilot_id,
        additional_fields=_PILOT_STATUS_FIELDS,
    )
    as_of = _timestamp(epoch.get("as_of"))
    enrolled = _response_int(epoch.get("enrolled_provider_count"), maximum=epoch["cohort_limit"])
    fresh = _response_int(epoch.get("fresh_enrolled_provider_count"), maximum=enrolled)
    tickets = _response_int(epoch.get("non_synthetic_ticket_count"), maximum=epoch["total_ticket_cap"])
    remaining = _response_int(epoch.get("remaining_ticket_capacity"), maximum=epoch["total_ticket_cap"])
    event_count = _response_int(epoch.get("event_count"))
    cohort_ready = epoch.get("cohort_ready")
    expected_ready = enrolled >= PILOT_MINIMUM_COHORT and fresh == enrolled
    if (
        not isinstance(cohort_ready, bool)
        or cohort_ready != expected_ready
        or remaining != epoch["total_ticket_cap"] - tickets
    ):
        raise OperatorError("invalid_response", http_status=status)
    return {
        "ok": True,
        "action": "status_pilot",
        "http_status": status,
        "pilot_id": pilot_id,
        "contract_version": epoch["contract_version"],
        "demand_topic": epoch["demand_topic"],
        "cohort_limit": epoch["cohort_limit"],
        "provider_ticket_cap": epoch["provider_ticket_cap"],
        "total_ticket_cap": epoch["total_ticket_cap"],
        "status": epoch["status"],
        "activated_at": epoch.get("activated_at", ""),
        "closed_at": epoch.get("closed_at", ""),
        "as_of": as_of,
        "enrolled_provider_count": enrolled,
        "fresh_enrolled_provider_count": fresh,
        "non_synthetic_ticket_count": tickets,
        "remaining_ticket_capacity": remaining,
        "event_count": event_count,
        "cohort_ready": cohort_ready,
        "commercial_proof_created": False,
    }


_SIGNED_PROOF_MANIFEST_FIELDS = frozenset(
    {
        "v",
        "kid",
        "signature_verification_scope",
        "manifest_contract_version",
        "manifest_id",
        "provider_pilot_epoch_id",
        "provider_pilot_contract_version",
        "review_contract_version",
        "review_evidence_contract_version",
        "market_policy_contract_version",
        "proof_snapshot_sha256",
        "review_evidence_sha256",
        "pilot_demand_topic",
        "pilot_status",
        "issued_at",
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
_PROOF_MANIFEST_RECORD_FIELDS = frozenset(
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
_PROOF_REVIEW_COVERAGE_FIELDS = frozenset(
    {"providers", "offers", "tickets", "handoffs", "callbacks"}
)


def _validate_proof_manifest_currency_amounts(value: object) -> None:
    if not isinstance(value, list) or len(value) > 16:
        raise OperatorError("invalid_response")
    previous = ""
    for item in value:
        if not isinstance(item, dict) or set(item) != {"currency", "amount_minor"}:
            raise OperatorError("invalid_response")
        currency = item.get("currency")
        amount = item.get("amount_minor")
        if (
            not isinstance(currency, str)
            or not re.fullmatch(r"[a-z]{3}", currency)
            or currency <= previous
            or isinstance(amount, bool)
            or not isinstance(amount, int)
            or not -(2**63) <= amount <= 2**63 - 1
            or amount == 0
        ):
            raise OperatorError("invalid_response")
        previous = currency


def _validate_issued_proof_manifest(
    value: object,
    pilot_id: str,
    expected_digest: str,
) -> Mapping[str, object]:
    if not isinstance(value, dict) or set(value) != _PROOF_MANIFEST_RECORD_FIELDS:
        raise OperatorError("invalid_response")
    manifest_id = value.get("id")
    key_id = value.get("key_id")
    signed_manifest = value.get("signed_manifest")
    signature = value.get("signature")
    payload_sha256 = value.get("payload_sha256")
    review_evidence_sha256 = value.get("review_evidence_sha256")
    if (
        not isinstance(manifest_id, str)
        or not _UUID_PATTERN.fullmatch(manifest_id)
        or value.get("provider_pilot_epoch_id") != pilot_id
        or value.get("manifest_contract_version") != PROOF_MANIFEST_CONTRACT_VERSION
        or value.get("proof_snapshot_sha256") != expected_digest
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
        raise OperatorError("invalid_response")
    issued_at = _timestamp(value.get("issued_at"))
    try:
        signed = json.loads(signed_manifest)
    except json.JSONDecodeError as error:
        raise OperatorError("invalid_response") from error
    if (
        not isinstance(signed, dict)
        or set(signed) != _SIGNED_PROOF_MANIFEST_FIELDS
        or json.dumps(signed, separators=(",", ":"), ensure_ascii=True) != signed_manifest
        or signed.get("v") != 1
        or isinstance(signed.get("v"), bool)
        or signed.get("kid") != key_id
        or signed.get("signature_verification_scope") != PROOF_MANIFEST_VERIFICATION_SCOPE
        or signed.get("manifest_contract_version") != PROOF_MANIFEST_CONTRACT_VERSION
        or signed.get("manifest_id") != manifest_id
        or signed.get("provider_pilot_epoch_id") != pilot_id
        or signed.get("provider_pilot_contract_version") != PILOT_CONTRACT_VERSION
        or signed.get("review_contract_version") != PILOT_REVIEW_CONTRACT_VERSION
        or signed.get("review_evidence_contract_version")
        != PROOF_MANIFEST_REVIEW_EVIDENCE_VERSION
        or signed.get("market_policy_contract_version")
        != PROOF_MANIFEST_MARKET_POLICY_VERSION
        or signed.get("proof_snapshot_sha256") != expected_digest
        or signed.get("review_evidence_sha256") != review_evidence_sha256
        or signed.get("pilot_demand_topic") not in PILOT_TOPICS
        or signed.get("pilot_status") != "closed"
        or signed.get("outcome_receipt_integrity_valid") is not True
        or signed.get("review_integrity_valid") is not True
        or signed.get("pilot_thresholds_met") is not True
        or signed.get("organic_rank_sold") is not False
        or signed.get("raw_queries_sold") is not False
        or signed.get("agent_identities_sold") is not False
        or signed.get("evidence_scope") != PROOF_MANIFEST_SCOPE
    ):
        raise OperatorError("invalid_response")

    signed_issued_at = signed.get("issued_at")
    if (
        isinstance(signed_issued_at, bool)
        or not isinstance(signed_issued_at, int)
        or signed_issued_at <= 0
        or int(datetime.datetime.fromisoformat(issued_at.replace("Z", "+00:00")).timestamp())
        != signed_issued_at
    ):
        raise OperatorError("invalid_response")

    count_fields = (
        "verified_outcome_receipts",
        "rejected_outcome_receipts",
        "verified_outcome_ledger_entries",
        "rejected_outcome_ledger_entries",
        "verified_provider_companies",
        "verified_provider_accepted_handoffs",
        "verified_provider_confirmed_activations",
        "verified_provider_renewals",
        "verified_provider_confirmed_conversions",
    )
    counts = {field: _response_int(signed.get(field), maximum=2**31 - 1) for field in count_fields}
    if (
        counts["rejected_outcome_receipts"] != 0
        or counts["rejected_outcome_ledger_entries"] != 0
        or counts["verified_provider_companies"] < 3
        or counts["verified_provider_accepted_handoffs"] < 5
        or counts["verified_provider_confirmed_activations"] < 2
        or counts["verified_provider_renewals"] < 1
        or counts["verified_provider_renewals"]
        > counts["verified_provider_companies"]
        or counts["verified_provider_accepted_handoffs"]
        > counts["verified_outcome_receipts"]
        or counts["verified_provider_confirmed_activations"]
        > counts["verified_provider_accepted_handoffs"]
        or counts["verified_provider_confirmed_conversions"]
        > counts["verified_provider_confirmed_activations"]
        or counts["verified_outcome_ledger_entries"]
        > counts["verified_outcome_receipts"]
    ):
        raise OperatorError("invalid_response")

    coverage = signed.get("review_coverage")
    if not isinstance(coverage, dict) or set(coverage) != _PROOF_REVIEW_COVERAGE_FIELDS:
        raise OperatorError("invalid_response")
    validated_coverage: dict[str, dict[str, int]] = {}
    for category in _PROOF_REVIEW_COVERAGE_FIELDS:
        review = coverage.get(category)
        if not isinstance(review, dict) or set(review) != {"required", "valid"}:
            raise OperatorError("invalid_response")
        required = _response_int(review.get("required"), minimum=1, maximum=2**31 - 1)
        valid = _response_int(review.get("valid"), minimum=1, maximum=required)
        if valid != required:
            raise OperatorError("invalid_response")
        validated_coverage[category] = {"required": required, "valid": valid}
    if (
        validated_coverage["providers"]["required"]
        != counts["verified_provider_companies"]
        or validated_coverage["offers"]["required"]
        < counts["verified_provider_companies"]
        or validated_coverage["tickets"]["required"]
        < counts["verified_provider_accepted_handoffs"]
        or validated_coverage["handoffs"]["required"]
        != validated_coverage["tickets"]["required"]
        or validated_coverage["callbacks"]["required"]
        != counts["verified_outcome_receipts"]
    ):
        raise OperatorError("invalid_response")
    if signed.get("monetary_amounts_withheld_for_privacy") is not True or any(
        signed.get(field) != []
        for field in (
            "verified_prepaid_settled",
            "verified_prepaid_net_debited",
            "verified_terms_net_receivable",
        )
    ):
        raise OperatorError("invalid_response")
    return value


def _project_issue_proof_manifest(
    status: int,
    document: Mapping[str, object],
    pilot_id: str,
    expected_digest: str,
) -> dict[str, object]:
    expected_fields = {
        "manifest",
        "created",
        "idempotent_replay",
        "commercial_proof_created",
        "publicly_released",
        "independently_verifiable",
        "evidence_scope",
    }
    if (
        set(document) != expected_fields
        or document.get("commercial_proof_created") is not True
        or document.get("publicly_released") is not False
        or document.get("independently_verifiable") is not False
        or document.get("evidence_scope") != PROOF_MANIFEST_EVIDENCE_SCOPE
    ):
        raise OperatorError("invalid_response", http_status=status)
    created = document.get("created")
    replay = document.get("idempotent_replay")
    if (
        not isinstance(created, bool)
        or not isinstance(replay, bool)
        or created == replay
        or (created and status != 201)
        or (replay and status != 200)
    ):
        raise OperatorError("invalid_response", http_status=status)
    manifest = _validate_issued_proof_manifest(
        document.get("manifest"), pilot_id, expected_digest
    )
    return {
        "ok": True,
        "action": "issue_proof_manifest",
        "http_status": status,
        "created": created,
        "idempotent_replay": replay,
        "manifest_id": manifest["id"],
        "pilot_id": pilot_id,
        "manifest_contract_version": PROOF_MANIFEST_CONTRACT_VERSION,
        "proof_snapshot_sha256": expected_digest,
        "review_evidence_sha256": manifest["review_evidence_sha256"],
        "key_id": manifest["key_id"],
        "payload_sha256": manifest["payload_sha256"],
        "issued_at": manifest["issued_at"],
        "commercial_proof_created": True,
        "publicly_released": False,
        "independently_verifiable": False,
    }


_REVIEW_CANDIDATE_COMMON_FIELDS = frozenset(
    {
        "review_contract_version", "provider_pilot_epoch_id", "review_type", "subject_id",
        "provider_pilot_contract_version", "pilot_demand_topic", "subject_snapshot_sha256",
        "provider_claim_id", "domain",
    }
)
_REVIEW_CANDIDATE_TYPE_FIELDS = {
    "provider": frozenset(
        {
            "provider_pilot_company_id", "provider_pilot_enrollment_id",
            "provider_acceptance_event_id", "provider_acceptance_reference", "provider_accepted_at",
            "company_owner_verified_at", "enrolled_at",
        }
    ),
    "offer": frozenset(
        {
            "provider_pilot_company_id", "provider_offer_id", "offer_version", "offer_name",
            "offer_summary", "action_type", "action_url", "disclosure_label", "charge_event",
            "bounty_cents", "currency", "principal_price_mode", "principal_price_cents",
            "principal_currency", "billing_mode",
            "terms_credit_limit_cents", "terms_period_days", "commercial_terms_contract_version",
            "commercial_terms_sha256", "commitment_event_id", "commitment_event_type",
            "provider_acceptance_event_id",
            "commitment_provider_accepted_at", "commitment_valid_until",
            "commitment_owner_verified_at",
        }
    ),
    "ticket": frozenset(
        {
            "provider_offer_id", "action_ticket_id", "offer_version", "offer_name",
            "offer_summary", "action_type", "action_url", "disclosure_label", "charge_event",
            "bounty_cents", "currency", "principal_price_mode", "principal_price_cents",
            "principal_currency", "billing_mode",
            "terms_credit_limit_cents", "terms_period_days", "commercial_terms_contract_version",
            "commercial_terms_sha256", "demand_topic", "region_code", "budget_band", "urgency",
            "requirement_flags", "principal_consent", "consent_version", "ticket_created_at",
            "ticket_expires_at",
        }
    ),
    "handoff": frozenset(
        {
            "provider_offer_id", "action_ticket_id", "handoff_receipt_id", "offer_version",
            "commercial_terms_contract_version", "commercial_terms_sha256", "action_type",
            "principal_handoff_consent", "handoff_consent_version",
            "controlled_intent_disclosure_consent",
            "controlled_intent_disclosure_consent_version", "handoff_event_contract_version",
            "handoff_observed_at",
        }
    ),
    "callback": frozenset(
        {
            "provider_offer_id", "action_ticket_id", "handoff_receipt_id", "outcome_receipt_id",
            "outcome_nhs_event_id", "provider_api_key_id", "offer_version", "action_type",
            "charge_event", "commercial_terms_contract_version", "commercial_terms_sha256",
            "outcome", "charge_status", "billed_cents", "currency",
            "outcome_signed_receipt_sha256", "outcome_signature_sha256",
            "provider_reported_at", "outcome_recorded_at",
        }
    ),
}
_REVIEW_CANDIDATE_REQUIRED = {
    "provider": _REVIEW_CANDIDATE_TYPE_FIELDS["provider"],
    "offer": _REVIEW_CANDIDATE_TYPE_FIELDS["offer"] - {"principal_price_cents"},
    "ticket": _REVIEW_CANDIDATE_TYPE_FIELDS["ticket"]
    - {"principal_price_cents", "region_code", "requirement_flags"},
    "handoff": _REVIEW_CANDIDATE_TYPE_FIELDS["handoff"]
    - {"controlled_intent_disclosure_consent", "controlled_intent_disclosure_consent_version"},
    "callback": _REVIEW_CANDIDATE_TYPE_FIELDS["callback"],
}
_REVIEW_EXISTING_FIELDS = frozenset({"existing_review_id", "existing_reviewed_at"})
_REVIEW_UUID_FIELDS = frozenset(
    {
        "provider_pilot_epoch_id", "subject_id", "provider_claim_id", "provider_pilot_company_id",
        "provider_pilot_enrollment_id", "provider_acceptance_event_id", "provider_offer_id",
        "commitment_event_id", "action_ticket_id", "handoff_receipt_id", "outcome_receipt_id",
        "outcome_nhs_event_id", "existing_review_id", "id",
    }
)
_REVIEW_HASH_FIELDS = frozenset(
    {
        "subject_snapshot_sha256", "commercial_terms_sha256",
        "outcome_signed_receipt_sha256", "outcome_signature_sha256",
    }
)
_REVIEW_TIMESTAMP_FIELDS = frozenset(
    {
        "provider_accepted_at", "company_owner_verified_at", "enrolled_at",
        "commitment_provider_accepted_at", "commitment_valid_until",
        "commitment_owner_verified_at", "ticket_created_at",
        "ticket_expires_at", "handoff_observed_at", "provider_reported_at",
        "outcome_recorded_at", "existing_reviewed_at", "reviewed_at",
    }
)
_REVIEW_INTEGER_FIELDS = frozenset(
    {
        "offer_version", "bounty_cents", "principal_price_cents", "terms_credit_limit_cents",
        "terms_period_days", "billed_cents", "provider_api_key_id",
    }
)
_REVIEW_BOOL_FIELDS = frozenset(
    {
        "principal_consent", "principal_handoff_consent",
        "controlled_intent_disclosure_consent",
    }
)


def _validate_review_fields(value: Mapping[str, object]) -> None:
    for field, item in value.items():
        if field in _REVIEW_UUID_FIELDS:
            if not isinstance(item, str) or not _UUID_PATTERN.fullmatch(item):
                raise OperatorError("invalid_response")
        elif field in _REVIEW_HASH_FIELDS:
            if not isinstance(item, str) or not _HASH_PATTERN.fullmatch(item):
                raise OperatorError("invalid_response")
        elif field in _REVIEW_TIMESTAMP_FIELDS:
            _timestamp(item)
        elif field in _REVIEW_INTEGER_FIELDS:
            if isinstance(item, bool) or not isinstance(item, int) or item < 0 or item > 100_000_000_000:
                raise OperatorError("invalid_response")
        elif field in _REVIEW_BOOL_FIELDS:
            if not isinstance(item, bool):
                raise OperatorError("invalid_response")
        elif field == "requirement_flags":
            if (
                not isinstance(item, list)
                or len(item) > 20
                or any(
                    not isinstance(flag, str)
                    or not re.fullmatch(r"[a-z0-9_]{1,64}", flag)
                    for flag in item
                )
            ):
                raise OperatorError("invalid_response")
        elif not isinstance(item, str) or not 1 <= len(item) <= 4096:
            raise OperatorError("invalid_response")


def _project_review_candidate(
    status: int,
    document: Mapping[str, object],
    pilot_id: str,
    review_type: str,
    subject_id: str,
) -> dict[str, object]:
    if (
        status != 200
        or set(document) != {"review_candidate", "evidence_scope"}
        or document.get("evidence_scope") != PILOT_REVIEW_EVIDENCE_SCOPE
    ):
        raise OperatorError("invalid_response", http_status=status)
    candidate = document.get("review_candidate")
    allowed = (
        _REVIEW_CANDIDATE_COMMON_FIELDS
        | _REVIEW_CANDIDATE_TYPE_FIELDS[review_type]
        | _REVIEW_EXISTING_FIELDS
    )
    required = _REVIEW_CANDIDATE_COMMON_FIELDS | _REVIEW_CANDIDATE_REQUIRED[review_type]
    if not isinstance(candidate, dict) or not required <= set(candidate) or not set(candidate) <= allowed:
        raise OperatorError("invalid_response", http_status=status)
    _validate_review_fields(candidate)
    if (
        candidate.get("review_contract_version") != PILOT_REVIEW_CONTRACT_VERSION
        or candidate.get("provider_pilot_epoch_id") != pilot_id
        or candidate.get("provider_pilot_contract_version") != PILOT_CONTRACT_VERSION
        or candidate.get("pilot_demand_topic") not in PILOT_TOPICS
        or candidate.get("review_type") != review_type
        or candidate.get("subject_id") != subject_id
        or not isinstance(candidate.get("domain"), str)
        or not _DOMAIN_PATTERN.fullmatch(candidate["domain"])
        or (("existing_review_id" in candidate) != ("existing_reviewed_at" in candidate))
    ):
        raise OperatorError("invalid_response", http_status=status)
    subject_fields = {
        "provider": "provider_claim_id",
        "offer": "provider_offer_id",
        "ticket": "action_ticket_id",
        "handoff": "handoff_receipt_id",
        "callback": "outcome_receipt_id",
    }
    if candidate.get(subject_fields[review_type]) != subject_id:
        raise OperatorError("invalid_response", http_status=status)
    if review_type == "offer" and (
        candidate.get("billing_mode") != "terms"
        or candidate.get("commitment_event_type") not in {"terms_acceptance", "terms_renewal"}
    ):
        raise OperatorError("invalid_response", http_status=status)
    return {
        "ok": True,
        "action": "review_candidate",
        "http_status": status,
        "review_candidate": {field: candidate[field] for field in sorted(candidate)},
        "commercial_proof_created": False,
    }


def _project_record_review(
    status: int,
    document: Mapping[str, object],
    pilot_id: str,
    review_type: str,
    subject_id: str,
    expected_digest: str,
    owner_reference: str,
    evidence_reference: str,
) -> dict[str, object]:
    if (
        set(document)
        != {"review", "created", "idempotent_replay", "commercial_proof_created", "evidence_scope"}
        or document.get("commercial_proof_created") is not False
        or document.get("evidence_scope") != PILOT_REVIEW_EVIDENCE_SCOPE
    ):
        raise OperatorError("invalid_response", http_status=status)
    created = document.get("created")
    replay = document.get("idempotent_replay")
    if (
        not isinstance(created, bool)
        or not isinstance(replay, bool)
        or created == replay
        or (created and status != 201)
        or (replay and status != 200)
    ):
        raise OperatorError("invalid_response", http_status=status)
    review = document.get("review")
    common = {
        "id", "provider_pilot_epoch_id", "review_contract_version", "review_type", "subject_id",
        "provider_claim_id", "subject_snapshot_sha256", "owner_reference", "evidence_reference",
        "reviewed_at",
    }
    relation_fields = {
        "provider": set(),
        "offer": {"provider_offer_id"},
        "ticket": {"provider_offer_id", "action_ticket_id"},
        "handoff": {"provider_offer_id", "action_ticket_id", "handoff_receipt_id"},
        "callback": {
            "provider_offer_id", "action_ticket_id", "handoff_receipt_id", "outcome_receipt_id",
        },
    }[review_type]
    expected_fields = common | relation_fields
    if not isinstance(review, dict) or set(review) != expected_fields:
        raise OperatorError("invalid_response", http_status=status)
    _validate_review_fields(review)
    if (
        review.get("review_contract_version") != PILOT_REVIEW_CONTRACT_VERSION
        or review.get("provider_pilot_epoch_id") != pilot_id
        or review.get("review_type") != review_type
        or review.get("subject_id") != subject_id
        or review.get("subject_snapshot_sha256") != expected_digest
        or review.get("owner_reference") != owner_reference
        or review.get("evidence_reference") != evidence_reference
    ):
        raise OperatorError("invalid_response", http_status=status)
    subject_fields = {
        "provider": "provider_claim_id",
        "offer": "provider_offer_id",
        "ticket": "action_ticket_id",
        "handoff": "handoff_receipt_id",
        "callback": "outcome_receipt_id",
    }
    if review.get(subject_fields[review_type]) != subject_id:
        raise OperatorError("invalid_response", http_status=status)
    return {
        "ok": True,
        "action": "record_review",
        "http_status": status,
        "created": created,
        "idempotent_replay": replay,
        "review_id": review["id"],
        "pilot_id": pilot_id,
        "review_type": review_type,
        "subject_id": subject_id,
        "subject_snapshot_sha256": expected_digest,
        "reviewed_at": review["reviewed_at"],
        "commercial_proof_created": False,
    }


def run(
    argv: Sequence[str],
    *,
    environ: Mapping[str, str] | None = None,
    opener=None,
) -> dict[str, object]:
    args = build_parser().parse_args(list(argv))
    read_only_commands = {"queue", "review-preflight", "status-pilot", "review-candidate"}
    if args.command not in read_only_commands and not getattr(args, "confirm_owner_authorized", False):
        raise OperatorError("owner_authorization_required")
    environment = os.environ if environ is None else environ
    key = admin_key(environment)
    client = build_opener() if opener is None else opener

    if args.command in {"queue", "review-preflight"}:
        if not 1 <= args.limit <= 100:
            raise OperatorError("invalid_limit")
        state = args.state if args.command == "queue" else "pre_event_review_required"
        query = urllib.parse.urlencode({"state": state, "limit": str(args.limit)})
        request = build_request(QUEUE_PATH + "?" + query, key)
        status, document = perform_request(client, request)
        if status != 200:
            raise OperatorError("invalid_response", http_status=status)
        return _project_queue(
            document,
            state,
            args.limit,
            action="queue" if args.command == "queue" else "review_preflight",
        )

    if args.command == "status-pilot":
        pilot_id = _uuid(args.pilot_id, "invalid_pilot_id")
        query = urllib.parse.urlencode({"pilot_id": pilot_id})
        request = build_request(PILOT_EPOCH_PATH + "?" + query, key)
        status, document = perform_request(client, request)
        return _project_pilot_status(status, document, pilot_id)

    if args.command == "issue-proof-manifest":
        pilot_id = _uuid(args.pilot_id, "invalid_pilot_id")
        expected_digest = _hash(
            args.expected_snapshot_sha256, "invalid_expected_snapshot_sha256"
        )
        owner_reference = _reference(args.owner_reference, "invalid_owner_reference")
        evidence_reference = _reference(args.evidence_reference, "invalid_evidence_reference")
        request = build_request(
            PROOF_MANIFEST_PATH,
            key,
            payload={
                "provider_pilot_epoch_id": pilot_id,
                "expected_snapshot_sha256": expected_digest,
                "owner_reference": owner_reference,
                "evidence_reference": evidence_reference,
            },
        )
        status, document = perform_request(client, request)
        return _project_issue_proof_manifest(status, document, pilot_id, expected_digest)

    if args.command in {"review-candidate", "record-review"}:
        pilot_id = _uuid(args.pilot_id, "invalid_pilot_id")
        review_type = _review_type(args.review_type)
        subject_id = _uuid(args.subject_id, "invalid_subject_id")
        if args.command == "review-candidate":
            query = urllib.parse.urlencode(
                {"pilot_id": pilot_id, "review_type": review_type, "subject_id": subject_id}
            )
            request = build_request(PILOT_REVIEW_PATH + "?" + query, key)
            status, document = perform_request(client, request)
            return _project_review_candidate(status, document, pilot_id, review_type, subject_id)

        expected_digest = _hash(
            args.expected_snapshot_sha256, "invalid_expected_snapshot_sha256"
        )
        owner_reference = _reference(args.owner_reference, "invalid_owner_reference")
        evidence_reference = _reference(args.evidence_reference, "invalid_evidence_reference")
        request = build_request(
            PILOT_REVIEW_PATH,
            key,
            payload={
                "provider_pilot_epoch_id": pilot_id,
                "review_type": review_type,
                "subject_id": subject_id,
                "expected_snapshot_sha256": expected_digest,
                "owner_reference": owner_reference,
                "evidence_reference": evidence_reference,
            },
        )
        status, document = perform_request(client, request)
        return _project_record_review(
            status,
            document,
            pilot_id,
            review_type,
            subject_id,
            expected_digest,
            owner_reference,
            evidence_reference,
        )

    if args.command == "authorize-pilot":
        topic = _pilot_topic(args.topic)
        if not PILOT_MINIMUM_COHORT <= args.cohort_limit <= PILOT_MAXIMUM_COHORT:
            raise OperatorError("invalid_cohort_limit")
        if not 1 <= args.provider_ticket_cap <= PILOT_MAXIMUM_PROVIDER_TICKETS:
            raise OperatorError("invalid_provider_ticket_cap")
        if (
            not PILOT_MINIMUM_TOTAL_TICKETS
            <= args.total_ticket_cap
            <= PILOT_MAXIMUM_TOTAL_TICKETS
            or args.total_ticket_cap < args.cohort_limit
        ):
            raise OperatorError("invalid_total_ticket_cap")
        owner_reference = _reference(args.owner_reference, "invalid_owner_reference")
        evidence_reference = _reference(args.evidence_reference, "invalid_evidence_reference")
        request = build_request(
            PILOT_ACTION_PATH,
            key,
            payload={
                "action": "authorize",
                "demand_topic": topic,
                "cohort_limit": args.cohort_limit,
                "provider_ticket_cap": args.provider_ticket_cap,
                "total_ticket_cap": args.total_ticket_cap,
                "owner_reference": owner_reference,
                "evidence_reference": evidence_reference,
            },
        )
        status, document = perform_request(client, request)
        return _project_pilot_epoch_mutation(status, document, "authorize")

    if args.command == "enroll-pilot":
        pilot_id = _uuid(args.pilot_id, "invalid_pilot_id")
        claim_id = _uuid(args.claim_id, "invalid_claim_id")
        owner_reference = _reference(args.owner_reference, "invalid_owner_reference")
        evidence_reference = _reference(args.evidence_reference, "invalid_evidence_reference")
        request = build_request(
            PILOT_ACTION_PATH,
            key,
            payload={
                "action": "enroll",
                "provider_pilot_epoch_id": pilot_id,
                "provider_claim_id": claim_id,
                "owner_reference": owner_reference,
                "evidence_reference": evidence_reference,
            },
        )
        status, document = perform_request(client, request)
        return _project_pilot_enrollment(status, document, pilot_id, claim_id)

    if args.command in {"activate-pilot", "close-pilot"}:
        pilot_id = _uuid(args.pilot_id, "invalid_pilot_id")
        owner_reference = _reference(args.owner_reference, "invalid_owner_reference")
        evidence_reference = _reference(args.evidence_reference, "invalid_evidence_reference")
        action = "activate" if args.command == "activate-pilot" else "close"
        request = build_request(
            PILOT_ACTION_PATH,
            key,
            payload={
                "action": action,
                "provider_pilot_epoch_id": pilot_id,
                "owner_reference": owner_reference,
                "evidence_reference": evidence_reference,
            },
        )
        status, document = perform_request(client, request)
        return _project_pilot_epoch_mutation(
            status,
            document,
            action,
            expected_pilot_id=pilot_id,
        )

    offer_id = _uuid(args.offer_id, "invalid_offer_id")
    operator_reference = _reference(args.operator_reference, "invalid_operator_reference")

    if args.command == "verify-terms":
        acceptance_id = _uuid(args.provider_acceptance_event_id, "invalid_provider_acceptance_event_id")
        related_id = _optional_uuid(args.related_commitment_event_id, "invalid_related_commitment_event_id")
        source_system = _source_system(args.source_system)
        source_event_id = _reference(args.source_event_id, "invalid_source_event_id")
        source_effective_at = _timestamp(args.source_effective_at, "invalid_source_effective_at")
        evidence_reference = _reference(args.owner_evidence_reference, "invalid_owner_evidence_reference")
        payload: dict[str, object] = {
            "action": "verify_terms",
            "offer_id": offer_id,
            "provider_acceptance_event_id": acceptance_id,
            "source_system": source_system,
            "source_event_id": source_event_id,
            "source_effective_at": source_effective_at,
            "operator_reference": operator_reference,
            "owner_evidence_reference": evidence_reference,
        }
        if related_id:
            payload["related_commitment_event_id"] = related_id
        request = build_request(COMMERCIAL_ACTION_PATH, key, payload=payload)
        status, document = perform_request(client, request)
        return _project_verify_terms(status, document)

    evidence_reference = _reference(args.evidence_reference, "invalid_evidence_reference")
    request = build_request(
        OFFER_ACTION_PATH,
        key,
        payload={
            "action": args.command,
            "offer_id": offer_id,
            "operator_reference": operator_reference,
            "evidence_reference": evidence_reference,
        },
    )
    status, document = perform_request(client, request)
    return _project_offer_action(status, document, args.command, offer_id)


def _emit(document: Mapping[str, object], stream) -> None:
    stream.write(json.dumps(dict(document), sort_keys=True, separators=(",", ":")) + "\n")


def main(argv: Sequence[str] | None = None) -> int:
    try:
        disable_core_dumps()
        receipt = run(sys.argv[1:] if argv is None else argv)
    except OperatorError as error:
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
