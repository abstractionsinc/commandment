package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type watchConfig struct {
	Events <-chan cli.SignalEvent
	Output io.Writer
}

func openAlarm(_ context.Context) (cli.SignalStream[string], error) {
	notifications := make(chan cli.SignalNotification[string], 1)
	notification := cli.SignalNotification[string]{
		Signal: "alarm",
		Origin: cli.SignalOriginRuntime,
	}
	notifications <- notification

	receive := func(ctx context.Context) (cli.SignalNotification[string], error) {
		select {
		case notification := <-notifications:
			return notification, nil
		case <-ctx.Done():
			return cli.SignalNotification[string]{}, context.Cause(ctx)
		}
	}
	stream := cli.SignalStream[string]{
		Receive: receive,
		Close:   func() error { return nil },
	}
	return stream, nil
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		descriptor := cli.SignalDescriptor[string]{
			Signal:   "alarm",
			Fidelity: cli.SignalSourceDeliveryFidelityExact,
		}
		source := cli.SignalSource[string]{
			Signals: []cli.SignalDescriptor[string]{descriptor},
			Open:    openAlarm,
		}
		registered := cb.Signals.Source(source)
		alarm := registered.Signal("alarm")

		watchCmd := cli.Command[watchConfig]{
			Name: "watch",
			Declare: func(cb *cli.CommandBuilder[watchConfig]) {
				events := cb.Signals.Endpoint(&cb.Config.Events)
				match := cb.Signals.Match.On(alarm).During(cli.SignalPhaseExecution).Each().Hint("alarm")
				cb.Signals.Policy(match, cb.Signals.Action.Emit(events))
				cb.Access.Stdout(&cb.Config.Output)
			},
			Run: func(ctx context.Context, cfg *watchConfig) error {
				select {
				case <-cfg.Events:
					if _, err := fmt.Fprintln(cfg.Output, "received alarm"); err != nil {
						return err //nolint:wrapcheck // Preserve the writer error at the application boundary.
					}
					return nil
				case <-ctx.Done():
					return context.Cause(ctx)
				}
			},
		}
		cb.Subcommands.Command(watchCmd)
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
