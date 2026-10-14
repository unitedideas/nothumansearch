#!/usr/bin/env python3

import json
import pathlib
import unittest


EVIDENCE_PATH = pathlib.Path(__file__).parents[1] / "docs/experiments/provider-processor-boundary-2026-08-12.json"


class ProviderProcessorBoundaryTest(unittest.TestCase):
    def setUp(self):
        self.evidence = json.loads(EVIDENCE_PATH.read_text())

    def test_live_mode_write_drill_failed_closed(self):
        self.assertEqual(self.evidence["contract"], "nhs-provider-processor-boundary-v1")
        self.assertEqual(self.evidence["scope"], "not-human-search-only")
        self.assertEqual(self.evidence["stripe_api_read"]["account_mode"], "live")
        self.assertFalse(self.evidence["processor_write_drill_performed"])
        self.assertEqual(self.evidence["processor_write_drill_reason"], "fail_closed_live_mode_credential")

    def test_zero_provider_settlement_objects_reconciles(self):
        search = self.evidence["stripe_api_read"]["provider_settlement_payment_intent_search"]
        self.assertEqual(search["metadata_product"], "nhs_provider_outcome_settlement")
        self.assertEqual(search["returned_count"], 0)
        self.assertFalse(search["has_more"])
        self.assertEqual(search["inferred_total_count"], 0)
        self.assertEqual(self.evidence["charges_created"], 0)

    def test_disabled_noncommercial_boundary(self):
        self.assertEqual(self.evidence["production_provider_exchange_mode"], "disabled")
        self.assertEqual(self.evidence["providers_contacted"], 0)
        self.assertEqual(self.evidence["providers_activated"], 0)
        self.assertFalse(self.evidence["commercial_proof"])
        self.assertIsNone(self.evidence["mechanism_winner"])

    def test_evidence_contains_no_private_processor_or_party_data(self):
        self.assertFalse(self.evidence["contains_stripe_object_ids"])
        self.assertFalse(self.evidence["contains_customer_data"])
        self.assertFalse(self.evidence["contains_provider_data"])
        raw = EVIDENCE_PATH.read_text().lower()
        for forbidden in ("customer_email", "billing_email", "payment_intent_id", "checkout_session_id", "provider_id"):
            self.assertNotIn(forbidden, raw)


if __name__ == "__main__":
    unittest.main()
