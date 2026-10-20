package cli

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
)

type signalAcceptanceSnapshot struct {
	phase    SignalPhase
	policies []SignalPolicyID
}

func acceptSignalAcceptanceSnapshot(phase SignalPhase, policies []SignalPolicyID) (signalAcceptanceSnapshot, error) {
	if !phase.valid() {
		return signalAcceptanceSnapshot{}, errInvalidSignalAcceptanceSnapshot
	}
	copied := append([]SignalPolicyID(nil), policies...)
	seen := make(map[SignalPolicyID]struct{}, len(copied))
	for _, policy := range copied {
		if !policy.present() {
			return signalAcceptanceSnapshot{}, errInvalidSignalAcceptanceSnapshot
		}
		if _, duplicate := seen[policy]; duplicate {
			return signalAcceptanceSnapshot{}, errInvalidSignalAcceptanceSnapshot
		}
		seen[policy] = struct{}{}
	}
	return signalAcceptanceSnapshot{
		phase:    phase,
		policies: copied,
	}, nil
}

func (snapshot signalAcceptanceSnapshot) Valid() bool { return snapshot.phase.valid() }

func (snapshot signalAcceptanceSnapshot) Phase() SignalPhase { return snapshot.phase }

func (snapshot signalAcceptanceSnapshot) Policies() []SignalPolicyID {
	return append([]SignalPolicyID(nil), snapshot.policies...)
}

type signalWatchEventKind uint8

const (
	signalWatchEventInvalid signalWatchEventKind = iota
	signalWatchEventNotification
	signalWatchEventFailure
)

type signalWatchEvent struct {
	kind         signalWatchEventKind
	revision     SignalWatchRevision
	notification signalNotification
	err          error
}

func acceptSignalWatchNotification(revision SignalWatchRevision, notification signalNotification) (signalWatchEvent, error) {
	if !revision.valid() || !notification.Valid() {
		return signalWatchEvent{}, errInvalidSignalWatchEvent
	}
	event := signalWatchEvent{
		kind:         signalWatchEventNotification,
		revision:     revision,
		notification: notification,
	}
	return event, nil
}

func acceptSignalWatchFailure(err error) (signalWatchEvent, error) {
	if err == nil {
		return signalWatchEvent{}, errInvalidSignalWatchEvent
	}
	return signalWatchEvent{
		kind: signalWatchEventFailure,
		err:  err,
	}, nil
}

func (event signalWatchEvent) Valid() bool {
	switch event.kind {
	case signalWatchEventNotification:
		return event.revision.valid() && event.notification.Valid() && event.err == nil
	case signalWatchEventFailure:
		return !event.revision.valid() && !event.notification.Valid() && event.err != nil
	default:
		return false
	}
}

func (event signalWatchEvent) Kind() signalWatchEventKind { return event.kind }

func (event signalWatchEvent) Revision() SignalWatchRevision { return event.revision }

func (event signalWatchEvent) Notification() signalNotification { return event.notification }

func (event signalWatchEvent) Err() error { return event.err }

var (
	errSignalIntakeInvalidCapacity = errors.New("signal intake capacity must be positive")
	errSignalIntakeNilWatch        = errors.New("signal intake requires a signal watch")
	errSignalIntakeTerminal        = errors.New("signal intake is terminal")
	errSignalWatchSource           = errors.New("signal watch source failure")
)

// runSignalWatchSourceError reports a source event that cannot belong to the
// authoritative accepted stream for the current watch state.
type runSignalWatchSourceError struct {
	Reason   string
	Revision SignalWatchRevision
}

func (err *runSignalWatchSourceError) Error() string {
	if err == nil {
		return errSignalWatchSource.Error()
	}
	if err.Revision.valid() {
		return fmt.Sprintf("%s: %s (revision %d)", errSignalWatchSource, err.Reason, err.Revision)
	}
	return fmt.Sprintf("%s: %s", errSignalWatchSource, err.Reason)
}

func (err *runSignalWatchSourceError) Unwrap() error { return errSignalWatchSource }

// signalRoutingOverflowError reports exhaustion of the frontend-owned intake
// boundary. It does not describe source-owned or application-endpoint pressure.
type signalRoutingOverflowError struct {
	Capacity int
	Buffered int
}

func (err *signalRoutingOverflowError) Error() string {
	if err == nil {
		return "signal routing overflow"
	}
	return fmt.Sprintf(
		"signal routing overflow: capacity %d, buffered %d",
		err.Capacity,
		err.Buffered,
	)
}

