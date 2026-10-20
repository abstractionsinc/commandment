package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
)

type relayConfig struct {
	Rover   string
	Command []string
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		cmd := cli.Command[relayConfig]{
			Name: "relay",
			Declare: func(cb *cli.CommandBuilder[relayConfig]) {
				command := cb.Values.Sequence(&cb.Config.Command, cli.Value.String())
				cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[relayConfig]) {
					form.String(&form.Config.Rover, "rover")
					form.PassThrough(command, "command", cli.Count.OneOrMore())
				})
			},
			Run: func(_ context.Context, cfg *relayConfig) error {
				fmt.Printf("relaying to %s: %s\n", cfg.Rover, strings.Join(cfg.Command, " "))
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
