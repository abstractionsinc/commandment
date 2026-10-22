package clitest

import (
	"fmt"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
)

// SignalIntegrationLocator identifies the operation-wide signal integration
// summary in Inspection for clitest semantic expectation. It describes validated
// source/host integration facts and does not subscribe to signals. Its zero
// value is invalid.
type SignalIntegrationLocator struct {
	_     [0]func()
	state *signalIntegrationLocatorState
}

// ExecutionAdmissionLocator identifies the operation-wide declarative execution-admission
// state in Inspection for clitest semantic expectation. It describes readiness and
// effective execution relationships; it is not an executable plan or trace. Its
// zero value is invalid.
type ExecutionAdmissionLocator struct {
	_     [0]func()
	state *executionAdmissionLocatorState
}

// MiddlewareLocator identifies one effective middleware entry in execution admission
// for clitest semantic expectation. Its zero value is invalid.
type MiddlewareLocator struct {
	_     [0]func()
	state *middlewareLocatorState
}

// ExecutionAdmission returns the root locator for Inspection execution-admission state.
func ExecutionAdmission() ExecutionAdmissionLocator {
	state := &executionAdmissionLocatorState{}
	locator := ExecutionAdmissionLocator{
		state: state,
	}
	return locator
}

// SignalIntegration returns the root locator for Inspection signal-integration state.
func SignalIntegration() SignalIntegrationLocator {
	state := &signalIntegrationLocatorState{}
	locator := SignalIntegrationLocator{
		state: state,
	}
	return locator
}

// MiddlewareAt returns the middleware locator at index in the effective public middleware
// order.
func (locator ExecutionAdmissionLocator) MiddlewareAt(index int) MiddlewareLocator {
	return middlewareIndexLocator(locator.state, index)
}

// SignalPolicyAt returns the signal-policy locator at index in the effective execution-admission
// policy order.
func (locator ExecutionAdmissionLocator) SignalPolicyAt(index int) SignalPolicyLocator {
	return executionAdmissionSignalPolicyIndexLocator(locator.state, index)
}

// SignalEndpointAt returns the signal-endpoint locator at index in the effective
// execution-admission endpoint order.
func (locator ExecutionAdmissionLocator) SignalEndpointAt(index int) SignalEndpointLocator {
	return executionAdmissionSignalEndpointIndexLocator(locator.state, index)
}

// Status returns an expectation that checks the declarative execution-admission status in
// Inspection.
func (locator ExecutionAdmissionLocator) Status(status cli.ExecutionAdmissionStatus) InspectionExpectation {
	return executionAdmissionStatusExpectation(locator.state, status)
}

// Ready returns an expectation that checks execution admission is Ready; it does not execute
// middleware or the handler.
func (locator ExecutionAdmissionLocator) Ready() InspectionExpectation {
	return executionAdmissionStatusExpectation(locator.state, cli.ExecutionAdmissionReady)
}

// Blocked returns an expectation that checks execution admission is Blocked; it does not
// manufacture an execution attempt.
func (locator ExecutionAdmissionLocator) Blocked() InspectionExpectation {
	return executionAdmissionStatusExpectation(locator.state, cli.ExecutionAdmissionBlocked)
}

// Handler returns an expectation that checks the selected handler belongs to the requested
// canonical command.
func (locator ExecutionAdmissionLocator) Handler(command CommandLocator) InspectionExpectation {
	return executionAdmissionHandlerExpectation(locator.state, command)
}

// MiddlewareExactly returns an expectation that checks the located execution admission has
// exactly the requested effective middleware entries in authoritative public order.
func (locator ExecutionAdmissionLocator) MiddlewareExactly(middleware ...MiddlewareLocator) InspectionExpectation {
	return executionAdmissionMiddlewareExactlyExpectation(locator.state, middleware)
}

// SignalPoliciesExactly returns an expectation that checks the located execution admission has
// exactly the requested effective signal policies in authoritative public order.
func (locator ExecutionAdmissionLocator) SignalPoliciesExactly(policies ...SignalPolicyLocator) InspectionExpectation {
	return executionAdmissionSignalPoliciesExactlyExpectation(locator.state, policies)
}

