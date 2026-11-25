#!/usr/bin/env python3
"""Hardened provider-side client for the NHS terms-only pilot.

The client is deliberately fixed to the NHS production origin.  It obtains the
claim-scoped provider key only from ``NHS_PROVIDER_PILOT_KEY`` and obtains an
attribution bearer only from bounded standard input (or an echo-disabled TTY
prompt).  Neither secret is ever included in output or an error message.
"""

from __future__ import annotations

import argparse
import datetime
import getpass
import hmac
import json
import os
import re
import resource
import ssl
import sys
import urllib.error
import urllib.parse
import urllib.request
from typing import IO, Mapping, Sequence


BASE_URL = "https://nothumansearch.ai"
PROVIDER_KEY_ENV = "NHS_PROVIDER_PILOT_KEY"
MAX_RESPONSE_BYTES = 64 * 1024
MAX_ATTRIBUTION_TOKEN_BYTES = 8 * 1024
REQUEST_TIMEOUT_SECONDS = 10
PROVIDER_DEMAND_PRIVACY_THRESHOLD = 20

EXIT_OK = 0
EXIT_INPUT = 2
EXIT_NETWORK = 3
EXIT_AUTH = 4
EXIT_RATE_LIMIT = 5
EXIT_REMOTE = 6
EXIT_RESPONSE = 7
EXIT_INTERNAL = 70
EXIT_INTERRUPTED = 130

_UUID_PATTERN = re.compile(
    r"^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"
)
_HASH_PATTERN = re.compile(r"^[0-9a-f]{64}$")
_REFERENCE_PATTERN = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._:/-]{7,199}$")
_IDEMPOTENCY_PATTERN = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$")
_SIGNATURE_PATTERN = re.compile(r"^[A-Za-z0-9_-]{32,128}$")
_KEY_ID_PATTERN = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{2,63}$")
_REGION_PATTERN = re.compile(r"^[A-Z]{2}(?:-[A-Z0-9]{1,3})?$")
_CURRENCY_PATTERN = re.compile(r"^[a-z]{3}$")
_DOMAIN_LABEL_PATTERN = re.compile(r"^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$")

_OUTCOMES = frozenset({"accepted", "activated", "converted", "rejected", "duplicate", "invalid"})
_ACTION_TYPES = frozenset({"lead", "demo", "trial", "signup", "purchase", "quote", "application", "booking"})
_CHARGE_EVENTS = frozenset({"accepted", "activated", "converted"})
_OFFER_STATUSES = frozenset({"draft", "active", "paused"})
_TICKET_STATUSES = frozenset(
    {"created", "redirected", "accepted", "activated", "converted", "rejected", "duplicate", "invalid", "expired", "revoked"}
)
_CHARGE_STATUSES = frozenset({"none", "charged", "credited"})
_DEMAND_TOPICS = frozenset(
    {
        "payments", "commerce", "jobs", "data", "search", "weather", "maps", "email",
        "messaging", "image", "video", "audio", "documents", "security", "finance", "health",
        "education", "news", "analytics", "automation", "productivity", "identity", "storage",
        "ai-tools", "developer-tools", "other",
    }
)
_BUDGET_BANDS = frozenset({"unspecified", "under_100", "100_499", "500_1999", "2000_plus"})
_URGENCIES = frozenset({"unspecified", "now", "7_days", "30_days", "researching"})
_REQUIREMENT_FLAGS = frozenset(
    {"api_access", "mcp", "sandbox", "self_serve", "enterprise", "compliance", "multilingual", "human_support"}
)
_SURFACES = frozenset({"web", "rest", "mcp", "unknown"})


class ClientError(Exception):
    """A stable public error code that cannot contain sensitive data."""

    def __init__(self, code: str, *, http_status: int | None = None):
        super().__init__(code)
        self.code = code
        self.http_status = http_status


class SafeArgumentParser(argparse.ArgumentParser):
    """Do not reflect malformed arguments, which may contain pasted secrets."""

    def error(self, _message: str) -> None:
        raise ClientError("invalid_arguments")


class NoRedirect(urllib.request.HTTPRedirectHandler):
    """Refuse every redirect, including same-origin redirects."""

    def redirect_request(self, req, fp, code, msg, headers, newurl):  # noqa: ANN001
        return None


def build_parser() -> argparse.ArgumentParser:
    parser = SafeArgumentParser(
        description="Operate one claim-scoped NHS provider pilot safely against the fixed production host."
    )
    subparsers = parser.add_subparsers(dest="command", required=True)

    accept_company = subparsers.add_parser("accept-company", help="Accept pilot-company participation.")
    accept_company.add_argument("--provider-acceptance-reference", required=True)
    accept_company.add_argument("--idempotency-key", required=True)
    accept_company.add_argument("--confirm-provider-authorized", action="store_true")

    def add_terms_arguments(command_parser: argparse.ArgumentParser, *, renewal: bool) -> None:
        command_parser.add_argument("--offer-id", required=True)
        command_parser.add_argument("--offer-version", required=True, type=int)
        command_parser.add_argument("--exact-terms-sha256", required=True)
        command_parser.add_argument("--provider-acceptance-reference", required=True)
        command_parser.add_argument("--idempotency-key", required=True)
        command_parser.add_argument("--confirm-provider-authorized", action="store_true")
        if renewal:
            command_parser.add_argument("--related-acceptance-event-id", required=True)

    add_terms_arguments(
        subparsers.add_parser("accept-terms", help="Accept the exact current terms for one draft offer."),
        renewal=False,
    )
    add_terms_arguments(
        subparsers.add_parser("renew-terms", help="Renew one exact accepted-terms chain."),
        renewal=True,
    )

    status = subparsers.add_parser("status", help="Read claim-scoped provider continuity status.")
    status.add_argument("--limit", type=int, default=25)

    demand = subparsers.add_parser("demand", help="Read privacy-thresholded claim-scoped demand.")
    demand.add_argument("--days", type=int, default=30)

    subparsers.add_parser(
        "resolve",
        help="Resolve separately consented controlled intent; bearer is read from stdin/TTY.",
    )

    outcome = subparsers.add_parser(
        "outcome",
        help="Report one provider outcome; bearer is read from stdin/TTY.",
    )
    outcome.add_argument("--outcome", required=True, choices=tuple(sorted(_OUTCOMES)))
    outcome.add_argument("--idempotency-key", required=True)
    outcome.add_argument("--confirm-provider-authorized", action="store_true")
    outcome.add_argument(
        "--ticket-id",
        help="Optional compatibility assertion; the signed bearer remains authoritative.",
    )

    receipt = subparsers.add_parser("receipt", help="Retrieve one provider-owned signed outcome receipt.")
    receipt.add_argument("--receipt-id", required=True)
    return parser


