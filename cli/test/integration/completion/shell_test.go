package completion

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/shelltest"
)

var (
	shellCompletionEnabled = flag.Bool("integration", false, "run real-shell completion integration tests")
	shellCompletionCase    = flag.String("case", "", "run one shell completion case as family/version")
)

var shellCompletionCases = [...]string{
	"bash/3.2.57",
	"bash/4.0.38",
	"bash/4.4.0",
	"bash/5.3.0",
	"elvish/0.11.0",
	"elvish/0.12.0",
	"fish/3.0.0",
	"fish/4.0.0",
	"nushell/0.72.0",
	"nushell/0.102.0",
	"nushell/0.108.0",
	"nushell/0.113.0",
	"nushell/0.115.1",
	"powershell/5.1",
	"powershell/7.6.6",
	"zsh/4.0.1",
	"zsh/5.9.2",
}

type shellObservation struct {
	version string
	screen  []string
}

type specimenConfig struct {
	specimen string
}

type completionMatrixRow struct {
	family     string
	version    string
	host       string
	executable string
}

func TestCompletionMatrix(t *testing.T) {
	t.Parallel()
	completionMatrix(t)
}

func TestShellCompletion(t *testing.T) {
	t.Parallel()
	if !*shellCompletionEnabled {
		t.Skip("real-shell completion tests require make test-shell-completion")
	}

	matrix := completionMatrix(t)
	if *shellCompletionCase != "" {
		row := completionMatrixCaseID(t, matrix, *shellCompletionCase)
		if row.host != runtime.GOOS {
			t.Fatalf("shell completion case %s requires %s; current host is %s", *shellCompletionCase, row.host, runtime.GOOS)
		}
	} else if !completionMatrixSupportsHost(matrix, runtime.GOOS) {
		t.Fatalf("shell completion compatibility has no proof cases for %s", runtime.GOOS)
	}

	t.Run("bash", func(t *testing.T) {
		t.Parallel()
		testBash(t, matrix)
	})
	t.Run("elvish", func(t *testing.T) {
		t.Parallel()
		testElvish(t, matrix)
	})
	t.Run("fish", func(t *testing.T) {
		t.Parallel()
		testFish(t, matrix)
	})
	t.Run("nushell", func(t *testing.T) {
		t.Parallel()
		testNushell(t, matrix)
	})
	t.Run("powershell", func(t *testing.T) {
		t.Parallel()
		testPowerShell(t, matrix)
	})
	t.Run("zsh", func(t *testing.T) {
		t.Parallel()
		testZsh(t, matrix)
	})
}

func completionRuntime(t *testing.T, matrix []completionMatrixRow, family, version string) string {
	t.Helper()
	caseID := family + "/" + version
	row := completionMatrixCase(t, matrix, family, version)
	if *shellCompletionCase != "" && *shellCompletionCase != caseID {
		t.SkipNow()
	}

	if row.host != runtime.GOOS {
		t.Skipf("%s requires %s", caseID, row.host)
	}

	executable := row.executable
	if strings.Contains(executable, "/") {
		runtimeRoot := os.Getenv("COMMANDMENT_SHELL_RUNTIME_ROOT")
		if runtimeRoot == "" {
			var err error
			runtimeRoot, err = filepath.Abs(filepath.Join("..", "..", "..", ".."))
			if err != nil {
				t.Fatalf("resolve repository root: %v", err)
			}
		}
		executable = filepath.Join(runtimeRoot, filepath.FromSlash(executable))
	}
	runtimePath, err := exec.LookPath(executable)
	if err != nil {
		t.Fatalf("shell completion runtime for %s is unavailable: %s; run make prepare-shell-runtimes", caseID, row.executable)
	}
	return runtimePath
}

func completionMatrixCase(t *testing.T, matrix []completionMatrixRow, family, version string) completionMatrixRow {
	t.Helper()
	for _, row := range matrix {
		if row.family == family && row.version == version {
			return row
		}
	}
	t.Fatalf("shell completion matrix has no case %s/%s", family, version)
	return completionMatrixRow{}
}

func completionMatrixCaseID(t *testing.T, matrix []completionMatrixRow, caseID string) completionMatrixRow {
	t.Helper()
	for _, row := range matrix {
		if row.family+"/"+row.version == caseID {
			return row
		}
	}
	t.Fatalf("unknown shell completion case %q", caseID)
	return completionMatrixRow{}
}

func completionMatrixSupportsHost(matrix []completionMatrixRow, host string) bool {
	for _, row := range matrix {
		if row.host == host {
			return true
		}
	}
	return false
}

