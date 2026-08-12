#!/usr/bin/env python3
"""Run one minimum-value Stripe test-mode processor drill.

The command fails closed before its first write unless Stripe's balance object
explicitly reports livemode=false. It creates and refunds one test PaymentIntent,
validates the charge balance transaction, and emits no Stripe object IDs.
"""

from __future__ import annotations

import argparse
import base64
import json
import os
import re
import urllib.error
import urllib.parse
import urllib.request
from typing import Any


API_ORIGIN = "https://api.stripe.com"
CONTRACT = "nhs-provider-stripe-test-mode-drill-v1"
PRODUCT = "nhs_provider_outcome_settlement_sandbox"
DRILL_ID_PATTERN = re.compile(r"^[a-z0-9][a-z0-9_-]{7,63}$")


class DrillError(RuntimeError):
    pass


class StripeClient:
    def __init__(self, secret: str):
        if not secret:
            raise DrillError("Stripe credential is unavailable")
        self._authorization = base64.b64encode((secret + ":").encode()).decode()

    def request(
        self,
        method: str,
        path: str,
        form: dict[str, str | int] | None = None,
        idempotency_key: str | None = None,
    ) -> dict[str, Any]:
        if not path.startswith("/v1/") or "//" in path:
            raise DrillError("Stripe API path is invalid")
        data = None
        headers = {"Authorization": "Basic " + self._authorization}
        if form is not None:
            data = urllib.parse.urlencode(form).encode()
            headers["Content-Type"] = "application/x-www-form-urlencoded"
        if idempotency_key:
            headers["Idempotency-Key"] = idempotency_key
        request = urllib.request.Request(API_ORIGIN + path, data=data, headers=headers, method=method)
        try:
            with urllib.request.urlopen(request, timeout=30) as response:
                return json.load(response)
        except urllib.error.HTTPError as error:
            try:
                document = json.load(error)
                error_type = document.get("error", {}).get("type", "unknown")
            except (json.JSONDecodeError, AttributeError):
                error_type = "unknown"
            raise DrillError(f"Stripe API request failed with status {error.code} and type {error_type}") from error


def require_string(document: dict[str, Any], field: str, prefix: str) -> str:
    value = document.get(field)
    if not isinstance(value, str) or not value.startswith(prefix):
        raise DrillError(f"Stripe {field} is invalid")
    return value


def run_drill(client: Any, amount_cents: int, drill_id: str) -> dict[str, Any]:
    if not 50 <= amount_cents <= 100:
        raise DrillError("sandbox amount must be 50..100 cents")
    if not DRILL_ID_PATTERN.fullmatch(drill_id):
        raise DrillError("drill ID is invalid")

    balance = client.request("GET", "/v1/balance")
    if balance.get("livemode") is not False:
        raise DrillError("refusing Stripe write because credential is not explicitly test-mode")

    charge_id = ""
    refund_attempted = False
    try:
        intent = client.request(
            "POST",
            "/v1/payment_intents",
            {
                "amount": amount_cents,
                "currency": "usd",
                "payment_method": "pm_card_visa",
                "payment_method_types[]": "card",
                "confirm": "true",
                "description": "NHS provider settlement processor sandbox drill",
                "metadata[product]": PRODUCT,
                "metadata[drill_contract]": CONTRACT,
                "metadata[drill_id]": drill_id,
            },
            f"nhs-provider-stripe-test:{drill_id}:payment",
        )
        require_string(intent, "id", "pi_")
        charge_id = require_string(intent, "latest_charge", "ch_")
        if intent.get("livemode") is not False or intent.get("status") != "succeeded" or intent.get("amount_received") != amount_cents:
            raise DrillError("test PaymentIntent did not settle exactly")

        charge = client.request("GET", f"/v1/charges/{urllib.parse.quote(charge_id)}?expand[]=balance_transaction")
        balance_transaction = charge.get("balance_transaction")
        if (
            charge.get("livemode") is not False
            or charge.get("paid") is not True
            or charge.get("amount") != amount_cents
            or charge.get("currency") != "usd"
            or not isinstance(balance_transaction, dict)
        ):
            raise DrillError("test charge did not reconcile")
        require_string(balance_transaction, "id", "txn_")
        fee_cents = balance_transaction.get("fee")
        net_cents = balance_transaction.get("net")
        if (
            balance_transaction.get("livemode") is not False
            or balance_transaction.get("amount") != amount_cents
            or balance_transaction.get("currency") != "usd"
            or balance_transaction.get("status") not in ("available", "pending")
            or not isinstance(fee_cents, int)
            or not isinstance(net_cents, int)
            or fee_cents < 0
            or fee_cents + net_cents != amount_cents
        ):
            raise DrillError("test balance transaction did not reconcile")

        refund_attempted = True
        refund = client.request(
            "POST",
            "/v1/refunds",
            {"charge": charge_id, "reason": "requested_by_customer", "metadata[drill_contract]": CONTRACT},
            f"nhs-provider-stripe-test:{drill_id}:refund",
        )
        require_string(refund, "id", "re_")
        if refund.get("status") not in ("pending", "succeeded") or refund.get("amount") != amount_cents:
            raise DrillError("test refund did not reconcile")

        return {
            "contract": CONTRACT,
            "scope": "not-human-search-only",
            "stripe_account_mode": "test",
            "amount_cents": amount_cents,
            "currency": "usd",
            "payment_intent_succeeded": True,
            "charge_paid": True,
            "balance_transaction_status": balance_transaction["status"],
            "processor_fee_cents": fee_cents,
            "processor_net_cents": net_cents,
            "refund_status": refund["status"],
            "contains_stripe_object_ids": False,
            "contains_customer_data": False,
            "contains_provider_data": False,
            "provider_exchange_activated": False,
            "commercial_proof": False,
            "mechanism_winner": None,
        }
    except Exception:
        if charge_id and not refund_attempted:
            try:
                client.request(
                    "POST",
                    "/v1/refunds",
                    {"charge": charge_id, "reason": "requested_by_customer", "metadata[drill_contract]": CONTRACT},
                    f"nhs-provider-stripe-test:{drill_id}:refund",
                )
            except Exception:
                pass
        raise


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--drill-id", required=True)
    parser.add_argument("--amount-cents", type=int, default=50)
    parser.add_argument("--confirm-test-mode-write", action="store_true")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    if not args.confirm_test_mode_write:
        print(json.dumps({"ok": False, "error": "explicit test-mode write confirmation required"}, separators=(",", ":")))
        return 2
    try:
        receipt = run_drill(StripeClient(os.environ.get("STRIPE_SECRET_KEY", "")), args.amount_cents, args.drill_id)
        print(json.dumps(receipt, separators=(",", ":"), sort_keys=True))
        return 0
    except DrillError as error:
        print(json.dumps({"ok": False, "error": str(error)}, separators=(",", ":"), sort_keys=True))
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
