package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type navigateConfig struct {
	World string
	X     int
	Y     int
	Z     int
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		cmd := cli.Command[navigateConfig]{
			Name: "navigate",
			Declare: func(cb *cli.CommandBuilder[navigateConfig]) {
				cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[navigateConfig]) {
					form.String(&form.Config.World, "world")
				})
				cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[navigateConfig]) {
					form.Int(&form.Config.X, "x")
					form.Int(&form.Config.Y, "y")
					form.Int(&form.Config.Z, "z")
				})
			},
			Run: func(_ context.Context, cfg *navigateConfig) error {
				if cfg.World != "" {
					fmt.Printf("navigating to %s\n", cfg.World)
					return nil
				}
				fmt.Printf("navigating to coordinates %d,%d,%d\n", cfg.X, cfg.Y, cfg.Z)
				return nil
			},
		}
		cb.Subcommands.Command(cmd)
	},
}

func main() {
	app, err := cli.Compile(definition)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := app.RunFromProcess(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.ExitCode(err))
	}
}
