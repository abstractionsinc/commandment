package clitest

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/abstractionsinc/commandment/cli"
)

// FactLocator identifies one declared fact for clitest semantic expectation. Its zero
// value is invalid.
type FactLocator struct {
	_     [0]func()
	state *factLocatorState
}

// ValueStaticDefaultLocator identifies the static default attached to a public value
// declaration. It distinguishes absence from scalar, sequence, keyed, and representation-free
// states and is valid only when obtained from a ValueLocator.
type ValueStaticDefaultLocator struct{ value ValueLocator }

// ValueSequenceLiteralLocator addresses the ordered sequence representation of a value's static
// default. Element indexes follow the public representation order; the zero value is invalid.
type ValueSequenceLiteralLocator struct{ root ValueStaticDefaultLocator }

// ValueKeyedLiteralLocator addresses the keyed representation of a value's static default.
// Entry indexes follow the public representation order; the zero value is invalid.
type ValueKeyedLiteralLocator struct{ root ValueStaticDefaultLocator }

// ValueKeyedEntryLocator identifies one ordered key/value entry in a keyed static default. It
// follows the public entry relationship and is invalid when zero-valued.
type ValueKeyedEntryLocator struct {
	root  ValueStaticDefaultLocator
	index int
}

// StaticValueLiteralLocator identifies one public literal within a static-default
// representation. Text expectations remain subject to Commandment sensitivity and redaction rules;
// the zero value is invalid.
type StaticValueLiteralLocator struct {
	root  ValueStaticDefaultLocator
	mode  staticValueLiteralMode
	index int
}

// ValueLocator identifies one declared or inspected value for clitest semantic
// expectation. Its zero value is invalid.
type ValueLocator struct {
	_     [0]func()
	state *valueLocatorState
}

// ValueProducerLocator identifies one producer record for a value in Inspection
// for clitest semantic expectation. Its zero value is invalid.
type ValueProducerLocator struct {
	_     [0]func()
	state *valueProducerLocatorState
}

// ValueCandidateLocator identifies one result-local value candidate in
// Inspection for clitest semantic expectation. Candidate IDs are result-local; the
// locator carries locator coordinates only. Its zero value is invalid.
type ValueCandidateLocator struct {
	_     [0]func()
	state *valueCandidateLocatorState
}

// ValueComponentLocator identifies one structural value component in Inspection
// for clitest semantic expectation. Its zero value is invalid.
type ValueComponentLocator struct {
	_     [0]func()
	state *valueComponentLocatorState
}

// ValueElementLocator identifies one sequence element through its
// value-component relationship for clitest semantic expectation. Its zero value is
// invalid.
type ValueElementLocator struct{ ValueComponentLocator }

// ValueEntryLocator identifies one keyed value entry in Inspection for clitest
// semantic expectation. Its zero value is invalid.
type ValueEntryLocator struct {
	_     [0]func()
	state *valueEntryLocatorState
}

// FactByID returns a locator for the exact public ValueID identifying a fact in the
// observation being evaluated.
func FactByID(id cli.ValueID) FactLocator { return factIDLocator(id) }

// ValueByID returns a locator for the exact public ValueID in the observation being evaluated.
func ValueByID(id cli.ValueID) ValueLocator { return valueIDLocator(id) }

// ValueCandidateByID returns a locator for the exact result-local ValueCandidateID in the
// Inspection being evaluated.
func ValueCandidateByID(id cli.ValueCandidateID) ValueCandidateLocator {
	return valueCandidateIDLocator(id)
}

func (FactLocator) tokenOwner() {}

// DependenciesExactly returns a declaration expectation that checks the fact's complete
// dependency list by public value identity and authoritative order.
func (locator FactLocator) DependenciesExactly(values ...ValueLocator) DeclarationExpectation {
	dependencies := make([]*valueLocatorState, len(values))
	for index, value := range values {
		dependencies[index] = value.state
	}
	expectation := factDependenciesExactlyExpectation{
		locator:      locator.state,
		dependencies: dependencies,
	}
	return expectation
}

// Value follows the public fact-to-value relationship.
func (locator FactLocator) Value() ValueLocator { return factValueLocator(locator) }

// Exists returns an expectation that checks the located fact exists in the relevant public
// declaration observation.
func (locator FactLocator) Exists() DeclarationExpectation {
	expectation := factExistenceExpectation{
		locator:  locator.state,
		expected: true,
	}
	return expectation
}

// DoesNotExist returns an expectation that checks the located fact is absent from a complete
// relevant public declaration domain; incomplete observation cannot manufacture absence.
func (locator FactLocator) DoesNotExist() DeclarationExpectation {
	expectation := factExistenceExpectation{
		locator:  locator.state,
		expected: false,
	}
	return expectation
}

// HasResolver returns an expectation that checks the declared fact has a resolver.
func (locator FactLocator) HasResolver() DeclarationExpectation {
	value := factHasResolverExpectationValue{
		locator: locator.state,
	}
	return value
}

// Available returns an expectation that checks the located fact is effective and available in
// the public Reference observation.
func (locator FactLocator) Available() ReferenceExpectation {
	expectation := factAvailabilityExpectation{
		locator:  locator.state,
		expected: true,
	}
	return expectation
}

// Unavailable returns an expectation that checks the located fact is unavailable in the public
// Reference observation.
func (locator FactLocator) Unavailable() ReferenceExpectation {
	expectation := factAvailabilityExpectation{
		locator:  locator.state,
		expected: false,
	}
	return expectation
}

// FactAt returns a locator for the fact at index in the located command's authoritative public
// fact order.
func (l CommandLocator) FactAt(index int) FactLocator {
	return commandFactAtLocator(l, index)
}

// ValueAt returns a locator for the value at index in the located command's authoritative public
// value order.
func (l CommandLocator) ValueAt(index int) ValueLocator {
	return commandValueAtLocator(l, index)
}

// FactsExactly returns an expectation that checks the located command's facts are exactly the
// supplied locators in authoritative public order. Exactness is not inferred from an incomplete
// observation.
func (l CommandLocator) FactsExactly(values ...FactLocator) DeclarationExpectation {
	expectation := commandFactsExpectation{
		command: l.state,
		items:   factLocatorStates(values),
	}
	return expectation
}

// ValuesExactly returns an expectation that checks the located command's values are exactly the
// supplied locators in authoritative public order. Exactness is not inferred from an incomplete
// observation.
func (l CommandLocator) ValuesExactly(values ...ValueLocator) DeclarationExpectation {
	expectation := commandValuesExpectation{
		command: l.state,
		items:   valueLocatorStates(values),
	}
	return expectation
}

// Kind returns a reusable declaration expectation that checks the public kind exposed by the
// located value.
func (locator ValueLocator) Kind(expected cli.ValueDestinationKind) DeclarationExpectation {
	expectation := valueCorePropertyExpectation{
		locator:         locator.state,
		kind:            valueCorePropertyDestinationKind,
		destinationKind: expected,
	}
	return expectation
}

// Type returns a reusable declaration expectation that checks the located value exposes the
// expected explicit ValueType. A value with HasType=false dischecks the expectation rather than
// supplying an inferred type.
func (locator ValueLocator) Type(expected cli.ValueType) DeclarationExpectation {
	expectation := valueCorePropertyExpectation{
		locator:   locator.state,
		kind:      valueCorePropertyType,
		valueType: expected,
	}
	return expectation
}

// NoType returns a reusable declaration expectation that checks the located value explicitly
// reports HasType=false.
func (locator ValueLocator) NoType() DeclarationExpectation {
	expectation := valueCorePropertyExpectation{
		locator: locator.state,
		kind:    valueCorePropertyNoType,
	}
	return expectation
}

// StaticDefault returns a locator that follows the public static default relationship from the
// located value. The returned locator carries locator coordinates, not observed state.
func (locator ValueLocator) StaticDefault() ValueStaticDefaultLocator {
	valueStaticDefaultLocator := ValueStaticDefaultLocator{
		value: locator,
	}
	return valueStaticDefaultLocator
}

// DependenciesExactly returns a static declaration expectation that checks the value
// declaration's complete dependency list by public value identity and authoritative order.
func (locator ValueLocator) DependenciesExactly(values ...ValueLocator) StaticDeclarationExpectation {
	dependencies := make([]*valueLocatorState, len(values))
	for index, value := range values {
		dependencies[index] = value.state
	}
	expectation := valueDependenciesExpectation{
		locator:      locator.state,
		dependencies: dependencies,
	}
	return expectation
}

// Origin returns an expectation that checks the located value has the framework-established
// DeclarationOrigin expected.
func (locator ValueLocator) Origin(expected cli.DeclarationOrigin) StaticDeclarationExpectation {
	value := valueOriginStaticExpectationValue{
		locator:  locator.state,
		expected: expected,
	}
	return value
}

// DiagnosticsExactly returns an Inspection expectation that the diagnostics attached to the located subject are exactly
// expected in the subject's public DiagnosticIndexes order.
// This checks the returned attachment relationship; it does not reinterpret diagnostic identity from prose.
func (locator ValueLocator) DiagnosticsExactly(expected ...DiagnosticLocator) InspectionExpectation {
	expectation := valueDiagnosticsExactlyExpectation{
		locator:  locator.state,
		expected: slices.Clone(expected),
	}
	return expectation
}

// NoRepresentation returns an Inspection expectation that the located value has no canonical ValueRepresentation. A
// redacted representation is still a present representation and does not satisfy this expectation.
func (locator ValueLocator) NoRepresentation() InspectionExpectation {
	expectation := valueNoRepresentationExpectation{
		locator: locator.state,
	}
	return expectation
}

// ProducerAt returns the producer at index in the value's authoritative producer order.
func (locator ValueLocator) ProducerAt(index int) ValueProducerLocator {
	return valueProducerIndexLocator(locator, index)
}

// CandidateAt returns the candidate at index in the value's authoritative candidate order.
func (locator ValueLocator) CandidateAt(index int) ValueCandidateLocator {
	return valueCandidateIndexLocator(locator, index)
}

// ContributorAt returns the contributing candidate at index in the value's authoritative
// contributor order.
func (locator ValueLocator) ContributorAt(index int) ValueCandidateLocator {
	return valueContributorIndexLocator(locator, index)
}

// Domain follows the public relationship to the value-domain use of the requested kind.
func (locator ValueLocator) Domain(kind cli.ValueDomainUseKind) ValueDomainLocator {
	return valueDomainUseLocator(locator, kind)
}

// Exists returns an expectation that checks the located value exists in the relevant public
// declaration observation.
func (locator ValueLocator) Exists() StaticDeclarationExpectation {
	value := valueExistenceExpectationValue{
		locator:  locator.state,
		expected: true,
	}
	return value
}

// DoesNotExist returns an expectation that checks the located value is absent from a complete
// relevant public declaration domain; incomplete observation cannot manufacture absence.
func (locator ValueLocator) DoesNotExist() StaticDeclarationExpectation {
	value := valueExistenceExpectationValue{
		locator:  locator.state,
		expected: false,
	}
	return value
}

// Required returns an expectation that checks the value is statically declared required.
func (locator ValueLocator) Required() StaticDeclarationExpectation {
	value := valueRequiredStaticExpectationValue{
		locator: locator.state,
	}
	return value
}

// DeclaredSensitive returns an expectation that checks the value is statically declared
// sensitive.
func (locator ValueLocator) DeclaredSensitive() StaticDeclarationExpectation {
	value := valueSensitiveStaticExpectationValue{
		locator: locator.state,
	}
	return value
}

// Observed returns an expectation that checks the value is present in the authoritative
// Inspection observation, independently of declaration existence.
func (locator ValueLocator) Observed() InspectionExpectation {
	return valueObservationExpectation(locator, true)
}

// NotObserved returns an expectation that checks the value is absent from a complete relevant
// Inspection domain; partial observation cannot manufacture absence.
func (locator ValueLocator) NotObserved() InspectionExpectation {
	return valueObservationExpectation(locator, false)
}

// Sensitive returns an expectation that checks the inspected value is sensitive; failure
// reporting preserves Commandment redaction semantics.
func (locator ValueLocator) Sensitive() InspectionExpectation {
	return valueSensitiveInspectionExpectation(locator)
}

// Available returns an expectation that checks the located value is effective and available in
// the public Reference observation.
func (locator ValueLocator) Available() ReferenceExpectation {
	value := valueReferenceAvailabilityExpectationValue{
		locator:  locator.state,
		expected: true,
	}
	return value
}

// Unavailable returns an expectation that checks the located value is unavailable in the public
// Reference observation.
func (locator ValueLocator) Unavailable() ReferenceExpectation {
	value := valueReferenceAvailabilityExpectationValue{
		locator:  locator.state,
		expected: false,
	}
	return value
}

// Visible returns an expectation that checks the located value is visible in the public
// Reference observation.
func (locator ValueLocator) Visible() ReferenceExpectation {
	value := valueReferenceVisibilityExpectationValue{
		locator:  locator.state,
		expected: true,
	}
	return value
}

// NotVisible returns an expectation that checks the located value is not visible in the public
// Reference observation.
func (locator ValueLocator) NotVisible() ReferenceExpectation {
	value := valueReferenceVisibilityExpectationValue{
		locator:  locator.state,
		expected: false,
	}
	return value
}

// State returns an expectation that checks the inspected value has the requested
// ResolutionState.
func (locator ValueLocator) State(state cli.ResolutionState) InspectionExpectation {
	return valueStateExpectation(locator, state)
}

// Inactive returns an expectation that checks the inspected value has ResolutionStateInactive.
func (locator ValueLocator) Inactive() InspectionExpectation {
	return valueStateExpectation(locator, cli.ResolutionStateInactive)
}

// StateIfObserved returns an expectation that allows the value to be unobserved, but if
// observed requires one of the supplied ResolutionState values. An empty allowed set is
// invalid expectation intent.
func (locator ValueLocator) StateIfObserved(states ...cli.ResolutionState) InspectionExpectation {
	return valueStateIfObservedExpectation(locator, states)
}

// Resolved returns an expectation that checks the inspected value has ResolutionStateResolved.
func (locator ValueLocator) Resolved() InspectionExpectation {
	return valueStateExpectation(locator, cli.ResolutionStateResolved)
}

// Failed returns an expectation that checks the inspected value has ResolutionStateFailed.
func (locator ValueLocator) Failed() InspectionExpectation {
	return valueStateExpectation(locator, cli.ResolutionStateFailed)
}

// Blocked returns an expectation that checks the inspected value has ResolutionStateBlocked.
func (locator ValueLocator) Blocked() InspectionExpectation {
	return valueStateExpectation(locator, cli.ResolutionStateBlocked)
}

// Text returns an expectation that checks the value has the exact unredacted canonical text
// representation. Expected and observed secret material is redacted from mismatch output.
func (locator ValueLocator) Text(text string) InspectionExpectation {
	return valueTextExpectation(locator, text)
}

// Tokens returns an expectation that checks the value has exactly the requested unredacted
// canonical token sequence. Expected and observed secret material is redacted from mismatch
// output.
func (locator ValueLocator) Tokens(tokens ...string) InspectionExpectation {
	return valueTokensExpectation(locator, tokens)
}

// Redacted returns an expectation that checks the value has a canonical representation marked
// redacted; this is distinct from the value's sensitivity flag.
func (locator ValueLocator) Redacted() InspectionExpectation {
	return valueRedactedExpectation(locator)
}

// ContributedBy returns an expectation that checks at least one final contributing candidate
// was produced from the requested ValueProducerKind.
func (locator ValueLocator) ContributedBy(source cli.ValueProducerKind) InspectionExpectation {
	return valueContributedByExpectation(locator, source)
}

// BlockedBy returns an expectation that checks the value's public blocking relationship
// includes the requested value.
func (locator ValueLocator) BlockedBy(blocker ValueLocator) InspectionExpectation {
	return valueBlockedByExpectation(locator, blocker)
}

// ProducersExactly returns an expectation that checks the located value has exactly the
// requested producer records in authoritative public order.
func (locator ValueLocator) ProducersExactly(producers ...ValueProducerLocator) InspectionExpectation {
	return valueProducersExactlyExpectation(locator, producers)
}

// CandidatesExactly returns an expectation that checks the located value has exactly the
// requested candidate records in authoritative public order.
func (locator ValueLocator) CandidatesExactly(candidates ...ValueCandidateLocator) InspectionExpectation {
	return valueCandidatesExactlyExpectation(locator, candidates)
}

// ContributorsExactly returns an expectation that checks the located value has exactly the
// requested contributing candidate records in authoritative public order.
func (locator ValueLocator) ContributorsExactly(candidates ...ValueCandidateLocator) InspectionExpectation {
	return valueContributorsExactlyExpectation(locator, candidates)
}

// ElementsExactly returns an expectation that checks the located value has exactly the
// requested resolved sequence elements in authoritative public order.
func (locator ValueLocator) ElementsExactly(elements ...ValueElementLocator) InspectionExpectation {
	return valueElementsExactlyExpectation(locator, elements)
}

// EntriesExactly returns an expectation that checks the located value has exactly the
// requested resolved keyed entries in authoritative public order.
func (locator ValueLocator) EntriesExactly(entries ...ValueEntryLocator) InspectionExpectation {
	return valueEntriesExactlyExpectation(locator, entries)
}

func (ValueLocator) tokenOwner() {}

func (ValueLocator) completionTargetSubject() {}

// Exists returns a reusable static declaration expectation that checks the located value static
// default is present in the public observation.
func (locator ValueStaticDefaultLocator) Exists() StaticDeclarationExpectation {
	value := staticDefaultExpectationValue{
		locator:  locator,
		property: staticDefaultExpectationExists,
		expected: true,
	}
	return value
}

// DoesNotExist returns a reusable static declaration expectation that checks the located value
// static default is absent from a complete relevant observation; incomplete evidence cannot
// manufacture absence.
func (locator ValueStaticDefaultLocator) DoesNotExist() StaticDeclarationExpectation {
	value := staticDefaultExpectationValue{
		locator:  locator,
		property: staticDefaultExpectationExists,
		expected: false,
	}
	return value
}

// Sensitive returns a reusable static declaration expectation that checks the located value
// static default is marked sensitive. Expectation failures preserve Commandment redaction boundaries.
func (locator ValueStaticDefaultLocator) Sensitive() StaticDeclarationExpectation {
	value := staticDefaultExpectationValue{
		locator:  locator,
		property: staticDefaultExpectationSensitive,
		expected: true,
	}
	return value
}

// Redacted returns a reusable static declaration expectation that checks the located value
// static default is marked redacted without exposing the protected representation.
func (locator ValueStaticDefaultLocator) Redacted() StaticDeclarationExpectation {
	value := staticDefaultExpectationValue{
		locator:  locator,
		property: staticDefaultExpectationRedacted,
		expected: true,
	}
	return value
}

