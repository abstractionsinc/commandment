package clitest

import (
	"fmt"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
)

// CommandDefaultLocator identifies one command-default declaration for clitest
// semantic expectation. Its zero value is invalid.
type CommandDefaultLocator struct {
	_     [0]func()
	state *commandDefaultLocatorState
}

// CommandDefaultByID returns a locator for the exact public CommandDefaultID in the
// observation being evaluated.
func CommandDefaultByID(id cli.CommandDefaultID) CommandDefaultLocator {
	return commandDefaultIDLocator(id)
}

// DefaultAt returns a locator for the command default at index in the located command's
// authoritative public default order.
func (locator CommandLocator) DefaultAt(index int) CommandDefaultLocator {
	return commandDefaultIndexLocator(locator, index)
}

// DefaultsExactly returns an expectation that checks the located command's command defaults are
// exactly the supplied locators in authoritative public order. Exactness is not inferred from an
// incomplete observation.
func (locator CommandLocator) DefaultsExactly(values ...CommandDefaultLocator) DeclarationExpectation {
	return commandDefaultsExpectation{
		command: locator.state,
		items:   commandDefaultLocatorStates(values),
	}
}

// DefaultGrammarProfile returns a static declaration expectation that checks the command's
// default grammar-profile relationship resolves to the expected public profile identity.
func (locator CommandLocator) DefaultGrammarProfile(expected GrammarProfileLocator) StaticDeclarationExpectation {
	return commandDefaultGrammarProfileExpectation{
		command: locator.state,
		profile: expected.state,
	}
}

// Condition returns a locator that follows the public condition relationship from the located
// command default. The returned locator carries locator coordinates, not observed state.
func (locator CommandDefaultLocator) Condition() ConditionExpressionLocator {
	return ConditionExpressionLocator{state: &conditionExpressionLocatorState{
		mode:           conditionExpressionLocatorCommandDefault,
		commandDefault: locator.state,
	}}
}

// Origin also applies to public declaration families that are not represented
// by DeclarationIdentity but still expose framework-established origin in the
// public Schema/Reference/Inspection model.
func (locator CommandDefaultLocator) Origin(expected cli.DeclarationOrigin) DeclarationExpectation {
	return commandDefaultOriginExpectationValue{
		locator:  locator.state,
		expected: expected,
	}
}

func (CommandDefaultLocator) diagnosticSubject() {}

// Exists returns an expectation that checks the located command default exists in the relevant
// public declaration observation.
func (locator CommandDefaultLocator) Exists() DeclarationExpectation {
	return commandDefaultExistenceExpectationValue{
		locator:  locator.state,
		expected: true,
	}
}

// DoesNotExist returns an expectation that checks the located command default is absent from a
// complete relevant public declaration domain; incomplete observation cannot manufacture
// absence.
func (locator CommandDefaultLocator) DoesNotExist() DeclarationExpectation {
	return commandDefaultExistenceExpectationValue{
		locator:  locator.state,
		expected: false,
	}
}

// Target returns an expectation that checks the command default targets the requested
// canonical command.
func (locator CommandDefaultLocator) Target(v CommandLocator) DeclarationExpectation {
	return commandDefaultTargetExpectationValue{
		locator: locator.state,
		target:  v.state,
	}
}

// Reachability returns an expectation that checks the declared command-default reachability.
func (locator CommandDefaultLocator) Reachability(v cli.Reachability) DeclarationExpectation {
	return commandDefaultReachabilityExpectationValue{
		locator:  locator.state,
		expected: v,
	}
}

// ConditionOutcome returns an expectation that checks the command default's invocation-time
// condition evaluation outcome.
func (locator CommandDefaultLocator) ConditionOutcome(outcome cli.EvaluationOutcome) InspectionExpectation {
	return commandDefaultConditionOutcomeExpectation(locator, outcome)
}

type commandDefaultsExpectation struct {
	command *commandLocatorState
	items   []*commandDefaultLocatorState
}

type commandDefaultLocatorMode uint8

const (
	commandDefaultLocatorInvalid commandDefaultLocatorMode = iota
	commandDefaultLocatorID
	commandDefaultLocatorIndex
)

