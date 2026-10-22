package cli

import (
	"context"
	"slices"
	"sync"
)

type executionConfigurations struct {
	values    map[CommandID]any
	handedOff bool
}

func prepareExecutionConfigurations(model *effectiveModel, path []CommandID) (*executionConfigurations, bool) {
	if len(path) == 0 {
		return nil, false
	}
	values := make(map[CommandID]any, len(path))
	for _, command := range path {
		if !command.present() {
			return nil, false
		}
		if _, duplicate := values[command]; duplicate {
			return nil, false
		}
		record, ok := model.Command(command)
		if !ok {
			return nil, false
		}
		factory, ok := record.Configuration()
		if !ok {
			return nil, false
		}
		configuration := factory.New()
		if configuration == nil {
			return nil, false
		}
		values[command] = configuration
	}
	configurations := &executionConfigurations{
		values: values,
	}
	return configurations, true
}

func (configurations *executionConfigurations) configuration(command CommandID) (any, bool) {
	if configurations == nil || configurations.handedOff || !command.present() {
		return nil, false
	}
	configuration, ok := configurations.values[command]
	return configuration, ok && configuration != nil
}

func (configurations *executionConfigurations) handoff() {
	if configurations == nil || configurations.handedOff {
		panic("cli: invalid execution configuration handoff")
	}
	configurations.handedOff = true
	configurations.values = nil
}

// Handler is the selected command's innermost application execution callback. It receives the
// invocation-local live configuration after execution access is realized and returns an ordinary
// application error.
type Handler[C any] func(context.Context, *C) error

// MiddlewareNext synchronously admits the next execution stage. It is non-nil, one-shot,
// callback-scoped, and must not be retained or invoked concurrently or after its middleware
// returns.
type MiddlewareNext = func(context.Context) error

// Middleware is application execution middleware over the invocation-local live configuration.
// Declaration order is outer-to-inner, and a middleware either calls next exactly once and returns
// after it settles or returns a non-nil application error without calling next.
type Middleware[C any] func(context.Context, *C, MiddlewareNext) error

func stageStaticExecution(input declarationPlan, source *declarationDraft, draft *declarationCandidate, state *staticAcceptanceState, rootExecution commandExecution) {
	if state == nil {
		panic("cli: nil static acceptance state")
	}
	rootAccess := make([]accessDefinition, 0)
	accessByCommand := make([][]accessDefinition, len(input.records))
	for index := range len(input.records) {
		record := input.records[index]
		if record.kind != declarationRecordAccess {
			continue
		}
		access := source.accesses.entries[record.familyIndex].value
		value := accessDefinition{
			kind:    access.kind,
			source:  access.source,
			target:  access.target,
			prepare: access.prepare,
		}
		if record.commandIndex == -1 {
			rootAccess = append(rootAccess, value)
			continue
		}
		if record.commandIndex < 0 || record.commandIndex >= len(input.records) {
			panic("cli: validated static access has invalid command index")
		}
		accessByCommand[record.commandIndex] = append(
			accessByCommand[record.commandIndex],
			value,
		)
	}
	if len(rootAccess) != 0 {
		access, valid := prepareExecutionAccess(rootAccess, state)
		if !valid {
			panic("cli: validated root access cannot be accepted")
		}
		draft.CommandAccess(state.root, access, 0)
	}
	for index, declarations := range accessByCommand {
		if len(declarations) == 0 {
			continue
		}
		command := state.mustCommand(index)
		access, valid := prepareExecutionAccess(declarations, state)
		if !valid {
			panic("cli: validated command access cannot be accepted")
		}
		draft.CommandAccess(command, access, input.positionFor(index))
	}
	draft.CommandExecution(state.root, rootExecution, 0)
	for index := range len(input.records) {
		record := input.records[index]
		if record.kind != declarationRecordCommand {
			continue
		}
		command := state.mustCommand(index)
		value := source.commands.entries[record.familyIndex].value
		execution := acceptCommandExecution(value.middleware, value.handler)
		draft.CommandExecution(command, execution, input.positionFor(index))
	}
}

type commandHandler func(context.Context, any) error

type commandMiddleware func(context.Context, any, func(context.Context) error) error