// NoRepresentation returns a reusable static declaration expectation that checks the located
// value static default explicitly has no representation.
func (locator ValueStaticDefaultLocator) NoRepresentation() StaticDeclarationExpectation {
	value := staticDefaultExpectationValue{
		locator:  locator,
		property: staticDefaultExpectationHasRepresentation,
		expected: false,
	}
	return value
}

// Scalar returns a locator for the scalar literal representation of the located static default.
// Evaluation fails when the public default does not expose that representation.
func (locator ValueStaticDefaultLocator) Scalar() StaticValueLiteralLocator {
	staticValueLiteralLocator := StaticValueLiteralLocator{
		root: locator,
		mode: staticValueLiteralScalar,
	}
	return staticValueLiteralLocator
}

// Sequence returns a locator for the ordered sequence representation of the located static
// default. Evaluation fails when the public default does not expose that representation.
func (locator ValueStaticDefaultLocator) Sequence() ValueSequenceLiteralLocator {
	valueSequenceLiteralLocator := ValueSequenceLiteralLocator{
		root: locator,
	}
	return valueSequenceLiteralLocator
}

// Keyed returns a locator for the ordered keyed representation of the located static default.
// Evaluation fails when the public default does not expose that representation.
func (locator ValueStaticDefaultLocator) Keyed() ValueKeyedLiteralLocator {
	valueKeyedLiteralLocator := ValueKeyedLiteralLocator{
		root: locator,
	}
	return valueKeyedLiteralLocator
}

// ElementType returns a reusable static declaration expectation that checks the public element
// type exposed by the located static-default sequence.
func (locator ValueSequenceLiteralLocator) ElementType(expected cli.ValueType) StaticDeclarationExpectation {
	value := sequenceExpectationValue{
		locator:     locator,
		property:    sequenceExpectationElementType,
		elementType: expected,
	}
	return value
}

// ElementCount returns a reusable static declaration expectation that checks the element count
// exposed by the located static-default sequence.
func (locator ValueSequenceLiteralLocator) ElementCount(expected int) StaticDeclarationExpectation {
	value := sequenceExpectationValue{
		locator:  locator,
		property: sequenceExpectationElementCount,
		count:    expected,
	}
	return value
}

// ElementAt returns a reusable expectation that checks the public element at exposed by the
// located static-default sequence.
func (locator ValueSequenceLiteralLocator) ElementAt(index int) StaticValueLiteralLocator {
	staticValueLiteralLocator := StaticValueLiteralLocator{
		root:  locator.root,
		mode:  staticValueLiteralSequenceElement,
		index: index,
	}
	return staticValueLiteralLocator
}

// KeyType returns a reusable static declaration expectation that checks the public key type
// exposed by the located keyed static default.
func (locator ValueKeyedLiteralLocator) KeyType(expected cli.ValueType) StaticDeclarationExpectation {
	value := keyedExpectationValue{
		locator:   locator,
		property:  keyedExpectationKeyType,
		valueType: expected,
	}
	return value
}

// ValueType returns a reusable static declaration expectation that checks the public value type
// exposed by the located keyed static default.
func (locator ValueKeyedLiteralLocator) ValueType(expected cli.ValueType) StaticDeclarationExpectation {
	value := keyedExpectationValue{
		locator:   locator,
		property:  keyedExpectationValueType,
		valueType: expected,
	}
	return value
}

// EntryCount returns a reusable static declaration expectation that checks the entry count
// exposed by the located keyed static default.
func (locator ValueKeyedLiteralLocator) EntryCount(expected int) StaticDeclarationExpectation {
	value := keyedExpectationValue{
		locator:  locator,
		property: keyedExpectationEntryCount,
		count:    expected,
	}
	return value
}

// EntryAt returns a locator for the entry at the supplied public ordered index of the located
// keyed static default. It does not search for or silently choose a first match.
func (locator ValueKeyedLiteralLocator) EntryAt(index int) ValueKeyedEntryLocator {
	valueKeyedEntryLocator := ValueKeyedEntryLocator{
		root:  locator.root,
		index: index,
	}
	return valueKeyedEntryLocator
}

// Key returns a locator that follows the public key relationship from the located keyed
// static-default entry. The returned locator carries locator coordinates, not observed state.
func (locator ValueKeyedEntryLocator) Key() StaticValueLiteralLocator {
	staticValueLiteralLocator := StaticValueLiteralLocator{
		root:  locator.root,
		mode:  staticValueLiteralKey,
		index: locator.index,
	}
	return staticValueLiteralLocator
}

// Value returns a locator that follows the public value relationship from the located keyed
// static-default entry. The returned locator carries locator coordinates, not observed state.
func (locator ValueKeyedEntryLocator) Value() StaticValueLiteralLocator {
	staticValueLiteralLocator := StaticValueLiteralLocator{
		root:  locator.root,
		mode:  staticValueLiteralValue,
		index: locator.index,
	}
	return staticValueLiteralLocator
}

// Type returns a reusable static declaration expectation that checks the public type exposed by
// the located static-default literal.
func (locator StaticValueLiteralLocator) Type(expected cli.ValueType) StaticDeclarationExpectation {
	value := staticLiteralExpectationValue{
		locator:   locator,
		property:  staticLiteralExpectationType,
		valueType: expected,
	}
	return value
}

// Sensitive returns a reusable static declaration expectation that checks the located
// static-default literal is marked sensitive. Expectation failures preserve Commandment redaction
// boundaries.
func (locator StaticValueLiteralLocator) Sensitive() StaticDeclarationExpectation {
	value := staticLiteralExpectationValue{
		locator:  locator,
		property: staticLiteralExpectationSensitive,
	}
	return value
}

// Redacted returns a reusable static declaration expectation that checks the located
// static-default literal is marked redacted without exposing the protected representation.
func (locator StaticValueLiteralLocator) Redacted() StaticDeclarationExpectation {
	value := staticLiteralExpectationValue{
		locator:  locator,
		property: staticLiteralExpectationRedacted,
	}
	return value
}

// Text returns a static declaration expectation that checks safely observable literal text in
// the static default. Sensitive or redacted material remains governed by Commandment's public
// representation.
func (locator StaticValueLiteralLocator) Text(expected string) StaticDeclarationExpectation {
	value := staticLiteralExpectationValue{
		locator:  locator,
		property: staticLiteralExpectationText,
		text:     expected,
	}
	return value
}

// InputText returns an Inspection expectation that the located candidate has raw input and that its text exactly equals
// expected.
// Mismatch reporting suppresses observed raw text when the candidate or input is sensitive/redacted.
func (locator ValueCandidateLocator) InputText(expected string) InspectionExpectation {
	return valueCandidateInputTextExpectation(locator, expected)
}

// InputRedacted returns an Inspection expectation that the located candidate has raw input whose Redacted marker is
// true.
func (locator ValueCandidateLocator) InputRedacted() InspectionExpectation {
	return valueCandidateInputRedactedExpectation(locator)
}

// Redacted returns an Inspection expectation that the located candidate has a canonical representation whose Redacted
// marker is true.
func (locator ValueCandidateLocator) Redacted() InspectionExpectation {
	return valueCandidateRedactedExpectation(locator)
}

// DiagnosticsExactly returns an Inspection expectation that the diagnostics attached to the located subject are exactly
// expected in the subject's public DiagnosticIndexes order.
// This checks the returned attachment relationship; it does not reinterpret diagnostic identity from prose.
func (locator ValueCandidateLocator) DiagnosticsExactly(expected ...DiagnosticLocator) InspectionExpectation {
	expectation := candidateDiagnosticsExactlyExpectation{
		locator:  locator.state,
		expected: slices.Clone(expected),
	}
	return expectation
}

// ElementAt returns the candidate element at index in its authoritative element order.
func (locator ValueCandidateLocator) ElementAt(index int) ValueElementLocator {
	return valueElementIndexLocator(locator, index)
}

// EntryAt returns the candidate entry at index in its authoritative keyed-entry order.
func (locator ValueCandidateLocator) EntryAt(index int) ValueEntryLocator {
	return valueEntryIndexLocator(locator, index)
}

// Disposition returns an expectation that checks the result-local candidate has the requested
// ValueCandidateDisposition.
func (locator ValueCandidateLocator) Disposition(disposition cli.ValueCandidateDisposition) InspectionExpectation {
	return valueCandidateDispositionExpectation(locator, disposition)
}

// Contribution returns an expectation that checks the candidate has the requested declared
// contribution granularity; it is not inferred from Go container shape.
func (locator ValueCandidateLocator) Contribution(contribution cli.ValueCandidateContribution) InspectionExpectation {
	return valueCandidateContributionExpectation(locator, contribution)
}

// Sensitive returns an expectation that checks the candidate is sensitive; mismatch reporting
// redacts sensitive expected and observed candidate properties.
func (locator ValueCandidateLocator) Sensitive() InspectionExpectation {
	return valueCandidateSensitiveExpectation(locator)
}

// ProducedBy returns an expectation that checks the candidate was produced by the requested
// producer record.
func (locator ValueCandidateLocator) ProducedBy(producer ValueProducerLocator) InspectionExpectation {
	return valueCandidateProducedByExpectation(locator, producer)
}

// Declaration returns an expectation that checks the candidate carries the exact public
// declaration identity identified by the supplied heterogeneous declaration locator.
func (locator ValueCandidateLocator) Declaration(declaration DeclarationIdentityLocator) InspectionExpectation {
	return valueCandidateDeclarationExpectation(locator, declaration)
}

// OriginsExactly returns an expectation that checks the candidate's public input origins are
// exactly the requested origins in authoritative order.
func (locator ValueCandidateLocator) OriginsExactly(origins ...InputOriginLocator) InspectionExpectation {
	return valueCandidateOriginsExactlyExpectation(locator, origins)
}

// Suppression returns an expectation that checks the candidate carries the requested
// ValueCandidateSuppression.
func (locator ValueCandidateLocator) Suppression(suppression cli.ValueCandidateSuppression) InspectionExpectation {
	return valueCandidateSuppressionExpectation(locator, suppression)
}

// Text returns an expectation that checks the candidate has the exact unredacted text
// representation; secret mismatch data remains redacted.
func (locator ValueCandidateLocator) Text(text string) InspectionExpectation {
	return valueCandidateTextExpectation(locator, text)
}

// Tokens returns an expectation that checks the candidate has exactly the requested unredacted
// token representation; secret mismatch data remains redacted.
func (locator ValueCandidateLocator) Tokens(tokens ...string) InspectionExpectation {
	return valueCandidateTokensExpectation(locator, tokens)
}

// DomainEntry returns an expectation that checks the candidate is related to the exact
// requested value-domain entry.
func (locator ValueCandidateLocator) DomainEntry(entry ValueDomainEntryLocator) InspectionExpectation {
	return valueCandidateDomainEntryExpectation(locator, entry)
}

// SuppressedByExactly returns an expectation that checks the candidate's ordered SuppressedBy
// relationship contains exactly the requested result-local candidates.
func (locator ValueCandidateLocator) SuppressedByExactly(candidates ...ValueCandidateLocator) InspectionExpectation {
	return valueCandidateSuppressedByExactlyExpectation(locator, candidates)
}

// ConflictsWithExactly returns an expectation that checks the candidate's ordered conflict
// relationship contains exactly the requested result-local candidates.
func (locator ValueCandidateLocator) ConflictsWithExactly(candidates ...ValueCandidateLocator) InspectionExpectation {
	return valueCandidateConflictsWithExactlyExpectation(locator, candidates)
}

// ElementCount returns an expectation that checks the candidate publishes exactly the
// requested number of sequence-element contributions.
func (locator ValueCandidateLocator) ElementCount(count int) InspectionExpectation {
	return valueCandidateElementCountExpectation(locator, count)
}

// EntryCount returns an expectation that checks the candidate publishes exactly the requested
// number of keyed-entry contributions.
func (locator ValueCandidateLocator) EntryCount(count int) InspectionExpectation {
	return valueCandidateEntryCountExpectation(locator, count)
}

// InputText returns an Inspection expectation that the located value component has raw input text exactly equal to
// expected.
// Mismatch reporting suppresses observed text when the component or input is sensitive/redacted.
func (locator ValueComponentLocator) InputText(expected string) InspectionExpectation {
	return valueComponentInputTextExpectation(locator, expected)
}

// Text returns an expectation that checks the component has the exact unredacted text
// representation; secret mismatch data remains redacted.
func (locator ValueComponentLocator) Text(text string) InspectionExpectation {
	return valueComponentTextExpectation(locator, text)
}

// Tokens returns an expectation that checks the component has exactly the requested unredacted
// token representation; secret mismatch data remains redacted.
func (locator ValueComponentLocator) Tokens(tokens ...string) InspectionExpectation {
	return valueComponentTokensExpectation(locator, tokens)
}

// Redacted returns an expectation that checks the component representation is redacted,
// distinct from its sensitivity flag.
func (locator ValueComponentLocator) Redacted() InspectionExpectation {
	return valueComponentRedactedExpectation(locator)
}

// DomainEntry returns an expectation that checks the component is related to the exact
// requested value-domain entry.
func (locator ValueComponentLocator) DomainEntry(entry ValueDomainEntryLocator) InspectionExpectation {
	return valueComponentDomainEntryExpectation(locator, entry)
}

// OriginsExactly returns an expectation that checks the component's public input origins are
// exactly the requested origins in authoritative order.
func (locator ValueComponentLocator) OriginsExactly(origins ...InputOriginLocator) InspectionExpectation {
	return valueComponentOriginsExactlyExpectation(locator, origins)
}

// Sensitive returns an expectation that checks the component is sensitive; mismatch reporting
// preserves redaction.
func (locator ValueComponentLocator) Sensitive() InspectionExpectation {
	return valueComponentSensitiveExpectation(locator)
}

// Key follows the public relationship to the keyed entry's key component.
func (locator ValueEntryLocator) Key() ValueComponentLocator { return valueEntryKeyLocator(locator) }

// Value follows the public relationship to the keyed entry's value component.
func (locator ValueEntryLocator) Value() ValueComponentLocator {
	return valueEntryValueLocator(locator)
}

// Kind returns an expectation that checks the producer has the requested ValueProducerKind.
func (locator ValueProducerLocator) Kind(source cli.ValueProducerKind) InspectionExpectation {
	return valueProducerKindExpectation(locator, source)
}

// Tier returns an expectation that checks the producer occupies the requested ResolutionTier.
func (locator ValueProducerLocator) Tier(tier cli.ResolutionTier) InspectionExpectation {
	return valueProducerTierExpectation(locator, tier)
}

// Outcome returns an expectation that checks the producer has the requested
// ValueProducerOutcome.
func (locator ValueProducerLocator) Outcome(outcome cli.ValueProducerOutcome) InspectionExpectation {
	return valueProducerOutcomeExpectation(locator, outcome)
}

// Sensitive returns an expectation that checks the producer is sensitive; mismatch reporting
// preserves redaction.
func (locator ValueProducerLocator) Sensitive() InspectionExpectation {
	return valueProducerSensitiveExpectation(locator)
}

// BlockedByExactly returns an expectation that checks the producer's blocking value IDs are
// exactly the requested values in authoritative order.
func (locator ValueProducerLocator) BlockedByExactly(values ...ValueLocator) InspectionExpectation {
	return valueProducerBlockedByExactlyExpectation(locator, values)
}

// OriginsExactly returns an expectation that checks the producer's public input-origin
// relationship is exactly the requested origins in authoritative order.
func (locator ValueProducerLocator) OriginsExactly(origins ...InputOriginLocator) InspectionExpectation {
	return valueProducerOriginsExactlyExpectation(locator, origins)
}

// DependenciesExactly returns an expectation that checks the producer's value dependencies are
// exactly the requested values in authoritative order.
func (locator ValueProducerLocator) DependenciesExactly(values ...ValueLocator) InspectionExpectation {
	return valueProducerDependenciesExactlyExpectation(locator, values)
}

type commandFactsExpectation struct {
	command *commandLocatorState
	items   []*factLocatorState
}

type factLocatorMode uint8

const (
	factLocatorInvalid factLocatorMode = iota
	factLocatorID
	factLocatorCommandIndex
)

type factLocatorState struct {
	mode factLocatorMode
	id   cli.ValueID

	command *commandLocatorState
	index   int
}

type factAvailabilityExpectation struct {
	locator  *factLocatorState
	expected bool
}

type valueReferenceAvailabilityExpectationValue struct {
	locator  *valueLocatorState
	expected bool
}

type valueReferenceVisibilityExpectationValue struct {
	locator  *valueLocatorState
	expected bool
}

type staticValueLiteralMode uint8

const (
	staticValueLiteralInvalid staticValueLiteralMode = iota
	staticValueLiteralScalar
	staticValueLiteralSequenceElement
	staticValueLiteralKey
	staticValueLiteralValue
)

type staticDefaultExpectationProperty uint8

const (
	staticDefaultExpectationInvalid staticDefaultExpectationProperty = iota
	staticDefaultExpectationExists
	staticDefaultExpectationSensitive
	staticDefaultExpectationRedacted
	staticDefaultExpectationHasRepresentation
)

type staticDefaultExpectationValue struct {
	locator  ValueStaticDefaultLocator
	property staticDefaultExpectationProperty
	expected bool
}

type sequenceExpectationProperty uint8

const (
	sequenceExpectationInvalid sequenceExpectationProperty = iota
	sequenceExpectationElementType
	sequenceExpectationElementCount
)

type sequenceExpectationValue struct {
	locator     ValueSequenceLiteralLocator
	property    sequenceExpectationProperty
	elementType cli.ValueType
	count       int
}

type keyedExpectationProperty uint8

const (
	keyedExpectationInvalid keyedExpectationProperty = iota
	keyedExpectationKeyType
	keyedExpectationValueType
	keyedExpectationEntryCount
)

type keyedExpectationValue struct {
	locator   ValueKeyedLiteralLocator
	property  keyedExpectationProperty
	valueType cli.ValueType
	count     int
}

type staticLiteralExpectationProperty uint8

const (
	staticLiteralExpectationInvalid staticLiteralExpectationProperty = iota
	staticLiteralExpectationType
	staticLiteralExpectationSensitive
	staticLiteralExpectationRedacted
	staticLiteralExpectationText
)

type staticLiteralExpectationValue struct {
	locator   StaticValueLiteralLocator
	property  staticLiteralExpectationProperty
	valueType cli.ValueType
	text      string
}

type factDependenciesExactlyExpectation struct {
	locator      *factLocatorState
	dependencies []*valueLocatorState
}

type factExistenceExpectation struct {
	locator  *factLocatorState
	expected bool
}

