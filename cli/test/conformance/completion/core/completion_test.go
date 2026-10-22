package conformance

import (
	"context"
	"errors"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

type authoringCompletionNamePairConfig struct {
	Name string
	Pair string
}

type authoringCompletionABCDEFGConfig struct{ A, B, C, D, E, F, G string }

func TestFlagCompletionContinuationPreservesValueType(t *testing.T) {
	t.Parallel()
	type config struct {
		Direct   string
		Composed string
	}
	complete := func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
		return cli.CompletionContribution{}, nil
	}

	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			var _ cli.FlagSpellingDeclaration[config, string] = root.Flags.String(&root.Config.Direct).
				CompleteBy(complete).
				Long("direct")
			var _ cli.FlagSpellingDeclaration[config, string] = root.Flags.String(&root.Config.Composed).
				CompleteBy(complete).
				FallbackTo(cli.CompletionNativeFiles).
				Long("composed")
		},
	})

	root := clitest.RootCommand()
	clitest.Schema(app).Require(t,
		root.Flag("--direct").Completion().Dynamic(),
		root.Flag("--composed").Completion().Dynamic(),
		root.Flag("--composed").Completion().Native(cli.CompletionNativeFiles, cli.CompletionNativeOnNoMatch),
	)
}

func TestCompletionAuthoringProjectsDeclaredSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringCompletionNamePairConfig
	language := cli.TokenLanguage[string]{
		Name:   "pair",
		Count:  cli.Count.Exactly(2),
		Parse:  func(context.Context, []string) (string, error) { return "pair", nil },
		Format: func(string) []string { return []string{"left", "right"} },
	}
	definition := cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			name := root.Values.String(&root.Config.Name)
			pair := root.Values.String(&root.Config.Pair)
			root.Flags.ValueFor(name).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				return cli.CompletionContribution{}, nil
			}).FallbackTo(cli.CompletionNativeFiles).
				Long("name")
			root.Flags.ValueFor(pair).CompleteTokensBy(func(context.Context, *config, cli.TokenLanguageCompletionRequest) (cli.CompletionContribution, error) {
				return cli.CompletionContribution{}, nil
			}).CombineWith(cli.CompletionNativeDirectories).
				Long("pair")
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.ValueFor(
					name,
					"name",
				).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
					return cli.CompletionContribution{}, nil
				}).
					CombineWith(cli.CompletionNativePaths)
			})
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Tokens(
					pair,
					"pair",
					language,
				).CompleteTokensBy(func(context.Context, *config, cli.TokenLanguageCompletionRequest) (cli.CompletionContribution, error) {
					return cli.CompletionContribution{}, nil
				}).
					FallbackTo(cli.CompletionNativeExecutables)
			})
		},
	}
	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	clitest.Schema(app).Require(t,
		root.Flag("--name").Completion().Dynamic(),
		root.Flag("--name").Completion().Native(cli.CompletionNativeFiles, cli.CompletionNativeOnNoMatch),
		root.Flag("--pair").Completion().Dynamic(),
		root.Flag("--pair").Completion().Native(cli.CompletionNativeDirectories, cli.CompletionNativeAlways),
		root.ArgumentFormAt(0).TermAt(0).Completion().Dynamic(),
		root.ArgumentFormAt(0).TermAt(0).Completion().Native(cli.CompletionNativePaths, cli.CompletionNativeAlways),
		root.ArgumentFormAt(1).TermAt(0).Completion().Dynamic(),
		root.ArgumentFormAt(1).
			TermAt(0).
			Completion().
			Native(cli.CompletionNativeExecutables, cli.CompletionNativeOnNoMatch),
	)
}

func TestCompletionAuthoringProjectsExactNativeKindsAndModes(t *testing.T) {
	t.Parallel()
	type config = authoringCompletionABCDEFGConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			a := root.Values.String(&root.Config.A)
			b := root.Values.String(&root.Config.B)
			c := root.Values.String(&root.Config.C)
			d := root.Values.String(&root.Config.D)
			e := root.Values.String(&root.Config.E)
			f := root.Values.String(&root.Config.F)
			g := root.Values.String(&root.Config.G)
			root.Flags.ValueFor(a).CompleteWith(cli.CompletionNativePaths).Long("paths")
			root.Flags.ValueFor(b).CompleteWith(cli.CompletionNativeDirectories).Long("directories")
			root.Flags.ValueFor(c).CompleteWith(cli.CompletionNativeExecutables).Long("executables")
			root.Flags.ValueFor(d).CompleteWith(cli.CompletionNativeUsers).Long("users")
			root.Flags.ValueFor(e).CompleteWith(cli.CompletionNativeHosts).Long("hosts")
			root.Flags.ValueFor(f).CompleteWith(cli.CompletionNativeEnvironmentVariables).Long("environment")
			root.Flags.ValueFor(g).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				return cli.CompletionContribution{}, nil
			}).FallbackTo(cli.CompletionNativeFiles).
				Long("fallback")
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.ValueFor(
					g,
					"combine",
				).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
					return cli.CompletionContribution{}, nil
				}).
					CombineWith(cli.CompletionNativeDirectories)
			})
		},
	}
	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	clitest.Schema(app).Require(t,
		root.Flag("--paths").Completion().Native(cli.CompletionNativePaths, cli.CompletionNativeAlways),
		root.Flag("--directories").Completion().Native(cli.CompletionNativeDirectories, cli.CompletionNativeAlways),
		root.Flag("--executables").Completion().Native(cli.CompletionNativeExecutables, cli.CompletionNativeAlways),
		root.Flag("--users").Completion().Native(cli.CompletionNativeUsers, cli.CompletionNativeAlways),
		root.Flag("--hosts").Completion().Native(cli.CompletionNativeHosts, cli.CompletionNativeAlways),
		root.Flag("--environment").
			Completion().
			Native(cli.CompletionNativeEnvironmentVariables, cli.CompletionNativeAlways),
		root.Flag("--fallback").Completion().Dynamic(),
		root.Flag("--fallback").Completion().Native(cli.CompletionNativeFiles, cli.CompletionNativeOnNoMatch),
		root.ArgumentFormAt(0).TermAt(0).Completion().Dynamic(),
		root.ArgumentFormAt(0).
			TermAt(0).
			Completion().
			Native(cli.CompletionNativeDirectories, cli.CompletionNativeAlways),
	)
}

type completionContributionModePathConfig struct {
	Mode int
	Path string
}

type completionContributionModePathConfig2 struct{ Mode, Path string }

type completionContributionPathConfig struct{ Path string }

func declareCompletionContributionFixture(root *cli.CommandBuilder[struct{}]) {
	root.Dialect.ResponseFiles(true)
}

func TestApplyValueCompletionContribution(t *testing.T) {
	t.Parallel()

	t.Run("invalid spacing", func(t *testing.T) {
		t.Parallel()
		type config = completionContributionPathConfig
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Flags.String(&root.Config.Path).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
					contribution := cli.CompletionContribution{
						Spacing: cli.CompletionSpacing(255),
					}
					return contribution, nil
				}).
					Long("path")
			},
		})
		clitest.Complete(t.Context(), app, cli.CompletionRequest{
			Args: []string{"--path", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
		}).Require(t,
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidCompletionContribution).Present()),
			clitest.CompletionPartial())
	})

	t.Run("duplicate native owner", func(t *testing.T) {
		t.Parallel()
		type config = completionContributionPathConfig
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Flags.String(&root.Config.Path).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
					intent := &cli.CompletionNativeIntent{
						Kind: cli.CompletionNativeFiles,
						Mode: cli.CompletionNativeAlways,
					}
					contribution := cli.CompletionContribution{
						Native: intent,
					}
					return contribution, nil
				}).FallbackTo(cli.CompletionNativeFiles).
					Long("path")
			},
		})
		completionRequest := cli.CompletionRequest{
			Args: []string{"--path", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					Native: cli.CompletionNativeCapabilities{
						Files: true,
					},
				},
			},
		}
		clitest.Complete(t.Context(), app, completionRequest).Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidCompletionContribution).Present()), clitest.CompletionPartial())
	})

	t.Run("callback native request", func(t *testing.T) {
		t.Parallel()
		type config = completionContributionPathConfig
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Flags.String(&root.Config.Path).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
					intent := &cli.CompletionNativeIntent{
						Kind: cli.CompletionNativeFiles,
						Mode: cli.CompletionNativeAlways,
					}
					contribution := cli.CompletionContribution{
						Native: intent,
					}
					return contribution, nil
				}).
					Long("path")
			},
		})
		request := cli.CompletionRequest{
			Args: []string{"--path", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					Native: cli.CompletionNativeCapabilities{
						Files: true,
					},
				},
			},
		}
		native := clitest.CompletionNativeRequestAt(0)
		clitest.Complete(t.Context(), app, request).Require(t,
			clitest.CompletionNativeRequestsExactly(native),
			native.Kind(cli.CompletionNativeFiles), native.Mode(cli.CompletionNativeAlways))
	})

	t.Run("declared native fallback", func(t *testing.T) {
		t.Parallel()
		type config = completionContributionPathConfig
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Flags.String(&root.Config.Path).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
					return cli.CompletionContribution{}, nil
				}).FallbackTo(cli.CompletionNativeFiles).
					Long("path")
			},
		})
		request := cli.CompletionRequest{
			Args: []string{"--path", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					Native: cli.CompletionNativeCapabilities{
						Files: true,
					},
				},
			},
		}
		native := clitest.CompletionNativeRequestAt(0)
		clitest.Complete(t.Context(), app, request).Require(t,
			clitest.CompletionNativeRequestsExactly(native),
			native.Kind(cli.CompletionNativeFiles), native.Mode(cli.CompletionNativeOnNoMatch))
	})
}

