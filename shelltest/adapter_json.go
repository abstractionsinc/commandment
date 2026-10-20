package shelltest

import (
	"bytes"
	"encoding/json"
	"encoding/json/jsontext"
	"errors"
	"io"
	"slices"
	"strconv"
	"unicode/utf8"

	"github.com/abstractionsinc/commandment/cli"
)

func decodeShellAdapterJSONRequest(reader io.Reader) (shellAdapterRequest, error) {
	data, err := io.ReadAll(io.LimitReader(reader, bridgeRequestFrameLimit+1))
	if err != nil {
		return shellAdapterRequest{}, failureError{
			subject: "read shell completion JSON",
			detail:  boundedFailureDetail(err.Error()),
		}
	}
	if len(data) > bridgeRequestFrameLimit || !utf8.Valid(data) {
		return shellAdapterRequest{}, errShellAdapter
	}

	decoder := jsontext.NewDecoder(bytes.NewReader(data))
	request := shellAdapterRequest{
		input: cli.CompletionInputObservation{
			Topology: cli.CompletionArgvTopologyUncertain,
			Cursor: cli.CompletionObservedCursor{
				Space: cli.CompletionCursorSpaceObservedWordBytes,
			},
		},
		matching: shellAdapterMatchingPrefix,
	}
	start, err := decoder.ReadToken()
	if err != nil || start.Kind() != jsontext.KindBeginObject {
		return shellAdapterRequest{}, errShellAdapter
	}

	var shellSeen, inputSeen, responseSeen, matchingSeen, capabilitiesSeen, nativeCapabilitiesSeen bool
	for decoder.PeekKind() != jsontext.KindEndObject {
		name, err := decoder.ReadToken()
		if err != nil || name.Kind() != jsontext.KindString {
			return shellAdapterRequest{}, errShellAdapter
		}
		switch name.String() {
		case "shell":
			if shellSeen {
				return shellAdapterRequest{}, errShellAdapter
			}
			shellSeen = true
			family, version, err := decodeShellAdapterJSONShell(decoder)
			if err != nil {
				return shellAdapterRequest{}, err
			}
			request.shellFamily = family
			request.shellVersion = version
		case "input":
			if inputSeen {
				return shellAdapterRequest{}, errShellAdapter
			}
			inputSeen = true
			input, err := decodeShellAdapterJSONInput(decoder)
			if err != nil {
				return shellAdapterRequest{}, err
			}
			request.input = input
		case "response_encoding":
			if responseSeen {
				return shellAdapterRequest{}, errShellAdapter
			}
			responseSeen = true
			value, err := readShellAdapterJSONString(decoder)
			if err != nil {
				return shellAdapterRequest{}, err
			}
			request.responseCodec = shellAdapterCodecFromToken(value)
			if request.responseCodec == shellAdapterCodecUnknown {
				return shellAdapterRequest{}, errShellAdapter
			}
		case "matching":
			if matchingSeen {
				return shellAdapterRequest{}, errShellAdapter
			}
			matchingSeen = true
			value, err := readShellAdapterJSONString(decoder)
			if err != nil {
				return shellAdapterRequest{}, err
			}
			var ok bool
			request.matching, ok = shellAdapterMatchingFromToken(value)
			if !ok {
				return shellAdapterRequest{}, errShellAdapter
			}
		case "capabilities":
			if capabilitiesSeen {
				return shellAdapterRequest{}, errShellAdapter
			}
			capabilitiesSeen = true
			if err := decodeShellAdapterJSONCapabilities(decoder, &request.capabilities, false); err != nil {
				return shellAdapterRequest{}, err
			}
		case "native_capabilities":
			if nativeCapabilitiesSeen {
				return shellAdapterRequest{}, errShellAdapter
			}
			nativeCapabilitiesSeen = true
			if err := decodeShellAdapterJSONCapabilities(decoder, &request.capabilities, true); err != nil {
				return shellAdapterRequest{}, err
			}
		default:
			return shellAdapterRequest{}, errShellAdapter
		}
	}
	if _, err := decoder.ReadToken(); err != nil {
		return shellAdapterRequest{}, errShellAdapter
	}
	if _, err := decoder.ReadToken(); !errors.Is(err, io.EOF) {
		return shellAdapterRequest{}, errShellAdapter
	}
	if !shellSeen || !inputSeen || request.shellFamily == "" || len(request.input.Words) == 0 {
		return shellAdapterRequest{}, errShellAdapter
	}
	return request, nil
}

