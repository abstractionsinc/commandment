package conformance

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

type authoringAcceptanceNameVerboseConfig struct {
	Name    string
	Verbose bool
}

type authoringAcceptanceNameRestConfig struct {
	Name string
	Rest []string
}

type authoringAcceptanceNumberConfig struct{ Number int }

type authoringAcceptanceFlagArgIntConfig struct {
	Flag int
	Arg  int
}

type authoringAcceptanceValueStringConfig struct{ Value string }

type authoringAcceptanceCountNameConfig struct {
	Count int
	Name  string
}

type authoringAcceptanceFirstSecondConfig struct {
	First  string
	Second string
}

type authoringAcceptanceValueConfig struct{ Value string }

func TestFlagAuthoringStagesExactDestinationSpellingAndModifierRelationships(t *testing.T) {
	t.Parallel()
	type config = authoringAcceptanceNameVerboseConfig
	forms := []cli.FlagValueForm{cli.FlagValueFormAssigned}
	sources := []cli.SourceOption{cli.Source.Key("fixed")}
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "flag-relationships",
		Declare: func(root *cli.CommandBuilder[config]) {
			name := root.Values.String(&root.Config.Name)
			root.Flags.ValueFor(name).Occurrences(cli.Count.Optional()).Long("name").ValueForms(forms...)
			root.Flags.Bool(&root.Config.Verbose).Short('v').FixedValue(true, sources...).Sensitive()
			forms[0] = cli.FlagValueFormSeparate
			sources[0] = cli.Source.Key("mutated")
		},
	})
	root := clitest.RootCommand()
	name := root.Flag("--name").Spelling("--name")
	verbose := root.Flag("-v").Spelling("-v")
	clitest.Schema(app).Require(t,
		root.FlagsExactly(root.Flag("--name"), root.Flag("-v")),
		name.ValueFormsExactly(cli.FlagValueFormAssigned),
		verbose.FixedValue().Redacted(),
		verbose.Sensitive(),
	)
}

func TestFlagAuthoringRejectsMalformedSpellings(t *testing.T) {
	t.Parallel()
	type config = authoringAcceptanceValueConfig
	tests := []struct {
		name string
		add  func(cli.FlagDeclaration[config, string])
	}{
		{"empty long", func(flag cli.FlagDeclaration[config, string]) { flag.Long("") }},
		{"long starts dash", func(flag cli.FlagDeclaration[config, string]) { flag.Long("-x") }},
		{"long assigned", func(flag cli.FlagDeclaration[config, string]) { flag.Long("x=y") }},
		{"short nul", func(flag cli.FlagDeclaration[config, string]) { flag.Short(0) }},
		{"literal empty", func(flag cli.FlagDeclaration[config, string]) { flag.Literal("") }},
		{"literal nul", func(flag cli.FlagDeclaration[config, string]) { flag.Literal("x\x00y") }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			clitest.CompileFails(t, cli.Definition[config]{
				Name: "bad-spelling",
				Declare: func(root *cli.CommandBuilder[config]) {
					test.add(root.Flags.String(&root.Config.Value))
				},
			}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
		})
	}
}

func TestAcceptanceRetainsLiteralDefaultAndDoesNotGuessRootDialect(t *testing.T) {
	t.Parallel()
	type config = authoringAcceptanceValueConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "default",
		Declare: func(root *cli.CommandBuilder[config]) {
			d := root.Values.String(&root.Config.Value)
			root.Values.Default(d, "literal")
		},
	})
	root := clitest.RootCommand()
	clitest.Schema(app).Require(t,
		root.EffectiveDialect(cli.CommandDialect{
			FlagOrder:            cli.FlagOrderBeforeArguments,
			ShortFlagClustering:  cli.ShortFlagClusteringDisabled,
			FlagMatching:         cli.NameMatchingExact,
			CommandMatching:      cli.NameMatchingExact,
			FlagNormalization:    cli.NameNormalizationExact,
			CommandNormalization: cli.NameNormalizationExact,
			DashlessFlags:        cli.DashlessFlagModeDisabled,
			EndOfOptions:         []string{"--"},
		}),
		root.ValueAt(0).StaticDefault().Scalar().Text("literal"),
	)
}

func TestArgumentAuthoringLowersFormsTermsAndExactDestinations(t *testing.T) {
	t.Parallel()
	type config = authoringAcceptanceNameRestConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "args",
		Declare: func(root *cli.CommandBuilder[config]) {
			name := root.Values.String(&root.Config.Name)
			rest := root.Values.Sequence(&root.Config.Rest, cli.Value.String())
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.ValueFor(name, "name").Description("name value")
				form.Remainder(rest, "rest", cli.Count.ZeroOrMore())
			})
		},
	})
	root := clitest.RootCommand()
	form := root.ArgumentFormAt(0)
	name := form.Term("name")
	rest := form.Term("rest")
	clitest.Schema(app).Require(t,
		root.ArgumentFormsExactly(form),
		form.TermsExactly(name, rest),
		name.Kind(cli.ArgumentTermValue), name.Value().Exists(),
		rest.Kind(cli.ArgumentTermRemainder), rest.Value().Exists(),
	)
}

