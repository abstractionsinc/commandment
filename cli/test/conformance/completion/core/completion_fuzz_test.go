package conformance

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func completionFuzzComplete(t *testing.T, app cli.Application, request cli.CompletionRequest) (cli.Completion, error) {
	t.Helper()
	return clitest.Complete(t.Context(), app, request).Completion()
}

var _ = errors.Is

type completionFuzzRootConfig struct{}

type completionFuzzModeNameChildConfig struct {
	Mode string
	Name string
}

type completionFuzzPathPairRestConfig struct {
	Path string
	Pair string
	Rest []string
}

func FuzzCompleteShortcutResidualMatchesCanonicalTarget(f *testing.F) {
	type rootConfig = completionFuzzRootConfig
	type childConfig = completionFuzzModeNameChildConfig
	app, err := cli.Compile(cli.Definition[rootConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[childConfig]{
				Name: "serve",
				Declare: func(child *cli.CommandBuilder[childConfig]) {
					mode := child.Flags.String(&child.Config.Mode)
					mode.Long("mode")
					domain := cli.Value.Domain(cli.Value.Assign[string](), func(builder *cli.ValueDomainBuilder[string]) {
						builder.Value("alpha", "alpha")
						builder.Value("beta", "beta")
					})
					name := child.Values.Value(&child.Config.Name, domain)
					child.Arguments.Form(
						func(form *cli.ArgumentFormBuilder[childConfig]) { form.ValueFor(name, "name") },
					)
					child.Shortcut(cli.CommandShortcut{
						Name: "quick",
						Args: []string{"--mode", "fixed"},
					})
				},
			})
		},
	})
	if err != nil {
		f.Fatal(err)
	}
	for _, seed := range []string{"", "a", "al", "b", "z"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		if len(text) > 32 {
			return
		}
		for _, r := range text {
			if r < 0x20 || r > 0x7e {
				return
			}
		}

		client := cli.CompletionClient{
			Match: cli.CompletionMatchOptions{
				Mode: cli.CompletionMatchModePrefix,
				Case: cli.CompletionCaseSensitive,
			},
			Capabilities: cli.CompletionClientCapabilities{
				CandidateKinds: true,
			},
		}

		shortcut, shortcutErr := completionFuzzComplete(t, app, cli.CompletionRequest{
			Args: []string{"quick", text},
			Cursor: cli.CompletionCursor{
				Argument: 1,
				Byte:     len(text),
			},
			Client: client,
		})
		canonical, canonicalErr := completionFuzzComplete(t, app, cli.CompletionRequest{
			Args: []string{"serve", "--mode", "fixed", text},
			Cursor: cli.CompletionCursor{
				Argument: 3,
				Byte:     len(text),
			},
			Client: client,
		})
		if (shortcutErr == nil) != (canonicalErr == nil) {
			t.Fatalf("shortcut/canonical error parity = %v/%v", shortcutErr, canonicalErr)
		}
		if shortcutErr != nil {
			return
		}
		if shortcut.Selection.Selected != canonical.Selection.Selected ||
			!slices.Equal(shortcut.Selection.Path, canonical.Selection.Path) {
			t.Fatalf("selection mismatch shortcut=%#v canonical=%#v", shortcut.Selection, canonical.Selection)
		}
		var shortcutValues, canonicalValues []string
		for _, item := range shortcut.Items {
			if item.Kind == cli.CompletionCandidateValue {
				shortcutValues = append(shortcutValues, item.Value)
			}
		}
		for _, item := range canonical.Items {
			if item.Kind == cli.CompletionCandidateValue {
				canonicalValues = append(canonicalValues, item.Value)
			}
		}
		if !slices.Equal(shortcutValues, canonicalValues) {
			t.Fatalf("value candidates mismatch shortcut=%#v canonical=%#v", shortcutValues, canonicalValues)
		}
	})
}

