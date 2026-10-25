package conformance

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

func TestAnalyzeGrammarMissingMinimumOccurrencesHonorDiagnosticLimit(t *testing.T) {
	t.Parallel()
	type config struct{ A, B bool }
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.Bool(&root.Config.A).Occurrences(cli.Count.OneOrMore()).Long("a")
			root.Flags.Bool(&root.Config.B).Occurrences(cli.Count.OneOrMore()).Long("b")
		},
	}
	app := clitest.Compile(t, definition)
	request := cli.InspectionRequest{
		Limits: cli.AnalysisLimits{
			Diagnostics: 1,
		},
	}
	proof := clitest.Inspect(t.Context(), app, request)
	proof.Require(t, clitest.FailsWith(
		clitest.DiagnosticAt(0).Code(cli.DiagnosticRepeatedFlag),
		clitest.DiagnosticsTruncated(),
	))
	inspection, err := proof.Inspection()
	if err == nil {
		t.Fatal("expected rejected inspection")
	}
	if inspection.Status != cli.InspectionStatusRejected || len(inspection.Diagnostics) != 1 || !inspection.Truncated {
		t.Fatalf("inspection = %#v", inspection)
	}
}

func TestAnalyzeGrammarStopsRecoveryAndMarksTruncatedWhenDiagnosticLimitExhausts(t *testing.T) {
	t.Parallel()
	type config struct {
		Output  string
		Verbose bool
	}
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Output).Long("output").ValueForms(cli.FlagValueFormSeparate)
			root.Flags.Bool(&root.Config.Verbose).Long("verbose")
		},
	}
	app := clitest.Compile(t, definition)
	request := cli.InspectionRequest{
		Args: []string{"--output", "--verbose"},
		Limits: cli.AnalysisLimits{
			Diagnostics: 1,
		},
	}
	proof := clitest.Inspect(t.Context(), app, request)
	proof.Require(t, clitest.FailsWith(
		clitest.DiagnosticAt(0).Code(cli.DiagnosticMissingValue),
		clitest.DiagnosticsTruncated(),
	))
	inspection, err := proof.Inspection()
	if err == nil {
		t.Fatal("expected rejected inspection")
	}
	if len(inspection.Tokens) != 2 || inspection.Tokens[0].Role != cli.TokenRoleFlag || inspection.Tokens[1].Role != cli.TokenRoleUnknown {
		t.Fatalf("tokens = %#v", inspection.Tokens)
	}
}

func missingArgumentTokenLanguage(count cli.CountRange) cli.TokenLanguage[string] {
	language := cli.TokenLanguage[string]{
		Name:  "tokens",
		Count: count,
		Parse: func(_ context.Context, tokens []string) (string, error) {
			return strings.Join(tokens, ":"), nil
		},
		Format: func(value string) []string { return strings.Split(value, ":") },
	}
	return language
}

func requireMissingArgument[C any](t *testing.T, definition cli.Definition[C], args []string, subject string) {
	t.Helper()
	app := clitest.Compile(t, definition)
	request := cli.InspectionRequest{
		Args: args,
	}
	_, err := app.Inspect(t.Context(), request)
	if err == nil {
		t.Fatal("Inspect() error = nil")
	}
	report := cli.Diagnostics(err)
	var missing []cli.Diagnostic
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Code == cli.DiagnosticMissingValue {
			missing = append(missing, diagnostic)
		}
	}
	if len(missing) != 1 || missing[0].Subject != subject {
		t.Fatalf("missing-value Diagnostics = %#v, want one subject %q", missing, subject)
	}
}

func requireNoMissingArgument[C any](t *testing.T, definition cli.Definition[C], args []string) {
	t.Helper()
	app := clitest.Compile(t, definition)
	request := cli.InspectionRequest{
		Args: args,
	}
	_, err := app.Inspect(t.Context(), request)
	if err == nil {
		t.Fatal("Inspect() error = nil")
	}
	for _, diagnostic := range cli.Diagnostics(err).Diagnostics {
		if diagnostic.Code == cli.DiagnosticMissingValue {
			t.Fatalf("unexpected missing-value Diagnostic = %#v", diagnostic)
		}
	}
}

func requireArgumentsReady[C any](t *testing.T, definition cli.Definition[C], args []string) {
	t.Helper()
	app := clitest.Compile(t, definition)
	request := cli.InspectionRequest{
		Args: args,
	}
	inspection, err := app.Inspect(t.Context(), request)
	if err != nil || inspection.Status != cli.InspectionStatusReady {
		t.Fatalf("Inspect() = status %v, error %v; want ready", inspection.Status, err)
	}
}

