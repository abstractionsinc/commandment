package main

import (
	"errors"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestApplicationErrorKeepsIdentityAndExitCode(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	request := cli.RunRequest{
		Args: []string{"land", "earth"},
	}
	err := app.Run(t.Context(), request)
	if !errors.Is(err, errLandingDenied) {
		t.Fatalf("Run() error = %v, want identity %v", err, errLandingDenied)
	}
	if got := cli.ExitCode(err); got != landingDeniedExitCode {
		t.Fatalf("ExitCode() = %d, want %d", got, landingDeniedExitCode)
	}
	clitest.Diagnostics(err).Require(t, clitest.DiagnosticsExactly())
}
