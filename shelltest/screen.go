package shelltest

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/clipperhouse/uax29/v2/graphemes"
	"github.com/mattn/go-runewidth"
)

const (
	terminalRows                    = 48
	terminalColumns                 = 160
	terminalScreenCellLimit         = terminalRows * terminalColumns * 2
	terminalPendingLimit            = 4096
	terminalCellTextLimit           = 256
	terminalQueryResponseLimit      = 64
	terminalDiagnosticTailLimit     = 4096
	terminalEscapeByte              = 0x1b
	terminalNullByte                = 0x00
	terminalBellByte                = 0x07
	terminalTabWidth                = 8
	terminalDoubleCellWidth         = 2
	terminalPrivateAutowrap         = 7
	terminalPrivateCursorBlink      = 12
	terminalPrivateAlternate        = 47
	terminalPrivateAlternateSave    = 1047
	terminalPrivateAlternateCursor  = 1049
	terminalPrivateCursorVisibility = 25
	terminalPrivateMouseX10         = 1000
	terminalPrivateMouseButton      = 1002
	terminalPrivateMouseAny         = 1003
	terminalPrivateFocus            = 1004
	terminalPrivateMouseUTF8        = 1005
	terminalPrivateMouseSGR         = 1006
	terminalPrivateMouseURXVT       = 1015
	terminalPrivateMeta8Bit         = 1034
	terminalPrivateBracketedPaste   = 2004
	terminalStatusReport            = 5
	terminalCursorReport            = 6
	terminalGeometryReport          = 18
	terminalEraseAll                = 2
	terminalEraseScrollback         = 3
)

type terminalParserState uint8

const (
	terminalGround terminalParserState = iota
	terminalEscape
	terminalEscapeIntermediate
	terminalCSI
	terminalOSC
	terminalStringControl
)

type terminalGrid struct {
	cells        [terminalRows][terminalColumns]string
	continuation [terminalRows][terminalColumns]bool
	row          int
	column       int
	savedRow     int
	savedColumn  int
}

type terminalOutput struct {
	mu              sync.Mutex
	changed         chan struct{}
	generation      uint64
	available       bool
	decckmKnown     bool
	decckm          bool
	autowrap        bool
	pendingWrap     bool
	scrollTop       int
	scrollBottom    int
	primary         terminalGrid
	alternate       terminalGrid
	alternateActive bool
	state           terminalParserState
	pending         []byte
	pendingUTF8     []byte
	pendingGrapheme string
	controlDiscard  bool
	stringEscaped   bool
	tail            []byte
	tailTruncated   bool
	reply           func([]byte) error
	replyFailure    error
}

func (output *terminalOutput) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	output.mu.Lock()
	output.appendTail(data)
	replies := output.consume(data)
	output.generation++
	output.mu.Unlock()
	select {
	case output.changed <- struct{}{}:
	default:
	}
	for _, reply := range replies {
		if output.reply == nil {
			continue
		}
		if len(reply) > terminalQueryResponseLimit {
			output.recordReplyFailure(failureError{subject: "terminal query response exceeds limit"})
			return len(data), output.failure()
		}
		if err := output.reply(reply); err != nil {
			output.recordReplyFailure(err)
			return len(data), err
		}
	}
	return len(data), nil
}

