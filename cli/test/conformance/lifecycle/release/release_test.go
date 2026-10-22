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

type authoringReplacementNameLegacyModeConfig struct{ Name, Legacy, Mode, Gate, OldFlag, NewFlag string }

type authoringReplacementLegacyCommandConfig struct{ Name string }

type authoringReplacementCurrentCommandConfig struct{ Verbose bool }

type authoringReplacementNameConfig struct{ Name string }

func TestReplacementAuthoringProjectsDeclaredSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringReplacementNameConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: func(root *cli.CommandBuilder[config]) {
			flag := root.Flags.String(&root.Config.Name)
			old := flag.Long("old").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2",
			})
			current := flag.Long("new").Release(cli.ReleaseLifecycle{
				IntroducedIn: "1",
			})
			old.ReplacedBy(current).Hidden()
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				a := form.Literal("old").Release(cli.ReleaseLifecycle{
					DeprecatedIn: "2",
				})
				b := form.Literal("new").Release(cli.ReleaseLifecycle{
					IntroducedIn: "1",
				})
				a.ReplacedBy(b).EndOfOptions()
			})
		},
	}
	app := clitest.Compile(t, definition)
	clitest.Schema(app).Require(t,
		clitest.RootCommand().FlagAt(0).Spelling("--old").ReplacedBy(clitest.RootCommand().FlagAt(0).Spelling("--new")),
		clitest.RootCommand().FlagAt(0).Spelling("--old").Hidden(),
		clitest.RootCommand().
			ArgumentFormAt(0).
			Literal("old").
			ReplacedBy(clitest.RootCommand().ArgumentFormAt(0).Literal("new")),
		clitest.RootCommand().ArgumentFormAt(0).Literal("old").EndOfOptions(),
	)
}

func TestReplacementAuthoringProjectsDistinctReplacementCarriers(t *testing.T) {
	t.Parallel()
	type config = authoringReplacementNameLegacyModeConfig
	domain := cli.Value.Domain(cli.Value.Assign[string](), func(b *cli.ValueDomainBuilder[string]) {
		modern := b.Value("modern", "modern")
		legacyValue := b.Value("legacy-value", "legacy-value")
		legacyValue.ReplacedBy(modern).Hidden()
		legacy := b.Alias("legacy", modern)
		legacy.ReplacedBy(modern).Hidden()
	})
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.Value(&root.Config.Mode, domain).Long("mode")
			oldFlag := root.Flags.String(&root.Config.OldFlag).Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2",
			})
			oldFlag.Long("old-value")
			newFlag := root.Flags.String(&root.Config.NewFlag).Release(cli.ReleaseLifecycle{
				IntroducedIn: "1",
			})
			newFlag.Long("new-value")
			oldFlag.ReplacedBy(newFlag).Hidden()
			gate := root.Values.String(&root.Config.Gate)
			root.Values.Default(gate, "visible")
			visible := root.When.Resolved(gate)
			oldCommand := root.Subcommands.Command(cli.Command[authoringReplacementLegacyCommandConfig]{
				Name: "old",
				Release: cli.ReleaseLifecycle{
					DeprecatedIn: "2",
				},
				Declare: func(child *cli.CommandBuilder[authoringReplacementLegacyCommandConfig]) {
					child.Flags.String(&child.Config.Name).Long("name")
				},
			})
			newCommand := root.Subcommands.Command(cli.Command[authoringReplacementCurrentCommandConfig]{
				Name: "new",
				Release: cli.ReleaseLifecycle{
					IntroducedIn: "1",
				},
			})
			oldCommand.ReplacedBy(newCommand).VisibleWhen(visible)
			var oldArg, newArg cli.ArgumentDeclaration[config, string]
			oldForm := root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				oldArg = form.String(&form.Config.Legacy, "old").Release(cli.ReleaseLifecycle{
					DeprecatedIn: "2",
				})
				newArg = form.String(&form.Config.Name, "new").Release(cli.ReleaseLifecycle{
					IntroducedIn: "1",
				})
				oldArg.ReplacedBy(newArg).Hidden()
			}).Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2",
			})
			newForm := root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) { form.Literal("replacement") }).
				Release(cli.ReleaseLifecycle{
					IntroducedIn: "1",
				})
			oldForm.ReplacedBy(newForm).VisibleWhen(visible)
		},
	}
	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	oldForm, newForm := root.ArgumentFormAt(0), root.ArgumentFormAt(1)
	domainLocator := root.Flag("--mode").Value().Domain(cli.ValueDomainUseDestination)
	clitest.Schema(app).Require(t,
		root.ArgumentFormsExactly(oldForm, newForm),
		oldForm.ReplacedBy(newForm),
		oldForm.Term("old").ReplacedBy(oldForm.Term("new")),
		oldForm.Term("old").Hidden(),
		root.Command("old").CanonicalEntry().ReplacedBy(root.Command("new").CanonicalEntry()),
		domainLocator.Entry("legacy").ReplacedBy(domainLocator.Entry("modern")),
		domainLocator.Entry("legacy-value").ReplacedBy(domainLocator.Entry("modern")),
		domainLocator.Entry("legacy-value").Hidden(),
		root.Flag("--old-value").ReplacedBy(root.Flag("--new-value")),
		root.Flag("--old-value").Hidden(),
	)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		root.Command("old").Visible(),
		root.ArgumentFormAt(0).Visible(),
	)
}

