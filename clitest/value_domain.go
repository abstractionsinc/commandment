package clitest

import (
	"fmt"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
)

// ValueDomainLocator identifies one value-domain use associated with a value for
// clitest semantic expectation. A value domain has no independent public domain ID;
// the locator follows the public value/domain relationship. Its zero value is
// invalid.
type ValueDomainLocator struct {
	_     [0]func()
	state *valueDomainLocatorState
}

// ValueDomainEntryLocator identifies one exact value-domain entry declaration
// for clitest semantic expectation. Its zero value is invalid.
type ValueDomainEntryLocator struct {
	_     [0]func()
	state *valueDomainEntryLocatorState
}

// InspectionObservedValueDomainsExactly returns an Inspection expectation that Inspection.ValueDomains contains exactly
// the supplied domains in the same public slice order.
// The expectation checks the returned observation order rather than inventing a separate domain identity namespace.
func InspectionObservedValueDomainsExactly(domains ...ValueDomainLocator) InspectionExpectation {
	states := make([]*valueDomainLocatorState, len(domains))
	for index, domain := range domains {
		states[index] = domain.state
	}
	expectation := observedValueDomainsExpectation{
		domains: states,
	}
	return expectation
}

// InspectionObservedValueDomainEntriesExactly returns an Inspection expectation that the returned ValueDomainEntries
// map contains exactly the supplied distinct semantic entries.
// The map has no semantic iteration order; entry identity and relationships are resolved through the public Inspection
// observation.
func InspectionObservedValueDomainEntriesExactly(entries ...ValueDomainEntryLocator) InspectionExpectation {
	states := make([]*valueDomainEntryLocatorState, len(entries))
	for index, entry := range entries {
		states[index] = entry.state
	}
	expectation := observedValueDomainEntriesExpectation{
		entries: states,
	}
	return expectation
}

// ValueDomainEntryByID returns a locator for the exact public ValueDomainEntryID in the
// observation being evaluated.
func ValueDomainEntryByID(id cli.ValueDomainEntryID) ValueDomainEntryLocator {
	return valueDomainEntryIDLocator(id)
}

