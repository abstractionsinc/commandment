package cli

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"sync"
	"time"
)

var (
	errSignalEndpointInvalidCapacity = errors.New("signal endpoint capacity must be positive")
	errSignalEndpointClosed          = errors.New("signal endpoint is closed")
	errSignalEndpointInvalidBinding  = errors.New("signal endpoint binding is invalid")
	errSignalEndpointDuplicate       = errors.New("signal endpoint binding is duplicated")
	errSignalEndpointState           = errors.New("signal endpoint preparation is in an invalid state")
)

// signalEndpointOverflowError reports private endpoint pressure. The caller adds
// policy/action/event context at the dispatch boundary.
type signalEndpointOverflowError struct {
	Capacity int
	Buffered int
}

func (overflow *signalEndpointOverflowError) Error() string {
	if overflow == nil {
		return "signal endpoint overflow"
	}
	return fmt.Sprintf(
		"signal endpoint overflow: capacity %d, buffered %d",
		overflow.Capacity,
		overflow.Buffered,
	)
}

// runSignalEndpoint is a bounded frontend-owned producer. Delivery never waits
// for application backlog capacity; overflow is explicit. Close and Deliver
// are synchronized so shutdown cannot race a send into a closed channel.
type runSignalEndpoint[T any] struct {
	mu sync.RWMutex

	channel chan T
	closed  bool
}

func openSignalEndpoint[T any](capacity int) (*runSignalEndpoint[T], error) {
	if capacity <= 0 {
		return nil, errSignalEndpointInvalidCapacity
	}
	return &runSignalEndpoint[T]{
		channel: make(chan T, capacity),
	}, nil
}

func (endpoint *runSignalEndpoint[T]) Receive() <-chan T {
	if endpoint == nil {
		return nil
	}
	endpoint.mu.RLock()
	defer endpoint.mu.RUnlock()
	return endpoint.channel
}

func (endpoint *runSignalEndpoint[T]) Deliver(value T) error {
	if endpoint == nil {
		return errSignalEndpointClosed
	}

	endpoint.mu.RLock()
	defer endpoint.mu.RUnlock()
	if endpoint.closed {
		return errSignalEndpointClosed
	}
	select {
	case endpoint.channel <- value:
		return nil
	default:
		return &signalEndpointOverflowError{
			Capacity: cap(endpoint.channel),
			Buffered: len(endpoint.channel),
		}
	}
}

func (endpoint *runSignalEndpoint[T]) Close() {
	if endpoint == nil {
		return
	}
	endpoint.mu.Lock()
	defer endpoint.mu.Unlock()
	if endpoint.closed {
		return
	}
	endpoint.closed = true
	close(endpoint.channel)
}

// signalEndpointBinding is an execution-plan adapter for one deduplicated
// endpoint destination. Install runs only while configuration is frontend-
// private. Restore is used only if final watch publication fails. Close owns
// the private producer/resource and must not depend on the application field
// still containing the installed endpoint value.
type signalEndpointBinding[C any] struct {
	ID      SignalEndpointID
	Install func(*C)
	Restore func(*C)
	Close   func()
}

func (binding signalEndpointBinding[C]) valid() bool {
	return binding.ID.present() && binding.Install != nil && binding.Restore != nil && binding.Close != nil
}

type signalEndpointPreparationState uint8

const (
	signalEndpointPreparationInvalid signalEndpointPreparationState = iota
	signalEndpointPreparationPrepared
	signalEndpointPreparationTransitioning
	signalEndpointPreparationCommitted
	signalEndpointPreparationRolledBack
	signalEndpointPreparationClosed
)

// signalEndpointPreparation owns one complete endpoint set between private
// configuration preparation and execution handoff.
type signalEndpointPreparation[C any] struct {
	mu sync.Mutex

	config    *C
	bindings  []signalEndpointBinding[C]
	state     signalEndpointPreparationState
	closeOnce sync.Once
	closeDone chan struct{}
}

func prepareSignalEndpoints[C any](config *C, bindings []signalEndpointBinding[C]) (*signalEndpointPreparation[C], error) {
	if config == nil {
		return nil, errSignalEndpointInvalidBinding
	}

	copied := append([]signalEndpointBinding[C](nil), bindings...)
	seen := make(map[SignalEndpointID]struct{}, len(copied))
	for _, binding := range copied {
		if !binding.valid() {
			return nil, errSignalEndpointInvalidBinding
		}
		if _, duplicate := seen[binding.ID]; duplicate {
			return nil, errSignalEndpointDuplicate
		}
		seen[binding.ID] = struct{}{}
	}

	for _, binding := range copied {
		binding.Install(config)
	}
	result := &signalEndpointPreparation[C]{
		config:    config,
		bindings:  copied,
		state:     signalEndpointPreparationPrepared,
		closeDone: make(chan struct{}),
	}
	return result, nil
}

func (preparation *signalEndpointPreparation[C]) Commit() error {
	if preparation == nil {
		return errSignalEndpointState
	}
	preparation.mu.Lock()
	defer preparation.mu.Unlock()
	if preparation.state != signalEndpointPreparationPrepared {
		return errSignalEndpointState
	}
	preparation.state = signalEndpointPreparationCommitted
	return nil
}