func FuzzCompletionSemanticProperties(f *testing.F) {
	type config = completionFuzzPathPairRestConfig
	var valueRequest cli.ValueCompletionRequest
	var tokenRequest cli.TokenLanguageCompletionRequest
	var delegationRequest cli.CompletionDelegationRequest
	var responseRequest cli.ResponseFileCompletionRequest
	loads := 0
	language := cli.TokenLanguage[string]{
		Name:   "pair",
		Count:  cli.Count.Exactly(2),
		Parse:  func(context.Context, []string) (string, error) { return "pair", nil },
		Format: func(string) []string { return []string{"left", "right"} },
	}
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
			root.Dialect.CommandNormalization(cli.NameNormalizationASCIICaseFold)
			root.Subcommands.Command(cli.Command[config]{
				Name: "Deploy",
			})
			root.Flags.String(&root.Config.Path).CompleteBy(func(_ context.Context, _ *config, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				valueRequest = request
				replace := cli.CompletionRange{
					Start: 0,
					End:   request.Cursor,
				}
				return cli.CompletionContribution{
					Replace: &replace,
					Candidates: []cli.CompletionCandidate{
						{
							Value:       request.Text + "-candidate",
							Replacement: []string{request.Text + "-replacement"},
						},
					},
				}, nil
			}).
				Long("path")
			root.Subcommands.Command(cli.Command[config]{
				Name: "tokens",
				Declare: func(child *cli.CommandBuilder[config]) {
					destination := child.Values.String(&child.Config.Pair)
					child.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
						form.Tokens(
							destination,
							"pair",
							language,
						).CompleteTokensBy(
							func(_ context.Context, _ *config, request cli.TokenLanguageCompletionRequest) (cli.CompletionContribution, error) {
								tokenRequest = request
								return cli.CompletionContribution{
									Candidates: []cli.CompletionCandidate{{Value: "right"}},
								}, nil
							},
						)
					})
				},
			})
			root.Subcommands.Command(cli.Command[config]{
				Name: "delegate",
				Declare: func(child *cli.CommandBuilder[config]) {
					destination := child.Values.Sequence(&child.Config.Rest, cli.Value.String())
					child.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
						form.Remainder(
							destination,
							"rest",
							cli.Count.ZeroOrMore(),
						).DelegateCompletionBy(
							func(_ context.Context, _ *config, request cli.CompletionDelegationRequest) (cli.CompletionDelegation, error) {
								delegationRequest = request
								return cli.CompletionDelegation{
									Args:   append([]string(nil), request.Args...),
									Cursor: request.Cursor,
								}, nil
							},
						)
					})
				},
			})
		},
	})
	if err != nil {
		f.Fatalf("Compile() error = %v", err)
	}
	f.Add("de")
	f.Add("PA")
	f.Add("")
	f.Fuzz(func(t *testing.T, text string) {
		text = completionFuzzFuzzASCIIFragment(text, 20, true)
		client := cli.CompletionClient{
			Match: cli.CompletionMatchOptions{
				Mode: cli.CompletionMatchModePrefix,
				Case: cli.CompletionCaseInsensitive,
			},
		}

		command, err := completionFuzzComplete(t, app, cli.CompletionRequest{
			Args: []string{text},
			Cursor: cli.CompletionCursor{
				Argument: 0,
				Byte:     len(text),
			},
			Client: client,
		})
		if err != nil {
			t.Fatalf("command Complete() error = %v", err)
		}
		if command.Replace.Argument != 0 || command.Replace.Start < 0 || command.Replace.End < command.Replace.Start ||
			command.Replace.End > len(text) {
			t.Fatalf("command replacement coordinates = %#v for %q", command.Replace, text)
		}

		valueRequest = cli.ValueCompletionRequest{}
		value, err := completionFuzzComplete(t, app, cli.CompletionRequest{
			Args: []string{"--path", text},
			Cursor: cli.CompletionCursor{
				Argument: 1,
				Byte:     len(text),
			},
			Client: client,
		})
		if err != nil {
			t.Fatalf("value Complete() error = %v", err)
		}
		if valueRequest.Target.Kind != cli.CompletionTargetFlagValue || valueRequest.Text != text ||
			valueRequest.Cursor != len(text) ||
			valueRequest.Replace.Start != 0 ||
			valueRequest.Replace.End != len(text) {
			t.Fatalf("flag-value request lost structural coordinates: %#v", valueRequest)
		}
		if value.Replace.Argument != 1 || value.Replace.Start != 0 || value.Replace.End != len(text) {
			t.Fatalf("structural replacement span = %#v", value.Replace)
		}

		tokenRequest = cli.TokenLanguageCompletionRequest{}
		token, err := completionFuzzComplete(t, app, cli.CompletionRequest{
			Args: []string{"tokens", "left", text},
			Cursor: cli.CompletionCursor{
				Argument: 2,
				Byte:     len(text),
			},
			Client: client,
		})
		if err != nil {
			t.Fatalf("token Complete() error = %v", err)
		}
		if tokenRequest.Target.Kind != cli.CompletionTargetTokenLanguage ||
			!slices.Equal(tokenRequest.Tokens, []string{"left"}) ||
			tokenRequest.Text != text ||
			tokenRequest.Cursor != len(text) {
			t.Fatalf("bounded language completion ownership = %#v", tokenRequest)
		}
		if len(token.Targets) == 0 {
			t.Fatalf("token-language target was lost: %#v", token)
		}

		delegationRequest = cli.CompletionDelegationRequest{}
		delegated, err := completionFuzzComplete(t, app, cli.CompletionRequest{
			Args: []string{"delegate", text},
			Cursor: cli.CompletionCursor{
				Argument: 1,
				Byte:     len(text),
			},
		})
		if err != nil {
			t.Fatalf("delegation Complete() error = %v", err)
		}
		if delegationRequest.Target.Kind != cli.CompletionTargetRemainder ||
			!slices.Equal(delegationRequest.Args, []string{text}) ||
			delegationRequest.Cursor != (cli.CompletionCursor{
				Argument: 0,
				Byte:     len(text),
			}) {
			t.Fatalf("delegation ownership transfer = %#v", delegationRequest)
		}
		if delegated.Delegation == nil || len(delegated.Items) != 0 || len(delegated.NativeRequests) != 0 {
			t.Fatalf("delegation not exclusive = %#v", delegated)
		}

		loads = 0
		responseRequest = cli.ResponseFileCompletionRequest{}
		response, err := completionFuzzComplete(t, app, cli.CompletionRequest{
			Args: []string{"@" + text},
			Cursor: cli.CompletionCursor{
				Argument: 0,
				Byte:     len(text) + 1,
			},
			ResponseFiles: cli.ResponseFileAccess{
				Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
					loads++
					return cli.ResponseFile{}, errors.New("cursor response must not load")
				},
				Complete: func(_ context.Context, request cli.ResponseFileCompletionRequest) (cli.CompletionContribution, error) {
					responseRequest = request
					return cli.CompletionContribution{
						Candidates: []cli.CompletionCandidate{{Value: request.Path + ".rsp"}},
					}, nil
				},
			},
		})
		if err != nil {
			t.Fatalf("response Complete() error = %v", err)
		}
		if loads != 0 || responseRequest.Path != text || responseRequest.Cursor != len(text) {
			t.Fatalf("response cursor was loaded or request unbounded: loads=%d request=%#v", loads, responseRequest)
		}
		foundResponseTarget := false
		for _, target := range response.Targets {
			foundResponseTarget = foundResponseTarget || target.Kind == cli.CompletionTargetResponseFile
		}
		if !foundResponseTarget {
			t.Fatalf("response-file target missing: %#v", response.Targets)
		}

		first, firstErr := completionFuzzComplete(t, app, cli.CompletionRequest{
			Args: []string{"--path", text},
			Cursor: cli.CompletionCursor{
				Argument: 1,
				Byte:     len(text),
			},
			Client: client,
		})
		second, secondErr := completionFuzzComplete(t, app, cli.CompletionRequest{
			Args: []string{"--path", text},
			Cursor: cli.CompletionCursor{
				Argument: 1,
				Byte:     len(text),
			},
			Client: client,
		})
		if (firstErr == nil) != (secondErr == nil) ||
			!equalCompletionFuzzSummary(completionFuzzShape(first), completionFuzzShape(second)) {
			t.Fatalf(
				"equivalent completion changed semantic shape: first=%#v second=%#v",
				completionFuzzShape(first),
				completionFuzzShape(second),
			)
		}
	})
}

