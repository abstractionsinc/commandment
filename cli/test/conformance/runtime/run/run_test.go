package conformance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCompiledApplicationRunsDeclaredHandler(t *testing.T) {
	t.Parallel()
	called := false
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "compile",
		Run:  func(context.Context, *struct{}) error { called = true; return nil },
	})
	if err := app.Run(t.Context(), cli.RunRequest{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !called {
		t.Fatal("compiled handler was not called")
	}
}

var _ = errors.Is

func runMustCompilePublic[C any](t testing.TB, definition cli.Definition[C]) cli.Application {
	t.Helper()
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return app
}

type frontendActionConfig struct{}

type failingVersionWriter struct{ err error }

func (writer failingVersionWriter) Write([]byte) (int, error) { return 0, writer.err }

func TestExecuteVersionActionWritesVersionWithoutCallingHandlerAndPropagatesWriteFailure(t *testing.T) {
	t.Parallel()
	type config = frontendActionConfig
	var output strings.Builder
	called := false
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.Ordered("v1.2.3", "v1.2.3"),
		Run:     func(context.Context, *config) error { called = true; return nil },
	}
	app := clitest.Compile(t, definition)
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--version"},
		Stdout: &output,
	}); err != nil {
		t.Fatal(err)
	}
	if output.String() != "v1.2.3\n" || called {
		t.Fatalf("output=%q called=%v", output.String(), called)
	}
	cause := errors.New("write")
	err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"--version"},
		Stdout: failingVersionWriter{
			err: cause,
		},
	})
	if !errors.Is(err, cause) {
		t.Fatalf("write error=%v", err)
	}
	clitest.Diagnostics(err).Require(t, clitest.Diagnostic(cli.DiagnosticOutputWriteFailure).Present())
}

func TestExecuteVersionActionUsesUnicodeCodePointCommitBound(t *testing.T) {
	t.Parallel()
	var output strings.Builder
	definition := cli.Definition[struct{}]{
		Name:    "tool",
		Release: cli.Release.SemVer("1.2.3"),
		Version: cli.VersionConfig{
			Commit:       "αβγdelta",
			CommitLength: 3,
		},
	}
	app := clitest.Compile(t, definition)
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--version"},
		Stdout: &output,
	}); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "1.2.3 (αβγ)\n"; got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}

func runOnlyInspectionCandidate(t *testing.T, inspection cli.Inspection) cli.ValueCandidateInspection {
	t.Helper()
	if len(inspection.ValueCandidates) != 1 {
		t.Fatalf("Inspection.ValueCandidates = %#v, want exactly one candidate", inspection.ValueCandidates)
	}
	for _, candidate := range inspection.ValueCandidates {
		return candidate
	}
	panic("unreachable")
}

type invocationContextRootConfig struct{}

type invocationContextLocaleSelectionChildConfig struct {
	Locale    cli.Locale
	Selection cli.CommandSelection
}

type invocationContextConfig struct{}

type invocationContextSelectorLocaleStdinStdoutStderrConfig struct {
	Selector string
	Locale   cli.Locale
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
}

type invocationContextLocaleConfig struct{ Locale cli.Locale }

type invocationContextStdinStdoutStderrConfig struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func TestWithInvocationObservationContextPreservesInvocationInvariant(t *testing.T) {
	t.Parallel()
	type rootConfig = invocationContextRootConfig
	type childConfig = invocationContextLocaleSelectionChildConfig
	var first cli.CommandSelection
	definition := cli.Definition[rootConfig]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
		},
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "child",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					child.Access.Locale(&child.Config.Locale)
					child.Access.CommandSelection(&child.Config.Selection)
				},
				Run: func(_ context.Context, cfg *childConfig) error {
					if cfg.Locale != "en-US" {
						t.Fatalf("access locale=%q, want en-US", cfg.Locale)
					}
					first = cfg.Selection
					cfg.Selection.Path[0] = 99
					if len(cfg.Selection.Steps) != 0 && len(cfg.Selection.Steps[0].Commands) != 0 {
						cfg.Selection.Steps[0].Commands[0] = 99
					}
					return nil
				},
			})
		},
	}
	app := runMustCompilePublic(t, definition)
	contextProof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args:   []string{"child"},
		Locale: "en-US",
	})
	contextProof.Require(
		t,
		clitest.EffectiveLocale("en-US"),
		clitest.SelectionPathExactly(clitest.RootCommand(), clitest.Command("child")),
	)
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"child"},
		Locale: "en-US",
	}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(first.Path) != 2 || first.Root == 0 || first.Selected == 0 {
		t.Fatalf("selection=%#v", first)
	}
	var second cli.CommandSelection
	app2 := runMustCompilePublic(t, cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[childConfig]{
				Name:    "child",
				Declare: func(child *cli.CommandBuilder[childConfig]) { child.Access.CommandSelection(&child.Config.Selection) },
				Run:     func(_ context.Context, cfg *childConfig) error { second = cfg.Selection; return nil },
			})
		},
	})
	if err := app2.Run(t.Context(), cli.RunRequest{
		Args: []string{"child"},
	}); err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if len(second.Path) != 2 || second.Path[0] == 99 {
		t.Fatalf("selection snapshot mutation leaked: %#v", second)
	}
}

