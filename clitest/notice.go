package clitest

import (
	"fmt"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
)

// NoticeSubject is the closed set of semantic subjects that can identify a
// Notice expectation target.
type NoticeSubject interface{ noticeSubject() }

// NoticeLocator identifies one structured Notice for clitest semantic expectation. Its
// zero value is invalid.
type NoticeLocator struct {
	_     [0]func()
	state *noticeLocatorState
}

// NoticesExactly checks the complete ordered Notice collection in Inspection.
// Exact absence or membership is not inferred from incomplete observation.
func NoticesExactly(notices ...NoticeLocator) InspectionExpectation {
	return noticesExactlyExpectation(notices...)
}

// Notice returns an expected coordinate for the unique structured Notice with code in an Inspection observation.
// Duplicate matches are ambiguous, and truncation prevents absence from being assumed.
func Notice(code cli.NoticeCode) NoticeLocator { return noticeCodeLocator(code) }

// NoticeAt returns an expected coordinate for the Notice at index in Inspection.Notices authoritative order. An
// out-of-range coordinate in a truncated Inspection remains not provable.
func NoticeAt(index int) NoticeLocator { return noticeIndexLocator(index) }

// Message returns an Inspection expectation that the located Notice message exactly equals expected.
// When the Notice is sensitive, mismatch reporting redacts both expected and observed presentation text.
func (locator NoticeLocator) Message(expected string) InspectionExpectation {
	value := noticePresentationExpectationValue{
		locator:  locator.state,
		property: noticePresentationMessage,
		expected: expected,
	}
	return value
}

// Reason returns an Inspection expectation that the located Notice reason exactly equals expected.
// When the Notice is sensitive, mismatch reporting redacts both expected and observed presentation text.
func (locator NoticeLocator) Reason(expected string) InspectionExpectation {
	value := noticePresentationExpectationValue{
		locator:  locator.state,
		property: noticePresentationReason,
		expected: expected,
	}
	return value
}

// Hint returns an Inspection expectation that the located Notice hint exactly equals expected.
// When the Notice is sensitive, mismatch reporting redacts both expected and observed presentation text.
func (locator NoticeLocator) Hint(expected string) InspectionExpectation {
	value := noticePresentationExpectationValue{
		locator:  locator.state,
		property: noticePresentationHint,
		expected: expected,
	}
	return value
}

// SubjectRedacted returns an Inspection expectation that the located Notice is sensitive and its public Subject
// presentation is empty.
// It checks the redacted public observation rather than attempting to reconstruct the hidden subject.
func (locator NoticeLocator) SubjectRedacted() InspectionExpectation {
	value := noticePresentationExpectationValue{
		locator:  locator.state,
		property: noticePresentationSubjectRedacted,
	}
	return value
}

// On refines a Notice coordinate by the structured public subject identity carried by the Inspection observation. It
// does not derive subject identity from Notice presentation text.
func (locator NoticeLocator) On(subject NoticeSubject) NoticeLocator {
	return noticeOn(locator, subject)
}

// Advisory refines the Notice coordinate to the exact AdvisoryCode relationship carried by the Notice observation.
func (locator NoticeLocator) Advisory(code cli.AdvisoryCode) NoticeLocator {
	return noticeAdvisory(locator, code)
}

// RepairAt returns the repair at index in the located Notice's ordered RepairIndexes relationship in the same
// Inspection observation.
func (locator NoticeLocator) RepairAt(index int) RepairLocator {
	return noticeRepairAtLocator(locator, index)
}

// Present returns an Inspection expectation that the Notice coordinate resolves in the returned Inspection.
func (locator NoticeLocator) Present() InspectionExpectation {
	return noticePresentExpectation(locator)
}

// Code returns an Inspection expectation that the located Notice carries exactly code.
func (locator NoticeLocator) Code(code cli.NoticeCode) InspectionExpectation {
	return noticeCodeExpectation(locator, code)
}

// Absent returns an Inspection expectation that the Notice coordinate is absent from a complete Inspection. A truncated
// Inspection cannot establish that absence.
func (locator NoticeLocator) Absent() InspectionExpectation { return noticeAbsentExpectation(locator) }

