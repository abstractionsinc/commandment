package benchmark

import (
	"fmt"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clibench"
	"github.com/abstractionsinc/commandment/clitest"
)

func BenchmarkInspect(b *testing.B) {
	definition := cli.Definition[emptyConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[emptyConfig]) {
			root.Subcommands.Command(cli.Command[emptyConfig]{
				Name: "target",
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		b.Fatalf("Inspect compile preflight: %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"target"},
	}
	clitest.Inspect(b.Context(), app, request).Require(b,
		clitest.InspectionReady(),
		clitest.SelectionPathExactly(clitest.RootCommand(), clitest.Command("target")),
	)
	if b.Failed() {
		return
	}
	b.ReportAllocs()
	clibench.Inspect(b, app, request)
}

// BenchmarkInspectArgumentsAndEnvironment varies detached request input size independently for arguments and environment.
func BenchmarkInspectArgumentsAndEnvironment(b *testing.B) {
	app := inspectionInputApplication(b)
	zero := cli.InspectionRequest{}
	preflightInspectionInputs(b, app, zero)
	b.Run("Zero", func(b *testing.B) {
		b.ReportAllocs()
		clibench.Inspect(b, app, zero)
	})

	for _, tokens := range [...]int{1, 16, 64, 256, 1024} {
		args := make([]string, tokens)
		for index := range args {
			args[index] = fmt.Sprintf("arg-%04d", index)
		}
		request := cli.InspectionRequest{Args: args}
		preflightInspectionInputs(b, app, request)
		b.Run(fmt.Sprintf("Args/Tokens=%d", tokens), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Inspect(b, app, request)
		})
	}

	for _, entries := range [...]int{1, 16, 64, 256, 1024} {
		env := make(cli.Env, entries)
		for index := range env {
			env[index] = cli.EnvEntry{
				Name:  fmt.Sprintf("KEY_%04d", index),
				Value: fmt.Sprintf("value-%04d", index),
			}
		}
		request := cli.InspectionRequest{Env: env}
		preflightInspectionInputs(b, app, request)
		b.Run(fmt.Sprintf("Environment/Entries=%d", entries), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Inspect(b, app, request)
		})
	}
}

func inspectionInputApplication(tb testing.TB) cli.Application {
	tb.Helper()
	definition := cli.Definition[inputConfig]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[inputConfig]) {
			values := root.Values.Sequence(&root.Config.Strings, cli.Value.String())
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[inputConfig]) {
				form.Sequence(values, "values", cli.Count.ZeroOrMore()).Greedy()
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("inspection input benchmark compile: %v", err)
	}
	return app
}

func preflightInspectionInputs(tb testing.TB, app cli.Application, request cli.InspectionRequest) {
	tb.Helper()
	proof := clitest.Inspect(tb.Context(), app, request)
	proof.Require(tb, clitest.InspectionReady())
	if tb.Failed() {
		return
	}
	inspection, err := proof.Inspection()
	if err != nil {
		tb.Fatalf("inspection input preflight: %v", err)
	}
	if len(inspection.Tokens) != len(request.Args) {
		tb.Fatalf("detached argument tokens=%d want=%d", len(inspection.Tokens), len(request.Args))
	}
	for index, arg := range request.Args {
		if inspection.Tokens[index].Text != arg {
			tb.Fatalf("detached argument token[%d]=%q want=%q", index, inspection.Tokens[index].Text, arg)
		}
	}

	var argumentSource, environmentSource cli.InputSourceID
	for id, source := range inspection.Sources {
		switch source.Kind {
		case cli.InputKindArguments:
			argumentSource = id
		case cli.InputKindEnvironment:
			environmentSource = id
		default:
		}
	}
	argumentOrigins := make(map[int]cli.InputOrigin)
	environmentOrigins := make(map[int]cli.InputOrigin)
	for _, origin := range inspection.Origins {
		switch origin.Span.Source {
		case argumentSource:
			argumentOrigins[origin.Span.Entry] = origin
		case environmentSource:
			environmentOrigins[origin.Span.Entry] = origin
		}
	}
	if len(argumentOrigins) != len(request.Args) {
		tb.Fatalf("detached argument origins=%d want=%d", len(argumentOrigins), len(request.Args))
	}
	if len(environmentOrigins) != len(request.Env) {
		tb.Fatalf("detached environment origins=%d want=%d", len(environmentOrigins), len(request.Env))
	}
	for index, entry := range request.Env {
		origin, ok := environmentOrigins[index]
		if !ok {
			tb.Fatalf("detached environment entry %d has no origin", index)
		}
		wantEnd := len(entry.Name) + 1 + len(entry.Value)
		if origin.Span.Start.Offset != 0 || origin.Span.End.Offset != wantEnd {
			tb.Fatalf("detached environment entry %d span=%d:%d want=0:%d", index, origin.Span.Start.Offset, origin.Span.End.Offset, wantEnd)
		}
	}
}
