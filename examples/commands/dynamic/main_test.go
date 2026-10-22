package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestProviderPublishesCommandsForTheSelectedSystem(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	outpost := root.Command("outpost")
	provider := outpost.ProviderAt(0)

	jovianRequest := cli.InspectionRequest{
		Args: []string{"outpost", "--system", "jovian", "europa"},
	}
	clitest.Inspect(t.Context(), app, jovianRequest).Require(t,
		clitest.InspectionReady(),
		provider.Outcome(cli.CommandProviderOutcomeAccepted),
		clitest.SelectionPathExactly(root, outpost, outpost.Command("europa")),
	)

	saturnianRequest := cli.InspectionRequest{
		Args: []string{"outpost", "--system", "saturnian", "titan"},
	}
	clitest.Inspect(t.Context(), app, saturnianRequest).Require(t,
		clitest.InspectionReady(),
		provider.Outcome(cli.CommandProviderOutcomeAccepted),
		clitest.SelectionPathExactly(root, outpost, outpost.Command("titan")),
	)
}