def disable_core_dumps() -> None:
    try:
        resource.setrlimit(resource.RLIMIT_CORE, (0, 0))
    except (OSError, ValueError):
        raise ClientError("core_dump_hardening_unavailable") from None


def provider_key(environ: Mapping[str, str]) -> str:
    value = environ.get(PROVIDER_KEY_ENV, "")
    if not isinstance(value, str):
        raise ClientError("provider_key_unavailable")
    try:
        encoded = value.encode("ascii", "strict")
    except UnicodeEncodeError:
        raise ClientError("provider_key_unavailable") from None
    if not 8 <= len(encoded) <= 4096 or any(byte <= 0x20 or byte == 0x7F for byte in encoded):
        raise ClientError("provider_key_unavailable")
    return value


def _validate_attribution_token(value: object) -> str:
    if not isinstance(value, str):
        raise ClientError("attribution_bearer_unavailable")
    try:
        encoded = value.encode("ascii", "strict")
    except UnicodeEncodeError:
        raise ClientError("attribution_bearer_unavailable") from None
    if not 16 <= len(encoded) <= MAX_ATTRIBUTION_TOKEN_BYTES:
        raise ClientError("attribution_bearer_unavailable")
    if any(byte <= 0x20 or byte == 0x7F for byte in encoded):
        raise ClientError("attribution_bearer_unavailable")
    return value


def read_attribution_token(stream: IO[str] | None = None) -> str:
    """Read one bearer without accepting it on the process command line."""

    source = sys.stdin if stream is None else stream
    try:
        interactive = bool(source.isatty())
    except (AttributeError, OSError):
        interactive = False
    try:
        if interactive:
            raw = getpass.getpass("NHS attribution bearer: ", stream=sys.stderr)
        else:
            raw = source.read(MAX_ATTRIBUTION_TOKEN_BYTES + 2)
    except (EOFError, OSError, UnicodeError):
        raise ClientError("attribution_bearer_unavailable") from None
    if not isinstance(raw, str):
        raise ClientError("attribution_bearer_unavailable")
    # A single conventional pipe newline is transport framing, not token data.
    if raw.endswith("\n"):
        raw = raw[:-1]
        if raw.endswith("\r"):
            raw = raw[:-1]
    return _validate_attribution_token(raw)


def build_opener():
    """Create a TLS-verifying opener with proxy inheritance and redirects disabled."""

    return urllib.request.build_opener(
        urllib.request.ProxyHandler({}),
        NoRedirect(),
        urllib.request.HTTPSHandler(context=ssl.create_default_context()),
    )


def _valid_provider_path(method: str, path: str) -> bool:
    if method == "POST":
        return path in {
            "/api/v1/provider/commercial-acceptances",
            "/api/v1/provider/action-tickets/resolve",
            "/api/v1/provider/outcomes",
        }
    if method != "GET":
        return False
    parsed = urllib.parse.urlsplit(path)
    if parsed.scheme or parsed.netloc or parsed.fragment:
        return False
    try:
        query = (
            urllib.parse.parse_qs(parsed.query, strict_parsing=True)
            if parsed.query
            else {}
        )
    except ValueError:
        return False
    if parsed.path == "/api/v1/provider/pilot-status":
        return set(query) <= {"limit"} and all(len(values) == 1 for values in query.values())
    if parsed.path == "/api/v1/provider/demand":
        return set(query) <= {"days"} and all(len(values) == 1 for values in query.values())
    if parsed.query:
        return False
    prefix = "/api/v1/provider/receipts/"
    return parsed.path.startswith(prefix) and bool(_UUID_PATTERN.fullmatch(parsed.path[len(prefix) :]))


def build_request(
    method: str,
    path: str,
    key: str,
    *,
    payload: Mapping[str, object] | None = None,
    idempotency_key: str | None = None,
) -> urllib.request.Request:
    method = method.upper()
    if not _valid_provider_path(method, path):
        raise ClientError("invalid_request_path")
    headers = {
        "Accept": "application/json",
        "X-NHS-Provider-Key": key,
        "User-Agent": "NHS-Provider-Pilot-Client/1.0",
    }
    body = None
    if payload is not None:
        if method != "POST":
            raise ClientError("invalid_request_shape")
        try:
            body = json.dumps(dict(payload), sort_keys=True, separators=(",", ":")).encode("ascii")
        except (TypeError, UnicodeEncodeError, ValueError):
            raise ClientError("invalid_request_shape") from None
        headers["Content-Type"] = "application/json"
    if idempotency_key is not None:
        headers["Idempotency-Key"] = idempotency_key
    return urllib.request.Request(BASE_URL + path, data=body, method=method, headers=headers)


