package shelltest

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/abstractionsinc/commandment/cli"
)

const (
	completionOperationTimeout = 60 * time.Second
	bridgeCleanupTimeout       = 5 * time.Second
)

// Complete proves default production completion generation through the real target shell/editor and
// returns the provider-owned semantic result. Infrastructure, runtime, preparation, bridge, causal,
// settlement, and interpretation failures are reported through t; only success returns R.
func Complete[R any](t testing.TB, app cli.Application, target Target[R], input Input) R {
	t.Helper()
	target, err := snapshotTarget(target, input)
	if err != nil {
		t.Fatalf("shelltest: invalid operation declaration: %v", err)
	}
	workspace := t.TempDir()
	if err := prepareOperationWorkspace(workspace); err != nil {
		t.Fatalf("shelltest: operation workspace: %v", err)
	}
	verified, err := verifyRuntime(t.Context(), workspace, target.Runtime)
	if err != nil {
		t.Fatalf("shelltest: runtime verification: %v", err)
	}
	generated, err := generateCompletion(app, target.Provider, workspace)
	if err != nil {
		t.Fatalf("shelltest: %v", err)
	}
	return runGeneratedCompletion(t, app, target, input, workspace, verified, generated)
}

// CompleteForGenerationTarget proves targeted production completion generation through the real
// target shell/editor and returns the provider-owned semantic result. generationTarget must be
// nonempty and is generation compatibility input rather than runtime-version truth. Infrastructure,
// runtime, preparation, bridge, causal, settlement, and interpretation failures are reported
// through t; only success returns R.
func CompleteForGenerationTarget[R any](t testing.TB, app cli.Application, target Target[R], generationTarget string, input Input) R {
	t.Helper()
	if generationTarget == "" {
		t.Fatalf("shelltest: invalid operation declaration: generation target is required")
	}
	target, err := snapshotTarget(target, input)
	if err != nil {
		t.Fatalf("shelltest: invalid operation declaration: %v", err)
	}
	workspace := t.TempDir()
	if err := prepareOperationWorkspace(workspace); err != nil {
		t.Fatalf("shelltest: operation workspace: %v", err)
	}
	verified, err := verifyRuntime(t.Context(), workspace, target.Runtime)
	if err != nil {
		t.Fatalf("shelltest: runtime verification: %v", err)
	}
	generated, err := generateCompletionForVersion(app, target.Provider, generationTarget, workspace)
	if err != nil {
		t.Fatalf("shelltest: %v", err)
	}
	return runGeneratedCompletion(t, app, target, input, workspace, verified, generated)
}

func runGeneratedCompletion[R any](t testing.TB, app cli.Application, target Target[R], input Input, workspace string, verified verifiedRuntime, generated generatedCompletion) R {
	t.Helper()
	plan, err := runSetup(target.Prepare, workspace, verified, generated)
	if err != nil {
		t.Fatalf("shelltest: invalid setup declaration: %v", err)
	}
	if err := materializeSetup(workspace, plan); err != nil {
		t.Fatalf("shelltest: materialize setup: %v", err)
	}

	operationContext, cancelOperation := context.WithTimeout(t.Context(), completionOperationTimeout)
	bridge, err := startApplicationBridge(operationContext, app, generated.generation)
	if err != nil {
		cancelOperation()
		t.Fatalf("shelltest: application bridge: %v", err)
	}

	helperPath, err := materializeBridgeHelper(workspace, generated, bridge)
	if err != nil {
		cancelOperation()
		cleanupBridgeAfterFailure(t, bridge)
		t.Fatalf("shelltest: application helper: %v", err)
	}
	environment, err := shellEnvironment(workspace, filepath.Dir(helperPath), plan)
	if err != nil {
		cancelOperation()
		cleanupBridgeAfterFailure(t, bridge)
		t.Fatalf("shelltest: prepare shell environment: %v", err)
	}
	if err := verifyRuntimeExecutable(verified); err != nil {
		cancelOperation()
		cleanupBridgeAfterFailure(t, bridge)
		t.Fatalf("shelltest: runtime verification: %v", err)
	}

	terminal, err := startTerminalSession(operationContext, verified.executable, plan.args, workspace, environment)
	if err != nil {
		cancelOperation()
		cleanupBridgeAfterFailure(t, bridge)
		t.Fatalf("shelltest: real shell editor: %v", err)
	}

	screenLines, operationErr := settleGeneratedCompletion(operationContext, bridge, terminal, input)
	cancelOperation()
	terminalCleanupErr := cleanupTerminal(t.Context(), terminal)
	bridgeCleanupErr := cleanupBridge(t.Context(), bridge)

	if recovered, panicked := bridge.panic(); panicked {
		if terminalCleanupErr != nil {
			t.Errorf("shelltest: terminal cleanup after application panic: %v", terminalCleanupErr)
		}
		if bridgeCleanupErr != nil {
			t.Errorf("shelltest: bridge cleanup after application panic: %v", bridgeCleanupErr)
		}
		panic(recovered)
	}
	if operationErr != nil {
		if terminalCleanupErr != nil {
			t.Errorf("shelltest: terminal cleanup after operation failure: %v", terminalCleanupErr)
		}
		if bridgeCleanupErr != nil {
			t.Errorf("shelltest: bridge cleanup after operation failure: %v", bridgeCleanupErr)
		}
		t.Fatalf("shelltest: %v", operationErr)
	}
	if terminalCleanupErr != nil {
		if bridgeCleanupErr != nil {
			t.Errorf("shelltest: bridge cleanup after terminal cleanup failure: %v", bridgeCleanupErr)
		}
		t.Fatalf("shelltest: terminal cleanup: %v", terminalCleanupErr)
	}
	if bridgeCleanupErr != nil {
		t.Fatalf("shelltest: bridge cleanup: %v", bridgeCleanupErr)
	}

	state := &observationState{
		active:      true,
		version:     verified.version,
		screenLines: screenLines,
	}
	var result R
	var interpretErr error
	func() {
		defer func() {
			if crossed := state.expire(); crossed {
				panic("shelltest: Observation use crossed Interpret callback boundary")
			}
		}()
		result, interpretErr = target.Interpret(Observation{state: state})
	}()
	if interpretErr != nil {
		t.Fatalf("shelltest: interpret settled observation: %v", interpretErr)
	}
	return result
}

