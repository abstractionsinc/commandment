package cli

import (
	"context"
	"errors"
)

const signalClassMask = SignalClassInterrupt | SignalClassTerminate

const signalSourceCapabilityMask = SignalSourceCapabilityOrigin |
	SignalSourceCapabilitySender |
	SignalSourceCapabilityReraise

type signalDescriptor struct {
	signal       Signal
	classes      SignalClass
	reserved     bool
	capabilities SignalSourceCapabilities
	fidelity     SignalSourceDeliveryFidelity
}

func acceptSignalDescriptor(signal Signal, classes SignalClass, reserved bool, capabilities SignalSourceCapabilities, fidelity SignalSourceDeliveryFidelity) (signalDescriptor, error) {
	if !validSignal(signal) || classes&^signalClassMask != 0 ||
		capabilities&^signalSourceCapabilityMask != 0 ||
		(fidelity != SignalSourceDeliveryFidelityExact && fidelity != SignalSourceDeliveryFidelityCoalesced) {
		return signalDescriptor{}, errInvalidSignalSourceDescription
	}
	result := signalDescriptor{
		signal:       signal,
		classes:      classes,
		reserved:     reserved,
		capabilities: capabilities,
		fidelity:     fidelity,
	}
	return result, nil
}

func (descriptor signalDescriptor) Valid() bool {
	return validSignal(descriptor.signal) && descriptor.classes&^signalClassMask == 0 &&
		descriptor.capabilities&^signalSourceCapabilityMask == 0 &&
		(descriptor.fidelity == SignalSourceDeliveryFidelityExact ||
			descriptor.fidelity == SignalSourceDeliveryFidelityCoalesced)
}

// Signal returns the semantic signal identity described by the source entry.
func (descriptor signalDescriptor) Signal() Signal { return descriptor.signal }

func (descriptor signalDescriptor) Classes() SignalClass { return descriptor.classes }

func (descriptor signalDescriptor) Reserved() bool { return descriptor.reserved }

type semanticSignalSourceDescription struct {
	descriptors []signalDescriptor
}

func acceptSignalSourceDescription(descriptors []signalDescriptor) (semanticSignalSourceDescription, error) {
	copied := append([]signalDescriptor(nil), descriptors...)
	for i, descriptor := range copied {
		if !descriptor.Valid() {
			return semanticSignalSourceDescription{}, errInvalidSignalSourceDescription
		}
		for j := range i {
			if copied[j].signal == descriptor.signal {
				return semanticSignalSourceDescription{}, errInvalidSignalSourceDescription
			}
		}
	}
	return semanticSignalSourceDescription{
		descriptors: copied,
	}, nil
}

func (description semanticSignalSourceDescription) Valid() bool {
	for i, descriptor := range description.descriptors {
		if !descriptor.Valid() {
			return false
		}
		for j := range i {
			if description.descriptors[j].signal == descriptor.signal {
				return false
			}
		}
	}
	return true
}

func (description semanticSignalSourceDescription) Descriptors() []signalDescriptor {
	return append([]signalDescriptor(nil), description.descriptors...)
}

// validSignal reports whether a Commandment semantic signal identity is valid.
func validSignal(signal Signal) bool { return signal.valid() }

const signalHostCapabilityMask = SignalHostCapabilityAbort

// signalRequirement is the already-derived effective source/host requirement
// for one activation frontier. Policy interpretation owns derivation; this value
// only carries the settled signals and capability bits into compatibility
// validation.
type signalRequirement struct {
	signals            []Signal
	sourceCapabilities SignalSourceCapabilities
	hostCapabilities   SignalHostCapabilities
}

