package main

import (
	"bytes"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestCustomHelpRendering(t *testing.T) {
	t.Parallel()

	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	var output bytes.Buffer
	request := cli.RunRequest{Args: []string{"help", "scan"}, Stdout: &output}
	if err := app.Run(t.Context(), request); err != nil {
		t.Fatalf("Run(help) error = %v", err)
	}
	want := "ALIENCTL scan\n" + "scan a nearby world\n"
	if got := output.String(); got != want {
		t.Fatalf("help output = %q, want %q", got, want)
	}
}
