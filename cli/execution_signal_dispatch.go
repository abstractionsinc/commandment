package cli

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"
)

var (
	errSignalOccurrenceInvalid   = errors.New("signal occurrence is invalid")
	errSignalOccurrenceOrder     = errors.New("signal occurrences are not strictly increasing")
	errSignalOccurrenceExhausted = errors.New("signal occurrence counter exhausted")
	errSignalAggregationInvalid  = errors.New("signal aggregation mode is invalid")
)

// runSignalAggregation is the runtime spelling of the canonical semantic
// aggregation value. Execution owns window formation, not a second aggregation
// vocabulary.
type runSignalAggregation = SignalAggregation

const (
	signalAggregationInvalid      = SignalAggregationUnknown
	runSignalAggregationEach      = SignalAggregationEach
	runSignalAggregationCoalesced = SignalAggregationCoalesced
	runSignalAggregationLatest    = SignalAggregationLatest
)

// signalOccurrenceCounter owns deterministic operation-local occurrence
// counting for already-selected signal counter identities. It does not decide
// whether a source notification matches a policy; the policy/routing owner
// calls Observe only at the exact semantic counting boundary.
type signalOccurrenceCounter struct {
	mu     sync.Mutex
	counts map[SignalCounterID]uint64
}

func prepareSignalOccurrenceCounter() *signalOccurrenceCounter {
	counter := &signalOccurrenceCounter{
		counts: make(map[SignalCounterID]uint64),
	}
	return counter
}

func (counter *signalOccurrenceCounter) Observe(id SignalCounterID) (uint64, error) {
	if counter == nil || !id.present() {
		return 0, errSignalOccurrenceInvalid
	}
	counter.mu.Lock()
	defer counter.mu.Unlock()
	current := counter.counts[id]
	if current == ^uint64(0) {
		return 0, errSignalOccurrenceExhausted
	}
	current++
	counter.counts[id] = current
	return current, nil
}

func (counter *signalOccurrenceCounter) Current(id SignalCounterID) (uint64, bool) {
	if counter == nil || !id.present() {
		return 0, false
	}
	counter.mu.Lock()
	defer counter.mu.Unlock()
	value, ok := counter.counts[id]
	return value, ok
}

// runSignalOccurrence binds one accepted source notification to the deterministic
// policy-counter occurrence assigned at the semantic routing boundary.
type runSignalOccurrence struct {
	event      signalAcceptedNotification
	occurrence uint64
}

func acceptSignalOccurrence(event signalAcceptedNotification, occurrence uint64) (runSignalOccurrence, error) {
	if occurrence == 0 || event.Sequence() == 0 || !event.Watch().Valid() || !event.Notification().Valid() {
		return runSignalOccurrence{}, errSignalOccurrenceInvalid
	}
	accepted := runSignalOccurrence{
		event:      event,
		occurrence: occurrence,
	}
	return accepted, nil
}

func (occurrence runSignalOccurrence) Valid() bool {
	return occurrence.occurrence != 0 && occurrence.event.Sequence() != 0 &&
		occurrence.event.Watch().Valid() && occurrence.event.Notification().Valid()
}

func (occurrence runSignalOccurrence) Event() signalAcceptedNotification {
	if !occurrence.Valid() {
		return signalAcceptedNotification{}
	}
	return occurrence.event
}

// signalDeliveryBatch is one explicitly bounded routing-state delivery
// group. Coalesced and Latest deliberately retain the complete accepted event
// set here; later public projection/action adaptation can select the exact
// representative required by the frozen policy without losing source facts.
type signalDeliveryBatch struct {
	aggregation runSignalAggregation
	occurrences []runSignalOccurrence
}

func (batch signalDeliveryBatch) Valid() bool {
	return batch.aggregation.valid() && validSignalOccurrenceWindow(batch.occurrences)
}

func (batch signalDeliveryBatch) Aggregation() runSignalAggregation { return batch.aggregation }

func (batch signalDeliveryBatch) Occurrences() []runSignalOccurrence {
	return append([]runSignalOccurrence(nil), batch.occurrences...)
}

func (batch signalDeliveryBatch) FirstOccurrence() uint64 {
	if !batch.Valid() {
		return 0
	}
	return batch.occurrences[0].occurrence
}

func (batch signalDeliveryBatch) LastOccurrence() uint64 {
	if !batch.Valid() {
		return 0
	}
	return batch.occurrences[len(batch.occurrences)-1].occurrence
}

func (batch signalDeliveryBatch) BatchSize() int {
	if !batch.Valid() {
		return 0
	}
	return len(batch.occurrences)
}

func (batch signalDeliveryBatch) FirstObservedAt() time.Time {
	if !batch.Valid() {
		return time.Time{}
	}
	return batch.occurrences[0].event.Notification().ObservedAt()
}

