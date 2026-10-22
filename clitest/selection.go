package clitest

import (
	"fmt"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
)

// SelectionStepLocator identifies one causal command-selection step for clitest
// semantic expectation. Its zero value is invalid.
type SelectionStepLocator struct {
	_     [0]func()
	state *selectionStepLocatorState
}

// SelectionPathExactly checks the complete command selection path in
// authoritative order.
func SelectionPathExactly(locators ...CommandLocator) SelectionExpectation {
	return selectionPathExactlyExpectation(locators)
}

// SelectionStepsExactly checks the complete ordered causal selection-step
// sequence.
func SelectionStepsExactly(locators ...SelectionStepLocator) SelectionExpectation {
	return selectionStepsExactlyExpectation(locators)
}

// SelectionStep returns a locator that matches a command-selection entry step for the located
// command entry.
func (locator CommandEntryLocator) SelectionStep() SelectionStepLocator {
	return commandEntrySelectionStepLocator(locator)
}

// Selected returns an expectation that checks the located command entry participates in the
// authoritative selection state.
func (locator CommandEntryLocator) Selected() SelectionExpectation {
	return commandEntrySelectionExpectation(locator, true)
}

// NotSelected returns an expectation that checks the located command entry does not participate in
// the authoritative selection state.
func (locator CommandEntryLocator) NotSelected() SelectionExpectation {
	return commandEntrySelectionExpectation(locator, false)
}

// Selected returns an expectation that checks the located command participates in the
// authoritative selection state.
func (l CommandLocator) Selected() SelectionExpectation {
	return commandSelectionExpectation(l, true)
}

// NotSelected returns an expectation that checks the located command does not participate in the
// authoritative selection state.
func (l CommandLocator) NotSelected() SelectionExpectation {
	return commandSelectionExpectation(l, false)
}

// DefaultSelectionOutcome returns an expectation that checks the located command has the expected
// default-command selection outcome.
func (l CommandLocator) DefaultSelectionOutcome(status cli.DefaultCommandSelectionOutcome) InspectionExpectation {
	return commandDefaultSelectionOutcomeExpectation(l, status)
}

// DefaultCandidatesExactly returns an expectation that checks the located command's
// default-command candidates are exactly the supplied locators in authoritative public order.
// Exactness is not inferred from an incomplete observation.
func (l CommandLocator) DefaultCandidatesExactly(defaults ...CommandDefaultLocator) InspectionExpectation {
	return commandDefaultSelectionSetExpectation(l, commandSelectionDefaultCandidates, defaults...)
}

// ApplicableDefaultsExactly returns an expectation that checks the located command's applicable
// defaults are exactly the supplied locators in authoritative public order. Exactness is not
// inferred from an incomplete observation.
func (l CommandLocator) ApplicableDefaultsExactly(defaults ...CommandDefaultLocator) InspectionExpectation {
	return commandDefaultSelectionSetExpectation(l, commandSelectionDefaultApplicable, defaults...)
}

// BlockedDefaultsExactly returns an expectation that checks the located command's blocked defaults
// are exactly the supplied locators in authoritative public order. Exactness is not inferred from
// an incomplete observation.
func (l CommandLocator) BlockedDefaultsExactly(defaults ...CommandDefaultLocator) InspectionExpectation {
	return commandDefaultSelectionSetExpectation(l, commandSelectionDefaultBlocked, defaults...)
}

// ArgumentFormSelectionOutcome returns an expectation that checks the located command has the expected
// argument-form selection outcome.
func (l CommandLocator) ArgumentFormSelectionOutcome(status cli.ArgumentFormSelectionOutcome) InspectionExpectation {
	return commandArgumentFormSelectionOutcomeExpectation(l, status)
}

// MatchingArgumentFormsExactly returns an expectation that checks the located command's matching
// argument forms are exactly the supplied locators in authoritative public order. Exactness is not
// inferred from an incomplete observation.
func (l CommandLocator) MatchingArgumentFormsExactly(forms ...ArgumentFormLocator) InspectionExpectation {
	return commandArgumentFormSelectionSetExpectation(l, commandSelectionArgumentFormMatches, forms...)
}

// BlockedArgumentFormsExactly returns an expectation that checks the located command's blocked
// argument forms are exactly the supplied locators in authoritative public order. Exactness is not
// inferred from an incomplete observation.
func (l CommandLocator) BlockedArgumentFormsExactly(forms ...ArgumentFormLocator) InspectionExpectation {
	return commandArgumentFormSelectionSetExpectation(l, commandSelectionArgumentFormBlocked, forms...)
}

// FailedArgumentFormsExactly returns an expectation that checks the located command's failed
// argument forms are exactly the supplied locators in authoritative public order. Exactness is not
// inferred from an incomplete observation.
func (l CommandLocator) FailedArgumentFormsExactly(forms ...ArgumentFormLocator) InspectionExpectation {
	return commandArgumentFormSelectionSetExpectation(l, commandSelectionArgumentFormFailed, forms...)
}

