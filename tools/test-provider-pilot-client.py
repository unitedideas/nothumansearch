#!/usr/bin/env python3
"""Offline security and contract tests for provider-pilot-client.py."""

import contextlib
import copy
import hmac
import importlib.util
import io
import json
import pathlib
import ssl
import sys
import unittest
import urllib.error
import urllib.parse
import urllib.request
from unittest import mock


MODULE_PATH = pathlib.Path(__file__).with_name("provider-pilot-client.py")
SPEC = importlib.util.spec_from_file_location("provider_pilot_client", MODULE_PATH)
client_tool = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = client_tool
SPEC.loader.exec_module(client_tool)

TEST_PROVIDER_KEY = "provider-key-fixture-that-must-never-be-emitted"
TEST_ATTRIBUTION = "signed.attribution.bearer.fixture-that-must-never-be-emitted"
TEST_ENVIRONMENT = {client_tool.PROVIDER_KEY_ENV: TEST_PROVIDER_KEY}

CLAIM_ID = "11111111-1111-4111-8111-111111111111"
OFFER_ID = "22222222-2222-4222-8222-222222222222"
ACCEPTANCE_ID = "33333333-3333-4333-8333-333333333333"
RELATED_ID = "44444444-4444-4444-8444-444444444444"
TICKET_ID = "55555555-5555-4555-8555-555555555555"
HANDOFF_ID = "66666666-6666-4666-8666-666666666666"
RECEIPT_ID = "77777777-7777-4777-8777-777777777777"
EVENT_ID = "88888888-8888-4888-8888-888888888888"
TERMS_HASH = "a" * 64
REFERENCE = "provider-case-2026-001"
IDEMPOTENCY = "provider-idempotency-001"
TIMESTAMP = "2026-08-02T12:00:00Z"


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
    def __init__(self, *items):
        self.items = list(items)
        self.calls = []

    def open(self, request, timeout):
        self.calls.append((request, timeout))
        if not self.items:
            raise AssertionError("unexpected request")
        item = self.items.pop(0)
        if isinstance(item, BaseException):
            raise item
        return item


class NonTTY(io.StringIO):
    def isatty(self):
        return False


def encoded(document):
    return json.dumps(document, separators=(",", ":")).encode("utf-8")


def acceptance_document(event_type, *, related=""):
    acceptance = {
        "id": ACCEPTANCE_ID,
        "provider_claim_id": CLAIM_ID,
        "provider_api_key_id": 17,
        "event_type": event_type,
        "provider_acceptance_reference": REFERENCE,
        "provider_accepted_at": TIMESTAMP,
        "created_at": TIMESTAMP,
        "server_private": "must-not-emit",
    }
    if event_type != "pilot_company":
        acceptance.update(
            {
                "provider_offer_id": OFFER_ID,
                "offer_version": 3,
                "terms_contract_version": "nhs-provider-commercial-terms-v1",
                "exact_terms_sha256": TERMS_HASH,
                "valid_until": "2026-09-01T12:00:00Z",
            }
        )
    if related:
        acceptance["related_acceptance_event_id"] = related
    return {
        "acceptance": acceptance,
        "created": True,
        "idempotent_replay": False,
        "provider_authenticated": True,
        "owner_verification_required": True,
        "commercial_proof_created": False,
        "evidence_scope": "provider-authenticated acceptance only",
        "server_private": "must-not-emit",
    }


