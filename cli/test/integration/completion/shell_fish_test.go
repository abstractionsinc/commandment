package completion

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/shelltest"
)

func testFish(t *testing.T, matrix []completionMatrixRow) {
	t.Run("3.0.0", func(t *testing.T) {
		runtimePath := completionRuntime(t, matrix, "fish", "3.0.0")
		result := shelltest.Complete(t, compileShellValueApplication(t, completeFishSpecimens), fishTarget(runtimePath), shelltest.Input{BeforeCursor: `alienctl "$SHELLTEST_FISH"`})
		requireRuntimeVersion(t, result, "3.0.0")
		requireScreenContains(t, result, "lexical current token")
		requireScreenExcludes(t, result, "expanded current token")
	})

	t.Run("4.0.0", func(t *testing.T) {
		runtimePath := completionRuntime(t, matrix, "fish", "4.0.0")
		result := shelltest.Complete(t, compileShellValueApplication(t, completeFishSpecimens), fishTarget(runtimePath), shelltest.Input{BeforeCursor: `alienctl "$SHELLTEST_FISH"`})
		requireRuntimeVersion(t, result, "4.0.0")
		requireScreenContains(t, result, "$SHELLTEST_FISH-one")
	})
}

func completeFishSpecimens(_ context.Context, _ *specimenConfig, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	if strings.Contains(request.Text, "SHELLTEST_FISH") {
		return cli.CompletionContribution{Candidates: []cli.CompletionCandidate{
			{Value: request.Text + "one", Description: "lexical current token"},
			{Value: request.Text + "two", Description: "lexical current token"},
		}}, nil
	}
	if request.Text == "fish-c" {
		// Fish 4 reports the expanded value to Commandment but still matches
		// dynamic candidates against the lexical source token. One source-
		// compatible sentinel makes that branch observable through the editor.
		return cli.CompletionContribution{Candidates: []cli.CompletionCandidate{
			{Value: "$SHELLTEST_FISH-one", Description: "expanded current token"},
		}}, nil
	}
	return cli.CompletionContribution{}, nil
}

func fishTarget(executable string) shelltest.Target[shellObservation] {
	return shelltest.Target[shellObservation]{
		Provider: cli.Shell.Fish,
		Runtime: shelltest.Runtime{
			Executable:  executable,
			VersionArgs: []string{"--version"},
			ParseVersion: func(output string) (string, error) {
				_, suffix, ok := strings.Cut(output, "fish, version ")
				if !ok {
					return "", errors.New("fish version marker missing")
				}
				fields := strings.Fields(suffix)
				if len(fields) == 0 {
					return "", errors.New("fish version token missing")
				}
				return fields[0], nil
			},
		},
		Prepare: func(setup shelltest.Setup) error {
			integration := setup.IntegrationFile("completion.fish")
			setup.File("xdg-config/fish/config.fish", "source "+fishQuote(integration)+"\n")
			setup.Env("LANG", "C")
			setup.Env("LC_ALL", "C")
			setup.Env("SHELLTEST_FISH", "fish-c")
			setup.Args("-i")
			return nil
		},
		Interpret: interpretShellObservation,
	}
}

func fishQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "\\'") + "'"
}
