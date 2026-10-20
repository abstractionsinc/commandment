package conformance

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

type concurrencyValueConfig struct{ Value string }

type concurrencyOperationTag struct {
	worker    int
	operation string
}

type concurrencyOperationContextKey struct{}

type concurrencyRunObservation struct {
	tag      concurrencyOperationTag
	config   *concurrencyValueConfig
	original string
	current  string
}

type concurrencyRunResult struct {
	worker int
	err    error
}

type inspectionConcurrencyConfig struct{}

func TestConcurrentIndependentPublicOperationsRemainOperationLocal(t *testing.T) {
	t.Parallel()
	type config = concurrencyValueConfig
	type operationTag = concurrencyOperationTag
	type operationContextKey = concurrencyOperationContextKey
	type runObservation = concurrencyRunObservation
	type runResult = concurrencyRunResult

	const operations = 8
	operationKey := operationContextKey{}
	providerCalls := make(map[operationTag]int)
	completerCalls := make(map[operationTag]int)
	var accountingMu sync.Mutex
	recordCall := func(calls map[operationTag]int, ctx context.Context) {
		tag, ok := ctx.Value(operationKey).(operationTag)
		if !ok {
			tag = operationTag{
				worker:    -1,
				operation: "missing operation context",
			}
		}
		accountingMu.Lock()
		calls[tag]++
		accountingMu.Unlock()
	}

	runStarted := make(chan runObservation, operations)
	runFinished := make(chan runObservation, operations)
	releaseRuns := make(chan struct{})
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(ctx context.Context, batch *cli.CommandBatch[config]) error {
				recordCall(providerCalls, ctx)
				batch.Command(cli.Command[config]{
					Name: "remote",
				})
				return nil
			})
			root.Flags.String(&root.Config.Value).CompleteBy(func(ctx context.Context, cfg *config, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				recordCall(completerCalls, ctx)
				if cfg.Value != "" {
					return cli.CompletionContribution{}, fmt.Errorf(
						"completion observed contaminated config value %q",
						cfg.Value,
					)
				}
				return cli.CompletionContribution{
					Candidates: []cli.CompletionCandidate{{Value: "stable"}},
				}, nil
			}).
				Long("value")
		},
		Run: func(ctx context.Context, cfg *config) error {
			tag, _ := ctx.Value(operationKey).(operationTag)
			original := cfg.Value
			cfg.Value = "mutated:" + original
			runStarted <- runObservation{
				tag:      tag,
				config:   cfg,
				original: original,
				current:  cfg.Value,
			}
			<-releaseRuns
			runFinished <- runObservation{
				tag:      tag,
				config:   cfg,
				original: original,
				current:  cfg.Value,
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	// Admit every Run and keep its mutable handed-off configuration live while
	// the other public operations execute. This is a causal overlap barrier, not
	// a scheduler-timing assumption.
	runResults := make(chan runResult, operations)
	for worker := range operations {
		ctx := context.WithValue(t.Context(), operationKey, operationTag{
			worker:    worker,
			operation: "Run",
		})
		go func(worker int, ctx context.Context) {
			runResults <- runResult{
				worker: worker,
				err: app.Run(ctx, cli.RunRequest{
					Args: []string{"--value", fmt.Sprintf("run-%d", worker)},
				}),
			}
		}(worker, ctx)
	}

	started := make(map[int]runObservation, operations)
	configs := make(map[*config]int, operations)
	for len(started) < operations {
		select {
		case result := <-runResults:
			t.Fatalf("Run %d returned before overlap barrier: %v", result.worker, result.err)
		case observation := <-runStarted:
			worker := observation.tag.worker
			if observation.tag.operation != "Run" || worker < 0 || worker >= operations {
				t.Fatalf("Run admitted without operation-local context: %#v", observation.tag)
			}
			if _, exists := started[worker]; exists {
				t.Fatalf("Run %d admitted more than once", worker)
			}
			wantOriginal := fmt.Sprintf("run-%d", worker)
			wantCurrent := "mutated:" + wantOriginal
			if observation.original != wantOriginal || observation.current != wantCurrent {
				t.Fatalf(
					"Run %d config = original %q current %q, want %q/%q",
					worker,
					observation.original,
					observation.current,
					wantOriginal,
					wantCurrent,
				)
			}
			if previous, exists := configs[observation.config]; exists {
				t.Fatalf("Runs %d and %d shared configuration storage %p", previous, worker, observation.config)
			}
			started[worker] = observation
			configs[observation.config] = worker
		}
	}

	startMixed := make(chan struct{})
	errs := make(chan error, operations)
	var wg sync.WaitGroup
	for worker := range operations {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			<-startMixed

			inspectCtx := context.WithValue(
				t.Context(),
				operationKey,
				operationTag{
					worker:    worker,
					operation: "Inspect",
				},
			)
			inspection, err := app.Inspect(inspectCtx, cli.InspectionRequest{
				Args: []string{"--value", fmt.Sprintf("inspect-%d", worker)},
			})
			if err != nil || inspection.Status != cli.InspectionStatusReady {
				errs <- concurrentOperationError{"Inspect", err}
				return
			}

			commandCtx := context.WithValue(
				t.Context(),
				operationKey,
				operationTag{
					worker:    worker,
					operation: "command Complete",
				},
			)
			commandCompletion, err := app.Complete(commandCtx, cli.CompletionRequest{
				Args: []string{""},
				Cursor: cli.CompletionCursor{
					Argument: 0,
				},
			})
			if err != nil || !completionContainsValue(commandCompletion, "remote") {
				errs <- concurrentOperationError{"command Complete", err}
				return
			}

			valueCtx := context.WithValue(
				t.Context(),
				operationKey,
				operationTag{
					worker:    worker,
					operation: "value Complete",
				},
			)
			valueCompletion, err := app.Complete(valueCtx, cli.CompletionRequest{
				Args: []string{"--value", ""},
				Cursor: cli.CompletionCursor{
					Argument: 1,
				},
			})
			if err != nil || !completionContainsValue(valueCompletion, "stable") {
				errs <- concurrentOperationError{"value Complete", err}
				return
			}
			if schema := app.Schema(); schema.Root == 0 {
				errs <- concurrentOperationError{"Schema", nil}
				return
			}
			reference, err := app.Reference(cli.ReferenceOptions{})
			if err != nil || reference.Root == 0 {
				errs <- concurrentOperationError{"Reference", err}
			}
		}(worker)
	}
	close(startMixed)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	if t.Failed() {
		close(releaseRuns)
		return
	}

	close(releaseRuns)
	for range operations {
		observation := <-runFinished
		worker := observation.tag.worker
		wantCurrent := "mutated:" + fmt.Sprintf("run-%d", worker)
		if observation.current != wantCurrent {
			t.Fatalf(
				"Run %d config changed while other public operations overlapped: got %q, want %q",
				worker,
				observation.current,
				wantCurrent,
			)
		}
		if startedObservation, ok := started[worker]; !ok || startedObservation.config != observation.config {
			t.Fatalf(
				"Run %d changed configuration identity across admission: %p -> %p",
				worker,
				startedObservation.config,
				observation.config,
			)
		}
	}
	for range operations {
		result := <-runResults
		if result.err != nil {
			t.Fatalf("Run %d error = %v", result.worker, result.err)
		}
	}

	// A completed Run's mutation must not contaminate later public-operation
	// observations from the same compiled Application.
	postWorker := operations
	postInspectCtx := context.WithValue(
		t.Context(),
		operationKey,
		operationTag{
			worker:    postWorker,
			operation: "Inspect",
		},
	)
	postInspection, err := app.Inspect(postInspectCtx, cli.InspectionRequest{Args: []string{"--value", "post"}})
	if err != nil || postInspection.Status != cli.InspectionStatusReady {
		t.Fatalf("post-Run Inspect = status %v err %v", postInspection.Status, err)
	}
	postCommandCtx := context.WithValue(
		t.Context(),
		operationKey,
		operationTag{
			worker:    postWorker,
			operation: "command Complete",
		},
	)
	postCommandCompletion, err := app.Complete(postCommandCtx, cli.CompletionRequest{
		Args:   []string{""},
		Cursor: cli.CompletionCursor{Argument: 0},
	})
	if err != nil || !completionContainsValue(postCommandCompletion, "remote") {
		t.Fatalf("post-Run command Complete = %#v err %v", postCommandCompletion, err)
	}
	postValueCtx := context.WithValue(
		t.Context(),
		operationKey,
		operationTag{
			worker:    postWorker,
			operation: "value Complete",
		},
	)
	postValueCompletion, err := app.Complete(postValueCtx, cli.CompletionRequest{
		Args:   []string{"--value", ""},
		Cursor: cli.CompletionCursor{Argument: 1},
	})
	if err != nil || !completionContainsValue(postValueCompletion, "stable") {
		t.Fatalf("post-Run value Complete = %#v err %v", postValueCompletion, err)
	}
	if schema := app.Schema(); schema.Root == 0 {
		t.Fatal("post-Run Schema() returned an invalid root")
	}
	if reference, err := app.Reference(cli.ReferenceOptions{}); err != nil || reference.Root == 0 {
		t.Fatalf("post-Run Reference() = root %v err %v", reference.Root, err)
	}

	accountingMu.Lock()
	providerSnapshot := make(map[operationTag]int, len(providerCalls))
	maps.Copy(providerSnapshot, providerCalls)
	completerSnapshot := make(map[operationTag]int, len(completerCalls))
	maps.Copy(completerSnapshot, completerCalls)
	accountingMu.Unlock()

	expectedProviders := make(map[operationTag]int, operations*4+3)
	expectedCompleters := make(map[operationTag]int, operations+1)
	for worker := range operations {
		expectedProviders[operationTag{
			worker:    worker,
			operation: "Run",
		}] = 1
		expectedProviders[operationTag{
			worker:    worker,
			operation: "Inspect",
		}] = 1
		expectedProviders[operationTag{
			worker:    worker,
			operation: "command Complete",
		}] = 1
		expectedProviders[operationTag{
			worker:    worker,
			operation: "value Complete",
		}] = 1
		expectedCompleters[operationTag{
			worker:    worker,
			operation: "value Complete",
		}] = 1
	}
	expectedProviders[operationTag{
		worker:    postWorker,
		operation: "Inspect",
	}] = 1
	expectedProviders[operationTag{
		worker:    postWorker,
		operation: "command Complete",
	}] = 1
	expectedProviders[operationTag{
		worker:    postWorker,
		operation: "value Complete",
	}] = 1
	expectedCompleters[operationTag{
		worker:    postWorker,
		operation: "value Complete",
	}] = 1
	if len(providerSnapshot) != len(expectedProviders) {
		t.Fatalf("provider accounting keys = %#v, want %#v", providerSnapshot, expectedProviders)
	}
	for tag, want := range expectedProviders {
		if got := providerSnapshot[tag]; got != want {
			t.Fatalf("provider calls for %#v = %d, want %d; all = %#v", tag, got, want, providerSnapshot)
		}
	}
	if len(completerSnapshot) != len(expectedCompleters) {
		t.Fatalf("completer accounting keys = %#v, want %#v", completerSnapshot, expectedCompleters)
	}
	for tag, want := range expectedCompleters {
		if got := completerSnapshot[tag]; got != want {
			t.Fatalf("completer calls for %#v = %d, want %d; all = %#v", tag, got, want, completerSnapshot)
		}
	}

	// Mutating one detached public snapshot cannot reconfigure the compiled
	// application or a later snapshot.
	first := app.Schema()
	if len(first.CommandOrder) == 0 {
		t.Fatal("Schema() returned empty command order")
	}
	first.CommandOrder[0] = 0
	second := app.Schema()
	if len(second.CommandOrder) == 0 || second.CommandOrder[0] == 0 {
		t.Fatalf("later Schema() was contaminated by retained snapshot mutation: %#v", second.CommandOrder)
	}
}

