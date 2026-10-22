package benchmark

import (
	"fmt"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clibench"
	"github.com/abstractionsinc/commandment/clitest"
)

// BenchmarkInspectEnvironmentMatching separates environment-entry volume, matched producer
// fraction, and the producer×entry interaction that distinguishes additive indexing from nested matching.
func BenchmarkInspectEnvironmentMatching(b *testing.B) {
	for _, entries := range [...]int{32, 128, 512} {
		app, request := environmentFixture(b, 32, 32, entries)
		preflightEnvironmentInspection(b, app, request, 32)
		b.Run(fmt.Sprintf("Entries/N=%d/Producers=32/Matched=32", entries), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Inspect(b, app, request)
		})
	}

	for _, producers := range [...]int{1, 16, 64} {
		app, request := environmentFixture(b, producers, 1, 256)
		preflightEnvironmentInspection(b, app, request, 1)
		b.Run(fmt.Sprintf("MatchedFraction/Producers=%d/Entries=256/Matched=1", producers), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Inspect(b, app, request)
		})
	}

	for _, producers := range [...]int{16, 64} {
		app, request := environmentFixture(b, producers, producers, 256)
		preflightEnvironmentInspection(b, app, request, producers)
		b.Run(fmt.Sprintf("MatchedFraction/Producers=%d/Entries=256/Matched=%d", producers, producers), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Inspect(b, app, request)
		})
	}

	for _, producers := range [...]int{16, 64, 256} {
		entries := 8 * producers
		app, request := environmentFixture(b, producers, producers, entries)
		preflightEnvironmentInspection(b, app, request, producers)
		b.Run(fmt.Sprintf("Interaction/Producers=%d/Entries=%d/Matched=%d", producers, entries, producers), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Inspect(b, app, request)
		})
	}
}

func environmentFixture(tb testing.TB, producers, matched, entries int) (cli.Application, cli.InspectionRequest) {
	tb.Helper()
	definition := cli.Definition[stringConfig]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[stringConfig]) {
			for index := range producers {
				destination := root.Values.String(&root.Config.Values[index])
				root.Values.Environment(destination, fmt.Sprintf("MATCH_%04d", index))
			}
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("environment benchmark compile: %v", err)
	}

	env := make(cli.Env, 0, entries)
	for index := range matched {
		env = append(env, cli.EnvEntry{
			Name:  fmt.Sprintf("MATCH_%04d", index),
			Value: fmt.Sprintf("value-%04d", index),
		})
	}
	for len(env) < entries {
		index := len(env)
		env = append(env, cli.EnvEntry{
			Name:  fmt.Sprintf("UNRELATED_%04d", index),
			Value: "value",
		})
	}
	request := cli.InspectionRequest{
		Env: env,
	}
	return app, request
}

func preflightEnvironmentInspection(tb testing.TB, app cli.Application, request cli.InspectionRequest, matched int) {
	tb.Helper()
	proof := clitest.Inspect(tb.Context(), app, request)
	proof.Require(tb, clitest.InspectionReady())
	if tb.Failed() {
		return
	}
	inspection, err := proof.Inspection()
	if err != nil {
		tb.Fatalf("environment benchmark preflight: %v", err)
	}
	if len(inspection.EnvironmentInputs) == 0 && matched != 0 {
		tb.Fatal("environment benchmark preflight observed no declared environment inputs")
	}
	resolved := 0
	for _, value := range inspection.Values {
		if value.State != cli.ResolutionStateResolved {
			continue
		}
		for _, producer := range value.Producers {
			if producer.Kind == cli.ValueProducerKindEnvironment && producer.Outcome == cli.ValueProducerOutcomeContributed {
				resolved++
				break
			}
		}
	}
	if resolved != matched {
		tb.Fatalf("environment matched values=%d want=%d", resolved, matched)
	}
	for index := range matched {
		root := clitest.RootCommand()
		proof.Require(tb,
			root.ValueAt(index).Resolved(),
			root.ValueAt(index).Text(fmt.Sprintf("value-%04d", index)),
		)
	}
}
