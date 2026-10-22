package main

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/shelltest"
)

func TestBashCompletesScanThroughARealShell(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("real Bash completion example requires a Unix-like host")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skipf("bash unavailable: %v", err)
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	input := shelltest.Input{
		BeforeCursor: "alienctl sc",
	}
	screen := shelltest.Complete(t, app, bashTarget(bash), input)
	if !strings.Contains(screen, "scan") {
		t.Fatalf("settled Bash screen does not contain scan: %q", screen)
	}
}

func parseBashVersion(output string) (string, error) {
	_, suffix, ok := strings.Cut(output, "version ")
	if !ok {
		return "", errors.New("Bash version marker missing")
	}
	fields := strings.Fields(suffix)
	if len(fields) == 0 {
		return "", errors.New("Bash version token missing")
	}
	return fields[0], nil
}

func prepareBash(setup shelltest.Setup) error {
	integration := setup.IntegrationFile("completion.bash")
	rc := setup.File("bashrc", "source "+shellQuote(integration)+"\n")
	setup.Env("SHELL", "/bin/sh")
	setup.Args("--noprofile", "--rcfile", rc, "-i")
	return nil
}

func interpretBash(observation shelltest.Observation) (string, error) {
	if observation.Version() == "" {
		return "", errors.New("missing verified Bash runtime version")
	}
	lines := observation.ScreenLines()
	if lines == nil {
		return "", errors.New("settled Bash screen unavailable")
	}
	return strings.Join(lines, "\n"), nil
}

func bashTarget(executable string) shelltest.Target[string] {
	runtimeConfig := shelltest.Runtime{
		Executable:   executable,
		VersionArgs:  []string{"--version"},
		ParseVersion: parseBashVersion,
	}
	target := shelltest.Target[string]{
		Provider:  cli.Shell.Bash,
		Runtime:   runtimeConfig,
		Prepare:   prepareBash,
		Interpret: interpretBash,
	}
	return target
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
