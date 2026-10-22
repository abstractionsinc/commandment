package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestHomeShortcutTargetsNavigateWithEarth(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	navigate := root.Command("navigate")
	home := root.Shortcut("home")
	clitest.Schema(app).Require(t,
		home.Exists(),
		home.Targets(navigate),
	)
	homeRequest := cli.InspectionRequest{
		Args: []string{"home"},
	}
	clitest.Inspect(t.Context(), app, homeRequest).Require(t,
		clitest.InspectionReady(),
		clitest.SelectionPathExactly(root, navigate),
		navigate.ArgumentFormAt(0).Term("destination").Value().Text("earth"),
	)
}
