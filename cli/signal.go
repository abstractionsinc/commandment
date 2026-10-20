package cli

import (
	"context"
	"errors"
	"time"
)

// Signals is the command-scoped signal-policy declaration surface. Policies participate in
// canonical root-to-leaf inheritance. A zero Signals value has no authoring scope and cannot
// declare policy; use the value supplied by CommandBuilder.
type Signals[C any] struct {
	Config *C
	Match  signalMatches[C]
	Action signalActions[C]

	session *declarationSession
	profile grammarProfileCaseRef
}

// SignalEndpointDeclaration identifies one Definition-owned structured signal-event endpoint. The
// channel storage pointer is used only to bind invocation state; semantic identity is carried by
// this declaration.
type SignalEndpointDeclaration[C any] struct {
	endpoint signalEndpointDraftRef
}

// Endpoint declares one structured SignalEvent endpoint bound to storage.
func (signals *Signals[C]) Endpoint[Events ~<-chan SignalEvent](storage *Events) SignalEndpointDeclaration[C] {
	if signals == nil || signals.session == nil || signals.Config == nil || storage == nil {
		return SignalEndpointDeclaration[C]{}
	}
	binding, ok := bindConfigurationStorage(signals.Config, storage)
	var endpoint signalEndpointDraftRef
	signals.session.use(func(draft *declarationDraft, _ commandRef) {
		record := signalEndpointDraft{
			binding: binding,
			adapt: func(value any) (any, bool) {
				receive, receiveOK := value.(<-chan SignalEvent)
				if !receiveOK {
					return nil, false
				}
				return Events(receive), true
			},
			valid: ok && !signalEndpointOverlapsDestination(draft, binding),
		}
		endpoint = draft.signalEndpoints.append(draft, record)
	})
	return SignalEndpointDeclaration[C]{endpoint: endpoint}
}

// ClearInherited removes every currently surviving signal policy owned by strict ancestors of this
// command. It preserves local policies, does not reorder survivors, and does not create a
// persistent clear mode.
func (signals *Signals[C]) ClearInherited() { stageSignalClear(signals) }

// Policy appends one signal policy at the current declaration position. The match and plan are
// validated together for required source metadata, host capabilities, phase legality, and action
// grammar before activation.
func (signals *Signals[C]) Policy(match SignalMatch[C], plan SignalActionPlan[C]) {
	stageSignalPolicy(signals, match, plan)
}

// signalEndpointAdapter is the typed authoring conversion captured before a named receive-channel
// endpoint is erased into the semantic model.
type signalEndpointAdapter func(any) (any, bool)

type signalEndpointRecord struct {
	meta    recordMeta[SignalEndpointID]
	command CommandID
	binding configurationBinding
	adapt   signalEndpointAdapter
}

func acceptSignalEndpointRecord(id SignalEndpointID, command CommandID, position declarationPosition) signalEndpointRecord {
	mustCommand(command)
	return signalEndpointRecord{
		meta:    acceptRecordMeta(id, position),
		command: command,
	}
}

// WithBinding returns record with its configuration binding and typed endpoint
// adapter captured before semantic erasure.

func (record signalEndpointRecord) Binding() (configurationBinding, bool) {
	return record.binding, record.binding.Valid()
}

// Adapt converts one erased runtime value through the typed endpoint adapter
// captured at authoring time.
func (record signalEndpointRecord) Adapt(value any) (any, bool) {
	if record.adapt == nil {
		return nil, false
	}
	return record.adapt(value)
}

var (
	errInvalidSignalNotification       = errors.New("semantic: invalid signal notification")
	errInvalidSignalAcceptanceSnapshot = errors.New("semantic: invalid signal acceptance snapshot")
	errInvalidSignalWatchEvent         = errors.New("semantic: invalid signal watch event")
	errInvalidSignalSourceDescription  = errors.New("semantic: invalid signal source description")
	errInvalidSignalRequirement        = errors.New("semantic: invalid signal requirement")
)

func acceptSignalSender(processID, userID, code int) SignalSender {
	return SignalSender{
		ProcessID: processID,
		UserID:    userID,
		Code:      code,
	}
}

type signalNotification struct {
	signal     Signal
	origin     SignalOrigin
	sender     *SignalSender
	observedAt time.Time
}

func acceptSignalNotification(signal Signal, origin SignalOrigin, sender *SignalSender, observedAt time.Time) (signalNotification, error) {
	if !validSignal(signal) || !origin.valid() {
		return signalNotification{}, errInvalidSignalNotification
	}
	var copiedSender *SignalSender
	if sender != nil {
		senderCopy := *sender
		copiedSender = &senderCopy
	}
	result := signalNotification{
		signal:     signal,
		origin:     origin,
		sender:     copiedSender,
		observedAt: observedAt,
	}
	return result, nil
}

