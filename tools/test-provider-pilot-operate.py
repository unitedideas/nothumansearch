#!/usr/bin/env python3
"""Security and contract tests for provider-pilot-operate.py."""

from __future__ import annotations

import contextlib
import hashlib
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


MODULE_PATH = pathlib.Path(__file__).with_name("provider-pilot-operate.py")
SPEC = importlib.util.spec_from_file_location("provider_pilot_operate", MODULE_PATH)
operate = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = operate
SPEC.loader.exec_module(operate)

ADMIN_KEY = "fixture-admin-key"
ENVIRONMENT = {operate.ADMIN_KEY_ENV: ADMIN_KEY}
CLAIM_ID = "123e4567-e89b-42d3-a456-426614174000"
OFFER_ID = "223e4567-e89b-42d3-a456-426614174000"
ACCEPTANCE_ID = "323e4567-e89b-42d3-a456-426614174000"
COMMITMENT_ID = "423e4567-e89b-42d3-a456-426614174000"
RELATED_COMMITMENT_ID = "523e4567-e89b-42d3-a456-426614174000"
PILOT_ID = "623e4567-e89b-42d3-a456-426614174000"
ENROLLMENT_ID = "723e4567-e89b-42d3-a456-426614174000"
COMPANY_ID = "823e4567-e89b-42d3-a456-426614174000"
REVIEW_ID = "923e4567-e89b-42d3-a456-426614174000"
MANIFEST_ID = "a23e4567-e89b-42d3-a456-426614174000"
SNAPSHOT_SHA256 = "b" * 64
REVIEW_EVIDENCE_SHA256 = "c" * 64
MANIFEST_KEY_ID = "nhs-provider-signing-v1"


class FakeResponse:
    def __init__(self, status: int, body: bytes):
        self.status = status
        self.body = body
        self.read_sizes: list[int] = []

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        return False

    def getcode(self):
        return self.status

    def read(self, size: int):
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


def response(document: dict[str, object], status: int = 200) -> FakeResponse:
    return FakeResponse(status, json.dumps(document).encode("utf-8"))


def queue_document(extra_item: dict[str, object] | None = None) -> dict[str, object]:
    item: dict[str, object] = {
        "state": "pending_terms",
        "provider_claim_id": CLAIM_ID,
        "domain": "provider.example",
        "acceptance_event_id": ACCEPTANCE_ID,
        "occurred_at": "2026-08-02T10:00:00Z",
    }
    if extra_item:
        item.update(extra_item)
    return {
        "queue": {
            "as_of": "2026-08-02T10:01:00Z",
            "state": "all",
            "limit_per_state": 25,
            "returned_counts": {"pending_terms": 1},
            "items": [item],
            "redaction_contract": "reviewed",
        },
        "evidence_scope": "read only",
    }


def review_preflight_document(*, blocked: bool = True) -> dict[str, object]:
    items: list[dict[str, object]] = []
    counts: dict[str, int] = {}
    if blocked:
        items.append(
            {
                "state": "ticket_review_required",
                "provider_pilot_epoch_id": PILOT_ID,
                "provider_claim_id": CLAIM_ID,
                "domain": "provider.example",
                "offer_id": OFFER_ID,
                "offer_version": 1,
                "commercial_terms_sha256": "a" * 64,
                "ticket_id": COMMITMENT_ID,
                "review_type": "ticket",
                "subject_id": COMMITMENT_ID,
                "subject_snapshot_sha256": SNAPSHOT_SHA256,
                "occurred_at": "2026-08-02T10:00:00Z",
                "valid_until": "2026-08-03T10:00:00Z",
            }
        )
        counts["ticket_review_required"] = 1
    return {
        "queue": {
            "as_of": "2026-08-02T10:01:00Z",
            "state": "pre_event_review_required",
            "limit_per_state": 25,
            "returned_counts": counts,
            "items": items,
            "redaction_contract": "review coordinates only",
        },
        "evidence_scope": "read only",
    }


def commercial_offer_review_preflight_document(*, blocked: bool = True) -> dict[str, object]:
    items: list[dict[str, object]] = []
    counts: dict[str, int] = {}
    if blocked:
        items = [
            {
                "state": "offer_review_required",
                "provider_pilot_epoch_id": PILOT_ID,
                "provider_claim_id": CLAIM_ID,
                "domain": "prepaid.provider.example",
                "offer_id": OFFER_ID,
                "offer_version": 1,
                "commercial_terms_sha256": "a" * 64,
                "commitment_event_id": COMMITMENT_ID,
                "commitment_event_type": "prepaid_fund",
                "review_type": "offer",
                "subject_id": OFFER_ID,
                "subject_snapshot_sha256": SNAPSHOT_SHA256,
                "occurred_at": "2026-08-02T10:00:00Z",
            },
            {
                "state": "offer_review_required",
                "provider_pilot_epoch_id": PILOT_ID,
                "provider_claim_id": CLAIM_ID,
                "domain": "terms.provider.example",
                "offer_id": RELATED_COMMITMENT_ID,
                "offer_version": 1,
                "commercial_terms_sha256": "d" * 64,
                "commitment_event_id": ACCEPTANCE_ID,
                "commitment_event_type": "terms_renewal",
                "review_type": "offer",
                "subject_id": RELATED_COMMITMENT_ID,
                "subject_snapshot_sha256": "e" * 64,
                "occurred_at": "2026-08-02T10:00:01Z",
            },
        ]
        counts["offer_review_required"] = 2
    return {
        "queue": {
            "as_of": "2026-08-02T10:01:00Z",
            "state": "pre_event_review_required",
            "limit_per_state": 25,
            "returned_counts": counts,
            "items": items,
            "redaction_contract": "review coordinates and bounded commitment type only",
        },
        "evidence_scope": "read only",
    }


def pilot_epoch(status: str = "draft", extra: dict[str, object] | None = None) -> dict[str, object]:
    epoch: dict[str, object] = {
        "id": PILOT_ID,
        "contract_version": operate.PILOT_CONTRACT_VERSION,
        "demand_topic": "developer-tools",
        "stage1_started_at": "2026-06-01T00:00:00Z",
        "stage1_evidence_as_of": "2026-07-02T00:00:00Z",
        "stage1_evidence_sha256": "a" * 64,
        "cohort_limit": 3,
        "provider_ticket_cap": 5,
        "total_ticket_cap": 5,
        "status": status,
        "owner_reference": "owner:pilot-case-001",
        "evidence_reference": "evidence:pilot-case-001",
        "created_at": "2026-08-02T10:00:00Z",
        "updated_at": "2026-08-02T10:00:00Z",
    }
    if status in {"active", "closed"}:
        epoch["activated_at"] = "2026-08-02T10:10:00Z"
        epoch["updated_at"] = "2026-08-02T10:10:00Z"
    if status == "closed":
        epoch["closed_at"] = "2026-08-02T11:00:00Z"
        epoch["updated_at"] = "2026-08-02T11:00:00Z"
    if extra:
        epoch.update(extra)
    return epoch