def status_document():
    return {
        "pilot_status": {
            "as_of": TIMESTAMP,
            "provider_claim_id": CLAIM_ID,
            "domain": "provider.example",
            "claim_status": "verified",
            "verification_last_succeeded_at": TIMESTAMP,
            "verification_next_check_at": "2026-08-03T12:00:00Z",
            "verification_consecutive_failures": 0,
            "company_acceptance_id": ACCEPTANCE_ID,
            "company_accepted_at": TIMESTAMP,
            "company_owner_verified": True,
            "company_owner_verified_at": TIMESTAMP,
            "offers": [
                {
                    "offer_id": OFFER_ID,
                    "status": "draft",
                    "version": 3,
                    "name": "Agent API trial",
                    "action_type": "trial",
                    "charge_event": "activated",
                    "bounty_cents": 1200,
                    "currency": "usd",
                    "billing_mode": "terms",
                    "terms_credit_limit_cents": 5000,
                    "terms_period_days": 30,
                    "commercial_terms_contract_version": "nhs-provider-commercial-terms-v1",
                    "commercial_terms_sha256": TERMS_HASH,
                    "provider_mor_acknowledgement_required": True,
                    "provider_acknowledges_merchant_of_record": False,
                    "latest_acceptance_id": ACCEPTANCE_ID,
                    "latest_acceptance_type": "terms_acceptance",
                    "latest_acceptance_at": TIMESTAMP,
                    "latest_acceptance_valid_until": "2026-09-01T12:00:00Z",
                    "latest_acceptance_owner_verified": True,
                    "latest_acceptance_owner_verified_at": TIMESTAMP,
                    "current_terms_owner_verified": True,
                    "current_terms_valid_until": "2026-09-01T12:00:00Z",
                    "renewal_eligible": True,
                    "activation_ready": True,
                    "action_url": "https://private.example/must-not-emit",
                }
            ],
            "recent_observed_handoffs": [
                {
                    "ticket_id": TICKET_ID,
                    "offer_id": OFFER_ID,
                    "offer_version": 3,
                    "ticket_status": "accepted",
                    "handoff_receipt_id": HANDOFF_ID,
                    "handoff_observed_at": TIMESTAMP,
                    "outcome_receipt_id": RECEIPT_ID,
                    "outcome": "accepted",
                    "charge_status": "charged",
                    "billed_cents": 1200,
                    "outcome_recorded_at": TIMESTAMP,
                    "attribution_token": TEST_ATTRIBUTION,
                }
            ],
            "server_private": "must-not-emit",
        },
        "evidence_scope": "claim-scoped continuity only",
        "server_private": "must-not-emit",
    }


def demand_document():
    return {
        "demand": {
            "domain": "provider.example",
            "days": 30,
            "retention_days": 30,
            "action_interest_cohort": "organic_result_returned_at",
            "topic_receipt_threshold": 20,
            "result_selection_receipt_threshold": 20,
            "action_interest_receipt_threshold": 20,
            "synthetic_excluded": True,
            "summary": {
                "organic_results_returned": 30,
                "search_receipts": 25,
                "average_organic_position": 2.4,
                "result_selections": 20,
                "result_selection_suppressed": False,
                "result_selection_rate": 0.8,
                "action_interest_receipts": None,
                "action_interest_rate": None,
                "action_interest_suppressed": True,
                "private_search": "must-not-emit",
            },
            "surfaces": [
                {
                    "surface": "rest",
                    "organic_results_returned": 30,
                    "result_selections": 20,
                    "result_selection_suppressed": False,
                    "action_interest_receipts": None,
                    "action_interest_suppressed": True,
                    "agent_identity": "must-not-emit",
                }
            ],
            "demand_topics": [
                {
                    "topic": "developer-tools",
                    "search_receipts": 25,
                    "average_organic_position": 2.4,
                    "result_selections": 20,
                    "result_selection_suppressed": False,
                    "action_interest_receipts": None,
                    "action_interest_suppressed": True,
                    "raw_query": "must-not-emit",
                }
            ],
            "action_types": [],
            "server_private": "must-not-emit",
        },
        "evidence_scope": "privacy-thresholded receipt aggregates only",
        "server_private": "must-not-emit",
    }


def resolution_document():
    return {
        "resolver_contract_version": "nhs-provider-controlled-intent-resolver-v1",
        "ticket_id": TICKET_ID,
        "offer_id": OFFER_ID,
        "offer_version": 3,
        "action_type": "trial",
        "controlled_intent": {
            "demand_topic": "developer-tools",
            "region_code": "US-WA",
            "budget_band": "500_1999",
            "urgency": "7_days",
            "requirement_flags": ["api_access", "mcp"],
            "contact_email": "must-not-emit",
        },
        "observed_at": TIMESTAMP,
        "intent_available_until": "2026-08-10T12:00:00Z",
        "consent_version": "nhs-provider-controlled-intent-disclosure-consent-v1",
        "attribution_token": TEST_ATTRIBUTION,
    }


