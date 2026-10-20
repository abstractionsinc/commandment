package conformance

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

type authoringPolicyRelationshipFamily struct {
	name string
	kind cli.RuleKind
	add  func(*cli.CommandBuilder[authoringPolicyRelationshipFirstSecondThirdConfig], cli.Dependency, cli.Dependency, cli.Dependency)
}

type authoringPolicyRelationshipValueConfig struct{ Value string }

type authoringPolicyRelationshipFirstSecondThirdConfig struct {
	First  string
	Second string
	Third  string
}

func TestAuthoringRuleDependencyRequirednessIsPubliclyObservable(t *testing.T) {
	t.Parallel()

	type config = authoringPolicyRelationshipValueConfig
	var requiredCalls, optionalCalls, optionalThenRequiredCalls, requiredThenOptionalCalls int
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			root.Rule.Check("required", func(context.Context, *config, cli.Dependencies) (bool, error) {
				requiredCalls++
				return true, nil
			}).DependsOn(value)
			root.Rule.Check("optional", func(_ context.Context, _ *config, dependencies cli.Dependencies) (bool, error) {
				optionalCalls++
				if _, present := dependencies.Get(value); present {
					t.Fatal("optional absent dependency was reported present")
				}
				return true, nil
			}).
				DependsOn(value.OptionalDependency())
			root.Rule.Check("optional-then-required", func(context.Context, *config, cli.Dependencies) (bool, error) {
				optionalThenRequiredCalls++
				return true, nil
			}).DependsOn(value.OptionalDependency()).DependsOn(value)
			root.Rule.Check("required-then-optional", func(context.Context, *config, cli.Dependencies) (bool, error) {
				requiredThenOptionalCalls++
				return true, nil
			}).DependsOn(value).DependsOn(value.OptionalDependency())
		},
	})

	root := clitest.RootCommand()
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		root.Rule("required").Outcome(cli.EvaluationOutcomeSkipped),
		root.Rule("optional").Outcome(cli.EvaluationOutcomeSatisfied),
		root.Rule("optional-then-required").Outcome(cli.EvaluationOutcomeSkipped),
		root.Rule("required-then-optional").Outcome(cli.EvaluationOutcomeSkipped),
	)
	if requiredCalls != 0 || optionalCalls != 1 || optionalThenRequiredCalls != 0 || requiredThenOptionalCalls != 0 {
		t.Fatalf(
			"dependency callbacks = required %d optional %d optional-then-required %d required-then-optional %d, want 0/1/0/0",
			requiredCalls,
			optionalCalls,
			optionalThenRequiredCalls,
			requiredThenOptionalCalls,
		)
	}
}

func TestAuthoringRuleGroupDependencyPositionPreservesPublicIdentityAndOwnership(t *testing.T) {
	t.Parallel()

	type config = authoringPolicyRelationshipFirstSecondThirdConfig
	type family = authoringPolicyRelationshipFamily
	families := []family{
		{
			name: "ExactlyOne",
			kind: cli.RuleExactlyOne,
			add: func(root *cli.CommandBuilder[config], first, second, third cli.Dependency) {
				root.Rule.ExactlyOne(first, second).And(third)
			},
		},
		{
			name: "AllOrNone",
			kind: cli.RuleAllOrNone,
			add: func(root *cli.CommandBuilder[config], first, second, third cli.Dependency) {
				root.Rule.AllOrNone(first, second).And(third)
			},
		},
		{
			name: "AtMostOne",
			kind: cli.RuleAtMostOne,
			add: func(root *cli.CommandBuilder[config], first, second, third cli.Dependency) {
				root.Rule.AtMostOne(first, second).And(third)
			},
		},
		{
			name: "AtLeastOne",
			kind: cli.RuleAtLeastOne,
			add: func(root *cli.CommandBuilder[config], first, second, third cli.Dependency) {
				root.Rule.AtLeastOne(first, second).And(third)
			},
		},
	}
	positions := []struct {
		name  string
		index int
	}{
		{
			name:  "first",
			index: 0,
		},
		{
			name:  "second",
			index: 1,
		},
		{
			name:  "and",
			index: 2,
		},
	}

	for _, family := range families {
		for _, position := range positions {
			t.Run(family.name+"/"+position.name, func(t *testing.T) {
				t.Parallel()

				app := clitest.Compile(t, cli.Definition[config]{
					Name: "tool",
					Declare: func(root *cli.CommandBuilder[config]) {
						first := root.Values.String(&root.Config.First)
						second := root.Values.String(&root.Config.Second)
						third := root.Values.String(&root.Config.Third)
						dependencies := []cli.Dependency{first, second, third}
						switch position.index {
						case 0:
							dependencies[0] = first.OptionalDependency()
						case 1:
							dependencies[1] = second.OptionalDependency()
						case 2:
							dependencies[2] = third.OptionalDependency()
						}
						family.add(root, dependencies[0], dependencies[1], dependencies[2])
					},
				})
				root := clitest.RootCommand()
				first := root.ValueAt(0)
				second := root.ValueAt(1)
				third := root.ValueAt(2)
				clitest.Schema(app).Require(t,
					root.RulesExactly(root.RuleAt(0)),
					root.RuleAt(0).Kind(family.kind),
					root.RuleAt(0).SubjectsExactly(first, second, third),
					root.RuleAt(0).DependenciesExactly(first, second, third),
				)
			})
		}
	}

	var foreign cli.Destination[config, string]
	clitest.Compile(t, cli.Definition[config]{
		Name: "foreign",
		Declare: func(root *cli.CommandBuilder[config]) {
			foreign = root.Values.String(&root.Config.First)
		},
	})
	for _, family := range families {
		for _, position := range positions {
			t.Run("foreign/"+family.name+"/"+position.name, func(t *testing.T) {
				t.Parallel()
				proof := clitest.CompileFails(t, cli.Definition[config]{
					Name: "tool",
					Declare: func(root *cli.CommandBuilder[config]) {
						first := root.Values.String(&root.Config.First)
						second := root.Values.String(&root.Config.Second)
						third := root.Values.String(&root.Config.Third)
						dependencies := []cli.Dependency{first, second, third}
						dependencies[position.index] = foreign.OptionalDependency()
						family.add(root, dependencies[0], dependencies[1], dependencies[2])
					},
				})
				proof.Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
			})
		}
	}
}

