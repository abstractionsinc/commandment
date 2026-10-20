package main

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/abstractionsinc/commandment/cli"
)

func TestInterruptReachesDeclaredEndpoint(t *testing.T) {
	const child = "commandment-example-interrupt-child"
	if slices.Contains(os.Args[1:], child) {
		os.Args = []string{os.Args[0], "listen"}
		app, err := cli.Compile(definition)
		if err != nil {
			t.Fatalf("Compile() error = %v", err)
		}
		if err := app.RunFromProcess(t.Context()); err != nil {
			t.Fatalf("RunFromProcess() error = %v", err)
		}
		return
	}

	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("real interrupt example requires a Unix-like host")
	}

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("Executable() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestInterruptReachesDeclaredEndpoint$", child)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe() error = %v", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	reader := bufio.NewReader(stdout)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read readiness line error = %v", err)
	}
	if line != "listening for signals\n" {
		t.Fatalf("readiness line = %q", line)
	}
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("Signal() error = %v", err)
	}
	line, err = reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read signal line error = %v", err)
	}
	if line != "signal received\n" {
		t.Fatalf("signal line = %q", line)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("Wait() error = %v; stderr = %q", err, stderr.String())
	}
	if ctx.Err() != nil {
		t.Fatalf("interrupt child timed out: %v", ctx.Err())
	}
}
