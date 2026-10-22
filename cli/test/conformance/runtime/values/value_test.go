package conformance

import (
	"context"
	"errors"
	"math"
	"strconv"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

type authoringComputationBaseCountFlagInputArgumentInputDerivedOptionalFactSelectedConfig struct {
	Base                                                                        string
	Count                                                                       uint
	FlagInput, ArgumentInput, Derived, Optional, Fact, Selected, SelectedFormat string
}

type authoringComputationSourceFromDestinationFromFlagFromArgumentArgumentConfig struct {
	Source          string
	FromDestination string
	FromFlag        string
	FromArgument    string
	Argument        string
}

func TestComputationAuthoringBindsExactDependencyIdentities(t *testing.T) {
	t.Parallel()
	type config = authoringComputationBaseCountFlagInputArgumentInputDerivedOptionalFactSelectedConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			base := root.Values.String(&root.Config.Base)
			count := root.Values.Uint(&root.Config.Count)
			root.Values.Default(count, uint(1))
			flagInput := root.Values.String(&root.Config.FlagInput)
			flag := root.Flags.ValueFor(flagInput)
			flag.Long("base")
			argumentInput := root.Values.String(&root.Config.ArgumentInput)
			var arg cli.ArgumentDeclaration[config, string]
			form := root.Arguments.Form(
				func(b *cli.ArgumentFormBuilder[config]) { arg = b.ValueFor(argumentInput, "base") },
			)
			form.AvailableWhen(root.When.Resolved(count))
			form.VisibleWhen(root.When.Equal(count, uint(1)))
			root.Values.Derive(
				&root.Config.Derived,
				cli.Value.ComputedBy(func(_ context.Context, _ *config, deps cli.Dependencies) (string, error) {
					v, _ := deps.Get(base)
					return v, nil
				}).From(base, flag.OptionalDependency(), arg.OptionalDependency(), count.OptionalDependency()),
			)
			root.Values.OptionalDerive(
				&root.Config.Optional,
				cli.Value.OptionallyComputedBy(func(_ context.Context, _ *config, deps cli.Dependencies) (cli.ValueContribution[string], error) {
					v, ok := deps.Get(base)
					if !ok {
						return cli.OmitValue[string](), nil
					}
					return cli.ContributeValue(v), nil
				}).
					From(base.OptionalDependency()),
			)
			root.Facts.Resolve(
				&root.Config.Fact,
				func(_ context.Context, _ *config, deps cli.Dependencies) (string, error) {
					v, _ := deps.Get(base)
					return v, nil
				},
			).From(base)
			selected := root.Values.String(&root.Config.Selected)
			selectedFormat := root.Values.String(&root.Config.SelectedFormat)
			selectionForm := root.Arguments.Form(func(b *cli.ArgumentFormBuilder[config]) { b.Literal("choose") })
			selectionForm.
				Selects(selected, "selected").
				Selects(selectedFormat, "json")
		},
	}
	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	base := root.ValueAt(0)
	count := root.ValueAt(1)
	flagInput := root.Flag("--base").Value()
	argumentInput := root.ArgumentFormAt(0).Term("base").Value()
	derived := root.ValueAt(4)
	optional := root.ValueAt(5)
	available := root.ConditionAt(0)
	visible := root.ConditionAt(1)
	clitest.Schema(app).Require(t,
		derived.Kind(cli.ValueDestinationDerived),
		derived.DependenciesExactly(base, flagInput, argumentInput, count),
		optional.Kind(cli.ValueDestinationOptionalDerived),
		optional.DependenciesExactly(base),
		root.FactAt(0).DependenciesExactly(base),
		available.Expression().Destination(count),
		visible.Expression().Destination(count),
	)
}

func TestComputationOptionalDependenciesAndGetAffectObservedValues(t *testing.T) {
	t.Parallel()
	type config = authoringComputationSourceFromDestinationFromFlagFromArgumentArgumentConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			source := root.Values.String(&root.Config.Source)
			flag := root.Flags.ValueFor(source)
			flag.Long("source")
			argumentValue := root.Values.String(&root.Config.Argument)
			var argument cli.ArgumentDeclaration[config, string]
			root.Arguments.Form(func(b *cli.ArgumentFormBuilder[config]) {
				argument = b.ValueFor(argumentValue, "argument").Optional()
			})
			root.Values.Derive(
				&root.Config.FromDestination,
				cli.Value.ComputedBy(func(_ context.Context, _ *config, deps cli.Dependencies) (string, error) {
					if v, ok := deps.Get(source); ok {
						return v, nil
					}
					return "destination-missing", nil
				}).From(source.OptionalDependency()),
			)
			root.Values.Derive(
				&root.Config.FromFlag,
				cli.Value.ComputedBy(func(_ context.Context, _ *config, deps cli.Dependencies) (string, error) {
					if v, ok := deps.Get(source); ok {
						return v, nil
					}
					return "flag-missing", nil
				}).From(flag.OptionalDependency()),
			)
			root.Values.Derive(
				&root.Config.FromArgument,
				cli.Value.ComputedBy(func(_ context.Context, _ *config, deps cli.Dependencies) (string, error) {
					if v, ok := deps.Get(argumentValue); ok {
						return v, nil
					}
					return "argument-missing", nil
				}).From(argument.OptionalDependency()),
			)
		},
	}
	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	source := root.Flag("--source").Value()
	argument := root.Argument("argument").Value()
	fromDestination := root.ValueAt(2)
	fromFlag := root.ValueAt(3)
	fromArgument := root.ValueAt(4)
	clitest.Schema(app).Require(t,
		fromDestination.DependenciesExactly(source),
		fromFlag.DependenciesExactly(source),
		fromArgument.DependenciesExactly(argument),
	)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		fromDestination.Resolved(),
		fromFlag.Resolved(),
		fromArgument.Resolved(),
	)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--source", "present", "argument"},
	}).Require(t,
		fromDestination.Resolved(),
		fromFlag.Resolved(),
		fromArgument.Resolved(),
	)
}

func TestComputationAvailableWhenAcceptsZeroValueDeclaration(t *testing.T) {
	t.Parallel()

	var form cli.ArgumentFormDeclaration[int]
	form.AvailableWhen(cli.Condition{})
}

func TestComputationVisibleWhenAcceptsZeroValueDeclaration(t *testing.T) {
	t.Parallel()

	var form cli.ArgumentFormDeclaration[int]
	form.VisibleWhen(cli.Condition{})
}

func TestSelectsAcceptsZeroValueDeclaration(t *testing.T) {
	t.Parallel()

	var form cli.ArgumentFormDeclaration[int]
	var destination cli.Destination[int, string]
	form.Selects(destination, "value").Selects(destination, "other")
}

