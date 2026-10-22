package conformance

import (
	"context"
	"errors"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

type completionShortcutRootConfig struct{}

type completionShortcutModeNameChildConfig struct {
	Mode string
	Name string
}

type completionShortcutFirstSecondChildConfig struct {
	First  string
	Second string
}

type completionShortcutModeChildConfig struct{ Mode string }

type completionShortcutGateRootConfig struct{ Gate string }

type completionShortcutVerboseChildConfig struct{ Verbose bool }

type completionShortcutProfileChildConfig struct{ Profile string }

type completionShortcutDynamicConfig struct{}

type completionShortcutParentConfig struct{}

type completionShortcutLeafConfig struct{}

type completionShortcutAlphaConfig struct{}

type completionShortcutBetaConfig struct{}

type completionShortcutOtherConfig struct{}

type completionShortcutNameChildConfig struct{ Name string }

type completionShortcutModeAlphaConfig struct{ Mode string }

type completionShortcutFileAlphaConfig struct{ File string }

type completionShortcutFileBetaConfig struct{ File string }

type completionShortcutPathParentConfig struct{ Path string }

type completionShortcutChildConfig struct{}

type completionShortcutValuesChildConfig struct{ Values []string }

func completionCommandByPath(t testing.TB, schema cli.Schema, path ...string) clitest.CommandLocator {
	t.Helper()
	if len(path) == 0 {
		return clitest.CommandByID(schema.Root)
	}
	for _, id := range schema.CommandOrder {
		if slices.Equal(schema.Commands[id].Path, path) {
			return clitest.CommandByID(id)
		}
	}
	t.Fatalf("command path %v is absent from schema", path)
	return clitest.CommandByID(0)
}

func TestCompleteShortcutPrependedArgsPreserveIdentityAndDriveResidualValueCompletion(t *testing.T) {
	t.Parallel()
	type rootConfig = completionShortcutRootConfig
	type childConfig = completionShortcutModeNameChildConfig
	var observedMode string
	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "serve",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					mode := child.Flags.String(&child.Config.Mode)
					mode.Long("mode")
					domain := child.Values.Domain(
						func(_ context.Context, cfg *childConfig, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
							observedMode = cfg.Mode
							builder.Value("alpha", "alpha")
							return nil
						},
						mode,
					)
					name := child.Values.Value(&child.Config.Name, domain)
					child.Arguments.Form(func(form *cli.ArgumentFormBuilder[childConfig]) {
						form.ValueFor(name, "name")
					})
					child.Shortcut(cli.CommandShortcut{
						Name: "quick",
						Args: []string{"--mode", "fast"},
					})
				},
			})
		},
	})
	schemaResult := clitest.Schema(app)
	schema := schemaResult.Schema()
	root := completionCommandByPath(t, schema)
	serve := completionCommandByPath(t, schema, "serve")
	shortcut := clitest.ResolveDeclaration(t, schemaResult, root.Shortcut("quick"))
	completionRequest := cli.CompletionRequest{
		Args: []string{""},
		Cursor: cli.CompletionCursor{
			Argument: 0,
		},
		Client: cli.CompletionClient{
			Capabilities: cli.CompletionClientCapabilities{
				CandidateKinds: true,
			},
		},
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t,
		clitest.CompletionItem("quick").Kind(cli.CompletionCandidateCommandShortcut),
		clitest.CompletionItem("quick").Declaration(shortcut),
	)

	shortcutSource := clitest.InputSource(cli.InputKindShortcut)
	completionRequestUnderTest := cli.CompletionRequest{
		Args: []string{"quick", "al"},
		Cursor: cli.CompletionCursor{
			Argument: 1,
			Byte:     2,
		},
		Client: cli.CompletionClient{
			Match: cli.CompletionMatchOptions{
				Mode: cli.CompletionMatchModePrefix,
				Case: cli.CompletionCaseSensitive,
			},
			Capabilities: cli.CompletionClientCapabilities{
				CandidateKinds: true,
			},
		},
	}
	shortcutProof := clitest.Complete(t.Context(), app, completionRequestUnderTest)
	shortcutProof.Require(t,
		shortcut.Selected(),
		clitest.SelectionPathExactly(root, serve),
		clitest.CompletionItem("alpha").Present(),
		shortcutSource.CommandShortcut(shortcut),
		clitest.CompletionObservedInputSourcesExactly(cli.InputKindShortcut, shortcutSource),
	)
	if observedMode != "fast" {
		t.Fatalf("shortcut fixed Args resolved mode = %q, want fast", observedMode)
	}

	completionRequestUnderTest2 := cli.CompletionRequest{
		Args: []string{"serve", "--mode", "fast", "al"},
		Cursor: cli.CompletionCursor{
			Argument: 3,
			Byte:     2,
		},
		Client: cli.CompletionClient{
			Match: cli.CompletionMatchOptions{
				Mode: cli.CompletionMatchModePrefix,
				Case: cli.CompletionCaseSensitive,
			},
			Capabilities: cli.CompletionClientCapabilities{
				CandidateKinds: true,
			},
		},
	}
	clitest.Complete(t.Context(), app, completionRequestUnderTest2).Require(t,
		clitest.SelectionPathExactly(root, serve),
		clitest.CompletionItem("alpha").Present(),
	)
}

