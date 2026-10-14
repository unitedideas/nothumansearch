#!/usr/bin/env python3
"""Security and contract tests for provider-company-verify.py."""

import contextlib
import hmac
import importlib.util
import io
import json
import pathlib
import ssl
import sys
import termios
import unittest
import urllib.error
import urllib.request
from unittest import mock


MODULE_PATH = pathlib.Path(__file__).with_name("provider-company-verify.py")
SPEC = importlib.util.spec_from_file_location("provider_company_verify", MODULE_PATH)
provider_verify = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = provider_verify
SPEC.loader.exec_module(provider_verify)

ACCEPTANCE_ID = "123e4567-e89b-42d3-a456-426614174000"
BASE_ARGS = [
    "--provider-acceptance-event-id",
    ACCEPTANCE_ID,
    "--identity-authority",
    "lei",
    "--operator-reference",
    "owner-case-2026-001",
    "--identity-evidence-reference",
    "identity-case-2026-001",
]
TEST_KEY = bytes(range(32))
TEST_ADMIN_KEY = "test-admin-key-fixture"
TEST_ENVIRONMENT = {
    provider_verify.DEDUP_KEY_ENV: TEST_KEY.hex(),
    provider_verify.ADMIN_KEY_ENV: TEST_ADMIN_KEY,
}


class FakeResponse:
    def __init__(self, status, body):
        self.status = status
        self.body = body
        self.read_sizes = []

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        return False

    def getcode(self):
        return self.status

    def read(self, size):
        self.read_sizes.append(size)
        return self.body[:size]


class FakeOpener:
    def __init__(self, response=None, error=None):
        self.response = response
        self.error = error
        self.calls = []

    def open(self, request, timeout):
        self.calls.append((request, timeout))
        if self.error is not None:
            raise self.error
        return self.response


def success_body(created=True, extra=None):
    document = {
        "created": created,
        "idempotent_replay": not created,
        "commercial_evidence_recorded": True,
        "pilot_threshold_evaluated": False,
    }
    if extra:
        document.update(extra)
    return json.dumps(document).encode("utf-8")


class CompanyDigestContractTest(unittest.TestCase):
    def test_fixed_hmac_vector(self):
        digest = provider_verify.compute_company_key_hash(
            TEST_KEY,
            "lei",
            "Acme, Inc. / 123-45",
        )
        expected = "d9fc50ef9be3292abf2db761a05963ed959bb4f6c20da880ecf14621acc52b4f"
        self.assertTrue(hmac.compare_digest(digest, expected))

    def test_normalization_and_authority_separation(self):
        normalized = provider_verify.normalize_identifier(" \tACME,   Inc. / 123-45\r\n")
        normalization_matches = hmac.compare_digest(normalized, "acme, inc. / 123-45")
        self.assertTrue(normalization_matches)
        canonical = provider_verify.compute_company_key_hash(
            TEST_KEY,
            "registry:us-wa:ubi",
            "ACME,   Inc. / 123-45",
        )
        equivalent = provider_verify.compute_company_key_hash(
            TEST_KEY,
            "registry:us-wa:ubi",
            "  acme, inc. / 123-45  ",
        )
        other_authority = provider_verify.compute_company_key_hash(
            TEST_KEY,
            "lei",
            "acme, inc. / 123-45",
        )
        self.assertTrue(hmac.compare_digest(canonical, equivalent))
        self.assertFalse(hmac.compare_digest(canonical, other_authority))

    def test_identifier_and_authority_validation(self):
        invalid_identifiers = [
            "",
            " \t\r\n ",
            "M\u00e1laga Holdings",
            "Acme\x00Holdings",
            "Acme\x1bHoldings",
            "Acme\x7fHoldings",
            "x" * 201,
        ]
        for case, identifier in enumerate(invalid_identifiers):
            with self.subTest(case=case):
                with self.assertRaisesRegex(provider_verify.OperatorError, "invalid_company_identifier"):
                    provider_verify.normalize_identifier(identifier)

        for authority in ["LEI", "registry us", "r\u00e9gistry", "-registry", "r" * 101]:
            with self.subTest(authority=repr(authority)):
                with self.assertRaisesRegex(provider_verify.OperatorError, "invalid_identity_authority"):
                    provider_verify.validate_authority(authority)

        normalized = provider_verify.normalize_identifier("ACME, INC. #42/A")
        punctuation_matches = hmac.compare_digest(normalized, "acme, inc. #42/a")
        self.assertTrue(punctuation_matches)
        self.assertEqual(provider_verify.validate_authority("registry:us-wa:ubi"), "registry:us-wa:ubi")

    def test_key_configuration_is_exact_and_never_falls_back(self):
        for value in ["", "0" * 62, "g" * 64, "0" * 66]:
            with self.subTest(length=len(value)):
                with self.assertRaisesRegex(provider_verify.OperatorError, "dedup_key_unavailable"):
                    provider_verify._dedup_key({provider_verify.DEDUP_KEY_ENV: value})
        self.assertTrue(hmac.compare_digest(provider_verify._dedup_key(TEST_ENVIRONMENT), TEST_KEY))
        with self.assertRaisesRegex(provider_verify.OperatorError, "admin_key_unavailable"):
            provider_verify._admin_key({})
        with self.assertRaisesRegex(provider_verify.OperatorError, "admin_key_unavailable"):
            provider_verify._admin_key({provider_verify.ADMIN_KEY_ENV: "bad\nkey-value"})


