package clitest

import (
	"fmt"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
)

// RepairLocator identifies one structured repair for clitest semantic expectation. Its
// zero value is invalid.
type RepairLocator struct {
	_     [0]func()
	state *repairLocatorState
}

// RepairSourceLocator identifies one source referenced by a repair for clitest
// semantic expectation. Its zero value is invalid.
type RepairSourceLocator struct {
	_     [0]func()
	state *repairSourceLocatorState
}

// RepairTextEditLocator identifies one text edit in a repair for clitest
// semantic expectation. Its zero value is invalid.
type RepairTextEditLocator struct {
	_     [0]func()
	state *repairTextEditLocatorState
}

// RepairEntryEditLocator identifies one entry edit in a repair for clitest
// semantic expectation. Its zero value is invalid.
type RepairEntryEditLocator struct {
	_     [0]func()
	state *repairEntryEditLocatorState
}

// RepairAt returns an expected coordinate for the repair at index in DiagnosticReport.Repairs authoritative order.
// Truncation prevents an out-of-range coordinate from checking absence.
func RepairAt(index int) RepairLocator { return repairByIndex(index) }

// Source returns the repair-source relationship whose Source identity equals the input source selected by source in the
// same diagnostic report.
func (locator RepairLocator) Source(source InputSourceLocator) RepairSourceLocator {
	return repairSourceBySource(locator, source)
}

// SourceAt returns the repair source at index in the located repair's authoritative Sources order.
func (locator RepairLocator) SourceAt(index int) RepairSourceLocator {
	return repairSourceByIndex(locator, index)
}

// Present returns an outcome expectation that the repair coordinate resolves in the structured diagnostic report.
func (locator RepairLocator) Present() OutcomeExpectation { return repairPresentExpectation(locator) }

// PreservesMeaning returns an outcome expectation that the located repair is marked as preserving meaning.
func (locator RepairLocator) PreservesMeaning() OutcomeExpectation {
	return repairPreservesMeaningExpectation(locator)
}

// Sensitive returns an outcome expectation that the located repair is marked sensitive. Mismatch reporting for
// sensitive repair data remains redacted.
func (locator RepairLocator) Sensitive() OutcomeExpectation {
	return repairSensitiveExpectation(locator)
}

// ProducerKindsExactly returns an outcome expectation that the located repair's source relationships exactly equal sources in
// authoritative order within the same diagnostic report.
func (locator RepairLocator) ProducerKindsExactly(sources ...InputSourceLocator) OutcomeExpectation {
	return repairSourcesExactlyExpectation(locator, sources)
}

// TextEditAt returns the text edit at index in the located repair source's authoritative TextEdits order.
func (locator RepairSourceLocator) TextEditAt(index int) RepairTextEditLocator {
	return repairTextEditByIndex(locator, index)
}

// EntryEditAt returns the entry edit at index in the located repair source's authoritative EntryEdits order.
func (locator RepairSourceLocator) EntryEditAt(index int) RepairEntryEditLocator {
	return repairEntryEditByIndex(locator, index)
}

// TextEditCount returns an outcome expectation that the located repair source carries exactly count text edits.
func (locator RepairSourceLocator) TextEditCount(count int) OutcomeExpectation {
	return repairSourceTextEditCountExpectation(locator, count)
}

// EntryEditCount returns an outcome expectation that the located repair source carries exactly count entry edits.
func (locator RepairSourceLocator) EntryEditCount(count int) OutcomeExpectation {
	return repairSourceEntryEditCountExpectation(locator, count)
}

// Span returns an outcome expectation that the located text edit has exactly span.
func (locator RepairTextEditLocator) Span(span cli.InputSpan) OutcomeExpectation {
	return repairTextEditSpanExpectation(locator, span)
}

// Replacement returns an outcome expectation that the located text edit carries exactly replacement.
func (locator RepairTextEditLocator) Replacement(replacement cli.InputReplacement) OutcomeExpectation {
	return repairTextEditReplacementExpectation(locator, replacement)
}

