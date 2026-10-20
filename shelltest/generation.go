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
		return generatedCompletion{}, &failureError{
			subject: "production completion generation",
			cause:   err,
		}
	}
	generation, err := observeGeneration(app)
	if err != nil {
		return generatedCompletion{}, err
	}
	if generation.TargetVersion != "" {
		return generatedCompletion{}, &failureError{
			subject: "observed completion generation target mismatch",
			detail:  strconv.Quote(generation.TargetVersion),
		}
	}
	return validateGeneratedCompletion(workspace, []byte(integration), generation)
}

func generateCompletionForVersion(app cli.Application, provider cli.ShellCompletionProvider, generationTarget, workspace string) (generatedCompletion, error) {
	integration, err := app.ShellCompletionScriptForVersion(provider, generationTarget)
	if err != nil {
		return generatedCompletion{}, &failureError{
			subject: "production completion generation",
			cause:   err,
		}
	}
	generation, err := observeGenerationForVersion(app, generationTarget)
	if err != nil {
		return generatedCompletion{}, err
	}
	if generation.TargetVersion != generationTarget {
		return generatedCompletion{}, &failureError{
			subject: "observed completion generation target mismatch",
			detail:  strconv.Quote(generation.TargetVersion) + " != " + strconv.Quote(generationTarget),
		}
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
		return cli.ShellCompletionGeneration{}, &failureError{
			subject: generationObservationSubject,
			cause:   err,
		}
	}
	if calls != 1 {
		return cli.ShellCompletionGeneration{}, &failureError{
			subject: generationObservationSubject,
			detail:  "provider invocation count=" + strconv.Itoa(calls),
		}
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
		return cli.ShellCompletionGeneration{}, &failureError{
			subject: generationObservationSubject,
			cause:   err,
		}
	}
	if calls != 1 {
		return cli.ShellCompletionGeneration{}, &failureError{
			subject: generationObservationSubject,
			detail:  "provider invocation count=" + strconv.Itoa(calls),
		}
	}
	return observed, nil
}

func validateGeneratedCompletion(workspace string, integration []byte, generation cli.ShellCompletionGeneration) (generatedCompletion, error) {
	if err := validateApplicationName(generation.ApplicationName); err != nil {
		return generatedCompletion{}, err
	}
	if !validEnvironmentName(generation.ActivationEnvironment) {
		return generatedCompletion{}, &failureError{
			subject: "invalid completion activation environment",
			detail:  strconv.Quote(generation.ActivationEnvironment),
		}
	}
	if generation.ActivationValue == "" || strings.IndexByte(generation.ActivationValue, 0) >= 0 {
		return generatedCompletion{}, &failureError{subject: "invalid completion activation value"}
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
		return &failureError{
			subject: "application command name cannot be materialized",
			detail:  strconv.Quote(name),
		}
	}
	if runtime.GOOS == windowsOS && !validWindowsFileComponent(name) {
		return &failureError{
			subject: "application command name cannot be materialized on Windows",
			detail:  strconv.Quote(name),
		}
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
			return &failureError{
				subject: "probe application helper name",
				detail:  strconv.Quote(name),
				cause:   err,
			}
		}
		if err := removeWorkspaceFile(workspace, relative); err != nil {
			return &failureError{
				subject: "remove application helper name probe",
				detail:  strconv.Quote(name),
				cause:   err,
			}
		}
	}
	if err := removeWorkspaceDirectory(workspace, directory); err != nil {
		return &failureError{
			subject: "remove framework name probe directory",
			cause:   err,
		}
	}
	return nil
}
