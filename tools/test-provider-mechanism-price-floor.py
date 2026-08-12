#!/usr/bin/env python3

import importlib.util
import json
import pathlib
import sys
import tempfile
import unittest


MODULE_PATH = pathlib.Path(__file__).with_name("provider-mechanism-price-floor.py")
SPEC = importlib.util.spec_from_file_location("provider_mechanism_price_floor", MODULE_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)
BASELINE_PATH = pathlib.Path(__file__).parents[1] / "docs/experiments/nhs-cost-baseline-2026-08-12.json"
POLICY_PATH = pathlib.Path(__file__).parents[1] / "docs/experiments/provider-mechanism-policy-v4.json"
REPORT_PATH = pathlib.Path(__file__).parents[1] / "docs/experiments/provider-mechanism-price-floor-2026-08-12.json"


class ProviderMechanismPriceFloorTest(unittest.TestCase):
    def test_baseline_is_exact_and_reconciled(self):
        baseline = MODULE.load_baseline(BASELINE_PATH)
        self.assertEqual(baseline["observed_cost_cents"], 1146)
        self.assertEqual(sum(item["observed_cost_cents"] for item in baseline["applications"]), 1146)

    def test_baseline_drift_fails_closed(self):
        baseline = json.loads(BASELINE_PATH.read_text())
        baseline["applications"][0]["observed_cost_cents"] += 1
        with tempfile.TemporaryDirectory() as directory:
            path = pathlib.Path(directory) / "baseline.json"
            path.write_text(json.dumps(baseline))
            with self.assertRaisesRegex(MODULE.PriceFloorError, "does not reconcile"):
                MODULE.load_baseline(path)

    def test_processing_allowance_rounds_percentage_up(self):
        policy = MODULE.ProcessingPolicy(290, 30)
        self.assertEqual(MODULE.processing_fee_cents(100, policy), 33)
        self.assertEqual(MODULE.processing_fee_cents(101, policy), 33)
        self.assertEqual(MODULE.processing_fee_cents(1000, policy), 59)

    def test_every_price_floor_is_minimal(self):
        report = MODULE.build_report(
            MODULE.load_baseline(BASELINE_PATH), MODULE.DEFAULT_EVENT_COUNTS,
            MODULE.ProcessingPolicy(290, 30),
        )
        for group in ("standalone_monthly_fixed_cost_coverage", "equal_three_arm_pilot_cost_allocation"):
            for mechanism, result in report[group].items():
                with self.subTest(group=group, mechanism=mechanism):
                    self.assertGreaterEqual(result["projected_total_processing_net_cents"], result["required_net_cents"])
                    self.assertLess(result["one_cent_lower_total_processing_net_cents"], result["required_net_cents"])

    def test_standalone_prices_reflect_charge_frequency(self):
        report = MODULE.build_report(
            MODULE.load_baseline(BASELINE_PATH), MODULE.DEFAULT_EVENT_COUNTS,
            MODULE.ProcessingPolicy(290, 30),
        )
        floors = report["standalone_monthly_fixed_cost_coverage"]
        self.assertLess(floors["accepted"]["minimum_gross_cents_per_event"], floors["activated"]["minimum_gross_cents_per_event"])
        self.assertLess(floors["activated"]["minimum_gross_cents_per_event"], floors["converted"]["minimum_gross_cents_per_event"])
        self.assertFalse(report["commercial_proof"])
        self.assertFalse(report["production_changed"])
        self.assertFalse(report["organic_discovery_paywalled"])

    def test_equal_allocation_reconciles_odd_cent(self):
        allocation = MODULE.allocate_equal_cost(1146)
        self.assertEqual(allocation, {"accepted": 382, "activated": 382, "converted": 382})
        self.assertEqual(sum(allocation.values()), 1146)

    def test_current_policy_absolute_lead_tracks_cost_baseline(self):
        baseline = MODULE.load_baseline(BASELINE_PATH)
        policy = json.loads(POLICY_PATH.read_text())
        self.assertEqual(policy["name"], "developer-tools-policy-v4")
        self.assertEqual(
            policy["min_processing_net_lead_per_1000_offer_returns_cents"],
            baseline["observed_cost_cents"],
        )

    def test_saved_price_floor_report_matches_current_inputs(self):
        expected = MODULE.build_report(
            MODULE.load_baseline(BASELINE_PATH), MODULE.DEFAULT_EVENT_COUNTS,
            MODULE.ProcessingPolicy(290, 30),
        )
        self.assertEqual(json.loads(REPORT_PATH.read_text()), expected)


if __name__ == "__main__":
    unittest.main()
