package cli

import (
	"errors"
	"io"
	"unicode/utf8"
)

const (
	shellCompletionAdapterFieldTokenDescriptions = "descriptions"
	shellCompletionAdapterFieldTokenDetails      = "details"
	shellCompletionAdapterFieldTokenGroups       = "groups"
	shellCompletionAdapterFieldTokenMessages     = "messages"
	shellCompletionAdapterFieldTokenClient       = "client"
)

var errInvalidShellCompletionAdapter = errors.New("cli: invalid shell completion adapter protocol")

type shellCompletionAdapterError struct {
	detail string
	cause  error
}

func (err *shellCompletionAdapterError) Error() string {
	text := errInvalidShellCompletionAdapter.Error()
	if err.detail != "" {
		text += ": " + err.detail
	}
	if err.cause != nil {
		text += ": " + err.cause.Error()
	}
	return text
}

func (*shellCompletionAdapterError) Is(target error) bool {
	return target == errInvalidShellCompletionAdapter
}

func (err *shellCompletionAdapterError) Unwrap() error { return err.cause }

type shellCompletionAdapterCodec string

const (
	shellCompletionAdapterCodecRecords shellCompletionAdapterCodec = "records"
	shellCompletionAdapterCodecJSON    shellCompletionAdapterCodec = "json"
)

type shellCompletionAdapterMatching uint8

const (
	shellCompletionAdapterMatchingPrefix shellCompletionAdapterMatching = iota + 1
	shellCompletionAdapterMatchingValuePrefix
	shellCompletionAdapterMatchingClient
)

const (
	shellCompletionAdapterTokenDelegation           = "delegation"
	shellCompletionAdapterTokenPaths                = "paths"
	shellCompletionAdapterTokenFiles                = "files"
	shellCompletionAdapterTokenDirectories          = "directories"
	shellCompletionAdapterTokenExecutables          = "executables"
	shellCompletionAdapterTokenUsers                = "users"
	shellCompletionAdapterTokenHosts                = "hosts"
	shellCompletionAdapterTokenEnvironmentVariables = "environment-variables"
	shellCompletionAdapterTokenExact                = "exact"
	shellCompletionAdapterTokenPrefixExact          = "prefix-exact"
	shellCompletionAdapterTokenUncertain            = "uncertain"
	shellCompletionAdapterTokenLexical              = "lexical"
	shellCompletionAdapterTokenExpanded             = "expanded"
	shellCompletionAdapterTokenArgumentBytes        = "argument-bytes"
	shellCompletionAdapterTokenObservedWordBytes    = "observed-word-bytes"
	shellCompletionAdapterTokenPrefix               = "prefix"
	shellCompletionAdapterTokenUnknown              = "unknown"
)

type shellCompletionAdapterRequest struct {
	responseCodec shellCompletionAdapterCodec
	shellFamily   string
	shellVersion  string
	input         CompletionInputObservation
	matching      shellCompletionAdapterMatching
	capabilities  CompletionClientCapabilities
}

func (request shellCompletionAdapterRequest) completionRequest() CompletionRequest {
	match := CompletionMatchOptions{
		Mode: CompletionMatchModePrefix,
		Case: CompletionCaseSmart,
	}
	switch request.matching {
	case shellCompletionAdapterMatchingPrefix:
	case shellCompletionAdapterMatchingValuePrefix:
		match.Fields = []CompletionMatchField{CompletionMatchFieldValue}
	case shellCompletionAdapterMatchingClient:
		match.Mode = CompletionMatchModeUnfiltered
		match.Fields = []CompletionMatchField{CompletionMatchFieldValue}
	}
	observation := request.input
	observation.Words = append([]CompletionObservedWord(nil), request.input.Words...)
	result := CompletionRequest{
		ObservedInput: &observation,
		Trigger: CompletionTrigger{
			Kind: CompletionTriggerInvoked,
		},
		Visibility: CompletionVisibilityPublic,
		Client: CompletionClient{
			Name:    request.shellFamily,
			Version: request.shellVersion,
			Shell: CompletionShell{
				Family:  ShellFamily(request.shellFamily),
				Version: request.shellVersion,
			},
			Match:        match,
			Ranking:      CompletionRankingModeRelevance,
			Capabilities: request.capabilities,
		},
		Limit: 0,
	}

	return result
}

