package shelltest

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

func testTerminalOutput(reply func([]byte) error) *terminalOutput {
	output := &terminalOutput{
		changed:      make(chan struct{}, 1),
		available:    true,
		decckmKnown:  true,
		autowrap:     true,
		scrollBottom: terminalRows - 1,
		reply:        reply,
	}
	return output
}

func TestTerminalOutputMaintainsLiveSemanticScreen(t *testing.T) {
	t.Parallel()
	output := testTerminalOutput(nil)
	if _, err := output.Write([]byte("prompt> abc\rprompt> abd\r\n\x1b[2Knext\r\n")); err != nil {
		t.Fatal(err)
	}
	got, ok := output.snapshot(output.currentGeneration())
	if !ok {
		t.Fatal("live terminal screen unavailable")
	}
	want := []string{"prompt> abd", "next"}
	if !slices.Equal(got, want) {
		t.Fatalf("screen = %#v, want %#v", got, want)
	}
}

func TestTerminalOutputDistinguishesObservedEmptyAndUncertain(t *testing.T) {
	t.Parallel()
	output := testTerminalOutput(nil)
	if _, err := output.Write([]byte("\x1b[2J")); err != nil {
		t.Fatal(err)
	}
	got, ok := output.snapshot(output.currentGeneration())
	if !ok || got == nil || len(got) != 0 {
		t.Fatalf("observed empty screen = %#v, %t", got, ok)
	}
	if _, err := output.Write([]byte("\x1b[999z")); err != nil {
		t.Fatal(err)
	}
	if got, ok := output.snapshot(output.currentGeneration()); !ok || got != nil {
		t.Fatalf("uncertain screen = %#v, %t", got, ok)
	}
}

func TestTerminalOutputAnswersCursorAndGeometryQueriesTruthfully(t *testing.T) {
	t.Parallel()
	var replies []string
	output := testTerminalOutput(func(data []byte) error { replies = append(replies, string(data)); return nil })
	if _, err := output.Write([]byte("abc\x1b[6n\x1b[18t")); err != nil {
		t.Fatal(err)
	}
	want := []string{"\x1b[1;4R", "\x1b[8;48;160t"}
	if !slices.Equal(replies, want) {
		t.Fatalf("terminal replies = %#v, want %#v", replies, want)
	}
}

func TestTerminalOutputIgnoresMetaInputModeForScreenTruth(t *testing.T) {
	t.Parallel()
	output := testTerminalOutput(nil)
	if _, err := output.Write([]byte("before\x1b[?1034h after\x1b[?1034l")); err != nil {
		t.Fatal(err)
	}
	got, ok := output.snapshot(output.currentGeneration())
	if !ok || !slices.Equal(got, []string{"before after"}) {
		t.Fatalf("screen = %#v, %t", got, ok)
	}
}

func TestTerminalOutputIgnoresKeyboardProtocolControlsForScreenTruth(t *testing.T) {
	t.Parallel()
	output := testTerminalOutput(nil)
	if _, err := output.Write([]byte("ab\x1b[scd\x1b[?uX\x1b[=5uY\x1b[>5uZ\x1b[<u")); err != nil {
		t.Fatal(err)
	}
	got, ok := output.snapshot(output.currentGeneration())
	if !ok || !slices.Equal(got, []string{"abcdXYZ"}) {
		t.Fatalf("screen = %#v, %t", got, ok)
	}
}

func TestTerminalOutputIgnoresCursorStyleForScreenTruth(t *testing.T) {
	t.Parallel()
	output := testTerminalOutput(nil)
	if _, err := output.Write([]byte("before\x1b[2 q after")); err != nil {
		t.Fatal(err)
	}
	got, ok := output.snapshot(output.currentGeneration())
	if !ok || !slices.Equal(got, []string{"before after"}) {
		t.Fatalf("screen = %#v, %t", got, ok)
	}
}

