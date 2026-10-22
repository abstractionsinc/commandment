package conformance

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

type lifecycleNoticeOldConfig2 struct{ Old int }

type lifecycleNoticeItemsConfig struct{ Items []string }

type lifecycleNoticeOldConfig struct{ Old bool }

type lifecycleNoticeModeConfig struct{ Mode string }

type lifecycleNoticeLabelsConfig struct{ Labels map[string]string }

func TestLifecycleNoticeInspectionsPreservesLifecycleSemantics(t *testing.T) {
	t.Parallel()
	type config = lifecycleNoticeOldConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Release: cli.Release.SemVer("2.0.0"),
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.Bool(&root.Config.Old).Long("old").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2.0.0",
				Reason:       "retired",
			})
		},
	}
	app := clitest.Compile(t, definition)
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--old"},
	})
	proof.Require(t, clitest.Notice(cli.NoticeDeprecatedFlag).Present())
}

func TestLifecycleNoticeDeliveryAttemptsPreservesLifecycleSemantics(t *testing.T) {
	t.Parallel()
	type config = lifecycleNoticeOldConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.SemVer("2.0.0"),
		Run:     func(context.Context, *config) error { return nil },
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.Bool(&root.Config.Old).Long("old").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2.0.0",
			})
		},
	}
	app := clitest.Compile(t, definition)
	var got []cli.Notice
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"--old"},
		Notify: func(_ context.Context, n cli.Notice) error { got = append(got, n); return nil },
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Code != cli.NoticeDeprecatedFlag {
		t.Fatalf("delivered = %#v", got)
	}
}

func TestLifecycleNoticeCandidatesPreservesLifecycleSemantics(t *testing.T) {
	t.Parallel()
	type config = lifecycleNoticeOldConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Release: cli.Release.SemVer("2.0.0"),
		Declare: func(root *cli.CommandBuilder[config]) {
			flag := root.Flags.Bool(&root.Config.Old)
			old := flag.Long("old").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2.0.0",
			})
			old.ReplacedBy(flag.Long("new"))
		},
	}
	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	old, current := root.Flag("--old").Spelling("--old"), root.Flag("--new").Spelling("--new")
	notice := clitest.Notice(cli.NoticeDeprecatedFlag).On(old)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--old"},
	}).Require(t, clitest.InspectionReady(), clitest.NoticesExactly(notice), notice.ReplacedBy(current))
}

func TestAcceptedDeprecatedDomainEntriesPreservesLifecycleSemantics(t *testing.T) {
	t.Parallel()
	type config = lifecycleNoticeModeConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Release: cli.Release.SemVer("2.0.0"),
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := cli.Value.Domain(cli.Value.Assign[string](), func(b *cli.ValueDomainBuilder[string]) {
				b.Value("old", "old").Release(cli.ReleaseLifecycle{
					DeprecatedIn: "2.0.0",
				})
				b.Value("new", "new")
			})
			root.Flags.Value(&root.Config.Mode, domain).Long("mode")
		},
	}
	app := clitest.Compile(t, definition)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--mode", "old"},
	}).Require(t, clitest.Notice(cli.NoticeDeprecatedValue).Present())
}

func TestAcceptedDomainInputsPreservesLifecycleSemantics(t *testing.T) {
	t.Parallel()
	type config = lifecycleNoticeModeConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Release: cli.Release.SemVer("2.0.0"),
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := cli.Value.Domain(cli.Value.Assign[string](), func(b *cli.ValueDomainBuilder[string]) {
				b.Value("old", "old").Release(cli.ReleaseLifecycle{
					DeprecatedIn: "2.0.0",
				})
			})
			root.Flags.Value(&root.Config.Mode, domain).Long("mode")
		},
	}
	app := clitest.Compile(t, definition)
	entry := clitest.RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination).Entry("old")
	notice := clitest.Notice(cli.NoticeDeprecatedValue).On(entry)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--mode", "old"},
	}).Require(t, clitest.InspectionReady(), clitest.NoticesExactly(notice))
}

