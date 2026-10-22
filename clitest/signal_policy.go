package clitest

import (
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/abstractionsinc/commandment/cli"
)

// SignalSelectorLocator identifies the selector owned by one public signal policy. It checks
// only selector facts exposed by Commandment; expected collections retain exact public order,
// and the zero value is invalid.
type SignalSelectorLocator struct{ policy SignalPolicyLocator }

// SignalActionLocator identifies one public signal-policy action by its authoritative action
// index. It can follow the action's real endpoint and value relationships; it contains expectation
// coordinates, not runtime signal state, and its zero value is invalid.
type SignalActionLocator struct {
	policy SignalPolicyLocator
	index  int
}

// SignalPolicyLocator identifies one signal-policy declaration or effective
// policy record for clitest semantic expectation. Its zero value is invalid.
type SignalPolicyLocator struct {
	_     [0]func()
	state *signalPolicyLocatorState
}

// SignalEndpointLocator identifies one signal-endpoint declaration or effective
// endpoint record for clitest semantic expectation. Its zero value is invalid.
type SignalEndpointLocator struct {
	_     [0]func()
	state *signalEndpointLocatorState
}

// SignalEndpointCount returns a reusable static declaration expectation that checks the number
// of public signal endpoints in the exact Schema or Reference observation.
func SignalEndpointCount(expected int) StaticDeclarationExpectation {
	expectation := signalEndpointCountExpectation{
		expected: expected,
	}
	return expectation
}

// InspectionObservedSignalEndpointCount returns an Inspection expectation for the number of signal endpoints present in
// the returned SignalEndpoints map.
// It is an observed host-integration count; endpoint status and integration readiness are separate public facts.
func InspectionObservedSignalEndpointCount(expected int) InspectionExpectation {
	return observedSignalEndpointCountExpectation(expected)
}

// InspectionSignalEndpoint returns a signal-endpoint locator selecting exactly one observed endpoint with status.
// Because status is not an identity, multiple matching endpoints are ambiguous and the expectation fails instead of depending
// on map iteration order.
func InspectionSignalEndpoint(status cli.SignalEndpointStatus) SignalEndpointLocator {
	state := &signalEndpointLocatorState{
		mode:   signalEndpointLocatorStatus,
		status: status,
	}
	locator := SignalEndpointLocator{
		state: state,
	}
	return locator
}

// SignalPolicyByID returns a locator for the exact public SignalPolicyID in the observation
// being evaluated.
func SignalPolicyByID(id cli.SignalPolicyID) SignalPolicyLocator { return signalPolicyIDLocator(id) }

// SignalEndpointByID returns a locator for the exact public SignalEndpointID in the
// observation being evaluated.
func SignalEndpointByID(id cli.SignalEndpointID) SignalEndpointLocator {
	return signalEndpointIDLocator(id)
}

// SignalPolicyAt returns a locator for the signal policy at index in the located command's
// authoritative public signal-policy order.
func (l CommandLocator) SignalPolicyAt(index int) SignalPolicyLocator {
	return commandSignalPolicyAtLocator(l, index)
}

// Activation returns a locator that follows the public activation relationship from the located
// signal policy. The returned locator carries locator coordinates, not observed state.
func (locator SignalPolicyLocator) Activation() ConditionExpressionLocator {
	state := &conditionExpressionLocatorState{
		mode:         conditionExpressionLocatorSignalPolicyActivation,
		signalPolicy: locator.state,
	}
	conditionExpressionLocator := ConditionExpressionLocator{
		state: state,
	}
	return conditionExpressionLocator
}

// Selector returns a locator that follows the public selector relationship from the located
// signal policy. The returned locator carries locator coordinates, not observed state.
func (locator SignalPolicyLocator) Selector() SignalSelectorLocator {
	signalSelectorLocator := SignalSelectorLocator{
		policy: locator,
	}
	return signalSelectorLocator
}

// Occurrence returns a reusable declaration expectation that checks the public occurrence range
// exposed by the located signal policy.
func (locator SignalPolicyLocator) Occurrence(expected cli.SignalOccurrenceRange) DeclarationExpectation {
	value := signalPolicyOccurrenceExpectationValue{
		locator:  locator.state,
		expected: expected,
	}
	return value
}

// Aggregation returns a reusable declaration expectation that checks the public aggregation
// mode exposed by the located signal policy.
func (locator SignalPolicyLocator) Aggregation(expected cli.SignalAggregation) DeclarationExpectation {
	value := signalPolicyAggregationExpectationValue{
		locator:  locator.state,
		expected: expected,
	}
	return value
}

// Deferred returns a reusable declaration expectation that checks the public deferred state
// exposed by the located signal policy.
func (locator SignalPolicyLocator) Deferred() DeclarationExpectation {
	value := signalPolicyDeferredExpectationValue{
		locator:  locator.state,
		expected: true,
	}
	return value
}

// NotDeferred returns a reusable declaration expectation that checks the located signal policy
// is not deferred.
func (locator SignalPolicyLocator) NotDeferred() DeclarationExpectation {
	value := signalPolicyDeferredExpectationValue{
		locator:  locator.state,
		expected: false,
	}
	return value
}

// ContinuesMatching returns a reusable declaration expectation that checks the public
// continue-matching state exposed by the located signal policy.
func (locator SignalPolicyLocator) ContinuesMatching() DeclarationExpectation {
	value := signalPolicyContinueMatchingExpectationValue{
		locator:  locator.state,
		expected: true,
	}
	return value
}

// StopsMatching returns a reusable declaration expectation that checks the public stop-matching
// state exposed by the located signal policy.
func (locator SignalPolicyLocator) StopsMatching() DeclarationExpectation {
	value := signalPolicyContinueMatchingExpectationValue{
		locator:  locator.state,
		expected: false,
	}
	return value
}

// Hint returns a reusable declaration expectation that checks the public hint exposed by the
// located signal policy.
func (locator SignalPolicyLocator) Hint(expected string) DeclarationExpectation {
	value := signalPolicyHintExpectationValue{
		locator:  locator.state,
		expected: expected,
	}
	return value
}

// ActionCount returns a reusable declaration expectation that checks the action count exposed
// by the located signal policy.
func (locator SignalPolicyLocator) ActionCount(expected int) DeclarationExpectation {
	value := signalPolicyActionCountExpectationValue{
		locator:  locator.state,
		expected: expected,
	}
	return value
}

// ActionAt returns a locator for the action at the supplied public ordered index of the located
// signal policy. It does not search for or silently choose a first match.
func (locator SignalPolicyLocator) ActionAt(index int) SignalActionLocator {
	signalActionLocator := SignalActionLocator{
		policy: locator,
		index:  index,
	}
	return signalActionLocator
}

// Origin returns an expectation that checks the located signal policy has the
// framework-established DeclarationOrigin expected.
func (locator SignalPolicyLocator) Origin(expected cli.DeclarationOrigin) DeclarationExpectation {
	value := signalPolicyOriginExpectationValue{
		locator:  locator.state,
		expected: expected,
	}
	return value
}

// Exists returns an expectation that checks the located signal policy exists in the relevant
// public declaration observation.
func (locator SignalPolicyLocator) Exists() DeclarationExpectation {
	value := signalPolicyExistenceExpectationValue{
		locator:  locator.state,
		expected: true,
	}
	return value
}

// DoesNotExist returns an expectation that checks the located signal policy is absent from a
// complete relevant public declaration domain; incomplete observation cannot manufacture
// absence.
func (locator SignalPolicyLocator) DoesNotExist() DeclarationExpectation {
	value := signalPolicyExistenceExpectationValue{
		locator:  locator.state,
		expected: false,
	}
	return value
}