// SignalEndpointsExactly returns an expectation that checks the located execution admission
// has exactly the requested effective signal endpoints in authoritative public order.
func (locator ExecutionAdmissionLocator) SignalEndpointsExactly(endpoints ...SignalEndpointLocator) InspectionExpectation {
	return executionAdmissionSignalEndpointsExactlyExpectation(locator.state, endpoints)
}

// Status returns an expectation that checks the published signal-integration status.
func (locator SignalIntegrationLocator) Status(status cli.SignalIntegrationStatus) InspectionExpectation {
	expectation := signalIntegrationExpectation{
		locator:  locator.state,
		property: signalIntegrationExpectationStatus,
		status:   status,
	}
	return expectation
}

// Required returns an expectation that checks signal integration is required for the analyzed
// invocation.
func (locator SignalIntegrationLocator) Required() InspectionExpectation {
	expectation := signalIntegrationExpectation{
		locator:  locator.state,
		property: signalIntegrationExpectationRequired,
	}
	return expectation
}

// Supplied returns an expectation that checks a signal source integration was supplied for the
// analyzed invocation.
func (locator SignalIntegrationLocator) Supplied() InspectionExpectation {
	expectation := signalIntegrationExpectation{
		locator:  locator.state,
		property: signalIntegrationExpectationSupplied,
	}
	return expectation
}

// ResolvedSignalsExactly returns an expectation that checks the exact published
// resolved-signal sequence.
func (locator SignalIntegrationLocator) ResolvedSignalsExactly(signals ...cli.Signal) InspectionExpectation {
	expectation := signalIntegrationExpectation{
		locator:  locator.state,
		property: signalIntegrationExpectationResolvedSignals,
		signals:  slices.Clone(signals),
	}
	return expectation
}

// MissingSignalsExactly returns an expectation that checks the exact published missing-signal
// sequence.
func (locator SignalIntegrationLocator) MissingSignalsExactly(signals ...cli.Signal) InspectionExpectation {
	expectation := signalIntegrationExpectation{
		locator:  locator.state,
		property: signalIntegrationExpectationMissingSignals,
		signals:  slices.Clone(signals),
	}
	return expectation
}

// RequiredSourceCapabilities returns an expectation that checks the exact source capabilities
// required by the analyzed invocation.
func (locator SignalIntegrationLocator) RequiredSourceCapabilities(capabilities cli.SignalSourceCapabilities) InspectionExpectation {
	expectation := signalIntegrationExpectation{
		locator:            locator.state,
		property:           signalIntegrationExpectationRequiredSourceCapabilities,
		sourceCapabilities: capabilities,
	}
	return expectation
}

// MissingSourceCapabilities returns an expectation that checks the exact required source
// capabilities that the supplied integration lacks.
func (locator SignalIntegrationLocator) MissingSourceCapabilities(capabilities cli.SignalSourceCapabilities) InspectionExpectation {
	expectation := signalIntegrationExpectation{
		locator:            locator.state,
		property:           signalIntegrationExpectationMissingSourceCapabilities,
		sourceCapabilities: capabilities,
	}
	return expectation
}

// RequiredHostCapabilities returns an expectation that checks the exact host capabilities
// required by the analyzed invocation.
func (locator SignalIntegrationLocator) RequiredHostCapabilities(capabilities cli.SignalHostCapabilities) InspectionExpectation {
	expectation := signalIntegrationExpectation{
		locator:          locator.state,
		property:         signalIntegrationExpectationRequiredHostCapabilities,
		hostCapabilities: capabilities,
	}
	return expectation
}

// MissingHostCapabilities returns an expectation that checks the exact required host
// capabilities that the host integration lacks.
func (locator SignalIntegrationLocator) MissingHostCapabilities(capabilities cli.SignalHostCapabilities) InspectionExpectation {
	expectation := signalIntegrationExpectation{
		locator:          locator.state,
		property:         signalIntegrationExpectationMissingHostCapabilities,
		hostCapabilities: capabilities,
	}
	return expectation
}

