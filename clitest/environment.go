package clitest

import (
	"fmt"

	"github.com/abstractionsinc/commandment/cli"
)

// EnvironmentInputLocator identifies one environment-input declaration for
// clitest semantic expectation. Its zero value is invalid.
type EnvironmentInputLocator struct {
	_     [0]func()
	state *environmentInputLocatorState
}

// EnvironmentInputByID returns a locator for the environment input with the exact public
// EnvironmentInputID within the observation being evaluated.
func EnvironmentInputByID(id cli.EnvironmentInputID) EnvironmentInputLocator {
	return environmentInputIDLocator(id)
}

func (EnvironmentInputLocator) declarationIdentityLocator() {}

func (EnvironmentInputLocator) diagnosticSubject() {}

func (EnvironmentInputLocator) noticeSubject() {}

func (EnvironmentInputLocator) tokenOwner() {}

// Value returns a locator for the public value associated with the located environment input.
func (locator EnvironmentInputLocator) Value() ValueLocator { return environmentValueLocator(locator) }

// Exists returns an expectation that checks the located environment input exists. Positive
// existence may be satisfied by truthful partial output when the subject is present.
func (locator EnvironmentInputLocator) Exists() DeclarationExpectation {
	return environmentInputExistenceExpectation{
		locator:  locator.state,
		expected: true,
	}
}

// DoesNotExist returns an expectation that checks the located environment input does not exist; an
// incomplete observation cannot establish absence.
func (locator EnvironmentInputLocator) DoesNotExist() DeclarationExpectation {
	return environmentInputExistenceExpectation{
		locator:  locator.state,
		expected: false,
	}
}

// Sensitive returns an expectation that checks the located environment input is marked sensitive.
// Expectation failure preserves Commandment redaction and does not expose secret material.
func (locator EnvironmentInputLocator) Sensitive() DeclarationExpectation {
	return environmentInputSensitiveExpectationValue{locator: locator.state}
}

// Lifecycle returns an expectation that checks the located environment input has the expected
// effective lifecycle state; this is distinct from release-lifecycle metadata.
func (locator EnvironmentInputLocator) Lifecycle(v cli.LifecycleState) DeclarationExpectation {
	return environmentInputLifecycleExpectationValue{
		locator:  locator.state,
		expected: v,
	}
}

// ReplacedBy returns an expectation that checks the located environment input replacement points
// to the exact declaration identified by v, using public declaration identity rather than
// presentation text.
func (locator EnvironmentInputLocator) ReplacedBy(v DeclarationIdentityLocator) DeclarationExpectation {
	return environmentInputReplacementExpectationValue{
		locator:  locator.state,
		expected: v,
	}
}

// Available returns an expectation that checks the located environment input is available in the
// applicable public observation.
func (locator EnvironmentInputLocator) Available() EffectiveDeclarationExpectation {
	return environmentInputAvailabilityExpectation{
		locator:  locator.state,
		expected: true,
	}
}

// Unavailable returns an expectation that checks the located environment input is unavailable in
// the applicable public observation.
func (locator EnvironmentInputLocator) Unavailable() EffectiveDeclarationExpectation {
	return environmentInputAvailabilityExpectation{
		locator:  locator.state,
		expected: false,
	}
}

// Prefix returns a reusable declaration expectation that checks the public prefix mode exposed
// by the located environment-input declaration.
func (locator EnvironmentInputLocator) Prefix() DeclarationExpectation {
	return environmentInputCorePropertyExpectation{
		locator: locator.state,
		kind:    environmentInputCorePropertyPrefix,
		prefix:  true,
	}
}

// NotPrefix returns a reusable declaration expectation that checks the located
// environment-input declaration does not use prefix mode.
func (locator EnvironmentInputLocator) NotPrefix() DeclarationExpectation {
	return environmentInputCorePropertyExpectation{
		locator: locator.state,
		kind:    environmentInputCorePropertyPrefix,
		prefix:  false,
	}
}

// Delimiter returns a reusable declaration expectation that checks the public delimiter exposed
// by the located environment-input declaration.
func (locator EnvironmentInputLocator) Delimiter(expected string) DeclarationExpectation {
	return environmentInputCorePropertyExpectation{
		locator:   locator.state,
		kind:      environmentInputCorePropertyDelimiter,
		delimiter: expected,
	}
}