// commandExecution is the immutable erased execution composition declared for
// one exact command. It preserves declaration order and exact typed callback
// behavior without making mutable configuration semantic authority.
type commandExecution struct {
	middleware []commandMiddleware
	handler    commandHandler
}

func acceptCommandExecution(middleware []commandMiddleware, handler commandHandler) commandExecution {
	for _, middleware := range middleware {
		if middleware == nil {
			panic("semantic: nil command middleware")
		}
	}
	execution := commandExecution{
		middleware: append([]commandMiddleware(nil), middleware...),
		handler:    handler,
	}
	return execution
}

func (execution commandExecution) Middleware() []commandMiddleware {
	return append([]commandMiddleware(nil), execution.middleware...)
}

func (execution commandExecution) Handler() (commandHandler, bool) {
	return execution.handler, execution.handler != nil
}

func (execution commandExecution) Runnable() bool { return execution.handler != nil }

func execute(ctx context.Context, prepared preparedExecution) error {
	return executePrepared(ctx, prepared, nil)
}

func executeSignaled(ctx context.Context, prepared preparedExecution, onCleanup func()) error {
	return executePrepared(ctx, prepared, onCleanup)
}

func executePrepared(ctx context.Context, prepared preparedExecution, onCleanup func()) error {
	if ctx == nil || prepared.handler == nil {
		panic("cli: invalid prepared execution")
	}

	var cleanupOnce sync.Once
	beginCleanup := func() {
		if onCleanup != nil {
			cleanupOnce.Do(onCleanup)
		}
	}
	downstream := func(callCtx context.Context) error {
		err := prepared.handler(callCtx)
		beginCleanup()
		return err
	}
	for _, middleware := range slices.Backward(prepared.middleware) {
		next := downstream
		downstream = func(callCtx context.Context) error {
			return executePreparedMiddleware(callCtx, middleware, next, beginCleanup, prepared.presentation)
		}
	}
	return downstream(ctx)
}

func executePreparedMiddleware(ctx context.Context, middleware boundMiddleware, downstream func(context.Context) error, beginCleanup func(), presentation *frameworkPresentationState) error {
	state := prepareContinuationState(presentation)
	next := func(supplied context.Context) error {
		return state.call(ctx, supplied, downstream)
	}
	middlewareErr := middleware(ctx, next)
	called, downstreamErr, protocolErr := state.finish()
	if !called && beginCleanup != nil {
		beginCleanup()
	}
	if !called && middlewareErr == nil && protocolErr == nil {
		protocolErr = middlewareProtocolFailure(presentation, middlewareViolationMissingNext)
	}
	return preserveMiddlewareErrors(protocolErr, downstreamErr, middlewareErr)
}

type middlewareViolationKind uint8

const (
	middlewareViolationInvalid middlewareViolationKind = iota
	middlewareViolationMissingNext
	middlewareViolationRepeatedNext
	middlewareViolationNilContext
	middlewareViolationExpiredNext
	middlewareViolationActiveAtReturn
)

type middlewareProtocolError struct {
	kind middlewareViolationKind
}

func (err *middlewareProtocolError) Error() string {
	return "commandment: middleware protocol violation"
}

// continuationState owns one middleware continuation token. The erased live
// execution path is the only chain executor; this state enforces exactly-once,
// non-nil-context, no-escape, and unwind-settlement semantics.
type continuationState struct {
	mu sync.Mutex

	presentation  *frameworkPresentationState
	used          bool
	active        bool
	expired       bool
	called        bool
	downstreamErr error
	protocolErr   error
	done          chan struct{}
}

func prepareContinuationState(presentation *frameworkPresentationState) *continuationState {
	state := &continuationState{
		presentation: presentation,
		done:         make(chan struct{}),
	}
	return state
}

func (state *continuationState) call(authoritative, supplied context.Context, downstream func(context.Context) error) error {
	state.mu.Lock()
	if state.expired {
		err := state.recordViolationLocked(middlewareViolationExpiredNext)
		state.mu.Unlock()
		return err
	}
	if state.used {
		err := state.recordViolationLocked(middlewareViolationRepeatedNext)
		state.mu.Unlock()
		return err
	}
	state.used = true
	if supplied == nil {
		err := state.recordViolationLocked(middlewareViolationNilContext)
		state.mu.Unlock()
		return err
	}
	state.active = true
	state.called = true
	state.mu.Unlock()

	linked, release := linkCancellation(authoritative, supplied)
	var err error
	defer func() {
		release()
		state.mu.Lock()
		state.active = false
		close(state.done)
		state.mu.Unlock()
	}()
	err = downstream(linked)
	state.mu.Lock()
	state.downstreamErr = err
	state.mu.Unlock()
	return err
}

