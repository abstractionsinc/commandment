package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestMoonChoicesDependOnTheSelectedSystem(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	command := clitest.Command("visit")
	system := command.Flag("--system").Value()
	moon := command.ArgumentFormAt(0).Term("moon").Value()
	domain := moon.Domain(cli.ValueDomainUseDestination)
	request := cli.InspectionRequest{
		Args: []string{"visit", "--system", "jovian", "europa"},
	}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionReady(),
		system.Text("jovian"),
		moon.Text("europa"),
		domain.EntriesExactly(domain.Entry("europa"), domain.Entry("ganymede")),
	)
}
