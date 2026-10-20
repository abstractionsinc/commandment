package clitest

import (
	"context"
	"fmt"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

// InspectionExpectation is a closed semantic expectation applicable to an
// InspectionResult. Values returned by clitest are immutable reusable snapshots.
// Custom inspection logic remains ordinary Go over the public Inspection
// returned by InspectionResult.Inspection.
type InspectionExpectation interface{ inspectionExpectation() }

// CompletionExpectation is a closed semantic expectation applicable to a
// CompletionResult. Values returned by clitest are immutable reusable snapshots.
// Custom completion logic remains ordinary Go over the public Completion
// returned by CompletionResult.Completion.
type CompletionExpectation interface{ completionExpectation() }

// SchemaExpectation is a closed semantic expectation applicable to a SchemaResult.
// Values returned by clitest are immutable reusable snapshots.
type SchemaExpectation interface{ schemaExpectation() }

// ReferenceExpectation is a closed semantic expectation applicable to a
// ReferenceResult. Values returned by clitest are immutable reusable snapshots.
type ReferenceExpectation interface{ referenceExpectation() }

// DiagnosticReportExpectation is a closed semantic expectation applicable to a
// DiagnosticsResult. Values returned by clitest are immutable reusable
// snapshots. Application-owned error interpretation remains ordinary Go.
type DiagnosticReportExpectation interface{ diagnosticReportExpectation() }

// SelectionExpectation is a selection expectation applicable to Inspection and
// Completion observations.
type SelectionExpectation interface {
	InspectionExpectation
	CompletionExpectation
}

// OutcomeExpectation is an outcome expectation applicable where the same structured
// Commandment fact is authoritative in Inspection, Completion, and Diagnostics.
type OutcomeExpectation interface {
	InspectionExpectation
	CompletionExpectation
	DiagnosticReportExpectation
}

// FailureExpectation states that a fallible operation is expected to fail with
// structured Commandment diagnostics. It is produced by FailsWith.
type FailureExpectation interface {
	InspectionExpectation
	CompletionExpectation
	ReferenceExpectation
}

// EffectiveLocaleExpectation checks the authoritative locale on Reference, Inspection,
// Completion, or an already-observed HelpPage.
type EffectiveLocaleExpectation interface {
	ReferenceExpectation
	InspectionExpectation
	CompletionExpectation
	HelpPresentationExpectation
}

// ReleaseExpectation checks the evaluated release where it is authoritative in
// Schema, Reference, and Inspection observations.
//
//nolint:iface // Distinct closed public expectation roles are part of the accepted clitest API.
type ReleaseExpectation interface {
	SchemaExpectation
	ReferenceExpectation
	InspectionExpectation
}

// DiagnosticTruncationExpectation checks diagnostic truncation on Completion
// or Diagnostics observations.
type DiagnosticTruncationExpectation interface {
	CompletionExpectation
	DiagnosticReportExpectation
}

// InspectionResult is a testing carrier for one exact public Inspection
// observation and its operation error; it has no independent Commandment
// semantic identity. Its zero value is invalid; obtain a value from Inspect.
// Requiring or reading it does not rerun inspection.
type InspectionResult struct {
	_     [0]func()
	state *inspectionResultState
}

// CompletionResult is a testing carrier for one exact public Completion
// observation and its operation error; it has no independent Commandment
// semantic identity. Its zero value is invalid; obtain a value from Complete.
// Requiring or reading it does not rerun completion.
type CompletionResult struct {
	_     [0]func()
	state *completionResultState
}

// SchemaResult is a testing carrier for one exact public Schema observation; it
// has no independent Commandment semantic identity. Its zero value is invalid;
// obtain a value from Schema. Requiring or reading it does not rerun schema
// projection.
type SchemaResult struct {
	_     [0]func()
	state *schemaResultState
}

// ReferenceResult is a testing carrier for one exact public Reference
// observation and its operation error; it has no independent Commandment
// semantic identity. Its zero value is invalid; obtain a value from Reference.
// Requiring or reading it does not rerun reference projection.
type ReferenceResult struct {
	_     [0]func()
	state *referenceResultState
}

// DiagnosticsResult is a testing carrier for the exact public DiagnosticReport
// extracted from an existing error; it has no independent Commandment semantic
// identity. Its zero value is invalid; obtain a value from Diagnostics or
// CompileFails. It does not reinterpret or rerun the operation that failed.
type DiagnosticsResult struct {
	_     [0]func()
	state *diagnosticsResultState
}

// Inspect performs exactly one public Application.Inspect operation and retains
// its exact Inspection and error for reusable expectation evaluation or ordinary-Go
// access.
func Inspect(ctx context.Context, app cli.Application, request cli.InspectionRequest) InspectionResult {
	observation, err := app.Inspect(ctx, request)

	state := &inspectionResultState{
		observation: observation,
		err:         err,
	}

	result := InspectionResult{
		state: state,
	}

	return result
}

// Complete performs exactly one public Application.Complete operation and
// retains its exact Completion and error for reusable semantic expectation or
// ordinary-Go access.
func Complete(ctx context.Context, app cli.Application, request cli.CompletionRequest) CompletionResult {
	observation, err := app.Complete(ctx, request)

	state := &completionResultState{
		observation: observation,
		err:         err,
	}

	result := CompletionResult{
		state: state,
	}

	return result
}

// Schema captures the application's exact public Schema once for reusable
// expectation evaluation or ordinary-Go access.
func Schema(app cli.Application) SchemaResult {
	state := &schemaResultState{
		observation: app.Schema(),
	}

	result := SchemaResult{
		state: state,
	}

	return result
}

// Reference performs exactly one public Application.Reference operation and
// retains its exact Reference and error for reusable semantic expectation or
// ordinary-Go access.
func Reference(app cli.Application, options cli.ReferenceOptions) ReferenceResult {
	observation, err := app.Reference(options)

	state := &referenceResultState{
		observation: observation,
		err:         err,
	}

	result := ReferenceResult{
		state: state,
	}

	return result
}

// Diagnostics captures the exact public DiagnosticReport exposed by err. It
// does not rerun or reinterpret the operation that produced err.
func Diagnostics(err error) DiagnosticsResult {
	state := &diagnosticsResultState{
		observation: cli.Diagnostics(err),
	}

	result := DiagnosticsResult{
		state: state,
	}

	return result
}

// FailsWith states that a fallible wrapped operation is expected to fail and
// that its structured DiagnosticReport must satisfy expectations. Without
// FailsWith, success remains the default expectation.
func FailsWith(expectations ...DiagnosticReportExpectation) FailureExpectation {
	return failureExpectation(expectations)
}

// EffectiveLocale checks the authoritative effective locale on each operation to which
// the returned expectation is applicable.
func EffectiveLocale(locale cli.Locale) EffectiveLocaleExpectation {
	return effectiveLocaleExpectationValue{
		locale: locale,
	}
}

// EvaluatedRelease checks the release value under which the observed declaration
// was evaluated.
func EvaluatedRelease(release string) ReleaseExpectation {
	return releaseExpectationValue{
		release: release,
	}
}

// InspectionReady checks that the Inspection reached ready status.
func InspectionReady() InspectionExpectation {
	return inspectionStatusExpectation{
		status: cli.InspectionStatusReady,
	}
}

// InspectionRejected checks that the Inspection reached rejected status.
func InspectionRejected() InspectionExpectation {
	return inspectionStatusExpectation{
		status: cli.InspectionStatusRejected,
	}
}

// InspectionIncomplete checks that the Inspection reached incomplete status.
func InspectionIncomplete() InspectionExpectation {
	return inspectionStatusExpectation{
		status: cli.InspectionStatusIncomplete,
	}
}

// InspectionTruncated checks that the Inspection reports truncated output.
func InspectionTruncated() InspectionExpectation { return inspectionTruncatedExpectation{} }

// Require evaluates the retained Inspection against expectations. It marks itself
// as a test helper, aggregates independent mismatches for this observation, and
// never reruns inspection.
//
//nolint:dupl // Each result type dispatches directly to its closed operation-specific evaluator interface.
func (result InspectionResult) Require(t testing.TB, expectations ...InspectionExpectation) {
	t.Helper()

	if result.state == nil {
		panic("clitest: invalid inspection result")
	}

	if result.state.err != nil && !expectsOperationFailure(expectations) {
		outcome := disprovedResult(
			evaluationText("inspection operation"),
			evaluationText("success"),
			evaluationText("operation returned an error"),
		)
		outcomes := []evaluationResult{outcome}

		reportEvaluationResults(t, outcomes)

		return
	}

	outcomes := make([]evaluationResult, 0, len(expectations))
	for _, expectation := range expectations {
		if expectation == nil {
			outcome := disprovedResult(
				evaluationText("inspection"),
				evaluationText("valid expectation"),
				evaluationText("nil expectation"),
			)
			outcomes = append(outcomes, outcome)
			continue
		}

		evaluator, ok := expectation.(inspectionEvaluator)
		if !ok {
			panic(fmt.Sprintf("clitest: invalid InspectionExpectation implementation %T", expectation))
		}

		outcome := evaluator.evaluateInspection(result.state.observation, result.state.err)
		outcomes = append(outcomes, outcome)
	}

	reportEvaluationResults(t, outcomes)
}

// Inspection returns a detached structural snapshot of the exact public Inspection retained by Inspect, plus the exact
// retained opaque error unchanged. It panics on an invalid zero result.
func (result InspectionResult) Inspection() (cli.Inspection, error) {
	if result.state == nil {
		panic("clitest: invalid inspection result")
	}
	observation := snapshotInspection(result.state.observation)

	return observation, result.state.err
}

// Require evaluates the retained Completion against expectations. It marks itself
// as a test helper, aggregates independent mismatches for this observation, and
// never reruns completion.
//
//nolint:dupl // Each result type dispatches directly to its closed operation-specific evaluator interface.
func (result CompletionResult) Require(t testing.TB, expectations ...CompletionExpectation) {
	t.Helper()

	if result.state == nil {
		panic("clitest: invalid completion result")
	}

	if result.state.err != nil && !expectsOperationFailure(expectations) {
		outcome := disprovedResult(
			evaluationText("completion operation"),
			evaluationText("success"),
			evaluationText("operation returned an error"),
		)
		outcomes := []evaluationResult{outcome}

		reportEvaluationResults(t, outcomes)

		return
	}

	outcomes := make([]evaluationResult, 0, len(expectations))
	for _, expectation := range expectations {
		if expectation == nil {
			outcome := disprovedResult(
				evaluationText("completion"),
				evaluationText("valid expectation"),
				evaluationText("nil expectation"),
			)
			outcomes = append(outcomes, outcome)
			continue
		}

		evaluator, ok := expectation.(completionEvaluator)
		if !ok {
			panic(fmt.Sprintf("clitest: invalid CompletionExpectation implementation %T", expectation))
		}

		outcome := evaluator.evaluateCompletion(result.state.observation, result.state.err)
		outcomes = append(outcomes, outcome)
	}

	reportEvaluationResults(t, outcomes)
}

// Completion returns a detached structural snapshot of the exact public Completion retained by Complete, plus the
// exact retained opaque error unchanged. It panics on an invalid zero result.
func (result CompletionResult) Completion() (cli.Completion, error) {
	if result.state == nil {
		panic("clitest: invalid completion result")
	}
	observation := snapshotCompletion(result.state.observation)

	return observation, result.state.err
}

// Require evaluates the retained Schema against expectations. It marks itself as a
// test helper, aggregates independent mismatches for this observation, and never
// reacquires the Schema.
func (result SchemaResult) Require(t testing.TB, expectations ...SchemaExpectation) {
	t.Helper()

	if result.state == nil {
		panic("clitest: invalid schema result")
	}

	outcomes := make([]evaluationResult, 0, len(expectations))
	for _, expectation := range expectations {
		if expectation == nil {
			outcome := disprovedResult(
				evaluationText("schema"),
				evaluationText("valid expectation"),
				evaluationText("nil expectation"),
			)
			outcomes = append(outcomes, outcome)
			continue
		}

		evaluator, ok := expectation.(schemaEvaluator)
		if !ok {
			panic(fmt.Sprintf("clitest: invalid SchemaExpectation implementation %T", expectation))
		}

		outcome := evaluator.evaluateSchema(&result.state.observation)
		outcomes = append(outcomes, outcome)
	}

	reportEvaluationResults(t, outcomes)
}

// Schema returns a detached structural snapshot of the exact public Schema retained by the Schema operation. It
// panics on an invalid zero result.
func (result SchemaResult) Schema() cli.Schema {
	if result.state == nil {
		panic("clitest: invalid schema result")
	}
	observation := snapshotSchema(result.state.observation)

	return observation
}

// Require evaluates the retained Reference against expectations. It marks itself
// as a test helper, aggregates independent mismatches for this observation, and
// never reruns reference projection.
//
//nolint:dupl // Each result type dispatches directly to its closed operation-specific evaluator interface.
func (result ReferenceResult) Require(t testing.TB, expectations ...ReferenceExpectation) {
	t.Helper()

	if result.state == nil {
		panic("clitest: invalid reference result")
	}

	if result.state.err != nil && !expectsOperationFailure(expectations) {
		outcome := disprovedResult(
			evaluationText("reference operation"),
			evaluationText("success"),
			evaluationText("operation returned an error"),
		)
		outcomes := []evaluationResult{outcome}

		reportEvaluationResults(t, outcomes)

		return
	}

	outcomes := make([]evaluationResult, 0, len(expectations))
	for _, expectation := range expectations {
		if expectation == nil {
			outcome := disprovedResult(
				evaluationText("reference"),
				evaluationText("valid expectation"),
				evaluationText("nil expectation"),
			)
			outcomes = append(outcomes, outcome)
			continue
		}

		evaluator, ok := expectation.(referenceEvaluator)
		if !ok {
			panic(fmt.Sprintf("clitest: invalid ReferenceExpectation implementation %T", expectation))
		}

		outcome := evaluator.evaluateReference(result.state.observation, result.state.err)
		outcomes = append(outcomes, outcome)
	}

	reportEvaluationResults(t, outcomes)
}

// Reference returns a detached structural snapshot of the exact public Reference retained by Reference, plus the exact
// retained opaque error unchanged. It panics on an invalid zero result.
func (result ReferenceResult) Reference() (cli.Reference, error) {
	if result.state == nil {
		panic("clitest: invalid reference result")
	}
	observation := snapshotReference(result.state.observation)

	return observation, result.state.err
}

// Require evaluates the retained DiagnosticReport against expectations. It marks
// itself as a test helper, aggregates independent mismatches for this observation,
// and does not reinterpret the source error.
func (result DiagnosticsResult) Require(t testing.TB, expectations ...DiagnosticReportExpectation) {
	t.Helper()

	if result.state == nil {
		panic("clitest: invalid diagnostics result")
	}

	outcomes := make([]evaluationResult, 0, len(expectations))
	for _, expectation := range expectations {
		if expectation == nil {
			outcome := disprovedResult(
				evaluationText("diagnostic report"),
				evaluationText("valid expectation"),
				evaluationText("nil expectation"),
			)
			outcomes = append(outcomes, outcome)
			continue
		}

		evaluator, ok := expectation.(diagnosticEvaluator)
		if !ok {
			panic(fmt.Sprintf("clitest: invalid DiagnosticReportExpectation implementation %T", expectation))
		}

		outcome := evaluator.evaluateDiagnosticReport(result.state.observation)
		outcomes = append(outcomes, outcome)
	}

	reportEvaluationResults(t, outcomes)
}

// Report returns a detached structural snapshot of the exact public DiagnosticReport retained by Diagnostics or
// CompileFails. It panics on an invalid zero result.
func (result DiagnosticsResult) Report() cli.DiagnosticReport {
	if result.state == nil {
		panic("clitest: invalid diagnostics result")
	}
	report := snapshotDiagnosticReport(result.state.observation)

	return report
}

type explicitFailureExpectation interface {
	expectsOperationFailure()
}

type releaseExpectationValue struct{ release string }

type inspectionStatusExpectation struct{ status cli.InspectionStatus }

type inspectionTruncatedExpectation struct{}

func compareLocale(subject string, expected, actual cli.Locale) evaluationResult {
	if actual == expected {
		return provedResult()
	}
	return disprovedResult(evaluationText(subject), quotedTextField(expected), quotedTextField(actual))
}

func compareRelease(subject, expected, actual string) evaluationResult {
	if actual == expected {
		return provedResult()
	}
	return disprovedResult(evaluationText(subject), quotedTextField(expected), quotedTextField(actual))
}

func (releaseExpectationValue) schemaExpectation() {}

func (releaseExpectationValue) referenceExpectation() {}

func (releaseExpectationValue) inspectionExpectation() {}

func (expectation releaseExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	if schema == nil {
		return notProvableResult(evaluationText("evaluated release"), quotedTextField(expectation.release), evaluationText("schema is unavailable"))
	}
	return compareRelease("schema evaluated release", expectation.release, schema.EvaluatedRelease)
}

func (expectation releaseExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return compareRelease("reference evaluated release", expectation.release, reference.EvaluatedRelease)
}

func (expectation releaseExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return compareRelease("inspection evaluated release", expectation.release, inspection.EvaluatedRelease)
}

func (inspectionStatusExpectation) inspectionExpectation() {}

func (expectation inspectionStatusExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	if inspection.Status == expectation.status {
		return provedResult()
	}
	return disprovedResult(
		evaluationText("inspection status"),
		evaluationText(fmt.Sprintf("%v", expectation.status)),
		evaluationText(fmt.Sprintf("%v", inspection.Status)),
	)
}

func (inspectionTruncatedExpectation) inspectionExpectation() {}

func (inspectionTruncatedExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	if inspection.Truncated {
		return provedResult()
	}
	return disprovedResult(evaluationText("inspection"), evaluationText("truncated observation"), evaluationText("observation is complete"))
}