func TestLifecycleNoticeSuppressionsPreservesLifecycleSemantics(t *testing.T) {
	t.Parallel()
	type config = lifecycleNoticeModeConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Release: cli.Release.SemVer("2.0.0"),
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := cli.Value.Domain(cli.Value.Assign[string](), func(b *cli.ValueDomainBuilder[string]) {
				b.Value("old", "old").Release(cli.ReleaseLifecycle{
					DeprecatedIn: "2.0.0",
				})
			})
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Flags.ValueFor(mode).Long("mode")
			root.Notice.SuppressWhen(root.When.Resolved(mode), cli.NoticeMatch{
				Code: cli.NoticeDeprecatedValue,
			})
		},
	}
	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	entry := root.Flag("--mode").Value().Domain(cli.ValueDomainUseDestination).Entry("old")
	notice := clitest.Notice(cli.NoticeDeprecatedValue).On(entry)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--mode", "old"},
	}).Require(t,
		clitest.InspectionReady(),
		clitest.NoticesExactly(notice),
		notice.Suppression(cli.NoticeSuppressionSuppressed),
		notice.SuppressedByExactly(root.NoticeSuppressionAt(0)),
	)
}

func TestLifecycleNoticeMatchesPreservesLifecycleSemantics(t *testing.T) {
	t.Parallel()
	type config = lifecycleNoticeOldConfig
	match := cli.NoticeMatch{
		Severity: cli.NoticeSeverityWarning,
		Code:     cli.NoticeDeprecatedFlag,
	}
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Release: cli.Release.SemVer("2.0.0"),
		Declare: func(root *cli.CommandBuilder[config]) {
			old := root.Flags.Bool(&root.Config.Old)
			old.Long("old").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2.0.0",
			})
			root.Notice.SuppressWhen(root.When.Resolved(old), match)
		},
	}
	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	notice := clitest.Notice(cli.NoticeDeprecatedFlag).On(root.Flag("--old").Spelling("--old"))
	clitest.Schema(app).Require(t, root.NoticeSuppressionAt(0).Match(match))
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--old"},
	}).Require(t,
		clitest.InspectionReady(),
		clitest.NoticesExactly(notice),
		notice.Suppression(cli.NoticeSuppressionSuppressed),
		notice.SuppressedByExactly(root.NoticeSuppressionAt(0)),
	)
}

func TestLifecycleNoticeDeliveryAttempts(t *testing.T) {
	t.Parallel()
	type config = lifecycleNoticeOldConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Release: cli.Release.SemVer("2.0.0"),
		Run:     func(context.Context, *config) error { return nil },
		Declare: func(root *cli.CommandBuilder[config]) {
			old := root.Flags.Bool(&root.Config.Old)
			old.Long("old").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2.0.0",
			})
			root.Notice.SuppressWhen(root.When.Resolved(old), cli.NoticeMatch{
				Code: cli.NoticeDeprecatedFlag,
			})
		},
	}
	app := clitest.Compile(t, definition)
	var delivered []cli.Notice
	if err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"--old"},
		Notify: func(_ context.Context, notice cli.Notice) error {
			delivered = append(delivered, notice)
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if len(delivered) != 0 {
		t.Fatalf("suppressed lifecycle notices delivered = %#v", delivered)
	}
}

