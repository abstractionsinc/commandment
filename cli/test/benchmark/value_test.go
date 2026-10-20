package benchmark

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clibench"
	"github.com/abstractionsinc/commandment/clitest"
)

type valueGraphShape string

const (
	valueGraphIndependent valueGraphShape = "Independent"
	valueGraphChain       valueGraphShape = "Chain"
)

// BenchmarkInspectValueDependencies contrasts identical destination volume placed on
// the selected command or an unrelated sibling.
func BenchmarkInspectValueDependencies(b *testing.B) {
	for _, placement := range [...]declarationPlacement{placementUnrelated, placementSelected} {
		for _, destinations := range [...]int{16, 64, 256} {
			app, request := valueDependencyFixture(b, placement, destinations)
			preflightValueDependencyPlacement(b, app, request, placement, destinations)
			b.Run(fmt.Sprintf("Placement=%s/Destinations/N=%d", placement, destinations), func(b *testing.B) {
				b.ReportAllocs()
				clibench.Inspect(b, app, request)
			})
		}
	}
}

// BenchmarkInspectValueDependencyGraph compares independent values with a dependency chain as graph size grows.
func BenchmarkInspectValueDependencyGraph(b *testing.B) {
	for _, shape := range [...]valueGraphShape{valueGraphIndependent, valueGraphChain} {
		for _, values := range [...]int{16, 64, 256} {
			app := valueGraphFixture(b, values, shape)
			request := cli.InspectionRequest{}
			preflightValueGraph(b, app, request, shape, values)
			b.Run(fmt.Sprintf("%s/Values/N=%d", shape, values), func(b *testing.B) {
				b.ReportAllocs()
				clibench.Inspect(b, app, request)
			})
		}
	}
}

// BenchmarkInspectDynamicDomains separates entries emitted by one demanded domain from
// demand locality across a fixed static domain set.
func BenchmarkInspectDynamicDomains(b *testing.B) {
	for _, entries := range [...]int{1, 8, 32, 128} {
		app, request := dynamicDomainFixture(b, 1, 1, entries)
		preflightDynamicDomainInspection(b, app, request, 1, 1, entries)
		b.Run(fmt.Sprintf("Entries/N=%d/Domains=1/Demanded=1", entries), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Inspect(b, app, request)
		})
	}
	for _, domains := range [...]int{16, 64} {
		for _, demanded := range [...]int{1, domains} {
			app, request := dynamicDomainFixture(b, domains, demanded, 1)
			preflightDynamicDomainInspection(b, app, request, domains, demanded, 1)
			demandLabel := strconv.Itoa(demanded)
			if demanded == domains {
				demandLabel = "All"
			}
			b.Run(fmt.Sprintf("Demand/Domains=%d/Demanded=%s/EntriesPerDomain=1", domains, demandLabel), func(b *testing.B) {
				b.ReportAllocs()
				clibench.Inspect(b, app, request)
			})
		}
	}
}

func valueDependencyFixture(tb testing.TB, placement declarationPlacement, destinations int) (cli.Application, cli.InspectionRequest) {
	tb.Helper()
	definition := cli.Definition[stringConfig]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[stringConfig]) {
			root.Subcommands.Command(cli.Command[stringConfig]{
				Name: "target",
				Declare: func(target *cli.CommandBuilder[stringConfig]) {
					if placement == placementSelected {
						declareDerivedValues(target, destinations)
					}
				},
			})
			root.Subcommands.Command(cli.Command[stringConfig]{
				Name: "bulk",
				Declare: func(bulk *cli.CommandBuilder[stringConfig]) {
					if placement == placementUnrelated {
						declareDerivedValues(bulk, destinations)
					}
				},
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("value dependency compile: %v", err)
	}
	request := cli.InspectionRequest{Args: []string{"target"}}
	return app, request
}

func declareDerivedValues(command *cli.CommandBuilder[stringConfig], count int) {
	if count == 0 {
		return
	}
	base := command.Values.String(&command.Config.Values[0])
	command.Values.Default(base, "base")
	for index := 1; index < count; index++ {
		storage := &command.Config.Values[index]
		command.Values.Derive(storage, cli.Value.ComputedBy(func(_ context.Context, _ *stringConfig, dependencies cli.Dependencies) (string, error) {
			value, _ := dependencies.Get(base)
			return value, nil
		}).From(base))
	}
}

func preflightValueDependencyPlacement(tb testing.TB, app cli.Application, request cli.InspectionRequest, placement declarationPlacement, destinations int) {
	tb.Helper()
	proof := clitest.Inspect(tb.Context(), app, request)
	target := clitest.Command("target")
	bulk := clitest.Command("bulk")
	proof.Require(tb,
		clitest.InspectionReady(),
		target.Selected(),
	)
	if tb.Failed() {
		return
	}
	owner := bulk
	other := target
	if placement == placementSelected {
		owner = target
		other = bulk
	}
	values := make([]clitest.ValueLocator, destinations)
	for index := range values {
		value := owner.ValueAt(index)
		values[index] = value
		if placement == placementSelected {
			proof.Require(tb, value.Resolved())
			continue
		}
		proof.Require(tb, value.NotObserved())
	}
	schema := clitest.Schema(app)
	schema.Require(tb,
		owner.ValuesExactly(values...),
		other.ValuesExactly(),
	)
}

func valueGraphFixture(tb testing.TB, values int, shape valueGraphShape) cli.Application {
	tb.Helper()
	definition := cli.Definition[stringConfig]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[stringConfig]) {
			var previous cli.Destination[stringConfig, string]
			for index := range values {
				storage := &root.Config.Values[index]
				if shape == valueGraphChain && index != 0 {
					dependency := previous
					previous = root.Values.Derive(storage, cli.Value.ComputedBy(func(_ context.Context, _ *stringConfig, dependencies cli.Dependencies) (string, error) {
						value, _ := dependencies.Get(dependency)
						return value + "x", nil
					}).From(dependency))
					continue
				}
				previous = root.Values.Derive(storage, cli.Value.ComputedBy(func(context.Context, *stringConfig, cli.Dependencies) (string, error) {
					return "x", nil
				}))
			}
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("value graph compile: %v", err)
	}
	return app
}

