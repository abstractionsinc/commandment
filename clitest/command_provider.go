package clitest

import (
	"fmt"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
)

// CommandProviderLocator identifies one command-provider declaration or
// materialization record for clitest semantic expectation. Its zero value is invalid.
type CommandProviderLocator struct {
	_     [0]func()
	state *commandProviderLocatorState
}

// InspectionObservedCommandProvidersExactly returns an Inspection expectation that the returned CommandProviders map
// contains exactly the supplied distinct provider identities.
// This is observed membership of the materialized Inspection snapshot; map iteration order is not semantic and provider
// lifecycle/status remains a separate property expectation.
func InspectionObservedCommandProvidersExactly(providers ...CommandProviderLocator) InspectionExpectation {
	states := make([]*commandProviderLocatorState, len(providers))
	for index, provider := range providers {
		states[index] = provider.state
	}
	expectation := observedCommandProvidersExpectation{
		providers: states,
	}
	return expectation
}

// CommandProviderByID returns a locator for the exact public CommandProviderID in the
// observation being evaluated.
func CommandProviderByID(id cli.CommandProviderID) CommandProviderLocator {
	return commandProviderIDLocator(id)
}

// ProviderAt returns a locator for the command provider at index in the located command's
// authoritative public provider order.
func (l CommandLocator) ProviderAt(index int) CommandProviderLocator {
	return commandProviderIndexLocator(l, index)
}

// ProvidersExactly returns an expectation that checks the located command's command providers are
// exactly the supplied locators in authoritative public order. Exactness is not inferred from an
// incomplete observation.
func (l CommandLocator) ProvidersExactly(values ...CommandProviderLocator) DeclarationExpectation {
	expectation := commandProvidersExpectation{
		command: l.state,
		items:   commandProviderLocatorStates(values),
	}
	return expectation
}

func (CommandProviderLocator) diagnosticSubject() {}

// Exists returns an expectation that checks the located command provider exists in the
// relevant public declaration observation.
func (l CommandProviderLocator) Exists() DeclarationExpectation {
	value := commandProviderExistenceExpectationValue{
		locator:  l.state,
		expected: true,
	}
	return value
}

// DoesNotExist returns an expectation that checks the located command provider is absent from
// a complete relevant public declaration domain; incomplete observation cannot manufacture
// absence.
func (l CommandProviderLocator) DoesNotExist() DeclarationExpectation {
	value := commandProviderExistenceExpectationValue{
		locator:  l.state,
		expected: false,
	}
	return value
}

// Outcome returns an expectation that checks the provider materialization outcome in Inspection.
func (l CommandProviderLocator) Outcome(status cli.CommandProviderOutcome) InspectionExpectation {
	return commandProviderOutcomeExpectation(l, status)
}

// EntriesExactly returns an expectation that checks the provider committed exactly the
// requested command entries in authoritative public order; truncated Inspection cannot establish
// exactness.
func (l CommandProviderLocator) EntriesExactly(entries ...CommandEntryLocator) InspectionExpectation {
	return commandProviderEntriesExactlyExpectation(l, entries)
}

// DependenciesExactly returns an expectation that checks the declared provider dependencies
// are exactly the requested values in authoritative order.
func (l CommandProviderLocator) DependenciesExactly(values ...ValueLocator) DeclarationExpectation {
	states := make([]*valueLocatorState, len(values))
	for i, value := range values {
		states[i] = value.state
	}
	expectation := commandProviderDependenciesExpectationValue{
		locator:  l.state,
		expected: states,
	}
	return expectation
}

// HasLocalization returns an expectation that checks the provider published localization
// payload for this Inspection.
func (l CommandProviderLocator) HasLocalization() InspectionExpectation {
	return commandProviderLocalizationExpectation(l, true)
}

// NoLocalization returns an expectation that checks the provider published no localization
// payload; truncated Inspection cannot establish this absence.
func (l CommandProviderLocator) NoLocalization() InspectionExpectation {
	return commandProviderLocalizationExpectation(l, false)
}

// LocalizationExactly returns an Inspection expectation that checks the located
// provider's complete canonical localization payload. Use HasLocalization or
// NoLocalization when only presence is under test.
func (l CommandProviderLocator) LocalizationExactly(expected cli.Localization) InspectionExpectation {
	return commandProviderLocalizationPayloadExpectation(l, expected)
}

