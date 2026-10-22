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

func responseFileMustCompilePublic[C any](t testing.TB, definition cli.Definition[C]) cli.Application {
	t.Helper()
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return app
}

type responseFileNameConfig struct{ Name string }

type responseFileValuesConfig struct{ Values []string }

type responseFileArgsConfig struct{ Args []string }

type responseFileConfig struct{}

func responseFileApp(t *testing.T) cli.Application {
	t.Helper()
	type config = responseFileNameConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
			root.Flags.String(&root.Config.Name).Long("name")
		},
	}
	return responseFileMustCompilePublic(t, definition)
}

func responseFileSequenceApp(t *testing.T) cli.Application {
	t.Helper()
	type config = responseFileValuesConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
			values := root.Values.Sequence(&root.Config.Values, cli.Value.String())
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Sequence(values, "value", cli.Count.ZeroOrMore()).Greedy()
			})
		},
	}
	return responseFileMustCompilePublic(t, definition)
}

func declareResponseFileFixture(root *cli.CommandBuilder[struct{}]) {
	root.Dialect.ResponseFiles(true)
}

func TestResponseFileLoadAdapterSnapshotsHostResult(t *testing.T) {
	t.Parallel()
	app := responseFileApp(t)
	content := []byte("--name before")
	file := cli.ResponseFile{
		Key:         "args",
		Revision:    cli.Revision("r1"),
		Location:    "/tmp/args",
		IncludeBase: "/tmp",
		Content:     content,
	}
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"@args"},
		ResponseFiles: cli.ResponseFileAccess{
			Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
				return file, nil
			},
		},
	})
	content[0] = 'X'
	file.Key = "mutated"
	source := clitest.InputSourceByKey(cli.InputKindResponseFile, "args")
	proof.Require(
		t,
		clitest.TokenAt(2).Text("before"),
		clitest.InspectionObservedInputSourcesExactly(cli.InputKindResponseFile, source),
		source.Kind(cli.InputKindResponseFile),
		source.Key("args"),
		source.Location("/tmp/args"),
	)
}

func TestResponseFileLimitsAdapterPreservesAllLimitFields(t *testing.T) {
	t.Parallel()
	app := responseFileApp(t)
	files := clitest.ResponseFiles{
		Loads: map[cli.ResponseFileRequest]cli.ResponseFile{
			{
				Path: "one",
			}: {
				Key:      "one",
				Revision: cli.Revision("r1"),
				Content:  []byte("@two"),
			},
			{
				Path: "two",
			}: {
				Key:      "two",
				Revision: cli.Revision("r1"),
				Content:  []byte("--name value"),
			},
		},
	}
	inspectionRequest := cli.InspectionRequest{
		Args:          []string{"@one"},
		ResponseFiles: files.Access(),
		Limits: cli.AnalysisLimits{
			ResponseFiles: cli.ResponseFileLimits{
				Depth:             1,
				SourceBytes:       1024,
				TotalSourceBytes:  1024,
				TotalTokens:       100,
				TokenBytes:        1024,
				IncludeReferences: 10,
			},
		},
	}
	clitest.Inspect(t.Context(), app, inspectionRequest).Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticResponseFileLimitExceeded).Present()))
	inspectionRequestUnderTest := cli.InspectionRequest{
		Args:          []string{"@one"},
		ResponseFiles: files.Access(),
		Limits: cli.AnalysisLimits{
			ResponseFiles: cli.ResponseFileLimits{
				Depth:             10,
				SourceBytes:       1,
				TotalSourceBytes:  1024,
				TotalTokens:       100,
				TokenBytes:        1024,
				IncludeReferences: 10,
			},
		},
	}
	clitest.Inspect(t.Context(), app, inspectionRequestUnderTest).Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticResponseFileLimitExceeded).Present()))
}

func TestResponseFileCompleteOwnsMarkerBoundaryAndSnapshotsContribution(t *testing.T) {
	t.Parallel()
	app := responseFileApp(t)
	request := cli.ResponseFileCompletionRequest{
		Path:   "fi",
		Base:   "/work",
		Cursor: 2,
		Replace: cli.CompletionRange{
			Start: 0,
			End:   2,
		},
		Trigger: cli.CompletionTrigger{
			Kind: cli.CompletionTriggerInvoked,
		},
	}
	replace := cli.CompletionRange{
		Start: 0,
		End:   2,
	}
	candidates := []cli.CompletionCandidate{
		{
			Value:       "file.txt",
			Replacement: []string{"file.txt"},
			Kind:        cli.CompletionCandidateFile,
		},
	}
	var requests clitest.ResponseFileRequests
	files := clitest.ResponseFiles{
		Completions: map[cli.ResponseFileCompletionRequest]cli.CompletionContribution{
			request: {
				Replace:    &replace,
				Candidates: candidates,
			},
		},
		Requests: &requests,
	}
	result, err := func() (cli.Completion, error) {
		completionRequest := cli.CompletionRequest{
			Args: []string{"@fi"},
			Cursor: cli.CompletionCursor{
				Argument: 0,
				Byte:     3,
			},
			WorkingDirectory: "/work",
			ResponseFiles:    files.Access(),
		}
		proof := clitest.Complete(t.Context(), app, completionRequest)
		proof.Require(t, clitest.CompletionItem("file.txt").Present())
		return proof.Completion()
	}()
	if err != nil || len(result.Items) != 1 || result.Items[0].Value != "file.txt" {
		t.Fatalf("Complete=%#v err=%v", result, err)
	}
	candidates[0].Value = "mutated"
	replace.End = 99
	if result.Items[0].Value != "file.txt" || result.Replace.End == 99 {
		t.Fatalf("completion aliases callback contribution: %#v", result)
	}
	if got := requests.Completions(); !slices.Equal(got, []cli.ResponseFileCompletionRequest{request}) {
		t.Fatalf("requests=%#v", got)
	}
}

