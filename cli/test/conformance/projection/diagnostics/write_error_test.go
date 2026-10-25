package conformance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

type diagnosticPresentationModeConfig struct{ Mode string }

type diagnosticRichApplicationError struct{}

func (diagnosticRichApplicationError) Error() string { return "APP" }

func (diagnosticRichApplicationError) Format(state fmt.State, verb rune) {
	if verb == 'v' && state.Flag('+') {
		_, _ = fmt.Fprint(state, "APP-RICH")
		return
	}
	_, _ = fmt.Fprint(state, "APP")
}

type diagnosticShortWriter struct{ published strings.Builder }

func (writer *diagnosticShortWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	n := len(p) - 1
	_, _ = writer.published.Write(p[:n])
	return n, nil
}

func TestWriteErrorOwnsCommandmentHumanPresentationWithoutTakingApplicationOwnership(t *testing.T) {
	t.Parallel()
	diagnosticDefinition := cli.Definition[struct{}]{
		Name: "alienctl",
		Run:  func(context.Context, *struct{}) error { return nil },
	}
	diagnosticApp := clitest.Compile(t, diagnosticDefinition)
	diagnosticRequest := cli.RunRequest{
		Args: []string{"--definitely-unknown"},
	}
	diagnosticErr := diagnosticApp.Run(t.Context(), diagnosticRequest)
	if diagnosticErr == nil {
		t.Fatal("Run() error = nil, want Commandment Diagnostic")
	}
	if got := fmt.Sprintf("%v", diagnosticErr); got != "commandment: unknown-flag" {
		t.Fatalf("Diagnostic %%v = %q, want concise ordinary identity", got)
	}
	debug := fmt.Sprintf("%+v", diagnosticErr)
	for _, want := range []string{"Commandment diagnostic [unknown-flag]", "command id:"} {
		if !strings.Contains(debug, want) {
			t.Fatalf("Diagnostic %%+v missing %q:\n%s", want, debug)
		}
	}
	var rendered strings.Builder
	if err := cli.WriteError(&rendered, diagnosticErr); err != nil {
		t.Fatalf("WriteError() error = %v", err)
	}
	if got, want := rendered.String(), "alienctl: unknown flag --definitely-unknown\n"; got != want {
		t.Fatalf("WriteError() = %q, want %q", got, want)
	}
	for _, forbidden := range []string{"Commandment diagnostic", "command id:", "producer kind:", "resolution tier:"} {
		if strings.Contains(rendered.String(), forbidden) {
			t.Fatalf("end-user presentation leaked debug detail %q: %s", forbidden, rendered.String())
		}
	}
	if !strings.HasSuffix(rendered.String(), "\n") || strings.HasSuffix(rendered.String(), "\n\n") {
		t.Fatalf("WriteError() line termination = %q", rendered.String())
	}

	applicationCause := errors.New("opaque application failure\x1b[31m")
	applicationDefinition := cli.Definition[struct{}]{
		Name: "alienctl",
		Run:  func(context.Context, *struct{}) error { return applicationCause },
	}
	applicationApp := clitest.Compile(t, applicationDefinition)
	applicationErr := applicationApp.Run(t.Context(), cli.RunRequest{})
	if applicationErr != applicationCause { //nolint:err113,errorlint // Run must preserve an ordinary application error as the original object.
		t.Fatalf("Run() ordinary error = %T %v, want original application error", applicationErr, applicationErr)
	}
	if got := fmt.Sprintf("%+v", applicationErr); got != applicationCause.Error() {
		t.Fatalf("Commandment reformatted application-owned error: %q", got)
	}
	rendered.Reset()
	if err := cli.WriteError(&rendered, applicationErr); err != nil {
		t.Fatalf("WriteError(application) error = %v", err)
	}
	if got, want := rendered.String(), applicationCause.Error()+"\n"; got != want {
		t.Fatalf("WriteError(application) = %q, want %q", got, want)
	}
}

