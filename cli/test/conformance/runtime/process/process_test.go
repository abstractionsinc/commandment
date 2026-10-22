package conformance

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

const (
	processTestChildEnv = "COMMANDMENT_CONFORMANCE_PROCESS_TEST_CHILD"
	processTestCaseEnv  = "COMMANDMENT_CONFORMANCE_PROCESS_TEST_CASE"
)

func processMustCompilePublic[C any](t testing.TB, definition cli.Definition[C]) cli.Application {
	t.Helper()
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return app
}

func runProcessTestChild(t *testing.T, testName, directory string, environment ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^"+regexp.QuoteMeta(testName)+"$")
	if directory != "" {
		command.Dir = directory
	}
	command.Env = append(os.Environ(), processTestChildEnv+"="+testName)
	command.Env = append(command.Env, environment...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("isolated process %s: %v\n%s", testName, err, output)
	}
}

func isProcessTestChild(testName string) bool {
	return os.Getenv(processTestChildEnv) == testName
}

type processAdapterConfig struct{}

type processAdapterFlagConfig struct{ Flag string }

type processAdapterWorkingDirectoryConfig struct{ WorkingDirectory string }

func TestProcessResponseFileAccessPreservesProcessContract(t *testing.T) {
	const testName = "TestProcessResponseFileAccessPreservesProcessContract"
	if isProcessTestChild(testName) {
		type config = processAdapterFlagConfig
		var got config
		definition := cli.Definition[config]{
			Name: "commandment",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Dialect.ResponseFiles(true)
				root.Flags.String(&root.Config.Flag).Long("flag")
			},
			Run: func(_ context.Context, cfg *config) error { got = *cfg; return nil },
		}
		app, err := cli.Compile(definition)
		if err != nil {
			t.Fatal(err)
		}
		os.Args = []string{"commandment", "@nested/args.txt"}
		if err := app.RunFromProcess(t.Context()); err != nil {
			t.Fatalf("RunFromProcess: %v", err)
		}
		if got.Flag != "value" {
			t.Fatalf("response-file value = %q, want value", got.Flag)
		}
		return
	}

	t.Parallel()
	dir := t.TempDir()
	childDir := filepath.Join(dir, "nested")
	if err := os.Mkdir(childDir, 0o750); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(childDir, "args.txt")
	if err := os.WriteFile(path, []byte("--flag value"), 0o600); err != nil {
		t.Fatal(err)
	}
	runProcessTestChild(t, testName, dir)
}

func TestRunFromProcessReturnsOrdinaryExecutionFailure(t *testing.T) {
	const testName = "TestRunFromProcessReturnsOrdinaryExecutionFailure"
	if isProcessTestChild(testName) {
		type config = processAdapterConfig
		sentinel := errors.New("ordinary execution failure")
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "commandment",
			Run:  func(context.Context, *config) error { return sentinel },
		})
		os.Args = []string{"commandment"}
		err := app.RunFromProcess(t.Context())
		if !errors.Is(err, sentinel) {
			t.Fatalf("RunFromProcess() error = %v, want ordinary execution cause", err)
		}
		return
	}

	t.Parallel()
	runProcessTestChild(t, testName, "")
}

func TestRunFromProcessSuppliesCurrentWorkingDirectory(t *testing.T) {
	const testName = "TestRunFromProcessSuppliesCurrentWorkingDirectory"
	if isProcessTestChild(testName) {
		type config = processAdapterWorkingDirectoryConfig
		var got string
		definition := cli.Definition[config]{
			Name: "commandment",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Access.WorkingDirectory(&root.Config.WorkingDirectory)
			},
			Run: func(_ context.Context, cfg *config) error {
				got = cfg.WorkingDirectory
				return nil
			},
		}
		app := clitest.Compile(t, definition)
		os.Args = []string{"commandment"}
		if err := app.RunFromProcess(t.Context()); err != nil {
			t.Fatalf("RunFromProcess() error = %v", err)
		}
		want, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("working directory = %q, want %q", got, want)
		}
		return
	}

	t.Parallel()
	runProcessTestChild(t, testName, t.TempDir())
}

