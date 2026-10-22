package cli

import (
	"context"
	"os"
	"os/signal"
	"slices"
	"sync"
	"time"
)

const (
	processSignalEventCapacity = 64
	processSignalRawCapacity   = 64
)

type processSignalError struct {
	operation string
	cause     error
}

func (err *processSignalError) Error() string { return err.operation + ": " + err.cause.Error() }

func (err *processSignalError) Unwrap() error { return err.cause }

func processSignalSourceValue() SignalWatchSource[os.Signal] {
	platform := processPlatformSignalDescriptors()
	descriptors := make([]SignalDescriptor[os.Signal], 1, 1+len(platform))
	descriptors[0] = SignalDescriptor[os.Signal]{
		Signal:       os.Interrupt,
		Classes:      SignalClassInterrupt,
		Capabilities: SignalSourceCapabilityOrigin | SignalSourceCapabilityReraise,
		Fidelity:     SignalSourceDeliveryFidelityCoalesced,
	}
	descriptors = append(descriptors, platform...)
	signalWatchSource := SignalWatchSource[os.Signal]{
		Signals:   descriptors,
		Subscribe: processSignalSubscribe,
		Reraise: func(_ context.Context, signal os.Signal) error {
			return processSignalReraise(signal)
		},
	}

	return signalWatchSource
}

func registerProcessSignalSource(draft *applicationDraft) (signalSourceScope[os.Signal], bool) {
	if draft == nil {
		return signalSourceScope[os.Signal]{}, false
	}
	before := len(draft.signalSources)
	scope := registerSignalSource(draft, nil, processSignalSourceValue().signalSourceDefinition())
	if !scope.valid || len(draft.signalSources) != before+1 {
		return scope, false
	}
	draft.signalSources[before].process = true
	return scope, true
}

func processSignalSubscribe(ctx context.Context, revision SignalWatchRevision, watched []os.Signal) (SignalSubscription[os.Signal], error) {
	if ctx == nil || revision == 0 {
		return SignalSubscription[os.Signal]{}, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		failure := &processSignalError{
			operation: "process signal subscription context",
			cause:     err,
		}
		return SignalSubscription[os.Signal]{}, failure
	}
	subscription := &processSignalSubscription{
		revision: revision,
		events:   make(chan SignalSubscriptionEvent[os.Signal], processSignalEventCapacity),
		raw:      make(chan os.Signal, processSignalRawCapacity),
		done:     make(chan struct{}),
		stop:     make(chan struct{}),
	}
	subscription.setWatchLocked(watched)
	go subscription.run()
	signalSubscription := SignalSubscription[os.Signal]{
		SetWatch: subscription.SetWatch,
		Events:   subscription.events,
		Close:    subscription.Close,
	}
	return signalSubscription, nil
}

type processSignalSubscription struct {
	mu sync.Mutex

	revision SignalWatchRevision
	watched  []os.Signal
	raw      chan os.Signal
	events   chan SignalSubscriptionEvent[os.Signal]
	done     chan struct{}
	stop     chan struct{}

	closed    bool
	closeOnce sync.Once
}

// SetWatch applies the SignalSubscription watched-state transition contract to the
// process adapter. A successful call establishes the supplied revision boundary.
func (subscription *processSignalSubscription) SetWatch(ctx context.Context, revision SignalWatchRevision, watched []os.Signal) error {
	if ctx == nil || revision == 0 {
		return context.Canceled
	}
	subscription.mu.Lock()
	defer subscription.mu.Unlock()
	if subscription.closed {
		return context.Canceled
	}
	if err := ctx.Err(); err != nil {
		failure := &processSignalError{
			operation: "set process signal watch context",
			cause:     err,
		}
		return failure
	}
	subscription.revision = revision
	subscription.setWatchLocked(watched)
	return nil
}

func (subscription *processSignalSubscription) setWatchLocked(watched []os.Signal) {
	signal.Stop(subscription.raw)
	subscription.watched = append(subscription.watched[:0], watched...)
	if len(watched) != 0 {
		signal.Notify(subscription.raw, watched...)
	}
}

// run owns the process adapter's source event loop for one subscription and
// closes the ordered event stream when that subscription stops.
func (subscription *processSignalSubscription) run() {
	defer close(subscription.events)
	defer close(subscription.done)
	for {
		select {
		case <-subscription.stop:
			return
		case signal := <-subscription.raw:
			subscription.mu.Lock()
			if subscription.closed {
				subscription.mu.Unlock()
				return
			}
			revision := subscription.revision
			watched := processSignalIsWatched(subscription.watched, signal)
			subscription.mu.Unlock()
			if !watched {
				continue
			}
			event := SignalSubscriptionEvent[os.Signal]{
				Kind:     SignalSubscriptionEventNotification,
				Revision: revision,
				Notification: SignalNotification[os.Signal]{
					Signal:     signal,
					Origin:     SignalOriginProcess,
					ObservedAt: time.Now(),
				},
			}
			select {
			case subscription.events <- event:
			case <-subscription.stop:
				return
			}
		}
	}
}

func processSignalIsWatched(watched []os.Signal, signal os.Signal) bool {
	return slices.Contains(watched, signal)
}

// Close ends the process adapter subscription according to SignalSubscription.Close.
// It is idempotent and leaves Events as the same closed channel.
func (subscription *processSignalSubscription) Close() error {
	if subscription == nil {
		return nil
	}
	subscription.closeOnce.Do(func() {
		subscription.mu.Lock()
		subscription.closed = true
		signal.Stop(subscription.raw)
		close(subscription.stop)
		subscription.mu.Unlock()
	})
	<-subscription.done
	return nil
}

func processSignalAbort(code int) error {
	os.Exit(code)
	return nil
}
