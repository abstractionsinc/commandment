package main

import (
	"bytes"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestVersionUsesDefinitionRelease(t *testing.T) {
	t.Parallel()

	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	var output bytes.Buffer
	request := cli.RunRequest{
		Args:   []string{"--version"},
		Stdout: &output,
	}
	if err := app.Run(t.Context(), request); err != nil {
		t.Fatalf("Run(version) error = %v", err)
	}
	if got, want := output.String(), "1.2.3\n"; got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}