def perform_request(
    opener,
    request: urllib.request.Request,
    *,
    success_statuses: frozenset[int],
) -> dict[str, object]:
    status = None
    try:
        with opener.open(request, timeout=REQUEST_TIMEOUT_SECONDS) as response:
            status = int(response.getcode())
            if 300 <= status <= 399:
                raise ClientError("redirect_refused", http_status=status)
            if status not in success_statuses:
                raise ClientError("http_error", http_status=status)
            body = response.read(MAX_RESPONSE_BYTES + 1)
    except urllib.error.HTTPError as error:
        safe_status = error.code if isinstance(error.code, int) and 100 <= error.code <= 599 else None
        code = "redirect_refused" if safe_status is not None and 300 <= safe_status <= 399 else "http_error"
        # Intentionally do not read the response body or reflect its reason.
        raise ClientError(code, http_status=safe_status) from None
    except (urllib.error.URLError, TimeoutError, OSError, ssl.SSLError):
        raise ClientError("network_error") from None

    if len(body) > MAX_RESPONSE_BYTES:
        raise ClientError("response_too_large", http_status=status)
    try:
        document = json.loads(body.decode("utf-8", "strict"))
    except (UnicodeDecodeError, json.JSONDecodeError):
        raise ClientError("invalid_response", http_status=status) from None
    if not isinstance(document, dict):
        raise ClientError("invalid_response", http_status=status)
    return document


def _mapping(value: object) -> Mapping[str, object]:
    if not isinstance(value, dict):
        raise ClientError("invalid_response")
    return value


def _string(value: object, *, minimum: int = 1, maximum: int = 4096) -> str:
    if not isinstance(value, str) or not minimum <= len(value) <= maximum:
        raise ClientError("invalid_response")
    if any(ord(character) < 0x20 or ord(character) == 0x7F for character in value):
        raise ClientError("invalid_response")
    return value


def _enum(value: object, allowed: frozenset[str]) -> str:
    result = _string(value, maximum=128)
    if result not in allowed:
        raise ClientError("invalid_response")
    return result


def _boolean(value: object) -> bool:
    if not isinstance(value, bool):
        raise ClientError("invalid_response")
    return value


def _integer(value: object, low: int = 0, high: int = 2**63 - 1) -> int:
    if isinstance(value, bool) or not isinstance(value, int) or not low <= value <= high:
        raise ClientError("invalid_response")
    return value


def _number(value: object, low: float = 0.0, high: float = 1_000_000_000.0) -> float | int:
    if isinstance(value, bool) or not isinstance(value, (int, float)) or not low <= value <= high:
        raise ClientError("invalid_response")
    return value


def _uuid(value: object) -> str:
    result = _string(value, maximum=36)
    if not _UUID_PATTERN.fullmatch(result):
        raise ClientError("invalid_response")
    return result


def _timestamp(value: object) -> str:
    result = _string(value, maximum=64)
    try:
        parsed = datetime.datetime.fromisoformat(result.replace("Z", "+00:00"))
    except ValueError:
        raise ClientError("invalid_response") from None
    if parsed.tzinfo is None or parsed.utcoffset() is None:
        raise ClientError("invalid_response")
    return result


def _domain(value: object) -> str:
    result = _string(value, maximum=253)
    if result != result.lower() or ":" in result or "/" in result or result.endswith("."):
        raise ClientError("invalid_response")
    labels = result.split(".")
    if len(labels) < 2 or any(not _DOMAIN_LABEL_PATTERN.fullmatch(label) for label in labels):
        raise ClientError("invalid_response")
    return result


def _nullable(value: object, validator):  # noqa: ANN001
    return None if value is None else validator(value)


def _validate_reference_argument(value: str, code: str) -> str:
    if not isinstance(value, str) or not _REFERENCE_PATTERN.fullmatch(value):
        raise ClientError(code)
    return value


def _validate_idempotency_argument(value: str) -> str:
    if not isinstance(value, str) or not _IDEMPOTENCY_PATTERN.fullmatch(value):
        raise ClientError("invalid_idempotency_key")
    return value


def _validate_uuid_argument(value: str | None, code: str, *, optional: bool = False) -> str:
    if optional and value is None:
        return ""
    if not isinstance(value, str):
        raise ClientError(code)
    normalized = value.lower()
    if value != normalized or not _UUID_PATTERN.fullmatch(normalized):
        raise ClientError(code)
    return normalized