func TestMissingArgumentRecoveryHonorsTokenMinimum(t *testing.T) {
	t.Parallel()
	type config struct{ Pair string }
	pairLanguage := missingArgumentTokenLanguage(cli.Count.Exactly(2))
	definition := cli.Definition[config]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[config]) {
			pair := root.Values.String(&root.Config.Pair)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Tokens(pair, "pair", pairLanguage)
			})
		},
		Run: func(context.Context, *config) error { return nil },
	}

	missingCases := []struct {
		name string
		args []string
	}{
		{
			name: "zero tokens",
		},
		{
			name: "one token",
			args: []string{"one"},
		},
	}
	for _, test := range missingCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			requireMissingArgument(t, definition, test.args, "<pair>")
		})
	}
	t.Run("minimum satisfied", func(t *testing.T) {
		t.Parallel()
		requireArgumentsReady(t, definition, []string{"one", "two"})
	})
}

func TestMissingArgumentRecoveryHonorsBoundedAndUnboundedMinimums(t *testing.T) {
	t.Parallel()
	type config struct{ Pair string }
	tests := []struct {
		name  string
		count cli.CountRange
	}{
		{
			name:  "bounded",
			count: cli.Count.Between(2, 4),
		},
		{
			name:  "unbounded",
			count: cli.Count.AtLeast(2),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			language := missingArgumentTokenLanguage(test.count)
			definition := cli.Definition[config]{
				Name: "alienctl",
				Declare: func(root *cli.CommandBuilder[config]) {
					pair := root.Values.String(&root.Config.Pair)
					root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
						form.Tokens(pair, "pair", language)
					})
				},
				Run: func(context.Context, *config) error { return nil },
			}
			requireMissingArgument(t, definition, []string{"one"}, "<pair>")
		})
	}
}

func TestMissingArgumentRecoveryHonorsSequenceMinimum(t *testing.T) {
	t.Parallel()
	type config struct{ Items []string }
	definition := cli.Definition[config]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[config]) {
			items := root.Values.Sequence(&root.Config.Items, cli.Value.String())
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Sequence(items, "items", cli.Count.Exactly(2))
			})
		},
		Run: func(context.Context, *config) error { return nil },
	}
	requireMissingArgument(t, definition, []string{"one"}, "<items>")
}

func TestMissingArgumentRecoveryHonorsRemainderMinimum(t *testing.T) {
	t.Parallel()
	type config struct{ Rest []string }
	definition := cli.Definition[config]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[config]) {
			rest := root.Values.Sequence(&root.Config.Rest, cli.Value.String())
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Remainder(rest, "rest", cli.Count.Exactly(2))
			})
		},
		Run: func(context.Context, *config) error { return nil },
	}
	requireMissingArgument(t, definition, []string{"one"}, "<rest>")
}

func TestMissingArgumentRecoveryHonorsPassThroughMinimum(t *testing.T) {
	t.Parallel()
	type config struct{ Pass []string }
	definition := cli.Definition[config]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[config]) {
			pass := root.Values.Sequence(&root.Config.Pass, cli.Value.String())
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.PassThrough(pass, "pass", cli.Count.Exactly(2))
			})
		},
		Run: func(context.Context, *config) error { return nil },
	}
	requireMissingArgument(t, definition, []string{"one"}, "<pass>")
}

func TestMissingArgumentRecoveryPreservesLeadingLiteralOwnership(t *testing.T) {
	t.Parallel()
	type config struct{ X string }
	definition := cli.Definition[config]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Literal("foo")
				form.String(&root.Config.X, "x")
			})
		},
		Run: func(context.Context, *config) error { return nil },
	}
	requireMissingArgument(t, definition, []string{"foo"}, "<x>")
}

func TestMissingArgumentRecoveryPreservesTrailingLiteralOwnership(t *testing.T) {
	t.Parallel()
	type config struct{ X string }
	definition := cli.Definition[config]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.String(&root.Config.X, "x")
				form.Literal("foo")
			})
		},
		Run: func(context.Context, *config) error { return nil },
	}
	requireMissingArgument(t, definition, []string{"foo"}, "<x>")
}

func TestMissingArgumentRecoveryRejectsMultipleDeficits(t *testing.T) {
	t.Parallel()
	type config struct {
		X string
		Y string
	}
	definition := cli.Definition[config]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.String(&root.Config.X, "x")
				form.Literal("foo")
				form.String(&root.Config.Y, "y")
			})
		},
		Run: func(context.Context, *config) error { return nil },
	}
	requireNoMissingArgument(t, definition, []string{"foo"})
}