func TestArgumentFormSelectsRejectsDuplicateDestination(t *testing.T) {
	t.Parallel()
	type config struct{ Value string }
	clitest.CompileFails(t, cli.Definition[config]{
		Name: "duplicate-form-selection",
		Declare: func(root *cli.CommandBuilder[config]) {
			destination := root.Values.String(&root.Config.Value)
			form := root.Arguments.Form(func(builder *cli.ArgumentFormBuilder[config]) {
				builder.Literal("choose")
			})
			form.
				Selects(destination, "first").
				Selects(destination, "second")
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestDependenciesGetReturnsAbsentForZeroValue(t *testing.T) {
	t.Parallel()

	var dependencies cli.Dependencies
	var destination cli.Destination[int, string]
	if value, ok := dependencies.Get(destination); ok || value != "" {
		t.Fatalf("zero dependencies Get = (%q, %v)", value, ok)
	}
}

type authoringSourceNameArgumentFactItemsLabelsWholeConfig struct {
	Name     string
	Argument string
	Fact     string
	Items    []string
	Labels   map[string]string
	Whole    map[string]string
}

func declareAuthoringSourceFixture(root *cli.CommandBuilder[authoringSourceNameArgumentFactItemsLabelsWholeConfig]) {
	name := root.Values.String(&root.Config.Name).Sensitive()
	root.Values.Default(name, "guest").Sensitive()
	root.Flags.ValueFor(name).
		Long("name").
		FixedValue("fixed", cli.Source.Key("default"), cli.Source.Location("built-in"), cli.Source.Text("default source"), cli.Source.Sensitive()).
		Sensitive()
	root.Values.Environment(name, "TOOL_NAME").Sensitive()
	items := root.Values.Sequence(&root.Config.Items, cli.Value.String())
	root.Values.Environment(items, "TOOL_ITEMS").WholeSequence(",")
	labels := root.Values.Keyed(&root.Config.Labels, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
	root.Values.EnvironmentPrefix(labels, "TOOL_")
	whole := root.Values.Keyed(&root.Config.Whole, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
	root.Values.Environment(whole, "TOOL_WHOLE").WholeKeyed(";")
	root.Arguments.String(&root.Config.Argument, "argument").Sensitive()
	root.Facts.Require(&root.Config.Fact).Sensitive()
}

func TestAuthoringSourceOwnerMethodsPreserveAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSourceNameArgumentFactItemsLabelsWholeConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSourceFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().Exists())
}

func TestSourceOptionPreservesProducerMetadataValidation(t *testing.T) {
	t.Parallel()
	type config = authoringSourceNameArgumentFactItemsLabelsWholeConfig
	clitest.CompileFails(t, cli.Definition[config]{
		Name: "bad",
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Name)
			var invalid cli.SourceOption
			root.Flags.ValueFor(value).Long("value").FixedValue("x", invalid)
		},
	})
}

func valueMustCompilePublic[C any](t testing.TB, definition cli.Definition[C]) cli.Application {
	t.Helper()
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return app
}

type operationPreparationValueConfig struct{ Value string }

type operationPreparationSourceDerivedConfig struct {
	Source  string
	Derived string
}

func TestRunPreservesDefaultSupplyFailureAfterSelection(t *testing.T) {
	t.Parallel()
	type config = operationPreparationValueConfig
	cause := errors.New("default supply")
	definition := cli.Definition[config]{
		Name: "default-failure",
		Run:  func(context.Context, *config) error { return nil },
	}
	app := clitest.Compile(t, definition)
	request := cli.RunRequest{
		Defaults: []cli.DefaultProvider{
			cli.DefaultsFor[config](func(context.Context, *cli.DefaultSupply[config]) error { return cause }),
		},
	}

	if err := app.Run(t.Context(), request); !errors.Is(err, cause) {
		t.Fatalf("Run() error = %v, want %v", err, cause)
	}
}

func TestRunAndInspectionBlockedDerivedValueRemainIncomplete(t *testing.T) {
	t.Parallel()
	type config = operationPreparationSourceDerivedConfig
	handlerCalled := false
	definition := cli.Definition[config]{
		Name: "derived-incomplete",
		Declare: func(root *cli.CommandBuilder[config]) {
			source := root.Values.String(&root.Config.Source)
			root.Values.Derive(
				&root.Config.Derived,
				cli.Value.ComputedBy(func(_ context.Context, _ *config, deps cli.Dependencies) (string, error) {
					value, _ := deps.Get(source)
					return value, nil
				}).From(source),
			)
		},
		Run: func(context.Context, *config) error {
			handlerCalled = true
			return nil
		},
	}
	app := clitest.Compile(t, definition)

	runErr := app.Run(t.Context(), cli.RunRequest{})
	if runErr == nil {
		t.Fatal("Run() error = nil, want incomplete invocation to fail closed")
	}
	if handlerCalled {
		t.Fatal("Run() executed handler for incomplete invocation")
	}
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t, clitest.InspectionIncomplete())
}

type valueStateNameEnabledConfig struct {
	Name    string
	Enabled bool
}

type valueStateModeConfig struct{ Mode string }

type valueStateSecretPathConfig struct {
	Secret string
	Path   string
}

type valueStateModePathConfig struct {
	Mode string
	Path string
}

func TestInspectResolvedRootSelectionSelectsRootWithoutDiagnostics(t *testing.T) {
	t.Parallel()
	type config = valueStateNameEnabledConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Name).Long("name")
			root.Flags.Bool(&root.Config.Enabled).Long("enabled")
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "child",
			})
		},
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--name", "alpha", "--enabled"},
	}).Require(t,
		clitest.DiagnosticsExactly(),
		clitest.RootCommand().Selected(),
	)
}

func TestZeroValueStateContributorsAreEmpty(t *testing.T) {
	t.Parallel()
	if got := (cli.ValueState{}).Contributors(); got != nil {
		t.Fatalf("Contributors() = %#v, want nil", got)
	}
}

func TestZeroValueStateProducersAreEmpty(t *testing.T) {
	t.Parallel()
	if got := (cli.ValueState{}).Producers(); got != nil {
		t.Fatalf("Producers() = %#v, want nil", got)
	}
}

func TestZeroValueStateAndCandidateAreNotSensitive(t *testing.T) {
	t.Parallel()
	if (cli.ValueState{}).Sensitive() {
		t.Fatal("zero ValueState is sensitive")
	}
	if (cli.ValueCandidate{}).Sensitive() {
		t.Fatal("zero ValueCandidate is sensitive")
	}
}

func TestZeroValueCandidateSourceIsUnknown(t *testing.T) {
	t.Parallel()
	if got := (cli.ValueCandidate{}).ProducerKind(); got != cli.ValueProducerKindUnknown {
		t.Fatalf("Source() = %v, want unknown", got)
	}
}

func TestZeroValueCandidateTierIsUnknown(t *testing.T) {
	t.Parallel()
	if got := (cli.ValueCandidate{}).Tier(); got != cli.ResolutionTierUnknown {
		t.Fatalf("Tier() = %v, want unknown", got)
	}
}

func TestZeroValueCandidateDispositionIsUnknown(t *testing.T) {
	t.Parallel()
	if got := (cli.ValueCandidate{}).Disposition(); got != cli.ValueCandidateDispositionUnknown {
		t.Fatalf("Disposition() = %v, want unknown", got)
	}
}

func TestZeroValueCandidateContributionIsUnknown(t *testing.T) {
	t.Parallel()
	if got := (cli.ValueCandidate{}).Contribution(); got != cli.ValueCandidateContributionUnknown {
		t.Fatalf("Contribution() = %v, want unknown", got)
	}
}

func TestZeroValueCandidateDeclarationIsZero(t *testing.T) {
	t.Parallel()
	if got := (cli.ValueCandidate{}).Declaration(); got != (cli.DeclarationIdentity{}) {
		t.Fatalf("Declaration() = %#v, want zero", got)
	}
}

func TestZeroValueCandidateInputSourceIsAbsent(t *testing.T) {
	t.Parallel()
	if got, ok := (cli.ValueCandidate{}).InputSource(); ok || got != (cli.InputSourceID{}) {
		t.Fatalf("InputSource() = %#v, %v, want zero, false", got, ok)
	}
}

func TestZeroValueCandidateOriginsAreEmpty(t *testing.T) {
	t.Parallel()
	if got := (cli.ValueCandidate{}).Origins(); got != nil {
		t.Fatalf("Origins() = %#v, want nil", got)
	}
}

func TestZeroValueCandidateInputIsAbsent(t *testing.T) {
	t.Parallel()
	if got, ok := (cli.ValueCandidate{}).Input(); ok || got != "" {
		t.Fatalf("Input() = %q, %v, want empty, false", got, ok)
	}
}

func TestZeroValueCandidateInputIsNotRedacted(t *testing.T) {
	t.Parallel()
	if (cli.ValueCandidate{}).InputRedacted() {
		t.Fatal("zero ValueCandidate reports redacted input")
	}
}

func TestZeroValueCandidateRepresentationIsAbsent(t *testing.T) {
	t.Parallel()
	if got, ok := (cli.ValueCandidate{}).Representation(); ok || got.Kind != cli.ValueRepresentationUnknown ||
		got.Text != "" ||
		got.Tokens != nil ||
		got.Redacted {
		t.Fatalf("Representation() = %#v, %v, want zero, false", got, ok)
	}
}

