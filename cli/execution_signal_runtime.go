package cli

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	errSignalRuntimeState              = errors.New("signal runtime is in an invalid state")
	errSignalRuntimeTerminal           = errors.New("signal runtime reached a terminal signal outcome")
	errSignalActionGenerationExhausted = errors.New("signal action generation exhausted")
)

type signalRuntimeResult struct {
	TerminalOutcome    signalTerminalOutcome
	HasTerminalOutcome bool
	ActionFailures     []error
	RuntimeFailures    []error
}

type signalRuntimeBarrierKind uint8

const (
	signalRuntimeBarrierInvalid signalRuntimeBarrierKind = iota
	signalRuntimeBarrierAcceptedThrough
	signalRuntimeBarrierTransitionPhase
	signalRuntimeBarrierDispatchableThrough
)

type signalRuntimeBarrier struct {
	kind     signalRuntimeBarrierKind
	sequence uint64
	phase    SignalPhase
	done     chan error
}

type signalDelayedArm[C any] struct {
	id          uint64
	policyID    SignalPolicyID
	actionIndex int
	batch       signalDeliveryBatch
	action      signalDispatchAction[C]
	timer       *time.Timer
	done        sync.Once
}

// signalRuntime owns one invocation's source stream, signal routing,
// serial dispatcher, delayed escalation timers, dynamic deferral count, and
// committed endpoint resources. It owns one terminal action-admission gate.
type signalRuntime[C any] struct {
	mu              sync.Mutex
	admissionSerial sync.Mutex

	watch      *signalWatch
	intake     *signalIntake
	dispatcher *signalDispatcher[C]
	routing    *signalRouting
	endpoints  *signalEndpointPreparation[C]

	cancelPrimary func(error)

	started             bool
	discard             bool
	shutdownSettling    bool
	admissionClosed     bool
	actionBusy          bool
	actionGeneration    uint64
	completedGeneration uint64
	deferCount          uint64

	transitioning     bool
	releaseTransition bool
	settlementBarrier *signalRuntimeBarrier

	shutdownCause  error
	actionFailures []error

	nextArm uint64
	arms    map[uint64]*signalDelayedArm[C]
	armWG   sync.WaitGroup

	wake         chan struct{}
	sourceDone   chan struct{}
	dispatchDone chan struct{}

	sourceCloseOnce sync.Once
	sourceCloseDone chan struct{}
	sourceCloseErr  error

	dispatcherCloseOnce sync.Once
	dispatcherCloseDone chan struct{}

	primaryCancelOnce sync.Once
}

func prepareSignalRuntime[C any](watch *signalWatch, intake *signalIntake, dispatcher *signalDispatcher[C], policies []signalDispatchPolicy[C], endpoints *signalEndpointPreparation[C], cancelPrimary func(error)) (*signalRuntime[C], error) {
	if watch == nil || intake == nil || dispatcher == nil {
		return nil, errSignalRuntimeState
	}
	runtime := &signalRuntime[C]{
		watch:               watch,
		intake:              intake,
		dispatcher:          dispatcher,
		endpoints:           endpoints,
		cancelPrimary:       cancelPrimary,
		arms:                make(map[uint64]*signalDelayedArm[C]),
		wake:                make(chan struct{}, 1),
		sourceDone:          make(chan struct{}),
		dispatchDone:        make(chan struct{}),
		sourceCloseDone:     make(chan struct{}),
		dispatcherCloseDone: make(chan struct{}),
	}
	routing, err := prepareSignalRouting(policies, runtime.admitBatch)
	if err != nil {
		return nil, err
	}
	if err := dispatcher.SetBatchEscalationArm(runtime.armEscalation); err != nil {
		return nil, err
	}
	runtime.routing = routing
	return runtime, nil
}

