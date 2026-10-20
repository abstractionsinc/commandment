package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestFleetSelectsStatusByDefault(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	fleet := root.Command("fleet")
	status := fleet.Command("status")
	launch := fleet.Command("launch")

	defaultRequest := cli.InspectionRequest{Args: []string{"fleet"}}
	clitest.Inspect(t.Context(), app, defaultRequest).Require(t,
		clitest.InspectionReady(),
		clitest.SelectionPathExactly(root, fleet, status),
	)
	launchRequest := cli.InspectionRequest{Args: []string{"fleet", "launch", "scout-7"}}
	clitest.Inspect(t.Context(), app, launchRequest).Require(t,
		clitest.InspectionReady(),
		clitest.SelectionPathExactly(root, fleet, launch),
	)
}