func acceptSignalRequirement(signals []Signal, sourceCapabilities SignalSourceCapabilities, hostCapabilities SignalHostCapabilities) (signalRequirement, error) {
	if sourceCapabilities&^signalSourceCapabilityMask != 0 || hostCapabilities&^signalHostCapabilityMask != 0 {
		return signalRequirement{}, errInvalidSignalRequirement
	}
	copied := append([]Signal(nil), signals...)
	for i, signal := range copied {
		if !validSignal(signal) {
			return signalRequirement{}, errInvalidSignalRequirement
		}
		for j := range i {
			if copied[j] == signal {
				return signalRequirement{}, errInvalidSignalRequirement
			}
		}
	}
	result := signalRequirement{
		signals:            copied,
		sourceCapabilities: sourceCapabilities,
		hostCapabilities:   hostCapabilities,
	}
	return result, nil
}

// Signals returns the required semantic signal identities.
func (requirement signalRequirement) Signals() []Signal {
	return append([]Signal(nil), requirement.signals...)
}

type signalCompatibility struct {
	resolvedSignals           []Signal
	missingSignals            []Signal
	missingSourceCapabilities SignalSourceCapabilities
	availableHostCapabilities SignalHostCapabilities
	missingHostCapabilities   SignalHostCapabilities
}

func evaluateSignalCompatibility(description semanticSignalSourceDescription, requirement signalRequirement, availableHostCapabilities SignalHostCapabilities) (signalCompatibility, error) {
	if !description.Valid() || availableHostCapabilities&^signalHostCapabilityMask != 0 {
		return signalCompatibility{}, errInvalidSignalSourceDescription
	}
	bySignal := make(map[Signal]signalDescriptor, len(description.descriptors))
	for _, descriptor := range description.descriptors {
		bySignal[descriptor.signal] = descriptor
	}
	compatibility := signalCompatibility{
		availableHostCapabilities: availableHostCapabilities,
	}
	for _, signal := range requirement.signals {
		descriptor, ok := bySignal[signal]
		if !ok {
			compatibility.missingSignals = append(compatibility.missingSignals, signal)
			continue
		}
		compatibility.resolvedSignals = append(compatibility.resolvedSignals, signal)
		compatibility.missingSourceCapabilities |= requirement.sourceCapabilities &^ descriptor.capabilities
	}
	compatibility.missingHostCapabilities = requirement.hostCapabilities &^ availableHostCapabilities
	return compatibility, nil
}

// ResolvedSignals returns required signals supplied by the selected sources.
func (compatibility signalCompatibility) ResolvedSignals() []Signal {
	return append([]Signal(nil), compatibility.resolvedSignals...)
}

// MissingSignals returns required signals not supplied by the selected sources.
func (compatibility signalCompatibility) MissingSignals() []Signal {
	return append([]Signal(nil), compatibility.missingSignals...)
}

func (compatibility signalCompatibility) MissingSourceCapabilities() SignalSourceCapabilities {
	return compatibility.missingSourceCapabilities
}

func (compatibility signalCompatibility) MissingHostCapabilities() SignalHostCapabilities {
	return compatibility.missingHostCapabilities
}

var errInvalidSignalSourceDeclaration = errors.New("cli: invalid signal source declaration")

type signalSourceDescriptionError struct{ cause error }

func (err *signalSourceDescriptionError) Error() string {
	return "compile signal source description: " + err.cause.Error()
}

func (err *signalSourceDescriptionError) Unwrap() error { return err.cause }

type signalSourceInput[K comparable] interface {
	signalSourceDefinition() signalSourceDefinition[K]
}

type signalSourceDefinition[K comparable] struct {
	descriptors []SignalDescriptor[K]
	open        SignalOpen[K]
	subscribe   SignalSubscribe[K]
	reraise     SignalReraise[K]
	advanced    bool
}

func (source SignalSource[K]) signalSourceDefinition() signalSourceDefinition[K] {
	definition := signalSourceDefinition[K]{
		descriptors: source.Signals,
		open:        source.Open,
		reraise:     source.Reraise,
	}

	return definition
}