type authoringPolicyNameLimitConfig struct {
	Name  string
	Limit int
}

type authoringPolicyNSConfig struct {
	N int
	S string
}

type authoringPolicyABSSeqASeqBMapAMapBConfig struct {
	A, B       int
	S          string
	SeqA, SeqB []string
	MapA, MapB map[string]string
}

type authoringPolicyValueConfig struct{ Value string }

func TestPolicyAuthoringProjectsDeclaredSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringPolicyNameLimitConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			name := root.Values.String(&root.Config.Name)
			limit := root.Values.Int(&root.Config.Limit)
			root.AvailableWhen(root.When.Resolved(name))
			root.VisibleWhen(root.When.Any(root.When.Equal(name, "x"), root.When.Not(root.When.Equal(name, "y"))))
			root.Rule.Require(name)
			root.Rule.When(root.Rule.ContributedBy(name, cli.ValueProducerKindFlag)).Between(limit, 1, 10)
			root.Rule.Match(name, "x.*")
			root.Rule.Check("custom", func(context.Context, *config, cli.Dependencies) (bool, error) { return true, nil }).
				Subject(name).
				DependsOn(limit.OptionalDependency())
			root.Advisory.Always(cli.Advisory{
				Code:     "always",
				Severity: cli.NoticeSeverityInformation,
				Message:  "message",
			}).Subject(name)
			root.Advisory.Detect(func(context.Context, *config, cli.Dependencies) (bool, error) { return true, nil }, cli.Advisory{
				Code:     "dynamic",
				Severity: cli.NoticeSeverityWarning,
				Message:  "warning",
			}).
				DependsOn(limit)
			root.Notice.SuppressWhen(root.When.Resolved(name), cli.NoticeMatch{
				AdvisoryCode: "dynamic",
			})
		},
	})
	root := clitest.RootCommand()
	rules := []clitest.RuleLocator{root.RuleAt(0), root.RuleAt(1), root.RuleAt(2), root.RuleAt(3)}
	advisories := []clitest.AdvisoryLocator{root.AdvisoryAt(0), root.AdvisoryAt(1)}
	clitest.Schema(app).Require(t,
		root.ConditionAt(0).Exists(),
		root.ConditionAt(1).Exists(),
		root.RulesExactly(rules...),
		rules[0].Kind(cli.RuleRequire),
		rules[1].Kind(cli.RuleBetween),
		rules[1].Activation().Exists(),
		rules[2].Kind(cli.RuleMatch),
		rules[3].Code("custom"),
		root.AdvisoriesExactly(advisories...),
		advisories[0].Code("always"),
		advisories[1].Code("dynamic"),
		root.NoticeSuppressionsExactly(root.NoticeSuppressionAt(0)),
	)
}

func TestPolicyAuthoringProjectsConditionKinds(t *testing.T) {
	t.Parallel()
	type config = authoringPolicyNSConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			n := root.Values.Int(&root.Config.N)
			root.AvailableWhen(root.When.All(
				root.When.NotEqual(n, 1),
				root.When.In(n, 1, 2),
				root.When.NotIn(n, 3, 4),
				root.When.GreaterThan(n, 5),
				root.When.AtLeast(n, 6),
				root.When.LessThan(n, 7),
				root.When.AtMost(n, 8),
			))
			root.VisibleWhen(root.When.Matches(root.Values.String(&root.Config.S), "x.*"))
		},
	})
	root := clitest.RootCommand()
	availability := root.ConditionAt(0)
	visibility := root.ConditionAt(1)
	all := availability.Expression()
	clitest.Schema(app).Require(t,
		root.ConditionsExactly(availability, visibility),
		all.Kind(cli.ConditionExpressionAll),
		all.ChildCount(7),
		all.ChildAt(0).Kind(cli.ConditionExpressionNotEqual),
		all.ChildAt(1).Kind(cli.ConditionExpressionIn),
		all.ChildAt(2).Kind(cli.ConditionExpressionNotIn),
		all.ChildAt(3).Kind(cli.ConditionExpressionGreaterThan),
		all.ChildAt(4).Kind(cli.ConditionExpressionAtLeast),
		all.ChildAt(5).Kind(cli.ConditionExpressionLessThan),
		all.ChildAt(6).Kind(cli.ConditionExpressionAtMost),
		visibility.Expression().Kind(cli.ConditionExpressionMatch),
		visibility.Expression().Pattern("x.*"),
	)
}

