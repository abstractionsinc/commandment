package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCustomResponseFileLoaderSuppliesArgument(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	request := cli.InspectionRequest{
		Args: []string{"hail", "@crew"},
		ResponseFiles: cli.ResponseFileAccess{
			Load: loadCrew,
		},
	}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionReady(),
		clitest.Command("hail").ArgumentFormAt(0).TermAt(0).Value().Text("Zorblax"),
	)
}