class OwnerGateAndTTYTest(unittest.TestCase):
    def test_main_disables_core_dumps_before_processing_arguments(self):
        receipt = {
            "ok": True,
            "action": "verify_company",
            "http_status": 201,
            "created": True,
            "idempotent_replay": False,
            "commercial_evidence_recorded": True,
            "pilot_threshold_evaluated": False,
        }
        stdout = io.StringIO()
        with (
            mock.patch.object(provider_verify.resource, "setrlimit") as set_limit,
            mock.patch.object(provider_verify, "run", return_value=receipt) as run,
            contextlib.redirect_stdout(stdout),
        ):
            status = provider_verify.main([])

        self.assertEqual(status, 0)
        set_limit.assert_called_once_with(provider_verify.resource.RLIMIT_CORE, (0, 0))
        run.assert_called_once_with([])
        self.assertEqual(json.loads(stdout.getvalue()), receipt)

    def test_core_dump_hardening_failure_is_sanitized_and_fails_closed(self):
        stderr = io.StringIO()
        with (
            mock.patch.object(provider_verify.resource, "setrlimit", side_effect=OSError("private")),
            mock.patch.object(provider_verify, "run") as run,
            contextlib.redirect_stderr(stderr),
        ):
            status = provider_verify.main([])

        self.assertEqual(status, 1)
        run.assert_not_called()
        self.assertEqual(
            json.loads(stderr.getvalue()),
            {"error": "core_dump_hardening_unavailable", "ok": False},
        )

    def test_owner_gate_precedes_secret_lookup_and_tty(self):
        tty_reader = mock.Mock(side_effect=AssertionError("TTY must not be read"))
        with self.assertRaisesRegex(provider_verify.OperatorError, "owner_authorization_required"):
            provider_verify.run(BASE_ARGS, environ={}, tty_reader=tty_reader, opener=mock.Mock())
        tty_reader.assert_not_called()

    def test_raw_identifier_has_no_cli_option_and_parser_does_not_echo_it(self):
        raw_identifier = "Raw Company Identifier 987654"
        stdout = io.StringIO()
        stderr = io.StringIO()
        with contextlib.redirect_stdout(stdout), contextlib.redirect_stderr(stderr):
            status = provider_verify.main(["--identifier", raw_identifier])
        self.assertEqual(status, 1)
        emitted = stdout.getvalue() + stderr.getvalue()
        self.assertNotIn(raw_identifier, emitted)
        self.assertEqual(json.loads(stderr.getvalue()), {"error": "invalid_arguments", "ok": False})
        help_text = provider_verify.build_parser().format_help()
        self.assertNotIn("--identifier", help_text)
        self.assertNotIn("stdin", help_text.lower())
        self.assertNotIn("clipboard", help_text.lower())
        source = MODULE_PATH.read_text(encoding="utf-8").lower()
        forbidden_input_surfaces = ["sys.stdin", "input(", "clipboard", "pasteboard", "--identifier-file"]
        self.assertFalse(any(surface in source for surface in forbidden_input_surfaces))

    def test_tty_reader_opens_dev_tty_and_disables_then_restores_echo(self):
        original = [1, 2, 3, termios.ECHO | 0x200, 5, 6, [7]]
        reads = [b"A", b"C", b"M", b"E", b"\n"]
        with (
            mock.patch.object(provider_verify.os, "open", return_value=19) as open_tty,
            mock.patch.object(provider_verify.termios, "tcgetattr", return_value=original) as get_attributes,
            mock.patch.object(provider_verify.termios, "tcsetattr") as set_attributes,
            mock.patch.object(provider_verify.os, "write") as write_tty,
            mock.patch.object(provider_verify.os, "read", side_effect=reads),
            mock.patch.object(provider_verify.os, "close") as close_tty,
        ):
            identifier = provider_verify.read_identifier_from_tty()

        identifier_matches = hmac.compare_digest(identifier, "ACME")
        self.assertTrue(identifier_matches)
        open_tty.assert_called_once()
        self.assertEqual(open_tty.call_args.args[0], "/dev/tty")
        self.assertTrue(open_tty.call_args.args[1] & provider_verify.os.O_RDWR)
        get_attributes.assert_called_once_with(19)
        self.assertEqual(set_attributes.call_count, 2)
        hidden = set_attributes.call_args_list[0].args[2]
        self.assertEqual(hidden[3] & termios.ECHO, 0)
        self.assertEqual(set_attributes.call_args_list[1].args, (19, termios.TCSADRAIN, original))
        self.assertEqual(write_tty.call_args_list[0].args, (19, b"Authoritative company identifier: "))
        self.assertEqual(write_tty.call_args_list[-1].args, (19, b"\n"))
        close_tty.assert_called_once_with(19)

    def test_tty_restore_failure_fails_closed(self):
        original = [1, 2, 3, termios.ECHO | 0x200, 5, 6, [7]]
        with (
            mock.patch.object(provider_verify.os, "open", return_value=19),
            mock.patch.object(provider_verify.termios, "tcgetattr", return_value=original),
            mock.patch.object(provider_verify.termios, "tcsetattr", side_effect=[None, OSError()]),
            mock.patch.object(provider_verify.os, "write"),
            mock.patch.object(provider_verify.os, "read", side_effect=[b"A", b"\n"]),
            mock.patch.object(provider_verify.os, "close"),
        ):
            with self.assertRaisesRegex(provider_verify.OperatorError, "tty_unavailable"):
                provider_verify.read_identifier_from_tty()


