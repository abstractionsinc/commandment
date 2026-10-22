package conformance

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

func TestExitCodePreservesErrorsAndSelectsApplicationPolicy(t *testing.T) {
	t.Parallel()
	cause := errors.New("application failure")
	wrapped := cli.WithExitCode(cause, 23)
	if wrapped == cause { //nolint:errorlint // The test proves that withExitCode creates a distinct wrapper.
		t.Fatal("WithExitCode did not retain explicit policy wrapper")
	}
	if !errors.Is(wrapped, cause) {
		t.Fatal("WithExitCode did not preserve application error identity")
	}
	if got := cli.ExitCode(wrapped); got != 23 {
		t.Fatalf("ExitCode(WithExitCode(...)) = %d, want 23", got)
	}
	if got := cli.ExitCode(cause); got != 1 {
		t.Fatalf("ExitCode(ordinary failure) = %d, want %d", got, 1)
	}
	if got := cli.ExitCode(nil); got != 0 {
		t.Fatalf("ExitCode(nil) = %d, want 0", got)
	}
	if got := cli.WithExitCode(nil, 17); got != nil {
		t.Fatalf("WithExitCode(nil, ...) = %v, want nil", got)
	}
}

type coincidentalExitCodeError struct {
	err  error
	code int
}

func (err coincidentalExitCodeError) Error() string { return err.err.Error() }

func (err coincidentalExitCodeError) Unwrap() error { return err.err }

func (err coincidentalExitCodeError) ExitCode() int { return err.code }

func TestExitCodeRequiresExplicitApplicationPolicy(t *testing.T) {
	t.Parallel()
	cause := errors.New("application failure")
	ordinary := coincidentalExitCodeError{
		err:  cause,
		code: 91,
	}

	if got := cli.ExitCode(ordinary); got != 1 {
		t.Fatalf("ExitCode(error with coincidental ExitCode method) = %d, want 1", got)
	}
	explicit := cli.WithExitCode(cause, 23)
	outer := coincidentalExitCodeError{
		err:  explicit,
		code: 91,
	}
	if got := cli.ExitCode(outer); got != 23 {
		t.Fatalf("ExitCode(error wrapping explicit policy) = %d, want 23", got)
	}
}

func TestExitCodeRespectsCommandmentIntegrationBarriers(t *testing.T) {
	t.Parallel()

	cause := errors.New("nested application failure")
	application := cli.WithExitCode(cause, 41)
	definition := cli.Definition[struct{}]{
		Name: "integration-barrier",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Advisory.Always(cli.Advisory{
				Code:     "notice",
				Severity: cli.NoticeSeverityWarning,
				Message:  "notice",
			})
		},
		Run: func(context.Context, *struct{}) error { return nil },
	}
	app := clitest.Compile(t, definition)
	notice := app.Run(t.Context(), cli.RunRequest{
		Notify: func(context.Context, cli.Notice) error {
			return application
		},
	})
	if !errors.Is(notice, cause) {
		t.Fatal("notice integration failure did not preserve nested cause")
	}
	if got := cli.ExitCode(notice); got != 1 {
		t.Fatalf("ExitCode(notice integration failure) = %d, want %d", got, 1)
	}
}

func TestExitCodeUsesCanonicalAggregateOrderWithoutDiscardingCauses(t *testing.T) {
	t.Parallel()
	firstCause := errors.New("first")
	secondCause := errors.New("second")
	first := cli.WithExitCode(firstCause, 31)
	second := cli.WithExitCode(secondCause, 47)
	aggregate := exitStatusTestOrderedErrors{first, second}

	if got := cli.ExitCode(aggregate); got != 31 {
		t.Fatalf("ExitCode(aggregate) = %d, want 31", got)
	}
	if !errors.Is(aggregate, firstCause) || !errors.Is(aggregate, secondCause) {
		t.Fatal("aggregate lost a preserved application cause")
	}
}

type exitStatusTestOrderedErrors []error

func (errs exitStatusTestOrderedErrors) Error() string { return "ordered failures" }

func (errs exitStatusTestOrderedErrors) Unwrap() []error { return append([]error(nil), errs...) }

type cyclicExitStatusError struct{}

func (*cyclicExitStatusError) Error() string { return "cyclic exit-status failure" }

func (err *cyclicExitStatusError) Unwrap() error { return err }

func TestExitCodeTerminatesOnCyclicErrorGraphs(t *testing.T) {
	t.Parallel()
	cycle := &cyclicExitStatusError{}
	if got := cli.ExitCode(cycle); got != 1 {
		t.Fatalf("ExitCode(cycle) = %d, want 1", got)
	}
	if got := cli.ExitCode(exitStatusTestOrderedErrors{cycle, cli.WithExitCode(errors.New("later"), 47)}); got != 1 {
		t.Fatalf("ExitCode(cycle before coded branch) = %d, want 1", got)
	}
	if got := cli.ExitCode(exitStatusTestOrderedErrors{cli.WithExitCode(errors.New("first"), 31), cycle}); got != 31 {
		t.Fatalf("ExitCode(coded branch before cycle) = %d, want 31", got)
	}
}