// Reachability returns an expectation that checks the signal policy has the requested declared
// reachability.
func (locator SignalPolicyLocator) Reachability(v cli.Reachability) DeclarationExpectation {
	value := signalPolicyReachabilityExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Available returns an expectation that checks the located signal policy is effective and
// available in the public Reference observation.
func (locator SignalPolicyLocator) Available() ReferenceExpectation {
	value := signalPolicyAvailabilityExpectationValue{
		locator:  locator.state,
		expected: true,
	}
	return value
}

// Unavailable returns an expectation that checks the located signal policy is unavailable in
// the public Reference observation.
func (locator SignalPolicyLocator) Unavailable() ReferenceExpectation {
	value := signalPolicyAvailabilityExpectationValue{
		locator:  locator.state,
		expected: false,
	}
	return value
}

// Visible returns an expectation that checks the located signal policy is visible in the
// public Reference observation.
func (locator SignalPolicyLocator) Visible() ReferenceExpectation {
	value := signalPolicyVisibilityExpectationValue{
		locator:  locator.state,
		expected: true,
	}
	return value
}

// NotVisible returns an expectation that checks the located signal policy is not visible in
// the public Reference observation.
func (locator SignalPolicyLocator) NotVisible() ReferenceExpectation {
	value := signalPolicyVisibilityExpectationValue{
		locator:  locator.state,
		expected: false,
	}
	return value
}

// Status returns an expectation that checks the signal policy's invocation-time status.
func (locator SignalPolicyLocator) Status(status cli.SignalPolicyStatus) InspectionExpectation {
	return signalPolicyStatusExpectation(locator, status)
}

// ActivationOutcome returns an expectation that checks the signal policy's activation
// evaluation outcome.
func (locator SignalPolicyLocator) ActivationOutcome(outcome cli.EvaluationOutcome) InspectionExpectation {
	return signalPolicyActivationOutcomeExpectation(locator, outcome)
}

// RequiredSourceCapabilities returns an expectation that checks the policy declares exactly
// the requested signal-source capabilities.
func (locator SignalPolicyLocator) RequiredSourceCapabilities(v cli.SignalSourceCapabilities) DeclarationExpectation {
	value := signalPolicySourceCapabilitiesExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// RequiredHostCapabilities returns an expectation that checks the policy declares exactly the
// requested signal-host capabilities.
func (locator SignalPolicyLocator) RequiredHostCapabilities(v cli.SignalHostCapabilities) DeclarationExpectation {
	value := signalPolicyHostCapabilitiesExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Sensitive returns an expectation that checks the signal policy is declared sensitive; expectation
// failures preserve redaction.
func (locator SignalPolicyLocator) Sensitive() DeclarationExpectation {
	value := signalPolicySensitiveExpectationValue{
		locator: locator.state,
	}
	return value
}

// Kind returns a reusable declaration expectation that checks the public kind exposed by the
// located signal selector.
func (locator SignalSelectorLocator) Kind(expected cli.SignalSelectorKind) DeclarationExpectation {
	value := signalSelectorExpectationValue{
		locator:  locator.policy.state,
		property: signalSelectorExpectationKind,
		kind:     expected,
	}
	return value
}

// SignalsExactly returns a reusable declaration expectation that checks the complete signal
// list exactly in authoritative public order.
func (locator SignalSelectorLocator) SignalsExactly(expected ...cli.Signal) DeclarationExpectation {
	value := signalSelectorExpectationValue{
		locator:  locator.policy.state,
		property: signalSelectorExpectationSignals,
		signals:  slices.Clone(expected),
	}
	return value
}

// ExcludedSignalsExactly returns a reusable declaration expectation that checks the complete
// excluded-signal list exactly in authoritative public order.
func (locator SignalSelectorLocator) ExcludedSignalsExactly(expected ...cli.Signal) DeclarationExpectation {
	value := signalSelectorExpectationValue{
		locator:  locator.policy.state,
		property: signalSelectorExpectationExcludedSignals,
		signals:  slices.Clone(expected),
	}
	return value
}

// OriginsExactly returns a reusable declaration expectation that checks the complete origin
// list exactly in authoritative public order.
func (locator SignalSelectorLocator) OriginsExactly(expected ...cli.SignalOrigin) DeclarationExpectation {
	value := signalSelectorExpectationValue{
		locator:  locator.policy.state,
		property: signalSelectorExpectationOrigins,
		origins:  slices.Clone(expected),
	}
	return value
}

// ExcludedOriginsExactly returns a reusable declaration expectation that checks the complete
// excluded-origin list exactly in authoritative public order.
func (locator SignalSelectorLocator) ExcludedOriginsExactly(expected ...cli.SignalOrigin) DeclarationExpectation {
	value := signalSelectorExpectationValue{
		locator:  locator.policy.state,
		property: signalSelectorExpectationExcludedOrigins,
		origins:  slices.Clone(expected),
	}
	return value
}

// ExcludedSender returns a reusable declaration expectation that checks the public excluded
// sender exposed by the located signal selector.
func (locator SignalSelectorLocator) ExcludedSender(expected string) DeclarationExpectation {
	value := signalSelectorExpectationValue{
		locator:        locator.policy.state,
		property:       signalSelectorExpectationExcludedSender,
		excludedSender: expected,
	}
	return value
}

// PhasesExactly returns a reusable declaration expectation that checks the complete phase list
// exactly in authoritative public order.
func (locator SignalSelectorLocator) PhasesExactly(expected ...cli.SignalPhase) DeclarationExpectation {
	value := signalSelectorExpectationValue{
		locator:  locator.policy.state,
		property: signalSelectorExpectationPhases,
		phases:   slices.Clone(expected),
	}
	return value
}

// Kind returns a reusable declaration expectation that checks the public kind exposed by the
// located signal action.
func (locator SignalActionLocator) Kind(expected cli.SignalActionKind) DeclarationExpectation {
	value := signalActionExpectationValue{
		locator:  locator,
		property: signalActionExpectationKind,
		kind:     expected,
	}
	return value
}

// Name returns a reusable declaration expectation that checks the public name exposed by the
// located signal action.
func (locator SignalActionLocator) Name(expected string) DeclarationExpectation {
	value := signalActionExpectationValue{
		locator:  locator,
		property: signalActionExpectationName,
		name:     expected,
	}
	return value
}

// Signal returns a reusable declaration expectation that checks the public signal exposed by
// the located signal action.
func (locator SignalActionLocator) Signal(expected cli.Signal) DeclarationExpectation {
	value := signalActionExpectationValue{
		locator:  locator,
		property: signalActionExpectationSignal,
		signal:   expected,
	}
	return value
}

// SignalRedacted returns a reusable declaration expectation that checks the public
// signal-redacted state exposed by the located signal action.
func (locator SignalActionLocator) SignalRedacted() DeclarationExpectation {
	value := signalActionExpectationValue{
		locator:  locator,
		property: signalActionExpectationSignalRedacted,
		boolean:  true,
	}
	return value
}

// Delayed returns a reusable declaration expectation that checks the public delayed state
// exposed by the located signal action.
func (locator SignalActionLocator) Delayed() DeclarationExpectation {
	value := signalActionExpectationValue{
		locator:  locator,
		property: signalActionExpectationDelayed,
		boolean:  true,
	}
	return value
}

// Immediate returns a reusable declaration expectation that checks the public immediate state
// exposed by the located signal action.
func (locator SignalActionLocator) Immediate() DeclarationExpectation {
	value := signalActionExpectationValue{
		locator:  locator,
		property: signalActionExpectationDelayed,
	}
	return value
}

// Delay returns a reusable declaration expectation that checks the public delay exposed by the
// located signal action.
func (locator SignalActionLocator) Delay(expected time.Duration) DeclarationExpectation {
	value := signalActionExpectationValue{
		locator:  locator,
		property: signalActionExpectationDelay,
		delay:    expected,
	}
	return value
}

// DelayRedacted returns a reusable declaration expectation that checks the public
// delay-redacted state exposed by the located signal action.
func (locator SignalActionLocator) DelayRedacted() DeclarationExpectation {
	value := signalActionExpectationValue{
		locator:  locator,
		property: signalActionExpectationDelayRedacted,
		boolean:  true,
	}
	return value
}

// ExitCode returns a reusable declaration expectation that checks the public exit code exposed
// by the located signal action.
func (locator SignalActionLocator) ExitCode(expected int) DeclarationExpectation {
	value := signalActionExpectationValue{
		locator:  locator,
		property: signalActionExpectationExitCode,
		exitCode: expected,
	}
	return value
}

// Sensitive returns a reusable declaration expectation that checks the located signal action is
// marked sensitive. Expectation failures preserve Commandment redaction boundaries.
func (locator SignalActionLocator) Sensitive() DeclarationExpectation {
	value := signalActionExpectationValue{
		locator:  locator,
		property: signalActionExpectationSensitive,
		boolean:  true,
	}
	return value
}

// SignalEndpoint returns a locator by following the public endpoint identity carried by the
// located signal action. The returned locator does not invent an endpoint when that
// relationship is absent.
func (locator SignalActionLocator) SignalEndpoint() SignalEndpointLocator {
	return signalActionEndpointLocator(locator)
}

// Endpoint returns a declaration expectation that checks the signal action's public EndpointID
// resolves to the same signal-endpoint identity as the expected locator.
func (locator SignalActionLocator) Endpoint(expected SignalEndpointLocator) DeclarationExpectation {
	value := signalActionEndpointExpectationValue{
		locator:  locator,
		expected: expected.state,
	}
	return value
}

// SignalValue returns a declaration expectation that checks the action's signal-value
// relationship resolves to the same public value identity as the expected locator.
func (locator SignalActionLocator) SignalValue(expected ValueLocator) DeclarationExpectation {
	value := signalActionValueExpectationValue{
		locator:  locator,
		expected: expected.state,
	}
	return value
}

// DelayValue returns a declaration expectation that checks the action's delay-value
// relationship resolves to the same public value identity as the expected locator.
func (locator SignalActionLocator) DelayValue(expected ValueLocator) DeclarationExpectation {
	value := signalActionValueExpectationValue{
		locator:  locator,
		expected: expected.state,
		delay:    true,
	}
	return value
}

// Capacity returns a reusable declaration expectation that checks the public capacity exposed
// by the located signal endpoint.
func (locator SignalEndpointLocator) Capacity(expected int) DeclarationExpectation {
	value := signalEndpointCapacityExpectationValue{
		locator:  locator.state,
		expected: expected,
	}
	return value
}

// Origin returns an expectation that checks the located signal endpoint has the
// framework-established DeclarationOrigin expected.
func (locator SignalEndpointLocator) Origin(expected cli.DeclarationOrigin) DeclarationExpectation {
	value := signalEndpointOriginExpectationValue{
		locator:  locator.state,
		expected: expected,
	}
	return value
}

// Exists returns an expectation that checks the located signal endpoint exists in the relevant
// public declaration observation.
func (locator SignalEndpointLocator) Exists() DeclarationExpectation {
	value := signalEndpointExistenceExpectationValue{
		locator:  locator.state,
		expected: true,
	}
	return value
}

// DoesNotExist returns an expectation that checks the located signal endpoint is absent from a
// complete relevant public declaration domain; incomplete observation cannot manufacture
// absence.
func (locator SignalEndpointLocator) DoesNotExist() DeclarationExpectation {
	value := signalEndpointExistenceExpectationValue{
		locator:  locator.state,
		expected: false,
	}
	return value
}

// Status returns an expectation that checks the signal endpoint's invocation-time status.
func (locator SignalEndpointLocator) Status(status cli.SignalEndpointStatus) InspectionExpectation {
	return signalEndpointStatusExpectation(locator, status)
}

// EffectivePoliciesExactly returns an expectation that checks the endpoint has exactly the
// requested effective signal policies in authoritative public order.
func (locator SignalEndpointLocator) EffectivePoliciesExactly(policies ...SignalPolicyLocator) InspectionExpectation {
	return signalEndpointPoliciesExpectation(locator, true, policies)
}

// BlockedPoliciesExactly returns an expectation that checks the endpoint has exactly the
// requested blocked signal policies in authoritative public order.
func (locator SignalEndpointLocator) BlockedPoliciesExactly(policies ...SignalPolicyLocator) InspectionExpectation {
	return signalEndpointPoliciesExpectation(locator, false, policies)
}

type signalEndpointCountExpectation struct{ expected int }

type executionAdmissionSignalEndpointsExactlyExpectationValue struct {
	admission *executionAdmissionLocatorState
	locators  []*signalEndpointLocatorState
}

type signalPolicyStatusExpectationValue struct {
	locator *signalPolicyLocatorState
	status  cli.SignalPolicyStatus
}

type signalPolicyActivationOutcomeExpectationValue struct {
	locator *signalPolicyLocatorState
	outcome cli.EvaluationOutcome
}

type signalEndpointStatusExpectationValue struct {
	locator *signalEndpointLocatorState
	status  cli.SignalEndpointStatus
}

type signalEndpointPoliciesExactExpectation struct {
	locator   *signalEndpointLocatorState
	effective bool
	states    []*signalPolicyLocatorState
}

type observedSignalEndpointCountExpectation int

type signalPolicyLocatorMode uint8

const (
	signalPolicyLocatorInvalid signalPolicyLocatorMode = iota
	signalPolicyLocatorID
	signalPolicyLocatorCommandIndex
	signalPolicyLocatorExecutionAdmissionIndex
)

type signalPolicyLocatorState struct {
	mode signalPolicyLocatorMode
	id   cli.SignalPolicyID

	command *commandLocatorState
	index   int
}

type signalEndpointLocatorMode uint8

const (
	signalEndpointLocatorInvalid signalEndpointLocatorMode = iota
	signalEndpointLocatorID
	signalEndpointLocatorStatus
	signalEndpointLocatorExecutionAdmissionIndex
	signalEndpointLocatorAction
)

type signalEndpointLocatorState struct {
	mode   signalEndpointLocatorMode
	id     cli.SignalEndpointID
	status cli.SignalEndpointStatus
	index  int

	policy      *signalPolicyLocatorState
	actionIndex int
}

type signalPolicyExistenceExpectationValue struct {
	locator  *signalPolicyLocatorState
	expected bool
}

type signalPolicyReachabilityExpectationValue struct {
	locator  *signalPolicyLocatorState
	expected cli.Reachability
}

type signalPolicySensitiveExpectationValue struct {
	locator *signalPolicyLocatorState
}

type signalPolicySourceCapabilitiesExpectationValue struct {
	locator  *signalPolicyLocatorState
	expected cli.SignalSourceCapabilities
}

type signalPolicyHostCapabilitiesExpectationValue struct {
	locator  *signalPolicyLocatorState
	expected cli.SignalHostCapabilities
}

type signalPolicyAvailabilityExpectationValue struct {
	locator  *signalPolicyLocatorState
	expected bool
}

type signalPolicyVisibilityExpectationValue struct {
	locator  *signalPolicyLocatorState
	expected bool
}

type signalEndpointExistenceExpectationValue struct {
	locator  *signalEndpointLocatorState
	expected bool
}

type signalPolicyOccurrenceExpectationValue struct {
	locator  *signalPolicyLocatorState
	expected cli.SignalOccurrenceRange
}

type signalPolicyAggregationExpectationValue struct {
	locator  *signalPolicyLocatorState
	expected cli.SignalAggregation
}

type signalPolicyDeferredExpectationValue struct {
	locator  *signalPolicyLocatorState
	expected bool
}

type signalPolicyContinueMatchingExpectationValue struct {
	locator  *signalPolicyLocatorState
	expected bool
}

type signalPolicyHintExpectationValue struct {
	locator  *signalPolicyLocatorState
	expected string
}

type signalPolicyActionCountExpectationValue struct {
	locator  *signalPolicyLocatorState
	expected int
}

type signalPolicyOriginExpectationValue struct {
	locator  *signalPolicyLocatorState
	expected cli.DeclarationOrigin
}

type signalEndpointCapacityExpectationValue struct {
	locator  *signalEndpointLocatorState
	expected int
}

type signalEndpointOriginExpectationValue struct {
	locator  *signalEndpointLocatorState
	expected cli.DeclarationOrigin
}

type signalSelectorExpectationProperty uint8

const (
	signalSelectorExpectationInvalid signalSelectorExpectationProperty = iota
	signalSelectorExpectationKind
	signalSelectorExpectationSignals
	signalSelectorExpectationExcludedSignals
	signalSelectorExpectationOrigins
	signalSelectorExpectationExcludedOrigins
	signalSelectorExpectationExcludedSender
	signalSelectorExpectationPhases
)

type signalSelectorExpectationValue struct {
	locator        *signalPolicyLocatorState
	property       signalSelectorExpectationProperty
	kind           cli.SignalSelectorKind
	signals        []cli.Signal
	origins        []cli.SignalOrigin
	excludedSender string
	phases         []cli.SignalPhase
}

type signalActionExpectationProperty uint8

const (
	signalActionExpectationInvalid signalActionExpectationProperty = iota
	signalActionExpectationKind
	signalActionExpectationName
	signalActionExpectationSignal
	signalActionExpectationSignalRedacted
	signalActionExpectationDelayed
	signalActionExpectationDelay
	signalActionExpectationDelayRedacted
	signalActionExpectationExitCode
	signalActionExpectationSensitive
)

type signalActionExpectationValue struct {
	locator  SignalActionLocator
	property signalActionExpectationProperty
	kind     cli.SignalActionKind
	name     string
	signal   cli.Signal
	boolean  bool
	delay    time.Duration
	exitCode int
}

type signalActionEndpointExpectationValue struct {
	locator  SignalActionLocator
	expected *signalEndpointLocatorState
}

type signalActionValueExpectationValue struct {
	locator  SignalActionLocator
	expected *valueLocatorState
	delay    bool
}

func commandSignalPolicyAtLocator(parent CommandLocator, index int) SignalPolicyLocator {
	state := &signalPolicyLocatorState{
		mode:    signalPolicyLocatorCommandIndex,
		command: parent.state,
		index:   index,
	}

	locator := SignalPolicyLocator{
		state: state,
	}
	return locator
}

func resolveSignalActionSchema(locator SignalActionLocator, schema *cli.Schema) (cli.SignalAction, locatorResolutionStatus, evaluationField) {
	policy := resolveSchemaSignalPolicyLocator(locator.policy.state, schema)
	if policy.status != locatorResolutionFound {
		return cli.SignalAction{}, policy.status, policy.detail
	}
	if locator.index < 0 {
		return cli.SignalAction{}, locatorResolutionInvalid, observed(semanticSubjectSignalAction, observationInvalidLocator)
	}
	if locator.index >= len(policy.value.Actions) {
		return cli.SignalAction{}, locatorResolutionAbsent, observed(semanticSubjectSignalAction, observationAbsent)
	}
	return policy.value.Actions[locator.index], locatorResolutionFound, evaluationText("")
}

func resolveSignalActionReference(locator SignalActionLocator, reference cli.Reference) (cli.SignalAction, locatorResolutionStatus, evaluationField) {
	policy := resolveReferenceSignalPolicyLocator(locator.policy.state, reference)
	if policy.status != locatorResolutionFound {
		return cli.SignalAction{}, policy.status, policy.detail
	}
	if locator.index < 0 {
		return cli.SignalAction{}, locatorResolutionInvalid, observed(semanticSubjectSignalAction, observationInvalidLocator)
	}
	if locator.index >= len(policy.value.Actions) {
		return cli.SignalAction{}, locatorResolutionAbsent, observed(semanticSubjectSignalAction, observationAbsent)
	}
	return policy.value.Actions[locator.index], locatorResolutionFound, evaluationText("")
}

func resolveSignalActionInspection(locator SignalActionLocator, inspection cli.Inspection) (cli.SignalAction, locatorResolutionStatus, evaluationField) {
	policy := resolveInspectionSignalPolicyLocator(locator.policy.state, inspection)
	if policy.status != locatorResolutionFound {
		return cli.SignalAction{}, policy.status, policy.detail
	}
	if locator.index < 0 {
		return cli.SignalAction{}, locatorResolutionInvalid, observed(semanticSubjectSignalAction, observationInvalidLocator)
	}
	if locator.index >= len(policy.value.Actions) {
		return cli.SignalAction{}, locatorResolutionAbsent, observed(semanticSubjectSignalAction, observationAbsent)
	}
	return policy.value.Actions[locator.index], locatorResolutionFound, evaluationText("")
}

func signalActionEndpointLocator(action SignalActionLocator) SignalEndpointLocator {
	state := &signalEndpointLocatorState{
		mode:        signalEndpointLocatorAction,
		policy:      action.policy.state,
		actionIndex: action.index,
	}

	locator := SignalEndpointLocator{
		state: state,
	}
	return locator
}

func resolveExpectedSignalEndpointSchema(state *signalEndpointLocatorState, schema *cli.Schema) (cli.SignalEndpointID, locatorResolutionStatus, evaluationField) {
	if schema == nil || state == nil || state.mode == signalEndpointLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectSignalEndpoint, observationInvalidLocator)
	}
	switch state.mode {
	case signalEndpointLocatorID:
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectSignalEndpoint, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	case signalEndpointLocatorAction:
		policy := resolveSchemaSignalPolicyLocator(state.policy, schema)
		if policy.status != locatorResolutionFound {
			return 0, policy.status, policy.detail
		}
		if state.actionIndex < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectSignalAction, observationInvalidLocator)
		}
		if state.actionIndex >= len(policy.value.Actions) {
			return 0, locatorResolutionAbsent, observed(semanticSubjectSignalAction, observationAbsent)
		}
		id := policy.value.Actions[state.actionIndex].EndpointID
		if id == 0 {
			return 0, locatorResolutionAbsent, observed(semanticSubjectSignalAction, observationNoSignalEndpointRelation)
		}
		return id, locatorResolutionFound, evaluationText("")
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectSignalEndpoint, observationInvalidLocator)
	}
}

