package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type visitConfig struct {
	Moon     string
	Approach string
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		cmd := cli.Command[visitConfig]{
			Name: "visit",
			Declare: func(cb *cli.CommandBuilder[visitConfig]) {
				approaches := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
					values.Value("orbit", "orbit")
					values.Value("land", "land")
					values.Value("flyby", "flyby")
				})
				approach := cb.Values.Value(&cb.Config.Approach, approaches)
				cb.Values.Default(approach, "orbit")
				cb.Flags.ValueFor(approach).Long("approach")
				cb.Arguments.String(&cb.Config.Moon, "moon")
			},
			Run: func(_ context.Context, cfg *visitConfig) error {
				fmt.Printf("visiting %s by %s\n", cfg.Moon, cfg.Approach)
				return nil
			},
		}
		cb.Subcommands.Command(cmd)
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