def _project_acceptance(
    document: Mapping[str, object],
    *,
    expected_event: str,
    expected_offer_id: str = "",
    expected_version: int = 0,
    expected_hash: str = "",
    expected_related_id: str = "",
    expected_reference: str,
) -> dict[str, object]:
    raw = _mapping(document.get("acceptance"))
    event_type = _enum(raw.get("event_type"), frozenset({"pilot_company", "terms_acceptance", "terms_renewal"}))
    if event_type != expected_event:
        raise ClientError("invalid_response")
    reference = _string(raw.get("provider_acceptance_reference"), maximum=200)
    if not hmac.compare_digest(reference, expected_reference):
        raise ClientError("invalid_response")
    projected: dict[str, object] = {
        "id": _uuid(raw.get("id")),
        "provider_claim_id": _uuid(raw.get("provider_claim_id")),
        "event_type": event_type,
        "provider_acceptance_reference": reference,
        "provider_accepted_at": _timestamp(raw.get("provider_accepted_at")),
        "created_at": _timestamp(raw.get("created_at")),
    }
    _integer(raw.get("provider_api_key_id"), 1)
    if expected_event == "pilot_company":
        for field in (
            "provider_offer_id", "related_acceptance_event_id", "offer_version",
            "terms_contract_version", "exact_terms_sha256", "valid_until",
        ):
            if raw.get(field) not in (None, "", 0):
                raise ClientError("invalid_response")
    else:
        offer_id = _uuid(raw.get("provider_offer_id"))
        version = _integer(raw.get("offer_version"), 1, 2**31 - 1)
        terms_hash = _string(raw.get("exact_terms_sha256"), minimum=64, maximum=64)
        if (
            not hmac.compare_digest(offer_id, expected_offer_id)
            or version != expected_version
            or not hmac.compare_digest(terms_hash, expected_hash)
            or raw.get("terms_contract_version") != "nhs-provider-commercial-terms-v1"
        ):
            raise ClientError("invalid_response")
        projected.update(
            {
                "provider_offer_id": offer_id,
                "offer_version": version,
                "terms_contract_version": "nhs-provider-commercial-terms-v1",
                "exact_terms_sha256": terms_hash,
                "valid_until": _timestamp(raw.get("valid_until")),
            }
        )
        related = raw.get("related_acceptance_event_id")
        if expected_event == "terms_renewal":
            related_id = _uuid(related)
            if not hmac.compare_digest(related_id, expected_related_id):
                raise ClientError("invalid_response")
            projected["related_acceptance_event_id"] = related_id
        elif related not in (None, ""):
            raise ClientError("invalid_response")

    created = _boolean(document.get("created"))
    replayed = _boolean(document.get("idempotent_replay"))
    if created == replayed:
        raise ClientError("invalid_response")
    if (
        _boolean(document.get("provider_authenticated")) is not True
        or _boolean(document.get("owner_verification_required")) is not True
        or _boolean(document.get("commercial_proof_created")) is not False
    ):
        raise ClientError("invalid_response")
    _string(document.get("evidence_scope"), maximum=1000)
    return {
        "acceptance": projected,
        "created": created,
        "idempotent_replay": replayed,
        "provider_authenticated": True,
        "owner_verification_required": True,
        "commercial_proof_created": False,
    }


def _project_offer_status(value: object) -> dict[str, object]:
    raw = _mapping(value)
    projected: dict[str, object] = {
        "offer_id": _uuid(raw.get("offer_id")),
        "status": _enum(raw.get("status"), _OFFER_STATUSES),
        "version": _integer(raw.get("version"), 1, 2**31 - 1),
        "name": _string(raw.get("name"), maximum=80),
        "action_type": _enum(raw.get("action_type"), _ACTION_TYPES),
        "charge_event": _enum(raw.get("charge_event"), _CHARGE_EVENTS),
        "bounty_cents": _integer(raw.get("bounty_cents"), 1, 1_000_000),
        "currency": _enum(raw.get("currency"), frozenset({"usd"})),
        "billing_mode": _enum(raw.get("billing_mode"), frozenset({"terms"})),
        "commercial_terms_contract_version": _enum(
            raw.get("commercial_terms_contract_version"), frozenset({"nhs-provider-commercial-terms-v1"})
        ),
        "commercial_terms_sha256": _string(raw.get("commercial_terms_sha256"), minimum=64, maximum=64),
        "provider_mor_acknowledgement_required": _boolean(raw.get("provider_mor_acknowledgement_required")),
        "provider_acknowledges_merchant_of_record": _boolean(raw.get("provider_acknowledges_merchant_of_record")),
        "latest_acceptance_owner_verified": _boolean(raw.get("latest_acceptance_owner_verified")),
        "current_terms_owner_verified": _boolean(raw.get("current_terms_owner_verified")),
        "renewal_eligible": _boolean(raw.get("renewal_eligible")),
        "activation_ready": _boolean(raw.get("activation_ready")),
    }
    if not _HASH_PATTERN.fullmatch(projected["commercial_terms_sha256"]):
        raise ClientError("invalid_response")
    if projected["provider_mor_acknowledgement_required"] is not True:
        raise ClientError("invalid_response")
    optional_validators = {
        "terms_credit_limit_cents": lambda item: _integer(item, 1, 10_000_000),
        "terms_period_days": lambda item: _integer(item, 1, 90),
        "latest_acceptance_id": _uuid,
        "latest_acceptance_type": lambda item: _enum(
            item, frozenset({"terms_acceptance", "terms_renewal"})
        ),
        "latest_acceptance_at": _timestamp,
        "latest_acceptance_valid_until": _timestamp,
        "latest_acceptance_owner_verified_at": _timestamp,
        "current_terms_valid_until": _timestamp,
    }
    for field, validator in optional_validators.items():
        if field in raw and raw[field] is not None:
            projected[field] = validator(raw[field])
    if projected["latest_acceptance_owner_verified"] != ("latest_acceptance_owner_verified_at" in projected):
        raise ClientError("invalid_response")
    if projected["current_terms_owner_verified"] != ("current_terms_valid_until" in projected):
        raise ClientError("invalid_response")
    if projected["activation_ready"] and (
        projected["status"] != "draft" or not projected["current_terms_owner_verified"]
    ):
        raise ClientError("invalid_response")
    return projected


