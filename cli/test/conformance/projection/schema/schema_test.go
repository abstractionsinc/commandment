package conformance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestZeroApplicationRejectsSchemaUse(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Fatal("zero Application did not reject use")
		}
	}()
	var app cli.Application
	_ = app.Schema()
}

func TestApplicationSchemaRetainsRootName(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "schema",
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().Name("schema"))
}

var _ = errors.Is

func schemaMustCompilePublic[C any](t testing.TB, definition cli.Definition[C]) cli.Application {
	t.Helper()
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return app
}

type schemaNameSelectorItemsOpaqueEventsConfig struct {
	Name     string
	Selector string
	Items    []string
	Opaque   schemaNumberOpaqueValue
	Events   <-chan cli.SignalEvent
}

type schemaNameItemsOpaqueEventsConfig struct {
	Name   string
	Items  []string
	Opaque schemaNumberOpaqueValue
	Events <-chan cli.SignalEvent
}

type schemaDynamicItemsOpaqueConfig struct {
	Dynamic string
	Items   []string
	Opaque  schemaNOpaque
}

type schemaNameSecretVerboseRootConfig struct {
	Name    string
	Secret  string
	Verbose bool
}

type schemaInputChildConfig struct{ Input string }

type schemaNameLimitConfig struct {
	Name  string
	Limit int
}

type schemaNumberOpaqueValue struct{ Number int }

type schemaABConfig struct{ A, B string }

type schemaConfig struct{}

type schemaEnabledModeConfig struct {
	Enabled bool
	Mode    string
}

type schemaModeConfig struct{ Mode string }

type schemaProfileValueConfig struct {
	Profile string
	Gate    string
	Value   string
}

type schemaItemsSecretLabelsLabelsConfig struct {
	Items        []string
	SecretLabels map[string]int
	Labels       map[string]int
}

type schemaValueConfig struct{ Value string }

type schemaLabelsConfig struct{ Labels map[string]string }

type schemaOldNewLegacyCurrentConfig struct {
	Old, New string
	Legacy   string
	Current  string
}

type schemaOldFlagNewFlagOldArgNewArgOldEnvNewEnvOldMapNewMapConfig struct {
	OldFlag, NewFlag string
	OldArg, NewArg   string
	OldEnv, NewEnv   string
	OldMap, NewMap   map[string]string
}

type schemaOldNewLegacyCurrentConfig2 struct{ Old, New, Legacy, Current string }

type schemaValuesConfig struct{ Values [10]string }

type schemaNSConfig struct {
	N int
	S string
}

type schemaCountConfig struct{ Count int }

type schemaNOpaque struct{ N int }

type schemaSItemsConfig struct {
	S     string
	Items []string
}

type schemaSConfig struct{ S string }

type schemaGateConfig struct{ Gate string }

type schemaSignalDelayConfig struct {
	Signal cli.Signal
	Delay  time.Duration
}

type schemaNameSecretItemsEventsConfig struct {
	Name   string
	Secret string
	Items  []string
	Events <-chan cli.SignalEvent
}

func declareSchemaProjectionFixture(root *cli.CommandBuilder[schemaNameSecretItemsEventsConfig]) {
	name := root.Values.String(&root.Config.Name)
	root.Values.Default(name, "guest")
	secret := root.Values.String(&root.Config.Secret).Sensitive()
	root.Flags.ValueFor(secret).Description("secret").Long("secret").Hidden().Sensitive()
	items := root.Values.Sequence(&root.Config.Items, cli.Value.String())
	root.Arguments.Form(func(form *cli.ArgumentFormBuilder[schemaNameSecretItemsEventsConfig]) {
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

func TestCompiledApplicationPublishesDetachedStaticSchemaWithoutLoss(t *testing.T) {
	t.Parallel()
	type rootConfig = schemaNameSecretVerboseRootConfig
	type childConfig = schemaInputChildConfig
	localization := cli.Localization{
		DefaultLocale: "en",
		Bundles: []cli.TranslationBundle{{
			Locale: "fr",
			Translations: cli.Translations{
				"tool short": "outil court",
			},
		}},
	}
	definition := cli.Definition[rootConfig]{
		Name: "tool",
		Description: cli.Description{
			Short: "tool short",
			Long:  "tool long",
		},
		Examples: []cli.Example{{
			Description: "child example",
			CommandLine: "tool child value",
		}},
		Localization: localization,
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			name := root.Values.String(&root.Config.Name)
			root.Values.Default(name, "guest")
			secret := root.Values.String(&root.Config.Secret).Sensitive()
			root.Values.Default(secret, "classified").Sensitive()
			root.Flags.ValueFor(secret).ValueName("SECRET").
				Long("secret").
				FixedValue("classified", cli.Source.Sensitive()).
				Sensitive()
			root.Flags.HelpGroup("Options", func(flags *cli.Flags[rootConfig]) {
				verbose := flags.Bool(&root.Config.Verbose).Description("verbose output")
				verbose.Long("verbose")
				verbose.Short('v')
			})
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[rootConfig]) {
				form.ValueFor(name, "name").Optional().Description("display name")
			})
			root.Subcommands.HelpGroup("Commands", func(subcommands *cli.Subcommands[rootConfig]) {
				subcommands.Command(cli.Command[childConfig]{
					Name: "child",
					Description: cli.Description{
						Short: "child short",
					},
					Declare: func(child *cli.CommandBuilder[childConfig]) {
						child.Arguments.Form(
							func(form *cli.ArgumentFormBuilder[childConfig]) { form.String(&child.Config.Input, "input") },
						)
					},
				})
			})
		},
	}
	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	child := root.Command("child")
	completion := root.Command("completion")
	secret := root.ValueAt(1)
	secretSpelling := root.Flag("--secret").Spelling("--secret")
	staticExpectations := []clitest.SchemaExpectation{
		clitest.CommandsExactly(root, child, completion),
		root.ChildrenExactly(child, completion),
		root.SourceDescription(cli.Description{
			Short: "tool short",
			Long:  "tool long",
		}),
		root.SourceExamplesExactly(cli.Example{
			Description: "child example",
			CommandLine: "tool child value",
		}),
		clitest.LocalizationExactly(localization),
		secret.DeclaredSensitive(),
		secret.StaticDefault().Exists(),
		secret.StaticDefault().Sensitive(),
		secret.StaticDefault().Redacted(),
		secret.StaticDefault().Scalar().Type(cli.ValueType{
			Kind: cli.ValueKindString,
		}),
		secret.StaticDefault().Scalar().Sensitive(),
		secret.StaticDefault().Scalar().Redacted(),
		secret.StaticDefault().Scalar().Text(""),
		secretSpelling.Sensitive(),
		secretSpelling.FixedValue().Type(cli.ValueType{
			Kind: cli.ValueKindString,
		}),
		secretSpelling.FixedValue().Sensitive(),
		secretSpelling.FixedValue().Redacted(),
		secretSpelling.FixedValue().Text(""),
		root.HelpFlagGroup("Options").FlagsExactly(root.Flag("--verbose")),
		root.HelpCommandGroup("Commands").EntriesExactly(child.CanonicalEntry()),
	}
	firstProof := clitest.Schema(app)
	firstProof.Require(t, staticExpectations...)
	first := firstProof.Schema()
	first.CommandOrder[0] = 0
	first.Localization.Bundles[0].Translations["tool short"] = "mutated"
	for id, value := range first.Values {
		value.StaticDefault = cli.ValueStaticDefault{}
		first.Values[id] = value
	}
	clitest.Schema(app).Require(t, staticExpectations...)
}

