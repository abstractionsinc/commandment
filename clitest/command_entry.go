package clitest

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/abstractionsinc/commandment/cli"
)

// CommandEntryLocator identifies one command entry for clitest semantic expectation.
// Its zero value is invalid.
type CommandEntryLocator struct {
	_     [0]func()
	state *commandEntryLocatorState
}

// ActionEntryLocator identifies one action entry for clitest semantic expectation. Its
// zero value is invalid.
type ActionEntryLocator struct {
	_     [0]func()
	state *actionEntryLocatorState
}

// CommandShortcutDefaultLocator identifies one public shortcut default on a command entry by
// its ordered index. It preserves the exposed destination/value relationships and sensitivity
// state; the zero value is invalid.
type CommandShortcutDefaultLocator struct {
	entry CommandEntryLocator
	index int
}

// ActionEntriesLocator identifies the action-entry collection for one command and
// InvocationAction. Exact collection expectations use the authoritative Commandment order and require
// a complete relevant observation; the zero value is invalid.
type ActionEntriesLocator struct {
	_       [0]func()
	command *commandLocatorState
	action  cli.InvocationAction
}

// CommandEntryByID returns a locator for the command entry with the exact public CommandEntryID
// within the observation being evaluated.
func CommandEntryByID(id cli.CommandEntryID) CommandEntryLocator { return commandEntryIDLocator(id) }

// ActionEntryByID returns a locator for the action entry with the exact public ActionEntryID
// within the observation being evaluated.
func ActionEntryByID(id cli.ActionEntryID) ActionEntryLocator { return actionEntryIDLocator(id) }

func (CommandEntryLocator) declarationIdentityLocator() {}

func (CommandEntryLocator) diagnosticSubject() {}

func (CommandEntryLocator) noticeSubject() {}

func (CommandEntryLocator) tokenOwner() {}

// Exists returns an expectation that checks the located command entry exists. Positive existence
// may be satisfied by truthful partial output when the subject is present.
func (locator CommandEntryLocator) Exists() DeclarationExpectation {
	expectation := commandEntryExistenceExpectation{
		locator:  locator.state,
		expected: true,
	}
	return expectation
}

// DoesNotExist returns an expectation that checks the located command entry does not exist; an
// incomplete observation cannot establish absence.
func (locator CommandEntryLocator) DoesNotExist() DeclarationExpectation {
	expectation := commandEntryExistenceExpectation{
		locator:  locator.state,
		expected: false,
	}
	return expectation
}

