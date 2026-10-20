package clitest

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/abstractionsinc/commandment/cli"
)

// FlagLocator identifies one flag declaration for clitest semantic expectation. Its
// zero value is invalid.
type FlagLocator struct {
	_     [0]func()
	state *flagLocatorState
}

// FlagSpellingLocator identifies one exact flag-spelling declaration for clitest
// semantic expectation. Its zero value is invalid.
type FlagSpellingLocator struct {
	_     [0]func()
	state *flagSpellingLocatorState
}

// FlagByID returns a locator for the flag with the exact public FlagID within the observation
// being evaluated.
func FlagByID(id cli.FlagID) FlagLocator { return flagIDLocator(id) }

// FlagSpellingByID returns a locator for the flag spelling with the exact public FlagSpellingID
// within the observation being evaluated.
func FlagSpellingByID(id cli.FlagSpellingID) FlagSpellingLocator { return flagSpellingIDLocator(id) }

func (FlagLocator) declarationIdentityLocator() {}

func (FlagLocator) diagnosticSubject() {}

func (FlagLocator) noticeSubject() {}

func (FlagLocator) tokenOwner() {}

func (FlagLocator) completionTargetSubject() {}

// Spelling returns a locator for the uniquely resolved spelling text belonging to the located
// flag.
func (locator FlagLocator) Spelling(text string) FlagSpellingLocator {
	return flagSpellingLocator(locator, text)
}

// SpellingAt returns a locator for the flag spelling at index in the located flag's authoritative
// public spelling order.
func (locator FlagLocator) SpellingAt(index int) FlagSpellingLocator {
	return flagSpellingAtLocator(locator, index)
}

// Value returns a locator for the public value associated with the located flag.
func (locator FlagLocator) Value() ValueLocator { return flagValueLocator(locator) }

// Exists returns an expectation that checks the located flag exists. Positive existence may be
// satisfied by truthful partial output when the subject is present.
func (locator FlagLocator) Exists() DeclarationExpectation {
	return flagExistenceExpectation{
		locator:  locator.state,
		expected: true,
	}
}

// DoesNotExist returns an expectation that checks the located flag does not exist; an incomplete
// observation cannot establish absence.
func (locator FlagLocator) DoesNotExist() DeclarationExpectation {
	return flagExistenceExpectation{
		locator:  locator.state,
		expected: false,
	}
}

// Required returns an expectation that checks the located flag is required.
func (locator FlagLocator) Required() DeclarationExpectation {
	return flagRequiredExpectationValue{locator: locator.state}
}

// Hidden returns an expectation that checks the located flag is marked hidden by the public
// Commandment observation.
func (locator FlagLocator) Hidden() DeclarationExpectation {
	return flagHiddenExpectationValue{locator: locator.state}
}

// Sensitive returns an expectation that checks the located flag is marked sensitive. Expectation failure
// preserves Commandment redaction and does not expose secret material.
func (locator FlagLocator) Sensitive() DeclarationExpectation {
	return flagSensitiveExpectationValue{locator: locator.state}
}

// Lifecycle returns an expectation that checks the located flag has the expected effective
// lifecycle state; this is distinct from release-lifecycle metadata.
func (locator FlagLocator) Lifecycle(v cli.LifecycleState) DeclarationExpectation {
	return flagLifecycleExpectationValue{
		locator:  locator.state,
		expected: v,
	}
}

// ReplacedBy returns an expectation that checks the located flag replacement points to the exact
// declaration identified by v, using public declaration identity rather than presentation text.
func (locator FlagLocator) ReplacedBy(v DeclarationIdentityLocator) DeclarationExpectation {
	return flagReplacementExpectationValue{
		locator:  locator.state,
		expected: v,
	}
}

// Available returns an expectation that checks the located flag is available in the applicable
// public observation.
func (locator FlagLocator) Available() EffectiveDeclarationExpectation {
	return flagAvailabilityExpectation{
		locator:  locator.state,
		expected: true,
	}
}

// Unavailable returns an expectation that checks the located flag is unavailable in the applicable
// public observation.
func (locator FlagLocator) Unavailable() EffectiveDeclarationExpectation {
	return flagAvailabilityExpectation{
		locator:  locator.state,
		expected: false,
	}
}

// Visible returns an expectation that checks the located flag is visible in the applicable public
// observation.
func (locator FlagLocator) Visible() EffectiveDeclarationExpectation {
	return flagVisibilityExpectation{
		locator:  locator.state,
		expected: true,
	}
}

// NotVisible returns an expectation that checks the located flag is not visible in the applicable
// public observation.
func (locator FlagLocator) NotVisible() EffectiveDeclarationExpectation {
	return flagVisibilityExpectation{
		locator:  locator.state,
		expected: false,
	}
}

// SpellingsExactly returns an expectation that checks the located flag's flag spellings are
// exactly the supplied locators in authoritative public order. Exactness is not inferred from an
// incomplete observation.
func (locator FlagLocator) SpellingsExactly(v ...FlagSpellingLocator) DeclarationExpectation {
	expected := make([]*flagSpellingLocatorState, len(v))
	for index, locator := range v {
		expected[index] = locator.state
	}
	return flagSpellingsExpectationValue{
		locator:  locator.state,
		expected: expected,
	}
}

// Present returns an expectation that checks the located flag is present in the analyzed
// invocation, independently of whether its declaration merely exists.
func (locator FlagLocator) Present() InspectionExpectation {
	return flagPresenceExpectationValue{
		locator:  locator.state,
		expected: true,
	}
}

// Absent returns an expectation that checks the located flag is absent from the analyzed
// invocation; incomplete invocation evidence cannot be treated as fabricated absence.
func (locator FlagLocator) Absent() InspectionExpectation {
	return flagPresenceExpectationValue{
		locator:  locator.state,
		expected: false,
	}
}

// LocaleSelector returns a reusable declaration expectation that checks the public
// locale-selector state exposed by the located flag.
func (locator FlagLocator) LocaleSelector() DeclarationExpectation {
	return flagCorePropertyExpectation{
		locator: locator.state,
		kind:    flagCorePropertyLocaleSelector,
	}
}

// Occurrences returns a reusable declaration expectation that checks the public occurrence
// range exposed by the located flag.
func (locator FlagLocator) Occurrences(expected cli.CountRange) DeclarationExpectation {
	return flagCorePropertyExpectation{
		locator: locator.state,
		kind:    flagCorePropertyOccurrences,
		count:   expected,
	}
}

// ValuesPerOccurrence returns a reusable declaration expectation that checks the public
// values-per-occurrence range exposed by the located flag.
func (locator FlagLocator) ValuesPerOccurrence(expected cli.CountRange) DeclarationExpectation {
	return flagCorePropertyExpectation{
		locator: locator.state,
		kind:    flagCorePropertyValuesPerOccurrence,
		count:   expected,
	}
}

// Placement returns a reusable declaration expectation that checks the public placement exposed
// by the located flag.
func (locator FlagLocator) Placement(expected cli.FlagPlacement) DeclarationExpectation {
	return flagCorePropertyExpectation{
		locator:   locator.state,
		kind:      flagCorePropertyPlacement,
		placement: expected,
	}
}

// ValueName returns a reusable declaration expectation that checks the public value name
// exposed by the located flag.
func (locator FlagLocator) ValueName(expected string) DeclarationExpectation {
	return flagCorePropertyExpectation{
		locator: locator.state,
		kind:    flagCorePropertyValueName,
		text:    expected,
	}
}

// Description returns a reusable declaration expectation that checks the public description
// exposed by the located flag.
func (locator FlagLocator) Description(expected string) DeclarationExpectation {
	return flagCorePropertyExpectation{
		locator: locator.state,
		kind:    flagCorePropertyDescription,
		text:    expected,
	}
}

// Completion returns a locator that follows the public completion relationship from the located
// flag. The returned locator carries locator coordinates, not observed state.
func (locator FlagLocator) Completion() CompletionDeclarationLocator {
	return CompletionDeclarationLocator{state: &completionDeclarationLocatorState{
		mode: completionDeclarationLocatorFlag,
		flag: locator.state,
	}}
}

// Origin returns an expectation that checks the located flag has the framework-established
// DeclarationOrigin expected.
func (locator FlagLocator) Origin(expected cli.DeclarationOrigin) DeclarationExpectation {
	return flagOriginExpectation{
		locator:  locator.state,
		expected: expected,
	}
}

// ReleaseLifecycle returns an expectation that checks the located flag has the exact public
// release-lifecycle metadata expected.
func (locator FlagLocator) ReleaseLifecycle(expected cli.ReleaseLifecycle) DeclarationExpectation {
	return flagReleaseLifecycleExpectation{
		locator:  locator.state,
		expected: expected,
	}
}

// NotReplaced returns an expectation that checks the located flag has no public
// declaration-replacement relationship. Incomplete observation cannot manufacture replacement
// absence.
func (locator FlagLocator) NotReplaced() DeclarationExpectation {
	return flagReplacementAbsenceExpectation{locator: locator.state}
}

// ReplacementPresentation returns an expectation that checks the located flag replacement carries
// the expected public presentation text.
func (locator FlagLocator) ReplacementPresentation(expected string) DeclarationExpectation {
	return flagReplacementPresentationExpectation{
		locator:  locator.state,
		expected: expected,
	}
}

// ReplacementTargetDeprecatedDuringMigration returns an expectation that checks the located flag
// replacement carries the expected migration-deprecation marker.
func (locator FlagLocator) ReplacementTargetDeprecatedDuringMigration(expected bool) DeclarationExpectation {
	return flagReplacementTargetDeprecatedExpectation{
		locator:  locator.state,
		expected: expected,
	}
}

func (FlagSpellingLocator) declarationIdentityLocator() {}

func (FlagSpellingLocator) diagnosticSubject() {}

func (FlagSpellingLocator) noticeSubject() {}

func (FlagSpellingLocator) tokenOwner() {}

// Exists returns an expectation that checks the located flag spelling exists. Positive existence
// may be satisfied by truthful partial output when the subject is present.
func (locator FlagSpellingLocator) Exists() DeclarationExpectation {
	return flagSpellingExistenceExpectation{
		locator:  locator.state,
		expected: true,
	}
}

// DoesNotExist returns an expectation that checks the located flag spelling does not exist; an
// incomplete observation cannot establish absence.
func (locator FlagSpellingLocator) DoesNotExist() DeclarationExpectation {
	return flagSpellingExistenceExpectation{
		locator:  locator.state,
		expected: false,
	}
}

