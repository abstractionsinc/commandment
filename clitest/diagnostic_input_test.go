package clitest

import (
	"context"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestNoticeInspectionExpectations(t *testing.T) {
	t.Parallel()
	command := cli.CommandID(1)
	flag := cli.FlagID(2)
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			command: {
				ID:   command,
				Path: []string{"root"},
				NoticeSuppressions: []cli.NoticeSuppressionInspection{{
					Command:          command,
					DeclarationIndex: 2,
				}},
			},
		},
		Notices: []cli.NoticeInspection{
			{
				Code:         cli.NoticeAdvisory,
				AdvisoryCode: cli.AdvisoryCode("warn"),
				Severity:     cli.NoticeSeverityWarning,
				Command:      command,
				Suppression:  cli.NoticeSuppressionSuppressed,
				SuppressedBy: []cli.NoticeSuppressionLocator{{
					Command:          command,
					DeclarationIndex: 2,
				}},
			},
			{
				Code:     cli.NoticeDeprecatedFlag,
				Severity: cli.NoticeSeverityWarning,
				Declaration: cli.DeclarationIdentity{
					Kind: cli.DeclarationKindFlag,
					Flag: flag,
				},
				Sensitive: true,
				Replacement: &cli.DeclarationReplacement{
					Target: cli.DeclarationIdentity{
						Kind: cli.DeclarationKindFlag,
						Flag: cli.FlagID(3),
					},
				},
				RepairIndexes: []int{0},
			},
		},
	}
	first := Notice(cli.NoticeAdvisory).Advisory(cli.AdvisoryCode("warn")).On(CommandByID(command))
	second := Notice(cli.NoticeDeprecatedFlag).On(FlagByID(flag))
	(InspectionResult{state: &inspectionResultState{observation: inspection, err: nil}}).Require(t,
		first.Present(),
		first.Code(cli.NoticeAdvisory),
		first.Severity(cli.NoticeSeverityWarning),
		first.Suppression(cli.NoticeSuppressionSuppressed),
		first.SuppressedByExactly(CommandByID(command).NoticeSuppressionAt(0)),
		second.Present(),
		second.Sensitive(),
		second.ReplacedBy(FlagByID(cli.FlagID(3))),
		second.RepairCount(1),
		NoticesExactly(first, second),
		Notice(cli.NoticeSignalHint).Absent())
}

func TestOutcomeDiagnosticExpectationsUseOperationOwnedDiagnosticCompleteness(t *testing.T) {
	t.Parallel()
	expectation, ok := DiagnosticsExactly(Diagnostic(cli.DiagnosticInvalidValue)).(diagnosticsExactlyExpectationValue)
	if !ok {
		t.Fatal("DiagnosticsExactly must satisfy diagnosticsExactlyExpectationValue")
	}
	inspection := cli.Inspection{
		Diagnostics: []cli.Diagnostic{{Code: cli.DiagnosticInvalidValue}},
	}
	if got := expectation.evaluateInspection(inspection, nil); got.status != evaluationProved {
		t.Fatalf("inspection diagnostics = %#v", got)
	}
	inspection.Truncated = true
	if got := expectation.evaluateInspection(inspection, nil); got.status != evaluationNotProvable {
		t.Fatalf("truncated inspection diagnostics = %#v, want not provable", got)
	}

	completion := cli.Completion{
		Diagnostics:          []cli.Diagnostic{{Code: cli.DiagnosticInvalidValue}},
		DiagnosticsTruncated: true,
	}
	if got := expectation.evaluateCompletion(completion, nil); got.status != evaluationNotProvable {
		t.Fatalf("truncated completion diagnostics = %#v, want not provable", got)
	}
	checkedGot, ok := DiagnosticsTruncated().(diagnosticsTruncatedExpectation)
	if !ok {
		t.Fatal("DiagnosticsTruncated must satisfy diagnosticsTruncatedExpectation")
	}
	if got := checkedGot.evaluateCompletion(
		completion,
		nil,
	); got.status != evaluationProved {
		t.Fatalf("completion diagnostic truncation = %#v", got)
	}
}

