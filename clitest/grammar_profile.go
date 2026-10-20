package clitest

import (
	"fmt"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
)

// GrammarProfileLocator identifies one grammar profile for clitest semantic
// expectation. Its zero value is invalid.
type GrammarProfileLocator struct {
	_     [0]func()
	state *grammarProfileLocatorState
}

// GrammarProfileByID returns a locator for the grammar profile with the exact public
// GrammarProfileID within the observation being evaluated.
func GrammarProfileByID(id cli.GrammarProfileID) GrammarProfileLocator {
	return grammarProfileIDLocator(id)
}

// GrammarProfile returns a locator for the uniquely resolved grammar profile whose public value is
// value under the located command.
func (locator CommandLocator) GrammarProfile(value string) GrammarProfileLocator {
	return commandGrammarProfileLocator(locator, value)
}

// GrammarProfileAt returns a locator for the grammar profile at index in the located command's
// authoritative public profile order.
func (locator CommandLocator) GrammarProfileAt(index int) GrammarProfileLocator {
	return commandGrammarProfileAtLocator(locator, index)
}

// GrammarProfilesExactly returns a static declaration expectation that checks the command's
// complete grammar-profile sequence by public profile identity and authoritative order.
func (locator CommandLocator) GrammarProfilesExactly(expected ...GrammarProfileLocator) StaticDeclarationExpectation {
	profiles := make([]*grammarProfileLocatorState, len(expected))
	for index, profile := range expected {
		profiles[index] = profile.state
	}
	return commandGrammarProfilesExactlyExpectation{
		command:  locator.state,
		profiles: profiles,
	}
}

// Exists returns an expectation that checks the located grammar profile exists. Positive existence
// may be satisfied by truthful partial output when the subject is present.
func (locator GrammarProfileLocator) Exists() StaticDeclarationExpectation {
	return grammarProfileExistenceExpectationValue{
		locator:  locator.state,
		expected: true,
	}
}

// DoesNotExist returns an expectation that checks the located grammar profile does not exist; an
// incomplete observation cannot establish absence.
func (locator GrammarProfileLocator) DoesNotExist() StaticDeclarationExpectation {
	return grammarProfileExistenceExpectationValue{
		locator:  locator.state,
		expected: false,
	}
}

// Reachability returns an expectation that checks the located grammar profile has the expected
// public reachability state.
func (locator GrammarProfileLocator) Reachability(v cli.Reachability) StaticDeclarationExpectation {
	return grammarProfileReachabilityExpectationValue{
		locator:  locator.state,
		expected: v,
	}
}

// Origin returns an expectation that checks the located grammar profile has the
// framework-established DeclarationOrigin expected.
func (locator GrammarProfileLocator) Origin(expected cli.DeclarationOrigin) StaticDeclarationExpectation {
	return grammarProfileOriginExpectationValue{
		locator:  locator.state,
		expected: expected,
	}
}

type grammarProfileLocatorMode uint8

const (
	grammarProfileLocatorInvalid grammarProfileLocatorMode = iota
	grammarProfileLocatorID
	grammarProfileLocatorValue
	grammarProfileLocatorIndex
)

type grammarProfileLocatorState struct {
	mode grammarProfileLocatorMode
	id   cli.GrammarProfileID

	command *commandLocatorState
	value   string
	index   int
}

type argumentFormGrammarProfileExpectation struct {
	form    *argumentFormLocatorState
	profile *grammarProfileLocatorState
}

type commandDefaultGrammarProfileExpectation struct {
	command *commandLocatorState
	profile *grammarProfileLocatorState
}

type commandGrammarProfilesExactlyExpectation struct {
	command  *commandLocatorState
	profiles []*grammarProfileLocatorState
}

type grammarProfileExistenceExpectationValue struct {
	locator  *grammarProfileLocatorState
	expected bool
}

type grammarProfileReachabilityExpectationValue struct {
	locator  *grammarProfileLocatorState
	expected cli.Reachability
}