type concurrentOperationError struct {
	operation string
	err       error
}

func (err concurrentOperationError) Error() string {
	if err.err != nil {
		return err.operation + ": " + err.err.Error()
	}
	return err.operation + ": unexpected result"
}

func completionContainsValue(result cli.Completion, want string) bool {
	for _, item := range result.Items {
		if item.Value == want {
			return true
		}
	}
	return false
}

func TestInspectionDoesNotEnterExecutionOrSignalRuntime(t *testing.T) {
	t.Parallel()
	type config = inspectionConcurrencyConfig
	middlewareCalled := false
	handlerCalled := false
	noticeDelivered := false
	source := &inspectionCountingSignalSource{}
	app, err := cli.Compile(cli.Definition[config]{
		Name: "commandment",
		Declare: func(root *cli.CommandBuilder[config]) {
			registeredSignal := root.Signals.Source(source.Source()).Signal("counted")
			root.Signals.Policy(
				root.Signals.Match.On(registeredSignal).During(cli.SignalPhaseExecution).Each(),
				root.Signals.Action.Ignore(),
			)
			root.Advisory.Always(cli.Advisory{
				Code:     "observed",
				Severity: cli.NoticeSeverityWarning,
				Message:  "observed",
			})
		},
		Middleware: []cli.Middleware[config]{func(ctx context.Context, _ *config, next cli.MiddlewareNext) error {
			middlewareCalled = true
			return next(ctx)
		}},
		Run: func(context.Context, *config) error {
			handlerCalled = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		clitest.InspectionReady(),
		clitest.NoticesExactly(clitest.Notice(cli.NoticeAdvisory).Advisory("observed")),
	)
	if middlewareCalled || handlerCalled || noticeDelivered {
		t.Fatalf(
			"Inspection entered execution side effects: middleware=%v handler=%v notice=%v",
			middlewareCalled,
			handlerCalled,
			noticeDelivered,
		)
	}
	if source.open != 0 {
		t.Fatalf("Inspection created signal runtime: open=%d", source.open)
	}
}

func inspectionMustCompilePublic[C any](t testing.TB, definition cli.Definition[C]) cli.Application {
	t.Helper()
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return app
}

type inspectionCountingSignalSource struct {
	open int
}

func (source *inspectionCountingSignalSource) Source() cli.SignalSource[string] {
	return cli.SignalSource[string]{
		Signals: []cli.SignalDescriptor[string]{{
			Signal:   "counted",
			Fidelity: cli.SignalSourceDeliveryFidelityExact,
		}},
		Open: func(_ context.Context) (cli.SignalStream[string], error) {
			source.open++
			return cli.SignalStream[string]{
				Receive: func(ctx context.Context) (cli.SignalNotification[string], error) {
					<-ctx.Done()
					return cli.SignalNotification[string]{}, ctx.Err()
				},
			}, nil
		},
	}
}

type inspectionModeConfig struct{ Mode string }

type inspectionFirstSecondConfig struct{ First, Second string }

type inspectionDeclaredUndeclaredConfig struct {
	Declared   string
	Undeclared string
}

type inspectionVerboseConfig struct{ Verbose int }

type inspectionSecretConfig struct{ Secret string }

type inspectionTenantConfig struct{ Tenant string }

type inspectionLabelsConfig2 struct{ Labels map[string]string }

type inspectionRootConfig struct{}

type inspectionGateProviderConfig struct{ Gate string }

type inspectionRestConfig struct{ Rest []string }

type inspectionPassConfig struct{ Pass []string }

type inspectionValueConfig2 struct{ Value int }

type inspectionRootOnlyConfig struct{ RootOnly bool }

type inspectionVerboseConfig2 struct{ Verbose bool }

type inspectionVerboseVersionConfig struct {
	Verbose bool
	Version bool
}

type inspectionGateConfig struct{ Gate string }

type inspectionGateModeConfig struct {
	Gate string
	Mode string
}

type inspectionGateValueConfig struct {
	Gate  string
	Value string
}

type inspectionNameConfig struct{ Name string }

type inspectionConfig struct{}

type inspectionCountConfig struct{ Count int }

type inspectionTagsConfig struct{ Tags []string }

type inspectionLabelsConfig struct{ Labels map[string]int }

type inspectionValueConfig struct{ Value string }

func TestInspectionMaterializesDynamicDomainOnlyOnDemandAndReusesCanonicalDomainLanguage(t *testing.T) {
	t.Parallel()
	type config = inspectionModeConfig
	calls := 0
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(_ context.Context, _ *config, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
					calls++
					canonical := builder.Value(
						"prod",
						"production",
					).Description("Production environment").Hidden()

					builder.Alias("p", canonical)
					return nil
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Flags.ValueFor(mode).Long("mode")
		},
	})
	if calls != 0 {
		t.Fatalf("Compile() executed dynamic domain %d times", calls)
	}

	root := clitest.RootCommand()
	value := root.Flag("--mode").Value()
	domain := value.Domain(cli.ValueDomainUseDestination)
	canonical := domain.Entry("prod")
	alias := domain.Entry("p")
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		clitest.InspectionReady(), domain.Outcome(cli.ValueDomainOutcomeNotNeeded), domain.InvocationScoped(),
	)
	if calls != 0 {
		t.Fatalf("idle Inspect() executed dynamic domain %d times", calls)
	}

	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--mode", "p"},
	}).Require(t,
		clitest.InspectionReady(),
		domain.Outcome(cli.ValueDomainOutcomeReady), domain.InvocationScoped(), domain.EntriesExactly(canonical, alias),
		canonical.Hidden(), canonical.PresentedDescription(cli.Description{
			Short: "Production environment",
		}),
		alias.Canonical(canonical), alias.PresentedDescription(cli.Description{}), value.Resolved(), value.Text("prod"),
	)
	if calls != 1 {
		t.Fatalf("dynamic domain calls = %d, want 1", calls)
	}

	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--mode", "prod"},
	}).Require(t,
		clitest.InspectionReady(), domain.Outcome(cli.ValueDomainOutcomeReady), domain.InvocationScoped(),
	)
	if calls != 2 {
		t.Fatalf("dynamic domain calls after second operation = %d, want 2", calls)
	}
}