func TestDiagnosticRepairAndInputGraphExpectations(t *testing.T) {
	t.Parallel()
	var sourceID cli.InputSourceID
	var originID cli.InputOriginID
	span := cli.InputSpan{
		Source: sourceID,
		Entry:  1,
		Start: cli.InputPosition{
			Offset: 2,
			Line:   1,
			Column: 3,
		},
		End: cli.InputPosition{
			Offset: 5,
			Line:   1,
			Column: 6,
		},
	}
	providerOutput := cli.CommandProviderOutputLocator{
		Index:    2,
		Kind:     cli.CommandEntryAlias,
		Key:      "provider",
		Location: "remote",
	}
	replacement := &cli.DeclarationReplacement{
		Target: cli.DeclarationIdentity{
			Kind: cli.DeclarationKindFlag,
			Flag: 9,
		},
	}
	actualTextReplacement := cli.InputReplacement{{Literal: "fixed"}, {Copy: &span}}
	actualEntryReplacement := cli.InputReplacement{{Literal: "entry"}}
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{
			Code: cli.DiagnosticInvalidValue,
			Rule: &cli.RuleFailure{
				ViolationKind: cli.ViolationInvalid,
			},
			InputSource:    sourceID,
			Span:           &span,
			Origins:        []cli.InputOriginID{originID},
			ProviderOutput: &providerOutput,
			Replacement:    replacement,
			Sensitive:      true,
			ExitCode:       17,
			ProducerKind:   cli.ValueProducerKindFlag,
			Tier:           cli.ResolutionTierExplicit,
			Disposition:    cli.ValueCandidateDispositionEffective,
			RepairIndexes:  []int{0},
		}},
		Sources: map[cli.InputSourceID]cli.InputSource{
			sourceID: {
				ID:              sourceID,
				Kind:            cli.InputKindShortcut,
				CommandShortcut: 7,
				Key:             "argv",
				Location:        "argument 1",
				Sensitive:       true,
			},
		},
		Origins: map[cli.InputOriginID]cli.InputOrigin{
			originID: {
				ID:        originID,
				Span:      span,
				Parents:   []cli.InputOriginID{originID},
				Sensitive: true,
			},
		},
		Repairs: []cli.InputRepair{{
			Title:            "replace input",
			PreservesMeaning: true,
			Sensitive:        true,
			Sources: []cli.InputRepairSource{{
				Source: sourceID,
				TextEdits: []cli.InputTextEdit{{
					Span:        span,
					Replacement: actualTextReplacement,
				}},
				EntryEdits: []cli.InputEntryEdit{{
					Start:   1,
					End:     2,
					Entries: []cli.InputReplacement{actualEntryReplacement},
				}},
			}},
		}},
	}

	diagnostic := DiagnosticAt(0)
	source := InputSourceByID(sourceID)
	origin := InputOriginByID(originID)
	repair := diagnostic.RepairAt(0)
	repairSource := repair.Source(source)
	textEdit := repairSource.TextEditAt(0)
	entryEdit := repairSource.EntryEditAt(0)

	expectedCopySpan := span
	expectedTextReplacement := cli.InputReplacement{{Literal: "fixed"}, {Copy: &expectedCopySpan}}
	expectedEntryReplacements := []cli.InputReplacement{{{Literal: "entry"}}}
	textReplacementExpectation := textEdit.Replacement(expectedTextReplacement)
	entryReplacementExpectation := entryEdit.EntriesExactly(expectedEntryReplacements...)
	expectedTextReplacement[0].Literal = "mutated"
	expectedTextReplacement[1].Copy.Start.Offset = 999
	expectedEntryReplacements[0][0].Literal = "mutated"
	(DiagnosticsResult{state: &diagnosticsResultState{observation: report}}).Require(t,
		diagnostic.Present(),
		diagnostic.Violation(cli.ViolationInvalid),
		diagnostic.ProducerKind(cli.ValueProducerKindFlag),
		diagnostic.Tier(cli.ResolutionTierExplicit),
		diagnostic.Disposition(cli.ValueCandidateDispositionEffective),
		diagnostic.ExitCode(17),
		diagnostic.Sensitive(),
		diagnostic.InputSource(source),
		diagnostic.Span(span),
		diagnostic.ProviderOutput(providerOutput),
		diagnostic.OriginsExactly(origin),
		diagnostic.ReplacedBy(FlagByID(9)),
		diagnostic.RepairCount(1),
		repair.Present(),
		repair.PreservesMeaning(),
		repair.Sensitive(),
		repair.ProducerKindsExactly(source),
		RepairAt(0).Present(),
		RepairAt(0).SourceAt(0).TextEditCount(1),
		repairSource.TextEditCount(1),
		repairSource.EntryEditCount(1),
		textEdit.Span(span),
		textReplacementExpectation,
		entryEdit.Range(1, 2),
		entryReplacementExpectation,
		source.Present(),
		source.Kind(cli.InputKindShortcut),
		source.Sensitive(),
		source.Key("argv"),
		source.Location("argument 1"),
		source.CommandShortcut(CommandEntryByID(7)),
		origin.Present(),
		origin.Sensitive(),
		origin.ParentsExactly(origin),
		origin.ParentAt(0).Present(),
		origin.Span(span),
		origin.Source().Present())

	checkedMismatch, ok := source.Key("wrong").(diagnosticEvaluator)
	if !ok {
		t.Fatal("Key must satisfy diagnosticEvaluator")
	}
	mismatch := checkedMismatch.evaluateDiagnosticReport(report)
	if mismatch.status != evaluationDisproved || renderEvaluationField(mismatch.observed) != "value differs (redacted)" {
		t.Fatalf("sensitive source mismatch = %#v, want redacted disproval", mismatch)
	}
}

func TestRepairSourceBySourcePreservesRedaction(t *testing.T) {
	t.Parallel()

	locator := repairSourceBySource(RepairAt(0), InputSourceLocator{})
	resolved := resolveRepairSource(locator.state, cli.DiagnosticReport{})
	if resolved.status != locatorResolutionInvalid {
		t.Fatalf("invalid source locator = %#v", resolved)
	}
}

func TestOriginSourceLocatorPreservesRedaction(t *testing.T) {
	t.Parallel()

	if got := originSourceLocator(InputOriginLocator{}); got.state != nil {
		t.Fatalf("invalid origin source locator = %#v", got.state)
	}
}

func TestDiagnosticParentResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	t.Run("invalid diagnostic premise: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		resolved := resolveDiagnosticStateForReport(nil, cli.DiagnosticReport{})
		if resolved.status != locatorResolutionInvalid {
			t.Fatalf("nil diagnostic premise = %v, want invalid", resolved.status)
		}
	})
	t.Run("ambiguous identity", func(t *testing.T) {
		t.Parallel()
		state := Diagnostic(cli.DiagnosticInvalidValue).state
		report := cli.DiagnosticReport{
			Diagnostics: []cli.Diagnostic{{Code: cli.DiagnosticInvalidValue}, {Code: cli.DiagnosticInvalidValue}},
		}
		resolved := resolveDiagnosticStateForReport(state, report)
		if resolved.status != locatorResolutionAmbiguous {
			t.Fatalf("ambiguous diagnostic = %v", resolved.status)
		}
	})
	t.Run("one observed diagnostic match resolves under truncation", func(t *testing.T) {
		t.Parallel()
		state := Diagnostic(cli.DiagnosticInvalidValue).state
		report := cli.DiagnosticReport{
			Diagnostics: []cli.Diagnostic{{Code: cli.DiagnosticInvalidValue}},
			Truncated:   true,
		}
		resolved := resolveDiagnosticStateForReport(state, report)
		if resolved.status != locatorResolutionFound {
			t.Fatalf("truncated diagnostic identity = %v %q", resolved.status, renderEvaluationField(resolved.detail))
		}
	})
	t.Run("incomplete observation", func(t *testing.T) {
		t.Parallel()
		state := Diagnostic(cli.DiagnosticInvalidValue).state
		resolved := resolveDiagnosticStateForReport(state, cli.DiagnosticReport{Truncated: true})
		if resolved.status != locatorResolutionNotProvable {
			t.Fatalf("truncated missing diagnostic = %v", resolved.status)
		}
	})
	t.Run("diagnostic report is truncated and premise is not present: incomplete observation", func(t *testing.T) {
		t.Parallel()
		state := Diagnostic(cli.DiagnosticInvalidValue).state
		resolved := resolveDiagnosticStateForReport(state, cli.DiagnosticReport{Truncated: true})
		if resolved.status != locatorResolutionNotProvable ||
			renderEvaluationField(resolved.detail) != "diagnostic report is truncated and premise is not present" {
			t.Fatalf("truncated missing diagnostic = %v %q", resolved.status, renderEvaluationField(resolved.detail))
		}
	})
	t.Run("diagnostic premise is absent: semantic absence", func(t *testing.T) {
		t.Parallel()
		state := Diagnostic(cli.DiagnosticInvalidValue).state
		resolved := resolveDiagnosticStateForReport(state, cli.DiagnosticReport{})
		if resolved.status != locatorResolutionAbsent || renderEvaluationField(resolved.detail) != "diagnostic premise is absent" {
			t.Fatalf("absent diagnostic premise = %v %q", resolved.status, renderEvaluationField(resolved.detail))
		}
	})
}

func TestRepairResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	t.Run("repair: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		got := resolveRepair(nil, cli.DiagnosticReport{})
		if got.status != locatorResolutionInvalid || renderEvaluationField(got.actual) != "invalid repair locator" {
			t.Fatalf("invalid repair = %#v", got)
		}
	})
	t.Run("repair index: incomplete observation", func(t *testing.T) {
		t.Parallel()
		got := resolveRepair(RepairAt(1).state, cli.DiagnosticReport{
			Truncated: true,
		})
		if got.status != locatorResolutionNotProvable || renderEvaluationField(got.actual) != "diagnostic report is truncated" {
			t.Fatalf("truncated repair = %#v", got)
		}
	})
	t.Run("repair index: semantic absence", func(t *testing.T) {
		t.Parallel()
		got := resolveRepair(RepairAt(1).state, cli.DiagnosticReport{})
		if got.status != locatorResolutionAbsent || renderEvaluationField(got.actual) != "repair is absent" {
			t.Fatalf("absent repair = %#v", got)
		}
	})
	t.Run("repair diagnostic index: semantic absence", func(t *testing.T) {
		t.Parallel()
		state := Diagnostic(cli.DiagnosticInvalidValue).RepairAt(0).state
		got := resolveRepair(state, cli.DiagnosticReport{})
		if got.status != locatorResolutionAbsent {
			t.Fatalf("absent diagnostic relation = %#v", got)
		}
		report := cli.DiagnosticReport{
			Diagnostics: []cli.Diagnostic{{Code: cli.DiagnosticInvalidValue}},
		}
		got = resolveRepair(state, report)
		if got.status != locatorResolutionAbsent || renderEvaluationField(got.actual) != "diagnostic repair relationship is absent" {
			t.Fatalf("missing repair edge = %#v", got)
		}
	})
	t.Run("repair diagnostic index: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		got := resolveRepair(&repairLocatorState{}, cli.DiagnosticReport{})
		if got.status != locatorResolutionInvalid {
			t.Fatalf("wrong repair kind = %#v", got)
		}
	})
}

func TestRepairSourceResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	var sourceID cli.InputSourceID
	repair := cli.InputRepair{
		Sources: []cli.InputRepairSource{{Source: sourceID}, {Source: sourceID}},
	}
	report := cli.DiagnosticReport{
		Repairs: []cli.InputRepair{repair},
		Sources: map[cli.InputSourceID]cli.InputSource{sourceID: {ID: sourceID}},
	}
	t.Run("repair source: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		got := resolveRepairSource(nil, report)
		if got.status != locatorResolutionInvalid {
			t.Fatalf("invalid repair source = %#v", got)
		}
	})
	t.Run("expected subject unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		state := &repairSourceLocatorState{
			mode:   repairSourceLocatorIndex,
			repair: RepairAt(9).state,
			index:  0,
		}
		got := resolveRepairSource(state, report)
		if got.status != locatorResolutionAbsent {
			t.Fatalf("unresolved repair parent = %#v", got)
		}
	})
	t.Run("repair source index: semantic absence", func(t *testing.T) {
		t.Parallel()
		state := RepairAt(0).SourceAt(9).state
		got := resolveRepairSource(state, report)
		if got.status != locatorResolutionAbsent || renderEvaluationField(got.actual) != "repair source is absent" {
			t.Fatalf("absent repair source = %#v", got)
		}
	})
	t.Run("repair source id: matching observation", func(t *testing.T) {
		t.Parallel()
		state := repairSourceBySource(RepairAt(0), InputSourceByID(sourceID)).state
		missing := report
		missing.Sources = nil
		got := resolveRepairSource(state, missing)
		if got.status != locatorResolutionAbsent || renderEvaluationField(got.actual) == "" {
			t.Fatalf("unresolved expected source = %#v", got)
		}
	})
	t.Run("repair source id: ambiguous identity", func(t *testing.T) {
		t.Parallel()
		state := repairSourceBySource(RepairAt(0), InputSourceByID(sourceID)).state
		got := resolveRepairSource(state, report)
		if got.status != locatorResolutionAmbiguous || renderEvaluationField(got.actual) != "matches repair source indexes; count=2" {
			t.Fatalf("ambiguous repair source = %#v", got)
		}
	})
	t.Run("repair source id: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		got := resolveRepairSource(&repairSourceLocatorState{repair: RepairAt(0).state}, report)
		if got.status != locatorResolutionInvalid {
			t.Fatalf("wrong repair source kind = %#v", got)
		}
	})
}

//nolint:dupl // Text-edit and entry-edit coordinates intentionally prove the same closure law through distinct typed resolvers.
func TestTextEditResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()
	report := cli.DiagnosticReport{Repairs: []cli.InputRepair{{Sources: []cli.InputRepairSource{{}}}}}

	t.Run("text edit: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		got := resolveTextEdit(nil, report)
		if got.status != locatorResolutionInvalid {
			t.Fatalf("invalid text edit status = %v", got.status)
		}
	})
	t.Run("text edit: semantic absence", func(t *testing.T) {
		t.Parallel()
		locator := RepairAt(0).SourceAt(0).TextEditAt(0)
		got := resolveTextEdit(locator.state, report)
		if got.status != locatorResolutionAbsent || renderEvaluationField(got.actual) != "text edit is absent" {
			t.Fatalf("absent text edit = %#v", got)
		}

		locator.state.source = RepairAt(9).SourceAt(0).state
		got = resolveTextEdit(locator.state, report)
		if got.status != locatorResolutionAbsent {
			t.Fatalf("unresolved text-edit source = %#v", got)
		}
	})
}

//nolint:dupl // Text-edit and entry-edit coordinates intentionally prove the same closure law through distinct typed resolvers.
func TestEntryEditResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()
	report := cli.DiagnosticReport{Repairs: []cli.InputRepair{{Sources: []cli.InputRepairSource{{}}}}}

	t.Run("entry edit: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		got := resolveEntryEdit(nil, report)
		if got.status != locatorResolutionInvalid {
			t.Fatalf("invalid entry edit status = %v", got.status)
		}
	})
	t.Run("entry edit: semantic absence", func(t *testing.T) {
		t.Parallel()
		locator := RepairAt(0).SourceAt(0).EntryEditAt(0)
		got := resolveEntryEdit(locator.state, report)
		if got.status != locatorResolutionAbsent || renderEvaluationField(got.actual) != "entry edit is absent" {
			t.Fatalf("absent entry edit = %#v", got)
		}

		locator.state.source = RepairAt(9).SourceAt(0).state
		got = resolveEntryEdit(locator.state, report)
		if got.status != locatorResolutionAbsent {
			t.Fatalf("unresolved entry-edit source = %#v", got)
		}
	})
}

func TestSourceResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	var sourceID cli.InputSourceID
	var originID cli.InputOriginID
	t.Run("input source: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		for _, state := range []*inputSourceLocatorState{nil, {}} {
			got := resolveSource(state, cli.DiagnosticReport{})
			if got.status != locatorResolutionInvalid || renderEvaluationField(got.actual) != "invalid input source locator" {
				t.Fatalf("invalid source = %#v", got)
			}
		}
	})
	t.Run("expected subject unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		state := originSourceLocator(InputOriginByID(originID)).state
		got := resolveSource(state, cli.DiagnosticReport{})
		if got.status != locatorResolutionAbsent {
			t.Fatalf("unresolved origin source = %#v", got)
		}
	})
	t.Run("incomplete observation", func(t *testing.T) {
		t.Parallel()
		got := resolveSource(InputSourceByID(sourceID).state, cli.DiagnosticReport{
			Truncated: true,
		})
		if got.status != locatorResolutionNotProvable {
			t.Fatalf("truncated source = %#v", got)
		}
	})
	t.Run("diagnostic report is truncated: incomplete observation", func(t *testing.T) {
		t.Parallel()
		got := resolveSource(InputSourceByID(sourceID).state, cli.DiagnosticReport{
			Truncated: true,
		})
		if renderEvaluationField(got.actual) != "diagnostic report is truncated" {
			t.Fatalf("truncated source detail = %#v", got)
		}
	})
	t.Run("input source is absent: semantic absence", func(t *testing.T) {
		t.Parallel()
		got := resolveSource(InputSourceByID(sourceID).state, cli.DiagnosticReport{})
		if got.status != locatorResolutionAbsent || renderEvaluationField(got.actual) != "input source is absent" {
			t.Fatalf("absent source = %#v", got)
		}
	})
}

func TestOriginResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	var originID cli.InputOriginID
	t.Run("input origin: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		got := resolveOrigin(nil, cli.DiagnosticReport{})
		if got.status != locatorResolutionInvalid {
			t.Fatalf("invalid origin = %#v", got)
		}
	})
	t.Run("input origin parent index: semantic absence", func(t *testing.T) {
		t.Parallel()
		state := InputOriginByID(originID).ParentAt(0).state
		got := resolveOrigin(state, cli.DiagnosticReport{})
		if got.status != locatorResolutionAbsent {
			t.Fatalf("missing origin parent = %#v", got)
		}
		report := cli.DiagnosticReport{
			Origins: map[cli.InputOriginID]cli.InputOrigin{originID: {ID: originID}},
		}
		got = resolveOrigin(state, report)
		if got.status != locatorResolutionAbsent || renderEvaluationField(got.actual) != "parent origin is absent" {
			t.Fatalf("missing parent edge = %#v", got)
		}
		got = resolveOrigin(InputOriginByID(originID).state, cli.DiagnosticReport{})
		if got.status != locatorResolutionAbsent || renderEvaluationField(got.actual) != "input origin is absent" {
			t.Fatalf("absent origin = %#v", got)
		}
	})
	t.Run("input origin parent index: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		got := resolveOrigin(&inputOriginLocatorState{}, cli.DiagnosticReport{})
		if got.status != locatorResolutionInvalid {
			t.Fatalf("wrong origin kind = %#v", got)
		}
	})
}

func TestExpectedOriginsResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	states := []*inputOriginLocatorState{InputOriginByID(cli.InputOriginID{}).state}
	ids, outcome, ok := resolveExpectedOrigins(states, cli.DiagnosticReport{}, nil)
	if ok || ids != nil || outcome.status != evaluationDisproved {
		t.Fatalf("unresolved expected origins = ids %#v outcome %#v ok %v", ids, outcome, ok)
	}
}

func TestRepairPresentPreservesRedaction(t *testing.T) {
	t.Parallel()

	checkedOutcome, ok := repairPresentExpectation(RepairAt(0)).(diagnosticEvaluator)
	if !ok {
		t.Fatal("repairPresentExpectation must satisfy diagnosticEvaluator")
	}
	outcome := checkedOutcome.evaluateDiagnosticReport(
		cli.DiagnosticReport{},
	)
	if outcome.status != evaluationDisproved {
		t.Fatalf("missing repair present = %#v", outcome)
	}
}

func TestRepairPreservesMeaningPreservesRedaction(t *testing.T) {
	t.Parallel()

	t.Run("matching observation", func(t *testing.T) {
		t.Parallel()
		report := cli.DiagnosticReport{
			Repairs: []cli.InputRepair{{PreservesMeaning: true}},
		}
		checkedOutcome, ok := repairPreservesMeaningExpectation(RepairAt(0)).(diagnosticEvaluator)
		if !ok {
			t.Fatal("repairPreservesMeaningExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(report)
		if outcome.status != evaluationProved {
			t.Fatalf("preserves meaning = %#v", outcome)
		}
	})
	t.Run("preserves meaning: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		report := cli.DiagnosticReport{
			Repairs: []cli.InputRepair{{Sensitive: true}},
		}
		checkedOutcome, ok := repairPreservesMeaningExpectation(RepairAt(0)).(diagnosticEvaluator)
		if !ok {
			t.Fatal("repairPreservesMeaningExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(report)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "value differs (redacted)" {
			t.Fatalf("sensitive preserves-meaning mismatch = %#v", outcome)
		}
	})
}

func TestRepairSensitivePreservesRedaction(t *testing.T) {
	t.Parallel()

	t.Run("matching observation", func(t *testing.T) {
		t.Parallel()
		report := cli.DiagnosticReport{
			Repairs: []cli.InputRepair{{Sensitive: true}},
		}
		checkedOutcome, ok := repairSensitiveExpectation(RepairAt(0)).(diagnosticEvaluator)
		if !ok {
			t.Fatal("repairSensitiveExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(report)
		if outcome.status != evaluationProved {
			t.Fatalf("repair sensitive = %#v", outcome)
		}
	})
	t.Run("sensitive: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		report := cli.DiagnosticReport{
			Repairs: []cli.InputRepair{{}},
		}
		checkedOutcome, ok := repairSensitiveExpectation(RepairAt(0)).(diagnosticEvaluator)
		if !ok {
			t.Fatal("repairSensitiveExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(report)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "not sensitive" {
			t.Fatalf("repair sensitivity mismatch = %#v", outcome)
		}
	})
}

func TestRepairSourcesExactlyPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	var sourceID cli.InputSourceID
	report := cli.DiagnosticReport{
		Sources: map[cli.InputSourceID]cli.InputSource{sourceID: {ID: sourceID}},
		Repairs: []cli.InputRepair{{Sources: []cli.InputRepairSource{{Source: sourceID}}}},
	}
	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := repairSourcesExactlyExpectation(
			RepairAt(0),
			[]InputSourceLocator{InputSourceByID(sourceID)},
		).(diagnosticEvaluator)
		if !ok {
			t.Fatal("repairSourcesExactlyExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationProved {
			t.Fatalf("repair sources exact = %#v", outcome)
		}
	})
	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := repairSourcesExactlyExpectation(RepairAt(0), nil).(diagnosticEvaluator)
		if !ok {
			t.Fatal("repairSourcesExactlyExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationDisproved {
			t.Fatalf("repair sources mismatch = %#v", outcome)
		}
		bad := report
		bad.Sources = nil
		unresolvedEvaluator, ok := repairSourcesExactlyExpectation(
			RepairAt(0),
			[]InputSourceLocator{InputSourceByID(sourceID)},
		).(diagnosticEvaluator)
		if !ok {
			t.Fatal("repairSourcesExactlyExpectation must satisfy diagnosticEvaluator")
		}
		outcome = unresolvedEvaluator.evaluateDiagnosticReport(
			bad,
		)
		if outcome.status != evaluationDisproved {
			t.Fatalf("unresolved expected repair source = %#v", outcome)
		}
	})
}