func TestPolicyAuthoringProjectsRuleKindsAndComposition(t *testing.T) {
	t.Parallel()
	type config = authoringPolicyABSSeqASeqBMapAMapBConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			a := root.Values.Int(&root.Config.A)
			b := root.Values.Int(&root.Config.B)
			s := root.Values.String(&root.Config.S)
			seqA := root.Values.Sequence(&root.Config.SeqA, cli.Value.String())
			seqB := root.Values.Sequence(&root.Config.SeqB, cli.Value.String())
			mapA := root.Values.Keyed(&root.Config.MapA, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
			mapB := root.Values.Keyed(&root.Config.MapB, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
			root.Flags.ValueFor(a).Long("policy-a")
			root.Flags.ValueFor(b).Long("policy-b")
			root.Flags.ValueFor(s).Long("policy-s")
			root.Flags.Sequence(seqA).Long("policy-seq-a")
			root.Flags.Sequence(seqB).Long("policy-seq-b")
			root.Flags.Keyed(mapA).Long("policy-map-a")
			root.Flags.Keyed(mapB).Long("policy-map-b")
			root.Rule.Forbid(a)
			root.Rule.NotEmpty(s)
			root.Rule.In(a, 1, 2)
			root.Rule.NotIn(a, 3, 4)
			root.Rule.Equal(a, 5)
			root.Rule.NotEqual(a, 6)
			root.Rule.AtLeast(a, 7)
			root.Rule.AtMost(a, 8)
			root.Rule.GreaterThan(a, 9)
			root.Rule.LessThan(a, 10)
			root.Rule.Contains(seqA, "a")
			root.Rule.EachIn(seqA, "a", "b")
			root.Rule.Unique(seqA)
			root.Rule.EntryCount(mapA, cli.Count.Exactly(2))
			root.Rule.HasKeys(mapA, "a")
			root.Rule.KeysIn(mapA, "a", "b")
			root.Rule.Compare.Equal(a, b)
			root.Rule.Compare.NotEqual(a, b)
			root.Rule.Compare.LessThan(a, b)
			root.Rule.Compare.AtMost(a, b)
			root.Rule.Compare.GreaterThan(a, b)
			root.Rule.Compare.AtLeast(a, b)
			root.Rule.Compare.In(s, seqA)
			root.Rule.Compare.NotIn(s, seqA)
			root.Rule.Compare.SubsetOf(seqA, seqB)
			root.Rule.Compare.Disjoint(seqA, seqB)
			root.Rule.Compare.KeysSubsetOf(mapA, mapB)
			root.Rule.Compare.KeysDisjoint(mapA, mapB)
			root.Rule.ExactlyOne(a, b).And(s)
			root.Rule.AllOrNone(a, b).And(s)
			root.Rule.AtMostOne(a, b).And(s)
			root.Rule.AtLeastOne(a, b).And(s)
			structural := root.When.Resolved(a)
			root.Rule.When(root.Rule.All(structural, root.Rule.ContributedBy(a, cli.ValueProducerKindFlag))).Require(a)
			root.Rule.Unless(root.Rule.Any(structural, root.Rule.Not(root.Rule.ContributedBy(a, cli.ValueProducerKindEnvironment)))).
				Require(b)
			root.Rule.Validate("validate", func(context.Context, *config, cli.Dependencies) ([]cli.Violation, error) { return nil, nil }).
				Subject(a)
			root.Advisory.When(structural, cli.Advisory{
				Code:     "conditional",
				Severity: cli.NoticeSeverityInformation,
				Message:  "conditional",
			}).Subject(a)
		},
	})
	root := clitest.RootCommand()
	a := root.Flag("--policy-a").Value()
	b := root.Flag("--policy-b").Value()
	s := root.Flag("--policy-s").Value()
	seqA := root.Flag("--policy-seq-a").Value()
	seqB := root.Flag("--policy-seq-b").Value()
	mapA := root.Flag("--policy-map-a").Value()
	mapB := root.Flag("--policy-map-b").Value()
	rules := make([]clitest.RuleLocator, 35)
	for i := range rules {
		rules[i] = root.RuleAt(i)
	}
	clitest.Schema(app).Require(t,
		root.RulesExactly(rules...),
		rules[0].Kind(cli.RuleForbid),
		rules[1].Kind(cli.RuleNotEmpty), rules[2].Kind(cli.RuleIn), rules[3].Kind(cli.RuleNotIn),
		rules[4].Kind(cli.RuleEqual),
		rules[5].Kind(cli.RuleNotEqual), rules[6].Kind(cli.RuleAtLeast), rules[7].Kind(cli.RuleAtMost),
		rules[8].Kind(cli.RuleGreaterThan),
		rules[9].Kind(cli.RuleLessThan), rules[10].Kind(cli.RuleContains), rules[11].Kind(cli.RuleEachIn),
		rules[12].Kind(cli.RuleUnique),
		rules[13].Kind(cli.RuleEntryCount), rules[14].Kind(cli.RuleHasKeys), rules[15].Kind(cli.RuleKeysIn),
		rules[16].Kind(cli.RuleEqual),
		rules[17].Kind(cli.RuleNotEqual), rules[18].Kind(cli.RuleLessThan), rules[19].Kind(cli.RuleAtMost),
		rules[20].Kind(cli.RuleGreaterThan),
		rules[21].Kind(cli.RuleAtLeast), rules[22].Kind(cli.RuleIn), rules[23].Kind(cli.RuleNotIn),
		rules[24].Kind(cli.RuleSubsetOf),
		rules[25].Kind(cli.RuleDisjoint), rules[26].Kind(cli.RuleKeysSubsetOf), rules[27].Kind(cli.RuleKeysDisjoint),
		rules[28].Kind(cli.RuleExactlyOne),
		rules[29].Kind(cli.RuleAllOrNone), rules[30].Kind(cli.RuleAtMostOne), rules[31].Kind(cli.RuleAtLeastOne),
		rules[32].Kind(cli.RuleRequire),
		rules[33].Kind(cli.RuleRequire), rules[34].Kind(cli.RuleCustom), rules[34].Code("validate"),
		rules[16].SubjectsExactly(a, b),
		rules[17].SubjectsExactly(a, b), rules[18].SubjectsExactly(a, b), rules[19].SubjectsExactly(a, b),
		rules[20].SubjectsExactly(a, b),
		rules[21].SubjectsExactly(a, b), rules[22].SubjectsExactly(s, seqA), rules[23].SubjectsExactly(s, seqA),
		rules[24].SubjectsExactly(seqA, seqB),
		rules[25].SubjectsExactly(seqA, seqB),
		rules[26].SubjectsExactly(mapA, mapB), rules[27].SubjectsExactly(mapA, mapB),
		rules[28].DependenciesExactly(a, b, s),
		rules[29].DependenciesExactly(a, b, s),
		rules[30].DependenciesExactly(a, b, s), rules[31].DependenciesExactly(a, b, s),
		rules[32].Activation().Kind(cli.ConditionExpressionAll), rules[32].Activation().ChildCount(2),
		rules[32].Activation().
			ChildAt(0).
			Kind(cli.ConditionExpressionResolved),
		rules[32].Activation().ChildAt(0).Destination(a),
		rules[32].Activation().
			ChildAt(1).
			Kind(cli.ConditionExpressionContributedBy),
		rules[32].Activation().
			ChildAt(1).
			Destination(a),
		rules[32].Activation().ChildAt(1).ProducerKindsExactly(cli.ValueProducerKindFlag),
		rules[33].Activation().Kind(cli.ConditionExpressionNot), rules[33].Activation().ChildCount(1),
		rules[33].Activation().
			ChildAt(0).
			Kind(cli.ConditionExpressionAny),
		rules[33].Activation().ChildAt(0).ChildCount(2),
		rules[33].Activation().
			ChildAt(0).
			ChildAt(0).
			Kind(cli.ConditionExpressionResolved),
		rules[33].Activation().ChildAt(0).ChildAt(0).Destination(a),
		rules[33].Activation().
			ChildAt(0).
			ChildAt(1).
			Kind(cli.ConditionExpressionNot),
		rules[33].Activation().ChildAt(0).ChildAt(1).ChildCount(1),
		rules[33].Activation().ChildAt(0).ChildAt(1).ChildAt(0).Kind(cli.ConditionExpressionContributedBy),
		rules[33].Activation().ChildAt(0).ChildAt(1).ChildAt(0).Destination(a),
		rules[33].Activation().ChildAt(0).ChildAt(1).ChildAt(0).ProducerKindsExactly(cli.ValueProducerKindEnvironment),
		root.AdvisoriesExactly(root.AdvisoryAt(0)),
		root.AdvisoryAt(0).Kind(cli.AdvisoryKindConditional),
		root.AdvisoryAt(0).Activation().Kind(cli.ConditionExpressionResolved),
		root.AdvisoryAt(0).Activation().Destination(a),
	)
}

