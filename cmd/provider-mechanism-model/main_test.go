package main

import "testing"

const (
	testProcessingBasisPoints           int64 = 290
	testProcessingFixedCents            int64 = 30
	testMinProcessingNetRate                  = 0.5
	testMinProcessingNetPerThousand     int64 = 1
	testMinProcessingNetLeadPerThousand int64 = 1
	testMinProcessingNetLeadRate              = 0.001
	testSelectionConfidenceLevel              = 0.95
	testMaxProcessorNetPerOfferCents    int64 = 1_000_000
)

func attachFixtureMoments(evidence mechanismEvidence) mechanismEvidence {
	if evidence.PaidSettlements == 0 {
		return evidence
	}
	grossQuotient, grossRemainder := evidence.PaidCents/evidence.PaidSettlements, evidence.PaidCents%evidence.PaidSettlements
	evidence.MaxBountyCents = grossQuotient
	if grossRemainder > 0 {
		evidence.MaxBountyCents++
	}
	netQuotient, netRemainder := evidence.ProcessorNetCents/evidence.PaidSettlements, evidence.ProcessorNetCents%evidence.PaidSettlements
	evidence.MaxProcessorNetCents = netQuotient
	if netRemainder > 0 {
		evidence.MaxProcessorNetCents++
	}
	evidence.ProcessorNetSumSquares = netRemainder*(netQuotient+1)*(netQuotient+1) +
		(evidence.PaidSettlements-netRemainder)*netQuotient*netQuotient
	return evidence
}

func attachActualProcessorNetScenario(input scenario) scenario {
	for name, evidence := range input.VerifiedMechanisms {
		if evidence.PaidSettlements == 0 {
			continue
		}
		fee := testPublishedProcessingFee(evidence.PaidCents, evidence.PaidSettlements)
		evidence.AvailableSettlements = evidence.PaidSettlements
		evidence.ProcessorFeeCents = fee
		evidence.ProcessorNetCents = evidence.PaidCents - fee
		evidence = attachFixtureMoments(evidence)
		input.VerifiedMechanisms[name] = evidence
	}
	return input
}

func attachActualProcessorNetProof(proof *verifiedProof) {
	var fees, net int64
	for name, evidence := range proof.VerifiedMechanisms {
		if evidence.PaidSettlements == 0 {
			continue
		}
		fee := testPublishedProcessingFee(evidence.PaidCents, evidence.PaidSettlements)
		evidence.AvailableSettlements = evidence.PaidSettlements
		evidence.ProcessorFeeCents = fee
		evidence.ProcessorNetCents = evidence.PaidCents - fee
		evidence = attachFixtureMoments(evidence)
		fees += fee
		net += evidence.ProcessorNetCents
		proof.VerifiedMechanisms[name] = evidence
	}
	truth := true
	zero := int64(0)
	proof.ProcessorNetIntegrity = &truth
	proof.RejectedProcessorNet = &zero
	proof.AvailableSettlements = proof.PaidSettlements
	proof.ProcessorFeesByCurrency = map[string]int64{"usd": fees}
	proof.ProcessorNetByCurrency = map[string]int64{"usd": net}
}

