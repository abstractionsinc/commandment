package shelltest

import (
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
)

const (
	bridgeBootstrapSuffix        = ".shelltest-bootstrap"
	generationObservationSubject = "observe completion generation"
)

type generatedCompletion struct {
	integration []byte
	generation  cli.ShellCompletionGeneration
	reserved    []string
}

func generateCompletion(app cli.Application, provider cli.ShellCompletionProvider, workspace string) (generatedCompletion, error) {
	integration, err := app.ShellCompletionScript(provider)
	if err != nil {
		failure := &failureError{
			subject: "production completion generation",
			cause:   err,
		}
		return generatedCompletion{}, failure
	}
	generation, err := observeGeneration(app)
	if err != nil {
		return generatedCompletion{}, err
	}
	if generation.TargetVersion != "" {
		failure := &failureError{
			subject: "observed completion generation target mismatch",
			detail:  strconv.Quote(generation.TargetVersion),
		}
		return generatedCompletion{}, failure
	}
	return validateGeneratedCompletion(workspace, []byte(integration), generation)
}

func generateCompletionForVersion(app cli.Application, provider cli.ShellCompletionProvider, generationTarget, workspace string) (generatedCompletion, error) {
	integration, err := app.ShellCompletionScriptForVersion(provider, generationTarget)
	if err != nil {
		failure := &failureError{
			subject: "production completion generation",
			cause:   err,
		}
		return generatedCompletion{}, failure
	}
	generation, err := observeGenerationForVersion(app, generationTarget)
	if err != nil {
		return generatedCompletion{}, err
	}
	if generation.TargetVersion != generationTarget {
		failure := &failureError{
			subject: "observed completion generation target mismatch",
			detail:  strconv.Quote(generation.TargetVersion) + " != " + strconv.Quote(generationTarget),
		}
		return generatedCompletion{}, failure
	}
	return validateGeneratedCompletion(workspace, []byte(integration), generation)
}

func observeGeneration(app cli.Application) (cli.ShellCompletionGeneration, error) {
	var observed cli.ShellCompletionGeneration
	calls := 0
	provider := cli.Shell.Custom("shelltest-generation-observer", func(generation cli.ShellCompletionGeneration) (string, error) {
		calls++
		observed = generation
		return "", nil
	})
	if _, err := app.ShellCompletionScript(provider); err != nil {
		failure := &failureError{
			subject: generationObservationSubject,
			cause:   err,
		}
		return cli.ShellCompletionGeneration{}, failure
	}
	if calls != 1 {
		failure := &failureError{
			subject: generationObservationSubject,
			detail:  "provider invocation count=" + strconv.Itoa(calls),
		}
		return cli.ShellCompletionGeneration{}, failure
	}
	return observed, nil
}

func observeGenerationForVersion(app cli.Application, target string) (cli.ShellCompletionGeneration, error) {
	var observed cli.ShellCompletionGeneration
	calls := 0
	provider := cli.Shell.Custom("shelltest-generation-observer", func(generation cli.ShellCompletionGeneration) (string, error) {
		calls++
		observed = generation
		return "", nil
	})
	if _, err := app.ShellCompletionScriptForVersion(provider, target); err != nil {
		failure := &failureError{
			subject: generationObservationSubject,
			cause:   err,
		}
		return cli.ShellCompletionGeneration{}, failure
	}
	if calls != 1 {
		failure := &failureError{
			subject: generationObservationSubject,
			detail:  "provider invocation count=" + strconv.Itoa(calls),
		}
		return cli.ShellCompletionGeneration{}, failure
	}
	return observed, nil
}

func validateGeneratedCompletion(workspace string, integration []byte, generation cli.ShellCompletionGeneration) (generatedCompletion, error) {
	if err := validateApplicationName(generation.ApplicationName); err != nil {
		return generatedCompletion{}, err
	}
	if !validEnvironmentName(generation.ActivationEnvironment) {
		failure := &failureError{
			subject: "invalid completion activation environment",
			detail:  strconv.Quote(generation.ActivationEnvironment),
		}
		return generatedCompletion{}, failure
	}
	if generation.ActivationValue == "" || strings.IndexByte(generation.ActivationValue, 0) >= 0 {
		failure := &failureError{
			subject: "invalid completion activation value",
		}
		return generatedCompletion{}, failure
	}

	helper := filepath.Join("bin", generation.ApplicationName)
	bootstrap := helper + bridgeBootstrapSuffix
	if err := probeFrameworkNames(workspace, generation.ApplicationName, generation.ApplicationName+bridgeBootstrapSuffix); err != nil {
		return generatedCompletion{}, err
	}

	result := generatedCompletion{
		integration: integration,
		generation:  generation,
		reserved:    []string{helper, bootstrap},
	}
	return result, nil
}

func validateApplicationName(name string) error {
	invalid := name == "" || name == "." || name == ".." || strings.IndexByte(name, 0) >= 0 || filepath.IsAbs(name) || filepath.VolumeName(name) != "" || strings.ContainsAny(name, `/\\`)
	if invalid {
		failure := &failureError{
			subject: "application command name cannot be materialized",
			detail:  strconv.Quote(name),
		}
		return failure
	}
	if runtime.GOOS == windowsOS && !validWindowsFileComponent(name) {
		failure := &failureError{
			subject: "application command name cannot be materialized on Windows",
			detail:  strconv.Quote(name),
		}
		return failure
	}
	return nil
}

func validWindowsFileComponent(name string) bool {
	if name == "" || strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") || strings.ContainsAny(name, `<>:"|?*`) {
		return false
	}
	for _, r := range name {
		if r > 0 && r < 0x20 {
			return false
		}
	}
	base := name
	if index := strings.IndexByte(base, '.'); index >= 0 {
		base = base[:index]
	}
	base = strings.ToUpper(base)
	switch base {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9", "COM¹", "COM²", "COM³", "LPT¹", "LPT²", "LPT³":
		return false
	default:
		return true
	}
}

func probeFrameworkNames(workspace string, names ...string) error {
	const directory = ".shelltest-name-probe"
	for _, name := range names {
		relative := filepath.Join(directory, name)
		if err := createWorkspaceFile(workspace, relative, nil, privateFileMode); err != nil {
			_ = removeWorkspaceDirectory(workspace, directory)
			failure := &failureError{
				subject: "probe application helper name",
				detail:  strconv.Quote(name),
				cause:   err,
			}
			return failure
		}
		if err := removeWorkspaceFile(workspace, relative); err != nil {
			failure := &failureError{
				subject: "remove application helper name probe",
				detail:  strconv.Quote(name),
				cause:   err,
			}
			return failure
		}
	}
	if err := removeWorkspaceDirectory(workspace, directory); err != nil {
		failure := &failureError{
			subject: "remove framework name probe directory",
			cause:   err,
		}
		return failure
	}
	return nil
}