func TestMatchesRejectsInvalidPattern(t *testing.T) {
	t.Parallel()
	type config = authoringPolicyValueConfig
	clitest.CompileFails(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			root.VisibleWhen(root.When.Matches(value, "["))
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

type policySelectorConfig struct{ Selector string }

type policyNameTagsConfig struct {
	Name string
	Tags []string
}

type policyValueIntConfig struct{ Value int }

type policyValueStringConfig struct{ Value string }

func declarePolicyFixture(root *cli.CommandBuilder[policySelectorConfig]) {
	selector := root.Values.String(&root.Config.Selector)
	condition := root.When.Equal(selector, "on")
	root.Flags.ValueFor(selector).Long("selector")
	root.Subcommands.Command(cli.Command[struct{}]{
		Name: "child",
	}).DefaultWhen(condition)
}

func TestConditionalCommandDefaultLowersAndProjectsCondition(t *testing.T) {
	t.Parallel()
	type config = policySelectorConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "conditional-default",
		Declare: func(root *cli.CommandBuilder[config]) {
			selector := root.Values.String(&root.Config.Selector)
			condition := root.When.Equal(selector, "on")
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "child",
			}).DefaultWhen(condition)
		},
	})
	root := clitest.RootCommand()
	selector := root.ValueAt(0)
	def := root.DefaultAt(0)
	clitest.Schema(app).Require(t,
		root.DefaultsExactly(def),
		def.Target(root.Command("child")),
		def.Condition().Kind(cli.ConditionExpressionEqual),
		def.Condition().Destination(selector),
		def.Condition().OperandCount(1),
		def.Condition().OperandAt(0).Text("on"),
	)
}

func TestAuthoringAcceptanceRetainsDestinationResolutionPolicyDimensions(t *testing.T) {
	t.Parallel()
	type config = policyNameTagsConfig
	var got config
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "resolution",
		Declare: func(root *cli.CommandBuilder[config]) {
			name := root.Values.String(&root.Config.Name)
			tags := root.Values.Sequence(&root.Config.Tags, cli.Value.String())
			root.Values.ResolveWith(name, cli.Value.KeepLast())
			root.Values.ResolveWith(tags, cli.Value.CombineTiers(cli.ResolutionTierLowerFirst))
			root.Values.Environment(name, "NAME")
			root.Values.Environment(tags, "TAGS").Elements(",")
			root.Flags.ValueFor(name).Long("name")
			root.Flags.Sequence(tags).Long("tag")
		},
		Run: func(_ context.Context, c *config) error { got = *c; return nil },
	})
	root := clitest.RootCommand()
	name := root.Flag("--name").Value()
	tags := root.Flag("--tag").Value()
	flagTags := tags.CandidateAt(0)
	environmentTags := tags.CandidateAt(1)
	request := cli.InspectionRequest{
		Env: cli.Env{{
			Name:  "NAME",
			Value: "env",
		}, {
			Name:  "TAGS",
			Value: "a,b",
		}},
		Args: []string{"--name", "flag", "--tag", "c"},
	}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.DiagnosticsExactly(),
		name.Resolved(),
		name.Text("flag"),
		name.ContributedBy(cli.ValueProducerKindFlag),
		tags.Resolved(),
		tags.CandidatesExactly(flagTags, environmentTags),
		tags.ContributorsExactly(flagTags, environmentTags),
		tags.ElementsExactly(environmentTags.ElementAt(0), environmentTags.ElementAt(1), flagTags.ElementAt(0)),
		environmentTags.ElementAt(0).Text("a"),
		environmentTags.ElementAt(1).Text("b"),
		flagTags.ElementAt(0).Text("c"),
		tags.ContributedBy(cli.ValueProducerKindEnvironment),
		tags.ContributedBy(cli.ValueProducerKindFlag),
	)
	if err := app.Run(t.Context(), cli.RunRequest{
		Env:  request.Env,
		Args: request.Args,
	}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got.Name != "flag" || !slices.Equal(got.Tags, []string{"a", "b", "c"}) {
		t.Fatalf("resolved config = %#v", got)
	}
}

