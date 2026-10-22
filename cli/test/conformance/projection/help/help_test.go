package conformance

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

type authoringHelpGroupVerboseConfig struct{ Verbose bool }

func declareAuthoringHelpGroupFixture(root *cli.CommandBuilder[authoringHelpGroupVerboseConfig]) {
	root.Flags.HelpGroup(
		"Options",
		func(flags *cli.Flags[authoringHelpGroupVerboseConfig]) {
			flags.Bool(&root.Config.Verbose).Long("verbose")
		},
	).OrderByName()
	root.Subcommands.HelpGroup("Commands", func(subcommands *cli.Subcommands[authoringHelpGroupVerboseConfig]) {
		subcommands.Command(cli.Command[struct{}]{
			Name: "child",
		})
	}).OrderByName()
}

func TestHelpGroupOptionPreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringHelpGroupVerboseConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringHelpGroupFixture,
	})
	clitest.Schema(app).
		Require(t, clitest.RootCommand().HelpFlagGroup("Options").Kind(cli.HelpGroupDeclared), clitest.RootCommand().HelpCommandGroup("Commands").Kind(cli.HelpGroupDeclared))
}

func TestHelpGroupOrderPreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringHelpGroupVerboseConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringHelpGroupFixture,
	})
	clitest.Schema(app).
		Require(t, clitest.RootCommand().HelpFlagGroup("Options").Kind(cli.HelpGroupDeclared), clitest.RootCommand().HelpCommandGroup("Commands").Kind(cli.HelpGroupDeclared))
}

func TestStageHelpGroupPreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringHelpGroupVerboseConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringHelpGroupFixture,
	})
	clitest.Schema(app).
		Require(t, clitest.RootCommand().HelpFlagGroup("Options").Kind(cli.HelpGroupDeclared), clitest.RootCommand().HelpCommandGroup("Commands").Kind(cli.HelpGroupDeclared))
}

func TestDeclareSubcommandHelpGroupPreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringHelpGroupVerboseConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringHelpGroupFixture,
	})
	clitest.Schema(app).
		Require(t, clitest.RootCommand().HelpFlagGroup("Options").Kind(cli.HelpGroupDeclared), clitest.RootCommand().HelpCommandGroup("Commands").Kind(cli.HelpGroupDeclared))
}

func TestDeclareFlagHelpGroupPreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringHelpGroupVerboseConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringHelpGroupFixture,
	})
	clitest.Schema(app).
		Require(t, clitest.RootCommand().HelpFlagGroup("Options").Kind(cli.HelpGroupDeclared), clitest.RootCommand().HelpCommandGroup("Commands").Kind(cli.HelpGroupDeclared))
}

func TestHelpGroupAcceptancePreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringHelpGroupVerboseConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringHelpGroupFixture,
	})
	clitest.Schema(app).
		Require(t, clitest.RootCommand().HelpFlagGroup("Options").Kind(cli.HelpGroupDeclared), clitest.RootCommand().HelpCommandGroup("Commands").Kind(cli.HelpGroupDeclared))
}

func helpMustCompilePublic[C any](t testing.TB, definition cli.Definition[C]) cli.Application {
	t.Helper()
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return app
}

type alienctlHelpConfig struct {
	Verbose bool
}

type alienctlAbductConfig struct {
	Target  string
	Mode    string
	Profile string
	Secret  string
}

func alienctlHelpDefinition(explicit bool) cli.Definition[alienctlHelpConfig] {
	definition := cli.Definition[alienctlHelpConfig]{
		Name: "alienctl",
		Description: cli.Description{
			Long: "Coordinate peaceful first-contact field operations.",
		},
		Release: cli.Release.Ordered("2", "1", "2"),
		Examples: []cli.Example{{
			Description: "Inspect the available field operations.",
			CommandLine: "alienctl help abduct",
		}},
		Declare: func(root *cli.CommandBuilder[alienctlHelpConfig]) {
			root.Flags.Bool(&root.Config.Verbose).Description("Show operational detail.").Placement(cli.FlagPlacementSubtree).
				Long("verbose")

			root.Subcommands.HelpGroup("Field operations", func(commands *cli.Subcommands[alienctlHelpConfig]) {
				commands.Command(cli.Command[alienctlAbductConfig]{
					Name:    "abduct",
					Aliases: cli.Aliases("extract"),
					Description: cli.Description{
						Long: "Acquire one specimen using a declared extraction method.",
					},
					Examples: []cli.Example{{
						Description: "Use a tractor beam for specimen ZQ-17.",
						CommandLine: "alienctl abduct ZQ-17 --mode=tractor",
					}},
					Declare: func(command *cli.CommandBuilder[alienctlAbductConfig]) {
						modeDomain := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
							tractor := values.Value(
								"tractor",
								"tractor",
							).Description("Use the tractor beam.")

							legacy := values.Value("teleport", "teleport").Description("Use the legacy transporter.").
								Release(cli.ReleaseLifecycle{
									DeprecatedIn: "2",
									Reason:       "transporter extraction is being retired",
								})
							warp := values.Value(
								"warp",
								"warp",
							).Description("Use a controlled warp corridor.")

							legacy.ReplacedBy(warp)
							values.Alias("beam", tractor)
							values.Value("classified", "classified").Hidden()
						})
						mode := command.Values.Value(&command.Config.Mode, modeDomain)
						method := command.Flags.ValueFor(
							mode,
						).ValueName("MODE").Description("Select the extraction method.").Required()

						old := method.Long("method").Release(cli.ReleaseLifecycle{
							DeprecatedIn: "2",
							Reason:       "use the neutral mode spelling",
						})
						current := method.Long("mode")
						old.ReplacedBy(current)
						method.Long("tractor").FixedValue("tractor")

						command.Flags.String(&command.Config.Profile).ValueName("PROFILE").Description("Select an observation profile.").
							Long("profile").
							ImplicitValue("standard")
						command.Flags.String(&command.Config.Secret).ValueName("TOKEN").Description("Authenticate the field link.").
							Long("token").
							ImplicitValue("classified", cli.Source.Sensitive()).
							Sensitive()
						command.Arguments.Form(func(form *cli.ArgumentFormBuilder[alienctlAbductConfig]) {
							form.String(&form.Config.Target, "TARGET").Description("Specimen identifier.")
						})
						command.Shortcut(cli.CommandShortcut{
							Name: "sample",
							Description: cli.Description{
								Short: "Acquire the standard sample specimen.",
							},
							Args: []string{"ZQ-17", "--mode=tractor"},
						})
					},
					Run: func(context.Context, *alienctlAbductConfig) error { return nil },
				})
			})
		},
	}
	if explicit {
		definition.Help = cli.HelpConfig{
			Renderer: cli.DefaultHelp(cli.DefaultHelpConfig{}),
		}
	}
	return definition
}

func TestDefaultHelpAlienctlZeroConfigurationMatchesExplicitFactory(t *testing.T) {
	t.Parallel()
	zero := clitest.Compile(t, alienctlHelpDefinition(false))
	explicit := clitest.Compile(t, alienctlHelpDefinition(true))
	var zeroOutput strings.Builder
	if err := zero.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--help"},
		Stdout: &zeroOutput,
	}); err != nil {
		t.Fatal(err)
	}
	var explicitOutput strings.Builder
	if err := explicit.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--help"},
		Stdout: &explicitOutput,
	}); err != nil {
		t.Fatal(err)
	}
	if zeroOutput.String() != explicitOutput.String() {
		t.Fatalf(
			"zero HelpAction and explicit DefaultHelp diverged:\nzero:\n%s\nexplicit:\n%s",
			zeroOutput.String(),
			explicitOutput.String(),
		)
	}
}

func TestDefaultHelpAlienctlRichCommandFormation(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, alienctlHelpDefinition(false))
	var root strings.Builder
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--help"},
		Stdout: &root,
	}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Coordinate peaceful first-contact field operations.",
		"Usage",
		"Field operations",
		"abduct",
		"extract",
		"[alias]",
		"sample",
		"[shortcut]",
		"help",
		"Show help.",
		"Examples",
		"alienctl help abduct",
	} {
		if !strings.Contains(root.String(), want) {
			t.Fatalf("root Help missing %q:\n%s", want, root.String())
		}
	}

	var command strings.Builder
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"help", "abduct"},
		Stdout: &command,
	}); err != nil {
		t.Fatal(err)
	}
	text := command.String()
	for _, want := range []string{
		"Acquire one specimen using a declared extraction method.",
		"alienctl abduct [flags] <TARGET>",
		"Arguments",
		"<TARGET>",
		"Specimen identifier.",
		"--method MODE",
		"--method=MODE",
		"deprecated 2: use the neutral mode spelling",
		"use --mode",
		"--mode MODE",
		"--mode=MODE",
		"--tractor",
		"Fixed value for --tractor: tractor",
		"[required]",
		"MODE: tractor - Use the tractor beam.",
		"beam (alias for tractor)",
		"--profile[=PROFILE]",
		"Implicit value for --profile: standard",
		"Global flags",
		"--verbose",
		"Examples",
		"alienctl abduct ZQ-17 --mode=tractor",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("command Help missing %q:\n%s", want, text)
		}
	}
	normalized := strings.Join(strings.Fields(text), " ")
	if want := "teleport [deprecated 2: transporter extraction is being retired; use warp] - Use the legacy transporter."; !strings.Contains(
		normalized,
		want,
	) {
		t.Fatalf("command Help missing wrapped semantic detail %q:\n%s", want, text)
	}
	for _, forbidden := range []string{"classified", "Fixed value for --verbose: true"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("command Help leaked/noised %q:\n%s", forbidden, text)
		}
	}
}

