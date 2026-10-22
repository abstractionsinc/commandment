package shelltest

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
)

func TestSetupDefersMaterializationAndSnapshotsDeclarations(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	generated := generatedCompletion{
		integration: []byte("generated"),
	}
	verified := verifiedRuntime{
		executable: "/runtime",
		version:    "1.2.3",
	}
	var integration, fixture string
	args := []string{"--interactive"}
	plan, err := runSetup(func(setup Setup) error {
		if got := setup.Executable(); got != "/runtime" {
			t.Fatalf("Executable() = %q", got)
		}
		if got := setup.Version(); got != "1.2.3" {
			t.Fatalf("Version() = %q", got)
		}
		integration = setup.IntegrationFile("completion/integration.sh")
		fixture = setup.File("config/rc", "fixture")
		setup.Env("SHELLTEST_VALUE", "one")
		setup.Env("SHELLTEST_VALUE", "one")
		setup.Args(args...)
		args[0] = "changed"
		for _, path := range []string{integration, fixture} {
			if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("setup file materialized during callback: %q, err=%v", path, err)
			}
		}
		return nil
	}, workspace, verified, generated)
	if err != nil {
		t.Fatal(err)
	}
	if err := materializeSetup(workspace, plan); err != nil {
		t.Fatal(err)
	}
	// #nosec G304 -- integration is an operation-local path returned by the Setup capability.
	if got, err := os.ReadFile(integration); err != nil || string(got) != "generated" {
		t.Fatalf("integration materialization = %q, %v", got, err)
	}
	// #nosec G304 -- fixture is an operation-local path returned by the Setup capability.
	if got, err := os.ReadFile(fixture); err != nil || string(got) != "fixture" {
		t.Fatalf("fixture materialization = %q, %v", got, err)
	}
	if got := plan.args; !slices.Equal(got, []string{"--interactive"}) {
		t.Fatalf("Args snapshot = %#v", got)
	}
}

func TestSetupRejectsCollisionsAndConflictingEnvironment(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(Setup)
		want string
	}{
		{
			name: "duplicate-file",
			run:  func(s Setup) { s.File("a", "1"); s.File("a", "2") },
			want: "collision",
		},
		{
			name: "parent-file",
			run:  func(s Setup) { s.File("a", "1"); s.File(filepath.Join("a", "b"), "2") },
			want: "collision",
		},
		{
			name: "escape",
			run:  func(s Setup) { s.File(filepath.Join("..", "a"), "1") },
			want: "invalid setup file",
		},
		{
			name: "environment",
			run:  func(s Setup) { s.Env("A", "one"); s.Env("A", "two") },
			want: "conflicting environment",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := runSetup(func(setup Setup) error { test.run(setup); return nil }, t.TempDir(), verifiedRuntime{}, generatedCompletion{})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("setup error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestSetupReservedPathsRejectEqualityAndContainment(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	generated := generatedCompletion{
		reserved: []string{filepath.Join("bin", "alienctl"), filepath.Join("bin", "alienctl") + bridgeBootstrapSuffix},
	}
	for _, name := range []string{filepath.Join("bin", "alienctl"), "bin", filepath.Join("bin", "alienctl", "child")} {
		_, err := runSetup(func(setup Setup) error { setup.File(name, "x"); return nil }, workspace, verifiedRuntime{}, generated)
		if err == nil || !strings.Contains(err.Error(), "framework path") {
			t.Fatalf("reserved collision %q error = %v", name, err)
		}
	}
}

func TestSetupAndObservationZeroAndExpiredValuesPanic(t *testing.T) {
	t.Parallel()
	requirePanic(t, func() { Setup{}.Executable() })
	requirePanic(t, func() { Observation{}.Version() })
	session := &setupSession{
		state: setupSessionClosed,
	}
	requirePanic(t, func() {
		(Setup{
			session: session,
		}).Version()
	})
	state := &observationState{
		active: false,
	}
	requirePanic(t, func() {
		(Observation{
			state: state,
		}).ScreenLines()
	})
}

func TestObservationDefensiveScreenSnapshotAndEditorInputUnavailable(t *testing.T) {
	t.Parallel()
	state := &observationState{
		active:      true,
		version:     "1",
		screenLines: []string{},
	}
	observation := Observation{
		state: state,
	}
	if lines := observation.ScreenLines(); lines == nil || len(lines) != 0 {
		t.Fatalf("nonnil-empty screen = %#v", lines)
	}
	state.screenLines = []string{"one"}
	lines := observation.ScreenLines()
	lines[0] = "changed"
	if got := observation.ScreenLines()[0]; got != "one" {
		t.Fatalf("ScreenLines aliased state: %q", got)
	}
	if input, ok := observation.EditorInput(); ok || input != (Input{}) {
		t.Fatalf("unavailable editor input = %#v, %t", input, ok)
	}
}

func TestSetupFileNamePlatformContainment(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"", ".", "..", string([]byte{'a', 0, 'b'})} {
		if _, err := validateSetupFileName(name); err == nil {
			t.Fatalf("validateSetupFileName(%q) succeeded", name)
		}
	}
	if runtime.GOOS != windowsOS {
		if got, err := validateSetupFileName("nested/file"); err != nil || got != filepath.Clean("nested/file") {
			t.Fatalf("valid nested name = %q, %v", got, err)
		}
	}
}

