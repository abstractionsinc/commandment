package clitest

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/abstractionsinc/commandment/cli"
)

const (
	failureReportingHelperEnv = "COMMANDMENT_CLITEST_FAILURE_REPORTING_HELPER"
	failureReportingHelperArg = "commandment-clitest-failure-reporting-child"
)

func failureReportingHelperMode() bool {
	if os.Getenv(failureReportingHelperEnv) != "1" {
		return false
	}
	return slices.Contains(os.Args[1:], failureReportingHelperArg)
}

func failureReportingChildEnv() []string {
	environment := make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		name, _, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(name, failureReportingHelperEnv) {
			continue
		}
		environment = append(environment, entry)
	}
	return append(environment, failureReportingHelperEnv+"=1")
}

func TestFailureReportingContract(t *testing.T) {
	t.Parallel()
	if failureReportingHelperMode() {
		runFailureReportingHelper(t)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	args := []string{"-test.run=^TestFailureReportingContract$", "-test.v"}
	if coverDir := os.Getenv("GOCOVERDIR"); coverDir != "" {
		args = append(args, "-test.gocoverdir="+coverDir)
	}
	args = append(args, failureReportingHelperArg)
	//nolint:gosec // The test deliberately launches the current test binary with test-owned arguments.
	cmd := exec.CommandContext(ctx, os.Args[0], args...)
	cmd.Env = failureReportingChildEnv()
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("failure helper exceeded test-owned watchdog: %v\n%s", ctx.Err(), output)
	}
	if err == nil {
		t.Fatalf("failure helper unexpectedly passed:\n%s", output)
	}
	text := string(output)

	// Independent mismatches are accumulated in caller expectation order.
	first := strings.Index(text, "observed: present")
	second := strings.Index(text, "not provable:")
	third := strings.Index(text, "expected: exact signal action failure")
	if first < 0 || second < 0 || third < 0 || first >= second || second >= third {
		t.Fatalf("failure ordering/aggregation missing:\n%s", text)
	}

	// A truncated-domain absence expectation reports one not-provable premise outcome;
	// it does not emit derivative invented-absence noise.
	if got := strings.Count(text, "not provable:"); got != 1 {
		t.Fatalf("not-provable derivative suppression count = %d, want 1:\n%s", got, text)
	}
	for _, want := range []string{
		"clitest: expectation failed",
		"diagnostic",
		"expected:",
		"observed:",
		"failure_reporting_test.go:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("failure output missing %q:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{
		"actual:",
		"observed-secret-action",
		"expected-secret-action",
		"clitest/main.go:",
		"clitest/evaluation.go:",
		"unsupported clitest.unsupportedInspectionExpectation",
		"unsupported clitest.unsupportedReferenceExpectation",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("failure output leaked/internal-attributed/derivative %q:\n%s", forbidden, text)
		}
	}
}

func TestExpectationFailureFormattingIsDeterministicSafeAndBounded(t *testing.T) {
	t.Parallel()
	outcomes := []evaluationResult{
		disprovedResult(evaluationText("flag --token\nsubject"), evaluationText("value=expected\t\x1b[31m"), evaluationText("value=observed\r\x00")),
		notProvableResult(evaluationText("diagnostic \"unknown-flag\""), evaluationText("absent"), evaluationText("diagnostic report is truncated")),
	}

	first := formatEvaluationReport(outcomes)
	second := formatEvaluationReport(outcomes)
	if first != second {
		t.Fatalf("failure report is nondeterministic:\nfirst=%q\nsecond=%q", first, second)
	}
	for _, want := range []string{"expected:", "observed:", "not provable:", `\n`, `\t`, `\r`, `\x1b`, `\x00`} {
		if !strings.Contains(first, want) {
			t.Fatalf("safe report missing %q: %q", want, first)
		}
	}
	for _, forbidden := range []string{"actual:", "\x1b[31m", "flag --token\nsubject"} {
		if strings.Contains(first, forbidden) {
			t.Fatalf("safe report contains structural/control payload %q: %q", forbidden, first)
		}
	}

	huge := strings.Repeat("x\n\t\x1b", failureReportMaxBytes)
	many := make([]evaluationResult, failureReportMaxOutcomes+17)
	for i := range many {
		many[i] = disprovedResult(evaluationText("repeated subject"), evaluationText(huge), evaluationText(huge))
	}
	bounded := formatEvaluationReport(many)
	if len(bounded) > failureReportMaxBytes {
		t.Fatalf("bounded report length = %d, max %d", len(bounded), failureReportMaxBytes)
	}
	if !strings.Contains(bounded, "additional failure(s) omitted") {
		t.Fatalf("bounded report omitted no root-count summary: %q", bounded)
	}
	for _, control := range []rune{'\x1b', '\t', '\r', '\x00'} {
		if strings.ContainsRune(bounded, control) {
			t.Fatalf("bounded report contains raw terminal control %q", control)
		}
	}
}