func TestCompiledApplicationPublishesStaticPolicyAndEndpointFreeSignalSchema(t *testing.T) {
	t.Parallel()
	type config = schemaNameLimitConfig
	ruleCalled := false
	advisoryCalled := false
	definition := cli.Definition[config]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"always message": "message toujours",
					"signal hint":    "indice signal",
				},
			}},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			name := root.Values.String(&root.Config.Name)
			limit := root.Values.Int(&root.Config.Limit)
			root.Rule.Require(name)
			root.Rule.When(root.Rule.ContributedBy(name, cli.ValueProducerKindFlag)).Between(limit, 1, 10)
			root.Rule.Check("custom-rule", func(context.Context, *config, cli.Dependencies) (bool, error) { ruleCalled = true; return true, nil }).
				Subject(name).
				DependsOn(limit.OptionalDependency())
			root.Advisory.Always(cli.Advisory{
				Code:     "always",
				Severity: cli.NoticeSeverityInformation,
				Message:  "always message",
			}).Subject(name)
			root.Advisory.Detect(func(context.Context, *config, cli.Dependencies) (bool, error) {
				advisoryCalled = true
				return true, nil
			}, cli.Advisory{
				Code:     "dynamic",
				Severity: cli.NoticeSeverityWarning,
				Message:  "dynamic message",
			}).DependsOn(limit)
			root.Notice.SuppressWhen(root.When.Resolved(name), cli.NoticeMatch{
				AdvisoryCode: "dynamic",
			})
			root.Signals.Policy(
				root.Signals.Match.Interrupt().Each().Hint("signal hint"),
				root.Signals.Action.Abort(17),
			)
		},
	}
	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	policy := root.SignalPolicyAt(0)
	clitest.Schema(app).Require(t,
		root.RulesExactly(root.RuleAt(0), root.RuleAt(1), root.RuleAt(2)),
		root.RuleAt(0).Kind(cli.RuleRequire),
		root.RuleAt(1).Kind(cli.RuleBetween),
		root.RuleAt(1).Activation().Kind(cli.ConditionExpressionContributedBy),
		root.RuleAt(1).OperandCount(2),
		root.RuleAt(2).Kind(cli.RuleCustom),
		root.AdvisoriesExactly(root.AdvisoryAt(0), root.AdvisoryAt(1)),
		root.AdvisoryAt(0).Kind(cli.AdvisoryKindAlways),
		root.AdvisoryAt(1).Kind(cli.AdvisoryKindDetector),
		root.NoticeSuppressionsExactly(root.NoticeSuppressionAt(0)),
		root.SignalPoliciesExactly(policy),
		policy.Selector().Kind(cli.SignalSelectorInterrupt),
		policy.Aggregation(cli.SignalAggregationEach),
		policy.ActionCount(1),
		policy.ActionAt(0).Kind(cli.SignalActionAbort),
		policy.ActionAt(0).ExitCode(17),
		clitest.SignalEndpointCount(0),
	)
	if ruleCalled || advisoryCalled {
		t.Fatal("analysis callbacks executed during static publication")
	}

	t.Run("required signal capabilities and callbacks", func(t *testing.T) {
		t.Parallel()
		target := cli.SignalTarget[config]{
			Name:          "peer",
			MatchesSender: func(context.Context, *config, cli.SignalSender) (bool, error) { return false, nil },
		}
		capabilityApp := clitest.Compile(t, cli.Definition[config]{
			Name: "commandment",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Signals.Policy(
					root.Signals.Match.Interrupt().Origin(cli.SignalOriginProcess).ExceptSender(target).Each(),
					root.Signals.Action.Abort(17),
				)
			},
		})
		capabilityPolicy := clitest.RootCommand().SignalPolicyAt(0)
		clitest.Schema(capabilityApp).Require(t,
			capabilityPolicy.RequiredSourceCapabilities(cli.SignalSourceCapabilityOrigin|cli.SignalSourceCapabilitySender),
			capabilityPolicy.RequiredHostCapabilities(cli.SignalHostCapabilityAbort),
		)
		for _, test := range []struct {
			name string
			plan func(*cli.Signals[config]) cli.SignalActionPlan[config]
		}{
			{
				name: "handler callback",
				plan: func(signals *cli.Signals[config]) cli.SignalActionPlan[config] {
					return signals.Action.Handle(cli.SignalHandler[config]{
						Name: "missing",
					}).ThenAbort(1)
				},
			},
			{
				name: "target callback",
				plan: func(signals *cli.Signals[config]) cli.SignalActionPlan[config] {
					return signals.Action.Forward(cli.SignalTarget[config]{
						Name: "missing",
					}).ThenAbort(1)
				},
			},
		} {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()
				clitest.CompileFails(t, cli.Definition[config]{
					Name: "commandment",
					Declare: func(root *cli.CommandBuilder[config]) {
						root.Signals.Policy(root.Signals.Match.Interrupt().Each(), test.plan(&root.Signals))
					},
				})
			})
		}
	})

	clitest.Reference(app, cli.ReferenceOptions{
		Locale:     "fr",
		Visibility: cli.ReferenceVisibilityPublic,
		Order:      cli.ReferenceOrderDeclaration,
	}).Require(t,
		clitest.EffectiveLocale("fr"),
		policy.Exists(),
		policy.Visible(),
		policy.Hint("indice signal"),
	)
}

