package benchmark

import (
	"fmt"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clibench"
	"github.com/abstractionsinc/commandment/clitest"
)

func BenchmarkInspectCommandAliases(b *testing.B) {
	for _, commands := range [...]int{16, 64, 256} {
		app, request := commandAliasFixture(b, commands, 0)
		preflightCanonicalCommandSelection(b, app, request)
		b.Run(fmt.Sprintf("Canonical/Commands/N=%d", commands), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Inspect(b, app, request)
		})
	}
	for _, aliases := range [...]int{16, 64, 256} {
		app, request := commandAliasFixture(b, 256, aliases)
		preflightCommandAlias(b, app, request)
		b.Run(fmt.Sprintf("Aliases/Commands=256/N=%d", aliases), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Inspect(b, app, request)
		})
	}
	for _, count := range [...]int{16, 64, 256} {
		app, request := commandAliasFixture(b, count, count)
		preflightCommandAlias(b, app, request)
		b.Run(fmt.Sprintf("Aliases/Proportional/N=%d", count), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Inspect(b, app, request)
		})
	}
}

// BenchmarkInspectGrammarLocality holds static projection constant while moving the same flag set
// between the selected command and an unrelated sibling.
func BenchmarkInspectGrammarLocality(b *testing.B) {
	for _, placement := range [...]declarationPlacement{placementUnrelated, placementSelected} {
		for _, flags := range [...]int{16, 64, 256} {
			app, request := grammarLocalityFixture(b, placement, flags)
			preflightGrammarLocality(b, app, request, placement, flags)
			b.Run(fmt.Sprintf("Placement=%s/Flags/N=%d", placement, flags), func(b *testing.B) {
				b.ReportAllocs()
				clibench.Inspect(b, app, request)
			})
		}
	}
}

// BenchmarkInspectCommandSelection protects exact command lookup from becoming declaration-position sensitive.
func BenchmarkInspectCommandSelection(b *testing.B) {
	for _, siblings := range [...]int{16, 64, 256} {
		for _, target := range []struct {
			name  string
			index int
		}{
			{
				name:  "First",
				index: 0,
			},
			{
				name:  "Last",
				index: siblings - 1,
			},
		} {
			app, request := commandSelectionBreadthFixture(b, siblings, target.index)
			selected := fmt.Sprintf("command-%04d", target.index)
			clitest.Inspect(b.Context(), app, request).Require(b,
				clitest.InspectionReady(),
				clitest.SelectionPathExactly(clitest.RootCommand(), clitest.Command(selected)),
			)
			b.Run(fmt.Sprintf("Breadth/Siblings=%d/Target=%s", siblings, target.name), func(b *testing.B) {
				b.ReportAllocs()
				clibench.Inspect(b, app, request)
			})
		}
	}
}

// BenchmarkInspectFlagMatching protects exact long/short flag lookup from becoming declaration-position sensitive.
func BenchmarkInspectFlagMatching(b *testing.B) {
	for _, spelling := range []struct {
		name string
		kind cli.FlagSpellingKind
	}{
		{
			name: "Long",
			kind: cli.FlagSpellingLong,
		},
		{
			name: "Short",
			kind: cli.FlagSpellingShort,
		},
	} {
		for _, flags := range [...]int{16, 64, 256} {
			for _, target := range []struct {
				name  string
				index int
			}{
				{
					name:  "First",
					index: 0,
				},
				{
					name:  "Last",
					index: flags - 1,
				},
			} {
				app, request := flagMatchingFixture(b, flags, target.index, spelling.kind)
				flag := clitest.RootCommand().FlagAt(target.index)
				clitest.Inspect(b.Context(), app, request).Require(b,
					clitest.InspectionReady(),
					clitest.SelectionPathExactly(clitest.RootCommand()),
					flag.Present(),
					flag.Value().Resolved(),
					flag.Value().Text("true"),
				)
				b.Run(fmt.Sprintf("%s/Flags=%d/Target=%s", spelling.name, flags, target.name), func(b *testing.B) {
					b.ReportAllocs()
					clibench.Inspect(b, app, request)
				})
			}
		}
	}
}