func TestApplyResponseFileCompletionContribution(t *testing.T) {
	t.Parallel()

	t.Run("invalid replacement", func(t *testing.T) {
		t.Parallel()
		app := clitest.Compile(t, cli.Definition[struct{}]{
			Name:    "tool",
			Declare: declareCompletionContributionFixture,
		})
		completionRequest := cli.CompletionRequest{
			Args: []string{"@pre"},
			Cursor: cli.CompletionCursor{
				Argument: 0,
				Byte:     4,
			},
			ResponseFiles: cli.ResponseFileAccess{
				Complete: func(context.Context, cli.ResponseFileCompletionRequest) (cli.CompletionContribution, error) {
					completionRange := &cli.CompletionRange{
						Start: 0,
						End:   4,
					}
					contribution := cli.CompletionContribution{
						Replace: completionRange,
					}
					return contribution, nil
				},
			},
		}
		proof := clitest.Complete(t.Context(), app, completionRequest)
		proof.Require(
			t,
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidCompletionContribution).Present()),
			clitest.CompletionPartial(),
		)
	})

	t.Run("invalid spacing", func(t *testing.T) {
		t.Parallel()
		app := clitest.Compile(t, cli.Definition[struct{}]{
			Name:    "tool",
			Declare: declareCompletionContributionFixture,
		})
		completionRequest := cli.CompletionRequest{
			Args: []string{"@pre"},
			Cursor: cli.CompletionCursor{
				Argument: 0,
				Byte:     4,
			},
			ResponseFiles: cli.ResponseFileAccess{
				Complete: func(context.Context, cli.ResponseFileCompletionRequest) (cli.CompletionContribution, error) {
					contribution := cli.CompletionContribution{
						Spacing: cli.CompletionSpacing(255),
					}
					return contribution, nil
				},
			},
		}
		proof := clitest.Complete(t.Context(), app, completionRequest)
		proof.Require(
			t,
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidCompletionContribution).Present()),
			clitest.CompletionPartial(),
		)
	})

	t.Run("unmatched candidate", func(t *testing.T) {
		t.Parallel()
		app := clitest.Compile(t, cli.Definition[struct{}]{
			Name:    "tool",
			Declare: declareCompletionContributionFixture,
		})
		completionRequest := cli.CompletionRequest{
			Args: []string{"@pre"},
			Cursor: cli.CompletionCursor{
				Argument: 0,
				Byte:     4,
			},
			ResponseFiles: cli.ResponseFileAccess{
				Complete: func(context.Context, cli.ResponseFileCompletionRequest) (cli.CompletionContribution, error) {
					contribution := cli.CompletionContribution{
						Candidates: []cli.CompletionCandidate{{
							Value: "other",
							Kind:  cli.CompletionCandidateFile,
						}},
					}
					return contribution, nil
				},
			},
			Client: cli.CompletionClient{
				Match: cli.CompletionMatchOptions{
					Mode: cli.CompletionMatchModePrefix,
					Case: cli.CompletionCaseSensitive,
				},
			},
		}
		proof := clitest.Complete(t.Context(), app, completionRequest)
		proof.Require(t, clitest.CompletionItemsExactly())
	})

	t.Run("fallback messages and refresh", func(t *testing.T) {
		t.Parallel()
		app := clitest.Compile(t, cli.Definition[struct{}]{
			Name:    "tool",
			Declare: declareCompletionContributionFixture,
		})
		completionRequest := cli.CompletionRequest{
			Args: []string{"@pre"},
			Cursor: cli.CompletionCursor{
				Argument: 0,
				Byte:     4,
			},
			ResponseFiles: cli.ResponseFileAccess{
				Complete: func(context.Context, cli.ResponseFileCompletionRequest) (cli.CompletionContribution, error) {
					contribution := cli.CompletionContribution{
						Candidates: []cli.CompletionCandidate{
							{
								Value:        "prefix",
								Kind:         cli.CompletionCandidateFile,
								FallbackOnly: true,
							},
						},
						Messages: []cli.CompletionMessage{
							{
								Kind: cli.CompletionMessageInformation,
								Text: "choose",
							},
						},
						RefreshOnInput: true,
					}
					return contribution, nil
				},
			},
			Client: cli.CompletionClient{
				Match: cli.CompletionMatchOptions{
					Mode: cli.CompletionMatchModePrefix,
					Case: cli.CompletionCaseSensitive,
				},
				Capabilities: cli.CompletionClientCapabilities{
					Messages:       true,
					RefreshOnInput: true,
				},
			},
		}
		proof := clitest.Complete(t.Context(), app, completionRequest)
		item := clitest.CompletionItem("prefix")
		proof.Require(t,
			clitest.CompletionItemsExactly(item), item.FallbackOnly(),
			clitest.CompletionMessagesExactly(cli.CompletionMessage{
				Kind: cli.CompletionMessageInformation,
				Text: "choose",
			}),
			clitest.CompletionRefreshOnInput(),
		)
	})
}

func TestNormalizeCompletionGroups(t *testing.T) {
	t.Parallel()

	for _, groups := range [][]cli.CompletionGroup{
		{{}},
		{{
			ID: "same",
		}, {
			ID: "same",
		}},
	} {
		t.Run("reject invalid group set", func(t *testing.T) {
			t.Parallel()
			type config = completionContributionPathConfig
			app := clitest.Compile(t, cli.Definition[config]{
				Name: "tool",
				Declare: func(root *cli.CommandBuilder[config]) {
					root.Flags.String(&root.Config.Path).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
						contribution := cli.CompletionContribution{
							Groups: groups,
						}
						return contribution, nil
					}).
						Long("path")
				},
			})
			completionRequest := cli.CompletionRequest{
				Args: []string{"--path", ""},
				Cursor: cli.CompletionCursor{
					Argument: 1,
				},
				Client: cli.CompletionClient{
					Capabilities: cli.CompletionClientCapabilities{
						Groups: true,
					},
				},
			}
			proof := clitest.Complete(t.Context(), app, completionRequest)
			proof.Require(
				t,
				clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidCompletionContribution).Present()),
				clitest.CompletionPartial(),
			)
		})
	}

	t.Run("redact sensitive group label", func(t *testing.T) {
		t.Parallel()
		type config = completionContributionPathConfig
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Flags.String(&root.Config.Path).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
					contribution := cli.CompletionContribution{
						Groups: []cli.CompletionGroup{{
							ID:        "secret",
							Label:     "classified",
							Sensitive: true,
						}},
					}
					return contribution, nil
				}).
					Long("path")
			},
		})
		group := clitest.CompletionGroup("secret")
		completionRequest := cli.CompletionRequest{
			Args: []string{"--path", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					Groups: true,
				},
			},
		}
		clitest.Complete(t.Context(), app, completionRequest).Require(t,
			clitest.CompletionGroupsExactly(group), group.Label(""), group.Sensitive(),
		)
	})
}

func TestNormalizeValueCompletionCandidate(t *testing.T) {
	t.Parallel()

	t.Run("reject invalid candidate", func(t *testing.T) {
		t.Parallel()
		type config = completionContributionPathConfig
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Flags.String(&root.Config.Path).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
					contribution := cli.CompletionContribution{
						Candidates: []cli.CompletionCandidate{
							{
								Value: "bad",
								Kind:  cli.CompletionCandidateKind("invalid"),
							},
						},
					}
					return contribution, nil
				}).
					Long("path")
			},
		})
		proof := clitest.Complete(t.Context(), app, cli.CompletionRequest{
			Args: []string{"--path", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
		})
		proof.Require(
			t,
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidCompletionContribution).Present()),
			clitest.CompletionPartial(),
		)
	})

	t.Run("hide unsupported group", func(t *testing.T) {
		t.Parallel()
		type config = completionContributionPathConfig
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				root.Flags.String(&root.Config.Path).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
					contribution := cli.CompletionContribution{
						Groups: []cli.CompletionGroup{{
							ID:    "paths",
							Label: "Paths",
						}},
						Candidates: []cli.CompletionCandidate{
							{
								Value:   "alpha",
								Kind:    cli.CompletionCandidateValue,
								GroupID: "paths",
							},
						},
					}
					return contribution, nil
				}).
					Long("path")
			},
		})
		item := clitest.CompletionItem("alpha")
		clitest.Complete(t.Context(), app, cli.CompletionRequest{
			Args: []string{"--path", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
		}).Require(t,
			clitest.CompletionItemsExactly(item), item.Ungrouped(),
		)
	})
}