def pilot_mutation_document(action: str, value: dict[str, object]) -> dict[str, object]:
    return {
        "action": action,
        "provider_pilot": value,
        "commercial_proof_created": False,
        "evidence_scope": operate.PILOT_MUTATION_EVIDENCE_SCOPE,
    }


def pilot_enrollment_document(extra: dict[str, object] | None = None) -> dict[str, object]:
    enrollment: dict[str, object] = {
        "id": ENROLLMENT_ID,
        "provider_pilot_epoch_id": PILOT_ID,
        "provider_pilot_company_id": COMPANY_ID,
        "provider_claim_id": CLAIM_ID,
        "owner_reference": "owner:pilot-case-001",
        "evidence_reference": "evidence:pilot-case-001",
        "enrolled_at": "2026-08-02T10:05:00Z",
    }
    if extra:
        enrollment.update(extra)
    return pilot_mutation_document("enroll", enrollment)


def pilot_status_document(extra: dict[str, object] | None = None) -> dict[str, object]:
    status = pilot_epoch(
        "active",
        {
            "as_of": "2026-08-02T10:30:00Z",
            "enrolled_provider_count": 3,
            "fresh_enrolled_provider_count": 3,
            "non_synthetic_ticket_count": 2,
            "remaining_ticket_capacity": 3,
            "event_count": 12,
            "cohort_ready": True,
        },
    )
    if extra:
        status.update(extra)
    return {"provider_pilot": status, "evidence_scope": operate.PILOT_STATUS_EVIDENCE_SCOPE}


def review_candidate_document(extra: dict[str, object] | None = None) -> dict[str, object]:
    candidate: dict[str, object] = {
        "review_contract_version": operate.PILOT_REVIEW_CONTRACT_VERSION,
        "provider_pilot_epoch_id": PILOT_ID,
        "provider_pilot_contract_version": operate.PILOT_CONTRACT_VERSION,
        "pilot_demand_topic": "developer-tools",
        "review_type": "provider",
        "subject_id": CLAIM_ID,
        "subject_snapshot_sha256": SNAPSHOT_SHA256,
        "provider_claim_id": CLAIM_ID,
        "domain": "provider.example",
        "provider_pilot_company_id": COMPANY_ID,
        "provider_pilot_enrollment_id": ENROLLMENT_ID,
        "provider_acceptance_event_id": ACCEPTANCE_ID,
        "provider_acceptance_reference": "provider:review:company-001",
        "provider_accepted_at": "2026-08-02T10:00:00Z",
        "company_owner_verified_at": "2026-08-02T10:01:00Z",
        "enrolled_at": "2026-08-02T10:02:00Z",
    }
    if extra:
        candidate.update(extra)
    return {
        "review_candidate": candidate,
        "evidence_scope": operate.PILOT_REVIEW_EVIDENCE_SCOPE,
    }


def renewed_terms_offer_review_candidate_document(
    extra: dict[str, object] | None = None,
) -> dict[str, object]:
    candidate: dict[str, object] = {
        "review_contract_version": operate.PILOT_REVIEW_CONTRACT_VERSION,
        "provider_pilot_epoch_id": PILOT_ID,
        "provider_pilot_contract_version": operate.PILOT_CONTRACT_VERSION,
        "pilot_demand_topic": "developer-tools",
        "review_type": "offer",
        "subject_id": OFFER_ID,
        "subject_snapshot_sha256": SNAPSHOT_SHA256,
        "provider_claim_id": CLAIM_ID,
        "domain": "terms.provider.example",
        "provider_pilot_company_id": COMPANY_ID,
        "provider_offer_id": OFFER_ID,
        "offer_version": 1,
        "offer_name": "Terms offer",
        "offer_summary": "Current renewed terms.",
        "action_type": "lead",
        "action_url": "https://terms.provider.example/start",
        "disclosure_label": "Sponsored provider option",
        "charge_event": "accepted",
        "bounty_cents": 1_000,
        "currency": "usd",
        "principal_price_mode": "free",
        "principal_currency": "usd",
        "billing_mode": "terms",
        "terms_credit_limit_cents": 10_000,
        "terms_period_days": 30,
        "commercial_terms_contract_version": "nhs-provider-commercial-terms-v1",
        "commercial_terms_sha256": "a" * 64,
        "commitment_event_id": COMMITMENT_ID,
        "commitment_event_type": "terms_renewal",
        "provider_acceptance_event_id": ACCEPTANCE_ID,
        "commitment_provider_accepted_at": "2026-08-02T10:00:00Z",
        "commitment_valid_until": "2026-09-01T10:00:00Z",
        "commitment_owner_verified_at": "2026-08-02T10:01:00Z",
    }
    if extra:
        candidate.update(extra)
    return {
        "review_candidate": candidate,
        "evidence_scope": operate.PILOT_REVIEW_EVIDENCE_SCOPE,
    }


def recorded_review_document(*, created: bool = True) -> dict[str, object]:
    return {
        "review": {
            "id": REVIEW_ID,
            "provider_pilot_epoch_id": PILOT_ID,
            "review_contract_version": operate.PILOT_REVIEW_CONTRACT_VERSION,
            "review_type": "provider",
            "subject_id": CLAIM_ID,
            "provider_claim_id": CLAIM_ID,
            "subject_snapshot_sha256": SNAPSHOT_SHA256,
            "owner_reference": "owner:review:provider-001",
            "evidence_reference": "evidence:review:provider-001",
            "reviewed_at": "2026-08-02T10:03:00Z",
        },
        "created": created,
        "idempotent_replay": not created,
        "commercial_proof_created": False,
        "evidence_scope": operate.PILOT_REVIEW_EVIDENCE_SCOPE,
    }


