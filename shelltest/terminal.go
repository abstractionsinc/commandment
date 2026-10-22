package shelltest

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	terminalQuietPeriod    = 1500 * time.Millisecond
	terminalPhaseTimeout   = 10 * time.Second
	terminalCleanupTimeout = 5 * time.Second
	terminalReadBufferSize = 32 << 10
)

type terminalSession struct {
	process  *terminalProcess
	input    *lockedWriter
	output   *terminalOutput
	readDone chan struct{}
	readMu   sync.Mutex
	readErr  error
}

func startTerminalSession(parent context.Context, executable string, args []string, workspace string, environment []string) (*terminalSession, error) {
	process, err := startTerminalProcess(parent, executable, args, workspace, environment)
	if err != nil {
		failure := failureError{
			subject: "start real shell editor",
			cause:   err,
		}
		return nil, failure
	}
	session := &terminalSession{
		process:  process,
		readDone: make(chan struct{}),
	}
	session.input = &lockedWriter{
		writer: process,
	}
	output := &terminalOutput{
		changed:      make(chan struct{}, 1),
		available:    true,
		decckmKnown:  true,
		autowrap:     true,
		scrollBottom: terminalRows - 1,
	}
	output.reply = func(data []byte) error {
		_, err := session.input.Write(data)
		return err
	}
	session.output = output
	go func() {
		defer close(session.readDone)
		buffer := make([]byte, terminalReadBufferSize)
		for {
			count, err := process.Read(buffer)
			if count != 0 {
				if _, writeErr := session.output.Write(buffer[:count]); writeErr != nil {
					session.setReadError(writeErr)
					return
				}
			}
			if err != nil {
				if !process.exited() {
					session.setReadError(err)
				}
				return
			}
		}
	}()
	return session, nil
}

func (session *terminalSession) waitQuiet(parent context.Context, subject string) (uint64, error) {
	ctx, cancel := context.WithTimeout(parent, terminalPhaseTimeout)
	defer cancel()
	generation := session.output.currentGeneration()
	timer := time.NewTimer(terminalQuietPeriod)
	defer timer.Stop()
	for {
		select {
		case <-session.output.changed:
			generation = session.output.currentGeneration()
			timer.Reset(terminalQuietPeriod)
		case <-timer.C:
			if err := session.readError(); err != nil {
				failure := failureError{
					subject: subject,
					cause:   err,
				}
				return 0, failure
			}
			if err := session.output.failure(); err != nil {
				failure := failureError{
					subject: subject,
					cause:   err,
				}
				return 0, failure
			}
			if session.process.exited() {
				failure := failureError{
					subject: subject,
					detail:  session.diagnostics(),
					cause:   session.process.waitError(),
				}
				return 0, failure
			}
			current := session.output.currentGeneration()
			if current != generation {
				generation = current
				timer.Reset(terminalQuietPeriod)
				continue
			}
			return generation, nil
		case <-session.process.done():
			failure := failureError{
				subject: subject,
				detail:  session.diagnostics(),
				cause:   session.process.waitError(),
			}
			return 0, failure
		case <-ctx.Done():
			failure := failureError{
				subject: subject,
				detail:  session.diagnostics(),
				cause:   ctx.Err(),
			}
			return 0, failure
		}
	}
}

func (session *terminalSession) enter(input Input) error {
	if _, err := io.WriteString(session.input, input.BeforeCursor+input.AfterCursor); err != nil {
		failure := failureError{
			subject: "enter shell editor input",
			cause:   err,
		}
		return failure
	}
	count := utf8.RuneCountInString(input.AfterCursor)
	if count == 0 {
		return nil
	}
	applicationMode, known := session.output.mode()
	if !known {
		failure := failureError{
			subject: "terminal cursor-key mode is not provable",
		}
		return failure
	}
	left := "\x1b[D"
	if applicationMode {
		left = "\x1bOD"
	}
	if _, err := io.WriteString(session.input, strings.Repeat(left, count)); err != nil {
		failure := failureError{
			subject: "move shell editor cursor",
			cause:   err,
		}
		return failure
	}
	return nil
}

func (session *terminalSession) complete() error {
	_, err := io.WriteString(session.input, "\t")
	if err != nil {
		failure := failureError{
			subject: "send shell completion action",
			cause:   err,
		}
		return failure
	}
	return nil
}

func (session *terminalSession) finalScreen(parent context.Context) ([]string, error) {
	for {
		generation, err := session.waitQuiet(parent, "wait for final terminal quiet")
		if err != nil {
			return nil, err
		}
		lines, ok := session.output.snapshot(generation)
		if err := session.output.failure(); err != nil {
			failure := failureError{
				subject: "wait for final terminal quiet",
				cause:   err,
			}
			return nil, failure
		}
		if ok {
			return lines, nil
		}
		if session.output.currentGeneration() == generation {
			failure := failureError{
				subject: "settled terminal presentation is unavailable",
			}
			return nil, failure
		}
	}
}

func (session *terminalSession) cleanup(parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, terminalCleanupTimeout)
	defer cancel()
	session.process.terminate()
	select {
	case <-session.process.done():
	case <-ctx.Done():
		failure := failureError{
			subject: "terminate real shell editor",
			cause:   ctx.Err(),
		}
		return failure
	}
	session.process.closeTerminal()
	select {
	case <-session.readDone:
	case <-ctx.Done():
		failure := failureError{
			subject: "drain real shell editor output",
			cause:   ctx.Err(),
		}
		return failure
	}
	return nil
}

func (session *terminalSession) diagnostics() string {
	tail := session.output.diagnosticTail()
	if len(tail) == 0 {
		return ""
	}
	return "terminal-tail=" + fmt.Sprintf("%q", string(tail))
}

type lockedWriter struct {
	mu     sync.Mutex
	writer io.Writer
}

func (writer *lockedWriter) Write(data []byte) (int, error) {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if err := writeAll(writer.writer, data); err != nil {
		failure := failureError{
			subject: "write terminal input",
			cause:   err,
		}
		return 0, failure
	}
	return len(data), nil
}

func (session *terminalSession) setReadError(err error) {
	if err == nil {
		return
	}
	session.readMu.Lock()
	if session.readErr == nil {
		session.readErr = err
	}
	session.readMu.Unlock()
}

func (session *terminalSession) readError() error {
	session.readMu.Lock()
	defer session.readMu.Unlock()
	return session.readErr
}
