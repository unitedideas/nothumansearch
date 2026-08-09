// Command provider-mechanism-model compares provider-funded outcome charge
// events without changing provider state or contacting an external service.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
)

type scenario struct {
	Name                           string                       `json:"name"`
	EvidenceKind                   string                       `json:"evidence_kind"`
	MatureCohort                   bool                         `json:"mature_cohort"`
	Handoffs                       int64                        `json:"handoffs"`
	Accepted                       int64                        `json:"accepted"`
	Activated                      int64                        `json:"activated"`
	Converted                      int64                        `json:"converted"`
	MaxCostPerActivationCents      int64                        `json:"max_cost_per_activation_cents"`
	MaxMedianDaysToCharge          float64                      `json:"max_median_days_to_charge"`
	MinChargedEvents               int64                        `json:"min_charged_events"`
	MinPaidSettlementsPerMechanism int64                        `json:"min_paid_settlements_per_mechanism,omitempty"`
	MaxReversalRate                float64                      `json:"max_reversal_rate,omitempty"`
	BountyPointsCents              []int64                      `json:"bounty_points_cents"`
	ChargeEvents                   []chargeEvent                `json:"charge_events"`
	PaidSettlements                int64                        `json:"paid_settlements,omitempty"`
	PaidByCurrency                 map[string]int64             `json:"paid_by_currency,omitempty"`
	PaidMedianDays                 float64                      `json:"paid_median_days,omitempty"`
	VerifiedMechanisms             map[string]mechanismEvidence `json:"verified_mechanisms,omitempty"`
}

type mechanismEvidence struct {
	ObservedHandoffs  int64 `json:"observed_handoffs"`
	Accepted          int64 `json:"accepted"`
	Activated         int64 `json:"activated"`
	Converted         int64 `json:"converted"`
	Reversed          int64 `json:"reversed"`
	PaidSettlements   int64 `json:"paid_settlements"`
	PaidCents         int64 `json:"paid_cents"`
	PaidMedianSeconds int64 `json:"paid_median_handoff_to_settlement_seconds"`
}

type chargeEvent struct {
	Name               string  `json:"name"`
	MedianDaysToCharge float64 `json:"median_days_to_charge"`
}

type result struct {
	ChargeEvent             string  `json:"charge_event"`
	BountyCents             int64   `json:"bounty_cents"`
	ChargedEvents           int64   `json:"charged_events"`
	GrossBillableCents      int64   `json:"gross_billable_cents"`
	RevenuePerHandoffCents  float64 `json:"revenue_per_handoff_cents"`
	CostPerActivationCents  float64 `json:"cost_per_activation_cents"`
	CostPerConversionCents  float64 `json:"cost_per_conversion_cents"`
	MedianDaysToCharge      float64 `json:"median_days_to_charge"`
	ValueKind               string  `json:"value_kind"`
	VerifiedPaidCents       int64   `json:"verified_paid_cents,omitempty"`
	VerifiedPaidSettlements int64   `json:"verified_paid_settlements,omitempty"`
	ReversalRate            float64 `json:"reversal_rate"`
	Viable                  bool    `json:"viable"`
	Failure                 string  `json:"failure,omitempty"`
}

type report struct {
	Contract                 string           `json:"contract"`
	Scenario                 string           `json:"scenario"`
	EvidenceKind             string           `json:"evidence_kind"`
	Synthetic                bool             `json:"synthetic"`
	SelectedEvent            string           `json:"selected_event,omitempty"`
	SelectedBountyCents      int64            `json:"selected_bounty_cents,omitempty"`
	SelectionReason          string           `json:"selection_reason"`
	Results                  []result         `json:"results"`
	CommercialProof          bool             `json:"commercial_proof"`
	CollectedRevenueEvidence bool             `json:"collected_revenue_evidence"`
	VerifiedPaidSettlements  int64            `json:"verified_paid_settlements,omitempty"`
	VerifiedPaidByCurrency   map[string]int64 `json:"verified_paid_by_currency,omitempty"`
	VerifiedPaidMedianDays   float64          `json:"verified_paid_median_days,omitempty"`
	ProductionChanged        bool             `json:"production_changed"`
}