func (source SignalWatchSource[K]) signalSourceDefinition() signalSourceDefinition[K] {
	definition := signalSourceDefinition[K]{
		descriptors: source.Signals,
		subscribe:   source.Subscribe,
		reraise:     source.Reraise,
		advanced:    true,
	}

	return definition
}

type signalSourceScope[K comparable] struct {
	session *declarationSession
	values  map[K]Signal
	valid   bool
}

// Signal returns the Commandment semantic identity already established for k by
// this exact source registration. Unknown k invalidates the enclosing declaration.
func (scope signalSourceScope[K]) Signal(k K) Signal {
	if !scope.valid || scope.session == nil {
		return Signal{}
	}
	signal, ok := scope.values[k]
	if ok {
		return signal
	}
	scope.session.useApplication(func(application *applicationDraft, _ *declarationDraft, _ commandRef) {
		application.invalidSignalDeclaration = true
	})
	return Signal{}
}

// Source installs one static signal source declaration into the root Definition.
// The returned source-scoped value performs lookup only; it never creates identities lazily.
func (signals *Signals[C]) Source[K comparable, S signalSourceInput[K]](source S) signalSourceScope[K] {
	if signals == nil || signals.session == nil {
		return signalSourceScope[K]{}
	}
	definition := source.signalSourceDefinition()
	var scope signalSourceScope[K]
	signals.session.useApplication(func(application *applicationDraft, draft *declarationDraft, command commandRef) {
		if draft == nil || command != draft.root {
			application.invalidSignalDeclaration = true
			return
		}
		scope = registerSignalSource(application, signals.session, definition)
	})
	return scope
}

type erasedOrdinarySignalStream struct {
	receive func(context.Context) (SignalNotification[Signal], error)
	close   SignalClose
}

type erasedAdvancedSignalEvent struct {
	kind         SignalSubscriptionEventKind
	revision     SignalWatchRevision
	notification SignalNotification[Signal]
	err          error
}

type erasedAdvancedSignalSubscription struct {
	setWatch func(context.Context, SignalWatchRevision, []Signal) error
	events   <-chan erasedAdvancedSignalEvent
	close    SignalClose
}

type compiledSignalSource struct {
	id          uint64
	process     bool
	descriptors []signalDescriptor
	ordinary    bool
	open        func(context.Context) (erasedOrdinarySignalStream, error)
	subscribe   func(
		context.Context,
		context.Context,
		SignalWatchRevision,
		[]Signal,
	) (erasedAdvancedSignalSubscription, error)
	reraise func(context.Context, Signal) error
}

func registerSignalSource[K comparable](application *applicationDraft, session *declarationSession, definition signalSourceDefinition[K]) signalSourceScope[K] {
	result := signalSourceScope[K]{session: session}
	if application == nil || len(definition.descriptors) == 0 || (definition.advanced && definition.subscribe == nil) ||
		(!definition.advanced && definition.open == nil) {
		if application != nil {
			application.invalidSignalDeclaration = true
		}
		return result
	}

	descriptors := append([]SignalDescriptor[K](nil), definition.descriptors...)
	values := make(map[K]Signal, len(descriptors))
	native := make(map[Signal]K, len(descriptors))
	canonical := make([]signalDescriptor, 0, len(descriptors))
	reraisable := 0
	for _, descriptor := range descriptors {
		if _, duplicate := values[descriptor.Signal]; duplicate ||
			!validSignalDescriptorMetadata(descriptor.Classes, descriptor.Capabilities, descriptor.Fidelity) {
			application.invalidSignalDeclaration = true
			return result
		}
		application.nextSignalID++
		semantic := Signal{id: application.nextSignalID}
		values[descriptor.Signal] = semantic
		native[semantic] = descriptor.Signal
		if descriptor.Capabilities&SignalSourceCapabilityReraise != 0 {
			reraisable++
		}
		adapted, err := acceptSignalDescriptor(
			semantic,
			descriptor.Classes,
			descriptor.Reserved,
			descriptor.Capabilities,
			descriptor.Fidelity,
		)
		if err != nil {
			application.invalidSignalDeclaration = true
			return result
		}
		canonical = append(canonical, adapted)
	}
	if (definition.reraise == nil) != (reraisable == 0) {
		application.invalidSignalDeclaration = true
		return result
	}

	record := compiledSignalSource{
		id:          uint64(len(application.signalSources) + 1),
		descriptors: canonical,
		ordinary:    !definition.advanced,
	}
	if definition.reraise != nil {
		reraiser := definition.reraise
		record.reraise = func(ctx context.Context, signal Signal) error {
			value, ok := native[signal]
			if !ok {
				return errInvalidSignalSourceDeclaration
			}
			return reraiser(ctx, value)
		}
	}
	if definition.advanced {
		record.subscribe = advancedSignalSourceSubscribe(definition.subscribe, values, native)
	} else {
		record.open = ordinarySignalSourceOpen(definition.open, values)
	}
	application.signalSources = append(application.signalSources, record)
	result.values = values
	result.valid = true
	return result
}

