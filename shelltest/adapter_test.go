package shelltest

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestShellAdapterJSONAndRecordsDecodeEquivalentCommonRequest(t *testing.T) {
	t.Parallel()
	jsonRequest := "json\n" + `{"response_encoding":"records","shell":{"family":"bash","version":"5.2"},"input":{"topology":"exact","words":[{"value":"alienctl","fidelity":"exact"}],"cursor":{"word":0,"byte":8,"space":"observed-word-bytes"}},"matching":"prefix","capabilities":["descriptions","future-capability"],"native_capabilities":["files","future-native"]}`
	recordsRequest := "records\n" + strings.Join([]string{
		"response-encoding\trecords",
		"shell\tbash\t5.2",
		"input\texact\t1\t0\t8\tobserved-word-bytes",
		"word\t0\talienctl\texact",
		"matching\tprefix",
		"capability\tdescriptions",
		"capability\tfuture-capability",
		"native-capability\tfiles",
		"native-capability\tfuture-native",
		"end",
		"",
	}, "\n")

	jsonDecoded, err := decodeShellAdapterRequest([]byte(jsonRequest))
	if err != nil {
		t.Fatal(err)
	}
	recordsDecoded, err := decodeShellAdapterRequest([]byte(recordsRequest))
	if err != nil {
		t.Fatal(err)
	}
	if jsonDecoded.responseCodec != recordsDecoded.responseCodec ||
		jsonDecoded.shellFamily != recordsDecoded.shellFamily ||
		jsonDecoded.shellVersion != recordsDecoded.shellVersion ||
		jsonDecoded.matching != recordsDecoded.matching ||
		jsonDecoded.capabilities != recordsDecoded.capabilities ||
		jsonDecoded.input.Topology != recordsDecoded.input.Topology ||
		jsonDecoded.input.ExactArgvPrefix != recordsDecoded.input.ExactArgvPrefix ||
		jsonDecoded.input.Cursor != recordsDecoded.input.Cursor ||
		!slices.Equal(jsonDecoded.input.Words, recordsDecoded.input.Words) {
		t.Fatalf("JSON and Records requests differ:\nJSON: %#v\nRecords: %#v", jsonDecoded, recordsDecoded)
	}
	if jsonDecoded.input.ExactArgvPrefix != len(jsonDecoded.input.Words) {
		t.Fatalf("exact JSON default frontier = %d, want %d", jsonDecoded.input.ExactArgvPrefix, len(jsonDecoded.input.Words))
	}
}

func TestShellAdapterJSONRejectsMalformedCore(t *testing.T) {
	t.Parallel()
	valid := `{"shell":{"family":"bash","version":"5.2"},"input":{"words":[{"value":"alienctl"}],"cursor":{"word":0,"byte":0}}}`
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "duplicate",
			input: []byte("json\n" + `{"shell":{"family":"bash","version":"5.2"},"shell":{"family":"bash","version":"5.2"},"input":{"words":[{"value":"alienctl"}],"cursor":{"word":0,"byte":0}}}`),
		},
		{
			name:  "unknown",
			input: []byte("json\n" + strings.TrimSuffix(valid, "}") + `,"unknown":true}`),
		},
		{
			name:  "null",
			input: []byte("json\n" + `{"shell":null,"input":{"words":[{"value":"alienctl"}],"cursor":{"word":0,"byte":0}}}`),
		},
		{
			name:  "fraction",
			input: []byte("json\n" + `{"shell":{"family":"bash","version":"5.2"},"input":{"words":[{"value":"alienctl"}],"cursor":{"word":0,"byte":0.5}}}`),
		},
		{
			name:  "trailing",
			input: []byte("json\n" + valid + ` {}`),
		},
		{
			name:  "utf8",
			input: append([]byte("json\n"+valid[:len(valid)-2]), 0xff, '}', '}'),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := decodeShellAdapterRequest(test.input); err == nil {
				t.Fatal("malformed JSON request accepted")
			}
		})
	}
}