// Origin returns an expectation that checks the located environment input has the
// framework-established DeclarationOrigin expected.
func (locator EnvironmentInputLocator) Origin(expected cli.DeclarationOrigin) DeclarationExpectation {
	return environmentInputOriginExpectation{
		locator:  locator.state,
		expected: expected,
	}
}

// ReleaseLifecycle returns an expectation that checks the located environment input has the exact
// public release-lifecycle metadata expected.
func (locator EnvironmentInputLocator) ReleaseLifecycle(expected cli.ReleaseLifecycle) DeclarationExpectation {
	return environmentInputReleaseLifecycleExpectation{
		locator:  locator.state,
		expected: expected,
	}
}

// NotReplaced returns an expectation that checks the located environment input has no public
// declaration-replacement relationship. Incomplete observation cannot manufacture replacement
// absence.
func (locator EnvironmentInputLocator) NotReplaced() DeclarationExpectation {
	return environmentInputReplacementAbsenceExpectation{locator: locator.state}
}

// ReplacementPresentation returns an expectation that checks the located environment input
// replacement carries the expected public presentation text.
func (locator EnvironmentInputLocator) ReplacementPresentation(expected string) DeclarationExpectation {
	return environmentInputReplacementPresentationExpectation{
		locator:  locator.state,
		expected: expected,
	}
}

// ReplacementTargetDeprecatedDuringMigration returns an expectation that checks the located
// environment input replacement carries the expected migration-deprecation marker.
func (locator EnvironmentInputLocator) ReplacementTargetDeprecatedDuringMigration(expected bool) DeclarationExpectation {
	return environmentInputReplacementTargetDeprecatedExpectation{
		locator:  locator.state,
		expected: expected,
	}
}

// Environment returns a locator for the uniquely resolved environment input named name under the
// located command.
func (locator CommandLocator) Environment(name string) EnvironmentInputLocator {
	return commandEnvironmentLocator(locator, name)
}

// EnvironmentAt returns a locator for the environment input at index in the located command's
// authoritative public environment-input order.
func (locator CommandLocator) EnvironmentAt(index int) EnvironmentInputLocator {
	return commandEnvironmentAtLocator(locator, index)
}

// EnvironmentInputsExactly returns a declaration expectation that checks the command's complete
// environment-input collection by public semantic locator and authoritative order.
func (locator CommandLocator) EnvironmentInputsExactly(inputs ...EnvironmentInputLocator) DeclarationExpectation {
	expected := make([]*environmentInputLocatorState, len(inputs))
	for index, input := range inputs {
		expected[index] = input.state
	}
	return commandEnvironmentInputsExactlyExpectation{
		command: locator.state,
		inputs:  expected,
	}
}

type environmentInputLocatorMode uint8

const (
	environmentInputLocatorInvalid environmentInputLocatorMode = iota
	environmentInputLocatorID
	environmentInputLocatorCommandName
	environmentInputLocatorCommandIndex
)

type environmentInputLocatorState struct {
	mode environmentInputLocatorMode
	id   cli.EnvironmentInputID

	command *commandLocatorState
	name    string
	index   int
}

type environmentInputAvailabilityExpectation struct {
	locator  *environmentInputLocatorState
	expected bool
}

type commandEnvironmentInputsExactlyExpectation struct {
	command *commandLocatorState
	inputs  []*environmentInputLocatorState
}

type environmentInputOriginExpectation struct {
	locator  *environmentInputLocatorState
	expected cli.DeclarationOrigin
}

type environmentInputReleaseLifecycleExpectation struct {
	locator  *environmentInputLocatorState
	expected cli.ReleaseLifecycle
}

type environmentInputReplacementAbsenceExpectation struct{ locator *environmentInputLocatorState }

type environmentInputReplacementPresentationExpectation struct {
	locator  *environmentInputLocatorState
	expected string
}

type environmentInputReplacementTargetDeprecatedExpectation struct {
	locator  *environmentInputLocatorState
	expected bool
}

type environmentInputExistenceExpectation struct {
	locator  *environmentInputLocatorState
	expected bool
}