func TestCompleteShortcutAppendArgsAndDefaultsEnterCanonicalResidualAnalysis(t *testing.T) {
	t.Parallel()
	type rootConfig = completionShortcutRootConfig
	type childConfig = completionShortcutModeNameChildConfig
	compile := func(shortcut cli.CommandShortcut, configure func(cli.CommandShortcutDeclaration[childConfig], cli.Destination[childConfig, string])) cli.Application {
		definition := cli.Definition[rootConfig]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[rootConfig]) {
				root.Subcommands.Command(cli.Command[childConfig]{
					Name: "serve",
					Declare: func(child *cli.CommandBuilder[childConfig]) {
						mode := child.Values.String(&child.Config.Mode)
						child.Flags.ValueFor(mode).Long("mode")
						child.Arguments.Form(func(form *cli.ArgumentFormBuilder[childConfig]) {
							form.String(
								&form.Config.Name,
								"name",
							).CompleteBy(func(_ context.Context, cfg *childConfig, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
								contribution := cli.CompletionContribution{
									Candidates: []cli.CompletionCandidate{
										{
											Value: cfg.Mode,
											Kind:  cli.CompletionCandidateValue,
										},
									},
								}
								return contribution, nil
							}).
								From(mode)
						})
						decl := child.Shortcut(shortcut)
						if configure != nil {
							configure(decl, mode)
						}
					},
				})
			},
		}
		return clitest.Compile(t, definition)
	}

	t.Run("append args", func(t *testing.T) {
		t.Parallel()
		app := compile(cli.CommandShortcut{
			Name:       "quick",
			AppendArgs: []string{"--mode", "tail"},
		}, nil)
		completionRequest := cli.CompletionRequest{
			Args: []string{"quick", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					CandidateKinds: true,
				},
			},
		}
		clitest.Complete(t.Context(), app, completionRequest).Require(t,
			clitest.CompletionItem("tail").Present(),
		)
	})

	t.Run("shortcut default fallback and explicit override", func(t *testing.T) {
		t.Parallel()
		app := compile(cli.CommandShortcut{
			Name: "safe",
		}, func(shortcut cli.CommandShortcutDeclaration[childConfig], mode cli.Destination[childConfig, string]) {
			shortcut.Default(mode, "safe")
		})
		completionRequest := cli.CompletionRequest{
			Args: []string{"safe", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					CandidateKinds: true,
				},
			},
		}
		clitest.Complete(t.Context(), app, completionRequest).Require(t,
			clitest.CompletionItem("safe").Present(),
		)
		completionRequestUnderTest := cli.CompletionRequest{
			Args: []string{"safe", "--mode", "fast", ""},
			Cursor: cli.CompletionCursor{
				Argument: 3,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					CandidateKinds: true,
				},
			},
		}
		clitest.Complete(t.Context(), app, completionRequestUnderTest).Require(t,
			clitest.CompletionItem("fast").Present(),
		)
	})

	t.Run("provider textual shortcut default", func(t *testing.T) {
		t.Parallel()
		providerCalls := 0
		app := clitest.Compile(t, cli.Definition[rootConfig]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[rootConfig]) {
				root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[rootConfig]) error {
					providerCalls++
					var mode cli.Destination[childConfig, string]
					child := batch.Command(cli.Command[childConfig]{
						Name: "remote",
						Declare: func(command *cli.CommandBuilder[childConfig]) {
							mode = command.Values.String(&command.Config.Mode)
							command.Flags.ValueFor(mode).Long("mode")
							command.Arguments.Form(func(form *cli.ArgumentFormBuilder[childConfig]) {
								form.String(
									&form.Config.Name,
									"name",
								).CompleteBy(func(_ context.Context, cfg *childConfig, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
									contribution := cli.CompletionContribution{
										Candidates: []cli.CompletionCandidate{
											{
												Value: cfg.Mode,
												Kind:  cli.CompletionCandidateValue,
											},
										},
									}
									return contribution, nil
								}).
									From(mode)
							})
						},
					})
					batch.Shortcut(child, cli.CommandShortcut{
						Name: "quick",
					}).DefaultFlag("--mode", []string{"provider"})
					return nil
				})
			},
		})
		completionRequest := cli.CompletionRequest{
			Args: []string{"quick", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					CandidateKinds: true,
				},
			},
		}
		clitest.Complete(t.Context(), app, completionRequest).Require(t,
			clitest.CompletionItem("provider").Present(),
		)
		if providerCalls != 1 {
			t.Fatalf("provider calls = %d, want 1", providerCalls)
		}
	})
}