func ordinarySignalSourceOpen[K comparable](open SignalOpen[K], values map[K]Signal) func(context.Context) (erasedOrdinarySignalStream, error) {
	return func(ctx context.Context) (erasedOrdinarySignalStream, error) {
		stream, err := open(ctx)
		if err != nil {
			return erasedOrdinarySignalStream{}, err
		}
		if stream.Receive == nil {
			if stream.Close != nil {
				_ = stream.Close()
			}
			return erasedOrdinarySignalStream{}, errInvalidSignalSourceDeclaration
		}
		return erasedOrdinarySignalStream{
			receive: func(ctx context.Context) (SignalNotification[Signal], error) {
				notification, receiveErr := stream.Receive(ctx)
				if receiveErr != nil {
					return SignalNotification[Signal]{}, receiveErr
				}
				semantic, ok := values[notification.Signal]
				if !ok {
					return SignalNotification[Signal]{}, errInvalidSignalSourceDeclaration
				}
				signalNotification := SignalNotification[Signal]{
					Signal:     semantic,
					Origin:     notification.Origin,
					Sender:     cloneSignalSender(notification.Sender),
					ObservedAt: notification.ObservedAt,
				}
				return signalNotification, nil
			},
			close: stream.Close,
		}, nil
	}
}

func advancedSignalSourceSubscribe[K comparable](subscribe SignalSubscribe[K], values map[K]Signal, native map[Signal]K) func(context.Context, context.Context, SignalWatchRevision, []Signal) (erasedAdvancedSignalSubscription, error) {
	return func(
		lifecycleCtx context.Context,
		callCtx context.Context,
		revision SignalWatchRevision,
		watched []Signal,
	) (erasedAdvancedSignalSubscription, error) {
		nativeWatch, ok := nativeSignalWatch(watched, native)
		if !ok {
			return erasedAdvancedSignalSubscription{}, errInvalidSignalSourceDeclaration
		}
		subscription, err := subscribe(callCtx, revision, nativeWatch)
		if err != nil {
			return erasedAdvancedSignalSubscription{}, err
		}
		if subscription.Events == nil {
			if subscription.Close != nil {
				_ = subscription.Close()
			}
			return erasedAdvancedSignalSubscription{}, errInvalidSignalSourceDeclaration
		}
		events := make(chan erasedAdvancedSignalEvent)
		go forwardAdvancedSignalEvents(lifecycleCtx, subscription.Events, events, values)
		result := erasedAdvancedSignalSubscription{
			setWatch: advancedSignalSetWatch(subscription.SetWatch, native),
			events:   events,
			close:    subscription.Close,
		}
		return result, nil
	}
}

