package clitest

import (
	"fmt"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
)

// AdvisoryLocator identifies one Advisory for clitest semantic expectation. Its zero
// value is invalid.
type AdvisoryLocator struct {
	_     [0]func()
	state *advisoryLocatorState
}

// Advisory returns a locator for the uniquely resolved advisory classified by code under the
// located command.
func (l CommandLocator) Advisory(code cli.AdvisoryCode) AdvisoryLocator {
	return advisoryCodeLocator(l, code)
}

// AdvisoryAt returns a locator for the advisory at index in the located command's authoritative
// public advisory order.
func (l CommandLocator) AdvisoryAt(index int) AdvisoryLocator {
	return advisoryIndexLocator(l, index)
}

// AdvisoriesExactly returns an expectation that checks the located command's advisories are
// exactly the supplied locators in authoritative public order. Exactness is not inferred from an
// incomplete observation.
func (l CommandLocator) AdvisoriesExactly(values ...AdvisoryLocator) DeclarationExpectation {
	expectation := commandAdvisoriesExpectation{
		command: l.state,
		items:   advisoryLocatorStates(values),
	}
	return expectation
}

// Activation returns a locator that follows the public activation relationship from the located
// Advisory. The returned locator carries locator coordinates, not observed state.
func (locator AdvisoryLocator) Activation() ConditionExpressionLocator {
	state := &conditionExpressionLocatorState{
		mode:     conditionExpressionLocatorAdvisoryActivation,
		advisory: locator.state,
	}
	conditionExpressionLocator := ConditionExpressionLocator{
		state: state,
	}
	return conditionExpressionLocator
}

// Origin returns an expectation that checks the located advisory has the framework-established
// DeclarationOrigin expected.
func (locator AdvisoryLocator) Origin(expected cli.DeclarationOrigin) DeclarationExpectation {
	value := advisoryOriginExpectationValue{
		locator:  locator.state,
		expected: expected,
	}
	return value
}

// Exists returns an expectation that checks the located Advisory exists in the relevant public
// declaration observation.
func (locator AdvisoryLocator) Exists() DeclarationExpectation {
	value := advisoryExistenceExpectationValue{
		locator:  locator.state,
		expected: true,
	}
	return value
}

// DoesNotExist returns an expectation that checks the located Advisory is absent from a
// complete relevant public declaration domain; incomplete observation cannot manufacture
// absence.
func (locator AdvisoryLocator) DoesNotExist() DeclarationExpectation {
	value := advisoryExistenceExpectationValue{
		locator:  locator.state,
		expected: false,
	}
	return value
}

