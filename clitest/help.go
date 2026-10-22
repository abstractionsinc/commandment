package clitest

import (
	"fmt"
	"slices"
	"strconv"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

// HelpPageCommandGroupLocator identifies one command group in the returned cli.HelpPage presentation artifact.
// Title selection requires an exact single match; index selection follows HelpPage.CommandGroups presentation order.
// The locator describes a expectation subject only and does not create a second command-group model.
// Its zero value is invalid.
type HelpPageCommandGroupLocator struct {
	_        [0]func()
	selector helpIndexSelector
}

// HelpPageCommandLocator identifies one command row in the returned, locale-selected cli.HelpPage presentation artifact.
// HelpCommand searches exact names across rendered command groups, while group-scoped and index relations retain their
// presentation scope and order. Ambiguous name selection fails rather than choosing an arbitrary row.
// Its zero value is invalid.
type HelpPageCommandLocator struct {
	_        [0]func()
	group    *HelpPageCommandGroupLocator
	selector helpIndexSelector
	global   bool
}

// HelpPageArgumentFormLocator identifies an argument form by its position in HelpPage.ArgumentForms.
// The position is a presentation-order coordinate in the complete HelpPage, not an independently invented declaration
// identity.
// Its zero value is invalid.
type HelpPageArgumentFormLocator struct {
	_     [0]func()
	index int
	valid bool
}

// HelpPageArgumentTermLocator identifies an argument term by its position within a located, locale-selected Help
// argument form.
// It follows the HelpPage presentation relationship and exposes only facts already present on cli.HelpArgumentTerm.
// Its zero value is invalid.
type HelpPageArgumentTermLocator struct {
	_     [0]func()
	form  HelpPageArgumentFormLocator
	index int
}

// HelpPageFlagGroupLocator identifies one flag group in the returned cli.HelpPage presentation artifact.
// Title selection requires an exact single match; index selection follows HelpPage.FlagGroups presentation order.
// Its zero value is invalid.
type HelpPageFlagGroupLocator struct {
	_        [0]func()
	selector helpIndexSelector
}

// HelpPageFlagLocator identifies one ordinary or action-backed flag row in the returned, locale-selected cli.HelpPage
// presentation artifact.
// Group relations preserve group scope; spelling selection is exact and rejects ambiguity. Ordinary Flag and ActionFlag
// selectors keep the public distinction between semantic Flag rows and ActionEntry-backed syntax.
// Its zero value is invalid.
type HelpPageFlagLocator struct {
	_          [0]func()
	group      *HelpPageFlagGroupLocator
	selector   helpIndexSelector
	actionOnly bool
}

// HelpPageFlagSpellingLocator identifies a Help flag-spelling node through exact spelling, ordered flag membership, or
// the public replacement relation.
// HelpPage.FlagSpellings is an identity map rather than an ordering source; ordered spelling traversal comes from a
// HelpFlag.FlagSpellings relationship.
// Its zero value is invalid.
type HelpPageFlagSpellingLocator struct {
	_           [0]func()
	selector    helpIndexSelector
	flag        *HelpPageFlagLocator
	id          cli.FlagSpellingID
	byID        bool
	replacement *HelpPageFlagSpellingLocator
}

// HelpPageValueDomainEntryLocator identifies a locale-selected Help value-domain entry through exact name, an ordered
// term/flag domain-use relationship, or canonical/replacement relations.
// The locator follows HelpPage.ValueDomainEntries closure but does not treat closure-only entries as independently
// rendered rows or create a replacement model beyond the direct public relations.
// Its zero value is invalid.
type HelpPageValueDomainEntryLocator struct {
	_        [0]func()
	selector helpIndexSelector
	term     *HelpPageArgumentTermLocator
	flag     *HelpPageFlagLocator
	kind     cli.ValueDomainUseKind
	index    int
	byUse    bool
	id       cli.ValueDomainEntryID
	byID     bool
	parent   *HelpPageValueDomainEntryLocator
	relation helpValueDomainRelation
}

// HelpLabels returns an expectation that the localized presentation labels in HelpPage.Labels exactly equal expected.
func HelpLabels(expected cli.HelpLabels) HelpPresentationExpectation {
	expectation := helpLabelsExpectation{
		expected: expected,
	}
	return expectation
}

// HelpUsageExactly returns an expectation that HelpPage.Usage exactly equals a snapshot of the supplied lines in
// presentation order.
func HelpUsageExactly(expected ...string) HelpPresentationExpectation {
	expectation := helpPathExpectation{
		property: helpUsage,
		expected: slices.Clone(expected),
	}
	return expectation
}

// HelpSelectedPathExactly returns an expectation that HelpPage.SelectedPath exactly equals a snapshot of the supplied
// path components in order.
func HelpSelectedPathExactly(expected ...string) HelpPresentationExpectation {
	expectation := helpPathExpectation{
		property: helpSelectedPath,
		expected: slices.Clone(expected),
	}
	return expectation
}

// HelpRequestedPathExactly returns an expectation that HelpPage.RequestedPath exactly equals a snapshot of the supplied
// path components in order.
func HelpRequestedPathExactly(expected ...string) HelpPresentationExpectation {
	expectation := helpPathExpectation{
		property: helpRequestedPath,
		expected: slices.Clone(expected),
	}
	return expectation
}

// ResolveDeclaration resolves locator exactly against the supplied SchemaResult and returns the same locator type
// rebound to that exact public declaration identity.
// It fails the test when the locator is absent, ambiguous, not provable, unsupported, or resolves to an identity not
// contained in the Schema. The rebinding adds no new semantic identity; it makes an existing Schema identity available
// to later relation expectations such as Help action-entry correlation.
func ResolveDeclaration[L DeclarationIdentityLocator](t testing.TB, schema SchemaResult, locator L) L {
	t.Helper()
	if schema.state == nil {
		failEvaluationResults(t, disprovedResult(evaluationText("declaration locator"), evaluationText("valid schema result"), evaluationText("schema result is invalid")))
		return locator
	}
	identity, status, _ := resolveExpectedDeclarationIdentity(locator, &schema.state.observation, nil, cli.Inspection{})
	if status != locatorResolutionFound {
		detail := "locator is invalid"
		disposition := evaluationDisproved
		switch status {
		case locatorResolutionAbsent:
			detail = "locator is absent"
		case locatorResolutionAmbiguous:
			detail = "locator is ambiguous"
		case locatorResolutionNotProvable:
			detail = "locator is not provable"
			disposition = evaluationNotProvable
		case locatorResolutionInvalid, locatorResolutionFound:
		}
		outcome := disprovedResult(evaluationText("declaration locator"), evaluationText("resolve exactly"), evaluationText(detail))
		if disposition == evaluationNotProvable {
			outcome = notProvableResult(evaluationText("declaration locator"), evaluationText("resolve exactly"), evaluationText(detail))
		}
		failEvaluationResults(t, outcome)
		return locator
	}
	if !schemaContainsDeclaration(schema.state.observation, identity) {
		failEvaluationResults(
			t,
			disprovedResult(
				evaluationText("declaration locator"),
				evaluationText("resolved identity present in schema"),
				evaluationText("resolved identity is absent from schema"),
			),
		)
		return locator
	}
	var rebound any
	switch any(locator).(type) {
	case CommandEntryLocator:
		rebound = commandEntryIDLocator(identity.CommandEntry)
	case ActionEntryLocator:
		rebound = actionEntryIDLocator(identity.ActionEntry)
	case FlagLocator:
		rebound = flagIDLocator(identity.Flag)
	case FlagSpellingLocator:
		rebound = flagSpellingIDLocator(identity.FlagSpelling)
	case ArgumentFormLocator:
		rebound = argumentFormIDLocator(identity.ArgumentForm)
	case ArgumentTermLocator:
		rebound = argumentTermIDLocator(identity.ArgumentTerm)
	case EnvironmentInputLocator:
		rebound = environmentInputIDLocator(identity.EnvironmentInput)
	case ValueDomainEntryLocator:
		rebound = valueDomainEntryIDLocator(identity.ValueDomainEntry)
	default:
		failEvaluationResults(
			t,
			disprovedResult(
				evaluationText("declaration locator"),
				evaluationText("supported declaration locator"),
				evaluationText(fmt.Sprintf("unsupported locator type %T", locator)),
			),
		)
		return locator
	}
	typed, ok := rebound.(L)
	if !ok {
		failEvaluationResults(
			t,
			disprovedResult(
				evaluationText("declaration locator"),
				evaluationText("rebound locator preserves type"),
				evaluationText(fmt.Sprintf("rebound locator type %T does not match source locator type %T", rebound, locator)),
			),
		)
		return locator
	}
	return typed
}

// HelpCommandGroup returns a command-group locator that selects an exact title match in HelpPage.CommandGroups.
func HelpCommandGroup(title string) HelpPageCommandGroupLocator {
	locator := HelpPageCommandGroupLocator{
		selector: helpTextSelector(title),
	}
	return locator
}

// HelpCommandGroupAt returns a command-group locator for the group at index in HelpPage.CommandGroups presentation
// order.
func HelpCommandGroupAt(index int) HelpPageCommandGroupLocator {
	locator := HelpPageCommandGroupLocator{
		selector: helpIndexSelectorAt(index),
	}
	return locator
}

// HelpCommand returns a command locator that selects an exact name match across HelpPage command groups and rejects
// ambiguity.
func HelpCommand(name string) HelpPageCommandLocator {
	locator := HelpPageCommandLocator{
		selector: helpTextSelector(name),
		global:   true,
	}
	return locator
}

// HelpArgumentFormAt returns an argument-form locator for index in HelpPage.ArgumentForms presentation order.
func HelpArgumentFormAt(index int) HelpPageArgumentFormLocator {
	locator := HelpPageArgumentFormLocator{
		index: index,
		valid: true,
	}
	return locator
}

// HelpFlagGroup returns a flag-group locator that selects an exact title match in HelpPage.FlagGroups.
func HelpFlagGroup(title string) HelpPageFlagGroupLocator {
	locator := HelpPageFlagGroupLocator{
		selector: helpTextSelector(title),
	}
	return locator
}

// HelpFlagGroupAt returns a flag-group locator for the group at index in HelpPage.FlagGroups presentation order.
func HelpFlagGroupAt(index int) HelpPageFlagGroupLocator {
	locator := HelpPageFlagGroupLocator{
		selector: helpIndexSelectorAt(index),
	}
	return locator
}

// HelpFlagSpelling returns a flag-spelling locator that requires one exact spelling-text match in the HelpPage spelling
// map.
func HelpFlagSpelling(spelling string) HelpPageFlagSpellingLocator {
	locator := HelpPageFlagSpellingLocator{
		selector: helpTextSelector(spelling),
	}
	return locator
}

// HelpValueDomainEntry returns a value-domain-entry locator that requires one exact name match in the HelpPage
// value-domain-entry map.
func HelpValueDomainEntry(name string) HelpPageValueDomainEntryLocator {
	locator := HelpPageValueDomainEntryLocator{
		selector: helpTextSelector(name),
	}
	return locator
}

// Exists returns a Help presentation expectation that the located help command group resolves to exactly one row;
// ambiguous selectors do not count as existence.
func (locator HelpPageCommandGroupLocator) Exists() HelpPresentationExpectation {
	expectation := helpCommandGroupExpectation{
		locator:  locator,
		property: helpCommandGroupExists,
		exists:   true,
	}
	return expectation
}

// DoesNotExist returns a Help presentation expectation that the located help command group is absent; ambiguous or
// invalid selectors fail rather than checking absence.
func (locator HelpPageCommandGroupLocator) DoesNotExist() HelpPresentationExpectation {
	expectation := helpCommandGroupExpectation{
		locator:  locator,
		property: helpCommandGroupExists,
		exists:   false,
	}
	return expectation
}

// Kind returns a Help presentation expectation that the located help command group has Kind exactly equal to expected.
func (locator HelpPageCommandGroupLocator) Kind(expected cli.HelpGroupKind) HelpPresentationExpectation {
	expectation := helpCommandGroupExpectation{
		locator:  locator,
		property: helpCommandGroupKind,
		kind:     expected,
	}
	return expectation
}

// Title returns a Help presentation expectation that the located help command group has Title exactly equal to
// expected.
func (locator HelpPageCommandGroupLocator) Title(expected string) HelpPresentationExpectation {
	expectation := helpCommandGroupExpectation{
		locator:  locator,
		property: helpCommandGroupTitle,
		text:     expected,
	}
	return expectation
}

// Order returns a Help presentation expectation that the located help command group has the requested public
// presentation Order.
func (locator HelpPageCommandGroupLocator) Order(expected cli.HelpGroupOrder) HelpPresentationExpectation {
	expectation := helpCommandGroupExpectation{
		locator:  locator,
		property: helpCommandGroupOrder,
		order:    expected,
	}
	return expectation
}

// Command returns a command locator scoped to this command group and requiring one exact command-name match within the
// group.
func (locator HelpPageCommandGroupLocator) Command(name string) HelpPageCommandLocator {
	clone := locator
	helpPageCommandLocator := HelpPageCommandLocator{
		group:    &clone,
		selector: helpTextSelector(name),
	}
	return helpPageCommandLocator
}

// CommandAt returns a locator for the command at index in the located help command group's public presentation order.
func (locator HelpPageCommandGroupLocator) CommandAt(index int) HelpPageCommandLocator {
	clone := locator
	helpPageCommandLocator := HelpPageCommandLocator{
		group:    &clone,
		selector: helpIndexSelectorAt(index),
	}
	return helpPageCommandLocator
}

// CommandCount returns a Help presentation expectation that the located help command group has exactly the expected
// number of command rows.
func (locator HelpPageCommandGroupLocator) CommandCount(expected int) HelpPresentationExpectation {
	expectation := helpCommandGroupExpectation{
		locator:  locator,
		property: helpCommandGroupCommandCount,
		count:    expected,
	}
	return expectation
}

// Exists returns a Help presentation expectation that the located help command row resolves to exactly one row;
// ambiguous selectors do not count as existence.
func (locator HelpPageCommandLocator) Exists() HelpPresentationExpectation {
	expectation := helpCommandExpectation{
		locator:  locator,
		property: helpCommandExists,
		exists:   true,
	}
	return expectation
}

// DoesNotExist returns a Help presentation expectation that the located help command row is absent; ambiguous or
// invalid selectors fail rather than checking absence.
func (locator HelpPageCommandLocator) DoesNotExist() HelpPresentationExpectation {
	expectation := helpCommandExpectation{
		locator:  locator,
		property: helpCommandExists,
		exists:   false,
	}
	return expectation
}

// Kind returns a Help presentation expectation that the located help command row has Kind exactly equal to expected.
func (locator HelpPageCommandLocator) Kind(expected cli.HelpCommandKind) HelpPresentationExpectation {
	expectation := helpCommandExpectation{
		locator:  locator,
		property: helpCommandKind,
		kind:     expected,
	}
	return expectation
}

// Origin returns a Help presentation expectation that the located help command row has DeclarationOrigin exactly equal
// to expected.
func (locator HelpPageCommandLocator) Origin(expected cli.DeclarationOrigin) HelpPresentationExpectation {
	expectation := helpCommandExpectation{
		locator:  locator,
		property: helpCommandOrigin,
		origin:   expected,
	}
	return expectation
}

// Name returns a Help presentation expectation that the located help command row has Name exactly equal to expected.
func (locator HelpPageCommandLocator) Name(expected string) HelpPresentationExpectation {
	expectation := helpCommandExpectation{
		locator:  locator,
		property: helpCommandName,
		text:     expected,
	}
	return expectation
}

// Description returns a Help presentation expectation that the located help command row has Description exactly equal
// to expected.
func (locator HelpPageCommandLocator) Description(expected cli.Description) HelpPresentationExpectation {
	expectation := helpCommandExpectation{
		locator:     locator,
		property:    helpCommandDescription,
		description: expected,
	}
	return expectation
}

// Hidden returns a Help presentation expectation that the located help command row is hidden (Hidden is true).
func (locator HelpPageCommandLocator) Hidden() HelpPresentationExpectation {
	expectation := helpCommandExpectation{
		locator:  locator,
		property: helpCommandHidden,
		boolean:  true,
	}
	return expectation
}

// Visible returns a Help presentation expectation that the located help command row is visible (Hidden is false).
func (locator HelpPageCommandLocator) Visible() HelpPresentationExpectation {
	expectation := helpCommandExpectation{
		locator:  locator,
		property: helpCommandHidden,
		boolean:  false,
	}
	return expectation
}

// Default returns a Help presentation expectation that the located help command row is marked default (Default is
// true).
func (locator HelpPageCommandLocator) Default() HelpPresentationExpectation {
	expectation := helpCommandExpectation{
		locator:  locator,
		property: helpCommandDefault,
		boolean:  true,
	}
	return expectation
}

// NotDefault returns a Help presentation expectation that the located help command row is not marked default (Default
// is false).
func (locator HelpPageCommandLocator) NotDefault() HelpPresentationExpectation {
	expectation := helpCommandExpectation{
		locator:  locator,
		property: helpCommandDefault,
		boolean:  false,
	}
	return expectation
}

// Runnable returns a Help presentation expectation that the located help command row is runnable in the Help
// presentation (Runnable is true).
func (locator HelpPageCommandLocator) Runnable() HelpPresentationExpectation {
	expectation := helpCommandExpectation{
		locator:  locator,
		property: helpCommandRunnable,
		boolean:  true,
	}
	return expectation
}

// NotRunnable returns a Help presentation expectation that the located help command row is not runnable in the Help
// presentation (Runnable is false).
func (locator HelpPageCommandLocator) NotRunnable() HelpPresentationExpectation {
	expectation := helpCommandExpectation{
		locator:  locator,
		property: helpCommandRunnable,
		boolean:  false,
	}
	return expectation
}

// HasSubcommands returns a Help presentation expectation that the located help command row reports subcommands
// (HasSubcommands is true).
func (locator HelpPageCommandLocator) HasSubcommands() HelpPresentationExpectation {
	expectation := helpCommandExpectation{
		locator:  locator,
		property: helpCommandHasSubcommands,
		boolean:  true,
	}
	return expectation
}

// NoSubcommands returns a Help presentation expectation that the located help command row reports no subcommands
// (HasSubcommands is false).
func (locator HelpPageCommandLocator) NoSubcommands() HelpPresentationExpectation {
	expectation := helpCommandExpectation{
		locator:  locator,
		property: helpCommandHasSubcommands,
		boolean:  false,
	}
	return expectation
}

// CommandEntryIdentityPresent returns a Help presentation expectation that the located command row carries a nonzero
// CommandEntry identity.
func (locator HelpPageCommandLocator) CommandEntryIdentityPresent() HelpPresentationExpectation {
	expectation := helpCommandExpectation{
		locator:  locator,
		property: helpCommandCommandEntryPresent,
	}
	return expectation
}

// ActionEntryIdentityPresent returns a Help presentation expectation that the located help command row carries a
// nonzero ActionEntry identity.
func (locator HelpPageCommandLocator) ActionEntryIdentityPresent() HelpPresentationExpectation {
	expectation := helpCommandExpectation{
		locator:  locator,
		property: helpCommandActionEntryPresent,
	}
	return expectation
}

// ActionEntry returns a Help presentation expectation that the located help command row's ActionEntry identity equals
// expected.
// expected must already be rebound to an exact ActionEntry identity, for example with ResolveDeclaration; the expectation
// compares the real public identity relationship rather than a presentation spelling.
func (locator HelpPageCommandLocator) ActionEntry(expected ActionEntryLocator) HelpPresentationExpectation {
	expectation := helpCommandExpectation{
		locator:     locator,
		property:    helpCommandActionEntry,
		actionEntry: expected.state,
	}
	return expectation
}

// Lifecycle returns a Help presentation expectation that the located help command row has LifecycleState exactly equal
// to expected.
func (locator HelpPageCommandLocator) Lifecycle(expected cli.LifecycleState) HelpPresentationExpectation {
	expectation := helpCommandExpectation{
		locator:   locator,
		property:  helpCommandLifecycle,
		lifecycle: expected,
	}
	return expectation
}

// Exists returns a Help presentation expectation that the located help argument form resolves to exactly one row;
// ambiguous selectors do not count as existence.
func (locator HelpPageArgumentFormLocator) Exists() HelpPresentationExpectation {
	expectation := helpArgumentFormExpectation{
		locator:  locator,
		property: helpArgumentFormExists,
		exists:   true,
	}
	return expectation
}

// DoesNotExist returns a Help presentation expectation that the located help argument form is absent; ambiguous or
// invalid selectors fail rather than checking absence.
func (locator HelpPageArgumentFormLocator) DoesNotExist() HelpPresentationExpectation {
	expectation := helpArgumentFormExpectation{
		locator:  locator,
		property: helpArgumentFormExists,
		exists:   false,
	}
	return expectation
}

// TermCount returns a Help presentation expectation that the located help argument form has exactly the expected number
// of terms.
func (locator HelpPageArgumentFormLocator) TermCount(expected int) HelpPresentationExpectation {
	expectation := helpArgumentFormExpectation{
		locator:  locator,
		property: helpArgumentFormTermCount,
		count:    expected,
	}
	return expectation
}

// TermAt returns a locator for the term at index in the located help argument form's public presentation order.
func (locator HelpPageArgumentFormLocator) TermAt(index int) HelpPageArgumentTermLocator {
	helpPageArgumentTermLocator := HelpPageArgumentTermLocator{
		form:  locator,
		index: index,
	}
	return helpPageArgumentTermLocator
}

// Exists returns a Help presentation expectation that the located help argument term resolves to exactly one row;
// ambiguous selectors do not count as existence.
func (locator HelpPageArgumentTermLocator) Exists() HelpPresentationExpectation {
	expectation := helpArgumentTermExpectation{
		locator:  locator,
		property: helpArgumentTermExists,
		exists:   true,
	}
	return expectation
}

// DoesNotExist returns a Help presentation expectation that the located help argument term is absent; ambiguous or
// invalid selectors fail rather than checking absence.
func (locator HelpPageArgumentTermLocator) DoesNotExist() HelpPresentationExpectation {
	expectation := helpArgumentTermExpectation{
		locator:  locator,
		property: helpArgumentTermExists,
		exists:   false,
	}
	return expectation
}

// Kind returns a Help presentation expectation that the located help argument term has Kind exactly equal to expected.
func (locator HelpPageArgumentTermLocator) Kind(expected cli.ArgumentTermKind) HelpPresentationExpectation {
	expectation := helpArgumentTermExpectation{
		locator:  locator,
		property: helpArgumentTermKind,
		kind:     expected,
	}
	return expectation
}

// Name returns a Help presentation expectation that the located help argument term has Name exactly equal to expected.
func (locator HelpPageArgumentTermLocator) Name(expected string) HelpPresentationExpectation {
	expectation := helpArgumentTermExpectation{
		locator:  locator,
		property: helpArgumentTermName,
		text:     expected,
	}
	return expectation
}

// Literal returns a Help presentation expectation that the located help argument term has Literal exactly equal to
// expected.
func (locator HelpPageArgumentTermLocator) Literal(expected string) HelpPresentationExpectation {
	expectation := helpArgumentTermExpectation{
		locator:  locator,
		property: helpArgumentTermLiteral,
		text:     expected,
	}
	return expectation
}

// Type returns a Help presentation expectation that the located help argument term has ValueType exactly equal to
// expected.
func (locator HelpPageArgumentTermLocator) Type(expected cli.ValueType) HelpPresentationExpectation {
	expectation := helpArgumentTermExpectation{
		locator:   locator,
		property:  helpArgumentTermType,
		valueType: expected,
	}
	return expectation
}

// Count returns a Help presentation expectation that the located help argument term has Count exactly equal to
// expected.
func (locator HelpPageArgumentTermLocator) Count(expected cli.CountRange) HelpPresentationExpectation {
	expectation := helpArgumentTermExpectation{
		locator:    locator,
		property:   helpArgumentTermCount,
		countRange: expected,
	}
	return expectation
}

// Hidden returns a Help presentation expectation that the located help argument term is hidden (Hidden is true).
func (locator HelpPageArgumentTermLocator) Hidden() HelpPresentationExpectation {
	expectation := helpArgumentTermExpectation{
		locator:  locator,
		property: helpArgumentTermHidden,
		boolean:  true,
	}
	return expectation
}

// Visible returns a Help presentation expectation that the located help argument term is visible (Hidden is false).
func (locator HelpPageArgumentTermLocator) Visible() HelpPresentationExpectation {
	expectation := helpArgumentTermExpectation{
		locator:  locator,
		property: helpArgumentTermHidden,
		boolean:  false,
	}
	return expectation
}

// Sensitive returns a Help presentation expectation that the located help argument term is marked sensitive (Sensitive
// is true).
// It checks the public presentation sensitivity marker and does not attempt to recover any suppressed sensitive value.
func (locator HelpPageArgumentTermLocator) Sensitive() HelpPresentationExpectation {
	expectation := helpArgumentTermExpectation{
		locator:  locator,
		property: helpArgumentTermSensitive,
		boolean:  true,
	}
	return expectation
}

// NotSensitive returns a Help presentation expectation that the located help argument term is not marked sensitive
// (Sensitive is false).
// It tests the public marker only; absence of the marker is not a license to reinterpret unrelated value checknance.
func (locator HelpPageArgumentTermLocator) NotSensitive() HelpPresentationExpectation {
	expectation := helpArgumentTermExpectation{
		locator:  locator,
		property: helpArgumentTermSensitive,
		boolean:  false,
	}
	return expectation
}

// TokenLanguageFragmentCount returns a Help presentation expectation that the located help argument term has exactly
// the expected number of token-language fragments.
func (locator HelpPageArgumentTermLocator) TokenLanguageFragmentCount(expected int) HelpPresentationExpectation {
	expectation := helpArgumentTermExpectation{
		locator:  locator,
		property: helpArgumentTermTokenLanguageFragmentCount,
		count:    expected,
	}
	return expectation
}

// ValueDomainEntryCount returns a Help presentation expectation for the number of value-domain entries referenced by
// all help argument term uses of kind, preserving each use's public entry order for related At selection.
func (locator HelpPageArgumentTermLocator) ValueDomainEntryCount(kind cli.ValueDomainUseKind, expected int) HelpPresentationExpectation {
	expectation := helpArgumentTermExpectation{
		locator:    locator,
		property:   helpArgumentTermValueDomainEntryCount,
		domainKind: kind,
		count:      expected,
	}
	return expectation
}

// ValueDomainEntryAt returns a value-domain-entry locator for index among the located help argument term's public
// domain-use entries of kind, in their retained use order.
func (locator HelpPageArgumentTermLocator) ValueDomainEntryAt(kind cli.ValueDomainUseKind, index int) HelpPageValueDomainEntryLocator {
	clone := locator
	helpPageValueDomainEntryLocator := HelpPageValueDomainEntryLocator{
		term:  &clone,
		kind:  kind,
		index: index,
		byUse: true,
	}
	return helpPageValueDomainEntryLocator
}

// Exists returns a Help presentation expectation that the located help flag group resolves to exactly one row;
// ambiguous selectors do not count as existence.
func (locator HelpPageFlagGroupLocator) Exists() HelpPresentationExpectation {
	expectation := helpFlagGroupExpectation{
		locator:  locator,
		property: helpFlagGroupExists,
		exists:   true,
	}
	return expectation
}

// DoesNotExist returns a Help presentation expectation that the located help flag group is absent; ambiguous or invalid
// selectors fail rather than checking absence.
func (locator HelpPageFlagGroupLocator) DoesNotExist() HelpPresentationExpectation {
	expectation := helpFlagGroupExpectation{
		locator:  locator,
		property: helpFlagGroupExists,
		exists:   false,
	}
	return expectation
}

// Kind returns a Help presentation expectation that the located help flag group has Kind exactly equal to expected.
func (locator HelpPageFlagGroupLocator) Kind(expected cli.HelpGroupKind) HelpPresentationExpectation {
	expectation := helpFlagGroupExpectation{
		locator:  locator,
		property: helpFlagGroupKind,
		kind:     expected,
	}
	return expectation
}

// Title returns a Help presentation expectation that the located help flag group has Title exactly equal to expected.
func (locator HelpPageFlagGroupLocator) Title(expected string) HelpPresentationExpectation {
	expectation := helpFlagGroupExpectation{
		locator:  locator,
		property: helpFlagGroupTitle,
		text:     expected,
	}
	return expectation
}

// Order returns a Help presentation expectation that the located help flag group has the requested public presentation
// Order.
func (locator HelpPageFlagGroupLocator) Order(expected cli.HelpGroupOrder) HelpPresentationExpectation {
	expectation := helpFlagGroupExpectation{
		locator:  locator,
		property: helpFlagGroupOrder,
		order:    expected,
	}
	return expectation
}

// Flag returns an ordinary-flag locator scoped to this flag group and requiring one exact spelling match;
// ActionEntry-backed rows are excluded.
func (locator HelpPageFlagGroupLocator) Flag(spelling string) HelpPageFlagLocator {
	clone := locator
	helpPageFlagLocator := HelpPageFlagLocator{
		group:    &clone,
		selector: helpTextSelector(spelling),
	}
	return helpPageFlagLocator
}

// ActionFlag returns an action-flag locator scoped to this flag group and requiring one exact action-spelling match;
// ordinary semantic Flag rows are excluded.
func (locator HelpPageFlagGroupLocator) ActionFlag(spelling string) HelpPageFlagLocator {
	clone := locator
	helpPageFlagLocator := HelpPageFlagLocator{
		group:      &clone,
		selector:   helpTextSelector(spelling),
		actionOnly: true,
	}
	return helpPageFlagLocator
}

// FlagAt returns a locator for the flag at index in the located help flag group's public presentation order.
func (locator HelpPageFlagGroupLocator) FlagAt(index int) HelpPageFlagLocator {
	clone := locator
	helpPageFlagLocator := HelpPageFlagLocator{
		group:    &clone,
		selector: helpIndexSelectorAt(index),
	}
	return helpPageFlagLocator
}

// FlagCount returns a Help presentation expectation that the located help flag group has exactly the expected number of
// flag rows.
func (locator HelpPageFlagGroupLocator) FlagCount(expected int) HelpPresentationExpectation {
	expectation := helpFlagGroupExpectation{
		locator:  locator,
		property: helpFlagGroupFlagCount,
		count:    expected,
	}
	return expectation
}

// Exists returns a Help presentation expectation that the located help flag row resolves to exactly one row; ambiguous
// selectors do not count as existence.
func (locator HelpPageFlagLocator) Exists() HelpPresentationExpectation {
	expectation := helpFlagExpectation{
		locator:  locator,
		property: helpFlagExists,
		exists:   true,
	}
	return expectation
}

// DoesNotExist returns a Help presentation expectation that the located help flag row is absent; ambiguous or invalid
// selectors fail rather than checking absence.
func (locator HelpPageFlagLocator) DoesNotExist() HelpPresentationExpectation {
	expectation := helpFlagExpectation{
		locator:  locator,
		property: helpFlagExists,
		exists:   false,
	}
	return expectation
}

// Origin returns a Help presentation expectation that the located help flag row has DeclarationOrigin exactly equal to
// expected.
func (locator HelpPageFlagLocator) Origin(expected cli.DeclarationOrigin) HelpPresentationExpectation {
	expectation := helpFlagExpectation{
		locator:  locator,
		property: helpFlagOrigin,
		origin:   expected,
	}
	return expectation
}

// Ordinary returns a Help presentation expectation that the located help flag row is an ordinary semantic Flag row
// (ActionEntry is zero).
func (locator HelpPageFlagLocator) Ordinary() HelpPresentationExpectation {
	expectation := helpFlagExpectation{
		locator:  locator,
		property: helpFlagOrdinary,
	}
	return expectation
}

// Action returns a Help presentation expectation that the located help flag row is ActionEntry-backed help syntax
// (ActionEntry is nonzero).
func (locator HelpPageFlagLocator) Action() HelpPresentationExpectation {
	expectation := helpFlagExpectation{
		locator:  locator,
		property: helpFlagAction,
	}
	return expectation
}

// ActionEntryIdentityPresent returns a Help presentation expectation that the located help flag row carries a nonzero
// ActionEntry identity.
func (locator HelpPageFlagLocator) ActionEntryIdentityPresent() HelpPresentationExpectation {
	expectation := helpFlagExpectation{
		locator:  locator,
		property: helpFlagActionEntryPresent,
	}
	return expectation
}

// ActionEntry returns a Help presentation expectation that the located help flag row's ActionEntry identity equals
// expected.
// expected must already be rebound to an exact ActionEntry identity, for example with ResolveDeclaration; the expectation
// compares the real public identity relationship rather than a presentation spelling.
func (locator HelpPageFlagLocator) ActionEntry(expected ActionEntryLocator) HelpPresentationExpectation {
	expectation := helpFlagExpectation{
		locator:     locator,
		property:    helpFlagActionEntry,
		actionEntry: expected.state,
	}
	return expectation
}

// SpellingCount returns a Help presentation expectation that the located help flag row has exactly the expected number
// of ordinary spelling identities.
func (locator HelpPageFlagLocator) SpellingCount(expected int) HelpPresentationExpectation {
	expectation := helpFlagExpectation{
		locator:  locator,
		property: helpFlagSpellingCount,
		count:    expected,
	}
	return expectation
}

// SpellingAt returns a locator for the spelling at index in the located help flag row's public presentation order.
func (locator HelpPageFlagLocator) SpellingAt(index int) HelpPageFlagSpellingLocator {
	clone := locator
	helpPageFlagSpellingLocator := HelpPageFlagSpellingLocator{
		flag:     &clone,
		selector: helpIndexSelectorAt(index),
	}
	return helpPageFlagSpellingLocator
}

// ActionSpelling returns a Help presentation expectation that the located help flag row has inline action spelling
// exactly equal to expected.
func (locator HelpPageFlagLocator) ActionSpelling(expected cli.FlagSpelling) HelpPresentationExpectation {
	expectation := helpFlagExpectation{
		locator:  locator,
		property: helpFlagActionSpelling,
		spelling: expected,
	}
	return expectation
}

// Description returns a Help presentation expectation that the located help flag row has Description exactly equal to
// expected.
func (locator HelpPageFlagLocator) Description(expected string) HelpPresentationExpectation {
	expectation := helpFlagExpectation{
		locator:  locator,
		property: helpFlagDescription,
		text:     expected,
	}
	return expectation
}

// OwnerPathExactly returns a Help presentation expectation that the located help flag row's OwnerPath exactly equals
// expected in path order.
func (locator HelpPageFlagLocator) OwnerPathExactly(expected ...string) HelpPresentationExpectation {
	expectation := helpFlagExpectation{
		locator:  locator,
		property: helpFlagOwnerPath,
		path:     slices.Clone(expected),
	}
	return expectation
}

// Placement returns a Help presentation expectation that the located help flag row has FlagPlacement exactly equal to
// expected.
func (locator HelpPageFlagLocator) Placement(expected cli.FlagPlacement) HelpPresentationExpectation {
	expectation := helpFlagExpectation{
		locator:   locator,
		property:  helpFlagPlacement,
		placement: expected,
	}
	return expectation
}

// Hidden returns a Help presentation expectation that the located help flag row is hidden (Hidden is true).
func (locator HelpPageFlagLocator) Hidden() HelpPresentationExpectation {
	expectation := helpFlagExpectation{
		locator:  locator,
		property: helpFlagHidden,
		boolean:  true,
	}
	return expectation
}

// Visible returns a Help presentation expectation that the located help flag row is visible (Hidden is false).
func (locator HelpPageFlagLocator) Visible() HelpPresentationExpectation {
	expectation := helpFlagExpectation{
		locator:  locator,
		property: helpFlagHidden,
		boolean:  false,
	}
	return expectation
}

// Sensitive returns a Help presentation expectation that the located help flag row is marked sensitive (Sensitive is
// true).
// It checks the public Help sensitivity marker without reconstructing or exposing sensitive value material.
func (locator HelpPageFlagLocator) Sensitive() HelpPresentationExpectation {
	expectation := helpFlagExpectation{
		locator:  locator,
		property: helpFlagSensitive,
		boolean:  true,
	}
	return expectation
}

// NotSensitive returns a Help presentation expectation that the located help flag row is not marked sensitive
// (Sensitive is false).
// It tests the public marker only and does not infer sensitivity state for other related declarations or inputs.
func (locator HelpPageFlagLocator) NotSensitive() HelpPresentationExpectation {
	expectation := helpFlagExpectation{
		locator:  locator,
		property: helpFlagSensitive,
		boolean:  false,
	}
	return expectation
}

// Lifecycle returns a Help presentation expectation that the located help flag row has LifecycleState exactly equal to
// expected.
func (locator HelpPageFlagLocator) Lifecycle(expected cli.LifecycleState) HelpPresentationExpectation {
	expectation := helpFlagExpectation{
		locator:   locator,
		property:  helpFlagLifecycle,
		lifecycle: expected,
	}
	return expectation
}

// ValueDomainEntryCount returns a Help presentation expectation for the number of value-domain entries referenced by
// all help flag row uses of kind, preserving each use's public entry order for related At selection.
func (locator HelpPageFlagLocator) ValueDomainEntryCount(kind cli.ValueDomainUseKind, expected int) HelpPresentationExpectation {
	expectation := helpFlagExpectation{
		locator:    locator,
		property:   helpFlagValueDomainEntryCount,
		domainKind: kind,
		count:      expected,
	}
	return expectation
}

// ValueDomainEntryAt returns a value-domain-entry locator for index among the located help flag row's public domain-use
// entries of kind, in their retained use order.
func (locator HelpPageFlagLocator) ValueDomainEntryAt(kind cli.ValueDomainUseKind, index int) HelpPageValueDomainEntryLocator {
	clone := locator
	helpPageValueDomainEntryLocator := HelpPageValueDomainEntryLocator{
		flag:  &clone,
		kind:  kind,
		index: index,
		byUse: true,
	}
	return helpPageValueDomainEntryLocator
}

// Exists returns a Help presentation expectation that the located help flag spelling resolves to exactly one row;
// ambiguous selectors do not count as existence.
func (locator HelpPageFlagSpellingLocator) Exists() HelpPresentationExpectation {
	expectation := helpFlagSpellingExpectation{
		locator:  locator,
		property: helpFlagSpellingExists,
		exists:   true,
	}
	return expectation
}

// DoesNotExist returns a Help presentation expectation that the located help flag spelling is absent; ambiguous or
// invalid selectors fail rather than checking absence.
func (locator HelpPageFlagSpellingLocator) DoesNotExist() HelpPresentationExpectation {
	expectation := helpFlagSpellingExpectation{
		locator:  locator,
		property: helpFlagSpellingExists,
		exists:   false,
	}
	return expectation
}

// Spelling returns a Help presentation expectation that the located help flag spelling has exact public FlagSpelling
// value equal to expected.
func (locator HelpPageFlagSpellingLocator) Spelling(expected cli.FlagSpelling) HelpPresentationExpectation {
	expectation := helpFlagSpellingExpectation{
		locator:  locator,
		property: helpFlagSpellingValue,
		spelling: expected,
	}
	return expectation
}

// Lifecycle returns a Help presentation expectation that the located help flag spelling has LifecycleState exactly
// equal to expected.
func (locator HelpPageFlagSpellingLocator) Lifecycle(expected cli.LifecycleState) HelpPresentationExpectation {
	expectation := helpFlagSpellingExpectation{
		locator:   locator,
		property:  helpFlagSpellingLifecycle,
		lifecycle: expected,
	}
	return expectation
}

// Replacement returns a locator that follows the located help flag spelling's direct public Replacement relation. It
// does not chase beyond the direct replacement relation.
func (locator HelpPageFlagSpellingLocator) Replacement() HelpPageFlagSpellingLocator {
	clone := locator
	helpPageFlagSpellingLocator := HelpPageFlagSpellingLocator{
		replacement: &clone,
	}
	return helpPageFlagSpellingLocator
}

// NotReplaced returns a Help presentation expectation that the located help flag spelling has no direct Replacement
// relation.
func (locator HelpPageFlagSpellingLocator) NotReplaced() HelpPresentationExpectation {
	expectation := helpFlagSpellingExpectation{
		locator:  locator,
		property: helpFlagSpellingNotReplaced,
	}
	return expectation
}

// Exists returns a Help presentation expectation that the located help value-domain entry resolves to exactly one row;
// ambiguous selectors do not count as existence.
func (locator HelpPageValueDomainEntryLocator) Exists() HelpPresentationExpectation {
	expectation := helpValueDomainEntryExpectation{
		locator:  locator,
		property: helpValueDomainEntryExists,
		exists:   true,
	}
	return expectation
}

// DoesNotExist returns a Help presentation expectation that the located help value-domain entry is absent; ambiguous or
// invalid selectors fail rather than checking absence.
func (locator HelpPageValueDomainEntryLocator) DoesNotExist() HelpPresentationExpectation {
	expectation := helpValueDomainEntryExpectation{
		locator:  locator,
		property: helpValueDomainEntryExists,
		exists:   false,
	}
	return expectation
}

// Kind returns a Help presentation expectation that the located help value-domain entry has Kind exactly equal to
// expected.
func (locator HelpPageValueDomainEntryLocator) Kind(expected cli.ValueDomainEntryKind) HelpPresentationExpectation {
	expectation := helpValueDomainEntryExpectation{
		locator:  locator,
		property: helpValueDomainEntryKind,
		kind:     expected,
	}
	return expectation
}

// Name returns a Help presentation expectation that the located help value-domain entry has Name exactly equal to
// expected.
func (locator HelpPageValueDomainEntryLocator) Name(expected string) HelpPresentationExpectation {
	expectation := helpValueDomainEntryExpectation{
		locator:  locator,
		property: helpValueDomainEntryName,
		text:     expected,
	}
	return expectation
}

// Description returns a Help presentation expectation that the located help value-domain entry has Description exactly
// equal to expected.
func (locator HelpPageValueDomainEntryLocator) Description(expected string) HelpPresentationExpectation {
	expectation := helpValueDomainEntryExpectation{
		locator:  locator,
		property: helpValueDomainEntryDescription,
		text:     expected,
	}
	return expectation
}

// Lifecycle returns a Help presentation expectation that the located help value-domain entry has LifecycleState exactly
// equal to expected.
func (locator HelpPageValueDomainEntryLocator) Lifecycle(expected cli.LifecycleState) HelpPresentationExpectation {
	expectation := helpValueDomainEntryExpectation{
		locator:   locator,
		property:  helpValueDomainEntryLifecycle,
		lifecycle: expected,
	}
	return expectation
}

// Canonical returns a locator that follows the located Help value-domain entry's direct public Canonical relation.
func (locator HelpPageValueDomainEntryLocator) Canonical() HelpPageValueDomainEntryLocator {
	clone := locator
	helpPageValueDomainEntryLocator := HelpPageValueDomainEntryLocator{
		parent:   &clone,
		relation: helpValueDomainRelationCanonical,
	}
	return helpPageValueDomainEntryLocator
}

// NoCanonical returns a Help presentation expectation that the located Help value-domain entry has no Canonical
// relation.
func (locator HelpPageValueDomainEntryLocator) NoCanonical() HelpPresentationExpectation {
	expectation := helpValueDomainEntryExpectation{
		locator:  locator,
		property: helpValueDomainEntryNoCanonical,
	}
	return expectation
}

// Replacement returns a locator that follows the located help value-domain entry's direct public Replacement relation.
// It does not chase beyond the direct replacement relation.
func (locator HelpPageValueDomainEntryLocator) Replacement() HelpPageValueDomainEntryLocator {
	clone := locator
	helpPageValueDomainEntryLocator := HelpPageValueDomainEntryLocator{
		parent:   &clone,
		relation: helpValueDomainRelationReplacement,
	}
	return helpPageValueDomainEntryLocator
}

// NotReplaced returns a Help presentation expectation that the located help value-domain entry has no direct
// Replacement relation.
func (locator HelpPageValueDomainEntryLocator) NotReplaced() HelpPresentationExpectation {
	expectation := helpValueDomainEntryExpectation{
		locator:  locator,
		property: helpValueDomainEntryNotReplaced,
	}
	return expectation
}

type helpIndexSelectorMode uint8

const (
	helpIndexSelectorInvalid helpIndexSelectorMode = iota
	helpIndexSelectorText
	helpIndexSelectorIndex
)

type helpIndexSelector struct {
	index int
	mode  helpIndexSelectorMode
	text  string
}

type helpValueDomainRelation uint8

const (
	helpValueDomainRelationNone helpValueDomainRelation = iota
	helpValueDomainRelationCanonical
	helpValueDomainRelationReplacement
)

type helpLabelsExpectation struct{ expected cli.HelpLabels }

type helpPathProperty uint8

const (
	helpPathInvalid helpPathProperty = iota
	helpUsage
	helpSelectedPath
	helpRequestedPath
)

type helpPathExpectation struct {
	property helpPathProperty
	expected []string
}

type helpCommandGroupProperty uint8

const (
	helpCommandGroupInvalid helpCommandGroupProperty = iota
	helpCommandGroupExists
	helpCommandGroupKind
	helpCommandGroupTitle
	helpCommandGroupOrder
	helpCommandGroupCommandCount
)

type helpCommandGroupExpectation struct {
	locator  HelpPageCommandGroupLocator
	property helpCommandGroupProperty
	exists   bool
	kind     cli.HelpGroupKind
	text     string
	order    cli.HelpGroupOrder
	count    int
}

type helpCommandProperty uint8

const (
	helpCommandInvalid helpCommandProperty = iota
	helpCommandExists
	helpCommandKind
	helpCommandOrigin
	helpCommandName
	helpCommandDescription
	helpCommandHidden
	helpCommandDefault
	helpCommandRunnable
	helpCommandHasSubcommands
	helpCommandCommandEntryPresent
	helpCommandActionEntryPresent
	helpCommandActionEntry
	helpCommandLifecycle
)

type helpCommandExpectation struct {
	locator     HelpPageCommandLocator
	property    helpCommandProperty
	exists      bool
	kind        cli.HelpCommandKind
	origin      cli.DeclarationOrigin
	text        string
	description cli.Description
	boolean     bool
	actionEntry *actionEntryLocatorState
	lifecycle   cli.LifecycleState
}

type helpArgumentFormProperty uint8

const (
	helpArgumentFormInvalid helpArgumentFormProperty = iota
	helpArgumentFormExists
	helpArgumentFormTermCount
)

type helpArgumentFormExpectation struct {
	locator  HelpPageArgumentFormLocator
	property helpArgumentFormProperty
	exists   bool
	count    int
}

type helpArgumentTermProperty uint8

const (
	helpArgumentTermInvalid helpArgumentTermProperty = iota
	helpArgumentTermExists
	helpArgumentTermKind
	helpArgumentTermName
	helpArgumentTermLiteral
	helpArgumentTermType
	helpArgumentTermCount
	helpArgumentTermHidden
	helpArgumentTermSensitive
	helpArgumentTermTokenLanguageFragmentCount
	helpArgumentTermValueDomainEntryCount
)

type helpArgumentTermExpectation struct {
	locator    HelpPageArgumentTermLocator
	property   helpArgumentTermProperty
	exists     bool
	kind       cli.ArgumentTermKind
	text       string
	valueType  cli.ValueType
	countRange cli.CountRange
	boolean    bool
	count      int
	domainKind cli.ValueDomainUseKind
}

type helpFlagGroupProperty uint8

const (
	helpFlagGroupInvalid helpFlagGroupProperty = iota
	helpFlagGroupExists
	helpFlagGroupKind
	helpFlagGroupTitle
	helpFlagGroupOrder
	helpFlagGroupFlagCount
)

type helpFlagGroupExpectation struct {
	locator  HelpPageFlagGroupLocator
	property helpFlagGroupProperty
	exists   bool
	kind     cli.HelpGroupKind
	text     string
	order    cli.HelpGroupOrder
	count    int
}

type helpFlagProperty uint8

const (
	helpFlagInvalid helpFlagProperty = iota
	helpFlagExists
	helpFlagOrigin
	helpFlagOrdinary
	helpFlagAction
	helpFlagActionEntryPresent
	helpFlagActionEntry
	helpFlagSpellingCount
	helpFlagActionSpelling
	helpFlagDescription
	helpFlagOwnerPath
	helpFlagPlacement
	helpFlagHidden
	helpFlagSensitive
	helpFlagLifecycle
	helpFlagValueDomainEntryCount
)

type helpFlagExpectation struct {
	locator     HelpPageFlagLocator
	property    helpFlagProperty
	exists      bool
	origin      cli.DeclarationOrigin
	actionEntry *actionEntryLocatorState
	count       int
	spelling    cli.FlagSpelling
	text        string
	path        []string
	placement   cli.FlagPlacement
	boolean     bool
	lifecycle   cli.LifecycleState
	domainKind  cli.ValueDomainUseKind
}

type helpFlagSpellingProperty uint8

const (
	helpFlagSpellingInvalid helpFlagSpellingProperty = iota
	helpFlagSpellingExists
	helpFlagSpellingValue
	helpFlagSpellingLifecycle
	helpFlagSpellingNotReplaced
)

type helpFlagSpellingExpectation struct {
	locator   HelpPageFlagSpellingLocator
	property  helpFlagSpellingProperty
	exists    bool
	spelling  cli.FlagSpelling
	lifecycle cli.LifecycleState
}

type helpValueDomainEntryProperty uint8

const (
	helpValueDomainEntryInvalid helpValueDomainEntryProperty = iota
	helpValueDomainEntryExists
	helpValueDomainEntryKind
	helpValueDomainEntryName
	helpValueDomainEntryDescription
	helpValueDomainEntryLifecycle
	helpValueDomainEntryNoCanonical
	helpValueDomainEntryNotReplaced
)

type helpValueDomainEntryExpectation struct {
	locator   HelpPageValueDomainEntryLocator
	property  helpValueDomainEntryProperty
	exists    bool
	kind      cli.ValueDomainEntryKind
	text      string
	lifecycle cli.LifecycleState
}

type (
	helpCommandGroupLocatorState = helpGroupLocatorState
	helpFlagGroupLocatorState    = helpGroupLocatorState
)

const (
	helpCommandGroupEntryInvalid helpCommandGroupEntryKind = iota
	helpCommandGroupEntryEntry
	helpCommandGroupEntryProvider
)

type (
	helpCommandGroupResolution = helpGroupResolution[cli.HelpCommandGroupSchema]
	helpFlagGroupResolution    = helpGroupResolution[cli.HelpFlagGroupSchema]
)

func schemaContainsDeclaration(schema cli.Schema, identity cli.DeclarationIdentity) bool {
	switch identity.Kind {
	case cli.DeclarationKindCommandEntry:
		_, ok := schema.CommandEntries[identity.CommandEntry]
		return ok
	case cli.DeclarationKindActionEntry:
		_, ok := schema.ActionEntries[identity.ActionEntry]
		return ok
	case cli.DeclarationKindFlag:
		_, ok := schema.Flags[identity.Flag]
		return ok
	case cli.DeclarationKindFlagSpelling:
		_, ok := schema.FlagSpellings[identity.FlagSpelling]
		return ok
	case cli.DeclarationKindArgumentForm:
		_, ok := schema.ArgumentForms[identity.ArgumentForm]
		return ok
	case cli.DeclarationKindArgumentTerm:
		_, ok := schema.ArgumentTerms[identity.ArgumentTerm]
		return ok
	case cli.DeclarationKindEnvironmentInput:
		_, ok := schema.EnvironmentInputs[identity.EnvironmentInput]
		return ok
	case cli.DeclarationKindValueDomainEntry:
		_, ok := schema.ValueDomainEntries[identity.ValueDomainEntry]
		return ok
	default:
		return false
	}
}

func helpTextSelector(text string) helpIndexSelector {
	selector := helpIndexSelector{
		mode: helpIndexSelectorText,
		text: text,
	}
	return selector
}

func helpIndexSelectorAt(index int) helpIndexSelector {
	selector := helpIndexSelector{
		index: index,
		mode:  helpIndexSelectorIndex,
	}
	return selector
}

//nolint:dupl // Command and flag Help groups intentionally keep direct typed resolution instead of a callback-driven generic selector.
func resolveHelpCommandGroup(locator HelpPageCommandGroupLocator, page cli.HelpPage) (cli.HelpCommandGroup, locatorResolutionStatus, evaluationField) {
	switch locator.selector.mode {
	case helpIndexSelectorIndex:
		if locator.selector.index < 0 || locator.selector.index >= len(page.CommandGroups) {
			return cli.HelpCommandGroup{}, locatorResolutionAbsent, evaluationText("command group is absent")
		}
		return page.CommandGroups[locator.selector.index], locatorResolutionFound, evaluationText("")
	case helpIndexSelectorText:
		matches := make([]cli.HelpCommandGroup, 0, 1)
		for _, group := range page.CommandGroups {
			if group.Title == locator.selector.text {
				matches = append(matches, group)
			}
		}
		if len(matches) == 1 {
			return matches[0], locatorResolutionFound, evaluationText("")
		}
		if len(matches) > 1 {
			return cli.HelpCommandGroup{}, locatorResolutionAmbiguous, evaluationText(fmt.Sprintf("command-group title matches %d groups", len(matches)))
		}
		return cli.HelpCommandGroup{}, locatorResolutionAbsent, evaluationText("command group is absent")
	default:
		return cli.HelpCommandGroup{}, locatorResolutionInvalid, evaluationText("command-group locator is invalid")
	}
}

func resolveHelpCommand(locator HelpPageCommandLocator, page cli.HelpPage) (cli.HelpCommand, locatorResolutionStatus, evaluationField) {
	groups := page.CommandGroups
	if locator.group != nil {
		group, status, detail := resolveHelpCommandGroup(*locator.group, page)
		if status != locatorResolutionFound {
			return cli.HelpCommand{}, status, detail
		}
		groups = []cli.HelpCommandGroup{group}
	}
	switch locator.selector.mode {
	case helpIndexSelectorIndex:
		if len(groups) != 1 {
			return cli.HelpCommand{}, locatorResolutionInvalid, evaluationText("command index requires one command group")
		}
		if locator.selector.index < 0 || locator.selector.index >= len(groups[0].Commands) {
			return cli.HelpCommand{}, locatorResolutionAbsent, evaluationText("command row is absent")
		}
		return groups[0].Commands[locator.selector.index], locatorResolutionFound, evaluationText("")
	case helpIndexSelectorText:
		matches := make([]cli.HelpCommand, 0, 1)
		for _, group := range groups {
			for _, command := range group.Commands {
				if command.Name == locator.selector.text {
					matches = append(matches, command)
				}
			}
		}
		if len(matches) == 1 {
			return matches[0], locatorResolutionFound, evaluationText("")
		}
		if len(matches) > 1 {
			return cli.HelpCommand{}, locatorResolutionAmbiguous, evaluationText(fmt.Sprintf(
				"command name matches %d rows",
				len(matches),
			))
		}
		return cli.HelpCommand{}, locatorResolutionAbsent, evaluationText("command row is absent")
	default:
		return cli.HelpCommand{}, locatorResolutionInvalid, evaluationText("command locator is invalid")
	}
}

func resolveHelpArgumentForm(locator HelpPageArgumentFormLocator, page cli.HelpPage) (cli.HelpArgumentForm, locatorResolutionStatus, evaluationField) {
	if !locator.valid {
		return cli.HelpArgumentForm{}, locatorResolutionInvalid, evaluationText("argument-form locator is invalid")
	}
	if locator.index < 0 || locator.index >= len(page.ArgumentForms) {
		return cli.HelpArgumentForm{}, locatorResolutionAbsent, observed(semanticSubjectArgumentForm, observationAbsent)
	}
	return page.ArgumentForms[locator.index], locatorResolutionFound, evaluationText("")
}

func resolveHelpArgumentTerm(locator HelpPageArgumentTermLocator, page cli.HelpPage) (cli.HelpArgumentTerm, locatorResolutionStatus, evaluationField) {
	form, status, detail := resolveHelpArgumentForm(locator.form, page)
	if status != locatorResolutionFound {
		return cli.HelpArgumentTerm{}, status, detail
	}
	if locator.index < 0 || locator.index >= len(form.Terms) {
		return cli.HelpArgumentTerm{}, locatorResolutionAbsent, observed(semanticSubjectArgumentTerm, observationAbsent)
	}
	return form.Terms[locator.index], locatorResolutionFound, evaluationText("")
}

//nolint:dupl // Command and flag Help groups intentionally keep direct typed resolution instead of a callback-driven generic selector.
func resolveHelpFlagGroup(locator HelpPageFlagGroupLocator, page cli.HelpPage) (cli.HelpFlagGroup, locatorResolutionStatus, evaluationField) {
	switch locator.selector.mode {
	case helpIndexSelectorIndex:
		if locator.selector.index < 0 || locator.selector.index >= len(page.FlagGroups) {
			return cli.HelpFlagGroup{}, locatorResolutionAbsent, evaluationText("flag group is absent")
		}
		return page.FlagGroups[locator.selector.index], locatorResolutionFound, evaluationText("")
	case helpIndexSelectorText:
		matches := make([]cli.HelpFlagGroup, 0, 1)
		for _, group := range page.FlagGroups {
			if group.Title == locator.selector.text {
				matches = append(matches, group)
			}
		}
		if len(matches) == 1 {
			return matches[0], locatorResolutionFound, evaluationText("")
		}
		if len(matches) > 1 {
			return cli.HelpFlagGroup{}, locatorResolutionAmbiguous, evaluationText(fmt.Sprintf("flag-group title matches %d groups", len(matches)))
		}
		return cli.HelpFlagGroup{}, locatorResolutionAbsent, evaluationText("flag group is absent")
	default:
		return cli.HelpFlagGroup{}, locatorResolutionInvalid, evaluationText("flag-group locator is invalid")
	}
}

func flagHasSpelling(flag cli.HelpFlag, text string, page cli.HelpPage) bool {
	for _, id := range flag.FlagSpellings {
		if spelling, ok := page.FlagSpellings[id]; ok && spelling.Spelling.Text == text {
			return true
		}
	}
	return false
}

func resolveHelpFlag(locator HelpPageFlagLocator, page cli.HelpPage) (cli.HelpFlag, locatorResolutionStatus, evaluationField) {
	groups := page.FlagGroups
	if locator.group != nil {
		group, status, detail := resolveHelpFlagGroup(*locator.group, page)
		if status != locatorResolutionFound {
			return cli.HelpFlag{}, status, detail
		}
		groups = []cli.HelpFlagGroup{group}
	}
	switch locator.selector.mode {
	case helpIndexSelectorIndex:
		if len(groups) != 1 {
			return cli.HelpFlag{}, locatorResolutionInvalid, evaluationText("flag index requires one flag group")
		}
		if locator.selector.index < 0 || locator.selector.index >= len(groups[0].Flags) {
			return cli.HelpFlag{}, locatorResolutionAbsent, evaluationText("flag row is absent")
		}
		return groups[0].Flags[locator.selector.index], locatorResolutionFound, evaluationText("")
	case helpIndexSelectorText:
		matches := make([]cli.HelpFlag, 0, 1)
		for _, group := range groups {
			for _, flag := range group.Flags {
				if locator.actionOnly && flag.ActionEntry == 0 {
					continue
				}
				if !locator.actionOnly && flag.ActionEntry != 0 {
					continue
				}
				if flagHasSpelling(flag, locator.selector.text, page) ||
					flag.ActionSpelling.Text == locator.selector.text {
					matches = append(matches, flag)
				}
			}
		}
		if len(matches) == 1 {
			return matches[0], locatorResolutionFound, evaluationText("")
		}
		if len(matches) > 1 {
			return cli.HelpFlag{}, locatorResolutionAmbiguous, evaluationText(fmt.Sprintf(
				"flag spelling matches %d rows",
				len(matches),
			))
		}
		return cli.HelpFlag{}, locatorResolutionAbsent, evaluationText("flag row is absent")
	default:
		return cli.HelpFlag{}, locatorResolutionInvalid, evaluationText("flag locator is invalid")
	}
}

func resolveHelpFlagSpelling(locator HelpPageFlagSpellingLocator, page cli.HelpPage) (cli.HelpFlagSpelling, locatorResolutionStatus, evaluationField) {
	if locator.replacement != nil {
		value, status, detail := resolveHelpFlagSpelling(*locator.replacement, page)
		if status != locatorResolutionFound {
			return cli.HelpFlagSpelling{}, status, detail
		}
		if value.Replacement == 0 {
			return cli.HelpFlagSpelling{}, locatorResolutionAbsent, evaluationText("replacement flag spelling is absent")
		}
		replacement, ok := page.FlagSpellings[value.Replacement]
		if !ok {
			return cli.HelpFlagSpelling{}, locatorResolutionAbsent, evaluationText("replacement flag spelling is absent")
		}
		return replacement, locatorResolutionFound, evaluationText("")
	}
	if locator.byID {
		value, ok := page.FlagSpellings[locator.id]
		if !ok {
			return cli.HelpFlagSpelling{}, locatorResolutionAbsent, observed(semanticSubjectFlagSpelling, observationAbsent)
		}
		return value, locatorResolutionFound, evaluationText("")
	}
	if locator.flag != nil {
		if locator.selector.mode != helpIndexSelectorIndex {
			return cli.HelpFlagSpelling{}, locatorResolutionInvalid, evaluationText("flag spelling locator is invalid")
		}
		flag, status, detail := resolveHelpFlag(*locator.flag, page)
		if status != locatorResolutionFound {
			return cli.HelpFlagSpelling{}, status, detail
		}
		if locator.selector.index < 0 || locator.selector.index >= len(flag.FlagSpellings) {
			return cli.HelpFlagSpelling{}, locatorResolutionAbsent, observed(semanticSubjectFlagSpelling, observationAbsent)
		}
		value, ok := page.FlagSpellings[flag.FlagSpellings[locator.selector.index]]
		if !ok {
			return cli.HelpFlagSpelling{}, locatorResolutionAbsent, observed(semanticSubjectFlagSpelling, observationAbsent)
		}
		return value, locatorResolutionFound, evaluationText("")
	}
	if locator.selector.mode != helpIndexSelectorText {
		return cli.HelpFlagSpelling{}, locatorResolutionInvalid, evaluationText("flag spelling locator is invalid")
	}
	matches := make([]cli.HelpFlagSpelling, 0, 1)
	for _, value := range page.FlagSpellings {
		if value.Spelling.Text == locator.selector.text {
			matches = append(matches, value)
		}
	}
	if len(matches) == 1 {
		return matches[0], locatorResolutionFound, evaluationText("")
	}
	if len(matches) > 1 {
		return cli.HelpFlagSpelling{}, locatorResolutionAmbiguous, evaluationText(fmt.Sprintf(
			"flag spelling matches %d subjects",
			len(matches),
		))
	}
	return cli.HelpFlagSpelling{}, locatorResolutionAbsent, observed(semanticSubjectFlagSpelling, observationAbsent)
}

func valueDomainUseEntries(uses []cli.HelpValueDomainUse, kind cli.ValueDomainUseKind) []cli.ValueDomainEntryID {
	var entries []cli.ValueDomainEntryID
	for _, use := range uses {
		if use.Kind == kind {
			entries = append(entries, use.Entries...)
		}
	}
	return entries
}

func resolveHelpValueDomainUseEntry(locator HelpPageValueDomainEntryLocator, page cli.HelpPage) (cli.HelpValueDomainEntry, locatorResolutionStatus, evaluationField) {
	var ids []cli.ValueDomainEntryID
	switch {
	case locator.term != nil:
		term, status, detail := resolveHelpArgumentTerm(*locator.term, page)
		if status != locatorResolutionFound {
			return cli.HelpValueDomainEntry{}, status, detail
		}
		ids = valueDomainUseEntries(term.ValueDomainUses, locator.kind)
	case locator.flag != nil:
		flag, status, detail := resolveHelpFlag(*locator.flag, page)
		if status != locatorResolutionFound {
			return cli.HelpValueDomainEntry{}, status, detail
		}
		ids = valueDomainUseEntries(flag.ValueDomainUses, locator.kind)
	default:
		return cli.HelpValueDomainEntry{}, locatorResolutionInvalid, evaluationText("value-domain use locator is invalid")
	}
	if locator.index < 0 || locator.index >= len(ids) {
		return cli.HelpValueDomainEntry{}, locatorResolutionAbsent, observed(semanticSubjectValueDomainEntry, observationAbsent)
	}
	value, ok := page.ValueDomainEntries[ids[locator.index]]
	if !ok {
		return cli.HelpValueDomainEntry{}, locatorResolutionAbsent, observed(semanticSubjectValueDomainEntry, observationAbsent)
	}
	return value, locatorResolutionFound, evaluationText("")
}

func resolveHelpValueDomainEntry(locator HelpPageValueDomainEntryLocator, page cli.HelpPage) (cli.HelpValueDomainEntry, locatorResolutionStatus, evaluationField) {
	if locator.parent != nil {
		value, status, detail := resolveHelpValueDomainEntry(*locator.parent, page)
		if status != locatorResolutionFound {
			return cli.HelpValueDomainEntry{}, status, detail
		}
		var id cli.ValueDomainEntryID
		switch locator.relation {
		case helpValueDomainRelationCanonical:
			id = value.Canonical
		case helpValueDomainRelationReplacement:
			id = value.Replacement
		default:
			return cli.HelpValueDomainEntry{}, locatorResolutionInvalid, evaluationText("value-domain relation is invalid")
		}
		if id == 0 {
			return cli.HelpValueDomainEntry{}, locatorResolutionAbsent, evaluationText("related value-domain entry is absent")
		}
		related, ok := page.ValueDomainEntries[id]
		if !ok {
			return cli.HelpValueDomainEntry{}, locatorResolutionAbsent, evaluationText("related value-domain entry is absent")
		}
		return related, locatorResolutionFound, evaluationText("")
	}
	if locator.byID {
		value, ok := page.ValueDomainEntries[locator.id]
		if !ok {
			return cli.HelpValueDomainEntry{}, locatorResolutionAbsent, observed(semanticSubjectValueDomainEntry, observationAbsent)
		}
		return value, locatorResolutionFound, evaluationText("")
	}
	if locator.byUse {
		return resolveHelpValueDomainUseEntry(locator, page)
	}
	if locator.selector.mode != helpIndexSelectorText {
		return cli.HelpValueDomainEntry{}, locatorResolutionInvalid, evaluationText("value-domain-entry locator is invalid")
	}
	matches := make([]cli.HelpValueDomainEntry, 0, 1)
	for _, value := range page.ValueDomainEntries {
		if value.Name == locator.selector.text {
			matches = append(matches, value)
		}
	}
	if len(matches) == 1 {
		return matches[0], locatorResolutionFound, evaluationText("")
	}
	if len(matches) > 1 {
		return cli.HelpValueDomainEntry{}, locatorResolutionAmbiguous, evaluationText(fmt.Sprintf(
			"value-domain name matches %d subjects",
			len(matches),
		))
	}
	return cli.HelpValueDomainEntry{}, locatorResolutionAbsent, observed(semanticSubjectValueDomainEntry, observationAbsent)
}

func equalHelpValueLiteralPointer(left, right *cli.ValueLiteral) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func equalHelpFlagSpelling(got, expected cli.FlagSpelling) bool {
	return got.Kind == expected.Kind &&
		got.Text == expected.Text &&
		got.ValueMode == expected.ValueMode &&
		slices.Equal(got.ValueForms, expected.ValueForms) &&
		got.Sensitive == expected.Sensitive &&
		equalHelpValueLiteralPointer(got.FixedValue, expected.FixedValue) &&
		equalHelpValueLiteralPointer(got.ImplicitValue, expected.ImplicitValue)
}

func evaluateHelpExistence(subject evaluationField, expected bool, status locatorResolutionStatus) evaluationResult {
	present := status == locatorResolutionFound
	if status == locatorResolutionAmbiguous || status == locatorResolutionInvalid {
		return disprovedResult(subject, property(evaluationText("exists"), evaluationBool(expected)), evaluationText("locator is ambiguous or invalid"))
	}
	if present == expected {
		return provedResult()
	}
	return disprovedResult(subject, property(evaluationText("exists"), evaluationBool(expected)), property(evaluationText("exists"), evaluationBool(present)))
}

func helpResolutionFailure(subject, expected, detail evaluationField) evaluationResult {
	return disprovedResult(subject, expected, detail)
}

func helpPropertyMismatch(subject evaluationField, name string, expected, actual evaluationField) evaluationResult {
	return disprovedResult(subject, property(evaluationText(name), expected), property(evaluationText(name), actual))
}

func (helpLabelsField) evaluationField() {}

func (helpLabelsExpectation) helpPresentationExpectation() {}

func (expectation helpLabelsExpectation) evaluateHelp(page cli.HelpPage) evaluationResult {
	if page.Labels == expectation.expected {
		return provedResult()
	}
	field := helpLabelsField{
		value: expectation.expected,
	}
	observedField := helpLabelsField{
		value: page.Labels,
	}
	return disprovedResult(evaluationText("help labels"), property(evaluationText("labels"), field), property(evaluationText("labels"), observedField))
}

func (helpPathExpectation) helpPresentationExpectation() {}

func (expectation helpPathExpectation) evaluateHelp(page cli.HelpPage) evaluationResult {
	var subject evaluationField
	var name string
	var actual []string
	switch expectation.property {
	case helpUsage:
		subject = evaluationText("help usage")
		name = "usage"
		actual = page.Usage
	case helpSelectedPath:
		subject = evaluationText("help selected path")
		name = "selected path"
		actual = page.SelectedPath
	case helpRequestedPath:
		subject = evaluationText("help requested path")
		name = "requested path"
		actual = page.RequestedPath
	default:
		panic("clitest: invalid Help path expectation")
	}
	if slices.Equal(actual, expectation.expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(subject, name, expectation.expected, actual)
}

func (helpCommandExpectation) helpPresentationExpectation() {}

func (expectation helpCommandExpectation) evaluateHelp(page cli.HelpPage) evaluationResult {
	command, status, detail := resolveHelpCommand(expectation.locator, page)
	subject := evaluationText("help command")
	if expectation.property == helpCommandExists {
		return evaluateHelpExistence(subject, expectation.exists, status)
	}
	if expectation.property == helpCommandActionEntry && (expectation.actionEntry == nil || expectation.actionEntry.mode != actionEntryLocatorID) {
		return disprovedResult(subject, evaluationText("exact bound action-entry identity"), evaluationText("expected locator is not exact-identity-bound"))
	}
	if status != locatorResolutionFound {
		return helpResolutionFailure(subject, expectation.commandExpected(), detail)
	}
	switch expectation.property {
	case helpCommandKind:
		if command.Kind == expectation.kind {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "kind", evaluationText(fmt.Sprint(expectation.kind)), evaluationText(fmt.Sprint(command.Kind)))
	case helpCommandOrigin:
		if command.Origin == expectation.origin {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "origin", evaluationText(fmt.Sprint(expectation.origin)), evaluationText(fmt.Sprint(command.Origin)))
	case helpCommandName:
		if command.Name == expectation.text {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "name", evaluationText(expectation.text), evaluationText(command.Name))
	case helpCommandDescription:
		if command.Description == expectation.description {
			return provedResult()
		}
		field := descriptionField{
			value: expectation.description,
		}
		observedField := descriptionField{
			value: command.Description,
		}
		return helpPropertyMismatch(subject, "description", field, observedField)
	case helpCommandHidden:
		if command.Hidden == expectation.boolean {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "hidden", evaluationBool(expectation.boolean), evaluationBool(command.Hidden))
	case helpCommandDefault:
		if command.Default == expectation.boolean {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "default", evaluationBool(expectation.boolean), evaluationBool(command.Default))
	case helpCommandRunnable:
		if command.Runnable == expectation.boolean {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "runnable", evaluationBool(expectation.boolean), evaluationBool(command.Runnable))
	case helpCommandHasSubcommands:
		if command.HasSubcommands == expectation.boolean {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "has-subcommands", evaluationBool(expectation.boolean), evaluationBool(command.HasSubcommands))
	case helpCommandCommandEntryPresent:
		if command.CommandEntry != 0 {
			return provedResult()
		}
		return disprovedResult(subject, evaluationText("command-entry identity present"), evaluationText("command-entry identity is zero"))
	case helpCommandActionEntryPresent:
		if command.ActionEntry != 0 {
			return provedResult()
		}
		return disprovedResult(subject, evaluationText("action-entry identity present"), evaluationText("action-entry identity is zero"))
	case helpCommandActionEntry:
		if command.ActionEntry == expectation.actionEntry.id {
			return provedResult()
		}
		return disprovedResult(subject, evaluationText("action-entry matches expected identity"), evaluationText("action-entry has different identity"))
	case helpCommandLifecycle:
		if command.LifecycleState == expectation.lifecycle {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "lifecycle", evaluationText(fmt.Sprint(expectation.lifecycle)), evaluationText(fmt.Sprint(command.LifecycleState)))
	default:
		panic("clitest: invalid Help command expectation")
	}
}

func (expectation helpCommandExpectation) commandExpected() evaluationField {
	switch expectation.property {
	case helpCommandKind:
		return property(evaluationText("kind"), evaluationText(fmt.Sprint(expectation.kind)))
	case helpCommandOrigin:
		return property(evaluationText("origin"), evaluationText(fmt.Sprint(expectation.origin)))
	case helpCommandName:
		return property(evaluationText("name"), evaluationText(expectation.text))
	case helpCommandDescription:
		field := descriptionField{
			value: expectation.description,
		}
		return property(evaluationText("description"), field)
	case helpCommandHidden:
		return property(evaluationText("hidden"), evaluationText(strconv.FormatBool(expectation.boolean)))
	case helpCommandDefault:
		return property(evaluationText("default"), evaluationText(strconv.FormatBool(expectation.boolean)))
	case helpCommandRunnable:
		return property(evaluationText("runnable"), evaluationText(strconv.FormatBool(expectation.boolean)))
	case helpCommandHasSubcommands:
		return property(evaluationText("has-subcommands"), evaluationText(strconv.FormatBool(expectation.boolean)))
	case helpCommandCommandEntryPresent:
		return evaluationText("command-entry identity present")
	case helpCommandActionEntryPresent:
		return evaluationText("action-entry identity present")
	case helpCommandActionEntry:
		return evaluationText("action-entry matches expected identity")
	case helpCommandLifecycle:
		return property(evaluationText("lifecycle"), evaluationText(fmt.Sprint(expectation.lifecycle)))
	default:
		return evaluationText("valid command expectation")
	}
}

func (helpDescriptionExpectation) helpPresentationExpectation() {}

func (e helpDescriptionExpectation) evaluateHelp(page cli.HelpPage) evaluationResult {
	return compareDescription(evaluationText("help description"), e.expected, page.Description)
}

func (helpExamplesExpectation) helpPresentationExpectation() {}

func (e helpExamplesExpectation) evaluateHelp(page cli.HelpPage) evaluationResult {
	if slices.Equal(page.Examples, e.expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(evaluationText("help examples"), semanticNameExamples, e.expected, page.Examples)
}
