package main

import (
	"io"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestMissingCallSignProducesStructuredDiagnostic(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	request := cli.RunRequest{
		Args:   []string{"contact"},
		Stdout: io.Discard,
	}
	err := app.Run(t.Context(), request)
	if err == nil {
		t.Fatal("Run() error = nil, want rule violation")
	}
	clitest.Diagnostics(err).Require(t,
		clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticRuleViolation)),
	)
}