// Kind returns an expectation that checks the Advisory has the requested declared
// AdvisoryKind.
func (locator AdvisoryLocator) Kind(v cli.AdvisoryKind) DeclarationExpectation {
	value := advisoryKindExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Code returns an expectation that checks the Advisory carries the requested AdvisoryCode; the
// code is metadata, not Advisory identity.
func (locator AdvisoryLocator) Code(v cli.AdvisoryCode) DeclarationExpectation {
	value := advisoryCodeExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Severity returns an expectation that checks the Advisory has the requested Notice severity.
func (locator AdvisoryLocator) Severity(v cli.NoticeSeverity) DeclarationExpectation {
	value := advisorySeverityExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Reachability returns an expectation that checks the Advisory has the requested declared
// reachability.
func (locator AdvisoryLocator) Reachability(v cli.Reachability) DeclarationExpectation {
	value := advisoryReachabilityExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Outcome returns an expectation that checks the Advisory's invocation-time outcome.
func (locator AdvisoryLocator) Outcome(status cli.AdvisoryOutcome) InspectionExpectation {
	return advisoryOutcomeExpectation(locator, status)
}

// ActivationOutcome returns an expectation that checks the Advisory's optional activation
// evaluation outcome.
func (locator AdvisoryLocator) ActivationOutcome(outcome cli.EvaluationOutcome) InspectionExpectation {
	return advisoryActivationOutcomeExpectation(locator, outcome)
}

// SubjectsExactly returns an expectation that checks the Advisory declares exactly the
// requested value subjects in authoritative order.
func (locator AdvisoryLocator) SubjectsExactly(values ...ValueLocator) DeclarationExpectation {
	states := make([]*valueLocatorState, len(values))
	for i, value := range values {
		states[i] = value.state
	}
	expectation := advisoryValuesExpectationValue{
		locator:  locator.state,
		expected: states,
	}
	return expectation
}

// DependenciesExactly returns an expectation that checks the Advisory declares exactly the
// requested value dependencies in authoritative order.
func (locator AdvisoryLocator) DependenciesExactly(values ...ValueLocator) DeclarationExpectation {
	states := make([]*valueLocatorState, len(values))
	for i, value := range values {
		states[i] = value.state
	}
	expectation := advisoryValuesExpectationValue{
		locator:      locator.state,
		expected:     states,
		dependencies: true,
	}
	return expectation
}

type commandAdvisoriesExpectation struct {
	command *commandLocatorState
	items   []*advisoryLocatorState
}

type advisoryLocatorMode uint8

const (
	advisoryLocatorInvalid advisoryLocatorMode = iota
	advisoryLocatorCode
	advisoryLocatorIndex
)

type advisoryLocatorState struct {
	mode advisoryLocatorMode

	command *commandLocatorState
	code    cli.AdvisoryCode
	index   int
}

type advisoryExistenceExpectationValue struct {
	locator  *advisoryLocatorState
	expected bool
}

type advisoryKindExpectationValue struct {
	locator  *advisoryLocatorState
	expected cli.AdvisoryKind
}

type advisoryCodeExpectationValue struct {
	locator  *advisoryLocatorState
	expected cli.AdvisoryCode
}

type advisorySeverityExpectationValue struct {
	locator  *advisoryLocatorState
	expected cli.NoticeSeverity
}

type advisoryReachabilityExpectationValue struct {
	locator  *advisoryLocatorState
	expected cli.Reachability
}

type advisoryValuesExpectationValue struct {
	locator      *advisoryLocatorState
	expected     []*valueLocatorState
	dependencies bool
}

type advisoryOriginExpectationValue struct {
	locator  *advisoryLocatorState
	expected cli.DeclarationOrigin
}

type advisoryOutcomeExpectationValue struct {
	locator  *advisoryLocatorState
	expected cli.AdvisoryOutcome
}

type advisoryActivationOutcomeExpectationValue struct {
	locator  *advisoryLocatorState
	expected cli.EvaluationOutcome
}

func advisoryLocatorLabel(state *advisoryLocatorState) evaluationField {
	if state == nil {
		return semanticSubjectAdvisory
	}
	switch state.mode {
	case advisoryLocatorCode:
		return qualifySubject(commandLocatorLabel(state.command), subjectNamed(semanticSubjectAdvisory, string(state.code)))
	case advisoryLocatorIndex:
		return qualifySubject(commandLocatorLabel(state.command), subjectAt(semanticSubjectAdvisory, state.index))
	default:
		return semanticSubjectAdvisory
	}
}

func resolveInspectionAdvisoryLocator(state *advisoryLocatorState, inspection cli.Inspection) locatorResolution[cli.AdvisoryInspection] {
	var resolution locatorResolution[cli.AdvisoryInspection]
	label := advisoryLocatorLabel(state)
	if state == nil || state.command == nil || state.mode == advisoryLocatorInvalid {
		resolution = locatorResolution[cli.AdvisoryInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectAdvisory, observationInvalidLocator),
		}
		return resolution
	}
	command := resolveInspectionCommandLocator(state.command, inspection)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.AdvisoryInspection]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	switch state.mode {
	case advisoryLocatorIndex:
		if state.index < 0 {
			resolution = locatorResolution[cli.AdvisoryInspection]{
				status:  locatorResolutionInvalid,
				subject: label,
				detail:  observed(semanticSubjectAdvisory, observationInvalidLocator),
			}
			return resolution
		}
		if state.index < len(command.value.Advisories) {
			resolution = locatorResolution[cli.AdvisoryInspection]{
				status:  locatorResolutionFound,
				value:   command.value.Advisories[state.index],
				subject: label,
			}
			return resolution
		}
	case advisoryLocatorCode:
		matches := make([]int, 0, 1)
		for index, advisory := range command.value.Advisories {
			if advisory.Code == state.code {
				matches = append(matches, index)
			}
		}
		if len(matches) == 1 {
			resolution = locatorResolution[cli.AdvisoryInspection]{
				status:  locatorResolutionFound,
				value:   command.value.Advisories[matches[0]],
				subject: label,
			}
			return resolution
		}
		if len(matches) > 1 {
			slices.Sort(matches)
			resolution = locatorResolution[cli.AdvisoryInspection]{
				status:  locatorResolutionAmbiguous,
				subject: label,
				detail: collectionCardinalityField{
					name:  semanticNameMatchesAdvisoryCodeIndexes,
					count: len(matches),
				},
			}
			return resolution
		}
	default:
		resolution = locatorResolution[cli.AdvisoryInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectAdvisory, observationInvalidLocator),
		}
		return resolution
	}
	if detail, incomplete := policyInspectionNamespaceIncomplete(command.value, inspection); incomplete {
		resolution = locatorResolution[cli.AdvisoryInspection]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	resolution = locatorResolution[cli.AdvisoryInspection]{
		status:  locatorResolutionAbsent,
		subject: label,
		detail:  observed(semanticSubjectAdvisory, observationAbsent),
	}
	return resolution
}

func resolveSchemaAdvisoryLocator(state *advisoryLocatorState, schema *cli.Schema) locatorResolution[cli.AdvisorySchema] {
	var resolution locatorResolution[cli.AdvisorySchema]
	label := advisoryLocatorLabel(state)
	if schema == nil || state == nil || state.command == nil || state.mode == advisoryLocatorInvalid {
		resolution = locatorResolution[cli.AdvisorySchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectAdvisory, observationInvalidLocator),
		}
		return resolution
	}
	command := resolveSchemaCommandLocator(state.command, schema)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.AdvisorySchema]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	return selectSchemaAdvisoryLocator(state, command.value.Advisories, label)
}