type valueExistenceExpectationValue struct {
	locator  *valueLocatorState
	expected bool
}

type factHasResolverExpectationValue struct{ locator *factLocatorState }

type (
	valueRequiredStaticExpectationValue  struct{ locator *valueLocatorState }
	valueSensitiveStaticExpectationValue struct{ locator *valueLocatorState }
	valueOriginStaticExpectationValue    struct {
		locator  *valueLocatorState
		expected cli.DeclarationOrigin
	}
)

type valueCorePropertyKind uint8

const (
	valueCorePropertyInvalid valueCorePropertyKind = iota
	valueCorePropertyDestinationKind
	valueCorePropertyType
	valueCorePropertyNoType
)

type valueCorePropertyExpectation struct {
	locator         *valueLocatorState
	kind            valueCorePropertyKind
	destinationKind cli.ValueDestinationKind
	valueType       cli.ValueType
}

type valueDependenciesExpectation struct {
	locator      *valueLocatorState
	dependencies []*valueLocatorState
}

type valueLocatorMode uint8

const (
	valueLocatorInvalid valueLocatorMode = iota
	valueLocatorID
	valueLocatorCommandIndex
	valueLocatorFlag
	valueLocatorArgumentTerm
	valueLocatorEnvironment
	valueLocatorFact
)

type valueLocatorState struct {
	mode valueLocatorMode
	id   cli.ValueID

	command     *commandLocatorState
	flag        *flagLocatorState
	argument    *argumentTermLocatorState
	environment *environmentInputLocatorState
	fact        *factLocatorState
	index       int
}

type valueProducerLocatorState struct {
	value *valueLocatorState
	index int
}

type valueCandidateLocatorMode uint8

const (
	valueCandidateLocatorInvalid valueCandidateLocatorMode = iota
	valueCandidateLocatorID
	valueCandidateLocatorCandidateIndex
	valueCandidateLocatorContributorIndex
)

type valueCandidateLocatorState struct {
	mode valueCandidateLocatorMode
	id   cli.ValueCandidateID

	value *valueLocatorState
	index int
}

type valueEntryLocatorState struct {
	candidate *valueCandidateLocatorState
	index     int
}

type valueComponentLocatorMode uint8

const (
	valueComponentLocatorInvalid valueComponentLocatorMode = iota
	valueComponentLocatorElement
	valueComponentLocatorEntryKey
	valueComponentLocatorEntryValue
)

type valueComponentLocatorState struct {
	mode valueComponentLocatorMode

	candidate *valueCandidateLocatorState
	entry     *valueEntryLocatorState
	index     int
}

type valueResolution struct {
	status locatorResolutionStatus
	value  cli.ValueInspection
	label  evaluationField
	actual evaluationField
}

type producerResolution struct {
	status locatorResolutionStatus
	value  cli.ValueProducerInspection
	index  int
	label  evaluationField
	actual evaluationField
}

type candidateResolution struct {
	status locatorResolutionStatus
	value  cli.ValueCandidateInspection
	id     cli.ValueCandidateID
	label  evaluationField
	actual evaluationField
}

type componentResolution struct {
	status locatorResolutionStatus
	value  cli.ValueComponentInspection
	label  evaluationField
	actual evaluationField
}

type domainResolution struct {
	status locatorResolutionStatus
	value  cli.ValueDomainInspection
	index  int
	label  evaluationField
	actual evaluationField
}

type valuePropertyKind uint8

const (
	valuePropertyInvalid valuePropertyKind = iota
	valuePropertyState
	valuePropertySensitive
	valuePropertyText
	valuePropertyTokens
	valuePropertyRedacted
	valuePropertyContributedBy
)

type valuePropertyExpectation struct {
	locator      *valueLocatorState
	kind         valuePropertyKind
	state        cli.ResolutionState
	text         string
	tokens       []string
	producerKind cli.ValueProducerKind
}

type valueObservationExpectationValue struct {
	locator  *valueLocatorState
	observed bool
}

type valueBlockedByExpectationValue struct {
	value   *valueLocatorState
	blocker *valueLocatorState
}

type valueProducerPropertyKind uint8

const (
	valueProducerPropertyInvalid valueProducerPropertyKind = iota
	valueProducerPropertyProducerKind
	valueProducerPropertyTier
	valueProducerPropertyOutcome
	valueProducerPropertySensitive
	valueProducerPropertyBlockedBy
	valueProducerPropertyDependencies
	valueProducerPropertyOrigins
)

type producerPropertyExpectation struct {
	locator      *valueProducerLocatorState
	kind         valueProducerPropertyKind
	producerKind cli.ValueProducerKind
	tier         cli.ResolutionTier
	outcome      cli.ValueProducerOutcome
	values       []*valueLocatorState
	origins      []cli.InputOriginID
	originsValid bool
}

type valueCandidatePropertyKind uint8

const (
	valueCandidatePropertyInvalid valueCandidatePropertyKind = iota
	valueCandidatePropertyDisposition
	valueCandidatePropertyContribution
	valueCandidatePropertySensitive
	valueCandidatePropertyOrigins
	valueCandidatePropertySuppression
	valueCandidatePropertyText
	valueCandidatePropertyTokens
	valueCandidatePropertyElementCount
	valueCandidatePropertyEntryCount
	valueCandidatePropertySuppressedBy
	valueCandidatePropertyConflictsWith
	valueCandidatePropertyInputText
	valueCandidatePropertyInputRedacted
	valueCandidatePropertyRedacted
)

type candidatePropertyExpectation struct {
	locator      *valueCandidateLocatorState
	kind         valueCandidatePropertyKind
	disposition  cli.ValueCandidateDisposition
	contribution cli.ValueCandidateContribution
	suppression  cli.ValueCandidateSuppression
	text         string
	tokens       []string
	origins      []cli.InputOriginID
	originsValid bool
	count        int
	candidates   []*valueCandidateLocatorState
}

type valueCandidateProducedByExpectationValue struct {
	candidate *valueCandidateLocatorState
	producer  *valueProducerLocatorState
}

type valueCandidateDeclarationExpectationValue struct {
	candidate   *valueCandidateLocatorState
	declaration DeclarationIdentityLocator
}

type valueComponentPropertyKind uint8

const (
	valueComponentPropertyInvalid valueComponentPropertyKind = iota
	valueComponentPropertyText
	valueComponentPropertyTokens
	valueComponentPropertyRedacted
	valueComponentPropertyOrigins
	valueComponentPropertySensitive
	valueComponentPropertyInputText
)

type componentPropertyExpectation struct {
	locator      *valueComponentLocatorState
	kind         valueComponentPropertyKind
	text         string
	tokens       []string
	origins      []cli.InputOriginID
	originsValid bool
}

type valueCollectionKind uint8

const (
	valueCollectionInvalid valueCollectionKind = iota
	valueCollectionProducers
	valueCollectionCandidates
	valueCollectionContributors
	valueCollectionElements
	valueCollectionEntries
)

type valueCollectionExpectation struct {
	locator    *valueLocatorState
	kind       valueCollectionKind
	producers  []*valueProducerLocatorState
	candidates []*valueCandidateLocatorState
	elements   []ValueElementLocator
	entries    []ValueEntryLocator
}

type domainPropertyExpectation struct {
	locator *valueDomainLocatorState
	kind    valueDomainPropertyKind
	outcome cli.ValueDomainOutcome
	values  []*valueLocatorState
}

type valueCandidateDomainEntryExpectationValue struct {
	candidate *valueCandidateLocatorState
	entry     *valueDomainEntryLocatorState
}

type componentDomainEntryExpectation struct {
	locator *valueComponentLocatorState
	entry   *valueDomainEntryLocatorState
}

func factLocatorStates(values []FactLocator) []*factLocatorState {
	states := make([]*factLocatorState, len(values))
	for index, value := range values {
		states[index] = value.state
	}
	return states
}

func valueLocatorStates(values []ValueLocator) []*valueLocatorState {
	states := make([]*valueLocatorState, len(values))
	for index, value := range values {
		states[index] = value.state
	}
	return states
}

func commandFactAtLocator(parent CommandLocator, index int) FactLocator {
	state := &factLocatorState{
		mode:    factLocatorCommandIndex,
		command: parent.state,
		index:   index,
	}
	locator := FactLocator{
		state: state,
	}
	return locator
}

func environmentValueLocator(parent EnvironmentInputLocator) ValueLocator {
	state := &valueLocatorState{
		mode:        valueLocatorEnvironment,
		environment: parent.state,
	}
	locator := ValueLocator{
		state: state,
	}
	return locator
}

func factValueLocator(parent FactLocator) ValueLocator {
	state := &valueLocatorState{
		mode: valueLocatorFact,
		fact: parent.state,
	}
	locator := ValueLocator{
		state: state,
	}
	return locator
}

func cloneValueLocatorStates(values []ValueLocator) []*valueLocatorState {
	states := make([]*valueLocatorState, len(values))
	for i, value := range values {
		states[i] = value.state
	}
	return states
}

func expectedTypedValueIDsSchema(states []*valueLocatorState, schema *cli.Schema) ([]cli.ValueID, locatorResolutionStatus, evaluationField) {
	ids := make([]cli.ValueID, len(states))
	for index, state := range states {
		if state != nil && state.mode == valueLocatorID && state.id != 0 {
			ids[index] = state.id
			continue
		}
		resolution := resolveSchemaValueLocator(state, schema)
		if resolution.status != locatorResolutionFound {
			return nil, resolution.status, withDetail(fmt.Sprintf("value %d", index), resolution.detail)
		}
		ids[index] = resolution.value.ID
	}
	return ids, locatorResolutionFound, evaluationText("")
}

func expectedTypedValueIDsReference(states []*valueLocatorState, reference cli.Reference) ([]cli.ValueID, locatorResolutionStatus, evaluationField) {
	ids := make([]cli.ValueID, len(states))
	for index, state := range states {
		if state != nil && state.mode == valueLocatorID && state.id != 0 {
			ids[index] = state.id
			continue
		}
		resolution := resolveReferenceValueLocator(state, reference)
		if resolution.status != locatorResolutionFound {
			return nil, resolution.status, withDetail(fmt.Sprintf("value %d", index), resolution.detail)
		}
		ids[index] = resolution.value.ID
	}
	return ids, locatorResolutionFound, evaluationText("")
}

func expectedTypedValueIDsInspection(states []*valueLocatorState, inspection cli.Inspection) ([]cli.ValueID, locatorResolutionStatus, evaluationField) {
	ids := make([]cli.ValueID, len(states))
	for index, state := range states {
		if state != nil && state.mode == valueLocatorID && state.id != 0 {
			ids[index] = state.id
			continue
		}
		resolution := resolveInspectionValueLocator(state, inspection)
		if resolution.status != locatorResolutionFound {
			return nil, resolution.status, withDetail(fmt.Sprintf("value %d", index), resolution.detail)
		}
		ids[index] = resolution.value.ID
	}
	return ids, locatorResolutionFound, evaluationText("")
}

func factLocatorLabel(state *factLocatorState) evaluationField {
	if state == nil {
		return semanticSubjectFact
	}
	switch state.mode {
	case factLocatorID:
		return evaluationText(fmt.Sprintf("fact value id %d", state.id))
	case factLocatorCommandIndex:
		return qualifySubject(commandLocatorLabel(state.command), evaluationText(fmt.Sprintf("fact %d", state.index)))
	default:
		return semanticSubjectFact
	}
}

func resolveSchemaFactID(state *factLocatorState, schema *cli.Schema) (cli.ValueID, locatorResolutionStatus, evaluationField) {
	if schema == nil || state == nil || state.mode == factLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectFact, observationInvalidLocator)
	}
	switch state.mode {
	case factLocatorID:
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectFact, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	case factLocatorCommandIndex:
		command := resolveSchemaCommandLocator(state.command, schema)
		if command.status != locatorResolutionFound {
			return 0, command.status, command.detail
		}
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectFact, observationInvalidLocator)
		}
		if state.index >= len(command.value.Facts) {
			return 0, locatorResolutionAbsent, observed(semanticSubjectFact, observationAbsent)
		}
		return command.value.Facts[state.index], locatorResolutionFound, evaluationText("")
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectFact, observationInvalidLocator)
	}
}

func resolveReferenceFactID(state *factLocatorState, reference cli.Reference) (cli.ValueID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == factLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectFact, observationInvalidLocator)
	}
	switch state.mode {
	case factLocatorID:
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectFact, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	case factLocatorCommandIndex:
		command := resolveReferenceCommandLocator(state.command, reference)
		if command.status != locatorResolutionFound {
			return 0, command.status, command.detail
		}
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectFact, observationInvalidLocator)
		}
		if state.index >= len(command.value.Facts) {
			return 0, locatorResolutionAbsent, observed(semanticSubjectFact, observationAbsent)
		}
		return command.value.Facts[state.index], locatorResolutionFound, evaluationText("")
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectFact, observationInvalidLocator)
	}
}

func resolveInspectionFactID(state *factLocatorState, inspection cli.Inspection) (cli.ValueID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == factLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectFact, observationInvalidLocator)
	}
	switch state.mode {
	case factLocatorID:
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectFact, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	case factLocatorCommandIndex:
		command := resolveInspectionCommandLocator(state.command, inspection)
		if command.status != locatorResolutionFound {
			return 0, command.status, command.detail
		}
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectFact, observationInvalidLocator)
		}
		if state.index >= len(command.value.Facts) {
			if command.value.NamespaceCompleteness != cli.CommandNamespaceComplete || inspection.Truncated {
				return 0, locatorResolutionNotProvable, evaluationText("fact order is incomplete")
			}
			return 0, locatorResolutionAbsent, observed(semanticSubjectFact, observationAbsent)
		}
		return command.value.Facts[state.index], locatorResolutionFound, evaluationText("")
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectFact, observationInvalidLocator)
	}
}

func resolveSchemaFactLocator(state *factLocatorState, schema *cli.Schema) locatorResolution[cli.FactSchema] {
	var resolution locatorResolution[cli.FactSchema]
	label := factLocatorLabel(state)
	if schema == nil || state == nil || state.mode == factLocatorInvalid {
		resolution = locatorResolution[cli.FactSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectFact, observationInvalidLocator),
		}
		return resolution
	}
	id := state.id
	if state.mode == factLocatorCommandIndex {
		command := resolveSchemaCommandLocator(state.command, schema)
		if command.status != locatorResolutionFound {
			resolution = locatorResolution[cli.FactSchema]{
				status:  command.status,
				subject: label,
				detail:  command.detail,
			}
			return resolution
		}
		if state.index < 0 {
			resolution = locatorResolution[cli.FactSchema]{
				status:  locatorResolutionInvalid,
				subject: label,
				detail:  observed(semanticSubjectFact, observationInvalidLocator),
			}
			return resolution
		}
		if state.index >= len(command.value.Facts) {
			resolution = locatorResolution[cli.FactSchema]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectFact, observationAbsent),
			}
			return resolution
		}
		id = command.value.Facts[state.index]
	}
	value, ok := schema.Facts[id]
	if !ok {
		resolution = locatorResolution[cli.FactSchema]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectFact, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.FactSchema]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

func resolveReferenceFactLocator(state *factLocatorState, reference cli.Reference) locatorResolution[cli.FactReference] {
	var resolution locatorResolution[cli.FactReference]
	label := factLocatorLabel(state)
	if state == nil || state.mode == factLocatorInvalid {
		resolution = locatorResolution[cli.FactReference]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectFact, observationInvalidLocator),
		}
		return resolution
	}
	id := state.id
	if state.mode == factLocatorCommandIndex {
		command := resolveReferenceCommandLocator(state.command, reference)
		if command.status != locatorResolutionFound {
			resolution = locatorResolution[cli.FactReference]{
				status:  command.status,
				subject: label,
				detail:  command.detail,
			}
			return resolution
		}
		if state.index < 0 {
			resolution = locatorResolution[cli.FactReference]{
				status:  locatorResolutionInvalid,
				subject: label,
				detail:  observed(semanticSubjectFact, observationInvalidLocator),
			}
			return resolution
		}
		if state.index >= len(command.value.Facts) {
			resolution = locatorResolution[cli.FactReference]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectFact, observationAbsent),
			}
			return resolution
		}
		id = command.value.Facts[state.index]
	}
	value, ok := reference.Facts[id]
	if !ok {
		resolution = locatorResolution[cli.FactReference]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectFact, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.FactReference]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

func resolveInspectionFactLocator(state *factLocatorState, inspection cli.Inspection) locatorResolution[cli.FactInspection] {
	var resolution locatorResolution[cli.FactInspection]
	label := factLocatorLabel(state)
	if state == nil || state.mode == factLocatorInvalid {
		resolution = locatorResolution[cli.FactInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectFact, observationInvalidLocator),
		}
		return resolution
	}

	var id cli.ValueID
	switch state.mode {
	case factLocatorID:
		id = state.id
	case factLocatorCommandIndex:
		command := resolveInspectionCommandLocator(state.command, inspection)
		if command.status != locatorResolutionFound {
			resolution = locatorResolution[cli.FactInspection]{
				status:  command.status,
				subject: label,
				detail:  command.detail,
			}
			return resolution
		}
		if state.index < 0 {
			resolution = locatorResolution[cli.FactInspection]{
				status:  locatorResolutionInvalid,
				subject: label,
				detail:  observed(semanticSubjectFact, observationInvalidLocator),
			}
			return resolution
		}
		if state.index >= len(command.value.Facts) {
			if command.value.NamespaceCompleteness != cli.CommandNamespaceComplete || inspection.Truncated {
				resolution = locatorResolution[cli.FactInspection]{
					status:  locatorResolutionNotProvable,
					subject: label,
					detail:  evaluationText("fact order is incomplete"),
				}
				return resolution
			}
			resolution = locatorResolution[cli.FactInspection]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectFact, observationAbsent),
			}
			return resolution
		}
		id = command.value.Facts[state.index]
	default:
		resolution = locatorResolution[cli.FactInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectFact, observationInvalidLocator),
		}
		return resolution
	}

	value, ok := inspection.Facts[id]
	if ok {
		resolution = locatorResolution[cli.FactInspection]{
			status:  locatorResolutionFound,
			value:   value,
			subject: label,
		}
		return resolution
	}
	if inspection.Truncated {
		resolution = locatorResolution[cli.FactInspection]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  observed(semanticSubjectInspection, observationTruncated),
		}
		return resolution
	}
	resolution = locatorResolution[cli.FactInspection]{
		status:  locatorResolutionAbsent,
		subject: label,
		detail:  observed(semanticSubjectFact, observationAbsent),
	}
	return resolution
}

