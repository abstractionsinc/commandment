package clitest

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/abstractionsinc/commandment/cli"
)

// CommandLocator identifies one canonical command for clitest semantic expectation.
// Its zero value is invalid.
type CommandLocator struct {
	_     [0]func()
	state *commandLocatorState
}

// InvocationActionLocator identifies one command action for clitest semantic expectation. Its
// zero value is invalid.
type InvocationActionLocator struct {
	_     [0]func()
	state *actionLocatorState
}

// RootCommand returns a locator for the application root command.
func RootCommand() CommandLocator { return commandRootLocator() }

// Command returns a locator for the command identified by the exact semantic path.
// Human-coordinate resolution never silently chooses the first ambiguous match.
func Command(path ...string) CommandLocator { return commandPathLocator(path) }

// CommandByID returns a locator for the command with the exact public CommandID within the
// observation being evaluated.
func CommandByID(id cli.CommandID) CommandLocator { return commandIDLocator(id) }

// InvocationAction returns a locator for the exact InvocationAction selection subject.
func InvocationAction(action cli.InvocationAction) InvocationActionLocator {
	return actionLocator(action)
}

// CommandsExactly checks the complete authoritative command collection for the
// applicable Schema, Reference, or Inspection observation.
func CommandsExactly(commands ...CommandLocator) DeclarationExpectation {
	return commandsExactlyExpectation(commands...)
}

func (CommandLocator) diagnosticSubject() {}

func (CommandLocator) noticeSubject() {}

func (CommandLocator) tokenOwner() {}

func (CommandLocator) completionTargetSubject() {}

// Command returns a locator for the uniquely resolved child command named name under the located
// command.
func (locator CommandLocator) Command(name string) CommandLocator {
	return commandChildLocator(locator, name)
}

// Exists returns an expectation that checks the located command exists. Positive existence may be
// satisfied by truthful partial output when the subject is present.
func (locator CommandLocator) Exists() DeclarationExpectation {
	return commandExistenceExpectation(locator, true)
}

// DoesNotExist returns an expectation that checks the located command does not exist; an
// incomplete observation cannot establish absence.
func (locator CommandLocator) DoesNotExist() DeclarationExpectation {
	return commandExistenceExpectation(locator, false)
}

// Hidden returns an expectation that checks the located command is marked hidden by the public
// Commandment observation.
func (locator CommandLocator) Hidden() DeclarationExpectation {
	return commandHiddenExpectation(locator)
}

// Runnable returns an expectation that checks the located command is declared runnable.
func (locator CommandLocator) Runnable() DeclarationExpectation {
	return commandRunnableExpectation(locator, true)
}

// NotRunnable returns an expectation that checks the located command is not declared runnable.
func (locator CommandLocator) NotRunnable() DeclarationExpectation {
	return commandRunnableExpectation(locator, false)
}

// Lifecycle returns an expectation that checks the located command has the expected effective
// lifecycle state; this is distinct from release-lifecycle metadata.
func (locator CommandLocator) Lifecycle(state cli.LifecycleState) DeclarationExpectation {
	return commandLifecycleExpectation(locator, state)
}

// ChildrenExactly returns an expectation that checks the located command's child commands are
// exactly the supplied locators in authoritative public order. Exactness is not inferred from an
// incomplete observation.
func (locator CommandLocator) ChildrenExactly(values ...CommandLocator) DeclarationExpectation {
	expectation := commandChildrenExpectation{
		command: locator.state,
		items:   commandLocatorStates(values),
	}
	return expectation
}

// SignalPoliciesExactly returns an expectation that checks the located command's signal policies
// are exactly the supplied locators in authoritative public order. Exactness is not inferred from
// an incomplete observation.
func (locator CommandLocator) SignalPoliciesExactly(values ...SignalPolicyLocator) DeclarationExpectation {
	expectation := commandSignalPoliciesExpectation{
		command: locator.state,
		items:   signalPolicyLocatorStates(values),
	}
	return expectation
}

// Available returns an expectation that checks the located command is available in the applicable
// public observation.
func (locator CommandLocator) Available() InspectionExpectation {
	return commandAvailabilityExpectation(locator, true)
}

// Unavailable returns an expectation that checks the located command is unavailable in the
// applicable public observation.
func (locator CommandLocator) Unavailable() InspectionExpectation {
	return commandAvailabilityExpectation(locator, false)
}

// Visible returns an expectation that checks the located command is visible in the applicable
// public observation.
func (locator CommandLocator) Visible() InspectionExpectation {
	return commandVisibilityExpectation(locator, true)
}

// NotVisible returns an expectation that checks the located command is not visible in the
// applicable public observation.
func (locator CommandLocator) NotVisible() InspectionExpectation {
	return commandVisibilityExpectation(locator, false)
}

// NamespaceCompleteness returns an expectation that checks the located command has the expected
// invocation-time CommandNamespaceCompleteness.
func (locator CommandLocator) NamespaceCompleteness(state cli.CommandNamespaceCompleteness) InspectionExpectation {
	return commandNamespaceExpectation(locator, state)
}

// EffectiveDialect checks the exact effective command dialect exposed by the
// public Schema, Reference, or Inspection observation for this command.
func (locator CommandLocator) EffectiveDialect(expected cli.CommandDialect) DeclarationExpectation {
	value := commandDialectExpectationValue{
		locator:  locator.state,
		expected: cloneExpectedCommandDialect(expected),
	}
	return value
}

// Origin checks the framework-established declaration origin projected for this
// command. It observes the public artifact; it does not infer origin from names
// or caller-provided metadata.
func (locator CommandLocator) Origin(expected cli.DeclarationOrigin) DeclarationExpectation {
	value := commandOriginExpectationValue{
		locator: locator.state,
		origin:  expected,
	}
	return value
}

// Name returns a reusable declaration expectation that checks the public name exposed by the
// located command.
func (locator CommandLocator) Name(expected string) DeclarationExpectation {
	value := commandNameExpectationValue{
		locator: locator.state,
		name:    expected,
	}
	return value
}

// ActionEntries returns a locator that follows the public action entries relationship from the
// located command. The returned locator carries locator coordinates, not observed state.
func (locator CommandLocator) ActionEntries(action cli.InvocationAction) ActionEntriesLocator {
	actionEntriesLocator := ActionEntriesLocator{
		command: locator.state,
		action:  action,
	}
	return actionEntriesLocator
}

// ObservedDoesNotExist returns an Inspection expectation that no matching command is present in the returned
// Inspection.Commands snapshot.
// This is observed-snapshot absence and may succeed even when a namespace is incomplete; unlike complete-domain
// nonexistence expectation, it makes no claim about commands omitted from the observation.
func (locator CommandLocator) ObservedDoesNotExist() InspectionExpectation {
	expectation := observedCommandDoesNotExistExpectation{
		locator: locator.state,
	}
	return expectation
}

