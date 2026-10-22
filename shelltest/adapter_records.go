package shelltest

import (
	"bytes"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/abstractionsinc/commandment/cli"
)

const (
	shellAdapterHexNibbleBits      = 4
	shellAdapterHexAlphaOffset     = 10
	shellAdapterRecordWriteSubject = "write shell adapter record"
)

func decodeShellAdapterRecordsRequest(reader io.Reader) (shellAdapterRequest, error) {
	data, err := io.ReadAll(io.LimitReader(reader, bridgeRequestFrameLimit+1))
	if err != nil || len(data) > bridgeRequestFrameLimit {
		return shellAdapterRequest{}, errShellAdapter
	}
	request := shellAdapterRequest{
		matching: shellAdapterMatchingPrefix,
	}
	var shellSeen, inputSeen, matchingSeen, responseSeen bool
	ended := false
	for len(data) != 0 {
		lineEnd := bytes.IndexByte(data, '\n')
		if lineEnd < 0 || ended {
			return shellAdapterRequest{}, errShellAdapter
		}
		line := data[:lineEnd]
		data = data[lineEnd+1:]
		kind, fields, fieldCount, ok := splitShellAdapterRecord(line)
		if !ok {
			return shellAdapterRequest{}, errShellAdapter
		}
		if kind == "end" {
			if fieldCount != 0 || len(data) != 0 {
				return shellAdapterRequest{}, errShellAdapter
			}
			ended = true
			break
		}

		switch kind {
		case "response-encoding":
			if responseSeen || fieldCount != 1 {
				return shellAdapterRequest{}, errShellAdapter
			}
			responseSeen = true
			value, err := unescapeShellAdapterField(fields[0])
			if err != nil {
				return shellAdapterRequest{}, errShellAdapter
			}
			request.responseCodec = shellAdapterCodecFromToken(value)
			if request.responseCodec == shellAdapterCodecUnknown {
				return shellAdapterRequest{}, errShellAdapter
			}
		case "shell":
			if shellSeen || fieldCount != 2 {
				return shellAdapterRequest{}, errShellAdapter
			}
			shellSeen = true
			family, err := unescapeShellAdapterField(fields[0])
			if err != nil || family == "" {
				return shellAdapterRequest{}, errShellAdapter
			}
			version, err := unescapeShellAdapterField(fields[1])
			if err != nil {
				return shellAdapterRequest{}, errShellAdapter
			}
			request.shellFamily = family
			request.shellVersion = version
		case "input":
			if inputSeen || fieldCount != 5 {
				return shellAdapterRequest{}, errShellAdapter
			}
			inputSeen = true
			values, err := unescapeShellAdapterFields(fields[:fieldCount])
			if err != nil {
				return shellAdapterRequest{}, errShellAdapter
			}
			var ok bool
			request.input.Topology, ok = shellAdapterTopologyFromToken(values[0])
			if !ok {
				return shellAdapterRequest{}, errShellAdapter
			}
			request.input.ExactArgvPrefix, ok = parseShellAdapterInt(values[1])
			if !ok {
				return shellAdapterRequest{}, errShellAdapter
			}
			request.input.Cursor.Word, ok = parseShellAdapterInt(values[2])
			if !ok {
				return shellAdapterRequest{}, errShellAdapter
			}
			request.input.Cursor.Byte, ok = parseShellAdapterInt(values[3])
			if !ok {
				return shellAdapterRequest{}, errShellAdapter
			}
			request.input.Cursor.Space, ok = shellAdapterCursorSpaceFromToken(values[4])
			if !ok {
				return shellAdapterRequest{}, errShellAdapter
			}
		case "word":
			if fieldCount != 3 || len(request.input.Words) >= shellAdapterWordLimit {
				return shellAdapterRequest{}, errShellAdapter
			}
			values, err := unescapeShellAdapterFields(fields[:fieldCount])
			if err != nil {
				return shellAdapterRequest{}, errShellAdapter
			}
			index, ok := parseShellAdapterInt(values[0])
			if !ok || index != len(request.input.Words) {
				return shellAdapterRequest{}, errShellAdapter
			}
			fidelity, ok := shellAdapterFidelityFromToken(values[2])
			if !ok {
				return shellAdapterRequest{}, errShellAdapter
			}
			request.input.Words = append(request.input.Words, cli.CompletionObservedWord{
				Value:    values[1],
				Fidelity: fidelity,
			})
		case "matching":
			if matchingSeen || fieldCount != 1 {
				return shellAdapterRequest{}, errShellAdapter
			}
			matchingSeen = true
			value, err := unescapeShellAdapterField(fields[0])
			if err != nil {
				return shellAdapterRequest{}, errShellAdapter
			}
			request.matching, ok = shellAdapterMatchingFromToken(value)
			if !ok {
				return shellAdapterRequest{}, errShellAdapter
			}
		case "capability":
			if fieldCount != 1 {
				return shellAdapterRequest{}, errShellAdapter
			}
			value, err := unescapeShellAdapterField(fields[0])
			if err != nil {
				return shellAdapterRequest{}, errShellAdapter
			}
			applyShellAdapterCapability(&request.capabilities, value)
		case "native-capability":
			if fieldCount != 1 {
				return shellAdapterRequest{}, errShellAdapter
			}
			value, err := unescapeShellAdapterField(fields[0])
			if err != nil {
				return shellAdapterRequest{}, errShellAdapter
			}
			applyShellAdapterNativeCapability(&request.capabilities, value)
		default:
			return shellAdapterRequest{}, errShellAdapter
		}
	}
	if !ended || !shellSeen || !inputSeen || len(request.input.Words) == 0 {
		return shellAdapterRequest{}, errShellAdapter
	}
	return request, nil
}

