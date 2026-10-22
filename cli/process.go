package cli

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
)

type processAdapterError struct {
	operation string
	subject   string
	cause     error
}

func (err *processAdapterError) Error() string {
	text := "cli: " + err.operation
	if err.subject != "" {
		text += " " + err.subject
	}
	if err.cause != nil {
		text += ": " + err.cause.Error()
	}
	return text
}

func (err *processAdapterError) Unwrap() error { return err.cause }

// runApplicationFromProcess captures current-process inputs once and enters
// the same private Run engine used by Application.Run.
// Process capture is adapter work only; it does not pre-analyze or pre-expand
// command input.
func runApplicationFromProcess(ctx context.Context, app Application) error {
	model := app.mustStaticProgram()
	activationName := shellCompletionActivationName(model.Application().name)
	if activationValue, present := os.LookupEnv(activationName); present {
		return serveCompletionProcess(
			ctx, app, activationName, activationValue, append([]string(nil), os.Args[1:]...), os.Stdin, os.Stdout,
			func() (Env, string, ResponseFileAccess, error) {
				environment, err := EnvFromEnviron(os.Environ())
				if err != nil {
					failure := &processAdapterError{
						operation: "capture process environment",
						cause:     err,
					}
					return nil, "", ResponseFileAccess{}, failure
				}
				workingDirectory, err := os.Getwd()
				if err != nil {
					failure := &processAdapterError{
						operation: "capture process working directory",
						cause:     err,
					}
					return nil, "", ResponseFileAccess{}, failure
				}
				return environment, workingDirectory, processResponseFileAccess(), nil
			},
		)
	}
	environment, err := EnvFromEnviron(os.Environ())
	if err != nil {
		failure := &processAdapterError{
			operation: "capture process environment",
			cause:     err,
		}
		return failure
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		failure := &processAdapterError{
			operation: "capture process working directory",
			cause:     err,
		}
		return failure
	}
	availableSignals := app.signalCatalog().available(true)
	request := RunRequest{
		Args:             os.Args[1:],
		Env:              environment,
		WorkingDirectory: workingDirectory,
		ResponseFiles:    processResponseFileAccess(),
		Stdin:            os.Stdin,
		Stdout:           os.Stdout,
		Stderr:           os.Stderr,
		Signals: SignalHost{
			Abort: processSignalAbort,
		},
	}

	return runApplication(ctx, model, app.callbacks, app.identity, app.shellCompletion, availableSignals, request)
}

func processResponseFileAccess() ResponseFileAccess {
	access := ResponseFileAccess{
		Load: func(ctx context.Context, request ResponseFileRequest) (ResponseFile, error) {
			if err := ctx.Err(); err != nil {
				failure := &processAdapterError{
					operation: "check response-file context",
					cause:     err,
				}
				return ResponseFile{}, failure
			}
			path := request.Path
			if !filepath.IsAbs(path) {
				path = filepath.Join(request.Base, path)
			}
			path, err := filepath.Abs(path)
			if err != nil {
				result := &processAdapterError{
					operation: "resolve response file",
					subject:   strconv.Quote(request.Path),
					cause:     err,
				}
				return ResponseFile{}, result
			}
			content, err := os.ReadFile(path)
			if err != nil {
				result := &processAdapterError{
					operation: "read response file",
					subject:   strconv.Quote(path),
					cause:     err,
				}
				return ResponseFile{}, result
			}
			if err := ctx.Err(); err != nil {
				failure := &processAdapterError{
					operation: "check response-file context",
					cause:     err,
				}
				return ResponseFile{}, failure
			}
			responseFile := ResponseFile{
				Key:         path,
				Location:    path,
				IncludeBase: filepath.Dir(path),
				Content:     content,
			}
			return responseFile, nil
		},
	}
	return access
}