// AvailableHostCapabilities returns an Inspection expectation that the signal integration reports exactly the supplied
// available host-capability set.
func (locator SignalIntegrationLocator) AvailableHostCapabilities(expected cli.SignalHostCapabilities) InspectionExpectation {
	expectation := signalIntegrationExpectation{
		locator:          locator.state,
		property:         signalIntegrationExpectationAvailableHostCapabilities,
		hostCapabilities: expected,
	}
	return expectation
}

// Scope returns an expectation that checks the located effective middleware has the requested
// public scope.
func (locator MiddlewareLocator) Scope(scope cli.MiddlewareScope) InspectionExpectation {
	return middlewareScopeExpectation(locator, scope)
}

// Command returns an expectation that checks the located effective middleware is owned by the
// requested canonical command.
func (locator MiddlewareLocator) Command(command CommandLocator) InspectionExpectation {
	return middlewareCommandExpectation(locator, command)
}

type executionAdmissionLocatorState struct {
	_ byte
}

type middlewareLocatorState struct {
	valid bool
	index int
}

type signalIntegrationLocatorState struct {
	_ byte
}

type executionAdmissionStatusExpectationValue struct {
	locator *executionAdmissionLocatorState
	status  cli.ExecutionAdmissionStatus
}

type executionAdmissionHandlerExpectationValue struct {
	locator *executionAdmissionLocatorState
	command *commandLocatorState
}

type middlewareScopeExpectationValue struct {
	locator *middlewareLocatorState
	scope   cli.MiddlewareScope
}

type middlewareCommandExpectationValue struct {
	middleware *middlewareLocatorState
	command    *commandLocatorState
}

type executionAdmissionMiddlewareExactlyExpectationValue struct {
	admission *executionAdmissionLocatorState
	locators  []*middlewareLocatorState
}

type executionAdmissionSignalPoliciesExactlyExpectationValue struct {
	admission *executionAdmissionLocatorState
	locators  []*signalPolicyLocatorState
}

type signalIntegrationExpectationProperty uint8

const (
	signalIntegrationExpectationInvalid signalIntegrationExpectationProperty = iota
	signalIntegrationExpectationStatus
	signalIntegrationExpectationRequired
	signalIntegrationExpectationSupplied
	signalIntegrationExpectationResolvedSignals
	signalIntegrationExpectationMissingSignals
	signalIntegrationExpectationRequiredSourceCapabilities
	signalIntegrationExpectationMissingSourceCapabilities
	signalIntegrationExpectationRequiredHostCapabilities
	signalIntegrationExpectationMissingHostCapabilities
	signalIntegrationExpectationAvailableHostCapabilities
)

type signalIntegrationExpectation struct {
	locator            *signalIntegrationLocatorState
	property           signalIntegrationExpectationProperty
	status             cli.SignalIntegrationStatus
	signals            []cli.Signal
	sourceCapabilities cli.SignalSourceCapabilities
	hostCapabilities   cli.SignalHostCapabilities
}

func tokenIndexLocator(index int) TokenLocator {
	state := &tokenLocatorState{
		valid: true,
		index: index,
	}
	locator := TokenLocator{
		state: state,
	}
	return locator
}

func middlewareIndexLocator(admission *executionAdmissionLocatorState, index int) MiddlewareLocator {
	if admission == nil {
		return MiddlewareLocator{}
	}
	state := &middlewareLocatorState{
		valid: true,
		index: index,
	}
	locator := MiddlewareLocator{
		state: state,
	}
	return locator
}

func resolveTokenAt(state *tokenLocatorState, inspection cli.Inspection) (cli.TokenInspection, locatorResolutionStatus, evaluationField, evaluationField) {
	if state == nil || !state.valid || state.index < 0 {
		return cli.TokenInspection{}, locatorResolutionInvalid, evaluationText("token"), evaluationText("invalid token locator")
	}
	label := evaluationText(fmt.Sprintf("token at index %d", state.index))
	if state.index < len(inspection.Tokens) {
		return inspection.Tokens[state.index], locatorResolutionFound, label, evaluationText("")
	}
	if inspection.Truncated {
		return cli.TokenInspection{}, locatorResolutionNotProvable, label, observed(semanticSubjectInspection, observationTruncated)
	}
	return cli.TokenInspection{}, locatorResolutionAbsent, label, evaluationText("token is absent")
}