func TestMissingArgumentRecoveryLeavesSuccessfulAllocationUnchanged(t *testing.T) {
	t.Parallel()
	type config struct{ X string }
	definition := cli.Definition[config]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.String(&root.Config.X, "x")
				form.Literal("foo")
			})
		},
		Run: func(context.Context, *config) error { return nil },
	}
	requireArgumentsReady(t, definition, []string{"foo", "foo"})
}

func TestMissingArgumentRecoveryDistinguishesRepeatedLiteralOccurrences(t *testing.T) {
	t.Parallel()
	type config struct{ X string }
	definition := cli.Definition[config]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Literal("foo")
				form.String(&root.Config.X, "x")
				form.Literal("foo")
			})
		},
		Run: func(context.Context, *config) error { return nil },
	}
	requireMissingArgument(t, definition, []string{"foo", "foo"}, "<x>")
}

func TestMissingArgumentRecoveryRequiresOneDeficientTermAcrossForms(t *testing.T) {
	t.Parallel()
	type config struct {
		X string
		Y string
	}
	definition := cli.Definition[config]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Literal("foo")
				form.String(&root.Config.X, "x")
			})
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Literal("foo")
				form.String(&root.Config.Y, "y")
			})
		},
		Run: func(context.Context, *config) error { return nil },
	}
	requireNoMissingArgument(t, definition, []string{"foo"})
}

func TestMissingArgumentRecoveryPreservesGreedyAllocation(t *testing.T) {
	t.Parallel()
	type config struct {
		Values []string
		X      string
	}
	definition := cli.Definition[config]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Strings(&root.Config.Values, "values", cli.Count.Optional()).Greedy()
				form.Literal("foo")
				form.String(&root.Config.X, "x")
			})
		},
		Run: func(context.Context, *config) error { return nil },
	}
	requireMissingArgument(t, definition, []string{"foo"}, "<x>")
}

func TestMissingArgumentRecoveryPreservesMinimalAllocation(t *testing.T) {
	t.Parallel()
	type config struct {
		Values []string
		X      string
	}
	definition := cli.Definition[config]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Strings(&root.Config.Values, "values", cli.Count.Optional()).Minimal()
				form.Literal("foo")
				form.String(&root.Config.X, "x")
			})
		},
		Run: func(context.Context, *config) error { return nil },
	}
	requireMissingArgument(t, definition, []string{"foo"}, "<x>")
}

func TestRunUnknownCommandDiagnosticRequiresCommandNamespaceWithoutPositionalOwner(t *testing.T) {
	t.Parallel()
	definition := cli.Definition[struct{}]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			fleet := cli.Command[struct{}]{
				Name: "fleet",
			}
			root.Subcommands.Command(fleet)
		},
	}
	app := clitest.Compile(t, definition)
	request := cli.RunRequest{
		Args: []string{"missing"},
	}

	err := app.Run(t.Context(), request)
	if err == nil {
		t.Fatal("Run() error = nil, want unknown-command")
	}
	clitest.Diagnostics(err).Require(t,
		clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticUnknownCommand)),
	)
}

func TestRunActionConflictDiagnosticsAttributeEachOccurrence(t *testing.T) {
	t.Parallel()
	definition := cli.Definition[struct{}]{
		Name:    "alienctl",
		Release: cli.Release.SemVer("1.0.0"),
	}
	app := clitest.Compile(t, definition)

	for _, args := range [][]string{{"--help", "--version"}, {"--help", "--help"}} {
		request := cli.RunRequest{
			Args: args,
		}
		err := app.Run(t.Context(), request)
		if err == nil {
			t.Fatalf("Run(%q) error = nil, want action-conflict", args)
		}
		report := cli.Diagnostics(err)
		if len(report.Diagnostics) != 2 {
			t.Fatalf("Diagnostics = %d, want 2", len(report.Diagnostics))
		}
		for index, diagnostic := range report.Diagnostics {
			if diagnostic.Code != cli.DiagnosticActionConflict {
				t.Fatalf("Diagnostic %d code = %q, want %q", index, diagnostic.Code, cli.DiagnosticActionConflict)
			}
			if diagnostic.Declaration.Kind != cli.DeclarationKindActionEntry || diagnostic.Span == nil {
				t.Fatalf("Diagnostic %d = %#v, want action-entry identity and span", index, diagnostic)
			}
		}
	}
}

