package cli

import (
	"context"
	"slices"
	"time"
)

// SignalActionPlan is an immutable declared action sequence for a matching signal policy. Its zero value
// is not a valid committed plan, and immediate actions execute in deterministic declaration order.
type SignalActionPlan[C any] interface {
	signalActionPlan(config *C)
}

type signalCompletePlan[C any] struct {
	state *signalActionPlanState
}

func (signalCompletePlan[C]) signalActionPlan(*C) {}

type signalSequence[C any] struct {
	state *signalActionPlanState
}

func (signalSequence[C]) signalActionPlan(*C) {}

type signalEscalation[C any] struct {
	state *signalActionPlanState
}

type signalActions[C any] struct {
	config  *C
	session *declarationSession
	profile grammarProfileCaseRef
}

// Ignore completes a plan that consumes the match without application action.
func (actions signalActions[C]) Ignore() SignalActionPlan[C] { return signalIgnore(actions) }

// Cancel appends primary-command cancellation and returns a continuation because later actions in
// the same successful sequence are still permitted.
func (actions signalActions[C]) Cancel() signalSequence[C] { return signalCancel(actions) }

// Handle appends the application signal handler; signal callbacks are serialized per invocation but
// may overlap command execution, so application code owns synchronization of shared mutable
// configuration.
func (actions signalActions[C]) Handle(handler SignalHandler[C]) signalSequence[C] {
	return signalHandle(actions, handler)
}

// Forward appends forwarding of the matched signal event to target.
func (actions signalActions[C]) Forward(target SignalTarget[C]) signalSequence[C] {
	return signalForward(actions, target)
}

// Send appends delivery of the specified signal to target.
func (actions signalActions[C]) Send(target SignalTarget[C], signal Signal) signalSequence[C] {
	return signalSend(actions, target, signal)
}

// SendFrom appends delivery to target using the signal read from source when the action executes.
func (actions signalActions[C]) SendFrom(target SignalTarget[C], source Destination[C, Signal]) signalSequence[C] {
	return signalSendFrom(actions, target, source)
}

// Emit appends delivery of the structured SignalEvent to endpoint.
func (actions signalActions[C]) Emit(endpoint SignalEndpointDeclaration[C]) signalSequence[C] {
	return signalEmit(actions, endpoint)
}

// Abort completes the plan with the terminal abort action and exit code.
func (actions signalActions[C]) Abort(code int) SignalActionPlan[C] {
	return signalAbort(actions, code)
}

// Reraise completes the plan with the terminal host reraise action.
func (actions signalActions[C]) Reraise() SignalActionPlan[C] { return signalReraise(actions) }

// ThenCancel appends primary-command cancellation without ending the immediate action sequence.
func (sequence signalSequence[C]) ThenCancel() signalSequence[C] {
	return sequenceAppend(sequence, signalCancelAction())
}

// ThenHandle appends an application signal handler after the preceding immediate actions.
func (sequence signalSequence[C]) ThenHandle(handler SignalHandler[C]) signalSequence[C] {
	return sequenceAppend(sequence, eraseSignalHandler(handler))
}

// ThenForward appends forwarding after the preceding immediate actions.
func (sequence signalSequence[C]) ThenForward(target SignalTarget[C]) signalSequence[C] {
	return sequenceAppend(sequence, signalForwardAction(target))
}

// ThenSend appends delivery of signal to target after the preceding immediate actions.
func (sequence signalSequence[C]) ThenSend(target SignalTarget[C], signal Signal) signalSequence[C] {
	return sequenceAppend(sequence, signalSendAction(target, signal))
}

// ThenSendFrom appends target delivery using the signal read from source after the preceding
// immediate actions.
func (sequence signalSequence[C]) ThenSendFrom(target SignalTarget[C], source Destination[C, Signal]) signalSequence[C] {
	return sequenceAppend(sequence, signalSendFromAction(sequence.state, target, source))
}