func TestExpectationFailureFormattingSuppressesMetOutcomesAndPreservesRootOrder(t *testing.T) {
	t.Parallel()
	value := []evaluationResult{
		provedResult(),
		disprovedResult(evaluationText("first semantic subject"), evaluationText("one"), evaluationText("two")),
		provedResult(),
		notProvableResult(evaluationText("second semantic subject"), evaluationText("complete"), evaluationText("observation is partial")),
	}
	report := formatEvaluationReport(value)
	first := strings.Index(report, "1. first semantic subject")
	second := strings.Index(report, "2. second semantic subject")
	if first < 0 || second < 0 || first >= second {
		t.Fatalf("root order = %q", report)
	}
	if strings.Contains(report, "3.") {
		t.Fatalf("proved outcomes consumed failure roots: %q", report)
	}
}

func runFailureReportingHelper(t *testing.T) {
	t.Helper()

	observedAt := time.Unix(1700000400, 43)
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{
			{Code: cli.DiagnosticInvalidValue},
			{
				Code: cli.DiagnosticSignalActionFailure,
				SignalAction: &cli.SignalActionFailure{
					CommandPath: []string{"root", "serve"},
					PolicyID:    cli.SignalPolicyID(1),
					Index:       2,
					Action: cli.SignalAction{
						Kind:       cli.SignalActionHandle,
						Name:       "observed-secret-action",
						EndpointID: cli.SignalEndpointID(2),
						Sensitive:  true,
					},
					Event: cli.SignalEvent{
						Signal:          testSemanticSignal(t),
						Origin:          cli.SignalOriginProcess,
						Phase:           cli.SignalPhaseExecution,
						ObservedAt:      observedAt,
						FirstObservedAt: observedAt,
						LastObservedAt:  observedAt,
						FirstOccurrence: 1,
						LastOccurrence:  1,
						BatchSize:       1,
					},
					Phase: cli.SignalPhaseExecution,
				},
			},
		},
	}
	report.Truncated = true
	result := DiagnosticsResult{
		state: &diagnosticsResultState{observation: report},
	}

	expectedAction := *report.Diagnostics[1].SignalAction
	expectedAction.Action.Name = "expected-secret-action"
	result.Require(t,
		Diagnostic(cli.DiagnosticInvalidValue).Absent(),
		Diagnostic(cli.DiagnosticUnknownFlag).Absent(),
		DiagnosticAt(1).SignalActionFailure(expectedAction),
	)
	(DiagnosticsResult{state: &diagnosticsResultState{observation: cli.DiagnosticReport{}}}).Require(t, nil)
}

func TestFailsWithRequiresAndChecksStructuredCommandmentFailure(t *testing.T) {
	t.Parallel()
	nested := []DiagnosticReportExpectation{Diagnostic(cli.DiagnosticInvalidDeclaration).Present()}
	expectation, ok := FailsWith(nested...).(failureExpectationValue)
	if !ok {
		t.Fatal("FailsWith must satisfy failureExpectationValue")
	}
	nested[0] = nil

	if len(expectation.expectations) != 1 || expectation.expectations[0] == nil {
		t.Fatal("FailsWith did not snapshot retained expectations")
	}
	if got := expectation.evaluateFailure("inspection", nil); got.status != evaluationDisproved {
		t.Fatalf("success under FailsWith = %#v, want disproved", got)
	}
	if got := expectation.evaluateFailure(
		"inspection",
		errors.New("application failure"),
	); got.status != evaluationDisproved {
		t.Fatalf("ordinary application error under FailsWith = %#v, want disproved", got)
	}

	err := invalidDeclarationFailure(t)
	if got := expectation.evaluateFailure("inspection", err); got.status != evaluationProved {
		t.Fatalf("structured Commandment failure = %#v, want proved", got)
	}
}