func (batch signalDeliveryBatch) LastObservedAt() time.Time {
	if !batch.Valid() {
		return time.Time{}
	}
	return batch.occurrences[len(batch.occurrences)-1].event.Notification().ObservedAt()
}

func (batch signalDeliveryBatch) FirstEvent() signalAcceptedNotification {
	if !batch.Valid() {
		return signalAcceptedNotification{}
	}
	return batch.occurrences[0].event
}

func (batch signalDeliveryBatch) LastEvent() signalAcceptedNotification {
	if !batch.Valid() {
		return signalAcceptedNotification{}
	}
	return batch.occurrences[len(batch.occurrences)-1].event
}

// buildSignalDeliveryBatches applies only the declared aggregation rule to an
// already-selected routing-state window. It never drains a channel, waits
// for time, inspects scheduler state, or invents source notifications.
func buildSignalDeliveryBatches(aggregation runSignalAggregation, window []runSignalOccurrence) ([]signalDeliveryBatch, error) {
	if !aggregation.valid() {
		return nil, errSignalAggregationInvalid
	}
	if len(window) == 0 {
		return nil, nil
	}
	copied := append([]runSignalOccurrence(nil), window...)
	if !validSignalOccurrenceWindow(copied) {
		return nil, errSignalOccurrenceOrder
	}

	switch aggregation {
	case runSignalAggregationEach:
		result := make([]signalDeliveryBatch, len(copied))
		for i := range copied {
			result[i] = signalDeliveryBatch{
				aggregation: runSignalAggregationEach,
				occurrences: []runSignalOccurrence{copied[i]},
			}
		}
		return result, nil
	case runSignalAggregationCoalesced, runSignalAggregationLatest:
		return []signalDeliveryBatch{{
			aggregation: aggregation,
			occurrences: copied,
		}}, nil
	default:
		return nil, errSignalAggregationInvalid
	}
}

func validSignalOccurrenceWindow(window []runSignalOccurrence) bool {
	if len(window) == 0 {
		return false
	}
	var previous uint64
	for i, occurrence := range window {
		if !occurrence.Valid() {
			return false
		}
		if i != 0 && occurrence.occurrence <= previous {
			return false
		}
		previous = occurrence.occurrence
	}
	return true
}

var (
	errSignalDispatchContextRequired         = errors.New("signal dispatcher requires a context")
	errSignalDispatchConfigurationRequired   = errors.New("signal dispatcher requires handed-off configuration")
	errSignalDispatchInvalidPolicyIdentity   = errors.New("signal dispatcher received an invalid policy identity")
	errSignalDispatchDuplicatePolicyIdentity = errors.New("signal dispatcher received a duplicate policy identity")
)

var (
	errSignalDispatchClosed            = errors.New("signal dispatcher is closed")
	errSignalDispatchInvalidEvent      = errors.New("signal dispatcher received an invalid accepted event")
	errSignalDispatchMissingPolicy     = errors.New("signal dispatcher is missing an accepted policy")
	errSignalDispatchInvalidAction     = errors.New("signal dispatcher received an invalid action")
	errSignalDispatchEscalation        = errors.New("signal escalation scheduling failed")
	errSignalDispatchCancelUnavailable = errors.New("signal primary cancellation is unavailable")
)

type signalDispatchMissingPolicyError struct {
	policyID SignalPolicyID
}

func (err signalDispatchMissingPolicyError) Error() string {
	return fmt.Sprintf("%s: policy %d", errSignalDispatchMissingPolicy, err.policyID.uint32())
}

func (err signalDispatchMissingPolicyError) Unwrap() error { return errSignalDispatchMissingPolicy }

type signalDispatchActionKind uint8

const (
	signalDispatchActionInvalid signalDispatchActionKind = iota
	signalDispatchActionCallback
	signalDispatchActionCancel
	signalDispatchActionAbort
	signalDispatchActionReraise
)

// signalDispatchAction receives one immutable routing delivery batch.
type signalDispatchAction[C any] struct {
	kind signalDispatchActionKind
	run  func(context.Context, *C, signalDeliveryBatch) error
}

func signalBatchCallbackAction[C any](run func(context.Context, *C, signalDeliveryBatch) error) signalDispatchAction[C] {
	action := signalDispatchAction[C]{
		kind: signalDispatchActionCallback,
		run:  run,
	}
	return action
}

func signalCancelDispatchAction[C any]() signalDispatchAction[C] {
	action := signalDispatchAction[C]{
		kind: signalDispatchActionCancel,
	}
	return action
}

func signalBatchAbortAction[C any](run func(context.Context, *C, signalDeliveryBatch) error) signalDispatchAction[C] {
	action := signalDispatchAction[C]{
		kind: signalDispatchActionAbort,
		run:  run,
	}
	return action
}

func signalBatchReraiseAction[C any](run func(context.Context, *C, signalDeliveryBatch) error) signalDispatchAction[C] {
	action := signalDispatchAction[C]{
		kind: signalDispatchActionReraise,
		run:  run,
	}
	return action
}

