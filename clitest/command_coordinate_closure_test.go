package clitest

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestSelectionCommandChildNameResolutionRequiresNamespaceClosure(t *testing.T) {
	t.Parallel()

	rootID := cli.CommandID(1)
	serveID := cli.CommandID(2)
	dupAID := cli.CommandID(3)
	dupBID := cli.CommandID(4)

	tests := []struct {
		name     string
		locator  CommandLocator
		commands map[cli.CommandID]cli.CommandInspection
		wantID   cli.CommandID
		want     locatorResolutionStatus
	}{
		{
			name:    "positive survives incomplete namespace",
			locator: RootCommand().Command("serve"),
			commands: map[cli.CommandID]cli.CommandInspection{
				rootID: {
					ID:                    rootID,
					Path:                  []string{},
					Children:              []cli.CommandID{serveID},
					NamespaceCompleteness: cli.CommandNamespaceIncomplete,
				},
				serveID: {
					ID:   serveID,
					Name: "serve",
					Path: []string{"serve"},
				},
			},
			wantID: serveID,
			want:   locatorResolutionFound,
		},
		{
			name:    "missing referenced child detail is not provable",
			locator: RootCommand().Command("serve"),
			commands: map[cli.CommandID]cli.CommandInspection{
				rootID: {
					ID:                    rootID,
					Path:                  []string{},
					Children:              []cli.CommandID{serveID},
					NamespaceCompleteness: cli.CommandNamespaceComplete,
				},
			},
			want: locatorResolutionNotProvable,
		},
		{
			name:    "complete namespace checks absence",
			locator: RootCommand().Command("missing"),
			commands: map[cli.CommandID]cli.CommandInspection{
				rootID: {
					ID:                    rootID,
					Path:                  []string{},
					Children:              []cli.CommandID{serveID},
					NamespaceCompleteness: cli.CommandNamespaceComplete,
				},
				serveID: {
					ID:   serveID,
					Name: "serve",
					Path: []string{"serve"},
				},
			},
			want: locatorResolutionAbsent,
		},
		{
			name:    "unknown namespace does not establish absence",
			locator: RootCommand().Command("missing"),
			commands: map[cli.CommandID]cli.CommandInspection{
				rootID: {
					ID:                    rootID,
					Path:                  []string{},
					NamespaceCompleteness: cli.CommandNamespaceUnknown,
				},
			},
			want: locatorResolutionNotProvable,
		},
		{
			name:    "unresolved namespace does not establish absence",
			locator: RootCommand().Command("missing"),
			commands: map[cli.CommandID]cli.CommandInspection{
				rootID: {
					ID:                    rootID,
					Path:                  []string{},
					NamespaceCompleteness: cli.CommandNamespaceUnresolved,
				},
			},
			want: locatorResolutionNotProvable,
		},
		{
			name:    "incomplete namespace does not establish absence",
			locator: RootCommand().Command("missing"),
			commands: map[cli.CommandID]cli.CommandInspection{
				rootID: {
					ID:                    rootID,
					Path:                  []string{},
					NamespaceCompleteness: cli.CommandNamespaceIncomplete,
				},
			},
			want: locatorResolutionNotProvable,
		},
		{
			name:    "positive ambiguity does not require closure",
			locator: RootCommand().Command("dup"),
			commands: map[cli.CommandID]cli.CommandInspection{
				rootID: {
					ID:                    rootID,
					Path:                  []string{},
					Children:              []cli.CommandID{dupAID, dupBID},
					NamespaceCompleteness: cli.CommandNamespaceIncomplete,
				},
				dupAID: {
					ID:   dupAID,
					Name: "dup",
					Path: []string{"dup"},
				},
				dupBID: {
					ID:   dupBID,
					Name: "dup",
					Path: []string{"dup"},
				},
			},
			want: locatorResolutionAmbiguous,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			gotID, got, _ := resolveSelectionCommand(test.locator.state, test.commands)
			if got != test.want || gotID != test.wantID {
				t.Fatalf("resolveSelectionCommand() = (%v, %v), want (%v, %v)", gotID, got, test.wantID, test.want)
			}
		})
	}
}

