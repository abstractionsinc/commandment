package cli

import (
	"fmt"
	"slices"
)

// WithExitCode wraps a non-nil error with an explicit application exit status while
// preserving the original error for errors.Is and errors.As. A nil error remains nil.
// Application errors do not opt into exit policy through their own method sets.
func WithExitCode(err error, code int) error {
	return withExitCode(err, code)
}

// ExitCode selects the process status represented by err. Nil maps to 0; terminal
// Abort policy or application policy attached by WithExitCode supplies its code;
// otherwise a returned failure maps to 1. The host retains authority over
// whether to exit the process.
func ExitCode(err error) int {
	return exitCode(err)
}

type abortError struct {
	code int
}

// Error reports the terminal abort outcome.
func (err abortError) Error() string {
	return abortErrorText(err)
}

const genericFailureExitCode = 1

type exitCodeError struct {
	err  error
	code int
}

type integrationExitPolicy uint8

const (
	integrationExitPolicyBarrier integrationExitPolicy = iota
	integrationExitPolicyGenericFailure
)

// integrationFailureError is a private cause-preserving presentation and exit-policy boundary.
// Structured Commandment identity belongs to Diagnostics; this carrier exists only to keep the
// original integration cause reachable without publishing a duplicate machine identity.
type integrationFailureError struct {
	message string
	err     error
	policy  integrationExitPolicy
}

func (err *integrationFailureError) Error() string {
	if err == nil {
		return "commandment integration failed"
	}
	return err.message
}

func (err *integrationFailureError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.err
}

// Error delegates presentation to the wrapped application error.
func (err *exitCodeError) Error() string {
	if err == nil || err.err == nil {
		return ""
	}
	return err.err.Error()
}

// Unwrap returns the application error whose cause identity is preserved.
func (err *exitCodeError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.err
}

func withExitCode(err error, code int) error {
	if err == nil {
		return nil
	}
	return &exitCodeError{
		err:  err,
		code: code,
	}
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if code, ok := findExitCode(err); ok {
		return code
	}
	return genericFailureExitCode
}

func findExitCode(err error) (int, bool) {
	if err == nil {
		return 0, false
	}
	const maximumErrorGraphNodes = 4096
	stack := []error{err}
	visited := 0
	for len(stack) != 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current == nil {
			continue
		}
		visited++
		if visited > maximumErrorGraphNodes {
			return 0, false
		}

		// Commandment-owned integration failures are exit-policy barriers. Their
		// nested causes remain available through errors.Is/errors.As, but application
		// exit policy inside such a cause cannot seize invocation policy.
		if integration, ok := current.(*integrationFailureError); ok { //nolint:errorlint // Exit-policy barriers intentionally inspect only the current outer error.
			if integration.policy == integrationExitPolicyGenericFailure {
				return genericFailureExitCode, true
			}
			continue
		}
		switch current := current.(type) { //nolint:errorlint // Traversal owns descent and intentionally classifies only this exact node.
		case abortError:
			return current.code, true
		case *exitCodeError:
			return current.code, true
		}
		if many, ok := current.(interface{ Unwrap() []error }); ok {
			for _, child := range slices.Backward(many.Unwrap()) {
				stack = append(stack, child)
			}
			continue
		}
		if one, ok := current.(interface{ Unwrap() error }); ok {
			stack = append(stack, one.Unwrap())
		}
	}
	return 0, false
}

func noticeDeliveryErrorText(index int) string {
	return fmt.Sprintf("notice delivery failed at index %d", index)
}

func abortErrorText(err abortError) string {
	return fmt.Sprintf("execution aborted with exit code %d", err.code)
}

func signalActionErrorText(index int) string {
	return fmt.Sprintf("signal action failed at index %d", index)
}

func signalEndpointOverflowErrorText(capacity, buffered int) string {
	return fmt.Sprintf("signal endpoint overflow: buffered %d exceeds capacity %d", buffered, capacity)
}

func signalCancellationErrorText(_ *SignalCancellation) string {
	return "signal canceled primary execution"
}

func signalTerminationErrorText(err *SignalTermination) string {
	if err == nil || !err.Signal.valid() {
		return "execution terminated by signal"
	}
	return "execution terminated by signal"
}
