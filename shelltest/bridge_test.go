package shelltest

import (
	"context"
	"encoding/binary"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/abstractionsinc/commandment/cli"
)

func TestBridgeProcessRequestRoundTripsAndRejectsOversizeFrameBeforePayload(t *testing.T) {
	t.Parallel()
	request := bridgeProcessRequest{
		args:             []string{"alienctl"},
		environ:          []string{"A=B"},
		workingDirectory: "/workspace",
		stdin:            []byte("records\nend\n"),
	}
	encoded, err := encodeBridgeProcessRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeBridgeProcessRequest(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, request) {
		t.Fatalf("bridge request round trip = %#v, want %#v", decoded, request)
	}

	reader, writer := net.Pipe()
	defer func() { _ = reader.Close() }()
	defer func() { _ = writer.Close() }()
	go func() {
		var header [4]byte
		binary.BigEndian.PutUint32(header[:], bridgeAckFrameLimit+1)
		_, _ = writer.Write(header[:])
	}()
	if _, err := readBridgeFrame(reader, bridgeAckFrameLimit); err == nil {
		t.Fatal("oversize frame header accepted")
	}
}

func TestBoundedWriterFailsWithoutTruncatingSemanticOutput(t *testing.T) {
	t.Parallel()
	writer := boundedWriter{limit: 3}
	if _, err := writer.Write([]byte("abcd")); err == nil {
		t.Fatal("overflowing write succeeded")
	}
	if data, err := writer.bytes(); err == nil || data != nil {
		t.Fatalf("overflowed writer bytes = %q, %v", data, err)
	}
}

func TestBridgeBootstrapRoundTripAndTrailingDataRejection(t *testing.T) {
	t.Parallel()
	var capability bridgeCapability
	for index := range capability {
		capability[index] = byte(index + 1)
	}
	bootstrap := bridgeBootstrap{endpoint: "127.0.0.1:12345", capability: capability}
	encoded := encodeBridgeBootstrap(bootstrap)
	decoded, err := decodeBridgeBootstrap(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded != bootstrap {
		t.Fatalf("bootstrap round trip = %#v, want %#v", decoded, bootstrap)
	}
	if _, err := decodeBridgeBootstrap(append(encoded, 'x')); err == nil {
		t.Fatal("bootstrap trailing data accepted")
	}
}

func TestBridgeUnauthenticatedNoiseDoesNotCreateSemanticFailure(t *testing.T) {
	t.Parallel()
	app := compileRemoteApplication(t)
	bridge, err := startApplicationBridge(t.Context(), app, testGeneration())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = bridge.cleanup(ctx)
	}()

	connection, err := (&net.Dialer{Timeout: time.Second}).DialContext(t.Context(), "tcp", bridge.endpoint)
	if err != nil {
		t.Fatal(err)
	}
	wrong := bridge.capability
	wrong[0] ^= 0xff
	if _, err := connection.Write(wrong[:]); err != nil {
		t.Fatal(err)
	}
	_ = connection.Close()

	deadline := time.Now().Add(time.Second)
	for {
		bridge.mu.Lock()
		active := len(bridge.connections)
		failure := bridge.failure
		bridge.mu.Unlock()
		if failure != nil {
			t.Fatalf("unauthenticated noise became semantic failure: %v", failure)
		}
		if active == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("unauthenticated connection did not drain")
		}
		time.Sleep(time.Millisecond)
	}
}

func testGeneration() cli.ShellCompletionGeneration {
	return cli.ShellCompletionGeneration{
		ApplicationName:       "alienctl",
		ActivationEnvironment: "COMMANDMENT_COMPLETE",
		ActivationValue:       "adapter",
	}
}

func TestBridgeCausalFrontierUsesAcceptanceTimeAndRequiresCompletedExchange(t *testing.T) {
	t.Parallel()
	app := compileRemoteApplication(t)

	t.Run("accepted-before-arm", func(t *testing.T) {
		t.Parallel()
		bridge, err := startApplicationBridge(t.Context(), app, testGeneration())
		if err != nil {
			t.Fatal(err)
		}
		defer cleanupTestBridge(t, bridge)

		connection, err := (&net.Dialer{Timeout: time.Second}).DialContext(t.Context(), "tcp", bridge.endpoint)
		if err != nil {
			t.Fatal(err)
		}
		waitBridgeAccepted(t, bridge, 1)
		if err := bridge.arm(); err != nil {
			t.Fatal(err)
		}
		performBridgeExchange(t, bridge, connection)
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()
		if err := bridge.waitWitness(ctx); err == nil {
			t.Fatal("pre-arm accepted connection became causal witness")
		}
		bridge.mu.Lock()
		failure := bridge.failure
		bridge.mu.Unlock()
		if failure != nil {
			t.Fatalf("valid pre-arm exchange became semantic failure: %v", failure)
		}
	})

	t.Run("accepted-after-arm", func(t *testing.T) {
		t.Parallel()
		bridge, err := startApplicationBridge(t.Context(), app, testGeneration())
		if err != nil {
			t.Fatal(err)
		}
		defer cleanupTestBridge(t, bridge)
		if err := bridge.arm(); err != nil {
			t.Fatal(err)
		}
		connection, err := (&net.Dialer{Timeout: time.Second}).DialContext(t.Context(), "tcp", bridge.endpoint)
		if err != nil {
			t.Fatal(err)
		}
		performBridgeExchange(t, bridge, connection)
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		if err := bridge.waitWitness(ctx); err != nil {
			t.Fatalf("post-arm valid exchange did not witness: %v", err)
		}
	})
}

func performBridgeExchange(t *testing.T, bridge *applicationBridge, connection net.Conn) {
	t.Helper()
	request := bridgeProcessRequest{
		args:             []string{"alienctl"},
		environ:          []string{bridge.activationEnvironment + "=" + bridge.activationValue},
		workingDirectory: t.TempDir(),
		stdin: []byte("records\nresponse-encoding\trecords\nshell\tbash\t5.2\n" +
			"input\texact\t1\t0\t8\tobserved-word-bytes\nword\t0\talienctl\texact\nend\n"),
	}
	payload, err := encodeBridgeProcessRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Write(bridge.capability[:]); err != nil {
		t.Fatal(err)
	}
	if err := writeBridgeFrame(connection, payload, bridgeRequestFrameLimit); err != nil {
		t.Fatal(err)
	}
	if _, err := readBridgeFrame(connection, bridgeResponseFrameLimit); err != nil {
		t.Fatal(err)
	}
	if err := writeBridgeFrame(connection, []byte{1}, bridgeAckFrameLimit); err != nil {
		t.Fatal(err)
	}
	if err := connection.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := bridge.waitConnectionsEmpty(ctx); err != nil {
		t.Fatal(err)
	}
}

func waitBridgeAccepted(t *testing.T, bridge *applicationBridge, count uint64) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		bridge.mu.Lock()
		accepted := bridge.nextAccepted
		bridge.mu.Unlock()
		if accepted >= count {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("bridge accepted %d connections, want at least %d", accepted, count)
		}
		time.Sleep(time.Millisecond)
	}
}

func cleanupTestBridge(t *testing.T, bridge *applicationBridge) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := bridge.cleanup(ctx); err != nil {
		t.Fatal(err)
	}
}