def _project_recent_event(value: object) -> dict[str, object]:
    raw = _mapping(value)
    projected: dict[str, object] = {
        "ticket_id": _uuid(raw.get("ticket_id")),
        "offer_id": _uuid(raw.get("offer_id")),
        "offer_version": _integer(raw.get("offer_version"), 1, 2**31 - 1),
        "ticket_status": _enum(raw.get("ticket_status"), _TICKET_STATUSES),
        "handoff_receipt_id": _uuid(raw.get("handoff_receipt_id")),
        "handoff_observed_at": _timestamp(raw.get("handoff_observed_at")),
    }
    optional_validators = {
        "outcome_receipt_id": _uuid,
        "outcome": lambda item: _enum(item, _OUTCOMES),
        "charge_status": lambda item: _enum(item, _CHARGE_STATUSES),
        "billed_cents": lambda item: _integer(item, 0, 100_000_000),
        "outcome_recorded_at": _timestamp,
    }
    for field, validator in optional_validators.items():
        if field in raw and raw[field] is not None:
            projected[field] = validator(raw[field])
    outcome_fields = {"outcome_receipt_id", "outcome", "charge_status", "billed_cents", "outcome_recorded_at"}
    present = outcome_fields & set(projected)
    if present and present != outcome_fields:
        raise ClientError("invalid_response")
    return projected


def project_status(document: Mapping[str, object], expected_limit: int) -> dict[str, object]:
    raw = _mapping(document.get("pilot_status"))
    offers_raw = raw.get("offers")
    events_raw = raw.get("recent_observed_handoffs")
    if not isinstance(offers_raw, list) or not isinstance(events_raw, list):
        raise ClientError("invalid_response")
    if len(offers_raw) > expected_limit or len(events_raw) > expected_limit:
        raise ClientError("invalid_response")
    projected: dict[str, object] = {
        "as_of": _timestamp(raw.get("as_of")),
        "provider_claim_id": _uuid(raw.get("provider_claim_id")),
        "domain": _domain(raw.get("domain")),
        "claim_status": _enum(raw.get("claim_status"), frozenset({"verified"})),
        "verification_last_succeeded_at": _timestamp(raw.get("verification_last_succeeded_at")),
        "verification_consecutive_failures": _integer(raw.get("verification_consecutive_failures"), 0, 1_000_000),
        "company_owner_verified": _boolean(raw.get("company_owner_verified")),
        "offers": [_project_offer_status(item) for item in offers_raw],
        "recent_observed_handoffs": [_project_recent_event(item) for item in events_raw],
    }
    optional_validators = {
        "verification_next_check_at": _timestamp,
        "company_acceptance_id": _uuid,
        "company_accepted_at": _timestamp,
        "company_owner_verified_at": _timestamp,
    }
    for field, validator in optional_validators.items():
        if field in raw and raw[field] is not None:
            projected[field] = validator(raw[field])
    acceptance_fields = {"company_acceptance_id", "company_accepted_at"}
    if bool(acceptance_fields & set(projected)) != acceptance_fields.issubset(projected):
        raise ClientError("invalid_response")
    if projected["company_owner_verified"] != ("company_owner_verified_at" in projected):
        raise ClientError("invalid_response")
    if projected["company_owner_verified"] and not acceptance_fields.issubset(projected):
        raise ClientError("invalid_response")
    _string(document.get("evidence_scope"), maximum=1500)
    return projected


def _project_suppressed_count(raw: Mapping[str, object], count_key: str, suppressed_key: str) -> tuple[int | None, bool]:
    suppressed = _boolean(raw.get(suppressed_key))
    count = _nullable(raw.get(count_key), lambda item: _integer(item, PROVIDER_DEMAND_PRIVACY_THRESHOLD))
    if suppressed != (count is None):
        raise ClientError("invalid_response")
    return count, suppressed


def _project_demand_summary(value: object) -> dict[str, object]:
    raw = _mapping(value)
    selections, selection_suppressed = _project_suppressed_count(
        raw, "result_selections", "result_selection_suppressed"
    )
    interests, interest_suppressed = _project_suppressed_count(
        raw, "action_interest_receipts", "action_interest_suppressed"
    )
    search_receipts = _integer(raw.get("search_receipts"))
    projected = {
        "organic_results_returned": _integer(raw.get("organic_results_returned")),
        "search_receipts": search_receipts,
        "average_organic_position": _number(raw.get("average_organic_position")),
        "result_selections": selections,
        "result_selection_suppressed": selection_suppressed,
        "result_selection_rate": _nullable(raw.get("result_selection_rate"), lambda item: _number(item, 0, 1)),
        "action_interest_receipts": interests,
        "action_interest_rate": _nullable(raw.get("action_interest_rate"), lambda item: _number(item, 0, 1)),
        "action_interest_suppressed": interest_suppressed,
    }
    if selection_suppressed != (projected["result_selection_rate"] is None):
        raise ClientError("invalid_response")
    if interest_suppressed != (projected["action_interest_rate"] is None):
        raise ClientError("invalid_response")
    if projected["organic_results_returned"] < search_receipts:
        raise ClientError("invalid_response")
    return projected


def _project_demand_segment(value: object, *, topic: bool) -> dict[str, object]:
    raw = _mapping(value)
    selections, selection_suppressed = _project_suppressed_count(
        raw, "result_selections", "result_selection_suppressed"
    )
    interests, interest_suppressed = _project_suppressed_count(
        raw, "action_interest_receipts", "action_interest_suppressed"
    )
    if topic:
        projected: dict[str, object] = {
            "topic": _enum(raw.get("topic"), _DEMAND_TOPICS),
            "search_receipts": _integer(raw.get("search_receipts"), PROVIDER_DEMAND_PRIVACY_THRESHOLD),
            "average_organic_position": _number(raw.get("average_organic_position")),
        }
    else:
        projected = {
            "surface": _enum(raw.get("surface"), _SURFACES),
            "organic_results_returned": _integer(raw.get("organic_results_returned")),
        }
    projected.update(
        {
            "result_selections": selections,
            "result_selection_suppressed": selection_suppressed,
            "action_interest_receipts": interests,
            "action_interest_suppressed": interest_suppressed,
        }
    )
    return projected