func TestReusableStaticExpectationStateHasNoHiddenCrossGoroutineMutation(t *testing.T) {
	t.Parallel()
	definition := cli.Definition[struct{}]{
		Name: "clitest-concurrent-reuse",
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	result := Schema(app)
	expectation := RootCommand().Exists()
	evaluator, ok := expectation.(interface {
		evaluateSchema(schema *cli.Schema) evaluationResult
	})
	if !ok {
		t.Fatalf("expectation %T has no schema evaluator", expectation)
	}

	const goroutines = 32
	const passes = 64
	var failures atomic.Int64
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			for range passes {
				if evaluator.evaluateSchema(&result.state.observation).status != evaluationProved {
					failures.Add(1)
				}
			}
		}()
	}
	wg.Wait()
	if got := failures.Load(); got != 0 {
		t.Fatalf("concurrent expectation failures = %d", got)
	}
}

func TestDiagnosticsResultRetainsExactObservation(t *testing.T) {
	t.Parallel()
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{Code: cli.DiagnosticInvalidDeclaration}},
		Truncated:   true,
	}
	result := DiagnosticsResult{
		state: &diagnosticsResultState{observation: report},
	}

	got := result.Report()
	if len(got.Diagnostics) != 1 || got.Diagnostics[0].Code != cli.DiagnosticInvalidDeclaration || !got.Truncated {
		t.Fatalf("Report() = %#v", got)
	}
}

func TestOperationResultZeroValueIsInvalid(t *testing.T) {
	t.Parallel()
	assertPanics := func(name string, call func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("zero %s did not panic", name)
			}
		}()
		call()
	}
	assertPanics("InspectionResult", func() { var result InspectionResult; _, _ = result.Inspection() })
	assertPanics("CompletionResult", func() { var result CompletionResult; _, _ = result.Completion() })
	assertPanics("SchemaResult", func() { var result SchemaResult; _ = result.Schema() })
	assertPanics("ReferenceResult", func() { var result ReferenceResult; _, _ = result.Reference() })
	assertPanics("DiagnosticsResult", func() { var result DiagnosticsResult; _ = result.Report() })

	checkedZeroDiagnostic, ok := diagnosticPresentExpectation(DiagnosticLocator{}).(diagnosticExpectation)
	if !ok {
		t.Fatal("diagnosticPresentExpectation must satisfy diagnosticExpectation")
	}
	zeroDiagnostic := checkedZeroDiagnostic.evaluateDiagnosticReport(
		cli.DiagnosticReport{},
	)
	if zeroDiagnostic.status == evaluationProved {
		t.Fatalf("zero DiagnosticLocator unexpectedly proved: %#v", zeroDiagnostic)
	}
	checkedZeroValue, ok := ValueLocator{}.Resolved().(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !ok {
		t.Fatal("Resolved must satisfy expected evaluator interface")
	}
	zeroValue := checkedZeroValue.evaluateInspection(cli.Inspection{}, nil)
	if zeroValue.status == evaluationProved {
		t.Fatalf("zero ValueLocator unexpectedly proved: %#v", zeroValue)
	}
	checkedZeroCompletion, ok := CompletionTargetLocator{}.Present().(interface {
		evaluateCompletion(completion cli.Completion, err error) evaluationResult
	})
	if !ok {
		t.Fatal("Present must satisfy expected evaluator interface")
	}
	zeroCompletion := checkedZeroCompletion.evaluateCompletion(cli.Completion{}, nil)
	if zeroCompletion.status == evaluationProved {
		t.Fatalf("zero CompletionTargetLocator unexpectedly proved: %#v", zeroCompletion)
	}
	checkedZeroCommand, ok := CommandLocator{}.Exists().(interface {
		evaluateSchema(schema *cli.Schema) evaluationResult
	})
	if !ok {
		t.Fatal("Exists must satisfy expected evaluator interface")
	}
	zeroCommand := checkedZeroCommand.evaluateSchema(&cli.Schema{})
	if zeroCommand.status == evaluationProved {
		t.Fatalf("zero CommandLocator unexpectedly proved: %#v", zeroCommand)
	}
}

func TestDiagnosticProducerKindFailureUsesSemanticNames(t *testing.T) {
	t.Parallel()
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{ProducerKind: cli.ValueProducerKindUnknown}},
	}
	expectation, ok := diagnosticProducerKindExpectation(DiagnosticAt(0), cli.ValueProducerKindEnvironment).(diagnosticExpectation)
	if !ok {
		t.Fatal("diagnosticProducerKindExpectation must satisfy diagnosticExpectation")
	}
	got := expectation.evaluateDiagnosticReport(report)
	if got.status != evaluationDisproved || renderEvaluationField(got.expected) != "producer kind = environment" || renderEvaluationField(got.observed) != "unknown" {
		t.Fatalf("producer kind mismatch = %#v", got)
	}
}

