package clibench_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clibench"
)

const (
	benchmarkCorrectnessProbeEnv    = "CLIBENCH_CORRECTNESS_PROBE"
	benchmarkCorrectnessProbeResult = "CLIBENCH_CORRECTNESS_PROBE_PASS"
	correctnessBenchmarkDuration    = "2x"
	benchmarkChildName              = "child"
)

func currentTestCoverDirArg() []string {
	coverDir := os.Getenv("GOCOVERDIR")
	if coverDir == "" {
		return nil
	}
	return []string{"-test.gocoverdir=" + coverDir}
}

func correctnessBenchmark(t *testing.T, benchmark func(*testing.B)) testing.BenchmarkResult {
	t.Helper()
	if os.Getenv(benchmarkCorrectnessProbeEnv) != t.Name() {
		t.Fatal("correctness benchmark must run in isolated test process")
	}
	return testing.Benchmark(benchmark)
}

func runBenchmarkCorrectnessCase(t *testing.T, test func()) {
	t.Helper()
	name := t.Name()
	if os.Getenv(benchmarkCorrectnessProbeEnv) == name {
		test()
		fmt.Fprintf(os.Stderr, "%s=%s\n", benchmarkCorrectnessProbeResult, name)
		return
	}

	parts := strings.Split(name, "/")
	for i, part := range parts {
		parts[i] = "^" + regexp.QuoteMeta(part) + "$"
	}
	commandCtx, commandCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer commandCancel()
	//nolint:gosec // os.Args[0] is the current test binary; the run pattern comes only from testing.T.Name.
	args := []string{
		"-test.run=" + strings.Join(parts, "/"),
		"-test.benchtime=" + correctnessBenchmarkDuration,
		"-test.count=1",
	}
	args = append(args, currentTestCoverDirArg()...)
	command := exec.CommandContext(commandCtx, os.Args[0], args...)
	command.Env = append(os.Environ(), benchmarkCorrectnessProbeEnv+"="+name)
	output, err := command.CombinedOutput()
	if commandCtx.Err() != nil {
		t.Fatalf("benchmark correctness probe timed out: %v:\n%s", commandCtx.Err(), output)
	}
	if err != nil {
		t.Fatalf("benchmark correctness probe failed: %v:\n%s", err, output)
	}
	wantResult := benchmarkCorrectnessProbeResult + "=" + name
	if !strings.Contains(string(output), wantResult) {
		t.Fatalf("benchmark correctness probe did not execute %s:\n%s", name, output)
	}
}

type benchmarkCallRecorder struct {
	mu    sync.Mutex
	calls int
	wrong bool
}

func (recorder *benchmarkCallRecorder) reset() {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	recorder.calls = 0
	recorder.wrong = false
}

func (recorder *benchmarkCallRecorder) record(matches bool) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	recorder.calls++
	if !matches {
		recorder.wrong = true
	}
}

func (recorder *benchmarkCallRecorder) result() (int, bool) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	return recorder.calls, recorder.wrong
}