func TestResponseFileCompleteRejectsPathExternalReplacementAndPreservesCallbackFailure(t *testing.T) {
	t.Parallel()
	app := responseFileApp(t)
	bad := cli.ResponseFileAccess{
		Complete: func(context.Context, cli.ResponseFileCompletionRequest) (cli.CompletionContribution, error) {
			r := cli.CompletionRange{
				Start: 0,
				End:   99,
			}
			contribution := cli.CompletionContribution{
				Replace: &r,
			}
			return contribution, nil
		},
	}
	completionRequest := cli.CompletionRequest{
		Args: []string{"@fi"},
		Cursor: cli.CompletionCursor{
			Argument: 0,
			Byte:     3,
		},
		ResponseFiles: bad,
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidCompletionContribution).Present()))

	sentinel := errors.New("completion failed")
	failure := cli.ResponseFileAccess{
		Complete: func(context.Context, cli.ResponseFileCompletionRequest) (cli.CompletionContribution, error) {
			return cli.CompletionContribution{}, sentinel
		},
	}
	completionRequestUnderTest := cli.CompletionRequest{
		Args: []string{"@fi"},
		Cursor: cli.CompletionCursor{
			Argument: 0,
			Byte:     3,
		},
		ResponseFiles: failure,
	}
	proof := clitest.Complete(t.Context(), app, completionRequestUnderTest)
	proof.Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticCompletionFailure).Present()))
	_, err := proof.Completion()
	if !errors.Is(err, sentinel) {
		t.Fatalf("callback failure=%v", err)
	}
}

func TestResponseFileDiagnosticFailureMapsBoundedDiagnosticIdentities(t *testing.T) {
	t.Parallel()
	app := responseFileApp(t)
	cases := []struct {
		name   string
		access cli.ResponseFileAccess
		args   []string
		code   cli.DiagnosticCode
	}{
		{
			name:   "access unavailable",
			access: cli.ResponseFileAccess{},
			args:   []string{"@missing"},
			code:   cli.DiagnosticResponseFileAccessUnavailable,
		},
		{
			name: "load failure",
			access: cli.ResponseFileAccess{
				Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
					return cli.ResponseFile{}, errors.New("load failed")
				},
			},
			args: []string{"@missing"},
			code: cli.DiagnosticResponseFileLoadFailure,
		},
		{
			name: "invalid source",
			access: cli.ResponseFileAccess{
				Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
					return cli.ResponseFile{}, nil
				},
			},
			args: []string{"@missing"},
			code: cli.DiagnosticInvalidResponseFileSource,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			clitest.Inspect(t.Context(), app, cli.InspectionRequest{
				Args:          tc.args,
				ResponseFiles: tc.access,
			}).Require(
				t,
				clitest.FailsWith(clitest.Diagnostic(tc.code).Present()),
			)
		})
	}
}

func TestResponseFileDiagnosticFailurePreservesCauseAndLimitIdentity(t *testing.T) {
	t.Parallel()
	app := responseFileApp(t)
	sentinel := errors.New("load failed")
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"@x"},
		ResponseFiles: cli.ResponseFileAccess{
			Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
				return cli.ResponseFile{}, sentinel
			},
		},
	})
	proof.Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticResponseFileLoadFailure).Present()))
	_, err := proof.Inspection()
	if !errors.Is(err, sentinel) {
		t.Fatalf("load failure err=%v, want cause %v", err, sentinel)
	}

	files := clitest.ResponseFiles{
		Loads: map[cli.ResponseFileRequest]cli.ResponseFile{
			{
				Path: "x",
			}: {
				Key:      "x",
				Revision: cli.Revision("r1"),
				Content:  []byte("0123456789"),
			},
		},
	}
	inspectionRequest := cli.InspectionRequest{
		Args:          []string{"@x"},
		ResponseFiles: files.Access(),
		Limits: cli.AnalysisLimits{
			ResponseFiles: cli.ResponseFileLimits{
				Depth:             10,
				SourceBytes:       2,
				TotalSourceBytes:  20,
				TotalTokens:       20,
				TokenBytes:        20,
				IncludeReferences: 20,
			},
		},
	}
	clitest.Inspect(t.Context(), app, inspectionRequest).Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticResponseFileLimitExceeded).Present()))
}