func TestAuthoringAcceptanceRetainsTypedNegativeNumberOwnership(t *testing.T) {
	t.Parallel()
	type config = authoringAcceptanceNumberConfig
	var got int
	app := compileMustCompilePublic(t, cli.Definition[config]{
		Name: "negative",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.Int(&root.Config.Number).HyphenValues(cli.HyphenValueNegativeNumbers).Long("number")
		},
		Run: func(_ context.Context, c *config) error { got = c.Number; return nil },
	})
	if err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"--number", "-7"},
	}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got != -7 {
		t.Fatalf("number = %d, want -7", got)
	}
}

func TestHyphenNegativeNumbersLowerOnlyForBuiltinSignedIntegerLanguage(t *testing.T) {
	t.Parallel()
	type intConfig = authoringAcceptanceFlagArgIntConfig
	clitest.Compile(t, cli.Definition[intConfig]{
		Name: "signed",
		Declare: func(root *cli.CommandBuilder[intConfig]) {
			root.Flags.Int(&root.Config.Flag).HyphenValues(cli.HyphenValueNegativeNumbers).Long("number")
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[intConfig]) {
				form.Int(&form.Config.Arg, "number").HyphenValues(cli.HyphenValueNegativeNumbers)
			})
		},
	})
	type stringConfig = authoringAcceptanceValueStringConfig
	clitest.CompileFails(t, cli.Definition[stringConfig]{
		Name: "string-hyphen",
		Declare: func(root *cli.CommandBuilder[stringConfig]) {
			root.Flags.String(&root.Config.Value).HyphenValues(cli.HyphenValueNegativeNumbers).Long("value")
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestAuthoringAcceptanceRetainsBuiltinAndCodecDestinationLanguages(t *testing.T) {
	t.Parallel()
	type config = authoringAcceptanceCountNameConfig
	codec := cli.ValueCodec[string]{
		Name:     "upper",
		Parse:    func(text string) (string, error) { return strings.ToUpper(text), nil },
		Format:   strings.ToLower,
		Snapshot: cli.Value.Assign[string](),
	}
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "languages",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			count := root.Values.Int(&root.Config.Count)
			name := root.Values.Value(&root.Config.Name, codec)
			root.Flags.ValueFor(count).Long("count")
			root.Flags.ValueFor(name).Long("name")
		},
	})
	root := clitest.RootCommand()
	count := root.Flag("--count").Value()
	name := root.Flag("--name").Value()
	clitest.Schema(app).Require(t,
		count.Type(cli.ValueType{
			Kind: cli.ValueKindInt,
		}),
		name.Type(cli.ValueType{
			Kind: cli.ValueKindCustom,
			Name: "upper",
		}),
	)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--count", "3", "--name", "mixed"},
	}).Require(t,
		clitest.InspectionReady(),
		count.Text("3"),
		name.Text("mixed"),
	)
}

var _ = errors.Is

func TestArgumentsRejectMixingDirectAndExplicitFormsAcrossFacetCopies(t *testing.T) {
	t.Parallel()
	type config = authoringAcceptanceFirstSecondConfig
	cases := []struct {
		name    string
		declare func(*cli.CommandBuilder[config])
	}{
		{
			name: "direct then explicit",
			declare: func(root *cli.CommandBuilder[config]) {
				copyOfArguments := root.Arguments
				copyOfArguments.String(&root.Config.First, "first")
				root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
					form.String(&form.Config.Second, "second")
				})
			},
		},
		{
			name: "explicit then direct",
			declare: func(root *cli.CommandBuilder[config]) {
				copyOfArguments := root.Arguments
				root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
					form.String(&form.Config.First, "first")
				})
				copyOfArguments.String(&root.Config.Second, "second")
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			clitest.CompileFails(t, cli.Definition[config]{
				Name:    "mixed-arguments",
				Declare: tc.declare,
			}).Require(t,
				clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present(),
			)
		})
	}
}

func TestSubcommandAuthoringRejectsMalformedCommandAtomically(t *testing.T) {
	t.Parallel()
	clitest.CompileFails(t, cli.Definition[struct{}]{
		Name: "malformed-child",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "",
			})
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestHelpEntriesDeclareFlagSpellings(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "actions",
		Help: cli.HelpConfig{ReplaceEntries: func(entries *cli.HelpActionEntries) {
			entries.Long("assist")
			entries.Short('h')
			entries.Literal("/?")
		}},
	})
	help := clitest.RootCommand().ActionEntries(cli.InvocationActionHelp)
	long := help.Long("--assist")
	short := help.Short('h')
	literal := help.Literal("/?")
	clitest.Schema(app).Require(t,
		help.Exactly(long, short, literal),
		long.Kind(cli.ActionEntryFlag), short.Kind(cli.ActionEntryFlag), literal.Kind(cli.ActionEntryFlag),
	)
}

