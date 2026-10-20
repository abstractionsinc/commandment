package clitest

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func assertCompletionExpectation(t *testing.T, expectation CompletionExpectation, result cli.Completion) {
	t.Helper()
	if outcome := evaluateCompletionExpectation(t, expectation, result); outcome.status != evaluationProved {
		t.Fatalf("completion expectation = %#v", outcome)
	}
}

func assertInspectionExpectation(t *testing.T, expectation InspectionExpectation, inspection cli.Inspection) {
	t.Helper()
	evaluator, ok := expectation.(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !ok {
		t.Fatalf("expectation %T has no inspection evaluator", expectation)
	}
	if outcome := evaluator.evaluateInspection(inspection, nil); outcome.status != evaluationProved {
		t.Fatalf("inspection expectation = %#v", outcome)
	}
}

func evaluateCompletionExpectation(t *testing.T, expectation CompletionExpectation, result cli.Completion) evaluationResult {
	t.Helper()
	evaluator, ok := expectation.(interface {
		evaluateCompletion(completion cli.Completion, err error) evaluationResult
	})
	if !ok {
		t.Fatalf("expectation %T has no completion evaluator", expectation)
	}
	return evaluator.evaluateCompletion(result, nil)
}
