package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/clitest"
)

func TestSchemaContainsStaticCommands(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	clitest.Schema(app).Require(t,
		clitest.Command("scan").Exists(),
		clitest.Command("status").Exists(),
		clitest.Command("completion").Exists(),
	)
}