func (preparation *signalEndpointPreparation[C]) beginTransition() error {
	if preparation == nil {
		return errSignalEndpointState
	}
	preparation.mu.Lock()
	defer preparation.mu.Unlock()
	if preparation.state != signalEndpointPreparationPrepared {
		return errSignalEndpointState
	}
	preparation.state = signalEndpointPreparationTransitioning
	return nil
}

func (preparation *signalEndpointPreparation[C]) commitTransition() {
	preparation.mu.Lock()
	defer preparation.mu.Unlock()
	if preparation.state != signalEndpointPreparationTransitioning {
		panic("run: signal endpoint preparation lost transition ownership")
	}
	preparation.state = signalEndpointPreparationCommitted
}

func (preparation *signalEndpointPreparation[C]) rollbackTransition() {
	preparation.mu.Lock()
	if preparation.state != signalEndpointPreparationTransitioning {
		preparation.mu.Unlock()
		panic("run: signal endpoint preparation lost transition ownership")
	}
	preparation.state = signalEndpointPreparationRolledBack
	config := preparation.config
	bindings := append([]signalEndpointBinding[C](nil), preparation.bindings...)
	preparation.mu.Unlock()

	for _, binding := range slices.Backward(bindings) {
		binding.Restore(config)
	}
	preparation.closeBindings(bindings)
}

// Rollback restores private configuration and closes all prepared producers.
// It is legal only before execution handoff.
func (preparation *signalEndpointPreparation[C]) Rollback() error {
	if preparation == nil {
		return errSignalEndpointState
	}
	preparation.mu.Lock()
	if preparation.state != signalEndpointPreparationPrepared {
		preparation.mu.Unlock()
		return errSignalEndpointState
	}
	preparation.state = signalEndpointPreparationRolledBack
	config := preparation.config
	bindings := append([]signalEndpointBinding[C](nil), preparation.bindings...)
	preparation.mu.Unlock()

	for _, binding := range slices.Backward(bindings) {
		binding.Restore(config)
	}
	preparation.closeBindings(bindings)
	return nil
}

// Close closes committed endpoint producers exactly once without restoring
// application-owned configuration fields after handoff.
func (preparation *signalEndpointPreparation[C]) Close() error {
	if preparation == nil {
		return nil
	}
	preparation.mu.Lock()
	switch preparation.state {
	case signalEndpointPreparationCommitted:
		preparation.state = signalEndpointPreparationClosed
		bindings := append([]signalEndpointBinding[C](nil), preparation.bindings...)
		preparation.mu.Unlock()
		preparation.closeBindings(bindings)
		return nil
	case signalEndpointPreparationClosed, signalEndpointPreparationRolledBack:
		bindings := append([]signalEndpointBinding[C](nil), preparation.bindings...)
		preparation.mu.Unlock()
		preparation.closeBindings(bindings)
		return nil
	default:
		preparation.mu.Unlock()
		return errSignalEndpointState
	}
}

func (preparation *signalEndpointPreparation[C]) closeBindings(bindings []signalEndpointBinding[C]) {
	preparation.closeOnce.Do(func() {
		for _, binding := range bindings {
			binding.Close()
		}
		close(preparation.closeDone)
	})
	<-preparation.closeDone
}

// commitSignalExecutionWatch performs the final endpoint/watch atomicity seam:
// endpoints are already installed while config is private; failed watch
// publication restores/closes them; successful publication commits them before
// the caller resumes normal dispatch or hands configuration to application code.
func commitSignalExecutionWatch[C any](ctx context.Context, watch *signalWatch, watched []Signal, acceptance signalAcceptanceSnapshot, endpoints *signalEndpointPreparation[C]) (signalWatchState, error) {
	if watch == nil || endpoints == nil {
		return signalWatchState{}, errSignalEndpointState
	}
	if err := endpoints.beginTransition(); err != nil {
		return signalWatchState{}, err
	}
	state, err := watch.Transition(ctx, watched, acceptance)
	if err != nil {
		endpoints.rollbackTransition()
		return state, err
	}
	endpoints.commitTransition()
	return state, nil
}

type runSignals struct {
	runtime *signalRuntime[struct{}]
	watch   *signalWatch
	plan    resolvedSignalPlan
	model   *effectiveModel
	values  valueAnalysis
	sources []compiledSignalSource

	cancelPrimary context.CancelCauseFunc

	endpointPlaceholder struct{}
	endpointProducers   map[SignalEndpointID]*runSignalEndpoint[SignalEvent]

	callbackSerial     sync.Mutex
	mu                 sync.Mutex
	cancellation       error
	primaryCancelCause error
	runtimeFailures    []error
	phase              SignalPhase
	cleanupStarted     bool
	closed             bool
}

type signalDeferralContextKey struct{}

type runSignalPreparation struct {
	state     *runSignals
	supported bool
}

type preparedSignalExecution struct {
	state   *runSignals
	release func()
}

func (prepared *preparedSignalExecution) commit() {
	if prepared == nil || prepared.state == nil || prepared.release == nil {
		panic("cli: invalid prepared Signal Execution transition")
	}
	prepared.state.mu.Lock()
	if prepared.state.closed || prepared.state.phase != SignalPhaseExecution {
		prepared.state.mu.Unlock()
		panic("cli: Signal Execution actions released before Execution phase")
	}
	prepared.state.mu.Unlock()

	release := prepared.release
	prepared.release = nil
	prepared.state = nil
	release()
}