func TestDiagnosticPositiveExpectationCanUseTruthfulTruncatedEvidence(t *testing.T) {
	t.Parallel()
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{Code: cli.DiagnosticInvalidDeclaration}},
		Truncated:   true,
	}
	expectation, ok := diagnosticPresentExpectation(Diagnostic(cli.DiagnosticInvalidDeclaration)).(diagnosticExpectation)
	if !ok {
		t.Fatal("diagnosticPresentExpectation must satisfy diagnosticExpectation")
	}
	if got := expectation.evaluateDiagnosticReport(report); got.status != evaluationProved {
		t.Fatalf("positive expectation status = %v, want proved (%#v)", got.status, got)
	}
}

func TestDiagnosticAbsenceIsNotProvableFromTruncatedEvidence(t *testing.T) {
	t.Parallel()
	report := cli.DiagnosticReport{
		Truncated: true,
	}
	expectation, ok := diagnosticAbsentExpectation(Diagnostic(cli.DiagnosticInvalidDeclaration)).(diagnosticExpectation)
	if !ok {
		t.Fatal("diagnosticAbsentExpectation must satisfy diagnosticExpectation")
	}
	if got := expectation.evaluateDiagnosticReport(report); got.status != evaluationNotProvable {
		t.Fatalf("absence expectation status = %v, want not provable (%#v)", got.status, got)
	}
}

func TestDiagnosticsExactlyRequiresCompleteAuthoritativeOrder(t *testing.T) {
	t.Parallel()
	expectation := diagnosticsExactlyExpectationValue{
		locators: []*diagnosticLocatorState{
			Diagnostic(cli.DiagnosticUnknownFlag).state,
			Diagnostic(cli.DiagnosticInvalidValue).state,
		},
	}
	complete := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{
			{Code: cli.DiagnosticUnknownFlag},
			{Code: cli.DiagnosticInvalidValue},
		},
	}
	if got := expectation.evaluateDiagnosticReport(complete); got.status != evaluationProved {
		t.Fatalf("complete exact expectation = %#v", got)
	}

	reversed := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{
			{Code: cli.DiagnosticInvalidValue},
			{Code: cli.DiagnosticUnknownFlag},
		},
	}
	if got := expectation.evaluateDiagnosticReport(reversed); got.status != evaluationDisproved {
		t.Fatalf("reversed exact expectation status = %v, want disproved (%#v)", got.status, got)
	}

	complete.Truncated = true
	if got := expectation.evaluateDiagnosticReport(complete); got.status != evaluationNotProvable {
		t.Fatalf("truncated exact expectation status = %v, want not provable (%#v)", got.status, got)
	}
}

func TestDiagnosticLocatorRefinementSnapshotsPriorValue(t *testing.T) {
	t.Parallel()
	base := Diagnostic(cli.DiagnosticRuleViolation)
	refined := base.Rule(cli.RuleCode("required"))
	again := base.Rule(cli.RuleCode("conflict"))

	if refined.state == nil || again.state == nil || base.state == nil {
		t.Fatal("locator state missing")
	}
	if refined.state == base.state || again.state == base.state || refined.state == again.state {
		t.Fatal("locator refinement mutated or reused prior state")
	}
	if refined.state.rule != cli.RuleCode("required") || again.state.rule != cli.RuleCode("conflict") ||
		base.state.ruleSet {
		t.Fatalf("locator snapshot mismatch: base=%#v refined=%#v again=%#v", base.state, refined.state, again.state)
	}
}

func TestDiagnosticsRequireAcceptsImplementedExpectations(t *testing.T) {
	t.Parallel()
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{
			Code:          cli.DiagnosticInvalidValue,
			Sensitive:     true,
			InputRedacted: true,
		}},
	}
	result := DiagnosticsResult{
		state: &diagnosticsResultState{observation: report},
	}
	result.Require(t,
		Diagnostic(cli.DiagnosticInvalidValue).Present(),
		Diagnostic(cli.DiagnosticInvalidValue).Sensitive(),
		Diagnostic(cli.DiagnosticInvalidValue).InputRedacted(),
		DiagnosticsExactly(Diagnostic(cli.DiagnosticInvalidValue)),
	)
}

