package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestHostFactSuppliesTheCurrentShip(t *testing.T) {
	t.Parallel()

	definition, facts := newProgram()
	app := clitest.Compile(t, definition)
	ship := clitest.Command("contact").FactAt(0).Value()
	request := cli.InspectionRequest{
		Args:  []string{"contact", "martian"},
		Facts: []cli.FactProvider{facts},
	}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionReady(),
		ship.Resolved(),
		ship.ContributedBy(cli.ValueProducerKindSuppliedFact),
	)
}