type shellCompletionAdapterItem struct {
	Value            string
	Replacement      []string
	ReplacementSet   bool
	Spacing          CompletionSpacing
	DisplayText      string
	Description      string
	Detail           string
	Kind             CompletionCandidateKind
	Group            string
	MatchRanges      []CompletionRange
	CommitCharacters []string
}

type shellCompletionAdapterResponse struct {
	Failed     bool
	Replace    CompletionSpan
	Spacing    CompletionSpacing
	Items      []shellCompletionAdapterItem
	Groups     []CompletionGroup
	Messages   []CompletionMessage
	Native     []CompletionNativeKind
	Delegation *CompletionDelegation
}

func projectShellCompletionAdapterResponse(completion Completion, failed bool, capabilities CompletionClientCapabilities) shellCompletionAdapterResponse {
	response := shellCompletionAdapterResponse{
		Failed:  failed,
		Replace: completion.Replace,
		Spacing: completion.Spacing,
		Items:   make([]shellCompletionAdapterItem, 0, len(completion.Items)),
	}
	for _, item := range completion.Items {
		projected := shellCompletionAdapterItem{
			Value:          item.Value,
			Spacing:        item.Spacing,
			DisplayText:    item.DisplayText,
			ReplacementSet: item.Replacement != nil,
		}
		if item.Replacement != nil {
			projected.Replacement = append([]string(nil), item.Replacement...)
		}
		if capabilities.Descriptions {
			projected.Description = item.Description
		}
		if capabilities.Details {
			projected.Detail = item.Detail
		}
		if capabilities.Groups {
			projected.Group = item.GroupID
		}
		if capabilities.CandidateKinds {
			projected.Kind = item.Kind
		}
		if capabilities.MatchRanges {
			projected.MatchRanges = append([]CompletionRange(nil), item.Match.Ranges...)
		}
		if capabilities.CommitCharacters {
			projected.CommitCharacters = make([]string, 0, len(item.CommitCharacters))
			for _, character := range item.CommitCharacters {
				projected.CommitCharacters = append(projected.CommitCharacters, string(character))
			}
		}
		response.Items = append(response.Items, projected)
	}
	if capabilities.Groups {
		response.Groups = append([]CompletionGroup(nil), completion.Groups...)
	}
	if capabilities.Messages {
		response.Messages = append([]CompletionMessage(nil), completion.Messages...)
	}
	for _, native := range completion.NativeRequests {
		if shellCompletionNativeCapabilityAllows(capabilities.Native, native.Kind) {
			response.Native = append(response.Native, native.Kind)
		}
	}
	if capabilities.Delegation && completion.Delegation != nil {
		copyDelegation := *completion.Delegation
		copyDelegation.Args = append([]string(nil), completion.Delegation.Args...)
		response.Delegation = &copyDelegation
	}
	return response
}

func shellCompletionNativeCapabilityAllows(capabilities CompletionNativeCapabilities, kind CompletionNativeKind) bool {
	switch kind {
	case CompletionNativePaths:
		return capabilities.Paths
	case CompletionNativeFiles:
		return capabilities.Files
	case CompletionNativeDirectories:
		return capabilities.Directories
	case CompletionNativeExecutables:
		return capabilities.Executables
	case CompletionNativeUsers:
		return capabilities.Users
	case CompletionNativeHosts:
		return capabilities.Hosts
	case CompletionNativeEnvironmentVariables:
		return capabilities.EnvironmentVariables
	default:
		return false
	}
}

