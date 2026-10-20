package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestOptionalShipMayBePresentOrAbsent(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	ship := clitest.Command("status").ArgumentFormAt(0).Term("ship")

	defaultRequest := cli.InspectionRequest{Args: []string{"status"}}
	clitest.Inspect(t.Context(), app, defaultRequest).Require(t,
		clitest.InspectionReady(),
		ship.Absent(),
	)
	namedRequest := cli.InspectionRequest{Args: []string{"status", "scout-7"}}
	clitest.Inspect(t.Context(), app, namedRequest).Require(t,
		clitest.InspectionReady(),
		ship.Present(),
		ship.Value().Text("scout-7"),
	)
}
