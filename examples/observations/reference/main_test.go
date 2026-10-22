package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestReferenceContainsPresentationDescriptions(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	clitest.Reference(app, cli.ReferenceOptions{}).Require(t,
		clitest.Command("scan").PresentedDescription(cli.Description{
			Short: "scan nearby worlds",
		}),
		clitest.Command("status").PresentedDescription(cli.Description{
			Short: "show fleet status",
		}),
		clitest.Command("completion").Exists(),
	)
}
