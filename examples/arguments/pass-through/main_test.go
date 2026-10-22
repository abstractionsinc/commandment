package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestRelayPreservesPassThroughTokens(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	request := cli.InspectionRequest{
		Args: []string{"relay", "rover-7", "--", "echo", "hello", "earth"},
	}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionReady(),
		clitest.TokenAt(3).Role(cli.TokenRolePassThrough),
		clitest.TokenAt(4).Role(cli.TokenRolePassThrough),
		clitest.TokenAt(5).Role(cli.TokenRolePassThrough),
	)
}