func decodeShellAdapterJSONShell(decoder *jsontext.Decoder) (string, string, error) {
	start, err := decoder.ReadToken()
	if err != nil || start.Kind() != jsontext.KindBeginObject {
		return "", "", errShellAdapter
	}
	var family, version string
	var familySeen, versionSeen bool
	for decoder.PeekKind() != jsontext.KindEndObject {
		name, err := decoder.ReadToken()
		if err != nil || name.Kind() != jsontext.KindString {
			return "", "", errShellAdapter
		}
		switch name.String() {
		case "family":
			if familySeen {
				return "", "", errShellAdapter
			}
			familySeen = true
			family, err = readShellAdapterJSONString(decoder)
			if err != nil {
				return "", "", err
			}
		case "version":
			if versionSeen {
				return "", "", errShellAdapter
			}
			versionSeen = true
			version, err = readShellAdapterJSONString(decoder)
			if err != nil {
				return "", "", err
			}
		default:
			return "", "", errShellAdapter
		}
	}
	if _, err := decoder.ReadToken(); err != nil || !familySeen || !versionSeen || family == "" {
		return "", "", errShellAdapter
	}
	return family, version, nil
}

func decodeShellAdapterJSONInput(decoder *jsontext.Decoder) (cli.CompletionInputObservation, error) {
	input := cli.CompletionInputObservation{
		Topology: cli.CompletionArgvTopologyUncertain,
		Cursor: cli.CompletionObservedCursor{
			Space: cli.CompletionCursorSpaceObservedWordBytes,
		},
	}
	start, err := decoder.ReadToken()
	if err != nil || start.Kind() != jsontext.KindBeginObject {
		return cli.CompletionInputObservation{}, errShellAdapter
	}
	var wordsSeen, cursorSeen, topologySeen, prefixSeen bool
	for decoder.PeekKind() != jsontext.KindEndObject {
		name, err := decoder.ReadToken()
		if err != nil || name.Kind() != jsontext.KindString {
			return cli.CompletionInputObservation{}, errShellAdapter
		}
		switch name.String() {
		case "words":
			if wordsSeen {
				return cli.CompletionInputObservation{}, errShellAdapter
			}
			wordsSeen = true
			words, err := decodeShellAdapterJSONWords(decoder)
			if err != nil {
				return cli.CompletionInputObservation{}, err
			}
			input.Words = words
		case "cursor":
			if cursorSeen {
				return cli.CompletionInputObservation{}, errShellAdapter
			}
			cursorSeen = true
			cursor, err := decodeShellAdapterJSONCursor(decoder)
			if err != nil {
				return cli.CompletionInputObservation{}, err
			}
			input.Cursor = cursor
		case "topology":
			if topologySeen {
				return cli.CompletionInputObservation{}, errShellAdapter
			}
			topologySeen = true
			value, err := readShellAdapterJSONString(decoder)
			if err != nil {
				return cli.CompletionInputObservation{}, err
			}
			var ok bool
			input.Topology, ok = shellAdapterTopologyFromToken(value)
			if !ok {
				return cli.CompletionInputObservation{}, errShellAdapter
			}
		case "exact_argv_prefix":
			if prefixSeen {
				return cli.CompletionInputObservation{}, errShellAdapter
			}
			prefixSeen = true
			value, err := readShellAdapterJSONNonnegativeInt(decoder)
			if err != nil {
				return cli.CompletionInputObservation{}, err
			}
			input.ExactArgvPrefix = value
		default:
			return cli.CompletionInputObservation{}, errShellAdapter
		}
	}
	if _, err := decoder.ReadToken(); err != nil || !wordsSeen || !cursorSeen || len(input.Words) == 0 {
		return cli.CompletionInputObservation{}, errShellAdapter
	}
	return input, nil
}

