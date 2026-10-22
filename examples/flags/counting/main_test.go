package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestRepeatedVerboseFlagsResolveToTheCount(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	verbosity := clitest.Command("scan").Flag("-v").Value()
	request := cli.InspectionRequest{
		Args: []string{"scan", "-v", "-v", "-v", "mars"},
	}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionReady(),
		verbosity.Text("3"),
	)
}