// Range returns an outcome expectation that the located entry edit has exactly the half-open input range [start, end).
func (locator RepairEntryEditLocator) Range(start, end int) OutcomeExpectation {
	return repairEntryEditRangeExpectation(locator, start, end)
}

// EntriesExactly returns an outcome expectation that the located entry edit carries exactly entries in authoritative
// order.
func (locator RepairEntryEditLocator) EntriesExactly(entries ...cli.InputReplacement) OutcomeExpectation {
	return repairEntryEditEntriesExpectation(locator, entries)
}

type repairResolution struct {
	status locatorResolutionStatus
	value  cli.InputRepair
	index  int
	label  evaluationField
	actual evaluationField
}

type repairSourceResolution struct {
	status locatorResolutionStatus
	value  cli.InputRepairSource
	index  int
	label  evaluationField
	actual evaluationField
}

type textEditResolution struct {
	status locatorResolutionStatus
	value  cli.InputTextEdit
	label  evaluationField
	actual evaluationField
}

type entryEditResolution struct {
	status locatorResolutionStatus
	value  cli.InputEntryEdit
	label  evaluationField
	actual evaluationField
}

type repairExpectationKind uint8

const (
	repairExpectationInvalid repairExpectationKind = iota
	repairExpectationPresent
	repairExpectationPreservesMeaning
	repairExpectationSensitive
	repairExpectationSourcesExactly
)

type repairExpectation struct {
	locator *repairLocatorState
	kind    repairExpectationKind
	sources []*inputSourceLocatorState
}

type repairSourceExpectation struct {
	locator *repairSourceLocatorState
	text    bool
	count   int
}

type textEditExpectationKind uint8

const (
	textEditExpectationInvalid textEditExpectationKind = iota
	textEditExpectationSpan
	textEditExpectationReplacement
)

type textEditExpectation struct {
	locator     *repairTextEditLocatorState
	kind        textEditExpectationKind
	span        cli.InputSpan
	replacement cli.InputReplacement
}

type entryEditExpectationKind uint8

const (
	entryEditExpectationInvalid entryEditExpectationKind = iota
	entryEditExpectationRange
	entryEditExpectationEntries
)

type entryEditExpectation struct {
	locator    *repairEntryEditLocatorState
	kind       entryEditExpectationKind
	start, end int
	entries    []cli.InputReplacement
}

type repairLocatorMode uint8

const (
	repairLocatorInvalid repairLocatorMode = iota
	repairLocatorIndex
	repairLocatorDiagnosticIndex
	repairLocatorNoticeIndex
)

type repairLocatorState struct {
	mode       repairLocatorMode
	index      int
	diagnostic *diagnosticLocatorState
	notice     *noticeLocatorState
}

type repairSourceLocatorMode uint8

const (
	repairSourceLocatorInvalid repairSourceLocatorMode = iota
	repairSourceLocatorIndex
	repairSourceLocatorSource
)

type repairSourceLocatorState struct {
	mode   repairSourceLocatorMode
	repair *repairLocatorState
	index  int
	source *inputSourceLocatorState
}

type repairTextEditLocatorState struct {
	source *repairSourceLocatorState
	index  int
}

type repairEntryEditLocatorState struct {
	source *repairSourceLocatorState
	index  int
}

func repairByIndex(index int) RepairLocator {
	return RepairLocator{state: &repairLocatorState{
		mode:  repairLocatorIndex,
		index: index,
	}}
}

func repairSourceBySource(repair RepairLocator, source InputSourceLocator) RepairSourceLocator {
	return RepairSourceLocator{state: &repairSourceLocatorState{
		mode:   repairSourceLocatorSource,
		repair: repair.state,
		source: source.state,
	}}
}

func repairSourceByIndex(repair RepairLocator, index int) RepairSourceLocator {
	return RepairSourceLocator{state: &repairSourceLocatorState{
		mode:   repairSourceLocatorIndex,
		repair: repair.state,
		index:  index,
	}}
}

func repairTextEditByIndex(source RepairSourceLocator, index int) RepairTextEditLocator {
	return RepairTextEditLocator{state: &repairTextEditLocatorState{
		source: source.state,
		index:  index,
	}}
}