func (output *terminalOutput) consume(data []byte) [][]byte {
	var replies [][]byte
	for index := 0; index < len(data); {
		byteValue := data[index]
		switch output.state {
		case terminalGround:
			if byteValue == terminalEscapeByte {
				output.flushPrintableBoundary()
				output.state = terminalEscape
				index++
				continue
			}
			if byteValue < 0x20 || byteValue == 0x7f {
				output.flushPrintableBoundary()
				output.control(byteValue)
				index++
				continue
			}
			start := index
			for index < len(data) && data[index] != terminalEscapeByte && data[index] >= 0x20 && data[index] != 0x7f {
				index++
			}
			output.consumePrintable(data[start:index])
		case terminalEscape:
			switch byteValue {
			case '[':
				output.state = terminalCSI
				output.pending = output.pending[:0]
			case ']':
				output.state = terminalOSC
				output.pending = output.pending[:0]
				output.stringEscaped = false
			case 'P', '^', '_':
				output.state = terminalStringControl
				output.pending = output.pending[:0]
				output.stringEscaped = false
			case '(', ')', '*', '+', '-', '.', '/':
				output.state = terminalEscapeIntermediate
			case '7':
				output.active().savedRow, output.active().savedColumn = output.active().row, output.active().column
				output.state = terminalGround
			case '8':
				output.pendingWrap = false
				output.active().row, output.active().column = output.active().savedRow, output.active().savedColumn
				output.clampCursor()
				output.state = terminalGround
			case 'D':
				output.pendingWrap = false
				output.lineFeed()
				output.state = terminalGround
			case 'E':
				output.pendingWrap = false
				output.lineFeed()
				output.active().column = 0
				output.state = terminalGround
			case 'M':
				output.pendingWrap = false
				output.reverseIndex()
				output.state = terminalGround
			case 'c':
				output.reset()
				output.state = terminalGround
			case '=', '>':
				output.state = terminalGround
			default:
				output.markUnavailable()
				output.state = terminalGround
			}
			index++
		case terminalEscapeIntermediate:
			output.state = terminalGround
			index++
		case terminalCSI:
			if byteValue >= 0x40 && byteValue <= 0x7e {
				if !output.controlDiscard {
					replies = append(replies, output.csi(string(output.pending), byteValue)...)
				} else if byteValue == 'h' || byteValue == 'l' {
					output.decckmKnown = false
				}
				output.pending = output.pending[:0]
				output.controlDiscard = false
				output.state = terminalGround
				index++
				continue
			}
			if output.controlDiscard {
				index++
				continue
			}
			if len(output.pending) >= terminalPendingLimit {
				output.markUnavailable()
				output.pending = output.pending[:0]
				output.controlDiscard = true
				index++
				continue
			}
			output.pending = append(output.pending, byteValue)
			index++
		case terminalOSC, terminalStringControl:
			if byteValue == 0x07 && output.state == terminalOSC {
				output.state = terminalGround
				output.pending = output.pending[:0]
				output.controlDiscard = false
				output.stringEscaped = false
				index++
				continue
			}
			if output.stringEscaped {
				if byteValue == '\\' {
					output.state = terminalGround
					output.pending = output.pending[:0]
					output.controlDiscard = false
					output.stringEscaped = false
					index++
					continue
				}
				output.stringEscaped = false
			}
			if byteValue == terminalEscapeByte {
				output.stringEscaped = true
				index++
				continue
			}
			if output.controlDiscard {
				index++
				continue
			}
			if len(output.pending) >= terminalPendingLimit {
				output.markUnavailable()
				output.pending = output.pending[:0]
				output.controlDiscard = true
			} else {
				output.pending = append(output.pending, byteValue)
			}
			index++
		default:
			panic("shelltest: invalid terminal parser state")
		}
	}
	return replies
}

func (output *terminalOutput) control(value byte) {
	switch value {
	case '\r':
		output.pendingWrap = false
		output.active().column = 0
	case '\n', '\v', '\f':
		output.pendingWrap = false
		output.lineFeed()
	case '\b':
		output.pendingWrap = false
		if output.active().column > 0 {
			output.active().column--
		}
	case '\t':
		output.pendingWrap = false
		next := min(((output.active().column/terminalTabWidth)+1)*terminalTabWidth, terminalColumns-1)
		output.active().column = next
	case terminalNullByte, terminalBellByte: // NUL/BEL do not change presentation.
	default: // Other C0 controls have no screen model effect here.
	}
}