func tokenRoleExpectation(locator TokenLocator, role cli.TokenRole) InspectionExpectation {
	expectation := tokenExpectation{
		locator: locator.state,
		kind:    tokenExpectationRole,
		role:    role,
	}
	return expectation
}

func tokenRejectionExpectation(locator TokenLocator, code cli.DiagnosticCode) InspectionExpectation {
	expectation := tokenExpectation{
		locator:   locator.state,
		kind:      tokenExpectationRejection,
		rejection: code,
	}
	return expectation
}

func tokenSpellingExpectation(locator TokenLocator, spelling string) InspectionExpectation {
	expectation := tokenExpectation{
		locator:  locator.state,
		kind:     tokenExpectationSpelling,
		spelling: spelling,
	}
	return expectation
}

func tokenMatchExpectation(locator TokenLocator, match cli.NameMatchKind) InspectionExpectation {
	expectation := tokenExpectation{
		locator: locator.state,
		kind:    tokenExpectationMatch,
		match:   match,
	}
	return expectation
}

func tokenRedactedExpectation(locator TokenLocator) InspectionExpectation {
	expectation := tokenExpectation{
		locator: locator.state,
		kind:    tokenExpectationRedacted,
	}
	return expectation
}

func tokenOccurrenceIndexExpectation(locator TokenLocator, index int) InspectionExpectation {
	expectation := tokenExpectation{
		locator: locator.state,
		kind:    tokenExpectationOccurrenceIndex,
		index:   index,
	}
	return expectation
}

func tokenElementIndexExpectation(locator TokenLocator, index int) InspectionExpectation {
	expectation := tokenExpectation{
		locator: locator.state,
		kind:    tokenExpectationElementIndex,
		index:   index,
	}
	return expectation
}

func tokenDelegatedIndexExpectation(locator TokenLocator, index int) InspectionExpectation {
	expectation := tokenExpectation{
		locator: locator.state,
		kind:    tokenExpectationDelegatedIndex,
		index:   index,
	}
	return expectation
}

func tokenTextExpectation(locator TokenLocator, text string) InspectionExpectation {
	expectation := tokenExpectation{
		locator: locator.state,
		kind:    tokenExpectationText,
		text:    text,
	}
	return expectation
}

func executionAdmissionStatusExpectation(locator *executionAdmissionLocatorState, status cli.ExecutionAdmissionStatus) InspectionExpectation {
	value := executionAdmissionStatusExpectationValue{
		locator: locator,
		status:  status,
	}
	return value
}

func executionAdmissionHandlerExpectation(locator *executionAdmissionLocatorState, command CommandLocator) InspectionExpectation {
	value := executionAdmissionHandlerExpectationValue{
		locator: locator,
		command: command.state,
	}
	return value
}

func resolveMiddleware(state *middlewareLocatorState, inspection cli.Inspection) (cli.MiddlewareInspection, locatorResolutionStatus, evaluationField, evaluationField) {
	if state == nil || !state.valid || state.index < 0 {
		return cli.MiddlewareInspection{}, locatorResolutionInvalid, evaluationText("middleware"), evaluationText("invalid middleware locator")
	}
	label := fmt.Sprintf("execution admission middleware at index %d", state.index)
	if state.index >= 0 && state.index < len(inspection.ExecutionAdmission.Middleware) {
		return inspection.ExecutionAdmission.Middleware[state.index], locatorResolutionFound, evaluationText(label), evaluationText("")
	}
	if inspection.Truncated {
		return cli.MiddlewareInspection{}, locatorResolutionNotProvable, evaluationText(label), observed(semanticSubjectInspection, observationTruncated)
	}
	return cli.MiddlewareInspection{}, locatorResolutionAbsent, evaluationText(label), evaluationText("middleware is absent")
}

func middlewareScopeExpectation(locator MiddlewareLocator, scope cli.MiddlewareScope) InspectionExpectation {
	value := middlewareScopeExpectationValue{
		locator: locator.state,
		scope:   scope,
	}
	return value
}