type (
	environmentInputSensitiveExpectationValue struct{ locator *environmentInputLocatorState }
	environmentInputLifecycleExpectationValue struct {
		locator  *environmentInputLocatorState
		expected cli.LifecycleState
	}
)

type environmentInputReplacementExpectationValue struct {
	locator  *environmentInputLocatorState
	expected DeclarationIdentityLocator
}

type environmentInputCorePropertyKind uint8

const (
	environmentInputCorePropertyInvalid environmentInputCorePropertyKind = iota
	environmentInputCorePropertyPrefix
	environmentInputCorePropertyDelimiter
)

type environmentInputCorePropertyExpectation struct {
	locator   *environmentInputLocatorState
	kind      environmentInputCorePropertyKind
	prefix    bool
	delimiter string
}

func environmentInputLocatorLabel(state *environmentInputLocatorState) evaluationField {
	if state == nil {
		return semanticSubjectEnvironmentInput
	}
	switch state.mode {
	case environmentInputLocatorID:
		return evaluationText(fmt.Sprintf("environment input id %d", state.id))
	case environmentInputLocatorCommandName:
		return qualifySubject(commandLocatorLabel(state.command), subjectNamed(evaluationText("environment"), state.name))
	case environmentInputLocatorCommandIndex:
		return qualifySubject(commandLocatorLabel(state.command), evaluationText(fmt.Sprintf("environment input %d", state.index)))
	default:
		return semanticSubjectEnvironmentInput
	}
}

