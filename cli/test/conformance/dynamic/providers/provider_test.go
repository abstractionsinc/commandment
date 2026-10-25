package conformance

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

func TestProviderAuthoringProjectsDeclaredSemantics(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.CommandsFrom(func(context.Context, *cli.CommandBatch[struct{}]) error { return nil })
		},
	})
	root := clitest.RootCommand()
	provider := root.ProviderAt(0)
	clitest.Schema(app).Require(t,
		root.ProvidersExactly(provider),
		provider.Exists(),
		provider.DependenciesExactly(),
	)
}

func TestProviderCommandDeclarationUsesParentScopeForHeterogeneousReplacement(t *testing.T) {
	t.Parallel()
	type rootConfig struct{}
	type legacyConfig struct{ Name string }
	type currentConfig struct{ Name string }

	definition := cli.Definition[rootConfig]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[rootConfig]) error {
				legacy := batch.Command(cli.Command[legacyConfig]{
					Name: "legacy",
					Release: cli.ReleaseLifecycle{
						DeprecatedIn: "2",
					},
				})
				current := batch.Command(cli.Command[currentConfig]{
					Name: "current",
					Release: cli.ReleaseLifecycle{
						IntroducedIn: "1",
					},
					Declare: func(child *cli.CommandBuilder[currentConfig]) {
						child.Flags.String(&child.Config.Name).Long("name")
					},
				})
				legacy.ReplacedBy(current)
				oldShortcut := batch.Shortcut(current, cli.CommandShortcut{
					Name:       "old-quick",
					AppendArgs: []string{"--name", "old"},
				})
				newShortcut := batch.Shortcut(current, cli.CommandShortcut{
					Name:       "new-quick",
					AppendArgs: []string{"--name", "new"},
				})
				oldShortcut.ReplacedBy(newShortcut).DefaultFlag("--name", []string{"legacy"})
				return nil
			})
		},
	}
	app := clitest.Compile(t, definition)

	root := clitest.RootCommand()
	legacy := root.Command("legacy").CanonicalEntry()
	current := root.Command("current").CanonicalEntry()
	oldShortcut := root.Shortcut("old-quick")
	newShortcut := root.Shortcut("new-quick")
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		clitest.InspectionReady(),
		root.ProviderAt(0).Outcome(cli.CommandProviderOutcomeAccepted),
		legacy.ReplacedBy(current),
		oldShortcut.ReplacedBy(newShortcut),
		oldShortcut.ShortcutDefaultCount(1),
		oldShortcut.ShortcutDefaultAt(0).Spelling("--name"),
	)
}

type providerBuilderMisuseConfig struct {
	Value string
}
type concurrencyLocaleRootConfig struct{ Locale string }

type concurrencyLocaleChildConfig struct{ Locale string }

type concurrencyProfileRootConfig struct{ Profile string }

type concurrencyProfileChildConfig struct{ Profile string }

type providerConcurrencyConfig struct{}

func TestProviderFailurePartialInspectionIsDetachedAndOperationLocal(t *testing.T) {
	t.Parallel()
	type config = providerConcurrencyConfig
	acceptedCalls := 0
	failedCalls := 0
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				acceptedCalls++
				batch.Command(cli.Command[config]{
					Name: "survivor",
				})
				return nil
			})
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				failedCalls++
				batch.Command(cli.Command[config]{
					Name: "discarded",
				})
				return errors.New("provider failure")
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	check := func(label string, inspection cli.Inspection, err error, wantCalls int) cli.CommandEntryID {
		t.Helper()
		if err == nil || inspection.Status != cli.InspectionStatusRejected || acceptedCalls != wantCalls ||
			failedCalls != wantCalls {
			t.Fatalf(
				"%s partial Inspection = status %v calls %d/%d err %v",
				label,
				inspection.Status,
				acceptedCalls,
				failedCalls,
				err,
			)
		}
		var root cli.CommandInspection
		var survivor, discarded bool
		for _, command := range inspection.Commands {
			switch command.Name {
			case "tool":
				root = command
			case "survivor":
				survivor = true
			case "discarded":
				discarded = true
			}
		}
		if !survivor || discarded || root.NamespaceCompleteness != cli.CommandNamespaceIncomplete {
			t.Fatalf("%s provider containment = survivor %v discarded %v root %#v", label, survivor, discarded, root)
		}
		var acceptedEntry cli.CommandEntryID
		var failedProvider cli.CommandProviderID
		accepted, failed := 0, 0
		for providerID, provider := range inspection.CommandProviders {
			switch provider.Outcome {
			case cli.CommandProviderOutcomeAccepted:
				accepted++
				if len(provider.CommandEntries) != 1 {
					t.Fatalf("%s accepted entries = %#v", label, provider.CommandEntries)
				}
				acceptedEntry = provider.CommandEntries[0]
				entry, ok := inspection.CommandEntries[acceptedEntry]
				if !ok || entry.Provider != providerID || entry.Origin != cli.DeclarationOriginProvider {
					t.Fatalf("%s accepted entry closure = %#v", label, entry)
				}
			case cli.CommandProviderOutcomeFailed:
				failed++
				failedProvider = providerID
				if len(provider.CommandEntries) != 0 {
					t.Fatalf("%s failed provider leaked committed entries: %#v", label, provider.CommandEntries)
				}
			case cli.CommandProviderOutcomeUnknown,
				cli.CommandProviderOutcomeNotNeeded,
				cli.CommandProviderOutcomeRejected,
				cli.CommandProviderOutcomeInvalid,
				cli.CommandProviderOutcomeBlocked,
				cli.CommandProviderOutcomeCanceled:
			}
		}
		if accepted != 1 || failed != 1 || acceptedEntry == 0 || failedProvider == 0 {
			t.Fatalf("%s provider states = %#v", label, inspection.CommandProviders)
		}
		if len(inspection.Diagnostics) != 1 || inspection.Diagnostics[0].Code != cli.DiagnosticCommandProviderFailure ||
			inspection.Diagnostics[0].Provider != failedProvider {
			t.Fatalf(
				"%s provider diagnostic closure = %#v, failed provider %v",
				label,
				inspection.Diagnostics,
				failedProvider,
			)
		}
		return acceptedEntry
	}

	first, firstErr := func() (cli.Inspection, error) {
		proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"survivor"},
		})
		proof.Require(
			t,
			clitest.InspectionRejected(),
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticCommandProviderFailure).Present()),
		)
		return proof.Inspection()
	}()
	entry := check("first", first, firstErr, 1)
	delete(first.CommandEntries, entry)
	for id, provider := range first.CommandProviders {
		provider.CommandEntries = nil
		first.CommandProviders[id] = provider
	}
	for id, command := range first.Commands {
		if command.Name == "tool" {
			command.NamespaceCompleteness = cli.CommandNamespaceComplete
			first.Commands[id] = command
		}
	}
	first.Diagnostics = nil

	second, secondErr := app.Inspect(t.Context(), cli.InspectionRequest{
		Args: []string{"survivor"},
	})
	check("second", second, secondErr, 2)
}

func TestInvalidProviderDeclarationValidationIsOperationLocal(t *testing.T) {
	t.Parallel()
	type rootConfig = concurrencyLocaleRootConfig
	type childConfig = concurrencyLocaleChildConfig
	calls := 0
	definition := cli.Definition[rootConfig]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
		},
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Flags.String(&root.Config.Locale).LocaleSelector().Occurrences(cli.Count.Optional()).
				Long("locale")
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[rootConfig]) error {
				calls++
				batch.Command(cli.Command[childConfig]{
					Name: "remote",
					Declare: func(child *cli.CommandBuilder[childConfig]) {
						child.Flags.String(&child.Config.Locale).LocaleSelector().Occurrences(cli.Count.Optional()).
							Long("locale")
					},
				})
				return nil
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatal(err)
	}
	for run := 1; run <= 2; run++ {
		inspection, err := func() (cli.Inspection, error) {
			proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
				Args: []string{"--locale", "fr", "remote"},
			})
			proof.Require(
				t,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidCommandBatch).Present()),
			)
			return proof.Inspection()
		}()
		if err == nil || calls != run || len(inspection.Diagnostics) != 1 ||
			inspection.Diagnostics[0].Code != cli.DiagnosticInvalidCommandBatch {
			t.Fatalf(
				"run %d invalid provider declaration = calls %d diagnostics %#v err=%v",
				run,
				calls,
				inspection.Diagnostics,
				err,
			)
		}
	}
}