func middlewareCommandExpectation(locator MiddlewareLocator, command CommandLocator) InspectionExpectation {
	value := middlewareCommandExpectationValue{
		middleware: locator.state,
		command:    command.state,
	}
	return value
}

func executionAdmissionMiddlewareExactlyExpectation(admission *executionAdmissionLocatorState, locators []MiddlewareLocator) InspectionExpectation {
	states := make([]*middlewareLocatorState, len(locators))
	for index, locator := range locators {
		states[index] = locator.state
	}
	value := executionAdmissionMiddlewareExactlyExpectationValue{
		admission: admission,
		locators:  states,
	}
	return value
}

func executionAdmissionSignalPoliciesExactlyExpectation(admission *executionAdmissionLocatorState, locators []SignalPolicyLocator) InspectionExpectation {
	states := make([]*signalPolicyLocatorState, len(locators))
	for index, locator := range locators {
		states[index] = locator.state
	}
	value := executionAdmissionSignalPoliciesExactlyExpectationValue{
		admission: admission,
		locators:  states,
	}
	return value
}

func (executionAdmissionStatusExpectationValue) inspectionExpectation() {}

func (expectation executionAdmissionStatusExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	if expectation.locator == nil {
		return disprovedResult(evaluationText("execution admission"), evaluationText(fmt.Sprintf("status %v", expectation.status)), evaluationText("invalid execution admission locator"))
	}
	if inspection.ExecutionAdmission.Status == expectation.status {
		return provedResult()
	}
	return disprovedResult(
		evaluationText("execution admission"),
		evaluationText(fmt.Sprintf("status %v", expectation.status)),
		evaluationText(fmt.Sprintf("status %v", inspection.ExecutionAdmission.Status)),
	)
}

func (executionAdmissionHandlerExpectationValue) inspectionExpectation() {}

func (expectation executionAdmissionHandlerExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	if expectation.locator == nil {
		return disprovedResult(evaluationText("execution admission"), evaluationText("configured handler command"), evaluationText("invalid execution admission locator"))
	}
	id, status, actual := resolveSelectionCommand(expectation.command, inspection.Commands)
	if status == locatorResolutionNotProvable {
		return notProvableResult(evaluationText("execution admission"), evaluationText("configured handler command"), actual)
	}
	if status != locatorResolutionFound {
		return disprovedResult(evaluationText("execution admission"), evaluationText("configured handler command"), actual)
	}
	if inspection.ExecutionAdmission.HandlerCommand == id {
		return provedResult()
	}
	return disprovedResult(
		evaluationText("execution admission"),
		evaluationText(fmt.Sprintf("handler command %v", id)),
		evaluationText(fmt.Sprintf("handler command %v", inspection.ExecutionAdmission.HandlerCommand)),
	)
}

func (middlewareScopeExpectationValue) inspectionExpectation() {}

func (expectation middlewareScopeExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	middleware, status, label, actual := resolveMiddleware(expectation.locator, inspection)
	expected := evaluationText(fmt.Sprintf("scope %v", expectation.scope))
	if status == locatorResolutionNotProvable {
		return notProvableResult(label, expected, actual)
	}
	if status != locatorResolutionFound {
		return disprovedResult(label, expected, actual)
	}
	if middleware.Scope == expectation.scope {
		return provedResult()
	}
	return disprovedResult(label, expected, evaluationText(fmt.Sprintf("scope %v", middleware.Scope)))
}

func (middlewareCommandExpectationValue) inspectionExpectation() {}

func (expectation middlewareCommandExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	middleware, status, label, actual := resolveMiddleware(expectation.middleware, inspection)
	if status == locatorResolutionNotProvable {
		return notProvableResult(label, evaluationText("configured command"), actual)
	}
	if status != locatorResolutionFound {
		return disprovedResult(label, evaluationText("configured command"), actual)
	}
	id, commandStatus, commandActual := resolveSelectionCommand(expectation.command, inspection.Commands)
	if commandStatus == locatorResolutionNotProvable {
		return notProvableResult(label, evaluationText("configured command"), commandActual)
	}
	if commandStatus != locatorResolutionFound {
		return disprovedResult(label, evaluationText("configured command"), commandActual)
	}
	if middleware.Command == id {
		return provedResult()
	}
	return disprovedResult(label, evaluationText(fmt.Sprintf("command %v", id)), evaluationText(fmt.Sprintf("command %v", middleware.Command)))
}