// Selected returns an expectation that checks the located invocation action participates in the
// authoritative selection state.
func (locator InvocationActionLocator) Selected() InspectionExpectation {
	return actionSelectionExpectation(locator, true)
}

// NotSelected returns an expectation that checks the located invocation action does not
// participate in the authoritative selection state.
func (locator InvocationActionLocator) NotSelected() InspectionExpectation {
	return actionSelectionExpectation(locator, false)
}

// Trigger returns an expectation that checks the selected invocation action has the expected
// public trigger classification.
func (locator InvocationActionLocator) Trigger(trigger cli.InvocationActionTrigger) InspectionExpectation {
	return actionTriggerExpectation(locator, trigger)
}

// SelectedEntry returns an expectation that checks the selected invocation action is attributed to
// the exact action entry identified by entry.
func (locator InvocationActionLocator) SelectedEntry(entry ActionEntryLocator) InspectionExpectation {
	return actionEntryExpectation(locator, entry)
}

// TriggeredBy returns an expectation that checks the selected invocation action is attributed to
// the exact input origin identified by origin.
func (locator InvocationActionLocator) TriggeredBy(origin InputOriginLocator) InspectionExpectation {
	return actionOriginExpectation(locator, origin)
}

// Selected returns an expectation that checks the located action entry participates in the
// authoritative selection state.
func (l ActionEntryLocator) Selected() InspectionExpectation {
	return actionEntrySelectedExpectation(l, true)
}

// NotSelected returns an expectation that checks the located action entry does not participate in
// the authoritative selection state.
func (l ActionEntryLocator) NotSelected() InspectionExpectation {
	return actionEntrySelectedExpectation(l, false)
}

// Selected returns an expectation that checks the located grammar profile participates in the
// authoritative selection state.
func (l GrammarProfileLocator) Selected() InspectionExpectation {
	return grammarProfileSelectionExpectation(l, true)
}

// NotSelected returns an expectation that checks the located grammar profile does not participate
// in the authoritative selection state.
func (l GrammarProfileLocator) NotSelected() InspectionExpectation {
	return grammarProfileSelectionExpectation(l, false)
}

// Outcome returns an expectation that checks the located argument form has the expected
// invocation-time ArgumentFormOutcome.
func (l ArgumentFormLocator) Outcome(status cli.ArgumentFormOutcome) InspectionExpectation {
	return argumentFormOutcomeExpectation(l, status)
}

// Selected returns an expectation that checks the located argument form participates in the
// authoritative selection state.
func (l ArgumentFormLocator) Selected() InspectionExpectation {
	return argumentFormSelectedExpectation(l, true)
}

// NotSelected returns an expectation that checks the located argument form does not participate in
// the authoritative selection state.
func (l ArgumentFormLocator) NotSelected() InspectionExpectation {
	return argumentFormSelectedExpectation(l, false)
}

// From returns a copy of the selection-step locator constrained to the exact public input-origin
// relationship identified by origin.
func (locator SelectionStepLocator) From(origin InputOriginLocator) SelectionStepLocator {
	return selectionStepFrom(locator, origin)
}

// Selects returns a copy of the selection-step locator constrained to the exact selected-command
// sequence identified by commands.
func (locator SelectionStepLocator) Selects(commands ...CommandLocator) SelectionStepLocator {
	return selectionStepSelects(locator, commands)
}

// SelectionStep follows the public relationship from the command default to its causal
// selection step.
func (locator CommandDefaultLocator) SelectionStep() SelectionStepLocator {
	return commandDefaultSelectionStepLocator(locator)
}

// Selected returns an expectation that checks this command default was the effective
// default-selection cause.
func (locator CommandDefaultLocator) Selected() SelectionExpectation {
	return commandDefaultSelectedExpectation(locator, true)
}

// NotSelected returns an expectation that checks this command default was not the effective
// default-selection cause.
func (locator CommandDefaultLocator) NotSelected() SelectionExpectation {
	return commandDefaultSelectedExpectation(locator, false)
}

type commandSelectionDetailKind uint8

const (
	commandSelectionDetailInvalid commandSelectionDetailKind = iota
	commandSelectionDefaultOutcome
	commandSelectionDefaultCandidates
	commandSelectionDefaultApplicable
	commandSelectionDefaultBlocked
	commandSelectionArgumentFormOutcome
	commandSelectionArgumentFormMatches
	commandSelectionArgumentFormBlocked
	commandSelectionArgumentFormFailed
)

type commandSelectionDetailExpectation struct {
	command        *commandLocatorState
	kind           commandSelectionDetailKind
	defaultOutcome cli.DefaultCommandSelectionOutcome
	formOutcome    cli.ArgumentFormSelectionOutcome
	defaults       []*commandDefaultLocatorState
	forms          []*argumentFormLocatorState
}

type selectionStepLocatorKind uint8

const (
	selectionStepLocatorInvalid selectionStepLocatorKind = iota
	selectionStepLocatorEntry
	selectionStepLocatorDefault
)

