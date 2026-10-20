package conformance

import (
	"context"
	"errors"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

type authoringStorageNameNestConfig struct {
	Name string
	Nest authoringStorageCountNested
}

type authoringStorageNestedConfig struct{ Nested authoringStorageCountNested }

type authoringStorageCountNested struct{ Count int }

type authoringStorageValueConfig struct{ Value int }

func TestBindConfigurationStorageReplaysTypedFieldRelationship(t *testing.T) {
	t.Parallel()
	type config = authoringStorageNameNestConfig
	var got config
	app, err := cli.Compile(cli.Definition[config]{
		Name: "storage-owned",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.Int(&root.Config.Nest.Count).Long("count")
		},
		Run: func(_ context.Context, cfg *config) error {
			got = *cfg
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"--count", "7"},
	}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got.Nest.Count != 7 {
		t.Fatalf("nested configuration = %#v, want count 7", got)
	}
}

func TestBindConfigurationStorageRejectsExternalStorage(t *testing.T) {
	t.Parallel()
	type config = authoringStorageValueConfig
	external := new(int)
	clitest.CompileFails(t, cli.Definition[config]{
		Name: "storage-external",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.Int(external).Long("count")
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestBindConfigurationStoragePreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringStorageNestedConfig
	var got config
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.Int(&root.Config.Nested.Count).Long("count")
		},
		Run: func(_ context.Context, cfg *config) error {
			got = *cfg
			return nil
		},
	})
	if err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"--count", "9"},
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.Nested.Count != 9 {
		t.Fatalf("configuration = %#v, want nested count 9", got)
	}

	external := new(int)
	clitest.CompileFails(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.Int(external).Long("count")
		},
	})
}

type authoringTransactionValueFlagEventsConfig struct {
	Value  string
	Flag   bool
	Events <-chan cli.SignalEvent
}

type authoringTransactionValueConfig struct{ Value string }

type authoringTransactionProfileRootConfig struct{ Profile string }

type authoringTransactionProfileLocaleChildConfig struct {
	Profile string
	Locale  cli.Locale
}

type authoringTransactionNameConfig struct {
	Name string
}

func TestAuthoringTransactionRejectsInvalidStagedRecords(t *testing.T) {
	t.Parallel()
	type config = authoringTransactionValueFlagEventsConfig
	tests := []struct {
		name       string
		definition cli.Definition[config]
	}{
		{"embedded destination", cli.Definition[config]{
			Name:    "tool",
			Declare: func(root *cli.CommandBuilder[config]) { root.Flags.String(nil).Long("value") },
		}},
		{"argument options duplicate", cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Arguments.String(&root.Config.Value, "value").Optional().Optional()
			},
		}},
		{"argument options greedy minimal", cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Arguments.String(&root.Config.Value, "value").Greedy().Minimal()
			},
		}},
		{"literal end marker duplicate", cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Arguments.Literal("stop").EndOfOptions().EndOfOptions()
			},
		}},
		{"flag spelling", cli.Definition[config]{
			Name:    "tool",
			Declare: func(root *cli.CommandBuilder[config]) { root.Flags.Bool(&root.Config.Flag).Short(0) },
		}},
		{"flag spelling directive", cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Flags.String(&root.Config.Value).
					Long("value").
					ValueForms(cli.FlagValueFormAssigned, cli.FlagValueFormAssigned)
			},
		}},
		{"command declaration", cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Subcommands.Command(cli.Command[struct{}]{
					Name: "",
				})
			},
		}},
		{"dialect directive", cli.Definition[config]{
			Name:    "tool",
			Declare: func(root *cli.CommandBuilder[config]) { root.Dialect.FlagOrder(cli.FlagOrderUnknown) },
		}},
		{"resolution policies", cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				value := root.Values.String(&root.Config.Value)
				root.Values.ResolveWith(value)
			},
		}},
		{"signal policy", cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Signals.Policy(
					root.Signals.Match.Interrupt().During(cli.SignalPhaseResolution).Each(),
					root.Signals.Action.Abort(2),
				)
			},
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			proof := clitest.CompileFails(t, tc.definition)
			proof.Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
		})
	}
}

