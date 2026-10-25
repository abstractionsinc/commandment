package cli

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
)

// Env is an explicit environment snapshot for an invocation. Entry order is
// preserved; use EnvFromMap when a deterministic name-sorted projection of a map
// is desired.
type Env []EnvEntry

// EnvEntry is one environment name/value pair supplied to invocation analysis.
type EnvEntry struct {
	Name  string
	Value string
}

// EnvFromEnviron converts name=value entries to Env in input order. It returns an
// error when any entry has no '=' separator.
func EnvFromEnviron(entries []string) (Env, error) {
	environment := make(Env, 0, len(entries))
	for _, entry := range entries {
		name, value, found := strings.Cut(entry, "=")
		if !found {
			return nil, &invalidEnvironmentEntryError{}
		}
		environment = append(environment, EnvEntry{
			Name:  name,
			Value: value,
		})
	}
	return environment, nil
}

// EnvFromMap converts a map to Env in deterministic lexical name order.
func EnvFromMap(values map[string]string) Env {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	slices.Sort(names)
	environment := make(Env, 0, len(names))
	for _, name := range names {
		environment = append(environment, EnvEntry{
			Name:  name,
			Value: values[name],
		})
	}
	return environment
}

type invalidEnvironmentEntryError struct{}

// Error returns the stable malformed-environment-entry failure description.
func (err *invalidEnvironmentEntryError) Error() string { return "cli: malformed environment entry" }

// RunRequest supplies one explicit in-process execution. Its slices and host
// capabilities belong to that operation; absent optional capabilities remain
// absent rather than being read implicitly from the process. WorkingDirectory,
// environment, streams, signals, notice delivery, locale, and analysis limits
// are therefore caller-owned invocation inputs.
type RunRequest struct {
	Args []string
	Env  Env

	// WorkingDirectory is the invocation-relative working-directory value. An empty
	// value remains empty; Run does not read the process working directory.
	WorkingDirectory string
	ResponseFiles    ResponseFileAccess
	Defaults         []DefaultProvider
	Facts            []FactProvider

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	Signals SignalHost
	Notify  NoticeHandler

	Locale Locale
	Limits AnalysisLimits
}

// InspectionRequest supplies the explicit host and input state used to analyze
// an invocation without executing it. SignalHostCapabilities carries only
// invocation-owned host capability facts; no live signal runtime is started.
type InspectionRequest struct {
	Args []string
	Env  Env

	// WorkingDirectory is the invocation-relative working-directory value. An empty
	// value remains empty; Inspect does not read the process working directory.
	WorkingDirectory string
	ResponseFiles    ResponseFileAccess
	Defaults         []DefaultProvider
	Facts            []FactProvider

	SignalHostCapabilities SignalHostCapabilities

	Locale Locale
	Limits AnalysisLimits
}

// EffectiveLocale returns the locale carried by a phase-authorized Commandment
// context. It does not establish execution access or another semantic dependency.
func EffectiveLocale(ctx context.Context) Locale { return currentLocale(ctx) }

type invocationContextKey struct{}

type invocationContextState struct {
	locale       Locale
	selection    CommandSelection
	hasSelection bool
}

// withInvocationObservationContext installs already-resolved operation state.
// Callers must resolve and canonicalize locale and command selection before
// installing them; this context carrier is not a second selection authority.
func withInvocationObservationContext(ctx context.Context, locale Locale, selection CommandSelection, hasSelection bool) context.Context {
	ctx = requireContext(ctx)
	state := invocationContextState{
		locale:       locale,
		selection:    cloneCommandSelection(selection),
		hasSelection: hasSelection,
	}
	return context.WithValue(ctx, invocationContextKey{}, state)
}

func invocationObservationContext(ctx context.Context) (invocationContextState, bool) {
	if ctx == nil {
		return invocationContextState{}, false
	}
	state, ok := ctx.Value(invocationContextKey{}).(invocationContextState)
	return state, ok
}

func cloneCommandSelection(selection CommandSelection) CommandSelection {
	selection.Path = append([]CommandID(nil), selection.Path...)
	selection.Steps = append([]CommandSelectionStep(nil), selection.Steps...)
	for index := range selection.Steps {
		selection.Steps[index].Commands = append([]CommandID(nil), selection.Steps[index].Commands...)
	}
	return selection
}

