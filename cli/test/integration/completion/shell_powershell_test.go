package completion

import (
	"errors"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/shelltest"
)

func testPowerShell(t *testing.T, matrix []completionMatrixRow) {
	t.Run("5.1", func(t *testing.T) {
		runtimePath := completionRuntime(t, matrix, "powershell", "5.1")
		result := shelltest.Complete(t, compileShellValueApplication(t, completeNoCandidates), powerShellTarget(runtimePath, true), shelltest.Input{BeforeCursor: "alienctl fallback-"})
		requireRuntimeVersion(t, result, "5.1")
		requireScreenContains(t, result, "fallback-native")
	})

	t.Run("7.6.6", func(t *testing.T) {
		runtimePath := completionRuntime(t, matrix, "powershell", "7.6.6")
		result := shelltest.Complete(t, compileShellValueApplication(t, completeNoCandidates), powerShellTarget(runtimePath, false), shelltest.Input{BeforeCursor: "alienctl fallback-"})
		requireRuntimeVersion(t, result, "7.6.6")
		requireScreenContains(t, result, "fallback-")
		requireScreenExcludes(t, result, "fallback-native")
	})
}

func powerShellTarget(executable string, windows51 bool) shelltest.Target[shellObservation] {
	return shelltest.Target[shellObservation]{
		Provider: cli.Shell.PowerShell,
		Runtime: shelltest.Runtime{
			Executable: executable,
			VersionArgs: []string{
				"-NoLogo",
				"-NoProfile",
				"-NonInteractive",
				"-Command",
				"$PSVersionTable.PSVersion.ToString()",
			},
			ParseVersion: func(output string) (string, error) {
				fields := strings.Fields(strings.TrimSpace(output))
				if len(fields) == 0 {
					return "", errors.New("PowerShell version token missing")
				}
				version := fields[len(fields)-1]
				if windows51 && strings.HasPrefix(version, "5.1") {
					return "5.1", nil
				}
				return version, nil
			},
		},
		Prepare: func(setup shelltest.Setup) error {
			integration := setup.IntegrationFile("completion.ps1")
			setup.File("fallback-native.txt", "native fallback control\n")
			setup.Args("-NoLogo", "-NoProfile", "-NoExit", "-Command", ". "+powerShellQuote(integration))
			return nil
		},
		Interpret: interpretShellObservation,
	}
}

func powerShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
