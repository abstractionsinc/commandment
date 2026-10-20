package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

const (
	completionArgumentCount = 3
	usageExitCode           = 2
)

type uploadConfig struct {
	Manifest string
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		cmd := cli.Command[uploadConfig]{
			Name: "upload",
			Declare: func(cb *cli.CommandBuilder[uploadConfig]) {
				manifest := cb.Flags.String(&cb.Config.Manifest)
				manifest.CompleteWith(cli.CompletionNativeFiles).Long("manifest")
			},
		}
		cb.Subcommands.Command(cmd)
	},
}

func main() {
	args := os.Args[1:]
	if len(args) != completionArgumentCount || args[1] != "--manifest" {
		fmt.Fprintln(os.Stderr, "usage: alienctl-complete upload --manifest <partial-path>")
		os.Exit(usageExitCode)
	}

	app, err := cli.Compile(definition)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	partial := args[len(args)-1]
	client := cli.CompletionClient{
		Capabilities: cli.CompletionClientCapabilities{
			Native: cli.CompletionNativeCapabilities{Files: true},
		},
	}
	request := cli.CompletionRequest{
		Args:   args,
		Cursor: cli.CompletionCursor{Argument: len(args) - 1, Byte: len(partial)},
		Client: client,
	}
	completion, err := app.Complete(context.Background(), request)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, request := range completion.NativeRequests {
		if request.Kind == cli.CompletionNativeFiles {
			fmt.Println("native completion: files")
		}
	}
}