func TestInspectionDemandMaterializesCanonicalProviderProgramWithAccessAndOperationIsolation(t *testing.T) {
	t.Parallel()
	type rootConfig = concurrencyProfileRootConfig
	type childConfig = concurrencyProfileChildConfig

	calls := 0
	app, err := cli.Compile(cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			profile := root.Values.String(&root.Config.Profile)
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[rootConfig]) error {
				calls++
				child := batch.Command(cli.Command[childConfig]{
					Name: "remote",
					Declare: func(child *cli.CommandBuilder[childConfig]) {
						child.Access.Value(profile, &child.Config.Profile)
					},
				})
				batch.Alias(child, cli.CommandAlias{
					Name: "r",
				})
				batch.Shortcut(child, cli.CommandShortcut{
					Name: "old",
					Args: []string{"--reference"},
				})
				batch.Shortcut(cli.CommandPath{"remote"}, cli.CommandShortcut{
					Name: "quick",
					Args: []string{"--fast"},
				})
				return nil
			})
		},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if calls != 0 {
		t.Fatalf("Compile() executed provider %d times", calls)
	}

	inspection, err := func() (cli.Inspection, error) {
		proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"remote"},
		})
		proof.Require(t, clitest.InspectionReady())
		return proof.Inspection()
	}()
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("provider calls after first Inspect = %d, want 1", calls)
	}
	if inspection.Status != cli.InspectionStatusReady || inspection.Selection.Command.Selected == 0 {
		t.Fatalf("inspection status/selection = %#v", inspection)
	}

	var root cli.CommandInspection
	var remote cli.CommandInspection
	for _, command := range inspection.Commands {
		switch command.Name {
		case "tool":
			root = command
		case "remote":
			remote = command
		}
	}
	if root.ID == 0 || remote.ID == 0 {
		t.Fatalf("materialized commands missing: root=%#v remote=%#v", root, remote)
	}
	if root.NamespaceCompleteness != cli.CommandNamespaceComplete {
		t.Fatalf("root namespace state = %v", root.NamespaceCompleteness)
	}
	if len(root.Values) != 1 || len(remote.Access.Values) != 1 || remote.Access.Values[0] != root.Values[0] {
		t.Fatalf("provider-produced Access = %#v; root values = %#v", remote.Access, root.Values)
	}
	if len(inspection.CommandProviders) != 1 {
		t.Fatalf("providers = %#v", inspection.CommandProviders)
	}
	for _, provider := range inspection.CommandProviders {
		if provider.Outcome != cli.CommandProviderOutcomeAccepted || len(provider.CommandEntries) != 4 {
			t.Fatalf("provider inspection = %#v", provider)
		}
		for _, entryID := range provider.CommandEntries {
			entry := inspection.CommandEntries[entryID]
			if entry.Provider != provider.ID {
				t.Fatalf("entry %v provider = %v, want %v", entryID, entry.Provider, provider.ID)
			}
		}
	}

	// A later public operation owns a fresh provider execution and fresh
	// operation-local declaration identities; the compiled Application is not a
	// provider-output cache.
	second, err := app.Inspect(t.Context(), cli.InspectionRequest{
		Args: []string{"remote"},
	})
	if err != nil {
		t.Fatalf("second Inspect() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("provider calls after second Inspect = %d, want 2", calls)
	}
	if second.Selection.Command.Selected == 0 {
		t.Fatalf("second selection = %#v", second.Selection)
	}
}

func errorsNew(text string) error {
	failure := &externalGuardrailError{
		text: text,
	}
	return failure
}

type externalGuardrailError struct{ text string }

func (e *externalGuardrailError) Error() string { return e.text }
func providerHasDiagnosticCode(err error, code cli.DiagnosticCode) bool {
	for _, diagnostic := range cli.Diagnostics(err).Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func providerRequireDiagnosticCode(t testing.TB, err error, code cli.DiagnosticCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want diagnostic %q", code)
	}
	if !providerHasDiagnosticCode(err, code) {
		t.Fatalf("diagnostics = %#v, want code %q", cli.Diagnostics(err).Diagnostics, code)
	}
}

func providerMustCompilePublic[C any](t testing.TB, definition cli.Definition[C]) cli.Application {
	t.Helper()
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return app
}

type operationPreparationFactConfig struct{ Fact string }

func TestRunAndInspectionProviderIncompleteRemainIncomplete(t *testing.T) {
	t.Parallel()
	type config = operationPreparationFactConfig
	handlerCalled := false
	definition := cli.Definition[config]{
		Name: "provider-incomplete",
		Declare: func(root *cli.CommandBuilder[config]) {
			fact := root.Facts.Require(&root.Config.Fact)
			root.Subcommands.CommandsFrom(func(context.Context, *cli.CommandBatch[config]) error { return nil }, fact)
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

func TestRunAndInspectionProviderFailureRemainRejected(t *testing.T) {
	t.Parallel()
	cause := errors.New("provider failure")
	definition := cli.Definition[struct{}]{
		Name: "provider-failure",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.CommandsFrom(func(context.Context, *cli.CommandBatch[struct{}]) error { return cause })
		},
		Run: func(context.Context, *struct{}) error { return nil },
	}
	app := clitest.Compile(t, definition)
	args := []string{"remote"}

	if err := app.Run(t.Context(), cli.RunRequest{
		Args: args,
	}); !errors.Is(err, cause) {
		t.Fatalf("Run() error = %v, want %v", err, cause)
	}
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: args,
	})
	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticCommandProviderFailure).Present()),
	)
	_, inspectErr := proof.Inspection()
	if !errors.Is(inspectErr, cause) {
		t.Fatalf("Inspect() error = %v, want %v", inspectErr, cause)
	}
}

func TestRunAndInspectionFactSupplyFailureRemainRejected(t *testing.T) {
	t.Parallel()
	type config = operationPreparationFactConfig
	cause := errors.New("fact supply failure")
	definition := cli.Definition[config]{
		Name: "supply-failure",
		Declare: func(root *cli.CommandBuilder[config]) {
			fact := root.Facts.Require(&root.Config.Fact)
			root.Subcommands.CommandsFrom(func(context.Context, *cli.CommandBatch[config]) error { return nil }, fact)
		},
		Run: func(context.Context, *config) error { return nil },
	}
	app := clitest.Compile(t, definition)
	facts := []cli.FactProvider{
		cli.FactsFor[config](func(context.Context, *cli.FactSupply[config]) error { return cause }),
	}

	if err := app.Run(t.Context(), cli.RunRequest{
		Facts: facts,
	}); !errors.Is(err, cause) {
		t.Fatalf("Run() error = %v, want %v", err, cause)
	}
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Facts: facts,
	})
	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticFactSupplyFailure).Present()),
	)
	_, inspectErr := proof.Inspection()
	if !errors.Is(inspectErr, cause) {
		t.Fatalf("Inspect() error = %v, want %v", inspectErr, cause)
	}
}

func TestRunPreservesFactSupplyFailureDuringResponseGrammar(t *testing.T) {
	t.Parallel()
	type config = operationPreparationFactConfig
	cause := errors.New("fact supply")
	definition := cli.Definition[config]{
		Name: "response-fact",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
			fact := root.Facts.Require(&root.Config.Fact)
			root.Subcommands.CommandsFrom(func(context.Context, *cli.CommandBatch[config]) error { return nil }, fact)
		},
		Run: func(context.Context, *config) error { return nil },
	}
	app := clitest.Compile(t, definition)
	request := cli.RunRequest{
		Args:  []string{"@source"},
		Facts: []cli.FactProvider{cli.FactsFor[config](func(context.Context, *cli.FactSupply[config]) error { return cause })},
		ResponseFiles: cli.ResponseFileAccess{
			Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
				file := cli.ResponseFile{
					Key:      "source",
					Revision: cli.Revision("r1"),
				}
				return file, nil
			},
		},
	}

	if err := app.Run(t.Context(), request); !errors.Is(err, cause) {
		t.Fatalf("Run() error = %v, want %v", err, cause)
	}
}

func TestRunPreservesFactSupplyFailureDuringGrammarAnalysis(t *testing.T) {
	t.Parallel()
	type config = operationPreparationFactConfig
	cause := errors.New("fact supply")
	definition := cli.Definition[config]{
		Name: "grammar-fact",
		Declare: func(root *cli.CommandBuilder[config]) {
			fact := root.Facts.Require(&root.Config.Fact)
			root.Subcommands.CommandsFrom(func(context.Context, *cli.CommandBatch[config]) error { return nil }, fact)
		},
		Run: func(context.Context, *config) error { return nil },
	}
	app := clitest.Compile(t, definition)
	request := cli.RunRequest{
		Facts: []cli.FactProvider{cli.FactsFor[config](func(context.Context, *cli.FactSupply[config]) error { return cause })},
	}

	if err := app.Run(t.Context(), request); !errors.Is(err, cause) {
		t.Fatalf("Run() error = %v, want %v", err, cause)
	}
}

const providerAuthoringCommandmentName = "commandment"

type providerAuthoringRegistryConfig struct{ Registry string }

type providerAuthoringValueConfig struct{ Value string }

func TestDynamicProviderClassifiesActiveBuilderLifecycleMisuse(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[struct{}]) error {
				batch.Command(cli.Command[providerBuilderMisuseConfig]{
					Name: "child",
					Declare: func(child *cli.CommandBuilder[providerBuilderMisuseConfig]) {
						child.Arguments.Form(func(_ *cli.ArgumentFormBuilder[providerBuilderMisuseConfig]) {
							child.Values.String(&child.Config.Value)
						})
					},
				})
				return nil
			})
		},
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticBuilderMisuse).Present()),
		clitest.RootCommand().ProviderAt(0).Outcome(cli.CommandProviderOutcomeRejected),
		clitest.RootCommand().ProviderAt(0).EntriesExactly(),
	)
}

func TestDefinitionAuthoringLowersCommandProviderPlanWithoutExecutingIt(t *testing.T) {
	t.Parallel()
	type config = providerAuthoringRegistryConfig
	called := false
	definition := cli.Definition[config]{
		Name: providerAuthoringCommandmentName,
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			registry := root.Values.String(&root.Config.Registry)
			root.Values.Default(registry, "ready")
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				called = true
				if batch.Config.Registry != "ready" {
					return errorsNew("bad registry")
				}
				batch.Command(cli.Command[struct{}]{
					Name: "dynamic",
				})
				return nil
			}, registry.OptionalDependency())
		},
	}
	app := clitest.Compile(t, definition)
	if called {
		t.Fatal("Compile executed provider callback")
	}
	provider := clitest.RootCommand().ProviderAt(0)
	clitest.Schema(app).Require(t,
		clitest.RootCommand().ProvidersExactly(provider),
		provider.DependenciesExactly(clitest.RootCommand().ValueAt(0)),
		clitest.CommandsExactly(clitest.RootCommand()),
	)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		clitest.InspectionReady(),
		provider.Outcome(cli.CommandProviderOutcomeAccepted),
	)
	if !called {
		t.Fatal("provider callback did not execute at operation time")
	}
}