func TestWriteErrorOwnershipAndPublicationContract(t *testing.T) {
	t.Parallel()
	definition := cli.Definition[struct{}]{
		Name: "alienctl",
		Run:  func(context.Context, *struct{}) error { return nil },
	}
	app := clitest.Compile(t, definition)
	request := cli.RunRequest{
		Args: []string{"--bad"},
	}
	diagnosticErr := app.Run(t.Context(), request)
	if diagnosticErr == nil {
		t.Fatal("Run() error = nil, want Diagnostic")
	}

	if err := cli.WriteError(nil, nil); err != nil {
		t.Fatalf("WriteError(nil, nil) = %v, want nil", err)
	}
	if err := cli.WriteError(nil, diagnosticErr); err == nil {
		t.Fatal("WriteError(nil, diagnostic) = nil, want publication error")
	}

	var rendered strings.Builder
	if err := cli.WriteError(&rendered, diagnosticRichApplicationError{}); err != nil {
		t.Fatalf("WriteError(application formatter) = %v", err)
	}
	if got, want := rendered.String(), "APP\n"; got != want {
		t.Fatalf("WriteError(application formatter) = %q, want %q", got, want)
	}

	wrapper := fmt.Errorf("outer: %w", diagnosticErr)
	rendered.Reset()
	if err := cli.WriteError(&rendered, wrapper); err != nil {
		t.Fatalf("WriteError(wrapper) = %v", err)
	}
	if got, want := rendered.String(), fmt.Sprintf("%v\n", wrapper); got != want {
		t.Fatalf("WriteError(wrapper) = %q, want %q", got, want)
	}
	if len(cli.Diagnostics(wrapper).Diagnostics) == 0 {
		t.Fatal("Diagnostics(wrapper) lost nested Diagnostic")
	}

	for _, joined := range []error{
		errors.Join(errors.New("app"), diagnosticErr),
		errors.Join(diagnosticErr, errors.New("app")),
	} {
		rendered.Reset()
		if err := cli.WriteError(&rendered, joined); err != nil {
			t.Fatalf("WriteError(external join) = %v", err)
		}
		if got, want := rendered.String(), fmt.Sprintf("%v\n", joined); got != want {
			t.Fatalf("WriteError(external join) = %q, want ordinary external presentation %q", got, want)
		}
		if len(cli.Diagnostics(joined).Diagnostics) == 0 {
			t.Fatal("Diagnostics(external join) lost nested Diagnostic")
		}
	}

	short := &diagnosticShortWriter{}
	if err := cli.WriteError(short, diagnosticErr); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("WriteError(short writer) = %v, want io.ErrShortWrite", err)
	}
	if short.published.Len() == 0 {
		t.Fatal("short writer observed no permitted partial publication")
	}
}

func TestWriteErrorDynamicDomainDiagnosticIncludesOwningCommandWithoutRenderingCauseText(t *testing.T) {
	t.Parallel()
	type config = diagnosticPresentationModeConfig
	cause := errors.New("application-owned domain cause")
	definition := cli.Definition[config]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(context.Context, *config, cli.Dependencies, *cli.ValueDomainBuilder[string]) error {
					return cause
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Flags.ValueFor(mode).Long("mode")
		},
		Run: func(context.Context, *config) error { return nil },
	}
	app := clitest.Compile(t, definition)

	request := cli.RunRequest{
		Args: []string{"--mode", "missing"},
	}
	err := app.Run(t.Context(), request)
	if err == nil || !errors.Is(err, cause) {
		t.Fatalf("Run() error = %v, want preserved dynamic-domain cause", err)
	}
	debug := fmt.Sprintf("%+v", err)
	if !strings.Contains(debug, "Commandment diagnostic [value-domain-provision-failure]") {
		t.Fatalf("Diagnostic %%+v missing debug identity:\n%s", debug)
	}
	if strings.Contains(debug, cause.Error()) {
		t.Fatalf("Diagnostic %%+v leaked application-owned cause: %q", debug)
	}
	var rendered strings.Builder
	if writeErr := cli.WriteError(&rendered, err); writeErr != nil {
		t.Fatalf("WriteError() error = %v", writeErr)
	}
	if got, want := rendered.String(), "alienctl: value domain provision failed [value-domain-provision-failure]\n"; got != want {
		t.Fatalf("dynamic-domain WriteError() = %q, want %q", got, want)
	}
	if strings.Contains(rendered.String(), cause.Error()) {
		t.Fatalf("Commandment rendered application-owned domain cause text: %q", rendered.String())
	}
}