func repairEntryEditByIndex(source RepairSourceLocator, index int) RepairEntryEditLocator {
	return RepairEntryEditLocator{state: &repairEntryEditLocatorState{
		source: source.state,
		index:  index,
	}}
}

func resolveRepair(state *repairLocatorState, report cli.DiagnosticReport) repairResolution {
	return resolveRepairObservation(state, report, nil)
}

func resolveRepairInspection(state *repairLocatorState, inspection cli.Inspection) repairResolution {
	report := diagnosticReportFromInspection(inspection)
	return resolveRepairObservation(state, report, &inspection)
}

func resolveRepairObservation(state *repairLocatorState, report cli.DiagnosticReport, inspection *cli.Inspection) repairResolution {
	var resolution repairResolution
	invalid := repairResolution{
		status: locatorResolutionInvalid,
		label:  evaluationText("repair"),
		actual: evaluationText("invalid repair locator"),
	}
	if state == nil || state.index < 0 {
		return invalid
	}

	switch state.mode {
	case repairLocatorIndex:
		label := evaluationText(fmt.Sprintf("repair at index %d", state.index))
		if state.index < len(report.Repairs) {
			resolution = repairResolution{
				status: locatorResolutionFound,
				value:  report.Repairs[state.index],
				index:  state.index,
				label:  label,
			}
			return resolution
		}
		if report.Truncated {
			resolution = repairResolution{
				status: locatorResolutionNotProvable,
				label:  label,
				actual: observed(semanticSubjectDiagnosticReport, observationTruncated),
			}
			return resolution
		}
		resolution = repairResolution{
			status: locatorResolutionAbsent,
			label:  label,
			actual: evaluationText("repair is absent"),
		}
		return resolution
	case repairLocatorDiagnosticIndex:
		if state.diagnostic == nil {
			return invalid
		}
		var diagnostic locatorResolution[diagnosticLocation]
		if inspection != nil {
			diagnostic = resolveDiagnosticStateForInspection(state.diagnostic, *inspection)
		} else {
			diagnostic = resolveDiagnosticStateForReport(state.diagnostic, report)
		}
		label := qualifySubject(diagnostic.subject, evaluationText(fmt.Sprintf("repair %d", state.index)))
		if diagnostic.status != locatorResolutionFound {
			resolution = repairResolution{
				status: diagnostic.status,
				label:  label,
				actual: diagnostic.detail,
			}
			return resolution
		}
		indexes := diagnostic.value.diagnostic.RepairIndexes
		if state.index >= len(indexes) {
			resolution = repairResolution{
				status: locatorResolutionAbsent,
				label:  label,
				actual: evaluationText("diagnostic repair relationship is absent"),
			}
			return resolution
		}
		absolute := indexes[state.index]
		if absolute < 0 || absolute >= len(report.Repairs) {
			resolution = repairResolution{
				status: locatorResolutionInvalid,
				label:  label,
				actual: evaluationText("diagnostic references an unavailable repair"),
			}
			return resolution
		}
		resolution = repairResolution{
			status: locatorResolutionFound,
			value:  report.Repairs[absolute],
			index:  absolute,
			label:  label,
		}
		return resolution
	case repairLocatorNoticeIndex:
		if inspection == nil || state.notice == nil {
			return invalid
		}
		notice := resolveNotice(state.notice, *inspection)
		label := qualifySubject(notice.label, evaluationText(fmt.Sprintf("repair %d", state.index)))
		if notice.status != locatorResolutionFound {
			resolution = repairResolution{
				status: notice.status,
				label:  label,
				actual: notice.actual,
			}
			return resolution
		}
		if state.index >= len(notice.value.RepairIndexes) {
			if inspection.Truncated {
				resolution = repairResolution{
					status: locatorResolutionNotProvable,
					label:  label,
					actual: observed(semanticSubjectInspection, observationTruncated),
				}
				return resolution
			}
			resolution = repairResolution{
				status: locatorResolutionAbsent,
				label:  label,
				actual: evaluationText("notice repair relationship is absent"),
			}
			return resolution
		}
		absolute := notice.value.RepairIndexes[state.index]
		if absolute < 0 || absolute >= len(report.Repairs) {
			resolution = repairResolution{
				status: locatorResolutionInvalid,
				label:  label,
				actual: evaluationText("notice references an unavailable repair"),
			}
			return resolution
		}
		resolution = repairResolution{
			status: locatorResolutionFound,
			value:  report.Repairs[absolute],
			index:  absolute,
			label:  label,
		}
		return resolution
	default:
		return invalid
	}
}

