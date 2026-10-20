package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type scanConfig struct {
	World string
	Range int
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		cmd := cli.Command[scanConfig]{
			Name: "scan",
			Declare: func(cb *cli.CommandBuilder[scanConfig]) {
				cb.Arguments.String(&cb.Config.World, "world")
				scanRange := cb.Flags.Int(&cb.Config.Range)
				scanRange.Long("range")
				scanRange.Short('r')
				cb.Rule.Require(scanRange)
			},
			Run: func(_ context.Context, cfg *scanConfig) error {
				fmt.Printf("scanning %s at range %d\n", cfg.World, cfg.Range)
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