func settleGeneratedCompletion(ctx context.Context, bridge *applicationBridge, terminal *terminalSession, input Input) ([]string, error) {
	if _, err := terminal.waitQuiet(ctx, "wait for shell startup quiet/alive"); err != nil {
		return nil, err
	}
	if err := terminal.enter(input); err != nil {
		return nil, err
	}
	if _, err := terminal.waitQuiet(ctx, "wait for shell input quiet/alive"); err != nil {
		return nil, err
	}
	if err := bridge.arm(); err != nil {
		return nil, err
	}
	if err := terminal.complete(); err != nil {
		return nil, err
	}
	if err := bridge.waitWitness(ctx); err != nil {
		return nil, err
	}
	if err := bridge.waitConnectionsEmpty(ctx); err != nil {
		return nil, err
	}
	frontier := bridge.authenticatedFrontier()
	if _, err := terminal.waitQuiet(ctx, "wait for pre-seal terminal quiet/alive"); err != nil {
		return nil, err
	}
	if err := bridge.beginSealing(); err != nil {
		return nil, err
	}
	if err := bridge.waitAcceptDrained(ctx); err != nil {
		return nil, err
	}
	if err := bridge.waitConnectionsEmpty(ctx); err != nil {
		return nil, err
	}
	if err := bridge.settle(frontier); err != nil {
		return nil, err
	}
	return terminal.finalScreen(ctx)
}

func cleanupTerminal(parent context.Context, terminal *terminalSession) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), terminalCleanupTimeout)
	defer cancel()
	return terminal.cleanup(ctx)
}

func cleanupBridge(parent context.Context, bridge *applicationBridge) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), bridgeCleanupTimeout)
	defer cancel()
	return bridge.cleanup(ctx)
}

func cleanupBridgeAfterFailure(t testing.TB, bridge *applicationBridge) {
	t.Helper()
	if err := cleanupBridge(t.Context(), bridge); err != nil {
		t.Errorf("shelltest: bridge cleanup after setup failure: %v", err)
	}
}

func snapshotTarget[R any](target Target[R], input Input) (Target[R], error) {
	if target.Provider == nil {
		return Target[R]{}, failureError{subject: "target Provider is required"}
	}
	if target.Runtime.Executable == "" {
		return Target[R]{}, failureError{subject: "runtime Executable is required"}
	}
	if len(target.Runtime.VersionArgs) == 0 {
		return Target[R]{}, failureError{subject: "runtime VersionArgs is required"}
	}
	if target.Runtime.ParseVersion == nil {
		return Target[R]{}, failureError{subject: "runtime ParseVersion is required"}
	}
	if target.Interpret == nil {
		return Target[R]{}, failureError{subject: "target Interpret is required"}
	}
	if !utf8.ValidString(input.BeforeCursor) || !utf8.ValidString(input.AfterCursor) {
		return Target[R]{}, failureError{subject: "editor input must be valid UTF-8 on both sides of the cursor"}
	}
	target.Runtime.VersionArgs = slices.Clone(target.Runtime.VersionArgs)
	return target, nil
}
