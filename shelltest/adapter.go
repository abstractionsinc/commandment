package shelltest

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
)

const (
	shellAdapterWordLimit    = 65_536
	shellAdapterUnknownToken = "unknown"
	responseFileReadSubject  = "read completion response file"
)

type shellAdapterCodec uint8

const (
	shellAdapterCodecUnknown shellAdapterCodec = iota
	shellAdapterCodecJSON
	shellAdapterCodecRecords
)

type shellAdapterMatching uint8

const (
	shellAdapterMatchingPrefix shellAdapterMatching = iota + 1
	shellAdapterMatchingValuePrefix
	shellAdapterMatchingClient
)

type shellAdapterRequest struct {
	responseCodec shellAdapterCodec
	shellFamily   string
	shellVersion  string
	input         cli.CompletionInputObservation
	matching      shellAdapterMatching
	capabilities  cli.CompletionClientCapabilities
}

type shellAdapterItem struct {
	value            string
	replacement      []string
	replacementSet   bool
	spacing          cli.CompletionSpacing
	displayText      string
	description      string
	detail           string
	kind             cli.CompletionCandidateKind
	group            string
	matchRanges      []cli.CompletionRange
	commitCharacters []string
}

type shellAdapterResponse struct {
	failed     bool
	replace    cli.CompletionSpan
	spacing    cli.CompletionSpacing
	items      []shellAdapterItem
	groups     []cli.CompletionGroup
	messages   []cli.CompletionMessage
	native     []cli.CompletionNativeKind
	delegation *cli.CompletionDelegation
}

func decodeShellAdapterRequest(input []byte) (shellAdapterRequest, error) {
	reader := bufio.NewReader(bytes.NewReader(input))
	prelude, err := reader.ReadString('\n')
	if err != nil {
		return shellAdapterRequest{}, failureError{
			subject: "decode shell completion prelude",
			detail:  boundedFailureDetail(err.Error()),
		}
	}
	var codec shellAdapterCodec
	switch prelude {
	case "json\n":
		codec = shellAdapterCodecJSON
	case "records\n":
		codec = shellAdapterCodecRecords
	default:
		return shellAdapterRequest{}, failureError{subject: "invalid shell completion prelude"}
	}
	var request shellAdapterRequest
	switch codec {
	case shellAdapterCodecJSON:
		request, err = decodeShellAdapterJSONRequest(reader)
	case shellAdapterCodecRecords:
		request, err = decodeShellAdapterRecordsRequest(reader)
	default:
		panic("shelltest: invalid shell adapter codec")
	}
	if err != nil {
		return shellAdapterRequest{}, err
	}
	if request.responseCodec == shellAdapterCodecUnknown {
		request.responseCodec = codec
	}
	if request.responseCodec != shellAdapterCodecJSON && request.responseCodec != shellAdapterCodecRecords {
		return shellAdapterRequest{}, failureError{subject: "invalid shell completion response encoding"}
	}
	if request.shellFamily == "" {
		return shellAdapterRequest{}, failureError{subject: "invalid shell completion shell identity"}
	}
	if len(request.input.Words) == 0 || len(request.input.Words) > shellAdapterWordLimit {
		return shellAdapterRequest{}, failureError{subject: "invalid shell completion observed words"}
	}
	if request.input.Cursor.Word < 0 || request.input.Cursor.Word >= len(request.input.Words) || request.input.Cursor.Byte < 0 || request.input.Cursor.Byte > len(request.input.Words[request.input.Cursor.Word].Value) {
		return shellAdapterRequest{}, failureError{subject: "invalid shell completion cursor"}
	}
	if request.input.ExactArgvPrefix < 0 || request.input.ExactArgvPrefix > len(request.input.Words) {
		return shellAdapterRequest{}, failureError{subject: "invalid shell completion exact argv prefix"}
	}
	switch request.input.Topology {
	case cli.CompletionArgvTopologyExact:
		request.input.ExactArgvPrefix = len(request.input.Words)
	case cli.CompletionArgvTopologyPrefixExact:
	case cli.CompletionArgvTopologyUncertain:
		if request.input.ExactArgvPrefix != 0 {
			return shellAdapterRequest{}, failureError{subject: "invalid uncertain shell completion topology"}
		}
	default:
		return shellAdapterRequest{}, failureError{subject: "invalid shell completion topology"}
	}
	if request.input.Cursor.Space == cli.CompletionCursorSpaceArgumentBytes {
		word := request.input.Words[request.input.Cursor.Word]
		if request.input.Cursor.Word >= request.input.ExactArgvPrefix || word.Fidelity != cli.CompletionInputFidelityExact {
			return shellAdapterRequest{}, failureError{subject: "invalid argument-byte shell completion cursor"}
		}
	}
	if request.matching == 0 {
		request.matching = shellAdapterMatchingPrefix
	}
	return request, nil
}