func TestDefinitionAuthoringRejectsInvalidCommandProviderDeclarations(t *testing.T) {
	t.Parallel()
	t.Run("nil provider", func(t *testing.T) {
		t.Parallel()
		var provider cli.CommandProvider[struct{}]
		_, err := cli.Compile(cli.Definition[struct{}]{
			Name:    providerAuthoringCommandmentName,
			Declare: func(root *cli.CommandBuilder[struct{}]) { root.Subcommands.CommandsFrom(provider) },
		})
		clitest.Diagnostics(err).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
		providerRequireDiagnosticCode(t, err, cli.DiagnosticInvalidDeclaration)
	})
	t.Run("duplicate dependency", func(t *testing.T) {
		t.Parallel()
		type config = providerAuthoringValueConfig
		_, err := cli.Compile(cli.Definition[config]{
			Name: providerAuthoringCommandmentName,
			Declare: func(root *cli.CommandBuilder[config]) {
				value := root.Values.String(&root.Config.Value)
				root.Subcommands.CommandsFrom(
					func(context.Context, *cli.CommandBatch[config]) error { return nil },
					value,
					value,
				)
			},
		})
		clitest.Diagnostics(err).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
		providerRequireDiagnosticCode(t, err, cli.DiagnosticInvalidDeclaration)
	})
}

type providerBatchConfig struct{}

func TestProviderBatchCaptureRejectsInvalidAtomicContentAndCallbackFailure(t *testing.T) {
	t.Parallel()
	type config = providerBatchConfig
	providers := []struct {
		name     string
		provider cli.CommandProvider[config]
	}{
		{
			name: "alias path is not immediate",
			provider: func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Alias(cli.CommandPath{"parent", "child"}, cli.CommandAlias{
					Name: "alias",
				})
				return nil
			},
		},
		{
			name: "duplicate localization",
			provider: func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Localization(cli.Localization{
					DefaultLocale: "en",
				})
				batch.Localization(cli.Localization{
					DefaultLocale: "en",
				})
				return nil
			},
		},
		{
			name: "empty localization when explicitly declared",
			provider: func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Localization(cli.Localization{})
				return nil
			},
		},
		{
			name: "textual shortcut default requires values",
			provider: func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Shortcut(cli.CommandPath{"child"}, cli.CommandShortcut{
					Name: "quick",
				}).DefaultFlag("--mode", nil)
				return nil
			},
		},
	}
	for _, tc := range providers {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			app := providerMustCompilePublic(t, cli.Definition[config]{
				Name:    "tool",
				Declare: func(root *cli.CommandBuilder[config]) { root.Subcommands.CommandsFrom(tc.provider) },
			})
			clitest.Inspect(t.Context(), app, cli.InspectionRequest{
				Args: []string{"missing"},
			}).Require(
				t,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidCommandBatch).Present()),
			)
		})
	}

	callbackErr := errors.New("provider failed")
	app := providerMustCompilePublic(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Command(cli.Command[struct{}]{
					Name: "discarded",
				})
				return callbackErr
			})
		},
	})
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"discarded"},
	})
	proof.Require(
		t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticCommandProviderFailure).Present()),
	)
	_, err := proof.Inspection()
	if !errors.Is(err, callbackErr) {
		t.Fatalf("callback failure = %v, want %v", err, callbackErr)
	}
}

func TestProviderBatchCapabilityExpiresAfterCallback(t *testing.T) {
	t.Parallel()
	type config = providerBatchConfig
	var retained *cli.CommandBatch[config]
	app := providerMustCompilePublic(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				retained = batch
				batch.Command(cli.Command[config]{
					Name: "accepted",
				})
				return nil
			})
		},
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"accepted"},
	}).Require(
		t,
		clitest.InspectionReady(),
		clitest.RootCommand().Command("accepted").Exists(),
	)

	panicked := false
	func() {
		defer func() { panicked = recover() != nil }()
		retained.Command(cli.Command[config]{
			Name: "late",
		})
	}()
	if !panicked {
		t.Fatal("provider batch remained usable after callback")
	}
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"late"},
	}).Require(
		t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticUnknownCommand).Present()),
		clitest.RootCommand().Command("late").ObservedDoesNotExist(),
	)
}

func TestErasedProviderEvaluationCapturesCanonicalBatchProgram(t *testing.T) {
	t.Parallel()
	type config = providerBatchConfig
	app := providerMustCompilePublic(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				child := batch.Command(cli.Command[struct{}]{
					Name: "child",
				})
				batch.Alias(child, cli.CommandAlias{
					Name: "c",
				})
				return nil
			})
		},
	})
	root := clitest.RootCommand()
	child := root.Command("child")
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"c"},
	}).Require(
		t,
		clitest.InspectionReady(),
		child.Selected(),
		root.Alias("c").Selected(),
	)
}

type providerDemandValueConfig struct{ Value string }

func TestDynamicDomainDemandLeavesFailureAndCancellationClassificationIntact(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("domain failure")
	type config = providerDemandValueConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(context.Context, *config, cli.Dependencies, *cli.ValueDomainBuilder[string]) error {
					return sentinel
				},
			)
			value := root.Values.Value(&root.Config.Value, domain)
			root.Flags.ValueFor(value).Long("value")
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	result := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--value=x"},
	})
	_, gotErr := result.Inspection()
	if !errors.Is(gotErr, sentinel) {
		t.Fatalf("Inspect() error = %v, want dynamic-domain failure", gotErr)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, gotErr = app.Inspect(ctx, cli.InspectionRequest{
		Args: []string{"--value=x"},
	})
	if !errors.Is(gotErr, context.Canceled) {
		t.Fatalf("canceled Inspect() error = %v, want context.Canceled", gotErr)
	}
}

type providerLocalizationConfig struct{}

func TestProviderIndependentLocalizationOwnsMaterializedPresentation(t *testing.T) {
	t.Parallel()
	type config = providerLocalizationConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"remote command": "application translation must not win",
				},
			}},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Localization(cli.Localization{
					DefaultLocale: "en",
					Bundles: []cli.TranslationBundle{{
						Locale: "fr",
						Translations: cli.Translations{
							"remote command": "commande distante",
						},
					}},
				})
				batch.Command(cli.Command[config]{
					Name: "remote",
					Description: cli.Description{
						Short: "remote command",
					},
				})
				return nil
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	completionRequest := cli.CompletionRequest{
		Args: []string{""},
		Cursor: cli.CompletionCursor{
			Argument: 0,
		},
		Locale: "fr",
		Client: cli.CompletionClient{
			Capabilities: cli.CompletionClientCapabilities{
				Descriptions: true,
			},
		},
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t, clitest.CompletionItem("remote").Presentation("remote", "commande distante"))

	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args:   []string{"remote"},
		Locale: "fr",
	}).Require(t,
		clitest.InspectionReady(),
		clitest.Command("remote").PresentedDescription(cli.Description{
			Short: "commande distante",
		}),
		clitest.RootCommand().ProviderAt(0).LocalizationExactly(cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"remote command": "commande distante",
				},
			}},
		}),
	)
}

func TestProviderIndependentLocalizationOwnsContributedAliasPresentation(t *testing.T) {
	t.Parallel()
	type config = providerLocalizationConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"remote alias": "application alias translation must not win",
				},
			}},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.Command(cli.Command[config]{
				Name: "serve",
			})
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Localization(cli.Localization{
					DefaultLocale: "en",
					Bundles: []cli.TranslationBundle{{
						Locale: "fr",
						Translations: cli.Translations{
							"remote alias": "alias distant",
						},
					}},
				})
				batch.Alias(cli.CommandPath{"serve"}, cli.CommandAlias{
					Name: "remote",
					Description: cli.Description{
						Short: "remote alias",
					},
				})
				return nil
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	completionRequest := cli.CompletionRequest{
		Args: []string{""},
		Cursor: cli.CompletionCursor{
			Argument: 0,
		},
		Locale: "fr",
		Client: cli.CompletionClient{
			Capabilities: cli.CompletionClientCapabilities{
				Descriptions: true,
			},
		},
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t, clitest.CompletionItem("remote").Presentation("remote", "alias distant"))
}

func TestProviderAliasWithoutLocalDescriptionInheritsTargetDescriptionOwner(t *testing.T) {
	t.Parallel()
	type config = providerLocalizationConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"serve command": "commande servir",
				},
			}},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.Command(cli.Command[config]{
				Name: "serve",
				Description: cli.Description{
					Short: "serve command",
				},
			})
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Localization(cli.Localization{
					DefaultLocale: "en",
					Bundles: []cli.TranslationBundle{{
						Locale: "fr",
						Translations: cli.Translations{
							"serve command": "provider translation must not win",
						},
					}},
				})
				batch.Alias(cli.CommandPath{"serve"}, cli.CommandAlias{
					Name: "remote",
				})
				return nil
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	completionRequest := cli.CompletionRequest{
		Args: []string{""},
		Cursor: cli.CompletionCursor{
			Argument: 0,
		},
		Locale: "fr",
		Client: cli.CompletionClient{
			Capabilities: cli.CompletionClientCapabilities{
				Descriptions: true,
			},
		},
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t, clitest.CompletionItem("remote").Presentation("remote", "commande servir"))
}