// SetEndpoints attaches the already atomically committed Execution endpoint
// preparation to the invocation runtime. It may be called once after a
// Resolution-only runtime has already started.
func (runtime *signalRuntime[C]) SetEndpoints(endpoints *signalEndpointPreparation[C]) error {
	if runtime == nil || endpoints == nil {
		return errSignalRuntimeState
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.endpoints != nil {
		return errSignalRuntimeState
	}
	runtime.endpoints = endpoints
	return nil
}

func (runtime *signalRuntime[C]) Start(ctx context.Context) error {
	if runtime == nil {
		return errSignalRuntimeState
	}
	runtime.mu.Lock()
	if runtime.started {
		runtime.mu.Unlock()
		return errSignalRuntimeState
	}
	runtime.started = true
	runtime.mu.Unlock()
	go runtime.readSource()
	go runtime.dispatchAccepted(ctx)
	return nil
}

func (runtime *signalRuntime[C]) readSource() {
	defer close(runtime.sourceDone)
	defer runtime.signalWake()

	runtime.watch.ReadEvents(func(event signalWatchEvent) {
		runtime.mu.Lock()
		closed := runtime.admissionClosed
		routingSnapshot := signalRoutingSnapshot{
			Blocked:            runtime.actionBusy,
			Generation:         runtime.actionGeneration,
			Deferred:           runtime.deferCount != 0,
			TransitionDeferred: runtime.transitioning,
		}
		if closed && event.Kind() == signalWatchEventNotification {
			runtime.mu.Unlock()
			return
		}
		err := runtime.intake.AcceptWithRoutingSnapshot(event, routingSnapshot)
		runtime.mu.Unlock()
		runtime.signalWake()
		if err == nil {
			return
		}

		if _, ok := errors.AsType[*signalRoutingOverflowError](err); ok {
			runtime.beginTerminalShutdown(err)
			return
		}
		// Source failure/protocol failure stops source acceptance but preserves
		// already accepted notifications for ordered settlement.
		runtime.requestSourceClose(err)
	})
}

func (runtime *signalRuntime[C]) dispatchAccepted(ctx context.Context) {
	defer close(runtime.dispatchDone)
	defer func() {
		runtime.mu.Lock()
		cause := runtime.shutdownCause
		runtime.mu.Unlock()
		runtime.requestDispatcherClose(cause)
		<-runtime.dispatcherCloseDone
	}()

	for {
		for {
			event, ok := runtime.intake.Next()
			if !ok {
				break
			}
			runtime.mu.Lock()
			discard := runtime.discard
			runtime.mu.Unlock()
			if discard {
				continue
			}

			if err := runtime.routing.Accept(ctx, event); err != nil {
				runtime.recordRoutingError(err)
			}
			if runtime.dispatchTerminal() {
				return
			}
		}

		if terminal := runtime.intake.Terminal(); terminal != nil {
			runtime.requestSourceClose(terminal)
		}

		runtime.mu.Lock()
		closed := runtime.admissionClosed
		deferred := runtime.deferCount != 0
		busy := runtime.actionBusy
		completedGeneration := runtime.completedGeneration
		releaseTransition := runtime.releaseTransition
		if releaseTransition {
			runtime.releaseTransition = false
		}
		runtime.mu.Unlock()

		if releaseTransition {
			if err := runtime.routing.ReleaseTransition(ctx); err != nil {
				runtime.recordRoutingError(err)
			}
		}

		switch {
		case closed:
			runtime.routing.CloseAdmission()
		case !busy:
			if err := runtime.routing.FlushBlockedThrough(ctx, completedGeneration); err != nil {
				runtime.recordRoutingError(err)
			}
			if !deferred {
				if err := runtime.routing.FlushDeferred(ctx); err != nil {
					runtime.recordRoutingError(err)
				}
			}
		}

		runtime.settleSettlementBarrier(ctx)

		if runtime.dispatchTerminal() {
			return
		}

		pendingWindows := runtime.routing.PendingWindows()
		pendingIntake := runtime.intake.Pending()
		if terminal := runtime.intake.Terminal(); terminal != nil &&
			runtime.sourceFinished() && pendingIntake == 0 && pendingWindows == 0 {
			runtime.beginTerminalShutdown(terminal)
			return
		}
		if runtime.sourceFinished() && pendingIntake == 0 && (pendingWindows == 0 || closed) {
			return
		}

		select {
		case <-runtime.wake:
		case <-runtime.sourceDone:
		}
	}
}

func (runtime *signalRuntime[C]) recordRoutingError(err error) {
	if err == nil {
		return
	}
	if _, ok := errors.AsType[*signalDispatchActionFailureError](err); ok {
		// admitBatch already recorded this action failure exactly once.
		return
	}
	runtime.mu.Lock()
	runtime.actionFailures = append(runtime.actionFailures, err)
	runtime.mu.Unlock()
	runtime.beginTerminalShutdown(err)
}

func (runtime *signalRuntime[C]) dispatchTerminal() bool {
	if terminal, ok := runtime.dispatcher.TerminalOutcome(); ok {
		_ = terminal
		runtime.beginTerminalShutdown(errSignalRuntimeTerminal)
		return true
	}
	return false
}

func (runtime *signalRuntime[C]) admitBatch(_ context.Context, policyID SignalPolicyID, batch signalDeliveryBatch) (bool, error) {
	runtime.admissionSerial.Lock()
	defer runtime.admissionSerial.Unlock()
	admitted, err := runtime.beginActionAdmission()
	if err != nil {
		runtime.recordActionAdmissionFailure(err)
		return false, nil
	}
	if !admitted {
		return false, nil
	}
	continueMatching, err := runtime.dispatcher.DispatchBatch(policyID, batch)
	runtime.endActionAdmission()
	if err != nil {
		runtime.mu.Lock()
		runtime.actionFailures = append(runtime.actionFailures, err)
		runtime.mu.Unlock()
	}
	if terminal, ok := runtime.dispatcher.TerminalOutcome(); ok {
		_ = terminal
		runtime.beginTerminalShutdown(errSignalRuntimeTerminal)
		return false, err
	}
	if err != nil {
		runtime.beginTerminalShutdown(err)
		return false, err
	}
	return continueMatching, nil
}

func (runtime *signalRuntime[C]) beginActionAdmission() (bool, error) {
	runtime.mu.Lock()
	if runtime.admissionClosed {
		runtime.mu.Unlock()
		return false, nil
	}
	if runtime.actionGeneration == ^uint64(0) {
		runtime.closeAdmissionLocked(errSignalActionGenerationExhausted)
		runtime.mu.Unlock()
		runtime.signalWake()
		return false, errSignalActionGenerationExhausted
	}
	runtime.actionGeneration++
	runtime.actionBusy = true
	runtime.mu.Unlock()
	return true, nil
}

func (runtime *signalRuntime[C]) recordActionAdmissionFailure(err error) {
	if err == nil {
		return
	}
	runtime.mu.Lock()
	runtime.actionFailures = append(runtime.actionFailures, err)
	runtime.mu.Unlock()
	runtime.beginTerminalShutdown(err)
}

func (runtime *signalRuntime[C]) endActionAdmission() {
	runtime.mu.Lock()
	runtime.actionBusy = false
	runtime.completedGeneration = runtime.actionGeneration
	runtime.mu.Unlock()
	runtime.signalWake()
}

func (runtime *signalRuntime[C]) armEscalation(ctx context.Context, _ *C, batch signalDeliveryBatch, policyID SignalPolicyID, actionIndex int, delay time.Duration, action signalDispatchAction[C]) error {
	if delay <= 0 || !action.valid() || !batch.Valid() {
		return errSignalDispatchEscalation
	}
	runtime.mu.Lock()
	if runtime.admissionClosed || runtime.shutdownSettling {
		runtime.mu.Unlock()
		return nil
	}
	runtime.nextArm++
	if runtime.nextArm == 0 {
		runtime.mu.Unlock()
		return errSignalDispatchEscalation
	}
	arm := &signalDelayedArm[C]{
		id:          runtime.nextArm,
		policyID:    policyID,
		actionIndex: actionIndex,
		batch:       batch,
		action:      action,
	}
	runtime.arms[arm.id] = arm
	runtime.armWG.Add(1)
	arm.timer = time.AfterFunc(delay, func() { runtime.fireEscalation(ctx, arm) })
	runtime.mu.Unlock()
	return nil
}

func (runtime *signalRuntime[C]) fireEscalation(_ context.Context, arm *signalDelayedArm[C]) {
	defer arm.done.Do(runtime.armWG.Done)
	runtime.admissionSerial.Lock()
	defer runtime.admissionSerial.Unlock()

	runtime.mu.Lock()
	delete(runtime.arms, arm.id)
	closed := runtime.admissionClosed || runtime.shutdownSettling
	runtime.mu.Unlock()
	if closed {
		return
	}
	admitted, admissionErr := runtime.beginActionAdmission()
	if admissionErr != nil {
		runtime.recordActionAdmissionFailure(admissionErr)
		return
	}
	if !admitted {
		return
	}
	err := runtime.dispatcher.DispatchEscalation(arm.policyID, arm.actionIndex, arm.batch, arm.action)
	runtime.endActionAdmission()
	if err != nil {
		runtime.mu.Lock()
		runtime.actionFailures = append(runtime.actionFailures, err)
		runtime.mu.Unlock()
	}
	if runtime.dispatchTerminal() {
		return
	}
	if err != nil {
		runtime.beginTerminalShutdown(err)
	}
}

// beginExecutionTransitionGate closes application action admission for newly
// accepted notifications while preserving source/watch acceptance. It waits
// until every notification accepted before the gate and every action derived
// from that frontier has settled.
func (runtime *signalRuntime[C]) beginExecutionTransitionGate(ctx context.Context) error {
	if runtime == nil {
		return errSignalRuntimeState
	}
	runtime.mu.Lock()
	if runtime.admissionClosed || runtime.transitioning || runtime.settlementBarrier != nil {
		runtime.mu.Unlock()
		return errSignalRuntimeState
	}
	runtime.transitioning = true
	frontier := runtime.intake.acceptedThrough()
	runtime.mu.Unlock()

	barrier := &signalRuntimeBarrier{
		kind:     signalRuntimeBarrierAcceptedThrough,
		sequence: frontier,
		done:     make(chan error, 1),
	}
	return runtime.waitSettlementBarrier(ctx, barrier)
}

// drainTransitionPhase releases and settles transition-gated work belonging to
// the named pre-handoff phase while leaving later-phase application actions
// gated.
func (runtime *signalRuntime[C]) drainTransitionPhase(ctx context.Context, phase SignalPhase) error {
	if runtime == nil || !phase.valid() {
		return errSignalRuntimeState
	}
	runtime.mu.Lock()
	valid := runtime.transitioning && !runtime.admissionClosed && runtime.settlementBarrier == nil
	runtime.mu.Unlock()
	if !valid {
		return errSignalRuntimeState
	}
	barrier := &signalRuntimeBarrier{
		kind:  signalRuntimeBarrierTransitionPhase,
		phase: phase,
		done:  make(chan error, 1),
	}
	return runtime.waitSettlementBarrier(ctx, barrier)
}

// drainDispatchableThrough settles accepted work through sequence that is already
// dispatch-eligible at terminal shutdown. Delayed escalations are intentionally
// excluded: an arm that has not begun action admission when shutdown starts is
// not admitted merely because earlier source input was accepted.
func (runtime *signalRuntime[C]) drainDispatchableThrough(ctx context.Context, sequence uint64) error {
	if runtime == nil {
		return errSignalRuntimeState
	}
	barrier := &signalRuntimeBarrier{
		kind:     signalRuntimeBarrierDispatchableThrough,
		sequence: sequence,
		done:     make(chan error, 1),
	}
	return runtime.waitSettlementBarrier(ctx, barrier)
}

func (runtime *signalRuntime[C]) waitSettlementBarrier(ctx context.Context, barrier *signalRuntimeBarrier) error {
	if barrier == nil || barrier.done == nil {
		return errSignalRuntimeState
	}
	runtime.mu.Lock()
	if runtime.admissionClosed || runtime.settlementBarrier != nil {
		cause := runtime.shutdownCause
		runtime.mu.Unlock()
		if cause != nil {
			return cause
		}
		return errSignalRuntimeState
	}
	runtime.settlementBarrier = barrier
	runtime.mu.Unlock()
	runtime.signalWake()

	select {
	case err := <-barrier.done:
		return err
	case <-runtime.dispatchDone:
		runtime.mu.Lock()
		if runtime.settlementBarrier == barrier {
			runtime.settlementBarrier = nil
		}
		busy := runtime.actionBusy
		cause := runtime.shutdownCause
		runtime.mu.Unlock()
		if barrier.kind == signalRuntimeBarrierDispatchableThrough && !busy &&
			runtime.intake.pendingThrough(barrier.sequence) == 0 &&
			runtime.routing.PendingThrough(barrier.sequence) == 0 {
			return nil
		}
		if cause != nil {
			return cause
		}
		return errSignalRuntimeState
	case <-ctx.Done():
		runtime.mu.Lock()
		if runtime.settlementBarrier == barrier {
			runtime.settlementBarrier = nil
		}
		runtime.mu.Unlock()
		runtime.signalWake()
		return ctx.Err() //nolint:wrapcheck // Barrier cancellation must preserve the exact context error identity.
	}
}

func (runtime *signalRuntime[C]) settleSettlementBarrier(ctx context.Context) {
	runtime.mu.Lock()
	barrier := runtime.settlementBarrier
	closed := runtime.admissionClosed
	cause := runtime.shutdownCause
	busy := runtime.actionBusy
	runtime.mu.Unlock()
	if barrier == nil {
		return
	}
	if closed {
		runtime.completeSettlementBarrier(barrier, cause)
		return
	}

	var ready bool
	switch barrier.kind {
	case signalRuntimeBarrierAcceptedThrough:
		runtime.mu.Lock()
		armsPending := runtime.hasArmsThroughLocked(barrier.sequence)
		runtime.mu.Unlock()
		ready = !busy && !armsPending && runtime.intake.pendingThrough(barrier.sequence) == 0 &&
			runtime.routing.PendingThrough(barrier.sequence) == 0

	case signalRuntimeBarrierTransitionPhase:
		if err := runtime.routing.FlushTransitionPhase(ctx, barrier.phase); err != nil {
			runtime.recordRoutingError(err)
			runtime.completeSettlementBarrier(barrier, err)
			return
		}
		runtime.mu.Lock()
		armsPending := runtime.hasArmsInPhaseLocked(barrier.phase)
		busy = runtime.actionBusy
		runtime.mu.Unlock()
		ready = !busy && !armsPending && runtime.intake.pendingPhase(barrier.phase) == 0 &&
			runtime.routing.PendingTransitionPhase(barrier.phase) == 0

	case signalRuntimeBarrierDispatchableThrough:
		ready = !busy && runtime.intake.pendingThrough(barrier.sequence) == 0 &&
			runtime.routing.PendingThrough(barrier.sequence) == 0

	default:
		runtime.completeSettlementBarrier(barrier, errSignalRuntimeState)
		return
	}

	if ready {
		runtime.completeSettlementBarrier(barrier, nil)
	}
}

func (runtime *signalRuntime[C]) completeSettlementBarrier(barrier *signalRuntimeBarrier, err error) {
	if err == nil {
		runtime.mu.Lock()
		if runtime.admissionClosed {
			err = runtime.shutdownCause
		}
		runtime.mu.Unlock()
	}
	runtime.mu.Lock()
	if runtime.settlementBarrier != barrier {
		runtime.mu.Unlock()
		return
	}
	runtime.settlementBarrier = nil
	runtime.mu.Unlock()
	if err == nil {
		barrier.done <- nil
		return
	}
	barrier.done <- err
}

func (runtime *signalRuntime[C]) hasArmsThroughLocked(sequence uint64) bool {
	for _, arm := range runtime.arms {
		for _, occurrence := range arm.batch.Occurrences() {
			if occurrence.Event().Sequence() <= sequence {
				return true
			}
		}
	}
	return false
}

func (runtime *signalRuntime[C]) hasArmsInPhaseLocked(phase SignalPhase) bool {
	for _, arm := range runtime.arms {
		for _, occurrence := range arm.batch.Occurrences() {
			if occurrence.Event().Phase() == phase {
				return true
			}
		}
	}
	return false
}

// commitExecutionTransitionGate is the infallible post-handoff release. Actual
// application action dispatch resumes on the runtime goroutine after the Run
// owner has already published Execution phase and handed off configuration.
func (runtime *signalRuntime[C]) commitExecutionTransitionGate() {
	if runtime == nil {
		panic("cli: invalid Signal Execution transition gate")
	}
	runtime.mu.Lock()
	if !runtime.transitioning || runtime.settlementBarrier != nil {
		runtime.mu.Unlock()
		panic("cli: invalid Signal Execution transition gate")
	}
	runtime.transitioning = false
	if !runtime.admissionClosed {
		runtime.releaseTransition = true
	}
	runtime.mu.Unlock()
	runtime.signalWake()
}

// Defer acquires one independent dispatch-only token. The returned release is
// idempotent. Tokens acquired after terminal closure are inert.
func (runtime *signalRuntime[C]) Defer() func() {
	if runtime == nil {
		return func() {}
	}
	runtime.mu.Lock()
	if runtime.admissionClosed {
		runtime.mu.Unlock()
		return func() {}
	}
	runtime.deferCount++
	runtime.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			runtime.mu.Lock()
			if runtime.admissionClosed || runtime.deferCount == 0 {
				runtime.mu.Unlock()
				return
			}
			runtime.deferCount--
			becameEligible := runtime.deferCount == 0
			runtime.mu.Unlock()
			if becameEligible {
				runtime.signalWake()
			}
		})
	}
}