// SourceDescription returns a Schema expectation that checks the located
// command's source description.
func (locator CommandLocator) SourceDescription(expected cli.Description) SchemaExpectation {
	expectation := commandSourceDescriptionExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// PresentedDescription returns a Reference and Inspection expectation that
// checks the located command's resolved description.
func (locator CommandLocator) PresentedDescription(expected cli.Description) PresentedDeclarationExpectation {
	expectation := commandPresentedDescriptionExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// SourceExamplesExactly returns a Schema expectation that checks the located
// command's complete ordered source examples.
func (locator CommandLocator) SourceExamplesExactly(expected ...cli.Example) SchemaExpectation {
	expectation := commandSourceExamplesExpectation{
		locator:  locator.state,
		expected: slices.Clone(expected),
	}
	return expectation
}

// PresentedExamplesExactly returns a Reference and Inspection expectation that
// checks the located command's complete ordered resolved examples. Exactness is
// not provable from a truncated Inspection.
func (locator CommandLocator) PresentedExamplesExactly(expected ...cli.Example) PresentedDeclarationExpectation {
	expectation := commandPresentedExamplesExpectation{
		locator:  locator.state,
		expected: slices.Clone(expected),
	}
	return expectation
}

type commandChildrenExpectation struct {
	command *commandLocatorState
	items   []*commandLocatorState
}

type commandEntriesExpectation struct {
	command *commandLocatorState
	items   []*commandEntryLocatorState
}

type commandValuesExpectation struct {
	command *commandLocatorState
	items   []*valueLocatorState
}

type commandsExactlyExpectationValue struct{ states []*commandLocatorState }

type commandLocatorMode uint8

const (
	commandLocatorInvalid commandLocatorMode = iota
	commandLocatorPath
	commandLocatorID
	commandLocatorChild
)

type commandLocatorState struct {
	mode commandLocatorMode

	path []string
	id   cli.CommandID

	parent *commandLocatorState
	name   string
}

type actionLocatorState struct {
	valid  bool
	action cli.InvocationAction
}

type commandDialectExpectationValue struct {
	locator  *commandLocatorState
	expected cli.CommandDialect
}

type commandExistenceExpectationValue struct {
	locator *commandLocatorState
	exists  bool
}

type commandHiddenExpectationValue struct {
	locator *commandLocatorState
}

type commandRunnableExpectationValue struct {
	locator  *commandLocatorState
	runnable bool
}

type commandLifecycleExpectationValue struct {
	locator   *commandLocatorState
	lifecycle cli.LifecycleState
}

type commandNameExpectationValue struct {
	locator *commandLocatorState
	name    string
}

type commandOriginExpectationValue struct {
	locator *commandLocatorState
	origin  cli.DeclarationOrigin
}

type commandAvailabilityExpectationValue struct {
	locator   *commandLocatorState
	available bool
}

type commandVisibilityExpectationValue struct {
	locator *commandLocatorState
	visible bool
}

type commandNamespaceExpectationValue struct {
	locator   *commandLocatorState
	namespace cli.CommandNamespaceCompleteness
}

type commandSignalPoliciesExpectation struct {
	command *commandLocatorState
	items   []*signalPolicyLocatorState
}

func commandLocatorLabel(state *commandLocatorState) evaluationField {
	return commandLocatorLabelAtDepth(state, 0)
}

func commandLocatorLabelAtDepth(state *commandLocatorState, depth int) evaluationField {
	if state == nil {
		return evaluationText("")
	}
	if depth >= failureSubjectMaxLocatorDepth {
		return evaluationText("nested locator")
	}

	switch state.mode {
	case commandLocatorPath:
		if len(state.path) == 0 {
			return evaluationText("root command")
		}
		field := commandPathField{
			values: state.path,
		}
		return field
	case commandLocatorID:
		field := commandIDField{
			value: state.id,
		}
		return field
	case commandLocatorChild:
		parent := commandLocatorLabelAtDepth(state.parent, depth+1)
		if emptyEvaluationField(parent) {
			parent = evaluationText("command")
		}
		return qualifySubject(parent, subjectNamed(semanticSubjectChildCommand, state.name))
	default:
		return evaluationText("")
	}
}

//nolint:dupl // Schema owns direct typed command resolution; sharing with Reference would require semantic projection callbacks.
func resolveSchemaCommandLocator(state *commandLocatorState, schema *cli.Schema) locatorResolution[cli.CommandSchema] {
	label := commandLocatorLabel(state)
	if schema == nil {
		resolution := locatorResolution[cli.CommandSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  evaluationText("invalid schema"),
		}

		return resolution
	}
	if state == nil || state.mode == commandLocatorInvalid {
		resolution := locatorResolution[cli.CommandSchema]{
			status:  locatorResolutionInvalid,
			subject: semanticSubjectCommand,
			detail:  observed(semanticSubjectCommand, observationInvalidLocator),
		}

		return resolution
	}

	switch state.mode {
	case commandLocatorID:
		command, ok := schema.Commands[state.id]
		if ok {
			resolution := locatorResolution[cli.CommandSchema]{
				status:  locatorResolutionFound,
				value:   command,
				subject: label,
			}

			return resolution
		}

		resolution := locatorResolution[cli.CommandSchema]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectCommand, observationAbsent),
		}

		return resolution
	case commandLocatorPath:
		matches := make([]cli.CommandSchema, 0, 1)
		for _, command := range schema.Commands {
			if slices.Equal(command.Path, state.path) {
				matches = append(matches, command)
			}
		}

		return resolveUniqueObserved(
			matches,
			true,
			label,
			observed(semanticSubjectCommand, observationAbsent),
			evaluationText("command path domain is incomplete"),
			evaluationText("command path is ambiguous"),
		)
	case commandLocatorChild:
		parent := resolveSchemaCommandLocator(state.parent, schema)
		if parent.status != locatorResolutionFound {
			resolution := locatorResolution[cli.CommandSchema]{
				status:  parent.status,
				subject: label,
				detail:  parent.detail,
			}

			return resolution
		}

		matches := make([]cli.CommandSchema, 0, 1)
		for _, id := range parent.value.Children {
			command, ok := schema.Commands[id]
			if ok && command.Name == state.name {
				matches = append(matches, command)
			}
		}

		return resolveUniqueObserved(
			matches,
			true,
			label,
			observed(semanticSubjectChildCommand, observationAbsent),
			evaluationText("child command domain is incomplete"),
			evaluationText("child command is ambiguous"),
		)
	default:
		resolution := locatorResolution[cli.CommandSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCommand, observationInvalidLocator),
		}

		return resolution
	}
}

//nolint:dupl // Reference owns direct typed command resolution; sharing with Schema would require semantic projection callbacks.
func resolveReferenceCommandLocator(state *commandLocatorState, reference cli.Reference) locatorResolution[cli.CommandReference] {
	label := commandLocatorLabel(state)
	if state == nil || state.mode == commandLocatorInvalid {
		resolution := locatorResolution[cli.CommandReference]{
			status:  locatorResolutionInvalid,
			subject: semanticSubjectCommand,
			detail:  observed(semanticSubjectCommand, observationInvalidLocator),
		}

		return resolution
	}

	switch state.mode {
	case commandLocatorID:
		command, ok := reference.Commands[state.id]
		if ok {
			resolution := locatorResolution[cli.CommandReference]{
				status:  locatorResolutionFound,
				value:   command,
				subject: label,
			}

			return resolution
		}

		resolution := locatorResolution[cli.CommandReference]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectCommand, observationAbsent),
		}

		return resolution
	case commandLocatorPath:
		matches := make([]cli.CommandReference, 0, 1)
		for _, command := range reference.Commands {
			if slices.Equal(command.Path, state.path) {
				matches = append(matches, command)
			}
		}

		return resolveUniqueObserved(
			matches,
			true,
			label,
			observed(semanticSubjectCommand, observationAbsent),
			evaluationText("command path domain is incomplete"),
			evaluationText("command path is ambiguous"),
		)
	case commandLocatorChild:
		parent := resolveReferenceCommandLocator(state.parent, reference)
		if parent.status != locatorResolutionFound {
			resolution := locatorResolution[cli.CommandReference]{
				status:  parent.status,
				subject: label,
				detail:  parent.detail,
			}

			return resolution
		}

		matches := make([]cli.CommandReference, 0, 1)
		for _, id := range parent.value.Children {
			command, ok := reference.Commands[id]
			if ok && command.Name == state.name {
				matches = append(matches, command)
			}
		}

		return resolveUniqueObserved(
			matches,
			true,
			label,
			observed(semanticSubjectChildCommand, observationAbsent),
			evaluationText("child command domain is incomplete"),
			evaluationText("child command is ambiguous"),
		)
	default:
		resolution := locatorResolution[cli.CommandReference]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCommand, observationInvalidLocator),
		}

		return resolution
	}
}

