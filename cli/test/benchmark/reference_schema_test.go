package benchmark

import (
	"fmt"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clibench"
)

func BenchmarkReference(b *testing.B) {
	options := cli.ReferenceOptions{
		Visibility: cli.ReferenceVisibilityAll,
		Order:      cli.ReferenceOrderDeclaration,
	}
	for _, declarations := range [...]int{1, 16, 64, 256} {
		definition := cli.Definition[stringConfig]{
			Name:            "tool",
			Help:            cli.HelpConfig{Disabled: true},
			ShellCompletion: cli.ShellCompletion{Disabled: true},
			Declare: func(root *cli.CommandBuilder[stringConfig]) {
				for index := range declarations {
					root.Flags.String(&root.Config.Values[index]).Long(fmt.Sprintf("flag-%03d", index))
				}
			},
		}
		app, err := cli.Compile(definition)
		if err != nil {
			b.Fatalf("Reference compile preflight: %v", err)
		}
		reference, err := app.Reference(options)
		if err != nil {
			b.Fatalf("Reference preflight: %v", err)
		}
		if len(reference.Commands) != 1 || len(reference.Flags) != declarations {
			b.Fatalf("Reference preflight: commands=%d flags=%d", len(reference.Commands), len(reference.Flags))
		}
		b.Run(fmt.Sprintf("Flags/N=%d", declarations), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Reference(b, app, options)
		})
	}
	for _, commands := range [...]int{1, 16, 64, 256} {
		app := projectionBreadthFixture(b, commands)
		reference, err := app.Reference(options)
		if err != nil {
			b.Fatalf("Reference breadth preflight: %v", err)
		}
		if len(reference.Commands) != commands+1 || len(reference.Flags) != commands {
			b.Fatalf("Reference breadth preflight: commands=%d flags=%d", len(reference.Commands), len(reference.Flags))
		}
		b.Run(fmt.Sprintf("Breadth/Children/N=%d/FlagsPerCommand=1", commands), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Reference(b, app, options)
		})
	}
	for _, depth := range [...]int{1, 4, 16, 64} {
		app := projectionDepthFixture(b, depth)
		reference, err := app.Reference(options)
		if err != nil {
			b.Fatalf("Reference depth preflight: %v", err)
		}
		if len(reference.Commands) != depth+1 {
			b.Fatalf("Reference depth preflight: commands=%d want=%d", len(reference.Commands), depth+1)
		}
		b.Run(fmt.Sprintf("Depth/N=%d", depth), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Reference(b, app, options)
		})
	}
}

func BenchmarkSchema(b *testing.B) {
	for _, declarations := range [...]int{1, 16, 64, 256} {
		definition := cli.Definition[stringConfig]{
			Name:            "tool",
			Help:            cli.HelpConfig{Disabled: true},
			ShellCompletion: cli.ShellCompletion{Disabled: true},
			Declare: func(root *cli.CommandBuilder[stringConfig]) {
				for index := range declarations {
					root.Flags.String(&root.Config.Values[index]).Long(fmt.Sprintf("flag-%03d", index))
				}
			},
		}
		app, err := cli.Compile(definition)
		if err != nil {
			b.Fatalf("Schema compile preflight: %v", err)
		}
		schema := app.Schema()
		if len(schema.Commands) != 1 || len(schema.Flags) != declarations {
			b.Fatalf("Schema preflight: commands=%d flags=%d", len(schema.Commands), len(schema.Flags))
		}
		b.Run(fmt.Sprintf("Flags/N=%d", declarations), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Schema(b, app)
		})
	}
	for _, commands := range [...]int{1, 16, 64, 256} {
		app := projectionBreadthFixture(b, commands)
		schema := app.Schema()
		if len(schema.Commands) != commands+1 || len(schema.Flags) != commands {
			b.Fatalf("Schema breadth preflight: commands=%d flags=%d", len(schema.Commands), len(schema.Flags))
		}
		b.Run(fmt.Sprintf("Breadth/Children/N=%d/FlagsPerCommand=1", commands), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Schema(b, app)
		})
	}
	for _, depth := range [...]int{1, 4, 16, 64} {
		app := projectionDepthFixture(b, depth)
		schema := app.Schema()
		if len(schema.Commands) != depth+1 {
			b.Fatalf("Schema depth preflight: commands=%d want=%d", len(schema.Commands), depth+1)
		}
		b.Run(fmt.Sprintf("Depth/N=%d", depth), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Schema(b, app)
		})
	}
}

func projectionBreadthFixture(tb testing.TB, commands int) cli.Application {
	tb.Helper()
	definition := cli.Definition[stringConfig]{
		Name:            "tool",
		Help:            cli.HelpConfig{Disabled: true},
		ShellCompletion: cli.ShellCompletion{Disabled: true},
		Declare: func(root *cli.CommandBuilder[stringConfig]) {
			for index := range commands {
				root.Subcommands.Command(cli.Command[stringConfig]{
					Name: fmt.Sprintf("command-%04d", index),
					Declare: func(child *cli.CommandBuilder[stringConfig]) {
						child.Flags.String(&child.Config.Values[index]).Long(fmt.Sprintf("flag-%04d", index))
					},
				})
			}
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("projection breadth compile: %v", err)
	}
	return app
}

func projectionDepthFixture(tb testing.TB, depth int) cli.Application {
	tb.Helper()
	var declare func(*cli.CommandBuilder[stringConfig], int)
	declare = func(parent *cli.CommandBuilder[stringConfig], level int) {
		if level == depth {
			return
		}
		parent.Subcommands.Command(cli.Command[stringConfig]{
			Name: fmt.Sprintf("level-%03d", level),
			Declare: func(child *cli.CommandBuilder[stringConfig]) {
				declare(child, level+1)
			},
		})
	}
	definition := cli.Definition[stringConfig]{
		Name:            "tool",
		Help:            cli.HelpConfig{Disabled: true},
		ShellCompletion: cli.ShellCompletion{Disabled: true},
		Declare: func(root *cli.CommandBuilder[stringConfig]) {
			declare(root, 0)
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("projection depth compile: %v", err)
	}
	return app
}
