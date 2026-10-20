package cli

import (
	"context"
	"errors"
	"unicode/utf8"
)

// ResponseFileAccess supplies operation-scoped host capabilities for response files.
// Load is required only when analysis actually encounters an include; Complete is
// optional and supports virtual/non-filesystem response-source namespaces.
type ResponseFileAccess struct {
	Load     ResponseFileLoader
	Complete ResponseFileCompleter
}

// ResponseFileLoader acquires one logical response source for request. It runs with the
// Commandment operation context; returned mutable Content is copied before later analysis can
// observe host mutation.
type ResponseFileLoader func(context.Context, ResponseFileRequest) (ResponseFile, error)

// ResponseFileRequest identifies one decoded include path and its resolution base.
// Top-level relative paths use the invocation working directory as Base; nested
// relative paths use the containing source's IncludeBase.
type ResponseFileRequest struct {
	Path string
	Base string
}

// ResponseFile is one host-provided logical response-source snapshot. Key is required
// operation-local cycle/snapshot identity; Revision is an optional opaque stable
// revision; Location is presentation metadata; IncludeBase resolves nested includes;
// Content is exact bytes; Sensitive enters monotonic input redaction.
type ResponseFile struct {
	Key         string
	Revision    InputRevision
	Location    string
	IncludeBase string
	Content     []byte
	Sensitive   bool
}

// ResponseFileCompleter completes an incomplete response-source path without loading
// it. Its contribution uses the ordinary completion model and failure rules.
type ResponseFileCompleter func(context.Context, ResponseFileCompletionRequest) (CompletionContribution, error)

// ResponseFileCompletionRequest describes the path portion of an incomplete response
// reference. Cursor and Replace are byte coordinates within Path; negative Cursor,
// replacement coordinates, or Limit values are invalid callback input.
type ResponseFileCompletionRequest struct {
	Path    string
	Base    string
	Cursor  int
	Replace CompletionRange
	Trigger CompletionTrigger
	Limit   int
}

// ResponseFileLimits bounds response-source expansion. For every field, zero selects
// the package default, a positive value is an exact override, and a negative value
// makes invocation input invalid. There is no unlimited mode.
type ResponseFileLimits struct {
	Depth             int
	SourceBytes       int
	TotalSourceBytes  int
	TotalTokens       int
	TokenBytes        int
	IncludeReferences int
}

func responseFileLoad(access ResponseFileAccess) analysisResponseFileLoad {
	if access.Load == nil {
		return nil
	}
	return func(ctx context.Context, path, base string) (responseFileSnapshot, error) {
		file, err := access.Load(ctx, ResponseFileRequest{
			Path: path,
			Base: base,
		})
		if err != nil {
			return responseFileSnapshot{}, err
		}
		snapshot := responseFileSnapshot{
			Key:         file.Key,
			Revision:    file.Revision.value,
			Location:    file.Location,
			IncludeBase: file.IncludeBase,
			Content:     append([]byte(nil), file.Content...),
			Sensitive:   file.Sensitive,
		}
		return snapshot, nil
	}
}

var errInvalidResponseFileCompletion = errors.New("cli: invalid response-file completion")

func responseFileComplete(ctx context.Context, access ResponseFileAccess, request ResponseFileCompletionRequest) (CompletionContribution, error) {
	if ctx == nil || request.Cursor < 0 || request.Cursor > len(request.Path) ||
		request.Replace.Start < 0 || request.Replace.End < request.Replace.Start ||
		request.Replace.End > len(request.Path) || request.Limit < 0 {
		return CompletionContribution{}, errInvalidResponseFileCompletion
	}
	if access.Complete == nil {
		return CompletionContribution{}, nil
	}
	contribution, err := access.Complete(ctx, request)
	if err != nil {
		return CompletionContribution{}, err
	}
	if contribution.Replace != nil {
		if contribution.Replace.Start < 0 || contribution.Replace.End < contribution.Replace.Start ||
			contribution.Replace.End > len(request.Path) {
			return CompletionContribution{}, errInvalidCompletionContribution
		}
	}
	return cloneCompletionContribution(contribution), nil
}

