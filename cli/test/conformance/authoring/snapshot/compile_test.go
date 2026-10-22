package conformance

import (
	"context"
	"errors"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

func TestCompileKeepsOrdinaryInvalidDeclarationDistinctFromBuilderMisuse(t *testing.T) {
	t.Parallel()
	clitest.CompileFails(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "",
			})
		},
	}).Require(t, clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticInvalidDeclaration)))
}

func TestPostCallbackBuilderMisuseRemainsProgrammerPanic(t *testing.T) {
	t.Parallel()
	var retained *cli.ArgumentFormBuilder[compileBuilderMisuseConfig]
	_, err := cli.Compile(cli.Definition[compileBuilderMisuseConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[compileBuilderMisuseConfig]) {
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[compileBuilderMisuseConfig]) {
				retained = form
				form.String(&form.Config.Value, "value")
			})
		},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("post-callback builder use did not panic")
		}
	}()
	retained.Literal("late")
}

const authoringShortcutToolName = "tool"

type authoringShortcutNameConfig struct{ Name string }

func TestShortcutAuthoringProjectsDeclaredSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringShortcutNameConfig
	definition := cli.Definition[config]{
		Name: authoringShortcutToolName,
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			name := root.Values.String(&root.Config.Name)
			root.Flags.ValueFor(name).Placement(cli.FlagPlacementSubtree).Long("name")
			root.Subcommands.Command(cli.Command[config]{
				Name: "serve",
				Declare: func(child *cli.CommandBuilder[config]) {
					first := child.Shortcut(cli.CommandShortcut{
						Name: "quick",
						Args: []string{"--name", "fast"},
					})
					first.Default(name, "fast")
					second := child.Shortcut(cli.CommandShortcut{
						Name:       "quicker",
						AppendArgs: []string{"--name", "faster"},
					})
					first.ReplacedBy(second)
				},
			})
		},
	}
	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	serve := root.Command("serve")
	quick := root.Shortcut("quick")
	quicker := root.Shortcut("quicker")
	clitest.Schema(app).Require(t,
		root.EntriesExactly(serve.CanonicalEntry(), quick, quicker),
		quick.ReplacedBy(quicker),
	)
}

func TestShortcutAuthoringProjectsDefaultAndReplacement(t *testing.T) {
	t.Parallel()
	type config = authoringShortcutNameConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: authoringShortcutToolName,
		Declare: func(root *cli.CommandBuilder[config]) {
			name := root.Values.String(&root.Config.Name)
			root.Flags.ValueFor(name).Placement(cli.FlagPlacementSubtree).Long("name")
			root.Subcommands.Command(cli.Command[config]{
				Name: "serve",
				Declare: func(child *cli.CommandBuilder[config]) {
					first := child.Shortcut(cli.CommandShortcut{
						Name: "quick",
						Args: []string{"--name", "fast"},
					})
					first.Default(name, "fast")
					second := child.Shortcut(cli.CommandShortcut{
						Name:       "quicker",
						AppendArgs: []string{"--name", "faster"},
					})
					first.ReplacedBy(second)
				},
			})
		},
	})
	root := clitest.RootCommand()
	quick := root.Shortcut("quick")
	quicker := root.Shortcut("quicker")
	def := quick.ShortcutDefaultAt(0)
	clitest.Schema(app).Require(t,
		quick.ShortcutDefaultCount(1),
		def.Destination(root.Flag("--name").Value()),
		def.ValueCount(1),
		def.ValueAt(0).Text("fast"),
		quick.ReplacedBy(quicker),
	)
}

func TestShortcutReplacedByPublishesReplacementRelation(t *testing.T) {
	t.Parallel()
	type config = authoringShortcutNameConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: authoringShortcutToolName,
		Declare: func(root *cli.CommandBuilder[config]) {
			name := root.Values.String(&root.Config.Name)
			root.Flags.ValueFor(name).Placement(cli.FlagPlacementSubtree).Long("name")
			root.Subcommands.Command(cli.Command[config]{
				Name: "serve",
				Declare: func(child *cli.CommandBuilder[config]) {
					legacy := child.Shortcut(cli.CommandShortcut{
						Name: "quick",
						Args: []string{"--name", "fast"},
					})
					current := child.Shortcut(cli.CommandShortcut{
						Name:       "quicker",
						AppendArgs: []string{"--name", "faster"},
					})
					legacy.ReplacedBy(current).Default(name, "legacy")
				},
			})
		},
	})
	root := clitest.RootCommand()
	quick := root.Shortcut("quick")
	quicker := root.Shortcut("quicker")
	clitest.Schema(app).Require(t,
		quick.ReplacedBy(quicker),
		quick.ShortcutDefaultCount(1),
		quick.ShortcutDefaultAt(0).Destination(root.Flag("--name").Value()),
		quick.ShortcutDefaultAt(0).ValueAt(0).Text("legacy"),
	)
}

