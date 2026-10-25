package conformance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

type diagnosticsSecretConfig struct{ Secret int }

type diagnosticsNameConfig struct{ Name string }

type diagnosticsModeConfig struct{ Mode string }

type diagnosticsLocaleOtherGateConfig struct {
	Locale string
	Other  string
	Gate   string
}

type diagnosticsSwitchModeConfig struct {
	Switch bool
	Mode   string
}

type diagnosticsConfig struct{}

type diagnosticsABSSeqASeqBMapAMapBConfig struct {
	A, B       int
	S          string
	SeqA, SeqB []string
	MapA, MapB map[string]string
}

type diagnosticsValueConfig2 struct{ Value bool }

type diagnosticsValueConfig3 struct{ Value uint }

type diagnosticsValueConfig4 struct{ Value diagnosticCustomValue }

type diagnosticsLeftRightConfig struct {
	Left  string
	Right string
}

type diagnosticsDerivedConfig struct{ Derived string }

type diagnosticsFactConfig struct{ Fact string }

type diagnosticsNumberConfig struct{ Number int }

func TestDiagnosticsDoesNotSynthesizeOrdinaryErrors(t *testing.T) {
	t.Parallel()
	clitest.Diagnostics(nil).Require(t, clitest.DiagnosticsExactly())
	clitest.Diagnostics(errors.New("application failure")).Require(t, clitest.DiagnosticsExactly())
}

func TestDiagnosticsExtractsDiagnosticReport(t *testing.T) {
	t.Parallel()
	proof := clitest.CompileFails(t, cli.Definition[struct{}]{
		Name: "tool",
		Localization: cli.Localization{
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"x": "y",
				},
			}},
		},
	})
	proof.Require(t, clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticInvalidDeclaration)))
}

func TestDiagnosticsProjectsRebasedSourceClosedDetachedReport(t *testing.T) {
	t.Parallel()
	type config = diagnosticsSecretConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			secret := root.Values.Int(&root.Config.Secret).Sensitive()
			root.Flags.ValueFor(secret).Long("secret").Sensitive()
		},
	}
	app := clitest.Compile(t, definition)
	args := []string{"--secret", "not-an-int"}
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: args,
	})
	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(
			clitest.Diagnostic(cli.DiagnosticInvalidValue).InputRedacted(),
		),
	)
	inspection, err := proof.Inspection()
	if err == nil {
		t.Fatal("Inspect() error = nil, want invalid-value failure")
	}
	diagnostics := clitest.Diagnostics(err)
	diagnostics.Require(t,
		clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticInvalidValue)),
		clitest.Diagnostic(cli.DiagnosticInvalidValue).InputRedacted(),
		clitest.DiagnosticInputGraphClosed(),
	)

	args[0], args[1] = "--mutated", "mutated"
	inspection.Diagnostics = nil
	inspection.Sources = nil
	first := diagnostics.Report()
	if len(first.Diagnostics) != 0 {
		first.Diagnostics[0].Code = cli.DiagnosticUnknownFlag
		first.Diagnostics[0].InputRedacted = false
	}
	clear(first.Sources)
	clear(first.Origins)

	clitest.Diagnostics(err).Require(t,
		clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticInvalidValue)),
		clitest.Diagnostic(cli.DiagnosticInvalidValue).InputRedacted(),
		clitest.DiagnosticInputGraphClosed(),
	)
}

func TestDiagnosticsProjectsStructuredIdentityClassificationAndSourceMetadata(t *testing.T) {
	t.Parallel()
	type config = diagnosticsNameConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			name := root.Values.String(&root.Config.Name)
			root.Flags.ValueFor(name).Long("name")
			root.Rule.Require(name)
		},
	}
	app := clitest.Compile(t, definition)
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{})
	rule := clitest.Diagnostic(cli.DiagnosticRuleViolation).On(clitest.RootCommand())
	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.RootCommand().Exists(),
		clitest.RootCommand().ValueAt(0).Observed(),
		clitest.FailsWith(
			rule.Present(),
			rule.RuleKind(cli.RuleRequire),
			rule.RuleIdentityPresent(),
		),
	)
	_, err := proof.Inspection()
	if err == nil {
		t.Fatal("Inspect() error = nil, want rule violation")
	}
	diagnostics := clitest.Diagnostics(err)
	diagnostics.Require(t, rule.Present(), rule.RuleKind(cli.RuleRequire), rule.RuleIdentityPresent())
	first := diagnostics.Report()
	for i := range first.Diagnostics {
		if first.Diagnostics[i].Rule != nil {
			first.Diagnostics[i].Rule.RuleKind = cli.RuleUnknown
		}
	}
	clitest.Diagnostics(err).Require(t, rule.Present(), rule.RuleKind(cli.RuleRequire), rule.RuleIdentityPresent())
}