func TestAuthoringTransactionRejectsCrossTransactionReplacement(t *testing.T) {
	t.Parallel()
	type config = authoringTransactionValueConfig
	var foreign cli.FlagDeclaration[config, string]
	clitest.Compile(t, cli.Definition[config]{
		Name: "foreign",
		Declare: func(root *cli.CommandBuilder[config]) {
			foreign = root.Flags.String(&root.Config.Value)
			foreign.Long("foreign")
		},
	})
	proof := clitest.CompileFails(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			local := root.Flags.String(&root.Config.Value)
			local.Long("local")
			local.ReplacedBy(foreign)
		},
	})
	proof.Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestCommandDeclarationRejectsForeignParentStream(t *testing.T) {
	t.Parallel()
	type config struct{}
	var foreign cli.CommandDeclaration[config]
	clitest.Compile(t, cli.Definition[config]{
		Name: "foreign",
		Declare: func(root *cli.CommandBuilder[config]) {
			foreign = root.Subcommands.Command(cli.Command[struct{}]{Name: "foreign-child"})
		},
	})
	clitest.CompileFails(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			local := root.Subcommands.Command(cli.Command[struct{}]{Name: "local-child"})
			local.ReplacedBy(foreign)
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestCommandDeclarationRejectsRepeatedDefaultAuthoring(t *testing.T) {
	t.Parallel()
	t.Run("duplicate unconditional", func(t *testing.T) {
		clitest.CompileFails(t, cli.Definition[struct{}]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[struct{}]) {
				root.Subcommands.Command(cli.Command[struct{}]{Name: "child"}).Default().Default()
			},
		}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
	})
	t.Run("unconditional then conditional", func(t *testing.T) {
		type config struct{ Gate string }
		clitest.CompileFails(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				gate := root.Values.String(&root.Config.Gate)
				condition := root.When.Resolved(gate)
				root.Subcommands.Command(cli.Command[struct{}]{Name: "child"}).
					Default().DefaultWhen(condition)
			},
		}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
	})
}

func TestAuthoringTransactionRejectsDuplicateLifecycleDirective(t *testing.T) {
	t.Parallel()
	proof := clitest.CompileFails(t, cli.Definition[struct{}]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			form := root.Arguments.Form(func(b *cli.ArgumentFormBuilder[struct{}]) { b.Literal("x") })
			form.Release(cli.ReleaseLifecycle{
				IntroducedIn: "1",
			}).Release(cli.ReleaseLifecycle{
				IntroducedIn: "1",
			})
		},
	})
	proof.Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestAuthoringTransactionRejectsInvalidAccessTopology(t *testing.T) {
	t.Parallel()
	type rootConfig = authoringTransactionProfileRootConfig
	type childConfig = authoringTransactionProfileLocaleChildConfig
	t.Run("local value", func(t *testing.T) {
		t.Parallel()
		proof := clitest.CompileFails(t, cli.Definition[rootConfig]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[rootConfig]) {
				profile := root.Values.String(&root.Config.Profile)
				root.Access.Value(profile, &root.Config.Profile)
			},
		})
		proof.Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
	})
	t.Run("duplicate singleton", func(t *testing.T) {
		t.Parallel()
		proof := clitest.CompileFails(t, cli.Definition[rootConfig]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[rootConfig]) {
				root.Subcommands.Command(cli.Command[childConfig]{
					Name: "child",
					Declare: func(child *cli.CommandBuilder[childConfig]) {
						child.Access.Locale(&child.Config.Locale)
						child.Access.Locale(&child.Config.Locale)
					},
				})
			},
		})
		proof.Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
	})
	t.Run("overlapping binding", func(t *testing.T) {
		t.Parallel()
		proof := clitest.CompileFails(t, cli.Definition[rootConfig]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[rootConfig]) {
				profile := root.Values.String(&root.Config.Profile)
				root.Subcommands.Command(cli.Command[childConfig]{
					Name: "child",
					Declare: func(child *cli.CommandBuilder[childConfig]) {
						child.Values.String(&child.Config.Profile)
						child.Access.Value(profile, &child.Config.Profile)
					},
				})
			},
		})
		proof.Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
	})
}

func TestAuthoringTransactionStagesOrderedRecordsForPublicProjection(t *testing.T) {
	t.Parallel()
	type config = authoringTransactionNameConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			name := root.Values.String(&root.Config.Name)
			root.Values.Default(name, "default")
			root.Flags.ValueFor(name).Long("name")
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.ValueFor(name, "name").Optional()
			})
		},
	})
	root := clitest.RootCommand()
	flag := root.Flag("--name")
	form := root.ArgumentFormAt(0)
	term := form.Term("name")
	clitest.Schema(app).Require(t,
		root.FlagsExactly(flag),
		root.ArgumentFormsExactly(form),
		form.TermsExactly(term),
		root.ValuesExactly(flag.Value()),
		root.ValuesExactly(term.Value()),
		flag.Value().StaticDefault().Scalar().Text("default"),
	)
}

