package clitest_test

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCompileReturnsRealApplicationAndRunsDeclarationOnce(t *testing.T) {
	t.Parallel()
	calls := 0
	definition := cli.Definition[struct{}]{
		Name: "compile-success",
		Declare: func(_ *cli.CommandBuilder[struct{}]) {
			calls++
		},
	}
	app := clitest.Compile(t, definition)
	if calls != 1 {
		t.Fatalf("declaration calls = %d, want 1", calls)
	}
	clitest.Schema(app).Require(t, clitest.RootCommand().Exists())
}

func TestCompileFailsReturnsDiagnosticsResultAndRunsDeclarationOnce(t *testing.T) {
	t.Parallel()
	calls := 0
	definition := cli.Definition[struct{}]{
		Name: "compile-rejection",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			calls++
			root.Arguments.Form(nil)
		},
	}
	clitest.CompileFails(t, definition).Require(t,
		clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present(),
	)
	if calls != 1 {
		t.Fatalf("declaration calls = %d, want 1", calls)
	}
}