func TestNormalizeCompletionMessages(t *testing.T) {
	t.Parallel()
	type config = completionContributionPathConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Path).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				contribution := cli.CompletionContribution{
					Messages: []cli.CompletionMessage{
						{
							Kind:      cli.CompletionMessageInformation,
							Text:      "secret",
							Sensitive: true,
						},
					},
				}
				return contribution, nil
			}).
				Long("path")
		},
	})
	completionRequest := cli.CompletionRequest{
		Args: []string{"--path", ""},
		Cursor: cli.CompletionCursor{
			Argument: 1,
		},
		Client: cli.CompletionClient{
			Capabilities: cli.CompletionClientCapabilities{
				Messages: true,
			},
		},
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t, clitest.CompletionMessagesExactly(cli.CompletionMessage{
		Kind:      cli.CompletionMessageInformation,
		Text:      "",
		Sensitive: true,
	}))
}

func TestCompletionCallbackContext(t *testing.T) {
	t.Parallel()

	t.Run("invalid dependency value", func(t *testing.T) {
		t.Parallel()
		type config = completionContributionModePathConfig
		called := false
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				mode := root.Flags.Int(&root.Config.Mode)
				mode.Long("mode")
				root.Flags.String(&root.Config.Path).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
					called = true
					return cli.CompletionContribution{}, nil
				}).From(mode).
					Long("path")
			},
		})
		_, err := clitest.Complete(t.Context(), app, cli.CompletionRequest{
			Args: []string{"--mode", "not-an-int", "--path", ""},
			Cursor: cli.CompletionCursor{
				Argument: 3,
			},
		}).Completion()
		if err == nil {
			t.Fatal("Complete() unexpectedly succeeded with an invalid required dependency")
		}
		if called {
			t.Fatal("completion callback ran with an invalid required dependency")
		}
	})

	t.Run("unresolved required dependency", func(t *testing.T) {
		t.Parallel()
		type config = completionContributionModePathConfig2
		called := false
		app := clitest.Compile(t, cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				mode := root.Flags.String(&root.Config.Mode)
				mode.Long("mode")
				root.Flags.String(&root.Config.Path).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
					called = true
					return cli.CompletionContribution{}, nil
				}).From(mode).
					Long("path")
			},
		})
		_, err := clitest.Complete(t.Context(), app, cli.CompletionRequest{
			Args: []string{"--path", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
		}).Completion()
		if err == nil {
			t.Fatal("Complete() unexpectedly succeeded with an unresolved required dependency")
		}
		if called {
			t.Fatal("completion callback ran with an unresolved required dependency")
		}
	})
}

type completionStaticClassificationVerboseConfig struct {
	Verbose bool
}

func TestInheritedStaticFlagCompletionRetainsOwningClassification(t *testing.T) {
	t.Parallel()

	type config = completionStaticClassificationVerboseConfig

	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.HelpGroup("Global", func(flags *cli.Flags[config]) {
				flags.
					Bool(
						&root.Config.Verbose,
					).Placement(cli.FlagPlacementSubtree).
					Long("verbose")
			})
			root.Subcommands.Command(cli.Command[config]{
				Name: "child",
			})
		},
	})

	groupID := func(args []string, cursor int) string {
		completionRequest := cli.CompletionRequest{
			Args: args,
			Cursor: cli.CompletionCursor{
				Argument: cursor,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					Groups: true,
				},
			},
		}
		result, err := app.Complete(t.Context(), completionRequest)
		if err != nil {
			t.Fatalf("Complete(%v) error = %v", args, err)
		}
		for _, item := range result.Items {
			if item.Value != "--verbose" {
				continue
			}
			if item.GroupID == "" {
				t.Fatalf("--verbose group ID empty in %#v", result)
			}
			return item.GroupID
		}
		t.Fatalf("--verbose absent from %#v", result.Items)
		return ""
	}

	rootID := groupID([]string{""}, 0)
	childID := groupID([]string{"child", ""}, 1)
	if rootID != childID {
		t.Fatalf(
			"inherited static flag classification changed by presentation scope: root %q, child %q",
			rootID,
			childID,
		)
	}
}

type completionVerboseRootConfig struct{ Verbose bool }

type completionChildConfig struct{}

type completionSecretConfig struct{ Secret string }

type completionRestConfig struct{ Rest []string }

type completionLeftRightConfig struct{ Left, Right string }

type completionDessertConfig struct{ Dessert string }

type completionConfig struct{}

type completionPathConfig struct{ Path string }

type completionVerboseConfig struct{ Verbose bool }

type completionValueConfig struct{ Value string }

func TestCompleteFreshTokenUsesCompiledStaticSyntaxWithoutExecution(t *testing.T) {
	t.Parallel()
	type rootConfig = completionVerboseRootConfig
	type childConfig = completionChildConfig

	runs := 0
	definition := cli.Definition[rootConfig]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"child description": "description enfant",
				},
			}},
		},
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Flags.Bool(&root.Config.Verbose).Description("verbose description").Long("verbose")
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "child",
				Description: cli.Description{
					Short: "child description",
				},
				Run: func(context.Context, *childConfig) error { runs++; return nil },
			})
			root.Subcommands.Command(cli.Command[childConfig]{
				Name:   "hidden",
				Hidden: true,
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	result, err := func() (cli.Completion, error) {
		completionRequest := cli.CompletionRequest{
			Args: []string{""},
			Cursor: cli.CompletionCursor{
				Argument: 0,
				Byte:     0,
			},
			Locale: "fr",
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					Descriptions:   true,
					CandidateKinds: true,
				},
			},
		}
		proof := clitest.Complete(t.Context(), app, completionRequest)
		proof.Require(t,
			clitest.CompletionTargetAt(0).Present(),
			clitest.CompletionReplace(cli.CompletionSpan{
				Argument: 0,
				Start:    0,
				End:      0,
			}),
			clitest.CompletionItem("child").Kind(cli.CompletionCandidateCommand),
			clitest.CompletionItem("child").DisplayText("child"),
			clitest.CompletionItem("child").Description("description enfant"),
			clitest.CompletionItem("--verbose").Kind(cli.CompletionCandidateFlag),
			clitest.CompletionItem("--verbose").DisplayText("--verbose"),
			clitest.CompletionItem("--verbose").Description("verbose description"),
			clitest.CompletionItem("hidden").Absent(),
			clitest.CompletionRankingAlgorithm(cli.CompletionRankingAlgorithmRelevance),
		)
		return proof.Completion()
	}()
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if runs != 0 {
		t.Fatalf("completion executed handler %d times", runs)
	}
	if result.Locale != "fr" || result.Selection.Selected != result.Selection.Root {
		t.Fatalf("result locale/selection = %#v", result)
	}
	for _, item := range result.Items {
		if item.Value == "child" &&
			(item.Declaration.Kind != cli.DeclarationKindCommandEntry || item.Declaration.CommandEntry == 0) {
			t.Fatalf("child declaration identity = %#v", item.Declaration)
		}
	}

	result.Items[0].TargetIndexes[0] = 99
	result.Targets[0].CommandPath = append(result.Targets[0].CommandPath, "mutated")
	again, err := app.Complete(t.Context(), cli.CompletionRequest{
		Args: []string{""},
		Cursor: cli.CompletionCursor{
			Argument: 0,
		},
	})
	if err != nil {
		t.Fatalf("second Complete() error = %v", err)
	}
	for _, item := range again.Items {
		for _, index := range item.TargetIndexes {
			if index == 99 {
				t.Fatalf("second result retained caller mutation: %#v", again.Items)
			}
		}
	}
}

func TestCompleteRejectsInvalidCursor(t *testing.T) {
	t.Parallel()
	type config = completionConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
	})
	requests := []cli.CompletionRequest{
		{
			Args: []string{""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
		},
		{
			Args: []string{""},
			Cursor: cli.CompletionCursor{
				Argument: 0,
			},
			Client: cli.CompletionClient{
				Ranking: cli.CompletionRankingMode(255),
			},
		},
		{
			Args: []string{""},
			Cursor: cli.CompletionCursor{
				Argument: 0,
			},
			Limit: -1,
		},
	}
	for _, request := range requests {
		clitest.Complete(t.Context(), app, request).
			Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidCompletionRequest).Present()))
	}
}

func TestCompleteAppliesDeterministicLimitAfterRanking(t *testing.T) {
	t.Parallel()
	type config = completionConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.Command(cli.Command[config]{
				Name: "zeta",
			})
			root.Subcommands.Command(cli.Command[config]{
				Name: "alpha",
			})
		},
	}
	app := clitest.Compile(t, definition)
	completionRequest := cli.CompletionRequest{
		Args: []string{""},
		Cursor: cli.CompletionCursor{
			Argument: 0,
		},
		Limit: 1,
		Client: cli.CompletionClient{
			Ranking: cli.CompletionRankingModeLexical,
			Capabilities: cli.CompletionClientCapabilities{
				CandidateKinds: true,
			},
		},
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t, clitest.CompletionObservedItemsExactly(clitest.CompletionItem("alpha")), clitest.CompletionTruncated())
}

func TestCompleteAllInvocableIncludesHiddenEntriesWithoutChangingPublicVisibility(t *testing.T) {
	t.Parallel()
	type config = completionConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.Command(cli.Command[config]{
				Name:   "hidden",
				Hidden: true,
			})
		},
	})
	public := cli.CompletionRequest{
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
	clitest.Complete(t.Context(), app, public).Require(t, clitest.CompletionItem("hidden").Absent())
	all := public
	all.Visibility = cli.CompletionVisibilityAllInvocable
	clitest.Complete(t.Context(), app, all).Require(t, clitest.CompletionItem("hidden").Present())
}