func TestCompilePublishesStaticArtifactAuthorityJoins(t *testing.T) {
	t.Parallel()
	type opaqueValue = schemaNumberOpaqueValue
	type config = schemaNameSelectorItemsOpaqueEventsConfig

	cases := []struct {
		name    string
		release func() cli.ReleaseScheme
		declare func(*cli.CommandBuilder[config])
	}{
		{
			name: "command condition target identity",
			declare: func(root *cli.CommandBuilder[config]) {
				name := root.Values.String(&root.Config.Name)
				root.AvailableWhen(root.When.Resolved(name))
			},
		},
		{
			name: "condition match pattern carrier",
			declare: func(root *cli.CommandBuilder[config]) {
				selector := root.Values.String(&root.Config.Selector)
				name := root.Values.String(&root.Config.Name)
				flag := root.Flags.ValueFor(name)
				flag.Long("name")
				flag.AvailableWhen(root.When.Matches(selector, "^[a-z]+$"))
			},
		},
		{
			name: "rule match pattern carrier",
			declare: func(root *cli.CommandBuilder[config]) {
				name := root.Values.String(&root.Config.Name)
				root.Rule.Match(name, "^[a-z]+$")
			},
		},
		{
			name: "rule item-count range carrier",
			declare: func(root *cli.CommandBuilder[config]) {
				items := root.Values.Sequence(&root.Config.Items, cli.Value.String())
				root.Rule.ItemCount(items, cli.Count.Between(1, 2))
			},
		},
		{
			name: "non-scalar static default literal encoding",
			declare: func(root *cli.CommandBuilder[config]) {
				items := root.Values.Sequence(&root.Config.Items, cli.Value.String())
				root.Values.Default(items, []string{"one", "two"})
			},
		},
		{
			name: "opaque destination value type",
			declare: func(root *cli.CommandBuilder[config]) {
				opaque := root.Values.Destination(&root.Config.Opaque, cli.Value.Assign[opaqueValue]())
				root.Values.Default(opaque, opaqueValue{
					Number: 7,
				})
			},
		},
		{
			name: "signal endpoint capacity",
			declare: func(root *cli.CommandBuilder[config]) {
				events := root.Signals.Endpoint(&root.Config.Events)
				root.Signals.Policy(
					root.Signals.Match.Interrupt().Each(),
					root.Signals.Action.Emit(events),
				)
			},
		},
		{
			name:    "replacement artifact presentation",
			release: func() cli.ReleaseScheme { return cli.Release.Ordered("2", "1", "2", "3") },
			declare: func(root *cli.CommandBuilder[config]) {
				old := root.Subcommands.Command(cli.Command[config]{
					Name: "old",
					Release: cli.ReleaseLifecycle{
						DeprecatedIn: "2",
					},
				})
				newer := root.Subcommands.Command(cli.Command[config]{
					Name: "new",
					Release: cli.ReleaseLifecycle{
						IntroducedIn: "1",
					},
				})
				old.ReplacedBy(newer)
			},
		},
		{
			name:    "flag-spelling replacement compatibility",
			release: func() cli.ReleaseScheme { return cli.Release.Ordered("2", "1", "2", "3") },
			declare: func(root *cli.CommandBuilder[config]) {
				flag := root.Flags.String(&root.Config.Name)
				old := flag.Long("old").Release(cli.ReleaseLifecycle{
					DeprecatedIn: "2",
				})
				newer := flag.Long("new").Release(cli.ReleaseLifecycle{
					IntroducedIn: "1",
				})
				old.ReplacedBy(newer)
			},
		},
		{
			name:    "positional-literal replacement compatibility",
			release: func() cli.ReleaseScheme { return cli.Release.Ordered("2", "1", "2", "3") },
			declare: func(root *cli.CommandBuilder[config]) {
				root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
					old := form.Literal("old").Release(cli.ReleaseLifecycle{
						DeprecatedIn: "2",
					})
					newer := form.Literal("new").Release(cli.ReleaseLifecycle{
						IntroducedIn: "1",
					})
					old.ReplacedBy(newer)
				})
			},
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			release := cli.Release.Ordered("1", "1")
			if test.release != nil {
				release = test.release()
			}
			app, err := cli.Compile(cli.Definition[config]{
				Name:    "tool",
				Release: release,
				Declare: test.declare,
			})
			if err != nil {
				t.Fatalf("Compile() error = %v", err)
			}
			schemaProof := clitest.Schema(app)
			schemaProof.Require(t, clitest.RootCommand().Exists())
		})
	}
}

