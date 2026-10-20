package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type visitConfig struct {
	System string
	Moon   string
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		cmd := cli.Command[visitConfig]{
			Name: "visit",
			Declare: func(cb *cli.CommandBuilder[visitConfig]) {
				systems := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
					values.Value("jovian", "jovian")
					values.Value("saturnian", "saturnian")
				})
				system := cb.Values.Value(&cb.Config.System, systems)
				cb.Flags.ValueFor(system).Long("system")
				cb.Rule.Require(system)

				moons := cb.Values.Domain(func(_ context.Context, _ *visitConfig, dependencies cli.Dependencies, domain *cli.ValueDomainBuilder[string]) error {
					resolvedSystem, _ := dependencies.Get(system)
					switch resolvedSystem {
					case "jovian":
						domain.Value("europa", "europa")
						domain.Value("ganymede", "ganymede")
					case "saturnian":
						domain.Value("titan", "titan")
						domain.Value("enceladus", "enceladus")
					}
					return nil
				}, system)
				moon := cb.Values.Value(&cb.Config.Moon, moons)
				cb.Arguments.ValueFor(moon, "moon")
			},
			Run: func(_ context.Context, cfg *visitConfig) error {
				fmt.Printf("visiting %s in the %s system\n", cfg.Moon, cfg.System)
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
