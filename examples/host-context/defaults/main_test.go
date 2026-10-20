package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestHostDefaultCanBeOverriddenByCLIInput(t *testing.T) {
	t.Parallel()

	definition, defaults := newProgram()
	app := clitest.Compile(t, definition)
	ship := clitest.Command("status").Flag("--ship").Value()
	base := cli.InspectionRequest{
		Args:     []string{"status"},
		Defaults: []cli.DefaultProvider{defaults},
	}
	clitest.Inspect(t.Context(), app, base).Require(t,
		clitest.InspectionReady(),
		ship.Text("scout-7"),
		ship.ContributedBy(cli.ValueProducerKindInvocationDefault),
	)
	override := cli.InspectionRequest{
		Args:     []string{"status", "--ship", "voyager"},
		Defaults: []cli.DefaultProvider{defaults},
	}
	clitest.Inspect(t.Context(), app, override).Require(t,
		clitest.InspectionReady(),
		ship.Text("voyager"),
		ship.ContributedBy(cli.ValueProducerKindFlag),
	)
}