func TestVersionEntriesDeclareFlagAndCommand(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name:    "actions",
		Release: cli.Release.SemVer("1.2.3"),
		Version: cli.VersionConfig{ReplaceEntries: func(entries *cli.VersionActionEntries) {
			entries.Long("build")
			entries.Command("build")
		}},
	})
	version := clitest.RootCommand().ActionEntries(cli.InvocationActionVersion)
	flag := version.Long("--build")
	command := version.Command("build")
	clitest.Schema(app).Require(t, version.Exactly(flag, command), flag.Kind(cli.ActionEntryFlag), command.Kind(cli.ActionEntryCommand))
}

func TestHelpEntryFluentMetadataPreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "actions",
		Help: cli.HelpConfig{ReplaceEntries: func(entries *cli.HelpActionEntries) {
			entries.Long("assist").
				Placement(cli.FlagPlacementSubtree).
				Description(cli.Description{Short: "custom help"}).
				Hidden()
			entries.Command("manual").Description(cli.Description{Short: "manual help"}).Hidden()
		}},
	})
	help := clitest.RootCommand().ActionEntries(cli.InvocationActionHelp)
	flag := help.Long("--assist")
	command := help.Command("manual")
	clitest.Schema(app).Require(t,
		help.Exactly(flag, command),
		flag.Placement(cli.FlagPlacementSubtree), flag.SourceDescription(cli.Description{Short: "custom help"}), flag.Hidden(),
		command.SourceDescription(cli.Description{Short: "manual help"}), command.Hidden(),
	)
}

func TestHelpEntryReplacementPreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "actions",
		Help: cli.HelpConfig{ReplaceEntries: func(entries *cli.HelpActionEntries) {
			legacy := entries.Long("help")
			current := entries.Long("assist")
			legacy.ReplacedBy(current)
		}},
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().ActionEntryAt(0).ReplacedBy(clitest.RootCommand().ActionEntryAt(1)))

	clitest.CompileFails(t, cli.Definition[struct{}]{
		Name: "actions",
		Help: cli.HelpConfig{ReplaceEntries: func(entries *cli.HelpActionEntries) {
			entry := entries.Long("help")
			entry.ReplacedBy(entry)
		}},
	})
}

func TestHelpEntryScopeAndReplacementCycleAreRejected(t *testing.T) {
	t.Parallel()
	var escaped func()
	clitest.Compile(t, cli.Definition[struct{}]{
		Name: "actions",
		Help: cli.HelpConfig{ReplaceEntries: func(entries *cli.HelpActionEntries) {
			first := entries.Long("one")
			second := entries.Long("two")
			escaped = func() { first.ReplacedBy(second) }
		}},
	})
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("retained action entry remained usable after callback")
			}
		}()
		escaped()
	}()

	clitest.CompileFails(t, cli.Definition[struct{}]{
		Name: "actions",
		Help: cli.HelpConfig{ReplaceEntries: func(entries *cli.HelpActionEntries) {
			one := entries.Long("one")
			two := entries.Long("two")
			three := entries.Long("three")
			one.ReplacedBy(two)
			two.ReplacedBy(three)
			three.ReplacedBy(one)
		}},
	})
}

func TestActionEntryInvalidDeclarationsAreRejected(t *testing.T) {
	t.Parallel()
	for _, declare := range []func(*cli.HelpActionEntries){
		func(entries *cli.HelpActionEntries) { entries.Long("") },
		func(entries *cli.HelpActionEntries) { entries.Long("bad name") },
		func(entries *cli.HelpActionEntries) { entries.Long("bad\nname") },
		func(entries *cli.HelpActionEntries) { entries.Short('-') },
		func(entries *cli.HelpActionEntries) { entries.Literal("") },
		func(entries *cli.HelpActionEntries) { entries.Long("ok").Placement(cli.FlagPlacement(255)) },
		func(entries *cli.HelpActionEntries) { entries.Command("") },
		func(entries *cli.HelpActionEntries) { entries.Command("bad\nname") },
	} {
		clitest.CompileFails(t, cli.Definition[struct{}]{
			Name: "actions",
			Help: cli.HelpConfig{ReplaceEntries: declare},
		})
	}
}

func TestVersionDerivesTextFromReleaseAndCommit(t *testing.T) {
	t.Parallel()
	for _, definition := range []cli.Definition[struct{}]{
		{Name: "actions", Version: cli.VersionConfig{Commit: "abc"}},
		{Name: "actions", Release: cli.Release.SemVer("1.2.3"), Version: cli.VersionConfig{CommitLength: -1}},
		{Name: "actions", Release: cli.Release.SemVer("1.2.3"), Version: cli.VersionConfig{CommitLength: 1}},
		{Name: "actions", Release: cli.Release.SemVer("1.2.3"), Version: cli.VersionConfig{Commit: "bad\ncommit"}},
	} {
		clitest.CompileFails(t, definition)
	}
	clitest.Compile(t, cli.Definition[struct{}]{
		Name:    "actions",
		Release: cli.Release.SemVer("1.2.3"),
	})
	clitest.Compile(t, cli.Definition[struct{}]{
		Name:    "actions",
		Release: cli.Release.SemVer("1.2.3"),
		Version: cli.VersionConfig{
			Commit:       "abcdef",
			CommitLength: 3,
		},
	})
}

func TestIntrinsicActionAcceptancePreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "actions",
		Help: cli.HelpConfig{ReplaceEntries: func(entries *cli.HelpActionEntries) {
			old := entries.Long("help").Description(cli.Description{Short: "old help"})
			current := entries.Command("assist").Description(cli.Description{Short: "new help"}).Hidden()
			old.ReplacedBy(current).Hidden()
		}},
	})
	help := clitest.RootCommand().ActionEntries(cli.InvocationActionHelp)
	old := help.Long("--help")
	current := help.Command("assist")
	clitest.Schema(app).Require(t,
		help.Exactly(old, current), old.SourceDescription(cli.Description{Short: "old help"}), old.ReplacedBy(current),
		old.Hidden(), current.SourceDescription(cli.Description{Short: "new help"}), current.Hidden(),
	)
}

type authoringArgumentTextNumberValuesTailPassPairConfig struct {
	Text   string
	Number int
	Values []string
	Tail   []string
	Pass   []string
	Pair   string
}

type authoringArgumentABCDEFGHPassConfig struct {
	A, B, C, D, E, F, G, H string
	Pass                   []string
}

func TestArgumentAuthoringProjectsDeclaredSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringArgumentTextNumberValuesTailPassPairConfig
	language := cli.TokenLanguage[string]{
		Name:   "pair",
		Count:  cli.Count.Exactly(2),
		Parse:  func(context.Context, []string) (string, error) { return "pair", nil },
		Format: func(string) []string { return []string{"left", "right"} },
	}
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: func(root *cli.CommandBuilder[config]) {
			text := root.Values.String(&root.Config.Text)
			values := root.Values.Sequence(&root.Config.Values, cli.Value.String())
			tail := root.Values.Sequence(&root.Config.Tail, cli.Value.String())
			pass := root.Values.Sequence(&root.Config.Pass, cli.Value.String())
			pair := root.Values.String(&root.Config.Pair)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.ValueFor(text, "text").
					Optional().
					Description("text").
					Hidden().
					Release(cli.ReleaseLifecycle{IntroducedIn: "1"}).
					CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
						return cli.CompletionContribution{}, nil
					}).
					From(text.OptionalDependency()).
					FallbackTo(cli.CompletionNativePaths)

				form.Int(&root.Config.Number, "number").HyphenValues(cli.HyphenValueNegativeNumbers)
				form.Sequence(values, "values", cli.Count.Between(1, 3)).Greedy()
				form.Tokens(
					pair,
					"pair",
					language,
				).CompleteTokensBy(func(context.Context, *config, cli.TokenLanguageCompletionRequest) (cli.CompletionContribution, error) {
					return cli.CompletionContribution{}, nil
				}).
					From(pair.OptionalDependency()).
					CombineWith(cli.CompletionNativeFiles)
				form.Literal("literal").EndOfOptions()
			})
			root.Arguments.Form(
				func(form *cli.ArgumentFormBuilder[config]) { form.Remainder(tail, "tail", cli.Count.ZeroOrMore()) },
			)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.PassThrough(
					pass,
					"pass",
					cli.Count.ZeroOrMore(),
				).DelegateCompletionBy(func(context.Context, *config, cli.CompletionDelegationRequest) (cli.CompletionDelegation, error) {
					return cli.CompletionDelegation{}, nil
				}).
					From(text)
			})
		},
	}
	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	form0, form1, form2 := root.ArgumentFormAt(0), root.ArgumentFormAt(1), root.ArgumentFormAt(2)
	clitest.Schema(app).Require(t,
		root.ArgumentFormsExactly(form0, form1, form2),
		form0.TermsExactly(form0.Term("text"), form0.Term("number"), form0.Term("values"), form0.Term("pair"), form0.Literal("literal")),
		form1.TermsExactly(form1.Term("tail")),
		form2.TermsExactly(form2.Term("pass")),
		form0.Term("text").Hidden(),
		form0.Term("text").Lifecycle(cli.LifecycleStateCurrent),
	)

	implicitDefinition := cli.Definition[config]{
		Name: "implicit",
		Declare: func(root *cli.CommandBuilder[config]) {
			values := root.Values.Sequence(&root.Config.Values, cli.Value.String())
			tail := root.Values.Sequence(&root.Config.Tail, cli.Value.String())
			pair := root.Values.String(&root.Config.Pair)
			root.Arguments.String(&root.Config.Text, "text")
			root.Arguments.Int(&root.Config.Number, "number")
			root.Arguments.Strings(&root.Config.Values, "strings", cli.Count.ZeroOrMore())
			root.Arguments.Sequence(values, "sequence", cli.Count.OneOrMore())
			root.Arguments.Tokens(pair, "tokens", language)
			root.Arguments.Literal("literal")
			root.Arguments.Remainder(tail, "tail", cli.Count.ZeroOrMore())
		},
	}
	implicit := clitest.Compile(t, implicitDefinition)
	implicitRoot := clitest.RootCommand()
	implicitForm := implicitRoot.ArgumentFormAt(0)
	clitest.Schema(implicit).Require(t,
		implicitRoot.ArgumentFormsExactly(implicitForm),
		implicitForm.TermsExactly(
			implicitForm.Term("text"),
			implicitForm.Term("number"),
			implicitForm.Term("strings"),
			implicitForm.Term(
				"sequence",
			),
			implicitForm.Term("tokens"),
			implicitForm.Literal("literal"),
			implicitForm.Term("tail"),
		),
	)

	implicitPass := clitest.Compile(t, cli.Definition[config]{
		Name: "implicit-pass",
		Declare: func(root *cli.CommandBuilder[config]) {
			pass := root.Values.Sequence(&root.Config.Pass, cli.Value.String())
			root.Arguments.PassThrough(pass, "pass", cli.Count.ZeroOrMore())
		},
	})
	passRoot := clitest.RootCommand()
	passForm := passRoot.ArgumentFormAt(0)
	clitest.Schema(implicitPass).Require(t,
		passRoot.ArgumentFormsExactly(passForm),
		passForm.TermsExactly(passForm.Term("pass")),
	)
}