// Kind returns an expectation that checks the located flag spelling has the expected public kind.
func (locator FlagSpellingLocator) Kind(v cli.FlagSpellingKind) DeclarationExpectation {
	return flagSpellingKindExpectationValue{
		locator:  locator.state,
		expected: v,
	}
}

// Lifecycle returns an expectation that checks the located flag spelling has the expected
// effective lifecycle state; this is distinct from release-lifecycle metadata.
func (locator FlagSpellingLocator) Lifecycle(v cli.LifecycleState) DeclarationExpectation {
	return flagSpellingLifecycleExpectationValue{
		locator:  locator.state,
		expected: v,
	}
}

// ReplacedBy returns an expectation that checks the located flag spelling replacement points to
// the exact declaration identified by v, using public declaration identity rather than
// presentation text.
func (locator FlagSpellingLocator) ReplacedBy(v DeclarationIdentityLocator) DeclarationExpectation {
	return flagSpellingReplacementExpectationValue{
		locator:  locator.state,
		expected: v,
	}
}

// Available returns an expectation that checks the located flag spelling is available in the
// applicable public observation.
func (locator FlagSpellingLocator) Available() EffectiveDeclarationExpectation {
	return flagSpellingAvailabilityExpectation{
		locator:  locator.state,
		expected: true,
	}
}

// Unavailable returns an expectation that checks the located flag spelling is unavailable in the
// applicable public observation.
func (locator FlagSpellingLocator) Unavailable() EffectiveDeclarationExpectation {
	return flagSpellingAvailabilityExpectation{
		locator:  locator.state,
		expected: false,
	}
}

// Visible returns an expectation that checks the located flag spelling is visible in the
// applicable public observation.
func (locator FlagSpellingLocator) Visible() EffectiveDeclarationExpectation {
	return flagSpellingVisibilityExpectation{
		locator:  locator.state,
		expected: true,
	}
}

// NotVisible returns an expectation that checks the located flag spelling is not visible in the
// applicable public observation.
func (locator FlagSpellingLocator) NotVisible() EffectiveDeclarationExpectation {
	return flagSpellingVisibilityExpectation{
		locator:  locator.state,
		expected: false,
	}
}

// Present returns an expectation that checks the located flag spelling is present in the analyzed
// invocation, independently of whether its declaration merely exists.
func (locator FlagSpellingLocator) Present() InspectionExpectation {
	return flagSpellingPresenceExpectationValue{
		locator:  locator.state,
		expected: true,
	}
}

// Absent returns an expectation that checks the located flag spelling is absent from the analyzed
// invocation; incomplete invocation evidence cannot be treated as fabricated absence.
func (locator FlagSpellingLocator) Absent() InspectionExpectation {
	return flagSpellingPresenceExpectationValue{
		locator:  locator.state,
		expected: false,
	}
}

// Hidden returns a reusable declaration expectation that checks the public hidden state exposed
// by the located flag spelling.
func (locator FlagSpellingLocator) Hidden() DeclarationExpectation {
	return flagSpellingCorePropertyExpectation{
		locator: locator.state,
		kind:    flagSpellingCorePropertyHidden,
	}
}

// Sensitive returns a reusable declaration expectation that checks the located flag spelling is
// marked sensitive. Expectation failures preserve Commandment redaction boundaries.
func (locator FlagSpellingLocator) Sensitive() DeclarationExpectation {
	return flagSpellingCorePropertyExpectation{
		locator: locator.state,
		kind:    flagSpellingCorePropertySensitive,
	}
}

// FixedValue returns a locator that follows the public fixed value relationship from the
// located flag spelling. The returned locator carries locator coordinates, not observed state.
func (locator FlagSpellingLocator) FixedValue() DeclarationValueLiteralLocator {
	return DeclarationValueLiteralLocator{state: &declarationValueLiteralLocatorState{
		mode:         declarationValueLiteralLocatorFlagFixedValue,
		flagSpelling: locator.state,
	}}
}

// ValueFormsExactly returns a reusable declaration expectation that checks the complete flag
// value-form sequence exactly in authoritative public order.
func (locator FlagSpellingLocator) ValueFormsExactly(expected ...cli.FlagValueForm) DeclarationExpectation {
	return flagSpellingCorePropertyExpectation{
		locator:    locator.state,
		kind:       flagSpellingCorePropertyValueForms,
		valueForms: slices.Clone(expected),
	}
}

// Origin returns an expectation that checks the located flag spelling has the
// framework-established DeclarationOrigin expected.
func (locator FlagSpellingLocator) Origin(expected cli.DeclarationOrigin) DeclarationExpectation {
	return flagSpellingOriginExpectation{
		locator:  locator.state,
		expected: expected,
	}
}

// ReleaseLifecycle returns an expectation that checks the located flag spelling has the exact
// public release-lifecycle metadata expected.
func (locator FlagSpellingLocator) ReleaseLifecycle(expected cli.ReleaseLifecycle) DeclarationExpectation {
	return flagSpellingReleaseLifecycleExpectation{
		locator:  locator.state,
		expected: expected,
	}
}

// NotReplaced returns an expectation that checks the located flag spelling has no public
// declaration-replacement relationship. Incomplete observation cannot manufacture replacement
// absence.
func (locator FlagSpellingLocator) NotReplaced() DeclarationExpectation {
	return flagSpellingReplacementAbsenceExpectation{locator: locator.state}
}

// ReplacementPresentation returns an expectation that checks the located flag spelling replacement
// carries the expected public presentation text.
func (locator FlagSpellingLocator) ReplacementPresentation(expected string) DeclarationExpectation {
	return flagSpellingReplacementPresentationExpectation{
		locator:  locator.state,
		expected: expected,
	}
}

// ReplacementTargetDeprecatedDuringMigration returns an expectation that checks the located flag
// spelling replacement carries the expected migration-deprecation marker.
func (locator FlagSpellingLocator) ReplacementTargetDeprecatedDuringMigration(expected bool) DeclarationExpectation {
	return flagSpellingReplacementTargetDeprecatedExpectation{
		locator:  locator.state,
		expected: expected,
	}
}

// Flag returns a locator for the uniquely resolved flag named name under the located command.
func (l CommandLocator) Flag(name string) FlagLocator { return commandFlagLocator(l, name) }

// FlagAt returns a locator for the flag at index in the located command's authoritative public
// flag order.
func (l CommandLocator) FlagAt(index int) FlagLocator {
	return commandFlagAtLocator(l, index)
}

// FlagsExactly returns an expectation that checks the located command's flags are exactly the
// supplied locators in authoritative public order. Exactness is not inferred from an incomplete
// observation.
func (l CommandLocator) FlagsExactly(values ...FlagLocator) DeclarationExpectation {
	return commandFlagsExpectation{
		command: l.state,
		items:   flagLocatorStates(values),
	}
}

type commandFlagsExpectation struct {
	command *commandLocatorState
	items   []*flagLocatorState
}

type flagLocatorMode uint8

const (
	flagLocatorInvalid flagLocatorMode = iota
	flagLocatorID
	flagLocatorName
	flagLocatorIndex
)

type flagLocatorState struct {
	mode flagLocatorMode
	id   cli.FlagID

	command *commandLocatorState
	name    string
	index   int
}

type flagSpellingLocatorMode uint8

const (
	flagSpellingLocatorInvalid flagSpellingLocatorMode = iota
	flagSpellingLocatorID
	flagSpellingLocatorText
	flagSpellingLocatorIndex
)

type flagSpellingLocatorState struct {
	mode flagSpellingLocatorMode
	id   cli.FlagSpellingID

	flag  *flagLocatorState
	text  string
	index int
}

type flagAvailabilityExpectation struct {
	locator  *flagLocatorState
	expected bool
}

type flagVisibilityExpectation struct {
	locator  *flagLocatorState
	expected bool
}

type flagSpellingAvailabilityExpectation struct {
	locator  *flagSpellingLocatorState
	expected bool
}

type flagSpellingVisibilityExpectation struct {
	locator  *flagSpellingLocatorState
	expected bool
}

type flagOriginExpectation struct {
	locator  *flagLocatorState
	expected cli.DeclarationOrigin
}

type flagReleaseLifecycleExpectation struct {
	locator  *flagLocatorState
	expected cli.ReleaseLifecycle
}

type flagReplacementAbsenceExpectation struct{ locator *flagLocatorState }

type flagReplacementPresentationExpectation struct {
	locator  *flagLocatorState
	expected string
}

type flagReplacementTargetDeprecatedExpectation struct {
	locator  *flagLocatorState
	expected bool
}

type flagSpellingOriginExpectation struct {
	locator  *flagSpellingLocatorState
	expected cli.DeclarationOrigin
}

type flagSpellingReleaseLifecycleExpectation struct {
	locator  *flagSpellingLocatorState
	expected cli.ReleaseLifecycle
}

type flagSpellingReplacementAbsenceExpectation struct{ locator *flagSpellingLocatorState }

type flagSpellingReplacementPresentationExpectation struct {
	locator  *flagSpellingLocatorState
	expected string
}

type flagSpellingReplacementTargetDeprecatedExpectation struct {
	locator  *flagSpellingLocatorState
	expected bool
}

type flagExistenceExpectation struct {
	locator  *flagLocatorState
	expected bool
}

type flagSpellingExistenceExpectation struct {
	locator  *flagSpellingLocatorState
	expected bool
}

type (
	flagRequiredExpectationValue  struct{ locator *flagLocatorState }
	flagHiddenExpectationValue    struct{ locator *flagLocatorState }
	flagSensitiveExpectationValue struct{ locator *flagLocatorState }
	flagLifecycleExpectationValue struct {
		locator  *flagLocatorState
		expected cli.LifecycleState
	}
)

type flagReplacementExpectationValue struct {
	locator  *flagLocatorState
	expected DeclarationIdentityLocator
}

type flagSpellingsExpectationValue struct {
	locator  *flagLocatorState
	expected []*flagSpellingLocatorState
}

type flagPresenceExpectationValue struct {
	locator  *flagLocatorState
	expected bool
}

type flagSpellingKindExpectationValue struct {
	locator  *flagSpellingLocatorState
	expected cli.FlagSpellingKind
}

type flagSpellingLifecycleExpectationValue struct {
	locator  *flagSpellingLocatorState
	expected cli.LifecycleState
}

type flagSpellingReplacementExpectationValue struct {
	locator  *flagSpellingLocatorState
	expected DeclarationIdentityLocator
}

type flagSpellingPresenceExpectationValue struct {
	locator  *flagSpellingLocatorState
	expected bool
}

type flagCorePropertyKind uint8