func (action signalDispatchAction[C]) valid() bool {
	switch action.kind {
	case signalDispatchActionCallback, signalDispatchActionAbort, signalDispatchActionReraise:
		return action.run != nil
	case signalDispatchActionCancel:
		return action.run == nil
	default:
		return false
	}
}

type signalDispatchEscalation[C any] struct {
	Action      signalDispatchAction[C]
	Delay       time.Duration
	ActionIndex int
}

type signalDispatchPolicy[C any] struct {
	ID          SignalPolicyID
	Counter     SignalCounterID
	Match       func(context.Context, signalAcceptedNotification) (bool, error)
	Occurrence  func(uint64) bool
	Aggregation runSignalAggregation
	Deferred    bool

	Immediate        []signalDispatchAction[C]
	Escalation       *signalDispatchEscalation[C]
	ContinueMatching bool
}

type signalBatchEscalationArm[C any] func(context.Context, *C, signalDeliveryBatch, SignalPolicyID, int, time.Duration, signalDispatchAction[C]) error

type signalDispatchActionFailureError struct {
	PolicyID   SignalPolicyID
	Index      int
	Kind       signalDispatchActionKind
	Event      signalAcceptedNotification
	Occurrence uint64
	Batch      signalDeliveryBatch
	Err        error
}

func (failure *signalDispatchActionFailureError) Error() string {
	if failure == nil {
		return "signal action failed"
	}
	return fmt.Sprintf(
		"signal action failed: policy %d action %d kind %d",
		failure.PolicyID.uint32(),
		failure.Index,
		failure.Kind,
	)
}

func (failure *signalDispatchActionFailureError) Unwrap() error {
	if failure == nil {
		return nil
	}
	return failure.Err
}

type signalPolicyMatchFailureError struct {
	PolicyID SignalPolicyID
	Event    signalAcceptedNotification
	Err      error
}

func (failure *signalPolicyMatchFailureError) Error() string {
	if failure == nil {
		return "signal policy evaluation failed"
	}
	return fmt.Sprintf("signal policy evaluation failed: policy %d", failure.PolicyID.uint32())
}

func (failure *signalPolicyMatchFailureError) Unwrap() error {
	if failure == nil {
		return nil
	}
	return failure.Err
}

type signalCancellationCauseError struct {
	PolicyID SignalPolicyID
	Event    signalAcceptedNotification
	Batch    signalDeliveryBatch
}

func (cause *signalCancellationCauseError) Error() string { return "signal canceled primary execution" }

type signalTerminalOutcome struct {
	Event       signalAcceptedNotification
	Batch       signalDeliveryBatch
	PolicyID    SignalPolicyID
	ActionIndex int
	Kind        signalDispatchActionKind
}

func (terminal signalTerminalOutcome) Valid() bool {
	return terminal.Event.Sequence() != 0 && terminal.PolicyID.present() &&
		(terminal.Kind == signalDispatchActionAbort || terminal.Kind == signalDispatchActionReraise)
}

type signalTerminalOutcomeAuthority struct {
	mu       sync.Mutex
	terminal signalTerminalOutcome
}

func (authority *signalTerminalOutcomeAuthority) Assign(terminal signalTerminalOutcome) bool {
	if authority == nil || !terminal.Valid() {
		return false
	}
	authority.mu.Lock()
	defer authority.mu.Unlock()
	if authority.terminal.Valid() {
		return false
	}
	authority.terminal = terminal
	return true
}

func (authority *signalTerminalOutcomeAuthority) TerminalOutcome() (signalTerminalOutcome, bool) {
	if authority == nil {
		return signalTerminalOutcome{}, false
	}
	authority.mu.Lock()
	defer authority.mu.Unlock()
	if !authority.terminal.Valid() {
		return signalTerminalOutcome{}, false
	}
	return authority.terminal, true
}

type signalDispatcher[C any] struct {
	serial sync.Mutex
	mu     sync.Mutex

	config             *C
	policies           map[SignalPolicyID]signalDispatchPolicy[C]
	callbackContext    context.Context
	cancelCallbacks    context.CancelCauseFunc
	cancelPrimary      func(error)
	armBatchEscalation signalBatchEscalationArm[C]
	terminal           signalTerminalOutcomeAuthority
	closed             bool
}