func resolveRepairSource(state *repairSourceLocatorState, report cli.DiagnosticReport) repairSourceResolution {
	return resolveRepairSourceObservation(state, report, nil)
}

func resolveRepairSourceInspection(state *repairSourceLocatorState, inspection cli.Inspection) repairSourceResolution {
	report := diagnosticReportFromInspection(inspection)
	return resolveRepairSourceObservation(state, report, &inspection)
}

func resolveRepairSourceObservation(state *repairSourceLocatorState, report cli.DiagnosticReport, inspection *cli.Inspection) repairSourceResolution {
	var resolution repairSourceResolution
	if state == nil || state.repair == nil {
		resolution = repairSourceResolution{
			status: locatorResolutionInvalid,
			label:  semanticSubjectRepairSource,
			actual: observed(semanticSubjectRepairSource, observationInvalidLocator),
		}
		return resolution
	}
	switch state.mode {
	case repairSourceLocatorIndex:
		if state.index < 0 {
			resolution = repairSourceResolution{
				status: locatorResolutionInvalid,
				label:  semanticSubjectRepairSource,
				actual: observed(semanticSubjectRepairSource, observationInvalidLocator),
			}
			return resolution
		}
	case repairSourceLocatorSource:
		if state.source == nil {
			resolution = repairSourceResolution{
				status: locatorResolutionInvalid,
				label:  semanticSubjectRepairSource,
				actual: observed(semanticSubjectRepairSource, observationInvalidLocator),
			}
			return resolution
		}
	default:
		resolution = repairSourceResolution{
			status: locatorResolutionInvalid,
			label:  semanticSubjectRepairSource,
			actual: observed(semanticSubjectRepairSource, observationInvalidLocator),
		}
		return resolution
	}
	var repair repairResolution
	if inspection != nil {
		repair = resolveRepairInspection(state.repair, *inspection)
	} else {
		repair = resolveRepair(state.repair, report)
	}
	var label evaluationField = qualifySubject(repair.label, evaluationText("source"))
	if repair.status != locatorResolutionFound {
		resolution = repairSourceResolution{
			status: repair.status,
			label:  label,
			actual: repair.actual,
		}
		return resolution
	}

	switch state.mode {
	case repairSourceLocatorIndex:
		if state.index < 0 {
			resolution = repairSourceResolution{
				status: locatorResolutionInvalid,
				label:  label,
				actual: observed(semanticSubjectRepairSource, observationInvalidLocator),
			}
			return resolution
		}
		label = indexedSubject(evaluationText("source"), state.index, repair.label)
		if state.index < len(repair.value.Sources) {
			resolution = repairSourceResolution{
				status: locatorResolutionFound,
				value:  repair.value.Sources[state.index],
				index:  state.index,
				label:  label,
			}
			return resolution
		}
		resolution = repairSourceResolution{
			status: locatorResolutionAbsent,
			label:  label,
			actual: evaluationText("repair source is absent"),
		}
		return resolution
	case repairSourceLocatorSource:
		if state.source == nil {
			resolution = repairSourceResolution{
				status: locatorResolutionInvalid,
				label:  label,
				actual: observed(semanticSubjectRepairSource, observationInvalidLocator),
			}
			return resolution
		}
		var source sourceResolution
		if inspection != nil {
			source = resolveSourceInspection(state.source, *inspection)
		} else {
			source = resolveSource(state.source, report)
		}
		if source.status != locatorResolutionFound {
			resolution = repairSourceResolution{
				status: source.status,
				label:  label,
				actual: withDetail("expected input-source locator could not be resolved: ", source.actual),
			}
			return resolution
		}
		matches := make([]int, 0, 1)
		for index, repairSource := range repair.value.Sources {
			if repairSource.Source == source.value.ID {
				matches = append(matches, index)
			}
		}
		switch len(matches) {
		case 1:
			index := matches[0]
			resolution = repairSourceResolution{
				status: locatorResolutionFound,
				value:  repair.value.Sources[index],
				index:  index,
				label:  label,
			}
			return resolution
		case 0:
			resolution = repairSourceResolution{
				status: locatorResolutionAbsent,
				label:  label,
				actual: evaluationText("repair source is absent"),
			}
			return resolution
		default:
			resolution = repairSourceResolution{
				status: locatorResolutionAmbiguous,
				label:  label,
				actual: collectionCardinalityField{
					name:  "matches repair source indexes",
					count: len(matches),
				},
			}
			return resolution
		}
	default:
		resolution = repairSourceResolution{
			status: locatorResolutionInvalid,
			label:  label,
			actual: observed(semanticSubjectRepairSource, observationInvalidLocator),
		}
		return resolution
	}
}

