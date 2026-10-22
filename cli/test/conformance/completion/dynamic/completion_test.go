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

func TestCompleteStaticFiniteDomainDoesNotDemandInvocationDefaults(t *testing.T) {
	t.Parallel()
	type config struct{ Mode string }

	compile := func(t *testing.T, positional bool) cli.Application {
		t.Helper()
		app, err := cli.Compile(cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				domain := cli.Value.Domain(cli.Value.Assign[string](), func(builder *cli.ValueDomainBuilder[string]) {
					builder.Value("prod", "production")
				})
				mode := root.Values.Value(&root.Config.Mode, domain)
				if positional {
					root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
						form.ValueFor(mode, "mode")
					})
					return
				}
				root.Flags.ValueFor(mode).Long("mode")
			},
		})
		if err != nil {
			t.Fatalf("Compile() error = %v", err)
		}
		return app
	}

	for _, test := range []struct {
		name       string
		positional bool
		args       []string
		cursor     int
	}{
		{
			name:       "positional",
			positional: true,
			args:       []string{""},
			cursor:     0,
		},
		{
			name:   "flag",
			args:   []string{"--mode", ""},
			cursor: 1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			cause := errors.New("unrelated default failure")
			completionRequest := cli.CompletionRequest{
				Args: test.args,
				Cursor: cli.CompletionCursor{
					Argument: test.cursor,
				},
				Defaults: []cli.DefaultProvider{
					cli.DefaultsFor[config](func(context.Context, *cli.DefaultSupply[config]) error {
						calls++
						return cause
					}),
				},
			}
			result, err := compile(t, test.positional).Complete(t.Context(), completionRequest)
			if err != nil {
				t.Fatalf("Complete() error = %v", err)
			}
			if calls != 0 {
				t.Fatalf("DefaultProvider calls = %d, want 0", calls)
			}
			if result.Partial {
				t.Fatal("Completion.Partial = true, want false")
			}
			if !completionDemandContainsValue(result, "prod") {
				t.Fatalf("Completion.Items = %#v, want prod", result.Items)
			}
		})
	}
}

func TestCompletePositionalDynamicDomainStillDemandsRequiredInvocationDefault(t *testing.T) {
	t.Parallel()
	type config struct {
		Seed string
		Mode string
	}

	var seed cli.Destination[config, string]
	calls := 0
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			seed = root.Values.String(&root.Config.Seed)
			domain := root.Values.Domain(
				func(_ context.Context, _ *config, dependencies cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
					value, ok := dependencies.Get(seed)
					if !ok {
						return errors.New("required seed unavailable")
					}
					builder.Value(value, value)
					return nil
				},
				seed,
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.ValueFor(mode, "mode")
			})
		},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	completionRequest := cli.CompletionRequest{
		Args: []string{""},
		Cursor: cli.CompletionCursor{
			Argument: 0,
		},
		Defaults: []cli.DefaultProvider{
			cli.DefaultsFor[config](func(_ context.Context, supply *cli.DefaultSupply[config]) error {
				calls++
				supply.Set(seed, "prod")
				return nil
			}),
		},
	}
	result, err := app.Complete(t.Context(), completionRequest)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("DefaultProvider calls = %d, want 1", calls)
	}
	if !completionDemandContainsValue(result, "prod") {
		t.Fatalf("Completion.Items = %#v, want prod", result.Items)
	}
}