func (output *terminalOutput) consumePrintable(data []byte) {
	if len(data) == 0 {
		return
	}
	combined := make([]byte, 0, len(output.pendingUTF8)+len(data))
	combined = append(combined, output.pendingUTF8...)
	combined = append(combined, data...)
	output.pendingUTF8 = output.pendingUTF8[:0]

	var decoded strings.Builder
	for len(combined) != 0 {
		if !utf8.FullRune(combined) {
			if len(combined) > utf8.UTFMax-1 {
				output.markUnavailable()
				combined = combined[1:]
				continue
			}
			output.pendingUTF8 = append(output.pendingUTF8[:0], combined...)
			break
		}
		r, size := utf8.DecodeRune(combined)
		if r == utf8.RuneError && size == 1 {
			output.markUnavailable()
			combined = combined[1:]
			continue
		}
		decoded.Write(combined[:size])
		combined = combined[size:]
	}
	if decoded.Len() == 0 {
		return
	}

	text := output.pendingGrapheme + decoded.String()
	output.pendingGrapheme = ""
	iterator := graphemes.FromString(text)
	last := ""
	for iterator.Next() {
		cluster := iterator.Value()
		if last != "" {
			output.emitGrapheme(last)
		}
		last = cluster
	}
	if last == "" {
		return
	}
	if len(last) > terminalCellTextLimit {
		output.markUnavailable()
		return
	}
	output.pendingGrapheme = last
}

func (output *terminalOutput) flushPrintableBoundary() {
	if len(output.pendingUTF8) != 0 {
		output.markUnavailable()
		output.pendingUTF8 = output.pendingUTF8[:0]
	}
	if output.pendingGrapheme == "" {
		return
	}
	cluster := output.pendingGrapheme
	output.pendingGrapheme = ""
	output.emitGrapheme(cluster)
}

func (output *terminalOutput) emitGrapheme(cluster string) {
	if !output.available {
		return
	}
	if len(cluster) > terminalCellTextLimit {
		output.markUnavailable()
		return
	}
	for _, r := range cluster {
		if runewidth.IsAmbiguousWidth(r) {
			output.markUnavailable()
			return
		}
	}
	condition := runewidth.Condition{EastAsianWidth: false}
	width := condition.StringWidth(cluster)
	if width < 0 || width > terminalDoubleCellWidth {
		output.markUnavailable()
		return
	}
	if width == 0 {
		output.appendZeroWidth(cluster)
		return
	}
	output.writeCluster(cluster, width)
}

func (output *terminalOutput) appendZeroWidth(cluster string) {
	grid := output.active()
	column := grid.column - 1
	if output.pendingWrap {
		column = grid.column
	}
	if column < 0 || column >= terminalColumns {
		output.markUnavailable()
		return
	}
	for column > 0 && grid.continuation[grid.row][column] {
		column--
	}
	if grid.cells[grid.row][column] == "" || len(grid.cells[grid.row][column])+len(cluster) > terminalCellTextLimit {
		output.markUnavailable()
		return
	}
	grid.cells[grid.row][column] += cluster
}

func (output *terminalOutput) writeCluster(cluster string, width int) {
	grid := output.active()
	if output.pendingWrap {
		if output.autowrap {
			output.pendingWrap = false
			output.lineFeed()
			grid = output.active()
			grid.column = 0
		} else {
			output.pendingWrap = false
		}
	}
	if width == terminalDoubleCellWidth && grid.column == terminalColumns-1 {
		if !output.autowrap {
			output.markUnavailable()
			return
		}
		output.lineFeed()
		grid = output.active()
		grid.column = 0
	}
	grid.cells[grid.row][grid.column] = cluster
	grid.continuation[grid.row][grid.column] = false
	if width == terminalDoubleCellWidth {
		grid.cells[grid.row][grid.column+1] = ""
		grid.continuation[grid.row][grid.column+1] = true
	}
	if grid.column+width >= terminalColumns {
		grid.column = terminalColumns - 1
		output.pendingWrap = output.autowrap
		return
	}
	grid.column += width
}