// Severity returns an Inspection expectation that the located Notice carries exactly severity.
func (locator NoticeLocator) Severity(severity cli.NoticeSeverity) InspectionExpectation {
	return noticeSeverityExpectation(locator, severity)
}

// Suppression returns an Inspection expectation that the located Notice carries exactly status as its suppression
// state.
func (locator NoticeLocator) Suppression(disposition cli.NoticeSuppressionDisposition) InspectionExpectation {
	return noticeSuppressionExpectation(locator, disposition)
}

// Sensitive returns an Inspection expectation that the located Notice is marked sensitive. It checks the structured
// sensitivity marker and does not require or expose protected presentation text.
func (locator NoticeLocator) Sensitive() InspectionExpectation {
	return noticeSensitiveExpectation(locator)
}

// ReplacedBy returns an Inspection expectation that the Notice's structured Replacement.Target equals the declaration
// selected by replacement under the same Inspection carrier. The expected locator remains test configuration, not
// checknance evidence.
func (locator NoticeLocator) ReplacedBy(replacement DeclarationIdentityLocator) InspectionExpectation {
	return noticeReplacementExpectation(locator, replacement)
}

// SuppressedByExactly returns an Inspection expectation that the Notice's suppression relationships exactly equal
// suppressions in authoritative order, after each expected suppression resolves in the same Inspection.
func (locator NoticeLocator) SuppressedByExactly(suppressions ...NoticeSuppressionLocator) InspectionExpectation {
	return noticeSuppressedByExpectation(locator, suppressions...)
}

// RepairCount returns an Inspection expectation that the Notice carries exactly count repair relationships.
func (locator NoticeLocator) RepairCount(count int) InspectionExpectation {
	return noticeRepairCountExpectation(locator, count)
}

type noticePresentationProperty uint8

const (
	noticePresentationInvalid noticePresentationProperty = iota
	noticePresentationMessage
	noticePresentationReason
	noticePresentationHint
	noticePresentationSubjectRedacted
)

type noticePresentationExpectationValue struct {
	locator  *noticeLocatorState
	property noticePresentationProperty
	expected string
}

type noticeResolution struct {
	status locatorResolutionStatus
	value  cli.NoticeInspection
	index  int
	label  evaluationField
	actual evaluationField
}

type noticeSubjectResolution struct {
	command     cli.CommandID
	declaration cli.DeclarationIdentity
}

type noticeExpectationKind uint8

const (
	noticeExpectationInvalid noticeExpectationKind = iota
	noticeExpectationPresent
	noticeExpectationAbsent
	noticeExpectationCode
	noticeExpectationSeverity
	noticeExpectationSuppression
	noticeExpectationSensitive
	noticeExpectationReplacement
	noticeExpectationSuppressedBy
	noticeExpectationRepairCount
)

type noticeExpectation struct {
	locator *noticeLocatorState
	kind    noticeExpectationKind

	code         cli.NoticeCode
	severity     cli.NoticeSeverity
	suppression  cli.NoticeSuppressionDisposition
	replacement  DeclarationIdentityLocator
	suppressions []*noticeSuppressionLocatorState
	count        int
}

type noticesExactlyExpectationValue struct{ states []*noticeLocatorState }

type noticeLocatorMode uint8

const (
	noticeLocatorInvalid noticeLocatorMode = iota
	noticeLocatorCode
	noticeLocatorIndex
)

type noticeLocatorState struct {
	mode  noticeLocatorMode
	code  cli.NoticeCode
	index int

	advisorySet bool
	advisory    cli.AdvisoryCode
	subjectSet  bool
	subject     NoticeSubject
}

func noticeCodeLocator(code cli.NoticeCode) NoticeLocator {
	state := &noticeLocatorState{
		mode: noticeLocatorCode,
		code: code,
	}
	locator := NoticeLocator{
		state: state,
	}
	return locator
}

func noticeIndexLocator(index int) NoticeLocator {
	state := &noticeLocatorState{
		mode:  noticeLocatorIndex,
		index: index,
	}
	locator := NoticeLocator{
		state: state,
	}
	return locator
}