type definitionAuthoringNameRootConfig struct{ Name string }

type definitionAuthoringCountChildConfig struct{ Count int }

type definitionAuthoringNameConfig struct{ Name string }

func TestDefinitionAuthoringLowersRootAndChildConfigurationFactories(t *testing.T) {
	t.Parallel()
	type rootConfig = definitionAuthoringNameRootConfig
	type childConfig = definitionAuthoringCountChildConfig
	var rootSeen, childSeen bool
	app := compileMustCompilePublic(t, cli.Definition[rootConfig]{
		Name: "commandment",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Flags.String(&root.Config.Name).Long("name")
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "child",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					child.Flags.Int(&child.Config.Count).Long("count")
				},
				Run: func(_ context.Context, c *childConfig) error { childSeen = c.Count == 3; return nil },
			})
		},
		Run: func(_ context.Context, c *rootConfig) error { rootSeen = c.Name == "root"; return nil },
	})
	if err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"--name", "root"},
	}); err != nil {
		t.Fatalf("root Run: %v", err)
	}
	if err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"child", "--count", "3"},
	}); err != nil {
		t.Fatalf("child Run: %v", err)
	}
	if !rootSeen || !childSeen {
		t.Fatalf("configuration factories not observed: root=%v child=%v", rootSeen, childSeen)
	}
}

func TestCaptureDefinitionAuthoringPreservesDefinitionAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = definitionAuthoringNameConfig
	var ran string
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Description: cli.Description{
			Short: "short",
			Long:  "long",
		},
		Examples: []cli.Example{{
			Description: "example",
			CommandLine: "tool --name value",
		}},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Name).Long("name")
		},
		Run: func(_ context.Context, cfg *config) error {
			ran = cfg.Name
			return nil
		},
	})
	if err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"--name", "value"},
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if ran != "value" {
		t.Fatalf("captured definition configuration = %q, want value", ran)
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

type selectionRestLateConfig struct {
	Rest []string
	Late string
}

type selectionRootConfig struct{ Root string }

type selectionValueCountConfig struct {
	Value string
	Count int
}

type selectionRootChildConfig struct {
	Root  string
	Child string
}

type selectionRegionRootConfig struct{ Region string }

type selectionModeChildConfig struct{ Mode string }

type selectionValueConfig struct{ Value string }

func TestArgumentFormUsesNestedDeclarationCapability(t *testing.T) {
	t.Parallel()
	type config = selectionValueConfig
	clitest.CompileFails(t, cli.Definition[config]{
		Name: "argument-parent-suspension",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Arguments.Form(func(*cli.ArgumentFormBuilder[config]) {
				root.Values.String(&root.Config.Value)
			})
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticBuilderMisuse).Present())
}

func TestArgumentFormChildExpiresAfterNestedCallback(t *testing.T) {
	t.Parallel()
	type config = selectionValueConfig
	var retained *cli.ArgumentFormBuilder[config]
	_, err := cli.Compile(cli.Definition[config]{
		Name: "argument-child-expiry",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				retained = form
				form.String(&form.Config.Value, "value")
			})
		},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	deferred := false
	func() {
		defer func() { deferred = recover() != nil }()
		retained.Literal("late")
	}()
	if !deferred {
		t.Fatal("retained form builder remained usable after nested callback")
	}
}

func TestArgumentFormsAcceptExplicitEmptyAndRejectNilOrNonterminalTail(t *testing.T) {
	t.Parallel()
	t.Run("explicit empty", func(t *testing.T) {
		t.Parallel()
		app := clitest.Compile(t, cli.Definition[struct{}]{
			Name: "empty-form",
			Declare: func(root *cli.CommandBuilder[struct{}]) {
				root.Arguments.Form(func(*cli.ArgumentFormBuilder[struct{}]) {})
			},
		})
		form := clitest.RootCommand().ArgumentFormAt(0)
		clitest.Schema(app).Require(
			t,
			clitest.RootCommand().ArgumentFormsExactly(form),
			form.TermsExactly(),
		)
	})

	t.Run("nil callback", func(t *testing.T) {
		t.Parallel()
		clitest.CompileFails(t, cli.Definition[struct{}]{
			Name:    "nil-form",
			Declare: func(root *cli.CommandBuilder[struct{}]) { root.Arguments.Form(nil) },
		}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
	})

	t.Run("term after remainder", func(t *testing.T) {
		t.Parallel()
		type config = selectionRestLateConfig
		clitest.CompileFails(t, cli.Definition[config]{
			Name: "terminal-form",
			Declare: func(root *cli.CommandBuilder[config]) {
				rest := root.Values.Sequence(&root.Config.Rest, cli.Value.String())
				root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
					form.Remainder(rest, "rest", cli.Count.ZeroOrMore())
					form.String(&form.Config.Late, "late")
				})
			},
		}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
	})
}

