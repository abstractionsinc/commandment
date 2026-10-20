package completion

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/shelltest"
)

func testBash(t *testing.T, matrix []completionMatrixRow) {
	t.Run("3.2.57", func(t *testing.T) {
		runtimePath := completionRuntime(t, matrix, "bash", "3.2.57")
		result := shelltest.Complete(t, compileShellCommandApplication(t), bashTarget(runtimePath), shelltest.Input{BeforeCursor: "alienctl ana"})
		requireRuntimeVersion(t, result, "3.2.57")
		requireScreenContains(t, result, "analyze")
	})

	t.Run("4.0.38", func(t *testing.T) {
		runtimePath := completionRuntime(t, matrix, "bash", "4.0.38")
		result := shelltest.Complete(t, compileShellValueApplication(t, completeBashOrdering), bashListingTarget(runtimePath), shelltest.Input{BeforeCursor: "alienctl order-"})
		requireRuntimeVersion(t, result, "4.0.38")
		requireScreenOrder(t, result, "order-alpha", "order-zeta")
	})

	t.Run("4.4.0", func(t *testing.T) {
		runtimePath := completionRuntime(t, matrix, "bash", "4.4.0")
		result := shelltest.Complete(t, compileShellValueApplication(t, completeBashOrdering), bashListingTarget(runtimePath), shelltest.Input{BeforeCursor: "alienctl order-"})
		requireRuntimeVersion(t, result, "4.4.0")
		requireScreenOrder(t, result, "order-zeta", "order-alpha")
	})

	t.Run("5.3.0", func(t *testing.T) {
		runtimePath := completionRuntime(t, matrix, "bash", "5.3.0")
		result := shelltest.Complete(t, compileShellValueApplication(t, completeBashSpecimens), bashTarget(runtimePath), shelltest.Input{BeforeCursor: "alienctl crop"})
		requireRuntimeVersion(t, result, "5.3.0")
		requireScreenContains(t, result, `crop\ circle`)
	})
}

func completeBashSpecimens(context.Context, *specimenConfig, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	return cli.CompletionContribution{
		Candidates: []cli.CompletionCandidate{
			{Value: "cow", Description: "bovine specimen", GroupID: "lifeforms"},
			{Value: "crop circle", Description: "field anomaly", GroupID: "evidence"},
			{Value: "lambda-signal", Description: "signal specimen", GroupID: "signals"},
		},
		Groups: []cli.CompletionGroup{
			{ID: "lifeforms", Label: "Lifeforms"},
			{ID: "evidence", Label: "Evidence"},
			{ID: "signals", Label: "Signals"},
		},
	}, nil
}

func completeBashOrdering(context.Context, *specimenConfig, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	return cli.CompletionContribution{Candidates: []cli.CompletionCandidate{
		{Value: "order-zeta", Priority: 100},
		{Value: "order-alpha", Priority: 1},
	}}, nil
}

func bashTarget(executable string) shelltest.Target[shellObservation] {
	return bashTargetWithBinding(executable, "")
}

func bashListingTarget(executable string) shelltest.Target[shellObservation] {
	return bashTargetWithBinding(executable, `bind '"\C-i": possible-completions'`+"\n")
}

func bashTargetWithBinding(executable, binding string) shelltest.Target[shellObservation] {
	return shelltest.Target[shellObservation]{
		Provider: cli.Shell.Bash,
		Runtime: shelltest.Runtime{
			Executable:  executable,
			VersionArgs: []string{"--version"},
			ParseVersion: func(output string) (string, error) {
				_, suffix, ok := strings.Cut(output, "version ")
				if !ok {
					return "", errors.New("Bash version marker missing")
				}
				fields := strings.Fields(suffix)
				if len(fields) == 0 {
					return "", errors.New("Bash version token missing")
				}
				return strings.TrimSuffix(fields[0], "(1)-release"), nil
			},
		},
		Prepare: func(setup shelltest.Setup) error {
			integration := setup.IntegrationFile("completion.bash")
			rc := setup.File("bashrc", "source "+bashQuote(integration)+"\n"+binding)
			setup.Env("SHELL", "/bin/sh")
			setup.Args("--noprofile", "--rcfile", rc, "-i")
			return nil
		},
		Interpret: interpretShellObservation,
	}
}

func bashQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
