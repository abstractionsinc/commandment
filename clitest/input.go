package clitest

import (
	"fmt"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
)

// TokenOwner is the closed set of public semantic subjects that can own an
// input token in expectation relationships.
type TokenOwner interface{ tokenOwner() }

// InputSourceLocator identifies one public input source for clitest semantic
// expectation. Its zero value is invalid.
type InputSourceLocator struct {
	_     [0]func()
	state *inputSourceLocatorState
}

// InputOriginLocator identifies one public input origin for clitest semantic
// expectation. Its zero value is invalid.
type InputOriginLocator struct {
	_     [0]func()
	state *inputOriginLocatorState
}

// TokenLocator identifies one analyzed input token for clitest semantic expectation.
// Its zero value is invalid.
type TokenLocator struct {
	_     [0]func()
	state *tokenLocatorState
}

// InputSource returns an input-source locator selecting exactly one observed source of kind.
// InputSourceID is artifact-local, so this human locator uses the public source kind and rejects ambiguity instead of
// manufacturing cross-operation identity.
func InputSource(kind cli.InputKind) InputSourceLocator {
	return InputSourceLocator{
		state: &inputSourceLocatorState{
			mode: inputSourceLocatorKind,
			kind: kind,
		},
	}
}

// InputSourceByKey returns an input-source locator selecting exactly one observed source with kind and key.
// Key is source metadata used to disambiguate within the current detached observation; it is not a declaration
// identity.
func InputSourceByKey(kind cli.InputKind, key string) InputSourceLocator {
	return InputSourceLocator{
		state: &inputSourceLocatorState{
			mode: inputSourceLocatorKindKey,
			kind: kind,
			key:  key,
		},
	}
}

// InspectionObservedInputSourcesExactly returns an Inspection expectation that the sources of kind present in the
// returned Inspection source map have exactly the supplied distinct semantic members.
// The expectation concerns observed map membership; InputSource map iteration order is not semantic.
func InspectionObservedInputSourcesExactly(kind cli.InputKind, sources ...InputSourceLocator) InspectionExpectation {
	return inspectionObservedInputSourcesExpectation{
		kind:    kind,
		sources: slices.Clone(sources),
	}
}

// InputSourceByID returns an exact expected coordinate for the structured input source with id in the same diagnostic report;
// the ID is a caller-supplied coordinate and does not establish membership.
func InputSourceByID(id cli.InputSourceID) InputSourceLocator { return inputSourceByID(id) }

// InputOriginByID returns an exact expected coordinate for the structured input origin with id in the same diagnostic report;
// the ID is a caller-supplied coordinate and does not establish membership.
func InputOriginByID(id cli.InputOriginID) InputOriginLocator { return inputOriginByID(id) }

// Text returns an Inspection expectation that the located token has exactly expected text.
// If observed token text is redacted, mismatch reporting does not reveal that redacted text.
func (locator TokenLocator) Text(expected string) InspectionExpectation {
	return tokenTextExpectation(locator, expected)
}

// Origin returns the structured input origin referenced by the located token's Origin identity in the same Inspection
// observation.
func (locator TokenLocator) Origin() InputOriginLocator { return tokenOriginLocator(locator) }

// Role returns an Inspection expectation that the located token carries exactly role.
func (locator TokenLocator) Role(role cli.TokenRole) InspectionExpectation {
	return tokenRoleExpectation(locator, role)
}

// OwnedBy returns an Inspection expectation that the located token's structured owner relationship resolves to owner in
// the same Inspection observation.
func (locator TokenLocator) OwnedBy(owner TokenOwner) InspectionExpectation {
	return tokenOwnerExpectation(locator, owner)
}

// RejectedWith returns an Inspection expectation that the located token is rejected with exactly code as its structured
// DiagnosticCode.
func (locator TokenLocator) RejectedWith(code cli.DiagnosticCode) InspectionExpectation {
	return tokenRejectionExpectation(locator, code)
}

// MatchedSpelling returns an Inspection expectation that the located token carries exactly spelling as its matched
// spelling.
func (locator TokenLocator) MatchedSpelling(spelling string) InspectionExpectation {
	return tokenSpellingExpectation(locator, spelling)
}

// Match returns an Inspection expectation that the located token carries exactly match as its structured name-match
// result.
func (locator TokenLocator) Match(match cli.NameMatchKind) InspectionExpectation {
	return tokenMatchExpectation(locator, match)
}

// Redacted returns an Inspection expectation that the located token is explicitly redacted. It checks the carrier's
// redaction marker without exposing or reconstructing protected token text.
func (locator TokenLocator) Redacted() InspectionExpectation {
	return tokenRedactedExpectation(locator)
}

// FlagValueForm returns an Inspection expectation that the located token carries exactly form as its flag-value form.
func (locator TokenLocator) FlagValueForm(form cli.FlagValueForm) InspectionExpectation {
	return tokenFlagValueFormExpectation(locator, form)
}

// OccurrenceIndex returns an Inspection expectation that the located token carries exactly index as its flag-occurrence
// coordinate.
func (locator TokenLocator) OccurrenceIndex(index int) InspectionExpectation {
	return tokenOccurrenceIndexExpectation(locator, index)
}

// ElementIndex returns an Inspection expectation that the located token carries exactly index as its element
// coordinate.
func (locator TokenLocator) ElementIndex(index int) InspectionExpectation {
	return tokenElementIndexExpectation(locator, index)
}