// testPublishedProcessingFee creates realistic exact processor values for
// fixtures. Production selection consumes Stripe-observed fee/net evidence;
// it never computes these values from a published rate.
func testPublishedProcessingFee(paidCents, paidSettlements int64) int64 {
	percentage := (paidCents*testProcessingBasisPoints + 9999) / 10000
	return percentage + paidSettlements*(testProcessingFixedCents+1)
}

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
		MechanismAssignmentContract: "nhs-provider-mechanism-arm-v1", MechanismObservationUnit: "returned_offer_opportunity_not_unique_agent",
		MechanismCountsAreUniqueAgents: &falsity,
		OutcomeIntegrity:               &truth, RejectedOutcomes: &zero, RejectedLedger: &zero,
		ObservedHandoffs: 50, Accepted: 30, Activated: 12, Converted: 3,
		AcceptedLatencySamples: 30, ActivatedLatencySamples: 12, ConvertedLatencySamples: 3,
		AcceptedMedianSeconds: 86400, ActivatedMedianSeconds: 604800, ConvertedMedianSeconds: 2592000,
		PilotThresholdsMet: &truth, OrganicRankSold: &falsity, RawQueriesSold: &falsity,
		RawPromptsSold: &falsity, AgentIdentitiesSold: &falsity, PrincipalIdentitiesSold: &falsity,
		SettlementIntegrity: &truth, PaidSettlements: 3, RejectedSettlements: &zero,
		PaidLatencySamples: 3, PaidMedianSeconds: 691200, PaidByCurrency: map[string]int64{"usd": 12500},
		VerifiedMechanisms: map[string]mechanismEvidence{
			"accepted": {
				ChargedProviderCompanies: 3, OfferReturns: 25, ObservedHandoffs: 20, Accepted: 15, Activated: 8, Converted: 1,
				PaidSettlements: 1, PaidCents: 2000, PaidMedianSeconds: 172800,
			},
			"activated": {
				ChargedProviderCompanies: 3, OfferReturns: 25, ObservedHandoffs: 20, Accepted: 10, Activated: 3, Converted: 1,
				PaidSettlements: 1, PaidCents: 7500, PaidMedianSeconds: 691200,
			},
			"converted": {
				ChargedProviderCompanies: 3, OfferReturns: 15, ObservedHandoffs: 10, Accepted: 5, Activated: 1, Converted: 1,
				PaidSettlements: 1, PaidCents: 3000, PaidMedianSeconds: 1036800,
			},
		},
	}
	attachActualProcessorNetProof(&proof)
	decision := policy{
		Name: "verified-policy", DemandTopic: "developer-tools", MaxCostPerActivationCents: 10000,
		MaxMedianDaysToCharge: 14, MinChargedEvents: 1, MinChargedProviderCompanies: 3,
		MinOfferReturnsPerMechanism: 1, MinPaidSettlementsPerMechanism: 1,
		MinProcessingNetMarginRate: testMinProcessingNetRate, MinProcessingNetPerThousand: testMinProcessingNetPerThousand,
		MinProcessingNetLeadPerThousand: testMinProcessingNetLeadPerThousand, MinProcessingNetLeadRate: testMinProcessingNetLeadRate,
		SelectionConfidenceLevel: testSelectionConfidenceLevel, MaxProcessorNetPerOfferCents: testMaxProcessorNetPerOfferCents,
	}
	input, err := scenarioFromVerifiedProof(proof, decision)
	if err != nil {
		t.Fatal(err)
	}
	report, err := evaluate(input)
	if err != nil {
		t.Fatal(err)
	}
	if report.Contract != "nhs-provider-mechanism-model-v5" || report.Synthetic ||
		report.EvidenceKind != "verified_closed_pilot" || report.SelectedEvent != "" ||
		!report.CollectedRevenueEvidence || report.VerifiedPaidSettlements != 3 ||
		report.Results[0].ChargeEvent != "activated" || report.Results[0].ValueKind != "verified_processor_net_available" ||
		report.RequiredNetLeadPerThousand != testMinProcessingNetLeadPerThousand ||
		report.RequiredNetLeadRate != testMinProcessingNetLeadRate || report.ConfidenceSeparated ||
		report.SelectionReason != "top mechanism is not separated from the runner-up at the declared simultaneous processing-net confidence level" {
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
	proof.MechanismCountsAreUniqueAgents = &truth
	if _, err := scenarioFromVerifiedProof(proof, decision); err == nil {
		t.Fatal("expected unique-agent confidence claim to fail closed")
	}
	proof.MechanismCountsAreUniqueAgents = &falsity
	proof.ProcessorFeesByCurrency["eur"] = 1
	if _, err := scenarioFromVerifiedProof(proof, decision); err == nil {
		t.Fatal("expected non-USD processor fee aggregate to fail closed")
	}
	delete(proof.ProcessorFeesByCurrency, "eur")
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
		MechanismAssignmentContract: "nhs-provider-mechanism-arm-v1", MechanismObservationUnit: "returned_offer_opportunity_not_unique_agent",
		MechanismCountsAreUniqueAgents: &falsity,
		OutcomeIntegrity:               &truth, RejectedOutcomes: &zero, RejectedLedger: &zero,
		ObservedHandoffs: 6, Accepted: 6, Activated: 3, Converted: 1,
		AcceptedLatencySamples: 6, ActivatedLatencySamples: 3, ConvertedLatencySamples: 1,
		AcceptedMedianSeconds: 3600, ActivatedMedianSeconds: 86400, ConvertedMedianSeconds: 604800,
		PilotThresholdsMet: &truth, OrganicRankSold: &falsity, RawQueriesSold: &falsity,
		RawPromptsSold: &falsity, AgentIdentitiesSold: &falsity, PrincipalIdentitiesSold: &falsity,
		SettlementIntegrity: &truth, PaidSettlements: 2, RejectedSettlements: &zero,
		PaidLatencySamples: 2, PaidMedianSeconds: 172800, PaidByCurrency: map[string]int64{"usd": 5000},
		VerifiedMechanisms: map[string]mechanismEvidence{
			"accepted":  {ChargedProviderCompanies: 1, OfferReturns: 2, ObservedHandoffs: 2, Accepted: 2, Activated: 1, PaidSettlements: 1, PaidCents: 2500, PaidMedianSeconds: 86400},
			"activated": {ChargedProviderCompanies: 1, OfferReturns: 2, ObservedHandoffs: 2, Accepted: 2, Activated: 1, PaidSettlements: 1, PaidCents: 2500, PaidMedianSeconds: 172800},
			"converted": {ChargedProviderCompanies: 1, OfferReturns: 2, ObservedHandoffs: 2, Accepted: 2, Activated: 1, Converted: 1},
		},
	}
	attachActualProcessorNetProof(&proof)
	decision := policy{
		Name: "verified-policy", DemandTopic: "developer-tools", MaxCostPerActivationCents: 10000,
		MaxMedianDaysToCharge: 14, MinChargedEvents: 1, MinChargedProviderCompanies: 1,
		MinOfferReturnsPerMechanism: 1, MinPaidSettlementsPerMechanism: 1,
		MinProcessingNetMarginRate: testMinProcessingNetRate, MinProcessingNetPerThousand: testMinProcessingNetPerThousand,
		MinProcessingNetLeadPerThousand: testMinProcessingNetLeadPerThousand, MinProcessingNetLeadRate: testMinProcessingNetLeadRate,
		SelectionConfidenceLevel: testSelectionConfidenceLevel, MaxProcessorNetPerOfferCents: testMaxProcessorNetPerOfferCents,
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
		MinChargedEvents: 1, MinChargedProviderCompanies: 3,
		MinOfferReturnsPerMechanism: 1, MinPaidSettlementsPerMechanism: 1, MaxReversalRate: 0.05,
		MinProcessingNetMarginRate: testMinProcessingNetRate, MinProcessingNetPerThousand: testMinProcessingNetPerThousand,
		MinProcessingNetLeadPerThousand: testMinProcessingNetLeadPerThousand, MinProcessingNetLeadRate: testMinProcessingNetLeadRate,
		SelectionConfidenceLevel: testSelectionConfidenceLevel, MaxProcessorNetPerOfferCents: testMaxProcessorNetPerOfferCents,
		PaidSettlements: 3, PaidByCurrency: map[string]int64{"usd": 7500},
		VerifiedMechanisms: map[string]mechanismEvidence{
			"accepted": {
				ChargedProviderCompanies: 3, OfferReturns: 10, ObservedHandoffs: 10, Accepted: 7, Activated: 3, Converted: 1,
				Reversed: 1, PaidSettlements: 1, PaidCents: 2500, PaidMedianSeconds: 86400,
			},
			"activated": {
				ChargedProviderCompanies: 3, OfferReturns: 10, ObservedHandoffs: 10, Accepted: 7, Activated: 3, Converted: 1,
				PaidSettlements: 1, PaidCents: 2500, PaidMedianSeconds: 86400,
			},
			"converted": {
				ChargedProviderCompanies: 3, OfferReturns: 10, ObservedHandoffs: 10, Accepted: 7, Activated: 3, Converted: 1,
				PaidSettlements: 1, PaidCents: 2500, PaidMedianSeconds: 86400,
			},
		},
	}
	report, err := evaluate(attachActualProcessorNetScenario(input))
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