func (notification signalNotification) Valid() bool {
	return validSignal(notification.signal) && notification.origin.valid()
}

// Signal returns the semantic signal identity carried by the notification.
func (notification signalNotification) Signal() Signal { return notification.signal }

func (notification signalNotification) Origin() SignalOrigin { return notification.origin }

func (notification signalNotification) ObservedAt() time.Time { return notification.observedAt }

func (notification signalNotification) Sender() *SignalSender {
	if notification.sender == nil {
		return nil
	}
	senderCopy := *notification.sender
	return &senderCopy
}

// Signal is Commandment's opaque semantic identity for one declared control signal.
// Its zero value is invalid. Equality is meaningful only for Signal values obtained
// from declaration-scoped source lookup.
type Signal struct {
	id uint64
}

func (signal Signal) valid() bool { return signal.id != 0 }

// SignalAggregation selects how one fixed signal routing window is represented
// as signal deliveries. The zero value is unknown and does not select an
// aggregation mode. Aggregation is defined by semantic eligibility windows, not
// elapsed time or scheduler turns.
type SignalAggregation uint8

func (aggregation SignalAggregation) valid() bool {
	return aggregation >= SignalAggregationEach && aggregation <= SignalAggregationLatest
}

const (
	// SignalAggregationUnknown does not select an aggregation mode.
	SignalAggregationUnknown SignalAggregation = iota
	// SignalAggregationEach delivers each occurrence in the fixed window separately.
	SignalAggregationEach
	// SignalAggregationCoalesced emits one delivery for the fixed window using
	// the first occurrence as its singular representative while preserving the
	// complete occurrence and observation-time range.
	SignalAggregationCoalesced
	// SignalAggregationLatest emits one delivery for the fixed window using
	// the last occurrence as its singular representative while preserving the
	// complete occurrence and observation-time range.
	SignalAggregationLatest
)

// SignalPhase identifies the semantic operation phase governing a routed signal
// event. Phase is assigned from authoritative operation state rather than callback
// scheduling. The zero value is SignalPhaseUnknown.
type SignalPhase uint8

func (phase SignalPhase) valid() bool {
	return phase >= SignalPhaseResolution && phase <= SignalPhaseCleanup
}

const (
	// SignalPhaseUnknown means no semantic phase has been established.
	SignalPhaseUnknown SignalPhase = iota
	// SignalPhaseResolution identifies invocation analysis before execution admission.
	SignalPhaseResolution
	// SignalPhaseExecution identifies admitted command execution.
	SignalPhaseExecution
	// SignalPhaseCleanup identifies unwinding of already-entered execution work.
	SignalPhaseCleanup
)

// SignalOrigin classifies where a signal notification originated according to the
// selected source integration. It is metadata, not signal identity or presentation
// text. The zero value is SignalOriginUnknown.
type SignalOrigin uint8

func (origin SignalOrigin) valid() bool { return origin <= SignalOriginRuntime }

const (
	// SignalOriginUnknown means the source supplied no more specific origin class.
	SignalOriginUnknown SignalOrigin = iota
	// SignalOriginTerminal identifies a terminal source origin.
	SignalOriginTerminal
	// SignalOriginProcess identifies a process-level source origin.
	SignalOriginProcess
	// SignalOriginKernel identifies a kernel-reported source origin.
	SignalOriginKernel
	// SignalOriginChild identifies an origin associated with a child process.
	SignalOriginChild
	// SignalOriginRuntime identifies an origin synthesized by a runtime integration.
	SignalOriginRuntime
)

// SignalSender carries optional source-provided sender metadata. Its fields are
// integration metadata and do not define the stable identity of the signal itself.
// A notification omits sender metadata by leaving its Sender nil.
type SignalSender struct {
	ProcessID int
	UserID    int
	Code      int
}

// SignalNotification is one source-accepted signal notification before Commandment occurrence
// selection and aggregation. Signal identifies the observed signal; Origin, Sender, and
// ObservedAt are source metadata. The governing watch revision is carried by the enclosing
// SignalSubscriptionEvent.
type SignalNotification[K comparable] struct {
	Signal     K
	Origin     SignalOrigin
	Sender     *SignalSender
	ObservedAt time.Time
}