// ThenEmit appends structured event delivery to endpoint after the preceding immediate actions.
func (sequence signalSequence[C]) ThenEmit(endpoint SignalEndpointDeclaration[C]) signalSequence[C] {
	return sequenceAppend(sequence, signalEmitAction(sequence.state, endpoint))
}

// ThenAbort completes the plan with a terminal abort action after all preceding immediate actions
// succeed.
func (sequence signalSequence[C]) ThenAbort(code int) SignalActionPlan[C] {
	return sequenceTerminal(sequence, signalAbortAction(code))
}

// ThenReraise completes the plan with terminal host reraise after all preceding immediate actions
// succeed.
func (sequence signalSequence[C]) ThenReraise() SignalActionPlan[C] {
	return sequenceTerminal(sequence, signalReraiseAction())
}

// After schedules a delayed escalation only after all preceding immediate actions succeed; the
// returned escalation must be completed with one terminal action.
func (sequence signalSequence[C]) After(duration time.Duration) signalEscalation[C] {
	return sequenceAfter(sequence, duration, nil)
}

// AfterFrom schedules delayed escalation using the duration stored in delay after all preceding
// immediate actions succeed.
func (sequence signalSequence[C]) AfterFrom(delay Destination[C, time.Duration]) signalEscalation[C] {
	return sequenceAfter(sequence, 0, signalActionPlanDestination(sequence.state, delay.handle))
}

// Cancel completes the delayed escalation with primary-command cancellation.
func (escalation signalEscalation[C]) Cancel() SignalActionPlan[C] {
	return escalationComplete(escalation, signalCancelAction())
}

// Handle completes the delayed escalation with an application signal handler.
func (escalation signalEscalation[C]) Handle(handler SignalHandler[C]) SignalActionPlan[C] {
	return escalationComplete(escalation, eraseSignalHandler(handler))
}

// Forward completes the delayed escalation by forwarding to target.
func (escalation signalEscalation[C]) Forward(target SignalTarget[C]) SignalActionPlan[C] {
	return escalationComplete(escalation, signalForwardAction(target))
}

// Send completes the delayed escalation by delivering signal to target.
func (escalation signalEscalation[C]) Send(target SignalTarget[C], signal Signal) SignalActionPlan[C] {
	return escalationComplete(escalation, signalSendAction(target, signal))
}

// SendFrom completes the delayed escalation by reading the signal from source and delivering it to
// target.
func (escalation signalEscalation[C]) SendFrom(target SignalTarget[C], source Destination[C, Signal]) SignalActionPlan[C] {
	return escalationComplete(escalation, signalSendFromAction(escalation.state, target, source))
}

// Abort completes the delayed escalation with terminal abort and exit code.
func (escalation signalEscalation[C]) Abort(code int) SignalActionPlan[C] {
	return escalationComplete(escalation, signalAbortAction(code))
}

// Reraise completes the delayed escalation with terminal host reraise.
func (escalation signalEscalation[C]) Reraise() SignalActionPlan[C] {
	return escalationComplete(escalation, signalReraiseAction())
}

func newSignalActionPlan[C any](actions signalActions[C], action authoringSignalAction) *signalActionPlanState {
	plan := &signalActionPlanState{
		config:  actions.config,
		actions: []authoringSignalAction{action},
		valid:   action.valid,
	}
	if actions.session != nil {
		actions.session.use(func(draft *declarationDraft, _ commandRef) {
			plan.owner = draft.identity
		})
	}
	plan.valid = plan.valid && plan.owner != nil
	return plan
}

func cloneSignalActionPlan(state *signalActionPlanState) *signalActionPlanState {
	if state == nil {
		return nil
	}
	cloned := *state
	cloned.actions = append([]authoringSignalAction(nil), state.actions...)
	return &cloned
}