func resolveReferenceAdvisoryLocator(state *advisoryLocatorState, reference cli.Reference) locatorResolution[cli.AdvisoryReference] {
	var resolution locatorResolution[cli.AdvisoryReference]
	label := advisoryLocatorLabel(state)
	if state == nil || state.command == nil || state.mode == advisoryLocatorInvalid {
		resolution = locatorResolution[cli.AdvisoryReference]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectAdvisory, observationInvalidLocator),
		}
		return resolution
	}
	command := resolveReferenceCommandLocator(state.command, reference)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.AdvisoryReference]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	switch state.mode {
	case advisoryLocatorIndex:
		if state.index < 0 {
			resolution = locatorResolution[cli.AdvisoryReference]{
				status:  locatorResolutionInvalid,
				subject: label,
				detail:  observed(semanticSubjectAdvisory, observationInvalidLocator),
			}
			return resolution
		}
		if state.index < len(command.value.Advisories) {
			resolution = locatorResolution[cli.AdvisoryReference]{
				status:  locatorResolutionFound,
				value:   command.value.Advisories[state.index],
				subject: label,
			}
			return resolution
		}
		resolution = locatorResolution[cli.AdvisoryReference]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectAdvisory, observationAbsent),
		}
		return resolution
	case advisoryLocatorCode:
		matches := make([]int, 0, 1)
		for index, value := range command.value.Advisories {
			if value.Code == state.code {
				matches = append(matches, index)
			}
		}
		if len(matches) == 1 {
			resolution = locatorResolution[cli.AdvisoryReference]{
				status:  locatorResolutionFound,
				value:   command.value.Advisories[matches[0]],
				subject: label,
			}
			return resolution
		}
		if len(matches) > 1 {
			slices.Sort(matches)
			resolution = locatorResolution[cli.AdvisoryReference]{
				status:  locatorResolutionAmbiguous,
				subject: label,
				detail: collectionCardinalityField{
					name:  semanticNameMatchesAdvisoryCodeIndexes,
					count: len(matches),
				},
			}
			return resolution
		}
		resolution = locatorResolution[cli.AdvisoryReference]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectAdvisory, observationAbsent),
		}
		return resolution
	default:
		resolution = locatorResolution[cli.AdvisoryReference]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectAdvisory, observationInvalidLocator),
		}
		return resolution
	}
}