func TestDiagnosticLocatorUsesExactSubjectAndRejectsAmbiguousHumanMatch(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	flagID := cli.FlagID(2)
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{
			{
				Code: cli.DiagnosticInvalidValue,
				Rule: &cli.RuleFailure{
					RuleID:   cli.RuleID(1),
					RuleKind: cli.RuleCustom,
					RuleCode: "first",
				},
				CommandPath: []string{"root"},
				Command:     commandID,
				Declaration: cli.DeclarationIdentity{
					Kind: cli.DeclarationKindFlag,
					Flag: flagID,
				},
			},
			{
				Code: cli.DiagnosticInvalidValue,
				Rule: &cli.RuleFailure{
					RuleID:   cli.RuleID(2),
					RuleKind: cli.RuleCustom,
					RuleCode: "second",
				},
				CommandPath: []string{"root"},
				Command:     commandID,
				Declaration: cli.DeclarationIdentity{
					Kind: cli.DeclarationKindFlag,
					Flag: flagID,
				},
			},
		},
	}

	ambiguous, ok := diagnosticPresentExpectation(Diagnostic(cli.DiagnosticInvalidValue)).(diagnosticExpectation)
	if !ok {
		t.Fatal("diagnosticPresentExpectation must satisfy diagnosticExpectation")
	}
	if got := ambiguous.evaluateDiagnosticReport(
		report,
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.observed), "matched 2 diagnostics") {
		t.Fatalf("ambiguous diagnostic locator = %#v", got)
	}
	refined := Diagnostic(cli.DiagnosticInvalidValue).Rule("first").On(CommandByID(commandID))
	refinedExpectation, ok := refined.Present().(diagnosticExpectation)
	if !ok {
		t.Fatal("Present must satisfy diagnosticExpectation")
	}
	if got := refinedExpectation.evaluateDiagnosticReport(
		report,
	); got.status != evaluationProved {
		t.Fatalf("exact subject-refined diagnostic = %#v", got)
	}
	byDeclaration := Diagnostic(cli.DiagnosticInvalidValue).Rule("first").On(FlagByID(flagID))
	declarationExpectation, ok := byDeclaration.Present().(diagnosticExpectation)
	if !ok {
		t.Fatal("Present must satisfy diagnosticExpectation")
	}
	if got := declarationExpectation.evaluateDiagnosticReport(
		report,
	); got.status != evaluationProved {
		t.Fatalf("declaration-subject diagnostic = %#v", got)
	}
	missing := Diagnostic(cli.DiagnosticCode("missing"))
	missingExpectation, ok := missing.Present().(diagnosticExpectation)
	if !ok {
		t.Fatal("Present must satisfy diagnosticExpectation")
	}
	if got := missingExpectation.evaluateDiagnosticReport(
		report,
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.observed), "absent") {
		t.Fatalf("complete-domain missing subject = %#v, want disproved/absent", got)
	}
}

func TestDiagnosticExpectationDoesNotInferMissingGraphFactsFromTruncationOrLeakSensitiveSourceText(t *testing.T) {
	t.Parallel()
	missing := InputOriginByID(cli.InputOriginID{}).Present()
	missingEvaluator, ok := missing.(diagnosticEvaluator)
	if !ok {
		t.Fatal("missing must satisfy diagnosticEvaluator")
	}
	if got := missingEvaluator.evaluateDiagnosticReport(cli.DiagnosticReport{
		Truncated: true,
	}); got.status != evaluationNotProvable {
		t.Fatalf("missing origin from truncated report = %#v", got)
	}
}