// signalRoutingSnapshot is the exact action-admission routing state observed
// at the source/frontend acceptance boundary for one notification.
type signalRoutingSnapshot struct {
	Blocked            bool
	Generation         uint64
	Deferred           bool
	TransitionDeferred bool
}

// signalAcceptedNotification is one source notification admitted to the
// canonical operation stream together with the exact watch acceptance state
// and routing snapshot that governed source acceptance.
type signalAcceptedNotification struct {
	sequence     uint64
	notification signalNotification
	watch        signalWatchState
	routing      signalRoutingSnapshot
}

type signalIntakeEntry struct {
	accepted          signalAcceptedNotification
	revision          SignalWatchRevision
	transitionSettled <-chan struct{}
}

func (accepted signalAcceptedNotification) Sequence() uint64 {
	return accepted.sequence
}

func (accepted signalAcceptedNotification) Notification() signalNotification {
	return accepted.notification
}

func (accepted signalAcceptedNotification) Phase() SignalPhase {
	if !accepted.watch.Valid() {
		return SignalPhaseUnknown
	}
	return accepted.watch.Acceptance().Phase()
}

func (accepted signalAcceptedNotification) DispatchBlocked() (bool, uint64) {
	return accepted.routing.Blocked, accepted.routing.Generation
}

func (accepted signalAcceptedNotification) DispatchDeferred() bool {
	return accepted.routing.Deferred
}

func (accepted signalAcceptedNotification) TransitionDeferred() bool {
	return accepted.routing.TransitionDeferred
}

func (accepted signalAcceptedNotification) Watch() signalWatchState {
	if !accepted.watch.Valid() {
		return signalWatchState{}
	}
	return snapshotSignalWatchState(
		accepted.watch.revision,
		accepted.watch.signals,
		accepted.watch.acceptance,
	)
}

// signalIntake owns the bounded frontend acceptance queue between the source
// event stream and serialized routing/action execution. It never blocks source
// failure observation on notification backlog availability.
type signalIntake struct {
	mu sync.Mutex

	watch    *signalWatch
	capacity int
	queue    []signalIntakeEntry

	nextSequence   uint64
	lastRevision   SignalWatchRevision
	terminal       error
	failures       []error
	sourceTerminal bool
}

func prepareSignalIntake(watch *signalWatch, capacity int) (*signalIntake, error) {
	if watch == nil {
		return nil, errSignalIntakeNilWatch
	}
	if capacity <= 0 {
		return nil, errSignalIntakeInvalidCapacity
	}
	result := &signalIntake{
		watch:        watch,
		capacity:     capacity,
		nextSequence: 1,
	}
	return result, nil
}

// Accept admits one canonical watch event. Notifications are either queued in
// source order or turn the intake terminal on protocol/pressure failure. A
// terminal source failure is recorded even when the notification queue is full.
func (intake *signalIntake) Accept(event signalWatchEvent) error {
	return intake.AcceptWithRoutingSnapshot(event, signalRoutingSnapshot{})
}

// AcceptWithDispatchState admits one watch event together with the exact
// global action-admission state observed at the source/frontend acceptance
// boundary. The generation changes for each admitted action sequence and lets
// routing distinguish notifications that became pending behind one
// sequence from notifications accepted after the dispatcher was eligible again.
func (intake *signalIntake) AcceptWithDispatchState(event signalWatchEvent, blocked bool, generation uint64) error {
	return intake.AcceptWithRoutingSnapshot(event, signalRoutingSnapshot{
		Blocked:    blocked,
		Generation: generation,
	})
}

// AcceptWithRoutingSnapshot admits one watch event with the exact acceptance-time
// routing snapshot, including whether dispatch-only dynamic deferral was active.
func (intake *signalIntake) AcceptWithRoutingSnapshot(event signalWatchEvent, routing signalRoutingSnapshot) error {
	if intake == nil {
		return errSignalIntakeTerminal
	}

	intake.mu.Lock()
	defer intake.mu.Unlock()

	if !event.Valid() {
		return intake.failLocked(&runSignalWatchSourceError{
			Reason: "invalid source event",
		})
	}

	if event.Kind() == signalWatchEventFailure {
		if intake.sourceTerminal {
			return intake.failLocked(&runSignalWatchSourceError{
				Reason: "watch event followed terminal source failure",
			})
		}
		intake.sourceTerminal = true
		return intake.sourceFailureLocked(event.Err())
	}
	if intake.sourceTerminal {
		return intake.failLocked(&runSignalWatchSourceError{
			Reason: "watch event followed terminal source failure",
		})
	}
	if intake.terminal != nil {
		return errors.Join(errSignalIntakeTerminal, intake.terminal)
	}

	switch event.Kind() {
	case signalWatchEventNotification:
		return intake.acceptNotificationLocked(event, routing)
	default:
		return intake.failLocked(&runSignalWatchSourceError{
			Reason: "unknown watch event kind",
		})
	}
}

