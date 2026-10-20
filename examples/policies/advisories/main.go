package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type approachConfig struct {
	Target   string
	Reckless bool
	Quiet    bool
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		cmd := cli.Command[approachConfig]{
			Name: "approach",
			Declare: func(cb *cli.CommandBuilder[approachConfig]) {
				cb.Arguments.String(&cb.Config.Target, "target")
				reckless := cb.Flags.Bool(&cb.Config.Reckless).Occurrences(cli.Count.Optional())
				reckless.Long("reckless")
				quiet := cb.Flags.Bool(&cb.Config.Quiet).Occurrences(cli.Count.Optional())
				quiet.Long("quiet")

				advisory := cli.Advisory{
					Code:     "reckless-approach",
					Severity: cli.NoticeSeverityWarning,
					Message:  "reckless approach enabled",
				}
				cb.Advisory.When(cb.When.Equal(reckless, true), advisory).Subject(reckless)
				cb.Notice.SuppressWhen(cb.When.Equal(quiet, true), cli.NoticeMatch{AdvisoryCode: advisory.Code})
			},
			Run: func(_ context.Context, cfg *approachConfig) error {
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
	notify := func(_ context.Context, notice cli.Notice) error {
		if _, err := fmt.Fprintf(os.Stderr, "warning: %s\n", notice.Message); err != nil {
			return err //nolint:wrapcheck // Preserve the stderr writer error at the process boundary.
		}
		return nil
	}
	request := cli.RunRequest{Args: os.Args[1:], Notify: notify}
	if err := app.Run(context.Background(), request); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.ExitCode(err))
	}
}