func TestReplacementAuthoringRejectsDuplicateAndCyclicRelations(t *testing.T) {
	t.Parallel()
	t.Run("duplicate source", func(t *testing.T) {
		t.Parallel()
		type config struct{ Old, Current, Other string }
		definition := cli.Definition[config]{
			Name:    "tool",
			Release: cli.Release.Ordered("2", "1", "2"),
			Declare: func(root *cli.CommandBuilder[config]) {
				old := root.Flags.String(&root.Config.Old).Release(cli.ReleaseLifecycle{
					DeprecatedIn: "2",
				})
				old.Long("old")
				current := root.Flags.String(&root.Config.Current).Release(cli.ReleaseLifecycle{
					IntroducedIn: "1",
				})
				current.Long("current")
				other := root.Flags.String(&root.Config.Other).Release(cli.ReleaseLifecycle{
					IntroducedIn: "1",
				})
				other.Long("other")
				old.ReplacedBy(current).ReplacedBy(other)
			},
		}
		clitest.CompileFails(t, definition).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
	})
	t.Run("cycle", func(t *testing.T) {
		t.Parallel()
		type config struct{ First, Second string }
		definition := cli.Definition[config]{
			Name:    "tool",
			Release: cli.Release.Ordered("2", "1", "2"),
			Declare: func(root *cli.CommandBuilder[config]) {
				first := root.Flags.String(&root.Config.First).Release(cli.ReleaseLifecycle{
					IntroducedIn: "1",
					DeprecatedIn: "2",
				})
				first.Long("first")
				second := root.Flags.String(&root.Config.Second).Release(cli.ReleaseLifecycle{
					IntroducedIn: "1",
					DeprecatedIn: "2",
				})
				second.Long("second")
				first.ReplacedBy(second)
				second.ReplacedBy(first)
			},
		}
		clitest.CompileFails(t, definition).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
	})
}

func TestReplacedByPublishesReplacementRelation(t *testing.T) {
	t.Parallel()
	type config = authoringReplacementNameConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: func(root *cli.CommandBuilder[config]) {
			flag := root.Flags.String(&root.Config.Name)
			legacy := flag.Long("legacy").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2",
			})
			current := flag.Long("current").Release(cli.ReleaseLifecycle{
				IntroducedIn: "1",
			})
			legacy.ReplacedBy(current)
		},
	}
	app := clitest.Compile(t, definition)
	clitest.Schema(app).
		Require(t, clitest.RootCommand().FlagAt(0).Spelling("--legacy").ReplacedBy(clitest.RootCommand().FlagAt(0).Spelling("--current")))
}

type releaseBuilderMisuseConfig struct {
	Value string
}
type commandEntryProjectionGateConfig struct{ Gate string }

type commandEntryProjectionFixture struct {
	app  cli.Application
	page *cli.HelpPage
}

