package conformance

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"testing"
	"time"

	"github.com/abstractionsinc/commandment/cli"
)

var _ = errors.Is

const processSignalProcessSignalReady = "COMMANDMENT_PROCESS_SIGNAL_READY"

func TestRunFromProcessAbortTerminatesWithSelectedCode(t *testing.T) {
	t.Parallel()
	const child = "commandment-process-abort-child"
	if processSignalProcessTestChild(child) {
		os.Args = []string{os.Args[0]}
		app, err := cli.Compile(cli.Definition[struct{}]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[struct{}]) {
				root.Signals.Policy(
					root.Signals.Match.Interrupt().During(cli.SignalPhaseExecution).Each(),
					root.Signals.Action.Abort(23),
				)
			},
			Run: func(ctx context.Context, _ *struct{}) error {
				if _, err := fmt.Fprintln(os.Stdout, processSignalProcessSignalReady); err != nil {
					return fmt.Errorf("write child readiness: %w", err)
				}
				<-ctx.Done()
				return ctx.Err()
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := app.RunFromProcess(t.Context()); err != nil {
			t.Fatalf("RunFromProcess() error = %v", err)
		}
		t.Fatal("RunFromProcess returned after terminal Abort")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(
		ctx,
		os.Args[0],
		"-test.run=^TestRunFromProcessAbortTerminatesWithSelectedCode$",
		child,
	)
	configureProcessSignalChild(command)
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(stdout)
	if !scanner.Scan() || scanner.Text() != processSignalProcessSignalReady {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf(
			"process Abort child readiness = %q, scan error = %v, stderr = %s",
			scanner.Text(),
			scanner.Err(),
			stderr.String(),
		)
	}
	if err := sendProcessInterrupt(command); err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf("send process interrupt: %v", err)
	}
	err = command.Wait()
	if ctx.Err() != nil {
		t.Fatalf("process Abort child timed out: %v: %s", ctx.Err(), stderr.String())
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("RunFromProcess Abort child error = %v, want exit status 23: %s", err, stderr.String())
	}
	if code := exitErr.ExitCode(); code != 23 {
		t.Fatalf("RunFromProcess Abort exit code = %d, want 23: %s", code, stderr.String())
	}
}

func processSignalProcessTestChild(marker string) bool {
	return slices.Contains(os.Args[1:], marker)
}