// SignalEvent is Commandment's semantic event for one selected occurrence or one aggregated
// signal routing window. FirstOccurrence and LastOccurrence are the inclusive occurrence
// range, BatchSize records its size, and FirstObservedAt and LastObservedAt preserve the
// observation-time range. Signal, Origin, Sender, and ObservedAt are the aggregation mode's
// singular representative. Phase is the semantic phase that governed the event.
type SignalEvent struct {
	Signal Signal
	Origin SignalOrigin
	Sender *SignalSender
	Phase  SignalPhase

	ObservedAt      time.Time
	FirstObservedAt time.Time
	LastObservedAt  time.Time

	FirstOccurrence int
	LastOccurrence  int
	BatchSize       int
}

// SignalDelivery is the payload passed to a SignalTarget delivery callback. Event
// identifies the triggering semantic event; Signal is the signal value the action
// actually forwards or sends to the target and may differ from Event.Signal.
type SignalDelivery struct {
	Event  SignalEvent
	Signal Signal
}

// SignalHandler names an application callback used by a Handle signal action.
// Signal handlers for one invocation execute serially in the signal dispatcher and
// receive the handed-off live configuration together with the triggering event.
type SignalHandler[C any] struct {
	Name   string
	Handle func(context.Context, *C, SignalEvent) error
}

// SignalTarget names an application integration used for signal delivery and, when
// declared, sender matching. Only capabilities required by committed policy need be
// provided. Target callbacks execute serially within one invocation's signal
// dispatcher and receive the handed-off live configuration.
type SignalTarget[C any] struct {
	Name string

	Deliver       func(context.Context, *C, SignalDelivery) error
	MatchesSender func(context.Context, *C, SignalSender) (bool, error)
}

// SignalClass is a bit set of semantic classes attached to a supported signal
// descriptor. The zero value contains no classes.
type SignalClass uint32

const (
	// SignalClassInterrupt marks signals with interrupt semantics.
	SignalClassInterrupt SignalClass = 1 << iota
	// SignalClassTerminate marks signals with termination semantics.
	SignalClassTerminate
)

// SignalSourceDeliveryFidelity describes only how an integration turns
// upstream signal occurrences into accepted source notifications. It does not
// authorize pressure loss after a notification has been accepted.
type SignalSourceDeliveryFidelity uint8

const (
	// SignalSourceDeliveryFidelityUnknown does not declare a valid delivery fidelity.
	SignalSourceDeliveryFidelityUnknown SignalSourceDeliveryFidelity = iota

	// SignalSourceDeliveryFidelityExact preserves every upstream occurrence
	// accepted into the source protocol as a distinct notification.
	SignalSourceDeliveryFidelityExact

	// SignalSourceDeliveryFidelityCoalesced permits equivalent pending upstream
	// occurrences to combine before they become distinct accepted notifications.
	SignalSourceDeliveryFidelityCoalesced
)

// SignalSourceCapabilities is a bit set of source metadata capabilities advertised
// for a signal descriptor. The zero value advertises neither optional capability.
type SignalSourceCapabilities uint32

const (
	// SignalSourceCapabilityOrigin says accepted notifications can carry origin metadata.
	SignalSourceCapabilityOrigin SignalSourceCapabilities = 1 << iota
	// SignalSourceCapabilitySender says accepted notifications can carry sender metadata.
	SignalSourceCapabilitySender
	// SignalSourceCapabilityReraise says this signal supports source-native reraising.
	SignalSourceCapabilityReraise
)

// SignalHostCapabilities is a bit set of invocation-owned host terminal operations available to
// signal policy. The zero value provides no host operation.
type SignalHostCapabilities uint8

const (
	// SignalHostCapabilityAbort represents availability of the host Abort operation.
	SignalHostCapabilityAbort SignalHostCapabilities = 1 << iota
)

// SignalDescriptor describes one source-domain signal and its static semantic metadata.
type SignalDescriptor[K comparable] struct {
	Signal       K
	Classes      SignalClass
	Reserved     bool
	Capabilities SignalSourceCapabilities
	Fidelity     SignalSourceDeliveryFidelity
}

// SignalWatchRevision is the operation-scoped monotonic identity of a successfully
// established watched-signal state. Zero is invalid.
type SignalWatchRevision uint64

func (revision SignalWatchRevision) valid() bool { return revision != 0 }

// SignalSubscriptionEventKind selects the arm of an advanced SignalSubscriptionEvent.
type SignalSubscriptionEventKind uint8

const (
	SignalSubscriptionEventUnknown SignalSubscriptionEventKind = iota
	SignalSubscriptionEventNotification
	SignalSubscriptionEventFailure
)

// SignalSource is the ordinary passive/full-feed source carrier. Signals is snapshotted
// at declaration. Open establishes one invocation-scoped stream.
type SignalSource[K comparable] struct {
	Signals []SignalDescriptor[K]
	Open    SignalOpen[K]
	Reraise SignalReraise[K]
}

// SignalOpen establishes one invocation-scoped ordinary source stream.
type SignalOpen[K comparable] func(context.Context) (SignalStream[K], error)