func TestDiagnosticStructuredIntegrationFailureExpectationsUseExactPublicObservation(t *testing.T) {
	t.Parallel()
	observedAt := time.Unix(1700000400, 43)
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{
			{
				Code: cli.DiagnosticSignalActionFailure,
				SignalAction: &cli.SignalActionFailure{
					CommandPath: []string{"root", "serve"},
					PolicyID:    cli.SignalPolicyID(1),
					Index:       2,
					Action: cli.SignalAction{
						Kind:       cli.SignalActionHandle,
						Name:       "reload",
						EndpointID: cli.SignalEndpointID(2),
						Signal:     testSemanticSignal(t),
						Sensitive:  true,
					},
					Event: cli.SignalEvent{
						Signal: testSemanticSignal(t),
						Origin: cli.SignalOriginProcess,
						Sender: &cli.SignalSender{
							ProcessID: 51,
							UserID:    52,
							Code:      53,
						},
						Phase:           cli.SignalPhaseExecution,
						ObservedAt:      observedAt,
						FirstObservedAt: observedAt,
						LastObservedAt:  observedAt,
						FirstOccurrence: 1,
						LastOccurrence:  1,
						BatchSize:       1,
					},
					Phase: cli.SignalPhaseExecution,
				},
			},
			{
				Code: cli.DiagnosticSignalOverflow,
				SignalOverflow: &cli.SignalOverflowFailure{
					Boundary: cli.SignalOverflowBoundaryRouting,
					Capacity: 16,
					Buffered: 16,
				},
			},
			{
				Code: cli.DiagnosticNoticeDeliveryFailure,
				NoticeDelivery: &cli.NoticeDeliveryFailure{
					Index:       4,
					Phase:       cli.NoticePhaseCleanup,
					Code:        "migration",
					CommandPath: []string{"root", "serve"},
				},
			},
		},
	}
	result := DiagnosticsResult{
		state: &diagnosticsResultState{observation: report},
	}
	expectedAction := cloneSignalActionFailure(*report.Diagnostics[0].SignalAction)
	expectedOverflow := *report.Diagnostics[1].SignalOverflow
	expectedNotice := *report.Diagnostics[2].NoticeDelivery
	expectedNotice.CommandPath = append([]string(nil), expectedNotice.CommandPath...)
	actionExpectation := Diagnostic(cli.DiagnosticSignalActionFailure).SignalActionFailure(expectedAction)
	overflowExpectation := Diagnostic(cli.DiagnosticSignalOverflow).SignalOverflowFailure(expectedOverflow)
	noticeExpectation := Diagnostic(cli.DiagnosticNoticeDeliveryFailure).NoticeDeliveryFailure(expectedNotice)
	expectedAction.CommandPath[0] = "mutated"
	expectedAction.Event.Sender.ProcessID = 999
	expectedNotice.CommandPath[0] = "mutated"
	result.Require(t, actionExpectation, overflowExpectation, noticeExpectation)
}

func TestDiagnosticStructuredIntegrationFailureMismatchDoesNotRevealStructuredValues(t *testing.T) {
	t.Parallel()
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{
			Code: cli.DiagnosticSignalActionFailure,
			SignalAction: &cli.SignalActionFailure{
				CommandPath: []string{"actual-secret-path"},
				PolicyID:    cli.SignalPolicyID(1),
				Index:       0,
				Action: cli.SignalAction{
					Kind:      cli.SignalActionHandle,
					Name:      "actual-secret-action",
					Sensitive: true,
				},
				Event: cli.SignalEvent{
					Signal: testSemanticSignal(t),
					Phase:  cli.SignalPhaseExecution,
				},
				Phase: cli.SignalPhaseExecution,
			},
		}},
	}
	wrong := cli.SignalActionFailure{
		CommandPath: []string{"expected-secret-path"},
		PolicyID:    report.Diagnostics[0].SignalAction.PolicyID,
		Index:       0,
		Action: cli.SignalAction{
			Kind:      cli.SignalActionHandle,
			Name:      "expected-secret-action",
			Sensitive: true,
		},
		Event: cli.SignalEvent{
			Signal: testSemanticSignal(t),
			Phase:  cli.SignalPhaseExecution,
		},
		Phase: cli.SignalPhaseExecution,
	}
	expectation, ok := diagnosticSignalActionFailureExpectation(
		Diagnostic(cli.DiagnosticSignalActionFailure),
		wrong,
	).(diagnosticExpectation)
	if !ok {
		t.Fatal("diagnosticSignalActionFailureExpectation must satisfy diagnosticExpectation")
	}
	got := expectation.evaluateDiagnosticReport(report)
	if got.status != evaluationDisproved {
		t.Fatalf("structured failure mismatch status = %v, want disproved (%#v)", got.status, got)
	}
	joined := renderEvaluationField(got.subject) + " " + renderEvaluationField(got.expected) + " " + renderEvaluationField(got.observed)
	for _, secret := range []string{
		"actual-secret-path",
		"actual-secret-action",
		"expected-secret-path",
		"expected-secret-action",
	} {
		if strings.Contains(joined, secret) {
			t.Fatalf("structured failure mismatch leaked %q in %q", secret, joined)
		}
	}
}

func TestDiagnosticSignalActionExpectationUsesExactSignalIdentity(t *testing.T) {
	t.Parallel()
	signals := testSemanticSignals(t, "actual", "expected")
	actualSignal := signals[0]
	expectedSignal := signals[1]
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{
			Code: cli.DiagnosticSignalActionFailure,
			SignalAction: &cli.SignalActionFailure{
				PolicyID: cli.SignalPolicyID(1),
				Index:    0,
				Action: cli.SignalAction{
					Kind: cli.SignalActionHandle,
				},
				Event: cli.SignalEvent{
					Signal: actualSignal,
					Phase:  cli.SignalPhaseExecution,
				},
				Phase: cli.SignalPhaseExecution,
			},
		}},
	}
	failure := *report.Diagnostics[0].SignalAction
	failure.Event.Signal = expectedSignal
	expectation, ok := diagnosticSignalActionFailureExpectation(
		Diagnostic(cli.DiagnosticSignalActionFailure),
		failure,
	).(diagnosticExpectation)
	if !ok {
		t.Fatal("diagnosticSignalActionFailureExpectation must satisfy diagnosticExpectation")
	}
	if got := expectation.evaluateDiagnosticReport(report); got.status != evaluationDisproved {
		t.Fatalf("distinct comparable signal identities = %#v, want disproved", got)
	}
}