func (intake *signalIntake) acceptNotificationLocked(event signalWatchEvent, routing signalRoutingSnapshot) error {
	revision := event.Revision()
	if intake.lastRevision.valid() && revision < intake.lastRevision {
		return intake.failLocked(&runSignalWatchSourceError{
			Reason:   "notification revision decreased",
			Revision: revision,
		})
	}

	watch, published, transitionSettled := intake.watch.lookupRevision(revision)
	if !published && transitionSettled == nil {
		return intake.failLocked(&runSignalWatchSourceError{
			Reason:   "notification references an unestablished watch revision",
			Revision: revision,
		})
	}

	if len(intake.queue) >= intake.capacity {
		return intake.failLocked(&signalRoutingOverflowError{
			Capacity: intake.capacity,
			Buffered: len(intake.queue),
		})
	}
	if intake.nextSequence == 0 {
		return intake.failLocked(&runSignalWatchSourceError{
			Reason: "accepted notification sequence exhausted",
		})
	}

	entry := signalIntakeEntry{
		accepted: signalAcceptedNotification{
			sequence:     intake.nextSequence,
			notification: event.Notification(),
			watch:        watch,
			routing:      routing,
		},
		revision:          revision,
		transitionSettled: transitionSettled,
	}
	intake.queue = append(intake.queue, entry)
	intake.nextSequence++
	intake.lastRevision = revision
	return nil
}

func (intake *signalIntake) failLocked(err error) error {
	if err == nil {
		err = errSignalWatchSource
	}
	if intake.terminal == nil {
		intake.terminal = err
	}
	intake.failures = append(intake.failures, err)
	return err
}

func (intake *signalIntake) sourceFailureLocked(err error) error {
	if err == nil {
		err = errSignalWatchSource
	}
	establishedTerminal := intake.terminal == nil
	if establishedTerminal {
		intake.terminal = err
	}
	intake.failures = append(intake.failures, err)
	if establishedTerminal {
		return err
	}
	return errors.Join(intake.terminal, err)
}

// Next removes the oldest accepted notification. A notification from the exact
// transition currently being established waits here, after source intake, until
// that transition either publishes its watch state or is rejected.
func (intake *signalIntake) Next() (signalAcceptedNotification, bool) {
	if intake == nil {
		return signalAcceptedNotification{}, false
	}

	for {
		intake.mu.Lock()
		if len(intake.queue) == 0 {
			intake.mu.Unlock()
			return signalAcceptedNotification{}, false
		}

		entry := intake.queue[0]
		if entry.transitionSettled == nil {
			intake.removeFirstLocked()
			intake.mu.Unlock()
			return entry.accepted, true
		}

		sequence := entry.accepted.sequence
		revision := entry.revision
		transitionSettled := entry.transitionSettled
		intake.mu.Unlock()

		<-transitionSettled

		intake.mu.Lock()
		if len(intake.queue) == 0 || intake.queue[0].accepted.sequence != sequence {
			intake.mu.Unlock()
			continue
		}
		watch, ok := intake.watch.StateAt(revision)
		if !ok {
			_ = intake.failLocked(&runSignalWatchSourceError{
				Reason:   "notification references an unestablished watch revision",
				Revision: revision,
			})
			intake.queue = nil
			intake.mu.Unlock()
			return signalAcceptedNotification{}, false
		}
		intake.queue[0].accepted.watch = watch
		intake.queue[0].transitionSettled = nil
		intake.mu.Unlock()
	}
}

func (intake *signalIntake) removeFirstLocked() {
	copy(intake.queue, intake.queue[1:])
	intake.queue[len(intake.queue)-1] = signalIntakeEntry{}
	intake.queue = intake.queue[:len(intake.queue)-1]
}

func (intake *signalIntake) acceptedThrough() uint64 {
	if intake == nil {
		return 0
	}
	intake.mu.Lock()
	defer intake.mu.Unlock()
	if intake.nextSequence == 0 {
		return ^uint64(0)
	}
	return intake.nextSequence - 1
}

