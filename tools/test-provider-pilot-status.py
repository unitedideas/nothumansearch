#!/usr/bin/env python3
"""Offline security and truth-contract tests for provider-pilot-status.py."""

import contextlib
import copy
import hashlib
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


MODULE_PATH = pathlib.Path(__file__).with_name("provider-pilot-status.py")
SPEC = importlib.util.spec_from_file_location("provider_pilot_status", MODULE_PATH)
status_tool = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = status_tool
SPEC.loader.exec_module(status_tool)

TEST_ADMIN_KEY = "test-admin-key-fixture"
TEST_ENVIRONMENT = {status_tool.ADMIN_KEY_ENV: TEST_ADMIN_KEY}
TEST_PILOT_ID = "4b69ca8e-d61d-47e2-91dd-fecd9f711234"
TEST_MANIFEST_ID = "5b69ca8e-d61d-47e2-91dd-fecd9f711234"
TEST_PROOF_SNAPSHOT_SHA256 = "b" * 64
TEST_REVIEW_EVIDENCE_SHA256 = "c" * 64
TEST_KEY_ID = "nhs-provider-signing-v1"


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


def stage1_document():
    return {
        "stage1_demand": {
            "days": 30,
            "retention_days": 30,
            "as_of": "2026-08-02T12:00:00Z",
            "stage1_started_at": "2026-07-15T12:00:00Z",
            "stage1_epoch_enforced": True,
            "synthetic_excluded": True,
            "eligible_surfaces": ["mcp", "rest"],
            "counts_are_receipts_not_unique_agents": True,
            "commercial_proof": False,
            "meaningful_search_receipts": 120,
            "result_selections": 31,
            "search_receipts_with_selection": 24,
            "action_interest_receipts": 25,
            "search_receipts_with_action_interest": 22,
            "distinct_interest_domains": 8,
            "bucket_receipt_threshold": 20,
            "topic_buckets_may_overlap": True,
            "demand_topics": [{"value": "developer-tools", "receipt_count": 42}],
            "pilot_candidate_topics": [{"value": "developer-tools", "receipt_count": 42}],
            "pilot_candidate_topic_available": True,
            "action_types": [{"value": "quote", "receipt_count": 20}],
            "observation_window_days": 14,
            "observation_span_seconds": 16 * 24 * 60 * 60,
            "observation_span_days": 16,
            "observation_window_met": True,
            "stage1_ready": True,
            "targets": dict(status_tool.STAGE1_TARGETS),
            "targets_met": {key: True for key in status_tool.STAGE1_TARGETS},
            "server_private_row": "must-not-emit",
        },
        "action_interest_attempt_funnel": {
            "days": 30,
            "as_of": "2026-08-02T12:00:01Z",
            "counts_are_attempts_not_unique_agents": True,
            "contains_request_coordinates": False,
            "commercial_proof": False,
            "total_attempts": 27,
            "outcomes": [
                {"surface": "mcp", "outcome": "created", "attempt_count": 22},
                {"surface": "mcp", "outcome": "invalid_request", "attempt_count": 5},
            ],
            "evidence_scope": "aggregate attempts only",
            "server_private_row": "must-not-emit",
        },
        "evidence_scope": "aggregate receipts only",
        "server_private": "must-not-emit",
    }