func TestExpectationOutcomeKeepsProvedDisprovedAndNotProvableDistinct(t *testing.T) {
	t.Parallel()
	outcomes := []evaluationResult{
		provedResult(),
		disprovedResult(evaluationText("subject"), evaluationText("expected"), evaluationText("actual")),
		notProvableResult(evaluationText("subject"), evaluationText("expected"), evaluationText("partial observation")),
	}
	want := []evaluationStatus{evaluationProved, evaluationDisproved, evaluationNotProvable}
	for i := range outcomes {
		if outcomes[i].status != want[i] {
			t.Fatalf("outcome %d status = %v, want %v", i, outcomes[i].status, want[i])
		}
	}
}

func TestDiagnosticsConsumesExactRealPublicFailureReport(t *testing.T) {
	t.Parallel()
	definition := cli.Definition[struct{}]{
		Name: "diagnostic-integration",
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	request := cli.RunRequest{
		Args: []string{"--definitely-unknown"},
	}
	err = app.Run(context.Background(), request)
	if err == nil {
		t.Fatal("Run unexpectedly succeeded")
	}
	want := cli.Diagnostics(err)
	result := Diagnostics(err)
	got := result.Report()
	if len(got.Diagnostics) == 0 || len(got.Diagnostics) != len(want.Diagnostics) ||
		got.Truncated != want.Truncated || got.Diagnostics[0].Code != want.Diagnostics[0].Code {
		t.Fatalf("Diagnostics = %#v, want %#v", got, want)
	}
	code := got.Diagnostics[0].Code
	result.Require(t, Diagnostic(code).Present())

	// Mutating a separately extracted public report cannot change the retained
	// report owned by the result carrier.
	want.Diagnostics[0].Code = cli.DiagnosticCode("mutated")
	if again := result.Report(); len(again.Diagnostics) == 0 || again.Diagnostics[0].Code != code {
		t.Fatalf("retained diagnostic report changed: %#v", again)
	}
}

func TestTypedStaticPolicyExpectationsUseObservedRecordsAcrossOperations(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	subjectID := cli.ValueID(2)
	dependencyID := cli.ValueID(3)
	ruleCode := cli.RuleCode("required")

	ruleSchema := cli.RuleSchema{
		ID:           cli.RuleID(4),
		Code:         ruleCode,
		Subjects:     []cli.ValueID{subjectID},
		Dependencies: []cli.ValueID{dependencyID},
		Sensitive:    true,
	}
	ruleInspection := cli.RuleInspection{
		ID:           ruleSchema.ID,
		Code:         ruleCode,
		Subjects:     append([]cli.ValueID(nil), ruleSchema.Subjects...),
		Dependencies: append([]cli.ValueID(nil), ruleSchema.Dependencies...),
		Sensitive:    true,
	}

	schema := cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{
			commandID: {
				ID:    commandID,
				Rules: []cli.RuleSchema{ruleSchema},
			},
		},
	}
	reference := cli.Reference{
		Commands: map[cli.CommandID]cli.CommandReference{
			commandID: {
				ID:    commandID,
				Rules: []cli.RuleSchema{ruleSchema},
			},
		},
	}
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			commandID: {
				ID:    commandID,
				Rules: []cli.RuleInspection{ruleInspection},
			},
		},
	}

	rule := CommandByID(commandID).Rule(ruleCode)
	expectations := []DeclarationExpectation{
		rule.SubjectsExactly(ValueByID(subjectID)),
		rule.DependenciesExactly(ValueByID(dependencyID)),
		rule.Sensitive(),
	}
	for _, expectation := range expectations {
		(SchemaResult{state: &schemaResultState{observation: schema}}).Require(t, expectation)
		(ReferenceResult{state: &referenceResultState{observation: reference, err: nil}}).Require(t, expectation)
		(InspectionResult{state: &inspectionResultState{observation: inspection, err: nil}}).Require(t, expectation)
	}
}

