package benchmark

import (
	"context"
	"fmt"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clibench"
	"github.com/abstractionsinc/commandment/clitest"
)

// BenchmarkCompleteProviderPublication separates unrelated static command growth from provider-count growth at completion.
func BenchmarkCompleteProviderPublication(b *testing.B) {
	for _, base := range [...]int{16, 64, 256, 1024} {
		app, request := providerPublicationFixture(b, base, 1)
		preflightProviderCompletion(b, app, request, "provider", 1)
		b.Run(fmt.Sprintf("BaseCommands/N=%d/Providers=1", base), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Complete(b, app, request)
		})
	}
	for _, providers := range [...]int{1, 4, 16, 64} {
		app, request := providerPublicationFixture(b, 0, providers)
		preflightProviderCompletion(b, app, request, "provider", providers)
		b.Run(fmt.Sprintf("Providers/N=%d/BaseCommands=0", providers), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Complete(b, app, request)
		})
	}
}

// BenchmarkCompleteProviderPeerSettlement varies provider peers and emitted outputs to expose settlement cost.
func BenchmarkCompleteProviderPeerSettlement(b *testing.B) {
	for _, peers := range [...]int{1, 4, 16, 64} {
		for _, outputs := range [...]int{0, 1} {
			app, request := providerPeerFixture(b, peers, outputs)
			preflightProviderCompletion(b, app, request, "peer", peers*outputs)
			b.Run(fmt.Sprintf("Peers/N=%d/Outputs=%d", peers, outputs), func(b *testing.B) {
				b.ReportAllocs()
				clibench.Complete(b, app, request)
			})
		}
	}
}

// BenchmarkInspectProviderGrammarLocality keeps one selected provider alias fixed and moves an
// equal variable alias set between the selected target and an unrelated decoy.
func BenchmarkInspectProviderGrammarLocality(b *testing.B) {
	for _, placement := range [...]declarationPlacement{placementUnrelated, placementSelected} {
		for _, aliases := range [...]int{16, 64, 256} {
			app, request := providerGrammarLocalityFixture(b, placement, aliases)
			preflightProviderGrammarLocality(b, app, request, placement, aliases)
			b.Run(fmt.Sprintf("Placement=%s/Aliases/N=%d", placement, aliases), func(b *testing.B) {
				b.ReportAllocs()
				clibench.Inspect(b, app, request)
			})
		}
	}
}

// BenchmarkInspectProviderCommandSelection compares static and provider-authored targets in the
// same application while unrelated static command breadth grows.
func BenchmarkInspectProviderCommandSelection(b *testing.B) {
	for _, base := range [...]int{16, 64, 256} {
		app := providerCommandSelectionFixture(b, base)
		for _, target := range []struct {
			name    string
			command string
		}{
			{
				name:    "Static",
				command: "static-target",
			},
			{
				name:    "Provider",
				command: "provider-target",
			},
		} {
			request := cli.InspectionRequest{Args: []string{target.command}}
			preflightProviderCommandSelection(b, app, request, target.command)
			b.Run(fmt.Sprintf("BaseCommands=%d/Target=%s", base, target.name), func(b *testing.B) {
				b.ReportAllocs()
				clibench.Inspect(b, app, request)
			})
		}
	}
}

func providerPublicationFixture(tb testing.TB, base, providers int) (cli.Application, cli.CompletionRequest) {
	tb.Helper()
	definition := cli.Definition[emptyConfig]{
		Name:            "tool",
		Help:            cli.HelpConfig{Disabled: true},
		ShellCompletion: cli.ShellCompletion{Disabled: true},
		Declare: func(root *cli.CommandBuilder[emptyConfig]) {
			root.Subcommands.Command(cli.Command[emptyConfig]{
				Name: "target",
				Declare: func(target *cli.CommandBuilder[emptyConfig]) {
					for index := range providers {
						name := fmt.Sprintf("provider-%04d", index)
						target.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[emptyConfig]) error {
							batch.Command(cli.Command[emptyConfig]{Name: name})
							return nil
						})
					}
				},
			})
			if base != 0 {
				root.Subcommands.Command(cli.Command[emptyConfig]{
					Name: "bulk",
					Declare: func(bulk *cli.CommandBuilder[emptyConfig]) {
						for index := range base {
							bulk.Subcommands.Command(cli.Command[emptyConfig]{Name: fmt.Sprintf("base-%04d", index)})
						}
					},
				})
			}
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("provider publication compile: %v", err)
	}
	return app, cli.CompletionRequest{
		Args:   []string{"target", ""},
		Cursor: cli.CompletionCursor{Argument: 1},
	}
}

