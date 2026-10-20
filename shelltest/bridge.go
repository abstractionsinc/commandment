package shelltest

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/abstractionsinc/commandment/cli"
)

const (
	bridgeCapabilityBytes = 32
	bridgeConnectionLimit = 64
	privateDirectoryMode  = 0o700
	privateExecutableMode = 0o700
	privateFileMode       = 0o600
)

type bridgeCapability [bridgeCapabilityBytes]byte

const applicationCompletionPanicSubject = "application completion panicked"

type applicationBridgePhase uint8

const (
	applicationBridgeBeforeCompletion applicationBridgePhase = iota
	applicationBridgeAwaitingCompletion
	applicationBridgeWitnessed
	applicationBridgeSealing
	applicationBridgeSettled
	applicationBridgeClosed
)

type applicationBridge struct {
	cancel                context.CancelFunc
	listener              net.Listener
	endpoint              string
	capability            bridgeCapability
	activationEnvironment string
	activationValue       string
	app                   cli.Application
	acceptDone            chan struct{}
	changed               chan struct{}
	mu                    sync.Mutex
	phase                 applicationBridgePhase
	nextAccepted          uint64
	armedAfterAccepted    uint64
	authenticatedCount    uint64
	failure               error
	panicSet              bool
	panicValue            any
	connections           map[net.Conn]struct{}
}

func startApplicationBridge(parent context.Context, app cli.Application, generation cli.ShellCompletionGeneration) (*applicationBridge, error) {
	listener, err := (&net.ListenConfig{}).Listen(parent, "tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, failureError{
			subject: "listen for application bridge",
			cause:   err,
		}
	}
	ctx, cancel := context.WithCancel(parent)
	bridge := &applicationBridge{
		cancel:                cancel,
		listener:              listener,
		endpoint:              listener.Addr().String(),
		activationEnvironment: generation.ActivationEnvironment,
		activationValue:       generation.ActivationValue,
		app:                   app,
		acceptDone:            make(chan struct{}),
		changed:               make(chan struct{}, 1),
		phase:                 applicationBridgeBeforeCompletion,
		connections:           make(map[net.Conn]struct{}),
	}
	if _, err := rand.Read(bridge.capability[:]); err != nil {
		cancel()
		_ = listener.Close()
		return nil, failureError{
			subject: "create application bridge capability",
			cause:   err,
		}
	}
	go bridge.accept(ctx)
	return bridge, nil
}

func (bridge *applicationBridge) accept(ctx context.Context) {
	defer close(bridge.acceptDone)
	for {
		connection, err := bridge.listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return
			}
			bridge.recordFailure(failureError{
				subject: "accept application bridge session",
				cause:   err,
			})
			return
		}
		bridge.mu.Lock()
		if bridge.nextAccepted == ^uint64(0) {
			bridge.mu.Unlock()
			_ = connection.Close()
			bridge.recordFailure(failureError{subject: "application bridge acceptance counter exhausted"})
			return
		}
		bridge.nextAccepted++
		accepted := bridge.nextAccepted
		if len(bridge.connections) >= bridgeConnectionLimit {
			bridge.mu.Unlock()
			_ = connection.Close()
			continue
		}
		bridge.connections[connection] = struct{}{}
		bridge.mu.Unlock()
		bridge.notify()
		go bridge.handle(ctx, connection, accepted)
	}
}