func (intake *signalIntake) pendingThrough(sequence uint64) int {
	if intake == nil || sequence == 0 {
		return 0
	}
	intake.mu.Lock()
	defer intake.mu.Unlock()
	pending := 0
	for _, entry := range intake.queue {
		if entry.accepted.sequence <= sequence {
			pending++
		}
	}
	return pending
}

func (intake *signalIntake) pendingPhase(phase SignalPhase) int {
	if intake == nil || !phase.valid() {
		return 0
	}
	intake.mu.Lock()
	defer intake.mu.Unlock()
	pending := 0
	for _, entry := range intake.queue {
		watch := entry.accepted.watch
		if !watch.Valid() {
			resolved, ok := intake.watch.StateAt(entry.revision)
			if !ok {
				// An in-flight transition has not published this revision yet. Keep
				// the drain conservative until the transition settles.
				pending++
				continue
			}
			watch = resolved
		}
		if watch.Acceptance().Phase() == phase {
			pending++
		}
	}
	return pending
}

func (intake *signalIntake) Pending() int {
	if intake == nil {
		return 0
	}
	intake.mu.Lock()
	defer intake.mu.Unlock()
	return len(intake.queue)
}

func (intake *signalIntake) Terminal() error {
	if intake == nil {
		return errSignalIntakeTerminal
	}
	intake.mu.Lock()
	defer intake.mu.Unlock()
	return intake.terminal
}

// Failures returns the ordered independent intake/source failures observed so
// far. The first entry that established terminal state remains Terminal().
func (intake *signalIntake) Failures() []error {
	if intake == nil {
		return nil
	}
	intake.mu.Lock()
	defer intake.mu.Unlock()
	return append([]error(nil), intake.failures...)
}

var (
	errSignalWatchClosed            = errors.New("signal watch is closed")
	errSignalWatchNilSubscription   = errors.New("signal source returned a nil subscription")
	errSignalWatchRevisionExhausted = errors.New("signal watch revision exhausted")
)

// signalWatchState is an immutable snapshot of the watched-signal state that is
// authoritative at the frontend/source acceptance boundary.
type signalWatchState struct {
	revision   SignalWatchRevision
	signals    []Signal
	acceptance signalAcceptanceSnapshot
}

func snapshotSignalWatchState(revision SignalWatchRevision, signals []Signal, acceptance signalAcceptanceSnapshot) signalWatchState {
	state := signalWatchState{
		revision:   revision,
		signals:    cloneSignals(signals),
		acceptance: acceptance,
	}

	return state
}

func (state signalWatchState) Valid() bool {
	return state.revision.valid() && state.acceptance.Valid()
}

func (state signalWatchState) Revision() SignalWatchRevision {
	return state.revision
}

// Signals returns the semantic signals watched by this state.
func (state signalWatchState) Signals() []Signal {
	return cloneSignals(state.signals)
}

func (state signalWatchState) Acceptance() signalAcceptanceSnapshot {
	return state.acceptance
}

// signalWatchSubscription is the execution package's narrow adapter over the
// frozen public signal-subscription transition and shutdown protocol.
type signalWatchSubscription interface {
	SetWatch(ctx context.Context, revision SignalWatchRevision, signals []Signal) error
	ReadEvents(consume func(signalWatchEvent))
	Close() error
}

// signalWatchSubscribe establishes the single invocation-scoped subscription.
type signalWatchSubscribe func(context.Context, SignalWatchRevision, []Signal) (signalWatchSubscription, error)

// signalWatchTransition tracks one in-flight watched-set transition for the
// invocation-scoped signal watch.
type signalWatchTransition struct {
	revision SignalWatchRevision
	settled  chan struct{}
}

type signalWatch struct {
	// mu protects published watch state and close bookkeeping only. Host calls
	// must never run while mu is held: shutdown must remain able to mark the
	// watch closed and cancel an in-flight transition independently.
	mu sync.Mutex

	// transitionMu serializes SetWatch proposals without coupling host progress
	// to state/close progress. At most one transition can be in the host at once.
	transitionMu sync.Mutex

	subscription     signalWatchSubscription
	state            signalWatchState
	states           map[SignalWatchRevision]signalWatchState
	nextRevision     SignalWatchRevision
	activeTransition *signalWatchTransition

	closed                 bool
	activeTransitionCancel context.CancelFunc
	closeOnce              sync.Once
	closeErr               error
}