func appendSignalAction(state *signalActionPlanState, action authoringSignalAction) *signalActionPlanState {
	plan := cloneSignalActionPlan(state)
	if plan == nil {
		return nil
	}
	if plan.pendingDelay {
		action.delayed = true
		action.delay = plan.delay
		action.delayValue = plan.delayValue
		plan.pendingDelay = false
		plan.delay = 0
		plan.delayValue = nil
	}
	plan.actions = append(plan.actions, action)
	plan.valid = plan.valid && action.valid
	return plan
}

func signalEscalate(state *signalActionPlanState, duration time.Duration, destination *destinationHandle) *signalActionPlanState {
	plan := cloneSignalActionPlan(state)
	if plan == nil {
		return nil
	}
	if plan.pendingDelay || (destination == nil && duration <= 0) ||
		destination != nil && !stagedDestinationOwnedBy(destination, plan.owner) {
		plan.valid = false
	} else {
		plan.pendingDelay = true
		plan.delay = duration
		plan.delayValue = destination
	}
	return plan
}

func signalIgnore[C any](actions signalActions[C]) SignalActionPlan[C] {
	action := authoringSignalAction{
		kind:  signalActionKindIgnore,
		valid: true,
	}
	plan := signalCompletePlan[C]{
		state: newSignalActionPlan(actions, action),
	}
	return plan
}

func signalCancel[C any](actions signalActions[C]) signalSequence[C] {
	action := authoringSignalAction{
		kind:  signalActionKindCancel,
		valid: true,
	}
	sequence := signalSequence[C]{
		state: newSignalActionPlan(actions, action),
	}
	return sequence
}

func signalHandle[C any](actions signalActions[C], handler SignalHandler[C]) signalSequence[C] {
	sequence := signalSequence[C]{
		state: newSignalActionPlan(actions, eraseSignalHandler(handler)),
	}
	return sequence
}

func signalForward[C any](actions signalActions[C], target SignalTarget[C]) signalSequence[C] {
	sequence := signalSequence[C]{
		state: newSignalActionPlan(actions, signalTargetAction(signalActionKindForward, target, Signal{}, nil)),
	}
	return sequence
}

func signalSend[C any](actions signalActions[C], target SignalTarget[C], signal Signal) signalSequence[C] {
	sequence := signalSequence[C]{
		state: newSignalActionPlan(actions, signalTargetAction(signalActionKindSend, target, signal, nil)),
	}
	return sequence
}

func signalSendFrom[C any](actions signalActions[C], target SignalTarget[C], source Destination[C, Signal]) signalSequence[C] {
	var owner *draftIdentity
	if actions.session != nil {
		actions.session.use(func(draft *declarationDraft, _ commandRef) {
			owner = draft.identity
		})
	}
	sequence := signalSequence[C]{
		state: newSignalActionPlan(
			actions,
			signalTargetAction(
				signalActionKindSend,
				target,
				Signal{},
				signalDestination(owner, source.handle),
			),
		),
	}
	return sequence
}

func signalEmit[C any](actions signalActions[C], endpoint SignalEndpointDeclaration[C]) signalSequence[C] {
	var owner *draftIdentity
	if actions.session != nil {
		actions.session.use(func(draft *declarationDraft, _ commandRef) {
			owner = draft.identity
		})
	}
	ref := signalEndpoint(owner, endpoint.endpoint)
	authoringSignalActionValue := authoringSignalAction{
		kind:     signalActionKindEmit,
		endpoint: ref,
		valid:    ref.valid(),
	}
	sequence := signalSequence[C]{
		state: newSignalActionPlan(actions, authoringSignalActionValue),
	}
	return sequence
}

func signalAbort[C any](actions signalActions[C], code int) SignalActionPlan[C] {
	authoringSignalActionValue := authoringSignalAction{
		kind:     signalActionKindAbort,
		exitCode: code,
		valid:    true,
	}
	plan := signalCompletePlan[C]{
		state: newSignalActionPlan(actions, authoringSignalActionValue),
	}
	return plan
}