func TestSelectionCommandPathResolutionRequiresIntermediateNamespaceClosure(t *testing.T) {
	t.Parallel()

	rootID := cli.CommandID(1)
	serveID := cli.CommandID(2)

	partial := map[cli.CommandID]cli.CommandInspection{
		rootID: {
			ID:                    rootID,
			Path:                  []string{},
			Children:              []cli.CommandID{serveID},
			NamespaceCompleteness: cli.CommandNamespaceComplete,
		},
		serveID: {
			ID:                    serveID,
			Name:                  "serve",
			Path:                  []string{"serve"},
			NamespaceCompleteness: cli.CommandNamespaceIncomplete,
		},
	}

	if id, status, _ := resolveSelectionCommand(
		Command("serve").state,
		partial,
	); status != locatorResolutionFound ||
		id != serveID {
		t.Fatalf("positive path = (%v, %v), want (%v, %v)", id, status, serveID, locatorResolutionFound)
	}
	if _, status, _ := resolveSelectionCommand(
		Command("serve", "missing").state,
		partial,
	); status != locatorResolutionNotProvable {
		t.Fatalf(
			"negative path under incomplete intermediate namespace = %v, want %v",
			status,
			locatorResolutionNotProvable,
		)
	}

	closed := partial
	serve := closed[serveID]
	serve.NamespaceCompleteness = cli.CommandNamespaceComplete
	closed[serveID] = serve
	if _, status, _ := resolveSelectionCommand(
		Command("serve", "missing").state,
		closed,
	); status != locatorResolutionAbsent {
		t.Fatalf("negative path under complete intermediate namespace = %v, want %v", status, locatorResolutionAbsent)
	}
}

func TestSelectionCommandNotProvablePropagatesThroughSemanticExpectations(t *testing.T) {
	t.Parallel()

	rootID := cli.CommandID(1)
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			rootID: {
				ID:                    rootID,
				Path:                  []string{},
				NamespaceCompleteness: cli.CommandNamespaceIncomplete,
			},
		},
		Tokens: []cli.TokenInspection{{
			Command: rootID,
		}},
		ExecutionAdmission: cli.ExecutionAdmissionInspection{
			HandlerCommand: rootID,
			Middleware: []cli.MiddlewareInspection{{
				Scope:   cli.MiddlewareScopeCommand,
				Command: rootID,
			}},
		},
	}
	missing := RootCommand().Command("missing")

	expectations := []InspectionExpectation{
		missing.Selected(),
		missing.DoesNotExist(),
		missing.EntryAt(0).DoesNotExist(),
		missing.FlagAt(0).DoesNotExist(),
		missing.ArgumentFormAt(0).DoesNotExist(),
		missing.DefaultAt(0).DoesNotExist(),
		missing.DefaultAt(0).Origin(cli.DeclarationOriginFrontend),
		missing.ProviderAt(0).DoesNotExist(),
		missing.AdvisoryAt(0).Origin(cli.DeclarationOriginFrontend),
		missing.NoticeSuppressionAt(0).Origin(cli.DeclarationOriginFrontend),
		TokenAt(0).OwnedBy(missing),
		ExecutionAdmission().Handler(missing),
		ExecutionAdmission().MiddlewareAt(0).Command(missing),
		missing.ConditionAt(0).Exists(),
		missing.RuleAt(0).Exists(),
		missing.AdvisoryAt(0).Exists(),
		missing.NoticeSuppressionAt(0).Exists(),
	}

	for _, expectation := range expectations {
		outcome := inspectionExpectationOutcome(t, expectation, inspection)
		if outcome.status != evaluationNotProvable {
			t.Fatalf("%T = %#v, want not provable", expectation, outcome)
		}
	}
}

func TestCommandDefaultExactIDMissingParentRequiresCommandClosure(t *testing.T) {
	t.Parallel()

	expectation := CommandByID(99).DefaultAt(0).DoesNotExist()
	if outcome := inspectionExpectationOutcome(t, expectation, cli.Inspection{}); outcome.status != evaluationNotProvable {
		t.Fatalf("open command domain missing exact-ID parent = %#v, want not provable", outcome)
	}
	closed := cli.Inspection{
		CommandOrder: []cli.CommandID{1},
		Commands: map[cli.CommandID]cli.CommandInspection{1: {
			ID:                    1,
			NamespaceCompleteness: cli.CommandNamespaceComplete,
		}},
	}
	if outcome := inspectionExpectationOutcome(t, expectation, closed); outcome.status != evaluationProved {
		t.Fatalf("closed command domain missing exact-ID parent = %#v, want proved absence", outcome)
	}
	closed.Truncated = true
	if outcome := inspectionExpectationOutcome(t, expectation, closed); outcome.status != evaluationNotProvable {
		t.Fatalf("truncated missing exact-ID parent = %#v, want not provable", outcome)
	}
}