func TestLifecycleNoticeCandidates(t *testing.T) {
	t.Parallel()
	t.Run("duplicate flag occurrence", func(t *testing.T) {
		t.Parallel()
		type config = lifecycleNoticeOldConfig2
		definition := cli.Definition[config]{
			Name: "tool",
			Help: cli.HelpConfig{
				Disabled: true,
			},
			Release: cli.Release.SemVer("2.0.0"),
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Flags.Count(&root.Config.Old).Occurrences(cli.Count.ZeroOrMore()).
					Long("old").
					Release(cli.ReleaseLifecycle{
						DeprecatedIn: "2.0.0",
					})
			},
		}
		app := clitest.Compile(t, definition)
		old := clitest.RootCommand().Flag("--old").Spelling("--old")
		notice := clitest.Notice(cli.NoticeDeprecatedFlag).On(old)
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"--old", "--old"},
		}).Require(t, clitest.InspectionReady(), clitest.NoticesExactly(notice))
	})

	t.Run("duplicate domain entry", func(t *testing.T) {
		t.Parallel()
		type config = lifecycleNoticeLabelsConfig
		domain := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
			values.Value("old", "old").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2.0.0",
			})
		})
		definition := cli.Definition[config]{
			Name: "tool",
			Help: cli.HelpConfig{
				Disabled: true,
			},
			Release: cli.Release.SemVer("2.0.0"),
			Declare: func(root *cli.CommandBuilder[config]) {
				labels := root.Values.Keyed(&root.Config.Labels, cli.Value.Keyed(domain, domain))
				root.Flags.Keyed(labels).Long("label")
			},
		}
		app := clitest.Compile(t, definition)
		entry := clitest.RootCommand().Flag("--label").Value().Domain(cli.ValueDomainUseMapKey).Entry("old")
		notice := clitest.Notice(cli.NoticeDeprecatedValue).On(entry)
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"--label", "old=old"},
		}).Require(t, clitest.InspectionReady(), clitest.NoticesExactly(notice))
	})

	t.Run("sensitive value", func(t *testing.T) {
		t.Parallel()
		type config = lifecycleNoticeModeConfig
		domain := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
			values.Value("old", "old").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2.0.0",
			})
		})
		definition := cli.Definition[config]{
			Name: "tool",
			Help: cli.HelpConfig{
				Disabled: true,
			},
			Release: cli.Release.SemVer("2.0.0"),
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Flags.Value(&root.Config.Mode, domain).Long("mode").Sensitive()
			},
		}
		app := clitest.Compile(t, definition)
		entry := clitest.RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination).Entry("old")
		notice := clitest.Notice(cli.NoticeDeprecatedValue).On(entry)
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"--mode", "old"},
		}).Require(t,
			clitest.InspectionReady(), clitest.NoticesExactly(notice), notice.Sensitive(), notice.SubjectRedacted(),
		)
	})
}

func TestAcceptedDeprecatedDomainEntries(t *testing.T) {
	t.Parallel()
	type config = lifecycleNoticeLabelsConfig
	domain := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
		values.Value("old", "old").Release(cli.ReleaseLifecycle{
			DeprecatedIn: "2.0.0",
		})
	})
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Release: cli.Release.SemVer("2.0.0"),
		Declare: func(root *cli.CommandBuilder[config]) {
			labels := root.Values.Keyed(&root.Config.Labels, cli.Value.Keyed(domain, domain))
			root.Flags.Keyed(labels).Long("label")
		},
	}
	app := clitest.Compile(t, definition)
	entry := clitest.RootCommand().Flag("--label").Value().Domain(cli.ValueDomainUseMapKey).Entry("old")
	notice := clitest.Notice(cli.NoticeDeprecatedValue).On(entry)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--label", "old=old"},
	}).Require(t, clitest.InspectionReady(), clitest.NoticesExactly(notice))
}

func TestAcceptedDomainInputs(t *testing.T) {
	t.Parallel()
	t.Run("sequence element", func(t *testing.T) {
		t.Parallel()
		type config = lifecycleNoticeItemsConfig
		domain := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
			values.Value("old", "old").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2.0.0",
			})
		})
		definition := cli.Definition[config]{
			Name: "tool",
			Help: cli.HelpConfig{
				Disabled: true,
			},
			Release: cli.Release.SemVer("2.0.0"),
			Declare: func(root *cli.CommandBuilder[config]) {
				items := root.Values.Sequence(&root.Config.Items, domain)
				root.Flags.Sequence(items).Long("item")
			},
		}
		app := clitest.Compile(t, definition)
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"--item", "old"},
		}).Require(t, clitest.Notice(cli.NoticeDeprecatedValue).Present())
	})

	t.Run("key", func(t *testing.T) {
		t.Parallel()
		type config = lifecycleNoticeLabelsConfig
		domain := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
			values.Value("old", "old").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2.0.0",
			})
		})
		definition := cli.Definition[config]{
			Name: "tool",
			Help: cli.HelpConfig{
				Disabled: true,
			},
			Release: cli.Release.SemVer("2.0.0"),
			Declare: func(root *cli.CommandBuilder[config]) {
				labels := root.Values.Keyed(&root.Config.Labels, cli.Value.Keyed(domain, cli.Value.String()))
				root.Flags.Keyed(labels).Long("label")
			},
		}
		app := clitest.Compile(t, definition)
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"--label", "old=value"},
		}).Require(t, clitest.Notice(cli.NoticeDeprecatedValue).Present())
	})

	t.Run("value", func(t *testing.T) {
		t.Parallel()
		type config = lifecycleNoticeLabelsConfig
		domain := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
			values.Value("old", "old").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2.0.0",
			})
		})
		definition := cli.Definition[config]{
			Name: "tool",
			Help: cli.HelpConfig{
				Disabled: true,
			},
			Release: cli.Release.SemVer("2.0.0"),
			Declare: func(root *cli.CommandBuilder[config]) {
				labels := root.Values.Keyed(&root.Config.Labels, cli.Value.Keyed(cli.Value.String(), domain))
				root.Flags.Keyed(labels).Long("label")
			},
		}
		app := clitest.Compile(t, definition)
		clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"--label", "key=old"},
		}).Require(t, clitest.Notice(cli.NoticeDeprecatedValue).Present())
	})
}

