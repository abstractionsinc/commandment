package cli

import (
	"bufio"
	"bytes"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	shellCompletionAdapterTokenInput           = "input"
	shellCompletionAdapterTokenShell           = "shell"
	shellCompletionAdapterTokenEnd             = "end"
	shellCompletionAdapterWordRecordFieldCount = 3
	shellCompletionAdapterHexNibbleShift       = 4
	shellCompletionAdapterHexUpperLetterOffset = 10
)

func decodeShellCompletionAdapterRecordsRequest(reader io.Reader) (shellCompletionAdapterRequest, error) {
	records, err := readShellCompletionAdapterRecords(reader)
	if err != nil {
		return shellCompletionAdapterRequest{}, err
	}
	request := shellCompletionAdapterRequest{
		matching: shellCompletionAdapterMatchingPrefix,
	}
	var shellSeen, inputSeen, matchingSeen, responseSeen bool
	words := make(map[int]CompletionObservedWord)
	capabilities := make([]string, 0)
	nativeCapabilities := make([]string, 0)
	for _, record := range records {
		switch record.kind {
		case "response-encoding":
			if responseSeen || len(record.fields) != 1 {
				return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
			}
			responseSeen = true
			codec, ok := shellCompletionAdapterCodecFromToken(record.fields[0])
			if !ok {
				return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
			}
			request.responseCodec = codec
		case shellCompletionAdapterTokenShell:
			if shellSeen || len(record.fields) != 2 {
				return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
			}
			shellSeen = true
			request.shellFamily, request.shellVersion = record.fields[0], record.fields[1]
		case shellCompletionAdapterTokenInput:
			if inputSeen || len(record.fields) != 5 {
				return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
			}
			inputSeen = true
			var ok bool
			request.input.Topology, ok = shellCompletionAdapterTopologyFromToken(record.fields[0])
			if !ok {
				return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
			}
			request.input.ExactArgvPrefix, ok = shellCompletionAdapterParseInt(record.fields[1])
			if !ok {
				return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
			}
			request.input.Cursor.Word, ok = shellCompletionAdapterParseInt(record.fields[2])
			if !ok {
				return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
			}
			request.input.Cursor.Byte, ok = shellCompletionAdapterParseInt(record.fields[3])
			if !ok {
				return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
			}
			request.input.Cursor.Space, ok = shellCompletionAdapterCursorSpaceFromToken(record.fields[4])
			if !ok {
				return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
			}
		case "word":
			if len(record.fields) != shellCompletionAdapterWordRecordFieldCount {
				return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
			}
			index, ok := shellCompletionAdapterParseInt(record.fields[0])
			if !ok || index < 0 {
				return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
			}
			if _, duplicate := words[index]; duplicate {
				return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
			}
			fidelity, ok := shellCompletionAdapterFidelityFromToken(record.fields[2])
			if !ok {
				return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
			}
			words[index] = CompletionObservedWord{
				Value:    record.fields[1],
				Fidelity: fidelity,
			}
		case "matching":
			if matchingSeen || len(record.fields) != 1 {
				return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
			}
			matchingSeen = true
			var ok bool
			request.matching, ok = shellCompletionAdapterMatchingFromToken(record.fields[0])
			if !ok {
				return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
			}
		case "capability":
			if len(record.fields) != 1 {
				return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
			}
			capabilities = append(capabilities, record.fields[0])
		case "native-capability":
			if len(record.fields) != 1 {
				return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
			}
			nativeCapabilities = append(nativeCapabilities, record.fields[0])
		default:
			return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
		}
	}
	if !shellSeen || !inputSeen || len(words) == 0 {
		return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
	}
	request.input.Words = make([]CompletionObservedWord, len(words))
	for index := range len(words) {
		word, ok := words[index]
		if !ok {
			return shellCompletionAdapterRequest{}, errInvalidShellCompletionAdapter
		}
		request.input.Words[index] = word
	}
	applyShellCompletionAdapterCapabilities(&request.capabilities, capabilities, nativeCapabilities)
	return request, nil
}

type shellCompletionAdapterRecord struct {
	kind   string
	fields []string
}