func TestRepairSourceCountPreservesRedaction(t *testing.T) {
	t.Parallel()

	report := cli.DiagnosticReport{
		Repairs: []cli.InputRepair{{Sources: []cli.InputRepairSource{{}}}},
	}
	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := repairSourceTextEditCountExpectation(RepairAt(0).SourceAt(0), 0).(diagnosticEvaluator)
		if !ok {
			t.Fatal("repairSourceTextEditCountExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationProved {
			t.Fatalf("repair source count = %#v", outcome)
		}
	})
	t.Run("count mismatch: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := repairSourceTextEditCountExpectation(RepairAt(0).SourceAt(0), 1).(diagnosticEvaluator)
		if !ok {
			t.Fatal("repairSourceTextEditCountExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationDisproved {
			t.Fatalf("repair source count mismatch = %#v", outcome)
		}
	})
}

func TestRepairTextEditSpanPreservesRedaction(t *testing.T) {
	t.Parallel()

	span := cli.InputSpan{
		Entry: 1,
	}
	report := cli.DiagnosticReport{
		Repairs: []cli.InputRepair{{Sources: []cli.InputRepairSource{{TextEdits: []cli.InputTextEdit{{Span: span}}}}}},
	}
	locator := RepairAt(0).SourceAt(0).TextEditAt(0)
	t.Run("matching observation", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := repairTextEditSpanExpectation(locator, span).(diagnosticEvaluator)
		if !ok {
			t.Fatal("repairTextEditSpanExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(report)
		if outcome.status != evaluationProved {
			t.Fatalf("text edit span = %#v", outcome)
		}
	})
	t.Run("edit span differs: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := repairTextEditSpanExpectation(locator, cli.InputSpan{
			Entry: 2,
		}).(diagnosticEvaluator)
		if !ok {
			t.Fatal("repairTextEditSpanExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(report)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) == "" {
			t.Fatalf("text edit span mismatch = %#v", outcome)
		}
	})
}

func TestRepairTextEditReplacementPreservesRedaction(t *testing.T) {
	t.Parallel()

	replacement := cli.InputReplacement{{Literal: "x"}}
	report := cli.DiagnosticReport{
		Repairs: []cli.InputRepair{
			{Sources: []cli.InputRepairSource{{TextEdits: []cli.InputTextEdit{{Replacement: replacement}}}}},
		},
	}
	locator := RepairAt(0).SourceAt(0).TextEditAt(0)
	t.Run("matching observation", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := repairTextEditReplacementExpectation(locator, replacement).(diagnosticEvaluator)
		if !ok {
			t.Fatal("repairTextEditReplacementExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationProved {
			t.Fatalf("text edit replacement = %#v", outcome)
		}
	})
	t.Run("text edit replacement: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		replacement := cli.InputReplacement{{Literal: "y"}}
		checkedOutcome, ok := repairTextEditReplacementExpectation(locator, replacement).(diagnosticEvaluator)
		if !ok {
			t.Fatal("repairTextEditReplacementExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "replacement differs" {
			t.Fatalf("text edit replacement mismatch = %#v", outcome)
		}
	})
}