func TestEvaluateVerifiedMechanismsAppliesChargedEventSamplePerArm(t *testing.T) {
	input := scenario{
		Name: "verified", EvidenceKind: "verified_closed_pilot", MatureCohort: true,
		Handoffs: 30, Accepted: 21, Activated: 9, Converted: 3,
		MaxCostPerActivationCents: 10000, MaxMedianDaysToCharge: 14,
		MinChargedEvents: 5, MinChargedProviderCompanies: 3,
		MinOfferReturnsPerMechanism: 1, MinPaidSettlementsPerMechanism: 1, MaxReversalRate: 0.2,
		MinProcessingNetMarginRate: testMinProcessingNetRate, MinProcessingNetPerThousand: testMinProcessingNetPerThousand,
		MinProcessingNetLeadPerThousand: testMinProcessingNetLeadPerThousand, MinProcessingNetLeadRate: testMinProcessingNetLeadRate,
		SelectionConfidenceLevel: testSelectionConfidenceLevel, MaxProcessorNetPerOfferCents: testMaxProcessorNetPerOfferCents,
		PaidSettlements: 3, PaidByCurrency: map[string]int64{"usd": 7500},
		VerifiedMechanisms: map[string]mechanismEvidence{
			"accepted": {
				ChargedProviderCompanies: 3, OfferReturns: 10, ObservedHandoffs: 10, Accepted: 7, Activated: 3, Converted: 1,
				PaidSettlements: 1, PaidCents: 1000, PaidMedianSeconds: 86400,
			},
			"activated": {
				ChargedProviderCompanies: 3, OfferReturns: 10, ObservedHandoffs: 10, Accepted: 7, Activated: 3, Converted: 1,
				PaidSettlements: 1, PaidCents: 2500, PaidMedianSeconds: 86400,
			},
			"converted": {
				ChargedProviderCompanies: 3, OfferReturns: 10, ObservedHandoffs: 10, Accepted: 7, Activated: 3, Converted: 1,
				PaidSettlements: 1, PaidCents: 4000, PaidMedianSeconds: 86400,
			},
		},
	}
	report, err := evaluate(attachActualProcessorNetScenario(input))
	if err != nil {
		t.Fatal(err)
	}
	if report.SelectedEvent != "" || report.SelectionReason != "mechanism comparison incomplete because at least one arm lacks the declared offer-return sample, provider coverage, or charged-event sample" {
		t.Fatalf("under-sampled comparison selected a winner: %+v", report)
	}
	for _, candidate := range report.Results {
		if candidate.ChargeEvent == "accepted" {
			if !candidate.Viable {
				t.Fatalf("accepted arm should meet sample gate: %+v", candidate)
			}
			continue
		}
		if candidate.Viable || candidate.Failure != "insufficient charged-event sample" {
			t.Fatalf("under-sampled %s arm did not fail closed: %+v", candidate.ChargeEvent, candidate)
		}
	}
}