// DelegatedIndex returns an Inspection expectation that the located token carries exactly index as its delegated-input
// coordinate.
func (locator TokenLocator) DelegatedIndex(index int) InspectionExpectation {
	return tokenDelegatedIndexExpectation(locator, index)
}

// Source returns the input source referenced by the located origin's structured Source identity in the same diagnostic
// report.
func (locator InputOriginLocator) Source() InputSourceLocator { return originSourceLocator(locator) }

// ParentAt returns the parent origin at index in the located origin's authoritative Parents order.
func (locator InputOriginLocator) ParentAt(index int) InputOriginLocator {
	return originParentByIndex(locator, index)
}

// Present returns an outcome expectation that the input-origin coordinate resolves in the structured diagnostic report.
func (locator InputOriginLocator) Present() OutcomeExpectation {
	return inputOriginPresentExpectation(locator)
}

// Sensitive returns an outcome expectation that the located input origin is marked sensitive. Origin properties remain
// structured facts and protected values are not reconstructed from presentation text.
func (locator InputOriginLocator) Sensitive() OutcomeExpectation {
	return inputOriginSensitiveExpectation(locator)
}

// ParentsExactly returns an outcome expectation that the located origin's parent identities exactly equal parents in
// authoritative order within the same diagnostic report.
func (locator InputOriginLocator) ParentsExactly(parents ...InputOriginLocator) OutcomeExpectation {
	return inputOriginParentsExactlyExpectation(locator, parents)
}

// Span returns an outcome expectation that the located input origin carries exactly span; sensitive-origin mismatch
// details remain redacted.
func (locator InputOriginLocator) Span(span cli.InputSpan) OutcomeExpectation {
	return inputOriginSpanExpectation(locator, span)
}

// Present returns an outcome expectation that the input-source coordinate resolves in the structured diagnostic report.
func (locator InputSourceLocator) Present() OutcomeExpectation {
	return inputSourcePresentExpectation(locator)
}

// Kind returns an outcome expectation that the located input source carries exactly kind.
func (locator InputSourceLocator) Kind(kind cli.InputKind) OutcomeExpectation {
	return inputSourceKindExpectation(locator, kind)
}

// Sensitive returns an outcome expectation that the located input source is marked sensitive. Other input-source
// property mismatches use that marker to redact protected observed values.
func (locator InputSourceLocator) Sensitive() OutcomeExpectation {
	return inputSourceSensitiveExpectation(locator)
}

// Key returns an outcome expectation that the located input source carries exactly key. If the source is sensitive,
// mismatch reporting does not expose the protected key.
func (locator InputSourceLocator) Key(key string) OutcomeExpectation {
	return inputSourceKeyExpectation(locator, key)
}

// Location returns an outcome expectation that the located input source carries exactly location. If the source is
// sensitive, mismatch reporting does not expose the protected location.
func (locator InputSourceLocator) Location(location string) OutcomeExpectation {
	return inputSourceLocationExpectation(locator, location)
}

// CommandShortcut returns an outcome expectation that the located input source references the exact CommandEntry
// identity already carried by shortcut. It does not resolve a human command-entry coordinate or treat locator origin as
// checknance.
func (locator InputSourceLocator) CommandShortcut(shortcut CommandEntryLocator) OutcomeExpectation {
	return inputSourceShortcutExpectation(locator, shortcut)
}

type sourceResolution struct {
	status locatorResolutionStatus
	value  cli.InputSource
	label  evaluationField
	actual evaluationField
}

type originResolution struct {
	status locatorResolutionStatus
	value  cli.InputOrigin
	label  evaluationField
	actual evaluationField
}

type inputSourceExpectationKind uint8

const (
	inputSourceExpectationInvalid inputSourceExpectationKind = iota
	inputSourceExpectationPresent
	inputSourceExpectationKindValue
	inputSourceExpectationSensitive
	inputSourceExpectationKey
	inputSourceExpectationLocation
	inputSourceExpectationShortcut
)

type inputSourceExpectation struct {
	locator   *inputSourceLocatorState
	kind      inputSourceExpectationKind
	inputKind cli.InputKind
	text      string
	shortcut  *commandEntryLocatorState
}

type inputOriginExpectationKind uint8

const (
	inputOriginExpectationInvalid inputOriginExpectationKind = iota
	inputOriginExpectationPresent
	inputOriginExpectationSensitive
	inputOriginExpectationParents
	inputOriginExpectationSpan
)

type inputOriginExpectation struct {
	locator *inputOriginLocatorState
	kind    inputOriginExpectationKind
	parents []*inputOriginLocatorState
	span    cli.InputSpan
}

type tokenExpectationKind uint8

const (
	tokenExpectationInvalid tokenExpectationKind = iota
	tokenExpectationRole
	tokenExpectationRejection
	tokenExpectationSpelling
	tokenExpectationMatch
	tokenExpectationRedacted
	tokenExpectationFlagValueForm
	tokenExpectationOccurrenceIndex
	tokenExpectationElementIndex
	tokenExpectationDelegatedIndex
	tokenExpectationText
)

type tokenExpectation struct {
	locator *tokenLocatorState
	kind    tokenExpectationKind

	role          cli.TokenRole
	rejection     cli.DiagnosticCode
	spelling      string
	match         cli.NameMatchKind
	flagValueForm cli.FlagValueForm
	index         int
	text          string
}

