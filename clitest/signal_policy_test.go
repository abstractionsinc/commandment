package clitest

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestSignalInspectionFindersRejectUnresolvedRelationalLocators(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{}

	t.Run("signal policy", func(t *testing.T) {
		t.Parallel()

		if got := resolveInspectionSignalPolicyLocator(ExecutionAdmission().SignalPolicyAt(0).state, inspection).status; got == locatorResolutionFound {
			t.Fatal("unresolved relational signal-policy locator resolved")
		}
	})

	t.Run("signal endpoint", func(t *testing.T) {
		t.Parallel()

		if got := resolveInspectionSignalEndpointLocator(ExecutionAdmission().SignalEndpointAt(0).state, inspection).status; got == locatorResolutionFound {
			t.Fatal("unresolved relational signal-endpoint locator resolved")
		}
	})
}

func TestSignalPolicyStaticExpectationUsesOwnedRecordsAcrossArtifacts(t *testing.T) {
	t.Parallel()
	policyID := cli.SignalPolicyID(7)
	endpointID := cli.SignalEndpointID(8)
	sourceCaps := cli.SignalSourceCapabilityOrigin | cli.SignalSourceCapabilitySender | cli.SignalSourceCapabilityReraise
	hostCaps := cli.SignalHostCapabilityAbort
	policySchema := cli.SignalPolicySchema{
		ID:                         policyID,
		Reachability:               cli.ReachabilityReachable,
		Sensitive:                  true,
		RequiredSourceCapabilities: sourceCaps,
		RequiredHostCapabilities:   hostCaps,
	}
	endpoint := cli.SignalEndpoint{
		ID:       endpointID,
		Policies: []cli.SignalPolicyID{policyID},
	}
	schema := cli.Schema{
		SignalPolicies:  map[cli.SignalPolicyID]cli.SignalPolicySchema{policyID: policySchema},
		SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpoint{endpointID: endpoint},
	}
	reference := cli.Reference{
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyReference{
			policyID: {
				SignalPolicySchema: policySchema,
				Available:          true,
				Visible:            true,
			},
		},
		SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpoint{endpointID: endpoint},
	}
	inspection := cli.Inspection{
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyInspection{
			policyID: {
				ID:                         policyID,
				Reachability:               cli.ReachabilityReachable,
				Sensitive:                  true,
				RequiredSourceCapabilities: sourceCaps,
				RequiredHostCapabilities:   hostCaps,
			},
		},
		SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpointInspection{
			endpointID: {
				SignalEndpoint: endpoint,
			},
		},
	}

	policy := SignalPolicyByID(policyID)
	declarations := []DeclarationExpectation{
		policy.Exists(),
		policy.Reachability(cli.ReachabilityReachable),
		policy.Sensitive(),
		policy.RequiredSourceCapabilities(sourceCaps),
		policy.RequiredHostCapabilities(hostCaps),
		SignalEndpointByID(endpointID).Exists(),
	}
	for _, expectation := range declarations {
		(SchemaResult{
			state: &schemaResultState{
				observation: schema,
			},
		}).Require(t, expectation)
		(ReferenceResult{
			state: &referenceResultState{
				observation: reference,
				err:         nil,
			},
		}).Require(t, expectation)
		(InspectionResult{
			state: &inspectionResultState{
				observation: inspection,
				err:         nil,
			},
		}).Require(t, expectation)
	}
	(ReferenceResult{
		// Availability and visibility are represented by Reference only. Inspection
		// does not carry those booleans, so neither true nor false may be inferred
		// from signal-policy existence there.
		state: &referenceResultState{
			observation: reference,
			err:         nil,
		},
	}).Require(t,

		policy.Available(), policy.Visible())

	falseReference := reference
	falseReference.SignalPolicies = map[cli.SignalPolicyID]cli.SignalPolicyReference{
		policyID: {
			SignalPolicySchema: policySchema,
			Available:          false,
			Visible:            false,
		},
	}
	(ReferenceResult{
		state: &referenceResultState{
			observation: falseReference,
			err:         nil,
		},
	}).Require(t,

		policy.Unavailable(), policy.NotVisible())

	for _, test := range []struct {
		name        string
		expectation ReferenceExpectation
	}{
		{
			name:        "available",
			expectation: policy.Available(),
		},
		{
			name:        "unavailable",
			expectation: policy.Unavailable(),
		},
		{
			name:        "visible",
			expectation: policy.Visible(),
		},
		{
			name:        "not-visible",
			expectation: policy.NotVisible(),
		},
	} {
		name := test.name
		expectation := test.expectation
		if _, ok := expectation.(InspectionExpectation); ok {
			t.Fatalf("signal policy %s expectation unexpectedly applies to Inspection", name)
		}
	}
}