// Fail begins terminal runtime shutdown for an externally owned live-runtime
// transition failure (for example a failed semantic phase watch commit). The
// caller remains responsible for recording the failure in its canonical
// SignalRuntime settlement branch.
func (runtime *signalRuntime[C]) Fail(cause error) {
	if runtime == nil || cause == nil {
		return
	}
	runtime.beginTerminalShutdown(cause)
}

func (runtime *signalRuntime[C]) sourceFinished() bool {
	select {
	case <-runtime.sourceDone:
		return true
	default:
		return false
	}
}

func (runtime *signalRuntime[C]) signalWake() {
	select {
	case runtime.wake <- struct{}{}:
	default:
	}
}

func (runtime *signalRuntime[C]) closeAdmission(cause error) {
	runtime.mu.Lock()
	runtime.closeAdmissionLocked(cause)
	runtime.mu.Unlock()
	runtime.signalWake()
}

func (runtime *signalRuntime[C]) closeAdmissionLocked(cause error) {
	if runtime.shutdownCause == nil && cause != nil {
		runtime.shutdownCause = cause
	}
	if runtime.admissionClosed {
		return
	}
	runtime.admissionClosed = true
	runtime.discard = true
	for id, arm := range runtime.arms {
		delete(runtime.arms, id)
		if arm.timer != nil && arm.timer.Stop() {
			arm.done.Do(runtime.armWG.Done)
		}
	}
}