type observedCommandProvidersExpectation struct {
	providers []*commandProviderLocatorState
}

type commandProvidersExpectation struct {
	command *commandLocatorState
	items   []*commandProviderLocatorState
}

type commandProviderLocatorMode uint8

const (
	commandProviderLocatorInvalid commandProviderLocatorMode = iota
	commandProviderLocatorID
	commandProviderLocatorIndex
)

type commandProviderLocatorState struct {
	mode commandProviderLocatorMode
	id   cli.CommandProviderID

	command *commandLocatorState
	index   int
}

type commandProviderExistenceExpectationValue struct {
	locator  *commandProviderLocatorState
	expected bool
}

type commandProviderDependenciesExpectationValue struct {
	locator  *commandProviderLocatorState
	expected []*valueLocatorState
}

type providerLocalizationPresenceExpectation struct {
	locator *commandProviderLocatorState
	present bool
}

type providerLocalizationPayloadExpectation struct {
	locator  *commandProviderLocatorState
	expected cli.Localization
}

type providerEntriesExactlyExpectation struct {
	locator *commandProviderLocatorState
	entries []*commandEntryLocatorState
}

type commandProviderOutcomeExpectationValue struct {
	locator  *commandProviderLocatorState
	expected cli.CommandProviderOutcome
}

func commandProviderIDLocator(id cli.CommandProviderID) CommandProviderLocator {
	state := &commandProviderLocatorState{
		mode: commandProviderLocatorID,
		id:   id,
	}
	locator := CommandProviderLocator{
		state: state,
	}
	return locator
}

func commandProviderLocatorLabel(state *commandProviderLocatorState) evaluationField {
	if state == nil {
		return semanticSubjectCommandProvider
	}
	switch state.mode {
	case commandProviderLocatorID:
		return evaluationText(fmt.Sprintf("command provider id %d", state.id))
	case commandProviderLocatorIndex:
		return qualifySubject(commandLocatorLabel(state.command), subjectAt(semanticSubjectCommandProvider, state.index))
	default:
		return semanticSubjectCommandProvider
	}
}

func resolveInspectionCommandProviderLocator(state *commandProviderLocatorState, inspection cli.Inspection) locatorResolution[cli.CommandProviderInspection] {
	var resolution locatorResolution[cli.CommandProviderInspection]
	label := commandProviderLocatorLabel(state)
	if state == nil || state.mode == commandProviderLocatorInvalid {
		resolution = locatorResolution[cli.CommandProviderInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCommandProvider, observationInvalidLocator),
		}
		return resolution
	}
	var id cli.CommandProviderID
	switch state.mode {
	case commandProviderLocatorID:
		id = state.id
	case commandProviderLocatorIndex:
		if state.index < 0 {
			resolution = locatorResolution[cli.CommandProviderInspection]{
				status:  locatorResolutionInvalid,
				subject: label,
				detail:  observed(semanticSubjectCommandProvider, observationInvalidLocator),
			}
			return resolution
		}
		command := resolveInspectionCommandLocator(state.command, inspection)
		if command.status != locatorResolutionFound {
			resolution = locatorResolution[cli.CommandProviderInspection]{
				status:  command.status,
				subject: label,
				detail:  command.detail,
			}
			return resolution
		}
		if state.index >= len(command.value.CommandProviders) {
			if detail, incomplete := policyInspectionNamespaceIncomplete(command.value, inspection); incomplete {
				resolution = locatorResolution[cli.CommandProviderInspection]{
					status:  locatorResolutionNotProvable,
					subject: label,
					detail:  detail,
				}
				return resolution
			}
			resolution = locatorResolution[cli.CommandProviderInspection]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectCommandProvider, observationAbsent),
			}
			return resolution
		}
		id = command.value.CommandProviders[state.index]
	default:
		resolution = locatorResolution[cli.CommandProviderInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCommandProvider, observationInvalidLocator),
		}
		return resolution
	}
	value, ok := inspection.CommandProviders[id]
	if ok {
		resolution = locatorResolution[cli.CommandProviderInspection]{
			status:  locatorResolutionFound,
			value:   value,
			subject: label,
		}
		return resolution
	}
	if inspection.Truncated {
		resolution = locatorResolution[cli.CommandProviderInspection]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  observed(semanticSubjectInspection, observationTruncated),
		}
		return resolution
	}
	resolution = locatorResolution[cli.CommandProviderInspection]{
		status:  locatorResolutionAbsent,
		subject: label,
		detail:  observed(semanticSubjectCommandProvider, observationAbsent),
	}
	return resolution
}

