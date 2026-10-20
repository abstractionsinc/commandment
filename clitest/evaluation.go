package clitest

import (
	"fmt"
	"slices"
	"time"

	"github.com/abstractionsinc/commandment/cli"
)

type inspectionResultState struct {
	observation cli.Inspection
	err         error
}

type completionResultState struct {
	observation cli.Completion
	err         error
}

type schemaResultState struct{ observation cli.Schema }

type referenceResultState struct {
	observation cli.Reference
	err         error
}

type diagnosticsResultState struct{ observation cli.DiagnosticReport }

type evaluationStatus uint8

const (
	evaluationStatusInvalid evaluationStatus = iota
	evaluationProved
	evaluationDisproved
	evaluationNotProvable
)

const (
	semanticNameCandidate                  = "candidate"
	semanticNameDependencies               = "dependencies"
	semanticNameEnvironment                = "environment"
	semanticNameExamples                   = "examples"
	semanticNameHelpCommandGroupEntries    = "help command-group entries"
	semanticNameMatchesValueDomainIndexes  = "matches value-domain indexes"
	semanticNameMatchesAdvisoryCodeIndexes = "matches advisory-code indexes"
	semanticNameMatchesRuleCodeRecords     = "matches rule-code records"
	semanticNameMatchesAdvisoryCodeRecords = "matches advisory-code records"
	semanticNameOriginIDs                  = "origin IDs"
	semanticLabelBlocked                   = "blocked"
	semanticLabelNotice                    = "notice"
)

type semanticSubject uint8

const (
	semanticSubjectInvalid semanticSubject = iota
	semanticSubjectArgumentForm
	semanticSubjectDelegation
	semanticSubjectDeclaration
	semanticSubjectValue
	semanticSubjectFlagSpelling
	semanticSubjectCommandDefault
	semanticSubjectCondition
	semanticSubjectRule
	semanticSubjectAdvisory
	semanticSubjectNoticeSuppression
	semanticSubjectSignalPolicy
	semanticSubjectConditionChild
	semanticSubjectConditionOperand
	semanticSubjectFixedValue
	semanticSubjectKeyedEntry
	semanticSubjectRuleOperand
	semanticSubjectRuleOperandLiteral
	semanticSubjectSignalAction
	semanticSubjectActionEntry
	semanticSubjectCommandEntry
	semanticSubjectShortcutDefault
	semanticSubjectShortcutDefaultValue
	semanticSubjectValueDomainUse
	semanticSubjectTokenLanguage
	semanticSubjectCompletionParent
	semanticSubjectCommand
	semanticSubjectDiagnostic
	semanticSubjectSignalActionFailure
	semanticSubjectRepairSource
	semanticSubjectDiagnosticReport
	semanticSubjectDiagnosticSpanSource
	semanticSubjectDiagnosticSpan
	semanticSubjectInputSource
	semanticSubjectInputOrigin
	semanticSubjectHelpCommandGroup
	semanticSubjectLocator
	semanticSubjectStaticLiteral
	semanticSubjectHelpFlagGroup
	semanticSubjectArgumentTerm
	semanticSubjectValueDomainEntry
	semanticSubjectInspection
	semanticSubjectSignalEndpoint
	semanticSubjectRawInput
	semanticSubjectNotice
	semanticSubjectCommandProvider
	semanticSubjectStructuredFailureDetail
	semanticSubjectGrammarProfileSet
	semanticSubjectGrammarProfile
	semanticSubjectFlag
	semanticSubjectEnvironmentInput
	semanticSubjectValueDomain
	semanticSubjectParentCommand
	semanticSubjectChildCommand
	semanticSubjectFact
	semanticSubjectRelatedSignalEndpoint
	semanticSubjectOrigin
)

type semanticRequirement uint8

const (
	semanticRequirementInvalid semanticRequirement = iota
	semanticRequirementSensitive
	semanticRequirementAbsent
	semanticRequirementRequestedReplacement
	semanticRequirementNoCanonicalRepresentation
	semanticRequirementExactCanonicalTokens
	semanticRequirementExactOrigins
)

type observationCondition uint8

