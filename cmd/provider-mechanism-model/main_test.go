package main

import "testing"

func TestEvaluateSelectsActivatedForPilotScenario(t *testing.T) {
	report, err := evaluate(testScenario())
	if err != nil {
		t.Fatal(err)
	}
	if report.SelectedEvent != "activated" {
		t.Fatalf("selected event = %q, want activated", report.SelectedEvent)
	}
	if report.SelectedBountyCents != 7500 {
		t.Fatalf("selected bounty = %d, want 7500", report.SelectedBountyCents)
	}
	if report.CommercialProof || report.ProductionChanged || !report.Synthetic {
		t.Fatalf("unsafe proof flags: %+v", report)
	}
	if got := report.Results[0].GrossBillableCents; got != 90000 {
		t.Fatalf("top gross = %d, want 90000", got)
	}
	if len(report.Results) != 9 {
		t.Fatalf("results = %d, want a 3x3 event/price grid", len(report.Results))
	}
}

func TestEvaluateRejectsImmatureCohort(t *testing.T) {
	input := testScenario()
	input.MatureCohort = false
	if _, err := evaluate(input); err == nil {
		t.Fatal("expected immature cohort to fail")
	}
}

func TestEvaluateRejectsImpossibleFunnel(t *testing.T) {
	input := testScenario()
	input.Activated = input.Accepted + 1
	if _, err := evaluate(input); err == nil {
		t.Fatal("expected impossible funnel to fail")
	}
}

func TestEvaluateSensitivityBoundaries(t *testing.T) {
	tests := []struct {
		name   string
		change func(*scenario)
	}{
		{
			name: "provider cost ceiling below activated seventy-five dollar package",
			change: func(input *scenario) {
				input.MaxCostPerActivationCents = 6500
			},
		},
		{
			name: "cash latency below activation delay",
			change: func(input *scenario) {
				input.MaxMedianDaysToCharge = 3
			},
		},
		{
			name: "minimum sample above activation count",
			change: func(input *scenario) {
				input.MinChargedEvents = 20
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := testScenario()
			test.change(&input)
			report, err := evaluate(input)
			if err != nil {
				t.Fatal(err)
			}
			if report.SelectedEvent != "accepted" || report.SelectedBountyCents != 2500 {
				t.Fatalf("selected %s at %d, want accepted at 2500", report.SelectedEvent, report.SelectedBountyCents)
			}
		})
	}
}

func testScenario() scenario {
	return scenario{
		Name:                      "synthetic-developer-tools-v1",
		EvidenceKind:              "synthetic",
		MatureCohort:              true,
		Handoffs:                  50,
		Accepted:                  30,
		Activated:                 12,
		Converted:                 3,
		MaxCostPerActivationCents: 10000,
		MaxMedianDaysToCharge:     14,
		MinChargedEvents:          5,
		BountyPointsCents:         []int64{2500, 7500, 20000},
		ChargeEvents: []chargeEvent{
			{Name: "accepted", MedianDaysToCharge: 1},
			{Name: "activated", MedianDaysToCharge: 7},
			{Name: "converted", MedianDaysToCharge: 30},
		},
	}
}

func TestScenarioFromVerifiedProof(t *testing.T) {
	truth, falsity := true, false
	zero := int64(0)
	proof := verifiedProof{
		PilotID: "11111111-1111-4111-8111-111111111111", DemandTopic: "developer-tools", Status: "closed",
		OutcomeIntegrity: &truth, RejectedOutcomes: &zero, RejectedLedger: &zero,
		ObservedHandoffs: 50, Accepted: 30, Activated: 12, Converted: 3,
		AcceptedLatencySamples: 30, ActivatedLatencySamples: 12, ConvertedLatencySamples: 3,
		AcceptedMedianSeconds: 86400, ActivatedMedianSeconds: 604800, ConvertedMedianSeconds: 2592000,
		PilotThresholdsMet: &truth, OrganicRankSold: &falsity, RawQueriesSold: &falsity,
		RawPromptsSold: &falsity, AgentIdentitiesSold: &falsity, PrincipalIdentitiesSold: &falsity,
		SettlementIntegrity: &truth, PaidSettlements: 3, RejectedSettlements: &zero,
		PaidLatencySamples: 3, PaidMedianSeconds: 691200, PaidByCurrency: map[string]int64{"usd": 12500},
		VerifiedMechanisms: map[string]mechanismEvidence{
			"accepted": {
				ObservedHandoffs: 20, Accepted: 15, Activated: 8, Converted: 1,
				PaidSettlements: 1, PaidCents: 2000, PaidMedianSeconds: 172800,
			},
			"activated": {
				ObservedHandoffs: 20, Accepted: 10, Activated: 3, Converted: 1,
				PaidSettlements: 1, PaidCents: 7500, PaidMedianSeconds: 691200,
			},
			"converted": {
				ObservedHandoffs: 10, Accepted: 5, Activated: 1, Converted: 1,
				PaidSettlements: 1, PaidCents: 3000, PaidMedianSeconds: 1036800,
			},
		},
	}
	decision := policy{
		Name: "verified-policy", DemandTopic: "developer-tools", MaxCostPerActivationCents: 10000,
		MaxMedianDaysToCharge: 14, MinChargedEvents: 5, MinPaidSettlementsPerMechanism: 1,
		BountyPointsCents: []int64{2500, 7500, 20000},
	}
	input, err := scenarioFromVerifiedProof(proof, decision)
	if err != nil {
		t.Fatal(err)
	}
	report, err := evaluate(input)
	if err != nil {
		t.Fatal(err)
	}
	if report.Synthetic || report.EvidenceKind != "verified_closed_pilot" || report.SelectedEvent != "activated" ||
		!report.CollectedRevenueEvidence || report.VerifiedPaidSettlements != 3 ||
		report.Results[0].ValueKind != "verified_paid" {
		t.Fatalf("unexpected verified report: %+v", report)
	}

	proof.Status = "active"
	if _, err := scenarioFromVerifiedProof(proof, decision); err == nil {
		t.Fatal("expected active pilot proof to fail closed")
	}
	proof.Status = "closed"
	proof.RawQueriesSold = &truth
	if _, err := scenarioFromVerifiedProof(proof, decision); err == nil {
		t.Fatal("expected privacy-boundary violation to fail closed")
	}
	proof.RawQueriesSold = nil
	if _, err := scenarioFromVerifiedProof(proof, decision); err == nil {
		t.Fatal("expected missing privacy-boundary field to fail closed")
	}
	proof.RawQueriesSold = &falsity
	proof.PaidSettlements = 0
	if _, err := scenarioFromVerifiedProof(proof, decision); err == nil {
		t.Fatal("expected proof without a paid settlement to fail closed")
	}
}

