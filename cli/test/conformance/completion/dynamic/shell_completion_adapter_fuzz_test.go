package conformance

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

const (
	adapterFuzzChildMode     = "COMMANDMENT_ADAPTER_FUZZ_CHILD"
	adapterFuzzRequestPath   = "COMMANDMENT_ADAPTER_FUZZ_REQUEST"
	adapterFuzzCandidatePath = "COMMANDMENT_ADAPTER_FUZZ_CANDIDATE"
	adapterFuzzOutputPath    = "COMMANDMENT_ADAPTER_FUZZ_OUTPUT"
	adapterFuzzResultPath    = "COMMANDMENT_ADAPTER_FUZZ_RESULT"
)

type adapterFuzzChildResult struct {
	DiagnosticCode string `json:"diagnostic_code,omitempty"`
}

func adapterFuzzApp(t testing.TB, candidate string) cli.Application {
	type config struct{ Value string }
	return clitest.Compile(t, cli.Definition[config]{Name: "tool", Help: cli.HelpConfig{Disabled: true}, Declare: func(root *cli.CommandBuilder[config]) {
		root.Flags.String(&root.Config.Value).CompleteBy(func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
			return cli.CompletionContribution{Candidates: []cli.CompletionCandidate{{Value: candidate}}}, nil
		}).Long("value")
	}})
}

func adapterFuzzCaptureGeneration(t testing.TB, app cli.Application) cli.ShellCompletionGeneration {
	t.Helper()
	var got cli.ShellCompletionGeneration
	provider := cli.Shell.Custom("probe", func(value cli.ShellCompletionGeneration) (string, error) { got = value; return "probe", nil })
	if _, err := app.ShellCompletionScript(provider); err != nil {
		t.Fatalf("capture generation: %v", err)
	}
	if got.ActivationEnvironment == "" || got.ActivationValue == "" {
		t.Fatalf("incomplete generation: %#v", got)
	}
	return got
}

func runAdapterFuzzChild(t *testing.T, candidate, request string) ([]byte, adapterFuzzChildResult) {
	t.Helper()
	dir := t.TempDir()
	requestPath := filepath.Join(dir, "request")
	candidatePath := filepath.Join(dir, "candidate")
	outputPath := filepath.Join(dir, "output")
	resultPath := filepath.Join(dir, "result.json")
	if err := os.WriteFile(requestPath, []byte(request), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidatePath, []byte(candidate), 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(os.Args[0], "-test.run=^TestRunFromProcessRejectsInvalidShellCompletionProcessFraming$")
	command.Env = append(os.Environ(),
		adapterFuzzChildMode+"=1",
		adapterFuzzRequestPath+"="+requestPath,
		adapterFuzzCandidatePath+"="+candidatePath,
		adapterFuzzOutputPath+"="+outputPath,
		adapterFuzzResultPath+"="+resultPath,
	)
	childOutput, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("adapter child: %v\n%s", err, childOutput)
	}
	wire, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	resultBytes, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	var result adapterFuzzChildResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("decode result: %v bytes=%q", err, resultBytes)
	}
	return wire, result
}

func parseAdapterFuzzRecords(t *testing.T, data []byte) [][]string {
	t.Helper()
	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Fatalf("records response missing LF: %q", data)
	}
	lines := strings.Split(string(data[:len(data)-1]), "\n")
	out := make([][]string, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		for i := 1; i < len(parts); i++ {
			parts[i] = unescapeAdapterFuzzField(t, parts[i])
		}
		out = append(out, parts)
	}
	return out
}

func unescapeAdapterFuzzField(t *testing.T, value string) string {
	t.Helper()
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] != '\\' {
			b.WriteByte(value[i])
			continue
		}
		if i+1 >= len(value) {
			t.Fatalf("bad escape %q", value)
		}
		i++
		switch value[i] {
		case '\\':
			b.WriteByte('\\')
		case 't':
			b.WriteByte('\t')
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 'x':
			if i+2 >= len(value) {
				t.Fatalf("short hex %q", value)
			}
			v, err := strconv.ParseUint(value[i+1:i+3], 16, 8)
			if err != nil {
				t.Fatalf("bad hex %q", value)
			}
			b.WriteByte(byte(v))
			i += 2
		default:
			t.Fatalf("unknown escape %q", value)
		}
	}
	return b.String()
}

func adapterFuzzRecordExists(records [][]string, want ...string) bool {
	return slices.ContainsFunc(records, func(record []string) bool { return slices.Equal(record, want) })
}

func recordsFuzzRequest(wordLines []string, cursorWord, cursorByte string) string {
	lines := []string{"records", "shell\tprobe\t1", "input\tuncertain\t0\t" + cursorWord + "\t" + cursorByte + "\tobserved-word-bytes"}
	lines = append(lines, wordLines...)
	lines = append(lines, "matching\tclient", "end", "")
	return strings.Join(lines, "\n")
}

func FuzzShellCompletionAdapterRecordsFieldRoundTrip(f *testing.F) {
	for _, seed := range []string{"plain", "tab\tvalue", "line\nvalue", "return\rvalue", `slash\\value`, "βash"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		request := strings.Join([]string{"records", "shell\tprobe\t1", "input\texact\t1\t1\t0\targument-bytes", "word\t0\t--value\texact", "word\t1\t\texact", "matching\tclient", "end", ""}, "\n")
		wire, result := runAdapterFuzzChild(t, value, request)
		if strings.IndexByte(value, 0) >= 0 {
			if result.DiagnosticCode == "" {
				t.Fatalf("Records ABI accepted NUL candidate; wire=%q", wire)
			}
			return
		}
		if result.DiagnosticCode != "" {
			t.Fatalf("Records ABI round trip failed: %s wire=%q", result.DiagnosticCode, wire)
		}
		records := parseAdapterFuzzRecords(t, wire)
		if !adapterFuzzRecordExists(records, "item", "0", value, "false", "default") {
			t.Fatalf("candidate did not round-trip: records=%#v", records)
		}
	})
}