func TestResponseFileDiagnosticFailurePublishesSourceClosedFlatTextSpan(t *testing.T) {
	t.Parallel()
	app := responseFileApp(t)
	files := clitest.ResponseFiles{
		Loads: map[cli.ResponseFileRequest]cli.ResponseFile{
			{
				Path: "root",
				Base: "/work",
			}: {
				Key:         "root-key",
				Revision:    cli.Revision("r1"),
				Location:    "virtual:root",
				IncludeBase: "/work",
				Content:     []byte("@missing"),
			},
		},
		LoadErrors: map[cli.ResponseFileRequest]error{
			{
				Path: "missing",
				Base: "/work",
			}: errors.New("gone"),
		},
	}
	diagnostic := clitest.Diagnostic(cli.DiagnosticResponseFileLoadFailure)
	inspectionRequest := cli.InspectionRequest{
		Args:             []string{"@root"},
		WorkingDirectory: "/work",
		ResponseFiles:    files.Access(),
	}
	clitest.Inspect(t.Context(), app, inspectionRequest).Require(
		t,
		clitest.FailsWith(
			diagnostic.Present(),
			diagnostic.SpanSource().Kind(cli.InputKindResponseFile),
			diagnostic.SpanSource().Key("root-key"),
			diagnostic.SpanSource().Location("virtual:root"),
		),
	)
}

func TestExpandResponseArgumentsSharesSiblingSnapshotIdentityTransactionally(t *testing.T) {
	t.Parallel()
	app := responseFileSequenceApp(t)
	calls := 0
	access := cli.ResponseFileAccess{
		Load: func(_ context.Context, r cli.ResponseFileRequest) (cli.ResponseFile, error) {
			calls++
			file := cli.ResponseFile{
				Key:      r.Path,
				Revision: cli.Revision("r1"),
				Content:  []byte(r.Path),
			}
			return file, nil
		},
	}
	one := clitest.InputSourceByKey(cli.InputKindResponseFile, "one")
	two := clitest.InputSourceByKey(cli.InputKindResponseFile, "two")
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args:          []string{"@one", "@two"},
		ResponseFiles: access,
	}).Require(
		t,
		clitest.InspectionReady(),
		clitest.InspectionObservedInputSourcesExactly(cli.InputKindResponseFile, one, two),
	)
	if calls != 2 {
		t.Fatalf("loads=%d, want 2", calls)
	}
}

func TestExpandResponseArgumentsSharesSiblingCumulativeResponseBudgets(t *testing.T) {
	t.Parallel()
	app := responseFileSequenceApp(t)
	access := cli.ResponseFileAccess{
		Load: func(_ context.Context, r cli.ResponseFileRequest) (cli.ResponseFile, error) {
			file := cli.ResponseFile{
				Key:      r.Path,
				Revision: cli.Revision("r1"),
				Content:  []byte("1234567890"),
			}
			return file, nil
		},
	}
	inspectionRequest := cli.InspectionRequest{
		Args:          []string{"@one", "@two"},
		ResponseFiles: access,
		Limits: cli.AnalysisLimits{
			ResponseFiles: cli.ResponseFileLimits{
				Depth:             10,
				SourceBytes:       100,
				TotalSourceBytes:  15,
				TotalTokens:       100,
				TokenBytes:        100,
				IncludeReferences: 10,
			},
		},
	}
	clitest.Inspect(t.Context(), app, inspectionRequest).Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticResponseFileLimitExceeded).Present()))
}

func TestRevisionIsOpaqueComparableIngress(t *testing.T) {
	t.Parallel()
	if cli.Revision("") != (cli.InputRevision{}) {
		t.Fatal("empty revision token did not produce zero revision")
	}
	left := cli.Revision("same")
	right := cli.Revision("same")
	if left != right {
		t.Fatal("equal revision tokens did not compare equal")
	}
	if cli.Revision("left") == cli.Revision("right") {
		t.Fatal("distinct revision tokens compared equal")
	}
}

func TestCompleteRejectsLateInvalidResponseFileContributionAtomically(t *testing.T) {
	t.Parallel()
	type config = responseFileConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: func(root *cli.CommandBuilder[config]) { root.Dialect.ResponseFiles(true) },
	})
	completionRequest := cli.CompletionRequest{
		Args: []string{"@pre"},
		Cursor: cli.CompletionCursor{
			Argument: 0,
			Byte:     4,
		},
		ResponseFiles: cli.ResponseFileAccess{
			Complete: func(context.Context, cli.ResponseFileCompletionRequest) (cli.CompletionContribution, error) {
				replace := cli.CompletionRange{
					Start: 1,
					End:   3,
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
			},
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
	clitest.Complete(t.Context(), app, completionRequest).Require(
		t,
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidCompletionContribution).Present()),
		clitest.CompletionPartial(),
		clitest.CompletionTarget(cli.CompletionTargetResponseFile).Present(),
		clitest.CompletionReplace(cli.CompletionSpan{
			Argument: 0,
			Start:    1,
			End:      4,
		}),
		clitest.CompletionSpacing(cli.CompletionSpacingDefault),
		clitest.CompletionObservedItemsExactly(),
		clitest.CompletionObservedGroupsExactly(),
		clitest.CompletionObservedMessagesExactly(),
		clitest.CompletionObservedNativeRequestsExactly(),
		clitest.CompletionNotTruncated(),
		clitest.CompletionNoRefreshOnInput(),
	)
}