func TestDefaultHelpAlienctlPreservesFormBoundariesAndSafeClosure(t *testing.T) {
	t.Parallel()
	const (
		legacySpelling  cli.FlagSpellingID     = 1
		currentSpelling cli.FlagSpellingID     = 2
		hiddenSpelling  cli.FlagSpellingID     = 3
		shortSpelling   cli.FlagSpellingID     = 4
		tractorValue    cli.ValueDomainEntryID = 10
		beamValue       cli.ValueDomainEntryID = 11
		hiddenValue     cli.ValueDomainEntryID = 12
	)
	page := cli.HelpPage{
		ApplicationName: "alienctl",
		Labels: cli.HelpLabels{
			Usage:         "Usage",
			Commands:      "Commands",
			Arguments:     "Arguments",
			Flags:         "Flags",
			GlobalFlags:   "Global flags",
			Aliases:       "Aliases",
			Environment:   "Environment",
			Default:       "default",
			Required:      "required",
			Optional:      "optional",
			Deprecated:    "deprecated",
			Examples:      "Examples",
			Alias:         "alias",
			Shortcut:      "shortcut",
			Values:        "Values",
			Keys:          "Keys",
			Syntax:        "Syntax",
			AliasFor:      "alias for",
			Use:           "use",
			ImplicitValue: "Implicit value",
			FixedValue:    "Fixed value",
		},
		SelectedPath: []string{"scan"},
		Description: cli.Description{
			Long: "Scan a sector and classify encountered objects.",
		},
		Release: cli.ReleaseLifecycle{
			DeprecatedIn: "3",
			Reason:       "use survey for new missions",
		},
		LifecycleState: cli.LifecycleStateDeprecated,
		Runnable:       true,
		Usage: []string{
			"alienctl scan [flags] <SECTOR>",
			"alienctl scan [flags] object <OBJECT...>",
		},
		ArgumentForms: []cli.HelpArgumentForm{
			{
				Terms: []cli.HelpArgumentTerm{
					{
						Name:        "SECTOR",
						Description: "Sector coordinate.",
						Count:       cli.Count.Exactly(1),
					},
				},
			},
			{
				Terms: []cli.HelpArgumentTerm{
					{
						Literal: "object",
						Count:   cli.Count.Exactly(1),
					},
					{
						Name:        "OBJECT",
						Description: "Object identifiers.",
						Count:       cli.Count.OneOrMore(),
					},
				},
			},
		},
		FlagGroups: []cli.HelpFlagGroup{{
			Title: "Scan options",
			Flags: []cli.HelpFlag{{
				FlagSpellings: []cli.FlagSpellingID{legacySpelling, currentSpelling, shortSpelling},
				ValueName:     "MODE",
				Description:   "Select scanner mode.",
				ValueDomainUses: []cli.HelpValueDomainUse{{
					Kind:    cli.ValueDomainUseDestination,
					Entries: []cli.ValueDomainEntryID{tractorValue, beamValue},
				}},
			}},
		}, {
			Title: "Secrets",
			Flags: []cli.HelpFlag{{
				Sensitive:     true,
				FlagSpellings: []cli.FlagSpellingID{hiddenSpelling},
				ValueName:     "TOKEN",
				Description:   "Authenticate the scanner.",
				ValueDomainUses: []cli.HelpValueDomainUse{{
					Kind:    cli.ValueDomainUseDestination,
					Entries: []cli.ValueDomainEntryID{hiddenValue},
				}},
			}},
		}},
		FlagSpellings: map[cli.FlagSpellingID]cli.HelpFlagSpelling{
			legacySpelling: {
				ID: legacySpelling,
				Spelling: cli.FlagSpelling{
					Kind:       cli.FlagSpellingLong,
					Text:       "--method",
					ValueMode:  cli.FlagValueModeRequired,
					ValueForms: []cli.FlagValueForm{cli.FlagValueFormSeparate, cli.FlagValueFormAssigned},
				},
				Release: cli.ReleaseLifecycle{
					DeprecatedIn: "2",
					Reason:       "use mode",
				}, LifecycleState: cli.LifecycleStateDeprecated,
				Replacement: hiddenSpelling,
			},
			currentSpelling: {
				ID: currentSpelling,
				Spelling: cli.FlagSpelling{
					Kind:       cli.FlagSpellingLong,
					Text:       "--mode",
					ValueMode:  cli.FlagValueModeRequired,
					ValueForms: []cli.FlagValueForm{cli.FlagValueFormAssigned},
				},
			},
			shortSpelling: {
				ID: shortSpelling,
				Spelling: cli.FlagSpelling{
					Kind:       cli.FlagSpellingShort,
					Text:       "-m",
					ValueMode:  cli.FlagValueModeRequired,
					ValueForms: []cli.FlagValueForm{cli.FlagValueFormJoined, cli.FlagValueFormSeparate},
				},
			},
			hiddenSpelling: {
				ID: hiddenSpelling,
				Spelling: cli.FlagSpelling{
					Kind:       cli.FlagSpellingLong,
					Text:       "--classified",
					ValueMode:  cli.FlagValueModeOptional,
					ValueForms: []cli.FlagValueForm{cli.FlagValueFormAssigned},
					Sensitive:  true,
					ImplicitValue: &cli.ValueLiteral{
						Type: cli.ValueType{
							Kind: cli.ValueKindString,
						},
						Text:      "ultraviolet",
						Sensitive: true,
					},
				},
			},
		},
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.HelpValueDomainEntry{
			tractorValue: {
				ID:          tractorValue,
				Kind:        cli.ValueDomainEntryCanonical,
				Name:        "tractor",
				Description: "Tractor scan.",
			},
			beamValue: {
				ID:          beamValue,
				Kind:        cli.ValueDomainEntryAlias,
				Name:        "beam",
				Canonical:   tractorValue,
				Replacement: hiddenValue,
			},
			hiddenValue: {
				ID:          hiddenValue,
				Kind:        cli.ValueDomainEntryCanonical,
				Name:        "ultraviolet",
				Description: "classified scan",
			},
		},
		Examples: []cli.Example{
			{
				Description: "Preserve an opaque operator note exactly.",
				CommandLine: "alienctl scan A-17 --note='two  spaces'",
			},
		},
	}

	plain, err := cli.DefaultHelp(cli.DefaultHelpConfig{})(t.Context(), page)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"deprecated 3: use survey for new missions",
		"alienctl scan [flags] <SECTOR>",
		"alienctl scan [flags] object <OBJECT...>",
		"<SECTOR>",
		"object",
		"<OBJECT...>",
		"--method MODE",
		"--method=MODE",
		"[deprecated 2: use mode]",
		"--mode=MODE",
		"-mMODE",
		"-m MODE",
		"MODE: tractor - Tractor scan., beam (alias for tractor)",
		"  alienctl scan A-17 --note='two  spaces'",
	} {
		if !strings.Contains(plain, want) {
			t.Fatalf("manual Help missing %q:\n%s", want, plain)
		}
	}
	for _, forbidden := range []string{"--classified", "ultraviolet", "classified scan", "use --classified"} {
		if strings.Contains(plain, forbidden) {
			t.Fatalf("closure/sensitivity leak %q:\n%s", forbidden, plain)
		}
	}
	firstUsage := strings.Index(plain, "alienctl scan [flags] <SECTOR>")
	firstTerm := strings.Index(plain, "<SECTOR>")
	secondUsage := strings.Index(plain, "alienctl scan [flags] object <OBJECT...>")
	secondTerm := strings.LastIndex(plain, "<OBJECT...>")
	if firstUsage < 0 || firstTerm <= firstUsage || secondUsage <= firstTerm || secondTerm <= secondUsage {
		t.Fatalf("argument form boundary/order lost:\n%s", plain)
	}
}

func TestDefaultHelpAlienctlCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	text, err := cli.DefaultHelp(cli.DefaultHelpConfig{})(ctx, cli.HelpPage{
		ApplicationName: "alienctl",
		Usage:           []string{"alienctl"},
	})
	if !errors.Is(err, context.Canceled) || text != "" {
		t.Fatalf("canceled DefaultHelp = %q, %v", text, err)
	}
}

func TestDefaultHelpAlienctlEscapesPresentationControls(t *testing.T) {
	t.Parallel()
	page := cli.HelpPage{
		ApplicationName: "alienctl",
		Description: cli.Description{
			Long: "safe\x1b[31m description",
		},
		Usage: []string{"alienctl scan\t<TARGET>"},
		Examples: []cli.Example{{
			CommandLine: "alienctl scan A-17\t\x1b[2J",
		}},
	}
	text, err := cli.DefaultHelp(cli.DefaultHelpConfig{})(t.Context(), page)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(text, "\x1b") || !strings.Contains(text, `safe\x1b[31m description`) ||
		!strings.Contains(text, `alienctl scan\x09<TARGET>`) ||
		!strings.Contains(text, `alienctl scan A-17\x09\x1b[2J`) {
		t.Fatalf("Help control-character escaping = %q", text)
	}
}

func TestDefaultHelpConfiguredSectionsSelectPresentationOnly(t *testing.T) {
	t.Parallel()
	definition := alienctlHelpDefinition(false)
	definition.Help.Renderer = cli.DefaultHelp(cli.DefaultHelpConfig{
		Sections: []cli.HelpSection{cli.HelpSectionUsage},
	})
	app := clitest.Compile(t, definition)
	var output strings.Builder
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--help"},
		Stdout: &output,
	}); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if !strings.Contains(text, "Usage") {
		t.Fatalf("configured Help missing Usage section:\n%s", text)
	}
	if strings.Contains(text, "Coordinate peaceful first-contact field operations.") ||
		strings.Contains(text, "Field operations") || strings.Contains(text, "Examples") {
		t.Fatalf("configured Help rendered unselected sections:\n%s", text)
	}
}

type helpProfileGateUnavailableInvisibleRemovedCurrentConfig struct {
	Profile     string
	Gate        string
	Unavailable string
	Invisible   string
	Removed     string
	Current     string
}

type helpModeUnusedVisibleUnrelatedConfig struct {
	Mode      string
	Unused    string
	Visible   string
	Unrelated string
}