func decodeShellAdapterJSONWords(decoder *jsontext.Decoder) ([]cli.CompletionObservedWord, error) {
	start, err := decoder.ReadToken()
	if err != nil || start.Kind() != jsontext.KindBeginArray {
		return nil, errShellAdapter
	}
	words := make([]cli.CompletionObservedWord, 0)
	for decoder.PeekKind() != jsontext.KindEndArray {
		if len(words) >= shellAdapterWordLimit {
			return nil, errShellAdapter
		}
		word, err := decodeShellAdapterJSONWord(decoder)
		if err != nil {
			return nil, err
		}
		words = append(words, word)
	}
	if _, err := decoder.ReadToken(); err != nil {
		return nil, errShellAdapter
	}
	return words, nil
}

func decodeShellAdapterJSONWord(decoder *jsontext.Decoder) (cli.CompletionObservedWord, error) {
	word := cli.CompletionObservedWord{Fidelity: cli.CompletionInputFidelityLexical}
	start, err := decoder.ReadToken()
	if err != nil || start.Kind() != jsontext.KindBeginObject {
		return cli.CompletionObservedWord{}, errShellAdapter
	}
	var valueSeen, fidelitySeen bool
	for decoder.PeekKind() != jsontext.KindEndObject {
		name, err := decoder.ReadToken()
		if err != nil || name.Kind() != jsontext.KindString {
			return cli.CompletionObservedWord{}, errShellAdapter
		}
		switch name.String() {
		case "value":
			if valueSeen {
				return cli.CompletionObservedWord{}, errShellAdapter
			}
			valueSeen = true
			word.Value, err = readShellAdapterJSONString(decoder)
			if err != nil {
				return cli.CompletionObservedWord{}, err
			}
		case "fidelity":
			if fidelitySeen {
				return cli.CompletionObservedWord{}, errShellAdapter
			}
			fidelitySeen = true
			value, err := readShellAdapterJSONString(decoder)
			if err != nil {
				return cli.CompletionObservedWord{}, err
			}
			var ok bool
			word.Fidelity, ok = shellAdapterFidelityFromToken(value)
			if !ok {
				return cli.CompletionObservedWord{}, errShellAdapter
			}
		default:
			return cli.CompletionObservedWord{}, errShellAdapter
		}
	}
	if _, err := decoder.ReadToken(); err != nil || !valueSeen {
		return cli.CompletionObservedWord{}, errShellAdapter
	}
	return word, nil
}

func decodeShellAdapterJSONCursor(decoder *jsontext.Decoder) (cli.CompletionObservedCursor, error) {
	cursor := cli.CompletionObservedCursor{Space: cli.CompletionCursorSpaceObservedWordBytes}
	start, err := decoder.ReadToken()
	if err != nil || start.Kind() != jsontext.KindBeginObject {
		return cli.CompletionObservedCursor{}, errShellAdapter
	}
	var wordSeen, byteSeen, spaceSeen bool
	for decoder.PeekKind() != jsontext.KindEndObject {
		name, err := decoder.ReadToken()
		if err != nil || name.Kind() != jsontext.KindString {
			return cli.CompletionObservedCursor{}, errShellAdapter
		}
		switch name.String() {
		case "word":
			if wordSeen {
				return cli.CompletionObservedCursor{}, errShellAdapter
			}
			wordSeen = true
			cursor.Word, err = readShellAdapterJSONNonnegativeInt(decoder)
			if err != nil {
				return cli.CompletionObservedCursor{}, err
			}
		case "byte":
			if byteSeen {
				return cli.CompletionObservedCursor{}, errShellAdapter
			}
			byteSeen = true
			cursor.Byte, err = readShellAdapterJSONNonnegativeInt(decoder)
			if err != nil {
				return cli.CompletionObservedCursor{}, err
			}
		case "space":
			if spaceSeen {
				return cli.CompletionObservedCursor{}, errShellAdapter
			}
			spaceSeen = true
			value, err := readShellAdapterJSONString(decoder)
			if err != nil {
				return cli.CompletionObservedCursor{}, err
			}
			var ok bool
			cursor.Space, ok = shellAdapterCursorSpaceFromToken(value)
			if !ok {
				return cli.CompletionObservedCursor{}, errShellAdapter
			}
		default:
			return cli.CompletionObservedCursor{}, errShellAdapter
		}
	}
	if _, err := decoder.ReadToken(); err != nil || !wordSeen || !byteSeen {
		return cli.CompletionObservedCursor{}, errShellAdapter
	}
	return cursor, nil
}

