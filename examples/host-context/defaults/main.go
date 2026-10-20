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

func newProgram() (cli.Definition[struct{}], cli.DefaultProvider) {
	var ship cli.Destination[statusConfig, string]
	definition := cli.Definition[struct{}]{
		Name: "alienctl",
		Declare: func(cb *cli.CommandBuilder[struct{}]) {
			cmd := cli.Command[statusConfig]{
				Name: "status",
				Declare: func(cb *cli.CommandBuilder[statusConfig]) {
					ship = cb.Values.String(&cb.Config.Ship)
					cb.Flags.ValueFor(ship).Long("ship")
				},
				Run: func(_ context.Context, cfg *statusConfig) error {
					fmt.Printf("%s status: nominal\n", cfg.Ship)
					return nil
				},
			}
			cb.Subcommands.Command(cmd)
		},
	}
	defaults := cli.DefaultsFor[statusConfig](func(_ context.Context, supply *cli.DefaultSupply[statusConfig]) error {
		supply.Set(ship, "scout-7")
		return nil
	})
	return definition, defaults
}

func main() {
	definition, defaults := newProgram()
	app, err := cli.Compile(definition)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	request := cli.RunRequest{Args: os.Args[1:], Defaults: []cli.DefaultProvider{defaults}}
	if err := app.Run(context.Background(), request); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.ExitCode(err))
	}
}