func (state *continuationState) finish() (bool, error, error) {
	state.mu.Lock()
	state.expired = true
	active := state.active
	if active {
		_ = state.recordViolationLocked(middlewareViolationActiveAtReturn)
	}
	state.mu.Unlock()

	// If middleware returned while an admitted downstream call is still active,
	// settle that work before determining the invocation result.
	if active {
		<-state.done
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	return state.called, state.downstreamErr, state.protocolErr
}

func (state *continuationState) recordViolationLocked(kind middlewareViolationKind) error {
	if state.protocolErr == nil {
		state.protocolErr = middlewareProtocolFailure(state.presentation, kind)
	}
	return state.protocolErr
}

func middlewareProtocolFailure(presentation *frameworkPresentationState, kind middlewareViolationKind) error {
	cause := &middlewareProtocolError{
		kind: kind,
	}
	failure := payload{
		Diagnostics: []diagnostic{{
			Code:             string(DiagnosticMiddlewareProtocolViolation),
			Message:          "middleware protocol violation",
			FrameworkMessage: true,
		}},
		Presentation: presentation,
	}
	return retainDiagnosticFailure(failure, cause)
}

func linkCancellation(authoritative, supplied context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancelCause(supplied)
	stop := context.AfterFunc(authoritative, func() {
		cause := context.Cause(authoritative)
		if cause == nil {
			cause = authoritative.Err()
		}
		cancel(cause)
	})
	return ctx, func() {
		stop()
		cancel(nil)
	}
}

func preserveMiddlewareErrors(protocol, downstream, middleware error) error {
	var independent []error
	if protocol != nil {
		independent = append(independent, protocol)
	}
	if downstream != nil {
		if middleware == nil {
			independent = append(independent, downstream)
			return joinOrdered(independent...)
		}
		if exactBranchContains(middleware, downstream) {
			independent = append(independent, middleware)
			return joinOrdered(independent...)
		}
		independent = append(independent, downstream, middleware)
		return joinOrdered(independent...)
	}
	if middleware != nil {
		independent = append(independent, middleware)
	}
	return joinOrdered(independent...)
}

// exactBranchContains proves branch identity only through exact comparable
// error values reachable by ordinary single/multi-unwrapping. Custom Is
// equivalence is intentionally insufficient to erase a preserved branch.
func exactBranchContains(root, target error) bool {
	if root == nil || target == nil {
		return false
	}
	const maximumErrorGraphNodes = 4096
	stack := []error{root}
	for visited := 0; len(stack) != 0 && visited < maximumErrorGraphNodes; {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current == nil {
			continue
		}
		visited++
		if exactErrorIdentity(current, target) {
			return true
		}
		if many, ok := current.(interface{ Unwrap() []error }); ok {
			children := many.Unwrap()
			for _, child := range slices.Backward(children) {
				stack = append(stack, child)
			}
			continue
		}
		if one, ok := current.(interface{ Unwrap() error }); ok {
			stack = append(stack, one.Unwrap())
		}
	}
	return false
}

type boundMiddleware func(context.Context, MiddlewareNext) error

type preparedExecution struct {
	middleware   []boundMiddleware
	handler      func(context.Context) error
	presentation *frameworkPresentationState
}

func bindExecution(middleware []func(context.Context, func(context.Context) error) error, handler func(context.Context) error, presentation *frameworkPresentationState) preparedExecution {
	if handler == nil {
		panic("cli: accepted execution has no handler")
	}
	bound := make([]boundMiddleware, len(middleware))
	for index, item := range middleware {
		if item == nil {
			panic("cli: accepted execution has nil middleware")
		}
		bound[index] = boundMiddleware(item)
	}
	prepared := preparedExecution{
		middleware:   bound,
		handler:      handler,
		presentation: presentation,
	}
	return prepared
}