func prepareSignalDispatcher[C any](ctx context.Context, config *C, policies []signalDispatchPolicy[C], cancelPrimary func(error)) (*signalDispatcher[C], context.Context, error) {
	if ctx == nil {
		return nil, nil, errSignalDispatchContextRequired
	}
	if config == nil {
		return nil, nil, errSignalDispatchConfigurationRequired
	}
	indexed := make(map[SignalPolicyID]signalDispatchPolicy[C], len(policies))
	for _, policy := range policies {
		if !policy.ID.present() {
			return nil, nil, errSignalDispatchInvalidPolicyIdentity
		}
		if _, duplicate := indexed[policy.ID]; duplicate {
			return nil, nil, errSignalDispatchDuplicatePolicyIdentity
		}
		for _, action := range policy.Immediate {
			if !action.valid() {
				return nil, nil, errSignalDispatchInvalidAction
			}
		}
		if policy.Escalation != nil && !policy.Escalation.Action.valid() {
			return nil, nil, errSignalDispatchInvalidAction
		}
		indexed[policy.ID] = cloneSignalDispatchPolicy(policy)
	}
	callbackContext, cancelCallbacks := context.WithCancelCause(ctx)
	result := &signalDispatcher[C]{
		config:          config,
		policies:        indexed,
		callbackContext: callbackContext,
		cancelCallbacks: cancelCallbacks,
		cancelPrimary:   cancelPrimary,
	}
	return result, callbackContext, nil
}

func (dispatcher *signalDispatcher[C]) cancelCallbackContext(cause error) {
	if dispatcher == nil {
		return
	}
	dispatcher.cancelCallbacks(cause)
}

func (dispatcher *signalDispatcher[C]) SetBatchEscalationArm(arm signalBatchEscalationArm[C]) error {
	if dispatcher == nil || arm == nil {
		return errSignalDispatchEscalation
	}
	dispatcher.mu.Lock()
	defer dispatcher.mu.Unlock()
	if dispatcher.closed || dispatcher.armBatchEscalation != nil {
		return errSignalDispatchEscalation
	}
	dispatcher.armBatchEscalation = arm
	return nil
}

func cloneSignalDispatchPolicy[C any](policy signalDispatchPolicy[C]) signalDispatchPolicy[C] {
	clone := policy
	clone.Immediate = append([]signalDispatchAction[C](nil), policy.Immediate...)
	if policy.Escalation == nil {
		return clone
	}
	escalation := *policy.Escalation
	clone.Escalation = &escalation
	return clone
}

// DispatchBatch executes one already-matched, occurrence-selected, immutable
// routing batch. It performs no selector interpretation, counting, or window
// formation. The returned bool is the policy's ContinueMatching result after a
// successful nonterminal sequence.
func (dispatcher *signalDispatcher[C]) DispatchBatch(policyID SignalPolicyID, batch signalDeliveryBatch) (bool, error) {
	if dispatcher == nil || !batch.Valid() {
		return false, errSignalDispatchInvalidEvent
	}
	dispatcher.serial.Lock()
	defer dispatcher.serial.Unlock()
	dispatcher.mu.Lock()
	if dispatcher.closed {
		dispatcher.mu.Unlock()
		return false, errSignalDispatchClosed
	}
	ctx := dispatcher.callbackContext
	config := dispatcher.config
	policy, ok := dispatcher.policies[policyID]
	dispatcher.mu.Unlock()
	if !ok {
		failure := signalDispatchMissingPolicyError{
			policyID: policyID,
		}
		return false, failure
	}
	terminal, err := dispatcher.executePolicyBatch(ctx, config, batch, policy)
	if err != nil || terminal {
		return false, err
	}
	return policy.ContinueMatching, nil
}

// DispatchEscalation executes one previously armed delayed action through the
// same serial application-callback lane as immediate Signal actions.
func (dispatcher *signalDispatcher[C]) DispatchEscalation(policyID SignalPolicyID, actionIndex int, batch signalDeliveryBatch, action signalDispatchAction[C]) error {
	if dispatcher == nil || !batch.Valid() {
		return errSignalDispatchInvalidEvent
	}
	dispatcher.serial.Lock()
	defer dispatcher.serial.Unlock()
	dispatcher.mu.Lock()
	if dispatcher.closed {
		dispatcher.mu.Unlock()
		return errSignalDispatchClosed
	}
	ctx := dispatcher.callbackContext
	config := dispatcher.config
	dispatcher.mu.Unlock()
	_, err := dispatcher.executeActionBatch(ctx, config, batch, policyID, actionIndex, action)
	return err
}

func (dispatcher *signalDispatcher[C]) executePolicyBatch(ctx context.Context, config *C, batch signalDeliveryBatch, policy signalDispatchPolicy[C]) (bool, error) {
	for index, action := range policy.Immediate {
		if dispatcher.isClosed() || (index > 0 && context.Cause(ctx) != nil) {
			return true, nil //nolint:nilerr // Cancellation/close terminates remaining policy work; the lifecycle owner retains the cause.
		}
		terminal, err := dispatcher.executeActionBatch(ctx, config, batch, policy.ID, index, action)
		if err != nil {
			return terminal, err
		}
		if terminal {
			return true, nil
		}
	}
	if policy.Escalation == nil {
		return false, nil
	}
	if dispatcher.isClosed() || context.Cause(ctx) != nil {
		return true, nil //nolint:nilerr // Cancellation/close terminates escalation work; the lifecycle owner retains the cause.
	}
	escalation := policy.Escalation
	index := escalation.ActionIndex
	if index < 0 {
		index = len(policy.Immediate)
	}
	action := escalation.Action
	delay := escalation.Delay
	if delay <= 0 {
		return false, signalDispatchActionFailure(
			policy.ID,
			index,
			action.kind,
			batch,
			errSignalDispatchEscalation,
		)
	}
	dispatcher.mu.Lock()
	arm := dispatcher.armBatchEscalation
	dispatcher.mu.Unlock()
	if arm == nil {
		return false, signalDispatchActionFailure(
			policy.ID,
			index,
			action.kind,
			batch,
			errSignalDispatchEscalation,
		)
	}
	err := arm(ctx, config, batch, policy.ID, index, delay, action)
	if err != nil {
		return false, signalDispatchActionFailure(policy.ID, index, action.kind, batch, err)
	}
	return false, nil
}