const (
	flagCorePropertyInvalid flagCorePropertyKind = iota
	flagCorePropertyOccurrences
	flagCorePropertyValuesPerOccurrence
	flagCorePropertyPlacement
	flagCorePropertyValueName
	flagCorePropertyDescription
	flagCorePropertyLocaleSelector
)

type flagCorePropertyExpectation struct {
	locator   *flagLocatorState
	kind      flagCorePropertyKind
	count     cli.CountRange
	placement cli.FlagPlacement
	text      string
}

type flagSpellingCorePropertyKind uint8

const (
	flagSpellingCorePropertyInvalid flagSpellingCorePropertyKind = iota
	flagSpellingCorePropertyHidden
	flagSpellingCorePropertySensitive
	flagSpellingCorePropertyValueForms
)

type flagSpellingCorePropertyExpectation struct {
	locator    *flagSpellingLocatorState
	kind       flagSpellingCorePropertyKind
	valueForms []cli.FlagValueForm
}

func flagLocatorStates(values []FlagLocator) []*flagLocatorState {
	states := make([]*flagLocatorState, len(values))
	for index, value := range values {
		states[index] = value.state
	}
	return states
}

func commandFlagLocator(parent CommandLocator, name string) FlagLocator {
	return FlagLocator{state: &flagLocatorState{
		mode:    flagLocatorName,
		command: parent.state,
		name:    name,
	}}
}

func commandFlagAtLocator(parent CommandLocator, index int) FlagLocator {
	return FlagLocator{state: &flagLocatorState{
		mode:    flagLocatorIndex,
		command: parent.state,
		index:   index,
	}}
}

func flagSpellingLocator(parent FlagLocator, text string) FlagSpellingLocator {
	return FlagSpellingLocator{state: &flagSpellingLocatorState{
		mode: flagSpellingLocatorText,
		flag: parent.state,
		text: text,
	}}
}

func flagSpellingAtLocator(parent FlagLocator, index int) FlagSpellingLocator {
	return FlagSpellingLocator{state: &flagSpellingLocatorState{
		mode:  flagSpellingLocatorIndex,
		flag:  parent.state,
		index: index,
	}}
}

func flagValueLocator(parent FlagLocator) ValueLocator {
	return ValueLocator{state: &valueLocatorState{
		mode: valueLocatorFlag,
		flag: parent.state,
	}}
}

func flagLocatorLabel(state *flagLocatorState) evaluationField {
	if state == nil {
		return semanticSubjectFlag
	}
	switch state.mode {
	case flagLocatorID:
		return evaluationText(fmt.Sprintf("flag id %d", state.id))
	case flagLocatorName:
		return qualifySubject(commandLocatorLabel(state.command), subjectNamed(evaluationText("flag"), state.name))
	case flagLocatorIndex:
		return qualifySubject(commandLocatorLabel(state.command), evaluationText(fmt.Sprintf("flag %d", state.index)))
	default:
		return semanticSubjectFlag
	}
}

//nolint:dupl // Schema resolves flag relationships directly instead of projecting through artifact callbacks.
func resolveSchemaFlagLocator(state *flagLocatorState, schema *cli.Schema) locatorResolution[cli.FlagSchema] {
	var resolution locatorResolution[cli.FlagSchema]
	label := flagLocatorLabel(state)
	if schema == nil || state == nil || state.mode == flagLocatorInvalid {
		resolution = locatorResolution[cli.FlagSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectFlag, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == flagLocatorID {
		value, ok := schema.Flags[state.id]
		if !ok {
			resolution = locatorResolution[cli.FlagSchema]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectFlag, observationAbsent),
			}
			return resolution
		}
		resolution = locatorResolution[cli.FlagSchema]{
			status:  locatorResolutionFound,
			value:   value,
			subject: label,
		}
		return resolution
	}
	command := resolveSchemaCommandLocator(state.command, schema)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.FlagSchema]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	id, status, detail := selectSchemaFlag(state, command.value.Flags, schema, true)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.FlagSchema]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	value, ok := schema.Flags[id]
	if !ok {
		resolution = locatorResolution[cli.FlagSchema]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("flag relationship points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.FlagSchema]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

func resolveReferenceFlagLocator(state *flagLocatorState, reference cli.Reference) locatorResolution[cli.FlagReference] {
	var resolution locatorResolution[cli.FlagReference]
	label := flagLocatorLabel(state)
	if state == nil || state.mode == flagLocatorInvalid {
		resolution = locatorResolution[cli.FlagReference]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectFlag, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == flagLocatorID {
		value, ok := reference.Flags[state.id]
		if !ok {
			resolution = locatorResolution[cli.FlagReference]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectFlag, observationAbsent),
			}
			return resolution
		}
		resolution = locatorResolution[cli.FlagReference]{
			status:  locatorResolutionFound,
			value:   value,
			subject: label,
		}
		return resolution
	}
	command := resolveReferenceCommandLocator(state.command, reference)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.FlagReference]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	id, status, detail := selectReferenceFlag(state, command.value.Flags, reference, true)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.FlagReference]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	value, ok := reference.Flags[id]
	if !ok {
		resolution = locatorResolution[cli.FlagReference]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("flag relationship points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.FlagReference]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

func resolveInspectionFlagLocator(state *flagLocatorState, inspection cli.Inspection) locatorResolution[cli.FlagInspection] {
	var resolution locatorResolution[cli.FlagInspection]
	label := flagLocatorLabel(state)
	if state == nil || state.mode == flagLocatorInvalid {
		resolution = locatorResolution[cli.FlagInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectFlag, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == flagLocatorID {
		value, ok := inspection.Flags[state.id]
		if ok {
			resolution = locatorResolution[cli.FlagInspection]{
				status:  locatorResolutionFound,
				value:   value,
				subject: label,
			}
			return resolution
		}
		if inspection.Truncated {
			resolution = locatorResolution[cli.FlagInspection]{
				status:  locatorResolutionNotProvable,
				subject: label,
				detail:  observed(semanticSubjectInspection, observationTruncated),
			}
			return resolution
		}
		resolution = locatorResolution[cli.FlagInspection]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectFlag, observationAbsent),
		}
		return resolution
	}
	command := resolveInspectionCommandLocator(state.command, inspection)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.FlagInspection]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	complete := command.value.NamespaceCompleteness == cli.CommandNamespaceComplete
	id, status, detail := selectInspectionFlag(state, command.value.Flags, inspection, complete)
	if status == locatorResolutionAbsent && inspection.Truncated {
		status = locatorResolutionNotProvable
		detail = observed(semanticSubjectInspection, observationTruncated)
	}
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.FlagInspection]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	value, ok := inspection.Flags[id]
	if !ok {
		resolution = locatorResolution[cli.FlagInspection]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("flag relationship points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.FlagInspection]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

//nolint:dupl // Separate public artifacts resolve their own flag relationships directly.
func selectSchemaFlag(state *flagLocatorState, ids []cli.FlagID, schema *cli.Schema, complete bool) (cli.FlagID, locatorResolutionStatus, evaluationField) {
	switch state.mode {
	case flagLocatorIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectFlag, observationInvalidLocator)
		}
		if state.index < len(ids) {
			return ids[state.index], locatorResolutionFound, evaluationText("")
		}
		if complete {
			return 0, locatorResolutionAbsent, observed(semanticSubjectFlag, observationAbsent)
		}
		return 0, locatorResolutionNotProvable, evaluationText("flag lookup domain is incomplete")
	case flagLocatorName:
		matches := make([]cli.FlagID, 0, 1)
		missingDetail := false
		for _, id := range ids {
			flag, ok := schema.Flags[id]
			if !ok {
				missingDetail = true
				continue
			}
			for _, spellingID := range flag.Spellings {
				spelling, ok := schema.FlagSpellings[spellingID]
				if !ok {
					missingDetail = true
					continue
				}
				if spelling.Spelling.Text == state.name {
					matches = append(matches, id)
					break
				}
			}
		}
		slices.Sort(matches)
		if len(matches) == 1 {
			return matches[0], locatorResolutionFound, evaluationText("")
		}
		if len(matches) > 1 {
			return 0, locatorResolutionAmbiguous, collectionCardinalityField{
				name:  flagAmbiguityDetail,
				count: len(matches),
			}
		}
		if missingDetail || !complete {
			return 0, locatorResolutionNotProvable, evaluationText("flag lookup domain is incomplete")
		}
		return 0, locatorResolutionAbsent, observed(semanticSubjectFlag, observationAbsent)
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectFlag, observationInvalidLocator)
	}
}

//nolint:dupl // Separate public artifacts resolve their own flag relationships directly.
func selectReferenceFlag(state *flagLocatorState, ids []cli.FlagID, reference cli.Reference, complete bool) (cli.FlagID, locatorResolutionStatus, evaluationField) {
	switch state.mode {
	case flagLocatorIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectFlag, observationInvalidLocator)
		}
		if state.index < len(ids) {
			return ids[state.index], locatorResolutionFound, evaluationText("")
		}
		if complete {
			return 0, locatorResolutionAbsent, observed(semanticSubjectFlag, observationAbsent)
		}
		return 0, locatorResolutionNotProvable, evaluationText("flag lookup domain is incomplete")
	case flagLocatorName:
		matches := make([]cli.FlagID, 0, 1)
		missingDetail := false
		for _, id := range ids {
			flag, ok := reference.Flags[id]
			if !ok {
				missingDetail = true
				continue
			}
			for _, spellingID := range flag.Spellings {
				spelling, ok := reference.FlagSpellings[spellingID]
				if !ok {
					missingDetail = true
					continue
				}
				if spelling.Spelling.Text == state.name {
					matches = append(matches, id)
					break
				}
			}
		}
		slices.Sort(matches)
		if len(matches) == 1 {
			return matches[0], locatorResolutionFound, evaluationText("")
		}
		if len(matches) > 1 {
			return 0, locatorResolutionAmbiguous, collectionCardinalityField{
				name:  flagAmbiguityDetail,
				count: len(matches),
			}
		}
		if missingDetail || !complete {
			return 0, locatorResolutionNotProvable, evaluationText("flag lookup domain is incomplete")
		}
		return 0, locatorResolutionAbsent, observed(semanticSubjectFlag, observationAbsent)
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectFlag, observationInvalidLocator)
	}
}

