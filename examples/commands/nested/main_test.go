package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestNestedSelectionFollowsFleetLaunch(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	fleet := root.Command("fleet")
	launch := fleet.Command("launch")
	request := cli.InspectionRequest{Args: []string{"fleet", "launch", "scout-7"}}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionReady(),
		clitest.SelectionPathExactly(root, fleet, launch),
		launch.ArgumentFormAt(0).Term("ship").Value().Text("scout-7"),
	)
}