// Origin returns an expectation that checks the located value-domain entry has the
// framework-established DeclarationOrigin expected.
func (locator ValueDomainEntryLocator) Origin(expected cli.DeclarationOrigin) DeclarationExpectation {
	expectation := valueDomainEntryOriginExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// ReleaseLifecycle returns an expectation that checks the located value-domain entry has the exact
// public release-lifecycle metadata expected.
func (locator ValueDomainEntryLocator) ReleaseLifecycle(expected cli.ReleaseLifecycle) DeclarationExpectation {
	expectation := valueDomainEntryReleaseLifecycleExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// NotReplaced returns an expectation that checks the located value-domain entry has no public
// declaration-replacement relationship. Incomplete observation cannot manufacture replacement
// absence.
func (locator ValueDomainEntryLocator) NotReplaced() DeclarationExpectation {
	expectation := valueDomainEntryReplacementAbsenceExpectation{
		locator: locator.state,
	}
	return expectation
}

// ReplacementPresentation returns an expectation that checks the located value-domain entry
// replacement carries the expected public presentation text.
func (locator ValueDomainEntryLocator) ReplacementPresentation(expected string) DeclarationExpectation {
	expectation := valueDomainEntryReplacementPresentationExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// ReplacementTargetDeprecatedDuringMigration returns an expectation that checks the located
// value-domain entry replacement carries the expected migration-deprecation marker.
func (locator ValueDomainEntryLocator) ReplacementTargetDeprecatedDuringMigration(expected bool) DeclarationExpectation {
	expectation := valueDomainEntryReplacementTargetDeprecatedExpectation{
		locator:  locator.state,
		expected: expected,
	}
	return expectation
}

// Exists returns an expectation that checks the located value-domain entry exists in the
// relevant public declaration observation.
func (locator ValueDomainEntryLocator) Exists() DeclarationExpectation {
	value := valueDomainEntryExistenceExpectationValue{
		locator:  locator.state,
		expected: true,
	}
	return value
}

// DoesNotExist returns an expectation that checks the located value-domain entry is absent
// from a complete relevant public declaration domain; incomplete observation cannot
// manufacture absence.
func (locator ValueDomainEntryLocator) DoesNotExist() DeclarationExpectation {
	value := valueDomainEntryExistenceExpectationValue{
		locator:  locator.state,
		expected: false,
	}
	return value
}

// Kind returns an expectation that checks the value-domain entry has the requested
// ValueDomainEntryKind.
func (locator ValueDomainEntryLocator) Kind(v cli.ValueDomainEntryKind) DeclarationExpectation {
	value := valueDomainEntryKindExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Hidden returns an expectation that checks the value-domain entry is declared hidden.
func (locator ValueDomainEntryLocator) Hidden() DeclarationExpectation {
	value := valueDomainEntryHiddenExpectationValue{
		locator: locator.state,
	}
	return value
}

// Lifecycle returns an expectation that checks the value-domain entry has the requested
// declared lifecycle state.
func (locator ValueDomainEntryLocator) Lifecycle(v cli.LifecycleState) DeclarationExpectation {
	value := valueDomainEntryLifecycleExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// ReplacedBy returns an expectation that checks the value-domain entry's replacement targets
// the exact heterogeneous declaration identity identified by the supplied locator.
func (locator ValueDomainEntryLocator) ReplacedBy(v DeclarationIdentityLocator) DeclarationExpectation {
	value := valueDomainEntryReplacementExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Available returns an expectation that checks the located value-domain entry is available in
// the effective public declaration observation (Reference or Inspection).
func (locator ValueDomainEntryLocator) Available() EffectiveDeclarationExpectation {
	value := valueDomainEntryAvailabilityExpectationValue{
		locator:  locator.state,
		expected: true,
	}
	return value
}

// Unavailable returns an expectation that checks the located value-domain entry is unavailable
// in the effective public declaration observation (Reference or Inspection).
func (locator ValueDomainEntryLocator) Unavailable() EffectiveDeclarationExpectation {
	value := valueDomainEntryAvailabilityExpectationValue{
		locator:  locator.state,
		expected: false,
	}
	return value
}

// Visible returns an expectation that checks the located value-domain entry is visible in the
// effective public declaration observation (Reference or Inspection).
func (locator ValueDomainEntryLocator) Visible() EffectiveDeclarationExpectation {
	value := valueDomainEntryVisibilityExpectationValue{
		locator:  locator.state,
		expected: true,
	}
	return value
}

// NotVisible returns an expectation that checks the located value-domain entry is not visible
// in the effective public declaration observation (Reference or Inspection).
func (locator ValueDomainEntryLocator) NotVisible() EffectiveDeclarationExpectation {
	value := valueDomainEntryVisibilityExpectationValue{
		locator:  locator.state,
		expected: false,
	}
	return value
}

// Canonical returns an expectation that checks the value-domain entry's canonical-entry
// relationship points to the requested exact entry identity.
func (locator ValueDomainEntryLocator) Canonical(v ValueDomainEntryLocator) DeclarationExpectation {
	value := valueDomainEntryCanonicalExpectationValue{
		locator: locator.state,
		target:  v.state,
	}
	return value
}

// SourceDescription returns a reusable Schema expectation that checks the public source
// description exposed by the located value-domain entry.
func (locator ValueDomainEntryLocator) SourceDescription(expected cli.Description) SchemaExpectation {
	value := valueDomainEntrySourceDescriptionExpectationValue{
		locator:  locator.state,
		expected: expected,
	}
	return value
}

// PresentedDescription returns a reusable declaration expectation that checks the public
// presented description exposed by the located value-domain entry.
func (locator ValueDomainEntryLocator) PresentedDescription(expected cli.Description) PresentedDeclarationExpectation {
	value := valueDomainEntryPresentedDescriptionExpectationValue{
		locator:  locator.state,
		expected: expected,
	}
	return value
}

func (ValueDomainEntryLocator) declarationIdentityLocator() {}

func (ValueDomainEntryLocator) diagnosticSubject() {}

func (ValueDomainEntryLocator) noticeSubject() {}

func (ValueDomainEntryLocator) tokenOwner() {}

// Entry returns a locator for the uniquely named public entry in the located value domain.
func (l ValueDomainLocator) Entry(name string) ValueDomainEntryLocator {
	return valueDomainEntryNameLocator(l, name)
}

// EntryAt returns the domain-entry locator at index in authoritative public domain-entry
// order.
func (l ValueDomainLocator) EntryAt(index int) ValueDomainEntryLocator {
	return valueDomainEntryAtLocator(l, index)
}

// Exists returns an expectation that checks the located value domain exists in the relevant
// public declaration observation.
func (l ValueDomainLocator) Exists() DeclarationExpectation {
	value := valueDomainExistenceExpectationValue{
		locator:  l.state,
		expected: true,
	}
	return value
}

// DoesNotExist returns an expectation that checks the located value domain is absent from a
// complete relevant public declaration domain; incomplete observation cannot manufacture
// absence.
func (l ValueDomainLocator) DoesNotExist() DeclarationExpectation {
	value := valueDomainExistenceExpectationValue{
		locator:  l.state,
		expected: false,
	}
	return value
}

// Outcome returns an expectation that checks the inspected value domain has the requested
// ValueDomainOutcome.
func (l ValueDomainLocator) Outcome(outcome cli.ValueDomainOutcome) InspectionExpectation {
	return valueDomainOutcomeExpectation(l, outcome)
}

// InvocationScoped returns an expectation that checks the inspected value domain is
// invocation-scoped.
func (l ValueDomainLocator) InvocationScoped() InspectionExpectation {
	return valueDomainInvocationScopedExpectation(l)
}

// DependenciesExactly returns an expectation that checks the inspected value domain depends on
// exactly the requested values in authoritative order.
func (l ValueDomainLocator) DependenciesExactly(values ...ValueLocator) InspectionExpectation {
	return valueDomainDependenciesExactlyExpectation(l, values)
}

// EntriesExactly returns an expectation that checks the value domain declares exactly the
// requested entries in authoritative public order.
func (l ValueDomainLocator) EntriesExactly(values ...ValueDomainEntryLocator) DeclarationExpectation {
	states := make([]*valueDomainEntryLocatorState, len(values))
	for i, value := range values {
		states[i] = value.state
	}
	expectation := valueDomainEntriesExpectationValue{
		locator: l.state,
		entries: states,
	}
	return expectation
}

// UsesExactly returns a declaration expectation that checks the value domain's complete uses
// relationship by public domain identity and authoritative order.
func (l ValueDomainLocator) UsesExactly(uses ...ValueDomainLocator) DeclarationExpectation {
	states := make([]*valueDomainLocatorState, len(uses))
	for index, use := range uses {
		states[index] = use.state
	}
	value := valueDomainUsesExpectationValue{
		locator: l.state,
		uses:    states,
	}
	return value
}

type valueDomainEntryAvailabilityExpectationValue struct {
	locator  *valueDomainEntryLocatorState
	expected bool
}

type valueDomainEntryVisibilityExpectationValue struct {
	locator  *valueDomainEntryLocatorState
	expected bool
}

type valueDomainEntryOriginExpectation struct {
	locator  *valueDomainEntryLocatorState
	expected cli.DeclarationOrigin
}

type valueDomainEntryReleaseLifecycleExpectation struct {
	locator  *valueDomainEntryLocatorState
	expected cli.ReleaseLifecycle
}

type valueDomainEntryReplacementAbsenceExpectation struct{ locator *valueDomainEntryLocatorState }

type valueDomainEntryReplacementPresentationExpectation struct {
	locator  *valueDomainEntryLocatorState
	expected string
}

type valueDomainEntryReplacementTargetDeprecatedExpectation struct {
	locator  *valueDomainEntryLocatorState
	expected bool
}

type valueDomainEntryExistenceExpectationValue struct {
	locator  *valueDomainEntryLocatorState
	expected bool
}

type observedValueDomainsExpectation struct {
	domains []*valueDomainLocatorState
}

type observedValueDomainEntriesExpectation struct {
	entries []*valueDomainEntryLocatorState
}

type valueDomainExistenceExpectationValue struct {
	locator  *valueDomainLocatorState
	expected bool
}

type valueDomainEntriesExpectationValue struct {
	locator *valueDomainLocatorState
	entries []*valueDomainEntryLocatorState
}

type valueDomainEntryKindExpectationValue struct {
	locator  *valueDomainEntryLocatorState
	expected cli.ValueDomainEntryKind
}

type valueDomainEntryHiddenExpectationValue struct{ locator *valueDomainEntryLocatorState }

type valueDomainEntryLifecycleExpectationValue struct {
	locator  *valueDomainEntryLocatorState
	expected cli.LifecycleState
}

type valueDomainEntryReplacementExpectationValue struct {
	locator  *valueDomainEntryLocatorState
	expected DeclarationIdentityLocator
}

type valueDomainEntryCanonicalExpectationValue struct {
	locator *valueDomainEntryLocatorState
	target  *valueDomainEntryLocatorState
}

type valueDomainEntrySourceDescriptionExpectationValue struct {
	locator  *valueDomainEntryLocatorState
	expected cli.Description
}

type valueDomainEntryPresentedDescriptionExpectationValue struct {
	locator  *valueDomainEntryLocatorState
	expected cli.Description
}

type valueDomainUsesExpectationValue struct {
	locator *valueDomainLocatorState
	uses    []*valueDomainLocatorState
}

type valueDomainLocatorState struct {
	valid bool
	value *valueLocatorState
	use   cli.ValueDomainUseKind
}

type valueDomainEntryLocatorMode uint8

const (
	valueDomainEntryLocatorInvalid valueDomainEntryLocatorMode = iota
	valueDomainEntryLocatorByID
	valueDomainEntryLocatorName
	valueDomainEntryLocatorIndex
)

type valueDomainEntryLocatorState struct {
	mode valueDomainEntryLocatorMode
	id   cli.ValueDomainEntryID

	domain *valueDomainLocatorState
	name   string
	index  int
}

type valueDomainPropertyKind uint8

const (
	valueDomainPropertyInvalid valueDomainPropertyKind = iota
	valueDomainPropertyOutcome
	valueDomainPropertyInvocationScoped
	valueDomainPropertyDependencies
)

func valueDomainLocatorLabel(state *valueDomainLocatorState) evaluationField {
	if state == nil {
		return semanticSubjectValueDomain
	}
	return qualifySubject(valueLocatorLabel(state.value), evaluationText(fmt.Sprintf("domain use %v", state.use)))
}

func resolveSchemaValueDomainLocator(state *valueDomainLocatorState, schema cli.Schema) locatorResolution[cli.ValueDomainSchema] {
	var resolution locatorResolution[cli.ValueDomainSchema]
	label := valueDomainLocatorLabel(state)
	if state == nil || !state.valid {
		resolution = locatorResolution[cli.ValueDomainSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectValueDomain, observationInvalidLocator),
		}
		return resolution
	}
	valueID, status, detail := resolveSchemaValueID(state.value, &schema)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.ValueDomainSchema]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	matches := make([]int, 0, 1)
	for index, domain := range schema.ValueDomains {
		for _, use := range domain.Uses {
			if use.Destination == valueID && use.Kind == state.use {
				matches = append(matches, index)
				break
			}
		}
	}
	switch len(matches) {
	case 1:
		resolution = locatorResolution[cli.ValueDomainSchema]{
			status:  locatorResolutionFound,
			value:   schema.ValueDomains[matches[0]],
			subject: label,
		}
		return resolution
	case 0:
		resolution = locatorResolution[cli.ValueDomainSchema]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectValueDomain, observationAbsent),
		}
		return resolution
	default:
		resolution = locatorResolution[cli.ValueDomainSchema]{
			status:  locatorResolutionAmbiguous,
			subject: label,
			detail: collectionCardinalityField{
				name:  semanticNameMatchesValueDomainIndexes,
				count: len(matches),
			},
		}
		return resolution
	}
}

func resolveReferenceValueDomainLocator(state *valueDomainLocatorState, reference cli.Reference) locatorResolution[cli.ValueDomainReference] {
	var resolution locatorResolution[cli.ValueDomainReference]
	label := valueDomainLocatorLabel(state)
	if state == nil || !state.valid {
		resolution = locatorResolution[cli.ValueDomainReference]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectValueDomain, observationInvalidLocator),
		}
		return resolution
	}
	valueID, status, detail := resolveReferenceValueID(state.value, reference)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.ValueDomainReference]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	matches := make([]int, 0, 1)
	for index, domain := range reference.ValueDomains {
		for _, use := range domain.Uses {
			if use.Destination == valueID && use.Kind == state.use {
				matches = append(matches, index)
				break
			}
		}
	}
	switch len(matches) {
	case 1:
		resolution = locatorResolution[cli.ValueDomainReference]{
			status:  locatorResolutionFound,
			value:   reference.ValueDomains[matches[0]],
			subject: label,
		}
		return resolution
	case 0:
		resolution = locatorResolution[cli.ValueDomainReference]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectValueDomain, observationAbsent),
		}
		return resolution
	default:
		resolution = locatorResolution[cli.ValueDomainReference]{
			status:  locatorResolutionAmbiguous,
			subject: label,
			detail: collectionCardinalityField{
				name:  semanticNameMatchesValueDomainIndexes,
				count: len(matches),
			},
		}
		return resolution
	}
}