type tokenOwnerExpectationValue struct {
	token *tokenLocatorState
	owner TokenOwner
}

type inspectionObservedInputSourcesExpectation struct {
	kind    cli.InputKind
	sources []InputSourceLocator
}

type inputSourceLocatorMode uint8

const (
	inputSourceLocatorInvalid inputSourceLocatorMode = iota
	inputSourceLocatorID
	inputSourceLocatorKind
	inputSourceLocatorKindKey
	inputSourceLocatorDiagnosticSpan
	inputSourceLocatorOrigin
)

type inputSourceLocatorState struct {
	mode       inputSourceLocatorMode
	id         cli.InputSourceID
	kind       cli.InputKind
	key        string
	diagnostic *diagnosticLocatorState
	origin     *inputOriginLocatorState
}

type inputOriginLocatorMode uint8

const (
	inputOriginLocatorInvalid inputOriginLocatorMode = iota
	inputOriginLocatorID
	inputOriginLocatorParentIndex
	inputOriginLocatorToken
)

type inputOriginLocatorState struct {
	mode   inputOriginLocatorMode
	id     cli.InputOriginID
	parent *inputOriginLocatorState
	token  *tokenLocatorState
	index  int
}

type tokenLocatorState struct {
	valid bool
	index int
}

func inputSourceByID(id cli.InputSourceID) InputSourceLocator {
	return InputSourceLocator{state: &inputSourceLocatorState{
		mode: inputSourceLocatorID,
		id:   id,
	}}
}

func inputOriginByID(id cli.InputOriginID) InputOriginLocator {
	return InputOriginLocator{state: &inputOriginLocatorState{
		mode: inputOriginLocatorID,
		id:   id,
	}}
}

func originSourceLocator(origin InputOriginLocator) InputSourceLocator {
	if origin.state == nil {
		return InputSourceLocator{}
	}
	return InputSourceLocator{state: &inputSourceLocatorState{
		mode:   inputSourceLocatorOrigin,
		origin: origin.state,
	}}
}

func originParentByIndex(origin InputOriginLocator, index int) InputOriginLocator {
	return InputOriginLocator{state: &inputOriginLocatorState{
		mode:   inputOriginLocatorParentIndex,
		parent: origin.state,
		index:  index,
	}}
}

func resolveOriginID(report cli.DiagnosticReport, id cli.InputOriginID, label evaluationField) originResolution {
	var resolution originResolution
	origin, ok := report.Origins[id]
	if ok {
		resolution = originResolution{
			status: locatorResolutionFound,
			value:  origin,
			label:  label,
		}
		return resolution
	}
	if report.Truncated {
		resolution = originResolution{
			status: locatorResolutionNotProvable,
			label:  label,
			actual: observed(semanticSubjectDiagnosticReport, observationTruncated),
		}
		return resolution
	}

	resolution = originResolution{
		status: locatorResolutionAbsent,
		label:  label,
		actual: evaluationText("input origin is absent"),
	}
	return resolution
}

func resolveSource(state *inputSourceLocatorState, report cli.DiagnosticReport) sourceResolution {
	return resolveSourceObservation(state, report, nil)
}

func resolveSourceInspection(state *inputSourceLocatorState, inspection cli.Inspection) sourceResolution {
	return resolveSourceObservation(state, diagnosticReportFromInspection(inspection), &inspection)
}

func resolveSourceObservation(state *inputSourceLocatorState, report cli.DiagnosticReport, inspection *cli.Inspection) sourceResolution {
	var resolution sourceResolution
	invalid := sourceResolution{
		status: locatorResolutionInvalid,
		label:  evaluationText("input source"),
		actual: observed(semanticSubjectInputSource, observationInvalidLocator),
	}
	if state == nil {
		return invalid
	}

	switch state.mode {
	case inputSourceLocatorID:
		return resolveSourceID(report, state.id, evaluationText("input source by ID"))
	case inputSourceLocatorKind, inputSourceLocatorKindKey:
		matches := make([]cli.InputSource, 0, 1)
		for _, source := range report.Sources {
			if source.Kind != state.kind {
				continue
			}
			if state.mode == inputSourceLocatorKindKey && source.Key != state.key {
				continue
			}
			matches = append(matches, source)
		}
		var label evaluationField = evaluationText(fmt.Sprintf("input source kind %v", state.kind))
		if state.mode == inputSourceLocatorKindKey {
			label = qualifySubject(label, subjectNamed(evaluationText("key"), state.key))
		}
		switch len(matches) {
		case 1:
			if report.Truncated {
				resolution = sourceResolution{
					status: locatorResolutionNotProvable,
					label:  label,
					actual: evaluationText("diagnostic report is truncated and locator uniqueness is not established"),
				}
				return resolution
			}
			resolution = sourceResolution{
				status: locatorResolutionFound,
				value:  matches[0],
				label:  label,
			}
			return resolution
		case 0:
			if report.Truncated {
				resolution = sourceResolution{
					status: locatorResolutionNotProvable,
					label:  label,
					actual: observed(semanticSubjectDiagnosticReport, observationTruncated),
				}
				return resolution
			}
			resolution = sourceResolution{
				status: locatorResolutionAbsent,
				label:  label,
				actual: evaluationText("input source is absent"),
			}
			return resolution
		default:
			resolution = sourceResolution{
				status: locatorResolutionAmbiguous,
				label:  label,
				actual: evaluationText(fmt.Sprintf("matches %d input sources", len(matches))),
			}
			return resolution
		}
	case inputSourceLocatorDiagnosticSpan:
		if state.diagnostic == nil {
			return invalid
		}
		var diagnostic locatorResolution[diagnosticLocation]
		if inspection != nil {
			diagnostic = resolveDiagnosticStateForInspection(state.diagnostic, *inspection)
		} else {
			diagnostic = resolveDiagnosticStateForReport(state.diagnostic, report)
		}
		label := qualifySubject(diagnostic.subject, evaluationText("span source"))
		if diagnostic.status != locatorResolutionFound {
			resolution = sourceResolution{
				status: diagnostic.status,
				label:  label,
				actual: diagnostic.detail,
			}
			return resolution
		}
		if diagnostic.value.diagnostic.Span == nil {
			resolution = sourceResolution{
				status: locatorResolutionAbsent,
				label:  label,
				actual: observed(semanticSubjectDiagnosticSpan, observationAbsent),
			}
			return resolution
		}
		return resolveSourceID(report, diagnostic.value.diagnostic.Span.Source, label)
	case inputSourceLocatorOrigin:
		if state.origin == nil {
			return invalid
		}
		var origin originResolution
		if inspection != nil {
			origin = resolveOriginInspection(state.origin, *inspection)
		} else {
			origin = resolveOrigin(state.origin, report)
		}
		label := qualifySubject(origin.label, evaluationText("source"))
		if origin.status != locatorResolutionFound {
			resolution = sourceResolution{
				status: origin.status,
				label:  label,
				actual: origin.actual,
			}
			return resolution
		}
		return resolveSourceID(report, origin.value.Span.Source, label)
	default:
		return invalid
	}
}

