package clibench

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

// Run benchmarks app.Run for request once per B.Loop iteration. It reuses one
// benchmark-lifetime context from b.Context, and any operation error fails the
// benchmark.
func Run(b *testing.B, app cli.Application, request cli.RunRequest) {
	b.Helper()
	ctx := b.Context()
	for b.Loop() {
		if err := app.Run(ctx, request); err != nil {
			b.Fatal(err)
		}
	}
}

// Inspect benchmarks app.Inspect for request once per B.Loop iteration. It
// reuses one benchmark-lifetime context from b.Context, discards each
// Inspection, and fails the benchmark on any operation error.
func Inspect(b *testing.B, app cli.Application, request cli.InspectionRequest) {
	b.Helper()
	ctx := b.Context()
	for b.Loop() {
		if _, err := app.Inspect(ctx, request); err != nil {
			b.Fatal(err)
		}
	}
}

// Complete benchmarks app.Complete for request once per B.Loop iteration. It
// reuses one benchmark-lifetime context from b.Context, discards each
// Completion, and fails the benchmark on any operation error.
func Complete(b *testing.B, app cli.Application, request cli.CompletionRequest) {
	b.Helper()
	ctx := b.Context()
	for b.Loop() {
		if _, err := app.Complete(ctx, request); err != nil {
			b.Fatal(err)
		}
	}
}

// Schema benchmarks app.Schema once per B.Loop iteration and discards each
// Schema.
func Schema(b *testing.B, app cli.Application) {
	b.Helper()
	for b.Loop() {
		_ = app.Schema()
	}
}

// Reference benchmarks app.Reference for options once per B.Loop iteration. It
// discards each Reference and fails the benchmark on any operation error.
func Reference(b *testing.B, app cli.Application, options cli.ReferenceOptions) {
	b.Helper()
	for b.Loop() {
		if _, err := app.Reference(options); err != nil {
			b.Fatal(err)
		}
	}
}

// Compile benchmarks cli.Compile for definition once per B.Loop iteration. It
// discards each compiled Application and fails the benchmark on any compile
// error. The definition must remain an equivalent reusable workload across
// iterations.
func Compile[C any](b *testing.B, definition cli.Definition[C]) {
	b.Helper()
	for b.Loop() {
		if _, err := cli.Compile(definition); err != nil {
			b.Fatal(err)
		}
	}
}