//nolint:dupl // Distinct typed policy families intentionally retain direct resolution logic.
func resolveSchemaCommandProviderLocator(state *commandProviderLocatorState, schema *cli.Schema) locatorResolution[cli.CommandProviderSchema] {
	var resolution locatorResolution[cli.CommandProviderSchema]
	label := commandProviderLocatorLabel(state)
	if schema == nil || state == nil || state.mode == commandProviderLocatorInvalid {
		resolution = locatorResolution[cli.CommandProviderSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCommandProvider, observationInvalidLocator),
		}
		return resolution
	}
	var id cli.CommandProviderID
	switch state.mode {
	case commandProviderLocatorID:
		id = state.id
	case commandProviderLocatorIndex:
		if state.index < 0 {
			resolution = locatorResolution[cli.CommandProviderSchema]{
				status:  locatorResolutionInvalid,
				subject: label,
				detail:  observed(semanticSubjectCommandProvider, observationInvalidLocator),
			}
			return resolution
		}
		command := resolveSchemaCommandLocator(state.command, schema)
		if command.status != locatorResolutionFound {
			resolution = locatorResolution[cli.CommandProviderSchema]{
				status:  command.status,
				subject: label,
				detail:  command.detail,
			}
			return resolution
		}
		if state.index >= len(command.value.CommandProviders) {
			resolution = locatorResolution[cli.CommandProviderSchema]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectCommandProvider, observationAbsent),
			}
			return resolution
		}
		id = command.value.CommandProviders[state.index]
	default:
		resolution = locatorResolution[cli.CommandProviderSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCommandProvider, observationInvalidLocator),
		}
		return resolution
	}
	value, ok := schema.CommandProviders[id]
	if !ok {
		resolution = locatorResolution[cli.CommandProviderSchema]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectCommandProvider, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.CommandProviderSchema]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

//nolint:dupl // Distinct typed policy families intentionally retain direct resolution logic.
func resolveReferenceCommandProviderLocator(state *commandProviderLocatorState, reference cli.Reference) locatorResolution[cli.CommandProviderReference] {
	var resolution locatorResolution[cli.CommandProviderReference]
	label := commandProviderLocatorLabel(state)
	if state == nil || state.mode == commandProviderLocatorInvalid {
		resolution = locatorResolution[cli.CommandProviderReference]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCommandProvider, observationInvalidLocator),
		}
		return resolution
	}
	var id cli.CommandProviderID
	switch state.mode {
	case commandProviderLocatorID:
		id = state.id
	case commandProviderLocatorIndex:
		if state.index < 0 {
			resolution = locatorResolution[cli.CommandProviderReference]{
				status:  locatorResolutionInvalid,
				subject: label,
				detail:  observed(semanticSubjectCommandProvider, observationInvalidLocator),
			}
			return resolution
		}
		command := resolveReferenceCommandLocator(state.command, reference)
		if command.status != locatorResolutionFound {
			resolution = locatorResolution[cli.CommandProviderReference]{
				status:  command.status,
				subject: label,
				detail:  command.detail,
			}
			return resolution
		}
		if state.index >= len(command.value.CommandProviders) {
			resolution = locatorResolution[cli.CommandProviderReference]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectCommandProvider, observationAbsent),
			}
			return resolution
		}
		id = command.value.CommandProviders[state.index]
	default:
		resolution = locatorResolution[cli.CommandProviderReference]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCommandProvider, observationInvalidLocator),
		}
		return resolution
	}
	value, ok := reference.CommandProviders[id]
	if !ok {
		resolution = locatorResolution[cli.CommandProviderReference]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectCommandProvider, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.CommandProviderReference]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

