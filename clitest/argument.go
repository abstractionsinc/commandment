package clitest

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/abstractionsinc/commandment/cli"
)

// ArgumentFormLocator identifies one argument form for clitest semantic expectation.
// Its zero value is invalid.
type ArgumentFormLocator struct {
	_     [0]func()
	state *argumentFormLocatorState
}

// ArgumentTermLocator identifies one argument term for clitest semantic expectation.
// Its zero value is invalid.
type ArgumentTermLocator struct {
	_     [0]func()
	state *argumentTermLocatorState
}

// ArgumentFormByID returns a locator for the argument form with the exact public ArgumentFormID
// within the observation being evaluated.
func ArgumentFormByID(id cli.ArgumentFormID) ArgumentFormLocator { return argumentFormIDLocator(id) }

// ArgumentTermByID returns a locator for the argument term with the exact public ArgumentTermID
// within the observation being evaluated.
func ArgumentTermByID(id cli.ArgumentTermID) ArgumentTermLocator { return argumentTermIDLocator(id) }

func (ArgumentFormLocator) declarationIdentityLocator() {}

func (ArgumentFormLocator) diagnosticSubject() {}

func (ArgumentFormLocator) noticeSubject() {}

func (ArgumentFormLocator) tokenOwner() {}

func (ArgumentFormLocator) completionTargetSubject() {}

// Term returns a locator for the uniquely resolved named term in the located argument form.
func (locator ArgumentFormLocator) Term(name string) ArgumentTermLocator {
	return argumentTermNameLocator(locator, name)
}

// Literal returns a locator for the uniquely resolved literal term in the located argument form.
func (locator ArgumentFormLocator) Literal(value string) ArgumentTermLocator {
	return argumentTermLiteralLocator(locator, value)
}

// TermAt returns a locator for the argument term at index in the located form's authoritative
// public term order.
func (locator ArgumentFormLocator) TermAt(index int) ArgumentTermLocator {
	return argumentTermAtLocator(locator, index)
}

// Exists returns an expectation that checks the located argument form exists. Positive existence
// may be satisfied by truthful partial output when the subject is present.
func (locator ArgumentFormLocator) Exists() DeclarationExpectation {
	expectation := argumentFormExistenceExpectation{
		locator:  locator.state,
		expected: true,
	}
	return expectation
}

// DoesNotExist returns an expectation that checks the located argument form does not exist; an
// incomplete observation cannot establish absence.
func (locator ArgumentFormLocator) DoesNotExist() DeclarationExpectation {
	expectation := argumentFormExistenceExpectation{
		locator:  locator.state,
		expected: false,
	}
	return expectation
}