func (state *runSignals) enterExecutionPhase() {
	if state == nil {
		panic("cli: nil Run Signal state")
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.closed || state.phase != SignalPhaseResolution {
		panic("cli: invalid Signal Execution phase transition")
	}
	state.phase = SignalPhaseExecution
}

func prepareRunSignals(ctx context.Context, snapshot *runState, sources []compiledSignalSource, model *effectiveModel, selection commandSelection, selectedProfiles map[CommandID]GrammarProfileID, values valueAnalysis) (runSignalPreparation, context.Context, context.Context, error) {
	base, ok := buildSignalPlan(model, selection, selectedProfiles, values)
	if !ok {
		return runSignalPreparation{}, nil, nil, nil
	}
	if len(base.activePolicyIDs()) == 0 {
		return runSignalPreparation{supported: true}, nil, nil, nil
	}
	description := snapshot.invocation.signalDescription
	resolved, err := resolveSignalPlan(description, base, availableSignalHostCapabilities(snapshot))
	if err != nil {
		return runSignalPreparation{supported: true}, nil, nil, signalRunAdmissionFailure(snapshot.invocation, DiagnosticSignalSourceFailure, err)
	}
	if !resolved.compatibility.compatible() {
		return runSignalPreparation{supported: true}, nil, nil, signalRunAdmissionFailure(snapshot.invocation, DiagnosticIncompatibleSignalSource, nil)
	}

	acceptance, err := acceptSignalAcceptanceSnapshot(
		SignalPhaseResolution,
		resolved.activePolicyIDs(SignalPhaseResolution),
	)
	if err != nil {
		return runSignalPreparation{}, nil, nil, err
	}
	watch, err := prepareSignalWatch(
		ctx,
		resolved.watch(SignalPhaseResolution),
		acceptance,
		adaptSignalSources(sources),
	)
	if err != nil {
		return runSignalPreparation{supported: true}, nil, nil, signalRunAdmissionFailure(snapshot.invocation, DiagnosticSignalSourceFailure, err)
	}
	intake, err := prepareSignalIntake(watch, SignalBacklogCapacity)
	if err != nil {
		_ = watch.Close()
		return runSignalPreparation{}, nil, nil, err
	}
	primaryCtx, cancelPrimary := context.WithCancelCause(ctx)
	state := &runSignals{
		watch:             watch,
		plan:              resolved,
		model:             model,
		values:            values,
		sources:           append([]compiledSignalSource(nil), sources...),
		cancelPrimary:     cancelPrimary,
		endpointProducers: make(map[SignalEndpointID]*runSignalEndpoint[SignalEvent]),
		phase:             SignalPhaseResolution,
	}
	policies := state.buildSignalDispatchPolicies(snapshot)
	signalCallbackParent := context.WithValue(ctx, signalDeferralContextKey{}, state)
	dispatcher, signalCallbackCtx, err := prepareSignalDispatcher(
		signalCallbackParent,
		&state.endpointPlaceholder,
		policies,
		state.cancelPrimaryFromRuntime,
	)
	if err != nil {
		cancelPrimary(err)
		_ = watch.Close()
		return runSignalPreparation{}, nil, nil, err
	}
	runtime, err := prepareSignalRuntime(watch, intake, dispatcher, policies, nil, state.cancelPrimaryFromRuntime)
	if err != nil {
		cancelPrimary(err)
		dispatcher.Close(err)
		_ = watch.Close()
		return runSignalPreparation{}, nil, nil, err
	}
	state.runtime = runtime
	return runSignalPreparation{
		state:     state,
		supported: true,
	}, primaryCtx, signalCallbackCtx, nil
}

func (state *runSignals) buildSignalDispatchPolicies(snapshot *runState) []signalDispatchPolicy[struct{}] {
	result := make([]signalDispatchPolicy[struct{}], 0, len(state.plan.policies))
	for _, planned := range state.plan.policies {
		if !planned.active {
			continue
		}
		configuration, ok := snapshot.configurations.configuration(planned.record.command)
		if !ok || configuration == nil {
			panic("cli: accepted Signal policy has no execution configuration")
		}
		counter, ok := planned.record.Counter()
		if !ok || !counter.present() {
			panic("cli: accepted Signal policy has no counter")
		}
		definition := planned.definition
		occurrence := definition.Occurrence()
		aggregation := definition.Aggregation()
		if !occurrence.Valid() || !aggregation.valid() {
			panic("cli: accepted Signal policy has invalid occurrence or aggregation")
		}
		minimum := occurrence.Minimum()
		maximum := occurrence.Maximum()
		if minimum < 0 || maximum < 0 {
			panic("cli: accepted Signal policy has invalid occurrence bounds")
		}
		minimumOccurrence := uint64(minimum)
		maximumOccurrence := uint64(maximum)
		resolvedSignals := append([]Signal(nil), planned.resolvedSignals...)
		policy := signalDispatchPolicy[struct{}]{
			ID:      planned.record.meta.id,
			Counter: counter,
			Match: func(callbackCtx context.Context, event signalAcceptedNotification) (bool, error) {
				return state.signalDispatchMatches(callbackCtx, definition.Selector(), resolvedSignals, configuration, event)
			},
			Occurrence: func(count uint64) bool {
				if count < minimumOccurrence {
					return false
				}
				return maximumOccurrence == 0 || count <= maximumOccurrence
			},
			Aggregation:      aggregation,
			Deferred:         definition.Deferred(),
			ContinueMatching: definition.ContinueMatching(),
		}
		actions := definition.Actions()
		for index, action := range actions {
			if !action.Delayed() {
				adapted, ok := state.buildSignalDispatchAction(snapshot, action, configuration)
				if !ok {
					panic("cli: accepted Signal action cannot bind for Run")
				}
				policy.Immediate = append(policy.Immediate, adapted)
				continue
			}
			if policy.Escalation != nil || index != len(actions)-1 {
				panic("cli: accepted delayed Signal action is not terminal")
			}
			delay, ok := state.resolveSignalActionDelay(action)
			if !ok {
				panic("cli: accepted delayed Signal action has unresolved delay")
			}
			var fixedSignal Signal
			if _, present := action.SignalValue(); present {
				fixedSignal, ok = state.resolveSignalActionSignal(action)
				if !ok {
					panic("cli: accepted delayed Signal action has unresolved signal")
				}
			}
			adapted, ok := state.buildResolvedSignalDispatchAction(snapshot, action, configuration, fixedSignal)
			if !ok {
				panic("cli: accepted delayed Signal action cannot bind for Run")
			}
			policy.Escalation = &signalDispatchEscalation[struct{}]{
				Action:      adapted,
				Delay:       delay,
				ActionIndex: index,
			}
		}
		result = append(result, policy)
	}
	return result
}

func (state *runSignals) signalDispatchMatches(ctx context.Context, selector signalSelectorDefinition, resolvedSignals []Signal, configuration any, event signalAcceptedNotification) (bool, error) {
	notification := event.Notification()
	listed := false
	for _, candidate := range resolvedSignals {
		if candidate == notification.Signal() {
			listed = true
			break
		}
	}
	if !listed {
		return false, nil
	}
	if origins := selector.Origins(); len(origins) != 0 {
		matched := false
		for _, origin := range origins {
			if origin == notification.Origin() {
				matched = true
				break
			}
		}
		if !matched {
			return false, nil
		}
	}
	for _, excluded := range selector.ExcludedOrigins() {
		if excluded == notification.Origin() {
			return false, nil
		}
	}
	if selector.ExcludedSender() != "" && notification.Sender() != nil {
		target, ok := selector.ExcludedSenderTarget()
		if !ok || target.MatchesSender() == nil {
			panic("cli: accepted Signal sender exclusion has no matcher")
		}
		state.callbackSerial.Lock()
		matched, err := target.MatchesSender()(ctx, configuration, *notification.Sender())
		state.callbackSerial.Unlock()
		if cause := context.Cause(ctx); cause != nil {
			err = suppressConsequentialCancellation(err, cause, cause)
			return false, err
		}
		if err != nil {
			return false, err
		}
		if matched {
			return false, nil
		}
	}
	return true, nil
}

func (state *runSignals) buildSignalDispatchAction(snapshot *runState, action signalActionDefinition, configuration any) (signalDispatchAction[struct{}], bool) {
	return state.buildResolvedSignalDispatchAction(snapshot, action, configuration, Signal{})
}

func (state *runSignals) buildResolvedSignalDispatchAction(snapshot *runState, action signalActionDefinition, configuration any, fixedSignal Signal) (signalDispatchAction[struct{}], bool) {
	handler := func(ctx context.Context, _ *struct{}, batch signalDeliveryBatch) error {
		event, ok := projectSignalDeliveryEvent(batch)
		if !ok {
			panic("cli: accepted Signal delivery batch cannot project an event")
		}
		state.callbackSerial.Lock()
		defer state.callbackSerial.Unlock()
		switch action.Kind() {
		case signalActionKindIgnore:
			return nil
		case signalActionKindHandle:
			return action.Handler()(ctx, configuration, event)
		case signalActionKindForward:
			return action.Target().Deliver()(ctx, configuration, SignalDelivery{
				Event:  event,
				Signal: event.Signal,
			})
		case signalActionKindSend:
			signal := fixedSignal
			if !signal.valid() {
				var ok bool
				signal, ok = state.resolveSignalActionSignal(action)
				if !ok {
					panic("cli: accepted Signal send action has unresolved signal")
				}
			}
			return action.Target().Deliver()(ctx, configuration, SignalDelivery{
				Event:  event,
				Signal: signal,
			})
		case signalActionKindEmit:
			endpointID, present := action.Endpoint()
			endpoint := state.endpointProducers[endpointID]
			if !present || endpoint == nil {
				panic("cli: accepted Signal emit action has no prepared endpoint")
			}
			return endpoint.Deliver(event)
		default:
			panic("cli: accepted Signal action has invalid kind")
		}
	}

	switch action.Kind() {
	case signalActionKindCancel:
		return signalCancelDispatchAction[struct{}](), true
	case signalActionKindAbort:
		if snapshot.abort == nil {
			return signalDispatchAction[struct{}]{}, false
		}
		return signalBatchAbortAction(func(context.Context, *struct{}, signalDeliveryBatch) error {
			state.callbackSerial.Lock()
			defer state.callbackSerial.Unlock()
			return snapshot.abort(action.ExitCode())
		}), true
	case signalActionKindReraise:
		return signalBatchReraiseAction(
			func(ctx context.Context, _ *struct{}, batch signalDeliveryBatch) error {
				state.callbackSerial.Lock()
				defer state.callbackSerial.Unlock()
				event, ok := projectSignalDeliveryEvent(batch)
				if !ok {
					panic("cli: accepted Signal reraise batch cannot project an event")
				}
				source, ok := sourceForSignal(state.sources, event.Signal)
				if !ok || source.reraise == nil {
					panic("cli: accepted Signal reraise action has no source capability")
				}
				return source.reraise(ctx, event.Signal)
			},
		), true
	case signalActionKindIgnore, signalActionKindHandle, signalActionKindForward,
		signalActionKindSend, signalActionKindEmit:
		return signalBatchCallbackAction(handler), true
	default:
		return signalDispatchAction[struct{}]{}, false
	}
}

func (state *runSignals) resolveSignalActionSignal(action signalActionDefinition) (Signal, bool) {
	if destination, present := action.SignalValue(); present {
		resolution, ok := state.values.Destinations[destination]
		if !ok || resolution.State != destinationStateResolved {
			return Signal{}, false
		}
		signal, ok := resolution.Value.(Signal)
		return signal, ok && signal.valid()
	}
	signal := action.Signal()
	return signal, signal.valid()
}

func (state *runSignals) resolveSignalActionDelay(action signalActionDefinition) (time.Duration, bool) {
	if destination, present := action.DelayValue(); present {
		resolution, ok := state.values.Destinations[destination]
		if !ok || resolution.State != destinationStateResolved {
			return 0, false
		}
		delay, ok := resolution.Value.(time.Duration)
		return delay, ok && delay > 0
	}
	return action.Delay(), action.Delay() > 0
}

func projectSignalDeliveryEvent(batch signalDeliveryBatch) (SignalEvent, bool) {
	if !batch.Valid() || batch.FirstOccurrence() > uint64(^uint(0)>>1) || batch.LastOccurrence() > uint64(^uint(0)>>1) {
		return SignalEvent{}, false
	}
	representative := batch.FirstEvent()
	if batch.Aggregation() == runSignalAggregationLatest {
		representative = batch.LastEvent()
	}
	notification := representative.Notification()
	origin, ok := projectInvocationSignalOrigin(notification.Origin())
	if !ok {
		return SignalEvent{}, false
	}
	phase, ok := projectInvocationSignalPhase(representative.Phase())
	if !ok {
		return SignalEvent{}, false
	}
	var sender *SignalSender
	if semanticSender := notification.Sender(); semanticSender != nil {
		sender = &SignalSender{
			ProcessID: semanticSender.ProcessID,
			UserID:    semanticSender.UserID,
			Code:      semanticSender.Code,
		}
	}
	firstOccurrence := batch.FirstOccurrence()
	lastOccurrence := batch.LastOccurrence()
	if firstOccurrence > uint64(math.MaxInt) || lastOccurrence > uint64(math.MaxInt) {
		return SignalEvent{}, false
	}
	event := SignalEvent{
		Signal:          notification.Signal(),
		Origin:          origin,
		Sender:          sender,
		Phase:           phase,
		ObservedAt:      notification.ObservedAt(),
		FirstObservedAt: batch.FirstObservedAt(),
		LastObservedAt:  batch.LastObservedAt(),
		FirstOccurrence: int(firstOccurrence),
		LastOccurrence:  int(lastOccurrence),
		BatchSize:       batch.BatchSize(),
	}
	return event, true
}

func projectInvocationSignalPhase(phase SignalPhase) (SignalPhase, bool) {
	switch phase {
	case SignalPhaseResolution:
		return SignalPhaseResolution, true
	case SignalPhaseExecution:
		return SignalPhaseExecution, true
	case SignalPhaseCleanup:
		return SignalPhaseCleanup, true
	default:
		return SignalPhaseUnknown, false
	}
}

func projectInvocationSignalOrigin(origin SignalOrigin) (SignalOrigin, bool) {
	switch origin {
	case SignalOriginUnknown:
		return SignalOriginUnknown, true
	case SignalOriginTerminal:
		return SignalOriginTerminal, true
	case SignalOriginProcess:
		return SignalOriginProcess, true
	case SignalOriginKernel:
		return SignalOriginKernel, true
	case SignalOriginChild:
		return SignalOriginChild, true
	case SignalOriginRuntime:
		return SignalOriginRuntime, true
	default:
		return SignalOriginUnknown, false
	}
}

func (state *runSignals) cancelPrimaryFromRuntime(cause error) {
	if state == nil || cause == nil {
		return
	}
	signalCause, ok := errors.AsType[*signalCancellationCauseError](cause)
	if ok && signalCause.Batch.Valid() {
		event, ok := projectSignalDeliveryEvent(signalCause.Batch)
		if ok {
			cancellation := &SignalCancellation{
				Event: event,
			}
			state.mu.Lock()
			if state.cancellation == nil {
				state.cancellation = cancellation
			}
			state.mu.Unlock()
			state.cancelPrimaryWithCause(cancellation)
			return
		}
	}
	state.cancelPrimaryWithCause(cause)
}

func (state *runSignals) cancelPrimaryWithCause(cause error) {
	state.mu.Lock()
	shouldCancel := state.primaryCancelCause == nil
	if shouldCancel {
		state.primaryCancelCause = cause
	}
	state.mu.Unlock()
	if shouldCancel {
		state.cancelPrimary(cause)
	}
}

func (state *runSignals) prepareExecutionTransition(ctx context.Context, run *runState) (preparedSignalExecution, error) {
	if state == nil || state.runtime == nil || state.watch == nil {
		panic("cli: invalid Run Signal Execution transition owner")
	}
	if err := state.runtime.beginExecutionTransitionGate(ctx); err != nil {
		return preparedSignalExecution{}, err
	}

	bindings := make([]signalEndpointBinding[struct{}], 0, len(state.plan.endpoints))
	created := make([]*runSignalEndpoint[SignalEvent], 0, len(state.plan.endpoints))
	closeCreated := func() {
		for _, endpoint := range created {
			endpoint.Close()
		}
	}
	for _, endpointID := range state.plan.endpoints {
		record, ok := state.model.SignalEndpoint(endpointID)
		if !ok {
			closeCreated()
			panic("cli: accepted Signal endpoint cannot resolve")
		}
		binding, ok := record.Binding()
		if !ok {
			closeCreated()
			panic("cli: accepted Signal endpoint has no binding")
		}
		configuration, ok := run.configurations.configuration(record.command)
		if !ok || configuration == nil {
			closeCreated()
			panic("cli: accepted Signal endpoint has no execution configuration")
		}
		original, ok := binding.Read(configuration)
		if !ok {
			closeCreated()
			panic("cli: accepted Signal endpoint binding cannot read configuration")
		}
		endpoint, err := openSignalEndpoint[SignalEvent](SignalBacklogCapacity)
		if err != nil {
			closeCreated()
			return preparedSignalExecution{}, err
		}
		installed, ok := record.Adapt(endpoint.Receive())
		if !ok {
			endpoint.Close()
			closeCreated()
			panic("cli: accepted Signal endpoint cannot adapt producer")
		}
		created = append(created, endpoint)
		state.endpointProducers[endpointID] = endpoint
		composite := signalEndpointBinding[struct{}]{
			ID: endpointID,
			Install: func(*struct{}) {
				if !binding.Write(configuration, installed) {
					panic("cli: signal endpoint binding lost typed relationship")
				}
			},
			Restore: func(*struct{}) {
				if !binding.Write(configuration, original) {
					panic("cli: signal endpoint binding lost typed relationship")
				}
			},
			Close: endpoint.Close,
		}
		bindings = append(bindings, composite)
	}
	preparation, err := prepareSignalEndpoints(&state.endpointPlaceholder, bindings)
	if err != nil {
		closeCreated()
		return preparedSignalExecution{}, err
	}
	acceptance, err := acceptSignalAcceptanceSnapshot(
		SignalPhaseExecution,
		state.plan.activePolicyIDs(SignalPhaseExecution),
	)
	if err != nil {
		_ = preparation.Rollback()
		return preparedSignalExecution{}, err
	}
	_, err = commitSignalExecutionWatch(
		ctx,
		state.watch,
		state.plan.watch(SignalPhaseExecution),
		acceptance,
		preparation,
	)
	if err != nil {
		return preparedSignalExecution{}, err
	}
	if err := state.runtime.SetEndpoints(preparation); err != nil {
		panic("cli: committed Signal Execution endpoints cannot attach to runtime")
	}
	if err := state.runtime.drainTransitionPhase(ctx, SignalPhaseResolution); err != nil {
		return preparedSignalExecution{}, err
	}
	prepared := preparedSignalExecution{
		state:   state,
		release: state.runtime.commitExecutionTransitionGate,
	}
	return prepared, nil
}

func (state *runSignals) beginCleanup(ctx context.Context) {
	if state == nil || state.runtime == nil {
		return
	}
	state.mu.Lock()
	if state.closed || state.phase == SignalPhaseCleanup || state.cleanupStarted {
		state.mu.Unlock()
		return
	}
	state.cleanupStarted = true
	state.mu.Unlock()

	if ctx.Err() != nil {
		return
	}
	result := state.runtime.Result()
	if result.HasTerminalOutcome || len(result.ActionFailures) != 0 || len(result.RuntimeFailures) != 0 {
		return
	}

	acceptance, err := acceptSignalAcceptanceSnapshot(
		SignalPhaseCleanup,
		state.plan.activePolicyIDs(SignalPhaseCleanup),
	)
	if err == nil {
		_, err = state.watch.Transition(ctx, state.plan.watch(SignalPhaseCleanup), acceptance)
	}
	if err != nil {
		if errors.Is(err, errSignalWatchClosed) {
			return
		}
		if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
			return
		}
	}
	state.mu.Lock()
	if err != nil {
		state.runtimeFailures = append(state.runtimeFailures, err)
	} else {
		state.phase = SignalPhaseCleanup
	}
	state.mu.Unlock()
	if err != nil {
		// Fail synchronously cancels primary execution through state callbacks.
		// Never invoke it while state.mu is held.
		state.runtime.Fail(err)
	}
}