def signed_proof_manifest() -> dict[str, object]:
    return {
        "v": 1,
        "kid": MANIFEST_KEY_ID,
        "signature_verification_scope": operate.PROOF_MANIFEST_VERIFICATION_SCOPE,
        "manifest_contract_version": operate.PROOF_MANIFEST_CONTRACT_VERSION,
        "manifest_id": MANIFEST_ID,
        "provider_pilot_epoch_id": PILOT_ID,
        "provider_pilot_contract_version": operate.PILOT_CONTRACT_VERSION,
        "review_contract_version": operate.PILOT_REVIEW_CONTRACT_VERSION,
        "review_evidence_contract_version": operate.PROOF_MANIFEST_REVIEW_EVIDENCE_VERSION,
        "market_policy_contract_version": operate.PROOF_MANIFEST_MARKET_POLICY_VERSION,
        "proof_snapshot_sha256": SNAPSHOT_SHA256,
        "review_evidence_sha256": REVIEW_EVIDENCE_SHA256,
        "pilot_demand_topic": "developer-tools",
        "pilot_status": "closed",
        "issued_at": 1785672000,
        "outcome_receipt_integrity_valid": True,
        "review_integrity_valid": True,
        "verified_outcome_receipts": 8,
        "rejected_outcome_receipts": 0,
        "verified_outcome_ledger_entries": 8,
        "rejected_outcome_ledger_entries": 0,
        "verified_provider_companies": 3,
        "verified_provider_accepted_handoffs": 5,
        "verified_provider_confirmed_activations": 2,
        "verified_provider_renewals": 1,
        "verified_provider_confirmed_conversions": 1,
        "review_coverage": {
            "providers": {"required": 3, "valid": 3},
            "offers": {"required": 3, "valid": 3},
            "tickets": {"required": 5, "valid": 5},
            "handoffs": {"required": 5, "valid": 5},
            "callbacks": {"required": 8, "valid": 8},
        },
        "monetary_amounts_withheld_for_privacy": True,
        "verified_prepaid_settled": [],
        "verified_prepaid_net_debited": [],
        "verified_terms_net_receivable": [],
        "pilot_thresholds_met": True,
        "organic_rank_sold": False,
        "raw_queries_sold": False,
        "agent_identities_sold": False,
        "evidence_scope": operate.PROOF_MANIFEST_SCOPE,
    }


def issued_proof_manifest_document(*, created: bool = True) -> dict[str, object]:
    signed = json.dumps(signed_proof_manifest(), separators=(",", ":"))
    return {
        "manifest": {
            "id": MANIFEST_ID,
            "provider_pilot_epoch_id": PILOT_ID,
            "manifest_contract_version": operate.PROOF_MANIFEST_CONTRACT_VERSION,
            "proof_snapshot_sha256": SNAPSHOT_SHA256,
            "review_evidence_sha256": REVIEW_EVIDENCE_SHA256,
            "key_id": MANIFEST_KEY_ID,
            "signed_manifest": signed,
            "signature": "A" * 43,
            "payload_sha256": hashlib.sha256(signed.encode("utf-8")).hexdigest(),
            "issued_at": "2026-08-02T12:00:00Z",
        },
        "created": created,
        "idempotent_replay": not created,
        "commercial_proof_created": True,
        "publicly_released": False,
        "independently_verifiable": False,
        "evidence_scope": operate.PROOF_MANIFEST_EVIDENCE_SCOPE,
    }


class TransportSecurityTest(unittest.TestCase):
    def test_opener_disables_proxies_refuses_redirects_and_uses_tls(self):
        with mock.patch.object(operate.urllib.request, "build_opener", return_value="fixture") as builder:
            self.assertEqual(operate.build_opener(), "fixture")
        configured_handlers = builder.call_args.args
        proxy_handler = next(handler for handler in configured_handlers if isinstance(handler, urllib.request.ProxyHandler))
        self.assertEqual(proxy_handler.proxies, {})
        opener = operate.build_opener()
        self.assertTrue(any(isinstance(handler, operate.NoRedirect) for handler in opener.handlers))
        https_handler = next(handler for handler in opener.handlers if isinstance(handler, urllib.request.HTTPSHandler))
        self.assertIsInstance(https_handler._context, ssl.SSLContext)
        self.assertEqual(https_handler._context.verify_mode, ssl.CERT_REQUIRED)
        self.assertTrue(https_handler._context.check_hostname)

    def test_request_paths_and_host_are_fixed(self):
        request = operate.build_request(operate.QUEUE_PATH + "?state=all&limit=25", ADMIN_KEY)
        self.assertEqual(request.full_url, operate.BASE_URL + operate.QUEUE_PATH + "?state=all&limit=25")
        self.assertEqual(request.get_method(), "GET")
        self.assertEqual(request.get_header("Authorization"), "Bearer " + ADMIN_KEY)
        pilot_request = operate.build_request(operate.PILOT_EPOCH_PATH + "?pilot_id=" + PILOT_ID, ADMIN_KEY)
        self.assertEqual(pilot_request.full_url, operate.BASE_URL + operate.PILOT_EPOCH_PATH + "?pilot_id=" + PILOT_ID)
        self.assertEqual(pilot_request.get_method(), "GET")
        with self.assertRaisesRegex(operate.OperatorError, "invalid_operator_path"):
            operate.build_request(operate.PILOT_EPOCH_PATH, ADMIN_KEY)
        with self.assertRaisesRegex(operate.OperatorError, "invalid_operator_path"):
            operate.build_request(operate.PILOT_EPOCH_PATH + "?pilot_id=" + PILOT_ID + "&raw=true", ADMIN_KEY)
        review_query = urllib.parse.urlencode(
            {"pilot_id": PILOT_ID, "review_type": "provider", "subject_id": CLAIM_ID}
        )
        review_request = operate.build_request(operate.PILOT_REVIEW_PATH + "?" + review_query, ADMIN_KEY)
        self.assertEqual(review_request.get_method(), "GET")
        with self.assertRaisesRegex(operate.OperatorError, "invalid_operator_path"):
            operate.build_request(operate.PILOT_REVIEW_PATH, ADMIN_KEY)
        with self.assertRaisesRegex(operate.OperatorError, "invalid_operator_path"):
            operate.build_request(operate.PILOT_REVIEW_PATH + "?" + review_query + "&query=private", ADMIN_KEY)
        manifest_request = operate.build_request(
            operate.PROOF_MANIFEST_PATH,
            ADMIN_KEY,
            payload={
                "provider_pilot_epoch_id": PILOT_ID,
                "expected_snapshot_sha256": SNAPSHOT_SHA256,
                "owner_reference": "owner:proof:case-001",
                "evidence_reference": "evidence:proof:case-001",
            },
        )
        self.assertEqual(manifest_request.get_method(), "POST")
        with self.assertRaisesRegex(operate.OperatorError, "invalid_operator_path"):
            operate.build_request(operate.PROOF_MANIFEST_PATH, ADMIN_KEY)
        with self.assertRaisesRegex(operate.OperatorError, "invalid_operator_path"):
            operate.build_request(
                operate.PROOF_MANIFEST_PATH + "?pilot_id=" + PILOT_ID,
                ADMIN_KEY,
                payload={"provider_pilot_epoch_id": PILOT_ID},
            )
        with self.assertRaisesRegex(operate.OperatorError, "invalid_operator_path"):
            operate.build_request("https://attacker.example/collect", ADMIN_KEY)

    def test_one_bounded_request_and_no_error_body_read(self):
        fake = FakeOpener(response=response(queue_document()))
        result = operate.run(["queue"], environ=ENVIRONMENT, opener=fake)
        self.assertTrue(result["ok"])
        self.assertEqual(len(fake.calls), 1)
        self.assertEqual(fake.calls[0][1], operate.REQUEST_TIMEOUT_SECONDS)
        self.assertEqual(fake.response.read_sizes, [operate.MAX_RESPONSE_BYTES + 1])

        error = urllib.error.HTTPError(
            operate.BASE_URL + operate.QUEUE_PATH, 401, "private token text", {}, io.BytesIO(b"secret body")
        )
        failing = FakeOpener(error=error)
        with self.assertRaisesRegex(operate.OperatorError, "http_error"):
            operate.run(["queue"], environ=ENVIRONMENT, opener=failing)

    def test_response_size_and_redirect_fail_closed(self):
        oversized = FakeOpener(response=FakeResponse(200, b"x" * (operate.MAX_RESPONSE_BYTES + 1)))
        with self.assertRaisesRegex(operate.OperatorError, "response_too_large"):
            operate.run(["queue"], environ=ENVIRONMENT, opener=oversized)
        redirected = FakeOpener(response=FakeResponse(302, b"{}"))
        with self.assertRaisesRegex(operate.OperatorError, "redirect_refused"):
            operate.run(["queue"], environ=ENVIRONMENT, opener=redirected)