def project_demand(document: Mapping[str, object], expected_days: int) -> dict[str, object]:
    raw = _mapping(document.get("demand"))
    if _integer(raw.get("days"), 1, 30) != expected_days:
        raise ClientError("invalid_response")
    if (
        _integer(raw.get("retention_days"), 1, 30) != 30
        or raw.get("action_interest_cohort") != "organic_result_returned_at"
        or _integer(raw.get("topic_receipt_threshold"), 1) != PROVIDER_DEMAND_PRIVACY_THRESHOLD
        or _integer(raw.get("result_selection_receipt_threshold"), 1) != PROVIDER_DEMAND_PRIVACY_THRESHOLD
        or _integer(raw.get("action_interest_receipt_threshold"), 1) != PROVIDER_DEMAND_PRIVACY_THRESHOLD
        or _boolean(raw.get("synthetic_excluded")) is not True
    ):
        raise ClientError("invalid_response")
    surfaces = raw.get("surfaces")
    topics = raw.get("demand_topics")
    action_types = raw.get("action_types")
    if (
        not isinstance(surfaces, list) or len(surfaces) > len(_SURFACES)
        or not isinstance(topics, list) or len(topics) > 20
        or not isinstance(action_types, list) or len(action_types) > len(_ACTION_TYPES)
    ):
        raise ClientError("invalid_response")
    projected_actions: list[dict[str, object]] = []
    seen_actions: set[str] = set()
    for value in action_types:
        item = _mapping(value)
        action_type = _enum(item.get("action_type"), _ACTION_TYPES)
        if action_type in seen_actions:
            raise ClientError("invalid_response")
        seen_actions.add(action_type)
        projected_actions.append(
            {
                "action_type": action_type,
                "receipt_count": _integer(item.get("receipt_count"), PROVIDER_DEMAND_PRIVACY_THRESHOLD),
            }
        )
    projected_surfaces = [_project_demand_segment(item, topic=False) for item in surfaces]
    projected_topics = [_project_demand_segment(item, topic=True) for item in topics]
    if len({item["surface"] for item in projected_surfaces}) != len(projected_surfaces):
        raise ClientError("invalid_response")
    if len({item["topic"] for item in projected_topics}) != len(projected_topics):
        raise ClientError("invalid_response")
    _string(document.get("evidence_scope"), maximum=1500)
    return {
        "domain": _domain(raw.get("domain")),
        "days": expected_days,
        "retention_days": 30,
        "action_interest_cohort": "organic_result_returned_at",
        "topic_receipt_threshold": PROVIDER_DEMAND_PRIVACY_THRESHOLD,
        "result_selection_receipt_threshold": PROVIDER_DEMAND_PRIVACY_THRESHOLD,
        "action_interest_receipt_threshold": PROVIDER_DEMAND_PRIVACY_THRESHOLD,
        "synthetic_excluded": True,
        "summary": _project_demand_summary(raw.get("summary")),
        "surfaces": projected_surfaces,
        "demand_topics": projected_topics,
        "action_types": projected_actions,
        "counts_are_receipts_not_unique_agents": True,
    }


def project_resolution(document: Mapping[str, object]) -> dict[str, object]:
    raw_intent = _mapping(document.get("controlled_intent"))
    flags = raw_intent.get("requirement_flags")
    if not isinstance(flags, list) or len(flags) > 8:
        raise ClientError("invalid_response")
    projected_flags = [_enum(flag, _REQUIREMENT_FLAGS) for flag in flags]
    if len(set(projected_flags)) != len(projected_flags):
        raise ClientError("invalid_response")
    intent: dict[str, object] = {
        "demand_topic": _enum(raw_intent.get("demand_topic"), _DEMAND_TOPICS),
        "budget_band": _enum(raw_intent.get("budget_band"), _BUDGET_BANDS),
        "urgency": _enum(raw_intent.get("urgency"), _URGENCIES),
        "requirement_flags": projected_flags,
    }
    if "region_code" in raw_intent:
        region = _string(raw_intent.get("region_code"), maximum=6)
        if not _REGION_PATTERN.fullmatch(region):
            raise ClientError("invalid_response")
        intent["region_code"] = region
    return {
        "resolver_contract_version": _enum(
            document.get("resolver_contract_version"), frozenset({"nhs-provider-controlled-intent-resolver-v1"})
        ),
        "ticket_id": _uuid(document.get("ticket_id")),
        "offer_id": _uuid(document.get("offer_id")),
        "offer_version": _integer(document.get("offer_version"), 1, 2**31 - 1),
        "action_type": _enum(document.get("action_type"), _ACTION_TYPES),
        "controlled_intent": intent,
        "observed_at": _timestamp(document.get("observed_at")),
        "intent_available_until": _timestamp(document.get("intent_available_until")),
        "consent_version": _enum(
            document.get("consent_version"), frozenset({"nhs-provider-controlled-intent-disclosure-consent-v1"})
        ),
        "read_only": True,
        "charge_created": False,
        "commercial_proof_created": False,
    }