func commandProviderLocatorStates(values []CommandProviderLocator) []*commandProviderLocatorState {
	states := make([]*commandProviderLocatorState, len(values))
	for index, value := range values {
		states[index] = value.state
	}
	return states
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func expectedSchemaCommandProviderID(state *commandProviderLocatorState, schema *cli.Schema) (cli.CommandProviderID, locatorResolutionStatus, evaluationField) {
	if schema == nil || state == nil || state.mode == commandProviderLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectCommandProvider, observationInvalidLocator)
	}
	switch state.mode {
	case commandProviderLocatorID:
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectCommandProvider, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	case commandProviderLocatorIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectCommandProvider, observationInvalidLocator)
		}
		command := resolveSchemaCommandLocator(state.command, schema)
		if command.status != locatorResolutionFound {
			return 0, command.status, command.detail
		}
		if state.index >= len(command.value.CommandProviders) {
			return 0, locatorResolutionAbsent, observed(semanticSubjectCommandProvider, observationAbsent)
		}
		return command.value.CommandProviders[state.index], locatorResolutionFound, evaluationText("")
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectCommandProvider, observationInvalidLocator)
	}
}

func expectedReferenceCommandProviderID(state *commandProviderLocatorState, reference cli.Reference) (cli.CommandProviderID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == commandProviderLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectCommandProvider, observationInvalidLocator)
	}
	switch state.mode {
	case commandProviderLocatorID:
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectCommandProvider, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	case commandProviderLocatorIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectCommandProvider, observationInvalidLocator)
		}
		command := resolveReferenceCommandLocator(state.command, reference)
		if command.status != locatorResolutionFound {
			return 0, command.status, command.detail
		}
		if state.index >= len(command.value.CommandProviders) {
			return 0, locatorResolutionAbsent, observed(semanticSubjectCommandProvider, observationAbsent)
		}
		return command.value.CommandProviders[state.index], locatorResolutionFound, evaluationText("")
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectCommandProvider, observationInvalidLocator)
	}
}

func expectedInspectionCommandProviderID(state *commandProviderLocatorState, inspection cli.Inspection) (cli.CommandProviderID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == commandProviderLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectCommandProvider, observationInvalidLocator)
	}
	switch state.mode {
	case commandProviderLocatorID:
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectCommandProvider, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	case commandProviderLocatorIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectCommandProvider, observationInvalidLocator)
		}
		command := resolveInspectionCommandLocator(state.command, inspection)
		if command.status != locatorResolutionFound {
			return 0, command.status, command.detail
		}
		if state.index < len(command.value.CommandProviders) {
			return command.value.CommandProviders[state.index], locatorResolutionFound, evaluationText("")
		}
		if inspection.Truncated || command.value.NamespaceCompleteness != cli.CommandNamespaceComplete {
			return 0, locatorResolutionNotProvable, evaluationText("command-provider collection is not closed")
		}
		return 0, locatorResolutionAbsent, observed(semanticSubjectCommandProvider, observationAbsent)
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectCommandProvider, observationInvalidLocator)
	}
}

func commandProviderIndexLocator(command CommandLocator, index int) CommandProviderLocator {
	state := &commandProviderLocatorState{
		mode:    commandProviderLocatorIndex,
		command: command.state,
		index:   index,
	}

	locator := CommandProviderLocator{
		state: state,
	}
	return locator
}

func commandProviderOutcomeExpectation(locator CommandProviderLocator, outcome cli.CommandProviderOutcome) InspectionExpectation {
	value := commandProviderOutcomeExpectationValue{
		locator:  locator.state,
		expected: outcome,
	}
	return value
}

func commandProviderLocalizationExpectation(locator CommandProviderLocator, present bool) InspectionExpectation {
	expectation := providerLocalizationPresenceExpectation{
		locator: locator.state,
		present: present,
	}
	return expectation
}

func commandProviderLocalizationPayloadExpectation(locator CommandProviderLocator, expected cli.Localization) InspectionExpectation {
	expectation := providerLocalizationPayloadExpectation{
		locator:  locator.state,
		expected: cloneLocalization(expected),
	}
	return expectation
}

func commandProviderEntriesExactlyExpectation(locator CommandProviderLocator, entries []CommandEntryLocator) InspectionExpectation {
	states := make([]*commandEntryLocatorState, len(entries))
	for index, entry := range entries {
		states[index] = entry.state
	}
	expectation := providerEntriesExactlyExpectation{
		locator: locator.state,
		entries: states,
	}
	return expectation
}

func (observedCommandProvidersExpectation) inspectionExpectation() {}