class CommandContractTest(unittest.TestCase):
    def test_queue_projects_only_reviewed_fields(self):
        fake = FakeOpener(response=response(queue_document()))
        result = operate.run(["queue"], environ=ENVIRONMENT, opener=fake)
        self.assertEqual(
            set(result),
            {"ok", "action", "as_of", "state", "limit_per_state", "returned_counts", "items", "commercial_proof_created"},
        )
        self.assertEqual(result["returned_counts"], {"pending_terms": 1})
        self.assertFalse(result["commercial_proof_created"])
        self.assertNotIn(ADMIN_KEY, json.dumps(result))

    def test_queue_rejects_sensitive_or_inconsistent_response(self):
        with_sensitive = queue_document({"attribution_token": "sensitive-token"})
        with self.assertRaisesRegex(operate.OperatorError, "invalid_response"):
            operate.run(["queue"], environ=ENVIRONMENT, opener=FakeOpener(response=response(with_sensitive)))
        inconsistent = queue_document()
        inconsistent["queue"]["returned_counts"] = {"pending_terms": 2}
        with self.assertRaisesRegex(operate.OperatorError, "invalid_response"):
            operate.run(["queue"], environ=ENVIRONMENT, opener=FakeOpener(response=response(inconsistent)))

    def test_review_preflight_projects_exact_pre_event_blockers(self):
        fake = FakeOpener(response=response(review_preflight_document()))
        result = operate.run(["review-preflight"], environ=ENVIRONMENT, opener=fake)
        self.assertEqual(result["action"], "review_preflight")
        self.assertEqual(result["state"], "pre_event_review_required")
        self.assertFalse(result["pre_event_reviews_ready"])
        self.assertEqual(result["review_blocker_count"], 1)
        self.assertEqual(result["items"][0]["review_type"], "ticket")
        self.assertEqual(result["items"][0]["subject_snapshot_sha256"], SNAPSHOT_SHA256)
        request = fake.calls[0][0]
        self.assertEqual(request.get_method(), "GET")
        query = urllib.parse.parse_qs(urllib.parse.urlsplit(request.full_url).query)
        self.assertEqual(query, {"state": ["pre_event_review_required"], "limit": ["25"]})

        ready = operate.run(
            ["review-preflight"],
            environ=ENVIRONMENT,
            opener=FakeOpener(response=response(review_preflight_document(blocked=False))),
        )
        self.assertTrue(ready["pre_event_reviews_ready"])
        self.assertEqual(ready["review_blocker_count"], 0)

    def test_review_preflight_projects_commercial_offer_entries_then_removal(self):
        blocked = operate.run(
            ["review-preflight"],
            environ=ENVIRONMENT,
            opener=FakeOpener(
                response=response(commercial_offer_review_preflight_document())
            ),
        )
        self.assertFalse(blocked["pre_event_reviews_ready"])
        self.assertEqual(blocked["review_blocker_count"], 2)
        self.assertEqual(
            {
                (item["commitment_event_type"], item["commitment_event_id"])
                for item in blocked["items"]
            },
            {
                ("prepaid_fund", COMMITMENT_ID),
                ("terms_renewal", ACCEPTANCE_ID),
            },
        )

        after_reviews = operate.run(
            ["review-preflight"],
            environ=ENVIRONMENT,
            opener=FakeOpener(
                response=response(
                    commercial_offer_review_preflight_document(blocked=False)
                )
            ),
        )
        self.assertTrue(after_reviews["pre_event_reviews_ready"])
        self.assertEqual(after_reviews["review_blocker_count"], 0)
        self.assertEqual(after_reviews["items"], [])

    def test_review_preflight_rejects_unknown_commitment_type(self):
        document = commercial_offer_review_preflight_document()
        document["queue"]["items"][0]["commitment_event_type"] = "unverified_fund"
        with self.assertRaisesRegex(operate.OperatorError, "invalid_response"):
            operate.run(
                ["review-preflight"],
                environ=ENVIRONMENT,
                opener=FakeOpener(response=response(document)),
            )

    def test_review_preflight_rejects_missing_snapshot_coordinates(self):
        document = review_preflight_document()
        del document["queue"]["items"][0]["subject_snapshot_sha256"]
        with self.assertRaisesRegex(operate.OperatorError, "invalid_response"):
            operate.run(
                ["review-preflight"],
                environ=ENVIRONMENT,
                opener=FakeOpener(response=response(document)),
            )

    def test_mutations_require_explicit_owner_authority_before_network(self):
        fake = FakeOpener(response=AssertionError("network must not be called"))
        cases = {
            "activate": [
                "--offer-id", OFFER_ID,
                "--operator-reference", "owner-case-001",
                "--evidence-reference", "evidence-case-001",
            ],
            "pause": [
                "--offer-id", OFFER_ID,
                "--operator-reference", "owner-case-001",
                "--evidence-reference", "evidence-case-001",
            ],
            "authorize-pilot": [
                "--topic", "developer-tools",
                "--cohort-limit", "3",
                "--provider-ticket-cap", "5",
                "--total-ticket-cap", "5",
                "--owner-reference", "owner:pilot-case-001",
                "--evidence-reference", "evidence:pilot-case-001",
            ],
            "enroll-pilot": [
                "--pilot-id", PILOT_ID,
                "--claim-id", CLAIM_ID,
                "--owner-reference", "owner:pilot-case-001",
                "--evidence-reference", "evidence:pilot-case-001",
            ],
            "activate-pilot": [
                "--pilot-id", PILOT_ID,
                "--owner-reference", "owner:pilot-case-001",
                "--evidence-reference", "evidence:pilot-case-001",
            ],
            "close-pilot": [
                "--pilot-id", PILOT_ID,
                "--owner-reference", "owner:pilot-case-001",
                "--evidence-reference", "evidence:pilot-case-001",
            ],
            "record-review": [
                "--pilot-id", PILOT_ID,
                "--review-type", "provider",
                "--subject-id", CLAIM_ID,
                "--expected-snapshot-sha256", SNAPSHOT_SHA256,
                "--owner-reference", "owner:review:provider-001",
                "--evidence-reference", "evidence:review:provider-001",
            ],
            "issue-proof-manifest": [
                "--pilot-id", PILOT_ID,
                "--expected-snapshot-sha256", SNAPSHOT_SHA256,
                "--owner-reference", "owner:proof:case-001",
                "--evidence-reference", "evidence:proof:case-001",
            ],
        }
        for command, arguments in cases.items():
            with self.subTest(command=command):
                with self.assertRaisesRegex(operate.OperatorError, "owner_authorization_required"):
                    operate.run([command, *arguments], environ=ENVIRONMENT, opener=fake)
        self.assertEqual(fake.calls, [])

    def test_review_candidate_projects_exact_privacy_bounded_snapshot(self):
        fake = FakeOpener(response=response(review_candidate_document()))
        result = operate.run(
            [
                "review-candidate",
                "--pilot-id", PILOT_ID,
                "--review-type", "provider",
                "--subject-id", CLAIM_ID,
            ],
            environ=ENVIRONMENT,
            opener=fake,
        )
        self.assertEqual(result["action"], "review_candidate")
        self.assertEqual(result["review_candidate"]["subject_snapshot_sha256"], SNAPSHOT_SHA256)
        self.assertFalse(result["commercial_proof_created"])
        request = fake.calls[0][0]
        self.assertEqual(request.get_method(), "GET")
        self.assertTrue(request.full_url.startswith(operate.BASE_URL + operate.PILOT_REVIEW_PATH + "?"))
        self.assertNotIn(ADMIN_KEY, json.dumps(result))

    def test_review_candidate_projects_current_terms_renewal_without_expanding_pilot_policy(self):
        result = operate.run(
            [
                "review-candidate",
                "--pilot-id", PILOT_ID,
                "--review-type", "offer",
                "--subject-id", OFFER_ID,
            ],
            environ=ENVIRONMENT,
            opener=FakeOpener(
                response=response(renewed_terms_offer_review_candidate_document())
            ),
        )
        candidate = result["review_candidate"]
        self.assertEqual(candidate["billing_mode"], "terms")
        self.assertEqual(candidate["commitment_event_type"], "terms_renewal")
        self.assertEqual(candidate["commitment_event_id"], COMMITMENT_ID)

        prepaid = renewed_terms_offer_review_candidate_document(
            {
                "billing_mode": "prepaid",
                "commitment_event_type": "prepaid_fund",
            }
        )
        with self.assertRaisesRegex(operate.OperatorError, "invalid_response"):
            operate.run(
                [
                    "review-candidate",
                    "--pilot-id", PILOT_ID,
                    "--review-type", "offer",
                    "--subject-id", OFFER_ID,
                ],
                environ=ENVIRONMENT,
                opener=FakeOpener(response=response(prepaid)),
            )

    def test_review_candidate_rejects_sensitive_or_cross_subject_response(self):
        sensitive = review_candidate_document({"query": "private words"})
        with self.assertRaisesRegex(operate.OperatorError, "invalid_response"):
            operate.run(
                [
                    "review-candidate", "--pilot-id", PILOT_ID,
                    "--review-type", "provider", "--subject-id", CLAIM_ID,
                ],
                environ=ENVIRONMENT,
                opener=FakeOpener(response=response(sensitive)),
            )
        cross_subject = review_candidate_document({"provider_claim_id": OFFER_ID})
        with self.assertRaisesRegex(operate.OperatorError, "invalid_response"):
            operate.run(
                [
                    "review-candidate", "--pilot-id", PILOT_ID,
                    "--review-type", "provider", "--subject-id", CLAIM_ID,
                ],
                environ=ENVIRONMENT,
                opener=FakeOpener(response=response(cross_subject)),
            )

    def test_record_review_sends_exact_digest_and_disclaims_commercial_proof(self):
        fake = FakeOpener(response=response(recorded_review_document(), 201))
        result = operate.run(
            [
                "record-review",
                "--pilot-id", PILOT_ID,
                "--review-type", "provider",
                "--subject-id", CLAIM_ID,
                "--expected-snapshot-sha256", SNAPSHOT_SHA256,
                "--owner-reference", "owner:review:provider-001",
                "--evidence-reference", "evidence:review:provider-001",
                "--confirm-owner-authorized",
            ],
            environ=ENVIRONMENT,
            opener=fake,
        )
        self.assertEqual(result["action"], "record_review")
        self.assertEqual(result["review_id"], REVIEW_ID)
        self.assertTrue(result["created"])
        self.assertFalse(result["idempotent_replay"])
        self.assertFalse(result["commercial_proof_created"])
        request = fake.calls[0][0]
        self.assertEqual(request.full_url, operate.BASE_URL + operate.PILOT_REVIEW_PATH)
        self.assertEqual(request.get_method(), "POST")
        self.assertEqual(
            json.loads(request.data),
            {
                "provider_pilot_epoch_id": PILOT_ID,
                "review_type": "provider",
                "subject_id": CLAIM_ID,
                "expected_snapshot_sha256": SNAPSHOT_SHA256,
                "owner_reference": "owner:review:provider-001",
                "evidence_reference": "evidence:review:provider-001",
            },
        )

    def test_record_review_rejects_receipt_drift(self):
        drifted = recorded_review_document()
        drifted["review"]["subject_snapshot_sha256"] = "c" * 64
        with self.assertRaisesRegex(operate.OperatorError, "invalid_response"):
            operate.run(
                [
                    "record-review", "--pilot-id", PILOT_ID,
                    "--review-type", "provider", "--subject-id", CLAIM_ID,
                    "--expected-snapshot-sha256", SNAPSHOT_SHA256,
                    "--owner-reference", "owner:review:provider-001",
                    "--evidence-reference", "evidence:review:provider-001",
                    "--confirm-owner-authorized",
                ],
                environ=ENVIRONMENT,
                opener=FakeOpener(response=response(drifted, 201)),
            )

    def test_issue_proof_manifest_sends_exact_owner_authorized_shape(self):
        fake = FakeOpener(response=response(issued_proof_manifest_document(), 201))
        result = operate.run(
            [
                "issue-proof-manifest",
                "--pilot-id", PILOT_ID,
                "--expected-snapshot-sha256", SNAPSHOT_SHA256,
                "--owner-reference", "owner:proof:case-001",
                "--evidence-reference", "evidence:proof:case-001",
                "--confirm-owner-authorized",
            ],
            environ=ENVIRONMENT,
            opener=fake,
        )
        self.assertEqual(
            set(result),
            {
                "ok", "action", "http_status", "created", "idempotent_replay", "manifest_id",
                "pilot_id", "manifest_contract_version", "proof_snapshot_sha256", "key_id",
                "review_evidence_sha256", "payload_sha256", "issued_at",
                "commercial_proof_created", "publicly_released", "independently_verifiable",
            },
        )
        self.assertEqual(result["action"], "issue_proof_manifest")
        self.assertEqual(result["manifest_id"], MANIFEST_ID)
        self.assertTrue(result["created"])
        self.assertFalse(result["idempotent_replay"])
        self.assertTrue(result["commercial_proof_created"])
        self.assertFalse(result["publicly_released"])
        self.assertFalse(result["independently_verifiable"])
        self.assertEqual(result["review_evidence_sha256"], REVIEW_EVIDENCE_SHA256)
        serialized = json.dumps(result, sort_keys=True)
        self.assertNotIn("owner:proof:case-001", serialized)
        self.assertNotIn("evidence:proof:case-001", serialized)
        self.assertNotIn(ADMIN_KEY, serialized)
        request = fake.calls[0][0]
        self.assertEqual(request.full_url, operate.BASE_URL + operate.PROOF_MANIFEST_PATH)
        self.assertEqual(request.get_method(), "POST")
        self.assertEqual(
            json.loads(request.data),
            {
                "provider_pilot_epoch_id": PILOT_ID,
                "expected_snapshot_sha256": SNAPSHOT_SHA256,
                "owner_reference": "owner:proof:case-001",
                "evidence_reference": "evidence:proof:case-001",
            },
        )

    def test_issue_proof_manifest_exact_replay_is_bounded_and_private(self):
        fake = FakeOpener(
            response=response(issued_proof_manifest_document(created=False), 200)
        )
        result = operate.run(
            [
                "issue-proof-manifest",
                "--pilot-id", PILOT_ID,
                "--expected-snapshot-sha256", SNAPSHOT_SHA256,
                "--owner-reference", "owner:proof:case-001",
                "--evidence-reference", "evidence:proof:case-001",
                "--confirm-owner-authorized",
            ],
            environ=ENVIRONMENT,
            opener=fake,
        )
        self.assertFalse(result["created"])
        self.assertTrue(result["idempotent_replay"])
        self.assertEqual(result["http_status"], 200)
        self.assertNotIn("signed_manifest", result)
        self.assertNotIn("signature", result)

    def test_issue_proof_manifest_rejects_drift_and_nonaggregate_signed_fields(self):
        drifted = issued_proof_manifest_document()
        drifted["manifest"]["proof_snapshot_sha256"] = "c" * 64
        with self.assertRaisesRegex(operate.OperatorError, "invalid_response"):
            operate.run(
                [
                    "issue-proof-manifest", "--pilot-id", PILOT_ID,
                    "--expected-snapshot-sha256", SNAPSHOT_SHA256,
                    "--owner-reference", "owner:proof:case-001",
                    "--evidence-reference", "evidence:proof:case-001",
                    "--confirm-owner-authorized",
                ],
                environ=ENVIRONMENT,
                opener=FakeOpener(response=response(drifted, 201)),
            )

        private = issued_proof_manifest_document()
        signed = json.loads(private["manifest"]["signed_manifest"])
        signed["provider_company_ids"] = [COMPANY_ID]
        private["manifest"]["signed_manifest"] = json.dumps(signed, separators=(",", ":"))
        private["manifest"]["payload_sha256"] = hashlib.sha256(
            private["manifest"]["signed_manifest"].encode("utf-8")
        ).hexdigest()
        with self.assertRaisesRegex(operate.OperatorError, "invalid_response"):
            operate.run(
                [
                    "issue-proof-manifest", "--pilot-id", PILOT_ID,
                    "--expected-snapshot-sha256", SNAPSHOT_SHA256,
                    "--owner-reference", "owner:proof:case-001",
                    "--evidence-reference", "evidence:proof:case-001",
                    "--confirm-owner-authorized",
                ],
                environ=ENVIRONMENT,
                opener=FakeOpener(response=response(private, 201)),
            )

        leaking_amount = issued_proof_manifest_document()
        leaking_signed = json.loads(leaking_amount["manifest"]["signed_manifest"])
        leaking_signed["verified_terms_net_receivable"] = [
            {"currency": "usd", "amount_minor": 500}
        ]
        leaking_amount["manifest"]["signed_manifest"] = json.dumps(
            leaking_signed, separators=(",", ":")
        )
        leaking_amount["manifest"]["payload_sha256"] = hashlib.sha256(
            leaking_amount["manifest"]["signed_manifest"].encode("utf-8")
        ).hexdigest()
        with self.assertRaisesRegex(operate.OperatorError, "invalid_response"):
            operate.run(
                [
                    "issue-proof-manifest", "--pilot-id", PILOT_ID,
                    "--expected-snapshot-sha256", SNAPSHOT_SHA256,
                    "--owner-reference", "owner:proof:case-001",
                    "--evidence-reference", "evidence:proof:case-001",
                    "--confirm-owner-authorized",
                ],
                environ=ENVIRONMENT,
                opener=FakeOpener(response=response(leaking_amount, 201)),
            )

    def test_issue_proof_manifest_validates_exact_inputs_before_network(self):
        cases = (
            (
                PILOT_ID.upper(), SNAPSHOT_SHA256, "owner:proof:case-001",
                "evidence:proof:case-001", "invalid_pilot_id",
            ),
            (
                PILOT_ID, SNAPSHOT_SHA256.upper(), "owner:proof:case-001",
                "evidence:proof:case-001", "invalid_expected_snapshot_sha256",
            ),
            (
                PILOT_ID, SNAPSHOT_SHA256, "short", "evidence:proof:case-001",
                "invalid_owner_reference",
            ),
            (
                PILOT_ID, SNAPSHOT_SHA256, "owner:proof:case-001", "short",
                "invalid_evidence_reference",
            ),
        )
        fake = FakeOpener(response=AssertionError("network must not be called"))
        for pilot_id, digest, owner_ref, evidence_ref, error in cases:
            with self.subTest(error=error):
                with self.assertRaisesRegex(operate.OperatorError, error):
                    operate.run(
                        [
                            "issue-proof-manifest", "--pilot-id", pilot_id,
                            "--expected-snapshot-sha256", digest,
                            "--owner-reference", owner_ref,
                            "--evidence-reference", evidence_ref,
                            "--confirm-owner-authorized",
                        ],
                        environ=ENVIRONMENT,
                        opener=fake,
                    )
        self.assertEqual(fake.calls, [])

    def test_verify_terms_sends_exact_terms_only_shape(self):
        document = {
            "commitment": {"id": COMMITMENT_ID, "event_type": "terms_acceptance"},
            "created": True,
            "idempotent_replay": False,
            "commercial_evidence_recorded": True,
            "pilot_threshold_evaluated": False,
            "evidence_scope": "bounded",
        }
        fake = FakeOpener(response=response(document, 201))
        result = operate.run(
            [
                "verify-terms",
                "--offer-id", OFFER_ID,
                "--provider-acceptance-event-id", ACCEPTANCE_ID,
                "--source-system", "owner-ledger",
                "--source-event-id", "owner-source-event-001",
                "--source-effective-at", "2026-08-02T10:00:00Z",
                "--operator-reference", "owner-case-001",
                "--owner-evidence-reference", "terms-case-001",
                "--confirm-owner-authorized",
            ],
            environ=ENVIRONMENT,
            opener=fake,
        )
        self.assertEqual(result["action"], "verify_terms")
        self.assertEqual(result["commitment_event_id"], COMMITMENT_ID)
        self.assertEqual(result["commitment_event_type"], "terms_acceptance")
        self.assertEqual(result["related_commitment_event_id"], "")
        self.assertTrue(result["commercial_evidence_recorded"])
        request = fake.calls[0][0]
        self.assertEqual(request.full_url, operate.BASE_URL + operate.COMMERCIAL_ACTION_PATH)
        payload = json.loads(request.data)
        self.assertEqual(
            set(payload),
            {
                "action", "offer_id", "provider_acceptance_event_id", "source_system",
                "source_event_id", "source_effective_at", "operator_reference", "owner_evidence_reference",
            },
        )
        self.assertEqual(payload["action"], "verify_terms")
        self.assertNotIn("amount_cents", payload)
        self.assertNotIn("currency", payload)

    def test_queue_and_verify_terms_preserve_exact_renewal_commitment_chain(self):
        queue = queue_document(
            {
                "acceptance_event_type": "terms_renewal",
                "related_acceptance_event_id": "623e4567-e89b-42d3-a456-426614174000",
                "related_commitment_event_id": RELATED_COMMITMENT_ID,
            }
        )
        projected = operate.run(["queue"], environ=ENVIRONMENT, opener=FakeOpener(response=response(queue)))
        self.assertEqual(projected["items"][0]["related_commitment_event_id"], RELATED_COMMITMENT_ID)

        document = {
            "commitment": {
                "id": COMMITMENT_ID,
                "event_type": "terms_renewal",
                "related_event_id": RELATED_COMMITMENT_ID,
            },
            "created": True,
            "idempotent_replay": False,
            "commercial_evidence_recorded": True,
            "pilot_threshold_evaluated": False,
            "evidence_scope": "bounded",
        }
        fake = FakeOpener(response=response(document, 201))
        result = operate.run(
            [
                "verify-terms",
                "--offer-id", OFFER_ID,
                "--provider-acceptance-event-id", ACCEPTANCE_ID,
                "--related-commitment-event-id", RELATED_COMMITMENT_ID,
                "--source-system", "owner-ledger",
                "--source-event-id", "owner-renewal-event-001",
                "--source-effective-at", "2026-08-02T10:00:00Z",
                "--operator-reference", "owner-renewal-case-001",
                "--owner-evidence-reference", "renewal-case-001",
                "--confirm-owner-authorized",
            ],
            environ=ENVIRONMENT,
            opener=fake,
        )
        self.assertEqual(result["commitment_event_type"], "terms_renewal")
        self.assertEqual(result["related_commitment_event_id"], RELATED_COMMITMENT_ID)
        payload = json.loads(fake.calls[0][0].data)
        self.assertEqual(payload["related_commitment_event_id"], RELATED_COMMITMENT_ID)

    def test_activate_and_pause_return_nonproof_receipts(self):
        for command, status_field in (("activate", "active"), ("pause", "paused")):
            with self.subTest(command=command):
                document: dict[str, object] = {"offer": {"id": OFFER_ID, "status": status_field}}
                if command == "activate":
                    document["commercial_proof_created"] = False
                else:
                    document.update({"paused": True, "evidence_reference": "evidence-case-001"})
                fake = FakeOpener(response=response(document))
                result = operate.run(
                    [
                        command,
                        "--offer-id", OFFER_ID,
                        "--operator-reference", "owner-case-001",
                        "--evidence-reference", "evidence-case-001",
                        "--confirm-owner-authorized",
                    ],
                    environ=ENVIRONMENT,
                    opener=fake,
                )
                self.assertEqual(result["status"], status_field)
                self.assertFalse(result["commercial_proof_created"])


