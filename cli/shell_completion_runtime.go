package cli

import (
	"bufio"
	"context"
	"io"
	"runtime"
	"strings"
)

func stripCompletionActivation(environment Env, name string) Env {
	result := make(Env, 0, len(environment))
	for _, entry := range environment {
		if entry.Name == name || (runtime.GOOS == "windows" && strings.EqualFold(entry.Name, name)) {
			continue
		}
		result = append(result, entry)
	}
	return result
}

func serveCompletionProcess(ctx context.Context, app Application, activationName, activationValue string, args []string, stdin io.Reader, stdout io.Writer, captureHost func() (Env, string, ResponseFileAccess, error)) error {
	if activationValue != shellCompletionActivationValue {
		return projectDiagnosticFailure(
			DiagnosticCompletionProtocolMismatch,
			"invalid completion process activation",
			errInvalidShellCompletionAdapter,
		)
	}
	if len(args) != 0 {
		return projectDiagnosticFailure(
			DiagnosticCompletionProtocolMismatch,
			"completion process arguments are not permitted",
			errInvalidShellCompletionAdapter,
		)
	}

	reader := bufio.NewReader(stdin)
	codec, err := readShellCompletionAdapterPrelude(reader)
	if err != nil {
		return projectDiagnosticFailure(
			DiagnosticCompletionProtocolMismatch,
			"invalid completion request codec prelude",
			err,
		)
	}

	request, err := decodeShellCompletionAdapterRequest(reader, codec)
	if err != nil {
		return projectDiagnosticFailure(
			DiagnosticCompletionProtocolMismatch,
			"invalid shell completion adapter request",
			err,
		)
	}

	environment, workingDirectory, responseFiles, err := captureHost()
	if err != nil {
		return err
	}
	semanticRequest := request.completionRequest()
	semanticRequest.Env = stripCompletionActivation(environment, activationName)
	semanticRequest.WorkingDirectory = workingDirectory
	semanticRequest.ResponseFiles = responseFiles

	result, completionErr := app.Complete(ctx, semanticRequest)
	response := projectShellCompletionAdapterResponse(result, completionErr != nil, request.capabilities)
	if err := encodeShellCompletionAdapterResponse(stdout, request.responseCodec, response); err != nil {
		return &shellCompletionAdapterError{
			detail: "encode response",
			cause:  err,
		}
	}
	return nil
}

func readShellCompletionAdapterPrelude(reader *bufio.Reader) (shellCompletionAdapterCodec, error) {
	const maxPreludeBytes = len("records\n")

	line := make([]byte, 0, maxPreludeBytes)
	for len(line) < maxPreludeBytes {
		value, err := reader.ReadByte()
		if err != nil {
			return "", errInvalidShellCompletionAdapter
		}
		line = append(line, value)
		if value != '\n' {
			continue
		}

		switch string(line) {
		case "records\n":
			return shellCompletionAdapterCodecRecords, nil
		case "json\n":
			return shellCompletionAdapterCodecJSON, nil
		default:
			return "", errInvalidShellCompletionAdapter
		}
	}
	return "", errInvalidShellCompletionAdapter
}