func TestCompleteShortcutAppendArgsResponseReferenceExpandsAfterResidualCursor(t *testing.T) {
	t.Parallel()
	type rootConfig = completionShortcutRootConfig
	type childConfig = completionShortcutModeNameChildConfig
	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "serve",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					child.Dialect.ResponseFiles(true)
					child.Dialect.FlagOrder(cli.FlagOrderInterspersed)
					mode := child.Values.String(&child.Config.Mode)
					child.Flags.ValueFor(mode).Long("mode")
					child.Arguments.Form(func(form *cli.ArgumentFormBuilder[childConfig]) {
						form.String(
							&form.Config.Name,
							"name",
						).CompleteBy(func(_ context.Context, cfg *childConfig, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
							contribution := cli.CompletionContribution{
								Candidates: []cli.CompletionCandidate{
									{
										Value: cfg.Mode,
										Kind:  cli.CompletionCandidateValue,
									},
								},
							}
							return contribution, nil
						}).
							From(mode)
					})
					child.Shortcut(cli.CommandShortcut{
						Name:       "quick",
						AppendArgs: []string{"@cfg"},
					})
				},
			})
		},
	})

	t.Run("expands exactly once and affects residual value completion", func(t *testing.T) {
		t.Parallel()
		loads := 0
		completionRequest := cli.CompletionRequest{
			Args: []string{"quick", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					CandidateKinds: true,
				},
			},
			ResponseFiles: cli.ResponseFileAccess{
				Load: func(_ context.Context, request cli.ResponseFileRequest) (cli.ResponseFile, error) {
					loads++
					if request.Path != "cfg" {
						return cli.ResponseFile{}, errors.New("unexpected response path")
					}
					file := cli.ResponseFile{
						Key:      "cfg",
						Revision: cli.Revision("r1"),
						Content:  []byte("--mode tail"),
					}
					return file, nil
				},
			},
		}
		clitest.Complete(t.Context(), app, completionRequest).Require(t,
			clitest.CompletionItem("tail").Present(),
			clitest.CompletionNotPartial(),
		)
		if loads != 1 {
			t.Fatalf("AppendArgs response loads = %d, want 1", loads)
		}
	})

	t.Run("load failure preserves partial response failure", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("append response load failed")
		loads := 0
		completionRequest := cli.CompletionRequest{
			Args: []string{"quick", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					CandidateKinds: true,
				},
			},
			ResponseFiles: cli.ResponseFileAccess{
				Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
					loads++
					return cli.ResponseFile{}, cause
				},
			},
		}
		proof := clitest.Complete(t.Context(), app, completionRequest)
		proof.Require(t,
			clitest.CompletionPartial(),
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticResponseFileLoadFailure).Present()),
		)
		_, err := proof.Completion()
		if err == nil || !errors.Is(err, cause) {
			t.Fatalf("Complete() error = %v, want AppendArgs response load cause", err)
		}
		if loads != 1 {
			t.Fatalf("AppendArgs response loads = %d, want 1", loads)
		}
	})
}

func TestCompleteShortcutAppendArgsRemainAfterResidualCursorForPositionalOwnership(t *testing.T) {
	t.Parallel()
	type rootConfig = completionShortcutRootConfig
	type childConfig = completionShortcutFirstSecondChildConfig
	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "serve",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					child.Arguments.Form(func(form *cli.ArgumentFormBuilder[childConfig]) {
						form.String(
							&form.Config.First,
							"first",
						).CompleteBy(
							func(_ context.Context, cfg *childConfig, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
								contribution := cli.CompletionContribution{
									Candidates: []cli.CompletionCandidate{
										{
											Value: "first:" + cfg.Second,
											Kind:  cli.CompletionCandidateValue,
										},
									},
								}
								return contribution, nil
							},
						)
						form.String(
							&form.Config.Second,
							"second",
						).CompleteBy(
							func(context.Context, *childConfig, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
								contribution := cli.CompletionContribution{
									Candidates: []cli.CompletionCandidate{
										{
											Value: "second-owner",
											Kind:  cli.CompletionCandidateValue,
										},
									},
								}
								return contribution, nil
							},
						)
					})
					child.Shortcut(cli.CommandShortcut{
						Name:       "quick",
						AppendArgs: []string{"fixed-tail"},
					})
				},
			})
		},
	})

	completionRequest := cli.CompletionRequest{
		Args: []string{"quick", ""},
		Cursor: cli.CompletionCursor{
			Argument: 1,
		},
		Client: cli.CompletionClient{
			Capabilities: cli.CompletionClientCapabilities{
				CandidateKinds: true,
			},
		},
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t,
		clitest.CompletionTargetsExactly(clitest.CompletionTargetAt(0)),
		clitest.CompletionTargetAt(0).Kind(cli.CompletionTargetArgument),
		clitest.CompletionTargetAt(0).Name("first"),
		clitest.CompletionTargetAt(0).Occurrence(0),
		clitest.CompletionItemsExactly(clitest.CompletionItem("first:fixed-tail")),
	)
}

func TestCompleteShortcutAppendArgsConstrainStaticFlagOccurrenceCandidates(t *testing.T) {
	t.Parallel()
	type rootConfig = completionShortcutRootConfig
	type childConfig = completionShortcutModeChildConfig
	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "serve",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					mode := child.Values.String(&child.Config.Mode)
					child.Flags.ValueFor(mode).Occurrences(cli.Count.Optional()).Long("mode")
					child.Shortcut(cli.CommandShortcut{
						Name:       "quick",
						AppendArgs: []string{"--mode", "tail"},
					})
				},
			})
		},
	})
	clitest.Complete(t.Context(), app, cli.CompletionRequest{
		Args: []string{"quick", "--mode"},
		Cursor: cli.CompletionCursor{
			Argument: 1,
			Byte:     len("--mode"),
		},
	}).Require(t,
		clitest.CompletionNotPartial(),
		clitest.CompletionItem("--mode").Absent(),
	)
}