type selectionStepLocatorState struct {
	kind selectionStepLocatorKind

	entry          *commandEntryLocatorState
	commandDefault *commandDefaultLocatorState
	origin         *inputOriginLocatorState
	commands       []*commandLocatorState
}

type commandSelectionExpectationValue struct {
	locator  *commandLocatorState
	selected bool
}

type commandEntrySelectionExpectationValue struct {
	locator  *commandEntryLocatorState
	selected bool
}

type selectionPathExpectationValue struct{ commands []*commandLocatorState }

type selectionStepsExpectationValue struct{ steps []*selectionStepLocatorState }

type actionSelectionExpectationValue struct {
	locator  *actionLocatorState
	selected bool
}

type actionTriggerExpectationValue struct {
	locator *actionLocatorState
	trigger cli.InvocationActionTrigger
}

type actionEntryExpectationValue struct {
	action *actionLocatorState
	entry  *actionEntryLocatorState
}

type actionOriginExpectationValue struct {
	action *actionLocatorState
	origin *inputOriginLocatorState
}

type actionEntrySelectedExpectationValue struct {
	locator  *actionEntryLocatorState
	selected bool
}

type grammarProfileSelectionExpectationValue struct {
	locator  *grammarProfileLocatorState
	selected bool
}

type argumentFormOutcomeExpectationValue struct {
	locator *argumentFormLocatorState
	outcome cli.ArgumentFormOutcome
}

type argumentFormSelectedExpectationValue struct {
	locator  *argumentFormLocatorState
	selected bool
}

type commandDefaultSelectedExpectationValue struct {
	locator  *commandDefaultLocatorState
	selected bool
}

type commandDefaultConditionExpectationValue struct {
	locator *commandDefaultLocatorState
	outcome cli.EvaluationOutcome
}

func commandDefaultSelectionOutcomeExpectation(locator CommandLocator, status cli.DefaultCommandSelectionOutcome) InspectionExpectation {
	expectation := commandSelectionDetailExpectation{
		command:        locator.state,
		kind:           commandSelectionDefaultOutcome,
		defaultOutcome: status,
	}
	return expectation
}

func commandDefaultSelectionSetExpectation(locator CommandLocator, kind commandSelectionDetailKind, expected ...CommandDefaultLocator) InspectionExpectation {
	states := make([]*commandDefaultLocatorState, len(expected))
	for index, value := range expected {
		states[index] = value.state
	}
	expectation := commandSelectionDetailExpectation{
		command:  locator.state,
		kind:     kind,
		defaults: states,
	}
	return expectation
}

func expectedInspectionCommandDefaultIDs(states []*commandDefaultLocatorState, inspection cli.Inspection, command cli.CommandInspection) ([]cli.CommandDefaultID, locatorResolutionStatus, evaluationField) {
	ids := make([]cli.CommandDefaultID, len(states))
	for index, state := range states {
		id, status, detail := expectedInspectionCommandDefaultID(state, inspection)
		if status != locatorResolutionFound || !slices.Contains(command.CommandDefaults, id) {
			if status == locatorResolutionFound {
				status = locatorResolutionInvalid
				detail = evaluationText("invalid or foreign command-default locator")
			}
			return nil, status, detail
		}
		ids[index] = id
	}
	return ids, locatorResolutionFound, evaluationText("")
}

func commandArgumentFormSelectionOutcomeExpectation(locator CommandLocator, status cli.ArgumentFormSelectionOutcome) InspectionExpectation {
	expectation := commandSelectionDetailExpectation{
		command:     locator.state,
		kind:        commandSelectionArgumentFormOutcome,
		formOutcome: status,
	}
	return expectation
}

func commandArgumentFormSelectionSetExpectation(locator CommandLocator, kind commandSelectionDetailKind, expected ...ArgumentFormLocator) InspectionExpectation {
	states := make([]*argumentFormLocatorState, len(expected))
	for index, value := range expected {
		states[index] = value.state
	}
	expectation := commandSelectionDetailExpectation{
		command: locator.state,
		kind:    kind,
		forms:   states,
	}
	return expectation
}

func expectedInspectionArgumentFormID(state *argumentFormLocatorState, inspection cli.Inspection) (cli.ArgumentFormID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == argumentFormLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectArgumentForm, observationInvalidOrForeignLocator)
	}
	if state.mode == argumentFormLocatorID {
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectArgumentForm, observationInvalidOrForeignLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	}
	if state.mode != argumentFormLocatorCommandIndex || state.index < 0 {
		return 0, locatorResolutionInvalid, observed(semanticSubjectArgumentForm, observationInvalidOrForeignLocator)
	}
	command := resolveInspectionCommandLocator(state.command, inspection)
	if command.status != locatorResolutionFound {
		return 0, command.status, command.detail
	}
	if state.index < len(command.value.ArgumentForms) {
		return command.value.ArgumentForms[state.index], locatorResolutionFound, evaluationText("")
	}
	if inspection.Truncated || command.value.NamespaceCompleteness != cli.CommandNamespaceComplete {
		return 0, locatorResolutionNotProvable, evaluationText("argument-form collection is not closed")
	}
	return 0, locatorResolutionAbsent, observed(semanticSubjectArgumentForm, observationAbsent)
}