func selectSchemaAdvisoryLocator(state *advisoryLocatorState, values []cli.AdvisorySchema, label evaluationField) locatorResolution[cli.AdvisorySchema] {
	var resolution locatorResolution[cli.AdvisorySchema]
	switch state.mode {
	case advisoryLocatorIndex:
		if state.index < 0 {
			resolution = locatorResolution[cli.AdvisorySchema]{
				status:  locatorResolutionInvalid,
				subject: label,
				detail:  observed(semanticSubjectAdvisory, observationInvalidLocator),
			}
			return resolution
		}
		if state.index < len(values) {
			resolution = locatorResolution[cli.AdvisorySchema]{
				status:  locatorResolutionFound,
				value:   values[state.index],
				subject: label,
			}
			return resolution
		}
		resolution = locatorResolution[cli.AdvisorySchema]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectAdvisory, observationAbsent),
		}
		return resolution
	case advisoryLocatorCode:
		matches := make([]int, 0, 1)
		for index, value := range values {
			if value.Code == state.code {
				matches = append(matches, index)
			}
		}
		if len(matches) == 1 {
			resolution = locatorResolution[cli.AdvisorySchema]{
				status:  locatorResolutionFound,
				value:   values[matches[0]],
				subject: label,
			}
			return resolution
		}
		if len(matches) > 1 {
			slices.Sort(matches)
			resolution = locatorResolution[cli.AdvisorySchema]{
				status:  locatorResolutionAmbiguous,
				subject: label,
				detail: collectionCardinalityField{
					name:  semanticNameMatchesAdvisoryCodeIndexes,
					count: len(matches),
				},
			}
			return resolution
		}
		resolution = locatorResolution[cli.AdvisorySchema]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectAdvisory, observationAbsent),
		}
		return resolution
	default:
		resolution = locatorResolution[cli.AdvisorySchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectAdvisory, observationInvalidLocator),
		}
		return resolution
	}
}

