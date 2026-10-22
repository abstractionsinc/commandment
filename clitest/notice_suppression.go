package clitest

import (
	"fmt"

	"github.com/abstractionsinc/commandment/cli"
)

// NoticeSuppressionLocator identifies one Notice-suppression declaration for
// clitest semantic expectation. Its zero value is invalid.
type NoticeSuppressionLocator struct {
	_     [0]func()
	state *noticeSuppressionLocatorState
}

// NoticeSuppressionAt returns a locator for the notice suppression at index in the located
// command's authoritative public suppression order.
func (l CommandLocator) NoticeSuppressionAt(index int) NoticeSuppressionLocator {
	return noticeSuppressionIndexLocator(l, index)
}

// NoticeSuppressionsExactly returns an expectation that checks the located command's notice
// suppressions are exactly the supplied locators in authoritative public order. Exactness is not
// inferred from an incomplete observation.
func (l CommandLocator) NoticeSuppressionsExactly(values ...NoticeSuppressionLocator) DeclarationExpectation {
	expectation := commandNoticeSuppressionsExpectation{
		command: l.state,
		items:   noticeSuppressionLocatorStates(values),
	}
	return expectation
}

// ConditionExpression returns a reusable expectation that checks the public condition
// expression exposed by the located Notice suppression.
func (locator NoticeSuppressionLocator) ConditionExpression() ConditionExpressionLocator {
	state := &conditionExpressionLocatorState{
		mode:              conditionExpressionLocatorNoticeSuppression,
		noticeSuppression: locator.state,
	}
	conditionExpressionLocator := ConditionExpressionLocator{
		state: state,
	}
	return conditionExpressionLocator
}

// Origin returns an expectation that checks the located notice suppression has the
// framework-established DeclarationOrigin expected.
func (locator NoticeSuppressionLocator) Origin(expected cli.DeclarationOrigin) DeclarationExpectation {
	value := noticeSuppressionOriginExpectationValue{
		locator:  locator.state,
		expected: expected,
	}
	return value
}

// Exists returns an expectation that checks the located Notice suppression exists in the
// relevant public declaration observation.
func (locator NoticeSuppressionLocator) Exists() DeclarationExpectation {
	value := noticeSuppressionExistenceExpectationValue{
		locator:  locator.state,
		expected: true,
	}
	return value
}

// DoesNotExist returns an expectation that checks the located Notice suppression is absent
// from a complete relevant public declaration domain; incomplete observation cannot
// manufacture absence.
func (locator NoticeSuppressionLocator) DoesNotExist() DeclarationExpectation {
	value := noticeSuppressionExistenceExpectationValue{
		locator:  locator.state,
		expected: false,
	}
	return value
}