const authoringSnapshotCommandmentName = "commandment"

type authoringSnapshotValueDepConfig struct {
	Value string
	Dep   string
}

type authoringSnapshotRootConfig struct{}

type authoringSnapshotForceNameChildConfig struct {
	Force bool
	Name  string
}

type authoringSnapshotNameAllConfig struct {
	Name string
	All  bool
}

type authoringSnapshotValuesConfig struct{ Values map[string]int }

type authoringSnapshotBaseComputedConfig struct{ Base, Computed string }

type authoringSnapshotDepFlagArgConfig struct{ Dep, Flag, Arg string }

type authoringSnapshotVerboseRootConfig struct{ Verbose bool }

type authoringSnapshotNameChildConfig struct{ Name string }

type authoringSnapshotOwnerConfig struct{}

type authoringSnapshotModeChildConfig struct {
	Mode   string
	Format string
	Before string
	After  string
}

type authoringSnapshotValueConfig struct{ Value string }

func TestSnapshotDefinitionMetadataDetachesRetainedPublicInput(t *testing.T) {
	t.Parallel()
	translations := cli.Translations{
		"hello": "bonjour",
	}
	bundles := []cli.TranslationBundle{{
		Locale:       "fr",
		Translations: translations,
	}}
	examples := []cli.Example{{
		Description: "show",
		CommandLine: "tool show",
	}}
	order := []string{"one", "two"}
	definition := cli.Definition[struct{}]{
		Name: "tool",
		Description: cli.Description{
			Short: "short",
			Long:  "long",
		},
		Examples: examples,
		Release:  cli.Release.Ordered("two", order...),
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles:       bundles,
		},
	}
	app := compileMustCompilePublic(t, definition)
	examples[0].CommandLine = "mutated"
	translations["hello"] = "mutated"
	bundles[0].Locale = "de"
	order[0] = "mutated"

	root := clitest.RootCommand()
	clitest.Schema(app).Require(t,
		root.Name("tool"),
		root.SourceDescription(cli.Description{
			Short: "short",
			Long:  "long",
		}),
		root.SourceExamplesExactly(cli.Example{
			Description: "show",
			CommandLine: "tool show",
		}),
		clitest.LocalizationExactly(cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"hello": "bonjour",
				},
			}},
		}),
	)
	clitest.Reference(app, cli.ReferenceOptions{
		Release: "one",
		Locale:  "fr",
	}).Require(t,
		clitest.EffectiveLocale("fr"),
		clitest.EvaluatedRelease("one"),
	)
	clitest.Schema(app).Require(t,
		root.SourceExamplesExactly(cli.Example{
			Description: "show",
			CommandLine: "tool show",
		}),
		clitest.LocalizationExactly(cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"hello": "bonjour",
				},
			}},
		}),
	)
}

func TestArgumentAuthoringSnapshotsCompletionAndAcceptsSourceSensitivity(t *testing.T) {
	t.Parallel()
	type config = authoringSnapshotValueDepConfig
	var callbackCalls int
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "argument-completion",
		Declare: func(root *cli.CommandBuilder[config]) {
			dep := root.Values.String(&root.Config.Dep)
			root.Values.Default(dep, "ready")
			deps := []cli.Dependency{dep}
			sources := []cli.SourceOption{cli.Source.Key("original")}
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				argument := form.String(&form.Config.Value, "value").Sensitive()
				argument.CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
					callbackCalls++
					contribution := cli.CompletionContribution{
						Candidates: []cli.CompletionCandidate{{
							Value: "candidate",
						}},
					}
					return contribution, nil
				}, sources...).From(deps...)
			})
			sources[0] = cli.Source.Key("mutated")
			deps[0] = dep.OptionalDependency()
		},
	})
	root := clitest.RootCommand()
	term := root.ArgumentFormAt(0).Term("value")
	clitest.Schema(app).Require(t,
		term.SourceSensitive(),
		term.Completion().Dynamic(),
		term.Completion().DependenciesExactly(root.ValueAt(0)),
	)
	clitest.Complete(t.Context(), app, cli.CompletionRequest{
		Args: []string{""},
		Cursor: cli.CompletionCursor{
			Argument: 0,
		},
	}).Require(t, clitest.CompletionItem("candidate").Present())
	if callbackCalls != 1 {
		t.Fatalf("completion calls = %d, want 1", callbackCalls)
	}
}