def proof_document():
    return {
        "proof": {
            "provider_pilot_epoch_id": TEST_PILOT_ID,
            "provider_pilot_demand_topic": "developer-tools",
            "provider_pilot_status": "closed",
            "outcome_receipt_integrity_valid": True,
            "verified_outcome_receipts": 8,
            "rejected_outcome_receipts": 0,
            "verified_outcome_ledger_entries": 8,
            "rejected_outcome_ledger_entries": 0,
            "verified_provider_companies": 3,
            "verified_provider_offer_returns": 7,
            "verified_observed_handoffs": 7,
            "verified_provider_accepted_handoffs": 5,
            "verified_provider_confirmed_activations": 2,
            "verified_provider_renewals": 1,
            "verified_provider_confirmed_conversions": 1,
            "verified_accepted_latency_samples": 5,
            "verified_activated_latency_samples": 2,
            "verified_converted_latency_samples": 1,
            "verified_accepted_median_handoff_to_outcome_seconds": 3600,
            "verified_activated_median_handoff_to_outcome_seconds": 86400,
            "verified_converted_median_handoff_to_outcome_seconds": 604800,
            "settlement_receipt_integrity_valid": True,
            "processor_net_receipt_integrity_valid": True,
            "verified_provider_paid_settlements": 1,
            "verified_provider_available_settlements": 1,
            "rejected_provider_settlement_receipts": 0,
            "rejected_provider_processor_net_receipts": 0,
            "verified_paid_latency_samples": 1,
            "verified_paid_median_handoff_to_settlement_seconds": 691200,
            "verified_terms_paid_by_currency": {"usd": 500},
            "verified_processor_fees_by_currency": {"usd": 44},
            "verified_processor_net_by_currency": {"usd": 456},
            "verified_mechanisms": {
                "accepted": {
                    "charged_provider_companies": 0, "offer_returns": 3, "observed_handoffs": 3, "accepted": 2, "activated": 0,
                    "converted": 0, "reversed": 0, "paid_settlements": 0, "paid_cents": 0,
                    "available_settlements": 0, "processor_fee_cents": 0, "processor_net_cents": 0,
                    "paid_median_handoff_to_settlement_seconds": 0,
                },
                "activated": {
                    "charged_provider_companies": 1, "offer_returns": 2, "observed_handoffs": 2, "accepted": 2, "activated": 1,
                    "converted": 0, "reversed": 0, "paid_settlements": 1, "paid_cents": 500,
                    "available_settlements": 1, "processor_fee_cents": 44, "processor_net_cents": 456,
                    "paid_median_handoff_to_settlement_seconds": 691200,
                },
                "converted": {
                    "charged_provider_companies": 0, "offer_returns": 2, "observed_handoffs": 2, "accepted": 1, "activated": 1,
                    "converted": 1, "reversed": 0, "paid_settlements": 0, "paid_cents": 0,
                    "available_settlements": 0, "processor_fee_cents": 0, "processor_net_cents": 0,
                    "paid_median_handoff_to_settlement_seconds": 0,
                },
            },
            "verified_prepaid_settled_by_currency": {"usd": 15000},
            "verified_prepaid_net_debited_by_currency": {"usd": 2500},
            "verified_terms_net_receivable_by_currency": {"usd": 500},
            "operator_recorded_provider_budgets": 8,
            "provider_reported_accepted_handoffs": 9,
            "provider_reported_activations": 4,
            "renewed_provider_budgets": 2,
            "provider_reported_conversions": 1,
            "prepaid_net_debited_by_currency": {"usd": 3000},
            "terms_net_receivable_by_currency": {"usd": 700},
            "operator_recorded_collected_by_currency": {"usd": 20000},
            "pilot_thresholds_met": True,
            "company_identity": "must-not-emit",
        },
        "targets": dict(status_tool.PROOF_TARGETS),
        "evidence_scope": "verified evidence only",
        "operational_progress_scope": "diagnostic observations only",
        "organic_rank_sold": False,
        "raw_queries_sold": False,
        "raw_prompts_sold": False,
        "agent_identities_sold": False,
        "principal_identities_sold": False,
        "server_private": "must-not-emit",
    }


def encoded(document):
    return json.dumps(document).encode("utf-8")


def manifest_candidate():
    return {
        "manifest_contract_version": status_tool.PROOF_MANIFEST_CONTRACT_VERSION,
        "signature_verification_scope": status_tool.PROOF_MANIFEST_VERIFICATION_SCOPE,
        "provider_pilot_epoch_id": TEST_PILOT_ID,
        "provider_pilot_contract_version": "nhs-provider-pilot-v1",
        "review_contract_version": "nhs-provider-pilot-review-v1",
        "review_evidence_contract_version": status_tool.PROOF_MANIFEST_REVIEW_EVIDENCE_VERSION,
        "market_policy_contract_version": status_tool.PROOF_MANIFEST_MARKET_POLICY_VERSION,
        "proof_snapshot_sha256": TEST_PROOF_SNAPSHOT_SHA256,
        "review_evidence_sha256": TEST_REVIEW_EVIDENCE_SHA256,
        "pilot_demand_topic": "developer-tools",
        "pilot_status": "closed",
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
        "issuable": True,
        "issuance_blockers": [],
        "evidence_scope": status_tool.PROOF_MANIFEST_SCOPE,
    }


