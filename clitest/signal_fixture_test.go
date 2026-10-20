package clitest

import (
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/abstractionsinc/commandment/cli"
)

const (
	fixtureSignalInterrupt = "interrupt"
	fixtureSignalMutated   = "mutated"
	fixtureSignalTerminate = "terminate"
	fixtureSignalReload    = "reload"
)

func fixtureDescriptor(signal string, reraisable bool) cli.SignalDescriptor[string] {
	capabilities := cli.SignalSourceCapabilities(0)
	if reraisable {
		capabilities |= cli.SignalSourceCapabilityReraise
	}
	return cli.SignalDescriptor[string]{
		Signal:       signal,
		Capabilities: capabilities,
		Fidelity:     cli.SignalSourceDeliveryFidelityExact,
	}
}

func TestSignalSourceFixtureOrdinaryLifecycleAndRetainedReraiseHistory(t *testing.T) {
	t.Parallel()
	wantClose := fmt.Errorf("close: %w", errors.ErrUnsupported)
	descriptor := fixtureDescriptor(fixtureSignalInterrupt, true)
	options := SignalSourceOptions[string]{
		Signals:    []cli.SignalDescriptor[string]{descriptor},
		CloseError: wantClose,
	}
	source, control := SignalSource(options)
	stream, err := source.Open(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-control.Opened():
	case <-time.After(time.Second):
		t.Fatal("ordinary source did not report Opened")
	}
	sentNotification := cli.SignalNotification[string]{Signal: fixtureSignalInterrupt}
	if err := control.Notify(sentNotification); err != nil {
		t.Fatal(err)
	}
	notification, err := stream.Receive(t.Context())
	if err != nil || notification.Signal != fixtureSignalInterrupt {
		t.Fatalf("Receive() = %#v, %v", notification, err)
	}
	if err := source.Reraise(t.Context(), fixtureSignalInterrupt); err != nil {
		t.Fatal(err)
	}
	if err := source.Reraise(t.Context(), fixtureSignalInterrupt); err != nil {
		t.Fatal(err)
	}
	reraises := control.Reraises()
	for index := range 2 {
		select {
		case got := <-reraises:
			if got != fixtureSignalInterrupt {
				t.Fatalf("Reraises()[%d] = %q", index, got)
			}
		case <-time.After(time.Second):
			t.Fatal("retained reraise history not observable")
		}
	}
	if err := stream.Close(); !errors.Is(err, wantClose) {
		t.Fatalf("Close() = %v, want %v", err, wantClose)
	}
	select {
	case <-control.Closed():
	case <-time.After(time.Second):
		t.Fatal("ordinary source did not report Closed")
	}
}

func TestSignalSourceFixtureOrdinaryFailureIsOrderedAfterNotifications(t *testing.T) {
	t.Parallel()
	wantFailure := fmt.Errorf("source failed: %w", errors.ErrUnsupported)
	descriptor := fixtureDescriptor(fixtureSignalInterrupt, false)
	options := SignalSourceOptions[string]{
		Signals: []cli.SignalDescriptor[string]{descriptor},
	}
	source, control := SignalSource(options)
	stream, err := source.Open(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	notification := cli.SignalNotification[string]{Signal: fixtureSignalInterrupt}
	if err := control.Notify(notification); err != nil {
		t.Fatal(err)
	}
	if err := control.Fail(wantFailure); err != nil {
		t.Fatal(err)
	}
	got, err := stream.Receive(t.Context())
	if err != nil || got.Signal != fixtureSignalInterrupt {
		t.Fatalf("first Receive() = %#v, %v", got, err)
	}
	if _, err := stream.Receive(t.Context()); !errors.Is(err, wantFailure) {
		t.Fatalf("second Receive() = %v, want %v", err, wantFailure)
	}
	_ = stream.Close()
}

func TestSignalWatchSourceFixtureRecordsDetachedWatchStates(t *testing.T) {
	t.Parallel()
	descriptors := []cli.SignalDescriptor[string]{
		fixtureDescriptor(fixtureSignalInterrupt, false),
		fixtureDescriptor(fixtureSignalTerminate, false),
	}
	options := SignalWatchSourceOptions[string]{Signals: descriptors}
	source, control := SignalWatchSource(options)
	initial := []string{fixtureSignalInterrupt}
	subscription, err := source.Subscribe(t.Context(), 1, initial)
	if err != nil {
		t.Fatal(err)
	}
	initial[0] = fixtureSignalMutated
	next := []string{fixtureSignalTerminate}
	if err := subscription.SetWatch(t.Context(), 2, next); err != nil {
		t.Fatal(err)
	}
	next[0] = fixtureSignalMutated
	changes := control.WatchStates()
	want := []SignalWatchState[string]{
		{Revision: 1, Signals: []string{fixtureSignalInterrupt}},
		{Revision: 2, Signals: []string{fixtureSignalTerminate}},
	}
	for index, expected := range want {
		select {
		case got := <-changes:
			if got.Revision != expected.Revision || !slices.Equal(got.Signals, expected.Signals) {
				t.Fatalf("WatchStates()[%d] = %#v, want %#v", index, got, expected)
			}
			got.Signals[0] = "caller-mutated"
		case <-time.After(time.Second):
			t.Fatal("watch change not observable")
		}
	}
	if err := subscription.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-control.Closed():
	case <-time.After(time.Second):
		t.Fatal("advanced source did not report Closed")
	}
}