func newStaticCommandEntryProjectionFixture(t *testing.T, targetHidden bool, targetRelease cli.ReleaseLifecycle, entryHidden bool, entryRelease cli.ReleaseLifecycle) commandEntryProjectionFixture {
	t.Helper()
	var page cli.HelpPage
	definition := cli.Definition[struct{}]{
		Name:    "tool",
		Release: cli.Release.SemVer("2.0.0"),
		Help: cli.HelpConfig{
			Renderer: func(_ context.Context, got cli.HelpPage) (string, error) {
				page = got
				return "help\n", nil
			},
		},
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.Command(cli.Command[struct{}]{
				Name:    "target",
				Hidden:  targetHidden,
				Release: targetRelease,
				Aliases: []cli.CommandAlias{{
					Name:    "alias",
					Hidden:  entryHidden,
					Release: entryRelease,
				}},
				Declare: func(target *cli.CommandBuilder[struct{}]) {
					target.Shortcut(cli.CommandShortcut{
						Name:    "shortcut",
						Args:    []string{"--"},
						Hidden:  entryHidden,
						Release: entryRelease,
					})
				},
				Run: func(context.Context, *struct{}) error { return nil },
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile(): %v", err)
	}
	fixture := commandEntryProjectionFixture{
		app:  app,
		page: &page,
	}
	return fixture
}

func TestCommandEntryProjectionNeverBroadensStaticCanonicalTarget(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name               string
		targetHidden       bool
		targetRelease      cli.ReleaseLifecycle
		entryHidden        bool
		entryRelease       cli.ReleaseLifecycle
		wantAvailable      bool
		wantVisible        bool
		wantProjected      bool
		wantEffectiveState cli.LifecycleState
		wantInvocationOK   bool
	}{
		{
			name:               "target hidden entry visible",
			targetHidden:       true,
			wantAvailable:      true,
			wantEffectiveState: cli.LifecycleStateCurrent,
			wantInvocationOK:   true,
		},
		{
			name:               "target visible entry hidden",
			entryHidden:        true,
			wantAvailable:      true,
			wantEffectiveState: cli.LifecycleStateCurrent,
			wantInvocationOK:   true,
		},
		{
			name: "target deprecated entry current",
			targetRelease: cli.ReleaseLifecycle{
				DeprecatedIn: "2.0.0",
			},
			wantAvailable:      true,
			wantVisible:        true,
			wantProjected:      true,
			wantEffectiveState: cli.LifecycleStateDeprecated,
			wantInvocationOK:   true,
		},
		{
			name: "target removed entry current",
			targetRelease: cli.ReleaseLifecycle{
				RemovedIn: "2.0.0",
			},
			wantEffectiveState: cli.LifecycleStateRemoved,
		},
		{
			name: "target current entry removed",
			entryRelease: cli.ReleaseLifecycle{
				RemovedIn: "2.0.0",
			},
			wantEffectiveState: cli.LifecycleStateRemoved,
		},
		{
			name: "target not introduced",
			targetRelease: cli.ReleaseLifecycle{
				IntroducedIn: "3.0.0",
			},
			wantEffectiveState: cli.LifecycleStateNotIntroduced,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			fixture := newStaticCommandEntryProjectionFixture(
				t,
				test.targetHidden,
				test.targetRelease,
				test.entryHidden,
				test.entryRelease,
			)
			for _, name := range []string{"alias", "shortcut"} {
				entry := commandEntryProjectionReferenceEntry(t, fixture.app, name)
				if entry.Available != test.wantAvailable || entry.Visible != test.wantVisible {
					t.Fatalf(
						"Reference entry %q available/visible = %v/%v, want %v/%v",
						name,
						entry.Available,
						entry.Visible,
						test.wantAvailable,
						test.wantVisible,
					)
				}
				if entry.LifecycleState != test.wantEffectiveState {
					t.Fatalf(
						"Reference entry %q effective lifecycle = %v, want %v",
						name,
						entry.LifecycleState,
						test.wantEffectiveState,
					)
				}
				if entry.Release != test.entryRelease || entry.Hidden != test.entryHidden {
					t.Fatalf(
						"Reference entry %q local metadata = release %#v hidden %v, want release %#v hidden %v",
						name,
						entry.Release,
						entry.Hidden,
						test.entryRelease,
						test.entryHidden,
					)
				}
				item, projected := commandEntryProjectionCompletionItem(t, fixture.app, name)
				if projected != test.wantProjected {
					t.Fatalf("Completion published %q = %v, want %v", name, projected, test.wantProjected)
				}
				if projected && item.LifecycleState != test.wantEffectiveState {
					t.Fatalf(
						"Completion entry %q lifecycle = %v, want %v",
						name,
						item.LifecycleState,
						test.wantEffectiveState,
					)
				}
			}
			commandEntryProjectionRunHelp(t, fixture)
			for _, name := range []string{"alias", "shortcut"} {
				command, projected := commandEntryProjectionHelpCommand(*fixture.page, name)
				if projected != test.wantProjected {
					t.Fatalf("Help published %q = %v, want %v", name, projected, test.wantProjected)
				}
				if projected && command.LifecycleState != test.wantEffectiveState {
					t.Fatalf(
						"Help entry %q lifecycle = %v, want %v",
						name,
						command.LifecycleState,
						test.wantEffectiveState,
					)
				}
				_, err := fixture.app.Inspect(t.Context(), cli.InspectionRequest{
					Args: []string{name},
				})
				if (err == nil) != test.wantInvocationOK {
					t.Fatalf("Inspect(%q) error = %v, want success %v", name, err, test.wantInvocationOK)
				}
			}
		})
	}
}