func resolveInspectionValueDomainLocator(state *valueDomainLocatorState, inspection cli.Inspection) locatorResolution[cli.ValueDomainInspection] {
	var resolution locatorResolution[cli.ValueDomainInspection]
	label := valueDomainLocatorLabel(state)
	if state == nil || !state.valid {
		resolution = locatorResolution[cli.ValueDomainInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectValueDomain, observationInvalidLocator),
		}
		return resolution
	}
	valueID, status, detail := resolveInspectionValueID(state.value, inspection)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.ValueDomainInspection]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	matches := make([]int, 0, 1)
	for index, domain := range inspection.ValueDomains {
		for _, use := range domain.Uses {
			if use.Destination == valueID && use.Kind == state.use {
				matches = append(matches, index)
				break
			}
		}
	}
	switch len(matches) {
	case 1:
		resolution = locatorResolution[cli.ValueDomainInspection]{
			status:  locatorResolutionFound,
			value:   inspection.ValueDomains[matches[0]],
			subject: label,
		}
		return resolution
	case 0:
		if inspection.Truncated {
			resolution = locatorResolution[cli.ValueDomainInspection]{
				status:  locatorResolutionNotProvable,
				subject: label,
				detail:  observed(semanticSubjectInspection, observationTruncated),
			}
			return resolution
		}
		resolution = locatorResolution[cli.ValueDomainInspection]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectValueDomain, observationAbsent),
		}
		return resolution
	default:
		resolution = locatorResolution[cli.ValueDomainInspection]{
			status:  locatorResolutionAmbiguous,
			subject: label,
			detail: collectionCardinalityField{
				name:  semanticNameMatchesValueDomainIndexes,
				count: len(matches),
			},
		}
		return resolution
	}
}

func valueDomainEntryLocatorLabel(state *valueDomainEntryLocatorState) evaluationField {
	if state == nil {
		return semanticSubjectValueDomainEntry
	}
	switch state.mode {
	case valueDomainEntryLocatorByID:
		return evaluationText(fmt.Sprintf("value-domain entry id %d", state.id))
	case valueDomainEntryLocatorName:
		return qualifySubject(valueDomainLocatorLabel(state.domain), subjectNamed(evaluationText("entry"), state.name))
	case valueDomainEntryLocatorIndex:
		return qualifySubject(valueDomainLocatorLabel(state.domain), evaluationText(fmt.Sprintf("entry %d", state.index)))
	default:
		return semanticSubjectValueDomainEntry
	}
}

func resolveSchemaValueDomainEntryLocator(state *valueDomainEntryLocatorState, schema *cli.Schema) locatorResolution[cli.ValueDomainEntrySchema] {
	var resolution locatorResolution[cli.ValueDomainEntrySchema]
	label := valueDomainEntryLocatorLabel(state)
	if schema == nil {
		resolution = locatorResolution[cli.ValueDomainEntrySchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectValueDomainEntry, observationInvalidLocator),
		}
		return resolution
	}
	if state == nil || state.mode == valueDomainEntryLocatorInvalid {
		resolution = locatorResolution[cli.ValueDomainEntrySchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectValueDomainEntry, observationInvalidLocator),
		}
		return resolution
	}
	id := state.id
	if state.mode != valueDomainEntryLocatorByID {
		domain := resolveSchemaValueDomainLocator(state.domain, *schema)
		if domain.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ValueDomainEntrySchema]{
				status:  domain.status,
				subject: label,
				detail:  domain.detail,
			}
			return resolution
		}
		var status locatorResolutionStatus
		var detail evaluationField
		id, status, detail = selectSchemaValueDomainEntry(state, domain.value.Entries, schema, true)
		if status != locatorResolutionFound {
			resolution = locatorResolution[cli.ValueDomainEntrySchema]{
				status:  status,
				subject: label,
				detail:  detail,
			}
			return resolution
		}
	}
	value, ok := schema.ValueDomainEntries[id]
	if !ok {
		resolution = locatorResolution[cli.ValueDomainEntrySchema]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectValueDomainEntry, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ValueDomainEntrySchema]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

func resolveReferenceValueDomainEntryLocator(state *valueDomainEntryLocatorState, reference cli.Reference) locatorResolution[cli.ValueDomainEntryReference] {
	var resolution locatorResolution[cli.ValueDomainEntryReference]
	label := valueDomainEntryLocatorLabel(state)
	if state == nil || state.mode == valueDomainEntryLocatorInvalid {
		resolution = locatorResolution[cli.ValueDomainEntryReference]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectValueDomainEntry, observationInvalidLocator),
		}
		return resolution
	}
	id := state.id
	if state.mode != valueDomainEntryLocatorByID {
		domain := resolveReferenceValueDomainLocator(state.domain, reference)
		if domain.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ValueDomainEntryReference]{
				status:  domain.status,
				subject: label,
				detail:  domain.detail,
			}
			return resolution
		}
		var status locatorResolutionStatus
		var detail evaluationField
		id, status, detail = selectReferenceValueDomainEntry(state, domain.value.Entries, reference, true)
		if status != locatorResolutionFound {
			resolution = locatorResolution[cli.ValueDomainEntryReference]{
				status:  status,
				subject: label,
				detail:  detail,
			}
			return resolution
		}
	}
	value, ok := reference.ValueDomainEntries[id]
	if !ok {
		resolution = locatorResolution[cli.ValueDomainEntryReference]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectValueDomainEntry, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ValueDomainEntryReference]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

func resolveInspectionValueDomainEntryLocator(state *valueDomainEntryLocatorState, inspection cli.Inspection) locatorResolution[cli.ValueDomainEntryInspection] {
	var resolution locatorResolution[cli.ValueDomainEntryInspection]
	label := valueDomainEntryLocatorLabel(state)
	if state == nil || state.mode == valueDomainEntryLocatorInvalid {
		resolution = locatorResolution[cli.ValueDomainEntryInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectValueDomainEntry, observationInvalidLocator),
		}
		return resolution
	}
	id := state.id
	if state.mode != valueDomainEntryLocatorByID {
		domain := resolveInspectionValueDomainLocator(state.domain, inspection)
		if domain.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ValueDomainEntryInspection]{
				status:  domain.status,
				subject: label,
				detail:  domain.detail,
			}
			return resolution
		}
		var status locatorResolutionStatus
		var detail evaluationField
		id, status, detail = selectInspectionValueDomainEntry(state, domain.value.Entries, inspection, !inspection.Truncated)
		if status != locatorResolutionFound {
			resolution = locatorResolution[cli.ValueDomainEntryInspection]{
				status:  status,
				subject: label,
				detail:  detail,
			}
			return resolution
		}
	}
	value, ok := inspection.ValueDomainEntries[id]
	if ok {
		resolution = locatorResolution[cli.ValueDomainEntryInspection]{
			status:  locatorResolutionFound,
			value:   value,
			subject: label,
		}
		return resolution
	}
	if inspection.Truncated {
		resolution = locatorResolution[cli.ValueDomainEntryInspection]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  observed(semanticSubjectInspection, observationTruncated),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ValueDomainEntryInspection]{
		status:  locatorResolutionAbsent,
		subject: label,
		detail:  observed(semanticSubjectValueDomainEntry, observationAbsent),
	}
	return resolution
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
//nolint:dupl // Separate public artifacts resolve their own value-domain-entry relationships directly.
func selectSchemaValueDomainEntry(state *valueDomainEntryLocatorState, ids []cli.ValueDomainEntryID, schema *cli.Schema, complete bool) (cli.ValueDomainEntryID, locatorResolutionStatus, evaluationField) {
	switch state.mode {
	case valueDomainEntryLocatorIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectValueDomainEntry, observationInvalidLocator)
		}
		if state.index < len(ids) {
			return ids[state.index], locatorResolutionFound, evaluationText("")
		}
		if complete {
			return 0, locatorResolutionAbsent, observed(semanticSubjectValueDomainEntry, observationAbsent)
		}
		return 0, locatorResolutionNotProvable, evaluationText("value-domain entry order is incomplete")
	case valueDomainEntryLocatorName:
		matches := make([]cli.ValueDomainEntryID, 0, 1)
		missingDetail := false
		for _, id := range ids {
			entry, ok := schema.ValueDomainEntries[id]
			if !ok {
				missingDetail = true
				continue
			}
			if entry.Name == state.name {
				matches = append(matches, id)
			}
		}
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
			return 0, locatorResolutionNotProvable, evaluationText("value-domain entry lookup domain is incomplete")
		}
		return 0, locatorResolutionAbsent, observed(semanticSubjectValueDomainEntry, observationAbsent)
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectValueDomainEntry, observationInvalidLocator)
	}
}