func resolveInspectionCommandID(state *commandLocatorState, commands map[cli.CommandID]cli.CommandInspection) (cli.CommandID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == commandLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectCommand, observationInvalidLocator)
	}

	switch state.mode {
	case commandLocatorID:
		return state.id, locatorResolutionFound, evaluationText("")
	case commandLocatorChild:
		if commands == nil {
			return 0, locatorResolutionNotProvable, evaluationText("this operation exposes command IDs but not the command relationship domain")
		}
		parentID, status, detail := resolveInspectionCommandID(state.parent, commands)
		if status != locatorResolutionFound {
			return 0, status, detail
		}
		return resolveInspectionCommandChild(parentID, state.name, commands)
	case commandLocatorPath:
		if commands == nil {
			return 0, locatorResolutionNotProvable, evaluationText("this operation exposes command IDs but not the command-path lookup domain")
		}
		matches := make([]cli.CommandID, 0, 1)
		for id, command := range commands {
			if slices.Equal(command.Path, state.path) {
				matches = append(matches, id)
			}
		}
		slices.Sort(matches)
		if len(matches) == 1 {
			return matches[0], locatorResolutionFound, evaluationText("")
		}
		if len(matches) > 1 {
			field := collectionCardinalityField{
				name:  "matches command-path IDs",
				count: len(matches),
			}
			return 0, locatorResolutionAmbiguous, field
		}
		return resolveInspectionCommandPath(state.path, commands)
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectCommand, observationInvalidLocator)
	}
}

func resolveInspectionCommandChild(parentID cli.CommandID, name string, commands map[cli.CommandID]cli.CommandInspection) (cli.CommandID, locatorResolutionStatus, evaluationField) {
	parent, ok := commands[parentID]
	if !ok {
		return 0, locatorResolutionNotProvable, evaluationText("parent command details are absent")
	}

	matches := make([]cli.CommandID, 0, 1)
	missingDetail := false
	for _, id := range parent.Children {
		command, exists := commands[id]
		if !exists {
			missingDetail = true
			continue
		}
		if command.Name == name {
			matches = append(matches, id)
		}
	}
	slices.Sort(matches)
	if len(matches) == 1 {
		return matches[0], locatorResolutionFound, evaluationText("")
	}
	if len(matches) > 1 {
		field := collectionCardinalityField{
			name:  "matches child-command IDs",
			count: len(matches),
		}
		return 0, locatorResolutionAmbiguous, field
	}
	if missingDetail {
		return 0, locatorResolutionNotProvable, evaluationText("child command details needed for name lookup are absent")
	}
	if parent.NamespaceCompleteness == cli.CommandNamespaceComplete {
		return 0, locatorResolutionAbsent, observed(semanticSubjectChildCommand, observationAbsent)
	}
	return 0, locatorResolutionNotProvable, evaluationText(fmt.Sprintf("parent command namespace is %v", parent.NamespaceCompleteness))
}

func resolveInspectionCommandPath(path []string, commands map[cli.CommandID]cli.CommandInspection) (cli.CommandID, locatorResolutionStatus, evaluationField) {
	roots := make([]cli.CommandID, 0, 1)
	for id, command := range commands {
		if len(command.Path) == 0 {
			roots = append(roots, id)
		}
	}
	slices.Sort(roots)
	if len(roots) == 0 {
		return 0, locatorResolutionNotProvable, evaluationText("root command is not observable")
	}
	if len(roots) > 1 {
		field := collectionCardinalityField{
			name:  "matches root-command IDs",
			count: len(roots),
		}
		return 0, locatorResolutionAmbiguous, field
	}
	if len(path) == 0 {
		return roots[0], locatorResolutionFound, evaluationText("")
	}

	current := roots[0]
	prefix := make([]string, 0, len(path))
	for _, segment := range path {
		next, status, detail := resolveInspectionCommandChild(current, segment, commands)
		if status != locatorResolutionFound {
			return 0, status, detail
		}
		prefix = append(prefix, segment)
		command := commands[next]
		if !slices.Equal(command.Path, prefix) {
			return 0, locatorResolutionNotProvable, evaluationText("command relationship topology contradicts observed command path")
		}
		current = next
	}
	return current, locatorResolutionFound, evaluationText("")
}

func inspectionCommandsComplete(inspection cli.Inspection) (bool, evaluationField) {
	if inspection.Truncated {
		return false, observed(semanticSubjectInspection, observationTruncated)
	}

	seen := make(map[cli.CommandID]struct{}, len(inspection.CommandOrder))
	roots := 0

	for _, id := range inspection.CommandOrder {
		if _, duplicate := seen[id]; duplicate {
			return false, evaluationText("command order contains a duplicate identity")
		}
		seen[id] = struct{}{}

		command, ok := inspection.Commands[id]
		if !ok {
			return false, evaluationText("command order references missing command detail")
		}
		if command.ID != id {
			return false, evaluationText("command detail identity disagrees with command order")
		}
		if command.NamespaceCompleteness != cli.CommandNamespaceComplete {
			return false, evaluationText("command namespace is incomplete")
		}
		if len(command.Path) == 0 {
			roots++
		}
		for _, childID := range command.Children {
			if _, ok := inspection.Commands[childID]; !ok {
				return false, evaluationText("command child relationship references missing detail")
			}
		}
	}

	if len(seen) != len(inspection.Commands) {
		return false, evaluationText("command order and command details are not exhaustive peers")
	}
	for id, command := range inspection.Commands {
		if _, ok := seen[id]; !ok {
			return false, evaluationText("command detail exists outside command order")
		}
		if command.ID != id {
			return false, evaluationText("command map identity disagrees with command detail")
		}
	}
	if roots != 1 {
		return false, evaluationText(fmt.Sprintf("observable command domain has %d roots", roots))
	}

	return true, evaluationText("")
}

func resolveInspectionCommandLocator(state *commandLocatorState, inspection cli.Inspection) locatorResolution[cli.CommandInspection] {
	var resolution locatorResolution[cli.CommandInspection]
	label := commandLocatorLabel(state)
	id, status, detail := resolveInspectionCommandID(state, inspection.Commands)
	if status != locatorResolutionFound {
		if status == locatorResolutionAbsent && inspection.Truncated {
			status = locatorResolutionNotProvable
			detail = observed(semanticSubjectInspection, observationTruncated)
		}
		resolution = locatorResolution[cli.CommandInspection]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}

	command, ok := inspection.Commands[id]
	if ok {
		resolution = locatorResolution[cli.CommandInspection]{
			status:  locatorResolutionFound,
			value:   command,
			subject: label,
		}
		return resolution
	}
	complete, closureDetail := inspectionCommandsComplete(inspection)
	if !complete {
		resolution = locatorResolution[cli.CommandInspection]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  closureDetail,
		}
		return resolution
	}
	resolution = locatorResolution[cli.CommandInspection]{
		status:  locatorResolutionAbsent,
		subject: label,
		detail:  observed(semanticSubjectCommand, observationAbsent),
	}
	return resolution
}