func TestInspectionArgumentFormSelectionPreservesAmbiguousMatches(t *testing.T) {
	t.Parallel()
	type config = inspectionFirstSecondConfig
	app := inspectionMustCompilePublic(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			first := root.Values.String(&root.Config.First)
			second := root.Values.String(&root.Config.Second)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) { form.ValueFor(first, "first") })
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) { form.ValueFor(second, "second") })
		},
	})
	request := cli.InspectionRequest{
		Args: []string{"value"},
	}
	root := clitest.RootCommand()
	f0, f1 := root.ArgumentFormAt(0), root.ArgumentFormAt(1)
	proof := clitest.Inspect(t.Context(), app, request)
	proof.Require(
		t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticAmbiguousArgumentForm).Present()),
		root.ArgumentFormSelectionOutcome(cli.ArgumentFormSelectionAmbiguous),
		root.MatchingArgumentFormsExactly(f0, f1),
		f0.Outcome(cli.ArgumentFormOutcomeMatched),
		f1.Outcome(cli.ArgumentFormOutcomeMatched),
	)
	inspection, _ := proof.Inspection()
	selection := inspection.Commands[inspection.Selection.Command.Selected].ArgumentFormSelection
	selection.Matches[0] = 0
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticAmbiguousArgumentForm).Present()),
		root.MatchingArgumentFormsExactly(f0, f1),
		f0.Outcome(cli.ArgumentFormOutcomeMatched), f1.Outcome(cli.ArgumentFormOutcomeMatched),
	)
}

func TestInspectionArgumentFormSelectionDistinguishesNoMatchFromUnrelatedGrammarFailure(t *testing.T) {
	t.Parallel()
	app := inspectionMustCompilePublic(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[struct{}]) { form.Literal("fixed") })
		},
	})
	root := clitest.RootCommand()
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"other"},
	}).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticUnexpectedArgument).Present()),
		root.ArgumentFormSelectionOutcome(cli.ArgumentFormSelectionNoMatch),
	)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--unknown"},
	}).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticUnknownFlag).Present()),
		root.ArgumentFormSelectionOutcome(cli.ArgumentFormSelectionNotEvaluated),
	)
}

func TestInspectionProjectsPolicyOutcomesAndNoticeSuppression(t *testing.T) {
	t.Parallel()
	type config = inspectionNameConfig
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			name := root.Values.String(&root.Config.Name)
			root.Values.Default(name, "ready")
			root.Rule.Require(name)
			root.Advisory.Always(cli.Advisory{
				Code:     "ready",
				Severity: cli.NoticeSeverityWarning,
				Message:  "ready",
			}).Subject(name)
			root.Notice.SuppressWhen(root.When.Resolved(name), cli.NoticeMatch{
				AdvisoryCode: "ready",
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	root := clitest.RootCommand()
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		clitest.InspectionReady(),
		root.RuleAt(0).Outcome(cli.EvaluationOutcomeSatisfied),
		root.Advisory("ready").Outcome(cli.AdvisoryOutcomeMatched),
		root.NoticeSuppressionAt(0).ConditionOutcome(cli.EvaluationOutcomeSatisfied),
		clitest.Notice(cli.NoticeAdvisory).Advisory("ready").Suppression(cli.NoticeSuppressionSuppressed),
		clitest.Notice(cli.NoticeAdvisory).Advisory("ready").SuppressedByExactly(root.NoticeSuppressionAt(0)),
	)
}

func TestInspectionUsesStableOperationLocaleAndResolvedPresentation(t *testing.T) {
	t.Parallel()
	type config = inspectionConfig
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Description: cli.Description{
			Short: "tool short",
		},
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"tool short": "outil court",
				},
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	clitest.Schema(app).Require(t, clitest.RootCommand().SourceDescription(cli.Description{
		Short: "tool short",
	}))
	clitest.Reference(app, cli.ReferenceOptions{
		Locale: "C",
	}).Require(t,
		clitest.EffectiveLocale("C"),
		clitest.RootCommand().PresentedDescription(cli.Description{
			Short: "tool short",
		}),
	)
	clitest.Reference(app, cli.ReferenceOptions{
		Locale: "fr",
	}).Require(t,
		clitest.EffectiveLocale("fr"),
		clitest.RootCommand().PresentedDescription(cli.Description{
			Short: "outil court",
		}),
	)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Locale: "fr",
	}).Require(t,
		clitest.InspectionReady(),
		clitest.EffectiveLocale("fr"),
		clitest.RootCommand().PresentedDescription(cli.Description{
			Short: "outil court",
		}),
	)
}

func TestContextualValueStateAuthorizationDoesNotCreateDependency(t *testing.T) {
	t.Parallel()
	type config = inspectionDeclaredUndeclaredConfig
	var ruleCalled, advisoryCalled bool
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			declared := root.Values.String(&root.Config.Declared)
			undeclared := root.Values.String(&root.Config.Undeclared)
			root.Values.Default(declared, "declared")
			root.Values.Default(undeclared, "undeclared")

			root.Rule.Check("context-authority", func(ctx context.Context, _ *config, dependencies cli.Dependencies) (bool, error) {
				ruleCalled = true
				if got, ok := dependencies.Get(declared); !ok || got != "declared" {
					t.Fatalf("declared Rule dependency = %q, %v", got, ok)
				}
				if _, ok := cli.ValueStateFromContext(ctx, declared); !ok {
					t.Fatal("declared Rule contextual value state unavailable")
				}
				if _, ok := cli.ValueStateFromContext(ctx, undeclared); ok {
					t.Fatal("undeclared Rule contextual value state became available")
				}
				return true, nil
			}).
				Subject(declared).
				DependsOn(declared)

			root.Advisory.Detect(func(ctx context.Context, _ *config, dependencies cli.Dependencies) (bool, error) {
				advisoryCalled = true
				if got, ok := dependencies.Get(declared); !ok || got != "declared" {
					t.Fatalf("declared Advisory dependency = %q, %v", got, ok)
				}
				if _, ok := cli.ValueStateFromContext(ctx, declared); !ok {
					t.Fatal("declared Advisory contextual value state unavailable")
				}
				if _, ok := cli.ValueStateFromContext(ctx, undeclared); ok {
					t.Fatal("undeclared Advisory contextual value state became available")
				}
				return true, nil
			}, cli.Advisory{
				Code:     "context-authority",
				Severity: cli.NoticeSeverityInformation,
				Message:  "context",
			}).Subject(declared).DependsOn(declared)
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	root := clitest.RootCommand()
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		clitest.InspectionReady(),
		root.Rule("context-authority").Outcome(cli.EvaluationOutcomeSatisfied),
		root.Advisory("context-authority").Outcome(cli.AdvisoryOutcomeMatched),
	)
	if !ruleCalled || !advisoryCalled {
		t.Fatalf("ruleCalled=%v advisoryCalled=%v", ruleCalled, advisoryCalled)
	}
}

func TestInspectionHelpAndVersionStopBeforeRunOnlyDemand(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		args   []string
		action cli.InvocationAction
	}{
		{name: "help", args: []string{"--help"}, action: cli.InvocationActionHelp},
		{name: "version", args: []string{"--version"}, action: cli.InvocationActionVersion},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			type config struct {
				Required string
				Mode     string
			}

			factCalls := 0
			domainCalls := 0
			ruleCalls := 0
			middlewareCalls := 0
			handlerCalls := 0

			app := clitest.Compile(t, cli.Definition[config]{
				Name:    "tool",
				Release: cli.Release.Ordered("v1", "v1"),
				Declare: func(root *cli.CommandBuilder[config]) {
					root.Facts.Require(&root.Config.Required)
					domain := root.Values.Domain(func(context.Context, *config, cli.Dependencies, *cli.ValueDomainBuilder[string]) error {
						domainCalls++
						return errors.New("run-only domain executed")
					})
					mode := root.Values.Value(&root.Config.Mode, domain)
					root.Rule.Check("run-only", func(context.Context, *config, cli.Dependencies) (bool, error) {
						ruleCalls++
						return false, errors.New("run-only policy executed")
					}).DependsOn(mode)
				},
				Middleware: []cli.Middleware[config]{func(ctx context.Context, _ *config, next cli.MiddlewareNext) error {
					middlewareCalls++
					return next(ctx)
				}},
				Run: func(context.Context, *config) error {
					handlerCalls++
					return nil
				},
			})

			facts := []cli.FactProvider{cli.FactsFor[config](func(context.Context, *cli.FactSupply[config]) error {
				factCalls++
				return errors.New("run-only facts executed")
			})}
			clitest.Inspect(t.Context(), app, cli.InspectionRequest{
				Args:  test.args,
				Facts: facts,
			}).Require(t,
				clitest.InspectionReady(),
				clitest.InvocationAction(test.action).Selected(),
			)

			if factCalls != 0 || domainCalls != 0 || ruleCalls != 0 || middlewareCalls != 0 || handlerCalls != 0 {
				t.Fatalf(
					"Help/Version Inspection demanded Run-only work: facts=%d domains=%d rules=%d middleware=%d handler=%d",
					factCalls,
					domainCalls,
					ruleCalls,
					middlewareCalls,
					handlerCalls,
				)
			}
		})
	}
}

func TestInspectionPreservesEstablishedActionSelectionAcrossLaterGrammarFailure(t *testing.T) {
	t.Parallel()
	type config = inspectionConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "commandment",
		Release: cli.Release.Ordered("v1.2.3", "v1.2.3"),
	})
	root := clitest.RootCommand()
	entry := root.ActionEntries(cli.InvocationActionVersion).Long("--version")
	token := clitest.TokenAt(0)
	origin := token.Origin()
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--version", "unexpected"},
	}).Require(t,
		clitest.InspectionRejected(), clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
		clitest.InvocationAction(cli.InvocationActionVersion).
			Selected(),
		clitest.InvocationAction(cli.InvocationActionVersion).Trigger(cli.InvocationActionTriggerEntry),
		clitest.InvocationAction(cli.InvocationActionVersion).
			SelectedEntry(entry),
		clitest.InvocationAction(cli.InvocationActionVersion).TriggeredBy(origin),
		entry.Kind(cli.ActionEntryFlag), token.Role(cli.TokenRoleAction), token.OwnedBy(entry), origin.Present(),
	)
}