func TestDiagnosticsProjectsIntegrationFailureMetadata(t *testing.T) {
	t.Parallel()

	sender := &cli.SignalSender{
		ProcessID: 12,
		UserID:    34,
		Code:      56,
	}
	observedAt := time.Unix(1700000200, 31)
	actionCause := errors.New("signal action failed")
	actionDone := make(chan struct{})
	actionSource, actionControl := clitest.SignalSource(clitest.SignalSourceOptions[string]{
		Signals: []cli.SignalDescriptor[string]{{
			Signal:   "diagnostic",
			Fidelity: cli.SignalSourceDeliveryFidelityExact,
		}},
	})
	var signal cli.Signal
	definition := cli.Definition[struct{}]{
		Name: "tool",
		Run: func(ctx context.Context, _ *struct{}) error {
			select {
			case <-actionDone:
				return nil
			case <-ctx.Done():
				return context.Cause(ctx)
			}
		},
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			registered := root.Signals.Source(actionSource)
			signal = registered.Signal("diagnostic")
			root.Signals.Policy(
				root.Signals.Match.On(signal).During(cli.SignalPhaseExecution).Each(),
				root.Signals.Action.Handle(cli.SignalHandler[struct{}]{
					Name: "notify",
					Handle: func(context.Context, *struct{}, cli.SignalEvent) error {
						close(actionDone)
						return actionCause
					},
				}),
			)
		},
	}
	actionApp := clitest.Compile(t, definition)
	actionDoneResult := make(chan error, 1)
	go func() { actionDoneResult <- actionApp.Run(t.Context(), cli.RunRequest{}) }()
	select {
	case <-actionControl.Opened():
	case <-time.After(2 * time.Second):
		t.Fatal("signal source never became active")
	}
	if err := actionControl.Notify(cli.SignalNotification[string]{
		Signal:     "diagnostic",
		Origin:     cli.SignalOriginProcess,
		Sender:     sender,
		ObservedAt: observedAt,
	}); err != nil {
		t.Fatal(err)
	}
	var actionErr error
	select {
	case actionErr = <-actionDoneResult:
	case <-time.After(3 * time.Second):
		t.Fatal("signal action failure Run did not complete")
	}
	if actionErr == nil || !errors.Is(actionErr, actionCause) {
		t.Fatalf("signal action Run error = %v, want action cause", actionErr)
	}

	overflowCause := &cli.SignalSourceOverflowError{
		Capacity: 4,
		Buffered: 4,
	}
	overflowSource, overflowControl := clitest.SignalSource(clitest.SignalSourceOptions[string]{
		Signals: []cli.SignalDescriptor[string]{{
			Signal:   "overflow",
			Fidelity: cli.SignalSourceDeliveryFidelityExact,
		}},
	})
	definitionUnderTest := cli.Definition[struct{}]{
		Name: "overflow",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			semantic := root.Signals.Source(overflowSource).Signal("overflow")
			root.Signals.Policy(
				root.Signals.Match.On(semantic).During(cli.SignalPhaseExecution).Each(),
				root.Signals.Action.Ignore(),
			)
		},
		Run: func(ctx context.Context, _ *struct{}) error {
			<-ctx.Done()
			return context.Cause(ctx)
		},
	}
	overflowApp := clitest.Compile(t, definitionUnderTest)
	overflowDone := make(chan error, 1)
	go func() { overflowDone <- overflowApp.Run(t.Context(), cli.RunRequest{}) }()
	select {
	case <-overflowControl.Opened():
	case <-time.After(2 * time.Second):
		t.Fatal("overflow source never became active")
	}
	if err := overflowControl.Fail(overflowCause); err != nil {
		t.Fatal(err)
	}
	var overflowErr error
	select {
	case overflowErr = <-overflowDone:
	case <-time.After(3 * time.Second):
		t.Fatal("signal overflow Run did not complete")
	}
	if overflowErr == nil || !errors.Is(overflowErr, overflowCause) {
		t.Fatalf("signal overflow Run error = %v, want overflow cause", overflowErr)
	}

	noticeCause := errors.New("notice delivery failed")
	noticeApp := diagnosticNoticeDeliveryApp(t, nil, "migration")
	noticeErr := noticeApp.Run(t.Context(), cli.RunRequest{
		Args:   []string{"child"},
		Notify: func(context.Context, cli.Notice) error { return noticeCause },
	})
	if noticeErr == nil || !errors.Is(noticeErr, noticeCause) {
		t.Fatalf("notice Run error = %v, want delivery cause", noticeErr)
	}

	combined := errors.Join(actionErr, overflowErr, noticeErr)
	action := clitest.Diagnostic(cli.DiagnosticSignalActionFailure).SignalAction()
	overflow := clitest.Diagnostic(cli.DiagnosticSignalOverflow).SignalOverflow()
	notice := clitest.Diagnostic(cli.DiagnosticNoticeDeliveryFailure).NoticeDelivery()
	expectations := []clitest.DiagnosticReportExpectation{
		action.PolicyIdentityPresent(),
		action.Index(0),
		action.Phase(cli.SignalPhaseExecution),
		action.Action().Kind(cli.SignalActionHandle),
		action.Action().Name("notify"),
		action.Event().Signal(signal),
		action.Event().Origin(cli.SignalOriginProcess),
		action.Event().Sender(cli.SignalSender{
			ProcessID: 12,
			UserID:    34,
			Code:      56,
		}),
		action.Event().Phase(cli.SignalPhaseExecution),
		action.Event().ObservedAt(observedAt),
		overflow.Boundary(cli.SignalOverflowBoundarySource),
		overflow.NoEndpoint(),
		overflow.Capacity(4),
		overflow.Buffered(4),
		notice.Index(0),
		notice.Phase(cli.NoticePhaseAnalysis),
		notice.Code(cli.NoticeAdvisory),
		notice.CommandPathExactly("child"),
	}
	diagnostics := clitest.Diagnostics(combined)
	diagnostics.Require(t, expectations...)

	sender.ProcessID = 99
	first := diagnostics.Report()
	for i := range first.Diagnostics {
		if first.Diagnostics[i].SignalAction != nil {
			first.Diagnostics[i].SignalAction.Action.Name = "mutated"
			if first.Diagnostics[i].SignalAction.Event.Sender != nil {
				first.Diagnostics[i].SignalAction.Event.Sender.ProcessID = 98
			}
		}
		if first.Diagnostics[i].NoticeDelivery != nil && len(first.Diagnostics[i].NoticeDelivery.CommandPath) != 0 {
			first.Diagnostics[i].NoticeDelivery.CommandPath[0] = "mutated"
		}
	}
	clitest.Diagnostics(combined).Require(t, expectations...)
}

