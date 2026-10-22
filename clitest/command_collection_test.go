package clitest

import (
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestPolicyCommandCollectionsUseParentScopedIdentityAcrossArtifacts(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	ruleCode := cli.RuleCode("required")
	advisoryCode := cli.AdvisoryCode("tip")
	policyID := cli.SignalPolicyID(9)

	schema := cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{
			commandID: {
				ID: commandID,
				Conditions: []cli.ConditionSchema{{
					Reachability: cli.ReachabilityReachable,
				}},
				Rules: []cli.RuleSchema{{
					Code: ruleCode,
				}},
				Advisories: []cli.AdvisorySchema{{
					Code:    advisoryCode,
					Command: commandID,
				}},
				NoticeSuppressions: []cli.NoticeSuppressionSchema{{
					Command: commandID,
				}},
				SignalPolicies: []cli.SignalPolicyID{policyID},
			},
		},
	}
	reference := cli.Reference{
		Commands: map[cli.CommandID]cli.CommandReference{
			commandID: {
				ID: commandID,
				Conditions: []cli.ConditionSchema{{
					Reachability: cli.ReachabilityReachable,
				}},
				Rules: []cli.RuleSchema{{
					Code: ruleCode,
				}},
				Advisories: []cli.AdvisoryReference{{
					Code:    advisoryCode,
					Command: commandID,
				}},
				NoticeSuppressions: []cli.NoticeSuppressionSchema{{
					Command: commandID,
				}},
				SignalPolicies: []cli.SignalPolicyID{policyID},
			},
		},
	}
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			commandID: {
				ID:                    commandID,
				NamespaceCompleteness: cli.CommandNamespaceComplete,
				Conditions: []cli.ConditionInspection{{
					Reachability: cli.ReachabilityReachable,
				}},
				Rules: []cli.RuleInspection{{
					Code: ruleCode,
				}},
				Advisories: []cli.AdvisoryInspection{{
					Code:    advisoryCode,
					Command: commandID,
				}},
				NoticeSuppressions: []cli.NoticeSuppressionInspection{{
					Command: commandID,
				}},
				SignalPolicies: []cli.SignalPolicyID{policyID},
			},
		},
	}

	command := CommandByID(commandID)
	expectations := []DeclarationExpectation{
		command.ConditionsExactly(command.ConditionAt(0)),
		command.RulesExactly(command.Rule(ruleCode)),
		command.AdvisoriesExactly(command.Advisory(advisoryCode)),
		command.NoticeSuppressionsExactly(command.NoticeSuppressionAt(0)),
		command.SignalPoliciesExactly(SignalPolicyByID(policyID)),
	}
	for _, expectation := range expectations {
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
}

func TestPolicyCommandCollectionExactnessRejectsWrongOrderAndTruncatedAbsence(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	first := cli.RuleCode("first")
	second := cli.RuleCode("second")
	command := CommandByID(commandID)
	expectation := command.RulesExactly(command.Rule(first), command.Rule(second))
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			commandID: {
				ID:                    commandID,
				NamespaceCompleteness: cli.CommandNamespaceComplete,
				Rules: []cli.RuleInspection{{
					Code: second,
				}, {
					Code: first,
				}},
			},
		},
	}
	evaluator, evaluatorOK := expectation.(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !evaluatorOK {
		t.Fatal("expectation does not expose inspection evaluation")
	}
	outcome := evaluator.evaluateInspection(inspection, nil)
	if outcome.status != evaluationDisproved {
		t.Fatalf("reordered rules = %#v, want disproved", outcome)
	}

	inspection = cli.Inspection{
		Truncated: true,
		Commands: map[cli.CommandID]cli.CommandInspection{commandID: {
			ID: commandID,
		}},
	}
	outcome = evaluator.evaluateInspection(inspection, nil)
	if outcome.status != evaluationNotProvable {
		t.Fatalf("truncated missing rule collection = %#v, want not provable", outcome)
	}
}

