package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestNavigateSelectsTheMatchingArgumentForm(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	command := clitest.Command("navigate")
	world := command.ArgumentFormAt(0)
	coordinates := command.ArgumentFormAt(1)

	marsRequest := cli.InspectionRequest{Args: []string{"navigate", "mars"}}
	clitest.Inspect(t.Context(), app, marsRequest).Require(t,
		clitest.InspectionReady(),
		world.Selected(),
		coordinates.NotSelected(),
		world.Term("world").Value().Text("mars"),
	)
	coordinatesRequest := cli.InspectionRequest{Args: []string{"navigate", "12", "8", "4"}}
	clitest.Inspect(t.Context(), app, coordinatesRequest).Require(t,
		clitest.InspectionReady(),
		coordinates.Selected(),
		world.NotSelected(),
		coordinates.Term("x").Value().Text("12"),
		coordinates.Term("y").Value().Text("8"),
		coordinates.Term("z").Value().Text("4"),
	)
}