func TestInvocationObservationContextHelpersHaveHarmlessNoStateBehavior(t *testing.T) {
	t.Parallel()
	if got := cli.EffectiveLocale(nil); got != "" { //nolint:staticcheck
		t.Fatalf("EffectiveLocale(nil) = %q", got)
	}
	type config = invocationContextConfig
	app := runMustCompilePublic(t, cli.Definition[config]{
		Name: "tool",
		Run: func(ctx context.Context, _ *config) error {
			if got := cli.EffectiveLocale(ctx); got != "" {
				t.Fatalf("EffectiveLocale without localization = %q", got)
			}
			return nil
		},
	})
	if err := app.Run(t.Context(), cli.RunRequest{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestStabilizeInvocationRuntimeContextUsesCanonicalLocalePrecedenceAndBorrowedStreams(t *testing.T) {
	t.Parallel()
	type config = invocationContextSelectorLocaleStdinStdoutStderrConfig
	stdin := bytes.NewBufferString("in")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	var wantLocale cli.Locale
	var runs int
	definition := cli.Definition[config]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			selector := root.Values.String(&root.Config.Selector)
			root.Flags.ValueFor(selector).LocaleSelector().Occurrences(cli.Count.Optional()).
				Long("locale")
			root.Access.Locale(&root.Config.Locale)
			root.Access.Stdin(&root.Config.Stdin)
			root.Access.Stdout(&root.Config.Stdout)
			root.Access.Stderr(&root.Config.Stderr)
		},
		Run: func(_ context.Context, cfg *config) error {
			runs++
			if cfg.Locale != wantLocale {
				t.Fatalf("access locale=%q want=%q", cfg.Locale, wantLocale)
			}
			if cfg.Stdin != stdin || cfg.Stdout != stdout || cfg.Stderr != stderr {
				t.Fatalf("borrowed streams changed: %#v %#v %#v", cfg.Stdin, cfg.Stdout, cfg.Stderr)
			}
			return nil
		},
	}
	app := runMustCompilePublic(t, definition)
	wantLocale = "fr-CA-x-host"
	runRequest := cli.RunRequest{
		Locale: "fr-ca-x-host",
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	}
	if err := app.Run(t.Context(), runRequest); err != nil {
		t.Fatalf("host locale Run: %v", err)
	}
	wantLocale = "de-DE"
	runRequestUnderTest := cli.RunRequest{
		Args:   []string{"--locale", "de-DE"},
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	}
	if err := app.Run(t.Context(), runRequestUnderTest); err != nil {
		t.Fatalf("selector locale Run: %v", err)
	}
	wantLocale = "en"
	runRequestUnderTest2 := cli.RunRequest{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	}
	if err := app.Run(t.Context(), runRequestUnderTest2); err != nil {
		t.Fatalf("source locale Run: %v", err)
	}
	if runs != 3 {
		t.Fatalf("runs=%d", runs)
	}
}

func TestStabilizeInvocationRuntimeContextRejectsInvalidHigherPrecedenceLocaleWithoutFallback(t *testing.T) {
	t.Parallel()
	type config = invocationContextLocaleConfig
	called := false
	definition := cli.Definition[config]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
		},
		Declare: func(root *cli.CommandBuilder[config]) { root.Access.Locale(&root.Config.Locale) },
		Run:     func(context.Context, *config) error { called = true; return nil },
	}
	app := runMustCompilePublic(t, definition)
	err := app.Run(t.Context(), cli.RunRequest{
		Locale: "not_a_locale",
	})
	if err == nil || called {
		t.Fatalf("invalid host locale = err %v called=%v", err, called)
	}
}

func TestInvocationRuntimeStreamHelpersHaveHarmlessNoStateBehavior(t *testing.T) {
	t.Parallel()
	type config = invocationContextStdinStdoutStderrConfig
	stdin := bytes.NewBufferString("in")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := runMustCompilePublic(t, cli.Definition[config]{
		Name: "tool",
		Run: func(_ context.Context, cfg *config) error {
			if cfg.Stdin != nil || cfg.Stdout != nil || cfg.Stderr != nil {
				t.Fatalf("undeclared stream access populated config: %#v", cfg)
			}
			return nil
		},
	})
	runRequest := cli.RunRequest{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	}
	if err := app.Run(t.Context(), runRequest); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

type runCountConfig struct{ count int }

type runProfileRootConfig struct {
	Profile string
}

type runProfileOptionalProfileProfileStateLocaleSelectionWorkingDirectoryStdinStdoutStderrChildConfig struct {
	Profile          string
	OptionalProfile  cli.OptionalValue[string]
	ProfileState     cli.OptionalValue[cli.ValueState]
	Locale           cli.Locale
	Selection        cli.CommandSelection
	WorkingDirectory string
	Stdin            io.Reader
	Stdout           io.Writer
	Stderr           io.Writer
}

type runValueCopyConfig struct {
	Value string
	Copy  string
}

type runPrimarySecondaryDuplicateConfig struct {
	Primary   cli.Locale
	Secondary cli.Locale
}

type runProfileRootConfig2 struct{ Profile string }

type runProfileLocaleChildConfig struct {
	Profile string
	Locale  cli.Locale
}

type runLocaleConfig struct{ Locale string }

type runProfileChildConfig struct{ Profile string }

type runABConfig struct {
	A string
	B string
}

type runValueConfig struct{ Value string }

type runRootConfig struct{}

type runPairChildConfig struct{ Pair runTokenPairValue }

type runCountConfig2 struct{ Count int }

type runConfig struct{}

type runPairConfig struct{ Pair runTokenPairValue }

type runModeConfig struct{ Mode string }

func TestRunApplicationPreservesRunSemantics(t *testing.T) {
	t.Parallel()
	type config = runCountConfig
	var seen []*config
	definition := cli.Definition[config]{
		Name: "commandment",
		Middleware: []cli.Middleware[config]{func(ctx context.Context, cfg *config, next cli.MiddlewareNext) error {
			cfg.count++
			seen = append(seen, cfg)
			return next(ctx)
		}},
		Run: func(_ context.Context, cfg *config) error {
			if cfg.count != 1 {
				t.Fatalf("handler config count = %d, want middleware mutation", cfg.count)
			}
			seen = append(seen, cfg)
			return nil
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	if err := app.Run(t.Context(), cli.RunRequest{}); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	if err := app.Run(t.Context(), cli.RunRequest{}); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}
	if len(seen) != 4 {
		t.Fatalf("observed configs = %d, want 4", len(seen))
	}
	if seen[0] != seen[1] || seen[2] != seen[3] {
		t.Fatalf(
			"middleware/handler did not share same-scope config: %p %p / %p %p",
			seen[0],
			seen[1],
			seen[2],
			seen[3],
		)
	}
	if seen[0] == seen[2] {
		t.Fatalf("distinct invocations shared configuration storage: %p", seen[0])
	}
}

func TestRunNonRunnableCommandReportsSelectedCommandWithoutExecution(t *testing.T) {
	t.Parallel()
	rootMiddlewareCalls := 0
	childMiddlewareCalls := 0
	definition := cli.Definition[struct{}]{
		Name: "commandment",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Middleware: []cli.Middleware[struct{}]{func(ctx context.Context, _ *struct{}, next cli.MiddlewareNext) error {
			rootMiddlewareCalls++
			return next(ctx)
		}},
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "child",
				Middleware: []cli.Middleware[struct{}]{
					func(ctx context.Context, _ *struct{}, next cli.MiddlewareNext) error {
						childMiddlewareCalls++
						return next(ctx)
					},
				},
			})
		},
	}
	app := clitest.Compile(t, definition)

	inspection, inspectErr := app.Inspect(t.Context(), cli.InspectionRequest{
		Args: []string{"child"},
	})
	if inspectErr != nil {
		t.Fatalf("Inspect() error = %v", inspectErr)
	}
	if inspection.ExecutionAdmission.Status != cli.ExecutionAdmissionNotRunnable {
		t.Fatalf(
			"Inspect() execution admission status = %v, want %v",
			inspection.ExecutionAdmission.Status,
			cli.ExecutionAdmissionNotRunnable,
		)
	}

	var childID cli.CommandID
	for _, command := range app.Schema().Commands {
		if command.Name == "child" {
			childID = command.ID
			break
		}
	}
	if childID == 0 {
		t.Fatal("child command missing from Schema")
	}

	err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"child"},
	})
	if err == nil {
		t.Fatal("Run() error = nil, want not-runnable diagnostic")
	}
	report := cli.Diagnostics(err)
	if len(report.Diagnostics) != 1 {
		t.Fatalf("Diagnostics() count = %d, want 1: %#v", len(report.Diagnostics), report.Diagnostics)
	}
	diagnostic := report.Diagnostics[0]
	if diagnostic.Code != cli.DiagnosticNotRunnableCommand {
		t.Fatalf("Diagnostic code = %q, want %q", diagnostic.Code, cli.DiagnosticNotRunnableCommand)
	}
	if diagnostic.Command != childID {
		t.Fatalf("Diagnostic command = %v, want child %v", diagnostic.Command, childID)
	}
	if got := cli.ExitCode(err); got != 1 {
		t.Fatalf("ExitCode() = %d, want 1", got)
	}
	if rootMiddlewareCalls != 0 || childMiddlewareCalls != 0 {
		t.Fatalf(
			"non-runnable selection admitted middleware: root=%d child=%d",
			rootMiddlewareCalls,
			childMiddlewareCalls,
		)
	}
}