func decodeShellCompletionAdapterRequest(reader io.Reader, codec shellCompletionAdapterCodec) (shellCompletionAdapterRequest, error) {
	if reader == nil {
		return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
	}
	var request shellCompletionAdapterRequest
	var err error
	switch codec {
	case shellCompletionAdapterCodecJSON:
		request, err = decodeShellCompletionAdapterJSONRequest(reader)
	case shellCompletionAdapterCodecRecords:
		request, err = decodeShellCompletionAdapterRecordsRequest(reader)
	default:
		err = errInvalidShellCompletionAdapter
	}
	if err != nil {
		return shellCompletionAdapterRequest{}, err
	}
	if request.responseCodec == "" {
		request.responseCodec = codec
	}
	if request.responseCodec != shellCompletionAdapterCodecJSON &&
		request.responseCodec != shellCompletionAdapterCodecRecords {
		return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
	}
	if request.shellFamily == "" || !utf8.ValidString(request.shellFamily) || !utf8.ValidString(request.shellVersion) {
		return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
	}
	if request.matching == 0 {
		request.matching = shellCompletionAdapterMatchingPrefix
	}
	if _, _, err := completionAnalysisInput(CompletionRequest{
		ObservedInput: &request.input,
	}); err != nil {
		return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
	}
	return request, nil
}

func encodeShellCompletionAdapterResponse(writer io.Writer, codec shellCompletionAdapterCodec, response shellCompletionAdapterResponse) error {
	if writer == nil {
		return errInvalidShellCompletionAdapter
	}
	switch codec {
	case shellCompletionAdapterCodecJSON:
		return encodeShellCompletionAdapterJSONResponse(writer, response)
	case shellCompletionAdapterCodecRecords:
		return encodeShellCompletionAdapterRecordsResponse(writer, response)
	default:
		return errInvalidShellCompletionAdapter
	}
}

func applyShellCompletionAdapterCapabilities(capabilities *CompletionClientCapabilities, presentation, native []string) {
	for _, token := range presentation {
		switch token {
		case shellCompletionAdapterFieldTokenDescriptions:
			capabilities.Descriptions = true
		case shellCompletionAdapterFieldTokenDetails:
			capabilities.Details = true
		case shellCompletionAdapterFieldTokenGroups:
			capabilities.Groups = true
		case "candidate-kinds":
			capabilities.CandidateKinds = true
		case "match-ranges":
			capabilities.MatchRanges = true
		case "commit-characters":
			capabilities.CommitCharacters = true
		case shellCompletionAdapterFieldTokenMessages:
			capabilities.Messages = true
		case shellCompletionAdapterTokenDelegation:
			capabilities.Delegation = true
		}
	}
	for _, token := range native {
		switch token {
		case shellCompletionAdapterTokenPaths:
			capabilities.Native.Paths = true
		case shellCompletionAdapterTokenFiles:
			capabilities.Native.Files = true
		case shellCompletionAdapterTokenDirectories:
			capabilities.Native.Directories = true
		case shellCompletionAdapterTokenExecutables:
			capabilities.Native.Executables = true
		case shellCompletionAdapterTokenUsers:
			capabilities.Native.Users = true
		case shellCompletionAdapterTokenHosts:
			capabilities.Native.Hosts = true
		case shellCompletionAdapterTokenEnvironmentVariables:
			capabilities.Native.EnvironmentVariables = true
		}
	}
}

func shellCompletionAdapterCodecFromToken(token string) (shellCompletionAdapterCodec, bool) {
	switch token {
	case string(shellCompletionAdapterCodecRecords):
		return shellCompletionAdapterCodecRecords, true
	case string(shellCompletionAdapterCodecJSON):
		return shellCompletionAdapterCodecJSON, true
	default:
		return "", false
	}
}

func shellCompletionAdapterTopologyFromToken(token string) (CompletionArgvTopology, bool) {
	switch token {
	case shellCompletionAdapterTokenExact:
		return CompletionArgvTopologyExact, true
	case shellCompletionAdapterTokenPrefixExact:
		return CompletionArgvTopologyPrefixExact, true
	case shellCompletionAdapterTokenUncertain:
		return CompletionArgvTopologyUncertain, true
	default:
		return CompletionArgvTopologyUnknown, false
	}
}