func TestZeroValueCandidateDomainEntryIsZero(t *testing.T) {
	t.Parallel()
	if got := (cli.ValueCandidate{}).DomainEntry(); got != 0 {
		t.Fatalf("DomainEntry() = %v, want zero", got)
	}
}

func TestZeroValueCandidateSuppressionIsUnknown(t *testing.T) {
	t.Parallel()
	if got := (cli.ValueCandidate{}).Suppression(); got != cli.ValueCandidateSuppressionUnknown {
		t.Fatalf("Suppression() = %v, want unknown", got)
	}
}

func TestZeroValueCandidateSuppressedByIsEmpty(t *testing.T) {
	t.Parallel()
	if got := (cli.ValueCandidate{}).SuppressedBy(); got != nil {
		t.Fatalf("SuppressedBy() = %#v, want nil", got)
	}
}

func TestZeroValueCandidateConflictsWithIsEmpty(t *testing.T) {
	t.Parallel()
	if got := (cli.ValueCandidate{}).ConflictsWith(); got != nil {
		t.Fatalf("ConflictsWith() = %#v, want nil", got)
	}
}

func TestValueStateFromContextNilContextReturnsUnknown(t *testing.T) {
	t.Parallel()
	type config = valueStateModeConfig
	var mode cli.Destination[config, string]
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			mode = root.Values.String(&root.Config.Mode)
		},
	})
	_ = app
	var nilContext context.Context
	if state, ok := cli.ValueStateFromContext(nilContext, mode); ok || state.State() != cli.ResolutionStateUnknown {
		t.Fatalf("ValueStateFromContext(nil) = (%#v, %v), want unknown,false", state, ok)
	}
}

func TestZeroValueStateResolutionStateIsUnknown(t *testing.T) {
	t.Parallel()
	if got := (cli.ValueState{}).State(); got != cli.ResolutionStateUnknown {
		t.Fatalf("zero ValueState.State() = %v, want unknown", got)
	}
}

func TestValueStateCandidatesExposeCompletionDependencyCandidates(t *testing.T) {
	t.Parallel()
	if got := (cli.ValueState{}).Candidates(); got != nil {
		t.Fatalf("zero ValueState.Candidates() = %#v, want nil", got)
	}

	type config = valueStateModePathConfig
	observed := false
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			mode := root.Values.String(&root.Config.Mode)
			root.Values.ResolveWith(mode, cli.Value.KeepLast())
			root.Flags.ValueFor(mode).Occurrences(cli.Count.Between(1, 2)).Long("mode")
			root.Flags.String(&root.Config.Path).CompleteBy(func(ctx context.Context, _ *config, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				state, ok := cli.ValueStateFromContext(ctx, mode)
				if !ok {
					t.Fatal("completion dependency ValueState is unavailable")
				}
				if len(state.Candidates()) != 2 {
					t.Fatalf("Candidates() = %#v, want two candidates", state.Candidates())
				}
				if len(state.Contributors()) != 1 {
					t.Fatalf("Contributors() = %#v, want one effective contributor", state.Contributors())
				}
				observed = true
				return cli.CompletionContribution{}, nil
			}).From(mode).
				Long("path")
		},
	})
	clitest.Complete(t.Context(), app, cli.CompletionRequest{
		Args: []string{"--mode", "first", "--mode", "second", "--path", ""},
		Cursor: cli.CompletionCursor{
			Argument: 5,
		},
	}).Require(t, clitest.DiagnosticsExactly())
	if !observed {
		t.Fatal("completion dependency callback did not observe ValueState")
	}
}

func TestValueStateProducersExposeExplicitAndDefaultProducers(t *testing.T) {
	t.Parallel()
	if got := (cli.ValueState{}).Producers(); got != nil {
		t.Fatalf("zero ValueState.Producers() = %#v, want nil", got)
	}

	type config = valueStateModePathConfig
	observed := false
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			mode := root.Values.String(&root.Config.Mode)
			root.Values.Default(mode, "default")
			root.Flags.ValueFor(mode).Long("mode")
			root.Flags.String(&root.Config.Path).CompleteBy(func(ctx context.Context, _ *config, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				state, ok := cli.ValueStateFromContext(ctx, mode)
				if !ok {
					t.Fatal("completion dependency ValueState is unavailable")
				}
				producers := state.Producers()
				if len(producers) < 2 {
					t.Fatalf("Producers() = %#v, want explicit and default producers", producers)
				}
				if producers[0].Kind == cli.ValueProducerKindUnknown || producers[0].Tier == cli.ResolutionTierUnknown ||
					producers[0].Outcome == cli.ValueProducerOutcomeUnknown {
					t.Fatalf("producer semantics = %#v", producers[0])
				}
				observed = true
				return cli.CompletionContribution{}, nil
			}).From(mode).
				Long("path")
		},
	})
	clitest.Complete(t.Context(), app, cli.CompletionRequest{
		Args: []string{"--mode", "explicit", "--path", ""},
		Cursor: cli.CompletionCursor{
			Argument: 3,
		},
	}).Require(t, clitest.DiagnosticsExactly())
	if !observed {
		t.Fatal("completion dependency callback did not observe producers")
	}
}

func TestValueStateSensitivityPropagatesToCandidates(t *testing.T) {
	t.Parallel()
	if (cli.ValueState{}).Sensitive() {
		t.Fatal("zero ValueState is sensitive")
	}
	var zeroCandidate cli.ValueCandidate
	if zeroCandidate.Sensitive() {
		t.Fatal("zero ValueCandidate is sensitive")
	}

	type config = valueStateSecretPathConfig
	observed := false
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			secret := root.Values.String(&root.Config.Secret).Sensitive()
			root.Flags.ValueFor(secret).Long("secret").Sensitive()
			root.Flags.String(&root.Config.Path).CompleteBy(func(ctx context.Context, _ *config, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				state, ok := cli.ValueStateFromContext(ctx, secret)
				if !ok || !state.Sensitive() || len(state.Candidates()) != 1 || !state.Candidates()[0].Sensitive() {
					t.Fatalf("sensitive ValueState = %#v, ok=%v", state, ok)
				}
				observed = true
				return cli.CompletionContribution{}, nil
			}).From(secret).
				Long("path")
		},
	})
	clitest.Complete(t.Context(), app, cli.CompletionRequest{
		Args: []string{"--secret", "hidden", "--path", ""},
		Cursor: cli.CompletionCursor{
			Argument: 3,
		},
	}).Require(t, clitest.DiagnosticsExactly())
	if !observed {
		t.Fatal("completion dependency callback did not observe sensitivity")
	}
}

func TestValueCandidateInspectionAccessorsExposeResolvedMetadata(t *testing.T) {
	t.Parallel()
	var zero cli.ValueCandidate
	if zero.Disposition() != cli.ValueCandidateDispositionUnknown ||
		zero.Contribution() != cli.ValueCandidateContributionUnknown ||
		zero.Declaration() != (cli.DeclarationIdentity{}) {
		t.Fatalf("zero candidate inspection methods returned nonzero semantics")
	}

	type config = valueStateModePathConfig
	observed := false
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			mode := root.Values.String(&root.Config.Mode)
			root.Flags.ValueFor(mode).Long("mode")
			root.Flags.String(&root.Config.Path).CompleteBy(func(ctx context.Context, _ *config, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				state, ok := cli.ValueStateFromContext(ctx, mode)
				if !ok || len(state.Candidates()) != 1 {
					t.Fatalf("ValueState = %#v, ok=%v", state, ok)
				}
				candidate := state.Candidates()[0]
				if candidate.Disposition() == cli.ValueCandidateDispositionUnknown ||
					candidate.Contribution() == cli.ValueCandidateContributionUnknown ||
					candidate.Declaration() == (cli.DeclarationIdentity{}) {
					t.Fatalf("candidate inspection semantics are incomplete")
				}
				observed = true
				return cli.CompletionContribution{}, nil
			}).From(mode).
				Long("path")
		},
	})
	clitest.Complete(t.Context(), app, cli.CompletionRequest{
		Args: []string{"--mode", "fast", "--path", ""},
		Cursor: cli.CompletionCursor{
			Argument: 3,
		},
	}).Require(t, clitest.DiagnosticsExactly())
	if !observed {
		t.Fatal("completion dependency callback did not inspect a candidate")
	}
}

