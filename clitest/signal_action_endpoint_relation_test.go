package clitest

import (
	"strconv"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestSignalActionEndpointRelationAcrossObservations(t *testing.T) {
	t.Parallel()

	rootID := cli.CommandID(1)
	policyID := cli.SignalPolicyID(41001)
	endpointID := cli.SignalEndpointID(42002)
	otherEndpointID := cli.SignalEndpointID(43003)
	policy := cli.SignalPolicySchema{
		ID: policyID,
		Actions: []cli.SignalAction{
			{EndpointID: endpointID},
			{EndpointID: otherEndpointID},
		},
	}
	endpoint := cli.SignalEndpoint{
		ID:       endpointID,
		Capacity: 64,
		Origin:   cli.DeclarationOriginApplication,
	}
	otherEndpoint := cli.SignalEndpoint{
		ID:       otherEndpointID,
		Capacity: 32,
		Origin:   cli.DeclarationOriginProvider,
	}

	schema := cli.Schema{
		Root: rootID,
		Commands: map[cli.CommandID]cli.CommandSchema{
			rootID: {
				ID:             rootID,
				SignalPolicies: []cli.SignalPolicyID{policyID},
			},
		},
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicySchema{policyID: policy},
		SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpoint{
			endpointID:      endpoint,
			otherEndpointID: otherEndpoint,
		},
	}
	reference := cli.Reference{
		Root: rootID,
		Commands: map[cli.CommandID]cli.CommandReference{
			rootID: {
				ID:             rootID,
				SignalPolicies: []cli.SignalPolicyID{policyID},
			},
		},
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyReference{
			policyID: {SignalPolicySchema: policy},
		},
		SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpoint{
			endpointID:      endpoint,
			otherEndpointID: otherEndpoint,
		},
	}
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			rootID: {
				ID:             rootID,
				SignalPolicies: []cli.SignalPolicyID{policyID},
			},
		},
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyInspection{
			policyID: {
				ID:      policyID,
				Actions: policy.Actions,
			},
		},
		SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpointInspection{
			endpointID:      {SignalEndpoint: endpoint},
			otherEndpointID: {SignalEndpoint: otherEndpoint},
		},
		ExecutionAdmission: cli.ExecutionAdmissionInspection{
			SignalEndpoints: []cli.SignalEndpointID{otherEndpointID},
		},
	}

	relation := RootCommand().SignalPolicyAt(0).ActionAt(0).SignalEndpoint()
	if relation.state == nil || relation.state.mode != signalEndpointLocatorAction ||
		relation.state.id != 0 {
		t.Fatalf("relation locator stores observed endpoint identity: %#v", relation.state)
	}

	for name, outcome := range map[string]evaluationResult{
		"schema capacity":     schemaExpectationOutcome(t, relation.Capacity(64), &schema),
		"schema origin":       schemaExpectationOutcome(t, relation.Origin(cli.DeclarationOriginApplication), &schema),
		"reference capacity":  referenceExpectationOutcome(t, relation.Capacity(64), reference),
		"reference origin":    referenceExpectationOutcome(t, relation.Origin(cli.DeclarationOriginApplication), reference),
		"inspection capacity": inspectionExpectationOutcome(t, relation.Capacity(64), inspection),
		"inspection status":   inspectionExpectationOutcome(t, relation.Status(cli.SignalEndpointStatusUnknown), inspection),
	} {
		if outcome.status != evaluationProved {
			t.Fatalf("%s = %#v", name, outcome)
		}
	}
	if relation.state.mode != signalEndpointLocatorAction || relation.state.id != 0 {
		t.Fatalf("relation locator mutated after evaluation: %#v", relation.state)
	}

	second := RootCommand().SignalPolicyAt(0).ActionAt(1).SignalEndpoint()
	for name, outcome := range map[string]evaluationResult{
		"schema selected action":     schemaExpectationOutcome(t, second.Capacity(32), &schema),
		"reference selected action":  referenceExpectationOutcome(t, second.Capacity(32), reference),
		"inspection selected action": inspectionExpectationOutcome(t, second.Capacity(32), inspection),
	} {
		if outcome.status != evaluationProved {
			t.Fatalf("%s = %#v", name, outcome)
		}
	}
}

