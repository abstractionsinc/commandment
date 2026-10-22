package conformance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

func shellCompletionComplete(t *testing.T, app cli.Application, request cli.CompletionRequest) (cli.Completion, error) {
	t.Helper()
	return clitest.Complete(t.Context(), app, request).Completion()
}

type shellCompletionCountingSignalSource struct {
	open int
}

func (source *shellCompletionCountingSignalSource) Source() cli.SignalSource[string] {
	signalSource := cli.SignalSource[string]{
		Signals: []cli.SignalDescriptor[string]{{
			Signal:   "counted",
			Fidelity: cli.SignalSourceDeliveryFidelityExact,
		}},
		Open: func(_ context.Context) (cli.SignalStream[string], error) {
			source.open++
			stream := cli.SignalStream[string]{
				Receive: func(ctx context.Context) (cli.SignalNotification[string], error) {
					<-ctx.Done()
					return cli.SignalNotification[string]{}, ctx.Err()
				},
			}
			return stream, nil
		},
	}
	return signalSource
}

const shellCompletionCompatibilityBashName = "bash"

func TestShellCompletionGeneratedAdaptersUseCurrentShellAdapter(t *testing.T) {
	t.Parallel()
	app, err := cli.Compile(cli.Definition[struct{}]{
		Name: "alienctl",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, integration := range []cli.ShellCompletionProvider{
		cli.Shell.Bash, cli.Shell.Zsh,
		cli.Shell.Fish, cli.Shell.PowerShell,
		cli.Shell.Elvish, cli.Shell.Nushell,
	} {
		script, err := app.ShellCompletionScript(integration)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(script, "__complete") || strings.Contains(script, "COMMANDMENT_COMPLETION") {
			t.Fatalf("script preserves superseded completion route/runtime")
		}
		if !strings.Contains(script, "_COMPLETE") {
			t.Fatalf("script missing application-derived activation")
		}
	}
}

func completionScriptForTest(t *testing.T, integration cli.ShellCompletionProvider) string {
	t.Helper()
	app, err := cli.Compile(cli.Definition[struct{}]{
		Name: "alienctl",
	})
	if err != nil {
		t.Fatal(err)
	}
	script, err := app.ShellCompletionScript(integration)
	if err != nil {
		t.Fatalf("ShellCompletionScript() error = %v", err)
	}
	return script
}

func TestShellCompletionBuiltinsSkipUnrepresentableMultiEntryReplacement(t *testing.T) {
	t.Parallel()

	checks := []struct {
		name     string
		provider cli.ShellCompletionProvider
		marker   string
	}{
		{
			name:     shellCompletionCompatibilityBashName,
			provider: cli.Shell.Bash,
			marker:   "((count == 1)) || continue",
		},
		{
			name:     "zsh",
			provider: cli.Shell.Zsh,
			marker:   "((count == 1)) || continue",
		},
		{
			name:     "fish",
			provider: cli.Shell.Fish,
			marker:   `-eq 1; or continue`,
		},
		{
			name:     "powershell",
			provider: cli.Shell.PowerShell,
			marker:   `.Count -ne 1) { continue }`,
		},
		{
			name:     "elvish",
			provider: cli.Shell.Elvish,
			marker:   `(!= (count $item[replacement]) 1) { continue }`,
		},
		{
			name:     "nushell",
			provider: cli.Shell.Nushell,
			marker:   `($replacement | length) != 1 { continue }`,
		},
	}
	app, err := cli.Compile(cli.Definition[struct{}]{
		Name: "alienctl",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			t.Parallel()
			script, err := app.ShellCompletionScript(check.provider)
			if err != nil {
				t.Fatalf("generate %s: %v", check.name, err)
			}
			if !strings.Contains(script, check.marker) {
				t.Fatalf("%s adapter does not skip unrepresentable multi-entry replacement", check.name)
			}
		})
	}
}

func TestFishCompletionScriptPreservesCursorOwnershipBoundaries(t *testing.T) {
	t.Parallel()
	script := completionScriptForTest(t, cli.Shell.Fish)
	steps := []string{
		`if test -n "$raw_current"`,
		`set -q all[$current_index]; or return 0`,
		`if test $major -ge 4`,
		`set -l expanded_current (commandline --current-token --tokens-expanded)`,
		`test (count $expanded_current) -eq 1; or return 0`,
		`test "$expanded_current[1]" = "$all[$current_index]"; or return 0`,
		`set current $expanded_current[1]`,
		`else`,
		`set current $all[$current_index]`,
	}
	offset := 0
	for _, step := range steps {
		index := strings.Index(script[offset:], step)
		if index < 0 {
			t.Fatalf("generated Fish script missing cursor-ownership step %q", step)
		}
		offset += index + len(step)
	}
	if count := strings.Count(script, "commandline --current-token --tokens-expanded"); count != 1 {
		t.Fatalf("expanded-current query count = %d, want 1", count)
	}
}

func TestPowerShellCompletionPreservesProvenHandledZeroBand(t *testing.T) {
	t.Parallel()
	script := completionScriptForTest(t, cli.Shell.PowerShell)
	for _, marker := range []string{
		"if (-not $failed -and $items.Count -eq 0 -and",
		"$PSVersionTable.PSVersion.Major -eq 7 -and $PSVersionTable.PSVersion.Minor -ge 6)",
		"return ''",
	} {
		if !strings.Contains(script, marker) {
			t.Fatalf("generated PowerShell script missing handled-zero marker %q", marker)
		}
	}
}

func TestElvishDefaultGenerationHasSingleActivationDeclaration(t *testing.T) {
	t.Parallel()
	script := completionScriptForTest(t, cli.Shell.Elvish)
	if strings.Contains(script, "var var ") {
		t.Fatal("generated current Elvish script contains duplicated var declaration")
	}
	if !strings.Contains(script, "_activation = '") {
		t.Fatal("generated current Elvish script lacks activation declaration")
	}
}

func TestElvishHistoricalGenerationPreservesSyntaxBands(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		version string
		want    []string
		forbid  []string
	}{
		{
			version: "0.11.0",
			want: []string{
				"$edit:arg-completer", "[@all-arguments]{", "$all-arguments[1:]",
				"} except _ {", "printf 'records\\n'",
			},
			forbid: []string{"edit:completion:arg-completer", "var ", "{|"},
		},
		{
			version: "0.12.0",
			want: []string{
				"$edit:completion:arg-completer", "[@all-arguments]{", "$all-arguments[1:]",
				"} except _ {", "printf 'records\\n'",
			},
			forbid: []string{"var ", "{|"},
		},
	} {
		app, err := cli.Compile(cli.Definition[struct{}]{
			Name: "alienctl",
		})
		if err != nil {
			t.Fatal(err)
		}
		script, err := app.ShellCompletionScriptForVersion(cli.Shell.Elvish, test.version)
		if err != nil {
			t.Fatal(err)
		}
		for _, marker := range test.want {
			if !strings.Contains(script, marker) {
				t.Fatalf("Elvish %s script missing %q", test.version, marker)
			}
		}
		for _, marker := range test.forbid {
			if strings.Contains(script, marker) {
				t.Fatalf("Elvish %s script contains incompatible syntax %q", test.version, marker)
			}
		}
	}
}