func TestResolutionPolicyValuesRetainExactDeclaredPolicy(t *testing.T) {
	t.Parallel()
	type config = policyNameTagsConfig
	var got config
	namePolicies := []cli.ResolutionPolicy{cli.Value.KeepLast()}
	combine := func(order cli.ResolutionTierOrder) cli.ResolutionPolicy {
		return cli.Value.CombineTiers(order)
	}
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "policies",
		Declare: func(root *cli.CommandBuilder[config]) {
			name := root.Values.String(&root.Config.Name)
			tags := root.Values.Sequence(&root.Config.Tags, cli.Value.String())
			root.Values.ResolveWith(name, namePolicies...)
			root.Values.ResolveWith(tags, combine(cli.ResolutionTierLowerFirst))
			root.Values.Environment(name, "NAME")
			root.Values.Environment(tags, "TAGS").Elements(",")
			root.Flags.ValueFor(name).Long("name")
			root.Flags.Sequence(tags).Long("tag")
		},
		Run: func(_ context.Context, c *config) error { got = *c; return nil },
	})
	root := clitest.RootCommand()
	name := root.Flag("--name").Value()
	tags := root.Flag("--tag").Value()
	flagTags := tags.CandidateAt(0)
	environmentTags := tags.CandidateAt(1)
	request := cli.InspectionRequest{
		Env: cli.Env{{
			Name:  "NAME",
			Value: "env",
		}, {
			Name:  "TAGS",
			Value: "a,b",
		}},
		Args: []string{"--name", "flag", "--tag", "c"},
	}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.DiagnosticsExactly(),
		name.Resolved(),
		name.Text("flag"),
		name.ContributedBy(cli.ValueProducerKindFlag),
		tags.Resolved(),
		tags.CandidatesExactly(flagTags, environmentTags),
		tags.ContributorsExactly(flagTags, environmentTags),
		tags.ElementsExactly(environmentTags.ElementAt(0), environmentTags.ElementAt(1), flagTags.ElementAt(0)),
		environmentTags.ElementAt(0).Text("a"),
		environmentTags.ElementAt(1).Text("b"),
		flagTags.ElementAt(0).Text("c"),
		tags.ContributedBy(cli.ValueProducerKindEnvironment),
		tags.ContributedBy(cli.ValueProducerKindFlag),
	)
	if err := app.Run(t.Context(), cli.RunRequest{
		Env:  request.Env,
		Args: request.Args,
	}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got.Name != "flag" || !slices.Equal(got.Tags, []string{"a", "b", "c"}) {
		t.Fatalf("resolved=%#v", got)
	}
	for _, policy := range []cli.ResolutionPolicy{
		{},
		cli.Value.CombineTiers(0),
	} {
		clitest.CompileFails(t, cli.Definition[config]{
			Name: "invalid-policy",
			Declare: func(root *cli.CommandBuilder[config]) {
				d := root.Values.String(&root.Config.Name)
				root.Values.ResolveWith(d, policy)
			},
		}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
	}
}

func TestConditionValuesAreInertImmutableExpressions(t *testing.T) {
	t.Parallel()
	type config = policyValueIntConfig
	operands := []int{1, 2, 3}
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "conditions",
		Declare: func(root *cli.CommandBuilder[config]) {
			d := root.Values.Int(&root.Config.Value)
			equal := root.When.Equal(d, 7)
			in := root.When.In(d, operands...)
			root.VisibleWhen(root.When.All(equal, in))
			operands[0] = 99
		},
	})
	root := clitest.RootCommand()
	value := root.ValueAt(0)
	expr := root.ConditionAt(0).Expression()
	clitest.Schema(app).Require(t,
		root.ConditionsExactly(root.ConditionAt(0)),
		expr.Kind(cli.ConditionExpressionAll),
		expr.ChildCount(2),
		expr.ChildAt(0).Kind(cli.ConditionExpressionEqual),
		expr.ChildAt(0).Destination(value),
		expr.ChildAt(0).OperandCount(1),
		expr.ChildAt(0).OperandAt(0).Text("7"),
		expr.ChildAt(1).Kind(cli.ConditionExpressionIn),
		expr.ChildAt(1).Destination(value),
		expr.ChildAt(1).OperandCount(3),
		expr.ChildAt(1).OperandAt(0).Text("1"),
		expr.ChildAt(1).OperandAt(1).Text("2"),
		expr.ChildAt(1).OperandAt(2).Text("3"),
	)
}