func (state *runSignals) executionContext(primaryCtx context.Context) context.Context {
	return context.WithValue(primaryCtx, signalDeferralContextKey{}, state)
}

func deferSignals(ctx context.Context) func() {
	state, ok := ctx.Value(signalDeferralContextKey{}).(*runSignals)
	if !ok || state == nil || state.runtime == nil {
		panic("cli: DeferSignalActions requires an active Run Execution/Cleanup context")
	}
	state.mu.Lock()
	phase := state.phase
	closed := state.closed
	state.mu.Unlock()
	if closed || (phase != SignalPhaseExecution && phase != SignalPhaseCleanup) {
		panic("cli: DeferSignalActions requires an active Run Execution/Cleanup context")
	}
	return state.runtime.Defer()
}

func (state *runSignals) settle(ctx context.Context, operation *invocation, primary error) runFailures {
	set := runFailures{
		primary: primary,
	}
	if state == nil || state.runtime == nil {
		return set
	}
	state.mu.Lock()
	state.closed = true
	state.mu.Unlock()
	result, cleanupErr := state.runtime.Shutdown(ctx, primary)
	for _, failure := range result.ActionFailures {
		set.signalActionFailures = append(
			set.signalActionFailures,
			projectSignalActionFailure(operation, state.model, state.plan.base, failure),
		)
	}
	for _, failure := range result.RuntimeFailures {
		set.signalRuntimeFailures = append(set.signalRuntimeFailures, projectSignalRuntimeFailure(operation, failure))
	}
	state.mu.Lock()
	phaseFailures := append([]error(nil), state.runtimeFailures...)
	cancellation := state.cancellation
	cancelCause := state.primaryCancelCause
	state.mu.Unlock()
	for _, failure := range phaseFailures {
		set.signalRuntimeFailures = append(set.signalRuntimeFailures, projectSignalRuntimeFailure(operation, failure))
	}
	if cleanupErr != nil {
		set.lifecycleCleanupFailures = append(
			set.lifecycleCleanupFailures,
			projectSignalRuntimeFailure(operation, cleanupErr),
		)
	}
	if result.HasTerminalOutcome {
		set.terminal = state.terminalError(result.TerminalOutcome)
	}
	if cancellation != nil {
		set.signalActionFailures = append(set.signalActionFailures, cancellation)
	}
	if cancelCause != nil {
		preserved := errors.Join(
			set.terminal,
			errors.Join(set.signalActionFailures...),
			errors.Join(set.signalRuntimeFailures...),
		)
		set.primary = suppressConsequentialCancellation(set.primary, cancelCause, preserved)
	}
	return set
}