func TestArgumentAuthoringProjectsOptionAndCompletionSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringArgumentABCDEFGHPassConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			a := root.Values.String(&root.Config.A)
			b := root.Values.String(&root.Config.B)
			c := root.Values.String(&root.Config.C)
			d := root.Values.String(&root.Config.D)
			e := root.Values.String(&root.Config.E)
			f := root.Values.String(&root.Config.F)
			g := root.Values.String(&root.Config.G)
			h := root.Values.String(&root.Config.H)
			pass := root.Values.Sequence(&root.Config.Pass, cli.Value.String())
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.ValueFor(a, "minimal").Optional().Minimal()
				form.Literal("stop").EndOfOptions()
				form.ValueFor(
					b,
					"fallback",
				).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
					return cli.CompletionContribution{}, nil
				}).
					From(a.OptionalDependency()).
					FallbackTo(cli.CompletionNativePaths)
				form.ValueFor(
					c,
					"combine",
				).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
					return cli.CompletionContribution{}, nil
				}).
					CombineWith(cli.CompletionNativeFiles)
				form.ValueFor(d, "directories").CompleteWith(cli.CompletionNativeDirectories)
				form.ValueFor(e, "executables").CompleteWith(cli.CompletionNativeExecutables)
				form.ValueFor(f, "users").CompleteWith(cli.CompletionNativeUsers)
				form.ValueFor(g, "hosts").CompleteWith(cli.CompletionNativeHosts)
				form.ValueFor(h, "environment").CompleteWith(cli.CompletionNativeEnvironmentVariables)
			})
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Literal("delegate")
				form.PassThrough(
					pass,
					"pass",
					cli.Count.ZeroOrMore(),
				).DelegateCompletionBy(func(context.Context, *config, cli.CompletionDelegationRequest) (cli.CompletionDelegation, error) {
					return cli.CompletionDelegation{}, nil
				}).
					From(a)
			})
		},
	}
	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	form := root.ArgumentFormAt(0)
	minimal := form.Term("minimal")
	clitest.Schema(app).Require(t,
		minimal.Minimal(),
		form.Literal("stop").EndOfOptions(),
		form.Term("fallback").Completion().Dynamic(),
		form.Term("fallback").Completion().Native(cli.CompletionNativePaths, cli.CompletionNativeOnNoMatch),
		form.Term("fallback").Completion().DependenciesExactly(minimal.Value()),
		form.Term("combine").Completion().Dynamic(),
		form.Term("combine").Completion().Native(cli.CompletionNativeFiles, cli.CompletionNativeAlways),
		form.Term("directories").Completion().Native(cli.CompletionNativeDirectories, cli.CompletionNativeAlways),
		form.Term("executables").Completion().Native(cli.CompletionNativeExecutables, cli.CompletionNativeAlways),
		form.Term("users").Completion().Native(cli.CompletionNativeUsers, cli.CompletionNativeAlways),
		form.Term("hosts").Completion().Native(cli.CompletionNativeHosts, cli.CompletionNativeAlways),
		form.Term("environment").
			Completion().
			Native(cli.CompletionNativeEnvironmentVariables, cli.CompletionNativeAlways),
		root.ArgumentFormAt(1).Term("pass").Completion().Delegated(),
		root.ArgumentFormAt(1).Term("pass").Completion().DependenciesExactly(minimal.Value()),
	)
}

type authoringCommandNameChildConfig struct{ Name string }

type authoringCommandValueChildConfig struct{ Value string }

type authoringCommandGateConfig struct{ Gate string }