func (runtime *signalRuntime[C]) beginTerminalShutdown(cause error) {
	if cause == nil {
		cause = errSignalRuntimeTerminal
	}
	runtime.closeAdmission(cause)
	runtime.mu.Lock()
	settledCause := runtime.shutdownCause
	runtime.mu.Unlock()
	runtime.primaryCancelOnce.Do(func() {
		if runtime.cancelPrimary != nil {
			runtime.cancelPrimary(settledCause)
		}
	})
	runtime.requestSourceClose(settledCause)
	runtime.requestDispatcherClose(settledCause)
}

func (runtime *signalRuntime[C]) requestSourceClose(cause error) {
	if runtime == nil {
		return
	}
	runtime.mu.Lock()
	if runtime.shutdownCause == nil && cause != nil {
		runtime.shutdownCause = cause
	}
	runtime.mu.Unlock()
	runtime.sourceCloseOnce.Do(func() {
		go func() {
			err := runtime.watch.Close()
			runtime.mu.Lock()
			runtime.sourceCloseErr = err
			runtime.mu.Unlock()
			close(runtime.sourceCloseDone)
			runtime.signalWake()
		}()
	})
}

func (runtime *signalRuntime[C]) requestDispatcherClose(cause error) {
	if runtime == nil {
		return
	}
	runtime.dispatcherCloseOnce.Do(func() {
		runtime.dispatcher.cancelCallbackContext(cause)
		go func() {
			runtime.admissionSerial.Lock()
			runtime.dispatcher.Close(cause)
			runtime.admissionSerial.Unlock()
			close(runtime.dispatcherCloseDone)
		}()
	})
}