func (output *terminalOutput) csi(raw string, final byte) [][]byte {
	if final == 'u' && isKeyboardProtocolCSI(raw) {
		return nil
	}
	if final == 'q' && strings.HasSuffix(raw, " ") {
		if _, ok := parseCSIParameters(strings.TrimSuffix(raw, " ")); ok {
			return nil
		}
	}

	private := byte(0)
	if raw != "" && (raw[0] == '?' || raw[0] == '>' || raw[0] == '!') {
		private = raw[0]
		raw = raw[1:]
	}
	params, ok := parseCSIParameters(raw)
	if !ok {
		output.markUnavailable()
		if final == 'h' || final == 'l' {
			output.decckmKnown = false
		}
		return nil
	}
	value := func(index, fallback int) int {
		if index >= len(params) || params[index] == 0 {
			return fallback
		}
		return params[index]
	}
	grid := output.active()
	switch final {
	case 'A':
		output.pendingWrap = false
		grid.row = max(0, grid.row-value(0, 1))
	case 'B', 'e':
		output.pendingWrap = false
		grid.row = min(terminalRows-1, grid.row+value(0, 1))
	case 'C', 'a':
		output.pendingWrap = false
		grid.column = min(terminalColumns-1, grid.column+value(0, 1))
	case 'D':
		output.pendingWrap = false
		grid.column = max(0, grid.column-value(0, 1))
	case 'E':
		output.pendingWrap = false
		grid.row = min(terminalRows-1, grid.row+value(0, 1))
		grid.column = 0
	case 'F':
		output.pendingWrap = false
		grid.row = max(0, grid.row-value(0, 1))
		grid.column = 0
	case 'G', '`':
		output.pendingWrap = false
		grid.column = min(terminalColumns-1, max(0, value(0, 1)-1))
	case 'd':
		output.pendingWrap = false
		grid.row = min(terminalRows-1, max(0, value(0, 1)-1))
	case 'H', 'f':
		output.pendingWrap = false
		grid.row = min(terminalRows-1, max(0, value(0, 1)-1))
		grid.column = min(terminalColumns-1, max(0, value(1, 1)-1))
	case 'J':
		output.pendingWrap = false
		output.eraseDisplay(value(0, 0))
	case 'K':
		output.pendingWrap = false
		output.eraseLine(value(0, 0))
	case 'm', 'q': // SGR and cursor styling do not alter retained text/cursor truth.
	case 'r':
		if private != 0 {
			output.markUnavailable()
			break
		}
		top := value(0, 1) - 1
		bottom := value(1, terminalRows) - 1
		if top < 0 || bottom < 0 || top >= bottom || bottom >= terminalRows {
			output.markUnavailable()
			break
		}
		output.scrollTop = top
		output.scrollBottom = bottom
		output.pendingWrap = false
		grid.row = 0
		grid.column = 0
	case 's':
		grid.savedRow, grid.savedColumn = grid.row, grid.column
	case 'u':
		output.pendingWrap = false
		grid.row, grid.column = grid.savedRow, grid.savedColumn
		output.clampCursor()
	case '@':
		output.pendingWrap = false
		output.insertCharacters(value(0, 1))
	case 'P':
		output.pendingWrap = false
		output.deleteCharacters(value(0, 1))
	case 'X':
		output.pendingWrap = false
		output.eraseCharacters(value(0, 1))
	case 'L':
		output.pendingWrap = false
		output.insertLines(value(0, 1))
	case 'M':
		output.pendingWrap = false
		output.deleteLines(value(0, 1))
	case 'S':
		output.pendingWrap = false
		output.scrollUp(value(0, 1))
	case 'T':
		output.pendingWrap = false
		output.scrollDown(value(0, 1))
	case 'h', 'l':
		if private != '?' {
			output.markUnavailable()
			break
		}
		set := final == 'h'
		for _, mode := range params {
			switch mode {
			case 1:
				output.decckm = set
				output.decckmKnown = true
			case terminalPrivateAutowrap:
				output.autowrap = set
				if !set {
					output.pendingWrap = false
				}
			case terminalPrivateAlternate, terminalPrivateAlternateSave, terminalPrivateAlternateCursor:
				output.setAlternate(mode, set)
			case terminalPrivateCursorBlink, terminalPrivateCursorVisibility, terminalPrivateBracketedPaste, terminalPrivateMouseX10, terminalPrivateMouseButton, terminalPrivateMouseAny, terminalPrivateFocus, terminalPrivateMouseUTF8, terminalPrivateMouseSGR, terminalPrivateMouseURXVT, terminalPrivateMeta8Bit:
				// Cursor, paste, focus, mouse, and keyboard-input modes do not change retained screen text.
			default:
				output.markUnavailable()
			}
		}
	case 'n':
		if private != 0 {
			output.recordReplyFailureLocked(failureError{subject: "unsupported private terminal status query"})
			break
		}
		switch value(0, 0) {
		case terminalStatusReport:
			return [][]byte{[]byte("\x1b[0n")}
		case terminalCursorReport:
			if !output.available {
				output.recordReplyFailureLocked(failureError{subject: "terminal cursor position is unavailable"})
				break
			}
			return [][]byte{[]byte(fmt.Sprintf("\x1b[%d;%dR", grid.row+1, grid.column+1))}
		default:
			output.recordReplyFailureLocked(failureError{subject: "unsupported terminal status query"})
		}
	case 't':
		if private != 0 || value(0, 0) != terminalGeometryReport {
			output.recordReplyFailureLocked(failureError{subject: "unsupported terminal geometry query"})
			break
		}
		return [][]byte{[]byte(fmt.Sprintf("\x1b[8;%d;%dt", terminalRows, terminalColumns))}
	case 'c':
		output.recordReplyFailureLocked(failureError{subject: "unsupported terminal device-attributes query"})
	default:
		output.markUnavailable()
	}
	return nil
}