func TestZshCompletionPreserves401CompatibleRegistration(t *testing.T) {
	t.Parallel()
	script := completionScriptForTest(t, cli.Shell.Zsh)
	for _, marker := range []string{
		"item_present[slot]=true",
		"[[ ${item_present[index]-false} == true ]] || continue",
		"if ! whence compdef >/dev/null 2>&1; then",
		"return 0",
		"records\\nresponse-encoding\\trecords\\n",
		`request_path=${TMPDIR:-/tmp}/completion-request-$$-$RANDOM`,
		`response_path=${TMPDIR:-/tmp}/completion-response-$$-$RANDOM`,
		`< "$request_path" >| "$response_path"`,
		`done < "$response_path"`,
	} {
		if !strings.Contains(script, marker) {
			t.Fatalf("generated Zsh script missing floor/shell-adapter marker %q", marker)
		}
	}
	if strings.Contains(script, "${+item_value[index]}") {
		t.Fatal("generated Zsh script uses parameter-existence syntax excluded from the 4.0.1 path")
	}
	if strings.Contains(script, `response=$(print -rn -- "$request" | env `) {
		t.Fatal("generated Zsh script uses interactive-ZLE-unsafe pipeline command substitution transport")
	}
}

func TestNushellCompletionGenerationBandsRemainDistinct(t *testing.T) {
	t.Parallel()
	app, err := cli.Compile(cli.Definition[struct{}]{
		Name: "alienctl",
	})
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := app.ShellCompletionScriptForVersion(cli.Shell.Nushell, "0.72.0")
	if err != nil {
		t.Fatal(err)
	}
	modern, err := app.ShellCompletionScriptForVersion(cli.Shell.Nushell, "0.115.1")
	if err != nil {
		t.Fatal(err)
	}
	if legacy == modern {
		t.Fatal("legacy and modern Nushell artifacts unexpectedly identical")
	}
	if strings.Contains(legacy, "...$__completion_") {
		t.Fatal("legacy artifact preserves transport argv")
	}
}

