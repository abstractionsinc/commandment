//go:build windows

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

const processSignalWindowsProcessSignalReady = "COMMANDMENT_PROCESS_SIGNAL_READY"

func processSignalWindowsProcessTestChild(marker string) bool {
	return slices.Contains(os.Args[1:], marker)
}

const windowsStatusControlCExit uint32 = 0xC000013A

func configureProcessSignalChild(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

func sendProcessInterrupt(command *exec.Cmd) error {
	dll, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return err
	}
	defer dll.Release()
	generate, err := dll.FindProc("GenerateConsoleCtrlEvent")
	if err != nil {
		return err
	}
	result, _, callErr := generate.Call(syscall.CTRL_BREAK_EVENT, uintptr(command.Process.Pid))
	if result == 0 {
		return callErr
	}
	return nil
}

func TestRunFromProcessReraiseTerminatesWindowsProcessWithControlCStatus(t *testing.T) {
	t.Parallel()
	const child = "commandment-process-reraise-windows-child"
	if processSignalWindowsProcessTestChild(child) {
		os.Args = []string{os.Args[0]}
		definition := cli.Definition[struct{}]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[struct{}]) {
				root.Signals.Policy(
					root.Signals.Match.Interrupt().During(cli.SignalPhaseExecution).Each(),
					root.Signals.Action.Reraise(),
				)
			},
			Run: func(ctx context.Context, _ *struct{}) error {
				fmt.Fprintln(os.Stdout, processSignalWindowsProcessSignalReady)
				<-ctx.Done()
				return ctx.Err()
			},
		}
		app, err := cli.Compile(definition)
		if err != nil {
			t.Fatal(err)
		}
		if err := app.RunFromProcess(t.Context()); err != nil {
			t.Fatalf("RunFromProcess() error = %v", err)
		}
		t.Fatal("RunFromProcess returned after terminal Reraise")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(
		ctx,
		os.Args[0],
		"-test.run=^TestRunFromProcessReraiseTerminatesWindowsProcessWithControlCStatus$",
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
	if !scanner.Scan() || scanner.Text() != processSignalWindowsProcessSignalReady {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf(
			"Windows process Reraise child readiness = %q, scan error = %v, stderr = %s",
			scanner.Text(),
			scanner.Err(),
			stderr.String(),
		)
	}
	if err := sendProcessInterrupt(command); err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf("send Ctrl+Break: %v", err)
	}
	err = command.Wait()
	if ctx.Err() != nil {
		t.Fatalf("Windows process Reraise child timed out: %v: %s", ctx.Err(), stderr.String())
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("RunFromProcess Reraise child error = %v, want terminal exit: %s", err, stderr.String())
	}
	if got := uint32(exitErr.ExitCode()); got != windowsStatusControlCExit {
		t.Fatalf(
			"RunFromProcess Reraise exit status = %#x, want %#x: %s",
			got,
			windowsStatusControlCExit,
			stderr.String(),
		)
	}
}
