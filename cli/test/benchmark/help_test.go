package benchmark

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clibench"
	"github.com/abstractionsinc/commandment/clitest"
)

func BenchmarkHelp(b *testing.B) {
	for _, shape := range []string{"Commands", "Flags", "Forms", "Groups"} {
		for _, units := range [...]int{1, 4, 16, 64} {
			var page cli.HelpPage
			definition := cli.Definition[helpConfig]{
				Name: "tool",
				Help: cli.HelpConfig{
					Renderer: func(_ context.Context, got cli.HelpPage) (string, error) {
						page = got
						return "help", nil
					},
				},
				Declare: func(root *cli.CommandBuilder[helpConfig]) {
					switch shape {
					case "Commands":
						for index := range units {
							root.Subcommands.Command(cli.Command[helpConfig]{
								Name: fmt.Sprintf("command-%03d", index),
							})
						}
					case "Flags":
						for index := range units {
							root.Flags.Bool(&root.Config.Flags[index]).Long(fmt.Sprintf("flag-%03d", index))
						}
					case "Forms":
						for index := range units {
							literal := fmt.Sprintf("literal-%03d", index)
							root.Arguments.Form(
								func(form *cli.ArgumentFormBuilder[helpConfig]) { form.Literal(literal) },
							)
						}
					case "Groups":
						for index := range units {
							flagIndex := index
							root.Flags.HelpGroup(
								fmt.Sprintf("Group %03d", index),
								func(flags *cli.Flags[helpConfig]) {
									flags.Bool(&flags.Config.Flags[flagIndex]).Long(fmt.Sprintf("flag-%03d", flagIndex))
								},
							).OrderByName()
						}
					}
				},
			}
			app, err := cli.Compile(definition)
			if err != nil {
				b.Fatalf("Help compile preflight: %v", err)
			}
			request := cli.RunRequest{
				Args:   []string{"--help"},
				Stdout: io.Discard,
			}
			if err := app.Run(b.Context(), request); err != nil {
				b.Fatalf("Help preflight: %v", err)
			}
			preflightHelpPage(b, page, shape, units)
			b.Run(fmt.Sprintf("%s/N=%d", shape, units), func(b *testing.B) {
				b.ReportAllocs()
				clibench.Run(b, app, request)
			})
		}
	}
}