func TestCompleteResponseFileTargetOwnsMarkerAndUsesPathRelativeCompleter(t *testing.T) {
	t.Parallel()
	type config = responseFileConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
		},
	})

	var request cli.ResponseFileCompletionRequest
	target := clitest.CompletionTarget(cli.CompletionTargetResponseFile)
	completionRequest := cli.CompletionRequest{
		Args: []string{"@fi"},
		Cursor: cli.CompletionCursor{
			Argument: 0,
			Byte:     3,
		},
		WorkingDirectory: "/work",
		ResponseFiles: cli.ResponseFileAccess{
			Complete: func(_ context.Context, got cli.ResponseFileCompletionRequest) (cli.CompletionContribution, error) {
				request = got
				replace := cli.CompletionRange{
					Start: 0,
					End:   2,
				}
				intent := &cli.CompletionNativeIntent{
					Kind: cli.CompletionNativeFiles,
					Mode: cli.CompletionNativeAlways,
				}
				contribution := cli.CompletionContribution{
					Replace: &replace,
					Candidates: []cli.CompletionCandidate{{
						Value: "file.txt",
						Kind:  cli.CompletionCandidateFile,
					}},
					Native: intent,
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
				CandidateKinds: true,
				Native: cli.CompletionNativeCapabilities{
					Files: true,
				},
			},
		},
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t,
		clitest.CompletionTargetsExactly(clitest.CompletionTargetAt(0)),
		clitest.CompletionTargetAt(0).Kind(cli.CompletionTargetResponseFile),
		target.OnSelectedCommand(),
		target.CommandIdentityPresent(),
		clitest.CompletionReplace(cli.CompletionSpan{
			Argument: 0,
			Start:    1,
			End:      3,
		}),
		clitest.CompletionItemsExactly(clitest.CompletionItem("file.txt")),
		clitest.CompletionItem("file.txt").Kind(cli.CompletionCandidateFile),
		clitest.CompletionItem("file.txt").Match(cli.CompletionMatchKindPrefix),
		clitest.CompletionNativeRequestsExactly(clitest.CompletionNativeRequestAt(0)),
		clitest.CompletionNativeRequestAt(0).Kind(cli.CompletionNativeFiles),
		clitest.CompletionNativeRequestAt(0).Mode(cli.CompletionNativeAlways),
		clitest.CompletionNativeRequestAt(0).Target().Kind(cli.CompletionTargetResponseFile),
	)
	wantRequest := cli.ResponseFileCompletionRequest{
		Path:   "fi",
		Base:   "/work",
		Cursor: 2,
		Replace: cli.CompletionRange{
			Start: 0,
			End:   2,
		},
		Trigger: cli.CompletionTrigger{
			Kind: cli.CompletionTriggerInvoked,
		},
	}
	if request != wantRequest {
		t.Fatalf("request = %#v, want %#v", request, wantRequest)
	}
}