func (request shellAdapterRequest) completionRequest(environ []string, workingDirectory, activationName, activationExpected string) (cli.CompletionRequest, error) {
	activationValue, present := environmentValue(environ, activationName)
	if !present || activationValue != activationExpected {
		return cli.CompletionRequest{}, failureError{subject: "completion activation value mismatch"}
	}
	semanticEnvironment := removeEnvironment(environ, activationName)
	env, err := cli.EnvFromEnviron(semanticEnvironment)
	if err != nil {
		return cli.CompletionRequest{}, failureError{
			subject: "invalid completion process environment",
			cause:   err,
		}
	}
	match := cli.CompletionMatchOptions{
		Mode: cli.CompletionMatchModePrefix,
		Case: cli.CompletionCaseSmart,
	}
	switch request.matching {
	case shellAdapterMatchingPrefix:
	case shellAdapterMatchingValuePrefix:
		match.Fields = []cli.CompletionMatchField{cli.CompletionMatchFieldValue}
	case shellAdapterMatchingClient:
		match.Mode = cli.CompletionMatchModeUnfiltered
		match.Fields = []cli.CompletionMatchField{cli.CompletionMatchFieldValue}
	default:
		return cli.CompletionRequest{}, failureError{subject: "invalid shell completion matching mode"}
	}
	observation := request.input
	observation.Words = slices.Clone(request.input.Words)
	completionRequest := cli.CompletionRequest{
		ObservedInput: &observation,
		Trigger: cli.CompletionTrigger{
			Kind: cli.CompletionTriggerInvoked,
		},
		Env:              env,
		WorkingDirectory: workingDirectory,
		ResponseFiles:    processResponseFileAccess(),
		Visibility:       cli.CompletionVisibilityPublic,
		Client: cli.CompletionClient{
			Name:    request.shellFamily,
			Version: request.shellVersion,
			Shell: cli.CompletionShell{
				Family:  cli.ShellFamily(request.shellFamily),
				Version: request.shellVersion,
			},
			Match:        match,
			Ranking:      cli.CompletionRankingModeRelevance,
			Capabilities: request.capabilities,
		},
		Limit: 0,
	}
	return completionRequest, nil
}

func projectShellAdapterResponse(completion cli.Completion, failed bool, capabilities cli.CompletionClientCapabilities) shellAdapterResponse {
	response := shellAdapterResponse{
		failed:  failed,
		replace: completion.Replace,
		spacing: completion.Spacing,
		items:   make([]shellAdapterItem, 0, len(completion.Items)),
	}
	for _, item := range completion.Items {
		projected := shellAdapterItem{
			value:          item.Value,
			spacing:        item.Spacing,
			displayText:    item.DisplayText,
			replacementSet: item.Replacement != nil,
		}
		if item.Replacement != nil {
			projected.replacement = slices.Clone(item.Replacement)
		}
		if capabilities.Descriptions {
			projected.description = item.Description
		}
		if capabilities.Details {
			projected.detail = item.Detail
		}
		if capabilities.Groups {
			projected.group = item.GroupID
		}
		if capabilities.CandidateKinds {
			projected.kind = item.Kind
		}
		if capabilities.MatchRanges {
			projected.matchRanges = slices.Clone(item.Match.Ranges)
		}
		if capabilities.CommitCharacters {
			projected.commitCharacters = make([]string, 0, len(item.CommitCharacters))
			for _, character := range item.CommitCharacters {
				projected.commitCharacters = append(projected.commitCharacters, string(character))
			}
		}
		response.items = append(response.items, projected)
	}
	if capabilities.Groups {
		response.groups = slices.Clone(completion.Groups)
	}
	if capabilities.Messages {
		response.messages = slices.Clone(completion.Messages)
	}
	for _, native := range completion.NativeRequests {
		if shellAdapterNativeCapabilityAllows(capabilities.Native, native.Kind) {
			response.native = append(response.native, native.Kind)
		}
	}
	if capabilities.Delegation && completion.Delegation != nil {
		delegation := *completion.Delegation
		delegation.Args = slices.Clone(completion.Delegation.Args)
		response.delegation = &delegation
	}
	return response
}

func encodeShellAdapterResponse(writer io.Writer, codec shellAdapterCodec, response shellAdapterResponse) error {
	switch codec {
	case shellAdapterCodecJSON:
		return encodeShellAdapterJSONResponse(writer, response)
	case shellAdapterCodecRecords:
		return encodeShellAdapterRecordsResponse(writer, response)
	default:
		return failureError{subject: "invalid shell completion response encoding"}
	}
}

func shellAdapterNativeCapabilityAllows(capabilities cli.CompletionNativeCapabilities, kind cli.CompletionNativeKind) bool {
	switch kind {
	case cli.CompletionNativePaths:
		return capabilities.Paths
	case cli.CompletionNativeFiles:
		return capabilities.Files
	case cli.CompletionNativeDirectories:
		return capabilities.Directories
	case cli.CompletionNativeExecutables:
		return capabilities.Executables
	case cli.CompletionNativeUsers:
		return capabilities.Users
	case cli.CompletionNativeHosts:
		return capabilities.Hosts
	case cli.CompletionNativeEnvironmentVariables:
		return capabilities.EnvironmentVariables
	default:
		return false
	}
}