func TestTerminalOutputTracksDECCKM(t *testing.T) {
	t.Parallel()
	output := testTerminalOutput(nil)
	if mode, ok := output.mode(); !ok || mode {
		t.Fatalf("initial DECCKM = %t, %t", mode, ok)
	}
	_, _ = output.Write([]byte("\x1b[?1h"))
	if mode, ok := output.mode(); !ok || !mode {
		t.Fatalf("application DECCKM = %t, %t", mode, ok)
	}
	_, _ = output.Write([]byte("\x1b[?1l"))
	if mode, ok := output.mode(); !ok || mode {
		t.Fatalf("normal DECCKM = %t, %t", mode, ok)
	}
}

func TestTerminalOutputReassemblesUTF8AcrossReads(t *testing.T) {
	t.Parallel()
	output := testTerminalOutput(nil)
	encoded := []byte("\u754c")
	if _, err := output.Write(encoded[:1]); err != nil {
		t.Fatal(err)
	}
	if _, err := output.Write(encoded[1:]); err != nil {
		t.Fatal(err)
	}
	got, ok := output.snapshot(output.currentGeneration())
	if !ok || !slices.Equal(got, []string{"\u754c"}) {
		t.Fatalf("screen = %#v, %t", got, ok)
	}
}

func TestTerminalOutputIncompleteUTF8MakesPresentationUnavailable(t *testing.T) {
	t.Parallel()
	output := testTerminalOutput(nil)
	if _, err := output.Write([]byte{0xc3}); err != nil {
		t.Fatal(err)
	}
	got, ok := output.snapshot(output.currentGeneration())
	if !ok || got != nil {
		t.Fatalf("screen = %#v, %t", got, ok)
	}
}

func TestTerminalOutputPreservesGraphemeAcrossReads(t *testing.T) {
	t.Parallel()
	output := testTerminalOutput(nil)
	if _, err := output.Write([]byte("\U0001f468")); err != nil {
		t.Fatal(err)
	}
	if _, err := output.Write([]byte("\u200d\U0001f4bb")); err != nil {
		t.Fatal(err)
	}
	got, ok := output.snapshot(output.currentGeneration())
	if !ok || !slices.Equal(got, []string{"\U0001f468\u200d\U0001f4bb"}) {
		t.Fatalf("screen = %#v, %t", got, ok)
	}
}

func TestTerminalOutputAmbiguousWidthMakesPresentationUnavailable(t *testing.T) {
	t.Parallel()
	output := testTerminalOutput(nil)
	if _, err := output.Write([]byte("·")); err != nil {
		t.Fatal(err)
	}
	got, ok := output.snapshot(output.currentGeneration())
	if !ok || got != nil {
		t.Fatalf("screen = %#v, %t", got, ok)
	}
}

func TestTerminalOutputUnavailableRemainsUnavailableAfterReset(t *testing.T) {
	t.Parallel()
	output := testTerminalOutput(nil)
	if _, err := output.Write([]byte("\x1b[999z\x1bcvisible")); err != nil {
		t.Fatal(err)
	}
	got, ok := output.snapshot(output.currentGeneration())
	if !ok || got != nil {
		t.Fatalf("screen = %#v, %t", got, ok)
	}
}

func TestTerminalOutputAutowrapPreservesCursorTruth(t *testing.T) {
	t.Parallel()
	var replies []string
	output := testTerminalOutput(func(data []byte) error {
		replies = append(replies, string(data))
		return nil
	})
	if _, err := output.Write([]byte(strings.Repeat("a", terminalColumns) + "\x1b[6n")); err != nil {
		t.Fatal(err)
	}
	if _, err := output.Write([]byte("b\x1b[6n")); err != nil {
		t.Fatal(err)
	}
	want := []string{"\x1b[1;160R", "\x1b[2;2R"}
	if !slices.Equal(replies, want) {
		t.Fatalf("terminal replies = %#v, want %#v", replies, want)
	}
}