func isKeyboardProtocolCSI(raw string) bool {
	if raw == "" || !strings.ContainsRune("?=><", rune(raw[0])) {
		return false
	}
	_, ok := parseCSIParameters(raw[1:])
	return ok
}

func parseCSIParameters(raw string) ([]int, bool) {
	if raw == "" {
		return nil, true
	}
	if strings.ContainsAny(raw, ":<=>") {
		return nil, false
	}
	parts := strings.Split(raw, ";")
	values := make([]int, len(parts))
	for index, part := range parts {
		if part == "" {
			continue
		}
		for _, r := range part {
			if !unicode.IsDigit(r) {
				return nil, false
			}
		}
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return nil, false
		}
		values[index] = value
	}
	return values, true
}

func (output *terminalOutput) active() *terminalGrid {
	if output.alternateActive {
		return &output.alternate
	}
	return &output.primary
}

func (output *terminalOutput) setAlternate(mode int, active bool) {
	if active == output.alternateActive {
		return
	}
	if active {
		if mode == terminalPrivateAlternateCursor {
			output.primary.savedRow = output.primary.row
			output.primary.savedColumn = output.primary.column
		}
		if mode == terminalPrivateAlternateSave || mode == terminalPrivateAlternateCursor {
			output.alternate = terminalGrid{}
		}
		output.pendingWrap = false
		output.scrollTop = 0
		output.scrollBottom = terminalRows - 1
		output.alternateActive = true
		return
	}
	output.alternateActive = false
	output.pendingWrap = false
	output.scrollTop = 0
	output.scrollBottom = terminalRows - 1
	if mode == terminalPrivateAlternateCursor {
		output.primary.row = output.primary.savedRow
		output.primary.column = output.primary.savedColumn
		output.clampCursor()
	}
}

func (output *terminalOutput) reset() {
	output.primary = terminalGrid{}
	output.alternate = terminalGrid{}
	output.alternateActive = false
	output.decckmKnown = true
	output.decckm = false
	output.autowrap = true
	output.pendingWrap = false
	output.scrollTop = 0
	output.scrollBottom = terminalRows - 1
	output.pending = output.pending[:0]
	output.pendingUTF8 = output.pendingUTF8[:0]
	output.pendingGrapheme = ""
	output.controlDiscard = false
}

func (output *terminalOutput) lineFeed() {
	grid := output.active()
	if grid.row == output.scrollBottom {
		output.scrollUp(1)
		return
	}
	if grid.row < terminalRows-1 {
		grid.row++
	}
}

func (output *terminalOutput) reverseIndex() {
	grid := output.active()
	if grid.row == output.scrollTop {
		output.scrollDown(1)
		return
	}
	if grid.row > 0 {
		grid.row--
	}
}