func TestEvaluateVerifiedMechanismsRequiresRealPaymentForEveryArm(t *testing.T) {
	truth, falsity := true, false
	zero := int64(0)
	proof := verifiedProof{
		PilotID: "11111111-1111-4111-8111-111111111111", DemandTopic: "developer-tools", Status: "closed",
		OutcomeIntegrity: &truth, RejectedOutcomes: &zero, RejectedLedger: &zero,
		ObservedHandoffs: 6, Accepted: 6, Activated: 3, Converted: 1,
		AcceptedLatencySamples: 6, ActivatedLatencySamples: 3, ConvertedLatencySamples: 1,
		AcceptedMedianSeconds: 3600, ActivatedMedianSeconds: 86400, ConvertedMedianSeconds: 604800,
		PilotThresholdsMet: &truth, OrganicRankSold: &falsity, RawQueriesSold: &falsity,
		RawPromptsSold: &falsity, AgentIdentitiesSold: &falsity, PrincipalIdentitiesSold: &falsity,
		SettlementIntegrity: &truth, PaidSettlements: 2, RejectedSettlements: &zero,
		PaidLatencySamples: 2, PaidMedianSeconds: 172800, PaidByCurrency: map[string]int64{"usd": 5000},
		VerifiedMechanisms: map[string]mechanismEvidence{
			"accepted":  {ObservedHandoffs: 2, Accepted: 2, Activated: 1, PaidSettlements: 1, PaidCents: 2500, PaidMedianSeconds: 86400},
			"activated": {ObservedHandoffs: 2, Accepted: 2, Activated: 1, PaidSettlements: 1, PaidCents: 2500, PaidMedianSeconds: 172800},
			"converted": {ObservedHandoffs: 2, Accepted: 2, Activated: 1, Converted: 1},
		},
	}
	decision := policy{
		Name: "verified-policy", DemandTopic: "developer-tools", MaxCostPerActivationCents: 10000,
		MaxMedianDaysToCharge: 14, MinChargedEvents: 1, MinPaidSettlementsPerMechanism: 1,
		BountyPointsCents: []int64{2500},
	}
	input, err := scenarioFromVerifiedProof(proof, decision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := evaluate(input); err == nil {
		t.Fatal("expected unpaid converted arm to fail closed")
	}
}

func TestEvaluateVerifiedMechanismsAppliesReversalCeiling(t *testing.T) {
	input := scenario{
		Name: "verified", EvidenceKind: "verified_closed_pilot", MatureCohort: true,
		Handoffs: 30, Accepted: 21, Activated: 9, Converted: 3,
		MaxCostPerActivationCents: 10000, MaxMedianDaysToCharge: 14,
		MinPaidSettlementsPerMechanism: 1, MaxReversalRate: 0.05,
		PaidSettlements: 3, PaidByCurrency: map[string]int64{"usd": 7500},
		VerifiedMechanisms: map[string]mechanismEvidence{
			"accepted": {
				ObservedHandoffs: 10, Accepted: 7, Activated: 3, Converted: 1,
				Reversed: 1, PaidSettlements: 1, PaidCents: 2500, PaidMedianSeconds: 86400,
			},
			"activated": {
				ObservedHandoffs: 10, Accepted: 7, Activated: 3, Converted: 1,
				PaidSettlements: 1, PaidCents: 2500, PaidMedianSeconds: 86400,
			},
			"converted": {
				ObservedHandoffs: 10, Accepted: 7, Activated: 3, Converted: 1,
				PaidSettlements: 1, PaidCents: 2500, PaidMedianSeconds: 86400,
			},
		},
	}
	report, err := evaluate(input)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range report.Results {
		if candidate.ChargeEvent == "accepted" {
			if candidate.Viable || candidate.Failure != "exceeds reversal-rate ceiling" {
				t.Fatalf("accepted reversal gate not applied: %+v", candidate)
			}
			return
		}
	}
	t.Fatal("accepted result missing")
}