func TestCommandDefaultBelongsToExactParentChildDeclaration(t *testing.T) {
	t.Parallel()
	reusable := cli.Command[struct{}]{
		Name: "leaf",
	}
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "left",
				Declare: func(left *cli.CommandBuilder[struct{}]) {
					left.Subcommands.Command(reusable).Default()
				},
			})
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "right",
				Declare: func(right *cli.CommandBuilder[struct{}]) {
					right.Subcommands.Command(reusable)
				},
			})
		},
	})
	root := clitest.RootCommand()
	left := root.Command("left")
	right := root.Command("right")
	clitest.Schema(app).Require(t,
		left.DefaultsExactly(left.DefaultAt(0)),
		left.DefaultAt(0).Target(left.Command("leaf")),
		right.DefaultsExactly(),
	)
}

func TestCommandDefaultPreservesProfileOfStagedChild(t *testing.T) {
	t.Parallel()
	type config struct{ Profile string }
	domain := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
		values.Value("active", "active")
	})
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			profile := root.Values.Value(&root.Config.Profile, domain)
			root.Values.Default(profile, "active")
			root.Profiles.Switch(profile, func(cases *cli.GrammarProfileCases[config, string]) {
				cases.Case("active", func(active *cli.CommandBuilder[config]) {
					active.Subcommands.Command(cli.Command[struct{}]{
						Name: "child",
					}).Default()
				})
			})
		},
	})
	schema := clitest.Schema(app).Schema()
	root := schema.Commands[schema.Root]
	if root.Profiles == nil || len(root.Profiles.Cases) != 1 {
		t.Fatalf("root grammar profiles = %#v, want one case", root.Profiles)
	}
	profile := root.Profiles.Cases[0]
	if got := root.Profiles.Profiles[profile].Value; got != "active" {
		t.Fatalf("profile value = %q, want active", got)
	}
	if len(root.CommandDefaults) != 1 {
		t.Fatalf("root command defaults = %v, want one", root.CommandDefaults)
	}
	defaultRecord := schema.CommandDefaults[root.CommandDefaults[0]]
	if defaultRecord.Profile != profile {
		t.Fatalf("default profile = %v, want %v", defaultRecord.Profile, profile)
	}
}

func TestSubcommandAuthoringSnapshotsAliasesAndLowersInheritedGrammar(t *testing.T) {
	t.Parallel()
	type rootConfig = authoringSnapshotRootConfig
	type childConfig = authoringSnapshotForceNameChildConfig
	aliases := []cli.CommandAlias{{
		Name: "c",
	}, {
		Name: "kid",
	}}
	definition := cli.Definition[rootConfig]{
		Name: "command-snapshot",
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Dialect.ResponseFiles(true)
			root.Subcommands.Command(cli.Command[childConfig]{
				Name:    "child",
				Aliases: aliases,
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					child.Dialect.DashlessFlags(cli.DashlessFlagModeInitialCluster)
					child.Flags.Bool(&child.Config.Force).Long("force")
					child.Arguments.Form(func(form *cli.ArgumentFormBuilder[childConfig]) {
						form.String(&form.Config.Name, "name")
					})
				},
			}).Default()
			aliases[0].Name = "mutated"
		},
	}
	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	child := root.Command("child")
	childForm := child.ArgumentFormAt(0)
	clitest.Schema(app).Require(t,
		child.EffectiveDialect(cli.CommandDialect{
			FlagOrder:            cli.FlagOrderBeforeArguments,
			ShortFlagClustering:  cli.ShortFlagClusteringDisabled,
			FlagMatching:         cli.NameMatchingExact,
			CommandMatching:      cli.NameMatchingExact,
			FlagNormalization:    cli.NameNormalizationExact,
			CommandNormalization: cli.NameNormalizationExact,
			DashlessFlags:        cli.DashlessFlagModeInitialCluster,
			EndOfOptions:         []string{"--"},
			ResponseFiles:        true,
		}),
		root.EntriesExactly(child.CanonicalEntry(), root.Alias("c"), root.Alias("kid")),
		root.DefaultsExactly(root.DefaultAt(0)),
		root.DefaultAt(0).Target(child),
		child.FlagsExactly(child.Flag("--force")),
		child.ArgumentFormsExactly(childForm),
		childForm.TermsExactly(childForm.Term("name")),
	)
}

