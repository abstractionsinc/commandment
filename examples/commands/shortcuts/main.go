package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type navigateConfig struct {
	Destination string
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		cmd := cli.Command[navigateConfig]{
			Name: "navigate",
			Declare: func(cb *cli.CommandBuilder[navigateConfig]) {
				cb.Arguments.String(&cb.Config.Destination, "destination")
				shortcut := cli.CommandShortcut{
					Name: "home",
					Args: []string{"earth"},
				}
				cb.Shortcut(shortcut)
			},
			Run: func(_ context.Context, cfg *navigateConfig) error {
				fmt.Printf("navigating to %s\n", cfg.Destination)
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
