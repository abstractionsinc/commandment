//go:build !windows

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
	"syscall"
	"testing"
	"time"

	"github.com/abstractionsinc/commandment/cli"
)

var _ = errors.Is

const processSignalUnixProcessSignalReady = "COMMANDMENT_PROCESS_SIGNAL_READY"

func configureProcessSignalChild(*exec.Cmd) {}

func sendProcessInterrupt(command *exec.Cmd) error {
	if err := command.Process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("send child interrupt: %w", err)
	}
	return nil
}

func processSignalUnixProcessTestChild(marker string) bool {
	return slices.Contains(os.Args[1:], marker)
}

func TestRunFromProcessReraiseRestoresUnixDefaultAndTerminatesBySignal(t *testing.T) {
	t.Parallel()
	const child = "commandment-process-reraise-unix-child"
	if processSignalUnixProcessTestChild(child) {
		os.Args = []string{os.Args[0]}
		definition := cli.Definition[struct{}]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[struct{}]) {
				root.Signals.Policy(
					root.Signals.Match.Terminate().During(cli.SignalPhaseExecution).Each(),
					root.Signals.Action.Reraise(),
				)
			},
			Run: func(ctx context.Context, _ *struct{}) error {
				if _, err := fmt.Fprintln(os.Stdout, processSignalUnixProcessSignalReady); err != nil {
					return fmt.Errorf("write child readiness: %w", err)
				}
				<-ctx.Done()
				return ctx.Err()
			},
		}
		app, err := cli.Compile(definition)
		if err != nil {
			t.Fatal(err)
		}
		if err := app.RunFromProcess(t.Context()); err != nil {
			t.Fatalf("RunFromProcess() returned after terminal Reraise: %v", err)
		}
		t.Fatal("RunFromProcess returned after terminal Reraise")
	}

	const attempts = 8
	for attempt := range attempts {
		t.Run(fmt.Sprintf("attempt-%02d", attempt+1), func(t *testing.T) {
			t.Parallel()
			assertRunFromProcessReraiseTerminatesBySignal(t, child)
		})
	}
}

func assertRunFromProcessReraiseTerminatesBySignal(t *testing.T, child string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(
		ctx,
		os.Args[0],
		"-test.run=^TestRunFromProcessReraiseRestoresUnixDefaultAndTerminatesBySignal$",
		child,
	)
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
	if !scanner.Scan() || scanner.Text() != processSignalUnixProcessSignalReady {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf(
			"process Reraise child readiness = %q, scan error = %v, stderr = %s",
			scanner.Text(),
			scanner.Err(),
			stderr.String(),
		)
	}
	if err := command.Process.Signal(syscall.SIGTERM); err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf("send SIGTERM: %v", err)
	}
	err = command.Wait()
	if ctx.Err() != nil {
		t.Fatalf("process Reraise child timed out: %v: %s", ctx.Err(), stderr.String())
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("RunFromProcess Reraise child error = %v, want SIGTERM termination: %s", err, stderr.String())
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() || status.Signal() != syscall.SIGTERM {
		t.Fatalf(
			"RunFromProcess Reraise wait status = %v, want SIGTERM termination: %s",
			exitErr.Sys(),
			stderr.String(),
		)
	}
}