func TestLifecycleNoticeMatches(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		match      cli.NoticeMatch
		suppressed bool
	}{
		{
			name: "all",
			match: cli.NoticeMatch{
				All: true,
			},
			suppressed: true,
		},
		{
			name: "severity mismatch",
			match: cli.NoticeMatch{
				Severity: cli.NoticeSeverityInformation,
			},
		},
		{
			name: "code mismatch",
			match: cli.NoticeMatch{
				Code: cli.NoticeDeprecatedValue,
			},
		},
		{
			name: "advisory mismatch",
			match: cli.NoticeMatch{
				AdvisoryCode: "other",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			type config = lifecycleNoticeOldConfig
			definition := cli.Definition[config]{
				Name: "tool",
				Help: cli.HelpConfig{
					Disabled: true,
				},
				Release: cli.Release.SemVer("2.0.0"),
				Declare: func(root *cli.CommandBuilder[config]) {
					old := root.Flags.Bool(&root.Config.Old)
					old.Long("old").Release(cli.ReleaseLifecycle{
						DeprecatedIn: "2.0.0",
					})
					root.Notice.SuppressWhen(root.When.Resolved(old), tc.match)
				},
			}
			app := clitest.Compile(t, definition)
			root := clitest.RootCommand()
			notice := clitest.Notice(cli.NoticeDeprecatedFlag).On(root.Flag("--old").Spelling("--old"))
			want := cli.NoticeSuppressionNotSuppressed
			if tc.suppressed {
				want = cli.NoticeSuppressionSuppressed
			}
			clitest.Schema(app).Require(t, root.NoticeSuppressionAt(0).Match(tc.match))
			clitest.Inspect(t.Context(), app, cli.InspectionRequest{
				Args: []string{"--old"},
			}).Require(t, clitest.InspectionReady(), clitest.NoticesExactly(notice), notice.Suppression(want))
		})
	}
}

type noticeDeliveryConfig struct{}

func declareNoticeDeliveryFixture(root *cli.CommandBuilder[noticeDeliveryConfig]) {
	root.Advisory.Always(cli.Advisory{
		Code:     "warning",
		Severity: cli.NoticeSeverityWarning,
		Message:  "warning",
	})
}

