package conformance

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

type referenceNameVerboseConfig struct {
	Name    string
	Verbose bool
}

type referenceNameEventsConfig struct {
	Name   string
	Events <-chan cli.SignalEvent
}

type referenceConfig struct{}

type referenceNameSecretItemsEventsConfig struct {
	Name   string
	Secret string
	Items  []string
	Events <-chan cli.SignalEvent
}

func declareReferenceProjectionFixture(root *cli.CommandBuilder[referenceNameSecretItemsEventsConfig]) {
	name := root.Values.String(&root.Config.Name)
	root.Values.Default(name, "guest")
	secret := root.Values.String(&root.Config.Secret).Sensitive()
	root.Flags.ValueFor(secret).Description("secret").Long("secret").Hidden().Sensitive()
	items := root.Values.Sequence(&root.Config.Items, cli.Value.String())
	root.Arguments.Form(func(form *cli.ArgumentFormBuilder[referenceNameSecretItemsEventsConfig]) {
		form.Sequence(items, "item", cli.Count.ZeroOrMore()).Description("items")
	})
	root.AvailableWhen(root.When.Resolved(name))
	root.Rule.Require(name)
	root.Advisory.Always(cli.Advisory{
		Code:     "a",
		Severity: cli.NoticeSeverityInformation,
		Message:  "m",
	})
	events := root.Signals.Endpoint(&root.Config.Events)
	root.Signals.Policy(root.Signals.Match.Interrupt().Each(), root.Signals.Action.Emit(events))
	root.Subcommands.Command(cli.Command[struct{}]{
		Name: "child",
		Description: cli.Description{
			Short: "child",
		},
	})
}

func TestReferenceRemainsStaticAcrossProviderAndDomainOperations(t *testing.T) {
	t.Parallel()
	type config struct{ Mode string }
	providerCalls := 0
	domainCalls := 0
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				providerCalls++
				batch.Command(cli.Command[config]{
					Name: "remote",
				})
				return nil
			})
			domain := root.Values.Domain(
				func(_ context.Context, _ *config, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
					domainCalls++
					builder.Value("prod", "production")
					return nil
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Flags.ValueFor(mode).Long("mode")
		},
	}
	app := clitest.Compile(t, definition)

	clitest.Reference(app, cli.ReferenceOptions{}).Require(t,
		clitest.CommandsExactly(clitest.RootCommand()),
		clitest.Command("remote").DoesNotExist(),
	)
	if providerCalls != 0 || domainCalls != 0 {
		t.Fatalf("Reference() executed dynamic callbacks: providers=%d domains=%d", providerCalls, domainCalls)
	}

	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"remote"},
	}).Require(t,
		clitest.InspectionReady(),
		clitest.Command("remote").Exists(),
	)
	if providerCalls != 1 || domainCalls != 0 {
		t.Fatalf("provider Inspect calls: providers=%d domains=%d", providerCalls, domainCalls)
	}

	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--mode", "prod"},
	}).Require(t,
		clitest.InspectionReady(),
	)
	if providerCalls != 2 || domainCalls != 1 {
		t.Fatalf("domain Inspect calls: providers=%d domains=%d", providerCalls, domainCalls)
	}

	clitest.Reference(app, cli.ReferenceOptions{}).Require(t,
		clitest.CommandsExactly(clitest.RootCommand()),
		clitest.Command("remote").DoesNotExist(),
	)
	if providerCalls != 2 || domainCalls != 1 {
		t.Fatalf("second Reference() executed dynamic callbacks: providers=%d domains=%d", providerCalls, domainCalls)
	}
}