// Reachability returns an expectation that checks the Notice suppression has the requested
// declared reachability.
func (locator NoticeSuppressionLocator) Reachability(v cli.Reachability) DeclarationExpectation {
	value := noticeSuppressionReachabilityExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// ConditionOutcome returns an expectation that checks the Notice suppression's invocation-time
// condition evaluation outcome.
func (locator NoticeSuppressionLocator) ConditionOutcome(outcome cli.EvaluationOutcome) InspectionExpectation {
	return noticeSuppressionConditionOutcomeExpectation(locator, outcome)
}

// Match returns an expectation that checks the Notice suppression has exactly the requested
// public NoticeMatch.
func (locator NoticeSuppressionLocator) Match(v cli.NoticeMatch) DeclarationExpectation {
	value := noticeSuppressionMatchExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

type commandNoticeSuppressionsExpectation struct {
	command *commandLocatorState
	items   []*noticeSuppressionLocatorState
}

type noticeSuppressionLocatorState struct {
	command *commandLocatorState
	index   int
}

type noticeSuppressionExistenceExpectationValue struct {
	locator  *noticeSuppressionLocatorState
	expected bool
}

type noticeSuppressionReachabilityExpectationValue struct {
	locator  *noticeSuppressionLocatorState
	expected cli.Reachability
}

type noticeSuppressionMatchExpectationValue struct {
	locator  *noticeSuppressionLocatorState
	expected cli.NoticeMatch
}

type noticeSuppressionOriginExpectationValue struct {
	locator  *noticeSuppressionLocatorState
	expected cli.DeclarationOrigin
}

type noticeSuppressionConditionOutcomeExpectationValue struct {
	locator  *noticeSuppressionLocatorState
	expected cli.EvaluationOutcome
}

func noticeSuppressionLocatorLabel(state *noticeSuppressionLocatorState) evaluationField {
	if state == nil {
		return semanticSubjectNoticeSuppression
	}
	return qualifySubject(commandLocatorLabel(state.command), subjectAt(semanticSubjectNoticeSuppression, state.index))
}

//nolint:dupl // Distinct typed policy families intentionally retain direct resolution logic.
func resolveInspectionNoticeSuppressionLocator(state *noticeSuppressionLocatorState, inspection cli.Inspection) locatorResolution[cli.NoticeSuppressionInspection] {
	var resolution locatorResolution[cli.NoticeSuppressionInspection]
	label := noticeSuppressionLocatorLabel(state)
	if state == nil || state.command == nil || state.index < 0 {
		resolution = locatorResolution[cli.NoticeSuppressionInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectNoticeSuppression, observationInvalidLocator),
		}
		return resolution
	}
	command := resolveInspectionCommandLocator(state.command, inspection)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.NoticeSuppressionInspection]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	if state.index < len(command.value.NoticeSuppressions) {
		resolution = locatorResolution[cli.NoticeSuppressionInspection]{
			status:  locatorResolutionFound,
			value:   command.value.NoticeSuppressions[state.index],
			subject: label,
		}
		return resolution
	}
	if detail, incomplete := policyInspectionNamespaceIncomplete(command.value, inspection); incomplete {
		resolution = locatorResolution[cli.NoticeSuppressionInspection]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	resolution = locatorResolution[cli.NoticeSuppressionInspection]{
		status:  locatorResolutionAbsent,
		subject: label,
		detail:  observed(semanticSubjectNoticeSuppression, observationAbsent),
	}
	return resolution
}

func resolveSchemaNoticeSuppressionLocator(state *noticeSuppressionLocatorState, schema *cli.Schema) locatorResolution[cli.NoticeSuppressionSchema] {
	var resolution locatorResolution[cli.NoticeSuppressionSchema]
	label := noticeSuppressionLocatorLabel(state)
	if schema == nil || state == nil || state.command == nil || state.index < 0 {
		resolution = locatorResolution[cli.NoticeSuppressionSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectNoticeSuppression, observationInvalidLocator),
		}
		return resolution
	}
	command := resolveSchemaCommandLocator(state.command, schema)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.NoticeSuppressionSchema]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	if state.index >= len(command.value.NoticeSuppressions) {
		resolution = locatorResolution[cli.NoticeSuppressionSchema]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectNoticeSuppression, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.NoticeSuppressionSchema]{
		status:  locatorResolutionFound,
		value:   command.value.NoticeSuppressions[state.index],
		subject: label,
	}
	return resolution
}

func resolveReferenceNoticeSuppressionLocator(state *noticeSuppressionLocatorState, reference cli.Reference) locatorResolution[cli.NoticeSuppressionSchema] {
	var resolution locatorResolution[cli.NoticeSuppressionSchema]
	label := noticeSuppressionLocatorLabel(state)
	if state == nil || state.command == nil || state.index < 0 {
		resolution = locatorResolution[cli.NoticeSuppressionSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectNoticeSuppression, observationInvalidLocator),
		}
		return resolution
	}
	command := resolveReferenceCommandLocator(state.command, reference)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.NoticeSuppressionSchema]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	if state.index >= len(command.value.NoticeSuppressions) {
		resolution = locatorResolution[cli.NoticeSuppressionSchema]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectNoticeSuppression, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.NoticeSuppressionSchema]{
		status:  locatorResolutionFound,
		value:   command.value.NoticeSuppressions[state.index],
		subject: label,
	}
	return resolution
}

func noticeSuppressionLocatorStates(values []NoticeSuppressionLocator) []*noticeSuppressionLocatorState {
	states := make([]*noticeSuppressionLocatorState, len(values))
	for index, value := range values {
		states[index] = value.state
	}
	return states
}

func noticeSuppressionIndexLocator(command CommandLocator, index int) NoticeSuppressionLocator {
	state := &noticeSuppressionLocatorState{
		command: command.state,
		index:   index,
	}

	locator := NoticeSuppressionLocator{
		state: state,
	}
	return locator
}

func noticeSuppressionConditionOutcomeExpectation(locator NoticeSuppressionLocator, outcome cli.EvaluationOutcome) InspectionExpectation {
	value := noticeSuppressionConditionOutcomeExpectationValue{
		locator:  locator.state,
		expected: outcome,
	}
	return value
}

func (commandNoticeSuppressionsExpectation) schemaExpectation() {}

func (commandNoticeSuppressionsExpectation) referenceExpectation() {}

func (commandNoticeSuppressionsExpectation) inspectionExpectation() {}

func (expectation commandNoticeSuppressionsExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	command := resolveSchemaCommandLocator(expectation.command, schema)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("notice suppressions exactly"), command.detail)
	}
	return compareSuppressionStatesSchema(command.value, expectation.items, schema, qualifySubject(command.subject, evaluationText("notice suppressions")))
}

func (expectation commandNoticeSuppressionsExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	command := resolveReferenceCommandLocator(expectation.command, reference)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("notice suppressions exactly"), command.detail)
	}
	return compareSuppressionStatesReference(command.value, expectation.items, reference, qualifySubject(command.subject, evaluationText("notice suppressions")))
}

func (expectation commandNoticeSuppressionsExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	command := resolveInspectionCommandLocator(expectation.command, inspection)
	label, failure, ok := exactPolicyCollectionInspection(command, inspection, "notice suppressions")
	if !ok {
		return failure
	}
	return compareSuppressionStatesInspection(command.value, expectation.items, inspection, label)
}

func (noticeSuppressionExistenceExpectationValue) schemaExpectation() {}

func (noticeSuppressionExistenceExpectationValue) referenceExpectation() {}

func (noticeSuppressionExistenceExpectationValue) inspectionExpectation() {}

func (expectation noticeSuppressionExistenceExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	return declarationExistenceResult(resolveSchemaNoticeSuppressionLocator(expectation.locator, schema), expectation.expected)
}

func (expectation noticeSuppressionExistenceExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return declarationExistenceResult(resolveReferenceNoticeSuppressionLocator(expectation.locator, reference), expectation.expected)
}

func (expectation noticeSuppressionExistenceExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return declarationExistenceResult(resolveInspectionNoticeSuppressionLocator(expectation.locator, inspection), expectation.expected)
}

func (noticeSuppressionReachabilityExpectationValue) schemaExpectation() {}

func (noticeSuppressionReachabilityExpectationValue) referenceExpectation() {}

func (noticeSuppressionReachabilityExpectationValue) inspectionExpectation() {}

func (expectation noticeSuppressionReachabilityExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaNoticeSuppressionLocator(expectation.locator, schema)
	actual := resolution.value.Reachability
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("reachability %v", expectation.expected)), evaluationText(fmt.Sprintf("reachability %v", actual)), actual == expectation.expected)
}

func (expectation noticeSuppressionReachabilityExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceNoticeSuppressionLocator(expectation.locator, reference)
	actual := resolution.value.Reachability
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("reachability %v", expectation.expected)), evaluationText(fmt.Sprintf("reachability %v", actual)), actual == expectation.expected)
}

func (expectation noticeSuppressionReachabilityExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionNoticeSuppressionLocator(expectation.locator, inspection)
	actual := resolution.value.Reachability
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("reachability %v", expectation.expected)), evaluationText(fmt.Sprintf("reachability %v", actual)), actual == expectation.expected)
}

func (noticeSuppressionMatchExpectationValue) schemaExpectation() {}

func (noticeSuppressionMatchExpectationValue) referenceExpectation() {}

func (noticeSuppressionMatchExpectationValue) inspectionExpectation() {}

func (expectation noticeSuppressionMatchExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaNoticeSuppressionLocator(expectation.locator, schema)
	actual := resolution.value.Match
	field := noticeMatchField{
		value: expectation.expected,
	}
	observedField := noticeMatchField{
		value: actual,
	}
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, property(evaluationText("match"), field), property(evaluationText("match"), observedField), actual == expectation.expected)
}

func (expectation noticeSuppressionMatchExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceNoticeSuppressionLocator(expectation.locator, reference)
	actual := resolution.value.Match
	field := noticeMatchField{
		value: expectation.expected,
	}
	observedField := noticeMatchField{
		value: actual,
	}
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, property(evaluationText("match"), field), property(evaluationText("match"), observedField), actual == expectation.expected)
}

func (expectation noticeSuppressionMatchExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionNoticeSuppressionLocator(expectation.locator, inspection)
	actual := resolution.value.Match
	field := noticeMatchField{
		value: expectation.expected,
	}
	observedField := noticeMatchField{
		value: actual,
	}
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, property(evaluationText("match"), field), property(evaluationText("match"), observedField), actual == expectation.expected)
}

func (noticeSuppressionOriginExpectationValue) schemaExpectation() {}

func (noticeSuppressionOriginExpectationValue) referenceExpectation() {}

func (noticeSuppressionOriginExpectationValue) inspectionExpectation() {}

func (expectation noticeSuppressionOriginExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaNoticeSuppressionLocator(expectation.locator, schema)
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("origin"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Origin)), resolution.value.Origin == expectation.expected)
}

func (expectation noticeSuppressionOriginExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceNoticeSuppressionLocator(expectation.locator, reference)
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("origin"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Origin)), resolution.value.Origin == expectation.expected)
}

func (expectation noticeSuppressionOriginExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionNoticeSuppressionLocator(expectation.locator, inspection)
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("origin"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Origin)), resolution.value.Origin == expectation.expected)
}

func (noticeSuppressionConditionOutcomeExpectationValue) inspectionExpectation() {}

func (expectation noticeSuppressionConditionOutcomeExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionNoticeSuppressionLocator(expectation.locator, inspection)
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("condition-outcome"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.ConditionOutcome)), resolution.value.ConditionOutcome == expectation.expected)
}