func TestSubcommandGrammarRejectsParentDestinationReuse(t *testing.T) {
	t.Parallel()
	type config = selectionRootConfig
	clitest.CompileFails(t, cli.Definition[config]{
		Name: "cross-command-destination",
		Declare: func(root *cli.CommandBuilder[config]) {
			rootDestination := root.Values.String(&root.Config.Root)
			root.Subcommands.Command(cli.Command[config]{
				Name: "child",
				Declare: func(child *cli.CommandBuilder[config]) {
					child.Flags.ValueFor(rootDestination).Long("foreign")
				},
			})
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestFlagAuthoringRejectsMissingOrContradictorySpellingGrammarAtomically(t *testing.T) {
	t.Parallel()
	type config = selectionValueCountConfig
	tests := []struct {
		name    string
		declare func(*cli.CommandBuilder[config])
	}{
		{"missing spelling", func(root *cli.CommandBuilder[config]) { root.Flags.String(&root.Config.Value) }},
		{"long joined", func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Value).Long("value").ValueForms(cli.FlagValueFormJoined)
		}},
		{"count forms", func(root *cli.CommandBuilder[config]) {
			root.Flags.Count(&root.Config.Count).Short('v').ValueForms(cli.FlagValueFormJoined)
		}},
		{"fixed and implicit", func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Value).Short('x').FixedValue("a").ImplicitValue("b")
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			clitest.CompileFails(t, cli.Definition[config]{
				Name:    "bad-flag",
				Declare: test.declare,
			}).Require(
				t,
				clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present(),
			)
		})
	}
}

func TestAuthoringAcceptancePreservesStructuralGrammarOwnershipFacts(t *testing.T) {
	t.Parallel()
	type config = selectionRootChildConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:            "ownership",
		ShellCompletion: cli.ShellCompletion{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Root).Long("root")
			root.Subcommands.Command(cli.Command[config]{
				Name:    "child",
				Declare: func(child *cli.CommandBuilder[config]) { child.Flags.String(&child.Config.Child).Long("child") },
			})
		},
	})
	root := clitest.RootCommand()
	child := root.Command("child")
	clitest.Schema(app).Require(
		t,
		clitest.CommandsExactly(root, child),
		root.FlagsExactly(root.Flag("--root")),
		child.FlagsExactly(child.Flag("--child")),
	)
}

func TestDefinitionAuthoringLowersStaticCommandShortcut(t *testing.T) {
	t.Parallel()
	type rootConfig = selectionRegionRootConfig
	type childConfig = selectionModeChildConfig
	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "commandment",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			region := root.Values.String(&root.Config.Region)
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "serve",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					mode := child.Values.String(&child.Config.Mode)
					short := child.Shortcut(cli.CommandShortcut{
						Name: "quick",
						Description: cli.Description{
							Short: "quick serve",
						},
						Examples: []cli.Example{{
							Description: "example",
							CommandLine: "commandment quick",
						}},
						Args:       []string{"--fast"},
						AppendArgs: []string{"tail"},
						Hidden:     true,
					})
					short.Default(region, "us-east", cli.Source.Key("shortcut-region"))
					short.Default(mode, "safe", cli.Source.Sensitive())
					replacement := child.Shortcut(cli.CommandShortcut{
						Name: "quicker",
						Args: []string{"--faster"},
					})
					short.ReplacedBy(replacement)
				},
			})
		},
	})
	root := clitest.RootCommand()
	serve := root.Command("serve")
	quick := root.Shortcut("quick")
	quicker := root.Shortcut("quicker")
	clitest.Schema(app).Require(
		t,
		quick.Exists(),
		quick.Kind(cli.CommandEntryShortcut),
		quick.Hidden(),
		quick.SourceDescription(cli.Description{
			Short: "quick serve",
		}),
		quick.ShortcutDefaultCount(2),
		quick.ShortcutDefaultAt(0).Destination(root.ValueAt(0)),
		quick.ShortcutDefaultAt(0).ValueCount(1),
		quick.ShortcutDefaultAt(0).ValueAt(0).Text("us-east"),
		quick.ShortcutDefaultAt(1).Destination(serve.ValueAt(0)),
		quick.ShortcutDefaultAt(1).Sensitive(),
		quick.ShortcutDefaultAt(1).ValueAt(0).Sensitive(),
		quick.ShortcutDefaultAt(1).ValueAt(0).Redacted(),
		quick.ReplacedBy(quicker),
	)
}