type commandDefaultLocatorState struct {
	mode commandDefaultLocatorMode
	id   cli.CommandDefaultID

	command *commandLocatorState
	index   int
}

type commandDefaultExistenceExpectationValue struct {
	locator  *commandDefaultLocatorState
	expected bool
}

type commandDefaultReachabilityExpectationValue struct {
	locator  *commandDefaultLocatorState
	expected cli.Reachability
}

type commandDefaultTargetExpectationValue struct {
	locator *commandDefaultLocatorState
	target  *commandLocatorState
}

type commandDefaultOriginExpectationValue struct {
	locator  *commandDefaultLocatorState
	expected cli.DeclarationOrigin
}

func commandDefaultIDLocator(id cli.CommandDefaultID) CommandDefaultLocator {
	return CommandDefaultLocator{state: &commandDefaultLocatorState{
		mode: commandDefaultLocatorID,
		id:   id,
	}}
}

func commandDefaultLocatorLabel(state *commandDefaultLocatorState) evaluationField {
	if state == nil {
		return semanticSubjectCommandDefault
	}
	switch state.mode {
	case commandDefaultLocatorID:
		return evaluationText(fmt.Sprintf("command default id %d", state.id))
	case commandDefaultLocatorIndex:
		return qualifySubject(commandLocatorLabel(state.command), subjectAt(semanticSubjectCommandDefault, state.index))
	default:
		return semanticSubjectCommandDefault
	}
}

func resolveInspectionCommandDefaultLocator(state *commandDefaultLocatorState, inspection cli.Inspection) locatorResolution[cli.CommandDefaultInspection] {
	var resolution locatorResolution[cli.CommandDefaultInspection]
	label := commandDefaultLocatorLabel(state)
	if state == nil || state.mode == commandDefaultLocatorInvalid {
		resolution = locatorResolution[cli.CommandDefaultInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCommandDefault, observationInvalidLocator),
		}
		return resolution
	}
	var id cli.CommandDefaultID
	switch state.mode {
	case commandDefaultLocatorID:
		id = state.id
	case commandDefaultLocatorIndex:
		if state.index < 0 {
			resolution = locatorResolution[cli.CommandDefaultInspection]{
				status:  locatorResolutionInvalid,
				subject: label,
				detail:  observed(semanticSubjectCommandDefault, observationInvalidLocator),
			}
			return resolution
		}
		command := resolveInspectionCommandLocator(state.command, inspection)
		if command.status != locatorResolutionFound {
			resolution = locatorResolution[cli.CommandDefaultInspection]{
				status:  command.status,
				subject: label,
				detail:  command.detail,
			}
			return resolution
		}
		if state.index >= len(command.value.CommandDefaults) {
			if detail, incomplete := policyInspectionNamespaceIncomplete(command.value, inspection); incomplete {
				resolution = locatorResolution[cli.CommandDefaultInspection]{
					status:  locatorResolutionNotProvable,
					subject: label,
					detail:  detail,
				}
				return resolution
			}
			resolution = locatorResolution[cli.CommandDefaultInspection]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectCommandDefault, observationAbsent),
			}
			return resolution
		}
		id = command.value.CommandDefaults[state.index]
	default:
		resolution = locatorResolution[cli.CommandDefaultInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCommandDefault, observationInvalidLocator),
		}
		return resolution
	}
	value, ok := inspection.CommandDefaults[id]
	if ok {
		resolution = locatorResolution[cli.CommandDefaultInspection]{
			status:  locatorResolutionFound,
			value:   value,
			subject: label,
		}
		return resolution
	}
	if inspection.Truncated {
		resolution = locatorResolution[cli.CommandDefaultInspection]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  observed(semanticSubjectInspection, observationTruncated),
		}
		return resolution
	}
	resolution = locatorResolution[cli.CommandDefaultInspection]{
		status:  locatorResolutionAbsent,
		subject: label,
		detail:  observed(semanticSubjectCommandDefault, observationAbsent),
	}
	return resolution
}

