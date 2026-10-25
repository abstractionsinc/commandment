package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type scanConfig struct {
	Range int
	World string
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		scanCmd := cli.Command[scanConfig]{
			Name: "scan",
			Declare: func(cb *cli.CommandBuilder[scanConfig]) {
				cb.Arguments.String(&cb.Config.World, "world")
				cb.Flags.Int(&cb.Config.Range).Long("range")
			},
			Run: func(_ context.Context, cfg *scanConfig) error {
				fmt.Printf("scanning %s at range %d\n", cfg.World, cfg.Range)
				return nil
			},
		}
		cb.Subcommands.Command(scanCmd)
	},
}

func scanValueIDs(schema cli.Schema) (cli.CommandID, cli.ValueID, cli.ValueID, bool) {
	for _, commandID := range schema.CommandOrder {
		command := schema.Commands[commandID]
		if command.Name != "scan" {
			continue
		}

		var rangeID cli.ValueID
		for _, flagID := range command.Flags {
			flag := schema.Flags[flagID]
			for _, spellingID := range flag.Spellings {
				if schema.FlagSpellings[spellingID].Spelling.Text == "--range" {
					rangeID = flag.Destination
				}
			}
		}

		var worldID cli.ValueID
		for _, formID := range command.ArgumentForms {
			for _, termID := range schema.ArgumentForms[formID].Terms {
				term := schema.ArgumentTerms[termID]
				if term.Name == "world" {
					worldID = term.Destination
				}
			}
		}
		return commandID, rangeID, worldID, rangeID != 0 && worldID != 0
	}
	return 0, 0, 0, false
}

func main() {
	app, err := cli.Compile(definition)
	if err != nil {
		_ = cli.WriteError(os.Stderr, err)
		os.Exit(1)
	}

	request := cli.InspectionRequest{
		Args: os.Args[1:],
	}
	inspection, err := app.Inspect(context.Background(), request)
	if err != nil {
		_ = cli.WriteError(os.Stderr, err)
		os.Exit(1)
	}

	commandID, rangeID, worldID, ok := scanValueIDs(app.Schema())
	if !ok || inspection.Selection.Command.Selected != commandID {
		fmt.Fprintln(os.Stderr, "scan invocation not selected")
		os.Exit(1)
	}

	fmt.Printf("command: %s\n", inspection.Commands[commandID].Name)
	fmt.Printf("range: %s\n", inspection.Values[rangeID].Representation.Text)
	fmt.Printf("world: %s\n", inspection.Values[worldID].Representation.Text)
}
