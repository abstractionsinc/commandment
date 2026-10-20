package conformance

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

type authoringLanguageModeItemsLabelsUnsignedPairConfig struct {
	Mode     string
	Items    []string
	Labels   map[string]string
	Unsigned uint
	Pair     string
}

type authoringLanguageModeItemsLabelsUnsignedPairCustomConfig struct {
	Mode     string
	Items    []string
	Labels   map[string]string
	Unsigned uint
	Pair     string
	Custom   string
}

type authoringLanguageLabelsConfig struct{ Labels map[string]string }

func TestLanguageAuthoringProjectsDeclaredSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringLanguageModeItemsLabelsUnsignedPairConfig
	domain := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
		old := values.Value("old", "old").Description("old").Hidden()
		current := values.Value("new", "new")
		old.ReplacedBy(current)
		values.Alias("alias", current).Hidden()
	})
	keyed := cli.Value.Keyed(cli.Value.String(), cli.Value.String()).SeparatedBy("=").KeyOnly("true")
	language := cli.TokenLanguage[string]{
		Name:   "pair",
		Count:  cli.Count.Exactly(2),
		Parse:  func(context.Context, []string) (string, error) { return "pair", nil },
		Format: func(string) []string { return []string{"a", "b"} },
	}
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Values.Sequence(&root.Config.Items, domain)
			root.Values.Keyed(&root.Config.Labels, keyed)
			root.Values.Value(&root.Config.Unsigned, cli.Value.Uint())
			root.Values.ResolveWith(mode, cli.Value.RejectMultipleTiers())
			pair := root.Values.String(&root.Config.Pair)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) { form.Tokens(pair, "pair", language) })
		},
	})
	root := clitest.RootCommand()
	modeDomain := root.ValueAt(0).Domain(cli.ValueDomainUseDestination)
	form := root.ArgumentFormAt(0)
	clitest.Schema(app).Require(t,
		modeDomain.EntriesExactly(modeDomain.Entry("old"), modeDomain.Entry("new"), modeDomain.Entry("alias")),
		modeDomain.Entry("old").Hidden(),
		modeDomain.Entry("old").ReplacedBy(modeDomain.Entry("new")),
		modeDomain.Entry("alias").Hidden(),
		modeDomain.Entry("alias").Canonical(modeDomain.Entry("new")),
		root.ArgumentFormsExactly(form),
		form.TermsExactly(form.Term("pair")),
		form.Term("pair").HasTokenLanguage(),
	)
	for _, count := range []cli.CountRange{cli.Count.Exactly(1), cli.Count.Between(1, 2), cli.Count.AtLeast(1), cli.Count.AtMost(2), cli.Count.Optional(), cli.Count.OneOrMore(), cli.Count.ZeroOrMore()} {
		_, _ = count.Maximum()
		_ = count.Minimum()
		_ = count.Unbounded()
	}
}