func TestRunLoadsTransactionalResponseReferenceThroughCanonicalGrammar(t *testing.T) {
	t.Parallel()
	type config = referenceConfig
	called, loads := 0, 0
	definition := cli.Definition[config]{
		Name: "commandment",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
		},
		Run: func(context.Context, *config) error {
			called++
			return nil
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatal(err)
	}
	semanticFiles := clitest.ResponseFiles{
		Loads: map[cli.ResponseFileRequest]cli.ResponseFile{
			{
				Path: "args",
			}: {
				Key:      "args",
				Revision: cli.Revision("r1"),
			},
		},
	}
	responseProof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args:          []string{"@args"},
		ResponseFiles: semanticFiles.Access(),
	})
	responseProof.Require(t, clitest.InspectionReady())
	err = app.Run(t.Context(), cli.RunRequest{
		Args: []string{"@args"},
		ResponseFiles: cli.ResponseFileAccess{
			Load: func(_ context.Context, request cli.ResponseFileRequest) (cli.ResponseFile, error) {
				loads++
				if request.Path != "args" {
					t.Fatalf("load path = %q", request.Path)
				}
				file := cli.ResponseFile{
					Key:      "args",
					Revision: cli.Revision("r1"),
					Content:  nil,
				}
				return file, nil
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if loads != 1 || called != 1 {
		t.Fatalf("loads=%d handler calls=%d", loads, called)
	}
}

func TestCompiledApplicationPublishesLocalizedReleaseFilteredReference(t *testing.T) {
	t.Parallel()
	type config = referenceConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Description: cli.Description{
			Short: "tool short",
		},
		Release: cli.Release.Ordered("2", "1", "2", "3"),
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"tool short":   "outil court",
					"alpha short":  "alpha fr",
					"hidden short": "cache fr",
				},
			}},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.Command(cli.Command[config]{
				Name:   "hidden",
				Hidden: true,
				Description: cli.Description{
					Short: "hidden short",
				},
			})
			root.Subcommands.Command(cli.Command[config]{
				Name: "old",
				Release: cli.ReleaseLifecycle{
					RemovedIn: "2",
				},
			})
			root.Subcommands.Command(cli.Command[config]{
				Name: "alpha",
				Description: cli.Description{
					Short: "alpha short",
				},
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	referenceOptions := cli.ReferenceOptions{
		Locale:     "fr",
		Visibility: cli.ReferenceVisibilityPublic,
		Order:      cli.ReferenceOrderName,
		Release:    "1",
	}
	publicProof := clitest.Reference(app, referenceOptions)
	publicProof.Require(t,
		clitest.EffectiveLocale("fr"),
		clitest.EvaluatedRelease("1"),
		clitest.CommandsExactly(
			clitest.RootCommand(),
			clitest.Command("alpha"),
			clitest.Command("completion"),
			clitest.Command("old"),
		),
		clitest.RootCommand().PresentedDescription(cli.Description{
			Short: "outil court",
		}),
		clitest.Command("old").Lifecycle(cli.LifecycleStateCurrent),
		clitest.Command("hidden").Exists(),
		clitest.Command("hidden").Hidden(),
		clitest.Command("hidden").PresentedDescription(cli.Description{
			Short: "cache fr",
		}),
	)

	referenceOptionsUnderTest := cli.ReferenceOptions{
		Locale:     "fr",
		Visibility: cli.ReferenceVisibilityAll,
		Order:      cli.ReferenceOrderDeclaration,
		Release:    "2",
	}
	allProof := clitest.Reference(app, referenceOptionsUnderTest)
	allProof.Require(t,
		clitest.EffectiveLocale("fr"),
		clitest.EvaluatedRelease("2"),
		clitest.CommandsExactly(
			clitest.RootCommand(),
			clitest.Command("hidden"),
			clitest.Command("alpha"),
			clitest.Command("completion"),
		),
		clitest.Command("old").Exists(),
		clitest.Command("old").Lifecycle(cli.LifecycleStateRemoved),
		clitest.Command("hidden").Hidden(),
		clitest.Command("hidden").PresentedDescription(cli.Description{
			Short: "cache fr",
		}),
	)
	if t.Failed() {
		return
	}

	all, _ := allProof.Reference()
	all.CommandOrder[0] = 0

	referenceOptionsUnderTest2 := cli.ReferenceOptions{
		Locale:     "fr",
		Visibility: cli.ReferenceVisibilityAll,
		Order:      cli.ReferenceOrderDeclaration,
		Release:    "2",
	}
	clitest.Reference(app, referenceOptionsUnderTest2).Require(t,
		clitest.EffectiveLocale("fr"),
		clitest.EvaluatedRelease("2"),
		clitest.CommandsExactly(
			clitest.RootCommand(),
			clitest.Command("hidden"),
			clitest.Command("alpha"),
			clitest.Command("completion"),
		),
		clitest.Command("old").Lifecycle(cli.LifecycleStateRemoved),
		clitest.Command("hidden").PresentedDescription(cli.Description{
			Short: "cache fr",
		}),
	)

	if _, err := app.Reference(cli.ReferenceOptions{
		Release: "missing",
	}); err == nil {
		t.Fatal("Reference(invalid release) error = nil")
	}
}

func TestPositionalFormSelectionClassifiesFixedAllocationsWithoutDeclarationOrderPreference(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		a          referencePositionalFormOutcome
		b          referencePositionalFormOutcome
		wantStatus cli.ArgumentFormSelectionOutcome
		wantMode   string
		wantCode   cli.DiagnosticCode
		wantFail   bool
	}{
		{
			name:       "only A accepts",
			a:          referencePositionalFormAccept,
			b:          referencePositionalFormReject,
			wantStatus: cli.ArgumentFormSelectionSelected,
			wantMode:   "A",
		},
		{
			name:       "only B accepts",
			a:          referencePositionalFormReject,
			b:          referencePositionalFormAccept,
			wantStatus: cli.ArgumentFormSelectionSelected,
			wantMode:   "B",
		},
		{
			name:       "both accept",
			a:          referencePositionalFormAccept,
			b:          referencePositionalFormAccept,
			wantStatus: cli.ArgumentFormSelectionAmbiguous,
			wantCode:   cli.DiagnosticAmbiguousArgumentForm,
		},
		{
			name:       "both reject",
			a:          referencePositionalFormReject,
			b:          referencePositionalFormReject,
			wantStatus: cli.ArgumentFormSelectionRejected,
			wantCode:   cli.DiagnosticInvalidValue,
		},
		{
			name:       "accept and operational failure",
			a:          referencePositionalFormAccept,
			b:          referencePositionalFormFail,
			wantStatus: cli.ArgumentFormSelectionFailed,
			wantCode:   cli.DiagnosticInputParserFailure,
			wantFail:   true,
		},
	}
	for _, tc := range cases {
		for _, reverse := range []bool{false, true} {
			name := tc.name + "/forward"
			if reverse {
				name = tc.name + "/reverse"
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				fixture := referenceNewFormSelectionFixture(t, tc.a, tc.b, reverse)
				inspectProof := clitest.Inspect(t.Context(), fixture.app, cli.InspectionRequest{
					Args: []string{"x"},
				})
				expectations := []clitest.InspectionExpectation{
					clitest.RootCommand().ArgumentFormSelectionOutcome(tc.wantStatus),
					clitest.SelectionPathExactly(clitest.RootCommand()),
				}
				if tc.wantCode != "" {
					expectations = append(expectations, clitest.FailsWith(clitest.Diagnostic(tc.wantCode).Present()))
				} else {
					expectations = append(expectations, clitest.InspectionReady())
				}

				formA := clitest.RootCommand().ArgumentFormAt(0)
				formB := clitest.RootCommand().ArgumentFormAt(1)
				if reverse {
					formA, formB = formB, formA
				}
				if tc.wantMode != "" {
					wantForm, loseForm := formA, formB
					wantValue, loseValue := fixture.valueAID, fixture.valueBID
					if tc.wantMode == "B" {
						wantForm, loseForm = formB, formA
						wantValue, loseValue = fixture.valueBID, fixture.valueAID
					}
					expectations = append(expectations,
						wantForm.Selected(),
						loseForm.NotSelected(),
						clitest.ValueByID(wantValue).Resolved(),
						clitest.ValueByID(loseValue).StateIfObserved(
							cli.ResolutionStateUnknown,
							cli.ResolutionStateInactive,
							cli.ResolutionStateAbsent,
							cli.ResolutionStateFailed,
							cli.ResolutionStateBlocked,
						),
					)
				} else {
					nonResolved := []cli.ResolutionState{
						cli.ResolutionStateUnknown,
						cli.ResolutionStateInactive,
						cli.ResolutionStateAbsent,
						cli.ResolutionStateFailed,
						cli.ResolutionStateBlocked,
					}
					expectations = append(expectations,
						clitest.ValueByID(fixture.valueAID).StateIfObserved(nonResolved...),
						clitest.ValueByID(fixture.valueBID).StateIfObserved(nonResolved...),
					)
				}
				inspectProof.Require(t, expectations...)

				if fixture.callsA.Load() != 1 || fixture.callsB.Load() != 1 {
					t.Fatalf(
						"conversion calls = A:%d B:%d, want each structurally viable form evaluated once",
						fixture.callsA.Load(),
						fixture.callsB.Load(),
					)
				}
				_, err := inspectProof.Inspection()
				if tc.wantFail {
					if err == nil || !errors.Is(err, fixture.failure) {
						t.Fatalf("Inspect() error = %v, want operational failure %v", err, fixture.failure)
					}
				} else if tc.wantStatus == cli.ArgumentFormSelectionSelected && err != nil {
					t.Fatalf("Inspect() error = %v", err)
				}
			})
		}
	}
}

func TestProjectOperationReferencePreservesReferenceProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = referenceNameSecretItemsEventsConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareReferenceProjectionFixture,
	}
	app := clitest.Compile(t, definition)
	clitest.Schema(app).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
	clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityAll,
	}).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
}

func TestProjectReferencePreservesReferenceProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = referenceNameSecretItemsEventsConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareReferenceProjectionFixture,
	}
	app := clitest.Compile(t, definition)
	clitest.Schema(app).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
	clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityAll,
	}).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
}

func TestReferenceCommandIsAvailablePreservesReferenceProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = referenceNameSecretItemsEventsConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareReferenceProjectionFixture,
	}
	app := clitest.Compile(t, definition)
	clitest.Schema(app).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
	clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityAll,
	}).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
}

func TestReferenceVisibleCommandPreservesReferenceProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = referenceNameSecretItemsEventsConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareReferenceProjectionFixture,
	}
	app := clitest.Compile(t, definition)
	clitest.Schema(app).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
	clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityAll,
	}).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
}

func TestProjectReferenceCommandsPreservesReferenceProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = referenceNameSecretItemsEventsConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareReferenceProjectionFixture,
	}
	app := clitest.Compile(t, definition)
	clitest.Schema(app).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
	clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityAll,
	}).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
}