func resolveSourceID(report cli.DiagnosticReport, id cli.InputSourceID, label evaluationField) sourceResolution {
	var resolution sourceResolution
	source, ok := report.Sources[id]
	if ok {
		resolution = sourceResolution{
			status: locatorResolutionFound,
			value:  source,
			label:  label,
		}
		return resolution
	}
	if report.Truncated {
		resolution = sourceResolution{
			status: locatorResolutionNotProvable,
			label:  label,
			actual: observed(semanticSubjectDiagnosticReport, observationTruncated),
		}
		return resolution
	}

	resolution = sourceResolution{
		status: locatorResolutionAbsent,
		label:  label,
		actual: evaluationText("input source is absent"),
	}
	return resolution
}

func resolveOrigin(state *inputOriginLocatorState, report cli.DiagnosticReport) originResolution {
	return resolveOriginObservation(state, report, nil)
}

func resolveOriginInspection(state *inputOriginLocatorState, inspection cli.Inspection) originResolution {
	return resolveOriginObservation(state, diagnosticReportFromInspection(inspection), &inspection)
}

func resolveOriginObservation(state *inputOriginLocatorState, report cli.DiagnosticReport, inspection *cli.Inspection) originResolution {
	var resolution originResolution
	invalid := originResolution{
		status: locatorResolutionInvalid,
		label:  evaluationText("input origin"),
		actual: observed(semanticSubjectInputOrigin, observationInvalidLocator),
	}
	if state == nil {
		return invalid
	}

	switch state.mode {
	case inputOriginLocatorID:
		return resolveOriginID(report, state.id, evaluationText("input origin by ID"))
	case inputOriginLocatorParentIndex:
		if state.parent == nil || state.index < 0 {
			return invalid
		}
		var parent originResolution
		if inspection != nil {
			parent = resolveOriginInspection(state.parent, *inspection)
		} else {
			parent = resolveOrigin(state.parent, report)
		}
		label := indexedSubject(evaluationText("parent"), state.index, parent.label)
		if parent.status != locatorResolutionFound {
			resolution = originResolution{
				status: parent.status,
				label:  label,
				actual: parent.actual,
			}
			return resolution
		}
		if state.index >= len(parent.value.Parents) {
			resolution = originResolution{
				status: locatorResolutionAbsent,
				label:  label,
				actual: evaluationText("parent origin is absent"),
			}
			return resolution
		}
		return resolveOriginID(report, parent.value.Parents[state.index], label)
	case inputOriginLocatorToken:
		if inspection == nil || state.token == nil {
			return invalid
		}
		token, status, label, detail := resolveTokenAt(state.token, *inspection)
		if status != locatorResolutionFound {
			resolution = originResolution{
				status: status,
				label:  qualifySubject(label, evaluationText("origin")),
				actual: detail,
			}
			return resolution
		}
		return resolveOriginID(report, token.Origin, qualifySubject(label, evaluationText("origin")))
	default:
		return invalid
	}
}

func redactedOutcomeField(sensitive bool, actual evaluationField) evaluationField {
	if sensitive {
		return evaluationText("value differs (redacted)")
	}
	return actual
}

func resolveExpectedOrigins(states []*inputOriginLocatorState, report cli.DiagnosticReport, inspection *cli.Inspection) ([]cli.InputOriginID, evaluationResult, bool) {
	ids := make([]cli.InputOriginID, len(states))
	for index, state := range states {
		if state != nil && state.mode == inputOriginLocatorID && state.id != (cli.InputOriginID{}) {
			ids[index] = state.id
			continue
		}
		var resolved originResolution
		if inspection != nil {
			resolved = resolveOriginInspection(state, *inspection)
		} else {
			resolved = resolveOrigin(state, report)
		}
		if resolved.status != locatorResolutionFound {
			outcome := outcomeForValueStatus(resolved.status, resolved.label, evaluationText("resolvable expected input origin"), resolved.actual)
			return nil, outcome, false
		}
		ids[index] = resolved.value.ID
	}
	return ids, evaluationResult{}, true
}

