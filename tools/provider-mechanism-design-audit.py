#!/usr/bin/env python3
"""Audit whether an NHS pilot design can support the declared mechanism selector."""

from __future__ import annotations

import argparse
import json
import math
import sys
from pathlib import Path
from typing import Any


MECHANISMS = ("accepted", "activated", "converted")


class DesignError(RuntimeError):
    pass


def _integer(policy: dict[str, Any], name: str, minimum: int = 1) -> int:
    value = policy.get(name)
    if not isinstance(value, int) or isinstance(value, bool) or value < minimum:
        raise DesignError(f"policy field {name} is invalid")
    return value


def _rate(policy: dict[str, Any], name: str) -> float:
    value = policy.get(name)
    if not isinstance(value, (int, float)) or isinstance(value, bool) or value <= 0 or value >= 1:
        raise DesignError(f"policy field {name} is invalid")
    return float(value)


def minimum_zero_variance_returns_per_arm(
    upper_bound_cents: int,
    confidence: float,
    arms: int,
    decisive_lead_per_thousand_cents: float,
) -> int:
    """Necessary, not sufficient, n for non-overlap at exactly the declared lead.

    This sets empirical variance to zero, leaving only the bounded-support term
    from the production selector's simultaneous empirical-Bernstein interval.
    Real variance can only increase the required sample.
    """
    if upper_bound_cents < 1 or arms < 2 or not 0.5 <= confidence < 1:
        raise DesignError("confidence design inputs are invalid")
    if decisive_lead_per_thousand_cents <= 0:
        raise DesignError("decisive lead must be positive")
    delta = (1 - confidence) / (2 * arms)
    log_term = math.log(2 / delta)
    threshold = 1 + (
        14 * upper_bound_cents * log_term * 1000
        / (3 * decisive_lead_per_thousand_cents)
    )
    return math.floor(threshold) + 1


def audit(
    policy: dict[str, Any],
    milestone_accepted: int,
    milestone_activated: int,
    milestone_converted: int,
    milestone_renewals: int,
) -> dict[str, Any]:
    for name, value in {
        "milestone_accepted": milestone_accepted,
        "milestone_activated": milestone_activated,
        "milestone_converted": milestone_converted,
        "milestone_renewals": milestone_renewals,
    }.items():
        if not isinstance(value, int) or isinstance(value, bool) or value < 0:
            raise DesignError(f"{name} is invalid")

    arms = len(MECHANISMS)
    min_charged = _integer(policy, "min_charged_events")
    min_providers = _integer(policy, "min_charged_provider_companies_per_mechanism")
    min_returns = _integer(policy, "min_offer_returns_per_mechanism")
    min_settlements = _integer(policy, "min_paid_settlements_per_mechanism")
    upper_bound = _integer(policy, "max_processor_net_cents_per_offer")
    absolute_lead = _integer(policy, "min_processing_net_lead_per_1000_offer_returns_cents")
    relative_lead = _rate(policy, "min_processing_net_lead_rate")
    minimum_viable_net = _integer(policy, "min_processing_net_revenue_per_1000_offer_returns_cents")
    confidence = _rate(policy, "selection_confidence_level")
    decisive_lead = max(float(absolute_lead), relative_lead * minimum_viable_net)

    minimum_accepted = arms * min_charged
    minimum_activated = 2 * min_charged
    minimum_converted = min_charged
    zero_variance_returns = minimum_zero_variance_returns_per_arm(
        upper_bound, confidence, arms, decisive_lead
    )
    milestone_meets_funnel = (
        milestone_accepted >= minimum_accepted
        and milestone_activated >= minimum_activated
        and milestone_converted >= minimum_converted
    )
    policy_floor_can_separate = min_returns >= zero_variance_returns
    passes_necessary_design_floors = milestone_meets_funnel and policy_floor_can_separate

    return {
        "contract": "nhs-provider-mechanism-design-audit-v1",
        "policy_name": policy.get("name"),
        "mechanisms": list(MECHANISMS),
        "declared_policy_floor": {
            "returned_offers_total": arms * min_returns,
            "returned_offers_per_mechanism": min_returns,
            "charged_events_total": arms * min_charged,
            "charged_events_per_mechanism": min_charged,
            "charged_provider_companies_per_mechanism": min_providers,
            "paid_settlements_total": arms * min_settlements,
            "paid_settlements_per_mechanism": min_settlements,
            "accepted_outcomes_implied_by_state_funnel": minimum_accepted,
            "activated_outcomes_implied_by_state_funnel": minimum_activated,
            "converted_outcomes": minimum_converted,
        },
        "revenue_proof_milestone": {
            "accepted": milestone_accepted,
            "activated": milestone_activated,
            "converted": milestone_converted,
            "renewals": milestone_renewals,
            "can_satisfy_per_arm_selection_funnel": milestone_meets_funnel,
            "can_select_mechanism": False,
            "purpose": "real revenue-rail proof only",
        },
        "confidence_design": {
            "confidence_level": confidence,
            "simultaneous_arms": arms,
            "max_processor_net_cents_per_offer": upper_bound,
            "declared_decisive_lead_per_1000_offer_returns_cents": decisive_lead,
            "minimum_zero_variance_returns_per_arm_at_declared_lead": zero_variance_returns,
            "necessary_not_sufficient": True,
            "real_variance_can_only_increase_required_returns": True,
            "policy_return_floor_can_separate_at_declared_lead": policy_floor_can_separate,
        },
        "passes_deterministic_necessary_design_floors": passes_necessary_design_floors,
        "selection_design_ready": False,
        "selection_design_ready_reason": (
            "This audit proves necessary floors only. A sufficient design must also preregister "
            "a traffic-feasible effect size and variance or stopping model."
        ),
        "commercial_proof": False,
        "authorizes_provider_contact_or_activation": False,
        "interpretation": (
            "The 3/5/2/1 milestone is an existence proof for consent, attribution, outcome, "
            "renewal, billing, and settlement rails. It cannot name a strongest mechanism. "
            "A selection cohort needs a separately preregistered, traffic-feasible effect size "
            "and sample design before provider activation."
        ),
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--policy", required=True)
    parser.add_argument("--accepted", type=int, default=5)
    parser.add_argument("--activated", type=int, default=2)
    parser.add_argument("--converted", type=int, default=0)
    parser.add_argument("--renewals", type=int, default=1)
    args = parser.parse_args()
    try:
        raw = Path(args.policy).read_text(encoding="utf-8")
        policy = json.loads(raw)
        if not isinstance(policy, dict):
            raise DesignError("policy must be a JSON object")
        result = audit(policy, args.accepted, args.activated, args.converted, args.renewals)
    except (OSError, json.JSONDecodeError, DesignError) as error:
        print(f"provider_mechanism_design_error: {error}", file=sys.stderr)
        return 2
    json.dump(result, sys.stdout, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