func advisoryLocatorStates(values []AdvisoryLocator) []*advisoryLocatorState {
	states := make([]*advisoryLocatorState, len(values))
	for index, value := range values {
		states[index] = value.state
	}
	return states
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func compareAdvisoryStatesSchema(command cli.CommandSchema, states []*advisoryLocatorState, schema *cli.Schema, label evaluationField) evaluationResult {
	if len(command.Advisories) != len(states) {
		return disprovedResult(label, evaluationText(fmt.Sprintf("advisories count %d", len(states))), evaluationText(fmt.Sprintf("advisories count %d", len(command.Advisories))))
	}
	for index, state := range states {
		if state == nil || !validateSchemaPolicyParent(state.command, command, schema) {
			return disprovedResult(label, evaluationText("ordered advisory locators"), evaluationText(fmt.Sprintf("invalid advisory locator at position %d", index)))
		}
		switch state.mode {
		case advisoryLocatorIndex:
			if state.index != index {
				return disprovedResult(label, evaluationText("ordered advisory locators"), evaluationText(fmt.Sprintf("advisory index %d at position %d", state.index, index)))
			}
		case advisoryLocatorCode:
			matches := 0
			matched := -1
			for candidate := range command.Advisories {
				if command.Advisories[candidate].Code == state.code {
					matches++
					matched = candidate
				}
			}
			if matches != 1 || matched != index {
				field := collectionCardinalityField{
					name:  semanticNameMatchesAdvisoryCodeRecords,
					count: matches,
				}
				return disprovedResult(label, evaluationText("ordered unambiguous advisory locators"), field)
			}
		default:
			return disprovedResult(label, evaluationText("advisory index/code locators"), evaluationText(fmt.Sprintf("invalid advisory locator at position %d", index)))
		}
	}
	return provedResult()
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func compareAdvisoryStatesReference(command cli.CommandReference, states []*advisoryLocatorState, reference cli.Reference, label evaluationField) evaluationResult {
	if len(command.Advisories) != len(states) {
		return disprovedResult(label, evaluationText(fmt.Sprintf("advisories count %d", len(states))), evaluationText(fmt.Sprintf("advisories count %d", len(command.Advisories))))
	}
	for index, state := range states {
		if state == nil || !validateReferencePolicyParent(state.command, command, reference) {
			return disprovedResult(label, evaluationText("ordered advisory locators"), evaluationText(fmt.Sprintf("invalid advisory locator at position %d", index)))
		}
		switch state.mode {
		case advisoryLocatorIndex:
			if state.index != index {
				return disprovedResult(label, evaluationText("ordered advisory locators"), evaluationText(fmt.Sprintf("advisory index %d at position %d", state.index, index)))
			}
		case advisoryLocatorCode:
			matches := 0
			matched := -1
			for candidate := range command.Advisories {
				if command.Advisories[candidate].Code == state.code {
					matches++
					matched = candidate
				}
			}
			if matches != 1 || matched != index {
				field := collectionCardinalityField{
					name:  semanticNameMatchesAdvisoryCodeRecords,
					count: matches,
				}
				return disprovedResult(label, evaluationText("ordered unambiguous advisory locators"), field)
			}
		default:
			return disprovedResult(label, evaluationText("advisory index/code locators"), evaluationText(fmt.Sprintf("invalid advisory locator at position %d", index)))
		}
	}
	return provedResult()
}

func compareAdvisoryStatesInspection(command cli.CommandInspection, states []*advisoryLocatorState, inspection cli.Inspection, label evaluationField) evaluationResult {
	if len(command.Advisories) != len(states) {
		return disprovedResult(label, evaluationText(fmt.Sprintf("advisories count %d", len(states))), evaluationText(fmt.Sprintf("advisories count %d", len(command.Advisories))))
	}
	for index, state := range states {
		if state == nil {
			return disprovedResult(label, evaluationText("ordered advisory locators"), evaluationText(fmt.Sprintf("invalid advisory locator at position %d", index)))
		}
		if outcome, ok := validateInspectionPolicyParent(state.command, command, inspection, label, evaluationText("ordered advisory locators")); !ok {
			return outcome
		}
		switch state.mode {
		case advisoryLocatorIndex:
			if state.index != index {
				return disprovedResult(label, evaluationText("ordered advisory locators"), evaluationText(fmt.Sprintf("advisory index %d at position %d", state.index, index)))
			}
		case advisoryLocatorCode:
			matches := 0
			matched := -1
			for candidate := range command.Advisories {
				if command.Advisories[candidate].Code == state.code {
					matches++
					matched = candidate
				}
			}
			if matches != 1 || matched != index {
				field := collectionCardinalityField{
					name:  semanticNameMatchesAdvisoryCodeRecords,
					count: matches,
				}
				return disprovedResult(label, evaluationText("ordered unambiguous advisory locators"), field)
			}
		default:
			return disprovedResult(label, evaluationText("advisory index/code locators"), evaluationText(fmt.Sprintf("invalid advisory locator at position %d", index)))
		}
	}
	return provedResult()
}

func advisoryCodeLocator(command CommandLocator, code cli.AdvisoryCode) AdvisoryLocator {
	state := &advisoryLocatorState{
		mode:    advisoryLocatorCode,
		command: command.state,
		code:    code,
	}

	locator := AdvisoryLocator{
		state: state,
	}
	return locator
}

func advisoryIndexLocator(command CommandLocator, index int) AdvisoryLocator {
	state := &advisoryLocatorState{
		mode:    advisoryLocatorIndex,
		command: command.state,
		index:   index,
	}

	locator := AdvisoryLocator{
		state: state,
	}
	return locator
}

func advisoryOutcomeExpectation(locator AdvisoryLocator, status cli.AdvisoryOutcome) InspectionExpectation {
	value := advisoryOutcomeExpectationValue{
		locator:  locator.state,
		expected: status,
	}
	return value
}

func advisoryActivationOutcomeExpectation(locator AdvisoryLocator, outcome cli.EvaluationOutcome) InspectionExpectation {
	value := advisoryActivationOutcomeExpectationValue{
		locator:  locator.state,
		expected: outcome,
	}
	return value
}

func (commandAdvisoriesExpectation) schemaExpectation() {}

func (commandAdvisoriesExpectation) referenceExpectation() {}

func (commandAdvisoriesExpectation) inspectionExpectation() {}

func (expectation commandAdvisoriesExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	command := resolveSchemaCommandLocator(expectation.command, schema)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("advisories exactly"), command.detail)
	}
	return compareAdvisoryStatesSchema(command.value, expectation.items, schema, qualifySubject(command.subject, evaluationText("advisories")))
}