func TestSchemaProjectsStaticArtifactRepresentations(t *testing.T) {
	t.Parallel()
	type opaqueValue = schemaNumberOpaqueValue
	type config = schemaNameItemsOpaqueEventsConfig
	app, err := cli.Compile(cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2", "3"),
		Declare: func(root *cli.CommandBuilder[config]) {
			name := root.Values.String(&root.Config.Name)
			root.AvailableWhen(root.When.Resolved(name))
			root.Rule.Match(name, "")
			items := root.Values.Sequence(&root.Config.Items, cli.Value.String())
			root.Values.Default(items, []string{"two", "one"})
			root.Rule.ItemCount(items, cli.Count.Between(0, 2))
			opaque := root.Values.Destination(&root.Config.Opaque, cli.Value.Assign[opaqueValue]())
			root.Values.Default(opaque, opaqueValue{
				Number: 7,
			})
			events := root.Signals.Endpoint(&root.Config.Events)
			root.Signals.Policy(root.Signals.Match.Interrupt().Each(), root.Signals.Action.Emit(events))
		},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	schemaProof := clitest.Schema(app)
	rootLocator := clitest.RootCommand()
	schemaProof.Require(t,
		rootLocator.Exists(),
		rootLocator.SignalPolicyAt(0).Exists(),
		rootLocator.SignalPolicyAt(0).ActionAt(0).SignalEndpoint().Capacity(cli.SignalBacklogCapacity),
	)
	schema := schemaProof.Schema()
	root := schema.Commands[schema.Root]
	if len(root.Conditions) != 1 || root.Conditions[0].Target.Kind != cli.DeclarationKindCommand ||
		root.Conditions[0].Target.Command != schema.Root {
		t.Fatalf("command condition target = %#v", root.Conditions)
	}
	if len(root.Rules) != 2 || root.Rules[0].Kind != cli.RuleMatch || root.Rules[0].Pattern != "" ||
		root.Rules[1].Count != cli.Count.Between(0, 2) {
		t.Fatalf("rules = %#v", root.Rules)
	}
	var sawSequence, sawOpaque bool
	for _, value := range schema.Values {
		switch value.Kind {
		case cli.ValueDestinationSequence:
			if !value.HasType || !value.HasStaticDefault || value.StaticDefault.Sequence == nil ||
				!value.StaticDefault.HasRepresentation ||
				len(value.StaticDefault.Sequence.Elements) != 2 {
				t.Fatalf("sequence value = %#v", value)
			}
			sawSequence = true
		case cli.ValueDestinationOpaque:
			if value.HasType || value.Type != (cli.ValueType{}) || !value.HasStaticDefault ||
				value.StaticDefault.HasRepresentation ||
				value.StaticDefault.Redacted ||
				value.StaticDefault.Scalar != nil ||
				value.StaticDefault.Sequence != nil ||
				value.StaticDefault.Keyed != nil {
				t.Fatalf("opaque value = %#v", value)
			}
			sawOpaque = true
		case cli.ValueDestinationUnknown,
			cli.ValueDestinationScalar,
			cli.ValueDestinationKeyed,
			cli.ValueDestinationDerived,
			cli.ValueDestinationOptionalDerived,
			cli.ValueDestinationFactRequired,
			cli.ValueDestinationFactResolved:
		}
	}
	if !sawSequence || !sawOpaque {
		t.Fatalf("values = %#v", schema.Values)
	}
}

func TestReplacementCompatibilityRejectsCrossFlagAndCrossCommandLiteral(t *testing.T) {
	t.Parallel()
	type config = schemaABConfig
	validApp := schemaMustCompilePublic(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: func(root *cli.CommandBuilder[config]) {
			flag := root.Flags.String(&root.Config.A)
			old := flag.Long("old").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2",
			})
			newer := flag.Long("new").Release(cli.ReleaseLifecycle{
				IntroducedIn: "1",
			})
			old.ReplacedBy(newer)
		},
	})
	validProof := clitest.Schema(validApp)
	validProof.Require(t, clitest.RootCommand().Exists())
	_, err := cli.Compile(cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: func(root *cli.CommandBuilder[config]) {
			left := root.Flags.String(&root.Config.A)
			right := root.Flags.String(&root.Config.B)
			old := left.Long("old").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2",
			})
			newer := right.Long("new").Release(cli.ReleaseLifecycle{
				IntroducedIn: "1",
			})
			old.ReplacedBy(newer)
		},
	})
	if err == nil {
		t.Fatal("cross-flag spelling replacement accepted")
	}

	_, err = cli.Compile(cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: func(root *cli.CommandBuilder[config]) {
			var old cli.ArgumentLiteralDeclaration[config]
			var newer cli.ArgumentLiteralDeclaration[config]
			root.Subcommands.Command(cli.Command[config]{
				Name: "left",
				Declare: func(left *cli.CommandBuilder[config]) {
					left.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
						old = form.Literal("old").Release(cli.ReleaseLifecycle{
							DeprecatedIn: "2",
						})
					})
				},
			})
			root.Subcommands.Command(cli.Command[config]{
				Name: "right",
				Declare: func(right *cli.CommandBuilder[config]) {
					right.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
						newer = form.Literal("new").Release(cli.ReleaseLifecycle{
							IntroducedIn: "1",
						})
					})
				},
			})
			old.ReplacedBy(newer)
		},
	})
	if err == nil {
		t.Fatal("cross-command literal replacement accepted")
	}
}

func TestSchemaRemainsStaticAcrossIndependentProviderOperations(t *testing.T) {
	t.Parallel()
	type config = schemaConfig
	providerCalls := 0
	app, err := cli.Compile(cli.Definition[config]{
		Name:            "tool",
		ShellCompletion: cli.ShellCompletion{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				providerCalls++
				batch.Command(cli.Command[config]{
					Name: "remote",
				})
				return nil
			})
		},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	firstProof := clitest.Schema(app)
	firstProof.Require(t,
		clitest.CommandsExactly(clitest.RootCommand()),
		clitest.Command("remote").DoesNotExist(),
	)
	if providerCalls != 0 {
		t.Fatalf("Schema() executed provider %d times", providerCalls)
	}
	if t.Failed() {
		return
	}

	first := firstProof.Schema()
	first.CommandOrder[0] = 0

	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"remote"},
	}).Require(t,
		clitest.InspectionReady(),
		clitest.Command("remote").Exists(),
	)
	if providerCalls != 1 {
		t.Fatalf("provider calls after Inspect = %d, want 1", providerCalls)
	}

	clitest.Schema(app).Require(t,
		clitest.CommandsExactly(clitest.RootCommand()),
		clitest.Command("remote").DoesNotExist(),
	)
	if providerCalls != 1 {
		t.Fatalf("second Schema() executed provider: calls=%d", providerCalls)
	}
}

func TestSpellingAndFiniteDomainPresentationMetadataProjectAcrossArtifacts(t *testing.T) {
	t.Parallel()
	type config = schemaEnabledModeConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.Bool(&root.Config.Enabled).Long("secret").Hidden()
			domain := cli.Value.Domain(cli.Value.Assign[string](), func(builder *cli.ValueDomainBuilder[string]) {
				builder.Value("prod", "production").Description("Production environment").Hidden()
			})
			root.Flags.Value(&root.Config.Mode, domain).Long("mode")
		},
	})
	root := clitest.RootCommand()
	schema := clitest.Schema(app)
	spelling := clitest.ResolveDeclaration(t, schema, root.Flag("--secret").Spelling("--secret"))
	entry := clitest.ResolveDeclaration(
		t,
		schema,
		root.Flag("--mode").Value().Domain(cli.ValueDomainUseDestination).Entry("prod"),
	)
	schema.Require(t,
		spelling.Hidden(),
		entry.Hidden(),
		entry.SourceDescription(cli.Description{
			Short: "Production environment",
		}),
	)
	clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityPublic,
	}).Require(t,
		spelling.NotVisible(),
		entry.NotVisible(),
		entry.PresentedDescription(cli.Description{
			Short: "Production environment",
		}),
	)
	clitest.Reference(app, cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityAll,
	}).Require(t,
		spelling.Visible(),
		entry.Visible(),
		entry.PresentedDescription(cli.Description{
			Short: "Production environment",
		}),
	)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		spelling.NotVisible(),
		entry.NotVisible(),
		entry.PresentedDescription(cli.Description{
			Short: "Production environment",
		}),
	)
}

func TestProjectApplicationSchemaPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestSchemaCommandPathPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectSchemaCommandsAndEntriesPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectSchemaActionsAndGrammarPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectSchemaValuesAndDomainsPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectSchemaStaticRelationsPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectLifecycleStatePreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestValueTypePreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectPublicDialectPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectStaticReachabilityPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectGrammarProfileSetPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectHelpPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectCompletionPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectValueLiteralPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectShortcutDefaultValuesPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectStaticDefaultPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectValueDestinationKindPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestDeclarationOriginPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestActionEntryKindPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestInvocationActionPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectFlagSpellingKindPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectArgumentTermKindPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestValueDomainEntryKindPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestValueDomainUseKindPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestValueCandidateContributionPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestFlagOrderPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestShortClusteringPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestNameMatchingPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestNameNormalizationPreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestDashlessModePreservesSchemaProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectStaticReplacementsPreservesReplacementProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectReplacementIdentityPreservesReplacementProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestReplacementPresentationPreservesReplacementProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestSetGenericReplacementPreservesReplacementProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestActionEntryReplacementPresentationPreservesReplacementProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestDomainReplacementPresentationPreservesReplacementProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestTargetDeprecatedDuringMigrationPreservesReplacementProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectStaticSignalPoliciesPreservesSignalProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectSignalPolicySchemaPreservesSignalProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectSignalSelectorPreservesSignalProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectSignalActionPreservesSignalProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectSignalSelectorKindPreservesSignalProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectSignalAggregationPreservesSignalProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectSignalActionKindPreservesSignalProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectSchemaSignalOriginPreservesSignalProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectSignalPhaseSchemaPreservesSignalProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectSignalHostCapabilitiesPreservesSignalProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectStaticPoliciesPreservesStaticPolicyProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectConditionTargetPreservesStaticPolicyProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectConditionEffectPreservesStaticPolicyProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectConditionExpressionPreservesStaticPolicyProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectPolicyConditionExpressionPreservesStaticPolicyProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestDestinationSensitivePreservesStaticPolicyProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectRuleSchemaPreservesStaticPolicyProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectRuleOperandPreservesStaticPolicyProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestRuleOperandLanguagePreservesStaticPolicyProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectRuleFailurePreservesStaticPolicyProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectRuleKindPreservesStaticPolicyProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectAdvisorySchemaPreservesStaticPolicyProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectAdvisoryKindPreservesStaticPolicyProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectNoticeSeverityPreservesStaticPolicyProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectNoticeMatchPreservesStaticPolicyProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectValueProducerKindPreservesStaticPolicyProjectionSemantics(t *testing.T) {
	t.Parallel()
	type config = schemaNameSecretItemsEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2"),
		Declare: declareSchemaProjectionFixture,
	})
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

func TestProjectSchemaCommandsAndEntries(t *testing.T) {
	t.Parallel()
	type config = schemaModeConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			mode := root.Values.String(&root.Config.Mode)
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "child",
				Declare: func(child *cli.CommandBuilder[struct{}]) {
					child.Shortcut(cli.CommandShortcut{
						Name: "quick",
						Args: []string{"--mode"},
					}).Default(mode, "secret", cli.Source.Sensitive())
				},
			})
		},
	})
	root := clitest.RootCommand()
	quick := root.Shortcut("quick")
	defaultValue := quick.ShortcutDefaultAt(0)
	clitest.Schema(app).Require(t,
		quick.Kind(cli.CommandEntryShortcut),
		quick.ShortcutDefaultCount(1),
		defaultValue.Destination(root.ValueAt(0)),
		defaultValue.Sensitive(),
		defaultValue.ValueCount(1),
		defaultValue.ValueAt(0).Sensitive(),
		defaultValue.ValueAt(0).Redacted(),
		defaultValue.ValueAt(0).Text(""),
	)
}

func TestProjectStaticReachability(t *testing.T) {
	t.Parallel()
	type config = schemaProfileValueConfig
	domain := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) { values.Value("active", "active") })
	gateDomain := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
		values.Value("active", "active")
		values.Value("inactive", "inactive")
	})
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			profile := root.Values.Value(&root.Config.Profile, domain)
			root.Values.Default(profile, "active")
			gate := root.Values.Value(&root.Config.Gate, gateDomain)
			root.Profiles.Switch(profile, func(cases *cli.GrammarProfileCases[config, string]) {
				cases.Case("active", func(*cli.CommandBuilder[config]) {})
				cases.Default(func(inactive *cli.CommandBuilder[config]) {
					inactive.Arguments.Form(
						func(form *cli.ArgumentFormBuilder[config]) { form.String(&form.Config.Value, "value") },
					)
				})
			})
			impossible := root.When.All(root.When.Equal(gate, "active"), root.When.Equal(gate, "inactive"))
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "never-default",
			}).DefaultWhen(impossible)
		},
	})
	root := clitest.RootCommand()
	activeProfile := root.GrammarProfile("active")
	defaultProfile := root.GrammarProfile("")
	form := root.ArgumentFormAt(0)
	conditionalDefault := root.DefaultAt(0)
	clitest.Schema(app).Require(t,
		root.GrammarProfilesExactly(activeProfile),
		root.DefaultGrammarProfile(defaultProfile),
		activeProfile.Reachability(cli.ReachabilityReachable),
		defaultProfile.Reachability(cli.ReachabilityUnreachable),
		form.GrammarProfile(defaultProfile),
		form.Reachability(cli.ReachabilityInherited),
		conditionalDefault.Reachability(cli.ReachabilityUnreachable),
	)
}