type helpProfileSourceGateVisibleConfig struct {
	Profile string
	Source  string
	Gate    string
	Visible string
}

type helpModeVisibleConfig struct {
	Mode    string
	Visible string
}

type helpZetaAlphaConfig struct {
	Zeta  bool
	Alpha bool
}

type helpZetaAlphaPlainConfig struct {
	Zeta  bool
	Alpha bool
	Plain bool
}

type helpEnabledConfig struct{ Enabled bool }

type helpModeConfig struct{ Mode string }

type helpNameItemsConfig struct {
	Name  string
	Items []string
}

type helpValueItemsConfig struct {
	Value string
	Items []string
}

type helpNameModeConfig struct {
	Name string
	Mode string
}

type helpSecretConfig struct {
	Secret string
}

type helpItemsPairConfig struct {
	Items []string
	Pair  string
}

type helpConfig struct{}

func TestRunHelpProjectsSelectedTargetWithoutBusinessExecution(t *testing.T) {
	t.Parallel()
	type config = helpConfig

	for _, test := range []struct {
		name string
		args []string
		path []string
	}{
		{
			name: "ordinary flag",
			args: []string{"--help"},
		},
		{
			name: "trailing command path",
			args: []string{"help", "child"},
			path: []string{"child"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var output strings.Builder
			var pages []cli.HelpPage
			middlewareCalls := 0
			handlerCalls := 0
			noticeCalls := 0
			factSupplyCalls := 0
			source := &helpCountingSignalSource{}

			definition := cli.Definition[config]{
				Name:    "tool",
				Release: cli.Release.SemVer("2.0.0"),
				Help: cli.HelpConfig{
					Renderer: func(ctx context.Context, page cli.HelpPage) (string, error) {
						if cli.EffectiveLocale(ctx) != page.Locale {
							t.Fatalf(
								"renderer locale context = %q, page locale = %q",
								cli.EffectiveLocale(ctx),
								page.Locale,
							)
						}
						pages = append(pages, page)
						return "help\n", nil
					},
				},
				Declare: func(root *cli.CommandBuilder[config]) {
					registeredSignal := root.Signals.Source(source.Source()).Signal("counted")
					root.Signals.Policy(
						root.Signals.Match.On(registeredSignal).During(cli.SignalPhaseExecution).Each(),
						root.Signals.Action.Ignore(),
					)
					root.Advisory.Always(cli.Advisory{
						Code:     "must-not-deliver",
						Severity: cli.NoticeSeverityWarning,
						Message:  "must not deliver",
					})
					root.Subcommands.Command(cli.Command[config]{
						Name: "child",
						Middleware: []cli.Middleware[config]{
							func(ctx context.Context, _ *config, next cli.MiddlewareNext) error {
								middlewareCalls++
								return next(ctx)
							},
						},
						Run: func(context.Context, *config) error {
							handlerCalls++
							return nil
						},
					})
				},
				Middleware: []cli.Middleware[config]{
					func(ctx context.Context, _ *config, next cli.MiddlewareNext) error {
						middlewareCalls++
						return next(ctx)
					},
				},
				Run: func(context.Context, *config) error {
					handlerCalls++
					return nil
				},
			}
			app, err := cli.Compile(definition)
			if err != nil {
				t.Fatalf("Compile() error = %v", err)
			}
			referenceProof := clitest.Reference(app, cli.ReferenceOptions{})
			referenceProof.Require(t, clitest.RootCommand().Exists())

			runRequest := cli.RunRequest{
				Args:   test.args,
				Stdout: &output,
				Facts: []cli.FactProvider{cli.FactsFor[config](func(context.Context, *cli.FactSupply[config]) error {
					factSupplyCalls++
					return nil
				})},
				Notify: func(context.Context, cli.Notice) error {
					noticeCalls++
					return nil
				},
			}
			err = app.Run(t.Context(), runRequest)
			if err != nil {
				t.Fatalf("Run(%v) error = %v", test.args, err)
			}
			if output.String() != "help\n" || len(pages) != 1 {
				t.Fatalf("help output/pages = %q / %d", output.String(), len(pages))
			}
			page := pages[0]
			if strings.Join(page.SelectedPath, "/") != strings.Join(test.path, "/") {
				t.Fatalf("selected path = %#v, want %#v", page.SelectedPath, test.path)
			}
			if middlewareCalls != 0 || handlerCalls != 0 || noticeCalls != 0 {
				t.Fatalf(
					"Help entered business execution: middleware=%d handler=%d notices=%d",
					middlewareCalls,
					handlerCalls,
					noticeCalls,
				)
			}
			if source.open != 0 {
				t.Fatalf("Help entered signal runtime: open=%d", source.open)
			}
			if factSupplyCalls != 0 {
				t.Fatalf("static Help demanded unrelated Fact supply %d times", factSupplyCalls)
			}

			// The renderer owns a detached page. Mutation after rendering must not alter
			// the compiled application or a later independent Help operation.
			for groupIndex := range pages[0].CommandGroups {
				for commandIndex := range pages[0].CommandGroups[groupIndex].Commands {
					if pages[0].CommandGroups[groupIndex].Commands[commandIndex].Name == "child" {
						pages[0].CommandGroups[groupIndex].Commands[commandIndex].Name = "mutated"
					}
				}
			}
			output.Reset()
			if err := app.Run(t.Context(), cli.RunRequest{
				Args:   []string{"--help"},
				Stdout: &output,
			}); err != nil {
				t.Fatalf("second Run(--help) error = %v", err)
			}
			names := helpTestCommandNames(pages[len(pages)-1])
			_, hasChild := names["child"]
			_, hasMutated := names["mutated"]
			if !hasChild || hasMutated {
				t.Fatalf("renderer mutation leaked across operations: %#v", pages[len(pages)-1].CommandGroups)
			}
		})
	}
}

func TestRunAutomaticHelpActivatesAfterCompleteDefaultSelectionChain(t *testing.T) {
	t.Parallel()
	type config = helpConfig
	var output strings.Builder
	var pages []cli.HelpPage
	handlerCalls := 0
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			WhenEmpty: true,
			Renderer: func(_ context.Context, page cli.HelpPage) (string, error) {
				pages = append(pages, page)
				return "automatic help\n", nil
			},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.Command(cli.Command[config]{
				Name: "branch",
				Declare: func(branch *cli.CommandBuilder[config]) {
					branch.Subcommands.Command(cli.Command[config]{
						Name: "leaf",
						Run: func(context.Context, *config) error {
							handlerCalls++
							return nil
						},
					}).Default()
				},
			}).Default()
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	referenceProof := clitest.Reference(app, cli.ReferenceOptions{})
	referenceProof.Require(t, clitest.RootCommand().Exists())

	if err := app.Run(t.Context(), cli.RunRequest{
		Stdout: &output,
	}); err != nil {
		t.Fatalf("Run(empty) error = %v", err)
	}
	if output.String() != "automatic help\n" || len(pages) != 1 {
		t.Fatalf("automatic Help output/pages = %q / %d", output.String(), len(pages))
	}
	if got := strings.Join(pages[0].SelectedPath, "/"); got != "branch/leaf" {
		t.Fatalf("automatic Help selected path = %q, want %q", got, "branch/leaf")
	}
	if handlerCalls != 0 {
		t.Fatalf("empty invocation executed default-selected handler %d times before Help", handlerCalls)
	}

	output.Reset()
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"branch", "leaf"},
		Stdout: &output,
	}); err != nil {
		t.Fatalf("Run(explicit runnable path) error = %v", err)
	}
	if len(pages) != 1 || output.Len() != 0 {
		t.Fatalf("WhenEmpty activated for non-empty runnable input: pages=%d output=%q", len(pages), output.String())
	}
	if handlerCalls != 1 {
		t.Fatalf("explicit runnable path handler calls = %d, want 1", handlerCalls)
	}
}

func TestRunAutomaticStructuralHelpUsesFinalDefaultSelectedCommand(t *testing.T) {
	t.Parallel()
	type config = helpConfig
	var output strings.Builder
	var pages []cli.HelpPage
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Renderer: func(_ context.Context, page cli.HelpPage) (string, error) {
				pages = append(pages, page)
				return "structural help\n", nil
			},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.Command(cli.Command[config]{
				Name: "branch",
				Declare: func(branch *cli.CommandBuilder[config]) {
					branch.Subcommands.Command(cli.Command[config]{
						Name: "leaf",
						Declare: func(leaf *cli.CommandBuilder[config]) {
							leaf.Subcommands.Command(cli.Command[config]{
								Name: "nested",
								Run:  func(context.Context, *config) error { return nil },
							})
						},
					}).Default()
				},
			}).Default()
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	referenceProof := clitest.Reference(app, cli.ReferenceOptions{})
	referenceProof.Require(t, clitest.RootCommand().Exists())

	for _, args := range [][]string{nil, {"branch"}} {
		output.Reset()
		before := len(pages)
		if err := app.Run(t.Context(), cli.RunRequest{
			Args:   args,
			Stdout: &output,
		}); err != nil {
			t.Fatalf("Run(%v) error = %v", args, err)
		}
		if output.String() != "structural help\n" || len(pages) != before+1 {
			t.Fatalf("Run(%v) structural Help output/pages = %q / %d", args, output.String(), len(pages)-before)
		}
		if got := strings.Join(pages[len(pages)-1].SelectedPath, "/"); got != "branch/leaf" {
			t.Fatalf("Run(%v) selected path = %q, want %q", args, got, "branch/leaf")
		}
	}
}

