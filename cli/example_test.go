package cli_test

import (
	"context"
	"fmt"

	"github.com/abstractionsinc/commandment/cli"
)

type exampleVerboseConfig struct{ Verbose bool }

func ExampleCompile() {
	run := false
	app, err := cli.Compile(cli.Definition[struct{}]{
		Name: "tool",
		Run: func(context.Context, *struct{}) error {
			run = true
			return nil
		},
	})
	if err != nil {
		panic(err)
	}
	if err := app.Run(context.Background(), cli.RunRequest{}); err != nil {
		panic(err)
	}

	fmt.Println(run)
	// Output: true
}

func ExampleApplication_Schema() {
	app, err := cli.Compile(cli.Definition[struct{}]{
		Name: "tool",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(len(app.Schema().Commands))
	// Output: 2
}

func ExampleCommandBuilder() {
	type config = exampleVerboseConfig
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.Bool(&root.Config.Verbose).Long("verbose")
		},
		Run: func(_ context.Context, cfg *config) error {
			fmt.Println(cfg.Verbose)
			return nil
		},
	})
	if err != nil {
		panic(err)
	}
	if err := app.Run(context.Background(), cli.RunRequest{
		Args: []string{"--verbose"},
	}); err != nil {
		panic(err)
	}

	// Output: true
}