def signed_receipt_document(outcome="accepted"):
    signed = {
        "v": 1,
        "kid": "nhs-provider-pilot-v1",
        "receipt_id": RECEIPT_ID,
        "ticket_id": TICKET_ID,
        "offer_id": OFFER_ID,
        "nhs_event_id": EVENT_ID,
        "outcome": outcome,
        "provider_reported_at": 1785681600,
        "recorded_at": 1785681600,
        "expires_at": 2101041600,
        "charged_minor": 1200,
        "currency": "usd",
        "charge_status": "charged",
    }
    return {
        "id": RECEIPT_ID,
        "nhs_event_id": EVENT_ID,
        "provider_claim_id": CLAIM_ID,
        "provider_offer_id": OFFER_ID,
        "action_ticket_id": TICKET_ID,
        "provider_api_key_id": 17,
        "outcome": outcome,
        "billed_cents": 1200,
        "charge_status": "charged",
        "currency": "usd",
        "signed_receipt": json.dumps(signed, separators=(",", ":")),
        "signature": "A" * 43,
        "provider_reported_at": TIMESTAMP,
        "created_at": TIMESTAMP,
        "idempotency_key_hash": "must-not-emit",
    }


def outcome_document(outcome="accepted"):
    return {
        "receipt": signed_receipt_document(outcome),
        "created": True,
        "idempotent_replay": False,
        "principal_charged": False,
        "provider_mor_contract_required": True,
        "principal_charged_by_nhs": False,
        "server_private": "must-not-emit",
    }