func TestProviderPublicationMetadataRejectsInvalidLocalizationWithoutOutcomeIdentity(t *testing.T) {
	t.Parallel()
	type config = providerLocalizationConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Localization(cli.Localization{
					Bundles: []cli.TranslationBundle{{
						Locale: "fr",
						Translations: cli.Translations{
							"candidate": "candidat",
						},
					}},
				})
				batch.Command(cli.Command[config]{
					Name: "candidate",
				})
				return nil
			})
		},
	})
	root := clitest.RootCommand()
	provider := root.ProviderAt(0)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidCommandBatch).Present()),
		provider.EntriesExactly(),
		provider.NoLocalization(),
		clitest.Command("candidate").ObservedDoesNotExist(),
		clitest.InspectionObservedCommandProvidersExactly(provider),
	)
}

const providerMaterializationToolName = "tool"

type (
	providerMaterializationAccessKind     int
	providerMaterializationRelation       int
	providerMaterializationRelationFamily int
)

type providerMaterializationRootConfig struct{}

type providerMaterializationTargetConfig struct{}

type providerMaterializationProviderConfig struct{}

type providerMaterializationBulkConfig struct{}

type providerMaterializationValueDecoyConfig struct{ Value string }

type providerMaterializationValueCopyConfig struct {
	Value string
	Copy  string
}

type providerMaterializationProfileRootConfig struct{ Profile string }

type providerMaterializationProfilePairModeChildConfig struct {
	Profile string
	Pair    providerTokenPairValue
	Mode    string
}

type providerMaterializationLocaleRootConfig struct{ Locale string }

type providerMaterializationLocaleChildConfig struct{ Locale string }

type providerMaterializationSourceValueOptionalStateConfig struct {
	Source   string
	Value    string
	Optional cli.OptionalValue[string]
	State    cli.OptionalValue[cli.ValueState]
}

type providerMaterializationSourceConfig struct{ Source string }

type providerMaterializationSeedLocalConfig struct {
	Seed  string
	Local string
}

type providerMaterializationSeedPathConfig struct {
	Seed string
	Path string
}

type providerMaterializationConfig struct{}

type providerMaterializationSourceValueConfig struct {
	Source string
	Value  string
}

func TestProviderSettlementConvergesConcurrentDemandWhileSchedulingDoesNotDefineOrder(t *testing.T) {
	t.Parallel()
	type config = providerMaterializationConfig
	var calls [3]atomic.Int32
	definition := cli.Definition[config]{
		Name: providerMaterializationToolName,
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				calls[0].Add(1)
				batch.Command(cli.Command[config]{
					Name: "first",
				})
				return nil
			})
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				calls[1].Add(1)
				batch.Command(cli.Command[config]{
					Name: "second",
				})
				batch.Alias(cli.CommandPath{"first"}, cli.CommandAlias{
					Name: "first-alias",
				})
				return nil
			})
			root.Subcommands.CommandsFrom(func(context.Context, *cli.CommandBatch[config]) error {
				calls[2].Add(1)
				return nil
			})
		},
	}
	app := providerMustCompilePublic(t, definition)

	root := clitest.RootCommand()
	first := root.Command("first")
	second := root.Command("second")
	alias := root.Alias("first-alias")
	provider0 := root.ProviderAt(0)
	provider1 := root.ProviderAt(1)
	provider2 := root.ProviderAt(2)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		clitest.InspectionReady(),
		clitest.InspectionObservedCommandProvidersExactly(provider0, provider1, provider2),
		provider0.Outcome(cli.CommandProviderOutcomeAccepted),
		provider1.Outcome(cli.CommandProviderOutcomeAccepted),
		provider2.Outcome(cli.CommandProviderOutcomeAccepted),
		root.EntriesExactly(first.CanonicalEntry(), second.CanonicalEntry(), alias),
		alias.Kind(cli.CommandEntryAlias),
		alias.Targets(first),
	)
	if calls[0].Load() != 1 || calls[1].Load() != 1 || calls[2].Load() != 1 {
		t.Fatalf("provider calls = (%d,%d,%d), want (1,1,1)", calls[0].Load(), calls[1].Load(), calls[2].Load())
	}
}

func TestProviderPublicationScopeClosesSiblingTargetsWithoutOwningSiblingSubtree(t *testing.T) {
	t.Parallel()
	type rootConfig = providerMaterializationRootConfig
	type targetConfig = providerMaterializationTargetConfig
	type providerConfig = providerMaterializationProviderConfig
	type bulkConfig = providerMaterializationBulkConfig
	type decoyConfig = providerMaterializationValueDecoyConfig
	providerCalls := 0
	app := providerMustCompilePublic(t, cli.Definition[rootConfig]{
		Name: providerMaterializationToolName,
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[targetConfig]{
				Name: "target",
				Declare: func(target *cli.CommandBuilder[targetConfig]) {
					target.Subcommands.CommandsFrom(
						func(_ context.Context, batch *cli.CommandBatch[targetConfig]) error {
							providerCalls++
							batch.Command(cli.Command[providerConfig]{
								Name: "provider-delta",
							})
							return nil
						},
					)
				},
			})
			root.Subcommands.Command(cli.Command[bulkConfig]{
				Name: "bulk",
				Declare: func(bulk *cli.CommandBuilder[bulkConfig]) {
					bulk.Subcommands.Command(cli.Command[decoyConfig]{
						Name: "decoy",
						Declare: func(decoy *cli.CommandBuilder[decoyConfig]) {
							value := decoy.Values.String(&decoy.Config.Value)
							decoy.Flags.ValueFor(value).Long("value")
						},
					})
				},
			})
		},
	})

	root := clitest.RootCommand()
	target := root.Command("target")
	provider := target.ProviderAt(0)
	providerDelta := target.Command("provider-delta")
	decoy := root.Command("bulk").Command("decoy")
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"target"},
	}).Require(t,
		clitest.InspectionReady(),
		provider.Outcome(cli.CommandProviderOutcomeAccepted),
		provider.EntriesExactly(providerDelta.CanonicalEntry()),
		decoy.Flag("--value").Exists(),
	)
	if providerCalls != 1 {
		t.Fatalf("target provider calls = %d, want 1", providerCalls)
	}
}

func TestProviderTargetsSettleAgainstCompleteCanonicalProviderNamespace(t *testing.T) {
	t.Parallel()
	type config = providerMaterializationConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: providerMaterializationToolName,
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Command(cli.Command[config]{
					Name: "canonical",
				})
				return nil
			})
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Alias(cli.CommandPath{"canonical"}, cli.CommandAlias{
					Name: "alias",
				})
				return nil
			})
		},
	})
	root := clitest.RootCommand()
	canonical := root.Command("canonical")
	alias := root.Alias("alias")
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"alias"},
	}).Require(t,
		clitest.InspectionReady(),
		alias.Targets(canonical),
		canonical.Selected(),
	)
}

func TestProviderSettlementRejectsSymmetricCollisionAndUnresolvedTargetWithoutOrderWinner(t *testing.T) {
	t.Parallel()
	type config = providerMaterializationConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: providerMaterializationToolName,
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.CommandNormalization(cli.NameNormalizationASCIICaseFold)
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Command(cli.Command[config]{
					Name: "same",
				})
				return nil
			})
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Command(cli.Command[config]{
					Name: "SAME",
				})
				return nil
			})
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Alias(cli.CommandPath{"missing"}, cli.CommandAlias{
					Name: "alias",
				})
				return nil
			})
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Command(cli.Command[config]{
					Name: "survivor",
				})
				return nil
			})
		},
	})
	root := clitest.RootCommand()
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"survivor"},
	}).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.DiagnosticAt(0).Code(cli.DiagnosticInvalidCommandBatch)),
		root.Command("same").ObservedDoesNotExist(),
		root.Command("SAME").ObservedDoesNotExist(),
		root.Command("survivor").Exists(),
		root.NamespaceCompleteness(cli.CommandNamespaceIncomplete),
	)
}

func TestProviderCanceledOrFailedDemandSettlesOnceWithoutHiddenRetry(t *testing.T) {
	t.Parallel()
	type config = providerMaterializationConfig
	var calls atomic.Int32
	failure := errors.New("provider failed")
	app := providerMustCompilePublic(t, cli.Definition[config]{
		Name: providerMaterializationToolName,
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				calls.Add(1)
				batch.Command(cli.Command[config]{
					Name: "discarded",
				})
				return failure
			})
		},
	})
	root := clitest.RootCommand()
	provider := root.ProviderAt(0)
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"discarded"},
	})
	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticCommandProviderFailure))),
		provider.Outcome(cli.CommandProviderOutcomeFailed),
		provider.EntriesExactly(),
		root.Command("discarded").ObservedDoesNotExist(),
	)
	_, err := proof.Inspection()
	if !errors.Is(err, failure) {
		t.Fatalf("Inspect() error = %v, want provider failure", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("failed demand provider calls = %d, want 1", calls.Load())
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	before := calls.Load()
	canceled := clitest.Inspect(ctx, app, cli.InspectionRequest{
		Args: []string{"discarded"},
	})
	canceled.Require(t,
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticCommandProviderCancellation).Present()),
		root.Command("discarded").ObservedDoesNotExist(),
	)
	_, cancelErr := canceled.Inspection()
	if !errors.Is(cancelErr, context.Canceled) {
		t.Fatalf("pre-canceled Inspect() error = %v", cancelErr)
	}
	if calls.Load() != before {
		t.Fatalf("pre-canceled demand executed provider: before=%d after=%d", before, calls.Load())
	}
}