type grammarProfileOriginExpectationValue struct {
	locator  *grammarProfileLocatorState
	expected cli.DeclarationOrigin
}

func commandGrammarProfileLocator(parent CommandLocator, value string) GrammarProfileLocator {
	return GrammarProfileLocator{state: &grammarProfileLocatorState{
		mode:    grammarProfileLocatorValue,
		command: parent.state,
		value:   value,
	}}
}

func commandGrammarProfileAtLocator(parent CommandLocator, index int) GrammarProfileLocator {
	return GrammarProfileLocator{state: &grammarProfileLocatorState{
		mode:    grammarProfileLocatorIndex,
		command: parent.state,
		index:   index,
	}}
}

func grammarProfileLocatorLabel(state *grammarProfileLocatorState) evaluationField {
	if state == nil {
		return semanticSubjectGrammarProfile
	}
	switch state.mode {
	case grammarProfileLocatorID:
		return evaluationText(fmt.Sprintf("grammar profile id %d", state.id))
	case grammarProfileLocatorValue:
		return qualifySubject(commandLocatorLabel(state.command), subjectNamed(evaluationText("grammar profile"), state.value))
	case grammarProfileLocatorIndex:
		return qualifySubject(commandLocatorLabel(state.command), evaluationText(fmt.Sprintf("grammar profile %d", state.index)))
	default:
		return semanticSubjectGrammarProfile
	}
}

func resolveSchemaGrammarProfileLocator(state *grammarProfileLocatorState, schema *cli.Schema) locatorResolution[cli.GrammarProfile] {
	var resolution locatorResolution[cli.GrammarProfile]
	label := grammarProfileLocatorLabel(state)
	if schema == nil || state == nil || state.mode == grammarProfileLocatorInvalid {
		resolution = locatorResolution[cli.GrammarProfile]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectGrammarProfile, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == grammarProfileLocatorID {
		for _, command := range schema.Commands {
			if command.Profiles == nil {
				continue
			}
			if profile, ok := command.Profiles.Profiles[state.id]; ok {
				resolution = locatorResolution[cli.GrammarProfile]{
					status:  locatorResolutionFound,
					value:   profile,
					subject: label,
				}
				return resolution
			}
		}
		resolution = locatorResolution[cli.GrammarProfile]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectGrammarProfile, observationAbsent),
		}
		return resolution
	}
	command := resolveSchemaCommandLocator(state.command, schema)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.GrammarProfile]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	return resolveGrammarProfileSet(state, command.value.Profiles, label)
}

func resolveReferenceGrammarProfileLocator(state *grammarProfileLocatorState, reference cli.Reference) locatorResolution[cli.GrammarProfile] {
	var resolution locatorResolution[cli.GrammarProfile]
	label := grammarProfileLocatorLabel(state)
	if state == nil || state.mode == grammarProfileLocatorInvalid {
		resolution = locatorResolution[cli.GrammarProfile]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectGrammarProfile, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == grammarProfileLocatorID {
		for _, command := range reference.Commands {
			if command.Profiles == nil {
				continue
			}
			if profile, ok := command.Profiles.Profiles[state.id]; ok {
				resolution = locatorResolution[cli.GrammarProfile]{
					status:  locatorResolutionFound,
					value:   profile,
					subject: label,
				}
				return resolution
			}
		}
		resolution = locatorResolution[cli.GrammarProfile]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectGrammarProfile, observationAbsent),
		}
		return resolution
	}
	command := resolveReferenceCommandLocator(state.command, reference)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.GrammarProfile]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	return resolveGrammarProfileSet(state, command.value.Profiles, label)
}