func TestCompleteResponseFileSupportLoadsEarlierExpansionBeforeCompletingCurrentReference(t *testing.T) {
	t.Parallel()
	type config = responseFileConfig
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
			root.Subcommands.Command(cli.Command[config]{
				Name: "child",
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	ordinary, err := func() (cli.Completion, error) {
		proof := clitest.Complete(t.Context(), app, cli.CompletionRequest{
			Args: []string{""},
			Cursor: cli.CompletionCursor{
				Argument: 0,
			},
		})
		proof.Require(t, clitest.CompletionItem("child").Present())
		return proof.Completion()
	}()
	if err != nil {
		t.Fatalf("ordinary Complete() error = %v", err)
	}
	foundChild := false
	for _, item := range ordinary.Items {
		foundChild = foundChild || item.Value == "child"
	}
	if !foundChild {
		t.Fatalf("ordinary response-enabled completion omitted child: %#v", ordinary.Items)
	}

	loads, completions := 0, 0
	completionRequest := cli.CompletionRequest{
		Args: []string{"@prior", "@fi"},
		Cursor: cli.CompletionCursor{
			Argument: 1,
			Byte:     3,
		},
		ResponseFiles: cli.ResponseFileAccess{
			Load: func(_ context.Context, request cli.ResponseFileRequest) (cli.ResponseFile, error) {
				loads++
				if request.Path != "prior" {
					t.Fatalf("load path = %q", request.Path)
				}
				file := cli.ResponseFile{
					Key:      "prior",
					Revision: cli.Revision("r1"),
					Content:  nil,
				}
				return file, nil
			},
			Complete: func(context.Context, cli.ResponseFileCompletionRequest) (cli.CompletionContribution, error) {
				completions++
				return cli.CompletionContribution{}, nil
			},
		},
		Client: cli.CompletionClient{
			Match: cli.CompletionMatchOptions{
				Mode: cli.CompletionMatchModePrefix,
				Case: cli.CompletionCaseSensitive,
			},
		},
	}
	result, err := app.Complete(t.Context(), completionRequest)
	if err != nil || loads != 1 || completions != 1 {
		t.Fatalf("earlier response join = err %v loads=%d completions=%d", err, loads, completions)
	}
	if result.Replace.Argument != 1 || len(result.Targets) != 1 ||
		result.Targets[0].Kind != cli.CompletionTargetResponseFile {
		t.Fatalf("completion result = %#v", result)
	}
}

func TestCompleteResponseFileCallbackFailureReturnsPartialTargetAndStructuredCompletionFailure(t *testing.T) {
	t.Parallel()
	type config = responseFileConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: func(root *cli.CommandBuilder[config]) { root.Dialect.ResponseFiles(true) },
	})
	cause := errors.New("response completion failed")
	completionRequest := cli.CompletionRequest{
		Args: []string{"@fi"},
		Cursor: cli.CompletionCursor{
			Argument: 0,
			Byte:     3,
		},
		ResponseFiles: cli.ResponseFileAccess{
			Complete: func(context.Context, cli.ResponseFileCompletionRequest) (cli.CompletionContribution, error) {
				contribution := cli.CompletionContribution{
					Candidates: []cli.CompletionCandidate{{
						Value: "must-not-commit",
					}},
				}
				return contribution, cause
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
	proof.Require(
		t,
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticCompletionFailure).Present()),
		clitest.CompletionPartial(),
		clitest.CompletionObservedTargetsExactly(clitest.CompletionTargetAt(0)),
		clitest.CompletionObservedItemsExactly(),
	)
	_, err := proof.Completion()
	if !errors.Is(err, cause) {
		t.Fatalf("completion error = %v, want cause %v", err, cause)
	}
}

func TestResponseExpansionFailureRollsBackLexicalFrameButPreservesAcceptedProviderBatch(t *testing.T) {
	t.Parallel()
	type config = responseFileConfig
	providerCalls := 0
	app := responseFileMustCompilePublic(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				providerCalls++
				batch.Command(cli.Command[config]{
					Name:    "remote",
					Declare: func(child *cli.CommandBuilder[config]) { child.Dialect.ResponseFiles(true) },
				})
				return nil
			})
		},
	})
	nestedFailure := errors.New("nested response failure")
	loadedBroken := false
	failed := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"@outer"},
		ResponseFiles: cli.ResponseFileAccess{
			Load: func(_ context.Context, request cli.ResponseFileRequest) (cli.ResponseFile, error) {
				switch request.Path {
				case "outer":
					file := cli.ResponseFile{
						Key:      "outer",
						Revision: cli.Revision("r1"),
						Content:  []byte("remote @broken"),
					}
					return file, nil
				case "broken":
					loadedBroken = true
					return cli.ResponseFile{}, nestedFailure
				default:
					return cli.ResponseFile{}, errors.New("unexpected response path")
				}
			},
		},
	})
	failed.Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticResponseFileLoadFailure).Present()))
	_, err := failed.Inspection()
	if !errors.Is(err, nestedFailure) {
		t.Fatalf("Inspect() error = %v, want nested response failure", err)
	}
	if providerCalls != 1 {
		t.Fatalf("provider calls after failed lexical frame = %d, want 1", providerCalls)
	}
	if !loadedBroken {
		t.Fatal(
			"provider-contributed remote command did not remain usable through nested response recognition before failure",
		)
	}

	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"remote"},
	}).Require(
		t,
		clitest.InspectionReady(),
		clitest.RootCommand().Command("remote").Exists(),
		clitest.InspectionObservedInputSourcesExactly(cli.InputKindResponseFile),
	)
	if providerCalls != 2 {
		t.Fatalf("provider calls after fresh operation = %d, want 2", providerCalls)
	}
}

func TestResponseFileProviderDemandUsesChildFactSnapshot(t *testing.T) {
	t.Parallel()
	type config struct{ Gate string }
	var gate cli.Destination[config, string]
	factCalls := 0
	providerCalls := 0

	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
			gate = root.Facts.Require(&root.Config.Gate)
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				providerCalls++
				batch.Command(cli.Command[config]{
					Name: "remote",
				})
				return nil
			}, gate)
		},
	})

	access := cli.ResponseFileAccess{
		Load: func(_ context.Context, request cli.ResponseFileRequest) (cli.ResponseFile, error) {
			file := cli.ResponseFile{
				Key:      request.Path,
				Revision: cli.Revision("r1"),
				Content:  []byte("remote"),
			}
			return file, nil
		},
	}
	facts := []cli.FactProvider{cli.FactsFor[config](func(_ context.Context, supply *cli.FactSupply[config]) error {
		factCalls++
		if factCalls == 1 {
			supply.Set(gate, "open")
		}
		return nil
	})}

	inspectionRequest := cli.InspectionRequest{
		Args:          []string{"@args"},
		Facts:         facts,
		ResponseFiles: access,
	}
	clitest.Inspect(t.Context(), app, inspectionRequest).Require(
		t,
		clitest.InspectionReady(),
		clitest.RootCommand().Command("remote").Exists(),
	)

	if factCalls != 2 || providerCalls != 1 {
		t.Fatalf("response child demand calls = facts %d providers %d, want 2/1", factCalls, providerCalls)
	}
}