const (
	observationConditionInvalid observationCondition = iota
	observationAbsent
	observationInvalidLocator
	observationInvalidOrForeignLocator
	observationLocatorDoesNotIdentify
	observationTruncated
	observationNotDeclaration
	observationMustBeFlagOrArgumentTerm
	observationNoRuleFailure
	observationAbsentOrAmbiguous
	observationDifferent
	observationNoSignalEndpointRelation
	observationInvalidLocatorInExpectedCollection
)

type semanticObservation struct {
	subject   semanticSubject
	condition observationCondition
}

type evaluationField interface {
	evaluationField()
}

type evaluationText string

type quotedTextField string

type evaluationBool bool

type releaseLifecycleField struct{ value cli.ReleaseLifecycle }

type descriptionField struct{ value cli.Description }

type helpLabelsField struct{ value cli.HelpLabels }

type valueTypeField struct{ value cli.ValueType }

type countRangeField struct{ value cli.CountRange }

type flagSpellingField struct{ value cli.FlagSpelling }

type stringCollectionField struct{ values []string }

type collectionCardinalityField struct {
	name  string
	count int
}

type orderedCollectionField struct {
	name          string
	length        int
	mismatchIndex int
	expected      bool
}

type declarationIdentityField struct{ value cli.DeclarationIdentity }

type completionRangeCollectionField struct{ values []cli.CompletionRange }

type runeCollectionField struct{ values []rune }

type completionMessageField struct {
	index int
	value cli.CompletionMessage
}

type diagnosticCodeCollectionField struct{ values []cli.DiagnosticCode }

type resolutionStateCollectionField struct{ values []cli.ResolutionState }

type completionCursorField struct{ value cli.CompletionCursor }

type commandPathField struct{ values []string }

type commandIDField struct{ value cli.CommandID }

type inputSpanField struct{ value cli.InputSpan }

type completionSpanField struct{ value cli.CompletionSpan }

type signalSenderField struct{ value cli.SignalSender }

type timeField struct{ value time.Time }

type noticeMatchField struct{ value cli.NoticeMatch }

type semanticProperty struct {
	name  evaluationField
	value evaluationField
}

type qualifiedSubject struct {
	base      evaluationField
	qualifier evaluationField
}

type indexedRelationSubject struct {
	noun   evaluationField
	index  int
	parent evaluationField
}

type namedSubject struct {
	base evaluationField
	name string
}

type indexedSubjectField struct {
	base  evaluationField
	index int
}

type prefixedDetail struct {
	prefix string
	detail evaluationField
}

type comparisonField struct {
	name      evaluationField
	separator string
	value     evaluationField
}

type evaluationSummary struct {
	outcomes []evaluationResult
}

type evaluationResult struct {
	status   evaluationStatus
	subject  evaluationField
	expected evaluationField
	observed evaluationField
}

type inspectionEvaluator interface {
	evaluateInspection(inspection cli.Inspection, err error) evaluationResult
}

type completionEvaluator interface {
	evaluateCompletion(completion cli.Completion, err error) evaluationResult
}

type schemaEvaluator interface {
	evaluateSchema(schema *cli.Schema) evaluationResult
}

type referenceEvaluator interface {
	evaluateReference(reference cli.Reference, err error) evaluationResult
}

type diagnosticEvaluator interface {
	evaluateDiagnosticReport(report cli.DiagnosticReport) evaluationResult
}

type diagnosticLocation struct {
	diagnostic cli.Diagnostic
	index      int
	count      int
	subject    string
}

type diagnosticsTruncatedExpectation struct{}

type diagnosticsExactlyExpectationValue struct {
	locators []*diagnosticLocatorState
}

type locatorResolutionStatus uint8

const (
	locatorResolutionInvalid locatorResolutionStatus = iota
	locatorResolutionFound
	locatorResolutionAbsent
	locatorResolutionNotProvable
	locatorResolutionAmbiguous
)

type locatorResolution[T any] struct {
	status  locatorResolutionStatus
	value   T
	subject evaluationField
	detail  evaluationField
}

func observed(subject semanticSubject, condition observationCondition) semanticObservation {
	return semanticObservation{
		subject:   subject,
		condition: condition,
	}
}