//nolint:dupl // Distinct typed policy families intentionally retain direct resolution logic.
func resolveSchemaCommandDefaultLocator(state *commandDefaultLocatorState, schema *cli.Schema) locatorResolution[cli.CommandDefaultSchema] {
	var resolution locatorResolution[cli.CommandDefaultSchema]
	label := commandDefaultLocatorLabel(state)
	if schema == nil || state == nil || state.mode == commandDefaultLocatorInvalid {
		resolution = locatorResolution[cli.CommandDefaultSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCommandDefault, observationInvalidLocator),
		}
		return resolution
	}
	var id cli.CommandDefaultID
	switch state.mode {
	case commandDefaultLocatorID:
		id = state.id
	case commandDefaultLocatorIndex:
		if state.index < 0 {
			resolution = locatorResolution[cli.CommandDefaultSchema]{
				status:  locatorResolutionInvalid,
				subject: label,
				detail:  observed(semanticSubjectCommandDefault, observationInvalidLocator),
			}
			return resolution
		}
		command := resolveSchemaCommandLocator(state.command, schema)
		if command.status != locatorResolutionFound {
			resolution = locatorResolution[cli.CommandDefaultSchema]{
				status:  command.status,
				subject: label,
				detail:  command.detail,
			}
			return resolution
		}
		if state.index >= len(command.value.CommandDefaults) {
			resolution = locatorResolution[cli.CommandDefaultSchema]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectCommandDefault, observationAbsent),
			}
			return resolution
		}
		id = command.value.CommandDefaults[state.index]
	default:
		resolution = locatorResolution[cli.CommandDefaultSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCommandDefault, observationInvalidLocator),
		}
		return resolution
	}
	value, ok := schema.CommandDefaults[id]
	if !ok {
		resolution = locatorResolution[cli.CommandDefaultSchema]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectCommandDefault, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.CommandDefaultSchema]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

//nolint:dupl // Distinct typed policy families intentionally retain direct resolution logic.
func resolveReferenceCommandDefaultLocator(state *commandDefaultLocatorState, reference cli.Reference) locatorResolution[cli.CommandDefaultReference] {
	var resolution locatorResolution[cli.CommandDefaultReference]
	label := commandDefaultLocatorLabel(state)
	if state == nil || state.mode == commandDefaultLocatorInvalid {
		resolution = locatorResolution[cli.CommandDefaultReference]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCommandDefault, observationInvalidLocator),
		}
		return resolution
	}
	var id cli.CommandDefaultID
	switch state.mode {
	case commandDefaultLocatorID:
		id = state.id
	case commandDefaultLocatorIndex:
		if state.index < 0 {
			resolution = locatorResolution[cli.CommandDefaultReference]{
				status:  locatorResolutionInvalid,
				subject: label,
				detail:  observed(semanticSubjectCommandDefault, observationInvalidLocator),
			}
			return resolution
		}
		command := resolveReferenceCommandLocator(state.command, reference)
		if command.status != locatorResolutionFound {
			resolution = locatorResolution[cli.CommandDefaultReference]{
				status:  command.status,
				subject: label,
				detail:  command.detail,
			}
			return resolution
		}
		if state.index >= len(command.value.CommandDefaults) {
			resolution = locatorResolution[cli.CommandDefaultReference]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectCommandDefault, observationAbsent),
			}
			return resolution
		}
		id = command.value.CommandDefaults[state.index]
	default:
		resolution = locatorResolution[cli.CommandDefaultReference]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCommandDefault, observationInvalidLocator),
		}
		return resolution
	}
	value, ok := reference.CommandDefaults[id]
	if !ok {
		resolution = locatorResolution[cli.CommandDefaultReference]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectCommandDefault, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.CommandDefaultReference]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