func TestAbortErrorErrorPreservesExitStatusSemantics(t *testing.T) {
	t.Parallel()
	source, control := clitest.SignalSource(clitest.SignalSourceOptions[string]{
		Signals: []cli.SignalDescriptor[string]{
			{
				Signal:   exitStatusSignalNativeInterrupt,
				Fidelity: cli.SignalSourceDeliveryFidelityExact,
			},
		},
	})
	definition := cli.Definition[struct{}]{
		Name: "abort-error",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			signal := root.Signals.Source(source).Signal(exitStatusSignalNativeInterrupt)
			root.Signals.Policy(
				root.Signals.Match.On(signal).During(cli.SignalPhaseExecution).Each(),
				root.Signals.Action.Abort(7),
			)
		},
		Run: func(ctx context.Context, _ *struct{}) error { <-ctx.Done(); return ctx.Err() },
	}
	app := clitest.Compile(t, definition)
	done := make(chan error, 1)
	go func() {
		done <- app.Run(t.Context(), cli.RunRequest{
			Signals: cli.SignalHost{
				Abort: func(int) error { return nil },
			},
		})
	}()
	select {
	case <-control.Opened():
	case <-time.After(2 * time.Second):
		t.Fatal("signal source did not open")
	}
	_ = control.Notify(cli.SignalNotification[string]{
		Signal: exitStatusSignalNativeInterrupt,
	})
	select {
	case err := <-done:
		if err == nil || err.Error() != "execution aborted with exit code 7" {
			t.Fatalf("error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not abort")
	}
}

func TestAbortErrorExitCodePreservesExitStatusSemantics(t *testing.T) {
	t.Parallel()
	source, control := clitest.SignalSource(clitest.SignalSourceOptions[string]{
		Signals: []cli.SignalDescriptor[string]{
			{
				Signal:   exitStatusSignalNativeInterrupt,
				Fidelity: cli.SignalSourceDeliveryFidelityExact,
			},
		},
	})
	definition := cli.Definition[struct{}]{
		Name: "abort-code",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			signal := root.Signals.Source(source).Signal(exitStatusSignalNativeInterrupt)
			root.Signals.Policy(
				root.Signals.Match.On(signal).During(cli.SignalPhaseExecution).Each(),
				root.Signals.Action.Abort(13),
			)
		},
		Run: func(ctx context.Context, _ *struct{}) error { <-ctx.Done(); return ctx.Err() },
	}
	app := clitest.Compile(t, definition)
	done := make(chan error, 1)
	go func() {
		done <- app.Run(t.Context(), cli.RunRequest{
			Signals: cli.SignalHost{
				Abort: func(int) error { return nil },
			},
		})
	}()
	select {
	case <-control.Opened():
	case <-time.After(2 * time.Second):
		t.Fatal("signal source did not open")
	}
	_ = control.Notify(cli.SignalNotification[string]{
		Signal: exitStatusSignalNativeInterrupt,
	})
	var err error
	select {
	case err = <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not abort")
	}
	if cli.ExitCode(err) != 13 {
		t.Fatalf("ExitCode = %d err=%v", cli.ExitCode(err), err)
	}
}

func TestExitCodeErrorErrorPreservesExitStatusSemantics(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("sentinel")
	if got := cli.WithExitCode(sentinel, 4).Error(); got != "sentinel" {
		t.Fatalf("Error = %q", got)
	}
}

func TestExitCodeErrorUnwrapPreservesExitStatusSemantics(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("sentinel")
	if !errors.Is(cli.WithExitCode(sentinel, 4), sentinel) {
		t.Fatal("cause not preserved")
	}
}

func TestExitCodeErrorExitCodePreservesExitStatusSemantics(t *testing.T) {
	t.Parallel()
	if got := cli.ExitCode(cli.WithExitCode(errors.New("x"), 29)); got != 29 {
		t.Fatalf("ExitCode = %d", got)
	}
}

func TestFindExitCodePreservesExitStatusSemantics(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("outer: %w", cli.WithExitCode(errors.New("inner"), 31))
	if got := cli.ExitCode(err); got != 31 {
		t.Fatalf("ExitCode = %d", got)
	}
	joined := errors.Join(errors.New("plain"), cli.WithExitCode(errors.New("coded"), 37))
	if got := cli.ExitCode(joined); got != 37 {
		t.Fatalf("joined ExitCode = %d", got)
	}
}

func TestAbortErrorTextPreservesExitStatusSemantics(t *testing.T) {
	t.Parallel()
	err := cli.WithExitCode(errors.New("application"), 17)
	if cli.ExitCode(err) != 17 {
		t.Fatalf("ExitCode = %d", cli.ExitCode(err))
	}
}

func TestSignalCancellationErrorTextPreservesExitStatusSemantics(t *testing.T) {
	t.Parallel()
	var err *cli.SignalCancellation
	if err.Error() != "signal canceled primary execution" {
		t.Fatalf("text = %q", err.Error())
	}
}

func TestSignalTerminationErrorTextPreservesExitStatusSemantics(t *testing.T) {
	t.Parallel()
	var nilErr *cli.SignalTermination
	if nilErr.Error() != "execution terminated by signal" {
		t.Fatalf("nil text = %q", nilErr.Error())
	}
	err := cli.TerminatedBy(cli.Signal{})
	if err.Error() != "execution terminated by signal" {
		t.Fatalf("text = %q", err.Error())
	}
}

const (
	exitStatusSignalNativeInterrupt        = "interrupt"
	exitStatusSignalNativeMutated          = "mutated"
	exitStatusSignalNativeDynamic          = "dynamic"
	exitStatusSignalNativeTerminate        = "terminate"
	exitStatusSignalNativeFirst            = "first"
	exitStatusSignalAlienctlName           = "alienctl"
	exitStatusSignalRunFromProcessChildEnv = "COMMANDMENT_SIGNAL_RUN_FROM_PROCESS_CHILD"
)
