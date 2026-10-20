package clitest

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestInspectionCompletionDependenciesResolveDeclarationIdentityWithoutValueObservation(t *testing.T) {
	t.Parallel()

	const (
		rootID  cli.CommandID = 1
		flagID  cli.FlagID    = 2
		valueID cli.ValueID   = 3
	)
	inspection := cli.Inspection{
		CommandOrder: []cli.CommandID{rootID},
		Commands: map[cli.CommandID]cli.CommandInspection{
			rootID: {
				ID:                    rootID,
				NamespaceCompleteness: cli.CommandNamespaceComplete,
				Flags:                 []cli.FlagID{flagID},
				Values:                []cli.ValueID{valueID},
			},
		},
		Flags: map[cli.FlagID]cli.FlagInspection{
			flagID: {
				ID:      flagID,
				Command: rootID,
				Completion: cli.CompletionDeclarationCapabilities{
					Dependencies: []cli.ValueID{valueID},
				},
			},
		},
		Values: map[cli.ValueID]cli.ValueInspection{},
	}

	expectation := RootCommand().FlagAt(0).Completion().DependenciesExactly(RootCommand().ValueAt(0))
	evaluator, ok := expectation.(inspectionEvaluator)
	if !ok {
		t.Fatalf("expectation %T does not implement inspectionEvaluator", expectation)
	}
	if outcome := evaluator.evaluateInspection(inspection, nil); outcome.status != evaluationProved {
		t.Fatalf("completion dependency outcome = %#v, want proved", outcome)
	}
}