func TestRunFrameworkDiagnosticLocalizationUsesFrameworkOwnerAndDetachedSnapshot(t *testing.T) {
	t.Parallel()
	applicationTranslations := cli.Translations{
		"Commandment diagnostic":           "APPLICATION COLLISION TITLE",
		"message":                          "APPLICATION COLLISION LABEL",
		"selected command is not runnable": "APPLICATION COLLISION MESSAGE",
	}
	frameworkTranslations := cli.Translations{
		"Commandment diagnostic":           "Diagnostic Commandment FR",
		"message":                          "message-cadre-fr",
		"selected command is not runnable": "commande sélectionnée non exécutable",
	}
	definition := cli.Definition[struct{}]{
		Name: "commandment",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale:       "fr",
				Translations: applicationTranslations,
			}},
		},
		FrameworkTranslations: []cli.TranslationBundle{{
			Locale:       "fr",
			Translations: frameworkTranslations,
		}},
	}
	app := clitest.Compile(t, definition)

	err := app.Run(t.Context(), cli.RunRequest{
		Locale: "fr",
	})
	if err == nil {
		t.Fatal("Run() error = nil, want not-runnable diagnostic")
	}
	report := cli.Diagnostics(err)
	if len(report.Diagnostics) != 1 {
		t.Fatalf("Diagnostics() count = %d, want 1: %#v", len(report.Diagnostics), report.Diagnostics)
	}
	if got := report.Diagnostics[0].Message; got != "commande sélectionnée non exécutable" {
		t.Fatalf("framework diagnostic message = %q", got)
	}

	// Mutating the caller-owned authoring maps after Compile must not affect a
	// delayed presentation of the already-returned diagnostic.
	applicationTranslations["selected command is not runnable"] = "MUTATED APPLICATION"
	frameworkTranslations["selected command is not runnable"] = "MUTATED FRAMEWORK"
	rich := fmt.Sprintf("%+v", err)
	for _, want := range []string{
		"Diagnostic Commandment FR [not-runnable-command]",
		"message-cadre-fr: commande sélectionnée non exécutable",
	} {
		if !strings.Contains(rich, want) {
			t.Fatalf("localized delayed diagnostic missing %q:\n%s", want, rich)
		}
	}
	for _, forbidden := range []string{"APPLICATION COLLISION", "MUTATED APPLICATION", "MUTATED FRAMEWORK"} {
		if strings.Contains(rich, forbidden) {
			t.Fatalf("localized delayed diagnostic crossed owner/snapshot boundary %q: %s", forbidden, rich)
		}
	}
}

func TestRunGrammarDiagnosticFailurePreservesRunSemantics(t *testing.T) {
	t.Parallel()
	type config = runConfig
	middlewareCalled := false
	handlerCalled := false
	definition := cli.Definition[config]{
		Name: "commandment",
		Middleware: []cli.Middleware[config]{func(ctx context.Context, _ *config, next cli.MiddlewareNext) error {
			middlewareCalled = true
			return next(ctx)
		}},
		Run: func(context.Context, *config) error {
			handlerCalled = true
			return nil
		},
	}
	app := clitest.Compile(t, definition)
	err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"--definitely-unknown"},
	})
	clitest.Diagnostics(err).Require(t, clitest.DiagnosticAt(0).Present())
	if middlewareCalled || handlerCalled {
		t.Fatalf("failed preparation admitted execution: middleware=%v handler=%v", middlewareCalled, handlerCalled)
	}
}

func TestRunDoesNotRejectResponseCapableApplicationWithoutResponseLexicalEntry(t *testing.T) {
	t.Parallel()
	type config = runConfig
	called := 0
	definition := cli.Definition[config]{
		Name: "commandment",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
		},
		Run: func(context.Context, *config) error {
			called++
			return nil
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	semanticFiles := clitest.ResponseFiles{}
	responseProof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args:          nil,
		ResponseFiles: semanticFiles.Access(),
	})
	responseProof.Require(t, clitest.InspectionReady())
	if err := app.Run(t.Context(), cli.RunRequest{}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if called != 1 {
		t.Fatalf("handler calls = %d, want 1", called)
	}
}

func TestRunResponseLiteralEscapeDoesNotLoadSource(t *testing.T) {
	t.Parallel()
	type config = runConfig
	loads := 0
	app, err := cli.Compile(cli.Definition[config]{
		Name: "commandment",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	semanticFiles := clitest.ResponseFiles{}
	responseProof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args:          []string{"@@literal"},
		ResponseFiles: semanticFiles.Access(),
	})
	_, semanticErr := responseProof.Inspection()
	clitest.Diagnostics(semanticErr).Require(t, clitest.Diagnostic(cli.DiagnosticResponseFileLoadFailure).Absent())
	// No positional grammar exists, so the rewritten literal will be rejected
	// by ordinary grammar. The response layer must nevertheless not load it.
	_ = app.Run(t.Context(), cli.RunRequest{
		Args: []string{"@@literal"},
		ResponseFiles: cli.ResponseFileAccess{
			Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
				loads++
				return cli.ResponseFile{}, nil
			},
		},
	})
	if loads != 0 {
		t.Fatalf("literal response escape caused %d loads", loads)
	}
}