func BenchmarkInspectGrammarNormalization(b *testing.B) {
	for _, normalization := range []struct {
		name string
		mode cli.NameNormalization
	}{
		{
			name: "Exact",
			mode: cli.NameNormalizationExact,
		},
		{
			name: "ASCIICaseFold",
			mode: cli.NameNormalizationASCIICaseFold,
		},
	} {
		for _, nameBytes := range [...]int{8, 64, 256} {
			app, request, commandName, flagName := grammarNormalizationFixture(b, normalization.mode, nameBytes)
			flag := clitest.Command(commandName).Flag("--" + flagName)
			clitest.Inspect(b.Context(), app, request).Require(b,
				clitest.InspectionReady(),
				clitest.SelectionPathExactly(clitest.RootCommand(), clitest.Command(commandName)),
				flag.Present(),
				flag.Value().Resolved(),
				flag.Value().Text("value"),
			)
			b.Run(fmt.Sprintf("Normalization=%s/NameBytes=%d", normalization.name, nameBytes), func(b *testing.B) {
				b.ReportAllocs()
				clibench.Inspect(b, app, request)
			})
		}
	}
}

func grammarNormalizationFixture(tb testing.TB, normalization cli.NameNormalization, nameBytes int) (cli.Application, cli.InspectionRequest, string, string) {
	tb.Helper()
	commandName := "c" + strings.Repeat("a", nameBytes-1)
	flagName := "f" + strings.Repeat("a", nameBytes-1)
	definition := cli.Definition[inputConfig]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[inputConfig]) {
			switch normalization {
			case cli.NameNormalizationExact:
			case cli.NameNormalizationASCIICaseFold:
				root.Dialect.CommandNormalization(cli.NameNormalizationASCIICaseFold)
				root.Dialect.FlagNormalization(cli.NameNormalizationASCIICaseFold)
			default:
				tb.Fatalf("unsupported grammar normalization %v", normalization)
			}
			root.Subcommands.Command(cli.Command[inputConfig]{
				Name: commandName,
				Declare: func(target *cli.CommandBuilder[inputConfig]) {
					target.Flags.String(&target.Config.Value).Long(flagName)
				},
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("grammar normalization compile: %v", err)
	}
	commandInput := commandName
	flagInput := flagName
	if normalization == cli.NameNormalizationASCIICaseFold {
		commandInput = strings.ToUpper(commandName)
		flagInput = strings.ToUpper(flagName)
	}
	request := cli.InspectionRequest{
		Args: []string{commandInput, "--" + flagInput, "value"},
	}
	return app, request, commandName, flagName
}

func commandAliasFixture(tb testing.TB, commands, aliases int) (cli.Application, cli.InspectionRequest) {
	tb.Helper()
	definition := cli.Definition[emptyConfig]{
		Name:    "tool",
		Release: cli.Release.SemVer("1.0.0"),
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[emptyConfig]) {
			for index := range commands {
				commandAliases := make([]cli.CommandAlias, 0)
				for alias := index; alias < aliases; alias += commands {
					commandAliases = append(commandAliases, cli.CommandAlias{
						Name: fmt.Sprintf("alias-%04d", alias),
						Release: cli.ReleaseLifecycle{
							IntroducedIn: "1.0.0",
						},
					})
				}
				root.Subcommands.Command(cli.Command[emptyConfig]{
					Name:    fmt.Sprintf("command-%04d", index),
					Aliases: commandAliases,
				})
			}
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("command alias compile: %v", err)
	}
	selected := "command-0000"
	if aliases != 0 {
		selected = "alias-0000"
	}
	request := cli.InspectionRequest{
		Args: []string{selected},
	}
	return app, request
}

func preflightCanonicalCommandSelection(tb testing.TB, app cli.Application, request cli.InspectionRequest) {
	tb.Helper()
	target := clitest.Command("command-0000")
	clitest.Inspect(tb.Context(), app, request).Require(tb,
		clitest.InspectionReady(),
		clitest.SelectionPathExactly(clitest.RootCommand(), target),
	)
}

func preflightCommandAlias(tb testing.TB, app cli.Application, request cli.InspectionRequest) {
	tb.Helper()
	target := clitest.Command("command-0000")
	entry := clitest.RootCommand().Alias("alias-0000")
	clitest.Inspect(tb.Context(), app, request).Require(tb,
		clitest.InspectionReady(),
		clitest.SelectionPathExactly(clitest.RootCommand(), target),
		entry.Exists(),
		entry.Targets(target),
		entry.ReleaseLifecycle(cli.ReleaseLifecycle{
			IntroducedIn: "1.0.0",
		}),
		entry.Lifecycle(cli.LifecycleStateCurrent),
	)
}

func grammarLocalityFixture(tb testing.TB, placement declarationPlacement, flags int) (cli.Application, cli.InspectionRequest) {
	tb.Helper()
	declareFlags := func(command *cli.CommandBuilder[flagConfig]) {
		for index := range flags {
			command.Flags.Bool(&command.Config.Flags[index]).Long(fmt.Sprintf("flag-%04d", index))
		}
	}
	definition := cli.Definition[flagConfig]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[flagConfig]) {
			root.Subcommands.Command(cli.Command[flagConfig]{
				Name: "target",
				Declare: func(target *cli.CommandBuilder[flagConfig]) {
					if placement == placementSelected {
						declareFlags(target)
					}
				},
			})
			root.Subcommands.Command(cli.Command[flagConfig]{
				Name: "decoy",
				Declare: func(decoy *cli.CommandBuilder[flagConfig]) {
					if placement == placementUnrelated {
						declareFlags(decoy)
					}
				},
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("grammar locality compile: %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"target"},
	}
	return app, request
}

func preflightGrammarLocality(tb testing.TB, app cli.Application, request cli.InspectionRequest, placement declarationPlacement, flags int) {
	tb.Helper()
	target := clitest.Command("target")
	decoy := clitest.Command("decoy")
	owner := decoy
	other := target
	if placement == placementSelected {
		owner = target
		other = decoy
	}
	declared := make([]clitest.FlagLocator, flags)
	for index := range declared {
		declared[index] = owner.Flag(fmt.Sprintf("--flag-%04d", index))
	}
	clitest.Inspect(tb.Context(), app, request).Require(tb,
		clitest.InspectionReady(),
		clitest.SelectionPathExactly(clitest.RootCommand(), target),
		owner.FlagsExactly(declared...),
		other.FlagsExactly(),
	)
}

func commandSelectionBreadthFixture(tb testing.TB, siblings, target int) (cli.Application, cli.InspectionRequest) {
	tb.Helper()
	definition := cli.Definition[emptyConfig]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[emptyConfig]) {
			for index := range siblings {
				root.Subcommands.Command(cli.Command[emptyConfig]{
					Name: fmt.Sprintf("command-%04d", index),
				})
			}
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("command breadth compile: %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{fmt.Sprintf("command-%04d", target)},
	}
	return app, request
}

func flagMatchingFixture(tb testing.TB, flags, target int, kind cli.FlagSpellingKind) (cli.Application, cli.InspectionRequest) {
	tb.Helper()
	definition := cli.Definition[flagConfig]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[flagConfig]) {
			shortName := '\uE000'
			for index := range flags {
				flag := root.Flags.Bool(&root.Config.Flags[index])
				switch kind {
				case cli.FlagSpellingLong:
					flag.Long(fmt.Sprintf("flag-%04d", index))
				case cli.FlagSpellingShort:
					flag.Short(shortName)
					shortName++
				default:
					tb.Fatalf("unsupported flag spelling kind %v", kind)
				}
			}
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("flag matching compile: %v", err)
	}
	spelling := fmt.Sprintf("--flag-%04d", target)
	if kind == cli.FlagSpellingShort {
		shortName := '\uE000'
		for range target {
			shortName++
		}
		spelling = "-" + string(shortName)
	}
	request := cli.InspectionRequest{
		Args: []string{spelling},
	}
	return app, request
}