func TestCompleteNativeIntentHonorsClientCapability(t *testing.T) {
	t.Parallel()
	type config = completionPathConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Path).CompleteWith(cli.CompletionNativeFiles).Long("path")
		},
	})
	target := clitest.CompletionTargetAt(0)
	clitest.Complete(t.Context(), app, cli.CompletionRequest{
		Args: []string{"--path", ""},
		Cursor: cli.CompletionCursor{
			Argument: 1,
		},
	}).Require(t,
		clitest.CompletionTargetsExactly(target), clitest.CompletionNativeRequestsExactly(),
	)
}

func TestCompleteNonemptyCursorAppliesClientPrefixAfterSemanticGeneration(t *testing.T) {
	t.Parallel()
	type config = completionVerboseConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.Command(cli.Command[config]{
				Name: "serve",
			})
			root.Subcommands.Command(cli.Command[config]{
				Name: "status",
			})
			root.Flags.Bool(&root.Config.Verbose).Long("verbose")
		},
	}
	app := clitest.Compile(t, definition)
	client := cli.CompletionClient{
		Match: cli.CompletionMatchOptions{
			Mode:   cli.CompletionMatchModePrefix,
			Case:   cli.CompletionCaseSensitive,
			Fields: []cli.CompletionMatchField{cli.CompletionMatchFieldValue},
		},
		Capabilities: cli.CompletionClientCapabilities{
			CandidateKinds: true,
			MatchRanges:    true,
		},
	}
	serve := clitest.CompletionItem("serve")
	completionRequest := cli.CompletionRequest{
		Args: []string{"se"},
		Cursor: cli.CompletionCursor{
			Argument: 0,
			Byte:     2,
		},
		Client: client,
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t,
		clitest.CompletionReplace(cli.CompletionSpan{
			Argument: 0,
			Start:    0,
			End:      2,
		}),
		clitest.CompletionItemsExactly(serve),
		serve.Match(cli.CompletionMatchKindPrefix),
		serve.MatchField(cli.CompletionMatchFieldValue), serve.MatchRangesExactly(cli.CompletionRange{
			Start: 0,
			End:   2,
		}),
	)
	verbose := clitest.CompletionItem("--verbose")
	completionRequestUnderTest := cli.CompletionRequest{
		Args: []string{"--v"},
		Cursor: cli.CompletionCursor{
			Argument: 0,
			Byte:     3,
		},
		Client: client,
	}
	clitest.Complete(t.Context(), app, completionRequestUnderTest).Require(t,
		clitest.CompletionReplace(cli.CompletionSpan{
			Argument: 0,
			Start:    0,
			End:      3,
		}),
		clitest.CompletionItemsExactly(verbose),
		verbose.Kind(cli.CompletionCandidateFlag),
		verbose.Match(cli.CompletionMatchKindPrefix),
		verbose.MatchField(cli.CompletionMatchFieldValue), verbose.MatchRangesExactly(cli.CompletionRange{
			Start: 0,
			End:   3,
		}),
	)
}

func TestCompleteClientMatchingDoesNotExposeSensitiveCandidateText(t *testing.T) {
	t.Parallel()
	type config = completionSecretConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Secret).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				contribution := cli.CompletionContribution{
					Candidates: []cli.CompletionCandidate{
						{
							Value:       "prefix-secret",
							DisplayText: "prefix-secret",
							Sensitive:   true,
						},
					},
				}
				return contribution, nil
			}).
				Long("secret")
		},
	})
	item := clitest.CompletionItemAt(0)
	completionRequest := cli.CompletionRequest{
		Args: []string{"--secret", "pre"},
		Cursor: cli.CompletionCursor{
			Argument: 1,
			Byte:     3,
		},
		Client: cli.CompletionClient{
			Match: cli.CompletionMatchOptions{
				Mode: cli.CompletionMatchModePrefix,
				Case: cli.CompletionCaseSensitive,
			},
			Capabilities: cli.CompletionClientCapabilities{
				MatchRanges: true,
			},
		},
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t,
		clitest.CompletionItemsExactly(item),
		item.Sensitive(),
		item.Value(""),
		item.DisplayText(""), item.Match(cli.CompletionMatchKindPrefix), item.MatchRangesExactly(cli.CompletionRange{
			Start: 0,
			End:   3,
		}),
	)
}

func TestCompleteRejectsCallbackReplacementOutsideCurrentArgument(t *testing.T) {
	t.Parallel()
	type config = completionPathConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Path).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				completionRange := &cli.CompletionRange{
					Start: 0,
					End:   4,
				}
				contribution := cli.CompletionContribution{
					Replace: completionRange,
				}
				return contribution, nil
			}).
				Long("path")
		},
	})
	completionRequest := cli.CompletionRequest{
		Args: []string{"--path", "pre"},
		Cursor: cli.CompletionCursor{
			Argument: 1,
			Byte:     3,
		},
		Client: cli.CompletionClient{
			Match: cli.CompletionMatchOptions{
				Mode: cli.CompletionMatchModePrefix,
				Case: cli.CompletionCaseSensitive,
			},
		},
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t,
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidCompletionContribution).Present()),
		clitest.CompletionPartial(),
		clitest.CompletionObservedTargetsExactly(clitest.CompletionTargetAt(0)),
		clitest.CompletionObservedItemsExactly(),
	)
}

func TestCompleteRejectsLateInvalidValueContributionAtomically(t *testing.T) {
	t.Parallel()
	type config = completionPathConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Path).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				replace := cli.CompletionRange{
					Start: 1,
					End:   2,
				}
				contribution := cli.CompletionContribution{
					Replace: &replace,
					Candidates: []cli.CompletionCandidate{{
						Value:   "prefix",
						GroupID: "group",
					}},
					Groups: []cli.CompletionGroup{{
						ID:    "group",
						Label: "Group",
					}},
					Messages: []cli.CompletionMessage{{
						Kind: cli.CompletionMessageKind(99),
						Text: "invalid",
					}},
					Spacing:   cli.CompletionSpacingNoSpace,
					Truncated: true,
				}
				return contribution, nil
			}).
				Long("path")
		},
	})
	completionRequest := cli.CompletionRequest{
		Args: []string{"--path", "pre"},
		Cursor: cli.CompletionCursor{
			Argument: 1,
			Byte:     3,
		},
		Client: cli.CompletionClient{
			Match: cli.CompletionMatchOptions{
				Mode: cli.CompletionMatchModePrefix,
				Case: cli.CompletionCaseSensitive,
			},
			Capabilities: cli.CompletionClientCapabilities{
				Groups:   true,
				Messages: true,
			},
		},
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t,
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidCompletionContribution).Present()),
		clitest.CompletionPartial(),
		clitest.CompletionReplace(cli.CompletionSpan{
			Argument: 1,
			Start:    0,
			End:      3,
		}), clitest.CompletionSpacing(cli.CompletionSpacingDefault),
		clitest.CompletionObservedTargetsExactly(clitest.CompletionTargetAt(0)),
		clitest.CompletionObservedItemsExactly(),
		clitest.CompletionObservedGroupsExactly(),
		clitest.CompletionObservedMessagesExactly(),
		clitest.CompletionObservedNativeRequestsExactly(),
		clitest.CompletionNotTruncated(), clitest.CompletionNoRefreshOnInput(),
	)
}

func TestCompleteDelegatesOnlyAfterAuthoritativeRemainderOwnership(t *testing.T) {
	t.Parallel()
	type config = completionRestConfig
	var observed cli.CompletionDelegationRequest
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			destination := root.Values.Sequence(&root.Config.Rest, cli.Value.String())
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Remainder(
					destination,
					"rest",
					cli.Count.ZeroOrMore(),
				).DelegateCompletionBy(
					func(_ context.Context, _ *config, request cli.CompletionDelegationRequest) (cli.CompletionDelegation, error) {
						observed = request
						delegation := cli.CompletionDelegation{
							Args:   append([]string(nil), request.Args...),
							Cursor: request.Cursor,
						}
						return delegation, nil
					},
				)
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := func() (cli.Completion, error) {
		proof := clitest.Complete(t.Context(), app, cli.CompletionRequest{
			Args: []string{"left", "ri"},
			Cursor: cli.CompletionCursor{
				Argument: 1,
				Byte:     2,
			},
		})
		proof.Require(t, clitest.CompletionTarget(cli.CompletionTargetRemainder).Present())
		return proof.Completion()
	}()
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if observed.Target.Kind != cli.CompletionTargetRemainder || len(observed.Args) != 2 || observed.Args[0] != "left" ||
		observed.Args[1] != "ri" ||
		observed.Cursor != (cli.CompletionCursor{
			Argument: 1,
			Byte:     2,
		}) {
		t.Fatalf("delegation request = %#v", observed)
	}
	if result.Delegation == nil || len(result.Delegation.Args) != 2 || len(result.Items) != 0 ||
		len(result.NativeRequests) != 0 {
		t.Fatalf("delegated result = %#v", result)
	}
	if len(result.Targets) != 1 || result.Targets[0].Kind != cli.CompletionTargetRemainder {
		t.Fatalf("delegation targets = %#v", result.Targets)
	}
}

