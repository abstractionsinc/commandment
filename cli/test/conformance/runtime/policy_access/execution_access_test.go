package conformance

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

type (
	authoringAccessAccessKind int
	authoringAccessRelation   int
)

type authoringAccessProfileRootConfig struct{ Profile string }

type authoringAccessProfileChildConfig struct{ Profile string }

type authoringAccessProfileChildConfig2 struct{ Profile cli.OptionalValue[string] }

type authoringAccessProfileChildConfig3 struct {
	Profile cli.OptionalValue[cli.ValueState]
}

type authoringAccessProfileLocaleSelectionWorkingDirectoryStdinStdoutStderrChildConfig struct {
	Profile          string
	Locale           cli.Locale
	Selection        cli.CommandSelection
	WorkingDirectory string
	Stdin            io.Reader
	Stdout           io.Writer
	Stderr           io.Writer
}

type authoringAccessSourceValueOptionalStateConfig struct {
	Source   string
	Value    string
	Optional cli.OptionalValue[string]
	State    cli.OptionalValue[cli.ValueState]
}

func TestAccessCapabilityValuePreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type rootConfig = authoringAccessProfileRootConfig
	type childConfig = authoringAccessProfileChildConfig
	var seen string

	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			profile := root.Values.String(&root.Config.Profile)
			root.Flags.ValueFor(profile).Long("profile")
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "child",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					child.Access.Value(profile, &child.Config.Profile)
				},
				Run: func(_ context.Context, config *childConfig) error {
					seen = config.Profile
					return nil
				},
			})
		},
	})
	expectedAccess := clitest.ExecutionAccess{
		Values: []clitest.ValueLocator{clitest.RootCommand().ValueAt(0)},
	}
	clitest.Schema(app).Require(t, clitest.Command("child").ExecutionAccessExactly(expectedAccess))
	if err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"--profile", "prod", "child"},
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if seen != "prod" {
		t.Fatalf("accessed value = %q, want %q", seen, "prod")
	}
}

func TestAccessCapabilityOptionalValuePreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type rootConfig = authoringAccessProfileRootConfig
	type childConfig = authoringAccessProfileChildConfig2

	var seen cli.OptionalValue[string]
	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			profile := root.Values.String(&root.Config.Profile)
			root.Flags.ValueFor(profile).Long("profile")
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "child",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					child.Access.OptionalValue(profile, &child.Config.Profile)
				},
				Run: func(_ context.Context, config *childConfig) error {
					seen = config.Profile
					return nil
				},
			})
		},
	})
	expectedAccess := clitest.ExecutionAccess{
		OptionalValues: []clitest.ValueLocator{clitest.RootCommand().ValueAt(0)},
	}
	clitest.Schema(app).Require(t, clitest.Command("child").ExecutionAccessExactly(expectedAccess))

	if err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"child"},
	}); err != nil {
		t.Fatalf("Run() without value error = %v", err)
	}
	if seen.Present {
		t.Fatalf("optional access without value = %#v, want absent", seen)
	}
	if err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"--profile", "prod", "child"},
	}); err != nil {
		t.Fatalf("Run() with value error = %v", err)
	}
	if !seen.Present || seen.Value != "prod" {
		t.Fatalf("optional access with value = %#v, want present prod", seen)
	}
}

func TestAccessCapabilityValueStatePreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type rootConfig = authoringAccessProfileRootConfig
	type childConfig = authoringAccessProfileChildConfig3
	var seen cli.OptionalValue[cli.ValueState]

	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			profile := root.Values.String(&root.Config.Profile)
			root.Flags.ValueFor(profile).Long("profile")
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "child",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					child.Access.ValueState(profile, &child.Config.Profile)
				},
				Run: func(_ context.Context, config *childConfig) error {
					seen = config.Profile
					return nil
				},
			})
		},
	})
	expectedAccess := clitest.ExecutionAccess{
		ValueStates: []clitest.ValueLocator{clitest.RootCommand().ValueAt(0)},
	}
	clitest.Schema(app).Require(t, clitest.Command("child").ExecutionAccessExactly(expectedAccess))
	if err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"--profile", "prod", "child"},
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !seen.Present || seen.Value.State() == cli.ResolutionStateAbsent || len(seen.Value.Candidates()) == 0 {
		t.Fatalf("value-state access = %#v, want present resolved provenance", seen)
	}
}

func TestStageAccessPreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type rootConfig = authoringAccessProfileRootConfig
	type childConfig = authoringAccessProfileLocaleSelectionWorkingDirectoryStdinStdoutStderrChildConfig

	var seen *childConfig
	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			profile := root.Values.String(&root.Config.Profile)
			root.Flags.ValueFor(profile).Long("profile")
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "child",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					child.Access.Value(profile, &child.Config.Profile)
					child.Access.Locale(&child.Config.Locale)
					child.Access.CommandSelection(&child.Config.Selection)
					child.Access.WorkingDirectory(&child.Config.WorkingDirectory)
					child.Access.Stdin(&child.Config.Stdin)
					child.Access.Stdout(&child.Config.Stdout)
					child.Access.Stderr(&child.Config.Stderr)
				},
				Run: func(_ context.Context, config *childConfig) error {
					seen = config
					return nil
				},
			})
		},
	})
	expectedAccess := clitest.ExecutionAccess{
		Values:           []clitest.ValueLocator{clitest.RootCommand().ValueAt(0)},
		Locale:           true,
		CommandSelection: true,
		WorkingDirectory: true,
		Stdin:            true,
		Stdout:           true,
		Stderr:           true,
	}
	clitest.Schema(app).Require(t, clitest.Command("child").ExecutionAccessExactly(expectedAccess))

	stdin := strings.NewReader("input")
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:             []string{"--profile", "prod", "child"},
		Locale:           "en-US",
		WorkingDirectory: "/workspace",
		Stdin:            stdin,
		Stdout:           stdout,
		Stderr:           stderr,
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if seen == nil || seen.Profile != "prod" || seen.Locale != "en-US" || seen.WorkingDirectory != "/workspace" ||
		seen.Stdin != stdin ||
		seen.Stdout != stdout ||
		seen.Stderr != stderr {
		t.Fatalf("realized access = %#v", seen)
	}
	if len(seen.Selection.Path) != 2 {
		t.Fatalf("command selection = %#v, want root and child", seen.Selection)
	}

	clitest.CompileFails(t, cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			profile := root.Values.String(&root.Config.Profile)
			root.Access.Value(profile, &root.Config.Profile)
		},
	})
}

func TestAccessSelectedAncestorPolarity(t *testing.T) {
	t.Parallel()

	type config = authoringAccessSourceValueOptionalStateConfig
	type accessKind = authoringAccessAccessKind
	const (
		accessValue accessKind = iota
		accessOptionalValue
		accessValueState
	)
	type relation = authoringAccessRelation
	const (
		relationSame relation = iota
		relationAncestor
		relationSibling
		relationDescendant
	)

	apply := func(command *cli.CommandBuilder[config], source cli.Destination[config, string], kind accessKind) {
		switch kind {
		case accessValue:
			command.Access.Value(source, &command.Config.Value)
		case accessOptionalValue:
			command.Access.OptionalValue(source, &command.Config.Optional)
		case accessValueState:
			command.Access.ValueState(source, &command.Config.State)
		}
	}
	definition := func(kind accessKind, relationship relation) cli.Definition[config] {
		return cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				switch relationship {
				case relationSame:
					source := root.Values.String(&root.Config.Source)
					apply(root, source, kind)
				case relationAncestor:
					source := root.Values.String(&root.Config.Source)
					root.Subcommands.Command(cli.Command[config]{
						Name: "child",
						Declare: func(child *cli.CommandBuilder[config]) {
							apply(child, source, kind)
						},
					})
				case relationSibling:
					var source cli.Destination[config, string]
					root.Subcommands.Command(cli.Command[config]{
						Name: "source",
						Declare: func(command *cli.CommandBuilder[config]) {
							source = command.Values.String(&command.Config.Source)
						},
					})
					root.Subcommands.Command(cli.Command[config]{
						Name: "target",
						Declare: func(command *cli.CommandBuilder[config]) {
							apply(command, source, kind)
						},
					})
				case relationDescendant:
					root.Subcommands.Command(cli.Command[config]{
						Name: "parent",
						Declare: func(parent *cli.CommandBuilder[config]) {
							var source cli.Destination[config, string]
							parent.Subcommands.Command(cli.Command[config]{
								Name: "source",
								Declare: func(command *cli.CommandBuilder[config]) {
									source = command.Values.String(&command.Config.Source)
								},
							})
							apply(parent, source, kind)
						},
					})
				}
			},
		}
	}

	tests := []struct {
		name         string
		kind         accessKind
		relationship relation
		valid        bool
	}{
		{"value same command", accessValue, relationSame, false},
		{"value ancestor", accessValue, relationAncestor, true},
		{"value sibling", accessValue, relationSibling, false},
		{"value descendant", accessValue, relationDescendant, false},
		{"optional value same command", accessOptionalValue, relationSame, false},
		{"optional value ancestor", accessOptionalValue, relationAncestor, true},
		{"optional value sibling", accessOptionalValue, relationSibling, false},
		{"optional value descendant", accessOptionalValue, relationDescendant, false},
		{"value state same command", accessValueState, relationSame, true},
		{"value state ancestor", accessValueState, relationAncestor, true},
		{"value state sibling", accessValueState, relationSibling, false},
		{"value state descendant", accessValueState, relationDescendant, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			candidate := definition(test.kind, test.relationship)
			if test.valid {
				clitest.Compile(t, candidate)
				return
			}
			clitest.CompileFails(t, candidate).
				Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
		})
	}
}