func resolveTextEdit(state *repairTextEditLocatorState, report cli.DiagnosticReport) textEditResolution {
	return resolveTextEditObservation(state, report, nil)
}

func resolveTextEditInspection(state *repairTextEditLocatorState, inspection cli.Inspection) textEditResolution {
	return resolveTextEditObservation(state, diagnosticReportFromInspection(inspection), &inspection)
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func resolveTextEditObservation(state *repairTextEditLocatorState, report cli.DiagnosticReport, inspection *cli.Inspection) textEditResolution {
	var resolution textEditResolution
	if state == nil || state.source == nil || state.index < 0 {
		resolution = textEditResolution{
			status: locatorResolutionInvalid,
			label:  evaluationText("repair text edit"),
			actual: observed(semanticSubjectRepairSource, observationInvalidLocator),
		}
		return resolution
	}
	var source repairSourceResolution
	if inspection != nil {
		source = resolveRepairSourceInspection(state.source, *inspection)
	} else {
		source = resolveRepairSource(state.source, report)
	}
	label := indexedSubject(evaluationText("text edit"), state.index, source.label)
	if source.status != locatorResolutionFound {
		resolution = textEditResolution{
			status: source.status,
			label:  label,
			actual: source.actual,
		}
		return resolution
	}
	if state.index < len(source.value.TextEdits) {
		resolution = textEditResolution{
			status: locatorResolutionFound,
			value:  source.value.TextEdits[state.index],
			label:  label,
		}
		return resolution
	}
	resolution = textEditResolution{
		status: locatorResolutionAbsent,
		label:  label,
		actual: evaluationText("text edit is absent"),
	}
	return resolution
}

func resolveEntryEdit(state *repairEntryEditLocatorState, report cli.DiagnosticReport) entryEditResolution {
	return resolveEntryEditObservation(state, report, nil)
}

func resolveEntryEditInspection(state *repairEntryEditLocatorState, inspection cli.Inspection) entryEditResolution {
	return resolveEntryEditObservation(state, diagnosticReportFromInspection(inspection), &inspection)
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func resolveEntryEditObservation(state *repairEntryEditLocatorState, report cli.DiagnosticReport, inspection *cli.Inspection) entryEditResolution {
	var resolution entryEditResolution
	if state == nil || state.source == nil || state.index < 0 {
		resolution = entryEditResolution{
			status: locatorResolutionInvalid,
			label:  evaluationText("repair entry edit"),
			actual: observed(semanticSubjectRepairSource, observationInvalidLocator),
		}
		return resolution
	}
	var source repairSourceResolution
	if inspection != nil {
		source = resolveRepairSourceInspection(state.source, *inspection)
	} else {
		source = resolveRepairSource(state.source, report)
	}
	label := indexedSubject(evaluationText("entry edit"), state.index, source.label)
	if source.status != locatorResolutionFound {
		resolution = entryEditResolution{
			status: source.status,
			label:  label,
			actual: source.actual,
		}
		return resolution
	}
	if state.index < len(source.value.EntryEdits) {
		resolution = entryEditResolution{
			status: locatorResolutionFound,
			value:  source.value.EntryEdits[state.index],
			label:  label,
		}
		return resolution
	}
	resolution = entryEditResolution{
		status: locatorResolutionAbsent,
		label:  label,
		actual: evaluationText("entry edit is absent"),
	}
	return resolution
}

func repairPresentExpectation(locator RepairLocator) OutcomeExpectation {
	return repairExpectation{
		locator: locator.state,
		kind:    repairExpectationPresent,
	}
}

func repairPreservesMeaningExpectation(locator RepairLocator) OutcomeExpectation {
	return repairExpectation{
		locator: locator.state,
		kind:    repairExpectationPreservesMeaning,
	}
}

func repairSensitiveExpectation(locator RepairLocator) OutcomeExpectation {
	return repairExpectation{
		locator: locator.state,
		kind:    repairExpectationSensitive,
	}
}

func repairSourcesExactlyExpectation(locator RepairLocator, sources []InputSourceLocator) OutcomeExpectation {
	states := make([]*inputSourceLocatorState, len(sources))
	for index, source := range sources {
		states[index] = source.state
	}
	return repairExpectation{
		locator: locator.state,
		kind:    repairExpectationSourcesExactly,
		sources: states,
	}
}

func repairSourceTextEditCountExpectation(locator RepairSourceLocator, count int) OutcomeExpectation {
	return repairSourceExpectation{
		locator: locator.state,
		text:    true,
		count:   count,
	}
}

func repairSourceEntryEditCountExpectation(locator RepairSourceLocator, count int) OutcomeExpectation {
	return repairSourceExpectation{
		locator: locator.state,
		count:   count,
	}
}

func repairTextEditSpanExpectation(locator RepairTextEditLocator, span cli.InputSpan) OutcomeExpectation {
	return textEditExpectation{
		locator: locator.state,
		kind:    textEditExpectationSpan,
		span:    span,
	}
}

func repairTextEditReplacementExpectation(locator RepairTextEditLocator, replacement cli.InputReplacement) OutcomeExpectation {
	return textEditExpectation{
		locator:     locator.state,
		kind:        textEditExpectationReplacement,
		replacement: snapshotInputReplacement(replacement),
	}
}

func repairEntryEditRangeExpectation(locator RepairEntryEditLocator, start, end int) OutcomeExpectation {
	return entryEditExpectation{
		locator: locator.state,
		kind:    entryEditExpectationRange,
		start:   start,
		end:     end,
	}
}

func repairEntryEditEntriesExpectation(locator RepairEntryEditLocator, entries []cli.InputReplacement) OutcomeExpectation {
	snapshot := make([]cli.InputReplacement, len(entries))
	for i, entry := range entries {
		snapshot[i] = snapshotInputReplacement(entry)
	}
	return entryEditExpectation{
		locator: locator.state,
		kind:    entryEditExpectationEntries,
		entries: snapshot,
	}
}

func (repairsExactlyExpectation) diagnosticReportExpectation() {}

func (expectation repairsExactlyExpectation) evaluateDiagnosticReport(report cli.DiagnosticReport) evaluationResult {
	if len(expectation.states) != len(report.Repairs) {
		return disprovedResult(
			evaluationText("diagnostic repairs"),
			evaluationText(fmt.Sprintf("count=%d", len(expectation.states))),
			evaluationText(fmt.Sprintf("count=%d", len(report.Repairs))),
		)
	}
	for index, state := range expectation.states {
		resolved := resolveRepair(state, report)
		if resolved.status != locatorResolutionFound {
			return disprovedResult(
				evaluationText("diagnostic repairs"),
				evaluationText(fmt.Sprintf("repair %d resolvable", index)),
				resolved.actual,
			)
		}
		if resolved.index != index {
			return disprovedResult(
				evaluationText("diagnostic repairs"),
				evaluationText(fmt.Sprintf("repair-index[%d]=%d", index, index)),
				evaluationText(fmt.Sprintf("repair-index[%d]=%d", index, resolved.index)),
			)
		}
	}
	return provedResult()
}

func (repairExpectation) inspectionExpectation() {}

func (repairExpectation) completionExpectation() {}

func (repairExpectation) diagnosticReportExpectation() {}

func (expectation repairExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return expectation.evaluate(resolveRepairInspection(expectation.locator, inspection), diagnosticReportFromInspection(inspection), &inspection)
}

func (expectation repairExpectation) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	report := diagnosticReportFromCompletion(result)
	return expectation.evaluate(resolveRepair(expectation.locator, report), report, nil)
}

func (expectation repairExpectation) evaluateDiagnosticReport(report cli.DiagnosticReport) evaluationResult {
	return expectation.evaluate(resolveRepair(expectation.locator, report), report, nil)
}

func (expectation repairExpectation) evaluate(resolved repairResolution, report cli.DiagnosticReport, inspection *cli.Inspection) evaluationResult {
	var expected evaluationField = evaluationText("present")
	switch expectation.kind {
	case repairExpectationPreservesMeaning:
		expected = evaluationText("preserves meaning")
	case repairExpectationSensitive:
		expected = semanticRequirementSensitive
	case repairExpectationSourcesExactly:
		expected = evaluationText("exact repair sources in authoritative order")
	default:
	}
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.label, expected, resolved.actual)
	}
	switch expectation.kind {
	case repairExpectationPresent:
		return provedResult()
	case repairExpectationPreservesMeaning:
		if resolved.value.PreservesMeaning {
			return provedResult()
		}
		return disprovedResult(resolved.label, expected, redactedOutcomeField(resolved.value.Sensitive, evaluationText("false")))
	case repairExpectationSensitive:
		if resolved.value.Sensitive {
			return provedResult()
		}
		return disprovedResult(resolved.label, expected, evaluationText("not sensitive"))
	case repairExpectationSourcesExactly:
		actual := make([]cli.InputSourceID, len(resolved.value.Sources))
		for index, source := range resolved.value.Sources {
			actual[index] = source.Source
		}
		expectedIDs := make([]cli.InputSourceID, len(expectation.sources))
		for index, state := range expectation.sources {
			var source sourceResolution
			if inspection != nil {
				source = resolveSourceInspection(state, *inspection)
			} else {
				source = resolveSource(state, report)
			}
			if source.status != locatorResolutionFound {
				return outcomeForValueStatus(source.status, source.label, expected, source.actual)
			}
			expectedIDs[index] = source.value.ID
		}
		if slices.Equal(actual, expectedIDs) {
			return provedResult()
		}
		if resolved.value.Sensitive {
			return disprovedResult(resolved.label, expected, evaluationText("value differs (redacted)"))
		}
		return orderedCollectionMismatch(resolved.label, "sources", expectedIDs, actual)
	default:
		panic("clitest: invalid repair expectation")
	}
}

