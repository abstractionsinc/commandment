package cli

import (
	"bytes"
	"encoding/json"
	"encoding/json/jsontext"
	"errors"
	"io"
	"unicode/utf8"
)

type shellCompletionAdapterJSONRequest struct {
	ResponseEncoding   string                           `json:"response_encoding,omitempty"`
	Shell              *shellCompletionAdapterJSONShell `json:"shell"`
	Input              *shellCompletionAdapterJSONInput `json:"input"`
	Matching           string                           `json:"matching,omitempty"`
	Capabilities       []string                         `json:"capabilities,omitempty"`
	NativeCapabilities []string                         `json:"native_capabilities,omitempty"`
}

type shellCompletionAdapterJSONShell struct {
	Family  *string `json:"family"`
	Version *string `json:"version"`
}

type shellCompletionAdapterJSONInput struct {
	Topology        string                            `json:"topology,omitempty"`
	ExactArgvPrefix int                               `json:"exact_argv_prefix,omitempty"`
	Words           []shellCompletionAdapterJSONWord  `json:"words"`
	Cursor          *shellCompletionAdapterJSONCursor `json:"cursor"`
}

type shellCompletionAdapterJSONWord struct {
	Value    *string `json:"value"`
	Fidelity string  `json:"fidelity,omitempty"`
}

type shellCompletionAdapterJSONCursor struct {
	Word  *int   `json:"word"`
	Byte  *int   `json:"byte"`
	Space string `json:"space,omitempty"`
}

func decodeShellCompletionAdapterJSONRequest(reader io.Reader) (shellCompletionAdapterRequest, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		failure := &shellCompletionAdapterError{
			detail: "read request",
			cause:  err,
		}
		return shellCompletionAdapterRequest{}, failure
	}
	if !utf8.Valid(data) || validateShellCompletionAdapterJSON(data) != nil {
		return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var wire shellCompletionAdapterJSONRequest
	if err := decoder.Decode(&wire); err != nil {
		return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
	}
	return shellCompletionAdapterRequestFromJSON(wire)
}

func shellCompletionAdapterRequestFromJSON(wire shellCompletionAdapterJSONRequest) (shellCompletionAdapterRequest, error) {
	if wire.Shell == nil || wire.Shell.Family == nil || wire.Shell.Version == nil ||
		wire.Input == nil || wire.Input.Cursor == nil || wire.Input.Cursor.Word == nil || wire.Input.Cursor.Byte == nil ||
		len(wire.Input.Words) == 0 {
		return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
	}
	request := shellCompletionAdapterRequest{
		shellFamily:  *wire.Shell.Family,
		shellVersion: *wire.Shell.Version,
		input: CompletionInputObservation{
			Topology:        CompletionArgvTopologyUncertain,
			ExactArgvPrefix: wire.Input.ExactArgvPrefix,
			Cursor: CompletionObservedCursor{
				Word:  *wire.Input.Cursor.Word,
				Byte:  *wire.Input.Cursor.Byte,
				Space: CompletionCursorSpaceObservedWordBytes,
			},
		},
		matching: shellCompletionAdapterMatchingPrefix,
	}
	if wire.ResponseEncoding != "" {
		codec, ok := shellCompletionAdapterCodecFromToken(wire.ResponseEncoding)
		if !ok {
			return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
		}
		request.responseCodec = codec
	}
	if wire.Input.Topology != "" {
		var ok bool
		request.input.Topology, ok = shellCompletionAdapterTopologyFromToken(wire.Input.Topology)
		if !ok {
			return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
		}
	}
	if wire.Input.Cursor.Space != "" {
		var ok bool
		request.input.Cursor.Space, ok = shellCompletionAdapterCursorSpaceFromToken(wire.Input.Cursor.Space)
		if !ok {
			return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
		}
	}
	request.input.Words = make([]CompletionObservedWord, len(wire.Input.Words))
	for index, word := range wire.Input.Words {
		if word.Value == nil {
			return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
		}
		fidelity := CompletionInputFidelityLexical
		if word.Fidelity != "" {
			var ok bool
			fidelity, ok = shellCompletionAdapterFidelityFromToken(word.Fidelity)
			if !ok {
				return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
			}
		}
		request.input.Words[index] = CompletionObservedWord{
			Value:    *word.Value,
			Fidelity: fidelity,
		}
	}
	if wire.Matching != "" {
		var ok bool
		request.matching, ok = shellCompletionAdapterMatchingFromToken(wire.Matching)
		if !ok {
			return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
		}
	}
	applyShellCompletionAdapterCapabilities(&request.capabilities, wire.Capabilities, wire.NativeCapabilities)
	return request, nil
}

