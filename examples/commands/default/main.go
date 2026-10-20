package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type launchConfig struct {
	Ship string
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		fleetCmd := cli.Command[struct{}]{
			Name: "fleet",
			Declare: func(cb *cli.CommandBuilder[struct{}]) {
				statusCmd := cli.Command[struct{}]{
					Name: "status",
					Run: func(context.Context, *struct{}) error {
						fmt.Println("fleet status: 3 ships ready")
						return nil
					},
				}
				cb.Subcommands.Command(statusCmd).Default()

				launchCmd := cli.Command[launchConfig]{
					Name: "launch",
					Declare: func(cb *cli.CommandBuilder[launchConfig]) {
						cb.Arguments.String(&cb.Config.Ship, "ship")
					},
					Run: func(_ context.Context, cfg *launchConfig) error {
						fmt.Printf("launching %s\n", cfg.Ship)
						return nil
					},
				}
				cb.Subcommands.Command(launchCmd)
			},
		}
		cb.Subcommands.Command(fleetCmd)
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