func commandLocatorStates(values []CommandLocator) []*commandLocatorState {
	states := make([]*commandLocatorState, len(values))
	for index, value := range values {
		states[index] = value.state
	}
	return states
}

func collectionResolutionFailure(label evaluationField, name string, index int, status locatorResolutionStatus, detail evaluationField) (evaluationResult, bool) {
	switch status {
	case locatorResolutionFound:
		return evaluationResult{}, false
	case locatorResolutionNotProvable:
		return notProvableResult(label, evaluationText("resolvable ordered "+name+" locators"), detail), true
	default:
		detail := prefixedDetail{
			prefix: fmt.Sprintf("item %d: ", index),
			detail: detail,
		}
		return disprovedResult(label, evaluationText("resolvable ordered "+name+" locators"), detail), true
	}
}

func collectionClosureFailure(label evaluationField, name string, detail evaluationField) evaluationResult {
	return notProvableResult(label, evaluationText("closed exact ordered "+name+" collection"), detail)
}

func commandsExactlyExpectation(locators ...CommandLocator) DeclarationExpectation {
	value := commandsExactlyExpectationValue{
		states: commandLocatorStates(locators),
	}
	return value
}

func inspectionCommandTopologyClosure(inspection cli.Inspection) (locatorResolutionStatus, evaluationField) {
	ordered := make(map[cli.CommandID]struct{}, len(inspection.CommandOrder))
	for _, id := range inspection.CommandOrder {
		if _, duplicate := ordered[id]; duplicate {
			return locatorResolutionNotProvable, evaluationText(fmt.Sprintf("command order contains duplicate command %v", id))
		}
		command, ok := inspection.Commands[id]
		if !ok {
			return locatorResolutionNotProvable, evaluationText(fmt.Sprintf("command order references missing command details for %v", id))
		}
		if command.ID != id {
			return locatorResolutionNotProvable, evaluationText(fmt.Sprintf("command detail identity %v disagrees with command order identity %v", command.ID, id))
		}
		if command.NamespaceCompleteness != cli.CommandNamespaceComplete {
			return locatorResolutionNotProvable, evaluationText(fmt.Sprintf("command %v namespace is %v", id, command.NamespaceCompleteness))
		}
		ordered[id] = struct{}{}
	}
	if len(ordered) != len(inspection.Commands) {
		return locatorResolutionNotProvable, evaluationText("command order and command details are not exhaustive peers")
	}
	roots := 0
	for id, command := range inspection.Commands {
		if len(command.Path) == 0 {
			roots++
		}
		for _, child := range command.Children {
			if _, ok := inspection.Commands[child]; !ok {
				return locatorResolutionNotProvable, evaluationText(fmt.Sprintf("command %v references missing child details for %v", id, child))
			}
		}
	}
	if roots != 1 {
		return locatorResolutionNotProvable, evaluationText(fmt.Sprintf("command topology has %d observable roots", roots))
	}
	return locatorResolutionFound, evaluationText("")
}

func commandChildLocator(parent CommandLocator, name string) CommandLocator {
	state := &commandLocatorState{
		mode:   commandLocatorChild,
		parent: parent.state,
		name:   name,
	}
	locator := CommandLocator{
		state: state,
	}
	return locator
}

func commandCanonicalEntryLocator(parent CommandLocator) CommandEntryLocator {
	state := &commandEntryLocatorState{
		mode:    commandEntryLocatorCanonical,
		command: parent.state,
	}
	locator := CommandEntryLocator{
		state: state,
	}
	return locator
}

func commandAliasLocator(parent CommandLocator, name string) CommandEntryLocator {
	state := &commandEntryLocatorState{
		mode:    commandEntryLocatorAlias,
		command: parent.state,
		name:    name,
	}
	locator := CommandEntryLocator{
		state: state,
	}
	return locator
}

func commandShortcutLocator(parent CommandLocator, name string) CommandEntryLocator {
	state := &commandEntryLocatorState{
		mode:    commandEntryLocatorShortcut,
		command: parent.state,
		name:    name,
	}
	locator := CommandEntryLocator{
		state: state,
	}
	return locator
}

func commandArgumentLocator(parent CommandLocator, name string) ArgumentTermLocator {
	state := &argumentTermLocatorState{
		mode:    argumentTermLocatorCommandName,
		command: parent.state,
		text:    name,
	}
	locator := ArgumentTermLocator{
		state: state,
	}
	return locator
}

func commandArgumentLiteralLocator(parent CommandLocator, literal string) ArgumentTermLocator {
	state := &argumentTermLocatorState{
		mode:    argumentTermLocatorCommandLiteral,
		command: parent.state,
		text:    literal,
	}
	locator := ArgumentTermLocator{
		state: state,
	}
	return locator
}

func commandEnvironmentLocator(parent CommandLocator, name string) EnvironmentInputLocator {
	state := &environmentInputLocatorState{
		mode:    environmentInputLocatorCommandName,
		command: parent.state,
		name:    name,
	}
	locator := EnvironmentInputLocator{
		state: state,
	}
	return locator
}

func commandEnvironmentAtLocator(parent CommandLocator, index int) EnvironmentInputLocator {
	state := &environmentInputLocatorState{
		mode:    environmentInputLocatorCommandIndex,
		command: parent.state,
		index:   index,
	}
	locator := EnvironmentInputLocator{
		state: state,
	}
	return locator
}

func commandValueAtLocator(parent CommandLocator, index int) ValueLocator {
	state := &valueLocatorState{
		mode:    valueLocatorCommandIndex,
		command: parent.state,
		index:   index,
	}
	locator := ValueLocator{
		state: state,
	}
	return locator
}

func cloneExpectedCommandDialect(value cli.CommandDialect) cli.CommandDialect {
	value.EndOfOptions = slices.Clone(value.EndOfOptions)
	return value
}

func compareCommandDialect(label evaluationField, expected cli.CommandDialect, actual *cli.CommandDialect) evaluationResult {
	if actual == nil {
		return disprovedResult(qualifySubject(label, evaluationText("effective dialect")), evaluationText("present"), semanticRequirementAbsent)
	}
	base := qualifySubject(label, evaluationText("effective dialect"))
	if result, mismatch := compareTypedField(base, "FlagOrder", evaluationText(fmt.Sprintf("%v", expected.FlagOrder)), evaluationText(fmt.Sprintf("%v", actual.FlagOrder)), expected.FlagOrder == actual.FlagOrder); mismatch {
		return result
	}
	if result, mismatch := compareTypedField(base, "ShortFlagClustering", evaluationText(fmt.Sprintf("%v", expected.ShortFlagClustering)), evaluationText(fmt.Sprintf("%v", actual.ShortFlagClustering)), expected.ShortFlagClustering == actual.ShortFlagClustering); mismatch {
		return result
	}
	if result, mismatch := compareTypedField(base, "FlagMatching", evaluationText(fmt.Sprintf("%v", expected.FlagMatching)), evaluationText(fmt.Sprintf("%v", actual.FlagMatching)), expected.FlagMatching == actual.FlagMatching); mismatch {
		return result
	}
	if result, mismatch := compareTypedField(base, "CommandMatching", evaluationText(fmt.Sprintf("%v", expected.CommandMatching)), evaluationText(fmt.Sprintf("%v", actual.CommandMatching)), expected.CommandMatching == actual.CommandMatching); mismatch {
		return result
	}
	if result, mismatch := compareTypedField(base, "FlagNormalization", evaluationText(fmt.Sprintf("%v", expected.FlagNormalization)), evaluationText(fmt.Sprintf("%v", actual.FlagNormalization)), expected.FlagNormalization == actual.FlagNormalization); mismatch {
		return result
	}
	if result, mismatch := compareTypedField(base, "CommandNormalization", evaluationText(fmt.Sprintf("%v", expected.CommandNormalization)), evaluationText(fmt.Sprintf("%v", actual.CommandNormalization)), expected.CommandNormalization == actual.CommandNormalization); mismatch {
		return result
	}
	if result, mismatch := compareTypedField(base, "DashlessFlags", evaluationText(fmt.Sprintf("%v", expected.DashlessFlags)), evaluationText(fmt.Sprintf("%v", actual.DashlessFlags)), expected.DashlessFlags == actual.DashlessFlags); mismatch {
		return result
	}
	if !slices.Equal(expected.EndOfOptions, actual.EndOfOptions) {
		return orderedCollectionMismatch(qualifySubject(base, evaluationText("EndOfOptions")), "end-of-options", expected.EndOfOptions, actual.EndOfOptions)
	}
	if result, mismatch := compareTypedField(base, "ResponseFiles", evaluationText(strconv.FormatBool(expected.ResponseFiles)), evaluationText(strconv.FormatBool(actual.ResponseFiles)), expected.ResponseFiles == actual.ResponseFiles); mismatch {
		return result
	}
	return provedResult()
}