func resolveGrammarProfileSet(state *grammarProfileLocatorState, set *cli.GrammarProfileSet, label evaluationField) locatorResolution[cli.GrammarProfile] {
	var resolution locatorResolution[cli.GrammarProfile]
	if set == nil {
		resolution = locatorResolution[cli.GrammarProfile]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectGrammarProfileSet, observationAbsent),
		}
		return resolution
	}
	var id cli.GrammarProfileID
	switch state.mode {
	case grammarProfileLocatorIndex:
		if state.index < 0 {
			resolution = locatorResolution[cli.GrammarProfile]{
				status:  locatorResolutionInvalid,
				subject: label,
				detail:  observed(semanticSubjectGrammarProfile, observationInvalidLocator),
			}
			return resolution
		}
		if state.index >= len(set.Cases) {
			resolution = locatorResolution[cli.GrammarProfile]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectGrammarProfile, observationAbsent),
			}
			return resolution
		}
		id = set.Cases[state.index]
	case grammarProfileLocatorValue:
		matches := make([]cli.GrammarProfileID, 0, 1)
		for candidate, profile := range set.Profiles {
			if profile.Value == state.value {
				matches = append(matches, candidate)
			}
		}
		slices.Sort(matches)
		if len(matches) == 0 {
			resolution = locatorResolution[cli.GrammarProfile]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectGrammarProfile, observationAbsent),
			}
			return resolution
		}
		if len(matches) > 1 {
			resolution = locatorResolution[cli.GrammarProfile]{
				status:  locatorResolutionAmbiguous,
				subject: label,
				detail: collectionCardinalityField{
					name:  "matches profile-value IDs",
					count: len(matches),
				},
			}
			return resolution
		}
		id = matches[0]
	default:
		resolution = locatorResolution[cli.GrammarProfile]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectGrammarProfile, observationInvalidLocator),
		}
		return resolution
	}
	profile, ok := set.Profiles[id]
	if !ok {
		resolution = locatorResolution[cli.GrammarProfile]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("grammar-profile order points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.GrammarProfile]{
		status:  locatorResolutionFound,
		value:   profile,
		subject: label,
	}
	return resolution
}

func resolveInspectionGrammarProfileID(state *grammarProfileLocatorState) (cli.GrammarProfileID, locatorResolutionStatus, evaluationField, evaluationField) {
	label := grammarProfileLocatorLabel(state)
	if state == nil || state.mode == grammarProfileLocatorInvalid {
		return 0, locatorResolutionInvalid, label, observed(semanticSubjectGrammarProfile, observationInvalidLocator)
	}
	if state.mode == grammarProfileLocatorID {
		return state.id, locatorResolutionFound, label, evaluationText("")
	}
	return 0, locatorResolutionNotProvable, label, evaluationText("inspection does not expose the grammar-profile lookup domain")
}

func expectedSchemaGrammarProfileID(state *grammarProfileLocatorState, schema *cli.Schema) (cli.GrammarProfileID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == grammarProfileLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectGrammarProfile, observationInvalidLocator)
	}
	if state.mode == grammarProfileLocatorID {
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectGrammarProfile, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	}
	resolution := resolveSchemaGrammarProfileLocator(state, schema)
	if resolution.status != locatorResolutionFound {
		return 0, resolution.status, resolution.detail
	}
	return resolution.value.ID, locatorResolutionFound, evaluationText("")
}

func expectedReferenceGrammarProfileID(state *grammarProfileLocatorState, reference cli.Reference) (cli.GrammarProfileID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == grammarProfileLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectGrammarProfile, observationInvalidLocator)
	}
	if state.mode == grammarProfileLocatorID {
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectGrammarProfile, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	}
	resolution := resolveReferenceGrammarProfileLocator(state, reference)
	if resolution.status != locatorResolutionFound {
		return 0, resolution.status, resolution.detail
	}
	return resolution.value.ID, locatorResolutionFound, evaluationText("")
}

func (argumentFormGrammarProfileExpectation) schemaExpectation() {}

func (argumentFormGrammarProfileExpectation) referenceExpectation() {}

func (expectation argumentFormGrammarProfileExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	form := resolveSchemaArgumentFormLocator(expectation.form, schema)
	if form.status != locatorResolutionFound {
		return outcomeForValueStatus(form.status, form.subject, evaluationText("grammar profile"), form.detail)
	}
	profileID, status, detail := expectedSchemaGrammarProfileID(expectation.profile, schema)
	if status == locatorResolutionNotProvable {
		return notProvableResult(form.subject, evaluationText("grammar profile"), detail)
	}
	if status != locatorResolutionFound {
		return relationLocatorFailureOutcome("argument form", "grammar profile", "expected grammar-profile locator", status)
	}
	if form.value.Profile == profileID {
		return provedResult()
	}
	return disprovedResult(form.subject, evaluationText("grammar profile matches expected identity"), evaluationText("grammar profile has different identity"))
}

func (expectation argumentFormGrammarProfileExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	form := resolveReferenceArgumentFormLocator(expectation.form, reference)
	if form.status != locatorResolutionFound {
		return outcomeForValueStatus(form.status, form.subject, evaluationText("grammar profile"), form.detail)
	}
	profileID, status, detail := expectedReferenceGrammarProfileID(expectation.profile, reference)
	if status == locatorResolutionNotProvable {
		return notProvableResult(form.subject, evaluationText("grammar profile"), detail)
	}
	if status != locatorResolutionFound {
		return relationLocatorFailureOutcome("argument form", "grammar profile", "expected grammar-profile locator", status)
	}
	if form.value.Profile == profileID {
		return provedResult()
	}
	return disprovedResult(form.subject, evaluationText("grammar profile matches expected identity"), evaluationText("grammar profile has different identity"))
}

func (commandGrammarProfilesExactlyExpectation) schemaExpectation() {}