func TestProviderIndependentFailureRacingWithCancellationRemainsProviderFailure(t *testing.T) {
	t.Parallel()
	type config = providerMaterializationConfig
	failure := errors.New("independent provider failure")
	callerCancellation := errors.New("caller cancellation")
	started := make(chan struct{})
	release := make(chan struct{})
	app := providerMustCompilePublic(t, cli.Definition[config]{
		Name: providerMaterializationToolName,
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(context.Context, *cli.CommandBatch[config]) error {
				close(started)
				<-release
				return failure
			})
		},
	})
	provider := clitest.RootCommand().ProviderAt(0)
	ctx, cancel := context.WithCancelCause(t.Context())
	result := make(chan clitest.InspectionResult, 1)
	go func() { result <- clitest.Inspect(ctx, app, cli.InspectionRequest{}) }()
	<-started
	cancel(callerCancellation)
	<-ctx.Done()
	close(release)
	proof := <-result
	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticCommandProviderFailure))),
		provider.Outcome(cli.CommandProviderOutcomeFailed),
		provider.EntriesExactly(),
	)
	_, err := proof.Inspection()
	if !errors.Is(err, failure) {
		t.Fatalf("Inspect() error = %v, want independent provider failure", err)
	}
	if !errors.Is(err, callerCancellation) {
		t.Fatalf("Inspect() error = %v, want independent caller cancellation", err)
	}
}

func TestProviderCallbackPanicPropagatesBeforeAnyConcurrentPeerPublication(t *testing.T) {
	t.Parallel()
	type config = providerMaterializationConfig
	panicValue := errors.New("provider panic")
	definition := cli.Definition[config]{
		Name: providerMaterializationToolName,
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(context.Context, *cli.CommandBatch[config]) error { panic(panicValue) })
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Command(cli.Command[config]{
					Name: "peer",
				})
				return nil
			})
		},
	}
	app := providerMustCompilePublic(t, definition)
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		_, _ = clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Inspection()
	}()
	if recovered != panicValue { //nolint:errorlint // The public callback panic value itself is the contract under test.
		t.Fatalf("recovered panic = %#v, want exact %#v", recovered, panicValue)
	}
	clitest.Schema(app).Require(t, clitest.CommandsExactly(clitest.RootCommand()))
}

func TestDemandedProviderFailurePreservesIndependentAcceptedPeerAndNoFailedOutput(t *testing.T) {
	t.Parallel()
	type config = providerMaterializationConfig
	providerFailure := errors.New("provider failure")
	failedCalls := 0
	acceptedCalls := 0
	app := clitest.Compile(t, cli.Definition[config]{
		Name: providerMaterializationToolName,
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				acceptedCalls++
				batch.Command(cli.Command[config]{
					Name: "survivor",
				})
				return nil
			})
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				failedCalls++
				batch.Command(cli.Command[config]{
					Name: "discarded",
				})
				return providerFailure
			})
		},
	})
	root := clitest.RootCommand()
	acceptedProvider := root.ProviderAt(0)
	failedProvider := root.ProviderAt(1)
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"survivor"},
	})
	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticCommandProviderFailure))),
		root.NamespaceCompleteness(cli.CommandNamespaceIncomplete),
		root.Command("survivor").Exists(),
		root.Command("discarded").ObservedDoesNotExist(),
		clitest.InspectionObservedCommandProvidersExactly(acceptedProvider, failedProvider),
		acceptedProvider.Outcome(cli.CommandProviderOutcomeAccepted),
		failedProvider.Outcome(cli.CommandProviderOutcomeFailed),
		failedProvider.EntriesExactly(),
	)
	_, err := proof.Inspection()
	if !errors.Is(err, providerFailure) {
		t.Fatalf("Inspect() error = %v, want provider failure", err)
	}
	if acceptedCalls != 1 || failedCalls != 1 {
		t.Fatalf("provider calls = accepted %d failed %d", acceptedCalls, failedCalls)
	}
}

func TestInvalidProviderPolicyAndAccessRemainRejectedAcrossOperations(t *testing.T) {
	t.Parallel()
	type config = providerMaterializationValueCopyConfig
	tests := []struct {
		name    string
		declare func(*cli.CommandBuilder[config])
	}{
		{
			name: "malformed policy",
			declare: func(child *cli.CommandBuilder[config]) {
				value := child.Values.String(&child.Config.Value)
				child.Rule.Between(value, "z", "a")
			},
		},
		{
			name: "local required Access",
			declare: func(child *cli.CommandBuilder[config]) {
				value := child.Values.String(&child.Config.Value)
				child.Access.Value(value, &child.Config.Copy)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			providerCalls := 0
			app := clitest.Compile(t, cli.Definition[config]{
				Name: providerMaterializationToolName,
				Declare: func(root *cli.CommandBuilder[config]) {
					root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
						providerCalls++
						batch.Command(cli.Command[config]{
							Name:    "remote",
							Declare: test.declare,
						})
						return nil
					})
				},
			})
			root := clitest.RootCommand()
			provider := root.ProviderAt(0)
			for run := 1; run <= 2; run++ {
				clitest.Inspect(t.Context(), app, cli.InspectionRequest{
					Args: []string{"remote"},
				}).Require(t,
					clitest.InspectionRejected(),
					clitest.FailsWith(clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticInvalidCommandBatch))),
					root.Command("remote").ObservedDoesNotExist(),
					provider.Outcome(cli.CommandProviderOutcomeRejected),
					provider.EntriesExactly(),
				)
				if providerCalls != run {
					t.Fatalf("provider calls after run %d = %d, want %d", run, providerCalls, run)
				}
			}
		})
	}
}

func TestProviderCancellationAtPublicationBoundariesHasDeterministicVisibility(t *testing.T) {
	t.Parallel()
	type config = providerMaterializationConfig

	t.Run("during callback", func(t *testing.T) {
		t.Parallel()
		started := make(chan struct{})
		app := providerMustCompilePublic(t, cli.Definition[config]{
			Name: providerMaterializationToolName,
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Subcommands.CommandsFrom(func(ctx context.Context, batch *cli.CommandBatch[config]) error {
					close(started)
					<-ctx.Done()
					batch.Command(cli.Command[config]{
						Name: "late",
					})
					return ctx.Err()
				})
			},
		})
		root := clitest.RootCommand()
		provider := root.ProviderAt(0)
		ctx, cancel := context.WithCancel(t.Context())
		result := make(chan clitest.InspectionResult, 1)
		go func() { result <- clitest.Inspect(ctx, app, cli.InspectionRequest{}) }()
		<-started
		cancel()
		proof := <-result
		proof.Require(t,
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticCommandProviderCancellation).Present()),
			root.Command("late").ObservedDoesNotExist(),
			provider.Outcome(cli.CommandProviderOutcomeCanceled),
			provider.EntriesExactly(),
		)
		_, err := proof.Inspection()
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("callback cancellation error = %v", err)
		}
	})

	t.Run("contribution complete before cancellation", func(t *testing.T) {
		t.Parallel()
		contributionReady := make(chan struct{})
		app := providerMustCompilePublic(t, cli.Definition[config]{
			Name: providerMaterializationToolName,
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Subcommands.CommandsFrom(func(ctx context.Context, batch *cli.CommandBatch[config]) error {
					batch.Command(cli.Command[config]{
						Name: "candidate",
					})
					close(contributionReady)
					<-ctx.Done()
					return ctx.Err()
				})
			},
		})
		root := clitest.RootCommand()
		provider := root.ProviderAt(0)
		ctx, cancel := context.WithCancel(t.Context())
		result := make(chan clitest.InspectionResult, 1)
		go func() { result <- clitest.Inspect(ctx, app, cli.InspectionRequest{}) }()
		<-contributionReady
		cancel()
		proof := <-result
		proof.Require(t,
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticCommandProviderCancellation).Present()),
			root.Command("candidate").ObservedDoesNotExist(),
			provider.Outcome(cli.CommandProviderOutcomeCanceled),
			provider.EntriesExactly(),
		)
		_, err := proof.Inspection()
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("prepublication cancellation error = %v", err)
		}
	})

	t.Run("after successful publication", func(t *testing.T) {
		t.Parallel()
		app := providerMustCompilePublic(t, cli.Definition[config]{
			Name: providerMaterializationToolName,
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
					batch.Command(cli.Command[config]{
						Name: "accepted",
					})
					return nil
				})
			},
		})
		root := clitest.RootCommand()
		accepted := root.Command("accepted")
		provider := root.ProviderAt(0)
		ctx, cancel := context.WithCancel(t.Context())
		proof := clitest.Inspect(ctx, app, cli.InspectionRequest{})
		cancel()
		proof.Require(t,
			clitest.InspectionReady(),
			clitest.InspectionObservedCommandProvidersExactly(provider),
			provider.Outcome(cli.CommandProviderOutcomeAccepted),
			provider.EntriesExactly(accepted.CanonicalEntry()),
			accepted.Exists(),
		)
	})
}

