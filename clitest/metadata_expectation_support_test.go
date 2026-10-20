package clitest

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func schemaExpectationOutcome(t *testing.T, expectation SchemaExpectation, schema *cli.Schema) evaluationResult {
	t.Helper()
	evaluator, ok := expectation.(interface {
		evaluateSchema(schema *cli.Schema) evaluationResult
	})
	if !ok {
		t.Fatalf("expectation %T has no schema evaluator", expectation)
	}
	return evaluator.evaluateSchema(schema)
}

func referenceExpectationOutcome(t *testing.T, expectation ReferenceExpectation, reference cli.Reference) evaluationResult {
	t.Helper()
	evaluator, ok := expectation.(interface {
		evaluateReference(reference cli.Reference, err error) evaluationResult
	})
	if !ok {
		t.Fatalf("expectation %T has no reference evaluator", expectation)
	}
	return evaluator.evaluateReference(reference, nil)
}