func TestValueCandidateInputSourceIsAvailableForObservedInput(t *testing.T) {
	t.Parallel()
	var zero cli.ValueCandidate
	if source, ok := zero.InputSource(); ok || source != (cli.InputSourceID{}) {
		t.Fatalf("zero ValueCandidate.InputSource() = (%#v,%v)", source, ok)
	}

	type config = valueStateModePathConfig
	observed := false
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			mode := root.Values.String(&root.Config.Mode)
			root.Flags.ValueFor(mode).Long("mode")
			root.Flags.String(&root.Config.Path).CompleteBy(func(ctx context.Context, _ *config, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				state, ok := cli.ValueStateFromContext(ctx, mode)
				if !ok || len(state.Candidates()) != 1 {
					t.Fatalf("ValueState = %#v, ok=%v", state, ok)
				}
				if source, ok := state.Candidates()[0].InputSource(); !ok || source == (cli.InputSourceID{}) {
					t.Fatalf("candidate InputSource() = (%#v,%v)", source, ok)
				}
				observed = true
				return cli.CompletionContribution{}, nil
			}).From(mode).
				Long("path")
		},
	})
	clitest.Complete(t.Context(), app, cli.CompletionRequest{
		Args: []string{"--mode", "fast", "--path", ""},
		Cursor: cli.CompletionCursor{
			Argument: 3,
		},
	}).Require(t, clitest.DiagnosticsExactly())
	if !observed {
		t.Fatal("completion dependency callback did not observe input source")
	}
}

func TestValueCandidateInputReturnsObservedInput(t *testing.T) {
	t.Parallel()
	type config = valueStateModePathConfig
	observed := false
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			mode := root.Values.String(&root.Config.Mode)
			root.Flags.ValueFor(mode).Long("mode")
			root.Flags.String(&root.Config.Path).CompleteBy(func(ctx context.Context, _ *config, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				state, ok := cli.ValueStateFromContext(ctx, mode)
				if !ok || len(state.Candidates()) != 1 {
					t.Fatalf("ValueState = %#v, ok=%v", state, ok)
				}
				input, ok := state.Candidates()[0].Input()
				if !ok || input != "fast" {
					t.Fatalf("candidate Input() = (%q,%v), want fast,true", input, ok)
				}
				observed = true
				return cli.CompletionContribution{}, nil
			}).From(mode).
				Long("path")
		},
	})
	clitest.Complete(t.Context(), app, cli.CompletionRequest{
		Args: []string{"--mode", "fast", "--path", ""},
		Cursor: cli.CompletionCursor{
			Argument: 3,
		},
	}).Require(t, clitest.DiagnosticsExactly())
	if !observed {
		t.Fatal("completion dependency callback did not observe candidate input")
	}
}

func TestValueCandidateRepresentationReturnsObservedText(t *testing.T) {
	t.Parallel()
	type config = valueStateModePathConfig
	observed := false
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			mode := root.Values.String(&root.Config.Mode)
			root.Flags.ValueFor(mode).Long("mode")
			root.Flags.String(&root.Config.Path).CompleteBy(func(ctx context.Context, _ *config, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				state, ok := cli.ValueStateFromContext(ctx, mode)
				if !ok || len(state.Candidates()) != 1 {
					t.Fatalf("ValueState = %#v, ok=%v", state, ok)
				}
				representation, ok := state.Candidates()[0].Representation()
				if !ok || representation.Text != "fast" || representation.Redacted {
					t.Fatalf("candidate Representation() = (%#v,%v)", representation, ok)
				}
				observed = true
				return cli.CompletionContribution{}, nil
			}).From(mode).
				Long("path")
		},
	})
	clitest.Complete(t.Context(), app, cli.CompletionRequest{
		Args: []string{"--mode", "fast", "--path", ""},
		Cursor: cli.CompletionCursor{
			Argument: 3,
		},
	}).Require(t, clitest.DiagnosticsExactly())
	if !observed {
		t.Fatal("completion dependency callback did not observe representation")
	}
}

func TestValueCandidateRelationsExposeSuppressionAndConflicts(t *testing.T) {
	t.Parallel()
	t.Run("suppression", func(t *testing.T) {
		t.Parallel()
		type config = valueStateModePathConfig
		observed := false
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				mode := root.Values.String(&root.Config.Mode)
				root.Values.ResolveWith(mode, cli.Value.KeepLast())
				root.Flags.ValueFor(mode).Occurrences(cli.Count.Between(1, 2)).Long("mode")
				root.Flags.String(&root.Config.Path).CompleteBy(func(ctx context.Context, _ *config, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
					state, ok := cli.ValueStateFromContext(ctx, mode)
					if !ok || len(state.Candidates()) != 2 {
						t.Fatalf("ValueState = %#v, ok=%v", state, ok)
					}
					found := false
					for _, candidate := range state.Candidates() {
						_ = candidate.ConflictsWith()
						if related := candidate.SuppressedBy(); len(related) != 0 {
							found = true
							if candidate.Suppression() != cli.ValueCandidateSuppressionKeepLast {
								t.Fatalf("suppressed candidate reason = %v", candidate.Suppression())
							}
						}
					}
					if !found {
						t.Fatal("no candidate exposed its suppressing candidate")
					}
					observed = true
					return cli.CompletionContribution{}, nil
				}).From(mode).
					Long("path")
			},
		})
		clitest.Complete(t.Context(), app, cli.CompletionRequest{
			Args: []string{"--mode", "first", "--mode", "second", "--path", ""},
			Cursor: cli.CompletionCursor{
				Argument: 5,
			},
		}).Require(t, clitest.DiagnosticsExactly())
		if !observed {
			t.Fatal("completion dependency callback did not observe suppression relation")
		}
	})

	t.Run("conflict", func(t *testing.T) {
		t.Parallel()
		type config = valueStateModePathConfig
		observed := false
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				mode := root.Values.String(&root.Config.Mode)
				root.Flags.ValueFor(mode).Occurrences(cli.Count.Between(1, 2)).Long("mode")
				root.Flags.String(&root.Config.Path).CompleteBy(func(ctx context.Context, _ *config, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
					state, ok := cli.ValueStateFromContext(ctx, mode)
					if !ok || state.State() != cli.ResolutionStateFailed || len(state.Candidates()) != 2 {
						t.Fatalf("conflicting ValueState = %#v, ok=%v", state, ok)
					}
					found := false
					for _, candidate := range state.Candidates() {
						if len(candidate.ConflictsWith()) != 0 {
							found = true
						}
					}
					if !found {
						t.Fatal("no candidate exposed its conflict relation")
					}
					observed = true
					return cli.CompletionContribution{}, nil
				}).From(mode.OptionalDependency()).
					Long("path")
			},
		})
		clitest.Complete(t.Context(), app, cli.CompletionRequest{
			Args: []string{"--mode", "first", "--mode", "second", "--path", ""},
			Cursor: cli.CompletionCursor{
				Argument: 5,
			},
		})
		if !observed {
			t.Fatal("optional completion dependency did not observe conflicting ValueState")
		}
	})
}

type valueEnvironmentPortConfig struct {
	Environment string
	Port        int
}

type valueBaseDerivedDomainConfig struct{ Base, Derived, Domain string }

type valueElementsWholeEntriesWholeMapConfig struct {
	Elements, Whole   []string
	Entries, WholeMap map[string]string
}

type valueBuiltinCustomConfig struct {
	Builtin string
	Custom  string
}

type valueABCConfig struct {
	A string
	B int
	C bool
}

type valueGoodConfig struct{ Good string }

type valueValuesArgConfig struct {
	Values []string
	Arg    int
}

type valueValueConfig struct{ Value int }

type valueRequiredResolvedConfig struct {
	Required string
	Resolved int
}

type valueModeConfig struct{ Mode string }

type valueStringConfig struct{ Value string }

