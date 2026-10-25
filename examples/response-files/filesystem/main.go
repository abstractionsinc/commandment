package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type hailConfig struct {
	Name string
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		hailCmd := cli.Command[hailConfig]{
			Name: "hail",
			Declare: func(cb *cli.CommandBuilder[hailConfig]) {
				cb.Dialect.ResponseFiles(true)
				cb.Arguments.String(&cb.Config.Name, "name")
			},
			Run: func(_ context.Context, cfg *hailConfig) error {
				fmt.Printf("hail %s\n", cfg.Name)
				return nil
			},
		}
		cb.Subcommands.Command(hailCmd)
	},
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
