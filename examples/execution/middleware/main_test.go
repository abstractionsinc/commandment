package main

import (
	"bytes"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestMiddlewareWrapsTheHandler(t *testing.T) {
	t.Parallel()

	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	var output bytes.Buffer
	request := cli.RunRequest{Args: []string{"scan", "mars"}, Stdout: &output}
	if err := app.Run(t.Context(), request); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	const want = "preparing sensors\nscanning mars\nrecording scan\n"
	if got := output.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