type shellCompletionAdapterJSONResponse struct {
	Failed     bool                                  `json:"failed"`
	Replace    shellCompletionAdapterJSONReplace     `json:"replace"`
	Spacing    string                                `json:"spacing"`
	Items      []shellCompletionAdapterJSONItem      `json:"items"`
	Groups     []shellCompletionAdapterJSONGroup     `json:"groups,omitempty"`
	Messages   []shellCompletionAdapterJSONMessage   `json:"messages,omitempty"`
	Native     []string                              `json:"native,omitempty"`
	Delegation *shellCompletionAdapterJSONDelegation `json:"delegation,omitempty"`
}

type shellCompletionAdapterJSONReplace struct {
	Argument int `json:"argument"`
	Start    int `json:"start"`
	End      int `json:"end"`
}

type shellCompletionAdapterJSONRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type shellCompletionAdapterJSONItem struct {
	Value            string                            `json:"value"`
	Replacement      []string                          `json:"replacement,omitempty"`
	Spacing          string                            `json:"spacing"`
	DisplayText      string                            `json:"display_text,omitempty"`
	Description      string                            `json:"description,omitempty"`
	Detail           string                            `json:"detail,omitempty"`
	Kind             string                            `json:"kind,omitempty"`
	Group            string                            `json:"group,omitempty"`
	MatchRanges      []shellCompletionAdapterJSONRange `json:"match_ranges,omitempty"`
	CommitCharacters []string                          `json:"commit_characters,omitempty"`
}

type shellCompletionAdapterJSONGroup struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type shellCompletionAdapterJSONMessage struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

type shellCompletionAdapterJSONDelegation struct {
	Args   []string                          `json:"args"`
	Cursor shellCompletionAdapterJSONCursor2 `json:"cursor"`
}

type shellCompletionAdapterJSONCursor2 struct {
	Argument int `json:"argument"`
	Byte     int `json:"byte"`
}

func encodeShellCompletionAdapterJSONResponse(writer io.Writer, response shellCompletionAdapterResponse) error {
	spacing, ok := shellCompletionAdapterSpacingToken(response.Spacing)
	if !ok {
		return errInvalidShellCompletionAdapter
	}
	wire := shellCompletionAdapterJSONResponse{
		Failed: response.Failed,
		Replace: shellCompletionAdapterJSONReplace{
			Argument: response.Replace.Argument,
			Start:    response.Replace.Start,
			End:      response.Replace.End,
		},
		Spacing: spacing,
		Items:   make([]shellCompletionAdapterJSONItem, 0, len(response.Items)),
	}
	for _, item := range response.Items {
		itemSpacing, ok := shellCompletionAdapterSpacingToken(item.Spacing)
		if !ok || item.ReplacementSet && len(item.Replacement) == 0 {
			return errInvalidShellCompletionAdapter
		}
		kind := ""
		if item.Kind != CompletionCandidateUnknown {
			kind, ok = shellCompletionAdapterCandidateKindToken(item.Kind)
			if !ok {
				return errInvalidShellCompletionAdapter
			}
		}
		wireItem := shellCompletionAdapterJSONItem{
			Value:            item.Value,
			Spacing:          itemSpacing,
			DisplayText:      item.DisplayText,
			Description:      item.Description,
			Detail:           item.Detail,
			Kind:             kind,
			Group:            item.Group,
			CommitCharacters: append([]string(nil), item.CommitCharacters...),
		}
		if item.ReplacementSet {
			wireItem.Replacement = append([]string(nil), item.Replacement...)
		}
		wireItem.MatchRanges = make([]shellCompletionAdapterJSONRange, len(item.MatchRanges))
		for index, matchRange := range item.MatchRanges {
			wireItem.MatchRanges[index] = shellCompletionAdapterJSONRange(matchRange)
		}
		wire.Items = append(wire.Items, wireItem)
	}
	wire.Groups = make([]shellCompletionAdapterJSONGroup, len(response.Groups))
	for index, group := range response.Groups {
		wire.Groups[index] = shellCompletionAdapterJSONGroup{
			ID:    group.ID,
			Label: group.Label,
		}
	}
	wire.Messages = make([]shellCompletionAdapterJSONMessage, 0, len(response.Messages))
	for _, message := range response.Messages {
		kind, ok := shellCompletionAdapterMessageKindToken(message.Kind)
		if !ok || kind == shellCompletionAdapterTokenUnknown {
			return errInvalidShellCompletionAdapter
		}
		wire.Messages = append(wire.Messages, shellCompletionAdapterJSONMessage{
			Kind: kind,
			Text: message.Text,
		})
	}
	for _, native := range response.Native {
		token, ok := shellCompletionAdapterNativeKindToken(native)
		if !ok || token == shellCompletionAdapterTokenUnknown {
			return errInvalidShellCompletionAdapter
		}
		wire.Native = append(wire.Native, token)
	}
	if response.Delegation != nil {
		wire.Delegation = &shellCompletionAdapterJSONDelegation{
			Args: append([]string(nil), response.Delegation.Args...),
			Cursor: shellCompletionAdapterJSONCursor2{
				Argument: response.Delegation.Cursor.Argument,
				Byte:     response.Delegation.Cursor.Byte,
			},
		}
	}
	return encodeShellCompletionAdapterJSONResponseValue(writer, wire)
}