func providerPeerFixture(tb testing.TB, peers, outputs int) (cli.Application, cli.CompletionRequest) {
	tb.Helper()
	definition := cli.Definition[emptyConfig]{
		Name:            "tool",
		Help:            cli.HelpConfig{Disabled: true},
		ShellCompletion: cli.ShellCompletion{Disabled: true},
		Declare: func(root *cli.CommandBuilder[emptyConfig]) {
			for index := range peers {
				name := fmt.Sprintf("peer-%04d", index)
				root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[emptyConfig]) error {
					if outputs != 0 {
						batch.Command(cli.Command[emptyConfig]{Name: name})
					}
					return nil
				})
			}
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("provider peer compile: %v", err)
	}
	return app, cli.CompletionRequest{Args: []string{""}, Cursor: cli.CompletionCursor{Argument: 0}}
}

func providerGrammarLocalityFixture(tb testing.TB, placement declarationPlacement, aliases int) (cli.Application, cli.InspectionRequest) {
	tb.Helper()
	definition := cli.Definition[emptyConfig]{
		Name:            "tool",
		Help:            cli.HelpConfig{Disabled: true},
		ShellCompletion: cli.ShellCompletion{Disabled: true},
		Declare: func(root *cli.CommandBuilder[emptyConfig]) {
			root.Subcommands.Command(cli.Command[emptyConfig]{Name: "target"})
			root.Subcommands.Command(cli.Command[emptyConfig]{Name: "decoy"})
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[emptyConfig]) error {
				batch.Alias(cli.CommandPath{"target"}, cli.CommandAlias{Name: "selected"})
				owner := cli.CommandPath{"decoy"}
				if placement == placementSelected {
					owner = cli.CommandPath{"target"}
				}
				for index := range aliases {
					batch.Alias(owner, cli.CommandAlias{Name: fmt.Sprintf("variable-%04d", index)})
				}
				return nil
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("provider grammar locality compile: %v", err)
	}
	request := cli.InspectionRequest{Args: []string{"selected"}}
	return app, request
}

func preflightProviderGrammarLocality(tb testing.TB, app cli.Application, request cli.InspectionRequest, placement declarationPlacement, aliases int) {
	tb.Helper()
	proof := clitest.Inspect(tb.Context(), app, request)
	target := clitest.Command("target")
	decoy := clitest.Command("decoy")
	root := clitest.RootCommand()
	provider := root.ProviderAt(0)
	proof.Require(tb,
		clitest.InspectionReady(),
		clitest.SelectionPathExactly(root, target),
		provider.Outcome(cli.CommandProviderOutcomeAccepted),
		root.Alias("selected").Exists(),
		root.Alias("selected").Targets(target),
	)
	if tb.Failed() {
		return
	}
	owner := decoy
	if placement == placementSelected {
		owner = target
	}
	entries := make([]clitest.CommandEntryLocator, aliases+1)
	entries[0] = root.Alias("selected")
	for index := range aliases {
		alias := root.Alias(fmt.Sprintf("variable-%04d", index))
		entries[index+1] = alias
		proof.Require(tb, alias.Targets(owner))
	}
	proof.Require(tb, provider.EntriesExactly(entries...))
}

func providerCommandSelectionFixture(tb testing.TB, base int) cli.Application {
	tb.Helper()
	definition := cli.Definition[emptyConfig]{
		Name:            "tool",
		Help:            cli.HelpConfig{Disabled: true},
		ShellCompletion: cli.ShellCompletion{Disabled: true},
		Declare: func(root *cli.CommandBuilder[emptyConfig]) {
			root.Subcommands.Command(cli.Command[emptyConfig]{Name: "static-target"})
			for index := range base {
				root.Subcommands.Command(cli.Command[emptyConfig]{Name: fmt.Sprintf("base-%04d", index)})
			}
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[emptyConfig]) error {
				batch.Command(cli.Command[emptyConfig]{Name: "provider-target"})
				return nil
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("provider selection compile: %v", err)
	}
	return app
}

func preflightProviderCommandSelection(tb testing.TB, app cli.Application, request cli.InspectionRequest, command string) {
	tb.Helper()
	providerTarget := clitest.Command("provider-target")
	clitest.Inspect(tb.Context(), app, request).Require(tb,
		clitest.InspectionReady(),
		clitest.SelectionPathExactly(clitest.RootCommand(), clitest.Command(command)),
		clitest.RootCommand().ProviderAt(0).Outcome(cli.CommandProviderOutcomeAccepted),
		clitest.RootCommand().ProviderAt(0).EntriesExactly(providerTarget.CanonicalEntry()),
	)
}

func preflightProviderCompletion(tb testing.TB, app cli.Application, request cli.CompletionRequest, prefix string, want int) {
	tb.Helper()
	items := make([]clitest.CompletionItemLocator, want)
	for index := range items {
		items[index] = clitest.CompletionItem(fmt.Sprintf("%s-%04d", prefix, index))
	}
	clitest.Complete(tb.Context(), app, request).Require(tb, clitest.CompletionItemsExactly(items...))
}