func TestCompleteShortcutFixedTerminalTailFiltersEarlierActionFlagCandidate(t *testing.T) {
	t.Parallel()
	type rootConfig = completionShortcutGateRootConfig
	type childConfig = completionShortcutVerboseChildConfig

	for _, test := range []struct {
		name           string
		forceReference bool
	}{
		{
			name: "static projection",
		},
		{
			name:           "detached reference",
			forceReference: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			definition := cli.Definition[rootConfig]{
				Name: "tool",
				Help: cli.HelpConfig{
					ReplaceEntries: func(entries *cli.HelpActionEntries) {
						entries.Long("help").Placement(cli.FlagPlacementSubtree)
					},
				},
				Declare: func(root *cli.CommandBuilder[rootConfig]) {
					if test.forceReference {
						gate := root.Values.String(&root.Config.Gate)
						root.Values.Default(gate, "yes")
						root.VisibleWhen(root.When.Equal(gate, "yes"))
					}
					root.Subcommands.Command(cli.Command[childConfig]{
						Name: "serve",
						Declare: func(child *cli.CommandBuilder[childConfig]) {
							child.Flags.Bool(&child.Config.Verbose).Long("verbose")
							child.Subcommands.CommandsFrom(
								func(context.Context, *cli.CommandBatch[childConfig]) error { return nil },
							)
							child.Shortcut(cli.CommandShortcut{
								Name:       "quick",
								AppendArgs: []string{"--help"},
							})
						},
					})
				},
			}
			app := clitest.Compile(t, definition)

			completionRequest := cli.CompletionRequest{
				Args: []string{"quick", ""},
				Cursor: cli.CompletionCursor{
					Argument: 1,
				},
				Client: cli.CompletionClient{
					Capabilities: cli.CompletionClientCapabilities{
						CandidateKinds: true,
					},
				},
			}
			clitest.Complete(t.Context(), app, completionRequest).Require(t,
				clitest.CompletionNotPartial(),
				clitest.CompletionItem("--help").Absent(),
				clitest.CompletionItem("--verbose").Present(),
			)
		})
	}
}

func TestCompleteShortcutFixedTailUsesProfileQualifiedProviderSettlement(t *testing.T) {
	t.Parallel()
	type rootConfig = completionShortcutGateRootConfig
	type childConfig = completionShortcutProfileChildConfig
	type dynamicConfig = completionShortcutDynamicConfig

	for _, test := range []struct {
		name           string
		forceReference bool
	}{
		{
			name: "static projection",
		},
		{
			name:           "detached reference",
			forceReference: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var providerCalls atomic.Int32
			profileDomain := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
				values.Value("a", "a")
			})
			app := clitest.Compile(t, cli.Definition[rootConfig]{
				Name: "tool",
				Declare: func(root *cli.CommandBuilder[rootConfig]) {
					if test.forceReference {
						gate := root.Values.String(&root.Config.Gate)
						root.Values.Default(gate, "yes")
						root.VisibleWhen(root.When.Equal(gate, "yes"))
					}
					root.Subcommands.Command(cli.Command[childConfig]{
						Name: "serve",
						Declare: func(child *cli.CommandBuilder[childConfig]) {
							profile := child.Values.Value(&child.Config.Profile, profileDomain)
							child.Flags.ValueFor(profile).Occurrences(cli.Count.Optional()).Long("profile")
							child.Profiles.Switch(profile, func(cases *cli.GrammarProfileCases[childConfig, string]) {
								cases.Case("a", func(active *cli.CommandBuilder[childConfig]) {
									active.Subcommands.CommandsFrom(
										func(_ context.Context, batch *cli.CommandBatch[childConfig]) error {
											providerCalls.Add(1)
											batch.Command(cli.Command[dynamicConfig]{
												Name: "dyn",
											})
											return nil
										},
									)
								})
							})
							child.Shortcut(cli.CommandShortcut{
								Name:       "quick",
								AppendArgs: []string{"a", "dyn"},
							})
						},
					})
				},
			})

			completionRequest := cli.CompletionRequest{
				Args: []string{"quick", ""},
				Cursor: cli.CompletionCursor{
					Argument: 1,
				},
				Client: cli.CompletionClient{
					Capabilities: cli.CompletionClientCapabilities{
						CandidateKinds: true,
					},
				},
			}
			clitest.Complete(t.Context(), app, completionRequest).Require(t,
				clitest.CompletionNotPartial(),
				clitest.CompletionItem("--profile").Present(),
			)
			if calls := providerCalls.Load(); calls != 0 {
				t.Fatalf("profile provider calls = %d, want 0 during static candidate validation", calls)
			}
		})
	}
}

