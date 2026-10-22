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
		t.Parallel()
		runtimePath := completionRuntime(t, matrix, "powershell", "5.1")
		result := shelltest.Complete(t, compileShellValueApplication(t, completeNoCandidates), windowsPowerShellTarget(runtimePath), shelltest.Input{
			BeforeCursor: "alienctl fallback-",
		})
		requireRuntimeVersion(t, result, "5.1")
		requireScreenContains(t, result, "fallback-native")
	})

	t.Run("7.6.6", func(t *testing.T) {
		t.Parallel()
		runtimePath := completionRuntime(t, matrix, "powershell", "7.6.6")
		result := shelltest.Complete(t, compileShellValueApplication(t, completeNoCandidates), powerShellTarget(runtimePath), shelltest.Input{
			BeforeCursor: "alienctl fallback-",
		})
		requireRuntimeVersion(t, result, "7.6.6")
		requireScreenContains(t, result, "fallback-")
		requireScreenExcludes(t, result, "fallback-native")
	})
}

func powerShellTarget(executable string) shelltest.Target[shellObservation] {
	runtime := shelltest.Runtime{
		Executable: executable,
		VersionArgs: []string{
			"-NoLogo",
			"-NoProfile",
			"-NonInteractive",
			"-Command",
			"$PSVersionTable.PSVersion.ToString()",
		},
		ParseVersion: parsePowerShellVersion,
	}
	target := shelltest.Target[shellObservation]{
		Provider: cli.Shell.PowerShell,
		Runtime:  runtime,
		Prepare: func(setup shelltest.Setup) error {
			integration := setup.IntegrationFile("completion.ps1")
			setup.File("fallback-native.txt", "native fallback control\n")
			setup.Args("-NoLogo", "-NoProfile", "-NoExit", "-Command", ". "+powerShellQuote(integration))
			return nil
		},
		Interpret: interpretShellObservation,
	}
	return target
}

func windowsPowerShellTarget(executable string) shelltest.Target[shellObservation] {
	target := powerShellTarget(executable)
	target.Runtime.ParseVersion = parseWindowsPowerShellVersion
	return target
}

func parsePowerShellVersion(output string) (string, error) {
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) == 0 {
		return "", errors.New("PowerShell version token missing")
	}
	return fields[len(fields)-1], nil
}

func parseWindowsPowerShellVersion(output string) (string, error) {
	version, err := parsePowerShellVersion(output)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(version, "5.1") {
		return "5.1", nil
	}
	return version, nil
}

func powerShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
