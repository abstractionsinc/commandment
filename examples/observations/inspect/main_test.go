package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestInspectResolvesInvocationWithoutRunningHandler(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	scan := clitest.Command("scan")
	rangeValue := scan.Flag("--range").Value()
	world := scan.ArgumentFormAt(0).TermAt(0).Value()
	request := cli.InspectionRequest{
		Args: []string{"scan", "--range", "5", "mars"},
	}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionReady(),
		scan.Selected(),
		rangeValue.Text("5"),
		world.Text("mars"),
	)
}