func emptyEvaluationField(field evaluationField) bool {
	if field == nil {
		return true
	}
	text, ok := field.(evaluationText)
	return ok && text == ""
}

func property(name, value evaluationField) semanticProperty {
	return semanticProperty{
		name:  name,
		value: value,
	}
}

func qualifySubject(base, qualifier evaluationField) qualifiedSubject {
	return qualifiedSubject{
		base:      base,
		qualifier: qualifier,
	}
}

func indexedSubject(noun evaluationField, index int, parent evaluationField) indexedRelationSubject {
	return indexedRelationSubject{
		noun:   noun,
		index:  index,
		parent: parent,
	}
}

func subjectNamed(base evaluationField, name string) namedSubject {
	return namedSubject{
		base: base,
		name: name,
	}
}

func subjectAt(base evaluationField, index int) indexedSubjectField {
	return indexedSubjectField{
		base:  base,
		index: index,
	}
}

func withDetail(prefix string, detail evaluationField) prefixedDetail {
	return prefixedDetail{
		prefix: prefix,
		detail: detail,
	}
}

func comparison(name, value evaluationField) comparisonField {
	return comparisonField{
		name:      name,
		separator: " = ",
		value:     value,
	}
}

func provedResult() evaluationResult {
	return evaluationResult{status: evaluationProved}
}

func disprovedResult(subject, expect, actual evaluationField) evaluationResult {
	result := evaluationResult{
		status:   evaluationDisproved,
		subject:  subject,
		expected: expect,
		observed: actual,
	}
	return result
}

func notProvableResult(subject, expect, actual evaluationField) evaluationResult {
	result := evaluationResult{
		status:   evaluationNotProvable,
		subject:  subject,
		expected: expect,
		observed: actual,
	}
	return result
}

func expectsOperationFailure[T any](expectations []T) bool {
	for _, expectation := range expectations {
		if _, ok := any(expectation).(explicitFailureExpectation); ok {
			return true
		}
	}

	return false
}

func diagnosticByCode(code cli.DiagnosticCode) DiagnosticLocator {
	return DiagnosticLocator{
		state: &diagnosticLocatorState{
			mode: diagnosticLocatorCode,
			code: code,
		},
	}
}

func diagnosticByIndex(index int) DiagnosticLocator {
	return DiagnosticLocator{
		state: &diagnosticLocatorState{
			mode:  diagnosticLocatorIndex,
			index: index,
		},
	}
}

func diagnosticRule(locator DiagnosticLocator, code cli.RuleCode) DiagnosticLocator {
	if locator.state == nil {
		return DiagnosticLocator{}
	}
	state := *locator.state
	state.ruleSet = true
	state.rule = code
	return DiagnosticLocator{state: &state}
}

func diagnosticAdvisory(locator DiagnosticLocator, code cli.AdvisoryCode) DiagnosticLocator {
	if locator.state == nil {
		return DiagnosticLocator{}
	}
	state := *locator.state
	state.advisorySet = true
	state.advisory = code
	return DiagnosticLocator{state: &state}
}

func diagnosticOn(locator DiagnosticLocator, subject DiagnosticSubject) DiagnosticLocator {
	if locator.state == nil {
		return DiagnosticLocator{}
	}
	state := *locator.state
	state.subjectSet = true
	state.subject = subject
	return DiagnosticLocator{state: &state}
}

func diagnosticRepairAt(diagnostic DiagnosticLocator, index int) RepairLocator {
	return RepairLocator{
		state: &repairLocatorState{
			mode:       repairLocatorDiagnosticIndex,
			diagnostic: diagnostic.state,
			index:      index,
		},
	}
}

func diagnosticAmbiguityActual(location diagnosticLocation) string {
	return fmt.Sprintf(
		"locator matched %d diagnostics; refine with public subject identity or DiagnosticAt",
		location.count,
	)
}

func diagnosticPresentExpectation(locator DiagnosticLocator) OutcomeExpectation {
	return diagnosticExpectation{
		locator:  locator.state,
		property: diagnosticPropertyPresent,
	}
}

func diagnosticAbsentExpectation(locator DiagnosticLocator) OutcomeExpectation {
	return diagnosticExpectation{
		locator:  locator.state,
		property: diagnosticPropertyAbsent,
	}
}

