package clitest

import (
	"context"
	"errors"
	"slices"
	"sync"

	"github.com/abstractionsinc/commandment/cli"
)

// ResponseFiles declares exact deterministic behavior for the public
// cli.ResponseFileAccess host protocol. Access validates and snapshots the
// declaration into isolated runtime callbacks. Requests optionally records
// observed host requests without becoming part of route declaration state.
// The zero value is valid and has no configured routes.
type ResponseFiles struct {
	Loads            map[cli.ResponseFileRequest]cli.ResponseFile
	LoadErrors       map[cli.ResponseFileRequest]error
	Completions      map[cli.ResponseFileCompletionRequest]cli.CompletionContribution
	CompletionErrors map[cli.ResponseFileCompletionRequest]error
	Requests         *ResponseFileRequests
}

// ResponseFileRequests records response-file host requests in invocation order.
// Its zero value is ready for use. A value must not be copied after first use.
type ResponseFileRequests struct {
	mu sync.Mutex

	loads       []cli.ResponseFileRequest
	completions []cli.ResponseFileCompletionRequest
}

// Access validates and snapshots the complete declaration into isolated public
// response-file host callbacks.
func (files ResponseFiles) Access() cli.ResponseFileAccess {
	state := snapshotResponseFiles(files)
	access := cli.ResponseFileAccess{
		Load:     state.load,
		Complete: state.complete,
	}
	return access
}

// Loads returns a detached snapshot of observed load requests in invocation order.
func (requests *ResponseFileRequests) Loads() []cli.ResponseFileRequest {
	if requests == nil {
		return nil
	}
	requests.mu.Lock()
	defer requests.mu.Unlock()
	return slices.Clone(requests.loads)
}

// Completions returns a detached snapshot of observed completion requests in invocation order.
func (requests *ResponseFileRequests) Completions() []cli.ResponseFileCompletionRequest {
	if requests == nil {
		return nil
	}
	requests.mu.Lock()
	defer requests.mu.Unlock()
	return slices.Clone(requests.completions)
}

func (requests *ResponseFileRequests) recordLoad(request cli.ResponseFileRequest) {
	if requests == nil {
		return
	}
	requests.mu.Lock()
	requests.loads = append(requests.loads, request)
	requests.mu.Unlock()
}

func (requests *ResponseFileRequests) recordCompletion(request cli.ResponseFileCompletionRequest) {
	if requests == nil {
		return
	}
	requests.mu.Lock()
	requests.completions = append(requests.completions, request)
	requests.mu.Unlock()
}

type responseFilesContextError struct {
	operation string
	cause     error
}

var (
	errResponseFileRouteConflict       = errors.New("clitest: response-file request has both a response and an error")
	errResponseFileRouteMissing        = errors.New("clitest: response-file request has no configured route")
	errResponseFileFailureRequired     = errors.New("clitest: response-file failure is nil")
	errResponseCompletionRouteConflict = errors.New(
		"clitest: response-file completion request has both a contribution and an error",
	)
	errResponseCompletionRouteMissing = errors.New(
		"clitest: response-file completion request has no configured route",
	)
	errResponseCompletionFailureRequired = errors.New("clitest: response-file completion failure is nil")
)

type responseFilesState struct {
	loads            map[cli.ResponseFileRequest]cli.ResponseFile
	loadErrors       map[cli.ResponseFileRequest]error
	completions      map[cli.ResponseFileCompletionRequest]cli.CompletionContribution
	completionErrors map[cli.ResponseFileCompletionRequest]error
	requests         *ResponseFileRequests
}