func (bridge *applicationBridge) handle(ctx context.Context, connection net.Conn, accepted uint64) {
	defer func() {
		_ = connection.Close()
		bridge.mu.Lock()
		delete(bridge.connections, connection)
		bridge.mu.Unlock()
		bridge.notify()
	}()
	if err := connection.SetDeadline(time.Now().Add(bridgeAdmissionTimeout)); err != nil {
		bridge.recordFailure(failureError{
			subject: "bound application bridge admission",
			cause:   err,
		})
		return
	}
	var presented bridgeCapability
	if _, err := io.ReadFull(connection, presented[:]); err != nil {
		return
	}
	if subtle.ConstantTimeCompare(presented[:], bridge.capability[:]) != 1 {
		return
	}

	bridge.mu.Lock()
	if bridge.authenticatedCount == ^uint64(0) {
		bridge.failure = firstFailure(bridge.failure, failureError{subject: "application bridge authentication counter exhausted"})
		bridge.mu.Unlock()
		bridge.notify()
		return
	}
	bridge.authenticatedCount++
	phase := bridge.phase
	causal := accepted > bridge.armedAfterAccepted && (phase == applicationBridgeAwaitingCompletion || phase == applicationBridgeWitnessed)
	if phase >= applicationBridgeSealing {
		bridge.failure = firstFailure(bridge.failure, failureError{subject: "authenticated helper exchange raced bridge sealing"})
	}
	bridge.mu.Unlock()
	bridge.notify()
	if phase >= applicationBridgeSealing {
		return
	}

	if err := connection.SetReadDeadline(time.Now().Add(bridgeRequestTimeout)); err != nil {
		bridge.recordFailure(failureError{
			subject: "bound application bridge request",
			cause:   err,
		})
		return
	}
	payload, err := readBridgeFrame(connection, bridgeRequestFrameLimit)
	if err != nil {
		bridge.recordFailure(failureError{
			subject: "read application bridge request",
			cause:   err,
		})
		return
	}
	processRequest, err := decodeBridgeProcessRequest(payload)
	if err != nil {
		bridge.recordFailure(err)
		return
	}
	if len(processRequest.args) != 1 {
		bridge.recordFailure(failureError{subject: "completion helper received transport arguments"})
		return
	}
	adapterRequest, err := decodeShellAdapterRequest(processRequest.stdin)
	if err != nil {
		bridge.recordFailure(err)
		return
	}
	completionRequest, err := adapterRequest.completionRequest(processRequest.environ, processRequest.workingDirectory, bridge.activationEnvironment, bridge.activationValue)
	if err != nil {
		bridge.recordFailure(err)
		return
	}

	var completion cli.Completion
	var completionErr error
	completed := false
	func() {
		defer func() {
			if completed {
				return
			}
			bridge.recordPanic(recover())
		}()
		completion, completionErr = bridge.app.Complete(ctx, completionRequest)
		completed = true
	}()
	if !completed {
		return
	}

	response := projectShellAdapterResponse(completion, completionErr != nil, adapterRequest.capabilities)
	writer := boundedWriter{limit: bridgeCompletionResponseLimit}
	if err := encodeShellAdapterResponse(&writer, adapterRequest.responseCodec, response); err != nil {
		bridge.recordFailure(failureError{
			subject: "encode shell completion response",
			cause:   err,
		})
		return
	}
	responseBytes, err := writer.bytes()
	if err != nil {
		bridge.recordFailure(err)
		return
	}
	if err := connection.SetWriteDeadline(time.Now().Add(bridgeResponseTimeout)); err != nil {
		bridge.recordFailure(failureError{
			subject: "bound application bridge response",
			cause:   err,
		})
		return
	}
	if err := writeBridgeFrame(connection, responseBytes, bridgeResponseFrameLimit); err != nil {
		bridge.recordFailure(failureError{
			subject: "write application bridge response",
			cause:   err,
		})
		return
	}
	if err := connection.SetReadDeadline(time.Now().Add(bridgeAckTimeout)); err != nil {
		bridge.recordFailure(failureError{
			subject: "bound application bridge acknowledgement",
			cause:   err,
		})
		return
	}
	ack, err := readBridgeFrame(connection, bridgeAckFrameLimit)
	if err != nil {
		bridge.recordFailure(failureError{
			subject: "read application bridge acknowledgement",
			cause:   err,
		})
		return
	}
	if !bytes.Equal(ack, []byte{1}) {
		bridge.recordFailure(failureError{subject: "invalid application bridge acknowledgement"})
		return
	}
	if err := connection.SetReadDeadline(time.Now().Add(bridgeAckTimeout)); err != nil {
		bridge.recordFailure(failureError{
			subject: "bound application helper close",
			cause:   err,
		})
		return
	}
	var trailing [1]byte
	if count, err := connection.Read(trailing[:]); count != 0 || !errors.Is(err, io.EOF) {
		bridge.recordFailure(failureError{
			subject: "application helper transport did not close cleanly",
			cause:   err,
		})
		return
	}

	if causal {
		bridge.mu.Lock()
		if bridge.phase == applicationBridgeAwaitingCompletion {
			bridge.phase = applicationBridgeWitnessed
		}
		bridge.mu.Unlock()
		bridge.notify()
	}
}

func (bridge *applicationBridge) arm() error {
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	if bridge.failure != nil {
		return bridge.failure
	}
	if bridge.panicSet {
		return failureError{subject: applicationCompletionPanicSubject}
	}
	if bridge.phase != applicationBridgeBeforeCompletion {
		return failureError{subject: "application bridge causal frontier already armed"}
	}
	bridge.armedAfterAccepted = bridge.nextAccepted
	bridge.phase = applicationBridgeAwaitingCompletion
	bridge.notifyLocked()
	return nil
}