func equalInputReplacementPart(left, right cli.InputReplacementPart) bool {
	if left.Literal != right.Literal {
		return false
	}
	if left.Copy == nil || right.Copy == nil {
		return left.Copy == nil && right.Copy == nil
	}

	return *left.Copy == *right.Copy
}

func equalInputReplacement(left, right cli.InputReplacement) bool {
	return slices.EqualFunc(left, right, equalInputReplacementPart)
}

func equalReplacements(a, b []cli.InputReplacement) bool {
	return slices.EqualFunc(a, b, equalInputReplacement)
}

func inputSourcePresentExpectation(locator InputSourceLocator) OutcomeExpectation {
	return inputSourceExpectation{
		locator: locator.state,
		kind:    inputSourceExpectationPresent,
	}
}

func inputSourceKindExpectation(locator InputSourceLocator, kind cli.InputKind) OutcomeExpectation {
	return inputSourceExpectation{
		locator:   locator.state,
		kind:      inputSourceExpectationKindValue,
		inputKind: kind,
	}
}

func inputSourceSensitiveExpectation(locator InputSourceLocator) OutcomeExpectation {
	return inputSourceExpectation{
		locator: locator.state,
		kind:    inputSourceExpectationSensitive,
	}
}

func inputSourceKeyExpectation(locator InputSourceLocator, key string) OutcomeExpectation {
	return inputSourceExpectation{
		locator: locator.state,
		kind:    inputSourceExpectationKey,
		text:    key,
	}
}

func inputSourceLocationExpectation(locator InputSourceLocator, location string) OutcomeExpectation {
	return inputSourceExpectation{
		locator: locator.state,
		kind:    inputSourceExpectationLocation,
		text:    location,
	}
}

func inputSourceShortcutExpectation(locator InputSourceLocator, shortcut CommandEntryLocator) OutcomeExpectation {
	return inputSourceExpectation{
		locator:  locator.state,
		kind:     inputSourceExpectationShortcut,
		shortcut: shortcut.state,
	}
}

func inputOriginPresentExpectation(locator InputOriginLocator) OutcomeExpectation {
	return inputOriginExpectation{
		locator: locator.state,
		kind:    inputOriginExpectationPresent,
	}
}

func inputOriginSensitiveExpectation(locator InputOriginLocator) OutcomeExpectation {
	return inputOriginExpectation{
		locator: locator.state,
		kind:    inputOriginExpectationSensitive,
	}
}

func inputOriginParentsExactlyExpectation(locator InputOriginLocator, parents []InputOriginLocator) OutcomeExpectation {
	states := make([]*inputOriginLocatorState, len(parents))
	for i, parent := range parents {
		states[i] = parent.state
	}
	return inputOriginExpectation{
		locator: locator.state,
		kind:    inputOriginExpectationParents,
		parents: states,
	}
}

func inputOriginSpanExpectation(locator InputOriginLocator, span cli.InputSpan) OutcomeExpectation {
	return inputOriginExpectation{
		locator: locator.state,
		kind:    inputOriginExpectationSpan,
		span:    span,
	}
}

func locatorInputOriginIDs(locators []InputOriginLocator) ([]cli.InputOriginID, bool) {
	ids := make([]cli.InputOriginID, len(locators))
	for index, locator := range locators {
		if locator.state == nil || locator.state.mode != inputOriginLocatorID {
			return nil, false
		}
		ids[index] = locator.state.id
	}
	return ids, true
}

func locatorDeclarationIdentityValue(locator DeclarationIdentityLocator) (cli.DeclarationIdentity, bool) {
	var identity cli.DeclarationIdentity
	switch typed := locator.(type) {
	case CommandEntryLocator:
		if typed.state != nil && typed.state.mode == commandEntryLocatorID {
			identity = cli.DeclarationIdentity{
				Kind:         cli.DeclarationKindCommandEntry,
				CommandEntry: typed.state.id,
			}
			return identity, true
		}
	case ActionEntryLocator:
		if typed.state != nil && typed.state.mode == actionEntryLocatorID {
			identity = cli.DeclarationIdentity{
				Kind:        cli.DeclarationKindActionEntry,
				ActionEntry: typed.state.id,
			}
			return identity, true
		}
	case FlagLocator:
		if typed.state != nil && typed.state.mode == flagLocatorID {
			identity = cli.DeclarationIdentity{
				Kind: cli.DeclarationKindFlag,
				Flag: typed.state.id,
			}
			return identity, true
		}
	case FlagSpellingLocator:
		if typed.state != nil && typed.state.mode == flagSpellingLocatorID {
			identity = cli.DeclarationIdentity{
				Kind:         cli.DeclarationKindFlagSpelling,
				FlagSpelling: typed.state.id,
			}
			return identity, true
		}
	case ArgumentFormLocator:
		if typed.state != nil && typed.state.mode == argumentFormLocatorID {
			identity = cli.DeclarationIdentity{
				Kind:         cli.DeclarationKindArgumentForm,
				ArgumentForm: typed.state.id,
			}
			return identity, true
		}
	case ArgumentTermLocator:
		if typed.state != nil && typed.state.mode == argumentTermLocatorID {
			identity = cli.DeclarationIdentity{
				Kind:         cli.DeclarationKindArgumentTerm,
				ArgumentTerm: typed.state.id,
			}
			return identity, true
		}
	case EnvironmentInputLocator:
		if typed.state != nil && typed.state.mode == environmentInputLocatorID {
			identity = cli.DeclarationIdentity{
				Kind:             cli.DeclarationKindEnvironmentInput,
				EnvironmentInput: typed.state.id,
			}
			return identity, true
		}
	case ValueDomainEntryLocator:
		if typed.state != nil && typed.state.mode == valueDomainEntryLocatorByID {
			identity = cli.DeclarationIdentity{
				Kind:             cli.DeclarationKindValueDomainEntry,
				ValueDomainEntry: typed.state.id,
			}
			return identity, true
		}
	}
	return cli.DeclarationIdentity{}, false
}