// Kind returns an expectation that checks the located command entry has the expected public kind.
func (locator CommandEntryLocator) Kind(v cli.CommandEntryKind) DeclarationExpectation {
	value := commandEntryKindExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Hidden returns an expectation that checks the located command entry is marked hidden by the
// public Commandment observation.
func (locator CommandEntryLocator) Hidden() DeclarationExpectation {
	value := commandEntryHiddenExpectationValue{
		locator: locator.state,
	}
	return value
}

// Lifecycle returns an expectation that checks the located command entry has the expected
// effective lifecycle state; this is distinct from release-lifecycle metadata.
func (locator CommandEntryLocator) Lifecycle(v cli.LifecycleState) DeclarationExpectation {
	value := commandEntryLifecycleExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// ReplacedBy returns an expectation that checks the located command entry replacement points to
// the exact declaration identified by v, using public declaration identity rather than
// presentation text.
func (locator CommandEntryLocator) ReplacedBy(v DeclarationIdentityLocator) DeclarationExpectation {
	value := commandEntryReplacementExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Targets returns an expectation that checks the located command entry targets the exact command
// identified by v.
func (locator CommandEntryLocator) Targets(v CommandLocator) DeclarationExpectation {
	value := commandEntryTargetExpectationValue{
		locator: locator.state,
		target:  v.state,
	}
	return value
}

// Available returns an expectation that checks the located command entry is available in the
// applicable public observation.
func (locator CommandEntryLocator) Available() EffectiveDeclarationExpectation {
	expectation := commandEntryAvailabilityExpectation{
		locator:  locator.state,
		expected: true,
	}
	return expectation
}

// Unavailable returns an expectation that checks the located command entry is unavailable in the
// applicable public observation.
func (locator CommandEntryLocator) Unavailable() EffectiveDeclarationExpectation {
	expectation := commandEntryAvailabilityExpectation{
		locator:  locator.state,
		expected: false,
	}
	return expectation
}

// Visible returns an expectation that checks the located command entry is visible in the
// applicable public observation.
func (locator CommandEntryLocator) Visible() EffectiveDeclarationExpectation {
	expectation := commandEntryVisibilityExpectation{
		locator:  locator.state,
		expected: true,
	}
	return expectation
}

// NotVisible returns an expectation that checks the located command entry is not visible in the
// applicable public observation.
func (locator CommandEntryLocator) NotVisible() EffectiveDeclarationExpectation {
	expectation := commandEntryVisibilityExpectation{
		locator:  locator.state,
		expected: false,
	}
	return expectation
}

// ShortcutDefaultCount returns a reusable declaration expectation that checks the shortcut
// default count exposed by the located command entry.
func (locator CommandEntryLocator) ShortcutDefaultCount(expected int) DeclarationExpectation {
	expectation := commandEntryShortcutDefaultCountExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// ShortcutDefaultAt returns a locator for the shortcut default at the supplied public ordered
// index of the located command entry. It does not search for or silently choose a first match.
func (locator CommandEntryLocator) ShortcutDefaultAt(index int) CommandShortcutDefaultLocator {
	commandShortcutDefaultLocator := CommandShortcutDefaultLocator{
		entry: locator,
		index: index,
	}
	return commandShortcutDefaultLocator
}

// SourceDescription returns a reusable Schema expectation that checks the public source
// description exposed by the located command entry.
func (locator CommandEntryLocator) SourceDescription(expected cli.Description) SchemaExpectation {
	expectation := commandEntrySourceDescriptionExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// Origin returns an expectation that checks the located command entry has the
// framework-established DeclarationOrigin expected.
func (locator CommandEntryLocator) Origin(expected cli.DeclarationOrigin) DeclarationExpectation {
	expectation := commandEntryOriginExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// ReleaseLifecycle returns an expectation that checks the located command entry has the exact
// public release-lifecycle metadata expected.
func (locator CommandEntryLocator) ReleaseLifecycle(expected cli.ReleaseLifecycle) DeclarationExpectation {
	expectation := commandEntryReleaseLifecycleExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// NotReplaced returns an expectation that checks the located command entry has no public
// declaration-replacement relationship. Incomplete observation cannot manufacture replacement
// absence.
func (locator CommandEntryLocator) NotReplaced() DeclarationExpectation {
	expectation := commandEntryReplacementAbsenceExpectation{
		locator: locator.state,
	}
	return expectation
}

// ReplacementPresentation returns an expectation that checks the located command entry replacement
// carries the expected public presentation text.
func (locator CommandEntryLocator) ReplacementPresentation(expected string) DeclarationExpectation {
	expectation := commandEntryReplacementPresentationExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// ReplacementTargetDeprecatedDuringMigration returns an expectation that checks the located
// command entry replacement carries the expected migration-deprecation marker.
func (locator CommandEntryLocator) ReplacementTargetDeprecatedDuringMigration(expected bool) DeclarationExpectation {
	expectation := commandEntryReplacementTargetDeprecatedExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

func (ActionEntryLocator) declarationIdentityLocator() {}

func (ActionEntryLocator) diagnosticSubject() {}

func (ActionEntryLocator) noticeSubject() {}

func (ActionEntryLocator) tokenOwner() {}

// Exists returns an expectation that checks the located action entry exists. Positive existence
// may be satisfied by truthful partial output when the subject is present.
func (locator ActionEntryLocator) Exists() DeclarationExpectation {
	expectation := actionEntryExistenceExpectation{
		locator:  locator.state,
		expected: true,
	}
	return expectation
}

// DoesNotExist returns an expectation that checks the located action entry does not exist; an
// incomplete observation cannot establish absence.
func (locator ActionEntryLocator) DoesNotExist() DeclarationExpectation {
	expectation := actionEntryExistenceExpectation{
		locator:  locator.state,
		expected: false,
	}
	return expectation
}

// Kind returns an expectation that checks the located action entry has the expected public kind.
func (locator ActionEntryLocator) Kind(v cli.ActionEntryKind) DeclarationExpectation {
	value := actionEntryKindExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Hidden returns an expectation that checks the located action entry is marked hidden by the
// public Commandment observation.
func (locator ActionEntryLocator) Hidden() DeclarationExpectation {
	value := actionEntryHiddenExpectationValue{
		locator: locator.state,
	}
	return value
}

// Lifecycle returns an expectation that checks the located action entry has the expected effective
// lifecycle state; this is distinct from release-lifecycle metadata.
func (locator ActionEntryLocator) Lifecycle(v cli.LifecycleState) DeclarationExpectation {
	value := actionEntryLifecycleExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// ReplacedBy returns an expectation that checks the located action entry replacement points to the
// exact declaration identified by v, using public declaration identity rather than presentation
// text.
func (locator ActionEntryLocator) ReplacedBy(v DeclarationIdentityLocator) DeclarationExpectation {
	value := actionEntryReplacementExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Available returns an expectation that checks the located action entry is available in the
// applicable public observation.
func (locator ActionEntryLocator) Available() EffectiveDeclarationExpectation {
	expectation := actionEntryAvailabilityExpectation{
		locator:  locator.state,
		expected: true,
	}
	return expectation
}

// Unavailable returns an expectation that checks the located action entry is unavailable in the
// applicable public observation.
func (locator ActionEntryLocator) Unavailable() EffectiveDeclarationExpectation {
	expectation := actionEntryAvailabilityExpectation{
		locator:  locator.state,
		expected: false,
	}
	return expectation
}

// Visible returns an expectation that checks the located action entry is visible in the applicable
// public observation.
func (locator ActionEntryLocator) Visible() EffectiveDeclarationExpectation {
	expectation := actionEntryVisibilityExpectation{
		locator:  locator.state,
		expected: true,
	}
	return expectation
}

// NotVisible returns an expectation that checks the located action entry is not visible in the
// applicable public observation.
func (locator ActionEntryLocator) NotVisible() EffectiveDeclarationExpectation {
	expectation := actionEntryVisibilityExpectation{
		locator:  locator.state,
		expected: false,
	}
	return expectation
}

// Placement returns a reusable declaration expectation that checks the public placement exposed
// by the located action entry.
func (locator ActionEntryLocator) Placement(expected cli.FlagPlacement) DeclarationExpectation {
	expectation := actionEntryPlacementExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// HasSourceShortDescription returns a Schema expectation that checks the action entry's source
// short description is present in the static Schema observation.
func (locator ActionEntryLocator) HasSourceShortDescription() SchemaExpectation {
	expectation := actionEntrySourceShortDescriptionExpectation{
		locator: locator.state,
	}
	return expectation
}

// SourceDescription returns a Schema expectation that checks the action entry's exact source
// Description in the static Schema observation.
func (locator ActionEntryLocator) SourceDescription(expected cli.Description) SchemaExpectation {
	expectation := actionEntrySourceDescriptionExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// Origin returns an expectation that checks the located action entry has the framework-established
// DeclarationOrigin expected.
func (locator ActionEntryLocator) Origin(expected cli.DeclarationOrigin) DeclarationExpectation {
	expectation := actionEntryOriginExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// ReleaseLifecycle returns an expectation that checks the located action entry has the exact
// public release-lifecycle metadata expected.
func (locator ActionEntryLocator) ReleaseLifecycle(expected cli.ReleaseLifecycle) DeclarationExpectation {
	expectation := actionEntryReleaseLifecycleExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// NotReplaced returns an expectation that checks the located action entry has no public
// declaration-replacement relationship. Incomplete observation cannot manufacture replacement
// absence.
func (locator ActionEntryLocator) NotReplaced() DeclarationExpectation {
	expectation := actionEntryReplacementAbsenceExpectation{
		locator: locator.state,
	}
	return expectation
}

// ReplacementPresentation returns an expectation that checks the located action entry replacement
// carries the expected public presentation text.
func (locator ActionEntryLocator) ReplacementPresentation(expected string) DeclarationExpectation {
	expectation := actionEntryReplacementPresentationExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// ReplacementTargetDeprecatedDuringMigration returns an expectation that checks the located action
// entry replacement carries the expected migration-deprecation marker.
func (locator ActionEntryLocator) ReplacementTargetDeprecatedDuringMigration(expected bool) DeclarationExpectation {
	expectation := actionEntryReplacementTargetDeprecatedExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// CanonicalEntry returns a locator for the located command's canonical command entry.
func (l CommandLocator) CanonicalEntry() CommandEntryLocator {
	return commandCanonicalEntryLocator(l)
}

// Alias returns a locator for the uniquely resolved alias entry named name under the located
// command.
func (l CommandLocator) Alias(name string) CommandEntryLocator {
	return commandAliasLocator(l, name)
}

// Shortcut returns a locator for the uniquely resolved shortcut entry named name under the located
// command.
func (l CommandLocator) Shortcut(name string) CommandEntryLocator {
	return commandShortcutLocator(l, name)
}

// EntryAt returns a locator for the command entry at index in the located command's authoritative
// public entry order.
func (l CommandLocator) EntryAt(index int) CommandEntryLocator {
	return commandEntryAtLocator(l, index)
}

// ActionEntryAt returns a locator for the action entry at index in the located command's
// authoritative public action-entry order.
func (l CommandLocator) ActionEntryAt(index int) ActionEntryLocator {
	return commandActionEntryAtLocator(l, index)
}

// EntriesExactly returns an expectation that checks the located command's command entries are
// exactly the supplied locators in authoritative public order. Exactness is not inferred from an
// incomplete observation.
func (l CommandLocator) EntriesExactly(values ...CommandEntryLocator) DeclarationExpectation {
	expectation := commandEntriesExpectation{
		command: l.state,
		items:   commandEntryLocatorStates(values),
	}
	return expectation
}

// ActionEntriesExactly returns an expectation that checks the located command's action entries are
// exactly the supplied locators in authoritative public order. Exactness is not inferred from an
// incomplete observation.
func (l CommandLocator) ActionEntriesExactly(values ...ActionEntryLocator) DeclarationExpectation {
	expectation := commandActionEntriesExpectation{
		command: l.state,
		items:   actionEntryLocatorStates(values),
	}
	return expectation
}

// Destination returns a declaration expectation that checks the shortcut default's destination
// relationship resolves to the same public value identity as the expected locator.
func (locator CommandShortcutDefaultLocator) Destination(expected ValueLocator) DeclarationExpectation {
	expectation := shortcutDefaultExpectation{
		locator:     locator,
		property:    shortcutDefaultExpectationDestination,
		destination: expected.state,
	}
	return expectation
}

// Spelling returns a reusable declaration expectation that checks the public spelling exposed
// by the located public subject.
func (locator CommandShortcutDefaultLocator) Spelling(expected string) DeclarationExpectation {
	expectation := shortcutDefaultExpectation{
		locator:  locator,
		property: shortcutDefaultExpectationSpelling,
		spelling: expected,
	}
	return expectation
}

// Sensitive returns a reusable declaration expectation that checks the located public subject
// is marked sensitive. Expectation failures preserve Commandment redaction boundaries.
func (locator CommandShortcutDefaultLocator) Sensitive() DeclarationExpectation {
	expectation := shortcutDefaultExpectation{
		locator:   locator,
		property:  shortcutDefaultExpectationSensitive,
		sensitive: true,
	}
	return expectation
}

// ValueCount returns a reusable declaration expectation that checks the value count exposed by
// the located public subject.
func (locator CommandShortcutDefaultLocator) ValueCount(expected int) DeclarationExpectation {
	expectation := shortcutDefaultExpectation{
		locator:  locator,
		property: shortcutDefaultExpectationValueCount,
		count:    expected,
	}
	return expectation
}

// ValueAt returns a locator for the value at the supplied public ordered index of the located
// public subject. It does not search for or silently choose a first match.
func (locator CommandShortcutDefaultLocator) ValueAt(index int) DeclarationValueLiteralLocator {
	state := &declarationValueLiteralLocatorState{
		mode:         declarationValueLiteralLocatorShortcutDefaultValue,
		entry:        locator.entry.state,
		defaultIndex: locator.index,
		index:        index,
	}
	declarationValueLiteralLocator := DeclarationValueLiteralLocator{
		state: state,
	}
	return declarationValueLiteralLocator
}

// Long returns an ActionEntryLocator for the entry whose public long-form coordinate matches
// expected. Ambiguous coordinates fail expectation resolution; no first match is selected.
func (locator ActionEntriesLocator) Long(expected string) ActionEntryLocator {
	return actionEntriesScopedLocator(locator, actionEntryLocatorScopedLong, expected, 0)
}

// Short returns an ActionEntryLocator for the entry whose public short-form coordinate matches
// expected. Ambiguous coordinates fail expectation resolution; no first match is selected.
func (locator ActionEntriesLocator) Short(expected rune) ActionEntryLocator {
	return actionEntriesScopedLocator(locator, actionEntryLocatorScopedShort, "", expected)
}

// Literal returns an ActionEntryLocator for the entry whose public literal coordinate matches
// expected. Ambiguous coordinates fail expectation resolution; no first match is selected.
func (locator ActionEntriesLocator) Literal(expected string) ActionEntryLocator {
	return actionEntriesScopedLocator(locator, actionEntryLocatorScopedLiteral, expected, 0)
}

// Command returns an ActionEntryLocator for the command entry whose public command-name
// coordinate matches name. Ambiguous coordinates fail expectation resolution; no first match is
// selected.
func (locator ActionEntriesLocator) Command(name string) ActionEntryLocator {
	return actionEntriesScopedLocator(locator, actionEntryLocatorScopedCommand, name, 0)
}

// Exactly returns a reusable declaration expectation that checks complete action-entry
// membership and authoritative Commandment order. Exactness is not provable from an incomplete
// relevant observation.
func (locator ActionEntriesLocator) Exactly(entries ...ActionEntryLocator) DeclarationExpectation {
	expected := make([]*actionEntryLocatorState, len(entries))
	for index, entry := range entries {
		expected[index] = entry.state
	}
	expectation := actionEntriesExactlyExpectation{
		scope:   locator,
		entries: expected,
	}
	return expectation
}

type commandActionEntriesExpectation struct {
	command *commandLocatorState
	items   []*actionEntryLocatorState
}

type commandEntryLocatorMode uint8

const (
	commandEntryLocatorInvalid commandEntryLocatorMode = iota
	commandEntryLocatorID
	commandEntryLocatorCanonical
	commandEntryLocatorAlias
	commandEntryLocatorShortcut
	commandEntryLocatorIndex
)

type commandEntryLocatorState struct {
	mode commandEntryLocatorMode
	id   cli.CommandEntryID

	command *commandLocatorState
	name    string
	index   int
}

type actionEntryLocatorMode uint8

const (
	actionEntryLocatorInvalid actionEntryLocatorMode = iota
	actionEntryLocatorID
	actionEntryLocatorCommandIndex
	actionEntryLocatorScopedLong
	actionEntryLocatorScopedShort
	actionEntryLocatorScopedLiteral
	actionEntryLocatorScopedCommand
)

type actionEntryLocatorState struct {
	mode actionEntryLocatorMode
	id   cli.ActionEntryID

	command *commandLocatorState
	action  cli.InvocationAction
	text    string
	short   rune
	index   int
}

type commandEntryAvailabilityExpectation struct {
	locator  *commandEntryLocatorState
	expected bool
}

type commandEntryVisibilityExpectation struct {
	locator  *commandEntryLocatorState
	expected bool
}

type actionEntryAvailabilityExpectation struct {
	locator  *actionEntryLocatorState
	expected bool
}

type actionEntryVisibilityExpectation struct {
	locator  *actionEntryLocatorState
	expected bool
}

type commandEntrySourceDescriptionExpectation struct {
	locator  *commandEntryLocatorState
	expected cli.Description
}

type actionEntriesExactlyExpectation struct {
	scope   ActionEntriesLocator
	entries []*actionEntryLocatorState
}

type actionEntryPlacementExpectation struct {
	locator  *actionEntryLocatorState
	expected cli.FlagPlacement
}

type actionEntrySourceShortDescriptionExpectation struct{ locator *actionEntryLocatorState }

type actionEntrySourceDescriptionExpectation struct {
	locator  *actionEntryLocatorState
	expected cli.Description
}

type commandEntryOriginExpectation struct {
	locator  *commandEntryLocatorState
	expected cli.DeclarationOrigin
}

type commandEntryReleaseLifecycleExpectation struct {
	locator  *commandEntryLocatorState
	expected cli.ReleaseLifecycle
}

type commandEntryReplacementAbsenceExpectation struct{ locator *commandEntryLocatorState }

type commandEntryReplacementPresentationExpectation struct {
	locator  *commandEntryLocatorState
	expected string
}

type commandEntryReplacementTargetDeprecatedExpectation struct {
	locator  *commandEntryLocatorState
	expected bool
}

type actionEntryOriginExpectation struct {
	locator  *actionEntryLocatorState
	expected cli.DeclarationOrigin
}

type actionEntryReleaseLifecycleExpectation struct {
	locator  *actionEntryLocatorState
	expected cli.ReleaseLifecycle
}

type actionEntryReplacementAbsenceExpectation struct{ locator *actionEntryLocatorState }

type actionEntryReplacementPresentationExpectation struct {
	locator  *actionEntryLocatorState
	expected string
}

type actionEntryReplacementTargetDeprecatedExpectation struct {
	locator  *actionEntryLocatorState
	expected bool
}

type commandEntryExistenceExpectation struct {
	locator  *commandEntryLocatorState
	expected bool
}

type actionEntryExistenceExpectation struct {
	locator  *actionEntryLocatorState
	expected bool
}

type commandEntryKindExpectationValue struct {
	locator  *commandEntryLocatorState
	expected cli.CommandEntryKind
}

type commandEntryHiddenExpectationValue struct{ locator *commandEntryLocatorState }

type commandEntryLifecycleExpectationValue struct {
	locator  *commandEntryLocatorState
	expected cli.LifecycleState
}

type commandEntryReplacementExpectationValue struct {
	locator  *commandEntryLocatorState
	expected DeclarationIdentityLocator
}

type commandEntryTargetExpectationValue struct {
	locator *commandEntryLocatorState
	target  *commandLocatorState
}

type actionEntryKindExpectationValue struct {
	locator  *actionEntryLocatorState
	expected cli.ActionEntryKind
}

type actionEntryHiddenExpectationValue struct{ locator *actionEntryLocatorState }

type actionEntryLifecycleExpectationValue struct {
	locator  *actionEntryLocatorState
	expected cli.LifecycleState
}

type actionEntryReplacementExpectationValue struct {
	locator  *actionEntryLocatorState
	expected DeclarationIdentityLocator
}

type commandEntryShortcutDefaultCountExpectation struct {
	locator  *commandEntryLocatorState
	expected int
}

type shortcutDefaultExpectationProperty uint8

const (
	shortcutDefaultExpectationInvalid shortcutDefaultExpectationProperty = iota
	shortcutDefaultExpectationSpelling
	shortcutDefaultExpectationSensitive
	shortcutDefaultExpectationValueCount
	shortcutDefaultExpectationDestination
)

type shortcutDefaultExpectation struct {
	locator     CommandShortcutDefaultLocator
	property    shortcutDefaultExpectationProperty
	spelling    string
	sensitive   bool
	count       int
	destination *valueLocatorState
}

func commandEntryLocatorStates(values []CommandEntryLocator) []*commandEntryLocatorState {
	states := make([]*commandEntryLocatorState, len(values))
	for index, value := range values {
		states[index] = value.state
	}
	return states
}

func actionEntryLocatorStates(values []ActionEntryLocator) []*actionEntryLocatorState {
	states := make([]*actionEntryLocatorState, len(values))
	for index, value := range values {
		states[index] = value.state
	}
	return states
}

func commandEntryAtLocator(parent CommandLocator, index int) CommandEntryLocator {
	state := &commandEntryLocatorState{
		mode:    commandEntryLocatorIndex,
		command: parent.state,
		index:   index,
	}
	locator := CommandEntryLocator{
		state: state,
	}
	return locator
}

func commandActionEntryAtLocator(parent CommandLocator, index int) ActionEntryLocator {
	state := &actionEntryLocatorState{
		mode:    actionEntryLocatorCommandIndex,
		command: parent.state,
		index:   index,
	}
	locator := ActionEntryLocator{
		state: state,
	}
	return locator
}

func commandEntryLocatorLabel(state *commandEntryLocatorState) evaluationField {
	if state == nil {
		return semanticSubjectCommandEntry
	}
	switch state.mode {
	case commandEntryLocatorID:
		return evaluationText(fmt.Sprintf("command entry id %d", state.id))
	case commandEntryLocatorCanonical:
		return qualifySubject(commandLocatorLabel(state.command), evaluationText("canonical entry"))
	case commandEntryLocatorAlias:
		return qualifySubject(commandLocatorLabel(state.command), subjectNamed(evaluationText("alias"), state.name))
	case commandEntryLocatorShortcut:
		return qualifySubject(commandLocatorLabel(state.command), subjectNamed(evaluationText("shortcut"), state.name))
	case commandEntryLocatorIndex:
		return qualifySubject(commandLocatorLabel(state.command), evaluationText(fmt.Sprintf("command entry %d", state.index)))
	default:
		return semanticSubjectCommandEntry
	}
}

func resolveSchemaCommandEntryLocator(state *commandEntryLocatorState, schema *cli.Schema) locatorResolution[cli.CommandEntrySchema] {
	var resolution locatorResolution[cli.CommandEntrySchema]
	label := commandEntryLocatorLabel(state)
	if schema == nil || state == nil || state.mode == commandEntryLocatorInvalid {
		resolution = locatorResolution[cli.CommandEntrySchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCommandEntry, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == commandEntryLocatorID {
		entry, ok := schema.CommandEntries[state.id]
		if !ok {
			resolution = locatorResolution[cli.CommandEntrySchema]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectCommandEntry, observationAbsent),
			}
			return resolution
		}
		resolution = locatorResolution[cli.CommandEntrySchema]{
			status:  locatorResolutionFound,
			value:   entry,
			subject: label,
		}
		return resolution
	}
	parent := resolveSchemaCommandLocator(state.command, schema)
	if parent.status != locatorResolutionFound {
		resolution = locatorResolution[cli.CommandEntrySchema]{
			status:  parent.status,
			subject: label,
			detail:  parent.detail,
		}
		return resolution
	}
	id, status, detail := selectSchemaCommandEntry(state, parent.value.CanonicalEntry, parent.value.CommandEntries, schema, true)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.CommandEntrySchema]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	entry, ok := schema.CommandEntries[id]
	if !ok {
		resolution = locatorResolution[cli.CommandEntrySchema]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("command-entry relationship points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.CommandEntrySchema]{
		status:  locatorResolutionFound,
		value:   entry,
		subject: label,
	}
	return resolution
}

func resolveReferenceCommandEntryLocator(state *commandEntryLocatorState, reference cli.Reference) locatorResolution[cli.CommandEntryReference] {
	var resolution locatorResolution[cli.CommandEntryReference]
	label := commandEntryLocatorLabel(state)
	if state == nil || state.mode == commandEntryLocatorInvalid {
		resolution = locatorResolution[cli.CommandEntryReference]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCommandEntry, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == commandEntryLocatorID {
		entry, ok := reference.CommandEntries[state.id]
		if !ok {
			resolution = locatorResolution[cli.CommandEntryReference]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectCommandEntry, observationAbsent),
			}
			return resolution
		}
		resolution = locatorResolution[cli.CommandEntryReference]{
			status:  locatorResolutionFound,
			value:   entry,
			subject: label,
		}
		return resolution
	}
	parent := resolveReferenceCommandLocator(state.command, reference)
	if parent.status != locatorResolutionFound {
		resolution = locatorResolution[cli.CommandEntryReference]{
			status:  parent.status,
			subject: label,
			detail:  parent.detail,
		}
		return resolution
	}
	id, status, detail := selectReferenceCommandEntry(state, parent.value.CanonicalEntry, parent.value.CommandEntries, reference, true)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.CommandEntryReference]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	entry, ok := reference.CommandEntries[id]
	if !ok {
		resolution = locatorResolution[cli.CommandEntryReference]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("command-entry relationship points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.CommandEntryReference]{
		status:  locatorResolutionFound,
		value:   entry,
		subject: label,
	}
	return resolution
}

func resolveInspectionCommandEntryLocator(state *commandEntryLocatorState, inspection cli.Inspection) locatorResolution[cli.CommandEntryInspection] {
	var resolution locatorResolution[cli.CommandEntryInspection]
	label := commandEntryLocatorLabel(state)
	if state == nil || state.mode == commandEntryLocatorInvalid {
		resolution = locatorResolution[cli.CommandEntryInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCommandEntry, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == commandEntryLocatorID {
		entry, ok := inspection.CommandEntries[state.id]
		if ok {
			resolution = locatorResolution[cli.CommandEntryInspection]{
				status:  locatorResolutionFound,
				value:   entry,
				subject: label,
			}
			return resolution
		}
		if inspection.Truncated {
			resolution = locatorResolution[cli.CommandEntryInspection]{
				status:  locatorResolutionNotProvable,
				subject: label,
				detail:  observed(semanticSubjectInspection, observationTruncated),
			}
			return resolution
		}
		resolution = locatorResolution[cli.CommandEntryInspection]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectCommandEntry, observationAbsent),
		}
		return resolution
	}
	parent := resolveInspectionCommandLocator(state.command, inspection)
	if parent.status != locatorResolutionFound {
		resolution = locatorResolution[cli.CommandEntryInspection]{
			status:  parent.status,
			subject: label,
			detail:  parent.detail,
		}
		return resolution
	}
	complete := parent.value.NamespaceCompleteness == cli.CommandNamespaceComplete
	id, status, detail := selectInspectionCommandEntry(state, parent.value.CanonicalEntry, parent.value.CommandEntries, inspection, complete)
	if status == locatorResolutionAbsent && inspection.Truncated {
		status = locatorResolutionNotProvable
		detail = observed(semanticSubjectInspection, observationTruncated)
	}
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.CommandEntryInspection]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	entry, ok := inspection.CommandEntries[id]
	if !ok {
		resolution = locatorResolution[cli.CommandEntryInspection]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("command-entry relationship points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.CommandEntryInspection]{
		status:  locatorResolutionFound,
		value:   entry,
		subject: label,
	}
	return resolution
}

//nolint:dupl // Schema resolves command-entry relationships directly instead of projecting through artifact callbacks.
func selectSchemaCommandEntry(state *commandEntryLocatorState, canonical cli.CommandEntryID, ids []cli.CommandEntryID, schema *cli.Schema, complete bool) (cli.CommandEntryID, locatorResolutionStatus, evaluationField) {
	switch state.mode {
	case commandEntryLocatorCanonical:
		if canonical == 0 {
			if complete {
				return 0, locatorResolutionAbsent, observed(semanticSubjectCommandEntry, observationAbsent)
			}
			return 0, locatorResolutionNotProvable, evaluationText("command namespace is incomplete")
		}
		return canonical, locatorResolutionFound, evaluationText("")
	case commandEntryLocatorIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectCommandEntry, observationInvalidLocator)
		}
		if state.index < len(ids) {
			return ids[state.index], locatorResolutionFound, evaluationText("")
		}
		if complete {
			return 0, locatorResolutionAbsent, observed(semanticSubjectCommandEntry, observationAbsent)
		}
		return 0, locatorResolutionNotProvable, evaluationText("command namespace is incomplete")
	case commandEntryLocatorAlias, commandEntryLocatorShortcut:
		want := cli.CommandEntryAlias
		if state.mode == commandEntryLocatorShortcut {
			want = cli.CommandEntryShortcut
		}
		matches := make([]cli.CommandEntryID, 0, 1)
		missingDetail := false
		for _, id := range ids {
			entry, ok := schema.CommandEntries[id]
			if !ok {
				missingDetail = true
				continue
			}
			if entry.Kind == want && entry.Name == state.name {
				matches = append(matches, id)
			}
		}
		slices.Sort(matches)
		if len(matches) == 1 {
			return matches[0], locatorResolutionFound, evaluationText("")
		}
		if len(matches) > 1 {
			field := collectionCardinalityField{
				name:  commandEntryAmbiguityDetail,
				count: len(matches),
			}
			return 0, locatorResolutionAmbiguous, field
		}
		if missingDetail || !complete {
			return 0, locatorResolutionNotProvable, evaluationText("command-entry lookup domain is incomplete")
		}
		return 0, locatorResolutionAbsent, observed(semanticSubjectCommandEntry, observationAbsent)
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectCommandEntry, observationInvalidLocator)
	}
}

//nolint:dupl // Separate public artifacts resolve their own command-entry relationships directly.
func selectReferenceCommandEntry(state *commandEntryLocatorState, canonical cli.CommandEntryID, ids []cli.CommandEntryID, reference cli.Reference, complete bool) (cli.CommandEntryID, locatorResolutionStatus, evaluationField) {
	switch state.mode {
	case commandEntryLocatorCanonical:
		if canonical == 0 {
			if complete {
				return 0, locatorResolutionAbsent, observed(semanticSubjectCommandEntry, observationAbsent)
			}
			return 0, locatorResolutionNotProvable, evaluationText("command namespace is incomplete")
		}
		return canonical, locatorResolutionFound, evaluationText("")
	case commandEntryLocatorIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectCommandEntry, observationInvalidLocator)
		}
		if state.index < len(ids) {
			return ids[state.index], locatorResolutionFound, evaluationText("")
		}
		if complete {
			return 0, locatorResolutionAbsent, observed(semanticSubjectCommandEntry, observationAbsent)
		}
		return 0, locatorResolutionNotProvable, evaluationText("command namespace is incomplete")
	case commandEntryLocatorAlias, commandEntryLocatorShortcut:
		want := cli.CommandEntryAlias
		if state.mode == commandEntryLocatorShortcut {
			want = cli.CommandEntryShortcut
		}
		matches := make([]cli.CommandEntryID, 0, 1)
		missingDetail := false
		for _, id := range ids {
			entry, ok := reference.CommandEntries[id]
			if !ok {
				missingDetail = true
				continue
			}
			if entry.Kind == want && entry.Name == state.name {
				matches = append(matches, id)
			}
		}
		slices.Sort(matches)
		if len(matches) == 1 {
			return matches[0], locatorResolutionFound, evaluationText("")
		}
		if len(matches) > 1 {
			field := collectionCardinalityField{
				name:  commandEntryAmbiguityDetail,
				count: len(matches),
			}
			return 0, locatorResolutionAmbiguous, field
		}
		if missingDetail || !complete {
			return 0, locatorResolutionNotProvable, evaluationText("command-entry lookup domain is incomplete")
		}
		return 0, locatorResolutionAbsent, observed(semanticSubjectCommandEntry, observationAbsent)
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectCommandEntry, observationInvalidLocator)
	}
}

//nolint:dupl // Separate public artifacts resolve their own command-entry relationships directly.
func selectInspectionCommandEntry(state *commandEntryLocatorState, canonical cli.CommandEntryID, ids []cli.CommandEntryID, inspection cli.Inspection, complete bool) (cli.CommandEntryID, locatorResolutionStatus, evaluationField) {
	switch state.mode {
	case commandEntryLocatorCanonical:
		if canonical == 0 {
			if complete {
				return 0, locatorResolutionAbsent, observed(semanticSubjectCommandEntry, observationAbsent)
			}
			return 0, locatorResolutionNotProvable, evaluationText("command namespace is incomplete")
		}
		return canonical, locatorResolutionFound, evaluationText("")
	case commandEntryLocatorIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectCommandEntry, observationInvalidLocator)
		}
		if state.index < len(ids) {
			return ids[state.index], locatorResolutionFound, evaluationText("")
		}
		if complete {
			return 0, locatorResolutionAbsent, observed(semanticSubjectCommandEntry, observationAbsent)
		}
		return 0, locatorResolutionNotProvable, evaluationText("command namespace is incomplete")
	case commandEntryLocatorAlias, commandEntryLocatorShortcut:
		want := cli.CommandEntryAlias
		if state.mode == commandEntryLocatorShortcut {
			want = cli.CommandEntryShortcut
		}
		matches := make([]cli.CommandEntryID, 0, 1)
		missingDetail := false
		for _, id := range ids {
			entry, ok := inspection.CommandEntries[id]
			if !ok {
				missingDetail = true
				continue
			}
			if entry.Kind == want && entry.Name == state.name {
				matches = append(matches, id)
			}
		}
		slices.Sort(matches)
		if len(matches) == 1 {
			return matches[0], locatorResolutionFound, evaluationText("")
		}
		if len(matches) > 1 {
			field := collectionCardinalityField{
				name:  commandEntryAmbiguityDetail,
				count: len(matches),
			}
			return 0, locatorResolutionAmbiguous, field
		}
		if missingDetail || !complete {
			return 0, locatorResolutionNotProvable, evaluationText("command-entry lookup domain is incomplete")
		}
		return 0, locatorResolutionAbsent, observed(semanticSubjectCommandEntry, observationAbsent)
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectCommandEntry, observationInvalidLocator)
	}
}

func actionEntryLocatorLabel(state *actionEntryLocatorState) evaluationField {
	if state == nil {
		return semanticSubjectActionEntry
	}
	switch state.mode {
	case actionEntryLocatorID:
		return evaluationText(fmt.Sprintf("action entry id %d", state.id))
	case actionEntryLocatorCommandIndex:
		return qualifySubject(commandLocatorLabel(state.command), evaluationText(fmt.Sprintf("action entry %d", state.index)))
	case actionEntryLocatorScopedLong:
		return qualifySubject(commandLocatorLabel(state.command), qualifySubject(evaluationText(fmt.Sprintf("action %v", state.action)), subjectNamed(evaluationText("--"), state.text)))
	case actionEntryLocatorScopedShort:
		return qualifySubject(commandLocatorLabel(state.command), evaluationText(fmt.Sprintf("action %v -%c", state.action, state.short)))
	case actionEntryLocatorScopedLiteral:
		return qualifySubject(commandLocatorLabel(state.command), qualifySubject(evaluationText(fmt.Sprintf("action %v", state.action)), subjectNamed(evaluationText("literal"), state.text)))
	case actionEntryLocatorScopedCommand:
		return qualifySubject(commandLocatorLabel(state.command), qualifySubject(evaluationText(fmt.Sprintf("action %v", state.action)), subjectNamed(evaluationText("command"), state.text)))
	default:
		return semanticSubjectActionEntry
	}
}

func actionEntryMatches(state *actionEntryLocatorState, kind cli.ActionEntryKind, action cli.InvocationAction, long string, short rune, literal, name string) bool {
	if state == nil || action != state.action {
		return false
	}
	switch state.mode {
	case actionEntryLocatorScopedLong:
		return kind == cli.ActionEntryFlag && long == state.text && short == 0 && literal == ""
	case actionEntryLocatorScopedShort:
		return kind == cli.ActionEntryFlag && short == state.short && long == "" && literal == ""
	case actionEntryLocatorScopedLiteral:
		return kind == cli.ActionEntryFlag && literal == state.text && long == "" && short == 0
	case actionEntryLocatorScopedCommand:
		return kind == cli.ActionEntryCommand && name == state.text
	default:
		return false
	}
}

//nolint:dupl // Schema resolves action-entry relationships directly instead of projecting through artifact callbacks.
func resolveSchemaActionEntryLocator(state *actionEntryLocatorState, schema *cli.Schema) locatorResolution[cli.ActionEntrySchema] {
	var resolution locatorResolution[cli.ActionEntrySchema]
	label := actionEntryLocatorLabel(state)
	if schema == nil || state == nil || state.mode == actionEntryLocatorInvalid {
		resolution = locatorResolution[cli.ActionEntrySchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectActionEntry, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == actionEntryLocatorID {
		entry, ok := schema.ActionEntries[state.id]
		if !ok {
			resolution = locatorResolution[cli.ActionEntrySchema]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectActionEntry, observationAbsent),
			}
			return resolution
		}
		resolution = locatorResolution[cli.ActionEntrySchema]{
			status:  locatorResolutionFound,
			value:   entry,
			subject: label,
		}
		return resolution
	}
	parent := resolveSchemaCommandLocator(state.command, schema)
	if parent.status != locatorResolutionFound {
		resolution = locatorResolution[cli.ActionEntrySchema]{
			status:  parent.status,
			subject: label,
			detail:  parent.detail,
		}
		return resolution
	}
	id, status, detail := selectSchemaActionEntry(state, parent.value.ActionEntries, schema, true)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.ActionEntrySchema]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	entry, ok := schema.ActionEntries[id]
	if !ok {
		resolution = locatorResolution[cli.ActionEntrySchema]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("action-entry relationship points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ActionEntrySchema]{
		status:  locatorResolutionFound,
		value:   entry,
		subject: label,
	}
	return resolution
}

func resolveReferenceActionEntryLocator(state *actionEntryLocatorState, reference cli.Reference) locatorResolution[cli.ActionEntryReference] {
	var resolution locatorResolution[cli.ActionEntryReference]
	label := actionEntryLocatorLabel(state)
	if state == nil || state.mode == actionEntryLocatorInvalid {
		resolution = locatorResolution[cli.ActionEntryReference]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectActionEntry, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == actionEntryLocatorID {
		entry, ok := reference.ActionEntries[state.id]
		if !ok {
			resolution = locatorResolution[cli.ActionEntryReference]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectActionEntry, observationAbsent),
			}
			return resolution
		}
		resolution = locatorResolution[cli.ActionEntryReference]{
			status:  locatorResolutionFound,
			value:   entry,
			subject: label,
		}
		return resolution
	}
	parent := resolveReferenceCommandLocator(state.command, reference)
	if parent.status != locatorResolutionFound {
		resolution = locatorResolution[cli.ActionEntryReference]{
			status:  parent.status,
			subject: label,
			detail:  parent.detail,
		}
		return resolution
	}
	id, status, detail := selectReferenceActionEntry(state, parent.value.ActionEntries, reference, true)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.ActionEntryReference]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	entry, ok := reference.ActionEntries[id]
	if !ok {
		resolution = locatorResolution[cli.ActionEntryReference]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("action-entry relationship points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ActionEntryReference]{
		status:  locatorResolutionFound,
		value:   entry,
		subject: label,
	}
	return resolution
}

func resolveInspectionActionEntryLocator(state *actionEntryLocatorState, inspection cli.Inspection) locatorResolution[cli.ActionEntryInspection] {
	var resolution locatorResolution[cli.ActionEntryInspection]
	label := actionEntryLocatorLabel(state)
	if state == nil || state.mode == actionEntryLocatorInvalid {
		resolution = locatorResolution[cli.ActionEntryInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectActionEntry, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == actionEntryLocatorID {
		entry, ok := inspection.ActionEntries[state.id]
		if ok {
			resolution = locatorResolution[cli.ActionEntryInspection]{
				status:  locatorResolutionFound,
				value:   entry,
				subject: label,
			}
			return resolution
		}
		if inspection.Truncated {
			resolution = locatorResolution[cli.ActionEntryInspection]{
				status:  locatorResolutionNotProvable,
				subject: label,
				detail:  observed(semanticSubjectInspection, observationTruncated),
			}
			return resolution
		}
		resolution = locatorResolution[cli.ActionEntryInspection]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectActionEntry, observationAbsent),
		}
		return resolution
	}
	parent := resolveInspectionCommandLocator(state.command, inspection)
	if parent.status != locatorResolutionFound {
		resolution = locatorResolution[cli.ActionEntryInspection]{
			status:  parent.status,
			subject: label,
			detail:  parent.detail,
		}
		return resolution
	}
	complete := parent.value.NamespaceCompleteness == cli.CommandNamespaceComplete
	id, status, detail := selectInspectionActionEntry(state, parent.value.ActionEntries, inspection, complete)
	if status == locatorResolutionAbsent && inspection.Truncated {
		status = locatorResolutionNotProvable
		detail = observed(semanticSubjectInspection, observationTruncated)
	}
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.ActionEntryInspection]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	entry, ok := inspection.ActionEntries[id]
	if !ok {
		resolution = locatorResolution[cli.ActionEntryInspection]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("action-entry relationship points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ActionEntryInspection]{
		status:  locatorResolutionFound,
		value:   entry,
		subject: label,
	}
	return resolution
}

//nolint:dupl // Separate public artifacts resolve their own action-entry relationships directly.
func selectSchemaActionEntry(state *actionEntryLocatorState, ids []cli.ActionEntryID, schema *cli.Schema, complete bool) (cli.ActionEntryID, locatorResolutionStatus, evaluationField) {
	if state.mode == actionEntryLocatorCommandIndex {
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectActionEntry, observationInvalidLocator)
		}
		if state.index < len(ids) {
			return ids[state.index], locatorResolutionFound, evaluationText("")
		}
		if complete {
			return 0, locatorResolutionAbsent, observed(semanticSubjectActionEntry, observationAbsent)
		}
		return 0, locatorResolutionNotProvable, evaluationText("command namespace is incomplete")
	}
	matches := make([]cli.ActionEntryID, 0, 1)
	missingDetail := false
	for _, id := range ids {
		entry, ok := schema.ActionEntries[id]
		if !ok {
			missingDetail = true
			continue
		}
		if actionEntryMatches(state, entry.Kind, entry.Action, entry.Long, entry.Short, entry.Literal, entry.Name) {
			matches = append(matches, id)
		}
	}
	if len(matches) == 1 {
		return matches[0], locatorResolutionFound, evaluationText("")
	}
	if len(matches) > 1 {
		field := collectionCardinalityField{
			name:  actionEntryAmbiguityDetail,
			count: len(matches),
		}
		return 0, locatorResolutionAmbiguous, field
	}
	if missingDetail || !complete {
		return 0, locatorResolutionNotProvable, evaluationText("action-entry lookup domain is incomplete")
	}
	return 0, locatorResolutionAbsent, observed(semanticSubjectActionEntry, observationAbsent)
}

//nolint:dupl // Separate public artifacts resolve their own action-entry relationships directly.
func selectReferenceActionEntry(state *actionEntryLocatorState, ids []cli.ActionEntryID, reference cli.Reference, complete bool) (cli.ActionEntryID, locatorResolutionStatus, evaluationField) {
	if state.mode == actionEntryLocatorCommandIndex {
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectActionEntry, observationInvalidLocator)
		}
		if state.index < len(ids) {
			return ids[state.index], locatorResolutionFound, evaluationText("")
		}
		if complete {
			return 0, locatorResolutionAbsent, observed(semanticSubjectActionEntry, observationAbsent)
		}
		return 0, locatorResolutionNotProvable, evaluationText("command namespace is incomplete")
	}
	matches := make([]cli.ActionEntryID, 0, 1)
	missingDetail := false
	for _, id := range ids {
		entry, ok := reference.ActionEntries[id]
		if !ok {
			missingDetail = true
			continue
		}
		if actionEntryMatches(state, entry.Kind, entry.Action, entry.Long, entry.Short, entry.Literal, entry.Name) {
			matches = append(matches, id)
		}
	}
	if len(matches) == 1 {
		return matches[0], locatorResolutionFound, evaluationText("")
	}
	if len(matches) > 1 {
		field := collectionCardinalityField{
			name:  actionEntryAmbiguityDetail,
			count: len(matches),
		}
		return 0, locatorResolutionAmbiguous, field
	}
	if missingDetail || !complete {
		return 0, locatorResolutionNotProvable, evaluationText("action-entry lookup domain is incomplete")
	}
	return 0, locatorResolutionAbsent, observed(semanticSubjectActionEntry, observationAbsent)
}

//nolint:dupl // Separate public artifacts resolve their own action-entry relationships directly.
func selectInspectionActionEntry(state *actionEntryLocatorState, ids []cli.ActionEntryID, inspection cli.Inspection, complete bool) (cli.ActionEntryID, locatorResolutionStatus, evaluationField) {
	if state.mode == actionEntryLocatorCommandIndex {
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectActionEntry, observationInvalidLocator)
		}
		if state.index < len(ids) {
			return ids[state.index], locatorResolutionFound, evaluationText("")
		}
		if complete {
			return 0, locatorResolutionAbsent, observed(semanticSubjectActionEntry, observationAbsent)
		}
		return 0, locatorResolutionNotProvable, evaluationText("command namespace is incomplete")
	}
	matches := make([]cli.ActionEntryID, 0, 1)
	missingDetail := false
	for _, id := range ids {
		entry, ok := inspection.ActionEntries[id]
		if !ok {
			missingDetail = true
			continue
		}
		if actionEntryMatches(state, entry.Kind, entry.Action, entry.Long, entry.Short, entry.Literal, entry.Name) {
			matches = append(matches, id)
		}
	}
	if len(matches) == 1 {
		return matches[0], locatorResolutionFound, evaluationText("")
	}
	if len(matches) > 1 {
		field := collectionCardinalityField{
			name:  actionEntryAmbiguityDetail,
			count: len(matches),
		}
		return 0, locatorResolutionAmbiguous, field
	}
	if missingDetail || !complete {
		return 0, locatorResolutionNotProvable, evaluationText("action-entry lookup domain is incomplete")
	}
	return 0, locatorResolutionAbsent, observed(semanticSubjectActionEntry, observationAbsent)
}

func resolveCommandEntrySchema(state *commandEntryLocatorState, schema *cli.Schema) (cli.CommandEntrySchema, locatorResolutionStatus, evaluationField) {
	resolution := resolveSchemaCommandEntryLocator(state, schema)
	return resolution.value, resolution.status, resolution.detail
}

func resolveCommandEntryReference(state *commandEntryLocatorState, reference cli.Reference) (cli.CommandEntryReference, locatorResolutionStatus, evaluationField) {
	resolution := resolveReferenceCommandEntryLocator(state, reference)
	return resolution.value, resolution.status, resolution.detail
}

func resolveCommandEntryInspection(state *commandEntryLocatorState, inspection cli.Inspection) (cli.CommandEntryInspection, locatorResolutionStatus, evaluationField) {
	resolution := resolveInspectionCommandEntryLocator(state, inspection)
	return resolution.value, resolution.status, resolution.detail
}

func resolveShortcutDefaultSchema(locator CommandShortcutDefaultLocator, schema *cli.Schema) (cli.CommandShortcutDefaultSchema, locatorResolutionStatus, evaluationField) {
	v, s, d := resolveCommandEntrySchema(locator.entry.state, schema)
	if s != locatorResolutionFound {
		return cli.CommandShortcutDefaultSchema{}, s, d
	}
	if locator.index < 0 || locator.index >= len(v.Defaults) {
		return cli.CommandShortcutDefaultSchema{}, locatorResolutionAbsent, observed(semanticSubjectShortcutDefault, observationAbsent)
	}
	return v.Defaults[locator.index], locatorResolutionFound, evaluationText("")
}

func resolveShortcutDefaultReference(locator CommandShortcutDefaultLocator, reference cli.Reference) (cli.CommandShortcutDefaultSchema, locatorResolutionStatus, evaluationField) {
	v, s, d := resolveCommandEntryReference(locator.entry.state, reference)
	if s != locatorResolutionFound {
		return cli.CommandShortcutDefaultSchema{}, s, d
	}
	if locator.index < 0 || locator.index >= len(v.Defaults) {
		return cli.CommandShortcutDefaultSchema{}, locatorResolutionAbsent, observed(semanticSubjectShortcutDefault, observationAbsent)
	}
	return cli.CommandShortcutDefaultSchema(v.Defaults[locator.index]), locatorResolutionFound, evaluationText("")
}

func resolveShortcutDefaultInspection(locator CommandShortcutDefaultLocator, inspection cli.Inspection) (cli.CommandShortcutDefaultInspection, locatorResolutionStatus, evaluationField) {
	v, s, d := resolveCommandEntryInspection(locator.entry.state, inspection)
	if s != locatorResolutionFound {
		return cli.CommandShortcutDefaultInspection{}, s, d
	}
	if locator.index < 0 || locator.index >= len(v.Defaults) {
		return cli.CommandShortcutDefaultInspection{}, locatorResolutionAbsent, observed(semanticSubjectShortcutDefault, observationAbsent)
	}
	return v.Defaults[locator.index], locatorResolutionFound, evaluationText("")
}

func actionEntriesScopedLocator(scope ActionEntriesLocator, mode actionEntryLocatorMode, text string, short rune) ActionEntryLocator {
	state := &actionEntryLocatorState{
		mode:    mode,
		command: scope.command,
		action:  scope.action,
		text:    text,
		short:   short,
	}
	locator := ActionEntryLocator{
		state: state,
	}
	return locator
}

func scopedActionEntryIDsSchema(scope ActionEntriesLocator, schema *cli.Schema) ([]cli.ActionEntryID, locatorResolutionStatus, evaluationField) {
	command, ok, detail := resolveSchemaCommand(scope.command, schema)
	if !ok {
		return nil, locatorResolutionAbsent, detail
	}
	ids := make([]cli.ActionEntryID, 0)
	for _, id := range command.ActionEntries {
		if entry, ok := schema.ActionEntries[id]; ok && entry.Action == scope.action {
			ids = append(ids, id)
		}
	}
	return ids, locatorResolutionFound, evaluationText("")
}

func scopedActionEntryIDsReference(scope ActionEntriesLocator, reference cli.Reference) ([]cli.ActionEntryID, locatorResolutionStatus, evaluationField) {
	command, ok, detail := resolveReferenceCommand(scope.command, reference)
	if !ok {
		return nil, locatorResolutionAbsent, detail
	}
	ids := make([]cli.ActionEntryID, 0)
	for _, id := range command.ActionEntries {
		if entry, ok := reference.ActionEntries[id]; ok && entry.Action == scope.action {
			ids = append(ids, id)
		}
	}
	return ids, locatorResolutionFound, evaluationText("")
}

func scopedActionEntryIDsInspection(scope ActionEntriesLocator, inspection cli.Inspection) ([]cli.ActionEntryID, locatorResolutionStatus, evaluationField) {
	resolved := resolveInspectionCommandLocator(scope.command, inspection)
	if resolved.status != locatorResolutionFound {
		return nil, resolved.status, resolved.detail
	}
	command := resolved.value
	ids := make([]cli.ActionEntryID, 0)
	for _, id := range command.ActionEntries {
		if entry, ok := inspection.ActionEntries[id]; ok && entry.Action == scope.action {
			ids = append(ids, id)
		}
	}
	return ids, locatorResolutionFound, evaluationText("")
}

func expectedSchemaCommandEntryID(state *commandEntryLocatorState, schema *cli.Schema) (cli.CommandEntryID, locatorResolutionStatus, evaluationField) {
	if schema == nil || state == nil || state.mode == commandEntryLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectCommandEntry, observationInvalidLocator)
	}
	if state.mode == commandEntryLocatorID {
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectCommandEntry, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	}
	parent := resolveSchemaCommandLocator(state.command, schema)
	if parent.status != locatorResolutionFound {
		return 0, parent.status, parent.detail
	}
	return selectSchemaCommandEntry(state, parent.value.CanonicalEntry, parent.value.CommandEntries, schema, true)
}