func TestEvaluateVerifiedMechanismsRequiresEveryProviderInEveryArm(t *testing.T) {
	input := scenario{
		Name: "verified", EvidenceKind: "verified_closed_pilot", MatureCohort: true,
		Handoffs: 30, Accepted: 21, Activated: 15, Converted: 6,
		MaxCostPerActivationCents: 10000, MaxMedianDaysToCharge: 14,
		MinChargedEvents: 1, MinChargedProviderCompanies: 3,
		MinOfferReturnsPerMechanism: 1, MinPaidSettlementsPerMechanism: 1, MaxReversalRate: 0.2,
		MinProcessingNetMarginRate: testMinProcessingNetRate, MinProcessingNetPerThousand: testMinProcessingNetPerThousand,
		MinProcessingNetLeadPerThousand: testMinProcessingNetLeadPerThousand, MinProcessingNetLeadRate: testMinProcessingNetLeadRate,
		SelectionConfidenceLevel: testSelectionConfidenceLevel, MaxProcessorNetPerOfferCents: testMaxProcessorNetPerOfferCents,
		PaidSettlements: 3, PaidByCurrency: map[string]int64{"usd": 7500},
		VerifiedMechanisms: map[string]mechanismEvidence{
			"accepted": {
				ChargedProviderCompanies: 3, OfferReturns: 10, ObservedHandoffs: 10, Accepted: 7, Activated: 5, Converted: 2,
				PaidSettlements: 1, PaidCents: 1000, PaidMedianSeconds: 86400,
			},
			"activated": {
				ChargedProviderCompanies: 2, OfferReturns: 10, ObservedHandoffs: 10, Accepted: 7, Activated: 5, Converted: 2,
				PaidSettlements: 1, PaidCents: 5000, PaidMedianSeconds: 86400,
			},
			"converted": {
				ChargedProviderCompanies: 3, OfferReturns: 10, ObservedHandoffs: 10, Accepted: 7, Activated: 5, Converted: 2,
				PaidSettlements: 1, PaidCents: 1500, PaidMedianSeconds: 86400,
			},
		},
	}
	report, err := evaluate(attachActualProcessorNetScenario(input))
	if err != nil {
		t.Fatal(err)
	}
	if report.SelectedEvent != "" {
		t.Fatalf("provider-confounded comparison selected %q", report.SelectedEvent)
	}
	for _, candidate := range report.Results {
		if candidate.ChargeEvent == "activated" {
			if candidate.Viable || candidate.Failure != "insufficient provider coverage" {
				t.Fatalf("provider coverage gate not applied: %+v", candidate)
			}
			return
		}
	}
	t.Fatal("activated result missing")
}