func TestCompletePositionalCallbackStillDemandsRequiredInvocationDefault(t *testing.T) {
	t.Parallel()
	type config struct {
		Seed  string
		Query string
	}

	var seed cli.Destination[config, string]
	calls := 0
	observed := ""
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			seed = root.Values.String(&root.Config.Seed)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.String(
					&form.Config.Query,
					"query",
				).CompleteBy(
					func(_ context.Context, cfg *config, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
						observed = cfg.Seed
						contribution := cli.CompletionContribution{
							Candidates: []cli.CompletionCandidate{{
								Value: cfg.Seed,
							}},
						}
						return contribution, nil
					},
				).From(seed)
			})
		},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	completionRequest := cli.CompletionRequest{
		Args: []string{""},
		Cursor: cli.CompletionCursor{
			Argument: 0,
		},
		Defaults: []cli.DefaultProvider{
			cli.DefaultsFor[config](func(_ context.Context, supply *cli.DefaultSupply[config]) error {
				calls++
				supply.Set(seed, "provided")
				return nil
			}),
		},
	}
	result, err := app.Complete(t.Context(), completionRequest)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("DefaultProvider calls = %d, want 1", calls)
	}
	if observed != "provided" {
		t.Fatalf("callback observed seed = %q, want provided", observed)
	}
	if !completionDemandContainsValue(result, "provided") {
		t.Fatalf("Completion.Items = %#v, want provided", result.Items)
	}
}

func TestCompletePositionalStaticDomainStillDemandsStructuralVisibilityInputs(t *testing.T) {
	t.Parallel()
	type config struct {
		Gate  string
		Mode  string
		Extra bool
	}

	var gate cli.Destination[config, string]
	calls := 0
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			gate = root.Values.String(&root.Config.Gate)
			extra := root.Flags.Bool(&root.Config.Extra)
			extra.Long("extra")
			extra.VisibleWhen(root.When.Equal(gate, "yes"))

			domain := cli.Value.Domain(cli.Value.Assign[string](), func(builder *cli.ValueDomainBuilder[string]) {
				builder.Value("prod", "production")
			})
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.ValueFor(mode, "mode")
			})
		},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	completionRequest := cli.CompletionRequest{
		Args: []string{""},
		Cursor: cli.CompletionCursor{
			Argument: 0,
		},
		Defaults: []cli.DefaultProvider{
			cli.DefaultsFor[config](func(_ context.Context, supply *cli.DefaultSupply[config]) error {
				calls++
				supply.Set(gate, "yes")
				return nil
			}),
		},
	}
	result, err := app.Complete(t.Context(), completionRequest)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("DefaultProvider calls = %d, want 1", calls)
	}
	if !completionDemandContainsValue(result, "prod") {
		t.Fatalf("Completion.Items = %#v, want prod", result.Items)
	}
}

func completionDemandContainsValue(result cli.Completion, value string) bool {
	for _, item := range result.Items {
		if item.Value == value {
			return true
		}
	}
	return false
}

type completionDomainModeConfig struct{ Mode string }

type completionDomainModeOtherConfig struct {
	Mode  string
	Other string
}

func TestCompletionSettlesTargetDynamicDomain(t *testing.T) {
	t.Parallel()
	type config = completionDomainModeConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(_ context.Context, _ *config, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
					builder.Value("prod", "production")
					return nil
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Flags.ValueFor(mode).Long("mode")
		},
	}
	app := clitest.Compile(t, definition)

	t.Run("target domain with all invocable visibility", func(t *testing.T) {
		t.Parallel()
		completionRequest := cli.CompletionRequest{
			Args: []string{"--mode", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
			Visibility: cli.CompletionVisibilityAllInvocable,
		}
		proof := clitest.Complete(t.Context(), app, completionRequest)
		proof.Require(t, clitest.CompletionItemAt(0).Value("prod"))
	})

	t.Run("reference projection failure", func(t *testing.T) {
		t.Parallel()
		completionRequest := cli.CompletionRequest{
			Args: []string{"--mode", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
			Locale: "not_a_locale",
		}
		if _, err := app.Complete(t.Context(), completionRequest); err == nil {
			t.Fatal("Complete() accepted invalid locale after domain materialization")
		}
	})
}

func TestCompletionProjectsOnlyTargetDomainCandidates(t *testing.T) {
	t.Parallel()
	type config = completionDomainModeOtherConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			modeDomain := cli.Value.Domain(cli.Value.Assign[string](), func(builder *cli.ValueDomainBuilder[string]) {
				builder.Value("prod", "production")
			})
			otherDomain := cli.Value.Domain(cli.Value.Assign[string](), func(builder *cli.ValueDomainBuilder[string]) {
				builder.Value("other", "other")
			})
			mode := root.Values.Value(&root.Config.Mode, modeDomain)
			root.Values.Value(&root.Config.Other, otherDomain)
			root.Flags.ValueFor(mode).Long("mode")
		},
	})
	proof := clitest.Complete(t.Context(), app, cli.CompletionRequest{
		Args: []string{"--mode", ""},
		Cursor: cli.CompletionCursor{
			Argument: 1,
		},
	})
	proof.Require(t,
		clitest.CompletionItemAt(0).Value("prod"),
		clitest.CompletionItemAt(1).Absent(),
	)
}

