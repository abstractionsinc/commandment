package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type contactConfig struct {
	Ship   string
	Target string
}

func newProgram() (cli.Definition[struct{}], cli.FactProvider) {
	var ship cli.Destination[contactConfig, string]
	definition := cli.Definition[struct{}]{
		Name: "alienctl",
		Declare: func(cb *cli.CommandBuilder[struct{}]) {
			cmd := cli.Command[contactConfig]{
				Name: "contact",
				Declare: func(cb *cli.CommandBuilder[contactConfig]) {
					ship = cb.Facts.Require(&cb.Config.Ship)
					cb.Arguments.String(&cb.Config.Target, "target")
				},
				Run: func(_ context.Context, cfg *contactConfig) error {
					fmt.Printf("%s contacting %s\n", cfg.Ship, cfg.Target)
					return nil
				},
			}
			cb.Subcommands.Command(cmd)
		},
	}
	facts := cli.FactsFor[contactConfig](func(_ context.Context, supply *cli.FactSupply[contactConfig]) error {
		supply.Set(ship, "mothership")
		return nil
	})
	return definition, facts
}

func main() {
	definition, facts := newProgram()
	app, err := cli.Compile(definition)
	if err != nil {
		_ = cli.WriteError(os.Stderr, err)
		os.Exit(1)
	}
	request := cli.RunRequest{
		Args:  os.Args[1:],
		Facts: []cli.FactProvider{facts},
	}
	if err := app.Run(context.Background(), request); err != nil {
		_ = cli.WriteError(os.Stderr, err)
		os.Exit(cli.ExitCode(err))
	}
}