func currentLocale(ctx context.Context) Locale {
	state, ok := invocationObservationContext(ctx)
	if !ok {
		return ""
	}
	return state.locale
}

var errInvalidInvocationProvider = errors.New("cli: invalid invocation provider")

var errInvalidEnvironment = errors.New("invalid prepared environment input")

// invocationInput is the detached cross-frontend semantic input retained for one invocation.
// Frontend-only process, completion, and inspection capabilities remain with their frontend owners.
type invocationInput struct {
	args             []string
	environment      []EnvEntry
	workingDirectory string
	responseFiles    ResponseFileAccess
	defaults         []DefaultProvider
	facts            []FactProvider
	locale           Locale
	limits           AnalysisLimits
}

func (provider typedDefaultProvider[C]) validDefaultProvider() bool { return provider.provide != nil }

func (provider typedFactProvider[C]) validFactProvider() bool { return provider.provide != nil }

func snapshotInvocationInput(args []string, env Env, workingDirectory string, responseFiles ResponseFileAccess, defaults []DefaultProvider, facts []FactProvider, locale Locale, limits AnalysisLimits) (invocationInput, error) {
	normalized, err := normalizeAnalysisLimits(limits)
	if err != nil {
		return invocationInput{}, err
	}
	defaultSnapshot := append([]DefaultProvider(nil), defaults...)
	factSnapshot := append([]FactProvider(nil), facts...)
	if !validDefaultProviderList(defaultSnapshot) || !validFactProviderList(factSnapshot) {
		return invocationInput{}, errInvalidInvocationProvider
	}
	environment := append([]EnvEntry(nil), env...)
	input := invocationInput{
		args:             append([]string(nil), args...),
		environment:      environment,
		workingDirectory: workingDirectory,
		responseFiles:    responseFiles,
		defaults:         defaultSnapshot,
		facts:            factSnapshot,
		locale:           locale,
		limits:           normalized,
	}
	return input, nil
}

func validDefaultProviderList(providers []DefaultProvider) bool {
	for _, provider := range providers {
		if provider == nil {
			return false
		}
		state, ok := provider.(defaultProviderRunner)
		if !ok || !state.validDefaultProvider() {
			return false
		}
	}
	return true
}

func validFactProviderList(providers []FactProvider) bool {
	for _, provider := range providers {
		if provider == nil {
			return false
		}
		state, ok := provider.(factProviderRunner)
		if !ok || !state.validFactProvider() {
			return false
		}
	}
	return true
}

func validEnvironmentEntry(entry EnvEntry) bool {
	return entry.Name != "" && !strings.ContainsAny(entry.Name, "=\x00") && !strings.ContainsRune(entry.Value, '\x00')
}

type invocation struct {
	scope identityScope

	program   staticProgram
	callbacks declarationCallbacks
	input     invocationInput

	signalDescription semanticSignalSourceDescription
	presentation      *frameworkPresentationState
	publication       publicationBarrier

	analysis *analysisState
}

// beginInvocation creates the stable operation shell without starting semantic analysis.
func beginInvocation(base staticProgram, callbacks declarationCallbacks, identity *applicationIdentity, signalDescription semanticSignalSourceDescription, input invocationInput) (*invocation, error) {
	if !base.Valid() {
		panic("invocation: invalid application declarations")
	}
	if identity == nil {
		panic("invocation: invalid application identity")
	}
	if input.limits.Diagnostics < 0 {
		panic("invocation: negative diagnostic limit")
	}
	for _, entry := range input.environment {
		if !validEnvironmentEntry(entry) {
			return nil, errInvalidEnvironment
		}
	}

	operation := &invocation{
		scope: identityScope{
			application: identity,
			operation:   &operationIdentity{},
		},
		program:           base,
		callbacks:         callbacks,
		input:             input,
		signalDescription: signalDescription,
		presentation:      prepareFrameworkPresentationState(base.Application().frameworkLocalization, base.Application().name),
	}

	return operation, nil
}

func (operation *invocation) DiagnosticLimit() int { return operation.input.limits.Diagnostics }

func (operation *invocation) Env() []EnvEntry {
	return append([]EnvEntry(nil), operation.input.environment...)
}