func TestRepairEntryEditRangePreservesRedaction(t *testing.T) {
	t.Parallel()

	report := cli.DiagnosticReport{
		Repairs: []cli.InputRepair{
			{Sources: []cli.InputRepairSource{{EntryEdits: []cli.InputEntryEdit{{
				Start: 1,
				End:   2,
			}}}}},
		},
	}
	locator := RepairAt(0).SourceAt(0).EntryEditAt(0)
	t.Run("matching observation", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := repairEntryEditRangeExpectation(locator, 1, 2).(diagnosticEvaluator)
		if !ok {
			t.Fatal("repairEntryEditRangeExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(report)
		if outcome.status != evaluationProved {
			t.Fatalf("entry edit range = %#v", outcome)
		}
	})
	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := repairEntryEditRangeExpectation(locator, 0, 2).(diagnosticEvaluator)
		if !ok {
			t.Fatal("repairEntryEditRangeExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(report)
		if outcome.status != evaluationDisproved {
			t.Fatalf("entry edit range mismatch = %#v", outcome)
		}
	})
}

func TestRepairEntryEditEntriesPreservesRedaction(t *testing.T) {
	t.Parallel()

	entries := []cli.InputReplacement{{{Literal: "x"}}}
	report := cli.DiagnosticReport{
		Repairs: []cli.InputRepair{
			{Sources: []cli.InputRepairSource{{EntryEdits: []cli.InputEntryEdit{{Entries: entries}}}}},
		},
	}
	locator := RepairAt(0).SourceAt(0).EntryEditAt(0)
	t.Run("matching observation", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := repairEntryEditEntriesExpectation(locator, entries).(diagnosticEvaluator)
		if !ok {
			t.Fatal("repairEntryEditEntriesExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationProved {
			t.Fatalf("entry edit entries = %#v", outcome)
		}
	})
	t.Run("entry edit entries: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		replacement := []cli.InputReplacement{{{Literal: "y"}}}
		checkedOutcome, ok := repairEntryEditEntriesExpectation(
			locator, replacement,
		).(diagnosticEvaluator)
		if !ok {
			t.Fatal("repairEntryEditEntriesExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "replacement entries differ" {
			t.Fatalf("entry edit entries mismatch = %#v", outcome)
		}
	})
}

func TestInputSourcePresentPreservesRedaction(t *testing.T) {
	t.Parallel()

	var sourceID cli.InputSourceID
	checkedOutcome, ok := inputSourcePresentExpectation(InputSourceByID(sourceID)).(diagnosticEvaluator)
	if !ok {
		t.Fatal("inputSourcePresentExpectation must satisfy diagnosticEvaluator")
	}
	outcome := checkedOutcome.evaluateDiagnosticReport(
		cli.DiagnosticReport{},
	)
	if outcome.status != evaluationDisproved {
		t.Fatalf("missing source present = %#v", outcome)
	}
}

func TestInputSourceShortcutPreservesRedaction(t *testing.T) {
	t.Parallel()

	var sourceID cli.InputSourceID
	report := cli.DiagnosticReport{
		Sources: map[cli.InputSourceID]cli.InputSource{sourceID: {
			ID:              sourceID,
			CommandShortcut: 7,
		}},
	}
	t.Run("valid command shortcut: semantic result", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := inputSourceShortcutExpectation(
			InputSourceByID(sourceID),
			CommandEntryByID(7),
		).(diagnosticEvaluator)
		if !ok {
			t.Fatal("inputSourceShortcutExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationProved {
			t.Fatalf("command shortcut = %#v", outcome)
		}
	})
	t.Run("invalid coordinates", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := inputSourceShortcutExpectation(
			InputSourceByID(sourceID),
			CommandEntryLocator{},
		).(diagnosticEvaluator)
		if !ok {
			t.Fatal("inputSourceShortcutExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "invalid command-entry locator" {
			t.Fatalf("invalid shortcut locator = %#v", outcome)
		}
	})
	t.Run("valid command shortcut: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := inputSourceShortcutExpectation(
			InputSourceLocator{},
			CommandEntryLocator{},
		).(diagnosticEvaluator)
		if !ok {
			t.Fatal("inputSourceShortcutExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationDisproved {
			t.Fatalf("invalid source shortcut = %#v", outcome)
		}
	})
}

func TestInputOriginSensitivePreservesRedaction(t *testing.T) {
	t.Parallel()

	var originID cli.InputOriginID
	t.Run("matching observation", func(t *testing.T) {
		t.Parallel()
		report := cli.DiagnosticReport{
			Origins: map[cli.InputOriginID]cli.InputOrigin{originID: {
				ID:        originID,
				Sensitive: true,
			}},
		}
		checkedOutcome, ok := inputOriginSensitiveExpectation(InputOriginByID(originID)).(diagnosticEvaluator)
		if !ok {
			t.Fatal("inputOriginSensitiveExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationProved {
			t.Fatalf("origin sensitive = %#v", outcome)
		}
	})
	t.Run("input origin sensitive: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		report := cli.DiagnosticReport{
			Origins: map[cli.InputOriginID]cli.InputOrigin{originID: {ID: originID}},
		}
		checkedOutcome, ok := inputOriginSensitiveExpectation(InputOriginByID(originID)).(diagnosticEvaluator)
		if !ok {
			t.Fatal("inputOriginSensitiveExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "not sensitive" {
			t.Fatalf("origin sensitivity mismatch = %#v", outcome)
		}
	})
}

func TestInputOriginParentsExactlyPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	var originID cli.InputOriginID
	report := cli.DiagnosticReport{
		Origins: map[cli.InputOriginID]cli.InputOrigin{
			originID: {
				ID:      originID,
				Parents: []cli.InputOriginID{originID},
			},
		},
	}
	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := inputOriginParentsExactlyExpectation(
			InputOriginByID(originID),
			[]InputOriginLocator{InputOriginByID(originID)},
		).(diagnosticEvaluator)
		if !ok {
			t.Fatal("inputOriginParentsExactlyExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationProved {
			t.Fatalf("origin parents exact = %#v", outcome)
		}
	})
	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := inputOriginParentsExactlyExpectation(
			InputOriginByID(originID),
			nil,
		).(diagnosticEvaluator)
		if !ok {
			t.Fatal("inputOriginParentsExactlyExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationDisproved {
			t.Fatalf("origin parent mismatch = %#v", outcome)
		}
		bad := report
		bad.Origins = nil
		unresolvedEvaluator, ok := inputOriginParentsExactlyExpectation(
			InputOriginByID(originID),
			[]InputOriginLocator{InputOriginByID(originID)},
		).(diagnosticEvaluator)
		if !ok {
			t.Fatal("inputOriginParentsExactlyExpectation must satisfy diagnosticEvaluator")
		}
		outcome = unresolvedEvaluator.evaluateDiagnosticReport(
			bad,
		)
		if outcome.status != evaluationDisproved {
			t.Fatalf("unresolved observed origin = %#v", outcome)
		}
		invalidParents := []InputOriginLocator{{}}
		invalidExpectedEvaluator, ok := inputOriginParentsExactlyExpectation(
			InputOriginByID(originID),
			invalidParents,
		).(diagnosticEvaluator)
		if !ok {
			t.Fatal("inputOriginParentsExactlyExpectation must satisfy diagnosticEvaluator")
		}
		outcome = invalidExpectedEvaluator.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationDisproved {
			t.Fatalf("unresolved expected parent = %#v", outcome)
		}
	})
}

func TestInputOriginSpanPreservesRedaction(t *testing.T) {
	t.Parallel()

	var originID cli.InputOriginID
	span := cli.InputSpan{
		Entry: 1,
	}
	t.Run("matching observation", func(t *testing.T) {
		t.Parallel()
		report := cli.DiagnosticReport{
			Origins: map[cli.InputOriginID]cli.InputOrigin{originID: {
				ID:   originID,
				Span: span,
			}},
		}
		checkedOutcome, ok := inputOriginSpanExpectation(InputOriginByID(originID), span).(diagnosticEvaluator)
		if !ok {
			t.Fatal("inputOriginSpanExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationProved {
			t.Fatalf("origin span = %#v", outcome)
		}
	})
	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		report := cli.DiagnosticReport{
			Origins: map[cli.InputOriginID]cli.InputOrigin{originID: {
				ID:        originID,
				Span:      span,
				Sensitive: true,
			}},
		}
		checkedOutcome, ok := inputOriginSpanExpectation(InputOriginByID(originID), cli.InputSpan{
			Entry: 2,
		}).(diagnosticEvaluator)
		if !ok {
			t.Fatal("inputOriginSpanExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(report)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "value differs (redacted)" {
			t.Fatalf("sensitive origin span mismatch = %#v", outcome)
		}
	})
}