def _project_signed_receipt(value: object) -> dict[str, object]:
    raw = _mapping(value)
    projected = {
        "id": _uuid(raw.get("id")),
        "nhs_event_id": _uuid(raw.get("nhs_event_id")),
        "provider_claim_id": _uuid(raw.get("provider_claim_id")),
        "provider_offer_id": _uuid(raw.get("provider_offer_id")),
        "action_ticket_id": _uuid(raw.get("action_ticket_id")),
        "outcome": _enum(raw.get("outcome"), _OUTCOMES),
        "billed_cents": _integer(raw.get("billed_cents"), 0, 100_000_000),
        "charge_status": _enum(raw.get("charge_status"), _CHARGE_STATUSES),
        "currency": _string(raw.get("currency"), minimum=3, maximum=3),
        "signed_receipt": _string(raw.get("signed_receipt"), maximum=8192),
        "signature": _string(raw.get("signature"), maximum=128),
        "provider_reported_at": _timestamp(raw.get("provider_reported_at")),
        "created_at": _timestamp(raw.get("created_at")),
    }
    _integer(raw.get("provider_api_key_id"), 1)
    if not _CURRENCY_PATTERN.fullmatch(projected["currency"]) or not _SIGNATURE_PATTERN.fullmatch(projected["signature"]):
        raise ClientError("invalid_response")
    try:
        signed = json.loads(projected["signed_receipt"])
    except json.JSONDecodeError:
        raise ClientError("invalid_response") from None
    signed = _mapping(signed)
    if set(signed) != {
        "v", "kid", "receipt_id", "ticket_id", "offer_id", "nhs_event_id", "outcome",
        "provider_reported_at", "recorded_at", "expires_at", "charged_minor", "currency", "charge_status",
    }:
        raise ClientError("invalid_response")
    if (
        _integer(signed.get("v"), 1, 1) != 1
        or not _KEY_ID_PATTERN.fullmatch(_string(signed.get("kid"), maximum=64))
        or _uuid(signed.get("receipt_id")) != projected["id"]
        or _uuid(signed.get("ticket_id")) != projected["action_ticket_id"]
        or _uuid(signed.get("offer_id")) != projected["provider_offer_id"]
        or _uuid(signed.get("nhs_event_id")) != projected["nhs_event_id"]
        or _enum(signed.get("outcome"), _OUTCOMES) != projected["outcome"]
        or _integer(signed.get("charged_minor"), 0, 100_000_000) != projected["billed_cents"]
        or _string(signed.get("currency"), minimum=3, maximum=3) != projected["currency"]
        or _enum(signed.get("charge_status"), _CHARGE_STATUSES) != projected["charge_status"]
    ):
        raise ClientError("invalid_response")
    reported_unix = _integer(signed.get("provider_reported_at"), 0)
    recorded_unix = _integer(signed.get("recorded_at"), 0)
    expires_unix = _integer(signed.get("expires_at"), 1)
    if recorded_unix < reported_unix or expires_unix <= recorded_unix:
        raise ClientError("invalid_response")
    if projected["charge_status"] == "none" and projected["billed_cents"] != 0:
        raise ClientError("invalid_response")
    if projected["charge_status"] in {"charged", "credited"} and projected["billed_cents"] <= 0:
        raise ClientError("invalid_response")
    return projected


def project_outcome(document: Mapping[str, object], expected_outcome: str) -> dict[str, object]:
    receipt = _project_signed_receipt(document.get("receipt"))
    if receipt["outcome"] != expected_outcome:
        raise ClientError("invalid_response")
    created = _boolean(document.get("created"))
    replayed = _boolean(document.get("idempotent_replay"))
    if created == replayed:
        raise ClientError("invalid_response")
    if (
        _boolean(document.get("principal_charged")) is not False
        or _boolean(document.get("provider_mor_contract_required")) is not True
        or _boolean(document.get("principal_charged_by_nhs")) is not False
    ):
        raise ClientError("invalid_response")
    return {
        "receipt": receipt,
        "created": created,
        "idempotent_replay": replayed,
        "principal_charged": False,
        "provider_mor_contract_required": True,
        "principal_charged_by_nhs": False,
    }


def project_receipt(document: Mapping[str, object], expected_receipt_id: str) -> dict[str, object]:
    receipt = _project_signed_receipt(document.get("receipt"))
    if receipt["id"] != expected_receipt_id:
        raise ClientError("invalid_response")
    return receipt