func TestConditionExpressionsRejectForeignAuthoringScopeWithoutCommittingAnything(t *testing.T) {
	t.Parallel()
	type config = policyValueIntConfig
	var foreign cli.Destination[config, int]
	clitest.Compile(t, cli.Definition[config]{
		Name: "owner",
		Declare: func(root *cli.CommandBuilder[config]) {
			foreign = root.Values.Int(&root.Config.Value)
		},
	})
	clitest.CompileFails(t, cli.Definition[config]{
		Name: "foreign-condition",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.VisibleWhen(root.When.Equal(foreign, 1))
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestDefinitionAuthoringRejectsMalformedStaticPolicy(t *testing.T) {
	t.Parallel()
	type config = policyValueStringConfig
	tests := []struct {
		name    string
		declare func(*cli.CommandBuilder[config])
	}{
		{"invalid Rule code", func(root *cli.CommandBuilder[config]) {
			root.Rule.Check(
				"Bad Code",
				func(context.Context, *config, cli.Dependencies) (bool, error) { return true, nil },
			)
		}},
		{"invalid regexp", func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			root.Rule.Match(value, "[")
		}},
		{"invalid Advisory", func(root *cli.CommandBuilder[config]) {
			root.Advisory.Always(cli.Advisory{
				Code:     "bad",
				Severity: cli.NoticeSeverityUnknown,
				Message:  "message",
			})
		}},
		{"zero Notice match", func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			root.Notice.SuppressWhen(root.When.Resolved(value), cli.NoticeMatch{})
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			clitest.CompileFails(t, cli.Definition[config]{
				Name:    "commandment",
				Declare: tc.declare,
			}).Require(t,
				clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present(),
			)
		})
	}
}

func TestApplyPolicyAnalysisPreservesPolicySemantics(t *testing.T) {
	t.Parallel()
	type config = policySelectorConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declarePolicyFixture,
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--selector", "on"},
	}).Require(t,
		clitest.DiagnosticsExactly(),
	)
}

func TestRuleInspectionIndexPreservesPolicySemantics(t *testing.T) {
	t.Parallel()
	type config = policySelectorConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declarePolicyFixture,
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--selector", "on"},
	}).Require(t,
		clitest.DiagnosticsExactly(),
	)
}

func TestAdvisoryInspectionIndexPreservesPolicySemantics(t *testing.T) {
	t.Parallel()
	type config = policySelectorConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declarePolicyFixture,
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--selector", "on"},
	}).Require(t,
		clitest.DiagnosticsExactly(),
	)
}

func TestSuppressionInspectionIndexPreservesPolicySemantics(t *testing.T) {
	t.Parallel()
	type config = policySelectorConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declarePolicyFixture,
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--selector", "on"},
	}).Require(t,
		clitest.DiagnosticsExactly(),
	)
}

func TestPolicyRuleDiagnosticPreservesPolicySemantics(t *testing.T) {
	t.Parallel()
	type config = policySelectorConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declarePolicyFixture,
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--selector", "on"},
	}).Require(t,
		clitest.DiagnosticsExactly(),
	)
}

func TestPolicyRuleFailureDiagnosticPreservesPolicySemantics(t *testing.T) {
	t.Parallel()
	type config = policySelectorConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declarePolicyFixture,
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--selector", "on"},
	}).Require(t,
		clitest.DiagnosticsExactly(),
	)
}

func TestPolicyViolationKindPreservesPolicySemantics(t *testing.T) {
	t.Parallel()
	type config = policySelectorConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declarePolicyFixture,
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--selector", "on"},
	}).Require(t,
		clitest.DiagnosticsExactly(),
	)
}

func TestPolicyDiagnosticFailurePreservesPolicySemantics(t *testing.T) {
	t.Parallel()
	type config = policySelectorConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declarePolicyFixture,
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--selector", "on"},
	}).Require(t,
		clitest.DiagnosticsExactly(),
	)
}

func TestDiagnosticProjectRuleFailurePreservesPolicySemantics(t *testing.T) {
	t.Parallel()
	type config = policySelectorConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declarePolicyFixture,
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--selector", "on"},
	}).Require(t,
		clitest.DiagnosticsExactly(),
	)
}

func TestDiagnosticProjectValueTypePreservesPolicySemantics(t *testing.T) {
	t.Parallel()
	type config = policySelectorConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declarePolicyFixture,
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--selector", "on"},
	}).Require(t,
		clitest.DiagnosticsExactly(),
	)
}

func TestPrivatePolicyCommandIDPreservesPolicySemantics(t *testing.T) {
	t.Parallel()
	type config = policySelectorConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declarePolicyFixture,
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--selector", "on"},
	}).Require(t,
		clitest.DiagnosticsExactly(),
	)
}

func TestDiagnosticProjectRuleKindPreservesPolicySemantics(t *testing.T) {
	t.Parallel()
	type config = policySelectorConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declarePolicyFixture,
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--selector", "on"},
	}).Require(t,
		clitest.DiagnosticsExactly(),
	)
}

func TestDiagnosticProjectViolationKindPreservesPolicySemantics(t *testing.T) {
	t.Parallel()
	type config = policySelectorConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declarePolicyFixture,
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--selector", "on"},
	}).Require(t,
		clitest.DiagnosticsExactly(),
	)
}

func TestPolicyFailureDetailsPreservesPolicySemantics(t *testing.T) {
	t.Parallel()
	type config = policySelectorConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declarePolicyFixture,
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--selector", "on"},
	}).Require(t,
		clitest.DiagnosticsExactly(),
	)
}