func TestResponseFileProviderDemandPreservesChildFactFailure(t *testing.T) {
	t.Parallel()
	type config struct{ Gate string }
	var gate cli.Destination[config, string]
	cause := errors.New("response child fact failure")

	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
			gate = root.Facts.Require(&root.Config.Gate)
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Command(cli.Command[config]{
					Name: "remote",
				})
				return nil
			}, gate)
		},
	})

	inspectionRequest := cli.InspectionRequest{
		Args: []string{"@args"},
		Facts: []cli.FactProvider{cli.FactsFor[config](func(context.Context, *cli.FactSupply[config]) error {
			return cause
		})},
		ResponseFiles: cli.ResponseFileAccess{
			Load: func(_ context.Context, request cli.ResponseFileRequest) (cli.ResponseFile, error) {
				file := cli.ResponseFile{
					Key:      request.Path,
					Revision: cli.Revision("r1"),
					Content:  []byte("remote"),
				}
				return file, nil
			},
		},
	}
	proof := clitest.Inspect(t.Context(), app, inspectionRequest)
	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticFactSupplyFailure).Present()),
	)
	_, err := proof.Inspection()
	if !errors.Is(err, cause) {
		t.Fatalf("Inspect() error = %v, want %v", err, cause)
	}
}

func TestResponseFileSelectsProviderCommandConsistentlyAcrossInspectionAndRun(t *testing.T) {
	t.Parallel()
	type config = responseFileConfig
	providerCalls := 0
	handlerCalls := 0
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				providerCalls++
				batch.Command(cli.Command[config]{
					Name: "remote",
					Run: func(context.Context, *config) error {
						handlerCalls++
						return nil
					},
				})
				return nil
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	access := cli.ResponseFileAccess{
		Load: func(_ context.Context, request cli.ResponseFileRequest) (cli.ResponseFile, error) {
			if request.Path != "args" {
				return cli.ResponseFile{}, errors.New("unexpected response path")
			}
			file := cli.ResponseFile{
				Key:      "args",
				Revision: cli.Revision("r1"),
				Content:  []byte("remote"),
			}
			return file, nil
		},
	}
	inspection, err := func() (cli.Inspection, error) {
		proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args:          []string{"@args"},
			ResponseFiles: access,
		})
		proof.Require(t, clitest.InspectionReady())
		return proof.Inspection()
	}()
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	selected := inspection.Commands[inspection.Selection.Command.Selected]
	if selected.Name != "remote" || inspection.Status != cli.InspectionStatusReady {
		t.Fatalf("Inspection selection/status = %#v selected=%#v", inspection.Selection, selected)
	}
	if handlerCalls != 0 {
		t.Fatalf("Inspection executed provider handler %d times", handlerCalls)
	}
	if err := app.Run(t.Context(), cli.RunRequest{
		Args:          []string{"@args"},
		ResponseFiles: access,
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if providerCalls != 2 || handlerCalls != 1 {
		t.Fatalf("operation isolation/provider execution = providers %d handlers %d", providerCalls, handlerCalls)
	}
}

func TestInvocationNeedsResponseFileRecognitionIsLexicallyNarrow(t *testing.T) {
	t.Parallel()
	type config = responseFileArgsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "response-lexical",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Strings(&form.Config.Args, "ARG", cli.Count.ZeroOrMore())
			})
		},
	})
	for _, test := range []struct {
		name         string
		args         []string
		wantLoads    int
		wantLoadPath string
		wantToken    string
	}{
		{
			name: "empty",
			args: nil,
		},
		{
			name: "ordinary",
			args: []string{"serve", "file.txt"},
		},
		{
			name:         "include",
			args:         []string{"@args.txt"},
			wantLoads:    1,
			wantLoadPath: "args.txt",
		},
		{
			name:      "literal marker",
			args:      []string{"@@literal"},
			wantToken: "@literal",
		},
		{
			name:      "escaped marker",
			args:      []string{`\@literal`},
			wantToken: "@literal",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			loads := 0
			var path string
			proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
				Args: append([]string(nil), test.args...),
				ResponseFiles: cli.ResponseFileAccess{
					Load: func(_ context.Context, request cli.ResponseFileRequest) (cli.ResponseFile, error) {
						loads++
						path = request.Path
						file := cli.ResponseFile{
							Key:      request.Path,
							Revision: cli.Revision("r1"),
						}
						return file, nil
					},
				},
			})
			if test.wantToken != "" {
				proof.Require(t, clitest.TokenAt(0).Text(test.wantToken))
			}
			_, err := proof.Inspection()
			clitest.Diagnostics(err).Require(t, clitest.Diagnostic(cli.DiagnosticResponseFileLoadFailure).Absent())
			if loads != test.wantLoads || path != test.wantLoadPath {
				t.Fatalf("loads=%d path=%q, want loads=%d path=%q", loads, path, test.wantLoads, test.wantLoadPath)
			}
		})
	}
}

func TestResponseFilePositionPreservesResponseFileContract(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name:    "response-position",
		Declare: func(root *cli.CommandBuilder[struct{}]) { root.Dialect.ResponseFiles(true) },
	})
	files := clitest.ResponseFiles{
		Loads: map[cli.ResponseFileRequest]cli.ResponseFile{
			{
				Path: "root",
			}: {
				Key:      "root",
				Location: "virtual:root",
				Content:  []byte("one\n@missing"),
			},
		},
		LoadErrors: map[cli.ResponseFileRequest]error{
			{
				Path: "missing",
			}: errors.New("gone"),
		},
	}
	diagnostic := clitest.Diagnostic(cli.DiagnosticResponseFileLoadFailure)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args:          []string{"@root"},
		ResponseFiles: files.Access(),
	}).Require(
		t,
		clitest.FailsWith(diagnostic.Present(), diagnostic.SpanPositioned()),
	)
}