func TestProjectStaticDefault(t *testing.T) {
	t.Parallel()
	type config = schemaItemsSecretLabelsLabelsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			items := root.Values.Sequence(&root.Config.Items, cli.Value.String()).Sensitive()
			root.Values.Default(items, []string{"two", "one"}).Sensitive()
			root.Flags.Sequence(items).Long("items")
			secret := root.Values.Keyed(
				&root.Config.SecretLabels,
				cli.Value.Keyed(cli.Value.String(), cli.Value.Int()),
			).Sensitive()
			root.Values.Default(secret, map[string]int{"z": 2, "a": 1}).Sensitive()
			root.Flags.Keyed(secret).Long("secret-labels")
			labels := root.Values.Keyed(&root.Config.Labels, cli.Value.Keyed(cli.Value.String(), cli.Value.Int()))
			root.Values.Default(labels, map[string]int{"z": 2, "a": 1})
			root.Flags.Keyed(labels).Long("labels")
		},
	})
	root := clitest.RootCommand()
	itemsDefault := root.Flag("--items").Value().StaticDefault()
	secretDefault := root.Flag("--secret-labels").Value().StaticDefault()
	labelsDefault := root.Flag("--labels").Value().StaticDefault()
	clitest.Schema(app).Require(t,
		itemsDefault.Exists(),
		itemsDefault.Sensitive(), itemsDefault.Redacted(), itemsDefault.Sequence().ElementCount(0),
		secretDefault.Exists(),
		secretDefault.Sensitive(), secretDefault.Redacted(), secretDefault.Keyed().EntryCount(0),
		labelsDefault.Exists(), labelsDefault.Keyed().EntryCount(2),
		labelsDefault.Keyed().EntryAt(0).Key().Text("a"),
		labelsDefault.Keyed().EntryAt(1).Key().Text("z"),
	)
}

func TestProjectFlagSpellingKind(t *testing.T) {
	t.Parallel()
	type config = schemaValueConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Value).Literal("/value")
		},
	})

	clitest.Schema(app).Require(t,
		clitest.RootCommand().Flag("/value").Spelling("/value").Kind(cli.FlagSpellingLiteral),
	)
}

func TestValueDomainUseKind(t *testing.T) {
	t.Parallel()
	type config = schemaLabelsConfig
	keys := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) { values.Value("key", "key") })
	values := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) { values.Value("value", "value") })
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Values.Keyed(&root.Config.Labels, cli.Value.Keyed(keys, values))
		},
	})

	value := clitest.RootCommand().ValueAt(0)
	clitest.Schema(app).Require(t,
		value.Domain(cli.ValueDomainUseMapKey).Exists(),
		value.Domain(cli.ValueDomainUseMapValue).Exists(),
	)
}

func TestProjectReplacementIdentity(t *testing.T) {
	t.Parallel()
	type config = schemaOldNewLegacyCurrentConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2", "3"),
		Declare: func(root *cli.CommandBuilder[config]) {
			oldFlag := root.Flags.String(&root.Config.Old).Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2",
			})
			oldFlag.Long("old")
			newFlag := root.Flags.String(&root.Config.New)
			newFlag.Long("new")
			oldFlag.ReplacedBy(newFlag)
			oldEnv := root.Values.Environment(root.Values.String(&root.Config.Legacy), "OLD").
				Release(cli.ReleaseLifecycle{
					DeprecatedIn: "2",
				})
			newEnv := root.Values.Environment(root.Values.String(&root.Config.Current), "NEW").
				Release(cli.ReleaseLifecycle{
					IntroducedIn: "1",
				})
			oldEnv.ReplacedBy(newEnv)
		},
	})

	clitest.Schema(app).Require(t,
		clitest.RootCommand().Flag("--old").ReplacedBy(clitest.RootCommand().Flag("--new")),
		clitest.RootCommand().Environment("OLD").ReplacedBy(clitest.RootCommand().Environment("NEW")),
	)
}

func TestReplacementPresentation(t *testing.T) {
	t.Parallel()
	type config = schemaOldFlagNewFlagOldArgNewArgOldEnvNewEnvOldMapNewMapConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2", "3"),
		Declare: func(root *cli.CommandBuilder[config]) {
			oldFlag := root.Flags.String(&root.Config.OldFlag).Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2",
			})
			oldFlag.Long("old")
			newFlag := root.Flags.String(&root.Config.NewFlag)
			newFlag.Short('n')
			newFlag.Long("new")
			oldFlag.ReplacedBy(newFlag)

			oldForm := root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) { form.String(&form.Config.OldArg, "old") }).
				Release(cli.ReleaseLifecycle{
					DeprecatedIn: "2",
				})
			newForm := root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) { form.String(&form.Config.NewArg, "replacement") }).
				Release(cli.ReleaseLifecycle{
					IntroducedIn: "1",
				})
			oldForm.ReplacedBy(newForm)

			oldEnv := root.Values.Environment(root.Values.String(&root.Config.OldEnv), "OLD").
				Release(cli.ReleaseLifecycle{
					DeprecatedIn: "2",
				})
			newEnv := root.Values.Environment(root.Values.String(&root.Config.NewEnv), "NEW").
				Release(cli.ReleaseLifecycle{
					IntroducedIn: "1",
				})
			oldEnv.ReplacedBy(newEnv)

			oldMap := root.Values.Keyed(&root.Config.OldMap, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
			newMap := root.Values.Keyed(&root.Config.NewMap, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
			oldPrefix := root.Values.EnvironmentPrefix(oldMap, "OLD_").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2",
			})
			newPrefix := root.Values.EnvironmentPrefix(newMap, "NEW_").Release(cli.ReleaseLifecycle{
				IntroducedIn: "1",
			})
			oldPrefix.ReplacedBy(newPrefix)
		},
	})

	clitest.Schema(app).Require(t,
		clitest.RootCommand().Flag("--old").ReplacementPresentation("-n"),
		clitest.RootCommand().ArgumentFormAt(0).ReplacementPresentation("(replacement)"),
		clitest.RootCommand().Environment("OLD").ReplacementPresentation("NEW"),
		clitest.RootCommand().Environment("OLD_").ReplacementPresentation("NEW_*"),
	)
}

func TestSetGenericReplacement(t *testing.T) {
	t.Parallel()
	type config = schemaOldNewLegacyCurrentConfig2
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2", "3"),
		Declare: func(root *cli.CommandBuilder[config]) {
			oldFlag := root.Flags.String(&root.Config.Old).Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2",
			})
			oldFlag.Long("old")
			newFlag := root.Flags.String(&root.Config.New)
			newFlag.Long("new")
			oldFlag.ReplacedBy(newFlag)
			oldEnv := root.Values.Environment(root.Values.String(&root.Config.Legacy), "OLD").
				Release(cli.ReleaseLifecycle{
					DeprecatedIn: "2",
				})
			newEnv := root.Values.Environment(root.Values.String(&root.Config.Current), "NEW").
				Release(cli.ReleaseLifecycle{
					IntroducedIn: "1",
				})
			oldEnv.ReplacedBy(newEnv)
		},
	})

	clitest.Schema(app).Require(t,
		clitest.RootCommand().Flag("--old").ReplacedBy(clitest.RootCommand().Flag("--new")),
		clitest.RootCommand().Environment("OLD").ReplacedBy(clitest.RootCommand().Environment("NEW")),
	)
}