func TestEvaluateVerifiedMechanismsSelectsRevenuePerReturnedOffer(t *testing.T) {
	input := scenario{
		Name: "verified", EvidenceKind: "verified_closed_pilot", MatureCohort: true,
		Handoffs: 20, Accepted: 15, Activated: 9, Converted: 4,
		MaxCostPerActivationCents: 10000, MaxMedianDaysToCharge: 14,
		MinChargedEvents: 1, MinChargedProviderCompanies: 3,
		MinOfferReturnsPerMechanism: 1, MinPaidSettlementsPerMechanism: 1,
		MinProcessingNetMarginRate: testMinProcessingNetRate, MinProcessingNetPerThousand: testMinProcessingNetPerThousand,
		MinProcessingNetLeadPerThousand: testMinProcessingNetLeadPerThousand, MinProcessingNetLeadRate: testMinProcessingNetLeadRate,
		SelectionConfidenceLevel: testSelectionConfidenceLevel, MaxProcessorNetPerOfferCents: testMaxProcessorNetPerOfferCents,
		MaxReversalRate: 0.2, PaidSettlements: 3,
		PaidByCurrency: map[string]int64{"usd": 8000},
		VerifiedMechanisms: map[string]mechanismEvidence{
			"accepted": {
				ChargedProviderCompanies: 3, OfferReturns: 10, ObservedHandoffs: 10,
				Accepted: 7, Activated: 3, Converted: 1,
				PaidSettlements: 1, PaidCents: 3000, PaidMedianSeconds: 86400,
			},
			"activated": {
				ChargedProviderCompanies: 3, OfferReturns: 20, ObservedHandoffs: 5,
				Accepted: 4, Activated: 3, Converted: 1,
				PaidSettlements: 1, PaidCents: 4000, PaidMedianSeconds: 86400,
			},
			"converted": {
				ChargedProviderCompanies: 3, OfferReturns: 10, ObservedHandoffs: 5,
				Accepted: 4, Activated: 3, Converted: 2,
				PaidSettlements: 1, PaidCents: 1000, PaidMedianSeconds: 86400,
			},
		},
	}
	report, err := evaluate(attachActualProcessorNetScenario(input))
	if err != nil {
		t.Fatal(err)
	}
	if report.SelectedEvent != "" || report.Results[0].ChargeEvent != "accepted" ||
		report.SelectionReason != "top mechanism is not separated from the runner-up at the declared simultaneous processing-net confidence level" {
		t.Fatalf("small-sample report did not retain accepted as the unselected point-estimate leader: %+v", report)
	}
	if report.Results[0].RevenuePerOfferReturnCents != 300 ||
		report.Results[0].RevenuePerHandoffCents != 300 {
		t.Fatalf("unexpected top-arm yields: %+v", report.Results[0])
	}
	for _, candidate := range report.Results {
		if candidate.ChargeEvent == "activated" && candidate.RevenuePerHandoffCents != 800 {
			t.Fatalf("fixture no longer proves handoff-only metric would disagree: %+v", candidate)
		}
	}
}