func TestDeclaredExecutionAccessLowersProjectsAndRealizesSelectedAncestorAndFacilities(t *testing.T) {
	t.Parallel()
	type rootConfig = runProfileRootConfig
	type childConfig = runProfileOptionalProfileProfileStateLocaleSelectionWorkingDirectoryStdinStdoutStderrChildConfig

	var childSeen *childConfig
	app, err := cli.Compile(cli.Definition[rootConfig]{
		Name: "commandment",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			profile := root.Values.String(&root.Config.Profile)
			root.Flags.ValueFor(profile).Long("profile")
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "child",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					child.Access.Value(profile, &child.Config.Profile)
					child.Access.OptionalValue(profile, &child.Config.OptionalProfile)
					child.Access.ValueState(profile, &child.Config.ProfileState)
					child.Access.Locale(&child.Config.Locale)
					child.Access.CommandSelection(&child.Config.Selection)
					child.Access.WorkingDirectory(&child.Config.WorkingDirectory)
					child.Access.Stdin(&child.Config.Stdin)
					child.Access.Stdout(&child.Config.Stdout)
					child.Access.Stderr(&child.Config.Stderr)
				},
				Run: func(_ context.Context, config *childConfig) error {
					childSeen = config
					return nil
				},
			})
		},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	schemaProof := clitest.Schema(app)
	expectedAccess := clitest.ExecutionAccess{
		Values:           []clitest.ValueLocator{clitest.RootCommand().ValueAt(0)},
		OptionalValues:   []clitest.ValueLocator{clitest.RootCommand().ValueAt(0)},
		ValueStates:      []clitest.ValueLocator{clitest.RootCommand().ValueAt(0)},
		Locale:           true,
		CommandSelection: true,
		WorkingDirectory: true,
		Stdin:            true,
		Stdout:           true,
		Stderr:           true,
	}
	schemaProof.Require(t, clitest.Command("child").ExecutionAccessExactly(expectedAccess))
	schema := schemaProof.Schema()
	var child cli.CommandSchema
	for _, command := range schema.Commands {
		if command.Name == "child" {
			child = command
			break
		}
	}
	if child.ID == 0 {
		t.Fatal("child command missing from Schema")
	}

	stdin := strings.NewReader("input")
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	runRequest := cli.RunRequest{
		Args:             []string{"--profile", "prod", "child"},
		WorkingDirectory: "/workspace",
		Stdin:            stdin,
		Stdout:           stdout,
		Stderr:           stderr,
		Locale:           "en-US",
	}
	err = app.Run(t.Context(), runRequest)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if childSeen == nil {
		t.Fatal("child handler did not observe realized access")
	}
	got := childSeen
	if got.Profile != "prod" || !got.OptionalProfile.Present || got.OptionalProfile.Value != "prod" ||
		!got.ProfileState.Present {
		t.Fatalf("child value Access = %#v", got)
	}
	if got.Locale != "en-US" || got.WorkingDirectory != "/workspace" || got.Stdin != stdin || got.Stdout != stdout ||
		got.Stderr != stderr {
		t.Fatalf("child facility Access = %#v", got)
	}
	if len(got.Selection.Path) != 2 || got.Selection.Path[1] != child.ID {
		t.Fatalf("child selection Access = %#v", got.Selection)
	}
}

func TestDeclaredExecutionAccessRejectsLocalRequiredValueNilTargetAndDuplicateFacility(t *testing.T) {
	t.Parallel()
	type config = runValueCopyConfig

	_, err := cli.Compile(cli.Definition[config]{
		Name: "commandment",
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			root.Access.Value(value, &root.Config.Copy)
		},
	})
	if err == nil {
		t.Fatal("Compile() accepted local Access.Value")
	}

	_, err = cli.Compile(cli.Definition[config]{
		Name: "commandment",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Access.Locale((*cli.Locale)(nil))
		},
	})
	if err == nil {
		t.Fatal("Compile() accepted nil Access target")
	}

	type duplicateConfig = runPrimarySecondaryDuplicateConfig
	_, err = cli.Compile(cli.Definition[duplicateConfig]{
		Name: "commandment",
		Declare: func(root *cli.CommandBuilder[duplicateConfig]) {
			root.Access.Locale(&root.Config.Primary)
			root.Access.Locale(&root.Config.Secondary)
		},
	})
	if err == nil {
		t.Fatal("Compile() accepted duplicate Locale Access declarations")
	}
}

func TestRunAdmitsSupportedSelectedPathOnlyAfterAccessRealization(t *testing.T) {
	t.Parallel()
	type rootConfig = runProfileRootConfig2
	type childConfig = runProfileLocaleChildConfig
	applicationErr := errors.New("application failure")
	var order []string
	definition := cli.Definition[rootConfig]{
		Name: "commandment",
		Middleware: []cli.Middleware[rootConfig]{
			func(ctx context.Context, config *rootConfig, next cli.MiddlewareNext) error {
				order = append(order, "root:"+config.Profile)
				return next(ctx)
			},
		},
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			profile := root.Values.String(&root.Config.Profile)
			root.Flags.ValueFor(profile).Long("profile")
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "child",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					child.Access.Value(profile, &child.Config.Profile)
					child.Access.Locale(&child.Config.Locale)
				},
				Middleware: []cli.Middleware[childConfig]{
					func(ctx context.Context, config *childConfig, next cli.MiddlewareNext) error {
						order = append(order, "child:"+config.Profile+":"+string(config.Locale))
						return next(ctx)
					},
				},
				Run: func(_ context.Context, config *childConfig) error {
					order = append(order, "handler:"+config.Profile+":"+string(config.Locale))
					return applicationErr
				},
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	got := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--profile", "prod", "child"},
		Locale: "en-US",
	})
	if !errors.Is(got, applicationErr) {
		t.Fatalf("Run() error = %v, want preserved application error", got)
	}
	want := []string{"root:prod", "child:prod:en-US", "handler:prod:en-US"}
	if !slices.Equal(order, want) {
		t.Fatalf("execution order = %#v, want %#v", order, want)
	}
}

func TestLocaleSelectorUniqueApplicationDeclarationCompilesAndProjects(t *testing.T) {
	t.Parallel()
	type config = runLocaleConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			locale := root.Values.String(&root.Config.Locale)
			root.Values.Default(locale, "en")
			root.Values.Environment(locale, "TOOL_LOCALE")
			root.Flags.ValueFor(locale).LocaleSelector().Occurrences(cli.Count.Optional()).
				Long("locale")
		},
	}
	app := clitest.Compile(t, definition)
	locale := clitest.RootCommand().Flag("--locale")
	clitest.Schema(app).Require(t,
		clitest.RootCommand().FlagsExactly(locale),
		locale.LocaleSelector(),
	)
}