func TestNushellLegacyCompletionMatchingProfilesRemainBandSpecific(t *testing.T) {
	t.Parallel()
	app, err := cli.Compile(cli.Definition[struct{}]{
		Name: "alienctl",
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		target string
		want   string
	}{
		{
			name:   "legacy-client-filtered",
			target: "0.72.0",
			want:   `matching\tclient`,
		},
		{
			name:   "legacy-null-prefix",
			target: "0.102.0",
			want:   `matching\tprefix`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			script, err := app.ShellCompletionScriptForVersion(cli.Shell.Nushell, test.target)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(script, test.want) {
				t.Fatalf("Nushell %s artifact missing %q", test.target, test.want)
			}
		})
	}
}

const (
	shellCompletionTestApplicationName = "alienctl"
	shellCompletionTestCommandName     = "completion"
	shellCompletionTestXonshName       = "xonsh"
	shellCompletionTestArtifact        = "artifact"
	shellCompletionTestBashName        = "bash"
)

func TestShellCompletionGenerationUsesCurrentPrivateActivation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		applicationName string
		wantEnvironment string
	}{
		{
			applicationName: "alienctl",
			wantEnvironment: "_ALIENCTL_73D461C4BC2C89FB76D20703910821E4_COMPLETE",
		},
		{
			applicationName: "!!!",
			wantEnvironment: "_APP_E84C538E7FE250730EF62DE220C40DFA_COMPLETE",
		},
	}

	for _, test := range tests {
		t.Run(test.applicationName, func(t *testing.T) {
			t.Parallel()

			var generation cli.ShellCompletionGeneration
			provider := cli.Shell.Custom("probe", func(value cli.ShellCompletionGeneration) (string, error) {
				generation = value
				return "probe", nil
			})
			app, err := cli.Compile(cli.Definition[struct{}]{
				Name: test.applicationName,
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := app.ShellCompletionScript(provider); err != nil {
				t.Fatal(err)
			}
			if generation.ActivationEnvironment != test.wantEnvironment {
				t.Fatalf("activation environment = %q, want %q", generation.ActivationEnvironment, test.wantEnvironment)
			}
			if generation.ActivationValue != "1" {
				t.Fatalf("activation value = %q, want 1", generation.ActivationValue)
			}
		})
	}
}

func TestApplicationShellCompletionScriptGeneratesAllBuiltIns(t *testing.T) {
	t.Parallel()
	app, err := cli.Compile(cli.Definition[struct{}]{
		Name: shellCompletionTestApplicationName,
	})
	if err != nil {
		t.Fatal(err)
	}
	integrations := []cli.ShellCompletionProvider{
		cli.Shell.Bash,
		cli.Shell.Zsh,
		cli.Shell.Fish,
		cli.Shell.PowerShell,
		cli.Shell.Elvish,
		cli.Shell.Nushell,
	}
	for _, integration := range integrations {
		script, err := app.ShellCompletionScript(integration)
		if err != nil {
			t.Fatalf("ShellCompletionScript() error = %v", err)
		}
		if script == "" {
			t.Fatal("ShellCompletionScript() returned empty script")
		}
		if strings.Contains(strings.ToLower(script), "commandment") {
			t.Fatalf("generated script leaks framework branding: %q", script[:min(len(script), 160)])
		}
		if !strings.Contains(script, shellCompletionTestApplicationName) {
			t.Fatal("generated script does not contain application identity")
		}
	}
}

func TestApplicationShellCompletionScriptTargetVersions(t *testing.T) {
	t.Parallel()
	app, err := cli.Compile(cli.Definition[struct{}]{
		Name: shellCompletionTestApplicationName,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		provider cli.ShellCompletionProvider
		version  string
	}{
		{
			provider: cli.Shell.Elvish,
			version:  "0.11.0",
		},
		{
			provider: cli.Shell.Elvish,
			version:  "0.12.0",
		},
		{
			provider: cli.Shell.Nushell,
			version:  "0.72.0",
		},
		{
			provider: cli.Shell.Nushell,
			version:  "0.108.0",
		},
	} {
		if _, err := app.ShellCompletionScriptForVersion(test.provider, test.version); err != nil {
			t.Fatalf("targeted generation error = %v", err)
		}
	}
}

func TestApplicationShellCompletionScriptRejectsUnsupportedTarget(t *testing.T) {
	t.Parallel()
	app, err := cli.Compile(cli.Definition[struct{}]{
		Name: shellCompletionTestApplicationName,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.ShellCompletionScriptForVersion(cli.Shell.Nushell, "0.1.0"); err == nil {
		t.Fatal("unsupported Nushell target unexpectedly succeeded")
	}
}

func TestShellCompletionPublicCatalogContract(t *testing.T) {
	t.Parallel()
	customCalls := 0
	custom := cli.Shell.Custom(
		shellCompletionTestXonshName,
		func(generation cli.ShellCompletionGeneration) (string, error) {
			customCalls++
			if generation.ApplicationName != shellCompletionTestApplicationName ||
				generation.ActivationEnvironment == "" ||
				generation.ActivationValue == "" {
				t.Fatalf("generation = %#v", generation)
			}
			return "custom:" + generation.TargetVersion, nil
		},
	)

	app, err := cli.Compile(cli.Definition[struct{}]{
		Name: shellCompletionTestApplicationName,
		ShellCompletion: cli.ShellCompletion{
			Shells: []cli.ShellCompletionProvider{custom},
		},
	})
	if err != nil {
		t.Fatalf("Compile custom catalog: %v", err)
	}
	var output strings.Builder
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{shellCompletionTestCommandName, shellCompletionTestXonshName, "--shell-version", "0.19.3"},
		Stdout: &output,
	}); err != nil {
		t.Fatalf("Run exporter: %v", err)
	}
	if got := output.String(); got != "custom:0.19.3" {
		t.Fatalf("exporter stdout = %q", got)
	}
	if customCalls != 1 {
		t.Fatalf("custom generator calls = %d, want 1", customCalls)
	}

	// Direct projection is independent of catalog membership.
	script, err := app.ShellCompletionScript(cli.Shell.Bash)
	if err != nil || !strings.Contains(script, shellCompletionTestApplicationName) {
		t.Fatalf("direct non-catalog built-in generation = %q, %v", script, err)
	}

	disabled, err := cli.Compile(
		cli.Definition[struct{}]{
			Name: shellCompletionTestApplicationName,
			ShellCompletion: cli.ShellCompletion{
				Disabled: true,
			},
		},
	)
	if err != nil {
		t.Fatalf("Compile disabled: %v", err)
	}
	if _, err := disabled.ShellCompletionScript(custom); err != nil {
		t.Fatalf("direct generation while exporter disabled: %v", err)
	}
	if err := disabled.Run(
		t.Context(),
		cli.RunRequest{
			Args: []string{shellCompletionTestCommandName, shellCompletionTestXonshName},
		},
	); err == nil {
		t.Fatal("disabled exporter unexpectedly remained invocable")
	}
}

func TestShellCompletionCatalogValidation(t *testing.T) {
	t.Parallel()
	valid := cli.Shell.Custom(
		shellCompletionTestXonshName,
		func(cli.ShellCompletionGeneration) (string, error) { return "x", nil },
	)
	cases := []struct {
		name   string
		config cli.ShellCompletion
	}{
		{
			name: "disabled-with-catalog",
			config: cli.ShellCompletion{
				Disabled: true,
				Shells:   []cli.ShellCompletionProvider{valid},
			},
		},
		{
			name: "empty-catalog",
			config: cli.ShellCompletion{
				Shells: []cli.ShellCompletionProvider{},
			},
		},
		{
			name: "nil-provider",
			config: cli.ShellCompletion{
				Shells: []cli.ShellCompletionProvider{nil},
			},
		},
		{
			name: "duplicate-name",
			config: cli.ShellCompletion{
				Shells: []cli.ShellCompletionProvider{valid, valid},
			},
		},
		{
			name: "invalid-name",
			config: cli.ShellCompletion{
				Shells: []cli.ShellCompletionProvider{
					cli.Shell.Custom("-xonsh", func(cli.ShellCompletionGeneration) (string, error) { return "x", nil }),
				},
			},
		},
		{
			name: "whitespace-name",
			config: cli.ShellCompletion{
				Shells: []cli.ShellCompletionProvider{
					cli.Shell.Custom("xon sh", func(cli.ShellCompletionGeneration) (string, error) { return "x", nil }),
				},
			},
		},
		{
			name: "nil-generator",
			config: cli.ShellCompletion{
				Shells: []cli.ShellCompletionProvider{cli.Shell.Custom(shellCompletionTestXonshName, nil)},
			},
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := cli.Compile(
				cli.Definition[struct{}]{
					Name:            shellCompletionTestApplicationName,
					ShellCompletion: test.config,
				},
			); err == nil {
				t.Fatal("Compile unexpectedly accepted invalid shell-completion catalog")
			}
		})
	}
}

func TestShellCompletionCatalogNameContract(t *testing.T) {
	t.Parallel()
	generator := func(cli.ShellCompletionGeneration) (string, error) { return shellCompletionTestArtifact, nil }

	for _, test := range []struct {
		name string
		want bool
	}{
		{
			name: shellCompletionTestXonshName,
			want: true,
		},
		{
			name: "βash",
			want: true,
		},
		{
			name: "BASH",
			want: true,
		},
		{
			name: "é",
			want: true,
		},
		{
			name: "e\u0301",
			want: true,
		},
		{
			name: "",
			want: false,
		},
		{
			name: "-bash",
			want: false,
		},
		{
			name: "has space",
			want: false,
		},
		{
			name: "has\t tab",
			want: false,
		},
		{
			name: "has\nnewline",
			want: false,
		},
		{
			name: "has\rreturn",
			want: false,
		},
		{
			name: "has\x00nul",
			want: false,
		},
		{
			name: "has\u00a0space",
			want: false,
		},
		{
			name: "has\u0085control",
			want: false,
		},
		{
			name: string([]byte{0xff}),
			want: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := cli.Compile(cli.Definition[struct{}]{
				Name: shellCompletionTestApplicationName,
				ShellCompletion: cli.ShellCompletion{
					Shells: []cli.ShellCompletionProvider{cli.Shell.Custom(test.name, generator)},
				},
			})
			if test.want && err != nil {
				t.Fatalf("Compile rejected valid opaque provider name %q: %v", test.name, err)
			}
			if !test.want && err == nil {
				t.Fatalf("Compile accepted invalid provider name %q", test.name)
			}
		})
	}

	// Catalog names are exact opaque identity. Commandment does not case-fold or
	// Unicode-normalize provider names.
	exact, err := cli.Compile(cli.Definition[struct{}]{
		Name: shellCompletionTestApplicationName,
		ShellCompletion: cli.ShellCompletion{
			Shells: []cli.ShellCompletionProvider{
				cli.Shell.Custom(shellCompletionTestBashName, generator),
				cli.Shell.Custom("BASH", generator),
				cli.Shell.Custom("é", generator),
				cli.Shell.Custom("e\u0301", generator),
			},
		},
	})
	if err != nil {
		t.Fatalf("Compile exact opaque names: %v", err)
	}
	if got, want := strings.Join(shellCompletionCatalogNames(t, exact), ","), "bash,BASH,é,e\u0301"; got != want {
		t.Fatalf("opaque catalog identity/order = %q, want %q", got, want)
	}
}