//nolint:dupl // Separate public artifacts resolve their own value-domain-entry relationships directly.
func selectReferenceValueDomainEntry(state *valueDomainEntryLocatorState, ids []cli.ValueDomainEntryID, reference cli.Reference, complete bool) (cli.ValueDomainEntryID, locatorResolutionStatus, evaluationField) {
	switch state.mode {
	case valueDomainEntryLocatorIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectValueDomainEntry, observationInvalidLocator)
		}
		if state.index < len(ids) {
			return ids[state.index], locatorResolutionFound, evaluationText("")
		}
		if complete {
			return 0, locatorResolutionAbsent, observed(semanticSubjectValueDomainEntry, observationAbsent)
		}
		return 0, locatorResolutionNotProvable, evaluationText("value-domain entry order is incomplete")
	case valueDomainEntryLocatorName:
		matches := make([]cli.ValueDomainEntryID, 0, 1)
		missingDetail := false
		for _, id := range ids {
			entry, ok := reference.ValueDomainEntries[id]
			if !ok {
				missingDetail = true
				continue
			}
			if entry.Name == state.name {
				matches = append(matches, id)
			}
		}
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
			return 0, locatorResolutionNotProvable, evaluationText("value-domain entry lookup domain is incomplete")
		}
		return 0, locatorResolutionAbsent, observed(semanticSubjectValueDomainEntry, observationAbsent)
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectValueDomainEntry, observationInvalidLocator)
	}
}

//nolint:dupl // Separate public artifacts resolve their own value-domain-entry relationships directly.
func selectInspectionValueDomainEntry(state *valueDomainEntryLocatorState, ids []cli.ValueDomainEntryID, inspection cli.Inspection, complete bool) (cli.ValueDomainEntryID, locatorResolutionStatus, evaluationField) {
	switch state.mode {
	case valueDomainEntryLocatorIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectValueDomainEntry, observationInvalidLocator)
		}
		if state.index < len(ids) {
			return ids[state.index], locatorResolutionFound, evaluationText("")
		}
		if complete {
			return 0, locatorResolutionAbsent, observed(semanticSubjectValueDomainEntry, observationAbsent)
		}
		return 0, locatorResolutionNotProvable, evaluationText("value-domain entry order is incomplete")
	case valueDomainEntryLocatorName:
		matches := make([]cli.ValueDomainEntryID, 0, 1)
		missingDetail := false
		for _, id := range ids {
			entry, ok := inspection.ValueDomainEntries[id]
			if !ok {
				missingDetail = true
				continue
			}
			if entry.Name == state.name {
				matches = append(matches, id)
			}
		}
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
			return 0, locatorResolutionNotProvable, evaluationText("value-domain entry lookup domain is incomplete")
		}
		return 0, locatorResolutionAbsent, observed(semanticSubjectValueDomainEntry, observationAbsent)
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectValueDomainEntry, observationInvalidLocator)
	}
}

func expectedInspectionValueDomainEntryID(state *valueDomainEntryLocatorState, inspection cli.Inspection) (cli.ValueDomainEntryID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == valueDomainEntryLocatorInvalid {
		return 0, locatorResolutionInvalid, evaluationText("invalid value-domain-entry locator")
	}
	if state.mode == valueDomainEntryLocatorByID {
		if state.id == 0 {
			return 0, locatorResolutionInvalid, evaluationText("invalid value-domain-entry locator")
		}
		return state.id, locatorResolutionFound, evaluationText("")
	}
	domain := resolveInspectionValueDomainLocator(state.domain, inspection)
	if domain.status != locatorResolutionFound {
		return 0, domain.status, domain.detail
	}
	id, status, detail := selectInspectionValueDomainEntry(state, domain.value.Entries, inspection, !inspection.Truncated)
	if status == locatorResolutionInvalid && emptyEvaluationField(detail) {
		detail = evaluationText("invalid value-domain-entry locator")
	}
	return id, status, detail
}

func expectedSchemaValueDomainEntryID(state *valueDomainEntryLocatorState, schema *cli.Schema) (cli.ValueDomainEntryID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == valueDomainEntryLocatorInvalid {
		return 0, locatorResolutionInvalid, evaluationText("invalid value-domain-entry locator")
	}
	if state.mode == valueDomainEntryLocatorByID {
		if state.id == 0 {
			return 0, locatorResolutionInvalid, evaluationText("invalid value-domain-entry locator")
		}
		return state.id, locatorResolutionFound, evaluationText("")
	}
	resolution := resolveSchemaValueDomainEntryLocator(state, schema)
	if resolution.status != locatorResolutionFound {
		return 0, resolution.status, resolution.detail
	}
	return resolution.value.ID, locatorResolutionFound, evaluationText("")
}

func expectedReferenceValueDomainEntryID(state *valueDomainEntryLocatorState, reference cli.Reference) (cli.ValueDomainEntryID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == valueDomainEntryLocatorInvalid {
		return 0, locatorResolutionInvalid, evaluationText("invalid value-domain-entry locator")
	}
	if state.mode == valueDomainEntryLocatorByID {
		if state.id == 0 {
			return 0, locatorResolutionInvalid, evaluationText("invalid value-domain-entry locator")
		}
		return state.id, locatorResolutionFound, evaluationText("")
	}
	resolution := resolveReferenceValueDomainEntryLocator(state, reference)
	if resolution.status != locatorResolutionFound {
		return 0, resolution.status, resolution.detail
	}
	return resolution.value.ID, locatorResolutionFound, evaluationText("")
}

func evaluateValueDomainEntryReplacementPresentationSchema(expectation valueDomainEntryReplacementPresentationExpectation, schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaValueDomainEntryLocator(expectation.locator, schema)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func evaluateValueDomainEntryReplacementPresentationReference(expectation valueDomainEntryReplacementPresentationExpectation, reference cli.Reference) evaluationResult {
	resolution := resolveReferenceValueDomainEntryLocator(expectation.locator, reference)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func evaluateValueDomainEntryReplacementPresentationInspection(expectation valueDomainEntryReplacementPresentationExpectation, inspection cli.Inspection) evaluationResult {
	resolution := resolveInspectionValueDomainEntryLocator(expectation.locator, inspection)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement presentation")), resolution.detail, evaluationText(expectation.expected), evaluationText(""), false)
	}
	if resolution.value.Replacement == nil {
		return disprovedResult(qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(expectation.expected), evaluationText("replacement is absent"))
	}
	actual := resolution.value.Replacement.Presentation
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(resolution.subject, evaluationText("replacement presentation")), evaluationText(""), evaluationText(expectation.expected), evaluationText(actual), actual == expectation.expected)
}

func valueDomainEntryIDLocator(id cli.ValueDomainEntryID) ValueDomainEntryLocator {
	state := &valueDomainEntryLocatorState{
		mode: valueDomainEntryLocatorByID,
		id:   id,
	}
	locator := ValueDomainEntryLocator{
		state: state,
	}
	return locator
}