type completionDynamicDynamicPeerConfig struct {
	Dynamic string
	Peer    string
}

type completionDynamicModeConfig struct{ Mode string }

type completionDynamicConfig struct{}

func TestCompleteFiniteDomainsUseOperationLocalAuthoritativeSnapshot(t *testing.T) {
	t.Parallel()
	t.Run("static", func(t *testing.T) {
		t.Parallel()
		type config = completionDynamicModeConfig
		app, err := cli.Compile(cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				domain := cli.Value.Domain(cli.Value.Assign[string](), func(builder *cli.ValueDomainBuilder[string]) {
					canonical := builder.Value("prod", "production")
					builder.Alias("p", canonical)
				})
				mode := root.Values.Value(&root.Config.Mode, domain)
				root.Flags.ValueFor(mode).Long("mode")
			},
		})
		if err != nil {
			t.Fatalf("Compile() error = %v", err)
		}
		result, err := func() (cli.Completion, error) {
			completionRequest := cli.CompletionRequest{
				Args: []string{"--mode", ""},
				Cursor: cli.CompletionCursor{
					Argument: 1,
				},
				Client: cli.CompletionClient{
					Capabilities: cli.CompletionClientCapabilities{
						CandidateKinds: true,
					},
				},
			}
			proof := clitest.Complete(t.Context(), app, completionRequest)
			proof.Require(t, clitest.CompletionItemAt(0).Present())
			return proof.Completion()
		}()
		if err != nil {
			t.Fatalf("Complete() error = %v", err)
		}
		completionAssertCompletionValues(t, result.Items, "prod", "p")
	})

	t.Run("dynamic once per operation", func(t *testing.T) {
		t.Parallel()
		type config = completionDynamicModeConfig
		calls := 0
		app, err := cli.Compile(cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				domain := root.Values.Domain(
					func(_ context.Context, _ *config, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
						calls++
						canonical := builder.Value("prod", "production")
						builder.Alias("p", canonical)
						return nil
					},
				)
				mode := root.Values.Value(&root.Config.Mode, domain)
				root.Flags.ValueFor(mode).Long("mode")
			},
		})
		if err != nil {
			t.Fatalf("Compile() error = %v", err)
		}
		completionRequest := cli.CompletionRequest{
			Args: []string{"--mode", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					CandidateKinds: true,
				},
			},
		}
		result, err := app.Complete(t.Context(), completionRequest)
		if err != nil {
			t.Fatalf("Complete() error = %v", err)
		}
		if calls != 1 {
			t.Fatalf("dynamic domain calls = %d, want 1", calls)
		}
		completionAssertCompletionValues(t, result.Items, "prod", "p")
	})
}