class PilotEpochCommandContractTest(unittest.TestCase):
    def test_authorize_pilot_sends_exact_shape_and_projects_bounded_receipt(self):
        fake = FakeOpener(response=response(pilot_mutation_document("authorize", pilot_epoch()), 201))
        result = operate.run(
            [
                "authorize-pilot",
                "--topic", "developer-tools",
                "--cohort-limit", "3",
                "--provider-ticket-cap", "5",
                "--total-ticket-cap", "5",
                "--owner-reference", "owner:pilot-case-001",
                "--evidence-reference", "evidence:pilot-case-001",
                "--confirm-owner-authorized",
            ],
            environ=ENVIRONMENT,
            opener=fake,
        )
        self.assertEqual(
            set(result),
            {
                "ok", "action", "http_status", "pilot_id", "demand_topic", "cohort_limit",
                "provider_ticket_cap", "total_ticket_cap", "status", "created_at", "activated_at",
                "closed_at", "commercial_proof_created",
            },
        )
        self.assertEqual(result["action"], "authorize_pilot")
        self.assertEqual(result["pilot_id"], PILOT_ID)
        self.assertEqual(result["status"], "draft")
        self.assertFalse(result["commercial_proof_created"])
        self.assertNotIn("owner:pilot-case-001", json.dumps(result))
        request = fake.calls[0][0]
        self.assertEqual(request.full_url, operate.BASE_URL + operate.PILOT_ACTION_PATH)
        self.assertEqual(request.get_method(), "POST")
        self.assertEqual(
            json.loads(request.data),
            {
                "action": "authorize",
                "demand_topic": "developer-tools",
                "cohort_limit": 3,
                "provider_ticket_cap": 5,
                "total_ticket_cap": 5,
                "owner_reference": "owner:pilot-case-001",
                "evidence_reference": "evidence:pilot-case-001",
            },
        )

    def test_authorize_pilot_rejects_unsafe_bounds_before_network(self):
        cases = (
            ("other", "3", "5", "5", "invalid_pilot_topic"),
            ("developer-tools", "2", "5", "5", "invalid_cohort_limit"),
            ("developer-tools", "21", "5", "21", "invalid_cohort_limit"),
            ("developer-tools", "3", "0", "5", "invalid_provider_ticket_cap"),
            ("developer-tools", "3", "101", "101", "invalid_provider_ticket_cap"),
            ("developer-tools", "3", "5", "4", "invalid_total_ticket_cap"),
            ("developer-tools", "3", "5", "2001", "invalid_total_ticket_cap"),
        )
        fake = FakeOpener(response=AssertionError("network must not be called"))
        for topic, cohort, provider_cap, total_cap, error in cases:
            with self.subTest(error=error, topic=topic, cohort=cohort):
                with self.assertRaisesRegex(operate.OperatorError, error):
                    operate.run(
                        [
                            "authorize-pilot",
                            "--topic", topic,
                            "--cohort-limit", cohort,
                            "--provider-ticket-cap", provider_cap,
                            "--total-ticket-cap", total_cap,
                            "--owner-reference", "owner:pilot-case-001",
                            "--evidence-reference", "evidence:pilot-case-001",
                            "--confirm-owner-authorized",
                        ],
                        environ=ENVIRONMENT,
                        opener=fake,
                    )
        self.assertEqual(fake.calls, [])

    def test_enroll_pilot_sends_exact_shape_and_projects_opaque_ids(self):
        fake = FakeOpener(response=response(pilot_enrollment_document()))
        result = operate.run(
            [
                "enroll-pilot",
                "--pilot-id", PILOT_ID,
                "--claim-id", CLAIM_ID,
                "--owner-reference", "owner:pilot-case-001",
                "--evidence-reference", "evidence:pilot-case-001",
                "--confirm-owner-authorized",
            ],
            environ=ENVIRONMENT,
            opener=fake,
        )
        self.assertEqual(
            set(result),
            {
                "ok", "action", "http_status", "pilot_id", "provider_claim_id", "enrollment_id",
                "provider_pilot_company_id", "enrolled_at", "commercial_proof_created",
            },
        )
        self.assertEqual(result["action"], "enroll_pilot")
        self.assertEqual(result["enrollment_id"], ENROLLMENT_ID)
        self.assertNotIn("evidence:pilot-case-001", json.dumps(result))
        request = fake.calls[0][0]
        self.assertEqual(request.full_url, operate.BASE_URL + operate.PILOT_ACTION_PATH)
        self.assertEqual(
            json.loads(request.data),
            {
                "action": "enroll",
                "provider_pilot_epoch_id": PILOT_ID,
                "provider_claim_id": CLAIM_ID,
                "owner_reference": "owner:pilot-case-001",
                "evidence_reference": "evidence:pilot-case-001",
            },
        )

    def test_activate_and_close_pilot_use_exact_action_shapes(self):
        for command, action, pilot_status in (
            ("activate-pilot", "activate", "active"),
            ("close-pilot", "close", "closed"),
        ):
            with self.subTest(command=command):
                fake = FakeOpener(
                    response=response(pilot_mutation_document(action, pilot_epoch(pilot_status)))
                )
                result = operate.run(
                    [
                        command,
                        "--pilot-id", PILOT_ID,
                        "--owner-reference", "owner:pilot-case-001",
                        "--evidence-reference", "evidence:pilot-case-001",
                        "--confirm-owner-authorized",
                    ],
                    environ=ENVIRONMENT,
                    opener=fake,
                )
                self.assertEqual(result["action"], action + "_pilot")
                self.assertEqual(result["status"], pilot_status)
                self.assertFalse(result["commercial_proof_created"])
                self.assertEqual(
                    json.loads(fake.calls[0][0].data),
                    {
                        "action": action,
                        "provider_pilot_epoch_id": PILOT_ID,
                        "owner_reference": "owner:pilot-case-001",
                        "evidence_reference": "evidence:pilot-case-001",
                    },
                )

    def test_status_pilot_is_read_only_and_projects_exact_aggregate_schema(self):
        fake = FakeOpener(response=response(pilot_status_document()))
        result = operate.run(
            ["status-pilot", "--pilot-id", PILOT_ID],
            environ=ENVIRONMENT,
            opener=fake,
        )
        self.assertEqual(result["action"], "status_pilot")
        self.assertEqual(result["enrolled_provider_count"], 3)
        self.assertEqual(result["fresh_enrolled_provider_count"], 3)
        self.assertEqual(result["remaining_ticket_capacity"], 3)
        self.assertTrue(result["cohort_ready"])
        self.assertFalse(result["commercial_proof_created"])
        self.assertNotIn("owner:pilot-case-001", json.dumps(result))
        request = fake.calls[0][0]
        self.assertEqual(
            request.full_url,
            operate.BASE_URL + operate.PILOT_EPOCH_PATH + "?pilot_id=" + PILOT_ID,
        )
        self.assertEqual(request.get_method(), "GET")
        self.assertIsNone(request.data)

    def test_pilot_routes_reject_extra_or_inconsistent_response_fields(self):
        extra_epoch = pilot_epoch(extra={"provider_domain": "private.example"})
        with self.assertRaisesRegex(operate.OperatorError, "invalid_response"):
            operate.run(
                [
                    "authorize-pilot",
                    "--topic", "developer-tools",
                    "--cohort-limit", "3",
                    "--provider-ticket-cap", "5",
                    "--total-ticket-cap", "5",
                    "--owner-reference", "owner:pilot-case-001",
                    "--evidence-reference", "evidence:pilot-case-001",
                    "--confirm-owner-authorized",
                ],
                environ=ENVIRONMENT,
                opener=FakeOpener(response=response(pilot_mutation_document("authorize", extra_epoch), 201)),
            )
        inconsistent = pilot_status_document({"remaining_ticket_capacity": 9})
        with self.assertRaisesRegex(operate.OperatorError, "invalid_response"):
            operate.run(
                ["status-pilot", "--pilot-id", PILOT_ID],
                environ=ENVIRONMENT,
                opener=FakeOpener(response=response(inconsistent)),
            )