// prepareSignalWatch establishes the initial watch under revision one.
// An empty watched set is valid because later semantic phase transitions may
// activate a nonempty watch while retaining the same invocation subscription.
// The caller must validate effective source/host requirements first.
func prepareSignalWatch(ctx context.Context, initial []Signal, acceptance signalAcceptanceSnapshot, subscribe signalWatchSubscribe) (*signalWatch, error) {
	if subscribe == nil {
		return nil, errSignalWatchNilSubscription
	}
	if !acceptance.Valid() {
		return nil, errInvalidSignalAcceptanceSnapshot
	}

	const initialRevision SignalWatchRevision = 1
	watched := cloneSignals(initial)
	subscription, err := subscribe(ctx, initialRevision, cloneSignals(watched))
	if err != nil {
		return nil, err
	}
	if subscription == nil {
		return nil, errSignalWatchNilSubscription
	}

	initialState := snapshotSignalWatchState(initialRevision, watched, acceptance)
	result := &signalWatch{
		subscription: subscription,
		state:        initialState,
		states: map[SignalWatchRevision]signalWatchState{
			initialRevision: initialState,
		},
		nextRevision: initialRevision + 1,
	}
	return result, nil
}

// State returns a detached snapshot of the last successfully established watch.
func (watch *signalWatch) State() signalWatchState {
	if watch == nil {
		return signalWatchState{}
	}

	watch.mu.Lock()
	defer watch.mu.Unlock()
	return snapshotSignalWatchState(watch.state.revision, watch.state.signals, watch.state.acceptance)
}

// StateAt returns the detached watch state for one successfully established
// revision. Failed proposal revisions are absent. Established revisions remain
// available through shutdown so delayed accepted notifications can retain their
// original source-acceptance identity.
func (watch *signalWatch) StateAt(revision SignalWatchRevision) (signalWatchState, bool) {
	state, ok, _ := watch.lookupRevision(revision)
	return state, ok
}

func (watch *signalWatch) lookupRevision(revision SignalWatchRevision) (signalWatchState, bool, <-chan struct{}) {
	if watch == nil || !revision.valid() {
		return signalWatchState{}, false, nil
	}

	watch.mu.Lock()
	defer watch.mu.Unlock()
	state, ok := watch.states[revision]
	if ok {
		return snapshotSignalWatchState(state.revision, state.signals, state.acceptance), true, nil
	}
	if watch.activeTransition != nil && watch.activeTransition.revision == revision {
		return signalWatchState{}, false, watch.activeTransition.settled
	}
	return signalWatchState{}, false, nil
}

// Transition requests one new watch revision. A failed transition does not
// change the authoritative state. Proposal revisions are never reused.
func (watch *signalWatch) Transition(ctx context.Context, watched []Signal, acceptance signalAcceptanceSnapshot) (signalWatchState, error) {
	if watch == nil {
		return signalWatchState{}, errSignalWatchClosed
	}

	if !acceptance.Valid() {
		return signalWatchState{}, errInvalidSignalAcceptanceSnapshot
	}

	watch.transitionMu.Lock()
	defer watch.transitionMu.Unlock()

	watch.mu.Lock()
	if watch.closed {
		watch.mu.Unlock()
		return signalWatchState{}, errSignalWatchClosed
	}
	if watch.nextRevision == 0 {
		watch.mu.Unlock()
		return signalWatchState{}, errSignalWatchRevisionExhausted
	}

	proposalRevision := watch.nextRevision
	if proposalRevision == SignalWatchRevision(math.MaxUint64) {
		watch.nextRevision = 0
	} else {
		watch.nextRevision++
	}
	proposal := cloneSignals(watched)
	subscription := watch.subscription
	transitionCtx, transitionCancel := context.WithCancel(ctx)
	activeTransition := &signalWatchTransition{
		revision: proposalRevision,
		settled:  make(chan struct{}),
	}
	watch.activeTransition = activeTransition
	watch.activeTransitionCancel = transitionCancel
	watch.mu.Unlock()

	// SetWatch is host-owned and may block until its context is canceled. Never
	// hold watch.mu here: Close must be able to cancel this exact call.
	err := subscription.SetWatch(transitionCtx, proposalRevision, cloneSignals(proposal))
	transitionCancel()

	watch.mu.Lock()
	watch.activeTransitionCancel = nil
	authoritative := snapshotSignalWatchState(
		watch.state.revision,
		watch.state.signals,
		watch.state.acceptance,
	)
	if !watch.closed && err == nil {
		watch.state = snapshotSignalWatchState(proposalRevision, proposal, acceptance)
		watch.states[proposalRevision] = watch.state
		authoritative = snapshotSignalWatchState(
			watch.state.revision,
			watch.state.signals,
			watch.state.acceptance,
		)
	}
	watch.activeTransition = nil
	close(activeTransition.settled)
	closed := watch.closed
	watch.mu.Unlock()

	if closed {
		return authoritative, errSignalWatchClosed
	}
	if err != nil {
		return authoritative, err //nolint:wrapcheck // The source transition error must pass through unchanged.
	}
	return authoritative, nil
}