func (dispatcher *signalDispatcher[C]) executeActionBatch(ctx context.Context, config *C, batch signalDeliveryBatch, policyID SignalPolicyID, index int, action signalDispatchAction[C]) (bool, error) {
	if !action.valid() {
		return false, signalDispatchActionFailure(
			policyID,
			index,
			action.kind,
			batch,
			errSignalDispatchInvalidAction,
		)
	}
	terminal := false
	switch action.kind {
	case signalDispatchActionInvalid:
		return false, signalDispatchActionFailure(
			policyID,
			index,
			action.kind,
			batch,
			errSignalDispatchInvalidAction,
		)
	case signalDispatchActionCallback:
	case signalDispatchActionCancel:
		if dispatcher.cancelPrimary == nil {
			return false, signalDispatchActionFailure(
				policyID,
				index,
				action.kind,
				batch,
				errSignalDispatchCancelUnavailable,
			)
		}
		dispatcher.cancelPrimary(&signalCancellationCauseError{
			Event: batch.FirstEvent(),
			Batch: batch,
		})
		return false, nil
	case signalDispatchActionAbort, signalDispatchActionReraise:
		terminal = true
		signalTerminalOutcomeValue := signalTerminalOutcome{
			Event:       batch.FirstEvent(),
			Batch:       batch,
			PolicyID:    policyID,
			ActionIndex: index,
			Kind:        action.kind,
		}
		dispatcher.terminal.Assign(signalTerminalOutcomeValue)
	}
	err := action.run(ctx, config, batch)
	if cause := context.Cause(ctx); cause != nil {
		err = suppressConsequentialCancellation(err, cause, cause)
	}
	if err != nil {
		return terminal, signalDispatchActionFailure(policyID, index, action.kind, batch, err)
	}
	return terminal, nil
}

func signalDispatchActionFailure(policyID SignalPolicyID, index int, kind signalDispatchActionKind, batch signalDeliveryBatch, err error) *signalDispatchActionFailureError {
	failure := &signalDispatchActionFailureError{
		PolicyID: policyID,
		Index:    index,
		Kind:     kind,
		Batch:    batch,
		Err:      err,
	}
	if batch.Valid() {
		failure.Event = batch.FirstEvent()
		failure.Occurrence = batch.FirstOccurrence()
	}
	return failure
}

func (dispatcher *signalDispatcher[C]) isClosed() bool {
	if dispatcher == nil {
		return true
	}
	dispatcher.mu.Lock()
	defer dispatcher.mu.Unlock()
	return dispatcher.closed
}

func (dispatcher *signalDispatcher[C]) TerminalOutcome() (signalTerminalOutcome, bool) {
	if dispatcher == nil {
		return signalTerminalOutcome{}, false
	}
	return dispatcher.terminal.TerminalOutcome()
}

func (dispatcher *signalDispatcher[C]) Close(cause error) {
	if dispatcher == nil {
		return
	}
	dispatcher.mu.Lock()
	if !dispatcher.closed {
		dispatcher.closed = true
		dispatcher.cancelCallbacks(cause)
	}
	dispatcher.mu.Unlock()
	dispatcher.serial.Lock()
	dispatcher.serial.Unlock() //nolint:staticcheck // Lock/unlock is an intentional barrier that waits for the serialized dispatch action.
}

var (
	errSignalRoutingClosed           = errors.New("signal routing admission is closed")
	errSignalRoutingInvalidPolicy    = errors.New("signal routing received an invalid policy")
	errSignalRoutingOccurrenceUpdate = errors.New("signal routing occurrence update failed")
)

// signalActionSequenceAdmission is the single semantic action-sequence admission
// boundary. It returns whether matching may continue to later policies for the
// represented notification(s). The caller owns serial action execution and
// terminal admission closure.
type signalActionSequenceAdmission func(context.Context, SignalPolicyID, signalDeliveryBatch) (bool, error)

type signalRoutingCursor struct {
	accepted signalAcceptedNotification
	policies []SignalPolicyID
	index    int
	counts   map[SignalCounterID]uint64
}