func TestShellCompletionBuiltInAndCustomSameNameIdentityAndReplacement(t *testing.T) {
	t.Parallel()
	customBash := cli.Shell.Custom(shellCompletionTestBashName, func(cli.ShellCompletionGeneration) (string, error) {
		return "custom-bash", nil
	})

	if _, err := cli.Compile(cli.Definition[struct{}]{
		Name: shellCompletionTestApplicationName,
		ShellCompletion: cli.ShellCompletion{
			Shells: []cli.ShellCompletionProvider{
				cli.Shell.Bash,
				customBash,
			},
		},
	}); err == nil {
		t.Fatal("built-in bash plus custom bash unexpectedly compiled as distinct catalog identities")
	}

	replacement, err := cli.Compile(cli.Definition[struct{}]{
		Name: shellCompletionTestApplicationName,
		ShellCompletion: cli.ShellCompletion{
			Shells: []cli.ShellCompletionProvider{customBash},
		},
	})
	if err != nil {
		t.Fatalf("Compile custom bash replacement: %v", err)
	}
	if got := strings.Join(shellCompletionCatalogNames(t, replacement), ","); got != shellCompletionTestBashName {
		t.Fatalf("replacement catalog = %q, want bash", got)
	}
	var output strings.Builder
	if err := replacement.Run(
		t.Context(),
		cli.RunRequest{
			Args:   []string{shellCompletionTestCommandName, shellCompletionTestBashName},
			Stdout: &output,
		},
	); err != nil {
		t.Fatalf("Run custom bash replacement: %v", err)
	}
	if got := output.String(); got != "custom-bash" {
		t.Fatalf("custom bash replacement output = %q", got)
	}
}