func (expectation observedCommandProvidersExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	if len(inspection.CommandProviders) != len(expectation.providers) {
		return disprovedResult(
			evaluationText("observed command providers"),
			evaluationText(fmt.Sprintf("exactly %d providers", len(expectation.providers))),
			evaluationText(fmt.Sprintf("%d observed providers", len(inspection.CommandProviders))),
		)
	}
	seen := map[cli.CommandProviderID]struct{}{}
	for index, state := range expectation.providers {
		resolved := resolveInspectionCommandProviderLocator(state, inspection)
		if resolved.status != locatorResolutionFound {
			return expectedRelationFailure(resolved.status, evaluationText("observed command providers"), evaluationText(fmt.Sprintf("provider at position %d", index)), resolved.detail)
		}
		if _, duplicate := seen[resolved.value.ID]; duplicate {
			return disprovedResult(
				evaluationText("observed command providers"),
				evaluationText("distinct semantic providers"),
				evaluationText(fmt.Sprintf("duplicate expected provider at position %d", index)),
			)
		}
		seen[resolved.value.ID] = struct{}{}
	}
	return provedResult()
}

func (commandProvidersExpectation) schemaExpectation() {}

func (commandProvidersExpectation) referenceExpectation() {}

func (commandProvidersExpectation) inspectionExpectation() {}

func (expectation commandProvidersExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	command := resolveSchemaCommandLocator(expectation.command, schema)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("providers exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("providers"))
	expected := make([]cli.CommandProviderID, len(expectation.items))
	for index, state := range expectation.items {
		id, status, detail := expectedSchemaCommandProviderID(state, schema)
		if outcome, failed := collectionResolutionFailure(label, "providers", index, status, detail); failed {
			return outcome
		}
		expected[index] = id
	}
	if slices.Equal(command.value.CommandProviders, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "providers", expected, command.value.CommandProviders)
}

func (expectation commandProvidersExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	command := resolveReferenceCommandLocator(expectation.command, reference)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("providers exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("providers"))
	expected := make([]cli.CommandProviderID, len(expectation.items))
	for index, state := range expectation.items {
		id, status, detail := expectedReferenceCommandProviderID(state, reference)
		if outcome, failed := collectionResolutionFailure(label, "providers", index, status, detail); failed {
			return outcome
		}
		expected[index] = id
	}
	if slices.Equal(command.value.CommandProviders, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "providers", expected, command.value.CommandProviders)
}

func (expectation commandProvidersExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	command := resolveInspectionCommandLocator(expectation.command, inspection)
	label, failure, ok := exactPolicyCollectionInspection(command, inspection, "providers")
	if !ok {
		return failure
	}
	expected := make([]cli.CommandProviderID, len(expectation.items))
	for index, state := range expectation.items {
		id, status, detail := expectedInspectionCommandProviderID(state, inspection)
		if outcome, failed := collectionResolutionFailure(label, "providers", index, status, detail); failed {
			return outcome
		}
		expected[index] = id
	}
	if slices.Equal(command.value.CommandProviders, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "providers", expected, command.value.CommandProviders)
}

func (commandProviderExistenceExpectationValue) schemaExpectation() {}

func (commandProviderExistenceExpectationValue) referenceExpectation() {}

func (commandProviderExistenceExpectationValue) inspectionExpectation() {}

func (expectation commandProviderExistenceExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	return declarationExistenceResult(resolveSchemaCommandProviderLocator(expectation.locator, schema), expectation.expected)
}

func (expectation commandProviderExistenceExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return declarationExistenceResult(resolveReferenceCommandProviderLocator(expectation.locator, reference), expectation.expected)
}

func (expectation commandProviderExistenceExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return declarationExistenceResult(resolveInspectionCommandProviderLocator(expectation.locator, inspection), expectation.expected)
}

func (commandProviderDependenciesExpectationValue) schemaExpectation() {}

func (commandProviderDependenciesExpectationValue) referenceExpectation() {}

func (commandProviderDependenciesExpectationValue) inspectionExpectation() {}

func (expectation commandProviderDependenciesExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaCommandProviderLocator(expectation.locator, schema)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText("exact dependencies"), evaluationText(""), false)
	}
	expected, status, detail := resolveExpectedValueIDsSchema(expectation.expected, schema)
	return expectedValueIDsResult(resolution.subject, resolution.value.Dependencies, expected, status, detail, semanticNameDependencies)
}