func TestProviderCommandEntryProjectionNarrowsCompleteCanonicalTargetPath(t *testing.T) {
	t.Parallel()
	t.Run("intermediate hidden", func(t *testing.T) {
		t.Parallel()
		fixture := newProviderCommandEntryProjectionFixture(t, true, cli.ReleaseLifecycle{})
		for _, name := range []string{"provider-alias", "provider-shortcut"} {
			if _, err := fixture.app.Inspect(t.Context(), cli.InspectionRequest{
				Args: []string{name},
			}); err != nil {
				t.Fatalf("Inspect(%q): %v", name, err)
			}
			if _, ok := commandEntryProjectionCompletionItem(t, fixture.app, name); ok {
				t.Fatalf("Completion unexpectedly published provider entry %q", name)
			}
		}
		commandEntryProjectionRunHelp(t, fixture)
		for _, name := range []string{"provider-alias", "provider-shortcut"} {
			if _, ok := commandEntryProjectionHelpCommand(*fixture.page, name); ok {
				t.Fatalf("Help unexpectedly published provider entry %q", name)
			}
		}
		if _, err := fixture.app.Inspect(t.Context(), cli.InspectionRequest{
			Args: []string{"provider-shortcut"},
		}); err != nil {
			t.Fatalf("Inspect(provider-shortcut): %v", err)
		}
	})

	t.Run("intermediate not introduced", func(t *testing.T) {
		t.Parallel()
		fixture := newProviderCommandEntryProjectionFixture(t, false, cli.ReleaseLifecycle{
			IntroducedIn: "3.0.0",
		})
		if _, err := fixture.app.Inspect(t.Context(), cli.InspectionRequest{
			Args: []string{"provider-shortcut"},
		}); err == nil {
			t.Fatal("Inspect(provider-shortcut) error = nil for not-introduced intermediate target")
		}
	})

	t.Run("intermediate unavailable", func(t *testing.T) {
		t.Parallel()
		app := newProviderDeepShortcutAvailabilityFixture(t)
		if _, err := app.Inspect(t.Context(), cli.InspectionRequest{
			Args: []string{"provider-shortcut"},
		}); err == nil {
			t.Fatal("Inspect(provider-shortcut) error = nil for unavailable intermediate target")
		}
	})

	t.Run("intermediate removed", func(t *testing.T) {
		t.Parallel()
		fixture := newProviderCommandEntryProjectionFixture(t, false, cli.ReleaseLifecycle{
			RemovedIn: "2.0.0",
		})
		for _, name := range []string{"provider-alias", "provider-shortcut"} {
			if _, ok := commandEntryProjectionCompletionItem(t, fixture.app, name); ok {
				t.Fatalf("Completion unexpectedly published provider entry %q", name)
			}
			inspection, err := fixture.app.Inspect(t.Context(), cli.InspectionRequest{
				Args: []string{name},
			})
			if err == nil {
				t.Fatalf("Inspect(%q) error = nil for removed intermediate target", name)
			}
			entry, ok := commandEntryProjectionInspectionEntry(inspection, name)
			if !ok {
				t.Fatalf("Inspection missing provider entry %q", name)
			}
			if entry.Available || entry.LifecycleState != cli.LifecycleStateRemoved {
				t.Fatalf(
					"Inspection provider entry %q available/lifecycle = %v/%v, want false/%v",
					name,
					entry.Available,
					entry.LifecycleState,
					cli.LifecycleStateRemoved,
				)
			}
		}
		commandEntryProjectionRunHelp(t, fixture)
		for _, name := range []string{"provider-alias", "provider-shortcut"} {
			if _, ok := commandEntryProjectionHelpCommand(*fixture.page, name); ok {
				t.Fatalf("Help unexpectedly published provider entry %q", name)
			}
		}
	})
}