// Lifecycle returns an expectation that checks the located argument form has the expected
// effective lifecycle state; this is distinct from release-lifecycle metadata.
func (locator ArgumentFormLocator) Lifecycle(v cli.LifecycleState) DeclarationExpectation {
	value := argumentFormLifecycleExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// ReplacedBy returns an expectation that checks the located argument form replacement points to
// the exact declaration identified by v, using public declaration identity rather than
// presentation text.
func (locator ArgumentFormLocator) ReplacedBy(v DeclarationIdentityLocator) DeclarationExpectation {
	value := argumentFormReplacementExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Reachability returns an expectation that checks the located argument form has the expected
// public reachability state.
func (locator ArgumentFormLocator) Reachability(v cli.Reachability) DeclarationExpectation {
	value := argumentFormReachabilityExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Available returns an expectation that checks the located argument form is available in the
// applicable public observation.
func (locator ArgumentFormLocator) Available() EffectiveDeclarationExpectation {
	expectation := argumentFormAvailabilityExpectation{
		locator:  locator.state,
		expected: true,
	}
	return expectation
}

// Unavailable returns an expectation that checks the located argument form is unavailable in the
// applicable public observation.
func (locator ArgumentFormLocator) Unavailable() EffectiveDeclarationExpectation {
	expectation := argumentFormAvailabilityExpectation{
		locator:  locator.state,
		expected: false,
	}
	return expectation
}

// Visible returns an expectation that checks the located argument form is visible in the
// applicable public observation.
func (locator ArgumentFormLocator) Visible() EffectiveDeclarationExpectation {
	expectation := argumentFormVisibilityExpectation{
		locator:  locator.state,
		expected: true,
	}
	return expectation
}

// NotVisible returns an expectation that checks the located argument form is not visible in the
// applicable public observation.
func (locator ArgumentFormLocator) NotVisible() EffectiveDeclarationExpectation {
	expectation := argumentFormVisibilityExpectation{
		locator:  locator.state,
		expected: false,
	}
	return expectation
}

// TermsExactly returns an expectation that checks the located argument form's argument terms are
// exactly the supplied locators in authoritative public order. Exactness is not inferred from an
// incomplete observation.
func (locator ArgumentFormLocator) TermsExactly(v ...ArgumentTermLocator) DeclarationExpectation {
	expected := make([]*argumentTermLocatorState, len(v))
	for index, locator := range v {
		expected[index] = locator.state
	}
	value := argumentFormTermsExpectationValue{
		locator:  locator.state,
		expected: expected,
	}
	return value
}

// GrammarProfile returns a static declaration expectation that checks the argument form refers
// to the same public grammar-profile identity as the expected locator.
func (locator ArgumentFormLocator) GrammarProfile(expected GrammarProfileLocator) StaticDeclarationExpectation {
	expectation := argumentFormGrammarProfileExpectation{
		form:    locator.state,
		profile: expected.state,
	}
	return expectation
}

// Origin returns an expectation that checks the located argument form has the
// framework-established DeclarationOrigin expected.
func (locator ArgumentFormLocator) Origin(expected cli.DeclarationOrigin) DeclarationExpectation {
	expectation := argumentFormOriginExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// ReleaseLifecycle returns an expectation that checks the located argument form has the exact
// public release-lifecycle metadata expected.
func (locator ArgumentFormLocator) ReleaseLifecycle(expected cli.ReleaseLifecycle) DeclarationExpectation {
	expectation := argumentFormReleaseLifecycleExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// NotReplaced returns an expectation that checks the located argument form has no public
// declaration-replacement relationship. Incomplete observation cannot manufacture replacement
// absence.
func (locator ArgumentFormLocator) NotReplaced() DeclarationExpectation {
	expectation := argumentFormReplacementAbsenceExpectation{
		locator: locator.state,
	}
	return expectation
}

// ReplacementPresentation returns an expectation that checks the located argument form replacement
// carries the expected public presentation text.
func (locator ArgumentFormLocator) ReplacementPresentation(expected string) DeclarationExpectation {
	expectation := argumentFormReplacementPresentationExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// ReplacementTargetDeprecatedDuringMigration returns an expectation that checks the located
// argument form replacement carries the expected migration-deprecation marker.
func (locator ArgumentFormLocator) ReplacementTargetDeprecatedDuringMigration(expected bool) DeclarationExpectation {
	expectation := argumentFormReplacementTargetDeprecatedExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

func (ArgumentTermLocator) declarationIdentityLocator() {}

func (ArgumentTermLocator) diagnosticSubject() {}

func (ArgumentTermLocator) noticeSubject() {}

func (ArgumentTermLocator) tokenOwner() {}

func (ArgumentTermLocator) completionTargetSubject() {}

// Value returns a locator for the public value associated with the located argument term.
func (locator ArgumentTermLocator) Value() ValueLocator { return argumentTermValueLocator(locator) }

// Exists returns an expectation that checks the located argument term exists. Positive existence
// may be satisfied by truthful partial output when the subject is present.
func (locator ArgumentTermLocator) Exists() DeclarationExpectation {
	expectation := argumentTermExistenceExpectation{
		locator:  locator.state,
		expected: true,
	}
	return expectation
}

// DoesNotExist returns an expectation that checks the located argument term does not exist; an
// incomplete observation cannot establish absence.
func (locator ArgumentTermLocator) DoesNotExist() DeclarationExpectation {
	expectation := argumentTermExistenceExpectation{
		locator:  locator.state,
		expected: false,
	}
	return expectation
}

// Kind returns an expectation that checks the located argument term has the expected public kind.
func (locator ArgumentTermLocator) Kind(v cli.ArgumentTermKind) DeclarationExpectation {
	value := argumentTermKindExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Hidden returns an expectation that checks the located argument term is marked hidden by the
// public Commandment observation.
func (locator ArgumentTermLocator) Hidden() DeclarationExpectation {
	value := argumentTermHiddenExpectationValue{
		locator: locator.state,
	}
	return value
}

// Sensitive returns an expectation that checks the located argument term is marked sensitive.
// Expectation failure preserves Commandment redaction and does not expose secret material.
func (locator ArgumentTermLocator) Sensitive() DeclarationExpectation {
	value := argumentTermSensitiveExpectationValue{
		locator: locator.state,
	}
	return value
}

// Lifecycle returns an expectation that checks the located argument term has the expected
// effective lifecycle state; this is distinct from release-lifecycle metadata.
func (locator ArgumentTermLocator) Lifecycle(v cli.LifecycleState) DeclarationExpectation {
	value := argumentTermLifecycleExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// ReplacedBy returns an expectation that checks the located argument term replacement points to
// the exact declaration identified by v, using public declaration identity rather than
// presentation text.
func (locator ArgumentTermLocator) ReplacedBy(v DeclarationIdentityLocator) DeclarationExpectation {
	value := argumentTermReplacementExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Available returns an expectation that checks the located argument term is available in the
// applicable public observation.
func (locator ArgumentTermLocator) Available() EffectiveDeclarationExpectation {
	expectation := argumentTermAvailabilityExpectation{
		locator:  locator.state,
		expected: true,
	}
	return expectation
}

// Unavailable returns an expectation that checks the located argument term is unavailable in the
// applicable public observation.
func (locator ArgumentTermLocator) Unavailable() EffectiveDeclarationExpectation {
	expectation := argumentTermAvailabilityExpectation{
		locator:  locator.state,
		expected: false,
	}
	return expectation
}

// Visible returns an expectation that checks the located argument term is visible in the
// applicable public observation.
func (locator ArgumentTermLocator) Visible() EffectiveDeclarationExpectation {
	expectation := argumentTermVisibilityExpectation{
		locator:  locator.state,
		expected: true,
	}
	return expectation
}

// NotVisible returns an expectation that checks the located argument term is not visible in the
// applicable public observation.
func (locator ArgumentTermLocator) NotVisible() EffectiveDeclarationExpectation {
	expectation := argumentTermVisibilityExpectation{
		locator:  locator.state,
		expected: false,
	}
	return expectation
}

// Present returns an expectation that checks the located argument term is present in the analyzed
// invocation, independently of whether its declaration merely exists.
func (locator ArgumentTermLocator) Present() InspectionExpectation {
	value := argumentTermPresenceExpectationValue{
		locator:  locator.state,
		expected: true,
	}
	return value
}

// Absent returns an expectation that checks the located argument term is absent from the analyzed
// invocation; incomplete invocation evidence cannot be treated as fabricated absence.
func (locator ArgumentTermLocator) Absent() InspectionExpectation {
	value := argumentTermPresenceExpectationValue{
		locator:  locator.state,
		expected: false,
	}
	return value
}

// HyphenValues returns a reusable declaration expectation that checks the public hyphen-value
// mode exposed by the located argument term.
func (locator ArgumentTermLocator) HyphenValues(expected cli.HyphenValueMode) DeclarationExpectation {
	expectation := argumentTermCorePropertyExpectation{
		locator:      locator.state,
		kind:         argumentTermCorePropertyHyphenValues,
		hyphenValues: expected,
	}
	return expectation
}

// Description returns a reusable declaration expectation that checks the public description
// exposed by the located argument term.
func (locator ArgumentTermLocator) Description(expected string) DeclarationExpectation {
	expectation := argumentTermCorePropertyExpectation{
		locator: locator.state,
		kind:    argumentTermCorePropertyDescription,
		text:    expected,
	}
	return expectation
}

// Minimal returns a reusable declaration expectation that checks the public minimal-form state
// exposed by the located argument term.
func (locator ArgumentTermLocator) Minimal() DeclarationExpectation {
	expectation := argumentTermCorePropertyExpectation{
		locator: locator.state,
		kind:    argumentTermCorePropertyMinimal,
	}
	return expectation
}

// EndOfOptions returns a reusable declaration expectation that checks the public end-of-options
// state exposed by the located argument term.
func (locator ArgumentTermLocator) EndOfOptions() DeclarationExpectation {
	expectation := argumentTermCorePropertyExpectation{
		locator: locator.state,
		kind:    argumentTermCorePropertyEndOfOptions,
	}
	return expectation
}

// SourceSensitive returns a reusable declaration expectation that checks the public
// source-sensitive state exposed by the located argument term.
func (locator ArgumentTermLocator) SourceSensitive() DeclarationExpectation {
	expectation := argumentTermCorePropertyExpectation{
		locator: locator.state,
		kind:    argumentTermCorePropertySourceSensitive,
	}
	return expectation
}

// HasTokenLanguage returns a reusable declaration expectation that checks the public
// token-language presence exposed by the located argument term.
func (locator ArgumentTermLocator) HasTokenLanguage() DeclarationExpectation {
	expectation := argumentTermCorePropertyExpectation{
		locator: locator.state,
		kind:    argumentTermCorePropertyHasTokenLanguage,
	}
	return expectation
}

// TokenLanguageFragmentsExactly returns a reusable declaration expectation that checks the
// complete token-language fragment sequence exactly in authoritative public order.
func (locator ArgumentTermLocator) TokenLanguageFragmentsExactly(expected ...string) DeclarationExpectation {
	expectation := argumentTermCorePropertyExpectation{
		locator:   locator.state,
		kind:      argumentTermCorePropertyTokenLanguageFragments,
		fragments: slices.Clone(expected),
	}
	return expectation
}

// Completion returns a locator that follows the public completion relationship from the located
// argument term. The returned locator carries locator coordinates, not observed state.
func (locator ArgumentTermLocator) Completion() CompletionDeclarationLocator {
	state := &completionDeclarationLocatorState{
		mode: completionDeclarationLocatorArgumentTerm,
		term: locator.state,
	}
	completionDeclarationLocator := CompletionDeclarationLocator{
		state: state,
	}
	return completionDeclarationLocator
}

// Origin returns an expectation that checks the located argument term has the
// framework-established DeclarationOrigin expected.
func (locator ArgumentTermLocator) Origin(expected cli.DeclarationOrigin) DeclarationExpectation {
	expectation := argumentTermOriginExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// ReleaseLifecycle returns an expectation that checks the located argument term has the exact
// public release-lifecycle metadata expected.
func (locator ArgumentTermLocator) ReleaseLifecycle(expected cli.ReleaseLifecycle) DeclarationExpectation {
	expectation := argumentTermReleaseLifecycleExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// NotReplaced returns an expectation that checks the located argument term has no public
// declaration-replacement relationship. Incomplete observation cannot manufacture replacement
// absence.
func (locator ArgumentTermLocator) NotReplaced() DeclarationExpectation {
	expectation := argumentTermReplacementAbsenceExpectation{
		locator: locator.state,
	}
	return expectation
}

// ReplacementPresentation returns an expectation that checks the located argument term replacement
// carries the expected public presentation text.
func (locator ArgumentTermLocator) ReplacementPresentation(expected string) DeclarationExpectation {
	expectation := argumentTermReplacementPresentationExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// ReplacementTargetDeprecatedDuringMigration returns an expectation that checks the located
// argument term replacement carries the expected migration-deprecation marker.
func (locator ArgumentTermLocator) ReplacementTargetDeprecatedDuringMigration(expected bool) DeclarationExpectation {
	expectation := argumentTermReplacementTargetDeprecatedExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// Argument returns a locator for the uniquely resolved named argument term under the located
// command.
func (l CommandLocator) Argument(name string) ArgumentTermLocator {
	return commandArgumentLocator(l, name)
}

// ArgumentLiteral returns a locator for the uniquely resolved literal argument term under the
// located command.
func (l CommandLocator) ArgumentLiteral(value string) ArgumentTermLocator {
	return commandArgumentLiteralLocator(l, value)
}

// ArgumentFormAt returns a locator for the argument form at index in the located command's
// authoritative public form order.
func (l CommandLocator) ArgumentFormAt(index int) ArgumentFormLocator {
	return commandArgumentFormAtLocator(l, index)
}

// ArgumentFormsExactly returns an expectation that checks the located command's argument forms are
// exactly the supplied locators in authoritative public order. Exactness is not inferred from an
// incomplete observation.
func (l CommandLocator) ArgumentFormsExactly(values ...ArgumentFormLocator) DeclarationExpectation {
	expectation := commandArgumentFormsExpectation{
		command: l.state,
		items:   argumentFormLocatorStates(values),
	}
	return expectation
}

type commandArgumentFormsExpectation struct {
	command *commandLocatorState
	items   []*argumentFormLocatorState
}

type argumentFormLocatorMode uint8

const (
	argumentFormLocatorInvalid argumentFormLocatorMode = iota
	argumentFormLocatorID
	argumentFormLocatorCommandIndex
)

type argumentFormLocatorState struct {
	mode argumentFormLocatorMode
	id   cli.ArgumentFormID

	command *commandLocatorState
	index   int
}

type argumentTermLocatorMode uint8

const (
	argumentTermLocatorInvalid argumentTermLocatorMode = iota
	argumentTermLocatorID
	argumentTermLocatorCommandName
	argumentTermLocatorCommandLiteral
	argumentTermLocatorFormName
	argumentTermLocatorFormLiteral
	argumentTermLocatorFormIndex
)

type argumentTermLocatorState struct {
	mode argumentTermLocatorMode
	id   cli.ArgumentTermID

	command *commandLocatorState
	form    *argumentFormLocatorState
	text    string
	index   int
}

type argumentFormAvailabilityExpectation struct {
	locator  *argumentFormLocatorState
	expected bool
}

type argumentFormVisibilityExpectation struct {
	locator  *argumentFormLocatorState
	expected bool
}

type argumentTermAvailabilityExpectation struct {
	locator  *argumentTermLocatorState
	expected bool
}

type argumentTermVisibilityExpectation struct {
	locator  *argumentTermLocatorState
	expected bool
}

type argumentFormOriginExpectation struct {
	locator  *argumentFormLocatorState
	expected cli.DeclarationOrigin
}

type argumentFormReleaseLifecycleExpectation struct {
	locator  *argumentFormLocatorState
	expected cli.ReleaseLifecycle
}

type argumentFormReplacementAbsenceExpectation struct{ locator *argumentFormLocatorState }

type argumentFormReplacementPresentationExpectation struct {
	locator  *argumentFormLocatorState
	expected string
}

type argumentFormReplacementTargetDeprecatedExpectation struct {
	locator  *argumentFormLocatorState
	expected bool
}

type argumentTermOriginExpectation struct {
	locator  *argumentTermLocatorState
	expected cli.DeclarationOrigin
}

type argumentTermReleaseLifecycleExpectation struct {
	locator  *argumentTermLocatorState
	expected cli.ReleaseLifecycle
}

type argumentTermReplacementAbsenceExpectation struct{ locator *argumentTermLocatorState }

type argumentTermReplacementPresentationExpectation struct {
	locator  *argumentTermLocatorState
	expected string
}

type argumentTermReplacementTargetDeprecatedExpectation struct {
	locator  *argumentTermLocatorState
	expected bool
}

type argumentFormExistenceExpectation struct {
	locator  *argumentFormLocatorState
	expected bool
}

type argumentTermExistenceExpectation struct {
	locator  *argumentTermLocatorState
	expected bool
}

type argumentFormLifecycleExpectationValue struct {
	locator  *argumentFormLocatorState
	expected cli.LifecycleState
}

type argumentFormReplacementExpectationValue struct {
	locator  *argumentFormLocatorState
	expected DeclarationIdentityLocator
}

type argumentFormReachabilityExpectationValue struct {
	locator  *argumentFormLocatorState
	expected cli.Reachability
}

type argumentFormTermsExpectationValue struct {
	locator  *argumentFormLocatorState
	expected []*argumentTermLocatorState
}

type argumentTermKindExpectationValue struct {
	locator  *argumentTermLocatorState
	expected cli.ArgumentTermKind
}

type (
	argumentTermHiddenExpectationValue    struct{ locator *argumentTermLocatorState }
	argumentTermSensitiveExpectationValue struct{ locator *argumentTermLocatorState }
	argumentTermLifecycleExpectationValue struct {
		locator  *argumentTermLocatorState
		expected cli.LifecycleState
	}
)

type argumentTermReplacementExpectationValue struct {
	locator  *argumentTermLocatorState
	expected DeclarationIdentityLocator
}

type argumentTermPresenceExpectationValue struct {
	locator  *argumentTermLocatorState
	expected bool
}

type argumentTermCorePropertyKind uint8

const (
	argumentTermCorePropertyInvalid argumentTermCorePropertyKind = iota
	argumentTermCorePropertyHyphenValues
	argumentTermCorePropertyDescription
	argumentTermCorePropertyMinimal
	argumentTermCorePropertyEndOfOptions
	argumentTermCorePropertySourceSensitive
	argumentTermCorePropertyHasTokenLanguage
	argumentTermCorePropertyTokenLanguageFragments
)

type argumentTermCorePropertyExpectation struct {
	locator      *argumentTermLocatorState
	kind         argumentTermCorePropertyKind
	hyphenValues cli.HyphenValueMode
	text         string
	fragments    []string
}

func argumentFormLocatorStates(values []ArgumentFormLocator) []*argumentFormLocatorState {
	states := make([]*argumentFormLocatorState, len(values))
	for index, value := range values {
		states[index] = value.state
	}
	return states
}

func commandArgumentFormAtLocator(parent CommandLocator, index int) ArgumentFormLocator {
	state := &argumentFormLocatorState{
		mode:    argumentFormLocatorCommandIndex,
		command: parent.state,
		index:   index,
	}
	locator := ArgumentFormLocator{
		state: state,
	}
	return locator
}

func argumentTermNameLocator(parent ArgumentFormLocator, name string) ArgumentTermLocator {
	state := &argumentTermLocatorState{
		mode: argumentTermLocatorFormName,
		form: parent.state,
		text: name,
	}
	locator := ArgumentTermLocator{
		state: state,
	}
	return locator
}

func argumentTermLiteralLocator(parent ArgumentFormLocator, literal string) ArgumentTermLocator {
	state := &argumentTermLocatorState{
		mode: argumentTermLocatorFormLiteral,
		form: parent.state,
		text: literal,
	}
	locator := ArgumentTermLocator{
		state: state,
	}
	return locator
}

func argumentTermAtLocator(parent ArgumentFormLocator, index int) ArgumentTermLocator {
	state := &argumentTermLocatorState{
		mode:  argumentTermLocatorFormIndex,
		form:  parent.state,
		index: index,
	}
	locator := ArgumentTermLocator{
		state: state,
	}
	return locator
}

func argumentTermValueLocator(parent ArgumentTermLocator) ValueLocator {
	state := &valueLocatorState{
		mode:     valueLocatorArgumentTerm,
		argument: parent.state,
	}
	locator := ValueLocator{
		state: state,
	}
	return locator
}

func argumentFormLocatorLabel(state *argumentFormLocatorState) evaluationField {
	if state == nil {
		return semanticSubjectArgumentForm
	}
	switch state.mode {
	case argumentFormLocatorID:
		return evaluationText(fmt.Sprintf("argument form id %d", state.id))
	case argumentFormLocatorCommandIndex:
		return qualifySubject(commandLocatorLabel(state.command), evaluationText(fmt.Sprintf("argument form %d", state.index)))
	default:
		return semanticSubjectArgumentForm
	}
}

func resolveSchemaArgumentFormLocator(state *argumentFormLocatorState, schema *cli.Schema) locatorResolution[cli.ArgumentFormSchema] {
	var resolution locatorResolution[cli.ArgumentFormSchema]
	label := argumentFormLocatorLabel(state)
	if schema == nil || state == nil || state.mode == argumentFormLocatorInvalid {
		resolution = locatorResolution[cli.ArgumentFormSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectArgumentForm, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == argumentFormLocatorID {
		value, ok := schema.ArgumentForms[state.id]
		if !ok {
			resolution = locatorResolution[cli.ArgumentFormSchema]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectArgumentForm, observationAbsent),
			}
			return resolution
		}
		resolution = locatorResolution[cli.ArgumentFormSchema]{
			status:  locatorResolutionFound,
			value:   value,
			subject: label,
		}
		return resolution
	}
	command := resolveSchemaCommandLocator(state.command, schema)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.ArgumentFormSchema]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	if state.index < 0 {
		resolution = locatorResolution[cli.ArgumentFormSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectArgumentForm, observationInvalidLocator),
		}
		return resolution
	}
	if state.index >= len(command.value.ArgumentForms) {
		resolution = locatorResolution[cli.ArgumentFormSchema]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectArgumentForm, observationAbsent),
		}
		return resolution
	}
	id := command.value.ArgumentForms[state.index]
	value, ok := schema.ArgumentForms[id]
	if !ok {
		resolution = locatorResolution[cli.ArgumentFormSchema]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("argument-form relationship points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ArgumentFormSchema]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

func resolveReferenceArgumentFormLocator(state *argumentFormLocatorState, reference cli.Reference) locatorResolution[cli.ArgumentFormReference] {
	var resolution locatorResolution[cli.ArgumentFormReference]
	label := argumentFormLocatorLabel(state)
	if state == nil || state.mode == argumentFormLocatorInvalid {
		resolution = locatorResolution[cli.ArgumentFormReference]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectArgumentForm, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == argumentFormLocatorID {
		value, ok := reference.ArgumentForms[state.id]
		if !ok {
			resolution = locatorResolution[cli.ArgumentFormReference]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectArgumentForm, observationAbsent),
			}
			return resolution
		}
		resolution = locatorResolution[cli.ArgumentFormReference]{
			status:  locatorResolutionFound,
			value:   value,
			subject: label,
		}
		return resolution
	}
	command := resolveReferenceCommandLocator(state.command, reference)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.ArgumentFormReference]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	if state.index < 0 {
		resolution = locatorResolution[cli.ArgumentFormReference]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectArgumentForm, observationInvalidLocator),
		}
		return resolution
	}
	if state.index >= len(command.value.ArgumentForms) {
		resolution = locatorResolution[cli.ArgumentFormReference]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectArgumentForm, observationAbsent),
		}
		return resolution
	}
	id := command.value.ArgumentForms[state.index]
	value, ok := reference.ArgumentForms[id]
	if !ok {
		resolution = locatorResolution[cli.ArgumentFormReference]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("argument-form relationship points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ArgumentFormReference]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

func resolveInspectionArgumentFormLocator(state *argumentFormLocatorState, inspection cli.Inspection) locatorResolution[cli.ArgumentFormInspection] {
	var resolution locatorResolution[cli.ArgumentFormInspection]
	label := argumentFormLocatorLabel(state)
	if state == nil || state.mode == argumentFormLocatorInvalid {
		resolution = locatorResolution[cli.ArgumentFormInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectArgumentForm, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == argumentFormLocatorID {
		value, ok := inspection.ArgumentForms[state.id]
		if ok {
			resolution = locatorResolution[cli.ArgumentFormInspection]{
				status:  locatorResolutionFound,
				value:   value,
				subject: label,
			}
			return resolution
		}
		if inspection.Truncated {
			resolution = locatorResolution[cli.ArgumentFormInspection]{
				status:  locatorResolutionNotProvable,
				subject: label,
				detail:  observed(semanticSubjectInspection, observationTruncated),
			}
			return resolution
		}
		resolution = locatorResolution[cli.ArgumentFormInspection]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectArgumentForm, observationAbsent),
		}
		return resolution
	}
	command := resolveInspectionCommandLocator(state.command, inspection)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.ArgumentFormInspection]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	if state.index < 0 {
		resolution = locatorResolution[cli.ArgumentFormInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectArgumentForm, observationInvalidLocator),
		}
		return resolution
	}
	if state.index >= len(command.value.ArgumentForms) {
		if command.value.NamespaceCompleteness != cli.CommandNamespaceComplete || inspection.Truncated {
			resolution = locatorResolution[cli.ArgumentFormInspection]{
				status:  locatorResolutionNotProvable,
				subject: label,
				detail:  evaluationText("argument-form order is incomplete"),
			}
			return resolution
		}
		resolution = locatorResolution[cli.ArgumentFormInspection]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectArgumentForm, observationAbsent),
		}
		return resolution
	}
	id := command.value.ArgumentForms[state.index]
	value, ok := inspection.ArgumentForms[id]
	if !ok {
		resolution = locatorResolution[cli.ArgumentFormInspection]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("argument-form relationship points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ArgumentFormInspection]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

func argumentTermLocatorLabel(state *argumentTermLocatorState) evaluationField {
	if state == nil {
		return semanticSubjectArgumentTerm
	}
	switch state.mode {
	case argumentTermLocatorID:
		return evaluationText(fmt.Sprintf("argument term id %d", state.id))
	case argumentTermLocatorCommandName:
		return qualifySubject(commandLocatorLabel(state.command), subjectNamed(evaluationText("argument"), state.text))
	case argumentTermLocatorCommandLiteral:
		return qualifySubject(commandLocatorLabel(state.command), subjectNamed(evaluationText("literal argument"), state.text))
	case argumentTermLocatorFormName:
		return qualifySubject(argumentFormLocatorLabel(state.form), subjectNamed(evaluationText("argument"), state.text))
	case argumentTermLocatorFormLiteral:
		return qualifySubject(argumentFormLocatorLabel(state.form), subjectNamed(evaluationText("literal argument"), state.text))
	case argumentTermLocatorFormIndex:
		return qualifySubject(argumentFormLocatorLabel(state.form), evaluationText(fmt.Sprintf("argument term %d", state.index)))
	default:
		return semanticSubjectArgumentTerm
	}
}

func resolveSchemaArgumentTermLocator(state *argumentTermLocatorState, schema *cli.Schema) locatorResolution[cli.ArgumentTermSchema] {
	var resolution locatorResolution[cli.ArgumentTermSchema]
	label := argumentTermLocatorLabel(state)
	if schema == nil || state == nil || state.mode == argumentTermLocatorInvalid {
		resolution = locatorResolution[cli.ArgumentTermSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectArgumentTerm, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == argumentTermLocatorID {
		value, ok := schema.ArgumentTerms[state.id]
		if !ok {
			resolution = locatorResolution[cli.ArgumentTermSchema]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectArgumentTerm, observationAbsent),
			}
			return resolution
		}
		resolution = locatorResolution[cli.ArgumentTermSchema]{
			status:  locatorResolutionFound,
			value:   value,
			subject: label,
		}
		return resolution
	}
	ids, status, detail := schemaArgumentTermDomain(state, schema)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.ArgumentTermSchema]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	id, status, detail := selectSchemaArgumentTerm(state, ids, schema, true)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.ArgumentTermSchema]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	value, ok := schema.ArgumentTerms[id]
	if !ok {
		resolution = locatorResolution[cli.ArgumentTermSchema]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("argument-term relationship points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ArgumentTermSchema]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

func resolveReferenceArgumentTermLocator(state *argumentTermLocatorState, reference cli.Reference) locatorResolution[cli.ArgumentTermReference] {
	var resolution locatorResolution[cli.ArgumentTermReference]
	label := argumentTermLocatorLabel(state)
	if state == nil || state.mode == argumentTermLocatorInvalid {
		resolution = locatorResolution[cli.ArgumentTermReference]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectArgumentTerm, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == argumentTermLocatorID {
		value, ok := reference.ArgumentTerms[state.id]
		if !ok {
			resolution = locatorResolution[cli.ArgumentTermReference]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectArgumentTerm, observationAbsent),
			}
			return resolution
		}
		resolution = locatorResolution[cli.ArgumentTermReference]{
			status:  locatorResolutionFound,
			value:   value,
			subject: label,
		}
		return resolution
	}
	ids, status, detail := referenceArgumentTermDomain(state, reference)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.ArgumentTermReference]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	id, status, detail := selectReferenceArgumentTerm(state, ids, reference, true)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.ArgumentTermReference]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	value, ok := reference.ArgumentTerms[id]
	if !ok {
		resolution = locatorResolution[cli.ArgumentTermReference]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("argument-term relationship points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ArgumentTermReference]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

func resolveInspectionArgumentTermLocator(state *argumentTermLocatorState, inspection cli.Inspection) locatorResolution[cli.ArgumentTermInspection] {
	var resolution locatorResolution[cli.ArgumentTermInspection]
	label := argumentTermLocatorLabel(state)
	if state == nil || state.mode == argumentTermLocatorInvalid {
		resolution = locatorResolution[cli.ArgumentTermInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectArgumentTerm, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == argumentTermLocatorID {
		value, ok := inspection.ArgumentTerms[state.id]
		if ok {
			resolution = locatorResolution[cli.ArgumentTermInspection]{
				status:  locatorResolutionFound,
				value:   value,
				subject: label,
			}
			return resolution
		}
		if inspection.Truncated {
			resolution = locatorResolution[cli.ArgumentTermInspection]{
				status:  locatorResolutionNotProvable,
				subject: label,
				detail:  observed(semanticSubjectInspection, observationTruncated),
			}
			return resolution
		}
		resolution = locatorResolution[cli.ArgumentTermInspection]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectArgumentTerm, observationAbsent),
		}
		return resolution
	}
	ids, status, detail, complete := inspectionArgumentTermDomain(state, inspection)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.ArgumentTermInspection]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	id, status, detail := selectInspectionArgumentTerm(state, ids, inspection, complete && !inspection.Truncated)
	if (status == locatorResolutionAbsent || status == locatorResolutionNotProvable) && inspection.Truncated {
		status = locatorResolutionNotProvable
		detail = observed(semanticSubjectInspection, observationTruncated)
	}
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.ArgumentTermInspection]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	value, ok := inspection.ArgumentTerms[id]
	if !ok {
		resolution = locatorResolution[cli.ArgumentTermInspection]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("argument-term relationship points to missing detail"),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ArgumentTermInspection]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

func schemaArgumentTermDomain(state *argumentTermLocatorState, schema *cli.Schema) ([]cli.ArgumentTermID, locatorResolutionStatus, evaluationField) {
	if state.form != nil {
		form := resolveSchemaArgumentFormLocator(state.form, schema)
		if form.status != locatorResolutionFound {
			return nil, form.status, form.detail
		}
		return form.value.Terms, locatorResolutionFound, evaluationText("")
	}
	command := resolveSchemaCommandLocator(state.command, schema)
	if command.status != locatorResolutionFound {
		return nil, command.status, command.detail
	}
	if len(command.value.ArgumentForms) == 0 {
		return nil, locatorResolutionAbsent, observed(semanticSubjectArgumentForm, observationAbsent)
	}
	if len(command.value.ArgumentForms) > 1 {
		return nil, locatorResolutionAmbiguous, evaluationText(fmt.Sprintf("command has %d argument forms; select one explicitly", len(command.value.ArgumentForms)))
	}
	form, ok := schema.ArgumentForms[command.value.ArgumentForms[0]]
	if !ok {
		return nil, locatorResolutionNotProvable, evaluationText("argument-form relationship points to missing detail")
	}
	return form.Terms, locatorResolutionFound, evaluationText("")
}

func referenceArgumentTermDomain(state *argumentTermLocatorState, reference cli.Reference) ([]cli.ArgumentTermID, locatorResolutionStatus, evaluationField) {
	if state.form != nil {
		form := resolveReferenceArgumentFormLocator(state.form, reference)
		if form.status != locatorResolutionFound {
			return nil, form.status, form.detail
		}
		return form.value.Terms, locatorResolutionFound, evaluationText("")
	}
	command := resolveReferenceCommandLocator(state.command, reference)
	if command.status != locatorResolutionFound {
		return nil, command.status, command.detail
	}
	if len(command.value.ArgumentForms) == 0 {
		return nil, locatorResolutionAbsent, observed(semanticSubjectArgumentForm, observationAbsent)
	}
	if len(command.value.ArgumentForms) > 1 {
		return nil, locatorResolutionAmbiguous, evaluationText(fmt.Sprintf("command has %d argument forms; select one explicitly", len(command.value.ArgumentForms)))
	}
	form, ok := reference.ArgumentForms[command.value.ArgumentForms[0]]
	if !ok {
		return nil, locatorResolutionNotProvable, evaluationText("argument-form relationship points to missing detail")
	}
	return form.Terms, locatorResolutionFound, evaluationText("")
}

func inspectionArgumentTermDomain(state *argumentTermLocatorState, inspection cli.Inspection) ([]cli.ArgumentTermID, locatorResolutionStatus, evaluationField, bool) {
	if state.form != nil {
		form := resolveInspectionArgumentFormLocator(state.form, inspection)
		if form.status != locatorResolutionFound {
			return nil, form.status, form.detail, false
		}
		return form.value.Terms, locatorResolutionFound, evaluationText(""), true
	}
	command := resolveInspectionCommandLocator(state.command, inspection)
	if command.status != locatorResolutionFound {
		return nil, command.status, command.detail, false
	}
	if len(command.value.ArgumentForms) == 0 {
		if command.value.NamespaceCompleteness == cli.CommandNamespaceComplete && !inspection.Truncated {
			return nil, locatorResolutionAbsent, observed(semanticSubjectArgumentForm, observationAbsent), true
		}
		return nil, locatorResolutionNotProvable, evaluationText("command argument-form domain is incomplete"), false
	}
	if len(command.value.ArgumentForms) > 1 {
		return nil, locatorResolutionAmbiguous, evaluationText(fmt.Sprintf("command has %d argument forms; select one explicitly", len(command.value.ArgumentForms))), true
	}
	form, ok := inspection.ArgumentForms[command.value.ArgumentForms[0]]
	if !ok {
		return nil, locatorResolutionNotProvable, evaluationText("argument-form relationship points to missing detail"), false
	}
	return form.Terms, locatorResolutionFound, evaluationText(""), command.value.NamespaceCompleteness == cli.CommandNamespaceComplete
}

//nolint:dupl // Separate public artifacts resolve their own argument-term relationships directly.
func selectSchemaArgumentTerm(state *argumentTermLocatorState, ids []cli.ArgumentTermID, schema *cli.Schema, complete bool) (cli.ArgumentTermID, locatorResolutionStatus, evaluationField) {
	if state.mode == argumentTermLocatorFormIndex {
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectArgumentTerm, observationInvalidLocator)
		}
		if state.index < len(ids) {
			return ids[state.index], locatorResolutionFound, evaluationText("")
		}
		if complete {
			return 0, locatorResolutionAbsent, observed(semanticSubjectArgumentTerm, observationAbsent)
		}
		return 0, locatorResolutionNotProvable, evaluationText("argument-term lookup domain is incomplete")
	}
	wantName := state.mode == argumentTermLocatorCommandName || state.mode == argumentTermLocatorFormName
	wantLiteral := state.mode == argumentTermLocatorCommandLiteral || state.mode == argumentTermLocatorFormLiteral
	if !wantName && !wantLiteral {
		return 0, locatorResolutionInvalid, observed(semanticSubjectArgumentTerm, observationInvalidLocator)
	}
	matches := make([]cli.ArgumentTermID, 0, 1)
	missingDetail := false
	for _, id := range ids {
		term, ok := schema.ArgumentTerms[id]
		if !ok {
			missingDetail = true
			continue
		}
		if wantName && term.Name == state.text || wantLiteral && term.Literal == state.text {
			matches = append(matches, id)
		}
	}
	if len(matches) == 1 {
		return matches[0], locatorResolutionFound, evaluationText("")
	}
	if len(matches) > 1 {
		field := collectionCardinalityField{
			name:  argumentTermAmbiguityDetail,
			count: len(matches),
		}
		return 0, locatorResolutionAmbiguous, field
	}
	if missingDetail || !complete {
		return 0, locatorResolutionNotProvable, evaluationText("argument-term lookup domain is incomplete")
	}
	return 0, locatorResolutionAbsent, observed(semanticSubjectArgumentTerm, observationAbsent)
}

//nolint:dupl // Separate public artifacts resolve their own argument-term relationships directly.
func selectReferenceArgumentTerm(state *argumentTermLocatorState, ids []cli.ArgumentTermID, reference cli.Reference, complete bool) (cli.ArgumentTermID, locatorResolutionStatus, evaluationField) {
	if state.mode == argumentTermLocatorFormIndex {
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectArgumentTerm, observationInvalidLocator)
		}
		if state.index < len(ids) {
			return ids[state.index], locatorResolutionFound, evaluationText("")
		}
		if complete {
			return 0, locatorResolutionAbsent, observed(semanticSubjectArgumentTerm, observationAbsent)
		}
		return 0, locatorResolutionNotProvable, evaluationText("argument-term lookup domain is incomplete")
	}
	wantName := state.mode == argumentTermLocatorCommandName || state.mode == argumentTermLocatorFormName
	wantLiteral := state.mode == argumentTermLocatorCommandLiteral || state.mode == argumentTermLocatorFormLiteral
	if !wantName && !wantLiteral {
		return 0, locatorResolutionInvalid, observed(semanticSubjectArgumentTerm, observationInvalidLocator)
	}
	matches := make([]cli.ArgumentTermID, 0, 1)
	missingDetail := false
	for _, id := range ids {
		term, ok := reference.ArgumentTerms[id]
		if !ok {
			missingDetail = true
			continue
		}
		if wantName && term.Name == state.text || wantLiteral && term.Literal == state.text {
			matches = append(matches, id)
		}
	}
	if len(matches) == 1 {
		return matches[0], locatorResolutionFound, evaluationText("")
	}
	if len(matches) > 1 {
		field := collectionCardinalityField{
			name:  argumentTermAmbiguityDetail,
			count: len(matches),
		}
		return 0, locatorResolutionAmbiguous, field
	}
	if missingDetail || !complete {
		return 0, locatorResolutionNotProvable, evaluationText("argument-term lookup domain is incomplete")
	}
	return 0, locatorResolutionAbsent, observed(semanticSubjectArgumentTerm, observationAbsent)
}

//nolint:dupl // Separate public artifacts resolve their own argument-term relationships directly.
func selectInspectionArgumentTerm(state *argumentTermLocatorState, ids []cli.ArgumentTermID, inspection cli.Inspection, complete bool) (cli.ArgumentTermID, locatorResolutionStatus, evaluationField) {
	if state.mode == argumentTermLocatorFormIndex {
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectArgumentTerm, observationInvalidLocator)
		}
		if state.index < len(ids) {
			return ids[state.index], locatorResolutionFound, evaluationText("")
		}
		if complete {
			return 0, locatorResolutionAbsent, observed(semanticSubjectArgumentTerm, observationAbsent)
		}
		return 0, locatorResolutionNotProvable, evaluationText("argument-term lookup domain is incomplete")
	}
	wantName := state.mode == argumentTermLocatorCommandName || state.mode == argumentTermLocatorFormName
	wantLiteral := state.mode == argumentTermLocatorCommandLiteral || state.mode == argumentTermLocatorFormLiteral
	if !wantName && !wantLiteral {
		return 0, locatorResolutionInvalid, observed(semanticSubjectArgumentTerm, observationInvalidLocator)
	}
	matches := make([]cli.ArgumentTermID, 0, 1)
	missingDetail := false
	for _, id := range ids {
		term, ok := inspection.ArgumentTerms[id]
		if !ok {
			missingDetail = true
			continue
		}
		if wantName && term.Name == state.text || wantLiteral && term.Literal == state.text {
			matches = append(matches, id)
		}
	}
	if len(matches) == 1 {
		return matches[0], locatorResolutionFound, evaluationText("")
	}
	if len(matches) > 1 {
		field := collectionCardinalityField{
			name:  argumentTermAmbiguityDetail,
			count: len(matches),
		}
		return 0, locatorResolutionAmbiguous, field
	}
	if missingDetail || !complete {
		return 0, locatorResolutionNotProvable, evaluationText("argument-term lookup domain is incomplete")
	}
	return 0, locatorResolutionAbsent, observed(semanticSubjectArgumentTerm, observationAbsent)
}

func evaluateArgumentFormReplacementPresentationSchema(expectation argumentFormReplacementPresentationExpectation, schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaArgumentFormLocator(expectation.locator, schema)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func evaluateArgumentFormReplacementPresentationReference(expectation argumentFormReplacementPresentationExpectation, reference cli.Reference) evaluationResult {
	resolution := resolveReferenceArgumentFormLocator(expectation.locator, reference)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func evaluateArgumentFormReplacementPresentationInspection(expectation argumentFormReplacementPresentationExpectation, inspection cli.Inspection) evaluationResult {
	resolution := resolveInspectionArgumentFormLocator(expectation.locator, inspection)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func evaluateArgumentTermReplacementPresentationSchema(expectation argumentTermReplacementPresentationExpectation, schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaArgumentTermLocator(expectation.locator, schema)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func evaluateArgumentTermReplacementPresentationReference(expectation argumentTermReplacementPresentationExpectation, reference cli.Reference) evaluationResult {
	resolution := resolveReferenceArgumentTermLocator(expectation.locator, reference)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func evaluateArgumentTermReplacementPresentationInspection(expectation argumentTermReplacementPresentationExpectation, inspection cli.Inspection) evaluationResult {
	resolution := resolveInspectionArgumentTermLocator(expectation.locator, inspection)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func argumentFormIDLocator(id cli.ArgumentFormID) ArgumentFormLocator {
	state := &argumentFormLocatorState{
		mode: argumentFormLocatorID,
		id:   id,
	}
	locator := ArgumentFormLocator{
		state: state,
	}
	return locator
}

func argumentTermIDLocator(id cli.ArgumentTermID) ArgumentTermLocator {
	state := &argumentTermLocatorState{
		mode: argumentTermLocatorID,
		id:   id,
	}
	locator := ArgumentTermLocator{
		state: state,
	}
	return locator
}

func (commandArgumentFormsExpectation) schemaExpectation() {}

func (commandArgumentFormsExpectation) referenceExpectation() {}

func (commandArgumentFormsExpectation) inspectionExpectation() {}

//nolint:dupl // Typed collection proofs intentionally keep each public declaration family direct; generic resolver callbacks would recreate the rejected universal framework.
func (expectation commandArgumentFormsExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	command := resolveSchemaCommandLocator(expectation.command, schema)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("argument forms exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("argument forms"))
	actual := command.value.ArgumentForms
	for _, id := range actual {
		if _, ok := schema.ArgumentForms[id]; !ok {
			return collectionClosureFailure(label, "argument forms", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.ArgumentFormID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveSchemaArgumentFormLocator(state, schema)
		if outcome, failed := collectionResolutionFailure(label, "argument forms", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "argument forms", expected, actual)
}

//nolint:dupl // Typed collection proofs intentionally keep each public declaration family direct; generic resolver callbacks would recreate the rejected universal framework.
func (expectation commandArgumentFormsExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	command := resolveReferenceCommandLocator(expectation.command, reference)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("argument forms exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("argument forms"))
	actual := command.value.ArgumentForms
	for _, id := range actual {
		if _, ok := reference.ArgumentForms[id]; !ok {
			return collectionClosureFailure(label, "argument forms", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.ArgumentFormID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveReferenceArgumentFormLocator(state, reference)
		if outcome, failed := collectionResolutionFailure(label, "argument forms", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "argument forms", expected, actual)
}

func (expectation commandArgumentFormsExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	command := resolveInspectionCommandLocator(expectation.command, inspection)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("argument forms exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("argument forms"))
	if inspection.Truncated {
		return collectionClosureFailure(label, "argument forms", observed(semanticSubjectInspection, observationTruncated))
	}
	if command.value.NamespaceCompleteness != cli.CommandNamespaceComplete {
		return collectionClosureFailure(label, "argument forms", evaluationText(fmt.Sprintf("command namespace is %v", command.value.NamespaceCompleteness)))
	}
	actual := command.value.ArgumentForms
	for _, id := range actual {
		if _, ok := inspection.ArgumentForms[id]; !ok {
			return collectionClosureFailure(label, "argument forms", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.ArgumentFormID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveInspectionArgumentFormLocator(state, inspection)
		if outcome, failed := collectionResolutionFailure(label, "argument forms", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "argument forms", expected, actual)
}

func (argumentFormAvailabilityExpectation) referenceExpectation() {}

func (argumentFormAvailabilityExpectation) inspectionExpectation() {}

func (expectation argumentFormAvailabilityExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentFormLocator(expectation.locator, reference)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "available", expectation.expected, resolution.value.Available)
}

func (expectation argumentFormAvailabilityExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentFormLocator(expectation.locator, inspection)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "available", expectation.expected, resolution.value.Available)
}

func (argumentFormVisibilityExpectation) referenceExpectation() {}

func (argumentFormVisibilityExpectation) inspectionExpectation() {}

func (expectation argumentFormVisibilityExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentFormLocator(expectation.locator, reference)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "visible", expectation.expected, resolution.value.Visible)
}

func (expectation argumentFormVisibilityExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentFormLocator(expectation.locator, inspection)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "visible", expectation.expected, resolution.value.Visible)
}

func (argumentTermAvailabilityExpectation) referenceExpectation() {}

func (argumentTermAvailabilityExpectation) inspectionExpectation() {}

func (expectation argumentTermAvailabilityExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentTermLocator(expectation.locator, reference)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "available", expectation.expected, resolution.value.Available)
}

func (expectation argumentTermAvailabilityExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentTermLocator(expectation.locator, inspection)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "available", expectation.expected, resolution.value.Available)
}

func (argumentTermVisibilityExpectation) referenceExpectation() {}

func (argumentTermVisibilityExpectation) inspectionExpectation() {}

func (expectation argumentTermVisibilityExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentTermLocator(expectation.locator, reference)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "visible", expectation.expected, resolution.value.Visible)
}

func (expectation argumentTermVisibilityExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentTermLocator(expectation.locator, inspection)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "visible", expectation.expected, resolution.value.Visible)
}

func (argumentFormOriginExpectation) schemaExpectation() {}

func (argumentFormOriginExpectation) referenceExpectation() {}

func (argumentFormOriginExpectation) inspectionExpectation() {}

func (expectation argumentFormOriginExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaArgumentFormLocator(expectation.locator, schema)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (expectation argumentFormOriginExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentFormLocator(expectation.locator, reference)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (expectation argumentFormOriginExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentFormLocator(expectation.locator, inspection)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (argumentFormReleaseLifecycleExpectation) schemaExpectation() {}

func (argumentFormReleaseLifecycleExpectation) referenceExpectation() {}

func (argumentFormReleaseLifecycleExpectation) inspectionExpectation() {}

func (expectation argumentFormReleaseLifecycleExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaArgumentFormLocator(expectation.locator, schema)
	actual := resolution.value.Release
	field := releaseLifecycleField{
		value: expectation.expected,
	}
	observedField := releaseLifecycleField{
		value: actual,
	}
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, field, observedField, actual == expectation.expected)
}

func (expectation argumentFormReleaseLifecycleExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentFormLocator(expectation.locator, reference)
	actual := resolution.value.Release
	field := releaseLifecycleField{
		value: expectation.expected,
	}
	observedField := releaseLifecycleField{
		value: actual,
	}
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, field, observedField, actual == expectation.expected)
}

func (expectation argumentFormReleaseLifecycleExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentFormLocator(expectation.locator, inspection)
	actual := resolution.value.Release
	field := releaseLifecycleField{
		value: expectation.expected,
	}
	observedField := releaseLifecycleField{
		value: actual,
	}
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, field, observedField, actual == expectation.expected)
}

func (argumentFormReplacementAbsenceExpectation) schemaExpectation() {}

func (argumentFormReplacementAbsenceExpectation) referenceExpectation() {}

func (argumentFormReplacementAbsenceExpectation) inspectionExpectation() {}

func (expectation argumentFormReplacementAbsenceExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaArgumentFormLocator(expectation.locator, schema)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (expectation argumentFormReplacementAbsenceExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentFormLocator(expectation.locator, reference)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (expectation argumentFormReplacementAbsenceExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentFormLocator(expectation.locator, inspection)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (argumentFormReplacementPresentationExpectation) schemaExpectation() {}

func (argumentFormReplacementPresentationExpectation) referenceExpectation() {}

func (argumentFormReplacementPresentationExpectation) inspectionExpectation() {}

func (expectation argumentFormReplacementPresentationExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	return evaluateArgumentFormReplacementPresentationSchema(expectation, schema)
}

func (expectation argumentFormReplacementPresentationExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return evaluateArgumentFormReplacementPresentationReference(expectation, reference)
}

func (expectation argumentFormReplacementPresentationExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return evaluateArgumentFormReplacementPresentationInspection(expectation, inspection)
}

func (argumentFormReplacementTargetDeprecatedExpectation) schemaExpectation() {}

func (argumentFormReplacementTargetDeprecatedExpectation) referenceExpectation() {}

func (argumentFormReplacementTargetDeprecatedExpectation) inspectionExpectation() {}

func (expectation argumentFormReplacementTargetDeprecatedExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaArgumentFormLocator(expectation.locator, schema)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (expectation argumentFormReplacementTargetDeprecatedExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentFormLocator(expectation.locator, reference)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (expectation argumentFormReplacementTargetDeprecatedExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentFormLocator(expectation.locator, inspection)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (argumentTermOriginExpectation) schemaExpectation() {}

func (argumentTermOriginExpectation) referenceExpectation() {}

func (argumentTermOriginExpectation) inspectionExpectation() {}

func (expectation argumentTermOriginExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaArgumentTermLocator(expectation.locator, schema)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (expectation argumentTermOriginExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentTermLocator(expectation.locator, reference)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (expectation argumentTermOriginExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentTermLocator(expectation.locator, inspection)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (argumentTermReleaseLifecycleExpectation) schemaExpectation() {}

func (argumentTermReleaseLifecycleExpectation) referenceExpectation() {}

func (argumentTermReleaseLifecycleExpectation) inspectionExpectation() {}

func (expectation argumentTermReleaseLifecycleExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaArgumentTermLocator(expectation.locator, schema)
	actual := resolution.value.Release
	field := releaseLifecycleField{
		value: expectation.expected,
	}
	observedField := releaseLifecycleField{
		value: actual,
	}
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, field, observedField, actual == expectation.expected)
}

func (expectation argumentTermReleaseLifecycleExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentTermLocator(expectation.locator, reference)
	actual := resolution.value.Release
	field := releaseLifecycleField{
		value: expectation.expected,
	}
	observedField := releaseLifecycleField{
		value: actual,
	}
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, field, observedField, actual == expectation.expected)
}

func (expectation argumentTermReleaseLifecycleExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentTermLocator(expectation.locator, inspection)
	actual := resolution.value.Release
	field := releaseLifecycleField{
		value: expectation.expected,
	}
	observedField := releaseLifecycleField{
		value: actual,
	}
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, field, observedField, actual == expectation.expected)
}

func (argumentTermReplacementAbsenceExpectation) schemaExpectation() {}

func (argumentTermReplacementAbsenceExpectation) referenceExpectation() {}

func (argumentTermReplacementAbsenceExpectation) inspectionExpectation() {}

func (expectation argumentTermReplacementAbsenceExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaArgumentTermLocator(expectation.locator, schema)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (expectation argumentTermReplacementAbsenceExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentTermLocator(expectation.locator, reference)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (expectation argumentTermReplacementAbsenceExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentTermLocator(expectation.locator, inspection)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (argumentTermReplacementPresentationExpectation) schemaExpectation() {}

func (argumentTermReplacementPresentationExpectation) referenceExpectation() {}

func (argumentTermReplacementPresentationExpectation) inspectionExpectation() {}

func (expectation argumentTermReplacementPresentationExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	return evaluateArgumentTermReplacementPresentationSchema(expectation, schema)
}

func (expectation argumentTermReplacementPresentationExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return evaluateArgumentTermReplacementPresentationReference(expectation, reference)
}

func (expectation argumentTermReplacementPresentationExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return evaluateArgumentTermReplacementPresentationInspection(expectation, inspection)
}

func (argumentTermReplacementTargetDeprecatedExpectation) schemaExpectation() {}

func (argumentTermReplacementTargetDeprecatedExpectation) referenceExpectation() {}

func (argumentTermReplacementTargetDeprecatedExpectation) inspectionExpectation() {}

func (expectation argumentTermReplacementTargetDeprecatedExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaArgumentTermLocator(expectation.locator, schema)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (expectation argumentTermReplacementTargetDeprecatedExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentTermLocator(expectation.locator, reference)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (expectation argumentTermReplacementTargetDeprecatedExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentTermLocator(expectation.locator, inspection)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (argumentFormExistenceExpectation) schemaExpectation() {}

func (argumentFormExistenceExpectation) referenceExpectation() {}

func (argumentFormExistenceExpectation) inspectionExpectation() {}

func (expectation argumentFormExistenceExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	return declarationExistenceResult(resolveSchemaArgumentFormLocator(expectation.locator, schema), expectation.expected)
}

func (expectation argumentFormExistenceExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return declarationExistenceResult(resolveReferenceArgumentFormLocator(expectation.locator, reference), expectation.expected)
}

func (expectation argumentFormExistenceExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return declarationExistenceResult(resolveInspectionArgumentFormLocator(expectation.locator, inspection), expectation.expected)
}

func (argumentTermExistenceExpectation) schemaExpectation() {}

func (argumentTermExistenceExpectation) referenceExpectation() {}

func (argumentTermExistenceExpectation) inspectionExpectation() {}

func (expectation argumentTermExistenceExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	return declarationExistenceResult(resolveSchemaArgumentTermLocator(expectation.locator, schema), expectation.expected)
}

func (expectation argumentTermExistenceExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return declarationExistenceResult(resolveReferenceArgumentTermLocator(expectation.locator, reference), expectation.expected)
}

func (expectation argumentTermExistenceExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return declarationExistenceResult(resolveInspectionArgumentTermLocator(expectation.locator, inspection), expectation.expected)
}

func (argumentFormLifecycleExpectationValue) schemaExpectation() {}

func (argumentFormLifecycleExpectationValue) referenceExpectation() {}

func (argumentFormLifecycleExpectationValue) inspectionExpectation() {}

func (expectation argumentFormLifecycleExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaArgumentFormLocator(expectation.locator, schema)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (expectation argumentFormLifecycleExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentFormLocator(expectation.locator, reference)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (expectation argumentFormLifecycleExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentFormLocator(expectation.locator, inspection)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (argumentFormReplacementExpectationValue) schemaExpectation() {}

func (argumentFormReplacementExpectationValue) referenceExpectation() {}

func (argumentFormReplacementExpectationValue) inspectionExpectation() {}

func (expectation argumentFormReplacementExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaArgumentFormLocator(expectation.locator, schema)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, schema, nil, cli.Inspection{})
}

func (expectation argumentFormReplacementExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentFormLocator(expectation.locator, reference)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, nil, &reference, cli.Inspection{})
}

func (expectation argumentFormReplacementExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentFormLocator(expectation.locator, inspection)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, nil, nil, inspection)
}

func (argumentFormReachabilityExpectationValue) schemaExpectation() {}

func (argumentFormReachabilityExpectationValue) referenceExpectation() {}

func (argumentFormReachabilityExpectationValue) inspectionExpectation() {}

func (expectation argumentFormReachabilityExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaArgumentFormLocator(expectation.locator, schema)
	actual := resolution.value.Reachability
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("reachability %v", expectation.expected)), evaluationText(fmt.Sprintf("reachability %v", actual)), actual == expectation.expected)
}

func (expectation argumentFormReachabilityExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentFormLocator(expectation.locator, reference)
	actual := resolution.value.Reachability
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("reachability %v", expectation.expected)), evaluationText(fmt.Sprintf("reachability %v", actual)), actual == expectation.expected)
}

func (expectation argumentFormReachabilityExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentFormLocator(expectation.locator, inspection)
	actual := resolution.value.Reachability
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("reachability %v", expectation.expected)), evaluationText(fmt.Sprintf("reachability %v", actual)), actual == expectation.expected)
}

func (argumentFormTermsExpectationValue) schemaExpectation() {}

func (argumentFormTermsExpectationValue) referenceExpectation() {}

func (argumentFormTermsExpectationValue) inspectionExpectation() {}

func (expectation argumentFormTermsExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	form := resolveSchemaArgumentFormLocator(expectation.locator, schema)
	if form.status != locatorResolutionFound {
		return declarationResolvedComparison(form.status, form.subject, form.detail, evaluationText("exact terms"), evaluationText(""), false)
	}
	ids := make([]cli.ArgumentTermID, len(expectation.expected))
	for index, state := range expectation.expected {
		resolution := resolveSchemaArgumentTermLocator(state, schema)
		if resolution.status != locatorResolutionFound {
			return declarationResolvedComparison(resolution.status, form.subject, resolution.detail, evaluationText("exact terms"), evaluationText(""), false)
		}
		ids[index] = resolution.value.ID
	}
	if slices.Equal(form.value.Terms, ids) {
		return provedResult()
	}
	return orderedCollectionMismatch(form.subject, "terms", ids, form.value.Terms)
}

func (expectation argumentFormTermsExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	form := resolveReferenceArgumentFormLocator(expectation.locator, reference)
	if form.status != locatorResolutionFound {
		return declarationResolvedComparison(form.status, form.subject, form.detail, evaluationText("exact terms"), evaluationText(""), false)
	}
	ids := make([]cli.ArgumentTermID, len(expectation.expected))
	for index, state := range expectation.expected {
		resolution := resolveReferenceArgumentTermLocator(state, reference)
		if resolution.status != locatorResolutionFound {
			return declarationResolvedComparison(resolution.status, form.subject, resolution.detail, evaluationText("exact terms"), evaluationText(""), false)
		}
		ids[index] = resolution.value.ID
	}
	if slices.Equal(form.value.Terms, ids) {
		return provedResult()
	}
	return orderedCollectionMismatch(form.subject, "terms", ids, form.value.Terms)
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func (expectation argumentFormTermsExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	form := resolveInspectionArgumentFormLocator(expectation.locator, inspection)
	if form.status != locatorResolutionFound {
		return declarationResolvedComparison(form.status, form.subject, form.detail, evaluationText("exact terms"), evaluationText(""), false)
	}
	if inspection.Truncated {
		return notProvableResult(form.subject, evaluationText("exact terms"), observed(semanticSubjectInspection, observationTruncated))
	}
	ids := make([]cli.ArgumentTermID, len(expectation.expected))
	for index, state := range expectation.expected {
		resolution := resolveInspectionArgumentTermLocator(state, inspection)
		if resolution.status != locatorResolutionFound {
			return declarationResolvedComparison(resolution.status, form.subject, resolution.detail, evaluationText("exact terms"), evaluationText(""), false)
		}
		ids[index] = resolution.value.ID
	}
	if slices.Equal(form.value.Terms, ids) {
		return provedResult()
	}
	return orderedCollectionMismatch(form.subject, "terms", ids, form.value.Terms)
}

func (argumentTermKindExpectationValue) schemaExpectation() {}

func (argumentTermKindExpectationValue) referenceExpectation() {}

func (argumentTermKindExpectationValue) inspectionExpectation() {}

func (expectation argumentTermKindExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaArgumentTermLocator(expectation.locator, schema)
	actual := resolution.value.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (expectation argumentTermKindExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentTermLocator(expectation.locator, reference)
	actual := resolution.value.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (expectation argumentTermKindExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentTermLocator(expectation.locator, inspection)
	actual := resolution.value.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (argumentTermHiddenExpectationValue) schemaExpectation() {}

func (argumentTermHiddenExpectationValue) referenceExpectation() {}

func (argumentTermHiddenExpectationValue) inspectionExpectation() {}

func (expectation argumentTermHiddenExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaArgumentTermLocator(expectation.locator, schema)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("hidden"), true, resolution.value.Hidden)
}

func (expectation argumentTermHiddenExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentTermLocator(expectation.locator, reference)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("hidden"), true, resolution.value.Hidden)
}

func (expectation argumentTermHiddenExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentTermLocator(expectation.locator, inspection)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("hidden"), true, resolution.value.Hidden)
}

func (argumentTermSensitiveExpectationValue) schemaExpectation() {}

func (argumentTermSensitiveExpectationValue) referenceExpectation() {}

func (argumentTermSensitiveExpectationValue) inspectionExpectation() {}

func (expectation argumentTermSensitiveExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaArgumentTermLocator(expectation.locator, schema)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, semanticRequirementSensitive, true, resolution.value.Sensitive)
}

func (expectation argumentTermSensitiveExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentTermLocator(expectation.locator, reference)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, semanticRequirementSensitive, true, resolution.value.Sensitive)
}

func (expectation argumentTermSensitiveExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentTermLocator(expectation.locator, inspection)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, semanticRequirementSensitive, true, resolution.value.Sensitive)
}

func (argumentTermLifecycleExpectationValue) schemaExpectation() {}

func (argumentTermLifecycleExpectationValue) referenceExpectation() {}

func (argumentTermLifecycleExpectationValue) inspectionExpectation() {}

func (expectation argumentTermLifecycleExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaArgumentTermLocator(expectation.locator, schema)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (expectation argumentTermLifecycleExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentTermLocator(expectation.locator, reference)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (expectation argumentTermLifecycleExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentTermLocator(expectation.locator, inspection)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (argumentTermReplacementExpectationValue) schemaExpectation() {}

func (argumentTermReplacementExpectationValue) referenceExpectation() {}

func (argumentTermReplacementExpectationValue) inspectionExpectation() {}

func (expectation argumentTermReplacementExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaArgumentTermLocator(expectation.locator, schema)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, schema, nil, cli.Inspection{})
}

func (expectation argumentTermReplacementExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentTermLocator(expectation.locator, reference)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, nil, &reference, cli.Inspection{})
}

func (expectation argumentTermReplacementExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentTermLocator(expectation.locator, inspection)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, nil, nil, inspection)
}

func (argumentTermPresenceExpectationValue) inspectionExpectation() {}

func (expectation argumentTermPresenceExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentTermLocator(expectation.locator, inspection)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("present=%t", expectation.expected)), expectation.expected, resolution.value.Present)
}

func (argumentTermCorePropertyExpectation) schemaExpectation() {}

func (argumentTermCorePropertyExpectation) referenceExpectation() {}

func (argumentTermCorePropertyExpectation) inspectionExpectation() {}

func (expectation argumentTermCorePropertyExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaArgumentTermLocator(expectation.locator, schema)
	return expectation.evaluate(resolution.status, resolution.subject, resolution.detail, resolution.value.HyphenValues, resolution.value.Description, resolution.value.Minimal, resolution.value.EndOfOptions, resolution.value.SourceSensitive, resolution.value.TokenLanguage)
}

func (expectation argumentTermCorePropertyExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceArgumentTermLocator(expectation.locator, reference)
	return expectation.evaluate(resolution.status, resolution.subject, resolution.detail, resolution.value.HyphenValues, resolution.value.Description, resolution.value.Minimal, resolution.value.EndOfOptions, resolution.value.SourceSensitive, resolution.value.TokenLanguage)
}

func (expectation argumentTermCorePropertyExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentTermLocator(expectation.locator, inspection)
	return expectation.evaluate(resolution.status, resolution.subject, resolution.detail, resolution.value.HyphenValues, resolution.value.Description, resolution.value.Minimal, resolution.value.EndOfOptions, resolution.value.SourceSensitive, resolution.value.TokenLanguage)
}

func (expectation argumentTermCorePropertyExpectation) evaluate(status locatorResolutionStatus, subject, detail evaluationField, hyphenValues cli.HyphenValueMode, description string, minimal, endOfOptions, sourceSensitive bool, tokenLanguage *cli.TokenLanguageSyntax) evaluationResult {
	switch expectation.kind {
	case argumentTermCorePropertyHyphenValues:
		return declarationResolvedComparison(status, subject, detail, evaluationText("hyphen-values"), evaluationText(fmt.Sprint(hyphenValues)), hyphenValues == expectation.hyphenValues)
	case argumentTermCorePropertyDescription:
		return declarationResolvedComparison(status, subject, detail, evaluationText("description"), evaluationText(description), description == expectation.text)
	case argumentTermCorePropertyMinimal:
		return resolvedBoolProperty(status, subject, detail, evaluationText("minimal"), true, minimal)
	case argumentTermCorePropertyEndOfOptions:
		return resolvedBoolProperty(status, subject, detail, evaluationText("end-of-options"), true, endOfOptions)
	case argumentTermCorePropertySourceSensitive:
		return resolvedBoolProperty(status, subject, detail, evaluationText("source-sensitive"), true, sourceSensitive)
	case argumentTermCorePropertyHasTokenLanguage:
		return resolvedBoolProperty(status, subject, detail, evaluationText("token-language-present"), true, tokenLanguage != nil)
	case argumentTermCorePropertyTokenLanguageFragments:
		if status != locatorResolutionFound {
			return declarationResolvedComparison(status, subject, detail, evaluationText("token language present with exact fragments in order"), evaluationText(""), false)
		}
		if tokenLanguage == nil {
			return disprovedResult(subject, evaluationText("token language present with exact fragments in order"), observed(semanticSubjectTokenLanguage, observationAbsent))
		}
		if slices.Equal(tokenLanguage.Fragments, expectation.fragments) {
			return provedResult()
		}
		return orderedCollectionMismatch(subject, "token-language fragments", expectation.fragments, tokenLanguage.Fragments)
	default:
		panic("clitest: invalid argument-term core property expectation")
	}
}

func (helpArgumentFormExpectation) helpPresentationExpectation() {}

func (expectation helpArgumentFormExpectation) evaluateHelp(page cli.HelpPage) evaluationResult {
	form, status, detail := resolveHelpArgumentForm(expectation.locator, page)
	subject := evaluationText("help argument form")
	if expectation.property == helpArgumentFormExists {
		return evaluateHelpExistence(subject, expectation.exists, status)
	}
	if status != locatorResolutionFound {
		return helpResolutionFailure(subject, property(evaluationText("term-count"), evaluationText(strconv.Itoa(expectation.count))), detail)
	}
	if expectation.property != helpArgumentFormTermCount {
		panic("clitest: invalid Help argument-form expectation")
	}
	if len(form.Terms) == expectation.count {
		return provedResult()
	}
	return helpPropertyMismatch(subject, "term-count", evaluationText(strconv.Itoa(expectation.count)), evaluationText(strconv.Itoa(len(form.Terms))))
}

func (helpArgumentTermExpectation) helpPresentationExpectation() {}

func (expectation helpArgumentTermExpectation) evaluateHelp(page cli.HelpPage) evaluationResult {
	term, status, detail := resolveHelpArgumentTerm(expectation.locator, page)
	subject := evaluationText("help argument term")
	if expectation.property == helpArgumentTermExists {
		return evaluateHelpExistence(subject, expectation.exists, status)
	}
	if status != locatorResolutionFound {
		return helpResolutionFailure(subject, expectation.argumentTermExpected(), detail)
	}
	switch expectation.property {
	case helpArgumentTermKind:
		if term.Kind == expectation.kind {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "kind", evaluationText(fmt.Sprint(expectation.kind)), evaluationText(fmt.Sprint(term.Kind)))
	case helpArgumentTermName:
		if term.Name == expectation.text {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "name", evaluationText(expectation.text), evaluationText(term.Name))
	case helpArgumentTermLiteral:
		if term.Literal == expectation.text {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "literal", evaluationText(expectation.text), evaluationText(term.Literal))
	case helpArgumentTermType:
		if term.Type == expectation.valueType {
			return provedResult()
		}
		field := valueTypeField{
			value: expectation.valueType,
		}
		observedField := valueTypeField{
			value: term.Type,
		}
		return helpPropertyMismatch(subject, "type", field, observedField)
	case helpArgumentTermCount:
		if term.Count == expectation.countRange {
			return provedResult()
		}
		field := countRangeField{
			value: expectation.countRange,
		}
		observedField := countRangeField{
			value: term.Count,
		}
		return helpPropertyMismatch(subject, "count", field, observedField)
	case helpArgumentTermHidden:
		if term.Hidden == expectation.boolean {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "hidden", evaluationBool(expectation.boolean), evaluationBool(term.Hidden))
	case helpArgumentTermSensitive:
		if term.Sensitive == expectation.boolean {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "sensitive", evaluationBool(expectation.boolean), evaluationBool(term.Sensitive))
	case helpArgumentTermTokenLanguageFragmentCount:
		actual := 0
		if term.TokenLanguage != nil {
			actual = len(term.TokenLanguage.Fragments)
		}
		if actual == expectation.count {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "token-language-fragment-count", evaluationText(strconv.Itoa(expectation.count)), evaluationText(strconv.Itoa(actual)))
	case helpArgumentTermValueDomainEntryCount:
		actual := len(valueDomainUseEntries(term.ValueDomainUses, expectation.domainKind))
		if actual == expectation.count {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "value-domain-entry-count", evaluationText(strconv.Itoa(expectation.count)), evaluationText(strconv.Itoa(actual)))
	default:
		panic("clitest: invalid Help argument-term expectation")
	}
}

func (expectation helpArgumentTermExpectation) argumentTermExpected() evaluationField {
	switch expectation.property {
	case helpArgumentTermKind:
		return property(evaluationText("kind"), evaluationText(fmt.Sprint(expectation.kind)))
	case helpArgumentTermName:
		return property(evaluationText("name"), evaluationText(expectation.text))
	case helpArgumentTermLiteral:
		return property(evaluationText("literal"), evaluationText(expectation.text))
	case helpArgumentTermType:
		field := valueTypeField{
			value: expectation.valueType,
		}
		return property(evaluationText("type"), field)
	case helpArgumentTermCount:
		field := countRangeField{
			value: expectation.countRange,
		}
		return property(evaluationText("count"), field)
	case helpArgumentTermHidden:
		return property(evaluationText("hidden"), evaluationText(strconv.FormatBool(expectation.boolean)))
	case helpArgumentTermSensitive:
		return property(semanticRequirementSensitive, evaluationText(strconv.FormatBool(expectation.boolean)))
	case helpArgumentTermTokenLanguageFragmentCount:
		return property(evaluationText("token-language-fragment-count"), evaluationText(strconv.Itoa(expectation.count)))
	case helpArgumentTermValueDomainEntryCount:
		return property(evaluationText("value-domain-entry-count"), evaluationText(strconv.Itoa(expectation.count)))
	default:
		return evaluationText("valid argument-term expectation")
	}
}

func (argumentFormOutcomeExpectationValue) inspectionExpectation() {}

func (expectation argumentFormOutcomeExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentFormLocator(expectation.locator, inspection)
	expected := evaluationText(fmt.Sprintf("outcome %v", expectation.outcome))
	if resolution.status == locatorResolutionNotProvable {
		return notProvableResult(evaluationText("argument form"), expected, resolution.detail)
	}
	if resolution.status != locatorResolutionFound {
		return disprovedResult(evaluationText("argument form"), expected, resolution.detail)
	}
	if resolution.value.Outcome == expectation.outcome {
		return provedResult()
	}
	return disprovedResult(evaluationText("argument form"), expected, evaluationText(fmt.Sprintf("outcome %v", resolution.value.Outcome)))
}

func (argumentFormSelectedExpectationValue) inspectionExpectation() {}

func (expectation argumentFormSelectedExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionArgumentFormLocator(expectation.locator, inspection)
	expected := evaluationText(selectionPresenceWord(expectation.selected))
	if resolution.status == locatorResolutionNotProvable {
		return notProvableResult(evaluationText("argument form"), expected, resolution.detail)
	}
	if resolution.status != locatorResolutionFound {
		return disprovedResult(evaluationText("argument form"), expected, resolution.detail)
	}
	command, ok := inspection.Commands[resolution.value.Command]
	if !ok {
		if inspection.Truncated {
			return notProvableResult(evaluationText("argument form"), expected, evaluationText("owning command details are absent from a truncated inspection"))
		}
		return disprovedResult(evaluationText("argument form"), expected, evaluationText("owning command is unavailable"))
	}
	found := command.ArgumentFormSelection.Selected == resolution.value.ID
	if found == expectation.selected {
		return provedResult()
	}
	return disprovedResult(evaluationText("argument form"), expected, evaluationText(fmt.Sprintf("selected form %v", command.ArgumentFormSelection.Selected)))
}