func (executionAdmissionMiddlewareExactlyExpectationValue) inspectionExpectation() {}

func (expectation executionAdmissionMiddlewareExactlyExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	const subject = "execution admission middleware"

	if expectation.admission == nil {
		return disprovedResult(evaluationText(subject), evaluationText("valid execution admission coordinate"), evaluationText("invalid execution admission locator"))
	}
	if inspection.Truncated {
		return notProvableResult(evaluationText(subject), evaluationText("exact authoritative collection"), observed(semanticSubjectInspection, observationTruncated))
	}
	if len(inspection.ExecutionAdmission.Middleware) != len(expectation.locators) {
		return disprovedResult(
			evaluationText(subject),
			evaluationText(fmt.Sprintf("%d entries", len(expectation.locators))),
			evaluationText(fmt.Sprintf("%d entries", len(inspection.ExecutionAdmission.Middleware))),
		)
	}
	for index, state := range expectation.locators {
		_, status, _, actual := resolveMiddleware(state, inspection)
		if status != locatorResolutionFound {
			return disprovedResult(evaluationText(subject), evaluationText(fmt.Sprintf("locator at position %d", index)), actual)
		}
		if state.index != index {
			return disprovedResult(
				evaluationText(subject),
				evaluationText(fmt.Sprintf("middleware index %d", index)),
				evaluationText(fmt.Sprintf("locator resolves to index %d", state.index)),
			)
		}
	}
	return provedResult()
}

func (executionAdmissionSignalPoliciesExactlyExpectationValue) inspectionExpectation() {}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func (expectation executionAdmissionSignalPoliciesExactlyExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	if expectation.admission == nil {
		return disprovedResult(evaluationText("execution admission signal policies"), evaluationText("valid execution admission coordinate"), evaluationText("invalid execution admission locator"))
	}
	actual := inspection.ExecutionAdmission.SignalPolicies
	if len(actual) != len(expectation.locators) {
		return disprovedResult(evaluationText("execution admission signal policies"), evaluationText(fmt.Sprintf("%d policies", len(expectation.locators))), evaluationText(fmt.Sprintf("%d policies", len(actual))))
	}
	for index, locator := range expectation.locators {
		id, status, detail := resolveTypedSignalPolicyID(locator, inspection)
		if status == locatorResolutionNotProvable {
			return notProvableResult(evaluationText("execution admission signal policies"), evaluationText(fmt.Sprintf("policy at position %d", index)), detail)
		}
		if status != locatorResolutionFound {
			return disprovedResult(evaluationText("execution admission signal policies"), evaluationText(fmt.Sprintf("policy at position %d", index)), detail)
		}
		if actual[index] != id {
			return disprovedResult(evaluationText("execution admission signal policies"), evaluationText(fmt.Sprintf("policy %v at index %d", id, index)), evaluationText(fmt.Sprintf("policy %v", actual[index])))
		}
	}
	return provedResult()
}

func (executionAdmissionSignalEndpointsExactlyExpectationValue) inspectionExpectation() {}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func (expectation executionAdmissionSignalEndpointsExactlyExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	if expectation.admission == nil {
		return disprovedResult(evaluationText("execution admission signal endpoints"), evaluationText("valid execution admission coordinate"), evaluationText("invalid execution admission locator"))
	}
	actual := inspection.ExecutionAdmission.SignalEndpoints
	if len(actual) != len(expectation.locators) {
		return disprovedResult(evaluationText("execution admission signal endpoints"), evaluationText(fmt.Sprintf("%d endpoints", len(expectation.locators))), evaluationText(fmt.Sprintf("%d endpoints", len(actual))))
	}
	for index, locator := range expectation.locators {
		id, status, detail := resolveTypedSignalEndpointID(locator, inspection)
		if status == locatorResolutionNotProvable {
			return notProvableResult(evaluationText("execution admission signal endpoints"), evaluationText(fmt.Sprintf("endpoint at position %d", index)), detail)
		}
		if status != locatorResolutionFound {
			return disprovedResult(evaluationText("execution admission signal endpoints"), evaluationText(fmt.Sprintf("endpoint at position %d", index)), detail)
		}
		if actual[index] != id {
			return disprovedResult(evaluationText("execution admission signal endpoints"), evaluationText(fmt.Sprintf("endpoint %v at index %d", id, index)), evaluationText(fmt.Sprintf("endpoint %v", actual[index])))
		}
	}
	return provedResult()
}