func TestCompletePositionalTargetFailurePreservesValidPeerAsPartial(t *testing.T) {
	t.Parallel()
	type config = completionLeftRightConfig
	failed := errors.New("left completion failed")
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.String(
					&form.Config.Left,
					"left",
				).CompleteBy(
					func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
						contribution := cli.CompletionContribution{
							Candidates: []cli.CompletionCandidate{{
								Value: "discarded",
							}},
						}
						return contribution, failed
					},
				)
			})
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.String(
					&form.Config.Right,
					"right",
				).CompleteBy(
					func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
						contribution := cli.CompletionContribution{
							Candidates: []cli.CompletionCandidate{{
								Value: "retained",
							}},
						}
						return contribution, nil
					},
				)
			})
		},
	})
	proof := clitest.Complete(t.Context(), app, cli.CompletionRequest{
		Args: []string{""},
		Cursor: cli.CompletionCursor{
			Argument: 0,
		},
	})
	proof.Require(t,
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticCompletionFailure).Present()),
		clitest.CompletionPartial(),
		clitest.CompletionItem("retained").Present(),
	)
	_, err := proof.Completion()
	if err == nil || !errors.Is(err, failed) {
		t.Fatalf("Complete() error = %v", err)
	}
}

func TestCompleteDefaultChildOwnsUnknownCursorWhileParentCommandCandidateRemainsViable(t *testing.T) {
	t.Parallel()
	type config = completionValueConfig
	var observed cli.ValueCompletionRequest
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.Command(cli.Command[config]{
				Name: "visible",
			})
			root.Subcommands.Command(cli.Command[config]{
				Name: "default",
				Declare: func(child *cli.CommandBuilder[config]) {
					child.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
						form.String(
							&form.Config.Value,
							"value",
						).CompleteBy(
							func(_ context.Context, _ *config, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
								observed = request
								contribution := cli.CompletionContribution{
									Candidates: []cli.CompletionCandidate{{
										Value: "value",
									}},
								}
								return contribution, nil
							},
						)
					})
				},
			}).Default()
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := func() (cli.Completion, error) {
		completionRequest := cli.CompletionRequest{
			Args: []string{"v"},
			Cursor: cli.CompletionCursor{
				Argument: 0,
				Byte:     1,
			},
			Client: cli.CompletionClient{
				Match: cli.CompletionMatchOptions{
					Mode: cli.CompletionMatchModePrefix,
					Case: cli.CompletionCaseSensitive,
				},
			},
		}
		proof := clitest.Complete(t.Context(), app, completionRequest)
		proof.Require(t, clitest.CompletionTargetAt(0).Present())
		return proof.Completion()
	}()
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if result.Selection.Selected == result.Selection.Root || observed.Selection.Selected != result.Selection.Selected ||
		observed.Target.Kind != cli.CompletionTargetArgument {
		t.Fatalf("default selection/request = selection=%#v request=%#v", result.Selection, observed)
	}
	var commandCandidate, argumentCandidate bool
	for _, item := range result.Items {
		switch item.Value {
		case "visible":
			commandCandidate = true
		case "value":
			argumentCandidate = true
		}
	}
	if !commandCandidate || !argumentCandidate {
		t.Fatalf(
			"parent/child completion composition missing command=%v argument=%v result=%#v",
			commandCandidate,
			argumentCandidate,
			result,
		)
	}
}

func TestCompleteHelpPathUsesCanonicalCommandRoute(t *testing.T) {
	t.Parallel()
	type config = completionVerboseConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.Bool(&root.Config.Verbose).Long("verbose")
			root.Subcommands.Command(cli.Command[config]{
				Name: "child",
			})
		},
	})
	clitest.Complete(t.Context(), app, cli.CompletionRequest{
		Args: []string{"help", ""},
		Cursor: cli.CompletionCursor{
			Argument: 1,
		},
	}).Require(t,
		clitest.CompletionItem("child").Present(),
		clitest.CompletionItem("--verbose").Absent(),
	)
}

func TestCompleteAdvancedMatchingModesThroughPublicCompletion(t *testing.T) {
	t.Parallel()
	type config = completionDessertConfig
	candidate := cli.CompletionCandidate{
		Value:       "CrèmeBrulee",
		DisplayText: "Crème Brûlée",
		SearchTerms: []string{"dessert", "brulee"},
		Description: "French dessert",
		Detail:      "custard with caramelized top",
	}
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Dessert).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				contribution := cli.CompletionContribution{
					Candidates: []cli.CompletionCandidate{candidate},
				}
				return contribution, nil
			}).
				Long("dessert")
		},
	})

	tests := []struct {
		name   string
		query  string
		match  cli.CompletionMatchOptions
		kind   cli.CompletionMatchKind
		field  cli.CompletionMatchField
		index  int
		edits  int
		ranges []cli.CompletionRange
	}{
		{
			name:  "smart insensitive diacritic prefix",
			query: "creme",
			match: cli.CompletionMatchOptions{
				Mode:             cli.CompletionMatchModeApproximate,
				Case:             cli.CompletionCaseSmart,
				IgnoreDiacritics: true,
			},
			kind:  cli.CompletionMatchKindPrefix,
			field: cli.CompletionMatchFieldValue,
			ranges: []cli.CompletionRange{{
				Start: 0,
				End:   6,
			}},
		},
		{
			name:  "display text fallback",
			query: "Crème ",
			match: cli.CompletionMatchOptions{
				Mode: cli.CompletionMatchModePrefix,
				Case: cli.CompletionCaseSensitive,
			},
			kind:  cli.CompletionMatchKindPrefix,
			field: cli.CompletionMatchFieldDisplayText,
			ranges: []cli.CompletionRange{{
				Start: 0,
				End:   7,
			}},
		},
		{
			name:  "description field",
			query: "French",
			match: cli.CompletionMatchOptions{
				Mode:   cli.CompletionMatchModePrefix,
				Case:   cli.CompletionCaseSensitive,
				Fields: []cli.CompletionMatchField{cli.CompletionMatchFieldDescription},
			},
			kind:  cli.CompletionMatchKindPrefix,
			field: cli.CompletionMatchFieldDescription,
			ranges: []cli.CompletionRange{{
				Start: 0,
				End:   6,
			}},
		},
		{
			name:  "detail field",
			query: "custard",
			match: cli.CompletionMatchOptions{
				Mode:   cli.CompletionMatchModePrefix,
				Case:   cli.CompletionCaseSensitive,
				Fields: []cli.CompletionMatchField{cli.CompletionMatchFieldDetail},
			},
			kind:  cli.CompletionMatchKindPrefix,
			field: cli.CompletionMatchFieldDetail,
			ranges: []cli.CompletionRange{{
				Start: 0,
				End:   7,
			}},
		},
		{
			name:  "nonzero search term index",
			query: "brul",
			match: cli.CompletionMatchOptions{
				Mode:   cli.CompletionMatchModePrefix,
				Case:   cli.CompletionCaseSensitive,
				Fields: []cli.CompletionMatchField{cli.CompletionMatchFieldSearchTerm},
			},
			kind:  cli.CompletionMatchKindPrefix,
			field: cli.CompletionMatchFieldSearchTerm,
			index: 1,
			ranges: []cli.CompletionRange{{
				Start: 0,
				End:   4,
			}},
		},
		{
			name:  "explicit field preference",
			query: "dess",
			match: cli.CompletionMatchOptions{
				Mode:   cli.CompletionMatchModeFuzzy,
				Case:   cli.CompletionCaseSensitive,
				Fields: []cli.CompletionMatchField{cli.CompletionMatchFieldSearchTerm, cli.CompletionMatchFieldValue},
			},
			kind:  cli.CompletionMatchKindPrefix,
			field: cli.CompletionMatchFieldSearchTerm,
			index: 0,
			ranges: []cli.CompletionRange{{
				Start: 0,
				End:   4,
			}},
		},
		{
			name:  "substring before fuzzy",
			query: "Brul",
			match: cli.CompletionMatchOptions{
				Mode: cli.CompletionMatchModeFuzzy,
				Case: cli.CompletionCaseSensitive,
			},
			kind:  cli.CompletionMatchKindSubstring,
			field: cli.CompletionMatchFieldValue,
			ranges: []cli.CompletionRange{{
				Start: 6,
				End:   10,
			}},
		},
		{
			name:  "fuzzy leftmost",
			query: "CBl",
			match: cli.CompletionMatchOptions{
				Mode: cli.CompletionMatchModeFuzzy,
				Case: cli.CompletionCaseSensitive,
			},
			kind:  cli.CompletionMatchKindFuzzy,
			field: cli.CompletionMatchFieldValue,
			ranges: []cli.CompletionRange{{
				Start: 0,
				End:   1,
			}, {
				Start: 6,
				End:   7,
			}, {
				Start: 9,
				End:   10,
			}},
		},
		{
			name:  "approximate transposition",
			query: "Crmèe",
			match: cli.CompletionMatchOptions{
				Mode: cli.CompletionMatchModeApproximate,
				Case: cli.CompletionCaseSensitive,
			},
			kind:  cli.CompletionMatchKindApproximate,
			field: cli.CompletionMatchFieldValue,
			edits: 1,
			ranges: []cli.CompletionRange{{
				Start: 0,
				End:   6,
			}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			item := clitest.CompletionItem("CrèmeBrulee")
			completionRequest := cli.CompletionRequest{
				Args: []string{"--dessert", test.query},
				Cursor: cli.CompletionCursor{
					Argument: 1,
					Byte:     len(test.query),
				},
				Client: cli.CompletionClient{
					Match: test.match,
					Capabilities: cli.CompletionClientCapabilities{
						MatchRanges: true,
					},
				},
			}
			clitest.Complete(t.Context(), app, completionRequest).Require(t,
				clitest.CompletionItemsExactly(item),
				item.Match(test.kind),
				item.MatchField(test.field),
				item.MatchFieldIndex(test.index),
				item.MatchEdits(test.edits),
				item.MatchRangesExactly(test.ranges...),
			)
		})
	}

	t.Run("smart uppercase is sensitive", func(t *testing.T) {
		t.Parallel()
		completionRequest := cli.CompletionRequest{
			Args: []string{"--dessert", "CREME"},
			Cursor: cli.CompletionCursor{
				Argument: 1,
				Byte:     len("CREME"),
			},
			Client: cli.CompletionClient{
				Match: cli.CompletionMatchOptions{
					Mode:             cli.CompletionMatchModePrefix,
					Case:             cli.CompletionCaseSmart,
					IgnoreDiacritics: true,
				},
			},
		}
		clitest.Complete(t.Context(), app, completionRequest).Require(t, clitest.CompletionItemsExactly())
	})
}