func (repairSourceExpectation) inspectionExpectation() {}

func (repairSourceExpectation) completionExpectation() {}

func (repairSourceExpectation) diagnosticReportExpectation() {}

func (e repairSourceExpectation) expected() evaluationField {
	if e.text {
		return evaluationText(fmt.Sprintf("text edit count %d", e.count))
	}
	return evaluationText(fmt.Sprintf("entry edit count %d", e.count))
}

func (e repairSourceExpectation) evaluateResolution(r repairSourceResolution) evaluationResult {
	if r.status != locatorResolutionFound {
		return outcomeForValueStatus(r.status, r.label, e.expected(), r.actual)
	}
	actual := len(r.value.EntryEdits)
	name := "entry edit"
	if e.text {
		actual = len(r.value.TextEdits)
		name = "text edit"
	}
	if actual == e.count {
		return provedResult()
	}
	return disprovedResult(r.label, e.expected(), evaluationText(fmt.Sprintf("%s count %d", name, actual)))
}

func (e repairSourceExpectation) evaluateInspection(i cli.Inspection, _ error) evaluationResult {
	return e.evaluateResolution(resolveRepairSourceInspection(e.locator, i))
}

func (e repairSourceExpectation) evaluateCompletion(c cli.Completion, _ error) evaluationResult {
	return e.evaluateResolution(resolveRepairSource(e.locator, diagnosticReportFromCompletion(c)))
}

