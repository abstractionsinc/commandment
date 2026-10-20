package shelltest

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	bridgeBootstrapLimit   = 256
	bridgeUsageExitCode    = 2
	bridgeDialTimeout      = 5 * time.Second
	bridgeAdmissionTimeout = 2 * time.Second
	bridgeRequestTimeout   = 10 * time.Second
	bridgeResponseTimeout  = 60 * time.Second
	bridgeAckTimeout       = 5 * time.Second
)

type bridgeBootstrap struct {
	endpoint   string
	capability bridgeCapability
}

func materializeBridgeHelper(workspace string, generation generatedCompletion, bridge *applicationBridge) (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", failureError{
			subject: "resolve shelltest helper executable",
			cause:   err,
		}
	}
	relative := filepath.Join("bin", generation.generation.ApplicationName)
	if err := copyWorkspaceExecutable(workspace, relative, executable); err != nil {
		return "", failureError{
			subject: "copy shelltest helper executable",
			cause:   err,
		}
	}
	bootstrap := encodeBridgeBootstrap(bridgeBootstrap{
		endpoint:   bridge.endpoint,
		capability: bridge.capability,
	})
	if len(bootstrap) > bridgeBootstrapLimit {
		return "", failureError{subject: "application bridge bootstrap exceeds limit"}
	}
	if err := createWorkspaceFile(workspace, relative+bridgeBootstrapSuffix, bootstrap, privateFileMode); err != nil {
		return "", failureError{
			subject: "create application bridge bootstrap",
			cause:   err,
		}
	}
	return filepath.Join(workspace, relative), nil
}

func encodeBridgeBootstrap(bootstrap bridgeBootstrap) []byte {
	return []byte("commandment-shelltest-bridge-v1\n" + bootstrap.endpoint + "\n" + hex.EncodeToString(bootstrap.capability[:]) + "\n")
}

func decodeBridgeBootstrap(data []byte) (bridgeBootstrap, error) {
	if len(data) == 0 || len(data) > bridgeBootstrapLimit {
		return bridgeBootstrap{}, failureError{subject: "invalid application bridge bootstrap"}
	}
	parts := bytes.Split(data, []byte{'\n'})
	if len(parts) != 4 || string(parts[0]) != "commandment-shelltest-bridge-v1" || len(parts[3]) != 0 {
		return bridgeBootstrap{}, failureError{subject: "invalid application bridge bootstrap"}
	}
	endpoint := string(parts[1])
	address, err := netip.ParseAddrPort(endpoint)
	if err != nil || address.Addr() != netip.MustParseAddr("127.0.0.1") || address.Port() == 0 {
		return bridgeBootstrap{}, failureError{subject: "invalid application bridge endpoint"}
	}
	capabilityBytes, err := hex.DecodeString(string(parts[2]))
	if err != nil || len(capabilityBytes) != bridgeCapabilityBytes || strings.ToLower(string(parts[2])) != string(parts[2]) {
		return bridgeBootstrap{}, failureError{subject: "invalid application bridge capability"}
	}
	var capability bridgeCapability
	copy(capability[:], capabilityBytes)
	return bridgeBootstrap{
		endpoint:   endpoint,
		capability: capability,
	}, nil
}

//nolint:gochecknoinits // The copied helper must branch before testing package startup; the adjacent sidecar is its only discriminator.
func init() {
	executable, err := os.Executable()
	if err != nil {
		return
	}
	path := executable + bridgeBootstrapSuffix
	// #nosec G304 -- the sidecar path is derived only from os.Executable().
	file, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err != nil {
		helperDiagnostic("bootstrap", err)
		os.Exit(bridgeUsageExitCode)
	}
	data, readErr := io.ReadAll(io.LimitReader(file, bridgeBootstrapLimit+1))
	closeErr := file.Close()
	if readErr != nil {
		helperDiagnostic("bootstrap read", readErr)
		os.Exit(bridgeUsageExitCode)
	}
	if closeErr != nil {
		helperDiagnostic("bootstrap close", closeErr)
		os.Exit(bridgeUsageExitCode)
	}
	if len(data) > bridgeBootstrapLimit {
		helperDiagnostic("bootstrap exceeds limit", nil)
		os.Exit(bridgeUsageExitCode)
	}
	bootstrap, err := decodeBridgeBootstrap(data)
	if err != nil {
		helperDiagnostic("bootstrap decode", err)
		os.Exit(bridgeUsageExitCode)
	}
	os.Exit(runBridgeHelper(bootstrap))
}

func runBridgeHelper(bootstrap bridgeBootstrap) int {
	workingDirectory, err := os.Getwd()
	if err != nil {
		helperDiagnostic("working directory", err)
		return bridgeUsageExitCode
	}
	stdin, err := io.ReadAll(io.LimitReader(os.Stdin, bridgeStdinLimit+1))
	if err != nil || len(stdin) > bridgeStdinLimit {
		helperDiagnostic("stdin", err)
		return bridgeUsageExitCode
	}
	processRequest := bridgeProcessRequest{
		args:             slices.Clone(os.Args),
		environ:          slices.Clone(os.Environ()),
		workingDirectory: workingDirectory,
		stdin:            stdin,
	}
	request, err := encodeBridgeProcessRequest(processRequest)
	if err != nil {
		helperDiagnostic("request", err)
		return bridgeUsageExitCode
	}
	ctx, cancel := context.WithTimeout(context.Background(), bridgeDialTimeout)
	defer cancel()
	connection, err := (&net.Dialer{}).DialContext(ctx, "tcp4", bootstrap.endpoint)
	if err != nil {
		helperDiagnostic("connect", err)
		return bridgeUsageExitCode
	}
	defer func() { _ = connection.Close() }()
	if err := connection.SetWriteDeadline(time.Now().Add(bridgeAdmissionTimeout)); err != nil {
		return bridgeUsageExitCode
	}
	if err := writeAll(connection, bootstrap.capability[:]); err != nil {
		helperDiagnostic("capability", err)
		return bridgeUsageExitCode
	}
	if err := connection.SetWriteDeadline(time.Now().Add(bridgeRequestTimeout)); err != nil {
		return bridgeUsageExitCode
	}
	if err := writeBridgeFrame(connection, request, bridgeRequestFrameLimit); err != nil {
		helperDiagnostic("request", err)
		return bridgeUsageExitCode
	}
	if err := connection.SetReadDeadline(time.Now().Add(bridgeResponseTimeout)); err != nil {
		return bridgeUsageExitCode
	}
	response, err := readBridgeFrame(connection, bridgeResponseFrameLimit)
	if err != nil {
		helperDiagnostic("response", err)
		return bridgeUsageExitCode
	}
	if len(response) != 0 {
		if err := writeAll(os.Stdout, response); err != nil {
			helperDiagnostic("stdout", err)
			return bridgeUsageExitCode
		}
	}
	if err := connection.SetWriteDeadline(time.Now().Add(bridgeAckTimeout)); err != nil {
		return bridgeUsageExitCode
	}
	if err := writeBridgeFrame(connection, []byte{1}, bridgeAckFrameLimit); err != nil {
		helperDiagnostic("acknowledgement", err)
		return bridgeUsageExitCode
	}
	return 0
}

func helperDiagnostic(subject string, err error) {
	detail := ""
	if err != nil {
		detail = boundedFailureDetail(err.Error())
	}
	if detail == "" {
		_, _ = fmt.Fprintf(os.Stderr, "shelltest helper: %s\n", subject)
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "shelltest helper: %s: %s\n", subject, detail)
}