func TestCompleteMatchingUnfilteredRangeCapabilityAndOpaqueBytes(t *testing.T) {
	t.Parallel()
	type config = completionValueConfig
	candidate := cli.CompletionCandidate{
		Value: "alpha",
	}
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Value).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				contribution := cli.CompletionContribution{
					Candidates: []cli.CompletionCandidate{candidate},
				}
				return contribution, nil
			}).
				Long("value")
		},
	})

	for _, test := range []struct {
		name  string
		query string
		mode  cli.CompletionMatchMode
	}{
		{
			name: "empty",
			mode: cli.CompletionMatchModeApproximate,
		},
		{
			name:  "unfiltered",
			query: "zzz",
			mode:  cli.CompletionMatchModeUnfiltered,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			item := clitest.CompletionItem("alpha")
			completionRequest := cli.CompletionRequest{
				Args: []string{"--value", test.query},
				Cursor: cli.CompletionCursor{
					Argument: 1,
					Byte:     len(test.query),
				},
				Client: cli.CompletionClient{
					Match: cli.CompletionMatchOptions{
						Mode: test.mode,
						Case: cli.CompletionCaseSensitive,
					},
					Capabilities: cli.CompletionClientCapabilities{
						MatchRanges: true,
					},
				},
			}
			clitest.Complete(t.Context(), app, completionRequest).Require(t,
				clitest.CompletionItemsExactly(item),
				item.Match(cli.CompletionMatchKindUnknown),
				item.MatchField(cli.CompletionMatchFieldUnknown), item.MatchRangesExactly(), item.MatchEdits(0),
			)
		})
	}

	item := clitest.CompletionItem("alpha")
	completionRequest := cli.CompletionRequest{
		Args: []string{"--value", "al"},
		Cursor: cli.CompletionCursor{
			Argument: 1,
			Byte:     2,
		},
		Client: cli.CompletionClient{
			Match: cli.CompletionMatchOptions{
				Mode: cli.CompletionMatchModePrefix,
				Case: cli.CompletionCaseSensitive,
			},
		},
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t, clitest.CompletionItemsExactly(item), item.Match(cli.CompletionMatchKindPrefix), item.MatchRangesExactly())

	invalid := string([]byte{'x', 0xff, 'y'})
	invalidApp := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Value).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				contribution := cli.CompletionContribution{
					Candidates: []cli.CompletionCandidate{{
						Value: invalid,
					}},
				}
				return contribution, nil
			}).
				Long("value")
		},
	})
	invalidItem := clitest.CompletionItem(invalid)
	completionRequestUnderTest := cli.CompletionRequest{
		Args: []string{"--value", invalid},
		Cursor: cli.CompletionCursor{
			Argument: 1,
			Byte:     len(invalid),
		},
		Client: cli.CompletionClient{
			Match: cli.CompletionMatchOptions{
				Mode: cli.CompletionMatchModeExact,
				Case: cli.CompletionCaseInsensitive,
			},
		},
	}
	clitest.Complete(t.Context(), invalidApp, completionRequestUnderTest).Require(t, clitest.CompletionItemsExactly(invalidItem), invalidItem.Match(cli.CompletionMatchKindExact))

	other := string([]byte{'x', 0xfe, 'y'})
	completionRequestUnderTest2 := cli.CompletionRequest{
		Args: []string{"--value", other},
		Cursor: cli.CompletionCursor{
			Argument: 1,
			Byte:     len(other),
		},
		Client: cli.CompletionClient{
			Match: cli.CompletionMatchOptions{
				Mode: cli.CompletionMatchModeExact,
				Case: cli.CompletionCaseInsensitive,
			},
		},
	}
	clitest.Complete(t.Context(), invalidApp, completionRequestUnderTest2).Require(t, clitest.CompletionItemsExactly())

	completionRequestUnderTest3 := cli.CompletionRequest{
		Args: []string{"--value", "\u0301"},
		Cursor: cli.CompletionCursor{
			Argument: 1,
			Byte:     len("\u0301"),
		},
		Client: cli.CompletionClient{
			Match: cli.CompletionMatchOptions{
				Mode:             cli.CompletionMatchModeExact,
				Case:             cli.CompletionCaseInsensitive,
				IgnoreDiacritics: true,
			},
		},
	}
	clitest.Complete(t.Context(), app, completionRequestUnderTest3).Require(t, clitest.CompletionItemsExactly())
}

func TestCompletionPreservesExplicitCapabilitiesForBuiltInShellFamily(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "serve",
				Description: cli.Description{
					Short: "Serve requests",
				},
			})
		},
	})

	completionRequest := cli.CompletionRequest{
		Args: []string{""},
		Cursor: cli.CompletionCursor{
			Argument: 0,
		},
		Client: cli.CompletionClient{
			Shell: cli.CompletionShell{
				Family:  cli.ShellFamilyBash,
				Version: "5.2",
			},
			Capabilities: cli.CompletionClientCapabilities{
				Descriptions: true,
			},
		},
	}
	result, err := app.Complete(t.Context(), completionRequest)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	for _, item := range result.Items {
		if item.Value != "serve" {
			continue
		}
		if item.Description != "Serve requests" {
			t.Fatalf(
				"built-in shell family description = %q, want %q",
				item.Description,
				"Serve requests",
			)
		}
		return
	}
	t.Fatalf("serve absent from %#v", result.Items)
}

func TestCompletionRejectsShellVersionWithoutFamily(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
	})
	completionRequest := cli.CompletionRequest{
		Args: []string{""},
		Cursor: cli.CompletionCursor{
			Argument: 0,
		},
		Client: cli.CompletionClient{
			Shell: cli.CompletionShell{
				Version: "1",
			},
		},
	}
	_, err := app.Complete(t.Context(), completionRequest)
	if err == nil {
		t.Fatal("Complete() error = nil, want invalid shell client facts")
	}
}

type completionValueModePathConfig struct {
	Mode string
	Path string
}

type completionValuePathCancelConfig struct{ Path string }

type completionValueNameConfig struct{ Name string }

type completionValuePathConfig struct{ Path string }

type completionValuePairConfig struct{ Pair string }

func TestCompleteTrailingRequiredFlagValueProjectsNativeTarget(t *testing.T) {
	t.Parallel()
	type config = completionValuePathConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Path).CompleteWith(cli.CompletionNativeFiles).ValueName("path").Long("path")
		},
	})
	target := clitest.CompletionTarget(cli.CompletionTargetFlagValue)
	completionRequest := cli.CompletionRequest{
		Args: []string{"--path", ""},
		Cursor: cli.CompletionCursor{
			Argument: 1,
		},
		Client: cli.CompletionClient{
			Capabilities: cli.CompletionClientCapabilities{
				CandidateKinds: true,
				Native: cli.CompletionNativeCapabilities{
					Files: true,
				},
			},
		},
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t,
		clitest.CompletionTargetsExactly(clitest.CompletionTargetAt(0)),
		clitest.CompletionTargetAt(0).Kind(cli.CompletionTargetFlagValue),
		clitest.CompletionTargetAt(0).Name("path"),
		clitest.CompletionTargetAt(0).Spelling("--path"),
		clitest.CompletionTargetAt(0).CommandPathExactly(),
		clitest.CompletionTargetAt(0).FlagValueForm(cli.FlagValueFormSeparate),
		clitest.CompletionTargetAt(0).Occurrence(0),
		target.CommandIdentityPresent(),
		target.ValueIdentityPresent(),
		clitest.CompletionReplace(cli.CompletionSpan{
			Argument: 1,
			Start:    0,
			End:      0,
		}),
		clitest.CompletionItemsExactly(),
		clitest.CompletionNativeRequestsExactly(clitest.CompletionNativeRequestAt(0)),
		clitest.CompletionNativeRequestAt(0).Kind(cli.CompletionNativeFiles),
		clitest.CompletionNativeRequestAt(0).Mode(cli.CompletionNativeAlways),
		clitest.CompletionNativeRequestAt(0).Target().Kind(cli.CompletionTargetFlagValue),
	)
}

