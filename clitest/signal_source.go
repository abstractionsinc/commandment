package clitest

import (
	"context"
	"errors"
	"sync"

	"github.com/abstractionsinc/commandment/cli"
)

// SignalSourceOptions configures the ordinary deterministic Signal source fixture.
// Signals is snapshotted by SignalSource.
type SignalSourceOptions[K comparable] struct {
	Signals      []cli.SignalDescriptor[K]
	OpenError    error
	CloseError   error
	ReraiseError error
}

// SignalSourceControl controls and observes one ordinary Signal source fixture.
// Its zero value is invalid; obtain it from SignalSource.
type SignalSourceControl[K comparable] struct {
	state *ordinarySignalFixtureState[K]
}

// SignalWatchSourceOptions configures the advanced deterministic Signal source fixture.
// Signals is snapshotted by SignalWatchSource.
type SignalWatchSourceOptions[K comparable] struct {
	Signals        []cli.SignalDescriptor[K]
	SubscribeError error
	SetWatchError  error
	CloseError     error
	ReraiseError   error
}

// SignalWatchSourceControl controls and observes one advanced Signal source fixture.
// Its zero value is invalid; obtain it from SignalWatchSource.
type SignalWatchSourceControl[K comparable] struct {
	state *advancedSignalFixtureState[K]
}

// SignalWatchState is a detached complete observation of one successfully accepted advanced watched state.
type SignalWatchState[K comparable] struct {
	Revision cli.SignalWatchRevision
	Signals  []K
}

// SignalSource returns an ordinary production carrier plus its isolated test control.
// Definition code should receive only the returned cli.SignalSource value.
func SignalSource[K comparable](options SignalSourceOptions[K]) (cli.SignalSource[K], *SignalSourceControl[K]) {
	state := &ordinarySignalFixtureState[K]{
		openErr:     options.OpenError,
		closeErr:    options.CloseError,
		reraiseErr:  options.ReraiseError,
		wake:        make(chan struct{}, 1),
		opened:      make(chan struct{}),
		closed:      make(chan struct{}),
		reraiseWake: make(chan struct{}, 1),
	}
	control := &SignalSourceControl[K]{
		state: state,
	}
	signals := append([]cli.SignalDescriptor[K](nil), options.Signals...)
	source := cli.SignalSource[K]{
		Signals: signals,
		Open:    state.open,
	}
	if signalDescriptorsReraisable(signals) {
		source.Reraise = state.reraise
	}
	return source, control
}

// SignalWatchSource returns an advanced production carrier plus its isolated test control.
func SignalWatchSource[K comparable](options SignalWatchSourceOptions[K]) (cli.SignalWatchSource[K], *SignalWatchSourceControl[K]) {
	state := &advancedSignalFixtureState[K]{
		subscribeErr: options.SubscribeError,
		setWatchErr:  options.SetWatchError,
		closeErr:     options.CloseError,
		reraiseErr:   options.ReraiseError,
		eventWake:    make(chan struct{}, 1),
		watchWake:    make(chan struct{}, 1),
		reraiseWake:  make(chan struct{}, 1),
		closed:       make(chan struct{}),
	}
	control := &SignalWatchSourceControl[K]{
		state: state,
	}
	signals := append([]cli.SignalDescriptor[K](nil), options.Signals...)
	source := cli.SignalWatchSource[K]{
		Signals:   signals,
		Subscribe: state.subscribe,
	}
	if signalDescriptorsReraisable(signals) {
		source.Reraise = state.reraise
	}
	return source, control
}

// Opened closes after the fixture's Open callback succeeds.
func (control *SignalSourceControl[K]) Opened() <-chan struct{} {
	state, err := control.ordinaryState()
	if err != nil {
		panic(err)
	}
	return state.opened
}

// Notify appends one source-local notification for the established stream.
func (control *SignalSourceControl[K]) Notify(notification cli.SignalNotification[K]) error {
	state, err := control.ordinaryState()
	if err != nil {
		return err
	}
	record := ordinarySignalRecord[K]{
		notification: cloneSignalNotification(notification),
	}
	return state.enqueue(record)
}