class SecretAndFailureTest(unittest.TestCase):
    def test_admin_key_is_env_only_and_validated(self):
        for environment in ({}, {operate.ADMIN_KEY_ENV: "bad\nkey"}, {operate.ADMIN_KEY_ENV: "short"}):
            with self.assertRaisesRegex(operate.OperatorError, "admin_key_unavailable"):
                operate.admin_key(environment)
        help_text = operate.build_parser().format_help()
        self.assertNotIn("admin-key", help_text)
        self.assertNotIn("base-url", help_text)

    def test_main_sanitizes_failures_and_disables_core_dumps_first(self):
        private = "private-admin-key-value"
        stderr = io.StringIO()
        with (
            mock.patch.object(operate.resource, "setrlimit") as set_limit,
            mock.patch.object(operate, "run", side_effect=RuntimeError(private)),
            contextlib.redirect_stderr(stderr),
        ):
            status = operate.main(["queue"])
        self.assertEqual(status, 1)
        set_limit.assert_called_once_with(operate.resource.RLIMIT_CORE, (0, 0))
        self.assertEqual(json.loads(stderr.getvalue()), {"error": "internal_error", "ok": False})
        self.assertNotIn(private, stderr.getvalue())

    def test_invalid_argument_does_not_reflect_value(self):
        private = "private-argument-value"
        stderr = io.StringIO()
        with contextlib.redirect_stderr(stderr):
            status = operate.main([private])
        self.assertEqual(status, 1)
        self.assertEqual(json.loads(stderr.getvalue()), {"error": "invalid_arguments", "ok": False})
        self.assertNotIn(private, stderr.getvalue())


if __name__ == "__main__":
    unittest.main()