func TestAcceptedAuthoringLowersThroughDraftIntoDetachedStaticGrammar(t *testing.T) {
	t.Parallel()
	type config = authoringSnapshotNameAllConfig
	app := compileMustCompilePublic(t, cli.Definition[config]{
		Name: "detached-grammar",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.FlagOrder(cli.FlagOrderInterspersed)
			root.Dialect.FlagNormalization(cli.NameNormalizationASCIICaseFold)
			root.Dialect.DashlessFlags(cli.DashlessFlagModeInitialCluster)
			root.Dialect.ResponseFiles(true)
			root.Dialect.DefaultFlagPlacement(cli.FlagPlacementSubtree)
			name := root.Values.String(&root.Config.Name)
			root.Flags.ValueFor(name).Occurrences(cli.Count.Between(1, 2)).Placement(cli.FlagPlacementCommand).
				Long("name").
				ValueForms(cli.FlagValueFormAssigned).
				Sensitive()
			root.Flags.Bool(&root.Config.All).Short('a')
			root.Values.Environment(name, "COMMANDMENT_NAME")
		},
	})
	root := clitest.RootCommand()
	expectedDialect := cli.CommandDialect{
		FlagOrder:            cli.FlagOrderInterspersed,
		ShortFlagClustering:  cli.ShortFlagClusteringDisabled,
		FlagMatching:         cli.NameMatchingExact,
		CommandMatching:      cli.NameMatchingExact,
		FlagNormalization:    cli.NameNormalizationASCIICaseFold,
		CommandNormalization: cli.NameNormalizationExact,
		DashlessFlags:        cli.DashlessFlagModeInitialCluster,
		EndOfOptions:         []string{"--"},
		ResponseFiles:        true,
	}
	firstProof := clitest.Schema(app)
	firstProof.Require(t,
		root.EffectiveDialect(expectedDialect),
		root.FlagsExactly(root.Flag("--name"), root.Flag("-a")),
		root.EnvironmentInputsExactly(root.Environment("COMMANDMENT_NAME")),
	)
	first := firstProof.Schema()
	firstRoot := first.Commands[first.Root]
	firstRoot.Flags[0] = 0
	first.Commands[first.Root] = firstRoot
	clitest.Schema(app).Require(t,
		root.EffectiveDialect(expectedDialect),
		root.FlagsExactly(root.Flag("--name"), root.Flag("-a")),
		root.EnvironmentInputsExactly(root.Environment("COMMANDMENT_NAME")),
	)
}