func preflightValueGraph(tb testing.TB, app cli.Application, request cli.InspectionRequest, shape valueGraphShape, want int) {
	tb.Helper()
	proof := clitest.Inspect(tb.Context(), app, request)
	proof.Require(tb, clitest.InspectionReady())
	root := clitest.RootCommand()
	for index := range want {
		proof.Require(tb, root.ValueAt(index).Resolved())
	}
	terminal := root.ValueAt(want - 1)
	producer := terminal.ProducerAt(0)
	proof.Require(tb,
		terminal.ProducersExactly(producer),
		producer.Kind(cli.ValueProducerKindDerived),
		producer.Outcome(cli.ValueProducerOutcomeContributed),
	)
	if shape == valueGraphChain {
		proof.Require(tb, producer.DependenciesExactly(root.ValueAt(want-2)))
		return
	}
	proof.Require(tb, producer.DependenciesExactly())
}

func dynamicDomainFixture(tb testing.TB, domains, demanded, entriesPerDomain int) (cli.Application, cli.InspectionRequest) {
	tb.Helper()
	args := make([]string, 1, 1+2*demanded)
	args[0] = "target"
	for domainIndex := range demanded {
		args = append(args,
			fmt.Sprintf("--domain-%02d", domainIndex),
			fmt.Sprintf("d%02d-v000", domainIndex),
		)
	}
	definition := cli.Definition[stringConfig]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[stringConfig]) {
			root.Subcommands.Command(cli.Command[stringConfig]{
				Name: "target",
				Declare: func(target *cli.CommandBuilder[stringConfig]) {
					for domainIndex := range domains {
						index := domainIndex
						domain := target.Values.Domain(func(_ context.Context, _ *stringConfig, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
							for entry := range entriesPerDomain {
								builder.Value(
									fmt.Sprintf("d%02d-v%03d", index, entry),
									fmt.Sprintf("domain-%02d-value-%03d", index, entry),
								)
							}
							return nil
						})
						destination := target.Values.Value(&target.Config.Values[index], domain)
						target.Flags.ValueFor(destination).Long(fmt.Sprintf("domain-%02d", index))
					}
				},
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("dynamic-domain compile: %v", err)
	}
	request := cli.InspectionRequest{Args: args}
	return app, request
}

func preflightDynamicDomainInspection(tb testing.TB, app cli.Application, request cli.InspectionRequest, domains, demanded, entriesPerDomain int) {
	tb.Helper()
	proof := clitest.Inspect(tb.Context(), app, request)
	proof.Require(tb,
		clitest.InspectionReady(),
		clitest.Command("target").Selected(),
	)
	if tb.Failed() {
		return
	}
	inspection, err := proof.Inspection()
	if err != nil {
		tb.Fatalf("dynamic-domain inspection: %v", err)
	}
	ready := 0
	notNeeded := 0
	invocationScoped := 0
	materializedEntries := 0
	for _, domain := range inspection.ValueDomains {
		if !domain.InvocationScoped {
			continue
		}
		invocationScoped++
		switch domain.Outcome {
		case cli.ValueDomainOutcomeReady:
			ready++
			materializedEntries += len(domain.Entries)
		case cli.ValueDomainOutcomeNotNeeded:
			notNeeded++
		default:
		}
	}
	if invocationScoped != domains {
		tb.Fatalf("invocation-scoped domains=%d want=%d", invocationScoped, domains)
	}
	if ready != demanded {
		tb.Fatalf("ready domains=%d want=%d", ready, demanded)
	}
	if notNeeded != domains-demanded {
		tb.Fatalf("not-needed domains=%d want=%d", notNeeded, domains-demanded)
	}
	if want := demanded * entriesPerDomain; materializedEntries != want {
		tb.Fatalf("materialized invocation-domain entries=%d want=%d", materializedEntries, want)
	}
}