func TestEvaluateVerifiedMechanismsSelectsProcessingNetValue(t *testing.T) {
	input := scenario{
		Name: "verified", EvidenceKind: "verified_closed_pilot", MatureCohort: true,
		Handoffs: 30, Accepted: 21, Activated: 9, Converted: 3,
		MaxCostPerActivationCents: 10000, MaxMedianDaysToCharge: 14,
		MinChargedEvents: 1, MinChargedProviderCompanies: 3,
		MinOfferReturnsPerMechanism: 1, MinPaidSettlementsPerMechanism: 1,
		MaxReversalRate: 0.2, PaidSettlements: 7,
		PaidByCurrency:                  map[string]int64{"usd": 6900},
		MinProcessingNetMarginRate:      testMinProcessingNetRate,
		MinProcessingNetPerThousand:     testMinProcessingNetPerThousand,
		MinProcessingNetLeadPerThousand: testMinProcessingNetLeadPerThousand,
		MinProcessingNetLeadRate:        testMinProcessingNetLeadRate,
		SelectionConfidenceLevel:        testSelectionConfidenceLevel,
		MaxProcessorNetPerOfferCents:    testMaxProcessorNetPerOfferCents,
		VerifiedMechanisms: map[string]mechanismEvidence{
			"accepted": {
				ChargedProviderCompanies: 3, OfferReturns: 10, ObservedHandoffs: 10,
				Accepted: 7, Activated: 3, Converted: 1,
				PaidSettlements: 5, PaidCents: 3000, PaidMedianSeconds: 86400,
			},
			"activated": {
				ChargedProviderCompanies: 3, OfferReturns: 10, ObservedHandoffs: 10,
				Accepted: 7, Activated: 3, Converted: 1,
				PaidSettlements: 1, PaidCents: 2900, PaidMedianSeconds: 86400,
			},
			"converted": {
				ChargedProviderCompanies: 3, OfferReturns: 10, ObservedHandoffs: 10,
				Accepted: 7, Activated: 3, Converted: 1,
				PaidSettlements: 1, PaidCents: 1000, PaidMedianSeconds: 86400,
			},
		},
	}
	report, err := evaluate(attachActualProcessorNetScenario(input))
	if err != nil {
		t.Fatal(err)
	}
	if report.SelectedEvent != "" || report.Results[0].ChargeEvent != "activated" ||
		report.SelectionReason != "top mechanism is not separated from the runner-up at the declared simultaneous processing-net confidence level" {
		t.Fatalf("small-sample report did not retain activated as the unselected processing-net leader: %+v", report)
	}
	if report.Results[0].RevenuePerOfferReturnCents != 290 ||
		report.Results[0].ActualProcessorFeeCents != 116 ||
		report.Results[0].ActualProcessorNetCents != 2784 ||
		report.Results[0].ProcessingNetPerThousand != 278400 {
		t.Fatalf("unexpected activated processing-net economics: %+v", report.Results[0])
	}
	for _, candidate := range report.Results {
		if candidate.ChargeEvent == "accepted" && candidate.RevenuePerOfferReturnCents != 300 {
			t.Fatalf("fixture no longer proves gross revenue would disagree: %+v", candidate)
		}
	}

	for _, gate := range []struct {
		name         string
		absoluteLead int64
		relativeLead float64
	}{
		{name: "absolute", absoluteLead: 3000, relativeLead: 0.001},
		{name: "relative", absoluteLead: 2500, relativeLead: 0.2},
	} {
		t.Run("near tie fails "+gate.name+" lead floor", func(t *testing.T) {
			input.MinProcessingNetLeadPerThousand = gate.absoluteLead
			input.MinProcessingNetLeadRate = gate.relativeLead
			report, err = evaluate(attachActualProcessorNetScenario(input))
			if err != nil {
				t.Fatal(err)
			}
			if report.SelectedEvent != "" ||
				report.SelectionReason != "top mechanism lead is not economically decisive under the declared absolute and relative processing-net lead floors" ||
				report.ObservedNetLeadPerThousand != 2600 {
				t.Fatalf("near-tie comparison selected or misreported a winner: %+v", report)
			}
		})
	}

	input.MinProcessingNetLeadPerThousand = testMinProcessingNetLeadPerThousand
	input.MinProcessingNetLeadRate = testMinProcessingNetLeadRate
	input.MinProcessingNetPerThousand = 280000
	report, err = evaluate(attachActualProcessorNetScenario(input))
	if err != nil {
		t.Fatal(err)
	}
	if report.SelectedEvent != "" {
		t.Fatalf("processing-net floor selected %q", report.SelectedEvent)
	}
	for _, candidate := range report.Results {
		if candidate.Viable || candidate.Failure != "below processing-net revenue-per-return floor" {
			t.Fatalf("processing-net floor not applied to %s: %+v", candidate.ChargeEvent, candidate)
		}
	}
}

