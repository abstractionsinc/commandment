package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type approachConfig struct {
	Target     string
	Stealth    bool
	CloakLevel int
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		cmd := cli.Command[approachConfig]{
			Name: "approach",
			Declare: func(cb *cli.CommandBuilder[approachConfig]) {
				cb.Arguments.String(&cb.Config.Target, "target")
				stealth := cb.Flags.Bool(&cb.Config.Stealth).Occurrences(cli.Count.Optional())
				stealth.Long("stealth")
				cloakLevel := cb.Flags.Int(&cb.Config.CloakLevel)
				cloakLevel.AvailableWhen(cb.When.Equal(stealth, true)).Long("cloak-level")
			},
			Run: func(_ context.Context, cfg *approachConfig) error {
				if cfg.Stealth {
					fmt.Printf("approaching %s under cloak level %d\n", cfg.Target, cfg.CloakLevel)
					return nil
				}
				fmt.Printf("approaching %s\n", cfg.Target)
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