func TestProjectReferenceDeclarationsPreservesReferenceProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = referenceNameSecretItemsEventsConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareReferenceProjectionFixture,
	}
	app := clitest.Compile(t, definition)
	clitest.Schema(app).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
	clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityAll,
	}).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
}

func TestProjectReferenceValuesAndSignalsPreservesReferenceProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = referenceNameSecretItemsEventsConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareReferenceProjectionFixture,
	}
	app := clitest.Compile(t, definition)
	clitest.Schema(app).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
	clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityAll,
	}).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
}

func TestReferenceLifecycleStatePreservesReferenceProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = referenceNameSecretItemsEventsConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareReferenceProjectionFixture,
	}
	app := clitest.Compile(t, definition)
	clitest.Schema(app).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
	clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityAll,
	}).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
}

func TestReferenceHelpForCommandPreservesReferenceProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = referenceNameSecretItemsEventsConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareReferenceProjectionFixture,
	}
	app := clitest.Compile(t, definition)
	clitest.Schema(app).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
	clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityAll,
	}).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
}

func TestReferenceCommandOrderPreservesReferenceProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = referenceNameSecretItemsEventsConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareReferenceProjectionFixture,
	}
	app := clitest.Compile(t, definition)
	clitest.Schema(app).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
	clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityAll,
	}).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
}

func TestReferenceCommandEntryVisiblePreservesReferenceProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = referenceNameSecretItemsEventsConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareReferenceProjectionFixture,
	}
	app := clitest.Compile(t, definition)
	clitest.Schema(app).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
	clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityAll,
	}).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
}

func TestReferenceActionEntryVisiblePreservesReferenceProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = referenceNameSecretItemsEventsConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareReferenceProjectionFixture,
	}
	app := clitest.Compile(t, definition)
	clitest.Schema(app).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
	clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityAll,
	}).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
}

func TestReferenceFlagVisiblePreservesReferenceProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = referenceNameSecretItemsEventsConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareReferenceProjectionFixture,
	}
	app := clitest.Compile(t, definition)
	clitest.Schema(app).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
	clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityAll,
	}).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
}

func TestReferenceFlagSpellingVisiblePreservesReferenceProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = referenceNameSecretItemsEventsConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareReferenceProjectionFixture,
	}
	app := clitest.Compile(t, definition)
	clitest.Schema(app).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
	clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityAll,
	}).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
}

func TestReferenceFormVisiblePreservesReferenceProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = referenceNameSecretItemsEventsConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareReferenceProjectionFixture,
	}
	app := clitest.Compile(t, definition)
	clitest.Schema(app).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
	clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityAll,
	}).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
}

func TestReferenceArgumentTermVisiblePreservesReferenceProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = referenceNameSecretItemsEventsConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareReferenceProjectionFixture,
	}
	app := clitest.Compile(t, definition)
	clitest.Schema(app).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
	clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityAll,
	}).Require(t,
		clitest.RootCommand().Exists(),
		clitest.Command("child").Exists(),
	)
}

func TestProjectReference(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
	})
	if _, err := clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibility(255),
		Order:      cli.ReferenceOrderDeclaration,
	}).Reference(); err == nil {
		t.Fatal("invalid visibility accepted")
	}
	if _, err := clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityPublic,
		Order:      cli.ReferenceOrder(255),
	}).Reference(); err == nil {
		t.Fatal("invalid order accepted")
	}
}