func TestInspectionProjectsExplicitCommandSelectionPathAndCausalEntry(t *testing.T) {
	t.Parallel()
	type config = inspectionConfig
	app, err := cli.Compile(cli.Definition[config]{
		Name: "commandment",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.Command(cli.Command[config]{
				Name: "serve",
			})
		},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	serve := clitest.Command("serve")
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"serve"},
	}).Require(t,
		clitest.InspectionReady(),
		clitest.SelectionPathExactly(clitest.RootCommand(), serve),
		clitest.SelectionStepsExactly(serve.CanonicalEntry().SelectionStep().Selects(clitest.RootCommand(), serve)),
		clitest.TokenAt(0).Role(cli.TokenRoleCommand),
		clitest.TokenAt(0).OwnedBy(serve.CanonicalEntry()),
		clitest.TokenAt(0).Origin().Present(),
	)
}

func TestInspectionDoesNotRejectResponseCapableApplicationWithoutResponseLexicalEntry(t *testing.T) {
	t.Parallel()
	type config = inspectionConfig
	app, err := cli.Compile(cli.Definition[config]{
		Name: "commandment",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
		},
		Run: func(context.Context, *config) error { return nil },
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t, clitest.InspectionReady())
}

func TestInspectionProjectsSuccessfulResponseIncludeWithoutFlatteningItsLexicalFact(t *testing.T) {
	t.Parallel()
	type config = inspectionConfig
	loads := 0
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "commandment",
		Declare: func(root *cli.CommandBuilder[config]) { root.Dialect.ResponseFiles(true) },
		Run:     func(context.Context, *config) error { return nil },
	})
	token := clitest.TokenAt(0)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"@empty"},
		ResponseFiles: cli.ResponseFileAccess{
			Load: func(_ context.Context, request cli.ResponseFileRequest) (cli.ResponseFile, error) {
				loads++
				return cli.ResponseFile{
					Key:      request.Path,
					Revision: cli.Revision("r1"),
					Location: "virtual:" + request.Path,
					Content:  nil,
				}, nil
			},
		},
	}).Require(t, clitest.InspectionReady(), token.Role(cli.TokenRoleResponseFile), token.Text("@empty"), token.Origin().Present(), token.Origin().Source().Present())
	if loads != 1 {
		t.Fatalf("response loads = %d, want 1", loads)
	}
}

func TestInspectionProjectsResolvedExplicitScalarCandidate(t *testing.T) {
	t.Parallel()
	type config = inspectionNameConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Help:    cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) { root.Flags.String(&root.Config.Name).Long("name") },
	})
	root := clitest.RootCommand()
	flag := root.Flag("--name")
	value := flag.Value()
	candidate := value.CandidateAt(0)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--name", "alice"},
	}).Require(t,
		clitest.InspectionReady(),
		value.Resolved(), value.Text("alice"), value.CandidatesExactly(candidate), value.ContributorsExactly(candidate),
		candidate.Disposition(cli.ValueCandidateDispositionEffective),
		candidate.Contribution(cli.ValueCandidateContributionWhole),
		candidate.Declaration(flag.Spelling("--name")), candidate.InputText("alice"), candidate.Text("alice"),
	)
}

func TestInspectionResolvesEnvironmentCandidate(t *testing.T) {
	t.Parallel()
	type config = inspectionNameConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			name := root.Values.String(&root.Config.Name)
			root.Values.Environment(name, "TOOL_NAME")
		},
	})
	root := clitest.RootCommand()
	env := root.Environment("TOOL_NAME")
	value := env.Value()
	candidate := value.CandidateAt(0)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Env: cli.Env{{
			Name:  "TOOL_NAME",
			Value: "from-env",
		}},
	}).Require(t,
		clitest.InspectionReady(),
		value.Resolved(),
		value.Text("from-env"), value.CandidatesExactly(candidate), value.ContributorsExactly(candidate),
		candidate.Disposition(cli.ValueCandidateDispositionEffective), candidate.Declaration(env),
	)
}

func TestInspectionSkipsSuppressedComputedDefaultWithoutCallingIt(t *testing.T) {
	t.Parallel()
	type config = inspectionNameConfig
	calls := 0
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			name := root.Values.String(&root.Config.Name)
			root.Flags.ValueFor(name).Long("name")
			root.Values.ComputedDefault(
				name,
				cli.Value.ComputedBy(
					func(context.Context, *config, cli.Dependencies) (string, error) { calls++; return "computed", nil },
				),
			)
		},
	})
	value := clitest.RootCommand().Flag("--name").Value()
	explicit, computed := value.ProducerAt(0), value.ProducerAt(1)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--name", "explicit"},
	}).Require(t,
		clitest.InspectionReady(), value.Resolved(), value.Text("explicit"), value.ProducersExactly(explicit, computed),
		explicit.Kind(cli.ValueProducerKindFlag), explicit.Outcome(cli.ValueProducerOutcomeContributed),
		computed.Kind(cli.ValueProducerKindComputedDefault), computed.Outcome(cli.ValueProducerOutcomeSkipped),
	)
	if calls != 0 {
		t.Fatalf("computed default calls = %d, want 0", calls)
	}
}

func TestInspectionOptionalComputedDefaultOmissionFallsThroughStaticDefault(t *testing.T) {
	t.Parallel()
	type config = inspectionNameConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			name := root.Values.String(&root.Config.Name)
			root.Values.OptionalComputedDefault(
				name,
				cli.Value.OptionallyComputedBy(
					func(context.Context, *config, cli.Dependencies) (cli.ValueContribution[string], error) {
						return cli.OmitValue[string](), nil
					},
				),
			)
			root.Values.Default(name, "static")
		},
	})
	value := clitest.RootCommand().ValueAt(0)
	p0, p1 := value.ProducerAt(0), value.ProducerAt(1)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		clitest.InspectionReady(), value.Resolved(), value.Text("static"), value.ProducersExactly(p0, p1),
		p0.Kind(cli.ValueProducerKindComputedDefault),
		p0.Outcome(cli.ValueProducerOutcomeOmitted),
		p1.Kind(cli.ValueProducerKindStaticDefault), p1.Outcome(cli.ValueProducerOutcomeContributed),
	)
}

func TestInspectionShortCountClusterUsesIndependentTokensAndOneCandidate(t *testing.T) {
	t.Parallel()
	type config = inspectionVerboseConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ShortFlagClustering(cli.ShortFlagClusteringEnabled)
			root.Flags.Count(&root.Config.Verbose).Occurrences(cli.Count.ZeroOrMore()).Short('v')
		},
	})
	root := clitest.RootCommand()
	flag := root.Flag("-v")
	value := flag.Value()
	candidate := value.CandidateAt(0)
	expectations := make([]clitest.InspectionExpectation, 0, 17)
	expectations = append(expectations,
		clitest.InspectionReady(),
		value.Resolved(),
		value.Text("3"),
		value.CandidatesExactly(candidate),
		value.ContributorsExactly(candidate),
		candidate.Disposition(cli.ValueCandidateDispositionEffective),
		candidate.Text("3"),
		candidate.Declaration(flag),
	)
	for i := range 3 {
		token := clitest.TokenAt(i)
		expectations = append(expectations, token.Role(cli.TokenRoleFlag), token.OccurrenceIndex(i), token.Text("-v"))
	}
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"-vvv"},
	}).Require(t, expectations...)
}

func TestInspectionRedactsSensitiveValueAndCandidateInput(t *testing.T) {
	t.Parallel()
	type config = inspectionSecretConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			secret := root.Values.String(&root.Config.Secret).Sensitive()
			root.Flags.ValueFor(secret).Long("secret").Sensitive()
		},
	})
	value := clitest.RootCommand().Flag("--secret").Value()
	candidate := value.CandidateAt(0)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--secret", "classified"},
	}).Require(t,
		clitest.InspectionReady(), value.Sensitive(), value.Redacted(), value.CandidatesExactly(candidate),
		candidate.Sensitive(), candidate.InputRedacted(), candidate.InputText(""), candidate.Redacted(),
	)
}

func TestInspectionReturnsPartialValueGraphWithInvalidInputDiagnostic(t *testing.T) {
	t.Parallel()
	type config = inspectionCountConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Help:    cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) { root.Flags.Int(&root.Config.Count).Long("count") },
	})
	value := clitest.RootCommand().Flag("--count").Value()
	candidate := value.CandidateAt(0)
	diagnostic := clitest.Diagnostic(cli.DiagnosticInvalidValue)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--count", "not-an-int"},
	}).Require(t,
		clitest.InspectionRejected(), clitest.FailsWith(diagnostic.Present()), clitest.DiagnosticsExactly(diagnostic),
		value.Failed(),
		value.CandidatesExactly(candidate), candidate.Disposition(cli.ValueCandidateDispositionRejected),
		value.DiagnosticsExactly(diagnostic), candidate.DiagnosticsExactly(diagnostic),
	)
}

