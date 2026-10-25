package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
)

const (
	completionArgumentCount = 2
	usageExitCode           = 2
)

type visitConfig struct {
	World string
}

func completeWorld(_ context.Context, _ *visitConfig, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	worlds := []string{"europa", "ganymede", "io"}
	candidates := make([]cli.CompletionCandidate, 0, len(worlds))
	for _, world := range worlds {
		if strings.HasPrefix(world, request.Text) {
			candidates = append(candidates, cli.CompletionCandidate{
				Value: world,
			})
		}
	}
	contribution := cli.CompletionContribution{
		Candidates: candidates,
	}
	return contribution, nil
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		cmd := cli.Command[visitConfig]{
			Name: "visit",
			Declare: func(cb *cli.CommandBuilder[visitConfig]) {
				world := cb.Arguments.String(&cb.Config.World, "world")
				world.CompleteBy(completeWorld)
			},
		}
		cb.Subcommands.Command(cmd)
	},
}

func main() {
	args := os.Args[1:]
	if len(args) != completionArgumentCount {
		fmt.Fprintln(os.Stderr, "usage: alienctl-complete visit <partial-world>")
		os.Exit(usageExitCode)
	}

	app, err := cli.Compile(definition)
	if err != nil {
		_ = cli.WriteError(os.Stderr, err)
		os.Exit(1)
	}

	partial := args[1]
	request := cli.CompletionRequest{
		Args: args,
		Cursor: cli.CompletionCursor{
			Argument: 1,
			Byte:     len(partial),
		},
	}
	completion, err := app.Complete(context.Background(), request)
	if err != nil {
		_ = cli.WriteError(os.Stderr, err)
		os.Exit(1)
	}
	for _, item := range completion.Items {
		fmt.Println(item.Value)
	}
}