func expectedInspectionArgumentFormIDs(states []*argumentFormLocatorState, inspection cli.Inspection, command cli.CommandInspection) ([]cli.ArgumentFormID, locatorResolutionStatus, evaluationField) {
	ids := make([]cli.ArgumentFormID, len(states))
	for index, state := range states {
		id, status, detail := expectedInspectionArgumentFormID(state, inspection)
		if status != locatorResolutionFound || !slices.Contains(command.ArgumentForms, id) {
			if status == locatorResolutionFound {
				status = locatorResolutionInvalid
				detail = observed(semanticSubjectArgumentForm, observationInvalidOrForeignLocator)
			}
			return nil, status, detail
		}
		ids[index] = id
	}
	return ids, locatorResolutionFound, evaluationText("")
}

func commandSelectionExpectation(locator CommandLocator, selected bool) SelectionExpectation {
	value := commandSelectionExpectationValue{
		locator:  locator.state,
		selected: selected,
	}
	return value
}

func evaluateCommandSelection(state *commandLocatorState, selected bool, selection cli.CommandSelection, commands map[cli.CommandID]cli.CommandInspection) evaluationResult {
	id, status, actual := resolveInspectionCommandID(state, commands)
	label := evaluationText("command selection")
	expected := evaluationText(selectionPresenceWord(selected))
	if status == locatorResolutionNotProvable {
		return notProvableResult(label, expected, actual)
	}
	if status != locatorResolutionFound {
		return disprovedResult(label, expected, actual)
	}
	found := slices.Contains(selection.Path, id)
	if found == selected {
		return provedResult()
	}
	if selected {
		return disprovedResult(label, evaluationText("selected"), evaluationText(fmt.Sprintf("command %v is not in selected path", id)))
	}
	return disprovedResult(label, evaluationText("not selected"), evaluationText(fmt.Sprintf("command %v is in selected path", id)))
}

func resolveSelectionCommand(state *commandLocatorState, commands map[cli.CommandID]cli.CommandInspection) (cli.CommandID, locatorResolutionStatus, evaluationField) {
	return resolveInspectionCommandID(state, commands)
}

func selectionPresenceWord(selected bool) string {
	if selected {
		return "selected"
	}
	return "not selected"
}

func commandEntrySelectionExpectation(locator CommandEntryLocator, selected bool) SelectionExpectation {
	value := commandEntrySelectionExpectationValue{
		locator:  locator.state,
		selected: selected,
	}
	return value
}

func evaluateCommandEntrySelection(id cli.CommandEntryID, selected bool, selection cli.CommandSelection) evaluationResult {
	if id == 0 {
		return disprovedResult(evaluationText("command-entry selection"), evaluationText(selectionPresenceWord(selected)), observed(semanticSubjectCommandEntry, observationInvalidLocator))
	}
	found := false
	for _, step := range selection.Steps {
		if step.Kind == cli.CommandSelectionStepEntry && step.Entry == id {
			found = true
			break
		}
	}
	if found == selected {
		return provedResult()
	}
	if selected {
		return disprovedResult(evaluationText("command-entry selection"), evaluationText("selected"), evaluationText("entry is not in selection steps"))
	}
	return disprovedResult(evaluationText("command-entry selection"), evaluationText("not selected"), evaluationText("entry is in selection steps"))
}

func selectionPathExactlyExpectation(locators []CommandLocator) SelectionExpectation {
	states := make([]*commandLocatorState, len(locators))
	for index, locator := range locators {
		states[index] = locator.state
	}
	value := selectionPathExpectationValue{
		commands: states,
	}
	return value
}

func evaluateSelectionPath(states []*commandLocatorState, selection cli.CommandSelection, commands map[cli.CommandID]cli.CommandInspection) evaluationResult {
	if len(selection.Path) != len(states) {
		return disprovedResult(
			evaluationText("selection path"),
			evaluationText(fmt.Sprintf("%d commands", len(states))),
			evaluationText(fmt.Sprintf("%d commands", len(selection.Path))),
		)
	}
	for i, state := range states {
		id, status, actual := resolveSelectionCommand(state, commands)
		if status == locatorResolutionNotProvable {
			return notProvableResult(evaluationText("selection path"), evaluationText(fmt.Sprintf("command at position %d", i)), actual)
		}
		if status != locatorResolutionFound {
			return disprovedResult(evaluationText("selection path"), evaluationText(fmt.Sprintf("command at position %d", i)), actual)
		}
		if selection.Path[i] != id {
			return disprovedResult(
				evaluationText("selection path"),
				evaluationText(fmt.Sprintf("command %v at position %d", id, i)),
				evaluationText(fmt.Sprintf("command %v", selection.Path[i])),
			)
		}
	}
	return provedResult()
}