func TestInspectionCommandTopologyRequiresClosedTopology(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		inspection cli.Inspection
		actual     string
	}{
		{
			name: "duplicate command identity: incomplete observation",
			inspection: cli.Inspection{
				CommandOrder: []cli.CommandID{1, 1},
				Commands: map[cli.CommandID]cli.CommandInspection{
					1: {
						ID:                    1,
						NamespaceCompleteness: cli.CommandNamespaceComplete,
					},
				},
			},
			actual: "command order contains duplicate command 1",
		},
		{
			name: "missing command detail: incomplete observation",
			inspection: cli.Inspection{
				CommandOrder: []cli.CommandID{1},
			},
			actual: "command order references missing command details for 1",
		},
		{
			name: "command identity mismatch: incomplete observation",
			inspection: cli.Inspection{
				CommandOrder: []cli.CommandID{1},
				Commands: map[cli.CommandID]cli.CommandInspection{
					1: {
						ID:                    2,
						NamespaceCompleteness: cli.CommandNamespaceComplete,
					},
				},
			},
			actual: "command detail identity 2 disagrees with command order identity 1",
		},
		{
			name: "nonexhaustive topology: incomplete observation",
			inspection: cli.Inspection{
				CommandOrder: []cli.CommandID{1},
				Commands: map[cli.CommandID]cli.CommandInspection{
					1: {
						ID:                    1,
						NamespaceCompleteness: cli.CommandNamespaceComplete,
					},
					2: {
						ID:                    2,
						NamespaceCompleteness: cli.CommandNamespaceComplete,
					},
				},
			},
			actual: "command order and command details are not exhaustive peers",
		},
		{
			name: "orphan command detail: incomplete observation",
			inspection: cli.Inspection{
				CommandOrder: []cli.CommandID{1, 2},
				Commands: map[cli.CommandID]cli.CommandInspection{
					1: {
						ID:                    1,
						NamespaceCompleteness: cli.CommandNamespaceComplete,
					},
					2: {
						ID:                    2,
						NamespaceCompleteness: cli.CommandNamespaceComplete,
					},
				},
			},
			actual: "command topology has 2 observable roots",
		},
		{
			name: "missing child detail: incomplete observation",
			inspection: cli.Inspection{
				CommandOrder: []cli.CommandID{1},
				Commands: map[cli.CommandID]cli.CommandInspection{
					1: {
						ID:                    1,
						Children:              []cli.CommandID{2},
						NamespaceCompleteness: cli.CommandNamespaceComplete,
					},
				},
			},
			actual: "command 1 references missing child details for 2",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			status, actual := inspectionCommandTopologyClosure(test.inspection)
			if status != locatorResolutionNotProvable || actual != evaluationText(test.actual) {
				t.Fatalf("topology closure = status %v, actual %q; want not provable, %q", status, actual, test.actual)
			}
		})
	}
}

func TestCommandCollectionResolutionPreservesMissingAndTruncatedObservations(t *testing.T) {
	t.Parallel()

	expectation := CommandByID(9).ValuesExactly(ValueByID(1))
	if got := schemaExpectationOutcome(
		t,
		expectation,
		&cli.Schema{},
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.observed), "command is absent") {
		t.Fatalf("schema missing command = %+v", got)
	}
	if got := referenceExpectationOutcome(
		t,
		expectation,
		cli.Reference{},
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.observed), "command is absent") {
		t.Fatalf("reference missing command = %+v", got)
	}
	inspection := cli.Inspection{
		Truncated: true,
	}
	if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationNotProvable {
		t.Fatalf("inspection missing/truncated command = %+v", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		CommandsExactly(CommandByID(1)),
		inspection,
	); got.status != evaluationNotProvable ||
		!strings.Contains(renderEvaluationField(got.observed), "truncated") {
		t.Fatalf("truncated command exactness = %+v", got)
	}
}