type executionAccessProfileRootConfig struct{ Profile string }

type executionAccessProfileSelectionStdoutChildConfig struct {
	Profile   string
	Selection cli.CommandSelection
	Stdout    *strings.Builder
}

type executionAccessRootConfig struct{}

type executionAccessWorkingDirectoryChildConfig struct{ WorkingDirectory string }

type executionAccessLocaleNoiseRootConfig struct {
	Locale string
	Noise  [64]string
}

type executionAccessLocaleChildConfig struct{ Locale cli.Locale }

type executionAccessLocaleRootConfig struct{ Locale cli.Locale }

func TestRealizeExecutionAccessPreservesExecutionSemantics(t *testing.T) {
	t.Parallel()
	type rootConfig = executionAccessProfileRootConfig
	type childConfig = executionAccessProfileSelectionStdoutChildConfig
	var seen *childConfig
	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			profile := root.Values.String(&root.Config.Profile)
			root.Flags.ValueFor(profile).Long("profile")
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "child",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					child.Access.Value(profile, &child.Config.Profile)
					child.Access.CommandSelection(&child.Config.Selection)
				},
				Run: func(_ context.Context, cfg *childConfig) error { seen = cfg; return nil },
			})
		},
	})
	out := &strings.Builder{}
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--profile", "prod", "child"},
		Stdout: out,
	}); err != nil {
		t.Fatal(err)
	}
	if seen == nil || seen.Profile != "prod" || len(seen.Selection.Path) != 2 {
		t.Fatalf("realized access = %#v", seen)
	}
}

func TestSelectedPathNeedsWorkingDirectoryPreservesExecutionSemantics(t *testing.T) {
	t.Parallel()
	type rootConfig = executionAccessRootConfig
	type childConfig = executionAccessWorkingDirectoryChildConfig
	var got string
	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "child",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					child.Access.WorkingDirectory(&child.Config.WorkingDirectory)
				},
				Run: func(_ context.Context, cfg *childConfig) error { got = cfg.WorkingDirectory; return nil },
			})
		},
	})
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:             []string{"child"},
		WorkingDirectory: "/workspace",
	}); err != nil {
		t.Fatal(err)
	}
	if got != "/workspace" {
		t.Fatalf("working directory = %q", got)
	}
}

func TestResolvedLocaleSelectorPreservesExecutionSemantics(t *testing.T) {
	t.Parallel()
	type rootConfig = executionAccessLocaleNoiseRootConfig
	type childConfig = executionAccessLocaleChildConfig
	var got cli.Locale
	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
		},
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			locale := root.Values.String(&root.Config.Locale)
			root.Flags.ValueFor(locale).LocaleSelector().Occurrences(cli.Count.Optional()).
				Long("locale")
			root.Subcommands.Command(cli.Command[childConfig]{
				Name:    "child",
				Declare: func(child *cli.CommandBuilder[childConfig]) { child.Access.Locale(&child.Config.Locale) },
				Run:     func(_ context.Context, cfg *childConfig) error { got = cfg.Locale; return nil },
			})
			root.Subcommands.Command(cli.Command[rootConfig]{
				Name: "bulk",
				Declare: func(bulk *cli.CommandBuilder[rootConfig]) {
					for index := range 64 {
						bulk.Flags.String(&bulk.Config.Noise[index]).Long("noise-" + strconv.Itoa(index))
					}
				},
			})
		},
	})
	if err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"--locale", "fr", "child"},
	}); err != nil {
		t.Fatal(err)
	}
	if got != "fr" {
		t.Fatalf("locale = %q", got)
	}
}

