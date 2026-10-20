package benchmark

import (
	"fmt"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clibench"
)

func BenchmarkCompile(b *testing.B) {
	for _, breadth := range [...]int{1, 4, 16, 64} {
		names := make([]string, breadth)
		for index := range names {
			names[index] = fmt.Sprintf("command-%03d", index)
		}
		definition := cli.Definition[stringConfig]{
			Name:            "tool",
			Help:            cli.HelpConfig{Disabled: true},
			ShellCompletion: cli.ShellCompletion{Disabled: true},
			Declare: func(root *cli.CommandBuilder[stringConfig]) {
				for _, name := range names {
					root.Subcommands.Command(cli.Command[stringConfig]{
						Name: name,
						Declare: func(command *cli.CommandBuilder[stringConfig]) {
							command.Values.String(&command.Config.Value)
						},
					})
				}
			},
		}
		app, err := cli.Compile(definition)
		if err != nil {
			b.Fatalf("Compile breadth preflight: %v", err)
		}
		if schema := app.Schema(); len(schema.Commands) != breadth+1 || len(schema.Values) != breadth {
			b.Fatalf("Compile breadth preflight: commands=%d values=%d", len(schema.Commands), len(schema.Values))
		}
		b.Run(fmt.Sprintf("Breadth/N=%d", breadth), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Compile(b, definition)
		})
	}

	for _, depth := range [...]int{1, 4, 16, 64} {
		names := make([]string, depth)
		for index := range names {
			names[index] = fmt.Sprintf("level-%03d", index)
		}
		definition := cli.Definition[stringConfig]{
			Name:            "tool",
			Help:            cli.HelpConfig{Disabled: true},
			ShellCompletion: cli.ShellCompletion{Disabled: true},
			Declare: func(root *cli.CommandBuilder[stringConfig]) {
				declareCompileDepth(root, names, 0)
			},
		}
		app, err := cli.Compile(definition)
		if err != nil {
			b.Fatalf("Compile depth preflight: %v", err)
		}
		if schema := app.Schema(); len(schema.Commands) != depth+1 || len(schema.Values) != depth {
			b.Fatalf("Compile depth preflight: commands=%d values=%d", len(schema.Commands), len(schema.Values))
		}
		b.Run(fmt.Sprintf("Depth/N=%d", depth), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Compile(b, definition)
		})
	}

	for _, commands := range [...]int{1, 4, 16, 64} {
		names := make([]string, commands)
		for index := range names {
			names[index] = fmt.Sprintf("mixed-%03d", index)
		}
		definition := cli.Definition[stringConfig]{
			Name:            "tool",
			Help:            cli.HelpConfig{Disabled: true},
			ShellCompletion: cli.ShellCompletion{Disabled: true},
			Declare: func(root *cli.CommandBuilder[stringConfig]) {
				for _, name := range names {
					root.Subcommands.Command(cli.Command[stringConfig]{
						Name: name,
						Declare: func(command *cli.CommandBuilder[stringConfig]) {
							command.Flags.String(&command.Config.Values[1]).Long("enabled")
							command.Arguments.Form(func(form *cli.ArgumentFormBuilder[stringConfig]) {
								form.Literal("left")
								form.Literal("right")
							})
						},
					})
				}
			},
		}
		app, err := cli.Compile(definition)
		if err != nil {
			b.Fatalf("Compile mixed preflight: %v", err)
		}
		schema := app.Schema()
		if len(schema.Commands) != commands+1 || len(schema.Flags) != commands ||
			len(schema.ArgumentForms) != commands ||
			len(schema.ArgumentTerms) != commands*2 {
			b.Fatalf(
				"Compile mixed preflight: commands=%d flags=%d forms=%d terms=%d",
				len(schema.Commands),
				len(schema.Flags),
				len(schema.ArgumentForms),
				len(schema.ArgumentTerms),
			)
		}
		b.Run(fmt.Sprintf("Mixed/N=%d", commands), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Compile(b, definition)
		})
	}

	for _, values := range [...]int{16, 64, 256} {
		definition := cli.Definition[stringConfig]{
			Name:            "tool",
			Help:            cli.HelpConfig{Disabled: true},
			ShellCompletion: cli.ShellCompletion{Disabled: true},
			Declare: func(root *cli.CommandBuilder[stringConfig]) {
				for index := range values {
					root.Values.String(&root.Config.Values[index])
				}
			},
		}
		app, err := cli.Compile(definition)
		if err != nil {
			b.Fatalf("Compile values preflight: %v", err)
		}
		if schema := app.Schema(); len(schema.Commands) != 1 || len(schema.Values) != values {
			b.Fatalf("Compile values preflight: commands=%d values=%d", len(schema.Commands), len(schema.Values))
		}
		b.Run(fmt.Sprintf("Values/N=%d", values), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Compile(b, definition)
		})
	}
}

func declareCompileDepth(parent *cli.CommandBuilder[stringConfig], names []string, level int) {
	if level == len(names) {
		return
	}
	parent.Subcommands.Command(cli.Command[stringConfig]{
		Name: names[level],
		Declare: func(child *cli.CommandBuilder[stringConfig]) {
			child.Values.String(&child.Config.Value)
			declareCompileDepth(child, names, level+1)
		},
	})
}
