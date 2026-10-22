package clitest

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestResponseFilesZeroValueIsValidAndMissingRoutesFailDeterministically(t *testing.T) {
	t.Parallel()
	var files ResponseFiles
	request := cli.ResponseFileRequest{
		Path: "missing",
	}
	if _, err := files.Access().Load(t.Context(), request); !errors.Is(err, errResponseFileRouteMissing) {
		t.Fatalf("Load() error = %v, want %v", err, errResponseFileRouteMissing)
	}
}

func TestResponseFilesMatchesExactLoadRoutesAndSnapshotsAtAccess(t *testing.T) {
	t.Parallel()
	request := cli.ResponseFileRequest{
		Path: "./args.txt",
		Base: "/workspace",
	}
	response := cli.ResponseFile{
		Key:         "logical-key",
		Revision:    cli.Revision("r1"),
		Location:    "/workspace/args.txt",
		IncludeBase: "/workspace",
		Content:     []byte("--flag value"),
		Sensitive:   true,
	}
	var requests ResponseFileRequests
	files := ResponseFiles{
		Loads:    map[cli.ResponseFileRequest]cli.ResponseFile{request: response},
		Requests: &requests,
	}
	access := files.Access()

	response.Content[0] = 'X'
	files.Loads[request] = cli.ResponseFile{
		Content: []byte("changed"),
	}

	got, err := access.Load(t.Context(), request)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if string(got.Content) != "--flag value" || got.Key != "logical-key" || !got.Sensitive {
		t.Fatalf("Load() = %#v", got)
	}
	got.Content[0] = 'Y'
	again, err := access.Load(t.Context(), request)
	if err != nil {
		t.Fatalf("second Load() error = %v", err)
	}
	if string(again.Content) != "--flag value" {
		t.Fatalf("mutating returned response changed snapshot: %q", again.Content)
	}

	wantRequests := []cli.ResponseFileRequest{request, request}
	observed := requests.Loads()
	if !slices.Equal(observed, wantRequests) {
		t.Fatalf("Loads() = %#v, want %#v", observed, wantRequests)
	}
	observed[0].Path = "mutated"
	if got := requests.Loads()[0].Path; got != request.Path {
		t.Fatalf("mutating Loads result changed recorder: %q", got)
	}
}

func TestResponseFilesDoesNotNormalizeOrInventLoadRoutes(t *testing.T) {
	t.Parallel()
	registered := cli.ResponseFileRequest{
		Path: "./args.txt",
		Base: "/workspace",
	}
	files := ResponseFiles{
		Loads: map[cli.ResponseFileRequest]cli.ResponseFile{registered: {
			Content: []byte("ok"),
		}},
	}
	lookup := cli.ResponseFileRequest{
		Path: "args.txt",
		Base: "/workspace",
	}
	if _, err := files.Access().Load(t.Context(), lookup); !errors.Is(err, errResponseFileRouteMissing) {
		t.Fatalf("non-exact Load() error = %v, want %v", err, errResponseFileRouteMissing)
	}
}

func TestResponseFilesPreservesConfiguredFailures(t *testing.T) {
	t.Parallel()
	loadRequest := cli.ResponseFileRequest{
		Path: "broken.txt",
	}
	completionRequest := cli.ResponseFileCompletionRequest{
		Path: "broken",
	}
	loadErr := errors.New("load failed")
	completionErr := errors.New("completion failed")
	files := ResponseFiles{
		LoadErrors:       map[cli.ResponseFileRequest]error{loadRequest: loadErr},
		CompletionErrors: map[cli.ResponseFileCompletionRequest]error{completionRequest: completionErr},
	}
	access := files.Access()
	if _, err := access.Load(t.Context(), loadRequest); !errors.Is(err, loadErr) {
		t.Fatalf("Load() error = %v, want %v", err, loadErr)
	}
	if _, err := access.Complete(t.Context(), completionRequest); !errors.Is(err, completionErr) {
		t.Fatalf("Complete() error = %v, want %v", err, completionErr)
	}
}

func TestResponseFilesRejectsInvalidDeclarationsAtAccess(t *testing.T) {
	t.Parallel()
	loadRequest := cli.ResponseFileRequest{
		Path: "load",
	}
	completionRequest := cli.ResponseFileCompletionRequest{
		Path: "complete",
	}
	tests := []struct {
		name  string
		files ResponseFiles
		want  error
	}{
		{
			name: "load conflict",
			files: ResponseFiles{
				Loads:      map[cli.ResponseFileRequest]cli.ResponseFile{loadRequest: {}},
				LoadErrors: map[cli.ResponseFileRequest]error{loadRequest: errors.New("failure")},
			},
			want: errResponseFileRouteConflict,
		},
		{
			name: "nil load error",
			files: ResponseFiles{
				LoadErrors: map[cli.ResponseFileRequest]error{loadRequest: nil},
			},
			want: errResponseFileFailureRequired,
		},
		{
			name: "completion conflict",
			files: ResponseFiles{
				Completions:      map[cli.ResponseFileCompletionRequest]cli.CompletionContribution{completionRequest: {}},
				CompletionErrors: map[cli.ResponseFileCompletionRequest]error{completionRequest: errors.New("failure")},
			},
			want: errResponseCompletionRouteConflict,
		},
		{
			name: "nil completion error",
			files: ResponseFiles{
				CompletionErrors: map[cli.ResponseFileCompletionRequest]error{completionRequest: nil},
			},
			want: errResponseCompletionFailureRequired,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				got := recover()
				err, ok := got.(error)
				if !ok || !errors.Is(err, test.want) {
					t.Fatalf("Access() panic = %v, want %v", got, test.want)
				}
			}()
			_ = test.files.Access()
		})
	}
}