func TestShellAdapterRecordsCanonicalityAndByteDomain(t *testing.T) {
	t.Parallel()
	validPrefix := "records\nshell\tbash\t5.2\ninput\texact\t1\t0\t0\tobserved-word-bytes\nword\t0\talienctl\texact\n"
	for _, test := range []struct {
		name   string
		suffix string
	}{
		{
			name:   "missing-end",
			suffix: "",
		},
		{
			name:   "duplicate-shell",
			suffix: "shell\tbash\t5.2\nend\n",
		},
		{
			name:   "noncanonical-int",
			suffix: "input\texact\t01\t0\t0\tobserved-word-bytes\nend\n",
		},
		{
			name:   "lowercase-hex",
			suffix: "word\t1\t\\x6a\texact\nend\n",
		},
		{
			name:   "trailing-after-end",
			suffix: "end\nshell\tbash\t5.2\n",
		},
	} {
		name := test.name
		suffix := test.suffix
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			input := validPrefix + suffix
			if name == "noncanonical-int" {
				input = "records\nshell\tbash\t5.2\n" + suffix
			}
			if _, err := decodeShellAdapterRequest([]byte(input)); err == nil {
				t.Fatal("malformed Records request accepted")
			}
		})
	}

	byteRequest := "records\nshell\tba\\xFFsh\t5.2\ninput\texact\t1\t0\t0\tobserved-word-bytes\nword\t0\talienctl\texact\nend\n"
	decoded, err := decodeShellAdapterRequest([]byte(byteRequest))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.shellFamily != string([]byte{'b', 'a', 0xff, 's', 'h'}) {
		t.Fatalf("Records shell family = %q", decoded.shellFamily)
	}
}

func TestShellAdapterRejectsArgumentByteCursorWithoutExactEvidence(t *testing.T) {
	t.Parallel()
	input := []byte("records\nshell\tbash\t5.2\ninput\tprefix-exact\t1\t0\t1\targument-bytes\nword\t0\tx\tlexical\nend\n")
	if _, err := decodeShellAdapterRequest(input); err == nil {
		t.Fatal("argument-byte cursor accepted for nonexact word")
	}
}