func (state *runSignals) terminalError(terminal signalTerminalOutcome) error {
	for _, policy := range state.plan.base.policies {
		if policy.record.meta.id != terminal.PolicyID || terminal.ActionIndex < 0 {
			continue
		}
		actions := policy.definition.Actions()
		if terminal.ActionIndex >= len(actions) {
			panic("cli: terminal Signal action index exceeds accepted policy")
		}
		action := actions[terminal.ActionIndex]
		switch terminal.Kind {
		case signalDispatchActionAbort:
			return abortError{
				code: action.ExitCode(),
			}
		case signalDispatchActionReraise:
			event, ok := projectSignalDeliveryEvent(terminal.Batch)
			if !ok {
				panic("cli: terminal Signal batch cannot project an event")
			}
			return TerminatedBy(event.Signal)
		case signalDispatchActionInvalid, signalDispatchActionCallback, signalDispatchActionCancel:
		}
	}
	panic("cli: terminal Signal outcome does not match an accepted action")
}

func signalRunAdmissionFailure(operation *invocation, code DiagnosticCode, cause error) error {
	payload := operationDiagnosticPayload(operation, false)
	payload.Diagnostics = []diagnostic{{Code: string(code)}}
	return retainDiagnosticFailure(payload, cause)
}

func projectSignalActionFailure(operation *invocation, model *effectiveModel, plan signalPlan, failure error) error {
	actionFailure, ok := errors.AsType[*signalDispatchActionFailureError](failure)
	if !ok {
		return signalRunAdmissionFailure(operation, DiagnosticSignalActionFailure, failure)
	}
	for _, policy := range plan.policies {
		if policy.record.meta.id != actionFailure.PolicyID {
			continue
		}
		actions := policy.definition.Actions()
		if actionFailure.Index < 0 || actionFailure.Index >= len(actions) {
			break
		}
		publicAction, err := projectEffectiveSignalAction(model, actions[actionFailure.Index])
		if err != nil {
			break
		}
		event, ok := projectSignalDeliveryEvent(actionFailure.Batch)
		if !ok {
			break
		}
		if endpointOverflow, ok := errors.AsType[*signalEndpointOverflowError](actionFailure.Err); ok {
			endpointID, present := actions[actionFailure.Index].Endpoint()
			if !present {
				break
			}
			wrapped := &integrationFailureError{
				message: signalEndpointOverflowErrorText(
					endpointOverflow.Capacity,
					endpointOverflow.Buffered,
				),
				err:    actionFailure.Err,
				policy: integrationExitPolicyBarrier,
			}
			payload := operationDiagnosticPayload(operation, false)
			payload.Diagnostics = []diagnostic{
				{
					Code: string(DiagnosticSignalOverflow),
					SignalAction: buildDiagnosticSignalActionFailure(
						actionFailure.PolicyID,
						actionFailure.Index,
						actions[actionFailure.Index],
						publicAction,
						event,
					),
					SignalOverflow: &diagnosticSignalOverflowFailure{
						Boundary:   diagnosticSignalOverflowEndpoint,
						EndpointID: endpointID,
						Capacity:   endpointOverflow.Capacity,
						Buffered:   endpointOverflow.Buffered,
					},
				},
			}
			return retainDiagnosticFailure(payload, wrapped)
		}
		wrapped := &integrationFailureError{
			message: signalActionErrorText(actionFailure.Index),
			err:     actionFailure.Err,
			policy:  integrationExitPolicyBarrier,
		}
		payload := operationDiagnosticPayload(operation, false)
		payload.Diagnostics = []diagnostic{
			{
				Code: string(DiagnosticSignalActionFailure),
				SignalAction: buildDiagnosticSignalActionFailure(
					actionFailure.PolicyID,
					actionFailure.Index,
					actions[actionFailure.Index],
					publicAction,
					event,
				),
			},
		}
		return retainDiagnosticFailure(payload, wrapped)
	}
	return signalRunAdmissionFailure(operation, DiagnosticSignalActionFailure, failure)
}