func commandDefaultLocatorStates(values []CommandDefaultLocator) []*commandDefaultLocatorState {
	states := make([]*commandDefaultLocatorState, len(values))
	for index, value := range values {
		states[index] = value.state
	}
	return states
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func expectedSchemaCommandDefaultID(state *commandDefaultLocatorState, schema *cli.Schema) (cli.CommandDefaultID, locatorResolutionStatus, evaluationField) {
	if schema == nil || state == nil || state.mode == commandDefaultLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectCommandDefault, observationInvalidLocator)
	}
	switch state.mode {
	case commandDefaultLocatorID:
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectCommandDefault, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	case commandDefaultLocatorIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectCommandDefault, observationInvalidLocator)
		}
		command := resolveSchemaCommandLocator(state.command, schema)
		if command.status != locatorResolutionFound {
			return 0, command.status, command.detail
		}
		if state.index >= len(command.value.CommandDefaults) {
			return 0, locatorResolutionAbsent, observed(semanticSubjectCommandDefault, observationAbsent)
		}
		return command.value.CommandDefaults[state.index], locatorResolutionFound, evaluationText("")
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectCommandDefault, observationInvalidLocator)
	}
}

func expectedReferenceCommandDefaultID(state *commandDefaultLocatorState, reference cli.Reference) (cli.CommandDefaultID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == commandDefaultLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectCommandDefault, observationInvalidLocator)
	}
	switch state.mode {
	case commandDefaultLocatorID:
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectCommandDefault, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	case commandDefaultLocatorIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectCommandDefault, observationInvalidLocator)
		}
		command := resolveReferenceCommandLocator(state.command, reference)
		if command.status != locatorResolutionFound {
			return 0, command.status, command.detail
		}
		if state.index >= len(command.value.CommandDefaults) {
			return 0, locatorResolutionAbsent, observed(semanticSubjectCommandDefault, observationAbsent)
		}
		return command.value.CommandDefaults[state.index], locatorResolutionFound, evaluationText("")
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectCommandDefault, observationInvalidLocator)
	}
}

func expectedInspectionCommandDefaultID(state *commandDefaultLocatorState, inspection cli.Inspection) (cli.CommandDefaultID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == commandDefaultLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectCommandDefault, observationInvalidLocator)
	}
	switch state.mode {
	case commandDefaultLocatorID:
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectCommandDefault, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	case commandDefaultLocatorIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectCommandDefault, observationInvalidLocator)
		}
		command := resolveInspectionCommandLocator(state.command, inspection)
		if command.status != locatorResolutionFound {
			return 0, command.status, command.detail
		}
		if state.index < len(command.value.CommandDefaults) {
			return command.value.CommandDefaults[state.index], locatorResolutionFound, evaluationText("")
		}
		if inspection.Truncated || command.value.NamespaceCompleteness != cli.CommandNamespaceComplete {
			return 0, locatorResolutionNotProvable, evaluationText("command-default collection is not closed")
		}
		return 0, locatorResolutionAbsent, observed(semanticSubjectCommandDefault, observationAbsent)
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectCommandDefault, observationInvalidLocator)
	}
}

func commandDefaultIndexLocator(command CommandLocator, index int) CommandDefaultLocator {
	state := &commandDefaultLocatorState{
		mode:    commandDefaultLocatorIndex,
		command: command.state,
		index:   index,
	}

	return CommandDefaultLocator{state: state}
}

func (commandDefaultGrammarProfileExpectation) schemaExpectation() {}

func (commandDefaultGrammarProfileExpectation) referenceExpectation() {}

func (expectation commandDefaultGrammarProfileExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	command := resolveSchemaCommandLocator(expectation.command, schema)
	if command.status != locatorResolutionFound {
		return missingResult(evaluationText("command"), evaluationText("default grammar profile"), command.detail, false)
	}
	if command.value.Profiles == nil {
		return missingResult(evaluationText("command"), evaluationText("default grammar profile"), evaluationText("grammar profiles absent"), false)
	}
	profileID, status, detail := expectedSchemaGrammarProfileID(expectation.profile, schema)
	if status == locatorResolutionNotProvable {
		return notProvableResult(evaluationText("command"), evaluationText("default grammar profile"), detail)
	}
	if status != locatorResolutionFound {
		return relationLocatorFailureOutcome("command", "default grammar profile", "expected grammar-profile locator", status)
	}
	if command.value.Profiles.Default == profileID {
		return provedResult()
	}
	return disprovedResult(evaluationText("command"), evaluationText("default grammar profile matches expected identity"), evaluationText("default grammar profile has different identity"))
}

func (expectation commandDefaultGrammarProfileExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	command := resolveReferenceCommandLocator(expectation.command, reference)
	if command.status != locatorResolutionFound {
		return missingResult(evaluationText("command"), evaluationText("default grammar profile"), command.detail, false)
	}
	if command.value.Profiles == nil {
		return missingResult(evaluationText("command"), evaluationText("default grammar profile"), evaluationText("grammar profiles absent"), false)
	}
	profileID, status, detail := expectedReferenceGrammarProfileID(expectation.profile, reference)
	if status == locatorResolutionNotProvable {
		return notProvableResult(evaluationText("command"), evaluationText("default grammar profile"), detail)
	}
	if status != locatorResolutionFound {
		return relationLocatorFailureOutcome("command", "default grammar profile", "expected grammar-profile locator", status)
	}
	if command.value.Profiles.Default == profileID {
		return provedResult()
	}
	return disprovedResult(evaluationText("command"), evaluationText("default grammar profile matches expected identity"), evaluationText("default grammar profile has different identity"))
}

func (commandDefaultsExpectation) schemaExpectation() {}

func (commandDefaultsExpectation) referenceExpectation() {}

func (commandDefaultsExpectation) inspectionExpectation() {}

func (expectation commandDefaultsExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	command := resolveSchemaCommandLocator(expectation.command, schema)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("defaults exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("defaults"))
	expected := make([]cli.CommandDefaultID, len(expectation.items))
	for index, state := range expectation.items {
		id, status, detail := expectedSchemaCommandDefaultID(state, schema)
		if outcome, failed := collectionResolutionFailure(label, "defaults", index, status, detail); failed {
			return outcome
		}
		expected[index] = id
	}
	if slices.Equal(command.value.CommandDefaults, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "defaults", expected, command.value.CommandDefaults)
}

func (expectation commandDefaultsExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	command := resolveReferenceCommandLocator(expectation.command, reference)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("defaults exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("defaults"))
	expected := make([]cli.CommandDefaultID, len(expectation.items))
	for index, state := range expectation.items {
		id, status, detail := expectedReferenceCommandDefaultID(state, reference)
		if outcome, failed := collectionResolutionFailure(label, "defaults", index, status, detail); failed {
			return outcome
		}
		expected[index] = id
	}
	if slices.Equal(command.value.CommandDefaults, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "defaults", expected, command.value.CommandDefaults)
}

func (expectation commandDefaultsExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	command := resolveInspectionCommandLocator(expectation.command, inspection)
	label, failure, ok := exactPolicyCollectionInspection(command, inspection, "defaults")
	if !ok {
		return failure
	}
	expected := make([]cli.CommandDefaultID, len(expectation.items))
	for index, state := range expectation.items {
		id, status, detail := expectedInspectionCommandDefaultID(state, inspection)
		if outcome, failed := collectionResolutionFailure(label, "defaults", index, status, detail); failed {
			return outcome
		}
		expected[index] = id
	}
	if slices.Equal(command.value.CommandDefaults, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "defaults", expected, command.value.CommandDefaults)
}

func (commandDefaultExistenceExpectationValue) schemaExpectation() {}

func (commandDefaultExistenceExpectationValue) referenceExpectation() {}

func (commandDefaultExistenceExpectationValue) inspectionExpectation() {}

func (expectation commandDefaultExistenceExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	return declarationExistenceResult(resolveSchemaCommandDefaultLocator(expectation.locator, schema), expectation.expected)
}

func (expectation commandDefaultExistenceExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return declarationExistenceResult(resolveReferenceCommandDefaultLocator(expectation.locator, reference), expectation.expected)
}

func (expectation commandDefaultExistenceExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return declarationExistenceResult(resolveInspectionCommandDefaultLocator(expectation.locator, inspection), expectation.expected)
}

func (commandDefaultReachabilityExpectationValue) schemaExpectation() {}

func (commandDefaultReachabilityExpectationValue) referenceExpectation() {}

func (commandDefaultReachabilityExpectationValue) inspectionExpectation() {}

func (expectation commandDefaultReachabilityExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaCommandDefaultLocator(expectation.locator, schema)
	actual := resolution.value.Reachability
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("reachability %v", expectation.expected)), evaluationText(fmt.Sprintf("reachability %v", actual)), actual == expectation.expected)
}