func expectedReferenceCommandEntryID(state *commandEntryLocatorState, reference cli.Reference) (cli.CommandEntryID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == commandEntryLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectCommandEntry, observationInvalidLocator)
	}
	if state.mode == commandEntryLocatorID {
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectCommandEntry, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	}
	parent := resolveReferenceCommandLocator(state.command, reference)
	if parent.status != locatorResolutionFound {
		return 0, parent.status, parent.detail
	}
	return selectReferenceCommandEntry(state, parent.value.CanonicalEntry, parent.value.CommandEntries, reference, true)
}

func expectedInspectionCommandEntryID(state *commandEntryLocatorState, inspection cli.Inspection) (cli.CommandEntryID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == commandEntryLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectCommandEntry, observationInvalidLocator)
	}
	if state.mode == commandEntryLocatorID {
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectCommandEntry, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	}
	parent := resolveInspectionCommandLocator(state.command, inspection)
	if parent.status != locatorResolutionFound {
		return 0, parent.status, parent.detail
	}
	complete := parent.value.NamespaceCompleteness == cli.CommandNamespaceComplete && !inspection.Truncated
	id, status, detail := selectInspectionCommandEntry(state, parent.value.CanonicalEntry, parent.value.CommandEntries, inspection, complete)
	if status == locatorResolutionAbsent && inspection.Truncated {
		return 0, locatorResolutionNotProvable, observed(semanticSubjectInspection, observationTruncated)
	}
	return id, status, detail
}