def manifest_candidate_document():
    return {
        "manifest_candidate": manifest_candidate(),
        "issued": False,
        "commercial_proof_created": False,
        "publicly_released": False,
        "independently_verifiable": False,
        "evidence_scope": status_tool.PROOF_MANIFEST_EVIDENCE_SCOPE,
    }


def signed_manifest_payload():
    candidate = manifest_candidate()
    return {
        "v": 1,
        "kid": TEST_KEY_ID,
        "signature_verification_scope": candidate["signature_verification_scope"],
        "manifest_contract_version": candidate["manifest_contract_version"],
        "manifest_id": TEST_MANIFEST_ID,
        "provider_pilot_epoch_id": candidate["provider_pilot_epoch_id"],
        "provider_pilot_contract_version": candidate["provider_pilot_contract_version"],
        "review_contract_version": candidate["review_contract_version"],
        "review_evidence_contract_version": candidate["review_evidence_contract_version"],
        "market_policy_contract_version": candidate["market_policy_contract_version"],
        "proof_snapshot_sha256": candidate["proof_snapshot_sha256"],
        "review_evidence_sha256": candidate["review_evidence_sha256"],
        "pilot_demand_topic": candidate["pilot_demand_topic"],
        "pilot_status": candidate["pilot_status"],
        "issued_at": 1785672000,
        "outcome_receipt_integrity_valid": candidate["outcome_receipt_integrity_valid"],
        "review_integrity_valid": candidate["review_integrity_valid"],
        "verified_outcome_receipts": candidate["verified_outcome_receipts"],
        "rejected_outcome_receipts": candidate["rejected_outcome_receipts"],
        "verified_outcome_ledger_entries": candidate["verified_outcome_ledger_entries"],
        "rejected_outcome_ledger_entries": candidate["rejected_outcome_ledger_entries"],
        "verified_provider_companies": candidate["verified_provider_companies"],
        "verified_provider_accepted_handoffs": candidate["verified_provider_accepted_handoffs"],
        "verified_provider_confirmed_activations": candidate[
            "verified_provider_confirmed_activations"
        ],
        "verified_provider_renewals": candidate["verified_provider_renewals"],
        "verified_provider_confirmed_conversions": candidate[
            "verified_provider_confirmed_conversions"
        ],
        "review_coverage": candidate["review_coverage"],
        "monetary_amounts_withheld_for_privacy": candidate[
            "monetary_amounts_withheld_for_privacy"
        ],
        "verified_prepaid_settled": candidate["verified_prepaid_settled"],
        "verified_prepaid_net_debited": candidate["verified_prepaid_net_debited"],
        "verified_terms_net_receivable": candidate["verified_terms_net_receivable"],
        "pilot_thresholds_met": candidate["pilot_thresholds_met"],
        "organic_rank_sold": candidate["organic_rank_sold"],
        "raw_queries_sold": candidate["raw_queries_sold"],
        "agent_identities_sold": candidate["agent_identities_sold"],
        "evidence_scope": candidate["evidence_scope"],
    }


def issued_manifest_document():
    signed = json.dumps(signed_manifest_payload(), separators=(",", ":"))
    return {
        "manifest": {
            "id": TEST_MANIFEST_ID,
            "provider_pilot_epoch_id": TEST_PILOT_ID,
            "manifest_contract_version": status_tool.PROOF_MANIFEST_CONTRACT_VERSION,
            "proof_snapshot_sha256": TEST_PROOF_SNAPSHOT_SHA256,
            "review_evidence_sha256": TEST_REVIEW_EVIDENCE_SHA256,
            "key_id": TEST_KEY_ID,
            "signed_manifest": signed,
            "signature": "A" * 43,
            "payload_sha256": hashlib.sha256(signed.encode("utf-8")).hexdigest(),
            "issued_at": "2026-08-02T12:00:00Z",
        },
        "issued": True,
        "commercial_proof_created": True,
        "publicly_released": False,
        "independently_verifiable": False,
        "evidence_scope": status_tool.PROOF_MANIFEST_EVIDENCE_SCOPE,
    }