func TestResponseFilesSnapshotsCompletionContributionDeeply(t *testing.T) {
	t.Parallel()
	request := cli.ResponseFileCompletionRequest{
		Path:   "ar",
		Base:   "/workspace",
		Cursor: 2,
	}
	replace := &cli.CompletionRange{
		Start: 0,
		End:   2,
	}
	native := &cli.CompletionNativeIntent{
		Kind: cli.CompletionNativeFiles,
		Mode: cli.CompletionNativeAlways,
	}
	contribution := cli.CompletionContribution{
		Replace: replace,
		Candidates: []cli.CompletionCandidate{{
			Value:            "args.txt",
			Replacement:      []string{"args.txt"},
			SearchTerms:      []string{"args", "txt"},
			CommitCharacters: []rune{'/'},
		}},
		Groups: []cli.CompletionGroup{{
			ID:    "files",
			Label: "Files",
		}},
		Messages: []cli.CompletionMessage{{
			Kind: cli.CompletionMessageInformation,
			Text: "message",
		}},
		Native: native,
	}
	var requests ResponseFileRequests
	files := ResponseFiles{
		Completions: map[cli.ResponseFileCompletionRequest]cli.CompletionContribution{request: contribution},
		Requests:    &requests,
	}
	access := files.Access()

	replace.Start = 99
	native.Kind = cli.CompletionNativeHosts
	contribution.Candidates[0].Replacement[0] = "mutated"
	contribution.Groups[0].Label = "Mutated"

	got, err := access.Complete(t.Context(), request)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if got.Replace == nil || got.Replace.Start != 0 || got.Native == nil || got.Native.Kind != cli.CompletionNativeFiles || got.Candidates[0].Replacement[0] != "args.txt" || got.Groups[0].Label != "Files" {
		t.Fatalf("Complete() = %#v", got)
	}
	got.Candidates[0].Replacement[0] = "changed-result"
	again, err := access.Complete(t.Context(), request)
	if err != nil || again.Candidates[0].Replacement[0] != "args.txt" {
		t.Fatalf("second Complete() = %#v err %v", again, err)
	}
	if got := requests.Completions(); !slices.Equal(got, []cli.ResponseFileCompletionRequest{request, request}) {
		t.Fatalf("Completions() = %#v", got)
	}
}

func TestResponseFilesRecordsCanceledRequests(t *testing.T) {
	t.Parallel()
	request := cli.ResponseFileRequest{
		Path: "args.txt",
	}
	var requests ResponseFileRequests
	files := ResponseFiles{
		Loads: map[cli.ResponseFileRequest]cli.ResponseFile{request: {
			Content: []byte("ignored"),
		}},
		Requests: &requests,
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	got, err := files.Access().Load(ctx, request)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Load() error = %v, want %v", err, context.Canceled)
	}
	if got.Key != "" || got.Revision != (cli.InputRevision{}) || got.Location != "" || got.IncludeBase != "" || got.Sensitive || len(got.Content) != 0 {
		t.Fatalf("Load() = %#v, want zero response", got)
	}
	if observed := requests.Loads(); !slices.Equal(observed, []cli.ResponseFileRequest{request}) {
		t.Fatalf("Loads() = %#v", observed)
	}
}

func TestResponseFileRequestsConcurrentRecordingIsRaceSafe(t *testing.T) {
	t.Parallel()
	request := cli.ResponseFileRequest{
		Path: "args.txt",
		Base: "/workspace",
	}
	var requests ResponseFileRequests
	files := ResponseFiles{
		Loads: map[cli.ResponseFileRequest]cli.ResponseFile{request: {
			Content: []byte("ok"),
		}},
		Requests: &requests,
	}
	access := files.Access()

	const count = 128
	var wg sync.WaitGroup
	wg.Add(count)
	errs := make(chan error, count)
	for range count {
		go func() {
			defer wg.Done()
			file, err := access.Load(t.Context(), request)
			if err == nil && string(file.Content) != "ok" {
				err = errors.New("unexpected response content")
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Load() error = %v", err)
		}
	}
	if got := len(requests.Loads()); got != count {
		t.Fatalf("Loads count = %d, want %d", got, count)
	}
}