func TestCompleteShortcutLegalAppendArgsCanonicalSelectionDoesNotBecomePartial(t *testing.T) {
	t.Parallel()
	type rootConfig = completionShortcutRootConfig
	type parentConfig = completionShortcutParentConfig
	type leafConfig = completionShortcutLeafConfig
	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[parentConfig]{
				Name: "serve",
				Declare: func(parent *cli.CommandBuilder[parentConfig]) {
					parent.Subcommands.Command(cli.Command[leafConfig]{
						Name: "leaf",
					})
					parent.Shortcut(cli.CommandShortcut{
						Name:       "quick",
						AppendArgs: []string{"leaf"},
					})
				},
			})
		},
	})
	schema := clitest.Schema(app).Schema()
	root := completionCommandByPath(t, schema)
	serve := completionCommandByPath(t, schema, "serve")
	leaf := completionCommandByPath(t, schema, "serve", "leaf")
	path := []clitest.CommandLocator{root, serve, leaf}
	completionRequest := cli.CompletionRequest{
		Args: []string{"quick", ""},
		Cursor: cli.CompletionCursor{
			Argument: 1,
		},
		Client: cli.CompletionClient{
			Capabilities: cli.CompletionClientCapabilities{
				CandidateKinds: true,
			},
		},
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t,
		clitest.CompletionItem("leaf").Absent(),
		clitest.CompletionNotPartial(),
		clitest.SelectionPathExactly(path...),
	)
	clitest.Complete(t.Context(), app, cli.CompletionRequest{
		Args: []string{"serve", "leaf", ""},
		Cursor: cli.CompletionCursor{
			Argument: 2,
		},
	}).Require(t,
		clitest.SelectionPathExactly(path...),
	)
}

func TestCompleteShortcutCommandCandidatePreservesLegalFixedTailSelection(t *testing.T) {
	t.Parallel()
	type rootConfig = completionShortcutRootConfig
	type parentConfig = completionShortcutParentConfig
	type alphaConfig = completionShortcutAlphaConfig
	type betaConfig = completionShortcutBetaConfig
	type otherConfig = completionShortcutOtherConfig
	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[parentConfig]{
				Name: "serve",
				Declare: func(parent *cli.CommandBuilder[parentConfig]) {
					parent.Subcommands.Command(cli.Command[alphaConfig]{
						Name: "alpha",
						Declare: func(alpha *cli.CommandBuilder[alphaConfig]) {
							alpha.Subcommands.Command(cli.Command[betaConfig]{
								Name: "beta",
							})
						},
					})
					parent.Subcommands.Command(cli.Command[otherConfig]{
						Name: "other",
					})
					parent.Shortcut(cli.CommandShortcut{
						Name:       "quick",
						AppendArgs: []string{"beta"},
					})
				},
			})
		},
	})
	completionRequest := cli.CompletionRequest{
		Args: []string{"quick", ""},
		Cursor: cli.CompletionCursor{
			Argument: 1,
		},
		Client: cli.CompletionClient{
			Capabilities: cli.CompletionClientCapabilities{
				CandidateKinds: true,
			},
		},
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t,
		clitest.CompletionNotPartial(),
		clitest.CompletionItem("alpha").Present(),
		clitest.CompletionItem("other").Absent(),
	)
}

func TestCompleteCanonicalCommandRemainsShortcutIndependent(t *testing.T) {
	t.Parallel()
	type rootConfig = completionShortcutRootConfig
	type childConfig = completionShortcutNameChildConfig
	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "serve",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					child.Arguments.Form(func(form *cli.ArgumentFormBuilder[childConfig]) {
						form.String(
							&form.Config.Name,
							"name",
						).CompleteBy(
							func(context.Context, *childConfig, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
								contribution := cli.CompletionContribution{
									Candidates: []cli.CompletionCandidate{
										{
											Value: "canonical",
											Kind:  cli.CompletionCandidateValue,
										},
									},
								}
								return contribution, nil
							},
						)
					})
					child.Shortcut(cli.CommandShortcut{
						Name: "quick",
						Args: []string{"fixed"},
					})
				},
			})
		},
	})
	completionRequest := cli.CompletionRequest{
		Args: []string{"serve", ""},
		Cursor: cli.CompletionCursor{
			Argument: 1,
		},
		Client: cli.CompletionClient{
			Capabilities: cli.CompletionClientCapabilities{
				CandidateKinds: true,
			},
		},
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t,
		clitest.CompletionItemsExactly(clitest.CompletionItem("canonical")),
		clitest.CompletionObservedInputSourcesExactly(cli.InputKindShortcut),
	)
}