class ReadContractTest(unittest.TestCase):
    def test_all_scope_uses_three_fixed_gets_and_projects_only_safe_aggregates(self):
        stage_response = FakeResponse(200, encoded(stage1_document()))
        proof_response = FakeResponse(200, encoded(proof_document()))
        manifest_response = FakeResponse(200, encoded(manifest_candidate_document()))
        opener = FakeOpener(stage_response, proof_response, manifest_response)
        receipt = status_tool.run(
            ["--pilot-id", TEST_PILOT_ID], environ=TEST_ENVIRONMENT, opener=opener
        )

        self.assertEqual(len(opener.calls), 3)
        first, second, third = opener.calls
        self.assertEqual(first[1], 10)
        self.assertEqual(second[1], 10)
        self.assertEqual(third[1], 10)
        first_url = urllib.parse.urlsplit(first[0].full_url)
        self.assertEqual(first_url.scheme, "https")
        self.assertEqual(first_url.netloc, "nothumansearch.ai")
        self.assertEqual(first_url.path, "/api/v1/admin/demand-stage1")
        self.assertEqual(urllib.parse.parse_qs(first_url.query), {"days": ["30"]})
        second_url = urllib.parse.urlsplit(second[0].full_url)
        self.assertEqual(second_url.path, "/api/v1/admin/provider-proof")
        self.assertEqual(urllib.parse.parse_qs(second_url.query), {"pilot_id": [TEST_PILOT_ID]})
        third_url = urllib.parse.urlsplit(third[0].full_url)
        self.assertEqual(third_url.path, "/api/v1/admin/provider-proof-manifest")
        self.assertEqual(urllib.parse.parse_qs(third_url.query), {"pilot_id": [TEST_PILOT_ID]})
        for request, _ in opener.calls:
            self.assertEqual(request.get_method(), "GET")
            self.assertTrue(
                hmac.compare_digest(request.get_header("Authorization"), "Bearer " + TEST_ADMIN_KEY)
            )
            self.assertEqual(request.get_header("Accept"), "application/json")

        self.assertTrue(receipt["stage1"]["stage1_ready"])
        self.assertTrue(receipt["commercial_proof"]["pilot_thresholds_met"])
        self.assertTrue(receipt["commercial_proof_manifest"]["manifest_candidate"]["issuable"])
        self.assertFalse(receipt["commercial_proof_manifest"]["issued"])
        serialized = json.dumps(receipt, sort_keys=True)
        self.assertNotIn("must-not-emit", serialized)
        self.assertNotIn(TEST_ADMIN_KEY, serialized)
        self.assertNotIn("company_identity", serialized)
        self.assertEqual(stage_response.read_sizes, [status_tool.MAX_RESPONSE_BYTES + 1])
        self.assertEqual(proof_response.read_sizes, [status_tool.MAX_RESPONSE_BYTES + 1])
        self.assertEqual(manifest_response.read_sizes, [status_tool.MAX_RESPONSE_BYTES + 1])

    def test_each_narrow_scope_makes_exactly_one_request(self):
        stage = FakeOpener(FakeResponse(200, encoded(stage1_document())))
        result = status_tool.run(["--scope", "stage1"], environ=TEST_ENVIRONMENT, opener=stage)
        self.assertEqual(set(result), {"ok", "scope", "stage1"})
        self.assertEqual(len(stage.calls), 1)

        proof = FakeOpener(FakeResponse(200, encoded(proof_document())))
        result = status_tool.run(
            ["--scope", "proof", "--pilot-id", TEST_PILOT_ID],
            environ=TEST_ENVIRONMENT,
            opener=proof,
        )
        self.assertEqual(set(result), {"ok", "scope", "commercial_proof"})
        self.assertEqual(len(proof.calls), 1)

        manifest = FakeOpener(FakeResponse(200, encoded(manifest_candidate_document())))
        result = status_tool.run(
            ["--scope", "proof-manifest", "--pilot-id", TEST_PILOT_ID],
            environ=TEST_ENVIRONMENT,
            opener=manifest,
        )
        self.assertEqual(set(result), {"ok", "scope", "commercial_proof_manifest"})
        self.assertEqual(len(manifest.calls), 1)

    def test_proof_rejects_mechanism_totals_that_do_not_reconcile(self):
        document = proof_document()
        document["proof"]["verified_mechanisms"]["accepted"]["paid_cents"] += 1
        with self.assertRaises(status_tool.StatusError) as caught:
            status_tool.project_proof(document, TEST_PILOT_ID)
        self.assertEqual(caught.exception.code, "invalid_response")

    def test_proof_rejects_offer_return_totals_that_do_not_reconcile(self):
        document = proof_document()
        document["proof"]["verified_mechanisms"]["accepted"]["offer_returns"] += 1
        with self.assertRaises(status_tool.StatusError) as caught:
            status_tool.project_proof(document, TEST_PILOT_ID)
        self.assertEqual(caught.exception.code, "invalid_response")

    def test_proof_rejects_missing_mechanism_arm(self):
        document = proof_document()
        del document["proof"]["verified_mechanisms"]["converted"]
        with self.assertRaises(status_tool.StatusError) as caught:
            status_tool.project_proof(document, TEST_PILOT_ID)
        self.assertEqual(caught.exception.code, "invalid_response")

    def test_proof_rejects_mechanism_provider_count_above_cohort(self):
        document = proof_document()
        document["proof"]["verified_mechanisms"]["activated"][
            "charged_provider_companies"
        ] = 4
        with self.assertRaises(status_tool.StatusError) as caught:
            status_tool.project_proof(document, TEST_PILOT_ID)
        self.assertEqual(caught.exception.code, "invalid_response")

    def test_manifest_candidate_projection_is_exact_private_and_fail_closed(self):
        projected = status_tool.project_proof_manifest(
            manifest_candidate_document(), TEST_PILOT_ID
        )
        self.assertFalse(projected["issued"])
        self.assertFalse(projected["commercial_proof_created"])
        self.assertFalse(projected["publicly_released"])
        self.assertFalse(projected["independently_verifiable"])
        self.assertTrue(projected["manifest_candidate"]["issuable"])
        self.assertEqual(
            projected["manifest_candidate"]["proof_snapshot_sha256"],
            TEST_PROOF_SNAPSHOT_SHA256,
        )
        self.assertEqual(
            projected["manifest_candidate"]["review_evidence_sha256"],
            TEST_REVIEW_EVIDENCE_SHA256,
        )
        self.assertTrue(
            projected["manifest_candidate"]["monetary_amounts_withheld_for_privacy"]
        )

        private = manifest_candidate_document()
        private["manifest_candidate"]["provider_company_ids"] = [TEST_MANIFEST_ID]
        with self.assertRaisesRegex(status_tool.StatusError, "invalid_response"):
            status_tool.project_proof_manifest(private, TEST_PILOT_ID)

        leaking_amount = manifest_candidate_document()
        leaking_amount["manifest_candidate"]["verified_terms_net_receivable"] = [
            {"currency": "usd", "amount_minor": 500}
        ]
        with self.assertRaisesRegex(status_tool.StatusError, "invalid_response"):
            status_tool.project_proof_manifest(leaking_amount, TEST_PILOT_ID)

        overstated_verification = manifest_candidate_document()
        overstated_verification["independently_verifiable"] = True
        with self.assertRaisesRegex(status_tool.StatusError, "invalid_response"):
            status_tool.project_proof_manifest(overstated_verification, TEST_PILOT_ID)

        incomplete = manifest_candidate_document()
        incomplete["manifest_candidate"]["review_coverage"]["callbacks"]["valid"] = 7
        incomplete["manifest_candidate"]["review_integrity_valid"] = False
        incomplete["manifest_candidate"]["issuable"] = False
        incomplete["manifest_candidate"]["issuance_blockers"] = [
            "chronological_review_incomplete"
        ]
        projected = status_tool.project_proof_manifest(incomplete, TEST_PILOT_ID)
        self.assertFalse(projected["manifest_candidate"]["issuable"])

    def test_issued_manifest_projection_validates_safe_signed_readback(self):
        projected = status_tool.project_proof_manifest(issued_manifest_document(), TEST_PILOT_ID)
        self.assertTrue(projected["issued"])
        self.assertTrue(projected["commercial_proof_created"])
        self.assertFalse(projected["publicly_released"])
        self.assertFalse(projected["independently_verifiable"])
        self.assertEqual(projected["manifest"]["id"], TEST_MANIFEST_ID)
        self.assertEqual(projected["manifest"]["proof_snapshot_sha256"], TEST_PROOF_SNAPSHOT_SHA256)
        self.assertEqual(projected["manifest"]["review_evidence_sha256"], TEST_REVIEW_EVIDENCE_SHA256)

        private = issued_manifest_document()
        signed = json.loads(private["manifest"]["signed_manifest"])
        signed["query"] = "private words"
        private["manifest"]["signed_manifest"] = json.dumps(signed, separators=(",", ":"))
        private["manifest"]["payload_sha256"] = hashlib.sha256(
            private["manifest"]["signed_manifest"].encode("utf-8")
        ).hexdigest()
        with self.assertRaisesRegex(status_tool.StatusError, "invalid_response"):
            status_tool.project_proof_manifest(private, TEST_PILOT_ID)

    def test_stage1_truth_relationships_and_thresholds_fail_closed(self):
        cases = []
        wrong_ready = stage1_document()
        wrong_ready["stage1_demand"]["stage1_ready"] = False
        cases.append(wrong_ready)
        wrong_targets = stage1_document()
        wrong_targets["stage1_demand"]["targets"]["meaningful_search_receipts"] = 99
        cases.append(wrong_targets)
        leaking_bucket = stage1_document()
        leaking_bucket["stage1_demand"]["demand_topics"][0]["receipt_count"] = 19
        cases.append(leaking_bucket)
        wrong_span = stage1_document()
        wrong_span["stage1_demand"]["observation_span_seconds"] = 13 * 24 * 60 * 60
        wrong_span["stage1_demand"]["observation_span_days"] = 13
        cases.append(wrong_span)
        no_candidate = stage1_document()
        no_candidate["stage1_demand"]["pilot_candidate_topics"] = []
        cases.append(no_candidate)
        historical_epoch = stage1_document()
        historical_epoch["stage1_demand"]["stage1_epoch_enforced"] = False
        cases.append(historical_epoch)
        impossible_selection = stage1_document()
        impossible_selection["stage1_demand"]["search_receipts_with_selection"] = 121
        cases.append(impossible_selection)
        impossible_action_bucket = stage1_document()
        impossible_action_bucket["stage1_demand"]["action_types"][0]["receipt_count"] = 23
        cases.append(impossible_action_bucket)
        web_surface = stage1_document()
        web_surface["stage1_demand"]["eligible_surfaces"] = ["web", "rest"]
        cases.append(web_surface)
        reordered_surfaces = stage1_document()
        reordered_surfaces["stage1_demand"]["eligible_surfaces"] = ["rest", "mcp"]
        cases.append(reordered_surfaces)
        missing_surfaces = stage1_document()
        del missing_surfaces["stage1_demand"]["eligible_surfaces"]
        cases.append(missing_surfaces)

        for case, document in enumerate(cases):
            with self.subTest(case=case):
                with self.assertRaisesRegex(status_tool.StatusError, "invalid_response"):
                    status_tool.project_stage1(document, 30)

    def test_action_interest_attempt_funnel_rejects_coordinates_and_false_totals(self):
        wrong_total = stage1_document()
        wrong_total["action_interest_attempt_funnel"]["total_attempts"] = 28
        with self.assertRaisesRegex(status_tool.StatusError, "invalid_response"):
            status_tool.project_stage1(wrong_total, 30)

        private_coordinate = stage1_document()
        private_coordinate["action_interest_attempt_funnel"]["outcomes"][0]["domain"] = "private.example"
        with self.assertRaisesRegex(status_tool.StatusError, "invalid_response"):
            status_tool.project_stage1(private_coordinate, 30)

        false_privacy = stage1_document()
        false_privacy["action_interest_attempt_funnel"]["contains_request_coordinates"] = True
        with self.assertRaisesRegex(status_tool.StatusError, "invalid_response"):
            status_tool.project_stage1(false_privacy, 30)

    def test_commercial_gate_uses_only_verified_counters(self):
        below = proof_document()
        below["proof"]["verified_provider_companies"] = 2
        below["proof"]["pilot_thresholds_met"] = False
        below["proof"]["operator_recorded_provider_budgets"] = 1000
        below["proof"]["provider_reported_accepted_handoffs"] = 1000
        projected = status_tool.project_proof(below, TEST_PILOT_ID)
        self.assertFalse(projected["pilot_thresholds_met"])

        false_green = copy.deepcopy(below)
        false_green["proof"]["pilot_thresholds_met"] = True
        with self.assertRaisesRegex(status_tool.StatusError, "invalid_response"):
            status_tool.project_proof(false_green, TEST_PILOT_ID)

        contaminated = proof_document()
        contaminated["proof"]["outcome_receipt_integrity_valid"] = False
        contaminated["proof"]["rejected_outcome_receipts"] = 1
        contaminated["proof"]["pilot_thresholds_met"] = False
        projected = status_tool.project_proof(contaminated, TEST_PILOT_ID)
        self.assertFalse(projected["outcome_receipt_integrity_valid"])
        self.assertFalse(projected["pilot_thresholds_met"])

        forged_green = copy.deepcopy(contaminated)
        forged_green["proof"]["pilot_thresholds_met"] = True
        with self.assertRaisesRegex(status_tool.StatusError, "invalid_response"):
            status_tool.project_proof(forged_green, TEST_PILOT_ID)

        unsigned_ledger = proof_document()
        unsigned_ledger["proof"]["outcome_receipt_integrity_valid"] = False
        unsigned_ledger["proof"]["rejected_outcome_ledger_entries"] = 1
        unsigned_ledger["proof"]["pilot_thresholds_met"] = False
        projected = status_tool.project_proof(unsigned_ledger, TEST_PILOT_ID)
        self.assertEqual(projected["rejected_outcome_ledger_entries"], 1)
        self.assertFalse(projected["pilot_thresholds_met"])

    def test_proof_rejects_impossible_funnel_and_paid_rank_claim(self):
        impossible = proof_document()
        impossible["proof"]["verified_provider_confirmed_activations"] = 6
        with self.assertRaisesRegex(status_tool.StatusError, "invalid_response"):
            status_tool.project_proof(impossible, TEST_PILOT_ID)

        impossible_observed = proof_document()
        impossible_observed["proof"]["verified_observed_handoffs"] = 4
        with self.assertRaisesRegex(status_tool.StatusError, "invalid_response"):
            status_tool.project_proof(impossible_observed, TEST_PILOT_ID)

        missing_latency = proof_document()
        missing_latency["proof"]["verified_activated_latency_samples"] = 1
        with self.assertRaisesRegex(status_tool.StatusError, "invalid_response"):
            status_tool.project_proof(missing_latency, TEST_PILOT_ID)

        contaminated_settlement = proof_document()
        contaminated_settlement["proof"]["settlement_receipt_integrity_valid"] = False
        contaminated_settlement["proof"]["rejected_provider_settlement_receipts"] = 1
        contaminated_settlement["proof"]["pilot_thresholds_met"] = False
        projected = status_tool.project_proof(contaminated_settlement, TEST_PILOT_ID)
        self.assertFalse(projected["settlement_receipt_integrity_valid"])
        self.assertFalse(projected["pilot_thresholds_met"])

        paid_rank = proof_document()
        paid_rank["organic_rank_sold"] = True
        with self.assertRaisesRegex(status_tool.StatusError, "invalid_response"):
            status_tool.project_proof(paid_rank, TEST_PILOT_ID)

        for forbidden in ("raw_prompts_sold", "principal_identities_sold"):
            leaking = proof_document()
            leaking[forbidden] = True
            with self.assertRaisesRegex(status_tool.StatusError, "invalid_response"):
                status_tool.project_proof(leaking, TEST_PILOT_ID)