func TestProjectReferenceDeclarations(t *testing.T) {
	t.Parallel()
	type config = referenceNameVerboseConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			name := root.Values.String(&root.Config.Name)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) { form.ValueFor(name, "name") })
			root.Flags.Bool(&root.Config.Verbose).Long("verbose")
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "child",
			})
		},
	})

	clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityAll,
		Order:      cli.ReferenceOrderDeclaration,
	}).Require(t,
		clitest.CommandsExactly(
			clitest.RootCommand(),
			clitest.Command("child"),
			clitest.Command("completion"),
		),
		clitest.RootCommand().FlagsExactly(clitest.RootCommand().Flag("--verbose")),
		clitest.RootCommand().ArgumentFormAt(0).TermsExactly(clitest.RootCommand().Argument("name")),
	)
}

func TestProjectReferenceValuesAndSignals(t *testing.T) {
	t.Parallel()
	type config = referenceNameEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
				values.Value("one", "one")
			})
			root.Values.Value(&root.Config.Name, domain)
			events := root.Signals.Endpoint(&root.Config.Events)
			root.Signals.Policy(root.Signals.Match.Interrupt().Each(), root.Signals.Action.Emit(events))
		},
	})

	clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityAll,
		Order:      cli.ReferenceOrderDeclaration,
	}).Require(t,
		clitest.RootCommand().ValueAt(0).Exists(),
		clitest.RootCommand().SignalPoliciesExactly(clitest.RootCommand().SignalPolicyAt(0)),
	)
}

type referenceTokenLanguageRuntimeABConfig struct {
	A string
	B string
}

type referencePositionalFormOutcome uint8

const (
	referencePositionalFormAccept referencePositionalFormOutcome = iota
	referencePositionalFormReject
	referencePositionalFormFail
)

func referenceFormSelectionCodec(name string, outcome referencePositionalFormOutcome, failure error, calls *atomic.Int32) cli.ValueCodec[string] {
	codec := cli.ValueCodec[string]{
		Name: name,
		Parse: func(text string) (string, error) {
			calls.Add(1)
			switch outcome {
			case referencePositionalFormAccept:
				return name + "=" + text, nil
			case referencePositionalFormReject:
				return "", cli.Rejection.Message(
					"form-reject",
					"form rejected input",
					cli.Rejection.At(0, 0, len(text)),
				)
			case referencePositionalFormFail:
				return "", failure
			default:
				panic("unknown form outcome")
			}
		},
		Format:   func(value string) string { return value },
		Snapshot: cli.Value.Assign[string](),
	}
	return codec
}

type referenceFormSelectionFixture struct {
	app      cli.Application
	valueAID cli.ValueID
	valueBID cli.ValueID
	callsA   *atomic.Int32
	callsB   *atomic.Int32
	failure  error
}

func referenceNewFormSelectionFixture(t *testing.T, a, b referencePositionalFormOutcome, reverse bool) referenceFormSelectionFixture {
	t.Helper()
	type config = referenceTokenLanguageRuntimeABConfig
	callsA := &atomic.Int32{}
	callsB := &atomic.Int32{}
	failure := errors.New("form-b operational failure")
	var valueA cli.Destination[config, string]
	var valueB cli.Destination[config, string]
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			valueA = root.Values.Value(&root.Config.A, referenceFormSelectionCodec("A", a, failure, callsA))
			valueB = root.Values.Value(&root.Config.B, referenceFormSelectionCodec("B", b, failure, callsB))
			declareA := func() {
				root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) { form.ValueFor(valueA, "a") })
			}
			declareB := func() {
				root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) { form.ValueFor(valueB, "b") })
			}
			if reverse {
				declareB()
				declareA()
			} else {
				declareA()
				declareB()
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	schema := app.Schema()
	var valueAID, valueBID cli.ValueID
	for _, term := range schema.ArgumentTerms {
		switch term.Name {
		case "a":
			valueAID = term.Destination
		case "b":
			valueBID = term.Destination
		}
	}
	if valueAID == 0 || valueBID == 0 || valueAID == valueBID {
		t.Fatalf("argument destination identities a=%v b=%v terms=%#v", valueAID, valueBID, schema.ArgumentTerms)
	}
	if _, ok := schema.Values[valueAID]; !ok {
		t.Fatalf("argument a destination %v missing from schema values", valueAID)
	}
	if _, ok := schema.Values[valueBID]; !ok {
		t.Fatalf("argument b destination %v missing from schema values", valueBID)
	}
	fixture := referenceFormSelectionFixture{
		app:      app,
		valueAID: valueAID,
		valueBID: valueBID,
		callsA:   callsA,
		callsB:   callsB,
		failure:  failure,
	}
	return fixture
}