func (output *terminalOutput) scrollUp(count int) {
	grid := output.active()
	height := output.scrollBottom - output.scrollTop + 1
	count = min(max(count, 0), height)
	if count == 0 {
		return
	}
	for row := output.scrollTop; row <= output.scrollBottom-count; row++ {
		grid.cells[row] = grid.cells[row+count]
		grid.continuation[row] = grid.continuation[row+count]
	}
	for row := output.scrollBottom - count + 1; row <= output.scrollBottom; row++ {
		grid.cells[row] = [terminalColumns]string{}
		grid.continuation[row] = [terminalColumns]bool{}
	}
}

func (output *terminalOutput) scrollDown(count int) {
	grid := output.active()
	height := output.scrollBottom - output.scrollTop + 1
	count = min(max(count, 0), height)
	if count == 0 {
		return
	}
	for row := output.scrollBottom; row >= output.scrollTop+count; row-- {
		grid.cells[row] = grid.cells[row-count]
		grid.continuation[row] = grid.continuation[row-count]
	}
	for row := output.scrollTop; row < output.scrollTop+count; row++ {
		grid.cells[row] = [terminalColumns]string{}
		grid.continuation[row] = [terminalColumns]bool{}
	}
}

func (output *terminalOutput) eraseDisplay(mode int) {
	grid := output.active()
	switch mode {
	case 0:
		output.eraseLine(0)
		for row := grid.row + 1; row < terminalRows; row++ {
			grid.cells[row] = [terminalColumns]string{}
			grid.continuation[row] = [terminalColumns]bool{}
		}
	case 1:
		output.eraseLine(1)
		for row := range grid.row {
			grid.cells[row] = [terminalColumns]string{}
			grid.continuation[row] = [terminalColumns]bool{}
		}
	case terminalEraseAll, terminalEraseScrollback:
		grid.cells = [terminalRows][terminalColumns]string{}
		grid.continuation = [terminalRows][terminalColumns]bool{}
	default:
		output.markUnavailable()
	}
}

func (output *terminalOutput) eraseLine(mode int) {
	grid := output.active()
	start, end := grid.column, terminalColumns
	switch mode {
	case 0:
	case 1:
		start, end = 0, min(grid.column+1, terminalColumns)
	case terminalEraseAll:
		start, end = 0, terminalColumns
	default:
		output.markUnavailable()
		return
	}
	for column := start; column < end; column++ {
		grid.cells[grid.row][column] = ""
		grid.continuation[grid.row][column] = false
	}
}

func (output *terminalOutput) insertCharacters(count int) {
	grid := output.active()
	count = min(max(count, 0), terminalColumns-grid.column)
	if count == 0 {
		return
	}
	row := grid.row
	copy(grid.cells[row][grid.column+count:], grid.cells[row][grid.column:terminalColumns-count])
	copy(grid.continuation[row][grid.column+count:], grid.continuation[row][grid.column:terminalColumns-count])
	for column := grid.column; column < grid.column+count; column++ {
		grid.cells[row][column] = ""
		grid.continuation[row][column] = false
	}
}

func (output *terminalOutput) deleteCharacters(count int) {
	grid := output.active()
	count = min(max(count, 0), terminalColumns-grid.column)
	if count == 0 {
		return
	}
	row := grid.row
	copy(grid.cells[row][grid.column:], grid.cells[row][grid.column+count:])
	copy(grid.continuation[row][grid.column:], grid.continuation[row][grid.column+count:])
	for column := terminalColumns - count; column < terminalColumns; column++ {
		grid.cells[row][column] = ""
		grid.continuation[row][column] = false
	}
}

func (output *terminalOutput) eraseCharacters(count int) {
	grid := output.active()
	end := min(terminalColumns, grid.column+max(count, 0))
	for column := grid.column; column < end; column++ {
		grid.cells[grid.row][column] = ""
		grid.continuation[grid.row][column] = false
	}
}

func (output *terminalOutput) insertLines(count int) {
	grid := output.active()
	if grid.row < output.scrollTop || grid.row > output.scrollBottom {
		return
	}
	count = min(max(count, 0), output.scrollBottom-grid.row+1)
	for row := output.scrollBottom; row >= grid.row+count; row-- {
		grid.cells[row] = grid.cells[row-count]
		grid.continuation[row] = grid.continuation[row-count]
	}
	for row := grid.row; row < grid.row+count; row++ {
		grid.cells[row] = [terminalColumns]string{}
		grid.continuation[row] = [terminalColumns]bool{}
	}
}

