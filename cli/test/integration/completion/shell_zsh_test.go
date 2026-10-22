package completion

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/shelltest"
)

func testZsh(t *testing.T, matrix []completionMatrixRow) {
	t.Run("4.0.1", func(t *testing.T) {
		t.Parallel()
		runtimePath := completionRuntime(t, matrix, "zsh", "4.0.1")
		result := shelltest.Complete(t, compileShellCommandApplication(t), zshTarget(runtimePath), shelltest.Input{
			BeforeCursor: "alienctl ana",
		})
		requireRuntimeVersion(t, result, "4.0.1")
		requireScreenContains(t, result, "analyze")
	})

	t.Run("5.9.2", func(t *testing.T) {
		t.Parallel()
		runtimePath := completionRuntime(t, matrix, "zsh", "5.9.2")
		result := shelltest.Complete(t, compileShellValueApplication(t, completeZshSpecimens), zshTarget(runtimePath), shelltest.Input{
			BeforeCursor: "alienctl c",
		})
		requireRuntimeVersion(t, result, "5.9.2")
		requireScreenContains(t, result, "primary bovine", "duplicate bovine", "Lifeforms % literal", "Evidence % literal")
		requireScreenOccurrences(t, result, "cow", 2)
	})
}

func completeZshSpecimens(context.Context, *specimenConfig, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	contribution := cli.CompletionContribution{
		Candidates: []cli.CompletionCandidate{
			{
				Value:       "cow",
				Description: "primary bovine",
				GroupID:     "life%forms",
			},
			{
				Value:       "cow",
				Description: "duplicate bovine",
				GroupID:     "life%forms",
			},
			{
				Value:       "crop",
				Description: "field evidence",
				GroupID:     "evidence%set",
			},
		},
		Groups: []cli.CompletionGroup{
			{
				ID:    "life%forms",
				Label: "Lifeforms % literal",
			},
			{
				ID:    "evidence%set",
				Label: "Evidence % literal",
			},
		},
	}
	return contribution, nil
}

func zshTarget(executable string) shelltest.Target[shellObservation] {
	runtime := shelltest.Runtime{
		Executable:  executable,
		VersionArgs: []string{"--version"},
		ParseVersion: func(output string) (string, error) {
			fields := strings.Fields(output)
			if len(fields) < 2 || fields[0] != "zsh" {
				return "", errors.New("Zsh version token missing")
			}
			return fields[1], nil
		},
	}
	target := shelltest.Target[shellObservation]{
		Provider: cli.Shell.Zsh,
		Runtime:  runtime,
		Prepare: func(setup shelltest.Setup) error {
			integration := setup.IntegrationFile("completion.zsh")
			root := zshInstallRoot(setup.Executable())
			functions, err := filepath.Glob(filepath.Join(root, "share", "zsh", "*", "functions"))
			if err != nil {
				return fmt.Errorf("locate Zsh completion functions: %w", err)
			}
			slices.Sort(functions)
			modules, err := zshModuleRoots(root)
			if err != nil {
				return err
			}
			var environment strings.Builder
			if len(modules) != 0 {
				environment.WriteString("module_path=(")
				environment.WriteString(zshQuotedList(modules))
				environment.WriteString(" $module_path)\n")
			}
			if len(functions) != 0 {
				environment.WriteString("fpath=(")
				environment.WriteString(zshQuotedList(functions))
				environment.WriteString(" $fpath)\n")
			}
			setup.File("zdotdir/.zshenv", environment.String())
			var rc strings.Builder
			rc.WriteString("zmodload zsh/zle\n")
			rc.WriteString("zmodload zsh/complete\n")
			rc.WriteString("autoload -U compinit\n")
			rc.WriteString("compinit -D -i\n")
			rc.WriteString("zstyle ':completion:*' completer _complete\n")
			rc.WriteString("source ")
			rc.WriteString(zshQuote(integration))
			rc.WriteByte('\n')
			rc.WriteString("zle -C __commandment_shelltest_widget .complete-word _main_complete\n")
			rc.WriteString("bindkey '^I' __commandment_shelltest_widget\n")
			rc.WriteString("bindkey -M main '^I' __commandment_shelltest_widget 2>/dev/null || true\n")
			rc.WriteString("bindkey -M emacs '^I' __commandment_shelltest_widget 2>/dev/null || true\n")
			rc.WriteString("bindkey -M viins '^I' __commandment_shelltest_widget 2>/dev/null || true\n")
			setup.File("zdotdir/.zshrc", rc.String())
			setup.Env("SHELL", "/bin/sh")
			setup.Args("-i")
			return nil
		},
		Interpret: interpretShellObservation,
	}
	return target
}

func zshInstallRoot(executable string) string {
	root := filepath.Dir(filepath.Dir(executable))
	shellRuntimeRoot := os.Getenv("COMMANDMENT_SHELL_RUNTIME_ROOT")
	devRuntimeRoot := os.Getenv("COMMANDMENT_DEV_RUNTIME_ROOT")
	if shellRuntimeRoot == "" || devRuntimeRoot == "" {
		return root
	}
	relative, err := filepath.Rel(shellRuntimeRoot, executable)
	if err != nil || relative == "." || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return root
	}
	original := filepath.Join(devRuntimeRoot, relative)
	return filepath.Dir(filepath.Dir(original))
}

func zshModuleRoots(root string) ([]string, error) {
	candidates, err := filepath.Glob(filepath.Join(root, "lib", "zsh", "*"))
	if err != nil {
		return nil, fmt.Errorf("locate Zsh module roots: %w", err)
	}
	modules := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		info, statErr := os.Stat(filepath.Join(candidate, "zsh", "zle.so"))
		if statErr == nil && !info.IsDir() {
			modules = append(modules, candidate)
		}
	}
	slices.Sort(modules)
	return modules, nil
}

func zshQuotedList(values []string) string {
	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = zshQuote(value)
	}
	return strings.Join(quoted, " ")
}

func zshQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