func FuzzShellCompletionAdapterCanonicalInteger(f *testing.F) {
	for _, seed := range []string{"0", "1", "-1", "01", "+1", "-0", "15", "16", "9223372036854775807", "not-an-int"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		parsed, err := strconv.ParseInt(value, 10, 0)
		canonical := err == nil && value != "" && value != "-0" && !strings.HasPrefix(value, "+") && (len(value) <= 1 || value[0] != '0') && strconv.FormatInt(parsed, 10) == value
		inProbeRange := canonical && parsed >= 0 && parsed < 16
		index := 0
		if inProbeRange {
			index = int(parsed)
		}
		lines := make([]string, 0, index+1)
		for i := 0; i < index; i++ {
			lines = append(lines, "word\t"+strconv.Itoa(i)+"\tx\tlexical")
		}
		lines = append(lines, "word\t"+value+"\tx\tlexical")
		_, result := runAdapterFuzzChild(t, "x", recordsFuzzRequest(lines, strconv.Itoa(index), "0"))
		if inProbeRange && result.DiagnosticCode != "" {
			t.Fatalf("canonical in-range integer %q rejected: %s", value, result.DiagnosticCode)
		}
		if !inProbeRange && result.DiagnosticCode == "" {
			t.Fatalf("noncanonical/out-of-range integer %q accepted", value)
		}
	})
}

func FuzzShellCompletionAdapterRecordsRequireContiguousWordIndexes(f *testing.F) {
	for _, count := range []uint8{1, 2, 3, 8} {
		f.Add(count)
	}
	f.Fuzz(func(t *testing.T, rawCount uint8) {
		count := int(rawCount%16) + 1
		lines := make([]string, 0, count)
		for i := 0; i < count; i++ {
			lines = append(lines, "word\t"+strconv.Itoa(i)+"\tx\tlexical")
		}
		_, canonical := runAdapterFuzzChild(t, "x", recordsFuzzRequest(lines, "0", "0"))
		if canonical.DiagnosticCode != "" {
			t.Fatalf("contiguous indexes rejected: %s", canonical.DiagnosticCode)
		}
		if count == 1 {
			return
		}
		gap := append([]string(nil), lines...)
		gap[1] = "word\t" + strconv.Itoa(count+1) + "\tx\tlexical"
		_, bad := runAdapterFuzzChild(t, "x", recordsFuzzRequest(gap, "0", "0"))
		if bad.DiagnosticCode == "" {
			t.Fatal("non-contiguous word indexes accepted")
		}
	})
}

func FuzzShellCompletionAdapterJSONRejectsUnknownCoreField(f *testing.F) {
	for _, seed := range []string{"unknown", "future-core", "x", "shell_extra"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, field string) {
		if field == "" || field == "response_encoding" || field == "shell" || field == "input" || field == "matching" || field == "capabilities" || field == "native_capabilities" {
			t.Skip()
		}
		core := `"shell":{"family":"probe","version":"1"},"input":{"words":[{"value":""}],"cursor":{"word":0,"byte":0}}`
		request := "json\n{" + core + "," + strconv.Quote(field) + `:true}`
		_, result := runAdapterFuzzChild(t, "x", request)
		if result.DiagnosticCode == "" {
			t.Fatalf("unknown JSON core field %q accepted", field)
		}
	})
}

var _ = io.EOF

func runAdapterFuzzChildMode(t *testing.T) bool {
	if os.Getenv(adapterFuzzChildMode) != "1" {
		return false
	}
	requestBytes, err := os.ReadFile(os.Getenv(adapterFuzzRequestPath))
	if err != nil {
		t.Fatal(err)
	}
	candidateBytes, err := os.ReadFile(os.Getenv(adapterFuzzCandidatePath))
	if err != nil {
		t.Fatal(err)
	}
	stdin, err := os.Open(os.Getenv(adapterFuzzRequestPath))
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	output, err := os.Create(os.Getenv(adapterFuzzOutputPath))
	if err != nil {
		t.Fatal(err)
	}
	previousIn := os.Stdin
	previousOut := os.Stdout
	previousArgs := os.Args
	os.Stdin = stdin
	os.Stdout = output
	os.Args = []string{"tool"}
	defer func() {
		os.Stdin = previousIn
		os.Stdout = previousOut
		os.Args = previousArgs
	}()
	_ = requestBytes
	app := adapterFuzzApp(t, string(candidateBytes))
	generation := adapterFuzzCaptureGeneration(t, app)
	if err := os.Setenv(generation.ActivationEnvironment, generation.ActivationValue); err != nil {
		t.Fatal(err)
	}
	runErr := app.RunFromProcess(t.Context())
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	result := adapterFuzzChildResult{}
	if runErr != nil {
		report := cli.Diagnostics(runErr)
		if len(report.Diagnostics) > 0 {
			result.DiagnosticCode = string(report.Diagnostics[0].Code)
		} else {
			result.DiagnosticCode = "non-diagnostic-error"
		}
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv(adapterFuzzResultPath), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	return true
}