func snapshotResponseFiles(files ResponseFiles) *responseFilesState {
	var loads map[cli.ResponseFileRequest]cli.ResponseFile
	if len(files.Loads) != 0 {
		loads = make(map[cli.ResponseFileRequest]cli.ResponseFile, len(files.Loads))
	}
	for request, file := range files.Loads {
		if _, conflict := files.LoadErrors[request]; conflict {
			panic(errResponseFileRouteConflict)
		}
		loads[request] = cloneResponseFile(file)
	}

	var loadErrors map[cli.ResponseFileRequest]error
	if len(files.LoadErrors) != 0 {
		loadErrors = make(map[cli.ResponseFileRequest]error, len(files.LoadErrors))
	}
	for request, err := range files.LoadErrors {
		if err == nil {
			panic(errResponseFileFailureRequired)
		}
		loadErrors[request] = err
	}

	var completions map[cli.ResponseFileCompletionRequest]cli.CompletionContribution
	if len(files.Completions) != 0 {
		completions = make(map[cli.ResponseFileCompletionRequest]cli.CompletionContribution, len(files.Completions))
	}
	for request, contribution := range files.Completions {
		if _, conflict := files.CompletionErrors[request]; conflict {
			panic(errResponseCompletionRouteConflict)
		}
		completions[request] = cloneCompletionContribution(contribution)
	}

	var completionErrors map[cli.ResponseFileCompletionRequest]error
	if len(files.CompletionErrors) != 0 {
		completionErrors = make(map[cli.ResponseFileCompletionRequest]error, len(files.CompletionErrors))
	}
	for request, err := range files.CompletionErrors {
		if err == nil {
			panic(errResponseCompletionFailureRequired)
		}
		completionErrors[request] = err
	}

	state := &responseFilesState{
		loads:            loads,
		loadErrors:       loadErrors,
		completions:      completions,
		completionErrors: completionErrors,
		requests:         files.Requests,
	}
	return state
}

func cloneResponseFile(input cli.ResponseFile) cli.ResponseFile {
	result := input
	result.Content = slices.Clone(input.Content)
	return result
}

func cloneCompletionContribution(input cli.CompletionContribution) cli.CompletionContribution {
	result := input
	if input.Replace != nil {
		result.Replace = new(*input.Replace)
	}
	if input.Candidates != nil {
		result.Candidates = make([]cli.CompletionCandidate, len(input.Candidates))
		for index, candidate := range input.Candidates {
			result.Candidates[index] = cloneCompletionCandidate(candidate)
		}
	}
	result.Groups = slices.Clone(input.Groups)
	result.Messages = slices.Clone(input.Messages)
	if input.Native != nil {
		result.Native = new(*input.Native)
	}
	return result
}

func cloneCompletionCandidate(input cli.CompletionCandidate) cli.CompletionCandidate {
	result := input
	result.Replacement = slices.Clone(input.Replacement)
	result.SearchTerms = slices.Clone(input.SearchTerms)
	result.CommitCharacters = slices.Clone(input.CommitCharacters)
	return result
}

func (err responseFilesContextError) Error() string {
	return "clitest: response-files " + err.operation + ": " + err.cause.Error()
}

func (err responseFilesContextError) Unwrap() error { return err.cause }

func (state *responseFilesState) load(ctx context.Context, request cli.ResponseFileRequest) (cli.ResponseFile, error) {
	state.requests.recordLoad(request)
	if err := context.Cause(ctx); err != nil {
		return cli.ResponseFile{}, responseFilesContextError{
			operation: "load",
			cause:     err,
		}
	}
	if file, ok := state.loads[request]; ok {
		return cloneResponseFile(file), nil
	}
	if err, ok := state.loadErrors[request]; ok {
		return cli.ResponseFile{}, err
	}
	return cli.ResponseFile{}, errResponseFileRouteMissing
}

func (state *responseFilesState) complete(ctx context.Context, request cli.ResponseFileCompletionRequest) (cli.CompletionContribution, error) {
	state.requests.recordCompletion(request)
	if err := context.Cause(ctx); err != nil {
		return cli.CompletionContribution{}, responseFilesContextError{
			operation: "complete",
			cause:     err,
		}
	}
	if contribution, ok := state.completions[request]; ok {
		return cloneCompletionContribution(contribution), nil
	}
	if err, ok := state.completionErrors[request]; ok {
		return cli.CompletionContribution{}, err
	}
	return cli.CompletionContribution{}, errResponseCompletionRouteMissing
}