func TestCommandAliasesShapeAndConstruction(t *testing.T) {
	t.Parallel()
	requireCommandAliases := func(cli.CommandAliases) {}
	requireCommandAliases(cli.Aliases("alias"))
	requireCommandAliases(cli.Command[struct{}]{}.Aliases)

	simple := cli.Aliases("first", "second")
	if len(simple) != 2 || simple[0].Name != "first" || simple[1].Name != "second" {
		t.Fatalf("simple aliases = %#v, want first then second", simple)
	}

	detailed := cli.CommandAliases{
		{
			Name: "first",
			Description: cli.Description{
				Short: "first alias",
			},
			Hidden: true,
		},
		{
			Name:    "second",
			Release: cli.ReleaseLifecycle{},
		},
	}
	if len(detailed) != 2 || detailed[0].Name != "first" || detailed[0].Description.Short != "first alias" ||
		!detailed[0].Hidden || detailed[1].Name != "second" {
		t.Fatalf("detailed aliases = %#v", detailed)
	}
}

func TestCommandBuilderAvailableWhenPreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringCommandGateConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			gate := root.Values.String(&root.Config.Gate)
			root.AvailableWhen(root.When.Resolved(gate))
		},
	})
	root := clitest.RootCommand()
	condition := root.ConditionAt(0)
	clitest.Schema(app).Require(t,
		root.ConditionsExactly(condition),
		condition.Effect(cli.ConditionEffectAvailability),
		condition.Expression().Kind(cli.ConditionExpressionResolved),
		condition.Expression().Destination(root.ValueAt(0)),
	)
}

func TestCommandBuilderVisibleWhenPreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringCommandGateConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			gate := root.Values.String(&root.Config.Gate)
			root.VisibleWhen(root.When.Resolved(gate))
		},
	})
	root := clitest.RootCommand()
	condition := root.ConditionAt(0)
	clitest.Schema(app).Require(t,
		root.ConditionsExactly(condition),
		condition.Effect(cli.ConditionEffectVisibility),
		condition.Expression().Kind(cli.ConditionExpressionResolved),
		condition.Expression().Destination(root.ValueAt(0)),
	)
}

func TestCommandDeclarationPreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	commands := cli.Commands{
		cli.Command[struct{}]{
			Name: "one",
		},
		cli.Command[struct{}]{
			Name: "two",
		},
	}
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.Commands(commands)
		},
	})
	clitest.Schema(app).Require(t, clitest.Command("one").Exists(), clitest.Command("two").Exists())
}

func TestCommandDeclareIntoPreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type childConfig = authoringCommandNameChildConfig
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.Commands(cli.Commands{
				cli.Command[childConfig]{
					Name: "child",
					Declare: func(child *cli.CommandBuilder[childConfig]) {
						child.Flags.String(&child.Config.Name).Long("name")
					},
				},
			})
		},
	})
	clitest.Schema(app).Require(t, clitest.Command("child").FlagAt(0).Exists())
}

func TestSubcommandsCommandsPreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.Commands(cli.Commands{
				nil,
				cli.Command[struct{}]{
					Name: "one",
				},
				cli.Command[struct{}]{
					Name: "two",
				},
			})
		},
	})
	clitest.Schema(app).Require(t, clitest.Command("one").Exists(), clitest.Command("two").Exists())
}

func TestCommandTargetPreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "child",
			})
		},
	})
	clitest.Schema(app).Require(t, clitest.Command("child").Exists())
}

func TestDeclareSubcommandPreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type childConfig = authoringCommandValueChildConfig
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name:            "tool",
		ShellCompletion: cli.ShellCompletion{Disabled: true},
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.Command(cli.Command[childConfig]{
				Name:    "child",
				Aliases: cli.Aliases("c"),
				Description: cli.Description{
					Short: "child command",
				},
				Hidden: true,
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					child.Flags.String(&child.Config.Value).Long("value")
				},
			})
		},
	})
	child := clitest.RootCommand().Command("child")
	clitest.Schema(app).Require(t,
		clitest.RootCommand().ChildrenExactly(child),
		child.Exists(),
		child.Hidden(),
		child.SourceDescription(cli.Description{
			Short: "child command",
		}),
		clitest.RootCommand().Alias("c").Exists(),
		child.FlagsExactly(child.Flag("--value")),
	)
}

func TestNilCommandBuilderAvailableWhenRejectsLifecycleMisuse(t *testing.T) {
	t.Parallel()

	var command *cli.CommandBuilder[struct{}]

	call := func() {
		command.AvailableWhen(cli.Condition{})
	}

	requireBuilderMisusePanic(t, call)
}

func TestNilCommandBuilderVisibleWhenRejectsLifecycleMisuse(t *testing.T) {
	t.Parallel()

	var command *cli.CommandBuilder[struct{}]

	call := func() {
		command.VisibleWhen(cli.Condition{})
	}

	requireBuilderMisusePanic(t, call)
}