type signalRoutingWindow struct {
	policyID SignalPolicyID
	policy   signalDispatchPolicy[struct{}]

	revision           SignalWatchRevision
	phase              SignalPhase
	counter            SignalCounterID
	generation         uint64
	deferred           bool
	transitionDeferred bool
	closed             bool

	occurrences []runSignalOccurrence
	events      []*signalRoutingCursor
}

func (window *signalRoutingWindow) firstSequence() uint64 {
	if window == nil || len(window.occurrences) == 0 {
		return 0
	}
	return window.occurrences[0].Event().Sequence()
}

// signalRouting owns semantic policy progression, shared occurrence
// identity, routing-state aggregation windows, and dispatch-only deferral.
// It intentionally owns no endpoint queues, timers, goroutines, or source
// subscription state.
type signalRouting struct {
	policies    map[SignalPolicyID]signalDispatchPolicy[struct{}]
	occurrences *signalOccurrenceCounter
	admit       signalActionSequenceAdmission

	windows []*signalRoutingWindow
	open    map[SignalPolicyID]*signalRoutingWindow
	closed  bool
}

// prepareSignalRouting snapshots only the routing/counting fields of dispatch
// policies. Application callbacks/actions remain owned by SignalDispatcher;
// the routing policy copies use Match and Occurrence only.
func prepareSignalRouting[C any](policies []signalDispatchPolicy[C], admit signalActionSequenceAdmission) (*signalRouting, error) {
	if admit == nil {
		return nil, errSignalRoutingInvalidPolicy
	}
	indexed := make(map[SignalPolicyID]signalDispatchPolicy[struct{}], len(policies))
	for _, policy := range policies {
		if !policy.ID.present() || !policy.Counter.present() || policy.Match == nil || policy.Occurrence == nil ||
			!policy.Aggregation.valid() {
			return nil, errSignalRoutingInvalidPolicy
		}
		if _, duplicate := indexed[policy.ID]; duplicate {
			return nil, errSignalRoutingInvalidPolicy
		}
		indexed[policy.ID] = signalDispatchPolicy[struct{}]{
			ID:               policy.ID,
			Counter:          policy.Counter,
			Match:            policy.Match,
			Occurrence:       policy.Occurrence,
			Aggregation:      policy.Aggregation,
			Deferred:         policy.Deferred,
			ContinueMatching: policy.ContinueMatching,
		}
	}
	result := &signalRouting{
		policies:    indexed,
		occurrences: prepareSignalOccurrenceCounter(),
		admit:       admit,
		open:        make(map[SignalPolicyID]*signalRoutingWindow),
	}
	return result, nil
}

// Accept routes one already accepted notification. blocked/deferred state is
// snapshotted on SignalAcceptedNotification at the source/frontend acceptance
// boundary, so later goroutine scheduling cannot change window membership.
func (routing *signalRouting) Accept(ctx context.Context, accepted signalAcceptedNotification) error {
	if routing == nil || routing.closed {
		return errSignalRoutingClosed
	}
	if accepted.Sequence() == 0 || !accepted.Watch().Valid() || !accepted.Notification().Valid() {
		return errSignalDispatchInvalidEvent
	}
	state := &signalRoutingCursor{
		accepted: accepted,
		policies: accepted.Watch().Acceptance().Policies(),
		counts:   make(map[SignalCounterID]uint64),
	}
	return routing.route(ctx, state)
}

func (routing *signalRouting) route(ctx context.Context, state *signalRoutingCursor) error {
	for state.index < len(state.policies) {
		policyID := state.policies[state.index]
		state.index++
		policy, ok := routing.policies[policyID]
		if !ok {
			return errSignalDispatchMissingPolicy
		}
		matched, err := policy.Match(ctx, state.accepted)
		if err != nil {
			result := &signalPolicyMatchFailureError{
				PolicyID: policy.ID,
				Event:    state.accepted,
				Err:      err,
			}

			return result
		}
		if !matched {
			continue
		}

		occurrence, ok := state.counts[policy.Counter]
		if !ok {
			occurrence, err = routing.occurrences.Observe(policy.Counter)
			if err != nil {
				return errors.Join(errSignalRoutingOccurrenceUpdate, err)
			}
			state.counts[policy.Counter] = occurrence
		}
		if !policy.Occurrence(occurrence) {
			continue
		}

		routed, err := acceptSignalOccurrence(state.accepted, occurrence)
		if err != nil {
			return err
		}
		blocked, generation := state.accepted.DispatchBlocked()
		deferred := policy.Deferred && state.accepted.DispatchDeferred()
		transitionDeferred := state.accepted.TransitionDeferred()
		blocked, generation, err = routing.signalWindowState(
			policy.ID,
			blocked,
			deferred || transitionDeferred,
			generation,
		)
		if err != nil {
			return err
		}
		if blocked || deferred || transitionDeferred {
			if err := routing.appendWindow(
				policy,
				routed,
				state,
				deferred,
				transitionDeferred,
				generation,
			); err != nil {
				return err
			}
			return nil
		}

		batches, err := buildSignalDeliveryBatches(policy.Aggregation, []runSignalOccurrence{routed})
		if err != nil {
			return err
		}
		continueMatching, err := routing.admit(ctx, policy.ID, batches[0])
		if err != nil || !continueMatching {
			return err
		}
	}
	return nil
}

