package main

import (
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		scanCmd := cli.Command[struct{}]{
			Name: "scan",
			Description: cli.Description{
				Short: "scan nearby worlds",
			},
		}
		statusCmd := cli.Command[struct{}]{
			Name: "status",
			Description: cli.Description{
				Short: "show fleet status",
			},
		}
		cb.Subcommands.Command(scanCmd)
		cb.Subcommands.Command(statusCmd)
	},
}

func main() {
	app, err := cli.Compile(definition)
	if err != nil {
		_ = cli.WriteError(os.Stderr, err)
		os.Exit(1)
	}
	reference, err := app.Reference(cli.ReferenceOptions{})
	if err != nil {
		_ = cli.WriteError(os.Stderr, err)
		os.Exit(1)
	}
	for _, commandID := range reference.CommandOrder {
		if commandID == reference.Root {
			continue
		}
		command := reference.Commands[commandID]
		fmt.Printf("%-11s%s\n", command.Name, command.Description.Short)
	}
}