def run(
    argv: Sequence[str],
    *,
    environ: Mapping[str, str] | None = None,
    opener=None,
    bearer_reader=None,
) -> dict[str, object]:
    args = build_parser().parse_args(list(argv))
    command = args.command
    if command in {"accept-company", "accept-terms", "renew-terms", "outcome"} and not args.confirm_provider_authorized:
        raise ClientError("provider_authorization_required")
    environment = os.environ if environ is None else environ
    key = provider_key(environment)
    client = build_opener() if opener is None else opener

    if command in {"accept-company", "accept-terms", "renew-terms"}:
        reference = _validate_reference_argument(
            args.provider_acceptance_reference, "invalid_provider_acceptance_reference"
        )
        idempotency = _validate_idempotency_argument(args.idempotency_key)
        payload: dict[str, object] = {"provider_acceptance_reference": reference}
        expected_offer = ""
        expected_version = 0
        expected_hash = ""
        expected_related = ""
        if command == "accept-company":
            event = "pilot_company"
        else:
            event = "terms_acceptance" if command == "accept-terms" else "terms_renewal"
            expected_offer = _validate_uuid_argument(args.offer_id, "invalid_offer_id")
            if not 1 <= args.offer_version <= 2**31 - 1:
                raise ClientError("invalid_offer_version")
            expected_version = args.offer_version
            expected_hash = args.exact_terms_sha256
            if not isinstance(expected_hash, str) or not _HASH_PATTERN.fullmatch(expected_hash):
                raise ClientError("invalid_exact_terms_sha256")
            payload.update(
                {
                    "offer_id": expected_offer,
                    "offer_version": expected_version,
                    "exact_terms_sha256": expected_hash,
                }
            )
            if command == "renew-terms":
                expected_related = _validate_uuid_argument(
                    args.related_acceptance_event_id, "invalid_related_acceptance_event_id"
                )
                payload["related_acceptance_event_id"] = expected_related
        payload["event_type"] = event
        response = perform_request(
            client,
            build_request(
                "POST", "/api/v1/provider/commercial-acceptances", key,
                payload=payload, idempotency_key=idempotency,
            ),
            success_statuses=frozenset({200, 201}),
        )
        projected = _project_acceptance(
            response,
            expected_event=event,
            expected_offer_id=expected_offer,
            expected_version=expected_version,
            expected_hash=expected_hash,
            expected_related_id=expected_related,
            expected_reference=reference,
        )
        return {"ok": True, "command": command, **projected}

    if command == "status":
        if not 1 <= args.limit <= 100:
            raise ClientError("invalid_limit")
        query = urllib.parse.urlencode({"limit": str(args.limit)})
        response = perform_request(
            client,
            build_request("GET", "/api/v1/provider/pilot-status?" + query, key),
            success_statuses=frozenset({200}),
        )
        return {"ok": True, "command": command, "pilot_status": project_status(response, args.limit)}

    if command == "demand":
        if not 1 <= args.days <= 30:
            raise ClientError("invalid_days")
        query = urllib.parse.urlencode({"days": str(args.days)})
        response = perform_request(
            client,
            build_request("GET", "/api/v1/provider/demand?" + query, key),
            success_statuses=frozenset({200}),
        )
        return {"ok": True, "command": command, "demand": project_demand(response, args.days)}

    if command in {"resolve", "outcome"}:
        try:
            attribution_proof = (read_attribution_token if bearer_reader is None else bearer_reader)()
        except (ClientError, KeyboardInterrupt):
            raise
        except Exception:
            raise ClientError("attribution_bearer_unavailable") from None
        attribution_proof = _validate_attribution_token(attribution_proof)
        if command == "resolve":
            response = perform_request(
                client,
                build_request(
                    "POST", "/api/v1/provider/action-tickets/resolve", key,
                    payload={"attribution_token": attribution_proof},
                ),
                success_statuses=frozenset({200}),
            )
            return {
                "ok": True,
                "command": command,
                "controlled_intent_resolution": project_resolution(response),
            }
        idempotency = _validate_idempotency_argument(args.idempotency_key)
        payload = {"attribution_token": attribution_proof, "outcome": args.outcome}
        if args.ticket_id is not None:
            payload["ticket_id"] = _validate_uuid_argument(args.ticket_id, "invalid_ticket_id")
        response = perform_request(
            client,
            build_request(
                "POST", "/api/v1/provider/outcomes", key,
                payload=payload, idempotency_key=idempotency,
            ),
            success_statuses=frozenset({200, 201}),
        )
        return {"ok": True, "command": command, **project_outcome(response, args.outcome)}

    if command == "receipt":
        receipt_id = _validate_uuid_argument(args.receipt_id, "invalid_receipt_id")
        response = perform_request(
            client,
            build_request("GET", "/api/v1/provider/receipts/" + receipt_id, key),
            success_statuses=frozenset({200}),
        )
        return {"ok": True, "command": command, "receipt": project_receipt(response, receipt_id)}

    raise ClientError("invalid_arguments")


def exit_code_for_error(error: ClientError) -> int:
    if error.code in {
        "invalid_arguments", "provider_key_unavailable", "attribution_bearer_unavailable",
        "invalid_idempotency_key", "invalid_provider_acceptance_reference", "invalid_offer_id",
        "invalid_offer_version", "invalid_exact_terms_sha256", "invalid_related_acceptance_event_id",
        "invalid_limit", "invalid_days", "invalid_ticket_id", "invalid_receipt_id",
        "provider_authorization_required",
        "core_dump_hardening_unavailable", "invalid_request_path", "invalid_request_shape",
    }:
        return EXIT_INPUT
    if error.code == "network_error":
        return EXIT_NETWORK
    if error.http_status in {401, 403}:
        return EXIT_AUTH
    if error.http_status == 429:
        return EXIT_RATE_LIMIT
    if error.code in {"invalid_response", "response_too_large"}:
        return EXIT_RESPONSE
    return EXIT_REMOTE


def _emit(document: Mapping[str, object], stream) -> None:
    stream.write(json.dumps(dict(document), sort_keys=True, separators=(",", ":")) + "\n")


def main(argv: Sequence[str] | None = None) -> int:
    try:
        disable_core_dumps()
        result = run(sys.argv[1:] if argv is None else argv)
    except ClientError as error:
        document: dict[str, object] = {"ok": False, "error": error.code}
        if error.http_status is not None:
            document["http_status"] = error.http_status
        _emit(document, sys.stderr)
        return exit_code_for_error(error)
    except KeyboardInterrupt:
        _emit({"ok": False, "error": "interrupted"}, sys.stderr)
        return EXIT_INTERRUPTED
    except Exception:
        _emit({"ok": False, "error": "internal_error"}, sys.stderr)
        return EXIT_INTERNAL
    _emit(result, sys.stdout)
    return EXIT_OK


if __name__ == "__main__":
    raise SystemExit(main())
