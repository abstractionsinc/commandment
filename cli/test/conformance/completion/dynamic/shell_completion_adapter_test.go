package conformance

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"regexp"
	"testing"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

func processAdapterComplete(t *testing.T, app cli.Application, request cli.CompletionRequest) (cli.Completion, error) {
	t.Helper()
	return clitest.Complete(t.Context(), app, request).Completion()
}

const (
	processAdapterRemoteName   = "remote"
	processAdapterChildEnv     = "COMMANDMENT_CONFORMANCE_PROCESS_ADAPTER_CHILD"
	processAdapterChildCaseEnv = "COMMANDMENT_CONFORMANCE_PROCESS_ADAPTER_CASE"
)

func runProcessAdapterChild(t *testing.T, testName string, environment ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^"+regexp.QuoteMeta(testName)+"$")
	command.Env = append(os.Environ(), processAdapterChildEnv+"="+testName)
	command.Env = append(command.Env, environment...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("isolated process adapter %s: %v\n%s", testName, err, output)
	}
}

func isProcessAdapterChild(testName string) bool {
	return os.Getenv(processAdapterChildEnv) == testName
}

func TestRunFromProcessRejectsUnreleasedShellCompletionActivationValueTwo(t *testing.T) {
	const testName = "TestRunFromProcessRejectsUnreleasedShellCompletionActivationValueTwo"
	if isProcessAdapterChild(testName) {
		app, err := cli.Compile(cli.Definition[struct{}]{Name: "tool"})
		if err != nil {
			t.Fatal(err)
		}

		var generation cli.ShellCompletionGeneration
		provider := cli.Shell.Custom("probe", func(value cli.ShellCompletionGeneration) (string, error) {
			generation = value
			return "probe", nil
		})
		if _, err := app.ShellCompletionScript(provider); err != nil {
			t.Fatalf("capture shell-adapter activation: %v", err)
		}
		if generation.ActivationEnvironment == "" {
			t.Fatal("shell-adapter activation environment is empty")
		}

		os.Args = []string{"tool"}
		if err := os.Setenv(generation.ActivationEnvironment, "2"); err != nil {
			t.Fatal(err)
		}
		if err := app.RunFromProcess(t.Context()); err == nil {
			t.Fatal("RunFromProcess accepted unreleased shell-completion activation value 2")
		}
		return
	}

	t.Parallel()
	runProcessAdapterChild(t, testName)
}

func TestRunFromProcessRejectsInvalidShellCompletionProcessFraming(t *testing.T) {
	if runAdapterFuzzChildMode(t) {
		return
	}

	const testName = "TestRunFromProcessRejectsInvalidShellCompletionProcessFraming"
	if isProcessAdapterChild(testName) {
		app, err := cli.Compile(cli.Definition[struct{}]{Name: "tool"})
		if err != nil {
			t.Fatal(err)
		}

		var generation cli.ShellCompletionGeneration
		provider := cli.Shell.Custom("probe", func(value cli.ShellCompletionGeneration) (string, error) {
			generation = value
			return "probe", nil
		})
		if _, err := app.ShellCompletionScript(provider); err != nil {
			t.Fatalf("capture shell-adapter activation: %v", err)
		}

		var args []string
		var input string
		switch name := os.Getenv(processAdapterChildCaseEnv); name {
		case "arguments":
			args = []string{"unexpected"}
			input = "json\n"
		case "carriage-return":
			input = "records\r\n"
		case "case":
			input = "RECORDS\n"
		case "unterminated":
			input = "records"
		default:
			t.Fatalf("unknown isolated adapter case %q", name)
		}

		stdin, err := os.CreateTemp(t.TempDir(), "adapter-request-")
		if err != nil {
			t.Fatal(err)
		}
		defer stdin.Close()
		if _, err := stdin.WriteString(input); err != nil {
			t.Fatal(err)
		}
		if _, err := stdin.Seek(0, io.SeekStart); err != nil {
			t.Fatal(err)
		}
		stdout, err := os.CreateTemp(t.TempDir(), "adapter-response-")
		if err != nil {
			t.Fatal(err)
		}
		defer stdout.Close()

		previousStdin, previousStdout := os.Stdin, os.Stdout
		os.Stdin = stdin
		os.Stdout = stdout
		defer func() {
			os.Stdin = previousStdin
			os.Stdout = previousStdout
		}()
		os.Args = append([]string{"tool"}, args...)
		if err := os.Setenv(generation.ActivationEnvironment, generation.ActivationValue); err != nil {
			t.Fatal(err)
		}
		if err := app.RunFromProcess(t.Context()); err == nil {
			t.Fatal("RunFromProcess accepted invalid shell-completion process framing")
		}
		return
	}

	t.Parallel()
	for _, name := range []string{"arguments", "carriage-return", "case", "unterminated"} {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			runProcessAdapterChild(t, testName, processAdapterChildCaseEnv+"="+name)
		})
	}
}