func projectSignalRuntimeFailure(operation *invocation, failure error) error {
	sourceOverflow, ok := errors.AsType[*SignalSourceOverflowError](failure)
	if ok && validSignalSourceOverflow(sourceOverflow) {
		payload := operationDiagnosticPayload(operation, false)
		payload.Diagnostics = []diagnostic{{
			Code: string(DiagnosticSignalOverflow),
			SignalOverflow: &diagnosticSignalOverflowFailure{
				Boundary: diagnosticSignalOverflowSource,
				Capacity: sourceOverflow.Capacity,
				Buffered: sourceOverflow.Buffered,
			},
		}}
		return retainDiagnosticFailure(payload, failure)
	}

	if overflow, ok := errors.AsType[*signalRoutingOverflowError](failure); ok {
		payload := operationDiagnosticPayload(operation, false)
		payload.Diagnostics = []diagnostic{{
			Code: string(DiagnosticSignalOverflow),
			SignalOverflow: &diagnosticSignalOverflowFailure{
				Boundary: diagnosticSignalOverflowRouting,
				Capacity: overflow.Capacity,
				Buffered: overflow.Buffered,
			},
		}}
		return retainDiagnosticFailure(payload, failure)
	}
	return signalRunAdmissionFailure(operation, DiagnosticSignalSourceFailure, failure)
}

