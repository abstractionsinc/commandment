package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
)

const (
	shellCompletionActivationStemLimit = 32
	shellCompletionActivationValue     = "1"
)

var (
	errInvalidShellCompletionProvider        = errors.New("cli: invalid shell completion provider")
	errInvalidShellCompletionApplicationName = errors.New("cli: invalid shell completion application name")
	errInvalidShellCompletionGeneration      = errors.New("cli: invalid shell completion generation")
	errUnsupportedShellCompletionTarget      = errors.New("cli: unsupported shell completion generation target")
)

// shellCompletionError preserves shell-generation ownership while retaining the underlying cause.
type shellCompletionError struct {
	kind   error
	detail string
	cause  error
}

func (err *shellCompletionError) Error() string {
	text := err.kind.Error()
	if err.detail != "" {
		text += ": " + err.detail
	}
	if err.cause != nil {
		text += ": " + err.cause.Error()
	}
	return text
}

func (err *shellCompletionError) Is(target error) bool { return target == err.kind }

func (err *shellCompletionError) Unwrap() error { return err.cause }

func shellCompletionGeneration(applicationName, targetVersion string) ShellCompletionGeneration {
	shellCompletionGeneration := ShellCompletionGeneration{
		ApplicationName:       applicationName,
		TargetVersion:         targetVersion,
		ActivationEnvironment: shellCompletionActivationName(applicationName),
		ActivationValue:       shellCompletionActivationValue,
	}

	return shellCompletionGeneration
}

func generateCompiledShellCompletionScript(applicationName string, provider shellCompletionProvider, targetVersion string) (string, error) {
	if applicationName == "" || containsForbiddenDeclarationText(applicationName) || !provider.Valid() {
		return "", errInvalidShellCompletionProvider
	}
	generation := shellCompletionGeneration(applicationName, targetVersion)
	script, err := provider.Generate(generation)
	if err != nil {
		result := &shellCompletionError{
			kind:   errInvalidShellCompletionGeneration,
			detail: "generate shell completion script",
			cause:  err,
		}
		return "", result
	}
	return script, nil
}

func generateShellCompletionScript(applicationName string, descriptor shellCompletionDescriptor, targetVersion string) (string, error) {
	if applicationName == "" || containsForbiddenDeclarationText(applicationName) {
		return "", errInvalidShellCompletionApplicationName
	}
	if !descriptor.valid() {
		return "", errInvalidShellCompletionProvider
	}
	if targetVersion != "" && descriptor.family != "" && !descriptor.versionSensitive {
		return "", &shellCompletionError{
			kind:   errUnsupportedShellCompletionTarget,
			detail: string(descriptor.family),
		}
	}
	return descriptor.generator(shellCompletionGeneration(applicationName, targetVersion))
}

func generateBuiltInShellCompletionScript(family ShellFamily, generation ShellCompletionGeneration) (string, error) {
	request := shellCompletionScriptRequest{
		ApplicationName:       generation.ApplicationName,
		Family:                family,
		TargetVersion:         generation.TargetVersion,
		ActivationEnvironment: generation.ActivationEnvironment,
		ActivationValue:       generation.ActivationValue,
	}
	if err := validateShellCompletionScriptRequest(request); err != nil {
		return "", err
	}
	switch request.Family {
	case ShellFamilyBash:
		return generateBashCompletionScript(request), nil
	case ShellFamilyZsh:
		return generateZshCompletionScript(request), nil
	case ShellFamilyFish:
		return generateFishCompletionScript(request), nil
	case ShellFamilyPowerShell:
		return generatePowerShellCompletionScript(request), nil
	case ShellFamilyElvish:
		return generateElvishCompletionScript(request)
	case ShellFamilyNushell:
		return generateNushellCompletionScript(request)
	default:
		return "", &shellCompletionError{
			kind:   errUnsupportedShellCompletionTarget,
			detail: "built-in shell completion family " + strconv.Quote(string(request.Family)),
		}
	}
}

type shellCompletionScriptRequest struct {
	ApplicationName       string
	Family                ShellFamily
	TargetVersion         string
	ActivationEnvironment string
	ActivationValue       string
}

func validateShellCompletionScriptRequest(request shellCompletionScriptRequest) error {
	if request.ApplicationName == "" || containsForbiddenDeclarationText(request.ApplicationName) ||
		request.ActivationEnvironment == "" || request.ActivationValue == "" {
		return errInvalidShellCompletionGeneration
	}
	if request.TargetVersion != "" && request.Family != ShellFamilyElvish && request.Family != ShellFamilyNushell {
		return &shellCompletionError{
			kind:   errUnsupportedShellCompletionTarget,
			detail: string(request.Family),
		}
	}
	return nil
}

func shellCompletionActivationName(applicationName string) string {
	var stem strings.Builder
	underscore := false
	for index := range len(applicationName) {
		value := applicationName[index]
		if value >= 'a' && value <= 'z' {
			value -= 'a' - 'A'
		}
		if (value >= 'A' && value <= 'Z') || (value >= '0' && value <= '9') {
			stem.WriteByte(value)
			underscore = false
			continue
		}
		if stem.Len() > 0 && !underscore {
			stem.WriteByte('_')
			underscore = true
		}
	}

	name := strings.Trim(stem.String(), "_")
	if name == "" {
		name = "APP"
	}
	if len(name) > shellCompletionActivationStemLimit {
		name = name[:shellCompletionActivationStemLimit]
	}

	sum := sha256.Sum256([]byte(applicationName))
	return "_" + name + "_" + strings.ToUpper(hex.EncodeToString(sum[:16])) + "_COMPLETE"
}

func shellCompletionIdentifier(applicationName string) string {
	sum := sha256.Sum256([]byte(applicationName))
	return hex.EncodeToString(sum[:6])
}

func renderShellCompletionScript(template string, replacements ...string) string {
	return strings.NewReplacer(replacements...).Replace(template)
}

func shellSingleQuoted(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func powerShellSingleQuoted(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func fishSingleQuoted(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `'`, `\'`)
	return "'" + value + "'"
}

func elvishSingleQuoted(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }

func nushellDoubleQuoted(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`, "\t", `\t`)
	return `"` + replacer.Replace(value) + `"`
}
