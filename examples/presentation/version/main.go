package main

import (
	"context"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

var definition = cli.Definition[struct{}]{
	Name:    "alienctl",
	Release: cli.Release.SemVer("1.2.3"),
	Version: cli.VersionConfig{},
}

func main() {
	app, err := cli.Compile(definition)
	if err != nil {
		_ = cli.WriteError(os.Stderr, err)
		os.Exit(1)
	}
	if err := app.RunFromProcess(context.Background()); err != nil {
		_ = cli.WriteError(os.Stderr, err)
		os.Exit(cli.ExitCode(err))
	}
}