//nolint:dupl // Separate public artifacts resolve their own flag relationships directly.
func selectInspectionFlag(state *flagLocatorState, ids []cli.FlagID, inspection cli.Inspection, complete bool) (cli.FlagID, locatorResolutionStatus, evaluationField) {
	switch state.mode {
	case flagLocatorIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectFlag, observationInvalidLocator)
		}
		if state.index < len(ids) {
			return ids[state.index], locatorResolutionFound, evaluationText("")
		}
		if complete {
			return 0, locatorResolutionAbsent, observed(semanticSubjectFlag, observationAbsent)
		}
		return 0, locatorResolutionNotProvable, evaluationText("flag lookup domain is incomplete")
	case flagLocatorName:
		matches := make([]cli.FlagID, 0, 1)
		missingDetail := false
		for _, id := range ids {
			flag, ok := inspection.Flags[id]
			if !ok {
				missingDetail = true
				continue
			}
			for _, spellingID := range flag.Spellings {
				spelling, ok := inspection.FlagSpellings[spellingID]
				if !ok {
					missingDetail = true
					continue
				}
				if spelling.Spelling.Text == state.name {
					matches = append(matches, id)
					break
				}
			}
		}
		slices.Sort(matches)
		if len(matches) == 1 {
			return matches[0], locatorResolutionFound, evaluationText("")
		}
		if len(matches) > 1 {
			return 0, locatorResolutionAmbiguous, collectionCardinalityField{
				name:  flagAmbiguityDetail,
				count: len(matches),
			}
		}
		if missingDetail || !complete {
			return 0, locatorResolutionNotProvable, evaluationText("flag lookup domain is incomplete")
		}
		return 0, locatorResolutionAbsent, observed(semanticSubjectFlag, observationAbsent)
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectFlag, observationInvalidLocator)
	}
}

func flagSpellingLocatorLabel(state *flagSpellingLocatorState) evaluationField {
	if state == nil {
		return semanticSubjectFlagSpelling
	}
	switch state.mode {
	case flagSpellingLocatorID:
		return evaluationText(fmt.Sprintf("flag spelling id %d", state.id))
	case flagSpellingLocatorText:
		return qualifySubject(flagLocatorLabel(state.flag), subjectNamed(evaluationText("spelling"), state.text))
	case flagSpellingLocatorIndex:
		return qualifySubject(flagLocatorLabel(state.flag), evaluationText(fmt.Sprintf("spelling %d", state.index)))
	default:
		return semanticSubjectFlagSpelling
	}
}

//nolint:dupl // Schema resolves flag-spelling relationships directly instead of projecting through artifact callbacks.
func resolveSchemaFlagSpellingLocator(state *flagSpellingLocatorState, schema *cli.Schema) locatorResolution[cli.FlagSpellingSchema] {
	var resolution locatorResolution[cli.FlagSpellingSchema]
	label := flagSpellingLocatorLabel(state)
	if schema == nil || state == nil || state.mode == flagSpellingLocatorInvalid {
		resolution = locatorResolution[cli.FlagSpellingSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectFlagSpelling, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == flagSpellingLocatorID {
		value, ok := schema.FlagSpellings[state.id]
		if !ok {
			resolution = locatorResolution[cli.FlagSpellingSchema]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectFlagSpelling, observationAbsent),
			}
			return resolution
		}
		resolution = locatorResolution[cli.FlagSpellingSchema]{
			status:  locatorResolutionFound,
			value:   value,
			subject: label,
		}
		return resolution
	}
	flag := resolveSchemaFlagLocator(state.flag, schema)
	if flag.status != locatorResolutionFound {
		resolution = locatorResolution[cli.FlagSpellingSchema]{
			status:  flag.status,
			subject: label,
			detail:  flag.detail,
		}
		return resolution
	}
	id, status, detail := selectSchemaFlagSpelling(state, flag.value.Spellings, schema, true)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.FlagSpellingSchema]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	value, ok := schema.FlagSpellings[id]
	if !ok {
		resolution = locatorResolution[cli.FlagSpellingSchema]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("flag-spelling relationship points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.FlagSpellingSchema]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

func resolveReferenceFlagSpellingLocator(state *flagSpellingLocatorState, reference cli.Reference) locatorResolution[cli.FlagSpellingReference] {
	var resolution locatorResolution[cli.FlagSpellingReference]
	label := flagSpellingLocatorLabel(state)
	if state == nil || state.mode == flagSpellingLocatorInvalid {
		resolution = locatorResolution[cli.FlagSpellingReference]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectFlagSpelling, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == flagSpellingLocatorID {
		value, ok := reference.FlagSpellings[state.id]
		if !ok {
			resolution = locatorResolution[cli.FlagSpellingReference]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectFlagSpelling, observationAbsent),
			}
			return resolution
		}
		resolution = locatorResolution[cli.FlagSpellingReference]{
			status:  locatorResolutionFound,
			value:   value,
			subject: label,
		}
		return resolution
	}
	flag := resolveReferenceFlagLocator(state.flag, reference)
	if flag.status != locatorResolutionFound {
		resolution = locatorResolution[cli.FlagSpellingReference]{
			status:  flag.status,
			subject: label,
			detail:  flag.detail,
		}
		return resolution
	}
	id, status, detail := selectReferenceFlagSpelling(state, flag.value.Spellings, reference, true)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.FlagSpellingReference]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	value, ok := reference.FlagSpellings[id]
	if !ok {
		resolution = locatorResolution[cli.FlagSpellingReference]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("flag-spelling relationship points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.FlagSpellingReference]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

func resolveInspectionFlagSpellingLocator(state *flagSpellingLocatorState, inspection cli.Inspection) locatorResolution[cli.FlagSpellingInspection] {
	var resolution locatorResolution[cli.FlagSpellingInspection]
	label := flagSpellingLocatorLabel(state)
	if state == nil || state.mode == flagSpellingLocatorInvalid {
		resolution = locatorResolution[cli.FlagSpellingInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectFlagSpelling, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == flagSpellingLocatorID {
		value, ok := inspection.FlagSpellings[state.id]
		if ok {
			resolution = locatorResolution[cli.FlagSpellingInspection]{
				status:  locatorResolutionFound,
				value:   value,
				subject: label,
			}
			return resolution
		}
		if inspection.Truncated {
			resolution = locatorResolution[cli.FlagSpellingInspection]{
				status:  locatorResolutionNotProvable,
				subject: label,
				detail:  observed(semanticSubjectInspection, observationTruncated),
			}
			return resolution
		}
		resolution = locatorResolution[cli.FlagSpellingInspection]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectFlagSpelling, observationAbsent),
		}
		return resolution
	}
	flag := resolveInspectionFlagLocator(state.flag, inspection)
	if flag.status != locatorResolutionFound {
		resolution = locatorResolution[cli.FlagSpellingInspection]{
			status:  flag.status,
			subject: label,
			detail:  flag.detail,
		}
		return resolution
	}
	id, status, detail := selectInspectionFlagSpelling(state, flag.value.Spellings, inspection, !inspection.Truncated)
	if status == locatorResolutionAbsent && inspection.Truncated {
		status = locatorResolutionNotProvable
		detail = observed(semanticSubjectInspection, observationTruncated)
	}
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.FlagSpellingInspection]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	value, ok := inspection.FlagSpellings[id]
	if !ok {
		resolution = locatorResolution[cli.FlagSpellingInspection]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("flag-spelling relationship points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.FlagSpellingInspection]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
//nolint:dupl // Separate public artifacts resolve their own flag-spelling relationships directly.
func selectSchemaFlagSpelling(state *flagSpellingLocatorState, ids []cli.FlagSpellingID, schema *cli.Schema, complete bool) (cli.FlagSpellingID, locatorResolutionStatus, evaluationField) {
	switch state.mode {
	case flagSpellingLocatorIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectFlagSpelling, observationInvalidLocator)
		}
		if state.index < len(ids) {
			return ids[state.index], locatorResolutionFound, evaluationText("")
		}
		if complete {
			return 0, locatorResolutionAbsent, observed(semanticSubjectFlagSpelling, observationAbsent)
		}
		return 0, locatorResolutionNotProvable, evaluationText("flag-spelling lookup domain is incomplete")
	case flagSpellingLocatorText:
		matches := make([]cli.FlagSpellingID, 0, 1)
		missingDetail := false
		for _, id := range ids {
			spelling, ok := schema.FlagSpellings[id]
			if !ok {
				missingDetail = true
				continue
			}
			if spelling.Spelling.Text == state.text {
				matches = append(matches, id)
			}
		}
		if len(matches) == 1 {
			return matches[0], locatorResolutionFound, evaluationText("")
		}
		if len(matches) > 1 {
			return 0, locatorResolutionAmbiguous, collectionCardinalityField{
				name:  flagSpellingAmbiguityDetail,
				count: len(matches),
			}
		}
		if missingDetail || !complete {
			return 0, locatorResolutionNotProvable, evaluationText("flag-spelling lookup domain is incomplete")
		}
		return 0, locatorResolutionAbsent, observed(semanticSubjectFlagSpelling, observationAbsent)
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectFlagSpelling, observationInvalidLocator)
	}
}

//nolint:dupl // Separate public artifacts resolve their own flag-spelling relationships directly.
func selectReferenceFlagSpelling(state *flagSpellingLocatorState, ids []cli.FlagSpellingID, reference cli.Reference, complete bool) (cli.FlagSpellingID, locatorResolutionStatus, evaluationField) {
	switch state.mode {
	case flagSpellingLocatorIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectFlagSpelling, observationInvalidLocator)
		}
		if state.index < len(ids) {
			return ids[state.index], locatorResolutionFound, evaluationText("")
		}
		if complete {
			return 0, locatorResolutionAbsent, observed(semanticSubjectFlagSpelling, observationAbsent)
		}
		return 0, locatorResolutionNotProvable, evaluationText("flag-spelling lookup domain is incomplete")
	case flagSpellingLocatorText:
		matches := make([]cli.FlagSpellingID, 0, 1)
		missingDetail := false
		for _, id := range ids {
			spelling, ok := reference.FlagSpellings[id]
			if !ok {
				missingDetail = true
				continue
			}
			if spelling.Spelling.Text == state.text {
				matches = append(matches, id)
			}
		}
		if len(matches) == 1 {
			return matches[0], locatorResolutionFound, evaluationText("")
		}
		if len(matches) > 1 {
			return 0, locatorResolutionAmbiguous, collectionCardinalityField{
				name:  flagSpellingAmbiguityDetail,
				count: len(matches),
			}
		}
		if missingDetail || !complete {
			return 0, locatorResolutionNotProvable, evaluationText("flag-spelling lookup domain is incomplete")
		}
		return 0, locatorResolutionAbsent, observed(semanticSubjectFlagSpelling, observationAbsent)
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectFlagSpelling, observationInvalidLocator)
	}
}