func TestResponseFileCompleteHostProtocol(t *testing.T) {
	t.Parallel()
	app, err := cli.Compile(cli.Definition[struct{}]{
		Name:    "response-complete-host",
		Declare: declareResponseFileFixture,
	})
	if err != nil {
		t.Fatal(err)
	}
	completionRequest := cli.CompletionRequest{
		Args: []string{"@fi"},
		Cursor: cli.CompletionCursor{
			Argument: 0,
			Byte:     3,
		},
		ResponseFiles: cli.ResponseFileAccess{},
	}
	result, err := app.Complete(t.Context(), completionRequest)
	if err != nil {
		t.Fatalf("Complete() with no response-file completer: %v", err)
	}
	if result.Partial {
		t.Fatalf("Complete() unexpectedly partial: %#v", result)
	}
}

func TestResponseFilePositionHostProtocol(t *testing.T) {
	t.Parallel()
	app, err := cli.Compile(cli.Definition[struct{}]{
		Name:    "response-position-host",
		Declare: declareResponseFileFixture,
	})
	if err != nil {
		t.Fatal(err)
	}
	access := cli.ResponseFileAccess{
		Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
			file := cli.ResponseFile{
				Key:      "root",
				Location: "virtual:root",
				Content:  []byte("value\r\n\""),
			}
			return file, nil
		},
	}
	_, err = app.Inspect(t.Context(), cli.InspectionRequest{
		Args:          []string{"@root"},
		ResponseFiles: access,
	})
	if err == nil {
		t.Fatal("unterminated response-file quote succeeded")
	}
	report := cli.Diagnostics(err)
	if len(report.Diagnostics) == 0 || report.Diagnostics[0].Code != cli.DiagnosticInvalidResponseFile ||
		report.Diagnostics[0].Span == nil {
		t.Fatalf("diagnostics = %#v", report)
	}
	span := report.Diagnostics[0].Span
	if span.End.Line != 2 || span.End.Column != 2 {
		t.Fatalf("CRLF source position = %#v, want line 2 column 2", span.End)
	}
}

func TestResponseFileDiagnosticCodeHostProtocol(t *testing.T) {
	t.Parallel()
	app, err := cli.Compile(cli.Definition[struct{}]{
		Name:    "response-code-host",
		Declare: declareResponseFileFixture,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("text", func(t *testing.T) {
		t.Parallel()
		_, err := app.Inspect(t.Context(), cli.InspectionRequest{
			Args: []string{"@root"},
			ResponseFiles: cli.ResponseFileAccess{
				Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
					file := cli.ResponseFile{
						Key:     "root",
						Content: []byte{0xff},
					}
					return file, nil
				},
			},
		})
		if err == nil {
			t.Fatal("invalid UTF-8 response file succeeded")
		}
		report := cli.Diagnostics(err)
		if len(report.Diagnostics) == 0 || report.Diagnostics[0].Code != cli.DiagnosticInvalidResponseFile {
			t.Fatalf("diagnostics = %#v", report)
		}
	})

	t.Run("cycle", func(t *testing.T) {
		t.Parallel()
		_, err := app.Inspect(t.Context(), cli.InspectionRequest{
			Args: []string{"@root"},
			ResponseFiles: cli.ResponseFileAccess{
				Load: func(_ context.Context, request cli.ResponseFileRequest) (cli.ResponseFile, error) {
					if request.Path != "root" {
						t.Fatalf("response path = %q, want root", request.Path)
					}
					file := cli.ResponseFile{
						Key:     "root",
						Content: []byte("@root"),
					}
					return file, nil
				},
			},
		})
		if err == nil {
			t.Fatal("response-file cycle succeeded")
		}
		report := cli.Diagnostics(err)
		if len(report.Diagnostics) == 0 || report.Diagnostics[0].Code != cli.DiagnosticResponseFileCycle {
			t.Fatalf("diagnostics = %#v", report)
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		_, err := app.Inspect(ctx, cli.InspectionRequest{
			Args: []string{"@root"},
			ResponseFiles: cli.ResponseFileAccess{
				Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
					cancel()
					file := cli.ResponseFile{
						Key:     "root",
						Content: []byte("value"),
					}
					return file, nil
				},
			},
		})
		if err == nil {
			t.Fatal("canceled response-file expansion succeeded")
		}
		report := cli.Diagnostics(err)
		if len(report.Diagnostics) == 0 || report.Diagnostics[0].Code != cli.DiagnosticResponseFileCancellation {
			t.Fatalf("diagnostics = %#v", report)
		}
	})
}

func cutoverResponseApp(t *testing.T) cli.Application {
	t.Helper()
	type config struct{ Rest []string }
	helpConfig := cli.HelpConfig{
		Disabled: true,
	}
	definition := cli.Definition[config]{
		Name: "tool",
		Help: helpConfig,
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Strings(&form.Config.Rest, "REST", cli.Count.ZeroOrMore())
			})
		},
	}
	return clitest.Compile(t, definition)
}