func schemaValueDomainUseCoordinate(state *valueDomainLocatorState, schema *cli.Schema) (cli.ValueDomainUse, locatorResolutionStatus, evaluationField) {
	var domainUse cli.ValueDomainUse
	if state == nil || !state.valid {
		return cli.ValueDomainUse{}, locatorResolutionInvalid, observed(semanticSubjectValueDomainUse, observationInvalidLocator)
	}
	id, status, detail := resolveSchemaValueID(state.value, schema)
	if status != locatorResolutionFound {
		return cli.ValueDomainUse{}, status, detail
	}
	domainUse = cli.ValueDomainUse{
		Destination: id,
		Kind:        state.use,
	}
	return domainUse, locatorResolutionFound, evaluationText("")
}

func referenceValueDomainUseCoordinate(state *valueDomainLocatorState, reference cli.Reference) (cli.ValueDomainUse, locatorResolutionStatus, evaluationField) {
	var domainUse cli.ValueDomainUse
	if state == nil || !state.valid {
		return cli.ValueDomainUse{}, locatorResolutionInvalid, observed(semanticSubjectValueDomainUse, observationInvalidLocator)
	}
	id, status, detail := resolveReferenceValueID(state.value, reference)
	if status != locatorResolutionFound {
		return cli.ValueDomainUse{}, status, detail
	}
	domainUse = cli.ValueDomainUse{
		Destination: id,
		Kind:        state.use,
	}
	return domainUse, locatorResolutionFound, evaluationText("")
}

func inspectionValueDomainUseCoordinate(state *valueDomainLocatorState, inspection cli.Inspection) (cli.ValueDomainUse, locatorResolutionStatus, evaluationField) {
	var domainUse cli.ValueDomainUse
	if state == nil || !state.valid {
		return cli.ValueDomainUse{}, locatorResolutionInvalid, observed(semanticSubjectValueDomainUse, observationInvalidLocator)
	}
	id, status, detail := resolveInspectionValueID(state.value, inspection)
	if status != locatorResolutionFound {
		return cli.ValueDomainUse{}, status, detail
	}
	domainUse = cli.ValueDomainUse{
		Destination: id,
		Kind:        state.use,
	}
	return domainUse, locatorResolutionFound, evaluationText("")
}

func valueDomainEntryNameLocator(parent ValueDomainLocator, name string) ValueDomainEntryLocator {
	state := &valueDomainEntryLocatorState{
		mode:   valueDomainEntryLocatorName,
		domain: parent.state,
		name:   name,
	}
	locator := ValueDomainEntryLocator{
		state: state,
	}
	return locator
}

func valueDomainEntryAtLocator(parent ValueDomainLocator, index int) ValueDomainEntryLocator {
	state := &valueDomainEntryLocatorState{
		mode:   valueDomainEntryLocatorIndex,
		domain: parent.state,
		index:  index,
	}
	locator := ValueDomainEntryLocator{
		state: state,
	}
	return locator
}

func valueDomainUseLocator(value ValueLocator, kind cli.ValueDomainUseKind) ValueDomainLocator {
	state := &valueDomainLocatorState{
		valid: true,
		value: value.state,
		use:   kind,
	}
	locator := ValueDomainLocator{
		state: state,
	}
	return locator
}

func valueDomainOutcomeExpectation(locator ValueDomainLocator, outcome cli.ValueDomainOutcome) InspectionExpectation {
	expectation := domainPropertyExpectation{
		locator: locator.state,
		kind:    valueDomainPropertyOutcome,
		outcome: outcome,
	}
	return expectation
}

func valueDomainInvocationScopedExpectation(locator ValueDomainLocator) InspectionExpectation {
	expectation := domainPropertyExpectation{
		locator: locator.state,
		kind:    valueDomainPropertyInvocationScoped,
	}
	return expectation
}

func valueDomainDependenciesExactlyExpectation(locator ValueDomainLocator, values []ValueLocator) InspectionExpectation {
	states := make([]*valueLocatorState, len(values))
	for index, value := range values {
		states[index] = value.state
	}
	expectation := domainPropertyExpectation{
		locator: locator.state,
		kind:    valueDomainPropertyDependencies,
		values:  states,
	}
	return expectation
}

func (valueDomainEntryAvailabilityExpectationValue) referenceExpectation() {}

func (valueDomainEntryAvailabilityExpectationValue) inspectionExpectation() {}

func (expectation valueDomainEntryAvailabilityExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceValueDomainEntryLocator(expectation.locator, reference)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "available", expectation.expected, resolution.value.Available)
}

func (expectation valueDomainEntryAvailabilityExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionValueDomainEntryLocator(expectation.locator, inspection)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "available", expectation.expected, resolution.value.Available)
}

func (valueDomainEntryVisibilityExpectationValue) referenceExpectation() {}

func (valueDomainEntryVisibilityExpectationValue) inspectionExpectation() {}

func (expectation valueDomainEntryVisibilityExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceValueDomainEntryLocator(expectation.locator, reference)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "visible", expectation.expected, resolution.value.Visible)
}

func (expectation valueDomainEntryVisibilityExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionValueDomainEntryLocator(expectation.locator, inspection)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "visible", expectation.expected, resolution.value.Visible)
}

func (valueDomainEntryOriginExpectation) schemaExpectation() {}

func (valueDomainEntryOriginExpectation) referenceExpectation() {}

func (valueDomainEntryOriginExpectation) inspectionExpectation() {}

func (expectation valueDomainEntryOriginExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaValueDomainEntryLocator(expectation.locator, schema)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (expectation valueDomainEntryOriginExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceValueDomainEntryLocator(expectation.locator, reference)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (expectation valueDomainEntryOriginExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionValueDomainEntryLocator(expectation.locator, inspection)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (valueDomainEntryReleaseLifecycleExpectation) schemaExpectation() {}

func (valueDomainEntryReleaseLifecycleExpectation) referenceExpectation() {}

func (valueDomainEntryReleaseLifecycleExpectation) inspectionExpectation() {}

func (expectation valueDomainEntryReleaseLifecycleExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaValueDomainEntryLocator(expectation.locator, schema)
	actual := resolution.value.Release
	field := releaseLifecycleField{
		value: expectation.expected,
	}
	observedField := releaseLifecycleField{
		value: actual,
	}
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, field, observedField, actual == expectation.expected)
}

func (expectation valueDomainEntryReleaseLifecycleExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceValueDomainEntryLocator(expectation.locator, reference)
	actual := resolution.value.Release
	field := releaseLifecycleField{
		value: expectation.expected,
	}
	observedField := releaseLifecycleField{
		value: actual,
	}
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, field, observedField, actual == expectation.expected)
}

func (expectation valueDomainEntryReleaseLifecycleExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionValueDomainEntryLocator(expectation.locator, inspection)
	actual := resolution.value.Release
	field := releaseLifecycleField{
		value: expectation.expected,
	}
	observedField := releaseLifecycleField{
		value: actual,
	}
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("release lifecycle")), resolution.detail, field, observedField, actual == expectation.expected)
}

func (valueDomainEntryReplacementAbsenceExpectation) schemaExpectation() {}

func (valueDomainEntryReplacementAbsenceExpectation) referenceExpectation() {}

func (valueDomainEntryReplacementAbsenceExpectation) inspectionExpectation() {}

func (expectation valueDomainEntryReplacementAbsenceExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaValueDomainEntryLocator(expectation.locator, schema)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (expectation valueDomainEntryReplacementAbsenceExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceValueDomainEntryLocator(expectation.locator, reference)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (expectation valueDomainEntryReplacementAbsenceExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionValueDomainEntryLocator(expectation.locator, inspection)
	replacement := resolution.value.Replacement
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("replacement")), resolution.detail, evaluationText("replacement absent"), replacementAbsentField(replacement), replacement == nil)
}

func (valueDomainEntryReplacementPresentationExpectation) schemaExpectation() {}

func (valueDomainEntryReplacementPresentationExpectation) referenceExpectation() {}

func (valueDomainEntryReplacementPresentationExpectation) inspectionExpectation() {}

func (expectation valueDomainEntryReplacementPresentationExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	return evaluateValueDomainEntryReplacementPresentationSchema(expectation, schema)
}

func (expectation valueDomainEntryReplacementPresentationExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return evaluateValueDomainEntryReplacementPresentationReference(expectation, reference)
}

func (expectation valueDomainEntryReplacementPresentationExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return evaluateValueDomainEntryReplacementPresentationInspection(expectation, inspection)
}

func (valueDomainEntryReplacementTargetDeprecatedExpectation) schemaExpectation() {}

