package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type listenConfig struct {
	Events <-chan cli.SignalEvent
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		listenCmd := cli.Command[listenConfig]{
			Name: "listen",
			Declare: func(cb *cli.CommandBuilder[listenConfig]) {
				events := cb.Signals.Endpoint(&cb.Config.Events)
				interrupt := cb.Signals.Match.Interrupt().During(cli.SignalPhaseExecution).Each().Hint("interrupt")
				cb.Signals.Policy(interrupt, cb.Signals.Action.Emit(events))
			},
			Run: func(ctx context.Context, cfg *listenConfig) error {
				fmt.Println("listening for signals")
				select {
				case <-cfg.Events:
					fmt.Println("signal received")
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		}
		cb.Subcommands.Command(listenCmd)
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