func TestSignalStaticFindersRejectWrongLocatorKindsAndPreserveLabels(t *testing.T) {
	t.Parallel()
	policyID := cli.SignalPolicyID(7)
	endpointID := cli.SignalEndpointID(8)
	if got := renderEvaluationField(signalPolicyLocatorLabel(nil)); got != "signal policy" {
		t.Fatalf("signalPolicyLabel(nil) = %q", got)
	}
	if got := renderEvaluationField(signalEndpointLocatorLabel(nil)); got != "signal-endpoint" {
		t.Fatalf("renderEvaluationField(signalEndpointLocatorLabel(nil)) = %q", got)
	}
	if got := renderEvaluationField(signalPolicyLocatorLabel(SignalPolicyByID(policyID).state)); got != "signal policy id 7" {
		t.Fatalf("signalPolicyLabel = %q", got)
	}
	if got := renderEvaluationField(signalEndpointLocatorLabel(SignalEndpointByID(endpointID).state)); got != "signal endpoint id 8" {
		t.Fatalf("signalEndpointLabel = %q", got)
	}
}

func TestSignalNegativeExistenceAndBlockedPolicyPublicSurface(t *testing.T) {
	t.Parallel()
	policyID := cli.SignalPolicyID(7)
	endpointID := cli.SignalEndpointID(8)

	missingPolicy := SignalPolicyByID(policyID)
	missingEndpoint := SignalEndpointByID(endpointID)
	for _, expectation := range []DeclarationExpectation{missingPolicy.DoesNotExist(), missingEndpoint.DoesNotExist()} {
		if got := schemaExpectationOutcome(t, expectation, &cli.Schema{}); got.status != evaluationProved {
			t.Fatalf("schema missing %T = %#v", expectation, got)
		}
		if got := referenceExpectationOutcome(t, expectation, cli.Reference{}); got.status != evaluationProved {
			t.Fatalf("reference missing %T = %#v", expectation, got)
		}
		if got := inspectionExpectationOutcome(t, expectation, cli.Inspection{}); got.status != evaluationProved {
			t.Fatalf("inspection missing %T = %#v", expectation, got)
		}
	}

	inspection := cli.Inspection{
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyInspection{
			policyID: {
				ID: policyID,
			},
		},
		SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpointInspection{
			endpointID: {
				SignalEndpoint: cli.SignalEndpoint{
					ID:       endpointID,
					Policies: []cli.SignalPolicyID{policyID},
				},
				BlockedPolicies: []cli.SignalPolicyID{policyID},
			},
		},
	}
	if got := inspectionExpectationOutcome(
		t,
		SignalEndpointByID(endpointID).BlockedPoliciesExactly(SignalPolicyByID(policyID)),
		inspection,
	); got.status != evaluationProved {
		t.Fatalf("blocked policies exact = %#v", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		SignalEndpointByID(endpointID).BlockedPoliciesExactly(),
		inspection,
	); got.status != evaluationDisproved {
		t.Fatalf("blocked policies cardinality mismatch = %#v", got)
	}
}

func TestSignalReferenceAvailabilityVisibilityNegativeLaws(t *testing.T) {
	t.Parallel()
	policyID := cli.SignalPolicyID(7)
	policy := SignalPolicyByID(policyID)
	missing := cli.Reference{}
	for _, test := range []struct {
		name        string
		expectation ReferenceExpectation
	}{
		{
			name:        "availability",
			expectation: policy.Available(),
		},
		{
			name:        "visibility",
			expectation: policy.Visible(),
		},
	} {
		name := test.name
		expectation := test.expectation
		if got := referenceExpectationOutcome(t, expectation, missing); got.status != evaluationDisproved {
			t.Fatalf("missing %s = %#v", name, got)
		}
	}

	reference := cli.Reference{
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyReference{
			policyID: {
				SignalPolicySchema: cli.SignalPolicySchema{
					ID: policyID,
				},
				Available: false,
				Visible:   false,
			},
		},
	}
	for _, test := range []struct {
		name        string
		expectation ReferenceExpectation
	}{
		{
			name:        "availability",
			expectation: policy.Available(),
		},
		{
			name:        "visibility",
			expectation: policy.Visible(),
		},
	} {
		name := test.name
		expectation := test.expectation
		if got := referenceExpectationOutcome(t, expectation, reference); got.status != evaluationDisproved {
			t.Fatalf("mismatched %s = %#v", name, got)
		}
	}

	if got := resolveInspectionSignalPolicyLocator(SignalPolicyByID(99).state, cli.Inspection{}).status; got != locatorResolutionAbsent {
		t.Fatalf("missing inspection signal policy = %v, want absent", got)
	}
	if got := resolveInspectionSignalEndpointLocator(SignalEndpointByID(99).state, cli.Inspection{}).status; got != locatorResolutionAbsent {
		t.Fatalf("missing inspection signal endpoint = %v, want absent", got)
	}
}