func readShellCompletionAdapterRecords(reader io.Reader) ([]shellCompletionAdapterRecord, error) {
	buffered := bufio.NewReader(reader)
	records := make([]shellCompletionAdapterRecord, 0)
	ended := false
	for {
		line, err := buffered.ReadBytes('\n')
		if err != nil {
			if err == io.EOF && len(line) == 0 {
				break
			}
			return nil, errInvalidShellCompletionAdapter
		}
		if ended {
			return nil, errInvalidShellCompletionAdapter
		}
		line = line[:len(line)-1]
		if bytes.IndexByte(line, 0) >= 0 {
			return nil, errInvalidShellCompletionAdapter
		}
		parts := bytes.Split(line, []byte{'\t'})
		if len(parts) == 0 || len(parts[0]) == 0 {
			return nil, errInvalidShellCompletionAdapter
		}
		kind := string(parts[0])
		if kind == shellCompletionAdapterTokenEnd {
			if len(parts) != 1 {
				return nil, errInvalidShellCompletionAdapter
			}
			ended = true
			continue
		}
		fields := make([]string, len(parts)-1)
		for index := 1; index < len(parts); index++ {
			value, err := unescapeShellCompletionAdapterRecordField(parts[index])
			if err != nil {
				return nil, errInvalidShellCompletionAdapter
			}
			fields[index-1] = value
		}
		records = append(records, shellCompletionAdapterRecord{
			kind:   kind,
			fields: fields,
		})
	}
	if !ended {
		return nil, errInvalidShellCompletionAdapter
	}
	return records, nil
}

func writeShellCompletionAdapterRecord(writer io.Writer, kind string, fields ...string) error {
	if kind == "" || strings.ContainsAny(kind, "\t\r\n") {
		return errInvalidShellCompletionAdapter
	}
	if _, err := io.WriteString(writer, kind); err != nil {
		return &shellCompletionAdapterError{
			detail: "write record kind",
			cause:  err,
		}
	}
	for _, field := range fields {
		encoded, err := escapeShellCompletionAdapterRecordField(field)
		if err != nil {
			return errInvalidShellCompletionAdapter
		}
		if _, err := io.WriteString(writer, "\t"+encoded); err != nil {
			return &shellCompletionAdapterError{
				detail: "write record field",
				cause:  err,
			}
		}
	}
	if _, err := io.WriteString(writer, "\n"); err != nil {
		return &shellCompletionAdapterError{
			detail: "write record terminator",
			cause:  err,
		}
	}
	return nil
}

func shellCompletionAdapterParseInt(value string) (int, bool) {
	if value == "" || (len(value) > 1 && value[0] == '0') || strings.HasPrefix(value, "+") || value == "-0" {
		return 0, false
	}
	parsed, err := strconv.ParseInt(value, 10, 0)
	if err != nil || strconv.FormatInt(parsed, 10) != value {
		return 0, false
	}
	return int(parsed), true
}

func encodeShellCompletionAdapterRecordsResponse(writer io.Writer, response shellCompletionAdapterResponse) error {
	spacing, ok := shellCompletionAdapterSpacingToken(response.Spacing)
	if !ok {
		return errInvalidShellCompletionAdapter
	}
	if err := writeShellCompletionAdapterRecord(writer, "result", strconv.FormatBool(response.Failed)); err != nil {
		return err
	}
	if err := writeShellCompletionAdapterRecord(
		writer,
		"replace",
		strconv.Itoa(response.Replace.Argument),
		strconv.Itoa(response.Replace.Start),
		strconv.Itoa(response.Replace.End),
	); err != nil {
		return err
	}
	if err := writeShellCompletionAdapterRecord(writer, "spacing", spacing); err != nil {
		return err
	}
	for index, item := range response.Items {
		itemSpacing, ok := shellCompletionAdapterSpacingToken(item.Spacing)
		if !ok || item.ReplacementSet && len(item.Replacement) == 0 {
			return errInvalidShellCompletionAdapter
		}
		if err := writeShellCompletionAdapterRecord(
			writer, "item", strconv.Itoa(index), item.Value, strconv.FormatBool(item.ReplacementSet), itemSpacing,
		); err != nil {
			return err
		}
		if item.ReplacementSet {
			for replacementIndex, value := range item.Replacement {
				if err := writeShellCompletionAdapterRecord(
					writer, "item-replacement", strconv.Itoa(index), strconv.Itoa(replacementIndex), value,
				); err != nil {
					return err
				}
			}
		}
		if item.DisplayText != "" {
			if err := writeShellCompletionAdapterRecord(
				writer, "item-display", strconv.Itoa(index), item.DisplayText,
			); err != nil {
				return err
			}
		}
		if item.Description != "" {
			if err := writeShellCompletionAdapterRecord(
				writer, "item-description", strconv.Itoa(index), item.Description,
			); err != nil {
				return err
			}
		}
		if item.Detail != "" {
			if err := writeShellCompletionAdapterRecord(
				writer, "item-detail", strconv.Itoa(index), item.Detail,
			); err != nil {
				return err
			}
		}
		if item.Kind != CompletionCandidateUnknown {
			kind, ok := shellCompletionAdapterCandidateKindToken(item.Kind)
			if !ok {
				return errInvalidShellCompletionAdapter
			}
			if err := writeShellCompletionAdapterRecord(writer, "item-kind", strconv.Itoa(index), kind); err != nil {
				return err
			}
		}
		if item.Group != "" {
			if err := writeShellCompletionAdapterRecord(
				writer, "item-group", strconv.Itoa(index), item.Group,
			); err != nil {
				return err
			}
		}
		for rangeIndex, matchRange := range item.MatchRanges {
			if err := writeShellCompletionAdapterRecord(
				writer,
				"item-match-range",
				strconv.Itoa(index),
				strconv.Itoa(rangeIndex),
				strconv.Itoa(matchRange.Start),
				strconv.Itoa(matchRange.End),
			); err != nil {
				return err
			}
		}
		for characterIndex, character := range item.CommitCharacters {
			if utf8.RuneCountInString(character) != 1 {
				return errInvalidShellCompletionAdapter
			}
			if err := writeShellCompletionAdapterRecord(
				writer, "item-commit-character", strconv.Itoa(index), strconv.Itoa(characterIndex), character,
			); err != nil {
				return err
			}
		}
	}
	for index, group := range response.Groups {
		if err := writeShellCompletionAdapterRecord(
			writer, "group", strconv.Itoa(index), group.ID, group.Label,
		); err != nil {
			return err
		}
	}
	for index, message := range response.Messages {
		kind, ok := shellCompletionAdapterMessageKindToken(message.Kind)
		if !ok || kind == shellCompletionAdapterTokenUnknown {
			return errInvalidShellCompletionAdapter
		}
		if err := writeShellCompletionAdapterRecord(
			writer, "message", strconv.Itoa(index), kind, message.Text,
		); err != nil {
			return err
		}
	}
	for index, native := range response.Native {
		kind, ok := shellCompletionAdapterNativeKindToken(native)
		if !ok || kind == shellCompletionAdapterTokenUnknown {
			return errInvalidShellCompletionAdapter
		}
		if err := writeShellCompletionAdapterRecord(writer, "native", strconv.Itoa(index), kind); err != nil {
			return err
		}
	}
	if response.Delegation != nil {
		if err := writeShellCompletionAdapterRecord(
			writer,
			shellCompletionAdapterTokenDelegation,
			strconv.Itoa(response.Delegation.Cursor.Argument),
			strconv.Itoa(response.Delegation.Cursor.Byte),
		); err != nil {
			return err
		}
		for index, arg := range response.Delegation.Args {
			if err := writeShellCompletionAdapterRecord(
				writer, "delegation-arg", strconv.Itoa(index), arg,
			); err != nil {
				return err
			}
		}
	}
	return writeShellCompletionAdapterRecord(writer, shellCompletionAdapterTokenEnd)
}