func newProviderCommandEntryProjectionFixture(t *testing.T, parentHidden bool, parentRelease cli.ReleaseLifecycle) commandEntryProjectionFixture {
	t.Helper()
	var page cli.HelpPage
	definition := cli.Definition[struct{}]{
		Name:    "tool",
		Release: cli.Release.SemVer("2.0.0"),
		Help: cli.HelpConfig{
			Renderer: func(_ context.Context, got cli.HelpPage) (string, error) {
				page = got
				return "help\n", nil
			},
		},
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[struct{}]) error {
				batch.Command(cli.Command[struct{}]{
					Name:    "parent",
					Hidden:  parentHidden,
					Release: parentRelease,
					Declare: func(parent *cli.CommandBuilder[struct{}]) {
						parent.Subcommands.Command(cli.Command[struct{}]{
							Name: "terminal",
							Run:  func(context.Context, *struct{}) error { return nil },
						})
					},
				})
				batch.Alias(cli.CommandPath{"parent"}, cli.CommandAlias{
					Name: "provider-alias",
				})
				batch.Shortcut(cli.CommandPath{"parent", "terminal"}, cli.CommandShortcut{
					Name: "provider-shortcut",
					Args: []string{"--"},
				})
				return nil
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile(): %v", err)
	}
	fixture := commandEntryProjectionFixture{
		app:  app,
		page: &page,
	}
	return fixture
}

func commandEntryProjectionReferenceEntry(t *testing.T, app cli.Application, name string) cli.CommandEntryReference {
	t.Helper()
	reference, err := clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityPublic,
		Order:      cli.ReferenceOrderDeclaration,
	}).Reference()
	if err != nil {
		t.Fatalf("Reference(): %v", err)
	}
	for _, entry := range reference.CommandEntries {
		if entry.Name == name {
			return entry
		}
	}
	t.Fatalf("Reference missing command entry %q", name)
	return cli.CommandEntryReference{}
}

func commandEntryProjectionInspectionEntry(inspection cli.Inspection, name string) (cli.CommandEntryInspection, bool) {
	for _, entry := range inspection.CommandEntries {
		if entry.Name == name {
			return entry, true
		}
	}
	return cli.CommandEntryInspection{}, false
}

func commandEntryProjectionCompletionItem(t *testing.T, app cli.Application, name string) (cli.CompletionItem, bool) {
	t.Helper()
	completion, err := clitest.Complete(t.Context(), app, cli.CompletionRequest{
		Args: []string{""},
		Cursor: cli.CompletionCursor{
			Argument: 0,
		},
	}).Completion()
	if err != nil {
		t.Fatalf("Complete(): %v", err)
	}
	for _, item := range completion.Items {
		if item.Value == name {
			return item, true
		}
	}
	return cli.CompletionItem{}, false
}

func commandEntryProjectionRunHelp(t *testing.T, fixture commandEntryProjectionFixture) {
	t.Helper()
	*fixture.page = cli.HelpPage{}
	var output strings.Builder
	if err := fixture.app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--help"},
		Stdout: &output,
	}); err != nil {
		t.Fatalf("Run(--help): %v", err)
	}
}

func commandEntryProjectionHelpCommand(page cli.HelpPage, name string) (cli.HelpCommand, bool) {
	for _, group := range page.CommandGroups {
		for _, command := range group.Commands {
			if command.Name == name {
				return command, true
			}
		}
	}
	return cli.HelpCommand{}, false
}