func TestRunHelpProviderTargetAndImmediateNamespaceShareOperationSnapshot(t *testing.T) {
	t.Parallel()
	type config = helpConfig
	rootProviderCalls := 0
	immediateProviderCalls := 0
	descendantProviderCalls := 0
	var page cli.HelpPage
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Renderer: func(_ context.Context, observed cli.HelpPage) (string, error) {
				page = observed
				return "provider help", nil
			},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				rootProviderCalls++
				batch.Command(cli.Command[config]{
					Name: "remote",
					Declare: func(remote *cli.CommandBuilder[config]) {
						remote.Subcommands.HelpGroup("Remote commands", func(subcommands *cli.Subcommands[config]) {
							subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
								immediateProviderCalls++
								batch.Command(cli.Command[config]{
									Name: "leaf",
									Declare: func(leaf *cli.CommandBuilder[config]) {
										leaf.Subcommands.CommandsFrom(
											func(_ context.Context, batch *cli.CommandBatch[config]) error {
												descendantProviderCalls++
												batch.Command(cli.Command[config]{
													Name: "deep",
												})
												return nil
											},
										)
									},
								})
								return nil
							})
						})
					},
				})
				return nil
			})
		},
	}
	app := clitest.Compile(t, definition)

	expectHelp := func() {
		clitest.RequireHelpPresentation(t, page,
			clitest.HelpSelectedPathExactly("remote"),
			clitest.HelpCommandGroup("Remote commands").Kind(cli.HelpGroupDeclared),
			clitest.HelpCommandGroup("Remote commands").CommandCount(1),
			clitest.HelpCommandGroup("Remote commands").Command("leaf").Kind(cli.HelpCommandCanonical),
			clitest.HelpCommandGroup("Remote commands").Command("leaf").Origin(cli.DeclarationOriginProvider),
			clitest.HelpCommandGroup("Remote commands").Command("leaf").HasSubcommands(),
			clitest.HelpCommand("deep").DoesNotExist(),
		)
	}

	var output strings.Builder
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"help", "remote"},
		Stdout: &output,
	}); err != nil {
		t.Fatalf("Run(help remote) error = %v", err)
	}
	expectHelp()
	if rootProviderCalls != 1 || immediateProviderCalls != 1 || descendantProviderCalls != 0 {
		t.Fatalf(
			"provider calls = root %d immediate %d descendant %d",
			rootProviderCalls,
			immediateProviderCalls,
			descendantProviderCalls,
		)
	}

	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"remote"},
	}).Require(t,
		clitest.InspectionReady(),
		clitest.RootCommand().Command("remote").Selected(),
	)
	if rootProviderCalls != 2 || immediateProviderCalls != 2 || descendantProviderCalls != 0 {
		t.Fatalf(
			"Help/Inspection provider demand isolation = root %d immediate %d descendant %d",
			rootProviderCalls,
			immediateProviderCalls,
			descendantProviderCalls,
		)
	}

	output.Reset()
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"help", "remote"},
		Stdout: &output,
	}); err != nil {
		t.Fatalf("second Run(help remote) error = %v", err)
	}
	expectHelp()
	if rootProviderCalls != 3 || immediateProviderCalls != 3 || descendantProviderCalls != 0 {
		t.Fatalf(
			"second Help provider demand = root %d immediate %d descendant %d",
			rootProviderCalls,
			immediateProviderCalls,
			descendantProviderCalls,
		)
	}
}

func TestRunHelpAppliesAvailabilityVisibilityLifecycleProfileAndLocale(t *testing.T) {
	t.Parallel()
	type config = helpProfileGateUnavailableInvisibleRemovedCurrentConfig
	var page cli.HelpPage
	definition := cli.Definition[config]{
		Name: "tool",
		Description: cli.Description{
			Short: "tool short",
		},
		Release: cli.Release.Ordered("2", "1", "2"),
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"tool short": "outil court",
				},
			}},
		},
		Help: cli.HelpConfig{
			Renderer: func(_ context.Context, observed cli.HelpPage) (string, error) {
				page = observed
				return "help", nil
			},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			profileDomain := cli.Value.Domain(cli.Value.Assign[string](), func(domain *cli.ValueDomainBuilder[string]) {
				domain.Value("prod", "prod")
				domain.Value("other", "other")
			})
			profile := root.Values.Value(&root.Config.Profile, profileDomain)
			root.Values.Default(profile, "prod")
			gate := root.Values.String(&root.Config.Gate)
			root.Values.Default(gate, "off")

			unavailable := root.Flags.String(&root.Config.Unavailable)
			unavailable.Long("unavailable")
			unavailable.AvailableWhen(root.When.Equal(gate, "on"))
			invisible := root.Flags.String(&root.Config.Invisible)
			invisible.Long("invisible")
			invisible.VisibleWhen(root.When.Equal(gate, "on"))
			root.Flags.String(&root.Config.Removed).Release(cli.ReleaseLifecycle{
				RemovedIn: "2",
			}).Long("removed")
			root.Flags.String(&root.Config.Current).Long("current")

			root.Profiles.Switch(profile, func(cases *cli.GrammarProfileCases[config, string]) {
				cases.Case("prod", func(profile *cli.CommandBuilder[config]) {
					profile.Subcommands.Command(cli.Command[config]{
						Name: "prod-only",
					})
				})
				cases.Default(func(profile *cli.CommandBuilder[config]) {
					profile.Subcommands.Command(cli.Command[config]{
						Name: "default-only",
					})
				})
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	referenceProof := clitest.Reference(app, cli.ReferenceOptions{
		Locale: "fr",
	})
	referenceProof.Require(t, clitest.EffectiveLocale("fr"), clitest.RootCommand().Exists())
	var output strings.Builder
	runRequest := cli.RunRequest{
		Args:   []string{"--help"},
		Locale: "fr",
		Stdout: &output,
	}
	if err := app.Run(t.Context(), runRequest); err != nil {
		t.Fatalf("Run(--help) error = %v", err)
	}
	if page.Locale != "fr" || page.Description.Short != "outil court" ||
		page.LifecycleState != cli.LifecycleStateCurrent {
		t.Fatalf(
			"localized root page = locale %q description %#v lifecycle %v",
			page.Locale,
			page.Description,
			page.LifecycleState,
		)
	}
	commands := helpTestCommandNames(page)
	_, hasProdOnly := commands["prod-only"]
	_, hasDefaultOnly := commands["default-only"]
	if !hasProdOnly || hasDefaultOnly {
		t.Fatalf("profile command membership = %#v", commands)
	}
	flags := helpTestFlagNames(page)
	_, hasCurrent := flags["--current"]
	_, hasUnavailable := flags["--unavailable"]
	_, hasInvisible := flags["--invisible"]
	_, hasRemoved := flags["--removed"]
	if !hasCurrent || hasUnavailable || hasInvisible || hasRemoved {
		t.Fatalf("availability/visibility/lifecycle flags = %#v", flags)
	}
}

func TestRunHelpDemandsOnlyRequiredValuesDefaultsAndDomains(t *testing.T) {
	t.Parallel()
	type config = helpModeUnusedVisibleUnrelatedConfig
	domainCalls := 0
	unusedDomainCalls := 0
	defaultCalls := 0
	unusedDefaultCalls := 0
	var page cli.HelpPage
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Renderer: func(_ context.Context, observed cli.HelpPage) (string, error) {
				page = observed
				return "help", nil
			},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(_ context.Context, _ *config, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
					domainCalls++
					builder.Value("on", "enabled")
					return nil
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Values.ComputedDefault(
				mode,
				cli.Value.ComputedBy(func(context.Context, *config, cli.Dependencies) (string, error) {
					defaultCalls++
					return "enabled", nil
				}),
			)
			visible := root.Flags.String(&root.Config.Visible)
			visible.Long("visible")
			visible.VisibleWhen(root.When.Resolved(mode))

			unusedDomain := root.Values.Domain(
				func(_ context.Context, _ *config, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
					unusedDomainCalls++
					builder.Value("unused", "unused-value")
					return nil
				},
			)
			unused := root.Values.Value(&root.Config.Unused, unusedDomain)
			root.Values.ComputedDefault(
				unused,
				cli.Value.ComputedBy(func(context.Context, *config, cli.Dependencies) (string, error) {
					unusedDefaultCalls++
					return "unused-value", nil
				}),
			)
			root.Flags.String(&root.Config.Unrelated).Long("unrelated")
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	referenceProof := clitest.Reference(app, cli.ReferenceOptions{})
	referenceProof.Require(t, clitest.RootCommand().Exists())
	var output strings.Builder
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--help"},
		Stdout: &output,
	}); err != nil {
		t.Fatalf("Run(--help) error = %v", err)
	}
	if domainCalls != 1 || defaultCalls != 1 || unusedDomainCalls != 0 || unusedDefaultCalls != 0 {
		t.Fatalf(
			"bounded callbacks = domain %d default %d unused-domain %d unused-default %d",
			domainCalls,
			defaultCalls,
			unusedDomainCalls,
			unusedDefaultCalls,
		)
	}
	flags := helpTestFlagNames(page)
	_, hasVisible := flags["--visible"]
	_, hasUnrelated := flags["--unrelated"]
	if !hasVisible || !hasUnrelated {
		t.Fatalf("visible flags = %#v", flags)
	}
}

func TestRunHelpDoesNotDemandDependenciesOfInactiveProfileProducer(t *testing.T) {
	t.Parallel()
	type config = helpProfileSourceGateVisibleConfig
	var page cli.HelpPage
	sourceDomainCalls := 0
	inactiveDefaultCalls := 0
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Renderer: func(_ context.Context, observed cli.HelpPage) (string, error) {
				page = observed
				return "help", nil
			},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			profileDomain := cli.Value.Domain(cli.Value.Assign[string](), func(domain *cli.ValueDomainBuilder[string]) {
				domain.Value("prod", "prod")
				domain.Value("other", "other")
			})
			profile := root.Values.Value(&root.Config.Profile, profileDomain)
			root.Values.Default(profile, "prod")

			sourceDomain := root.Values.Domain(
				func(_ context.Context, _ *config, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
					sourceDomainCalls++
					builder.Value("source", "source")
					return nil
				},
			)
			source := root.Values.Value(&root.Config.Source, sourceDomain)
			gate := root.Values.String(&root.Config.Gate)
			visible := root.Flags.String(&root.Config.Visible)
			visible.Long("visible")
			visible.VisibleWhen(root.When.Equal(gate, "on"))

			root.Profiles.Switch(profile, func(cases *cli.GrammarProfileCases[config, string]) {
				cases.Case("prod", func(active *cli.CommandBuilder[config]) {
					active.Values.Default(gate, "on")
				})
				cases.Default(func(inactive *cli.CommandBuilder[config]) {
					inactive.Values.ComputedDefault(
						gate,
						cli.Value.ComputedBy(func(context.Context, *config, cli.Dependencies) (string, error) {
							inactiveDefaultCalls++
							return "on", nil
						}).From(source),
					)
				})
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	referenceProof := clitest.Reference(app, cli.ReferenceOptions{})
	referenceProof.Require(t, clitest.RootCommand().Exists())
	var output strings.Builder
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--help"},
		Stdout: &output,
	}); err != nil {
		t.Fatalf("Run(--help) error = %v", err)
	}
	if sourceDomainCalls != 0 || inactiveDefaultCalls != 0 {
		t.Fatalf(
			"inactive profile demanded source/domain: source-domain=%d inactive-default=%d",
			sourceDomainCalls,
			inactiveDefaultCalls,
		)
	}
	flags := helpTestFlagNames(page)
	if _, ok := flags["--visible"]; !ok {
		t.Fatalf("active-profile visibility dependency was not resolved: %#v", page.FlagGroups)
	}
}

func TestRunHelpRequiredMaterializationFailureSkipsRenderer(t *testing.T) {
	t.Parallel()
	type config = helpModeVisibleConfig
	failure := errors.New("domain unavailable")
	domainCalls := 0
	rendererCalls := 0
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Renderer: func(context.Context, cli.HelpPage) (string, error) {
				rendererCalls++
				return "must not render", nil
			},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(context.Context, *config, cli.Dependencies, *cli.ValueDomainBuilder[string]) error {
					domainCalls++
					return failure
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			visible := root.Flags.String(&root.Config.Visible)
			visible.Long("visible")
			visible.VisibleWhen(root.When.Resolved(mode))
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	referenceProof := clitest.Reference(app, cli.ReferenceOptions{})
	referenceProof.Require(t, clitest.RootCommand().Exists())
	var output strings.Builder
	got := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--help"},
		Stdout: &output,
	})
	if got == nil || !errors.Is(got, failure) {
		t.Fatalf("Run(--help) error = %v, want cause %v", got, failure)
	}
	if domainCalls != 1 || rendererCalls != 0 || output.Len() != 0 {
		t.Fatalf("failure boundary = domain %d renderer %d output %q", domainCalls, rendererCalls, output.String())
	}
}

