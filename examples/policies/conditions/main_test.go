package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCloakLevelIsAvailableOnlyInStealthMode(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	cloak := clitest.Command("approach").Flag("--cloak-level")
	ordinaryRequest := cli.InspectionRequest{
		Args: []string{"approach", "earth"},
	}
	clitest.Inspect(t.Context(), app, ordinaryRequest).Require(t,
		clitest.InspectionReady(),
		cloak.Unavailable(),
	)
	stealthRequest := cli.InspectionRequest{
		Args: []string{"approach", "--stealth", "--cloak-level", "3", "earth"},
	}
	clitest.Inspect(t.Context(), app, stealthRequest).Require(t,
		clitest.InspectionReady(),
		cloak.Available(),
		cloak.Value().Text("3"),
	)
}