// Fail terminates the live source with failure after all previously accepted notifications.
func (control *SignalSourceControl[K]) Fail(failure error) error {
	if failure == nil {
		return errSignalSourceFailureRequired
	}
	state, err := control.ordinaryState()
	if err != nil {
		return err
	}
	record := ordinarySignalRecord[K]{
		err: failure,
	}
	return state.enqueue(record)
}

// Reraises returns a retained-history observation stream of source-native reraises.
// Observation never controls or blocks the production Reraise callback.
func (control *SignalSourceControl[K]) Reraises() <-chan K {
	state, err := control.ordinaryState()
	if err != nil {
		panic(err)
	}
	return state.reraises()
}

// Closed closes after the established stream's Close callback is invoked.
func (control *SignalSourceControl[K]) Closed() <-chan struct{} {
	state, err := control.ordinaryState()
	if err != nil {
		panic(err)
	}
	return state.closed
}

func (control *SignalSourceControl[K]) ordinaryState() (*ordinarySignalFixtureState[K], error) {
	if control == nil || control.state == nil {
		return nil, errInvalidSignalSourceFixture
	}
	return control.state, nil
}

// Notify appends a notification under the currently accepted watch revision.
func (control *SignalWatchSourceControl[K]) Notify(notification cli.SignalNotification[K]) error {
	state, err := control.advancedState()
	if err != nil {
		return err
	}
	event := cli.SignalSubscriptionEvent[K]{
		Kind:         cli.SignalSubscriptionEventNotification,
		Notification: cloneSignalNotification(notification),
	}
	return state.enqueueEvent(event)
}

// Fail appends the terminal source failure under the currently accepted watch revision.
func (control *SignalWatchSourceControl[K]) Fail(failure error) error {
	if failure == nil {
		return errSignalSourceFailureRequired
	}
	state, err := control.advancedState()
	if err != nil {
		return err
	}
	event := cli.SignalSubscriptionEvent[K]{
		Kind: cli.SignalSubscriptionEventFailure,
		Err:  failure,
	}
	return state.enqueueEvent(event)
}

// WatchStates returns retained detached snapshots of successful Subscribe/SetWatch transitions.
func (control *SignalWatchSourceControl[K]) WatchStates() <-chan SignalWatchState[K] {
	state, err := control.advancedState()
	if err != nil {
		panic(err)
	}
	return state.watchStates()
}

// Reraises returns retained-history observations of source-native reraises.
func (control *SignalWatchSourceControl[K]) Reraises() <-chan K {
	state, err := control.advancedState()
	if err != nil {
		panic(err)
	}
	return state.reraises()
}

// Closed closes after the advanced subscription Close callback is invoked.
func (control *SignalWatchSourceControl[K]) Closed() <-chan struct{} {
	state, err := control.advancedState()
	if err != nil {
		panic(err)
	}
	return state.closed
}

func (control *SignalWatchSourceControl[K]) advancedState() (*advancedSignalFixtureState[K], error) {
	if control == nil || control.state == nil {
		return nil, errInvalidSignalSourceFixture
	}
	return control.state, nil
}

type signalFixtureContextError struct {
	operation string
	cause     error
}

var (
	errInvalidSignalSourceFixture       = errors.New("clitest: invalid signal source fixture")
	errSignalSourceAlreadyOpened        = errors.New("clitest: signal source fixture already opened")
	errSignalSourceAlreadySubscribed    = errors.New("clitest: signal source fixture already subscribed")
	errSignalSourceInvalidInitialWatch  = errors.New("clitest: signal source fixture initial watch is invalid")
	errSignalSourceInvalidWatchRevision = errors.New("clitest: signal source fixture watch revision is invalid")
	errSignalSourceClosed               = errors.New("clitest: signal source fixture is closed")
	errSignalSourceTerminal             = errors.New("clitest: signal source fixture is terminal")
	errSignalSourceNotEstablished       = errors.New("clitest: signal source fixture is not established")
	errSignalSourceFailureRequired      = errors.New("clitest: signal source fixture failure is nil")
)

type ordinarySignalRecord[K comparable] struct {
	notification cli.SignalNotification[K]
	err          error
}