type failingHelpWriter struct{ err error }

func (writer failingHelpWriter) Write([]byte) (int, error) { return 0, writer.err }

func TestRunHelpRendererAndOutputFailuresPreserveIsolation(t *testing.T) {
	t.Parallel()
	type config = helpConfig
	t.Run("renderer", func(t *testing.T) {
		t.Parallel()
		renderErr := errors.New("renderer failed")
		handlerCalls := 0
		definition := cli.Definition[config]{
			Name: "tool",
			ShellCompletion: cli.ShellCompletion{
				Disabled: true,
			},
			Help: cli.HelpConfig{
				Renderer: func(_ context.Context, page cli.HelpPage) (string, error) {
					if page.ApplicationName != "tool" || len(page.Usage) == 0 {
						t.Fatalf("renderer received incomplete page: %#v", page)
					}
					return "", renderErr
				},
			},
			Run: func(context.Context, *config) error {
				handlerCalls++
				return nil
			},
		}
		app, err := cli.Compile(definition)
		if err != nil {
			t.Fatalf("Compile() error = %v", err)
		}
		referenceProof := clitest.Reference(app, cli.ReferenceOptions{})
		referenceProof.Require(t, clitest.CommandsExactly(clitest.RootCommand()))
		got := app.Run(t.Context(), cli.RunRequest{
			Args:   []string{"--help"},
			Stdout: &strings.Builder{},
		})
		if got == nil || !errors.Is(got, renderErr) {
			t.Fatalf("Run(renderer failure) = %v", got)
		}
		report := cli.Diagnostics(got)
		if len(report.Diagnostics) != 1 || report.Diagnostics[0].Code != cli.DiagnosticHelpRenderingFailure ||
			handlerCalls != 0 {
			t.Fatalf("renderer failure diagnostics/handler = %#v / %d", report, handlerCalls)
		}
	})

	t.Run("output", func(t *testing.T) {
		t.Parallel()
		writeErr := errors.New("write failed")
		renderCalls := 0
		handlerCalls := 0
		definition := cli.Definition[config]{
			Name: "tool",
			ShellCompletion: cli.ShellCompletion{
				Disabled: true,
			},
			Help: cli.HelpConfig{
				Renderer: func(_ context.Context, page cli.HelpPage) (string, error) {
					renderCalls++
					if page.ApplicationName != "tool" || len(page.Usage) == 0 {
						t.Fatalf("renderer received incomplete page: %#v", page)
					}
					return "complete page", nil
				},
			},
			Run: func(context.Context, *config) error {
				handlerCalls++
				return nil
			},
		}
		app, err := cli.Compile(definition)
		if err != nil {
			t.Fatalf("Compile() error = %v", err)
		}
		got := app.Run(t.Context(), cli.RunRequest{
			Args: []string{"--help"},
			Stdout: failingHelpWriter{
				err: writeErr,
			},
		})
		if got == nil || !errors.Is(got, writeErr) {
			t.Fatalf("Run(output failure) = %v", got)
		}
		report := cli.Diagnostics(got)
		if len(report.Diagnostics) != 1 || report.Diagnostics[0].Code != cli.DiagnosticOutputWriteFailure ||
			renderCalls != 1 ||
			handlerCalls != 0 {
			t.Fatalf("output failure diagnostics/render/handler = %#v / %d / %d", report, renderCalls, handlerCalls)
		}
	})
}

func TestDefaultHelpPreservesAuthoritativeOrderAndIsDeterministic(t *testing.T) {
	t.Parallel()
	semanticApp := helpMustCompilePublic(t, cli.Definition[struct{}]{
		Name: "tool",
		Description: cli.Description{
			Short: "tool description",
		},
	})
	semanticReference := clitest.Reference(semanticApp, cli.ReferenceOptions{})
	semanticReference.Require(t,
		clitest.CommandsExactly(clitest.RootCommand(), clitest.Command("completion")),
	)
	zetaLong := cli.FlagSpellingID(1)
	zetaShort := cli.FlagSpellingID(2)
	alphaLong := cli.FlagSpellingID(3)
	page := cli.HelpPage{
		Description: cli.Description{
			Short: "ordered help",
		},
		Usage: []string{"tool z-first a-second"},
		Labels: cli.HelpLabels{
			Usage:     "Usage",
			Commands:  "Commands",
			Arguments: "Arguments",
			Flags:     "Flags",
		},
		CommandGroups: []cli.HelpCommandGroup{{
			Title: "Commands",
			Commands: []cli.HelpCommand{{
				Name: "zeta",
			}, {
				Name: "alpha",
			}},
		}},
		ArgumentForms: []cli.HelpArgumentForm{{Terms: []cli.HelpArgumentTerm{{
			Name: "ZETA",
		}, {
			Name: "ALPHA",
		}}}},
		FlagGroups: []cli.HelpFlagGroup{{
			Title: "Flags",
			Flags: []cli.HelpFlag{
				{
					FlagSpellings: []cli.FlagSpellingID{zetaLong, zetaShort},
				},
				{
					FlagSpellings: []cli.FlagSpellingID{alphaLong},
				},
			},
		}},
		FlagSpellings: map[cli.FlagSpellingID]cli.HelpFlagSpelling{
			zetaLong: {
				ID: zetaLong,
				Spelling: cli.FlagSpelling{
					Kind: cli.FlagSpellingLong,
					Text: "--zeta",
				},
			},
			zetaShort: {
				ID: zetaShort,
				Spelling: cli.FlagSpelling{
					Kind: cli.FlagSpellingShort,
					Text: "-z",
				},
			},
			alphaLong: {
				ID: alphaLong,
				Spelling: cli.FlagSpelling{
					Kind: cli.FlagSpellingLong,
					Text: "--alpha",
				},
			},
		},
		Examples: []cli.Example{
			{
				Description: "second lexical",
				CommandLine: "tool zeta",
			},
			{
				Description: "first lexical",
				CommandLine: "tool alpha",
			},
		},
	}
	first, err := cli.DefaultHelp(cli.DefaultHelpConfig{})(t.Context(), page)
	if err != nil {
		t.Fatalf("DefaultHelp() error = %v", err)
	}
	second, err := cli.DefaultHelp(cli.DefaultHelpConfig{})(t.Context(), page)
	if err != nil {
		t.Fatalf("second DefaultHelp() error = %v", err)
	}
	if first != second {
		t.Fatalf("DefaultHelp() nondeterministic:\nfirst=%q\nsecond=%q", first, second)
	}
	for _, pair := range [][2]string{{"zeta", "alpha"}, {"ZETA", "ALPHA"}, {"--zeta", "--alpha"}, {"-z", "--alpha"}, {"tool zeta", "tool alpha"}} {
		left, right := strings.Index(first, pair[0]), strings.Index(first, pair[1])
		if left < 0 || right < 0 || left >= right {
			t.Fatalf("authoritative order %q before %q not preserved in:\n%s", pair[0], pair[1], first)
		}
	}
}

func helpTestCommandNames(page cli.HelpPage) map[string]struct{} {
	result := make(map[string]struct{})
	for _, group := range page.CommandGroups {
		for _, command := range group.Commands {
			result[command.Name] = struct{}{}
		}
	}
	return result
}