func TestInspectionResolvesInvocationDefaultThroughOrdinaryCandidateModel(t *testing.T) {
	t.Parallel()
	type config = inspectionNameConfig
	var name cli.Destination[config, string]
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Help:    cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) { name = root.Values.String(&root.Config.Name) },
	})
	value := clitest.RootCommand().ValueAt(0)
	candidate := value.CandidateAt(0)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Defaults: []cli.DefaultProvider{
			cli.DefaultsFor[config](func(_ context.Context, supply *cli.DefaultSupply[config]) error {
				supply.Set(name, "from-invocation")
				return nil
			}),
		},
	}).Require(t,
		clitest.InspectionReady(),
		value.Resolved(),
		value.Text("from-invocation"), value.CandidatesExactly(candidate), value.ContributorsExactly(candidate),
		candidate.Disposition(cli.ValueCandidateDispositionEffective), candidate.Text("from-invocation"),
	)
}

func TestInspectionSuppliedFactSuppressesResolvedFact(t *testing.T) {
	t.Parallel()
	type config = inspectionTenantConfig
	var tenant cli.Destination[config, string]
	resolverCalls := 0
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			tenant = root.Facts.Resolve(
				&root.Config.Tenant,
				func(context.Context, *config, cli.Dependencies) (string, error) {
					resolverCalls++
					return "resolved", nil
				},
			).Destination()
		},
	})
	value := clitest.RootCommand().FactAt(0).Value()
	effective, suppressed := value.ContributorAt(0), value.CandidateAt(1)
	supplied, resolved := value.ProducerAt(0), value.ProducerAt(1)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Facts: []cli.FactProvider{
			cli.FactsFor[config](func(_ context.Context, supply *cli.FactSupply[config]) error {
				supply.Set(tenant, "supplied")
				return nil
			}),
		},
	}).Require(t,
		clitest.InspectionReady(),
		value.Resolved(),
		value.NoRepresentation(),
		value.ProducersExactly(supplied, resolved),
		value.CandidatesExactly(effective, suppressed), value.ContributorsExactly(effective),
		effective.Disposition(cli.ValueCandidateDispositionEffective),
		effective.ProducedBy(supplied), supplied.Kind(cli.ValueProducerKindSuppliedFact),
		suppressed.Disposition(cli.ValueCandidateDispositionSuppressed),
		suppressed.ProducedBy(resolved), resolved.Kind(cli.ValueProducerKindResolvedFact),
	)
	if resolverCalls != 0 {
		t.Fatalf("resolved fact called despite supplied fact: %d", resolverCalls)
	}
}

func TestInspectionEnvironmentEmptySequenceAdaptersPreserveWholeVersusSegment(t *testing.T) {
	t.Parallel()
	t.Run("elements empty is no effect", func(t *testing.T) {
		t.Parallel()
		type config = inspectionTagsConfig
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Help: cli.HelpConfig{Disabled: true},
			Declare: func(root *cli.CommandBuilder[config]) {
				tags := root.Values.Sequence(&root.Config.Tags, cli.Value.String())
				root.Values.Environment(tags, "TOOL_TAGS").Elements(",")
				root.Values.Default(tags, []string{"fallback"})
			},
		})
		envDecl := clitest.RootCommand().Environment("TOOL_TAGS")
		value := envDecl.Value()
		env, fallback := value.CandidateAt(0), value.CandidateAt(1)
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Env: cli.Env{{
				Name:  "TOOL_TAGS",
				Value: "",
			}},
		}).Require(t,
			clitest.InspectionReady(),
			value.Resolved(),
			value.CandidatesExactly(env, fallback),
			value.ContributorsExactly(fallback), value.ElementsExactly(fallback.ElementAt(0)),
			env.Declaration(envDecl),
			env.Contribution(cli.ValueCandidateContributionSequenceSegment),
			env.Disposition(cli.ValueCandidateDispositionNoEffect), env.ElementCount(0),
			fallback.Disposition(cli.ValueCandidateDispositionEffective),
			fallback.ElementCount(1), fallback.ElementAt(0).Text("fallback"),
		)
	})

	t.Run("whole sequence empty is effective", func(t *testing.T) {
		t.Parallel()
		type config = inspectionTagsConfig
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Help: cli.HelpConfig{Disabled: true},
			Declare: func(root *cli.CommandBuilder[config]) {
				tags := root.Values.Sequence(&root.Config.Tags, cli.Value.String())
				root.Values.Environment(tags, "TOOL_TAGS").WholeSequence(",")
				root.Values.Default(tags, []string{"fallback"})
			},
		})
		envDecl := clitest.RootCommand().Environment("TOOL_TAGS")
		value := envDecl.Value()
		env, fallback := value.CandidateAt(0), value.CandidateAt(1)
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Env: cli.Env{{
				Name:  "TOOL_TAGS",
				Value: "",
			}},
		}).Require(t,
			clitest.InspectionReady(),
			value.Resolved(),
			value.CandidatesExactly(env, fallback), value.ContributorsExactly(env), value.ElementsExactly(),
			env.Declaration(envDecl),
			env.Contribution(cli.ValueCandidateContributionWhole),
			env.Disposition(cli.ValueCandidateDispositionEffective), env.ElementCount(0),
			fallback.Disposition(cli.ValueCandidateDispositionSuppressed),
		)
	})
}

func TestInspectionEnvironmentKeyedEntriesPreserveAtomicEntryGranularity(t *testing.T) {
	t.Parallel()
	type config = inspectionLabelsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			labels := root.Values.Keyed(&root.Config.Labels, cli.Value.Keyed(cli.Value.String(), cli.Value.Int()))
			root.Values.Environment(labels, "TOOL_LABELS").KeyedEntries(",")
		},
	})
	env := clitest.RootCommand().Environment("TOOL_LABELS")
	value := env.Value()
	c0, c1 := value.CandidateAt(0), value.CandidateAt(1)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Env: cli.Env{{
			Name:  "TOOL_LABELS",
			Value: "a=1,b=2",
		}},
	}).Require(t,
		clitest.InspectionReady(),
		value.Resolved(),
		value.CandidatesExactly(c0, c1),
		value.ContributorsExactly(c0, c1), value.EntriesExactly(c0.EntryAt(0), c1.EntryAt(0)),
		c0.Declaration(env),
		c0.Contribution(cli.ValueCandidateContributionKeyedEntry),
		c0.Disposition(cli.ValueCandidateDispositionEffective),
		c0.EntryCount(1), c0.EntryAt(0).Key().Text("a"), c0.EntryAt(0).Value().Text("1"),
		c1.Declaration(env),
		c1.Contribution(cli.ValueCandidateContributionKeyedEntry),
		c1.Disposition(cli.ValueCandidateDispositionEffective),
		c1.EntryCount(1), c1.EntryAt(0).Key().Text("b"), c1.EntryAt(0).Value().Text("2"),
	)
}

func TestInspectionEnvironmentPrefixUsesSuffixAsKeyAndRawValue(t *testing.T) {
	t.Parallel()
	type config = inspectionLabelsConfig2
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			labels := root.Values.Keyed(&root.Config.Labels, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
			root.Values.EnvironmentPrefix(labels, "APP_LABEL_")
		},
	})
	env := clitest.RootCommand().Environment("APP_LABEL_")
	value := env.Value()
	c0, c1 := value.CandidateAt(0), value.CandidateAt(1)
	e0, e1 := c0.EntryAt(0), c1.EntryAt(0)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Env: cli.Env{{
			Name:  "APP_LABEL_name",
			Value: "alice",
		}, {
			Name:  "APP_LABEL_",
			Value: "",
		}},
	}).Require(t,
		clitest.InspectionReady(),
		value.Resolved(),
		value.CandidatesExactly(c0, c1), value.ContributorsExactly(c0, c1), value.EntriesExactly(e0, e1),
		c0.Declaration(env),
		c0.Contribution(cli.ValueCandidateContributionKeyedEntry),
		c0.Disposition(cli.ValueCandidateDispositionEffective), c0.EntryCount(1),
		e0.Key().InputText("name"), e0.Key().Text("name"), e0.Value().InputText("alice"), e0.Value().Text("alice"),
		c1.Declaration(env),
		c1.Contribution(cli.ValueCandidateContributionKeyedEntry),
		c1.Disposition(cli.ValueCandidateDispositionEffective), c1.EntryCount(1),
		e1.Key().InputText(""), e1.Key().Text(""), e1.Value().InputText(""), e1.Value().Text(""),
	)
}

func TestInspectionEnvironmentWholeKeyedEmptySuppressesLowerEntries(t *testing.T) {
	t.Parallel()
	type config = inspectionLabelsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			labels := root.Values.Keyed(&root.Config.Labels, cli.Value.Keyed(cli.Value.String(), cli.Value.Int()))
			root.Values.Environment(labels, "TOOL_LABELS").WholeKeyed(",")
			root.Values.Default(labels, map[string]int{"fallback": 1})
		},
	})
	envDecl := clitest.RootCommand().Environment("TOOL_LABELS")
	value := envDecl.Value()
	env, fallback := value.CandidateAt(0), value.CandidateAt(1)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Env: cli.Env{{
			Name:  "TOOL_LABELS",
			Value: "",
		}},
	}).Require(t,
		clitest.InspectionReady(),
		value.Resolved(),
		value.CandidatesExactly(env, fallback), value.ContributorsExactly(env), value.EntriesExactly(),
		env.Declaration(envDecl),
		env.Contribution(cli.ValueCandidateContributionKeyedReplacement),
		env.Disposition(cli.ValueCandidateDispositionEffective), env.EntryCount(0),
		fallback.Disposition(cli.ValueCandidateDispositionSuppressed),
	)
}

