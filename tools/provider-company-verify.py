#!/usr/bin/env python3
"""Owner-gated NHS provider-company deduplication verification.

The authoritative company identifier is read only from an echo-disabled
``/dev/tty`` prompt.  The tool sends a keyed, namespace-separated digest to a
fixed NHS admin endpoint and never emits the identifier, its normalized form,
the digest, either key, or the server response body.
"""

from __future__ import annotations

import argparse
import hashlib
import hmac
import json
import os
import re
import resource
import ssl
import sys
import termios
import urllib.error
import urllib.request
from typing import Mapping, Sequence


ENDPOINT = "https://nothumansearch.ai/api/v1/admin/provider-commercial/action"
DEDUP_KEY_ENV = "NHS_PROVIDER_COMPANY_DEDUP_KEY"
ADMIN_KEY_ENV = "NHS_PROVIDER_OPERATOR_ADMIN_KEY"
MESSAGE_NAMESPACE = b"nhs-provider-company-dedup-v1"
MAX_IDENTIFIER_BYTES = 200
MAX_RESPONSE_BYTES = 16 * 1024
REQUEST_TIMEOUT_SECONDS = 10
TTY_PATH = "/dev/tty"

_AUTHORITY_PATTERN = re.compile(r"^[a-z0-9][a-z0-9._:-]{0,99}$")
_EVIDENCE_REFERENCE_PATTERN = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._:/-]{7,199}$")
_UUID_PATTERN = re.compile(
    r"^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"
)
_HEX_KEY_PATTERN = re.compile(r"^[0-9A-Fa-f]{64}$")
_ASCII_WHITESPACE = " \t\n\r\v\f"
_PERMITTED_WHITESPACE_BYTES = frozenset(ord(char) for char in _ASCII_WHITESPACE)


class OperatorError(Exception):
    """An error represented by a stable code that cannot contain sensitive data."""

    def __init__(self, code: str, *, http_status: int | None = None):
        super().__init__(code)
        self.code = code
        self.http_status = http_status


class SafeArgumentParser(argparse.ArgumentParser):
    """Avoid argparse reflecting an accidental raw identifier back to output."""

    def error(self, _message: str) -> None:
        raise OperatorError("invalid_arguments")


class NoRedirect(urllib.request.HTTPRedirectHandler):
    """Refuse every HTTP redirect, including same-origin redirects."""

    def redirect_request(self, req, fp, code, msg, headers, newurl):  # noqa: ANN001
        return None


def build_parser() -> argparse.ArgumentParser:
    parser = SafeArgumentParser(
        description=(
            "Verify one owner-authorized NHS pilot company using an identifier "
            "entered privately at /dev/tty."
        )
    )
    parser.add_argument("--provider-acceptance-event-id", required=True)
    parser.add_argument("--identity-authority", required=True)
    parser.add_argument("--operator-reference", required=True)
    parser.add_argument("--identity-evidence-reference", required=True)
    parser.add_argument(
        "--confirm-owner-authorized",
        action="store_true",
        help="Confirm that the owner authorized this exact company verification.",
    )
    return parser


def validate_authority(value: str) -> str:
    if not isinstance(value, str):
        raise OperatorError("invalid_identity_authority")
    try:
        value.encode("ascii", "strict")
    except UnicodeEncodeError as error:
        raise OperatorError("invalid_identity_authority") from error
    if value != value.lower() or not _AUTHORITY_PATTERN.fullmatch(value):
        raise OperatorError("invalid_identity_authority")
    return value


def normalize_identifier(value: str) -> str:
    """Apply the v1 byte-level identifier normalization contract."""

    if not isinstance(value, str):
        raise OperatorError("invalid_company_identifier")
    try:
        raw = value.encode("ascii", "strict")
    except UnicodeEncodeError as error:
        raise OperatorError("invalid_company_identifier") from error
    if len(raw) > MAX_IDENTIFIER_BYTES:
        raise OperatorError("invalid_company_identifier")
    for byte in raw:
        if byte == 0x7F or (byte < 0x20 and byte not in _PERMITTED_WHITESPACE_BYTES):
            raise OperatorError("invalid_company_identifier")
    normalized = re.sub(r"[ \t\n\r\v\f]+", " ", value).strip(" ").lower()
    if not normalized:
        raise OperatorError("invalid_company_identifier")
    return normalized