class CommandRequestTest(unittest.TestCase):
    def assert_provider_boundary(self, opener, *, method, path):
        self.assertEqual(len(opener.calls), 1)
        request, timeout = opener.calls[0]
        self.assertEqual(timeout, 10)
        parsed = urllib.parse.urlsplit(request.full_url)
        self.assertEqual(parsed.scheme, "https")
        self.assertEqual(parsed.netloc, "nothumansearch.ai")
        self.assertEqual(parsed.path, path)
        self.assertEqual(request.get_method(), method)
        self.assertTrue(
            hmac.compare_digest(request.get_header("X-nhs-provider-key"), TEST_PROVIDER_KEY)
        )
        self.assertIsNone(request.get_header("Authorization"))
        return request, parsed

    def test_all_three_acceptance_commands_send_exact_bound_payloads(self):
        cases = [
            (
                [
                    "accept-company", "--provider-acceptance-reference", REFERENCE,
                    "--idempotency-key", IDEMPOTENCY,
                    "--confirm-provider-authorized",
                ],
                "pilot_company",
                {},
            ),
            (
                [
                    "accept-terms", "--offer-id", OFFER_ID, "--offer-version", "3",
                    "--exact-terms-sha256", TERMS_HASH,
                    "--provider-acceptance-reference", REFERENCE,
                    "--idempotency-key", IDEMPOTENCY,
                    "--confirm-provider-authorized",
                ],
                "terms_acceptance",
                {"offer_id": OFFER_ID, "offer_version": 3, "exact_terms_sha256": TERMS_HASH},
            ),
            (
                [
                    "renew-terms", "--offer-id", OFFER_ID, "--offer-version", "3",
                    "--exact-terms-sha256", TERMS_HASH,
                    "--related-acceptance-event-id", RELATED_ID,
                    "--provider-acceptance-reference", REFERENCE,
                    "--idempotency-key", IDEMPOTENCY,
                    "--confirm-provider-authorized",
                ],
                "terms_renewal",
                {
                    "offer_id": OFFER_ID, "offer_version": 3,
                    "exact_terms_sha256": TERMS_HASH,
                    "related_acceptance_event_id": RELATED_ID,
                },
            ),
        ]
        for argv, event_type, expected_extra in cases:
            with self.subTest(command=argv[0]):
                related = RELATED_ID if event_type == "terms_renewal" else ""
                response = FakeResponse(201, encoded(acceptance_document(event_type, related=related)))
                opener = FakeOpener(response)
                result = client_tool.run(argv, environ=TEST_ENVIRONMENT, opener=opener)
                request, _ = self.assert_provider_boundary(
                    opener, method="POST", path="/api/v1/provider/commercial-acceptances"
                )
                self.assertEqual(request.get_header("Idempotency-key"), IDEMPOTENCY)
                self.assertEqual(request.get_header("Content-type"), "application/json")
                expected = {
                    "event_type": event_type,
                    "provider_acceptance_reference": REFERENCE,
                    **expected_extra,
                }
                self.assertEqual(json.loads(request.data), expected)
                self.assertEqual(result["command"], argv[0])
                self.assertFalse(result["commercial_proof_created"])
                self.assertNotIn("provider_api_key_id", result["acceptance"])
                self.assertNotIn("must-not-emit", json.dumps(result))
                self.assertEqual(response.read_sizes, [client_tool.MAX_RESPONSE_BYTES + 1])

    def test_status_and_demand_are_claim_derived_fixed_gets(self):
        status_opener = FakeOpener(FakeResponse(200, encoded(status_document())))
        status = client_tool.run(
            ["status", "--limit", "25"], environ=TEST_ENVIRONMENT, opener=status_opener
        )
        _, parsed = self.assert_provider_boundary(
            status_opener, method="GET", path="/api/v1/provider/pilot-status"
        )
        self.assertEqual(urllib.parse.parse_qs(parsed.query), {"limit": ["25"]})
        self.assertTrue(status["pilot_status"]["company_owner_verified"])
        self.assertTrue(status["pilot_status"]["offers"][0]["activation_ready"])
        status_serialized = json.dumps(status)
        self.assertNotIn("must-not-emit", status_serialized)
        self.assertNotIn(TEST_ATTRIBUTION, status_serialized)
        self.assertNotIn("action_url", status_serialized)

        demand_opener = FakeOpener(FakeResponse(200, encoded(demand_document())))
        demand = client_tool.run(
            ["demand", "--days", "30"], environ=TEST_ENVIRONMENT, opener=demand_opener
        )
        _, parsed = self.assert_provider_boundary(
            demand_opener, method="GET", path="/api/v1/provider/demand"
        )
        self.assertEqual(urllib.parse.parse_qs(parsed.query), {"days": ["30"]})
        self.assertNotIn("domain", urllib.parse.parse_qs(parsed.query))
        self.assertEqual(demand["demand"]["domain"], "provider.example")
        self.assertTrue(demand["demand"]["counts_are_receipts_not_unique_agents"])
        serialized = json.dumps(demand)
        self.assertNotIn("must-not-emit", serialized)
        self.assertNotIn("raw_query", serialized)

    def test_resolve_reads_bearer_off_argv_and_projects_only_controlled_intent(self):
        response = FakeResponse(200, encoded(resolution_document()))
        opener = FakeOpener(response)
        reader = mock.Mock(return_value=TEST_ATTRIBUTION)
        result = client_tool.run(
            ["resolve"], environ=TEST_ENVIRONMENT, opener=opener, bearer_reader=reader
        )
        reader.assert_called_once_with()
        request, parsed = self.assert_provider_boundary(
            opener, method="POST", path="/api/v1/provider/action-tickets/resolve"
        )
        self.assertEqual(parsed.query, "")
        self.assertEqual(json.loads(request.data), {"attribution_token": TEST_ATTRIBUTION})
        self.assertNotIn(TEST_ATTRIBUTION, request.full_url)
        self.assertTrue(result["controlled_intent_resolution"]["read_only"])
        self.assertFalse(result["controlled_intent_resolution"]["charge_created"])
        serialized = json.dumps(result)
        self.assertNotIn(TEST_ATTRIBUTION, serialized)
        self.assertNotIn("contact_email", serialized)

    def test_outcome_uses_stdin_bearer_and_returns_bound_signed_receipt(self):
        response = FakeResponse(201, encoded(outcome_document()))
        opener = FakeOpener(response)
        result = client_tool.run(
            [
                "outcome", "--outcome", "accepted", "--idempotency-key", IDEMPOTENCY,
                "--ticket-id", TICKET_ID,
                "--confirm-provider-authorized",
            ],
            environ=TEST_ENVIRONMENT,
            opener=opener,
            bearer_reader=lambda: TEST_ATTRIBUTION,
        )
        request, _ = self.assert_provider_boundary(
            opener, method="POST", path="/api/v1/provider/outcomes"
        )
        self.assertEqual(request.get_header("Idempotency-key"), IDEMPOTENCY)
        self.assertEqual(
            json.loads(request.data),
            {"attribution_token": TEST_ATTRIBUTION, "outcome": "accepted", "ticket_id": TICKET_ID},
        )
        self.assertEqual(result["receipt"]["id"], RECEIPT_ID)
        self.assertFalse(result["principal_charged"])
        serialized = json.dumps(result)
        self.assertNotIn(TEST_ATTRIBUTION, serialized)
        self.assertNotIn("idempotency_key_hash", serialized)

    def test_receipt_uses_uuid_only_path_and_checks_returned_binding(self):
        response = FakeResponse(200, encoded({"receipt": signed_receipt_document(), "private": "x"}))
        opener = FakeOpener(response)
        result = client_tool.run(
            ["receipt", "--receipt-id", RECEIPT_ID],
            environ=TEST_ENVIRONMENT,
            opener=opener,
        )
        self.assert_provider_boundary(
            opener, method="GET", path="/api/v1/provider/receipts/" + RECEIPT_ID
        )
        self.assertEqual(result["receipt"]["id"], RECEIPT_ID)
        self.assertNotIn("private", result)