func TestSetupEnvironmentCaseSemantics(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == windowsOS {
		return
	}
	plan, err := runSetup(func(setup Setup) error { setup.Env("A", "one"); setup.Env("a", "two"); return nil }, t.TempDir(), verifiedRuntime{}, generatedCompletion{})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.environment) != 2 {
		t.Fatalf("environment declarations = %#v", plan.environment)
	}
}

func requirePanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
}

func TestSetupConcurrentAndCrossBoundaryUseIsRejected(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		session := &setupSession{
			state: setupSessionActive,
		}
		setup := Setup{
			session: session,
		}
		held := setup.acquire()
		requirePanic(t, func() { setup.Version() })

		closed := make(chan struct{})
		go func() {
			session.close()
			close(closed)
		}()
		synctest.Wait()
		session.mu.Lock()
		state := session.state
		session.mu.Unlock()
		if state != setupSessionClosed {
			t.Fatalf("Setup session state = %v, want closed", state)
		}
		requirePanic(t, func() { setup.Executable() })

		setup.release(held)
		<-closed
		if session.failure == nil || !strings.Contains(session.failure.Error(), "crossed Prepare callback boundary") {
			t.Fatalf("cross-boundary Setup failure = %v", session.failure)
		}
	})
}

func TestWorkspaceMaterializationRejectsPreexistingAndSymlinkTargets(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == windowsOS {
		t.Skip("Windows reparse-point behavior is target-platform verified")
	}
	workspace := t.TempDir()
	if err := prepareOperationWorkspace(workspace); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "existing"), []byte("ambient"), privateFileMode); err != nil {
		t.Fatal(err)
	}
	if err := createWorkspaceFile(workspace, "existing", []byte("replacement"), privateFileMode); err == nil {
		t.Fatal("preexisting final file was overwritten")
	}

	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(workspace, "alias")); err != nil {
		t.Fatal(err)
	}
	if err := createWorkspaceFile(workspace, filepath.Join("alias", "escape"), []byte("x"), privateFileMode); err == nil {
		t.Fatal("symlink parent was followed")
	}
	if _, err := os.Stat(filepath.Join(outside, "escape")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("workspace write escaped through symlink: %v", err)
	}

	target := filepath.Join(outside, "target")
	if err := os.Symlink(target, filepath.Join(workspace, "final-link")); err != nil {
		t.Fatal(err)
	}
	if err := createWorkspaceFile(workspace, "final-link", []byte("x"), privateFileMode); err == nil {
		t.Fatal("symlink final target was overwritten")
	}
	if _, err := os.Stat(target); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("final symlink target was created: %v", err)
	}
}