func diagnosticErrorText(t *testing.T, err error) string {
	t.Helper()
	var rendered strings.Builder
	if writeErr := cli.WriteError(&rendered, err); writeErr != nil {
		t.Fatalf("WriteError() error = %v", writeErr)
	}
	return rendered.String()
}

func renderRunError[C any](t *testing.T, definition cli.Definition[C], args ...string) (string, error) {
	t.Helper()
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	request := cli.RunRequest{
		Args: slices.Clone(args),
	}
	err = app.Run(t.Context(), request)
	if err == nil {
		t.Fatal("Run() error = nil, want failure")
	}
	return diagnosticErrorText(t, err), err
}

func TestWriteErrorRendersCanonicalHumanDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("invalid integer", func(t *testing.T) {
		t.Parallel()
		type config struct {
			Range int
			World string
		}
		definition := cli.Definition[config]{
			Name: "alienctl",
			Declare: func(root *cli.CommandBuilder[config]) {
				rangeValue := root.Values.Int(&root.Config.Range)
				root.Flags.ValueFor(rangeValue).Long("range")
				root.Arguments.String(&root.Config.World, "world")
			},
			Run: func(context.Context, *config) error { return nil },
		}
		got, _ := renderRunError(t, definition, "--range", "banana", "mars")
		if want := "alienctl: invalid value \"banana\" for --range: expected an integer\n"; got != want {
			t.Fatalf("WriteError() = %q, want %q", got, want)
		}
	})

	t.Run("rule range", func(t *testing.T) {
		t.Parallel()
		type config struct {
			Range int
			World string
		}
		definition := cli.Definition[config]{
			Name: "alienctl",
			Declare: func(root *cli.CommandBuilder[config]) {
				rangeValue := root.Values.Int(&root.Config.Range)
				root.Flags.ValueFor(rangeValue).Long("range")
				root.Arguments.String(&root.Config.World, "world")
				root.Rule.Between(rangeValue, 1, 10)
			},
			Run: func(context.Context, *config) error { return nil },
		}
		got, _ := renderRunError(t, definition, "--range", "99", "mars")
		if want := "alienctl: --range must be between 1 and 10\n"; got != want {
			t.Fatalf("WriteError() = %q, want %q", got, want)
		}
	})

	t.Run("required flag", func(t *testing.T) {
		t.Parallel()
		type config struct{ Range int }
		definition := cli.Definition[config]{
			Name: "alienctl",
			Declare: func(root *cli.CommandBuilder[config]) {
				rangeValue := root.Values.Int(&root.Config.Range)
				root.Flags.ValueFor(rangeValue).Long("range")
				root.Rule.Require(rangeValue)
			},
			Run: func(context.Context, *config) error { return nil },
		}
		got, _ := renderRunError(t, definition)
		if want := "alienctl: --range is required\n"; got != want {
			t.Fatalf("WriteError() = %q, want %q", got, want)
		}
	})

	t.Run("missing flag operand", func(t *testing.T) {
		t.Parallel()
		type config struct{ Range int }
		definition := cli.Definition[config]{
			Name: "alienctl",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Flags.Int(&root.Config.Range).Long("range")
			},
			Run: func(context.Context, *config) error { return nil },
		}
		got, err := renderRunError(t, definition, "--range")
		if want := "alienctl: flag --range requires a value\n"; got != want {
			t.Fatalf("WriteError() = %q, want %q", got, want)
		}
		if report := cli.Diagnostics(err); len(report.Diagnostics) != 1 {
			t.Fatalf("Diagnostics() count = %d, want 1: %#v", len(report.Diagnostics), report.Diagnostics)
		}
	})

	t.Run("unknown flag", func(t *testing.T) {
		t.Parallel()
		definition := cli.Definition[struct{}]{
			Name: "alienctl",
			Run:  func(context.Context, *struct{}) error { return nil },
		}
		got, _ := renderRunError(t, definition, "--rang")
		if want := "alienctl: unknown flag --rang\n"; got != want {
			t.Fatalf("WriteError() = %q, want %q", got, want)
		}
	})

	t.Run("unknown command", func(t *testing.T) {
		t.Parallel()
		definition := cli.Definition[struct{}]{
			Name: "alienctl",
			Declare: func(root *cli.CommandBuilder[struct{}]) {
				scan := cli.Command[struct{}]{
					Name: "scan",
					Run:  func(context.Context, *struct{}) error { return nil },
				}
				root.Subcommands.Command(scan)
			},
		}
		got, _ := renderRunError(t, definition, "scna")
		if want := "alienctl: unknown command \"scna\"\n"; got != want {
			t.Fatalf("WriteError() = %q, want %q", got, want)
		}
	})

	t.Run("missing positional", func(t *testing.T) {
		t.Parallel()
		type config struct{ World string }
		definition := cli.Definition[config]{
			Name: "alienctl",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Arguments.String(&root.Config.World, "world")
			},
			Run: func(context.Context, *config) error { return nil },
		}
		got, _ := renderRunError(t, definition)
		if want := "alienctl: missing required argument <world>\n"; got != want {
			t.Fatalf("WriteError() = %q, want %q", got, want)
		}
	})

	t.Run("extra positional", func(t *testing.T) {
		t.Parallel()
		type config struct{ World string }
		definition := cli.Definition[config]{
			Name: "alienctl",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Arguments.String(&root.Config.World, "world")
			},
			Run: func(context.Context, *config) error { return nil },
		}
		got, _ := renderRunError(t, definition, "mars", "extra")
		if want := "alienctl: unexpected argument \"extra\"\n"; got != want {
			t.Fatalf("WriteError() = %q, want %q", got, want)
		}
	})

	t.Run("finite domain", func(t *testing.T) {
		t.Parallel()
		type config struct {
			Approach string
			Moon     string
		}
		definition := cli.Definition[config]{
			Name: "alienctl",
			Declare: func(root *cli.CommandBuilder[config]) {
				approaches := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
					values.Value("orbit", "orbit")
					values.Value("land", "land")
					values.Value("flyby", "flyby")
				})
				approach := root.Values.Value(&root.Config.Approach, approaches)
				root.Flags.ValueFor(approach).Long("approach")
				root.Arguments.String(&root.Config.Moon, "moon")
			},
			Run: func(context.Context, *config) error { return nil },
		}
		got, _ := renderRunError(t, definition, "--approach", "teleport", "europa")
		if want := "alienctl: invalid value \"teleport\" for --approach\n  expected: orbit, land, flyby\n"; got != want {
			t.Fatalf("WriteError() = %q, want %q", got, want)
		}
	})

	for _, sensitive := range []bool{false, true} {
		name := "structured rejection"
		if sensitive {
			name = "sensitive rejection"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			type config struct{ Code string }
			definition := cli.Definition[config]{
				Name: "alienctl",
				Declare: func(root *cli.CommandBuilder[config]) {
					value := root.Values.String(&root.Config.Code)
					root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
						language := cli.TokenLanguage[string]{
							Name:  "dock-code",
							Count: cli.Count.Exactly(1),
							Parse: func(context.Context, []string) (string, error) {
								if sensitive {
									return "", cli.Rejection.Message(
										"secret-code",
										"secret message ZZ",
										cli.Rejection.At(0, 0, 2),
										cli.Rejection.Expected("SECRET-A", "SECRET-B"),
										cli.Rejection.Suggest("SECRET-A"),
										cli.Rejection.ReplaceWith("SECRET-A"),
										cli.Rejection.Hint("secret hint"),
										cli.Rejection.Sensitive(),
									)
								}
								return "", cli.Rejection.Message(
									"unknown-dock-code",
									"dock code must be A1 or B2",
									cli.Rejection.At(0, 0, 2),
									cli.Rejection.Expected("A1", "B2"),
									cli.Rejection.Suggest("A1"),
									cli.Rejection.ReplaceWith("A1"),
									cli.Rejection.Hint("Run `alienctl docks` to list available docking codes."),
								)
							},
							Format: func(value string) []string { return []string{value} },
						}
						form.Tokens(value, "subject", language)
					})
				},
				Run: func(context.Context, *config) error { return nil },
			}
			got, _ := renderRunError(t, definition, "ZZ")
			if sensitive {
				if want := "alienctl: invalid value for <subject>\n"; got != want {
					t.Fatalf("WriteError() = %q, want %q", got, want)
				}
				for _, secret := range []string{"ZZ", "SECRET-A", "SECRET-B", "secret message", "secret hint"} {
					if strings.Contains(got, secret) {
						t.Fatalf("sensitive WriteError() leaked %q: %q", secret, got)
					}
				}
				return
			}
			want := "alienctl: dock code must be A1 or B2\n" +
				"  input: ZZ\n" +
				"  expected: A1, B2\n" +
				"  suggestion: A1\n" +
				"  hint: Run `alienctl docks` to list available docking codes.\n"
			if got != want {
				t.Fatalf("WriteError() = %q, want %q", got, want)
			}
		})
	}

	t.Run("action conflict", func(t *testing.T) {
		t.Parallel()
		definition := cli.Definition[struct{}]{
			Name:    "alienctl",
			Release: cli.Release.SemVer("1.0.0"),
		}
		got, _ := renderRunError(t, definition, "--help", "--version")
		if want := "alienctl: conflicting actions: --help and --version\n"; got != want {
			t.Fatalf("WriteError() = %q, want %q", got, want)
		}
	})
}