func TestTerminalOutputDiagnosticTailMarksTruncation(t *testing.T) {
	t.Parallel()
	output := testTerminalOutput(nil)
	if _, err := output.Write([]byte(strings.Repeat("x", terminalDiagnosticTailLimit+1))); err != nil {
		t.Fatal(err)
	}
	tail := string(output.diagnosticTail())
	if !strings.HasPrefix(tail, "…[truncated]\n") {
		t.Fatalf("diagnostic tail prefix = %q", tail[:min(len(tail), 32)])
	}
	if len(tail) > terminalDiagnosticTailLimit {
		t.Fatalf("diagnostic tail length = %d", len(tail))
	}
}

func TestTerminalOutputAnswersStatusQueryAndFailsDSRAfterTruthLoss(t *testing.T) {
	t.Parallel()
	var replies []string
	output := testTerminalOutput(func(data []byte) error {
		replies = append(replies, string(data))
		return nil
	})
	if _, err := output.Write([]byte("\x1b[5n")); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(replies, []string{"\x1b[0n"}) {
		t.Fatalf("status replies = %#v", replies)
	}
	if _, err := output.Write([]byte("\x1b[999z\x1b[6n")); err != nil {
		t.Fatal(err)
	}
	if err := output.failure(); err == nil {
		t.Fatal("DSR after presentation truth loss did not fail")
	}
	if len(replies) != 1 {
		t.Fatalf("fabricated DSR reply = %#v", replies[1:])
	}
}

func TestTerminalOutputAlternateScreenDoesNotMergePrimaryPresentation(t *testing.T) {
	t.Parallel()
	output := testTerminalOutput(nil)
	if _, err := output.Write([]byte("primary\x1b[?1049halt\x1b[?1049l")); err != nil {
		t.Fatal(err)
	}
	lines, ok := output.snapshot(output.currentGeneration())
	if !ok || !slices.Equal(lines, []string{"primary"}) {
		t.Fatalf("restored primary screen = %#v, %t", lines, ok)
	}
	if _, err := output.Write([]byte("\x1b[?1049hsecond")); err != nil {
		t.Fatal(err)
	}
	lines, ok = output.snapshot(output.currentGeneration())
	if !ok || !slices.Equal(lines, []string{"second"}) {
		t.Fatalf("active alternate screen = %#v, %t", lines, ok)
	}
}

func TestTerminalOutputBoundedOSCBecomesUnavailable(t *testing.T) {
	t.Parallel()
	output := testTerminalOutput(nil)
	payload := "\x1b]0;" + strings.Repeat("x", terminalPendingLimit+1) + "\a"
	if _, err := output.Write([]byte(payload)); err != nil {
		t.Fatal(err)
	}
	lines, ok := output.snapshot(output.currentGeneration())
	if !ok || lines != nil {
		t.Fatalf("oversize OSC screen = %#v, %t", lines, ok)
	}
}

func TestTerminalSessionEnterUsesDECCKMAndRejectsUnknownMode(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		sequence string
		want     string
	}{
		{
			name: "normal",
			want: "ab\x1b[D",
		},
		{
			name:     "application",
			sequence: "\x1b[?1h",
			want:     "ab\x1bOD",
		},
	} {
		name := test.name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var terminal bytes.Buffer
			output := testTerminalOutput(nil)
			if test.sequence != "" {
				if _, err := output.Write([]byte(test.sequence)); err != nil {
					t.Fatal(err)
				}
			}
			session := &terminalSession{
				input: &lockedWriter{
					writer: &terminal,
				},
				output: output,
			}
			if err := session.enter(Input{
				BeforeCursor: "a",
				AfterCursor:  "b",
			}); err != nil {
				t.Fatal(err)
			}
			if terminal.String() != test.want {
				t.Fatalf("terminal input = %q, want %q", terminal.String(), test.want)
			}
		})
	}

	var terminal bytes.Buffer
	output := testTerminalOutput(nil)
	if _, err := output.Write([]byte("\x1b[?")); err != nil {
		t.Fatal(err)
	}
	session := &terminalSession{
		input: &lockedWriter{
			writer: &terminal,
		},
		output: output,
	}
	if err := session.enter(Input{
		AfterCursor: "b",
	}); err == nil {
		t.Fatal("cursor movement guessed while DECCKM mode was unresolved")
	}
}
