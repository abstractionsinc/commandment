package shelltest_test

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/shelltest"
)

func TestExternalBashClassGeneratorUsesOnlyPublicShellAdapterSurface(t *testing.T) {
	t.Parallel()
	bash := externalBashRuntime(t)
	provider := cli.Shell.Custom("external-bash", externalBashClassGenerator)
	app := externalCompletionApplication(t)
	source, err := app.ShellCompletionScript(provider)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		`records\n`, `shell\tbash`, `input\tuncertain`, `matching\tprefix`,
		`native-capability\tpaths`, `native-capability\tfiles`, `native-capability\tdirectories`,
		`native-capability\texecutables`, `native-capability\tusers`, `native-capability\thosts`,
		`native-capability\tenvironment-variables`,
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("external Bash-class adapter missing shell-adapter capability %q", required)
		}
	}
	got := shelltest.Complete(t, app, externalBashTarget(bash, provider), shelltest.Input{
		BeforeCursor: "alienctl re",
	})
	if !strings.Contains(got, "remote") {
		t.Fatalf("settled external Bash-class screen does not contain remote: %q", got)
	}
}

func TestExternalCustomProviderDrivesRealShellAdapterProcessRoundTrip(t *testing.T) {
	t.Parallel()
	bash := externalBashRuntime(t)
	provider := cli.Shell.Custom("xonsh", externalXonshClassGenerator)
	app := externalCompletionApplication(t)
	source, err := app.ShellCompletionScript(provider)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{`records\n`, `shell\txonsh`, `input\tuncertain`, `matching\tprefix`} {
		if !strings.Contains(source, required) {
			t.Fatalf("external Xonsh-class adapter missing shell-adapter material %q", required)
		}
	}
	got := shelltest.Complete(t, app, externalBashTarget(bash, provider), shelltest.Input{
		BeforeCursor: "alienctl re",
	})
	if !strings.Contains(got, "remote") {
		t.Fatalf("settled external custom-provider screen does not contain remote: %q", got)
	}
}

func externalBashRuntime(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("portable package-external shell adapter proof requires Bash")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skipf("bash unavailable: %v", err)
	}
	return bash
}

func externalCompletionApplication(t *testing.T) cli.Application {
	t.Helper()
	app, err := cli.Compile(cli.Definition[struct{}]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "remote",
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func externalBashTarget(executable string, provider cli.ShellCompletionProvider) shelltest.Target[string] {
	runtimeSpec := shelltest.Runtime{
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
			return fields[0], nil
		},
	}
	target := shelltest.Target[string]{
		Provider: provider,
		Runtime:  runtimeSpec,
		Prepare: func(setup shelltest.Setup) error {
			integration := setup.IntegrationFile("completion.bash")
			rc := setup.File("bashrc", "source "+externalShellQuote(integration)+"\n")
			setup.Env("SHELL", "/bin/sh")
			setup.Args("--noprofile", "--rcfile", rc, "-i")
			return nil
		},
		Interpret: func(observation shelltest.Observation) (string, error) {
			if observation.Version() == "" {
				return "", errors.New("missing verified Bash runtime version")
			}
			lines := observation.ScreenLines()
			if lines == nil {
				return "", errors.New("settled Bash screen unavailable")
			}
			return strings.Join(lines, "\n"), nil
		},
	}
	return target
}

func externalBashClassGenerator(generation cli.ShellCompletionGeneration) (string, error) {
	return externalAdapterSource(generation, "bash", true)
}

func externalXonshClassGenerator(generation cli.ShellCompletionGeneration) (string, error) {
	return externalAdapterSource(generation, "xonsh", false)
}

func externalAdapterSource(generation cli.ShellCompletionGeneration, shell string, nativeCapabilities bool) (string, error) {
	if generation.ApplicationName == "" || generation.ActivationEnvironment == "" || generation.ActivationValue == "" {
		return "", errors.New("incomplete shell-completion generation context")
	}
	native := ""
	if nativeCapabilities {
		native = `native-capability\tpaths\n` +
			`native-capability\tfiles\n` +
			`native-capability\tdirectories\n` +
			`native-capability\texecutables\n` +
			`native-capability\tusers\n` +
			`native-capability\thosts\n` +
			`native-capability\tenvironment-variables\n`
	}
	request := `records\n` +
		`response-encoding\trecords\n` +
		`shell\t` + shell + `\t1.0\n` +
		`input\tuncertain\t0\t0\t2\tobserved-word-bytes\n` +
		`word\t0\tre\tlexical\n` +
		`matching\tprefix\n` + native + `end\n`
	return fmt.Sprintf(`__external_complete() {
    local request response kind index value replacement spacing
    request=$'%s'
    response=$(printf '%%s' "$request" | env %s=%q %q)
    COMPREPLY=()
    while IFS=$'\t' read -r kind index value replacement spacing; do
        if [[ $kind == item ]]; then COMPREPLY+=("$value"); fi
    done <<< "$response"
}
complete -F __external_complete -- %s
`, request, generation.ActivationEnvironment, generation.ActivationValue, generation.ApplicationName, externalShellQuote(generation.ApplicationName)), nil
}

func externalShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