// Shutdown establishes the terminal source frontier before closing action-sequence
// admission. Work accepted through that frontier is allowed one final dispatch
// opportunity only when dynamic deferral has already been fully released.
// Unreleased deferred work and unadmitted delayed escalations remain inert.
func (runtime *signalRuntime[C]) Shutdown(ctx context.Context, cause error) (signalRuntimeResult, error) {
	if runtime == nil {
		return signalRuntimeResult{}, errSignalRuntimeState
	}
	runtime.mu.Lock()
	started := runtime.started
	runtime.shutdownSettling = true
	for id, arm := range runtime.arms {
		if arm.timer != nil && arm.timer.Stop() {
			delete(runtime.arms, id)
			arm.done.Do(runtime.armWG.Done)
		}
	}
	runtime.mu.Unlock()

	runtime.requestSourceClose(cause)
	if started {
		<-runtime.sourceCloseDone
		<-runtime.sourceDone

		runtime.mu.Lock()
		drainDispatchable := !runtime.admissionClosed && runtime.deferCount == 0 &&
			!runtime.transitioning && runtime.settlementBarrier == nil
		runtime.mu.Unlock()
		if drainDispatchable {
			frontier := runtime.intake.acceptedThrough()
			// Terminal shutdown owns closure even if the final drain loses a race
			// with another terminal condition. The detached context exists only for
			// this bounded pre-close settlement frontier.
			_ = runtime.drainDispatchableThrough(context.WithoutCancel(ctx), frontier)
		}
		runtime.closeAdmission(cause)
		runtime.requestDispatcherClose(cause)
		runtime.signalWake()
		<-runtime.dispatchDone
	} else {
		<-runtime.sourceCloseDone
		runtime.closeAdmission(cause)
		runtime.requestDispatcherClose(cause)
		<-runtime.dispatcherCloseDone
	}
	runtime.armWG.Wait()

	runtime.mu.Lock()
	endpoints := runtime.endpoints
	sourceCloseErr := runtime.sourceCloseErr
	runtime.mu.Unlock()

	var endpointErr error
	if endpoints != nil {
		endpointErr = endpoints.Close()
	}
	return runtime.Result(), joinOrdered(sourceCloseErr, endpointErr)
}

func (runtime *signalRuntime[C]) Result() signalRuntimeResult {
	if runtime == nil {
		return signalRuntimeResult{}
	}
	runtime.mu.Lock()
	actions := append([]error(nil), runtime.actionFailures...)
	runtime.mu.Unlock()

	runtimeFailures := runtime.intake.Failures()
	terminal, hasTerminal := runtime.dispatcher.TerminalOutcome()
	result := signalRuntimeResult{
		TerminalOutcome:    terminal,
		HasTerminalOutcome: hasTerminal,
		ActionFailures:     actions,
		RuntimeFailures:    append([]error(nil), runtimeFailures...),
	}

	return result
}