//nolint:dupl // Separate public artifacts resolve their own flag-spelling relationships directly.
func selectInspectionFlagSpelling(state *flagSpellingLocatorState, ids []cli.FlagSpellingID, inspection cli.Inspection, complete bool) (cli.FlagSpellingID, locatorResolutionStatus, evaluationField) {
	switch state.mode {
	case flagSpellingLocatorIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectFlagSpelling, observationInvalidLocator)
		}
		if state.index < len(ids) {
			return ids[state.index], locatorResolutionFound, evaluationText("")
		}
		if complete {
			return 0, locatorResolutionAbsent, observed(semanticSubjectFlagSpelling, observationAbsent)
		}
		return 0, locatorResolutionNotProvable, evaluationText("flag-spelling lookup domain is incomplete")
	case flagSpellingLocatorText:
		matches := make([]cli.FlagSpellingID, 0, 1)
		missingDetail := false
		for _, id := range ids {
			spelling, ok := inspection.FlagSpellings[id]
			if !ok {
				missingDetail = true
				continue
			}
			if spelling.Spelling.Text == state.text {
				matches = append(matches, id)
			}
		}
		if len(matches) == 1 {
			return matches[0], locatorResolutionFound, evaluationText("")
		}
		if len(matches) > 1 {
			return 0, locatorResolutionAmbiguous, collectionCardinalityField{
				name:  flagSpellingAmbiguityDetail,
				count: len(matches),
			}
		}
		if missingDetail || !complete {
			return 0, locatorResolutionNotProvable, evaluationText("flag-spelling lookup domain is incomplete")
		}
		return 0, locatorResolutionAbsent, observed(semanticSubjectFlagSpelling, observationAbsent)
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectFlagSpelling, observationInvalidLocator)
	}
}

func expectedSchemaFlagID(state *flagLocatorState, schema *cli.Schema) (cli.FlagID, locatorResolutionStatus, evaluationField) {
	if schema == nil || state == nil || state.mode == flagLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectFlag, observationInvalidLocator)
	}
	if state.mode == flagLocatorID {
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectFlag, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	}
	resolved := resolveSchemaFlagLocator(state, schema)
	if resolved.status != locatorResolutionFound {
		return 0, resolved.status, resolved.detail
	}
	return resolved.value.ID, locatorResolutionFound, evaluationText("")
}

func expectedReferenceFlagID(state *flagLocatorState, reference cli.Reference) (cli.FlagID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == flagLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectFlag, observationInvalidLocator)
	}
	if state.mode == flagLocatorID {
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectFlag, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	}
	resolved := resolveReferenceFlagLocator(state, reference)
	if resolved.status != locatorResolutionFound {
		return 0, resolved.status, resolved.detail
	}
	return resolved.value.ID, locatorResolutionFound, evaluationText("")
}