func TestDeliverNoticesPreservesNoticeSemantics(t *testing.T) {
	t.Parallel()
	type config = noticeDeliveryConfig
	cause := errors.New("delivery failed")
	definition := cli.Definition[config]{
		Name:    "tool",
		Declare: declareNoticeDeliveryFixture,
		Run:     func(context.Context, *config) error { return nil },
	}
	app := clitest.Compile(t, definition)
	calls := 0
	err := app.Run(t.Context(), cli.RunRequest{
		Notify: func(_ context.Context, n cli.Notice) error {
			calls++
			if n.AdvisoryCode != "warning" {
				t.Fatalf("notice=%#v", n)
			}
			return cause
		},
	})
	if calls != 1 || !errors.Is(err, cause) {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestDiagnosticNoticePhasePreservesNoticeSemantics(t *testing.T) {
	t.Parallel()
	type config = noticeDeliveryConfig
	cause := errors.New("delivery failed")
	definition := cli.Definition[config]{
		Name:    "tool",
		Declare: declareNoticeDeliveryFixture,
		Run:     func(context.Context, *config) error { return nil },
	}
	app := clitest.Compile(t, definition)
	err := app.Run(t.Context(), cli.RunRequest{
		Notify: func(context.Context, cli.Notice) error { return cause },
	})
	if !errors.Is(err, cause) {
		t.Fatalf("delivery cause not preserved: %v", err)
	}
	diagnostic := clitest.Diagnostic(cli.DiagnosticNoticeDeliveryFailure)
	clitest.Diagnostics(err).
		Require(t, diagnostic.Present(), diagnostic.NoticeDelivery().Phase(cli.NoticePhaseAnalysis))
}

func TestValidNoticePhasePreservesNoticeSemantics(t *testing.T) {
	t.Parallel()
	type config = noticeDeliveryConfig
	definition := cli.Definition[config]{
		Name:    "tool",
		Declare: declareNoticeDeliveryFixture,
		Run:     func(context.Context, *config) error { return nil },
	}
	app := clitest.Compile(t, definition)
	if err := app.Run(t.Context(), cli.RunRequest{
		Notify: func(context.Context, cli.Notice) error { return nil },
	}); err != nil {
		t.Fatal(err)
	}
}

type noticeNoticeFixtureConfig struct{}

func noticeNoticeDeliveryApp(t testing.TB, run func(), codes ...cli.AdvisoryCode) cli.Application {
	t.Helper()
	type config = noticeNoticeFixtureConfig
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

type noticeNameLimitConfig struct {
	Name  string
	Limit int
}

type noticeContextKey struct{}

type noticeReferenceModeConfig struct {
	Reference bool
	Mode      string
}

type noticeConfig struct{}

func TestDefinitionAuthoringLowersRulesAdvisoriesAndNoticeSuppression(t *testing.T) {
	t.Parallel()
	type config = noticeNameLimitConfig
	ruleCalled, advisoryCalled := false, false
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "commandment",
		Declare: func(root *cli.CommandBuilder[config]) {
			name := root.Values.String(&root.Config.Name)
			limit := root.Values.Int(&root.Config.Limit)
			root.Rule.Require(name)
			root.Rule.When(root.Rule.ContributedBy(name, cli.ValueProducerKindFlag)).Between(limit, 1, 10)
			root.Rule.Check("custom-rule", func(context.Context, *config, cli.Dependencies) (bool, error) { ruleCalled = true; return true, nil }).
				Subject(name).
				DependsOn(limit.OptionalDependency())
			root.Advisory.Always(cli.Advisory{
				Code:     "always",
				Severity: cli.NoticeSeverityInformation,
				Message:  "always message",
			}).Subject(name)
			root.Advisory.Detect(func(context.Context, *config, cli.Dependencies) (bool, error) {
				advisoryCalled = true
				return true, nil
			}, cli.Advisory{
				Code:     "dynamic",
				Severity: cli.NoticeSeverityWarning,
				Message:  "dynamic message",
			}).DependsOn(limit)
			root.Notice.SuppressWhen(root.When.Resolved(name), cli.NoticeMatch{
				AdvisoryCode: "dynamic",
			})
		},
	})
	if ruleCalled || advisoryCalled {
		t.Fatal("analysis callbacks executed during Compile")
	}
	root := clitest.RootCommand()
	clitest.Schema(app).Require(t,
		root.RulesExactly(root.RuleAt(0), root.RuleAt(1), root.RuleAt(2)),
		root.AdvisoriesExactly(root.AdvisoryAt(0), root.AdvisoryAt(1)),
		root.NoticeSuppressionsExactly(root.NoticeSuppressionAt(0)),
	)
}

func TestDeliverNoticesWithoutHandlerMakesNoDeliveryAttempt(t *testing.T) {
	t.Parallel()
	ran := false
	app := noticeNoticeDeliveryApp(t, func() { ran = true }, "one")
	inspection, err := app.Inspect(t.Context(), cli.InspectionRequest{
		Args: []string{"child"},
	})
	if err != nil || len(inspection.Notices) != 1 || inspection.Notices[0].AdvisoryCode != "one" {
		t.Fatalf("Inspect before no-handler Run: err=%v notices=%#v", err, inspection.Notices)
	}
	if err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"child"},
	}); err != nil {
		t.Fatalf("Run without Notify: %v", err)
	}
	if !ran {
		t.Fatal("Run without Notify did not execute application after Notice construction")
	}
}

