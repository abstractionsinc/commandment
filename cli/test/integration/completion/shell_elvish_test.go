package completion

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/shelltest"
)

func testElvish(t *testing.T, matrix []completionMatrixRow) {
	t.Run("0.11.0", func(t *testing.T) {
		t.Parallel()
		runtimePath := completionRuntime(t, matrix, "elvish", "0.11.0")
		result := shelltest.CompleteForGenerationTarget(t, compileShellCommandApplication(t), elvishTarget(runtimePath), "0.11.0", shelltest.Input{
			BeforeCursor: "alienctl ana",
		})
		requireRuntimeVersion(t, result, "0.11.0")
		requireScreenContains(t, result, "analyze")
	})

	t.Run("0.12.0", func(t *testing.T) {
		t.Parallel()
		runtimePath := completionRuntime(t, matrix, "elvish", "0.12.0")
		result := shelltest.CompleteForGenerationTarget(t, compileShellValueApplication(t, completeElvishSpecimens), elvishTarget(runtimePath), "0.12.0", shelltest.Input{
			BeforeCursor: "alienctl elvish-c",
		})
		requireRuntimeVersion(t, result, "0.12.0")
		requireScreenContains(t, result, "elvish-cow")
	})
}

func completeElvishSpecimens(context.Context, *specimenConfig, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	contribution := cli.CompletionContribution{
		Candidates: []cli.CompletionCandidate{
			{
				Value:       "elvish-cow",
				Description: "bovine specimen",
			},
		},
	}
	return contribution, nil
}

func elvishTarget(executable string) shelltest.Target[shellObservation] {
	runtime := shelltest.Runtime{
		Executable:  executable,
		VersionArgs: []string{"-version"},
		ParseVersion: func(output string) (string, error) {
			fields := strings.Fields(strings.TrimSpace(output))
			if len(fields) == 0 {
				return "", errors.New("Elvish version token missing")
			}
			for _, field := range fields {
				candidate := strings.TrimPrefix(field, "v")
				if len(candidate) > 0 && strings.Count(candidate, ".") >= 2 && candidate[0] >= '0' && candidate[0] <= '9' {
					return candidate, nil
				}
			}
			return "", errors.New("Elvish semantic version missing")
		},
	}
	target := shelltest.Target[shellObservation]{
		Provider: cli.Shell.Elvish,
		Runtime:  runtime,
		Prepare: func(setup shelltest.Setup) error {
			setup.IntegrationFile("home/.elvish/rc.elv")
			setup.Env("SHELL", "/bin/sh")
			return nil
		},
		Interpret: interpretShellObservation,
	}
	return target
}