func TestHelpersPreserveBenchmarkIterationAndContext(t *testing.T) {
	t.Parallel()
	t.Run("Run", func(t *testing.T) {
		t.Parallel()
		runBenchmarkCorrectnessCase(t, func() {
			recorder := &benchmarkCallRecorder{}
			var benchmarkCtx context.Context
			app := mustCompile(t, cli.Definition[struct{}]{
				Name: "tool",
				Run: func(ctx context.Context, _ *struct{}) error {
					recorder.record(ctx == benchmarkCtx)
					return nil
				},
			})
			if err := app.Run(context.Background(), cli.RunRequest{}); err != nil {
				t.Fatalf("Run preflight: %v", err)
			}
			result := correctnessBenchmark(t, func(b *testing.B) {
				benchmarkCtx = b.Context()
				recorder.reset()
				clibench.Run(b, app, cli.RunRequest{})
			})
			calls, wrongContext := recorder.result()
			if calls != result.N || result.N <= 1 {
				t.Fatalf("Run target calls = %d, benchmark N = %d", calls, result.N)
			}
			if wrongContext {
				t.Fatal("Run did not propagate the exact benchmark context")
			}
		})
	})

	t.Run("Inspect", func(t *testing.T) {
		t.Parallel()
		runBenchmarkCorrectnessCase(t, func() {
			recorder := &benchmarkCallRecorder{}
			var benchmarkCtx context.Context
			app := mustCompile(t, cli.Definition[struct{}]{
				Name: "tool",
				Run:  func(context.Context, *struct{}) error { return nil },
				Declare: func(root *cli.CommandBuilder[struct{}]) {
					root.Subcommands.CommandsFrom(func(ctx context.Context, batch *cli.CommandBatch[struct{}]) error {
						recorder.record(ctx == benchmarkCtx)
						batch.Command(cli.Command[struct{}]{Name: benchmarkChildName})
						return nil
					})
				},
			})
			request := cli.InspectionRequest{Args: []string{benchmarkChildName}}
			preflight, err := app.Inspect(context.Background(), request)
			if err != nil {
				t.Fatalf("Inspect preflight: %v", err)
			}
			if preflight.Status != cli.InspectionStatusReady {
				t.Fatalf("Inspect preflight = %#v, want ready observation", preflight)
			}
			result := correctnessBenchmark(t, func(b *testing.B) {
				benchmarkCtx = b.Context()
				recorder.reset()
				clibench.Inspect(b, app, request)
			})
			calls, wrongContext := recorder.result()
			if calls != result.N || result.N <= 1 {
				t.Fatalf("Inspect target-observable calls = %d, benchmark N = %d", calls, result.N)
			}
			if wrongContext {
				t.Fatal("Inspect did not propagate the exact benchmark context")
			}
		})
	})
	t.Run("Complete", func(t *testing.T) {
		t.Parallel()
		runBenchmarkCorrectnessCase(t, func() {
			recorder := &benchmarkCallRecorder{}
			var benchmarkCtx context.Context
			app := mustCompile(t, cli.Definition[struct{}]{
				Name: "tool",
				Declare: func(root *cli.CommandBuilder[struct{}]) {
					root.Subcommands.CommandsFrom(func(ctx context.Context, batch *cli.CommandBatch[struct{}]) error {
						recorder.record(ctx == benchmarkCtx)
						batch.Command(cli.Command[struct{}]{
							Name: "remote",
						})
						return nil
					})
				},
			})
			request := cli.CompletionRequest{
				Args: []string{""},
				Cursor: cli.CompletionCursor{
					Argument: 0,
				},
			}
			preflight, err := app.Complete(context.Background(), request)
			if err != nil {
				t.Fatalf("Complete preflight: %v", err)
			}
			found := false
			for _, item := range preflight.Items {
				if item.Value == "remote" {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("Complete preflight does not contain provider command: %#v", preflight.Items)
			}
			result := correctnessBenchmark(t, func(b *testing.B) {
				benchmarkCtx = b.Context()
				recorder.reset()
				clibench.Complete(b, app, request)
			})
			calls, wrongContext := recorder.result()
			if calls != result.N || result.N <= 1 {
				t.Fatalf("Complete provider calls = %d, benchmark N = %d", calls, result.N)
			}
			if wrongContext {
				t.Fatal("Complete did not propagate the exact benchmark context")
			}
		})
	})

	t.Run("Schema", func(t *testing.T) {
		t.Parallel()
		runBenchmarkCorrectnessCase(t, func() {
			app := mustCompile(t, cli.Definition[struct{}]{
				Name:            "tool",
				ShellCompletion: cli.ShellCompletion{Disabled: true},
			})
			preflight := app.Schema()
			if len(preflight.Commands) != 1 {
				t.Fatalf("Schema preflight commands = %d, want 1", len(preflight.Commands))
			}
			result := correctnessBenchmark(t, func(b *testing.B) {
				clibench.Schema(b, app)
			})
			if result.N <= 1 {
				t.Fatalf("Schema benchmark N = %d, want more than one calibrated iteration", result.N)
			}
		})
	})

	t.Run("Reference", func(t *testing.T) {
		t.Parallel()
		runBenchmarkCorrectnessCase(t, func() {
			app := mustCompile(t, cli.Definition[struct{}]{
				Name:            "tool",
				ShellCompletion: cli.ShellCompletion{Disabled: true},
			})
			preflight, err := app.Reference(cli.ReferenceOptions{})
			if err != nil {
				t.Fatalf("Reference preflight: %v", err)
			}
			if len(preflight.Commands) != 1 {
				t.Fatalf("Reference preflight commands = %d, want 1", len(preflight.Commands))
			}
			result := correctnessBenchmark(t, func(b *testing.B) {
				clibench.Reference(b, app, cli.ReferenceOptions{})
			})
			if result.N <= 1 {
				t.Fatalf("Reference benchmark N = %d, want more than one calibrated iteration", result.N)
			}
		})
	})

	t.Run("Compile", func(t *testing.T) {
		t.Parallel()
		runBenchmarkCorrectnessCase(t, func() {
			var declarations atomic.Int64
			definition := cli.Definition[struct{}]{
				Name:            "tool",
				ShellCompletion: cli.ShellCompletion{Disabled: true},
				Declare: func(*cli.CommandBuilder[struct{}]) {
					declarations.Add(1)
				},
			}
			app, err := cli.Compile(definition)
			if err != nil {
				t.Fatalf("Compile preflight: %v", err)
			}
			if schema := app.Schema(); len(schema.Commands) != 1 {
				t.Fatalf("Compile preflight schema commands = %d, want 1", len(schema.Commands))
			}
			result := correctnessBenchmark(t, func(b *testing.B) {
				declarations.Store(0)
				clibench.Compile(b, definition)
			})
			if got := declarations.Load(); got != int64(result.N) || result.N <= 1 {
				t.Fatalf("Compile declarations = %d, benchmark N = %d", got, result.N)
			}
		})
	})
}

func TestHelperFatalAttributionAndNoRetry(t *testing.T) {
	t.Parallel()
	commandCtx, commandCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer commandCancel()
	//nolint:gosec // os.Args[0] is the current test binary; all arguments are test-owned literals.
	args := []string{
		"-test.run=^$",
		"-test.bench=^BenchmarkFatalAttributionProbe$",
		"-test.benchtime=1x",
		"-test.count=1",
	}
	args = append(args, currentTestCoverDirArg()...)
	command := exec.CommandContext(commandCtx, os.Args[0], args...)
	command.Env = append(os.Environ(), "CLIBENCH_FATAL_PROBE=1")
	output, err := command.CombinedOutput()
	if commandCtx.Err() != nil {
		t.Fatalf("fatal probe timed out: %v:\n%s", commandCtx.Err(), output)
	}
	if err == nil {
		t.Fatalf("fatal probe unexpectedly succeeded:\n%s", output)
	}
	text := string(output)
	for _, name := range []string{"Run", "Inspect", "Complete", "Reference", "Compile"} {
		if !strings.Contains(text, "CLIBENCH_SETUP_"+name) {
			t.Fatalf("fatal probe did not reach %s helper setup:\n%s", name, text)
		}
		if name != "Reference" && !strings.Contains(text, "CLIBENCH_CALLS_"+name+"=1") {
			t.Fatalf("fatal probe did not stop %s after first target-observable call:\n%s", name, text)
		}
	}
	if got := strings.Count(text, "operations_test.go:"); got < 5 {
		t.Fatalf("fatal attribution reached benchmark caller %d times, want at least 5:\n%s", got, text)
	}
	normalized := strings.ReplaceAll(text, `\`, "/")
	if strings.Contains(normalized, "operations.go:") {
		t.Fatalf("fatal attribution leaked helper implementation frame:\n%s", text)
	}
}

func BenchmarkFatalAttributionProbe(b *testing.B) {
	if os.Getenv("CLIBENCH_FATAL_PROBE") != "1" {
		b.Skip("subprocess-only failure probe")
	}

	for _, name := range []string{"Run", "Inspect", "Complete", "Reference", "Compile"} {
		b.Run(name, func(b *testing.B) {
			var calls atomic.Int64
			defer func() {
				fmt.Fprintf(os.Stderr, "CLIBENCH_CALLS_%s=%d\n", name, calls.Load())
			}()

			failure := errors.New("clibench probe failure")
			switch name {
			case "Run":
				app := mustCompile(b, cli.Definition[struct{}]{
					Name: "tool",
					Run: func(context.Context, *struct{}) error {
						calls.Add(1)
						return failure
					},
				})
				fmt.Fprintln(os.Stderr, "CLIBENCH_SETUP_Run")
				clibench.Run(b, app, cli.RunRequest{})
			case "Inspect":
				app := mustCompile(b, cli.Definition[struct{}]{
					Name: "tool",
					Run:  func(context.Context, *struct{}) error { return nil },
					Declare: func(root *cli.CommandBuilder[struct{}]) {
						root.Subcommands.CommandsFrom(func(context.Context, *cli.CommandBatch[struct{}]) error {
							calls.Add(1)
							return failure
						})
					},
				})
				fmt.Fprintln(os.Stderr, "CLIBENCH_SETUP_Inspect")
				clibench.Inspect(b, app, cli.InspectionRequest{Args: []string{benchmarkChildName}})
			case "Complete":
				app := mustCompile(b, cli.Definition[struct{}]{
					Name: "tool",
					Declare: func(root *cli.CommandBuilder[struct{}]) {
						root.Subcommands.CommandsFrom(func(context.Context, *cli.CommandBatch[struct{}]) error {
							calls.Add(1)
							return failure
						})
					},
				})
				fmt.Fprintln(os.Stderr, "CLIBENCH_SETUP_Complete")
				clibench.Complete(b, app, cli.CompletionRequest{
					Args: []string{""},
					Cursor: cli.CompletionCursor{
						Argument: 0,
					},
				})
			case "Reference":
				app := mustCompile(b, cli.Definition[struct{}]{
					Name: "tool",
				})
				fmt.Fprintln(os.Stderr, "CLIBENCH_SETUP_Reference")
				clibench.Reference(b, app, cli.ReferenceOptions{
					Visibility: cli.ReferenceVisibility(255),
				})
			case "Compile":
				definition := cli.Definition[struct{}]{
					Name: "tool",
					Declare: func(root *cli.CommandBuilder[struct{}]) {
						calls.Add(1)
						root.Subcommands.Command(cli.Command[struct{}]{
							Name: "",
						})
					},
				}
				fmt.Fprintln(os.Stderr, "CLIBENCH_SETUP_Compile")
				clibench.Compile(b, definition)
			}
		})
	}
}

func mustCompile(t testing.TB, definition cli.Definition[struct{}]) cli.Application {
	t.Helper()
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return app
}