func TestLanguageAuthoringProjectsExactLanguageSemantics(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		count   cli.CountRange
		min     int
		max     int
		bounded bool
	}{
		{"exact", cli.Count.Exactly(2), 2, 2, true}, {"between", cli.Count.Between(1, 3), 1, 3, true}, {"at-least", cli.Count.AtLeast(2), 2, 0, false}, {"at-most", cli.Count.AtMost(3), 0, 3, true}, {"optional", cli.Count.Optional(), 0, 1, true}, {"one-or-more", cli.Count.OneOrMore(), 1, 0, false}, {"zero-or-more", cli.Count.ZeroOrMore(), 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.count.Minimum() != tc.min {
				t.Fatalf("minimum=%d want %d", tc.count.Minimum(), tc.min)
			}
			maximum, ok := tc.count.Maximum()
			if ok != tc.bounded || (ok && maximum != tc.max) {
				t.Fatalf("maximum=(%d,%v) want (%d,%v)", maximum, ok, tc.max, tc.bounded)
			}
			if tc.count.Unbounded() == tc.bounded {
				t.Fatalf("unbounded=%v want %v", tc.count.Unbounded(), !tc.bounded)
			}
		})
	}
	type config = authoringLanguageModeItemsLabelsUnsignedPairCustomConfig
	domain := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
		current := values.Value("new", "new")
		values.Value("old", "old").ReplacedBy(current)
		values.Alias("alias", current)
	})
	keyed := cli.Value.Keyed(cli.Value.String(), cli.Value.String()).SeparatedBy(":").KeyOnly("true")
	codec := cli.ValueCodec[string]{
		Name:     "prefixed",
		Parse:    func(text string) (string, error) { return "codec:" + text, nil },
		Format:   func(value string) string { return value },
		Snapshot: cli.Value.Assign[string](),
	}
	tokens := cli.TokenLanguage[string]{
		Name:   "pair",
		Count:  cli.Count.Exactly(2),
		Syntax: []string{"LEFT", "RIGHT"},
		Parse:  func(context.Context, []string) (string, error) { return "pair", nil },
		Format: func(string) []string { return []string{"a", "b"} },
	}
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Values.Sequence(&root.Config.Items, domain)
			labels := root.Values.Keyed(&root.Config.Labels, keyed)
			root.Values.Value(&root.Config.Unsigned, cli.Value.Uint())
			custom := root.Values.Value(&root.Config.Custom, codec)
			root.Values.ResolveWith(mode, cli.Value.RejectMultipleTiers())
			root.Values.Environment(mode, "MODE")
			root.Flags.ValueFor(mode).Long("mode")
			root.Flags.Keyed(labels).Long("label")
			root.Flags.ValueFor(custom).Long("custom")
			pair := root.Values.String(&root.Config.Pair)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) { form.Tokens(pair, "pair", tokens) })
		},
	})
	root := clitest.RootCommand()
	mode := root.Flag("--mode").Value()
	items := root.ValueAt(1)
	labels := root.Flag("--label").Value()
	custom := root.Flag("--custom").Value()
	pair := root.ArgumentFormAt(0).Term("pair")
	modeDomain := mode.Domain(cli.ValueDomainUseDestination)
	clitest.Schema(app).Require(t,
		root.ValueAt(3).Type(cli.ValueType{
			Kind: cli.ValueKindUint,
		}),
		custom.Type(cli.ValueType{
			Kind: cli.ValueKindCustom,
			Name: "prefixed",
		}),
		pair.TokenLanguageFragmentsExactly("LEFT", "RIGHT"),
		modeDomain.Entry("alias").Kind(cli.ValueDomainEntryAlias),
		modeDomain.Entry("alias").Canonical(modeDomain.Entry("new")),
		modeDomain.UsesExactly(modeDomain, items.Domain(cli.ValueDomainUseSequenceElement)),
	)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--label", "enabled", "left", "right"},
	}).Require(t,
		labels.Resolved(),
		labels.EntriesExactly(labels.CandidateAt(0).EntryAt(0)),
		labels.CandidateAt(0).EntryAt(0).Key().Text("enabled"),
		labels.CandidateAt(0).EntryAt(0).Value().Text("true"),
	)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--label", "name:value", "left", "right"},
	}).Require(t,
		labels.CandidateAt(0).EntryAt(0).Key().Text("name"),
		labels.CandidateAt(0).EntryAt(0).Value().Text("value"),
	)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--custom", "input", "left", "right"},
	}).Require(t,
		custom.Resolved(),
		custom.Text("codec:input"),
	)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--mode", "flag", "left", "right"},
		Env: cli.Env{{
			Name:  "MODE",
			Value: "environment",
		}},
	}).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidValue).Present()),
		mode.Failed(),
	)
}

func TestKeyOnlyPreservesZeroValueAndRejectsDuplicateImplicitKeys(t *testing.T) {
	t.Parallel()
	type config = authoringLanguageLabelsConfig
	var zero cli.KeyValueLanguage[string, string]
	clitest.CompileFails(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Values.Keyed(&root.Config.Labels, zero.KeyOnly("implicit"))
		},
	})

	clitest.CompileFails(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			language := cli.Value.Keyed(cli.Value.String(), cli.Value.String()).KeyOnly("first").KeyOnly("second")
			root.Values.Keyed(&root.Config.Labels, language)
		},
	})
}

type authoringProfileProfileVerboseConfig struct {
	Profile string
	Verbose bool
}

type authoringProfileProfileOneTwoOtherConfig struct {
	Profile         string
	One, Two, Other bool
}

type authoringProfileConfig struct{ Profile string }

func TestStageGrammarProfileSwitchPreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringProfileProfileVerboseConfig
	profileDomain := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) { values.Value("verbose", "verbose") })
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			profile := root.Values.Value(&root.Config.Profile, profileDomain)
			root.Values.Default(profile, "verbose")
			root.Profiles.Switch(profile, func(cases *cli.GrammarProfileCases[config, string]) {
				cases.Case("verbose", func(active *cli.CommandBuilder[config]) {
					active.Flags.Bool(&active.Config.Verbose).Occurrences(cli.Count.Optional()).
						Long("verbose-profile")
				})
			})
		},
	})
	root := clitest.RootCommand()
	verbose := root.GrammarProfile("verbose")
	clitest.Schema(app).Require(t,
		verbose.Exists(),
		root.GrammarProfilesExactly(verbose),
		root.Flag("--verbose-profile").Exists(),
	)
}

func TestStageGrammarProfileCasePreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringProfileProfileOneTwoOtherConfig
	profileDomain := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
		values.Value("one", "one")
		values.Value("two", "two")
		values.Value("other", "other")
	})
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			profile := root.Values.Value(&root.Config.Profile, profileDomain)
			root.Values.Default(profile, "one")
			root.Profiles.Switch(profile, func(cases *cli.GrammarProfileCases[config, string]) {
				cases.Case("one", func(active *cli.CommandBuilder[config]) {
					active.Flags.Bool(&active.Config.One).Occurrences(cli.Count.Optional()).Long("one")
				})
				cases.Case("two", func(active *cli.CommandBuilder[config]) {
					active.Flags.Bool(&active.Config.Two).Occurrences(cli.Count.Optional()).Long("two")
				})
				cases.Default(func(active *cli.CommandBuilder[config]) {
					active.Flags.Bool(&active.Config.Other).Occurrences(cli.Count.Optional()).Long("default")
				})
			})
		},
	})
	root := clitest.RootCommand()
	one := root.GrammarProfile("one")
	two := root.GrammarProfile("two")
	fallback := root.GrammarProfile("")
	clitest.Schema(app).Require(t,
		root.GrammarProfilesExactly(one, two),
		root.DefaultGrammarProfile(fallback),
		one.Exists(), two.Exists(), fallback.Exists(),
		root.Flag("--one").Exists(), root.Flag("--two").Exists(), root.Flag("--default").Exists(),
	)
}

func TestStageProfileSwitchPreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringProfileConfig
	clitest.CompileFails(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			profile := root.Values.String(&root.Config.Profile)
			root.Profiles.Switch(profile, nil)
		},
	})
}

func TestStageProfileCasePreservesAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringProfileConfig
	clitest.CompileFails(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			profile := root.Values.String(&root.Config.Profile)
			root.Profiles.Switch(profile, func(cases *cli.GrammarProfileCases[config, string]) {
				cases.Case("broken", nil)
			})
		},
	})
}

func TestGrammarDiagnosticRetainsAmbiguityContext(t *testing.T) {
	t.Parallel()
	declareParent := func(parent *cli.CommandBuilder[struct{}]) {
		parent.Dialect.CommandMatching(cli.NameMatchingUniquePrefix)
		parent.Subcommands.Command(cli.Command[struct{}]{Name: "serve"})
		parent.Subcommands.Command(cli.Command[struct{}]{Name: "service"})
	}
	parent := cli.Command[struct{}]{
		Name:    "parent",
		Declare: declareParent,
	}
	declareRoot := func(root *cli.CommandBuilder[struct{}]) {
		root.Subcommands.Command(parent)
	}
	definition := cli.Definition[struct{}]{
		Name:    "alienctl",
		Declare: declareRoot,
	}
	app := clitest.Compile(t, definition)
	request := cli.InspectionRequest{Args: []string{"parent", "ser"}}
	diagnosticLocator := clitest.Diagnostic(cli.DiagnosticAmbiguousCommand)

	proof := clitest.Inspect(t.Context(), app, request)
	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(diagnosticLocator.Present()),
	)
	_, err := proof.Inspection()
	report := cli.Diagnostics(err)
	var diagnostic *cli.Diagnostic
	for index := range report.Diagnostics {
		if report.Diagnostics[index].Code == cli.DiagnosticAmbiguousCommand {
			diagnostic = &report.Diagnostics[index]
			break
		}
	}
	if diagnostic == nil {
		t.Fatalf("diagnostics = %#v", report.Diagnostics)
	}
	if len(diagnostic.CommandPath) == 0 {
		t.Fatal("ambiguous command Diagnostic omitted CommandPath")
	}
	if diagnostic.Span == nil {
		t.Fatal("ambiguous command Diagnostic omitted Span")
	}
	if len(diagnostic.Suggestions) < 2 {
		t.Fatalf("ambiguous command suggestions = %v, want candidates/source spellings", diagnostic.Suggestions)
	}
}

func TestActionCommandSettlesPotentialProviderCollision(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	provide := func(_ context.Context, batch *cli.CommandBatch[struct{}]) error {
		calls.Add(1)
		batch.Command(cli.Command[struct{}]{Name: "help"})
		return nil
	}
	declare := func(root *cli.CommandBuilder[struct{}]) {
		root.Subcommands.CommandsFrom(provide)
	}
	definition := cli.Definition[struct{}]{
		Name:    "alienctl",
		Declare: declare,
	}
	app := clitest.Compile(t, definition)
	request := cli.InspectionRequest{Args: []string{"help"}}
	diagnostic := clitest.Diagnostic(cli.DiagnosticInvalidCommandBatch)

	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(diagnostic.Present()),
	)
	if calls.Load() == 0 {
		t.Fatal("help action command was recognized without settling potentially colliding command provider")
	}
}
