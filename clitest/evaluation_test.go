package clitest

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func requireSchemaEvaluator(t *testing.T, expectation any) schemaEvaluator {
	t.Helper()
	evaluator, ok := expectation.(schemaEvaluator)
	if !ok {
		t.Fatalf("expectation %T does not implement schemaEvaluator", expectation)
	}
	return evaluator
}

func requireInspectionEvaluator(t *testing.T, expectation any) inspectionEvaluator {
	t.Helper()
	evaluator, ok := expectation.(inspectionEvaluator)
	if !ok {
		t.Fatalf("expectation %T does not implement inspectionEvaluator", expectation)
	}
	return evaluator
}

func requireReferenceEvaluator(t *testing.T, expectation any) referenceEvaluator {
	t.Helper()
	evaluator, ok := expectation.(referenceEvaluator)
	if !ok {
		t.Fatalf("expectation %T does not implement referenceEvaluator", expectation)
	}
	return evaluator
}

func TestReportExpectationOutcomesPreservesRedaction(t *testing.T) {
	t.Parallel()

	// Non-satisfied reportEvaluationResults paths intentionally fail testing.TB and are
	// owned by the subprocess failure-reporting case. Keep the passing arm local.
	reportEvaluationResults(t, []evaluationResult{provedResult()})
}

func TestExpectedOperationFailureRequiresFailsWith(t *testing.T) {
	t.Parallel()

	if expectsOperationFailure([]InspectionExpectation{InspectionReady()}) {
		t.Fatal("ready inspection reported explicit failure")
	}
	if expectsOperationFailure([]InspectionExpectation{InspectionRejected()}) {
		t.Fatal("InspectionRejected must not authorize an operation error")
	}
	if !expectsOperationFailure([]InspectionExpectation{failureExpectation(nil)}) {
		t.Fatal("FailsWith was not recognized as explicit failure")
	}
}

func TestDiagnosticsPreservesRedaction(t *testing.T) {
	t.Parallel()

	t.Run("invalid coordinates", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if recover() == nil {
				t.Fatal("nil diagnostics state did not panic")
			}
		}()
		(DiagnosticsResult{state: nil}).Require(t)
	})
	t.Run("valid expectation: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		(DiagnosticsResult{
			// Error-emitting nil/unsupported expectation branches are exercised by
			// the subprocess failure-reporting family; the normal evaluator path is
			// kept here to check direct dispatch without inventing a fake testing.TB.
			state: &diagnosticsResultState{observation: cli.DiagnosticReport{}},
		}).Require(t,

			Diagnostic(cli.DiagnosticInvalidValue).Absent())
	})
}

func TestExpectationReportingInspectionPreservesRedaction(t *testing.T) {
	t.Parallel()

	t.Run("invalid coordinates", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if recover() == nil {
				t.Fatal("nil inspection state did not panic")
			}
		}()
		(InspectionResult{state: nil}).Require(t)
	})
	t.Run("inspection: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		err := invalidDeclarationFailure(t)
		(InspectionResult{state: &inspectionResultState{observation: cli.Inspection{}, err: err}}).Require(t,

			FailsWith(Diagnostic(cli.DiagnosticInvalidDeclaration).Present()))
	})
	t.Run("inspection: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		(InspectionResult{state: &inspectionResultState{observation: cli.Inspection{}, err: nil}}).Require(t,

			inspectionStatusExpectation{
				status: cli.InspectionStatusUnknown,
			})
	})
}

func TestCompletionRequirementPreservesRedaction(t *testing.T) {
	t.Parallel()

	t.Run("invalid coordinates", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if recover() == nil {
				t.Fatal("nil completion state did not panic")
			}
		}()
		(CompletionResult{state: nil}).Require(t)
	})
	t.Run("operation success: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		err := invalidDeclarationFailure(t)
		(CompletionResult{state: &completionResultState{observation: cli.Completion{}, err: err}}).Require(t,

			FailsWith(Diagnostic(cli.DiagnosticInvalidDeclaration).Present()))
	})
	t.Run("implemented expectation: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		result := CompletionResult{state: &completionResultState{observation: cli.Completion{}, err: nil}}
		expectation := completionBooleanExpectation{
			field:   completionBooleanPartial,
			want:    false,
			subject: "partial",
		}
		result.Require(t, expectation)
	})
}