func evaluateFlagReplacementPresentationSchema(expectation flagReplacementPresentationExpectation, schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFlagLocator(expectation.locator, schema)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func evaluateFlagReplacementPresentationReference(expectation flagReplacementPresentationExpectation, reference cli.Reference) evaluationResult {
	resolution := resolveReferenceFlagLocator(expectation.locator, reference)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func evaluateFlagReplacementPresentationInspection(expectation flagReplacementPresentationExpectation, inspection cli.Inspection) evaluationResult {
	resolution := resolveInspectionFlagLocator(expectation.locator, inspection)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func evaluateFlagSpellingReplacementPresentationSchema(expectation flagSpellingReplacementPresentationExpectation, schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFlagSpellingLocator(expectation.locator, schema)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func evaluateFlagSpellingReplacementPresentationReference(expectation flagSpellingReplacementPresentationExpectation, reference cli.Reference) evaluationResult {
	resolution := resolveReferenceFlagSpellingLocator(expectation.locator, reference)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func evaluateFlagSpellingReplacementPresentationInspection(expectation flagSpellingReplacementPresentationExpectation, inspection cli.Inspection) evaluationResult {
	resolution := resolveInspectionFlagSpellingLocator(expectation.locator, inspection)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func flagIDLocator(id cli.FlagID) FlagLocator {
	return FlagLocator{state: &flagLocatorState{
		mode: flagLocatorID,
		id:   id,
	}}
}

func flagSpellingIDLocator(id cli.FlagSpellingID) FlagSpellingLocator {
	return FlagSpellingLocator{state: &flagSpellingLocatorState{
		mode: flagSpellingLocatorID,
		id:   id,
	}}
}

func tokenFlagValueFormExpectation(locator TokenLocator, form cli.FlagValueForm) InspectionExpectation {
	return tokenExpectation{
		locator:       locator.state,
		kind:          tokenExpectationFlagValueForm,
		flagValueForm: form,
	}
}

func (commandFlagsExpectation) schemaExpectation() {}

func (commandFlagsExpectation) referenceExpectation() {}

func (commandFlagsExpectation) inspectionExpectation() {}

//nolint:dupl // Typed collection proofs intentionally keep each public declaration family direct; generic resolver callbacks would recreate the rejected universal framework.
func (expectation commandFlagsExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	command := resolveSchemaCommandLocator(expectation.command, schema)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("flags exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("flags"))
	actual := command.value.Flags
	for _, id := range actual {
		if _, ok := schema.Flags[id]; !ok {
			return collectionClosureFailure(label, "flags", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.FlagID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveSchemaFlagLocator(state, schema)
		if outcome, failed := collectionResolutionFailure(label, "flags", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "flags", expected, actual)
}

//nolint:dupl // Typed collection proofs intentionally keep each public declaration family direct; generic resolver callbacks would recreate the rejected universal framework.
func (expectation commandFlagsExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	command := resolveReferenceCommandLocator(expectation.command, reference)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("flags exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("flags"))
	actual := command.value.Flags
	for _, id := range actual {
		if _, ok := reference.Flags[id]; !ok {
			return collectionClosureFailure(label, "flags", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.FlagID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveReferenceFlagLocator(state, reference)
		if outcome, failed := collectionResolutionFailure(label, "flags", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "flags", expected, actual)
}

func (expectation commandFlagsExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	command := resolveInspectionCommandLocator(expectation.command, inspection)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("flags exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("flags"))
	if inspection.Truncated {
		return collectionClosureFailure(label, "flags", observed(semanticSubjectInspection, observationTruncated))
	}
	if command.value.NamespaceCompleteness != cli.CommandNamespaceComplete {
		return collectionClosureFailure(label, "flags", evaluationText(fmt.Sprintf("command namespace is %v", command.value.NamespaceCompleteness)))
	}
	actual := command.value.Flags
	for _, id := range actual {
		if _, ok := inspection.Flags[id]; !ok {
			return collectionClosureFailure(label, "flags", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.FlagID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveInspectionFlagLocator(state, inspection)
		if outcome, failed := collectionResolutionFailure(label, "flags", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "flags", expected, actual)
}

func (flagAvailabilityExpectation) referenceExpectation() {}

func (flagAvailabilityExpectation) inspectionExpectation() {}

func (expectation flagAvailabilityExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagLocator(expectation.locator, reference)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "available", expectation.expected, resolution.value.Available)
}

func (expectation flagAvailabilityExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagLocator(expectation.locator, inspection)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "available", expectation.expected, resolution.value.Available)
}

func (flagVisibilityExpectation) referenceExpectation() {}

func (flagVisibilityExpectation) inspectionExpectation() {}

func (expectation flagVisibilityExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagLocator(expectation.locator, reference)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "visible", expectation.expected, resolution.value.Visible)
}

func (expectation flagVisibilityExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagLocator(expectation.locator, inspection)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "visible", expectation.expected, resolution.value.Visible)
}

func (flagSpellingAvailabilityExpectation) referenceExpectation() {}

func (flagSpellingAvailabilityExpectation) inspectionExpectation() {}

func (expectation flagSpellingAvailabilityExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagSpellingLocator(expectation.locator, reference)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "available", expectation.expected, resolution.value.Available)
}

func (expectation flagSpellingAvailabilityExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagSpellingLocator(expectation.locator, inspection)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "available", expectation.expected, resolution.value.Available)
}

func (flagSpellingVisibilityExpectation) referenceExpectation() {}

func (flagSpellingVisibilityExpectation) inspectionExpectation() {}

func (expectation flagSpellingVisibilityExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagSpellingLocator(expectation.locator, reference)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "visible", expectation.expected, resolution.value.Visible)
}

func (expectation flagSpellingVisibilityExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagSpellingLocator(expectation.locator, inspection)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "visible", expectation.expected, resolution.value.Visible)
}

func (flagOriginExpectation) schemaExpectation() {}

func (flagOriginExpectation) referenceExpectation() {}

func (flagOriginExpectation) inspectionExpectation() {}

func (expectation flagOriginExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFlagLocator(expectation.locator, schema)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (expectation flagOriginExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagLocator(expectation.locator, reference)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (expectation flagOriginExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagLocator(expectation.locator, inspection)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (flagReleaseLifecycleExpectation) schemaExpectation() {}

func (flagReleaseLifecycleExpectation) referenceExpectation() {}

func (flagReleaseLifecycleExpectation) inspectionExpectation() {}

func (expectation flagReleaseLifecycleExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFlagLocator(expectation.locator, schema)
	actual := resolution.value.Release
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, releaseLifecycleField{value: expectation.expected}, releaseLifecycleField{value: actual}, actual == expectation.expected)
}

func (expectation flagReleaseLifecycleExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagLocator(expectation.locator, reference)
	actual := resolution.value.Release
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, releaseLifecycleField{value: expectation.expected}, releaseLifecycleField{value: actual}, actual == expectation.expected)
}

func (expectation flagReleaseLifecycleExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagLocator(expectation.locator, inspection)
	actual := resolution.value.Release
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, releaseLifecycleField{value: expectation.expected}, releaseLifecycleField{value: actual}, actual == expectation.expected)
}

func (flagReplacementAbsenceExpectation) schemaExpectation() {}

func (flagReplacementAbsenceExpectation) referenceExpectation() {}

func (flagReplacementAbsenceExpectation) inspectionExpectation() {}

func (expectation flagReplacementAbsenceExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFlagLocator(expectation.locator, schema)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (expectation flagReplacementAbsenceExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagLocator(expectation.locator, reference)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (expectation flagReplacementAbsenceExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagLocator(expectation.locator, inspection)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (flagReplacementPresentationExpectation) schemaExpectation() {}

func (flagReplacementPresentationExpectation) referenceExpectation() {}

func (flagReplacementPresentationExpectation) inspectionExpectation() {}

func (expectation flagReplacementPresentationExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	return evaluateFlagReplacementPresentationSchema(expectation, schema)
}

func (expectation flagReplacementPresentationExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return evaluateFlagReplacementPresentationReference(expectation, reference)
}

func (expectation flagReplacementPresentationExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return evaluateFlagReplacementPresentationInspection(expectation, inspection)
}

func (flagReplacementTargetDeprecatedExpectation) schemaExpectation() {}

func (flagReplacementTargetDeprecatedExpectation) referenceExpectation() {}

func (flagReplacementTargetDeprecatedExpectation) inspectionExpectation() {}

func (expectation flagReplacementTargetDeprecatedExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFlagLocator(expectation.locator, schema)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (expectation flagReplacementTargetDeprecatedExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagLocator(expectation.locator, reference)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (expectation flagReplacementTargetDeprecatedExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagLocator(expectation.locator, inspection)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (flagSpellingOriginExpectation) schemaExpectation() {}

func (flagSpellingOriginExpectation) referenceExpectation() {}

func (flagSpellingOriginExpectation) inspectionExpectation() {}

func (expectation flagSpellingOriginExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFlagSpellingLocator(expectation.locator, schema)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (expectation flagSpellingOriginExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagSpellingLocator(expectation.locator, reference)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (expectation flagSpellingOriginExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagSpellingLocator(expectation.locator, inspection)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (flagSpellingReleaseLifecycleExpectation) schemaExpectation() {}

func (flagSpellingReleaseLifecycleExpectation) referenceExpectation() {}

func (flagSpellingReleaseLifecycleExpectation) inspectionExpectation() {}

func (expectation flagSpellingReleaseLifecycleExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFlagSpellingLocator(expectation.locator, schema)
	actual := resolution.value.Release
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, releaseLifecycleField{value: expectation.expected}, releaseLifecycleField{value: actual}, actual == expectation.expected)
}

func (expectation flagSpellingReleaseLifecycleExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagSpellingLocator(expectation.locator, reference)
	actual := resolution.value.Release
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, releaseLifecycleField{value: expectation.expected}, releaseLifecycleField{value: actual}, actual == expectation.expected)
}

func (expectation flagSpellingReleaseLifecycleExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagSpellingLocator(expectation.locator, inspection)
	actual := resolution.value.Release
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, releaseLifecycleField{value: expectation.expected}, releaseLifecycleField{value: actual}, actual == expectation.expected)
}

func (flagSpellingReplacementAbsenceExpectation) schemaExpectation() {}

func (flagSpellingReplacementAbsenceExpectation) referenceExpectation() {}

func (flagSpellingReplacementAbsenceExpectation) inspectionExpectation() {}

func (expectation flagSpellingReplacementAbsenceExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFlagSpellingLocator(expectation.locator, schema)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (expectation flagSpellingReplacementAbsenceExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagSpellingLocator(expectation.locator, reference)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (expectation flagSpellingReplacementAbsenceExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagSpellingLocator(expectation.locator, inspection)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (flagSpellingReplacementPresentationExpectation) schemaExpectation() {}

func (flagSpellingReplacementPresentationExpectation) referenceExpectation() {}

func (flagSpellingReplacementPresentationExpectation) inspectionExpectation() {}

func (expectation flagSpellingReplacementPresentationExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	return evaluateFlagSpellingReplacementPresentationSchema(expectation, schema)
}

func (expectation flagSpellingReplacementPresentationExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return evaluateFlagSpellingReplacementPresentationReference(expectation, reference)
}

func (expectation flagSpellingReplacementPresentationExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return evaluateFlagSpellingReplacementPresentationInspection(expectation, inspection)
}

func (flagSpellingReplacementTargetDeprecatedExpectation) schemaExpectation() {}

func (flagSpellingReplacementTargetDeprecatedExpectation) referenceExpectation() {}

func (flagSpellingReplacementTargetDeprecatedExpectation) inspectionExpectation() {}

func (expectation flagSpellingReplacementTargetDeprecatedExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFlagSpellingLocator(expectation.locator, schema)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (expectation flagSpellingReplacementTargetDeprecatedExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagSpellingLocator(expectation.locator, reference)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (expectation flagSpellingReplacementTargetDeprecatedExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagSpellingLocator(expectation.locator, inspection)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (flagExistenceExpectation) schemaExpectation() {}

func (flagExistenceExpectation) referenceExpectation() {}

func (flagExistenceExpectation) inspectionExpectation() {}

func (expectation flagExistenceExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	return declarationExistenceResult(resolveSchemaFlagLocator(expectation.locator, schema), expectation.expected)
}

func (expectation flagExistenceExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return declarationExistenceResult(resolveReferenceFlagLocator(expectation.locator, reference), expectation.expected)
}

func (expectation flagExistenceExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return declarationExistenceResult(resolveInspectionFlagLocator(expectation.locator, inspection), expectation.expected)
}

func (flagSpellingExistenceExpectation) schemaExpectation() {}

func (flagSpellingExistenceExpectation) referenceExpectation() {}

func (flagSpellingExistenceExpectation) inspectionExpectation() {}

func (expectation flagSpellingExistenceExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	return declarationExistenceResult(resolveSchemaFlagSpellingLocator(expectation.locator, schema), expectation.expected)
}

func (expectation flagSpellingExistenceExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return declarationExistenceResult(resolveReferenceFlagSpellingLocator(expectation.locator, reference), expectation.expected)
}

func (expectation flagSpellingExistenceExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return declarationExistenceResult(resolveInspectionFlagSpellingLocator(expectation.locator, inspection), expectation.expected)
}

func (flagRequiredExpectationValue) schemaExpectation() {}

func (flagRequiredExpectationValue) referenceExpectation() {}

func (flagRequiredExpectationValue) inspectionExpectation() {}

func (expectation flagRequiredExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFlagLocator(expectation.locator, schema)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("required"), true, resolution.value.Required)
}

func (expectation flagRequiredExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagLocator(expectation.locator, reference)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("required"), true, resolution.value.Required)
}

func (expectation flagRequiredExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagLocator(expectation.locator, inspection)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("required"), true, resolution.value.Required)
}

func (flagHiddenExpectationValue) schemaExpectation() {}

func (flagHiddenExpectationValue) referenceExpectation() {}

func (flagHiddenExpectationValue) inspectionExpectation() {}

func (expectation flagHiddenExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFlagLocator(expectation.locator, schema)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("hidden"), true, resolution.value.Hidden)
}

func (expectation flagHiddenExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagLocator(expectation.locator, reference)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("hidden"), true, resolution.value.Hidden)
}

func (expectation flagHiddenExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagLocator(expectation.locator, inspection)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("hidden"), true, resolution.value.Hidden)
}

func (flagSensitiveExpectationValue) schemaExpectation() {}

func (flagSensitiveExpectationValue) referenceExpectation() {}

func (flagSensitiveExpectationValue) inspectionExpectation() {}

func (expectation flagSensitiveExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFlagLocator(expectation.locator, schema)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, semanticRequirementSensitive, true, resolution.value.Sensitive)
}

func (expectation flagSensitiveExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagLocator(expectation.locator, reference)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, semanticRequirementSensitive, true, resolution.value.Sensitive)
}

func (expectation flagSensitiveExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagLocator(expectation.locator, inspection)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, semanticRequirementSensitive, true, resolution.value.Sensitive)
}

func (flagLifecycleExpectationValue) schemaExpectation() {}

func (flagLifecycleExpectationValue) referenceExpectation() {}

func (flagLifecycleExpectationValue) inspectionExpectation() {}

func (expectation flagLifecycleExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFlagLocator(expectation.locator, schema)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (expectation flagLifecycleExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagLocator(expectation.locator, reference)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (expectation flagLifecycleExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagLocator(expectation.locator, inspection)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (flagReplacementExpectationValue) schemaExpectation() {}

func (flagReplacementExpectationValue) referenceExpectation() {}

func (flagReplacementExpectationValue) inspectionExpectation() {}

func (expectation flagReplacementExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFlagLocator(expectation.locator, schema)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, schema, nil, cli.Inspection{})
}

func (expectation flagReplacementExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagLocator(expectation.locator, reference)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, nil, &reference, cli.Inspection{})
}

func (expectation flagReplacementExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagLocator(expectation.locator, inspection)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, nil, nil, inspection)
}

func (flagSpellingsExpectationValue) schemaExpectation() {}

func (flagSpellingsExpectationValue) referenceExpectation() {}

func (flagSpellingsExpectationValue) inspectionExpectation() {}

func (expectation flagSpellingsExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	flag := resolveSchemaFlagLocator(expectation.locator, schema)
	if flag.status != locatorResolutionFound {
		return declarationResolvedComparison(flag.status, flag.subject, flag.detail, evaluationText("exact spellings"), evaluationText(""), false)
	}
	ids := make([]cli.FlagSpellingID, len(expectation.expected))
	for index, state := range expectation.expected {
		resolution := resolveSchemaFlagSpellingLocator(state, schema)
		if resolution.status != locatorResolutionFound {
			return declarationResolvedComparison(resolution.status, flag.subject, resolution.detail, evaluationText("exact spellings"), evaluationText(""), false)
		}
		ids[index] = resolution.value.ID
	}
	if slices.Equal(flag.value.Spellings, ids) {
		return provedResult()
	}
	return orderedCollectionMismatch(flag.subject, "spellings", ids, flag.value.Spellings)
}

func (expectation flagSpellingsExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	flag := resolveReferenceFlagLocator(expectation.locator, reference)
	if flag.status != locatorResolutionFound {
		return declarationResolvedComparison(flag.status, flag.subject, flag.detail, evaluationText("exact spellings"), evaluationText(""), false)
	}
	ids := make([]cli.FlagSpellingID, len(expectation.expected))
	for index, state := range expectation.expected {
		resolution := resolveReferenceFlagSpellingLocator(state, reference)
		if resolution.status != locatorResolutionFound {
			return declarationResolvedComparison(resolution.status, flag.subject, resolution.detail, evaluationText("exact spellings"), evaluationText(""), false)
		}
		ids[index] = resolution.value.ID
	}
	if slices.Equal(flag.value.Spellings, ids) {
		return provedResult()
	}
	return orderedCollectionMismatch(flag.subject, "spellings", ids, flag.value.Spellings)
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func (expectation flagSpellingsExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	flag := resolveInspectionFlagLocator(expectation.locator, inspection)
	if flag.status != locatorResolutionFound {
		return declarationResolvedComparison(flag.status, flag.subject, flag.detail, evaluationText("exact spellings"), evaluationText(""), false)
	}
	if inspection.Truncated {
		return notProvableResult(flag.subject, evaluationText("exact spellings"), observed(semanticSubjectInspection, observationTruncated))
	}
	ids := make([]cli.FlagSpellingID, len(expectation.expected))
	for index, state := range expectation.expected {
		resolution := resolveInspectionFlagSpellingLocator(state, inspection)
		if resolution.status != locatorResolutionFound {
			return declarationResolvedComparison(resolution.status, flag.subject, resolution.detail, evaluationText("exact spellings"), evaluationText(""), false)
		}
		ids[index] = resolution.value.ID
	}
	if slices.Equal(flag.value.Spellings, ids) {
		return provedResult()
	}
	return orderedCollectionMismatch(flag.subject, "spellings", ids, flag.value.Spellings)
}

func (flagPresenceExpectationValue) inspectionExpectation() {}

func (expectation flagPresenceExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagLocator(expectation.locator, inspection)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("present=%t", expectation.expected)), expectation.expected, resolution.value.Present)
}

func (flagSpellingKindExpectationValue) schemaExpectation() {}

func (flagSpellingKindExpectationValue) referenceExpectation() {}

func (flagSpellingKindExpectationValue) inspectionExpectation() {}

func (expectation flagSpellingKindExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFlagSpellingLocator(expectation.locator, schema)
	actual := resolution.value.Spelling.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (expectation flagSpellingKindExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagSpellingLocator(expectation.locator, reference)
	actual := resolution.value.Spelling.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (expectation flagSpellingKindExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagSpellingLocator(expectation.locator, inspection)
	actual := resolution.value.Spelling.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (flagSpellingLifecycleExpectationValue) schemaExpectation() {}

func (flagSpellingLifecycleExpectationValue) referenceExpectation() {}

func (flagSpellingLifecycleExpectationValue) inspectionExpectation() {}

func (expectation flagSpellingLifecycleExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFlagSpellingLocator(expectation.locator, schema)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (expectation flagSpellingLifecycleExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagSpellingLocator(expectation.locator, reference)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (expectation flagSpellingLifecycleExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagSpellingLocator(expectation.locator, inspection)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (flagSpellingReplacementExpectationValue) schemaExpectation() {}

func (flagSpellingReplacementExpectationValue) referenceExpectation() {}

func (flagSpellingReplacementExpectationValue) inspectionExpectation() {}

func (expectation flagSpellingReplacementExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFlagSpellingLocator(expectation.locator, schema)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, schema, nil, cli.Inspection{})
}

func (expectation flagSpellingReplacementExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagSpellingLocator(expectation.locator, reference)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, nil, &reference, cli.Inspection{})
}

func (expectation flagSpellingReplacementExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagSpellingLocator(expectation.locator, inspection)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, nil, nil, inspection)
}

func (flagSpellingPresenceExpectationValue) inspectionExpectation() {}

func (expectation flagSpellingPresenceExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagSpellingLocator(expectation.locator, inspection)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("present=%t", expectation.expected)), expectation.expected, resolution.value.Present)
}

func (flagCorePropertyExpectation) schemaExpectation() {}

func (flagCorePropertyExpectation) referenceExpectation() {}

func (flagCorePropertyExpectation) inspectionExpectation() {}

func (expectation flagCorePropertyExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFlagLocator(expectation.locator, schema)
	return expectation.evaluate(resolution.status, resolution.subject, resolution.detail, resolution.value.Occurrences, resolution.value.ValuesPerOccurrence, resolution.value.Placement, resolution.value.ValueName, resolution.value.Description, resolution.value.LocaleSelector)
}

func (expectation flagCorePropertyExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagLocator(expectation.locator, reference)
	return expectation.evaluate(resolution.status, resolution.subject, resolution.detail, resolution.value.Occurrences, resolution.value.ValuesPerOccurrence, resolution.value.Placement, resolution.value.ValueName, resolution.value.Description, resolution.value.LocaleSelector)
}

func (expectation flagCorePropertyExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagLocator(expectation.locator, inspection)
	return expectation.evaluate(resolution.status, resolution.subject, resolution.detail, resolution.value.Occurrences, resolution.value.ValuesPerOccurrence, resolution.value.Placement, resolution.value.ValueName, resolution.value.Description, resolution.value.LocaleSelector)
}

func (expectation flagCorePropertyExpectation) evaluate(status locatorResolutionStatus, subject, detail evaluationField, occurrences, valuesPerOccurrence cli.CountRange, placement cli.FlagPlacement, valueName, description string, localeSelector bool) evaluationResult {
	switch expectation.kind {
	case flagCorePropertyOccurrences:
		return declarationResolvedComparison(status, subject, detail, evaluationText("occurrences"), evaluationText(fmt.Sprint(occurrences)), occurrences == expectation.count)
	case flagCorePropertyValuesPerOccurrence:
		return declarationResolvedComparison(status, subject, detail, evaluationText("values-per-occurrence"), evaluationText(fmt.Sprint(valuesPerOccurrence)), valuesPerOccurrence == expectation.count)
	case flagCorePropertyPlacement:
		return declarationResolvedComparison(status, subject, detail, evaluationText("placement"), evaluationText(fmt.Sprint(placement)), placement == expectation.placement)
	case flagCorePropertyValueName:
		return declarationResolvedComparison(status, subject, detail, evaluationText("value-name"), evaluationText(valueName), valueName == expectation.text)
	case flagCorePropertyDescription:
		return declarationResolvedComparison(status, subject, detail, evaluationText("description"), evaluationText(description), description == expectation.text)
	case flagCorePropertyLocaleSelector:
		return resolvedBoolProperty(status, subject, detail, evaluationText("locale-selector"), true, localeSelector)
	default:
		panic("clitest: invalid flag core property expectation")
	}
}

func (flagSpellingCorePropertyExpectation) schemaExpectation() {}

func (flagSpellingCorePropertyExpectation) referenceExpectation() {}

func (flagSpellingCorePropertyExpectation) inspectionExpectation() {}

func (expectation flagSpellingCorePropertyExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFlagSpellingLocator(expectation.locator, schema)
	return expectation.evaluate(resolution.status, resolution.subject, resolution.detail, resolution.value.Hidden, resolution.value.Spelling.Sensitive, resolution.value.Spelling.ValueForms)
}

func (expectation flagSpellingCorePropertyExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFlagSpellingLocator(expectation.locator, reference)
	return expectation.evaluate(resolution.status, resolution.subject, resolution.detail, resolution.value.Hidden, resolution.value.Spelling.Sensitive, resolution.value.Spelling.ValueForms)
}

func (expectation flagSpellingCorePropertyExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFlagSpellingLocator(expectation.locator, inspection)
	return expectation.evaluate(resolution.status, resolution.subject, resolution.detail, resolution.value.Hidden, resolution.value.Spelling.Sensitive, resolution.value.Spelling.ValueForms)
}

func (expectation flagSpellingCorePropertyExpectation) evaluate(status locatorResolutionStatus, subject, detail evaluationField, hidden, sensitive bool, valueForms []cli.FlagValueForm) evaluationResult {
	switch expectation.kind {
	case flagSpellingCorePropertyHidden:
		return resolvedBoolProperty(status, subject, detail, evaluationText("hidden"), true, hidden)
	case flagSpellingCorePropertySensitive:
		return resolvedBoolProperty(status, subject, detail, semanticRequirementSensitive, true, sensitive)
	case flagSpellingCorePropertyValueForms:
		if status != locatorResolutionFound {
			return declarationResolvedComparison(status, subject, detail, evaluationText("exact value forms in order"), evaluationText(""), false)
		}
		if slices.Equal(valueForms, expectation.valueForms) {
			return provedResult()
		}
		return orderedCollectionMismatch(subject, "value forms", expectation.valueForms, valueForms)
	default:
		panic("clitest: invalid flag-spelling core property expectation")
	}
}

func (flagSpellingField) evaluationField() {}

func (helpFlagExpectation) helpPresentationExpectation() {}

func (expectation helpFlagExpectation) evaluateHelp(page cli.HelpPage) evaluationResult {
	flag, status, detail := resolveHelpFlag(expectation.locator, page)
	subject := evaluationText("help flag")
	if expectation.property == helpFlagExists {
		return evaluateHelpExistence(subject, expectation.exists, status)
	}
	if expectation.property == helpFlagActionEntry && (expectation.actionEntry == nil || expectation.actionEntry.mode != actionEntryLocatorID) {
		return disprovedResult(subject, evaluationText("exact bound action-entry identity"), evaluationText("expected locator is not exact-identity-bound"))
	}
	if status != locatorResolutionFound {
		return helpResolutionFailure(subject, expectation.flagExpected(), detail)
	}
	switch expectation.property {
	case helpFlagOrigin:
		if flag.Origin == expectation.origin {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "origin", evaluationText(fmt.Sprint(expectation.origin)), evaluationText(fmt.Sprint(flag.Origin)))
	case helpFlagOrdinary:
		if flag.ActionEntry == 0 {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "ordinary", evaluationBool(true), evaluationBool(false))
	case helpFlagAction:
		if flag.ActionEntry != 0 {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "action", evaluationBool(true), evaluationBool(false))
	case helpFlagActionEntryPresent:
		if flag.ActionEntry != 0 {
			return provedResult()
		}
		return disprovedResult(subject, evaluationText("action-entry identity present"), evaluationText("action-entry identity is zero"))
	case helpFlagActionEntry:
		if flag.ActionEntry == expectation.actionEntry.id {
			return provedResult()
		}
		return disprovedResult(subject, evaluationText("action-entry matches expected identity"), evaluationText("action-entry has different identity"))
	case helpFlagSpellingCount:
		if len(flag.FlagSpellings) == expectation.count {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "spelling-count", evaluationText(strconv.Itoa(expectation.count)), evaluationText(strconv.Itoa(len(flag.FlagSpellings))))
	case helpFlagActionSpelling:
		if equalHelpFlagSpelling(flag.ActionSpelling, expectation.spelling) {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "action-spelling", flagSpellingField{value: expectation.spelling}, flagSpellingField{value: flag.ActionSpelling})
	case helpFlagDescription:
		if flag.Description == expectation.text {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "description", evaluationText(expectation.text), evaluationText(flag.Description))
	case helpFlagOwnerPath:
		if slices.Equal(flag.OwnerPath, expectation.path) {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "owner-path", stringCollectionField{values: expectation.path}, stringCollectionField{values: flag.OwnerPath})
	case helpFlagPlacement:
		if flag.Placement == expectation.placement {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "placement", evaluationText(fmt.Sprint(expectation.placement)), evaluationText(fmt.Sprint(flag.Placement)))
	case helpFlagHidden:
		if flag.Hidden == expectation.boolean {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "hidden", evaluationBool(expectation.boolean), evaluationBool(flag.Hidden))
	case helpFlagSensitive:
		if flag.Sensitive == expectation.boolean {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "sensitive", evaluationBool(expectation.boolean), evaluationBool(flag.Sensitive))
	case helpFlagLifecycle:
		if flag.LifecycleState == expectation.lifecycle {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "lifecycle", evaluationText(fmt.Sprint(expectation.lifecycle)), evaluationText(fmt.Sprint(flag.LifecycleState)))
	case helpFlagValueDomainEntryCount:
		actual := len(valueDomainUseEntries(flag.ValueDomainUses, expectation.domainKind))
		if actual == expectation.count {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "value-domain-entry-count", evaluationText(strconv.Itoa(expectation.count)), evaluationText(strconv.Itoa(actual)))
	default:
		panic("clitest: invalid Help flag expectation")
	}
}

func (expectation helpFlagExpectation) flagExpected() evaluationField {
	switch expectation.property {
	case helpFlagOrigin:
		return property(evaluationText("origin"), evaluationText(fmt.Sprint(expectation.origin)))
	case helpFlagOrdinary:
		return property(evaluationText("ordinary"), evaluationText("true"))
	case helpFlagAction:
		return property(evaluationText("action"), evaluationText("true"))
	case helpFlagActionEntryPresent:
		return evaluationText("action-entry identity present")
	case helpFlagActionEntry:
		return evaluationText("action-entry matches expected identity")
	case helpFlagSpellingCount:
		return property(evaluationText("spelling-count"), evaluationText(strconv.Itoa(expectation.count)))
	case helpFlagActionSpelling:
		return property(evaluationText("action-spelling"), flagSpellingField{value: expectation.spelling})
	case helpFlagDescription:
		return property(evaluationText("description"), evaluationText(expectation.text))
	case helpFlagOwnerPath:
		return property(evaluationText("owner-path"), stringCollectionField{values: expectation.path})
	case helpFlagPlacement:
		return property(evaluationText("placement"), evaluationText(fmt.Sprint(expectation.placement)))
	case helpFlagHidden:
		return property(evaluationText("hidden"), evaluationText(strconv.FormatBool(expectation.boolean)))
	case helpFlagSensitive:
		return property(semanticRequirementSensitive, evaluationText(strconv.FormatBool(expectation.boolean)))
	case helpFlagLifecycle:
		return property(evaluationText("lifecycle"), evaluationText(fmt.Sprint(expectation.lifecycle)))
	case helpFlagValueDomainEntryCount:
		return property(evaluationText("value-domain-entry-count"), evaluationText(strconv.Itoa(expectation.count)))
	default:
		return evaluationText("valid flag expectation")
	}
}

func (helpFlagSpellingExpectation) helpPresentationExpectation() {}

func (expectation helpFlagSpellingExpectation) evaluateHelp(page cli.HelpPage) evaluationResult {
	spelling, status, detail := resolveHelpFlagSpelling(expectation.locator, page)
	subject := evaluationText("help flag spelling")
	if expectation.property == helpFlagSpellingExists {
		return evaluateHelpExistence(subject, expectation.exists, status)
	}
	if status != locatorResolutionFound {
		return helpResolutionFailure(subject, expectation.flagSpellingExpected(), detail)
	}
	switch expectation.property {
	case helpFlagSpellingValue:
		if equalHelpFlagSpelling(spelling.Spelling, expectation.spelling) {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "spelling", flagSpellingField{value: expectation.spelling}, flagSpellingField{value: spelling.Spelling})
	case helpFlagSpellingLifecycle:
		if spelling.LifecycleState == expectation.lifecycle {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "lifecycle", evaluationText(fmt.Sprint(expectation.lifecycle)), evaluationText(fmt.Sprint(spelling.LifecycleState)))
	case helpFlagSpellingNotReplaced:
		if spelling.Replacement == 0 {
			return provedResult()
		}
		return disprovedResult(subject, evaluationText("replacement absent"), evaluationText("replacement relation is present"))
	default:
		panic("clitest: invalid Help flag-spelling expectation")
	}
}

func (expectation helpFlagSpellingExpectation) flagSpellingExpected() evaluationField {
	switch expectation.property {
	case helpFlagSpellingValue:
		return property(evaluationText("spelling"), flagSpellingField{value: expectation.spelling})
	case helpFlagSpellingLifecycle:
		return property(evaluationText("lifecycle"), evaluationText(fmt.Sprint(expectation.lifecycle)))
	case helpFlagSpellingNotReplaced:
		return evaluationText("replacement absent")
	default:
		return evaluationText("valid flag-spelling expectation")
	}
}

func (staticHelpFlagGroupExpectation) schemaExpectation() {}

func (staticHelpFlagGroupExpectation) referenceExpectation() {}

func (expectation staticHelpFlagGroupExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolved := resolveSchemaHelpFlagGroup(expectation.locator, schema)
	return expectation.evaluateSchemaResolved(resolved, schema)
}

func (expectation staticHelpFlagGroupExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolved := resolveReferenceHelpFlagGroup(expectation.locator, reference)
	return expectation.evaluateReferenceResolved(resolved, reference)
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func (expectation staticHelpFlagGroupExpectation) evaluateSchemaResolved(resolved helpFlagGroupResolution, schema *cli.Schema) evaluationResult {
	switch expectation.property {
	case staticHelpFlagGroupExpectationExists:
		return helpGroupExistenceOutcome(resolved.status, resolved.label, resolved.actual, fmt.Sprintf("exists=%t", expectation.exists), expectation.exists)
	case staticHelpFlagGroupExpectationKind:
		return helpGroupComparableOutcome(resolved.status, resolved.label, resolved.actual, "kind", evaluationText(fmt.Sprint(expectation.kind)), evaluationText(fmt.Sprint(resolved.group.Kind)), resolved.group.Kind == expectation.kind)
	case staticHelpFlagGroupExpectationTitle:
		return helpGroupComparableOutcome(resolved.status, resolved.label, resolved.actual, "title", evaluationText(expectation.title), evaluationText(resolved.group.Title), resolved.group.Title == expectation.title)
	case staticHelpFlagGroupExpectationOrder:
		return helpGroupComparableOutcome(resolved.status, resolved.label, resolved.actual, "order", evaluationText(fmt.Sprint(expectation.order)), evaluationText(fmt.Sprint(resolved.group.Order)), resolved.group.Order == expectation.order)
	case staticHelpFlagGroupExpectationProfile:
		if resolved.status != locatorResolutionFound {
			return outcomeForValueStatus(resolved.status, qualifySubject(resolved.label, evaluationText("profile")), evaluationText("profile"), resolved.actual)
		}
		return compareHelpGroupProfileSchema(resolved.label, resolved.group.Profile, expectation.profile, expectation.noProfile, schema)
	case staticHelpFlagGroupExpectationFlags:
		if resolved.status != locatorResolutionFound {
			return outcomeForValueStatus(resolved.status, qualifySubject(resolved.label, evaluationText("flags")), evaluationText("exact flags"), resolved.actual)
		}
		ids, status, detail := resolveHelpFlagIDsSchema(expectation.flags, schema)
		if status != locatorResolutionFound {
			return outcomeForValueStatus(status, qualifySubject(resolved.label, evaluationText("flags")), evaluationText("valid expected flags"), detail)
		}
		return compareHelpFlagIDs(resolved.label, ids, resolved.group.Flags)
	default:
		return disprovedResult(semanticSubjectHelpFlagGroup, evaluationText("valid expectation"), observed(semanticSubjectLocator, observationInvalidLocator))
	}
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func (expectation staticHelpFlagGroupExpectation) evaluateReferenceResolved(resolved helpFlagGroupResolution, reference cli.Reference) evaluationResult {
	switch expectation.property {
	case staticHelpFlagGroupExpectationExists:
		return helpGroupExistenceOutcome(resolved.status, resolved.label, resolved.actual, fmt.Sprintf("exists=%t", expectation.exists), expectation.exists)
	case staticHelpFlagGroupExpectationKind:
		return helpGroupComparableOutcome(resolved.status, resolved.label, resolved.actual, "kind", evaluationText(fmt.Sprint(expectation.kind)), evaluationText(fmt.Sprint(resolved.group.Kind)), resolved.group.Kind == expectation.kind)
	case staticHelpFlagGroupExpectationTitle:
		return helpGroupComparableOutcome(resolved.status, resolved.label, resolved.actual, "title", evaluationText(expectation.title), evaluationText(resolved.group.Title), resolved.group.Title == expectation.title)
	case staticHelpFlagGroupExpectationOrder:
		return helpGroupComparableOutcome(resolved.status, resolved.label, resolved.actual, "order", evaluationText(fmt.Sprint(expectation.order)), evaluationText(fmt.Sprint(resolved.group.Order)), resolved.group.Order == expectation.order)
	case staticHelpFlagGroupExpectationProfile:
		if resolved.status != locatorResolutionFound {
			return outcomeForValueStatus(resolved.status, qualifySubject(resolved.label, evaluationText("profile")), evaluationText("profile"), resolved.actual)
		}
		return compareHelpGroupProfileReference(resolved.label, resolved.group.Profile, expectation.profile, expectation.noProfile, reference)
	case staticHelpFlagGroupExpectationFlags:
		if resolved.status != locatorResolutionFound {
			return outcomeForValueStatus(resolved.status, qualifySubject(resolved.label, evaluationText("flags")), evaluationText("exact flags"), resolved.actual)
		}
		ids, status, detail := resolveHelpFlagIDsReference(expectation.flags, reference)
		if status != locatorResolutionFound {
			return outcomeForValueStatus(status, qualifySubject(resolved.label, evaluationText("flags")), evaluationText("valid expected flags"), detail)
		}
		return compareHelpFlagIDs(resolved.label, ids, resolved.group.Flags)
	default:
		return disprovedResult(semanticSubjectHelpFlagGroup, evaluationText("valid expectation"), observed(semanticSubjectLocator, observationInvalidLocator))
	}
}