func resolveSchemaEnvironmentInputLocator(state *environmentInputLocatorState, schema *cli.Schema) locatorResolution[cli.EnvironmentInputSchema] {
	var resolution locatorResolution[cli.EnvironmentInputSchema]
	label := environmentInputLocatorLabel(state)
	if schema == nil || state == nil || state.mode == environmentInputLocatorInvalid {
		resolution = locatorResolution[cli.EnvironmentInputSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectEnvironmentInput, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == environmentInputLocatorID {
		value, ok := schema.EnvironmentInputs[state.id]
		if !ok {
			resolution = locatorResolution[cli.EnvironmentInputSchema]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectEnvironmentInput, observationAbsent),
			}
			return resolution
		}
		resolution = locatorResolution[cli.EnvironmentInputSchema]{
			status:  locatorResolutionFound,
			value:   value,
			subject: label,
		}
		return resolution
	}
	command := resolveSchemaCommandLocator(state.command, schema)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.EnvironmentInputSchema]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	ids := schemaEnvironmentOrder(command.value, *schema)
	id, status, detail := selectSchemaEnvironmentInput(state, ids, schema, true)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.EnvironmentInputSchema]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	value, ok := schema.EnvironmentInputs[id]
	if !ok {
		resolution = locatorResolution[cli.EnvironmentInputSchema]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("environment-input relationship points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.EnvironmentInputSchema]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

func resolveReferenceEnvironmentInputLocator(state *environmentInputLocatorState, reference cli.Reference) locatorResolution[cli.EnvironmentInputReference] {
	var resolution locatorResolution[cli.EnvironmentInputReference]
	label := environmentInputLocatorLabel(state)
	if state == nil || state.mode == environmentInputLocatorInvalid {
		resolution = locatorResolution[cli.EnvironmentInputReference]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectEnvironmentInput, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == environmentInputLocatorID {
		value, ok := reference.EnvironmentInputs[state.id]
		if !ok {
			resolution = locatorResolution[cli.EnvironmentInputReference]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectEnvironmentInput, observationAbsent),
			}
			return resolution
		}
		resolution = locatorResolution[cli.EnvironmentInputReference]{
			status:  locatorResolutionFound,
			value:   value,
			subject: label,
		}
		return resolution
	}
	command := resolveReferenceCommandLocator(state.command, reference)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.EnvironmentInputReference]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	ids := referenceEnvironmentOrder(command.value, reference)
	id, status, detail := selectReferenceEnvironmentInput(state, ids, reference, true)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.EnvironmentInputReference]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	value, ok := reference.EnvironmentInputs[id]
	if !ok {
		resolution = locatorResolution[cli.EnvironmentInputReference]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("environment-input relationship points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.EnvironmentInputReference]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

func resolveInspectionEnvironmentInputLocator(state *environmentInputLocatorState, inspection cli.Inspection) locatorResolution[cli.EnvironmentInputInspection] {
	var resolution locatorResolution[cli.EnvironmentInputInspection]
	label := environmentInputLocatorLabel(state)
	if state == nil || state.mode == environmentInputLocatorInvalid {
		resolution = locatorResolution[cli.EnvironmentInputInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectEnvironmentInput, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == environmentInputLocatorID {
		value, ok := inspection.EnvironmentInputs[state.id]
		if ok {
			resolution = locatorResolution[cli.EnvironmentInputInspection]{
				status:  locatorResolutionFound,
				value:   value,
				subject: label,
			}
			return resolution
		}
		if inspection.Truncated {
			resolution = locatorResolution[cli.EnvironmentInputInspection]{
				status:  locatorResolutionNotProvable,
				subject: label,
				detail:  observed(semanticSubjectInspection, observationTruncated),
			}
			return resolution
		}
		resolution = locatorResolution[cli.EnvironmentInputInspection]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectEnvironmentInput, observationAbsent),
		}
		return resolution
	}
	command := resolveInspectionCommandLocator(state.command, inspection)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.EnvironmentInputInspection]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	if state.mode == environmentInputLocatorCommandIndex {
		resolution = locatorResolution[cli.EnvironmentInputInspection]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("inspection does not expose authoritative command environment-input order"),
		}
		return resolution
	}
	matches := make([]cli.EnvironmentInputInspection, 0, 1)
	for _, value := range inspection.EnvironmentInputs {
		if value.Command == command.value.ID && value.Name == state.name {
			matches = append(matches, value)
		}
	}
	return resolveUniqueObserved(matches, !inspection.Truncated && command.value.NamespaceCompleteness == cli.CommandNamespaceComplete, label, observed(semanticSubjectEnvironmentInput, observationAbsent), observed(semanticSubjectInspection, observationTruncated), evaluationText("environment name is ambiguous"))
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
//nolint:dupl // Separate public artifacts resolve their own environment-input relationships directly.
func selectSchemaEnvironmentInput(state *environmentInputLocatorState, ids []cli.EnvironmentInputID, schema *cli.Schema, complete bool) (cli.EnvironmentInputID, locatorResolutionStatus, evaluationField) {
	switch state.mode {
	case environmentInputLocatorCommandIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectEnvironmentInput, observationInvalidLocator)
		}
		if state.index < len(ids) {
			return ids[state.index], locatorResolutionFound, evaluationText("")
		}
		if complete {
			return 0, locatorResolutionAbsent, observed(semanticSubjectEnvironmentInput, observationAbsent)
		}
		return 0, locatorResolutionNotProvable, evaluationText("environment-input lookup domain is incomplete")
	case environmentInputLocatorCommandName:
		matches := make([]cli.EnvironmentInputID, 0, 1)
		missingDetail := false
		for _, id := range ids {
			value, ok := schema.EnvironmentInputs[id]
			if !ok {
				missingDetail = true
				continue
			}
			if value.Name == state.name {
				matches = append(matches, id)
			}
		}
		if len(matches) == 1 {
			return matches[0], locatorResolutionFound, evaluationText("")
		}
		if len(matches) > 1 {
			return 0, locatorResolutionAmbiguous, collectionCardinalityField{
				name:  "matches environment-name IDs",
				count: len(matches),
			}
		}
		if missingDetail || !complete {
			return 0, locatorResolutionNotProvable, evaluationText("environment-input lookup domain is incomplete")
		}
		return 0, locatorResolutionAbsent, observed(semanticSubjectEnvironmentInput, observationAbsent)
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectEnvironmentInput, observationInvalidLocator)
	}
}