func completionMatrix(t *testing.T) []completionMatrixRow {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "scripts", "shells", "completion-matrix.tsv")
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open shell completion matrix: %v", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			t.Errorf("close shell completion matrix: %v", closeErr)
		}
	}()

	required := make(map[string]bool, len(shellCompletionCases))
	for _, caseID := range shellCompletionCases {
		required[caseID] = true
	}

	var rows []completionMatrixRow
	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 4 {
			t.Fatalf("shell completion matrix line %d has %d fields, want 4", lineNumber, len(fields))
		}
		row := completionMatrixRow{
			family:     fields[0],
			version:    fields[1],
			host:       fields[2],
			executable: fields[3],
		}
		caseID := row.family + "/" + row.version
		needed, known := required[caseID]
		if !known {
			t.Fatalf("shell completion matrix contains unknown case %q", caseID)
		}
		if !needed {
			t.Fatalf("shell completion matrix contains duplicate case %q", caseID)
		}
		if row.host != "linux" && row.host != "windows" {
			t.Fatalf("shell completion case %q has unknown host %q", caseID, row.host)
		}
		if row.executable == "" {
			t.Fatalf("shell completion case %q has no executable", caseID)
		}
		required[caseID] = false
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read shell completion matrix: %v", err)
	}
	var missing []string
	for _, caseID := range shellCompletionCases {
		if required[caseID] {
			missing = append(missing, caseID)
		}
	}
	if len(missing) != 0 {
		t.Fatalf("shell completion matrix is missing cases: %v", missing)
	}
	return rows
}

func compileShellCommandApplication(t *testing.T) cli.Application {
	t.Helper()
	app, err := cli.Compile(cli.Definition[struct{}]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "analyze",
				Description: cli.Description{
					Short: "analyze recovered evidence",
				},
			})
			root.Subcommands.Command(cli.Command[struct{}]{
				Name: "abduct",
				Description: cli.Description{
					Short: "abduct a terrestrial specimen",
				},
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func compileShellValueApplication(t *testing.T, complete cli.ValueCompleter[specimenConfig]) cli.Application {
	t.Helper()
	app, err := cli.Compile(cli.Definition[specimenConfig]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[specimenConfig]) {
			root.Arguments.String(&root.Config.specimen, "specimen").CompleteBy(complete)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func completeNoCandidates(_ context.Context, _ *specimenConfig, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	return cli.CompletionContribution{}, nil
}

func interpretShellObservation(observation shelltest.Observation) (shellObservation, error) {
	version := observation.Version()
	if version == "" {
		return shellObservation{}, errors.New("missing verified runtime version")
	}
	screen := observation.ScreenLines()
	if screen == nil {
		return shellObservation{}, errors.New("settled shell screen unavailable")
	}
	result := shellObservation{
		version: version,
		screen:  screen,
	}
	return result, nil
}

func requireRuntimeVersion(t *testing.T, observation shellObservation, expected string) {
	t.Helper()
	if observation.version != expected {
		t.Fatalf("runtime version = %q, want %q", observation.version, expected)
	}
}

func requireScreenContains(t *testing.T, observation shellObservation, fragments ...string) {
	t.Helper()
	screen := strings.Join(observation.screen, "\n")
	for _, fragment := range fragments {
		if !strings.Contains(screen, fragment) {
			t.Fatalf("settled shell screen does not contain %q: %q", fragment, screen)
		}
	}
}

func requireScreenOrder(t *testing.T, observation shellObservation, first, second string) {
	t.Helper()
	screen := strings.Join(observation.screen, "\n")
	firstIndex := strings.Index(screen, first)
	secondIndex := strings.Index(screen, second)
	if firstIndex < 0 || secondIndex < 0 {
		t.Fatalf("settled shell screen lacks ordered fragments %q then %q: %q", first, second, screen)
	}
	if firstIndex >= secondIndex {
		t.Fatalf("settled shell screen order = %q before %q, want %q before %q: %q", second, first, first, second, screen)
	}
}

func requireScreenOccurrences(t *testing.T, observation shellObservation, fragment string, minimum int) {
	t.Helper()
	screen := strings.Join(observation.screen, "\n")
	if count := strings.Count(screen, fragment); count < minimum {
		t.Fatalf("settled shell screen contains %q %d times, want at least %d: %q", fragment, count, minimum, screen)
	}
}

func requireScreenExcludes(t *testing.T, observation shellObservation, fragments ...string) {
	t.Helper()
	screen := strings.Join(observation.screen, "\n")
	for _, fragment := range fragments {
		if strings.Contains(screen, fragment) {
			t.Fatalf("settled shell screen unexpectedly contains %q: %q", fragment, screen)
		}
	}
}