func TestRemovedFlagSpellingReplacementOmitsUnprovenMachineRepair(t *testing.T) {
	t.Parallel()
	type config = diagnosticsModeConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.SemVer("2.0.0"),
		Declare: func(root *cli.CommandBuilder[config]) {
			flag := root.Flags.String(&root.Config.Mode)
			old := flag.Long("old").FixedValue("reference").Release(cli.ReleaseLifecycle{
				RemovedIn: "2.0.0",
			})
			newer := flag.Long("new")
			old.ReplacedBy(newer)
		},
	}
	app := clitest.Compile(t, definition)
	schema := clitest.Schema(app)
	old := clitest.ResolveDeclaration(t, schema, clitest.RootCommand().Flag("--old").Spelling("--old"))
	newer := clitest.ResolveDeclaration(t, schema, clitest.RootCommand().Flag("--new").Spelling("--new"))
	schema.Require(t,
		old.ReleaseLifecycle(cli.ReleaseLifecycle{
			RemovedIn: "2.0.0",
		}),
		old.ReplacedBy(newer),
	)
	diagnostic := clitest.Diagnostic(cli.DiagnosticRemovedFlag).On(old)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--old"},
	}).Require(t,
		clitest.FailsWith(diagnostic.Present(), diagnostic.RepairCount(0), clitest.RepairsExactly()),
	)
}

func TestLocaleSelectorInvalidStaticDeclarationsUseInvalidDeclarationDiagnostic(t *testing.T) {
	t.Parallel()
	type config = diagnosticsLocaleOtherGateConfig
	tests := []struct {
		name    string
		declare func(*cli.CommandBuilder[config])
	}{
		{
			name: "repeatable selector grammar",
			declare: func(root *cli.CommandBuilder[config]) {
				root.Flags.String(&root.Config.Locale).LocaleSelector().Long("locale")
			},
		},
		{
			name: "duplicate",
			declare: func(root *cli.CommandBuilder[config]) {
				root.Flags.String(&root.Config.Locale).LocaleSelector().Occurrences(cli.Count.Optional()).
					Long("locale")
				root.Flags.String(&root.Config.Other).LocaleSelector().Occurrences(cli.Count.Optional()).
					Long("language")
			},
		},
		{
			name: "availability conditioned",
			declare: func(root *cli.CommandBuilder[config]) {
				gate := root.Values.String(&root.Config.Gate)
				root.Values.Default(gate, "yes")
				flag := root.Flags.String(
					&root.Config.Locale,
				).LocaleSelector().Occurrences(cli.Count.Optional())

				flag.Long("locale")
				flag.AvailableWhen(root.When.Resolved(gate))
			},
		},
		{
			name: "visibility conditioned",
			declare: func(root *cli.CommandBuilder[config]) {
				gate := root.Values.String(&root.Config.Gate)
				root.Values.Default(gate, "yes")
				flag := root.Flags.String(
					&root.Config.Locale,
				).LocaleSelector().Occurrences(cli.Count.Optional())

				flag.Long("locale")
				flag.VisibleWhen(root.When.Resolved(gate))
			},
		},
		{
			name: "sensitive selector spelling",
			declare: func(root *cli.CommandBuilder[config]) {
				root.Flags.String(&root.Config.Locale).LocaleSelector().Occurrences(cli.Count.Optional()).
					Long("locale").
					Sensitive()
			},
		},
		{
			name: "sensitive static producer",
			declare: func(root *cli.CommandBuilder[config]) {
				locale := root.Values.String(&root.Config.Locale)
				root.Values.Default(locale, "en").Sensitive()
				root.Flags.ValueFor(locale).LocaleSelector().Occurrences(cli.Count.Optional()).
					Long("locale")
			},
		},
		{
			name: "sensitive environment producer",
			declare: func(root *cli.CommandBuilder[config]) {
				locale := root.Values.String(&root.Config.Locale)
				root.Values.Environment(locale, "TOOL_LOCALE").Sensitive()
				root.Flags.ValueFor(locale).LocaleSelector().Occurrences(cli.Count.Optional()).
					Long("locale")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			proof := clitest.CompileFails(t, cli.Definition[config]{
				Name:    "tool",
				Declare: test.declare,
			})
			proof.Require(t, clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticInvalidDeclaration)))
		})
	}
}