func (valueDomainEntryReplacementTargetDeprecatedExpectation) referenceExpectation() {}

func (valueDomainEntryReplacementTargetDeprecatedExpectation) inspectionExpectation() {}

func (expectation valueDomainEntryReplacementTargetDeprecatedExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaValueDomainEntryLocator(expectation.locator, schema)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (expectation valueDomainEntryReplacementTargetDeprecatedExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceValueDomainEntryLocator(expectation.locator, reference)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (expectation valueDomainEntryReplacementTargetDeprecatedExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionValueDomainEntryLocator(expectation.locator, inspection)
	return evaluateReplacementTargetDeprecated(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected)
}

func (valueDomainEntryExistenceExpectationValue) schemaExpectation() {}

func (valueDomainEntryExistenceExpectationValue) referenceExpectation() {}

func (valueDomainEntryExistenceExpectationValue) inspectionExpectation() {}

func (expectation valueDomainEntryExistenceExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	if schema == nil {
		resolution := locatorResolution[cli.ValueDomainEntrySchema]{
			status:  locatorResolutionInvalid,
			subject: semanticSubjectValueDomainEntry,
			detail:  observed(semanticSubjectValueDomainEntry, observationInvalidLocator),
		}
		return declarationExistenceResult(resolution, expectation.expected)
	}
	return declarationExistenceResult(resolveSchemaValueDomainEntryLocator(expectation.locator, schema), expectation.expected)
}

func (expectation valueDomainEntryExistenceExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return declarationExistenceResult(resolveReferenceValueDomainEntryLocator(expectation.locator, reference), expectation.expected)
}

func (expectation valueDomainEntryExistenceExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return declarationExistenceResult(resolveInspectionValueDomainEntryLocator(expectation.locator, inspection), expectation.expected)
}

func (helpValueDomainEntryExpectation) helpPresentationExpectation() {}

func (expectation helpValueDomainEntryExpectation) evaluateHelp(page cli.HelpPage) evaluationResult {
	entry, status, detail := resolveHelpValueDomainEntry(expectation.locator, page)
	subject := evaluationText("help value-domain entry")
	if expectation.property == helpValueDomainEntryExists {
		return evaluateHelpExistence(subject, expectation.exists, status)
	}
	if status != locatorResolutionFound {
		return helpResolutionFailure(subject, expectation.valueDomainEntryExpected(), detail)
	}
	switch expectation.property {
	case helpValueDomainEntryKind:
		if entry.Kind == expectation.kind {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "kind", evaluationText(fmt.Sprint(expectation.kind)), evaluationText(fmt.Sprint(entry.Kind)))
	case helpValueDomainEntryName:
		if entry.Name == expectation.text {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "name", evaluationText(expectation.text), evaluationText(entry.Name))
	case helpValueDomainEntryDescription:
		if entry.Description == expectation.text {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "description", evaluationText(expectation.text), evaluationText(entry.Description))
	case helpValueDomainEntryLifecycle:
		if entry.LifecycleState == expectation.lifecycle {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "lifecycle", evaluationText(fmt.Sprint(expectation.lifecycle)), evaluationText(fmt.Sprint(entry.LifecycleState)))
	case helpValueDomainEntryNoCanonical:
		if entry.Canonical == 0 {
			return provedResult()
		}
		return disprovedResult(subject, evaluationText("canonical absent"), evaluationText("canonical relation is present"))
	case helpValueDomainEntryNotReplaced:
		if entry.Replacement == 0 {
			return provedResult()
		}
		return disprovedResult(subject, evaluationText("replacement absent"), evaluationText("replacement relation is present"))
	default:
		panic("clitest: invalid Help value-domain-entry expectation")
	}
}

func (expectation helpValueDomainEntryExpectation) valueDomainEntryExpected() evaluationField {
	switch expectation.property {
	case helpValueDomainEntryKind:
		return property(evaluationText("kind"), evaluationText(fmt.Sprint(expectation.kind)))
	case helpValueDomainEntryName:
		return property(evaluationText("name"), evaluationText(expectation.text))
	case helpValueDomainEntryDescription:
		return property(evaluationText("description"), evaluationText(expectation.text))
	case helpValueDomainEntryLifecycle:
		return property(evaluationText("lifecycle"), evaluationText(fmt.Sprint(expectation.lifecycle)))
	case helpValueDomainEntryNoCanonical:
		return evaluationText("canonical absent")
	case helpValueDomainEntryNotReplaced:
		return evaluationText("replacement absent")
	default:
		return evaluationText("valid value-domain-entry expectation")
	}
}

func (observedValueDomainsExpectation) inspectionExpectation() {}

func (expectation observedValueDomainsExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	if len(inspection.ValueDomains) != len(expectation.domains) {
		return disprovedResult(
			evaluationText("observed value domains"),
			evaluationText(fmt.Sprintf("exactly %d domains in returned order", len(expectation.domains))),
			evaluationText(fmt.Sprintf("%d observed domains", len(inspection.ValueDomains))),
		)
	}
	for index, state := range expectation.domains {
		resolved := resolveDomain(state, inspection)
		if resolved.status != locatorResolutionFound {
			return disprovedResult(evaluationText("observed value domains"), evaluationText(fmt.Sprintf("domain at position %d", index)), resolved.actual)
		}
		if resolved.index != index {
			return disprovedResult(
				evaluationText("observed value domains"),
				evaluationText(fmt.Sprintf("semantic domain at position %d", index)),
				evaluationText("different observed domain order"),
			)
		}
	}
	return provedResult()
}

func (observedValueDomainEntriesExpectation) inspectionExpectation() {}

func (expectation observedValueDomainEntriesExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	if len(inspection.ValueDomainEntries) != len(expectation.entries) {
		return disprovedResult(
			evaluationText("observed value-domain entries"),
			evaluationText(fmt.Sprintf("exactly %d entries", len(expectation.entries))),
			evaluationText(fmt.Sprintf("%d observed entries", len(inspection.ValueDomainEntries))),
		)
	}
	seen := map[cli.ValueDomainEntryID]struct{}{}
	for index, state := range expectation.entries {
		resolved := resolveInspectionValueDomainEntryLocator(state, inspection)
		if resolved.status != locatorResolutionFound {
			return outcomeForValueStatus(
				resolved.status,
				evaluationText("observed value-domain entries"),
				evaluationText(fmt.Sprintf("entry at position %d", index)),
				resolved.detail,
			)
		}
		id := resolved.value.ID
		if _, duplicate := seen[id]; duplicate {
			return disprovedResult(
				evaluationText("observed value-domain entries"),
				evaluationText("distinct semantic entries"),
				evaluationText(fmt.Sprintf("duplicate expected entry at position %d", index)),
			)
		}
		seen[id] = struct{}{}
	}
	return provedResult()
}

func (valueDomainExistenceExpectationValue) schemaExpectation() {}

func (valueDomainExistenceExpectationValue) referenceExpectation() {}

func (valueDomainExistenceExpectationValue) inspectionExpectation() {}

func (expectation valueDomainExistenceExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	if schema == nil {
		resolution := locatorResolution[cli.ValueDomainSchema]{
			status:  locatorResolutionInvalid,
			subject: semanticSubjectValueDomain,
			detail:  observed(semanticSubjectValueDomain, observationInvalidLocator),
		}
		return declarationExistenceResult(resolution, expectation.expected)
	}
	return declarationExistenceResult(resolveSchemaValueDomainLocator(expectation.locator, *schema), expectation.expected)
}

func (expectation valueDomainExistenceExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return declarationExistenceResult(resolveReferenceValueDomainLocator(expectation.locator, reference), expectation.expected)
}

func (expectation valueDomainExistenceExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return declarationExistenceResult(resolveInspectionValueDomainLocator(expectation.locator, inspection), expectation.expected)
}

func (valueDomainEntriesExpectationValue) schemaExpectation() {}

func (valueDomainEntriesExpectationValue) referenceExpectation() {}

func (valueDomainEntriesExpectationValue) inspectionExpectation() {}

func (expectation valueDomainEntriesExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	if schema == nil {
		return disprovedResult(semanticSubjectValueDomain, evaluationText("exact value-domain entries"), observed(semanticSubjectValueDomain, observationInvalidLocator))
	}
	domain := resolveSchemaValueDomainLocator(expectation.locator, *schema)
	if domain.status != locatorResolutionFound {
		return declarationResolvedComparison(domain.status, domain.subject, domain.detail, evaluationText("exact value-domain entries"), evaluationText(""), false)
	}
	ids := make([]cli.ValueDomainEntryID, len(expectation.entries))
	for index, state := range expectation.entries {
		id, status, detail := expectedSchemaValueDomainEntryID(state, schema)
		if status == locatorResolutionNotProvable {
			return notProvableResult(domain.subject, evaluationText("exact value-domain entries"), withDetail(fmt.Sprintf("entry %d", index), detail))
		}
		if status != locatorResolutionFound {
			return disprovedResult(domain.subject, evaluationText("exact value-domain entries"), withDetail(fmt.Sprintf("entry %d", index), detail))
		}
		ids[index] = id
	}
	if slices.Equal(domain.value.Entries, ids) {
		return provedResult()
	}
	return orderedCollectionMismatch(domain.subject, "entries", ids, domain.value.Entries)
}

func (expectation valueDomainEntriesExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	domain := resolveReferenceValueDomainLocator(expectation.locator, reference)
	if domain.status != locatorResolutionFound {
		return declarationResolvedComparison(domain.status, domain.subject, domain.detail, evaluationText("exact value-domain entries"), evaluationText(""), false)
	}
	ids := make([]cli.ValueDomainEntryID, len(expectation.entries))
	for index, state := range expectation.entries {
		id, status, detail := expectedReferenceValueDomainEntryID(state, reference)
		if status == locatorResolutionNotProvable {
			return notProvableResult(domain.subject, evaluationText("exact value-domain entries"), withDetail(fmt.Sprintf("entry %d", index), detail))
		}
		if status != locatorResolutionFound {
			return disprovedResult(domain.subject, evaluationText("exact value-domain entries"), withDetail(fmt.Sprintf("entry %d", index), detail))
		}
		ids[index] = id
	}
	if slices.Equal(domain.value.Entries, ids) {
		return provedResult()
	}
	return orderedCollectionMismatch(domain.subject, "entries", ids, domain.value.Entries)
}

func (expectation valueDomainEntriesExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	domain := resolveInspectionValueDomainLocator(expectation.locator, inspection)
	if domain.status != locatorResolutionFound {
		return declarationResolvedComparison(domain.status, domain.subject, domain.detail, evaluationText("exact value-domain entries"), evaluationText(""), false)
	}
	if inspection.Truncated {
		return notProvableResult(domain.subject, evaluationText("exact value-domain entries"), observed(semanticSubjectInspection, observationTruncated))
	}
	ids := make([]cli.ValueDomainEntryID, len(expectation.entries))
	for index, state := range expectation.entries {
		id, status, detail := expectedInspectionValueDomainEntryID(state, inspection)
		if status == locatorResolutionNotProvable {
			return notProvableResult(domain.subject, evaluationText("exact value-domain entries"), withDetail(fmt.Sprintf("entry %d", index), detail))
		}
		if status != locatorResolutionFound {
			return disprovedResult(domain.subject, evaluationText("exact value-domain entries"), withDetail(fmt.Sprintf("entry %d", index), detail))
		}
		ids[index] = id
	}
	if slices.Equal(domain.value.Entries, ids) {
		return provedResult()
	}
	return orderedCollectionMismatch(domain.subject, "entries", ids, domain.value.Entries)
}

func (valueDomainEntryKindExpectationValue) schemaExpectation() {}

func (valueDomainEntryKindExpectationValue) referenceExpectation() {}

func (valueDomainEntryKindExpectationValue) inspectionExpectation() {}

func (expectation valueDomainEntryKindExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaValueDomainEntryLocator(expectation.locator, schema)
	actual := resolution.value.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (expectation valueDomainEntryKindExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceValueDomainEntryLocator(expectation.locator, reference)
	actual := resolution.value.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (expectation valueDomainEntryKindExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionValueDomainEntryLocator(expectation.locator, inspection)
	actual := resolution.value.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (valueDomainEntryHiddenExpectationValue) schemaExpectation() {}

func (valueDomainEntryHiddenExpectationValue) referenceExpectation() {}

func (valueDomainEntryHiddenExpectationValue) inspectionExpectation() {}

func (expectation valueDomainEntryHiddenExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaValueDomainEntryLocator(expectation.locator, schema)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("hidden"), true, resolution.value.Hidden)
}

func (expectation valueDomainEntryHiddenExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceValueDomainEntryLocator(expectation.locator, reference)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("hidden"), true, resolution.value.Hidden)
}

func (expectation valueDomainEntryHiddenExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionValueDomainEntryLocator(expectation.locator, inspection)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("hidden"), true, resolution.value.Hidden)
}

func (valueDomainEntryLifecycleExpectationValue) schemaExpectation() {}

func (valueDomainEntryLifecycleExpectationValue) referenceExpectation() {}

func (valueDomainEntryLifecycleExpectationValue) inspectionExpectation() {}

func (expectation valueDomainEntryLifecycleExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaValueDomainEntryLocator(expectation.locator, schema)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (expectation valueDomainEntryLifecycleExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceValueDomainEntryLocator(expectation.locator, reference)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (expectation valueDomainEntryLifecycleExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionValueDomainEntryLocator(expectation.locator, inspection)
	actual := resolution.value.LifecycleState
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("lifecycle %v", expectation.expected)), evaluationText(fmt.Sprintf("lifecycle %v", actual)), actual == expectation.expected)
}

func (valueDomainEntryReplacementExpectationValue) schemaExpectation() {}

func (valueDomainEntryReplacementExpectationValue) referenceExpectation() {}

func (valueDomainEntryReplacementExpectationValue) inspectionExpectation() {}

func (expectation valueDomainEntryReplacementExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaValueDomainEntryLocator(expectation.locator, schema)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, schema, nil, cli.Inspection{})
}

func (expectation valueDomainEntryReplacementExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceValueDomainEntryLocator(expectation.locator, reference)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, nil, &reference, cli.Inspection{})
}

func (expectation valueDomainEntryReplacementExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionValueDomainEntryLocator(expectation.locator, inspection)
	return replacementRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Replacement, expectation.expected, nil, nil, inspection)
}

func (valueDomainEntryCanonicalExpectationValue) schemaExpectation() {}

func (valueDomainEntryCanonicalExpectationValue) referenceExpectation() {}

func (valueDomainEntryCanonicalExpectationValue) inspectionExpectation() {}

func (expectation valueDomainEntryCanonicalExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	subject := resolveSchemaValueDomainEntryLocator(expectation.locator, schema)
	if subject.status != locatorResolutionFound {
		return declarationResolvedComparison(subject.status, subject.subject, subject.detail, evaluationText("valid canonical value-domain entry"), evaluationText(""), false)
	}
	targetID, status, detail := expectedSchemaValueDomainEntryID(expectation.target, schema)
	if status == locatorResolutionNotProvable {
		return notProvableResult(subject.subject, evaluationText("valid canonical value-domain entry"), detail)
	}
	if status != locatorResolutionFound {
		return disprovedResult(subject.subject, evaluationText("valid canonical value-domain entry"), detail)
	}
	if subject.value.Canonical == targetID {
		return provedResult()
	}
	return disprovedResult(subject.subject, evaluationText(fmt.Sprintf("canonical %d", targetID)), evaluationText(fmt.Sprintf("canonical %d", subject.value.Canonical)))
}

func (expectation valueDomainEntryCanonicalExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	subject := resolveReferenceValueDomainEntryLocator(expectation.locator, reference)
	if subject.status != locatorResolutionFound {
		return declarationResolvedComparison(subject.status, subject.subject, subject.detail, evaluationText("valid canonical value-domain entry"), evaluationText(""), false)
	}
	targetID, status, detail := expectedReferenceValueDomainEntryID(expectation.target, reference)
	if status == locatorResolutionNotProvable {
		return notProvableResult(subject.subject, evaluationText("valid canonical value-domain entry"), detail)
	}
	if status != locatorResolutionFound {
		return disprovedResult(subject.subject, evaluationText("valid canonical value-domain entry"), detail)
	}
	if subject.value.Canonical == targetID {
		return provedResult()
	}
	return disprovedResult(subject.subject, evaluationText(fmt.Sprintf("canonical %d", targetID)), evaluationText(fmt.Sprintf("canonical %d", subject.value.Canonical)))
}

func (expectation valueDomainEntryCanonicalExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	subject := resolveInspectionValueDomainEntryLocator(expectation.locator, inspection)
	if subject.status != locatorResolutionFound {
		return declarationResolvedComparison(subject.status, subject.subject, subject.detail, evaluationText("valid canonical value-domain entry"), evaluationText(""), false)
	}
	targetID, status, detail := expectedInspectionValueDomainEntryID(expectation.target, inspection)
	if status == locatorResolutionNotProvable {
		return notProvableResult(subject.subject, evaluationText("valid canonical value-domain entry"), detail)
	}
	if status != locatorResolutionFound {
		return disprovedResult(subject.subject, evaluationText("valid canonical value-domain entry"), detail)
	}
	if subject.value.Canonical == targetID {
		return provedResult()
	}
	return disprovedResult(subject.subject, evaluationText(fmt.Sprintf("canonical %d", targetID)), evaluationText(fmt.Sprintf("canonical %d", subject.value.Canonical)))
}

func (valueDomainEntrySourceDescriptionExpectationValue) schemaExpectation() {}

func (expectation valueDomainEntrySourceDescriptionExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaValueDomainEntryLocator(expectation.locator, schema)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText("exact source description"), evaluationText(""), false)
	}
	if resolution.value.Description == expectation.expected {
		return provedResult()
	}
	return disprovedResult(resolution.subject, evaluationText("exact source description"), evaluationText("different source description"))
}

func (valueDomainEntryPresentedDescriptionExpectationValue) referenceExpectation() {}

func (valueDomainEntryPresentedDescriptionExpectationValue) inspectionExpectation() {}

func (expectation valueDomainEntryPresentedDescriptionExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceValueDomainEntryLocator(expectation.locator, reference)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText("exact presented description"), evaluationText(""), false)
	}
	if resolution.value.Description == expectation.expected {
		return provedResult()
	}
	return disprovedResult(resolution.subject, evaluationText("exact presented description"), evaluationText("different presented description"))
}

func (expectation valueDomainEntryPresentedDescriptionExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionValueDomainEntryLocator(expectation.locator, inspection)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText("exact presented description"), evaluationText(""), false)
	}
	if resolution.value.Description == expectation.expected {
		return provedResult()
	}
	return disprovedResult(resolution.subject, evaluationText("exact presented description"), evaluationText("different presented description"))
}

func (valueDomainUsesExpectationValue) schemaExpectation() {}

func (valueDomainUsesExpectationValue) referenceExpectation() {}

func (valueDomainUsesExpectationValue) inspectionExpectation() {}

func (expectation valueDomainUsesExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	const requirement = "exact semantic value-domain uses in public order"
	if schema == nil {
		return disprovedResult(semanticSubjectValueDomain, evaluationText(requirement), observed(semanticSubjectValueDomain, observationInvalidLocator))
	}
	domain := resolveSchemaValueDomainLocator(expectation.locator, *schema)
	if domain.status != locatorResolutionFound {
		return declarationResolvedComparison(domain.status, domain.subject, domain.detail, evaluationText(requirement), evaluationText(""), false)
	}
	want := make([]cli.ValueDomainUse, len(expectation.uses))
	for index, state := range expectation.uses {
		use, status, detail := schemaValueDomainUseCoordinate(state, schema)
		if status == locatorResolutionNotProvable {
			return notProvableResult(domain.subject, evaluationText(requirement), withDetail(fmt.Sprintf("use %d", index), detail))
		}
		if status != locatorResolutionFound {
			return disprovedResult(domain.subject, evaluationText(requirement), withDetail(fmt.Sprintf("use %d", index), detail))
		}
		expectedDomain := resolveSchemaValueDomainLocator(state, *schema)
		if expectedDomain.status == locatorResolutionNotProvable {
			return notProvableResult(domain.subject, evaluationText(requirement), withDetail(fmt.Sprintf("use %d", index), expectedDomain.detail))
		}
		if expectedDomain.status != locatorResolutionFound {
			return disprovedResult(domain.subject, evaluationText(requirement), withDetail(fmt.Sprintf("use %d", index), expectedDomain.detail))
		}
		if !slices.Equal(expectedDomain.value.Uses, domain.value.Uses) {
			return disprovedResult(domain.subject, evaluationText(requirement), evaluationText(fmt.Sprintf("expected use at position %d resolves to a different domain", index)))
		}
		want[index] = use
	}
	if slices.Equal(domain.value.Uses, want) {
		return provedResult()
	}
	return disprovedResult(domain.subject, evaluationText(requirement), evaluationText("different semantic use sequence"))
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func (expectation valueDomainUsesExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	const requirement = "exact semantic value-domain uses in public order"
	domain := resolveReferenceValueDomainLocator(expectation.locator, reference)
	if domain.status != locatorResolutionFound {
		return declarationResolvedComparison(domain.status, domain.subject, domain.detail, evaluationText(requirement), evaluationText(""), false)
	}
	want := make([]cli.ValueDomainUse, len(expectation.uses))
	for index, state := range expectation.uses {
		use, status, detail := referenceValueDomainUseCoordinate(state, reference)
		if status == locatorResolutionNotProvable {
			return notProvableResult(domain.subject, evaluationText(requirement), withDetail(fmt.Sprintf("use %d", index), detail))
		}
		if status != locatorResolutionFound {
			return disprovedResult(domain.subject, evaluationText(requirement), withDetail(fmt.Sprintf("use %d", index), detail))
		}
		expectedDomain := resolveReferenceValueDomainLocator(state, reference)
		if expectedDomain.status == locatorResolutionNotProvable {
			return notProvableResult(domain.subject, evaluationText(requirement), withDetail(fmt.Sprintf("use %d", index), expectedDomain.detail))
		}
		if expectedDomain.status != locatorResolutionFound {
			return disprovedResult(domain.subject, evaluationText(requirement), withDetail(fmt.Sprintf("use %d", index), expectedDomain.detail))
		}
		if !slices.Equal(expectedDomain.value.Uses, domain.value.Uses) {
			return disprovedResult(domain.subject, evaluationText(requirement), evaluationText(fmt.Sprintf("expected use at position %d resolves to a different domain", index)))
		}
		want[index] = use
	}
	if slices.Equal(domain.value.Uses, want) {
		return provedResult()
	}
	return disprovedResult(domain.subject, evaluationText(requirement), evaluationText("different semantic use sequence"))
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func (expectation valueDomainUsesExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	const requirement = "exact semantic value-domain uses in public order"
	domain := resolveInspectionValueDomainLocator(expectation.locator, inspection)
	if domain.status != locatorResolutionFound {
		return declarationResolvedComparison(domain.status, domain.subject, domain.detail, evaluationText(requirement), evaluationText(""), false)
	}
	if inspection.Truncated {
		return notProvableResult(domain.subject, evaluationText(requirement), observed(semanticSubjectInspection, observationTruncated))
	}
	want := make([]cli.ValueDomainUse, len(expectation.uses))
	for index, state := range expectation.uses {
		use, status, detail := inspectionValueDomainUseCoordinate(state, inspection)
		if status == locatorResolutionNotProvable {
			return notProvableResult(domain.subject, evaluationText(requirement), withDetail(fmt.Sprintf("use %d", index), detail))
		}
		if status != locatorResolutionFound {
			return disprovedResult(domain.subject, evaluationText(requirement), withDetail(fmt.Sprintf("use %d", index), detail))
		}
		expectedDomain := resolveInspectionValueDomainLocator(state, inspection)
		if expectedDomain.status == locatorResolutionNotProvable {
			return notProvableResult(domain.subject, evaluationText(requirement), withDetail(fmt.Sprintf("use %d", index), expectedDomain.detail))
		}
		if expectedDomain.status != locatorResolutionFound {
			return disprovedResult(domain.subject, evaluationText(requirement), withDetail(fmt.Sprintf("use %d", index), expectedDomain.detail))
		}
		if !slices.Equal(expectedDomain.value.Uses, domain.value.Uses) {
			return disprovedResult(domain.subject, evaluationText(requirement), evaluationText(fmt.Sprintf("expected use at position %d resolves to a different domain", index)))
		}
		want[index] = use
	}
	if slices.Equal(domain.value.Uses, want) {
		return provedResult()
	}
	return disprovedResult(domain.subject, evaluationText(requirement), evaluationText("different semantic use sequence"))
}