func TestProviderRuntimeCarriesStaticProofFactsThroughPublicationScope(t *testing.T) {
	t.Parallel()
	type config = runConfig

	for _, test := range []struct {
		name              string
		staticHasDefault  bool
		wantProviderState cli.CommandProviderOutcome
		wantErr           bool
	}{
		{
			name:              "proved overlap rejects",
			staticHasDefault:  true,
			wantProviderState: cli.CommandProviderOutcomeRejected,
			wantErr:           true,
		},
		{
			name:              "no overlap accepts",
			staticHasDefault:  false,
			wantProviderState: cli.CommandProviderOutcomeAccepted,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			providerCalls := 0
			app := clitest.Compile(t, cli.Definition[config]{
				Name: "tool",
				Declare: func(root *cli.CommandBuilder[config]) {
					baseDefault := root.Subcommands.Command(cli.Command[config]{
						Name: "base",
					})
					if test.staticHasDefault {
						baseDefault.Default()
					}
					root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
						providerCalls++
						batch.Command(cli.Command[config]{
							Name: "provider-default",
						}).Default()
						return nil
					})
				},
			})

			provider := clitest.RootCommand().ProviderAt(0)
			providerCommand := clitest.RootCommand().Command("provider-default")
			proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
				Args: []string{"provider-default"},
			})
			if test.wantErr {
				proof.Require(t,
					clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidCommandBatch).Present()),
					clitest.InspectionObservedCommandProvidersExactly(provider),
					provider.Outcome(test.wantProviderState),
					provider.EntriesExactly(),
					providerCommand.ObservedDoesNotExist(),
				)
			} else {
				proof.Require(t,
					clitest.DiagnosticsExactly(),
					clitest.InspectionObservedCommandProvidersExactly(provider),
					provider.Outcome(test.wantProviderState),
					provider.EntriesExactly(providerCommand.CanonicalEntry()),
					providerCommand.Exists(),
				)
			}
			if providerCalls != 1 {
				t.Fatalf("provider calls = %d, want 1", providerCalls)
			}
		})
	}
}

func TestRunMaterializesProviderCommandAndRealizesProviderProducedAccess(t *testing.T) {
	t.Parallel()
	type rootConfig = runProfileRootConfig2
	type childConfig = runProfileChildConfig

	providerCalls := 0
	handlerCalls := 0
	var observed string
	definition := cli.Definition[rootConfig]{
		Name: "tool",
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			profile := root.Values.String(&root.Config.Profile)
			root.Flags.ValueFor(profile).Long("profile")
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[rootConfig]) error {
				providerCalls++
				dependency, ok := batch.Dependencies.Get(profile)
				if !ok || dependency != "prod" || batch.Config.Profile != "prod" {
					return errors.New("provider dependency not resolved from owned prefix")
				}
				batch.Command(cli.Command[childConfig]{
					Name: "remote",
					Declare: func(child *cli.CommandBuilder[childConfig]) {
						child.Access.Value(profile, &child.Config.Profile)
					},
					Run: func(_ context.Context, child *childConfig) error {
						handlerCalls++
						observed = child.Profile
						return nil
					},
				})
				return nil
			}, profile)
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	providerSchemaProof := clitest.Schema(app)
	providerSchemaProof.Require(
		t,
		clitest.RootCommand().ProviderAt(0).Exists(),
		clitest.CommandsExactly(clitest.RootCommand()),
	)
	if err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"--profile", "prod", "remote"},
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if providerCalls != 1 || handlerCalls != 1 || observed != "prod" {
		t.Fatalf("provider/handler result = provider %d handler %d profile %q", providerCalls, handlerCalls, observed)
	}
}

func TestRunVersionActionBypassesApplicationExecution(t *testing.T) {
	t.Parallel()
	type config = runConfig
	var output strings.Builder
	middlewareCalled := false
	handlerCalled := false
	definition := cli.Definition[config]{
		Name:    "commandment",
		Release: cli.Release.Ordered("v1.2.3", "v1.2.3"),
		Middleware: []cli.Middleware[config]{func(ctx context.Context, _ *config, next cli.MiddlewareNext) error {
			middlewareCalled = true
			return next(ctx)
		}},
		Run: func(context.Context, *config) error {
			handlerCalled = true
			return errors.New("must not execute")
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--version"},
		Stdout: &output,
	}); err != nil {
		t.Fatalf("Run(--version) error = %v", err)
	}
	if got := output.String(); got != "v1.2.3\n" {
		t.Fatalf("version output = %q, want %q", got, "v1.2.3\\n")
	}
	if middlewareCalled || handlerCalled {
		t.Fatalf(
			"version action executed application callbacks: middleware=%v handler=%v",
			middlewareCalled,
			handlerCalled,
		)
	}
}

func TestRunVersionActionReportsOutputWriteFailure(t *testing.T) {
	t.Parallel()
	type config = runConfig
	writeErr := errors.New("write failed")
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "commandment",
		Release: cli.Release.Ordered("v1.2.3", "v1.2.3"),
	})
	err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"--version"},
		Stdout: failingVersionWriter{
			err: writeErr,
		},
	})
	if !errors.Is(err, writeErr) {
		t.Fatalf("Run(--version) did not preserve write cause: %v", err)
	}
	clitest.Diagnostics(err).Require(t,
		clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticOutputWriteFailure)),
	)
}

func TestTokenLanguageRuntimeOwnsExactRangeAndProjectsAcceptedTypedValue(t *testing.T) {
	t.Parallel()
	var parseCalls atomic.Int32
	var formatCalls atomic.Int32
	var received []string
	tokenLanguage := cli.TokenLanguage[runTokenPairValue]{
		Name:  "pair",
		Count: cli.Count.Exactly(2),
		Parse: func(_ context.Context, tokens []string) (runTokenPairValue, error) {
			parseCalls.Add(1)
			received = append([]string(nil), tokens...)
			value := runTokenPairValue{
				Left:  tokens[0],
				Right: tokens[1],
			}
			tokens[0] = "parser-mutated"
			return value, nil
		},
		Format: func(value runTokenPairValue) []string {
			formatCalls.Add(1)
			return []string{"canon-" + value.Left, "canon-" + value.Right}
		},
	}

	type config = runPairConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			pair := root.Values.Value(&root.Config.Pair, runTokenPairCodec())
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Tokens(pair, "pair", tokenLanguage)
				form.Literal("tail")
			})
		},
	})
	value := clitest.RootCommand().Argument("pair").Value()
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"left", "right", "tail"},
	}).Require(t,
		clitest.InspectionReady(),
		clitest.TokenAt(0).Text("left"),
		clitest.TokenAt(1).Text("right"),
		clitest.TokenAt(2).Text("tail"),
		value.CandidateAt(0).Tokens("canon-left", "canon-right"),
		value.Text("left:right"),
	)
	if got := parseCalls.Load(); got != 1 {
		t.Fatalf("Parse calls = %d, want 1", got)
	}
	if got := formatCalls.Load(); got == 0 {
		t.Fatal("TokenLanguage Format was not used for canonical token projection")
	}
	if !slices.Equal(received, []string{"left", "right"}) {
		t.Fatalf("Parse tokens = %#v, want exact owned range", received)
	}
}