func splitShellAdapterRecord(line []byte) (string, [5][]byte, int, bool) {
	var fields [5][]byte
	if len(line) == 0 || bytes.IndexByte(line, 0) >= 0 {
		return "", fields, 0, false
	}
	kindBytes, remaining, found := bytes.Cut(line, []byte{'\t'})
	if !found {
		return string(line), fields, 0, true
	}
	kind := string(kindBytes)
	count := 0
	for {
		if count == len(fields) {
			return "", fields, 0, false
		}
		next := bytes.IndexByte(remaining, '\t')
		if next < 0 {
			fields[count] = remaining
			count++
			break
		}
		fields[count] = remaining[:next]
		count++
		remaining = remaining[next+1:]
	}
	return kind, fields, count, kind != ""
}

func unescapeShellAdapterFields(fields [][]byte) ([]string, error) {
	values := make([]string, len(fields))
	for index, field := range fields {
		value, err := unescapeShellAdapterField(field)
		if err != nil {
			return nil, err
		}
		values[index] = value
	}
	return values, nil
}

func encodeShellAdapterRecordsResponse(writer io.Writer, response shellAdapterResponse) error {
	spacing, ok := shellAdapterSpacingToken(response.spacing)
	if !ok {
		return errShellAdapter
	}
	if err := writeShellAdapterRecord(writer, "result", strconv.FormatBool(response.failed)); err != nil {
		return err
	}
	if err := writeShellAdapterRecord(writer, "replace", strconv.Itoa(response.replace.Argument), strconv.Itoa(response.replace.Start), strconv.Itoa(response.replace.End)); err != nil {
		return err
	}
	if err := writeShellAdapterRecord(writer, "spacing", spacing); err != nil {
		return err
	}
	for index, item := range response.items {
		itemSpacing, ok := shellAdapterSpacingToken(item.spacing)
		if !ok || item.replacementSet && len(item.replacement) == 0 {
			return errShellAdapter
		}
		if err := writeShellAdapterRecord(writer, "item", strconv.Itoa(index), item.value, strconv.FormatBool(item.replacementSet), itemSpacing); err != nil {
			return err
		}
		if item.replacementSet {
			for replacementIndex, value := range item.replacement {
				if err := writeShellAdapterRecord(writer, "item-replacement", strconv.Itoa(index), strconv.Itoa(replacementIndex), value); err != nil {
					return err
				}
			}
		}
		if item.displayText != "" {
			if err := writeShellAdapterRecord(writer, "item-display", strconv.Itoa(index), item.displayText); err != nil {
				return err
			}
		}
		if item.description != "" {
			if err := writeShellAdapterRecord(writer, "item-description", strconv.Itoa(index), item.description); err != nil {
				return err
			}
		}
		if item.detail != "" {
			if err := writeShellAdapterRecord(writer, "item-detail", strconv.Itoa(index), item.detail); err != nil {
				return err
			}
		}
		if item.kind != cli.CompletionCandidateUnknown {
			kind, ok := shellAdapterCandidateKindToken(item.kind)
			if !ok {
				return errShellAdapter
			}
			if err := writeShellAdapterRecord(writer, "item-kind", strconv.Itoa(index), kind); err != nil {
				return err
			}
		}
		if item.group != "" {
			if err := writeShellAdapterRecord(writer, "item-group", strconv.Itoa(index), item.group); err != nil {
				return err
			}
		}
		for rangeIndex, matchRange := range item.matchRanges {
			if err := writeShellAdapterRecord(writer, "item-match-range", strconv.Itoa(index), strconv.Itoa(rangeIndex), strconv.Itoa(matchRange.Start), strconv.Itoa(matchRange.End)); err != nil {
				return err
			}
		}
		for characterIndex, character := range item.commitCharacters {
			if utf8.RuneCountInString(character) != 1 {
				return errShellAdapter
			}
			if err := writeShellAdapterRecord(writer, "item-commit-character", strconv.Itoa(index), strconv.Itoa(characterIndex), character); err != nil {
				return err
			}
		}
	}
	for index, group := range response.groups {
		if err := writeShellAdapterRecord(writer, "group", strconv.Itoa(index), group.ID, group.Label); err != nil {
			return err
		}
	}
	for index, message := range response.messages {
		kind, ok := shellAdapterMessageKindToken(message.Kind)
		if !ok || kind == shellAdapterUnknownToken {
			return errShellAdapter
		}
		if err := writeShellAdapterRecord(writer, "message", strconv.Itoa(index), kind, message.Text); err != nil {
			return err
		}
	}
	for index, native := range response.native {
		kind, ok := shellAdapterNativeKindToken(native)
		if !ok || kind == shellAdapterUnknownToken {
			return errShellAdapter
		}
		if err := writeShellAdapterRecord(writer, "native", strconv.Itoa(index), kind); err != nil {
			return err
		}
	}
	if response.delegation != nil {
		if err := writeShellAdapterRecord(writer, "delegation", strconv.Itoa(response.delegation.Cursor.Argument), strconv.Itoa(response.delegation.Cursor.Byte)); err != nil {
			return err
		}
		for index, arg := range response.delegation.Args {
			if err := writeShellAdapterRecord(writer, "delegation-arg", strconv.Itoa(index), arg); err != nil {
				return err
			}
		}
	}
	return writeShellAdapterRecord(writer, "end")
}