func forwardAdvancedSignalEvents[K comparable](lifecycleCtx context.Context, source <-chan SignalSubscriptionEvent[K], events chan<- erasedAdvancedSignalEvent, values map[K]Signal) {
	defer close(events)
	for {
		var event SignalSubscriptionEvent[K]
		var ok bool
		select {
		case <-lifecycleCtx.Done():
			return
		case event, ok = <-source:
			if !ok {
				return
			}
		}
		erased := erasedAdvancedSignalEvent{
			kind:     event.Kind,
			revision: event.Revision,
			err:      event.Err,
		}
		if event.Kind == SignalSubscriptionEventNotification {
			semantic, found := values[event.Notification.Signal]
			if !found {
				erased = erasedAdvancedSignalEvent{
					kind: SignalSubscriptionEventFailure,
					err:  errInvalidSignalSourceDeclaration,
				}
			} else {
				erased.notification = SignalNotification[Signal]{
					Signal:     semantic,
					Origin:     event.Notification.Origin,
					Sender:     cloneSignalSender(event.Notification.Sender),
					ObservedAt: event.Notification.ObservedAt,
				}
			}
		}
		select {
		case <-lifecycleCtx.Done():
			return
		case events <- erased:
		}
	}
}

func advancedSignalSetWatch[K comparable](set SignalSetWatch[K], native map[Signal]K) func(context.Context, SignalWatchRevision, []Signal) error {
	if set == nil {
		return nil
	}
	return func(ctx context.Context, revision SignalWatchRevision, watched []Signal) error {
		translated, ok := nativeSignalWatch(watched, native)
		if !ok {
			return errInvalidSignalSourceDeclaration
		}
		return set(ctx, revision, translated)
	}
}

func nativeSignalWatch[K comparable](signals []Signal, native map[Signal]K) ([]K, bool) {
	result := make([]K, 0, len(signals))
	for _, signal := range signals {
		value, ok := native[signal]
		if !ok {
			return nil, false
		}
		result = append(result, value)
	}
	return result, true
}

func validSignalDescriptorMetadata(classes SignalClass, capabilities SignalSourceCapabilities, fidelity SignalSourceDeliveryFidelity) bool {
	classMask := SignalClassInterrupt | SignalClassTerminate
	capabilityMask := SignalSourceCapabilityOrigin | SignalSourceCapabilitySender | SignalSourceCapabilityReraise
	return classes&^classMask == 0 && capabilities&^capabilityMask == 0 &&
		(fidelity == SignalSourceDeliveryFidelityExact || fidelity == SignalSourceDeliveryFidelityCoalesced)
}

func cloneSignalSender(sender *SignalSender) *SignalSender {
	if sender == nil {
		return nil
	}
	cloned := *sender
	return &cloned
}

type compiledSignalCatalog struct {
	sources []compiledSignalSource
}

func freezeCompiledSignalCatalog(sources []compiledSignalSource) compiledSignalCatalog {
	result := compiledSignalCatalog{sources: append([]compiledSignalSource(nil), sources...)}
	for i := range result.sources {
		result.sources[i].descriptors = append([]signalDescriptor(nil), result.sources[i].descriptors...)
	}
	return result
}

func (catalog compiledSignalCatalog) available(includeProcess bool) []compiledSignalSource {
	result := make([]compiledSignalSource, 0, len(catalog.sources))
	for _, source := range catalog.sources {
		if source.process && !includeProcess {
			continue
		}
		result = append(result, source)
	}
	return result
}

func signalSourceDescription(sources []compiledSignalSource) (semanticSignalSourceDescription, error) {
	var descriptors []signalDescriptor
	for _, source := range sources {
		descriptors = append(descriptors, source.descriptors...)
	}
	description, err := acceptSignalSourceDescription(descriptors)
	if err != nil {
		return semanticSignalSourceDescription{}, &signalSourceDescriptionError{cause: err}
	}
	return description, nil
}

func sourceForSignal(sources []compiledSignalSource, signal Signal) (compiledSignalSource, bool) {
	for _, source := range sources {
		for _, descriptor := range source.descriptors {
			if descriptor.Signal() == signal {
				return source, true
			}
		}
	}
	return compiledSignalSource{}, false
}