// ReadEvents drains the canonical source stream until the owned subscription
// closes. The callback must return promptly; it is invoked in source order.
// Runtime shutdown may call Close concurrently, which closes the underlying
// source before this method returns.
func (watch *signalWatch) ReadEvents(accept func(signalWatchEvent)) {
	if watch == nil || accept == nil {
		return
	}

	watch.mu.Lock()
	subscription := watch.subscription
	watch.mu.Unlock()
	if subscription == nil {
		return
	}
	subscription.ReadEvents(accept)
}

// Close shuts down the owned subscription exactly once. Concurrent or repeated
// calls observe the same underlying close result.
func (watch *signalWatch) Close() error {
	if watch == nil {
		return nil
	}

	watch.mu.Lock()
	watch.closed = true
	transitionCancel := watch.activeTransitionCancel
	subscription := watch.subscription
	watch.mu.Unlock()

	// Cancellation is deliberately reachable without transitionMu. A pending
	// host SetWatch must never prevent shutdown from delivering its cancellation.
	if transitionCancel != nil {
		transitionCancel()
	}
	watch.closeOnce.Do(func() {
		err := subscription.Close()
		watch.mu.Lock()
		watch.closeErr = err
		watch.mu.Unlock()
	})

	watch.mu.Lock()
	defer watch.mu.Unlock()
	return watch.closeErr
}

func cloneSignals(signals []Signal) []Signal {
	return append([]Signal(nil), signals...)
}

type signalWatchSourceSubscription struct {
	afterDone func(func()) func() bool
	cancel    context.CancelFunc

	mu        sync.Mutex
	sources   []compiledSignalSource
	watch     map[Signal]struct{}
	revision  SignalWatchRevision
	active    map[uint64]*signalSourceActivation
	events    chan signalWatchEvent
	wg        sync.WaitGroup
	closed    bool
	closeOnce sync.Once
	closeErr  error
}

type signalSourceActivation struct {
	ordinary *erasedOrdinarySignalStream
	advanced *erasedAdvancedSignalSubscription
	cancel   context.CancelFunc
	read     func()
}

func adaptSignalSources(sources []compiledSignalSource) signalWatchSubscribe {
	copied := append([]compiledSignalSource(nil), sources...)
	return func(
		ctx context.Context,
		revision SignalWatchRevision,
		watched []Signal,
	) (signalWatchSubscription, error) {
		subscriptionCtx, cancel := context.WithCancel(ctx)
		subscription := &signalWatchSourceSubscription{
			afterDone: func(f func()) func() bool { return context.AfterFunc(subscriptionCtx, f) },
			cancel:    cancel,
			sources:   copied,
			watch:     make(map[Signal]struct{}),
			revision:  revision,
			active:    make(map[uint64]*signalSourceActivation),
			events:    make(chan signalWatchEvent, SignalBacklogCapacity),
		}
		if err := subscription.applyWatch(ctx, revision, watched, true); err != nil {
			cancel()
			_ = subscription.Close()
			return nil, err
		}
		return subscription, nil
	}
}

func (subscription *signalWatchSourceSubscription) SetWatch(ctx context.Context, revision SignalWatchRevision, watched []Signal) error {
	if subscription == nil {
		return errSignalWatchClosed
	}
	return subscription.applyWatch(ctx, revision, watched, false)
}