func TestArgumentsDirectImplicitFormAndCustomValue(t *testing.T) {
	t.Parallel()
	type config = valueEnvironmentPortConfig
	environment := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
		values.Value("prod", "production")
	})
	definition := cli.Definition[config]{
		Name: "direct-arguments",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Arguments.Value(&root.Config.Environment, "environment", environment)
			root.Arguments.Int(&root.Config.Port, "port")
		},
	}
	app := clitest.Compile(t, definition)
	form := clitest.RootCommand().ArgumentFormAt(0)
	clitest.Schema(app).Require(t,
		clitest.RootCommand().ArgumentFormsExactly(form),
		form.TermsExactly(clitest.RootCommand().Argument("environment"), clitest.RootCommand().Argument("port")),
	)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"prod", "8080"},
	}).Require(t,
		clitest.RootCommand().Argument("environment").Value().Resolved(),
		clitest.RootCommand().Argument("environment").Value().Text("prod"),
		clitest.RootCommand().Argument("port").Value().Resolved(),
		clitest.RootCommand().Argument("port").Value().Text("8080"),
	)
}

func TestArgumentScalarValueSplitSupportsExplicitFormAndExistingDestination(t *testing.T) {
	t.Parallel()
	type config = valueModeConfig
	modeLanguage := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
		values.Value("safe", "safe")
		values.Value("fast", "fast")
	})

	t.Run("explicit form direct value", func(t *testing.T) {
		t.Parallel()
		definition := cli.Definition[config]{
			Name: "explicit-value",
			Help: cli.HelpConfig{
				Disabled: true,
			},
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
					form.Value(&form.Config.Mode, "mode", modeLanguage)
				})
			},
		}
		app := clitest.Compile(t, definition)
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"fast"},
		}).Require(t,
			clitest.RootCommand().Argument("mode").Value().Resolved(),
			clitest.RootCommand().Argument("mode").Value().Text("fast"),
		)
	})

	t.Run("implicit form existing destination", func(t *testing.T) {
		t.Parallel()
		definition := cli.Definition[config]{
			Name: "implicit-value-for",
			Help: cli.HelpConfig{
				Disabled: true,
			},
			Declare: func(root *cli.CommandBuilder[config]) {
				mode := root.Values.Value(&root.Config.Mode, modeLanguage)
				root.Arguments.ValueFor(mode, "mode")
			},
		}
		app := clitest.Compile(t, definition)
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"safe"},
		}).Require(t,
			clitest.RootCommand().Argument("mode").Value().Resolved(),
			clitest.RootCommand().Argument("mode").Value().Text("safe"),
		)
	})
}

func TestRootCommandDialectInitialUsesAuthoritativeBaseValues(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "dialect-defaults",
	})
	expected := cli.CommandDialect{
		FlagOrder:            cli.FlagOrderBeforeArguments,
		ShortFlagClustering:  cli.ShortFlagClusteringDisabled,
		FlagMatching:         cli.NameMatchingExact,
		CommandMatching:      cli.NameMatchingExact,
		FlagNormalization:    cli.NameNormalizationExact,
		CommandNormalization: cli.NameNormalizationExact,
		DashlessFlags:        cli.DashlessFlagModeDisabled,
		EndOfOptions:         []string{"--"},
		ResponseFiles:        false,
	}
	proof := clitest.Schema(app)
	proof.Require(t, clitest.RootCommand().EffectiveDialect(expected))
	schema := proof.Schema()
	dialect := schema.Commands[schema.Root].Dialect
	if dialect == nil {
		t.Fatal("root dialect = nil")
	}
	dialect.EndOfOptions[0] = "mutated"
	clitest.Schema(app).Require(t, clitest.RootCommand().EffectiveDialect(expected))
}

func TestAuthoringAcceptanceRetainsExecutableAndFiniteDomainValue(t *testing.T) {
	t.Parallel()
	type config = valueBaseDerivedDomainConfig
	definition := cli.Definition[config]{
		Name: "domain",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			base := root.Values.String(&root.Config.Base)
			root.Values.Default(base, "base")
			root.Values.Derive(
				&root.Config.Derived,
				cli.Value.ComputedBy(func(_ context.Context, _ *config, deps cli.Dependencies) (string, error) {
					v, _ := deps.Get(base)
					return v + "-derived", nil
				}).From(base),
			)
			domain := cli.Value.Domain(cli.Value.Assign[string](), func(builder *cli.ValueDomainBuilder[string]) {
				canonical := builder.Value("canonical", "typed")
				builder.Alias("alias", canonical)
			})
			d := root.Values.Value(&root.Config.Domain, domain)
			root.Flags.ValueFor(d).Long("domain")
		},
	}
	app := clitest.Compile(t, definition)
	domain := clitest.RootCommand().Flag("--domain").Value().Domain(cli.ValueDomainUseDestination)
	clitest.Schema(app).Require(t,
		domain.EntriesExactly(domain.Entry("canonical"), domain.Entry("alias")),
		domain.Entry("alias").Canonical(domain.Entry("canonical")),
	)
	baseValue := clitest.RootCommand().ValueAt(0)
	derivedValue := clitest.RootCommand().ValueAt(1)
	domainValue := clitest.RootCommand().Flag("--domain").Value()
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--domain", "alias"},
	}).Require(t,
		baseValue.Resolved(), baseValue.Text("base"),
		derivedValue.Resolved(), derivedValue.NoRepresentation(),
		domainValue.Resolved(), domainValue.Text("canonical"),
	)
}

func TestCountRangeConstructionPreservesNormalizedBounds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		count    cli.CountRange
		min, max int
		bounded  bool
	}{
		{
			name:    "exact",
			count:   cli.Count.Exactly(2),
			min:     2,
			max:     2,
			bounded: true,
		},
		{
			name:    "between",
			count:   cli.Count.Between(1, 3),
			min:     1,
			max:     3,
			bounded: true,
		},
		{
			name:    "at least",
			count:   cli.Count.AtLeast(2),
			min:     2,
			max:     0,
			bounded: false,
		},
		{
			name:    "at most",
			count:   cli.Count.AtMost(4),
			min:     0,
			max:     4,
			bounded: true,
		},
		{
			name:    "optional",
			count:   cli.Count.Optional(),
			min:     0,
			max:     1,
			bounded: true,
		},
		{
			name:    "one or more",
			count:   cli.Count.OneOrMore(),
			min:     1,
			max:     0,
			bounded: false,
		},
		{
			name:    "zero or more",
			count:   cli.Count.ZeroOrMore(),
			min:     0,
			max:     0,
			bounded: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.count.Minimum() != tc.min {
				t.Fatalf("minimum=%d want %d", tc.count.Minimum(), tc.min)
			}
			maximum, ok := tc.count.Maximum()
			if ok != tc.bounded || (ok && maximum != tc.max) {
				t.Fatalf("maximum=%d,%v want %d,%v", maximum, ok, tc.max, tc.bounded)
			}
		})
	}
	invalid := []cli.CountRange{
		cli.Count.Exactly(-1),
		cli.Count.Between(-1, 1),
		cli.Count.AtMost(-1),
		cli.Count.Between(3, 2),
	}
	for i, count := range invalid {
		_, err := cli.Compile(cli.Definition[struct{ Values []string }]{
			Name: "bad-count",
			Declare: func(root *cli.CommandBuilder[struct{ Values []string }]) {
				root.Arguments.Form(func(form *cli.ArgumentFormBuilder[struct{ Values []string }]) {
					form.Strings(&form.Config.Values, "value", count)
				})
			},
		})
		if err == nil {
			t.Fatalf("invalid count[%d] compiled", i)
		}
	}
}