func (bridge *applicationBridge) waitWitness(ctx context.Context) error {
	return bridge.wait(ctx, func() (bool, error) {
		if bridge.failure != nil {
			return false, bridge.failure
		}
		if bridge.panicSet {
			return false, failureError{subject: applicationCompletionPanicSubject}
		}
		return bridge.phase == applicationBridgeWitnessed, nil
	})
}

func (bridge *applicationBridge) waitConnectionsEmpty(ctx context.Context) error {
	return bridge.wait(ctx, func() (bool, error) {
		if bridge.failure != nil {
			return false, bridge.failure
		}
		if bridge.panicSet {
			return false, failureError{subject: applicationCompletionPanicSubject}
		}
		return len(bridge.connections) == 0, nil
	})
}

func (bridge *applicationBridge) authenticatedFrontier() uint64 {
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	return bridge.authenticatedCount
}

func (bridge *applicationBridge) beginSealing() error {
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	if bridge.failure != nil {
		return bridge.failure
	}
	if bridge.panicSet {
		return failureError{subject: applicationCompletionPanicSubject}
	}
	if bridge.phase != applicationBridgeWitnessed {
		return failureError{subject: "application bridge completion not witnessed"}
	}
	bridge.phase = applicationBridgeSealing
	if err := bridge.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		bridge.failure = firstFailure(bridge.failure, failureError{
			subject: "seal application bridge listener",
			cause:   err,
		})
		return bridge.failure
	}
	bridge.notifyLocked()
	return nil
}

func (bridge *applicationBridge) waitAcceptDrained(ctx context.Context) error {
	select {
	case <-bridge.acceptDone:
		return nil
	case <-ctx.Done():
		return failureError{
			subject: "wait for application bridge accept loop",
			cause:   ctx.Err(),
		}
	}
}

func (bridge *applicationBridge) settle(frontier uint64) error {
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	if bridge.failure != nil {
		return bridge.failure
	}
	if bridge.panicSet {
		return failureError{subject: applicationCompletionPanicSubject}
	}
	if bridge.phase != applicationBridgeSealing {
		return failureError{subject: "application bridge is not sealing"}
	}
	if len(bridge.connections) != 0 {
		return failureError{subject: "application bridge handlers remain active"}
	}
	if bridge.authenticatedCount != frontier {
		return failureError{subject: "authenticated helper exchange raced bridge settlement"}
	}
	bridge.phase = applicationBridgeSettled
	bridge.notifyLocked()
	return nil
}

func (bridge *applicationBridge) cleanup(ctx context.Context) error {
	bridge.cancel()
	_ = bridge.listener.Close()
	bridge.mu.Lock()
	connections := make([]net.Conn, 0, len(bridge.connections))
	for connection := range bridge.connections {
		connections = append(connections, connection)
	}
	bridge.mu.Unlock()
	for _, connection := range connections {
		_ = connection.Close()
	}
	if err := bridge.waitAcceptDrained(ctx); err != nil {
		return err
	}
	if err := bridge.wait(ctx, func() (bool, error) { return len(bridge.connections) == 0, nil }); err != nil {
		return err
	}
	bridge.mu.Lock()
	bridge.phase = applicationBridgeClosed
	bridge.mu.Unlock()
	return nil
}

func (bridge *applicationBridge) panic() (any, bool) {
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	return bridge.panicValue, bridge.panicSet
}

func (bridge *applicationBridge) recordFailure(err error) {
	if err == nil {
		return
	}
	bridge.mu.Lock()
	bridge.failure = firstFailure(bridge.failure, err)
	bridge.mu.Unlock()
	bridge.notify()
}

func (bridge *applicationBridge) recordPanic(value any) {
	bridge.mu.Lock()
	if !bridge.panicSet {
		bridge.panicSet = true
		bridge.panicValue = value
	}
	bridge.mu.Unlock()
	bridge.notify()
}

func (bridge *applicationBridge) wait(ctx context.Context, predicate func() (bool, error)) error {
	for {
		bridge.mu.Lock()
		done, err := predicate()
		bridge.mu.Unlock()
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		select {
		case <-bridge.changed:
		case <-ctx.Done():
			return failureError{
				subject: "wait for application bridge",
				cause:   ctx.Err(),
			}
		}
	}
}

func (bridge *applicationBridge) notify() {
	select {
	case bridge.changed <- struct{}{}:
	default:
	}
}

func (bridge *applicationBridge) notifyLocked() {
	select {
	case bridge.changed <- struct{}{}:
	default:
	}
}

func firstFailure(current, candidate error) error {
	if current != nil {
		return current
	}
	return candidate
}