func TestShellCompletionTargetVersionContract(t *testing.T) {
	t.Parallel()
	app, err := cli.Compile(cli.Definition[struct{}]{
		Name: shellCompletionTestApplicationName,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.ShellCompletionScriptForVersion(cli.Shell.Bash, "5.2"); err == nil {
		t.Fatal("non-version-sensitive built-in accepted target version")
	}
	if _, err := app.ShellCompletionScriptForVersion(cli.Shell.Elvish, ""); err == nil {
		t.Fatal("targeted generation accepted empty version")
	}
	var got cli.ShellCompletionGeneration
	custom := cli.Shell.Custom(
		shellCompletionTestXonshName,
		func(generation cli.ShellCompletionGeneration) (string, error) {
			got = generation
			return "x", nil
		},
	)
	if _, err := app.ShellCompletionScriptForVersion(custom, "verbatim target"); err != nil {
		t.Fatal(err)
	}
	if got.TargetVersion != "verbatim target" {
		t.Fatalf("custom TargetVersion = %q", got.TargetVersion)
	}
}

func TestShellCompletionExporterBypassesBusinessExecution(t *testing.T) {
	t.Parallel()
	type config struct{ Required string }
	var output strings.Builder
	middlewareCalls := 0
	handlerCalls := 0
	ruleCalls := 0
	noticeCalls := 0
	factSupplyCalls := 0
	source := &shellCompletionCountingSignalSource{}
	custom := cli.Shell.Custom(
		shellCompletionTestXonshName,
		func(cli.ShellCompletionGeneration) (string, error) { return shellCompletionTestArtifact, nil },
	)

	definition := cli.Definition[config]{
		Name: shellCompletionTestApplicationName,
		ShellCompletion: cli.ShellCompletion{
			Shells: []cli.ShellCompletionProvider{custom},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Required)
			root.Rule.Require(value)
			root.Rule.Check("must-not-run", func(context.Context, *config, cli.Dependencies) (bool, error) {
				ruleCalls++
				return false, fmt.Errorf("business policy ran: %w", context.Canceled)
			})
			registered := root.Signals.Source(source.Source()).Signal("counted")
			root.Signals.Policy(
				root.Signals.Match.On(registered).During(cli.SignalPhaseExecution).Each(),
				root.Signals.Action.Ignore(),
			)
			root.Advisory.Always(
				cli.Advisory{
					Code:     "must-not-notify",
					Severity: cli.NoticeSeverityWarning,
					Message:  "must not notify",
				},
			)
		},
		Middleware: []cli.Middleware[config]{func(ctx context.Context, _ *config, next cli.MiddlewareNext) error {
			middlewareCalls++
			return next(ctx)
		}},
		Run: func(context.Context, *config) error { handlerCalls++; return nil },
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	runRequest := cli.RunRequest{
		Args:   []string{shellCompletionTestCommandName, shellCompletionTestXonshName},
		Stdout: &output,
		Facts: []cli.FactProvider{cli.FactsFor[config](func(context.Context, *cli.FactSupply[config]) error {
			factSupplyCalls++
			return nil
		})},
		Notify: func(context.Context, cli.Notice) error { noticeCalls++; return nil },
	}
	if err := app.Run(t.Context(), runRequest); err != nil {
		t.Fatalf("Run exporter: %v", err)
	}
	if output.String() != shellCompletionTestArtifact {
		t.Fatalf("stdout = %q", output.String())
	}
	if middlewareCalls != 0 || handlerCalls != 0 || ruleCalls != 0 || noticeCalls != 0 || factSupplyCalls != 0 ||
		source.open != 0 {
		t.Fatalf(
			"exporter entered business execution: middleware=%d handler=%d rules=%d notices=%d facts=%d signals=%d",
			middlewareCalls,
			handlerCalls,
			ruleCalls,
			noticeCalls,
			factSupplyCalls,
			source.open,
		)
	}
}

func shellCompletionCatalogNames(t *testing.T, app cli.Application) []string {
	t.Helper()
	schema := clitest.Schema(app).Schema()
	var command cli.CommandSchema
	found := false
	for _, id := range schema.CommandOrder {
		candidate := schema.Commands[id]
		if candidate.Name == shellCompletionTestCommandName {
			command, found = candidate, true
			break
		}
	}
	if !found || len(command.ArgumentForms) != 1 {
		t.Fatalf("completion command/form = %#v", command)
	}
	form := schema.ArgumentForms[command.ArgumentForms[0]]
	if len(form.Terms) != 1 {
		t.Fatalf("completion argument form = %#v", form)
	}
	destination := schema.ArgumentTerms[form.Terms[0]].Destination
	for _, domain := range schema.ValueDomains {
		for _, use := range domain.Uses {
			if use.Destination != destination {
				continue
			}
			names := make([]string, 0, len(domain.Entries))
			for _, entry := range domain.Entries {
				names = append(names, schema.ValueDomainEntries[entry].Name)
			}
			return names
		}
	}
	t.Fatal("completion shell catalog domain not found")
	return nil
}

func TestShellCompletionCatalogDefaultAndSubsetPreserveIdentityAndOrder(t *testing.T) {
	t.Parallel()
	defaultApp, err := cli.Compile(cli.Definition[struct{}]{
		Name: shellCompletionTestApplicationName,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(
		shellCompletionCatalogNames(t, defaultApp),
		",",
	), "bash,zsh,fish,powershell,elvish,nushell"; got != want {
		t.Fatalf("default shell catalog order = %q, want %q", got, want)
	}

	xonsh := cli.Shell.Custom(
		shellCompletionTestXonshName,
		func(cli.ShellCompletionGeneration) (string, error) { return "x", nil },
	)
	subset, err := cli.Compile(cli.Definition[struct{}]{
		Name: shellCompletionTestApplicationName,
		ShellCompletion: cli.ShellCompletion{
			Shells: []cli.ShellCompletionProvider{cli.Shell.Fish, xonsh, cli.Shell.Bash},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(shellCompletionCatalogNames(t, subset), ","), "fish,xonsh,bash"; got != want {
		t.Fatalf("subset shell catalog order = %q, want %q", got, want)
	}
}

func TestShellCompletionIntrinsicIsVisibleToStaticAndCompletionProjections(t *testing.T) {
	t.Parallel()
	app, err := cli.Compile(cli.Definition[struct{}]{
		Name: shellCompletionTestApplicationName,
	})
	if err != nil {
		t.Fatal(err)
	}
	schema := clitest.Schema(app).Schema()
	found := false
	for _, id := range schema.CommandOrder {
		if schema.Commands[id].Name == shellCompletionTestCommandName {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("static schema does not contain completion intrinsic")
	}
	completionRequest := cli.CompletionRequest{
		Args: []string{"comp"},
		Cursor: cli.CompletionCursor{
			Argument: 0,
			Byte:     4,
		},
		Visibility: cli.CompletionVisibilityPublic,
		Client: cli.CompletionClient{
			Match: cli.CompletionMatchOptions{
				Mode: cli.CompletionMatchModePrefix,
				Case: cli.CompletionCaseSmart,
			},
		},
	}
	completion, err := shellCompletionComplete(t, app, completionRequest)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	for _, item := range completion.Items {
		if item.Value == shellCompletionTestCommandName {
			return
		}
	}
	t.Fatalf("completion intrinsic absent from public completion items: %#v", completion.Items)
}