type ordinarySignalFixtureState[K comparable] struct {
	mu sync.Mutex

	openErr     error
	closeErr    error
	reraiseErr  error
	established bool
	closedState bool
	terminal    bool
	queue       []ordinarySignalRecord[K]
	wake        chan struct{}
	opened      chan struct{}
	closed      chan struct{}

	reraiseHistory []K
	reraiseWake    chan struct{}
	reraiseOut     chan K
	reraiseStarted bool
}

type advancedSignalFixtureState[K comparable] struct {
	mu sync.Mutex

	subscribeErr error
	setWatchErr  error
	closeErr     error
	reraiseErr   error
	subscribed   bool
	closedState  bool
	terminal     bool
	revision     cli.SignalWatchRevision

	eventHistory []cli.SignalSubscriptionEvent[K]
	eventWake    chan struct{}
	events       chan cli.SignalSubscriptionEvent[K]

	watchHistory []SignalWatchState[K]
	watchWake    chan struct{}
	watchOut     chan SignalWatchState[K]
	watchStarted bool

	reraiseHistory []K
	reraiseWake    chan struct{}
	reraiseOut     chan K
	reraiseStarted bool

	closed chan struct{}
}

func signalFixtureWake(wake chan struct{}) {
	select {
	case wake <- struct{}{}:
	default:
	}
}

func cloneSignalSubscriptionEvent[K comparable](input cli.SignalSubscriptionEvent[K]) cli.SignalSubscriptionEvent[K] {
	result := input
	result.Notification = cloneSignalNotification(input.Notification)
	return result
}

func cloneSignalNotification[K comparable](input cli.SignalNotification[K]) cli.SignalNotification[K] {
	result := input
	if input.Sender != nil {
		result.Sender = new(*input.Sender)
	}
	return result
}

func cloneSignalWatchState[K comparable](input SignalWatchState[K]) SignalWatchState[K] {
	result := SignalWatchState[K]{
		Revision: input.Revision,
		Signals:  append([]K(nil), input.Signals...),
	}
	return result
}

func signalDescriptorsReraisable[K comparable](descriptors []cli.SignalDescriptor[K]) bool {
	for _, descriptor := range descriptors {
		if descriptor.Capabilities&cli.SignalSourceCapabilityReraise != 0 {
			return true
		}
	}
	return false
}

func (err signalFixtureContextError) Error() string {
	return err.operation + " signal source fixture: " + err.cause.Error()
}

func (err signalFixtureContextError) Unwrap() error { return err.cause }

