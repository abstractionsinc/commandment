package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

const landingDeniedExitCode = 7

var errLandingDenied = errors.New("landing denied: earth is protected")

type landConfig struct {
	World string
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		cmd := cli.Command[landConfig]{
			Name: "land",
			Declare: func(cb *cli.CommandBuilder[landConfig]) {
				cb.Arguments.String(&cb.Config.World, "world")
			},
			Run: func(_ context.Context, cfg *landConfig) error {
				if cfg.World == "earth" {
					return cli.WithExitCode(errLandingDenied, landingDeniedExitCode)
				}
				fmt.Printf("landing on %s\n", cfg.World)
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