func TestCompleteShortcutFixedDescendantSelectsResidualResponseDialect(t *testing.T) {
	t.Parallel()
	type rootConfig = completionShortcutRootConfig
	type parentConfig = completionShortcutParentConfig
	type childConfig = completionShortcutModeNameChildConfig
	app, err := cli.Compile(cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[parentConfig]{
				Name: "serve",
				Declare: func(parent *cli.CommandBuilder[parentConfig]) {
					parent.Dialect.ResponseFiles(false)
					parent.Subcommands.Command(cli.Command[childConfig]{
						Name: "leaf",
						Declare: func(child *cli.CommandBuilder[childConfig]) {
							child.Dialect.ResponseFiles(true)
							mode := child.Values.String(&child.Config.Mode)
							child.Flags.ValueFor(mode).Long("mode")
							child.Arguments.Form(func(form *cli.ArgumentFormBuilder[childConfig]) {
								form.String(
									&form.Config.Name,
									"name",
								).CompleteBy(func(_ context.Context, cfg *childConfig, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
									contribution := cli.CompletionContribution{
										Candidates: []cli.CompletionCandidate{
											{
												Value: cfg.Mode,
												Kind:  cli.CompletionCandidateValue,
											},
										},
									}
									return contribution, nil
								}).
									From(mode)
							})
						},
					})
					parent.Shortcut(cli.CommandShortcut{
						Name: "quick",
						Args: []string{"leaf"},
					})
				},
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	loads := 0
	result, err := func() (cli.Completion, error) {
		completionRequest := cli.CompletionRequest{
			Args: []string{"quick", "@cfg", ""},
			Cursor: cli.CompletionCursor{
				Argument: 2,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					CandidateKinds: true,
				},
			},
			ResponseFiles: cli.ResponseFileAccess{
				Load: func(_ context.Context, request cli.ResponseFileRequest) (cli.ResponseFile, error) {
					loads++
					if request.Path != "cfg" {
						return cli.ResponseFile{}, errors.New("unexpected response path")
					}
					file := cli.ResponseFile{
						Key:      "cfg",
						Revision: cli.Revision("r1"),
						Content:  []byte("--mode fast"),
					}
					return file, nil
				},
			},
		}
		proof := clitest.Complete(t.Context(), app, completionRequest)
		proof.Require(t, clitest.CompletionItem("fast").Present())
		return proof.Completion()
	}()
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if loads != 1 {
		t.Fatalf("response loads = %d, want 1 after fixed descendant selects response-enabled dialect", loads)
	}
	completionShortcutAssertCompletionValues(t, result.Items, "fast")
}

func TestCompleteShortcutCursorChildEnablesAppendResponseRecognition(t *testing.T) {
	t.Parallel()
	type rootConfig = completionShortcutRootConfig
	type parentConfig = completionShortcutParentConfig
	type alphaConfig = completionShortcutModeAlphaConfig
	app, err := cli.Compile(cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[parentConfig]{
				Name: "serve",
				Declare: func(parent *cli.CommandBuilder[parentConfig]) {
					parent.Dialect.ResponseFiles(false)
					parent.Subcommands.Command(cli.Command[alphaConfig]{
						Name: "alpha",
						Declare: func(alpha *cli.CommandBuilder[alphaConfig]) {
							alpha.Dialect.ResponseFiles(true)
							alpha.Flags.String(&alpha.Config.Mode).Long("mode")
						},
					})
					parent.Shortcut(cli.CommandShortcut{
						Name:       "quick",
						AppendArgs: []string{"@cfg"},
					})
				},
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	loads := 0
	result, err := func() (cli.Completion, error) {
		completionRequest := cli.CompletionRequest{
			Args: []string{"quick", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					CandidateKinds: true,
				},
			},
			ResponseFiles: cli.ResponseFileAccess{
				Load: func(_ context.Context, request cli.ResponseFileRequest) (cli.ResponseFile, error) {
					loads++
					if request.Path != "cfg" {
						return cli.ResponseFile{}, errors.New("unexpected response path")
					}
					file := cli.ResponseFile{
						Key:      "cfg",
						Revision: cli.Revision("r1"),
						Content:  []byte("--mode tail"),
					}
					return file, nil
				},
			},
		}
		proof := clitest.Complete(t.Context(), app, completionRequest)
		proof.Require(t, clitest.CompletionItem("alpha").Present())
		return proof.Completion()
	}()
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if loads != 1 {
		t.Fatalf("candidate-selected response loads = %d, want 1", loads)
	}
	alphaOffered := false
	for _, item := range result.Items {
		if item.Value == "alpha" {
			alphaOffered = true
		}
	}
	if !alphaOffered {
		t.Fatalf("response-enabled child candidate missing: %#v", result.Items)
	}
}

func TestCompleteShortcutCursorChildDisablesAppendResponseRecognitionWithoutLoad(t *testing.T) {
	t.Parallel()
	type rootConfig = completionShortcutRootConfig
	type parentConfig = completionShortcutParentConfig
	type alphaConfig = completionShortcutFileAlphaConfig
	app, err := cli.Compile(cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[parentConfig]{
				Name: "serve",
				Declare: func(parent *cli.CommandBuilder[parentConfig]) {
					parent.Dialect.ResponseFiles(true)
					parent.Subcommands.Command(cli.Command[alphaConfig]{
						Name: "alpha",
						Declare: func(alpha *cli.CommandBuilder[alphaConfig]) {
							alpha.Dialect.ResponseFiles(false)
							alpha.Arguments.Form(func(form *cli.ArgumentFormBuilder[alphaConfig]) {
								form.String(&form.Config.File, "file")
							})
						},
					})
					parent.Shortcut(cli.CommandShortcut{
						Name:       "quick",
						AppendArgs: []string{"@cfg"},
					})
				},
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	loads := 0
	result, err := func() (cli.Completion, error) {
		completionRequest := cli.CompletionRequest{
			Args: []string{"quick", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					CandidateKinds: true,
				},
			},
			ResponseFiles: cli.ResponseFileAccess{
				Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
					loads++
					return cli.ResponseFile{}, errors.New("disabled child must not load response source")
				},
			},
		}
		proof := clitest.Complete(t.Context(), app, completionRequest)
		proof.Require(t, clitest.CompletionItem("alpha").Present())
		return proof.Completion()
	}()
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if loads != 0 {
		t.Fatalf("response-disabled child caused %d Load calls, want 0", loads)
	}
	alphaOffered := false
	for _, item := range result.Items {
		if item.Value == "alpha" {
			alphaOffered = true
		}
	}
	if !alphaOffered {
		t.Fatalf("response-disabled child candidate missing: %#v", result.Items)
	}
}

func TestCompleteShortcutCandidateResponseFailurePreservesIndependentCandidate(t *testing.T) {
	t.Parallel()
	type rootConfig = completionShortcutRootConfig
	type parentConfig = completionShortcutParentConfig
	type alphaConfig = completionShortcutAlphaConfig
	type betaConfig = completionShortcutFileBetaConfig
	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[parentConfig]{
				Name: "serve",
				Declare: func(parent *cli.CommandBuilder[parentConfig]) {
					parent.Dialect.ResponseFiles(false)
					parent.Subcommands.Command(cli.Command[alphaConfig]{
						Name: "alpha",
						Declare: func(alpha *cli.CommandBuilder[alphaConfig]) {
							alpha.Dialect.ResponseFiles(true)
						},
					})
					parent.Subcommands.Command(cli.Command[betaConfig]{
						Name: "beta",
						Declare: func(beta *cli.CommandBuilder[betaConfig]) {
							beta.Dialect.ResponseFiles(false)
							beta.Arguments.Form(
								func(form *cli.ArgumentFormBuilder[betaConfig]) { form.String(&form.Config.File, "file") },
							)
						},
					})
					parent.Shortcut(cli.CommandShortcut{
						Name:       "quick",
						AppendArgs: []string{"@cfg"},
					})
				},
			})
		},
	})
	cause := errors.New("candidate response load failed")
	loads := 0
	completionRequest := cli.CompletionRequest{
		Args: []string{"quick", ""},
		Cursor: cli.CompletionCursor{
			Argument: 1,
		},
		Client: cli.CompletionClient{
			Capabilities: cli.CompletionClientCapabilities{
				CandidateKinds: true,
			},
		},
		ResponseFiles: cli.ResponseFileAccess{
			Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
				loads++
				return cli.ResponseFile{}, cause
			},
		},
	}
	proof := clitest.Complete(t.Context(), app, completionRequest)
	proof.Require(t,
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticResponseFileLoadFailure).Present()),
		clitest.CompletionPartial(),
		clitest.CompletionObservedItemsExactly(clitest.CompletionItem("beta")),
	)
	_, err := proof.Completion()
	if !errors.Is(err, cause) {
		t.Fatalf("Complete() error = %v, want candidate response load cause", err)
	}
	if loads != 1 {
		t.Fatalf("candidate response loads = %d, want exactly 1", loads)
	}
}