func (expectation commandAdvisoriesExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	command := resolveReferenceCommandLocator(expectation.command, reference)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("advisories exactly"), command.detail)
	}
	return compareAdvisoryStatesReference(command.value, expectation.items, reference, qualifySubject(command.subject, evaluationText("advisories")))
}

func (expectation commandAdvisoriesExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	command := resolveInspectionCommandLocator(expectation.command, inspection)
	label, failure, ok := exactPolicyCollectionInspection(command, inspection, "advisories")
	if !ok {
		return failure
	}
	return compareAdvisoryStatesInspection(command.value, expectation.items, inspection, label)
}

func (advisoryExistenceExpectationValue) schemaExpectation() {}

func (advisoryExistenceExpectationValue) referenceExpectation() {}

func (advisoryExistenceExpectationValue) inspectionExpectation() {}

func (expectation advisoryExistenceExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	return declarationExistenceResult(resolveSchemaAdvisoryLocator(expectation.locator, schema), expectation.expected)
}

func (expectation advisoryExistenceExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return declarationExistenceResult(resolveReferenceAdvisoryLocator(expectation.locator, reference), expectation.expected)
}

func (expectation advisoryExistenceExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return declarationExistenceResult(resolveInspectionAdvisoryLocator(expectation.locator, inspection), expectation.expected)
}

func (advisoryKindExpectationValue) schemaExpectation() {}

func (advisoryKindExpectationValue) referenceExpectation() {}

func (advisoryKindExpectationValue) inspectionExpectation() {}

func (expectation advisoryKindExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaAdvisoryLocator(expectation.locator, schema)
	actual := resolution.value.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (expectation advisoryKindExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceAdvisoryLocator(expectation.locator, reference)
	actual := resolution.value.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (expectation advisoryKindExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionAdvisoryLocator(expectation.locator, inspection)
	actual := resolution.value.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (advisoryCodeExpectationValue) schemaExpectation() {}

func (advisoryCodeExpectationValue) referenceExpectation() {}

func (advisoryCodeExpectationValue) inspectionExpectation() {}

func (expectation advisoryCodeExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaAdvisoryLocator(expectation.locator, schema)
	actual := resolution.value.Code
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, subjectNamed(evaluationText("code"), string(expectation.expected)), subjectNamed(evaluationText("code"), string(actual)), actual == expectation.expected)
}

func (expectation advisoryCodeExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceAdvisoryLocator(expectation.locator, reference)
	actual := resolution.value.Code
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, subjectNamed(evaluationText("code"), string(expectation.expected)), subjectNamed(evaluationText("code"), string(actual)), actual == expectation.expected)
}

func (expectation advisoryCodeExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionAdvisoryLocator(expectation.locator, inspection)
	actual := resolution.value.Code
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, subjectNamed(evaluationText("code"), string(expectation.expected)), subjectNamed(evaluationText("code"), string(actual)), actual == expectation.expected)
}

func (advisorySeverityExpectationValue) schemaExpectation() {}

func (advisorySeverityExpectationValue) referenceExpectation() {}

func (advisorySeverityExpectationValue) inspectionExpectation() {}

func (expectation advisorySeverityExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaAdvisoryLocator(expectation.locator, schema)
	actual := resolution.value.Severity
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("severity %v", expectation.expected)), evaluationText(fmt.Sprintf("severity %v", actual)), actual == expectation.expected)
}

func (expectation advisorySeverityExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceAdvisoryLocator(expectation.locator, reference)
	actual := resolution.value.Severity
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("severity %v", expectation.expected)), evaluationText(fmt.Sprintf("severity %v", actual)), actual == expectation.expected)
}

func (expectation advisorySeverityExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionAdvisoryLocator(expectation.locator, inspection)
	actual := resolution.value.Severity
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("severity %v", expectation.expected)), evaluationText(fmt.Sprintf("severity %v", actual)), actual == expectation.expected)
}

func (advisoryReachabilityExpectationValue) schemaExpectation() {}

func (advisoryReachabilityExpectationValue) referenceExpectation() {}

func (advisoryReachabilityExpectationValue) inspectionExpectation() {}

func (expectation advisoryReachabilityExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaAdvisoryLocator(expectation.locator, schema)
	actual := resolution.value.Reachability
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("reachability %v", expectation.expected)), evaluationText(fmt.Sprintf("reachability %v", actual)), actual == expectation.expected)
}

