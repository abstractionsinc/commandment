package main

import (
	"bytes"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCustomSignalSourceEmitsThroughTypedEndpoint(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	policy := clitest.Command("watch").SignalPolicyAt(0)
	clitest.Schema(app).Require(t,
		policy.Exists(),
		policy.Hint("alarm"),
		policy.ActionCount(1),
	)

	var output bytes.Buffer
	request := cli.RunRequest{
		Args:   []string{"watch"},
		Stdout: &output,
	}
	if err := app.Run(t.Context(), request); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got, want := output.String(), "received alarm\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