func TestDiagnosticInputSourcePreservesRedaction(t *testing.T) {
	t.Parallel()

	var sourceID cli.InputSourceID
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{
			Code:        cli.DiagnosticInvalidValue,
			InputSource: sourceID,
		}},
		Sources: map[cli.InputSourceID]cli.InputSource{sourceID: {ID: sourceID}},
	}
	t.Run("matching observation", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := diagnosticInputSourceExpectation(
			DiagnosticAt(0),
			InputSourceByID(sourceID),
		).(diagnosticEvaluator)
		if !ok {
			t.Fatal("diagnosticInputSourceExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationProved {
			t.Fatalf("diagnostic input source = %#v", outcome)
		}
	})
	t.Run("expected subject unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		bad := report
		bad.Sources = nil
		checkedOutcome, ok := diagnosticInputSourceExpectation(
			DiagnosticAt(0),
			InputSourceByID(sourceID),
		).(diagnosticEvaluator)
		if !ok {
			t.Fatal("diagnosticInputSourceExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			bad,
		)
		if outcome.status != evaluationDisproved {
			t.Fatalf("unresolved expected source = %#v", outcome)
		}
	})
	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		definition := cli.Definition[struct{}]{
			Name: "diagnostic-source",
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
			t.Fatal("unknown flag unexpectedly succeeded")
		}
		actual := cli.Diagnostics(err)
		if len(actual.Diagnostics) == 0 {
			t.Fatalf("fixture did not produce a diagnostic: %#v", actual)
		}
		var observedSource cli.InputSourceID
		foundObservedSource := false
		for id := range actual.Sources {
			if id != sourceID {
				observedSource = id
				foundObservedSource = true
				break
			}
		}
		if !foundObservedSource {
			t.Fatalf("fixture did not expose a distinct input source: %#v", actual)
		}
		actual.Diagnostics[0].InputSource = observedSource
		if actual.Sources == nil {
			actual.Sources = make(map[cli.InputSourceID]cli.InputSource)
		}
		actual.Sources[sourceID] = cli.InputSource{
			ID:        sourceID,
			Sensitive: true,
		}
		checkedOutcome, ok := diagnosticInputSourceExpectation(
			DiagnosticAt(0),
			InputSourceByID(sourceID),
		).(diagnosticEvaluator)
		if !ok {
			t.Fatal("diagnosticInputSourceExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			actual,
		)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "value differs (redacted)" {
			t.Fatalf("diagnostic source mismatch = %#v", outcome)
		}
	})
}

func TestDiagnosticProviderOutputPreservesRedaction(t *testing.T) {
	t.Parallel()

	actual := cli.CommandProviderOutputLocator{
		Index:    1,
		Redacted: true,
	}
	expected := cli.CommandProviderOutputLocator{
		Index: 2,
	}
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{
			Code:           cli.DiagnosticInvalidValue,
			ProviderOutput: &actual,
		}},
	}
	t.Run("matching observation", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := diagnosticProviderOutputExpectation(
			DiagnosticAt(0),
			actual,
		).(diagnosticEvaluator)
		if !ok {
			t.Fatal("diagnosticProviderOutputExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationProved {
			t.Fatalf("provider output = %#v", outcome)
		}
	})
	t.Run("redacted value: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := diagnosticProviderOutputExpectation(
			DiagnosticAt(0),
			expected,
		).(diagnosticEvaluator)
		if !ok {
			t.Fatal("diagnosticProviderOutputExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "value differs (redacted)" {
			t.Fatalf("provider output mismatch = %#v", outcome)
		}
	})
}

func testDiagnosticInputOriginID(t *testing.T) cli.InputOriginID {
	t.Helper()

	definition := cli.Definition[struct{}]{Name: "diagnostic-origin-identity"}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	request := cli.InspectionRequest{Args: []string{"unexpected"}}
	inspection, _ := app.Inspect(context.Background(), request)
	for id := range inspection.Origins {
		if id != (cli.InputOriginID{}) {
			return id
		}
	}

	t.Fatalf("fixture did not expose a nonzero input origin: %#v", inspection.Origins)
	return cli.InputOriginID{}
}

func TestDiagnosticOriginsExactlyPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	originID := testDiagnosticInputOriginID(t)
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{
			Code:    cli.DiagnosticInvalidValue,
			Origins: []cli.InputOriginID{originID},
		}},
		Origins: map[cli.InputOriginID]cli.InputOrigin{originID: {ID: originID}},
	}
	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := diagnosticOriginsExactlyExpectation(
			DiagnosticAt(0),
			[]InputOriginLocator{InputOriginByID(originID)},
		).(diagnosticEvaluator)
		if !ok {
			t.Fatal("diagnosticOriginsExactlyExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationProved {
			t.Fatalf("diagnostic origins exact = %#v", outcome)
		}
	})
	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := diagnosticOriginsExactlyExpectation(DiagnosticAt(0), nil).(diagnosticEvaluator)
		if !ok {
			t.Fatal("diagnosticOriginsExactlyExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationDisproved {
			t.Fatalf("diagnostic origins mismatch = %#v", outcome)
		}
		bad := report
		bad.Origins = nil
		unresolvedEvaluator, ok := diagnosticOriginsExactlyExpectation(
			DiagnosticAt(0),
			[]InputOriginLocator{InputOriginByID(originID)},
		).(diagnosticEvaluator)
		if !ok {
			t.Fatal("diagnosticOriginsExactlyExpectation must satisfy diagnosticEvaluator")
		}
		outcome = unresolvedEvaluator.evaluateDiagnosticReport(
			bad,
		)
		if outcome.status != evaluationProved {
			t.Fatalf("direct origin identity without detail = %#v", outcome)
		}
	})
}

