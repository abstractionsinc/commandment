package benchmark

import (
	"context"
	"fmt"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clibench"
)

// BenchmarkRunLocaleSelection separates unrelated declaration growth from selected command-path depth.
func BenchmarkRunLocaleSelection(b *testing.B) {
	for _, unrelated := range [...]int{0, 16, 64, 256} {
		var observed cli.Locale
		definition := cli.Definition[flagConfig]{
			Name: "tool",
			Localization: cli.Localization{
				DefaultLocale: "en",
			},
			Declare: func(root *cli.CommandBuilder[flagConfig]) {
				locale := root.Values.String(&root.Config.Value)
				root.Flags.ValueFor(locale).LocaleSelector().Occurrences(cli.Count.Optional()).
					Long("locale")
				root.Subcommands.Command(cli.Command[flagConfig]{
					Name: "target",
					Declare: func(target *cli.CommandBuilder[flagConfig]) {
						target.Access.Locale(&target.Config.Locale)
					},
					Run: func(_ context.Context, config *flagConfig) error {
						observed = config.Locale
						return nil
					},
				})
				root.Subcommands.Command(cli.Command[flagConfig]{
					Name: "bulk",
					Declare: func(bulk *cli.CommandBuilder[flagConfig]) {
						for index := range unrelated {
							bulk.Flags.Bool(&bulk.Config.Flags[index]).Long(fmt.Sprintf("flag%04d", index))
						}
					},
				})
			},
		}
		app, err := cli.Compile(definition)
		if err != nil {
			b.Fatalf("locale unrelated compile: %v", err)
		}
		request := cli.RunRequest{
			Args: []string{"--locale", "fr", "target"},
		}
		localePreflight(b, app, request, &observed)
		b.Run(fmt.Sprintf("UnrelatedFlags/N=%d", unrelated), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Run(b, app, request)
		})
	}

	for _, depth := range [...]int{1, 4, 16, 64} {
		var observed cli.Locale
		args := make([]string, 0, 2+depth)
		args = append(args, "--locale", "fr")
		for index := range depth {
			args = append(args, fmt.Sprintf("c%04d", index))
		}
		var pathCommand func(int) cli.Command[flagConfig]
		pathCommand = func(index int) cli.Command[flagConfig] {
			command := cli.Command[flagConfig]{
				Name: fmt.Sprintf("c%04d", index),
			}
			if index+1 == depth {
				command.Declare = func(leaf *cli.CommandBuilder[flagConfig]) {
					leaf.Access.Locale(&leaf.Config.Locale)
				}
				command.Run = func(_ context.Context, config *flagConfig) error {
					observed = config.Locale
					return nil
				}
				return command
			}
			next := index + 1
			command.Declare = func(current *cli.CommandBuilder[flagConfig]) {
				current.Subcommands.Command(pathCommand(next))
			}
			return command
		}
		definition := cli.Definition[flagConfig]{
			Name: "tool",
			Localization: cli.Localization{
				DefaultLocale: "en",
			},
			Declare: func(root *cli.CommandBuilder[flagConfig]) {
				locale := root.Values.String(&root.Config.Value)
				root.Flags.ValueFor(locale).LocaleSelector().Occurrences(cli.Count.Optional()).
					Long("locale")
				root.Subcommands.Command(pathCommand(0))
			},
		}
		app, err := cli.Compile(definition)
		if err != nil {
			b.Fatalf("locale selected-path compile: %v", err)
		}
		request := cli.RunRequest{
			Args: args,
		}
		localePreflight(b, app, request, &observed)
		b.Run(fmt.Sprintf("SelectedPath/Depth=%d", depth), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Run(b, app, request)
		})
	}
}

func localePreflight(b testing.TB, app cli.Application, request cli.RunRequest, observed *cli.Locale) {
	b.Helper()
	*observed = ""
	if err := app.Run(b.Context(), request); err != nil {
		b.Fatalf("locale preflight: %v", err)
	}
	if *observed != "fr" {
		b.Fatalf("locale preflight: got %q want fr", *observed)
	}
}