func TestKeyValueLanguageSnapshotsSyntaxConfiguration(t *testing.T) {
	t.Parallel()
	type config = authoringSnapshotValuesConfig
	language := cli.Value.Keyed(cli.Value.String(), cli.Value.Int()).SeparatedBy(":")
	keyOnly := language.KeyOnly(7)
	definition := cli.Definition[config]{
		Name: "keyed",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			d := root.Values.Keyed(&root.Config.Values, keyOnly)
			root.Flags.Keyed(d).Long("label")
		},
	}
	app := clitest.Compile(t, definition)
	value := clitest.RootCommand().Flag("--label").Value()
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--label", "a:3", "--label", "b"},
	}).Require(t,
		clitest.InspectionReady(),
		value.Resolved(),
		value.CandidateAt(0).EntryAt(0).Key().Text("a"),
		value.CandidateAt(0).EntryAt(0).Value().Text("3"),
		value.CandidateAt(1).EntryAt(0).Key().Text("b"),
		value.CandidateAt(1).EntryAt(0).Value().Text("7"),
	)
	clitest.CompileFails(t, cli.Definition[config]{
		Name: "zero-keyed",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Values.Keyed(&root.Config.Values, cli.KeyValueLanguage[string, int]{})
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestComputationExpressionsSnapshotCallerOwnedSlices(t *testing.T) {
	t.Parallel()
	type config = authoringSnapshotBaseComputedConfig
	options := []cli.SourceOption{cli.Source.Key("before")}
	var base cli.Destination[config, string]
	var computationCalls int
	definition := cli.Definition[config]{
		Name: "computation-snapshot",
		Declare: func(root *cli.CommandBuilder[config]) {
			base = root.Values.String(&root.Config.Base)
			root.Values.Default(base, "base")
			deps := []cli.Dependency{base}
			comp := cli.Value.ComputedBy(func(_ context.Context, _ *config, d cli.Dependencies) (string, error) {
				computationCalls++
				v, _ := d.Get(base)
				return v + "-computed", nil
			}, options...).From(deps...)
			root.Values.Derive(&root.Config.Computed, comp)
			options[0] = cli.Source.Key("after")
			deps[0] = base.OptionalDependency()
		},
		Run: func(context.Context, *config) error { return nil },
	}
	app := compileMustCompilePublic(t, definition)
	if err := app.Run(t.Context(), cli.RunRequest{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if computationCalls != 1 {
		t.Fatalf("computation calls=%d", computationCalls)
	}
}

func TestTokenLanguageNormalizationValidatesRequiredShapeAndSnapshotsSyntax(t *testing.T) {
	t.Parallel()
	type config = authoringSnapshotValueConfig
	syntax := []string{"WORD"}
	language := cli.TokenLanguage[string]{
		Name:   "word",
		Count:  cli.Count.Exactly(1),
		Syntax: syntax,
		Parse:  func(_ context.Context, tokens []string) (string, error) { return tokens[0], nil },
		Format: func(v string) []string { return []string{v} },
	}
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "token-language",
		Declare: func(root *cli.CommandBuilder[config]) {
			d := root.Values.Value(&root.Config.Value, cli.Value.String())
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) { form.Tokens(d, "value", language) })
			syntax[0] = "MUTATED"
		},
	})
	clitest.Schema(app).Require(t,
		clitest.RootCommand().ArgumentFormAt(0).Term("value").TokenLanguageFragmentsExactly("WORD"),
	)
	bad := cli.TokenLanguage[string]{
		Name:  "bad",
		Count: cli.Count.Exactly(1),
	}
	clitest.CompileFails(t, cli.Definition[config]{
		Name: "bad-token",
		Declare: func(root *cli.CommandBuilder[config]) {
			d := root.Values.String(&root.Config.Value)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) { form.Tokens(d, "value", bad) })
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestCompletionExpressionsSnapshotDependenciesWhenExtended(t *testing.T) {
	t.Parallel()
	type config = authoringSnapshotDepFlagArgConfig
	var calls int
	app := compileMustCompilePublic(t, cli.Definition[config]{
		Name: "completion-deps",
		Declare: func(root *cli.CommandBuilder[config]) {
			dep := root.Values.String(&root.Config.Dep)
			root.Values.Default(dep, "ready")
			deps := []cli.Dependency{dep.OptionalDependency()}
			root.Flags.String(&root.Config.Flag).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				calls++
				contribution := cli.CompletionContribution{
					Candidates: []cli.CompletionCandidate{{
						Value: "f",
					}},
				}
				return contribution, nil
			}, cli.Source.Key("before")).From(deps...).
				Long("flag")
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.String(
					&form.Config.Arg,
					"arg",
				).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
					calls++
					contribution := cli.CompletionContribution{
						Candidates: []cli.CompletionCandidate{{
							Value: "a",
						}},
					}
					return contribution, nil
				}, cli.Source.Key("before")).
					From(deps...)
			})
			deps[0] = dep
		},
	})
	root := clitest.RootCommand()
	dep := root.ValueAt(0)
	clitest.Schema(app).Require(t,
		root.Flag("--flag").Completion().Dynamic(),
		root.Flag("--flag").Completion().DependenciesExactly(dep),
		root.ArgumentFormAt(0).Term("arg").Completion().Dynamic(),
		root.ArgumentFormAt(0).Term("arg").Completion().DependenciesExactly(dep),
	)
	clitest.Complete(t.Context(), app, cli.CompletionRequest{
		Args: []string{"--flag", ""},
		Cursor: cli.CompletionCursor{
			Argument: 1,
		},
	}).Require(t,
		clitest.CompletionItem("f").Present(),
	)
	if calls == 0 {
		t.Fatal("completion callback not retained")
	}
}

func TestValueDirectiveStagesAfterDestinationAndSnapshotsCallerSlices(t *testing.T) {
	t.Parallel()
	type config = authoringSnapshotValueConfig
	policies := []any{cli.Value.KeepFirst()}
	_ = policies
	definition := cli.Definition[config]{
		Name: "directive",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			d := root.Values.String(&root.Config.Value)
			root.Values.ResolveWith(d, cli.Value.KeepFirst())
			root.Values.Default(d, "first")
			root.Values.Default(d, "second")
		},
	}
	app := clitest.Compile(t, definition)
	value := clitest.RootCommand().ValueAt(0)
	c0, c1 := value.CandidateAt(0), value.CandidateAt(1)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		value.CandidatesExactly(c0, c1),
	)
}