func signalReraise[C any](actions signalActions[C]) SignalActionPlan[C] {
	action := authoringSignalAction{
		kind:  signalActionKindReraise,
		valid: true,
	}
	plan := signalCompletePlan[C]{
		state: newSignalActionPlan(actions, action),
	}
	return plan
}

func sequenceAppend[C any](sequence signalSequence[C], action authoringSignalAction) signalSequence[C] {
	result := signalSequence[C]{
		state: appendSignalAction(sequence.state, action),
	}
	return result
}

func sequenceTerminal[C any](sequence signalSequence[C], action authoringSignalAction) SignalActionPlan[C] {
	plan := signalCompletePlan[C]{
		state: appendSignalAction(sequence.state, action),
	}
	return plan
}

func sequenceAfter[C any](sequence signalSequence[C], duration time.Duration, destination *destinationHandle) signalEscalation[C] {
	escalation := signalEscalation[C]{
		state: signalEscalate(sequence.state, duration, destination),
	}
	return escalation
}

func escalationComplete[C any](escalation signalEscalation[C], action authoringSignalAction) SignalActionPlan[C] {
	plan := signalCompletePlan[C]{
		state: appendSignalAction(escalation.state, action),
	}
	return plan
}

func eraseSignalHandler[C any](handler SignalHandler[C]) authoringSignalAction {
	valid := handler.Name != "" && !containsForbiddenDeclarationText(handler.Name) && handler.Handle != nil
	var callback signalHandlerCallback
	if handler.Handle != nil {
		typed := handler.Handle
		callback = func(ctx context.Context, rawConfig, rawEvent any) error {
			config, ok := rawConfig.(*C)
			event, eventOK := rawEvent.(SignalEvent)
			if !ok || config == nil || !eventOK {
				panic("cli: invalid signal callback input")
			}
			return typed(ctx, config, event)
		}
	}
	action := authoringSignalAction{
		kind:    signalActionKindHandle,
		name:    handler.Name,
		handler: callback,
		valid:   valid,
	}

	return action
}

func eraseSignalTarget[C any](target SignalTarget[C]) authoringSignalTarget {
	result := authoringSignalTarget{
		name:      target.Name,
		validName: target.Name != "" && !containsForbiddenDeclarationText(target.Name),
	}
	if target.Deliver != nil {
		typed := target.Deliver
		result.deliver = func(ctx context.Context, rawConfig, rawDelivery any) error {
			config, ok := rawConfig.(*C)
			delivery, deliveryOK := rawDelivery.(SignalDelivery)
			if !ok || config == nil || !deliveryOK {
				panic("cli: invalid signal delivery callback input")
			}
			return typed(ctx, config, delivery)
		}
	}
	if target.MatchesSender != nil {
		typed := target.MatchesSender
		result.matchesSender = func(ctx context.Context, rawConfig any, sender SignalSender) (bool, error) {
			config, ok := rawConfig.(*C)
			if !ok || config == nil {
				panic("cli: invalid signal sender callback configuration")
			}
			signalSenderValue := SignalSender{
				ProcessID: sender.ProcessID,
				UserID:    sender.UserID,
				Code:      sender.Code,
			}
			return typed(ctx, config, signalSenderValue)
		}
	}
	return result
}

func signalTargetAction[C any](kind signalActionKind, target SignalTarget[C], signal Signal, destination *destinationHandle) authoringSignalAction {
	erased := eraseSignalTarget(target)
	valid := erased.validName && erased.deliver != nil
	if kind == signalActionKindSend {
		valid = valid && (signal.valid() || destination != nil) &&
			(!signal.valid() || destination == nil)
	}
	action := authoringSignalAction{
		kind:        kind,
		name:        erased.name,
		target:      erased,
		signal:      signal,
		signalValue: destination,
		valid:       valid,
	}

	return action
}

