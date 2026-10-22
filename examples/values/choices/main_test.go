package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestApproachValueUsesTheDeclaredChoices(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	approach := clitest.Command("visit").Flag("--approach").Value()
	domain := approach.Domain(cli.ValueDomainUseDestination)
	clitest.Schema(app).Require(t,
		domain.EntriesExactly(domain.Entry("orbit"), domain.Entry("land"), domain.Entry("flyby")),
	)
	landRequest := cli.InspectionRequest{
		Args: []string{"visit", "--approach", "land", "europa"},
	}
	clitest.Inspect(t.Context(), app, landRequest).Require(t,
		clitest.InspectionReady(),
		approach.Text("land"),
	)
}