func noticeOn(notice NoticeLocator, subject NoticeSubject) NoticeLocator {
	if notice.state == nil {
		return NoticeLocator{}
	}
	state := *notice.state
	state.subjectSet = true
	state.subject = subject
	locator := NoticeLocator{
		state: &state,
	}
	return locator
}

func noticeAdvisory(notice NoticeLocator, code cli.AdvisoryCode) NoticeLocator {
	if notice.state == nil {
		return NoticeLocator{}
	}
	state := *notice.state
	state.advisorySet = true
	state.advisory = code
	locator := NoticeLocator{
		state: &state,
	}
	return locator
}

func resolveNoticeSubject(subject NoticeSubject, inspection cli.Inspection) (noticeSubjectResolution, locatorResolutionStatus, evaluationField) {
	switch subject := subject.(type) {
	case CommandLocator:
		resolved := resolveInspectionCommandLocator(subject.state, inspection)
		if resolved.status != locatorResolutionFound {
			return noticeSubjectResolution{}, resolved.status, resolved.detail
		}
		resolution := noticeSubjectResolution{
			command: resolved.value.ID,
		}
		return resolution, locatorResolutionFound, evaluationText("")
	case CommandEntryLocator, ActionEntryLocator, FlagLocator, FlagSpellingLocator, ArgumentFormLocator, ArgumentTermLocator, EnvironmentInputLocator, ValueDomainEntryLocator:
		identityLocator, ok := subject.(DeclarationIdentityLocator)
		if !ok {
			panic("clitest: declaration notice subject lacks declaration identity")
		}
		identity, status, detail := resolveExpectedDeclarationIdentity(identityLocator, nil, nil, inspection)
		if status != locatorResolutionFound {
			return noticeSubjectResolution{}, status, detail
		}
		resolution := noticeSubjectResolution{
			declaration: identity,
		}
		return resolution, locatorResolutionFound, evaluationText("")
	default:
		return noticeSubjectResolution{}, locatorResolutionInvalid, evaluationText("invalid notice subject locator")
	}
}

func noticeLabel(state *noticeLocatorState) evaluationField {
	if state == nil {
		return evaluationText(semanticLabelNotice)
	}
	if state.mode == noticeLocatorIndex {
		return subjectAt(semanticSubjectNotice, state.index)
	}
	return subjectNamed(semanticSubjectNotice, string(state.code))
}