func resolveExpectedSignalEndpointReference(state *signalEndpointLocatorState, reference cli.Reference) (cli.SignalEndpointID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == signalEndpointLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectSignalEndpoint, observationInvalidLocator)
	}
	switch state.mode {
	case signalEndpointLocatorID:
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectSignalEndpoint, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	case signalEndpointLocatorAction:
		policy := resolveReferenceSignalPolicyLocator(state.policy, reference)
		if policy.status != locatorResolutionFound {
			return 0, policy.status, policy.detail
		}
		if state.actionIndex < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectSignalAction, observationInvalidLocator)
		}
		if state.actionIndex >= len(policy.value.Actions) {
			return 0, locatorResolutionAbsent, observed(semanticSubjectSignalAction, observationAbsent)
		}
		id := policy.value.Actions[state.actionIndex].EndpointID
		if id == 0 {
			return 0, locatorResolutionAbsent, observed(semanticSubjectSignalAction, observationNoSignalEndpointRelation)
		}
		return id, locatorResolutionFound, evaluationText("")
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectSignalEndpoint, observationInvalidLocator)
	}
}

func resolveExpectedSignalEndpointInspection(state *signalEndpointLocatorState, inspection cli.Inspection) (cli.SignalEndpointID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == signalEndpointLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectSignalEndpoint, observationInvalidLocator)
	}
	if state.mode == signalEndpointLocatorID && state.id == 0 {
		return 0, locatorResolutionInvalid, observed(semanticSubjectSignalEndpoint, observationInvalidLocator)
	}
	return resolveTypedSignalEndpointID(state, inspection)
}

func signalPolicyIDLocator(id cli.SignalPolicyID) SignalPolicyLocator {
	state := &signalPolicyLocatorState{
		mode: signalPolicyLocatorID,
		id:   id,
	}

	locator := SignalPolicyLocator{
		state: state,
	}
	return locator
}

func signalEndpointIDLocator(id cli.SignalEndpointID) SignalEndpointLocator {
	state := &signalEndpointLocatorState{
		mode: signalEndpointLocatorID,
		id:   id,
	}

	locator := SignalEndpointLocator{
		state: state,
	}
	return locator
}

func executionAdmissionSignalPolicyIndexLocator(admission *executionAdmissionLocatorState, index int) SignalPolicyLocator {
	if admission == nil {
		return SignalPolicyLocator{}
	}
	state := &signalPolicyLocatorState{
		mode:  signalPolicyLocatorExecutionAdmissionIndex,
		index: index,
	}

	locator := SignalPolicyLocator{
		state: state,
	}
	return locator
}

func executionAdmissionSignalEndpointIndexLocator(admission *executionAdmissionLocatorState, index int) SignalEndpointLocator {
	if admission == nil {
		return SignalEndpointLocator{}
	}
	state := &signalEndpointLocatorState{
		mode:  signalEndpointLocatorExecutionAdmissionIndex,
		index: index,
	}

	locator := SignalEndpointLocator{
		state: state,
	}
	return locator
}