func (signalIntegrationExpectation) inspectionExpectation() {}

func (expectation signalIntegrationExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	if expectation.locator == nil {
		return disprovedResult(evaluationText("signal integration"), evaluationText("valid signal-integration locator"), evaluationText("invalid signal-integration locator"))
	}
	source := inspection.SignalSource
	var expected, actual evaluationField
	var matched bool
	switch expectation.property {
	case signalIntegrationExpectationStatus:
		expected = evaluationText(fmt.Sprintf("status %v", expectation.status))
		actual = evaluationText(fmt.Sprintf("status %v", source.Status))
		matched = source.Status == expectation.status
	case signalIntegrationExpectationRequired:
		expected = evaluationText("required")
		actual = evaluationText(fmt.Sprintf("required=%t", source.Required))
		matched = source.Required
	case signalIntegrationExpectationSupplied:
		expected = evaluationText("supplied")
		actual = evaluationText(fmt.Sprintf("supplied=%t", source.Supplied))
		matched = source.Supplied
	case signalIntegrationExpectationResolvedSignals:
		expected = evaluationText("exact resolved signals")
		actual = evaluationText(fmt.Sprintf("%d resolved signals", len(source.ResolvedSignals)))
		matched = slices.Equal(source.ResolvedSignals, expectation.signals)
	case signalIntegrationExpectationMissingSignals:
		expected = evaluationText("exact missing signals")
		actual = evaluationText(fmt.Sprintf("%d missing signals", len(source.MissingSignals)))
		matched = slices.Equal(source.MissingSignals, expectation.signals)
	case signalIntegrationExpectationRequiredSourceCapabilities:
		expected = evaluationText(fmt.Sprintf("required source capabilities %v", expectation.sourceCapabilities))
		actual = evaluationText(fmt.Sprintf("required source capabilities %v", source.RequiredSourceCapabilities))
		matched = source.RequiredSourceCapabilities == expectation.sourceCapabilities
	case signalIntegrationExpectationMissingSourceCapabilities:
		expected = evaluationText(fmt.Sprintf("missing source capabilities %v", expectation.sourceCapabilities))
		actual = evaluationText(fmt.Sprintf("missing source capabilities %v", source.MissingSourceCapabilities))
		matched = source.MissingSourceCapabilities == expectation.sourceCapabilities
	case signalIntegrationExpectationRequiredHostCapabilities:
		expected = evaluationText(fmt.Sprintf("required host capabilities %v", expectation.hostCapabilities))
		actual = evaluationText(fmt.Sprintf("required host capabilities %v", source.RequiredHostCapabilities))
		matched = source.RequiredHostCapabilities == expectation.hostCapabilities
	case signalIntegrationExpectationMissingHostCapabilities:
		expected = evaluationText(fmt.Sprintf("missing host capabilities %v", expectation.hostCapabilities))
		actual = evaluationText(fmt.Sprintf("missing host capabilities %v", source.MissingHostCapabilities))
		matched = source.MissingHostCapabilities == expectation.hostCapabilities
	case signalIntegrationExpectationAvailableHostCapabilities:
		expected = evaluationText(fmt.Sprintf("available host capabilities %v", expectation.hostCapabilities))
		actual = evaluationText(fmt.Sprintf("available host capabilities %v", source.AvailableHostCapabilities))
		matched = source.AvailableHostCapabilities == expectation.hostCapabilities
	default:
		panic("clitest: invalid signal-integration expectation")
	}
	if matched {
		return provedResult()
	}
	return disprovedResult(evaluationText("signal integration"), expected, actual)
}