func signalActionPlanStateOf[C any](plan SignalActionPlan[C]) *signalActionPlanState {
	switch plan := plan.(type) {
	case signalCompletePlan[C]:
		return cloneSignalActionPlan(plan.state)
	case signalSequence[C]:
		return cloneSignalActionPlan(plan.state)
	default:
		return nil
	}
}

func signalDestination(owner *draftIdentity, destination *destinationHandle) *destinationHandle {
	if owner == nil || destination == nil {
		return nil
	}
	identity, ok := destination.currentIdentity().(stagedDestinationIdentity)
	if !ok || !identity.ref.ownedBy(owner) {
		return nil
	}
	return destination
}

func signalActionPlanDestination(state *signalActionPlanState, destination *destinationHandle) *destinationHandle {
	if state == nil {
		return nil
	}
	return signalDestination(state.owner, destination)
}

func signalEndpoint(owner *draftIdentity, endpoint signalEndpointDraftRef) signalEndpointDraftRef {
	if owner == nil || !endpoint.ownedBy(owner) {
		return signalEndpointDraftRef{}
	}
	return endpoint
}

func signalEndpointOverlapsDestination(draft *declarationDraft, binding configurationBinding) bool {
	if draft == nil || !binding.Valid() {
		return false
	}
	for _, entry := range draft.destinations.entries {
		if configurationBindingsOverlap(binding, entry.value.binding) {
			return true
		}
	}
	return false
}

func configurationBindingsOverlap(left, right configurationBinding) bool {
	leftOffset, leftSize, leftOK := left.StorageLocation()
	rightOffset, rightSize, rightOK := right.StorageLocation()
	if !leftOK || !rightOK {
		return false
	}
	if leftSize == 0 || rightSize == 0 {
		return leftOffset == rightOffset
	}
	return leftOffset < rightOffset+rightSize && rightOffset < leftOffset+leftSize
}

func signalCancelAction() authoringSignalAction {
	action := authoringSignalAction{
		kind:  signalActionKindCancel,
		valid: true,
	}
	return action
}

func signalAbortAction(code int) authoringSignalAction {
	action := authoringSignalAction{
		kind:     signalActionKindAbort,
		exitCode: code,
		valid:    true,
	}

	return action
}

func signalReraiseAction() authoringSignalAction {
	action := authoringSignalAction{
		kind:  signalActionKindReraise,
		valid: true,
	}
	return action
}

func signalForwardAction[C any](target SignalTarget[C]) authoringSignalAction {
	return signalTargetAction(signalActionKindForward, target, Signal{}, nil)
}

func signalSendAction[C any](target SignalTarget[C], signal Signal) authoringSignalAction {
	return signalTargetAction(signalActionKindSend, target, signal, nil)
}

func signalSendFromAction[C any](state *signalActionPlanState, target SignalTarget[C], source Destination[C, Signal]) authoringSignalAction {
	var owner *draftIdentity
	if state != nil {
		owner = state.owner
	}
	return signalTargetAction(
		signalActionKindSend,
		target,
		Signal{},
		signalDestination(owner, source.handle),
	)
}

func signalEmitAction[C any](state *signalActionPlanState, endpoint SignalEndpointDeclaration[C]) authoringSignalAction {
	var owner *draftIdentity
	if state != nil {
		owner = state.owner
	}
	endpointRef := signalEndpoint(owner, endpoint.endpoint)
	action := authoringSignalAction{
		kind:     signalActionKindEmit,
		endpoint: endpointRef,
		valid:    endpointRef.valid(),
	}

	return action
}

// signalPlan is the pure operation-effective signal state shared by Inspection
// and Run. Building it evaluates only already-finalized semantic inputs and
// never acquires a SignalSource or creates live runtime resources.
type signalPlan struct {
	policies        []signalPlanPolicy
	requiredSignals []Signal
	requiredSource  SignalSourceCapabilities
	requiredHost    SignalHostCapabilities
}

type signalPlanPolicy struct {
	record     signalPolicyRecord
	definition signalPolicyDefinition
	activation conditionOutcome
	active     bool
}