func TestTargetDeprecatedDuringMigration(t *testing.T) {
	t.Parallel()
	type config = schemaValuesConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("2", "1", "2", "3", "4"),
		Declare: func(root *cli.CommandBuilder[config]) {
			flag := root.Flags.String(&root.Config.Values[0])
			a := flag.Long("a").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2",
			})
			b := flag.Long("b").Release(cli.ReleaseLifecycle{
				IntroducedIn: "1",
				DeprecatedIn: "2",
			})
			a.ReplacedBy(b)
			flag2 := root.Flags.String(&root.Config.Values[1])
			c := flag2.Long("c").Release(cli.ReleaseLifecycle{
				RemovedIn: "3",
			})
			d := flag2.Long("d").Release(cli.ReleaseLifecycle{
				IntroducedIn: "1",
				DeprecatedIn: "2",
			})
			c.ReplacedBy(d)
			flag3 := root.Flags.String(&root.Config.Values[2])
			e := flag3.Long("e").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2",
			})
			f := flag3.Long("f").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2",
			})
			e.ReplacedBy(f)
			flag4 := root.Flags.String(&root.Config.Values[3])
			g := flag4.Long("g").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2",
				RemovedIn:    "3",
			})
			h := flag4.Long("h").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "4",
			})
			g.ReplacedBy(h)
		},
	})

	clitest.Schema(app).Require(t,
		clitest.RootCommand().Flag("--a").Spelling("--a").ReplacementTargetDeprecatedDuringMigration(true),
		clitest.RootCommand().Flag("--c").Spelling("--c").ReplacementTargetDeprecatedDuringMigration(false),
	)
}

func TestProjectPolicyConditionExpression(t *testing.T) {
	t.Parallel()
	type config = schemaNSConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			n := root.Values.Int(&root.Config.N)
			s := root.Values.String(&root.Config.S)
			root.Rule.When(root.When.All(root.When.Equal(n, 1), root.When.NotEqual(n, 2), root.When.In(n, 1, 2), root.When.NotIn(n, 3, 4), root.When.GreaterThan(n, 0), root.When.AtLeast(n, 1), root.When.LessThan(n, 9), root.When.AtMost(n, 8), root.When.Matches(s, "x+"))).
				Require(n)
		},
	})
	root := clitest.RootCommand()
	n := root.ValueAt(0)
	s := root.ValueAt(1)
	activation := root.RuleAt(0).Activation()
	expectations := []clitest.SchemaExpectation{activation.Kind(cli.ConditionExpressionAll), activation.ChildCount(9)}
	kinds := []cli.ConditionExpressionKind{
		cli.ConditionExpressionEqual,
		cli.ConditionExpressionNotEqual,
		cli.ConditionExpressionIn,
		cli.ConditionExpressionNotIn,
		cli.ConditionExpressionGreaterThan,
		cli.ConditionExpressionAtLeast,
		cli.ConditionExpressionLessThan,
		cli.ConditionExpressionAtMost,
		cli.ConditionExpressionMatch,
	}
	operands := [][]string{{"1"}, {"2"}, {"1", "2"}, {"3", "4"}, {"0"}, {"1"}, {"9"}, {"8"}, nil}
	for i, kind := range kinds {
		child := activation.ChildAt(i)
		expectations = append(expectations, child.Kind(kind), child.Destination(n))
		if i == 8 {
			expectations[len(expectations)-1] = child.Destination(s)
			expectations = append(expectations, child.Pattern("x+"), child.OperandCount(0))
			continue
		}
		expectations = append(expectations, child.OperandCount(len(operands[i])))
		for j, text := range operands[i] {
			expectations = append(expectations, child.OperandAt(j).Text(text))
		}
	}
	clitest.Schema(app).Require(t, expectations...)
}

func TestProjectRuleOperand(t *testing.T) {
	t.Parallel()
	type config = schemaCountConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			count := root.Flags.Count(&root.Config.Count)
			count.Long("verbose")
			root.Rule.Equal(count, 2)
		},
	})
	rule := clitest.RootCommand().RuleAt(0)
	clitest.Schema(app).Require(t,
		rule.Kind(cli.RuleEqual), rule.OperandCount(1),
		rule.OperandAt(0).Literal().Text("2"),
	)
}

func TestRuleOperandLanguage(t *testing.T) {
	t.Parallel()
	type opaque = schemaNOpaque
	type config = schemaDynamicItemsOpaqueConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(_ context.Context, _ *config, _ cli.Dependencies, values *cli.ValueDomainBuilder[string]) error {
					values.Value("x", "x")
					return nil
				},
			)
			d := root.Values.Value(&root.Config.Dynamic, domain)
			root.Rule.Equal(d, "x")
			s := root.Values.Sequence(&root.Config.Items, domain)
			root.Rule.Contains(s, "x")
			o := root.Values.Destination(&root.Config.Opaque, cli.Value.Assign[opaque]())
			root.Rule.Equal(o, opaque{
				N: 1,
			})
		},
	})
	root := clitest.RootCommand()
	clitest.Schema(app).Require(t,
		root.RulesExactly(root.RuleAt(0), root.RuleAt(1), root.RuleAt(2)),
		root.RuleAt(0).OperandCount(1), root.RuleAt(0).OperandAt(0).NoLiteral(),
		root.RuleAt(1).OperandCount(1), root.RuleAt(1).OperandAt(0).NoLiteral(),
		root.RuleAt(2).OperandCount(1), root.RuleAt(2).OperandAt(0).NoLiteral(),
	)
}

func TestProjectRuleFailure(t *testing.T) {
	t.Parallel()
	type config = schemaSItemsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			s := root.Values.String(&root.Config.S)
			root.Rule.Match(s, "x+")
			root.Rule.Equal(s, "x")
			items := root.Values.Sequence(&root.Config.Items, cli.Value.String())
			root.Rule.ItemCount(items, cli.Count.Between(1, 3))
		},
	})
	root := clitest.RootCommand()
	clitest.Schema(app).Require(t,
		root.RulesExactly(root.RuleAt(0), root.RuleAt(1), root.RuleAt(2)),
		root.RuleAt(0).Kind(cli.RuleMatch), root.RuleAt(0).Pattern("x+"),
		root.RuleAt(1).
			Kind(cli.RuleEqual),
		root.RuleAt(1).OperandCount(1), root.RuleAt(1).OperandAt(0).Literal().Text("x"),
		root.RuleAt(2).Kind(cli.RuleItemCount), root.RuleAt(2).Count(cli.Count.Between(1, 3)),
	)
}