func (subscription *signalWatchSourceSubscription) applyWatch(ctx context.Context, revision SignalWatchRevision, watched []Signal, initial bool) error {
	subscription.mu.Lock()
	if subscription.closed {
		subscription.mu.Unlock()
		return errSignalWatchClosed
	}
	subscription.mu.Unlock()

	desired := make(map[Signal]struct{}, len(watched))
	for _, signal := range watched {
		if !signal.valid() {
			return errInvalidSignalSourceDeclaration
		}
		desired[signal] = struct{}{}
	}

	// Establish newly required sources before publishing the frontend watch.
	newlyActivated := make(map[uint64]struct{})
	var readersToStart []*signalSourceActivation
	for i := range subscription.sources {
		source := subscription.sources[i]
		subset := sourceWatch(source, desired)
		if len(subset) == 0 {
			continue
		}
		subscription.mu.Lock()
		activation := subscription.active[source.id]
		subscription.mu.Unlock()
		if activation == nil {
			var err error
			activation, err = subscription.activate(ctx, source, revision, subset)
			if err != nil {
				return err
			}
			subscription.mu.Lock()
			if subscription.closed {
				subscription.mu.Unlock()
				_ = subscription.closeActivation(activation)
				return errSignalWatchClosed
			}
			subscription.active[source.id] = activation
			subscription.mu.Unlock()
			newlyActivated[source.id] = struct{}{}
			readersToStart = append(readersToStart, activation)
		}
	}

	// Advanced sources own fallible watched-state transitions. Subscribe already
	// established the requested revision for sources activated by this transition.
	if !initial {
		if err := subscription.updateAdvancedSignalWatches(ctx, revision, desired, newlyActivated); err != nil {
			return err
		}
	}

	subscription.mu.Lock()
	subscription.watch = desired
	subscription.revision = revision
	subscription.mu.Unlock()
	for _, activation := range readersToStart {
		subscription.startReader(activation)
	}
	return nil
}

func (subscription *signalWatchSourceSubscription) updateAdvancedSignalWatches(ctx context.Context, revision SignalWatchRevision, desired map[Signal]struct{}, newlyActivated map[uint64]struct{}) error {
	for i := range subscription.sources {
		source := subscription.sources[i]
		if _, fresh := newlyActivated[source.id]; fresh {
			continue
		}
		subscription.mu.Lock()
		activation := subscription.active[source.id]
		subscription.mu.Unlock()
		if activation == nil || activation.advanced == nil {
			continue
		}
		subset := sourceWatch(source, desired)
		if activation.advanced.setWatch == nil {
			if len(subset) != 0 {
				return errInvalidSignalSourceDeclaration
			}
			continue
		}
		if err := activation.advanced.setWatch(ctx, revision, subset); err != nil {
			return err
		}
	}
	return nil
}

func sourceWatch(source compiledSignalSource, desired map[Signal]struct{}) []Signal {
	result := make([]Signal, 0, len(source.descriptors))
	for _, descriptor := range source.descriptors {
		if _, ok := desired[descriptor.Signal()]; ok {
			result = append(result, descriptor.Signal())
		}
	}
	return result
}

func (subscription *signalWatchSourceSubscription) activate(ctx context.Context, source compiledSignalSource, revision SignalWatchRevision, watched []Signal) (*signalSourceActivation, error) {
	lifecycleCtx, lifecycleCancel := subscription.activationContext(ctx)
	activation := &signalSourceActivation{cancel: lifecycleCancel}
	if source.ordinary {
		stream, err := source.open(ctx)
		if err != nil {
			lifecycleCancel()
			return nil, err
		}
		activation.ordinary = &stream
		activation.read = func() { subscription.readOrdinary(lifecycleCtx, activation) }
		return activation, nil
	}
	advanced, err := source.subscribe(lifecycleCtx, ctx, revision, watched)
	if err != nil {
		lifecycleCancel()
		return nil, err
	}
	activation.advanced = &advanced
	activation.read = func() { subscription.readAdvanced(lifecycleCtx, activation) }
	return activation, nil
}

func (subscription *signalWatchSourceSubscription) activationContext(ctx context.Context) (context.Context, context.CancelFunc) {
	lifecycleCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	stop := subscription.afterDone(cancel)
	return lifecycleCtx, func() {
		stop()
		cancel()
	}
}

func (subscription *signalWatchSourceSubscription) startReader(activation *signalSourceActivation) {
	subscription.wg.Go(activation.read)
}

func (subscription *signalWatchSourceSubscription) readOrdinary(ctx context.Context, activation *signalSourceActivation) {
	for {
		notification, err := activation.ordinary.receive(ctx)
		if err != nil {
			if ctx.Err() == nil {
				subscription.publishFailure(ctx, err)
			}
			return
		}
		subscription.mu.Lock()
		_, watched := subscription.watch[notification.Signal]
		revision := subscription.revision
		subscription.mu.Unlock()
		if !watched {
			continue
		}
		canonical, err := adaptErasedSignalNotification(notification)
		if err != nil {
			subscription.publishFailure(ctx, err)
			return
		}
		event, err := acceptSignalWatchNotification(revision, canonical)
		if err != nil {
			subscription.publishFailure(ctx, err)
			return
		}
		if !subscription.publish(ctx, event) {
			return
		}
	}
}