// resolvedSignalPlan is the source-description-dependent frozen signal plan.
// Selector classes are interpreted exactly once here. Inspection projects this
// state and Run consumes it verbatim; live execution never reinterprets a
// selector class.
type resolvedSignalPlan struct {
	base            signalPlan
	policies        []resolvedSignalPlanPolicy
	resolvedSignals []Signal
	watchByPhase    map[SignalPhase][]Signal
	endpoints       []SignalEndpointID
	compatibility   signalPlanCompatibility
}

type resolvedSignalPlanPolicy struct {
	signalPlanPolicy

	resolvedSignals []Signal
}

type signalPlanCompatibility struct {
	missingSignals            []Signal
	missingSourceCapabilities SignalSourceCapabilities
	availableHostCapabilities SignalHostCapabilities
	missingHostCapabilities   SignalHostCapabilities
}

func (compatibility signalPlanCompatibility) compatible() bool {
	return len(compatibility.missingSignals) == 0 &&
		compatibility.missingSourceCapabilities == 0 &&
		compatibility.missingHostCapabilities == 0
}

func buildSignalPlan(model *effectiveModel, selection commandSelection, selectedProfiles map[CommandID]GrammarProfileID, values valueAnalysis) (signalPlan, bool) {
	effective, ok := effectiveSignalPolicies(model, selection, selectedProfiles)
	if !ok {
		return signalPlan{}, false
	}
	plan := signalPlan{
		policies: make([]signalPlanPolicy, 0, len(effective)),
	}
	seenSignals := make(map[Signal]struct{})
	for _, record := range effective {
		definition, ok := record.Definition()
		if !ok || !definition.Valid() {
			return signalPlan{}, false
		}
		item := signalPlanPolicy{
			record:     record,
			definition: definition,
			activation: conditionOutcomeSatisfied,
			active:     true,
		}
		if activation, present := definition.Activation(); present {
			item.activation = evaluateCondition(activation, values)
			switch item.activation {
			case conditionOutcomeSatisfied:
				item.active = true
			case conditionOutcomeUnsatisfied:
				item.active = false
			case conditionOutcomeBlocked:
				plan.policies = append(plan.policies, item)
				return plan, false
			default:
				return signalPlan{}, false
			}
		}
		plan.policies = append(plan.policies, item)
		if !item.active {
			continue
		}

		plan.requiredSource |= definition.RequiredSourceCapabilities()
		plan.requiredHost |= definition.RequiredHostCapabilities()
		// Listed membership is already concrete without a source description and
		// is retained for useful missing-source diagnostics. Class selectors are
		// frozen later against the source description.
		if definition.Selector().Kind() == signalSelectorKindListed {
			for _, signal := range effectiveListedSignals(definition.Selector()) {
				if _, duplicate := seenSignals[signal]; duplicate {
					continue
				}
				seenSignals[signal] = struct{}{}
				plan.requiredSignals = append(plan.requiredSignals, signal)
			}
		}
	}
	return plan, true
}

func validateCompiledSignalSourcePolicies(model staticProgram, description semanticSignalSourceDescription) error {
	if !model.Valid() || !description.Valid() {
		return errInvalidSignalSourceDeclaration
	}
	for _, record := range model.SignalPolicies() {
		definition, ok := record.Definition()
		if !ok || !definition.Valid() {
			return errInvalidSignalSourceDeclaration
		}
		if definition.RequiredSourceCapabilities()&SignalSourceCapabilityReraise == 0 {
			continue
		}
		if err := validateReraiseSelector(description, definition.Selector()); err != nil {
			return err
		}
	}
	return nil
}