func escapeShellCompletionAdapterRecordField(value string) (string, error) {
	var builder strings.Builder
	for index := range len(value) {
		character := value[index]
		if character == 0 {
			return "", errInvalidShellCompletionAdapter
		}
		switch character {
		case '\\':
			builder.WriteString("\\\\")
		case '\t':
			builder.WriteString("\\t")
		case '\n':
			builder.WriteString("\\n")
		case '\r':
			builder.WriteString("\\r")
		default:
			if character >= 0x20 && character <= 0x7e {
				builder.WriteByte(character)
				continue
			}
			builder.WriteString("\\x")
			builder.WriteByte("0123456789ABCDEF"[character>>4])
			builder.WriteByte("0123456789ABCDEF"[character&0x0f])
		}
	}
	return builder.String(), nil
}

func unescapeShellCompletionAdapterRecordField(input []byte) (string, error) {
	output := make([]byte, 0, len(input))
	for index := 0; index < len(input); index++ {
		character := input[index]
		if character == 0 || character < 0x20 || character > 0x7e {
			return "", errInvalidShellCompletionAdapter
		}
		if character != '\\' {
			output = append(output, character)
			continue
		}
		index++
		if index >= len(input) {
			return "", errInvalidShellCompletionAdapter
		}
		switch input[index] {
		case '\\':
			output = append(output, '\\')
		case 't':
			output = append(output, '\t')
		case 'n':
			output = append(output, '\n')
		case 'r':
			output = append(output, '\r')
		case 'x':
			if index+2 >= len(input) {
				return "", errInvalidShellCompletionAdapter
			}
			high, highOK := shellCompletionAdapterHexUpper(input[index+1])
			low, lowOK := shellCompletionAdapterHexUpper(input[index+2])
			if !highOK || !lowOK {
				return "", errInvalidShellCompletionAdapter
			}
			value := high<<shellCompletionAdapterHexNibbleShift | low
			if value == 0 {
				return "", errInvalidShellCompletionAdapter
			}
			output = append(output, value)
			index += 2
		default:
			return "", errInvalidShellCompletionAdapter
		}
	}
	return string(output), nil
}

func shellCompletionAdapterHexUpper(value byte) (byte, bool) {
	switch {
	case value >= '0' && value <= '9':
		return value - '0', true
	case value >= 'A' && value <= 'F':
		return value - 'A' + shellCompletionAdapterHexUpperLetterOffset, true
	default:
		return 0, false
	}
}