func resolveTypedSignalPolicyID(state *signalPolicyLocatorState, inspection cli.Inspection) (cli.SignalPolicyID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == signalPolicyLocatorInvalid {
		return 0, locatorResolutionInvalid, evaluationText("invalid signal-policy locator")
	}
	switch state.mode {
	case signalPolicyLocatorID:
		return state.id, locatorResolutionFound, evaluationText("")
	case signalPolicyLocatorCommandIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, evaluationText("invalid signal-policy locator")
		}
		command := resolveInspectionCommandLocator(state.command, inspection)
		if command.status != locatorResolutionFound {
			return 0, command.status, command.detail
		}
		if state.index < len(command.value.SignalPolicies) {
			return command.value.SignalPolicies[state.index], locatorResolutionFound, evaluationText("")
		}
		if detail, incomplete := policyInspectionNamespaceIncomplete(command.value, inspection); incomplete {
			return 0, locatorResolutionNotProvable, detail
		}
		return 0, locatorResolutionAbsent, evaluationText("command signal policy is absent")
	case signalPolicyLocatorExecutionAdmissionIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, evaluationText("invalid signal-policy locator")
		}
		if state.index < len(inspection.ExecutionAdmission.SignalPolicies) {
			return inspection.ExecutionAdmission.SignalPolicies[state.index], locatorResolutionFound, evaluationText("")
		}
		if inspection.Truncated {
			return 0, locatorResolutionNotProvable, observed(semanticSubjectInspection, observationTruncated)
		}
		return 0, locatorResolutionAbsent, evaluationText("execution admission signal policy is absent")
	default:
		return 0, locatorResolutionInvalid, evaluationText("invalid signal-policy locator")
	}
}

func resolveTypedSignalEndpointID(state *signalEndpointLocatorState, inspection cli.Inspection) (cli.SignalEndpointID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == signalEndpointLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectSignalEndpoint, observationInvalidLocator)
	}
	switch state.mode {
	case signalEndpointLocatorID:
		return state.id, locatorResolutionFound, evaluationText("")
	case signalEndpointLocatorStatus:
		var matched cli.SignalEndpointID
		count := 0
		for id, endpoint := range inspection.SignalEndpoints {
			if endpoint.Status != state.status {
				continue
			}
			matched = id
			count++
		}
		switch count {
		case 0:
			if inspection.Truncated {
				return 0, locatorResolutionNotProvable, observed(semanticSubjectInspection, observationTruncated)
			}
			return 0, locatorResolutionAbsent, evaluationText(fmt.Sprintf("no observed signal endpoint has status %v", state.status))
		case 1:
			return matched, locatorResolutionFound, evaluationText("")
		default:
			return 0, locatorResolutionAmbiguous, evaluationText(fmt.Sprintf("%d observed signal endpoints have status %v", count, state.status))
		}
	case signalEndpointLocatorExecutionAdmissionIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectSignalEndpoint, observationInvalidLocator)
		}
		if state.index < len(inspection.ExecutionAdmission.SignalEndpoints) {
			return inspection.ExecutionAdmission.SignalEndpoints[state.index], locatorResolutionFound, evaluationText("")
		}
		if inspection.Truncated {
			return 0, locatorResolutionNotProvable, observed(semanticSubjectInspection, observationTruncated)
		}
		return 0, locatorResolutionAbsent, evaluationText("execution admission signal endpoint is absent")
	case signalEndpointLocatorAction:
		policyID, status, detail := resolveTypedSignalPolicyID(state.policy, inspection)
		if status != locatorResolutionFound {
			return 0, status, detail
		}
		policy, ok := inspection.SignalPolicies[policyID]
		if !ok {
			if inspection.Truncated {
				return 0, locatorResolutionNotProvable, observed(semanticSubjectInspection, observationTruncated)
			}
			return 0, locatorResolutionAbsent, observed(semanticSubjectSignalPolicy, observationAbsent)
		}
		if state.actionIndex < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectSignalAction, observationInvalidLocator)
		}
		if state.actionIndex >= len(policy.Actions) {
			if inspection.Truncated {
				return 0, locatorResolutionNotProvable, observed(semanticSubjectInspection, observationTruncated)
			}
			return 0, locatorResolutionAbsent, observed(semanticSubjectSignalAction, observationAbsent)
		}
		endpointID := policy.Actions[state.actionIndex].EndpointID
		if endpointID == 0 {
			return 0, locatorResolutionAbsent, observed(semanticSubjectSignalAction, observationNoSignalEndpointRelation)
		}
		return endpointID, locatorResolutionFound, evaluationText("")
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectSignalEndpoint, observationInvalidLocator)
	}
}

func executionAdmissionSignalEndpointsExactlyExpectation(admission *executionAdmissionLocatorState, locators []SignalEndpointLocator) InspectionExpectation {
	states := make([]*signalEndpointLocatorState, len(locators))
	for index, locator := range locators {
		states[index] = locator.state
	}
	value := executionAdmissionSignalEndpointsExactlyExpectationValue{
		admission: admission,
		locators:  states,
	}
	return value
}

func signalPolicyStatusExpectation(locator SignalPolicyLocator, status cli.SignalPolicyStatus) InspectionExpectation {
	value := signalPolicyStatusExpectationValue{
		locator: locator.state,
		status:  status,
	}
	return value
}

func signalPolicyActivationOutcomeExpectation(locator SignalPolicyLocator, outcome cli.EvaluationOutcome) InspectionExpectation {
	value := signalPolicyActivationOutcomeExpectationValue{
		locator: locator.state,
		outcome: outcome,
	}
	return value
}

func signalEndpointStatusExpectation(locator SignalEndpointLocator, status cli.SignalEndpointStatus) InspectionExpectation {
	value := signalEndpointStatusExpectationValue{
		locator: locator.state,
		status:  status,
	}
	return value
}

func signalEndpointPoliciesExpectation(locator SignalEndpointLocator, effective bool, policies []SignalPolicyLocator) InspectionExpectation {
	states := make([]*signalPolicyLocatorState, len(policies))
	for index, policy := range policies {
		states[index] = policy.state
	}
	expectation := signalEndpointPoliciesExactExpectation{
		locator:   locator.state,
		effective: effective,
		states:    states,
	}
	return expectation
}

func signalPolicyLocatorStates(values []SignalPolicyLocator) []*signalPolicyLocatorState {
	states := make([]*signalPolicyLocatorState, len(values))
	for index, value := range values {
		states[index] = value.state
	}
	return states
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func expectedSchemaSignalPolicyID(state *signalPolicyLocatorState, schema *cli.Schema) (cli.SignalPolicyID, locatorResolutionStatus, evaluationField) {
	if schema == nil || state == nil || state.mode == signalPolicyLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectSignalPolicy, observationInvalidLocator)
	}
	switch state.mode {
	case signalPolicyLocatorID:
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectSignalPolicy, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	case signalPolicyLocatorCommandIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectSignalPolicy, observationInvalidLocator)
		}
		command := resolveSchemaCommandLocator(state.command, schema)
		if command.status != locatorResolutionFound {
			return 0, command.status, command.detail
		}
		if state.index >= len(command.value.SignalPolicies) {
			return 0, locatorResolutionAbsent, observed(semanticSubjectSignalPolicy, observationAbsent)
		}
		return command.value.SignalPolicies[state.index], locatorResolutionFound, evaluationText("")
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectSignalPolicy, observationInvalidLocator)
	}
}

func expectedReferenceSignalPolicyID(state *signalPolicyLocatorState, reference cli.Reference) (cli.SignalPolicyID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == signalPolicyLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectSignalPolicy, observationInvalidLocator)
	}
	switch state.mode {
	case signalPolicyLocatorID:
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectSignalPolicy, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	case signalPolicyLocatorCommandIndex:
		if state.index < 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectSignalPolicy, observationInvalidLocator)
		}
		command := resolveReferenceCommandLocator(state.command, reference)
		if command.status != locatorResolutionFound {
			return 0, command.status, command.detail
		}
		if state.index >= len(command.value.SignalPolicies) {
			return 0, locatorResolutionAbsent, observed(semanticSubjectSignalPolicy, observationAbsent)
		}
		return command.value.SignalPolicies[state.index], locatorResolutionFound, evaluationText("")
	default:
		return 0, locatorResolutionInvalid, observed(semanticSubjectSignalPolicy, observationInvalidLocator)
	}
}

func expectedInspectionSignalPolicyID(state *signalPolicyLocatorState, inspection cli.Inspection) (cli.SignalPolicyID, locatorResolutionStatus, evaluationField) {
	if state == nil || state.mode == signalPolicyLocatorInvalid {
		return 0, locatorResolutionInvalid, observed(semanticSubjectSignalPolicy, observationInvalidLocator)
	}
	if state.mode == signalPolicyLocatorID {
		if state.id == 0 {
			return 0, locatorResolutionInvalid, observed(semanticSubjectSignalPolicy, observationInvalidLocator)
		}
		return state.id, locatorResolutionFound, evaluationText("")
	}
	return resolveTypedSignalPolicyID(state, inspection)
}

func signalPropertyResult(status locatorResolutionStatus, subject, detail, name, expected, actual evaluationField, equal bool) evaluationResult {
	qualified := qualifySubject(subject, name)
	expectedField := property(name, expected)
	observedField := property(name, actual)
	return declarationResolvedComparison(status, qualified, detail, expectedField, observedField, equal)
}

func signalActionEndpointResult(value cli.SignalAction, status locatorResolutionStatus, detail evaluationField, expected cli.SignalEndpointID, expectedStatus locatorResolutionStatus, expectedDetail evaluationField, truncated bool) evaluationResult {
	if status != locatorResolutionFound {
		return missingResult(evaluationText("signal action"), evaluationText("endpoint"), detail, truncated)
	}
	if expectedStatus != locatorResolutionFound {
		return missingResult(evaluationText("signal action"), evaluationText("endpoint"), expectedDetail, truncated)
	}
	if value.EndpointID == expected {
		return provedResult()
	}
	return disprovedResult(evaluationText("signal action"), evaluationText("endpoint matches expected signal-endpoint identity"), evaluationText("endpoint has different signal-endpoint identity"))
}

func signalPolicyLocatorLabel(state *signalPolicyLocatorState) evaluationField {
	if state == nil {
		return semanticSubjectSignalPolicy
	}
	switch state.mode {
	case signalPolicyLocatorID:
		return evaluationText(fmt.Sprintf("signal policy id %d", state.id))
	case signalPolicyLocatorCommandIndex:
		return qualifySubject(commandLocatorLabel(state.command), subjectAt(semanticSubjectSignalPolicy, state.index))
	case signalPolicyLocatorExecutionAdmissionIndex:
		return evaluationText(fmt.Sprintf("execution admission signal policy %d", state.index))
	default:
		return semanticSubjectSignalPolicy
	}
}