func TestNilSubcommandsCommandsRejectsLifecycleMisuse(t *testing.T) {
	t.Parallel()

	var commands *cli.Subcommands[struct{}]

	call := func() {
		commands.Commands(nil)
	}

	requireBuilderMisusePanic(t, call)
}

type authoringDeclarationNameGateConfig struct{ Name, Gate string }

func TestDeclarationAuthoringBindsDestinationsAndConditionTargets(t *testing.T) {
	t.Parallel()
	type config = authoringDeclarationNameGateConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			name := root.Values.String(&root.Config.Name)
			gate := root.Values.String(&root.Config.Gate)
			root.Values.Default(gate, "yes")
			flag := root.Flags.ValueFor(name)
			flag.Long("name")
			_ = flag.Destination()
			flag.AvailableWhen(root.When.Equal(gate, "yes"))
			flag.VisibleWhen(root.When.NotEqual(gate, "no"))
			root.Arguments.Form(func(builder *cli.ArgumentFormBuilder[config]) {
				arg := builder.ValueFor(name, "name")
				_ = arg.Destination()
				arg.AvailableWhen(root.When.Equal(gate, "yes"))
				arg.VisibleWhen(root.When.NotEqual(gate, "no"))
			})
		},
	})
	root := clitest.RootCommand()
	flag := root.Flag("--name")
	argument := root.ArgumentFormAt(0).TermAt(0)
	shared := flag.Value()
	gate := root.ValueAt(1)
	availabilityFlag := root.ConditionAt(0)
	visibilityFlag := root.ConditionAt(1)
	availabilityArgument := root.ConditionAt(2)
	visibilityArgument := root.ConditionAt(3)
	clitest.Schema(app).Require(t,
		root.ValuesExactly(shared, gate),
		argument.Value().StaticDefault().DoesNotExist(),
		gate.StaticDefault().Scalar().Text("yes"),
		root.ConditionsExactly(availabilityFlag, visibilityFlag, availabilityArgument, visibilityArgument),
		availabilityFlag.Target(flag), availabilityFlag.Effect(cli.ConditionEffectAvailability),
		availabilityFlag.Expression().Kind(cli.ConditionExpressionEqual),
		availabilityFlag.Expression().Destination(gate),
		availabilityFlag.Expression().OperandAt(0).Text("yes"),
		visibilityFlag.Target(flag), visibilityFlag.Effect(cli.ConditionEffectVisibility),
		visibilityFlag.Expression().Kind(cli.ConditionExpressionNotEqual),
		visibilityFlag.Expression().Destination(gate),
		visibilityFlag.Expression().OperandAt(0).Text("no"),
		availabilityArgument.Target(argument), availabilityArgument.Effect(cli.ConditionEffectAvailability),
		availabilityArgument.Expression().Kind(cli.ConditionExpressionEqual),
		availabilityArgument.Expression().Destination(gate),
		availabilityArgument.Expression().OperandAt(0).Text("yes"),
		visibilityArgument.Target(argument), visibilityArgument.Effect(cli.ConditionEffectVisibility),
		visibilityArgument.Expression().Kind(cli.ConditionExpressionNotEqual),
		visibilityArgument.Expression().Destination(gate),
		visibilityArgument.Expression().OperandAt(0).Text("no"),
	)
}

func TestZeroFlagDeclarationAvailableWhenRejectsLifecycleMisuse(t *testing.T) {
	t.Parallel()

	var flag cli.FlagDeclaration[struct{}, int]

	call := func() {
		flag.AvailableWhen(cli.Condition{})
	}

	requireBuilderMisusePanic(t, call)
}

func TestZeroFlagDeclarationVisibleWhenRejectsLifecycleMisuse(t *testing.T) {
	t.Parallel()

	var flag cli.FlagDeclaration[struct{}, int]

	call := func() {
		flag.VisibleWhen(cli.Condition{})
	}

	requireBuilderMisusePanic(t, call)
}

func TestZeroArgumentDeclarationAvailableWhenRejectsLifecycleMisuse(t *testing.T) {
	t.Parallel()

	var argument cli.ArgumentDeclaration[struct{}, int]

	call := func() {
		argument.AvailableWhen(cli.Condition{})
	}

	requireBuilderMisusePanic(t, call)
}

func TestZeroArgumentDeclarationVisibleWhenRejectsLifecycleMisuse(t *testing.T) {
	t.Parallel()

	var argument cli.ArgumentDeclaration[struct{}, int]

	call := func() {
		argument.VisibleWhen(cli.Condition{})
	}

	requireBuilderMisusePanic(t, call)
}

func requireBuilderMisusePanic(t *testing.T, call func()) {
	t.Helper()

	panicked := false
	func() {
		defer func() {
			panicked = recover() != nil
		}()
		call()
	}()
	if !panicked {
		t.Fatal("builder lifecycle misuse did not panic")
	}
}

type authoringExecutionValueConfig struct{ Value string }

func TestSnapshotCommandExecutionPreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringExecutionValueConfig
	var order []string
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Middleware: []cli.Middleware[config]{
			func(ctx context.Context, cfg *config, next cli.MiddlewareNext) error {
				order = append(order, "before:"+cfg.Value)
				err := next(ctx)
				order = append(order, "after:"+cfg.Value)
				return err
			},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Value).Long("value")
		},
		Run: func(_ context.Context, cfg *config) error {
			order = append(order, "run:"+cfg.Value)
			return nil
		},
	})
	if err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"--value", "x"},
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	want := []string{"before:x", "run:x", "after:x"}
	if !slices.Equal(order, want) {
		t.Fatalf("execution order = %#v, want %#v", order, want)
	}

	clitest.CompileFails(t, cli.Definition[config]{
		Name:       "tool",
		Middleware: []cli.Middleware[config]{nil},
	})
}

func TestAuthoringCommandExecutionLowerPreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	called := false
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Run: func(context.Context, *struct{}) error {
			called = true
			return nil
		},
	})
	if err := app.Run(t.Context(), cli.RunRequest{}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !called {
		t.Fatal("accepted handler was not called")
	}
}

type authoringGrammarBCIUSCustomSSSeqPairConfig struct {
	B      bool
	C      int
	I      int
	U      uint
	S      string
	Custom string
	SS     []string
	Seq    []string
	Pair   string
}

func declareAuthoringGrammarFixture(root *cli.CommandBuilder[authoringGrammarBCIUSCustomSSSeqPairConfig]) {
	root.Flags.Bool(&root.Config.B).Short(0)
}

func TestAuthoringGrammarFlagDeclarationPreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringGrammarBCIUSCustomSSSeqPairConfig
	language := cli.TokenLanguage[string]{
		Name:   "pair",
		Count:  cli.Count.Exactly(2),
		Parse:  func(context.Context, []string) (string, error) { return "pair", nil },
		Format: func(string) []string { return []string{"a", "b"} },
	}
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
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
			root.Flags.String(&root.Config.S).ValueName("TEXT").Hidden().
				Long("string").
				FixedValue("x").
				Sensitive().
				Hidden()
			root.Flags.Value(&root.Config.Custom, cli.Value.String()).Occurrences(cli.Count.Optional()).Release(cli.ReleaseLifecycle{
				IntroducedIn: "1",
			}).
				Long("custom")
			root.Flags.Strings(&root.Config.SS).ValuesPerOccurrence(cli.Count.Between(1, 2)).DelimitedBy(",").TerminatedBy(";").
				Long("strings").
				ValueForms(cli.FlagValueFormAssigned)
			seq := root.Values.Sequence(&root.Config.Seq, cli.Value.String())
			root.Flags.Sequence(seq).Occurrences(cli.Count.Between(1, 2)).Long("seq")
			pair := root.Values.String(&root.Config.Pair)
			root.Flags.Tokens(pair, language).Long("pair")
		},
	}
	app := clitest.Compile(t, definition)
	clitest.Schema(app).Require(t, clitest.RootCommand().Exists())
	clitest.CompileFails(t, cli.Definition[config]{
		Name:    "bad",
		Declare: declareAuthoringGrammarFixture,
	})
}

type authoringLifecycleRootRootConfig struct{ Root string }

type authoringLifecycleChildChildConfig struct{ Child bool }

func TestSubcommandNestedBuilderSuspendsParentAndExpiresChild(t *testing.T) {
	t.Parallel()
	type rootConfig = authoringLifecycleRootRootConfig
	type childConfig = authoringLifecycleChildChildConfig
	var childBuilder *cli.CommandBuilder[childConfig]
	_, err := cli.Compile(cli.Definition[rootConfig]{
		Name: "parent-suspension",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "child",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					childBuilder = child
					root.Values.String(&root.Config.Root)
				},
			})
		},
	})
	compileRequireDiagnosticCode(t, err, cli.DiagnosticBuilderMisuse)

	_, err = cli.Compile(cli.Definition[rootConfig]{
		Name: "child-expiry",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[childConfig]{
				Name:    "child",
				Declare: func(child *cli.CommandBuilder[childConfig]) { childBuilder = child },
			})
		},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	panicked := false
	func() {
		defer func() { panicked = recover() != nil }()
		childBuilder.Flags.Bool(&childBuilder.Config.Child).Long("late")
	}()
	if !panicked {
		t.Fatal("nested command builder remained usable after callback")
	}
}

func compileHasDiagnosticCode(err error, code cli.DiagnosticCode) bool {
	for _, diagnostic := range cli.Diagnostics(err).Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func compileRequireDiagnosticCode(t testing.TB, err error, code cli.DiagnosticCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want diagnostic %q", code)
	}
	if !compileHasDiagnosticCode(err, code) {
		t.Fatalf("diagnostics = %#v, want code %q", cli.Diagnostics(err).Diagnostics, code)
	}
}

func compileMustCompilePublic[C any](t testing.TB, definition cli.Definition[C]) cli.Application {
	t.Helper()
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return app
}