func validateProviderSignalSourcePolicies(declarations batchDeclarations, description semanticSignalSourceDescription) error {
	if !description.Valid() {
		return errInvalidSignalSourceDeclaration
	}
	for _, record := range declarations.order {
		if record.kind != declarationRecordSignalPolicy {
			continue
		}
		if record.familyIndex < 0 || record.familyIndex >= len(declarations.signalPolicies) {
			return errInvalidSignalSourceDeclaration
		}
		policyValue := batchSignalPolicyAcceptanceInput(&declarations, declarations.signalPolicies[record.familyIndex])
		if policyValue.match == nil || policyValue.plan == nil {
			return errInvalidSignalSourceDeclaration
		}
		reraise := false
		for _, action := range policyValue.plan.actions {
			if action.kind == signalActionKindReraise {
				reraise = true
				break
			}
		}
		if !reraise {
			continue
		}
		phases, ok := prepareSignalPhases(policyValue.match.phases)
		if !ok {
			return errInvalidSignalSourceDeclaration
		}
		origins, ok := prepareSignalOrigins(policyValue.match.origins)
		if !ok {
			return errInvalidSignalSourceDeclaration
		}
		excludedOrigins, ok := prepareSignalOrigins(policyValue.match.excludedOrigins)
		if !ok {
			return errInvalidSignalSourceDeclaration
		}
		input := signalSelectorInput{
			Kind:            policyValue.match.kind,
			Signals:         semanticSignalIDs(policyValue.match.signals),
			ExcludedSignals: semanticSignalIDs(policyValue.match.excludedSignals),
			Origins:         origins,
			ExcludedOrigins: excludedOrigins,
			ExcludedSender:  policyValue.match.excludedSender.name,
			Phases:          phases,
		}
		selector := acceptSignalSelectorDefinition(input)
		if !selector.Valid() {
			return errInvalidSignalSourceDeclaration
		}
		if err := validateReraiseSelector(description, selector); err != nil {
			return err
		}
	}
	return nil
}

func validateReraiseSelector(description semanticSignalSourceDescription, selector signalSelectorDefinition) error {
	resolved := resolveSignalSelector(description, selector)
	requirement, err := acceptSignalRequirement(
		resolved,
		SignalSourceCapabilityReraise,
		0,
	)
	if err != nil {
		return errInvalidSignalSourceDeclaration
	}
	compatibility, err := evaluateSignalCompatibility(description, requirement, 0)
	if err != nil || len(compatibility.MissingSignals()) != 0 ||
		compatibility.MissingSourceCapabilities() != 0 {
		return errInvalidSignalSourceDeclaration
	}
	return nil
}

func resolveSignalPlan(description semanticSignalSourceDescription, plan signalPlan, availableHost SignalHostCapabilities) (resolvedSignalPlan, error) {
	resolved := resolvedSignalPlan{
		base:         plan,
		policies:     make([]resolvedSignalPlanPolicy, 0, len(plan.policies)),
		watchByPhase: make(map[SignalPhase][]Signal),
		compatibility: signalPlanCompatibility{
			availableHostCapabilities: availableHost,
		},
	}
	seenAll := make(map[Signal]struct{})
	seenEndpoint := make(map[SignalEndpointID]struct{})
	seenByPhase := make(map[SignalPhase]map[Signal]struct{})

	for _, policy := range plan.policies {
		item := resolvedSignalPlanPolicy{
			signalPlanPolicy: policy,
		}
		if !policy.active {
			resolved.policies = append(resolved.policies, item)
			continue
		}

		item.resolvedSignals = resolveSignalSelector(description, policy.definition.Selector())
		requirement, err := acceptSignalRequirement(
			item.resolvedSignals,
			policy.definition.RequiredSourceCapabilities(),
			policy.definition.RequiredHostCapabilities(),
		)
		if err != nil {
			return resolvedSignalPlan{}, err
		}
		compatibility, err := evaluateSignalCompatibility(description, requirement, availableHost)
		if err != nil {
			return resolvedSignalPlan{}, err
		}
		appendUniqueSignals(&resolved.compatibility.missingSignals, compatibility.MissingSignals())
		resolved.compatibility.missingSourceCapabilities |= compatibility.MissingSourceCapabilities()
		resolved.compatibility.missingHostCapabilities |= compatibility.MissingHostCapabilities()

		for _, signal := range compatibility.ResolvedSignals() {
			if _, duplicate := seenAll[signal]; !duplicate {
				seenAll[signal] = struct{}{}
				resolved.resolvedSignals = append(resolved.resolvedSignals, signal)
			}
			for _, phase := range policy.definition.Selector().Phases() {
				if seenByPhase[phase] == nil {
					seenByPhase[phase] = make(map[Signal]struct{})
				}
				if _, duplicate := seenByPhase[phase][signal]; duplicate {
					continue
				}
				seenByPhase[phase][signal] = struct{}{}
				resolved.watchByPhase[phase] = append(resolved.watchByPhase[phase], signal)
			}
		}
		for _, action := range policy.definition.Actions() {
			endpoint, present := action.Endpoint()
			if !present {
				continue
			}
			if _, duplicate := seenEndpoint[endpoint]; duplicate {
				continue
			}
			seenEndpoint[endpoint] = struct{}{}
			resolved.endpoints = append(resolved.endpoints, endpoint)
		}
		resolved.policies = append(resolved.policies, item)
	}
	return resolved, nil
}