func (state *ordinarySignalFixtureState[K]) open(ctx context.Context) (cli.SignalStream[K], error) {
	var stream cli.SignalStream[K]
	if err := context.Cause(ctx); err != nil {
		failure := signalFixtureContextError{
			operation: "open",
			cause:     err,
		}
		return cli.SignalStream[K]{}, failure
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.established {
		return cli.SignalStream[K]{}, errSignalSourceAlreadyOpened
	}
	if state.openErr != nil {
		return cli.SignalStream[K]{}, state.openErr
	}
	state.established = true
	close(state.opened)
	stream = cli.SignalStream[K]{
		Receive: state.receive,
		Close:   state.close,
	}
	return stream, nil
}

func (state *ordinarySignalFixtureState[K]) receive(ctx context.Context) (cli.SignalNotification[K], error) {
	for {
		state.mu.Lock()
		if len(state.queue) != 0 {
			record := state.queue[0]
			clear(state.queue[:1])
			state.queue = state.queue[1:]
			state.mu.Unlock()
			if record.err != nil {
				return cli.SignalNotification[K]{}, record.err
			}
			return cloneSignalNotification(record.notification), nil
		}
		if state.closedState {
			state.mu.Unlock()
			return cli.SignalNotification[K]{}, context.Canceled
		}
		if state.terminal {
			state.mu.Unlock()
			return cli.SignalNotification[K]{}, errSignalSourceTerminal
		}
		wake := state.wake
		state.mu.Unlock()
		select {
		case <-ctx.Done():
			failure := signalFixtureContextError{
				operation: "receive",
				cause:     context.Cause(ctx),
			}
			return cli.SignalNotification[K]{}, failure
		case <-wake:
		}
	}
}

func (state *ordinarySignalFixtureState[K]) enqueue(record ordinarySignalRecord[K]) error {
	state.mu.Lock()
	defer state.mu.Unlock()
	if !state.established {
		return errSignalSourceNotEstablished
	}
	if state.closedState {
		return errSignalSourceClosed
	}
	if state.terminal {
		return errSignalSourceTerminal
	}
	state.queue = append(state.queue, record)
	if record.err != nil {
		state.terminal = true
	}
	signalFixtureWake(state.wake)
	return nil
}

func (state *ordinarySignalFixtureState[K]) close() error {
	state.mu.Lock()
	if !state.closedState {
		state.closedState = true
		close(state.closed)
		signalFixtureWake(state.wake)
		signalFixtureWake(state.reraiseWake)
	}
	err := state.closeErr
	state.mu.Unlock()
	return err
}

func (state *ordinarySignalFixtureState[K]) reraise(ctx context.Context, signal K) error {
	if err := context.Cause(ctx); err != nil {
		failure := signalFixtureContextError{
			operation: "reraise",
			cause:     err,
		}
		return failure
	}
	state.mu.Lock()
	state.reraiseHistory = append(state.reraiseHistory, signal)
	err := state.reraiseErr
	signalFixtureWake(state.reraiseWake)
	state.mu.Unlock()
	return err
}

func (state *ordinarySignalFixtureState[K]) reraises() <-chan K {
	state.mu.Lock()
	if !state.reraiseStarted {
		state.reraiseStarted = true
		state.reraiseOut = make(chan K)
		go state.runReraiseStream()
	}
	result := state.reraiseOut
	state.mu.Unlock()
	return result
}

func (state *ordinarySignalFixtureState[K]) runReraiseStream() {
	defer close(state.reraiseOut)
	for index := 0; ; {
		state.mu.Lock()
		if index < len(state.reraiseHistory) {
			signal := state.reraiseHistory[index]
			index++
			out := state.reraiseOut
			state.mu.Unlock()
			out <- signal
			continue
		}
		if state.closedState {
			state.mu.Unlock()
			return
		}
		wake := state.reraiseWake
		state.mu.Unlock()
		<-wake
	}
}

func (state *advancedSignalFixtureState[K]) subscribe(ctx context.Context, revision cli.SignalWatchRevision, signals []K) (cli.SignalSubscription[K], error) {
	var subscription cli.SignalSubscription[K]
	if err := context.Cause(ctx); err != nil {
		failure := signalFixtureContextError{
			operation: "subscribe",
			cause:     err,
		}
		return cli.SignalSubscription[K]{}, failure
	}
	if revision == 0 {
		return cli.SignalSubscription[K]{}, errSignalSourceInvalidInitialWatch
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.subscribed {
		return cli.SignalSubscription[K]{}, errSignalSourceAlreadySubscribed
	}
	if state.subscribeErr != nil {
		return cli.SignalSubscription[K]{}, state.subscribeErr
	}
	state.subscribed = true
	state.revision = revision
	state.events = make(chan cli.SignalSubscriptionEvent[K])
	state.recordWatchLocked(revision, signals)
	go state.runEventStream()
	subscription = cli.SignalSubscription[K]{
		SetWatch: state.setWatch,
		Events:   state.events,
		Close:    state.close,
	}
	return subscription, nil
}

func (state *advancedSignalFixtureState[K]) setWatch(ctx context.Context, revision cli.SignalWatchRevision, signals []K) error {
	if err := context.Cause(ctx); err != nil {
		failure := signalFixtureContextError{
			operation: "set watch",
			cause:     err,
		}
		return failure
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if !state.subscribed {
		return errSignalSourceNotEstablished
	}
	if state.closedState {
		return errSignalSourceClosed
	}
	if state.terminal {
		return errSignalSourceTerminal
	}
	if revision == 0 || revision <= state.revision {
		return errSignalSourceInvalidWatchRevision
	}
	if state.setWatchErr != nil {
		return state.setWatchErr
	}
	state.revision = revision
	state.recordWatchLocked(revision, signals)
	return nil
}

func (state *advancedSignalFixtureState[K]) enqueueEvent(event cli.SignalSubscriptionEvent[K]) error {
	state.mu.Lock()
	defer state.mu.Unlock()
	if !state.subscribed {
		return errSignalSourceNotEstablished
	}
	if state.closedState {
		return errSignalSourceClosed
	}
	if state.terminal {
		return errSignalSourceTerminal
	}
	event.Revision = state.revision
	event.Notification = cloneSignalNotification(event.Notification)
	state.eventHistory = append(state.eventHistory, event)
	if event.Kind == cli.SignalSubscriptionEventFailure {
		state.terminal = true
	}
	signalFixtureWake(state.eventWake)
	return nil
}

func (state *advancedSignalFixtureState[K]) runEventStream() {
	defer close(state.events)
	index := 0
	for {
		state.mu.Lock()
		if index < len(state.eventHistory) {
			event := cloneSignalSubscriptionEvent(state.eventHistory[index])
			index++
			out := state.events
			state.mu.Unlock()
			out <- event
			continue
		}
		if state.closedState || state.terminal {
			state.mu.Unlock()
			return
		}
		wake := state.eventWake
		state.mu.Unlock()
		<-wake
	}
}

func (state *advancedSignalFixtureState[K]) close() error {
	state.mu.Lock()
	if !state.closedState {
		state.closedState = true
		close(state.closed)
		signalFixtureWake(state.eventWake)
		signalFixtureWake(state.watchWake)
		signalFixtureWake(state.reraiseWake)
	}
	err := state.closeErr
	state.mu.Unlock()
	return err
}

func (state *advancedSignalFixtureState[K]) recordWatchLocked(revision cli.SignalWatchRevision, signals []K) {
	watchState := SignalWatchState[K]{
		Revision: revision,
		Signals:  append([]K(nil), signals...),
	}
	state.watchHistory = append(state.watchHistory, watchState)
	signalFixtureWake(state.watchWake)
}

func (state *advancedSignalFixtureState[K]) watchStates() <-chan SignalWatchState[K] {
	state.mu.Lock()
	if !state.watchStarted {
		state.watchStarted = true
		state.watchOut = make(chan SignalWatchState[K])
		go state.runWatchStream()
	}
	result := state.watchOut
	state.mu.Unlock()
	return result
}

func (state *advancedSignalFixtureState[K]) runWatchStream() {
	defer close(state.watchOut)
	index := 0
	for {
		state.mu.Lock()
		if index < len(state.watchHistory) {
			watchState := cloneSignalWatchState(state.watchHistory[index])
			index++
			out := state.watchOut
			state.mu.Unlock()
			out <- watchState
			continue
		}
		if state.closedState {
			state.mu.Unlock()
			return
		}
		wake := state.watchWake
		state.mu.Unlock()
		<-wake
	}
}

func (state *advancedSignalFixtureState[K]) reraise(ctx context.Context, signal K) error {
	if err := context.Cause(ctx); err != nil {
		failure := signalFixtureContextError{
			operation: "reraise",
			cause:     err,
		}
		return failure
	}
	state.mu.Lock()
	state.reraiseHistory = append(state.reraiseHistory, signal)
	err := state.reraiseErr
	signalFixtureWake(state.reraiseWake)
	state.mu.Unlock()
	return err
}

func (state *advancedSignalFixtureState[K]) reraises() <-chan K {
	state.mu.Lock()
	if !state.reraiseStarted {
		state.reraiseStarted = true
		state.reraiseOut = make(chan K)
		go state.runReraiseStream()
	}
	result := state.reraiseOut
	state.mu.Unlock()
	return result
}

func (state *advancedSignalFixtureState[K]) runReraiseStream() {
	defer close(state.reraiseOut)
	for index := 0; ; {
		state.mu.Lock()
		if index < len(state.reraiseHistory) {
			signal := state.reraiseHistory[index]
			index++
			out := state.reraiseOut
			state.mu.Unlock()
			out <- signal
			continue
		}
		if state.closedState {
			state.mu.Unlock()
			return
		}
		wake := state.reraiseWake
		state.mu.Unlock()
		<-wake
	}
}