func helpTestFlagNames(page cli.HelpPage) map[string]struct{} {
	result := make(map[string]struct{})
	for _, group := range page.FlagGroups {
		for _, flag := range group.Flags {
			if flag.ActionEntry != 0 {
				result[flag.ActionSpelling.Text] = struct{}{}
				continue
			}
			for _, id := range flag.FlagSpellings {
				result[page.FlagSpellings[id].Spelling.Text] = struct{}{}
			}
		}
	}
	return result
}

func TestHelpGroupAuthoringMergesCompatibleReopenAndRejectsConflictsAndNesting(t *testing.T) {
	t.Parallel()
	type config = helpZetaAlphaConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.HelpGroup("Options", func(flags *cli.Flags[config]) {
				flags.Bool(&root.Config.Zeta).Long("zeta")
			}).OrderByName()
			root.Flags.HelpGroup("Options", func(flags *cli.Flags[config]) {
				flags.Bool(&root.Config.Alpha).Long("alpha")
			}).OrderByName()
		},
	})
	group := clitest.RootCommand().HelpFlagGroup("Options")
	clitest.Schema(app).Require(t,
		group.Kind(cli.HelpGroupDeclared),
		group.Title("Options"),
		group.Order(cli.HelpGroupOrderName),
		group.FlagsExactly(clitest.RootCommand().Flag("--zeta"), clitest.RootCommand().Flag("--alpha")),
	)

	for _, test := range []struct {
		name    string
		declare func(*cli.CommandBuilder[config])
	}{
		{
			name: "conflicting reopen options",
			declare: func(root *cli.CommandBuilder[config]) {
				root.Flags.HelpGroup("Options", func(flags *cli.Flags[config]) { flags.Bool(&root.Config.Zeta).Long("zeta") })
				root.Flags.HelpGroup("Options", func(flags *cli.Flags[config]) { flags.Bool(&root.Config.Alpha).Long("alpha") }).OrderByName()
			},
		},
		{
			name: "nested group",
			declare: func(root *cli.CommandBuilder[config]) {
				root.Flags.HelpGroup("Outer", func(flags *cli.Flags[config]) {
					flags.HelpGroup("Inner", func(inner *cli.Flags[config]) { inner.Bool(&root.Config.Zeta).Long("zeta") })
				})
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			proof := clitest.CompileFails(t, cli.Definition[config]{
				Name:    "tool",
				Declare: test.declare,
			})
			proof.Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
		})
	}
}

func TestHelpPresentationOrderingHiddenEntriesAndActionSyntax(t *testing.T) {
	t.Parallel()
	type config = helpZetaAlphaPlainConfig
	var page cli.HelpPage
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Renderer: func(_ context.Context, got cli.HelpPage) (string, error) {
				page = got
				return "help", nil
			},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.HelpGroup("Options", func(flags *cli.Flags[config]) {
				flags.Bool(&root.Config.Zeta).Long("zeta")
				flags.Bool(&root.Config.Alpha).Long("alpha")
			}).OrderByName()
			root.Flags.Bool(&root.Config.Plain).Long("plain")
			root.Subcommands.Command(cli.Command[config]{
				Name: "child",
				Aliases: cli.CommandAliases{
					{
						Name: "visible",
					},
					{
						Name:   "secret",
						Hidden: true,
					},
				},
				Description: cli.Description{
					Short: "child description",
				},
				Declare: func(child *cli.CommandBuilder[config]) {
					child.Shortcut(cli.CommandShortcut{
						Name:   "silent",
						Args:   []string{"--help"},
						Hidden: true,
					})
				},
			})
		},
	}
	app := clitest.Compile(t, definition)
	declared := clitest.RootCommand().HelpFlagGroup("Options")
	clitest.Schema(app).Require(t,
		declared.Kind(cli.HelpGroupDeclared),
		declared.Order(cli.HelpGroupOrderName),
		declared.FlagsExactly(clitest.RootCommand().Flag("--zeta"), clitest.RootCommand().Flag("--alpha")),
	)

	var output strings.Builder
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--help"},
		Stdout: &output,
	}); err != nil {
		t.Fatalf("Run(--help) error = %v", err)
	}
	options := clitest.HelpFlagGroup("Options")
	defaults := clitest.HelpFlagGroupAt(1)
	clitest.RequireHelpPresentation(t, page,
		options.Kind(cli.HelpGroupDeclared),
		options.Order(cli.HelpGroupOrderName),
		options.FlagCount(2),
		options.FlagAt(0).SpellingAt(0).Spelling(cli.FlagSpelling{
			Kind:      cli.FlagSpellingLong,
			Text:      "--alpha",
			ValueMode: cli.FlagValueModeNone,
			FixedValue: &cli.ValueLiteral{
				Type: cli.ValueType{
					Kind: cli.ValueKindBool,
				},
				Text: "true",
			},
		}),
		options.FlagAt(1).SpellingAt(0).Spelling(cli.FlagSpelling{
			Kind:      cli.FlagSpellingLong,
			Text:      "--zeta",
			ValueMode: cli.FlagValueModeNone,
			FixedValue: &cli.ValueLiteral{
				Type: cli.ValueType{
					Kind: cli.ValueKindBool,
				},
				Text: "true",
			},
		}),
		defaults.Kind(cli.HelpGroupDefault),
		defaults.Flag("--plain").Exists(),
		defaults.ActionFlag("--help").Action(),
		defaults.ActionFlag("--help").ActionSpelling(cli.FlagSpelling{
			Kind:      cli.FlagSpellingLong,
			Text:      "--help",
			ValueMode: cli.FlagValueModeNone,
		}),
		defaults.ActionFlag("--help").SpellingCount(0),
		clitest.HelpCommand("child").Kind(cli.HelpCommandCanonical),
		clitest.HelpCommand("visible").Kind(cli.HelpCommandAlias),
		clitest.HelpCommand("secret").DoesNotExist(),
		clitest.HelpCommand("silent").DoesNotExist(),
	)
}

func TestHelpNormalizationClosesHiddenSpellingReplacementWithoutRenderingClosureOnlyNode(t *testing.T) {
	t.Parallel()
	type config = helpEnabledConfig
	var pages []cli.HelpPage
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.SemVer("2.0.0"),
		Help: cli.HelpConfig{
			Renderer: func(_ context.Context, page cli.HelpPage) (string, error) {
				pages = append(pages, page)
				return "help", nil
			},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			flag := root.Flags.Bool(&root.Config.Enabled)
			old := flag.Long("old").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2.0.0",
			})
			replacement := flag.Long("new").Hidden()
			old.ReplacedBy(replacement)
			flag.Long("private").Hidden()
		},
	}
	app := clitest.Compile(t, definition)
	old := clitest.RootCommand().Flag("--old").Spelling("--old")
	newSpelling := clitest.RootCommand().Flag("--old").Spelling("--new")
	private := clitest.RootCommand().Flag("--old").Spelling("--private")
	clitest.Schema(app).Require(t,
		old.ReplacedBy(newSpelling),
		newSpelling.Hidden(),
		private.Hidden(),
	)
	var output strings.Builder
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--help"},
		Stdout: &output,
	}); err != nil {
		t.Fatalf("Run(--help): %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("renderer pages = %d", len(pages))
	}
	expect := []clitest.HelpPresentationExpectation{
		clitest.HelpFlagGroupAt(0).Flag("--old").Ordinary(),
		clitest.HelpFlagGroupAt(0).Flag("--old").SpellingCount(1),
		clitest.HelpFlagSpelling("--old").Exists(),
		clitest.HelpFlagSpelling("--old").Replacement().Exists(),
		clitest.HelpFlagSpelling("--new").Exists(),
		clitest.HelpFlagGroupAt(0).Flag("--new").DoesNotExist(),
		clitest.HelpFlagSpelling("--private").DoesNotExist(),
		clitest.HelpFlagGroupAt(0).Flag("--private").DoesNotExist(),
	}
	clitest.RequireHelpPresentation(t, pages[0], expect...)

	mutated := pages[0]
	for id, node := range mutated.FlagSpellings {
		node.Spelling.Text = "--mutated"
		mutated.FlagSpellings[id] = node
	}
	for gi := range mutated.FlagGroups {
		for fi := range mutated.FlagGroups[gi].Flags {
			if len(mutated.FlagGroups[gi].Flags[fi].FlagSpellings) != 0 {
				mutated.FlagGroups[gi].Flags[fi].FlagSpellings[0] = cli.FlagSpellingID(999999)
			}
		}
	}
	output.Reset()
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--help"},
		Stdout: &output,
	}); err != nil {
		t.Fatalf("second Run(--help): %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("renderer pages after mutation = %d", len(pages))
	}
	clitest.RequireHelpPresentation(t, pages[1], expect...)
}