func newProviderDeepShortcutAvailabilityFixture(t *testing.T) cli.Application {
	t.Helper()
	type config = commandEntryProjectionGateConfig
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			gate := root.Values.String(&root.Config.Gate)
			root.Values.Default(gate, "off")
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Command(cli.Command[config]{
					Name: "parent",
					Declare: func(parent *cli.CommandBuilder[config]) {
						parent.AvailableWhen(parent.When.Equal(gate, "on"))
						parent.Subcommands.Command(cli.Command[config]{
							Name: "terminal",
							Run:  func(context.Context, *config) error { return nil },
						})
					},
				})
				batch.Shortcut(cli.CommandPath{"parent", "terminal"}, cli.CommandShortcut{
					Name: "provider-shortcut",
					Args: []string{"--"},
				})
				return nil
			})
		},
	})
	if err != nil {
		t.Fatalf("Compile(): %v", err)
	}
	return app
}

type lifecycleRegionRootConfig struct{ Region string }

type lifecycleValueConfig struct{ Value string }

func TestAuthoringAcceptanceUsesEvaluatedReleaseForFiniteDomainMembership(t *testing.T) {
	t.Parallel()
	type config = lifecycleValueConfig
	definition := cli.Definition[config]{
		Name:    "release-domain",
		Release: cli.Release.SemVer("2.0.0"),
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := cli.Value.Domain(cli.Value.Assign[string](), func(builder *cli.ValueDomainBuilder[string]) {
				current := builder.Value("current", "current")
				builder.Alias("cur", current)
				builder.Value("deprecated", "deprecated").Release(cli.ReleaseLifecycle{
					DeprecatedIn: "2.0.0",
				})
				builder.Value("removed", "removed").Release(cli.ReleaseLifecycle{
					RemovedIn: "2.0.0",
				})
				builder.Value("future", "future").Release(cli.ReleaseLifecycle{
					IntroducedIn: "3.0.0",
				})
			})
			root.Values.Value(&root.Config.Value, domain)
		},
	}
	app := clitest.Compile(t, definition)
	schema := app.Schema()
	root := schema.Commands[schema.Root]
	if len(root.Values) != 1 {
		t.Fatalf("root values = %v, want one", root.Values)
	}
	entries := valueDomainEntryIDsForUse(t, schema, root.Values[0], cli.ValueDomainUseDestination)
	clitest.Reference(app, cli.ReferenceOptions{
		Release:    "2.0.0",
		Visibility: cli.ReferenceVisibilityAll,
	}).Require(t,
		clitest.ValueDomainEntryByID(entries["current"]).Lifecycle(cli.LifecycleStateCurrent),
		clitest.ValueDomainEntryByID(entries["deprecated"]).Lifecycle(cli.LifecycleStateDeprecated),
		clitest.ValueDomainEntryByID(entries["removed"]).Lifecycle(cli.LifecycleStateRemoved),
		clitest.ValueDomainEntryByID(entries["future"]).Lifecycle(cli.LifecycleStateNotIntroduced),
	)
}

func valueDomainEntryIDsForUse(t *testing.T, schema cli.Schema, destination cli.ValueID, kind cli.ValueDomainUseKind) map[string]cli.ValueDomainEntryID {
	t.Helper()
	for _, domain := range schema.ValueDomains {
		for _, use := range domain.Uses {
			if use.Destination != destination || use.Kind != kind {
				continue
			}
			entries := make(map[string]cli.ValueDomainEntryID, len(domain.Entries))
			for _, id := range domain.Entries {
				entry, ok := schema.ValueDomainEntries[id]
				if !ok {
					t.Fatalf("value-domain entry %v is absent", id)
				}
				entries[entry.Name] = id
			}
			return entries
		}
	}
	t.Fatalf("value-domain use destination=%v kind=%v is absent", destination, kind)
	return nil
}

