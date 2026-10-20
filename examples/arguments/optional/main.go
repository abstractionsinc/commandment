package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type statusConfig struct {
	Ship string
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		cmd := cli.Command[statusConfig]{
			Name: "status",
			Declare: func(cb *cli.CommandBuilder[statusConfig]) {
				cb.Arguments.String(&cb.Config.Ship, "ship").Optional()
			},
			Run: func(_ context.Context, cfg *statusConfig) error {
				if cfg.Ship == "" {
					fmt.Println("mothership status: nominal")
					return nil
				}
				fmt.Printf("%s status: nominal\n", cfg.Ship)
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