func TestCompleteDynamicDomainFailurePreservesValidPeerAsPartial(t *testing.T) {
	t.Parallel()
	type config = completionDynamicDynamicPeerConfig
	cause := errors.New("dynamic domain failed")
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(context.Context, *config, cli.Dependencies, *cli.ValueDomainBuilder[string]) error {
					return cause
				},
			)
			dynamic := root.Values.Value(&root.Config.Dynamic, domain)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.ValueFor(dynamic, "dynamic")
			})
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.String(
					&form.Config.Peer,
					"peer",
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
	if err != nil {
		t.Fatal(err)
	}
	proof := clitest.Complete(t.Context(), app, cli.CompletionRequest{
		Args: []string{""},
		Cursor: cli.CompletionCursor{
			Argument: 0,
		},
	})
	proof.Require(t,
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticValueDomainProvisionFailure).Present()),
		clitest.CompletionPartial(),
		clitest.CompletionItem("retained").Present(),
	)
	_, err = proof.Completion()
	if err == nil || !errors.Is(err, cause) {
		t.Fatalf("Complete() error = %v, want dynamic-domain cause", err)
	}
}

func TestCompletePreservesDynamicDomainFailureAndIndependentCallerCancellation(t *testing.T) {
	t.Parallel()
	type config = completionDynamicModeConfig
	domainFailure := errors.New("dynamic domain failure")
	callerCancellation := errors.New("caller cancellation")
	started := make(chan struct{})
	release := make(chan struct{})
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(context.Context, *config, cli.Dependencies, *cli.ValueDomainBuilder[string]) error {
					close(started)
					<-release
					return domainFailure
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Flags.ValueFor(mode).Long("mode")
		},
	})
	ctx, cancel := context.WithCancelCause(t.Context())
	result := make(chan error, 1)
	go func() {
		completionRequest := cli.CompletionRequest{
			Args: []string{"--mode", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					CandidateKinds: true,
				},
			},
		}
		_, err := app.Complete(ctx, completionRequest)
		result <- err
	}()
	<-started
	cancel(callerCancellation)
	<-ctx.Done()
	close(release)
	err := <-result
	if !errors.Is(err, domainFailure) {
		t.Fatalf("Complete() error = %v, want dynamic-domain failure", err)
	}
	if !errors.Is(err, callerCancellation) {
		t.Fatalf("Complete() error = %v, want independent caller cancellation", err)
	}
	clitest.Diagnostics(err).Require(t,
		clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticValueDomainProvisionFailure)),
	)
}

