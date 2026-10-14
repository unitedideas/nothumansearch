#!/usr/bin/env python3

import importlib.util
import pathlib
import sys
import unittest


MODULE_PATH = pathlib.Path(__file__).with_name("provider-stripe-test-mode-drill.py")
SPEC = importlib.util.spec_from_file_location("provider_stripe_test_mode_drill", MODULE_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


class FakeStripeClient:
    def __init__(self, livemode=False, mismatch=False):
        self.livemode = livemode
        self.mismatch = mismatch
        self.calls = []

    def request(self, method, path, form=None, idempotency_key=None):
        self.calls.append((method, path, form, idempotency_key))
        if path == "/v1/balance":
            return {"object": "balance", "livemode": self.livemode}
        if path == "/v1/payment_intents":
            return {"id": "pi_fixture", "latest_charge": "ch_fixture", "livemode": False, "status": "succeeded", "amount_received": 50}
        if path.startswith("/v1/charges/"):
            return {
                "id": "ch_fixture", "livemode": False, "paid": True,
                "amount": 50 if not self.mismatch else 51, "currency": "usd",
                "balance_transaction": {
                    "id": "txn_fixture", "livemode": False, "amount": 50,
                    "fee": 32, "net": 18, "currency": "usd", "status": "pending",
                },
            }
        if path == "/v1/refunds":
            return {"id": "re_fixture", "status": "succeeded", "amount": 50}
        raise AssertionError(path)


class ProviderStripeTestModeDrillTest(unittest.TestCase):
    def test_live_mode_fails_before_first_write(self):
        client = FakeStripeClient(livemode=True)
        with self.assertRaisesRegex(MODULE.DrillError, "not explicitly test-mode"):
            MODULE.run_drill(client, 50, "drill-live-0001")
        self.assertEqual(client.calls, [("GET", "/v1/balance", None, None)])

    def test_test_mode_receipt_reconciles_and_omits_ids(self):
        client = FakeStripeClient()
        receipt = MODULE.run_drill(client, 50, "drill-test-0001")
        self.assertEqual(receipt["processor_fee_cents"] + receipt["processor_net_cents"], 50)
        self.assertEqual(receipt["refund_status"], "succeeded")
        self.assertFalse(receipt["contains_stripe_object_ids"])
        self.assertFalse(receipt["commercial_proof"])
        self.assertIsNone(receipt["mechanism_winner"])
        self.assertEqual([call[0:2] for call in client.calls], [
            ("GET", "/v1/balance"),
            ("POST", "/v1/payment_intents"),
            ("GET", "/v1/charges/ch_fixture?expand[]=balance_transaction"),
            ("POST", "/v1/refunds"),
        ])

    def test_post_charge_validation_failure_attempts_refund(self):
        client = FakeStripeClient(mismatch=True)
        with self.assertRaisesRegex(MODULE.DrillError, "charge did not reconcile"):
            MODULE.run_drill(client, 50, "drill-bad-0001")
        self.assertEqual(client.calls[-1][0:2], ("POST", "/v1/refunds"))

    def test_amount_and_drill_id_are_bounded(self):
        client = FakeStripeClient()
        with self.assertRaisesRegex(MODULE.DrillError, "50..100"):
            MODULE.run_drill(client, 49, "drill-test-0001")
        with self.assertRaisesRegex(MODULE.DrillError, "drill ID"):
            MODULE.run_drill(client, 50, "bad")
        self.assertEqual(client.calls, [])


if __name__ == "__main__":
    unittest.main()