func TestDiagnosticReplacementPreservesRedaction(t *testing.T) {
	t.Parallel()

	replacement := &cli.DeclarationReplacement{
		Target: cli.DeclarationIdentity{
			Kind: cli.DeclarationKindFlag,
			Flag: 1,
		},
	}
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{
			Code:        cli.DiagnosticInvalidValue,
			Replacement: replacement,
			Sensitive:   true,
		}},
	}
	t.Run("invalid coordinates", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := diagnosticReplacementExpectation(
			DiagnosticAt(9),
			FlagByID(1),
		).(diagnosticEvaluator)
		if !ok {
			t.Fatal("diagnosticReplacementExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationDisproved {
			t.Fatalf("missing diagnostic replacement = %#v", outcome)
		}
	})
	t.Run("valid replacement declaration: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := diagnosticReplacementExpectation(DiagnosticAt(0), nil).(diagnosticEvaluator)
		if !ok {
			t.Fatal("diagnosticReplacementExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "invalid declaration locator" {
			t.Fatalf("invalid replacement locator = %#v", outcome)
		}
	})
	t.Run("subject mismatch: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		checkedOutcome, ok := diagnosticReplacementExpectation(
			DiagnosticAt(0),
			FlagByID(2),
		).(diagnosticEvaluator)
		if !ok {
			t.Fatal("diagnosticReplacementExpectation must satisfy diagnosticEvaluator")
		}
		outcome := checkedOutcome.evaluateDiagnosticReport(
			report,
		)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "value differs (redacted)" {
			t.Fatalf("replacement mismatch = %#v", outcome)
		}
	})
}

func TestDiagnosticOriginsExactlyAcceptsDirectIdentityWithoutOriginGraphDetails(t *testing.T) {
	t.Parallel()

	originID := testDiagnosticInputOriginID(t)

	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{
			Code:    "example",
			Origins: []cli.InputOriginID{originID},
		}},
	}
	expectation, ok := diagnosticOriginsExactlyExpectation(
		DiagnosticAt(0),
		[]InputOriginLocator{InputOriginByID(originID)},
	).(diagnosticEvaluator)
	if !ok {
		t.Fatal("diagnosticOriginsExactlyExpectation must satisfy diagnosticEvaluator")
	}
	if got := expectation.evaluateDiagnosticReport(report); got.status != evaluationProved {
		t.Fatalf("direct origin identity comparison = %+v", got)
	}
}

func TestLocatorInputOriginIDsPreservesRedaction(t *testing.T) {
	t.Parallel()

	locators := []InputOriginLocator{{}}
	ids, ok := locatorInputOriginIDs(locators)
	if ok || ids != nil {
		t.Fatalf("invalid origin locators = %#v ok %v", ids, ok)
	}
}

func TestLocatorDeclarationIdentityValuePreservesRedaction(t *testing.T) {
	t.Parallel()

	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		values := []DeclarationIdentityLocator{
			CommandEntryByID(
				1,
			),
			ActionEntryByID(2),
			FlagByID(3),
			FlagSpellingByID(4),
			ArgumentFormByID(5),
			ArgumentTermByID(6),
			EnvironmentInputByID(7),
			ValueDomainEntryByID(8),
			nil,
		}
		for i, value := range values {
			identity, ok := locatorDeclarationIdentityValue(value)
			if i < len(values)-1 {
				if !ok || identity.Kind == cli.DeclarationKindUnknown {
					t.Fatalf("declaration locator %d = %#v ok %v", i, identity, ok)
				}
			} else if ok {
				t.Fatalf("unsupported declaration locator unexpectedly resolved: %#v", identity)
			}
		}
		if _, ok := locatorDeclarationIdentityValue(CommandEntryLocator{}); ok {
			t.Fatal("zero declaration locator resolved")
		}
	})
}

func TestOutcomeInternalStateFailsClosed(t *testing.T) {
	t.Parallel()
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{RepairIndexes: []int{1}}},
	}
	if got := resolveRepair(
		DiagnosticAt(0).RepairAt(0).state,
		report,
	); got.status != locatorResolutionInvalid ||
		renderEvaluationField(got.actual) != "diagnostic references an unavailable repair" {
		t.Fatalf("repair resolution = %#v", got)
	}
}

func TestOutcomeExpectationsConserveUnresolvedSubjects(t *testing.T) {
	t.Parallel()
	empty := cli.DiagnosticReport{}

	checks := []struct {
		name string
		exp  OutcomeExpectation
	}{
		{"repair preserves meaning", RepairAt(0).PreservesMeaning()},
		{"repair sensitive", RepairAt(0).Sensitive()},
		{"repair sources", RepairAt(0).ProducerKindsExactly()},
		{"repair source count", RepairAt(0).SourceAt(0).TextEditCount(0)},
		{"text edit span", RepairAt(0).SourceAt(0).TextEditAt(0).Span(cli.InputSpan{})},
		{"text edit replacement", RepairAt(0).SourceAt(0).TextEditAt(0).Replacement(nil)},
		{"entry edit range", RepairAt(0).SourceAt(0).EntryEditAt(0).Range(0, 0)},
		{"entry edit entries", RepairAt(0).SourceAt(0).EntryEditAt(0).EntriesExactly()},
		{"input source property", InputSourceByID(cli.InputSourceID{}).Key("x")},
		{"input origin sensitive", InputOriginByID(cli.InputOriginID{}).Sensitive()},
		{"input origin parents", InputOriginByID(cli.InputOriginID{}).ParentsExactly()},
		{"input origin span", InputOriginByID(cli.InputOriginID{}).Span(cli.InputSpan{})},
		{"diagnostic input source", DiagnosticAt(0).InputSource(InputSourceByID(cli.InputSourceID{}))},
		{"diagnostic provider output", DiagnosticAt(0).ProviderOutput(cli.CommandProviderOutputLocator{})},
		{"diagnostic origins", DiagnosticAt(0).OriginsExactly()},
	}
	for _, check := range checks {
		checkedGot, ok := check.exp.(diagnosticEvaluator)
		if !ok {
			t.Fatal("exp must satisfy diagnosticEvaluator")
		}
		got := checkedGot.evaluateDiagnosticReport(empty)
		if got.status == evaluationProved {
			t.Fatalf("%s unexpectedly satisfied: %#v", check.name, got)
		}
	}

	sourceID := cli.InputSourceID{}
	report := cli.DiagnosticReport{
		Sources: map[cli.InputSourceID]cli.InputSource{sourceID: {ID: sourceID}},
		Repairs: []cli.InputRepair{{Sources: []cli.InputRepairSource{}}},
	}
	state := RepairAt(0).Source(InputSourceByID(sourceID)).state
	if got := resolveRepairSource(state, report); got.status != locatorResolutionAbsent {
		t.Fatalf("missing repair source identity = %#v", got)
	}
}