func selectionStepsExactlyExpectation(locators []SelectionStepLocator) SelectionExpectation {
	states := make([]*selectionStepLocatorState, len(locators))
	for index, locator := range locators {
		states[index] = locator.state
	}
	value := selectionStepsExpectationValue{
		steps: states,
	}
	return value
}

func evaluateSelectionSteps(states []*selectionStepLocatorState, selection cli.CommandSelection, commands map[cli.CommandID]cli.CommandInspection, entries map[cli.CommandEntryID]cli.CommandEntryInspection, defaults map[cli.CommandDefaultID]cli.CommandDefaultInspection, origins map[cli.InputOriginID]cli.InputOrigin) evaluationResult {
	if len(selection.Steps) != len(states) {
		return disprovedResult(
			evaluationText("selection steps"),
			evaluationText(fmt.Sprintf("%d steps", len(states))),
			evaluationText(fmt.Sprintf("%d steps", len(selection.Steps))),
		)
	}
	for i, state := range states {
		outcome := matchSelectionStep(state, selection.Steps[i], commands, entries, defaults, origins)
		if outcome.status != evaluationProved {
			outcome.subject = evaluationText(fmt.Sprintf("selection step %d", i))
			return outcome
		}
	}
	return provedResult()
}

func matchSelectionStep(state *selectionStepLocatorState, step cli.CommandSelectionStep, commands map[cli.CommandID]cli.CommandInspection, entries map[cli.CommandEntryID]cli.CommandEntryInspection, defaults map[cli.CommandDefaultID]cli.CommandDefaultInspection, origins map[cli.InputOriginID]cli.InputOrigin) evaluationResult {
	if state == nil || state.kind == selectionStepLocatorInvalid {
		return disprovedResult(evaluationText("selection step"), evaluationText("valid locator"), evaluationText("invalid selection-step locator"))
	}

	switch state.kind {
	case selectionStepLocatorEntry:
		if state.entry == nil {
			return disprovedResult(evaluationText("selection step"), evaluationText("configured entry"), observed(semanticSubjectCommandEntry, observationInvalidLocator))
		}
		var expected cli.CommandEntryID
		if state.entry.mode == commandEntryLocatorID {
			expected = state.entry.id
		} else {
			if commands == nil || entries == nil {
				return notProvableResult(evaluationText("selection step"), evaluationText("configured entry"), evaluationText("operation does not expose the command-entry lookup domain"))
			}
			inspection := cli.Inspection{
				Commands:       commands,
				CommandEntries: entries,
			}
			resolution := resolveInspectionCommandEntryLocator(state.entry, inspection)
			if resolution.status != locatorResolutionFound {
				return resolutionOutcome(resolution.status, evaluationText("selection step"), evaluationText("configured entry"), resolution.detail)
			}
			expected = resolution.value.ID
		}
		if step.Kind != cli.CommandSelectionStepEntry || step.Entry != expected {
			return disprovedResult(evaluationText("selection step"), evaluationText(fmt.Sprintf("entry %v", expected)), evaluationText(fmt.Sprintf("kind=%v entry=%v", step.Kind, step.Entry)))
		}

	case selectionStepLocatorDefault:
		if state.commandDefault == nil {
			return disprovedResult(evaluationText("selection step"), evaluationText("configured default"), evaluationText("invalid command-default locator"))
		}
		expected := state.commandDefault.id
		if state.commandDefault.mode != commandDefaultLocatorID {
			if commands == nil || defaults == nil {
				return notProvableResult(evaluationText("selection step"), evaluationText("configured default"), evaluationText("operation does not expose the command-default lookup domain"))
			}
			inspection := cli.Inspection{
				Commands:        commands,
				CommandDefaults: defaults,
			}
			resolution := resolveInspectionCommandDefaultLocator(state.commandDefault, inspection)
			if resolution.status != locatorResolutionFound {
				return resolutionOutcome(resolution.status, evaluationText("selection step"), evaluationText("configured default"), resolution.detail)
			}
			expected = resolution.value.ID
		}
		if step.Kind != cli.CommandSelectionStepDefault || step.Default != expected {
			return disprovedResult(evaluationText("selection step"), evaluationText(fmt.Sprintf("default %v", expected)), evaluationText(fmt.Sprintf("kind=%v default=%v", step.Kind, step.Default)))
		}

	default:
		return disprovedResult(evaluationText("selection step"), evaluationText("entry/default step"), evaluationText("invalid selection-step locator"))
	}

	if state.origin != nil {
		id, status, actual := resolveSelectionOriginID(state.origin, origins)
		if status == locatorResolutionNotProvable {
			return notProvableResult(evaluationText("selection step"), evaluationText("configured origin"), actual)
		}
		if status != locatorResolutionFound {
			return disprovedResult(evaluationText("selection step"), evaluationText("configured origin"), actual)
		}
		if step.Input != id {
			return disprovedResult(evaluationText("selection step"), evaluationText("configured origin identity"), evaluationText("different selected origin identity"))
		}
	}

	if state.commands != nil {
		if len(step.Commands) != len(state.commands) {
			return disprovedResult(evaluationText("selection step"), evaluationText(fmt.Sprintf("%d selected commands", len(state.commands))), evaluationText(fmt.Sprintf("%d selected commands", len(step.Commands))))
		}
		for index, commandState := range state.commands {
			id, status, actual := resolveInspectionCommandID(commandState, commands)
			if status == locatorResolutionNotProvable {
				return notProvableResult(evaluationText("selection step"), evaluationText(fmt.Sprintf("command at position %d", index)), actual)
			}
			if status != locatorResolutionFound {
				return disprovedResult(evaluationText("selection step"), evaluationText(fmt.Sprintf("command at position %d", index)), actual)
			}
			if step.Commands[index] != id {
				return disprovedResult(evaluationText("selection step"), evaluationText(fmt.Sprintf("command %v at position %d", id, index)), evaluationText(fmt.Sprintf("command %v", step.Commands[index])))
			}
		}
	}

	return provedResult()
}