class TransportAndFailureTest(unittest.TestCase):
    def test_admin_key_and_arguments_fail_before_network(self):
        for environment in ({}, {status_tool.ADMIN_KEY_ENV: "bad\nkey"}):
            with self.subTest(environment=bool(environment)):
                with self.assertRaisesRegex(status_tool.StatusError, "admin_key_unavailable"):
                    status_tool.run(["--pilot-id", TEST_PILOT_ID], environ=environment, opener=mock.Mock())
        for args in (
            ["--stage1-days", "14"],
            ["--stage1-days", "31"],
            ["--base-url", "https://evil.example"],
            ["--scope", "proof"],
            ["--scope", "proof", "--pilot-id", TEST_PILOT_ID.upper()],
            ["--scope", "proof-manifest"],
            ["--scope", "proof-manifest", "--pilot-id", TEST_PILOT_ID.upper()],
        ):
            with self.subTest(args=args):
                with self.assertRaisesRegex(status_tool.StatusError, "invalid_arguments"):
                    status_tool.run(args, environ=TEST_ENVIRONMENT, opener=mock.Mock())

    def test_proxy_inheritance_and_redirects_are_disabled(self):
        sentinel_opener = object()
        sentinel_context = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
        with (
            mock.patch.object(status_tool.ssl, "create_default_context", return_value=sentinel_context),
            mock.patch.object(status_tool.urllib.request, "build_opener", return_value=sentinel_opener) as builder,
        ):
            self.assertIs(status_tool.build_opener(), sentinel_opener)
        handlers = builder.call_args.args
        self.assertIsInstance(handlers[0], urllib.request.ProxyHandler)
        self.assertEqual(handlers[0].proxies, {})
        self.assertIsInstance(handlers[1], status_tool.NoRedirect)
        self.assertIsNone(handlers[1].redirect_request(None, None, 302, "", {}, "https://evil.example"))
        self.assertIsInstance(handlers[2], urllib.request.HTTPSHandler)

        redirect = FakeOpener(FakeResponse(302, b""))
        request = status_tool.build_request("/api/v1/admin/provider-proof", TEST_ADMIN_KEY)
        with self.assertRaisesRegex(status_tool.StatusError, "redirect_refused"):
            status_tool.perform_get(redirect, request)

    def test_http_error_body_and_network_details_are_never_read_or_reflected(self):
        private_body = io.BytesIO(b"private server body and admin details")
        error = urllib.error.HTTPError(
            status_tool.BASE_URL + "/api/v1/admin/provider-proof",
            503,
            "private diagnostic",
            {},
            private_body,
        )
        opener = FakeOpener(error)
        request = status_tool.build_request("/api/v1/admin/provider-proof", TEST_ADMIN_KEY)
        with self.assertRaises(status_tool.StatusError) as raised:
            status_tool.perform_get(opener, request)
        self.assertEqual(raised.exception.code, "http_error")
        self.assertEqual(raised.exception.http_status, 503)
        self.assertEqual(private_body.tell(), 0)
        self.assertNotIn("private diagnostic", str(raised.exception))

        network = FakeOpener(urllib.error.URLError("secret network detail"))
        with self.assertRaisesRegex(status_tool.StatusError, "network_error"):
            status_tool.perform_get(network, request)

    def test_response_is_bounded_and_invalid_json_fails_closed(self):
        request = status_tool.build_request("/api/v1/admin/provider-proof", TEST_ADMIN_KEY)
        oversized = FakeResponse(200, b"x" * (status_tool.MAX_RESPONSE_BYTES + 50))
        with self.assertRaisesRegex(status_tool.StatusError, "response_too_large"):
            status_tool.perform_get(FakeOpener(oversized), request)
        self.assertEqual(oversized.read_sizes, [status_tool.MAX_RESPONSE_BYTES + 1])
        with self.assertRaisesRegex(status_tool.StatusError, "invalid_response"):
            status_tool.perform_get(FakeOpener(FakeResponse(200, b"[]")), request)

    def test_main_hardens_core_dumps_and_sanitizes_unexpected_errors(self):
        stdout = io.StringIO()
        with (
            mock.patch.object(status_tool.resource, "setrlimit") as set_limit,
            mock.patch.object(status_tool, "run", return_value={"ok": True, "scope": "all"}),
            contextlib.redirect_stdout(stdout),
        ):
            result = status_tool.main([])
        self.assertEqual(result, 0)
        set_limit.assert_called_once_with(status_tool.resource.RLIMIT_CORE, (0, 0))

        stderr = io.StringIO()
        with (
            mock.patch.object(status_tool.resource, "setrlimit", side_effect=OSError("private")),
            contextlib.redirect_stderr(stderr),
        ):
            result = status_tool.main([])
        self.assertEqual(result, 1)
        self.assertEqual(
            json.loads(stderr.getvalue()),
            {"error": "core_dump_hardening_unavailable", "ok": False},
        )

    def test_source_has_no_arbitrary_host_or_secret_output_surface(self):
        help_text = status_tool.build_parser().format_help().lower()
        self.assertNotIn("base-url", help_text)
        source = MODULE_PATH.read_text(encoding="utf-8")
        for forbidden in ("sys.stdin", "input(", "--url", "--host", "HTTPPasswordMgr"):
            self.assertNotIn(forbidden, source)


if __name__ == "__main__":
    unittest.main()