// SignalReraise performs source-native re-propagation for one source-domain signal.
type SignalReraise[K comparable] func(context.Context, K) error

// SignalStream is one established ordinary source stream.
type SignalStream[K comparable] struct {
	Receive SignalReceive[K]
	Close   SignalClose
}

// SignalReceive returns the next source-local ordered notification.
type SignalReceive[K comparable] func(context.Context) (SignalNotification[K], error)

// SignalClose tears down one successfully established source activation.
type SignalClose func() error

// SignalWatchSource is the advanced source carrier for integrations that own
// fallible watched-state transitions. Its Signals catalog is static declaration data.
type SignalWatchSource[K comparable] struct {
	Signals   []SignalDescriptor[K]
	Subscribe SignalSubscribe[K]
	Reraise   SignalReraise[K]
}

// SignalSubscribe establishes the initial advanced watched state.
type SignalSubscribe[K comparable] func(context.Context, SignalWatchRevision, []K) (SignalSubscription[K], error)

// SignalSubscriptionEvent is one advanced source event under an accepted watch revision.
type SignalSubscriptionEvent[K comparable] struct {
	Kind         SignalSubscriptionEventKind
	Revision     SignalWatchRevision
	Notification SignalNotification[K]
	Err          error
}

// SignalSubscription owns one advanced source event stream and watched-state protocol.
type SignalSubscription[K comparable] struct {
	SetWatch SignalSetWatch[K]
	Events   <-chan SignalSubscriptionEvent[K]
	Close    SignalClose
}

// SignalSetWatch establishes a later advanced watched state.
type SignalSetWatch[K comparable] func(context.Context, SignalWatchRevision, []K) error

// SignalSourceOverflowError identifies terminal source-side delivery pressure.
// Capacity and Buffered are both zero when exact source occupancy is unavailable.
// Otherwise Capacity is positive and Buffered equals Capacity.
type SignalSourceOverflowError struct {
	Capacity int
	Buffered int
}

// Error returns the stable Commandment source-overflow description.
func (*SignalSourceOverflowError) Error() string { return "commandment: signal source overflow" }

// SignalBacklogCapacity is the Commandment-owned routing/endpoint pending-delivery capacity.
// It does not describe SignalSource buffering or backpressure.
const SignalBacklogCapacity = 64

// SignalHost supplies invocation-owned terminal operations that are not source semantics.
// Source registration is Definition-owned; Abort remains an explicit invocation capability.
type SignalHost struct {
	Abort func(int) error
}

// DeferSignalActions creates one dynamic action-dispatch deferral token for the active
// admitted Run represented by ctx during Execution or Cleanup. It affects only
// policies declared Deferred: source subscription, watched-state transitions,
// notification acceptance, counting, and non-deferred policies continue normally.
// The returned release function is idempotent. Passing an unrelated, stale, or
// otherwise ineligible context is programmer misuse and panics.
func DeferSignalActions(ctx context.Context) func() { return deferSignals(ctx) }

// SignalActionFailure is the structured context shared by signal action failures.
// It identifies the selected command path, policy, declared action index and action,
// triggering event, and semantic phase so callers can attribute the failure without
// parsing presentation text. Its zero value does not identify a specific failure.
type SignalActionFailure struct {
	CommandPath []string
	PolicyID    SignalPolicyID
	Index       int
	Action      SignalAction
	Event       SignalEvent
	Phase       SignalPhase
}

// SignalCancellation records the semantic signal event that canceled primary command
// execution. Cancellation is a distinct outcome rather than a generic signal-action
// failure; independent application errors, signal failures, and terminal outcomes
// remain separately observable.
//
//nolint:errname // Current canonical public API fixes SignalCancellation; rename only with synchronized API authority.
type SignalCancellation struct {
	Event SignalEvent
}

// Error returns the Commandment signal-cancellation description.
func (err *SignalCancellation) Error() string {
	return signalCancellationErrorText(err)
}

// SignalTermination records the signal selected as the operation's terminal reraised
// signal outcome. It is distinct from independent application and signal-action
// failures; the host-specific mechanism used to realize reraising is outside this value.
//
//nolint:errname // Current canonical public API fixes SignalTermination; rename only with synchronized API authority.
type SignalTermination struct {
	Signal Signal
}

// Error returns the Commandment signal-termination description.
func (err *SignalTermination) Error() string {
	return signalTerminationErrorText(err)
}

// TerminatedBy returns a SignalTermination carrying signal. It is the public error
// form used for a terminal reraised-signal outcome.
func TerminatedBy(signal Signal) error {
	return &SignalTermination{
		Signal: signal,
	}
}