//nolint:dupl // Separate public artifacts resolve their own environment-input relationships directly.
func selectReferenceEnvironmentInput(state *environmentInputLocatorState, ids []cli.EnvironmentInputID, reference cli.Reference, complete bool) (cli.EnvironmentInputID, locatorResolutionStatus, evaluationField) {
	switch state.mode {
	case environmentInputLocatorCommandIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectEnvironmentInput, observationInvalidLocator)
		}
		if state.index < len(ids) {
			return ids[state.index], locatorResolutionFound, evaluationText("")
		}
		if complete {
			return 0, locatorResolutionAbsent, observed(semanticSubjectEnvironmentInput, observationAbsent)
		}
		return 0, locatorResolutionNotProvable, evaluationText("environment-input lookup domain is incomplete")
	case environmentInputLocatorCommandName:
		matches := make([]cli.EnvironmentInputID, 0, 1)
		missingDetail := false
		for _, id := range ids {
			value, ok := reference.EnvironmentInputs[id]
			if !ok {
				missingDetail = true
				continue
			}
			if value.Name == state.name {
				matches = append(matches, id)
			}
		}
		if len(matches) == 1 {
			return matches[0], locatorResolutionFound, evaluationText("")
		}
		if len(matches) > 1 {
			return 0, locatorResolutionAmbiguous, collectionCardinalityField{
				name:  "matches environment-name IDs",
				count: len(matches),
			}
		}
		if missingDetail || !complete {
			return 0, locatorResolutionNotProvable, evaluationText("environment-input lookup domain is incomplete")
		}
		return 0, locatorResolutionAbsent, observed(semanticSubjectEnvironmentInput, observationAbsent)
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectEnvironmentInput, observationInvalidLocator)
	}
}

func schemaEnvironmentOrder(c cli.CommandSchema, a cli.Schema) []cli.EnvironmentInputID {
	var out []cli.EnvironmentInputID
	for _, vid := range c.Values {
		if v, ok := a.Values[vid]; ok {
			out = append(out, v.EnvironmentInputs...)
		}
	}
	return out
}

func referenceEnvironmentOrder(c cli.CommandReference, a cli.Reference) []cli.EnvironmentInputID {
	var out []cli.EnvironmentInputID
	for _, vid := range c.Values {
		if v, ok := a.Values[vid]; ok {
			out = append(out, v.EnvironmentInputs...)
		}
	}
	return out
}

func commandEnvironmentInputIDsSchema(state *commandLocatorState, schema *cli.Schema) ([]cli.EnvironmentInputID, locatorResolutionStatus, evaluationField) {
	resolved := resolveSchemaCommandLocator(state, schema)
	if resolved.status != locatorResolutionFound {
		return nil, resolved.status, resolved.detail
	}
	ids := make([]cli.EnvironmentInputID, 0)
	for id, input := range schema.EnvironmentInputs {
		if input.Command == resolved.value.ID {
			ids = append(ids, id)
		}
	}
	return ids, locatorResolutionFound, evaluationText("")
}

func commandEnvironmentInputIDsReference(state *commandLocatorState, reference cli.Reference) ([]cli.EnvironmentInputID, locatorResolutionStatus, evaluationField) {
	resolved := resolveReferenceCommandLocator(state, reference)
	if resolved.status != locatorResolutionFound {
		return nil, resolved.status, resolved.detail
	}
	ids := make([]cli.EnvironmentInputID, 0)
	for id, input := range reference.EnvironmentInputs {
		if input.Command == resolved.value.ID {
			ids = append(ids, id)
		}
	}
	return ids, locatorResolutionFound, evaluationText("")
}

func commandEnvironmentInputIDsInspection(state *commandLocatorState, inspection cli.Inspection) ([]cli.EnvironmentInputID, locatorResolutionStatus, evaluationField) {
	resolved := resolveInspectionCommandLocator(state, inspection)
	if resolved.status != locatorResolutionFound {
		return nil, resolved.status, resolved.detail
	}
	ids := make([]cli.EnvironmentInputID, 0)
	for id, input := range inspection.EnvironmentInputs {
		if input.Command == resolved.value.ID {
			ids = append(ids, id)
		}
	}
	return ids, locatorResolutionFound, evaluationText("")
}

func compareEnvironmentInputMembership(actual, expected []cli.EnvironmentInputID) evaluationResult {
	if len(actual) != len(expected) {
		return disprovedResult(evaluationText("command environment inputs"), evaluationText(fmt.Sprintf("exactly %d semantic inputs", len(expected))), evaluationText(fmt.Sprintf("%d inputs", len(actual))))
	}
	actualSet := make(map[cli.EnvironmentInputID]struct{}, len(actual))
	for _, id := range actual {
		actualSet[id] = struct{}{}
	}
	seen := make(map[cli.EnvironmentInputID]struct{}, len(expected))
	for index, id := range expected {
		if _, duplicate := seen[id]; duplicate {
			return disprovedResult(evaluationText("command environment inputs"), evaluationText("distinct semantic inputs"), evaluationText(fmt.Sprintf("duplicate expected input at position %d", index)))
		}
		seen[id] = struct{}{}
		if _, ok := actualSet[id]; !ok {
			return disprovedResult(evaluationText("command environment inputs"), evaluationText("exact semantic membership"), evaluationText(fmt.Sprintf("expected input at position %d is not owned by command", index)))
		}
	}
	return provedResult()
}