func TestSignalActionEndpointRelationAbsenceAmbiguityAndTruncation(t *testing.T) {
	t.Parallel()

	rootID := cli.CommandID(1)
	policyID := cli.SignalPolicyID(51001)
	endpointID := cli.SignalEndpointID(52002)
	relation := RootCommand().SignalPolicyAt(0).ActionAt(0).SignalEndpoint()
	baseCommand := cli.CommandSchema{
		ID:             rootID,
		SignalPolicies: []cli.SignalPolicyID{policyID},
	}
	basePolicy := cli.SignalPolicySchema{
		ID:      policyID,
		Actions: []cli.SignalAction{{EndpointID: endpointID}},
	}

	t.Run("no relation is absent even when ID zero exists", func(t *testing.T) {
		t.Parallel()
		zeroPolicy := basePolicy
		zeroPolicy.Actions = []cli.SignalAction{{EndpointID: 0}}
		schema := cli.Schema{
			Root:           rootID,
			Commands:       map[cli.CommandID]cli.CommandSchema{rootID: baseCommand},
			SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicySchema{policyID: zeroPolicy},
			SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpoint{0: {
				ID:       0,
				Capacity: 64,
			}},
		}
		if got := schemaExpectationOutcome(t, relation.DoesNotExist(), &schema); got.status != evaluationProved {
			t.Fatalf("zero relation absence = %#v", got)
		}

		inspection := cli.Inspection{
			Commands: map[cli.CommandID]cli.CommandInspection{
				rootID: {
					ID:             rootID,
					SignalPolicies: []cli.SignalPolicyID{policyID},
				},
			},
			SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyInspection{
				policyID: {
					ID:      policyID,
					Actions: zeroPolicy.Actions,
				},
			},
			SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpointInspection{
				0: {SignalEndpoint: cli.SignalEndpoint{
					ID:       0,
					Capacity: 64,
				}},
			},
			Truncated: true,
		}
		if got := inspectionExpectationOutcome(
			t,
			relation.DoesNotExist(),
			inspection,
		); got.status != evaluationProved {
			t.Fatalf("observed zero relation under truncation = %#v", got)
		}
	})

	t.Run("related target absent from complete static observations", func(t *testing.T) {
		t.Parallel()
		schema := cli.Schema{
			Root:           rootID,
			Commands:       map[cli.CommandID]cli.CommandSchema{rootID: baseCommand},
			SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicySchema{policyID: basePolicy},
			SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpoint{53003: {
				ID:       53003,
				Capacity: 64,
			}},
		}
		if got := schemaExpectationOutcome(
			t,
			relation.Capacity(64),
			&schema,
		); got.status != evaluationDisproved ||
			!strings.Contains(renderEvaluationField(got.observed), "related signal endpoint is absent") {
			t.Fatalf("schema missing target = %#v", got)
		}
		if got := schemaExpectationOutcome(t, relation.DoesNotExist(), &schema); got.status != evaluationProved {
			t.Fatalf("schema missing target existence = %#v", got)
		}

		reference := cli.Reference{
			Root: rootID,
			Commands: map[cli.CommandID]cli.CommandReference{
				rootID: {
					ID:             rootID,
					SignalPolicies: []cli.SignalPolicyID{policyID},
				},
			},
			SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyReference{
				policyID: {SignalPolicySchema: basePolicy},
			},
			SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpoint{53003: {
				ID:       53003,
				Capacity: 64,
			}},
		}
		if got := referenceExpectationOutcome(
			t,
			relation.Capacity(64),
			reference,
		); got.status != evaluationDisproved ||
			!strings.Contains(renderEvaluationField(got.observed), "related signal endpoint is absent") {
			t.Fatalf("reference missing target = %#v", got)
		}
	})

	t.Run("truncated inspection missing target is not provable", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Commands: map[cli.CommandID]cli.CommandInspection{
				rootID: {
					ID:             rootID,
					SignalPolicies: []cli.SignalPolicyID{policyID},
				},
			},
			SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyInspection{
				policyID: {
					ID:      policyID,
					Actions: basePolicy.Actions,
				},
			},
			Truncated: true,
		}
		if got := inspectionExpectationOutcome(
			t,
			relation.Capacity(64),
			inspection,
		); got.status != evaluationNotProvable {
			t.Fatalf("truncated capacity = %#v", got)
		}
		if got := inspectionExpectationOutcome(
			t,
			relation.DoesNotExist(),
			inspection,
		); got.status != evaluationNotProvable {
			t.Fatalf("truncated absence = %#v", got)
		}
	})

	t.Run("parent missing and ambiguous statuses propagate", func(t *testing.T) {
		t.Parallel()
		missing := cli.Schema{
			Root:     rootID,
			Commands: map[cli.CommandID]cli.CommandSchema{rootID: {ID: rootID}},
		}
		if resolved := resolveSchemaSignalEndpointLocator(relation.state, &missing); resolved.status != locatorResolutionAbsent {
			status := resolved.status
			t.Fatalf("missing parent status = %v", status)
		}

		childA, childB := cli.CommandID(2), cli.CommandID(3)
		ambiguousRelation := RootCommand().Command("dup").SignalPolicyAt(0).ActionAt(0).SignalEndpoint()
		ambiguous := cli.Schema{
			Root: rootID,
			Commands: map[cli.CommandID]cli.CommandSchema{
				rootID: {
					ID:       rootID,
					Children: []cli.CommandID{childA, childB},
				},
				childA: {
					ID:             childA,
					Name:           "dup",
					SignalPolicies: []cli.SignalPolicyID{policyID},
				},
				childB: {
					ID:             childB,
					Name:           "dup",
					SignalPolicies: []cli.SignalPolicyID{policyID},
				},
			},
			SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicySchema{policyID: basePolicy},
		}
		if resolved := resolveSchemaSignalEndpointLocator(ambiguousRelation.state, &ambiguous); resolved.status != locatorResolutionAmbiguous {
			status := resolved.status
			t.Fatalf("schema ambiguous parent status = %v", status)
		}
		if got := schemaExpectationOutcome(
			t,
			ambiguousRelation.DoesNotExist(),
			&ambiguous,
		); got.status != evaluationDisproved {
			t.Fatalf("schema ambiguous parent negative existence = %#v", got)
		}

		reference := cli.Reference{
			Root: rootID,
			Commands: map[cli.CommandID]cli.CommandReference{
				rootID: {
					ID:       rootID,
					Children: []cli.CommandID{childA, childB},
				},
				childA: {
					ID:             childA,
					Name:           "dup",
					SignalPolicies: []cli.SignalPolicyID{policyID},
				},
				childB: {
					ID:             childB,
					Name:           "dup",
					SignalPolicies: []cli.SignalPolicyID{policyID},
				},
			},
			SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyReference{
				policyID: {SignalPolicySchema: basePolicy},
			},
		}
		if resolved := resolveReferenceSignalEndpointLocator(ambiguousRelation.state, reference); resolved.status != locatorResolutionAmbiguous {
			status := resolved.status
			t.Fatalf("reference ambiguous parent status = %v", status)
		}
		if got := referenceExpectationOutcome(
			t,
			ambiguousRelation.DoesNotExist(),
			reference,
		); got.status != evaluationDisproved {
			t.Fatalf("reference ambiguous parent negative existence = %#v", got)
		}

		inspection := cli.Inspection{
			Commands: map[cli.CommandID]cli.CommandInspection{
				rootID: {
					ID:       rootID,
					Children: []cli.CommandID{childA, childB},
					Path:     []string{},
				},
				childA: {
					ID:             childA,
					Name:           "dup",
					Path:           []string{"dup"},
					SignalPolicies: []cli.SignalPolicyID{policyID},
				},
				childB: {
					ID:             childB,
					Name:           "dup",
					Path:           []string{"dup"},
					SignalPolicies: []cli.SignalPolicyID{policyID},
				},
			},
			SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyInspection{
				policyID: {
					ID:      policyID,
					Actions: basePolicy.Actions,
				},
			},
		}
		if resolved := resolveInspectionSignalEndpointLocator(ambiguousRelation.state, inspection); resolved.status != locatorResolutionAmbiguous {
			status := resolved.status
			t.Fatalf("inspection ambiguous parent status = %v", status)
		}
		if got := inspectionExpectationOutcome(
			t,
			ambiguousRelation.DoesNotExist(),
			inspection,
		); got.status != evaluationDisproved {
			t.Fatalf("inspection ambiguous parent negative existence = %#v", got)
		}
	})
}