func valueLocatorLabel(state *valueLocatorState) evaluationField {
	if state == nil {
		return semanticSubjectValue
	}
	switch state.mode {
	case valueLocatorID:
		return evaluationText(fmt.Sprintf("value id %d", state.id))
	case valueLocatorCommandIndex:
		return qualifySubject(commandLocatorLabel(state.command), evaluationText(fmt.Sprintf("value %d", state.index)))
	case valueLocatorFlag:
		return qualifySubject(flagLocatorLabel(state.flag), evaluationText("value"))
	case valueLocatorArgumentTerm:
		return qualifySubject(argumentTermLocatorLabel(state.argument), evaluationText("value"))
	case valueLocatorEnvironment:
		return qualifySubject(environmentInputLocatorLabel(state.environment), evaluationText("value"))
	case valueLocatorFact:
		return qualifySubject(factLocatorLabel(state.fact), evaluationText("value"))
	default:
		return semanticSubjectValue
	}
}

func resolveSchemaValueLocator(state *valueLocatorState, schema *cli.Schema) locatorResolution[cli.ValueSchema] {
	var resolution locatorResolution[cli.ValueSchema]
	label := valueLocatorLabel(state)
	id, status, detail := resolveSchemaValueID(state, schema)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.ValueSchema]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	value, ok := schema.Values[id]
	if !ok {
		resolution = locatorResolution[cli.ValueSchema]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectValue, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ValueSchema]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

func resolveReferenceValueLocator(state *valueLocatorState, reference cli.Reference) locatorResolution[cli.ValueReference] {
	var resolution locatorResolution[cli.ValueReference]
	label := valueLocatorLabel(state)
	id, status, detail := resolveReferenceValueID(state, reference)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.ValueReference]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	value, ok := reference.Values[id]
	if !ok {
		resolution = locatorResolution[cli.ValueReference]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectValue, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ValueReference]{
		status:  locatorResolutionFound,
		value:   value,
		subject: label,
	}
	return resolution
}

func resolveInspectionValueLocator(state *valueLocatorState, inspection cli.Inspection) locatorResolution[cli.ValueInspection] {
	var resolution locatorResolution[cli.ValueInspection]
	label := valueLocatorLabel(state)
	id, status, detail := resolveInspectionValueID(state, inspection)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.ValueInspection]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	value, ok := inspection.Values[id]
	if ok {
		resolution = locatorResolution[cli.ValueInspection]{
			status:  locatorResolutionFound,
			value:   value,
			subject: label,
		}
		return resolution
	}
	if inspection.Truncated {
		resolution = locatorResolution[cli.ValueInspection]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  observed(semanticSubjectInspection, observationTruncated),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ValueInspection]{
		status:  locatorResolutionAbsent,
		subject: label,
		detail:  observed(semanticSubjectValue, observationAbsent),
	}
	return resolution
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func resolveSchemaValueID(state *valueLocatorState, schema *cli.Schema) (cli.ValueID, locatorResolutionStatus, evaluationField) {
	if schema == nil || state == nil || state.mode == valueLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectValue, observationInvalidLocator)
	}
	switch state.mode {
	case valueLocatorID:
		return state.id, locatorResolutionFound, evaluationText("")
	case valueLocatorCommandIndex:
		command := resolveSchemaCommandLocator(state.command, schema)
		if command.status != locatorResolutionFound {
			return 0, command.status, command.detail
		}
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectValue, observationInvalidLocator)
		}
		if state.index >= len(command.value.Values) {
			return 0, locatorResolutionAbsent, observed(semanticSubjectValue, observationAbsent)
		}
		return command.value.Values[state.index], locatorResolutionFound, evaluationText("")
	case valueLocatorFlag:
		flag := resolveSchemaFlagLocator(state.flag, schema)
		if flag.status != locatorResolutionFound {
			return 0, flag.status, flag.detail
		}
		return flag.value.Destination, locatorResolutionFound, evaluationText("")
	case valueLocatorArgumentTerm:
		term := resolveSchemaArgumentTermLocator(state.argument, schema)
		if term.status != locatorResolutionFound {
			return 0, term.status, term.detail
		}
		return term.value.Destination, locatorResolutionFound, evaluationText("")
	case valueLocatorEnvironment:
		environment := resolveSchemaEnvironmentInputLocator(state.environment, schema)
		if environment.status != locatorResolutionFound {
			return 0, environment.status, environment.detail
		}
		return environment.value.Destination, locatorResolutionFound, evaluationText("")
	case valueLocatorFact:
		return resolveSchemaFactID(state.fact, schema)
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectValue, observationInvalidLocator)
	}
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func resolveReferenceValueID(state *valueLocatorState, reference cli.Reference) (cli.ValueID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == valueLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectValue, observationInvalidLocator)
	}
	switch state.mode {
	case valueLocatorID:
		return state.id, locatorResolutionFound, evaluationText("")
	case valueLocatorCommandIndex:
		command := resolveReferenceCommandLocator(state.command, reference)
		if command.status != locatorResolutionFound {
			return 0, command.status, command.detail
		}
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectValue, observationInvalidLocator)
		}
		if state.index >= len(command.value.Values) {
			return 0, locatorResolutionAbsent, observed(semanticSubjectValue, observationAbsent)
		}
		return command.value.Values[state.index], locatorResolutionFound, evaluationText("")
	case valueLocatorFlag:
		flag := resolveReferenceFlagLocator(state.flag, reference)
		if flag.status != locatorResolutionFound {
			return 0, flag.status, flag.detail
		}
		return flag.value.Destination, locatorResolutionFound, evaluationText("")
	case valueLocatorArgumentTerm:
		term := resolveReferenceArgumentTermLocator(state.argument, reference)
		if term.status != locatorResolutionFound {
			return 0, term.status, term.detail
		}
		return term.value.Destination, locatorResolutionFound, evaluationText("")
	case valueLocatorEnvironment:
		environment := resolveReferenceEnvironmentInputLocator(state.environment, reference)
		if environment.status != locatorResolutionFound {
			return 0, environment.status, environment.detail
		}
		return environment.value.Destination, locatorResolutionFound, evaluationText("")
	case valueLocatorFact:
		return resolveReferenceFactID(state.fact, reference)
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectValue, observationInvalidLocator)
	}
}

func resolveInspectionValueID(state *valueLocatorState, inspection cli.Inspection) (cli.ValueID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == valueLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectValue, observationInvalidLocator)
	}
	switch state.mode {
	case valueLocatorID:
		return state.id, locatorResolutionFound, evaluationText("")
	case valueLocatorCommandIndex:
		command := resolveInspectionCommandLocator(state.command, inspection)
		if command.status != locatorResolutionFound {
			return 0, command.status, command.detail
		}
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectValue, observationInvalidLocator)
		}
		if state.index >= len(command.value.Values) {
			if command.value.NamespaceCompleteness != cli.CommandNamespaceComplete || inspection.Truncated {
				return 0, locatorResolutionNotProvable, evaluationText("value order is incomplete")
			}
			return 0, locatorResolutionAbsent, observed(semanticSubjectValue, observationAbsent)
		}
		return command.value.Values[state.index], locatorResolutionFound, evaluationText("")
	case valueLocatorFlag:
		flag := resolveInspectionFlagLocator(state.flag, inspection)
		if flag.status != locatorResolutionFound {
			return 0, flag.status, flag.detail
		}
		return flag.value.Destination, locatorResolutionFound, evaluationText("")
	case valueLocatorArgumentTerm:
		term := resolveInspectionArgumentTermLocator(state.argument, inspection)
		if term.status != locatorResolutionFound {
			return 0, term.status, term.detail
		}
		return term.value.Destination, locatorResolutionFound, evaluationText("")
	case valueLocatorEnvironment:
		environment := resolveInspectionEnvironmentInputLocator(state.environment, inspection)
		if environment.status != locatorResolutionFound {
			return 0, environment.status, environment.detail
		}
		return environment.value.Destination, locatorResolutionFound, evaluationText("")
	case valueLocatorFact:
		return resolveInspectionFactID(state.fact, inspection)
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectValue, observationInvalidLocator)
	}
}

func resolveStaticDefaultSchema(locator ValueStaticDefaultLocator, schema *cli.Schema) (cli.ValueStaticDefault, locatorResolutionStatus, evaluationField) {
	resolved := resolveSchemaValueLocator(locator.value.state, schema)
	if resolved.status != locatorResolutionFound {
		return cli.ValueStaticDefault{}, resolved.status, resolved.detail
	}
	v := resolved.value
	if !v.HasStaticDefault {
		return cli.ValueStaticDefault{}, locatorResolutionAbsent, evaluationText("static default is absent")
	}
	return v.StaticDefault, locatorResolutionFound, evaluationText("")
}

func resolveStaticDefaultReference(locator ValueStaticDefaultLocator, reference cli.Reference) (cli.ValueStaticDefault, locatorResolutionStatus, evaluationField) {
	resolved := resolveReferenceValueLocator(locator.value.state, reference)
	if resolved.status != locatorResolutionFound {
		return cli.ValueStaticDefault{}, resolved.status, resolved.detail
	}
	v := resolved.value
	if !v.HasStaticDefault {
		return cli.ValueStaticDefault{}, locatorResolutionAbsent, evaluationText("static default is absent")
	}
	return v.StaticDefault, locatorResolutionFound, evaluationText("")
}

func evaluateStaticDefaultExpectation(expectation staticDefaultExpectationValue, value cli.ValueStaticDefault, status locatorResolutionStatus, detail evaluationField) evaluationResult {
	if expectation.property == staticDefaultExpectationExists {
		if expectation.expected && status == locatorResolutionFound {
			return provedResult()
		}
		if !expectation.expected && status == locatorResolutionAbsent {
			return provedResult()
		}
		if status == locatorResolutionFound {
			return disprovedResult(evaluationText("static default"), semanticRequirementAbsent, evaluationText("present"))
		}
		if status == locatorResolutionAbsent {
			return disprovedResult(evaluationText("static default"), evaluationText("exists"), evaluationText("absent"))
		}
		return disprovedResult(evaluationText("static default"), evaluationText("valid locator"), nonempty(detail, observed(semanticSubjectLocator, observationInvalidLocator)))
	}
	if status != locatorResolutionFound {
		return missingResult(evaluationText("static default"), evaluationText("property"), detail, false)
	}
	var name evaluationField
	var actual bool
	switch expectation.property {
	case staticDefaultExpectationSensitive:
		name, actual = semanticRequirementSensitive, value.Sensitive
	case staticDefaultExpectationRedacted:
		name, actual = evaluationText("redacted"), value.Redacted
	case staticDefaultExpectationHasRepresentation:
		name, actual = evaluationText("has-representation"), value.HasRepresentation
	default:
		panic("clitest: invalid static-default expectation property")
	}
	return equalFieldResult(evaluationText("static default"), name, evaluationBool(expectation.expected), evaluationBool(actual), actual == expectation.expected)
}

func evaluateSequenceExpectation(expectation sequenceExpectationValue, value cli.ValueSequenceLiteral, status locatorResolutionStatus, detail evaluationField) evaluationResult {
	if status != locatorResolutionFound {
		return missingResult(evaluationText("static sequence"), evaluationText("property"), detail, false)
	}
	switch expectation.property {
	case sequenceExpectationElementType:
		field := valueTypeField{
			value: expectation.elementType,
		}
		observedField := valueTypeField{
			value: value.ElementType,
		}
		return equalFieldResult(evaluationText("static sequence"), evaluationText("element-type"), field, observedField, value.ElementType == expectation.elementType)
	case sequenceExpectationElementCount:
		return equalFieldResult(evaluationText("static sequence"), evaluationText("element-count"), evaluationText(strconv.Itoa(expectation.count)), evaluationText(strconv.Itoa(len(value.Elements))), len(value.Elements) == expectation.count)
	default:
		panic("clitest: invalid static-sequence expectation property")
	}
}

func evaluateKeyedExpectation(expectation keyedExpectationValue, value cli.ValueKeyedLiteral, status locatorResolutionStatus, detail evaluationField) evaluationResult {
	if status != locatorResolutionFound {
		return missingResult(evaluationText("static keyed"), evaluationText("property"), detail, false)
	}
	switch expectation.property {
	case keyedExpectationKeyType:
		field := valueTypeField{
			value: expectation.valueType,
		}
		observedField := valueTypeField{
			value: value.KeyType,
		}
		return equalFieldResult(evaluationText("static keyed"), evaluationText("key-type"), field, observedField, value.KeyType == expectation.valueType)
	case keyedExpectationValueType:
		field := valueTypeField{
			value: expectation.valueType,
		}
		observedField := valueTypeField{
			value: value.ValueType,
		}
		return equalFieldResult(evaluationText("static keyed"), evaluationText("value-type"), field, observedField, value.ValueType == expectation.valueType)
	case keyedExpectationEntryCount:
		return equalFieldResult(evaluationText("static keyed"), evaluationText("entry-count"), evaluationText(strconv.Itoa(expectation.count)), evaluationText(strconv.Itoa(len(value.Entries))), len(value.Entries) == expectation.count)
	default:
		panic("clitest: invalid static-keyed expectation property")
	}
}

func staticLiteral(locator StaticValueLiteralLocator, value cli.ValueStaticDefault, status locatorResolutionStatus, detail evaluationField) (cli.ValueLiteral, locatorResolutionStatus, evaluationField) {
	if status != locatorResolutionFound {
		return cli.ValueLiteral{}, status, detail
	}
	switch locator.mode {
	case staticValueLiteralInvalid:
		return cli.ValueLiteral{}, locatorResolutionInvalid, observed(semanticSubjectStaticLiteral, observationInvalidLocator)
	case staticValueLiteralScalar:
		if value.Scalar == nil {
			return cli.ValueLiteral{}, locatorResolutionAbsent, evaluationText("scalar literal is absent")
		}
		return *value.Scalar, locatorResolutionFound, evaluationText("")
	case staticValueLiteralSequenceElement:
		if value.Sequence == nil || locator.index < 0 || locator.index >= len(value.Sequence.Elements) {
			return cli.ValueLiteral{}, locatorResolutionAbsent, evaluationText("sequence element is absent")
		}
		return value.Sequence.Elements[locator.index], locatorResolutionFound, evaluationText("")
	case staticValueLiteralKey:
		if value.Keyed == nil || locator.index < 0 || locator.index >= len(value.Keyed.Entries) {
			return cli.ValueLiteral{}, locatorResolutionAbsent, observed(semanticSubjectKeyedEntry, observationAbsent)
		}
		return value.Keyed.Entries[locator.index].Key, locatorResolutionFound, evaluationText("")
	case staticValueLiteralValue:
		if value.Keyed == nil || locator.index < 0 || locator.index >= len(value.Keyed.Entries) {
			return cli.ValueLiteral{}, locatorResolutionAbsent, observed(semanticSubjectKeyedEntry, observationAbsent)
		}
		return value.Keyed.Entries[locator.index].Value, locatorResolutionFound, evaluationText("")
	}
	return cli.ValueLiteral{}, locatorResolutionInvalid, observed(semanticSubjectStaticLiteral, observationInvalidLocator)
}

func resolveStaticLiteralSchema(locator StaticValueLiteralLocator, schema *cli.Schema) (cli.ValueLiteral, locatorResolutionStatus, evaluationField) {
	value, status, detail := resolveStaticDefaultSchema(locator.root, schema)
	return staticLiteral(locator, value, status, detail)
}

func resolveStaticLiteralReference(locator StaticValueLiteralLocator, reference cli.Reference) (cli.ValueLiteral, locatorResolutionStatus, evaluationField) {
	value, status, detail := resolveStaticDefaultReference(locator.root, reference)
	return staticLiteral(locator, value, status, detail)
}

func evaluateStaticLiteralExpectation(expectation staticLiteralExpectationValue, value cli.ValueLiteral, status locatorResolutionStatus, detail evaluationField) evaluationResult {
	if expectation.property == staticLiteralExpectationText {
		return safeStaticLiteralText(value, status, detail, expectation.text)
	}
	if status != locatorResolutionFound {
		return missingResult(evaluationText("static literal"), evaluationText("property"), detail, false)
	}
	switch expectation.property {
	case staticLiteralExpectationType:
		field := valueTypeField{
			value: expectation.valueType,
		}
		observedField := valueTypeField{
			value: value.Type,
		}
		return equalFieldResult(evaluationText("static literal"), evaluationText("type"), field, observedField, value.Type == expectation.valueType)
	case staticLiteralExpectationSensitive:
		return equalFieldResult(evaluationText("static literal"), semanticRequirementSensitive, evaluationBool(true), evaluationBool(value.Sensitive), value.Sensitive)
	case staticLiteralExpectationRedacted:
		return equalFieldResult(evaluationText("static literal"), evaluationText("redacted"), evaluationBool(true), evaluationBool(value.Redacted), value.Redacted)
	default:
		panic("clitest: invalid static-literal expectation property")
	}
}

func safeStaticLiteralText(v cli.ValueLiteral, s locatorResolutionStatus, d evaluationField, expected string) evaluationResult {
	if s != locatorResolutionFound {
		return missingResult(evaluationText("static literal"), evaluationText("text"), d, false)
	}
	if v.Text == expected {
		return provedResult()
	}
	if v.Sensitive || v.Redacted {
		return disprovedResult(evaluationText("static literal"), evaluationText("matching text"), evaluationText("text=<redacted>"))
	}
	return disprovedResult(evaluationText("static literal"), property(evaluationText("text"), quotedTextField(expected)), property(evaluationText("text"), quotedTextField(v.Text)))
}

func factIDLocator(id cli.ValueID) FactLocator {
	state := &factLocatorState{
		mode: factLocatorID,
		id:   id,
	}
	locator := FactLocator{
		state: state,
	}
	return locator
}

func valueIDLocator(id cli.ValueID) ValueLocator {
	state := &valueLocatorState{
		mode: valueLocatorID,
		id:   id,
	}
	locator := ValueLocator{
		state: state,
	}
	return locator
}

func resolveExpectedValueIDsSchema(states []*valueLocatorState, schema *cli.Schema) ([]cli.ValueID, locatorResolutionStatus, evaluationField) {
	ids := make([]cli.ValueID, len(states))
	for index, state := range states {
		id, status, detail := resolveSchemaValueID(state, schema)
		if status != locatorResolutionFound {
			return nil, status, detail
		}
		ids[index] = id
	}
	return ids, locatorResolutionFound, evaluationText("")
}

func resolveExpectedValueIDsReference(states []*valueLocatorState, reference cli.Reference) ([]cli.ValueID, locatorResolutionStatus, evaluationField) {
	ids := make([]cli.ValueID, len(states))
	for index, state := range states {
		id, status, detail := resolveReferenceValueID(state, reference)
		if status != locatorResolutionFound {
			return nil, status, detail
		}
		ids[index] = id
	}
	return ids, locatorResolutionFound, evaluationText("")
}