func TestShellAdapterResponseEncodingPreservesRequestedProjection(t *testing.T) {
	t.Parallel()
	response := shellAdapterResponse{
		failed: true,
		replace: cli.CompletionSpan{
			Argument: 1,
			Start:    2,
			End:      3,
		},
		spacing: cli.CompletionSpacingAppendSpace,
		items: []shellAdapterItem{{
			value:          "remote",
			replacementSet: true,
			replacement:    []string{"remote"},
			spacing:        cli.CompletionSpacingNoSpace,
			description:    "description",
		}},
	}
	var jsonOutput bytes.Buffer
	if err := encodeShellAdapterResponse(&jsonOutput, shellAdapterCodecJSON, response); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(jsonOutput.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if failed, _ := decoded["failed"].(bool); !failed {
		t.Fatalf("JSON failed projection = %#v", decoded["failed"])
	}

	var recordsOutput bytes.Buffer
	if err := encodeShellAdapterResponse(&recordsOutput, shellAdapterCodecRecords, response); err != nil {
		t.Fatal(err)
	}
	text := recordsOutput.String()
	for _, record := range []string{"result\ttrue\n", "replace\t1\t2\t3\n", "item\t0\tremote\ttrue\tno-space\n", "end\n"} {
		if !strings.Contains(text, record) {
			t.Fatalf("Records response missing %q in %q", record, text)
		}
	}
}

func TestShellAdapterJSONMinimalDefaultsAndNestedStrictness(t *testing.T) {
	t.Parallel()
	minimal := []byte("json\n" + `{"shell":{"family":"bash","version":"5.2"},"input":{"words":[{"value":"alienctl"}],"cursor":{"word":0,"byte":8}}}`)
	request, err := decodeShellAdapterRequest(minimal)
	if err != nil {
		t.Fatal(err)
	}
	if request.responseCodec != shellAdapterCodecJSON || request.matching != shellAdapterMatchingPrefix {
		t.Fatalf("defaults = codec %v, matching %v", request.responseCodec, request.matching)
	}
	if request.input.Topology != cli.CompletionArgvTopologyUncertain || request.input.ExactArgvPrefix != 0 {
		t.Fatalf("input defaults = topology %v, prefix %d", request.input.Topology, request.input.ExactArgvPrefix)
	}
	if request.input.Words[0].Fidelity != cli.CompletionInputFidelityLexical || request.input.Cursor.Space != cli.CompletionCursorSpaceObservedWordBytes {
		t.Fatalf("word/cursor defaults = fidelity %v, space %v", request.input.Words[0].Fidelity, request.input.Cursor.Space)
	}

	for _, test := range []struct {
		name    string
		payload string
	}{
		{
			name:    "nested-duplicate",
			payload: `{"shell":{"family":"bash","family":"zsh","version":"5.2"},"input":{"words":[{"value":"alienctl"}],"cursor":{"word":0,"byte":0}}}`,
		},
		{
			name:    "nested-unknown",
			payload: `{"shell":{"family":"bash","version":"5.2","future":true},"input":{"words":[{"value":"alienctl"}],"cursor":{"word":0,"byte":0}}}`,
		},
		{
			name:    "wrong-type",
			payload: `{"shell":{"family":"bash","version":"5.2"},"input":{"words":{},"cursor":{"word":0,"byte":0}}}`,
		},
		{
			name:    "null-word",
			payload: `{"shell":{"family":"bash","version":"5.2"},"input":{"words":[null],"cursor":{"word":0,"byte":0}}}`,
		},
	} {
		name := test.name
		payload := test.payload
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := decodeShellAdapterRequest([]byte("json\n" + payload)); err == nil {
				t.Fatal("invalid nested JSON accepted")
			}
		})
	}
}

func TestShellAdapterExactTopologyIgnoresSerializedPrefixAndUncertainRequiresZero(t *testing.T) {
	t.Parallel()
	exact := []byte("records\nshell\tbash\t5.2\ninput\texact\t1\t1\t1\tobserved-word-bytes\nword\t0\talienctl\texact\nword\t1\tx\tlexical\nend\n")
	request, err := decodeShellAdapterRequest(exact)
	if err != nil {
		t.Fatal(err)
	}
	if request.input.ExactArgvPrefix != 2 {
		t.Fatalf("exact semantic prefix = %d, want 2", request.input.ExactArgvPrefix)
	}

	uncertain := []byte("records\nshell\tbash\t5.2\ninput\tuncertain\t1\t0\t0\tobserved-word-bytes\nword\t0\talienctl\tlexical\nend\n")
	if _, err := decodeShellAdapterRequest(uncertain); err == nil {
		t.Fatal("uncertain topology accepted nonzero exact prefix")
	}
}

func TestShellAdapterRecordsRejectsNegativeAndMalformedFields(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name  string
		input []byte
	}{
		{
			name:  "negative",
			input: []byte("records\nshell\tbash\t5.2\ninput\tuncertain\t-1\t0\t0\tobserved-word-bytes\nword\t0\talienctl\tlexical\nend\n"),
		},
		{
			name:  "nul",
			input: append([]byte("records\nshell\tba"), append([]byte{0}, []byte("sh\t5.2\ninput\tuncertain\t0\t0\t0\tobserved-word-bytes\nword\t0\talienctl\tlexical\nend\n")...)...),
		},
		{
			name:  "escape",
			input: []byte("records\nshell\tba\\qsh\t5.2\ninput\tuncertain\t0\t0\t0\tobserved-word-bytes\nword\t0\talienctl\tlexical\nend\n"),
		},
	} {
		name := test.name
		input := test.input
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := decodeShellAdapterRequest(input); err == nil {
				t.Fatal("invalid Records request accepted")
			}
		})
	}
}