func TestWriteErrorEscapesUntrustedControlTextWithoutFlatteningDocumentStructure(t *testing.T) {
	t.Parallel()
	type config struct{ Code string }
	definition := cli.Definition[config]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Code)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				language := cli.TokenLanguage[string]{
					Name:  "code",
					Count: cli.Count.Exactly(1),
					Parse: func(context.Context, []string) (string, error) {
						return "", cli.Rejection.Message(
							"bad-code",
							"bad\x1b[31m\n\tΩ code",
							cli.Rejection.At(0, 0, 3),
							cli.Rejection.Hint("try Ω"),
						)
					},
					Format: func(value string) []string { return []string{value} },
				}
				form.Tokens(value, "code", language)
			})
		},
	}
	app := clitest.Compile(t, definition)
	request := cli.RunRequest{
		Args: []string{"bad"},
	}
	err := app.Run(t.Context(), request)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	got := diagnosticErrorText(t, err)
	for _, want := range []string{
		"alienctl: bad\\x1b[31m\\x0a\\x09Ω code\n",
		"  input: bad\n",
		"  hint: try Ω\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("WriteError() missing %q: %q", want, got)
		}
	}
	if strings.ContainsRune(got, '\x1b') {
		t.Fatalf("WriteError() contains raw ESC: %q", got)
	}
	if got := fmt.Sprintf("%+v", err); strings.ContainsRune(got, '\x1b') {
		t.Fatalf("%%+v contains raw ESC: %q", got)
	}
}