func TestLifecycleDiagnosticsAttributeExactInactiveDomainEntryAndFlagSpelling(t *testing.T) {
	t.Parallel()
	type config = diagnosticsSwitchModeConfig
	definition := cli.Definition[config]{
		Name: "lifecycle",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Release: cli.Release.SemVer("2.0.0"),
		Declare: func(root *cli.CommandBuilder[config]) {
			flag := root.Flags.Bool(&root.Config.Switch)
			flag.Long("future-flag").Release(cli.ReleaseLifecycle{
				IntroducedIn: "3.0.0",
			})
			flag.Long("removed-flag").Release(cli.ReleaseLifecycle{
				RemovedIn: "2.0.0",
			})
			domain := cli.Value.Domain(cli.Value.Assign[string](), func(builder *cli.ValueDomainBuilder[string]) {
				builder.Value("future-value", "future").Release(cli.ReleaseLifecycle{
					IntroducedIn: "3.0.0",
				})
				builder.Value("removed-value", "removed").Release(cli.ReleaseLifecycle{
					RemovedIn: "2.0.0",
				})
			})
			root.Flags.Value(&root.Config.Mode, domain).Long("mode")
		},
	}
	app := clitest.Compile(t, definition)

	schema := clitest.Schema(app)
	futureSpelling := clitest.ResolveDeclaration(
		t,
		schema,
		clitest.RootCommand().Flag("--future-flag").Spelling("--future-flag"),
	)
	removedSpelling := clitest.ResolveDeclaration(
		t,
		schema,
		clitest.RootCommand().Flag("--removed-flag").Spelling("--removed-flag"),
	)
	futureValue := clitest.ResolveDeclaration(
		t,
		schema,
		clitest.RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination).Entry("future-value"),
	)
	removedValue := clitest.ResolveDeclaration(
		t,
		schema,
		clitest.RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination).Entry("removed-value"),
	)

	for _, test := range []struct {
		name    string
		args    []string
		code    cli.DiagnosticCode
		subject clitest.DiagnosticSubject
	}{
		{
			name:    "future spelling",
			args:    []string{"--future-flag"},
			code:    cli.DiagnosticNotIntroducedFlag,
			subject: futureSpelling,
		},
		{
			name:    "removed spelling",
			args:    []string{"--removed-flag"},
			code:    cli.DiagnosticRemovedFlag,
			subject: removedSpelling,
		},
		{
			name:    "future value",
			args:    []string{"--mode", "future-value"},
			code:    cli.DiagnosticNotIntroducedValue,
			subject: futureValue,
		},
		{
			name:    "removed value",
			args:    []string{"--mode", "removed-value"},
			code:    cli.DiagnosticRemovedValue,
			subject: removedValue,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			diagnostic := clitest.Diagnostic(test.code).On(test.subject)
			proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
				Args: test.args,
			})
			proof.Require(t,
				clitest.FailsWith(clitest.DiagnosticsExactly(diagnostic), diagnostic.Present()),
			)
		})
	}
}

func TestRunPreservesApplicationHandlerErrorWithoutSyntheticDiagnostic(t *testing.T) {
	t.Parallel()
	type config = diagnosticsConfig
	want := errors.New("application failure")
	app, err := cli.Compile(cli.Definition[config]{
		Name: "commandment",
		Run:  func(context.Context, *config) error { return want },
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	got := app.Run(t.Context(), cli.RunRequest{})
	if !errors.Is(got, want) {
		t.Fatalf("Run() error = %v, want preserved cause", got)
	}
	if report := cli.Diagnostics(got); len(report.Diagnostics) != 0 {
		t.Fatalf("application error synthesized Diagnostics = %#v", report)
	}
}

func TestProjectDiagnosticReportPreservesDiagnosticSemantics(t *testing.T) {
	t.Parallel()
	proof := clitest.CompileFails(t, cli.Definition[struct{}]{
		Name: "tool",
		Localization: cli.Localization{
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"x": "y",
				},
			}},
		},
	})
	proof.Require(t, clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticInvalidDeclaration)))
}

func TestDiagnosticRuleFailurePreservesDiagnosticSemantics(t *testing.T) {
	t.Parallel()
	proof := clitest.CompileFails(t, cli.Definition[struct{}]{
		Name: "tool",
		Localization: cli.Localization{
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"x": "y",
				},
			}},
		},
	})
	proof.Require(t, clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticInvalidDeclaration)))
}

func TestDiagnosticValueTypePreservesDiagnosticSemantics(t *testing.T) {
	t.Parallel()
	proof := clitest.CompileFails(t, cli.Definition[struct{}]{
		Name: "tool",
		Localization: cli.Localization{
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"x": "y",
				},
			}},
		},
	})
	proof.Require(t, clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticInvalidDeclaration)))
}

func TestProjectDiagnosticSignalActionKindPreservesDiagnosticSemantics(t *testing.T) {
	t.Parallel()
	proof := clitest.CompileFails(t, cli.Definition[struct{}]{
		Name: "tool",
		Localization: cli.Localization{
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"x": "y",
				},
			}},
		},
	})
	proof.Require(t, clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticInvalidDeclaration)))
}

func TestProjectSignalPhasePreservesDiagnosticSemantics(t *testing.T) {
	t.Parallel()
	proof := clitest.CompileFails(t, cli.Definition[struct{}]{
		Name: "tool",
		Localization: cli.Localization{
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"x": "y",
				},
			}},
		},
	})
	proof.Require(t, clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticInvalidDeclaration)))
}

func TestProjectSignalOriginPreservesDiagnosticSemantics(t *testing.T) {
	t.Parallel()
	proof := clitest.CompileFails(t, cli.Definition[struct{}]{
		Name: "tool",
		Localization: cli.Localization{
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"x": "y",
				},
			}},
		},
	})
	proof.Require(t, clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticInvalidDeclaration)))
}

func TestProjectDiagnosticSignalOverflowBoundaryPreservesDiagnosticSemantics(t *testing.T) {
	t.Parallel()
	proof := clitest.CompileFails(t, cli.Definition[struct{}]{
		Name: "tool",
		Localization: cli.Localization{
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"x": "y",
				},
			}},
		},
	})
	proof.Require(t, clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticInvalidDeclaration)))
}

func TestProjectDiagnosticNoticePhasePreservesDiagnosticSemantics(t *testing.T) {
	t.Parallel()
	proof := clitest.CompileFails(t, cli.Definition[struct{}]{
		Name: "tool",
		Localization: cli.Localization{
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"x": "y",
				},
			}},
		},
	})
	proof.Require(t, clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticInvalidDeclaration)))
}