// BenchmarkHelpDependencyClosure checks that help cost follows relevant value dependencies instead of unrelated dynamic domains.
func BenchmarkHelpDependencyClosure(b *testing.B) {
	for _, unrelated := range [...]int{0, 16, 64, 256} {
		b.Run(fmt.Sprintf("UnrelatedDomains/N=%d", unrelated), func(b *testing.B) {
			calls := new(int)
			definition := cli.Definition[helpConfig]{
				Name: "tool",
				Help: helpAction(),
				Declare: func(root *cli.CommandBuilder[helpConfig]) {
					relevantDomain := root.Values.Domain(
						func(_ context.Context, _ *helpConfig, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
							builder.Value("on", "on")
							return nil
						},
					)
					relevant := root.Values.Value(&root.Config.Values[0], relevantDomain)
					root.Values.ComputedDefault(
						relevant,
						cli.Value.ComputedBy(
							func(context.Context, *helpConfig, cli.Dependencies) (string, error) {
								return "on", nil
							},
						),
					)
					visible := root.Flags.String(&root.Config.Visible)
					visible.Long("visible")
					visible.VisibleWhen(root.When.Resolved(relevant))
					for index := range unrelated {
						domain := root.Values.Domain(
							func(_ context.Context, _ *helpConfig, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
								(*calls)++
								value := fmt.Sprintf("u%04d", index)
								builder.Value(value, value)
								return nil
							},
						)
						root.Values.Value(&root.Config.Values[index+1], domain)
					}
				},
			}
			app, err := cli.Compile(definition)
			if err != nil {
				b.Fatalf("Help dependency compile: %v", err)
			}
			request := cli.RunRequest{
				Args:   []string{"--help"},
				Stdout: io.Discard,
			}
			helpPreflight(b, app, request)
			if *calls != 0 {
				b.Fatalf("Help preflight executed %d unrelated domains", *calls)
			}
			b.ReportAllocs()
			clibench.Run(b, app, request)
			if *calls != 0 {
				b.Fatalf("Help workload executed %d unrelated domains", *calls)
			}
		})
	}

	for _, relevant := range [...]int{1, 4, 16, 64} {
		b.Run(fmt.Sprintf("RelevantFanIn/N=%d", relevant), func(b *testing.B) {
			request := cli.RunRequest{
				Args:   []string{"--help"},
				Stdout: io.Discard,
			}
			var page cli.HelpPage
			preflightApp := helpFanInApplication(b, relevant, capturingHelpAction(&page))
			helpPreflight(b, preflightApp, request)
			preflightHelpDependencyPage(b, page)
			app := helpFanInApplication(b, relevant, helpAction())
			b.ReportAllocs()
			clibench.Run(b, app, request)
		})
		b.Run(fmt.Sprintf("RelevantChain/N=%d", relevant), func(b *testing.B) {
			request := cli.RunRequest{
				Args:   []string{"--help"},
				Stdout: io.Discard,
			}
			var page cli.HelpPage
			preflightApp := helpChainApplication(b, relevant, capturingHelpAction(&page))
			helpPreflight(b, preflightApp, request)
			preflightHelpDependencyPage(b, page)
			app := helpChainApplication(b, relevant, helpAction())
			b.ReportAllocs()
			clibench.Run(b, app, request)
		})
	}
}

func helpAction() cli.HelpConfig {
	config := cli.HelpConfig{
		Renderer: func(context.Context, cli.HelpPage) (string, error) {
			return "help", nil
		},
	}
	return config
}

func capturingHelpAction(page *cli.HelpPage) cli.HelpConfig {
	config := cli.HelpConfig{
		Renderer: func(_ context.Context, got cli.HelpPage) (string, error) {
			*page = got
			return "help", nil
		},
	}
	return config
}