func TestProviderAdvisoryNoticeUsesProviderLocalizationAcrossInspectionAndRun(t *testing.T) {
	t.Parallel()
	type rootConfig = inspectionRootConfig
	type providerConfig = inspectionGateProviderConfig
	const (
		message = "provider message"
		reason  = "provider reason"
		hint    = "provider hint"
	)
	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					message: "application message must not win",
					reason:  "application reason must not win",
					hint:    "application hint must not win",
				},
			}},
		},
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[rootConfig]) error {
				batch.Localization(cli.Localization{
					DefaultLocale: "en",
					Bundles: []cli.TranslationBundle{{
						Locale: "fr",
						Translations: cli.Translations{
							message: "message fournisseur",
							reason:  "raison fournisseur",
							hint:    "indice fournisseur",
						},
					}},
				})
				batch.Command(cli.Command[providerConfig]{
					Name: "remote",
					Declare: func(remote *cli.CommandBuilder[providerConfig]) {
						gate := remote.Values.String(&remote.Config.Gate)
						remote.Values.Default(gate, "ready")
						remote.Advisory.Always(cli.Advisory{
							Code:     "localized",
							Severity: cli.NoticeSeverityWarning,
							Message:  message,
							Reason:   reason,
							Hint:     hint,
						})
						remote.Advisory.Always(cli.Advisory{
							Code:     "suppressed",
							Severity: cli.NoticeSeverityInformation,
							Message:  "suppressed message",
						})
						remote.Notice.SuppressWhen(remote.When.Resolved(gate), cli.NoticeMatch{
							AdvisoryCode: "suppressed",
						})
						remote.Notice.SuppressWhen(remote.When.Equal(gate, "ready"), cli.NoticeMatch{
							AdvisoryCode: "suppressed",
						})
					},
					Run: func(context.Context, *providerConfig) error { return nil },
				})
				return nil
			})
		},
	})
	root := clitest.RootCommand()
	remote := root.Command("remote")
	localized := clitest.Notice(cli.NoticeAdvisory).Advisory("localized").On(remote)
	suppressed := clitest.Notice(cli.NoticeAdvisory).Advisory("suppressed").On(remote)

	assertInspection := func(_ string, locale cli.Locale, wantMessage, wantReason, wantHint string) cli.Inspection {
		t.Helper()
		proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args:   []string{"remote"},
			Locale: locale,
		})
		proof.Require(
			t,
			clitest.InspectionReady(),
			clitest.SelectionPathExactly(root, remote),
			clitest.NoticesExactly(localized, suppressed),
			localized.Severity(
				cli.NoticeSeverityWarning,
			),
			localized.Message(wantMessage),
			localized.Reason(wantReason),
			localized.Hint(wantHint),
			localized.Suppression(cli.NoticeSuppressionNotSuppressed),
			localized.SuppressedByExactly(),
			suppressed.Severity(
				cli.NoticeSeverityInformation,
			),
			suppressed.Suppression(cli.NoticeSuppressionSuppressed),
			suppressed.SuppressedByExactly(remote.NoticeSuppressionAt(0), remote.NoticeSuppressionAt(1)),
		)
		inspection, _ := proof.Inspection()
		return inspection
	}

	first := assertInspection("target locale", "fr", "message fournisseur", "raison fournisseur", "indice fournisseur")
	first.Notices[0].Message = "mutated result"
	first.Notices[0].CommandPath[0] = "mutated path"
	first.Notices[1].SuppressedBy[0].DeclarationIndex = 99
	_ = assertInspection(
		"detached target locale",
		"fr",
		"message fournisseur",
		"raison fournisseur",
		"indice fournisseur",
	)
	_ = assertInspection("source presentation", "C", message, reason, hint)

	assertRun := func(label string, locale cli.Locale, wantMessage, wantReason, wantHint string) cli.Notice {
		t.Helper()
		var delivered []cli.Notice
		err := app.Run(t.Context(), cli.RunRequest{
			Args:   []string{"remote"},
			Locale: locale,
			Notify: func(_ context.Context, notice cli.Notice) error { delivered = append(delivered, notice); return nil },
		})
		if err != nil {
			t.Fatalf("%s Run() error = %v", label, err)
		}
		if len(delivered) != 1 {
			t.Fatalf("%s delivered Notices = %#v", label, delivered)
		}
		notice := delivered[0]
		if notice.Code != cli.NoticeAdvisory || notice.AdvisoryCode != "localized" ||
			notice.Severity != cli.NoticeSeverityWarning ||
			notice.Message != wantMessage ||
			notice.Reason != wantReason ||
			notice.Hint != wantHint {
			t.Fatalf("%s delivered Notice = %#v", label, notice)
		}
		return notice
	}
	delivered := assertRun("target locale", "fr", "message fournisseur", "raison fournisseur", "indice fournisseur")
	delivered.Message = "mutated delivery"
	delivered.CommandPath[0] = "mutated delivery path"
	_ = assertRun("source presentation", "C", message, reason, hint)
}

func TestInspectionPlansExplicitSignalRequirementsWithoutSubscribing(t *testing.T) {
	t.Parallel()
	type config = inspectionConfig
	opens := 0
	source := cli.SignalSource[string]{
		Signals: []cli.SignalDescriptor[string]{
			{Signal: inspectionSignalNativeInterrupt, Fidelity: cli.SignalSourceDeliveryFidelityExact},
		},
		Open: func(context.Context) (cli.SignalStream[string], error) {
			opens++
			panic("Inspect opened signal source")
		},
	}
	var interrupt cli.Signal
	app, err := cli.Compile(cli.Definition[config]{
		Name: "commandment",
		Run:  func(context.Context, *config) error { return nil },
		Declare: func(root *cli.CommandBuilder[config]) {
			interrupt = root.Signals.Source(source).Signal(inspectionSignalNativeInterrupt)
			root.Signals.Policy(
				root.Signals.Match.On(interrupt).During(cli.SignalPhaseExecution).Each(),
				root.Signals.Action.Abort(17),
			)
		},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		SignalHostCapabilities: cli.SignalHostCapabilityAbort,
	}).Require(t,
		clitest.InspectionReady(),
		clitest.SignalIntegration().Status(cli.SignalIntegrationStatusReady),
		clitest.SignalIntegration().Required(),
		clitest.SignalIntegration().Supplied(),
		clitest.SignalIntegration().ResolvedSignalsExactly(interrupt),
		clitest.ExecutionAdmission().SignalPolicyAt(0).Status(cli.SignalPolicyStatusReady),
	)
	if opens != 0 {
		t.Fatalf("Inspect opened signal source %d times", opens)
	}
}

func TestInspectionReportsMissingSignalHostCapability(t *testing.T) {
	t.Parallel()
	type config = inspectionConfig
	source := cli.SignalSource[string]{
		Signals: []cli.SignalDescriptor[string]{
			{Signal: inspectionSignalNativeInterrupt, Fidelity: cli.SignalSourceDeliveryFidelityExact},
		},
		Open: func(context.Context) (cli.SignalStream[string], error) { panic("Inspect opened signal source") },
	}
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "commandment",
		Run:  func(context.Context, *config) error { return nil },
		Declare: func(root *cli.CommandBuilder[config]) {
			interrupt := root.Signals.Source(source).Signal(inspectionSignalNativeInterrupt)
			root.Signals.Policy(
				root.Signals.Match.On(interrupt).During(cli.SignalPhaseExecution).Each(),
				root.Signals.Action.Abort(17),
			)
		},
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticIncompatibleSignalSource).Present()),
		clitest.SignalIntegration().
			Status(cli.SignalIntegrationStatusIncompatible),
		clitest.SignalIntegration().MissingHostCapabilities(cli.SignalHostCapabilityAbort),
	)
}

func TestInspectionClearInheritedRemovesOnlyStrictAncestorPolicies(t *testing.T) {
	t.Parallel()
	type config = inspectionConfig
	source := cli.SignalSource[string]{
		Signals: []cli.SignalDescriptor[string]{
			{Signal: inspectionSignalNativeInterrupt, Fidelity: cli.SignalSourceDeliveryFidelityExact},
		},
		Open: func(context.Context) (cli.SignalStream[string], error) { panic("Inspect opened signal source") },
	}
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "commandment",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			interrupt := root.Signals.Source(source).Signal(inspectionSignalNativeInterrupt)
			root.Signals.Policy(
				root.Signals.Match.On(interrupt).During(cli.SignalPhaseExecution).Each(),
				root.Signals.Action.Ignore(),
			)
			root.Subcommands.Command(cli.Command[config]{
				Name: "child",
				Declare: func(child *cli.CommandBuilder[config]) {
					child.Signals.Policy(
						child.Signals.Match.On(interrupt).During(cli.SignalPhaseExecution).Each(),
						child.Signals.Action.Ignore(),
					)
					child.Signals.ClearInherited()
					child.Signals.Policy(
						child.Signals.Match.On(interrupt).During(cli.SignalPhaseExecution).Each(),
						child.Signals.Action.Ignore(),
					)
				},
			})
		},
	})
	root := clitest.RootCommand()
	child := root.Command("child")
	p0, p1 := child.SignalPolicyAt(0), child.SignalPolicyAt(1)
	clitest.Schema(app).
		Require(t, root.SignalPoliciesExactly(root.SignalPolicyAt(0)), child.SignalPoliciesExactly(p0, p1))
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{Args: []string{"child"}}).Require(t,
		clitest.InspectionReady(),
		clitest.SelectionPathExactly(root, child),
		clitest.ExecutionAdmission().
			SignalPoliciesExactly(p0, p1),
		p0.Status(cli.SignalPolicyStatusReady), p1.Status(cli.SignalPolicyStatusReady),
	)
}