def compute_company_key_hash(key: bytes, authority: str, identifier: str) -> str:
    if not isinstance(key, bytes) or len(key) != 32:
        raise OperatorError("invalid_dedup_key_configuration")
    authority = validate_authority(authority)
    normalized = normalize_identifier(identifier)
    message = MESSAGE_NAMESPACE + b"\0" + authority.encode("ascii") + b"\0" + normalized.encode("ascii")
    return hmac.new(key, message, hashlib.sha256).hexdigest()


def _validate_acceptance_event_id(value: str) -> str:
    normalized = value.strip().lower()
    if value != normalized or not _UUID_PATTERN.fullmatch(normalized):
        raise OperatorError("invalid_provider_acceptance_event_id")
    return normalized


def _validate_reference(value: str, code: str) -> str:
    normalized = value.strip()
    if value != normalized or not _EVIDENCE_REFERENCE_PATTERN.fullmatch(normalized):
        raise OperatorError(code)
    return normalized


def _dedup_key(environ: Mapping[str, str]) -> bytes:
    encoded = environ.get(DEDUP_KEY_ENV, "")
    if not isinstance(encoded, str) or not _HEX_KEY_PATTERN.fullmatch(encoded):
        raise OperatorError("dedup_key_unavailable")
    key = bytes.fromhex(encoded)
    if len(key) != 32:
        raise OperatorError("dedup_key_unavailable")
    return key


def _admin_key(environ: Mapping[str, str]) -> str:
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


def read_identifier_from_tty() -> str:
    """Read at most 200 bytes from the controlling terminal with echo disabled."""

    flags = os.O_RDWR | getattr(os, "O_NOCTTY", 0)
    try:
        descriptor = os.open(TTY_PATH, flags)
    except OSError as error:
        raise OperatorError("tty_unavailable") from error

    original = None
    echo_disabled = False
    restore_failed = False
    raw = bytearray()
    try:
        try:
            original = termios.tcgetattr(descriptor)
            hidden = list(original)
            hidden[3] &= ~termios.ECHO
            termios.tcsetattr(descriptor, termios.TCSAFLUSH, hidden)
            echo_disabled = True
            os.write(descriptor, b"Authoritative company identifier: ")
            while True:
                chunk = os.read(descriptor, 1)
                if not chunk:
                    raise OperatorError("tty_unavailable")
                if chunk in (b"\n", b"\r"):
                    break
                raw.extend(chunk)
                if len(raw) > MAX_IDENTIFIER_BYTES:
                    raise OperatorError("invalid_company_identifier")
        except OperatorError:
            raise
        except (OSError, termios.error) as error:
            raise OperatorError("tty_unavailable") from error
    finally:
        if original is not None and echo_disabled:
            try:
                termios.tcsetattr(descriptor, termios.TCSADRAIN, original)
                os.write(descriptor, b"\n")
            except (OSError, termios.error):
                restore_failed = True
        try:
            os.close(descriptor)
        except OSError:
            pass

    if restore_failed:
        raise OperatorError("tty_unavailable")

    try:
        return raw.decode("ascii", "strict")
    except UnicodeDecodeError as error:
        raise OperatorError("invalid_company_identifier") from error


def build_opener():
    """Create a TLS-verifying opener with proxies and redirects disabled."""

    return urllib.request.build_opener(
        urllib.request.ProxyHandler({}),
        NoRedirect(),
        urllib.request.HTTPSHandler(context=ssl.create_default_context()),
    )