func tokenOwnerExpectation(locator TokenLocator, owner TokenOwner) InspectionExpectation {
	return tokenOwnerExpectationValue{
		token: locator.state,
		owner: owner,
	}
}

func observedInputSourcesExactly(report cli.DiagnosticReport, kind cli.InputKind, expected []InputSourceLocator) evaluationResult {
	actual := make(map[cli.InputSourceID]struct{})
	for id, source := range report.Sources {
		if source.Kind == kind {
			actual[id] = struct{}{}
		}
	}
	if len(actual) != len(expected) {
		return disprovedResult(
			evaluationText("observed input sources"),
			evaluationText(fmt.Sprintf("exactly %d sources of kind %v", len(expected), kind)),
			evaluationText(fmt.Sprintf("%d observed sources of kind %v", len(actual), kind)),
		)
	}
	seen := make(map[cli.InputSourceID]struct{}, len(expected))
	for i, locator := range expected {
		resolved := resolveSource(locator.state, report)
		if resolved.status != locatorResolutionFound {
			return disprovedResult(evaluationText("observed input sources"), evaluationText(fmt.Sprintf("source at position %d", i)), resolved.actual)
		}
		if resolved.value.Kind != kind {
			return disprovedResult(
				evaluationText("observed input sources"),
				evaluationText(fmt.Sprintf("source kind %v", kind)),
				evaluationText(fmt.Sprintf("different kind at position %d", i)),
			)
		}
		if _, duplicate := seen[resolved.value.ID]; duplicate {
			return disprovedResult(
				evaluationText("observed input sources"),
				evaluationText("distinct semantic sources"),
				evaluationText(fmt.Sprintf("duplicate expected source at position %d", i)),
			)
		}
		seen[resolved.value.ID] = struct{}{}
		if _, ok := actual[resolved.value.ID]; !ok {
			return disprovedResult(
				evaluationText("observed input sources"),
				evaluationText("exact observed membership"),
				evaluationText(fmt.Sprintf("expected source at position %d is not observed", i)),
			)
		}
	}
	return provedResult()
}

func (inputSourceExpectation) inspectionExpectation() {}

func (inputSourceExpectation) completionExpectation() {}

func (inputSourceExpectation) diagnosticReportExpectation() {}

func (e inputSourceExpectation) expected() evaluationField {
	switch e.kind {
	case inputSourceExpectationPresent:
		return evaluationText("present")
	case inputSourceExpectationKindValue:
		return evaluationText(fmt.Sprintf("input source kind %v", e.inputKind))
	case inputSourceExpectationSensitive:
		return semanticRequirementSensitive
	case inputSourceExpectationKey:
		return property(evaluationText("input source key"), evaluationText(e.text))
	case inputSourceExpectationLocation:
		return property(evaluationText("input source location"), evaluationText(e.text))
	case inputSourceExpectationShortcut:
		return evaluationText("command shortcut")
	default:
		panic("clitest: invalid input-source expectation")
	}
}

func (e inputSourceExpectation) evaluateResolved(r sourceResolution) evaluationResult {
	if r.status != locatorResolutionFound {
		return outcomeForValueStatus(r.status, r.label, e.expected(), r.actual)
	}
	var matched bool
	var actual evaluationField
	switch e.kind {
	case inputSourceExpectationPresent:
		return provedResult()
	case inputSourceExpectationKindValue:
		matched = r.value.Kind == e.inputKind
		actual = evaluationText(fmt.Sprint(r.value.Kind))
	case inputSourceExpectationSensitive:
		matched = r.value.Sensitive
		actual = evaluationText("not sensitive")
	case inputSourceExpectationKey:
		matched = r.value.Key == e.text
		actual = evaluationText(r.value.Key)
	case inputSourceExpectationLocation:
		matched = r.value.Location == e.text
		actual = evaluationText(r.value.Location)
	case inputSourceExpectationShortcut:
		if e.shortcut == nil || e.shortcut.mode != commandEntryLocatorID {
			return disprovedResult(r.label, e.expected(), observed(semanticSubjectCommandEntry, observationInvalidLocator))
		}
		matched = r.value.CommandShortcut == e.shortcut.id
		actual = evaluationText(fmt.Sprintf("shortcut %v", r.value.CommandShortcut))
	default:
		panic("clitest: invalid input-source expectation")
	}
	if matched {
		return provedResult()
	}
	return disprovedResult(r.label, e.expected(), redactedOutcomeField(r.value.Sensitive, actual))
}