func TestInspectionSignalPolicyFollowsCanonicalCommandThroughAliasAndDefault(t *testing.T) {
	t.Parallel()
	type config = inspectionConfig
	newSource := func() cli.SignalSource[string] {
		return cli.SignalSource[string]{
			Signals: []cli.SignalDescriptor[string]{
				{Signal: inspectionSignalNativeInterrupt, Fidelity: cli.SignalSourceDeliveryFidelityExact},
			},
			Open: func(context.Context) (cli.SignalStream[string], error) { panic("Inspect opened signal source") },
		}
	}
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "commandment",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			interrupt := root.Signals.Source(newSource()).Signal(inspectionSignalNativeInterrupt)
			root.Subcommands.Command(cli.Command[config]{
				Name:    "child",
				Aliases: cli.Aliases("alias"),
				Declare: func(child *cli.CommandBuilder[config]) {
					child.Signals.Policy(
						child.Signals.Match.On(interrupt).During(cli.SignalPhaseExecution).Each(),
						child.Signals.Action.Ignore(),
					)
				},
			}).Default()
		},
	})
	root := clitest.RootCommand()
	child := root.Command("child")
	policy := child.SignalPolicyAt(0)
	for _, args := range [][]string{{"alias"}, {}} {
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{Args: args}).Require(t,
			clitest.InspectionReady(),
			clitest.SelectionPathExactly(root, child),
			clitest.ExecutionAdmission().SignalPoliciesExactly(policy), policy.Status(cli.SignalPolicyStatusReady),
		)
	}

	t.Run("provider policy is operation-local and rejected output is inert", func(t *testing.T) {
		t.Parallel()
		providerApp := clitest.Compile(t, cli.Definition[config]{
			Name: "commandment",
			Help: cli.HelpConfig{Disabled: true},
			Declare: func(rootBuilder *cli.CommandBuilder[config]) {
				interrupt := rootBuilder.Signals.Source(newSource()).Signal(inspectionSignalNativeInterrupt)
				rootBuilder.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
					batch.Command(cli.Command[config]{
						Name: "remote",
						Declare: func(remote *cli.CommandBuilder[config]) {
							remote.Signals.Policy(
								remote.Signals.Match.On(interrupt).During(cli.SignalPhaseExecution).Each(),
								remote.Signals.Action.Ignore(),
							)
						},
					})
					return nil
				})
			},
		})
		remote := root.Command("remote")
		remotePolicy := remote.SignalPolicyAt(0)
		for range 2 {
			clitest.Inspect(t.Context(), providerApp, cli.InspectionRequest{Args: []string{"remote"}}).
				Require(t,
					clitest.InspectionReady(),
					clitest.SelectionPathExactly(root, remote),
					clitest.ExecutionAdmission().SignalPoliciesExactly(remotePolicy),
					remotePolicy.Status(cli.SignalPolicyStatusReady),
				)
		}

		wantErr := errors.New("reject provider output")
		rejectedApp := clitest.Compile(t, cli.Definition[config]{
			Name: "commandment",
			Help: cli.HelpConfig{Disabled: true},
			Declare: func(rootBuilder *cli.CommandBuilder[config]) {
				interrupt := rootBuilder.Signals.Source(newSource()).Signal(inspectionSignalNativeInterrupt)
				rootBuilder.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
					batch.Command(cli.Command[config]{
						Name: "rejected",
						Declare: func(rejected *cli.CommandBuilder[config]) {
							rejected.Signals.Policy(
								rejected.Signals.Match.On(interrupt).During(cli.SignalPhaseExecution).Each(),
								rejected.Signals.Action.Ignore(),
							)
						},
					})
					return wantErr
				})
			},
		})
		proof := clitest.Inspect(t.Context(), rejectedApp, cli.InspectionRequest{Args: []string{"rejected"}})
		proof.Require(
			t,
			clitest.InspectionRejected(),
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticCommandProviderFailure).Present()),
			clitest.ExecutionAdmission().SignalPoliciesExactly(),
			root.Command("rejected").ObservedDoesNotExist(),
		)
		_, err := proof.Inspection()
		if err == nil || !errors.Is(err, wantErr) {
			t.Fatalf("rejected provider error = %v", err)
		}
	})
}

func TestInspectApplicationRejectsInvalidLocale(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
		},
	})
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Locale: "not_a_locale",
	})
	proof.Require(
		t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidValue).Present()),
	)
}

func TestCloneRuleOperandsPreservesInspectionSemantics(t *testing.T) {
	t.Parallel()
	type config = inspectionCountConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			count := root.Values.Int(&root.Config.Count).Sensitive()
			root.Values.Default(count, 2)
			root.Rule.Between(count, 1, 3)
		},
	})
	rule := clitest.RootCommand().RuleAt(0)
	o0, o1 := rule.OperandAt(0), rule.OperandAt(1)
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{})
	proof.Require(
		t,
		clitest.InspectionReady(),
		rule.OperandCount(2),
		o0.Sensitive(),
		o0.Literal().Sensitive(),
		o0.Literal().Redacted(),
		o1.Sensitive(),
		o1.Literal().Sensitive(),
		o1.Literal().Redacted(),
	)
	inspection, _ := proof.Inspection()
	root := inspection.Commands[inspection.Selection.Command.Root]
	root.Rules[0].Operands[0].Sensitive = false
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).
		Require(t, rule.OperandCount(2), o0.Sensitive(), o0.Literal().Sensitive(), o0.Literal().Redacted(), o1.Sensitive(), o1.Literal().Sensitive(), o1.Literal().Redacted())
}

func TestInspectionProjectsRemainderAndPassThroughTokenRoles(t *testing.T) {
	t.Parallel()
	t.Run("remainder", func(t *testing.T) {
		t.Parallel()
		type config = inspectionRestConfig
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				rest := root.Values.Sequence(&root.Config.Rest, cli.Value.String())
				root.Arguments.Remainder(rest, "rest", cli.Count.ZeroOrMore())
			},
		})
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"one", "two"},
		}).Require(t,
			clitest.InspectionReady(),
			clitest.TokenAt(0).Role(cli.TokenRoleRemainder),
			clitest.TokenAt(1).Role(cli.TokenRoleRemainder),
		)
	})
	t.Run("pass through", func(t *testing.T) {
		t.Parallel()
		type config = inspectionPassConfig
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				pass := root.Values.Sequence(&root.Config.Pass, cli.Value.String())
				root.Arguments.PassThrough(pass, "pass", cli.Count.ZeroOrMore())
			},
		})
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"one", "two"},
		}).Require(t,
			clitest.InspectionReady(),
			clitest.TokenAt(0).Role(cli.TokenRolePassThrough),
			clitest.TokenAt(1).Role(cli.TokenRolePassThrough),
		)
	})
}

func TestInspectionProjectsUniquePrefixNameMatch(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Dialect.CommandMatching(cli.NameMatchingUniquePrefix)
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "serve",
			})
		},
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"ser"},
	}).Require(t,
		clitest.InspectionReady(),
		clitest.TokenAt(0).Role(cli.TokenRoleCommand),
		clitest.TokenAt(0).Match(cli.NameMatchKindUniquePrefix),
	)
}

func TestInspectionClonesNestedConditionExpressions(t *testing.T) {
	t.Parallel()
	type config = inspectionValueConfig2
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.Int(&root.Config.Value)
			root.Values.Default(value, 1)
			root.VisibleWhen(root.When.All(root.When.Equal(value, 1), root.When.Not(root.When.Equal(value, 2))))
		},
	})
	expr := clitest.RootCommand().ConditionAt(0).Expression()
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{})
	proof.Require(
		t,
		clitest.InspectionReady(),
		expr.ChildCount(2),
		expr.ChildAt(0).OperandCount(1),
		expr.ChildAt(1).ChildCount(1),
	)
	inspection, _ := proof.Inspection()
	root := inspection.Commands[inspection.Selection.Command.Root]
	root.Conditions[0].Expression.Children[0].Operands = nil
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).
		Require(t, expr.ChildCount(2), expr.ChildAt(0).OperandCount(1), expr.ChildAt(1).ChildCount(1))
}