func resolveExpectedValueIDsInspection(states []*valueLocatorState, inspection cli.Inspection) ([]cli.ValueID, locatorResolutionStatus, evaluationField) {
	ids := make([]cli.ValueID, len(states))
	for index, state := range states {
		id, status, detail := resolveInspectionValueID(state, inspection)
		if status != locatorResolutionFound {
			return nil, status, detail
		}
		ids[index] = id
	}
	return ids, locatorResolutionFound, evaluationText("")
}

func expectedValueIDsResult(subject evaluationField, actual, expected []cli.ValueID, status locatorResolutionStatus, detail evaluationField, name string) evaluationResult {
	expectedField := evaluationText("exact " + name)
	if status != locatorResolutionFound {
		return expectedRelationFailure(status, subject, expectedField, detail)
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(subject, name, expected, actual)
}

func expectedValueIDsInspection(states []*valueLocatorState, inspection cli.Inspection) ([]cli.ValueID, locatorResolutionStatus, evaluationField) {
	ids := make([]cli.ValueID, len(states))
	for index, state := range states {
		id, status, detail := resolveInspectionValueID(state, inspection)
		if status != locatorResolutionFound {
			return nil, status, detail
		}
		ids[index] = id
	}

	return ids, locatorResolutionFound, evaluationText("")
}

func valueCandidateIDLocator(id cli.ValueCandidateID) ValueCandidateLocator {
	state := &valueCandidateLocatorState{
		mode: valueCandidateLocatorID,
		id:   id,
	}
	locator := ValueCandidateLocator{
		state: state,
	}
	return locator
}

func valueProducerIndexLocator(value ValueLocator, index int) ValueProducerLocator {
	state := &valueProducerLocatorState{
		value: value.state,
		index: index,
	}
	locator := ValueProducerLocator{
		state: state,
	}
	return locator
}

func valueCandidateIndexLocator(value ValueLocator, index int) ValueCandidateLocator {
	state := &valueCandidateLocatorState{
		mode:  valueCandidateLocatorCandidateIndex,
		value: value.state,
		index: index,
	}
	locator := ValueCandidateLocator{
		state: state,
	}
	return locator
}

func valueContributorIndexLocator(value ValueLocator, index int) ValueCandidateLocator {
	state := &valueCandidateLocatorState{
		mode:  valueCandidateLocatorContributorIndex,
		value: value.state,
		index: index,
	}
	locator := ValueCandidateLocator{
		state: state,
	}
	return locator
}

func valueElementIndexLocator(candidate ValueCandidateLocator, index int) ValueElementLocator {
	component := ValueComponentLocator{
		state: &valueComponentLocatorState{
			mode:      valueComponentLocatorElement,
			candidate: candidate.state,
			index:     index,
		},
	}
	locator := ValueElementLocator{
		ValueComponentLocator: component,
	}
	return locator
}

func valueEntryIndexLocator(candidate ValueCandidateLocator, index int) ValueEntryLocator {
	state := &valueEntryLocatorState{
		candidate: candidate.state,
		index:     index,
	}
	locator := ValueEntryLocator{
		state: state,
	}
	return locator
}

func valueEntryKeyLocator(entry ValueEntryLocator) ValueComponentLocator {
	state := &valueComponentLocatorState{
		mode:  valueComponentLocatorEntryKey,
		entry: entry.state,
	}
	locator := ValueComponentLocator{
		state: state,
	}
	return locator
}

func valueEntryValueLocator(entry ValueEntryLocator) ValueComponentLocator {
	state := &valueComponentLocatorState{
		mode:  valueComponentLocatorEntryValue,
		entry: entry.state,
	}
	locator := ValueComponentLocator{
		state: state,
	}
	return locator
}

func resolveValue(state *valueLocatorState, inspection cli.Inspection) valueResolution {
	var resolved valueResolution
	resolution := resolveInspectionValueLocator(state, inspection)
	resolved = valueResolution{
		status: resolution.status,
		value:  resolution.value,
		label:  resolution.subject,
		actual: resolution.detail,
	}
	return resolved
}

func valueProducerLocatorLabel(state *valueProducerLocatorState) evaluationField {
	if state == nil {
		return evaluationText("value producer")
	}
	return qualifySubject(valueLocatorLabel(state.value), evaluationText(fmt.Sprintf("producer at index %d", state.index)))
}

func resolveProducer(state *valueProducerLocatorState, inspection cli.Inspection) producerResolution {
	var resolution producerResolution
	label := valueProducerLocatorLabel(state)
	if state == nil || state.value == nil || state.index < 0 {
		resolution = producerResolution{
			status: locatorResolutionInvalid,
			label:  label,
			actual: evaluationText("invalid value-producer locator"),
		}
		return resolution
	}
	parent := resolveValue(state.value, inspection)
	if parent.status != locatorResolutionFound {
		resolution = producerResolution{
			status: parent.status,
			label:  label,
			actual: parent.actual,
		}
		return resolution
	}
	if state.index < len(parent.value.Producers) {
		resolution = producerResolution{
			status: locatorResolutionFound,
			value:  parent.value.Producers[state.index],
			index:  state.index,
			label:  label,
		}
		return resolution
	}
	if inspection.Truncated {
		resolution = producerResolution{
			status: locatorResolutionNotProvable,
			label:  label,
			actual: observed(semanticSubjectInspection, observationTruncated),
		}
		return resolution
	}
	resolution = producerResolution{
		status: locatorResolutionAbsent,
		label:  label,
		actual: evaluationText("producer is absent"),
	}
	return resolution
}

func valueCandidateLocatorLabel(state *valueCandidateLocatorState) evaluationField {
	if state == nil {
		return evaluationText("value candidate")
	}
	switch state.mode {
	case valueCandidateLocatorID:
		return evaluationText(fmt.Sprintf("value candidate %d", state.id))
	case valueCandidateLocatorCandidateIndex:
		return qualifySubject(valueLocatorLabel(state.value), evaluationText(fmt.Sprintf("candidate at index %d", state.index)))
	case valueCandidateLocatorContributorIndex:
		return qualifySubject(valueLocatorLabel(state.value), evaluationText(fmt.Sprintf("contributor at index %d", state.index)))
	default:
		return evaluationText("value candidate")
	}
}

func resolveCandidate(state *valueCandidateLocatorState, inspection cli.Inspection) candidateResolution {
	var resolution candidateResolution
	label := valueCandidateLocatorLabel(state)
	if state == nil || state.mode == valueCandidateLocatorInvalid {
		resolution = candidateResolution{
			status: locatorResolutionInvalid,
			label:  label,
			actual: evaluationText("invalid value-candidate locator"),
		}
		return resolution
	}

	var id cli.ValueCandidateID
	switch state.mode {
	case valueCandidateLocatorID:
		id = state.id
	case valueCandidateLocatorCandidateIndex, valueCandidateLocatorContributorIndex:
		if state.value == nil || state.index < 0 {
			resolution = candidateResolution{
				status: locatorResolutionInvalid,
				label:  label,
				actual: evaluationText("invalid value-candidate locator"),
			}
			return resolution
		}
		parent := resolveValue(state.value, inspection)
		if parent.status != locatorResolutionFound {
			resolution = candidateResolution{
				status: parent.status,
				label:  label,
				actual: parent.actual,
			}
			return resolution
		}
		ids := parent.value.Candidates
		relation := semanticNameCandidate
		if state.mode == valueCandidateLocatorContributorIndex {
			ids = parent.value.Contributors
			relation = "contributor"
		}
		if state.index >= len(ids) {
			if inspection.Truncated {
				resolution = candidateResolution{
					status: locatorResolutionNotProvable,
					label:  label,
					actual: observed(semanticSubjectInspection, observationTruncated),
				}
				return resolution
			}
			resolution = candidateResolution{
				status: locatorResolutionAbsent,
				label:  label,
				actual: evaluationText(relation + " is absent"),
			}
			return resolution
		}
		id = ids[state.index]
	default:
		resolution = candidateResolution{
			status: locatorResolutionInvalid,
			label:  label,
			actual: evaluationText("invalid value-candidate locator"),
		}
		return resolution
	}

	candidate, ok := inspection.ValueCandidates[id]
	if ok {
		resolution = candidateResolution{
			status: locatorResolutionFound,
			value:  candidate,
			id:     id,
			label:  label,
		}
		return resolution
	}
	if inspection.Truncated {
		resolution = candidateResolution{
			status: locatorResolutionNotProvable,
			id:     id,
			label:  label,
			actual: evaluationText("candidate details are unavailable from truncated inspection"),
		}
		return resolution
	}
	if state.mode == valueCandidateLocatorID {
		resolution = candidateResolution{
			status: locatorResolutionAbsent,
			id:     id,
			label:  label,
			actual: evaluationText("candidate is absent"),
		}
		return resolution
	}
	resolution = candidateResolution{
		status: locatorResolutionNotProvable,
		id:     id,
		label:  label,
		actual: evaluationText("referenced candidate details are unavailable"),
	}
	return resolution
}

func valueEntryLocatorLabel(state *valueEntryLocatorState) evaluationField {
	if state == nil {
		return evaluationText("value entry")
	}
	return qualifySubject(valueCandidateLocatorLabel(state.candidate), evaluationText(fmt.Sprintf("entry at index %d", state.index)))
}

func resolveEntry(state *valueEntryLocatorState, inspection cli.Inspection) (cli.ValueEntryInspection, locatorResolutionStatus, evaluationField) {
	if state == nil || state.candidate == nil || state.index < 0 {
		return cli.ValueEntryInspection{}, locatorResolutionInvalid, evaluationText("invalid value-entry locator")
	}
	candidate := resolveCandidate(state.candidate, inspection)
	if candidate.status != locatorResolutionFound {
		return cli.ValueEntryInspection{}, candidate.status, candidate.actual
	}
	if state.index < len(candidate.value.Entries) {
		return candidate.value.Entries[state.index], locatorResolutionFound, evaluationText("")
	}
	if inspection.Truncated {
		return cli.ValueEntryInspection{}, locatorResolutionNotProvable, observed(semanticSubjectInspection, observationTruncated)
	}
	return cli.ValueEntryInspection{}, locatorResolutionAbsent, evaluationText("entry is absent")
}

func valueComponentLocatorLabel(state *valueComponentLocatorState) evaluationField {
	if state == nil {
		return evaluationText("value component")
	}
	switch state.mode {
	case valueComponentLocatorElement:
		return qualifySubject(valueCandidateLocatorLabel(state.candidate), evaluationText(fmt.Sprintf("element at index %d", state.index)))
	case valueComponentLocatorEntryKey:
		return qualifySubject(valueEntryLocatorLabel(state.entry), evaluationText("key"))
	case valueComponentLocatorEntryValue:
		return qualifySubject(valueEntryLocatorLabel(state.entry), evaluationText("value"))
	default:
		return evaluationText("value component")
	}
}

func resolveComponent(state *valueComponentLocatorState, inspection cli.Inspection) componentResolution {
	var resolution componentResolution
	label := valueComponentLocatorLabel(state)
	if state == nil || state.mode == valueComponentLocatorInvalid {
		resolution = componentResolution{
			status: locatorResolutionInvalid,
			label:  label,
			actual: evaluationText("invalid value-component locator"),
		}
		return resolution
	}
	switch state.mode {
	case valueComponentLocatorElement:
		if state.candidate == nil || state.index < 0 {
			resolution = componentResolution{
				status: locatorResolutionInvalid,
				label:  label,
				actual: evaluationText("invalid value-component locator"),
			}
			return resolution
		}
		candidate := resolveCandidate(state.candidate, inspection)
		if candidate.status != locatorResolutionFound {
			resolution = componentResolution{
				status: candidate.status,
				label:  label,
				actual: candidate.actual,
			}
			return resolution
		}
		if state.index < len(candidate.value.Elements) {
			resolution = componentResolution{
				status: locatorResolutionFound,
				value:  candidate.value.Elements[state.index],
				label:  label,
			}
			return resolution
		}
		if inspection.Truncated {
			resolution = componentResolution{
				status: locatorResolutionNotProvable,
				label:  label,
				actual: observed(semanticSubjectInspection, observationTruncated),
			}
			return resolution
		}
		resolution = componentResolution{
			status: locatorResolutionAbsent,
			label:  label,
			actual: evaluationText("element is absent"),
		}
		return resolution
	case valueComponentLocatorEntryKey, valueComponentLocatorEntryValue:
		entry, status, actual := resolveEntry(state.entry, inspection)
		if status != locatorResolutionFound {
			resolution = componentResolution{
				status: status,
				label:  label,
				actual: actual,
			}
			return resolution
		}
		value := entry.Key
		if state.mode == valueComponentLocatorEntryValue {
			value = entry.Value
		}
		resolution = componentResolution{
			status: locatorResolutionFound,
			value:  value,
			label:  label,
		}
		return resolution
	default:
		resolution = componentResolution{
			status: locatorResolutionInvalid,
			label:  label,
			actual: evaluationText("invalid value-component locator"),
		}
		return resolution
	}
}

func resolveDomain(state *valueDomainLocatorState, inspection cli.Inspection) domainResolution {
	var resolution domainResolution
	label := valueDomainLocatorLabel(state)
	if state == nil || !state.valid {
		resolution = domainResolution{
			status: locatorResolutionInvalid,
			index:  -1,
			label:  label,
			actual: observed(semanticSubjectValueDomain, observationInvalidLocator),
		}
		return resolution
	}
	valueID, status, detail := resolveInspectionValueID(state.value, inspection)
	if status != locatorResolutionFound {
		resolution = domainResolution{
			status: status,
			index:  -1,
			label:  label,
			actual: detail,
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
		index := matches[0]
		resolution = domainResolution{
			status: locatorResolutionFound,
			value:  inspection.ValueDomains[index],
			index:  index,
			label:  label,
		}
		return resolution
	case 0:
		if inspection.Truncated {
			resolution = domainResolution{
				status: locatorResolutionNotProvable,
				index:  -1,
				label:  label,
				actual: observed(semanticSubjectInspection, observationTruncated),
			}
			return resolution
		}
		resolution = domainResolution{
			status: locatorResolutionAbsent,
			index:  -1,
			label:  label,
			actual: observed(semanticSubjectValueDomain, observationAbsent),
		}
		return resolution
	default:
		resolution = domainResolution{
			status: locatorResolutionAmbiguous,
			index:  -1,
			label:  label,
			actual: collectionCardinalityField{
				name:  semanticNameMatchesValueDomainIndexes,
				count: len(matches),
			},
		}
		return resolution
	}
}

func outcomeForValueStatus(status locatorResolutionStatus, label, expect, actual evaluationField) evaluationResult {
	if status == locatorResolutionNotProvable {
		return notProvableResult(label, expect, actual)
	}
	return disprovedResult(label, expect, actual)
}

func valueObservationExpectation(locator ValueLocator, observed bool) InspectionExpectation {
	value := valueObservationExpectationValue{
		locator:  locator.state,
		observed: observed,
	}
	return value
}

func valueStateExpectation(locator ValueLocator, state cli.ResolutionState) InspectionExpectation {
	expectation := valuePropertyExpectation{
		locator: locator.state,
		kind:    valuePropertyState,
		state:   state,
	}
	return expectation
}

func valueSensitiveInspectionExpectation(locator ValueLocator) InspectionExpectation {
	expectation := valuePropertyExpectation{
		locator: locator.state,
		kind:    valuePropertySensitive,
	}
	return expectation
}

func valueTextExpectation(locator ValueLocator, text string) InspectionExpectation {
	expectation := valuePropertyExpectation{
		locator: locator.state,
		kind:    valuePropertyText,
		text:    text,
	}
	return expectation
}

func valueTokensExpectation(locator ValueLocator, tokens []string) InspectionExpectation {
	expectation := valuePropertyExpectation{
		locator: locator.state,
		kind:    valuePropertyTokens,
		tokens:  slices.Clone(tokens),
	}
	return expectation
}

func valueRedactedExpectation(locator ValueLocator) InspectionExpectation {
	expectation := valuePropertyExpectation{
		locator: locator.state,
		kind:    valuePropertyRedacted,
	}
	return expectation
}

func valueContributedByExpectation(locator ValueLocator, kind cli.ValueProducerKind) InspectionExpectation {
	expectation := valuePropertyExpectation{
		locator:      locator.state,
		kind:         valuePropertyContributedBy,
		producerKind: kind,
	}
	return expectation
}

func valueBlockedByExpectation(locator, blocker ValueLocator) InspectionExpectation {
	value := valueBlockedByExpectationValue{
		value:   locator.state,
		blocker: blocker.state,
	}
	return value
}

func valueProducerKindExpectation(locator ValueProducerLocator, kind cli.ValueProducerKind) InspectionExpectation {
	expectation := producerPropertyExpectation{
		locator:      locator.state,
		kind:         valueProducerPropertyProducerKind,
		producerKind: kind,
	}
	return expectation
}

func valueProducerTierExpectation(locator ValueProducerLocator, tier cli.ResolutionTier) InspectionExpectation {
	expectation := producerPropertyExpectation{
		locator: locator.state,
		kind:    valueProducerPropertyTier,
		tier:    tier,
	}
	return expectation
}

func valueProducerOutcomeExpectation(locator ValueProducerLocator, outcome cli.ValueProducerOutcome) InspectionExpectation {
	expectation := producerPropertyExpectation{
		locator: locator.state,
		kind:    valueProducerPropertyOutcome,
		outcome: outcome,
	}
	return expectation
}

func valueProducerSensitiveExpectation(locator ValueProducerLocator) InspectionExpectation {
	expectation := producerPropertyExpectation{
		locator: locator.state,
		kind:    valueProducerPropertySensitive,
	}
	return expectation
}

func valueProducerBlockedByExactlyExpectation(locator ValueProducerLocator, values []ValueLocator) InspectionExpectation {
	states := make([]*valueLocatorState, len(values))
	for index, value := range values {
		states[index] = value.state
	}
	expectation := producerPropertyExpectation{
		locator: locator.state,
		kind:    valueProducerPropertyBlockedBy,
		values:  states,
	}
	return expectation
}

func valueProducerDependenciesExactlyExpectation(locator ValueProducerLocator, values []ValueLocator) InspectionExpectation {
	states := make([]*valueLocatorState, len(values))
	for index, value := range values {
		states[index] = value.state
	}
	expectation := producerPropertyExpectation{
		locator: locator.state,
		kind:    valueProducerPropertyDependencies,
		values:  states,
	}
	return expectation
}

func valueProducerOriginsExactlyExpectation(locator ValueProducerLocator, origins []InputOriginLocator) InspectionExpectation {
	expected, valid := locatorInputOriginIDs(origins)
	expectation := producerPropertyExpectation{
		locator:      locator.state,
		kind:         valueProducerPropertyOrigins,
		origins:      expected,
		originsValid: valid,
	}
	return expectation
}

func valueCandidateDispositionExpectation(locator ValueCandidateLocator, disposition cli.ValueCandidateDisposition) InspectionExpectation {
	expectation := candidatePropertyExpectation{
		locator:     locator.state,
		kind:        valueCandidatePropertyDisposition,
		disposition: disposition,
	}
	return expectation
}

func valueCandidateContributionExpectation(locator ValueCandidateLocator, contribution cli.ValueCandidateContribution) InspectionExpectation {
	expectation := candidatePropertyExpectation{
		locator:      locator.state,
		kind:         valueCandidatePropertyContribution,
		contribution: contribution,
	}
	return expectation
}

func valueCandidateSensitiveExpectation(locator ValueCandidateLocator) InspectionExpectation {
	expectation := candidatePropertyExpectation{
		locator: locator.state,
		kind:    valueCandidatePropertySensitive,
	}
	return expectation
}

func valueCandidateProducedByExpectation(locator ValueCandidateLocator, producer ValueProducerLocator) InspectionExpectation {
	value := valueCandidateProducedByExpectationValue{
		candidate: locator.state,
		producer:  producer.state,
	}
	return value
}

func valueCandidateDeclarationExpectation(locator ValueCandidateLocator, declaration DeclarationIdentityLocator) InspectionExpectation {
	value := valueCandidateDeclarationExpectationValue{
		candidate:   locator.state,
		declaration: declaration,
	}
	return value
}

func valueCandidateOriginsExactlyExpectation(locator ValueCandidateLocator, origins []InputOriginLocator) InspectionExpectation {
	expected, valid := locatorInputOriginIDs(origins)
	expectation := candidatePropertyExpectation{
		locator:      locator.state,
		kind:         valueCandidatePropertyOrigins,
		origins:      expected,
		originsValid: valid,
	}
	return expectation
}

func valueCandidateSuppressionExpectation(locator ValueCandidateLocator, suppression cli.ValueCandidateSuppression) InspectionExpectation {
	expectation := candidatePropertyExpectation{
		locator:     locator.state,
		kind:        valueCandidatePropertySuppression,
		suppression: suppression,
	}
	return expectation
}

func valueCandidateTextExpectation(locator ValueCandidateLocator, text string) InspectionExpectation {
	expectation := candidatePropertyExpectation{
		locator: locator.state,
		kind:    valueCandidatePropertyText,
		text:    text,
	}
	return expectation
}

func valueCandidateTokensExpectation(locator ValueCandidateLocator, tokens []string) InspectionExpectation {
	expectation := candidatePropertyExpectation{
		locator: locator.state,
		kind:    valueCandidatePropertyTokens,
		tokens:  slices.Clone(tokens),
	}
	return expectation
}

func valueCandidateElementCountExpectation(locator ValueCandidateLocator, count int) InspectionExpectation {
	expectation := candidatePropertyExpectation{
		locator: locator.state,
		kind:    valueCandidatePropertyElementCount,
		count:   count,
	}
	return expectation
}

func valueCandidateEntryCountExpectation(locator ValueCandidateLocator, count int) InspectionExpectation {
	expectation := candidatePropertyExpectation{
		locator: locator.state,
		kind:    valueCandidatePropertyEntryCount,
		count:   count,
	}
	return expectation
}

func candidateLocatorID(state *valueCandidateLocatorState, inspection cli.Inspection) (cli.ValueCandidateID, bool) {
	resolved := resolveCandidate(state, inspection)
	return resolved.id, resolved.status == locatorResolutionFound
}

func valueCandidateSuppressedByExactlyExpectation(locator ValueCandidateLocator, expectedLocators []ValueCandidateLocator) InspectionExpectation {
	states := make([]*valueCandidateLocatorState, len(expectedLocators))
	for index, expected := range expectedLocators {
		states[index] = expected.state
	}
	expectation := candidatePropertyExpectation{
		locator:    locator.state,
		kind:       valueCandidatePropertySuppressedBy,
		candidates: states,
	}
	return expectation
}

func valueCandidateConflictsWithExactlyExpectation(locator ValueCandidateLocator, expectedLocators []ValueCandidateLocator) InspectionExpectation {
	states := make([]*valueCandidateLocatorState, len(expectedLocators))
	for index, expected := range expectedLocators {
		states[index] = expected.state
	}
	expectation := candidatePropertyExpectation{
		locator:    locator.state,
		kind:       valueCandidatePropertyConflictsWith,
		candidates: states,
	}
	return expectation
}

func valueCandidateInputTextExpectation(locator ValueCandidateLocator, text string) InspectionExpectation {
	expectation := candidatePropertyExpectation{
		locator: locator.state,
		kind:    valueCandidatePropertyInputText,
		text:    text,
	}
	return expectation
}

func valueCandidateInputRedactedExpectation(locator ValueCandidateLocator) InspectionExpectation {
	expectation := candidatePropertyExpectation{
		locator: locator.state,
		kind:    valueCandidatePropertyInputRedacted,
	}
	return expectation
}

func valueCandidateRedactedExpectation(locator ValueCandidateLocator) InspectionExpectation {
	expectation := candidatePropertyExpectation{
		locator: locator.state,
		kind:    valueCandidatePropertyRedacted,
	}
	return expectation
}

func valueComponentTextExpectation(locator ValueComponentLocator, text string) InspectionExpectation {
	expectation := componentPropertyExpectation{
		locator: locator.state,
		kind:    valueComponentPropertyText,
		text:    text,
	}
	return expectation
}

func valueComponentTokensExpectation(locator ValueComponentLocator, tokens []string) InspectionExpectation {
	expectation := componentPropertyExpectation{
		locator: locator.state,
		kind:    valueComponentPropertyTokens,
		tokens:  slices.Clone(tokens),
	}
	return expectation
}

func valueComponentRedactedExpectation(locator ValueComponentLocator) InspectionExpectation {
	expectation := componentPropertyExpectation{
		locator: locator.state,
		kind:    valueComponentPropertyRedacted,
	}
	return expectation
}

func valueComponentOriginsExactlyExpectation(locator ValueComponentLocator, origins []InputOriginLocator) InspectionExpectation {
	expected, valid := locatorInputOriginIDs(origins)
	expectation := componentPropertyExpectation{
		locator:      locator.state,
		kind:         valueComponentPropertyOrigins,
		origins:      expected,
		originsValid: valid,
	}
	return expectation
}

func valueComponentSensitiveExpectation(locator ValueComponentLocator) InspectionExpectation {
	expectation := componentPropertyExpectation{
		locator: locator.state,
		kind:    valueComponentPropertySensitive,
	}
	return expectation
}

func valueComponentInputTextExpectation(locator ValueComponentLocator, text string) InspectionExpectation {
	expectation := componentPropertyExpectation{
		locator: locator.state,
		kind:    valueComponentPropertyInputText,
		text:    text,
	}
	return expectation
}

func valueProducersExactlyExpectation(locator ValueLocator, producers []ValueProducerLocator) InspectionExpectation {
	states := make([]*valueProducerLocatorState, len(producers))
	for index, producer := range producers {
		states[index] = producer.state
	}
	expectation := valueCollectionExpectation{
		locator:   locator.state,
		kind:      valueCollectionProducers,
		producers: states,
	}
	return expectation
}

func valueCandidatesExactlyExpectation(locator ValueLocator, candidates []ValueCandidateLocator) InspectionExpectation {
	states := make([]*valueCandidateLocatorState, len(candidates))
	for index, candidate := range candidates {
		states[index] = candidate.state
	}
	expectation := valueCollectionExpectation{
		locator:    locator.state,
		kind:       valueCollectionCandidates,
		candidates: states,
	}
	return expectation
}

func valueContributorsExactlyExpectation(locator ValueLocator, candidates []ValueCandidateLocator) InspectionExpectation {
	states := make([]*valueCandidateLocatorState, len(candidates))
	for index, candidate := range candidates {
		states[index] = candidate.state
	}
	expectation := valueCollectionExpectation{
		locator:    locator.state,
		kind:       valueCollectionContributors,
		candidates: states,
	}
	return expectation
}

func componentReferenceFromElementLocator(locator ValueElementLocator, inspection cli.Inspection) (cli.ValueElementReference, bool) {
	var reference cli.ValueElementReference
	state := locator.state
	if state == nil || state.mode != valueComponentLocatorElement || state.candidate == nil || state.index < 0 {
		return cli.ValueElementReference{}, false
	}
	candidate := resolveCandidate(state.candidate, inspection)
	if candidate.status != locatorResolutionFound || state.index < 0 || state.index >= len(candidate.value.Elements) {
		return cli.ValueElementReference{}, false
	}
	reference = cli.ValueElementReference{
		Candidate: candidate.id,
		Index:     state.index,
	}
	return reference, true
}

func entryReferenceFromLocator(locator ValueEntryLocator, inspection cli.Inspection) (cli.ValueEntryReference, bool) {
	var reference cli.ValueEntryReference
	state := locator.state
	if state == nil || state.candidate == nil || state.index < 0 {
		return cli.ValueEntryReference{}, false
	}
	candidate := resolveCandidate(state.candidate, inspection)
	if candidate.status != locatorResolutionFound || state.index < 0 || state.index >= len(candidate.value.Entries) {
		return cli.ValueEntryReference{}, false
	}
	reference = cli.ValueEntryReference{
		Candidate: candidate.id,
		Index:     state.index,
	}
	return reference, true
}

func valueElementsExactlyExpectation(locator ValueLocator, elements []ValueElementLocator) InspectionExpectation {
	expectation := valueCollectionExpectation{
		locator:  locator.state,
		kind:     valueCollectionElements,
		elements: slices.Clone(elements),
	}
	return expectation
}

func valueEntriesExactlyExpectation(locator ValueLocator, entries []ValueEntryLocator) InspectionExpectation {
	expectation := valueCollectionExpectation{
		locator: locator.state,
		kind:    valueCollectionEntries,
		entries: slices.Clone(entries),
	}
	return expectation
}

func valueCandidateDomainEntryExpectation(locator ValueCandidateLocator, entry ValueDomainEntryLocator) InspectionExpectation {
	value := valueCandidateDomainEntryExpectationValue{
		candidate: locator.state,
		entry:     entry.state,
	}
	return value
}

func valueComponentDomainEntryExpectation(locator ValueComponentLocator, entry ValueDomainEntryLocator) InspectionExpectation {
	expectation := componentDomainEntryExpectation{
		locator: locator.state,
		entry:   entry.state,
	}
	return expectation
}

func (commandFactsExpectation) schemaExpectation() {}

func (commandFactsExpectation) referenceExpectation() {}

func (commandFactsExpectation) inspectionExpectation() {}

//nolint:dupl // Typed collection proofs intentionally keep each public declaration family direct; generic resolver callbacks would recreate the rejected universal framework.
func (expectation commandFactsExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	command := resolveSchemaCommandLocator(expectation.command, schema)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("facts exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("facts"))
	actual := command.value.Facts
	for _, id := range actual {
		if _, ok := schema.Facts[id]; !ok {
			return collectionClosureFailure(label, "facts", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.ValueID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveSchemaFactLocator(state, schema)
		if outcome, failed := collectionResolutionFailure(label, "facts", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "facts", expected, actual)
}

//nolint:dupl // Typed collection proofs intentionally keep each public declaration family direct; generic resolver callbacks would recreate the rejected universal framework.
func (expectation commandFactsExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	command := resolveReferenceCommandLocator(expectation.command, reference)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("facts exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("facts"))
	actual := command.value.Facts
	for _, id := range actual {
		if _, ok := reference.Facts[id]; !ok {
			return collectionClosureFailure(label, "facts", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.ValueID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveReferenceFactLocator(state, reference)
		if outcome, failed := collectionResolutionFailure(label, "facts", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "facts", expected, actual)
}

func (expectation commandFactsExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	command := resolveInspectionCommandLocator(expectation.command, inspection)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("facts exactly"), command.detail)
	}
	label := qualifySubject(command.subject, evaluationText("facts"))
	if inspection.Truncated {
		return collectionClosureFailure(label, "facts", observed(semanticSubjectInspection, observationTruncated))
	}
	if command.value.NamespaceCompleteness != cli.CommandNamespaceComplete {
		return collectionClosureFailure(label, "facts", evaluationText(fmt.Sprintf("command namespace is %v", command.value.NamespaceCompleteness)))
	}
	actual := command.value.Facts
	for _, id := range actual {
		if _, ok := inspection.Facts[id]; !ok {
			return collectionClosureFailure(label, "facts", evaluationText(fmt.Sprintf("relationship references missing detail %v", id)))
		}
	}
	expected := make([]cli.ValueID, len(expectation.items))
	for index, state := range expectation.items {
		resolved := resolveInspectionFactLocator(state, inspection)
		if outcome, failed := collectionResolutionFailure(label, "facts", index, resolved.status, resolved.detail); failed {
			return outcome
		}
		expected[index] = resolved.value.ID
	}
	if slices.Equal(actual, expected) {
		return provedResult()
	}
	return orderedCollectionMismatch(label, "facts", expected, actual)
}

func (factAvailabilityExpectation) referenceExpectation() {}

func (expectation factAvailabilityExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFactLocator(expectation.locator, reference)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "available", expectation.expected, resolution.value.Available)
}

func (valueReferenceAvailabilityExpectationValue) referenceExpectation() {}

func (expectation valueReferenceAvailabilityExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceValueLocator(expectation.locator, reference)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "available", expectation.expected, resolution.value.Available)
}

func (valueReferenceVisibilityExpectationValue) referenceExpectation() {}

func (expectation valueReferenceVisibilityExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceValueLocator(expectation.locator, reference)
	return declarationEffectiveBoolResult(resolution.status, resolution.subject, resolution.detail, "visible", expectation.expected, resolution.value.Visible)
}

func (staticDefaultExpectationValue) schemaExpectation() {}

func (staticDefaultExpectationValue) referenceExpectation() {}

func (expectation staticDefaultExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	value, status, detail := resolveStaticDefaultSchema(expectation.locator, schema)
	return evaluateStaticDefaultExpectation(expectation, value, status, detail)
}

func (expectation staticDefaultExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	value, status, detail := resolveStaticDefaultReference(expectation.locator, reference)
	return evaluateStaticDefaultExpectation(expectation, value, status, detail)
}

func (sequenceExpectationValue) schemaExpectation() {}

func (sequenceExpectationValue) referenceExpectation() {}

func (expectation sequenceExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	value, status, detail := resolveSequenceSchema(expectation.locator, schema)
	return evaluateSequenceExpectation(expectation, value, status, detail)
}

func (expectation sequenceExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	value, status, detail := resolveSequenceReference(expectation.locator, reference)
	return evaluateSequenceExpectation(expectation, value, status, detail)
}

func (keyedExpectationValue) schemaExpectation() {}

func (keyedExpectationValue) referenceExpectation() {}

func (expectation keyedExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	value, status, detail := resolveKeyedSchema(expectation.locator, schema)
	return evaluateKeyedExpectation(expectation, value, status, detail)
}

func (expectation keyedExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	value, status, detail := resolveKeyedReference(expectation.locator, reference)
	return evaluateKeyedExpectation(expectation, value, status, detail)
}

func (staticLiteralExpectationValue) schemaExpectation() {}

func (staticLiteralExpectationValue) referenceExpectation() {}

func (expectation staticLiteralExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	value, status, detail := resolveStaticLiteralSchema(expectation.locator, schema)
	return evaluateStaticLiteralExpectation(expectation, value, status, detail)
}

func (expectation staticLiteralExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	value, status, detail := resolveStaticLiteralReference(expectation.locator, reference)
	return evaluateStaticLiteralExpectation(expectation, value, status, detail)
}

func (factDependenciesExactlyExpectation) schemaExpectation() {}

func (factDependenciesExactlyExpectation) referenceExpectation() {}

func (factDependenciesExactlyExpectation) inspectionExpectation() {}

func (expectation factDependenciesExactlyExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolved := resolveSchemaFactLocator(expectation.locator, schema)
	const expected = "exact semantic fact dependencies in order"
	if resolved.status != locatorResolutionFound {
		return declarationMissing(resolved.status, resolved.subject, evaluationText(expected), resolved.detail, false)
	}
	ids := make([]cli.ValueID, len(expectation.dependencies))
	for index, state := range expectation.dependencies {
		value := resolveSchemaValueLocator(state, schema)
		if value.status != locatorResolutionFound {
			return declarationMissing(value.status, resolved.subject, evaluationText(expected), value.detail, false)
		}
		ids[index] = value.value.ID
	}
	if slices.Equal(resolved.value.Dependencies, ids) {
		return provedResult()
	}
	return disprovedResult(resolved.subject, evaluationText(expected), evaluationText("different semantic dependency sequence"))
}

func (expectation factDependenciesExactlyExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolved := resolveReferenceFactLocator(expectation.locator, reference)
	const expected = "exact semantic fact dependencies in order"
	if resolved.status != locatorResolutionFound {
		return declarationMissing(resolved.status, resolved.subject, evaluationText(expected), resolved.detail, false)
	}
	ids := make([]cli.ValueID, len(expectation.dependencies))
	for index, state := range expectation.dependencies {
		value := resolveReferenceValueLocator(state, reference)
		if value.status != locatorResolutionFound {
			return declarationMissing(value.status, resolved.subject, evaluationText(expected), value.detail, false)
		}
		ids[index] = value.value.ID
	}
	if slices.Equal(resolved.value.Dependencies, ids) {
		return provedResult()
	}
	return disprovedResult(resolved.subject, evaluationText(expected), evaluationText("different semantic dependency sequence"))
}

func (expectation factDependenciesExactlyExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveInspectionFactLocator(expectation.locator, inspection)
	const expected = "exact semantic fact dependencies in order"
	if resolved.status != locatorResolutionFound {
		return declarationMissing(resolved.status, resolved.subject, evaluationText(expected), resolved.detail, inspection.Truncated)
	}
	ids := make([]cli.ValueID, len(expectation.dependencies))
	for index, state := range expectation.dependencies {
		value := resolveInspectionValueLocator(state, inspection)
		if value.status != locatorResolutionFound {
			return declarationMissing(value.status, resolved.subject, evaluationText(expected), value.detail, inspection.Truncated)
		}
		ids[index] = value.value.ID
	}
	if slices.Equal(resolved.value.Dependencies, ids) {
		return provedResult()
	}
	return disprovedResult(resolved.subject, evaluationText(expected), evaluationText("different semantic dependency sequence"))
}

func (factExistenceExpectation) schemaExpectation() {}

func (factExistenceExpectation) referenceExpectation() {}

func (factExistenceExpectation) inspectionExpectation() {}

func (expectation factExistenceExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	return declarationExistenceResult(resolveSchemaFactLocator(expectation.locator, schema), expectation.expected)
}

func (expectation factExistenceExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return declarationExistenceResult(resolveReferenceFactLocator(expectation.locator, reference), expectation.expected)
}

func (expectation factExistenceExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return declarationExistenceResult(resolveInspectionFactLocator(expectation.locator, inspection), expectation.expected)
}

func (valueExistenceExpectationValue) schemaExpectation() {}

func (valueExistenceExpectationValue) referenceExpectation() {}

func (expectation valueExistenceExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	return declarationExistenceResult(resolveSchemaValueLocator(expectation.locator, schema), expectation.expected)
}

func (expectation valueExistenceExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return declarationExistenceResult(resolveReferenceValueLocator(expectation.locator, reference), expectation.expected)
}

func (factHasResolverExpectationValue) schemaExpectation() {}

func (factHasResolverExpectationValue) referenceExpectation() {}

func (factHasResolverExpectationValue) inspectionExpectation() {}

func (expectation factHasResolverExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaFactLocator(expectation.locator, schema)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("has resolver"), true, resolution.value.HasResolver)
}

func (expectation factHasResolverExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceFactLocator(expectation.locator, reference)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("has resolver"), true, resolution.value.HasResolver)
}

func (expectation factHasResolverExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionFactLocator(expectation.locator, inspection)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("has resolver"), true, resolution.value.HasResolver)
}

func (valueRequiredStaticExpectationValue) schemaExpectation() {}

func (valueRequiredStaticExpectationValue) referenceExpectation() {}

func (expectation valueRequiredStaticExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaValueLocator(expectation.locator, schema)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("required"), true, resolution.value.Required)
}

func (expectation valueRequiredStaticExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceValueLocator(expectation.locator, reference)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("required"), true, resolution.value.Required)
}

func (valueSensitiveStaticExpectationValue) schemaExpectation() {}

func (valueSensitiveStaticExpectationValue) referenceExpectation() {}

func (expectation valueSensitiveStaticExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaValueLocator(expectation.locator, schema)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, semanticRequirementSensitive, true, resolution.value.Sensitive)
}

