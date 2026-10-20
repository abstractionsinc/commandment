package clitest

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestResultReuseNeverRerunsCompletionOperation(t *testing.T) {
	t.Parallel()
	type config struct{ Path string }
	var calls atomic.Int64
	//nolint:unparam // The public callback contract fixes this result signature.
	completePath := func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
		calls.Add(1)
		contribution := cli.CompletionContribution{
			Candidates: []cli.CompletionCandidate{{Value: "one"}},
		}
		return contribution, nil
	}
	declare := func(root *cli.CommandBuilder[config]) {
		root.Flags.String(&root.Config.Path).CompleteBy(completePath).Long("path")
	}
	definition := cli.Definition[config]{
		Name:    "clitest-no-rerun",
		Declare: declare,
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	request := cli.CompletionRequest{
		Args: []string{"--path", ""},
		Cursor: cli.CompletionCursor{
			Argument: 1,
		},
	}
	result := Complete(context.Background(), app, request)
	if got := calls.Load(); got != 1 {
		t.Fatalf("completion callback calls after operation = %d, want 1", got)
	}

	expectation := CompletionTargetAt(0).Present()
	result.Require(t, expectation)
	result.Require(t, expectation)
	if got := calls.Load(); got != 1 {
		t.Fatalf("completion callback calls after result reuse = %d, want 1", got)
	}
}

func TestTokenExpectationsUseInspectionOwnershipRecords(t *testing.T) {
	t.Parallel()
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{2: {
			ID:   2,
			Path: []string{"serve"},
		}},
		Tokens: []cli.TokenInspection{{
			Role:    cli.TokenRoleFlagValue,
			Command: 2,
			Declaration: cli.DeclarationIdentity{
				Kind: cli.DeclarationKindFlag,
				Flag: 4,
			},
			Destination:     7,
			MatchedSpelling: "--port",
			Match:           cli.NameMatchKindExact,
			FlagValueForm:   cli.FlagValueFormSeparate,
			OccurrenceIndex: 1,
			ElementIndex:    2,
			DelegatedIndex:  3,
			RejectionCode:   cli.DiagnosticInvalidValue,
		}},
	}

	assertInspectionExpectation(t, TokenAt(0).Role(cli.TokenRoleFlagValue), inspection)
	assertInspectionExpectation(t, TokenAt(0).OwnedBy(Command("serve")), inspection)
	assertInspectionExpectation(t, TokenAt(0).OwnedBy(FlagByID(4)), inspection)
	assertInspectionExpectation(t, TokenAt(0).OwnedBy(ValueByID(7)), inspection)
	assertInspectionExpectation(t, TokenAt(0).MatchedSpelling("--port"), inspection)
	assertInspectionExpectation(t, TokenAt(0).Match(cli.NameMatchKindExact), inspection)
	assertInspectionExpectation(t, TokenAt(0).FlagValueForm(cli.FlagValueFormSeparate), inspection)
	assertInspectionExpectation(t, TokenAt(0).OccurrenceIndex(1), inspection)
	assertInspectionExpectation(t, TokenAt(0).ElementIndex(2), inspection)
	assertInspectionExpectation(t, TokenAt(0).DelegatedIndex(3), inspection)
	assertInspectionExpectation(t, TokenAt(0).RejectedWith(cli.DiagnosticInvalidValue), inspection)
}