func TestDefinitionAuthoringEntersOneRootTransactionAndSnapshotsRetainedInput(t *testing.T) {
	t.Parallel()
	type rootConfig = authoringSnapshotVerboseRootConfig
	type childConfig = authoringSnapshotNameChildConfig
	middlewareCalls := 0
	middleware := []cli.Middleware[rootConfig]{func(ctx context.Context, _ *rootConfig, next cli.MiddlewareNext) error {
		middlewareCalls++
		return next(ctx)
	}}
	examples := []cli.Example{{
		Description: "before",
		CommandLine: "commandment child",
	}}
	definition := cli.Definition[rootConfig]{
		Name: authoringSnapshotCommandmentName,
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Examples:   examples,
		Middleware: middleware,
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Flags.Bool(&root.Config.Verbose).Long("verbose")
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "child",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					child.Arguments.Form(
						func(form *cli.ArgumentFormBuilder[childConfig]) { form.String(&form.Config.Name, "name") },
					)
				},
			})
		},
		Run: func(context.Context, *rootConfig) error { return nil },
	}
	app := compileMustCompilePublic(t, definition)
	examples[0].Description = "mutated"
	middleware[0] = nil
	schema := app.Schema()
	if schema.Commands[schema.Root].Name != authoringSnapshotCommandmentName ||
		schema.Commands[schema.Root].Examples[0].Description != "before" {
		t.Fatalf("metadata snapshot = %#v", schema.Commands[schema.Root])
	}
	if len(schema.CommandOrder) != 2 || len(schema.Commands[schema.Root].Flags) != 1 ||
		len(schema.Commands[schema.CommandOrder[1]].ArgumentForms) != 1 {
		t.Fatalf("root transaction topology = %#v", schema.Commands)
	}
	if err := app.Run(t.Context(), cli.RunRequest{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if middlewareCalls != 1 {
		t.Fatalf("middleware calls = %d, want 1", middlewareCalls)
	}
}

func TestProviderBatchCaptureUsesCanonicalAuthoringAndSnapshotsContributions(t *testing.T) {
	t.Parallel()

	const frenchExample = "exemple" //nolint:misspell
	type ownerConfig = authoringSnapshotOwnerConfig
	type childConfig = authoringSnapshotModeChildConfig

	command := cli.Command[childConfig]{
		Name: "child",
		Aliases: []cli.CommandAlias{{
			Name: "kid",
		}},
		Examples: []cli.Example{{
			Description: "example",
			CommandLine: "app child",
		}},
	}
	shortcut := cli.CommandShortcut{
		Name:       "quick",
		Args:       []string{"before"},
		AppendArgs: []string{"after"},
	}
	localization := cli.Localization{
		DefaultLocale: "en",
		Bundles: []cli.TranslationBundle{{
			Locale: "fr",
			Translations: cli.Translations{
				"example": frenchExample,
			},
		}},
	}

	var mode cli.Destination[childConfig, string]
	declareChild := func(child *cli.CommandBuilder[childConfig]) {
		mode = child.Values.String(&child.Config.Mode)
		child.Flags.String(&child.Config.Format).Long("format")
		child.Arguments.Form(func(form *cli.ArgumentFormBuilder[childConfig]) {
			form.String(&form.Config.Before, "before")
			form.String(&form.Config.After, "after")
		})
	}
	provide := func(_ context.Context, batch *cli.CommandBatch[ownerConfig]) error {
		command.Declare = declareChild
		child := batch.Command(command, cli.Source.Key("command-source"))
		batch.Alias(child, cli.CommandAlias{
			Name: "c",
		}, cli.Source.Location("alias-source"))
		binding := batch.Shortcut(cli.CommandPath{"child"}, shortcut, cli.Source.Key("shortcut-source"))
		binding.Default(mode, "safe", cli.Source.Key("typed-default"))
		binding.DefaultFlag("--format", []string{"fast"}, cli.Source.Location("flag-default"))
		batch.Localization(localization)
		command.Aliases[0].Name = "mutated"
		command.Examples[0].Description = "mutated"
		shortcut.Args[0] = "mutated"
		shortcut.AppendArgs[0] = "mutated"
		localization.Bundles[0].Translations["example"] = "mutated"
		return nil
	}
	declare := func(root *cli.CommandBuilder[ownerConfig]) {
		root.Subcommands.CommandsFrom(provide)
	}
	definition := cli.Definition[ownerConfig]{
		Name:    "tool",
		Declare: declare,
	}
	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	child := root.Command("child")
	aliasRequest := cli.InspectionRequest{
		Args:   []string{"c"},
		Locale: "fr",
	}
	shortcutRequest := cli.InspectionRequest{
		Args: []string{"quick"},
	}

	clitest.Inspect(t.Context(), app, aliasRequest).Require(t,
		clitest.InspectionReady(),
		clitest.DiagnosticsExactly(),
		clitest.SelectionPathExactly(root, child),
		child.PresentedExamplesExactly(cli.Example{
			Description: frenchExample,
			CommandLine: "app child",
		}),
	)
	clitest.Inspect(t.Context(), app, shortcutRequest).Require(t,
		clitest.InspectionReady(),
		clitest.SelectionPathExactly(root, child),
	)
}

type authoringStagingBCIUSSSSeqPairConfig struct {
	B    bool
	C    int
	I    int
	U    uint
	S    string
	SS   []string
	Seq  []string
	Pair string
}

type authoringStagingGateEnvSelectedDerivedConfig struct{ Gate, Env, Selected, Derived string }

func TestStagingProjectsDeclaredSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringStagingBCIUSSSSeqPairConfig
	language := cli.TokenLanguage[string]{
		Name:   "pair",
		Count:  cli.Count.Exactly(2),
		Parse:  func(context.Context, []string) (string, error) { return "pair", nil },
		Format: func(string) []string { return []string{"a", "b"} },
	}
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.FlagOrder(cli.FlagOrderInterspersed)
			root.Dialect.ShortFlagClustering(cli.ShortFlagClusteringEnabled)
			root.Dialect.FlagMatching(cli.NameMatchingUniquePrefix)
			root.Dialect.CommandMatching(cli.NameMatchingUniquePrefix)
			root.Dialect.FlagNormalization(cli.NameNormalizationASCIICaseFold)
			root.Dialect.CommandNormalization(cli.NameNormalizationASCIICaseFold)
			root.Dialect.DashlessFlags(cli.DashlessFlagModeInitialCluster)
			root.Dialect.EndOfOptions("--", "++")
			root.Dialect.ResponseFiles(true)
			root.Dialect.DefaultFlagPlacement(cli.FlagPlacementSubtree)
			boolFlag := root.Flags.Bool(&root.Config.B).Description("bool").Required()
			boolFlag.Long("bool")
			boolFlag.Short('b')
			root.Flags.Count(&root.Config.C).Long("count")
			root.Flags.Int(&root.Config.I).HyphenValues(cli.HyphenValueNegativeNumbers).
				Long("int").
				ImplicitValue(1)
			root.Flags.Uint(&root.Config.U).Long("uint")
			root.Flags.String(&root.Config.S).ValueName("TEXT").
				Long("string").
				FixedValue("x").
				Sensitive().
				Hidden()
			root.Flags.Strings(&root.Config.SS).ValuesPerOccurrence(cli.Count.Between(1, 2)).DelimitedBy(",").TerminatedBy(";").
				Long("strings").
				ValueForms(cli.FlagValueFormAssigned)
			seq := root.Values.Sequence(&root.Config.Seq, cli.Value.String())
			root.Flags.Sequence(seq).Occurrences(cli.Count.Between(1, 2)).Long("seq")
			pair := root.Values.String(&root.Config.Pair)
			root.Flags.Tokens(pair, language).Long("pair")
		},
	})
	root := clitest.RootCommand()
	boolFlag := root.Flag("--bool")
	countFlag := root.Flag("--count")
	intFlag := root.Flag("--int")
	uintFlag := root.Flag("--uint")
	stringFlag := root.Flag("--string")
	stringsFlag := root.Flag("--strings")
	seqFlag := root.Flag("--seq")
	pairFlag := root.Flag("--pair")
	clitest.Schema(app).Require(t,
		root.EffectiveDialect(cli.CommandDialect{
			FlagOrder:            cli.FlagOrderInterspersed,
			ShortFlagClustering:  cli.ShortFlagClusteringEnabled,
			FlagMatching:         cli.NameMatchingUniquePrefix,
			CommandMatching:      cli.NameMatchingUniquePrefix,
			FlagNormalization:    cli.NameNormalizationASCIICaseFold,
			CommandNormalization: cli.NameNormalizationASCIICaseFold,
			DashlessFlags:        cli.DashlessFlagModeInitialCluster,
			EndOfOptions:         []string{"--", "++"},
			ResponseFiles:        true,
		}),
		root.FlagsExactly(boolFlag, countFlag, intFlag, uintFlag, stringFlag, stringsFlag, seqFlag, pairFlag),
		boolFlag.Required(),
		boolFlag.Description("bool"),
		boolFlag.SpellingsExactly(boolFlag.Spelling("--bool"), boolFlag.Spelling("-b")),
		stringFlag.ValueName("TEXT"),
		stringFlag.Spelling("--string").Hidden(),
		stringFlag.Spelling("--string").Sensitive(),
		stringFlag.Spelling("--string").FixedValue().Sensitive(),
		stringFlag.Spelling("--string").FixedValue().Redacted(),
		stringsFlag.ValuesPerOccurrence(cli.Count.Between(1, 2)),
		stringsFlag.Spelling("--strings").ValueFormsExactly(cli.FlagValueFormAssigned),
		seqFlag.Occurrences(cli.Count.Between(1, 2)),
		pairFlag.Value().Type(cli.ValueType{
			Kind: cli.ValueKindString,
		}),
	)
}