func (e repairSourceExpectation) evaluateDiagnosticReport(r cli.DiagnosticReport) evaluationResult {
	return e.evaluateResolution(resolveRepairSource(e.locator, r))
}

func (textEditExpectation) inspectionExpectation() {}

func (textEditExpectation) completionExpectation() {}

func (textEditExpectation) diagnosticReportExpectation() {}

func (e textEditExpectation) evaluate(r textEditResolution) evaluationResult {
	var expected evaluationField = evaluationText("text edit")
	switch e.kind {
	case textEditExpectationSpan:
		expected = inputSpanField{value: e.span}
	case textEditExpectationReplacement:
		expected = evaluationText("exact replacement")
	default:
	}
	if r.status != locatorResolutionFound {
		return outcomeForValueStatus(r.status, r.label, expected, r.actual)
	}
	switch e.kind {
	case textEditExpectationSpan:
		if r.value.Span == e.span {
			return provedResult()
		}
		return disprovedResult(r.label, expected, inputSpanField{value: r.value.Span})
	case textEditExpectationReplacement:
		if slices.EqualFunc(r.value.Replacement, e.replacement, equalInputReplacementPart) {
			return provedResult()
		}
		return disprovedResult(r.label, expected, evaluationText("replacement differs"))
	default:
		panic("clitest: invalid text-edit expectation")
	}
}