func TestDiagnosticProjectRuleKindsPreservePolicy(t *testing.T) {
	t.Parallel()
	type config = diagnosticsABSSeqASeqBMapAMapBConfig
	tests := []struct {
		name    string
		want    cli.RuleKind
		declare func(*cli.CommandBuilder[config])
	}{
		{
			name: "forbid",
			want: cli.RuleForbid,
			declare: func(root *cli.CommandBuilder[config]) {
				a := root.Values.Int(&root.Config.A)
				root.Values.Default(a, 1)
				root.Rule.Forbid(a)
			},
		},
		{
			name: "not empty",
			want: cli.RuleNotEmpty,
			declare: func(root *cli.CommandBuilder[config]) {
				s := root.Values.String(&root.Config.S)
				root.Values.Default(s, "")
				root.Rule.NotEmpty(s)
			},
		},
		{
			name: "in",
			want: cli.RuleIn,
			declare: func(root *cli.CommandBuilder[config]) {
				a := root.Values.Int(&root.Config.A)
				root.Values.Default(a, 3)
				root.Rule.In(a, 1, 2)
			},
		},
		{
			name: "not in",
			want: cli.RuleNotIn,
			declare: func(root *cli.CommandBuilder[config]) {
				a := root.Values.Int(&root.Config.A)
				root.Values.Default(a, 3)
				root.Rule.NotIn(a, 3, 4)
			},
		},
		{
			name: "equal",
			want: cli.RuleEqual,
			declare: func(root *cli.CommandBuilder[config]) {
				a := root.Values.Int(&root.Config.A)
				root.Values.Default(a, 4)
				root.Rule.Equal(a, 5)
			},
		},
		{
			name: "not equal",
			want: cli.RuleNotEqual,
			declare: func(root *cli.CommandBuilder[config]) {
				a := root.Values.Int(&root.Config.A)
				root.Values.Default(a, 6)
				root.Rule.NotEqual(a, 6)
			},
		},
		{
			name: "between",
			want: cli.RuleBetween,
			declare: func(root *cli.CommandBuilder[config]) {
				a := root.Values.Int(&root.Config.A)
				root.Values.Default(a, 0)
				root.Rule.Between(a, 1, 10)
			},
		},
		{
			name: "at least",
			want: cli.RuleAtLeast,
			declare: func(root *cli.CommandBuilder[config]) {
				a := root.Values.Int(&root.Config.A)
				root.Values.Default(a, 6)
				root.Rule.AtLeast(a, 7)
			},
		},
		{
			name: "at most",
			want: cli.RuleAtMost,
			declare: func(root *cli.CommandBuilder[config]) {
				a := root.Values.Int(&root.Config.A)
				root.Values.Default(a, 9)
				root.Rule.AtMost(a, 8)
			},
		},
		{
			name: "greater than",
			want: cli.RuleGreaterThan,
			declare: func(root *cli.CommandBuilder[config]) {
				a := root.Values.Int(&root.Config.A)
				root.Values.Default(a, 9)
				root.Rule.GreaterThan(a, 9)
			},
		},
		{
			name: "less than",
			want: cli.RuleLessThan,
			declare: func(root *cli.CommandBuilder[config]) {
				a := root.Values.Int(&root.Config.A)
				root.Values.Default(a, 10)
				root.Rule.LessThan(a, 10)
			},
		},
		{
			name: "match",
			want: cli.RuleMatch,
			declare: func(root *cli.CommandBuilder[config]) {
				s := root.Values.String(&root.Config.S)
				root.Values.Default(s, "bad")
				root.Rule.Match(s, "^good$")
			},
		},
		{
			name: "item count",
			want: cli.RuleItemCount,
			declare: func(root *cli.CommandBuilder[config]) {
				v := root.Values.Sequence(&root.Config.SeqA, cli.Value.String())
				root.Values.Default(v, []string{})
				root.Rule.ItemCount(v, cli.Count.Exactly(1))
			},
		},
		{
			name: "contains",
			want: cli.RuleContains,
			declare: func(root *cli.CommandBuilder[config]) {
				v := root.Values.Sequence(&root.Config.SeqA, cli.Value.String())
				root.Values.Default(v, []string{"x"})
				root.Rule.Contains(v, "a")
			},
		},
		{
			name: "each in",
			want: cli.RuleEachIn,
			declare: func(root *cli.CommandBuilder[config]) {
				v := root.Values.Sequence(&root.Config.SeqA, cli.Value.String())
				root.Values.Default(v, []string{"c"})
				root.Rule.EachIn(v, "a", "b")
			},
		},
		{
			name: "unique",
			want: cli.RuleUnique,
			declare: func(root *cli.CommandBuilder[config]) {
				v := root.Values.Sequence(&root.Config.SeqA, cli.Value.String())
				root.Values.Default(v, []string{"a", "a"})
				root.Rule.Unique(v)
			},
		},
		{
			name: "entry count",
			want: cli.RuleEntryCount,
			declare: func(root *cli.CommandBuilder[config]) {
				v := root.Values.Keyed(&root.Config.MapA, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
				root.Values.Default(v, map[string]string{"a": "1"})
				root.Rule.EntryCount(v, cli.Count.Exactly(2))
			},
		},
		{
			name: "has keys",
			want: cli.RuleHasKeys,
			declare: func(root *cli.CommandBuilder[config]) {
				v := root.Values.Keyed(&root.Config.MapA, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
				root.Values.Default(v, map[string]string{"b": "1"})
				root.Rule.HasKeys(v, "a")
			},
		},
		{
			name: "keys in",
			want: cli.RuleKeysIn,
			declare: func(root *cli.CommandBuilder[config]) {
				v := root.Values.Keyed(&root.Config.MapA, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
				root.Values.Default(v, map[string]string{"c": "1"})
				root.Rule.KeysIn(v, "a", "b")
			},
		},
		{
			name: "subset of",
			want: cli.RuleSubsetOf,
			declare: func(root *cli.CommandBuilder[config]) {
				a := root.Values.Sequence(&root.Config.SeqA, cli.Value.String())
				b := root.Values.Sequence(&root.Config.SeqB, cli.Value.String())
				root.Values.Default(a, []string{"a"})
				root.Values.Default(b, []string{"b"})
				root.Rule.Compare.SubsetOf(a, b)
			},
		},
		{
			name: "disjoint",
			want: cli.RuleDisjoint,
			declare: func(root *cli.CommandBuilder[config]) {
				a := root.Values.Sequence(&root.Config.SeqA, cli.Value.String())
				b := root.Values.Sequence(&root.Config.SeqB, cli.Value.String())
				root.Values.Default(a, []string{"a"})
				root.Values.Default(b, []string{"a"})
				root.Rule.Compare.Disjoint(a, b)
			},
		},
		{
			name: "keys subset of",
			want: cli.RuleKeysSubsetOf,
			declare: func(root *cli.CommandBuilder[config]) {
				a := root.Values.Keyed(&root.Config.MapA, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
				b := root.Values.Keyed(&root.Config.MapB, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
				root.Values.Default(a, map[string]string{"a": "1"})
				root.Values.Default(b, map[string]string{"b": "1"})
				root.Rule.Compare.KeysSubsetOf(a, b)
			},
		},
		{
			name: "keys disjoint",
			want: cli.RuleKeysDisjoint,
			declare: func(root *cli.CommandBuilder[config]) {
				a := root.Values.Keyed(&root.Config.MapA, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
				b := root.Values.Keyed(&root.Config.MapB, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
				root.Values.Default(a, map[string]string{"a": "1"})
				root.Values.Default(b, map[string]string{"a": "2"})
				root.Rule.Compare.KeysDisjoint(a, b)
			},
		},
		{
			name: "exactly one",
			want: cli.RuleExactlyOne,
			declare: func(root *cli.CommandBuilder[config]) {
				a := root.Values.Int(&root.Config.A)
				b := root.Values.Int(&root.Config.B)
				root.Values.Default(a, 1)
				root.Values.Default(b, 2)
				root.Rule.ExactlyOne(a, b)
			},
		},
		{
			name: "all or none",
			want: cli.RuleAllOrNone,
			declare: func(root *cli.CommandBuilder[config]) {
				a := root.Values.Int(&root.Config.A)
				b := root.Values.Int(&root.Config.B)
				root.Values.Default(a, 1)
				root.Rule.AllOrNone(a, b)
			},
		},
		{
			name: "at most one",
			want: cli.RuleAtMostOne,
			declare: func(root *cli.CommandBuilder[config]) {
				a := root.Values.Int(&root.Config.A)
				b := root.Values.Int(&root.Config.B)
				root.Values.Default(a, 1)
				root.Values.Default(b, 2)
				root.Rule.AtMostOne(a, b)
			},
		},
		{
			name: "at least one",
			want: cli.RuleAtLeastOne,
			declare: func(root *cli.CommandBuilder[config]) {
				a := root.Values.Int(&root.Config.A)
				b := root.Values.Int(&root.Config.B)
				root.Rule.AtLeastOne(a, b)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			definition := cli.Definition[config]{
				Name: "tool",
				Help: cli.HelpConfig{
					Disabled: true,
				},
				Declare: tc.declare,
			}
			app := clitest.Compile(t, definition)
			clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticRuleViolation).RuleKind(tc.want)),
			)
		})
	}
}

type diagnosticCustomValue string

func TestDiagnosticsProjectScalarRuleLiteralValueTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want cli.ValueType
		app  func(*testing.T) cli.Application
	}{
		{
			name: "bool",
			want: cli.ValueType{
				Kind: cli.ValueKindBool,
			},
			app: func(t *testing.T) cli.Application {
				t.Helper()
				type config = diagnosticsValueConfig2
				helpConfig := cli.HelpConfig{
					Disabled: true,
				}
				definition := cli.Definition[config]{
					Name: "tool",
					Help: helpConfig,
					Declare: func(root *cli.CommandBuilder[config]) {
						value := root.Values.Bool(&root.Config.Value)
						root.Values.Default(value, false)
						root.Rule.Equal(value, true)
					},
				}
				return clitest.Compile(t, definition)
			},
		},
		{
			name: "uint",
			want: cli.ValueType{
				Kind: cli.ValueKindUint,
			},
			app: func(t *testing.T) cli.Application {
				t.Helper()
				type config = diagnosticsValueConfig3
				helpConfig := cli.HelpConfig{
					Disabled: true,
				}
				definition := cli.Definition[config]{
					Name: "tool",
					Help: helpConfig,
					Declare: func(root *cli.CommandBuilder[config]) {
						value := root.Values.Uint(&root.Config.Value)
						root.Values.Default(value, 1)
						root.Rule.Equal(value, uint(2))
					},
				}
				return clitest.Compile(t, definition)
			},
		},
		{
			name: "custom",
			want: cli.ValueType{
				Kind: cli.ValueKindCustom,
				Name: "custom",
			},
			app: func(t *testing.T) cli.Application {
				t.Helper()
				type config = diagnosticsValueConfig4
				helpConfig := cli.HelpConfig{
					Disabled: true,
				}
				definition := cli.Definition[config]{
					Name: "tool",
					Help: helpConfig,
					Declare: func(root *cli.CommandBuilder[config]) {
						value := root.Values.Value(&root.Config.Value, cli.ValueCodec[diagnosticCustomValue]{
							Name:     "custom",
							Parse:    func(text string) (diagnosticCustomValue, error) { return diagnosticCustomValue(text), nil },
							Format:   func(value diagnosticCustomValue) string { return string(value) },
							Snapshot: cli.Value.Assign[diagnosticCustomValue](),
						})
						root.Values.Default(value, diagnosticCustomValue("a"))
						root.Rule.Equal(value, diagnosticCustomValue("b"))
					},
				}
				return clitest.Compile(t, definition)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			literal := clitest.Diagnostic(cli.DiagnosticRuleViolation).RuleOperandAt(0).Literal()
			clitest.Inspect(t.Context(), test.app(t), cli.InspectionRequest{}).Require(t,
				clitest.InspectionRejected(),
				clitest.FailsWith(literal.Type(test.want)),
			)
		})
	}
}

func TestDiagnosticViolationKindsPreservePublic(t *testing.T) {
	t.Parallel()
	type config = diagnosticsLeftRightConfig
	var duplicateOrigins []cli.InputOriginID
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			left := root.Values.String(&root.Config.Left)
			right := root.Values.String(&root.Config.Right)
			root.Flags.ValueFor(left).Long("left")
			root.Flags.ValueFor(right).Long("right")
			root.Rule.Validate("duplicate", func(ctx context.Context, _ *config, _ cli.Dependencies) ([]cli.Violation, error) {
				leftState, leftOK := cli.ValueStateFromContext(ctx, left)
				rightState, rightOK := cli.ValueStateFromContext(ctx, right)
				if !leftOK || !rightOK || len(leftState.Candidates()) == 0 || len(rightState.Candidates()) == 0 {
					return nil, errors.New("value state unavailable")
				}
				leftOrigins := leftState.Candidates()[0].Origins()
				rightOrigins := rightState.Candidates()[0].Origins()
				if len(leftOrigins) == 0 || len(rightOrigins) == 0 {
					return nil, errors.New("input origins unavailable")
				}
				duplicateOrigins = []cli.InputOriginID{leftOrigins[0], rightOrigins[0]}
				violation := cli.Violation{
					Kind:    cli.ViolationDuplicate,
					Origins: append([]cli.InputOriginID(nil), duplicateOrigins...),
				}
				return []cli.Violation{violation}, nil
			}).
				Subject(left).
				Subject(right).
				DependsOn(left).
				DependsOn(right)
			root.Rule.Validate("conflict", func(context.Context, *config, cli.Dependencies) ([]cli.Violation, error) {
				violation := cli.Violation{
					Kind: cli.ViolationConflict,
				}
				return []cli.Violation{violation}, nil
			})
		},
	}
	app := clitest.Compile(t, definition)
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--left", "a", "--right", "b"},
	})
	inspection, inspectionErr := proof.Inspection()
	if inspectionErr == nil {
		t.Fatal("inspection succeeded, want rule violations")
	}
	if len(duplicateOrigins) != 2 {
		t.Fatalf("validator origins = %v, want 2", duplicateOrigins)
	}

	duplicate := clitest.Diagnostic(cli.DiagnosticRuleViolation).Rule("duplicate")
	conflict := clitest.Diagnostic(cli.DiagnosticRuleViolation).Rule("conflict")
	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(duplicate.Present(), conflict.Present()),
	)
	clitest.Diagnostics(inspectionErr).Require(t,
		duplicate.Violation(cli.ViolationDuplicate),
		conflict.Violation(cli.ViolationConflict),
	)

	report := cli.Diagnostics(inspectionErr)
	var duplicateDiagnostic, conflictDiagnostic *cli.Diagnostic
	for index := range report.Diagnostics {
		diagnostic := &report.Diagnostics[index]
		if diagnostic.Code != cli.DiagnosticRuleViolation || diagnostic.Rule == nil {
			continue
		}
		switch diagnostic.Rule.RuleCode {
		case "duplicate":
			duplicateDiagnostic = diagnostic
		case "conflict":
			conflictDiagnostic = diagnostic
		}
	}
	if duplicateDiagnostic == nil || conflictDiagnostic == nil {
		t.Fatalf("rule diagnostics = %#v, want duplicate and conflict", report.Diagnostics)
	}
	if len(duplicateDiagnostic.Origins) != len(duplicateOrigins) {
		t.Fatalf("duplicate diagnostic origins = %v, want %d", duplicateDiagnostic.Origins, len(duplicateOrigins))
	}
	if len(conflictDiagnostic.Origins) != 0 {
		t.Fatalf("conflict diagnostic origins = %v, want none", conflictDiagnostic.Origins)
	}
	for index, expectedID := range duplicateOrigins {
		expectedOrigin, ok := inspection.Origins[expectedID]
		if !ok {
			t.Fatalf("inspection origin %v is absent", expectedID)
		}
		actualID := duplicateDiagnostic.Origins[index]
		actualOrigin, ok := report.Origins[actualID]
		if !ok {
			t.Fatalf("diagnostic report origin %v is absent", actualID)
		}
		expectedSource, ok := inspection.Sources[expectedOrigin.Span.Source]
		if !ok {
			t.Fatalf("inspection source %v is absent", expectedOrigin.Span.Source)
		}
		actualSource, ok := report.Sources[actualOrigin.Span.Source]
		if !ok {
			t.Fatalf("diagnostic report source %v is absent", actualOrigin.Span.Source)
		}
		if expectedOrigin.Span.Entry != actualOrigin.Span.Entry ||
			expectedOrigin.Span.Start != actualOrigin.Span.Start ||
			expectedOrigin.Span.End != actualOrigin.Span.End ||
			expectedOrigin.Sensitive != actualOrigin.Sensitive ||
			len(expectedOrigin.Parents) != len(actualOrigin.Parents) {
			t.Fatalf(
				"duplicate origin %d semantic projection = %#v, want %#v modulo report-local identity",
				index,
				actualOrigin,
				expectedOrigin,
			)
		}
		if expectedSource.Kind != actualSource.Kind || expectedSource.CommandShortcut != actualSource.CommandShortcut ||
			expectedSource.Key != actualSource.Key ||
			expectedSource.Location != actualSource.Location ||
			expectedSource.Sensitive != actualSource.Sensitive {
			t.Fatalf(
				"duplicate origin %d source projection = %#v, want %#v modulo report-local identity",
				index,
				actualSource,
				expectedSource,
			)
		}
	}
}