func resolveSelectionOriginID(state *inputOriginLocatorState, origins map[cli.InputOriginID]cli.InputOrigin) (cli.InputOriginID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == inputOriginLocatorInvalid {
		return cli.InputOriginID{}, locatorResolutionInvalid, evaluationText("invalid origin locator")
	}
	switch state.mode {
	case inputOriginLocatorID:
		return state.id, locatorResolutionFound, evaluationText("")
	case inputOriginLocatorParentIndex:
		if state.parent == nil || state.index < 0 {
			return cli.InputOriginID{}, locatorResolutionInvalid, evaluationText("invalid origin locator")
		}
		parent, status, actual := resolveSelectionOriginID(state.parent, origins)
		if status != locatorResolutionFound {
			return cli.InputOriginID{}, status, actual
		}
		if origins == nil {
			return cli.InputOriginID{}, locatorResolutionNotProvable, evaluationText("operation does not expose origin relationships")
		}
		origin, ok := origins[parent]
		if !ok {
			return cli.InputOriginID{}, locatorResolutionAbsent, evaluationText("parent origin is absent")
		}
		if state.index >= len(origin.Parents) {
			return cli.InputOriginID{}, locatorResolutionAbsent, evaluationText("origin parent relationship is absent")
		}
		return origin.Parents[state.index], locatorResolutionFound, evaluationText("")
	case inputOriginLocatorToken:
		return cli.InputOriginID{}, locatorResolutionNotProvable, evaluationText("selection observation does not expose the token lookup domain")
	default:
		return cli.InputOriginID{}, locatorResolutionInvalid, evaluationText("invalid origin locator")
	}
}

func actionLocator(action cli.InvocationAction) InvocationActionLocator {
	state := &actionLocatorState{
		valid:  true,
		action: action,
	}
	locator := InvocationActionLocator{
		state: state,
	}
	return locator
}

func grammarProfileIDLocator(id cli.GrammarProfileID) GrammarProfileLocator {
	state := &grammarProfileLocatorState{
		mode: grammarProfileLocatorID,
		id:   id,
	}
	locator := GrammarProfileLocator{
		state: state,
	}
	return locator
}

func actionSelectionExpectation(locator InvocationActionLocator, selected bool) InspectionExpectation {
	value := actionSelectionExpectationValue{
		locator:  locator.state,
		selected: selected,
	}
	return value
}

func actionTriggerExpectation(locator InvocationActionLocator, trigger cli.InvocationActionTrigger) InspectionExpectation {
	value := actionTriggerExpectationValue{
		locator: locator.state,
		trigger: trigger,
	}
	return value
}

func actionEntryExpectation(locator InvocationActionLocator, entry ActionEntryLocator) InspectionExpectation {
	value := actionEntryExpectationValue{
		action: locator.state,
		entry:  entry.state,
	}
	return value
}

func actionOriginExpectation(locator InvocationActionLocator, origin InputOriginLocator) InspectionExpectation {
	value := actionOriginExpectationValue{
		action: locator.state,
		origin: origin.state,
	}
	return value
}

func actionEntrySelectedExpectation(locator ActionEntryLocator, selected bool) InspectionExpectation {
	value := actionEntrySelectedExpectationValue{
		locator:  locator.state,
		selected: selected,
	}
	return value
}

func grammarProfileSelectionExpectation(locator GrammarProfileLocator, selected bool) InspectionExpectation {
	value := grammarProfileSelectionExpectationValue{
		locator:  locator.state,
		selected: selected,
	}
	return value
}

func argumentFormOutcomeExpectation(locator ArgumentFormLocator, outcome cli.ArgumentFormOutcome) InspectionExpectation {
	value := argumentFormOutcomeExpectationValue{
		locator: locator.state,
		outcome: outcome,
	}
	return value
}

func argumentFormSelectedExpectation(locator ArgumentFormLocator, selected bool) InspectionExpectation {
	value := argumentFormSelectedExpectationValue{
		locator:  locator.state,
		selected: selected,
	}
	return value
}

