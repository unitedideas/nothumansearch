#!/usr/bin/env python3
"""Compute provider-funded NHS event price floors from observed fixed cost.

This is a pricing hypothesis tool, not a revenue or mechanism-selection tool.
It never changes production, contacts a provider, or charges a principal.
"""

from __future__ import annotations

import argparse
import json
import math
import pathlib
from dataclasses import dataclass
from typing import Any


DEFAULT_BASELINE = pathlib.Path("docs/experiments/nhs-cost-baseline-2026-08-12.json")
MECHANISMS = ("accepted", "activated", "converted")
DEFAULT_EVENT_COUNTS = {"accepted": 5, "activated": 2, "converted": 1}
BASELINE_KEYS = {
    "contract", "scope", "currency", "period_start", "period_end",
    "period_days", "observed_cost_cents", "applications", "source",
    "source_is_final_invoice", "contains_other_apps", "includes_owner_labor",
    "includes_support", "includes_fraud_losses", "includes_tax",
    "commercial_proof",
}


class PriceFloorError(RuntimeError):
    pass


@dataclass(frozen=True)
class ProcessingPolicy:
    basis_points: int
    fixed_cents: int


def load_baseline(path: pathlib.Path) -> dict[str, Any]:
    try:
        document = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        raise PriceFloorError("cost baseline is unavailable") from error
    if not isinstance(document, dict) or set(document) != BASELINE_KEYS:
        raise PriceFloorError("cost baseline contract drifted")
    if document["contract"] != "nhs-fixed-cost-baseline-v1" or document["scope"] != "not-human-search-only":
        raise PriceFloorError("cost baseline scope is invalid")
    if document["currency"] != "usd" or document["commercial_proof"] is not False:
        raise PriceFloorError("cost baseline evidence boundary is invalid")
    if document["contains_other_apps"] is not False or document["source_is_final_invoice"] is not False:
        raise PriceFloorError("cost baseline source boundary is invalid")
    days = document["period_days"]
    total = document["observed_cost_cents"]
    applications = document["applications"]
    if not isinstance(days, int) or not 28 <= days <= 35 or not isinstance(total, int) or total <= 0:
        raise PriceFloorError("cost baseline amount is invalid")
    if not isinstance(applications, list) or len(applications) != 2:
        raise PriceFloorError("cost baseline application inventory is invalid")
    if {item.get("app") for item in applications if isinstance(item, dict)} != {"nothumansearch", "nothumansearch-db"}:
        raise PriceFloorError("cost baseline application inventory is invalid")
    if any(set(item) != {"app", "observed_cost_cents"} for item in applications):
        raise PriceFloorError("cost baseline application entry drifted")
    if sum(item["observed_cost_cents"] for item in applications) != total:
        raise PriceFloorError("cost baseline does not reconcile")
    return document


def processing_fee_cents(gross_cents: int, policy: ProcessingPolicy) -> int:
    if gross_cents < 0:
        raise PriceFloorError("gross cents must be nonnegative")
    return math.ceil(gross_cents * policy.basis_points / 10_000) + policy.fixed_cents


def net_cents(gross_cents: int, policy: ProcessingPolicy) -> int:
    return gross_cents - processing_fee_cents(gross_cents, policy)


def minimum_equal_event_price(event_count: int, required_net_cents: int, policy: ProcessingPolicy) -> dict[str, int]:
    if event_count <= 0 or required_net_cents <= 0:
        raise PriceFloorError("event count and required net must be positive")
    low, high = 0, required_net_cents + policy.fixed_cents + 1
    while event_count * net_cents(high, policy) < required_net_cents:
        high *= 2
    while low < high:
        midpoint = (low + high) // 2
        if event_count * net_cents(midpoint, policy) >= required_net_cents:
            high = midpoint
        else:
            low = midpoint + 1
    gross_each = low
    fee_each = processing_fee_cents(gross_each, policy)
    total_gross = gross_each * event_count
    total_fees = fee_each * event_count
    return {
        "event_count": event_count,
        "required_net_cents": required_net_cents,
        "minimum_gross_cents_per_event": gross_each,
        "processing_fee_allowance_cents_per_event": fee_each,
        "projected_total_gross_cents": total_gross,
        "projected_total_processing_fee_cents": total_fees,
        "projected_total_processing_net_cents": total_gross - total_fees,
        "one_cent_lower_total_processing_net_cents": event_count * net_cents(max(0, gross_each - 1), policy),
    }


def allocate_equal_cost(total_cents: int) -> dict[str, int]:
    quotient, remainder = divmod(total_cents, len(MECHANISMS))
    return {
        mechanism: quotient + (1 if index < remainder else 0)
        for index, mechanism in enumerate(MECHANISMS)
    }


def build_report(baseline: dict[str, Any], event_counts: dict[str, int], policy: ProcessingPolicy) -> dict[str, Any]:
    fixed_cost = baseline["observed_cost_cents"]
    allocations = allocate_equal_cost(fixed_cost)
    return {
        "contract": "nhs-provider-mechanism-price-floor-v1",
        "scope": "not-human-search-only",
        "currency": "usd",
        "baseline": {
            "period_start": baseline["period_start"],
            "period_end": baseline["period_end"],
            "period_days": baseline["period_days"],
            "observed_fixed_cost_cents": fixed_cost,
            "source": baseline["source"],
            "source_is_final_invoice": baseline["source_is_final_invoice"],
        },
        "processing_policy": {
            "basis_points": policy.basis_points,
            "fixed_cents_per_successful_charge": policy.fixed_cents,
            "fee_kind": "conservative_published_allowance_not_observed_processor_receipt",
        },
        "standalone_monthly_fixed_cost_coverage": {
            mechanism: minimum_equal_event_price(event_counts[mechanism], fixed_cost, policy)
            for mechanism in MECHANISMS
        },
        "equal_three_arm_pilot_cost_allocation": {
            mechanism: minimum_equal_event_price(event_counts[mechanism], allocations[mechanism], policy)
            for mechanism in MECHANISMS
        },
        "interpretation": "Infrastructure contribution floor only; excludes labor, support, fraud, tax, and profit. Final mechanism selection requires actual available processor-net settlement receipts.",
        "organic_discovery_paywalled": False,
        "principal_charged": False,
        "provider_contacted": False,
        "production_changed": False,
        "commercial_proof": False,
    }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--baseline", type=pathlib.Path, default=DEFAULT_BASELINE)
    parser.add_argument("--accepted-events", type=int, default=5)
    parser.add_argument("--activated-events", type=int, default=2)
    parser.add_argument("--converted-events", type=int, default=1)
    parser.add_argument("--processing-basis-points", type=int, default=290)
    parser.add_argument("--processing-fixed-cents", type=int, default=30)
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    try:
        event_counts = {"accepted": args.accepted_events, "activated": args.activated_events, "converted": args.converted_events}
        if any(not 1 <= value <= 1_000_000 for value in event_counts.values()):
            raise PriceFloorError("event counts must be 1..1000000")
        if not 0 <= args.processing_basis_points <= 10_000 or not 0 <= args.processing_fixed_cents <= 100_000:
            raise PriceFloorError("processing policy is invalid")
        report = build_report(
            load_baseline(args.baseline),
            event_counts,
            ProcessingPolicy(args.processing_basis_points, args.processing_fixed_cents),
        )
        print(json.dumps(report, separators=(",", ":"), sort_keys=True))
        return 0
    except PriceFloorError as error:
        print(json.dumps({"ok": False, "error": str(error)}, separators=(",", ":"), sort_keys=True))
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