func (expectation valueSensitiveStaticExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceValueLocator(expectation.locator, reference)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, semanticRequirementSensitive, true, resolution.value.Sensitive)
}

func (valueOriginStaticExpectationValue) schemaExpectation() {}

func (valueOriginStaticExpectationValue) referenceExpectation() {}

func (expectation valueOriginStaticExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaValueLocator(expectation.locator, schema)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (expectation valueOriginStaticExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceValueLocator(expectation.locator, reference)
	actual := resolution.value.Origin
	return declarationResolvedComparison(resolution.status, qualifySubject(resolution.subject, evaluationText("origin")), resolution.detail, evaluationText(fmt.Sprintf("origin %v", expectation.expected)), evaluationText(fmt.Sprintf("origin %v", actual)), actual == expectation.expected)
}

func (valueCorePropertyExpectation) schemaExpectation() {}

func (valueCorePropertyExpectation) referenceExpectation() {}

func (valueCorePropertyExpectation) inspectionExpectation() {}

func (expectation valueCorePropertyExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaValueLocator(expectation.locator, schema)
	return expectation.evaluate(resolution.status, resolution.subject, resolution.detail, resolution.value.Kind, resolution.value.HasType, resolution.value.Type)
}

func (expectation valueCorePropertyExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceValueLocator(expectation.locator, reference)
	return expectation.evaluate(resolution.status, resolution.subject, resolution.detail, resolution.value.Kind, resolution.value.HasType, resolution.value.Type)
}

func (expectation valueCorePropertyExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionValueLocator(expectation.locator, inspection)
	return expectation.evaluate(resolution.status, resolution.subject, resolution.detail, resolution.value.Kind, resolution.value.HasType, resolution.value.Type)
}

func (expectation valueCorePropertyExpectation) evaluate(status locatorResolutionStatus, subject, detail evaluationField, kind cli.ValueDestinationKind, hasType bool, valueType cli.ValueType) evaluationResult {
	switch expectation.kind {
	case valueCorePropertyDestinationKind:
		return declarationResolvedComparison(status, subject, detail, evaluationText("kind"), evaluationText(fmt.Sprint(kind)), kind == expectation.destinationKind)
	case valueCorePropertyType:
		if status != locatorResolutionFound {
			return declarationResolvedComparison(status, subject, detail, evaluationText("typed value"), evaluationText(""), false)
		}
		if !hasType {
			field := valueTypeField{
				value: expectation.valueType,
			}
			return disprovedResult(subject, property(evaluationText("type"), field), evaluationText("has-type=false"))
		}
		field := valueTypeField{
			value: expectation.valueType,
		}
		observedField := valueTypeField{
			value: valueType,
		}
		return equalFieldResult(subject, evaluationText("type"), field, observedField, valueType == expectation.valueType)
	case valueCorePropertyNoType:
		return declarationResolvedComparison(status, subject, detail, evaluationText("has-type"), evaluationText(strconv.FormatBool(hasType)), !hasType)
	default:
		panic("clitest: invalid value core property expectation")
	}
}

func (valueDependenciesExpectation) schemaExpectation() {}

func (valueDependenciesExpectation) referenceExpectation() {}

func (expectation valueDependenciesExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolved := resolveSchemaValueLocator(expectation.locator, schema)
	const expected = "exact static semantic dependencies in order"
	if resolved.status != locatorResolutionFound {
		return declarationMissing(resolved.status, semanticSubjectValue, evaluationText(expected), resolved.detail, false)
	}
	dependencies, status, detail := resolveSchemaValueStates(expectation.dependencies, schema)
	if status != locatorResolutionFound {
		return declarationMissing(status, semanticSubjectValue, evaluationText(expected), detail, false)
	}
	if slices.Equal(resolved.value.Dependencies, dependencies) {
		return provedResult()
	}
	return disprovedResult(semanticSubjectValue, evaluationText(expected), evaluationText("different semantic dependency sequence"))
}

func (expectation valueDependenciesExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolved := resolveReferenceValueLocator(expectation.locator, reference)
	const expected = "exact static semantic dependencies in order"
	if resolved.status != locatorResolutionFound {
		return declarationMissing(resolved.status, semanticSubjectValue, evaluationText(expected), resolved.detail, false)
	}
	dependencies, status, detail := resolveReferenceValueStates(expectation.dependencies, reference)
	if status != locatorResolutionFound {
		return declarationMissing(status, semanticSubjectValue, evaluationText(expected), detail, false)
	}
	if slices.Equal(resolved.value.Dependencies, dependencies) {
		return provedResult()
	}
	return disprovedResult(semanticSubjectValue, evaluationText(expected), evaluationText("different semantic dependency sequence"))
}

func (valueTypeField) evaluationField() {}

func (valueDiagnosticsExactlyExpectation) inspectionExpectation() {}

func (expectation valueDiagnosticsExactlyExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveValue(expectation.locator, inspection)
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.label, evaluationText("exact attached diagnostics in public order"), resolved.actual)
	}
	return evaluateDiagnosticIndexesExactly(resolved.label, resolved.value.DiagnosticIndexes, expectation.expected, inspection)
}