func TestDeliverNoticesPreservesStructuredFailuresAndContinues(t *testing.T) {
	t.Parallel()
	ran := false
	app := noticeNoticeDeliveryApp(t, func() { ran = true }, "first", "second", "third")
	firstErr := errors.New("first Notice delivery failed")
	thirdErr := errors.New("third Notice delivery failed")
	var calls []cli.AdvisoryCode
	var originalPaths [][]string
	err := app.Run(
		t.Context(),
		cli.RunRequest{
			Args: []string{"child"},
			Notify: func(_ context.Context, notice cli.Notice) error {
				calls = append(calls, notice.AdvisoryCode)
				originalPaths = append(originalPaths, append([]string(nil), notice.CommandPath...))
				if len(notice.CommandPath) == 0 {
					t.Fatalf("nested delivered Notice has empty CommandPath: %#v", notice)
				}
				// The handler owns only its delivered snapshot. Mutation must not alter
				// retained delivery-failure metadata.
				notice.CommandPath[0] = "host-mutated"
				switch notice.AdvisoryCode {
				case "first":
					return firstErr
				case "third":
					return thirdErr
				default:
					return nil
				}
			},
		},
	)
	if !slices.Equal(calls, []cli.AdvisoryCode{"first", "second", "third"}) {
		t.Fatalf("delivery order = %#v", calls)
	}
	if !ran {
		t.Fatal("Notice delivery failures stopped the application execution pipeline")
	}
	if err == nil || !errors.Is(err, firstErr) || !errors.Is(err, thirdErr) {
		t.Fatalf("Run error = %v, want both delivery causes", err)
	}
	report := cli.Diagnostics(err)
	var deliveries []cli.NoticeDeliveryFailure
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Code == cli.DiagnosticNoticeDeliveryFailure && diagnostic.NoticeDelivery != nil {
			deliveries = append(deliveries, *diagnostic.NoticeDelivery)
		}
	}
	if len(deliveries) != 2 {
		t.Fatalf("notice delivery diagnostics = %#v", report.Diagnostics)
	}
	wantIndexes := []int{0, 2}
	wantPaths := [][]string{originalPaths[0], originalPaths[2]}
	for i, delivery := range deliveries {
		if delivery.Index != wantIndexes[i] || delivery.Phase != cli.NoticePhaseAnalysis ||
			delivery.Code != cli.NoticeAdvisory ||
			!slices.Equal(delivery.CommandPath, wantPaths[i]) {
			t.Fatalf(
				"delivery[%d] = %#v, want index %d analysis advisory path %#v",
				i,
				delivery,
				wantIndexes[i],
				wantPaths[i],
			)
		}
	}
}

func TestDeliverNoticesPreservesCallerContextAndSynchronousAttemptOrder(t *testing.T) {
	t.Parallel()
	app := noticeNoticeDeliveryApp(t, nil, "first", "second")
	type contextKey = noticeContextKey
	ctx := context.WithValue(t.Context(), contextKey{}, "value")
	var order []cli.AdvisoryCode
	err := app.Run(ctx, cli.RunRequest{
		Args: []string{"child"},
		Notify: func(callCtx context.Context, notice cli.Notice) error {
			if got := callCtx.Value(contextKey{}); got != "value" {
				t.Fatalf("context value = %#v", got)
			}
			order = append(order, notice.AdvisoryCode)
			return nil
		},
	})
	if err != nil || !slices.Equal(order, []cli.AdvisoryCode{"first", "second"}) {
		t.Fatalf("Run err=%v order=%#v", err, order)
	}
}