func buildDiagnosticSignalActionFailure(policyID SignalPolicyID, index int, input signalActionDefinition, publicAction SignalAction, event SignalEvent) *diagnosticSignalActionFailure {
	kind := diagnosticSignalActionKind("")
	switch input.Kind() {
	case signalActionKindIgnore:
		kind = diagnosticSignalActionIgnore
	case signalActionKindCancel:
		kind = diagnosticSignalActionCancel
	case signalActionKindHandle:
		kind = diagnosticSignalActionHandle
	case signalActionKindForward:
		kind = diagnosticSignalActionForward
	case signalActionKindSend:
		kind = diagnosticSignalActionSend
	case signalActionKindAbort:
		kind = diagnosticSignalActionAbort
	case signalActionKindReraise:
		kind = diagnosticSignalActionReraise
	case signalActionKindEmit:
		kind = diagnosticSignalActionEmit
	case signalActionKindUnknown:
	}
	endpoint, _ := input.Endpoint()
	signalValue, _ := input.SignalValue()
	delayValue, _ := input.DelayValue()
	failure := &diagnosticSignalActionFailure{
		PolicyID: policyID,
		Index:    index,
		Action: diagnosticSignalAction{
			Kind:           kind,
			Name:           input.Name(),
			EndpointID:     endpoint,
			Signal:         input.Signal(),
			SignalValue:    signalValue,
			SignalRedacted: publicAction.SignalRedacted,
			Delayed:        input.Delayed(),
			Delay:          input.Delay(),
			DelayValue:     delayValue,
			DelayRedacted:  publicAction.DelayRedacted,
			ExitCode:       input.ExitCode(),
			Sensitive:      input.Sensitive(),
		},
		Event: buildDiagnosticSignalEvent(event),
		Phase: event.Phase,
	}

	return failure
}

func buildDiagnosticSignalEvent(input SignalEvent) diagnosticSignalEvent {
	var sender *SignalSender
	if input.Sender != nil {
		sender = &SignalSender{
			ProcessID: input.Sender.ProcessID,
			UserID:    input.Sender.UserID,
			Code:      input.Sender.Code,
		}
	}
	event := diagnosticSignalEvent{
		Signal:          input.Signal,
		Origin:          input.Origin,
		Sender:          sender,
		Phase:           input.Phase,
		ObservedAt:      input.ObservedAt,
		FirstObservedAt: input.FirstObservedAt,
		LastObservedAt:  input.LastObservedAt,
		FirstOccurrence: input.FirstOccurrence,
		LastOccurrence:  input.LastOccurrence,
		BatchSize:       input.BatchSize,
	}

	return event
}
