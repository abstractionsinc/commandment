package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type scanConfig struct {
	World  string
	Output io.Writer
}

func scanMiddleware(ctx context.Context, cfg *scanConfig, next cli.MiddlewareNext) error {
	if _, err := fmt.Fprintln(cfg.Output, "preparing sensors"); err != nil {
		return err //nolint:wrapcheck // Preserve the writer error at the application boundary.
	}
	if err := next(ctx); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(cfg.Output, "recording scan"); err != nil {
		return err //nolint:wrapcheck // Preserve the writer error at the application boundary.
	}
	return nil
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		cmd := cli.Command[scanConfig]{
			Name: "scan",
			Declare: func(cb *cli.CommandBuilder[scanConfig]) {
				cb.Arguments.String(&cb.Config.World, "world")
				cb.Access.Stdout(&cb.Config.Output)
			},
			Middleware: []cli.Middleware[scanConfig]{scanMiddleware},
			Run: func(_ context.Context, cfg *scanConfig) error {
				if _, err := fmt.Fprintf(cfg.Output, "scanning %s\n", cfg.World); err != nil {
					return err //nolint:wrapcheck // Preserve the writer error at the application boundary.
				}
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