func resolveSignalSelector(description semanticSignalSourceDescription, selector signalSelectorDefinition) []Signal {
	excluded := make(map[Signal]struct{}, len(selector.ExcludedSignals()))
	for _, signal := range selector.ExcludedSignals() {
		excluded[signal] = struct{}{}
	}

	var candidates []Signal
	switch selector.Kind() {
	case signalSelectorKindListed:
		candidates = selector.Signals()
	case signalSelectorKindInterrupt, signalSelectorKindTerminate,
		signalSelectorKindAsynchronous, signalSelectorKindShutdown:
		for _, descriptor := range description.Descriptors() {
			selected := false
			switch selector.Kind() {
			case signalSelectorKindInterrupt:
				selected = descriptor.Classes()&SignalClassInterrupt != 0
			case signalSelectorKindTerminate:
				selected = descriptor.Classes()&SignalClassTerminate != 0
			case signalSelectorKindAsynchronous:
				selected = !descriptor.Reserved()
			case signalSelectorKindShutdown:
				selected = descriptor.Reserved()
			case signalSelectorKindUnknown, signalSelectorKindListed:
			}
			if selected {
				candidates = append(candidates, descriptor.Signal())
			}
		}
	case signalSelectorKindUnknown:
	}

	result := make([]Signal, 0, len(candidates))
	for _, signal := range candidates {
		if _, skip := excluded[signal]; skip {
			continue
		}
		result = append(result, signal)
	}
	return result
}

func appendUniqueSignals(dst *[]Signal, values []Signal) {
	seen := make(map[Signal]struct{}, len(*dst)+len(values))
	for _, signal := range *dst {
		seen[signal] = struct{}{}
	}
	for _, signal := range values {
		if _, duplicate := seen[signal]; duplicate {
			continue
		}
		seen[signal] = struct{}{}
		*dst = append(*dst, signal)
	}
}

func (plan signalPlan) activePolicyIDs() []SignalPolicyID {
	var result []SignalPolicyID
	for _, policy := range plan.policies {
		if policy.active {
			result = append(result, policy.record.meta.id)
		}
	}
	return result
}

func (plan resolvedSignalPlan) activePolicyIDs(phase SignalPhase) []SignalPolicyID {
	var result []SignalPolicyID
	for _, policy := range plan.policies {
		if !policy.active || !slices.Contains(policy.definition.Selector().Phases(), phase) {
			continue
		}
		result = append(result, policy.record.meta.id)
	}
	return result
}

func (plan resolvedSignalPlan) watch(phase SignalPhase) []Signal {
	return append([]Signal(nil), plan.watchByPhase[phase]...)
}