func (subscription *signalWatchSourceSubscription) readAdvanced(ctx context.Context, activation *signalSourceActivation) {
	for event := range activation.advanced.events {
		switch event.kind {
		case SignalSubscriptionEventNotification:
			canonical, err := adaptErasedSignalNotification(event.notification)
			if err != nil {
				subscription.publishFailure(ctx, err)
				return
			}
			notification, err := acceptSignalWatchNotification(
				event.revision,
				canonical,
			)
			if err != nil {
				subscription.publishFailure(ctx, err)
				return
			}
			if !subscription.publish(ctx, notification) {
				return
			}
		case SignalSubscriptionEventFailure:
			if event.err == nil {
				subscription.publishFailure(ctx, signalWatchSourceError("malformed failure event", event.revision))
			} else {
				subscription.publishFailure(ctx, event.err)
			}
			return
		default:
			subscription.publishFailure(ctx, signalWatchSourceError("unknown watch event kind", event.revision))
			return
		}
	}
	if ctx.Err() == nil {
		subscription.publishFailure(
			ctx,
			signalWatchSourceError("signal subscription event stream closed without terminal failure or Close", 0),
		)
	}
}

func (subscription *signalWatchSourceSubscription) publish(ctx context.Context, event signalWatchEvent) bool {
	select {
	case subscription.events <- event:
		return true
	case <-ctx.Done():
		return false
	}
}

func (subscription *signalWatchSourceSubscription) publishFailure(ctx context.Context, err error) {
	if err == nil {
		return
	}
	overflow, ok := errors.AsType[*SignalSourceOverflowError](err)
	if ok && !validSignalSourceOverflow(overflow) {
		err = signalWatchSourceError("invalid signal source overflow", 0)
	}
	failure, failureErr := acceptSignalWatchFailure(err)
	if failureErr == nil {
		subscription.publish(ctx, failure)
	}
}

func (subscription *signalWatchSourceSubscription) ReadEvents(accept func(signalWatchEvent)) {
	if subscription == nil || accept == nil {
		return
	}
	for event := range subscription.events {
		accept(event)
	}
}

func (subscription *signalWatchSourceSubscription) Close() error {
	if subscription == nil {
		return nil
	}
	subscription.closeOnce.Do(func() {
		subscription.mu.Lock()
		subscription.closed = true
		activations := make([]*signalSourceActivation, 0, len(subscription.active))
		for _, activation := range subscription.active {
			activations = append(activations, activation)
		}
		subscription.mu.Unlock()
		subscription.cancel()
		var errs []error
		for _, activation := range activations {
			if err := subscription.closeActivation(activation); err != nil {
				errs = append(errs, err)
			}
		}
		subscription.wg.Wait()
		close(subscription.events)
		subscription.closeErr = errors.Join(errs...)
	})
	return subscription.closeErr
}

func (subscription *signalWatchSourceSubscription) closeActivation(activation *signalSourceActivation) error {
	if activation == nil {
		return nil
	}
	if activation.cancel != nil {
		activation.cancel()
	}
	if activation.ordinary != nil && activation.ordinary.close != nil {
		return activation.ordinary.close()
	}
	if activation.advanced != nil && activation.advanced.close != nil {
		return activation.advanced.close()
	}
	return nil
}

func adaptErasedSignalNotification(input SignalNotification[Signal]) (signalNotification, error) {
	origin, ok := adaptSignalOrigin(input.Origin)
	if !input.Signal.valid() || !ok {
		return signalNotification{}, signalWatchSourceError("invalid signal notification", 0)
	}
	var sender *SignalSender
	if input.Sender != nil {
		adapted := acceptSignalSender(input.Sender.ProcessID, input.Sender.UserID, input.Sender.Code)
		sender = &adapted
	}
	notification, err := acceptSignalNotification(input.Signal, origin, sender, input.ObservedAt)
	if err != nil {
		return signalNotification{}, signalWatchSourceError("invalid signal notification", 0)
	}
	return notification, nil
}

func adaptSignalOrigin(origin SignalOrigin) (SignalOrigin, bool) {
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

func validSignalSourceOverflow(overflow *SignalSourceOverflowError) bool {
	if overflow == nil {
		return false
	}
	if overflow.Capacity == 0 && overflow.Buffered == 0 {
		return true
	}
	return overflow.Capacity > 0 && overflow.Buffered == overflow.Capacity
}

func signalWatchSourceError(reason string, revision SignalWatchRevision) error {
	return &runSignalWatchSourceError{
		Reason:   reason,
		Revision: revision,
	}
}