func TestEvaluateVerifiedMechanismsRequiresAndAcceptsConfidenceSeparation(t *testing.T) {
	evidence := func(paidSettlements int64) mechanismEvidence {
		const netPerSettlement = int64(100)
		return mechanismEvidence{
			ChargedProviderCompanies: 3,
			OfferReturns:             10000,
			MaxBountyCents:           netPerSettlement,
			ObservedHandoffs:         8000,
			Accepted:                 7000,
			Activated:                5000,
			Converted:                3000,
			PaidSettlements:          paidSettlements,
			AvailableSettlements:     paidSettlements,
			PaidCents:                paidSettlements * netPerSettlement,
			ProcessorNetCents:        paidSettlements * netPerSettlement,
			ProcessorNetSumSquares:   paidSettlements * netPerSettlement * netPerSettlement,
			MaxProcessorNetCents:     netPerSettlement,
			PaidMedianSeconds:        86400,
		}
	}
	input := scenario{
		Name: "confidence-separated", EvidenceKind: "verified_closed_pilot", MatureCohort: true,
		Handoffs: 24000, Accepted: 21000, Activated: 15000, Converted: 9000,
		MaxCostPerActivationCents: 10000, MaxMedianDaysToCharge: 14,
		MinChargedEvents: 5, MinChargedProviderCompanies: 3,
		MinOfferReturnsPerMechanism: 20, MinPaidSettlementsPerMechanism: 1,
		MaxReversalRate: 0.2, MinProcessingNetMarginRate: 0.5,
		MinProcessingNetPerThousand: 1000, MinProcessingNetLeadPerThousand: 1157,
		MinProcessingNetLeadRate: 0.2, SelectionConfidenceLevel: 0.95,
		MaxProcessorNetPerOfferCents: 100,
		PaidSettlements:              15000, PaidByCurrency: map[string]int64{"usd": 1500000},
		VerifiedMechanisms: map[string]mechanismEvidence{
			"accepted":  evidence(7000),
			"activated": evidence(5000),
			"converted": evidence(3000),
		},
	}
	report, err := evaluate(input)
	if err != nil {
		t.Fatal(err)
	}
	if report.SelectedEvent != "accepted" || !report.ConfidenceSeparated ||
		report.Results[0].ProcessingNetLowerPerThousand <= report.Results[1].ProcessingNetUpperPerThousand {
		t.Fatalf("separated evidence did not select accepted: %+v", report)
	}
}