func (valueNoRepresentationExpectation) inspectionExpectation() {}

func (expectation valueNoRepresentationExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveValue(expectation.locator, inspection)
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.label, evaluationText("representation absent"), resolved.actual)
	}
	if resolved.value.Representation == nil {
		return provedResult()
	}
	return disprovedResult(resolved.label, evaluationText("representation absent"), evaluationText("representation is present"))
}

func (valuePropertyExpectation) inspectionExpectation() {}

func (expectation valuePropertyExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveValue(expectation.locator, inspection)
	expected := expectation.expectedField()
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.label, expected, resolved.actual)
	}

	var matched bool
	var actual evaluationField
	secret := resolved.value.Sensitive

	switch expectation.kind {
	case valuePropertyState:
		matched = resolved.value.State == expectation.state
		actual = evaluationText(fmt.Sprintf("resolution state %v", resolved.value.State))
	case valuePropertySensitive:
		matched = resolved.value.Sensitive
		actual = evaluationText(fmt.Sprintf("sensitive=%t", resolved.value.Sensitive))
	case valuePropertyText:
		secret = true
		if resolved.value.Representation == nil {
			actual = semanticRequirementNoCanonicalRepresentation
			break
		}
		matched = resolved.value.Representation.Kind == cli.ValueRepresentationText && resolved.value.Representation.Text == expectation.text && !resolved.value.Representation.Redacted
		actual = evaluationText("canonical text differs")
	case valuePropertyTokens:
		secret = true
		if resolved.value.Representation == nil {
			actual = semanticRequirementNoCanonicalRepresentation
			break
		}
		matched = resolved.value.Representation.Kind == cli.ValueRepresentationTokens && slices.Equal(resolved.value.Representation.Tokens, expectation.tokens) && !resolved.value.Representation.Redacted
		actual = evaluationText("canonical tokens differ")
	case valuePropertyRedacted:
		matched = resolved.value.Representation != nil && resolved.value.Representation.Redacted
		actual = evaluationText("canonical representation is not redacted")
	case valuePropertyContributedBy:
		for _, id := range resolved.value.Contributors {
			candidate, ok := inspection.ValueCandidates[id]
			if !ok || candidate.ProducerIndex < 0 || candidate.ProducerIndex >= len(resolved.value.Producers) {
				continue
			}
			if resolved.value.Producers[candidate.ProducerIndex].Kind == expectation.producerKind {
				matched = true
				break
			}
		}
		actual = evaluationText("no contributing candidate has that producer kind")
	default:
		panic("clitest: invalid value property expectation")
	}

	if matched {
		return provedResult()
	}
	if secret {
		return disprovedResult(resolved.label, evaluationText("redacted expected value property"), evaluationText("redacted observed value property"))
	}
	return disprovedResult(resolved.label, expected, actual)
}

func (expectation valuePropertyExpectation) expectedField() evaluationField {
	switch expectation.kind {
	case valuePropertyState:
		return evaluationText(fmt.Sprintf("resolution state %v", expectation.state))
	case valuePropertySensitive:
		return semanticRequirementSensitive
	case valuePropertyText:
		return subjectNamed(evaluationText("text"), expectation.text)
	case valuePropertyTokens:
		return semanticRequirementExactCanonicalTokens
	case valuePropertyRedacted:
		return evaluationText("redacted canonical representation")
	case valuePropertyContributedBy:
		return evaluationText(fmt.Sprintf("contributed by producer kind %v", expectation.producerKind))
	default:
		panic("clitest: invalid value property expectation")
	}
}

func (valueObservationExpectationValue) inspectionExpectation() {}

func (expectation valueObservationExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveValue(expectation.locator, inspection)
	expect := "observed"
	if !expectation.observed {
		expect = "not observed"
	}
	switch resolved.status {
	case locatorResolutionFound:
		if expectation.observed {
			return provedResult()
		}
		return disprovedResult(resolved.label, evaluationText(expect), evaluationText("value is observed"))
	case locatorResolutionAbsent:
		if !expectation.observed {
			return provedResult()
		}
		return disprovedResult(resolved.label, evaluationText(expect), resolved.actual)
	case locatorResolutionNotProvable:
		return notProvableResult(resolved.label, evaluationText(expect), resolved.actual)
	default:
		return disprovedResult(resolved.label, evaluationText(expect), resolved.actual)
	}
}

func (valueStateIfObservedExpectationValue) inspectionExpectation() {}

func (expectation valueStateIfObservedExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	if len(expectation.states) == 0 {
		return disprovedResult(evaluationText("value state if observed"), evaluationText("at least one allowed resolution state"), evaluationText("no allowed states"))
	}
	resolved := resolveValue(expectation.locator, inspection)
	expected := property(evaluationText("state if observed in"), resolutionStateCollectionField{
		values: expectation.states,
	})
	switch resolved.status {
	case locatorResolutionAbsent:
		return provedResult()
	case locatorResolutionNotProvable:
		return notProvableResult(resolved.label, expected, resolved.actual)
	case locatorResolutionFound:
		if slices.Contains(expectation.states, resolved.value.State) {
			return provedResult()
		}
		return disprovedResult(resolved.label, expected, evaluationText(fmt.Sprintf("resolution state %v", resolved.value.State)))
	default:
		return disprovedResult(resolved.label, expected, resolved.actual)
	}
}

func (valueBlockedByExpectationValue) inspectionExpectation() {}

func (expectation valueBlockedByExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	value := resolveValue(expectation.value, inspection)
	expected := evaluationText("blocked by requested value")
	if value.status != locatorResolutionFound {
		return outcomeForValueStatus(value.status, value.label, expected, value.actual)
	}

	blockerID, status, detail := resolveInspectionValueID(expectation.blocker, inspection)
	if status == locatorResolutionNotProvable {
		return notProvableResult(value.label, expected, detail)
	}
	if status != locatorResolutionFound {
		if status == locatorResolutionInvalid || emptyEvaluationField(detail) {
			detail = evaluationText("invalid blocker locator")
		}
		return disprovedResult(value.label, expected, detail)
	}
	if slices.Contains(value.value.BlockedBy, blockerID) {
		return provedResult()
	}
	field := collectionCardinalityField{
		name:  "blocked-by IDs",
		count: len(value.value.BlockedBy),
	}
	return disprovedResult(
		value.label,
		evaluationText(fmt.Sprintf("blocked by value ID %d", blockerID)),
		field,
	)
}

func (producerPropertyExpectation) inspectionExpectation() {}

func (expectation producerPropertyExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveProducer(expectation.locator, inspection)
	expected := expectation.expectedField()
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.label, expected, resolved.actual)
	}

	var matched bool
	var actual evaluationField

	switch expectation.kind {
	case valueProducerPropertyProducerKind:
		matched = resolved.value.Kind == expectation.producerKind
		actual = evaluationText(fmt.Sprintf("producer kind %v", resolved.value.Kind))
	case valueProducerPropertyTier:
		matched = resolved.value.Tier == expectation.tier
		actual = evaluationText(fmt.Sprintf("tier %v", resolved.value.Tier))
	case valueProducerPropertyOutcome:
		matched = resolved.value.Outcome == expectation.outcome
		actual = evaluationText(fmt.Sprintf("outcome %v", resolved.value.Outcome))
	case valueProducerPropertySensitive:
		matched = resolved.value.Sensitive
		actual = evaluationText(fmt.Sprintf("sensitive=%t", resolved.value.Sensitive))
	case valueProducerPropertyBlockedBy, valueProducerPropertyDependencies:
		expectedIDs, status, detail := expectedValueIDsInspection(expectation.values, inspection)
		if status != locatorResolutionFound {
			if status == locatorResolutionInvalid || emptyEvaluationField(detail) {
				detail = evaluationText("invalid value locator in expected collection")
			}
			actual = detail
			break
		}
		actualIDs := resolved.value.BlockedBy
		label := "blocked by"
		if expectation.kind == valueProducerPropertyDependencies {
			actualIDs = resolved.value.Dependencies
			label = semanticNameDependencies
		}
		matched = slices.Equal(actualIDs, expectedIDs)
		actual = collectionCardinalityField{
			name:  label + " IDs",
			count: len(actualIDs),
		}
	case valueProducerPropertyOrigins:
		if !expectation.originsValid {
			actual = observed(semanticSubjectOrigin, observationInvalidLocatorInExpectedCollection)
			break
		}
		matched = slices.Equal(resolved.value.Origins, expectation.origins)
		actual = collectionCardinalityField{
			name:  semanticNameOriginIDs,
			count: len(resolved.value.Origins),
		}
	default:
		panic("clitest: invalid value producer property expectation")
	}

	if matched {
		return provedResult()
	}
	if resolved.value.Sensitive {
		return disprovedResult(resolved.label, evaluationText("redacted expected producer property"), evaluationText("redacted observed producer property"))
	}
	return disprovedResult(resolved.label, expected, actual)
}

func (expectation producerPropertyExpectation) expectedField() evaluationField {
	switch expectation.kind {
	case valueProducerPropertyProducerKind:
		return evaluationText(fmt.Sprintf("producer kind %v", expectation.producerKind))
	case valueProducerPropertyTier:
		return evaluationText(fmt.Sprintf("tier %v", expectation.tier))
	case valueProducerPropertyOutcome:
		return evaluationText(fmt.Sprintf("outcome %v", expectation.outcome))
	case valueProducerPropertySensitive:
		return semanticRequirementSensitive
	case valueProducerPropertyBlockedBy:
		return evaluationText("exact blocked by")
	case valueProducerPropertyDependencies:
		return evaluationText("exact dependencies")
	case valueProducerPropertyOrigins:
		return semanticRequirementExactOrigins
	default:
		panic("clitest: invalid value producer property expectation")
	}
}

func (candidatePropertyExpectation) inspectionExpectation() {}