func compareTypedField(base evaluationField, field string, expected, actual evaluationField, equal bool) (evaluationResult, bool) {
	if equal {
		return evaluationResult{}, false
	}
	return disprovedResult(qualifySubject(base, evaluationText(field)), expected, actual), true
}

func commandExistenceOutcome(status locatorResolutionStatus, subject, detail evaluationField, exists bool) evaluationResult {
	expected := evaluationText(fmt.Sprintf("exists=%t", exists))

	switch status {
	case locatorResolutionFound:
		return commandPropertyOutcome(subject, expected, exists, evaluationText("exists=true"))
	case locatorResolutionAbsent:
		return commandPropertyOutcome(subject, expected, !exists, evaluationText("exists=false"))
	case locatorResolutionNotProvable:
		return notProvableResult(subject, expected, detail)
	case locatorResolutionAmbiguous, locatorResolutionInvalid:
		return disprovedResult(subject, expected, detail)
	default:
		panic("clitest: invalid command locator resolution status")
	}
}

func commandExistenceExpectation(locator CommandLocator, exists bool) DeclarationExpectation {
	value := commandExistenceExpectationValue{
		locator: locator.state,
		exists:  exists,
	}
	return value
}

func commandHiddenExpectation(locator CommandLocator) DeclarationExpectation {
	value := commandHiddenExpectationValue{
		locator: locator.state,
	}
	return value
}

func commandRunnableExpectation(locator CommandLocator, runnable bool) DeclarationExpectation {
	value := commandRunnableExpectationValue{
		locator:  locator.state,
		runnable: runnable,
	}
	return value
}

func commandLifecycleExpectation(locator CommandLocator, lifecycle cli.LifecycleState) DeclarationExpectation {
	value := commandLifecycleExpectationValue{
		locator:   locator.state,
		lifecycle: lifecycle,
	}
	return value
}

func resolveSchemaCommand(state *commandLocatorState, schema *cli.Schema) (cli.CommandSchema, bool, evaluationField) {
	resolved := resolveSchemaCommandLocator(state, schema)
	return resolved.value, resolved.status == locatorResolutionFound, resolved.subject
}

func resolveReferenceCommand(state *commandLocatorState, reference cli.Reference) (cli.CommandReference, bool, evaluationField) {
	resolved := resolveReferenceCommandLocator(state, reference)
	return resolved.value, resolved.status == locatorResolutionFound, resolved.subject
}

func commandPropertyOutcome(label, expect evaluationField, ok bool, actual evaluationField) evaluationResult {
	if ok {
		return provedResult()
	}
	return disprovedResult(label, expect, actual)
}

func commandAvailabilityExpectation(locator CommandLocator, available bool) InspectionExpectation {
	value := commandAvailabilityExpectationValue{
		locator:   locator.state,
		available: available,
	}
	return value
}

func commandVisibilityExpectation(locator CommandLocator, visible bool) InspectionExpectation {
	value := commandVisibilityExpectationValue{
		locator: locator.state,
		visible: visible,
	}
	return value
}

func commandNamespaceExpectation(locator CommandLocator, namespace cli.CommandNamespaceCompleteness) InspectionExpectation {
	value := commandNamespaceExpectationValue{
		locator:   locator.state,
		namespace: namespace,
	}
	return value
}

func commandRootLocator() CommandLocator {
	state := &commandLocatorState{
		mode: commandLocatorPath,
		path: []string{},
	}
	locator := CommandLocator{
		state: state,
	}
	return locator
}

func commandPathLocator(path []string) CommandLocator {
	state := &commandLocatorState{
		mode: commandLocatorPath,
		path: slices.Clone(path),
	}
	locator := CommandLocator{
		state: state,
	}
	return locator
}

func commandIDLocator(id cli.CommandID) CommandLocator {
	state := &commandLocatorState{
		mode: commandLocatorID,
		id:   id,
	}
	locator := CommandLocator{
		state: state,
	}
	return locator
}

func commandPathCoordinate(state *commandLocatorState) ([]string, bool) {
	if state == nil {
		return nil, false
	}
	switch state.mode {
	case commandLocatorPath:
		return slices.Clone(state.path), true
	case commandLocatorChild:
		parent, ok := commandPathCoordinate(state.parent)
		if !ok {
			return nil, false
		}
		return append(parent, state.name), true
	default:
		return nil, false
	}
}

func (commandChildrenExpectation) schemaExpectation() {}

func (commandChildrenExpectation) referenceExpectation() {}

func (commandChildrenExpectation) inspectionExpectation() {}

