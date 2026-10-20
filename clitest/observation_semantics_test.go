package clitest

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func inspectionExpectationOutcome(t *testing.T, expectation InspectionExpectation, inspection cli.Inspection) evaluationResult {
	t.Helper()
	evaluator, ok := expectation.(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !ok {
		t.Fatalf("expectation %T has no inspection evaluator", expectation)
	}
	return evaluator.evaluateInspection(inspection, nil)
}