func TestDiagnosticsPreserveValueFailureSourceTierDispositionAndDeclaration(t *testing.T) {
	t.Parallel()

	t.Run("argument", func(t *testing.T) {
		t.Parallel()
		type config = diagnosticsNumberConfig
		definition := cli.Definition[config]{
			Name: "tool",
			Help: cli.HelpConfig{
				Disabled: true,
			},
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Arguments.Int(&root.Config.Number, "number")
			},
		}
		app := clitest.Compile(t, definition)
		schema := clitest.Schema(app)
		subject := clitest.ResolveDeclaration(t, schema, clitest.RootCommand().Argument("number"))
		diagnostic := clitest.Diagnostic(cli.DiagnosticInvalidValue).On(subject)
		proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"not-an-int"},
		})
		proof.Require(t,
			clitest.InspectionRejected(),
			clitest.FailsWith(
				clitest.DiagnosticsExactly(diagnostic),
				diagnostic.ProducerKind(cli.ValueProducerKindArgument),
				diagnostic.Tier(cli.ResolutionTierExplicit),
				diagnostic.Disposition(cli.ValueCandidateDispositionRejected),
			),
		)
	})

	t.Run("environment", func(t *testing.T) {
		t.Parallel()
		type config = diagnosticsNumberConfig
		definition := cli.Definition[config]{
			Name: "tool",
			Help: cli.HelpConfig{
				Disabled: true,
			},
			Declare: func(root *cli.CommandBuilder[config]) {
				number := root.Values.Int(&root.Config.Number)
				root.Values.Environment(number, "TOOL_NUMBER")
			},
		}
		app := clitest.Compile(t, definition)
		schema := clitest.Schema(app)
		subject := clitest.ResolveDeclaration(t, schema, clitest.RootCommand().Environment("TOOL_NUMBER"))
		diagnostic := clitest.Diagnostic(cli.DiagnosticInvalidValue).On(subject)
		proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Env: cli.Env{{
				Name:  "TOOL_NUMBER",
				Value: "not-an-int",
			}},
		})
		proof.Require(t,
			clitest.InspectionRejected(),
			clitest.FailsWith(
				clitest.DiagnosticsExactly(diagnostic),
				diagnostic.ProducerKind(cli.ValueProducerKindEnvironment),
				diagnostic.Tier(cli.ResolutionTierEnvironment),
				diagnostic.Disposition(cli.ValueCandidateDispositionRejected),
			),
		)
	})

	t.Run("derived", func(t *testing.T) {
		t.Parallel()
		type config = diagnosticsDerivedConfig
		definition := cli.Definition[config]{
			Name: "tool",
			Help: cli.HelpConfig{
				Disabled: true,
			},
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Values.Derive(
					&root.Config.Derived,
					cli.Value.ComputedBy(func(context.Context, *config, cli.Dependencies) (string, error) {
						return "", errors.New("derived failed")
					}),
				)
			},
		}
		app := clitest.Compile(t, definition)
		diagnostic := clitest.Diagnostic(cli.DiagnosticValueComputationFailure)
		proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{})
		proof.Require(t,
			clitest.InspectionRejected(),
			clitest.FailsWith(
				clitest.DiagnosticsExactly(diagnostic),
				diagnostic.ProducerKind(cli.ValueProducerKindDerived),
				diagnostic.Disposition(cli.ValueCandidateDispositionRejected),
			),
		)
	})

	t.Run("resolved fact", func(t *testing.T) {
		t.Parallel()
		type config = diagnosticsFactConfig
		definition := cli.Definition[config]{
			Name: "tool",
			Help: cli.HelpConfig{
				Disabled: true,
			},
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Facts.Resolve(
					&root.Config.Fact,
					func(context.Context, *config, cli.Dependencies) (string, error) {
						return "", errors.New("fact failed")
					},
				)
			},
		}
		app := clitest.Compile(t, definition)
		diagnostic := clitest.Diagnostic(cli.DiagnosticFactResolutionFailure)
		proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{})
		proof.Require(t,
			clitest.InspectionRejected(),
			clitest.FailsWith(
				clitest.DiagnosticsExactly(diagnostic),
				diagnostic.ProducerKind(cli.ValueProducerKindResolvedFact),
				diagnostic.Disposition(cli.ValueCandidateDispositionRejected),
			),
		)
	})
}

type diagnosticNoticeFixtureConfig struct{}

func diagnosticNoticeDeliveryApp(t testing.TB, run func(), codes ...cli.AdvisoryCode) cli.Application {
	t.Helper()
	type config = diagnosticNoticeFixtureConfig
	definition := cli.Definition[config]{
		Name: "notices",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.Command(cli.Command[config]{
				Name: "child",
				Run: func(context.Context, *config) error {
					if run != nil {
						run()
					}
					return nil
				},
				Declare: func(child *cli.CommandBuilder[config]) {
					for _, code := range codes {
						child.Advisory.Always(cli.Advisory{
							Code:     code,
							Severity: cli.NoticeSeverityWarning,
							Message:  string(code),
						})
					}
				},
			})
		},
	}
	return clitest.Compile(t, definition)
}