func TestPolicyAnalysisFailuresPreservePolicy(t *testing.T) {
	t.Parallel()

	t.Run("rule evaluation failure", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("rule failed")
		app := clitest.Compile(t, cli.Definition[struct{}]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[struct{}]) {
				root.Rule.Check(
					"failed",
					func(context.Context, *struct{}, cli.Dependencies) (bool, error) { return false, cause },
				)
			},
			Run: func(context.Context, *struct{}) error { return nil },
		})
		diagnostic := clitest.Diagnostic(cli.DiagnosticRuleEvaluationFailure).Rule("failed")
		proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{})
		proof.Require(t,
			clitest.FailsWith(diagnostic.Present()),
			clitest.RootCommand().RuleAt(0).Outcome(cli.EvaluationOutcomeFailed),
			diagnostic.RuleKind(cli.RuleCustom),
		)
		_, inspectErr := proof.Inspection()
		if !errors.Is(inspectErr, cause) {
			t.Fatalf("Inspect() error = %v, want cause", inspectErr)
		}
		runErr := app.Run(t.Context(), cli.RunRequest{})
		if !errors.Is(runErr, cause) {
			t.Fatalf("Run() error = %v, want cause", runErr)
		}
		clitest.Diagnostics(runErr).Require(t, diagnostic.Present(), diagnostic.RuleKind(cli.RuleCustom))
	})

	t.Run("invalid rule result", func(t *testing.T) {
		t.Parallel()
		app := clitest.Compile(t, cli.Definition[struct{}]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[struct{}]) {
				root.Rule.Validate(
					"invalid",
					func(context.Context, *struct{}, cli.Dependencies) ([]cli.Violation, error) {
						return []cli.Violation{{Kind: cli.ViolationUnknown}}, nil
					},
				)
			},
			Run: func(context.Context, *struct{}) error { return nil },
		})
		diagnostic := clitest.Diagnostic(cli.DiagnosticInvalidRuleResult).Rule("invalid")
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
			clitest.FailsWith(diagnostic.Present()),
			diagnostic.RuleKind(cli.RuleCustom),
		)
		runErr := app.Run(t.Context(), cli.RunRequest{})
		if runErr == nil {
			t.Fatal("Run() error = nil, want invalid rule result")
		}
		clitest.Diagnostics(runErr).Require(t, diagnostic.Present(), diagnostic.RuleKind(cli.RuleCustom))
	})

	t.Run("advisory evaluation failure", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("advisory failed")
		app := clitest.Compile(t, cli.Definition[struct{}]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[struct{}]) {
				root.Advisory.Detect(
					func(context.Context, *struct{}, cli.Dependencies) (bool, error) { return false, cause },
					cli.Advisory{
						Code:     "failed",
						Severity: cli.NoticeSeverityWarning,
						Message:  "failed",
					},
				)
			},
			Run: func(context.Context, *struct{}) error { return nil },
		})
		diagnostic := clitest.Diagnostic(cli.DiagnosticAdvisoryEvaluationFailure).Advisory("failed")
		proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{})
		proof.Require(t, clitest.FailsWith(diagnostic.Present()))
		_, inspectErr := proof.Inspection()
		if !errors.Is(inspectErr, cause) {
			t.Fatalf("Inspect() error = %v, want cause", inspectErr)
		}
		runErr := app.Run(t.Context(), cli.RunRequest{})
		if !errors.Is(runErr, cause) {
			t.Fatalf("Run() error = %v, want cause", runErr)
		}
		clitest.Diagnostics(runErr).Require(t, diagnostic.Present())
	})
}

func TestPolicyFailureDetailsTruncatesViolations(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Rule.Validate("many", func(context.Context, *struct{}, cli.Dependencies) ([]cli.Violation, error) {
				return []cli.Violation{{Kind: cli.ViolationInvalid}, {Kind: cli.ViolationInvalid}}, nil
			})
		},
		Run: func(context.Context, *struct{}) error { return nil },
	})
	err := app.Run(t.Context(), cli.RunRequest{
		Limits: cli.AnalysisLimits{
			Diagnostics: 1,
		},
	})
	if err == nil {
		t.Fatal("Run() error = nil, want rule violation")
	}
	diagnostic := clitest.DiagnosticAt(0)
	clitest.Diagnostics(err).Require(t,
		diagnostic.Code(cli.DiagnosticRuleViolation),
		diagnostic.Rule("many").RuleKind(cli.RuleCustom),
		clitest.DiagnosticsTruncated(),
	)
}

func TestPolicySkippedRuleAndUnmatchedAdvisoryPreservePublic(t *testing.T) {
	t.Parallel()
	type config = policyValueStringConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			root.Values.Default(value, "default")
			root.Rule.When(root.Rule.ContributedBy(value, cli.ValueProducerKindEnvironment)).Require(value)
			root.Advisory.Detect(func(context.Context, *config, cli.Dependencies) (bool, error) {
				return false, nil
			}, cli.Advisory{
				Code:     "not-matched",
				Severity: cli.NoticeSeverityInformation,
				Message:  "not matched",
			})
		},
	})
	root := clitest.RootCommand()
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		root.RuleAt(0).ActivationOutcome(cli.EvaluationOutcomeUnsatisfied),
		root.RuleAt(0).Outcome(cli.EvaluationOutcomeSkipped),
		root.Advisory("not-matched").Outcome(cli.AdvisoryOutcomeNotMatched),
	)
}

func TestPolicyBlockedDependenciesPreservePublic(t *testing.T) {
	t.Parallel()
	type config = policyValueStringConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			root.Rule.Check("blocked", func(context.Context, *config, cli.Dependencies) (bool, error) {
				t.Fatal("blocked rule callback executed")
				return true, nil
			}).DependsOn(value)
			root.Advisory.Detect(func(context.Context, *config, cli.Dependencies) (bool, error) {
				t.Fatal("blocked advisory callback executed")
				return true, nil
			}, cli.Advisory{
				Code:     "blocked",
				Severity: cli.NoticeSeverityInformation,
				Message:  "blocked",
			}).DependsOn(value)
		},
	})
	root := clitest.RootCommand()
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		root.Rule("blocked").Outcome(cli.EvaluationOutcomeSkipped),
		root.Advisory("blocked").Outcome(cli.AdvisoryOutcomeBlocked),
	)
}

type profileValuePolicyCompositionProfileActiveSeedInactiveSeedSelectedConfig struct {
	Profile      string
	ActiveSeed   string
	InactiveSeed string
	Selected     string
}