func TestTokenLanguageConversionParticipatesInPositionalFormSelectionWithoutReallocation(t *testing.T) {
	t.Parallel()
	type config = runABConfig
	var callsA, callsB atomic.Int32
	var rangesA, rangesB [][]string
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			a := root.Values.String(&root.Config.A)
			b := root.Values.String(&root.Config.B)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Tokens(a, "a", cli.TokenLanguage[string]{
					Name:  "a",
					Count: cli.Count.Between(1, 2),
					Parse: func(_ context.Context, tokens []string) (string, error) {
						callsA.Add(1)
						rangesA = append(rangesA, append([]string(nil), tokens...))
						return "", cli.Rejection.Message(
							"a-reject",
							"reject fixed greedy range",
							cli.Rejection.At(0, 0, 1),
						)
					},
					Format: func(value string) []string { return []string{value} },
				}).Greedy()
				form.Literal("tail")
			})
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Tokens(b, "b", cli.TokenLanguage[string]{
					Name:  "b",
					Count: cli.Count.Exactly(2),
					Parse: func(_ context.Context, tokens []string) (string, error) {
						callsB.Add(1)
						rangesB = append(rangesB, append([]string(nil), tokens...))
						return stringsJoinForTest(tokens), nil
					},
					Format: func(value string) []string { return []string{value} },
				})
				form.Literal("tail")
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := app.Inspect(t.Context(), cli.InspectionRequest{
		Args: []string{"one", "two", "tail"},
	})
	if err != nil {
		t.Fatalf("Inspect() error = %v; diagnostics=%#v", err, inspection.Diagnostics)
	}
	if callsA.Load() != 1 || callsB.Load() != 1 {
		t.Fatalf("Parse calls = A:%d B:%d, want one fixed evaluation each", callsA.Load(), callsB.Load())
	}
	if !slices.EqualFunc(rangesA, [][]string{{"one", "two"}}, slices.Equal) ||
		!slices.EqualFunc(rangesB, [][]string{{"one", "two"}}, slices.Equal) {
		t.Fatalf("fixed allocations changed after rejection: A=%#v B=%#v", rangesA, rangesB)
	}
	selection := inspection.Commands[inspection.Selection.Command.Selected].ArgumentFormSelection
	if selection.Outcome != cli.ArgumentFormSelectionSelected {
		t.Fatalf("selection = %#v, want selected B", selection)
	}
}

func TestTokenLanguageParserEvaluationIsIsolatedAcrossPublicOperations(t *testing.T) {
	t.Parallel()
	type config = runValueConfig
	var calls atomic.Int32
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Tokens(value, "value", cli.TokenLanguage[string]{
					Name:   "once",
					Count:  cli.Count.Exactly(1),
					Parse:  func(_ context.Context, tokens []string) (string, error) { calls.Add(1); return tokens[0], nil },
					Format: func(value string) []string { return []string{value} },
				})
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := range 2 {
		inspection, err := app.Inspect(t.Context(), cli.InspectionRequest{
			Args: []string{"same"},
		})
		if err != nil || inspection.Status != cli.InspectionStatusReady {
			t.Fatalf("operation %d: Inspect() = status %v, err %v", i, inspection.Status, err)
		}
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("Parse calls = %d, want one per independent public operation", got)
	}
}

func TestTokenLanguageFlagUsesRetainedRuntimeParserAndFormatter(t *testing.T) {
	t.Parallel()
	type config = runPairConfig
	var parseCalls atomic.Int32
	var received []string
	language := cli.TokenLanguage[runTokenPairValue]{
		Name:  "flag-pair",
		Count: cli.Count.Exactly(2),
		Parse: func(_ context.Context, tokens []string) (runTokenPairValue, error) {
			parseCalls.Add(1)
			received = append([]string(nil), tokens...)
			value := runTokenPairValue{
				Left:  tokens[0],
				Right: tokens[1],
			}
			return value, nil
		},
		Format: func(value runTokenPairValue) []string { return []string{"flag-" + value.Left, "flag-" + value.Right} },
	}
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			pair := root.Values.Value(&root.Config.Pair, runTokenPairCodec())
			root.Flags.Tokens(pair, language).Long("pair")
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := app.Inspect(t.Context(), cli.InspectionRequest{
		Args: []string{"--pair", "left", "right"},
	})
	if err != nil || inspection.Status != cli.InspectionStatusReady {
		t.Fatalf("Inspect() = status %v err %v diagnostics %#v", inspection.Status, err, inspection.Diagnostics)
	}
	if parseCalls.Load() != 1 || !slices.Equal(received, []string{"left", "right"}) {
		t.Fatalf("flag parser = calls %d tokens %#v", parseCalls.Load(), received)
	}
	candidate := runOnlyInspectionCandidate(t, inspection)
	if candidate.Representation == nil || candidate.Representation.Kind != cli.ValueRepresentationTokens ||
		!slices.Equal(candidate.Representation.Tokens, []string{"flag-left", "flag-right"}) {
		t.Fatalf("flag candidate representation = %#v", candidate.Representation)
	}
}