func TestRealizeExecutionAccessPreservesEmptyWorkingDirectory(t *testing.T) {
	t.Parallel()
	type rootConfig = executionAccessRootConfig
	type childConfig = executionAccessWorkingDirectoryChildConfig
	var got string
	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "child",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					child.Access.WorkingDirectory(&child.Config.WorkingDirectory)
				},
				Run: func(_ context.Context, cfg *childConfig) error {
					got = cfg.WorkingDirectory
					return nil
				},
			})
		},
	})
	if err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"child"},
	}); err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("working directory = %q, want empty", got)
	}
}

func TestResolvedLocaleSelectorAcceptsLocaleValues(t *testing.T) {
	t.Parallel()
	type rootConfig = executionAccessLocaleRootConfig
	type childConfig = executionAccessLocaleChildConfig
	var got cli.Locale
	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
		},
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			locale := root.Values.Value(&root.Config.Locale, cli.ValueCodec[cli.Locale]{
				Name:     "locale",
				Parse:    func(text string) (cli.Locale, error) { return cli.Locale(text), nil },
				Format:   func(value cli.Locale) string { return string(value) },
				Snapshot: cli.Value.Assign[cli.Locale](),
			})
			root.Flags.ValueFor(locale).LocaleSelector().Occurrences(cli.Count.Optional()).
				Long("locale")
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "child",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					child.Access.Locale(&child.Config.Locale)
				},
				Run: func(_ context.Context, cfg *childConfig) error {
					got = cfg.Locale
					return nil
				},
			})
		},
	})
	if err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"--locale", "fr", "child"},
	}); err != nil {
		t.Fatal(err)
	}
	if got != "fr" {
		t.Fatalf("locale = %q, want fr", got)
	}
}

type (
	executionAccessStdinConfig  struct{ Stdin io.Reader }
	executionAccessStdoutConfig struct{ Stdout io.Writer }
	executionAccessStderrConfig struct{ Stderr io.Writer }
)

func TestAbsentStdinRemainsValidExecutionAccess(t *testing.T) {
	t.Parallel()

	type config = executionAccessStdinConfig
	called := false
	declare := func(root *cli.CommandBuilder[config]) {
		root.Access.Stdin(&root.Config.Stdin)
	}
	run := func(_ context.Context, config *config) error {
		called = true
		if config.Stdin != nil {
			t.Fatalf("stdin = %#v, want nil", config.Stdin)
		}
		return nil
	}
	definition := cli.Definition[config]{
		Name:    "alienctl",
		Declare: declare,
		Run:     run,
	}
	app := clitest.Compile(t, definition)
	request := cli.RunRequest{}

	if err := app.Run(t.Context(), request); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !called {
		t.Fatal("Run handler was not called")
	}
}

func TestAbsentStdoutRemainsValidExecutionAccess(t *testing.T) {
	t.Parallel()

	type config = executionAccessStdoutConfig
	called := false
	declare := func(root *cli.CommandBuilder[config]) {
		root.Access.Stdout(&root.Config.Stdout)
	}
	run := func(_ context.Context, config *config) error {
		called = true
		if config.Stdout != nil {
			t.Fatalf("stdout = %#v, want nil", config.Stdout)
		}
		return nil
	}
	definition := cli.Definition[config]{
		Name:    "alienctl",
		Declare: declare,
		Run:     run,
	}
	app := clitest.Compile(t, definition)
	request := cli.RunRequest{}

	if err := app.Run(t.Context(), request); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !called {
		t.Fatal("Run handler was not called")
	}
}

func TestAbsentStderrRemainsValidExecutionAccess(t *testing.T) {
	t.Parallel()

	type config = executionAccessStderrConfig
	called := false
	declare := func(root *cli.CommandBuilder[config]) {
		root.Access.Stderr(&root.Config.Stderr)
	}
	run := func(_ context.Context, config *config) error {
		called = true
		if config.Stderr != nil {
			t.Fatalf("stderr = %#v, want nil", config.Stderr)
		}
		return nil
	}
	definition := cli.Definition[config]{
		Name:    "alienctl",
		Declare: declare,
		Run:     run,
	}
	app := clitest.Compile(t, definition)
	request := cli.RunRequest{}

	if err := app.Run(t.Context(), request); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !called {
		t.Fatal("Run handler was not called")
	}
}