def build_request(
    admin_key: str,
    acceptance_event_id: str,
    company_key_hash: str,
    operator_reference: str,
    identity_evidence_reference: str,
) -> urllib.request.Request:
    payload = {
        "action": "verify_company",
        "provider_acceptance_event_id": acceptance_event_id,
        "company_key_hash": company_key_hash,
        "operator_reference": operator_reference,
        "identity_evidence_reference": identity_evidence_reference,
    }
    body = json.dumps(payload, sort_keys=True, separators=(",", ":")).encode("ascii")
    return urllib.request.Request(
        ENDPOINT,
        data=body,
        method="POST",
        headers={
            "Accept": "application/json",
            "Authorization": "Bearer " + admin_key,
            "Content-Type": "application/json",
            "User-Agent": "NHS-Provider-Company-Verify/1.0",
        },
    )


def perform_request(opener, request: urllib.request.Request) -> dict[str, object]:
    """Make exactly one bounded request and project a safe response receipt."""

    try:
        with opener.open(request, timeout=REQUEST_TIMEOUT_SECONDS) as response:
            status = int(response.getcode())
            if 300 <= status <= 399:
                raise OperatorError("redirect_refused", http_status=status)
            if status not in (200, 201):
                raise OperatorError("http_error", http_status=status)
            body = response.read(MAX_RESPONSE_BYTES + 1)
    except urllib.error.HTTPError as error:
        status = error.code if isinstance(error.code, int) and 100 <= error.code <= 599 else None
        code = "redirect_refused" if status is not None and 300 <= status <= 399 else "http_error"
        raise OperatorError(code, http_status=status) from error
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

    created = document.get("created")
    replay = document.get("idempotent_replay")
    evidence_recorded = document.get("commercial_evidence_recorded")
    threshold_evaluated = document.get("pilot_threshold_evaluated")
    if (
        not isinstance(created, bool)
        or not isinstance(replay, bool)
        or created == replay
        or created != (status == 201)
        or evidence_recorded is not True
        or threshold_evaluated is not False
    ):
        raise OperatorError("invalid_response", http_status=status)

    return {
        "ok": True,
        "action": "verify_company",
        "http_status": status,
        "created": created,
        "idempotent_replay": replay,
        "commercial_evidence_recorded": True,
        "pilot_threshold_evaluated": False,
    }


def run(
    argv: Sequence[str],
    *,
    environ: Mapping[str, str] | None = None,
    tty_reader=None,
    opener=None,
) -> dict[str, object]:
    args = build_parser().parse_args(list(argv))
    if not args.confirm_owner_authorized:
        raise OperatorError("owner_authorization_required")

    acceptance_event_id = _validate_acceptance_event_id(args.provider_acceptance_event_id)
    authority = validate_authority(args.identity_authority)
    operator_reference = _validate_reference(args.operator_reference, "invalid_operator_reference")
    evidence_reference = _validate_reference(
        args.identity_evidence_reference,
        "invalid_identity_evidence_reference",
    )
    environment = os.environ if environ is None else environ
    dedup_key = _dedup_key(environment)
    admin_key = _admin_key(environment)

    identifier = (read_identifier_from_tty if tty_reader is None else tty_reader)()
    company_key_hash = compute_company_key_hash(dedup_key, authority, identifier)
    identifier = ""  # Drop the last local reference before any network operation.
    request = build_request(
        admin_key,
        acceptance_event_id,
        company_key_hash,
        operator_reference,
        evidence_reference,
    )
    return perform_request(build_opener() if opener is None else opener, request)


def _emit(document: Mapping[str, object], stream) -> None:
    stream.write(json.dumps(dict(document), sort_keys=True, separators=(",", ":")) + "\n")


def disable_core_dumps() -> None:
    """Prevent identifier or key material from entering a process core dump."""

    try:
        resource.setrlimit(resource.RLIMIT_CORE, (0, 0))
    except (OSError, ValueError) as error:
        raise OperatorError("core_dump_hardening_unavailable") from error


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
    except Exception:  # Fail closed without reflecting exception text or inputs.
        _emit({"ok": False, "error": "internal_error"}, sys.stderr)
        return 1
    _emit(receipt, sys.stdout)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