func TestProviderPublicationChangedUsesCurrentAcceptedEntryOwnership(t *testing.T) {
	t.Parallel()
	type config = providerMaterializationConfig
	for _, tc := range []struct {
		name       string
		contribute bool
	}{
		{
			name:       "accepted empty provider",
			contribute: false,
		},
		{
			name:       "accepted provider entry",
			contribute: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			app := providerMustCompilePublic(t, cli.Definition[config]{
				Name: providerMaterializationToolName,
				Declare: func(root *cli.CommandBuilder[config]) {
					root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
						if tc.contribute {
							batch.Command(cli.Command[config]{
								Name: "dynamic",
							})
						}
						return nil
					})
				},
			})
			root := clitest.RootCommand()
			provider := root.ProviderAt(0)
			expectations := []clitest.InspectionExpectation{
				clitest.InspectionReady(),
				clitest.InspectionObservedCommandProvidersExactly(provider),
				provider.Outcome(cli.CommandProviderOutcomeAccepted),
			}
			if tc.contribute {
				expectations = append(expectations, provider.EntriesExactly(root.Command("dynamic").CanonicalEntry()))
			} else {
				expectations = append(expectations, provider.EntriesExactly())
			}
			clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t, expectations...)
		})
	}
}

func TestOneOperationModelViewKeepsProviderGrammarDomainPolicyLocalizationAndAccessConsistent(t *testing.T) {
	t.Parallel()
	type rootConfig = providerMaterializationProfileRootConfig
	type childConfig = providerMaterializationProfilePairModeChildConfig

	var providerCalls atomic.Int32
	var parserCalls atomic.Int32
	var domainCalls atomic.Int32
	definition := cli.Definition[rootConfig]{
		Name: providerMaterializationToolName,
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			profile := root.Values.String(&root.Config.Profile)
			root.Values.Default(profile, "profiled")
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[rootConfig]) error {
				providerCalls.Add(1)
				batch.Command(cli.Command[childConfig]{
					Name: "remote",
					Description: cli.Description{
						Short: "remote short",
					},
					Declare: func(child *cli.CommandBuilder[childConfig]) {
						child.Access.Value(profile, &child.Config.Profile)

						pair := child.Values.Value(&child.Config.Pair, providerTokenPairCodec())
						child.Arguments.Form(func(form *cli.ArgumentFormBuilder[childConfig]) {
							form.Tokens(pair, "pair", cli.TokenLanguage[providerTokenPairValue]{
								Name:  "provider-pair",
								Count: cli.Count.Exactly(2),
								Parse: func(_ context.Context, tokens []string) (providerTokenPairValue, error) {
									parserCalls.Add(1)
									value := providerTokenPairValue{
										Left:  tokens[0],
										Right: tokens[1],
									}
									return value, nil
								},
								Format: func(value providerTokenPairValue) []string {
									return []string{"provider-" + value.Left, "provider-" + value.Right}
								},
							})
						})

						domain := child.Values.Domain(
							func(_ context.Context, _ *childConfig, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
								domainCalls.Add(1)
								canonical := builder.Value("prod", "production")
								builder.Alias("p", canonical)
								return nil
							},
						)
						mode := child.Values.Value(&child.Config.Mode, domain)
						child.Flags.ValueFor(mode).Long("mode")
						child.Rule.Require(mode)
						child.Advisory.Always(cli.Advisory{
							Code:     "mode-ready",
							Severity: cli.NoticeSeverityWarning,
							Message:  "mode ready",
						}).Subject(mode)
					},
					Run: func(context.Context, *childConfig) error { return nil },
				})
				batch.Localization(cli.Localization{
					DefaultLocale: "en",
					Bundles: []cli.TranslationBundle{{
						Locale: "fr",
						Translations: cli.Translations{
							"remote short": "distant court",
							"mode ready":   "mode pret",
						},
					}},
				})
				return nil
			})
		},
	}
	app := clitest.Compile(t, definition)

	root := clitest.RootCommand()
	remote := root.Command("remote")
	provider := root.ProviderAt(0)
	profile := root.ValueAt(0)
	pair := remote.Argument("pair").Value()
	mode := remote.Flag("--mode").Value()
	domain := mode.Domain(cli.ValueDomainUseDestination)
	canonical := domain.Entry("prod")
	alias := domain.Entry("p")
	notice := clitest.Notice(cli.NoticeAdvisory).Advisory("mode-ready")
	expectedAccess := clitest.ExecutionAccess{
		Values: []clitest.ValueLocator{profile},
	}

	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args:   []string{"remote", "--mode", "p", "left", "right"},
		Locale: "fr",
	}).Require(t,
		clitest.InspectionReady(),
		clitest.DiagnosticsExactly(),
		clitest.EffectiveLocale("fr"),
		remote.Selected(),
		remote.PresentedDescription(cli.Description{
			Short: "distant court",
		}),
		remote.ExecutionAccessExactly(expectedAccess),
		profile.Resolved(),
		profile.Text("profiled"),
		pair.Resolved(),
		pair.CandidateAt(0).Tokens("provider-left", "provider-right"),
		mode.Resolved(),
		mode.Text("prod"),
		clitest.InspectionObservedCommandProvidersExactly(provider),
		provider.Outcome(cli.CommandProviderOutcomeAccepted),
		provider.EntriesExactly(remote.CanonicalEntry()),
		clitest.InspectionObservedValueDomainsExactly(domain),
		domain.Outcome(cli.ValueDomainOutcomeReady),
		domain.EntriesExactly(canonical, alias),
		alias.Canonical(canonical),
		remote.RuleAt(0).Outcome(cli.EvaluationOutcomeSatisfied),
		remote.Advisory("mode-ready").Outcome(cli.AdvisoryOutcomeMatched),
		clitest.NoticesExactly(notice),
		notice.Suppression(cli.NoticeSuppressionNotSuppressed),
	)

	if providerCalls.Load() != 1 || parserCalls.Load() != 1 || domainCalls.Load() != 1 {
		t.Fatalf(
			"one-operation callbacks = provider %d parser %d domain %d, want 1/1/1",
			providerCalls.Load(),
			parserCalls.Load(),
			domainCalls.Load(),
		)
	}
}

func TestRejectedStagedProviderCandidatesLeaveNoPublicationLocalizationOrIdentityResidue(t *testing.T) {
	t.Parallel()
	type config = providerMaterializationConfig
	for _, test := range []struct {
		name    string
		declare func(*cli.CommandBuilder[config])
		staged  []string
	}{
		{
			name:   "mixed valid command and unresolved relationship",
			staged: []string{"candidate", "broken"},
			declare: func(root *cli.CommandBuilder[config]) {
				root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
					batch.Localization(cli.Localization{
						DefaultLocale: "en",
					})
					batch.Command(cli.Command[config]{
						Name: "candidate",
					})
					batch.Alias(cli.CommandPath{"missing"}, cli.CommandAlias{
						Name: "broken",
					})
					return nil
				})
			},
		},
		{
			name:   "canonical namespace collision",
			staged: []string{"same", "SAME"},
			declare: func(root *cli.CommandBuilder[config]) {
				root.Dialect.CommandNormalization(cli.NameNormalizationASCIICaseFold)
				root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
					batch.Localization(cli.Localization{
						DefaultLocale: "en",
					})
					batch.Command(cli.Command[config]{
						Name: "same",
					})
					batch.Command(cli.Command[config]{
						Name: "SAME",
					})
					return nil
				})
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			app := providerMustCompilePublic(t, cli.Definition[config]{
				Name:    providerMaterializationToolName,
				Declare: test.declare,
			})
			root := clitest.RootCommand()
			provider := root.ProviderAt(0)
			expectations := make([]clitest.InspectionExpectation, 0, 5+len(test.staged))
			expectations = append(expectations,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticInvalidCommandBatch))),
				clitest.InspectionObservedCommandProvidersExactly(provider),
				provider.Outcome(cli.CommandProviderOutcomeRejected),
				provider.EntriesExactly(),
			)
			for _, name := range test.staged {
				expectations = append(expectations, root.Command(name).ObservedDoesNotExist())
			}
			for run := 1; run <= 2; run++ {
				proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{})
				proof.Require(t, expectations...)
				if _, err := proof.Inspection(); err == nil {
					t.Fatalf("Inspect() run %d accepted rejected staged provider candidate", run)
				}
			}
		})
	}
}

func TestProviderPanicRemainsOrdinaryPanic(t *testing.T) {
	t.Parallel()
	panicValue := &struct{ marker string }{
		marker: "provider panic",
	}
	app, err := cli.Compile(cli.Definition[struct{}]{
		Name: providerMaterializationToolName,
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.CommandsFrom(func(context.Context, *cli.CommandBatch[struct{}]) error {
				panic(panicValue)
			})
		},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	defer func() {
		if got := recover(); got != panicValue {
			t.Fatalf("panic = %#v, want original %#v", got, panicValue)
		}
	}()
	_, _ = app.Inspect(t.Context(), cli.InspectionRequest{})
}

func TestProviderProducedLocaleSelectorIsRejectedDuringInvocationAnalysis(t *testing.T) {
	t.Parallel()
	type rootConfig = providerMaterializationLocaleRootConfig
	type childConfig = providerMaterializationLocaleChildConfig
	providerCalls := 0
	definition := cli.Definition[rootConfig]{
		Name: providerMaterializationToolName,
		Localization: cli.Localization{
			DefaultLocale: "en",
		},
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Flags.String(&root.Config.Locale).LocaleSelector().Occurrences(cli.Count.Optional()).
				Long("locale")
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[rootConfig]) error {
				providerCalls++
				batch.Command(cli.Command[childConfig]{
					Name: "remote",
					Declare: func(child *cli.CommandBuilder[childConfig]) {
						child.Flags.String(&child.Config.Locale).LocaleSelector().Occurrences(cli.Count.Optional()).
							Long("locale")
					},
				})
				return nil
			})
		},
	}
	app := clitest.Compile(t, definition)
	if providerCalls != 0 {
		t.Fatalf("Compile() executed provider %d times", providerCalls)
	}

	root := clitest.RootCommand()
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--locale", "fr", "remote"},
	}).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticInvalidCommandBatch))),
		root.Flag("--locale").LocaleSelector(),
	)
	if providerCalls != 1 {
		t.Fatalf("provider calls = %d, want 1", providerCalls)
	}
}