func (routing *signalRouting) signalWindowState(policyID SignalPolicyID, blocked, deferred bool, generation uint64) (bool, uint64, error) {
	if blocked || deferred {
		return blocked, generation, nil
	}
	pendingGeneration, pending := routing.pendingGeneration(policyID)
	if !pending {
		return blocked, generation, nil
	}
	blocked = true
	if pendingGeneration > generation {
		return blocked, pendingGeneration, nil
	}
	generation++
	if generation == 0 {
		return false, 0, errSignalOccurrenceExhausted
	}
	return blocked, generation, nil
}

func (routing *signalRouting) pendingGeneration(policyID SignalPolicyID) (uint64, bool) {
	var generation uint64
	var found bool
	for _, window := range routing.windows {
		if window == nil || len(window.occurrences) == 0 || window.policyID != policyID ||
			window.deferred || window.transitionDeferred {
			continue
		}
		if !found || window.generation > generation {
			generation = window.generation
			found = true
		}
	}
	return generation, found
}

func (routing *signalRouting) appendWindow(policy signalDispatchPolicy[struct{}], occurrence runSignalOccurrence, state *signalRoutingCursor, deferred, transitionDeferred bool, generation uint64) error {
	current := routing.open[policy.ID]
	if current != nil && !sameSignalRoutingWindow(
		current,
		occurrence.Event(),
		policy,
		deferred,
		transitionDeferred,
		generation,
	) {
		current.closed = true
		delete(routing.open, policy.ID)
		current = nil
	}
	if current == nil {
		current = &signalRoutingWindow{
			policyID:           policy.ID,
			policy:             policy,
			revision:           occurrence.Event().Watch().Revision(),
			phase:              occurrence.Event().Phase(),
			counter:            policy.Counter,
			generation:         generation,
			deferred:           deferred,
			transitionDeferred: transitionDeferred,
		}
		routing.windows = append(routing.windows, current)
		routing.open[policy.ID] = current
	}
	current.occurrences = append(current.occurrences, occurrence)
	current.events = append(current.events, state)
	return nil
}

func sameSignalRoutingWindow(window *signalRoutingWindow, accepted signalAcceptedNotification, policy signalDispatchPolicy[struct{}], deferred, transitionDeferred bool, generation uint64) bool {
	if window == nil || window.closed || window.policyID != policy.ID || window.counter != policy.Counter ||
		window.revision != accepted.Watch().
			Revision() ||
		window.phase != accepted.Phase() || window.deferred != deferred ||
		window.transitionDeferred != transitionDeferred {
		return false
	}
	// A dynamic deferral keeps the policy ineligible across intervening action
	// generations. Pure dispatcher contention forms one window per admitted
	// sequence generation.
	if deferred || transitionDeferred {
		return true
	}
	return window.generation == generation
}

// FlushBlockedThrough closes and admits non-deferred windows whose action
// generation has already completed. Windows belonging to a later generation
// remain open until that generation reaches its semantic eligibility boundary.
func (routing *signalRouting) FlushBlockedThrough(ctx context.Context, completedGeneration uint64) error {
	if routing == nil || routing.closed {
		return nil
	}
	for _, window := range routing.windows {
		if !window.deferred && !window.transitionDeferred && window.generation <= completedGeneration {
			window.closed = true
			if routing.open[window.policyID] == window {
				delete(routing.open, window.policyID)
			}
		}
	}
	return routing.flush(ctx, false)
}

// FlushDeferred closes all deferred windows at the authoritative final token
// release boundary, then admits them in accepted-source order.
func (routing *signalRouting) FlushDeferred(ctx context.Context) error {
	if routing == nil || routing.closed {
		return nil
	}
	for _, window := range routing.windows {
		if window.deferred && !window.transitionDeferred {
			window.closed = true
			delete(routing.open, window.policyID)
		}
	}
	return routing.flush(ctx, true)
}

// FlushTransitionPhase releases transition-gated windows for one pre-handoff
// phase. Other transition-gated phases remain queued until the irreversible
// execution handoff releases the transition gate.
func (routing *signalRouting) FlushTransitionPhase(ctx context.Context, phase SignalPhase) error {
	if routing == nil || routing.closed || !phase.valid() {
		return nil
	}
	for _, window := range routing.windows {
		if window == nil || !window.transitionDeferred || window.phase != phase {
			continue
		}
		window.transitionDeferred = false
		if !window.deferred {
			window.closed = true
			if routing.open[window.policyID] == window {
				delete(routing.open, window.policyID)
			}
		}
	}
	return routing.flush(ctx, false)
}