func TestSignalActionEndpointRelationFailureOutputSuppressesSignalIDs(t *testing.T) {
	t.Parallel()

	rootID := cli.CommandID(1)
	policyID := cli.SignalPolicyID(61001)
	endpointID := cli.SignalEndpointID(62002)
	relation := RootCommand().SignalPolicyAt(0).ActionAt(0).SignalEndpoint()
	baseCommand := cli.CommandSchema{
		ID:             rootID,
		SignalPolicies: []cli.SignalPolicyID{policyID},
	}
	basePolicy := cli.SignalPolicySchema{
		ID:      policyID,
		Actions: []cli.SignalAction{{EndpointID: endpointID}},
	}
	mismatch := cli.Schema{
		Root:           rootID,
		Commands:       map[cli.CommandID]cli.CommandSchema{rootID: baseCommand},
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicySchema{policyID: basePolicy},
		SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpoint{
			endpointID: {
				ID:       endpointID,
				Capacity: 32,
			},
		},
	}
	missingTarget := mismatch
	missingTarget.SignalEndpoints = nil
	truncated := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			rootID: {
				ID:             rootID,
				SignalPolicies: []cli.SignalPolicyID{policyID},
			},
		},
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyInspection{
			policyID: {
				ID:      policyID,
				Actions: basePolicy.Actions,
			},
		},
		Truncated: true,
	}
	childA, childB := cli.CommandID(2), cli.CommandID(3)
	ambiguousRelation := RootCommand().Command("dup").SignalPolicyAt(0).ActionAt(0).SignalEndpoint()
	ambiguous := cli.Schema{
		Root: rootID,
		Commands: map[cli.CommandID]cli.CommandSchema{
			rootID: {
				ID:       rootID,
				Children: []cli.CommandID{childA, childB},
			},
			childA: {
				ID:             childA,
				Name:           "dup",
				SignalPolicies: []cli.SignalPolicyID{policyID},
			},
			childB: {
				ID:             childB,
				Name:           "dup",
				SignalPolicies: []cli.SignalPolicyID{policyID},
			},
		},
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicySchema{policyID: basePolicy},
	}

	outcomes := map[string]evaluationResult{
		"property mismatch": schemaExpectationOutcome(t, relation.Capacity(64), &mismatch),
		"missing target":    schemaExpectationOutcome(t, relation.Capacity(64), &missingTarget),
		"ambiguous parent":  schemaExpectationOutcome(t, ambiguousRelation.Capacity(64), &ambiguous),
		"truncated target":  inspectionExpectationOutcome(t, relation.Capacity(64), truncated),
	}
	for name, outcome := range outcomes {
		if outcome.status == evaluationProved {
			t.Fatalf("%s unexpectedly satisfied: %#v", name, outcome)
		}
		serialized := renderEvaluationField(outcome.subject) + " " + renderEvaluationField(outcome.expected) + " " + renderEvaluationField(outcome.observed)
		if !strings.Contains(serialized, "signal action endpoint") {
			t.Fatalf("%s lost semantic relationship: %#v", name, outcome)
		}
		for _, id := range []uint32{uint32(policyID), uint32(endpointID)} {
			raw := strconv.FormatUint(uint64(id), 10)
			if strings.Contains(serialized, raw) {
				t.Fatalf("%s exposed raw signal ID %q: %#v", name, raw, outcome)
			}
		}
	}
	serialized := renderEvaluationField(outcomes["property mismatch"].subject) +
		" " + renderEvaluationField(outcomes["property mismatch"].expected) +
		" " + renderEvaluationField(outcomes["property mismatch"].observed)
	if !strings.Contains(serialized, "capacity=64") || !strings.Contains(serialized, "capacity=32") {
		t.Fatalf("capacity failure lost safe expected/observed facts: %#v", outcomes["property mismatch"])
	}
}

func TestSignalActionEndpointIdentityExpectationRemainsUnchanged(t *testing.T) {
	t.Parallel()

	policyID := cli.SignalPolicyID(71001)
	endpointID := cli.SignalEndpointID(72002)
	otherEndpointID := cli.SignalEndpointID(73003)
	schema := cli.Schema{
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicySchema{
			policyID: {
				ID:      policyID,
				Actions: []cli.SignalAction{{EndpointID: endpointID}},
			},
		},
	}
	action := SignalPolicyByID(policyID).ActionAt(0)
	if got := schemaExpectationOutcome(
		t,
		action.Endpoint(SignalEndpointByID(endpointID)),
		&schema,
	); got.status != evaluationProved {
		t.Fatalf("existing endpoint identity match = %#v", got)
	}
	if got := schemaExpectationOutcome(
		t,
		action.Endpoint(SignalEndpointByID(otherEndpointID)),
		&schema,
	); got.status != evaluationDisproved {
		t.Fatalf("existing endpoint identity mismatch = %#v", got)
	}
}