func TestHelpActionRowsPreserveExactOwnerPayloadAndDoNotConflateCrossFormReplacement(t *testing.T) {
	t.Parallel()
	var page cli.HelpPage
	definition := cli.Definition[struct{}]{
		Name:    "tool",
		Release: cli.Release.SemVer("2.0.0"),
		Help: cli.HelpConfig{
			Renderer: func(_ context.Context, got cli.HelpPage) (string, error) {
				page = got
				return "help", nil
			},
			ReplaceEntries: func(entries *cli.HelpActionEntries) {
				oldFlag := entries.Long("assist").
					Description(cli.Description{
						Short: "assist",
					}).
					Release(cli.ReleaseLifecycle{
						DeprecatedIn: "2.0.0",
					})
				hiddenCommand := entries.Command("guide").Hidden()
				oldFlag.ReplacedBy(hiddenCommand)
				oldCommand := entries.Command("manual").
					Description(cli.Description{
						Short: "manual",
					}).
					Release(cli.ReleaseLifecycle{
						DeprecatedIn: "2.0.0",
					})
				hiddenFlag := entries.Long("manual-new").Hidden()
				oldCommand.ReplacedBy(hiddenFlag)
			},
		},
	}
	app := clitest.Compile(t, definition)
	schema := clitest.Schema(app)
	assist := clitest.ResolveDeclaration(t, schema, clitest.RootCommand().ActionEntryAt(0))
	guide := clitest.ResolveDeclaration(t, schema, clitest.RootCommand().ActionEntryAt(1))
	manual := clitest.ResolveDeclaration(t, schema, clitest.RootCommand().ActionEntryAt(2))
	manualNew := clitest.ResolveDeclaration(t, schema, clitest.RootCommand().ActionEntryAt(3))
	schema.Require(t,
		assist.ReplacedBy(guide),
		manual.ReplacedBy(manualNew),
		guide.Hidden(),
		manualNew.Hidden(),
	)
	var output strings.Builder
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--assist"},
		Stdout: &output,
	}); err != nil {
		t.Fatalf("Run(--assist): %v", err)
	}
	clitest.RequireHelpPresentation(t, page,
		clitest.HelpFlagGroupAt(0).ActionFlag("--assist").Action(),
		clitest.HelpFlagGroupAt(0).ActionFlag("--assist").ActionEntry(assist),
		clitest.HelpFlagGroupAt(0).ActionFlag("--assist").SpellingCount(0),
		clitest.HelpFlagGroupAt(0).ActionFlag("--assist").ActionSpelling(cli.FlagSpelling{
			Kind:      cli.FlagSpellingLong,
			Text:      "--assist",
			ValueMode: cli.FlagValueModeNone,
		}),
		clitest.HelpFlagGroupAt(0).ActionFlag("--assist").Lifecycle(cli.LifecycleStateDeprecated),
		clitest.HelpFlagGroupAt(0).ActionFlag("--manual-new").DoesNotExist(),
		clitest.HelpCommandGroupAt(0).Command("manual").ActionEntry(manual),
		clitest.HelpCommandGroupAt(0).Command("manual").Lifecycle(cli.LifecycleStateDeprecated),
		clitest.HelpCommandGroupAt(0).Command("guide").DoesNotExist(),
		clitest.HelpFlagSpelling("--assist").DoesNotExist(),
		clitest.HelpFlagSpelling("--manual-new").DoesNotExist(),
	)
}

func TestHelpValueDomainNormalizationProjectsDescriptionHiddenCanonicalAndReplacementClosure(t *testing.T) {
	t.Parallel()
	type config = helpModeConfig
	var pages []cli.HelpPage
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Renderer: func(_ context.Context, got cli.HelpPage) (string, error) {
				pages = append(pages, got)
				return "help", nil
			},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := cli.Value.Domain(cli.Value.Assign[string](), func(builder *cli.ValueDomainBuilder[string]) {
				canonical := builder.Value(
					"prod",
					"production",
				).Description("Production environment").Hidden()

				builder.Alias("p", canonical)
				reference := builder.Value("reference", "reference").Description("Reference environment")
				modern := builder.Value(
					"modern",
					"modern",
				).Description("Modern environment").Hidden()

				reference.ReplacedBy(modern)
				builder.Alias("secret", canonical).Hidden()
			})
			root.Flags.Value(&root.Config.Mode, domain).Long("mode")
		},
	}
	app := clitest.Compile(t, definition)
	domain := clitest.RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination)
	clitest.Schema(app).Require(t,
		domain.Entry("prod").Hidden(),
		domain.Entry("p").Canonical(domain.Entry("prod")),
		domain.Entry("reference").ReplacedBy(domain.Entry("modern")),
		domain.Entry("modern").Hidden(),
		domain.Entry("secret").Hidden(),
	)
	var output strings.Builder
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--help"},
		Stdout: &output,
	}); err != nil {
		t.Fatalf("Run(--help): %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("renderer pages = %d", len(pages))
	}
	expect := []clitest.HelpPresentationExpectation{
		clitest.HelpFlagGroupAt(0).Flag("--mode").ValueDomainEntryCount(cli.ValueDomainUseDestination, 2),
		clitest.HelpFlagGroupAt(0).Flag("--mode").ValueDomainEntryAt(cli.ValueDomainUseDestination, 0).Name("p"),
		clitest.HelpFlagGroupAt(0).
			Flag("--mode").
			ValueDomainEntryAt(cli.ValueDomainUseDestination, 1).
			Name("reference"),
		clitest.HelpValueDomainEntry("p").Description("Production environment"),
		clitest.HelpValueDomainEntry("p").Canonical().Name("prod"),
		clitest.HelpValueDomainEntry("prod").Exists(),
		clitest.HelpValueDomainEntry("reference").Description("Reference environment"),
		clitest.HelpValueDomainEntry("reference").Replacement().Name("modern"),
		clitest.HelpValueDomainEntry("modern").Exists(),
		clitest.HelpValueDomainEntry("secret").DoesNotExist(),
	}
	clitest.RequireHelpPresentation(t, pages[0], expect...)

	mutated := pages[0]
	for id, node := range mutated.ValueDomainEntries {
		node.Description = "mutated"
		mutated.ValueDomainEntries[id] = node
	}
	for gi := range mutated.FlagGroups {
		for fi := range mutated.FlagGroups[gi].Flags {
			for ui := range mutated.FlagGroups[gi].Flags[fi].ValueDomainUses {
				if len(mutated.FlagGroups[gi].Flags[fi].ValueDomainUses[ui].Entries) != 0 {
					mutated.FlagGroups[gi].Flags[fi].ValueDomainUses[ui].Entries[0] = cli.ValueDomainEntryID(999999)
				}
			}
		}
	}
	output.Reset()
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--help"},
		Stdout: &output,
	}); err != nil {
		t.Fatalf("second Run(--help): %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("renderer pages after mutation = %d", len(pages))
	}
	clitest.RequireHelpPresentation(t, pages[1], expect...)
}

func TestHelpActionFlagSubtreeInheritancePreservesExactActionOwnerAndInlineSyntax(t *testing.T) {
	t.Parallel()
	var page cli.HelpPage
	definition := cli.Definition[struct{}]{
		Name: "tool",
		Help: cli.HelpConfig{
			Renderer: func(_ context.Context, got cli.HelpPage) (string, error) {
				page = got
				return "help", nil
			},
			ReplaceEntries: func(entries *cli.HelpActionEntries) {
				entries.Long("assist").
					Placement(cli.FlagPlacementSubtree).
					Description(cli.Description{
						Short: "global help",
					})
			},
		},
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "child",
			})
		},
	}
	app := clitest.Compile(t, definition)
	schema := clitest.Schema(app)
	assist := clitest.ResolveDeclaration(t, schema, clitest.RootCommand().ActionEntryAt(0))
	schema.Require(t, assist.Exists())
	var output strings.Builder
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"child", "--assist"},
		Stdout: &output,
	}); err != nil {
		t.Fatalf("Run(child --assist): %v", err)
	}
	row := clitest.HelpFlagGroupAt(0).ActionFlag("--assist")
	clitest.RequireHelpPresentation(t, page,
		clitest.HelpFlagGroupAt(0).Kind(cli.HelpGroupInherited),
		row.Action(),
		row.ActionEntry(assist),
		row.SpellingCount(0),
		row.ActionSpelling(cli.FlagSpelling{
			Kind:      cli.FlagSpellingLong,
			Text:      "--assist",
			ValueMode: cli.FlagValueModeNone,
		}),
		row.Placement(cli.FlagPlacementSubtree),
		clitest.HelpFlagSpelling("--assist").DoesNotExist(),
	)
}

func TestInspectionProjectsAutomaticHelpRoutingAfterDefaultSelection(t *testing.T) {
	t.Parallel()
	type config = helpConfig
	definition := cli.Definition[config]{
		Name: "commandment",
		Help: cli.HelpConfig{
			WhenEmpty: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.Command(cli.Command[config]{
				Name: "branch",
				Declare: func(branch *cli.CommandBuilder[config]) {
					branch.Subcommands.Command(cli.Command[config]{
						Name: "leaf",
						Run:  func(context.Context, *config) error { return nil },
					}).Default()
				},
			}).Default()
		},
	}
	whenEmptyApp, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile(WhenEmpty) error = %v", err)
	}
	result := clitest.Inspect(t.Context(), whenEmptyApp, cli.InspectionRequest{})
	result.Require(
		t,
		clitest.InspectionReady(),
		clitest.InvocationAction(cli.InvocationActionHelp).Selected(),
		clitest.InvocationAction(cli.InvocationActionHelp).Trigger(cli.InvocationActionTriggerEmpty),
		clitest.SelectionPathExactly(
			clitest.RootCommand(),
			clitest.Command("branch"),
			clitest.Command("branch", "leaf"),
		),
	)
	if t.Failed() {
		return
	}

	structuralApp, err := cli.Compile(cli.Definition[config]{
		Name: "commandment",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.Command(cli.Command[config]{
				Name: "branch",
				Declare: func(branch *cli.CommandBuilder[config]) {
					branch.Subcommands.Command(cli.Command[config]{
						Name: "leaf",
						Declare: func(leaf *cli.CommandBuilder[config]) {
							leaf.Subcommands.Command(cli.Command[config]{
								Name: "nested",
								Run:  func(context.Context, *config) error { return nil },
							})
						},
					}).Default()
				},
			}).Default()
		},
	})
	if err != nil {
		t.Fatalf("Compile(structural) error = %v", err)
	}
	clitest.Inspect(t.Context(), structuralApp, cli.InspectionRequest{
		Args: []string{"branch"},
	}).Require(t,
		clitest.InspectionReady(),
		clitest.InvocationAction(cli.InvocationActionHelp).Selected(),
		clitest.InvocationAction(cli.InvocationActionHelp).Trigger(cli.InvocationActionTriggerStructural),
		clitest.SelectionPathExactly(clitest.RootCommand(), clitest.Command("branch"), clitest.Command("branch", "leaf")),
		clitest.Command("branch", "leaf").NotRunnable(),
	)
	if t.Failed() {
		return
	}

	disabledApp, err := cli.Compile(cli.Definition[config]{
		Name: "commandment",
		Help: cli.HelpConfig{
			Disabled: true,
		},
	})
	if err != nil {
		t.Fatalf("Compile(disabled Help) error = %v", err)
	}
	clitest.Inspect(t.Context(), disabledApp, cli.InspectionRequest{}).Require(t,
		clitest.InspectionReady(),
		clitest.InvocationAction(cli.InvocationActionRun).Selected(),
		clitest.InvocationAction(cli.InvocationActionRun).Trigger(cli.InvocationActionTriggerUnknown),
		clitest.SelectionPathExactly(clitest.RootCommand()),
	)
}