func diagnosticCodeExpectation(locator DiagnosticLocator, code cli.DiagnosticCode) OutcomeExpectation {
	return diagnosticExpectation{
		locator:  locator.state,
		property: diagnosticPropertyCode,
		code:     code,
	}
}

func diagnosticViolationExpectation(locator DiagnosticLocator, kind cli.ViolationKind) OutcomeExpectation {
	return diagnosticExpectation{
		locator:   locator.state,
		property:  diagnosticPropertyViolation,
		violation: kind,
	}
}

func diagnosticProducerKindExpectation(locator DiagnosticLocator, kind cli.ValueProducerKind) OutcomeExpectation {
	return diagnosticExpectation{
		locator:      locator.state,
		property:     diagnosticPropertyProducerKind,
		producerKind: kind,
	}
}

func diagnosticTierExpectation(locator DiagnosticLocator, tier cli.ResolutionTier) OutcomeExpectation {
	return diagnosticExpectation{
		locator:  locator.state,
		property: diagnosticPropertyTier,
		tier:     tier,
	}
}

func diagnosticDispositionExpectation(locator DiagnosticLocator, disposition cli.ValueCandidateDisposition) OutcomeExpectation {
	return diagnosticExpectation{
		locator:     locator.state,
		property:    diagnosticPropertyDisposition,
		disposition: disposition,
	}
}

func diagnosticExitCodeExpectation(locator DiagnosticLocator, code int) OutcomeExpectation {
	return diagnosticExpectation{
		locator:  locator.state,
		property: diagnosticPropertyExitCode,
		integer:  code,
	}
}

func diagnosticSensitiveExpectation(locator DiagnosticLocator) OutcomeExpectation {
	return diagnosticExpectation{
		locator:  locator.state,
		property: diagnosticPropertySensitive,
	}
}

func diagnosticInputRedactedExpectation(locator DiagnosticLocator) OutcomeExpectation {
	return diagnosticExpectation{
		locator:  locator.state,
		property: diagnosticPropertyInputRedacted,
	}
}

func diagnosticSpanExpectation(locator DiagnosticLocator, span cli.InputSpan) OutcomeExpectation {
	return diagnosticExpectation{
		locator:  locator.state,
		property: diagnosticPropertySpan,
		span:     span,
	}
}

func diagnosticRepairCountExpectation(locator DiagnosticLocator, count int) OutcomeExpectation {
	return diagnosticExpectation{
		locator:  locator.state,
		property: diagnosticPropertyRepairCount,
		integer:  count,
	}
}

func diagnosticSignalActionFailureExpectation(locator DiagnosticLocator, failure cli.SignalActionFailure) OutcomeExpectation {
	return diagnosticExpectation{
		locator:             locator.state,
		property:            diagnosticPropertySignalActionFailure,
		signalActionFailure: cloneSignalActionFailure(failure),
	}
}

func diagnosticSignalOverflowFailureExpectation(locator DiagnosticLocator, failure cli.SignalOverflowFailure) OutcomeExpectation {
	return diagnosticExpectation{
		locator:               locator.state,
		property:              diagnosticPropertySignalOverflowFailure,
		signalOverflowFailure: failure,
	}
}

func diagnosticNoticeDeliveryFailureExpectation(locator DiagnosticLocator, failure cli.NoticeDeliveryFailure) OutcomeExpectation {
	expected := failure
	expected.CommandPath = slices.Clone(failure.CommandPath)
	return diagnosticExpectation{
		locator:               locator.state,
		property:              diagnosticPropertyNoticeDeliveryFailure,
		noticeDeliveryFailure: expected,
	}
}

func cloneSignalActionFailure(input cli.SignalActionFailure) cli.SignalActionFailure {
	clone := input
	clone.CommandPath = slices.Clone(input.CommandPath)
	if input.Event.Sender != nil {
		clone.Event.Sender = new(*input.Event.Sender)
	}
	return clone
}