func TestProjectValueProducerKind(t *testing.T) {
	t.Parallel()
	type config = schemaSConfig
	all := []cli.ValueProducerKind{
		cli.ValueProducerKindArgument,
		cli.ValueProducerKindRemainder,
		cli.ValueProducerKindPassThrough,
		cli.ValueProducerKindShortcutDefault,
		cli.ValueProducerKindInvocationDefault,
		cli.ValueProducerKindComputedDefault,
		cli.ValueProducerKindStaticDefault,
		cli.ValueProducerKindInitial,
		cli.ValueProducerKindSuppliedFact,
		cli.ValueProducerKindResolvedFact,
		cli.ValueProducerKindDerived,
	}
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			s := root.Values.String(&root.Config.S)
			root.Rule.When(root.Rule.ContributedBy(s, all[0], all[1:]...)).Require(s)
		},
	})
	root := clitest.RootCommand()
	activation := root.RuleAt(0).Activation()
	clitest.Schema(app).Require(t,
		activation.Kind(cli.ConditionExpressionContributedBy),
		activation.Destination(root.ValueAt(0)),
		activation.ProducerKindsExactly(all...),
	)
}

func TestProjectSignalPolicySchema(t *testing.T) {
	t.Parallel()
	type config = schemaGateConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			gate := root.Values.String(&root.Config.Gate).Sensitive()
			root.Signals.Policy(
				root.Signals.Match.Interrupt().When(root.When.Resolved(gate)).Each(),
				root.Signals.Action.Ignore(),
			)
		},
	})
	policy := clitest.RootCommand().SignalPolicyAt(0)
	clitest.Schema(app).
		Require(t, policy.Exists(), policy.Sensitive(), policy.Activation().Exists(), policy.Activation().Sensitive())
}

func TestProjectSignalSelector(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(
				root.Signals.Match.Interrupt().
					ExceptOrigin(cli.SignalOriginTerminal, cli.SignalOriginProcess, cli.SignalOriginKernel, cli.SignalOriginChild, cli.SignalOriginRuntime).
					Each(),
				root.Signals.Action.Ignore(),
			)
		},
	})
	policy := clitest.RootCommand().SignalPolicyAt(0)
	clitest.Schema(app).
		Require(t, policy.Selector().ExcludedOriginsExactly(cli.SignalOriginTerminal, cli.SignalOriginProcess, cli.SignalOriginKernel, cli.SignalOriginChild, cli.SignalOriginRuntime))
}

func TestProjectSchemaSignalOrigin(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(
				root.Signals.Match.Interrupt().
					Origin(cli.SignalOriginTerminal, cli.SignalOriginKernel, cli.SignalOriginChild, cli.SignalOriginRuntime).
					Each(),
				root.Signals.Action.Ignore(),
			)
		},
	})
	clitest.Schema(app).
		Require(t, clitest.RootCommand().SignalPolicyAt(0).Selector().OriginsExactly(cli.SignalOriginTerminal, cli.SignalOriginKernel, cli.SignalOriginChild, cli.SignalOriginRuntime))
}

func TestProjectSignalActionKind(t *testing.T) {
	t.Parallel()
	target := cli.SignalTarget[struct{}]{
		Name:    "target",
		Deliver: func(context.Context, *struct{}, cli.SignalDelivery) error { return nil },
	}
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			source := root.Signals.Source(cli.SignalSource[string]{
				Signals: []cli.SignalDescriptor[string]{
					{Signal: "relay", Fidelity: cli.SignalSourceDeliveryFidelityExact},
				},
				Open: func(context.Context) (cli.SignalStream[string], error) {
					return cli.SignalStream[string]{
						Receive: func(ctx context.Context) (cli.SignalNotification[string], error) {
							<-ctx.Done()
							return cli.SignalNotification[string]{}, ctx.Err()
						},
					}, nil
				},
			})
			root.Signals.Policy(
				root.Signals.Match.Interrupt().Each(),
				root.Signals.Action.Forward(target).ThenSend(target, source.Signal("relay")).ThenReraise(),
			)
		},
	})
	policy := clitest.RootCommand().SignalPolicyAt(0)
	clitest.Schema(app).Require(t,
		policy.ActionCount(3),
		policy.ActionAt(0).Kind(cli.SignalActionForward),
		policy.ActionAt(1).Kind(cli.SignalActionSend),
		policy.ActionAt(2).Kind(cli.SignalActionReraise),
	)
}

func TestProjectSignalAction(t *testing.T) {
	t.Parallel()
	type config = schemaSignalDelayConfig
	target := cli.SignalTarget[config]{
		Name:    "target",
		Deliver: func(context.Context, *config, cli.SignalDelivery) error { return nil },
	}
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			signal := root.Values.Destination(&root.Config.Signal, cli.Value.Assign[cli.Signal]()).Sensitive()
			delay := root.Values.Destination(&root.Config.Delay, cli.Value.Assign[time.Duration]()).Sensitive()
			root.Signals.Policy(
				root.Signals.Match.Interrupt().Each(),
				root.Signals.Action.SendFrom(target, signal).AfterFrom(delay).Abort(2),
			)
		},
	})
	root := clitest.RootCommand()
	policy := root.SignalPolicyAt(0)
	send := policy.ActionAt(0)
	abort := policy.ActionAt(1)
	clitest.Schema(app).Require(t,
		policy.Sensitive(), policy.ActionCount(2),
		send.Kind(cli.SignalActionSend), send.SignalValue(root.ValueAt(0)), send.SignalRedacted(), send.Sensitive(),
		abort.Kind(cli.SignalActionAbort),
		abort.ExitCode(2), abort.Delayed(), abort.DelayValue(root.ValueAt(1)), abort.DelayRedacted(),
	)
}

func TestProjectSignalHostCapabilities(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(root.Signals.Match.Interrupt().Each(), root.Signals.Action.Reraise())
		},
	})

	clitest.Schema(app).Require(t,
		clitest.RootCommand().SignalPolicyAt(0).RequiredSourceCapabilities(cli.SignalSourceCapabilityReraise),
	)
}
