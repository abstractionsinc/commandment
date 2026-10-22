package shelltest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/abstractionsinc/commandment/cli"
)

func TestCompleteUsesRealBashEditorAndExactSuppliedApplication(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == windowsOS {
		t.Skip("Windows ConPTY integration is target-platform verified")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skipf("bash unavailable: %v", err)
	}
	app := compileRemoteApplication(t)
	target := bashTarget[string](bash, cli.Shell.Bash)
	got := Complete(t, app, target, Input{
		BeforeCursor: "alienctl re",
	})
	if !strings.Contains(got, "remote") {
		t.Fatalf("settled Bash screen does not contain completed remote command: %q", got)
	}
}

func TestCompleteOperationsAreIsolated(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == windowsOS {
		t.Skip("Windows ConPTY integration is target-platform verified")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skipf("bash unavailable: %v", err)
	}
	app := compileRemoteApplication(t)
	target := bashTarget[string](bash, cli.Shell.Bash)
	target.Interpret = func(observation Observation) (string, error) {
		return observation.Version(), nil
	}
	for index := range 2 {
		t.Run(fmt.Sprintf("operation-%d", index), func(t *testing.T) {
			t.Parallel()
			got := Complete(t, app, target, Input{
				BeforeCursor: "alienctl re",
			})
			if got == "" {
				t.Fatal("isolated operation lost verified runtime version")
			}
		})
	}
}

func TestApplicationBridgeRequiresCausalCompletionEventAndClosesListener(t *testing.T) {
	t.Parallel()
	app := compileRemoteApplication(t)
	bridge, err := startApplicationBridge(t.Context(), app, cli.ShellCompletionGeneration{
		ActivationEnvironment: "COMMANDMENT_COMPLETE",
		ActivationValue:       "adapter",
	})
	if err != nil {
		t.Fatal(err)
	}
	endpoint := bridge.endpoint
	if err := bridge.arm(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	err = bridge.waitWitness(ctx)
	cancel()
	if err == nil {
		t.Fatal("bridge accepted completion without an application transport event")
	}
	cleanupContext, cleanupCancel := context.WithTimeout(t.Context(), time.Second)
	if err := bridge.cleanup(cleanupContext); err != nil {
		cleanupCancel()
		t.Fatal(err)
	}
	cleanupCancel()
	dialer := &net.Dialer{Timeout: 50 * time.Millisecond}
	connection, dialErr := dialer.DialContext(t.Context(), "tcp", endpoint)
	if dialErr == nil {
		_ = connection.Close()
		t.Fatal("application bridge listener remained reachable after close")
	}
}

func compileRemoteApplication(t testing.TB) cli.Application {
	t.Helper()
	app, err := cli.Compile(cli.Definition[struct{}]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "remote",
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func bashTarget[R any](executable string, provider cli.ShellCompletionProvider) Target[R] {
	runtime := Runtime{
		Executable:  executable,
		VersionArgs: []string{"--version"},
		ParseVersion: func(output string) (string, error) {
			_, versionOutput, ok := strings.Cut(output, "version ")
			if !ok {
				return "", errors.New("Bash version marker missing")
			}
			fields := strings.Fields(versionOutput)
			if len(fields) == 0 {
				return "", errors.New("Bash version token missing")
			}
			return fields[0], nil
		},
	}
	target := Target[R]{
		Provider: provider,
		Runtime:  runtime,
		Prepare: func(setup Setup) error {
			integration := setup.IntegrationFile("completion.bash")
			rc := setup.File("bashrc", "source "+bashQuote(integration)+"\n")
			setup.Env("SHELL", "/bin/sh")
			setup.Env("TERM", "xterm-256color")
			setup.Args("--noprofile", "--rcfile", rc, "-i")
			return nil
		},
		Interpret: func(observation Observation) (R, error) {
			var zero R
			if observation.Version() == "" {
				return zero, errors.New("missing verified runtime version")
			}
			lines := observation.ScreenLines()
			if lines == nil {
				return zero, errors.New("settled screen unavailable")
			}
			joined := strings.Join(lines, "\n")
			result, ok := any(joined).(R)
			if !ok {
				return zero, fmt.Errorf("test target result type %T is unsupported", zero)
			}
			return result, nil
		},
	}
	return target
}

func bashQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