func TestResponseExpansionEnforcesEveryFiniteBudget(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		limits cli.ResponseFileLimits
		files  clitest.ResponseFiles
		reason string
	}{
		{
			name: "source bytes",
			limits: cli.ResponseFileLimits{
				SourceBytes: 1,
			},
			files: clitest.ResponseFiles{
				Loads: map[cli.ResponseFileRequest]cli.ResponseFile{
					{
						Path: "root",
					}: {
						Key:     "root",
						Content: []byte("ab"),
					},
				},
			},
			reason: "source-bytes",
		},
		{
			name: "total source bytes",
			limits: cli.ResponseFileLimits{
				TotalSourceBytes: 2,
			},
			files: clitest.ResponseFiles{
				Loads: map[cli.ResponseFileRequest]cli.ResponseFile{
					{
						Path: "root",
					}: {
						Key:     "root",
						Content: []byte("@a"),
					},
					{
						Path: "a",
					}: {
						Key:     "a",
						Content: []byte("xx"),
					},
				},
			},
			reason: "total-source-bytes",
		},
		{
			name: "token bytes",
			limits: cli.ResponseFileLimits{
				TokenBytes: 1,
			},
			files: clitest.ResponseFiles{
				Loads: map[cli.ResponseFileRequest]cli.ResponseFile{
					{
						Path: "root",
					}: {
						Key:     "root",
						Content: []byte("ab"),
					},
				},
			},
			reason: "token-bytes",
		},
		{
			name: "total tokens",
			limits: cli.ResponseFileLimits{
				TotalTokens: 1,
			},
			files: clitest.ResponseFiles{
				Loads: map[cli.ResponseFileRequest]cli.ResponseFile{
					{
						Path: "root",
					}: {
						Key:     "root",
						Content: []byte("a b"),
					},
				},
			},
			reason: "total-tokens",
		},
		{
			name: "include references",
			limits: cli.ResponseFileLimits{
				IncludeReferences: 1,
			},
			files: clitest.ResponseFiles{
				Loads: map[cli.ResponseFileRequest]cli.ResponseFile{
					{
						Path: "root",
					}: {
						Key:     "root",
						Content: []byte("@a"),
					},
					{
						Path: "a",
					}: {
						Key: "a",
					},
				},
			},
			reason: "include-references",
		},
		{
			name: "depth",
			limits: cli.ResponseFileLimits{
				Depth: 1,
			},
			files: clitest.ResponseFiles{
				Loads: map[cli.ResponseFileRequest]cli.ResponseFile{
					{
						Path: "root",
					}: {
						Key:     "root",
						Content: []byte("@a"),
					},
					{
						Path: "a",
					}: {
						Key: "a",
					},
				},
			},
			reason: "depth",
		},
	}
	app := cutoverResponseApp(t)
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			diagnostic := clitest.Diagnostic(cli.DiagnosticResponseFileLimitExceeded)
			inspectionRequest := cli.InspectionRequest{
				Args:          []string{"@root"},
				ResponseFiles: test.files.Access(),
				Limits: cli.AnalysisLimits{
					ResponseFiles: test.limits,
				},
			}
			clitest.Inspect(t.Context(), app, inspectionRequest).Require(t, clitest.FailsWith(diagnostic.Present(), diagnostic.Reason(test.reason)))
		})
	}
}

type responseFileSensitiveCountConfig struct{ Count int }

func TestSensitiveResponseSourceRedactsFailureAndRetainsSourceIdentity(t *testing.T) {
	t.Parallel()

	type config = responseFileSensitiveCountConfig
	declare := func(root *cli.CommandBuilder[config]) {
		root.Dialect.ResponseFiles(true)
		root.Flags.Int(&root.Config.Count).Long("count")
	}
	definition := cli.Definition[config]{
		Name: "alienctl",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: declare,
	}
	app := clitest.Compile(t, definition)
	load := func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
		response := cli.ResponseFile{
			Key:       "secret-key",
			Revision:  cli.Revision("r1"),
			Location:  "vault:secret",
			Content:   []byte("--count classified"),
			Sensitive: true,
		}
		return response, nil
	}
	request := cli.InspectionRequest{
		Args: []string{"@secret"},
		ResponseFiles: cli.ResponseFileAccess{
			Load: load,
		},
	}

	proof := clitest.Inspect(t.Context(), app, request)
	_, err := proof.Inspection()
	if err == nil {
		t.Fatal("Inspect() error = nil, want invalid-value failure")
	}
	clitest.Diagnostics(err).Require(t,
		clitest.Diagnostic(cli.DiagnosticInvalidValue).InputRedacted(),
		clitest.DiagnosticInputGraphClosed(),
	)
	report := cli.Diagnostics(err)
	if len(report.Diagnostics) != 1 {
		t.Fatalf("Diagnostics() count = %d, want 1", len(report.Diagnostics))
	}
	diagnostic := report.Diagnostics[0]
	if diagnostic.Input != "" || !diagnostic.InputRedacted || !diagnostic.Sensitive {
		t.Fatalf("sensitive Diagnostic = %#v", diagnostic)
	}
	if diagnostic.InputSource == (cli.InputSourceID{}) {
		t.Fatal("sensitive Diagnostic lost response source identity")
	}
	source, ok := report.Sources[diagnostic.InputSource]
	if !ok || source.Key != "secret-key" || source.Location != "vault:secret" {
		t.Fatalf("response source = %#v, ok=%v", source, ok)
	}
}
