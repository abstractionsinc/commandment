package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type outpostConfig struct {
	System string
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		outpostCmd := cli.Command[outpostConfig]{
			Name: "outpost",
			Declare: func(cb *cli.CommandBuilder[outpostConfig]) {
				system := cb.Flags.String(&cb.Config.System)
				system.Long("system")
				cb.Rule.Require(system)

				cb.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[outpostConfig]) error {
					switch batch.Config.System {
					case "jovian":
						europaCmd := cli.Command[struct{}]{
							Name: "europa",
							Run: func(context.Context, *struct{}) error {
								fmt.Println("contacting europa outpost")
								return nil
							},
						}
						batch.Command(europaCmd)
					case "saturnian":
						titanCmd := cli.Command[struct{}]{
							Name: "titan",
							Run: func(context.Context, *struct{}) error {
								fmt.Println("contacting titan outpost")
								return nil
							},
						}
						batch.Command(titanCmd)
					}
					return nil
				}, system)
			},
		}
		cb.Subcommands.Command(outpostCmd)
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