func (expectation commandDefaultReachabilityExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceCommandDefaultLocator(expectation.locator, reference)
	actual := resolution.value.Reachability
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("reachability %v", expectation.expected)), evaluationText(fmt.Sprintf("reachability %v", actual)), actual == expectation.expected)
}

func (expectation commandDefaultReachabilityExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionCommandDefaultLocator(expectation.locator, inspection)
	actual := resolution.value.Reachability
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("reachability %v", expectation.expected)), evaluationText(fmt.Sprintf("reachability %v", actual)), actual == expectation.expected)
}

func (commandDefaultTargetExpectationValue) schemaExpectation() {}

func (commandDefaultTargetExpectationValue) referenceExpectation() {}

func (commandDefaultTargetExpectationValue) inspectionExpectation() {}

func (expectation commandDefaultTargetExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	subject := resolveSchemaCommandDefaultLocator(expectation.locator, schema)
	target := resolveSchemaCommandLocator(expectation.target, schema)
	return relatedCommandResult(subject.status, subject.subject, subject.detail, target, subject.value.Target)
}

func (expectation commandDefaultTargetExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	subject := resolveReferenceCommandDefaultLocator(expectation.locator, reference)
	target := resolveReferenceCommandLocator(expectation.target, reference)
	return relatedReferenceCommandResult(subject.status, subject.subject, subject.detail, target, subject.value.Target)
}

func (expectation commandDefaultTargetExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	subject := resolveInspectionCommandDefaultLocator(expectation.locator, inspection)
	target := resolveInspectionCommandLocator(expectation.target, inspection)
	return relatedInspectionCommandResult(subject.status, subject.subject, subject.detail, target, subject.value.Target)
}

func (commandDefaultOriginExpectationValue) schemaExpectation() {}

func (commandDefaultOriginExpectationValue) referenceExpectation() {}

func (commandDefaultOriginExpectationValue) inspectionExpectation() {}

func (expectation commandDefaultOriginExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaCommandDefaultLocator(expectation.locator, schema)
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("origin"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Origin)), resolution.value.Origin == expectation.expected)
}

func (expectation commandDefaultOriginExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceCommandDefaultLocator(expectation.locator, reference)
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("origin"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Origin)), resolution.value.Origin == expectation.expected)
}

func (expectation commandDefaultOriginExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionCommandDefaultLocator(expectation.locator, inspection)
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("origin"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Origin)), resolution.value.Origin == expectation.expected)
}

func (commandDefaultSelectedExpectationValue) inspectionExpectation() {}

func (commandDefaultSelectedExpectationValue) completionExpectation() {}

func (expectation commandDefaultSelectedExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return evaluateCommandDefaultSelection(expectation.locator, expectation.selected, inspection.Selection.Command)
}

func (expectation commandDefaultSelectedExpectationValue) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	return evaluateCommandDefaultSelection(expectation.locator, expectation.selected, result.Selection)
}

func (commandDefaultConditionExpectationValue) inspectionExpectation() {}

func (expectation commandDefaultConditionExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	expect := fmt.Sprintf("result %v", expectation.outcome)
	resolved := resolveInspectionCommandDefaultLocator(expectation.locator, inspection)
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, evaluationText("command default condition"), evaluationText(expect), resolved.detail)
	}
	if resolved.value.ConditionOutcome == expectation.outcome {
		return provedResult()
	}
	return disprovedResult(evaluationText("command default condition"), evaluationText(expect), evaluationText(fmt.Sprintf("result %v", resolved.value.ConditionOutcome)))
}