func (e inputSourceExpectation) evaluateInspection(i cli.Inspection, _ error) evaluationResult {
	return e.evaluateResolved(resolveSourceInspection(e.locator, i))
}

func (e inputSourceExpectation) evaluateCompletion(c cli.Completion, _ error) evaluationResult {
	return e.evaluateResolved(resolveSource(e.locator, diagnosticReportFromCompletion(c)))
}

func (e inputSourceExpectation) evaluateDiagnosticReport(r cli.DiagnosticReport) evaluationResult {
	return e.evaluateResolved(resolveSource(e.locator, r))
}

func (inputOriginExpectation) inspectionExpectation() {}

func (inputOriginExpectation) completionExpectation() {}

func (inputOriginExpectation) diagnosticReportExpectation() {}

func (e inputOriginExpectation) evaluate(r originResolution, report cli.DiagnosticReport, inspection *cli.Inspection) evaluationResult {
	var expected evaluationField = evaluationText("present")
	switch e.kind {
	case inputOriginExpectationSensitive:
		expected = semanticRequirementSensitive
	case inputOriginExpectationParents:
		expected = evaluationText("exact origin parents in authoritative order")
	case inputOriginExpectationSpan:
		expected = qualifySubject(evaluationText("origin span"), inputSpanField{value: e.span})
	default:
	}
	if r.status != locatorResolutionFound {
		return outcomeForValueStatus(r.status, r.label, expected, r.actual)
	}
	switch e.kind {
	case inputOriginExpectationPresent:
		return provedResult()
	case inputOriginExpectationSensitive:
		if r.value.Sensitive {
			return provedResult()
		}
		return disprovedResult(r.label, expected, evaluationText("not sensitive"))
	case inputOriginExpectationParents:
		ids, outcome, ok := resolveExpectedOrigins(e.parents, report, inspection)
		if !ok {
			return outcome
		}
		if slices.Equal(r.value.Parents, ids) {
			return provedResult()
		}
		if r.value.Sensitive {
			return disprovedResult(r.label, expected, evaluationText("value differs (redacted)"))
		}
		return orderedCollectionMismatch(r.label, "parents", ids, r.value.Parents)
	case inputOriginExpectationSpan:
		if r.value.Span == e.span {
			return provedResult()
		}
		return disprovedResult(r.label, expected, redactedOutcomeField(r.value.Sensitive, inputSpanField{value: r.value.Span}))
	default:
		panic("clitest: invalid input-origin expectation")
	}
}

func (e inputOriginExpectation) evaluateInspection(i cli.Inspection, _ error) evaluationResult {
	return e.evaluate(resolveOriginInspection(e.locator, i), diagnosticReportFromInspection(i), &i)
}

func (e inputOriginExpectation) evaluateCompletion(c cli.Completion, _ error) evaluationResult {
	r := diagnosticReportFromCompletion(c)
	return e.evaluate(resolveOrigin(e.locator, r), r, nil)
}

func (e inputOriginExpectation) evaluateDiagnosticReport(r cli.DiagnosticReport) evaluationResult {
	return e.evaluate(resolveOrigin(e.locator, r), r, nil)
}

func (tokenExpectation) inspectionExpectation() {}

func (expectation tokenExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	token, status, subject, detail := resolveTokenAt(expectation.locator, inspection)
	if status == locatorResolutionNotProvable {
		return notProvableResult(subject, expectation.expected(), detail)
	}
	if status != locatorResolutionFound {
		return disprovedResult(subject, expectation.expected(), detail)
	}

	matched, observedValue := expectation.compare(token)
	if matched {
		return provedResult()
	}
	if token.Redacted && expectation.kind != tokenExpectationRedacted {
		return disprovedResult(subject, evaluationText("redacted expected token property"), evaluationText("redacted observed token property"))
	}
	return disprovedResult(subject, expectation.expected(), observedValue)
}

func (expectation tokenExpectation) expected() evaluationField {
	switch expectation.kind {
	case tokenExpectationRole:
		return evaluationText(fmt.Sprintf("role %v", expectation.role))
	case tokenExpectationRejection:
		return subjectNamed(evaluationText("rejection"), string(expectation.rejection))
	case tokenExpectationSpelling:
		return subjectNamed(evaluationText("matched spelling"), expectation.spelling)
	case tokenExpectationMatch:
		return evaluationText(fmt.Sprintf("name match %v", expectation.match))
	case tokenExpectationRedacted:
		return evaluationText("redacted")
	case tokenExpectationFlagValueForm:
		return evaluationText(fmt.Sprintf("flag-value form %v", expectation.flagValueForm))
	case tokenExpectationOccurrenceIndex:
		return evaluationText(fmt.Sprintf("occurrence index %d", expectation.index))
	case tokenExpectationElementIndex:
		return evaluationText(fmt.Sprintf("element index %d", expectation.index))
	case tokenExpectationDelegatedIndex:
		return evaluationText(fmt.Sprintf("delegated index %d", expectation.index))
	case tokenExpectationText:
		return subjectNamed(evaluationText("text"), expectation.text)
	default:
		panic("clitest: invalid token expectation")
	}
}

