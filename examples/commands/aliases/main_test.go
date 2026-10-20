package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestSurveyAliasTargetsScan(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	scan := root.Command("scan")
	clitest.Schema(app).Require(t,
		root.Alias("survey").Exists(),
		root.Alias("survey").Targets(scan),
	)
	aliasRequest := cli.InspectionRequest{Args: []string{"survey", "mars"}}
	clitest.Inspect(t.Context(), app, aliasRequest).Require(t,
		clitest.InspectionReady(),
		clitest.SelectionPathExactly(root, scan),
	)
}