func TestActionConflictPresentationUsesRetainedActionIdentity(t *testing.T) {
	t.Parallel()
	definition := cli.Definition[struct{}]{
		Name:    "alienctl",
		Release: cli.Release.SemVer("1.0.0"),
	}
	app := clitest.Compile(t, definition)
	tests := []struct {
		name  string
		args  []string
		limit int
		want  string
	}{
		{
			name: "repeated action",
			args: []string{"--help", "--help"},
			want: "alienctl: action --help was specified more than once\n",
		},
		{
			name:  "distinct actions truncated to one participant",
			args:  []string{"--help", "--version"},
			limit: 1,
			want:  "alienctl: conflicting actions\nAnalysis was truncated; additional diagnostic information may be unavailable.\n",
		},
		{
			name:  "repeated action truncated to one occurrence",
			args:  []string{"--help", "--help"},
			limit: 1,
			want:  "alienctl: conflicting actions\nAnalysis was truncated; additional diagnostic information may be unavailable.\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := cli.RunRequest{
				Args: test.args,
			}
			if test.limit != 0 {
				request.Limits.Diagnostics = test.limit
			}
			err := app.Run(t.Context(), request)
			if err == nil {
				t.Fatal("Run() error = nil")
			}
			if got := diagnosticErrorText(t, err); got != test.want {
				t.Fatalf("WriteError() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestActionConflictHasNoAuthoritativeActionSelection(t *testing.T) {
	t.Parallel()
	definition := cli.Definition[struct{}]{
		Name:    "alienctl",
		Release: cli.Release.SemVer("1.2.3"),
		Version: cli.VersionConfig{},
	}
	app := clitest.Compile(t, definition)
	request := cli.InspectionRequest{
		Args: []string{"--help", "--version", "help"},
	}
	inspection, err := app.Inspect(t.Context(), request)
	if err == nil {
		t.Fatal("Inspect() error = nil")
	}
	if inspection.Selection.Action.Selected != cli.InvocationActionUnknown {
		t.Fatalf("action selection = %#v, want unknown", inspection.Selection.Action)
	}
	report := cli.Diagnostics(err)
	if len(report.Diagnostics) != 3 {
		t.Fatalf("Diagnostics = %d, want 3", len(report.Diagnostics))
	}
	for index, diagnostic := range report.Diagnostics {
		if diagnostic.Code != cli.DiagnosticActionConflict {
			t.Fatalf("Diagnostic %d code = %q", index, diagnostic.Code)
		}
		if diagnostic.Declaration.Kind != cli.DeclarationKindActionEntry || diagnostic.Span == nil {
			t.Fatalf("Diagnostic %d = %#v, want action-entry identity and span", index, diagnostic)
		}
	}
}

func TestUnknownCommandDoesNotStealPositionalOwnership(t *testing.T) {
	t.Parallel()
	type config struct{ World string }
	definition := cli.Definition[config]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Arguments.String(&root.Config.World, "world")
			scan := cli.Command[config]{
				Name: "scan",
			}
			root.Subcommands.Command(scan)
		},
		Run: func(context.Context, *config) error { return nil },
	}
	app := clitest.Compile(t, definition)
	request := cli.InspectionRequest{
		Args: []string{"mars"},
	}
	inspection, err := app.Inspect(t.Context(), request)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	for _, diagnostic := range inspection.Diagnostics {
		if diagnostic.Code == cli.DiagnosticUnknownCommand {
			t.Fatalf("unexpected unknown-command diagnostic: %#v", diagnostic)
		}
	}
}

func TestUnknownCommandRequiresPositionalOwnershipToBeDisproved(t *testing.T) {
	t.Parallel()
	definition := cli.Definition[struct{}]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[struct{}]) {
				form.Literal("status")
			})
			scan := cli.Command[struct{}]{
				Name: "scan",
				Run:  func(context.Context, *struct{}) error { return nil },
			}
			root.Subcommands.Command(scan)
		},
		Run: func(context.Context, *struct{}) error { return nil },
	}
	app := clitest.Compile(t, definition)
	statusRequest := cli.RunRequest{
		Args: []string{"status"},
	}
	if err := app.Run(t.Context(), statusRequest); err != nil {
		t.Fatalf("Run(status) error = %v", err)
	}
	unknownRequest := cli.RunRequest{
		Args: []string{"scna"},
	}
	err := app.Run(t.Context(), unknownRequest)
	if err == nil {
		t.Fatal("Run(scna) error = nil")
	}
	diagnostics := cli.Diagnostics(err)
	if len(diagnostics.Diagnostics) != 1 || diagnostics.Diagnostics[0].Code != cli.DiagnosticUnknownCommand {
		t.Fatalf("Diagnostics = %#v, want one unknown-command Diagnostic", diagnostics.Diagnostics)
	}
}