func signalEndpointLocatorLabel(state *signalEndpointLocatorState) evaluationField {
	if state == nil {
		return semanticSubjectSignalEndpoint
	}
	switch state.mode {
	case signalEndpointLocatorID:
		return evaluationText(fmt.Sprintf("signal endpoint id %d", state.id))
	case signalEndpointLocatorStatus:
		return evaluationText(fmt.Sprintf("signal endpoint status %v", state.status))
	case signalEndpointLocatorExecutionAdmissionIndex:
		return evaluationText(fmt.Sprintf("execution admission signal endpoint %d", state.index))
	case signalEndpointLocatorAction:
		return qualifySubject(signalPolicyLocatorLabel(state.policy), evaluationText(fmt.Sprintf("signal action endpoint at action %d", state.actionIndex)))
	default:
		return semanticSubjectSignalEndpoint
	}
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func resolveSchemaSignalPolicyLocator(state *signalPolicyLocatorState, schema *cli.Schema) locatorResolution[cli.SignalPolicySchema] {
	var resolution locatorResolution[cli.SignalPolicySchema]
	label := signalPolicyLocatorLabel(state)
	if schema == nil || state == nil || state.mode == signalPolicyLocatorInvalid {
		resolution = locatorResolution[cli.SignalPolicySchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectSignalPolicy, observationInvalidLocator),
		}
		return resolution
	}
	var id cli.SignalPolicyID
	switch state.mode {
	case signalPolicyLocatorID:
		id = state.id
	case signalPolicyLocatorCommandIndex:
		if state.index < 0 {
			resolution = locatorResolution[cli.SignalPolicySchema]{
				status:  locatorResolutionInvalid,
				subject: label,
				detail:  observed(semanticSubjectSignalPolicy, observationInvalidLocator),
			}
			return resolution
		}
		command := resolveSchemaCommandLocator(state.command, schema)
		if command.status != locatorResolutionFound {
			resolution = locatorResolution[cli.SignalPolicySchema]{
				status:  command.status,
				subject: label,
				detail:  command.detail,
			}
			return resolution
		}
		if state.index >= len(command.value.SignalPolicies) {
			resolution = locatorResolution[cli.SignalPolicySchema]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectSignalPolicy, observationAbsent),
			}
			return resolution
		}
		id = command.value.SignalPolicies[state.index]
	default:
		resolution = locatorResolution[cli.SignalPolicySchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectSignalPolicy, observationInvalidLocator),
		}
		return resolution
	}
	policy, ok := schema.SignalPolicies[id]
	if !ok {
		resolution = locatorResolution[cli.SignalPolicySchema]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectSignalPolicy, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.SignalPolicySchema]{
		status:  locatorResolutionFound,
		value:   policy,
		subject: label,
	}
	return resolution
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func resolveReferenceSignalPolicyLocator(state *signalPolicyLocatorState, reference cli.Reference) locatorResolution[cli.SignalPolicyReference] {
	var resolution locatorResolution[cli.SignalPolicyReference]
	label := signalPolicyLocatorLabel(state)
	if state == nil || state.mode == signalPolicyLocatorInvalid {
		resolution = locatorResolution[cli.SignalPolicyReference]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectSignalPolicy, observationInvalidLocator),
		}
		return resolution
	}
	var id cli.SignalPolicyID
	switch state.mode {
	case signalPolicyLocatorID:
		id = state.id
	case signalPolicyLocatorCommandIndex:
		if state.index < 0 {
			resolution = locatorResolution[cli.SignalPolicyReference]{
				status:  locatorResolutionInvalid,
				subject: label,
				detail:  observed(semanticSubjectSignalPolicy, observationInvalidLocator),
			}
			return resolution
		}
		command := resolveReferenceCommandLocator(state.command, reference)
		if command.status != locatorResolutionFound {
			resolution = locatorResolution[cli.SignalPolicyReference]{
				status:  command.status,
				subject: label,
				detail:  command.detail,
			}
			return resolution
		}
		if state.index >= len(command.value.SignalPolicies) {
			resolution = locatorResolution[cli.SignalPolicyReference]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectSignalPolicy, observationAbsent),
			}
			return resolution
		}
		id = command.value.SignalPolicies[state.index]
	default:
		resolution = locatorResolution[cli.SignalPolicyReference]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectSignalPolicy, observationInvalidLocator),
		}
		return resolution
	}
	policy, ok := reference.SignalPolicies[id]
	if !ok {
		resolution = locatorResolution[cli.SignalPolicyReference]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectSignalPolicy, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.SignalPolicyReference]{
		status:  locatorResolutionFound,
		value:   policy,
		subject: label,
	}
	return resolution
}

func resolveInspectionSignalPolicyLocator(state *signalPolicyLocatorState, inspection cli.Inspection) locatorResolution[cli.SignalPolicyInspection] {
	var resolution locatorResolution[cli.SignalPolicyInspection]
	label := signalPolicyLocatorLabel(state)
	id, status, detail := resolveTypedSignalPolicyID(state, inspection)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.SignalPolicyInspection]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	policy, ok := inspection.SignalPolicies[id]
	if !ok {
		if inspection.Truncated {
			resolution = locatorResolution[cli.SignalPolicyInspection]{
				status:  locatorResolutionNotProvable,
				subject: label,
				detail:  observed(semanticSubjectInspection, observationTruncated),
			}
			return resolution
		}
		resolution = locatorResolution[cli.SignalPolicyInspection]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectSignalPolicy, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.SignalPolicyInspection]{
		status:  locatorResolutionFound,
		value:   policy,
		subject: label,
	}
	return resolution
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func resolveSchemaSignalEndpointLocator(state *signalEndpointLocatorState, schema *cli.Schema) locatorResolution[cli.SignalEndpoint] {
	var resolution locatorResolution[cli.SignalEndpoint]
	label := signalEndpointLocatorLabel(state)
	if schema == nil || state == nil || state.mode == signalEndpointLocatorInvalid {
		resolution = locatorResolution[cli.SignalEndpoint]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectSignalEndpoint, observationInvalidLocator),
		}
		return resolution
	}
	var id cli.SignalEndpointID
	switch state.mode {
	case signalEndpointLocatorID:
		id = state.id
	case signalEndpointLocatorAction:
		policy := resolveSchemaSignalPolicyLocator(state.policy, schema)
		if policy.status != locatorResolutionFound {
			resolution = locatorResolution[cli.SignalEndpoint]{
				status:  policy.status,
				subject: label,
				detail:  policy.detail,
			}
			return resolution
		}
		if state.actionIndex < 0 {
			resolution = locatorResolution[cli.SignalEndpoint]{
				status:  locatorResolutionInvalid,
				subject: label,
				detail:  observed(semanticSubjectSignalAction, observationInvalidLocator),
			}
			return resolution
		}
		if state.actionIndex >= len(policy.value.Actions) {
			resolution = locatorResolution[cli.SignalEndpoint]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectSignalAction, observationAbsent),
			}
			return resolution
		}
		id = policy.value.Actions[state.actionIndex].EndpointID
		if id == 0 {
			resolution = locatorResolution[cli.SignalEndpoint]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectSignalAction, observationNoSignalEndpointRelation),
			}
			return resolution
		}
	default:
		resolution = locatorResolution[cli.SignalEndpoint]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectSignalEndpoint, observationInvalidLocator),
		}
		return resolution
	}
	endpoint, ok := schema.SignalEndpoints[id]
	if !ok {
		var detail evaluationField = observed(semanticSubjectSignalEndpoint, observationAbsent)
		if state.mode == signalEndpointLocatorAction {
			detail = evaluationText("related signal endpoint is absent")
		}
		resolution = locatorResolution[cli.SignalEndpoint]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	resolution = locatorResolution[cli.SignalEndpoint]{
		status:  locatorResolutionFound,
		value:   endpoint,
		subject: label,
	}
	return resolution
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func resolveReferenceSignalEndpointLocator(state *signalEndpointLocatorState, reference cli.Reference) locatorResolution[cli.SignalEndpoint] {
	var resolution locatorResolution[cli.SignalEndpoint]
	label := signalEndpointLocatorLabel(state)
	if state == nil || state.mode == signalEndpointLocatorInvalid {
		resolution = locatorResolution[cli.SignalEndpoint]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectSignalEndpoint, observationInvalidLocator),
		}
		return resolution
	}
	var id cli.SignalEndpointID
	switch state.mode {
	case signalEndpointLocatorID:
		id = state.id
	case signalEndpointLocatorAction:
		policy := resolveReferenceSignalPolicyLocator(state.policy, reference)
		if policy.status != locatorResolutionFound {
			resolution = locatorResolution[cli.SignalEndpoint]{
				status:  policy.status,
				subject: label,
				detail:  policy.detail,
			}
			return resolution
		}
		if state.actionIndex < 0 {
			resolution = locatorResolution[cli.SignalEndpoint]{
				status:  locatorResolutionInvalid,
				subject: label,
				detail:  observed(semanticSubjectSignalAction, observationInvalidLocator),
			}
			return resolution
		}
		if state.actionIndex >= len(policy.value.Actions) {
			resolution = locatorResolution[cli.SignalEndpoint]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectSignalAction, observationAbsent),
			}
			return resolution
		}
		id = policy.value.Actions[state.actionIndex].EndpointID
		if id == 0 {
			resolution = locatorResolution[cli.SignalEndpoint]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  observed(semanticSubjectSignalAction, observationNoSignalEndpointRelation),
			}
			return resolution
		}
	default:
		resolution = locatorResolution[cli.SignalEndpoint]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectSignalEndpoint, observationInvalidLocator),
		}
		return resolution
	}
	endpoint, ok := reference.SignalEndpoints[id]
	if !ok {
		var detail evaluationField = observed(semanticSubjectSignalEndpoint, observationAbsent)
		if state.mode == signalEndpointLocatorAction {
			detail = evaluationText("related signal endpoint is absent")
		}
		resolution = locatorResolution[cli.SignalEndpoint]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	resolution = locatorResolution[cli.SignalEndpoint]{
		status:  locatorResolutionFound,
		value:   endpoint,
		subject: label,
	}
	return resolution
}

