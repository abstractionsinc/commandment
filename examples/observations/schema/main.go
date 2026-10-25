package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
)

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		scanCmd := cli.Command[struct{}]{
			Name: "scan",
		}
		statusCmd := cli.Command[struct{}]{
			Name: "status",
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

	schema := app.Schema()
	names := make([]string, 0, len(schema.CommandOrder)-1)
	for _, commandID := range schema.CommandOrder {
		if commandID == schema.Root {
			continue
		}
		names = append(names, schema.Commands[commandID].Name)
	}
	fmt.Printf("commands: %s\n", strings.Join(names, ", "))
}