func TestSelectedProfileEvaluatesOnlyItsValueAndPolicyGraph(t *testing.T) {
	t.Parallel()
	type config = profileValuePolicyCompositionProfileActiveSeedInactiveSeedSelectedConfig

	var activeSeedCalls, activeValueCalls, inactiveSeedCalls, inactiveValueCalls int
	var activeRuleCalls, activeAdvisoryCalls, inactiveRuleCalls, inactiveAdvisoryCalls int
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			profileDomain := cli.Value.Domain(cli.Value.Assign[string](), func(domain *cli.ValueDomainBuilder[string]) {
				domain.Value("active", "active")
				domain.Value("inactive", "inactive")
			})
			profile := root.Values.Value(&root.Config.Profile, profileDomain)
			root.Values.Default(profile, "active")
			activeSeed := root.Values.String(&root.Config.ActiveSeed)
			inactiveSeed := root.Values.String(&root.Config.InactiveSeed)
			selected := root.Values.String(&root.Config.Selected)

			root.Profiles.Switch(profile, func(cases *cli.GrammarProfileCases[config, string]) {
				cases.Case("active", func(active *cli.CommandBuilder[config]) {
					active.Values.ComputedDefault(
						activeSeed,
						cli.Value.ComputedBy(func(context.Context, *config, cli.Dependencies) (string, error) {
							activeSeedCalls++
							return "active-seed", nil
						}),
					)
					active.Values.ComputedDefault(
						selected,
						cli.Value.ComputedBy(func(_ context.Context, _ *config, dependencies cli.Dependencies) (string, error) {
							activeValueCalls++
							seed, ok := dependencies.Get(activeSeed)
							if !ok || seed != "active-seed" {
								t.Fatalf("active selected-value dependency = %q, %v", seed, ok)
							}
							return seed + ":selected", nil
						}).
							From(activeSeed),
					)
					active.Rule.Check("active-selected", func(_ context.Context, _ *config, dependencies cli.Dependencies) (bool, error) {
						activeRuleCalls++
						value, ok := dependencies.Get(selected)
						return ok && value == "active-seed:selected", nil
					}).
						Subject(selected).
						DependsOn(selected)
					active.Advisory.Detect(func(_ context.Context, _ *config, dependencies cli.Dependencies) (bool, error) {
						activeAdvisoryCalls++
						value, ok := dependencies.Get(selected)
						return ok && value == "active-seed:selected", nil
					}, cli.Advisory{
						Code:     "active-selected",
						Severity: cli.NoticeSeverityInformation,
						Message:  "active selected",
					}).
						Subject(selected).
						DependsOn(selected)
				})
				cases.Default(func(inactive *cli.CommandBuilder[config]) {
					inactive.Values.ComputedDefault(
						inactiveSeed,
						cli.Value.ComputedBy(func(context.Context, *config, cli.Dependencies) (string, error) {
							inactiveSeedCalls++
							return "inactive-seed", nil
						}),
					)
					inactive.Values.ComputedDefault(
						selected,
						cli.Value.ComputedBy(func(_ context.Context, _ *config, dependencies cli.Dependencies) (string, error) {
							inactiveValueCalls++
							seed, _ := dependencies.Get(inactiveSeed)
							return seed + ":selected", nil
						}).
							From(inactiveSeed),
					)
					inactive.Rule.Check("inactive-selected", func(context.Context, *config, cli.Dependencies) (bool, error) {
						inactiveRuleCalls++
						return false, nil
					}).
						Subject(selected).
						DependsOn(selected)
					inactive.Advisory.Detect(func(context.Context, *config, cli.Dependencies) (bool, error) {
						inactiveAdvisoryCalls++
						return true, nil
					}, cli.Advisory{
						Code:     "inactive-selected",
						Severity: cli.NoticeSeverityWarning,
						Message:  "inactive selected",
					}).Subject(selected).DependsOn(selected)
				})
			})
		},
	})

	root := clitest.RootCommand()
	profile := root.ValueAt(0)
	activeSeed := root.ValueAt(1)
	selected := root.ValueAt(3)
	producer := selected.ProducerAt(0)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		clitest.InspectionReady(),
		profile.Resolved(), profile.Text("active"),
		activeSeed.Resolved(), activeSeed.Text("active-seed"),
		selected.Resolved(), selected.Text("active-seed:selected"),
		selected.ProducersExactly(producer),
		producer.Kind(cli.ValueProducerKindComputedDefault),
		producer.Outcome(cli.ValueProducerOutcomeContributed), producer.DependenciesExactly(activeSeed),
		root.Rule("active-selected").Outcome(cli.EvaluationOutcomeSatisfied),
		root.Advisory("active-selected").Outcome(cli.AdvisoryOutcomeMatched),
		clitest.NoticesExactly(clitest.Notice(cli.NoticeAdvisory).Advisory("active-selected")),
	)
	if activeSeedCalls != 1 || activeValueCalls != 1 || activeRuleCalls != 1 || activeAdvisoryCalls != 1 {
		t.Fatalf(
			"active calls seed/value/rule/advisory = %d/%d/%d/%d, want 1/1/1/1",
			activeSeedCalls,
			activeValueCalls,
			activeRuleCalls,
			activeAdvisoryCalls,
		)
	}
	if inactiveSeedCalls != 0 || inactiveValueCalls != 0 || inactiveRuleCalls != 0 || inactiveAdvisoryCalls != 0 {
		t.Fatalf(
			"inactive-profile work leaked: seed/value/rule/advisory = %d/%d/%d/%d",
			inactiveSeedCalls,
			inactiveValueCalls,
			inactiveRuleCalls,
			inactiveAdvisoryCalls,
		)
	}
}