func TestCompleteValueCallbackSeesOnlyPriorResolvedDependencyAndComposesFallback(t *testing.T) {
	t.Parallel()
	type config = completionValueModePathConfig
	var callbackCalls int
	var observedRequest cli.ValueCompletionRequest
	definition := cli.Definition[config]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			mode := root.Flags.String(&root.Config.Mode)
			mode.Long("mode")
			root.Flags.String(
				&root.Config.Path,
			).CompleteBy(func(ctx context.Context, cfg *config, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				callbackCalls++
				observedRequest = request
				if cfg.Mode != "fast" || cfg.Path != "" {
					t.Fatalf("callback config = %#v", cfg)
				}
				if cli.EffectiveLocale(ctx) != "en" {
					t.Fatalf("EffectiveLocale() = %q", cli.EffectiveLocale(ctx))
				}
				state, ok := cli.ValueStateFromContext(ctx, mode)
				if !ok || state.State() != cli.ResolutionStateResolved {
					t.Fatalf("dependency state = %#v, ok=%v", state, ok)
				}
				contribution := cli.CompletionContribution{
					Candidates: []cli.CompletionCandidate{
						{
							Value:            "primary",
							DisplayText:      "Primary",
							Description:      "primary description",
							Detail:           "detail",
							Kind:             cli.CompletionCandidateFile,
							GroupID:          "files",
							SearchTerms:      []string{"p"},
							SortText:         "0001",
							Priority:         4,
							CommitCharacters: []rune{'/'},
						},
						{
							Value:        "fallback",
							FallbackOnly: true,
							Kind:         cli.CompletionCandidateFile,
							GroupID:      "files",
						},
					},
					Groups: []cli.CompletionGroup{{
						ID:       "files",
						Label:    "Files",
						Priority: 4,
					}},
					Messages: []cli.CompletionMessage{
						{
							Kind: cli.CompletionMessageInformation,
							Text: "choose a path",
						},
					},
				}
				return contribution, nil
			}).
				From(mode).
				FallbackTo(cli.CompletionNativeFiles).Long("path")
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatal(err)
	}

	result, err := func() (cli.Completion, error) {
		completionRequest := cli.CompletionRequest{
			Args: []string{"--mode", "fast", "--path", ""},
			Cursor: cli.CompletionCursor{
				Argument: 3,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					Descriptions:     true,
					Details:          true,
					Groups:           true,
					CandidateKinds:   true,
					CommitCharacters: true,
					Messages:         true,
					Native: cli.CompletionNativeCapabilities{
						Files: true,
					},
				},
			},
		}
		proof := clitest.Complete(t.Context(), app, completionRequest)
		proof.Require(t,
			clitest.CompletionItemsExactly(clitest.CompletionItem("primary")),
			clitest.CompletionItem("primary").Kind(cli.CompletionCandidateFile),
			clitest.CompletionItem("primary").DisplayText("Primary"),
			clitest.CompletionItem("primary").Description("primary description"),
			clitest.CompletionItem("primary").Detail("detail"),
			clitest.CompletionItem("primary").SearchTermsExactly("p"),
			clitest.CompletionItem("primary").SortText("0001"),
			clitest.CompletionItem("primary").Priority(4),
			clitest.CompletionItem("primary").CommitCharactersExactly('/'),
			clitest.CompletionItem("primary").Group(clitest.CompletionGroup("files")),
			clitest.CompletionGroupsExactly(clitest.CompletionGroup("files")),
			clitest.CompletionGroup("files").Label("Files"),
			clitest.CompletionGroup("files").Priority(4),
			clitest.CompletionGroup("files").ItemsExactly(clitest.CompletionItem("primary")),
			clitest.CompletionMessagesExactly(cli.CompletionMessage{
				Kind: cli.CompletionMessageInformation,
				Text: "choose a path",
			}),
			clitest.CompletionNativeRequestsExactly(),
		)
		return proof.Completion()
	}()
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if callbackCalls != 1 {
		t.Fatalf("callback calls = %d", callbackCalls)
	}
	if observedRequest.Target.Kind != cli.CompletionTargetFlagValue || observedRequest.Selection.Selected == 0 ||
		observedRequest.Text != "" ||
		observedRequest.Cursor != 0 ||
		observedRequest.Replace != (cli.CompletionRange{}) {
		t.Fatalf("callback request = %#v", observedRequest)
	}
	if result.Items[0].FallbackOnly {
		t.Fatalf("primary semantic match unexpectedly remained fallback-only: %#v", result.Items[0])
	}

	t.Run("callback observes operation cancellation", func(t *testing.T) {
		t.Parallel()
		type cancelConfig = completionValuePathCancelConfig
		ctx, cancel := context.WithCancel(t.Context())
		observedCancellation := false
		cancelApp, err := cli.Compile(cli.Definition[cancelConfig]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[cancelConfig]) {
				root.Flags.String(&root.Config.Path).CompleteBy(func(callbackCtx context.Context, _ *cancelConfig, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
					cancel()
					<-callbackCtx.Done()
					observedCancellation = errors.Is(callbackCtx.Err(), context.Canceled)
					return cli.CompletionContribution{}, callbackCtx.Err()
				}).
					Long("path")
			},
		})
		if err != nil {
			t.Fatalf("Compile() error = %v", err)
		}
		_, err = cancelApp.Complete(ctx, cli.CompletionRequest{
			Args: []string{"--path", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
		})
		if !observedCancellation || !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation observed=%v error=%v", observedCancellation, err)
		}
	})
}

func TestCompleteValueCallbackFailureDiscardsContributionAndReturnsPartialResult(t *testing.T) {
	t.Parallel()
	type config = completionValuePathConfig
	callbackFailure := errors.New("callback failed")
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Path).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				contribution := cli.CompletionContribution{
					Candidates: []cli.CompletionCandidate{{
						Value: "must-not-commit",
					}},
				}
				return contribution, callbackFailure
			}).
				Long("path")
		},
	})
	proof := clitest.Complete(t.Context(), app, cli.CompletionRequest{
		Args: []string{"--path", ""},
		Cursor: cli.CompletionCursor{
			Argument: 1,
		},
	})
	proof.Require(t,
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticCompletionFailure).Present()),
		clitest.CompletionPartial(),
		clitest.CompletionObservedTargetsExactly(clitest.CompletionTargetAt(0)),
		clitest.CompletionObservedItemsExactly(),
	)
	_, err := proof.Completion()
	if err == nil || !errors.Is(err, callbackFailure) {
		t.Fatalf("Complete() error = %v", err)
	}
}

func TestCompleteTokenLanguageCallbackReceivesOnlyOwnedPriorChildTokens(t *testing.T) {
	t.Parallel()
	type config = completionValuePairConfig
	language := cli.TokenLanguage[string]{
		Name:   "pair",
		Count:  cli.Count.Exactly(2),
		Parse:  func(context.Context, []string) (string, error) { return "pair", nil },
		Format: func(string) []string { return []string{"left", "right"} },
	}
	var request cli.TokenLanguageCompletionRequest
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			destination := root.Values.String(&root.Config.Pair)
			root.Flags.Tokens(destination, language).CompleteTokensBy(func(_ context.Context, _ *config, observed cli.TokenLanguageCompletionRequest) (cli.CompletionContribution, error) {
				request = observed
				contribution := cli.CompletionContribution{
					Candidates: []cli.CompletionCandidate{{
						Value: "right",
						Kind:  cli.CompletionCandidateValue,
					}},
				}
				return contribution, nil
			}).
				Long("pair")
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := func() (cli.Completion, error) {
		completionRequest := cli.CompletionRequest{
			Args: []string{"--pair", "left", ""},
			Cursor: cli.CompletionCursor{
				Argument: 2,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					CandidateKinds: true,
				},
			},
		}
		proof := clitest.Complete(t.Context(), app, completionRequest)
		proof.Require(t, clitest.CompletionItem("right").Present())
		return proof.Completion()
	}()
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if request.Target.Kind != cli.CompletionTargetTokenLanguage || len(request.Tokens) != 1 ||
		request.Tokens[0] != "left" ||
		request.Text != "" ||
		request.Cursor != 0 {
		t.Fatalf("token-language request = %#v", request)
	}
	if len(result.Targets) != 1 || result.Targets[0].Kind != cli.CompletionTargetTokenLanguage ||
		len(result.Items) != 1 ||
		result.Items[0].Value != "right" {
		t.Fatalf("result = %#v", result)
	}
}