func commandDefaultSelectedExpectation(locator CommandDefaultLocator, selected bool) SelectionExpectation {
	value := commandDefaultSelectedExpectationValue{
		locator:  locator.state,
		selected: selected,
	}
	return value
}

func evaluateCommandDefaultSelection(state *commandDefaultLocatorState, selected bool, selection cli.CommandSelection) evaluationResult {
	if state == nil || state.mode != commandDefaultLocatorID {
		return disprovedResult(evaluationText("command default"), evaluationText(selectionPresenceWord(selected)), evaluationText("invalid command-default locator"))
	}
	found := false
	for _, step := range selection.Steps {
		if step.Kind == cli.CommandSelectionStepDefault && step.Default == state.id {
			found = true
			break
		}
	}
	if found == selected {
		return provedResult()
	}
	return disprovedResult(
		evaluationText("command default"),
		evaluationText(selectionPresenceWord(selected)),
		evaluationText("selection steps do not match expectation"),
	)
}

func commandDefaultConditionOutcomeExpectation(locator CommandDefaultLocator, outcome cli.EvaluationOutcome) InspectionExpectation {
	value := commandDefaultConditionExpectationValue{
		locator: locator.state,
		outcome: outcome,
	}
	return value
}

func (commandSelectionDetailExpectation) inspectionExpectation() {}

func (expectation commandSelectionDetailExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	command := resolveInspectionCommandLocator(expectation.command, inspection)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText(expectation.description()), command.detail)
	}

	switch expectation.kind {
	case commandSelectionDefaultOutcome:
		actual := command.value.DefaultSelection.Outcome
		if actual == expectation.defaultOutcome {
			return provedResult()
		}
		return disprovedResult(command.subject, evaluationText(expectation.description()), evaluationText(fmt.Sprintf("default selection outcome %v", actual)))
	case commandSelectionDefaultCandidates, commandSelectionDefaultApplicable, commandSelectionDefaultBlocked:
		expected, status, detail := expectedInspectionCommandDefaultIDs(expectation.defaults, inspection, command.value)
		if status != locatorResolutionFound {
			return outcomeForValueStatus(status, command.subject, evaluationText(expectation.description()), detail)
		}
		actual := expectation.defaultSelectionIDs(command.value.DefaultSelection)
		if slices.Equal(actual, expected) {
			return provedResult()
		}
		return orderedCollectionMismatch(command.subject, expectation.defaultSetName()+" defaults", expected, actual)
	case commandSelectionArgumentFormOutcome:
		actual := command.value.ArgumentFormSelection.Outcome
		if actual == expectation.formOutcome {
			return provedResult()
		}
		return disprovedResult(command.subject, evaluationText(expectation.description()), evaluationText(fmt.Sprintf("argument-form selection outcome %v", actual)))
	case commandSelectionArgumentFormMatches, commandSelectionArgumentFormBlocked, commandSelectionArgumentFormFailed:
		expected, status, detail := expectedInspectionArgumentFormIDs(expectation.forms, inspection, command.value)
		if status != locatorResolutionFound {
			return outcomeForValueStatus(status, command.subject, evaluationText(expectation.description()), detail)
		}
		actual := expectation.argumentFormSelectionIDs(command.value.ArgumentFormSelection)
		if slices.Equal(actual, expected) {
			return provedResult()
		}
		return orderedCollectionMismatch(command.subject, expectation.argumentFormSetName()+" argument forms", expected, actual)
	default:
		return disprovedResult(command.subject, evaluationText("valid command selection expectation"), evaluationText("invalid command selection expectation"))
	}
}

func (expectation commandSelectionDetailExpectation) description() string {
	switch expectation.kind {
	case commandSelectionDefaultOutcome:
		return fmt.Sprintf("default selection outcome %v", expectation.defaultOutcome)
	case commandSelectionDefaultCandidates, commandSelectionDefaultApplicable, commandSelectionDefaultBlocked:
		return expectation.defaultSetName() + " defaults exactly"
	case commandSelectionArgumentFormOutcome:
		return fmt.Sprintf("argument-form selection outcome %v", expectation.formOutcome)
	case commandSelectionArgumentFormMatches, commandSelectionArgumentFormBlocked, commandSelectionArgumentFormFailed:
		return expectation.argumentFormSetName() + " argument forms exactly"
	default:
		return "valid command selection expectation"
	}
}

func (expectation commandSelectionDetailExpectation) defaultSetName() string {
	switch expectation.kind {
	case commandSelectionDefaultCandidates:
		return semanticNameCandidate
	case commandSelectionDefaultApplicable:
		return "applicable"
	case commandSelectionDefaultBlocked:
		return semanticLabelBlocked
	default:
		return "invalid"
	}
}

func (expectation commandSelectionDetailExpectation) defaultSelectionIDs(selection cli.DefaultCommandSelection) []cli.CommandDefaultID {
	switch expectation.kind {
	case commandSelectionDefaultCandidates:
		return selection.Candidates
	case commandSelectionDefaultApplicable:
		return selection.Applicable
	case commandSelectionDefaultBlocked:
		return selection.Blocked
	default:
		return nil
	}
}