func TestProviderProducedTokenLanguageUsesCanonicalRuntimeCarrier(t *testing.T) {
	t.Parallel()
	type rootConfig = runRootConfig
	type childConfig = runPairChildConfig
	var parseCalls atomic.Int32
	var received []string

	app, err := cli.Compile(cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[rootConfig]) error {
				batch.Command(cli.Command[childConfig]{
					Name: "remote",
					Declare: func(child *cli.CommandBuilder[childConfig]) {
						pair := child.Values.Value(&child.Config.Pair, runTokenPairCodec())
						child.Arguments.Form(func(form *cli.ArgumentFormBuilder[childConfig]) {
							form.Tokens(pair, "pair", cli.TokenLanguage[runTokenPairValue]{
								Name:  "provider-pair",
								Count: cli.Count.Exactly(2),
								Parse: func(_ context.Context, tokens []string) (runTokenPairValue, error) {
									parseCalls.Add(1)
									received = append([]string(nil), tokens...)
									value := runTokenPairValue{
										Left:  tokens[0],
										Right: tokens[1],
									}
									return value, nil
								},
								Format: func(value runTokenPairValue) []string {
									return []string{"provider-" + value.Left, "provider-" + value.Right}
								},
							})
						})
					},
				})
				return nil
			})
		},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	inspectProof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"remote", "left", "right"},
	})
	inspectProof.Require(
		t,
		clitest.InspectionReady(),
		clitest.SelectionPathExactly(clitest.RootCommand(), clitest.Command("remote")),
	)
	inspection, err := inspectProof.Inspection()
	if err != nil || inspection.Status != cli.InspectionStatusReady {
		t.Fatalf("Inspect() = status %v err %v diagnostics %#v", inspection.Status, err, inspection.Diagnostics)
	}
	if got := parseCalls.Load(); got != 1 {
		t.Fatalf("provider TokenLanguage Parse calls = %d, want 1", got)
	}
	if !slices.Equal(received, []string{"left", "right"}) {
		t.Fatalf("provider TokenLanguage tokens = %#v", received)
	}
	candidate := runOnlyInspectionCandidate(t, inspection)
	if candidate.Representation == nil || candidate.Representation.Kind != cli.ValueRepresentationTokens ||
		!slices.Equal(candidate.Representation.Tokens, []string{"provider-left", "provider-right"}) {
		t.Fatalf("provider candidate representation = %#v", candidate.Representation)
	}
}

func TestFinalizeRunFailuresPreservesRunSemantics(t *testing.T) {
	t.Parallel()
	type config = runConfig
	primary := errors.New("application primary")
	noticeCause := errors.New("notice delivery")
	callerCause := errors.New("caller cancellation")
	ctx, cancel := context.WithCancelCause(t.Context())
	definition := cli.Definition[config]{
		Name: "commandment",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Advisory.Always(cli.Advisory{
				Code:     "settlement",
				Severity: cli.NoticeSeverityWarning,
				Message:  "settlement",
			})
		},
		Run: func(context.Context, *config) error { cancel(callerCause); return primary },
	}
	app := clitest.Compile(t, definition)
	got := app.Run(ctx, cli.RunRequest{
		Notify: func(context.Context, cli.Notice) error { return noticeCause },
	})
	if got == nil || got.Error() != primary.Error() {
		t.Fatalf("Run error=%v", got)
	}
	for _, branch := range []error{primary, noticeCause, callerCause} {
		if !errors.Is(got, branch) {
			t.Fatalf("Run error lost %v: %v", branch, got)
		}
	}
}

func TestRunRejectsInvalidAnalysisLimits(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "invalid-run-limits",
		Run:  func(context.Context, *struct{}) error { return nil },
	})
	err := app.Run(t.Context(), cli.RunRequest{
		Limits: cli.AnalysisLimits{
			Diagnostics: -1,
		},
	})
	if err == nil {
		t.Fatal("Run() error = nil, want invalid invocation")
	}
	clitest.Diagnostics(err).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidValue).Present())
}

func TestRunReportsMiddlewareProtocolViolation(t *testing.T) {
	t.Parallel()
	definition := cli.Definition[struct{}]{
		Name: "middleware-protocol",
		Middleware: []cli.Middleware[struct{}]{func(ctx context.Context, _ *struct{}, next cli.MiddlewareNext) error {
			if err := next(ctx); err != nil {
				return err
			}
			return next(ctx)
		}},
		Run: func(context.Context, *struct{}) error { return nil },
	}
	app := clitest.Compile(t, definition)
	err := app.Run(t.Context(), cli.RunRequest{})
	if err == nil {
		t.Fatal("Run() error = nil, want middleware protocol violation")
	}
	clitest.Diagnostics(err).Require(t, clitest.Diagnostic(cli.DiagnosticMiddlewareProtocolViolation).Present())
}

func TestRunPreservesResponseFileLoadFailure(t *testing.T) {
	t.Parallel()
	loadErr := errors.New("response load")
	definition := cli.Definition[struct{}]{
		Name: "response-run",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Dialect.ResponseFiles(true)
		},
		Run: func(context.Context, *struct{}) error { return nil },
	}
	app := clitest.Compile(t, definition)
	err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"@missing"},
		ResponseFiles: cli.ResponseFileAccess{
			Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
				return cli.ResponseFile{}, loadErr
			},
		},
	})
	if !errors.Is(err, loadErr) {
		t.Fatalf("Run() error = %v, want response load failure", err)
	}
	clitest.Diagnostics(err).Require(t, clitest.Diagnostic(cli.DiagnosticResponseFileLoadFailure).Present())
}

func TestRunPreservesCommandProviderFailure(t *testing.T) {
	t.Parallel()
	providerErr := errors.New("command provider")
	definition := cli.Definition[struct{}]{
		Name: "provider-run",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.CommandsFrom(
				func(context.Context, *cli.CommandBatch[struct{}]) error { return providerErr },
			)
		},
		Run: func(context.Context, *struct{}) error { return nil },
	}
	app := clitest.Compile(t, definition)
	err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"remote"},
	})
	if !errors.Is(err, providerErr) {
		t.Fatalf("Run() error = %v, want provider failure", err)
	}
}

func TestRunPreservesProviderFailureAndIndependentCallerCancellation(t *testing.T) {
	t.Parallel()
	providerFailure := errors.New("provider failure")
	callerCancellation := errors.New("caller cancellation")
	started := make(chan struct{})
	release := make(chan struct{})
	definition := cli.Definition[struct{}]{
		Name: "provider-run",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.CommandsFrom(func(context.Context, *cli.CommandBatch[struct{}]) error {
				close(started)
				<-release
				return providerFailure
			})
		},
		Run: func(context.Context, *struct{}) error { return nil },
	}
	app := clitest.Compile(t, definition)
	ctx, cancel := context.WithCancelCause(t.Context())
	result := make(chan error, 1)
	go func() { result <- app.Run(ctx, cli.RunRequest{}) }()
	<-started
	cancel(callerCancellation)
	<-ctx.Done()
	close(release)
	err := <-result
	if !errors.Is(err, providerFailure) {
		t.Fatalf("Run() error = %v, want provider failure", err)
	}
	if !errors.Is(err, callerCancellation) {
		t.Fatalf("Run() error = %v, want independent caller cancellation", err)
	}
	clitest.Diagnostics(err).Require(t,
		clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticCommandProviderFailure)),
	)
}