func (expectation advisoryReachabilityExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceAdvisoryLocator(expectation.locator, reference)
	actual := resolution.value.Reachability
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("reachability %v", expectation.expected)), evaluationText(fmt.Sprintf("reachability %v", actual)), actual == expectation.expected)
}

func (expectation advisoryReachabilityExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionAdvisoryLocator(expectation.locator, inspection)
	actual := resolution.value.Reachability
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("reachability %v", expectation.expected)), evaluationText(fmt.Sprintf("reachability %v", actual)), actual == expectation.expected)
}

func (advisoryValuesExpectationValue) schemaExpectation() {}

func (advisoryValuesExpectationValue) referenceExpectation() {}

func (advisoryValuesExpectationValue) inspectionExpectation() {}

func (expectation advisoryValuesExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaAdvisoryLocator(expectation.locator, schema)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, expectation.valuesField(), evaluationText(""), false)
	}
	expected, status, detail := resolveExpectedValueIDsSchema(expectation.expected, schema)
	return expectedValueIDsResult(resolution.subject, expectation.schemaValues(resolution.value), expected, status, detail, expectation.valuesName())
}

func (expectation advisoryValuesExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceAdvisoryLocator(expectation.locator, reference)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, expectation.valuesField(), evaluationText(""), false)
	}
	expected, status, detail := resolveExpectedValueIDsReference(expectation.expected, reference)
	return expectedValueIDsResult(resolution.subject, expectation.referenceValues(resolution.value), expected, status, detail, expectation.valuesName())
}

func (expectation advisoryValuesExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionAdvisoryLocator(expectation.locator, inspection)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, expectation.valuesField(), evaluationText(""), false)
	}
	expected, status, detail := resolveExpectedValueIDsInspection(expectation.expected, inspection)
	return expectedValueIDsResult(resolution.subject, expectation.inspectionValues(resolution.value), expected, status, detail, expectation.valuesName())
}

func (expectation advisoryValuesExpectationValue) valuesName() string {
	if expectation.dependencies {
		return semanticNameDependencies
	}
	return "subjects"
}

func (expectation advisoryValuesExpectationValue) valuesField() evaluationField {
	return evaluationText("exact " + expectation.valuesName())
}

func (expectation advisoryValuesExpectationValue) schemaValues(advisory cli.AdvisorySchema) []cli.ValueID {
	if expectation.dependencies {
		return advisory.Dependencies
	}
	return advisory.Subjects
}

func (expectation advisoryValuesExpectationValue) referenceValues(advisory cli.AdvisoryReference) []cli.ValueID {
	if expectation.dependencies {
		return advisory.Dependencies
	}
	return advisory.Subjects
}

func (expectation advisoryValuesExpectationValue) inspectionValues(advisory cli.AdvisoryInspection) []cli.ValueID {
	if expectation.dependencies {
		return advisory.Dependencies
	}
	return advisory.Subjects
}

func (advisoryOriginExpectationValue) schemaExpectation() {}

func (advisoryOriginExpectationValue) referenceExpectation() {}

func (advisoryOriginExpectationValue) inspectionExpectation() {}

func (expectation advisoryOriginExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaAdvisoryLocator(expectation.locator, schema)
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("origin"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Origin)), resolution.value.Origin == expectation.expected)
}

func (expectation advisoryOriginExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceAdvisoryLocator(expectation.locator, reference)
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("origin"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Origin)), resolution.value.Origin == expectation.expected)
}

func (expectation advisoryOriginExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionAdvisoryLocator(expectation.locator, inspection)
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("origin"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Origin)), resolution.value.Origin == expectation.expected)
}

func (advisoryOutcomeExpectationValue) inspectionExpectation() {}

func (expectation advisoryOutcomeExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionAdvisoryLocator(expectation.locator, inspection)
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("status"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Outcome)), resolution.value.Outcome == expectation.expected)
}

func (advisoryActivationOutcomeExpectationValue) inspectionExpectation() {}

func (expectation advisoryActivationOutcomeExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionAdvisoryLocator(expectation.locator, inspection)
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("activation-outcome"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.ActivationOutcome)), resolution.value.ActivationOutcome == expectation.expected)
}