func (output *terminalOutput) deleteLines(count int) {
	grid := output.active()
	if grid.row < output.scrollTop || grid.row > output.scrollBottom {
		return
	}
	count = min(max(count, 0), output.scrollBottom-grid.row+1)
	for row := grid.row; row <= output.scrollBottom-count; row++ {
		grid.cells[row] = grid.cells[row+count]
		grid.continuation[row] = grid.continuation[row+count]
	}
	for row := output.scrollBottom - count + 1; row <= output.scrollBottom; row++ {
		grid.cells[row] = [terminalColumns]string{}
		grid.continuation[row] = [terminalColumns]bool{}
	}
}

func (output *terminalOutput) clampCursor() {
	grid := output.active()
	grid.row = min(max(grid.row, 0), terminalRows-1)
	grid.column = min(max(grid.column, 0), terminalColumns-1)
}

func (output *terminalOutput) snapshot(expectedGeneration uint64) ([]string, bool) {
	output.mu.Lock()
	defer output.mu.Unlock()
	if output.generation != expectedGeneration {
		return nil, false
	}
	output.flushPrintableBoundary()
	if output.replyFailure != nil {
		return nil, true
	}
	if !output.available || output.state != terminalGround {
		return nil, true
	}
	grid := output.active()
	lines := make([]string, terminalRows)
	for row := range terminalRows {
		var builder strings.Builder
		for column := range terminalColumns {
			if grid.continuation[row][column] {
				continue
			}
			cell := grid.cells[row][column]
			if cell == "" {
				builder.WriteByte(' ')
			} else {
				builder.WriteString(cell)
			}
		}
		lines[row] = strings.TrimRight(builder.String(), " ")
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines, true
}

func (output *terminalOutput) mode() (bool, bool) {
	output.mu.Lock()
	defer output.mu.Unlock()
	return output.decckm, output.decckmKnown && output.state == terminalGround
}

func (output *terminalOutput) currentGeneration() uint64 {
	output.mu.Lock()
	defer output.mu.Unlock()
	return output.generation
}

func (output *terminalOutput) failure() error {
	output.mu.Lock()
	defer output.mu.Unlock()
	return output.replyFailure
}

func (output *terminalOutput) diagnosticTail() []byte {
	output.mu.Lock()
	defer output.mu.Unlock()
	if !output.tailTruncated {
		return bytes.Clone(output.tail)
	}
	const marker = "…[truncated]\n"
	limit := terminalDiagnosticTailLimit - len(marker)
	if limit < 0 {
		return []byte(marker)
	}
	start := max(0, len(output.tail)-limit)
	result := make([]byte, 0, len(marker)+len(output.tail)-start)
	result = append(result, marker...)
	result = append(result, output.tail[start:]...)
	return result
}

func (output *terminalOutput) appendTail(data []byte) {
	if len(data) >= terminalDiagnosticTailLimit {
		output.tailTruncated = output.tailTruncated || len(output.tail) != 0 || len(data) > terminalDiagnosticTailLimit
		output.tail = append(output.tail[:0], data[len(data)-terminalDiagnosticTailLimit:]...)
		return
	}
	if len(output.tail)+len(data) > terminalDiagnosticTailLimit {
		output.tailTruncated = true
		drop := len(output.tail) + len(data) - terminalDiagnosticTailLimit
		copy(output.tail, output.tail[drop:])
		output.tail = output.tail[:len(output.tail)-drop]
	}
	output.tail = append(output.tail, data...)
}

func (output *terminalOutput) markUnavailable() {
	output.available = false
}

func (output *terminalOutput) recordReplyFailure(err error) {
	if err == nil {
		return
	}
	output.mu.Lock()
	output.recordReplyFailureLocked(err)
	output.mu.Unlock()
	select {
	case output.changed <- struct{}{}:
	default:
	}
}

func (output *terminalOutput) recordReplyFailureLocked(err error) {
	if err != nil && output.replyFailure == nil {
		output.replyFailure = err
	}
}