func TestCompleteShortcutPendingFlagValueUsesFixedTailFinalCommandSelection(t *testing.T) {
	t.Parallel()
	type rootConfig = completionShortcutRootConfig
	type parentConfig = completionShortcutPathParentConfig
	type leafConfig = completionShortcutLeafConfig
	calls := 0
	var observed cli.CommandSelection
	app, err := cli.Compile(cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[parentConfig]{
				Name: "parent",
				Declare: func(parent *cli.CommandBuilder[parentConfig]) {
					parent.Flags.String(&parent.Config.Path).CompleteBy(func(_ context.Context, _ *parentConfig, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
						calls++
						observed = request.Selection
						contribution := cli.CompletionContribution{
							Candidates: []cli.CompletionCandidate{{
								Value: "file",
							}},
						}
						return contribution, nil
					}).
						Long("path")
					parent.Subcommands.Command(cli.Command[leafConfig]{
						Name: "leaf",
					})
					parent.Shortcut(cli.CommandShortcut{
						Name:       "quick",
						Args:       []string{"--path"},
						AppendArgs: []string{"leaf"},
					})
				},
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := func() (cli.Completion, error) {
		proof := clitest.Complete(t.Context(), app, cli.CompletionRequest{
			Args: []string{"quick", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
		})
		proof.Require(t, clitest.CompletionTargetAt(0).Present())
		return proof.Completion()
	}()
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("pending value completion calls = %d, want 1", calls)
	}
	if len(observed.Path) != 3 || observed.Selected == observed.Root {
		t.Fatalf("callback selection = %#v, want root -> parent -> leaf", observed)
	}
	if len(result.Selection.Path) != 3 || result.Selection.Selected != observed.Selected {
		t.Fatalf("result selection = %#v, callback = %#v", result.Selection, observed)
	}
}

func TestLoad(t *testing.T) {
	t.Parallel()

	t.Run("missing loader", func(t *testing.T) {
		t.Parallel()
		type rootConfig = completionShortcutRootConfig
		type parentConfig = completionShortcutParentConfig
		type childConfig = completionShortcutChildConfig
		app := clitest.Compile(t, cli.Definition[rootConfig]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[rootConfig]) {
				root.Subcommands.Command(cli.Command[parentConfig]{
					Name: "serve",
					Declare: func(parent *cli.CommandBuilder[parentConfig]) {
						parent.Subcommands.Command(cli.Command[childConfig]{
							Name: "alpha",
							Declare: func(alpha *cli.CommandBuilder[childConfig]) {
								alpha.Dialect.ResponseFiles(true)
							},
						})
						parent.Shortcut(cli.CommandShortcut{
							Name:       "quick",
							AppendArgs: []string{"@cfg"},
						})
					},
				})
			},
		})
		completionRequest := cli.CompletionRequest{
			Args: []string{"quick", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					CandidateKinds: true,
				},
			},
		}
		_, err := clitest.Complete(t.Context(), app, completionRequest).Completion()
		if err == nil {
			t.Fatal("Complete() unexpectedly accepted a required response expansion without a loader")
		}
	})

	t.Run("shared candidate load cache", func(t *testing.T) {
		t.Parallel()
		type rootConfig = completionShortcutRootConfig
		type parentConfig = completionShortcutParentConfig
		type childConfig = completionShortcutChildConfig
		app := clitest.Compile(t, cli.Definition[rootConfig]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[rootConfig]) {
				root.Subcommands.Command(cli.Command[parentConfig]{
					Name: "serve",
					Declare: func(parent *cli.CommandBuilder[parentConfig]) {
						parent.Subcommands.Command(cli.Command[childConfig]{
							Name: "alpha",
							Declare: func(alpha *cli.CommandBuilder[childConfig]) {
								alpha.Dialect.ResponseFiles(true)
							},
						})
						parent.Subcommands.Command(cli.Command[childConfig]{
							Name: "beta",
							Declare: func(beta *cli.CommandBuilder[childConfig]) {
								beta.Dialect.ResponseFiles(true)
							},
						})
						parent.Shortcut(cli.CommandShortcut{
							Name:       "quick",
							AppendArgs: []string{"@cfg"},
						})
					},
				})
			},
		})
		loads := 0
		completionRequest := cli.CompletionRequest{
			Args: []string{"quick", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					CandidateKinds: true,
				},
			},
			ResponseFiles: cli.ResponseFileAccess{
				Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
					loads++
					file := cli.ResponseFile{
						Key:      "cfg",
						Revision: cli.Revision("r1"),
					}
					return file, nil
				},
			},
		}
		result, err := clitest.Complete(t.Context(), app, completionRequest).Completion()
		if err != nil {
			t.Fatalf("Complete() error = %v", err)
		}
		if loads != 1 {
			t.Fatalf("response load calls = %d, want 1 shared load", loads)
		}
		seen := map[string]struct{}{}
		for _, item := range result.Items {
			seen[item.Value] = struct{}{}
		}
		_, sawAlpha := seen["alpha"]
		_, sawBeta := seen["beta"]
		if !sawAlpha || !sawBeta {
			t.Fatalf("Items = %#v, want both response-enabled candidates", result.Items)
		}
	})
}