func evaluateEnvironmentInputReplacementPresentationSchema(expectation environmentInputReplacementPresentationExpectation, schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaEnvironmentInputLocator(expectation.locator, schema)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func evaluateEnvironmentInputReplacementPresentationReference(expectation environmentInputReplacementPresentationExpectation, reference cli.Reference) evaluationResult {
	resolution := resolveReferenceEnvironmentInputLocator(expectation.locator, reference)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func evaluateEnvironmentInputReplacementPresentationInspection(expectation environmentInputReplacementPresentationExpectation, inspection cli.Inspection) evaluationResult {
	resolution := resolveInspectionEnvironmentInputLocator(expectation.locator, inspection)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func environmentInputIDLocator(id cli.EnvironmentInputID) EnvironmentInputLocator {
	return EnvironmentInputLocator{state: &environmentInputLocatorState{
		mode: environmentInputLocatorID,
		id:   id,
	}}
}

func (environmentInputAvailabilityExpectation) referenceExpectation() {}

func (environmentInputAvailabilityExpectation) inspectionExpectation() {}

func (expectation environmentInputAvailabilityExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceEnvironmentInputLocator(expectation.locator, reference)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "available", expectation.expected, resolution.value.Available)
}

func (expectation environmentInputAvailabilityExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionEnvironmentInputLocator(expectation.locator, inspection)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "available", expectation.expected, resolution.value.Available)
}

func (commandEnvironmentInputsExactlyExpectation) schemaExpectation() {}

func (commandEnvironmentInputsExactlyExpectation) referenceExpectation() {}

func (commandEnvironmentInputsExactlyExpectation) inspectionExpectation() {}

func (expectation commandEnvironmentInputsExactlyExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	actual, status, detail := commandEnvironmentInputIDsSchema(expectation.command, schema)
	if status != locatorResolutionFound {
		return declarationMissing(status, evaluationText("command environment inputs"), evaluationText("exact semantic membership"), detail, false)
	}
	expected := make([]cli.EnvironmentInputID, len(expectation.inputs))
	for index, state := range expectation.inputs {
		resolved := resolveSchemaEnvironmentInputLocator(state, schema)
		if resolved.status != locatorResolutionFound {
			return declarationMissing(resolved.status, evaluationText("command environment inputs"), evaluationText("exact semantic membership"), resolved.detail, false)
		}
		expected[index] = resolved.value.ID
	}
	return compareEnvironmentInputMembership(actual, expected)
}

func (expectation commandEnvironmentInputsExactlyExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	actual, status, detail := commandEnvironmentInputIDsReference(expectation.command, reference)
	if status != locatorResolutionFound {
		return declarationMissing(status, evaluationText("command environment inputs"), evaluationText("exact semantic membership"), detail, false)
	}
	expected := make([]cli.EnvironmentInputID, len(expectation.inputs))
	for index, state := range expectation.inputs {
		resolved := resolveReferenceEnvironmentInputLocator(state, reference)
		if resolved.status != locatorResolutionFound {
			return declarationMissing(resolved.status, evaluationText("command environment inputs"), evaluationText("exact semantic membership"), resolved.detail, false)
		}
		expected[index] = resolved.value.ID
	}
	return compareEnvironmentInputMembership(actual, expected)
}

func (expectation commandEnvironmentInputsExactlyExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	actual, status, detail := commandEnvironmentInputIDsInspection(expectation.command, inspection)
	if status != locatorResolutionFound {
		return declarationMissing(status, evaluationText("command environment inputs"), evaluationText("exact semantic membership"), detail, inspection.Truncated)
	}
	expected := make([]cli.EnvironmentInputID, len(expectation.inputs))
	for index, state := range expectation.inputs {
		resolved := resolveInspectionEnvironmentInputLocator(state, inspection)
		if resolved.status != locatorResolutionFound {
			return declarationMissing(resolved.status, evaluationText("command environment inputs"), evaluationText("exact semantic membership"), resolved.detail, inspection.Truncated)
		}
		expected[index] = resolved.value.ID
	}
	outcome := compareEnvironmentInputMembership(actual, expected)
	if outcome.status != evaluationProved && inspection.Truncated {
		return notProvableResult(evaluationText("command environment inputs"), evaluationText("exact semantic membership"), observed(semanticSubjectInspection, observationTruncated))
	}
	return outcome
}

func (environmentInputOriginExpectation) schemaExpectation() {}

func (environmentInputOriginExpectation) referenceExpectation() {}

func (environmentInputOriginExpectation) inspectionExpectation() {}

func (expectation environmentInputOriginExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaEnvironmentInputLocator(expectation.locator, schema)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (expectation environmentInputOriginExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceEnvironmentInputLocator(expectation.locator, reference)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (expectation environmentInputOriginExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionEnvironmentInputLocator(expectation.locator, inspection)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (environmentInputReleaseLifecycleExpectation) schemaExpectation() {}

func (environmentInputReleaseLifecycleExpectation) referenceExpectation() {}

func (environmentInputReleaseLifecycleExpectation) inspectionExpectation() {}

func (expectation environmentInputReleaseLifecycleExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaEnvironmentInputLocator(expectation.locator, schema)
	actual := resolution.value.Release
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, releaseLifecycleField{value: expectation.expected}, releaseLifecycleField{value: actual}, actual == expectation.expected)
}

func (expectation environmentInputReleaseLifecycleExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceEnvironmentInputLocator(expectation.locator, reference)
	actual := resolution.value.Release
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, releaseLifecycleField{value: expectation.expected}, releaseLifecycleField{value: actual}, actual == expectation.expected)
}

func (expectation environmentInputReleaseLifecycleExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionEnvironmentInputLocator(expectation.locator, inspection)
	actual := resolution.value.Release
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, releaseLifecycleField{value: expectation.expected}, releaseLifecycleField{value: actual}, actual == expectation.expected)
}

func (environmentInputReplacementAbsenceExpectation) schemaExpectation() {}

func (environmentInputReplacementAbsenceExpectation) referenceExpectation() {}

func (environmentInputReplacementAbsenceExpectation) inspectionExpectation() {}

func (expectation environmentInputReplacementAbsenceExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaEnvironmentInputLocator(expectation.locator, schema)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (expectation environmentInputReplacementAbsenceExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceEnvironmentInputLocator(expectation.locator, reference)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (expectation environmentInputReplacementAbsenceExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionEnvironmentInputLocator(expectation.locator, inspection)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (environmentInputReplacementPresentationExpectation) schemaExpectation() {}

func (environmentInputReplacementPresentationExpectation) referenceExpectation() {}

func (environmentInputReplacementPresentationExpectation) inspectionExpectation() {}

func (expectation environmentInputReplacementPresentationExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	return evaluateEnvironmentInputReplacementPresentationSchema(expectation, schema)
}

func (expectation environmentInputReplacementPresentationExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return evaluateEnvironmentInputReplacementPresentationReference(expectation, reference)
}

func (expectation environmentInputReplacementPresentationExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return evaluateEnvironmentInputReplacementPresentationInspection(expectation, inspection)
}

func (environmentInputReplacementTargetDeprecatedExpectation) schemaExpectation() {}

func (environmentInputReplacementTargetDeprecatedExpectation) referenceExpectation() {}

func (environmentInputReplacementTargetDeprecatedExpectation) inspectionExpectation() {}

func (expectation environmentInputReplacementTargetDeprecatedExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaEnvironmentInputLocator(expectation.locator, schema)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (expectation environmentInputReplacementTargetDeprecatedExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceEnvironmentInputLocator(expectation.locator, reference)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (expectation environmentInputReplacementTargetDeprecatedExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionEnvironmentInputLocator(expectation.locator, inspection)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (environmentInputExistenceExpectation) schemaExpectation() {}

func (environmentInputExistenceExpectation) referenceExpectation() {}

func (environmentInputExistenceExpectation) inspectionExpectation() {}

func (expectation environmentInputExistenceExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	return declarationExistenceResult(resolveSchemaEnvironmentInputLocator(expectation.locator, schema), expectation.expected)
}

func (expectation environmentInputExistenceExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return declarationExistenceResult(resolveReferenceEnvironmentInputLocator(expectation.locator, reference), expectation.expected)
}

func (expectation environmentInputExistenceExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return declarationExistenceResult(resolveInspectionEnvironmentInputLocator(expectation.locator, inspection), expectation.expected)
}

func (environmentInputSensitiveExpectationValue) schemaExpectation() {}

func (environmentInputSensitiveExpectationValue) referenceExpectation() {}

func (environmentInputSensitiveExpectationValue) inspectionExpectation() {}

func (expectation environmentInputSensitiveExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaEnvironmentInputLocator(expectation.locator, schema)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, semanticRequirementSensitive, true, resolution.value.Sensitive)
}

func (expectation environmentInputSensitiveExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceEnvironmentInputLocator(expectation.locator, reference)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, semanticRequirementSensitive, true, resolution.value.Sensitive)
}

func (expectation environmentInputSensitiveExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionEnvironmentInputLocator(expectation.locator, inspection)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, semanticRequirementSensitive, true, resolution.value.Sensitive)
}

func (environmentInputLifecycleExpectationValue) schemaExpectation() {}

func (environmentInputLifecycleExpectationValue) referenceExpectation() {}

func (environmentInputLifecycleExpectationValue) inspectionExpectation() {}

func (expectation environmentInputLifecycleExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaEnvironmentInputLocator(expectation.locator, schema)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (expectation environmentInputLifecycleExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceEnvironmentInputLocator(expectation.locator, reference)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (expectation environmentInputLifecycleExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionEnvironmentInputLocator(expectation.locator, inspection)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (environmentInputReplacementExpectationValue) schemaExpectation() {}

func (environmentInputReplacementExpectationValue) referenceExpectation() {}

func (environmentInputReplacementExpectationValue) inspectionExpectation() {}

func (expectation environmentInputReplacementExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaEnvironmentInputLocator(expectation.locator, schema)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, schema, nil, cli.Inspection{})
}

func (expectation environmentInputReplacementExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceEnvironmentInputLocator(expectation.locator, reference)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, nil, &reference, cli.Inspection{})
}

func (expectation environmentInputReplacementExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionEnvironmentInputLocator(expectation.locator, inspection)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, nil, nil, inspection)
}

func (environmentInputCorePropertyExpectation) schemaExpectation() {}

func (environmentInputCorePropertyExpectation) referenceExpectation() {}

func (environmentInputCorePropertyExpectation) inspectionExpectation() {}

func (expectation environmentInputCorePropertyExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaEnvironmentInputLocator(expectation.locator, schema)
	return expectation.evaluate(resolution.status, resolution.subject, resolution.detail, resolution.value.Prefix, resolution.value.Delimiter)
}

func (expectation environmentInputCorePropertyExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceEnvironmentInputLocator(expectation.locator, reference)
	return expectation.evaluate(resolution.status, resolution.subject, resolution.detail, resolution.value.Prefix, resolution.value.Delimiter)
}

func (expectation environmentInputCorePropertyExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionEnvironmentInputLocator(expectation.locator, inspection)
	return expectation.evaluate(resolution.status, resolution.subject, resolution.detail, resolution.value.Prefix, resolution.value.Delimiter)
}

func (expectation environmentInputCorePropertyExpectation) evaluate(status locatorResolutionStatus, subject, detail evaluationField, prefix bool, delimiter string) evaluationResult {
	switch expectation.kind {
	case environmentInputCorePropertyPrefix:
		return resolvedBoolProperty(status, subject, detail, evaluationText("prefix"), expectation.prefix, prefix)
	case environmentInputCorePropertyDelimiter:
		return declarationResolvedComparison(status, subject, detail, evaluationText("delimiter"), evaluationText(delimiter), delimiter == expectation.delimiter)
	default:
		panic("clitest: invalid environment-input core property expectation")
	}
}