//nolint:dupl // Typed collection proofs intentionally keep each public declaration family direct; generic resolver callbacks would recreate the rejected universal framework.
func (expectation commandChildrenExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	command := resolveSchemaCommandLocator(expectation.command, schema)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("children exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("children"))
	actual := command.value.Children
	for _, id := range actual {
		if _, ok := schema.Commands[id]; !ok {
			return collectionClosureFailure(label, "children", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.CommandID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveSchemaCommandLocator(state, schema)
		if outcome, failed := collectionResolutionFailure(label, "children", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "children", expected, actual)
}

//nolint:dupl // Typed collection proofs intentionally keep each public declaration family direct; generic resolver callbacks would recreate the rejected universal framework.
func (expectation commandChildrenExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	command := resolveReferenceCommandLocator(expectation.command, reference)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("children exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("children"))
	actual := command.value.Children
	for _, id := range actual {
		if _, ok := reference.Commands[id]; !ok {
			return collectionClosureFailure(label, "children", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.CommandID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveReferenceCommandLocator(state, reference)
		if outcome, failed := collectionResolutionFailure(label, "children", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "children", expected, actual)
}

func (expectation commandChildrenExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	command := resolveInspectionCommandLocator(expectation.command, inspection)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("children exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("children"))
	if inspection.Truncated {
		return collectionClosureFailure(label, "children", observed(semanticSubjectInspection, observationTruncated))
	}
	if command.value.NamespaceCompleteness != cli.CommandNamespaceComplete {
		return collectionClosureFailure(label, "children", evaluationText(fmt.Sprintf("command namespace is %v", command.value.NamespaceCompleteness)))
	}
	actual := command.value.Children
	for _, id := range actual {
		if _, ok := inspection.Commands[id]; !ok {
			return collectionClosureFailure(label, "children", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.CommandID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveInspectionCommandLocator(state, inspection)
		if outcome, failed := collectionResolutionFailure(label, "children", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "children", expected, actual)
}

func (commandEntriesExpectation) schemaExpectation() {}

func (commandEntriesExpectation) referenceExpectation() {}

func (commandEntriesExpectation) inspectionExpectation() {}

//nolint:dupl // Typed collection proofs intentionally keep each public declaration family direct; generic resolver callbacks would recreate the rejected universal framework.
func (expectation commandEntriesExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	command := resolveSchemaCommandLocator(expectation.command, schema)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("command entries exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("command entries"))
	actual := command.value.CommandEntries
	for _, id := range actual {
		if _, ok := schema.CommandEntries[id]; !ok {
			return collectionClosureFailure(label, "command entries", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.CommandEntryID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveSchemaCommandEntryLocator(state, schema)
		if outcome, failed := collectionResolutionFailure(label, "command entries", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "command entries", expected, actual)
}

//nolint:dupl // Typed collection proofs intentionally keep each public declaration family direct; generic resolver callbacks would recreate the rejected universal framework.
func (expectation commandEntriesExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	command := resolveReferenceCommandLocator(expectation.command, reference)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("command entries exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("command entries"))
	actual := command.value.CommandEntries
	for _, id := range actual {
		if _, ok := reference.CommandEntries[id]; !ok {
			return collectionClosureFailure(label, "command entries", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.CommandEntryID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveReferenceCommandEntryLocator(state, reference)
		if outcome, failed := collectionResolutionFailure(label, "command entries", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "command entries", expected, actual)
}

func (expectation commandEntriesExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	command := resolveInspectionCommandLocator(expectation.command, inspection)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("command entries exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("command entries"))
	if inspection.Truncated {
		return collectionClosureFailure(label, "command entries", observed(semanticSubjectInspection, observationTruncated))
	}
	if command.value.NamespaceCompleteness != cli.CommandNamespaceComplete {
		return collectionClosureFailure(label, "command entries", evaluationText(fmt.Sprintf("command namespace is %v", command.value.NamespaceCompleteness)))
	}
	actual := command.value.CommandEntries
	for _, id := range actual {
		if _, ok := inspection.CommandEntries[id]; !ok {
			return collectionClosureFailure(label, "command entries", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.CommandEntryID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveInspectionCommandEntryLocator(state, inspection)
		if outcome, failed := collectionResolutionFailure(label, "command entries", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "command entries", expected, actual)
}

func (commandValuesExpectation) schemaExpectation() {}

func (commandValuesExpectation) referenceExpectation() {}

func (commandValuesExpectation) inspectionExpectation() {}

//nolint:dupl // Typed collection proofs intentionally keep each public declaration family direct; generic resolver callbacks would recreate the rejected universal framework.
func (expectation commandValuesExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	command := resolveSchemaCommandLocator(expectation.command, schema)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("values exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("values"))
	actual := command.value.Values
	for _, id := range actual {
		if _, ok := schema.Values[id]; !ok {
			return collectionClosureFailure(label, "values", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.ValueID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveSchemaValueLocator(state, schema)
		if outcome, failed := collectionResolutionFailure(label, "values", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "values", expected, actual)
}

//nolint:dupl // Typed collection proofs intentionally keep each public declaration family direct; generic resolver callbacks would recreate the rejected universal framework.
func (expectation commandValuesExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	command := resolveReferenceCommandLocator(expectation.command, reference)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("values exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("values"))
	actual := command.value.Values
	for _, id := range actual {
		if _, ok := reference.Values[id]; !ok {
			return collectionClosureFailure(label, "values", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.ValueID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveReferenceValueLocator(state, reference)
		if outcome, failed := collectionResolutionFailure(label, "values", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "values", expected, actual)
}

func (expectation commandValuesExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	command := resolveInspectionCommandLocator(expectation.command, inspection)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("values exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("values"))
	if inspection.Truncated {
		return collectionClosureFailure(label, "values", observed(semanticSubjectInspection, observationTruncated))
	}
	if command.value.NamespaceCompleteness != cli.CommandNamespaceComplete {
		return collectionClosureFailure(label, "values", evaluationText(fmt.Sprintf("command namespace is %v", command.value.NamespaceCompleteness)))
	}
	actual := command.value.Values
	for _, id := range actual {
		if _, ok := inspection.Values[id]; !ok {
			return collectionClosureFailure(label, "values", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.ValueID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveInspectionValueLocator(state, inspection)
		if outcome, failed := collectionResolutionFailure(label, "values", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "values", expected, actual)
}

func (commandsExactlyExpectationValue) schemaExpectation() {}

func (commandsExactlyExpectationValue) referenceExpectation() {}

func (commandsExactlyExpectationValue) inspectionExpectation() {}

func (expectation commandsExactlyExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	label := evaluationText("commands")
	for _, id := range schema.CommandOrder {
		if _, ok := schema.Commands[id]; !ok {
			return collectionClosureFailure(label, "commands", evaluationText(fmt.Sprintf("command order references missing command %v", id)))
		}
	}
	expected := make([]cli.CommandID, len(expectation.states))
	for index, state := range expectation.states {
		resolved := resolveSchemaCommandLocator(state, schema)
		if outcome, failed := collectionResolutionFailure(label, "commands", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(schema.CommandOrder, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "commands", expected, schema.CommandOrder)
}

func (expectation commandsExactlyExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	label := evaluationText("commands")
	for _, id := range reference.CommandOrder {
		if _, ok := reference.Commands[id]; !ok {
			return collectionClosureFailure(label, "commands", evaluationText(fmt.Sprintf("command order references missing command %v", id)))
		}
	}
	expected := make([]cli.CommandID, len(expectation.states))
	for index, state := range expectation.states {
		resolved := resolveReferenceCommandLocator(state, reference)
		if outcome, failed := collectionResolutionFailure(label, "commands", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(reference.CommandOrder, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "commands", expected, reference.CommandOrder)
}

func (expectation commandsExactlyExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	label := evaluationText("commands")
	if inspection.Truncated {
		return collectionClosureFailure(label, "commands", observed(semanticSubjectInspection, observationTruncated))
	}
	if status, detail := inspectionCommandTopologyClosure(inspection); status != locatorResolutionFound {
		return collectionClosureFailure(label, "commands", detail)
	}
	expected := make([]cli.CommandID, len(expectation.states))
	for index, state := range expectation.states {
		resolved := resolveInspectionCommandLocator(state, inspection)
		if outcome, failed := collectionResolutionFailure(label, "commands", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(inspection.CommandOrder, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "commands", expected, inspection.CommandOrder)
}

func (commandDialectExpectationValue) schemaExpectation() {}

func (commandDialectExpectationValue) referenceExpectation() {}

func (commandDialectExpectationValue) inspectionExpectation() {}

func (expectation commandDialectExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	command := resolveSchemaCommandLocator(expectation.locator, schema)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("effective dialect"), command.detail)
	}
	return compareCommandDialect(command.subject, expectation.expected, command.value.Dialect)
}

func (expectation commandDialectExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	command := resolveReferenceCommandLocator(expectation.locator, reference)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("effective dialect"), command.detail)
	}
	return compareCommandDialect(command.subject, expectation.expected, command.value.Dialect)
}

func (expectation commandDialectExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	command := resolveInspectionCommandLocator(expectation.locator, inspection)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("effective dialect"), command.detail)
	}
	return compareCommandDialect(command.subject, expectation.expected, command.value.Dialect)
}

func (commandExistenceExpectationValue) schemaExpectation() {}

func (commandExistenceExpectationValue) referenceExpectation() {}

func (commandExistenceExpectationValue) inspectionExpectation() {}

func (expectation commandExistenceExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolved := resolveSchemaCommandLocator(expectation.locator, schema)
	return commandExistenceOutcome(resolved.status, resolved.subject, resolved.detail, expectation.exists)
}

func (expectation commandExistenceExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolved := resolveReferenceCommandLocator(expectation.locator, reference)
	return commandExistenceOutcome(resolved.status, resolved.subject, resolved.detail, expectation.exists)
}

func (expectation commandExistenceExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveInspectionCommandLocator(expectation.locator, inspection)
	return commandExistenceOutcome(resolved.status, resolved.subject, resolved.detail, expectation.exists)
}

func (commandHiddenExpectationValue) schemaExpectation() {}

func (commandHiddenExpectationValue) referenceExpectation() {}

func (commandHiddenExpectationValue) inspectionExpectation() {}

func (expectation commandHiddenExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolved := resolveSchemaCommandLocator(expectation.locator, schema)
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, evaluationText("hidden=true"), resolved.detail)
	}
	return commandPropertyOutcome(resolved.subject, evaluationText("hidden=true"), resolved.value.Hidden, evaluationText(fmt.Sprintf("hidden=%t", resolved.value.Hidden)))
}

func (expectation commandHiddenExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolved := resolveReferenceCommandLocator(expectation.locator, reference)
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, evaluationText("hidden=true"), resolved.detail)
	}
	return commandPropertyOutcome(resolved.subject, evaluationText("hidden=true"), resolved.value.Hidden, evaluationText(fmt.Sprintf("hidden=%t", resolved.value.Hidden)))
}

func (expectation commandHiddenExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveInspectionCommandLocator(expectation.locator, inspection)
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, evaluationText("hidden=true"), resolved.detail)
	}
	return commandPropertyOutcome(resolved.subject, evaluationText("hidden=true"), resolved.value.Hidden, evaluationText(fmt.Sprintf("hidden=%t", resolved.value.Hidden)))
}

func (commandRunnableExpectationValue) schemaExpectation() {}

func (commandRunnableExpectationValue) referenceExpectation() {}

func (commandRunnableExpectationValue) inspectionExpectation() {}

func (expectation commandRunnableExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolved := resolveSchemaCommandLocator(expectation.locator, schema)
	expected := evaluationText(fmt.Sprintf("runnable=%t", expectation.runnable))
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, expected, resolved.detail)
	}
	return commandPropertyOutcome(resolved.subject, expected, resolved.value.Runnable == expectation.runnable, evaluationText(fmt.Sprintf("runnable=%t", resolved.value.Runnable)))
}

func (expectation commandRunnableExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolved := resolveReferenceCommandLocator(expectation.locator, reference)
	expected := evaluationText(fmt.Sprintf("runnable=%t", expectation.runnable))
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, expected, resolved.detail)
	}
	return commandPropertyOutcome(resolved.subject, expected, resolved.value.Runnable == expectation.runnable, evaluationText(fmt.Sprintf("runnable=%t", resolved.value.Runnable)))
}

func (expectation commandRunnableExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveInspectionCommandLocator(expectation.locator, inspection)
	expected := evaluationText(fmt.Sprintf("runnable=%t", expectation.runnable))
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, expected, resolved.detail)
	}
	return commandPropertyOutcome(resolved.subject, expected, resolved.value.Runnable == expectation.runnable, evaluationText(fmt.Sprintf("runnable=%t", resolved.value.Runnable)))
}

func (commandLifecycleExpectationValue) schemaExpectation() {}

func (commandLifecycleExpectationValue) referenceExpectation() {}

func (commandLifecycleExpectationValue) inspectionExpectation() {}

func (expectation commandLifecycleExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolved := resolveSchemaCommandLocator(expectation.locator, schema)
	expected := evaluationText(fmt.Sprintf("lifecycle %v", expectation.lifecycle))
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, expected, resolved.detail)
	}
	return commandPropertyOutcome(resolved.subject, expected, resolved.value.LifecycleState == expectation.lifecycle, evaluationText(fmt.Sprintf("lifecycle %v", resolved.value.LifecycleState)))
}

func (expectation commandLifecycleExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolved := resolveReferenceCommandLocator(expectation.locator, reference)
	expected := evaluationText(fmt.Sprintf("lifecycle %v", expectation.lifecycle))
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, expected, resolved.detail)
	}
	return commandPropertyOutcome(resolved.subject, expected, resolved.value.LifecycleState == expectation.lifecycle, evaluationText(fmt.Sprintf("lifecycle %v", resolved.value.LifecycleState)))
}

func (expectation commandLifecycleExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveInspectionCommandLocator(expectation.locator, inspection)
	expected := evaluationText(fmt.Sprintf("lifecycle %v", expectation.lifecycle))
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, expected, resolved.detail)
	}
	return commandPropertyOutcome(resolved.subject, expected, resolved.value.LifecycleState == expectation.lifecycle, evaluationText(fmt.Sprintf("lifecycle %v", resolved.value.LifecycleState)))
}

func (commandNameExpectationValue) schemaExpectation() {}

func (commandNameExpectationValue) referenceExpectation() {}

func (commandNameExpectationValue) inspectionExpectation() {}

func (expectation commandNameExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolved := resolveSchemaCommandLocator(expectation.locator, schema)
	expected := subjectNamed(evaluationText("name"), expectation.name)
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, expected, resolved.detail)
	}
	return commandPropertyOutcome(resolved.subject, expected, resolved.value.Name == expectation.name, subjectNamed(evaluationText("name"), resolved.value.Name))
}

func (expectation commandNameExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolved := resolveReferenceCommandLocator(expectation.locator, reference)
	expected := subjectNamed(evaluationText("name"), expectation.name)
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, expected, resolved.detail)
	}
	return commandPropertyOutcome(resolved.subject, expected, resolved.value.Name == expectation.name, subjectNamed(evaluationText("name"), resolved.value.Name))
}

func (expectation commandNameExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveInspectionCommandLocator(expectation.locator, inspection)
	expected := subjectNamed(evaluationText("name"), expectation.name)
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, expected, resolved.detail)
	}
	return commandPropertyOutcome(resolved.subject, expected, resolved.value.Name == expectation.name, subjectNamed(evaluationText("name"), resolved.value.Name))
}

func (commandOriginExpectationValue) schemaExpectation() {}

func (commandOriginExpectationValue) referenceExpectation() {}

func (commandOriginExpectationValue) inspectionExpectation() {}

func (expectation commandOriginExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolved := resolveSchemaCommandLocator(expectation.locator, schema)
	expected := evaluationText(fmt.Sprintf("origin %v", expectation.origin))
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, expected, resolved.detail)
	}
	return commandPropertyOutcome(resolved.subject, expected, resolved.value.Origin == expectation.origin, evaluationText(fmt.Sprintf("origin %v", resolved.value.Origin)))
}

func (expectation commandOriginExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolved := resolveReferenceCommandLocator(expectation.locator, reference)
	expected := evaluationText(fmt.Sprintf("origin %v", expectation.origin))
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, expected, resolved.detail)
	}
	return commandPropertyOutcome(resolved.subject, expected, resolved.value.Origin == expectation.origin, evaluationText(fmt.Sprintf("origin %v", resolved.value.Origin)))
}

func (expectation commandOriginExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveInspectionCommandLocator(expectation.locator, inspection)
	expected := evaluationText(fmt.Sprintf("origin %v", expectation.origin))
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, expected, resolved.detail)
	}
	return commandPropertyOutcome(resolved.subject, expected, resolved.value.Origin == expectation.origin, evaluationText(fmt.Sprintf("origin %v", resolved.value.Origin)))
}

func (commandAvailabilityExpectationValue) inspectionExpectation() {}

func (expectation commandAvailabilityExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveInspectionCommandLocator(expectation.locator, inspection)
	expected := evaluationText(fmt.Sprintf("available=%t", expectation.available))
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, expected, resolved.detail)
	}
	return commandPropertyOutcome(resolved.subject, expected, resolved.value.Available == expectation.available, evaluationText(fmt.Sprintf("available=%t", resolved.value.Available)))
}

func (commandVisibilityExpectationValue) inspectionExpectation() {}

func (expectation commandVisibilityExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveInspectionCommandLocator(expectation.locator, inspection)
	expected := evaluationText(fmt.Sprintf("visible=%t", expectation.visible))
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, expected, resolved.detail)
	}
	return commandPropertyOutcome(resolved.subject, expected, resolved.value.Visible == expectation.visible, evaluationText(fmt.Sprintf("visible=%t", resolved.value.Visible)))
}

func (commandNamespaceExpectationValue) inspectionExpectation() {}

func (expectation commandNamespaceExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveInspectionCommandLocator(expectation.locator, inspection)
	expected := evaluationText(fmt.Sprintf("namespace %v", expectation.namespace))
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, expected, resolved.detail)
	}
	return commandPropertyOutcome(resolved.subject, expected, resolved.value.NamespaceCompleteness == expectation.namespace, evaluationText(fmt.Sprintf("namespace %v", resolved.value.NamespaceCompleteness)))
}