func TestProviderAccessSelectedAncestorPolarity(t *testing.T) {
	t.Parallel()

	type config = providerMaterializationSourceValueOptionalStateConfig
	type accessKind = providerMaterializationAccessKind
	const (
		accessValue accessKind = iota
		accessOptionalValue
		accessValueState
	)
	type relation = providerMaterializationRelation
	const (
		relationSame relation = iota
		relationAncestor
		relationSibling
		relationDescendant
		relationAcceptedExternal
	)

	apply := func(command *cli.CommandBuilder[config], source cli.Destination[config, string], kind accessKind) {
		switch kind {
		case accessValue:
			command.Access.Value(source, &command.Config.Value)
		case accessOptionalValue:
			command.Access.OptionalValue(source, &command.Config.Optional)
		case accessValueState:
			command.Access.ValueState(source, &command.Config.State)
		}
	}
	definition := func(kind accessKind, relationship relation) cli.Definition[config] {
		definition := cli.Definition[config]{
			Name: providerMaterializationToolName,
			Declare: func(root *cli.CommandBuilder[config]) {
				var accepted cli.Destination[config, string]
				if relationship == relationAcceptedExternal {
					accepted = root.Values.String(&root.Config.Source)
				}
				root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
					switch relationship {
					case relationSame:
						batch.Command(cli.Command[config]{
							Name: "same",
							Declare: func(command *cli.CommandBuilder[config]) {
								source := command.Values.String(&command.Config.Source)
								apply(command, source, kind)
							},
						})
					case relationAncestor:
						batch.Command(cli.Command[config]{
							Name: "parent",
							Declare: func(parent *cli.CommandBuilder[config]) {
								source := parent.Values.String(&parent.Config.Source)
								parent.Subcommands.Command(cli.Command[config]{
									Name: "child",
									Declare: func(child *cli.CommandBuilder[config]) {
										apply(child, source, kind)
									},
								})
							},
						})
					case relationSibling:
						var source cli.Destination[config, string]
						batch.Command(cli.Command[config]{
							Name: "source",
							Declare: func(command *cli.CommandBuilder[config]) {
								source = command.Values.String(&command.Config.Source)
							},
						})
						batch.Command(cli.Command[config]{
							Name: "target",
							Declare: func(command *cli.CommandBuilder[config]) {
								apply(command, source, kind)
							},
						})
					case relationDescendant:
						batch.Command(cli.Command[config]{
							Name: "parent",
							Declare: func(parent *cli.CommandBuilder[config]) {
								var source cli.Destination[config, string]
								parent.Subcommands.Command(cli.Command[config]{
									Name: "source",
									Declare: func(command *cli.CommandBuilder[config]) {
										source = command.Values.String(&command.Config.Source)
									},
								})
								apply(parent, source, kind)
							},
						})
					case relationAcceptedExternal:
						batch.Command(cli.Command[config]{
							Name: "target",
							Declare: func(command *cli.CommandBuilder[config]) {
								apply(command, accepted, kind)
							},
						})
					}
					return nil
				})
			},
		}
		return definition
	}

	tests := []struct {
		name         string
		kind         accessKind
		relationship relation
		valid        bool
	}{
		{
			name:         "value same command",
			kind:         accessValue,
			relationship: relationSame,
			valid:        false,
		},
		{
			name:         "value ancestor",
			kind:         accessValue,
			relationship: relationAncestor,
			valid:        true,
		},
		{
			name:         "value sibling",
			kind:         accessValue,
			relationship: relationSibling,
			valid:        false,
		},
		{
			name:         "value descendant",
			kind:         accessValue,
			relationship: relationDescendant,
			valid:        false,
		},
		{
			name:         "value accepted external ancestor",
			kind:         accessValue,
			relationship: relationAcceptedExternal,
			valid:        true,
		},
		{
			name:         "optional value same command",
			kind:         accessOptionalValue,
			relationship: relationSame,
			valid:        false,
		},
		{
			name:         "optional value ancestor",
			kind:         accessOptionalValue,
			relationship: relationAncestor,
			valid:        true,
		},
		{
			name:         "optional value sibling",
			kind:         accessOptionalValue,
			relationship: relationSibling,
			valid:        false,
		},
		{
			name:         "optional value descendant",
			kind:         accessOptionalValue,
			relationship: relationDescendant,
			valid:        false,
		},
		{
			name:         "optional value accepted external ancestor",
			kind:         accessOptionalValue,
			relationship: relationAcceptedExternal,
			valid:        true,
		},
		{
			name:         "value state same command",
			kind:         accessValueState,
			relationship: relationSame,
			valid:        true,
		},
		{
			name:         "value state ancestor",
			kind:         accessValueState,
			relationship: relationAncestor,
			valid:        true,
		},
		{
			name:         "value state sibling",
			kind:         accessValueState,
			relationship: relationSibling,
			valid:        false,
		},
		{
			name:         "value state descendant",
			kind:         accessValueState,
			relationship: relationDescendant,
			valid:        false,
		},
		{
			name:         "value state accepted external ancestor",
			kind:         accessValueState,
			relationship: relationAcceptedExternal,
			valid:        true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			app := clitest.Compile(t, definition(test.kind, test.relationship))
			proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{})
			if test.valid {
				proof.Require(t,
					clitest.InspectionReady(),
					clitest.RootCommand().ProviderAt(0).Outcome(cli.CommandProviderOutcomeAccepted),
				)
				return
			}
			proof.Require(t,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidCommandBatch).Present()),
				clitest.RootCommand().ProviderAt(0).Outcome(cli.CommandProviderOutcomeRejected),
			)
		})
	}
}

func TestProviderAccessRejectsForeignAcceptedDestination(t *testing.T) {
	t.Parallel()

	type config = providerMaterializationSourceValueConfig
	var foreign cli.Destination[config, string]
	clitest.Compile(t, cli.Definition[config]{
		Name: "foreign",
		Declare: func(root *cli.CommandBuilder[config]) {
			foreign = root.Values.String(&root.Config.Source)
		},
	})

	app := clitest.Compile(t, cli.Definition[config]{
		Name: providerMaterializationToolName,
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Command(cli.Command[config]{
					Name: "target",
					Declare: func(command *cli.CommandBuilder[config]) {
						command.Access.Value(foreign, &command.Config.Value)
					},
				})
				return nil
			})
		},
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidCommandBatch).Present()),
		clitest.RootCommand().ProviderAt(0).Outcome(cli.CommandProviderOutcomeRejected),
	)
}

func TestProviderAccessRejectsStaleLocalDestinationFromPriorOperation(t *testing.T) {
	t.Parallel()

	type config = providerMaterializationSourceValueConfig
	var stale cli.Destination[config, string]
	providerCalls := 0
	app := clitest.Compile(t, cli.Definition[config]{
		Name: providerMaterializationToolName,
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				providerCalls++
				if providerCalls == 1 {
					batch.Command(cli.Command[config]{
						Name: "source",
						Declare: func(command *cli.CommandBuilder[config]) {
							stale = command.Values.String(&command.Config.Source)
						},
					})
					return nil
				}
				batch.Command(cli.Command[config]{
					Name: "target",
					Declare: func(command *cli.CommandBuilder[config]) {
						command.Access.Value(stale, &command.Config.Value)
					},
				})
				return nil
			})
		},
	})

	root := clitest.RootCommand()
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"source"},
	}).Require(t,
		clitest.InspectionReady(),
		root.ProviderAt(0).Outcome(cli.CommandProviderOutcomeAccepted),
		root.Command("source").Exists(),
	)
	if providerCalls != 1 {
		t.Fatalf("provider calls after accepted operation = %d, want 1", providerCalls)
	}

	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"target"},
	}).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidCommandBatch).Present()),
		root.ProviderAt(0).Outcome(cli.CommandProviderOutcomeRejected),
		root.Command("target").ObservedDoesNotExist(),
	)
	if providerCalls != 2 {
		t.Fatalf("provider calls after stale-carrier operation = %d, want 2", providerCalls)
	}
}