func (expectation commandSelectionDetailExpectation) argumentFormSetName() string {
	switch expectation.kind {
	case commandSelectionArgumentFormMatches:
		return "matching"
	case commandSelectionArgumentFormBlocked:
		return semanticLabelBlocked
	case commandSelectionArgumentFormFailed:
		return "failed"
	default:
		return "invalid"
	}
}

func (expectation commandSelectionDetailExpectation) argumentFormSelectionIDs(selection cli.ArgumentFormSelection) []cli.ArgumentFormID {
	switch expectation.kind {
	case commandSelectionArgumentFormMatches:
		return selection.Matches
	case commandSelectionArgumentFormBlocked:
		return selection.Blocked
	case commandSelectionArgumentFormFailed:
		return selection.Failed
	default:
		return nil
	}
}

func (commandSelectionExpectationValue) inspectionExpectation() {}

func (commandSelectionExpectationValue) completionExpectation() {}

func (expectation commandSelectionExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return evaluateCommandSelection(expectation.locator, expectation.selected, inspection.Selection.Command, inspection.Commands)
}

func (expectation commandSelectionExpectationValue) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	return evaluateCommandSelection(expectation.locator, expectation.selected, result.Selection, nil)
}

func (selectionPathExpectationValue) inspectionExpectation() {}

func (selectionPathExpectationValue) completionExpectation() {}

func (expectation selectionPathExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return evaluateSelectionPath(expectation.commands, inspection.Selection.Command, inspection.Commands)
}

func (expectation selectionPathExpectationValue) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	return evaluateSelectionPath(expectation.commands, result.Selection, nil)
}

func (selectionStepsExpectationValue) inspectionExpectation() {}

func (selectionStepsExpectationValue) completionExpectation() {}

func (expectation selectionStepsExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return evaluateSelectionSteps(
		expectation.steps,
		inspection.Selection.Command,
		inspection.Commands,
		inspection.CommandEntries,
		inspection.CommandDefaults,
		inspection.Origins,
	)
}

func (expectation selectionStepsExpectationValue) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	return evaluateSelectionSteps(expectation.steps, result.Selection, nil, nil, nil, result.Origins)
}

func (actionSelectionExpectationValue) inspectionExpectation() {}

func (expectation actionSelectionExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	if expectation.locator == nil || !expectation.locator.valid {
		return disprovedResult(evaluationText("invocation action"), evaluationText(selectionPresenceWord(expectation.selected)), evaluationText("invalid action locator"))
	}
	actual := inspection.Selection.Action.Selected == expectation.locator.action
	if actual == expectation.selected {
		return provedResult()
	}
	return disprovedResult(evaluationText("invocation action"), evaluationText(selectionPresenceWord(expectation.selected)), evaluationText(fmt.Sprintf("selected action %v", inspection.Selection.Action.Selected)))
}

func (actionTriggerExpectationValue) inspectionExpectation() {}

func (expectation actionTriggerExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	if expectation.locator == nil || !expectation.locator.valid {
		return disprovedResult(evaluationText("invocation action"), evaluationText(fmt.Sprintf("trigger %v", expectation.trigger)), evaluationText("invalid action locator"))
	}
	selection := inspection.Selection.Action
	if selection.Selected != expectation.locator.action {
		return disprovedResult(evaluationText("invocation action"), evaluationText(fmt.Sprintf("trigger %v", expectation.trigger)), evaluationText(fmt.Sprintf("selected action %v", selection.Selected)))
	}
	if selection.Trigger == expectation.trigger {
		return provedResult()
	}
	return disprovedResult(evaluationText("invocation action"), evaluationText(fmt.Sprintf("trigger %v", expectation.trigger)), evaluationText(fmt.Sprintf("trigger %v", selection.Trigger)))
}

func (actionOriginExpectationValue) inspectionExpectation() {}

func (expectation actionOriginExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	if expectation.action == nil || !expectation.action.valid {
		return disprovedResult(evaluationText("invocation action"), evaluationText("configured trigger origin"), evaluationText("invalid action locator"))
	}
	selection := inspection.Selection.Action
	if selection.Selected != expectation.action.action {
		return disprovedResult(evaluationText("invocation action"), evaluationText("configured trigger origin"), evaluationText(fmt.Sprintf("selected action %v", selection.Selected)))
	}
	resolved := resolveOriginInspection(expectation.origin, inspection)
	if resolved.status == locatorResolutionNotProvable {
		return notProvableResult(evaluationText("invocation action"), evaluationText("configured trigger origin"), resolved.actual)
	}
	if resolved.status != locatorResolutionFound {
		return disprovedResult(evaluationText("invocation action"), evaluationText("configured trigger origin"), resolved.actual)
	}
	if selection.Input == resolved.value.ID {
		return provedResult()
	}
	return disprovedResult(evaluationText("invocation action"), evaluationText("configured trigger origin identity"), evaluationText("different selected trigger origin identity"))
}