func shellCompletionAdapterFidelityFromToken(token string) (CompletionInputFidelity, bool) {
	switch token {
	case shellCompletionAdapterTokenExact:
		return CompletionInputFidelityExact, true
	case shellCompletionAdapterTokenLexical:
		return CompletionInputFidelityLexical, true
	case shellCompletionAdapterTokenExpanded:
		return CompletionInputFidelityExpanded, true
	default:
		return CompletionInputFidelityUnknown, false
	}
}

func shellCompletionAdapterCursorSpaceFromToken(token string) (CompletionCursorSpace, bool) {
	switch token {
	case shellCompletionAdapterTokenArgumentBytes:
		return CompletionCursorSpaceArgumentBytes, true
	case shellCompletionAdapterTokenObservedWordBytes:
		return CompletionCursorSpaceObservedWordBytes, true
	default:
		return CompletionCursorSpaceUnknown, false
	}
}

func shellCompletionAdapterMatchingFromToken(token string) (shellCompletionAdapterMatching, bool) {
	switch token {
	case shellCompletionAdapterTokenPrefix:
		return shellCompletionAdapterMatchingPrefix, true
	case "value-prefix":
		return shellCompletionAdapterMatchingValuePrefix, true
	case shellCompletionAdapterFieldTokenClient:
		return shellCompletionAdapterMatchingClient, true
	default:
		return 0, false
	}
}

func shellCompletionAdapterSpacingToken(value CompletionSpacing) (string, bool) {
	switch value {
	case CompletionSpacingDefault:
		return "default", true
	case CompletionSpacingAppendSpace:
		return "append-space", true
	case CompletionSpacingNoSpace:
		return "no-space", true
	default:
		return "", false
	}
}

func shellCompletionAdapterCandidateKindToken(value CompletionCandidateKind) (string, bool) {
	switch value {
	case CompletionCandidateUnknown:
		return shellCompletionAdapterTokenUnknown, true
	case CompletionCandidateValue,
		CompletionCandidateCommand,
		CompletionCandidateCommandAlias,
		CompletionCandidateCommandShortcut,
		CompletionCandidateAction,
		CompletionCandidateFlag,
		CompletionCandidateKeyword,
		CompletionCandidateFile,
		CompletionCandidateDirectory,
		CompletionCandidateExecutable,
		CompletionCandidateResource,
		CompletionCandidateNamespace,
		CompletionCandidateUser,
		CompletionCandidateHost,
		CompletionCandidateEnvironmentVariable,
		CompletionCandidateURL:
		return string(value), true
	default:
		return "", false
	}
}

func shellCompletionAdapterMessageKindToken(value CompletionMessageKind) (string, bool) {
	switch value {
	case CompletionMessageUnknown:
		return shellCompletionAdapterTokenUnknown, true
	case CompletionMessageInformation:
		return "information", true
	case CompletionMessageWarning:
		return "warning", true
	case CompletionMessageError:
		return "error", true
	default:
		return "", false
	}
}

func shellCompletionAdapterNativeKindToken(value CompletionNativeKind) (string, bool) {
	switch value {
	case CompletionNativeUnknown:
		return shellCompletionAdapterTokenUnknown, true
	case CompletionNativePaths:
		return shellCompletionAdapterTokenPaths, true
	case CompletionNativeFiles:
		return shellCompletionAdapterTokenFiles, true
	case CompletionNativeDirectories:
		return shellCompletionAdapterTokenDirectories, true
	case CompletionNativeExecutables:
		return shellCompletionAdapterTokenExecutables, true
	case CompletionNativeUsers:
		return shellCompletionAdapterTokenUsers, true
	case CompletionNativeHosts:
		return shellCompletionAdapterTokenHosts, true
	case CompletionNativeEnvironmentVariables:
		return shellCompletionAdapterTokenEnvironmentVariables, true
	default:
		return "", false
	}
}