class RequestBoundaryTest(unittest.TestCase):
    def test_exact_fixed_request_and_whitelisted_receipt(self):
        raw_identifier = "  ACME,   Inc. / 123-45  "
        expected_digest = provider_verify.compute_company_key_hash(TEST_KEY, "lei", raw_identifier)
        server_only_secret = "server-private-value"
        response = FakeResponse(
            201,
            success_body(
                created=True,
                extra={
                    "company": {
                        "company_key_hash": expected_digest,
                        "internal": server_only_secret,
                    },
                    "evidence_scope": raw_identifier,
                },
            ),
        )
        opener = FakeOpener(response=response)
        receipt = provider_verify.run(
            BASE_ARGS + ["--confirm-owner-authorized"],
            environ=TEST_ENVIRONMENT,
            tty_reader=lambda: raw_identifier,
            opener=opener,
        )

        self.assertEqual(len(opener.calls), 1)
        request, timeout = opener.calls[0]
        self.assertEqual(timeout, 10)
        self.assertEqual(request.full_url, provider_verify.ENDPOINT)
        self.assertEqual(request.get_method(), "POST")
        authorization_matches = hmac.compare_digest(
            request.get_header("Authorization"),
            "Bearer " + TEST_ADMIN_KEY,
        )
        self.assertTrue(authorization_matches)
        self.assertEqual(request.get_header("Content-type"), "application/json")
        self.assertEqual(request.get_header("Accept"), "application/json")
        request_document = json.loads(request.data)
        self.assertEqual(
            set(request_document),
            {
                "action",
                "provider_acceptance_event_id",
                "company_key_hash",
                "operator_reference",
                "identity_evidence_reference",
            },
        )
        self.assertEqual(request_document["action"], "verify_company")
        self.assertEqual(request_document["provider_acceptance_event_id"], ACCEPTANCE_ID)
        self.assertEqual(request_document["operator_reference"], "owner-case-2026-001")
        self.assertEqual(request_document["identity_evidence_reference"], "identity-case-2026-001")
        self.assertTrue(hmac.compare_digest(request_document["company_key_hash"], expected_digest))
        self.assertFalse(raw_identifier.strip().encode("ascii") in request.data)
        self.assertEqual(response.read_sizes, [provider_verify.MAX_RESPONSE_BYTES + 1])
        self.assertEqual(
            receipt,
            {
                "ok": True,
                "action": "verify_company",
                "http_status": 201,
                "created": True,
                "idempotent_replay": False,
                "commercial_evidence_recorded": True,
                "pilot_threshold_evaluated": False,
            },
        )
        serialized = json.dumps(receipt)
        forbidden_values = [raw_identifier.strip(), expected_digest, TEST_ADMIN_KEY, server_only_secret]
        self.assertFalse(any(value in serialized for value in forbidden_values))

    def test_proxy_inheritance_and_redirects_are_disabled(self):
        sentinel_opener = object()
        sentinel_context = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
        with (
            mock.patch.object(provider_verify.ssl, "create_default_context", return_value=sentinel_context),
            mock.patch.object(provider_verify.urllib.request, "build_opener", return_value=sentinel_opener) as builder,
        ):
            self.assertIs(provider_verify.build_opener(), sentinel_opener)
        handlers = builder.call_args.args
        self.assertIsInstance(handlers[0], urllib.request.ProxyHandler)
        self.assertEqual(handlers[0].proxies, {})
        self.assertIsInstance(handlers[1], provider_verify.NoRedirect)
        self.assertIsNone(handlers[1].redirect_request(None, None, 302, "", {}, "https://example.invalid"))
        self.assertIsInstance(handlers[2], urllib.request.HTTPSHandler)

        redirect = FakeOpener(response=FakeResponse(302, b""))
        with self.assertRaisesRegex(provider_verify.OperatorError, "redirect_refused"):
            provider_verify.perform_request(redirect, provider_verify.build_request(
                TEST_ADMIN_KEY,
                ACCEPTANCE_ID,
                "a" * 64,
                "owner-case-2026-001",
                "identity-case-2026-001",
            ))
        self.assertEqual(len(redirect.calls), 1)

    def test_response_and_retry_boundaries(self):
        oversized = FakeResponse(201, b"x" * (provider_verify.MAX_RESPONSE_BYTES + 50))
        opener = FakeOpener(response=oversized)
        request = provider_verify.build_request(
            TEST_ADMIN_KEY,
            ACCEPTANCE_ID,
            "a" * 64,
            "owner-case-2026-001",
            "identity-case-2026-001",
        )
        with self.assertRaisesRegex(provider_verify.OperatorError, "response_too_large"):
            provider_verify.perform_request(opener, request)
        self.assertEqual(oversized.read_sizes, [provider_verify.MAX_RESPONSE_BYTES + 1])
        self.assertEqual(len(opener.calls), 1)

        failed = FakeOpener(error=urllib.error.URLError("private failure detail"))
        with self.assertRaisesRegex(provider_verify.OperatorError, "network_error"):
            provider_verify.perform_request(failed, request)
        self.assertEqual(len(failed.calls), 1)

    def test_errors_never_emit_identifier_keys_url_or_server_details(self):
        raw_identifier = "Sensitive Company 2468"
        private_detail = "private-server-body-and-identifier-2468"
        opener = FakeOpener(error=urllib.error.URLError(private_detail))
        stdout = io.StringIO()
        stderr = io.StringIO()
        with (
            mock.patch.object(provider_verify, "read_identifier_from_tty", return_value=raw_identifier),
            mock.patch.object(provider_verify, "build_opener", return_value=opener),
            mock.patch.dict(provider_verify.os.environ, TEST_ENVIRONMENT, clear=True),
            contextlib.redirect_stdout(stdout),
            contextlib.redirect_stderr(stderr),
        ):
            status = provider_verify.main(BASE_ARGS + ["--confirm-owner-authorized"])
        self.assertEqual(status, 1)
        self.assertEqual(stdout.getvalue(), "")
        self.assertEqual(json.loads(stderr.getvalue()), {"error": "network_error", "ok": False})
        emitted = stderr.getvalue()
        forbidden_values = [
            raw_identifier,
            private_detail,
            TEST_ENVIRONMENT[provider_verify.DEDUP_KEY_ENV],
            TEST_ADMIN_KEY,
            provider_verify.ENDPOINT,
        ]
        self.assertFalse(any(value in emitted for value in forbidden_values))

    def test_http_errors_are_status_only_and_server_body_is_not_read(self):
        private_detail = "sensitive server diagnostic"
        response_body = io.BytesIO(private_detail.encode("ascii"))
        http_error = urllib.error.HTTPError(
            provider_verify.ENDPOINT,
            503,
            private_detail,
            {},
            response_body,
        )
        opener = FakeOpener(error=http_error)
        request = provider_verify.build_request(
            TEST_ADMIN_KEY,
            ACCEPTANCE_ID,
            "a" * 64,
            "owner-case-2026-001",
            "identity-case-2026-001",
        )
        with self.assertRaises(provider_verify.OperatorError) as raised:
            provider_verify.perform_request(opener, request)
        self.assertEqual(raised.exception.code, "http_error")
        self.assertEqual(raised.exception.http_status, 503)
        self.assertEqual(response_body.tell(), 0)
        self.assertFalse(private_detail in str(raised.exception))


if __name__ == "__main__":
    unittest.main()