// ReleaseTransition removes the internal execution-transition gate. Dynamic
// DeferSignalActions ownership remains independent: dynamically deferred windows stay
// queued until the final dynamic token release.
func (routing *signalRouting) ReleaseTransition(ctx context.Context) error {
	if routing == nil || routing.closed {
		return nil
	}
	for _, window := range routing.windows {
		if window == nil || !window.transitionDeferred {
			continue
		}
		window.transitionDeferred = false
		if !window.deferred {
			window.closed = true
			if routing.open[window.policyID] == window {
				delete(routing.open, window.policyID)
			}
		}
	}
	return routing.flush(ctx, false)
}

func (routing *signalRouting) flush(ctx context.Context, includeDeferred bool) error {
	for {
		var ready []*signalRoutingWindow
		for _, window := range routing.windows {
			if window == nil || len(window.occurrences) == 0 || !window.closed || window.transitionDeferred ||
				window.deferred && !includeDeferred {
				continue
			}
			ready = append(ready, window)
		}
		if len(ready) == 0 {
			routing.compactWindows()
			return nil
		}
		slices.SortStableFunc(ready, func(left, right *signalRoutingWindow) int {
			return cmp.Compare(left.firstSequence(), right.firstSequence())
		})
		window := ready[0]
		if err := routing.flushWindow(ctx, window); err != nil {
			routing.removeWindow(window)
			return err
		}
		routing.removeWindow(window)
	}
}

func (routing *signalRouting) flushWindow(ctx context.Context, window *signalRoutingWindow) error {
	batches, err := buildSignalDeliveryBatches(window.policy.Aggregation, window.occurrences)
	if err != nil {
		return err
	}
	bySequence := make(map[uint64]*signalRoutingCursor, len(window.events))
	for _, event := range window.events {
		bySequence[event.accepted.Sequence()] = event
	}
	var failures []error
	for _, batch := range batches {
		continueMatching, dispatchErr := routing.admit(ctx, window.policyID, batch)
		for _, occurrence := range batch.Occurrences() {
			state := bySequence[occurrence.Event().Sequence()]
			if state == nil || dispatchErr != nil || !continueMatching {
				continue
			}
			if err := routing.route(ctx, state); err != nil {
				dispatchErr = err
			}
		}
		if dispatchErr == nil {
			continue
		}
		failures = append(failures, dispatchErr)

		// Every/Each conserves every represented occurrence even when one
		// already-admitted action sequence fails. Only a terminal admission
		// boundary (including endpoint overflow) may prevent a later batch
		// from starting.
		_, overflow := errors.AsType[*signalEndpointOverflowError](dispatchErr)
		if errors.Is(dispatchErr, errSignalRuntimeTerminal) || overflow {
			break
		}
	}
	return errors.Join(failures...)
}

func (routing *signalRouting) removeWindow(target *signalRoutingWindow) {
	for i, window := range routing.windows {
		if window != target {
			continue
		}
		copy(routing.windows[i:], routing.windows[i+1:])
		routing.windows[len(routing.windows)-1] = nil
		routing.windows = routing.windows[:len(routing.windows)-1]
		break
	}
	if routing.open[target.policyID] == target {
		delete(routing.open, target.policyID)
	}
}

func (routing *signalRouting) compactWindows() {
	result := routing.windows[:0]
	for _, window := range routing.windows {
		if window != nil && len(window.occurrences) != 0 {
			result = append(result, window)
		}
	}
	routing.windows = result
}

func (routing *signalRouting) PendingThrough(sequence uint64) int {
	if routing == nil || routing.closed || sequence == 0 {
		return 0
	}
	pending := 0
	for _, window := range routing.windows {
		if window == nil {
			continue
		}
		for _, occurrence := range window.occurrences {
			if occurrence.Event().Sequence() <= sequence {
				pending++
			}
		}
	}
	return pending
}

func (routing *signalRouting) PendingTransitionPhase(phase SignalPhase) int {
	if routing == nil || routing.closed || !phase.valid() {
		return 0
	}
	pending := 0
	for _, window := range routing.windows {
		if window == nil || !window.transitionDeferred || window.phase != phase {
			continue
		}
		pending += len(window.occurrences)
	}
	return pending
}

// CloseAdmission is the terminal boundary. Pending windows are discarded and
// later flush/release attempts are inert.
func (routing *signalRouting) CloseAdmission() {
	if routing == nil || routing.closed {
		return
	}
	routing.closed = true
	for i := range routing.windows {
		routing.windows[i] = nil
	}
	routing.windows = nil
	clear(routing.open)
}

func (routing *signalRouting) PendingWindows() int {
	if routing == nil || routing.closed {
		return 0
	}
	return len(routing.windows)
}