func TestEnvironmentDelimiterOptionsRetainContributionGranularity(t *testing.T) {
	t.Parallel()
	type config = valueElementsWholeEntriesWholeMapConfig
	definition := cli.Definition[config]{
		Name: "env-granularity",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			e := root.Values.Sequence(&root.Config.Elements, cli.Value.String())
			w := root.Values.Sequence(&root.Config.Whole, cli.Value.String())
			ke := root.Values.Keyed(&root.Config.Entries, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
			kw := root.Values.Keyed(&root.Config.WholeMap, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
			root.Values.Environment(e, "ELEMENTS").Elements(",")
			root.Values.Environment(w, "WHOLE").WholeSequence(",")
			root.Values.Environment(ke, "ENTRIES").KeyedEntries(";")
			root.Values.Environment(kw, "WHOLE_MAP").WholeKeyed(";")
		},
	}
	app := clitest.Compile(t, definition)
	request := cli.InspectionRequest{
		Env: cli.Env{
			{
				Name:  "ELEMENTS",
				Value: "a,b",
			},
			{
				Name:  "WHOLE",
				Value: "c,d",
			},
			{
				Name:  "ENTRIES",
				Value: "x=1;y=2",
			},
			{
				Name:  "WHOLE_MAP",
				Value: "m=3;n=4",
			},
		},
	}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.RootCommand().Environment("ELEMENTS").Value().CandidateAt(0).ElementCount(2),
		clitest.RootCommand().Environment("WHOLE").Value().CandidateAt(0).ElementCount(2),
		clitest.RootCommand().Environment("ENTRIES").Value().CandidatesExactly(
			clitest.RootCommand().Environment("ENTRIES").Value().CandidateAt(0),
			clitest.RootCommand().Environment("ENTRIES").Value().CandidateAt(1),
		),
		clitest.RootCommand().Environment("ENTRIES").Value().CandidateAt(0).EntryCount(1),
		clitest.RootCommand().Environment("ENTRIES").Value().CandidateAt(1).EntryCount(1),
		clitest.RootCommand().Environment("WHOLE_MAP").Value().CandidateAt(0).EntryCount(2),
	)
	definitionUnderTest := cli.Definition[struct{ V []string }]{
		Name: "bad-delimiter",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[struct{ V []string }]) {
			d := root.Values.Sequence(&root.Config.V, cli.Value.String())
			root.Values.Environment(d, "V").Elements("")
		},
	}
	clitest.CompileFails(t, definitionUnderTest).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestSourceOptionsRetainExactDeclaredMetadata(t *testing.T) {
	t.Parallel()
	type config = valueModeConfig
	sources := []cli.SourceOption{
		cli.Source.Key("logical-key"),
		cli.Source.Location("config/tool.args"),
		cli.Source.Text("--mode=fast"),
		cli.Source.Sensitive(),
	}
	definition := cli.Definition[config]{
		Name: "sources",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Mode).Long("mode").FixedValue("fast", sources...).Sensitive()
			sources[0] = cli.Source.Key("mutated")
			sources[1] = cli.Source.Location("mutated")
			sources[2] = cli.Source.Text("mutated")
		},
	}
	app := clitest.Compile(t, definition)
	fixed := clitest.RootCommand().Flag("--mode").Spelling("--mode").FixedValue()
	clitest.Schema(app).Require(t,
		fixed.Sensitive(),
		fixed.Redacted(),
		fixed.Text(""),
	)
	mode := clitest.RootCommand().Flag("--mode").Value()
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--mode"},
	}).Require(t,
		mode.Resolved(),
		mode.Redacted(),
	)
}

func TestValueLanguageNormalizationAcceptsOnlyExactCommittedForms(t *testing.T) {
	t.Parallel()
	type config = valueBuiltinCustomConfig
	valid := cli.ValueCodec[string]{
		Name:     "custom",
		Parse:    func(s string) (string, error) { return s, nil },
		Format:   func(s string) string { return s },
		Snapshot: cli.Value.Assign[string](),
	}
	_ = clitest.Compile(t, cli.Definition[config]{
		Name: "languages",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Values.String(&root.Config.Builtin)
			root.Values.Value(&root.Config.Custom, valid)
		},
	})
	invalid := []cli.ValueCodec[string]{
		{
			Name:   "",
			Parse:  valid.Parse,
			Format: valid.Format,
		},
		{
			Name:   "x",
			Format: valid.Format,
		},
		{
			Name:  "x",
			Parse: valid.Parse,
		},
	}
	for _, codec := range invalid {
		clitest.CompileFails(t, cli.Definition[config]{
			Name: "invalid-language",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Values.Value(&root.Config.Custom, codec)
			},
		}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
	}
}

func TestValueDomainBuildsOneImmutableStateWithExactAliasTargets(t *testing.T) {
	t.Parallel()
	type config = valueStringConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "domain",
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := cli.Value.Domain(cli.Value.Assign[string](), func(builder *cli.ValueDomainBuilder[string]) {
				canonical := builder.Value("canonical", "typed")
				builder.Alias("alias", canonical)
			})
			root.Values.Value(&root.Config.Value, domain)
		},
	})
	domain := clitest.RootCommand().ValueAt(0).Domain(cli.ValueDomainUseDestination)
	clitest.Schema(app).Require(t,
		domain.EntriesExactly(domain.Entry("canonical"), domain.Entry("alias")),
		domain.Entry("alias").Canonical(domain.Entry("canonical")),
	)
}