func evaluateCommandEntryReplacementPresentationSchema(expectation commandEntryReplacementPresentationExpectation, schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaCommandEntryLocator(expectation.locator, schema)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func evaluateCommandEntryReplacementPresentationReference(expectation commandEntryReplacementPresentationExpectation, reference cli.Reference) evaluationResult {
	resolution := resolveReferenceCommandEntryLocator(expectation.locator, reference)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func evaluateCommandEntryReplacementPresentationInspection(expectation commandEntryReplacementPresentationExpectation, inspection cli.Inspection) evaluationResult {
	resolution := resolveInspectionCommandEntryLocator(expectation.locator, inspection)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func evaluateActionEntryReplacementPresentationSchema(expectation actionEntryReplacementPresentationExpectation, schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaActionEntryLocator(expectation.locator, schema)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func evaluateActionEntryReplacementPresentationReference(expectation actionEntryReplacementPresentationExpectation, reference cli.Reference) evaluationResult {
	resolution := resolveReferenceActionEntryLocator(expectation.locator, reference)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func evaluateActionEntryReplacementPresentationInspection(expectation actionEntryReplacementPresentationExpectation, inspection cli.Inspection) evaluationResult {
	resolution := resolveInspectionActionEntryLocator(expectation.locator, inspection)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func resolveShortcutDefaultLiteral(state *declarationValueLiteralLocatorState, label evaluationField, status locatorResolutionStatus, detail evaluationField, values []cli.ValueLiteral) locatorResolution[cli.ValueLiteral] {
	var resolution locatorResolution[cli.ValueLiteral]
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.ValueLiteral]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	if state.index < 0 {
		resolution = locatorResolution[cli.ValueLiteral]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectShortcutDefaultValue, observationInvalidLocator),
		}
		return resolution
	}
	if state.index >= len(values) {
		resolution = locatorResolution[cli.ValueLiteral]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectShortcutDefaultValue, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ValueLiteral]{
		status:  locatorResolutionFound,
		value:   values[state.index],
		subject: label,
	}
	return resolution
}

func commandEntryIDLocator(id cli.CommandEntryID) CommandEntryLocator {
	state := &commandEntryLocatorState{
		mode: commandEntryLocatorID,
		id:   id,
	}
	locator := CommandEntryLocator{
		state: state,
	}
	return locator
}

func actionEntryIDLocator(id cli.ActionEntryID) ActionEntryLocator {
	state := &actionEntryLocatorState{
		mode: actionEntryLocatorID,
		id:   id,
	}
	locator := ActionEntryLocator{
		state: state,
	}
	return locator
}

func (commandActionEntriesExpectation) schemaExpectation() {}

func (commandActionEntriesExpectation) referenceExpectation() {}

func (commandActionEntriesExpectation) inspectionExpectation() {}

//nolint:dupl // Typed collection proofs intentionally keep each public declaration family direct; generic resolver callbacks would recreate the rejected universal framework.
func (expectation commandActionEntriesExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	command := resolveSchemaCommandLocator(expectation.command, schema)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("action entries exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("action entries"))
	actual := command.value.ActionEntries
	for _, id := range actual {
		if _, ok := schema.ActionEntries[id]; !ok {
			return collectionClosureFailure(label, "action entries", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.ActionEntryID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveSchemaActionEntryLocator(state, schema)
		if outcome, failed := collectionResolutionFailure(label, "action entries", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "action entries", expected, actual)
}

//nolint:dupl // Typed collection proofs intentionally keep each public declaration family direct; generic resolver callbacks would recreate the rejected universal framework.
func (expectation commandActionEntriesExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	command := resolveReferenceCommandLocator(expectation.command, reference)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("action entries exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("action entries"))
	actual := command.value.ActionEntries
	for _, id := range actual {
		if _, ok := reference.ActionEntries[id]; !ok {
			return collectionClosureFailure(label, "action entries", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.ActionEntryID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveReferenceActionEntryLocator(state, reference)
		if outcome, failed := collectionResolutionFailure(label, "action entries", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "action entries", expected, actual)
}

func (expectation commandActionEntriesExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	command := resolveInspectionCommandLocator(expectation.command, inspection)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("action entries exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("action entries"))
	if inspection.Truncated {
		return collectionClosureFailure(label, "action entries", observed(semanticSubjectInspection, observationTruncated))
	}
	if command.value.NamespaceCompleteness != cli.CommandNamespaceComplete {
		return collectionClosureFailure(label, "action entries", evaluationText(fmt.Sprintf("command namespace is %v", command.value.NamespaceCompleteness)))
	}
	actual := command.value.ActionEntries
	for _, id := range actual {
		if _, ok := inspection.ActionEntries[id]; !ok {
			return collectionClosureFailure(label, "action entries", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.ActionEntryID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveInspectionActionEntryLocator(state, inspection)
		if outcome, failed := collectionResolutionFailure(label, "action entries", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "action entries", expected, actual)
}

func (commandEntryAvailabilityExpectation) referenceExpectation() {}

func (commandEntryAvailabilityExpectation) inspectionExpectation() {}

func (expectation commandEntryAvailabilityExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceCommandEntryLocator(expectation.locator, reference)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "available", expectation.expected, resolution.value.Available)
}

func (expectation commandEntryAvailabilityExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionCommandEntryLocator(expectation.locator, inspection)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "available", expectation.expected, resolution.value.Available)
}

func (commandEntryVisibilityExpectation) referenceExpectation() {}

func (commandEntryVisibilityExpectation) inspectionExpectation() {}

func (expectation commandEntryVisibilityExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceCommandEntryLocator(expectation.locator, reference)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "visible", expectation.expected, resolution.value.Visible)
}

func (expectation commandEntryVisibilityExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionCommandEntryLocator(expectation.locator, inspection)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "visible", expectation.expected, resolution.value.Visible)
}

func (actionEntryAvailabilityExpectation) referenceExpectation() {}

func (actionEntryAvailabilityExpectation) inspectionExpectation() {}

func (expectation actionEntryAvailabilityExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceActionEntryLocator(expectation.locator, reference)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "available", expectation.expected, resolution.value.Available)
}

func (expectation actionEntryAvailabilityExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionActionEntryLocator(expectation.locator, inspection)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "available", expectation.expected, resolution.value.Available)
}

func (actionEntryVisibilityExpectation) referenceExpectation() {}

func (actionEntryVisibilityExpectation) inspectionExpectation() {}

func (expectation actionEntryVisibilityExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceActionEntryLocator(expectation.locator, reference)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "visible", expectation.expected, resolution.value.Visible)
}

func (expectation actionEntryVisibilityExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionActionEntryLocator(expectation.locator, inspection)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "visible", expectation.expected, resolution.value.Visible)
}

func (commandEntrySourceDescriptionExpectation) schemaExpectation() {}

func (expectation commandEntrySourceDescriptionExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolved := resolveSchemaCommandEntryLocator(expectation.locator, schema)
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, evaluationText("exact source description"), resolved.detail)
	}
	if resolved.value.Description == expectation.expected {
		return provedResult()
	}
	field := descriptionField{
		value: expectation.expected,
	}
	observedField := descriptionField{
		value: resolved.value.Description,
	}
	return disprovedResult(resolved.subject, field, observedField)
}

func (actionEntriesExactlyExpectation) schemaExpectation() {}

func (actionEntriesExactlyExpectation) referenceExpectation() {}

func (actionEntriesExactlyExpectation) inspectionExpectation() {}

func (expectation actionEntriesExactlyExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	expected := evaluationText(fmt.Sprintf("exact action-entry sequence for action %v", expectation.scope.action))
	actual, status, detail := scopedActionEntryIDsSchema(expectation.scope, schema)
	if status != locatorResolutionFound {
		return declarationMissing(status, evaluationText("action entries"), expected, detail, false)
	}
	ids := make([]cli.ActionEntryID, len(expectation.entries))
	for index, state := range expectation.entries {
		resolved := resolveSchemaActionEntryLocator(state, schema)
		if resolved.status != locatorResolutionFound {
			return declarationMissing(resolved.status, evaluationText("action entries"), expected, resolved.detail, false)
		}
		ids[index] = resolved.value.ID
	}
	if slices.Equal(actual, ids) {
		return provedResult()
	}
	return disprovedResult(evaluationText("action entries"), expected, evaluationText("different semantic entry sequence"))
}

func (expectation actionEntriesExactlyExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	expected := evaluationText(fmt.Sprintf("exact action-entry sequence for action %v", expectation.scope.action))
	actual, status, detail := scopedActionEntryIDsReference(expectation.scope, reference)
	if status != locatorResolutionFound {
		return declarationMissing(status, evaluationText("action entries"), expected, detail, false)
	}
	ids := make([]cli.ActionEntryID, len(expectation.entries))
	for index, state := range expectation.entries {
		resolved := resolveReferenceActionEntryLocator(state, reference)
		if resolved.status != locatorResolutionFound {
			return declarationMissing(resolved.status, evaluationText("action entries"), expected, resolved.detail, false)
		}
		ids[index] = resolved.value.ID
	}
	if slices.Equal(actual, ids) {
		return provedResult()
	}
	return disprovedResult(evaluationText("action entries"), expected, evaluationText("different semantic entry sequence"))
}

func (expectation actionEntriesExactlyExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	expected := evaluationText(fmt.Sprintf("exact action-entry sequence for action %v", expectation.scope.action))
	actual, status, detail := scopedActionEntryIDsInspection(expectation.scope, inspection)
	if status != locatorResolutionFound {
		return declarationMissing(status, evaluationText("action entries"), expected, detail, inspection.Truncated)
	}
	ids := make([]cli.ActionEntryID, len(expectation.entries))
	for index, state := range expectation.entries {
		resolved := resolveInspectionActionEntryLocator(state, inspection)
		if resolved.status != locatorResolutionFound {
			return declarationMissing(resolved.status, evaluationText("action entries"), expected, resolved.detail, inspection.Truncated)
		}
		ids[index] = resolved.value.ID
	}
	if slices.Equal(actual, ids) {
		return provedResult()
	}
	return disprovedResult(evaluationText("action entries"), expected, evaluationText("different semantic entry sequence"))
}

func (actionEntryPlacementExpectation) schemaExpectation() {}

func (actionEntryPlacementExpectation) referenceExpectation() {}

func (actionEntryPlacementExpectation) inspectionExpectation() {}

func (expectation actionEntryPlacementExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolved := resolveSchemaActionEntryLocator(expectation.locator, schema)
	expected := evaluationText(fmt.Sprintf("placement=%v", expectation.expected))
	if resolved.status != locatorResolutionFound {
		return declarationMissing(resolved.status, resolved.subject, expected, resolved.detail, false)
	}
	if resolved.value.Placement == expectation.expected {
		return provedResult()
	}
	return disprovedResult(resolved.subject, expected, evaluationText(fmt.Sprintf("placement=%v", resolved.value.Placement)))
}

func (expectation actionEntryPlacementExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolved := resolveReferenceActionEntryLocator(expectation.locator, reference)
	expected := evaluationText(fmt.Sprintf("placement=%v", expectation.expected))
	if resolved.status != locatorResolutionFound {
		return declarationMissing(resolved.status, resolved.subject, expected, resolved.detail, false)
	}
	if resolved.value.Placement == expectation.expected {
		return provedResult()
	}
	return disprovedResult(resolved.subject, expected, evaluationText(fmt.Sprintf("placement=%v", resolved.value.Placement)))
}

func (expectation actionEntryPlacementExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveInspectionActionEntryLocator(expectation.locator, inspection)
	expected := evaluationText(fmt.Sprintf("placement=%v", expectation.expected))
	if resolved.status != locatorResolutionFound {
		return declarationMissing(resolved.status, resolved.subject, expected, resolved.detail, inspection.Truncated)
	}
	if resolved.value.Placement == expectation.expected {
		return provedResult()
	}
	return disprovedResult(resolved.subject, expected, evaluationText(fmt.Sprintf("placement=%v", resolved.value.Placement)))
}

func (actionEntrySourceShortDescriptionExpectation) schemaExpectation() {}

func (expectation actionEntrySourceShortDescriptionExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolved := resolveSchemaActionEntryLocator(expectation.locator, schema)
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, evaluationText("non-empty source short description"), resolved.detail)
	}
	if resolved.value.Description.Short != "" {
		return provedResult()
	}
	return disprovedResult(resolved.subject, evaluationText("non-empty source short description"), evaluationText("source short description is empty"))
}

func (actionEntrySourceDescriptionExpectation) schemaExpectation() {}

func (expectation actionEntrySourceDescriptionExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolved := resolveSchemaActionEntryLocator(expectation.locator, schema)
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.subject, evaluationText("exact source description"), resolved.detail)
	}
	if resolved.value.Description == expectation.expected {
		return provedResult()
	}
	field := descriptionField{
		value: expectation.expected,
	}
	observedField := descriptionField{
		value: resolved.value.Description,
	}
	return disprovedResult(resolved.subject, field, observedField)
}

func (commandEntryOriginExpectation) schemaExpectation() {}

func (commandEntryOriginExpectation) referenceExpectation() {}

func (commandEntryOriginExpectation) inspectionExpectation() {}

func (expectation commandEntryOriginExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaCommandEntryLocator(expectation.locator, schema)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (expectation commandEntryOriginExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceCommandEntryLocator(expectation.locator, reference)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (expectation commandEntryOriginExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionCommandEntryLocator(expectation.locator, inspection)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (commandEntryReleaseLifecycleExpectation) schemaExpectation() {}

func (commandEntryReleaseLifecycleExpectation) referenceExpectation() {}

func (commandEntryReleaseLifecycleExpectation) inspectionExpectation() {}

func (expectation commandEntryReleaseLifecycleExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaCommandEntryLocator(expectation.locator, schema)
	actual := resolution.value.Release
	field := releaseLifecycleField{
		value: expectation.expected,
	}
	observedField := releaseLifecycleField{
		value: actual,
	}
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, field, observedField, actual == expectation.expected)
}

func (expectation commandEntryReleaseLifecycleExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceCommandEntryLocator(expectation.locator, reference)
	actual := resolution.value.Release
	field := releaseLifecycleField{
		value: expectation.expected,
	}
	observedField := releaseLifecycleField{
		value: actual,
	}
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, field, observedField, actual == expectation.expected)
}

func (expectation commandEntryReleaseLifecycleExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionCommandEntryLocator(expectation.locator, inspection)
	actual := resolution.value.Release
	field := releaseLifecycleField{
		value: expectation.expected,
	}
	observedField := releaseLifecycleField{
		value: actual,
	}
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, field, observedField, actual == expectation.expected)
}

func (commandEntryReplacementAbsenceExpectation) schemaExpectation() {}

func (commandEntryReplacementAbsenceExpectation) referenceExpectation() {}

func (commandEntryReplacementAbsenceExpectation) inspectionExpectation() {}

func (expectation commandEntryReplacementAbsenceExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaCommandEntryLocator(expectation.locator, schema)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (expectation commandEntryReplacementAbsenceExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceCommandEntryLocator(expectation.locator, reference)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (expectation commandEntryReplacementAbsenceExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionCommandEntryLocator(expectation.locator, inspection)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (commandEntryReplacementPresentationExpectation) schemaExpectation() {}

func (commandEntryReplacementPresentationExpectation) referenceExpectation() {}

func (commandEntryReplacementPresentationExpectation) inspectionExpectation() {}

func (expectation commandEntryReplacementPresentationExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	return evaluateCommandEntryReplacementPresentationSchema(expectation, schema)
}

func (expectation commandEntryReplacementPresentationExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return evaluateCommandEntryReplacementPresentationReference(expectation, reference)
}

func (expectation commandEntryReplacementPresentationExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return evaluateCommandEntryReplacementPresentationInspection(expectation, inspection)
}

func (commandEntryReplacementTargetDeprecatedExpectation) schemaExpectation() {}

func (commandEntryReplacementTargetDeprecatedExpectation) referenceExpectation() {}

func (commandEntryReplacementTargetDeprecatedExpectation) inspectionExpectation() {}

func (expectation commandEntryReplacementTargetDeprecatedExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaCommandEntryLocator(expectation.locator, schema)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (expectation commandEntryReplacementTargetDeprecatedExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceCommandEntryLocator(expectation.locator, reference)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (expectation commandEntryReplacementTargetDeprecatedExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionCommandEntryLocator(expectation.locator, inspection)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (actionEntryOriginExpectation) schemaExpectation() {}

func (actionEntryOriginExpectation) referenceExpectation() {}

func (actionEntryOriginExpectation) inspectionExpectation() {}

func (expectation actionEntryOriginExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaActionEntryLocator(expectation.locator, schema)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (expectation actionEntryOriginExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceActionEntryLocator(expectation.locator, reference)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (expectation actionEntryOriginExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionActionEntryLocator(expectation.locator, inspection)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (actionEntryReleaseLifecycleExpectation) schemaExpectation() {}

func (actionEntryReleaseLifecycleExpectation) referenceExpectation() {}

func (actionEntryReleaseLifecycleExpectation) inspectionExpectation() {}

func (expectation actionEntryReleaseLifecycleExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaActionEntryLocator(expectation.locator, schema)
	actual := resolution.value.Release
	field := releaseLifecycleField{
		value: expectation.expected,
	}
	observedField := releaseLifecycleField{
		value: actual,
	}
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, field, observedField, actual == expectation.expected)
}

func (expectation actionEntryReleaseLifecycleExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceActionEntryLocator(expectation.locator, reference)
	actual := resolution.value.Release
	field := releaseLifecycleField{
		value: expectation.expected,
	}
	observedField := releaseLifecycleField{
		value: actual,
	}
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, field, observedField, actual == expectation.expected)
}

func (expectation actionEntryReleaseLifecycleExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionActionEntryLocator(expectation.locator, inspection)
	actual := resolution.value.Release
	field := releaseLifecycleField{
		value: expectation.expected,
	}
	observedField := releaseLifecycleField{
		value: actual,
	}
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, field, observedField, actual == expectation.expected)
}

func (actionEntryReplacementAbsenceExpectation) schemaExpectation() {}

func (actionEntryReplacementAbsenceExpectation) referenceExpectation() {}

func (actionEntryReplacementAbsenceExpectation) inspectionExpectation() {}

func (expectation actionEntryReplacementAbsenceExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaActionEntryLocator(expectation.locator, schema)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (expectation actionEntryReplacementAbsenceExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceActionEntryLocator(expectation.locator, reference)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (expectation actionEntryReplacementAbsenceExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionActionEntryLocator(expectation.locator, inspection)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (actionEntryReplacementPresentationExpectation) schemaExpectation() {}

func (actionEntryReplacementPresentationExpectation) referenceExpectation() {}

func (actionEntryReplacementPresentationExpectation) inspectionExpectation() {}

func (expectation actionEntryReplacementPresentationExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	return evaluateActionEntryReplacementPresentationSchema(expectation, schema)
}

func (expectation actionEntryReplacementPresentationExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return evaluateActionEntryReplacementPresentationReference(expectation, reference)
}

func (expectation actionEntryReplacementPresentationExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return evaluateActionEntryReplacementPresentationInspection(expectation, inspection)
}

func (actionEntryReplacementTargetDeprecatedExpectation) schemaExpectation() {}

func (actionEntryReplacementTargetDeprecatedExpectation) referenceExpectation() {}

func (actionEntryReplacementTargetDeprecatedExpectation) inspectionExpectation() {}

func (expectation actionEntryReplacementTargetDeprecatedExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaActionEntryLocator(expectation.locator, schema)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (expectation actionEntryReplacementTargetDeprecatedExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceActionEntryLocator(expectation.locator, reference)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (expectation actionEntryReplacementTargetDeprecatedExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionActionEntryLocator(expectation.locator, inspection)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (commandEntryExistenceExpectation) schemaExpectation() {}

func (commandEntryExistenceExpectation) referenceExpectation() {}

func (commandEntryExistenceExpectation) inspectionExpectation() {}

func (expectation commandEntryExistenceExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	return declarationExistenceResult(resolveSchemaCommandEntryLocator(expectation.locator, schema), expectation.expected)
}

func (expectation commandEntryExistenceExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return declarationExistenceResult(resolveReferenceCommandEntryLocator(expectation.locator, reference), expectation.expected)
}

func (expectation commandEntryExistenceExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return declarationExistenceResult(resolveInspectionCommandEntryLocator(expectation.locator, inspection), expectation.expected)
}

func (actionEntryExistenceExpectation) schemaExpectation() {}

func (actionEntryExistenceExpectation) referenceExpectation() {}

func (actionEntryExistenceExpectation) inspectionExpectation() {}

func (expectation actionEntryExistenceExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	return declarationExistenceResult(resolveSchemaActionEntryLocator(expectation.locator, schema), expectation.expected)
}

func (expectation actionEntryExistenceExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return declarationExistenceResult(resolveReferenceActionEntryLocator(expectation.locator, reference), expectation.expected)
}

func (expectation actionEntryExistenceExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return declarationExistenceResult(resolveInspectionActionEntryLocator(expectation.locator, inspection), expectation.expected)
}

func (commandEntryKindExpectationValue) schemaExpectation() {}

func (commandEntryKindExpectationValue) referenceExpectation() {}

func (commandEntryKindExpectationValue) inspectionExpectation() {}

func (expectation commandEntryKindExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaCommandEntryLocator(expectation.locator, schema)
	actual := resolution.value.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (expectation commandEntryKindExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceCommandEntryLocator(expectation.locator, reference)
	actual := resolution.value.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (expectation commandEntryKindExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionCommandEntryLocator(expectation.locator, inspection)
	actual := resolution.value.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (commandEntryHiddenExpectationValue) schemaExpectation() {}

func (commandEntryHiddenExpectationValue) referenceExpectation() {}

func (commandEntryHiddenExpectationValue) inspectionExpectation() {}

func (expectation commandEntryHiddenExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaCommandEntryLocator(expectation.locator, schema)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("hidden"), true, resolution.value.Hidden)
}

func (expectation commandEntryHiddenExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceCommandEntryLocator(expectation.locator, reference)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("hidden"), true, resolution.value.Hidden)
}

func (expectation commandEntryHiddenExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionCommandEntryLocator(expectation.locator, inspection)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("hidden"), true, resolution.value.Hidden)
}

func (commandEntryLifecycleExpectationValue) schemaExpectation() {}

func (commandEntryLifecycleExpectationValue) referenceExpectation() {}

func (commandEntryLifecycleExpectationValue) inspectionExpectation() {}

func (expectation commandEntryLifecycleExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaCommandEntryLocator(expectation.locator, schema)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (expectation commandEntryLifecycleExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceCommandEntryLocator(expectation.locator, reference)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (expectation commandEntryLifecycleExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionCommandEntryLocator(expectation.locator, inspection)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (commandEntryReplacementExpectationValue) schemaExpectation() {}

func (commandEntryReplacementExpectationValue) referenceExpectation() {}

func (commandEntryReplacementExpectationValue) inspectionExpectation() {}

func (expectation commandEntryReplacementExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaCommandEntryLocator(expectation.locator, schema)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, schema, nil, cli.Inspection{})
}

func (expectation commandEntryReplacementExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceCommandEntryLocator(expectation.locator, reference)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, nil, &reference, cli.Inspection{})
}

func (expectation commandEntryReplacementExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionCommandEntryLocator(expectation.locator, inspection)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, nil, nil, inspection)
}

func (commandEntryTargetExpectationValue) schemaExpectation() {}

func (commandEntryTargetExpectationValue) referenceExpectation() {}

func (commandEntryTargetExpectationValue) inspectionExpectation() {}

func (expectation commandEntryTargetExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	entry := resolveSchemaCommandEntryLocator(expectation.locator, schema)
	target := resolveSchemaCommandLocator(expectation.target, schema)
	return relatedCommandResult(entry.status, entry.subject, entry.detail, target, lastCommandID(entry.value.TargetPath))
}

func (expectation commandEntryTargetExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	entry := resolveReferenceCommandEntryLocator(expectation.locator, reference)
	target := resolveReferenceCommandLocator(expectation.target, reference)
	return relatedReferenceCommandResult(entry.status, entry.subject, entry.detail, target, lastCommandID(entry.value.TargetPath))
}

func (expectation commandEntryTargetExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	entry := resolveInspectionCommandEntryLocator(expectation.locator, inspection)
	target := resolveInspectionCommandLocator(expectation.target, inspection)
	return relatedInspectionCommandResult(entry.status, entry.subject, entry.detail, target, lastCommandID(entry.value.TargetPath))
}

func (actionEntryKindExpectationValue) schemaExpectation() {}

func (actionEntryKindExpectationValue) referenceExpectation() {}

func (actionEntryKindExpectationValue) inspectionExpectation() {}

func (expectation actionEntryKindExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaActionEntryLocator(expectation.locator, schema)
	actual := resolution.value.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (expectation actionEntryKindExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceActionEntryLocator(expectation.locator, reference)
	actual := resolution.value.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (expectation actionEntryKindExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionActionEntryLocator(expectation.locator, inspection)
	actual := resolution.value.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (actionEntryHiddenExpectationValue) schemaExpectation() {}

func (actionEntryHiddenExpectationValue) referenceExpectation() {}

func (actionEntryHiddenExpectationValue) inspectionExpectation() {}

func (expectation actionEntryHiddenExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaActionEntryLocator(expectation.locator, schema)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("hidden"), true, resolution.value.Hidden)
}

func (expectation actionEntryHiddenExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceActionEntryLocator(expectation.locator, reference)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("hidden"), true, resolution.value.Hidden)
}

func (expectation actionEntryHiddenExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionActionEntryLocator(expectation.locator, inspection)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("hidden"), true, resolution.value.Hidden)
}

func (actionEntryLifecycleExpectationValue) schemaExpectation() {}

func (actionEntryLifecycleExpectationValue) referenceExpectation() {}

func (actionEntryLifecycleExpectationValue) inspectionExpectation() {}

func (expectation actionEntryLifecycleExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaActionEntryLocator(expectation.locator, schema)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (expectation actionEntryLifecycleExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceActionEntryLocator(expectation.locator, reference)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (expectation actionEntryLifecycleExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionActionEntryLocator(expectation.locator, inspection)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (actionEntryReplacementExpectationValue) schemaExpectation() {}

func (actionEntryReplacementExpectationValue) referenceExpectation() {}

func (actionEntryReplacementExpectationValue) inspectionExpectation() {}

func (expectation actionEntryReplacementExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaActionEntryLocator(expectation.locator, schema)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, schema, nil, cli.Inspection{})
}

func (expectation actionEntryReplacementExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceActionEntryLocator(expectation.locator, reference)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, nil, &reference, cli.Inspection{})
}

func (expectation actionEntryReplacementExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionActionEntryLocator(expectation.locator, inspection)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, nil, nil, inspection)
}

func (commandEntryShortcutDefaultCountExpectation) schemaExpectation() {}

func (commandEntryShortcutDefaultCountExpectation) referenceExpectation() {}

func (commandEntryShortcutDefaultCountExpectation) inspectionExpectation() {}

func (expectation commandEntryShortcutDefaultCountExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	value, status, detail := resolveCommandEntrySchema(expectation.locator, schema)
	if status != locatorResolutionFound {
		return missingResult(evaluationText("command entry"), evaluationText("shortcut-default-count"), detail, false)
	}
	return equalFieldResult(evaluationText("command entry"), evaluationText("shortcut-default-count"), evaluationText(strconv.Itoa(expectation.expected)), evaluationText(strconv.Itoa(len(value.Defaults))), len(value.Defaults) == expectation.expected)
}

func (expectation commandEntryShortcutDefaultCountExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	value, status, detail := resolveCommandEntryReference(expectation.locator, reference)
	if status != locatorResolutionFound {
		return missingResult(evaluationText("command entry"), evaluationText("shortcut-default-count"), detail, false)
	}
	return equalFieldResult(evaluationText("command entry"), evaluationText("shortcut-default-count"), evaluationText(strconv.Itoa(expectation.expected)), evaluationText(strconv.Itoa(len(value.Defaults))), len(value.Defaults) == expectation.expected)
}

func (expectation commandEntryShortcutDefaultCountExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	value, status, detail := resolveCommandEntryInspection(expectation.locator, inspection)
	if status != locatorResolutionFound {
		return missingResult(evaluationText("command entry"), evaluationText("shortcut-default-count"), detail, inspection.Truncated)
	}
	return equalFieldResult(evaluationText("command entry"), evaluationText("shortcut-default-count"), evaluationText(strconv.Itoa(expectation.expected)), evaluationText(strconv.Itoa(len(value.Defaults))), len(value.Defaults) == expectation.expected)
}

func (shortcutDefaultExpectation) schemaExpectation() {}

func (shortcutDefaultExpectation) referenceExpectation() {}

func (shortcutDefaultExpectation) inspectionExpectation() {}

func (expectation shortcutDefaultExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	value, status, detail := resolveShortcutDefaultSchema(expectation.locator, schema)
	if expectation.property == shortcutDefaultExpectationDestination {
		expected, expectedStatus, _ := resolveSchemaValueID(expectation.destination, schema)
		return expectation.evaluateDestination(value.Destination, status, detail, expected, expectedStatus, false)
	}
	return expectation.evaluateSchemaValue(value, status, detail)
}

func (expectation shortcutDefaultExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	value, status, detail := resolveShortcutDefaultReference(expectation.locator, reference)
	if expectation.property == shortcutDefaultExpectationDestination {
		expected, expectedStatus, _ := resolveReferenceValueID(expectation.destination, reference)
		return expectation.evaluateDestination(value.Destination, status, detail, expected, expectedStatus, false)
	}
	return expectation.evaluateSchemaValue(value, status, detail)
}

func (expectation shortcutDefaultExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	value, status, detail := resolveShortcutDefaultInspection(expectation.locator, inspection)
	if expectation.property == shortcutDefaultExpectationDestination {
		expected, expectedStatus, _ := resolveInspectionValueID(expectation.destination, inspection)
		return expectation.evaluateDestination(value.Destination, status, detail, expected, expectedStatus, inspection.Truncated)
	}
	if status != locatorResolutionFound {
		return missingResult(evaluationText("shortcut default"), expectation.propertyName(), detail, inspection.Truncated)
	}
	subject := evaluationText("shortcut default")
	switch expectation.property {
	case shortcutDefaultExpectationSpelling:
		return equalFieldResult(subject, evaluationText("spelling"), evaluationText(expectation.spelling), evaluationText(value.Spelling), value.Spelling == expectation.spelling)
	case shortcutDefaultExpectationSensitive:
		return equalFieldResult(subject, semanticRequirementSensitive, evaluationBool(expectation.sensitive), evaluationBool(value.Redacted), value.Redacted == expectation.sensitive)
	case shortcutDefaultExpectationValueCount:
		return equalFieldResult(subject, evaluationText("value-count"), evaluationText(strconv.Itoa(expectation.count)), evaluationText(strconv.Itoa(len(value.Values))), len(value.Values) == expectation.count)
	default:
		panic("clitest: invalid shortcut-default expectation")
	}
}

func (expectation shortcutDefaultExpectation) evaluateSchemaValue(value cli.CommandShortcutDefaultSchema, status locatorResolutionStatus, detail evaluationField) evaluationResult {
	if status != locatorResolutionFound {
		return missingResult(evaluationText("shortcut default"), expectation.propertyName(), detail, false)
	}
	subject := evaluationText("shortcut default")
	switch expectation.property {
	case shortcutDefaultExpectationSpelling:
		return equalFieldResult(subject, evaluationText("spelling"), evaluationText(expectation.spelling), evaluationText(value.Spelling), value.Spelling == expectation.spelling)
	case shortcutDefaultExpectationSensitive:
		return equalFieldResult(subject, semanticRequirementSensitive, evaluationBool(expectation.sensitive), evaluationBool(value.Sensitive), value.Sensitive == expectation.sensitive)
	case shortcutDefaultExpectationValueCount:
		return equalFieldResult(subject, evaluationText("value-count"), evaluationText(strconv.Itoa(expectation.count)), evaluationText(strconv.Itoa(len(value.Values))), len(value.Values) == expectation.count)
	default:
		panic("clitest: invalid shortcut-default expectation")
	}
}

func (expectation shortcutDefaultExpectation) evaluateDestination(actual cli.ValueID, status locatorResolutionStatus, detail evaluationField, expected cli.ValueID, expectedStatus locatorResolutionStatus, truncated bool) evaluationResult {
	if status != locatorResolutionFound {
		return missingResult(evaluationText("shortcut default"), evaluationText("destination"), detail, truncated)
	}
	if expectedStatus != locatorResolutionFound {
		return relationLocatorFailureOutcome("shortcut default", "destination", "expected value locator", expectedStatus)
	}
	if actual == expected {
		return provedResult()
	}
	return disprovedResult(evaluationText("shortcut default"), evaluationText("destination matches expected value identity"), evaluationText("destination has different value identity"))
}

func (expectation shortcutDefaultExpectation) propertyName() evaluationField {
	switch expectation.property {
	case shortcutDefaultExpectationSpelling:
		return evaluationText("spelling")
	case shortcutDefaultExpectationSensitive:
		return semanticRequirementSensitive
	case shortcutDefaultExpectationValueCount:
		return evaluationText("value-count")
	case shortcutDefaultExpectationDestination:
		return evaluationText("destination")
	default:
		return evaluationText("shortcut default")
	}
}

func (commandEntrySelectionExpectationValue) inspectionExpectation() {}

func (commandEntrySelectionExpectationValue) completionExpectation() {}

func (expectation commandEntrySelectionExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	if expectation.locator == nil || expectation.locator.mode == commandEntryLocatorInvalid {
		return disprovedResult(evaluationText("command-entry selection"), evaluationText(selectionPresenceWord(expectation.selected)), observed(semanticSubjectCommandEntry, observationInvalidLocator))
	}
	if expectation.locator.mode == commandEntryLocatorID {
		return evaluateCommandEntrySelection(expectation.locator.id, expectation.selected, inspection.Selection.Command)
	}
	resolution := resolveInspectionCommandEntryLocator(expectation.locator, inspection)
	if resolution.status == locatorResolutionNotProvable {
		return notProvableResult(evaluationText("command-entry selection"), evaluationText(selectionPresenceWord(expectation.selected)), resolution.detail)
	}
	if resolution.status != locatorResolutionFound {
		return disprovedResult(evaluationText("command-entry selection"), evaluationText(selectionPresenceWord(expectation.selected)), resolution.detail)
	}
	return evaluateCommandEntrySelection(resolution.value.ID, expectation.selected, inspection.Selection.Command)
}

func (expectation commandEntrySelectionExpectationValue) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	if expectation.locator == nil || expectation.locator.mode == commandEntryLocatorInvalid {
		return disprovedResult(evaluationText("command-entry selection"), evaluationText(selectionPresenceWord(expectation.selected)), observed(semanticSubjectCommandEntry, observationInvalidLocator))
	}
	if expectation.locator.mode != commandEntryLocatorID {
		return notProvableResult(evaluationText("command-entry selection"), evaluationText(selectionPresenceWord(expectation.selected)), evaluationText("completion does not expose the command-entry lookup domain"))
	}
	return evaluateCommandEntrySelection(expectation.locator.id, expectation.selected, result.Selection)
}

func (actionEntryExpectationValue) inspectionExpectation() {}

func (expectation actionEntryExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	if expectation.action == nil || !expectation.action.valid {
		return disprovedResult(evaluationText("invocation action"), evaluationText("configured selected action entry"), evaluationText("invalid action locator"))
	}
	selection := inspection.Selection.Action
	if selection.Selected != expectation.action.action {
		return disprovedResult(evaluationText("invocation action"), evaluationText("configured selected action entry"), evaluationText(fmt.Sprintf("selected action %v", selection.Selected)))
	}
	if expectation.entry == nil || expectation.entry.mode == actionEntryLocatorInvalid {
		return disprovedResult(evaluationText("invocation action"), evaluationText("configured selected action entry"), observed(semanticSubjectActionEntry, observationInvalidLocator))
	}
	if expectation.entry.mode == actionEntryLocatorID {
		if selection.Entry == expectation.entry.id {
			return provedResult()
		}
		return disprovedResult(evaluationText("invocation action"), evaluationText(fmt.Sprintf("action entry %v", expectation.entry.id)), evaluationText(fmt.Sprintf("action entry %v", selection.Entry)))
	}
	resolution := resolveInspectionActionEntryLocator(expectation.entry, inspection)
	if resolution.status == locatorResolutionNotProvable {
		return notProvableResult(evaluationText("invocation action"), evaluationText("configured selected action entry"), resolution.detail)
	}
	if resolution.status != locatorResolutionFound {
		return disprovedResult(evaluationText("invocation action"), evaluationText("configured selected action entry"), resolution.detail)
	}
	if selection.Entry == resolution.value.ID {
		return provedResult()
	}
	return disprovedResult(evaluationText("invocation action"), evaluationText(fmt.Sprintf("action entry %v", resolution.value.ID)), evaluationText(fmt.Sprintf("action entry %v", selection.Entry)))
}

func (actionEntrySelectedExpectationValue) inspectionExpectation() {}

func (expectation actionEntrySelectedExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	expected := evaluationText(selectionPresenceWord(expectation.selected))
	if expectation.locator == nil || expectation.locator.mode == actionEntryLocatorInvalid {
		return disprovedResult(evaluationText("action entry"), expected, observed(semanticSubjectActionEntry, observationInvalidLocator))
	}
	if expectation.locator.mode == actionEntryLocatorID {
		found := inspection.Selection.Action.Entry == expectation.locator.id
		if found == expectation.selected {
			return provedResult()
		}
		return disprovedResult(evaluationText("action entry"), expected, evaluationText(fmt.Sprintf("selected action entry %v", inspection.Selection.Action.Entry)))
	}
	resolution := resolveInspectionActionEntryLocator(expectation.locator, inspection)
	if resolution.status == locatorResolutionNotProvable {
		return notProvableResult(evaluationText("action entry"), expected, resolution.detail)
	}
	if resolution.status != locatorResolutionFound {
		return disprovedResult(evaluationText("action entry"), expected, resolution.detail)
	}
	found := inspection.Selection.Action.Entry == resolution.value.ID
	if found == expectation.selected {
		return provedResult()
	}
	return disprovedResult(evaluationText("action entry"), expected, evaluationText(fmt.Sprintf("selected action entry %v", inspection.Selection.Action.Entry)))
}