func (expectation tokenExpectation) compare(token cli.TokenInspection) (bool, evaluationField) {
	switch expectation.kind {
	case tokenExpectationRole:
		return token.Role == expectation.role, evaluationText(fmt.Sprintf("role %v", token.Role))
	case tokenExpectationRejection:
		return token.RejectionCode == expectation.rejection, subjectNamed(evaluationText("rejection"), string(token.RejectionCode))
	case tokenExpectationSpelling:
		return token.MatchedSpelling == expectation.spelling, subjectNamed(evaluationText("matched spelling"), token.MatchedSpelling)
	case tokenExpectationMatch:
		return token.Match == expectation.match, evaluationText(fmt.Sprintf("name match %v", token.Match))
	case tokenExpectationRedacted:
		return token.Redacted, evaluationText(fmt.Sprintf("redacted=%t", token.Redacted))
	case tokenExpectationFlagValueForm:
		return token.FlagValueForm == expectation.flagValueForm, evaluationText(fmt.Sprintf("flag-value form %v", token.FlagValueForm))
	case tokenExpectationOccurrenceIndex:
		return token.OccurrenceIndex == expectation.index, evaluationText(fmt.Sprintf("occurrence index %d", token.OccurrenceIndex))
	case tokenExpectationElementIndex:
		return token.ElementIndex == expectation.index, evaluationText(fmt.Sprintf("element index %d", token.ElementIndex))
	case tokenExpectationDelegatedIndex:
		return token.DelegatedIndex == expectation.index, evaluationText(fmt.Sprintf("delegated index %d", token.DelegatedIndex))
	case tokenExpectationText:
		if token.Text == expectation.text {
			return true, evaluationText("")
		}
		if token.Redacted {
			return false, evaluationText("redacted observed token text")
		}
		return false, subjectNamed(evaluationText("text"), token.Text)
	default:
		panic("clitest: invalid token expectation")
	}
}

func (tokenOwnerExpectationValue) inspectionExpectation() {}

func (expectation tokenOwnerExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	const expect = "configured semantic owner"
	token, status, label, actual := resolveTokenAt(expectation.token, inspection)
	if status == locatorResolutionNotProvable {
		return notProvableResult(label, evaluationText(expect), actual)
	}
	if status != locatorResolutionFound {
		return disprovedResult(label, evaluationText(expect), actual)
	}
	if expectation.owner == nil {
		return disprovedResult(label, evaluationText(expect), evaluationText("invalid token-owner locator"))
	}

	var matched bool
	switch owner := expectation.owner.(type) {
	case CommandLocator:
		resolution := resolveInspectionCommandLocator(owner.state, inspection)
		if resolution.status == locatorResolutionNotProvable {
			return notProvableResult(label, evaluationText(expect), resolution.detail)
		}
		if resolution.status != locatorResolutionFound {
			return disprovedResult(label, evaluationText(expect), resolution.detail)
		}
		matched = token.Command == resolution.value.ID
		actual = evaluationText(fmt.Sprintf("command %v", token.Command))

	case CommandEntryLocator, ActionEntryLocator, FlagLocator, FlagSpellingLocator, ArgumentFormLocator, ArgumentTermLocator, EnvironmentInputLocator, ValueDomainEntryLocator:
		identityLocator, ok := owner.(DeclarationIdentityLocator)
		if !ok {
			panic("clitest: declaration token owner lacks declaration identity")
		}
		declaration, declarationStatus, declarationActual := resolveExpectedDeclarationIdentity(identityLocator, nil, nil, inspection)
		if declarationStatus == locatorResolutionNotProvable {
			return notProvableResult(label, evaluationText(expect), declarationActual)
		}
		if declarationStatus != locatorResolutionFound {
			return disprovedResult(label, evaluationText(expect), declarationActual)
		}
		matched = token.Declaration == declaration
		actual = declarationIdentityField{value: token.Declaration}

	case FactLocator:
		id, ownerStatus, ownerDetail := resolveInspectionFactID(owner.state, inspection)
		if ownerStatus == locatorResolutionNotProvable {
			return notProvableResult(label, evaluationText(expect), ownerDetail)
		}
		if ownerStatus != locatorResolutionFound {
			return disprovedResult(label, evaluationText(expect), ownerDetail)
		}
		matched = token.Destination == id
		actual = evaluationText(fmt.Sprintf("destination %v", token.Destination))

	case ValueLocator:
		id, ownerStatus, ownerDetail := resolveInspectionValueID(owner.state, inspection)
		if ownerStatus == locatorResolutionNotProvable {
			return notProvableResult(label, evaluationText(expect), ownerDetail)
		}
		if ownerStatus != locatorResolutionFound {
			return disprovedResult(label, evaluationText(expect), ownerDetail)
		}
		matched = token.Destination == id
		actual = evaluationText(fmt.Sprintf("destination %v", token.Destination))

	default:
		return disprovedResult(label, evaluationText(expect), evaluationText("unsupported token-owner locator"))
	}

	if matched {
		return provedResult()
	}
	if token.Redacted {
		return disprovedResult(label, evaluationText("redacted expected token property"), evaluationText("redacted observed token property"))
	}
	return disprovedResult(label, evaluationText(expect), actual)
}

func (inspectionObservedInputSourcesExpectation) inspectionExpectation() {}

func (expectation inspectionObservedInputSourcesExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return observedInputSourcesExactly(diagnosticReportFromInspection(inspection), expectation.kind, expectation.sources)
}