func TestAuthoringAcceptanceRejectsEffectiveAliasToLifecycleInactiveCanonicalValue(t *testing.T) {
	t.Parallel()
	type config = lifecycleValueConfig
	definition := cli.Definition[config]{
		Name:    "bad-alias",
		Release: cli.Release.SemVer("2.0.0"),
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := cli.Value.Domain(cli.Value.Assign[string](), func(builder *cli.ValueDomainBuilder[string]) {
				removed := builder.Value("removed", "removed").Release(cli.ReleaseLifecycle{
					RemovedIn: "2.0.0",
				})
				builder.Alias("still-active", removed)
			})
			root.Values.Value(&root.Config.Value, domain)
		},
	}
	clitest.CompileFails(t, definition).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestAuthoringAcceptanceRejectsLifecycleInactiveStaticDomainDefault(t *testing.T) {
	t.Parallel()
	type config = lifecycleValueConfig
	definition := cli.Definition[config]{
		Name:    "domain-default-current",
		Release: cli.Release.SemVer("2.0.0"),
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := cli.Value.Domain(cli.Value.Assign[string](), func(builder *cli.ValueDomainBuilder[string]) {
				builder.Value("current", "current")
				builder.Value("removed", "removed").Release(cli.ReleaseLifecycle{
					RemovedIn: "2.0.0",
				})
			})
			d := root.Values.Value(&root.Config.Value, domain)
			root.Values.Default(d, "current")
		},
	}
	_ = clitest.Compile(t, definition)
	definitionUnderTest := cli.Definition[config]{
		Name:    "domain-default-removed",
		Release: cli.Release.SemVer("2.0.0"),
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := cli.Value.Domain(cli.Value.Assign[string](), func(builder *cli.ValueDomainBuilder[string]) {
				builder.Value("current", "current")
				builder.Value("removed", "removed").Release(cli.ReleaseLifecycle{
					RemovedIn: "2.0.0",
				})
			})
			d := root.Values.Value(&root.Config.Value, domain)
			root.Values.Default(d, "removed")
		},
	}
	clitest.CompileFails(t, definitionUnderTest).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestCompileClassifiesActiveBuilderLifecycleMisuse(t *testing.T) {
	t.Parallel()
	clitest.CompileFails(t, cli.Definition[releaseBuilderMisuseConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[releaseBuilderMisuseConfig]) {
			root.Arguments.Form(func(_ *cli.ArgumentFormBuilder[releaseBuilderMisuseConfig]) {
				root.Values.String(&root.Config.Value)
			})
		},
	}).Require(t, clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticBuilderMisuse)))
}

func TestDefinitionAuthoringRejectsLifecycleInactiveDomainShortcutDefault(t *testing.T) {
	t.Parallel()
	type rootConfig = lifecycleRegionRootConfig
	definition := cli.Definition[rootConfig]{
		Name:    "commandment",
		Release: cli.Release.SemVer("2.0.0"),
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			domain := cli.Value.Domain(cli.Value.Assign[string](), func(builder *cli.ValueDomainBuilder[string]) {
				builder.Value("current", "current")
				builder.Value("removed", "removed").Release(cli.ReleaseLifecycle{
					RemovedIn: "2.0.0",
				})
			})
			region := root.Values.Value(&root.Config.Region, domain)
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "serve",
				Declare: func(child *cli.CommandBuilder[struct{}]) {
					shortcut := child.Shortcut(cli.CommandShortcut{
						Name: "quick",
						Args: []string{"serve"},
					})
					shortcut.Default(region, "removed")
				},
			})
		},
	}
	clitest.CompileFails(t, definition).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestReleaseSchemeConstructorsSnapshotAndReuseCanonicalSemanticAuthority(t *testing.T) {
	t.Parallel()
	order := []string{"2026.01", "2026.04", "2026.08"}
	scheme := cli.Release.Ordered("2026.04", order...)
	order[1] = "mutated"
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name:    "tool",
		Release: scheme,
	})
	for _, release := range []string{"2026.01", "2026.04", "2026.08"} {
		clitest.Reference(app, cli.ReferenceOptions{
			Release: release,
		}).Require(t, clitest.EvaluatedRelease(release))
	}
	semverApp := clitest.Compile(t, cli.Definition[struct{}]{
		Name:    "tool",
		Release: cli.Release.SemVer("1.2.3-alpha.1+build.7"),
	})
	clitest.Reference(semverApp, cli.ReferenceOptions{
		Release: "1.2.3",
	}).Require(t, clitest.EvaluatedRelease("1.2.3"))
}