func writeShellAdapterRecord(writer io.Writer, kind string, fields ...string) error {
	if kind == "" || strings.ContainsAny(kind, "\t\r\n") {
		return errShellAdapter
	}
	if _, err := io.WriteString(writer, kind); err != nil {
		failure := failureError{
			subject: shellAdapterRecordWriteSubject,
			cause:   err,
		}
		return failure
	}
	for _, field := range fields {
		encoded, err := escapeShellAdapterField(field)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(writer, "\t"+encoded); err != nil {
			failure := failureError{
				subject: shellAdapterRecordWriteSubject,
				cause:   err,
			}
			return failure
		}
	}
	if _, err := io.WriteString(writer, "\n"); err != nil {
		failure := failureError{
			subject: shellAdapterRecordWriteSubject,
			cause:   err,
		}
		return failure
	}
	return nil
}

func parseShellAdapterInt(value string) (int, bool) {
	if !canonicalShellAdapterNonnegativeInteger(value) {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func escapeShellAdapterField(value string) (string, error) {
	var b strings.Builder
	for i := range len(value) {
		c := value[i]
		if c == 0 {
			return "", errShellAdapter
		}
		switch c {
		case '\\':
			b.WriteString("\\\\")
		case '\t':
			b.WriteString("\\t")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		default:
			if c >= 0x20 && c <= 0x7e {
				b.WriteByte(c)
			} else {
				b.WriteString("\\x")
				b.WriteByte("0123456789ABCDEF"[c>>4])
				b.WriteByte("0123456789ABCDEF"[c&0x0f])
			}
		}
	}
	return b.String(), nil
}

func unescapeShellAdapterField(input []byte) (string, error) {
	out := make([]byte, 0, len(input))
	for i := 0; i < len(input); i++ {
		c := input[i]
		if c == 0 || c < 0x20 || c > 0x7e {
			return "", errShellAdapter
		}
		if c != '\\' {
			out = append(out, c)
			continue
		}
		i++
		if i >= len(input) {
			return "", errShellAdapter
		}
		switch input[i] {
		case '\\':
			out = append(out, '\\')
		case 't':
			out = append(out, '\t')
		case 'n':
			out = append(out, '\n')
		case 'r':
			out = append(out, '\r')
		case 'x':
			if i+2 >= len(input) {
				return "", errShellAdapter
			}
			hi, ok1 := upperHex(input[i+1])
			lo, ok2 := upperHex(input[i+2])
			if !ok1 || !ok2 {
				return "", errShellAdapter
			}
			value := hi<<shellAdapterHexNibbleBits | lo
			if value == 0 {
				return "", errShellAdapter
			}
			out = append(out, value)
			i += 2
		default:
			return "", errShellAdapter
		}
	}
	return string(out), nil
}

func upperHex(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'A' && c <= 'F':
		return c - 'A' + shellAdapterHexAlphaOffset, true
	default:
		return 0, false
	}
}

func shellAdapterSpacingToken(value cli.CompletionSpacing) (string, bool) {
	switch value {
	case cli.CompletionSpacingDefault:
		return "default", true
	case cli.CompletionSpacingAppendSpace:
		return "append-space", true
	case cli.CompletionSpacingNoSpace:
		return "no-space", true
	default:
		return "", false
	}
}

func shellAdapterCandidateKindToken(value cli.CompletionCandidateKind) (string, bool) {
	if value == cli.CompletionCandidateUnknown {
		return shellAdapterUnknownToken, true
	}
	switch value {
	case cli.CompletionCandidateValue, cli.CompletionCandidateCommand, cli.CompletionCandidateCommandAlias, cli.CompletionCandidateCommandShortcut, cli.CompletionCandidateAction, cli.CompletionCandidateFlag, cli.CompletionCandidateKeyword, cli.CompletionCandidateFile, cli.CompletionCandidateDirectory, cli.CompletionCandidateExecutable, cli.CompletionCandidateResource, cli.CompletionCandidateNamespace, cli.CompletionCandidateUser, cli.CompletionCandidateHost, cli.CompletionCandidateEnvironmentVariable, cli.CompletionCandidateURL:
		return string(value), true
	default:
		return "", false
	}
}

func shellAdapterMessageKindToken(value cli.CompletionMessageKind) (string, bool) {
	switch value {
	case cli.CompletionMessageUnknown:
		return shellAdapterUnknownToken, true
	case cli.CompletionMessageInformation:
		return "information", true
	case cli.CompletionMessageWarning:
		return "warning", true
	case cli.CompletionMessageError:
		return "error", true
	default:
		return "", false
	}
}

func shellAdapterNativeKindToken(value cli.CompletionNativeKind) (string, bool) {
	switch value {
	case cli.CompletionNativeUnknown:
		return shellAdapterUnknownToken, true
	case cli.CompletionNativePaths:
		return "paths", true
	case cli.CompletionNativeFiles:
		return "files", true
	case cli.CompletionNativeDirectories:
		return "directories", true
	case cli.CompletionNativeExecutables:
		return "executables", true
	case cli.CompletionNativeUsers:
		return "users", true
	case cli.CompletionNativeHosts:
		return "hosts", true
	case cli.CompletionNativeEnvironmentVariables:
		return "environment-variables", true
	default:
		return "", false
	}
}
