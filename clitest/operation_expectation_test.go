package clitest

import (
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func invalidDeclarationFailure(t *testing.T) error {
	t.Helper()
	definition := cli.Definition[struct{}]{
		Name: "invalid-declaration-fixture",
		Localization: cli.Localization{
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"x": "y",
				},
			}},
		},
	}
	_, err := cli.Compile(definition)
	if err == nil {
		t.Fatal("invalid declaration fixture unexpectedly compiled")
	}
	return err //nolint:wrapcheck // The expectation helper must preserve the exact compile error for attribution.
}

func TestFailurePreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	err := invalidDeclarationFailure(t)
	t.Run("valid expectation: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		expectation, ok := failureExpectation([]DiagnosticReportExpectation{nil}).(failureExpectationValue)
		if !ok {
			t.Fatal("failureExpectation must return failureExpectationValue")
		}
		outcome := expectation.evaluateFailure(
			"inspection",
			err,
		)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) == "" {
			t.Fatalf("nil diagnostic expectation = %#v", outcome)
		}
	})
	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		nested := fixedDiagnosticReportExpectation{
			result: notProvableResult(
				evaluationText("diagnostic"),
				evaluationText("observable"),
				evaluationText("observation is incomplete"),
			),
		}
		expectation, ok := failureExpectation([]DiagnosticReportExpectation{nested}).(failureExpectationValue)
		if !ok {
			t.Fatal("failureExpectation must return failureExpectationValue")
		}
		outcome := expectation.evaluateFailure(
			"inspection",
			err,
		)
		if outcome.status != evaluationNotProvable || renderEvaluationField(outcome.observed) == "" {
			t.Fatalf("not provable diagnostic expectation = %#v", outcome)
		}
	})
	t.Run("expected structured failure diagnostics: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		nested := fixedDiagnosticReportExpectation{
			result: disprovedResult(
				evaluationText("diagnostic"),
				evaluationText("expected"),
				evaluationText("actual"),
			),
		}
		expectation, ok := failureExpectation([]DiagnosticReportExpectation{nested}).(failureExpectationValue)
		if !ok {
			t.Fatal("failureExpectation must return failureExpectationValue")
		}
		outcome := expectation.evaluateFailure(
			"inspection",
			err,
		)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) == "" {
			t.Fatalf("unsatisfied diagnostic expectation = %#v", outcome)
		}
	})
}

func TestEffectiveLocalePreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	expectation := effectiveLocaleExpectationValue{
		locale: "fr",
	}
	for _, outcome := range []evaluationResult{
		expectation.evaluateReference(cli.Reference{
			Locale: "en",
		}, nil),
		expectation.evaluateInspection(cli.Inspection{
			Locale: "en",
		}, nil),
		expectation.evaluateCompletion(cli.Completion{
			Locale: "en",
		}, nil),
	} {
		if outcome.status != evaluationDisproved {
			t.Fatalf("locale mismatch = %#v", outcome)
		}
	}
}

func TestReleasePreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	outcome := releaseExpectationValue{
		release: "1.0.0",
	}.evaluateSchema(nil)
	if outcome.status != evaluationNotProvable {
		t.Fatalf("nil schema release = %#v", outcome)
	}
}

func TestReleasePreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	expectation := releaseExpectationValue{
		release: "1.0.0",
	}
	for _, outcome := range []evaluationResult{
		expectation.evaluateSchema(&cli.Schema{
			EvaluatedRelease: "2.0.0",
		}),
		expectation.evaluateReference(cli.Reference{
			EvaluatedRelease: "2.0.0",
		}, nil),
		expectation.evaluateInspection(cli.Inspection{
			EvaluatedRelease: "2.0.0",
		}, nil),
	} {
		if outcome.status != evaluationDisproved {
			t.Fatalf("release mismatch = %#v", outcome)
		}
	}
}

func TestInspectionTruncatedPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	outcome := inspectionTruncatedExpectation{}.evaluateInspection(cli.Inspection{}, nil)
	if outcome.status != evaluationDisproved {
		t.Fatalf("complete inspection = %#v", outcome)
	}
}

func TestCompletionRankingAlgorithmPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	outcome := completionRankingExpectation{
		algorithm: cli.CompletionRankingAlgorithm("expected"),
	}.evaluateCompletion(cli.Completion{
		RankingAlgorithm: "actual",
	}, nil)
	if outcome.status != evaluationDisproved {
		t.Fatalf("ranking mismatch = %#v", outcome)
	}
}

func TestCompletionSpacingPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	outcome := completionSpacingExpectation{
		spacing: cli.CompletionSpacingDefault,
	}.evaluateCompletion(cli.Completion{
		Spacing: cli.CompletionSpacingAppendSpace,
	}, nil)
	if outcome.status != evaluationDisproved {
		t.Fatalf("spacing mismatch = %#v", outcome)
	}
}

type fixedDiagnosticReportExpectation struct {
	result evaluationResult
}

func (fixedDiagnosticReportExpectation) diagnosticReportExpectation() {}

func (expectation fixedDiagnosticReportExpectation) evaluateDiagnosticReport(cli.DiagnosticReport) evaluationResult {
	return expectation.result
}

type unsupportedDiagnosticReportExpectation struct{}

func (unsupportedDiagnosticReportExpectation) diagnosticReportExpectation() {}

func TestFailureExpectationRejectsUnsupportedInternalDiagnosticExpectation(t *testing.T) {
	t.Parallel()
	err := invalidDeclarationFailure(t)
	expectations := []DiagnosticReportExpectation{unsupportedDiagnosticReportExpectation{}}
	expectation, ok := failureExpectation(expectations).(failureExpectationValue)
	if !ok {
		t.Fatal("failureExpectation must return failureExpectationValue")
	}
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("unsupported diagnostic expectation did not panic")
		}
		message, ok := recovered.(string)
		if !ok || !strings.Contains(message, "invalid DiagnosticReportExpectation implementation") {
			t.Fatalf("panic = %#v", recovered)
		}
	}()
	expectation.evaluateFailure("inspection", err)
	t.Fatal("unsupported diagnostic expectation returned")
}
