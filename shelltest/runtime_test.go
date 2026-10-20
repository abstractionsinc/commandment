package shelltest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestVerifyRuntimePreservesSelectedLaunchSpellingAndRequiresObservedVersionToken(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == windowsOS {
		t.Skip("fixture uses POSIX shell")
	}
	workspace := t.TempDir()
	targetPath := filepath.Join(workspace, "runtime-target")
	if err := writeExecutableFixture(targetPath, "#!/bin/sh\nprintf 'shell version 4.2\\n'\n"); err != nil {
		t.Fatal(err)
	}
	runtimePath := filepath.Join(workspace, "runtime")
	if err := os.Symlink(targetPath, runtimePath); err != nil {
		t.Fatal(err)
	}
	verified, err := verifyRuntime(context.Background(), workspace, Runtime{
		Executable:  runtimePath,
		VersionArgs: []string{"--version"},
		ParseVersion: func(output string) (string, error) {
			if !strings.Contains(output, "shell version") {
				return "", errors.New("missing version")
			}
			return "4.2", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(runtimePath)
	if err != nil {
		t.Fatal(err)
	}
	if verified.executable != want || verified.version != "4.2" {
		t.Fatalf("verified runtime = %#v, want executable %q version 4.2", verified, want)
	}
	if err := verifyRuntimeExecutable(verified); err != nil {
		t.Fatalf("verified runtime changed unexpectedly: %v", err)
	}
	if err := writeExecutableFixture(targetPath, "#!/bin/sh\nprintf 'shell version 4.3\\n'\n"); err != nil {
		t.Fatal(err)
	}
	if err := verifyRuntimeExecutable(verified); err == nil || !strings.Contains(err.Error(), "changed after version verification") {
		t.Fatalf("mutated runtime verification error = %v", err)
	}
	_, err = verifyRuntime(context.Background(), workspace, Runtime{
		Executable:   runtimePath,
		VersionArgs:  []string{"--version"},
		ParseVersion: func(string) (string, error) { return "not-observed", nil },
	})
	if err == nil || !strings.Contains(err.Error(), "not observed verbatim") {
		t.Fatalf("unobserved parser token error = %v", err)
	}
}

func TestSnapshotTargetClonesVersionArgumentsAndAcceptsEmptyInput(t *testing.T) {
	t.Parallel()
	args := []string{"--version"}
	target := Target[struct{}]{
		Provider:  cli.Shell.Bash,
		Runtime:   Runtime{Executable: "runtime", VersionArgs: args, ParseVersion: func(string) (string, error) { return "v", nil }},
		Interpret: func(Observation) (struct{}, error) { return struct{}{}, nil },
	}
	got, err := snapshotTarget(target, Input{})
	if err != nil {
		t.Fatal(err)
	}
	args[0] = "changed"
	if got.Runtime.VersionArgs[0] != "--version" {
		t.Fatalf("VersionArgs not snapshotted: %#v", got.Runtime.VersionArgs)
	}
}

func TestSnapshotTargetRejectsInvalidDeclarations(t *testing.T) {
	t.Parallel()
	valid := Target[struct{}]{
		Provider:  cli.Shell.Bash,
		Runtime:   Runtime{Executable: "runtime", VersionArgs: []string{"--version"}, ParseVersion: func(string) (string, error) { return "v", nil }},
		Interpret: func(Observation) (struct{}, error) { return struct{}{}, nil },
	}
	tests := []struct {
		name   string
		target Target[struct{}]
		input  Input
	}{
		{name: "zero-target"},
		{name: "missing-executable", target: func() Target[struct{}] { value := valid; value.Runtime.Executable = ""; return value }()},
		{name: "missing-version-args", target: func() Target[struct{}] { value := valid; value.Runtime.VersionArgs = nil; return value }()},
		{name: "missing-version-parser", target: func() Target[struct{}] { value := valid; value.Runtime.ParseVersion = nil; return value }()},
		{name: "missing-interpret", target: func() Target[struct{}] { value := valid; value.Interpret = nil; return value }()},
		{name: "invalid-before-cursor", target: valid, input: Input{BeforeCursor: string([]byte{0xff})}},
		{name: "invalid-after-cursor", target: valid, input: Input{AfterCursor: string([]byte{0xff})}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := snapshotTarget(test.target, test.input); err == nil {
				t.Fatal("invalid declaration accepted")
			}
		})
	}
}

func writeExecutableFixture(path, contents string) error {
	if err := os.WriteFile(path, []byte(contents), privateFileMode); err != nil {
		return fmt.Errorf("write executable fixture: %w", err)
	}
	// #nosec G302 -- the fixture must be executable and remains private to the test.
	if err := os.Chmod(path, 0o700); err != nil {
		return fmt.Errorf("make executable fixture runnable: %w", err)
	}
	return nil
}

func TestBoundedRuntimeProbeOutputMarksOverflowWithoutGrowingPastLimit(t *testing.T) {
	t.Parallel()
	output := boundedOutput{limit: 4}
	if written, err := output.Write([]byte("abcdef")); err != nil || written != 6 {
		t.Fatalf("Write() = %d, %v", written, err)
	}
	if !output.exceeded {
		t.Fatal("overflow was not recorded")
	}
	if got := output.String(); got != "abcd" {
		t.Fatalf("bounded output = %q, want %q", got, "abcd")
	}
}

func TestValidateGeneratedCompletionRejectsUnmaterializableGenerationFacts(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	valid := cli.ShellCompletionGeneration{
		ApplicationName:       "alienctl",
		ActivationEnvironment: "COMMANDMENT_COMPLETE",
		ActivationValue:       "adapter",
	}
	for name, mutate := range map[string]func(*cli.ShellCompletionGeneration){
		"empty-application": func(g *cli.ShellCompletionGeneration) { g.ApplicationName = "" },
		"path-application":  func(g *cli.ShellCompletionGeneration) { g.ApplicationName = "bin/alienctl" },
		"empty-activation":  func(g *cli.ShellCompletionGeneration) { g.ActivationEnvironment = "" },
		"empty-value":       func(g *cli.ShellCompletionGeneration) { g.ActivationValue = "" },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			generation := valid
			mutate(&generation)
			if _, err := validateGeneratedCompletion(workspace, nil, generation); err == nil {
				t.Fatal("invalid generation facts accepted")
			}
		})
	}
}