func TestExpectationReportingSchemaPreservesRedaction(t *testing.T) {
	t.Parallel()

	t.Run("invalid coordinates", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if recover() == nil {
				t.Fatal("nil schema state did not panic")
			}
		}()
		(SchemaResult{state: nil}).Require(t)
	})
	t.Run("schema: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		(SchemaResult{state: &schemaResultState{observation: cli.Schema{}}}).Require(t, releaseExpectationValue{
			release: "",
		})
	})
}

func TestExpectationReportingReferencePreservesRedaction(t *testing.T) {
	t.Parallel()

	t.Run("invalid coordinates", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if recover() == nil {
				t.Fatal("nil reference state did not panic")
			}
		}()
		(ReferenceResult{state: nil}).Require(t)
	})
	t.Run("reference: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		err := invalidDeclarationFailure(t)
		(ReferenceResult{state: &referenceResultState{observation: cli.Reference{}, err: err}}).Require(t,

			FailsWith(Diagnostic(cli.DiagnosticInvalidDeclaration).Present()))
	})
	t.Run("reference: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		(ReferenceResult{state: &referenceResultState{observation: cli.Reference{}, err: nil}}).Require(t,

			releaseExpectationValue{
				release: "",
			})
	})
}

func TestDiagnosticRulePreservesRedaction(t *testing.T) {
	t.Parallel()

	if got := diagnosticRule(DiagnosticLocator{}, "rule"); got.state != nil {
		t.Fatalf("rule refinement of zero locator = %#v", got.state)
	}
	base := Diagnostic(cli.DiagnosticRuleViolation)
	got := diagnosticRule(base, "rule")
	if got.state == nil || !got.state.ruleSet || got.state.rule != "rule" || base.state.ruleSet {
		t.Fatalf("rule refinement = base %#v got %#v", base.state, got.state)
	}
}

func TestDiagnosticOnPreservesRedaction(t *testing.T) {
	t.Parallel()

	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		base := Diagnostic(cli.DiagnosticInvalidValue)
		subjects := []DiagnosticSubject{
			CommandByID(1),
			CommandEntryByID(2),
			ActionEntryByID(3),
			FlagByID(4),
			FlagSpellingByID(5),
			ArgumentFormByID(6),
			ArgumentTermByID(7),
			EnvironmentInputByID(8),
			ValueDomainEntryByID(9),
			CommandDefaultByID(10),
			CommandProviderByID(11),
		}
		for _, subject := range subjects {
			got := diagnosticOn(base, subject)
			if got.state == nil || !got.state.subjectSet || got.state.subject == nil {
				t.Fatalf("diagnostic subject %T = %#v", subject, got.state)
			}
		}
		if got := diagnosticOn(DiagnosticLocator{}, CommandByID(1)); got.state != nil {
			t.Fatalf("zero diagnostic subject refinement = %#v", got.state)
		}
	})
	t.Run("invalid coordinates", func(t *testing.T) {
		t.Parallel()
		got := diagnosticOn(Diagnostic(cli.DiagnosticInvalidValue), CommandLocator{})
		if got.state == nil || !got.state.subjectSet {
			t.Fatalf("invalid subject = %#v", got.state)
		}
		resolved := resolveDiagnosticStateForReport(got.state, cli.DiagnosticReport{})
		if resolved.status != locatorResolutionInvalid {
			t.Fatalf("invalid subject resolution = %#v", resolved)
		}
	})
}