func resolveInspectionSignalEndpointLocator(state *signalEndpointLocatorState, inspection cli.Inspection) locatorResolution[cli.SignalEndpointInspection] {
	var resolution locatorResolution[cli.SignalEndpointInspection]
	label := signalEndpointLocatorLabel(state)
	id, status, detail := resolveTypedSignalEndpointID(state, inspection)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.SignalEndpointInspection]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	endpoint, ok := inspection.SignalEndpoints[id]
	if !ok {
		if inspection.Truncated {
			resolution = locatorResolution[cli.SignalEndpointInspection]{
				status:  locatorResolutionNotProvable,
				subject: label,
				detail:  observed(semanticSubjectInspection, observationTruncated),
			}
			return resolution
		}
		var detail evaluationField = observed(semanticSubjectSignalEndpoint, observationAbsent)
		if state.mode == signalEndpointLocatorAction {
			detail = evaluationText("related signal endpoint is absent")
		}
		resolution = locatorResolution[cli.SignalEndpointInspection]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	resolution = locatorResolution[cli.SignalEndpointInspection]{
		status:  locatorResolutionFound,
		value:   endpoint,
		subject: label,
	}
	return resolution
}

func (signalEndpointCountExpectation) schemaExpectation() {}

func (signalEndpointCountExpectation) referenceExpectation() {}

func (expectation signalEndpointCountExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	return equalFieldResult(evaluationText("signal endpoints"), evaluationText("count"), evaluationText(strconv.Itoa(expectation.expected)), evaluationText(strconv.Itoa(len(schema.SignalEndpoints))), len(schema.SignalEndpoints) == expectation.expected)
}

func (expectation signalEndpointCountExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return equalFieldResult(evaluationText("signal endpoints"), evaluationText("count"), evaluationText(strconv.Itoa(expectation.expected)), evaluationText(strconv.Itoa(len(reference.SignalEndpoints))), len(reference.SignalEndpoints) == expectation.expected)
}

func (signalPolicyStatusExpectationValue) inspectionExpectation() {}

func (expectation signalPolicyStatusExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	id, status, actual := resolveTypedSignalPolicyID(expectation.locator, inspection)
	var label evaluationField = semanticSubjectSignalPolicy
	expected := evaluationText(fmt.Sprintf("status %v", expectation.status))
	if status == locatorResolutionNotProvable {
		return notProvableResult(label, expected, actual)
	}
	if status != locatorResolutionFound {
		return disprovedResult(label, expected, actual)
	}
	policy, ok := inspection.SignalPolicies[id]
	if !ok {
		if inspection.Truncated {
			return notProvableResult(label, expected, observed(semanticSubjectInspection, observationTruncated))
		}
		return disprovedResult(label, expected, observed(semanticSubjectSignalPolicy, observationAbsent))
	}
	if policy.Status == expectation.status {
		return provedResult()
	}
	if policy.Sensitive {
		return disprovedResult(
			label,
			evaluationText("redacted expected signal-policy property"),
			evaluationText("redacted observed signal-policy property"),
		)
	}
	return disprovedResult(label, expected, evaluationText(fmt.Sprintf("status %v", policy.Status)))
}

func (signalPolicyActivationOutcomeExpectationValue) inspectionExpectation() {}

func (expectation signalPolicyActivationOutcomeExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	id, status, actual := resolveTypedSignalPolicyID(expectation.locator, inspection)
	var label evaluationField = semanticSubjectSignalPolicy
	expected := evaluationText(fmt.Sprintf("activation %v", expectation.outcome))
	if status == locatorResolutionNotProvable {
		return notProvableResult(label, expected, actual)
	}
	if status != locatorResolutionFound {
		return disprovedResult(label, expected, actual)
	}
	policy, ok := inspection.SignalPolicies[id]
	if !ok {
		if inspection.Truncated {
			return notProvableResult(label, expected, observed(semanticSubjectInspection, observationTruncated))
		}
		return disprovedResult(label, expected, observed(semanticSubjectSignalPolicy, observationAbsent))
	}
	if policy.ActivationOutcome == expectation.outcome {
		return provedResult()
	}
	if policy.Sensitive {
		return disprovedResult(label, evaluationText("redacted expected signal-policy property"), evaluationText("redacted observed signal-policy property"))
	}
	return disprovedResult(label, expected, evaluationText(fmt.Sprintf("activation %v", policy.ActivationOutcome)))
}

func (signalEndpointStatusExpectationValue) inspectionExpectation() {}

func (expectation signalEndpointStatusExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	id, status, actual := resolveTypedSignalEndpointID(expectation.locator, inspection)
	label := "signal endpoint"
	if expectation.locator != nil && expectation.locator.mode == signalEndpointLocatorAction {
		label = "signal action endpoint"
	}
	expected := evaluationText(fmt.Sprintf("status %v", expectation.status))
	if status == locatorResolutionNotProvable {
		return notProvableResult(evaluationText(label), expected, actual)
	}
	if status != locatorResolutionFound {
		return disprovedResult(evaluationText(label), expected, actual)
	}
	endpoint, ok := inspection.SignalEndpoints[id]
	if !ok {
		if inspection.Truncated {
			return notProvableResult(evaluationText(label), expected, observed(semanticSubjectInspection, observationTruncated))
		}
		return disprovedResult(evaluationText(label), expected, evaluationText("signal endpoint is absent"))
	}
	if endpoint.Status == expectation.status {
		return provedResult()
	}
	return disprovedResult(evaluationText(label), expected, evaluationText(fmt.Sprintf("status %v", endpoint.Status)))
}

func (signalEndpointPoliciesExactExpectation) inspectionExpectation() {}

func (expectation signalEndpointPoliciesExactExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	id, status, actual := resolveTypedSignalEndpointID(expectation.locator, inspection)
	label := "signal endpoint policies"
	if expectation.locator != nil && expectation.locator.mode == signalEndpointLocatorAction {
		label = "signal action endpoint policies"
	}
	if status == locatorResolutionNotProvable {
		return notProvableResult(evaluationText(label), evaluationText("exact authoritative policy collection"), actual)
	}
	if status != locatorResolutionFound {
		return disprovedResult(evaluationText(label), evaluationText("existing signal endpoint"), actual)
	}
	endpoint, ok := inspection.SignalEndpoints[id]
	if !ok {
		if inspection.Truncated {
			return notProvableResult(evaluationText(label), evaluationText("exact authoritative policy collection"), observed(semanticSubjectInspection, observationTruncated))
		}
		return disprovedResult(evaluationText(label), evaluationText("existing signal endpoint"), evaluationText("signal endpoint is absent"))
	}
	if inspection.Truncated {
		return notProvableResult(evaluationText(label), evaluationText("exact authoritative policy collection"), observed(semanticSubjectInspection, observationTruncated))
	}
	actualIDs := endpoint.BlockedPolicies
	collection := semanticLabelBlocked
	if expectation.effective {
		actualIDs = endpoint.EffectivePolicies
		collection = "effective"
	}
	if len(actualIDs) != len(expectation.states) {
		return disprovedResult(
			evaluationText(label),
			evaluationText(fmt.Sprintf("%d %s policies", len(expectation.states), collection)),
			evaluationText(fmt.Sprintf("%d policies", len(actualIDs))),
		)
	}
	for index, state := range expectation.states {
		policyID, policyStatus, detail := resolveTypedSignalPolicyID(state, inspection)
		if policyStatus != locatorResolutionFound {
			return disprovedResult(evaluationText(label), evaluationText(fmt.Sprintf("policy at position %d", index)), detail)
		}
		if actualIDs[index] != policyID {
			return disprovedResult(
				evaluationText(label),
				evaluationText(fmt.Sprintf("policy %v at index %d", policyID, index)),
				evaluationText(fmt.Sprintf("policy %v", actualIDs[index])),
			)
		}
	}
	return provedResult()
}

func (observedSignalEndpointCountExpectation) inspectionExpectation() {}

func (expectation observedSignalEndpointCountExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	actual := len(inspection.SignalEndpoints)
	expected := int(expectation)
	if actual == expected {
		return provedResult()
	}
	return disprovedResult(
		evaluationText("observed signal endpoints"),
		evaluationText(fmt.Sprintf("count=%d", expected)),
		evaluationText(fmt.Sprintf("count=%d; signal-integration-status=%v", actual, inspection.SignalSource.Status)),
	)
}

func (signalPolicyExistenceExpectationValue) schemaExpectation() {}

func (signalPolicyExistenceExpectationValue) referenceExpectation() {}

func (signalPolicyExistenceExpectationValue) inspectionExpectation() {}

func (expectation signalPolicyExistenceExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	return declarationExistenceResult(resolveSchemaSignalPolicyLocator(expectation.locator, schema), expectation.expected)
}

func (expectation signalPolicyExistenceExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return declarationExistenceResult(resolveReferenceSignalPolicyLocator(expectation.locator, reference), expectation.expected)
}

func (expectation signalPolicyExistenceExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return declarationExistenceResult(resolveInspectionSignalPolicyLocator(expectation.locator, inspection), expectation.expected)
}

func (signalPolicyReachabilityExpectationValue) schemaExpectation() {}

func (signalPolicyReachabilityExpectationValue) referenceExpectation() {}

func (signalPolicyReachabilityExpectationValue) inspectionExpectation() {}

func (expectation signalPolicyReachabilityExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaSignalPolicyLocator(expectation.locator, schema)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("reachability"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Reachability)), resolution.value.Reachability == expectation.expected)
}

func (expectation signalPolicyReachabilityExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceSignalPolicyLocator(expectation.locator, reference)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("reachability"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Reachability)), resolution.value.Reachability == expectation.expected)
}

func (expectation signalPolicyReachabilityExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionSignalPolicyLocator(expectation.locator, inspection)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("reachability"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Reachability)), resolution.value.Reachability == expectation.expected)
}

func (signalPolicySensitiveExpectationValue) schemaExpectation() {}

func (signalPolicySensitiveExpectationValue) referenceExpectation() {}

func (signalPolicySensitiveExpectationValue) inspectionExpectation() {}

func (expectation signalPolicySensitiveExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaSignalPolicyLocator(expectation.locator, schema)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, semanticRequirementSensitive, evaluationBool(true), evaluationBool(resolution.value.Sensitive), resolution.value.Sensitive)
}

func (expectation signalPolicySensitiveExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceSignalPolicyLocator(expectation.locator, reference)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, semanticRequirementSensitive, evaluationBool(true), evaluationBool(resolution.value.Sensitive), resolution.value.Sensitive)
}

func (expectation signalPolicySensitiveExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionSignalPolicyLocator(expectation.locator, inspection)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, semanticRequirementSensitive, evaluationBool(true), evaluationBool(resolution.value.Sensitive), resolution.value.Sensitive)
}

func (signalPolicySourceCapabilitiesExpectationValue) schemaExpectation() {}

func (signalPolicySourceCapabilitiesExpectationValue) referenceExpectation() {}

func (signalPolicySourceCapabilitiesExpectationValue) inspectionExpectation() {}

func (expectation signalPolicySourceCapabilitiesExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaSignalPolicyLocator(expectation.locator, schema)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("source-capabilities"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.RequiredSourceCapabilities)), resolution.value.RequiredSourceCapabilities == expectation.expected)
}

func (expectation signalPolicySourceCapabilitiesExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceSignalPolicyLocator(expectation.locator, reference)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("source-capabilities"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.RequiredSourceCapabilities)), resolution.value.RequiredSourceCapabilities == expectation.expected)
}

func (expectation signalPolicySourceCapabilitiesExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionSignalPolicyLocator(expectation.locator, inspection)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("source-capabilities"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.RequiredSourceCapabilities)), resolution.value.RequiredSourceCapabilities == expectation.expected)
}

func (signalPolicyHostCapabilitiesExpectationValue) schemaExpectation() {}

func (signalPolicyHostCapabilitiesExpectationValue) referenceExpectation() {}

func (signalPolicyHostCapabilitiesExpectationValue) inspectionExpectation() {}

func (expectation signalPolicyHostCapabilitiesExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaSignalPolicyLocator(expectation.locator, schema)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("host-capabilities"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.RequiredHostCapabilities)), resolution.value.RequiredHostCapabilities == expectation.expected)
}

func (expectation signalPolicyHostCapabilitiesExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceSignalPolicyLocator(expectation.locator, reference)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("host-capabilities"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.RequiredHostCapabilities)), resolution.value.RequiredHostCapabilities == expectation.expected)
}

func (expectation signalPolicyHostCapabilitiesExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionSignalPolicyLocator(expectation.locator, inspection)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("host-capabilities"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.RequiredHostCapabilities)), resolution.value.RequiredHostCapabilities == expectation.expected)
}

func (signalPolicyAvailabilityExpectationValue) referenceExpectation() {}

func (expectation signalPolicyAvailabilityExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceSignalPolicyLocator(expectation.locator, reference)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("available"), evaluationBool(expectation.expected), evaluationBool(resolution.value.Available), resolution.value.Available == expectation.expected)
}

func (signalPolicyVisibilityExpectationValue) referenceExpectation() {}

func (expectation signalPolicyVisibilityExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceSignalPolicyLocator(expectation.locator, reference)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("visible"), evaluationBool(expectation.expected), evaluationBool(resolution.value.Visible), resolution.value.Visible == expectation.expected)
}

func (signalEndpointExistenceExpectationValue) schemaExpectation() {}

func (signalEndpointExistenceExpectationValue) referenceExpectation() {}

func (signalEndpointExistenceExpectationValue) inspectionExpectation() {}

func (expectation signalEndpointExistenceExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	return declarationExistenceResult(resolveSchemaSignalEndpointLocator(expectation.locator, schema), expectation.expected)
}

func (expectation signalEndpointExistenceExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return declarationExistenceResult(resolveReferenceSignalEndpointLocator(expectation.locator, reference), expectation.expected)
}

func (expectation signalEndpointExistenceExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return declarationExistenceResult(resolveInspectionSignalEndpointLocator(expectation.locator, inspection), expectation.expected)
}

func (signalPolicyOccurrenceExpectationValue) schemaExpectation() {}

func (signalPolicyOccurrenceExpectationValue) referenceExpectation() {}

func (signalPolicyOccurrenceExpectationValue) inspectionExpectation() {}

func (expectation signalPolicyOccurrenceExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaSignalPolicyLocator(expectation.locator, schema)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("occurrence"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Occurrence)), resolution.value.Occurrence == expectation.expected)
}

func (expectation signalPolicyOccurrenceExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceSignalPolicyLocator(expectation.locator, reference)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("occurrence"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Occurrence)), resolution.value.Occurrence == expectation.expected)
}

func (expectation signalPolicyOccurrenceExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionSignalPolicyLocator(expectation.locator, inspection)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("occurrence"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Occurrence)), resolution.value.Occurrence == expectation.expected)
}

func (signalPolicyAggregationExpectationValue) schemaExpectation() {}

func (signalPolicyAggregationExpectationValue) referenceExpectation() {}

func (signalPolicyAggregationExpectationValue) inspectionExpectation() {}

func (expectation signalPolicyAggregationExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaSignalPolicyLocator(expectation.locator, schema)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("aggregation"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Aggregation)), resolution.value.Aggregation == expectation.expected)
}

func (expectation signalPolicyAggregationExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceSignalPolicyLocator(expectation.locator, reference)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("aggregation"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Aggregation)), resolution.value.Aggregation == expectation.expected)
}

func (expectation signalPolicyAggregationExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionSignalPolicyLocator(expectation.locator, inspection)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("aggregation"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Aggregation)), resolution.value.Aggregation == expectation.expected)
}

func (signalPolicyDeferredExpectationValue) schemaExpectation() {}

func (signalPolicyDeferredExpectationValue) referenceExpectation() {}

func (signalPolicyDeferredExpectationValue) inspectionExpectation() {}

func (expectation signalPolicyDeferredExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaSignalPolicyLocator(expectation.locator, schema)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("deferred"), evaluationBool(expectation.expected), evaluationBool(resolution.value.Deferred), resolution.value.Deferred == expectation.expected)
}

func (expectation signalPolicyDeferredExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceSignalPolicyLocator(expectation.locator, reference)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("deferred"), evaluationBool(expectation.expected), evaluationBool(resolution.value.Deferred), resolution.value.Deferred == expectation.expected)
}

func (expectation signalPolicyDeferredExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionSignalPolicyLocator(expectation.locator, inspection)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("deferred"), evaluationBool(expectation.expected), evaluationBool(resolution.value.Deferred), resolution.value.Deferred == expectation.expected)
}

func (signalPolicyContinueMatchingExpectationValue) schemaExpectation() {}

func (signalPolicyContinueMatchingExpectationValue) referenceExpectation() {}

func (signalPolicyContinueMatchingExpectationValue) inspectionExpectation() {}

func (expectation signalPolicyContinueMatchingExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaSignalPolicyLocator(expectation.locator, schema)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("continue-matching"), evaluationBool(expectation.expected), evaluationBool(resolution.value.ContinueMatching), resolution.value.ContinueMatching == expectation.expected)
}

func (expectation signalPolicyContinueMatchingExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceSignalPolicyLocator(expectation.locator, reference)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("continue-matching"), evaluationBool(expectation.expected), evaluationBool(resolution.value.ContinueMatching), resolution.value.ContinueMatching == expectation.expected)
}

func (expectation signalPolicyContinueMatchingExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionSignalPolicyLocator(expectation.locator, inspection)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("continue-matching"), evaluationBool(expectation.expected), evaluationBool(resolution.value.ContinueMatching), resolution.value.ContinueMatching == expectation.expected)
}

func (signalPolicyHintExpectationValue) schemaExpectation() {}

func (signalPolicyHintExpectationValue) referenceExpectation() {}

func (signalPolicyHintExpectationValue) inspectionExpectation() {}

func (expectation signalPolicyHintExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaSignalPolicyLocator(expectation.locator, schema)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("hint"), evaluationText(expectation.expected), evaluationText(resolution.value.Hint), resolution.value.Hint == expectation.expected)
}

func (expectation signalPolicyHintExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceSignalPolicyLocator(expectation.locator, reference)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("hint"), evaluationText(expectation.expected), evaluationText(resolution.value.Hint), resolution.value.Hint == expectation.expected)
}

func (expectation signalPolicyHintExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionSignalPolicyLocator(expectation.locator, inspection)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("hint"), evaluationText(expectation.expected), evaluationText(resolution.value.Hint), resolution.value.Hint == expectation.expected)
}

func (signalPolicyActionCountExpectationValue) schemaExpectation() {}

func (signalPolicyActionCountExpectationValue) referenceExpectation() {}

func (signalPolicyActionCountExpectationValue) inspectionExpectation() {}

func (expectation signalPolicyActionCountExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaSignalPolicyLocator(expectation.locator, schema)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("action-count"), evaluationText(strconv.Itoa(expectation.expected)), evaluationText(strconv.Itoa(len(resolution.value.Actions))), len(resolution.value.Actions) == expectation.expected)
}

func (expectation signalPolicyActionCountExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceSignalPolicyLocator(expectation.locator, reference)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("action-count"), evaluationText(strconv.Itoa(expectation.expected)), evaluationText(strconv.Itoa(len(resolution.value.Actions))), len(resolution.value.Actions) == expectation.expected)
}

func (expectation signalPolicyActionCountExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionSignalPolicyLocator(expectation.locator, inspection)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("action-count"), evaluationText(strconv.Itoa(expectation.expected)), evaluationText(strconv.Itoa(len(resolution.value.Actions))), len(resolution.value.Actions) == expectation.expected)
}

func (signalPolicyOriginExpectationValue) schemaExpectation() {}

func (signalPolicyOriginExpectationValue) referenceExpectation() {}

func (signalPolicyOriginExpectationValue) inspectionExpectation() {}

func (expectation signalPolicyOriginExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaSignalPolicyLocator(expectation.locator, schema)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("origin"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Origin)), resolution.value.Origin == expectation.expected)
}

func (expectation signalPolicyOriginExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceSignalPolicyLocator(expectation.locator, reference)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("origin"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Origin)), resolution.value.Origin == expectation.expected)
}

func (expectation signalPolicyOriginExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionSignalPolicyLocator(expectation.locator, inspection)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("origin"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Origin)), resolution.value.Origin == expectation.expected)
}

func (signalEndpointCapacityExpectationValue) schemaExpectation() {}

func (signalEndpointCapacityExpectationValue) referenceExpectation() {}

func (signalEndpointCapacityExpectationValue) inspectionExpectation() {}

func (expectation signalEndpointCapacityExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaSignalEndpointLocator(expectation.locator, schema)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("capacity"), evaluationText(strconv.Itoa(expectation.expected)), evaluationText(strconv.Itoa(resolution.value.Capacity)), resolution.value.Capacity == expectation.expected)
}

func (expectation signalEndpointCapacityExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceSignalEndpointLocator(expectation.locator, reference)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("capacity"), evaluationText(strconv.Itoa(expectation.expected)), evaluationText(strconv.Itoa(resolution.value.Capacity)), resolution.value.Capacity == expectation.expected)
}

func (expectation signalEndpointCapacityExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionSignalEndpointLocator(expectation.locator, inspection)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("capacity"), evaluationText(strconv.Itoa(expectation.expected)), evaluationText(strconv.Itoa(resolution.value.Capacity)), resolution.value.Capacity == expectation.expected)
}

func (signalEndpointOriginExpectationValue) schemaExpectation() {}

func (signalEndpointOriginExpectationValue) referenceExpectation() {}

func (signalEndpointOriginExpectationValue) inspectionExpectation() {}

func (expectation signalEndpointOriginExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaSignalEndpointLocator(expectation.locator, schema)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("origin"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Origin)), resolution.value.Origin == expectation.expected)
}

func (expectation signalEndpointOriginExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceSignalEndpointLocator(expectation.locator, reference)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("origin"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Origin)), resolution.value.Origin == expectation.expected)
}

