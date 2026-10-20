package clitest

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

// Compile is a Commandment-specific test setup boundary. It marks itself as a
// testing helper, calls cli.Compile exactly once, and returns that exact
// Application on success. On failure it reports safe structured diagnostic
// context and terminates the calling test; it does not retry, cache, normalize,
// replay, or define alternate compilation semantics.
func Compile[C any](t testing.TB, definition cli.Definition[C]) cli.Application {
	t.Helper()
	app, err := cli.Compile(definition)
	if err == nil {
		return app
	}

	report := cli.Diagnostics(err)
	var actual evaluationField = evaluationText("non-diagnostic error")
	if len(report.Diagnostics) > 0 {
		codes := make([]cli.DiagnosticCode, len(report.Diagnostics))
		for i, diagnostic := range report.Diagnostics {
			codes[i] = diagnostic.Code
		}
		actual = diagnosticCodeCollectionField{values: codes}
	}
	failEvaluationResults(t, disprovedResult(evaluationText("compilation"), evaluationText("success"), actual))
	panic("clitest: testing.TB.FailNow returned")
}

// CompileFails acquires a structured compile rejection for semantic expectation. It
// marks itself as a testing helper, calls cli.Compile exactly once, and fails the
// calling test if compilation succeeds or the error exposes no Commandment
// diagnostics. On structured rejection it returns a DiagnosticsResult for that
// exact error; expectations remain the responsibility of DiagnosticsResult.Require.
func CompileFails[C any](t testing.TB, definition cli.Definition[C]) DiagnosticsResult {
	t.Helper()
	_, err := cli.Compile(definition)
	if err == nil {
		failEvaluationResults(t, disprovedResult(evaluationText("compilation"), evaluationText("structured diagnostic rejection"), evaluationText("success")))
		panic("clitest: testing.TB.FailNow returned")
	}

	report := cli.Diagnostics(err)
	if len(report.Diagnostics) == 0 {
		failEvaluationResults(t, disprovedResult(evaluationText("compilation"), evaluationText("structured diagnostic rejection"), evaluationText("non-diagnostic error")))
		panic("clitest: testing.TB.FailNow returned")
	}

	state := &diagnosticsResultState{
		observation: report,
	}

	result := DiagnosticsResult{
		state: state,
	}

	return result
}
