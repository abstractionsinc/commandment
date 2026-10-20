package main

import (
	"bytes"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestLocalizedDefaultHelp(t *testing.T) {
	t.Parallel()

	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	var output bytes.Buffer
	request := cli.RunRequest{
		Args:   []string{"--locale", "fr", "help", "scan"},
		Stdout: &output,
	}
	if err := app.Run(t.Context(), request); err != nil {
		t.Fatalf("Run(help) error = %v", err)
	}
	want := "analyser les mondes proches\n\nUsage:\n  alienctl scan\n"
	if got := output.String(); got != want {
		t.Fatalf("localized help = %q, want %q", got, want)
	}
}