func resolveNotice(state *noticeLocatorState, inspection cli.Inspection) noticeResolution {
	var resolution noticeResolution
	if state == nil || (state.mode != noticeLocatorCode && state.mode != noticeLocatorIndex) {
		resolution = noticeResolution{
			status: locatorResolutionInvalid,
			label:  semanticSubjectNotice,
			actual: observed(semanticSubjectNotice, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == noticeLocatorIndex && state.index < 0 {
		resolution = noticeResolution{
			status: locatorResolutionInvalid,
			label:  noticeLabel(state),
			actual: observed(semanticSubjectNotice, observationInvalidLocator),
		}
		return resolution
	}

	var subject noticeSubjectResolution
	if state.subjectSet {
		resolved, status, detail := resolveNoticeSubject(state.subject, inspection)
		if status != locatorResolutionFound {
			resolution = noticeResolution{
				status: status,
				label:  noticeLabel(state),
				actual: detail,
			}
			return resolution
		}
		subject = resolved
	}

	matches := make([]int, 0, 1)
	switch state.mode {
	case noticeLocatorIndex:
		switch {
		case state.index < len(inspection.Notices):
			matches = append(matches, state.index)
		case inspection.Truncated:
			resolution = noticeResolution{
				status: locatorResolutionNotProvable,
				label:  noticeLabel(state),
				actual: observed(semanticSubjectInspection, observationTruncated),
			}
			return resolution
		default:
			resolution = noticeResolution{
				status: locatorResolutionAbsent,
				label:  noticeLabel(state),
				actual: evaluationText("notice is absent"),
			}
			return resolution
		}
	case noticeLocatorCode:
		for index, notice := range inspection.Notices {
			if notice.Code == state.code {
				matches = append(matches, index)
			}
		}
	default:
		panic("clitest: invalid notice locator mode")
	}

	filtered := matches[:0]
	for _, index := range matches {
		notice := inspection.Notices[index]
		if state.advisorySet && notice.AdvisoryCode != state.advisory {
			continue
		}
		if state.subjectSet {
			if subject.declaration.Kind != cli.DeclarationKindUnknown {
				if notice.Declaration != subject.declaration {
					continue
				}
			} else if notice.Command != subject.command {
				continue
			}
		}
		filtered = append(filtered, index)
	}
	matches = filtered

	label := noticeLabel(state)
	switch len(matches) {
	case 1:
		resolution = noticeResolution{
			status: locatorResolutionFound,
			value:  inspection.Notices[matches[0]],
			index:  matches[0],
			label:  label,
		}
		return resolution
	case 0:
		if inspection.Truncated {
			resolution = noticeResolution{
				status: locatorResolutionNotProvable,
				label:  label,
				actual: observed(semanticSubjectInspection, observationTruncated),
			}
			return resolution
		}
		resolution = noticeResolution{
			status: locatorResolutionAbsent,
			label:  label,
			actual: evaluationText("notice is absent"),
		}
		return resolution
	default:
		resolution = noticeResolution{
			status: locatorResolutionAmbiguous,
			label:  label,
			actual: collectionCardinalityField{
				name:  "matches notice indexes",
				count: len(matches),
			},
		}
		return resolution
	}
}

func noticePresentExpectation(locator NoticeLocator) InspectionExpectation {
	expectation := noticeExpectation{
		locator: locator.state,
		kind:    noticeExpectationPresent,
	}
	return expectation
}

func noticeAbsentExpectation(locator NoticeLocator) InspectionExpectation {
	expectation := noticeExpectation{
		locator: locator.state,
		kind:    noticeExpectationAbsent,
	}
	return expectation
}

func noticeCodeExpectation(locator NoticeLocator, code cli.NoticeCode) InspectionExpectation {
	expectation := noticeExpectation{
		locator: locator.state,
		kind:    noticeExpectationCode,
		code:    code,
	}
	return expectation
}

func noticeSeverityExpectation(locator NoticeLocator, severity cli.NoticeSeverity) InspectionExpectation {
	expectation := noticeExpectation{
		locator:  locator.state,
		kind:     noticeExpectationSeverity,
		severity: severity,
	}
	return expectation
}

func noticeSuppressionExpectation(locator NoticeLocator, status cli.NoticeSuppressionDisposition) InspectionExpectation {
	expectation := noticeExpectation{
		locator:     locator.state,
		kind:        noticeExpectationSuppression,
		suppression: status,
	}
	return expectation
}

func noticeSensitiveExpectation(locator NoticeLocator) InspectionExpectation {
	expectation := noticeExpectation{
		locator: locator.state,
		kind:    noticeExpectationSensitive,
	}
	return expectation
}

func noticeReplacementExpectation(locator NoticeLocator, replacement DeclarationIdentityLocator) InspectionExpectation {
	expectation := noticeExpectation{
		locator:     locator.state,
		kind:        noticeExpectationReplacement,
		replacement: replacement,
	}
	return expectation
}

func noticeSuppressedByExpectation(locator NoticeLocator, expected ...NoticeSuppressionLocator) InspectionExpectation {
	states := make([]*noticeSuppressionLocatorState, len(expected))
	for index, value := range expected {
		states[index] = value.state
	}
	expectation := noticeExpectation{
		locator:      locator.state,
		kind:         noticeExpectationSuppressedBy,
		suppressions: states,
	}
	return expectation
}

func noticeRepairCountExpectation(locator NoticeLocator, count int) InspectionExpectation {
	expectation := noticeExpectation{
		locator: locator.state,
		kind:    noticeExpectationRepairCount,
		count:   count,
	}
	return expectation
}

func noticesExactlyExpectation(locators ...NoticeLocator) InspectionExpectation {
	states := make([]*noticeLocatorState, len(locators))
	for index, locator := range locators {
		states[index] = locator.state
	}
	value := noticesExactlyExpectationValue{
		states: states,
	}
	return value
}

func noticeRepairAtLocator(parent NoticeLocator, index int) RepairLocator {
	state := &repairLocatorState{
		mode:   repairLocatorNoticeIndex,
		notice: parent.state,
		index:  index,
	}
	locator := RepairLocator{
		state: state,
	}
	return locator
}

func (noticeMatchField) evaluationField() {}

func (noticePresentationExpectationValue) inspectionExpectation() {}

func (expectation noticePresentationExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveNotice(expectation.locator, inspection)
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.label, expectation.expectedField(), resolved.actual)
	}
	if expectation.property == noticePresentationSubjectRedacted {
		if resolved.value.Sensitive && resolved.value.Subject == "" {
			return provedResult()
		}
		return disprovedResult(
			resolved.label,
			evaluationText("subject redacted"),
			evaluationText(fmt.Sprintf("sensitive=%t; subject-redacted=%t", resolved.value.Sensitive, resolved.value.Subject == "")),
		)
	}
	field, actual := expectation.textValue(resolved.value)
	if actual == expectation.expected {
		return provedResult()
	}
	if resolved.value.Sensitive {
		redacted := evaluationText(field + "=<redacted>")
		return disprovedResult(resolved.label, redacted, redacted)
	}
	return disprovedResult(
		resolved.label,
		property(evaluationText(field), quotedTextField(expectation.expected)),
		property(evaluationText(field), quotedTextField(actual)),
	)
}

func (expectation noticePresentationExpectationValue) expectedField() evaluationField {
	switch expectation.property {
	case noticePresentationMessage:
		return evaluationText("message")
	case noticePresentationReason:
		return evaluationText("reason")
	case noticePresentationHint:
		return evaluationText("hint")
	case noticePresentationSubjectRedacted:
		return evaluationText("subject redacted")
	default:
		return evaluationText("valid notice presentation expectation")
	}
}

func (expectation noticePresentationExpectationValue) textValue(notice cli.NoticeInspection) (string, string) {
	switch expectation.property {
	case noticePresentationMessage:
		return "message", notice.Message
	case noticePresentationReason:
		return "reason", notice.Reason
	case noticePresentationHint:
		return "hint", notice.Hint
	default:
		return semanticLabelNotice, ""
	}
}

func (noticeExpectation) inspectionExpectation() {}

func (expectation noticeExpectation) expected() evaluationField {
	switch expectation.kind {
	case noticeExpectationPresent:
		return evaluationText("present")
	case noticeExpectationAbsent:
		return semanticRequirementAbsent
	case noticeExpectationCode:
		return property(evaluationText("code"), evaluationText(expectation.code))
	case noticeExpectationSeverity:
		return evaluationText(fmt.Sprintf("severity %v", expectation.severity))
	case noticeExpectationSuppression:
		return evaluationText(fmt.Sprintf("suppression %v", expectation.suppression))
	case noticeExpectationSensitive:
		return semanticRequirementSensitive
	case noticeExpectationReplacement:
		return semanticRequirementRequestedReplacement
	case noticeExpectationSuppressedBy:
		return evaluationText("suppressions exactly")
	case noticeExpectationRepairCount:
		return evaluationText(fmt.Sprintf("repair count %d", expectation.count))
	default:
		panic("clitest: invalid Notice expectation")
	}
}

func (expectation noticeExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveNotice(expectation.locator, inspection)
	if expectation.kind == noticeExpectationAbsent {
		switch resolved.status {
		case locatorResolutionAbsent:
			return provedResult()
		case locatorResolutionNotProvable:
			return notProvableResult(resolved.label, expectation.expected(), resolved.actual)
		case locatorResolutionFound:
			return disprovedResult(resolved.label, expectation.expected(), evaluationText("notice is present"))
		default:
			return outcomeForValueStatus(resolved.status, resolved.label, expectation.expected(), resolved.actual)
		}
	}
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.label, expectation.expected(), resolved.actual)
	}
	if expectation.kind == noticeExpectationPresent {
		return provedResult()
	}

	notice := resolved.value
	switch expectation.kind {
	case noticeExpectationCode:
		if notice.Code == expectation.code {
			return provedResult()
		}
		return disprovedResult(resolved.label, expectation.expected(), property(evaluationText("code"), evaluationText(notice.Code)))
	case noticeExpectationSeverity:
		if notice.Severity == expectation.severity {
			return provedResult()
		}
		return disprovedResult(resolved.label, expectation.expected(), evaluationText(fmt.Sprintf("severity %v", notice.Severity)))
	case noticeExpectationSuppression:
		if notice.Suppression == expectation.suppression {
			return provedResult()
		}
		return disprovedResult(resolved.label, expectation.expected(), evaluationText(fmt.Sprintf("suppression %v", notice.Suppression)))
	case noticeExpectationSensitive:
		if notice.Sensitive {
			return provedResult()
		}
		return disprovedResult(resolved.label, expectation.expected(), evaluationText("sensitive=false"))
	case noticeExpectationReplacement:
		identity, status, detail := resolveExpectedDeclarationIdentity(expectation.replacement, nil, nil, inspection)
		if status == locatorResolutionNotProvable {
			return notProvableResult(resolved.label, expectation.expected(), detail)
		}
		if status != locatorResolutionFound {
			return disprovedResult(resolved.label, expectation.expected(), nonempty(detail, evaluationText("invalid replacement locator")))
		}
		if notice.Replacement == nil {
			return disprovedResult(resolved.label, expectation.expected(), evaluationText("replacement is absent"))
		}
		if notice.Replacement.Target == identity {
			return provedResult()
		}
		field := declarationIdentityField{
			value: notice.Replacement.Target,
		}
		return disprovedResult(resolved.label, expectation.expected(), field)
	case noticeExpectationSuppressedBy:
		expected := make([]cli.NoticeSuppressionLocator, len(expectation.suppressions))
		for index, state := range expectation.suppressions {
			entry := resolveInspectionNoticeSuppressionLocator(state, inspection)
			if entry.status == locatorResolutionNotProvable {
				return notProvableResult(resolved.label, expectation.expected(), entry.detail)
			}
			if entry.status != locatorResolutionFound {
				return disprovedResult(resolved.label, expectation.expected(), nonempty(entry.detail, evaluationText("notice-suppression locator is not resolvable")))
			}
			expected[index] = cli.NoticeSuppressionLocator{
				Command:          entry.value.Command,
				DeclarationIndex: entry.value.DeclarationIndex,
			}
		}
		if slices.Equal(notice.SuppressedBy, expected) {
			return provedResult()
		}
		return orderedCollectionMismatch(resolved.label, "notice suppressions", expected, notice.SuppressedBy)
	case noticeExpectationRepairCount:
		if len(notice.RepairIndexes) == expectation.count {
			return provedResult()
		}
		return disprovedResult(resolved.label, expectation.expected(), evaluationText(fmt.Sprintf("repair count %d", len(notice.RepairIndexes))))
	default:
		panic("clitest: invalid Notice expectation")
	}
}

func (noticesExactlyExpectationValue) inspectionExpectation() {}

func (expectation noticesExactlyExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	if inspection.Truncated {
		return notProvableResult(evaluationText("notices"), evaluationText("exact ordered collection"), observed(semanticSubjectInspection, observationTruncated))
	}
	indexes := make([]int, len(expectation.states))
	for index, state := range expectation.states {
		resolved := resolveNotice(state, inspection)
		if resolved.status != locatorResolutionFound {
			return outcomeForValueStatus(resolved.status, resolved.label, evaluationText("notice in exact collection"), resolved.actual)
		}
		indexes[index] = resolved.index
	}
	expected := make([]int, len(inspection.Notices))
	for index := range expected {
		expected[index] = index
	}
	if slices.Equal(indexes, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(evaluationText("notices"), "notice indexes", indexes, expected)
}