func (expectation candidatePropertyExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveCandidate(expectation.locator, inspection)
	expected := expectation.expectedField()
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.label, expected, resolved.actual)
	}

	var matched bool
	var actual evaluationField
	secret := resolved.value.Sensitive

	switch expectation.kind {
	case valueCandidatePropertyDisposition:
		matched = resolved.value.Disposition == expectation.disposition
		actual = evaluationText(fmt.Sprintf("disposition %v", resolved.value.Disposition))
	case valueCandidatePropertyContribution:
		matched = resolved.value.Contribution == expectation.contribution
		actual = evaluationText(fmt.Sprintf("contribution %v", resolved.value.Contribution))
	case valueCandidatePropertySensitive:
		matched = resolved.value.Sensitive
		actual = evaluationText(fmt.Sprintf("sensitive=%t", resolved.value.Sensitive))
	case valueCandidatePropertyOrigins:
		if !expectation.originsValid {
			actual = observed(semanticSubjectOrigin, observationInvalidLocatorInExpectedCollection)
			break
		}
		matched = slices.Equal(resolved.value.Origins, expectation.origins)
		actual = collectionCardinalityField{
			name:  semanticNameOriginIDs,
			count: len(resolved.value.Origins),
		}
	case valueCandidatePropertySuppression:
		matched = resolved.value.Suppression == expectation.suppression
		actual = evaluationText(fmt.Sprintf("suppression %v", resolved.value.Suppression))
	case valueCandidatePropertyText:
		secret = true
		if resolved.value.Representation == nil {
			actual = semanticRequirementNoCanonicalRepresentation
			break
		}
		matched = resolved.value.Representation.Kind == cli.ValueRepresentationText && resolved.value.Representation.Text == expectation.text && !resolved.value.Representation.Redacted
		actual = evaluationText("candidate canonical text differs")
	case valueCandidatePropertyTokens:
		secret = true
		if resolved.value.Representation == nil {
			actual = semanticRequirementNoCanonicalRepresentation
			break
		}
		matched = resolved.value.Representation.Kind == cli.ValueRepresentationTokens && slices.Equal(resolved.value.Representation.Tokens, expectation.tokens) && !resolved.value.Representation.Redacted
		actual = evaluationText("candidate canonical tokens differ")
	case valueCandidatePropertyElementCount:
		matched = len(resolved.value.Elements) == expectation.count
		actual = evaluationText(fmt.Sprintf("element count %d", len(resolved.value.Elements)))
	case valueCandidatePropertyEntryCount:
		matched = len(resolved.value.Entries) == expectation.count
		actual = evaluationText(fmt.Sprintf("entry count %d", len(resolved.value.Entries)))
	case valueCandidatePropertySuppressedBy, valueCandidatePropertyConflictsWith:
		expectedIDs := make([]cli.ValueCandidateID, len(expectation.candidates))
		for index, state := range expectation.candidates {
			id, ok := candidateLocatorID(state, inspection)
			if !ok {
				actual = evaluationText("expected candidate locator cannot be resolved")
				return disprovedResult(resolved.label, expected, actual)
			}
			expectedIDs[index] = id
		}
		actualIDs := resolved.value.SuppressedBy
		label := "suppressed by"
		if expectation.kind == valueCandidatePropertyConflictsWith {
			actualIDs = resolved.value.ConflictsWith
			label = "conflicts with"
		}
		matched = slices.Equal(actualIDs, expectedIDs)
		actual = collectionCardinalityField{
			name:  label + " candidate IDs",
			count: len(actualIDs),
		}
	case valueCandidatePropertyInputText:
		if resolved.value.Input == nil {
			actual = observed(semanticSubjectRawInput, observationAbsent)
			break
		}
		matched = resolved.value.Input.Text == expectation.text
		switch {
		case matched:
			actual = evaluationText("")
		case resolved.value.Input.Redacted:
			secret = true
			actual = evaluationText("raw input text is <redacted>")
		default:
			actual = subjectNamed(evaluationText("raw input text"), resolved.value.Input.Text)
		}
	case valueCandidatePropertyInputRedacted:
		if resolved.value.Input == nil {
			actual = observed(semanticSubjectRawInput, observationAbsent)
			break
		}
		matched = resolved.value.Input.Redacted
		actual = evaluationText(fmt.Sprintf("raw input redacted=%t", resolved.value.Input.Redacted))
	case valueCandidatePropertyRedacted:
		if resolved.value.Representation == nil {
			actual = evaluationText("canonical representation is absent")
			break
		}
		matched = resolved.value.Representation.Redacted
		actual = evaluationText(fmt.Sprintf("canonical representation redacted=%t", resolved.value.Representation.Redacted))
	default:
		panic("clitest: invalid value candidate property expectation")
	}

	switch {
	case matched:
		return provedResult()
	case secret:
		return disprovedResult(resolved.label, evaluationText("redacted expected candidate property"), evaluationText("redacted observed candidate property"))
	default:
		return disprovedResult(resolved.label, expected, actual)
	}
}

func (expectation candidatePropertyExpectation) expectedField() evaluationField {
	switch expectation.kind {
	case valueCandidatePropertyDisposition:
		return evaluationText(fmt.Sprintf("disposition %v", expectation.disposition))
	case valueCandidatePropertyContribution:
		return evaluationText(fmt.Sprintf("contribution %v", expectation.contribution))
	case valueCandidatePropertySensitive:
		return semanticRequirementSensitive
	case valueCandidatePropertyOrigins:
		return semanticRequirementExactOrigins
	case valueCandidatePropertySuppression:
		return evaluationText(fmt.Sprintf("suppression %v", expectation.suppression))
	case valueCandidatePropertyText:
		return subjectNamed(evaluationText("text"), expectation.text)
	case valueCandidatePropertyTokens:
		return semanticRequirementExactCanonicalTokens
	case valueCandidatePropertyElementCount:
		return evaluationText(fmt.Sprintf("element count %d", expectation.count))
	case valueCandidatePropertyEntryCount:
		return evaluationText(fmt.Sprintf("entry count %d", expectation.count))
	case valueCandidatePropertySuppressedBy:
		return evaluationText("exact suppressed by")
	case valueCandidatePropertyConflictsWith:
		return evaluationText("exact conflicts with")
	case valueCandidatePropertyInputText:
		return subjectNamed(evaluationText("raw input text"), expectation.text)
	case valueCandidatePropertyInputRedacted:
		return evaluationText("raw input redacted")
	case valueCandidatePropertyRedacted:
		return evaluationText("canonical representation redacted")
	default:
		panic("clitest: invalid value candidate property expectation")
	}
}

func (valueCandidateProducedByExpectationValue) inspectionExpectation() {}

func (expectation valueCandidateProducedByExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	candidate := resolveCandidate(expectation.candidate, inspection)
	expected := evaluationText("produced by requested producer")
	if candidate.status != locatorResolutionFound {
		return outcomeForValueStatus(candidate.status, candidate.label, expected, candidate.actual)
	}

	producer := resolveProducer(expectation.producer, inspection)
	if producer.status == locatorResolutionNotProvable {
		return notProvableResult(candidate.label, expected, producer.actual)
	}
	if producer.status != locatorResolutionFound {
		return disprovedResult(candidate.label, expected, producer.actual)
	}
	parent := resolveValue(expectation.producer.value, inspection)
	if parent.status == locatorResolutionNotProvable {
		return notProvableResult(candidate.label, expected, parent.actual)
	}
	if parent.status != locatorResolutionFound {
		return disprovedResult(candidate.label, expected, parent.actual)
	}
	if candidate.value.Destination == parent.value.ID && candidate.value.ProducerIndex == producer.index {
		return provedResult()
	}
	return disprovedResult(
		candidate.label,
		evaluationText(fmt.Sprintf("producer for value ID %d at index %d", parent.value.ID, producer.index)),
		evaluationText(fmt.Sprintf("producer for value ID %d at index %d", candidate.value.Destination, candidate.value.ProducerIndex)),
	)
}

func (valueCandidateDeclarationExpectationValue) inspectionExpectation() {}

func (expectation valueCandidateDeclarationExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	candidate := resolveCandidate(expectation.candidate, inspection)
	expectedRequirement := evaluationText("requested declaration identity")
	if candidate.status != locatorResolutionFound {
		return outcomeForValueStatus(candidate.status, candidate.label, expectedRequirement, candidate.actual)
	}

	expected, status, detail := resolveExpectedDeclarationIdentity(expectation.declaration, nil, nil, inspection)
	if status == locatorResolutionNotProvable {
		return notProvableResult(candidate.label, expectedRequirement, detail)
	}
	if status != locatorResolutionFound {
		if emptyEvaluationField(detail) {
			detail = observed(semanticSubjectDeclaration, observationInvalidLocator)
		}
		return disprovedResult(candidate.label, expectedRequirement, detail)
	}
	if candidate.value.Declaration == expected {
		return provedResult()
	}
	field := declarationIdentityField{
		value: expected,
	}
	declarationIdentityField := declarationIdentityField{
		value: candidate.value.Declaration,
	}
	return disprovedResult(
		candidate.label,
		field,
		declarationIdentityField,
	)
}

func (componentPropertyExpectation) inspectionExpectation() {}

func (expectation componentPropertyExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveComponent(expectation.locator, inspection)
	expected := expectation.expectedField()
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.label, expected, resolved.actual)
	}

	var matched bool
	var actual evaluationField
	secret := resolved.value.Sensitive

	switch expectation.kind {
	case valueComponentPropertyText:
		secret = true
		if resolved.value.Representation == nil {
			actual = semanticRequirementNoCanonicalRepresentation
			break
		}
		matched = resolved.value.Representation.Kind == cli.ValueRepresentationText && resolved.value.Representation.Text == expectation.text && !resolved.value.Representation.Redacted
		actual = evaluationText("component canonical text differs")
	case valueComponentPropertyTokens:
		secret = true
		if resolved.value.Representation == nil {
			actual = semanticRequirementNoCanonicalRepresentation
			break
		}
		matched = resolved.value.Representation.Kind == cli.ValueRepresentationTokens && slices.Equal(resolved.value.Representation.Tokens, expectation.tokens) && !resolved.value.Representation.Redacted
		actual = evaluationText("component canonical tokens differ")
	case valueComponentPropertyRedacted:
		matched = resolved.value.Representation != nil && resolved.value.Representation.Redacted
		actual = evaluationText("canonical representation is not redacted")
	case valueComponentPropertyOrigins:
		if !expectation.originsValid {
			actual = observed(semanticSubjectOrigin, observationInvalidLocatorInExpectedCollection)
			break
		}
		matched = slices.Equal(resolved.value.Origins, expectation.origins)
		actual = collectionCardinalityField{
			name:  semanticNameOriginIDs,
			count: len(resolved.value.Origins),
		}
	case valueComponentPropertySensitive:
		matched = resolved.value.Sensitive
		actual = evaluationText(fmt.Sprintf("sensitive=%t", resolved.value.Sensitive))
	case valueComponentPropertyInputText:
		if resolved.value.Input == nil {
			actual = observed(semanticSubjectRawInput, observationAbsent)
			break
		}
		matched = resolved.value.Input.Text == expectation.text
		switch {
		case matched:
			actual = evaluationText("")
		case resolved.value.Input.Redacted:
			secret = true
			actual = evaluationText("raw input text is <redacted>")
		default:
			actual = subjectNamed(evaluationText("raw input text"), resolved.value.Input.Text)
		}
	default:
		panic("clitest: invalid value component property expectation")
	}

	switch {
	case matched:
		return provedResult()
	case secret:
		return disprovedResult(resolved.label, evaluationText("redacted expected component property"), evaluationText("redacted observed component property"))
	default:
		return disprovedResult(resolved.label, expected, actual)
	}
}

func (expectation componentPropertyExpectation) expectedField() evaluationField {
	switch expectation.kind {
	case valueComponentPropertyText:
		return subjectNamed(evaluationText("text"), expectation.text)
	case valueComponentPropertyTokens:
		return semanticRequirementExactCanonicalTokens
	case valueComponentPropertyRedacted:
		return evaluationText("redacted canonical representation")
	case valueComponentPropertyOrigins:
		return semanticRequirementExactOrigins
	case valueComponentPropertySensitive:
		return semanticRequirementSensitive
	case valueComponentPropertyInputText:
		return subjectNamed(evaluationText("raw input text"), expectation.text)
	default:
		panic("clitest: invalid value component property expectation")
	}
}

func (valueCollectionExpectation) inspectionExpectation() {}

func (expectation valueCollectionExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveValue(expectation.locator, inspection)
	expected := expectation.expectedField()
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.label, expected, resolved.actual)
	}
	if inspection.Truncated {
		return notProvableResult(resolved.label, expected, observed(semanticSubjectInspection, observationTruncated))
	}

	var matched bool
	var actual evaluationField

	switch expectation.kind {
	case valueCollectionProducers:
		if len(resolved.value.Producers) != len(expectation.producers) {
			actual = evaluationText(fmt.Sprintf("producer count %d", len(resolved.value.Producers)))
			break
		}
		matched = true
		for index, state := range expectation.producers {
			producer := resolveProducer(state, inspection)
			parent := resolveValue(state.value, inspection)
			if producer.status != locatorResolutionFound || producer.index != index || parent.status != locatorResolutionFound || parent.value.ID != resolved.value.ID {
				matched = false
				actual = evaluationText(fmt.Sprintf("producer locator at expected index %d does not identify that producer", index))
				break
			}
		}
	case valueCollectionCandidates, valueCollectionContributors:
		expectedIDs := make([]cli.ValueCandidateID, len(expectation.candidates))
		unresolved := "expected candidate locator cannot be resolved"
		if expectation.kind == valueCollectionContributors {
			unresolved = "expected contributor locator cannot be resolved"
		}
		for index, state := range expectation.candidates {
			id, ok := candidateLocatorID(state, inspection)
			if !ok {
				actual = evaluationText(unresolved)
				return disprovedResult(resolved.label, expected, actual)
			}
			expectedIDs[index] = id
		}
		actualIDs := resolved.value.Candidates
		label := semanticNameCandidate
		if expectation.kind == valueCollectionContributors {
			actualIDs = resolved.value.Contributors
			label = "contributor"
		}
		matched = slices.Equal(actualIDs, expectedIDs)
		actual = collectionCardinalityField{
			name:  label + " IDs",
			count: len(actualIDs),
		}
	case valueCollectionElements:
		expectedReferences := make([]cli.ValueElementReference, len(expectation.elements))
		for index, element := range expectation.elements {
			reference, ok := componentReferenceFromElementLocator(element, inspection)
			if !ok {
				actual = evaluationText("expected element locator cannot be resolved")
				return disprovedResult(resolved.label, expected, actual)
			}
			expectedReferences[index] = reference
		}
		matched = slices.Equal(resolved.value.Elements, expectedReferences)
		actual = collectionCardinalityField{
			name:  "element references",
			count: len(resolved.value.Elements),
		}
	case valueCollectionEntries:
		expectedReferences := make([]cli.ValueEntryReference, len(expectation.entries))
		for index, entry := range expectation.entries {
			reference, ok := entryReferenceFromLocator(entry, inspection)
			if !ok {
				actual = evaluationText("expected entry locator cannot be resolved")
				return disprovedResult(resolved.label, expected, actual)
			}
			expectedReferences[index] = reference
		}
		matched = slices.Equal(resolved.value.Entries, expectedReferences)
		actual = collectionCardinalityField{
			name:  "entry references",
			count: len(resolved.value.Entries),
		}
	default:
		panic("clitest: invalid value collection expectation")
	}

	if matched {
		return provedResult()
	}
	return disprovedResult(resolved.label, expected, actual)
}

func (expectation valueCollectionExpectation) expectedField() evaluationField {
	switch expectation.kind {
	case valueCollectionProducers:
		return evaluationText("exact producers in authoritative order")
	case valueCollectionCandidates:
		return evaluationText("exact candidates in authoritative order")
	case valueCollectionContributors:
		return evaluationText("exact contributors in authoritative order")
	case valueCollectionElements:
		return evaluationText("exact resolved elements in authoritative order")
	case valueCollectionEntries:
		return evaluationText("exact resolved entries in authoritative order")
	default:
		panic("clitest: invalid value collection expectation")
	}
}

func (domainPropertyExpectation) inspectionExpectation() {}

func (expectation domainPropertyExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveDomain(expectation.locator, inspection)
	expected := expectation.expectedField()
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.label, expected, resolved.actual)
	}

	var matched bool
	var actual evaluationField

	switch expectation.kind {
	case valueDomainPropertyOutcome:
		matched = resolved.value.Outcome == expectation.outcome
		actual = evaluationText(fmt.Sprintf("outcome %v", resolved.value.Outcome))
	case valueDomainPropertyInvocationScoped:
		matched = resolved.value.InvocationScoped
		actual = evaluationText(fmt.Sprintf("invocation-scoped=%t", resolved.value.InvocationScoped))
	case valueDomainPropertyDependencies:
		expectedIDs, status, detail := expectedValueIDsInspection(expectation.values, inspection)
		if status != locatorResolutionFound {
			if status == locatorResolutionInvalid || emptyEvaluationField(detail) {
				detail = evaluationText("invalid value locator in expected collection")
			}
			actual = detail
			break
		}
		matched = slices.Equal(resolved.value.Dependencies, expectedIDs)
		actual = collectionCardinalityField{
			name:  "dependency IDs",
			count: len(resolved.value.Dependencies),
		}
	default:
		panic("clitest: invalid value domain property expectation")
	}

	if matched {
		return provedResult()
	}
	return disprovedResult(resolved.label, expected, actual)
}

func (expectation domainPropertyExpectation) expectedField() evaluationField {
	switch expectation.kind {
	case valueDomainPropertyOutcome:
		return evaluationText(fmt.Sprintf("outcome %v", expectation.outcome))
	case valueDomainPropertyInvocationScoped:
		return evaluationText("invocation scoped")
	case valueDomainPropertyDependencies:
		return evaluationText("exact dependencies in authoritative order")
	default:
		panic("clitest: invalid value domain property expectation")
	}
}

func (valueCandidateDomainEntryExpectationValue) inspectionExpectation() {}

func (expectation valueCandidateDomainEntryExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	candidate := resolveCandidate(expectation.candidate, inspection)
	expectedRequirement := evaluationText("requested value-domain entry")
	if candidate.status != locatorResolutionFound {
		return outcomeForValueStatus(candidate.status, candidate.label, expectedRequirement, candidate.actual)
	}

	entryID, status, detail := expectedInspectionValueDomainEntryID(expectation.entry, inspection)
	if status == locatorResolutionNotProvable {
		return notProvableResult(candidate.label, expectedRequirement, detail)
	}
	if status != locatorResolutionFound {
		if emptyEvaluationField(detail) {
			detail = evaluationText("invalid value-domain-entry locator")
		}
		return disprovedResult(candidate.label, expectedRequirement, detail)
	}
	if candidate.value.DomainEntry == entryID {
		return provedResult()
	}
	return disprovedResult(
		candidate.label,
		evaluationText(fmt.Sprintf("domain-entry ID %d", entryID)),
		evaluationText(fmt.Sprintf("domain-entry ID %d", candidate.value.DomainEntry)),
	)
}

func (componentDomainEntryExpectation) inspectionExpectation() {}

func (expectation componentDomainEntryExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	component := resolveComponent(expectation.locator, inspection)
	expectedRequirement := evaluationText("requested value-domain entry")
	if component.status != locatorResolutionFound {
		return outcomeForValueStatus(component.status, component.label, expectedRequirement, component.actual)
	}

	entryID, status, detail := expectedInspectionValueDomainEntryID(expectation.entry, inspection)
	if status == locatorResolutionNotProvable {
		return notProvableResult(component.label, expectedRequirement, detail)
	}
	if status != locatorResolutionFound {
		if emptyEvaluationField(detail) {
			detail = evaluationText("invalid value-domain-entry locator")
		}
		return disprovedResult(component.label, expectedRequirement, detail)
	}
	if component.value.DomainEntry == entryID {
		return provedResult()
	}
	return disprovedResult(
		component.label,
		evaluationText(fmt.Sprintf("domain-entry ID %d", entryID)),
		evaluationText(fmt.Sprintf("domain-entry ID %d", component.value.DomainEntry)),
	)
}
