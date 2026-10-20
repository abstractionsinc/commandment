package conformance

import (
	"errors"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

type cutoverAvailabilityConfig struct{ Value string }

type cutoverGrammarConfig struct {
	Value   string
	Values  []string
	Profile string
}

func cutoverProfileDomain(values ...string) cli.ValueDomain[string] {
	return cli.Value.Domain(cli.Value.Assign[string](), func(builder *cli.ValueDomainBuilder[string]) {
		for _, value := range values {
			builder.Value(value, value)
		}
	})
}

func TestCompileRejectsFormAvailabilityDependingOnCompetingFormOutput(t *testing.T) {
	t.Parallel()
	clitest.CompileFails(t, cli.Definition[cutoverAvailabilityConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[cutoverAvailabilityConfig]) {
			value := root.Values.String(&root.Config.Value)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[cutoverAvailabilityConfig]) {
				form.ValueFor(value, "first")
			})
			second := root.Arguments.Form(func(form *cli.ArgumentFormBuilder[cutoverAvailabilityConfig]) {
				form.Literal("second")
			})
			second.AvailableWhen(root.When.Resolved(value))
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestCompileRejectsGreedyAllocationOnFixedArgumentTerm(t *testing.T) {
	t.Parallel()
	clitest.CompileFails(t, cli.Definition[cutoverGrammarConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[cutoverGrammarConfig]) {
			root.Arguments.String(&root.Config.Value, "value").Greedy()
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestCompileRejectsNonExactFlagGrammarProfileSelector(t *testing.T) {
	t.Parallel()
	clitest.CompileFails(t, cli.Definition[cutoverGrammarConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[cutoverGrammarConfig]) {
			root.Dialect.FlagMatching(cli.NameMatchingUniquePrefix)
			profile := root.Values.Value(&root.Config.Profile, cutoverProfileDomain("a"))
			root.Values.Default(profile, "a")
			root.Flags.ValueFor(profile).Long("profile")
			root.Profiles.Switch(profile, func(cases *cli.GrammarProfileCases[cutoverGrammarConfig, string]) {
				cases.Case("a", func(*cli.CommandBuilder[cutoverGrammarConfig]) {})
			})
		},
	}).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}