type policy struct {
	Name                           string  `json:"name"`
	DemandTopic                    string  `json:"demand_topic"`
	MaxCostPerActivationCents      int64   `json:"max_cost_per_activation_cents"`
	MaxMedianDaysToCharge          float64 `json:"max_median_days_to_charge"`
	MinChargedEvents               int64   `json:"min_charged_events"`
	MinPaidSettlementsPerMechanism int64   `json:"min_paid_settlements_per_mechanism"`
	MaxReversalRate                float64 `json:"max_reversal_rate"`
	BountyPointsCents              []int64 `json:"bounty_points_cents"`
}

type proofStatusDocument struct {
	CommercialProof verifiedProof `json:"commercial_proof"`
}

type verifiedProof struct {
	PilotID                 string                       `json:"provider_pilot_epoch_id"`
	DemandTopic             string                       `json:"provider_pilot_demand_topic"`
	Status                  string                       `json:"provider_pilot_status"`
	OutcomeIntegrity        *bool                        `json:"outcome_receipt_integrity_valid"`
	RejectedOutcomes        *int64                       `json:"rejected_outcome_receipts"`
	RejectedLedger          *int64                       `json:"rejected_outcome_ledger_entries"`
	ObservedHandoffs        int64                        `json:"verified_observed_handoffs"`
	Accepted                int64                        `json:"verified_provider_accepted_handoffs"`
	Activated               int64                        `json:"verified_provider_confirmed_activations"`
	Converted               int64                        `json:"verified_provider_confirmed_conversions"`
	AcceptedLatencySamples  int64                        `json:"verified_accepted_latency_samples"`
	ActivatedLatencySamples int64                        `json:"verified_activated_latency_samples"`
	ConvertedLatencySamples int64                        `json:"verified_converted_latency_samples"`
	AcceptedMedianSeconds   int64                        `json:"verified_accepted_median_handoff_to_outcome_seconds"`
	ActivatedMedianSeconds  int64                        `json:"verified_activated_median_handoff_to_outcome_seconds"`
	ConvertedMedianSeconds  int64                        `json:"verified_converted_median_handoff_to_outcome_seconds"`
	PilotThresholdsMet      *bool                        `json:"pilot_thresholds_met"`
	OrganicRankSold         *bool                        `json:"organic_rank_sold"`
	RawQueriesSold          *bool                        `json:"raw_queries_sold"`
	RawPromptsSold          *bool                        `json:"raw_prompts_sold"`
	AgentIdentitiesSold     *bool                        `json:"agent_identities_sold"`
	PrincipalIdentitiesSold *bool                        `json:"principal_identities_sold"`
	SettlementIntegrity     *bool                        `json:"settlement_receipt_integrity_valid"`
	PaidSettlements         int64                        `json:"verified_provider_paid_settlements"`
	RejectedSettlements     *int64                       `json:"rejected_provider_settlement_receipts"`
	PaidLatencySamples      int64                        `json:"verified_paid_latency_samples"`
	PaidMedianSeconds       int64                        `json:"verified_paid_median_handoff_to_settlement_seconds"`
	PaidByCurrency          map[string]int64             `json:"verified_terms_paid_by_currency"`
	VerifiedMechanisms      map[string]mechanismEvidence `json:"verified_mechanisms"`
}