func applyShellAdapterCapability(capabilities *cli.CompletionClientCapabilities, token string) {
	switch token {
	case "descriptions":
		capabilities.Descriptions = true
	case "details":
		capabilities.Details = true
	case "groups":
		capabilities.Groups = true
	case "candidate-kinds":
		capabilities.CandidateKinds = true
	case "match-ranges":
		capabilities.MatchRanges = true
	case "commit-characters":
		capabilities.CommitCharacters = true
	case "messages":
		capabilities.Messages = true
	case "delegation":
		capabilities.Delegation = true
	}
}

func applyShellAdapterNativeCapability(capabilities *cli.CompletionClientCapabilities, token string) {
	switch token {
	case "paths":
		capabilities.Native.Paths = true
	case "files":
		capabilities.Native.Files = true
	case "directories":
		capabilities.Native.Directories = true
	case "executables":
		capabilities.Native.Executables = true
	case "users":
		capabilities.Native.Users = true
	case "hosts":
		capabilities.Native.Hosts = true
	case "environment-variables":
		capabilities.Native.EnvironmentVariables = true
	}
}

func shellAdapterCodecFromToken(token string) shellAdapterCodec {
	switch token {
	case "json":
		return shellAdapterCodecJSON
	case "records":
		return shellAdapterCodecRecords
	default:
		return shellAdapterCodecUnknown
	}
}

func shellAdapterTopologyFromToken(token string) (cli.CompletionArgvTopology, bool) {
	switch token {
	case "exact":
		return cli.CompletionArgvTopologyExact, true
	case "prefix-exact":
		return cli.CompletionArgvTopologyPrefixExact, true
	case "uncertain":
		return cli.CompletionArgvTopologyUncertain, true
	default:
		return cli.CompletionArgvTopologyUnknown, false
	}
}

func shellAdapterFidelityFromToken(token string) (cli.CompletionInputFidelity, bool) {
	switch token {
	case "exact":
		return cli.CompletionInputFidelityExact, true
	case "lexical":
		return cli.CompletionInputFidelityLexical, true
	case "expanded":
		return cli.CompletionInputFidelityExpanded, true
	default:
		return cli.CompletionInputFidelityUnknown, false
	}
}

func shellAdapterCursorSpaceFromToken(token string) (cli.CompletionCursorSpace, bool) {
	switch token {
	case "argument-bytes":
		return cli.CompletionCursorSpaceArgumentBytes, true
	case "observed-word-bytes":
		return cli.CompletionCursorSpaceObservedWordBytes, true
	default:
		return cli.CompletionCursorSpaceUnknown, false
	}
}

func shellAdapterMatchingFromToken(token string) (shellAdapterMatching, bool) {
	switch token {
	case "prefix":
		return shellAdapterMatchingPrefix, true
	case "value-prefix":
		return shellAdapterMatchingValuePrefix, true
	case "client":
		return shellAdapterMatchingClient, true
	default:
		return 0, false
	}
}

func processResponseFileAccess() cli.ResponseFileAccess {
	loader := func(ctx context.Context, request cli.ResponseFileRequest) (cli.ResponseFile, error) {
		if err := ctx.Err(); err != nil {
			return cli.ResponseFile{}, failureError{
				subject: responseFileReadSubject,
				cause:   err,
			}
		}
		path := request.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(request.Base, path)
		}
		path, err := filepath.Abs(path)
		if err != nil {
			return cli.ResponseFile{}, failureError{
				subject: responseFileReadSubject,
				cause:   err,
			}
		}
		// #nosec G304 -- the public ResponseFileAccess request supplies the exact process path to acquire.
		data, err := os.ReadFile(path)
		if err != nil {
			return cli.ResponseFile{}, failureError{
				subject: responseFileReadSubject,
				detail:  path,
				cause:   err,
			}
		}
		if err := ctx.Err(); err != nil {
			return cli.ResponseFile{}, failureError{
				subject: responseFileReadSubject,
				cause:   err,
			}
		}
		result := cli.ResponseFile{
			Key:         path,
			Location:    path,
			IncludeBase: filepath.Dir(path),
			Content:     data,
		}
		return result, nil
	}
	return cli.ResponseFileAccess{Load: loader}
}

func removeEnvironment(environment []string, name string) []string {
	result := make([]string, 0, len(environment))
	for _, entry := range environment {
		entryName, _, ok := strings.Cut(entry, "=")
		if ok && (entryName == name || (runtime.GOOS == windowsOS && strings.EqualFold(entryName, name))) {
			continue
		}
		result = append(result, entry)
	}
	return result
}

var errShellAdapter = errors.New("shelltest: invalid shell completion adapter protocol")