func TestDiagnosticRefinementsPreserveSemanticMatching(t *testing.T) {
	t.Parallel()

	locator := Diagnostic(cli.DiagnosticInvalidValue).
		Rule("required").
		Advisory("warn").
		On(CommandByID(1))
	matching := cli.Diagnostic{
		Code: cli.DiagnosticInvalidValue,
		Rule: &cli.RuleFailure{
			RuleCode: "required",
		},
		AdvisoryCode: "warn",
		Command:      1,
	}
	report := cli.DiagnosticReport{Diagnostics: []cli.Diagnostic{matching}}
	resolved := resolveDiagnosticStateForReport(locator.state, report)
	if resolved.status != locatorResolutionFound || resolved.value.index != 0 {
		t.Fatalf("matching diagnostic = %#v", resolved)
	}

	for _, diagnostic := range []cli.Diagnostic{
		{
			Code:         cli.DiagnosticInvalidValue,
			Rule:         &cli.RuleFailure{RuleCode: "other"},
			AdvisoryCode: "warn",
			Command:      1,
		},
		{
			Code:         cli.DiagnosticInvalidValue,
			Rule:         &cli.RuleFailure{RuleCode: "required"},
			AdvisoryCode: "other",
			Command:      1,
		},
		{
			Code:         cli.DiagnosticInvalidValue,
			Rule:         &cli.RuleFailure{RuleCode: "required"},
			AdvisoryCode: "warn",
			Command:      2,
		},
	} {
		report := cli.DiagnosticReport{Diagnostics: []cli.Diagnostic{diagnostic}}
		resolved := resolveDiagnosticStateForReport(locator.state, report)
		if resolved.status != locatorResolutionAbsent {
			t.Fatalf("mismatching diagnostic resolved: %#v", resolved)
		}
	}
}

func TestDiagnosticSubjectMatchingUsesTypedCoordinates(t *testing.T) {
	t.Parallel()

	diagnostic := cli.Diagnostic{
		CommandPath: []string{"serve"},
		Command:     1,
		Declaration: cli.DeclarationIdentity{
			Kind: cli.DeclarationKindFlag,
			Flag: 2,
		},
		Default:  3,
		Provider: 4,
	}
	cases := []struct {
		name    string
		subject DiagnosticSubject
		match   bool
	}{
		{name: "command", subject: CommandByID(1), match: true},
		{name: "command path", subject: Command("serve"), match: true},
		{name: "different command path", subject: Command("other"), match: false},
		{name: "declaration", subject: FlagByID(2), match: true},
		{name: "default", subject: CommandDefaultByID(3), match: true},
		{name: "provider", subject: CommandProviderByID(4), match: true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			resolved, status, _ := diagnosticSubjectForReport(test.subject)
			if status != locatorResolutionFound {
				t.Fatalf("subject resolution = %v", status)
			}
			if got := resolved.matches(diagnostic); got != test.match {
				t.Fatalf("match = %v, want %v", got, test.match)
			}
		})
	}
	if _, status, _ := diagnosticSubjectForReport(CommandLocator{}); status != locatorResolutionInvalid {
		t.Fatalf("zero command subject status = %v", status)
	}
}

func TestDiagnosticPropertyPreservesPropertyTruth(t *testing.T) {
	t.Parallel()

	expectation, ok := diagnosticCodeExpectation(
		Diagnostic(cli.DiagnosticInvalidValue),
		cli.DiagnosticUnknownFlag,
	).(diagnosticExpectation)
	if !ok {
		t.Fatal("diagnosticCodeExpectation must return diagnosticExpectation")
	}
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{Code: cli.DiagnosticInvalidValue}},
	}
	outcome := expectation.evaluateDiagnosticReport(report)
	if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) == "" {
		t.Fatalf("diagnostic property mismatch = %#v", outcome)
	}
}

func TestDiagnosticViolationPreservesRedaction(t *testing.T) {
	t.Parallel()

	expectation, ok := diagnosticViolationExpectation(
		Diagnostic(cli.DiagnosticInvalidValue),
		cli.ViolationKind(1),
	).(diagnosticExpectation)
	if !ok {
		t.Fatal("diagnosticViolationExpectation must return diagnosticExpectation")
	}
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{Code: cli.DiagnosticInvalidValue}},
	}
	outcome := expectation.evaluateDiagnosticReport(report)
	if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "diagnostic has no Rule failure" {
		t.Fatalf("missing Rule failure = %#v", outcome)
	}
}