func validateShellCompletionAdapterJSON(data []byte) error {
	decoder := jsontext.NewDecoder(bytes.NewReader(data))
	if err := walkShellCompletionAdapterJSON(decoder); err != nil {
		return err
	}
	if _, err := decoder.ReadToken(); !errors.Is(err, io.EOF) {
		return errInvalidShellCompletionAdapter
	}
	return nil
}

func walkShellCompletionAdapterJSON(decoder *jsontext.Decoder) error {
	token, err := decoder.ReadToken()
	if err != nil {
		return errInvalidShellCompletionAdapter
	}
	switch token.Kind() {
	case jsontext.KindNull:
		return errInvalidShellCompletionAdapter
	case jsontext.KindNumber:
		if !canonicalShellCompletionAdapterDecimal(token.String()) {
			return errInvalidShellCompletionAdapter
		}
	case jsontext.KindBeginObject:
		seen := make(map[string]struct{})
		for decoder.PeekKind() != jsontext.KindEndObject {
			name, err := decoder.ReadToken()
			if err != nil || name.Kind() != jsontext.KindString {
				return errInvalidShellCompletionAdapter
			}
			if _, duplicate := seen[name.String()]; duplicate {
				return errInvalidShellCompletionAdapter
			}
			seen[name.String()] = struct{}{}
			if err := walkShellCompletionAdapterJSON(decoder); err != nil {
				return err
			}
		}
		if _, err := decoder.ReadToken(); err != nil {
			return errInvalidShellCompletionAdapter
		}
	case jsontext.KindBeginArray:
		for decoder.PeekKind() != jsontext.KindEndArray {
			if err := walkShellCompletionAdapterJSON(decoder); err != nil {
				return err
			}
		}
		if _, err := decoder.ReadToken(); err != nil {
			return errInvalidShellCompletionAdapter
		}
	case jsontext.KindString, jsontext.KindTrue, jsontext.KindFalse:
		return nil
	default:
		return errInvalidShellCompletionAdapter
	}
	return nil
}

func canonicalShellCompletionAdapterDecimal(value string) bool {
	if value == "0" {
		return true
	}
	if value == "" || value == "-0" || value[0] == '+' {
		return false
	}
	start := 0
	if value[0] == '-' {
		start = 1
		if len(value) == 1 {
			return false
		}
	}
	if value[start] == '0' {
		return false
	}
	for index := start; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}

func encodeShellCompletionAdapterJSONResponseValue(writer io.Writer, response shellCompletionAdapterJSONResponse) error {
	if err := validateShellCompletionAdapterJSONResponseStrings(response); err != nil {
		return err
	}
	data, err := json.Marshal(response)
	if err != nil {
		failure := &shellCompletionAdapterError{
			detail: "encode JSON response",
			cause:  err,
		}
		return failure
	}
	if _, err := writer.Write(data); err != nil {
		failure := &shellCompletionAdapterError{
			detail: "write JSON response",
			cause:  err,
		}
		return failure
	}
	return nil
}

func validateShellCompletionAdapterJSONResponseStrings(response shellCompletionAdapterJSONResponse) error {
	if !utf8.ValidString(response.Spacing) {
		return errInvalidShellCompletionAdapter
	}
	for _, item := range response.Items {
		for _, value := range []string{item.Value, item.Spacing, item.DisplayText, item.Description, item.Detail, item.Kind, item.Group} {
			if !utf8.ValidString(value) {
				return errInvalidShellCompletionAdapter
			}
		}
		for _, value := range item.Replacement {
			if !utf8.ValidString(value) {
				return errInvalidShellCompletionAdapter
			}
		}
		for _, value := range item.CommitCharacters {
			if !utf8.ValidString(value) {
				return errInvalidShellCompletionAdapter
			}
		}
	}
	for _, group := range response.Groups {
		if !utf8.ValidString(group.ID) || !utf8.ValidString(group.Label) {
			return errInvalidShellCompletionAdapter
		}
	}
	for _, message := range response.Messages {
		if !utf8.ValidString(message.Kind) || !utf8.ValidString(message.Text) {
			return errInvalidShellCompletionAdapter
		}
	}
	for _, native := range response.Native {
		if !utf8.ValidString(native) {
			return errInvalidShellCompletionAdapter
		}
	}
	if response.Delegation != nil {
		for _, arg := range response.Delegation.Args {
			if !utf8.ValidString(arg) {
				return errInvalidShellCompletionAdapter
			}
		}
	}
	return nil
}