func equalSignalActionFailure(actual, expected cli.SignalActionFailure) bool {
	if !slices.Equal(actual.CommandPath, expected.CommandPath) ||
		actual.PolicyID != expected.PolicyID || actual.Index != expected.Index || actual.Phase != expected.Phase {
		return false
	}

	actualAction, expectedAction := actual.Action, expected.Action
	actualActionSignal, expectedActionSignal := actualAction.Signal, expectedAction.Signal
	actualAction.Signal, expectedAction.Signal = cli.Signal{}, cli.Signal{}
	if actualAction != expectedAction ||
		!equalSignalIdentity(actualActionSignal, expectedActionSignal) {
		return false
	}

	actualEvent, expectedEvent := actual.Event, expected.Event
	actualEventSignal, expectedEventSignal := actualEvent.Signal, expectedEvent.Signal
	actualSender, expectedSender := actualEvent.Sender, expectedEvent.Sender
	actualEvent.Signal, expectedEvent.Signal = cli.Signal{}, cli.Signal{}
	actualEvent.Sender, expectedEvent.Sender = nil, nil
	if actualEvent != expectedEvent || !equalSignalIdentity(actualEventSignal, expectedEventSignal) {
		return false
	}
	if actualSender == nil || expectedSender == nil {
		return actualSender == nil && expectedSender == nil
	}
	return *actualSender == *expectedSender
}

func equalSignalIdentity(actual, expected cli.Signal) bool { return actual == expected }

func diagnosticsExactlyExpectation(locators []DiagnosticLocator) OutcomeExpectation {
	states := make([]*diagnosticLocatorState, len(locators))
	for index, locator := range locators {
		states[index] = locator.state
	}
	return diagnosticsExactlyExpectationValue{locators: states}
}

func summarizeOutcomes(outcomes []evaluationResult) evaluationField {
	return evaluationSummary{outcomes: slices.Clone(outcomes)}
}

func declarationMissing(status locatorResolutionStatus, label, expected, detail evaluationField, truncated bool) evaluationResult {
	if status == locatorResolutionNotProvable || (status == locatorResolutionAbsent && truncated) {
		if emptyEvaluationField(detail) {
			detail = evaluationText("observation is incomplete")
		}
		return notProvableResult(label, expected, detail)
	}
	if emptyEvaluationField(detail) {
		detail = evaluationText("declaration is absent")
	}
	return disprovedResult(label, expected, detail)
}

func nonempty(value, fallback evaluationField) evaluationField {
	if !emptyEvaluationField(value) {
		return value
	}
	return fallback
}

func resolveUniqueObserved[T any](matches []T, complete bool, subject, absentDetail, incompleteDetail, ambiguousDetail evaluationField) locatorResolution[T] {
	resolution := locatorResolution[T]{
		subject: subject,
	}

	switch {
	case len(matches) == 1:
		resolution.status = locatorResolutionFound
		resolution.value = matches[0]
	case len(matches) > 1:
		resolution.status = locatorResolutionAmbiguous
		resolution.detail = ambiguousDetail
	case complete:
		resolution.status = locatorResolutionAbsent
		resolution.detail = absentDetail
	default:
		resolution.status = locatorResolutionNotProvable
		resolution.detail = incompleteDetail
	}

	return resolution
}

func (semanticSubject) evaluationField() {}

func (semanticRequirement) evaluationField() {}

func (semanticObservation) evaluationField() {}

func (evaluationText) evaluationField() {}

func (quotedTextField) evaluationField() {}

func (evaluationBool) evaluationField() {}

func (releaseLifecycleField) evaluationField() {}

func (descriptionField) evaluationField() {}

func (countRangeField) evaluationField() {}

func (stringCollectionField) evaluationField() {}

func (collectionCardinalityField) evaluationField() {}

func (orderedCollectionField) evaluationField() {}

func (runeCollectionField) evaluationField() {}

func (resolutionStateCollectionField) evaluationField() {}

func (inputSpanField) evaluationField() {}

func (signalSenderField) evaluationField() {}

func (timeField) evaluationField() {}

func (semanticProperty) evaluationField() {}

func (qualifiedSubject) evaluationField() {}

func (indexedRelationSubject) evaluationField() {}

func (namedSubject) evaluationField() {}

func (indexedSubjectField) evaluationField() {}

func (prefixedDetail) evaluationField() {}

func (comparisonField) evaluationField() {}

func (evaluationSummary) evaluationField() {}
