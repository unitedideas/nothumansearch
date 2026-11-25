#!/usr/bin/env python3

from __future__ import annotations

import importlib.util
import unittest
from pathlib import Path


MODULE_PATH = Path(__file__).with_name("provider-mechanism-design-audit.py")
SPEC = importlib.util.spec_from_file_location("provider_mechanism_design_audit", MODULE_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
SPEC.loader.exec_module(MODULE)


def policy() -> dict:
    return {
        "name": "developer-tools-policy-v4",
        "min_charged_events": 5,
        "min_charged_provider_companies_per_mechanism": 3,
        "min_offer_returns_per_mechanism": 20,
        "min_paid_settlements_per_mechanism": 1,
        "max_processor_net_cents_per_offer": 20000,
        "min_processing_net_lead_per_1000_offer_returns_cents": 1146,
        "min_processing_net_lead_rate": 0.2,
        "min_processing_net_revenue_per_1000_offer_returns_cents": 1000,
        "selection_confidence_level": 0.95,
    }


class ProviderMechanismDesignAuditTest(unittest.TestCase):
    def test_current_milestone_is_revenue_proof_not_selection(self) -> None:
        result = MODULE.audit(policy(), 5, 2, 0, 1)
        floor = result["declared_policy_floor"]
        self.assertEqual(floor["returned_offers_total"], 60)
        self.assertEqual(floor["charged_events_total"], 15)
        self.assertEqual(floor["paid_settlements_total"], 3)
        self.assertEqual(floor["accepted_outcomes_implied_by_state_funnel"], 15)
        self.assertEqual(floor["activated_outcomes_implied_by_state_funnel"], 10)
        self.assertEqual(floor["converted_outcomes"], 5)
        self.assertFalse(result["revenue_proof_milestone"]["can_select_mechanism"])
        self.assertFalse(result["selection_design_ready"])
        self.assertFalse(result["commercial_proof"])

    def test_current_confidence_floor_is_not_traffic_feasible(self) -> None:
        result = MODULE.audit(policy(), 15, 10, 5, 1)
        confidence = result["confidence_design"]
        self.assertEqual(confidence["minimum_zero_variance_returns_per_arm_at_declared_lead"], 446360)
        self.assertFalse(confidence["policy_return_floor_can_separate_at_declared_lead"])
        self.assertFalse(result["selection_design_ready"])

    def test_larger_sample_can_clear_deterministic_design_floor(self) -> None:
        revised = policy()
        revised["min_offer_returns_per_mechanism"] = 500000
        result = MODULE.audit(revised, 15, 10, 5, 1)
        self.assertTrue(result["passes_deterministic_necessary_design_floors"])
        self.assertFalse(result["selection_design_ready"])

    def test_invalid_policy_fails_closed(self) -> None:
        revised = policy()
        revised["selection_confidence_level"] = 1
        with self.assertRaisesRegex(MODULE.DesignError, "selection_confidence_level"):
            MODULE.audit(revised, 5, 2, 0, 1)


if __name__ == "__main__":
    unittest.main()