func TestCompleteNonemptyFlagValueCallbackOwnsTextRangeAndFallbackAfterClientMatch(t *testing.T) {
	t.Parallel()
	type config = completionValuePathConfig
	var observed cli.ValueCompletionRequest
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Path).CompleteBy(func(_ context.Context, _ *config, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				observed = request
				completionRange := &cli.CompletionRange{
					Start: 1,
					End:   3,
				}
				contribution := cli.CompletionContribution{
					Replace: completionRange,
					Candidates: []cli.CompletionCandidate{
						{
							Value: "other",
							Kind:  cli.CompletionCandidateValue,
						},
						{
							Value:        "prefix-fallback",
							Kind:         cli.CompletionCandidateValue,
							FallbackOnly: true,
						},
					},
				}
				return contribution, nil
			}).FallbackTo(cli.CompletionNativeFiles).
				Long("path")
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := func() (cli.Completion, error) {
		completionRequest := cli.CompletionRequest{
			Args: []string{"--path", "pre"},
			Cursor: cli.CompletionCursor{
				Argument: 1,
				Byte:     3,
			},
			Client: cli.CompletionClient{
				Match: cli.CompletionMatchOptions{
					Mode: cli.CompletionMatchModePrefix,
					Case: cli.CompletionCaseSensitive,
				},
				Capabilities: cli.CompletionClientCapabilities{
					CandidateKinds: true,
					MatchRanges:    true,
					Native: cli.CompletionNativeCapabilities{
						Files: true,
					},
				},
			},
		}
		proof := clitest.Complete(t.Context(), app, completionRequest)
		proof.Require(t, clitest.CompletionItem("prefix-fallback").Present())
		return proof.Completion()
	}()
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if observed.Text != "pre" || observed.Cursor != 3 || observed.Replace != (cli.CompletionRange{
		Start: 0,
		End:   3,
	}) {
		t.Fatalf("callback request = %#v", observed)
	}
	if result.Replace != (cli.CompletionSpan{
		Argument: 1,
		Start:    1,
		End:      3,
	}) {
		t.Fatalf("result replace = %#v", result.Replace)
	}
	if len(result.Items) != 1 || result.Items[0].Value != "prefix-fallback" || !result.Items[0].FallbackOnly {
		t.Fatalf("items = %#v", result.Items)
	}
	if len(result.NativeRequests) != 0 {
		t.Fatalf("native fallback should be suppressed by matched semantic fallback: %#v", result.NativeRequests)
	}
}

func TestCompleteNonemptyTokenLanguageCallbackReceivesCurrentOwnedChild(t *testing.T) {
	t.Parallel()
	type config = completionValuePairConfig
	language := cli.TokenLanguage[string]{
		Name:   "pair",
		Count:  cli.Count.Exactly(2),
		Parse:  func(context.Context, []string) (string, error) { return "pair", nil },
		Format: func(string) []string { return []string{"left", "right"} },
	}
	var observed cli.TokenLanguageCompletionRequest
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			destination := root.Values.String(&root.Config.Pair)
			root.Flags.Tokens(destination, language).CompleteTokensBy(func(_ context.Context, _ *config, request cli.TokenLanguageCompletionRequest) (cli.CompletionContribution, error) {
				observed = request
				contribution := cli.CompletionContribution{
					Candidates: []cli.CompletionCandidate{
						{
							Value: "right",
							Kind:  cli.CompletionCandidateValue,
						},
						{
							Value: "left",
						},
					},
				}
				return contribution, nil
			}).
				Long("pair")
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := func() (cli.Completion, error) {
		completionRequest := cli.CompletionRequest{
			Args: []string{"--pair", "left", "ri"},
			Cursor: cli.CompletionCursor{
				Argument: 2,
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
		proof := clitest.Complete(t.Context(), app, completionRequest)
		proof.Require(t, clitest.CompletionItem("right").Present())
		return proof.Completion()
	}()
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if len(observed.Tokens) != 1 || observed.Tokens[0] != "left" || observed.Text != "ri" || observed.Cursor != 2 ||
		observed.Replace != (cli.CompletionRange{
			Start: 0,
			End:   2,
		}) {
		t.Fatalf("token-language request = %#v", observed)
	}
	if len(result.Items) != 1 || result.Items[0].Value != "right" {
		t.Fatalf("items = %#v", result.Items)
	}
}

func TestCompletePositionalValueUsesAuthoritativeArgumentOwner(t *testing.T) {
	t.Parallel()
	type config = completionValueNameConfig
	var observed cli.ValueCompletionRequest
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.String(
					&form.Config.Name,
					"name",
				).CompleteBy(
					func(_ context.Context, _ *config, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
						observed = request
						contribution := cli.CompletionContribution{
							Candidates: []cli.CompletionCandidate{
								{
									Value: "alpha",
									Kind:  cli.CompletionCandidateValue,
								},
							},
						}
						return contribution, nil
					},
				)
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := func() (cli.Completion, error) {
		completionRequest := cli.CompletionRequest{
			Args: []string{"al"},
			Cursor: cli.CompletionCursor{
				Argument: 0,
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
		proof := clitest.Complete(t.Context(), app, completionRequest)
		proof.Require(t, clitest.CompletionTarget(cli.CompletionTargetArgument).Present())
		return proof.Completion()
	}()
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if observed.Target.Kind != cli.CompletionTargetArgument || observed.Target.Name != "name" ||
		observed.Text != "al" ||
		observed.Cursor != 2 {
		t.Fatalf("request = %#v", observed)
	}
	found := false
	for _, item := range result.Items {
		if item.Value == "alpha" {
			found = true
			if len(item.TargetIndexes) != 1 ||
				result.Targets[item.TargetIndexes[0]].Kind != cli.CompletionTargetArgument {
				t.Fatalf("argument item target = %#v targets=%#v", item, result.Targets)
			}
		}
	}
	if !found {
		t.Fatalf("positional candidate missing: %#v", result)
	}
}

func TestCompletePositionalTokenLanguageReceivesOnlyTermOwnedPriorTokens(t *testing.T) {
	t.Parallel()
	type config = completionValuePairConfig
	language := cli.TokenLanguage[string]{
		Name:   "pair",
		Count:  cli.Count.Exactly(2),
		Parse:  func(context.Context, []string) (string, error) { return "pair", nil },
		Format: func(string) []string { return []string{"left", "right"} },
	}
	var observed cli.TokenLanguageCompletionRequest
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			destination := root.Values.String(&root.Config.Pair)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Tokens(
					destination,
					"pair",
					language,
				).CompleteTokensBy(
					func(_ context.Context, _ *config, request cli.TokenLanguageCompletionRequest) (cli.CompletionContribution, error) {
						observed = request
						contribution := cli.CompletionContribution{
							Candidates: []cli.CompletionCandidate{{
								Value: "right",
							}},
						}
						return contribution, nil
					},
				)
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := func() (cli.Completion, error) {
		completionRequest := cli.CompletionRequest{
			Args: []string{"left", "ri"},
			Cursor: cli.CompletionCursor{
				Argument: 1,
				Byte:     2,
			},
			Client: cli.CompletionClient{
				Match: cli.CompletionMatchOptions{
					Mode: cli.CompletionMatchModePrefix,
					Case: cli.CompletionCaseSensitive,
				},
			},
		}
		proof := clitest.Complete(t.Context(), app, completionRequest)
		proof.Require(t, clitest.CompletionItem("right").Present())
		return proof.Completion()
	}()
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if observed.Target.Kind != cli.CompletionTargetTokenLanguage || len(observed.Tokens) != 1 ||
		observed.Tokens[0] != "left" ||
		observed.Text != "ri" ||
		observed.Cursor != 2 {
		t.Fatalf("request = %#v", observed)
	}
	if len(result.Items) != 1 || result.Items[0].Value != "right" {
		t.Fatalf("result = %#v", result)
	}
}

func TestCompletionCarriesGrammarDiagnosticsAndTruncation(t *testing.T) {
	t.Parallel()

	type config struct {
		Output  string
		Verbose bool
	}
	declare := func(root *cli.CommandBuilder[config]) {
		root.Flags.String(&root.Config.Output).
			Long("output").
			ValueForms(cli.FlagValueFormSeparate)
		root.Flags.Bool(&root.Config.Verbose).Long("verbose")
	}
	definition := cli.Definition[config]{
		Name: "alienctl",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: declare,
	}
	app := clitest.Compile(t, definition)
	request := cli.CompletionRequest{
		Args: []string{"--output", "--verbose", ""},
		Cursor: cli.CompletionCursor{
			Argument: 2,
		},
		Limits: cli.AnalysisLimits{
			Diagnostics: 1,
		},
	}

	proof := clitest.Complete(t.Context(), app, request)
	completion, err := proof.Completion()
	if err == nil {
		t.Fatal("Complete() error = nil, want missing-value failure")
	}
	clitest.Diagnostics(err).Require(t,
		clitest.Diagnostic(cli.DiagnosticMissingValue).Present(),
		clitest.DiagnosticsTruncated(),
	)
	if len(completion.Diagnostics) != 1 || completion.Diagnostics[0].Code != cli.DiagnosticMissingValue {
		t.Fatalf("Completion.Diagnostics = %#v", completion.Diagnostics)
	}
	diagnostic := completion.Diagnostics[0]
	for _, origin := range diagnostic.Origins {
		if _, ok := completion.Origins[origin]; !ok {
			t.Fatalf("Completion Diagnostic origin %v is absent from detached origin graph", origin)
		}
	}
	if diagnostic.Span != nil {
		if _, ok := completion.Sources[diagnostic.Span.Source]; !ok {
			t.Fatalf("Completion Diagnostic source %v is absent from detached source graph", diagnostic.Span.Source)
		}
	}
	if !completion.DiagnosticsTruncated {
		t.Fatal("Completion.DiagnosticsTruncated = false, want true")
	}
	if !completion.Partial {
		t.Fatal("Completion.Partial = false, want true")
	}
	if completion.Truncated {
		t.Fatal("Completion.Truncated = true, diagnostic truncation must remain independent")
	}
}