func main() {
	scenarioPath := flag.String("scenario", "", "path to a JSON scenario")
	proofPath := flag.String("proof-status", "", "path to projected closed-pilot proof JSON")
	policyPath := flag.String("policy", "", "path to a JSON decision policy for proof ingestion")
	flag.Parse()
	var input scenario
	var err error
	switch {
	case *scenarioPath != "" && *proofPath == "" && *policyPath == "":
		err = readJSON(*scenarioPath, &input, true)
	case *scenarioPath == "" && *proofPath != "" && *policyPath != "":
		var document proofStatusDocument
		var decisionPolicy policy
		if err = readJSON(*proofPath, &document, false); err == nil {
			err = readJSON(*policyPath, &decisionPolicy, true)
		}
		if err == nil {
			input, err = scenarioFromVerifiedProof(document.CommercialProof, decisionPolicy)
		}
	default:
		err = errors.New("use either -scenario or both -proof-status and -policy")
	}
	if err != nil {
		fatal(err)
	}
	output, err := evaluate(input)
	if err != nil {
		fatal(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fatal(err)
	}
}

func readJSON(path string, target any, strict bool) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	if strict {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("JSON input contains trailing data")
		}
		return err
	}
	return nil
}

func scenarioFromVerifiedProof(proof verifiedProof, policy policy) (scenario, error) {
	if proof.PilotID == "" || proof.Status != "closed" || proof.DemandTopic != policy.DemandTopic || policy.DemandTopic != "developer-tools" {
		return scenario{}, errors.New("proof must identify the closed developer-tools pilot required by policy")
	}
	if proof.OutcomeIntegrity == nil || !*proof.OutcomeIntegrity ||
		proof.RejectedOutcomes == nil || *proof.RejectedOutcomes != 0 ||
		proof.RejectedLedger == nil || *proof.RejectedLedger != 0 ||
		proof.PilotThresholdsMet == nil || !*proof.PilotThresholdsMet {
		return scenario{}, errors.New("proof must pass outcome integrity and the commercial pilot thresholds")
	}
	if proof.OrganicRankSold == nil || proof.RawQueriesSold == nil || proof.RawPromptsSold == nil ||
		proof.AgentIdentitiesSold == nil || proof.PrincipalIdentitiesSold == nil ||
		*proof.OrganicRankSold || *proof.RawQueriesSold || *proof.RawPromptsSold ||
		*proof.AgentIdentitiesSold || *proof.PrincipalIdentitiesSold {
		return scenario{}, errors.New("proof violates the free-organic privacy boundary")
	}
	if proof.SettlementIntegrity == nil || !*proof.SettlementIntegrity ||
		proof.RejectedSettlements == nil || *proof.RejectedSettlements != 0 ||
		proof.PaidSettlements < 1 || proof.PaidLatencySamples != proof.PaidSettlements ||
		proof.PaidMedianSeconds < 0 || len(proof.PaidByCurrency) != 1 || proof.PaidByCurrency["usd"] < 1 {
		return scenario{}, errors.New("proof must contain an integrity-valid paid terms settlement")
	}
	if proof.ObservedHandoffs < proof.Accepted || proof.Accepted < proof.Activated || proof.Activated < proof.Converted {
		return scenario{}, errors.New("verified proof contains an impossible funnel")
	}
	if proof.AcceptedLatencySamples != proof.Accepted || proof.ActivatedLatencySamples != proof.Activated || proof.ConvertedLatencySamples != proof.Converted {
		return scenario{}, errors.New("verified proof latency samples must cover every positive ticket")
	}
	if proof.AcceptedMedianSeconds < 0 || proof.ActivatedMedianSeconds < 0 || proof.ConvertedMedianSeconds < 0 {
		return scenario{}, errors.New("verified proof latencies cannot be negative")
	}
	return scenario{
		Name:                           policy.Name + ":" + proof.PilotID,
		EvidenceKind:                   "verified_closed_pilot",
		MatureCohort:                   true,
		Handoffs:                       proof.ObservedHandoffs,
		Accepted:                       proof.Accepted,
		Activated:                      proof.Activated,
		Converted:                      proof.Converted,
		MaxCostPerActivationCents:      policy.MaxCostPerActivationCents,
		MaxMedianDaysToCharge:          policy.MaxMedianDaysToCharge,
		MinChargedEvents:               policy.MinChargedEvents,
		MinPaidSettlementsPerMechanism: policy.MinPaidSettlementsPerMechanism,
		MaxReversalRate:                policy.MaxReversalRate,
		BountyPointsCents:              policy.BountyPointsCents,
		ChargeEvents: []chargeEvent{
			{Name: "accepted", MedianDaysToCharge: float64(proof.AcceptedMedianSeconds) / 86400},
			{Name: "activated", MedianDaysToCharge: float64(proof.ActivatedMedianSeconds) / 86400},
			{Name: "converted", MedianDaysToCharge: float64(proof.ConvertedMedianSeconds) / 86400},
		},
		PaidSettlements:    proof.PaidSettlements,
		PaidByCurrency:     proof.PaidByCurrency,
		PaidMedianDays:     float64(proof.PaidMedianSeconds) / 86400,
		VerifiedMechanisms: proof.VerifiedMechanisms,
	}, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func evaluate(input scenario) (report, error) {
	if input.EvidenceKind != "synthetic" && input.EvidenceKind != "verified_closed_pilot" {
		return report{}, errors.New("evidence_kind must be synthetic or verified_closed_pilot")
	}
	if input.EvidenceKind == "verified_closed_pilot" {
		return evaluateVerifiedMechanisms(input)
	}
	if !input.MatureCohort {
		return report{}, errors.New("mature_cohort must be true; incomplete downstream outcomes cannot select a charge event")
	}
	if input.Handoffs < input.Accepted || input.Accepted < input.Activated || input.Activated < input.Converted || input.Converted < 0 {
		return report{}, errors.New("funnel counts must satisfy handoffs >= accepted >= activated >= converted >= 0")
	}
	if input.Handoffs == 0 || input.Activated == 0 || input.Converted == 0 {
		return report{}, errors.New("handoffs, activated, and converted must be nonzero for comparable unit economics")
	}
	if input.MaxCostPerActivationCents < 1 || input.MaxMedianDaysToCharge < 0 || input.MinChargedEvents < 1 {
		return report{}, errors.New("selection constraints must be positive")
	}
	if len(input.ChargeEvents) != 3 {
		return report{}, errors.New("exactly one accepted, activated, and converted variant is required")
	}
	if len(input.BountyPointsCents) == 0 {
		return report{}, errors.New("at least one common bounty point is required")
	}

	counts := map[string]int64{"accepted": input.Accepted, "activated": input.Activated, "converted": input.Converted}
	seen := make(map[string]bool, 3)
	output := report{
		Contract:                 "nhs-provider-mechanism-model-v1",
		Scenario:                 input.Name,
		EvidenceKind:             input.EvidenceKind,
		Synthetic:                input.EvidenceKind == "synthetic",
		CommercialProof:          false,
		CollectedRevenueEvidence: input.EvidenceKind == "verified_closed_pilot" && input.PaidSettlements > 0,
		VerifiedPaidSettlements:  input.PaidSettlements,
		VerifiedPaidByCurrency:   input.PaidByCurrency,
		VerifiedPaidMedianDays:   input.PaidMedianDays,
		ProductionChanged:        false,
	}
	for _, event := range input.ChargeEvents {
		charged, ok := counts[event.Name]
		if !ok || seen[event.Name] {
			return report{}, errors.New("variants must contain accepted, activated, and converted exactly once")
		}
		seen[event.Name] = true
		if event.MedianDaysToCharge < 0 {
			return report{}, errors.New("charge latency cannot be negative")
		}
		for _, bounty := range input.BountyPointsCents {
			if bounty < 1 {
				return report{}, errors.New("bounty points must be positive")
			}
			gross := charged * bounty
			item := result{
				ChargeEvent:            event.Name,
				BountyCents:            bounty,
				ChargedEvents:          charged,
				GrossBillableCents:     gross,
				RevenuePerHandoffCents: float64(gross) / float64(input.Handoffs),
				CostPerActivationCents: float64(gross) / float64(input.Activated),
				CostPerConversionCents: float64(gross) / float64(input.Converted),
				MedianDaysToCharge:     event.MedianDaysToCharge,
				ValueKind:              "modeled_gross",
				Viable:                 true,
			}
			switch {
			case charged < input.MinChargedEvents:
				item.Viable = false
				item.Failure = "insufficient charged-event sample"
			case item.CostPerActivationCents > float64(input.MaxCostPerActivationCents):
				item.Viable = false
				item.Failure = "exceeds provider cost-per-activation ceiling"
			case event.MedianDaysToCharge > input.MaxMedianDaysToCharge:
				item.Viable = false
				item.Failure = "exceeds time-to-charge ceiling"
			}
			output.Results = append(output.Results, item)
		}
	}

	sort.Slice(output.Results, func(i, j int) bool {
		if output.Results[i].Viable != output.Results[j].Viable {
			return output.Results[i].Viable
		}
		if output.Results[i].GrossBillableCents == output.Results[j].GrossBillableCents {
			return output.Results[i].MedianDaysToCharge < output.Results[j].MedianDaysToCharge
		}
		return output.Results[i].GrossBillableCents > output.Results[j].GrossBillableCents
	})
	for _, candidate := range output.Results {
		if candidate.Viable {
			output.SelectedEvent = candidate.ChargeEvent
			output.SelectedBountyCents = candidate.BountyCents
			output.SelectionReason = "highest gross billable value among variants meeting the sample, provider cost-per-activation, and time-to-charge constraints"
			return output, nil
		}
	}
	output.SelectionReason = "no variant meets the declared selection constraints"
	return output, nil
}

func evaluateVerifiedMechanisms(input scenario) (report, error) {
	if !input.MatureCohort {
		return report{}, errors.New("mature_cohort must be true; incomplete downstream outcomes cannot select a charge event")
	}
	if input.MinPaidSettlementsPerMechanism < 1 || input.MaxCostPerActivationCents < 1 ||
		input.MaxMedianDaysToCharge < 0 || input.MaxReversalRate < 0 || input.MaxReversalRate > 1 {
		return report{}, errors.New("verified selection constraints must be positive")
	}
	expected := []string{"accepted", "activated", "converted"}
	if len(input.VerifiedMechanisms) != len(expected) {
		return report{}, errors.New("verified proof must contain accepted, activated, and converted mechanism evidence")
	}
	output := report{
		Contract:                 "nhs-provider-mechanism-model-v2",
		Scenario:                 input.Name,
		EvidenceKind:             input.EvidenceKind,
		Synthetic:                false,
		CommercialProof:          false,
		CollectedRevenueEvidence: input.PaidSettlements > 0,
		VerifiedPaidSettlements:  input.PaidSettlements,
		VerifiedPaidByCurrency:   input.PaidByCurrency,
		VerifiedPaidMedianDays:   input.PaidMedianDays,
		ProductionChanged:        false,
	}
	var mechanismPaidTotal int64
	var mechanismSettlementTotal int64
	var mechanismHandoffTotal, mechanismAcceptedTotal int64
	var mechanismActivatedTotal, mechanismConvertedTotal int64
	for _, chargeEvent := range expected {
		evidence, ok := input.VerifiedMechanisms[chargeEvent]
		if !ok || evidence.ObservedHandoffs < evidence.Accepted ||
			evidence.Accepted < evidence.Activated || evidence.Activated < evidence.Converted ||
			evidence.Reversed < 0 || evidence.Reversed > evidence.ObservedHandoffs ||
			evidence.ObservedHandoffs < 1 || evidence.Activated < 1 ||
			evidence.PaidSettlements < input.MinPaidSettlementsPerMechanism ||
			evidence.PaidCents < evidence.PaidSettlements || evidence.PaidMedianSeconds <= 0 {
			return report{}, fmt.Errorf("%s mechanism lacks mature real paid evidence", chargeEvent)
		}
		chargedEvents := map[string]int64{
			"accepted":  evidence.Accepted,
			"activated": evidence.Activated,
			"converted": evidence.Converted,
		}[chargeEvent]
		if evidence.PaidSettlements > chargedEvents {
			return report{}, fmt.Errorf("%s paid settlements exceed authenticated charged outcomes", chargeEvent)
		}
		mechanismPaidTotal += evidence.PaidCents
		mechanismSettlementTotal += evidence.PaidSettlements
		mechanismHandoffTotal += evidence.ObservedHandoffs
		mechanismAcceptedTotal += evidence.Accepted
		mechanismActivatedTotal += evidence.Activated
		mechanismConvertedTotal += evidence.Converted
		costPerActivation := float64(evidence.PaidCents) / float64(evidence.Activated)
		costPerConversion := float64(0)
		if evidence.Converted > 0 {
			costPerConversion = float64(evidence.PaidCents) / float64(evidence.Converted)
		}
		item := result{
			ChargeEvent:             chargeEvent,
			ChargedEvents:           chargedEvents,
			RevenuePerHandoffCents:  float64(evidence.PaidCents) / float64(evidence.ObservedHandoffs),
			CostPerActivationCents:  costPerActivation,
			CostPerConversionCents:  costPerConversion,
			MedianDaysToCharge:      float64(evidence.PaidMedianSeconds) / 86400,
			ValueKind:               "verified_paid",
			VerifiedPaidCents:       evidence.PaidCents,
			VerifiedPaidSettlements: evidence.PaidSettlements,
			ReversalRate:            float64(evidence.Reversed) / float64(evidence.ObservedHandoffs),
			Viable:                  true,
		}
		switch {
		case item.ReversalRate > input.MaxReversalRate:
			item.Viable = false
			item.Failure = "exceeds reversal-rate ceiling"
		case costPerActivation > float64(input.MaxCostPerActivationCents):
			item.Viable = false
			item.Failure = "exceeds provider cost-per-activation ceiling"
		case item.MedianDaysToCharge > input.MaxMedianDaysToCharge:
			item.Viable = false
			item.Failure = "exceeds time-to-paid-settlement ceiling"
		}
		output.Results = append(output.Results, item)
	}
	if mechanismPaidTotal != input.PaidByCurrency["usd"] ||
		mechanismSettlementTotal != input.PaidSettlements ||
		mechanismHandoffTotal != input.Handoffs || mechanismAcceptedTotal != input.Accepted ||
		mechanismActivatedTotal != input.Activated || mechanismConvertedTotal != input.Converted {
		return report{}, errors.New("mechanism settlement evidence does not reconcile to aggregate paid proof")
	}
	sort.Slice(output.Results, func(i, j int) bool {
		if output.Results[i].Viable != output.Results[j].Viable {
			return output.Results[i].Viable
		}
		if output.Results[i].RevenuePerHandoffCents == output.Results[j].RevenuePerHandoffCents {
			return output.Results[i].MedianDaysToCharge < output.Results[j].MedianDaysToCharge
		}
		return output.Results[i].RevenuePerHandoffCents > output.Results[j].RevenuePerHandoffCents
	})
	for _, candidate := range output.Results {
		if candidate.Viable {
			output.SelectedEvent = candidate.ChargeEvent
			output.SelectionReason = "highest verified paid revenue per observed handoff among mechanisms meeting provider cost-per-activation and time-to-paid-settlement constraints"
			return output, nil
		}
	}
	output.SelectionReason = "no mechanism meets the declared verified-paid selection constraints"
	return output, nil
}