func TestFrameworkHelpPhrasesForLocaleUseSelectedLocale(t *testing.T) {
	t.Parallel()
	var page cli.HelpPage
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Help: cli.HelpConfig{
			Renderer: func(_ context.Context, observed cli.HelpPage) (string, error) {
				page = observed
				return "help", nil
			},
		},
	})
	var output strings.Builder
	runRequest := cli.RunRequest{
		Args:   []string{"--help"},
		Locale: "en-US",
		Stdout: &output,
	}
	if err := app.Run(t.Context(), runRequest); err != nil {
		t.Fatal(err)
	}
	clitest.RequireHelpPresentation(t, page,
		clitest.EffectiveLocale("en-US"),
		clitest.HelpLabels(cli.HelpLabels{
			Usage:         "Usage",
			Commands:      "Commands",
			Arguments:     "Arguments",
			Flags:         "Flags",
			GlobalFlags:   "Global flags",
			Aliases:       "Aliases",
			Environment:   "Environment",
			Default:       "default",
			Required:      "required",
			Optional:      "optional",
			Deprecated:    "deprecated",
			Examples:      "Examples",
			Alias:         "alias",
			Shortcut:      "shortcut",
			Values:        "Values",
			Keys:          "Keys",
			Syntax:        "Syntax",
			AliasFor:      "alias for",
			Use:           "use",
			ImplicitValue: "Implicit value",
			FixedValue:    "Fixed value",
		}),
	)
}

func TestHelpProjectionBuildsUsageAlternatives(t *testing.T) {
	t.Parallel()
	type config = helpNameItemsConfig
	var page cli.HelpPage
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Renderer: func(_ context.Context, got cli.HelpPage) (string, error) {
				page = got
				return "help\n", nil
			},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Name).Long("name")
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Literal("open")
				form.Strings(&form.Config.Items, "ITEM", cli.Count.ZeroOrMore())
			})
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "child",
			})
		},
		Run: func(context.Context, *config) error { return nil },
	}
	app := clitest.Compile(t, definition)
	var output strings.Builder
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--help"},
		Stdout: &output,
	}); err != nil {
		t.Fatal(err)
	}
	if output.String() != "help\n" {
		t.Fatalf("help output = %q", output.String())
	}
	clitest.RequireHelpPresentation(t, page,
		clitest.HelpUsageExactly("tool [flags] open [ITEM...]", "tool [flags] [command]"),
	)
}

func TestHelpUsageRendersLiteralOptionalAndRepeatedTerms(t *testing.T) {
	t.Parallel()
	type config = helpValueItemsConfig
	var page cli.HelpPage
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Renderer: func(_ context.Context, got cli.HelpPage) (string, error) {
				page = got
				return "help\n", nil
			},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Literal("literal")
				form.String(&form.Config.Value, "").Optional()
				form.Strings(&form.Config.Items, "ITEM", cli.Count.OneOrMore())
			})
		},
	}
	app := clitest.Compile(t, definition)
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--help"},
		Stdout: &strings.Builder{},
	}); err != nil {
		t.Fatal(err)
	}
	clitest.RequireHelpPresentation(t, page,
		clitest.HelpUsageExactly("tool [flags] literal [VALUE] <ITEM...>"),
	)
}

func TestDefaultHelpRendersStructuredPresentation(t *testing.T) {
	t.Parallel()
	type config = helpNameModeConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Description: cli.Description{
			Long: "Long description",
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Name).ValueName("NAME").Long("name")
			root.Flags.String(&root.Config.Mode).ValueName("MODE").Long("mode").ImplicitValue("auto")
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "child",
				Description: cli.Description{
					Long: "Long child description",
				},
			})
		},
	}
	app := clitest.Compile(t, definition)
	var output strings.Builder
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--help"},
		Stdout: &output,
	}); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, want := range []string{"Long description", "Commands", "child", "Long child description", "Flags", "--name NAME", "--mode[=MODE]", "--help"} {
		if !strings.Contains(text, want) {
			t.Fatalf("default help missing %q:\n%s", want, text)
		}
	}
}

func TestDefaultHelpUsesFallbackLabels(t *testing.T) {
	t.Parallel()
	page := cli.HelpPage{
		ApplicationName: "tool",
		Usage:           []string{"tool"},
	}
	text, err := cli.DefaultHelp(cli.DefaultHelpConfig{})(t.Context(), page)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Usage") {
		t.Fatalf("default fallback label missing: %q", text)
	}
}

func TestHelpProjectionPreservesExamplesSensitivityAndCommandKinds(t *testing.T) {
	t.Parallel()
	type config = helpSecretConfig
	var page cli.HelpPage
	definition := cli.Definition[config]{
		Name: "tool",
		Examples: []cli.Example{{
			Description: "application example",
			CommandLine: "tool child",
		}},
		Help: cli.HelpConfig{
			Renderer: func(_ context.Context, got cli.HelpPage) (string, error) {
				page = got
				return "help\n", nil
			},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.String(&form.Config.Secret, "SECRET").Sensitive()
			})
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "child",
				Examples: []cli.Example{{
					Description: "child example",
					CommandLine: "tool child",
				}},
				Declare: func(child *cli.CommandBuilder[struct{}]) {
					child.Shortcut(cli.CommandShortcut{
						Name: "quick",
						Description: cli.Description{
							Short: "quick child",
						},
						Args: []string{"grandchild"},
					})
					child.Subcommands.Command(cli.Command[struct{}]{
						Name: "grandchild",
					})
				},
			})
		},
	}
	app := clitest.Compile(t, definition)
	var output strings.Builder
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--help"},
		Stdout: &output,
	}); err != nil {
		t.Fatal(err)
	}
	clitest.RequireHelpPresentation(t, page,
		clitest.HelpExamplesExactly(cli.Example{
			Description: "application example",
			CommandLine: "tool child",
		}),
		clitest.HelpArgumentFormAt(0).TermCount(1),
		clitest.HelpArgumentFormAt(0).TermAt(0).NotSensitive(),
		clitest.HelpCommand("child").HasSubcommands(),
		clitest.HelpCommand("quick").Kind(cli.HelpCommandShortcut),
	)
	if !page.ArgumentForms[0].Terms[0].SourceSensitive {
		t.Fatal("help argument term did not preserve source sensitivity")
	}

	page = cli.HelpPage{}
	output.Reset()
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"help", "child"},
		Stdout: &output,
	}); err != nil {
		t.Fatal(err)
	}
	clitest.RequireHelpPresentation(t, page,
		clitest.HelpExamplesExactly(cli.Example{
			Description: "child example",
			CommandLine: "tool child",
		}),
	)
}

func TestHelpProjectionPreservesArgumentTermKinds(t *testing.T) {
	t.Parallel()
	type config = helpItemsPairConfig
	language := cli.TokenLanguage[string]{
		Name:   "pair",
		Count:  cli.Count.Exactly(2),
		Syntax: []string{"LEFT", "RIGHT"},
		Parse:  func(context.Context, []string) (string, error) { return "pair", nil },
		Format: func(string) []string { return []string{"left", "right"} },
	}
	var page cli.HelpPage
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Renderer: func(_ context.Context, got cli.HelpPage) (string, error) {
				page = got
				return "help\n", nil
			},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			pair := root.Values.String(&root.Config.Pair)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Strings(&form.Config.Items, "ITEM", cli.Count.OneOrMore())
				form.Tokens(pair, "PAIR", language)
			})
		},
	}
	app := clitest.Compile(t, definition)
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--help"},
		Stdout: &strings.Builder{},
	}); err != nil {
		t.Fatal(err)
	}
	form := clitest.HelpArgumentFormAt(0)
	clitest.RequireHelpPresentation(t, page,
		form.TermCount(2),
		form.TermAt(0).Kind(cli.ArgumentTermSequence),
		form.TermAt(1).Kind(cli.ArgumentTermTokens),
		form.TermAt(1).TokenLanguageFragmentCount(2),
	)
}

func TestFrameworkTranslationsRemainSeparateFromApplicationLocalization(t *testing.T) {
	t.Parallel()
	definition := cli.Definition[struct{}]{
		Name: "alienctl",
		Description: cli.Description{
			Short: "Usage",
		},
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: map[string]string{
					"Usage": "Application Usage",
				},
			}},
		},
		FrameworkTranslations: []cli.TranslationBundle{{
			Locale: "fr",
			Translations: map[string]string{
				"Usage": "Utilisation",
			},
		}},
	}
	app := clitest.Compile(t, definition)
	var output strings.Builder
	runRequest := cli.RunRequest{
		Args:   []string{"--help"},
		Stdout: &output,
		Locale: "fr",
	}
	if err := app.Run(t.Context(), runRequest); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if !strings.Contains(text, "Application Usage") || !strings.Contains(text, "Utilisation") {
		t.Fatalf("application/framework localization collision was not isolated:\n%s", text)
	}
	if got := app.Schema().Localization.DefaultLocale; got != "en" {
		t.Fatalf("schema application DefaultLocale = %q, want en", got)
	}
}

type helpCountingSignalSource struct {
	open int
}

func (source *helpCountingSignalSource) Source() cli.SignalSource[string] {
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
