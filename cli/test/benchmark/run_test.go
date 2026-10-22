package benchmark

import (
	"context"
	"fmt"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clibench"
	"github.com/abstractionsinc/commandment/clitest"
)

func BenchmarkRun(b *testing.B) {
	definition := cli.Definition[emptyConfig]{
		Name: "tool",
		Run:  func(context.Context, *emptyConfig) error { return nil },
	}
	app, err := cli.Compile(definition)
	if err != nil {
		b.Fatalf("Run compile preflight: %v", err)
	}
	request := cli.RunRequest{}
	if err := app.Run(b.Context(), request); err != nil {
		b.Fatalf("Run preflight: %v", err)
	}
	b.ReportAllocs()
	clibench.Run(b, app, request)
}

// BenchmarkRunMiddleware varies middleware count and selected command-path depth independently.
func BenchmarkRunMiddleware(b *testing.B) {
	for _, middleware := range [...]int{0, 1, 4, 16, 64} {
		fixture := runMiddlewareSingleApplication(b, middleware)
		runMiddlewarePreflight(b, fixture)
		b.Run(fmt.Sprintf("Middleware/N=%d/Depth=1", middleware), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Run(b, fixture.app, fixture.request)
		})
	}
	for _, depth := range [...]int{4, 16} {
		fixture := runMiddlewarePathApplication(b, 64, depth)
		runMiddlewarePreflight(b, fixture)
		b.Run(fmt.Sprintf("Middleware/N=64/Depth=%d", depth), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Run(b, fixture.app, fixture.request)
		})
	}
}

type runMiddlewareFixture struct {
	app        cli.Application
	request    cli.RunRequest
	middleware int
	depth      int
}

func runMiddlewareNext(ctx context.Context, _ *emptyConfig, next cli.MiddlewareNext) error {
	return next(ctx)
}

func runMiddlewareSlice(count int) []cli.Middleware[emptyConfig] {
	result := make([]cli.Middleware[emptyConfig], count)
	for index := range result {
		result[index] = runMiddlewareNext
	}
	return result
}

func runMiddlewareSingleApplication(b testing.TB, middleware int) runMiddlewareFixture {
	b.Helper()
	return runMiddlewareApplication(b, middleware, 1, runMiddlewareSlice)
}

func runMiddlewarePathApplication(b testing.TB, middleware, depth int) runMiddlewareFixture {
	b.Helper()
	return runMiddlewareApplication(b, middleware, depth, runMiddlewareSlice)
}

func runMiddlewareApplication(b testing.TB, middleware, depth int, middlewareSlice func(int) []cli.Middleware[emptyConfig]) runMiddlewareFixture {
	b.Helper()
	base := middleware / depth
	extra := middleware % depth
	countForLevel := func(level int) int {
		count := base
		if level < extra {
			count++
		}
		return count
	}
	definition := cli.Definition[emptyConfig]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Middleware: middlewareSlice(countForLevel(0)),
	}
	if depth == 1 {
		definition.Run = func(context.Context, *emptyConfig) error { return nil }
	} else {
		definition.Declare = func(root *cli.CommandBuilder[emptyConfig]) {
			declareRunMiddlewarePath(root, depth, 1, countForLevel, middlewareSlice)
		}
	}
	app, err := cli.Compile(definition)
	if err != nil {
		b.Fatalf("Run middleware compile: %v", err)
	}
	args := make([]string, depth-1)
	for level := range depth - 1 {
		args[level] = fmt.Sprintf("level-%03d", level+1)
	}
	request := cli.RunRequest{
		Args: args,
	}
	fixture := runMiddlewareFixture{
		app:        app,
		request:    request,
		middleware: middleware,
		depth:      depth,
	}
	return fixture
}

func declareRunMiddlewarePath(parent *cli.CommandBuilder[emptyConfig], depth, level int, countForLevel func(int) int, middlewareSlice func(int) []cli.Middleware[emptyConfig]) {
	command := cli.Command[emptyConfig]{
		Name:       fmt.Sprintf("level-%03d", level),
		Middleware: middlewareSlice(countForLevel(level)),
	}
	if level == depth-1 {
		command.Run = func(context.Context, *emptyConfig) error { return nil }
	} else {
		command.Declare = func(child *cli.CommandBuilder[emptyConfig]) {
			declareRunMiddlewarePath(child, depth, level+1, countForLevel, middlewareSlice)
		}
	}
	parent.Subcommands.Command(command)
}

func runMiddlewarePreflight(b testing.TB, fixture runMiddlewareFixture) {
	b.Helper()
	if err := fixture.app.Run(b.Context(), fixture.request); err != nil {
		b.Fatalf("Run middleware preflight: %v", err)
	}
	path := make([]clitest.CommandLocator, 0, fixture.depth)
	path = append(path, clitest.RootCommand())
	parts := make([]string, 0, fixture.depth-1)
	for level := 1; level < fixture.depth; level++ {
		parts = append(parts, fmt.Sprintf("level-%03d", level))
		path = append(path, clitest.Command(parts...))
	}
	inspectionRequest := cli.InspectionRequest{
		Args: fixture.request.Args,
	}
	clitest.Inspect(b.Context(), fixture.app, inspectionRequest).Require(b,
		clitest.InspectionReady(),
		clitest.SelectionPathExactly(path...),
	)
	if b.Failed() {
		return
	}
	if schema := fixture.app.Schema(); len(schema.Commands) != fixture.depth {
		b.Fatalf("Run middleware preflight: commands=%d want=%d", len(schema.Commands), fixture.depth)
	}

	var observed []int
	nextID := 0
	instrumented := runMiddlewareApplication(
		b,
		fixture.middleware,
		fixture.depth,
		func(count int) []cli.Middleware[emptyConfig] {
			result := make([]cli.Middleware[emptyConfig], count)
			for index := range result {
				id := nextID
				nextID++
				result[index] = func(ctx context.Context, _ *emptyConfig, next cli.MiddlewareNext) error {
					observed = append(observed, id)
					return next(ctx)
				}
			}
			return result
		},
	)
	if nextID != fixture.middleware {
		b.Fatalf("Run middleware preflight: declared callbacks=%d want=%d", nextID, fixture.middleware)
	}
	if err := instrumented.app.Run(b.Context(), instrumented.request); err != nil {
		b.Fatalf("Run middleware callback preflight: %v", err)
	}
	if len(observed) != fixture.middleware {
		b.Fatalf("Run middleware preflight: callbacks=%d want=%d", len(observed), fixture.middleware)
	}
	for index, id := range observed {
		if id != index {
			b.Fatalf("Run middleware preflight: callback[%d]=%d want=%d", index, id, index)
		}
	}
}
