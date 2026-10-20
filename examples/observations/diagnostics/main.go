package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type contactConfig struct {
	CallSign string
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		contactCmd := cli.Command[contactConfig]{
			Name: "contact",
			Declare: func(cb *cli.CommandBuilder[contactConfig]) {
				callSign := cb.Flags.String(&cb.Config.CallSign)
				callSign.Long("call-sign")
				cb.Rule.Require(callSign)
			},
			Run: func(_ context.Context, cfg *contactConfig) error {
				fmt.Printf("contacting %s\n", cfg.CallSign)
				return nil
			},
		}
		cb.Subcommands.Command(contactCmd)
	},
}

func main() {
	app, err := cli.Compile(definition)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	request := cli.RunRequest{Args: os.Args[1:]}
	err = app.Run(context.Background(), request)
	if err == nil {
		return
	}
	report := cli.Diagnostics(err)
	if len(report.Diagnostics) == 0 {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.ExitCode(err))
	}
	for _, diagnostic := range report.Diagnostics {
		fmt.Printf("diagnostic: %s\n", diagnostic.Code)
	}
}