func TestRunFromProcessSuppliesSignalIntegrationToSharedRunPipeline(t *testing.T) {
	const testName = "TestRunFromProcessSuppliesSignalIntegrationToSharedRunPipeline"
	if isProcessTestChild(testName) {
		type config = processAdapterConfig
		handlerCalled := false
		definition := cli.Definition[config]{
			Name: "commandment",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Signals.Policy(
					root.Signals.Match.Interrupt().During(cli.SignalPhaseExecution).Each(),
					root.Signals.Action.Ignore(),
				)
			},
			Run: func(context.Context, *config) error { handlerCalled = true; return nil },
		}
		app, err := cli.Compile(definition)
		if err != nil {
			t.Fatalf("Compile() error = %v", err)
		}
		clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
		os.Args = []string{"commandment"}
		if err := runSignalApplicationTest(t, func() error { return app.RunFromProcess(t.Context()) }); err != nil {
			t.Fatalf("RunFromProcess() error = %v", err)
		}
		if !handlerCalled {
			t.Fatal("RunFromProcess() did not reach ordinary shared Run execution")
		}
		return
	}

	t.Parallel()
	runProcessTestChild(t, testName, "")
}

func TestRunApplicationFromProcessPreservesProcessContract(t *testing.T) {
	const testName = "TestRunApplicationFromProcessPreservesProcessContract"
	if isProcessTestChild(testName) {
		type config = processAdapterConfig
		called := 0
		app, err := cli.Compile(cli.Definition[config]{
			Name: "commandment",
			Run: func(context.Context, *config) error {
				called++
				return nil
			},
		})
		if err != nil {
			t.Fatalf("Compile() error = %v", err)
		}
		os.Args = []string{"commandment"}
		if err := app.RunFromProcess(t.Context()); err != nil {
			t.Fatalf("RunFromProcess() error = %v", err)
		}
		if called != 1 {
			t.Fatalf("handler calls = %d, want 1", called)
		}
		return
	}

	t.Parallel()
	runProcessTestChild(t, testName, "")
}

func TestProcessSignalSourceDescriptionSupportsSharedRunPlanner(t *testing.T) {
	const testName = "TestProcessSignalSourceDescriptionSupportsSharedRunPlanner"
	if isProcessTestChild(testName) {
		name := os.Getenv(processTestCaseEnv)
		var declare func(*cli.CommandBuilder[struct{}])
		switch name {
		case "interrupt":
			declare = func(root *cli.CommandBuilder[struct{}]) {
				root.Signals.Policy(
					root.Signals.Match.Interrupt().During(cli.SignalPhaseExecution).Each(),
					root.Signals.Action.Ignore(),
				)
			}
		case "terminate":
			declare = func(root *cli.CommandBuilder[struct{}]) {
				root.Signals.Policy(
					root.Signals.Match.Terminate().During(cli.SignalPhaseExecution).Each(),
					root.Signals.Action.Ignore(),
				)
			}
		case "shutdown":
			declare = func(root *cli.CommandBuilder[struct{}]) {
				root.Signals.Policy(
					root.Signals.Match.Shutdown().During(cli.SignalPhaseExecution).Each(),
					root.Signals.Action.Ignore(),
				)
			}
		default:
			t.Fatalf("unknown isolated process case %q", name)
		}
		definition := cli.Definition[struct{}]{
			Name:    "tool",
			Run:     func(context.Context, *struct{}) error { return nil },
			Declare: declare,
		}
		app := processMustCompilePublic(t, definition)
		os.Args = []string{"tool"}
		semanticErr := runSignalApplicationTest(t, func() error { return app.RunFromProcess(t.Context()) })
		clitest.Diagnostics(semanticErr).Require(t, clitest.DiagnosticsExactly())
		if semanticErr != nil {
			t.Fatalf("RunFromProcess() error = %v", semanticErr)
		}
		return
	}

	t.Parallel()
	cases := []string{"interrupt"}
	if runtime.GOOS != "windows" {
		cases = append(cases, "terminate", "shutdown")
	}
	for _, name := range cases {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			runProcessTestChild(t, testName, "", processTestCaseEnv+"="+name)
		})
	}
}

func runSignalApplicationTest(t *testing.T, run func() error) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- run() }()
	select {
	case err := <-done:
		return err
	case <-time.After(3 * time.Second):
		t.Fatal("signal application timed out")
		return nil
	}
}
