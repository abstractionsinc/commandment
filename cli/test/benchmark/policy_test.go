package benchmark

import (
	"context"
	"fmt"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clibench"
	"github.com/abstractionsinc/commandment/clitest"
)

// BenchmarkInspectRulesAndAdvisories holds static policy projection constant while moving the same
// policies between the selected command and an unrelated sibling.
func BenchmarkInspectRulesAndAdvisories(b *testing.B) {
	for _, placement := range [...]declarationPlacement{placementUnrelated, placementSelected} {
		for _, policies := range [...]int{16, 64, 256} {
			app, request := policyFixture(b, placement, policies)
			preflightPolicyInspection(b, app, request, placement, policies)
			b.Run(fmt.Sprintf("Placement=%s/Policies/N=%d", placement, policies), func(b *testing.B) {
				b.ReportAllocs()
				clibench.Inspect(b, app, request)
			})
		}
	}
}

func policyFixture(tb testing.TB, placement declarationPlacement, policies int) (cli.Application, cli.InspectionRequest) {
	tb.Helper()
	definition := cli.Definition[emptyConfig]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[emptyConfig]) {
			root.Subcommands.Command(cli.Command[emptyConfig]{
				Name: "target",
				Declare: func(target *cli.CommandBuilder[emptyConfig]) {
					if placement == placementSelected {
						declarePolicies(target, policies, "selected")
					}
				},
			})
			root.Subcommands.Command(cli.Command[emptyConfig]{
				Name: "bulk",
				Declare: func(bulk *cli.CommandBuilder[emptyConfig]) {
					if placement == placementUnrelated {
						declarePolicies(bulk, policies, "unrelated")
					}
				},
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("policy benchmark compile: %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"target"},
	}
	return app, request
}

func preflightPolicyInspection(tb testing.TB, app cli.Application, request cli.InspectionRequest, placement declarationPlacement, policies int) {
	tb.Helper()
	proof := clitest.Inspect(tb.Context(), app, request)
	target := clitest.Command("target")
	bulk := clitest.Command("bulk")
	proof.Require(tb,
		clitest.InspectionReady(),
		clitest.SelectionPathExactly(clitest.RootCommand(), target),
	)
	if tb.Failed() {
		return
	}
	rules := policies / 2
	advisories := policies - rules
	owner := bulk
	other := target
	ruleOutcome := cli.EvaluationOutcomeUnknown
	advisoryOutcome := cli.AdvisoryOutcomeNotEvaluated
	if placement == placementSelected {
		owner = target
		other = bulk
		ruleOutcome = cli.EvaluationOutcomeSatisfied
		advisoryOutcome = cli.AdvisoryOutcomeMatched
	}
	ruleLocators := make([]clitest.RuleLocator, rules)
	advisoryLocators := make([]clitest.AdvisoryLocator, advisories)
	for index := range ruleLocators {
		rule := owner.RuleAt(index)
		ruleLocators[index] = rule
		proof.Require(tb, rule.Outcome(ruleOutcome))
	}
	for index := range advisoryLocators {
		advisory := owner.AdvisoryAt(index)
		advisoryLocators[index] = advisory
		proof.Require(tb, advisory.Outcome(advisoryOutcome))
	}
	proof.Require(tb,
		owner.RulesExactly(ruleLocators...),
		owner.AdvisoriesExactly(advisoryLocators...),
		other.RulesExactly(),
		other.AdvisoriesExactly(),
	)
}

func declarePolicies(command *cli.CommandBuilder[emptyConfig], count int, prefix string) {
	rules := count / 2
	for index := range rules {
		command.Rule.Check(cli.RuleCode(fmt.Sprintf("%s-rule-%04d", prefix, index)), func(context.Context, *emptyConfig, cli.Dependencies) (bool, error) {
			return true, nil
		})
	}
	for index := rules; index < count; index++ {
		command.Advisory.Detect(func(context.Context, *emptyConfig, cli.Dependencies) (bool, error) {
			return true, nil
		}, cli.Advisory{
			Code:     cli.AdvisoryCode(fmt.Sprintf("%s-advisory-%04d", prefix, index)),
			Severity: cli.NoticeSeverityInformation,
			Message:  "benchmark advisory",
		})
	}
}