func TestLifecycleNoticesAttributeDeprecatedSpellingAndDomainEntryExactly(t *testing.T) {
	t.Parallel()
	type config = noticeReferenceModeConfig
	definition := cli.Definition[config]{
		Name: "lifecycle",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Release: cli.Release.SemVer("2.0.0"),
		Run:     func(context.Context, *config) error { return nil },
		Declare: func(root *cli.CommandBuilder[config]) {
			flag := root.Flags.Bool(&root.Config.Reference)
			old := flag.Long("old").Release(cli.ReleaseLifecycle{
				DeprecatedIn: "2.0.0",
				Reason:       "use --new",
			})
			newer := flag.Long("new")
			old.ReplacedBy(newer)

			domain := cli.Value.Domain(cli.Value.Assign[string](), func(builder *cli.ValueDomainBuilder[string]) {
				reference := builder.Value("reference", "reference").Description("reference mode").
					Release(cli.ReleaseLifecycle{
						DeprecatedIn: "2.0.0",
						Reason:       "use modern",
					})
				modern := builder.Value("modern", "modern").Description("modern mode")
				reference.ReplacedBy(modern)
			})
			root.Flags.Value(&root.Config.Mode, domain).Long("mode")
		},
	}
	app := clitest.Compile(t, definition)

	oldSpelling := clitest.RootCommand().Flag("--old").Spelling("--old")
	newSpelling := clitest.RootCommand().Flag("--new").Spelling("--new")
	referenceEntry := clitest.RootCommand().
		Flag("--mode").
		Value().
		Domain(cli.ValueDomainUseDestination).
		Entry("reference")
	modernEntry := clitest.RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination).Entry("modern")
	deprecatedFlag := clitest.Notice(cli.NoticeDeprecatedFlag).On(oldSpelling)
	deprecatedValue := clitest.Notice(cli.NoticeDeprecatedValue).On(referenceEntry)
	request := cli.InspectionRequest{
		Args: []string{"--old", "--mode", "reference"},
	}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.NoticesExactly(deprecatedFlag, deprecatedValue),
		deprecatedFlag.ReplacedBy(newSpelling),
		deprecatedFlag.Reason("use --new"),
		deprecatedValue.ReplacedBy(modernEntry),
		deprecatedValue.Reason("use modern"),
	)

	// Runtime delivery remains an ordinary-Go application-effect assertion. IDs
	// are discovered only for that callback observation, not for Inspection proof.
	schema := app.Schema()
	var oldID cli.FlagSpellingID
	for id, spelling := range schema.FlagSpellings {
		if spelling.Spelling.Text == "--old" {
			oldID = id
		}
	}
	var referenceID cli.ValueDomainEntryID
	for id, entry := range schema.ValueDomainEntries {
		if entry.Name == "reference" {
			referenceID = id
		}
	}
	if oldID == 0 || referenceID == 0 {
		t.Fatalf("runtime delivery identities old=%v reference=%v", oldID, referenceID)
	}

	var delivered []cli.Notice
	if err := app.Run(t.Context(), cli.RunRequest{
		Args: []string{"--old", "--mode", "reference"},
		Notify: func(_ context.Context, notice cli.Notice) error {
			delivered = append(delivered, notice)
			return nil
		},
	}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(delivered) != 2 || delivered[0].Declaration.FlagSpelling != oldID ||
		delivered[1].Declaration.ValueDomainEntry != referenceID {
		t.Fatalf("delivered lifecycle notices = %#v", delivered)
	}
}

func TestRunFailureSettlementPreservesPrimaryNoticeAndIndependentCallerCancellation(t *testing.T) {
	t.Parallel()
	type config = noticeConfig
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
		Run: func(context.Context, *config) error {
			cancel(callerCause)
			return primary
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	got := app.Run(ctx, cli.RunRequest{
		Notify: func(context.Context, cli.Notice) error { return noticeCause },
	})
	if got == nil {
		t.Fatal("Run() error = nil")
	}
	if got.Error() != primary.Error() {
		t.Fatalf("highest-precedence error string = %q, want %q", got.Error(), primary.Error())
	}
	for _, branch := range []error{primary, noticeCause, callerCause} {
		if !errors.Is(got, branch) {
			t.Fatalf("Run() error lost branch %v: %v", branch, got)
		}
	}
	if report := cli.Diagnostics(
		got,
	); len(report.Diagnostics) != 1 ||
		report.Diagnostics[0].Code != cli.DiagnosticNoticeDeliveryFailure {
		t.Fatalf("Diagnostics(Run error) = %#v", report)
	}
}