func TestTypedStaticPolicyExactCollectionsRemainCompletenessAware(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	ruleCode := cli.RuleCode("required")
	expectation := CommandByID(commandID).Rule(ruleCode).SubjectsExactly(ValueByID(2))
	inspection := cli.Inspection{
		Truncated: true,
		Commands: map[cli.CommandID]cli.CommandInspection{
			commandID: {ID: commandID},
		},
	}

	checkedOutcome, ok := expectation.(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !ok {
		t.Fatal("expectation must satisfy expected evaluator interface")
	}
	outcome := checkedOutcome.evaluateInspection(inspection, nil)
	if outcome.status != evaluationNotProvable {
		t.Fatalf("truncated missing rule = %#v, want not provable", outcome)
	}
}

func TestExplicitFailureAndDiagnosticReportTruncationExpectationOwners(t *testing.T) {
	t.Parallel()
	err := invalidDeclarationFailure(t)
	explicit := FailsWith(Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
	(InspectionResult{state: &inspectionResultState{observation: cli.Inspection{}, err: err}}).Require(t, explicit)

	checkedMismatch, ok := FailsWith(Diagnostic(cli.DiagnosticUnknownFlag).Present()).(failureExpectationValue)
	if !ok {
		t.Fatal("FailsWith must satisfy failureExpectationValue")
	}
	mismatch := checkedMismatch.evaluateFailure(
		"inspection",
		err,
	)
	if mismatch.status != evaluationDisproved || renderEvaluationField(mismatch.observed) == "" {
		t.Fatalf("nested structured failure mismatch = %#v, want summarized disproval", mismatch)
	}

	DiagnosticsResult{
		state: &diagnosticsResultState{observation: cli.DiagnosticReport{
			Truncated: true,
		}},
	}.Require(t, DiagnosticsTruncated())
}

func TestDiagnosticCodeAdvisoryAndPresenceExpectations(t *testing.T) {
	t.Parallel()
	const advisory = cli.AdvisoryCode("warn")
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{
			Code:         cli.DiagnosticInvalidValue,
			AdvisoryCode: advisory,
		}},
	}

	checks := []DiagnosticReportExpectation{
		Diagnostic(cli.DiagnosticInvalidValue).Code(cli.DiagnosticInvalidValue),
		Diagnostic(cli.DiagnosticInvalidValue).Advisory(advisory).Present(),
		Diagnostic(cli.DiagnosticUnknownFlag).Absent(),
	}
	for index, expectation := range checks {
		checkedGot, ok := expectation.(diagnosticEvaluator)
		if !ok {
			t.Fatal("expectation must satisfy diagnosticEvaluator")
		}
		got := checkedGot.evaluateDiagnosticReport(report)
		if got.status != evaluationProved {
			t.Fatalf("diagnostic expectation %d = %#v", index, got)
		}
	}

	codeEvaluator, ok := Diagnostic(cli.DiagnosticInvalidValue).Code(
		cli.DiagnosticUnknownFlag,
	).(diagnosticEvaluator)
	if !ok {
		t.Fatal("Code must satisfy diagnosticEvaluator")
	}
	if got := codeEvaluator.evaluateDiagnosticReport(
		report,
	); got.status != evaluationDisproved {
		t.Fatalf("diagnostic code mismatch = %#v, want disproved", got)
	}
	advisoryEvaluator, ok := Diagnostic(cli.DiagnosticInvalidValue).Advisory(
		cli.AdvisoryCode("other"),
	).Present().(diagnosticEvaluator)
	if !ok {
		t.Fatal("Present must satisfy diagnosticEvaluator")
	}
	if got := advisoryEvaluator.evaluateDiagnosticReport(
		report,
	); got.status != evaluationDisproved {
		t.Fatalf("diagnostic advisory mismatch = %#v, want disproved", got)
	}
}

func TestFailureExpectationCoversCompletionAndReferenceOperations(t *testing.T) {
	t.Parallel()
	expectation, ok := FailsWith(Diagnostic(cli.DiagnosticInvalidDeclaration).Present()).(failureExpectationValue)
	if !ok {
		t.Fatal("FailsWith must satisfy failureExpectationValue")
	}
	if got := expectation.evaluateCompletion(
		cli.Completion{},
		nil,
	); got.status != evaluationDisproved ||
		renderEvaluationField(got.subject) != "completion" {
		t.Fatalf("completion failure adapter = %#v", got)
	}
	if got := expectation.evaluateReference(
		cli.Reference{},
		nil,
	); got.status != evaluationDisproved ||
		renderEvaluationField(got.subject) != "reference" {
		t.Fatalf("reference failure adapter = %#v", got)
	}
}
