package completion

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/shelltest"
)

func testNushell(t *testing.T, matrix []completionMatrixRow) {
	t.Run("0.72.0", func(t *testing.T) {
		t.Parallel()
		runtimePath := completionRuntime(t, matrix, "nushell", "0.72.0")
		result := shelltest.CompleteForGenerationTarget(t, compileShellCommandApplication(t), nushellTarget(runtimePath), "0.72.0", shelltest.Input{
			BeforeCursor: "alienctl ana",
		})
		requireRuntimeVersion(t, result, "0.72.0")
		requireScreenContains(t, result, "analyze")
	})

	t.Run("0.102.0", func(t *testing.T) {
		t.Parallel()
		runtimePath := completionRuntime(t, matrix, "nushell", "0.102.0")
		result := shelltest.CompleteForGenerationTarget(t, compileShellValueApplication(t, completeNoCandidates), nushellTarget(runtimePath), "0.102.0", shelltest.Input{
			BeforeCursor: "alienctl fallback-",
		})
		requireRuntimeVersion(t, result, "0.102.0")
		// The 0.102 band uses null to decline on integration failure. A successful
		// empty Commandment result must remain a handled empty list instead, or
		// Reedline would fall through to the matching native file.
		requireScreenContains(t, result, "fallback-")
		requireScreenExcludes(t, result, "fallback-native.txt")
	})

	t.Run("0.108.0", func(t *testing.T) {
		t.Parallel()
		runtimePath := completionRuntime(t, matrix, "nushell", "0.108.0")
		result := shelltest.CompleteForGenerationTarget(t, compileShellCommandApplication(t), nushellTarget(runtimePath), "0.108.0", shelltest.Input{
			BeforeCursor: "alienctl ana",
		})
		requireRuntimeVersion(t, result, "0.108.0")
		requireScreenContains(t, result, "analyze")
		requireScreenExcludes(t, result, "prior-hijack", "hostile global completer")
	})

	t.Run("0.113.0", func(t *testing.T) {
		t.Parallel()
		runtimePath := completionRuntime(t, matrix, "nushell", "0.113.0")
		result := shelltest.CompleteForGenerationTarget(t, compileShellValueApplication(t, completeNushellSpecimens), nushellTarget(runtimePath), "0.113.0", shelltest.Input{
			BeforeCursor: "alienctl nu-c",
		})
		requireRuntimeVersion(t, result, "0.113.0")
		requireScreenContains(t, result, "bovine specimen")
	})

	t.Run("0.115.1", func(t *testing.T) {
		t.Parallel()
		runtimePath := completionRuntime(t, matrix, "nushell", "0.115.1")
		result := shelltest.Complete(t, compileShellCommandApplication(t), nushellTarget(runtimePath), shelltest.Input{
			BeforeCursor: "alienctl ",
		})
		requireRuntimeVersion(t, result, "0.115.1")
		requireScreenContains(t, result, "abduct")
	})
}

func completeNushellSpecimens(context.Context, *specimenConfig, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	contribution := cli.CompletionContribution{
		Candidates: []cli.CompletionCandidate{
			{
				Value:       "nu-cow",
				Description: "bovine specimen",
			},
			{
				Value:       "nu-crop",
				Description: "field specimen",
			},
		},
	}
	return contribution, nil
}

func nushellTarget(executable string) shelltest.Target[shellObservation] {
	runtime := shelltest.Runtime{
		Executable:  executable,
		VersionArgs: []string{"--version"},
		ParseVersion: func(output string) (string, error) {
			fields := strings.Fields(strings.TrimSpace(output))
			if len(fields) == 0 {
				return "", errors.New("Nushell version token missing")
			}
			return fields[len(fields)-1], nil
		},
	}
	target := shelltest.Target[shellObservation]{
		Provider: cli.Shell.Nushell,
		Runtime:  runtime,
		Prepare: func(setup shelltest.Setup) error {
			integration := setup.IntegrationFile("completion.nu")
			setup.File("fallback-native.txt", "native fallback control\n")
			envConfig := setup.File("env.nu", "")
			config := "$env.config.show_banner = false\nsource " + nushellQuote(integration) + "\n"
			switch setup.Version() {
			case "0.72.0":
				config = "let-env config = { show_banner: false completions: { external: { enable: true max_results: 100 completer: null } } }; source " + nushellQuote(integration) + "\n"
			case "0.108.0":
				config += `$env.config.completions.external.completer = {|spans| [{value: "prior-hijack", description: "hostile global completer"}] }` + "\n"
			}
			configPath := setup.File("config.nu", config)
			setup.Env("SHELL", "/bin/sh")
			setup.Env("PROMPT_INDICATOR", "> ")
			setup.Args("--env-config", envConfig, "--config", configPath)
			return nil
		},
		Interpret: interpretShellObservation,
	}
	return target
}

func nushellQuote(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\t", `\t`, "\r", `\r`, "\n", `\n`)
	return `"` + replacer.Replace(value) + `"`
}
