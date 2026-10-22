package benchmark

import (
	"context"
	"fmt"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clibench"
	"github.com/abstractionsinc/commandment/clitest"
)

// BenchmarkInspectSignalPolicies holds static signal-policy projection constant while moving the
// same policies between the selected command and an unrelated sibling.
func BenchmarkInspectSignalPolicies(b *testing.B) {
	for _, placement := range [...]declarationPlacement{placementUnrelated, placementSelected} {
		for _, policies := range [...]int{16, 64, 256} {
			app := signalApplication(b, placement, policies)
			request := cli.InspectionRequest{
				Args: []string{"target"},
			}
			preflightSignalInspection(b, app, request, placement, policies)
			b.Run(fmt.Sprintf("Placement=%s/Policies/N=%d", placement, policies), func(b *testing.B) {
				b.ReportAllocs()
				clibench.Inspect(b, app, request)
			})
		}
	}
}

func signalApplication(tb testing.TB, placement declarationPlacement, policies int) cli.Application {
	tb.Helper()
	definition := cli.Definition[emptyConfig]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[emptyConfig]) {
			source := root.Signals.Source(cli.SignalSource[string]{
				Signals: []cli.SignalDescriptor[string]{{
					Signal:   "interrupt",
					Fidelity: cli.SignalSourceDeliveryFidelityExact,
				}},
				Open: func(context.Context) (cli.SignalStream[string], error) {
					panic("signal source opened during Inspect")
				},
			})
			signal := source.Signal("interrupt")
			declarePolicies := func(command *cli.CommandBuilder[emptyConfig]) {
				for range policies {
					command.Signals.Policy(
						command.Signals.Match.On(signal).During(cli.SignalPhaseExecution).Each(),
						command.Signals.Action.Ignore(),
					)
				}
			}
			root.Subcommands.Command(cli.Command[emptyConfig]{
				Name: "target",
				Declare: func(target *cli.CommandBuilder[emptyConfig]) {
					if placement == placementSelected {
						declarePolicies(target)
					}
				},
			})
			root.Subcommands.Command(cli.Command[emptyConfig]{
				Name: "bulk",
				Declare: func(bulk *cli.CommandBuilder[emptyConfig]) {
					if placement == placementUnrelated {
						declarePolicies(bulk)
					}
				},
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("signal benchmark compile: %v", err)
	}
	return app
}

func preflightSignalInspection(tb testing.TB, app cli.Application, request cli.InspectionRequest, placement declarationPlacement, policies int) {
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
	owner := bulk
	other := target
	if placement == placementSelected {
		owner = target
		other = bulk
	}
	locators := make([]clitest.SignalPolicyLocator, policies)
	for index := range locators {
		locators[index] = owner.SignalPolicyAt(index)
	}
	proof.Require(tb,
		owner.SignalPoliciesExactly(locators...),
		other.SignalPoliciesExactly(),
	)
	if placement == placementSelected {
		proof.Require(tb, clitest.ExecutionAdmission().SignalPoliciesExactly(locators...))
		return
	}
	proof.Require(tb, clitest.ExecutionAdmission().SignalPoliciesExactly())
}
