package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

const crewResponseSource = "crew"

var errUnknownResponseSource = errors.New("unknown response source")

type hailConfig struct {
	Name string
}

func loadCrew(_ context.Context, request cli.ResponseFileRequest) (cli.ResponseFile, error) {
	if request.Path != crewResponseSource {
		return cli.ResponseFile{}, errUnknownResponseSource
	}
	file := cli.ResponseFile{
		Key:      crewResponseSource,
		Location: crewResponseSource,
		Content:  []byte("Zorblax"),
	}
	return file, nil
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		hailCmd := cli.Command[hailConfig]{
			Name: "hail",
			Declare: func(cb *cli.CommandBuilder[hailConfig]) {
				cb.Dialect.ResponseFiles(true)
				cb.Arguments.String(&cb.Config.Name, "name")
			},
			Run: func(_ context.Context, cfg *hailConfig) error {
				fmt.Printf("hail %s\n", cfg.Name)
				return nil
			},
		}
		cb.Subcommands.Command(hailCmd)
	},
}

func main() {
	app, err := cli.Compile(definition)
	if err != nil {
		_ = cli.WriteError(os.Stderr, err)
		os.Exit(1)
	}
	request := cli.RunRequest{
		Args: os.Args[1:],
		ResponseFiles: cli.ResponseFileAccess{
			Load: loadCrew,
		},
	}
	if err := app.Run(context.Background(), request); err != nil {
		_ = cli.WriteError(os.Stderr, err)
		os.Exit(cli.ExitCode(err))
	}
}