func helpFanInApplication(b testing.TB, relevant int, help cli.HelpConfig) cli.Application {
	b.Helper()
	definition := cli.Definition[helpConfig]{
		Name: "tool",
		Help: help,
		Declare: func(root *cli.CommandBuilder[helpConfig]) {
			destinations := make([]cli.Destination[helpConfig, string], 0, relevant)
			for index := range relevant {
				var dependencies []cli.Dependency
				if index == relevant-1 {
					for _, destination := range destinations {
						dependencies = append(dependencies, destination)
					}
				}
				value := fmt.Sprintf("v%04d", index)
				domain := root.Values.Domain(
					func(_ context.Context, _ *helpConfig, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
						builder.Value(value, value)
						return nil
					},
					dependencies...,
				)
				destination := root.Values.Value(&root.Config.Values[index], domain)
				root.Values.ComputedDefault(
					destination,
					cli.Value.ComputedBy(func(context.Context, *helpConfig, cli.Dependencies) (string, error) {
						return value, nil
					}),
				)
				destinations = append(destinations, destination)
			}
			visible := root.Flags.String(&root.Config.Visible)
			visible.Long("visible")
			visible.VisibleWhen(root.When.Resolved(destinations[len(destinations)-1]))
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		b.Fatalf("Help fan-in compile: %v", err)
	}
	return app
}

func helpChainApplication(b testing.TB, relevant int, help cli.HelpConfig) cli.Application {
	b.Helper()
	definition := cli.Definition[helpConfig]{
		Name: "tool",
		Help: help,
		Declare: func(root *cli.CommandBuilder[helpConfig]) {
			destinations := make([]cli.Destination[helpConfig, string], 0, relevant)
			for index := range relevant {
				var dependencies []cli.Dependency
				if index > 0 {
					dependencies = []cli.Dependency{destinations[index-1]}
				}
				value := fmt.Sprintf("v%04d", index)
				domain := root.Values.Domain(
					func(_ context.Context, _ *helpConfig, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
						builder.Value(value, value)
						return nil
					},
					dependencies...,
				)
				destination := root.Values.Value(&root.Config.Values[index], domain)
				root.Values.ComputedDefault(
					destination,
					cli.Value.ComputedBy(func(context.Context, *helpConfig, cli.Dependencies) (string, error) {
						return value, nil
					}),
				)
				destinations = append(destinations, destination)
			}
			visible := root.Flags.String(&root.Config.Visible)
			visible.Long("visible")
			visible.VisibleWhen(root.When.Resolved(destinations[len(destinations)-1]))
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		b.Fatalf("Help chain compile: %v", err)
	}
	return app
}

func helpPreflight(b testing.TB, app cli.Application, request cli.RunRequest) {
	b.Helper()
	inspectionRequest := cli.InspectionRequest{
		Args: []string{"--help"},
	}
	proof := clitest.Inspect(b.Context(), app, inspectionRequest)
	proof.Require(b,
		clitest.InvocationAction(cli.InvocationActionHelp).Selected(),
		clitest.RootCommand().Flag("--visible").Visible(),
	)
	if b.Failed() {
		return
	}
	if err := app.Run(b.Context(), request); err != nil {
		b.Fatalf("Help preflight: %v", err)
	}
}

func preflightHelpDependencyPage(tb testing.TB, page cli.HelpPage) {
	tb.Helper()
	if got := benchmarkHelpFlagCount(page, "--visible"); got != 1 {
		tb.Fatalf("Help dependency preflight visible flags=%d want=1", got)
	}
}

func preflightHelpPage(tb testing.TB, page cli.HelpPage, shape string, want int) {
	tb.Helper()
	switch shape {
	case "Commands":
		got := 0
		for _, group := range page.CommandGroups {
			for _, command := range group.Commands {
				if strings.HasPrefix(command.Name, "command-") {
					got++
				}
			}
		}
		if got != want {
			tb.Fatalf("Help command rows=%d want=%d", got, want)
		}
	case "Flags":
		got := benchmarkHelpFlagCount(page, "--flag-")
		if got != want {
			tb.Fatalf("Help flag rows=%d want=%d", got, want)
		}
	case "Forms":
		if len(page.ArgumentForms) != want {
			tb.Fatalf("Help argument forms=%d want=%d", len(page.ArgumentForms), want)
		}
		for index, form := range page.ArgumentForms {
			if len(form.Terms) != 1 || form.Terms[0].Literal != fmt.Sprintf("literal-%03d", index) {
				tb.Fatalf("Help argument form %d = %#v, want one literal-%03d term", index, form.Terms, index)
			}
		}
	case "Groups":
		groups := 0
		for _, group := range page.FlagGroups {
			if !strings.HasPrefix(group.Title, "Group ") {
				continue
			}
			groups++
			if len(group.Flags) != 1 {
				tb.Fatalf("Help group %q flags=%d want=1", group.Title, len(group.Flags))
			}
		}
		if groups != want {
			tb.Fatalf("Help custom groups=%d want=%d", groups, want)
		}
	default:
		tb.Fatalf("unknown Help benchmark shape %q", shape)
	}
}

func benchmarkHelpFlagCount(page cli.HelpPage, prefix string) int {
	count := 0
	for _, group := range page.FlagGroups {
		for _, flag := range group.Flags {
			matched := false
			for _, spellingID := range flag.FlagSpellings {
				spelling := page.FlagSpellings[spellingID].Spelling.Text
				if strings.HasPrefix(spelling, prefix) {
					matched = true
					break
				}
			}
			if matched {
				count++
			}
		}
	}
	return count
}