func TestCompleteEarlierResponseSelectsProviderChildPositionalDynamicDomainOnce(t *testing.T) {
	t.Parallel()
	type config = completionDynamicModeConfig
	providerCalls := 0
	domainCalls := 0
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				providerCalls++
				batch.Command(cli.Command[config]{
					Name: "remote",
					Declare: func(child *cli.CommandBuilder[config]) {
						domain := child.Values.Domain(
							func(_ context.Context, _ *config, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
								domainCalls++
								builder.Value("prod", "production")
								return nil
							},
						)
						mode := child.Values.Value(&child.Config.Mode, domain)
						child.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
							form.ValueFor(mode, "mode")
						})
					},
				})
				return nil
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	completionRequest := cli.CompletionRequest{
		Args: []string{"@select", "pr"},
		Cursor: cli.CompletionCursor{
			Argument: 1,
			Byte:     2,
		},
		ResponseFiles: cli.ResponseFileAccess{
			Load: func(_ context.Context, request cli.ResponseFileRequest) (cli.ResponseFile, error) {
				if request.Path != "select" {
					return cli.ResponseFile{}, errors.New("unexpected response path")
				}
				file := cli.ResponseFile{
					Key:      "select",
					Revision: cli.Revision("r1"),
					Content:  []byte("remote"),
				}
				return file, nil
			},
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
	proof.Require(t,
		clitest.CompletionTargetsExactly(clitest.CompletionTargetAt(0)),
		clitest.CompletionTargetAt(0).Kind(cli.CompletionTargetArgument),
		clitest.CompletionTarget(cli.CompletionTargetArgument).OnSelectedCommand(),
		clitest.CompletionTargetAt(0).On(clitest.Command("remote")).Present(),
		clitest.CompletionTargetAt(0).CommandPathExactly("remote"),
		clitest.CompletionTargetAt(0).Name("mode"),
		clitest.CompletionItemsExactly(clitest.CompletionItem("prod")),
		clitest.CompletionItem("prod").Kind(cli.CompletionCandidateValue),
		clitest.CompletionItem("prod").TargetsExactly(clitest.CompletionTargetAt(0)),
	)
	if providerCalls != 1 || domainCalls != 1 {
		t.Fatalf("provider/domain calls = %d/%d, want 1/1", providerCalls, domainCalls)
	}
}

func TestCompletionDemandsCurrentProviderNamespaceAndReturnsDynamicCommandCandidates(t *testing.T) {
	t.Parallel()
	type config = completionDynamicConfig
	calls := 0
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				calls++
				batch.Command(cli.Command[config]{
					Name: "remote",
					Description: cli.Description{
						Short: "remote command",
					},
				})
				return nil
			})
		},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	completionRequest := cli.CompletionRequest{
		Args: []string{""},
		Cursor: cli.CompletionCursor{
			Argument: 0,
		},
		Client: cli.CompletionClient{
			Capabilities: cli.CompletionClientCapabilities{
				Descriptions:   true,
				CandidateKinds: true,
			},
		},
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t,
		clitest.CompletionItem("remote").Present(),
		clitest.CompletionItem("remote").Kind(cli.CompletionCandidateCommand),
		clitest.CompletionItem("remote").DisplayText("remote"),
		clitest.CompletionItem("remote").Description("remote command"),
		clitest.CompletionItem("remote").DeclarationKind(cli.DeclarationKindCommandEntry),
	)
	if calls != 1 {
		t.Fatalf("provider calls = %d, want 1", calls)
	}
}

func TestCompletionProviderFailurePreservesAcceptedPeerAsDetachedPartialOperation(t *testing.T) {
	t.Parallel()
	type config = completionDynamicConfig
	cause := errors.New("provider unavailable")
	acceptedCalls := 0
	failedCalls := 0
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				acceptedCalls++
				batch.Command(cli.Command[config]{
					Name: "remote",
					Description: cli.Description{
						Short: "remote command",
					},
				})
				return nil
			})
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				failedCalls++
				batch.Command(cli.Command[config]{
					Name: "discarded",
				})
				return cause
			})
		},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	invoke := func() (cli.Completion, error) {
		return func() (cli.Completion, error) {
			completionRequest := cli.CompletionRequest{
				Args: []string{""},
				Cursor: cli.CompletionCursor{
					Argument: 0,
				},
				Client: cli.CompletionClient{
					Capabilities: cli.CompletionClientCapabilities{
						Descriptions:   true,
						CandidateKinds: true,
					},
				},
			}
			proof := clitest.Complete(t.Context(), app, completionRequest)
			proof.Require(
				t,
				clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticCommandProviderFailure).Present()),
				clitest.CompletionPartial(),
			)
			return proof.Completion()
		}()
	}
	check := func(label string, result cli.Completion, err error, wantCalls int) {
		t.Helper()
		if err == nil || !errors.Is(err, cause) {
			t.Fatalf("%s Complete() error = %v, want provider cause", label, err)
		}
		if acceptedCalls != wantCalls || failedCalls != wantCalls {
			t.Fatalf(
				"%s provider calls = accepted %d failed %d, want %d each",
				label,
				acceptedCalls,
				failedCalls,
				wantCalls,
			)
		}
		if !result.Partial {
			t.Fatalf("%s completion must be partial: %#v", label, result)
		}
		remote := false
		discarded := false
		for _, item := range result.Items {
			switch item.Value {
			case "remote":
				remote = true
				if item.Kind != cli.CompletionCandidateCommand ||
					item.Declaration.Kind != cli.DeclarationKindCommandEntry ||
					item.Declaration.CommandEntry == 0 {
					t.Fatalf("%s accepted provider completion = %#v", label, item)
				}
			case "discarded":
				discarded = true
			}
		}
		if !remote || discarded {
			t.Fatalf(
				"%s provider completion containment: remote=%v discarded=%v items=%#v",
				label,
				remote,
				discarded,
				result.Items,
			)
		}
		if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != cli.DiagnosticCommandProviderFailure {
			t.Fatalf("%s provider diagnostics = %#v", label, result.Diagnostics)
		}
	}

	first, firstErr := invoke()
	check("first", first, firstErr, 1)
	first.Items[0].Value = "mutated"
	first.Diagnostics[0].Code = cli.DiagnosticCode("mutated")

	second, secondErr := invoke()
	check("second", second, secondErr, 2)
}