class InputAndSchemaTest(unittest.TestCase):
    def test_provider_mutations_require_explicit_authority_before_secret_or_network_access(self):
        cases = [
            [
                "accept-company", "--provider-acceptance-reference", REFERENCE,
                "--idempotency-key", IDEMPOTENCY,
            ],
            [
                "accept-terms", "--offer-id", OFFER_ID, "--offer-version", "3",
                "--exact-terms-sha256", TERMS_HASH,
                "--provider-acceptance-reference", REFERENCE,
                "--idempotency-key", IDEMPOTENCY,
            ],
            [
                "outcome", "--outcome", "accepted", "--idempotency-key", IDEMPOTENCY,
            ],
        ]
        for argv in cases:
            with self.subTest(command=argv[0]):
                opener = mock.Mock()
                with self.assertRaisesRegex(client_tool.ClientError, "provider_authorization_required"):
                    client_tool.run(argv, environ={}, opener=opener)
                opener.open.assert_not_called()

    def test_bearer_is_bounded_stdin_or_hidden_tty_and_never_a_parser_argument(self):
        self.assertEqual(
            client_tool.read_attribution_token(NonTTY(TEST_ATTRIBUTION + "\n")),
            TEST_ATTRIBUTION,
        )
        with self.assertRaisesRegex(client_tool.ClientError, "attribution_bearer_unavailable"):
            client_tool.read_attribution_token(NonTTY(TEST_ATTRIBUTION + "\nextra"))
        with self.assertRaisesRegex(client_tool.ClientError, "attribution_bearer_unavailable"):
            client_tool.read_attribution_token(NonTTY("x" * (client_tool.MAX_ATTRIBUTION_TOKEN_BYTES + 1)))

        tty = mock.Mock()
        tty.isatty.return_value = True
        with mock.patch.object(client_tool.getpass, "getpass", return_value=TEST_ATTRIBUTION) as hidden:
            self.assertEqual(client_tool.read_attribution_token(tty), TEST_ATTRIBUTION)
        hidden.assert_called_once_with("NHS attribution bearer: ", stream=sys.stderr)
        tty.read.assert_not_called()

        help_text = client_tool.build_parser().format_help().lower()
        source = MODULE_PATH.read_text(encoding="utf-8").lower()
        self.assertNotIn("--attribution", help_text)
        self.assertNotIn("--token", help_text)
        self.assertNotIn("base-url", help_text)
        self.assertNotIn("provider-key", help_text)
        self.assertNotIn("sys.argv[", source.split("def main", 1)[0])

    def test_arguments_and_configuration_fail_before_network(self):
        cases = [
            (["status"], {}, "provider_key_unavailable"),
            (["status"], {client_tool.PROVIDER_KEY_ENV: "bad\nkey"}, "provider_key_unavailable"),
            (["status", "--limit", "101"], TEST_ENVIRONMENT, "invalid_limit"),
            (["demand", "--days", "0"], TEST_ENVIRONMENT, "invalid_days"),
            (
                [
                    "accept-terms", "--offer-id", "../private", "--offer-version", "3",
                    "--exact-terms-sha256", TERMS_HASH,
                    "--provider-acceptance-reference", REFERENCE,
                    "--idempotency-key", IDEMPOTENCY,
                    "--confirm-provider-authorized",
                ],
                TEST_ENVIRONMENT,
                "invalid_offer_id",
            ),
            (["resolve", "--attribution-token", TEST_ATTRIBUTION], TEST_ENVIRONMENT, "invalid_arguments"),
            (["status", "--base-url", "https://evil.example"], TEST_ENVIRONMENT, "invalid_arguments"),
        ]
        for argv, environment, code in cases:
            with self.subTest(argv=argv):
                with self.assertRaisesRegex(client_tool.ClientError, code) as raised:
                    client_tool.run(argv, environ=environment, opener=mock.Mock())
                self.assertIsNone(raised.exception.__cause__)

        with self.assertRaisesRegex(client_tool.ClientError, "provider_key_unavailable") as raised:
            client_tool.provider_key({client_tool.PROVIDER_KEY_ENV: "private-credential-\N{SNOWMAN}"})
        self.assertIsNone(raised.exception.__cause__)
        self.assertNotIn("private-credential", repr(raised.exception))

        with self.assertRaisesRegex(client_tool.ClientError, "attribution_bearer_unavailable") as raised:
            client_tool.run(
                ["resolve"],
                environ=TEST_ENVIRONMENT,
                opener=mock.Mock(),
                bearer_reader=mock.Mock(side_effect=RuntimeError(TEST_ATTRIBUTION)),
            )
        self.assertIsNone(raised.exception.__cause__)
        self.assertNotIn(TEST_ATTRIBUTION, repr(raised.exception))

    def test_response_truth_relationships_fail_closed(self):
        cases = []
        bad_status = status_document()
        bad_status["pilot_status"]["offers"][0]["current_terms_owner_verified"] = False
        cases.append((client_tool.project_status, (bad_status, 25)))
        bad_demand = demand_document()
        bad_demand["demand"]["summary"]["result_selections"] = 19
        cases.append((client_tool.project_demand, (bad_demand, 30)))
        bad_suppression = demand_document()
        bad_suppression["demand"]["summary"]["action_interest_suppressed"] = False
        cases.append((client_tool.project_demand, (bad_suppression, 30)))
        bad_resolution = resolution_document()
        bad_resolution["controlled_intent"]["requirement_flags"] = ["api_access", "api_access"]
        cases.append((client_tool.project_resolution, (bad_resolution,)))
        bad_outcome = outcome_document()
        bad_outcome["receipt"]["billed_cents"] = 1300
        cases.append((client_tool.project_outcome, (bad_outcome, "accepted")))
        bad_binding = outcome_document()
        signed = json.loads(bad_binding["receipt"]["signed_receipt"])
        signed["ticket_id"] = CLAIM_ID
        bad_binding["receipt"]["signed_receipt"] = json.dumps(signed, separators=(",", ":"))
        cases.append((client_tool.project_outcome, (bad_binding, "accepted")))
        for projector, args in cases:
            with self.subTest(projector=projector.__name__):
                with self.assertRaisesRegex(client_tool.ClientError, "invalid_response"):
                    projector(*args)

    def test_acceptance_response_must_match_exact_requested_contract(self):
        response = acceptance_document("terms_acceptance")
        response["acceptance"]["exact_terms_sha256"] = "b" * 64
        with self.assertRaisesRegex(client_tool.ClientError, "invalid_response"):
            client_tool._project_acceptance(
                response,
                expected_event="terms_acceptance",
                expected_offer_id=OFFER_ID,
                expected_version=3,
                expected_hash=TERMS_HASH,
                expected_reference=REFERENCE,
            )