func decodeShellAdapterJSONCapabilities(decoder *jsontext.Decoder, capabilities *cli.CompletionClientCapabilities, native bool) error {
	start, err := decoder.ReadToken()
	if err != nil || start.Kind() != jsontext.KindBeginArray {
		return errShellAdapter
	}
	for decoder.PeekKind() != jsontext.KindEndArray {
		value, err := readShellAdapterJSONString(decoder)
		if err != nil {
			return err
		}
		if native {
			applyShellAdapterNativeCapability(capabilities, value)
		} else {
			applyShellAdapterCapability(capabilities, value)
		}
	}
	if _, err := decoder.ReadToken(); err != nil {
		return errShellAdapter
	}
	return nil
}

func readShellAdapterJSONString(decoder *jsontext.Decoder) (string, error) {
	token, err := decoder.ReadToken()
	if err != nil || token.Kind() != jsontext.KindString {
		return "", errShellAdapter
	}
	return token.String(), nil
}

func readShellAdapterJSONNonnegativeInt(decoder *jsontext.Decoder) (int, error) {
	token, err := decoder.ReadToken()
	if err != nil || token.Kind() != jsontext.KindNumber {
		return 0, errShellAdapter
	}
	text := token.String()
	if !canonicalShellAdapterNonnegativeInteger(text) {
		return 0, errShellAdapter
	}
	value, err := strconv.Atoi(text)
	if err != nil {
		return 0, errShellAdapter
	}
	return value, nil
}