func TestProviderRuleAndAdvisoryDestinationPolarityReachesDraft(t *testing.T) {
	t.Parallel()

	type config = providerMaterializationSourceConfig
	type relationFamily = providerMaterializationRelationFamily
	const (
		ruleSubject relationFamily = iota
		ruleDependency
		advisorySubject
		advisoryDependency
	)
	type relation = providerMaterializationRelation
	const (
		relationAcceptedExternal relation = iota
		relationAncestor
		relationSibling
		relationDescendant
	)

	apply := func(command *cli.CommandBuilder[config], source cli.Destination[config, string], family relationFamily) {
		switch family {
		case ruleSubject:
			command.Rule.Check("relation", func(context.Context, *config, cli.Dependencies) (bool, error) { return true, nil }).
				Subject(source)
		case ruleDependency:
			command.Rule.Check("relation", func(context.Context, *config, cli.Dependencies) (bool, error) { return true, nil }).
				DependsOn(source)
		case advisorySubject:
			command.Advisory.Always(cli.Advisory{
				Code:     "relation",
				Severity: cli.NoticeSeverityInformation,
				Message:  "relation",
			}).Subject(source)
		case advisoryDependency:
			command.Advisory.Detect(func(context.Context, *config, cli.Dependencies) (bool, error) { return true, nil }, cli.Advisory{
				Code:     "relation",
				Severity: cli.NoticeSeverityInformation,
				Message:  "relation",
			}).
				DependsOn(source)
		}
	}
	definition := func(family relationFamily, relationship relation) cli.Definition[config] {
		definition := cli.Definition[config]{
			Name: providerMaterializationToolName,
			Declare: func(root *cli.CommandBuilder[config]) {
				var accepted cli.Destination[config, string]
				if relationship == relationAcceptedExternal {
					accepted = root.Values.String(&root.Config.Source)
				}
				root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
					switch relationship {
					case relationAcceptedExternal:
						batch.Command(cli.Command[config]{
							Name: "target",
							Declare: func(command *cli.CommandBuilder[config]) {
								apply(command, accepted, family)
							},
						})
					case relationAncestor:
						batch.Command(cli.Command[config]{
							Name: "parent",
							Declare: func(parent *cli.CommandBuilder[config]) {
								source := parent.Values.String(&parent.Config.Source)
								parent.Subcommands.Command(cli.Command[config]{
									Name: "child",
									Declare: func(child *cli.CommandBuilder[config]) {
										apply(child, source, family)
									},
								})
							},
						})
					case relationSibling:
						var source cli.Destination[config, string]
						batch.Command(cli.Command[config]{
							Name: "source",
							Declare: func(command *cli.CommandBuilder[config]) {
								source = command.Values.String(&command.Config.Source)
							},
						})
						batch.Command(cli.Command[config]{
							Name: "target",
							Declare: func(command *cli.CommandBuilder[config]) {
								apply(command, source, family)
							},
						})
					case relationDescendant:
						batch.Command(cli.Command[config]{
							Name: "parent",
							Declare: func(parent *cli.CommandBuilder[config]) {
								var source cli.Destination[config, string]
								parent.Subcommands.Command(cli.Command[config]{
									Name: "source",
									Declare: func(command *cli.CommandBuilder[config]) {
										source = command.Values.String(&command.Config.Source)
									},
								})
								apply(parent, source, family)
							},
						})
					}
					return nil
				})
			},
		}
		return definition
	}

	families := []struct {
		name   string
		family relationFamily
	}{
		{
			name:   "rule subject",
			family: ruleSubject,
		},
		{
			name:   "rule dependency",
			family: ruleDependency,
		},
		{
			name:   "advisory subject",
			family: advisorySubject,
		},
		{
			name:   "advisory dependency",
			family: advisoryDependency,
		},
	}
	relations := []struct {
		name         string
		relationship relation
		valid        bool
	}{
		{
			name:         "accepted static ancestor",
			relationship: relationAcceptedExternal,
			valid:        true,
		},
		{
			name:         "provider-local ancestor",
			relationship: relationAncestor,
			valid:        true,
		},
		{
			name:         "provider-local sibling",
			relationship: relationSibling,
			valid:        false,
		},
		{
			name:         "provider-local descendant",
			relationship: relationDescendant,
			valid:        false,
		},
	}
	for _, family := range families {
		t.Run(family.name, func(t *testing.T) {
			t.Parallel()
			for _, relationship := range relations {
				t.Run(relationship.name, func(t *testing.T) {
					t.Parallel()
					app := clitest.Compile(t, definition(family.family, relationship.relationship))
					proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{})
					if !relationship.valid {
						proof.Require(t,
							clitest.InspectionRejected(),
							clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidCommandBatch).Present()),
							clitest.RootCommand().ProviderAt(0).Outcome(cli.CommandProviderOutcomeRejected),
						)
						return
					}

					root := clitest.RootCommand()
					var target clitest.CommandLocator
					var source clitest.ValueLocator
					switch relationship.relationship {
					case relationAcceptedExternal:
						target = root.Command("target")
						source = root.ValueAt(0)
					case relationAncestor:
						parent := root.Command("parent")
						target = parent.Command("child")
						source = parent.ValueAt(0)
					default:
						t.Fatalf(
							"valid relationship reached unsupported locator relation %d",
							relationship.relationship,
						)
						return
					}
					expectations := []clitest.InspectionExpectation{
						clitest.InspectionReady(),
						root.ProviderAt(0).Outcome(cli.CommandProviderOutcomeAccepted),
					}
					switch family.family {
					case ruleSubject:
						expectations = append(expectations, target.Rule("relation").SubjectsExactly(source))
					case ruleDependency:
						expectations = append(expectations, target.Rule("relation").DependenciesExactly(source))
					case advisorySubject:
						expectations = append(expectations, target.Advisory("relation").SubjectsExactly(source))
					case advisoryDependency:
						expectations = append(expectations, target.Advisory("relation").DependenciesExactly(source))
					}
					proof.Require(t, expectations...)
				})
			}
		})
	}
}

func TestProviderRuleGroupAcceptedExternalDestinationReachesDraft(t *testing.T) {
	t.Parallel()

	type config = providerMaterializationSeedLocalConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: providerMaterializationToolName,
		Declare: func(root *cli.CommandBuilder[config]) {
			seed := root.Values.String(&root.Config.Seed)
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Command(cli.Command[config]{
					Name: "target",
					Declare: func(command *cli.CommandBuilder[config]) {
						local := command.Values.String(&command.Config.Local)
						command.Rule.ExactlyOne(seed, local)
					},
				})
				return nil
			})
		},
	})

	root := clitest.RootCommand()
	target := root.Command("target")
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		clitest.InspectionReady(),
		root.ProviderAt(0).Outcome(cli.CommandProviderOutcomeAccepted),
		target.RuleAt(0).Kind(cli.RuleExactlyOne),
		target.RuleAt(0).DependenciesExactly(root.ValueAt(0), target.ValueAt(0)),
	)
}

func TestProviderAcceptedExternalDestinationCarrierPaths(t *testing.T) {
	t.Parallel()

	type config = providerMaterializationSeedPathConfig

	t.Run("completion dependency", func(t *testing.T) {
		t.Parallel()
		app := clitest.Compile(t, cli.Definition[config]{
			Name: providerMaterializationToolName,
			Declare: func(root *cli.CommandBuilder[config]) {
				seed := root.Values.String(&root.Config.Seed)
				root.Values.Default(seed, "seed")
				root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
					batch.Command(cli.Command[config]{
						Name: "target",
						Declare: func(child *cli.CommandBuilder[config]) {
							child.Flags.String(&child.Config.Path).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
								return cli.CompletionContribution{}, nil
							}).From(seed).
								Long("path")
						},
					})
					return nil
				})
			},
		})
		root := clitest.RootCommand()
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
			clitest.InspectionReady(),
			root.ProviderAt(0).Outcome(cli.CommandProviderOutcomeAccepted),
			root.Command("target").Flag("--path").Completion().DependenciesExactly(root.ValueAt(0)),
		)
	})

	t.Run("nested provider dependency", func(t *testing.T) {
		t.Parallel()
		app := clitest.Compile(t, cli.Definition[config]{
			Name: providerMaterializationToolName,
			Declare: func(root *cli.CommandBuilder[config]) {
				seed := root.Values.String(&root.Config.Seed)
				root.Values.Default(seed, "seed")
				root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
					batch.Command(cli.Command[config]{
						Name: "target",
						Declare: func(child *cli.CommandBuilder[config]) {
							child.Subcommands.CommandsFrom(
								func(_ context.Context, nested *cli.CommandBatch[config]) error {
									nested.Command(cli.Command[config]{
										Name: "nested",
									})
									return nil
								},
								seed,
							)
						},
					})
					return nil
				})
			},
		})
		root := clitest.RootCommand()
		target := root.Command("target")
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"target"},
		}).Require(t,
			clitest.InspectionReady(),
			root.ProviderAt(0).Outcome(cli.CommandProviderOutcomeAccepted),
			target.ProviderAt(0).Outcome(cli.CommandProviderOutcomeAccepted),
			target.ProviderAt(0).DependenciesExactly(root.ValueAt(0)),
			target.Command("nested").Exists(),
		)
	})

	t.Run("shortcut default", func(t *testing.T) {
		t.Parallel()
		app := clitest.Compile(t, cli.Definition[config]{
			Name: providerMaterializationToolName,
			Declare: func(root *cli.CommandBuilder[config]) {
				seed := root.Values.String(&root.Config.Seed)
				root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
					batch.Command(cli.Command[config]{
						Name: "target",
					})
					batch.Shortcut(cli.CommandPath{"target"}, cli.CommandShortcut{
						Name: "shortcut",
					}).Default(seed, "shortcut-seed")
					return nil
				})
			},
		})
		root := clitest.RootCommand()
		shortcut := root.Shortcut("shortcut")
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
			clitest.InspectionReady(),
			root.ProviderAt(0).Outcome(cli.CommandProviderOutcomeAccepted),
			shortcut.ShortcutDefaultCount(1),
			shortcut.ShortcutDefaultAt(0).Destination(root.ValueAt(0)),
			shortcut.ShortcutDefaultAt(0).ValueCount(1),
			shortcut.ShortcutDefaultAt(0).ValueAt(0).Text("shortcut-seed"),
		)
	})
}

type providerTokenPairValue struct {
	Left  string
	Right string
}

func providerTokenPairCodec() cli.ValueCodec[providerTokenPairValue] {
	codec := cli.ValueCodec[providerTokenPairValue]{
		Name: "pair-value",
		Parse: func(text string) (providerTokenPairValue, error) {
			return providerTokenPairValue{}, fmt.Errorf("destination codec must not parse token-language input %q", text)
		},
		Format:   func(value providerTokenPairValue) string { return value.Left + ":" + value.Right },
		Snapshot: cli.Value.Assign[providerTokenPairValue](),
	}
	return codec
}