func TestShellAdapterWordLimitRejectsBeforeTypedGrowth(t *testing.T) {
	t.Parallel()
	var input strings.Builder
	input.Grow(2 << 20)
	input.WriteString("records\nshell\tbash\t5.2\ninput\tuncertain\t0\t0\t0\tobserved-word-bytes\n")
	for index := 0; index <= shellAdapterWordLimit; index++ {
		input.WriteString("word\t")
		input.WriteString(strconv.Itoa(index))
		input.WriteString("\tx\tlexical\n")
	}
	input.WriteString("end\n")
	if _, err := decodeShellAdapterRequest([]byte(input.String())); err == nil {
		t.Fatal("word-limit overflow accepted")
	}
}

func TestShellAdapterCompletionRequestUsesLastActivationAndExactResponseFileSnapshot(t *testing.T) {
	t.Parallel()
	request := shellAdapterRequest{
		shellFamily:  "bash",
		shellVersion: "5.2",
		input: cli.CompletionInputObservation{
			Topology: cli.CompletionArgvTopologyUncertain,
			Words: []cli.CompletionObservedWord{{
				Value:    "alienctl",
				Fidelity: cli.CompletionInputFidelityLexical,
			}},
			Cursor: cli.CompletionObservedCursor{
				Word:  0,
				Byte:  8,
				Space: cli.CompletionCursorSpaceObservedWordBytes,
			},
		},
		matching: shellAdapterMatchingPrefix,
	}
	environ := []string{"A=one", "COMMANDMENT_COMPLETE=old", "B=two", "COMMANDMENT_COMPLETE=adapter"}
	completionRequest, err := request.completionRequest(environ, "/work", "COMMANDMENT_COMPLETE", "adapter")
	if err != nil {
		t.Fatal(err)
	}
	wantEnv := cli.Env{{
		Name:  "A",
		Value: "one",
	}, {
		Name:  "B",
		Value: "two",
	}}
	if !slices.Equal(completionRequest.Env, wantEnv) {
		t.Fatalf("semantic environment = %#v, want %#v", completionRequest.Env, wantEnv)
	}
	if completionRequest.Trigger.Kind != cli.CompletionTriggerInvoked || completionRequest.Visibility != cli.CompletionVisibilityPublic || completionRequest.Client.Ranking != cli.CompletionRankingModeRelevance || completionRequest.Limit != 0 {
		t.Fatalf("completion request fixed semantics = %#v", completionRequest)
	}

	workspace := t.TempDir()
	file := filepath.Join(workspace, "args.txt")
	content := []byte("--remote café\n")
	if err := os.WriteFile(file, content, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := completionRequest.ResponseFiles.Load(t.Context(), cli.ResponseFileRequest{
		Path: "args.txt",
		Base: workspace,
	})
	if err != nil {
		t.Fatal(err)
	}
	absolute, err := filepath.Abs(file)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Key != absolute || loaded.Location != absolute || loaded.IncludeBase != filepath.Dir(absolute) || !bytes.Equal(loaded.Content, content) {
		t.Fatalf("response file = %#v", loaded)
	}
}

func TestShellAdapterProjectionGatesOptionalMaterialAndNeverLeaksPrivateMetadata(t *testing.T) {
	t.Parallel()
	completion := cli.Completion{
		Replace: cli.CompletionSpan{
			Argument: 1,
			Start:    2,
			End:      3,
		},
		Spacing: cli.CompletionSpacingAppendSpace,
		Items: []cli.CompletionItem{{
			Value:            "remote",
			Replacement:      []string{"remote"},
			DisplayText:      "Remote",
			Description:      "description",
			Detail:           "detail",
			Kind:             cli.CompletionCandidateCommand,
			GroupID:          "commands",
			Spacing:          cli.CompletionSpacingNoSpace,
			CommitCharacters: []rune{'/'},
			Sensitive:        true,
			TargetIndexes:    []int{7},
			Match: cli.CompletionMatch{
				Score: 999,
				Ranges: []cli.CompletionRange{{
					Start: 0,
					End:   2,
				}},
			},
			Required: true,
		}},
		Groups: []cli.CompletionGroup{{
			ID:        "commands",
			Label:     "Commands",
			Sensitive: true,
		}},
		Messages: []cli.CompletionMessage{{
			Kind:      cli.CompletionMessageWarning,
			Text:      "message",
			Sensitive: true,
		}},
		NativeRequests: []cli.CompletionNativeRequest{{
			Kind: cli.CompletionNativeFiles,
		}, {
			Kind: cli.CompletionNativeHosts,
		}},
		Delegation: &cli.CompletionDelegation{
			Args: []string{"child"},
			Cursor: cli.CompletionCursor{
				Argument: 0,
				Byte:     5,
			},
			Sensitive: true,
		},
	}

	minimal := projectShellAdapterResponse(completion, false, cli.CompletionClientCapabilities{})
	if minimal.items[0].description != "" || minimal.items[0].detail != "" || minimal.items[0].kind != cli.CompletionCandidateUnknown || minimal.items[0].group != "" || minimal.items[0].matchRanges != nil || minimal.items[0].commitCharacters != nil || minimal.groups != nil || minimal.messages != nil || minimal.native != nil || minimal.delegation != nil {
		t.Fatalf("minimal projection leaked optional material: %#v", minimal)
	}
	if minimal.items[0].displayText != "Remote" || !minimal.items[0].replacementSet || !slices.Equal(minimal.items[0].replacement, []string{"remote"}) {
		t.Fatalf("minimal projection lost core item material: %#v", minimal.items[0])
	}

	capabilities := cli.CompletionClientCapabilities{
		Descriptions:     true,
		Details:          true,
		Groups:           true,
		CandidateKinds:   true,
		MatchRanges:      true,
		CommitCharacters: true,
		Messages:         true,
		Delegation:       true,
		Native: cli.CompletionNativeCapabilities{
			Files: true,
		},
	}
	full := projectShellAdapterResponse(completion, true, capabilities)
	item := full.items[0]
	if !full.failed || item.description != "description" || item.detail != "detail" || item.kind != cli.CompletionCandidateCommand || item.group != "commands" {
		t.Fatalf("full item projection = %#v", item)
	}
	wantRanges := []cli.CompletionRange{
		{
			Start: 0,
			End:   2,
		},
	}
	if !slices.Equal(item.matchRanges, wantRanges) {
		t.Fatalf("full item match ranges = %#v, want %#v", item.matchRanges, wantRanges)
	}
	wantCommitCharacters := []string{"/"}
	if !slices.Equal(item.commitCharacters, wantCommitCharacters) {
		t.Fatalf("full item commit characters = %#v, want %#v", item.commitCharacters, wantCommitCharacters)
	}
	wantNative := []cli.CompletionNativeKind{cli.CompletionNativeFiles}
	if len(full.groups) != 1 || len(full.messages) != 1 || !slices.Equal(full.native, wantNative) || full.delegation == nil {
		t.Fatalf("full root projection = %#v", full)
	}
}

func TestShellAdapterResponseEncodingFailsAtBoundWithoutTruncation(t *testing.T) {
	t.Parallel()
	response := shellAdapterResponse{
		spacing: cli.CompletionSpacingDefault,
		items: []shellAdapterItem{{
			value:   strings.Repeat("x", 256),
			spacing: cli.CompletionSpacingDefault,
		}},
	}
	for _, codec := range []shellAdapterCodec{shellAdapterCodecJSON, shellAdapterCodecRecords} {
		writer := boundedWriter{
			limit: 32,
		}
		if err := encodeShellAdapterResponse(&writer, codec, response); err == nil {
			t.Fatalf("codec %v encoded past semantic response budget", codec)
		}
		if encoded, err := writer.bytes(); err == nil || encoded != nil {
			t.Fatalf("codec %v returned truncated semantic output %q, %v", codec, encoded, err)
		}
	}
}