func (e textEditExpectation) evaluateInspection(i cli.Inspection, _ error) evaluationResult {
	return e.evaluate(resolveTextEditInspection(e.locator, i))
}

func (e textEditExpectation) evaluateCompletion(c cli.Completion, _ error) evaluationResult {
	return e.evaluate(resolveTextEdit(e.locator, diagnosticReportFromCompletion(c)))
}

func (e textEditExpectation) evaluateDiagnosticReport(r cli.DiagnosticReport) evaluationResult {
	return e.evaluate(resolveTextEdit(e.locator, r))
}

func (entryEditExpectation) inspectionExpectation() {}

func (entryEditExpectation) completionExpectation() {}

func (entryEditExpectation) diagnosticReportExpectation() {}

func (e entryEditExpectation) evaluate(r entryEditResolution) evaluationResult {
	expected := evaluationText("entry edit")
	switch e.kind {
	case entryEditExpectationRange:
		expected = evaluationText(fmt.Sprintf("range [%d,%d)", e.start, e.end))
	case entryEditExpectationEntries:
		expected = evaluationText("exact replacement entries")
	default:
	}
	if r.status != locatorResolutionFound {
		return outcomeForValueStatus(r.status, r.label, expected, r.actual)
	}
	switch e.kind {
	case entryEditExpectationRange:
		if r.value.Start == e.start && r.value.End == e.end {
			return provedResult()
		}
		return disprovedResult(r.label, expected, evaluationText(fmt.Sprintf("range [%d,%d)", r.value.Start, r.value.End)))
	case entryEditExpectationEntries:
		if equalReplacements(r.value.Entries, e.entries) {
			return provedResult()
		}
		return disprovedResult(r.label, expected, evaluationText("replacement entries differ"))
	default:
		panic("clitest: invalid entry-edit expectation")
	}
}

func (e entryEditExpectation) evaluateInspection(i cli.Inspection, _ error) evaluationResult {
	return e.evaluate(resolveEntryEditInspection(e.locator, i))
}

func (e entryEditExpectation) evaluateCompletion(c cli.Completion, _ error) evaluationResult {
	return e.evaluate(resolveEntryEdit(e.locator, diagnosticReportFromCompletion(c)))
}

func (e entryEditExpectation) evaluateDiagnosticReport(r cli.DiagnosticReport) evaluationResult {
	return e.evaluate(resolveEntryEdit(e.locator, r))
}