type completionResponseFileConfig struct{}

type completionResponseFileValueConfig struct{ Value string }

func TestCompleteEarlierResponseLoadFailurePreservesUnaffectedPartialObservation(t *testing.T) {
	t.Parallel()
	type config = completionResponseFileConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
			root.Subcommands.Command(cli.Command[config]{
				Name: "child",
			})
		},
	})
	cause := errors.New("response load failed")
	completions := 0
	completionRequest := cli.CompletionRequest{
		Args: []string{"@broken", ""},
		Cursor: cli.CompletionCursor{
			Argument: 1,
		},
		ResponseFiles: cli.ResponseFileAccess{
			Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
				return cli.ResponseFile{}, cause
			},
			Complete: func(context.Context, cli.ResponseFileCompletionRequest) (cli.CompletionContribution, error) {
				completions++
				return cli.CompletionContribution{}, nil
			},
		},
	}
	proof := clitest.Complete(t.Context(), app, completionRequest)
	proof.Require(t,
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticResponseFileLoadFailure).Present()),
		clitest.CompletionPartial(),
		clitest.CompletionReplace(cli.CompletionSpan{
			Argument: 1,
		}),
	)
	_, err := proof.Completion()
	if err == nil || !errors.Is(err, cause) {
		t.Fatalf("Complete() error = %v, want response load cause", err)
	}
	if completions != 0 {
		t.Fatalf("current response completer calls = %d, want 0 after earlier load failure", completions)
	}
}

func TestCompleteEarlierResponseReferencesShareOperationCumulativeBudget(t *testing.T) {
	t.Parallel()
	type config = completionResponseFileValueConfig
	producerCalls := 0
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.String(
					&form.Config.Value,
					"value",
				).CompleteBy(
					func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
						producerCalls++
						contribution := cli.CompletionContribution{
							Candidates: []cli.CompletionCandidate{{
								Value: "must-not-run",
							}},
						}
						return contribution, nil
					},
				)
			})
		},
	})
	completionRequest := cli.CompletionRequest{
		Args: []string{"@first", "@second", ""},
		Cursor: cli.CompletionCursor{
			Argument: 2,
		},
		Limits: cli.AnalysisLimits{
			ResponseFiles: cli.ResponseFileLimits{
				TotalSourceBytes: 1,
			},
		},
		ResponseFiles: cli.ResponseFileAccess{
			Load: func(_ context.Context, request cli.ResponseFileRequest) (cli.ResponseFile, error) {
				switch request.Path {
				case "first":
					file := cli.ResponseFile{
						Key:      "first",
						Revision: cli.Revision("r1"),
						Content:  []byte(" "),
					}
					return file, nil
				case "second":
					file := cli.ResponseFile{
						Key:      "second",
						Revision: cli.Revision("r1"),
						Content:  []byte(" "),
					}
					return file, nil
				default:
					return cli.ResponseFile{}, errors.New("unexpected response path")
				}
			},
		},
	}
	proof := clitest.Complete(t.Context(), app, completionRequest)
	firstSource := clitest.InputSourceByKey(cli.InputKindResponseFile, "first")
	proof.Require(t,
		clitest.FailsWith(
			clitest.Diagnostic(cli.DiagnosticResponseFileLimitExceeded).Present(),
			clitest.Diagnostic(cli.DiagnosticResponseFileLimitExceeded).Reason("total-source-bytes"),
		),
		clitest.CompletionPartial(),
		clitest.CompletionObservedItemsExactly(),
		clitest.CompletionObservedInputSourcesExactly(cli.InputKindResponseFile, firstSource),
	)
	_, err := proof.Completion()
	if err == nil {
		t.Fatal("Complete() accepted cumulative response-file budget breach")
	}
	if producerCalls != 0 {
		t.Fatalf("cursor producer calls = %d, want 0", producerCalls)
	}
}