func TestGrammarDiagnosticCode(t *testing.T) {
	t.Parallel()

	t.Run("not introduced command", func(t *testing.T) {
		t.Parallel()
		app := clitest.Compile(t, cli.Definition[struct{}]{
			Name:    "tool",
			Release: cli.Release.SemVer("2.0.0"),
			Declare: func(root *cli.CommandBuilder[struct{}]) {
				root.Subcommands.Command(cli.Command[struct{}]{
					Name: "future",
					Release: cli.ReleaseLifecycle{
						IntroducedIn: "3.0.0",
					},
				})
			},
		})
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"future"},
		}).Require(t,
			clitest.InspectionRejected(),
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticNotIntroducedCommand).Present()),
		)
	})

	t.Run("removed command", func(t *testing.T) {
		t.Parallel()
		app := clitest.Compile(t, cli.Definition[struct{}]{
			Name:    "tool",
			Release: cli.Release.SemVer("2.0.0"),
			Declare: func(root *cli.CommandBuilder[struct{}]) {
				root.Subcommands.Command(cli.Command[struct{}]{
					Name: "old",
					Release: cli.ReleaseLifecycle{
						RemovedIn: "2.0.0",
					},
				})
			},
		})
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"old"},
		}).Require(t,
			clitest.InspectionRejected(),
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticRemovedCommand).Present()),
		)
	})

	t.Run("not introduced argument", func(t *testing.T) {
		t.Parallel()
		type config = inspectionValueConfig
		app := clitest.Compile(t, cli.Definition[config]{
			Name:    "tool",
			Release: cli.Release.SemVer("2.0.0"),
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
					form.String(&form.Config.Value, "value").Release(cli.ReleaseLifecycle{
						IntroducedIn: "3.0.0",
					})
				})
			},
		})
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"value"},
		}).Require(t,
			clitest.InspectionRejected(),
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticNotIntroducedArgument).Present()),
		)
	})

	t.Run("removed argument", func(t *testing.T) {
		t.Parallel()
		type config = inspectionValueConfig
		app := clitest.Compile(t, cli.Definition[config]{
			Name:    "tool",
			Release: cli.Release.SemVer("2.0.0"),
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
					form.String(&form.Config.Value, "value").Release(cli.ReleaseLifecycle{
						RemovedIn: "2.0.0",
					})
				})
			},
		})
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"value"},
		}).Require(t,
			clitest.InspectionRejected(),
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticRemovedArgument).Present()),
		)
	})

	t.Run("misplaced flag", func(t *testing.T) {
		t.Parallel()
		type config = inspectionRootOnlyConfig
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Flags.Bool(&root.Config.RootOnly).Long("root-only")
				root.Subcommands.Command(cli.Command[config]{
					Name: "child",
				})
			},
		})
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"child", "--root-only"},
		}).Require(t,
			clitest.InspectionRejected(),
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticMisplacedFlag).Present()),
		)
	})

	t.Run("missing flag value", func(t *testing.T) {
		t.Parallel()
		type config = inspectionValueConfig
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Flags.String(&root.Config.Value).Long("value")
			},
		})
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"--value"},
		}).Require(t,
			clitest.InspectionRejected(),
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticMissingValue).Present()),
		)
	})

	t.Run("repeated flag", func(t *testing.T) {
		t.Parallel()
		type config = inspectionVerboseConfig2
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Flags.Bool(&root.Config.Verbose).Occurrences(cli.Count.Optional()).Long("verbose")
			},
		})
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"--verbose", "--verbose"},
		}).Require(t,
			clitest.InspectionRejected(),
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticRepeatedFlag).Present()),
		)
	})

	t.Run("ambiguous command", func(t *testing.T) {
		t.Parallel()
		app := clitest.Compile(t, cli.Definition[struct{}]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[struct{}]) {
				root.Dialect.CommandMatching(cli.NameMatchingUniquePrefix)
				root.Subcommands.Command(cli.Command[struct{}]{
					Name: "serve",
				})
				root.Subcommands.Command(cli.Command[struct{}]{
					Name: "service",
				})
			},
		})
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"ser"},
		}).Require(t,
			clitest.InspectionRejected(),
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticAmbiguousCommand).Present()),
		)
	})

	t.Run("ambiguous flag", func(t *testing.T) {
		t.Parallel()
		type config = inspectionVerboseVersionConfig
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Dialect.FlagMatching(cli.NameMatchingUniquePrefix)
				root.Flags.Bool(&root.Config.Verbose).Long("verbose")
				root.Flags.Bool(&root.Config.Version).Long("version")
			},
		})
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"--ver"},
		}).Require(t,
			clitest.InspectionRejected(),
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticAmbiguousFlag).Present()),
		)
	})

	t.Run("unavailable command", func(t *testing.T) {
		t.Parallel()
		type config = inspectionGateConfig
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				gate := root.Values.String(&root.Config.Gate)
				root.Values.Default(gate, "off")
				root.Subcommands.Command(cli.Command[config]{
					Name: "child",
					Declare: func(child *cli.CommandBuilder[config]) {
						child.AvailableWhen(child.When.Equal(gate, "on"))
					},
				})
			},
		})
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"child"},
		}).Require(t,
			clitest.InspectionRejected(),
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticUnavailableCommand).Present()),
			clitest.Command("child").ConditionAt(0).Outcome(cli.EvaluationOutcomeUnsatisfied),
		)
	})

	t.Run("unavailable flag", func(t *testing.T) {
		t.Parallel()
		type config = inspectionGateModeConfig
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				gate := root.Values.String(&root.Config.Gate)
				root.Values.Default(gate, "off")
				flag := root.Flags.String(&root.Config.Mode)
				flag.Long("mode")
				flag.AvailableWhen(root.When.Equal(gate, "on"))
			},
		})
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"--mode", "value"},
		}).Require(t,
			clitest.InspectionRejected(),
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticUnavailableFlag).Present()),
			clitest.RootCommand().ConditionAt(0).Outcome(cli.EvaluationOutcomeUnsatisfied),
		)
	})

	t.Run("unavailable argument", func(t *testing.T) {
		t.Parallel()
		type config = inspectionGateValueConfig
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				gate := root.Values.String(&root.Config.Gate)
				root.Values.Default(gate, "off")
				root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
					argument := form.String(&form.Config.Value, "value")
					argument.AvailableWhen(root.When.Equal(gate, "on"))
				})
			},
		})
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"value"},
		}).Require(t,
			clitest.InspectionRejected(),
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticUnavailableArgument).Present()),
			clitest.RootCommand().ConditionAt(0).Outcome(cli.EvaluationOutcomeUnsatisfied),
		)
	})
}

func TestProjectGrammarActionSelection(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"help"},
	}).Require(t,
		clitest.InvocationAction(cli.InvocationActionHelp).Selected(),
		clitest.InvocationAction(cli.InvocationActionHelp).Trigger(cli.InvocationActionTriggerCommand),
		clitest.ExecutionAdmission().Status(cli.ExecutionAdmissionNotApplicable),
	)
}

func inspectionOnlyInspectionCandidate(t *testing.T, inspection cli.Inspection) cli.ValueCandidateInspection {
	t.Helper()
	if len(inspection.ValueCandidates) != 1 {
		t.Fatalf("Inspection.ValueCandidates = %#v, want exactly one candidate", inspection.ValueCandidates)
	}
	for _, candidate := range inspection.ValueCandidates {
		return candidate
	}
	panic("unreachable")
}

func TestInspectionCandidateIDsAreResultLocal(t *testing.T) {
	t.Parallel()
	type config struct{ Name string }
	app := inspectionMustCompilePublic(t, cli.Definition[config]{
		Name:    "tool",
		Help:    cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) { root.Flags.String(&root.Config.Name).Long("name") },
	})

	inspect := func(value string) cli.Inspection {
		t.Helper()
		result, err := app.Inspect(t.Context(), cli.InspectionRequest{
			Args: []string{"--name", value},
		})
		if err != nil {
			t.Fatalf("Inspect(%q): %v", value, err)
		}
		if result.Status != cli.InspectionStatusReady {
			t.Fatalf("Inspect(%q).Status = %v, want Ready", value, result.Status)
		}
		return result
	}

	first := inspect("alpha")
	second := inspect("beta")
	firstCandidate := inspectionOnlyInspectionCandidate(t, first)
	secondCandidate := inspectionOnlyInspectionCandidate(t, second)
	if firstCandidate.ID != cli.ValueCandidateID(1) || secondCandidate.ID != cli.ValueCandidateID(1) {
		t.Fatalf("candidate IDs = first %v second %v, want result-local ID 1 in both artifacts", firstCandidate.ID, secondCandidate.ID)
	}
	if firstCandidate.Input == nil || secondCandidate.Input == nil || firstCandidate.Input.Text != "alpha" || secondCandidate.Input.Text != "beta" {
		t.Fatalf("candidate observations crossed result boundary: first=%#v second=%#v", firstCandidate, secondCandidate)
	}
}

const (
	inspectionSignalNativeInterrupt        = "interrupt"
	inspectionSignalNativeMutated          = "mutated"
	inspectionSignalNativeDynamic          = "dynamic"
	inspectionSignalNativeTerminate        = "terminate"
	inspectionSignalNativeFirst            = "first"
	inspectionSignalAlienctlName           = "alienctl"
	inspectionSignalRunFromProcessChildEnv = "COMMANDMENT_SIGNAL_RUN_FROM_PROCESS_CHILD"
)