func TestRunFromProcessShellAdapterUsesSameNonProcessSignalCatalogAsDirectComplete(t *testing.T) {
	const testName = "TestRunFromProcessShellAdapterUsesSameNonProcessSignalCatalogAsDirectComplete"
	if isProcessAdapterChild(testName) {
		type config struct{}
		const customSignal = "custom"
		source := cli.SignalSource[string]{
			Signals: []cli.SignalDescriptor[string]{{
				Signal:   customSignal,
				Fidelity: cli.SignalSourceDeliveryFidelityExact,
			}},
			Open: func(context.Context) (cli.SignalStream[string], error) {
				panic("completion must not open application signal runtime")
			},
		}
		app, err := cli.Compile(cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				signal := root.Signals.Source(source).Signal(customSignal)
				root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
					batch.Command(cli.Command[config]{
						Name: processAdapterRemoteName,
						Declare: func(child *cli.CommandBuilder[config]) {
							child.Signals.Policy(
								child.Signals.Match.On(signal).During(cli.SignalPhaseExecution).Each(),
								child.Signals.Action.Ignore(),
							)
						},
					})
					return nil
				})
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		direct, err := processAdapterComplete(t, app, cli.CompletionRequest{
			Args: []string{"re"},
			Cursor: cli.CompletionCursor{
				Argument: 0,
				Byte:     2,
			},
			Visibility: cli.CompletionVisibilityPublic,
			Client: cli.CompletionClient{
				Match: cli.CompletionMatchOptions{
					Mode: cli.CompletionMatchModePrefix,
					Case: cli.CompletionCaseSmart,
				},
				Ranking: cli.CompletionRankingModeRelevance,
			},
		})
		if err != nil {
			t.Fatalf("direct Complete: %v", err)
		}
		foundDirect := false
		for _, item := range direct.Items {
			foundDirect = foundDirect || item.Value == processAdapterRemoteName
		}
		if !foundDirect {
			t.Fatalf("direct Complete did not materialize signal-dependent provider command: %#v", direct.Items)
		}

		var generation cli.ShellCompletionGeneration
		probe := cli.Shell.Custom("probe", func(value cli.ShellCompletionGeneration) (string, error) {
			generation = value
			return "probe", nil
		})
		if _, err := app.ShellCompletionScript(probe); err != nil {
			t.Fatalf("capture shell-adapter activation: %v", err)
		}
		if generation.ActivationEnvironment == "" || generation.ActivationValue == "" {
			t.Fatalf("invalid shell-adapter activation: %#v", generation)
		}
		const request = "json\n" +
			`{"response_encoding":"json","shell":{"family":"probe","version":"1"},` +
			`"input":{"words":[{"value":"re"}],"cursor":{"word":0,"byte":2}},` +
			`"matching":"prefix"}`

		stdin, err := os.CreateTemp(t.TempDir(), "adapter-request-")
		if err != nil {
			t.Fatal(err)
		}
		defer stdin.Close()
		if _, err := stdin.WriteString(request); err != nil {
			t.Fatal(err)
		}
		if _, err := stdin.Seek(0, io.SeekStart); err != nil {
			t.Fatal(err)
		}
		stdout, err := os.CreateTemp(t.TempDir(), "adapter-response-")
		if err != nil {
			t.Fatal(err)
		}
		defer stdout.Close()
		previousStdin, previousStdout := os.Stdin, os.Stdout
		os.Stdin = stdin
		os.Stdout = stdout
		defer func() {
			os.Stdin = previousStdin
			os.Stdout = previousStdout
		}()
		os.Args = []string{"tool"}
		if err := os.Setenv(generation.ActivationEnvironment, generation.ActivationValue); err != nil {
			t.Fatal(err)
		}

		if err := app.RunFromProcess(t.Context()); err != nil {
			t.Fatalf("RunFromProcess shell adapter: %v", err)
		}
		if _, err := stdout.Seek(0, io.SeekStart); err != nil {
			t.Fatal(err)
		}
		encoded, err := io.ReadAll(stdout)
		if err != nil {
			t.Fatal(err)
		}
		var response struct {
			Failed bool `json:"failed"`
			Items  []struct {
				Value string `json:"value"`
			} `json:"items"`
		}
		if err := json.Unmarshal(encoded, &response); err != nil {
			t.Fatalf("decode shell-adapter response: %v; wire=%q", err, encoded)
		}
		if response.Failed {
			t.Fatalf("shell-adapter process completion reported failure: %s", encoded)
		}
		for _, item := range response.Items {
			if item.Value == processAdapterRemoteName {
				return
			}
		}
		t.Fatalf("shell-adapter process completion diverged from direct Complete signal catalog: %s", encoded)
		return
	}

	t.Parallel()
	runProcessAdapterChild(t, testName)
}
