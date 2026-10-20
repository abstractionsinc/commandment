package clitest_test

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func ExampleSchemaResult_Require() {
	verify := func(t testing.TB) {
		t.Helper()

		definition := cli.Definition[struct{}]{
			Name: "tool",
		}
		app := clitest.Compile(t, definition)
		clitest.Schema(app).Require(t,
			clitest.RootCommand().Name("tool"),
		)
	}

	_ = verify

	// Output:
}