func TestSettleCompletionShortcutPrefix(t *testing.T) {
	t.Parallel()
	type rootConfig = completionShortcutRootConfig
	type childConfig = completionShortcutChildConfig
	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "serve",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					child.Dialect.ResponseFiles(true)
					child.Shortcut(cli.CommandShortcut{
						Name: "quick",
						Args: []string{"@cfg"},
					})
				},
			})
		},
	})
	cause := errors.New("shortcut response load failed")
	completionRequest := cli.CompletionRequest{
		Args: []string{"quick", ""},
		Cursor: cli.CompletionCursor{
			Argument: 1,
		},
		ResponseFiles: cli.ResponseFileAccess{
			Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
				return cli.ResponseFile{}, cause
			},
		},
	}
	_, err := clitest.Complete(t.Context(), app, completionRequest).Completion()
	if !errors.Is(err, cause) {
		t.Fatalf("Complete() error = %v, want response load cause", err)
	}
}

func TestSettleNextCompletionShortcut(t *testing.T) {
	t.Parallel()
	type rootConfig = completionShortcutRootConfig
	type childConfig = completionShortcutValuesChildConfig
	app := clitest.Compile(t, cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "serve",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					values := child.Values.Sequence(&child.Config.Values, cli.Value.String())
					child.Arguments.Form(func(form *cli.ArgumentFormBuilder[childConfig]) {
						form.Sequence(values, "value", cli.Count.ZeroOrMore())
					})
					child.Shortcut(cli.CommandShortcut{
						Name: "quick",
						Args: []string{"fixed"},
					})
				},
			}).Default()
		},
	})
	schema := clitest.Schema(app).Schema()
	root := completionCommandByPath(t, schema)
	serve := completionCommandByPath(t, schema, "serve")
	clitest.Complete(t.Context(), app, cli.CompletionRequest{
		Args: []string{"first", ""},
		Cursor: cli.CompletionCursor{
			Argument: 1,
		},
	}).Require(t,
		serve.Selected(),
		clitest.SelectionPathExactly(root, serve),
	)
}

func completionShortcutAssertCompletionValues(t *testing.T, items []cli.CompletionItem, values ...string) {
	t.Helper()
	got := make(map[string]cli.CompletionCandidateKind, len(items))
	for _, item := range items {
		got[item.Value] = item.Kind
	}
	for _, value := range values {
		if got[value] != cli.CompletionCandidateValue {
			t.Fatalf("completion items = %#v; missing value %q", items, value)
		}
	}
}