func canonicalShellAdapterNonnegativeInteger(text string) bool {
	if text == "0" {
		return true
	}
	if text == "" || text[0] < '1' || text[0] > '9' {
		return false
	}
	for _, r := range text[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

type shellAdapterJSONResponse struct {
	Failed     bool                        `json:"failed"`
	Replace    shellAdapterJSONReplace     `json:"replace"`
	Spacing    string                      `json:"spacing"`
	Items      []shellAdapterJSONItem      `json:"items"`
	Groups     []shellAdapterJSONGroup     `json:"groups,omitempty"`
	Messages   []shellAdapterJSONMessage   `json:"messages,omitempty"`
	Native     []string                    `json:"native,omitempty"`
	Delegation *shellAdapterJSONDelegation `json:"delegation,omitempty"`
}

type (
	shellAdapterJSONReplace struct {
		Argument int `json:"argument"`
		Start    int `json:"start"`
		End      int `json:"end"`
	}
	shellAdapterJSONRange struct {
		Start int `json:"start"`
		End   int `json:"end"`
	}
	shellAdapterJSONItem struct {
		Value            string                  `json:"value"`
		Replacement      []string                `json:"replacement,omitempty"`
		Spacing          string                  `json:"spacing"`
		DisplayText      string                  `json:"display_text,omitempty"`
		Description      string                  `json:"description,omitempty"`
		Detail           string                  `json:"detail,omitempty"`
		Kind             string                  `json:"kind,omitempty"`
		Group            string                  `json:"group,omitempty"`
		MatchRanges      []shellAdapterJSONRange `json:"match_ranges,omitempty"`
		CommitCharacters []string                `json:"commit_characters,omitempty"`
	}
)

type (
	shellAdapterJSONGroup struct {
		ID    string `json:"id"`
		Label string `json:"label"`
	}
	shellAdapterJSONMessage struct {
		Kind string `json:"kind"`
		Text string `json:"text"`
	}
	shellAdapterJSONDelegation struct {
		Args   []string                         `json:"args"`
		Cursor shellAdapterJSONDelegationCursor `json:"cursor"`
	}
	shellAdapterJSONDelegationCursor struct {
		Argument int `json:"argument"`
		Byte     int `json:"byte"`
	}
)

func encodeShellAdapterJSONResponse(writer io.Writer, response shellAdapterResponse) error {
	spacing, ok := shellAdapterSpacingToken(response.spacing)
	if !ok {
		return errShellAdapter
	}
	wire := shellAdapterJSONResponse{
		Failed: response.failed,
		Replace: shellAdapterJSONReplace{
			Argument: response.replace.Argument,
			Start:    response.replace.Start,
			End:      response.replace.End,
		},
		Spacing: spacing,
		Items:   make([]shellAdapterJSONItem, 0, len(response.items)),
	}
	for _, item := range response.items {
		itemSpacing, ok := shellAdapterSpacingToken(item.spacing)
		if !ok || item.replacementSet && len(item.replacement) == 0 {
			return errShellAdapter
		}
		kind := ""
		if item.kind != cli.CompletionCandidateUnknown {
			kind, ok = shellAdapterCandidateKindToken(item.kind)
			if !ok {
				return errShellAdapter
			}
		}
		wireItem := shellAdapterJSONItem{
			Value:            item.value,
			Spacing:          itemSpacing,
			DisplayText:      item.displayText,
			Description:      item.description,
			Detail:           item.detail,
			Kind:             kind,
			Group:            item.group,
			CommitCharacters: slices.Clone(item.commitCharacters),
		}
		if item.replacementSet {
			wireItem.Replacement = slices.Clone(item.replacement)
		}
		wireItem.MatchRanges = make([]shellAdapterJSONRange, len(item.matchRanges))
		for index, matchRange := range item.matchRanges {
			wireItem.MatchRanges[index] = shellAdapterJSONRange{
				Start: matchRange.Start,
				End:   matchRange.End,
			}
		}
		wire.Items = append(wire.Items, wireItem)
	}
	wire.Groups = make([]shellAdapterJSONGroup, len(response.groups))
	for index, group := range response.groups {
		wire.Groups[index] = shellAdapterJSONGroup{
			ID:    group.ID,
			Label: group.Label,
		}
	}
	for _, message := range response.messages {
		kind, ok := shellAdapterMessageKindToken(message.Kind)
		if !ok || kind == shellAdapterUnknownToken {
			return errShellAdapter
		}
		wire.Messages = append(wire.Messages, shellAdapterJSONMessage{
			Kind: kind,
			Text: message.Text,
		})
	}
	for _, native := range response.native {
		token, ok := shellAdapterNativeKindToken(native)
		if !ok || token == shellAdapterUnknownToken {
			return errShellAdapter
		}
		wire.Native = append(wire.Native, token)
	}
	if response.delegation != nil {
		wire.Delegation = &shellAdapterJSONDelegation{
			Args: slices.Clone(response.delegation.Args),
			Cursor: shellAdapterJSONDelegationCursor{
				Argument: response.delegation.Cursor.Argument,
				Byte:     response.delegation.Cursor.Byte,
			},
		}
	}
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(wire); err != nil {
		return failureError{
			subject: "encode shell completion JSON",
			detail:  boundedFailureDetail(err.Error()),
		}
	}
	return nil
}