type completionFuzzSummary struct {
	Targets    int
	Items      []string
	Native     int
	Delegated  bool
	Replace    cli.CompletionSpan
	Partial    bool
	Truncated  bool
	Diagnostic int
}

func equalCompletionFuzzSummary(left, right completionFuzzSummary) bool {
	return left.Targets == right.Targets && slices.Equal(left.Items, right.Items) &&
		left.Native == right.Native && left.Delegated == right.Delegated && left.Replace == right.Replace &&
		left.Partial == right.Partial && left.Truncated == right.Truncated && left.Diagnostic == right.Diagnostic
}

func completionFuzzShape(result cli.Completion) completionFuzzSummary {
	summary := completionFuzzSummary{
		Targets:    len(result.Targets),
		Native:     len(result.NativeRequests),
		Delegated:  result.Delegation != nil,
		Replace:    result.Replace,
		Partial:    result.Partial,
		Truncated:  result.Truncated,
		Diagnostic: len(result.Diagnostics),
	}
	for _, item := range result.Items {
		summary.Items = append(summary.Items, item.Value)
	}
	return summary
}

func completionFuzzBoundedFuzzText(value string, maximum int) string {
	if len(value) > maximum {
		return value[:maximum]
	}
	return value
}

func completionFuzzFuzzASCIIFragment(value string, maximum int, preserveEmpty bool) string {
	value = completionFuzzBoundedFuzzText(value, maximum)
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 && !preserveEmpty {
		return "value"
	}
	return b.String()
}