func TestStagingProjectsEverySemanticCarrier(t *testing.T) {
	t.Parallel()
	type config = authoringStagingGateEnvSelectedDerivedConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: func(root *cli.CommandBuilder[config]) {
			gate := root.Values.String(&root.Config.Gate)
			selected := root.Values.String(&root.Config.Selected)
			root.Values.Default(gate, "open")
			root.Values.Derive(
				&root.Config.Derived,
				cli.Value.ComputedBy(func(_ context.Context, _ *config, deps cli.Dependencies) (string, error) {
					v, _ := deps.Get(gate)
					return v, nil
				}).From(gate),
			)
			root.AvailableWhen(root.When.Equal(gate, "open"))
			root.Values.Environment(gate, "GATE").Release(cli.ReleaseLifecycle{
				IntroducedIn: "1",
			})
			root.Subcommands.Command(cli.Command[config]{
				Name: "shortcuts",
				Declare: func(child *cli.CommandBuilder[config]) {
					first := child.Shortcut(cli.CommandShortcut{
						Name: "first",
						Args: []string{"fixed"},
					}).Default(selected, "shortcut")
					second := child.Shortcut(cli.CommandShortcut{
						Name:       "second",
						AppendArgs: []string{"tail"},
					})
					first.ReplacedBy(second)
				},
			})
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Command(cli.Command[struct{}]{
					Name: "dynamic",
				})
				return nil
			}, gate.OptionalDependency())
			form := root.Arguments.Form(func(b *cli.ArgumentFormBuilder[config]) {
				b.Literal("choose").EndOfOptions().Release(cli.ReleaseLifecycle{
					IntroducedIn: "1",
				})
			}).Release(cli.ReleaseLifecycle{
				IntroducedIn: "1",
			})
			form.Selects(selected, "chosen")
		},
	}
	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	gate := root.Environment("GATE").Value()
	selected := root.ValueAt(1)
	derived := root.ValueAt(2)
	condition := root.ConditionAt(0)
	environment := root.Environment("GATE")
	form := root.ArgumentFormAt(0)
	term := form.Literal("choose")
	provider := root.ProviderAt(0)
	first := root.Shortcut("first")
	second := root.Shortcut("second")
	clitest.Schema(app).Require(t,
		root.ConditionsExactly(condition),
		condition.Effect(cli.ConditionEffectAvailability),
		condition.Expression().Kind(cli.ConditionExpressionEqual),
		condition.Expression().Destination(gate),
		condition.Expression().OperandCount(1),
		condition.Expression().OperandAt(0).Text("open"),
		root.EnvironmentInputsExactly(environment),
		environment.ReleaseLifecycle(cli.ReleaseLifecycle{
			IntroducedIn: "1",
		}),
		root.ArgumentFormsExactly(form),
		form.ReleaseLifecycle(cli.ReleaseLifecycle{
			IntroducedIn: "1",
		}),
		form.TermsExactly(term),
		term.EndOfOptions(),
		term.ReleaseLifecycle(cli.ReleaseLifecycle{
			IntroducedIn: "1",
		}),
		root.ProvidersExactly(provider),
		provider.DependenciesExactly(gate),
		first.ShortcutDefaultCount(1),
		first.ShortcutDefaultAt(0).Destination(selected),
		first.ShortcutDefaultAt(0).ValueCount(1),
		first.ShortcutDefaultAt(0).ValueAt(0).Text("shortcut"),
		first.ReplacedBy(second),
		derived.Kind(cli.ValueDestinationDerived),
		derived.DependenciesExactly(gate),
	)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"choose"},
	}).Require(t,
		selected.Resolved(),
		selected.Text("chosen"),
		derived.Resolved(),
		derived.NoRepresentation(),
	)
}

type compileBuilderMisuseConfig struct {
	Value string
}

func compileMustCompilePublic[C any](t testing.TB, definition cli.Definition[C]) cli.Application {
	t.Helper()
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return app
}
