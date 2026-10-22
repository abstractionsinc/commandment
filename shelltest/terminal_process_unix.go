//go:build !windows

package shelltest

import (
	"context"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

type terminalProcess struct {
	command       *exec.Cmd
	pty           *os.File
	doneCh        chan struct{}
	waitErr       error
	terminateOnce sync.Once
}

func startTerminalProcess(parent context.Context, executable string, args []string, workspace string, environment []string) (*terminalProcess, error) {
	//nolint:gosec // executable is the exact verified runtime selected by the caller.
	command := exec.CommandContext(parent, executable, args...)
	command.Dir = workspace
	command.Env = environment
	command.Cancel = func() error {
		if command.Process == nil {
			return nil
		}
		return unix.Kill(-command.Process.Pid, unix.SIGKILL)
	}
	terminal, err := pty.StartWithSize(command, &pty.Winsize{
		Rows: terminalRows,
		Cols: terminalColumns,
	})
	if err != nil {
		failure := failureError{
			subject: "start PTY shell process",
			cause:   err,
		}
		return nil, failure
	}
	process := &terminalProcess{
		command: command,
		pty:     terminal,
		doneCh:  make(chan struct{}),
	}
	go func() {
		process.waitErr = command.Wait()
		close(process.doneCh)
	}()
	return process, nil
}

func (process *terminalProcess) Read(data []byte) (int, error) {
	count, err := process.pty.Read(data)
	if err != nil {
		failure := failureError{
			subject: "read PTY shell output",
			cause:   err,
		}
		return count, failure
	}
	return count, nil
}

func (process *terminalProcess) Write(data []byte) (int, error) {
	count, err := process.pty.Write(data)
	if err != nil {
		failure := failureError{
			subject: "write PTY shell input",
			cause:   err,
		}
		return count, failure
	}
	return count, nil
}

func (process *terminalProcess) done() <-chan struct{} { return process.doneCh }
func (process *terminalProcess) exited() bool {
	select {
	case <-process.doneCh:
		return true
	default:
		return false
	}
}

func (process *terminalProcess) waitError() error {
	<-process.doneCh
	return process.waitErr
}

func (process *terminalProcess) terminate() {
	process.terminateOnce.Do(func() {
		if process.command.Process != nil {
			_ = unix.Kill(-process.command.Process.Pid, unix.SIGKILL)
		}
	})
}

func (process *terminalProcess) closeTerminal() {
	_ = process.pty.Close()
}

var _ io.ReadWriter = (*terminalProcess)(nil)