func TestDefinitionAuthoringRejectsInvalidStaticShortcutShapes(t *testing.T) {
	t.Parallel()
	t.Run("root shortcut", func(t *testing.T) {
		t.Parallel()
		clitest.CompileFails(t, cli.Definition[struct{}]{
			Name: "commandment",
			Declare: func(root *cli.CommandBuilder[struct{}]) {
				root.Shortcut(cli.CommandShortcut{
					Name: "root-shortcut",
					Args: []string{"x"},
				})
			},
		}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
	})
	t.Run("name only", func(t *testing.T) {
		t.Parallel()
		clitest.CompileFails(t, cli.Definition[struct{}]{
			Name: "commandment",
			Declare: func(root *cli.CommandBuilder[struct{}]) {
				root.Subcommands.Command(cli.Command[struct{}]{
					Name: "child",
					Declare: func(child *cli.CommandBuilder[struct{}]) {
						child.Shortcut(cli.CommandShortcut{
							Name: "alias-shaped",
						})
					},
				})
			},
		}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
	})
	t.Run("self replacement", func(t *testing.T) {
		t.Parallel()
		clitest.CompileFails(t, cli.Definition[struct{}]{
			Name: "commandment",
			Declare: func(root *cli.CommandBuilder[struct{}]) {
				root.Subcommands.Command(cli.Command[struct{}]{
					Name: "child",
					Declare: func(child *cli.CommandBuilder[struct{}]) {
						shortcut := child.Shortcut(cli.CommandShortcut{
							Name: "quick",
							Args: []string{"x"},
						})
						shortcut.ReplacedBy(shortcut)
					},
				})
			},
		}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
	})
}

type cutoverAvailabilityConfig struct{ Value string }

type cutoverGrammarConfig struct {
	Value   string
	Values  []string
	Profile string
}

type cutoverDefaultConfig struct{ Selector string }

func TestCompileAllowsVisibilityDependingOnFormOutput(t *testing.T) {
	t.Parallel()
	clitest.Compile(t, cli.Definition[cutoverAvailabilityConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[cutoverAvailabilityConfig]) {
			value := root.Values.String(&root.Config.Value)
			form := root.Arguments.Form(func(form *cli.ArgumentFormBuilder[cutoverAvailabilityConfig]) {
				form.ValueFor(value, "value")
			})
			form.VisibleWhen(root.When.Resolved(value))
		},
	})
}

func TestCompileRejectsChildAvailabilityDependingOnChildLocalValue(t *testing.T) {
	t.Parallel()
	var condition cli.Condition
	clitest.CompileFails(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			child := root.Subcommands.Command(cli.Command[cutoverAvailabilityConfig]{
				Name: "child",
				Declare: func(child *cli.CommandBuilder[cutoverAvailabilityConfig]) {
					value := child.Values.String(&child.Config.Value)
					condition = child.When.Resolved(value)
				},
			})
			child.AvailableWhen(condition)
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestCompileRejectsDuplicateConditionEffectForExactTarget(t *testing.T) {
	t.Parallel()
	clitest.CompileFails(t, cli.Definition[cutoverDefaultConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[cutoverDefaultConfig]) {
			selector := root.Values.String(&root.Config.Selector)
			flag := root.Flags.ValueFor(selector)
			flag.Long("selector")
			condition := root.When.Resolved(selector)
			flag.AvailableWhen(condition).AvailableWhen(condition)
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestCompileRejectsEmptyCommandNameAtAcceptance(t *testing.T) {
	t.Parallel()
	clitest.CompileFails(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.Command(cli.Command[struct{}]{Name: ""})
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestCompileRejectsFlagAvailabilityDependingOnItsOwnValue(t *testing.T) {
	t.Parallel()
	clitest.CompileFails(t, cli.Definition[cutoverAvailabilityConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[cutoverAvailabilityConfig]) {
			flag := root.Flags.String(&root.Config.Value)
			flag.Long("value")
			flag.AvailableWhen(root.When.Resolved(flag))
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestCompileRejectsFlagTerminatorCollidingWithEndOfOptions(t *testing.T) {
	t.Parallel()
	clitest.CompileFails(t, cli.Definition[cutoverGrammarConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[cutoverGrammarConfig]) {
			root.Dialect.EndOfOptions("--")
			root.Flags.Strings(&root.Config.Values).
				ValuesPerOccurrence(cli.Count.Between(1, 2)).
				TerminatedBy("--").
				Long("item")
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}