func TestRunMaterializesDemandedValueDomain(t *testing.T) {
	t.Parallel()
	type config = runModeConfig
	calls := 0
	definition := cli.Definition[config]{
		Name: "domain-run",
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(_ context.Context, _ *config, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
					calls++
					builder.Value("ready", "ready")
					return nil
				},
			)
			root.Flags.Value(&root.Config.Mode, domain).Long("mode")
		},
		Run: func(context.Context, *config) error { return nil },
	}
	app := clitest.Compile(t, definition)
	if err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"--mode", "ready"},
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("dynamic domain calls = %d, want 1", calls)
	}
}

func TestRunPreservesDemandedValueDomainFailure(t *testing.T) {
	t.Parallel()
	type config = runModeConfig
	domainErr := errors.New("value domain")
	definition := cli.Definition[config]{
		Name: "domain-run-failure",
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(context.Context, *config, cli.Dependencies, *cli.ValueDomainBuilder[string]) error {
					return domainErr
				},
			)
			root.Flags.Value(&root.Config.Mode, domain).Long("mode")
		},
		Run: func(context.Context, *config) error { return nil },
	}
	app := clitest.Compile(t, definition)
	err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"--mode", "ready"},
	})
	if !errors.Is(err, domainErr) {
		t.Fatalf("Run() error = %v, want dynamic-domain failure", err)
	}
	clitest.Diagnostics(err).Require(t, clitest.Diagnostic(cli.DiagnosticValueDomainProvisionFailure).Present())
}

func TestRunReportsValueDiagnostic(t *testing.T) {
	t.Parallel()
	type config = runCountConfig2
	definition := cli.Definition[config]{
		Name: "value-diagnostic",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.Int(&root.Config.Count).Long("count")
		},
		Run: func(context.Context, *config) error { return nil },
	}
	app := clitest.Compile(t, definition)
	err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"--count", "not-an-integer"},
	})
	if err == nil {
		t.Fatal("Run() error = nil, want value diagnostic")
	}
	clitest.Diagnostics(err).Require(t, clitest.DiagnosticAt(0).Present())
}

func TestRunObservesResolutionCancellationAfterPolicyAnalysis(t *testing.T) {
	t.Parallel()
	entered := make(chan struct{})
	handlerCalled := false
	source, control := clitest.SignalSource(clitest.SignalSourceOptions[string]{
		Signals: []cli.SignalDescriptor[string]{{
			Signal:   runSignalNativeInterrupt,
			Fidelity: cli.SignalSourceDeliveryFidelityExact,
		}},
	})
	definition := cli.Definition[struct{}]{
		Name: "resolution-cancel-after-policy",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			registered := root.Signals.Source(source)
			root.Signals.Policy(
				root.Signals.Match.On(registered.Signal(runSignalNativeInterrupt)).
					During(cli.SignalPhaseResolution).
					Each(),
				root.Signals.Action.Cancel(),
			)
			root.Rule.Check(
				"wait-for-cancel",
				func(ctx context.Context, _ *struct{}, _ cli.Dependencies) (bool, error) {
					close(entered)
					<-ctx.Done()
					return true, nil
				},
			)
		},
		Run: func(context.Context, *struct{}) error {
			handlerCalled = true
			return nil
		},
	}
	app := clitest.Compile(t, definition)
	done := make(chan error, 1)
	go func() {
		done <- app.Run(t.Context(), cli.RunRequest{})
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("policy analysis did not begin")
	}
	select {
	case <-control.Opened():
	case <-time.After(2 * time.Second):
		t.Fatal("signal source did not open")
	}
	if err := control.Notify(cli.SignalNotification[string]{
		Signal: runSignalNativeInterrupt,
		Origin: cli.SignalOriginProcess,
	}); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}
	select {
	case err := <-done:
		if _, ok := errors.AsType[*cli.SignalCancellation](err); !ok {
			t.Fatalf("Run() error = %v, want SignalCancellation", err)
		}
		if handlerCalled {
			t.Fatal("resolution cancellation admitted handler")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not settle after resolution cancellation")
	}
}

const (
	runSignalNativeInterrupt        = "interrupt"
	runSignalNativeMutated          = "mutated"
	runSignalNativeDynamic          = "dynamic"
	runSignalNativeTerminate        = "terminate"
	runSignalNativeFirst            = "first"
	runSignalAlienctlName           = "alienctl"
	runSignalRunFromProcessChildEnv = "COMMANDMENT_SIGNAL_RUN_FROM_PROCESS_CHILD"
)

type runTokenPairValue struct {
	Left  string
	Right string
}

func runTokenPairCodec() cli.ValueCodec[runTokenPairValue] {
	codec := cli.ValueCodec[runTokenPairValue]{
		Name: "pair-value",
		Parse: func(text string) (runTokenPairValue, error) {
			return runTokenPairValue{}, fmt.Errorf("destination codec must not parse token-language input %q", text)
		},
		Format:   func(value runTokenPairValue) string { return value.Left + ":" + value.Right },
		Snapshot: cli.Value.Assign[runTokenPairValue](),
	}
	return codec
}

func stringsJoinForTest(tokens []string) string { return strings.Join(tokens, "/") }

func TestEscapedMiddlewareNextReturnsProtocolDiagnostic(t *testing.T) {
	t.Parallel()

	applicationErr := errors.New("application middleware failure")
	var escaped cli.MiddlewareNext
	handlerCalls := 0
	middleware := func(_ context.Context, _ *struct{}, next cli.MiddlewareNext) error {
		escaped = next
		return applicationErr
	}
	run := func(context.Context, *struct{}) error {
		handlerCalls++
		return nil
	}
	definition := cli.Definition[struct{}]{
		Name:       "alienctl",
		Middleware: []cli.Middleware[struct{}]{middleware},
		Run:        run,
	}
	app := clitest.Compile(t, definition)
	request := cli.RunRequest{}

	runErr := app.Run(t.Context(), request)
	if !errors.Is(runErr, applicationErr) {
		t.Fatalf("Run() error = %v, want application middleware failure", runErr)
	}
	if escaped == nil {
		t.Fatal("middleware did not expose continuation")
	}
	lateErr := escaped(t.Context())
	if lateErr == nil {
		t.Fatal("late MiddlewareNext call succeeded")
	}
	clitest.Diagnostics(lateErr).Require(t,
		clitest.Diagnostic(cli.DiagnosticMiddlewareProtocolViolation).Present(),
	)
	if handlerCalls != 0 {
		t.Fatalf("handler calls after late continuation = %d, want 0", handlerCalls)
	}
}