func (expectation commandProviderDependenciesExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceCommandProviderLocator(expectation.locator, reference)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText("exact dependencies"), evaluationText(""), false)
	}
	expected, status, detail := resolveExpectedValueIDsReference(expectation.expected, reference)
	return expectedValueIDsResult(resolution.subject, resolution.value.Dependencies, expected, status, detail, semanticNameDependencies)
}

func (expectation commandProviderDependenciesExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionCommandProviderLocator(expectation.locator, inspection)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText("exact dependencies"), evaluationText(""), false)
	}
	expected, status, detail := resolveExpectedValueIDsInspection(expectation.expected, inspection)
	return expectedValueIDsResult(resolution.subject, resolution.value.Dependencies, expected, status, detail, semanticNameDependencies)
}

func (providerLocalizationPresenceExpectation) inspectionExpectation() {}

func (e providerLocalizationPresenceExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	r := resolveInspectionCommandProviderLocator(e.locator, inspection)
	expect := "localization present"
	if !e.present {
		expect = "no localization"
	}
	if r.status != locatorResolutionFound {
		return outcomeForValueStatus(r.status, r.subject, evaluationText(expect), r.detail)
	}
	if !e.present && inspection.Truncated {
		return notProvableResult(r.subject, evaluationText(expect), observed(semanticSubjectInspection, observationTruncated))
	}
	actual := r.value.Localization != nil
	if actual == e.present {
		return provedResult()
	}
	return disprovedResult(r.subject, evaluationText(expect), evaluationText(fmt.Sprintf("localization present=%t", actual)))
}

func (providerLocalizationPayloadExpectation) inspectionExpectation() {}

func (e providerLocalizationPayloadExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	r := resolveInspectionCommandProviderLocator(e.locator, inspection)
	expect := qualifySubject(evaluationText("exact provider localization"), formatLocalization(e.expected))
	if r.status != locatorResolutionFound {
		return outcomeForValueStatus(r.status, r.subject, expect, r.detail)
	}
	if inspection.Truncated {
		return notProvableResult(qualifySubject(r.subject, evaluationText("localization")), expect, observed(semanticSubjectInspection, observationTruncated))
	}
	if r.value.Localization == nil {
		return disprovedResult(qualifySubject(r.subject, evaluationText("localization")), expect, evaluationText("localization is absent"))
	}
	if ok, actual := compareLocalization(e.expected, *r.value.Localization); !ok {
		return disprovedResult(qualifySubject(r.subject, evaluationText("localization")), expect, actual)
	}
	return provedResult()
}

func (providerEntriesExactlyExpectation) inspectionExpectation() {}

func (e providerEntriesExactlyExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	r := resolveInspectionCommandProviderLocator(e.locator, inspection)
	if r.status != locatorResolutionFound {
		return outcomeForValueStatus(
			r.status,
			r.subject,
			evaluationText("exact provider command entries in authoritative order"),
			r.detail,
		)
	}
	if inspection.Truncated {
		return notProvableResult(
			r.subject,
			evaluationText("exact provider command entries in authoritative order"),
			observed(semanticSubjectInspection, observationTruncated),
		)
	}
	expected := make([]cli.CommandEntryID, len(e.entries))
	for index, state := range e.entries {
		id, status, detail := expectedInspectionCommandEntryID(state, inspection)
		switch status {
		case locatorResolutionFound:
			expected[index] = id
		case locatorResolutionNotProvable:
			return notProvableResult(r.subject, evaluationText("exact provider command entries in authoritative order"), detail)
		case locatorResolutionAbsent, locatorResolutionAmbiguous, locatorResolutionInvalid:
			return disprovedResult(r.subject, evaluationText("exact provider command entries in authoritative order"), detail)
		default:
			panic("clitest: invalid locator resolution status")
		}
	}
	if slices.Equal(r.value.CommandEntries, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(r.subject, "entry IDs", expected, r.value.CommandEntries)
}

func (commandProviderOutcomeExpectationValue) inspectionExpectation() {}

func (expectation commandProviderOutcomeExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionCommandProviderLocator(expectation.locator, inspection)
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("outcome"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Outcome)), resolution.value.Outcome == expectation.expected)
}