func (commandGrammarProfilesExactlyExpectation) referenceExpectation() {}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func (expectation commandGrammarProfilesExactlyExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	command := resolveSchemaCommandLocator(expectation.command, schema)
	if command.status != locatorResolutionFound {
		return missingResult(evaluationText("command"), evaluationText("grammar profiles"), command.detail, false)
	}
	if command.value.Profiles == nil {
		if len(expectation.profiles) == 0 {
			return provedResult()
		}
		return missingResult(evaluationText("command"), evaluationText("grammar profiles"), evaluationText("grammar profiles absent"), false)
	}
	ids := make([]cli.GrammarProfileID, len(expectation.profiles))
	for index, profile := range expectation.profiles {
		id, status, detail := expectedSchemaGrammarProfileID(profile, schema)
		if status == locatorResolutionNotProvable {
			return notProvableResult(evaluationText("command"), evaluationText("grammar profiles"), detail)
		}
		if status != locatorResolutionFound {
			return relationLocatorFailureOutcome("command", "grammar profiles", "expected grammar-profile locator", status)
		}
		ids[index] = id
	}
	if slices.Equal(ids, command.value.Profiles.Cases) {
		return provedResult()
	}
	return disprovedResult(evaluationText("command"), evaluationText("grammar profiles match expected identity sequence"), evaluationText("grammar profiles have different identity sequence"))
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func (expectation commandGrammarProfilesExactlyExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	command := resolveReferenceCommandLocator(expectation.command, reference)
	if command.status != locatorResolutionFound {
		return missingResult(evaluationText("command"), evaluationText("grammar profiles"), command.detail, false)
	}
	if command.value.Profiles == nil {
		if len(expectation.profiles) == 0 {
			return provedResult()
		}
		return missingResult(evaluationText("command"), evaluationText("grammar profiles"), evaluationText("grammar profiles absent"), false)
	}
	ids := make([]cli.GrammarProfileID, len(expectation.profiles))
	for index, profile := range expectation.profiles {
		id, status, detail := expectedReferenceGrammarProfileID(profile, reference)
		if status == locatorResolutionNotProvable {
			return notProvableResult(evaluationText("command"), evaluationText("grammar profiles"), detail)
		}
		if status != locatorResolutionFound {
			return relationLocatorFailureOutcome("command", "grammar profiles", "expected grammar-profile locator", status)
		}
		ids[index] = id
	}
	if slices.Equal(ids, command.value.Profiles.Cases) {
		return provedResult()
	}
	return disprovedResult(evaluationText("command"), evaluationText("grammar profiles match expected identity sequence"), evaluationText("grammar profiles have different identity sequence"))
}

func (grammarProfileExistenceExpectationValue) schemaExpectation() {}

func (grammarProfileExistenceExpectationValue) referenceExpectation() {}

func (expectation grammarProfileExistenceExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	return declarationExistenceResult(resolveSchemaGrammarProfileLocator(expectation.locator, schema), expectation.expected)
}

func (expectation grammarProfileExistenceExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return declarationExistenceResult(resolveReferenceGrammarProfileLocator(expectation.locator, reference), expectation.expected)
}

func (grammarProfileReachabilityExpectationValue) schemaExpectation() {}

func (grammarProfileReachabilityExpectationValue) referenceExpectation() {}

func (expectation grammarProfileReachabilityExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaGrammarProfileLocator(expectation.locator, schema)
	actual := resolution.value.Reachability
	return declarationResolvedComparison(
		resolution.status,
		qualifySubject(resolution.subject, evaluationText("reachability")),
		resolution.detail,
		evaluationText(fmt.Sprintf("reachability %v", expectation.expected)),
		evaluationText(fmt.Sprintf("reachability %v", actual)),
		actual == expectation.expected,
	)
}

func (expectation grammarProfileReachabilityExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceGrammarProfileLocator(expectation.locator, reference)
	actual := resolution.value.Reachability
	return declarationResolvedComparison(
		resolution.status,
		qualifySubject(resolution.subject, evaluationText("reachability")),
		resolution.detail,
		evaluationText(fmt.Sprintf("reachability %v", expectation.expected)),
		evaluationText(fmt.Sprintf("reachability %v", actual)),
		actual == expectation.expected,
	)
}

func (grammarProfileOriginExpectationValue) schemaExpectation() {}

func (grammarProfileOriginExpectationValue) referenceExpectation() {}

func (expectation grammarProfileOriginExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaGrammarProfileLocator(expectation.locator, schema)
	actual := resolution.value.Origin
	return declarationResolvedComparison(
		resolution.status,
		qualifySubject(resolution.subject, evaluationText("origin")),
		resolution.detail,
		evaluationText(fmt.Sprintf("origin %v", expectation.expected)),
		evaluationText(fmt.Sprintf("origin %v", actual)),
		actual == expectation.expected,
	)
}

func (expectation grammarProfileOriginExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceGrammarProfileLocator(expectation.locator, reference)
	actual := resolution.value.Origin
	return declarationResolvedComparison(
		resolution.status,
		qualifySubject(resolution.subject, evaluationText("origin")),
		resolution.detail,
		evaluationText(fmt.Sprintf("origin %v", expectation.expected)),
		evaluationText(fmt.Sprintf("origin %v", actual)),
		actual == expectation.expected,
	)
}

func (grammarProfileSelectionExpectationValue) inspectionExpectation() {}

func (expectation grammarProfileSelectionExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	id, status, detail, label := resolveInspectionGrammarProfileID(expectation.locator)
	expected := evaluationText(selectionPresenceWord(expectation.selected))
	if status == locatorResolutionNotProvable {
		return notProvableResult(label, expected, detail)
	}
	if status != locatorResolutionFound {
		return disprovedResult(label, expected, detail)
	}
	found := false
	for _, commandID := range inspection.Selection.Command.Path {
		if command, ok := inspection.Commands[commandID]; ok && command.Profile == id {
			found = true
			break
		}
	}
	if found == expectation.selected {
		return provedResult()
	}
	return disprovedResult(label, expected, evaluationText(fmt.Sprintf("profile %v not in selected command path", id)))
}