func TestDiagnosticSpanPreservesRedaction(t *testing.T) {
	t.Parallel()

	expectation, ok := diagnosticSpanExpectation(
		Diagnostic(cli.DiagnosticInvalidValue),
		cli.InputSpan{},
	).(diagnosticExpectation)
	if !ok {
		t.Fatal("diagnosticSpanExpectation must return diagnosticExpectation")
	}
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{Code: cli.DiagnosticInvalidValue}},
	}
	outcome := expectation.evaluateDiagnosticReport(report)
	if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "diagnostic span is absent" {
		t.Fatalf("missing diagnostic span = %#v", outcome)
	}
}

func TestDiagnosticSignalActionFailurePreservesRedaction(t *testing.T) {
	t.Parallel()

	expectation, ok := diagnosticSignalActionFailureExpectation(
		Diagnostic(cli.DiagnosticSignalActionFailure),
		cli.SignalActionFailure{},
	).(diagnosticExpectation)
	if !ok {
		t.Fatal("diagnosticSignalActionFailureExpectation must return diagnosticExpectation")
	}
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{Code: cli.DiagnosticSignalActionFailure}},
	}
	outcome := expectation.evaluateDiagnosticReport(report)
	if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "structured failure detail is absent" {
		t.Fatalf("missing action failure = %#v", outcome)
	}
}

func TestDiagnosticSignalOverflowFailurePreservesRedaction(t *testing.T) {
	t.Parallel()

	expected := cli.SignalOverflowFailure{
		Capacity: 1,
	}
	expectation, ok := diagnosticSignalOverflowFailureExpectation(
		Diagnostic(cli.DiagnosticSignalOverflow),
		expected,
	).(diagnosticExpectation)
	if !ok {
		t.Fatal("diagnosticSignalOverflowFailureExpectation must return diagnosticExpectation")
	}
	t.Run("structured failure detail is absent: semantic absence", func(t *testing.T) {
		t.Parallel()
		report := cli.DiagnosticReport{
			Diagnostics: []cli.Diagnostic{{Code: cli.DiagnosticSignalOverflow}},
		}
		outcome := expectation.evaluateDiagnosticReport(report)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "structured failure detail is absent" {
			t.Fatalf("missing overflow failure = %#v", outcome)
		}
	})
	t.Run("different structured failure detail: semantic result", func(t *testing.T) {
		t.Parallel()
		actual := cli.SignalOverflowFailure{
			Capacity: 2,
		}
		report := cli.DiagnosticReport{
			Diagnostics: []cli.Diagnostic{{
				Code:           cli.DiagnosticSignalOverflow,
				SignalOverflow: &actual,
			}},
		}
		outcome := expectation.evaluateDiagnosticReport(report)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "different structured failure detail" {
			t.Fatalf("different overflow failure = %#v", outcome)
		}
	})
}

func TestDiagnosticNoticeDeliveryFailurePreservesRedaction(t *testing.T) {
	t.Parallel()

	expectation, ok := diagnosticNoticeDeliveryFailureExpectation(
		Diagnostic(cli.DiagnosticNoticeDeliveryFailure),
		cli.NoticeDeliveryFailure{},
	).(diagnosticExpectation)
	if !ok {
		t.Fatal("diagnosticNoticeDeliveryFailureExpectation must return diagnosticExpectation")
	}
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{Code: cli.DiagnosticNoticeDeliveryFailure}},
	}
	outcome := expectation.evaluateDiagnosticReport(report)
	if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "structured failure detail is absent" {
		t.Fatalf("missing Notice delivery failure = %#v", outcome)
	}
}

func TestDiagnosticPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	t.Run("diagnostic subject: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		outcome := diagnosticExpectation{}.evaluateDiagnosticReport(cli.DiagnosticReport{})
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "invalid diagnostic locator" {
			t.Fatalf("zero diagnostic expectation = %#v", outcome)
		}
	})
	t.Run("absent: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		expectation, ok := diagnosticAbsentExpectation(
			Diagnostic(cli.DiagnosticInvalidValue),
		).(diagnosticExpectation)
		if !ok {
			t.Fatal("diagnosticAbsentExpectation must return diagnosticExpectation")
		}
		report := cli.DiagnosticReport{
			Diagnostics: []cli.Diagnostic{{Code: cli.DiagnosticInvalidValue}},
		}
		outcome := expectation.evaluateDiagnosticReport(report)
		if outcome.status != evaluationDisproved {
			t.Fatalf("present diagnostic under absence expectation = %#v", outcome)
		}
	})
	t.Run("diagnostic report is truncated and subject is not present: semantic absence", func(t *testing.T) {
		t.Parallel()
		expectation, ok := diagnosticPresentExpectation(
			Diagnostic(cli.DiagnosticInvalidValue),
		).(diagnosticExpectation)
		if !ok {
			t.Fatal("diagnosticPresentExpectation must return diagnosticExpectation")
		}
		outcome := expectation.evaluateDiagnosticReport(cli.DiagnosticReport{
			Truncated: true,
		})
		if outcome.status != evaluationNotProvable {
			t.Fatalf("truncated missing diagnostic = %#v", outcome)
		}
	})
	t.Run(
		"one observed diagnostic code match proves positive evidence under truncation",
		func(t *testing.T) {
			t.Parallel()
			expectation, ok := diagnosticCodeExpectation(
				Diagnostic(cli.DiagnosticInvalidValue),
				cli.DiagnosticInvalidValue,
			).(diagnosticExpectation)
			if !ok {
				t.Fatal("diagnosticCodeExpectation must return diagnosticExpectation")
			}
			report := cli.DiagnosticReport{
				Diagnostics: []cli.Diagnostic{{Code: cli.DiagnosticInvalidValue}},
				Truncated:   true,
			}
			outcome := expectation.evaluateDiagnosticReport(report)
			if outcome.status != evaluationProved {
				t.Fatalf("truncated code locator = %#v", outcome)
			}
		},
	)
}

func TestEqualSignalActionFailurePreservesRedaction(t *testing.T) {
	t.Parallel()

	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		base := cli.SignalActionFailure{
			CommandPath: []string{"root"},
			PolicyID:    1,
			Index:       2,
			Phase:       cli.SignalPhaseExecution,
			Action: cli.SignalAction{
				Signal: testSemanticSignal(t),
			},
			Event: cli.SignalEvent{
				Signal: testSemanticSignal(t),
			},
		}
		if !equalSignalActionFailure(base, cloneSignalActionFailure(base)) {
			t.Fatal("equal signal action failure rejected")
		}
		other := cloneSignalActionFailure(base)
		other.Index++
		if equalSignalActionFailure(base, other) {
			t.Fatal("different signal action failure matched")
		}
		sender := cli.SignalSender{
			ProcessID: 1,
		}
		base.Event.Sender = &sender
		other = cloneSignalActionFailure(base)
		other.Event.Sender.ProcessID = 2
		if equalSignalActionFailure(base, other) {
			t.Fatal("different signal sender matched")
		}
	})
}

func TestEqualSignalIdentityPreservesRedaction(t *testing.T) {
	t.Parallel()

	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		signals := testSemanticSignals(t, "first", "second")
		if !equalSignalIdentity(cli.Signal{}, cli.Signal{}) {
			t.Fatal("zero signal identity did not equal itself")
		}
		if !equalSignalIdentity(signals[0], signals[0]) {
			t.Fatal("identical semantic signal rejected")
		}
		if equalSignalIdentity(signals[0], signals[1]) {
			t.Fatal("distinct semantic signal identities matched")
		}
	})
}

func TestDiagnosticsTruncatedPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	outcome := diagnosticsTruncatedExpectation{}.evaluateDiagnosticReport(cli.DiagnosticReport{})
	if outcome.status != evaluationDisproved {
		t.Fatalf("complete diagnostics = %#v", outcome)
	}
}

func TestCompletionExpectationPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	outcome := diagnosticsTruncatedExpectation{}.evaluateCompletion(cli.Completion{}, nil)
	if outcome.status != evaluationDisproved {
		t.Fatalf("complete completion diagnostics = %#v", outcome)
	}
}

func TestDiagnosticsExactlyPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		expectation := diagnosticsExactlyExpectationValue{locators: []*diagnosticLocatorState{Diagnostic(cli.DiagnosticInvalidValue).state}}
		outcome := expectation.evaluateDiagnosticReport(cli.DiagnosticReport{})
		if outcome.status != evaluationDisproved {
			t.Fatalf("diagnostic cardinality mismatch = %#v", outcome)
		}
	})
	t.Run("invalid coordinates", func(t *testing.T) {
		t.Parallel()
		expectation := diagnosticsExactlyExpectationValue{locators: []*diagnosticLocatorState{nil}}
		report := cli.DiagnosticReport{Diagnostics: []cli.Diagnostic{{}}}
		outcome := expectation.evaluateDiagnosticReport(report)
		if outcome.status != evaluationDisproved {
			t.Fatalf("nil exact diagnostic locator = %#v", outcome)
		}
	})
	t.Run("diagnostic index: semantic result", func(t *testing.T) {
		t.Parallel()
		expectation := diagnosticsExactlyExpectationValue{locators: []*diagnosticLocatorState{DiagnosticAt(0).state}}
		report := cli.DiagnosticReport{Diagnostics: []cli.Diagnostic{{Code: cli.DiagnosticInvalidValue}}}
		outcome := expectation.evaluateDiagnosticReport(report)
		if outcome.status != evaluationProved {
			t.Fatalf("diagnostic index exactness = %#v", outcome)
		}
	})
	t.Run("refined semantic result", func(t *testing.T) {
		t.Parallel()
		diagnostic := cli.Diagnostic{
			Code:         cli.DiagnosticInvalidValue,
			Rule:         &cli.RuleFailure{RuleCode: "rule"},
			AdvisoryCode: "advisory",
			Command:      1,
		}
		locator := Diagnostic(cli.DiagnosticInvalidValue).Rule("rule").Advisory("advisory").On(CommandByID(1))
		report := cli.DiagnosticReport{Diagnostics: []cli.Diagnostic{diagnostic}}
		outcome := (diagnosticsExactlyExpectationValue{locators: []*diagnosticLocatorState{locator.state}}).evaluateDiagnosticReport(report)
		if outcome.status != evaluationProved {
			t.Fatalf("refined exact diagnostic = %#v", outcome)
		}
	})
	t.Run("refined semantic mismatch", func(t *testing.T) {
		t.Parallel()
		locator := Diagnostic(cli.DiagnosticInvalidValue).Rule("other")
		report := cli.DiagnosticReport{Diagnostics: []cli.Diagnostic{{Code: cli.DiagnosticInvalidValue, Rule: &cli.RuleFailure{RuleCode: "rule"}}}}
		outcome := (diagnosticsExactlyExpectationValue{locators: []*diagnosticLocatorState{locator.state}}).evaluateDiagnosticReport(report)
		if outcome.status != evaluationDisproved {
			t.Fatalf("refined exact diagnostic mismatch = %#v", outcome)
		}
	})
}

func TestSummarizeOutcomesPreservesRedaction(t *testing.T) {
	t.Parallel()
	value := []evaluationResult{provedResult(), disprovedResult(evaluationText("first"), evaluationText("x"), evaluationText("a")), notProvableResult(evaluationText("second"), evaluationText("y"), evaluationText("b"))}
	got := renderEvaluationField(summarizeOutcomes(value))
	if got != "first: a; second: b" {
		t.Fatalf("summary = %q", got)
	}
}

func TestDiagnosticAdvisoryRejectsZeroLocator(t *testing.T) {
	t.Parallel()
	if got := diagnosticAdvisory(DiagnosticLocator{}, "warn"); got.state != nil {
		t.Fatalf("zero diagnostic locator produced state %#v", got.state)
	}
}