type completionStaticClassificationVerboseConfig struct {
	Verbose bool
}

func TestStaticCompletionGroupClassificationIsStableAcrossLocalizedHelpTitles(t *testing.T) {
	t.Parallel()

	type config = completionStaticClassificationVerboseConfig

	var helpPage cli.HelpPage
	definition := cli.Definition[config]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{
				{
					Locale: "fr",
					Translations: cli.Translations{
						"Options": "Options FR",
					},
				},
			},
		},
		Help: cli.HelpConfig{
			Renderer: func(_ context.Context, page cli.HelpPage) (string, error) {
				helpPage = page
				return "help", nil
			},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.HelpGroup("Options", func(flags *cli.Flags[config]) {
				flags.Bool(&root.Config.Verbose).Long("verbose")
			})
		},
	}
	app := clitest.Compile(t, definition)

	var output strings.Builder
	runRequest := cli.RunRequest{
		Args:   []string{"--help"},
		Locale: "fr",
		Stdout: &output,
	}
	err := app.Run(t.Context(), runRequest)
	if err != nil {
		t.Fatalf("Run(--help) error = %v", err)
	}
	if len(helpPage.FlagGroups) == 0 {
		t.Fatal("localized HelpPage has no flag groups")
	}

	var declared cli.HelpFlagGroup
	foundDeclared := false
	for _, group := range helpPage.FlagGroups {
		if group.Kind == cli.HelpGroupDeclared {
			declared = group
			foundDeclared = true
			break
		}
	}
	if !foundDeclared {
		t.Fatalf("HelpPage flag groups = %#v, want declared group", helpPage.FlagGroups)
	}
	if declared.Title != "Options FR" || declared.Classification.SourceTitle != "Options" {
		t.Fatalf(
			"declared Help group = title %q classification %#v",
			declared.Title,
			declared.Classification,
		)
	}

	complete := func(locale cli.Locale) cli.Completion {
		completionRequest := cli.CompletionRequest{
			Args: []string{""},
			Cursor: cli.CompletionCursor{
				Argument: 0,
			},
			Locale: locale,
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					Groups: true,
				},
			},
		}
		result, err := app.Complete(t.Context(), completionRequest)
		if err != nil {
			t.Fatalf("Complete(%q) error = %v", locale, err)
		}
		return result
	}

	groupID := func(result cli.Completion) string {
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

	enID := groupID(complete("en"))
	fr := complete("fr")
	frID := groupID(fr)
	if enID != frID {
		t.Fatalf(
			"static Completion group ID changed with locale: en %q, fr %q",
			enID,
			frID,
		)
	}
	if frID == declared.Title || frID == declared.Classification.SourceTitle {
		t.Fatalf("Completion group ID %q reused Help display/source title directly", frID)
	}
	for _, group := range fr.Groups {
		if group.ID != frID {
			continue
		}
		if group.Label != "" {
			t.Fatalf(
				"static Completion group label = %q, want non-display classification",
				group.Label,
			)
		}
		return
	}
	t.Fatalf("Completion groups = %#v, want static group %q", fr.Groups, frID)
}

func completionAssertCompletionValues(t *testing.T, items []cli.CompletionItem, values ...string) {
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