func TestReleaseSchemeDefersInvalidAuthoringDataToValidationBoundary(t *testing.T) {
	t.Parallel()
	for _, scheme := range []cli.ReleaseScheme{
		cli.Release.SemVer("v1.2.3"),
		cli.Release.Ordered("missing", "one", "two"),
		cli.Release.Ordered("one", "one", "one"),
	} {
		clitest.CompileFails(t, cli.Definition[struct{}]{
			Name:    "tool",
			Release: scheme,
		}).Require(t,
			clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present(),
		)
	}
	_ = clitest.Compile(t, cli.Definition[struct{}]{
		Name:    "tool",
		Release: cli.ReleaseScheme{},
	})
}

func TestReleaseSchemeOrderedSnapshotIsIsolatedFromCallerMutation(t *testing.T) {
	t.Parallel()
	order := []string{"one", "two", "three"}
	scheme := cli.Release.Ordered("two", order...)
	order[0] = "changed"
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name:    "tool",
		Release: scheme,
	})
	for _, release := range []string{"one", "two", "three"} {
		clitest.Reference(app, cli.ReferenceOptions{
			Release: release,
		}).Require(t, clitest.EvaluatedRelease(release))
	}
	if _, err := clitest.Reference(app, cli.ReferenceOptions{
		Release: "changed",
	}).Reference(); err == nil {
		t.Fatal("caller mutation became part of release scheme")
	}
}

type releaseValueConfig struct{ Value string }

func TestTReleasePreservesReleaseSemantics(t *testing.T) {
	t.Parallel()
	type config = releaseValueConfig
	definition := cli.Definition[config]{
		Name: "release-env",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Release: cli.Release.SemVer("2.0.0"),
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			root.Values.Environment(value, "LEGACY").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2.0.0",
				Reason:       "use NEW",
			})
		},
	}
	app := clitest.Compile(t, definition)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Env: cli.Env{{
			Name:  "LEGACY",
			Value: "x",
		}},
	}).Require(t,
		clitest.InspectionReady(),
		clitest.EvaluatedRelease("2.0.0"),
		clitest.RootCommand().Environment("LEGACY").Value().Resolved(),
	)
}

func TestArgumentFormDeclarationReleasePreservesReleaseSemantics(t *testing.T) {
	t.Parallel()
	definition := cli.Definition[struct{}]{
		Name:    "release-form",
		Release: cli.Release.SemVer("2.0.0"),
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[struct{}]) { form.Literal("old") }).
				Release(cli.ReleaseLifecycle{
					DeprecatedIn: "2.0.0",
				})
		},
	}
	app := clitest.Compile(t, definition)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"old"},
	}).Require(t,
		clitest.InspectionReady(),
		clitest.EvaluatedRelease("2.0.0"),
		clitest.RootCommand().ArgumentFormAt(0).Selected(),
	)
}

func TestDomainAliasDeclarationReleasePreservesReleaseSemantics(t *testing.T) {
	t.Parallel()
	type config = releaseValueConfig
	definition := cli.Definition[config]{
		Name: "release-alias",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Release: cli.Release.SemVer("2.0.0"),
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := cli.Value.Domain(cli.Value.Assign[string](), func(builder *cli.ValueDomainBuilder[string]) {
				value := builder.Value("new", "new")
				builder.Alias("old", value).Release(cli.ReleaseLifecycle{
					DeprecatedIn: "2.0.0",
				})
			})
			root.Flags.Value(&root.Config.Value, domain).Long("value")
		},
	}
	app := clitest.Compile(t, definition)
	value := clitest.RootCommand().Flag("--value").Value()
	alias := value.Domain(cli.ValueDomainUseDestination).Entry("old")
	notice := clitest.Notice(cli.NoticeDeprecatedValue).On(alias)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--value", "old"},
	}).Require(t,
		clitest.InspectionReady(),
		clitest.EvaluatedRelease("2.0.0"),
		value.Resolved(),
		value.Text("new"),
		clitest.NoticesExactly(notice),
	)
}

func TestReleaseSchemeSemanticReleaseSchemePreservesReleaseSemantics(t *testing.T) {
	t.Parallel()
	_ = clitest.Compile(t, cli.Definition[struct{}]{
		Name:    "semver",
		Release: cli.Release.SemVer("1.2.3"),
	})
	_ = clitest.Compile(t, cli.Definition[struct{}]{
		Name:    "ordered",
		Release: cli.Release.Ordered("beta", "alpha", "beta", "stable"),
	})
	_ = clitest.Compile(t, cli.Definition[struct{}]{
		Name: "none",
	})
}