func (expectation signalEndpointOriginExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionSignalEndpointLocator(expectation.locator, inspection)
	return signalPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("origin"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Origin)), resolution.value.Origin == expectation.expected)
}

func (signalSelectorExpectationValue) schemaExpectation() {}

func (signalSelectorExpectationValue) referenceExpectation() {}

func (signalSelectorExpectationValue) inspectionExpectation() {}

func (expectation signalSelectorExpectationValue) evaluate(status locatorResolutionStatus, subject, detail evaluationField, selector cli.SignalSelector) evaluationResult {
	switch expectation.property {
	case signalSelectorExpectationKind:
		return signalPropertyResult(status, subject, detail, evaluationText("selector kind"), evaluationText(fmt.Sprint(expectation.kind)), evaluationText(fmt.Sprint(selector.Kind)), selector.Kind == expectation.kind)
	case signalSelectorExpectationExcludedSender:
		return signalPropertyResult(status, subject, detail, evaluationText("selector excluded-sender"), evaluationText(expectation.excludedSender), evaluationText(selector.ExcludedSender), selector.ExcludedSender == expectation.excludedSender)
	default:
	}
	if status != locatorResolutionFound {
		return expectedRelationFailure(status, subject, evaluationText("selector "+expectation.collectionName()), detail)
	}
	switch expectation.property {
	case signalSelectorExpectationSignals:
		if slices.Equal(selector.Signals, expectation.signals) {
			return provedResult()
		}
		return orderedCollectionMismatch(subject, "selector signals", expectation.signals, selector.Signals)
	case signalSelectorExpectationExcludedSignals:
		if slices.Equal(selector.ExcludedSignals, expectation.signals) {
			return provedResult()
		}
		return orderedCollectionMismatch(subject, "selector excluded-signals", expectation.signals, selector.ExcludedSignals)
	case signalSelectorExpectationOrigins:
		if slices.Equal(selector.Origins, expectation.origins) {
			return provedResult()
		}
		return orderedCollectionMismatch(subject, "selector origins", expectation.origins, selector.Origins)
	case signalSelectorExpectationExcludedOrigins:
		if slices.Equal(selector.ExcludedOrigins, expectation.origins) {
			return provedResult()
		}
		return orderedCollectionMismatch(subject, "selector excluded-origins", expectation.origins, selector.ExcludedOrigins)
	case signalSelectorExpectationPhases:
		if slices.Equal(selector.Phases, expectation.phases) {
			return provedResult()
		}
		return orderedCollectionMismatch(subject, "selector phases", expectation.phases, selector.Phases)
	default:
		panic("clitest: invalid signal-selector expectation")
	}
}

func (expectation signalSelectorExpectationValue) collectionName() string {
	switch expectation.property {
	case signalSelectorExpectationSignals:
		return "signals"
	case signalSelectorExpectationExcludedSignals:
		return "excluded-signals"
	case signalSelectorExpectationOrigins:
		return "origins"
	case signalSelectorExpectationExcludedOrigins:
		return "excluded-origins"
	case signalSelectorExpectationPhases:
		return "phases"
	default:
		return "property"
	}
}

func (expectation signalSelectorExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaSignalPolicyLocator(expectation.locator, schema)
	return expectation.evaluate(resolution.status, resolution.subject, resolution.detail, resolution.value.Selector)
}

func (expectation signalSelectorExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceSignalPolicyLocator(expectation.locator, reference)
	return expectation.evaluate(resolution.status, resolution.subject, resolution.detail, resolution.value.Selector)
}

func (expectation signalSelectorExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionSignalPolicyLocator(expectation.locator, inspection)
	return expectation.evaluate(resolution.status, resolution.subject, resolution.detail, resolution.value.Selector)
}

func (signalActionExpectationValue) schemaExpectation() {}

func (signalActionExpectationValue) referenceExpectation() {}

func (signalActionExpectationValue) inspectionExpectation() {}

func (expectation signalActionExpectationValue) evaluate(value cli.SignalAction, status locatorResolutionStatus, detail evaluationField, truncated bool) evaluationResult {
	name := expectation.propertyName()
	if status != locatorResolutionFound {
		return missingResult(evaluationText("signal action"), name, detail, truncated)
	}
	subject := evaluationText("signal action")
	switch expectation.property {
	case signalActionExpectationKind:
		return equalFieldResult(subject, name, evaluationText(fmt.Sprint(expectation.kind)), evaluationText(fmt.Sprint(value.Kind)), value.Kind == expectation.kind)
	case signalActionExpectationName:
		return equalFieldResult(subject, name, evaluationText(expectation.name), evaluationText(value.Name), value.Name == expectation.name)
	case signalActionExpectationSignal:
		return equalFieldResult(subject, name, evaluationText(fmt.Sprint(expectation.signal)), evaluationText(fmt.Sprint(value.Signal)), value.Signal == expectation.signal)
	case signalActionExpectationSignalRedacted:
		return equalFieldResult(subject, name, evaluationBool(expectation.boolean), evaluationBool(value.SignalRedacted), value.SignalRedacted == expectation.boolean)
	case signalActionExpectationDelayed:
		return equalFieldResult(subject, name, evaluationBool(expectation.boolean), evaluationBool(value.Delayed), value.Delayed == expectation.boolean)
	case signalActionExpectationDelay:
		return equalFieldResult(subject, name, evaluationText(expectation.delay.String()), evaluationText(value.Delay.String()), value.Delay == expectation.delay)
	case signalActionExpectationDelayRedacted:
		return equalFieldResult(subject, name, evaluationBool(expectation.boolean), evaluationBool(value.DelayRedacted), value.DelayRedacted == expectation.boolean)
	case signalActionExpectationExitCode:
		return equalFieldResult(subject, name, evaluationText(strconv.Itoa(expectation.exitCode)), evaluationText(strconv.Itoa(value.ExitCode)), value.ExitCode == expectation.exitCode)
	case signalActionExpectationSensitive:
		return equalFieldResult(subject, name, evaluationBool(expectation.boolean), evaluationBool(value.Sensitive), value.Sensitive == expectation.boolean)
	default:
		panic("clitest: invalid signal-action expectation")
	}
}

func (expectation signalActionExpectationValue) propertyName() evaluationField {
	switch expectation.property {
	case signalActionExpectationKind:
		return evaluationText("kind")
	case signalActionExpectationName:
		return evaluationText("name")
	case signalActionExpectationSignal:
		return evaluationText("signal")
	case signalActionExpectationSignalRedacted:
		return evaluationText("signal-redacted")
	case signalActionExpectationDelayed:
		return evaluationText("delayed")
	case signalActionExpectationDelay:
		return evaluationText("delay")
	case signalActionExpectationDelayRedacted:
		return evaluationText("delay-redacted")
	case signalActionExpectationExitCode:
		return evaluationText("exit-code")
	case signalActionExpectationSensitive:
		return semanticRequirementSensitive
	default:
		return evaluationText("signal action")
	}
}

func (expectation signalActionExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	value, status, detail := resolveSignalActionSchema(expectation.locator, schema)
	return expectation.evaluate(value, status, detail, false)
}

func (expectation signalActionExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	value, status, detail := resolveSignalActionReference(expectation.locator, reference)
	return expectation.evaluate(value, status, detail, false)
}

func (expectation signalActionExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	value, status, detail := resolveSignalActionInspection(expectation.locator, inspection)
	return expectation.evaluate(value, status, detail, inspection.Truncated)
}

func (signalActionEndpointExpectationValue) schemaExpectation() {}

func (signalActionEndpointExpectationValue) referenceExpectation() {}

func (signalActionEndpointExpectationValue) inspectionExpectation() {}

func (expectation signalActionEndpointExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	value, status, detail := resolveSignalActionSchema(expectation.locator, schema)
	expected, expectedStatus, expectedDetail := resolveExpectedSignalEndpointSchema(expectation.expected, schema)
	return signalActionEndpointResult(value, status, detail, expected, expectedStatus, expectedDetail, false)
}

func (expectation signalActionEndpointExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	value, status, detail := resolveSignalActionReference(expectation.locator, reference)
	expected, expectedStatus, expectedDetail := resolveExpectedSignalEndpointReference(expectation.expected, reference)
	return signalActionEndpointResult(value, status, detail, expected, expectedStatus, expectedDetail, false)
}

func (expectation signalActionEndpointExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	value, status, detail := resolveSignalActionInspection(expectation.locator, inspection)
	expected, expectedStatus, expectedDetail := resolveExpectedSignalEndpointInspection(expectation.expected, inspection)
	return signalActionEndpointResult(value, status, detail, expected, expectedStatus, expectedDetail, inspection.Truncated)
}

func (signalActionValueExpectationValue) schemaExpectation() {}

func (signalActionValueExpectationValue) referenceExpectation() {}

func (signalActionValueExpectationValue) inspectionExpectation() {}

func (expectation signalActionValueExpectationValue) relationName() string {
	if expectation.delay {
		return "delay-value"
	}
	return "signal-value"
}

func (expectation signalActionValueExpectationValue) actual(action cli.SignalAction) cli.ValueID {
	if expectation.delay {
		return action.DelayValueID
	}
	return action.SignalValueID
}

func (expectation signalActionValueExpectationValue) compare(action cli.SignalAction, status locatorResolutionStatus, detail evaluationField, expected cli.ValueID, expectedStatus locatorResolutionStatus, truncated bool) evaluationResult {
	name := expectation.relationName()
	if status != locatorResolutionFound {
		return missingResult(evaluationText("signal action"), evaluationText(name), detail, truncated)
	}
	if expectedStatus != locatorResolutionFound {
		return relationLocatorFailureOutcome("signal action", name, "expected value locator", expectedStatus)
	}
	if expectation.actual(action) == expected {
		return provedResult()
	}
	return disprovedResult(evaluationText("signal action"), evaluationText(name+" matches expected value identity"), evaluationText(name+" has different value identity"))
}

func (expectation signalActionValueExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	action, status, detail := resolveSignalActionSchema(expectation.locator, schema)
	expected, expectedStatus, _ := resolveSchemaValueID(expectation.expected, schema)
	return expectation.compare(action, status, detail, expected, expectedStatus, false)
}

func (expectation signalActionValueExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	action, status, detail := resolveSignalActionReference(expectation.locator, reference)
	expected, expectedStatus, _ := resolveReferenceValueID(expectation.expected, reference)
	return expectation.compare(action, status, detail, expected, expectedStatus, false)
}

func (expectation signalActionValueExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	action, status, detail := resolveSignalActionInspection(expectation.locator, inspection)
	expected, expectedStatus, _ := resolveInspectionValueID(expectation.expected, inspection)
	return expectation.compare(action, status, detail, expected, expectedStatus, inspection.Truncated)
}