class TransportAndFailureTest(unittest.TestCase):
    def test_proxy_redirect_retry_and_tls_boundaries(self):
        sentinel_opener = object()
        sentinel_context = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
        with (
            mock.patch.object(client_tool.ssl, "create_default_context", return_value=sentinel_context),
            mock.patch.object(client_tool.urllib.request, "build_opener", return_value=sentinel_opener) as builder,
        ):
            self.assertIs(client_tool.build_opener(), sentinel_opener)
        handlers = builder.call_args.args
        self.assertIsInstance(handlers[0], urllib.request.ProxyHandler)
        self.assertEqual(handlers[0].proxies, {})
        self.assertIsInstance(handlers[1], client_tool.NoRedirect)
        self.assertIsNone(handlers[1].redirect_request(None, None, 302, "", {}, "https://evil.example"))
        self.assertIsInstance(handlers[2], urllib.request.HTTPSHandler)
        self.assertIs(handlers[2]._context, sentinel_context)
        # One opener.open call is the complete retry budget.
        request = client_tool.build_request(
            "GET", "/api/v1/provider/pilot-status?limit=25", TEST_PROVIDER_KEY
        )
        opener = FakeOpener(FakeResponse(302, b""))
        with self.assertRaisesRegex(client_tool.ClientError, "redirect_refused"):
            client_tool.perform_request(opener, request, success_statuses=frozenset({200}))
        self.assertEqual(len(opener.calls), 1)

    def test_only_exact_fixed_provider_paths_are_buildable(self):
        invalid = [
            ("GET", "https://evil.example/api/v1/provider/pilot-status"),
            ("GET", "/api/v1/provider/pilot-status?domain=private.example"),
            ("GET", "/api/v1/provider/receipts/../admin/provider-proof"),
            ("POST", "/api/v1/admin/provider-commercial/action"),
            ("POST", "/api/v1/provider/outcomes?token=secret"),
        ]
        for method, path in invalid:
            with self.subTest(path=path):
                with self.assertRaisesRegex(client_tool.ClientError, "invalid_request_path"):
                    client_tool.build_request(method, path, TEST_PROVIDER_KEY)
        self.assertEqual(client_tool.BASE_URL, "https://nothumansearch.ai")

    def test_http_and_network_failures_never_read_or_reflect_private_details(self):
        private_body = io.BytesIO(
            ("private server body " + TEST_PROVIDER_KEY + " " + TEST_ATTRIBUTION).encode("ascii")
        )
        error = urllib.error.HTTPError(
            client_tool.BASE_URL + "/api/v1/provider/pilot-status",
            401,
            "private diagnostic " + TEST_PROVIDER_KEY,
            {},
            private_body,
        )
        request = client_tool.build_request(
            "GET", "/api/v1/provider/pilot-status?limit=25", TEST_PROVIDER_KEY
        )
        with self.assertRaises(client_tool.ClientError) as raised:
            client_tool.perform_request(
                FakeOpener(error), request, success_statuses=frozenset({200})
            )
        self.assertEqual(raised.exception.code, "http_error")
        self.assertEqual(raised.exception.http_status, 401)
        self.assertEqual(private_body.tell(), 0)
        self.assertNotIn(TEST_PROVIDER_KEY, str(raised.exception))
        self.assertIsNone(raised.exception.__cause__)
        self.assertEqual(client_tool.exit_code_for_error(raised.exception), client_tool.EXIT_AUTH)

        network = urllib.error.URLError("private network " + TEST_ATTRIBUTION)
        with self.assertRaisesRegex(client_tool.ClientError, "network_error") as network_error:
            client_tool.perform_request(
                FakeOpener(network), request, success_statuses=frozenset({200})
            )
        self.assertNotIn(TEST_ATTRIBUTION, str(network_error.exception))
        self.assertIsNone(network_error.exception.__cause__)
        self.assertEqual(client_tool.exit_code_for_error(network_error.exception), client_tool.EXIT_NETWORK)

    def test_response_body_is_bounded_and_non_json_fails_closed(self):
        request = client_tool.build_request(
            "GET", "/api/v1/provider/pilot-status?limit=25", TEST_PROVIDER_KEY
        )
        oversized = FakeResponse(200, b"x" * (client_tool.MAX_RESPONSE_BYTES + 50))
        with self.assertRaisesRegex(client_tool.ClientError, "response_too_large"):
            client_tool.perform_request(
                FakeOpener(oversized), request, success_statuses=frozenset({200})
            )
        self.assertEqual(oversized.read_sizes, [client_tool.MAX_RESPONSE_BYTES + 1])
        with self.assertRaisesRegex(client_tool.ClientError, "invalid_response"):
            client_tool.perform_request(
                FakeOpener(FakeResponse(200, b"[]")), request, success_statuses=frozenset({200})
            )

    def test_main_disables_core_dumps_emits_safe_json_and_uses_distinct_exit_codes(self):
        stdout = io.StringIO()
        with (
            mock.patch.object(client_tool.resource, "setrlimit") as set_limit,
            mock.patch.object(client_tool, "run", return_value={"ok": True, "command": "status"}),
            contextlib.redirect_stdout(stdout),
        ):
            code = client_tool.main(["status"])
        self.assertEqual(code, client_tool.EXIT_OK)
        set_limit.assert_called_once_with(client_tool.resource.RLIMIT_CORE, (0, 0))
        self.assertEqual(json.loads(stdout.getvalue()), {"ok": True, "command": "status"})

        cases = [
            (client_tool.ClientError("invalid_arguments"), client_tool.EXIT_INPUT),
            (client_tool.ClientError("network_error"), client_tool.EXIT_NETWORK),
            (client_tool.ClientError("http_error", http_status=401), client_tool.EXIT_AUTH),
            (client_tool.ClientError("http_error", http_status=429), client_tool.EXIT_RATE_LIMIT),
            (client_tool.ClientError("http_error", http_status=503), client_tool.EXIT_REMOTE),
            (client_tool.ClientError("invalid_response"), client_tool.EXIT_RESPONSE),
        ]
        for error, expected in cases:
            stderr = io.StringIO()
            with (
                mock.patch.object(client_tool, "disable_core_dumps"),
                mock.patch.object(client_tool, "run", side_effect=error),
                contextlib.redirect_stderr(stderr),
            ):
                code = client_tool.main(["status"])
            self.assertEqual(code, expected)
            emitted = stderr.getvalue()
            self.assertNotIn(TEST_PROVIDER_KEY, emitted)
            self.assertNotIn(TEST_ATTRIBUTION, emitted)
            self.assertEqual(json.loads(emitted)["error"], error.code)

        stderr = io.StringIO()
        with (
            mock.patch.object(client_tool, "disable_core_dumps"),
            mock.patch.object(client_tool, "run", side_effect=RuntimeError(TEST_ATTRIBUTION)),
            contextlib.redirect_stderr(stderr),
        ):
            code = client_tool.main(["status"])
        self.assertEqual(code, client_tool.EXIT_INTERNAL)
        self.assertEqual(json.loads(stderr.getvalue()), {"ok": False, "error": "internal_error"})


if __name__ == "__main__":
    unittest.main()