func (commandPathField) evaluationField() {}

func (commandIDField) evaluationField() {}

func (commandSourceDescriptionExpectation) schemaExpectation() {}

func (e commandSourceDescriptionExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolved := resolveSchemaCommandLocator(e.locator, schema)
	if resolved.status != locatorResolutionFound {
		field := descriptionField{
			value: e.expected,
		}
		return outcomeForValueStatus(resolved.status, resolved.subject, field, resolved.detail)
	}
	return compareDescription(qualifySubject(resolved.subject, evaluationText("source description")), e.expected, resolved.value.Description)
}

func (commandPresentedDescriptionExpectation) referenceExpectation() {}

func (commandPresentedDescriptionExpectation) inspectionExpectation() {}

func (e commandPresentedDescriptionExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolved := resolveReferenceCommandLocator(e.locator, reference)
	if resolved.status != locatorResolutionFound {
		field := descriptionField{
			value: e.expected,
		}
		return outcomeForValueStatus(resolved.status, resolved.subject, field, resolved.detail)
	}
	return compareDescription(qualifySubject(resolved.subject, evaluationText("presented description")), e.expected, resolved.value.Description)
}

func (e commandPresentedDescriptionExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveInspectionCommandLocator(e.locator, inspection)
	if resolved.status != locatorResolutionFound {
		field := descriptionField{
			value: e.expected,
		}
		return outcomeForValueStatus(resolved.status, resolved.subject, field, resolved.detail)
	}
	return compareDescription(qualifySubject(resolved.subject, evaluationText("presented description")), e.expected, resolved.value.Description)
}