func TestValueDomainRetainsInvalidStateIssuesForCommitValidation(t *testing.T) {
	t.Parallel()
	type config = valueStringConfig
	cases := []func(*cli.ValueDomainBuilder[string]){
		func(b *cli.ValueDomainBuilder[string]) { b.Value("", "x") },
		func(b *cli.ValueDomainBuilder[string]) { b.Value("same", "a"); b.Value("same", "b") },
		func(b *cli.ValueDomainBuilder[string]) { b.Alias("alias", cli.DomainValueDeclaration[string]{}) },
	}
	for _, declare := range cases {
		clitest.CompileFails(t, cli.Definition[config]{
			Name: "bad-domain",
			Declare: func(root *cli.CommandBuilder[config]) {
				domain := cli.Value.Domain(cli.Value.Assign[string](), declare)
				root.Values.Value(&root.Config.Value, domain)
			},
		}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
	}
}

func TestValueDomainPropagatesCallbackPanicAndStillExpiresRetainedBuilder(t *testing.T) {
	t.Parallel()
	panicValue := errors.New("domain panic")
	func() {
		defer func() {
			recovered, ok := recover().(error)
			if !ok || !errors.Is(recovered, panicValue) {
				t.Fatalf("panic not preserved: %v", recovered)
			}
		}()
		_, _ = cli.Compile(cli.Definition[struct{ V string }]{
			Name: "panic-domain",
			Declare: func(_ *cli.CommandBuilder[struct{ V string }]) {
				_ = cli.Value.Domain(cli.Value.Assign[string](), func(*cli.ValueDomainBuilder[string]) { panic(panicValue) })
			},
		})
	}()
	var retained *cli.ValueDomainBuilder[string]
	_, err := cli.Compile(cli.Definition[struct{ V string }]{
		Name: "domain-expiry",
		Declare: func(root *cli.CommandBuilder[struct{ V string }]) {
			domain := cli.Value.Domain(cli.Value.Assign[string](), func(b *cli.ValueDomainBuilder[string]) { retained = b; b.Value("x", "x") })
			root.Values.Value(&root.Config.V, domain)
		},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	panicked := false
	func() { defer func() { panicked = recover() != nil }(); retained.Value("late", "late") }()
	if !panicked {
		t.Fatal("retained domain builder remained usable")
	}
}

func TestAuthoringTransactionStagesDestinationsInCallOrder(t *testing.T) {
	t.Parallel()
	type config = valueABCConfig
	definition := cli.Definition[config]{
		Name: "value-order",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			a := root.Values.String(&root.Config.A)
			root.Values.Default(a, "first")
			root.Flags.ValueFor(a).Long("a")
			b := root.Values.Int(&root.Config.B)
			root.Values.Default(b, 2)
			root.Flags.ValueFor(b).Long("b")
			c := root.Values.Bool(&root.Config.C)
			root.Values.Default(c, true)
			root.Flags.ValueFor(c).Long("c")
		},
	}
	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	text := root.Flag("--a").Value()
	integer := root.Flag("--b").Value()
	boolean := root.Flag("--c").Value()
	clitest.Schema(app).Require(t,
		root.ValuesExactly(root.ValueAt(0), root.ValueAt(1), root.ValueAt(2)),
		text.Kind(cli.ValueDestinationScalar),
		text.Type(cli.ValueType{
			Kind: cli.ValueKindString,
		}),
		text.StaticDefault().Scalar().Text("first"),
		integer.Kind(cli.ValueDestinationScalar),
		integer.Type(cli.ValueType{
			Kind: cli.ValueKindInt,
		}),
		integer.StaticDefault().Scalar().Text("2"),
		boolean.Kind(cli.ValueDestinationScalar),
		boolean.Type(cli.ValueType{
			Kind: cli.ValueKindBool,
		}),
		boolean.StaticDefault().Scalar().Text("true"),
	)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		text.Text("first"),
		integer.Text("2"),
		boolean.Text("true"),
	)
}

func TestAuthoringTransactionRetainedBuilderExpiresWithOwningCallback(t *testing.T) {
	t.Parallel()
	type config = valueStringConfig
	var retained *cli.Values[config]
	_, err := cli.Compile(cli.Definition[config]{
		Name: "builder-expiry",
		Declare: func(root *cli.CommandBuilder[config]) {
			retained = &root.Values
			root.Values.String(&root.Config.Value)
		},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	panicked := false
	func() { defer func() { panicked = recover() != nil }(); retained.String(&retained.Config.Value) }()
	if !panicked {
		t.Fatal("retained values builder remained usable")
	}
}

func TestDestinationOptionalRetainsExactDestinationRelationship(t *testing.T) {
	t.Parallel()
	type config = valueStringConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "optional-dependency",
		Declare: func(root *cli.CommandBuilder[config]) {
			d := root.Values.String(&root.Config.Value)
			root.Subcommands.CommandsFrom(
				func(context.Context, *cli.CommandBatch[config]) error { return nil },
				d.OptionalDependency(),
			)
		},
	})
	clitest.Schema(app).Require(t,
		clitest.RootCommand().ProviderAt(0).DependenciesExactly(clitest.RootCommand().ValueAt(0)),
	)
}

func TestAuthoringTransactionRejectsInvalidDestinationAtomically(t *testing.T) {
	t.Parallel()
	type config = valueGoodConfig
	clitest.CompileFails(t, cli.Definition[config]{
		Name: "bad-destination",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Values.String(&root.Config.Good)
			root.Values.String(nil)
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestAuthoringValueAndGrammarOptionsRetainExactArguments(t *testing.T) {
	t.Parallel()
	type config = valueValuesArgConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "options",
		Declare: func(root *cli.CommandBuilder[config]) {
			d := root.Values.Sequence(&root.Config.Values, cli.Value.String())
			root.Flags.Sequence(d).Occurrences(cli.Count.Between(1, 3)).ValuesPerOccurrence(cli.Count.Exactly(2)).DelimitedBy(",").TerminatedBy(";").HyphenValues(cli.HyphenValueAllow).Placement(cli.FlagPlacementSubtree).ValueName("PATH").Description("desc").
				Long("path")
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Int(
					&form.Config.Arg,
					"arg",
				).HyphenValues(cli.HyphenValueNegativeNumbers).Description("argument")
			})
		},
	})
	flag := clitest.RootCommand().Flag("--path")
	arg := clitest.RootCommand().Argument("arg")
	clitest.Schema(app).Require(t,
		flag.Occurrences(cli.Count.Between(1, 3)),
		flag.ValuesPerOccurrence(cli.Count.Exactly(2)),
		flag.Placement(cli.FlagPlacementSubtree),
		flag.ValueName("PATH"),
		flag.Description("desc"),
		arg.HyphenValues(cli.HyphenValueNegativeNumbers),
		arg.Description("argument"),
	)
}

func TestContributionPreservesValueOmissionAndMalformedZeroAsDistinctStates(t *testing.T) {
	t.Parallel()
	type config = valueValueConfig

	definition := cli.Definition[config]{
		Name: "contributed",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Values.OptionalDerive(
				&root.Config.Value,
				cli.Value.OptionallyComputedBy(
					func(context.Context, *config, cli.Dependencies) (cli.ValueContribution[int], error) {
						return cli.ContributeValue(0), nil
					},
				),
			)
		},
	}
	contributed := clitest.Compile(t, definition)
	clitest.Inspect(t.Context(), contributed, cli.InspectionRequest{}).Require(t,
		clitest.RootCommand().ValueAt(0).State(cli.ResolutionStateResolved),
	)

	definitionUnderTest := cli.Definition[config]{
		Name: "omitted",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Values.OptionalDerive(
				&root.Config.Value,
				cli.Value.OptionallyComputedBy(
					func(context.Context, *config, cli.Dependencies) (cli.ValueContribution[int], error) {
						return cli.OmitValue[int](), nil
					},
				),
			)
		},
	}
	omitted := clitest.Compile(t, definitionUnderTest)
	clitest.Inspect(t.Context(), omitted, cli.InspectionRequest{}).Require(t,
		clitest.RootCommand().ValueAt(0).State(cli.ResolutionStateAbsent),
	)

	definitionUnderTest2 := cli.Definition[config]{
		Name: "malformed",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Values.OptionalDerive(
				&root.Config.Value,
				cli.Value.OptionallyComputedBy(
					func(context.Context, *config, cli.Dependencies) (cli.ValueContribution[int], error) {
						return cli.ValueContribution[int]{}, nil
					},
				),
			)
		},
	}
	malformed := clitest.Compile(t, definitionUnderTest2)
	clitest.Inspect(t.Context(), malformed, cli.InspectionRequest{}).Require(t,
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticValueComputationFailure).Present()),
	)
}

func TestValueDirectiveRejectsForeignDestinationAtomically(t *testing.T) {
	t.Parallel()
	type config = valueStringConfig
	var foreign cli.Destination[config, string]
	_ = clitest.Compile(t, cli.Definition[config]{
		Name: "owner",
		Declare: func(root *cli.CommandBuilder[config]) {
			foreign = root.Values.String(&root.Config.Value)
		},
	})
	clitest.CompileFails(t, cli.Definition[config]{
		Name: "foreign",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Values.Default(foreign, "x")
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestResolvedFactDeclarationUsesDestinationContractDirectly(t *testing.T) {
	t.Parallel()
	type config = valueRequiredResolvedConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "facts",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Facts.Require(&root.Config.Required)
			resolved := root.Facts.Resolve(
				&root.Config.Resolved,
				func(context.Context, *config, cli.Dependencies) (int, error) { return 1, nil },
			)
			root.VisibleWhen(root.When.Resolved(resolved))
			root.Rule.Equal(resolved, 1)
			_ = resolved.Destination()
		},
	})
	root := clitest.RootCommand()
	resolved := root.FactAt(1).Value()
	condition := root.ConditionAt(0)
	rule := root.RuleAt(0)
	clitest.Schema(app).Require(t,
		root.FactsExactly(root.FactAt(0), root.FactAt(1)),
		root.ConditionsExactly(condition),
		condition.Effect(cli.ConditionEffectVisibility),
		condition.Expression().Destination(resolved),
		root.RulesExactly(rule),
		rule.Kind(cli.RuleEqual),
		rule.SubjectsExactly(resolved),
	)
}

func TestDialectDirectivesStageExplicitResolvedInputsWithoutChoosingRootDefaults(t *testing.T) {
	t.Parallel()
	tokens := []string{"--", "---"}
	app := valueMustCompilePublic(t, cli.Definition[struct{}]{
		Name: "dialect",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Dialect.FlagOrder(cli.FlagOrderInterspersed)
			root.Dialect.ShortFlagClustering(cli.ShortFlagClusteringEnabled)
			root.Dialect.FlagMatching(cli.NameMatchingUniquePrefix)
			root.Dialect.CommandMatching(cli.NameMatchingExact)
			root.Dialect.FlagNormalization(cli.NameNormalizationASCIICaseFold)
			root.Dialect.CommandNormalization(cli.NameNormalizationExact)
			root.Dialect.DashlessFlags(cli.DashlessFlagModeInitialCluster)
			root.Dialect.EndOfOptions(tokens...)
			root.Dialect.ResponseFiles(true)
			root.Dialect.DefaultFlagPlacement(cli.FlagPlacementSubtree)
			tokens[0] = "mutated"
		},
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().EffectiveDialect(cli.CommandDialect{
		FlagOrder:            cli.FlagOrderInterspersed,
		ShortFlagClustering:  cli.ShortFlagClusteringEnabled,
		FlagMatching:         cli.NameMatchingUniquePrefix,
		CommandMatching:      cli.NameMatchingExact,
		FlagNormalization:    cli.NameNormalizationASCIICaseFold,
		CommandNormalization: cli.NameNormalizationExact,
		DashlessFlags:        cli.DashlessFlagModeInitialCluster,
		EndOfOptions:         []string{"--", "---"},
		ResponseFiles:        true,
	}))
}