func cloneCompletionContribution(input CompletionContribution) CompletionContribution {
	result := input
	if input.Replace != nil {
		replacement := *input.Replace
		result.Replace = &replacement
	}
	result.Candidates = append([]CompletionCandidate(nil), input.Candidates...)
	for index := range result.Candidates {
		result.Candidates[index].Replacement = append([]string(nil), input.Candidates[index].Replacement...)
		result.Candidates[index].SearchTerms = append([]string(nil), input.Candidates[index].SearchTerms...)
		result.Candidates[index].CommitCharacters = append([]rune(nil), input.Candidates[index].CommitCharacters...)
	}
	result.Groups = append([]CompletionGroup(nil), input.Groups...)
	result.Messages = append([]CompletionMessage(nil), input.Messages...)
	if input.Native != nil {
		native := *input.Native
		result.Native = &native
	}
	return result
}

func responseFileDiagnosticFailure(operation *invocation, err error) error {
	responseErr, ok := errors.AsType[*responseFileError](err)
	if !ok {
		return err
	}
	code := responseFileDiagnosticCode(responseErr.Kind)
	if code == "" {
		return err
	}
	projectedDiagnostic := diagnostic{
		Code:             string(code),
		Message:          responseErr.Error(),
		FrameworkMessage: true,
	}
	if responseErr.Kind == responseFileErrorLimit {
		projectedDiagnostic.Reason = responseErr.Limit
	}
	payload := payload{
		Diagnostics:  []diagnostic{projectedDiagnostic},
		Presentation: operation.FrameworkPresentation(),
	}
	if failureSource, ok := responseFileFailureSourceOf(err); ok {
		text := string(failureSource.Content)
		projectSource := source{
			ID:        1,
			Kind:      InputKindResponseFile,
			Revision:  failureSource.Revision,
			Key:       failureSource.Key,
			Location:  failureSource.Location,
			Sensitive: failureSource.Sensitive,
			FlatText:  &text,
		}
		span := span{
			Source: 1,
			Entry:  -1,
			Start:  responseFilePosition(failureSource.Content, failureSource.Start),
			End:    responseFilePosition(failureSource.Content, failureSource.End),
		}
		payload.Sources = map[sourceID]source{1: projectSource}
		payload.Diagnostics[0].Source = 1
		payload.Diagnostics[0].Span = &span
		payload.Diagnostics[0].Sensitive = failureSource.Sensitive
	}
	return retainDiagnosticFailure(payload, err)
}

func responseFilePosition(content []byte, offset int) analysisPosition {
	position := analysisPosition{
		Offset: offset,
	}
	if offset < 0 || offset > len(content) || !utf8.Valid(content) {
		return position
	}
	line, column := 1, 1
	for index := 0; index < offset; {
		r, size := utf8.DecodeRune(content[index:])
		if index+size > offset {
			return position
		}
		switch r {
		case '\r':
			line++
			column = 1
			if index+size < offset && content[index+size] == '\n' {
				index += size + 1
				continue
			}
		case '\n':
			line++
			column = 1
		default:
			column++
		}
		index += size
	}
	position.Line = line
	position.Column = column
	return position
}

func responseFileDiagnosticCode(kind responseFileErrorKind) DiagnosticCode {
	switch kind {
	case responseFileErrorAccessUnavailable:
		return DiagnosticResponseFileAccessUnavailable
	case responseFileErrorLoad:
		return DiagnosticResponseFileLoadFailure
	case responseFileErrorText:
		return DiagnosticInvalidResponseFile
	case responseFileErrorSource:
		return DiagnosticInvalidResponseFileSource
	case responseFileErrorCycle:
		return DiagnosticResponseFileCycle
	case responseFileErrorLimit:
		return DiagnosticResponseFileLimitExceeded
	case responseFileErrorCancellation:
		return DiagnosticResponseFileCancellation
	default:
		return ""
	}
}