func TestSignalWatchSourceFixtureEventsAndReraiseUseNativeDomain(t *testing.T) {
	t.Parallel()
	descriptor := fixtureDescriptor(fixtureSignalTerminate, true)
	options := SignalWatchSourceOptions[string]{
		Signals: []cli.SignalDescriptor[string]{descriptor},
	}
	source, control := SignalWatchSource(options)
	initial := []string{fixtureSignalTerminate}
	subscription, err := source.Subscribe(t.Context(), 1, initial)
	if err != nil {
		t.Fatal(err)
	}
	notification := cli.SignalNotification[string]{Signal: fixtureSignalTerminate}
	if err := control.Notify(notification); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-subscription.Events:
		if event.Kind != cli.SignalSubscriptionEventNotification || event.Revision != 1 ||
			event.Notification.Signal != fixtureSignalTerminate {
			t.Fatalf("event = %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("advanced notification not delivered")
	}
	if err := source.Reraise(t.Context(), fixtureSignalTerminate); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-control.Reraises():
		if got != fixtureSignalTerminate {
			t.Fatalf("Reraises() = %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("advanced reraise not observed")
	}
	if err := subscription.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSignalSourceFixtureConstructorsSnapshotDescriptors(t *testing.T) {
	t.Parallel()
	descriptors := []cli.SignalDescriptor[string]{fixtureDescriptor(fixtureSignalInterrupt, false)}
	ordinaryOptions := SignalSourceOptions[string]{Signals: descriptors}
	advancedOptions := SignalWatchSourceOptions[string]{Signals: descriptors}
	ordinary, _ := SignalSource(ordinaryOptions)
	advanced, _ := SignalWatchSource(advancedOptions)
	descriptors[0].Signal = fixtureSignalMutated
	if ordinary.Signals[0].Signal != fixtureSignalInterrupt || advanced.Signals[0].Signal != fixtureSignalInterrupt {
		t.Fatalf("fixture constructors did not snapshot descriptors: %#v %#v", ordinary.Signals, advanced.Signals)
	}
}

func TestSignalSourceFixtureInvalidObservationMethodsPanic(t *testing.T) {
	t.Parallel()

	operations := []struct {
		name string
		run  func()
	}{
		{name: "ordinary opened", run: func() { var control SignalSourceControl[string]; _ = control.Opened() }},
		{name: "ordinary reraises", run: func() { var control SignalSourceControl[string]; _ = control.Reraises() }},
		{name: "ordinary closed", run: func() { var control SignalSourceControl[string]; _ = control.Closed() }},
		{name: "advanced watch changes", run: func() { var control SignalWatchSourceControl[string]; _ = control.WatchStates() }},
		{name: "advanced reraises", run: func() { var control SignalWatchSourceControl[string]; _ = control.Reraises() }},
		{name: "advanced closed", run: func() { var control SignalWatchSourceControl[string]; _ = control.Closed() }},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				recovered := recover()
				recoveredErr, ok := recovered.(error)
				if !ok || !errors.Is(recoveredErr, errInvalidSignalSourceFixture) {
					t.Fatalf("panic = %#v, want %v", recovered, errInvalidSignalSourceFixture)
				}
			}()
			operation.run()
		})
	}
}

func TestSignalSourceFixtureDescriptorSnapshotOwnsReraiseCapability(t *testing.T) {
	t.Parallel()

	for _, reraisable := range []bool{false, true} {
		descriptors := []cli.SignalDescriptor[string]{fixtureDescriptor(fixtureSignalInterrupt, reraisable)}
		ordinaryOptions := SignalSourceOptions[string]{Signals: descriptors}
		advancedOptions := SignalWatchSourceOptions[string]{Signals: descriptors}
		ordinary, _ := SignalSource(ordinaryOptions)
		advanced, _ := SignalWatchSource(advancedOptions)

		want := ordinary.Signals[0].Capabilities&cli.SignalSourceCapabilityReraise != 0
		if (ordinary.Reraise != nil) != want {
			t.Fatalf("ordinary Reraise presence = %t, snapshot capability = %t", ordinary.Reraise != nil, want)
		}
		want = advanced.Signals[0].Capabilities&cli.SignalSourceCapabilityReraise != 0
		if (advanced.Reraise != nil) != want {
			t.Fatalf("advanced Reraise presence = %t, snapshot capability = %t", advanced.Reraise != nil, want)
		}
	}
}