func TestDialectDirectiveRejectsUnknownEnumeratedValue(t *testing.T) {
	t.Parallel()
	clitest.CompileFails(t, cli.Definition[struct{}]{
		Name: "bad-dialect",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Dialect.FlagOrder(cli.FlagOrderUnknown)
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestFlagsDirectCustomValueCreatesCanonicalDestination(t *testing.T) {
	t.Parallel()
	type config = valueModeConfig
	mode := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
		values.Value("safe", "safe")
		values.Value("fast", "fast")
	})
	definition := cli.Definition[config]{
		Name: "direct-flag",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.Value(&root.Config.Mode, mode).Long("mode")
		},
	}
	app := clitest.Compile(t, definition)
	flag := clitest.RootCommand().Flag("--mode")
	clitest.Schema(app).Require(t,
		clitest.RootCommand().FlagsExactly(flag),
		clitest.RootCommand().ValuesExactly(flag.Value()),
	)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--mode", "fast"},
	}).Require(t,
		flag.Value().Resolved(),
		flag.Value().Text("fast"),
	)
}

func TestKeyedBatchRejectsNonReflexiveComparableKey(t *testing.T) {
	t.Parallel()
	type config struct{ Labels map[float64]string }
	nanCodec := cli.ValueCodec[float64]{
		Name:     "nan-key",
		Parse:    func(string) (float64, error) { return math.NaN(), nil },
		Format:   func(value float64) string { return strconv.FormatFloat(value, 'g', -1, 64) },
		Snapshot: cli.Value.Assign[float64](),
	}
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			labels := root.Values.Keyed(&root.Config.Labels, cli.Value.Keyed(nanCodec, cli.Value.String()))
			root.Flags.Keyed(labels).Long("label")
		},
	}
	app := clitest.Compile(t, definition)
	value := clitest.RootCommand().Flag("--label").Value()
	candidate := value.CandidateAt(0)
	diagnostic := clitest.Diagnostic(cli.DiagnosticInputParserFailure)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--label", "nan=x"},
	}).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(diagnostic.Present()),
		value.Failed(),
		value.CandidatesExactly(candidate),
		candidate.Disposition(cli.ValueCandidateDispositionRejected),
	)
}

type valueDerivedSensitivityConfig struct {
	Secret  string
	Derived string
}

func TestDerivedValuePropagatesSensitivityFromEffectiveDependency(t *testing.T) {
	t.Parallel()

	type config = valueDerivedSensitivityConfig
	declare := func(root *cli.CommandBuilder[config]) {
		secret := root.Values.String(&root.Config.Secret).Sensitive()
		root.Flags.ValueFor(secret).Long("secret").Sensitive()
		derive := func(_ context.Context, _ *config, dependencies cli.Dependencies) (string, error) {
			value, ok := dependencies.Get(secret)
			if !ok {
				return "", errors.New("secret dependency unavailable")
			}
			return "derived:" + value, nil
		}
		producer := cli.Value.ComputedBy(derive).From(secret)
		derived := root.Values.Derive(&root.Config.Derived, producer)
		root.Rule.Require(derived)
	}
	definition := cli.Definition[config]{
		Name: "alienctl",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: declare,
	}
	app := clitest.Compile(t, definition)
	derived := clitest.RootCommand().ValueAt(1)
	request := cli.InspectionRequest{
		Args: []string{"--secret", "classified"},
	}

	clitest.Inspect(t.Context(), app, request).Require(t,
		derived.Resolved(),
		derived.Sensitive(),
		derived.Redacted(),
	)
}

func TestDerivedValueDoesNotInventSensitivityForNonsensitiveDependency(t *testing.T) {
	t.Parallel()

	type config = valueDerivedSensitivityConfig
	declare := func(root *cli.CommandBuilder[config]) {
		input := root.Values.String(&root.Config.Secret)
		root.Flags.ValueFor(input).Long("input")
		derive := func(_ context.Context, _ *config, dependencies cli.Dependencies) (string, error) {
			value, ok := dependencies.Get(input)
			if !ok {
				return "", errors.New("input dependency unavailable")
			}
			return "derived:" + value, nil
		}
		producer := cli.Value.ComputedBy(derive).From(input)
		derived := root.Values.Derive(&root.Config.Derived, producer)
		root.Rule.Require(derived)
	}
	definition := cli.Definition[config]{
		Name: "alienctl",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: declare,
	}
	app := clitest.Compile(t, definition)
	derived := clitest.RootCommand().ValueAt(1)
	request := cli.InspectionRequest{
		Args: []string{"--input", "ordinary"},
	}

	proof := clitest.Inspect(t.Context(), app, request)
	proof.Require(t,
		derived.Resolved(),
	)
	inspection, err := proof.Inspection()
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	root := inspection.Commands[inspection.Selection.Command.Root]
	if len(root.Values) < 2 {
		t.Fatalf("root values = %#v, want derived destination", root.Values)
	}
	if inspection.Values[root.Values[1]].Sensitive {
		t.Fatal("nonsensitive dependency made derived value sensitive")
	}
}

type valueShortcutFixedDefaultRootConfig struct{}

type valueShortcutFixedDefaultChildConfig struct {
	Mode string
	Name string
}

func TestShortcutFixedInputAndFallbackDefaultKeepDistinctProvenance(t *testing.T) {
	t.Parallel()

	type rootConfig = valueShortcutFixedDefaultRootConfig
	type childConfig = valueShortcutFixedDefaultChildConfig
	var ran childConfig
	declareChild := func(command *cli.CommandBuilder[childConfig]) {
		mode := command.Values.String(&command.Config.Mode)
		name := command.Values.String(&command.Config.Name)
		command.Flags.ValueFor(mode).Long("mode")
		command.Flags.ValueFor(name).Long("name")
		shortcut := cli.CommandShortcut{
			Name: "quick",
			Args: []string{"--mode", "fixed"},
		}
		command.Shortcut(shortcut).Default(name, "fallback")
	}
	runChild := func(_ context.Context, config *childConfig) error {
		ran = *config
		return nil
	}
	child := cli.Command[childConfig]{
		Name:    "serve",
		Declare: declareChild,
		Run:     runChild,
	}
	declareRoot := func(root *cli.CommandBuilder[rootConfig]) {
		root.Subcommands.Command(child)
	}
	definition := cli.Definition[rootConfig]{
		Name:    "alienctl",
		Declare: declareRoot,
	}
	app := clitest.Compile(t, definition)
	command := clitest.Command("serve")
	mode := command.Flag("--mode").Value()
	name := command.Flag("--name").Value()
	shortcutDefaultProducer := name.ProducerAt(1)
	shortcutDefault := name.CandidateAt(1)
	inspectionRequest := cli.InspectionRequest{
		Args: []string{"quick", "--name", "explicit"},
	}
	runRequest := cli.RunRequest{
		Args: inspectionRequest.Args,
	}

	clitest.Inspect(t.Context(), app, inspectionRequest).Require(t,
		clitest.InspectionReady(),
		mode.Resolved(),
		mode.Text("fixed"),
		name.Resolved(),
		name.Text("explicit"),
		shortcutDefaultProducer.Kind(cli.ValueProducerKindShortcutDefault),
		shortcutDefault.Disposition(cli.ValueCandidateDispositionSuppressed),
		shortcutDefault.ProducedBy(shortcutDefaultProducer),
	)
	if err := app.Run(t.Context(), runRequest); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if ran.Mode != "fixed" || ran.Name != "explicit" {
		t.Fatalf("Run config = %#v, want fixed mode and explicit name", ran)
	}
}
