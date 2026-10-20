package clitest

import (
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestCommandSelectionDetailRejectsForeignExpectedCoordinates(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			1: {
				ID:                    1,
				CommandDefaults:       []cli.CommandDefaultID{11},
				NamespaceCompleteness: cli.CommandNamespaceComplete,
			},
			2: {
				ID:                    2,
				CommandDefaults:       []cli.CommandDefaultID{22},
				ArgumentForms:         []cli.ArgumentFormID{22},
				NamespaceCompleteness: cli.CommandNamespaceComplete,
			},
		},
	}

	t.Run("command default", func(t *testing.T) {
		t.Parallel()

		expectation := CommandByID(1).DefaultCandidatesExactly(CommandByID(2).DefaultAt(0))
		got := inspectionExpectationOutcome(t, expectation, inspection)
		if got.status != evaluationDisproved || renderEvaluationField(got.observed) != "invalid or foreign command-default locator" {
			t.Fatalf("foreign command-default coordinate = %+v", got)
		}
	})

	t.Run("argument form", func(t *testing.T) {
		t.Parallel()

		expectation := CommandByID(1).MatchingArgumentFormsExactly(CommandByID(2).ArgumentFormAt(0))
		got := inspectionExpectationOutcome(t, expectation, inspection)
		if got.status != evaluationDisproved || renderEvaluationField(got.observed) != "invalid or foreign argument-form locator" {
			t.Fatalf("foreign argument-form coordinate = %+v", got)
		}
	})
}

func TestCommandSelectionDetailResolvesSemanticArgumentFormLocator(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			1: {
				ID:            1,
				Path:          []string{},
				ArgumentForms: []cli.ArgumentFormID{21},
				ArgumentFormSelection: cli.ArgumentFormSelection{
					Outcome:  cli.ArgumentFormSelectionSelected,
					Selected: 21,
					Matches:  []cli.ArgumentFormID{21},
				},
			},
		},
		ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormInspection{
			21: {
				ID:      21,
				Command: 1,
			},
		},
	}

	root := RootCommand()
	expectation := root.MatchingArgumentFormsExactly(root.ArgumentFormAt(0))
	if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationProved {
		t.Fatalf("semantic argument-form locator = %+v", got)
	}
}

func TestCommandSelectionDetailPreservesResolutionAndMismatchOutcomes(t *testing.T) {
	t.Parallel()

	missing := commandDefaultSelectionOutcomeExpectation(CommandByID(99), cli.DefaultCommandSelectionSelected)
	if got := inspectionExpectationOutcome(t, missing, cli.Inspection{}); got.status != evaluationNotProvable {
		t.Fatalf("missing command detail = %+v", got)
	}
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{1: {
			ID: 1,
			DefaultSelection: cli.DefaultCommandSelection{
				Outcome: cli.DefaultCommandSelectionSelected,
			},
		}},
	}
	mismatch := commandDefaultSelectionOutcomeExpectation(CommandByID(1), cli.DefaultCommandSelectionNone)
	if got := inspectionExpectationOutcome(
		t,
		mismatch,
		inspection,
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.observed), "default selection outcome") {
		t.Fatalf("selection outcome mismatch = %+v", got)
	}
}

func TestExpectedInspectionArgumentFormIDsRejectsInvalidAndForeignCoordinates(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			1: {ID: 1, ArgumentForms: []cli.ArgumentFormID{21}, NamespaceCompleteness: cli.CommandNamespaceComplete},
			2: {ID: 2, ArgumentForms: []cli.ArgumentFormID{22}, NamespaceCompleteness: cli.CommandNamespaceComplete},
		},
	}
	command := inspection.Commands[1]
	cases := []struct {
		name   string
		state  *argumentFormLocatorState
		status locatorResolutionStatus
	}{
		{name: "direct id", state: ArgumentFormByID(21).state, status: locatorResolutionFound},
		{name: "scoped index", state: CommandByID(1).ArgumentFormAt(0).state, status: locatorResolutionFound},
		{name: "foreign scoped index", state: CommandByID(2).ArgumentFormAt(0).state, status: locatorResolutionInvalid},
		{name: "nil", status: locatorResolutionInvalid},
		{name: "invalid mode", state: &argumentFormLocatorState{mode: argumentFormLocatorMode(255)}, status: locatorResolutionInvalid},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, status, _ := expectedInspectionArgumentFormIDs([]*argumentFormLocatorState{test.state}, inspection, command)
			if status != test.status {
				t.Fatalf("status = %v, want %v", status, test.status)
			}
		})
	}
}