func (commandSourceExamplesExpectation) schemaExpectation() {}

func (e commandSourceExamplesExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolved := resolveSchemaCommandLocator(e.locator, schema)
	if resolved.status != locatorResolutionFound {
		field := collectionCardinalityField{
			name:  semanticNameExamples,
			count: len(e.expected),
		}
		return outcomeForValueStatus(resolved.status, resolved.subject, field, resolved.detail)
	}
	if slices.Equal(resolved.value.Examples, e.expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(qualifySubject(resolved.subject, evaluationText("source examples")), semanticNameExamples, e.expected, resolved.value.Examples)
}

func (commandPresentedExamplesExpectation) referenceExpectation() {}

func (commandPresentedExamplesExpectation) inspectionExpectation() {}

func (e commandPresentedExamplesExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolved := resolveReferenceCommandLocator(e.locator, reference)
	if resolved.status != locatorResolutionFound {
		field := collectionCardinalityField{
			name:  semanticNameExamples,
			count: len(e.expected),
		}
		return outcomeForValueStatus(resolved.status, resolved.subject, field, resolved.detail)
	}
	if slices.Equal(resolved.value.Examples, e.expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(qualifySubject(resolved.subject, evaluationText("presented examples")), semanticNameExamples, e.expected, resolved.value.Examples)
}

func (e commandPresentedExamplesExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveInspectionCommandLocator(e.locator, inspection)
	if resolved.status != locatorResolutionFound {
		field := collectionCardinalityField{
			name:  semanticNameExamples,
			count: len(e.expected),
		}
		return outcomeForValueStatus(resolved.status, resolved.subject, field, resolved.detail)
	}
	if inspection.Truncated {
		field := collectionCardinalityField{
			name:  "exact examples",
			count: len(e.expected),
		}
		return notProvableResult(
			qualifySubject(resolved.subject, evaluationText("presented examples")),
			field,
			observed(semanticSubjectInspection, observationTruncated),
		)
	}
	if slices.Equal(resolved.value.Examples, e.expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(qualifySubject(resolved.subject, evaluationText("presented examples")), semanticNameExamples, e.expected, resolved.value.Examples)
}

func (commandSignalPoliciesExpectation) schemaExpectation() {}

func (commandSignalPoliciesExpectation) referenceExpectation() {}

func (commandSignalPoliciesExpectation) inspectionExpectation() {}

func (expectation commandSignalPoliciesExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	command := resolveSchemaCommandLocator(expectation.command, schema)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("signal policies exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("signal policies"))
	expected := make([]cli.SignalPolicyID, len(expectation.items))
	for index, state := range expectation.items {
		id, status, detail := expectedSchemaSignalPolicyID(state, schema)
		if outcome, failed := collectionResolutionFailure(label, "signal policies", index, status, detail); failed {
			return outcome
		}
		expected[index] = id
	}
	if slices.Equal(command.value.SignalPolicies, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "signal policies", expected, command.value.SignalPolicies)
}

func (expectation commandSignalPoliciesExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	command := resolveReferenceCommandLocator(expectation.command, reference)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("signal policies exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("signal policies"))
	expected := make([]cli.SignalPolicyID, len(expectation.items))
	for index, state := range expectation.items {
		id, status, detail := expectedReferenceSignalPolicyID(state, reference)
		if outcome, failed := collectionResolutionFailure(label, "signal policies", index, status, detail); failed {
			return outcome
		}
		expected[index] = id
	}
	if slices.Equal(command.value.SignalPolicies, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "signal policies", expected, command.value.SignalPolicies)
}

func (expectation commandSignalPoliciesExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	command := resolveInspectionCommandLocator(expectation.command, inspection)
	label, failure, ok := exactPolicyCollectionInspection(command, inspection, "signal policies")
	if !ok {
		return failure
	}
	expected := make([]cli.SignalPolicyID, len(expectation.items))
	for index, state := range expectation.items {
		id, status, detail := expectedInspectionSignalPolicyID(state, inspection)
		if outcome, failed := collectionResolutionFailure(label, "signal policies", index, status, detail); failed {
			return outcome
		}
		expected[index] = id
	}
	if slices.Equal(command.value.SignalPolicies, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "signal policies", expected, command.value.SignalPolicies)
}
